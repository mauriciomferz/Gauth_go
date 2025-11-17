package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestPolicyProvenanceTamperDetection ensures provenance endpoint reports verification failure after in-memory tampering.
func TestPolicyProvenanceTamperDetection(t *testing.T) {
	os.Setenv("GAUTH_POLICY_ADMIN_TOKEN", "adm")
	defer os.Unsetenv("GAUTH_POLICY_ADMIN_TOKEN")
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })

	// helper to POST bundle
	post := func(body string, want int) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/bundles", bytesReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Token", "adm")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("bundle post got %d want %d body=%s", w.Code, want, w.Body.String())
		}
	}
	// minimal valid bundle 1
	post(`{"id":"b1","policies":[{"id":"p1","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`, 201)
	// second bundle
	post(`{"id":"b2","policies":[{"id":"p2","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`, 201)

	// Sanity: provenance verified true
	prov := func() (verified bool, verr string) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/policy/provenance", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("provenance status %d", w.Code)
		}
		var resp struct {
			Verified          bool   `json:"verified"`
			VerificationError string `json:"verification_error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return resp.Verified, resp.VerificationError
	}
	if v, e := prov(); !v || e != "" {
		t.Fatalf("expected verified chain initially, v=%v err=%s", v, e)
	}

	// Tamper: break prev hash link of second bundle
	if srv.policyRegistry == nil || len(srv.policyRegistry.ChainHashes()) < 2 {
		t.Fatalf("expected at least 2 bundles")
	}
	// Access underlying registry via reflection of known field; we can mutate by retrieving head then altering PrevHash
	// (Simpler: directly set bundles[1].PrevHash = "deadbeef" )
	// Directly tamper: modify PrevHash of second bundle (unsafe test-only access)
	// NOTE: registry.bundles is unexported; we can't reach it directly from here without adding a helper.
	// Instead we exploit that we can alter the head bundle Hash to break link by reassigning Hash field (accessible via pointer returned by Head()).
	head := srv.policyRegistry.Head()
	if head == nil {
		t.Fatalf("expected head bundle")
	}
	head.PrevHash = "deadbeef" // break link expectation for genesis vs second

	if v, e := prov(); v || e == "" {
		t.Fatalf("expected verification failure after tamper; v=%v err=%s", v, e)
	}
}
