package web

import (
	"encoding/json"
	"net/http"
	"testing"
)

// NOTE: uses doPOST helper defined in token_status_test.go (same package)

func TestDelegationStatusLifecycle(t *testing.T) {
	srv := NewBetaServer("")
	// 1. Initialize delegation with active
	rrInit := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"del1","new_status":"active"}`)
	if rrInit.Code != http.StatusOK {
		t.Fatalf("init active expected 200 got %d", rrInit.Code)
	}
	var initResp struct {
		Success      bool
		DelegationID string `json:"delegation_id"`
		OldStatus    string `json:"old_status"`
		NewStatus    string `json:"new_status"`
		Initialized  bool   `json:"initialized"`
	}
	if err := json.Unmarshal(rrInit.Body.Bytes(), &initResp); err != nil {
		t.Fatalf("decode init: %v", err)
	}
	if !initResp.Success || !initResp.Initialized || initResp.NewStatus != "active" || initResp.OldStatus != "" {
		t.Fatalf("unexpected init resp: %+v", initResp)
	}

	// 2. active -> suspended
	rrSusp := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"del1","new_status":"suspended"}`)
	if rrSusp.Code != http.StatusOK {
		t.Fatalf("suspend expected 200 got %d", rrSusp.Code)
	}
	var suspResp struct {
		Success   bool   `json:"success"`
		OldStatus string `json:"old_status"`
		NewStatus string `json:"new_status"`
	}
	if err := json.Unmarshal(rrSusp.Body.Bytes(), &suspResp); err != nil {
		t.Fatalf("decode suspend: %v", err)
	}
	if !suspResp.Success || suspResp.OldStatus != "active" || suspResp.NewStatus != "suspended" {
		t.Fatalf("bad suspend resp: %+v", suspResp)
	}

	// 3. suspended -> active (reactivate)
	rrReact := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"del1","new_status":"active"}`)
	if rrReact.Code != http.StatusOK {
		t.Fatalf("reactivate expected 200 got %d", rrReact.Code)
	}
	var reactResp struct {
		Success   bool   `json:"success"`
		OldStatus string `json:"old_status"`
		NewStatus string `json:"new_status"`
	}
	if err := json.Unmarshal(rrReact.Body.Bytes(), &reactResp); err != nil {
		t.Fatalf("decode reactivate: %v", err)
	}
	if !reactResp.Success || reactResp.OldStatus != "suspended" || reactResp.NewStatus != "active" {
		t.Fatalf("bad reactivate resp: %+v", reactResp)
	}

	// 4. active -> terminated
	rrTerm := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"del1","new_status":"terminated"}`)
	if rrTerm.Code != http.StatusOK {
		t.Fatalf("terminate expected 200 got %d", rrTerm.Code)
	}
	var termResp struct {
		Success   bool   `json:"success"`
		OldStatus string `json:"old_status"`
		NewStatus string `json:"new_status"`
	}
	if err := json.Unmarshal(rrTerm.Body.Bytes(), &termResp); err != nil {
		t.Fatalf("decode terminate: %v", err)
	}
	if !termResp.Success || termResp.OldStatus != "active" || termResp.NewStatus != "terminated" {
		t.Fatalf("bad terminate resp: %+v", termResp)
	}

	// 5. terminated -> active should 409
	rrInvalid := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"del1","new_status":"active"}`)
	if rrInvalid.Code != http.StatusConflict {
		t.Fatalf("expected 409 for terminated->active got %d", rrInvalid.Code)
	}
}

func TestDelegationStatusInitializeTerminated(t *testing.T) {
	srv := NewBetaServer("")
	rrInit := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"delT","new_status":"terminated"}`)
	if rrInit.Code != http.StatusOK {
		t.Fatalf("init terminated expected 200 got %d", rrInit.Code)
	}
	// Attempt suspended (should 409)
	rrSusp := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"delT","new_status":"suspended"}`)
	if rrSusp.Code != http.StatusConflict {
		t.Fatalf("expected 409 after terminated init got %d", rrSusp.Code)
	}
}

func TestDelegationStatusUnsupported(t *testing.T) {
	srv := NewBetaServer("")
	rrBad := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"delX","new_status":"disabled"}`)
	if rrBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 unsupported status got %d", rrBad.Code)
	}
}

func TestDelegationStatusNoChange(t *testing.T) {
	srv := NewBetaServer("")
	rrInit := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"delNC","new_status":"active"}`)
	if rrInit.Code != http.StatusOK {
		t.Fatalf("init active expected 200 got %d", rrInit.Code)
	}
	rrNoChange := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"delNC","new_status":"active"}`)
	if rrNoChange.Code != http.StatusOK {
		t.Fatalf("no-change expected 200 got %d", rrNoChange.Code)
	}
	var ncResp struct {
		Success   bool   `json:"success"`
		NoChange  bool   `json:"no_change"`
		OldStatus string `json:"old_status"`
		NewStatus string `json:"new_status"`
	}
	if err := json.Unmarshal(rrNoChange.Body.Bytes(), &ncResp); err != nil {
		t.Fatalf("decode no-change: %v", err)
	}
	if !ncResp.Success || !ncResp.NoChange || ncResp.OldStatus != "active" || ncResp.NewStatus != "active" {
		t.Fatalf("unexpected no-change resp: %+v", ncResp)
	}
}
