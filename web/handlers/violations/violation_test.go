package violations

import (
	"path/filepath"
	"testing"
	"time"
)

// MockService implements ViolationProvider for testing.
type MockService struct {
	snapshot map[string]uint64
}

func (m *MockService) ViolationSnapshot() map[string]uint64 {
	return m.snapshot
}

func (m *MockService) RestoreViolations(counters map[string]uint64) {
	m.snapshot = counters
}

func TestViolationRateComputation(t *testing.T) {
	mock := &MockService{snapshot: make(map[string]uint64)}
	h := NewHandler(mock, nil, "")

	base := time.Now().Add(-10 * time.Minute)
	// Add baseline
	mock.snapshot["ratelimit"] = 100
	h.AddSnapshot(base, mock.snapshot)

	// Add data points
	// T+1m: +10 violations (rate ~10/60 = 0.166)
	base = base.Add(time.Minute)
	mock.snapshot["ratelimit"] = 110
	h.AddSnapshot(base, mock.snapshot)

	rates := h.ComputeRates(60 * time.Second)
	if val, ok := rates["ratelimit"]; !ok || val < 9.0 || val > 11.0 {
		t.Errorf("expected rate ~10, got %f", val)
	}
}

func TestPersistenceSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "violations.json")

	mock := &MockService{snapshot: make(map[string]uint64)}
	h := NewHandler(mock, nil, path)

	mock.snapshot["ratelimit"] = 50
	h.Update() // Creates history entry, prevHash starts empty

	if err := h.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Create new handler to load
	mock2 := &MockService{snapshot: make(map[string]uint64)}
	h2 := NewHandler(mock2, nil, path)
	if err := h2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Verify counters restored to service
	if mock2.snapshot["ratelimit"] != 50 {
		t.Errorf("expected restored counter 50, got %d", mock2.snapshot["ratelimit"])
	}

	// Verify integrity
	status, _ := h2.VerifyPersistence()
	if status != "ok" {
		t.Errorf("expected integrity ok, got %s", status)
	}
}
