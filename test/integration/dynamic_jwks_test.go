package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mauriciomferz/AgentAuth/pkg/auth"
	"github.com/mauriciomferz/AgentAuth/web/handlers/grant_jwt"
)

func TestDynamicJWKS_Integration(t *testing.T) {
	// 1. Generate RSA Key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey

	// 2. Setup Mock JWKS Server
	jwk := jose.JSONWebKey{
		Key:       publicKey,
		KeyID:     "test-key-1",
		Algorithm: "RS256",
		Use:       "sig",
	}
	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{jwk},
	}

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if encodeErr := json.NewEncoder(w).Encode(jwks); encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer jwksServer.Close()

	// 3. Register Trusted Issuer
	issuerURL := "https://mock-issuer.example.com"
	issuer := &auth.TrustedIssuer{
		Issuer:  issuerURL,
		JWKSURI: jwksServer.URL,
		// No audience required for this test, but can add
		CacheTTL: 1 * time.Second, // Short TTL for repeated testing possibilities
	}
	// Use a fresh registry for test isolation (if Handler supports injection)
	testRegistry := auth.NewIssuerRegistry()
	testRegistry.Register(issuer)

	// 4. Setup Grant Handler
	// Mock Authenticator (not used for this flow, but required for struct)
	mockAuth := &mockClientAuthenticator{}
	handler := grant_jwt.NewHandler(mockAuth)
	handler.SetIssuerRegistry(testRegistry)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router)

	// 5. Generate Assertion Signed by Generated Key
	claims := jwt.MapClaims{
		"iss": issuerURL,
		"sub": "test-user",
		"aud": "agentauth-gateway",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-1"

	assertion, err := token.SignedString(privateKey)
	require.NoError(t, err)

	// 6. Execute Request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/oauth/token", nil)
	// Form Data
	form := url.Values{}
	form.Add("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Add("assertion", assertion)
	req.PostForm = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	router.ServeHTTP(w, req)

	// 7. Verify Result
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["access_token"])
}

// Mock Authenticator
type mockClientAuthenticator struct{}

func (m *mockClientAuthenticator) Authenticate(clientID, assertion string, assertionType string) error {
	return nil
}
