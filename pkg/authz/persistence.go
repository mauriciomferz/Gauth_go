package authz

// Basic persistence interfaces and a lightweight JSON file store for policies.
// This is an initial minimal implementation to support loading and reloading
// policies from disk; advanced indexing, validation, and versioning will follow.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

// PolicyStore abstracts retrieval of policies. Future implementations may include
// database-backed stores, remote APIs, or hierarchical overlays.
type PolicyStore interface {
	// Load returns the full current set of policies from the backing store.
	Load() ([]Policy, error)
	// LastModified returns a monotonic-ish timestamp for change detection.
	LastModified() time.Time
}

// FilePolicyStore implements PolicyStore backed by a single JSON file.
// Format: JSON array of Policy objects.
// Example:
// [ {"id":"p1","subject":"alice","resource":"vault","actions":["read"],"effect":"allow"} ]
type FilePolicyStore struct {
	path         string
	mu           sync.RWMutex
	lastModified time.Time
}

// NewFilePolicyStore creates a new file-backed policy store for the provided path.
func NewFilePolicyStore(path string) (*FilePolicyStore, error) {
	st := &FilePolicyStore{path: path}
	if err := st.refreshModTime(); err != nil {
		return nil, err
	}
	return st, nil
}

// Load reads and unmarshals policies from the JSON file.
func (f *FilePolicyStore) Load() ([]Policy, error) {
	f.mu.RLock()
	path := f.path
	f.mu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Policy{}, nil // treat missing file as empty set
		}
		return nil, fmt.Errorf("read policy file: %w", err)
	}
	if len(data) == 0 {
		return []Policy{}, nil
	}
	var policies []Policy
	if err := json.Unmarshal(data, &policies); err != nil {
		return nil, fmt.Errorf("unmarshal policies: %w", err)
	}
	return policies, nil
}

// LastModified reports last known file mod time (cached).
func (f *FilePolicyStore) LastModified() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lastModified
}

// refreshModTime updates the cached modification time.
func (f *FilePolicyStore) refreshModTime() error {
	info, err := os.Stat(f.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			f.mu.Lock()
			f.lastModified = time.Time{}
			f.mu.Unlock()
			return nil
		}
		return err
	}
	f.mu.Lock()
	f.lastModified = info.ModTime()
	f.mu.Unlock()
	return nil
}

// PersistentAuthorizer wraps MemoryAuthorizer and reloads policies when the underlying store changes.
// Simple polling approach; can be replaced with fsnotify for efficiency later.
type PersistentAuthorizer struct {
	*MemoryAuthorizer
	store        PolicyStore
	pollInterval time.Duration
	lastSeenMod  time.Time
	stopCh       chan struct{}
	watcher      *fsnotify.Watcher
	mu           sync.RWMutex // protects watchErr, lastAdded, lastRemoved
	watchErr     error
	lastAdded    []string
	lastRemoved  []string
}

// NewPersistentAuthorizer constructs a PersistentAuthorizer with given store.
// It performs an initial load. Use Start() to begin polling for changes.
func NewPersistentAuthorizer(store PolicyStore, pollInterval time.Duration) (*PersistentAuthorizer, error) {
	ma := NewMemoryAuthorizer()
	pa := &PersistentAuthorizer{MemoryAuthorizer: ma, store: store, pollInterval: pollInterval, stopCh: make(chan struct{})}
	if err := pa.reload(); err != nil {
		return nil, err
	}
	pa.lastSeenMod = store.LastModified()
	return pa, nil
}

// Start begins a background polling loop to detect changes and reload policies.
func (p *PersistentAuthorizer) Start() {
	go func() {
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Refresh mod time for file-backed store before comparison
				if fps, ok := p.store.(*FilePolicyStore); ok {
					if err := fps.refreshModTime(); err != nil {
						fmt.Fprintf(os.Stderr, "persistence: refreshModTime error: %v\n", err) // nolint:errcheck
					}
				}
				mod := p.store.LastModified()
				if mod.After(p.lastSeenMod) { // change detected
					if err := p.reload(); err == nil {
						p.lastSeenMod = mod
						atomic.AddUint64(&p.MemoryAuthorizer.metricReloads, 1)
					} else {
						fmt.Fprintf(os.Stderr, "persistence: reload error: %v\n", err) // nolint:errcheck
					}
				}
			case <-p.stopCh:
				return
			}
		}
	}()
}

// Stop terminates the polling loop.
func (p *PersistentAuthorizer) Stop() { close(p.stopCh) }

// StartWatch initializes a filesystem watcher for the underlying store if file-based.
// Falls back silently if watcher cannot be created; polling continues.
func (p *PersistentAuthorizer) StartWatch() error {
	fps, ok := p.store.(*FilePolicyStore)
	if !ok {
		return fmt.Errorf("watch only supported for FilePolicyStore")
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		p.mu.Lock()
		p.watchErr = err
		p.mu.Unlock()
		return err
	}
	p.watcher = w
	path := fps.path
	if err := w.Add(path); err != nil {
		p.mu.Lock()
		p.watchErr = err
		p.mu.Unlock()
		if cerr := w.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "watch: close error: %v\n", cerr) // nolint:errcheck
		}
		return err
	}
	go p.watchLoop(path)
	return nil
}

func (p *PersistentAuthorizer) watchLoop(path string) {
	for {
		select {
		case <-p.stopCh:
			if p.watcher != nil {
				if cerr := p.watcher.Close(); cerr != nil {
					fmt.Fprintf(os.Stderr, "watch: close error: %v\n", cerr) // nolint:errcheck
				}
			}
			return
		case event, ok := <-p.watcher.Events:
			if !ok {
				return // channel closed
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// reload on write/create (handles some editor rename patterns)
				if err := p.reload(); err != nil {
					fmt.Fprintf(os.Stderr, "watch reload error: %v\n", err) // nolint:errcheck
					p.mu.Lock()
					p.watchErr = err
					p.mu.Unlock()
				} else {
					atomic.AddUint64(&p.MemoryAuthorizer.metricReloads, 1)
				}
			}
		case err, ok := <-p.watcher.Errors:
			if !ok {
				return
			}
			p.watchErr = err
			return
		}
	}
}

// reload replaces in-memory policies with those from the store.
func (p *PersistentAuthorizer) reload() error {
	policies, err := p.store.Load()
	if err != nil {
		return err
	}
	// compute diff
	oldMap := make(map[string]struct{})
	p.policiesMu.RLock()
	oldPolicies := append([]Policy(nil), p.policies...) // copy slice for diff computation
	p.policiesMu.RUnlock()
	for _, op := range oldPolicies {
		oldMap[op.ID] = struct{}{}
	}
	newMap := make(map[string]Policy)
	for _, np := range policies {
		newMap[np.ID] = np
	}
	// pre-validate regex patterns in conditions
	for _, pol := range policies {
		for _, cond := range pol.Conditions {
			if cond.Operator == "regex" {
				for _, pattern := range cond.Values {
					p.regexMu.RLock()
					_, ok := p.regexCache[pattern]
					p.regexMu.RUnlock()
					if ok {
						continue
					}
					// attempt compile (but do not store here; caching handled on demand)
					if _, err := regexp.Compile(pattern); err != nil {
						atomic.AddUint64(&p.metricRegexCompileErrors, 1)
					} else {
						// successful validation counts as compile metric even if not cached yet
						atomic.AddUint64(&p.metricRegexCompiles, 1)
					}
				}
			}
		}
	}
	added := make([]string, 0)
	removed := make([]string, 0)
	for id := range newMap {
		if _, ok := oldMap[id]; !ok {
			added = append(added, id)
		}
	}
	for id := range oldMap {
		if _, ok := newMap[id]; !ok {
			removed = append(removed, id)
		}
	}
	// Replace slice with write lock to prevent concurrent reads during reload
	p.policiesMu.Lock()
	p.policies = make([]Policy, 0, len(policies))
	p.policies = append(p.policies, policies...)
	p.policiesMu.Unlock()
	p.mu.Lock()
	p.lastAdded = added
	p.lastRemoved = removed
	p.mu.Unlock()
	return nil
}

// LastDiff returns the most recent added and removed policy IDs from a reload.
func (p *PersistentAuthorizer) LastDiff() (added, removed []string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]string{}, p.lastAdded...), append([]string{}, p.lastRemoved...)
}

// WatchErr returns the last error from the watch loop (for testing).
func (p *PersistentAuthorizer) WatchErr() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.watchErr
}

// PolicyCount returns the number of policies (for testing).
func (p *PersistentAuthorizer) PolicyCount() int {
	p.policiesMu.RLock()
	defer p.policiesMu.RUnlock()
	return len(p.policies)
}
