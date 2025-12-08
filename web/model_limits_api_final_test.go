package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestModelLimitsAPIIntegration verifies that BetaServer correctly routes and handles
// model limits requests using the new modellimits package integration.
func TestModelLimitsAPIIntegration(t *testing.T) {
	// Setup temp limits config
	limitsFile, err := os.CreateTemp(t.TempDir(), "limits_integration_*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	_, _ = limitsFile.WriteString(`{"model_limits":{"api-test-model":{"max_input_tokens":100}}}`)
	limitsFile.Close()

	t.Setenv("GAUTH_MODEL_LIMITS_CONFIG_PATH", limitsFile.Name())
	// Set other envs to avoid side effects or enable features
	t.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "0")

	// Initialize Server
	gin.SetMode(gin.TestMode)
	bs := NewBetaServer("0")
	// Note: NewBetaServer initializes modellimits API if config present?
	// server_clean.go: `s.modelLimitsAPI = modellimits.NewAPI(...)`
	// And `RegisterRoutes`.
	// We need to ensure handler picks up the config.
	// NewBetaServer doesn't automatically pass env vars to NewHandler?
	// Wait, I updated NewHandler to read ENVs for opts, but Paths are passed in args.
	// In `NewBetaServerWithMetrics` (or NewBetaServer), how is API initialized?

	// Let's assume standard NewBetaServer wiring works if ENV vars are set for paths?
	// Actually, `NewBetaServer` in `server_clean.go` calls `modellimits.NewAPI(handler)`.
	// And `handler` is likely created with `os.Getenv(...)`?
	// Check `server_clean.go`.
	// Lines 2200ish were removed.
	// I need to check how `modelLimitsAPI` is created in `NewBetaServer`.
	// If I missed adding initialization logic back to `NewBetaServer`, this test will fail (or panic).
	// This is a CRITICAL check.
	// I'll add the test assuming it works, or I will inspect server_clean.go next if it fails.

	t.Cleanup(func() { bs.Shutdown() })

	// 1. Test /api/v1/model/limits/snapshot (Get)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/model/limits/snapshot", nil)
	bs.router.ServeHTTP(w, req)
	if w.Code != 200 {
		// If 404, routing failed. If 500, handler failed.
		t.Fatalf("snapshot endpoint failed code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "hash") {
		t.Errorf("snapshot missing hash: %s", w.Body.String())
	}

	// 2. Test /api/v1/model/validate (Post) - Allowed
	w = httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"model_id": "api-test-model", "input_tokens": 50})
	req, _ = http.NewRequest("POST", "/api/v1/model/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("validate allowed failed code=%d body=%s", w.Code, w.Body.String())
	}

	// 3. Test /api/v1/model/validate (Post) - Denied
	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]any{"model_id": "api-test-model", "input_tokens": 150})
	req, _ = http.NewRequest("POST", "/api/v1/model/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("validate denied expected 400 got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "model_limit_exceeded") {
		t.Errorf("expected error string missing: %s", w.Body.String())
	}
}
