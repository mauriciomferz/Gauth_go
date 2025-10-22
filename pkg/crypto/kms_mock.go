package crypto

// kms_mock.go
// MockKMS provides a demo external-style KMS implementing the KMS interface with
// simulated latency and simple rotation semantics. Intended for local testing
// and future integration patterns (Vault/HSM/etc.).

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
    // KMS error taxonomy (expandable for real providers)
    ErrKMSUnavailable     = errors.New("kms: unavailable")
    ErrKMSPermission      = errors.New("kms: permission denied")
    ErrKMSNotFound        = errors.New("kms: key not found")
    ErrKMSUnsupportedAlgo = errors.New("kms: unsupported algorithm")
)

// MockKMS implements KMS with an in-memory map of keys, simulated latency, and counters.
type MockKMS struct {
    mu       sync.RWMutex
    active   *ed25519Signer
    publics  map[string]ed25519.PublicKey
    created  map[string]time.Time
    // Counters (internal) – exposed via Snapshot for tests; Prometheus optional later.
    cActiveSigner int64
    cRotate       int64
    cListKeys     int64
    latency       time.Duration
}

// NewMockKMS constructs a mock KMS with one initial Ed25519 key.
func NewMockKMS() (*MockKMS, error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { return nil, fmt.Errorf("generate initial key: %w", err) }
    keyID := deriveKeyID(pub)
    signer := &ed25519Signer{keyID: keyID, priv: priv, pub: pub, algo: AlgoEd25519}
    mk := &MockKMS{active: signer, publics: map[string]ed25519.PublicKey{keyID: pub}, created: map[string]time.Time{keyID: time.Now().UTC()}}
    // Optional latency via env (ms)
    if v := os.Getenv("GAUTH_KMS_LATENCY_MS"); v != "" {
        if ms, err := strconv.Atoi(v); err == nil && ms >= 0 && ms < 60000 { mk.latency = time.Duration(ms) * time.Millisecond }
    }
    maybeEnableKMSMetrics()
    return mk, nil
}

func (m *MockKMS) ActiveSigner() (Signer, error) {
    recordKMSMetric("mock", "active_signer", func() { m.simLatency() })
    m.mu.RLock(); defer m.mu.RUnlock()
    if m.active == nil { return nil, ErrKMSUnavailable }
    m.cActiveSigner++
    return m.active, nil
}

func (m *MockKMS) PublicKey(id string) ([]byte, string, error) {
    recordKMSMetric("mock", "public_key", func() { m.simLatency() })
    m.mu.RLock(); defer m.mu.RUnlock()
    pk, ok := m.publics[id]
    if !ok { return nil, "", ErrKMSNotFound }
    return pk, AlgoEd25519, nil
}

    // rotate operation metrics & latency tracked under op=rotate
func (m *MockKMS) Rotate() (string, error) {
    recordKMSMetric("mock", "rotate", func() { m.simLatency() })
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { return "", fmt.Errorf("kms rotate: %w", err) }
    keyID := deriveKeyID(pub)
    signer := &ed25519Signer{keyID: keyID, priv: priv, pub: pub, algo: AlgoEd25519}
    m.mu.Lock()
    m.publics[keyID] = pub
    m.created[keyID] = time.Now().UTC()
    m.active = signer
    m.cRotate++
    m.mu.Unlock()
    return keyID, nil
}

func (m *MockKMS) ListKeys() ([]KeyMetadata, error) {
    recordKMSMetric("mock", "list_keys", func() { m.simLatency() })
    m.mu.RLock(); defer m.mu.RUnlock()
    m.cListKeys++
    out := make([]KeyMetadata, 0, len(m.publics))
    for id, pub := range m.publics {
        _ = pub
        out = append(out, KeyMetadata{ID: id, Algorithm: AlgoEd25519, CreatedAt: m.created[id].Unix(), Active: m.active != nil && m.active.keyID == id})
    }
    return out, nil
}

// Snapshot exposes internal counters for tests.
func (m *MockKMS) Snapshot() map[string]int64 {
    m.mu.RLock(); defer m.mu.RUnlock()
    return map[string]int64{"active_signer_calls": m.cActiveSigner, "rotate_calls": m.cRotate, "list_calls": m.cListKeys}
}

func (m *MockKMS) simLatency() {
    if m.latency > 0 { time.Sleep(m.latency) }
}
