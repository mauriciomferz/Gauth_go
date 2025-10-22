package web

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// TestModelLimitsDynamicReload verifies that tightening a limit in the JSON file is applied after reload interval.
func TestModelLimitsDynamicReload(t *testing.T) {
    tmp, err := os.CreateTemp(t.TempDir(), "model_limits_reload_*.json")
    if err != nil { t.Fatalf("temp: %v", err) }
    initial := `{"model_limits":{"reload-model":{"max_input_tokens":500}}}`
    if _, err := tmp.Write([]byte(initial)); err != nil { t.Fatalf("write: %v", err) }
    tmp.Close()
    os.Setenv("GAUTH_MODEL_LIMITS_PATH", tmp.Name())
    os.Setenv("GAUTH_MODEL_LIMITS_RELOAD_INTERVAL", "1") // poll every second
    bs := NewBetaServer("")
    // Initial request within 400 tokens allowed (limit 500)
    if r := doModelReq(bs, map[string]any{"model_id":"reload-model","input_tokens":400}); r.Code != 200 {
        t.Fatalf("expected initial allow got %d body=%s", r.Code, r.Body.String())
    }
    // Overwrite file with tighter limit 300
    tightened := `{"model_limits":{"reload-model":{"max_input_tokens":300}}}`
    if err := os.WriteFile(tmp.Name(), []byte(tightened), 0600); err != nil { t.Fatalf("rewrite: %v", err) }
    // Wait up to 3 seconds for reload to apply
    deadline := time.Now().Add(3 * time.Second)
    for {
        r := doModelReq(bs, map[string]any{"model_id":"reload-model","input_tokens":400})
        if r.Code == 400 && bytes.Contains(r.Body.Bytes(), []byte("model_limit_exceeded")) {
            break // reloaded
        }
        if time.Now().After(deadline) {
            t.Fatalf("limit did not tighten after reload window code=%d body=%s", r.Code, r.Body.String())
        }
        time.Sleep(200 * time.Millisecond)
    }
    // Confirm lower allowed request passes
    if r := doModelReq(bs, map[string]any{"model_id":"reload-model","input_tokens":250}); r.Code != 200 {
        t.Fatalf("expected allow under tightened limit got %d body=%s", r.Code, r.Body.String())
    }
}
