package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
)

func (a *App) openSession(w http.ResponseWriter, r *http.Request) {
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if deviceID == "" {
		writeErr(w, http.StatusBadRequest, "device_id is required")
		return
	}

	var out sessionResponse

	err := withTx(a.DB, func(tx *sql.Tx) error {
		id, bookID, startPage, startedAt, createdAt, err := openSessionByDevice(tx, deviceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeErr(w, http.StatusNotFound, "no open session for this device")
				return errors.New("notfound")
			}
			return err
		}

		title, author, source, err := getBookInfo(tx, bookID)
		if err != nil {
			return err
		}

		out = sessionResponse{
			ID:        id,
			BookID:    bookID,
			DeviceID:  deviceID,
			StartPage: startPage,
			StartedAt: startedAt,
			CreatedAt: createdAt,
			BookTitle: title,
			Author:    author,
			Source:    source,
		}
		return nil
	})

	if err != nil {
		if err.Error() == "notfound" {
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, out)
}
