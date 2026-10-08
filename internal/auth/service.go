package auth

import (
	"github.com/duvlsag-mail/webmail/db/sqlc"
)

type AuthService struct {
	db        *sqlc.Queries
	masterKey []byte
}

func NewAuthService(db *sqlc.Queries, masterKey []byte) *AuthService {
	return &AuthService{
		db:        db,
		masterKey: masterKey,
	}
}
