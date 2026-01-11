package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLogger_Rotation(t *testing.T) {
	// Create temp dir
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")

	// 1. Create logger
	fl, err := OpenFileLogger(logPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, fl.Close())
	})

	// 2. Set small limits to force rotation quickly
	// An event is roughly ~150-200 bytes depending on content.
	// Set limit to 300 bytes -> should rotate after 2nd or 3rd event.
	fl.SetLimits(300, 5)

	ctx := context.Background()

	// 3. Write Event 1 (should fit)
	ev1 := &Event{
		Type:      "test_event",
		Subject:   "1",
		Timestamp: time.Now().UTC(),
	}
	require.NoError(t, fl.Log(ctx, ev1))

	// Check size
	fi, err := os.Stat(logPath)
	require.NoError(t, err)
	size1 := fi.Size()
	assert.Greater(t, size1, int64(0))

	// 4. Write Event 2 (might fit or trigger soon)
	ev2 := &Event{
		Type:      "test_event",
		Subject:   "2",
		Timestamp: time.Now().UTC(),
	}
	require.NoError(t, fl.Log(ctx, ev2))

	// 5. Write Event 3 (Should definitely trigger rotation if limit is tight)
	ev3 := &Event{
		Type:      "test_event",
		Subject:   "3",
		Timestamp: time.Now().UTC(),
	}
	require.NoError(t, fl.Log(ctx, ev3))

	// check if rotation happened
	// We expect 'audit.log' to be small (just ev3 or empty if just rotated?)
	// If rotated before write, audit.log has ev3.
	// Old log should be audit.log.<timestamp>

	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	rotatedFiles := []string{}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "audit.log.") {
			rotatedFiles = append(rotatedFiles, f.Name())
		}
	}

	assert.NotEmpty(t, rotatedFiles, "Expected at least one rotated file")

	// 6. Verify chain continuity
	// The event in current log should have PrevHash equal to hash of last event in rotated log.

	// Read rotated log last line (Pick the LATEST rotated file if multiple)
	lastRotatedFile := rotatedFiles[len(rotatedFiles)-1]
	rotatedContent, err := os.ReadFile(filepath.Join(dir, lastRotatedFile))
	require.NoError(t, err)
	rotatedLines := strings.Split(strings.TrimSpace(string(rotatedContent)), "\n")
	lastRotatedLine := rotatedLines[len(rotatedLines)-1]

	var lastRotatedEv Event
	err = json.Unmarshal([]byte(lastRotatedLine), &lastRotatedEv)
	require.NoError(t, err)

	// Read current log first line
	currentContent, err := os.ReadFile(logPath)
	require.NoError(t, err)
	currentLines := strings.Split(strings.TrimSpace(string(currentContent)), "\n")
	firstCurrentLine := currentLines[0]

	var firstCurrentEv Event
	err = json.Unmarshal([]byte(firstCurrentLine), &firstCurrentEv)
	require.NoError(t, err)

	// Verification
	assert.Equal(t, lastRotatedEv.Hash, firstCurrentEv.PrevHash, "Chain link broken across files")
	assert.Equal(t, lastRotatedEv.ChainIndex+1, firstCurrentEv.ChainIndex, "Chain index broken across files")
}

func TestFileLogger_Pruning(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "prune.log")

	fl, err := OpenFileLogger(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = fl.Close() })

	// Limit: very small size, max 2 backups
	fl.SetLimits(50, 2)
	ctx := context.Background()

	// Write enough events to trigger multiple rotations
	// Each event is ~100 bytes + overhead, so each likely triggers rotation or fills quickly.
	for i := 0; i < 10; i++ {
		ev := &Event{
			Type:      "prune_test",
			Subject:   string(rune('A' + i)),
			Timestamp: time.Now().UTC(),
		}
		require.NoError(t, fl.Log(ctx, ev))
		// Sleep slightly to ensure timestamps differ for rotation naming
		time.Sleep(20 * time.Millisecond) // Increased sleep to ensure distinct timestamps on fast FS
	}

	// Check files
	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	backups := []string{}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "prune.log.") {
			backups = append(backups, f.Name())
		}
	}

	// We expect exactly 2 backups + 1 active file (active file is not in backups list)
	// Actually logic is: delete oldest until len <= maxBackups.
	// So we should have <= 2 backups.
	assert.LessOrEqual(t, len(backups), 2, "Too many backups retained")
	// We wrote 10 items with limit 50 bytes. Assuming event > 50 bytes.
	// We expect ~9 rotations. So clearly pruning should have happened.
	assert.Equal(t, 2, len(backups), "Expected exactly maxBackups to be retained")
}
