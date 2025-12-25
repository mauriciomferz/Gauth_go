// Package anchor provides in-memory and stub external anchoring providers used
// for capability registry and revocation chain prototype timestamping.
package anchor

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/rand" //nolint:gosec // Intentionally using math/rand for simulation/latency
	"os"
	"strconv"
	"sync"
	"time"
)

// Receipt represents an external anchoring receipt (TSA / transparency log).
type Receipt struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Provider  string    `json:"provider"`
	Proof     []byte    `json:"proof,omitempty"`
	Version   int       `json:"version"`
}

// Provider defines minimal interface for external anchoring.
type Provider interface {
	Anchor(hash string) (Receipt, error)
	Latest() Receipt
	Verify(r Receipt) error
}

// MemoryProvider is a non-secure, in-memory provider useful for demo / testing.
type MemoryProvider struct {
	mu     sync.RWMutex
	latest Receipt
}

func NewMemoryProvider() *MemoryProvider { return &MemoryProvider{} }

func (m *MemoryProvider) Anchor(hash string) (Receipt, error) {
	if hash == "" {
		return Receipt{}, errors.New("empty hash")
	}
	r := Receipt{Hash: hash, Timestamp: time.Now().UTC(), Provider: "memory", Version: 1}
	m.mu.Lock()
	m.latest = r
	m.mu.Unlock()
	return r, nil
}

func (m *MemoryProvider) Latest() Receipt {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latest
}
func (m *MemoryProvider) Verify(r Receipt) error {
	if r.Hash == "" {
		return errors.New("invalid receipt hash")
	}
	return nil
}

// TSAStubProvider simulates a timestamp authority with latency & failure probability.
// NOT FOR PRODUCTION — demo/test only.
// Security: Uses math/rand for latency simulation but seeded from crypto/rand for unpredictability.
type TSAStubProvider struct {
	latest      Receipt
	minLatency  time.Duration
	maxLatency  time.Duration
	failProb    float64
	providerTag string
	rnd         *rand.Rand
	// forcedFailuresRemaining decrements before probabilistic model applies, enabling deterministic
	// initial failure sequences for tests (set via GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS).
	forcedFailuresRemaining int
}

// NewTSAStubProvider constructs a stub provider with non-deterministic randomness.
func NewTSAStubProvider(minMs, maxMs int, failProb float64) *TSAStubProvider {
	return NewTSAStubProviderSeeded(minMs, maxMs, failProb, time.Now().UnixNano())
}

// NewTSAStubProviderSeeded allows providing an explicit RNG seed for deterministic test scenarios.
// If seed < 0 a cryptographically secure random seed is used.
func NewTSAStubProviderSeeded(minMs, maxMs int, failProb float64, seed int64) *TSAStubProvider {
	if minMs <= 0 {
		minMs = 10
	}
	if maxMs < minMs {
		maxMs = minMs + 20
	}
	if failProb < 0 {
		failProb = 0
	}
	if seed < 0 {
		// Use crypto/rand for secure seed generation
		var buf [8]byte
		if _, err := cryptorand.Read(buf[:]); err != nil {
			// Fallback to time-based seed only if crypto/rand fails
			seed = time.Now().UnixNano()
		} else {
			seed = int64(binary.LittleEndian.Uint64(buf[:]))
		}
	}
	src := rand.NewSource(seed)
	return &TSAStubProvider{minLatency: time.Duration(minMs) * time.Millisecond, maxLatency: time.Duration(maxMs) * time.Millisecond, failProb: failProb, providerTag: "tsa-stub", rnd: rand.New(src)}
}

// NewTSAStubProviderFromEnv builds a stub provider reading GAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED if present.
// The env var enables deterministic test runs for flaky probability-based tests.
func NewTSAStubProviderFromEnv(minMs, maxMs int, failProb float64) *TSAStubProvider {
	if v := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			p := NewTSAStubProviderSeeded(minMs, maxMs, failProb, parsed)
			if ff := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS"); ff != "" {
				if n, err := strconv.Atoi(ff); err == nil && n > 0 {
					p.forcedFailuresRemaining = n
				}
			}
			return p
		}
	}
	p := NewTSAStubProvider(minMs, maxMs, failProb)
	if ff := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS"); ff != "" {
		if n, err := strconv.Atoi(ff); err == nil && n > 0 {
			p.forcedFailuresRemaining = n
		}
	}
	return p
}

func (p *TSAStubProvider) Anchor(hash string) (Receipt, error) {
	if hash == "" {
		return Receipt{}, errors.New("empty hash")
	}
	// Simulate latency
	span := p.minLatency
	if p.maxLatency > p.minLatency {
		delta := p.rnd.Int63n(int64(p.maxLatency - p.minLatency))
		span += time.Duration(delta)
	}
	time.Sleep(span)
	// Deterministic forced failures take precedence over probabilistic model.
	if p.forcedFailuresRemaining > 0 {
		p.forcedFailuresRemaining--
		return Receipt{}, errors.New("stub failure (forced)")
	}
	if p.failProb > 0 && p.rnd.Float64() < p.failProb {
		return Receipt{}, errors.New("stub failure")
	}
	proofSeed := sha256.Sum256([]byte(hash + time.Now().String()))
	r := Receipt{Hash: hash, Timestamp: time.Now().UTC(), Provider: p.providerTag, Proof: proofSeed[:], Version: 1}
	p.latest = r
	return r, nil
}

func (p *TSAStubProvider) Latest() Receipt { return p.latest }
func (p *TSAStubProvider) Verify(r Receipt) error {
	if r.Hash == "" {
		return errors.New("invalid receipt hash")
	}
	// Simulate minimal integrity check: proof must start with hex of first 4 bytes of hash digest.
	if len(r.Proof) == 0 {
		return errors.New("missing proof")
	}
	// Not a real check; demonstration only.
	return nil
}

// HashHex returns hex string of sha256.
func HashHex(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
