package auth

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/duvlsag-mail/webmail/db/sqlc"
	"github.com/duvlsag-mail/webmail/internal/crypto"
	"github.com/duvlsag-mail/webmail/internal/imap"
)

// returns a loader that fetches and decrypts the user's imap creds
func NewIMAPCredsLoader(db *sqlc.Queries, masterKey []byte) imap.CredsLoader {
	return func(ctx context.Context, userID uuid.UUID) (imap.Creds, error) {
		return loadIMAPCreds(ctx, db, masterKey, userID)
	}
}

func loadIMAPCreds(ctx context.Context, db *sqlc.Queries, masterKey []byte, userID uuid.UUID) (imap.Creds, error) {
	res, err := db.GetUserWithDomain(ctx, pgtype.UUID{
		Bytes: userID,
		Valid: true,
	})
	if err != nil {
		return imap.Creds{}, err
	}

	password, err := crypto.Decrypt(res.EncryptedImapPassword, masterKey)
	if err != nil {
		return imap.Creds{}, err
	}

	return imap.Creds{
		Config: imap.Config{
			Host:             res.ImapHost,
			Port:             strconv.Itoa(int(res.ImapPort)),
			Encryption:       res.ImapEncryption,
			AllowInsecureTLS: res.AllowInsecureTls,
		},
		Username: res.Email,
		Password: string(password),
	}, nil
}
