package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/AgentAuth/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityProvisioningFlow(t *testing.T) {
	// 1. Setup Server
	server := web.NewBetaServer(":8092")
	go server.Run()
	defer server.Shutdown()

	time.Sleep(200 * time.Millisecond)
	baseURL := "http://localhost:8092"

	t.Run("Provision_Assertion_Success", func(t *testing.T) {
		reqBody := map[string]string{
			"agent_id":        "agent-123",
			"target_audience": "https://graph.microsoft.com",
		}
		jsonBody, _ := json.Marshal(reqBody)

		// 2. Call Provisioning Endpoint
		resp, err := http.Post(baseURL+"/api/v1/identity/provision", "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var respData struct {
			Assertion string `json:"assertion"`
			ExpiresIn int    `json:"expires_in"`
		}
		err = json.NewDecoder(resp.Body).Decode(&respData)
		require.NoError(t, err)

		// 3. Verify Assertion
		assert.NotEmpty(t, respData.Assertion)
		assert.Equal(t, 3600, respData.ExpiresIn)

		// Parse JWT (without verifying signature since we don't have the public key easily here)
		token, _, err := new(jwt.Parser).ParseUnverified(respData.Assertion, jwt.MapClaims{})
		require.NoError(t, err)

		claims, ok := token.Claims.(jwt.MapClaims)
		require.True(t, ok)

		// Check Standard Claims
		assert.Equal(t, "agent-123", claims["sub"])
		assert.Equal(t, "https://graph.microsoft.com", claims["aud"])

		// Issuer typically defaults to "agentauth-issuer" or env var
		// We can't strictly assert exact string unless we know env, but it should be present
		assert.NotEmpty(t, claims["iss"])

		// Check Compliance Claim
		compliance, ok := claims["x_gauth_compliance"].(map[string]interface{})
		require.True(t, ok, "x_gauth_compliance claim should be a map")
		assert.Equal(t, true, compliance["verified"])
		assert.Equal(t, "aap-001", compliance["level"])
	})
}
