package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// helper to perform authorize POST
func performAuthorize(bs *BetaServer, body any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	reqBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/poa/authorize", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	return w
}

// TestDelegationAuthorizeSuccess builds a 2-link narrowing chain and expects chain_verified.
func TestDelegationAuthorizeSuccess(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	expires := time.Now().Add(2 * time.Minute).Format(time.RFC3339)
	body := map[string]any{
		"client_id":    "c1",
		"principal_id": "p0",
		"ai_agent_id":  "a0",
		"power_type":   "financial_transactions",
		"jurisdiction": "US",
		"scope":        "read,financial_transactions", // include delegated action token
		"delegations": []map[string]any{
			{"id": "d1", "subject": "principal-xyz", "delegate": "agent-123", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
			{"id": "d2", "subject": "agent-123", "delegate": "agent-999", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
		},
	}
	resp := performAuthorize(bs, body)
	if resp.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("\"chain_verified\":true")) {
		t.Fatalf("expected chain_verified true, body=%s", resp.Body.String())
	}
}

// TestDelegationAuthorizeScopeWidening ensures widening causes 400.
func TestDelegationAuthorizeScopeWidening(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	expires := time.Now().Add(2 * time.Minute).Format(time.RFC3339)
	body := map[string]any{
		"client_id":    "c1",
		"principal_id": "p0",
		"ai_agent_id":  "a0",
		"power_type":   "financial_transactions",
		"jurisdiction": "US",
		"scope":        "read,financial_transactions", // requested scope includes read only
		"delegations": []map[string]any{
			{"id": "d1", "subject": "principal-xyz", "delegate": "agent-123", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
			// widening action from read to write should fail
			{"id": "d2", "subject": "agent-123", "delegate": "agent-999", "scope": map[string]any{"resource": "acct", "action": "write"}, "expires_at": expires},
		},
	}
	resp := performAuthorize(bs, body)
	if resp.Code != 400 {
		t.Fatalf("expected 400 widening failure, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("delegation scope widening")) {
		t.Fatalf("expected delegation scope widening error body=%s", resp.Body.String())
	}
}

// TestDelegationAuthorizeScopeViolation ensures request scope lacking delegated action fails.
func TestDelegationAuthorizeScopeViolation(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	expires := time.Now().Add(2 * time.Minute).Format(time.RFC3339)
	body := map[string]any{
		"client_id":    "c1",
		"principal_id": "p0",
		"ai_agent_id":  "a0",
		"power_type":   "financial_transactions",
		"jurisdiction": "US",
		// omit 'read' token while delegation has action=read
		"scope": "financial_transactions",
		"delegations": []map[string]any{
			{"id": "d1", "subject": "principal-xyz", "delegate": "agent-123", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
		},
	}
	resp := performAuthorize(bs, body)
	if resp.Code != 400 {
		t.Fatalf("expected 400 scope violation, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("delegation_scope_violation")) {
		t.Fatalf("expected delegation_scope_violation error body=%s", resp.Body.String())
	}
}

// TestDelegationAuthorizeExpired ensures expired delegation rejected.
func TestDelegationAuthorizeExpired(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	expired := time.Now().Add(-1 * time.Minute).Format(time.RFC3339)
	body := map[string]any{
		"client_id":    "c1",
		"principal_id": "p0",
		"ai_agent_id":  "a0",
		"power_type":   "financial_transactions",
		"jurisdiction": "US",
		"scope":        "read,financial_transactions",
		"delegations": []map[string]any{
			{"id": "d1", "subject": "principal-xyz", "delegate": "agent-123", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expired},
		},
	}
	resp := performAuthorize(bs, body)
	if resp.Code != 400 {
		t.Fatalf("expected 400 expired failure, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("delegation append failed")) {
		t.Fatalf("expected append failed error body=%s", resp.Body.String())
	}
}
