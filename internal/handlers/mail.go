package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/duvlsag-mail/webmail/templates/pages"
)

type MailHandler struct {
}

func NewMailHandler() *MailHandler {
	return &MailHandler{}
}

func (h *MailHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	handler := templ.Handler(pages.Mail())
	handler.ServeHTTP(w, r)
}
