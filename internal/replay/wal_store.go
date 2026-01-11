package replay

import (
	"os"
	"sync"
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
}

// NewWALStore creates a new WALStore backed by the given file path.
func NewWALStore(path string) (*WALStore, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600) //nolint:gosec // path is controlled
	if err != nil {
		return nil, err
	}
	return &WALStore{file: f}, nil
}

// AppendRecord writes a WALRecord to the log.
func (w *WALStore) AppendRecord(rec WALRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	// TODO: Serialize WALRecord (e.g., JSON, binary)
	// _, err := w.file.Write(...)
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
