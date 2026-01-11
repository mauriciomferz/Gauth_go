package audit

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestFileLoggerArchival(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")
	archiveDir := filepath.Join(dir, "archives")

	// Set env vars for OpenFileLogger
	t.Setenv("AGENTAUTH_AUDIT_ARCHIVE_DIR", archiveDir)
	t.Setenv("AGENTAUTH_AUDIT_ARCHIVE_COMPRESS", "1")
	t.Setenv("AGENTAUTH_AUDIT_ARCHIVE_MAX_COUNT", "2")

	fl, err := OpenFileLogger(logPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if err := fl.Close(); err != nil {
			t.Errorf("file logger close: %v", err)
		}
	})

	// Set very small size to trigger rotation after almost every event
	fl.SetLimits(10, 1) // 10 bytes

	// Log some events
	for i := 0; i < 5; i++ {
		ev := NewEvent(EventTypeAuthorization, "test-archival", ResultSuccess)
		if err := fl.Log(context.Background(), ev); err != nil {
			t.Fatalf("log %d: %v", i, err)
		}
		// Small sleep to ensure different timestamps if rotation format uses them
		time.Sleep(10 * time.Millisecond)
	}

	// Standard rotation creates fl.path.timestamp
	// archival moves it to archiveDir/fl.path.timestamp.gz

	// Wait for async archival
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	found := false
loop:
	for {
		select {
		case <-timeout:
			break loop
		case <-ticker.C:
			matches, _ := filepath.Glob(filepath.Join(archiveDir, "*.gz"))
			if len(matches) > 0 {
				found = true
				break loop
			}
		}
	}

	if !found {
		t.Fatal("expected at least one archived file (.gz), found none")
	}

	// Verify pruning: log many more events
	for i := 0; i < 20; i++ {
		ev := NewEvent(EventTypeAuthorization, "test-pruning", ResultSuccess)
		_ = fl.Log(context.Background(), ev)
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for pruning
	time.Sleep(500 * time.Millisecond)

	matches, _ := filepath.Glob(filepath.Join(archiveDir, "*.gz"))
	if len(matches) == 0 {
		t.Fatal("expected archives to exist, but none found")
	}
	if len(matches) > 2 {
		t.Fatalf("expected at most 2 archives due to pruning (AGENTAUTH_AUDIT_ARCHIVE_MAX_COUNT=2), found %d", len(matches))
	}
}
