package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testSubject = "tester"
	testObject  = "obj"
)

func TestFileLoggerAppendAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	fl, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = fl.Close() }()
	// Append events
	for i := 0; i < 3; i++ {
		ev := NewEvent(EventTypeAuthorization, "action", ResultSuccess)
		ev.Subject = "alice"
		ev.Object = "resource"
		if err2 := fl.Log(context.Background(), ev); err2 != nil {
			t.Fatalf("log %d: %v", i, err)
		}
	}

	// Verify persistence by counting lines
	count, err := countEventsInFile(path)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 events, got %d", count)
	}
	if err2 := fl.VerifyChain(); err2 != nil {
		t.Fatalf("verify: %v", err)
	}
	// Re-open
	if err2 := fl.Close(); err2 != nil {
		t.Fatalf("close: %v", err2)
	}
	fl2, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = fl2.Close() }()
	if err := fl2.VerifyChain(); err != nil {
		t.Fatalf("verify reloaded: %v", err)
	}

	// Check count again
	count, err = countEventsInFile(path)
	if err != nil {
		t.Fatalf("count reloaded: %v", err)
	}
	if count != 3 {
		t.Fatalf("reloaded events mismatch: %d", count)
	}
}

// TestFileLoggerReloadIntegrity performs a multi-phase write/reopen to ensure integrity holds
// across application restarts and that querying after reload returns the full set.
func TestFileLoggerReloadIntegrity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	// Phase 1: create and write 5 events
	fl, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("open phase1: %v", err)
	}
	for i := 0; i < 5; i++ {
		ev := NewEvent(EventTypeAuthorization, "phase1", ResultSuccess)
		ev.Subject = testSubject
		ev.Object = testObject
		if err2 := fl.Log(context.Background(), ev); err2 != nil {
			t.Fatalf("log phase1 %d: %v", i, err)
		}
	}
	if err2 := fl.Close(); err2 != nil {
		t.Fatalf("close phase1: %v", err)
	}

	// Phase 2: reopen and append 3 more
	fl2, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("reopen phase2: %v", err)
	}
	for i := 0; i < 3; i++ {
		ev := NewEvent(EventTypeAuthorization, "phase2", ResultSuccess)
		ev.Subject = testSubject
		ev.Object = testObject
		if err2 := fl2.Log(context.Background(), ev); err2 != nil {
			t.Fatalf("log phase2 %d: %v", i, err)
		}
	}

	count, err := countEventsInFile(path)
	if err != nil {
		t.Fatalf("count phase2: %v", err)
	}
	if count != 8 {
		t.Fatalf("expected 8 events after phase2, got %d", count)
	}
	if err2 := fl2.VerifyChain(); err2 != nil {
		t.Fatalf("verify phase2: %v", err2)
	}
	if err2 := fl2.Close(); err2 != nil {
		t.Fatalf("close phase2: %v", err2)
	}

	// Phase 3: final reopen for verification only
	fl3, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("reopen phase3: %v", err)
	}
	defer func() { _ = fl3.Close() }()
	defer func() { _ = fl3.Close() }()

	count, err = countEventsInFile(path)
	if err != nil {
		t.Fatalf("count phase3: %v", err)
	}
	if count != 8 {
		t.Fatalf("expected 8 events phase3, got %d", count)
	}
	if err := fl3.VerifyChain(); err != nil {
		t.Fatalf("verify phase3: %v", err)
	}

	// Query is no longer supported in FileLogger
	_, qerr := fl3.Query(context.TODO(), nil)
	if qerr == nil {
		t.Fatal("expected query to fail with not supported")
	}
}

func countEventsInFile(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	// Filter empty lines
	lines := 0
	for _, line := range strings.Split(string(b), "\n") {
		if len(line) > 0 {
			lines++
		}
	}
	return lines, nil
}

func TestFileLoggerTamperDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	fl, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ev := NewEvent(EventTypeAuthorization, "act", ResultSuccess)
	if err2 := fl.Log(context.Background(), ev); err2 != nil {
		t.Fatalf("log: %v", err2)
	}
	fl.Close()
	// Tamper: modify first line hash
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// naive replace of hash with all zeros; find "\"hash\":"
	tampered := b
	// This is fragile but adequate for demonstration
	tamperedStr := string(tampered)
	// Replace first occurrence of hash value (64 hex chars) with 64 zeros
	// locate pattern "\"hash\":\"" then take next 64 chars
	idx := -1
	if off := indexOf(tamperedStr, "\"hash\":\""); off != -1 {
		idx = off + len("\"hash\":\"")
		if idx+64 <= len(tamperedStr) {
			zeros := make([]byte, 64)
			for i := range zeros {
				zeros[i] = '0'
			}
			tamperedStr = tamperedStr[:idx] + string(zeros) + tamperedStr[idx+64:]
		}
	}
	if idx == -1 {
		t.Skip("could not locate hash field for tamper test")
	}
	if err := os.WriteFile(path, []byte(tamperedStr), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Reopen should fail load
	if _, err := OpenFileLogger(path); err == nil {
		t.Fatalf("expected tamper detection error")
	}
}

// indexOf returns first index of sub in s (string.Index replacement allowing building for tests without extra import).
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
