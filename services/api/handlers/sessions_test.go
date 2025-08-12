package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mk-slmn/booksmart/services/api/handlers"
)

func TestSessionStart_CreatesBookAndSession(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	body := map[string]any{
		"device_id":  "iphone-14",
		"book_title": "Dune",
		"author":     "Frank Herbert",
		"start_page": 1,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/session/start", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if resp["device_id"] != "iphone-14" {
		t.Fatalf("device_id mismatch: %#v", resp["device_id"])
	}
	if resp["book_title"] != "Dune" {
		t.Fatalf("book_title mismatch: %#v", resp["book_title"])
	}
	if resp["start_page"] != float64(1) { // JSON numbers decode as float64
		t.Fatalf("start_page mismatch: %#v", resp["start_page"])
	}
	if _, ok := resp["id"]; !ok {
		t.Fatalf("expected id in response")
	}
}

func TestSessionStart_ConflictIfOpenSessionExists(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	first := map[string]any{
		"device_id":  "ipad",
		"book_title": "Dune",
		"start_page": 10,
	}
	b1, _ := json.Marshal(first)
	req1 := httptest.NewRequest(http.MethodPost, "/v1/session/start", bytes.NewReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for first start, got %d body=%s", w1.Code, w1.Body.String())
	}

	second := map[string]any{
		"device_id":  "ipad",
		"book_title": "Dune",
		"start_page": 11,
	}
	b2, _ := json.Marshal(second)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/session/start", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d body=%s", w2.Code, w2.Body.String())
	}
}
