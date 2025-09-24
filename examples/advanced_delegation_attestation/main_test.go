package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestMainExample runs the main function to ensure the example runs without panicking.
func TestMainExample(t *testing.T) {
	// Capture stdout
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = origStdout
		if r := recover(); r != nil {
			t.Errorf("main panicked: %v", r)
		}
	}()

	main()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Delegated token valid: true") {
		t.Errorf("Expected output to contain 'Delegated token valid: true', got: %s", output)
	}
}
