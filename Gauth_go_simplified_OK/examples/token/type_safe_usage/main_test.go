package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestTypeSafeUsageOutput(t *testing.T) {
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

	if err := w.Close(); err != nil {
		t.Errorf("w.Close() error: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Errorf("io.Copy error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "Token ID:") {
		t.Errorf("Expected output to contain 'Token ID:', got: %s", output)
	}
	if !strings.Contains(output, "Token successfully rotated!") {
		t.Errorf("Expected output to contain 'Token successfully rotated!', got: %s", output)
	}
}
