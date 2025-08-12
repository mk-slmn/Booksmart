package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type startSessionRequest struct {
	DeviceID  string  `json:"device_id"`
	BookTitle string  `json:"book_title"`
	Author    *string `json:"author,omitempty"`
	Source    *string `json:"source,omitempty"`
	StartPage int     `json:"start_page"`
	StartedAt *string `json:"started_at,omitempty"`
}

type sessionResponse struct {
	ID              int64   `json:"id"`
	BookID          int64   `json:"book_id"`
	DeviceID        string  `json:"device_id"`
	StartPage       int     `json:"start_page"`
	EndPage         *int    `json:"end_page,omitempty"`
	StartedAt       string  `json:"started_at"`
	EndedAt         *string `json:"ended_at,omitempty"`
	DurationSeconds *int64  `json:"duration_seconds,omitempty"`
	CreatedAt       string  `json:"created_at"`
	BookTitle       string  `json:"book_title"`
	Author          *string `json:"author,omitempty"`
	Source          *string `json:"source,omitempty"`
}

func (a *App) startSession(w http.ResponseWriter, r *http.Request) {
	var req startSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.BookTitle = strings.TrimSpace(req.BookTitle)

	if req.DeviceID == "" {
		writeErr(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if req.BookTitle == "" {
		writeErr(w, http.StatusBadRequest, "book_title is required")
		return
	}
	if req.StartPage < 0 {
		writeErr(w, http.StatusBadRequest, "start_page must be >= 0")
		return
	}

	var startedAt string
	if req.StartedAt != nil && strings.TrimSpace(*req.StartedAt) != "" {
		t, err := parseRFC3339UTC(*req.StartedAt)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "started_at must be RFC3339 (e.g., 2025-09-16T21:25:00Z)")
			return
		}
		startedAt = t.Format(time.RFC3339)
	} else {
		startedAt = timeOrNowRFC3339(nil)
	}

	var out sessionResponse

	err := withTx(a.DB, func(tx *sql.Tx) error {
		var existing int64
		err := tx.QueryRow(`
			SELECT id FROM sessions
			WHERE device_id = ? AND ended_at IS NULL
			LIMIT 1
		`, req.DeviceID).Scan(&existing)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			writeErr(w, http.StatusConflict, "an open session already exists for this device")
			return errors.New("conflict")
		}

		var bookID int64
		if err := tx.QueryRow(`
			SELECT id FROM books WHERE title = ?
		`, req.BookTitle).Scan(&bookID); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			res, err := tx.Exec(`
				INSERT INTO books (title, author, source, created_at)
				VALUES (?, ?, ?, ?)
			`, req.BookTitle, req.Author, req.Source, timeOrNowRFC3339(nil))
			if err != nil {
				return err
			}
			bookID, err = res.LastInsertId()
			if err != nil {
				return err
			}
		}

		now := timeOrNowRFC3339(nil)
		res, err := tx.Exec(`
			INSERT INTO sessions (
				book_id, device_id, start_page, started_at, created_at
			) VALUES (?, ?, ?, ?, ?)
		`, bookID, req.DeviceID, req.StartPage, startedAt, now)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}

		out = sessionResponse{
			ID:        id,
			BookID:    bookID,
			DeviceID:  req.DeviceID,
			StartPage: req.StartPage,
			StartedAt: startedAt,
			CreatedAt: now,
			BookTitle: req.BookTitle,
			Author:    req.Author,
			Source:    req.Source,
		}
		return nil
	})

	if err != nil {
		if err.Error() == "conflict" {
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, out)
}
