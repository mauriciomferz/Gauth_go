package modellimits

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestModelUserLimits validates per-user quota overrides for input/output/rate dimensions.
func TestModelUserLimits(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "model_user_limits_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	//nolint:lll // JSON test data
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":200,"max_output_tokens":150,"max_requests_per_minute":10}},"user_limits":{"demo-model":{"alice":{"max_input_tokens":100,"max_output_tokens":80,"max_requests_per_minute":2},"bob":{"max_input_tokens":50}}}}`
	_, _ = tmp.Write([]byte(limitsJSON))
	if err := tmp.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	h := NewHandler(tmp.Name(), "", "")
	metrics := &mockMetrics{}
	h.Metrics = metrics

	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	// 1. Alice within all per-user limits
	res := h.CheckLimit("demo-model", "alice", 50, 70)
	if !res.Allowed {
		t.Fatalf("alice within limits expected allowed, got denied: %v", res.Error)
	}

	// 2. Alice input exceed (100 limit)
	res = h.CheckLimit("demo-model", "alice", 120, 10)
	if res.Allowed {
		t.Fatalf("expected alice input exceed denied")
	}
	// Error string from Handler is "model_user_input_limit_exceeded"
	if res.Error != "model_user_input_limit_exceeded" {
		t.Fatalf("expected model_user_input_limit_exceeded, got %v", res.Error)
	}

	// 3. Alice output exceed (80 limit)
	res = h.CheckLimit("demo-model", "alice", 10, 120)
	if res.Allowed {
		t.Fatalf("expected alice output exceed denied")
	}
	if res.Error != "model_user_output_limit_exceeded" {
		t.Fatalf("expected model_user_output_limit_exceeded, got %v", res.Error)
	}

	// 4. Alice rate limiting (limit=2). Perform 3 requests.
	// Reset per-user rate window manually
	h.mu.Lock()
	if h.userRateState != nil && h.userRateState["demo-model"] != nil {
		st := h.userRateState["demo-model"]["alice"]
		if st != nil {
			st.WindowStart = time.Now().Unix()
			st.Count = 0
			h.userRateState["demo-model"]["alice"] = st
		}
	}
	h.mu.Unlock()

	for i := 0; i < 3; i++ {
		res = h.CheckLimit("demo-model", "alice", 10, 10)
		if i < 2 {
			if !res.Allowed {
				t.Fatalf("alice rate allow #%d failed, error: %v", i, res.Error)
			}
		} else {
			if res.Allowed {
				t.Fatalf("alice expected rate limited on 3rd request")
			}
			if res.Error != "model_user_rate_limit_exceeded" {
				t.Fatalf("expected model_user_rate_limit_exceeded call %d, got %v", i, res.Error)
			}
		}
	}

	// 5. Bob input exceed (50 limit)
	res = h.CheckLimit("demo-model", "bob", 60, 10)
	if res.Allowed {
		t.Fatalf("expected bob input exceed denied")
	}
	if res.Error != "model_user_input_limit_exceeded" {
		t.Fatalf("expected model_user_input_limit_exceeded, got %v", res.Error)
	}

	// 6. Charlie (no per-user entry) uses global model limit (200).
	res = h.CheckLimit("demo-model", "charlie", 250, 10)
	if res.Allowed {
		t.Fatalf("expected global input exceed for charlie denied")
	}
	if res.Error != "model_limit_exceeded" {
		t.Fatalf("expected model_limit_exceeded for charlie, got %v", res.Error)
	}

	// Metrics assertions
	if got := atomic.LoadUint64(&metrics.userInputLimitExceeded); got < 2 { // alice + bob input exceed
		t.Fatalf("expected >=2 user input exceed counter got %d", got)
	}
	if got := atomic.LoadUint64(&metrics.userOutputLimitExceeded); got < 1 {
		t.Fatalf("expected user output exceed counter >=1 got %d", got)
	}
	if got := atomic.LoadUint64(&metrics.userRateLimitExceeded); got < 1 {
		t.Fatalf("expected user rate exceed counter >=1 got %d", got)
	}
}
