package handlers

import "github.com/go-chi/chi/v5"

func (h *AuthHandler) RegisterRoutes(r *chi.Mux) {
	r.Get("/", h.HandleHome)
	r.Get("/login", h.HandleLoginPage)
	r.Post("/login", h.HandleLoginRequest)
	r.Post("/logout", h.HandleLogout)
}

func (h *MailHandler) RegisterRoutes(r *chi.Mux) {
	r.Get("/mail", h.HandleHome)
}
