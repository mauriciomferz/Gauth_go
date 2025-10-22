package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func postModelValidate(bs *BetaServer, body any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	return w
}

// TestModelValidateLimits exercises unknown model (no limit), allowed within limit, and over-limit denial.
func TestModelValidateLimits(t *testing.T) {
	// Create temp limits file
	// Ensure strict-unknown mode is disabled for this test regardless of prior tests
	// that may have enabled GAUTH_MODEL_LIMITS_STRICT_UNKNOWN without unsetting.
	t.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "0")
	f, err := os.CreateTemp(t.TempDir(), "model_limits_*.json")
	if err != nil {
		t.Fatalf("temp file err=%v", err)
	}
	jsonData := []byte(`{"model_limits":{"demo-model":{"max_input_tokens":1024}}}`)
	if _, err := f.Write(jsonData); err != nil {
		t.Fatalf("write err=%v", err)
	}
	f.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", f.Name())
	bs := NewBetaServer("")
	// Unknown model (should allow)
	resp := postModelValidate(bs, map[string]any{"model_id": "other-model", "input_tokens": 500})
	if resp.Code != 200 {
		t.Fatalf("expected 200 unknown model got %d body=%s", resp.Code, resp.Body.String())
	}
	// Allowed within limit
	resp = postModelValidate(bs, map[string]any{"model_id": "demo-model", "input_tokens": 100})
	if resp.Code != 200 {
		t.Fatalf("expected 200 within limit body=%s", resp.Body.String())
	}
	// Over limit
	resp = postModelValidate(bs, map[string]any{"model_id": "demo-model", "input_tokens": 2000})
	if resp.Code != 400 || !bytes.Contains(resp.Body.Bytes(), []byte("model_limit_exceeded")) {
		t.Fatalf("expected 400 model_limit_exceeded got %d body=%s", resp.Code, resp.Body.String())
	}
}
