package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

func (a *App) continueSession(w http.ResponseWriter, r *http.Request) {
	var req continueSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" {
		writeErr(w, http.StatusBadRequest, "device_id is required")
		return
	}

	var startedAt string
	if req.StartedAt != nil && strings.TrimSpace(*req.StartedAt) != "" {
		t, err := parseRFC3339UTC(*req.StartedAt)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "started_at must be RFC3339 (e.g., 2025-09-16T23:25:00Z)")
			return
		}
		startedAt = t.Format(time.RFC3339)
	} else {
		startedAt = timeOrNowRFC3339(nil)
	}

	var out sessionResponse

	err := withTx(a.DB, func(tx *sql.Tx) error {
		var (
			openID      int64
			openBookID  int64
			openStart   int
			openStarted string
			openCreated string
		)
		err := tx.QueryRow(`
			SELECT id, book_id, start_page, started_at, created_at
			FROM sessions
			WHERE device_id = ? AND ended_at IS NULL
			LIMIT 1
		`, req.DeviceID).Scan(&openID, &openBookID, &openStart, &openStarted, &openCreated)

		if err == nil {
			var title string
			var author, source *string
			if err := tx.QueryRow(`SELECT title, author, source FROM books WHERE id = ?`, openBookID).Scan(&title, &author, &source); err != nil {
				return err
			}
			out = sessionResponse{
				ID:        openID,
				BookID:    openBookID,
				DeviceID:  req.DeviceID,
				StartPage: openStart,
				StartedAt: openStarted,
				CreatedAt: openCreated,
				BookTitle: title,
				Author:    author,
				Source:    source,
			}
			writeJSON(w, http.StatusOK, out)
			return errors.New("returned-open")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var (
			lastID        int64
			lastBookID    int64
			lastStartPage int
			lastEndPage   *int
			lastStartedAt string
			lastCreatedAt string
		)
		err = tx.QueryRow(`
			SELECT id, book_id, start_page, end_page, started_at, created_at
			FROM sessions
			WHERE device_id = ?
			ORDER BY started_at DESC
			LIMIT 1
		`, req.DeviceID).Scan(&lastID, &lastBookID, &lastStartPage, &lastEndPage, &lastStartedAt, &lastCreatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "no prior session to continue")
			return errors.New("notfound")
		} else if err != nil {
			return err
		}

		startPage := lastStartPage
		if lastEndPage != nil {
			startPage = *lastEndPage
		}

		now := timeOrNowRFC3339(nil)
		res, err := tx.Exec(`
			INSERT INTO sessions (book_id, device_id, start_page, started_at, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, lastBookID, req.DeviceID, startPage, startedAt, now)
		if err != nil {
			return err
		}
		newID, err := res.LastInsertId()
		if err != nil {
			return err
		}

		var title string
		var author, source *string
		if err := tx.QueryRow(`SELECT title, author, source FROM books WHERE id = ?`, lastBookID).Scan(&title, &author, &source); err != nil {
			return err
		}

		out = sessionResponse{
			ID:        newID,
			BookID:    lastBookID,
			DeviceID:  req.DeviceID,
			StartPage: startPage,
			StartedAt: startedAt,
			CreatedAt: now,
			BookTitle: title,
			Author:    author,
			Source:    source,
		}
		return nil
	})

	if err != nil {
		switch err.Error() {
		case "returned-open":
			return
		case "notfound":
			return
		default:
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	writeJSON(w, http.StatusCreated, out)
}
