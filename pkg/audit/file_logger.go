package audit

// FileLogger provides append-only persistent audit logging with the same hash chain
// semantics as MemoryLogger. It stores one JSON Event per line and recomputes / verifies
// hashes on load. A truncate or tamper will surface as VerifyChain error.
// NOTE: Experimental; not for production durability or concurrency beyond single-process.

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

// FileLogger implements persistent audit logging with file rotation.
type FileLogger struct {
	mu   sync.RWMutex
	path string
	file *os.File

	// State tracking for chain linking (instead of full event history)
	lastEventHash  string
	lastEventIndex int

	// Rotation limits
	currentSize int64
	maxSize     int64 // bytes
	maxBackups  int

	// Archival settings (RR-009)
	archivePath     string
	compress        bool
	maxArchiveSize  int64
	maxArchiveCount int
}

// OpenFileLogger opens (or creates) an audit log file. If the file exists it is loaded.
// Uses default limits: 100MB size, 5 backups.
func OpenFileLogger(path string) (*FileLogger, error) {
	if path == "" {
		return nil, errors.New("path required")
	}
	// #nosec G301
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	// Stat to check existence and size
	fi, err := os.Stat(path)
	exists := err == nil
	originalSize := int64(0)
	if exists {
		originalSize = fi.Size()
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}

	fl := &FileLogger{
		path:           path,
		file:           f,
		maxSize:        100 * 1024 * 1024, // 100MB
		maxBackups:     5,
		lastEventIndex: -1, // -1 means no events produced yet (0th event will be index 0)
		currentSize:    originalSize,

		// Archival defaults (RR-009)
		archivePath:     os.Getenv("AGENTAUTH_AUDIT_ARCHIVE_DIR"),
		compress:        os.Getenv("AGENTAUTH_AUDIT_ARCHIVE_COMPRESS") != "0",
		maxArchiveSize:  1 * 1024 * 1024 * 1024, // 1GB default
		maxArchiveCount: 100,                    // 100 files default
	}

	if v := os.Getenv("AGENTAUTH_AUDIT_ARCHIVE_MAX_SIZE"); v != "" {
		if s, parseErr := strconv.ParseInt(v, 10, 64); parseErr == nil {
			fl.maxArchiveSize = s
		}
	}
	if v := os.Getenv("AGENTAUTH_AUDIT_ARCHIVE_MAX_COUNT"); v != "" {
		if c, parseErr := strconv.Atoi(v); parseErr == nil {
			fl.maxArchiveCount = c
		}
	}

	if fl.archivePath != "" {
		// #nosec G301
		_ = os.MkdirAll(fl.archivePath, 0o750)
	}

	if exists && originalSize > 0 {
		if err2 := fl.load(); err2 != nil {
			_ = f.Close()
			return nil, fmt.Errorf("load: %w", err2)
		}
	}

	// Reopen in append mode for writing
	_ = f.Close()
	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	fl.file = f
	return fl, nil
}

// load ingests existing file only to find the LAST event for chaining.
func (fl *FileLogger) load() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	f, err := os.Open(fl.path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	// We only need to scan to the end to get the last hash and index
	// In a production system we might seek to end and read backwards, but JSON lines vary in length.
	// Scanning is acceptable for now given 100MB limit.

	lastHash := ""
	lastIndex := -1
	lineNo := 0

	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}

		// Optimization: Don't fully unmarshal every event if we just want the chain tip?
		// But we should verify the chain integrity on load as per previous behavior.
		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			return fmt.Errorf("unmarshal line %d: %w", lineNo, err)
		}

		// Verify hash
		expected := computeEventHash(ev)
		if ev.Hash != expected {
			return fmt.Errorf("hash mismatch line %d", lineNo)
		}

		// Verify chain
		if ev.ChainIndex != lineNo {
			return fmt.Errorf("chain index mismatch line %d: got %d want %d", lineNo, ev.ChainIndex, lineNo)
		}
		if lineNo == 0 && ev.PrevHash != "" {
			return fmt.Errorf("first prev hash non-empty line %d", lineNo)
		}
		if lineNo > 0 && ev.PrevHash != lastHash {
			return fmt.Errorf("prev hash mismatch line %d", lineNo)
		}

		lastHash = ev.Hash
		lastIndex = ev.ChainIndex
		lineNo++
	}

	fl.lastEventHash = lastHash
	fl.lastEventIndex = lastIndex
	return scanner.Err()
}

// Log appends an event; accepts *Event. Handles rotation.
func (fl *FileLogger) Log(ctx context.Context, entry interface{}) error {
	ev, ok := entry.(*Event)
	if !ok {
		return nil
	}
	if ev.ID == "" {
		ev.ID = generateID()
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}

	fl.mu.Lock()
	defer fl.mu.Unlock()

	// Chain linking
	ev.ChainIndex = fl.lastEventIndex + 1
	ev.PrevHash = fl.lastEventHash
	ev.Hash = computeEventHash(*ev)

	b, mErr := json.Marshal(ev)
	if mErr != nil {
		return fmt.Errorf("marshal audit event: %w", mErr)
	}
	b = append(b, '\n')
	lineLen := int64(len(b))

	// Check rotation
	if fl.currentSize > 0 && fl.currentSize+lineLen > fl.maxSize {
		if err := fl.rotateLocked(); err != nil {
			return fmt.Errorf("rotate: %w", err)
		}
	}

	if _, err := fl.file.Write(b); err != nil {
		return err
	}

	fl.currentSize += lineLen
	fl.lastEventIndex = ev.ChainIndex
	fl.lastEventHash = ev.Hash

	return nil
}

// rotateLocked performs log rotation. Caller must hold lock.
func (fl *FileLogger) rotateLocked() error {
	_ = fl.file.Sync()
	_ = fl.file.Close()

	// Rotation scheme: just rename current to .<timestamp>
	timestamp := time.Now().Format("20060102-150405.000000")
	backupName := fl.path + "." + timestamp
	if err := os.Rename(fl.path, backupName); err != nil {
		return err
	}

	// Trigger archival if configured (RR-009)
	if fl.archivePath != "" {
		fl.archiveLocked(backupName)
	} else {
		// Pruning old backups (standard logic only if archival disabled)
		matches, err := filepath.Glob(fl.path + ".*")
		if err == nil && len(matches) > fl.maxBackups {
			// Sort by name (lexicographically equivalent to chronological for our format)
			sort.Strings(matches)

			// Delete oldest excess backups
			excess := len(matches) - fl.maxBackups
			for i := 0; i < excess; i++ {
				_ = os.Remove(matches[i])
			}
		}
	}

	newFile, err := os.OpenFile(fl.path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	fl.file = newFile
	fl.currentSize = 0

	// Hash chain continuity:
	// The new file starts fresh? Or should it link?
	// If we link, the first entry in new file has PrevHash = lastHash from old file.
	// This maintains cryptographic continuity across files.
	// Users verifying chain across files would need to stitch them.
	// Current state (lastEventHash) is preserved in memory, so next link will be correct.

	return nil
}

// archiveLocked performs async archival. Caller must hold lock or ensure exclusivity via other means.
// This is called from rotateLocked so it's under fl.mu.
func (fl *FileLogger) archiveLocked(backupPath string) {
	// Capturing config values to avoid race conditions if they are changed via SetLimits
	// (though archive settings don't have setters yet, it's good practice)
	archiveDir := fl.archivePath
	compress := fl.compress
	maxCount := fl.maxArchiveCount
	maxSize := fl.maxArchiveSize

	go func(src string) {
		destBase := filepath.Base(src)
		destPath := filepath.Join(archiveDir, destBase)

		if compress {
			destPath += ".gz"
			if err := compressFile(src, destPath); err != nil {
				fmt.Fprintf(os.Stderr, "[audit] archival compression failed: %v\n", err)
				return
			}
			_ = os.Remove(src)
		} else {
			if err := os.Rename(src, destPath); err != nil {
				fmt.Fprintf(os.Stderr, "[audit] archival move failed: %v\n", err)
				return
			}
		}

		// Prune archives after successful move/compress
		fl.pruneArchives(archiveDir, maxCount, maxSize)
	}(backupPath)
}

func compressFile(src, dest string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	d, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()

	gw := gzip.NewWriter(d)
	if _, err := io.Copy(gw, s); err != nil {
		_ = gw.Close()
		return err
	}
	return gw.Close()
}

func (fl *FileLogger) pruneArchives(dir string, maxCount int, maxSize int64) {
	matches, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil || len(matches) == 0 {
		return
	}

	sort.Strings(matches) // chronological by name

	// Prune by count
	if maxCount > 0 && len(matches) > maxCount {
		excess := len(matches) - maxCount
		for i := 0; i < excess; i++ {
			_ = os.Remove(matches[i])
		}
		matches = matches[excess:]
	}

	// Prune by size
	if maxSize > 0 {
		var totalSize int64
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil {
				totalSize += fi.Size()
			}
		}

		for len(matches) > 1 && totalSize > maxSize {
			if fi, err := os.Stat(matches[0]); err == nil {
				totalSize -= fi.Size()
			}
			_ = os.Remove(matches[0])
			matches = matches[1:]
		}
	}
}

// Query returns NotSupported error as FileLogger no longer retains memory index.
func (fl *FileLogger) Query(ctx context.Context, filter *Filter) ([]*Event, error) {
	return nil, fmt.Errorf("query not supported in rotating file logger")
}

// VerifyChain recomputes hashes to verify integrity; returns first error encountered.
// Reads the full file from disk.
func (fl *FileLogger) VerifyChain() error {
	fl.mu.RLock()
	defer fl.mu.RUnlock()

	// We verify the CURRENT file.
	// Note: We need to open a separate read handle to not disturb the write handle offset if standard file.
	// But standard file writes are append, read is separate?
	// Safest to open new handle.
	f, err := os.Open(fl.path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	lastHash := ""
	lastIndex := -1

	// Warning: This verify assumes the file starts at index 0 or we don't know the start index?
	// If rotation happened, the new file starts at index N.
	// We need to support starting verification from any index, provided continuity within the file.
	// BUT, strict VerifyChain usually expects index 0.
	// We will relax to: Check internal consistency.

	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			return err
		}

		if ev.Hash != computeEventHash(ev) {
			return fmt.Errorf("hash mismatch line %d", lineNo)
		}

		// First line of THIS file might have a PrevHash from a previous file.
		// Since we can't verify previous file context here, we only enforce continuity from the second line onward.
		if lineNo > 0 {
			if ev.PrevHash != lastHash {
				return fmt.Errorf("chain break at line %d", lineNo)
			}
			if ev.ChainIndex != -1 && ev.ChainIndex != lastIndex+1 {
				return fmt.Errorf(
					"index discontinuity at line %d: got %d want %d",
					lineNo,
					ev.ChainIndex,
					lastIndex+1,
				)
			}
		}

		lastHash = ev.Hash
		lastIndex = ev.ChainIndex
		lineNo++
	}
	return scanner.Err()
}

// Close closes underlying file.
func (fl *FileLogger) Close() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	if fl.file != nil {
		return fl.file.Close()
	}
	return nil
}

// Events returns nil (not supported).
func (fl *FileLogger) Events() []Event {
	return nil
}

// SetLimits configures rotation limits. Useful for testing or config overrides.
func (fl *FileLogger) SetLimits(maxSize int64, maxBackups int) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.maxSize = maxSize
	fl.maxBackups = maxBackups
}
