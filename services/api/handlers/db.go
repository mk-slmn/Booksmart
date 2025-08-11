package handlers

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func OpenDB() (*sql.DB, error) {
	path := os.Getenv("SQLITE_PATH")
	if path == "" {
		path = "reading.sqlite"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(ReadSchema()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func ReadSchema() string {
	return `	
	PRAGMA journal_mode=WAL;
	PRAGMA foreign_keys=ON;
	PRAGMA busy_timeout=3000;

	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL UNIQUE,
		author TEXT,
		source TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY, 
		book_id INTEGER NOT NULL REFERENCES books(id),
		device_id TEXT NOT NULL,
		start_page INTEGER NOT NULL CHECK (start_page >= 0),
		end_page INTEGER CHECK (end_page IS NULL OR end_page >= 0),
		started_at TEXT NOT NULL, -- RFC3339 UTC
		ended_at TEXT, -- RFC3339 UTC
		duration_seconds INTEGER, -- set when stopping
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_device_open
		ON sessions(device_id)
		WHERE ended_at IS NULL;
	
	CREATE INDEX IF NOT EXISTS idx_sessions_book_started
		ON sessions(book_id, started_at DESC);
	
	CREATE INDEX IF NOT EXISTS idx_sessions_date
		ON sessions(started_at);`
}
