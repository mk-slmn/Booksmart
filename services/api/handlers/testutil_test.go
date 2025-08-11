package handlers_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/mk-slmn/booksmart/services/api/handlers"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory DB: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(handlers.ReadSchema()); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	return db
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	db := newTestDB(t)
	return handlers.NewServer(db)
}
