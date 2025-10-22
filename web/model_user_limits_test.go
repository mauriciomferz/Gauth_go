package web

import (
	"bytes"
	"os"
	"testing"
	"time"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestModelUserLimits validates per-user quota overrides for input/output/rate dimensions.
func TestModelUserLimits(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "model_user_limits_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":200,"max_output_tokens":150,"max_requests_per_minute":10}},"user_limits":{"demo-model":{"alice":{"max_input_tokens":100,"max_output_tokens":80,"max_requests_per_minute":2},"bob":{"max_input_tokens":50}}}}`
	if _, err := f.Write([]byte(limitsJSON)); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", f.Name())
	bs := NewBetaServer("")
	// 1. Alice within all per-user limits
	resp := doModelReq(bs, map[string]any{"model_id": "demo-model", "user_id": "alice", "input_tokens": 50, "output_tokens": 70})
	if resp.Code != 200 {
		t.Fatalf("alice within limits expected 200 got %d body=%s", resp.Code, resp.Body.String())
	}
	// 2. Alice input exceed (100 limit)
	resp = doModelReq(bs, map[string]any{"model_id": "demo-model", "user_id": "alice", "input_tokens": 120, "output_tokens": 10})
	if resp.Code != 400 || !bytes.Contains(resp.Body.Bytes(), []byte("model_user_input_limit_exceeded")) {
		t.Fatalf("expected alice input exceed 400 got %d body=%s", resp.Code, resp.Body.String())
	}
	// 3. Alice output exceed (80 limit)
	resp = doModelReq(bs, map[string]any{"model_id": "demo-model", "user_id": "alice", "input_tokens": 10, "output_tokens": 120})
	if resp.Code != 400 || !bytes.Contains(resp.Body.Bytes(), []byte("model_user_output_limit_exceeded")) {
		t.Fatalf("expected alice output exceed 400 got %d body=%s", resp.Code, resp.Body.String())
	}
	// 4. Alice rate limiting (limit=2). Perform 3 requests.
	// Reset per-user rate window to avoid earlier exceed requests inflating count.
	bs.modelUserRateStateMu.Lock()
	if bs.modelUserRateState != nil && bs.modelUserRateState["demo-model"] != nil {
		st := bs.modelUserRateState["demo-model"]["alice"]
		st.WindowStart = time.Now()
		st.Count = 0
		bs.modelUserRateState["demo-model"]["alice"] = st
	}
	bs.modelUserRateStateMu.Unlock()
	for i := 0; i < 3; i++ {
		resp = doModelReq(bs, map[string]any{"model_id": "demo-model", "user_id": "alice", "input_tokens": 10, "output_tokens": 10})
		if i < 2 && resp.Code != 200 {
			t.Fatalf("alice rate allow #%d failed code=%d body=%s", i, resp.Code, resp.Body.String())
		}
		if i == 2 && resp.Code != 429 {
			t.Fatalf("alice expected rate limited on 3rd request code=%d body=%s", resp.Code, resp.Body.String())
		}
	}
	// 5. Bob input exceed (50 limit)
	resp = doModelReq(bs, map[string]any{"model_id": "demo-model", "user_id": "bob", "input_tokens": 60, "output_tokens": 10})
	if resp.Code != 400 || !bytes.Contains(resp.Body.Bytes(), []byte("model_user_input_limit_exceeded")) {
		t.Fatalf("expected bob input exceed 400 got %d body=%s", resp.Code, resp.Body.String())
	}
	// 6. Charlie (no per-user entry) uses global model limit (200). Exceed triggers global error not user-specific.
	resp = doModelReq(bs, map[string]any{"model_id": "demo-model", "user_id": "charlie", "input_tokens": 250, "output_tokens": 10})
	if resp.Code != 400 || !bytes.Contains(resp.Body.Bytes(), []byte("model_limit_exceeded")) {
		t.Fatalf("expected global input exceed for charlie 400 got %d body=%s", resp.Code, resp.Body.String())
	}
	// Metrics assertions (memory implementation)
	if mem, ok := bs.metrics.(*imetrics.Memory); ok {
		if mem.ModelUserInputLimitExceeded() < 2 { // alice + bob input exceed
			t.Fatalf("expected >=2 user input exceed counter got %d", mem.ModelUserInputLimitExceeded())
		}
		if mem.ModelUserOutputLimitExceeded() < 1 {
			t.Fatalf("expected user output exceed counter >=1 got %d", mem.ModelUserOutputLimitExceeded())
		}
		if mem.ModelUserRateLimitExceeded() < 1 {
			t.Fatalf("expected user rate exceed counter >=1 got %d", mem.ModelUserRateLimitExceeded())
		}
	}
}
