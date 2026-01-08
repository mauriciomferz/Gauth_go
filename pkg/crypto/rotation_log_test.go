package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotationLogSignatureVerification(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "rotation_log.jsonl")
	t.Setenv("AGENTAUTH_EDDSA_ROTATION_LOG", logPath)

	m, err := NewManager(10 * time.Second)
	if err != nil {
		t.Fatalf("manager err: %v", err)
	}
	defer m.Stop()

	// Trigger a rotation to generate a log entry
	_, err = m.Rotate()
	if err != nil {
		t.Fatalf("manual rotate err: %v", err)
	}

	// Read the last log entry
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	lines := splitLines(data)
	if len(lines) == 0 {
		t.Fatalf("no log entries found")
	}
	var entry map[string]any
	if err2 := json.Unmarshal(lines[len(lines)-1], &entry); err2 != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	// Extract signature, public key, and canonical JSON
	sigB64, ok := entry["signature"].(string)
	if !ok {
		t.Fatalf("signature missing from log entry")
	}
	pubB64, ok := entry["public_key"].(string)
	if !ok {
		t.Fatalf("public_key missing from log entry")
	}
	// Remove hash, signature, public_key for canonical JSON
	delete(entry, "hash")
	delete(entry, "signature")
	delete(entry, "public_key")
	canon, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal canonical JSON: %v", err)
	}

	// Verify signature
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		t.Fatalf("failed to decode signature: %v", err)
	}
	pub, err := base64.RawURLEncoding.DecodeString(pubB64)
	if err != nil {
		t.Fatalf("failed to decode public key: %v", err)
	}
	if !ed25519.Verify(pub, canon, sig) {
		t.Fatalf("signature verification failed")
	}

	// Optionally verify hash
	hashB64, ok := entry["hash"].(string)
	if ok {
		h := sha256.Sum256(canon)
		if base64.RawURLEncoding.EncodeToString(h[:]) != hashB64 {
			t.Fatalf("hash mismatch")
		}
	}
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
