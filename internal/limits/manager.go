// Package limits implements a lightweight persistent counter store used for
// demo rate / issuance tracking with optional periodic JSON snapshotting.
package limits

// Manager provides lifecycle management for the persistent limits Store including
// environment-based initialization and periodic persistence. It exposes a singleton
// style API for convenience while keeping explicit start/stop semantics for graceful
// shutdown.
//
// Environment variables:
//   GAUTH_LIMITS_PERSIST_PATH           override on-disk JSON path (default tmp/limits_counters.json)
//   GAUTH_LIMITS_PERSIST_INTERVAL_SEC   persistence interval in seconds (default 60)
//   GAUTH_LIMITS_DISABLE_PERSIST        when set to "1" disables background persistence entirely
//
// Typical usage:
//   lm, err := limits.InitFromEnv(); if err != nil { ... }
//   defer lm.Close()
//   limits.Inc("tokens_issued_total", 1)
//
// NOTE: The Store itself is concurrency safe; Manager coordinates periodic Persist calls.

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/config"
)

// Manager coordinates a Store and a persistence ticker.
type Manager struct {
	store      *Store
	ticker     *time.Ticker
	done       chan struct{}
	wg         sync.WaitGroup
	enabled    bool
	onSnapshot func(map[string]any) // optional callback invoked each successful periodic snapshot persist
}

var (
	mgrMu     sync.RWMutex
	globalMgr *Manager
)

// InitFromEnv initializes the global limits manager from environment variables.
// Safe to call multiple times; subsequent calls return existing manager.
func InitFromEnv() (*Manager, error) {
	mgrMu.Lock()
	defer mgrMu.Unlock()
	if globalMgr != nil {
		return globalMgr, nil
	}

	path := config.Get("GAUTH_LIMITS_PERSIST_PATH", DefaultPath())
	intervalSec := config.Get("GAUTH_LIMITS_PERSIST_INTERVAL_SEC", "60")
	disable := config.Get("GAUTH_LIMITS_DISABLE_PERSIST", "") == "1"

	// Ensure directory exists when persistence enabled.
	if !disable {
		if dir := filepath.Dir(path); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("limits: mkdir %s: %w", dir, err)
			}
		}
	}

	st, err := New(path)
	if err != nil {
		return nil, err
	}
	m := &Manager{store: st, enabled: !disable}

	if !disable {
		// parse interval
		dur := 60 * time.Second
		if v, err := time.ParseDuration(intervalSec + "s"); err == nil && v > 0 {
			dur = v
		}
		m.ticker = time.NewTicker(dur)
		m.done = make(chan struct{})
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			for {
				select {
				case <-m.ticker.C:
					// best-effort persist; log only on error (stdout for now)
					if err := m.store.Persist(); err != nil {
						fmt.Fprintf(os.Stderr, "[limits] persist error: %v\n", err)
					} else if m.onSnapshot != nil {
						// Build ledger-style entry and invoke callback
						entry := m.store.LedgerEntry()
						entry["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
						// Non-blocking safeguard via recover
						func() { defer func() { _ = recover() }(); m.onSnapshot(entry) }()
					}
				case <-m.done:
					return
				}
			}
		}()
	}
	globalMgr = m
	return m, nil
}

// GetManager returns the global Manager (nil if not initialized).
func GetManager() *Manager { mgrMu.RLock(); defer mgrMu.RUnlock(); return globalMgr }

// Store returns the underlying limits Store (nil if manager not initialized).
func (m *Manager) Store() *Store {
	if m == nil {
		return nil
	}
	return m.store
}

// SetSnapshotCallback registers a callback invoked after each successful periodic persistence with the ledger entry map.
// Passing nil removes any existing callback.
func (m *Manager) SetSnapshotCallback(cb func(map[string]any)) {
	if m != nil {
		m.onSnapshot = cb
	}
}

// Inc convenience increments a counter using the global manager; returns new value or 0 if manager nil.
func Inc(name string, delta uint64) uint64 {
	mgrMu.RLock()
	m := globalMgr
	mgrMu.RUnlock()
	if m == nil || m.store == nil {
		return 0
	}
	return m.store.Inc(name, delta)
}

// Get returns current value via global manager (0 if absent or manager nil).
func Get(name string) uint64 {
	mgrMu.RLock()
	m := globalMgr
	mgrMu.RUnlock()
	if m == nil || m.store == nil {
		return 0
	}
	return m.store.Get(name)
}

// Snapshot exposes a copy of counters via global manager.
func Snapshot() map[string]uint64 {
	mgrMu.RLock()
	m := globalMgr
	mgrMu.RUnlock()
	if m == nil || m.store == nil {
		return map[string]uint64{}
	}
	return m.store.Snapshot()
}

// Persist forces an immediate persistence (no-op if disabled or manager nil).
func Persist() error {
	mgrMu.RLock()
	m := globalMgr
	mgrMu.RUnlock()
	if m == nil || m.store == nil || !m.enabled {
		return nil
	}
	return m.store.Persist()
}

// Close stops ticker and performs a final persist (best-effort).
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	if m.ticker != nil {
		m.ticker.Stop()
	}
	if m.done != nil {
		close(m.done)
	}
	m.wg.Wait()
	if m.enabled && m.store != nil {
		if err := m.store.Persist(); err != nil {
			return err
		}
	}
	return nil
}
