package handlers

import "github.com/go-chi/chi/v5"

func (h *AuthHandler) RegisterRoutes(r *chi.Mux) {
	r.Get("/", h.HandleHome)
}
