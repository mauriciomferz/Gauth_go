package modellimits

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestMultiPeriodLimits validates enforcement of multiple rate limit windows.
func TestMultiPeriodLimits(t *testing.T) {
	limitsFile, err := os.CreateTemp(t.TempDir(), "limits_multi_*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	// Model A: 5/minute AND 10/hour
	// Model B: 2/minute
	jsonContent := `{"model_limits":{"modelA":{"rate_limits_extended":["5/minute", "10/hour"]}, "modelB":{"max_requests_per_minute":2}}}`
	_, _ = limitsFile.WriteString(jsonContent)
	limitsFile.Close()

	h := NewHandler(limitsFile.Name(), "", "")
	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	// 1. Model A: Consume 5/minute limit
	for i := 0; i < 5; i++ {
		res := h.CheckLimit("modelA", "", 1, 1)
		if !res.Allowed {
			t.Fatalf("modelA req %d expected allowed, got denied: %v", i, res.Error)
		}
	}
	// 6th request blocked by minute limit
	res := h.CheckLimit("modelA", "", 1, 1)
	if res.Allowed {
		t.Fatalf("modelA expected minute limit exceed")
	}

	// Reset minute window manually to test hour limit
	h.mu.Lock()
	if st, ok := h.rateStateExtended["modelA"]; ok {
		if minSt, ok := st[time.Minute]; ok {
			minSt.WindowStart = time.Now().Add(-2 * time.Minute).Unix()
			minSt.Count = 0
		}
	}
	h.mu.Unlock()

	// Consume remaining 5 to reach 10/hour
	for i := 0; i < 5; i++ {
		res := h.CheckLimit("modelA", "", 1, 1)
		if !res.Allowed {
			t.Fatalf("modelA hour leg req %d expected allowed, got denied: %v", i, res.Error)
		}
	}
	// 11th request blocked by hour limit
	res = h.CheckLimit("modelA", "", 1, 1)
	if res.Allowed {
		t.Fatalf("modelA expected hour limit exceed")
	}

	// 2. Model B: standard rate limit
	for i := 0; i < 2; i++ {
		res := h.CheckLimit("modelB", "", 1, 1)
		if !res.Allowed {
			t.Fatalf("modelB req %d expected allowed", i)
		}
	}
	res = h.CheckLimit("modelB", "", 1, 1)
	if res.Allowed {
		t.Fatalf("modelB expected rate limit exceed")
	}
}
