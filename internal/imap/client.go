package imap

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

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
	client, conn, err := dial(ctx, cfg)
	if err != nil {
		return nil, err
	}

	context.AfterFunc(ctx, func() { conn.Close() })

	return client, nil
}

func dial(ctx context.Context, cfg Config) (*imapclient.Client, net.Conn, error) {
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
		return nil, nil, err
	}

	// set a deadline for everything
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	options := &imapclient.Options{TLSConfig: tlsConfig}
	var client *imapclient.Client
	if cfg.Encryption == "starttls" {
		client, err = imapclient.NewStartTLS(conn, options)
		if err != nil {
			conn.Close()
			return nil, nil, err
		}
	} else {
		client = imapclient.New(conn, options)
	}

	return client, conn, nil
}

func FetchSubjects(client *imapclient.Client) ([]string, error) {
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

	var emails []string
	for {
		data := fetchCmd.Next()
		if data == nil {
			break
		}

		buf, err := data.Collect()
		if err != nil {
			fetchCmd.Close()
			return nil, err
		}

		if buf.Envelope != nil {
			emails = append(emails, buf.Envelope.Subject)
		}
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, err
	}

	return emails, nil
}
