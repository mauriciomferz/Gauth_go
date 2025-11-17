package web

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMultiPeriodRateLimits verifies extended multi-period rate limit enforcement.
func TestMultiPeriodRateLimits(t *testing.T) {
	// Create temp limits file with multi-period configuration
	limitsFile, err := os.CreateTemp("", "limits-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(limitsFile.Name())

	// Configure: 10/minute + 50/hour
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":1000,"rate_limits_extended":["10/minute","50/hour"]}}}`
	_, _ = limitsFile.Write([]byte(limitsJSON))
	limitsFile.Close()

	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	defer os.Unsetenv("GAUTH_MODEL_LIMITS_PATH")

	s := NewBetaServer("0")
	t.Cleanup(func() { s.Shutdown() })
	s.loadModelLimitsFromDisk()

	// Send 11 requests within 1 minute - should hit per-minute limit
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest("POST", "/api/v1/model/validate", strings.NewReader(`{"model_id":"demo-model","input_tokens":10,"output_tokens":5}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		if i < 10 {
			// First 10 requests should succeed
			if w.Code != 200 {
				t.Fatalf("request %d failed with status %d, expected 200", i+1, w.Code)
			}
		} else {
			// 11th request should be rate limited (per-minute exceeded)
			if w.Code != 429 {
				t.Fatalf("request %d got status %d, expected 429 (rate limit)", i+1, w.Code)
			}
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
			if resp["error"] != "model_rate_limit_exceeded" {
				t.Errorf("expected error 'model_rate_limit_exceeded', got %v", resp["error"])
			}
			if resp["limit"] != float64(10) {
				t.Errorf("expected limit 10, got %v", resp["limit"])
			}
			if resp["period"] != "minute" {
				t.Errorf("expected period 'minute', got %v", resp["period"])
			}
		}
	}
}

// TestMultiPeriodHourlyLimit verifies hourly limit enforcement with fast clock simulation.
func TestMultiPeriodHourlyLimit(t *testing.T) {
	limitsFile, err := os.CreateTemp("", "limits-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(limitsFile.Name())

	// Configure: 100/minute + 15/hour (hourly limit more restrictive for testing)
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":1000,"rate_limits_extended":["100/minute","15/hour"]}}}`
	_, _ = limitsFile.Write([]byte(limitsJSON))
	limitsFile.Close()

	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	defer os.Unsetenv("GAUTH_MODEL_LIMITS_PATH")

	s := NewBetaServer("0")
	t.Cleanup(func() { s.Shutdown() })
	s.loadModelLimitsFromDisk()

	// Send 16 requests - should hit hourly limit before minute limit
	for i := 0; i < 16; i++ {
		req := httptest.NewRequest("POST", "/api/v1/model/validate", strings.NewReader(`{"model_id":"demo-model","input_tokens":10,"output_tokens":5}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		if i < 15 {
			if w.Code != 200 {
				t.Fatalf("request %d failed with status %d, expected 200", i+1, w.Code)
			}
		} else {
			// 16th request should be rate limited (hourly exceeded)
			if w.Code != 429 {
				t.Fatalf("request %d got status %d, expected 429 (rate limit)", i+1, w.Code)
			}
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
			if resp["period"] != "hour" {
				t.Errorf("expected period 'hour', got %v", resp["period"])
			}
			if resp["limit"] != float64(15) {
				t.Errorf("expected limit 15, got %v", resp["limit"])
			}
		}
	}
}

// TestMultiPeriodDailyLimit verifies daily limit parsing and storage.
func TestMultiPeriodDailyLimit(t *testing.T) {
	limitsFile, err := os.CreateTemp("", "limits-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(limitsFile.Name())

	// Configure: 1000/minute + 50K/day
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":1000,"rate_limits_extended":["1000/minute","50K/day"]}}}`
	_, _ = limitsFile.Write([]byte(limitsJSON))
	limitsFile.Close()

	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	defer os.Unsetenv("GAUTH_MODEL_LIMITS_PATH")

	s := NewBetaServer("0")
	t.Cleanup(func() { s.Shutdown() })
	s.loadModelLimitsFromDisk()

	// Verify extended limits parsed correctly
	s.modelRateLimitsExtendedMu.Lock()
	extLimits := s.modelRateLimitsExtended["demo-model"]
	s.modelRateLimitsExtendedMu.Unlock()

	if len(extLimits) != 2 {
		t.Fatalf("expected 2 extended limits, got %d", len(extLimits))
	}

	// Verify minute limit
	if extLimits[0].Limit != 1000 || extLimits[0].Period != time.Minute {
		t.Errorf("expected {1000, 1m}, got {%d, %v}", extLimits[0].Limit, extLimits[0].Period)
	}

	// Verify daily limit (50K)
	if extLimits[1].Limit != 50000 || extLimits[1].Period != 24*time.Hour {
		t.Errorf("expected {50000, 24h}, got {%d, %v}", extLimits[1].Limit, extLimits[1].Period)
	}
}

// TestBackwardCompatibilityPerMinute verifies legacy max_requests_per_minute still works.
func TestBackwardCompatibilityPerMinute(t *testing.T) {
	limitsFile, err := os.CreateTemp("", "limits-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(limitsFile.Name())

	// Use only legacy max_requests_per_minute (no rate_limits_extended)
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":1000,"max_requests_per_minute":5}}}`
	_, _ = limitsFile.Write([]byte(limitsJSON))
	limitsFile.Close()

	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	defer os.Unsetenv("GAUTH_MODEL_LIMITS_PATH")

	s := NewBetaServer("0")
	t.Cleanup(func() { s.Shutdown() })
	s.loadModelLimitsFromDisk()

	// Send 6 requests - should hit legacy limit
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/api/v1/model/validate", strings.NewReader(`{"model_id":"demo-model","input_tokens":10,"output_tokens":5}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		if i < 5 {
			if w.Code != 200 {
				t.Fatalf("request %d failed with status %d, expected 200", i+1, w.Code)
			}
		} else {
			// 6th request should be rate limited
			if w.Code != 429 {
				t.Fatalf("request %d got status %d, expected 429", i+1, w.Code)
			}
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
			if resp["window_seconds"] != float64(60) {
				t.Errorf("expected window_seconds 60, got %v", resp["window_seconds"])
			}
		}
	}
}

// TestBothLegacyAndExtended verifies both legacy + extended limits are enforced.
func TestBothLegacyAndExtended(t *testing.T) {
	limitsFile, err := os.CreateTemp("", "limits-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(limitsFile.Name())

	// Configure both: max_requests_per_minute=5 AND rate_limits_extended=["20/hour"]
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":1000,"max_requests_per_minute":5,"rate_limits_extended":["20/hour"]}}}`
	_, _ = limitsFile.Write([]byte(limitsJSON))
	limitsFile.Close()

	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	defer os.Unsetenv("GAUTH_MODEL_LIMITS_PATH")

	s := NewBetaServer("0")
	t.Cleanup(func() { s.Shutdown() })
	s.loadModelLimitsFromDisk()

	// Send 6 requests - should hit legacy per-minute limit first
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/api/v1/model/validate", strings.NewReader(`{"model_id":"demo-model","input_tokens":10,"output_tokens":5}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		if i < 5 {
			if w.Code != 200 {
				t.Fatalf("request %d failed with status %d, expected 200", i+1, w.Code)
			}
		} else {
			// 6th request should be rate limited (legacy per-minute)
			if w.Code != 429 {
				t.Fatalf("request %d got status %d, expected 429", i+1, w.Code)
			}
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
			// Should be legacy limit (no "period" field)
			if resp["period"] != nil {
				t.Errorf("expected legacy rate limit (no period), got period=%v", resp["period"])
			}
		}
	}
}

// TestPeriodRollover verifies period window reset after expiration.
func TestPeriodRollover(t *testing.T) {
	limitsFile, err := os.CreateTemp("", "limits-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(limitsFile.Name())

	// Configure: 3/minute (short limit for testing rollover)
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":1000,"rate_limits_extended":["3/minute"]}}}`
	_, _ = limitsFile.Write([]byte(limitsJSON))
	limitsFile.Close()

	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	defer os.Unsetenv("GAUTH_MODEL_LIMITS_PATH")

	s := NewBetaServer("0")
	t.Cleanup(func() { s.Shutdown() })
	s.loadModelLimitsFromDisk()

	// Send 3 requests - should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/model/validate", strings.NewReader(`{"model_id":"demo-model","input_tokens":10,"output_tokens":5}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("request %d failed with status %d", i+1, w.Code)
		}
	}

	// 4th request should fail
	req := httptest.NewRequest("POST", "/api/v1/model/validate", strings.NewReader(`{"model_id":"demo-model","input_tokens":10,"output_tokens":5}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Fatalf("4th request got status %d, expected 429", w.Code)
	}

	// Manually reset window (simulate time passing)
	s.modelRateStateExtendedMu.Lock()
	if s.modelRateStateExtended["demo-model"] != nil {
		periodState := s.modelRateStateExtended["demo-model"][time.Minute]
		periodState.WindowStart = time.Now().Add(-2 * time.Minute) // force expired
		s.modelRateStateExtended["demo-model"][time.Minute] = periodState
	}
	s.modelRateStateExtendedMu.Unlock()

	// Next request should succeed (window rolled over)
	req = httptest.NewRequest("POST", "/api/v1/model/validate", strings.NewReader(`{"model_id":"demo-model","input_tokens":10,"output_tokens":5}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("post-rollover request got status %d, expected 200", w.Code)
	}
}

// TestInvalidRateLimitFormat verifies graceful handling of malformed rate limits.
func TestInvalidRateLimitFormat(t *testing.T) {
	limitsFile, err := os.CreateTemp("", "limits-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(limitsFile.Name())

	// Include invalid rate limit format (should be skipped)
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":1000,"rate_limits_extended":["invalid","10/minute","bad/format"]}}}`
	_, _ = limitsFile.Write([]byte(limitsJSON))
	limitsFile.Close()

	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	defer os.Unsetenv("GAUTH_MODEL_LIMITS_PATH")

	s := NewBetaServer("0")
	t.Cleanup(func() { s.Shutdown() })

	// Capture stderr to verify error logging
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	s.loadModelLimitsFromDisk()

	w.Close()
	os.Stderr = oldStderr

	stderrOutput, _ := io.ReadAll(r)
	stderrStr := string(stderrOutput)

	// Verify error messages logged for invalid formats
	if !strings.Contains(stderrStr, "invalid rate limit") {
		t.Errorf("expected error log for invalid rate limit, got: %s", stderrStr)
	}

	// Verify only valid limit was parsed
	s.modelRateLimitsExtendedMu.Lock()
	extLimits := s.modelRateLimitsExtended["demo-model"]
	s.modelRateLimitsExtendedMu.Unlock()

	if len(extLimits) != 1 {
		t.Fatalf("expected 1 valid extended limit, got %d", len(extLimits))
	}
	if extLimits[0].Limit != 10 || extLimits[0].Period != time.Minute {
		t.Errorf("expected {10, 1m}, got {%d, %v}", extLimits[0].Limit, extLimits[0].Period)
	}
}
