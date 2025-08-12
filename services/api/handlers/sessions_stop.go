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
		id, bookID, startPage, startedAt, createdAt, err := openSessionByDevice(tx, req.DeviceID)
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

		if err := closeSession(tx, id, endedAt, sec, req.EndPage); err != nil {
			if err.Error() == "end_page must be >= 0" {
				writeErr(w, http.StatusBadRequest, "end_page must be >= 0")
				return errors.New("badend")
			}
			return err
		}

		title, author, source, err := getBookInfo(tx, bookID)
		if err != nil {
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
