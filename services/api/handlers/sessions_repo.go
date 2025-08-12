package handlers

import (
	"database/sql"
	"errors"
)

// -- Books --
func findBookIDByTitle(tx *sql.Tx, title string) (int64, error) {
	var id int64
	err := tx.QueryRow(`SELECT id FROM books WHERE title = ?`, title).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func insertBook(tx *sql.Tx, title string, author, source *string, createdAt string) (int64, error) {
	res, err := tx.Exec(`
		INSERT INTO books (title, author, source, created_at)
		VALUES (?, ?, ?, ?)
	`, title, author, source, createdAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func getBookInfo(tx *sql.Tx, bookID int64) (title string, author, source *string, err error) {
	err = tx.QueryRow(`SELECT title, author, source FROM books WHERE id = ?`, bookID).
		Scan(&title, &author, &source)
	return
}

// -- Sessions --
func openSessionIDByDevice(tx *sql.Tx, deviceID string) (int64, error) {
	var id int64
	err := tx.QueryRow(`
		SELECT id
		FROM sessions
		WHERE device_id = ? AND ended_at IS NULL
		LIMIT 1
	`, deviceID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func openSessionByDevice(tx *sql.Tx, deviceID string) (id int64, bookID int64, startPage int, startedAt, createdAt string, err error) {
	err = tx.QueryRow(`
		SELECT id, book_id, start_page, started_at, created_at
		FROM sessions
		WHERE device_id = ? AND ended_at IS NULL
		LIMIT 1
	`, deviceID).Scan(&id, &bookID, &startPage, &startedAt, &createdAt)
	return
}

func mostRecentSessionByDevice(tx *sql.Tx, deviceID string) (id int64, bookID int64, startPage int, endPage *int, startedAt, createdAt string, err error) {
	err = tx.QueryRow(`
		SELECT id, book_id, start_page, end_page, started_at, created_at
		FROM sessions
		WHERE device_id = ?
		ORDER BY started_at DESC
		LIMIT 1
	`, deviceID).Scan(&id, &bookID, &startPage, &endPage, &startedAt, &createdAt)
	return
}

func insertSession(tx *sql.Tx, bookID int64, deviceID string, startPage int, startedAt, createdAt string) (int64, error) {
	res, err := tx.Exec(`
		INSERT INTO sessions (book_id, device_id, start_page, started_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, bookID, deviceID, startPage, startedAt, createdAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func closeSession(tx *sql.Tx, id int64, endedAt string, durationSeconds int64, endPage *int) error {
	if endPage != nil {
		if *endPage < 0 {
			return errors.New("end_page must be >= 0")
		}
		_, err := tx.Exec(`
			UPDATE sessions
			SET end_page = ?, ended_at = ?, duration_seconds = ?
			WHERE id = ?
		`, *endPage, endedAt, durationSeconds, id)
		return err
	}
	_, err := tx.Exec(`
		UPDATE sessions
		SET ended_at = ?, duration_seconds = ?
		WHERE id = ?
	`, endedAt, durationSeconds, id)
	return err
}
