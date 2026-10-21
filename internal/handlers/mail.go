package handlers

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/duvlsag-mail/webmail/internal/mail"
	"github.com/duvlsag-mail/webmail/templates/components"
	"github.com/duvlsag-mail/webmail/templates/pages"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
)

type MailHandler struct {
	pool *mail.Pool
}

func NewMailHandler(pool *mail.Pool) *MailHandler {
	return &MailHandler{
		pool: pool,
	}
}

func (h *MailHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	// get user id injected by auth middleware in the context
	userID := r.Context().Value(userIDKey).(uuid.UUID)

	folder := r.URL.Query().Get("folder")
	// uh convention ig
	if folder == "" {
		folder = "INBOX"
	}

	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 32) // default to 0 if not provided

	var msgs []mail.MessageSummary
	var next uint32

	err := h.pool.With(r.Context(), userID, func(client *imapclient.Client) error {
		var err error
		msgs, next, err = mail.FetchPage(client, folder, uint32(before))
		return err
	}, true) // retry cause read only, we don't really care

	if err != nil {
		http.Error(w, "Failed to fetch messages: "+err.Error(), http.StatusInternalServerError)
		return
	}
	templ.Handler(components.MessageRows(folder, msgs, next)).ServeHTTP(w, r)

}

func (h *MailHandler) HandleHome(w http.ResponseWriter, r *http.Request) {

	handler := templ.Handler(pages.Mail())
	handler.ServeHTTP(w, r)
}
