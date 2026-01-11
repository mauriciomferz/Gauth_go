package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// TestPolicyProvenanceTamperDetection ensures provenance endpoint reports verification failure after in-memory tampering.
func TestPolicyProvenanceTamperDetection(t *testing.T) {
	t.Setenv("AGENTAUTH_POLICY_ADMIN_TOKEN", "adm")

	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })

	// helper to POST bundle
	post := func(body string, want int) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/policy/bundles", bytesReader(body))
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
		req := httptest.NewRequest(http.MethodGet, "/api/v1/policy/provenance", nil)
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
	if srv.policyHandler.Store == nil {
		t.Fatalf("Store is nil")
	}
	ctx := context.TODO()
	chainHashes, _ := srv.policyHandler.Store.ChainHashes(ctx)
	if len(chainHashes) < 2 {
		t.Fatalf("expected at least 2 bundles")
	}
	// Access underlying registry via reflection of known field; we can mutate by retrieving head then altering PrevHash
	// Cast to InMemoryStore to access deprecated Registry() method for tampering
	memStore, ok := srv.policyHandler.Store.(*policy.InMemoryStore)
	if !ok {
		t.Fatalf("Store is not InMemoryStore, cannot tamper")
	}
	head := memStore.Registry().Head()
	if head == nil {
		t.Fatalf("expected head bundle")
	}
	head.PrevHash = "deadbeef" // break link expectation for genesis vs second

	if v, e := prov(); v || e == "" {
		t.Fatalf("expected verification failure after tamper; v=%v err=%s", v, e)
	}
}
