package crypto

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRotationAuditTrail ensures rotation emits JSON line when GAUTH_EDDSA_ROTATION_LOG set.
func TestRotationAuditTrail(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rotation.log")
	os.Setenv("GAUTH_EDDSA_ROTATION_LOG", logPath)
	defer os.Unsetenv("GAUTH_EDDSA_ROTATION_LOG")
	m, err := NewManager(1 * time.Hour)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	// First rotation already logged (initial key generation). Force another rotation.
	if _, rotErr := m.Rotate(); rotErr != nil {
		t.Fatalf("rotate: %v", rotErr)
	}
	// Small sleep to ensure write flush
	time.Sleep(50 * time.Millisecond)
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			t.Fatalf("close log: %v", cerr)
		}
	}()
	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if scanErr := scanner.Err(); scanErr != nil {
		t.Fatalf("scan err: %v", scanErr)
	}
	if len(lines) < 2 {
		t.Fatalf("expected >=2 log lines (initial + rotation) got %d", len(lines))
	}
	// Basic field presence check
	last := lines[len(lines)-1]
	if !strings.Contains(last, "eddsa_key_rotated") {
		t.Fatalf("latest line missing event: %s", last)
	}
	if !strings.Contains(last, "new_kid") {
		t.Fatalf("latest line missing new_kid: %s", last)
	}
}
