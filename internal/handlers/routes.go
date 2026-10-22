package handlers

import "github.com/go-chi/chi/v5"

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.HandleAuthRouter)
	r.Get("/login", h.HandleLoginPage)
	r.Post("/login", h.HandleLoginRequest)
	r.Post("/logout", h.HandleLogout)
}

func (h *MailHandler) RegisterRoutes(r chi.Router) {
	r.Get("/mail", h.HandleHome)
	r.Get("/mail/list", h.HandleMessageList)
	r.Get("/mail/view", h.HandleMessageView)
}
