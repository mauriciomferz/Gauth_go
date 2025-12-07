package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	policyHandler "github.com/mauriciomferz/Gauth_go/web/handlers/policy"
)

const (
	policyEvalAllowJSON   = `{"subject":"alice@example.com","action":"read","resource":"report:finance","attrs":{"department":"finance"}}`
	policyEvalMissingAttr = `{"subject":"alice@example.com","action":"read","resource":"report:finance","attrs":{}}`
)

// helper to perform a request and decode JSON
func performJSON(t *testing.T, srv *BetaServer, method, path, body string, hdr map[string]string, out interface{}, wantStatus int) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, w.Code, wantStatus, w.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("decode json: %v body=%s", err, w.Body.String())
		}
	}
}

//nolint:gocyclo // Policy lifecycle integration test
func TestPolicyLifecycleIntegration(t *testing.T) {
	// Ensure admin token for authorization path
	os.Setenv("GAUTH_POLICY_ADMIN_TOKEN", "test-admin")
	defer os.Unsetenv("GAUTH_POLICY_ADMIN_TOKEN")
	// Disable automatic demo bundle seeding to start from empty chain for lifecycle assertions
	os.Setenv("GAUTH_SEED_POLICY", "0")
	defer os.Unsetenv("GAUTH_SEED_POLICY")

	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })

	// 1. provenance should be empty
	var prov struct {
		Success  bool     `json:"success"`
		HeadHash string   `json:"head_hash"`
		Chain    []string `json:"chain"`
		Verified bool     `json:"verified"`
	}
	performJSON(t, srv, http.MethodGet, "/api/v1/policy/provenance", "", nil, &prov, 200)
	if len(prov.Chain) != 0 || prov.HeadHash != "" {
		t.Fatalf("expected empty chain, got %+v", prov)
	}

	// 2. unauthorized submission (missing header)
	var unauthResp map[string]any
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", `{"id":"b1","policies":[]}`, nil, &unauthResp, 401)

	// 3. invalid bundle (empty policies) with auth header
	var invalid map[string]any
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", `{"id":"b1","policies":[]}`, map[string]string{"X-Admin-Token": "test-admin"}, &invalid, 400)
	msg, ok := invalid["message"].(string)
	if !ok || !strings.Contains(msg, "at least one policy") {
		t.Fatalf("unexpected validation message: %v", invalid)
	}

	// 4. valid bundle submission
	bundleBody := `{"id":"b1","policies":[{"id":"p1","subjects":["alice@example.com"],"rules":[{"actions":["read"],"resources":["report:finance"],"effect":"allow","expr":"department == 'finance'"}]}]}`
	var addResp struct {
		Success    bool     `json:"success"`
		BundleHash string   `json:"bundle_hash"`
		Chain      []string `json:"chain"`
		Verified   bool     `json:"verified"`
	}
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", bundleBody, map[string]string{"X-Admin-Token": "test-admin"}, &addResp, 201)
	if !addResp.Success || addResp.BundleHash == "" || len(addResp.Chain) != 1 {
		t.Fatalf("unexpected add response: %+v", addResp)
	}

	headHash := addResp.BundleHash

	// 5. provenance now should reflect head
	var prov2 struct {
		HeadHash string   `json:"head_hash"`
		Chain    []string `json:"chain"`
		Verified bool     `json:"verified"`
	}
	performJSON(t, srv, http.MethodGet, "/api/v1/policy/provenance", "", nil, &prov2, 200)
	if prov2.HeadHash != headHash || len(prov2.Chain) != 1 {
		t.Fatalf("unexpected provenance after add: %+v", prov2)
	}

	// 6. fetch bundle by hash
	var getBundle struct {
		Success bool `json:"success"`
		Bundle  struct {
			Hash     string `json:"hash"`
			ID       string `json:"id"`
			PrevHash string `json:"prev_hash"`
		} `json:"bundle"`
	}
	performJSON(t, srv, http.MethodGet, "/api/v1/policy/bundles/"+headHash, "", nil, &getBundle, 200)
	if getBundle.Bundle.Hash != headHash || getBundle.Bundle.ID != "b1" {
		t.Fatalf("unexpected get bundle: %+v", getBundle)
	}

	// 7. evaluate allow
	evalBody := policyEvalAllowJSON
	var evalResp struct {
		Success    bool   `json:"success"`
		Allow      bool   `json:"allow"`
		BundleHash string `json:"bundle_hash"`
		ChainHead  string `json:"chain_head"`
	}
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/evaluate", evalBody, nil, &evalResp, 200)
	if !evalResp.Allow || evalResp.BundleHash != headHash || evalResp.ChainHead != headHash {
		t.Fatalf("unexpected eval allow resp: %+v", evalResp)
	}

	// 7b. negative evaluation (missing attr -> should NOT match expression -> deny by default)
	evalBodyMissing := policyEvalMissingAttr
	var evalRespMissing struct {
		Allow   bool     `json:"allow"`
		Deny    bool     `json:"deny"`
		Matched []string `json:"matched"`
	}
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/evaluate", evalBodyMissing, nil, &evalRespMissing, 200)
	if evalRespMissing.Allow || len(evalRespMissing.Matched) != 0 {
		t.Fatalf("expected deny (no match) missing attrs: %+v", evalRespMissing)
	}

	// 8. add deny bundle overriding (different hash)
	denyBody := `{"id":"b2","policies":[{"id":"p2","subjects":["alice@example.com"],"rules":[{"actions":["read"],"resources":["report:finance"],"effect":"deny"}]}]}`
	var denyAdd struct {
		BundleHash string   `json:"bundle_hash"`
		Chain      []string `json:"chain"`
	}
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", denyBody, map[string]string{"X-Admin-Token": "test-admin"}, &denyAdd, 201)
	if len(denyAdd.Chain) != 2 {
		t.Fatalf("expected chain length 2, got %+v", denyAdd)
	}

	// 9. evaluate now deny
	var evalRespD struct {
		Allow bool `json:"allow"`
		Deny  bool `json:"deny"`
	}
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/evaluate", evalBody, nil, &evalRespD, 200)
	if evalRespD.Allow || !evalRespD.Deny {
		t.Fatalf("expected deny after second bundle, got %+v", evalRespD)
	}

	// 9b. audit log should contain at least one evaluation entry with provenance metadata
	// Query audit logs endpoint
	var auditResp struct {
		Success bool `json:"success"`
		Entries []struct {
			Action string            `json:"action"`
			Meta   map[string]string `json:"meta"`
		} `json:"entries"`
	}
	performJSON(t, srv, http.MethodGet, "/api/v1/audit/logs", "", nil, &auditResp, 200)
	foundEval := false
	for _, e := range auditResp.Entries {
		if e.Action == "evaluate" && e.Meta["bundle_hash"] != "" && e.Meta["chain_head"] != "" {
			foundEval = true
			break
		}
	}
	if !foundEval {
		t.Fatalf("expected at least one audit evaluation entry with provenance metadata")
	}

	// 10. exercise rate limit quickly (override limiter to small window)
	srv.policyHandler.RateLimiter = policyHandler.NewSimpleRateLimiter(1, time.Minute) // allow only one submission
	var rlAdd map[string]any
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", denyBody, map[string]string{"X-Admin-Token": "test-admin"}, &rlAdd, 201) // first ok in new limiter
	var rlHit map[string]any
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", denyBody, map[string]string{"X-Admin-Token": "test-admin"}, &rlHit, 429)
	rlMsg, ok := rlHit["message"].(string)
	if !ok || rlMsg != "rate limit exceeded" {
		t.Fatalf("expected rate limit exceeded, got %+v", rlHit)
	}

	// 10b. simulate window reset by advancing time (replace slot reset manually for test determinism)
	// Directly mutate internal slot for IP (since we know requests come from 192.0.2.1 style default 'ClientIP' -> empty translates to '::1' / '127.0.0.1'). We'll try both.
	srv.policyHandler.RateLimiter.ForceReset()
	var rlAfterReset map[string]any
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", denyBody, map[string]string{"X-Admin-Token": "test-admin"}, &rlAfterReset, 201)

	// 11. malformed JSON (reset limiter first to avoid 429 masking 400)
	srv.policyHandler.RateLimiter = policyHandler.NewSimpleRateLimiter(10, time.Minute)
	var malformed map[string]any
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", `{"id":"oops"`, map[string]string{"X-Admin-Token": "test-admin"}, &malformed, 400)

	// 12. missing required field (no id key) -> validation failure (still under fresh limiter)
	var missing map[string]any
	performJSON(t, srv, http.MethodPost, "/api/v1/policy/bundles", `{"policies":[{"id":"pX","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`, map[string]string{"X-Admin-Token": "test-admin"}, &missing, 400)
}
