package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

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
		existing, err := openSessionIDByDevice(tx, req.DeviceID)
		if err != nil {
			return err
		}
		if existing != 0 {
			writeErr(w, http.StatusConflict, "an open session already exists for this device")
			return errors.New("conflict")
		}

		bookID, err := findBookIDByTitle(tx, req.BookTitle)
		if err != nil {
			return err
		}
		if bookID == 0 {
			bookID, err = insertBook(tx, req.BookTitle, req.Author, req.Source, timeOrNowRFC3339(nil))
			if err != nil {
				return err
			}
		}

		now := timeOrNowRFC3339(nil)
		id, err := insertSession(tx, bookID, req.DeviceID, req.StartPage, startedAt, now)
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
