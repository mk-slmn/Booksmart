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
		openID, openBookID, openStart, openStarted, openCreated, err := openSessionByDevice(tx, req.DeviceID)
		if err == nil {
			title, author, source, err := getBookInfo(tx, openBookID)
			if err != nil {
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

		_, lastBookID, lastStartPage, lastEndPage, _, _, err :=
			mostRecentSessionByDevice(tx, req.DeviceID)
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
		newID, err := insertSession(tx, lastBookID, req.DeviceID, startPage, startedAt, now)
		if err != nil {
			return err
		}

		title, author, source, err := getBookInfo(tx, lastBookID)
		if err != nil {
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
