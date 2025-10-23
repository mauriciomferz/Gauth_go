package tsa

// Package tsa provides interfaces for external Time Stamping / Notarization services
// enabling anchoring of capability registry states and audit ledger entries.
// Initial version focuses on abstraction; concrete implementations will follow (e.g. RFC3161, Blockchain, Internal TSA).

import (
	"context"
	"errors"
	"time"
)

// TimestampRequest encapsulates data to be timestamped (canonical hash preferred) and optional metadata.
type TimestampRequest struct {
	Digest    []byte            // canonical digest (e.g. SHA256) of object being anchored
	DigestAlg string            // digest algorithm identifier (e.g. sha256)
	Nonce     []byte            // optional nonce for replay protection
	Meta      map[string]string // optional metadata (e.g. object_type, version)
}

// TimestampResponse represents TSA issued receipt with verifiable evidence.
type TimestampResponse struct {
	ReceiptBytes []byte            // opaque receipt serialization (RFC3161 or custom)
	IssuedAt     time.Time         // timestamp when TSA issued receipt
	ChainID      string            // optional chain / ledger identifier
	TxID         string            // optional transaction ID / anchor ID
	EvidenceHash []byte            // hash of receipt for audit inclusion
	Meta         map[string]string // propagated metadata
}

// AnchorClient defines minimal TSA operations for advanced anchoring receipts.
type AnchorClient interface {
	// Anchor submits a timestamp request and returns a verifiable receipt.
	Anchor(ctx context.Context, req *TimestampRequest) (*TimestampResponse, error)
	// Name returns a stable identifier for the client implementation (e.g. rfc3161, eth-sepolia, mock).
	Name() string
}

// Verifier validates a receipt against the original request digest and TSA trust parameters.
type Verifier interface {
	Verify(ctx context.Context, req *TimestampRequest, resp *TimestampResponse) error
	// TrustAnchorID returns identifier of trust anchor (e.g. CA cert fingerprint / contract address).
	TrustAnchorID() string
}

// ReceiptVerificationResult contains structured outcome of verification beyond error presence.
type ReceiptVerificationResult struct {
	Valid        bool
	Reason       string
	IssuedAt     time.Time
	DriftSeconds int64 // difference between claimed timestamp and local clock
}

// AdvancedVerifier optionally returns structured result (implementations may embed Verifier).
type AdvancedVerifier interface {
	Verifier
	VerifyDetailed(ctx context.Context, req *TimestampRequest, resp *TimestampResponse) (*ReceiptVerificationResult, error)
}

// MemoryClient is a minimal in-memory TSA client for tests; issues synthetic receipts.
type MemoryClient struct{ id string }

// NewMemoryClient constructs a memory TSA client.
func NewMemoryClient(id string) *MemoryClient {
	if id == "" {
		id = "memory-tsa"
	}
	return &MemoryClient{id: id}
}

// Anchor issues a synthetic receipt containing provided digest and current time.
func (m *MemoryClient) Anchor(ctx context.Context, req *TimestampRequest) (*TimestampResponse, error) {
	if req == nil || len(req.Digest) == 0 {
		return nil, ErrInvalidRequest
	}
	now := time.Now().UTC()
	rb := make([]byte, len(req.Digest))
	copy(rb, req.Digest)
	return &TimestampResponse{ReceiptBytes: rb, IssuedAt: now, EvidenceHash: rb, Meta: req.Meta}, nil
}

func (m *MemoryClient) Name() string { return m.id }

// MemoryVerifier trusts memory receipts that echo digest.
type MemoryVerifier struct{ anchorID string }

func NewMemoryVerifier(anchorID string) *MemoryVerifier {
	if anchorID == "" {
		anchorID = "memory-anchor"
	}
	return &MemoryVerifier{anchorID: anchorID}
}

func (v *MemoryVerifier) Verify(ctx context.Context, req *TimestampRequest, resp *TimestampResponse) error {
	r, err := v.VerifyDetailed(ctx, req, resp)
	if err != nil {
		return err
	}
	if !r.Valid {
		return ErrVerificationFailed
	}
	return nil
}

func (v *MemoryVerifier) TrustAnchorID() string { return v.anchorID }

func (v *MemoryVerifier) VerifyDetailed(ctx context.Context, req *TimestampRequest,
	resp *TimestampResponse) (*ReceiptVerificationResult, error) {
	if req == nil || resp == nil {
		return &ReceiptVerificationResult{Valid: false, Reason: "nil_request_or_response"}, ErrInvalidRequest
	}
	if len(req.Digest) == 0 || len(resp.ReceiptBytes) == 0 {
		return &ReceiptVerificationResult{Valid: false, Reason: "empty_digest_or_receipt"}, ErrInvalidRequest
	}
	valid := len(req.Digest) == len(resp.ReceiptBytes)
	if valid {
		for i := range req.Digest {
			if req.Digest[i] != resp.ReceiptBytes[i] {
				valid = false
				break
			}
		}
	}
	res := &ReceiptVerificationResult{
		Valid:        valid,
		Reason:       "ok",
		IssuedAt:     resp.IssuedAt,
		DriftSeconds: time.Now().UTC().Unix() - resp.IssuedAt.Unix(),
	}
	if !valid {
		res.Reason = "digest_mismatch"
	}
	return res, nil
}

var (
	ErrInvalidRequest     = errors.New("tsa: invalid request")
	ErrVerificationFailed = errors.New("tsa: verification failed")
)
