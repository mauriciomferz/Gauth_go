// Package crypto implements EdDSA key management (rotation, persistence and optional
// immutable audit logging) for the GAuth demo. It is intentionally minimal and not a
// production-grade HSM integration.
package crypto

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	ledger "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/ledger"
)

// Key represents a signing key (Ed25519) with metadata.
type Key struct {
	ID        string             `json:"kid"`
	CreatedAt time.Time          `json:"created_at"`
	ExpiresAt time.Time          `json:"expires_at"`
	Private   ed25519.PrivateKey `json:"-"`
	Public    ed25519.PublicKey  `json:"public"`
	Alg       string             `json:"alg"` // EdDSA
	Use       string             `json:"use"` // sig
}

// Manager manages active + previous keys (simple in-memory rotation).
type Manager struct {
	mu          sync.RWMutex
	active      *Key
	history     []*Key // previous keys retained until expiry
	ttl         time.Duration
	persistPath string        // optional persistence path for key material (private keys base64 encoded)
	stopCh      chan struct{} // for scheduler shutdown (not yet exposed)
	ledgerStore ledger.Store  // optional immutable audit ledger for rotation events
}

// OnKeyRotated is an optional global callback fired after a successful rotation.
// Arguments: previous key (may be nil on first rotation), new active key.
// Intended usage: downstream transparency modules auto-sign tree heads when keys change.
// Panics inside the callback are recovered and logged.
var OnKeyRotated func(prev, curr *Key)

// NewManager constructs a manager with given key lifetime.
// NewManager constructs a manager with given key lifetime.
// If GAUTH_EDDSA_PERSIST_PATH is set, keys will be loaded from disk (if file exists)
// and future rotations will be saved. Persistence format (JSON):
//
//	{
//	  "ttl_hours": 24,
//	  "active": {"kid":"...","created_at":"RFC3339","expires_at":"RFC3339","private_b64":"...","public_b64":"..."},
//	  "history": [ { ... same fields ... } ]
//	}
//
// Only non-expired history keys are kept on load. Missing or corrupt file results in fresh key generation.
func NewManager(ttl time.Duration) (*Manager, error) {
	m := &Manager{ttl: ttl, stopCh: make(chan struct{})}
	if p := os.Getenv("GAUTH_EDDSA_PERSIST_PATH"); p != "" {
		// Expand ~ and relative path
		if p[0] == '~' {
			if home, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(home, p[1:])
			}
		}
		m.persistPath = p
		if _, err := os.Stat(p); err == nil {
			if err := m.loadFromDisk(); err != nil {
				// Continue with fresh generation; log via stderr (no logger yet)
				// NOTE: we do not return error to avoid startup failure
				fmt.Fprintf(os.Stderr, "[crypto] load persistence failed: %v\n", err)
			} else {
				return m, nil // loaded successfully
			}
		}
	}
	// Optional ledger integration for rotation audit events (hash-chained & persistent)
	if lp := os.Getenv("GAUTH_EDDSA_ROTATION_LEDGER_PATH"); lp != "" {
		if abs, err := filepath.Abs(lp); err == nil {
			if st, err := ledger.NewBoltStore(abs); err == nil {
				m.ledgerStore = st
			} else {
				fmt.Fprintf(os.Stderr, "[crypto] rotation ledger init failed: %v\n", err)
			}
		}
	}
	if err := m.rotateLocked(); err != nil {
		return nil, err
	}
	// Save initial key if persistence enabled
	if err := m.saveLocked(); err != nil {
		fmt.Fprintf(os.Stderr, "[crypto] initial save failed: %v\n", err)
	}
	// Optional auto-rotation scheduler: if GAUTH_EDDSA_AUTO_ROTATE=1 run background ticker
	if os.Getenv("GAUTH_EDDSA_AUTO_ROTATE") == "1" {
		interval := m.ttl / 2
		if interval < time.Minute {
			interval = time.Minute
		}
		go m.runScheduler(interval)
	}
	// Fire initial callback (prev nil) outside lock
	if OnKeyRotated != nil && m.active != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "[crypto] OnKeyRotated panic: %v\n%s\n", r, debug.Stack())
				}
			}()
			OnKeyRotated(nil, m.active)
		}()
	}
	return m, nil
}

// rotateLocked generates a new Ed25519 key pair. Caller must hold write lock.
func (m *Manager) rotateLocked() error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	id := base64.RawURLEncoding.EncodeToString(pub[:8]) // short kid derivation (placeholder)
	k := &Key{ID: id, CreatedAt: now, ExpiresAt: now.Add(m.ttl), Private: priv, Public: pub, Alg: "EdDSA", Use: "sig"}
	prev := m.active
	if prev != nil {
		m.history = append(m.history, prev)
	}
	m.active = k
	// prune expired history
	var kept []*Key
	for _, hk := range m.history {
		if time.Now().Before(hk.ExpiresAt) {
			kept = append(kept, hk)
		}
	}
	m.history = kept
	// Persist after rotation if enabled
	if err := m.saveLocked(); err != nil {
		fmt.Fprintf(os.Stderr, "[crypto] rotate persistence failed: %v\n", err)
	}
	m.emitRotationLog(prev, k)
	m.appendLedgerRotation(prev, k)
	return nil
}

// Rotate generates and sets a new active key.
func (m *Manager) Rotate() (*Key, error) {
	m.mu.Lock()
	prev := m.active
	if err := m.rotateLocked(); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	curr := m.active
	m.mu.Unlock()
	if OnKeyRotated != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "[crypto] OnKeyRotated panic: %v\n%s\n", r, debug.Stack())
				}
			}()
			OnKeyRotated(prev, curr)
		}()
	}
	return curr, nil
}

// runScheduler periodically rotates keys at given interval until stopCh closed.
func (m *Manager) runScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Perform rotation; log errors (will retry next tick)
			if _, err := m.Rotate(); err != nil {
				fmt.Fprintf(os.Stderr, "[crypto] scheduled rotation failed: %v\n", err)
			}
		case <-m.stopCh:
			return
		}
	}
}

// Stop halts the background rotation scheduler if running.
func (m *Manager) Stop() {
	if m == nil || m.stopCh == nil {
		return
	}
	select {
	case <-m.stopCh: // already closed
		return
	default:
		close(m.stopCh)
	}
}

// emitRotationLog writes a JSON line record describing the rotation if GAUTH_EDDSA_ROTATION_LOG is set.
// Fields: ts, event, prev_kid, new_kid, ttl_hours, history_size
func (m *Manager) emitRotationLog(prev, curr *Key) {
	path := os.Getenv("GAUTH_EDDSA_ROTATION_LOG")
	if path == "" || curr == nil {
		return
	}
	// Load last hash (prev_hash) by reading final non-empty line of log file (if exists)
	prevHash := ""
	if b, err := os.ReadFile(path); err == nil {
		lines := bytes.Split(b, []byte{'\n'})
		for i := len(lines) - 1; i >= 0; i-- {
			line := bytes.TrimSpace(lines[i])
			if len(line) == 0 {
				continue
			}
			var lr map[string]any
			if json.Unmarshal(line, &lr) == nil {
				if h, ok := lr["hash"].(string); ok {
					prevHash = h
				}
			}
			break
		}
	}
	rec := map[string]any{
		"ts":           time.Now().UTC().Format(time.RFC3339Nano),
		"event":        "eddsa_key_rotated",
		"new_kid":      curr.ID,
		"ttl_hours":    int(m.ttl.Hours()),
		"history_size": len(m.history),
	}
	if prev != nil {
		rec["prev_kid"] = prev.ID
	}
	if prevHash != "" {
		rec["prev_hash"] = prevHash
	}
	// Compute hash over canonical JSON of record without hash field; add after computation.
	tmpData, err := json.Marshal(rec)
	if err != nil {
		return
	}
	h := sha256.Sum256(tmpData)
	rec["hash"] = base64.RawURLEncoding.EncodeToString(h[:])
	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gofumpt // spacing acceptable; functional logic prioritized
	if err != nil {
		fmt.Fprintf(os.Stderr, "[crypto] rotation log open error: %v\n", err)
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "[crypto] rotation log close error: %v\n", cerr)
		}
	}()
	if _, err := f.Write(append(data, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "[crypto] rotation log write error: %v\n", err)
	}
}

// appendLedgerRotation writes a rotation event to the immutable ledger if configured.
// Entry Type: key_rotation; Subject: ed25519; Object: new key kid.
// Metadata includes ttl_hours, history_size and prev_kid (if exists).
func (m *Manager) appendLedgerRotation(prev, curr *Key) {
	if m.ledgerStore == nil || curr == nil {
		return
	}
	meta := map[string]any{
		"ttl_hours":    int(m.ttl.Hours()),
		"history_size": len(m.history),
	}
	if prev != nil {
		meta["prev_kid"] = prev.ID
	}
	// Use deterministic ID (timestamp + kid) to allow lookup; collisions practically impossible.
	id := fmt.Sprintf("rot-%d-%s", time.Now().UTC().UnixNano(), curr.ID)
	if err := m.ledgerStore.Append(context.Background(), &ledger.Entry{
		ID:       id,
		TS:       time.Now().UTC(),
		Type:     "key_rotation",
		Subject:  "ed25519",
		Object:   curr.ID,
		Metadata: meta,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "[crypto] ledger append error: %v\n", err)
	}
}

// Active returns current active key.
func (m *Manager) Active() *Key {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// FindByID returns key by kid searching active + history.
func (m *Manager) FindByID(id string) *Key {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.active != nil && m.active.ID == id {
		return m.active
	}
	for _, k := range m.history {
		if k.ID == id {
			return k
		}
	}
	return nil
}

// ListCurrent returns slice containing active + non-expired history.
func (m *Manager) ListCurrent() []*Key {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []*Key{}
	if m.active != nil {
		out = append(out, m.active)
	}
	out = append(out, m.history...)
	return out
}

// ValidateSignature verifies an Ed25519 signature for given kid.
func (m *Manager) ValidateSignature(kid string, payload, sig []byte) error {
	k := m.FindByID(kid)
	if k == nil {
		return errors.New("unknown_kid")
	}
	if !ed25519.Verify(k.Public, payload, sig) {
		return errors.New("invalid_signature")
	}
	return nil
}

// ImportPublic inserts a public-only key (no private material) for verification purposes.
// If a key with same kid exists it is replaced only if existing is expired.
func (m *Manager) ImportPublic(kid string, pub []byte, expires time.Time) {
	if kid == "" || len(pub) != ed25519.PublicKeySize {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	k := &Key{ID: kid, CreatedAt: time.Now().UTC(), ExpiresAt: expires, Public: ed25519.PublicKey(pub), Alg: "EdDSA", Use: "sig"}
	// If no active set, assign active. Otherwise append to history for lookup.
	if m.active == nil {
		m.active = k
		return
	}
	// Replace if same kid found in active or history (simplify by just appending; FindByID returns first match active then history).
	// Remove any existing with same kid from history.
	var kept []*Key
	for _, hk := range m.history {
		if hk.ID != kid {
			kept = append(kept, hk)
		}
	}
	// If active has same kid but expired, move it to history and set new active.
	if m.active.ID == kid {
		if time.Now().After(m.active.ExpiresAt) {
			m.history = kept
			m.active = k
			return
		}
		// If not expired, keep existing active and add new to history for redundancy.
		kept = append(kept, k)
		m.history = kept
		return
	}
	// Append new key
	m.history = append(kept, k)
}

// persistenceRecord models on-disk JSON.
type persistenceRecord struct {
	TTLHours int        `json:"ttl_hours"`
	Active   *diskKey   `json:"active"`
	History  []*diskKey `json:"history"`
}
type diskKey struct {
	ID         string `json:"kid"`
	CreatedAt  string `json:"created_at"`
	ExpiresAt  string `json:"expires_at"`
	PrivateB64 string `json:"private_b64"`
	PublicB64  string `json:"public_b64"`
	Alg        string `json:"alg"`
	Use        string `json:"use"`
}

// saveLocked writes current state to disk if persistPath set. Caller holds write lock.
func (m *Manager) saveLocked() error {
	if m.persistPath == "" {
		return nil
	}
	rec := persistenceRecord{TTLHours: int(m.ttl.Hours())}
	if m.active != nil {
		rec.Active = &diskKey{ID: m.active.ID, CreatedAt: m.active.CreatedAt.Format(time.RFC3339), ExpiresAt: m.active.ExpiresAt.Format(time.RFC3339), PrivateB64: base64.RawStdEncoding.EncodeToString(m.active.Private), PublicB64: base64.RawStdEncoding.EncodeToString(m.active.Public), Alg: m.active.Alg, Use: m.active.Use}
	}
	for _, hk := range m.history {
		rec.History = append(rec.History, &diskKey{ID: hk.ID, CreatedAt: hk.CreatedAt.Format(time.RFC3339), ExpiresAt: hk.ExpiresAt.Format(time.RFC3339), PrivateB64: base64.RawStdEncoding.EncodeToString(hk.Private), PublicB64: base64.RawStdEncoding.EncodeToString(hk.Public), Alg: hk.Alg, Use: hk.Use})
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.persistPath, data, 0o600) //nolint:gofumpt // permission literal acceptable; formatting stable
}

// loadFromDisk restores keys; caller sets persistPath first. Caller does NOT hold lock yet (used in constructor before rotation).
func (m *Manager) loadFromDisk() error {
	b, err := os.ReadFile(m.persistPath)
	if err != nil {
		return err
	}
	var rec persistenceRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return err
	}
	// Reconstruct keys
	var active *Key
	var history []*Key
	now := time.Now().UTC()
	parseKey := func(dk *diskKey) (*Key, error) {
		pubBytes, err := base64.RawStdEncoding.DecodeString(dk.PublicB64)
		if err != nil {
			return nil, err
		}
		privBytes, err := base64.RawStdEncoding.DecodeString(dk.PrivateB64)
		if err != nil {
			return nil, err
		}
		created, err := time.Parse(time.RFC3339, dk.CreatedAt)
		if err != nil {
			return nil, err
		}
		expires, err := time.Parse(time.RFC3339, dk.ExpiresAt)
		if err != nil {
			return nil, err
		}
		return &Key{ID: dk.ID, CreatedAt: created, ExpiresAt: expires, Private: ed25519.PrivateKey(privBytes), Public: ed25519.PublicKey(pubBytes), Alg: dk.Alg, Use: dk.Use}, nil
	}
	if rec.Active != nil {
		ak, err := parseKey(rec.Active)
		if err == nil && now.Before(ak.ExpiresAt) {
			active = ak
		}
	}
	for _, dk := range rec.History {
		if hk, err := parseKey(dk); err == nil && now.Before(hk.ExpiresAt) {
			history = append(history, hk)
		}
	}
	// Assign under lock
	m.mu.Lock()
	m.active = active
	m.history = history
	// restore ttl if file includes it and >0
	if rec.TTLHours > 0 {
		m.ttl = time.Duration(rec.TTLHours) * time.Hour
	}
	m.mu.Unlock()
	// If no active key (expired), rotate fresh one
	if m.Active() == nil {
		m.mu.Lock()
		if err := m.rotateLocked(); err != nil {
			fmt.Fprintf(os.Stderr, "[crypto] post-load rotate failed: %v\n", err)
		}
		m.mu.Unlock()
	}
	return nil
}
