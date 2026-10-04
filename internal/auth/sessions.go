package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/duvlsag-mail/webmail/db/sqlc"
	"github.com/duvlsag-mail/webmail/internal/crypto"
)

var (
	tokenLength              = 32
	ErrTokenGenerationFailed = errors.New("failed to generate token")
	ErrSessionCreationFailed = errors.New("failed to create session")
)

// CreateSession creates a new session for a given user id and returns an error if something went wrong and a session token
func (s *AuthService) CreateSession(ctx context.Context, userId uuid.UUID, ipAdress, userAgent string) (error, string) {
	// generate a token
	err, token := NewToken()
	if err != nil {
		return ErrTokenGenerationFailed, ""
	}

	// hash the token :scheming:
	hashedToken := crypto.Hash(token)

	args := sqlc.CreateSessionParams{
		UserID: pgtype.UUID{
			Bytes: userId,
			Valid: true,
		},
		TokenHash: hashedToken,
		IpAddress: ipAdress,
		UserAgent: userAgent,
	}

	_, err = s.db.CreateSession(ctx, args)

	if err != nil {
		return ErrSessionCreationFailed, ""
	}

	return nil, token
}

func NewToken() (error, string) {
	bytes := make([]byte, tokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return err, ""
	}
	return nil, string(base64.RawURLEncoding.EncodeToString(bytes))
}
