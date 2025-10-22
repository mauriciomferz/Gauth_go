package notary

// tsa_stub.go provides a scaffold for integrating a Time-Stamp Authority (TSA) per RFC3161
// without introducing external network calls yet. This prepares the codebase for a real
// implementation while enabling metrics wiring and configuration patterns similar to other
// notarizers. The stub generates deterministic pseudo-responses and DOES NOT provide
// cryptographic guarantees. It must be replaced with a production implementation to achieve
// trusted timestamping.

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// TSARequest represents the input to a TSA. In a real implementation this would include
// a hash (e.g. SHA-256 of data) and optionally a nonce to prevent replay along with a
// requested policy OID. We keep only Hash for initial scaffold.
type TSARequest struct {
	Hash string
}

// TSAResponse represents a minimal timestamp token result. A production version would
// carry a DER-encoded TimeStampToken plus parsed fields (e.g., genTime, serialNumber,
// policy OID, nonce echo, accuracy, ordering, TSA name, and signature algorithm data).
// Here we store simplified fields for wiring and future extension.
type TSAResponse struct {
	Hash              string  `json:"hash"`
	GenTime           string  `json:"gen_time"` // RFC3339Nano formatted
	Provider          string  `json:"provider"`
	Version           int     `json:"version"` // schema version for evolution
	Success           bool    `json:"success"`
	LatencySeconds    float64 `json:"latency_seconds"`
	EmulatedSerial    string  `json:"emulated_serial"` // placeholder for real serial
	EmulatedPolicyOID string  `json:"emulated_policy_oid"`
}

// TSA defines the interface for requesting a timestamp for a given hash.
// Real implementations must ensure cryptographic integrity and may return extended
// error information (e.g., PKI failures). For scaffold we keep it minimal.
type TSA interface {
	Timestamp(req TSARequest) (TSAResponse, error)
}

// StubTSA implements TSA using local time without external calls. It simulates latency
// and always "succeeds" unless the hash is empty. Environment variables configure basic
// behavior:
//
//	GAUTH_TSA_STUB_MIN_LATENCY_MS (default 30)
//	GAUTH_TSA_STUB_MAX_LATENCY_MS (default 120)
//	GAUTH_TSA_STUB_PROVIDER_NAME  (default "tsa_stub")
//	GAUTH_TSA_STUB_POLICY_OID     (default "1.3.6.1.4.1.example.stub")
//
// Latency is uniform in [min,max].
type StubTSA struct {
	minLatency time.Duration
	maxLatency time.Duration
	provider   string
	policyOID  string
}

// NewStubTSA constructs a StubTSA from environment configuration.
func NewStubTSA() *StubTSA {
	minMs := parseIntEnvBounds("GAUTH_TSA_STUB_MIN_LATENCY_MS", 30, 0, 2000)
	maxMs := parseIntEnvBounds("GAUTH_TSA_STUB_MAX_LATENCY_MS", 120, minMs, 5000)
	provider := os.Getenv("GAUTH_TSA_STUB_PROVIDER_NAME")
	if provider == "" {
		provider = "tsa_stub"
	}
	oid := os.Getenv("GAUTH_TSA_STUB_POLICY_OID")
	if oid == "" {
		oid = "1.3.6.1.4.1.example.stub"
	}
	return &StubTSA{minLatency: time.Duration(minMs) * time.Millisecond, maxLatency: time.Duration(maxMs) * time.Millisecond, provider: provider, policyOID: oid}
}

// Timestamp returns a synthetic timestamp response. Serial is a monotonic timestamp-based string.
func (s *StubTSA) Timestamp(req TSARequest) (TSAResponse, error) {
	if req.Hash == "" {
		return TSAResponse{}, errors.New("hash required")
	}
	start := time.Now()
	delay := s.minLatency
	span := s.maxLatency - s.minLatency
	if span > 0 {
		// simple deterministic pseudo-random: mod of current unix nano span
		n := time.Now().UnixNano() % int64(span)
		delay += time.Duration(n)
	}
	time.Sleep(delay)
	resp := TSAResponse{
		Hash:              req.Hash,
		GenTime:           time.Now().UTC().Format(time.RFC3339Nano),
		Provider:          s.provider,
		Version:           1,
		Success:           true,
		LatencySeconds:    time.Since(start).Seconds(),
		EmulatedSerial:    strconv.FormatInt(time.Now().UnixNano(), 10),
		EmulatedPolicyOID: s.policyOID,
	}
	return resp, nil
}

// parseIntEnvBounds mirrors existing helper patterns with local scope to avoid cross-file reuse
// until we refactor common env parsing utilities.
func parseIntEnvBounds(key string, def, min, max int) int {
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
