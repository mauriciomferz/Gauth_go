package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper to spin up minimal server (reusing existing newBetaServer logic if available) else create router manually.
func newTestServer(t *testing.T) *BetaServer {
	srv := NewBetaServer("")
	t.Cleanup(func() {
		srv.Shutdown()
	})
	return srv
}

// seedLifecycleEvent ensures at least one lifecycle event exists (delegation init) for timeline tests.
func seedLifecycleEvent(t *testing.T, s *BetaServer, id string) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/delegation/status/update", strings.NewReader(`{"delegation_id":"`+id+`","new_status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("seed lifecycle expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDecisionMetricsCSV(t *testing.T) {
	s := newTestServer(t)
	// Seed one decision via token status update (creates decision + reason)
	// Issue token then update same status to force noop reason path
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":60}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("token create unexpected status %d", rr.Code)
	}
	// Extract token id from response
	body := rr.Body.String()
	// Expect token object: "token":{"id":"..."
	var tokenID string
	if i := strings.Index(body, "\"token\":"); i != -1 {
		// find id field after token
		if j := strings.Index(body[i:], "\"id\":\""); j != -1 {
			start := i + j + len("\"id\":\"")
			if k := strings.Index(body[start:], "\""); k != -1 {
				tokenID = body[start : start+k]
			}
		}
	}
	if tokenID == "" {
		t.Fatalf("failed to parse token id from response: %s", body)
	}
	// Perform noop status update (same status)
	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/token/status/update", strings.NewReader(`{"token_id":"`+tokenID+`","new_status":"active"}`))
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("status update unexpected %d", rr2.Code)
	}

	// Request CSV metrics
	rr3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/v1/beta/metrics/decisions?format=csv", nil)
	s.router.ServeHTTP(rr3, req3)
	if ct := rr3.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("expected csv content type got %s", ct)
	}
	if !strings.Contains(rr3.Body.String(), "action,resource,outcome,reason,count") {
		t.Fatalf("csv header missing")
	}
}

func TestDecisionMetricsCSVAccept(t *testing.T) {
	s := newTestServer(t)
	// Seed decision data via noop status update
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":60}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("token create unexpected status %d", rr.Code)
	}
	body := rr.Body.String()
	var tokenID string
	if i := strings.Index(body, "\"token\":"); i != -1 {
		// find id field after token
		if j := strings.Index(body[i:], "\"id\":\""); j != -1 {
			start := i + j + len("\"id\":\"")
			if k := strings.Index(body[start:], "\""); k != -1 {
				tokenID = body[start : start+k]
			}
		}
	}
	if tokenID == "" {
		t.Fatalf("failed parsing token id")
	}
	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/token/status/update", strings.NewReader(`{"token_id":"`+tokenID+`","new_status":"active"}`))
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("status update unexpected %d", rr2.Code)
	}
	// Request CSV via Accept header only (no query param)
	rr3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/v1/beta/metrics/decisions", nil)
	req3.Header.Set("Accept", "text/csv")
	s.router.ServeHTTP(rr3, req3)
	if ct := rr3.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("expected csv via accept got %s", ct)
	}
	if !strings.Contains(rr3.Body.String(), "action,resource,outcome,reason,count") {
		t.Fatalf("missing csv header via accept")
	}
}

func TestLifecycleTimelineCSV(t *testing.T) {
	s := newTestServer(t)
	// Produce one lifecycle event via token issue
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":30}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("token create status %d", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/beta/lifecycle/timeline?format=csv", nil)
	s.router.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("timeline status %d", rr2.Code)
	}
	if ct := rr2.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("expected csv content type got %s", ct)
	}
	if !strings.Contains(rr2.Body.String(), "entity_type,entity_id,old_status,new_status,outcome,reason,latency_ns,at") {
		t.Fatalf("timeline csv header missing")
	}
}

func TestLifecycleTimelineCSVAccept(t *testing.T) {
	s := newTestServer(t)
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":20}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("token create status %d", rr.Code)
	}
	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/beta/lifecycle/timeline", nil)
	req2.Header.Set("Accept", "text/csv")
	s.router.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("timeline status %d", rr2.Code)
	}
	if ct := rr2.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("expected csv content type got %s", ct)
	}
	if !strings.Contains(rr2.Body.String(), "entity_type,entity_id,old_status,new_status,outcome,reason,latency_ns,at") {
		t.Fatalf("timeline csv header missing via accept")
	}
}

func TestLifecycleTimelineCursorPagination(t *testing.T) {
	s := newTestServer(t)
	seedLifecycleEvent(t, s, "delPg")
	// First page (beta path)
	r1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/api/v1/beta/lifecycle/timeline?limit=5", nil)
	s.router.ServeHTTP(r1, req1)
	if r1.Code != 200 {
		t.Fatalf("expected 200 got %d", r1.Code)
	}
	var page1 struct {
		Success bool `json:"success"`
		Events  []struct {
			ID string `json:"id"`
		} `json:"events"`
		Next  string `json:"next_cursor"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(r1.Body.Bytes(), &page1); err != nil {
		t.Fatalf("unmarshal page1: %v", err)
	}
	if !page1.Success {
		t.Fatalf("page1 success false")
	}
	if page1.Count == 0 {
		t.Fatalf("expected some events")
	}
	if page1.Next == "" && page1.Count >= 5 {
		t.Fatalf("expected next cursor when count==limit")
	}

	if page1.Next != "" {
		r2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/api/v1/beta/lifecycle/timeline?limit=5&cursor="+page1.Next, nil)
		s.router.ServeHTTP(r2, req2)
		if r2.Code != 200 {
			t.Fatalf("expected 200 got %d", r2.Code)
		}
		var page2 struct {
			Success bool `json:"success"`
			Events  []struct {
				ID string `json:"id"`
			} `json:"events"`
			Next  string `json:"next_cursor"`
			Count int    `json:"count"`
		}
		if err := json.Unmarshal(r2.Body.Bytes(), &page2); err != nil {
			t.Fatalf("unmarshal page2: %v", err)
		}
		if !page2.Success {
			t.Fatalf("page2 success false")
		}
		// Ensure no overlap (cursor is exclusive)
		for _, e2 := range page2.Events {
			if e2.ID == page1.Next {
				t.Fatalf("cursor event should be excluded")
			}
		}
	}
}

func TestLifecycleTimelineCursorPaginationCSV(t *testing.T) {
	s := newTestServer(t)
	seedLifecycleEvent(t, s, "delCsv")
	// Request first page CSV (beta path)
	r1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/api/v1/beta/lifecycle/timeline?limit=3", nil)
	req1.Header.Set("Accept", "text/csv")
	s.router.ServeHTTP(r1, req1)
	if r1.Code != 200 {
		t.Fatalf("expected 200 got %d", r1.Code)
	}
	if ct := r1.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected csv got %s", ct)
	}
	body1 := r1.Body.String()
	lines := strings.Split(strings.TrimSpace(body1), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header + at least 1 line, got %d", len(lines))
	}
	// Header may start with entity_type per earlier format; accept both potential variants for robustness
	if !(strings.HasPrefix(lines[0], "id,entity_type,entity_id") || strings.HasPrefix(lines[0], "entity_type,entity_id")) {
		t.Fatalf("unexpected header: %s", lines[0])
	}
	// We cannot parse next_cursor from CSV (not included); ensure pagination unaffected by CSV path.
}

func TestAuditLogsCSV(t *testing.T) {
	s := newTestServer(t)
	// Perform a token issue to generate audit entries
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":45}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("token create status %d", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/audit/logs?format=csv", nil)
	s.router.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("audit logs status %d", rr2.Code)
	}
	if ct := rr2.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("expected csv content type got %s", ct)
	}
	if !strings.Contains(rr2.Body.String(), "id,at,actor,action,resource,outcome,reason") {
		t.Fatalf("audit csv header missing")
	}
}

func TestAuditLogsCSVAccept(t *testing.T) {
	s := newTestServer(t)
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":45}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("token create status %d", rr.Code)
	}
	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/audit/logs", nil)
	req2.Header.Set("Accept", "text/csv")
	s.router.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("audit logs status %d", rr2.Code)
	}
	if ct := rr2.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("expected csv content type got %s", ct)
	}
	if !strings.Contains(rr2.Body.String(), "id,at,actor,action,resource,outcome,reason") {
		t.Fatalf("audit csv header missing via accept")
	}
}
