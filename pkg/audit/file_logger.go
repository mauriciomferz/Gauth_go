package audit

// FileLogger provides append-only persistent audit logging with the same hash chain
// semantics as MemoryLogger. It stores one JSON Event per line and recomputes / verifies
// hashes on load. A truncate or tamper will surface as VerifyChain error.
// NOTE: Experimental; not for production durability or concurrency beyond single-process.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileLogger implements persistent audit logging.
type FileLogger struct {
	mu     sync.RWMutex
	path   string
	file   *os.File
	events []Event
	// lazyWrite flushes every append; batching could be added later.
}

// OpenFileLogger opens (or creates) an audit log file. If the file exists it is loaded.
func OpenFileLogger(path string) (*FileLogger, error) {
	if path == "" {
		return nil, errors.New("path required")
	}
	// #nosec G301
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	fl := &FileLogger{path: path, file: f, events: make([]Event, 0, 128)}
	if err2 := fl.load(); err2 != nil {
		f.Close()
		return nil, fmt.Errorf("load: %w", err)
	}
	// Reopen in append mode to avoid rewriting existing content.
	f.Close()
	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	fl.file = f
	return fl, nil
}

// load ingests existing file contents and validates hash chain.
func (fl *FileLogger) load() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.events = fl.events[:0]
	f, err := os.Open(fl.path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			return fmt.Errorf("unmarshal line %d: %w", lineNo, err)
		}
		// Recompute hash to detect tamper.
		expected := computeEventHash(ev)
		if ev.Hash != expected {
			return fmt.Errorf("hash mismatch line %d", lineNo)
		}
		// Chain linkage check.
		if ev.ChainIndex != lineNo {
			return fmt.Errorf("chain index mismatch line %d", lineNo)
		}
		if lineNo == 0 && ev.PrevHash != "" {
			return fmt.Errorf("first prev hash non-empty line %d", lineNo)
		}
		if lineNo > 0 && ev.PrevHash != fl.events[lineNo-1].Hash {
			return fmt.Errorf("prev hash mismatch line %d", lineNo)
		}
		fl.events = append(fl.events, ev)
		lineNo++
	}
	return scanner.Err()
}

// Log appends an event; accepts *Event similar to MemoryLogger.
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
	idx := len(fl.events)
	prev := ""
	if idx > 0 {
		prev = fl.events[idx-1].Hash
	}
	ev.ChainIndex = idx
	ev.PrevHash = prev
	ev.Hash = computeEventHash(*ev)
	fl.events = append(fl.events, *ev)
	// Marshal with error handling (previously ignored by errcheck). On marshal failure
	// we roll back the in-memory append to keep chain consistent.
	b, mErr := json.Marshal(ev)
	if mErr != nil {
		fl.events = fl.events[:len(fl.events)-1]
		fl.mu.Unlock()
		return fmt.Errorf("marshal audit event: %w", mErr)
	}
	// write line (append newline delimiter)
	if _, err := fl.file.Write(append(b, '\n')); err != nil {
		fl.mu.Unlock()
		return err
	}
	fl.mu.Unlock()
	return nil
}

// Query returns matching events (basic filters only - identical logic to MemoryLogger).
func (fl *FileLogger) Query(ctx context.Context, filter *Filter) ([]*Event, error) {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	var out []*Event
	for i := range fl.events {
		ev := &fl.events[i]
		if filter != nil {
			if len(filter.EventTypes) > 0 {
				match := false
				for _, t := range filter.EventTypes {
					if ev.Type == t {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}
			if filter.Subject != "" && ev.Subject != filter.Subject {
				continue
			}
			if filter.StartTime != nil && ev.Timestamp.Before(*filter.StartTime) {
				continue
			}
			if filter.EndTime != nil && ev.Timestamp.After(*filter.EndTime) {
				continue
			}
		}
		out = append(out, ev)
	}
	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(out) {
			out = out[filter.Offset:]
		}
		if filter.Limit > 0 && filter.Limit < len(out) {
			out = out[:filter.Limit]
		}
	}
	return out, nil
}

// VerifyChain recomputes hashes to verify integrity; returns first error encountered.
func (fl *FileLogger) VerifyChain() error {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	for i, ev := range fl.events {
		if ev.Hash != computeEventHash(ev) {
			return fmt.Errorf("hash mismatch index %d", i)
		}
		if i == 0 && ev.PrevHash != "" {
			return fmt.Errorf("first prev hash non-empty")
		}
		if i > 0 && ev.PrevHash != fl.events[i-1].Hash {
			return fmt.Errorf("broken link at %d", i)
		}
		if ev.ChainIndex != i {
			return fmt.Errorf("index field mismatch %d", i)
		}
	}
	return nil
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

// Events exposes a snapshot for tests.
func (fl *FileLogger) Events() []Event {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	out := make([]Event, len(fl.events))
	copy(out, fl.events)
	return out
}
