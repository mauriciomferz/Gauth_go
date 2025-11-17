package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// timelineResponse mirrors minimal JSON returned by apiLifecycleTimeline
type timelineResponse struct {
	Success bool            `json:"success"`
	Count   int             `json:"count"`
	Events  []timelineEvent `json:"events"`
}

type timelineEvent struct {
	ID         string `json:"id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	OldStatus  string `json:"old_status"`
	NewStatus  string `json:"new_status"`
	Outcome    string `json:"outcome"`
	Reason     string `json:"reason"`
}

// TestLifecycleTimelineBasic exercises initialization, noop and status_change events for token and delegation flows.
func TestLifecycleTimelineBasic(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })

	// 1. Initialize a delegation and token status transitions to populate events
	// Delegation init
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/delegation/status/update", jsonBody(t, map[string]string{"delegation_id": "delA", "new_status": statusActive}))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 delegation init got %d body=%s", w.Code, w.Body.String())
	}

	// Delegation noop (same status)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/delegation/status/update", jsonBody(t, map[string]string{"delegation_id": "delA", "new_status": statusActive}))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 delegation noop got %d body=%s", w.Code, w.Body.String())
	}

	// Create internal token first
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/token/create", jsonBody(t, map[string]any{"subject": "alice@example.com"}))
	srv.router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201 token create got %d body=%s", w.Code, w.Body.String())
	}
	var createResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	tokObj, ok := createResp["token"].(map[string]any)
	if !ok {
		t.Fatalf("create response missing token object: %v", createResp)
	}
	tokIDAny, ok := tokObj["id"]
	if !ok {
		t.Fatalf("token object missing id: %v", tokObj)
	}
	tokID, _ := tokIDAny.(string)
	if tokID == "" {
		t.Fatalf("empty token_id")
	}
	// Token status transitions
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/token/status/update", jsonBody(t, map[string]string{"token_id": tokID, "new_status": statusActive}))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 token init got %d body=%s", w.Code, w.Body.String())
	}
	// Token suspend
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/token/status/update", jsonBody(t, map[string]string{"token_id": tokID, "new_status": statusSuspended}))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 token suspend got %d body=%s", w.Code, w.Body.String())
	}
	// Token terminate
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/token/status/update", jsonBody(t, map[string]string{"token_id": tokID, "new_status": "terminated"}))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 token terminate got %d body=%s", w.Code, w.Body.String())
	}

	// 2. Query timeline filtered by entity_id & entity_type (delegation)
	q := url.Values{}
	q.Set("entity_type", "delegation")
	q.Set("entity_id", "delA")
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/beta/lifecycle/timeline?"+q.Encode(), nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 timeline delegation got %d body=%s", w.Code, w.Body.String())
	}
	var td timelineResponse
	if err := json.Unmarshal(w.Body.Bytes(), &td); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !td.Success || td.Count == 0 {
		t.Fatalf("expected success events>0 got %+v", td)
	}
	// Expect last event outcome either success or noop
	if td.Events[0].EntityType != "delegation" {
		t.Fatalf("unexpected entity_type %s", td.Events[0].EntityType)
	}

	// 3. Query token events limited
	q = url.Values{}
	q.Set("entity_type", "token")
	q.Set("entity_id", tokID)
	q.Set("limit", "2")
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/beta/lifecycle/timeline?"+q.Encode(), nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 token timeline got %d body=%s", w.Code, w.Body.String())
	}
	var tt timelineResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tt); err != nil {
		t.Fatalf("unmarshal token timeline error: %v", err)
	}
	if tt.Count != 2 {
		t.Fatalf("expected limit=2 count got %d", tt.Count)
	}
	for _, ev := range tt.Events {
		if ev.EntityType != "token" {
			t.Fatalf("mixed entity types in token timeline: %+v", ev)
		}
	}

	// 4. Since filter (future time should yield zero) to exercise filter path
	future := time.Now().Add(2 * time.Hour).Unix()
	q = url.Values{}
	q.Set("since", fmtInt64(future))
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/beta/lifecycle/timeline?"+q.Encode(), nil)
	srv.router.ServeHTTP(w, req)
	var tf timelineResponse
	if w.Code != 200 {
		t.Fatalf("expected 200 future since got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tf); err != nil {
		t.Fatalf("unmarshal future since error: %v", err)
	}
	if tf.Count != 0 {
		t.Fatalf("expected zero events for future since got %d", tf.Count)
	}
}

// jsonBody helper constructs request body.
func jsonBody(t *testing.T, v any) *readCloserBuffer {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return newReadCloserBuffer(b)
}

// fmtInt64 simple helper without importing strconv again (already in main file).
func fmtInt64(i int64) string { return fmt.Sprintf("%d", i) }

// Minimal readCloser implementation for test bodies.
type readCloserBuffer struct {
	data []byte
	off  int
}

func newReadCloserBuffer(b []byte) *readCloserBuffer { return &readCloserBuffer{data: b} }
func (r *readCloserBuffer) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}
func (r *readCloserBuffer) Close() error { return nil }
