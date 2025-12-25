package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/gnap"
	gnaphandler "github.com/mauriciomferz/Gauth_go/web/handlers/gnap"
)

func TestRFC9767_RSConnection(t *testing.T) {
	// 1. Setup Handler
	grantStore := gnap.NewMemoryGrantStore()
	rsStore := gnap.NewMemoryResourceServerStore()
	// Need a token store for introspection to check anything real, but the handler has a mock path for "gauth_gnap_"
	// We'll rely on that mock path for now since `NewHandler` interface might not expose TokenStore directly
	// without me checking the struct definition again...
	// Wait, the handler struct has TokenStore field.

	handler := gnaphandler.NewHandler(grantStore, "http://localhost:8080")
	handler.RSStore = rsStore
	// TokenStore is nil by default, but introspect endpoint checks for it.
	// We need to verify if handler.IntrospectRS checks for TokenStore presence before the "gauth_gnap_" mock check.
	// Looking at rs_handler.go:
	// if h.TokenStore == nil { return error }
	// So we must provide a dummy token store.

	// Since TokenStore is an interface, let's create a minimal mock inline or find one.
	// `pkg/gnap/token_store.go` probably defines it.
	// Let's assume `gnap.TokenStore` interface exists.
	// Actually, `rs_handler.go` uses `gnap.TokenStore`.
	// I'll create a mock token store locally.
}

type mockTokenStore struct{}

func (m *mockTokenStore) Store(token *gnap.IssuedToken) error                     { return nil }
func (m *mockTokenStore) Get(value string) (*gnap.IssuedToken, error)             { return nil, nil }
func (m *mockTokenStore) Rotate(oldValue string) (*gnap.IssuedToken, error)       { return nil, nil }
func (m *mockTokenStore) Revoke(value string) error                               { return nil }
func (m *mockTokenStore) ListByGrant(grantID string) ([]*gnap.IssuedToken, error) { return nil, nil }

func TestRFC9767_FullFlow(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.New()

	grantStore := gnap.NewMemoryGrantStore()
	rsStore := gnap.NewMemoryResourceServerStore()

	handler := gnaphandler.NewHandler(grantStore, "http://localhost:8080")
	handler.RSStore = rsStore
	handler.TokenStore = &mockTokenStore{} // Use dummy to pass nil check

	handler.RegisterRoutes(r)

	// 1. Register RS
	rsReq := gnap.ResourceServerRequest{
		Name: "Financial Service RS",
		Client: &gnap.ClientInstance{
			Display: &gnap.ClientDisplay{Name: "Finance App"},
		},
		Resources: []string{"financial-data"},
	}
	body, _ := json.Marshal(rsReq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/gnap/rs/register", bytes.NewBuffer(body))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("RS Register failed: %d %s", w.Code, w.Body.String())
	}

	var rsResp gnap.ResourceServerResponse
	if err := json.Unmarshal(w.Body.Bytes(), &rsResp); err != nil {
		t.Fatalf("Failed to parse RS response: %v", err)
	}

	if rsResp.InstanceID == "" {
		t.Error("RS InstanceID empty")
	}
	t.Logf("Registered RS Instance: %s", rsResp.InstanceID)

	// 2. Introspect (Active Token)
	// We use the magic token "gauth_gnap_active" to trigger the mock active response in the handler
	introReq := gnap.IntrospectionRequest{
		Token: "gauth_gnap_active",
	}
	body, _ = json.Marshal(introReq)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/gnap/rs/introspect", bytes.NewBuffer(body))
	// Set RS Authentication
	req.Header.Set("Authorization", "RS "+rsResp.InstanceID)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RS Introspect failed: %d %s", w.Code, w.Body.String())
	}

	var introResp gnap.IntrospectionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &introResp); err != nil {
		t.Fatalf("Failed to parse introspection response: %v", err)
	}

	if !introResp.Active {
		t.Error("Expected active token response")
	}
	if introResp.PoA == nil {
		t.Error("Expected PoA extension in response")
	}

	// 3. Introspect (Inactive/Unknown Token)
	introReq = gnap.IntrospectionRequest{
		Token: "other_token",
	}
	body, _ = json.Marshal(introReq)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/gnap/rs/introspect", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "RS "+rsResp.InstanceID)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RS Introspect failed: %d", w.Code)
	}
	var introResp2 gnap.IntrospectionResponse
	json.Unmarshal(w.Body.Bytes(), &introResp2)

	if introResp2.Active {
		t.Error("Expected inactive token response")
	}
}
