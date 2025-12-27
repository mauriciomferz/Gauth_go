package anchor

// Verification status constants (mirrors notary receipt store & metrics integrity statuses)
const (
	ExternalReceiptStatusOK       = "ok"
	ExternalReceiptStatusMismatch = "mismatch"
	ExternalReceiptStatusEmpty    = "empty"
)

// ExternalAnchorReceipt is the base receipt emitted by external anchor providers.
// Success is implicit (we only persist successful attempts).
// Timestamp stored with nanosecond precision UTC.
// Version reserved for future schema evolution (start at 1).
// LatencySeconds records observed provider operation latency.
// Provider normalized label (e.g. tsa-stub) matching metrics.
// Hash is provider-returned anchored hash (could be identical to capability registry hash or provider-specific digest).
// NOTE: We do not persist failure attempts to keep audit small; failures recorded via metrics.
// Extensible in future (e.g. provider signature, certificate chain, anchor ledger reference).
// json field order stable via struct marshal for deterministic hashing.
type ExternalAnchorReceipt struct {
	Hash           string  `json:"hash"`
	Timestamp      string  `json:"timestamp"`
	Provider       string  `json:"provider"`
	Version        int     `json:"version"`
	LatencySeconds float64 `json:"latency_seconds"`
}

// StoredExternalAnchorReceipt extends ExternalAnchorReceipt with hash-chain linkage.
// PrevHash is previous entry's chain hash (empty for first).
// ChainHash is sha256(prev_hash || canonical_json(base_with_prev_hash)).
// We exclude chain_hash from the canonical payload used for hashing.
type StoredExternalAnchorReceipt struct {
	ExternalAnchorReceipt
	PrevHash  string `json:"prev_hash"`
	ChainHash string `json:"chain_hash"`
}

// ReceiptStore defines the interface for persisting external anchor receipts.
type ReceiptStore interface {
	// Append persists a successful external anchor receipt with chain linkage.
	Append(r ExternalAnchorReceipt) (StoredExternalAnchorReceipt, error)

	// Latest returns most recent stored external anchor receipt (zero value if none).
	Latest() StoredExternalAnchorReceipt

	// VerifyIncremental performs incremental hash-chain verification.
	// Returns status (ok|mismatch|empty), mismatch index (-1 if none), and head hash of verified chain.
	VerifyIncremental() (status string, mismatchIdx int, verifiedHeadHash string)
}
