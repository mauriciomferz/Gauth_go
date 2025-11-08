package web

// testmain_trace.go provides a custom TestMain that can emit early diagnostic output to help
// debug truncated / piped test runs (e.g., when using `| head` which can trigger SIGPIPE).
// Enable tracing by setting GAUTH_TEST_TRACE_SIGPIPE=1.
// The tracing prints:
//   [trace] test harness starting (pid=...)
//   [trace] heartbeat t=<unix_ms>
// at ~100ms intervals until tests begin executing (first RUN line). Once m.Run() starts,
// heartbeats stop to keep output noise minimal.

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	// Deferred panic recovery to surface any hidden post-test panics that could force a non-zero exit.
	// If a panic occurs after m.Run() (e.g. in a background goroutine finishing late) we log it.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[diag] recovered panic after m.Run(): %v at=%s\n", r, time.Now().Format(time.RFC3339Nano))
		}
	}()

	if os.Getenv("GAUTH_TEST_TRACE_SIGPIPE") == "1" {
		fmt.Fprintf(os.Stderr, "[trace] test harness starting pid=%d at=%s\n", os.Getpid(), time.Now().Format(time.RFC3339Nano))
		// Determine heartbeat phase duration (default 120ms)
		durMs := 120
		if v := os.Getenv("GAUTH_TEST_TRACE_HEARTBEAT_MS"); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed < 10_000 {
				durMs = parsed
			} else {
				fmt.Fprintf(os.Stderr, "[trace] invalid GAUTH_TEST_TRACE_HEARTBEAT_MS=%q (must be 1..9999) using default=%d\n", v, durMs)
			}
		}
		stop := make(chan struct{})
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			deadline := time.Now().Add(time.Duration(durMs) * time.Millisecond)
			for {
				select {
				case <-stop:
					return
				case t := <-ticker.C:
					// Stop heartbeats once deadline passed to avoid noise
					if time.Now().After(deadline) {
						return
					}
					fmt.Fprintf(os.Stderr, "[trace] heartbeat t=%d\n", t.UnixMilli())
				}
			}
		}()
		// Allow main goroutine to block until heartbeat window elapses (or shorter if sleeps tuned)
		time.Sleep(time.Duration(durMs) * time.Millisecond)
		close(stop)
	}

	goroutinesBefore := runtime.NumGoroutine()
	start := time.Now()
	code := m.Run()
	duration := time.Since(start)
	goroutinesAfter := runtime.NumGoroutine()

	// Sentinel diagnostic line to aid log parsing during flake triage.
	fmt.Fprintf(os.Stderr, "[diag] m.Run returned=%d duration_ms=%d goroutines_before=%d goroutines_after=%d at=%s\n", code, duration.Milliseconds(), goroutinesBefore, goroutinesAfter, time.Now().Format(time.RFC3339Nano))
	if os.Getenv("GAUTH_TEST_TRACE_SIGPIPE") == "1" {
		fmt.Fprintf(os.Stderr, "[trace] test harness finished code=%d at=%s\n", code, time.Now().Format(time.RFC3339Nano))
	}
	// Return code to allow defers to complete before exit.
	return code
}
