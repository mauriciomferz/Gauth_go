package notary

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// Rotation summary mode constants.
const (
	rotationModeEdDSA = "EdDSA"
)

// Rotation summary verification reasons.
const (
	reasonSummaryNil        = "summary_nil"
	reasonMissingSignature  = "missing_signature"
	reasonModeUnsupported   = "mode_unsupported"
	reasonPublicKeyInvalid  = "public_key_invalid"
	reasonKidMismatch       = "kid_mismatch"
	reasonSignatureInvalid  = "signature_invalid"
	reasonSerializationFail = "serialization_error"
)

// RotationLedgerRecord represents a single hash-chained rotation descriptor entry.
// Hash is computed as sha256(prev_hash || canonical_descriptor_bytes).
// canonical_descriptor_bytes are produced by canonicalRotationDescriptor (domain separated).
type RotationLedgerRecord struct {
	Index      int                    `json:"index"`
	Hash       string                 `json:"hash"`
	PrevHash   string                 `json:"prev_hash"`
	Descriptor *KeyRotationDescriptor `json:"descriptor"`
	Timestamp  string                 `json:"timestamp"`
}

// rotationLedgerFileModel persisted file schema (append-friendly rewrite-on-append approach).
type rotationLedgerFileModel struct {
	Entries   []RotationLedgerRecord `json:"entries"`
	HeadHash  string                 `json:"head_hash"`
	UpdatedAt string                 `json:"updated_at"`
	Version   int                    `json:"version"`
}

// RotationLedger provides minimal persistence for rotation descriptors, independent from receipts.
// Append rewrites entire file (acceptable for small chain lengths). Not optimized for large scale.
type RotationLedger struct {
	path    string
	entries []RotationLedgerRecord
	head    string
}

// NewRotationLedger creates a ledger bound to given path (may not yet exist).
func NewRotationLedger(path string) *RotationLedger { return &RotationLedger{path: path} }

// Load reads existing file if present.
func (l *RotationLedger) Load() error {
	if l.path == "" {
		return errors.New("rotation ledger path empty")
	}
	b, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var fm rotationLedgerFileModel
	if err := json.Unmarshal(b, &fm); err != nil {
		return err
	}
	l.entries = fm.Entries
	l.head = fm.HeadHash
	return nil
}

// AppendDescriptor adds a rotation descriptor to the ledger computing chained hash.
// Returns resulting record.
func (l *RotationLedger) AppendDescriptor(rd *KeyRotationDescriptor) (RotationLedgerRecord, error) {
	if rd == nil {
		return RotationLedgerRecord{}, errors.New("descriptor nil")
	}
	if l.path == "" {
		return RotationLedgerRecord{}, errors.New("rotation ledger path empty")
	}
	// Build canonical bytes (without signatures) and compute hash chaining with previous head.
	msg, err := canonicalRotationDescriptor(rd)
	if err != nil {
		return RotationLedgerRecord{}, err
	}
	prev := l.head
	h := sha256Sum(prev, msg)
	rec := RotationLedgerRecord{Index: len(l.entries), Hash: fmt.Sprintf("%x", h), PrevHash: prev, Descriptor: rd, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
	l.entries = append(l.entries, rec)
	l.head = rec.Hash
	// Persist full file atomically.
	fm := rotationLedgerFileModel{Entries: l.entries, HeadHash: l.head, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano), Version: 1}
	buf, err := json.Marshal(fm)
	if err != nil {
		return RotationLedgerRecord{}, err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o600); err != nil {
		return RotationLedgerRecord{}, err
	}
	if err := os.Rename(tmp, l.path); err != nil {
		return RotationLedgerRecord{}, err
	}
	return rec, nil
}

// Entries returns a copy of records.
func (l *RotationLedger) Entries() []RotationLedgerRecord {
	out := make([]RotationLedgerRecord, len(l.entries))
	copy(out, l.entries)
	return out
}

// HeadHash returns head hash.
func (l *RotationLedger) HeadHash() string { return l.head }

// sha256Sum computes sha256(prev || data).
func sha256Sum(prev string, data []byte) []byte {
	sum := sha256.New()
	sum.Write([]byte(prev))
	sum.Write(data)
	return sum.Sum(nil)
}

// RotationSummary describes chain status (for signing & endpoint exposure).
type RotationSummary struct {
	ChainLength   int    `json:"chain_length"`
	HeadHash      string `json:"head_hash"`
	AggregateHash string `json:"aggregate_hash"`
	GeneratedAt   string `json:"generated_at"`
	// Optional signature fields
	Kid       string `json:"kid,omitempty"`
	Signature string `json:"signature,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

// BuildRotationSummary constructs a summary from ledger entries computing aggregate hash.
// Aggregate hash: sha256(concat(all record hashes in order)).
func BuildRotationSummary(l *RotationLedger) RotationSummary {
	var hashesConcat []byte
	for _, rec := range l.entries {
		hashesConcat = append(hashesConcat, []byte(rec.Hash)...)
	}
	agg := fmt.Sprintf("%x", sha256Raw(hashesConcat))
	return RotationSummary{ChainLength: len(l.entries), HeadHash: l.head, AggregateHash: agg, GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano)}
}

// sha256Raw returns sha256 sum of data.
func sha256Raw(data []byte) []byte { h := sha256.New(); h.Write(data); return h.Sum(nil) }

// SignRotationSummary attaches EdDSA signature if active key provided. Domain separation applied.
func SignRotationSummary(sum *RotationSummary, priv ed25519.PrivateKey, kid string) error {
	if sum == nil || len(priv) != ed25519.PrivateKeySize {
		return errors.New("invalid inputs")
	}
	enc, err := canonicalRotationSummaryPayload(sum)
	if err != nil {
		return err
	}
	msg := append([]byte("GAUTH_ROTATION_SUMMARY:"), enc...)
	sig := ed25519.Sign(priv, msg)
	sum.Kid = kid
	sum.Signature = base64.RawURLEncoding.EncodeToString(sig)
	sum.Mode = rotationModeEdDSA
	return nil
}

// VerifyRotationSummary verifies the EdDSA signature on a rotation summary.
// Returns (valid, reason). Reasons: missing_signature, kid_mismatch, serialization_error, signature_invalid, mode_unsupported.
// kid_mismatch is returned when provided public key does not derive the same kid as summary.Kid.
func VerifyRotationSummary(sum *RotationSummary, pub ed25519.PublicKey) (bool, string) {
	if sum == nil {
		return false, reasonSummaryNil
	}
	if sum.Signature == "" {
		return false, reasonMissingSignature
	}
	if sum.Mode != "" && sum.Mode != rotationModeEdDSA {
		return false, reasonModeUnsupported
	}
	if len(pub) != ed25519.PublicKeySize {
		return false, reasonPublicKeyInvalid
	}
	derivedKid := fmt.Sprintf("ed25519:%x", pub[:8])
	if sum.Kid != "" && sum.Kid != derivedKid {
		return false, reasonKidMismatch
	}
	enc, err := canonicalRotationSummaryPayload(sum)
	if err != nil {
		return false, reasonSerializationFail
	}
	msg := append([]byte("GAUTH_ROTATION_SUMMARY:"), enc...)
	sigBytes, err := base64.RawURLEncoding.DecodeString(sum.Signature)
	if err != nil {
		return false, reasonSignatureInvalid
	}
	if !ed25519.Verify(pub, msg, sigBytes) {
		return false, reasonSignatureInvalid
	}
	return true, ""
}

// canonicalRotationSummaryPayload marshals the signed subset of RotationSummary fields
// in a deterministic ordering for signature generation & verification.
func canonicalRotationSummaryPayload(sum *RotationSummary) ([]byte, error) {
	payload := struct {
		ChainLength   int    `json:"chain_length"`
		HeadHash      string `json:"head_hash"`
		AggregateHash string `json:"aggregate_hash"`
		GeneratedAt   string `json:"generated_at"`
	}{ChainLength: sum.ChainLength, HeadHash: sum.HeadHash, AggregateHash: sum.AggregateHash, GeneratedAt: sum.GeneratedAt}
	return json.Marshal(payload)
}
