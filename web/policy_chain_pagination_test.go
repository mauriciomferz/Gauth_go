package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const evalPayloadMinimal = `{"subject":"u","action":"a","resource":"r","attrs":{}}`

// TestPolicyChainPaginationAndConsistency covers /policy/chain and /policy/audit-consistency endpoints.
func TestPolicyChainPaginationAndConsistency(t *testing.T) {
	srv := NewTestServerNoSeed(t)
	// Seed several bundles
	for i := 0; i < 5; i++ {
		payload := map[string]any{
			"id": "b" + string('A'+rune(i)),
			"policies": []map[string]any{{
				"id":       "p" + string('A'+rune(i)),
				"subjects": []string{"u"},
				"rules": []map[string]any{{
					"actions":   []string{"a"},
					"resources": []string{"r"},
					"effect":    "allow",
				}},
			}},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/bundles", bytesReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("add bundle status %d body=%s", w.Code, w.Body.String())
		}
	}
	// Page 1: offset 0 limit 2
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/beta/policy/chain?offset=0&limit=2", nil)
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("chain page1 status %d", w1.Code)
	}
	var page1 struct {
		Success  bool
		Hashes   []string
		Total    int
		Returned int
		Offset   int
		Limit    int
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &page1); err != nil {
		t.Fatalf("decode page1: %v", err)
	}
	if !page1.Success || page1.Returned != 2 || page1.Offset != 0 || page1.Limit != 2 || page1.Total != 5 {
		t.Fatalf("unexpected page1 %+v", page1)
	}
	if len(page1.Hashes) != 2 {
		t.Fatalf("expected 2 hashes, got %d", len(page1.Hashes))
	}
	// Page 2: offset 2 limit 2
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/beta/policy/chain?offset=2&limit=2", nil)
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("chain page2 status %d", w2.Code)
	}
	var page2 struct {
		Success  bool
		Hashes   []string
		Offset   int
		Returned int
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &page2); err != nil {
		t.Fatalf("decode page2: %v", err)
	}
	if !page2.Success || page2.Offset != 2 || page2.Returned != 2 {
		t.Fatalf("unexpected page2 %+v", page2)
	}
	// Page 3: offset beyond tail
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/beta/policy/chain?offset=10&limit=2", nil)
	w3 := httptest.NewRecorder()
	srv.router.ServeHTTP(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("chain page3 status %d", w3.Code)
	}
	var page3 struct{ Returned int }
	if err := json.Unmarshal(w3.Body.Bytes(), &page3); err != nil {
		t.Fatalf("decode page3: %v", err)
	}
	if page3.Returned != 0 {
		t.Fatalf("expected 0 returned page3, got %d", page3.Returned)
	}

	// Perform a policy evaluation to create audit evaluation entry
	reqEval := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/evaluate", bytesReader(evalPayloadMinimal))
	wEval := httptest.NewRecorder()
	srv.router.ServeHTTP(wEval, reqEval)
	if wEval.Code != 200 {
		t.Fatalf("eval status %d", wEval.Code)
	}

	// Consistency endpoint should be consistent
	reqCons := httptest.NewRequest(http.MethodGet, "/api/v1/beta/policy/audit-consistency", nil)
	wCons := httptest.NewRecorder()
	srv.router.ServeHTTP(wCons, reqCons)
	if wCons.Code != 200 {
		t.Fatalf("consistency status %d", wCons.Code)
	}
	var cons struct {
		Success       bool
		Consistent    bool
		ChainVerified bool `json:"chain_verified"`
	}
	if err := json.Unmarshal(wCons.Body.Bytes(), &cons); err != nil {
		t.Fatalf("decode consistency: %v", err)
	}
	if !cons.Success || !cons.Consistent || !cons.ChainVerified {
		t.Fatalf("unexpected consistency %+v", cons)
	}
}

// TestPolicyAuditConsistencyStale ensures endpoint reports inconsistent when head advanced since last evaluation.
func TestPolicyAuditConsistencyStale(t *testing.T) {
	srv := NewTestServerNoSeed(t)
	// Add first bundle and evaluate
	b1 := `{"id":"b1","policies":[{"id":"p1","subjects":["u"],"rules":[{"actions":["a"],"resources":["r"],"effect":"allow"}]}]}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/bundles", bytesReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, req1)
	if w1.Code != 201 {
		t.Fatalf("add b1 status %d", w1.Code)
	}
	reqEval := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/evaluate", bytesReader(evalPayloadMinimal))
	reqEval.Header.Set("Content-Type", "application/json")
	wEval := httptest.NewRecorder()
	srv.router.ServeHTTP(wEval, reqEval)
	if wEval.Code != 200 {
		t.Fatalf("eval status %d", wEval.Code)
	}
	// Add second bundle (new head)
	b2 := `{"id":"b2","policies":[{"id":"p2","subjects":["u"],"rules":[{"actions":["a"],"resources":["r"],"effect":"allow"}]}]}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/bundles", bytesReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 201 {
		t.Fatalf("add b2 status %d", w2.Code)
	}
	// Consistency should now be false (logged head from evaluation != current head)
	reqC := httptest.NewRequest(http.MethodGet, "/api/v1/beta/policy/audit-consistency", nil)
	wC := httptest.NewRecorder()
	srv.router.ServeHTTP(wC, reqC)
	if wC.Code != 200 {
		t.Fatalf("consistency status %d", wC.Code)
	}
	var resp struct {
		Success    bool
		Consistent bool
	}
	if err := json.Unmarshal(wC.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success || resp.Consistent {
		t.Fatalf("expected inconsistent after head advanced: %+v", resp)
	}
}
