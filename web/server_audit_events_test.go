package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuditRecordAndList(t *testing.T) {
	srv := setupTestServer()
	w := httptest.NewRecorder()
	body := map[string]any{"actor": "tester", "action": "login", "resource": "/demo", "outcome": "success"}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/v1/audit/record", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}

	// list
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/audit/logs", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 got %d", w2.Code)
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte("login")) {
		t.Fatalf("expected body to contain action 'login' got %s", w2.Body.String())
	}
}

func TestEventsEmit(t *testing.T) {
	srv := setupTestServer()
	w := httptest.NewRecorder()
	body := map[string]any{"type": "demo_event", "data": map[string]string{"k": "v"}}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/v1/events/emit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
}
