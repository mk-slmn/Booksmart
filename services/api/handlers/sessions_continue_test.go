package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mk-slmn/booksmart/services/api/handlers"
)

func TestSessionContinue_ReturnsOpenIfExists(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	start := map[string]any{
		"device_id":  "phone",
		"book_title": "Dune",
		"start_page": 5,
	}
	bs, _ := json.Marshal(start)
	reqS := httptest.NewRequest(http.MethodPost, "/v1/session/start", bytes.NewReader(bs))
	reqS.Header.Set("Content-Type", "application/json")
	ws := httptest.NewRecorder()
	r.ServeHTTP(ws, reqS)
	if ws.Code != http.StatusCreated {
		t.Fatalf("start expected 201, got %d body=%s", ws.Code, ws.Body.String())
	}

	cont := map[string]any{"device_id": "phone"}
	bc, _ := json.Marshal(cont)
	reqC := httptest.NewRequest(http.MethodPost, "/v1/session/continue", bytes.NewReader(bc))
	reqC.Header.Set("Content-Type", "application/json")
	wc := httptest.NewRecorder()
	r.ServeHTTP(wc, reqC)
	if wc.Code != http.StatusOK {
		t.Fatalf("continue expected 200 when open exists, got %d body=%s", wc.Code, wc.Body.String())
	}
}

func TestSessionContinue_CreatesNewFromLastClosed(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	start := map[string]any{
		"device_id":  "laptop",
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
		"device_id": "laptop",
		"end_page":  20,
		"ended_at":  "2025-09-16T23:30:00Z",
	}
	bt, _ := json.Marshal(stop)
	reqT := httptest.NewRequest(http.MethodPost, "/v1/session/stop", bytes.NewReader(bt))
	reqT.Header.Set("Content-Type", "application/json")
	wt := httptest.NewRecorder()
	r.ServeHTTP(wt, reqT)
	if wt.Code != http.StatusOK {
		t.Fatalf("stop expected 200, got %d body=%s", wt.Code, wt.Body.String())
	}

	cont := map[string]any{"device_id": "laptop", "started_at": "2025-09-16T23:40:00Z"}
	bc, _ := json.Marshal(cont)
	reqC := httptest.NewRequest(http.MethodPost, "/v1/session/continue", bytes.NewReader(bc))
	reqC.Header.Set("Content-Type", "application/json")
	wc := httptest.NewRecorder()
	r.ServeHTTP(wc, reqC)
	if wc.Code != http.StatusCreated {
		t.Fatalf("continue expected 201, got %d body=%s", wc.Code, wc.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(wc.Body.Bytes(), &resp)
	if resp["start_page"] != float64(20) {
		t.Fatalf("expected new start_page 20, got %#v", resp["start_page"])
	}
}

func TestSessionContinue_404WhenNoHistory(t *testing.T) {
	db := newTestDB(t)
	r := handlers.NewServer(db)

	cont := map[string]any{"device_id": "unknown"}
	bc, _ := json.Marshal(cont)
	reqC := httptest.NewRequest(http.MethodPost, "/v1/session/continue", bytes.NewReader(bc))
	reqC.Header.Set("Content-Type", "application/json")
	wc := httptest.NewRecorder()
	r.ServeHTTP(wc, reqC)

	if wc.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", wc.Code, wc.Body.String())
	}
}
