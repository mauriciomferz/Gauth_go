// Package obligations defines execution interfaces for policy decision
// obligations/advice (prototype; no side-effectful implementations yet).
package obligations

import "context"

// Result represents the outcome of an obligation/advice execution.
type Result struct {
	Name    string
	Success bool
	Error   error
}

// Executor defines interface for executing obligations/advice derived from policy decisions.
type Executor interface {
	Execute(ctx context.Context, names []string) []Result
}

// NoopExecutor executes nothing (returns success for each name) useful for testing / disabled mode.
type NoopExecutor struct{}

func (n NoopExecutor) Execute(ctx context.Context, names []string) []Result {
	out := make([]Result, 0, len(names))
	for _, name := range names {
		out = append(out, Result{Name: name, Success: true})
	}
	return out
}
