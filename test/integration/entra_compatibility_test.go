package integration

import (
	"encoding/json"
	"net/http"
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
	// 1. Setup Server
	server := web.NewBetaServer(":8090") // 8090 for Entra tests
	go server.Run()
	defer server.Shutdown()

	time.Sleep(200 * time.Millisecond) // Wait for start
	baseURL := "http://localhost:8090"

	// 2. Configure Trusted Issuer
	// In a real scenario we'd query JWKS, but here we still rely on the internal authenticator
	// knowing the key for "mock-entra".
	// The TrustedIssuer config primarily enables Aud validation and Claim mapping here.
	trustedIssuer := &auth.TrustedIssuer{
		Issuer:   "https://sts.windows.net/mock-tenant-id/",
		Audience: "agentauth-service-id",
		ClaimsMapping: map[string]string{
			"oid":   "sub",          // Map Entra Object ID to Subject
			"roles": "realm_access", // Map roles to realm_access
		},
	}
	auth.GlobalRegistry.Register(trustedIssuer)

	// 3. Register Client Key (Simulate internal trust of the issuer's key for now)
	// NOTE: In a full dynamic JWKS implementation, we wouldn't need to manually register the client
	// in the memory store, but the handler implementation currently reuses h.authenticator.
	// We need to access the authenticator to register a key.
	// Since we can't easily access the internal authenticator from here without exposing it,
	// we will rely on key registration IF the test suite allows it.
	//
	// Workaround: We will use the existing "TestIdentityAssertionFlow_OBO" logic
	// but since we can't inject keys easily into the private server methods,
	// we might default to expecting a 401 Unauthorized if the key isn't known,
	// OR we assume the server has some default dev keys.
	//
	// However, to properly test this end-to-end, we need to bypass or mock the signature check,
	// or register the key.
	//
	// Let's rely on the fact that if we start the server, we might not have access to inject keys.
	// BUT, if we can't inject keys, we can't sign a valid assertion.
	// So this integration test focuses on the *rejection* of invalid Audience if trusted issuer is enforced.

	t.Run("Reject_Invalid_Audience", func(t *testing.T) {
		// Forge a token with wrong audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iss": "https://sts.windows.net/mock-tenant-id/",
			"aud": "wrong-audience",
			"sub": "user-a",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		// We sign with garbage because we expect rejection based on Audience BEFORE signature check?
		// Actually handler code checks Aud AFTER trusted issuer lookup but BEFORE signature verify?
		// Checking handler code...
		// It checks Aud at step 2.
		// It authenticates at step 3.
		// So if we fail Aud check, we get 401.

		tokenString, _ := token.SignedString([]byte("secret"))

		vals := url.Values{}
		vals.Set("grant_type", grant_jwt.GrantTypeIdentityAssertion)
		vals.Set("assertion", tokenString)

		resp, err := http.PostForm(baseURL+"/oauth/token", vals)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Contains(t, body["error_description"], "audience mismatch")
	})
}
