// Package testutil provides lightweight helpers for integration and unit tests
// (environment scoping, request constructors, etc.). Kept intentionally minimal
// to avoid masking production issues; not a general purpose test framework.
package testutil

import (
	"os"
	"testing"
)

// WithEnv sets environment variable k to v for the duration of fn and restores previous value afterward.
// Pass empty string v to unset variable during fn execution.
func WithEnv(t *testing.T, k, v string, fn func()) {
	prev := os.Getenv(k)
	if v == "" {
		if err := os.Unsetenv(k); err != nil {
			// Non-fatal for tests; record failure
			t.Fatalf("unset %s failed: %v", k, err)
		}
	} else {
		if err := os.Setenv(k, v); err != nil {
			// Fail fast; environment mutation required for test semantics
			t.Fatalf("set %s failed: %v", k, err)
		}
	}
	t.Cleanup(func() {
		if err := os.Setenv(k, prev); err != nil {
			// Use Logf in cleanup to avoid masking prior test result
			t.Logf("restore %s failed: %v", k, err)
		}
	})
	fn()
}
