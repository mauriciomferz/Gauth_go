package notary

// transparency_log_stub.go introduces a scaffold for integrating with an external transparency
// log service (e.g., Sigstore Rekor). Real implementations would submit structured entries and
// receive inclusion proofs (Merkle Tree leaf hash, inclusion proof path, signed tree head).
// The stub allows wiring metrics and configuration without external dependencies.

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// TransparencyLogEntryRequest represents a request to log a hash (e.g., capability anchor or audit chain tip).
// Production systems might include signature, public key, artifact type, and content metadata.
type TransparencyLogEntryRequest struct {
	Hash string
}

// TransparencyLogEntryResponse captures minimal metadata returned by a transparency log.
// Fields for future expansion: inclusion_proof (array of sibling hashes), signed_tree_head,
// log_index, log_id, leaf_hash, root_hash.
type TransparencyLogEntryResponse struct {
	Hash            string  `json:"hash"`
	LoggedTime      string  `json:"logged_time"` // RFC3339Nano
	Provider        string  `json:"provider"`
	Version         int     `json:"version"`
	Success         bool    `json:"success"`
	LatencySeconds  float64 `json:"latency_seconds"`
	EmulatedLeafID  string  `json:"emulated_leaf_id"`  // placeholder for real log index/id
	EmulatedRootRef string  `json:"emulated_root_ref"` // placeholder for tree root reference
}

// TransparencyLogger defines interface for submitting entries to a transparency log.
type TransparencyLogger interface {
	Log(req TransparencyLogEntryRequest) (TransparencyLogEntryResponse, error)
}

// StubTransparencyLogger simulates logging by locally synthesizing metadata and optional latency.
// Environment variables:
//
//	AGENTAUTH_TLOG_STUB_MIN_LATENCY_MS (default 25)
//	AGENTAUTH_TLOG_STUB_MAX_LATENCY_MS (default 90)
//	AGENTAUTH_TLOG_STUB_PROVIDER_NAME  (default "tlog_stub")
//
// Latency uniform in range; leaf id & root ref are time-based placeholders.
type StubTransparencyLogger struct {
	minLatency time.Duration
	maxLatency time.Duration
	provider   string
}

// NewStubTransparencyLogger builds a logger from environment configuration.
func NewStubTransparencyLogger() *StubTransparencyLogger {
	minMs := parseIntEnvBounds("AGENTAUTH_TLOG_STUB_MIN_LATENCY_MS", 25, 0, 2000)
	maxMs := parseIntEnvBounds("AGENTAUTH_TLOG_STUB_MAX_LATENCY_MS", 90, minMs, 5000)
	name := os.Getenv("AGENTAUTH_TLOG_STUB_PROVIDER_NAME")
	if name == "" {
		name = "tlog_stub"
	}
	return &StubTransparencyLogger{
		minLatency: time.Duration(minMs) * time.Millisecond,
		maxLatency: time.Duration(maxMs) * time.Millisecond,
		provider:   name,
	}
}

// Log produces a synthetic response. Hash required.
func (s *StubTransparencyLogger) Log(req TransparencyLogEntryRequest) (TransparencyLogEntryResponse, error) {
	if req.Hash == "" {
		return TransparencyLogEntryResponse{}, errors.New("hash required")
	}
	start := time.Now()
	delay := s.minLatency
	span := s.maxLatency - s.minLatency
	if span > 0 {
		n := time.Now().UnixNano() % int64(span)
		delay += time.Duration(n)
	}
	time.Sleep(delay)
	now := time.Now().UTC()
	resp := TransparencyLogEntryResponse{
		Hash:            req.Hash,
		LoggedTime:      now.Format(time.RFC3339Nano),
		Provider:        s.provider,
		Version:         1,
		Success:         true,
		LatencySeconds:  time.Since(start).Seconds(),
		EmulatedLeafID:  strconv.FormatInt(now.UnixNano(), 10),
		EmulatedRootRef: "root-" + strconv.FormatInt(now.Unix(), 10),
	}
	return resp, nil
}
