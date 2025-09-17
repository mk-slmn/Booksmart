package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

func (a *App) stopSession(w http.ResponseWriter, r *http.Request) {
	var req stopSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" {
		writeErr(w, http.StatusBadRequest, "device_id is required")
		return
	}

	var endedAt string
	if req.EndedAt != nil && strings.TrimSpace(*req.EndedAt) != "" {
		t, err := parseRFC3339UTC(*req.EndedAt)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "ended_at must be RFC3339 (e.g., 2025-09-16T21:25:00Z)")
			return
		}
		endedAt = t.Format(time.RFC3339)
	} else {
		endedAt = timeOrNowRFC3339(nil)
	}

	var out sessionResponse

	err := withTx(a.DB, func(tx *sql.Tx) error {
		var (
			id        int64
			bookID    int64
			startPage int
			startedAt string
			createdAt string
		)
		err := tx.QueryRow(`
			SELECT id, book_id, start_page, started_at, created_at
			FROM sessions
			WHERE device_id = ? AND ended_at IS NULL
			LIMIT 1
		`, req.DeviceID).Scan(&id, &bookID, &startPage, &startedAt, &createdAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeErr(w, http.StatusNotFound, "no open session for this device")
				return errors.New("notfound")
			}
			return err
		}

		st, err := parseRFC3339UTC(startedAt)
		if err != nil {
			return err
		}
		en, err := parseRFC3339UTC(endedAt)
		if err != nil {
			return err
		}
		dur := en.Sub(st)
		if dur < 0 {
			dur = 0
		}
		sec := int64(dur / time.Second)

		if req.EndPage != nil {
			if *req.EndPage < 0 {
				writeErr(w, http.StatusBadRequest, "end_page must be >= 0")
				return errors.New("badend")
			}
			_, err = tx.Exec(`
				UPDATE sessions
				SET end_page = ?, ended_at = ?, duration_seconds = ?
				WHERE id = ?
			`, *req.EndPage, endedAt, sec, id)
		} else {
			_, err = tx.Exec(`
				UPDATE sessions
				SET ended_at = ?, duration_seconds = ?
				WHERE id = ?
			`, endedAt, sec, id)
		}
		if err != nil {
			return err
		}

		var title string
		var author, source *string
		if err := tx.QueryRow(`SELECT title, author, source FROM books WHERE id = ?`, bookID).
			Scan(&title, &author, &source); err != nil {
			return err
		}

		out = sessionResponse{
			ID:              id,
			BookID:          bookID,
			DeviceID:        req.DeviceID,
			StartPage:       startPage,
			EndPage:         req.EndPage,
			StartedAt:       startedAt,
			EndedAt:         &endedAt,
			DurationSeconds: &sec,
			CreatedAt:       createdAt,
			BookTitle:       title,
			Author:          author,
			Source:          source,
		}
		return nil
	})

	if err != nil {
		if err.Error() == "notfound" || err.Error() == "badend" {
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, out)
}
