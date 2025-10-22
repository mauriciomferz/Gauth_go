package testutil

import "time"

// FakeClock provides controllable time for tests
type FakeClock struct {
	now time.Time
}

func NewFakeClock(start time.Time) *FakeClock { return &FakeClock{now: start} }
func (c *FakeClock) Now() time.Time           { return c.now }
func (c *FakeClock) Advance(d time.Duration)  { c.now = c.now.Add(d) }
