package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mk-slmn/booksmart/services/api/handlers"
)

func TestOpenSession_ReturnsOpen(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	start := map[string]any{
		"device_id":  "phone",
		"book_title": "Dune",
		"start_page": 7,
	}
	b, _ := json.Marshal(start)
	req := httptest.NewRequest(http.MethodPost, "/v1/session/start", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("start expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/sessions/open?device_id=phone", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("open expected 200, got %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestOpenSession_404WhenNone(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/open?device_id=unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestOpenSession_400WhenMissingDeviceID(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/open", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}
