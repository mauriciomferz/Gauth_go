package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// helper for revocation tests posting body directly.
func performAuthorizePost(bs *BetaServer, body any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	reqBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/poa/authorize", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	return w
}

// TestDelegationRevocationHead ensures revoking the head delegation denies authorization.
func TestDelegationRevocationHead(t *testing.T) {
	bs := NewBetaServer("")
	expires := time.Now().Add(2 * time.Minute).Format(time.RFC3339)
	body := map[string]any{
		"client_id":    "c1",
		"principal_id": "p0",
		"ai_agent_id":  "a0",
		"power_type":   "financial_transactions",
		"jurisdiction": "US",
		"scope":        "read,financial_transactions",
		"delegations": []map[string]any{
			{"id": "d1", "subject": "principal-xyz", "delegate": "agent-123", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
			{"id": "d2", "subject": "agent-123", "delegate": "agent-999", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
		},
	}
	// revoke head (d2)
	body["revocations"] = []map[string]any{{"delegation_id": "d2", "reason": "compromise"}}
	resp := performAuthorizePost(bs, body)
	if resp.Code != 400 {
		t.Fatalf("expected 400 revoked head denial, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("delegation_revoked")) {
		t.Fatalf("expected delegation_revoked message body=%s", resp.Body.String())
	}
}

// TestDelegationRevocationMiddle ensures revoking a middle delegation denies authorization.
func TestDelegationRevocationMiddle(t *testing.T) {
	bs := NewBetaServer("")
	expires := time.Now().Add(2 * time.Minute).Format(time.RFC3339)
	body := map[string]any{
		"client_id":    "c1",
		"principal_id": "p0",
		"ai_agent_id":  "a0",
		"power_type":   "financial_transactions",
		"jurisdiction": "US",
		"scope":        "read,financial_transactions",
		"delegations": []map[string]any{
			{"id": "d1", "subject": "principal-xyz", "delegate": "agent-123", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
			{"id": "d2", "subject": "agent-123", "delegate": "agent-456", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
			{"id": "d3", "subject": "agent-456", "delegate": "agent-999", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
		},
	}
	body["revocations"] = []map[string]any{{"delegation_id": "d2", "reason": "policy change"}}
	resp := performAuthorizePost(bs, body)
	if resp.Code != 400 {
		t.Fatalf("expected 400 revoked middle denial, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("delegation_revoked")) {
		t.Fatalf("expected delegation_revoked message body=%s", resp.Body.String())
	}
}

// TestDelegationRevocationNone ensures no revocations allow success.
func TestDelegationRevocationNone(t *testing.T) {
	bs := NewBetaServer("")
	expires := time.Now().Add(2 * time.Minute).Format(time.RFC3339)
	body := map[string]any{
		"client_id":    "c1",
		"principal_id": "p0",
		"ai_agent_id":  "a0",
		"power_type":   "financial_transactions",
		"jurisdiction": "US",
		"scope":        "read,financial_transactions",
		"delegations": []map[string]any{
			{"id": "d1", "subject": "principal-xyz", "delegate": "agent-123", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
			{"id": "d2", "subject": "agent-123", "delegate": "agent-999", "scope": map[string]any{"resource": "acct", "action": "read"}, "expires_at": expires},
		},
	}
	resp := performAuthorizePost(bs, body)
	if resp.Code != 200 {
		t.Fatalf("expected 200 success got %d body=%s", resp.Code, resp.Body.String())
	}
	if bytes.Contains(resp.Body.Bytes(), []byte("delegation_revoked")) {
		t.Fatalf("unexpected revocation message in body=%s", resp.Body.String())
	}
}
