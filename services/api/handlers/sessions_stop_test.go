package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mk-slmn/booksmart/services/api/handlers"
)

func TestSessionStop_ClosesSessionAndComputesDuration(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	start := map[string]any{
		"device_id":  "iphone-14",
		"book_title": "Dune",
		"start_page": 1,
		"started_at": "2025-09-16T23:00:00Z",
	}
	bs, _ := json.Marshal(start)
	reqS := httptest.NewRequest(http.MethodPost, "/v1/session/start", bytes.NewReader(bs))
	reqS.Header.Set("Content-Type", "application/json")
	ws := httptest.NewRecorder()
	r.ServeHTTP(ws, reqS)
	if ws.Code != http.StatusCreated {
		t.Fatalf("start expected 201, got %d body=%s", ws.Code, ws.Body.String())
	}

	stop := map[string]any{
		"device_id": "iphone-14",
		"end_page":  25,
		"ended_at":  "2025-09-16T23:20:00Z",
	}
	bt, _ := json.Marshal(stop)
	reqT := httptest.NewRequest(http.MethodPost, "/v1/session/stop", bytes.NewReader(bt))
	reqT.Header.Set("Content-Type", "application/json")
	wt := httptest.NewRecorder()
	r.ServeHTTP(wt, reqT)

	if wt.Code != http.StatusOK {
		t.Fatalf("stop expected 200, got %d body=%s", wt.Code, wt.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(wt.Body.Bytes(), &resp)

	if resp["device_id"] != "iphone-14" {
		t.Fatalf("device mismatch: %#v", resp["device_id"])
	}
	if resp["book_title"] != "Dune" {
		t.Fatalf("book_title mismatch: %#v", resp["book_title"])
	}
	if resp["end_page"] != float64(25) {
		t.Fatalf("end_page mismatch: %#v", resp["end_page"])
	}
	if resp["duration_seconds"] != float64(20*60) {
		t.Fatalf("duration_seconds mismatch: %#v", resp["duration_seconds"])
	}
	if resp["ended_at"] == nil {
		t.Fatalf("expected ended_at in response")
	}
}

func TestSessionStop_NotFoundIfNoOpenSession(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	stop := map[string]any{
		"device_id": "ipad",
	}
	b, _ := json.Marshal(stop)
	req := httptest.NewRequest(http.MethodPost, "/v1/session/stop", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}
