package auth

import (
	"github.com/duvlsag-mail/webmail/db/sqlc"
	"github.com/duvlsag-mail/webmail/internal/mail"
)

type AuthService struct {
	db        *sqlc.Queries
	masterKey []byte
	imapPool  *mail.Pool
}

func NewAuthService(db *sqlc.Queries, masterKey []byte, imapPool *mail.Pool) *AuthService {
	return &AuthService{
		db:        db,
		imapPool:  imapPool,
		masterKey: masterKey,
	}
}
