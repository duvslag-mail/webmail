package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/duvlsag-mail/webmail/internal/auth"
)

type contextKey string

const userIDKey contextKey = "userID"

func RequireAuth(s *auth.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			for _, c := range r.Cookies() {
				if c.Name == "token" {
					token = c.Value
					break
				}
			}
			userID, err := s.VerifySession(r.Context(), token)
			if err != nil {
				if errors.Is(err, auth.ErrSessionNotFound) || errors.Is(err, auth.ErrSessionExpired) {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return // next never runs
				}

				log.Printf("session lookup failed: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// stash the user for the handler downstream
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
