package mail

import (
	"bytes"
	"cmp"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"slices"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	gomail "github.com/emersion/go-message/mail"
)

const pageSize = 50

var ErrMessageNotFound = errors.New("message not found")

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

func FetchPage(c *imapclient.Client, folder string, before uint32) ([]MessageSummary, uint32, error) {
	if _, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return nil, 0, err
	}

	// in go-imap a range ending in 0 means "*", so 1:0 would match every message
	if before == 1 {
		return nil, 0, nil
	}

	// search for all messages with UID < before
	criteria := &imap.SearchCriteria{}
	if before > 0 {
		criteria.UID = []imap.UIDSet{{imap.UIDRange{Start: 1, Stop: imap.UID(before - 1)}}}
	}
	data, err := c.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, 0, err
	}

	uids := data.AllUIDs()
	if len(uids) == 0 {
		return nil, 0, nil
	}

	start := max(0, len(uids)-pageSize)
	page := uids[start:]

	var set imap.UIDSet
	set.AddNum(page...)

	fetchCmd := c.Fetch(set, &imap.FetchOptions{
		UID:          true,
		Envelope:     true,
		InternalDate: true,
	})

	msgs, err := fetchCmd.Collect()
	if err != nil {
		return nil, 0, err
	}

	summaries := make([]MessageSummary, 0, len(msgs))
	for _, m := range msgs {
		summaries = append(summaries, toSummary(m))
	}

	// the server doesn't have to answer in UID order, so sort newest first ourselves
	slices.SortFunc(summaries, func(a, b MessageSummary) int {
		return cmp.Compare(b.UID, a.UID)
	})

	var next uint32
	if start > 0 {
		next = uint32(page[0])
	}
	return summaries, next, nil
}

func toSummary(m *imapclient.FetchMessageBuffer) MessageSummary {
	s := MessageSummary{
		UID:  uint32(m.UID),
		Date: m.InternalDate,
	}
	if m.Envelope != nil {
		s.Subject = m.Envelope.Subject
		s.From = toAddresses(m.Envelope.From)
		if !m.Envelope.Date.IsZero() {
			s.Date = m.Envelope.Date
		}
	}
	return s
}

func toAddresses(in []imap.Address) []Address {
	out := make([]Address, 0, len(in))
	for _, a := range in {
		if a.IsGroupStart() || a.IsGroupEnd() {
			continue
		}
		out = append(out, Address{Name: a.Name, Email: a.Addr()})
	}
	return out
}

func FetchMessage(c *imapclient.Client, folder string, uid uint32) (Message, error) {
	if _, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return Message{}, err
	}

	section := &imap.FetchItemBodySection{Peek: false}

	msgs, err := c.Fetch(imap.UIDSetNum(imap.UID(uid)), &imap.FetchOptions{
		UID:          true,
		Envelope:     true,
		InternalDate: true,
		BodySection:  []*imap.FetchItemBodySection{section},
	}).Collect()
	if err != nil {
		return Message{}, err
	}

	if len(msgs) == 0 {
		return Message{}, ErrMessageNotFound
	}

	buf := msgs[0] // choose the first message, there should in theory only be one anyway
	msg := Message{MessageSummary: toSummary(buf)}
	if buf.Envelope != nil {
		msg.To = toAddresses(buf.Envelope.To)
		msg.Cc = toAddresses(buf.Envelope.Cc)
	}

	if err := parseBody(&msg, buf.FindBodySection(section)); err != nil {
		return Message{}, err
	}

	return msg, nil
}

func parseBody(msg *Message, raw []byte) error {
	mr, err := gomail.CreateReader(bytes.NewReader(raw))

	if err != nil && !message.IsUnknownCharset(err) {
		return err
	}

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil && !message.IsUnknownCharset(err) {
			return err
		}
		if part == nil {
			continue
		}

		switch h := part.Header.(type) {
		case *gomail.InlineHeader:
			contentType, _, _ := h.ContentType()

			switch contentType {
			case "text/plain":
				body, err := io.ReadAll(part.Body)
				if err != nil {
					return err
				}
				msg.TextBody += string(body)
			case "text/html":
				if msg.HTMLBody != "" {
					continue
				}
				body, err := io.ReadAll(part.Body)
				if err != nil {
					return err
				}
				msg.HTMLBody = string(body)
			default:
				size, err := io.Copy(io.Discard, part.Body)
				if err != nil {
					return err
				}
				msg.Attachments = append(msg.Attachments, Attachment{
					MIMEType: contentType,
					Size:     uint32(size),
				})
			}

		case *gomail.AttachmentHeader:
			filename, _ := h.Filename()
			contentType, _, _ := h.ContentType()

			size, err := io.Copy(io.Discard, part.Body)
			if err != nil {
				return err
			}

			msg.Attachments = append(msg.Attachments, Attachment{
				Filename: filename,
				MIMEType: contentType,
				Size:     uint32(size),
			})
		}
	}

	return nil
}
