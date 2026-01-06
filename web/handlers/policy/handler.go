package policy

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

type Handler struct {
	Store           policy.Store
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

func NewHandler(store policy.Store, m metrics.Metrics) *Handler {
	h := &Handler{
		Store:       store,
		Config:      Config{},
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

// EnsureInitialized makes sure the engine is ready
func (h *Handler) EnsureInitialized() {
	if h.Engine == nil {
		h.Engine = policy.NewChainEngine(h.Store)
	}
}

func (h *Handler) SaveState() error {
	// Persistence is now handled by the Store implementation internally.
	// This method is kept for compatibility but might be no-op for DB stores.
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
	return h.Engine.Evaluate(ctx, req)
}
