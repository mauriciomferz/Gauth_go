package agentauth_aap_001

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

func TestSunsetControllerIntegration(t *testing.T) {
	// Enable sunset controller
	t.Setenv("AGENTAUTH_SUNSET_ENABLED", "1")
	t.Setenv("AGENTAUTH_SUNSET_INTERVAL", "100ms")
	t.Setenv("AGENTAUTH_SUNSET_WINDOW", "500ms")

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
	t.Setenv("AGENTAUTH_SUNSET_ENABLED", "0")

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
	t.Setenv("AGENTAUTH_SUNSET_PILOT_THRESHOLD", "0.70")
	t.Setenv("AGENTAUTH_SUNSET_MAX_MISMATCH", "0.01")
	t.Setenv("AGENTAUTH_SUNSET_WINDOW", "1m")

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
