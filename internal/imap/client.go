package imap

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

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

func TestConnection(ctx context.Context, cfg Config, username string, password string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

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
		// tls, we immediately wrap the conn in tls
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", addr)
	case "starttls", "none":
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	default:
		err = ErrInvalidEncryption
	}
	if err != nil {
		return err
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
			return err
		}
	} else {
		client = imapclient.New(conn, options)
	}
	defer client.Close()

	// if the context is canceled, close the client
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			client.Close()
		case <-done:
		}
	}()

	if err := client.Login(username, password).Wait(); err != nil {
		return ErrInvalidConnection
	}
	client.Logout().Wait()
	return nil
}
