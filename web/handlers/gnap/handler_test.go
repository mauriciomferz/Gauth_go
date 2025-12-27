package gnap

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/gnap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDiscovery(t *testing.T) {
	store := gnap.NewMemoryGrantStore()
	handler := NewHandler(store, nil, "http://localhost:8080")

	r := gin.New()
	handler.RegisterRoutes(r)

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("GET", "/.well-known/gnap-as-rs", nil)
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp DiscoveryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if resp.GrantRequestEndpoint != "http://localhost:8080/gnap/tx" {
		t.Errorf("Unexpected endpoint: %s", resp.GrantRequestEndpoint)
	}
	if len(resp.InteractionStart) == 0 {
		t.Error("Expected interaction start modes")
	}
	if len(resp.KeyProofs) == 0 {
		t.Error("Expected key proofs")
	}
}

func TestGrantRequest_Simple(t *testing.T) {
	store := gnap.NewMemoryGrantStore()
	handler := NewHandler(store, nil, "http://localhost:8080")

	r := gin.New()
	handler.RegisterRoutes(r)

	// Create grant request
	req := gnap.GrantRequest{
		AccessToken: &gnap.AccessTokenRequest{
			Access: []gnap.AccessRight{{Type: "read", Actions: []string{"get"}}},
		},
		Client: &gnap.ClientInstance{
			ClassID: "test-app",
		},
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("POST", "/gnap/tx", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp gnap.GrantResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.AccessToken == nil {
		t.Error("Expected access token for simple request without interaction")
	}
	if resp.AccessToken != nil && resp.AccessToken.Value == "" {
		t.Error("Access token value should not be empty")
	}
}

func TestGrantRequest_WithInteraction(t *testing.T) {
	store := gnap.NewMemoryGrantStore()
	handler := NewHandler(store, nil, "http://localhost:8080")

	r := gin.New()
	handler.RegisterRoutes(r)

	// Create grant request with interaction
	req := gnap.GrantRequest{
		AccessToken: &gnap.AccessTokenRequest{
			Access: []gnap.AccessRight{{Type: "admin", Actions: []string{"all"}}},
		},
		Interact: &gnap.InteractionRequest{
			Start: []gnap.InteractionStartMode{gnap.InteractionStartRedirect},
			Finish: &gnap.InteractionFinish{
				Method: gnap.InteractionFinishRedirect,
				URI:    "http://client.example/callback",
				Nonce:  "client-nonce-123",
			},
		},
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("POST", "/gnap/tx", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp gnap.GrantResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should have interaction response, not token
	if resp.AccessToken != nil {
		t.Error("Should not have token when interaction is required")
	}
	if resp.Interact == nil {
		t.Fatal("Expected interaction response")
	}
	if resp.Interact.Redirect == "" {
		t.Error("Expected redirect URI")
	}
	if resp.Continue == nil {
		t.Fatal("Expected continuation info")
	}
	if resp.Continue.URI == "" {
		t.Error("Expected continuation URI")
	}
}

func TestGrantRequest_Invalid(t *testing.T) {
	store := gnap.NewMemoryGrantStore()
	handler := NewHandler(store, nil, "http://localhost:8080")

	r := gin.New()
	handler.RegisterRoutes(r)

	// Empty request should fail
	body := []byte("{}")
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("POST", "/gnap/tx", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestContinue_WithToken(t *testing.T) {
	store := gnap.NewMemoryGrantStore()
	handler := NewHandler(store, nil, "http://localhost:8080")

	r := gin.New()
	handler.RegisterRoutes(r)

	// First create a grant
	req := gnap.GrantRequest{
		AccessToken: &gnap.AccessTokenRequest{},
		Interact:    &gnap.InteractionRequest{Start: []gnap.InteractionStartMode{gnap.InteractionStartRedirect}},
	}
	grant, _ := store.Create(&req)
	_ = grant.Transition(gnap.GrantStatePending)
	grant.InteractRef = "test-ref"
	_ = store.Update(grant)

	// Now continue with interaction reference
	contBody, _ := json.Marshal(map[string]string{"interact_ref": "test-ref"})
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("POST", "/gnap/continue/"+grant.ID, bytes.NewReader(contBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "GNAP "+grant.ContinueToken)

	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp gnap.GrantResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.AccessToken == nil {
		t.Error("Expected access token after successful continuation")
	}
}

func TestContinueCancel(t *testing.T) {
	store := gnap.NewMemoryGrantStore()
	handler := NewHandler(store, nil, "http://localhost:8080")

	r := gin.New()
	handler.RegisterRoutes(r)

	// Create a grant
	req := gnap.GrantRequest{AccessToken: &gnap.AccessTokenRequest{}}
	grant, _ := store.Create(&req)

	// Cancel it
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("DELETE", "/gnap/continue/"+grant.ID, nil)

	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", w.Code)
	}

	// Verify grant is denied
	updated, _ := store.Get(grant.ID)
	if updated.State != gnap.GrantStateDenied {
		t.Errorf("Expected state Denied, got %s", updated.State)
	}
}
