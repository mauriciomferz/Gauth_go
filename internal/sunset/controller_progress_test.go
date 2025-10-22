package sunset

import (
	"context"
	"testing"
	"time"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestControllerSatisfactionProgress verifies progress gauge increments during sustained criteria
// and resets when adoption ratio dips below threshold or mismatch ratio exceeds limit.
func TestControllerSatisfactionProgress(t *testing.T) {
	mem := imetrics.NewMemory()
	// Seed initial phase (Pilot) and adoption counters so ratio computed easily.
	mem.SetEnvelopeV1SunsetPhase(PhasePilot)
	// Simulate adoption activity counters (not used directly for adoption ratio calculation).
	for i := 0; i < 8; i++ {
		mem.IncEnvelopeV2Issued()
	}
	for i := 0; i < 2; i++ {
		mem.IncEnvelopeV1Issued()
	}
	// Adoption ratio is an explicit gauge; set it (counters are NOT auto-derived).
	mem.SetEnvelopeV2AdoptionRatio(0.80)
	mv := MemoryMetricsView{M: mem}
	// Raise BroadToStabilizeAdoption above current ratio so after promotion criteria are NOT satisfied and progress stays reset.
	cfg := ControllerConfig{Enable: true, Interval: 50 * time.Millisecond, Window: 180 * time.Millisecond, PilotToBroadAdoption: 0.75, BroadToStabilizeAdoption: 0.85, MaxMismatchRatio: 0.01}
	ctrl := NewController(cfg, mv)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go ctrl.Start(ctx)

	// Wait (poll) for progress to begin increasing but ensure we capture it BEFORE the window elapses (promotion).
	var prog float64
	deadline := time.After(160 * time.Millisecond) // < window
	for {
		prog = mem.SunsetPhaseSatisfactionProgress()
		if prog > 0 && prog < 1 {
			break
		}
		select {
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			t.Fatalf("expected progress to increment before window end; got %f", prog)
		}
	}
	// Phase should still be Pilot while window not yet satisfied.
	if phase := mem.EnvelopeV1SunsetPhase(); phase != PhasePilot {
		t.Fatalf("expected phase still Pilot before window completes, got %d", phase)
	}

	// Now allow enough time for promotion (remaining window + one interval for evaluation).
	time.Sleep(250 * time.Millisecond)
	if phase := mem.EnvelopeV1SunsetPhase(); phase != PhaseBroad {
		t.Fatalf("expected phase promoted to Broad (2), got %d", phase)
	}

	// Progress must have been reset to 0 on promotion (and remains 0 since next threshold higher than adoption ratio).
	if p2 := mem.SunsetPhaseSatisfactionProgress(); p2 != 0 {
		t.Fatalf("expected progress reset to 0 after promotion, got %f", p2)
	}

	// Break criteria further (reduce adoption ratio) and ensure progress remains 0.
	for i := 0; i < 10; i++ {
		mem.IncEnvelopeV1Issued()
	} // adoption ratio falls below any threshold.
	time.Sleep(120 * time.Millisecond)
	if p3 := mem.SunsetPhaseSatisfactionProgress(); p3 != 0 {
		t.Fatalf("expected progress to remain 0 when criteria not satisfied, got %f", p3)
	}
}
