package modellimits

import (
	"context"
	"os"
	"testing"
)

// TestModelExtendedLimits validates output token and rate limiting enforcement.
func TestModelExtendedLimits(t *testing.T) {
	limitsFile, err := os.CreateTemp(t.TempDir(), "model_limits_ext_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":200,"max_output_tokens":150,"max_requests_per_minute":5}}}`
	_, _ = limitsFile.WriteString(limitsJSON)
	if err := limitsFile.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	h := NewHandler(limitsFile.Name(), "", "") // No audit needed for this test
	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	// 1. Within all limits (input 50, output 75)
	res := h.CheckLimit("demo-model", "", 50, 75)
	if !res.Allowed {
		t.Fatalf("expected allowed within limits, got denied: %s", res.Error)
	}

	// 2. Output tokens exceed (output 200 > 150)
	res = h.CheckLimit("demo-model", "", 50, 200)
	if res.Allowed {
		t.Fatalf("expected output exceed denied")
	}
	if res.Error != "model_output_limit_exceeded" {
		t.Fatalf("expected model_output_limit_exceeded, got %s", res.Error)
	}

	// 3. Input tokens exceed (input 500 > 200)
	res = h.CheckLimit("demo-model", "", 500, 10)
	if res.Allowed {
		t.Fatalf("expected input exceed denied")
	}
	if res.Error != "model_limit_exceeded" {
		t.Fatalf("expected model_limit_exceeded, got %s", res.Error)
	}

	// 4. Rate limiting
	// Reconfigure internal rate limit manually to speed up test or just rely on limit=5
	// Original test set limit=3 then ran 3 times.
	// We have limit=5. Let's run 6 times.
	// We need to simulate time passing? No, same window.
	for i := 0; i < 6; i++ {
		res := h.CheckLimit("demo-model", "", 10, 10)
		if i < 4 {
			if !res.Allowed {
				t.Fatalf("req %d expected allowed, got denided: %s", i, res.Error)
			}
		} else {
			if res.Allowed {
				t.Fatalf("req %d expected rate limited, got allowed", i)
			}
			if res.Error != "model_rate_limit_exceeded" {
				t.Fatalf("req %d expected model_rate_limit_exceeded, got %s", i, res.Error)
			}
		}
	}

	// 5. Wait window reset (Simulate by re-initializing handler to clear memory state)
	// Since we can't easily mock time passing validly for the rate limiter
	// window update in this test style without re-init or expose internals
	h = NewHandler(limitsFile.Name(), "", "")
	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("re-init: %v", err)
	}

	// Test rate limiting again after reset
	for i := 0; i < 6; i++ {
		res = h.CheckLimit("demo-model", "", 10, 10)
		if i < 5 {
			if !res.Allowed {
				t.Fatalf("req %d after reset expected allowed, got denided: %s", i, res.Error)
			}
		} else {
			if res.Allowed {
				t.Fatalf("req %d after reset expected rate limited, got allowed", i)
			}
			if res.Error != "model_rate_limit_exceeded" {
				t.Fatalf("req %d after reset expected model_rate_limit_exceeded, got %s", i, res.Error)
			}
		}
	}
}
