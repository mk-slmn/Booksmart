package handlers

import (
	"encoding/json"
	"net/http"
	"os"
)

var (
	appName    = "booksmart"
	appVersion = "dev"
)

type versionResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (a *App) version(w http.ResponseWriter, r *http.Request) {
	name := appName
	version := appVersion

	if v := os.Getenv("APP_NAME"); v != "" {
		name = v
	}
	if v := os.Getenv("APP_VERSION"); v != "" {
		version = v
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(versionResponse{
		Name:    name,
		Version: version,
	})
}
