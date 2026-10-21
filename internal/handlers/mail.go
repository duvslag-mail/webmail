package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/duvlsag-mail/webmail/db/sqlc"
	"github.com/duvlsag-mail/webmail/internal/crypto"
	"github.com/duvlsag-mail/webmail/internal/imap"
	"github.com/duvlsag-mail/webmail/templates/pages"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type MailHandler struct {
	db        *sqlc.Queries
	masterKey []byte
}

func NewMailHandler(db *sqlc.Queries, masterKey []byte) *MailHandler {
	return &MailHandler{
		db:        db,
		masterKey: masterKey,
	}
}

func (h *MailHandler) getSubjectsForUser(ctx context.Context, userID uuid.UUID) []string {
	res, err := h.db.GetUserWithDomain(ctx, pgtype.UUID{
		Bytes: userID,
		Valid: true,
	})

	if err != nil {
		return []string{"Error fetching subjects"}
	}

	password, err := crypto.Decrypt(res.EncryptedImapPassword, h.masterKey)
	if err != nil {
		return []string{"Error decrypting password"}
	}

	// build a config

	config := imap.Config{
		Host:             res.ImapHost,
		Port:             strconv.Itoa(int(res.ImapPort)),
		Encryption:       res.ImapEncryption,
		AllowInsecureTLS: res.AllowInsecureTls,
	}

	subjects, err := imap.GetSubjects(ctx, config, res.Email, string(password))

	return subjects
}

func (h *MailHandler) HandleHome(w http.ResponseWriter, r *http.Request) {

	var userID = r.Context().Value(userIDKey)
	subjects := h.getSubjectsForUser(r.Context(), userID.(uuid.UUID))

	handler := templ.Handler(pages.Mail(subjects))
	handler.ServeHTTP(w, r)
}
