package main

import (
	"testing"
)

// TestMainFunction ensures the demo main executes without panic. It indirectly
// increases coverage for printing code paths otherwise unexercised.
func TestMainFunction(t *testing.T) {
	// Defer recover to catch any unexpected panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("main panicked: %v", r)
		}
	}()
	main()
}
