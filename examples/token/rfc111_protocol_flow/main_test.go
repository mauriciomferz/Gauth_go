package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRFC111ProtocolFlowOutput(t *testing.T) {
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

	if !strings.Contains(output, "Owner proof for user-42:") {
		t.Errorf("Expected output to contain 'Owner proof for user-42:', got: %s", output)
	}
	if !strings.Contains(output, "Token granted.") {
		t.Errorf("Expected output to contain 'Token granted.', got: %s", output)
	}
	if !strings.Contains(output, "Attestation:") {
		t.Errorf("Expected output to contain 'Attestation:', got: %s", output)
	}
	if !strings.Contains(output, "Token valid: true") {
		t.Errorf("Expected output to contain 'Token valid: true', got: %s", output)
	}
	if !strings.Contains(output, "Token revoked.") {
		t.Errorf("Expected output to contain 'Token revoked.', got: %s", output)
	}
	if !strings.Contains(output, "Get after revoke error: token not found") {
		t.Errorf("Expected output to contain 'Get after revoke error: token not found', got: %s", output)
	}
}
