package crypto

import (
	"testing"
	"time"
)

func TestRecoveryAutomation_HealthCheck(t *testing.T) {
	m, err := NewManager(1 * time.Second)
	if err != nil {
		t.Fatalf("manager err: %v", err)
	}
	// Simulate health check
	if m.Active() == nil {
		t.Fatalf("active key missing for health check")
	}
	// Simulate periodic health check
	for i := 0; i < 3; i++ {
		time.Sleep(400 * time.Millisecond)
		if m.Active() == nil {
			t.Fatalf("active key missing at health check iteration %d", i)
		}
	}
	m.Stop()
}
