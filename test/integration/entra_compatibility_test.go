package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/AgentAuth/pkg/auth"
	"github.com/mauriciomferz/AgentAuth/web"
	"github.com/mauriciomferz/AgentAuth/web/handlers/grant_jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntraCompatibilityFlow(t *testing.T) {
	// 1. Setup Mock JWKS Server
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwksHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Minimal JWKS response
		pub := privateKey.PublicKey
		n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
		eBytes := big.NewInt(int64(pub.E)).Bytes()
		e := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"kid": "test-key-1",
					"use": "sig",
					"alg": "RS256",
					"n":   n,
					"e":   e,
				},
			},
		}
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	jwksServer := httptest.NewServer(jwksHandler)
	defer jwksServer.Close()

	// 2. Setup Server
	server := web.NewBetaServer(":8090") // 8090 for Entra tests
	go func() { _ = server.Run() }()
	defer server.Shutdown()

	time.Sleep(200 * time.Millisecond) // Wait for start
	baseURL := "http://localhost:8090"

	// 3. Configure Trusted Issuer with Mock JWKS
	trustedIssuer := &auth.TrustedIssuer{
		Issuer:   "https://sts.windows.net/mock-tenant-id/",
		Audience: "agentauth-service-id",
		JWKSURI:  jwksServer.URL,
		ClaimsMapping: map[string]string{
			"oid":   "sub",          // Map Entra Object ID to Subject
			"roles": "realm_access", // Map roles to realm_access
		},
	}
	auth.GlobalRegistry.Register(trustedIssuer)

	t.Run("Reject_Invalid_Audience", func(t *testing.T) {
		// Valid signature (RS256), but wrong audience
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss": "https://sts.windows.net/mock-tenant-id/",
			"aud": "wrong-audience",
			"sub": "user-a",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		token.Header["kid"] = "test-key-1"

		tokenString, err := token.SignedString(privateKey)
		require.NoError(t, err)

		vals := url.Values{}
		vals.Set("grant_type", grant_jwt.GrantTypeIdentityAssertion)
		vals.Set("assertion", tokenString)

		resp, err := http.PostForm(baseURL+"/oauth/token", vals)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

		desc, _ := body["error_description"].(string)
		assert.Contains(t, desc, "audience mismatch")
	})
}
