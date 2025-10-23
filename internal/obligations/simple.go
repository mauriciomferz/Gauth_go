package obligations

import (
	"context"
	"fmt"
	"time"
)

// SimpleExecutor is a placeholder demonstrating latency + failure injection via context.
// Future: integrate with policy engine to translate decision outputs into obligation names and parameters.
type SimpleExecutor struct {
	MinLatency time.Duration
	MaxLatency time.Duration
	FailSet    map[string]struct{} // names that should fail (demo)
}

func NewSimpleExecutor(minMs, maxMs int, failNames []string) *SimpleExecutor {
	if minMs < 0 {
		minMs = 0
	}
	if maxMs < minMs {
		maxMs = minMs
	}
	fs := make(map[string]struct{}, len(failNames))
	for _, n := range failNames {
		fs[n] = struct{}{}
	}
	return &SimpleExecutor{
		MinLatency: time.Duration(minMs) * time.Millisecond,
		MaxLatency: time.Duration(maxMs) * time.Millisecond,
		FailSet:    fs,
	}
}

func (s *SimpleExecutor) Execute(ctx context.Context, names []string) []Result {
	out := make([]Result, 0, len(names))
	for _, name := range names {
		// Simulate latency
		span := s.MinLatency
		if s.MaxLatency > s.MinLatency {
			delta := time.Duration(time.Now().UnixNano() % int64(s.MaxLatency-s.MinLatency))
			span += delta
		}
		select {
		case <-time.After(span):
			// proceed
		case <-ctx.Done():
			out = append(out, Result{Name: name, Success: false, Error: ctx.Err()})
			continue
		}
		if _, shouldFail := s.FailSet[name]; shouldFail {
			out = append(out, Result{Name: name, Success: false, Error: fmt.Errorf("obligation %s failed (simulated)", name)})
			continue
		}
		out = append(out, Result{Name: name, Success: true})
	}
	return out
}
