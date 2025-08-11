package handlers

import (
	"github.com/go-chi/cors"
)

var corsMW = cors.Handler(cors.Options{
	AllowedOrigins: []string{"*"}, // change for production

	AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	AllowedHeaders: []string{
		"Accept",
		"Authorization",
		"Content-Type",
		"X-CSRF-Token",
	},
	ExposedHeaders:   []string{"Link"},
	AllowCredentials: false,
	MaxAge:           300,
})
