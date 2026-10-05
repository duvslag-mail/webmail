package auth

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/duvlsag-mail/webmail/db/sqlc"
	"github.com/duvlsag-mail/webmail/internal/crypto"
	"github.com/duvlsag-mail/webmail/internal/imap"
)

var (
	ErrEmailRequired            = errors.New("email is required")
	ErrPasswordRequired         = errors.New("password is required")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrPasswordEncryptionFailed = errors.New("failed to encrypt password")
	ErrDatabaseError            = errors.New("database error")

	EmailRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
)

type LoginInput struct {
	Email     string `json:"email" form:"email"`
	Password  string `json:"password" form:"password"`
	IpAddress string `json:"ip_address" form:"ip_address"`
	UserAgent string `json:"user_agent" form:"user_agent"`
}

// Validate checks if you didn't give shit to login sob
func (l *LoginInput) Validate() error {
	if l.Email == "" {
		return ErrEmailRequired
	}
	if l.Password == "" {
		return ErrPasswordRequired
	}
	// regex
	matched, err := regexp.MatchString(EmailRegex, l.Email)
	if err != nil || !matched {
		return ErrInvalidCredentials
	}

	return nil
}

// GetDomain extracts the domain from the email address.
func (l *LoginInput) GetDomain() string {
	domain := strings.SplitN(l.Email, "@", 2)
	if len(domain) != 2 {
		return ""
	}
	return domain[1]
}

// Login returns a token if the login is successful, otherwise returns an error.
func (s *AuthService) Login(input LoginInput) (string, error) {
	ctx := context.Background()

	if err := input.Validate(); err != nil {
		return "", err // err here is already a proper error message
	}

	// check if the domain is registered
	domain := input.GetDomain()
	if domain == "" {
		return "", ErrInvalidCredentials
	}
	dbDomain, err := s.db.GetDomainByName(ctx, domain)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	// test the connection
	imapConfig := imap.Config{
		Host:             dbDomain.ImapHost,
		Port:             strconv.Itoa(int(dbDomain.ImapPort)),
		Encryption:       dbDomain.ImapEncryption,
		AllowInsecureTLS: dbDomain.AllowInsecureTls,
	}
	err = imap.TestConnection(imapConfig, input.Email, input.Password)

	if err != nil {
		return "", ErrInvalidCredentials
	}

	// encrypt the password
	encryptedPassword, err := crypto.Encrypt([]byte(input.Password), s.masterKey)

	if err != nil {
		return "", ErrPasswordEncryptionFailed
	}

	// upsert the user
	upsertUserArgs := sqlc.UpsertUserParams{
		DomainID:              dbDomain.ID,
		Email:                 input.Email,
		EncryptedImapPassword: []byte(encryptedPassword),
	}
	row, err := s.db.UpsertUser(ctx, upsertUserArgs)
	if err != nil {
		return "", ErrDatabaseError
	}
	// check if the user exists
	if !row.ID.Valid {
		return "", ErrInvalidCredentials
	}

	// create a session
	token, err := s.CreateSession(ctx, row.ID.Bytes, input.IpAddress, input.UserAgent)

	if err != nil {
		return "", err
	}

	return token, nil
}
