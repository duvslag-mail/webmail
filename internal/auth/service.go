package auth

import (
	"github.com/duvlsag-mail/webmail/db/sqlc"
	"github.com/duvlsag-mail/webmail/internal/imap"
)

type AuthService struct {
	db        *sqlc.Queries
	masterKey []byte
	imapPool  *imap.Pool
}

func NewAuthService(db *sqlc.Queries, masterKey []byte, imapPool *imap.Pool) *AuthService {
	return &AuthService{
		db:        db,
		imapPool:  imapPool,
		masterKey: masterKey,
	}
}
