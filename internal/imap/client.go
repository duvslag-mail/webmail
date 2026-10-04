package imap

import (
	"crypto/tls"
	"errors"
	"net"

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

func TestConnection(cfg Config, username string, password string) error {
	adress := net.JoinHostPort(cfg.Host, cfg.Port)

	options := &imapclient.Options{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: cfg.AllowInsecureTLS,
			ServerName:         cfg.Host,
		},
	}

	var client *imapclient.Client
	var err error

	switch cfg.Encryption {
	case "tls":
		client, err = imapclient.DialTLS(adress, options)
	case "starttls":
		client, err = imapclient.DialStartTLS(adress, options)
	case "none":
		client, err = imapclient.DialInsecure(adress, options)
	default:
		err = ErrInvalidEncryption
	}

	if err != nil {
		return err
	}
	defer client.Logout()

	cmd := client.Login(username, password)

	if cmd.Wait() != nil {
		return ErrInvalidConnection
	}

	return nil
}
