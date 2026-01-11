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

	"github.com/mauriciomferz/AgentAuth/pkg/metrics"
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
	// RB5 signing fields (optional for backward compatibility)
	Kid       string `json:"kid,omitempty"`
	Signature string `json:"signature,omitempty"`
	Mode      string `json:"mode,omitempty"`
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
	// optional signer (RB5)
	signerPriv ed25519.PrivateKey
	signerKid  string
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
	rec := RotationLedgerRecord{
		Index:      len(l.entries),
		Hash:       fmt.Sprintf("%x", h),
		PrevHash:   prev,
		Descriptor: rd,
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	// RB5: optional entry signature (domain separated) if signer configured
	if len(l.signerPriv) == ed25519.PrivateKeySize {
		payload := append([]byte("AGENTAUTH_ROTATION_LEDGER_ENTRY:"), append([]byte(prev), msg...)...)
		sig := ed25519.Sign(l.signerPriv, payload)
		rec.Signature = base64.RawURLEncoding.EncodeToString(sig)
		if l.signerKid == "" {
			// derive kid similar to summary verify (prefix ed25519: first 8 bytes)
			pub := l.signerPriv.Public().(ed25519.PublicKey)
			l.signerKid = fmt.Sprintf("ed25519:%x", pub[:8])
		}
		rec.Kid = l.signerKid
		rec.Mode = rotationModeEdDSA
	}
	l.entries = append(l.entries, rec)
	l.head = rec.Hash
	// Persist full file atomically.
	fm := rotationLedgerFileModel{
		Entries:   l.entries,
		HeadHash:  l.head,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Version:   1,
	}
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

// ConfigureEd25519Signer enables RB5 per-entry signatures for future appends.
// Existing entries remain unsigned and are still valid unless strict mode enforced elsewhere.
func (l *RotationLedger) ConfigureEd25519Signer(priv ed25519.PrivateKey, kid string) {
	if len(priv) != ed25519.PrivateKeySize {
		return
	}
	l.signerPriv = priv
	l.signerKid = kid
}

// VerifyRotationLedger performs chained hash recomputation and signature checks.
// Returns (mismatches, invalidSigs). Unsigned entries do not count as invalid unless strict=true.
func VerifyRotationLedger(
	entries []RotationLedgerRecord,
	strict bool,
	pubResolver func(kid string) ed25519.PublicKey,
) (int, int) {
	mismatches := 0
	invalidSigs := 0
	prev := ""
	for _, rec := range entries {
		msg, err := canonicalRotationDescriptor(rec.Descriptor)
		if err != nil {
			mismatches++
			prev = rec.Hash
			continue
		}
		expected := fmt.Sprintf("%x", sha256Sum(prev, msg))
		if expected != rec.Hash {
			mismatches++
		}
		// Signature verification
		if rec.Signature != "" && rec.Kid != "" && rec.Mode == rotationModeEdDSA {
			pub := pubResolver(rec.Kid)
			if len(pub) != ed25519.PublicKeySize {
				invalidSigs++
			} else {
				payload := append(
					[]byte("AGENTAUTH_ROTATION_LEDGER_ENTRY:"),
					append([]byte(rec.PrevHash), msg...)...,
				)
				sigBytes, err := base64.RawURLEncoding.DecodeString(rec.Signature)
				if err != nil || len(sigBytes) != ed25519.SignatureSize || !ed25519.Verify(pub, payload, sigBytes) {
					invalidSigs++
				}
			}
		} else if strict {
			// in strict mode unsigned entries treated as invalid signature
			invalidSigs++
		}
		prev = rec.Hash
	}
	return mismatches, invalidSigs
}

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
	// Multi-signature extensions (beta)
	Threshold       int                 `json:"threshold,omitempty"`
	SatisfiedWeight int                 `json:"satisfied_weight,omitempty"`
	Signatures      []RotationSignature `json:"signatures,omitempty"`
}

// RotationSignature models an individual signature for multi-sig summaries.
type RotationSignature struct {
	Kid       string `json:"kid"`
	Mode      string `json:"mode"`
	Signature string `json:"signature"`
}

// BuildRotationSummary constructs a summary from ledger entries computing aggregate hash.
// Aggregate hash: sha256(concat(all record hashes in order)).
func BuildRotationSummary(l *RotationLedger) RotationSummary {
	var hashesConcat []byte
	for _, rec := range l.entries {
		hashesConcat = append(hashesConcat, []byte(rec.Hash)...)
	}
	agg := fmt.Sprintf("%x", sha256Raw(hashesConcat))
	return RotationSummary{
		ChainLength:   len(l.entries),
		HeadHash:      l.head,
		AggregateHash: agg,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
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
	msg := append([]byte("AGENTAUTH_ROTATION_SUMMARY:"), enc...)
	sig := ed25519.Sign(priv, msg)
	sum.Kid = kid
	sum.Signature = base64.RawURLEncoding.EncodeToString(sig)
	sum.Mode = rotationModeEdDSA
	// Also append to multi-signature slice for backward compatibility bridging.
	sum.Signatures = append(sum.Signatures, RotationSignature{Kid: kid, Mode: rotationModeEdDSA, Signature: sum.Signature})
	if sum.SatisfiedWeight == 0 { // initialize if unset
		sum.SatisfiedWeight = 1
	}
	return nil
}

// AppendSignatureToSummary adds an additional signature (multi-sig path). Does not mutate legacy Kid/Signature unless first.
func AppendSignatureToSummary(sum *RotationSummary, priv ed25519.PrivateKey, kid string) error {
	if sum == nil || len(priv) != ed25519.PrivateKeySize {
		return errors.New("invalid_inputs")
	}
	enc, err := canonicalRotationSummaryPayload(sum)
	if err != nil {
		return err
	}
	msg := append([]byte("AGENTAUTH_ROTATION_SUMMARY:"), enc...)
	sig := ed25519.Sign(priv, msg)
	sigStr := base64.RawURLEncoding.EncodeToString(sig)
	// If legacy fields empty, populate for backward compatibility.
	if sum.Signature == "" {
		sum.Kid = kid
		sum.Signature = sigStr
		sum.Mode = rotationModeEdDSA
	}
	sum.Signatures = append(sum.Signatures, RotationSignature{Kid: kid, Mode: rotationModeEdDSA, Signature: sigStr})
	sum.SatisfiedWeight = len(sum.Signatures)
	return nil
}

// VerifyRotationSummary verifies the EdDSA signature on a rotation summary.
// Returns (valid, reason). Reasons: missing_signature, kid_mismatch, serialization_error, signature_invalid, mode_unsupported.
// kid_mismatch is returned when provided public key does not derive the same kid as summary.Kid.
func VerifyRotationSummary(sum *RotationSummary, pub ed25519.PublicKey) (bool, string) {
	start := time.Now()
	if sum == nil {
		rotationSignatureVerifyFailures.WithLabelValues(reasonSummaryNil).Inc()
		metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
		return false, reasonSummaryNil
	}
	if sum.Signature == "" {
		rotationSignatureVerifyFailures.WithLabelValues(reasonMissingSignature).Inc()
		metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
		return false, reasonMissingSignature
	}
	if sum.Mode != "" && sum.Mode != rotationModeEdDSA {
		rotationSignatureVerifyFailures.WithLabelValues(reasonModeUnsupported).Inc()
		metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
		return false, reasonModeUnsupported
	}
	if len(pub) != ed25519.PublicKeySize {
		rotationSignatureVerifyFailures.WithLabelValues(reasonPublicKeyInvalid).Inc()
		metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
		return false, reasonPublicKeyInvalid
	}
	derivedKid := fmt.Sprintf("ed25519:%x", pub[:8])
	if sum.Kid != "" && sum.Kid != derivedKid {
		rotationSignatureVerifyFailures.WithLabelValues(reasonKidMismatch).Inc()
		metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
		return false, reasonKidMismatch
	}
	enc, err := canonicalRotationSummaryPayload(sum)
	if err != nil {
		rotationSignatureVerifyFailures.WithLabelValues(reasonSerializationFail).Inc()
		metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
		return false, reasonSerializationFail
	}
	msg := append([]byte("AGENTAUTH_ROTATION_SUMMARY:"), enc...)
	sigBytes, err := base64.RawURLEncoding.DecodeString(sum.Signature)
	if err != nil {
		rotationSignatureVerifyFailures.WithLabelValues(reasonSignatureInvalid).Inc()
		metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
		return false, reasonSignatureInvalid
	}
	if !ed25519.Verify(pub, msg, sigBytes) {
		rotationSignatureVerifyFailures.WithLabelValues(reasonSignatureInvalid).Inc()
		metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
		return false, reasonSignatureInvalid
	}
	metrics.RotationSignatureVerifyLatency.Observe(time.Since(start).Seconds())
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
	}{
		ChainLength:   sum.ChainLength,
		HeadHash:      sum.HeadHash,
		AggregateHash: sum.AggregateHash,
		GeneratedAt:   sum.GeneratedAt,
	}
	return json.Marshal(payload)
}
