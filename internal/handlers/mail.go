package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/duvlsag-mail/webmail/internal/imap"
	"github.com/duvlsag-mail/webmail/templates/pages"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
)

type MailHandler struct {
	pool *imap.Pool
}

func NewMailHandler(pool *imap.Pool) *MailHandler {
	return &MailHandler{
		pool: pool,
	}
}

func (h *MailHandler) getSubjectsForUser(ctx context.Context, userID uuid.UUID) []string {
	var subjects []string

	// read only, so it's safe to retry on a stale connection
	err := h.pool.With(ctx, userID, func(c *imapclient.Client) error {
		var err error
		subjects, err = imap.FetchSubjects(c)
		return err
	}, true)

	if err != nil {
		log.Printf("fetching subjects failed: %v", err)
		return []string{"Error fetching subjects"}
	}

	return subjects
}

func (h *MailHandler) HandleHome(w http.ResponseWriter, r *http.Request) {

	var userID = r.Context().Value(userIDKey)
	subjects := h.getSubjectsForUser(r.Context(), userID.(uuid.UUID))

	handler := templ.Handler(pages.Mail(subjects))
	handler.ServeHTTP(w, r)
}
