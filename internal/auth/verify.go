package auth

import (
	"context"
	"net/http"
)

func (s *AuthService) VerifyAuth(ctx context.Context, cookies []*http.Cookie) (bool, error) {
	// get the auth cookie
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "token" {
			authCookie = cookie
			break
		}
	}

	if authCookie == nil {
		return false, nil
	}

	// verify the auth cookie
	_, err := s.VerifySession(ctx, authCookie.Value)
	if err != nil {
		return false, err
	}

	return true, nil
}
