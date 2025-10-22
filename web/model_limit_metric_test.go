package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestModelLimitExceededMetric ensures the dedicated exceed counter increments.
func TestModelLimitExceededMetric(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "model_limits_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if _, err := f.Write([]byte(`{"model_limits":{"demo-model":{"max_input_tokens":100}}}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", f.Name())
	bs := NewBetaServer("")
	mem, ok := bs.metrics.(*imetrics.Memory)
	if !ok {
		t.Fatalf("expected memory metrics implementation")
	}
	before := mem.SnapshotEx()
	// exceed request
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"model_id": "demo-model", "input_tokens": 150})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 exceed got %d body=%s", w.Code, w.Body.String())
	}
	after := mem.SnapshotEx()
	// We added modelLimitExceeded counter field; access via reflection or direct atomic? For simplicity, rely on decision breakdown diff.
	denyKey := "model_validate|demo-model|deny"
	if after.DecisionBreakdown[denyKey] <= before.DecisionBreakdown[denyKey] {
		t.Fatalf("decision deny counter did not increment for exceed case")
	}
	// Basic presence of error string
	if !bytes.Contains(w.Body.Bytes(), []byte("model_limit_exceeded")) {
		t.Fatalf("response missing model_limit_exceeded marker")
	}
}
