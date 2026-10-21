package imap

import (
	"context"
	"errors"
	"maps"
	"net"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
)

const commandTimeout = 15 * time.Second

// pool keeps an imap client for every user to reduce overhead of each request by not reconnecting
type Pool struct {
	load    CredsLoader
	idleTTL time.Duration
	mu      sync.Mutex
	entries map[uuid.UUID]*entry // uuid is the user's id
}

// conn is an open connection to an imap server that isn't in the pool yet
type Conn struct {
	imapclient *imapclient.Client
	conn       net.Conn
}

func (c *Conn) Close() error {
	if c.imapclient != nil {
		c.imapclient.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// puts a connection that isn't in the pool yet into the pool, replacing any existing connection for that user
func (p *Pool) Put(userID uuid.UUID, conn *Conn) {
	// clear the deadline cause it was only for login
	conn.conn.SetDeadline(time.Time{})

	e := &entry{client: conn.imapclient, conn: conn.conn, lastUsed: time.Now()}

	p.mu.Lock()
	old := p.entries[userID]
	p.entries[userID] = e
	p.mu.Unlock()

	// close old connection if it exists
	if old != nil {
		old.mu.Lock()
		p.evictLocked(userID, old) // map already points to e, so this only closes old
		old.mu.Unlock()
	}
}

func (p *Pool) Login(ctx context.Context, creds Creds) (*Conn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	client, conn, err := dial(dialCtx, creds.Config)
	if err != nil {
		return nil, err
	}

	if err := client.Login(creds.Username, creds.Password).Wait(); err != nil {
		client.Close()
		return nil, ErrInvalidConnection
	}

	return &Conn{imapclient: client, conn: conn}, nil
}

type entry struct {
	mu       sync.Mutex // guards client, conn, lastUsed, evicted
	client   *imapclient.Client
	conn     net.Conn
	lastUsed time.Time
	evicted  bool // removed from the pool, whoever holds this entry must start over
}

type Creds struct {
	Config   Config
	Username string
	Password string
}

// only called when a connection is needed, so we don't use unnecessary io when we already have a connection
type CredsLoader func(ctx context.Context, userID uuid.UUID) (Creds, error)

func NewPool(load CredsLoader, idleTTL time.Duration) *Pool {
	return &Pool{
		load:    load,
		idleTTL: idleTTL,
		entries: make(map[uuid.UUID]*entry),
	}
}

// retry is a boolean because it should only be used for read only operations or operations that can be retried safely (2 append commands in a row isn't nice)
func (p *Pool) With(ctx context.Context, userID uuid.UUID, fn func(*imapclient.Client) error, retry bool) error {
	for {
		e := p.get(userID)
		e.mu.Lock()
		if e.evicted {
			// if it got evicted while we were waiting, we try again
			e.mu.Unlock()
			continue
		}

		reused := e.client != nil // check if the client isn't nil, if it has already been instancied
		err := p.run(ctx, userID, e, fn)
		e.mu.Unlock()

		// only retry when the connection itself failed, not when the server said nope,
		// and not when we just connected (a fresh connection would fail the same way mmmh)
		if err != nil && retry && reused && isTransportErr(err) {
			retry = false
			continue
		}
		return err
	}
}

// closes the user's if it exists + removes it from the pool
func (p *Pool) Evict(userID uuid.UUID) {
	p.mu.Lock()
	e, ok := p.entries[userID]
	p.mu.Unlock()
	if !ok {
		return
	}

	e.mu.Lock() // waits for everything to finish
	p.evictLocked(userID, e)
	e.mu.Unlock()
}

// close idle sessions every minute & close everything when the context is canceled
func (p *Pool) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.Close()
			return
		case <-ticker.C:
			p.reap()
		}
	}
}

func (p *Pool) Close() {
	p.mu.Lock()
	entries := maps.Clone(p.entries)
	p.mu.Unlock()

	for userID, e := range entries {
		e.mu.Lock()
		p.evictLocked(userID, e)
		e.mu.Unlock()
	}
}

func (p *Pool) get(userID uuid.UUID) *entry {
	p.mu.Lock()
	defer p.mu.Unlock()

	e, ok := p.entries[userID]
	if !ok {
		e = &entry{lastUsed: time.Now()}
		p.entries[userID] = e
	}
	return e
}

// run must be called with e.mu held.
func (p *Pool) run(ctx context.Context, userID uuid.UUID, e *entry, fn func(*imapclient.Client) error) error {
	// the connection might be dead
	if e.client != nil {
		select {
		case <-e.client.Closed():
			e.client, e.conn = nil, nil
		default:
		}
	}

	if e.client == nil {
		if err := p.connect(ctx, userID, e); err != nil {
			p.evictLocked(userID, e)
			return err
		}
	}

	e.conn.SetDeadline(deadlineFor(ctx))
	err := fn(e.client)
	e.conn.SetDeadline(time.Time{})
	e.lastUsed = time.Now()

	if err != nil && isTransportErr(err) {
		p.evictLocked(userID, e)
	}
	return err
}

// connect must be called with e.mu held.
func (p *Pool) connect(ctx context.Context, userID uuid.UUID, e *entry) error {
	creds, err := p.load(ctx, userID)
	if err != nil {
		return err
	}

	// the connection outlives this request, so ctx only bounds the dial and LOGIN
	dialCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	client, conn, err := dial(dialCtx, creds.Config)
	if err != nil {
		return err
	}

	if err := client.Login(creds.Username, creds.Password).Wait(); err != nil {
		client.Close()
		return ErrInvalidConnection
	}

	e.client, e.conn = client, conn
	return nil
}

// evictLocked must be called with e.mu held.
func (p *Pool) evictLocked(userID uuid.UUID, e *entry) {
	p.mu.Lock()
	if p.entries[userID] == e { // don't delete an entry that already replaced this one
		delete(p.entries, userID)
	}
	p.mu.Unlock()

	e.evicted = true
	if e.client != nil {
		e.client.Close()
		e.client, e.conn = nil, nil
	}
}

func (p *Pool) reap() {
	p.mu.Lock()
	entries := maps.Clone(p.entries)
	p.mu.Unlock()

	for userID, e := range entries {
		if !e.mu.TryLock() {
			continue // in use, so not idle
		}
		if time.Since(e.lastUsed) > p.idleTTL {
			p.evictLocked(userID, e)
		}
		e.mu.Unlock()
	}
}

// returns now + command timeout or the context's deadline if it's sooner
func deadlineFor(ctx context.Context) time.Time {
	deadline := time.Now().Add(commandTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		return d
	}
	return deadline
}

func isTransportErr(err error) bool {
	var imapErr *imap.Error
	return !errors.As(err, &imapErr)
}
