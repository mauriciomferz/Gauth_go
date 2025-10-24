package replay

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

// WALRecord represents a write-ahead log entry for replay store operations.
type WALRecord struct {
	Op      string // e.g., "Put", "Delete"
	Key     []byte
	Value   []byte
	TS      int64  // Unix timestamp
}

// WALStore provides write-ahead logging for replay store durability.
type WALStore struct {
	file   *os.File
	mu     sync.Mutex
}

// NewWALStore creates a new WALStore backed by the given file path.
func NewWALStore(path string) (*WALStore, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	return &WALStore{file: f}, nil
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
func (w *WALStore) Recover(apply func(WALRecord) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(w.file)
	for {
		var rec WALRecord
		if err := dec.Decode(&rec); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := apply(rec); err != nil {
			return err
		}
	}
	return nil
}

// Snapshot creates a point-in-time snapshot of the replay store.
func (w *WALStore) Snapshot() ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// TODO: Implement snapshot logic
	return nil, nil
}

// Close closes the WAL file.
func (w *WALStore) Close() error {
	return w.file.Close()
}
