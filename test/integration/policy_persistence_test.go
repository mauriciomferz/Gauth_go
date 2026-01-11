package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyPersistenceAndAPI(t *testing.T) {
	// Setup server in In-Memory mode
	t.Setenv("AGENTAUTH_DB_ENABLED", "0")
	t.Setenv("POLICY_CHAIN_STATE_PATH", "")

	m := metrics.NewMemory()
	s := web.NewBetaServerWithMetrics(":0", m)

	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	client := ts.Client()
	baseURL := ts.URL + "/api/v1/policy"

	// 1. Initial State: HeadPolicies should be empty? Or default?
	// NewInMemoryStore starts empty.
	t.Run("Initial State", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/provenance")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 200, resp.StatusCode)

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		// Should be valid but maybe empty chain
		assert.Equal(t, true, body["success"])
		assert.Equal(t, "", body["head_hash"])
	})

	// 2. Add Bundle
	var bundleHash string
	t.Run("Add Bundle", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"id": "bundle-1",
			"policies": []map[string]interface{}{
				{
					"id":       "policy-1",
					"version":  "1.0",
					"subjects": []string{"user:alice"},
					"rules": []map[string]interface{}{
						{"resources": []string{"foo"}, "actions": []string{"read"}, "effect": "allow"},
					},
				},
			},
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)
		req, err := http.NewRequest("POST", baseURL+"/bundles", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 201, resp.StatusCode)

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		if !assert.Equal(t, true, body["success"]) {
			t.Logf("Response Body: %+v", body)
		}
		if val, ok := body["bundle_hash"]; ok {
			bundleHash = val.(string)
			assert.NotEmpty(t, bundleHash)
		} else {
			t.Logf("bundle_hash missing in response: %+v", body)
			t.FailNow()
		}
		assert.Equal(t, float64(1), body["policy_version"])
	})

	// 3. Verify Active Version
	t.Run("Verify Active", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/timeline")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 200, resp.StatusCode)

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, float64(1), body["active_version"])
		assert.Equal(t, false, body["rolled_back"])
	})

	// 4. Add Second Bundle
	t.Run("Add Second Bundle", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"id": "bundle-2",
			"policies": []map[string]interface{}{
				{
					"id":       "policy-2",
					"version":  "1.0.1",
					"subjects": []string{"user:bob"},
					"rules": []map[string]interface{}{
						{"resources": []string{"bar"}, "actions": []string{"write"}, "effect": "deny"},
					},
				},
			},
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)
		req, err := http.NewRequest("POST", baseURL+"/bundles", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 201, resp.StatusCode)

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, float64(2), body["policy_version"])
	})

	// 5. Rollback
	t.Run("Rollback", func(t *testing.T) {
		req, err := http.NewRequest("POST", baseURL+"/rollback?version=1", nil)
		require.NoError(t, err)
		req.Header.Set("X-Admin-Token", "") // Assuming default is open or check env
		// Wait, API checks X-Admin-Token against AGENTAUTH_POLICY_ADMIN_TOKEN.
		// If env not set, checks if empty? No, line 408: if adminToken != "" && ...
		// By default AGENTAUTH_POLICY_ADMIN_TOKEN is empty in test? Yes.

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 200, resp.StatusCode)

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, float64(1), body["active_version"])
	})

	// 6. Verify Rollback State
	t.Run("Verify Rollback State", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/timeline")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, float64(1), body["active_version"])
		assert.Equal(t, true, body["rolled_back"])
	})
}
