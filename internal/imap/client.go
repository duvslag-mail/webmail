package imap

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

var (
	ErrInvalidConnection = errors.New("invalid connection")
	ErrInvalidEncryption = errors.New("invalid encryption type")
)

type Config struct {
	Host             string
	Port             string
	Encryption       string // "tls", "starttls", or "none"
	AllowInsecureTLS bool
}

func Connect(ctx context.Context, cfg Config) (*imapclient.Client, error) {
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.AllowInsecureTLS,
		ServerName:         cfg.Host,
	}

	var conn net.Conn
	var err error
	dialer := &net.Dialer{}

	switch cfg.Encryption {
	case "tls":
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", addr)
	case "starttls", "none":
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	default:
		err = ErrInvalidEncryption
	}
	if err != nil {
		return nil, err
	}

	// set a deadline for everything
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	// if the context is canceled, close the connection
	context.AfterFunc(ctx, func() { conn.Close() })

	options := &imapclient.Options{TLSConfig: tlsConfig}
	var client *imapclient.Client
	if cfg.Encryption == "starttls" {
		client, err = imapclient.NewStartTLS(conn, options)
		if err != nil {
			conn.Close()
			return nil, err
		}
	} else {
		client = imapclient.New(conn, options)
	}

	return client, nil
}

func login(ctx context.Context, cfg Config, username, password string) (*imapclient.Client, error) {
	client, err := Connect(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := client.Login(username, password).Wait(); err != nil {
		client.Close()
		return nil, ErrInvalidConnection
	}

	return client, nil
}

func Login(ctx context.Context, cfg Config, username, password string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	client, err := login(ctx, cfg, username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	client.Logout().Wait()
	return nil
}

func GetSubjects(ctx context.Context, cfg Config, username, password string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	client, err := login(ctx, cfg, username, password)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// assumes the INBOX mailbox exists, this is just wip to test things :)
	mbox, err := client.Select("INBOX", &imap.SelectOptions{
		ReadOnly: true,
	}).Wait()
	if err != nil {
		return nil, err
	}

	if mbox.NumMessages == 0 {
		return nil, nil
	}

	var start uint32 = 1
	if mbox.NumMessages > 10 {
		start = mbox.NumMessages - 9 // Fetch last 10
	}

	var seqSet imap.SeqSet
	seqSet.AddRange(start, mbox.NumMessages)

	fetchOptions := imap.FetchOptions{
		Envelope: true,
	}

	fetchCmd := client.Fetch(seqSet, &fetchOptions)
	defer fetchCmd.Close()

	var emails []string
	for {
		data := fetchCmd.Next()
		if data == nil {
			break
		}

		buf, err := data.Collect()
		if err != nil {
			return nil, err
		}

		if buf.Envelope != nil {
			emails = append(emails, buf.Envelope.Subject)
		}
	}

	return emails, nil
}
