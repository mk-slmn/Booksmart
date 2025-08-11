package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	})
}
