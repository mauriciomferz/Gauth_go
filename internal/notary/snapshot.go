package notary

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"time"
)

// Snapshot verification / integrity reason constants (centralized to avoid goconst duplication).
const (
	snapshotReasonMerkleMismatch       = "merkle_mismatch"
	snapshotReasonHashMismatch         = "hash_mismatch"
	snapshotReasonChainHeadMismatch    = "chain_head_mismatch"
	snapshotReasonReceiptCountMismatch = "receipt_count_mismatch"
	snapshotReasonSerializationError   = "serialization_error"
)

// Snapshot provides a summarized integrity checkpoint of the receipt chain.
// Signatures & external anchors omitted in scaffold; hash provides local integrity.
// PreviousHash enables chaining snapshots.
type Snapshot struct {
	Version         int         `json:"version"`
	GeneratedAt     string      `json:"generated_at"`
	ReceiptCount    int         `json:"receipt_count"`
	ChainHead       string      `json:"chain_head"`
	MerkleRoot      string      `json:"merkle_root,omitempty"`
	RotationHead    string      `json:"rotation_head,omitempty"` // last rotation receipt chain_hash if any
	PreviousHash    string      `json:"previous_snapshot_hash,omitempty"`
	Hash            string      `json:"hash"`
	ExternalAnchors []AnchorRef `json:"external_anchors,omitempty"`
}

// AnchorRef placeholder for future external anchoring metadata.
type AnchorRef struct {
	Type      string `json:"type"` // tsa | tlog
	Provider  string `json:"provider"`
	Reference string `json:"reference"` // serial or leaf id
	Timestamp string `json:"timestamp"`
}

// GenerateSnapshot builds a snapshot over the current state of a ReceiptStore.
// It recomputes Merkle root only if enabled via AGENTAUTH_NOTARY_MERKLE_ENABLED.
// RotationHead: scans entries for last with Rotation descriptor.
// PreviousHash must be provided by caller (persisted from prior snapshot) for chaining.
func GenerateSnapshot(rs *ReceiptStore, previousHash string) (Snapshot, error) {
	start := time.Now()
	entries := rs.Entries()
	receiptCount := len(entries)
	chainHead := ""
	if receiptCount > 0 {
		chainHead = entries[receiptCount-1].ChainHash
	}
	merkleRoot := ""
	if receiptCount > 0 && isMerkleEnabled() {
		// reuse computeMerkleRoot building leaves from stored receipts base JSON
		leaves := make([][32]byte, 0, receiptCount)
		for _, e := range entries {
			tmp := struct {
				Hash           string  `json:"hash"`
				Timestamp      string  `json:"timestamp"`
				Provider       string  `json:"provider"`
				Version        int     `json:"version"`
				Success        bool    `json:"success"`
				LatencySeconds float64 `json:"latency_seconds"`
				PrevHash       string  `json:"prev_hash"`
			}{
				Hash:           e.Hash,
				Timestamp:      e.Timestamp,
				Provider:       e.Provider,
				Version:        e.Version,
				Success:        e.Success,
				LatencySeconds: e.LatencySeconds,
				PrevHash:       e.PrevHash,
			}
			b, err := json.Marshal(tmp)
			if err != nil {
				recordSnapshotGeneration(start, err)
				return Snapshot{}, err
			}
			leaves = append(leaves, sha256.Sum256(b))
		}
		root := computeMerkleRoot(leaves)
		merkleRoot = encodeHex(root[:])
	}
	rotationHead := ""
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Rotation != nil {
			rotationHead = entries[i].ChainHash
			break
		}
	}
	s := Snapshot{
		Version:      1,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		ReceiptCount: receiptCount,
		ChainHead:    chainHead,
		MerkleRoot:   merkleRoot,
		RotationHead: rotationHead,
		PreviousHash: previousHash,
	}
	// compute snapshot hash over canonical subset (excluding Hash field itself initially)
	canonical := struct {
		Version      int    `json:"version"`
		GeneratedAt  string `json:"generated_at"`
		ReceiptCount int    `json:"receipt_count"`
		ChainHead    string `json:"chain_head"`
		MerkleRoot   string `json:"merkle_root,omitempty"`
		RotationHead string `json:"rotation_head,omitempty"`
		PreviousHash string `json:"previous_snapshot_hash,omitempty"`
	}{
		Version:      s.Version,
		GeneratedAt:  s.GeneratedAt,
		ReceiptCount: s.ReceiptCount,
		ChainHead:    s.ChainHead,
		MerkleRoot:   s.MerkleRoot,
		RotationHead: s.RotationHead,
		PreviousHash: s.PreviousHash,
	}
	enc, err := json.Marshal(canonical)
	if err != nil {
		recordSnapshotGeneration(start, err)
		return Snapshot{}, err
	}
	digest := sha256.Sum256(enc)
	s.Hash = encodeHex(digest[:])
	recordSnapshotGeneration(start, nil)
	return s, nil
}

// isMerkleEnabled helper (avoids circular import).
func isMerkleEnabled() bool { return os.Getenv("AGENTAUTH_NOTARY_MERKLE_ENABLED") == "1" }

// minimal hex encoder wrapper to avoid pulling fmt.
func encodeHex(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}

// (Removed layered getenv indirection; use os.Getenv directly for simplicity.)

// SnapshotVerificationResult captures integrity evaluation.
type SnapshotVerificationResult struct {
	Valid          bool   `json:"valid"`
	HashMatch      bool   `json:"hash_match"`
	MerkleMatch    bool   `json:"merkle_match"`
	ChainHeadMatch bool   `json:"chain_head_match"`
	ReceiptCountOk bool   `json:"receipt_count_ok"`
	Reason         string `json:"reason,omitempty"`
}

// VerifySnapshot recomputes expected snapshot hash and (if merkle root present) merkle root
// from the provided ReceiptStore then compares with the supplied snapshot fields.
// It does NOT verify signatures or external anchors (future work).
func VerifySnapshot(rs *ReceiptStore, snap Snapshot) (SnapshotVerificationResult, error) {
	start := time.Now()
	entries := rs.Entries()
	rc := len(entries)
	// Recompute merkle if snapshot has it (merkle feature may be disabled now but we still attempt to reproduce)
	recomputedMerkle := ""
	if snap.MerkleRoot != "" {
		leaves := make([][32]byte, 0, rc)
		for _, e := range entries {
			tmp := struct {
				Hash           string  `json:"hash"`
				Timestamp      string  `json:"timestamp"`
				Provider       string  `json:"provider"`
				Version        int     `json:"version"`
				Success        bool    `json:"success"`
				LatencySeconds float64 `json:"latency_seconds"`
				PrevHash       string  `json:"prev_hash"`
			}{
				Hash:           e.Hash,
				Timestamp:      e.Timestamp,
				Provider:       e.Provider,
				Version:        e.Version,
				Success:        e.Success,
				LatencySeconds: e.LatencySeconds,
				PrevHash:       e.PrevHash,
			}
			b, err := json.Marshal(tmp)
			if err != nil {
				recordSnapshotVerification(start, false, err)
				return SnapshotVerificationResult{Valid: false, Reason: "serialization_error"}, err
			}
			leaves = append(leaves, sha256.Sum256(b))
		}
		root := computeMerkleRoot(leaves)
		recomputedMerkle = encodeHex(root[:])
	}
	chainHead := ""
	if rc > 0 {
		chainHead = entries[rc-1].ChainHash
	}
	canonical := struct {
		Version      int    `json:"version"`
		GeneratedAt  string `json:"generated_at"`
		ReceiptCount int    `json:"receipt_count"`
		ChainHead    string `json:"chain_head"`
		MerkleRoot   string `json:"merkle_root,omitempty"`
		RotationHead string `json:"rotation_head,omitempty"`
		PreviousHash string `json:"previous_snapshot_hash,omitempty"`
	}{
		Version:      snap.Version,
		GeneratedAt:  snap.GeneratedAt,
		ReceiptCount: snap.ReceiptCount,
		ChainHead:    snap.ChainHead,
		MerkleRoot:   snap.MerkleRoot,
		RotationHead: snap.RotationHead,
		PreviousHash: snap.PreviousHash,
	}
	enc, err := json.Marshal(canonical)
	if err != nil {
		recordSnapshotVerification(start, false, err)
		return SnapshotVerificationResult{Valid: false, Reason: "serialization_error"}, err
	}
	digest := sha256.Sum256(enc)
	expectedHash := encodeHex(digest[:])
	res := SnapshotVerificationResult{
		HashMatch: expectedHash == snap.Hash,
		MerkleMatch: (snap.MerkleRoot == "" && recomputedMerkle == "") ||
			(snap.MerkleRoot != "" && recomputedMerkle == snap.MerkleRoot),
		ChainHeadMatch: chainHead == snap.ChainHead,
		ReceiptCountOk: rc == snap.ReceiptCount,
	}
	res.Valid = res.HashMatch && res.MerkleMatch && res.ChainHeadMatch && res.ReceiptCountOk
	if !res.Valid {
		// Priority: merkle mismatch > hash mismatch > chain head > count
		switch {
		case !res.MerkleMatch:
			res.Reason = snapshotReasonMerkleMismatch
		case !res.HashMatch:
			res.Reason = snapshotReasonHashMismatch
		case !res.ChainHeadMatch:
			res.Reason = snapshotReasonChainHeadMismatch
		case !res.ReceiptCountOk:
			res.Reason = snapshotReasonReceiptCountMismatch
		}
	}
	recordSnapshotVerification(start, res.Valid, nil)
	return res, nil
}
