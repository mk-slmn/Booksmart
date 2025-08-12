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

		if err == nil {
			writeErr(w, http.StatusConflict, "an open session already exists for this device")
			return errors.New("conflict")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		var bookID int64
		err = tx.QueryRow(`SELECT id FROM books WHERE title = ?`, req.BookTitle).Scan(&bookID)
		if errors.Is(err, sql.ErrNoRows) {
			// create it
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
		} else if err != nil {
			return err
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

type stopSessionRequest struct {
	DeviceID string  `json:"device_id"`
	EndPage  *int    `json:"end_page,omitempty"`
	EndedAt  *string `json:"ended_at,omitempty"`
}

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

type continueSessionRequest struct {
	DeviceID  string  `json:"device_id"`
	StartedAt *string `json:"started_at,omitempty"`
}

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
