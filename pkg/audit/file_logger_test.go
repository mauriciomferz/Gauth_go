package audit

import (
	"context"
	"os"
	"path/filepath"
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
	if len(fl.Events()) != 3 {
		t.Fatalf("expected 3 events, got %d", len(fl.Events()))
	}
	if err2 := fl.VerifyChain(); err2 != nil {
		t.Fatalf("verify: %v", err)
	}
	// Re-open
	if err := fl.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	fl2, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = fl2.Close() }()
	if len(fl2.Events()) != 3 {
		t.Fatalf("reloaded events mismatch: %d", len(fl2.Events()))
	}
	if err := fl2.VerifyChain(); err != nil {
		t.Fatalf("verify reloaded: %v", err)
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
	if len(fl2.Events()) != 8 {
		t.Fatalf("expected 8 events after phase2, got %d", len(fl2.Events()))
	}
	if err := fl2.VerifyChain(); err != nil {
		t.Fatalf("verify phase2: %v", err)
	}
	if err := fl2.Close(); err != nil {
		t.Fatalf("close phase2: %v", err)
	}

	// Phase 3: final reopen for verification only
	fl3, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("reopen phase3: %v", err)
	}
	defer func() { _ = fl3.Close() }()
	if len(fl3.Events()) != 8 {
		t.Fatalf("expected 8 events phase3, got %d", len(fl3.Events()))
	}
	if err := fl3.VerifyChain(); err != nil {
		t.Fatalf("verify phase3: %v", err)
	}

	// Query all events (nil filters)
	queried, qerr := fl3.Query(context.TODO(), nil)
	if qerr != nil {
		t.Fatalf("query: %v", qerr)
	}
	if len(queried) != 8 {
		t.Fatalf("query expected 8 events, got %d", len(queried))
	}
}

func TestFileLoggerTamperDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	fl, err := OpenFileLogger(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ev := NewEvent(EventTypeAuthorization, "act", ResultSuccess)
	if err := fl.Log(context.Background(), ev); err != nil {
		t.Fatalf("log: %v", err)
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
