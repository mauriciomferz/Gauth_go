package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestModelLimitsStrictUnknown verifies that unknown model is denied when strict mode is enabled.
func TestModelLimitsStrictUnknown(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "model_limits_strict_*.json")
	f.Write([]byte(`{"model_limits":{"known-model":{"max_input_tokens":100}}}`))
	f.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", f.Name())
	os.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "1")
	bs := NewBetaServer("")
	// Request for unknown model
	w := httptest.NewRecorder()
	body := map[string]any{"model_id": "unknown-model", "input_tokens": 10}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	if w.Code != 400 || !bytes.Contains(w.Body.Bytes(), []byte("model_unknown")) {
		t.Fatalf("expected strict unknown rejection 400 model_unknown got %d body=%s", w.Code, w.Body.String())
	}
	// Known model still allowed when within limit
	w2 := httptest.NewRecorder()
	body2 := map[string]any{"model_id": "known-model", "input_tokens": 50}
	b2, _ := json.Marshal(body2)
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected known model allow got %d body=%s", w2.Code, w2.Body.String())
	}
}
