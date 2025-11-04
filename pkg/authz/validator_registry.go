package authz

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ValidatorFunc defines a validation function that must return nil for success.
// Any non-nil error causes validation failure (policy fails to match).
type ValidatorFunc func(req Request, policy Policy) error

// ValidatorEntry holds metadata and counters for a validator.
type ValidatorEntry struct {
	ID          string
	Description string
	Version     string
	Tags        []string
	Timeout     time.Duration // optional deadline (<=0 means no timeout wrapping)
	Func        ValidatorFunc
	// Metrics counters
	invocations    uint64
	failures       uint64
	latencyBuckets []int64
	bucketCounts   []uint64 // atomic via AddUint64
}

// ValidatorRegistry manages validator entries.
type ValidatorRegistry struct {
	mu             sync.RWMutex
	entries        map[string]*ValidatorEntry
	defaultBuckets []int64
}

// NewValidatorRegistry constructs a registry with default latency buckets.
func NewValidatorRegistry() *ValidatorRegistry {
	b := []int64{50_000, 100_000, 250_000, 500_000, 1_000_000, 2_500_000, 5_000_000, 10_000_000, 25_000_000, 50_000_000, 100_000_000}
	return &ValidatorRegistry{entries: make(map[string]*ValidatorEntry), defaultBuckets: b}
}

// Register adds a new validator; overwrites existing ID.
func (vr *ValidatorRegistry) Register(id string, fn ValidatorFunc, opts ...func(*ValidatorEntry)) error {
	if id == "" || fn == nil {
		return errors.New("validator id and function required")
	}
	ve := &ValidatorEntry{ID: id, Func: fn, latencyBuckets: vr.defaultBuckets, bucketCounts: make([]uint64, len(vr.defaultBuckets))}
	for _, opt := range opts {
		opt(ve)
	}
	vr.mu.Lock()
	vr.entries[id] = ve
	vr.mu.Unlock()
	return nil
}

// WithDescription option.
func WithDescription(desc string) func(*ValidatorEntry) {
	return func(v *ValidatorEntry) { v.Description = desc }
}

// WithVersion option.
func WithVersion(ver string) func(*ValidatorEntry) {
	return func(v *ValidatorEntry) { v.Version = ver }
}

// WithTags option.
func WithTags(tags ...string) func(*ValidatorEntry) { return func(v *ValidatorEntry) { v.Tags = tags } }

// WithTimeout option.
func WithTimeout(d time.Duration) func(*ValidatorEntry) {
	return func(v *ValidatorEntry) { v.Timeout = d }
}

// Invoke runs the validator by ID, updating metrics. Returns error on failure or missing ID.
func (vr *ValidatorRegistry) Invoke(id string, req Request, policy Policy) error {
	vr.mu.RLock()
	ve := vr.entries[id]
	vr.mu.RUnlock()
	if ve == nil {
		return fmt.Errorf("validator %s not found", id)
	}
	start := time.Now()
	atomic.AddUint64(&ve.invocations, 1)
	var err error
	if ve.Timeout > 0 {
		c := make(chan error, 1)
		go func() { c <- ve.Func(req, policy) }()
		select {
		case err = <-c:
		case <-time.After(ve.Timeout):
			err = fmt.Errorf("validator %s timeout", ve.ID)
		}
	} else {
		err = ve.Func(req, policy)
	}
	lat := time.Since(start).Nanoseconds()
	for i, ub := range ve.latencyBuckets {
		if lat <= ub {
			atomic.AddUint64(&ve.bucketCounts[i], 1)
			break
		}
	}
	if err != nil {
		atomic.AddUint64(&ve.failures, 1)
	}
	return err
}

// ValidatorMetrics snapshot for one validator.
type ValidatorMetrics struct {
	ID               string           `json:"id"`
	Description      string           `json:"description,omitempty"`
	Version          string           `json:"version,omitempty"`
	Tags             []string         `json:"tags,omitempty"`
	Invocations      uint64           `json:"invocations"`
	Failures         uint64           `json:"failures"`
	LatencyHistogram map[int64]uint64 `json:"latency_histogram"`
}

// Snapshot returns metrics for all validators.
func (vr *ValidatorRegistry) Snapshot() []ValidatorMetrics {
	vr.mu.RLock()
	res := make([]ValidatorMetrics, 0, len(vr.entries))
	for _, ve := range vr.entries {
		vm := ValidatorMetrics{ID: ve.ID, Description: ve.Description, Version: ve.Version, Tags: ve.Tags,
			Invocations: atomic.LoadUint64(&ve.invocations), Failures: atomic.LoadUint64(&ve.failures), LatencyHistogram: make(map[int64]uint64)}
		for i, ub := range ve.latencyBuckets {
			cnt := atomic.LoadUint64(&ve.bucketCounts[i])
			if cnt > 0 {
				vm.LatencyHistogram[ub] = cnt
			}
		}
		res = append(res, vm)
	}
	vr.mu.RUnlock()
	return res
}
