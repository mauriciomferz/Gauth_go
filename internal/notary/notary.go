package notary

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

// Receipt represents a notarization receipt for a capability registry hash or audit chain tip.
// Future fields: external transparency log ID, inclusion proof, signature.
// Hash should be the exact string submitted (e.g., sha256:<hex>). Timestamp is UTC.
// Provider indicates the backend (memory, tsa, rekor, etc.).
// Version reserved for future receipt schema changes.
// Success indicates whether external provider accepted the hash; failures may still produce partial metadata.
// LatencySeconds is the measured round-trip (client side) latency.
// Note: Keep struct stable for JSON encoding (used by status endpoints/tests).
//
// JSON tags kept explicit for forward compatibility.
//
// Minimal prototype only. Production implementation will integrate an external TSA or transparency log.
//
// Receipt may be embedded directly in status endpoint.
//
// Additional metadata (optional): chain_tip, previous_hash.
// We keep it small for now.
//
// All fields exported for encoding & external usage.
//
// Nonce / request id could be added later for correlation.
//
// Integrity fields: provider_signature (future) covering hash+timestamp.
//
// Security note: This memory implementation does not provide cryptographic guarantees.
// It is solely for wiring metrics and code paths.
//
// IMPORTANT: Do not rely on MemoryNotarizer for production integrity.
// Use proper external notary service (RFC3161 TSA, Sigstore Rekor, etc.).
//
// Backward compatibility: adding omitempty fields acceptable; avoid removing existing ones.
//
// LatencySeconds kept as float64 for potential high-resolution timing in future (e.g., fractional seconds).
// For prototype we record seconds with sub-second precision truncated.
type Receipt struct {
	Hash           string  `json:"hash"`
	Timestamp      string  `json:"timestamp"`
	Provider       string  `json:"provider"`
	Version        int     `json:"version"`
	Success        bool    `json:"success"`
	LatencySeconds float64 `json:"latency_seconds"`
	// Rotation contains key rotation continuity metadata when this receipt represents
	// a rotation event rather than a standard hash notarization. Omitted otherwise.
	Rotation *KeyRotationDescriptor `json:"rotation,omitempty"`
}

// KeyRotationDescriptor captures continuity data for rotating signing or anchor keys.
// This scaffold focuses on minimal linkage; a production version would include signatures
// over the descriptor itself by both old and new keys, plus possibly a grace period policy.
// Fields:
//
//	OldKeyID      - identifier (e.g., fingerprint) of retiring key.
//	NewKeyID      - identifier of replacing key.
//	EffectiveTime - UTC timestamp (RFC3339Nano) after which new key SHOULD be used.
//	Reason        - short code or human-readable motive (e.g., scheduled, compromised, expiring).
//	PrevRotationHash - hash of previous rotation receipt (forming a continuity chain distinct from receipts chain).
//
// Future expansion: dual signatures, deprecation policy, grace period end, verification status.
type KeyRotationDescriptor struct {
	OldKeyID         string `json:"old_key_id"`
	NewKeyID         string `json:"new_key_id"`
	EffectiveTime    string `json:"effective_time"`
	Reason           string `json:"reason"`
	PrevRotationHash string `json:"prev_rotation_hash,omitempty"`
	// Dual signatures (optional until enforcement enabled). Each signature covers the canonical
	// descriptor payload excluding signature fields, domain-separated. Old key attests retirement
	// and succession; new key attests acceptance of role. Both MUST be present for full continuity.
	OldKeySignature string `json:"old_key_signature,omitempty"`
	NewKeySignature string `json:"new_key_signature,omitempty"`
}

// Notarizer defines minimal interface for submitting a hash and receiving a receipt.
type Notarizer interface {
	Notarize(hash string) (Receipt, error)
}

// MemoryNotarizer is an in-memory, non-secure Notarizer implementation for prototype/testing.
// Records latest successful receipt and exposes it via Last().
// Not thread-safe beyond simple atomic usage; Serialize access in server if needed.
type MemoryNotarizer struct {
	mu     sync.RWMutex
	latest Receipt
}

// NewMemory returns a new MemoryNotarizer.
func NewMemory() *MemoryNotarizer { return &MemoryNotarizer{} }

// Notarize returns a receipt with current time for non-empty hash.
// Returns error for empty hash. Latency simulated as near-zero.
func (m *MemoryNotarizer) Notarize(hash string) (Receipt, error) {
	if hash == "" {
		return Receipt{}, errors.New("hash required")
	}
	start := time.Now()
	// Simulated external call latency placeholder (can be replaced with sleep or actual network call).
	// For now no delay.
	elapsed := time.Since(start).Seconds()
	r := Receipt{
		Hash:           hash,
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Provider:       "memory",
		Version:        1,
		Success:        true,
		LatencySeconds: elapsed,
	}
	m.mu.Lock()
	m.latest = r
	m.mu.Unlock()
	return r, nil
}

// Latest returns latest receipt (zero Timestamp if none).
func (m *MemoryNotarizer) Latest() Receipt {
	m.mu.RLock()
	r := m.latest
	m.mu.RUnlock()
	return r
}

// ExternalStubNotarizer simulates an external network-backed notarization provider.
// It introduces random latency and optional failure probability configurable via env.
// Security: Uses math/rand for latency simulation but seeded from crypto/rand for unpredictability.
// Env:
//
//	GAUTH_NOTARY_STUB_MIN_LATENCY_MS  (default 40)
//	GAUTH_NOTARY_STUB_MAX_LATENCY_MS  (default 250)
//	GAUTH_NOTARY_STUB_FAIL_PROB       (0.0-1.0, default 0) e.g. 0.05 for 5% failures
//	GAUTH_NOTARY_STUB_PROVIDER_NAME   (default "external_stub")
//
// This is strictly for integration & metrics wiring; provides NO integrity guarantees.
type ExternalStubNotarizer struct {
	mu           sync.RWMutex
	latest       Receipt
	minLatency   time.Duration
	maxLatency   time.Duration
	failProb     float64
	providerName string
	rnd          *rand.Rand
}

// NewExternalStub constructs a stub notarizer reading configuration from environment.
func NewExternalStub() *ExternalStubNotarizer {
	minMs := parseEnvInt("GAUTH_NOTARY_STUB_MIN_LATENCY_MS", 40, 0, 1000)
	maxMs := parseEnvInt("GAUTH_NOTARY_STUB_MAX_LATENCY_MS", 250, minMs, 5000)
	fp := parseEnvFloat("GAUTH_NOTARY_STUB_FAIL_PROB", 0, 0, 1)
	name := os.Getenv("GAUTH_NOTARY_STUB_PROVIDER_NAME")
	if name == "" {
		name = "external_stub"
	}
	// Use crypto/rand for secure seed generation
	var buf [8]byte
	var seed int64
	if _, err := cryptorand.Read(buf[:]); err != nil {
		// Fallback to time-based seed only if crypto/rand fails
		seed = time.Now().UnixNano()
	} else {
		// #nosec G115: int64 seed overflow is acceptable for RNG
		seed = int64(binary.LittleEndian.Uint64(buf[:]))
	}
	return &ExternalStubNotarizer{minLatency: time.Duration(minMs) * time.Millisecond, maxLatency: time.Duration(maxMs) * time.Millisecond, failProb: fp, providerName: name, rnd: rand.New(rand.NewSource(seed))} // #nosec G404
}

// Notarize simulates a network call with random latency and probabilistic failure.
func (e *ExternalStubNotarizer) Notarize(hash string) (Receipt, error) {
	if hash == "" {
		return Receipt{}, errors.New("hash required")
	}
	start := time.Now()
	// Simulate latency in [minLatency, maxLatency]
	span := e.maxLatency - e.minLatency
	delay := e.minLatency
	if span > 0 {
		delay += time.Duration(e.rnd.Int63n(int64(span)))
	}
	time.Sleep(delay)
	success := true
	if e.failProb > 0 && e.rnd.Float64() < e.failProb {
		success = false
	}
	elapsed := time.Since(start).Seconds()
	receipt := Receipt{Hash: hash, Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Provider: e.providerName, Version: 1, Success: success, LatencySeconds: elapsed}
	if success {
		e.mu.Lock()
		e.latest = receipt
		e.mu.Unlock()
		return receipt, nil
	}
	return receipt, errors.New("external_stub_notarization_failed")
}

// Latest returns last successful receipt (zero Timestamp if none).
func (e *ExternalStubNotarizer) Latest() Receipt {
	e.mu.RLock()
	r := e.latest
	e.mu.RUnlock()
	return r
}

// parseEnvInt reads an int env with bounds and default fallback.
func parseEnvInt(key string, def, min, max int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if i < min {
		i = min
	}
	if i > max {
		i = max
	}
	return i
}

// parseEnvFloat reads a float env with bounds and default fallback.
func parseEnvFloat(key string, def, min, max float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	if f < min {
		f = min
	}
	if f > max {
		f = max
	}
	return f
}
