package handlers

import "github.com/go-chi/chi/v5"

func NewServer() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/v1/health", Health)
	return r
}
