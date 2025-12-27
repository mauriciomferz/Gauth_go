package replay

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WALRecord represents a write-ahead log entry for replay store operations.
type WALRecord struct {
	Op    string // e.g., "Put", "Delete"
	Key   []byte
	Value []byte
	TS    int64 // Unix timestamp
}

// WALStore provides write-ahead logging for replay store durability.
type WALStore struct {
	file *os.File
	mu   sync.Mutex
	path string
}

// NewWALStore creates a new WALStore backed by the given file path.
func NewWALStore(path string) (*WALStore, error) {
	// #nosec G301
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &WALStore{file: f, path: path}, nil
}

// AppendRecord writes a WALRecord to the log.
func (w *WALStore) AppendRecord(rec WALRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.file.Write(data)
	return err
}

// Recover replays all WALRecords from the log file.
// Recover replays all WALRecords from the log file, aborting on I/O or apply errors.
// It is tolerant of individual malformed lines (they are skipped). For detailed
// stats use RecoverWithStats.
func (w *WALStore) Recover(apply func(WALRecord) error) error {
	_, _, err := w.RecoverWithStats(apply)
	return err
}

// RecoverWithStats replays all WALRecords, returning number applied and skipped (malformed lines).
// Malformed lines are ignored; only I/O or apply errors abort recovery.
func (w *WALStore) RecoverWithStats(apply func(WALRecord) error) (applied int, skipped int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, 0, err
	}
	scanner := bufio.NewScanner(w.file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 { // skip empty
			continue
		}
		var rec WALRecord
		if uErr := json.Unmarshal(line, &rec); uErr != nil {
			skipped++
			continue
		}
		if aErr := apply(rec); aErr != nil {
			return applied, skipped, aErr
		}
		applied++
	}
	if sErr := scanner.Err(); sErr != nil && sErr != io.EOF {
		return applied, skipped, sErr
	}
	return applied, skipped, nil
}

// Snapshot creates a point-in-time snapshot of the replay store.
// Snapshot writes a snapshot of provided key->timestamp map (seconds) to a companion file.
// Caller supplies state to avoid WALStore knowing higher-level semantics.
func (w *WALStore) Snapshot(state map[string]time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return errors.New("wal closed")
	}
	tmp := w.path + ".snapshot.tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	// Serialize as array of objects for easy future extension.
	type snapEntry struct {
		Key string `json:"key"`
		TS  int64  `json:"ts"`
	}
	arr := make([]snapEntry, 0, len(state))
	for k, v := range state {
		arr = append(arr, snapEntry{Key: k, TS: v.Unix()})
	}
	if err := enc.Encode(arr); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, w.path+".snapshot")
}

// Rotate truncates the WAL file (after flush) returning it to empty.
func (w *WALStore) Rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return errors.New("wal closed")
	}
	// Close existing file handle
	if err := w.file.Close(); err != nil {
		return err
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	w.file = f
	return nil
}

// Path returns underlying WAL path.
func (w *WALStore) Path() string { return w.path }

// Close closes the WAL file.
func (w *WALStore) Close() error {
	return w.file.Close()
}
