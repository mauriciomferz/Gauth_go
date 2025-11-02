package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestModelLimitsSnapshotHashChange ensures snapshot hash changes after limits file reload.
func TestModelLimitsSnapshotHashChange(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "model_limits_snapshot_*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	initial := `{"model_limits":{"snap-model":{"max_input_tokens":100}}}`
	if _, err := tmp.Write([]byte(initial)); err != nil {
		t.Fatalf("write: %v", err)
	}
	tmp.Close()
	t.Setenv("GAUTH_MODEL_LIMITS_PATH", tmp.Name())
	t.Setenv("GAUTH_MODEL_LIMITS_RELOAD_INTERVAL", "1")
	bs := NewBetaServer("")

	getHash := func() string {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/snapshot", nil)
		bs.router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("snapshot status %d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Hash string `json:"hash"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
		}
		return resp.Hash
	}

	h1 := getHash()
	// Modify limits (tighten) to force different canonical representation
	updated := `{"model_limits":{"snap-model":{"max_input_tokens":80}}}`
	if err := os.WriteFile(tmp.Name(), []byte(updated), 0600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var h2 string
	for time.Now().Before(deadline) {
		h2 = getHash()
		if h2 != h1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if h1 == h2 {
		t.Fatalf("expected hash change after reload h1=%s h2=%s", h1, h2)
	}
}
