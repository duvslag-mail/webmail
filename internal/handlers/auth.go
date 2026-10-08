package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/duvlsag-mail/webmail/internal/auth"
	"github.com/duvlsag-mail/webmail/templates/pages"
)

type AuthHandler struct {
	AuthService *auth.AuthService
}

func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		AuthService: authService,
	}
}

func (h *AuthHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	valid, err := h.AuthService.VerifyAuth(ctx, r.Cookies())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !valid {
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
	token := ""
	cookies := r.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "token" {
			token = cookie.Value
			break
		}
	}

	if token == "" {
		// return a server error, who tries to logout without a token :<
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err := h.AuthService.DeleteSession(r.Context(), token)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

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
	token, err := h.AuthService.Login(loginInput)
	if err != nil {
		h.renderError(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	// set the token in a cookie
	http.SetCookie(w, &http.Cookie{
		Name:  "token",
		Value: token,
		Path:  "/",
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
