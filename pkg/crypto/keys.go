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
	"sort"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/anchor"
	"github.com/mauriciomferz/Gauth_go/internal/tracing"
	ledger "github.com/mauriciomferz/Gauth_go/pkg/ledger"
)

// Key represents a signing key (Ed25519) with metadata.
type Key struct {
	ID              string             `json:"kid"`
	CreatedAt       time.Time          `json:"created_at"`
	ExpiresAt       time.Time          `json:"expires_at"`
	DeprecatedAfter time.Time          `json:"deprecated_after,omitempty"` // RFC0115 deprecation warning timestamp (recommended: 80% of TTL)
	SunsetAfter     time.Time          `json:"sunset_after,omitempty"`     // RFC0115 hard cutoff timestamp (same as ExpiresAt)
	Private         ed25519.PrivateKey `json:"-"`
	Public          ed25519.PublicKey  `json:"public"`
	Alg             string             `json:"alg"` // EdDSA
	Use             string             `json:"use"` // sig
}

// Manager manages active + previous keys (simple in-memory rotation).
type Manager struct {
	mu          sync.RWMutex
	active      *Key
	history     []*Key // previous keys retained until expiry
	archived    []*Key // expired keys retained for verification
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

// RotationTracerProvider optionally enables RB9 tracing spans for key rotation operations.
// It is set by the server initialization when tracing is enabled. If nil, rotation operations
// proceed without span creation. Operation name: "rotation.perform".
var RotationTracerProvider *tracing.TracerProvider

// SetRotationTracerProvider assigns the tracer provider used for key rotation spans.
func SetRotationTracerProvider(tp *tracing.TracerProvider) { RotationTracerProvider = tp }

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
	fmt.Fprintf(os.Stderr, "[crypto] NewManager: ttl=%v persistPath=%s autoRotate=%s\n", ttl, m.persistPath, os.Getenv("GAUTH_EDDSA_AUTO_ROTATE"))
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
			// Check for generic External Anchor configuration
			anchorProvider := os.Getenv("GAUTH_ROTATION_ANCHOR_PROVIDER")
			anchorURL := os.Getenv("GAUTH_ROTATION_ANCHOR_URL")

			if anchorProvider != "" && anchorURL != "" {
				// Use ExternalAuditLedger
				fmt.Fprintf(os.Stderr, "[crypto] initializing external audit ledger with provider %s\n", anchorProvider)

				var provider anchor.Provider
				if anchorProvider == "rfc3161" {
					provider = ledger.NewRFC3161Provider(anchorURL)
				}

				if provider != nil {
					// Use same path for receipt store with .receipts extension if not specified
					receiptPath := abs + ".receipts"
					if rp := os.Getenv("GAUTH_ROTATION_ANCHOR_RECEIPT_PATH"); rp != "" {
						receiptPath = rp
					}

					if st, err := ledger.NewExternalAuditLedger(abs, provider, receiptPath, 60*time.Second); err == nil {
						m.ledgerStore = st
					} else {
						fmt.Fprintf(os.Stderr, "[crypto] external rotation ledger init failed: %v\n", err)
						// Fallback to basic bolt store
						if st, err := ledger.NewBoltStore(abs); err == nil {
							m.ledgerStore = st
						}
					}
				} else {
					// Invalid provider, fallback
					if st, err := ledger.NewBoltStore(abs); err == nil {
						m.ledgerStore = st
					}
				}
			} else {
				// Standard BoltStore
				if st, err := ledger.NewBoltStore(abs); err == nil {
					m.ledgerStore = st
				} else {
					fmt.Fprintf(os.Stderr, "[crypto] rotation ledger init failed: %v\n", err)
				}
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
		if envInterval := os.Getenv("GAUTH_EDDSA_ROTATE_INTERVAL"); envInterval != "" {
			if parsed, err := time.ParseDuration(envInterval); err == nil {
				interval = parsed
			} else {
				fmt.Fprintf(os.Stderr, "[crypto] invalid GAUTH_EDDSA_ROTATE_INTERVAL: %v\n", err)
			}
		} else if interval < time.Minute {
			interval = time.Minute
		}
		fmt.Fprintf(os.Stderr, "[crypto] NewManager: about to launch auto-rotation scheduler goroutine with interval %v\n", interval)
		go func() {
			fmt.Fprintf(os.Stderr, "[crypto] NewManager: auto-rotation scheduler goroutine launched\n")
			m.runScheduler(interval)
		}()
		fmt.Fprintf(os.Stderr, "[crypto] NewManager: goroutine launch line completed\n")
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
	expiresAt := now.Add(m.ttl)
	deprecatedAfter := now.Add(time.Duration(float64(m.ttl) * 0.8)) // Deprecation warning at 80% of TTL
	k := &Key{
		ID:              id,
		CreatedAt:       now,
		ExpiresAt:       expiresAt,
		DeprecatedAfter: deprecatedAfter,
		SunsetAfter:     expiresAt, // Sunset = hard expiration
		Private:         priv,
		Public:          pub,
		Alg:             "EdDSA",
		Use:             "sig",
	}
	prev := m.active
	if prev != nil {
		m.history = append(m.history, prev)
	}
	m.active = k
	// prune expired history to archive
	var kept []*Key
	for _, hk := range m.history {
		if time.Now().Before(hk.ExpiresAt) {
			kept = append(kept, hk)
		} else {
			m.archived = append(m.archived, hk)
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
	fmt.Fprintf(os.Stderr, "[crypto] Rotate called. Previous key: %v\n", prev)
	// RB9 tracing instrumentation: start span (sample logic handled by provider upstream)
	var span *tracing.Span
	if RotationTracerProvider != nil {
		ctx := context.Background()
		_, span = RotationTracerProvider.StartSpan(ctx, "rotation.perform")
		if span != nil {
			if prev != nil {
				span.SetTag("prev_kid", prev.ID)
			}
			span.SetTag("ttl_hours", int(m.ttl.Hours()))
			span.SetTag("history_size", len(m.history))
		}
	}
	if err := m.rotateLocked(); err != nil {
		if span != nil {
			span.SetTag("error", err.Error())
			span.End()
		}
		m.mu.Unlock()
		fmt.Fprintf(os.Stderr, "[crypto] Rotate failed: %v\n", err)
		return nil, err
	}
	curr := m.active
	m.mu.Unlock()
	fmt.Fprintf(os.Stderr, "[crypto] Rotate succeeded. New key: %v\n", curr)
	if span != nil {
		span.SetTag("new_kid", curr.ID)
		span.End()
	}
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
	fmt.Fprintf(os.Stderr, "[crypto] runScheduler started with interval %v\n", interval)
	tickCount := 0
	for {
		select {
		case <-ticker.C:
			tickCount++
			fmt.Fprintf(os.Stderr, "[crypto] runScheduler tick #%d: attempting rotation\n", tickCount)
			if _, err := m.Rotate(); err != nil {
				fmt.Fprintf(os.Stderr, "[crypto] scheduled rotation failed: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "[crypto] scheduled rotation succeeded\n")
			}
		case <-m.stopCh:
			fmt.Fprintf(os.Stderr, "[crypto] runScheduler stopped after %d ticks\n", tickCount)
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
	// Compute hash over canonical JSON of record without hash/signature; add after computation.
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	buf := bytes.NewBuffer(nil)
	buf.WriteByte('{')
	for i, k := range keys {
		v, _ := json.Marshal(rec[k])
		buf.WriteString("\"")
		buf.WriteString(k)
		buf.WriteString("\":")
		buf.Write(v)
		if i < len(keys)-1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte('}')
	h := sha256.Sum256(buf.Bytes())
	rec["hash"] = base64.RawURLEncoding.EncodeToString(h[:])

	// Sign the canonical JSON (without hash/signature) using the current key
	var sig []byte
	if curr.Private != nil {
		sig = ed25519.Sign(curr.Private, buf.Bytes())
		rec["signature"] = base64.RawURLEncoding.EncodeToString(sig)
		rec["public_key"] = base64.RawURLEncoding.EncodeToString(curr.Public)
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
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
	// RB9: tracing span for rotation.append (distinct from rotation.perform) to capture ledger emission timing & attributes.
	var span *tracing.Span
	if RotationTracerProvider != nil {
		ctx := context.Background()
		_, span = RotationTracerProvider.StartSpan(ctx, "rotation.append")
		if span != nil {
			// new_key_set_size approximated as history length + active (curr)
			span.SetTag("new_key_set_size", len(m.history)+1)
			span.SetTag("new_kid", curr.ID)
			if prev != nil {
				span.SetTag("prev_kid", prev.ID)
			}
		}
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
		if span != nil {
			span.SetTag("error", err.Error())
		}
	} else if span != nil {
		span.SetTag("append_success", true)
	}
	if span != nil {
		span.End()
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
	for _, k := range m.archived {
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
	now := time.Now().UTC()
	ttl := expires.Sub(now)
	deprecatedAfter := now.Add(time.Duration(float64(ttl) * 0.8))
	if deprecatedAfter.After(expires) {
		deprecatedAfter = expires // Safety: ensure deprecation <= expiration
	}
	k := &Key{
		ID:              kid,
		CreatedAt:       now,
		ExpiresAt:       expires,
		DeprecatedAfter: deprecatedAfter,
		SunsetAfter:     expires,
		Public:          ed25519.PublicKey(pub),
		Alg:             "EdDSA",
		Use:             "sig",
	}
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
	kept = append(kept, k)
	m.history = kept
}

// persistenceRecord models on-disk JSON.
type persistenceRecord struct {
	TTLHours int        `json:"ttl_hours"`
	Active   *diskKey   `json:"active"`
	History  []*diskKey `json:"history"`
	Archived []*diskKey `json:"archived,omitempty"`
}
type diskKey struct {
	ID              string `json:"kid"`
	CreatedAt       string `json:"created_at"`
	ExpiresAt       string `json:"expires_at"`
	DeprecatedAfter string `json:"deprecated_after,omitempty"`
	SunsetAfter     string `json:"sunset_after,omitempty"`
	PrivateB64      string `json:"private_b64"`
	PublicB64       string `json:"public_b64"`
	Alg             string `json:"alg"`
	Use             string `json:"use"`
}

// saveLocked writes current state to disk if persistPath set. Caller holds write lock.
func (m *Manager) saveLocked() error {
	if m.persistPath == "" {
		return nil
	}
	rec := persistenceRecord{TTLHours: int(m.ttl.Hours())}
	if m.active != nil {
		dk := &diskKey{
			ID:         m.active.ID,
			CreatedAt:  m.active.CreatedAt.Format(time.RFC3339),
			ExpiresAt:  m.active.ExpiresAt.Format(time.RFC3339),
			PrivateB64: base64.RawStdEncoding.EncodeToString(m.active.Private),
			PublicB64:  base64.RawStdEncoding.EncodeToString(m.active.Public),
			Alg:        m.active.Alg,
			Use:        m.active.Use,
		}
		if !m.active.DeprecatedAfter.IsZero() {
			dk.DeprecatedAfter = m.active.DeprecatedAfter.Format(time.RFC3339)
		}
		if !m.active.SunsetAfter.IsZero() {
			dk.SunsetAfter = m.active.SunsetAfter.Format(time.RFC3339)
		}
		rec.Active = dk
	}
	for _, hk := range m.history {
		dk := &diskKey{
			ID:         hk.ID,
			CreatedAt:  hk.CreatedAt.Format(time.RFC3339),
			ExpiresAt:  hk.ExpiresAt.Format(time.RFC3339),
			PrivateB64: base64.RawStdEncoding.EncodeToString(hk.Private),
			PublicB64:  base64.RawStdEncoding.EncodeToString(hk.Public),
			Alg:        hk.Alg,
			Use:        hk.Use,
		}
		if !hk.DeprecatedAfter.IsZero() {
			dk.DeprecatedAfter = hk.DeprecatedAfter.Format(time.RFC3339)
		}
		if !hk.SunsetAfter.IsZero() {
			dk.SunsetAfter = hk.SunsetAfter.Format(time.RFC3339)
		}
		rec.History = append(rec.History, dk)
	}
	for _, ak := range m.archived {
		dk := &diskKey{
			ID:         ak.ID,
			CreatedAt:  ak.CreatedAt.Format(time.RFC3339),
			ExpiresAt:  ak.ExpiresAt.Format(time.RFC3339),
			PrivateB64: base64.RawStdEncoding.EncodeToString(ak.Private),
			PublicB64:  base64.RawStdEncoding.EncodeToString(ak.Public),
			Alg:        ak.Alg,
			Use:        ak.Use,
		}
		if !ak.DeprecatedAfter.IsZero() {
			dk.DeprecatedAfter = ak.DeprecatedAfter.Format(time.RFC3339)
		}
		if !ak.SunsetAfter.IsZero() {
			dk.SunsetAfter = ak.SunsetAfter.Format(time.RFC3339)
		}
		rec.Archived = append(rec.Archived, dk)
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
	var archived []*Key
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
		k := &Key{
			ID:        dk.ID,
			CreatedAt: created,
			ExpiresAt: expires,
			Private:   ed25519.PrivateKey(privBytes),
			Public:    ed25519.PublicKey(pubBytes),
			Alg:       dk.Alg,
			Use:       dk.Use,
		}
		// Parse optional deprecation timestamps (backward compatible)
		if dk.DeprecatedAfter != "" {
			if deprecated, err := time.Parse(time.RFC3339, dk.DeprecatedAfter); err == nil {
				k.DeprecatedAfter = deprecated
			}
		}
		if dk.SunsetAfter != "" {
			if sunset, err := time.Parse(time.RFC3339, dk.SunsetAfter); err == nil {
				k.SunsetAfter = sunset
			}
		}
		return k, nil
	}
	if rec.Active != nil {
		ak, err := parseKey(rec.Active)
		if err == nil && now.Before(ak.ExpiresAt) {
			active = ak
		}
	}
	for _, dk := range rec.History {
		if hk, err := parseKey(dk); err == nil {
			if now.Before(hk.ExpiresAt) {
				history = append(history, hk)
			} else {
				// Migrate expired history to archive on load
				archived = append(archived, hk)
			}
		}
	}
	for _, dk := range rec.Archived {
		if ak, err := parseKey(dk); err == nil {
			archived = append(archived, ak)
		}
	}
	// Assign under lock
	m.mu.Lock()
	m.active = active
	m.history = history
	m.archived = archived
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

// -------------------- KeyProvider Implementation --------------------

// ActiveSigner returns a Signer for the currently active key.
func (m *Manager) ActiveSigner() (Signer, error) {
	k := m.Active()
	if k == nil {
		return nil, errors.New("no active key")
	}
	// Create a signer that implements the full Signer interface
	return &ed25519Signer{
		keyID: k.ID,
		priv:  k.Private,
		pub:   k.Public,
		algo:  k.Alg,
	}, nil
}

// PublicKey returns the public key bytes and algorithm for a given key ID.
func (m *Manager) PublicKey(keyID string) ([]byte, string, error) {
	k := m.FindByID(keyID)
	if k == nil {
		return nil, "", errors.New("unknown key")
	}
	return k.Public, k.Alg, nil
}

// VerifyWith verifies a signature using a specific key ID.
func (m *Manager) VerifyWith(msg, sig []byte, keyID string) error {
	return m.ValidateSignature(keyID, msg, sig)
}
