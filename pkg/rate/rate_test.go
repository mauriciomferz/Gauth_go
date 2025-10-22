package rate

import (
	"testing"
	"time"
)

// fakeClock provides deterministic advancement of time for limiter tests.
type fakeClock struct{ now time.Time }

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }
func (c *fakeClock) Now() time.Time          { return c.now }

// TestLimiterDeterministicRefill validates token refill without relying on real time.
func TestLimiterDeterministicRefill(t *testing.T) {
	start := time.Unix(0, 0)
	fc := &fakeClock{now: start}

	cfg := Config{RequestsPerSecond: 10, BurstSize: 10, WindowSize: time.Second}
	lim := NewLimiter(cfg)
	// inject fake clock
	lim.now = fc.Now
	lim.lastRefill = fc.Now()

	// Consume entire burst
	for i := 0; i < cfg.BurstSize; i++ {
		if !lim.Allow() {
			t.Fatalf("expected initial burst token %d to be allowed", i)
		}
	}
	if lim.Allow() { // one more should fail
		t.Fatalf("expected limiter to deny after exhausting burst tokens")
	}

	// Advance half a second: expect ~5 tokens (RequestsPerSecond * 0.5)
	fc.advance(500 * time.Millisecond)
	// First 5 should pass, 6th should fail
	allowed := 0
	for i := 0; i < 6; i++ {
		if lim.Allow() {
			allowed++
		}
	}
	if allowed != 5 { // strict deterministic expectation
		t.Fatalf("expected 5 tokens after 500ms, got %d", allowed)
	}

	// Advance another 500ms to reach 1s total -> should refill another 5
	fc.advance(500 * time.Millisecond)
	allowed = 0
	for i := 0; i < 6; i++ {
		if lim.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("expected 5 more tokens after total 1s, got %d", allowed)
	}

	// Advance a large interval to ensure cap at BurstSize
	fc.advance(10 * time.Second)
	if !lim.Allow() {
		t.Fatalf("expected token after long advance")
	}
	// Drain remaining tokens and ensure not exceeding burst
	drained := 1
	for lim.Allow() {
		drained++
	}
	if drained != cfg.BurstSize {
		t.Fatalf("expected to refill to burst size %d, drained %d", cfg.BurstSize, drained)
	}
}
