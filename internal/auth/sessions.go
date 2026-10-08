package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/duvlsag-mail/webmail/db/sqlc"
	"github.com/duvlsag-mail/webmail/internal/crypto"
)

var (
	tokenLength              = 32
	ErrTokenGenerationFailed = errors.New("failed to generate token")
	ErrSessionCreationFailed = errors.New("failed to create session")
	sessionDuration          = 24 * time.Hour
)

// CreateSession creates a new session for a given user id and returns an error if something went wrong and a session token
func (s *AuthService) CreateSession(ctx context.Context, userId uuid.UUID, ipAdress, userAgent string) (string, error) {
	// generate a token
	err, token := NewToken()
	if err != nil {
		return "", ErrTokenGenerationFailed
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
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(sessionDuration),
			Valid: true,
		},
	}

	_, err = s.db.CreateSession(ctx, args)

	if err != nil {
		log.Printf("failed to create session: %v", err)
		return "", ErrSessionCreationFailed
	}

	return token, nil
}

func NewToken() (error, string) {
	bytes := make([]byte, tokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return err, ""
	}
	return nil, string(base64.RawURLEncoding.EncodeToString(bytes))
}

// VerifySession verifies a session token and returns an error if the session is not valid and the user id of the session
func (s *AuthService) VerifySession(ctx context.Context, token string) (uuid.UUID, error) {
	hashedToken := crypto.Hash(token)

	session, err := s.db.GetSessionByTokenHash(ctx, hashedToken)
	if err != nil {
		return uuid.Nil, err
	}

	if !session.ID.Valid {
		return uuid.Nil, errors.New("session is not valid")
	}

	return session.UserID.Bytes, nil
}

func (s *AuthService) DeleteSession(ctx context.Context, token string) error {
	hashedToken := crypto.Hash(token)

	err := s.db.DeleteSessionByTokenHash(ctx, hashedToken)
	if err != nil {
		return err
	}

	return nil
}
