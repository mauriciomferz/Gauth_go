package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestAdvancedRevocationFlowOutput(t *testing.T) {
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

	if !strings.Contains(output, "Token granted with multi-attestation.") {
		t.Errorf("Expected output to contain 'Token granted with multi-attestation.', got: %s", output)
	}
	if !strings.Contains(output, "Token delegated to agent-1.") {
		t.Errorf("Expected output to contain 'Token delegated to agent-1.', got: %s", output)
	}
	if !strings.Contains(output, "Delegated token for agent-2 revoked.") {
		t.Errorf("Expected output to contain 'Delegated token for agent-2 revoked.', got: %s", output)
	}
       if !strings.Contains(output, "Get after revoke error for agent-2: token not found") {
	       t.Errorf("Expected output to contain 'Get after revoke error for agent-2: token not found', got: %s", output)
       }
	if !strings.Contains(output, "Token for agent-1 valid: true") {
		t.Errorf("Expected output to contain 'Token for agent-1 valid: true', got: %s", output)
	}
	if !strings.Contains(output, "Token for agent-3 valid: true") {
		t.Errorf("Expected output to contain 'Token for agent-3 valid: true', got: %s", output)
	}
}
