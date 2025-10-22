package policy

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore is a simple append-only JSON file backed implementation of Store.
// Layout: a single JSON array of Bundle objects. On append we rewrite the file (ok for demo scale).
// Future optimization could switch to newline-delimited JSON with periodic compaction.
type FileStore struct {
	mu     sync.RWMutex
	path   string
	reg    *Registry
	loaded bool
	perm   os.FileMode
}

// NewFileStore creates (or loads) a file-backed store at the given path.
// If the file does not exist it is created with an empty bundle chain.
func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{path: path, reg: NewRegistry(), perm: 0o600}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (f *FileStore) load() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.loaded {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		// initialize empty file
		if werr := os.WriteFile(f.path, []byte("[]"), f.perm); werr != nil {
			return werr
		}
		f.loaded = true
		return nil
	} else if err != nil {
		return err
	}
	if len(data) == 0 {
		data = []byte("[]")
	}
	var bundles []Bundle
	if err := json.Unmarshal(data, &bundles); err != nil {
		return err
	}
	// Rebuild registry (hashes are trusted but we verify chain after load)
	f.reg.bundles = append(f.reg.bundles, bundles...)
	if err := f.reg.VerifyChain(); err != nil {
		return err
	}
	f.loaded = true
	return nil
}

func (f *FileStore) persist() error {
	// Write to temp file then atomic rename
	tmp := f.path + ".tmp"
	enc, err := json.MarshalIndent(f.reg.bundles, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, enc, f.perm); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

// AppendBundle appends and persists a new bundle.
func (f *FileStore) AppendBundle(b Bundle) (Bundle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.loaded {
		if err := f.load(); err != nil {
			return Bundle{}, err
		}
	}
	// Preserve supplied Created if present (for tests) else set now
	if b.Created.IsZero() {
		b.Created = time.Now().UTC()
	}
	if len(f.reg.bundles) > 0 {
		b.PrevHash = f.reg.bundles[len(f.reg.bundles)-1].Hash
	}
	h, err := hashBundle(b)
	if err != nil {
		return Bundle{}, err
	}
	b.Hash = h
	f.reg.bundles = append(f.reg.bundles, b)
	if err := f.persist(); err != nil {
		return Bundle{}, err
	}
	return b, nil
}

func (f *FileStore) Head() *Bundle {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if len(f.reg.bundles) == 0 {
		return nil
	}
	b := f.reg.bundles[len(f.reg.bundles)-1]
	return &b
}

func (f *FileStore) GetByHash(hash string) *Bundle {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for i := range f.reg.bundles {
		if f.reg.bundles[i].Hash == hash {
			b := f.reg.bundles[i]
			return &b
		}
	}
	return nil
}

func (f *FileStore) List(offset, limit int) ([]Bundle, int) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	total := len(f.reg.bundles)
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []Bundle{}, total
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	out := make([]Bundle, end-offset)
	copy(out, f.reg.bundles[offset:end])
	return out, total
}

func (f *FileStore) ChainHashes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	hashes := make([]string, len(f.reg.bundles))
	for i, b := range f.reg.bundles {
		hashes[i] = b.Hash
	}
	return hashes
}

func (f *FileStore) VerifyChain() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.reg.VerifyChain()
}
func (f *FileStore) Registry() *Registry { return f.reg }

// Export writes bundles to the provided writer (pretty JSON) - helper for docs or debugging.
func (f *FileStore) Export(w io.Writer) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(f.reg.bundles)
}
