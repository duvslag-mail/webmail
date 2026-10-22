package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/duvlsag-mail/webmail/internal/auth"
	"github.com/duvlsag-mail/webmail/internal/mail"
	"github.com/duvlsag-mail/webmail/templates/pages"
)

type AuthHandler struct {
	AuthService *auth.AuthService
	imapPool    *mail.Pool
}

func NewAuthHandler(authService *auth.AuthService, imapPool *mail.Pool) *AuthHandler {
	return &AuthHandler{
		AuthService: authService,
		imapPool:    imapPool,
	}
}

func (h *AuthHandler) HandleAuthRouter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := h.AuthService.VerifySession(ctx, sessionToken(r))

	if err != nil {
		if errors.Is(err, auth.ErrDbError) {
			log.Printf("session lookup failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// session is invalid so redirect to login
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/mail", http.StatusSeeOther)
}

func (h *AuthHandler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	handler := templ.Handler(pages.Login(""))
	handler.ServeHTTP(w, r)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	token := sessionToken(r)

	if token == "" {
		// return a server error, who tries to logout without a token :<
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// drop the pooled imap connection
	if userID, err := h.AuthService.VerifySession(r.Context(), token); err == nil {
		h.imapPool.Evict(userID)
	}

	err := h.AuthService.DeleteSession(r.Context(), token)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// delete the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})

	// redirect to /login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) HandleLoginRequest(w http.ResponseWriter, r *http.Request) {
	// get the data from the form data
	email := r.FormValue("email")
	password := r.FormValue("password")

	loginInput := auth.LoginInput{
		Email:     email,
		Password:  password,
		IpAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}

	// validate the input
	if err := loginInput.Validate(); err != nil {
		h.renderError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	// attempt to login
	token, err := h.AuthService.Login(r.Context(), loginInput)
	if err != nil {
		h.renderError(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	// set the token in a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   int(auth.SessionDuration.Seconds()),
		SameSite: http.SameSiteLaxMode,
		Value:    token,
		Path:     "/",
	})

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/mail")
		w.WriteHeader(http.StatusOK)
		return
	}

	// redirect to /mail
	http.Redirect(w, r, "/mail", http.StatusSeeOther)
}

func (h *AuthHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {

	if isHTMX(r) {
		handler := templ.Handler(pages.LoginForm(message))
		// return 2xx for htmx
		w.WriteHeader(http.StatusOK)
		handler.ServeHTTP(w, r)
		return
	}

	w.WriteHeader(status)
	handler := templ.Handler(pages.Login(message))
	handler.ServeHTTP(w, r)
}
