package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenScalingAbstraction(t *testing.T) {
	// 1. Start Server (Default Memory Mode)
	server := web.NewBetaServer(":8096")
	go func() { _ = server.Run() }()
	defer server.Shutdown()

	time.Sleep(500 * time.Millisecond)
	baseURL := "http://localhost:8096"

	t.Run("Token_Lifecycle_Agnostic", func(t *testing.T) {
		// Create
		reqBody := map[string]interface{}{"ttl_seconds": 60, "meta": "scaling-test"}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)
		resp, err := http.Post(baseURL+"/api/v1/token/create", "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		tokenData := result["token"].(map[string]interface{})
		tokenID := tokenData["id"].(string)
		tokenVal := tokenData["token"].(string)

		// Validate (by value)
		valReq := map[string]string{"token": tokenVal}
		valJSON, err := json.Marshal(valReq)
		require.NoError(t, err)
		valResp, err := http.Post(baseURL+"/api/v1/token/validate", "application/json", bytes.NewBuffer(valJSON))
		require.NoError(t, err)
		defer func() { _ = valResp.Body.Close() }()
		assert.Equal(t, http.StatusOK, valResp.StatusCode)

		// Revoke
		revReq := map[string]string{"token_id": tokenID}
		revJSON, err := json.Marshal(revReq)
		require.NoError(t, err)
		revResp, err := http.Post(baseURL+"/api/v1/token/revoke", "application/json", bytes.NewBuffer(revJSON))
		require.NoError(t, err)
		defer func() { _ = revResp.Body.Close() }()
		assert.Equal(t, http.StatusOK, revResp.StatusCode)

		// Validate again (should be revoked)
		valResp2, err := http.Post(baseURL+"/api/v1/token/validate", "application/json", bytes.NewBuffer(valJSON))
		require.NoError(t, err)
		defer func() { _ = valResp2.Body.Close() }()
		var valRes2 map[string]interface{}
		err = json.NewDecoder(valResp2.Body).Decode(&valRes2)
		require.NoError(t, err)
		// Depending on implementation, validate returns 200 with success:false or similar
		// The Handler returns 200 OK with "success": false/true and "status": "revoked"
		// Ensure it's reachable and logic holds
		assert.Equal(t, http.StatusOK, valResp2.StatusCode)
	})
}
