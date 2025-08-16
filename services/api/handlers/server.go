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
	r.Use(corsMW)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/v1", func(v chi.Router) {
		v.Get("/health", app.health)
		v.Get("/version", app.version)

		v.Post("/session/start", app.startSession)
		v.Post("/session/stop", app.stopSession)
		v.Post("/session/continue", app.continueSession)
		v.Get("/sessions/open", app.openSession)

		v.Get("/books", app.listBooks)
		v.Get("/books/recent", app.recentBooks)

		v.Get("/stats/weekly", app.statsWeekly)

		v.Get("/sessions", app.listSessions)

	})

	return r
}
