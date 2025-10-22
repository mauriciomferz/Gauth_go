package web

import (
	"runtime"
	"testing"
)

// TestZZZFinalDiagnostics runs last alphabetically to capture final runtime state.
// It intentionally does not fail; it emits goroutine count to help diagnose hidden package-level FAIL.
func TestZZZFinalDiagnostics(t *testing.T) {
	t.Logf("[final-diagnostics] goroutines=%d", runtime.NumGoroutine())
}
