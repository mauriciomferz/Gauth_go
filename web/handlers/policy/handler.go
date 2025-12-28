package policy

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
	"github.com/mauriciomferz/Gauth_go/pkg/policy"
)

type Handler struct {
	Registry        *policy.Registry
	Engine          *policy.ChainEngine
	RateLimiter     *SimpleRateLimiter
	Metrics         *Metrics
	Config          Config
	PromMetrics     metrics.Metrics // For Prometheus integration
	RevocationChain interface {
		IsDelegationRevoked(id, hash string) bool
		LatestTreeHead() *delegation.SignedTreeHead
	}
	OnPolicyChange func()
}

func NewHandler(persistPath string, m metrics.Metrics) *Handler {
	h := &Handler{
		Config:      Config{PersistPath: persistPath},
		RateLimiter: NewSimpleRateLimiter(20, time.Minute),
		Metrics: &Metrics{
			LatencyBuckets: make(map[int64]*uint64),
		},
		PromMetrics: m,
	}

	// Initialize legacy buckets
	bounds := []int64{1000, 5000, 10000, 50000, 100000, 500000, 1000000, 5000000}
	for _, b := range bounds {
		var z uint64
		h.Metrics.LatencyBuckets[b] = &z
	}

	// Initialize registry and engine
	// Note: In server_clean.go these were potentially nil. We'll initialize them lazily or here.
	// For now, mirroring lazy initialization pattern if strictly required, but explicit is better.
	// We'll leave them nil and let the logic handle initialization if that's how the server worked,
	// OR better, initialize them here if possible.
	// Looking at server code: "Experimental policy engine provenance (optional; initialized lazily)"
	// We will follow the lazy pattern or check if we can init now.

	return h
}

// EnsureInitialized makes sure the registry and engine are ready
func (h *Handler) EnsureInitialized() {
	if h.Registry == nil {
		if h.Config.PersistPath != "" {
			if err := h.loadState(h.Config.PersistPath); err != nil {
				fmt.Printf("[policy] Warning: Failed to load state: %v\n", err)
			}
		}
		if h.Registry == nil {
			h.Registry = policy.NewRegistry()
		}
	}
	if h.Engine == nil {
		h.Engine = policy.NewChainEngine(h.Registry)
	}
}

func (h *Handler) SaveState() error {
	if h.Registry != nil && h.Config.PersistPath != "" {
		return h.saveState(h.Config.PersistPath)
	}
	return nil
}

func (h *Handler) RecordLatency(d time.Duration) {
	ns := d.Nanoseconds()
	h.Metrics.Lock()
	defer h.Metrics.Unlock()
	for bound, countPtr := range h.Metrics.LatencyBuckets {
		if ns <= bound {
			atomic.AddUint64(countPtr, 1)
		}
	}
	// Simplified P99 tracking (just keeping last high value for demo)
	if ns > h.Metrics.P99LatencyNS {
		h.Metrics.P99LatencyNS = ns
	}
}

// Evaluate delegates policy evaluation to the engine.
func (h *Handler) Evaluate(ctx context.Context, req policy.EvalRequest) (policy.EvalDecision, error) {
	if h.Engine == nil {
		h.EnsureInitialized()
	}
	if h.Engine == nil {
		return policy.EvalDecision{}, fmt.Errorf("policy engine not initialized")
	}
	return h.Engine.Evaluate(ctx, req)
}
