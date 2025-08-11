package handlers

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type App struct {
	DB *sql.DB
}

func NewServer(db *sql.DB) http.Handler {
	app := &App{DB: db}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/v1", func(v chi.Router) {
		v.Get("/health", app.health)
		v.Get("/version", app.version)
	})

	return r
}
