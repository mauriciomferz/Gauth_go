package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper to perform POST JSON
func doPOST(t *testing.T, srv *BetaServer, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	return rr
}

func TestTokenStatusTransitions(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	// 1. Create token
	rrCreate := doPOST(t, srv, "/api/v1/token/create", `{"ttl_seconds":120}`)
	if rrCreate.Code != 201 {
		t.Fatalf("create expected 201 got %d", rrCreate.Code)
	}
	var respCreate struct {
		Success bool `json:"success"`
		Token   struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"token"`
	}
	if err := json.Unmarshal(rrCreate.Body.Bytes(), &respCreate); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if !respCreate.Success || respCreate.Token.ID == "" {
		t.Fatalf("invalid create response")
	}
	if respCreate.Token.Status != statusActive {
		t.Fatalf("expected active status, got %s", respCreate.Token.Status)
	}
	tokenID := respCreate.Token.ID

	// 2. Suspend token
	rrSuspend := doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+tokenID+`","new_status":"suspended"}`)
	if rrSuspend.Code != 200 {
		t.Fatalf("suspend expected 200 got %d", rrSuspend.Code)
	}
	var respSuspend struct {
		Success   bool
		NewStatus string `json:"new_status"`
		TokenID   string `json:"token_id"`
	}
	_ = json.Unmarshal(rrSuspend.Body.Bytes(), &respSuspend)
	if !respSuspend.Success || respSuspend.NewStatus != statusSuspended {
		t.Fatalf("suspend failed: %+v", respSuspend)
	}

	// 3. Validate suspended -> should return suspended status
	rrValidateSusp := doPOST(t, srv, "/api/v1/token/validate", `{"token_id":"`+tokenID+`"}`)
	if rrValidateSusp.Code != 200 {
		t.Fatalf("validate suspended expected 200 got %d", rrValidateSusp.Code)
	}
	var respValidateSusp map[string]any
	_ = json.Unmarshal(rrValidateSusp.Body.Bytes(), &respValidateSusp)
	if respValidateSusp["status"] != statusSuspended {
		t.Fatalf("expected suspended validation status, got %v", respValidateSusp["status"])
	}

	// 4. Reactivate token
	rrActivate := doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+tokenID+`","new_status":"active"}`)
	if rrActivate.Code != 200 {
		t.Fatalf("activate expected 200 got %d", rrActivate.Code)
	}
	var respActivate struct {
		Success   bool
		NewStatus string `json:"new_status"`
	}
	_ = json.Unmarshal(rrActivate.Body.Bytes(), &respActivate)
	if !respActivate.Success || respActivate.NewStatus != statusActive {
		t.Fatalf("reactivate failed")
	}

	// 5. Terminate token from active
	rrTerminate := doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+tokenID+`","new_status":"terminated"}`)
	if rrTerminate.Code != 200 {
		t.Fatalf("terminate expected 200 got %d", rrTerminate.Code)
	}
	var respTerminate struct {
		Success   bool
		NewStatus string `json:"new_status"`
	}
	_ = json.Unmarshal(rrTerminate.Body.Bytes(), &respTerminate)
	if !respTerminate.Success || respTerminate.NewStatus != statusTerminated {
		t.Fatalf("terminate failed")
	}

	// 6. Attempt invalid transition terminated->active (should 409)
	rrTermInvalid := doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+tokenID+`","new_status":"active"}`)
	if rrTermInvalid.Code != 409 {
		t.Fatalf("expected 409 for terminated->active got %d", rrTermInvalid.Code)
	}

	// 7. Validate terminated -> should report revoked status mapping (Validate returns revoked for terminated)
	rrValidateTerm := doPOST(t, srv, "/api/v1/token/validate", `{"token_id":"`+tokenID+`"}`)
	if rrValidateTerm.Code != 200 {
		t.Fatalf("validate terminated expected 200 got %d", rrValidateTerm.Code)
	}
	var respValidateTerm map[string]any
	_ = json.Unmarshal(rrValidateTerm.Body.Bytes(), &respValidateTerm)
	if respValidateTerm["status"] != "revoked" {
		t.Fatalf("expected revoked status for terminated token validation, got %v", respValidateTerm["status"])
	}
}
