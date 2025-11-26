// Package sunset implements automated lifecycle phase management for deprecating
// envelope V1 in favor of V2. It evaluates adoption & integrity metrics over
// sliding windows to promote phases and exposes progress gauges.
package sunset

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
)

// Phase values mirror OBSERVABILITY documentation.
const (
	PhasePilot            = 1
	PhaseBroad            = 2
	PhaseStabilization    = 3
	PhaseSoftDeprecation  = 4
	PhaseSunset           = 5
	PhasePostVerification = 6
)

// ControllerConfig defines thresholds & cadence for automatic phase promotion.
type ControllerConfig struct {
	// Required minimum adoption ratio to promote each phase (>= value sustained for window).
	PilotToBroadAdoption       float64
	BroadToStabilizeAdoption   float64
	StabilizeToSoftDepAdoption float64
	SoftDepToSunsetAdoption    float64
	// Maximum allowed mismatch ratio over the window (mismatches / issued V2) for promotion.
	MaxMismatchRatio float64
	// Observation window duration (continuous satisfaction needed).
	Window time.Duration
	// Evaluation interval.
	Interval time.Duration
	// Enable automatic promotions (disabled if false).
	Enable bool
	// Allow emergency rollback via env or explicit SetPhase calls (controller never decrements phase unless ForceRollback flag set).
	AllowRollback bool
}

// MetricsView abstracts required metric reads for controller (memory implementation satisfies this).
type MetricsView interface {
	EnvelopeV2IssuedCount() uint64
	EnvelopeDigestMismatchCount() uint64
	EnvelopeV2AdoptionRatio() float64
	EnvelopeV1SunsetPhase() uint64
	SetEnvelopeV1SunsetPhase(int)
	// Optional progress gauge: fraction (0..1) of current satisfaction window elapsed while criteria held.
	SetSunsetPhaseSatisfactionProgress(float64)
}

// Controller manages automatic phase transitions.
type Controller struct {
	cfg              ControllerConfig
	mv               MetricsView
	running          atomic.Bool
	lastSatisfyStart time.Time
}

// Progress Gauge Semantics:
// The satisfaction progress gauge represents the fractional time elapsed within the current
// promotion window while the adoption & mismatch criteria remain continuously satisfied.
// It is updated only on evaluation ticks (Interval). When criteria break it resets to 0.
// On successful phase promotion it is also reset to 0 and the next phase's threshold applies.
// NOTE: The adoption ratio is expected to be provided by instrumentation (e.g. SetEnvelopeV2AdoptionRatio);
// it is not auto-derived from raw V1/V2 issuance counters inside this controller.

// NewController constructs a controller.
func NewController(cfg ControllerConfig, mv MetricsView) *Controller {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.Window <= 0 {
		cfg.Window = 15 * time.Minute
	}
	if cfg.PilotToBroadAdoption == 0 {
		cfg.PilotToBroadAdoption = 0.60
	}
	if cfg.BroadToStabilizeAdoption == 0 {
		cfg.BroadToStabilizeAdoption = 0.80
	}
	if cfg.StabilizeToSoftDepAdoption == 0 {
		cfg.StabilizeToSoftDepAdoption = 0.90
	}
	if cfg.SoftDepToSunsetAdoption == 0 {
		cfg.SoftDepToSunsetAdoption = 0.95
	}
	if cfg.MaxMismatchRatio == 0 {
		cfg.MaxMismatchRatio = 0.005
	}
	return &Controller{cfg: cfg, mv: mv}
}

// Start begins periodic evaluation until context canceled.
func (c *Controller) Start(ctx context.Context) {
	if c.mv == nil || !c.cfg.Enable || !c.running.CompareAndSwap(false, true) {
		return
	}
	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done(): // context canceled
			return
		case <-ticker.C:
			c.evaluate()
		}
	}
}

func (c *Controller) evaluate() {
	adoption := c.mv.EnvelopeV2AdoptionRatio()
	mismatches := c.mv.EnvelopeDigestMismatchCount()
	v2 := c.mv.EnvelopeV2IssuedCount()
	phase := c.mv.EnvelopeV1SunsetPhase()
	var mismatchRatio float64
	if v2 > 0 {
		mismatchRatio = float64(mismatches) / float64(v2)
	}
	//nolint:gosec // G115: phase enum value, small range
	threshold := adoptionThresholdForPhase(c.cfg, int(phase))
	if adoption >= threshold && mismatchRatio <= c.cfg.MaxMismatchRatio {
		// Start or continue satisfy window
		if c.lastSatisfyStart.IsZero() {
			c.lastSatisfyStart = time.Now()
		}
		elapsed := time.Since(c.lastSatisfyStart)
		// Update progress gauge
		prog := elapsed.Seconds() / c.cfg.Window.Seconds()
		if prog < 0 {
			prog = 0
		}
		if prog > 1 {
			prog = 1
		}
		c.mv.SetSunsetPhaseSatisfactionProgress(prog)
		if time.Since(c.lastSatisfyStart) >= c.cfg.Window {
			//nolint:gosec // G115: phase enum value, small range
			next := nextPhase(int(phase))
			//nolint:gosec // G115: phase enum value, small range
			if next > int(phase) {
				c.mv.SetEnvelopeV1SunsetPhase(next)
				log.Printf("sunset controller: promoted phase %d -> %d (adoption=%.3f mismatch=%.5f)", phase, next, adoption, mismatchRatio)
				c.lastSatisfyStart = time.Time{} // reset for next promotion
				c.mv.SetSunsetPhaseSatisfactionProgress(0)
			}
		}
	} else {
		// Reset window if criteria break
		c.lastSatisfyStart = time.Time{}
		c.mv.SetSunsetPhaseSatisfactionProgress(0)
	}
}

func adoptionThresholdForPhase(cfg ControllerConfig, phase int) float64 {
	switch phase {
	case PhasePilot:
		return cfg.PilotToBroadAdoption
	case PhaseBroad:
		return cfg.BroadToStabilizeAdoption
	case PhaseStabilization:
		return cfg.StabilizeToSoftDepAdoption
	case PhaseSoftDeprecation:
		return cfg.SoftDepToSunsetAdoption
	case PhaseSunset:
		return 1.0 // no further promotion
	case PhasePostVerification:
		return 1.0
	default:
		return cfg.PilotToBroadAdoption
	}
}

func nextPhase(current int) int {
	switch current {
	case PhasePilot:
		return PhaseBroad
	case PhaseBroad:
		return PhaseStabilization
	case PhaseStabilization:
		return PhaseSoftDeprecation
	case PhaseSoftDeprecation:
		return PhaseSunset
	case PhaseSunset:
		return PhasePostVerification
	default:
		return current
	}
}

// MemoryMetricsView adapts *metrics.Memory to MetricsView.
type MemoryMetricsView struct{ M *metrics.Memory }

func (v MemoryMetricsView) EnvelopeV2IssuedCount() uint64 { return v.M.EnvelopeV2IssuedCount() }
func (v MemoryMetricsView) EnvelopeDigestMismatchCount() uint64 {
	return v.M.EnvelopeDigestMismatchCount()
}
func (v MemoryMetricsView) EnvelopeV2AdoptionRatio() float64 { return v.M.EnvelopeV2AdoptionRatio() }
func (v MemoryMetricsView) EnvelopeV1SunsetPhase() uint64    { return v.M.EnvelopeV1SunsetPhase() }
func (v MemoryMetricsView) SetEnvelopeV1SunsetPhase(p int)   { v.M.SetEnvelopeV1SunsetPhase(p) }
func (v MemoryMetricsView) SetSunsetPhaseSatisfactionProgress(p float64) {
	v.M.SetSunsetPhaseSatisfactionProgress(p)
}
