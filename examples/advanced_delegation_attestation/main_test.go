package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestDemoRun runs the RunDemo function and validates output.
func TestDemoRun(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = origStdout
		if r := recover(); r != nil {
			t.Errorf("RunDemo panicked: %v", r)
		}
	}()

	RunDemo()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Delegated token valid: true") {
		t.Errorf("Expected output to contain 'Delegated token valid: true', got: %s", output)
	}
}
