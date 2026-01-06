package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/apikey"
	"github.com/mauriciomferz/AgentAuth/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyEndpoints_NoDB(t *testing.T) {
	// Ensure DB is disabled to test graceful degradation
	os.Setenv("AGENTAUTH_DB_ENABLED", "0")
	defer os.Unsetenv("AGENTAUTH_DB_ENABLED")

	server := web.NewBetaServerWithMetrics(":0", nil)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// Try to access API keys endpoint
	resp, err := http.Get(ts.URL + "/api/v1/apikeys")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 404 because handler is not initialized without DB
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAPIKeyEndpoints_WithDB(t *testing.T) {
	// This test requires a running Postgres instance.
	// It is skipped if AGENTAUTH_INTEGRATION_DB_CONN is not set.
	dbConn := os.Getenv("AGENTAUTH_INTEGRATION_DB_CONN")
	if dbConn == "" {
		t.Skip("Skipping API Key integration test: AGENTAUTH_INTEGRATION_DB_CONN not set")
	}

	// Setup environment
	os.Setenv("AGENTAUTH_DB_ENABLED", "1")
	os.Setenv("AGENTAUTH_DB_MIGRATE", "1")
	// Parse connection string to set individual env vars required by factory if needed,
	// or assume factory uses these if set.
	// Factory uses AGENTAUTH_DB_HOST etc.
	// For simplicity, we assume environment is already set up for this test execution environment
	// if DB_CONN is provided, or we parse it here.
	// But let's just use the skip logic.

	// Create server
	server := web.NewBetaServerWithMetrics(":0", nil)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	client := ts.Client()

	// 1. Create API Key
	reqBody := apikey.CreateAPIKeyRequest{
		Name:        "Test Key",
		Description: "Integration Test Key",
		IPWhitelist: []string{"127.0.0.1"},
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/apikeys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Implicit admin/user ID handling in handler

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create API key: status %d", resp.StatusCode)
	}

	var createdKey apikey.APIKeyWithSecret
	json.NewDecoder(resp.Body).Decode(&createdKey)
	assert.NotEmpty(t, createdKey.ID)
	assert.NotEmpty(t, createdKey.SecretKey)
	assert.Equal(t, "Test Key", createdKey.Name)

	// 2. List API Keys
	resp, err = client.Get(ts.URL + "/api/v1/apikeys")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 3. Get API Key
	resp, err = client.Get(ts.URL + "/api/v1/apikeys/" + createdKey.KeyID) // Handler expects ID or KeyID?
	// Handler Get uses `c.Param("id")` passed to `manager.GetAPIKey(ctx, id)`.
	// Manager GetAPIKey SQL uses `WHERE key_id = $1`.
	// Use KeyID.
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 4. Update API Key
	updateBody := apikey.UpdateAPIKeyRequest{
		Name: strPtr("Updated Test Key"),
	}
	uBody, _ := json.Marshal(updateBody)
	req, _ = http.NewRequest("PUT", ts.URL+"/api/v1/apikeys/"+createdKey.KeyID, bytes.NewReader(uBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 5. Delete API Key
	req, _ = http.NewRequest("DELETE", ts.URL+"/api/v1/apikeys/"+createdKey.KeyID, nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func strPtr(s string) *string {
	return &s
}
