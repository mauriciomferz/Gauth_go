package gnap

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/gnap"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	store := gnap.NewMemoryGrantStore()
	rsStore := gnap.NewMemoryResourceServerStore()
	h := NewHandler(store, "http://localhost")
	h.RSStore = rsStore

	r := gin.New()
	h.RegisterRoutes(r)

	// Payload
	reqBody := gnap.ResourceServerRequest{
		Name:      "Test RS",
		Resources: []string{"account-data"},
		Client: &gnap.ClientInstance{
			Key: &gnap.ClientKey{Proof: gnap.ProofHTTPSig},
		},
	}
	body, _ := json.Marshal(reqBody)

	// Execute
	req, _ := http.NewRequest("POST", "/gnap/rs/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Verify
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp gnap.ResourceServerResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.InstanceID)
	assert.Equal(t, gnap.ProofHTTPSig, resp.Key.Proof)

	// Verify persistence in store
	saved, err := rsStore.Get(resp.InstanceID)
	assert.NoError(t, err)
	assert.Equal(t, "Test RS", saved.Name)
}

func TestIntrospectRS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	store := gnap.NewMemoryGrantStore()
	rsStore := gnap.NewMemoryResourceServerStore() // Needed for handler validity
	tokenStore := gnap.NewMemoryTokenStore()
	h := NewHandler(store, "http://localhost")
	h.RSStore = rsStore
	h.TokenStore = tokenStore

	r := gin.New()
	h.RegisterRoutes(r)

	// Test 1: No Auth Header
	reqBody := gnap.IntrospectionRequest{Token: "foo"}
	body, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/gnap/rs/introspect", bytes.NewBuffer(body))
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	// Test 2: Inactive Token (Mock logic returns active=false)
	req2, _ := http.NewRequest("POST", "/gnap/rs/introspect", bytes.NewBuffer(body))
	req2.Header.Set("Authorization", "RS my-rs-id")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var resp2 gnap.IntrospectionResponse
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	assert.False(t, resp2.Active)

	// Test 3: Active Token (Mock logic for "gauth_gnap_")
	reqBody3 := gnap.IntrospectionRequest{Token: "gauth_gnap_12345"}
	body3, _ := json.Marshal(reqBody3)
	req3, _ := http.NewRequest("POST", "/gnap/rs/introspect", bytes.NewBuffer(body3))
	req3.Header.Set("Authorization", "RS my-rs-id")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	var resp3 gnap.IntrospectionResponse
	_ = json.Unmarshal(w3.Body.Bytes(), &resp3)
	assert.True(t, resp3.Active)
	assert.NotNil(t, resp3.PoA)
	assert.Equal(t, "poa_mock_123", resp3.PoA.PoAID)
}
