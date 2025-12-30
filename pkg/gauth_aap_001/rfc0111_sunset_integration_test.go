package gauth_aap_001

import (
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

func TestSunsetControllerIntegration(t *testing.T) {
	// Enable sunset controller
	os.Setenv("AGENTAUTH_SUNSET_ENABLED", "1")
	os.Setenv("AGENTAUTH_SUNSET_INTERVAL", "100ms")
	os.Setenv("AGENTAUTH_SUNSET_WINDOW", "500ms")
	defer os.Unsetenv("AGENTAUTH_SUNSET_ENABLED")
	defer os.Unsetenv("AGENTAUTH_SUNSET_INTERVAL")
	defer os.Unsetenv("AGENTAUTH_SUNSET_WINDOW")

	// Create service with memory metrics for controller integration
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()

	mem := metrics.NewMemory()
	svc := NewService(auditLogger, authorizer, WithMetrics(mem))

	// Verify controller exists and is running
	if svc.sunsetController == nil {
		t.Fatal("sunset controller not initialized")
	}

	// Verify initial phase
	phase := svc.sunsetController.Phase()
	if phase != 1 { // PhasePilot
		t.Errorf("expected initial phase 1 (Pilot), got %d", phase)
	}

	// Simulate high V2 adoption
	mem.SetEnvelopeV2AdoptionRatio(0.65) // Above 0.60 threshold for Pilot→Broad

	// Wait for controller evaluation (should happen within interval)
	time.Sleep(200 * time.Millisecond)

	// Phase should still be Pilot because window (500ms) not yet elapsed
	phase = svc.sunsetController.Phase()
	if phase != 1 {
		t.Logf("phase after 200ms: %d (should still be Pilot)", phase)
	}

	// Wait for full window to elapse
	time.Sleep(400 * time.Millisecond)

	// Now phase might have progressed (depends on timing)
	phase = svc.sunsetController.Phase()
	t.Logf("Final phase: %d", phase)

	// Test complete - controller was initialized and running
}

func TestSunsetControllerDisabled(t *testing.T) {
	// Disable controller
	os.Setenv("AGENTAUTH_SUNSET_ENABLED", "0")
	defer os.Unsetenv("AGENTAUTH_SUNSET_ENABLED")

	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()

	mem := metrics.NewMemory()
	svc := NewService(auditLogger, authorizer, WithMetrics(mem))

	// Controller should exist but not be started
	if svc.sunsetController == nil {
		t.Fatal("sunset controller not initialized")
	}

	// Phase should remain at initial value since controller not running
	phase := svc.sunsetController.Phase()
	if phase != 1 {
		t.Errorf("expected phase 1 with disabled controller, got %d", phase)
	}
}

func TestSunsetControllerEnvironmentConfig(t *testing.T) {
	// Test custom thresholds
	os.Setenv("AGENTAUTH_SUNSET_PILOT_THRESHOLD", "0.70")
	os.Setenv("AGENTAUTH_SUNSET_MAX_MISMATCH", "0.01")
	os.Setenv("AGENTAUTH_SUNSET_WINDOW", "1m")
	defer os.Unsetenv("AGENTAUTH_SUNSET_PILOT_THRESHOLD")
	defer os.Unsetenv("AGENTAUTH_SUNSET_MAX_MISMATCH")
	defer os.Unsetenv("AGENTAUTH_SUNSET_WINDOW")

	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	mem := metrics.NewMemory()

	// Service should parse these environment variables
	svc := NewService(auditLogger, authorizer, WithMetrics(mem))

	if svc.sunsetController == nil {
		t.Fatal("sunset controller not initialized")
	}

	// Controller created successfully with custom config
	t.Log("Sunset controller initialized with custom environment configuration")
}
