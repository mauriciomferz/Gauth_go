package attest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// AttestationProof provides a signed assertion about a subject and statement.
// Signature covers the canonical serialization of the proof excluding signature-related fields.
// Version is currently fixed to "att/v1".
// DigestHex is SHA-256 of canonical bytes (future hash agility may add algo field).
// Raw chain reference fields are optional and allow binding proof to a delegation chain snapshot.
// Nonce is a UUID v4 (optional) for replay protection / uniqueness.
// Statement SHOULD be short (<=1024 bytes) textual description or a reference (e.g. hash URI).
// Subject identifies entity the statement pertains to.
// Issuer identifies entity producing the attestation.
// ExpiresAt optional; zero value indicates no explicit expiry.
// Algorithm + KeyID identify signature scheme and public key.
// Signature is base64 encoded.
// CanonicalDigest duplicates DigestHex for explicit naming alignment with other components.
// Future: add Advice / Evidence fields.
type AttestationProof struct {
	Version              string    `json:"ver"`
	Statement            string    `json:"stmt"`
	Subject              string    `json:"sub"`
	Issuer               string    `json:"iss"`
	IssuedAt             time.Time `json:"iat"`
	ExpiresAt            time.Time `json:"exp,omitempty"`
	Nonce                string    `json:"nonce,omitempty"`
	DigestHex            string    `json:"dig"`
	Algorithm            string    `json:"alg"`
	KeyID                string    `json:"kid"`
	Signature            string    `json:"sig"`
	RawPOAChainHash      string    `json:"chain_hash,omitempty"`
	RawPOAChainAlgo      string    `json:"chain_hash_alg,omitempty"`
	CanonicalDigest      string    `json:"canonical_digest,omitempty"`
}

// CanonicalAttestationDigest returns hex digest and canonical JSON bytes for signing.
// It performs deterministic key ordering and excludes signature-bearing fields (dig, sig).
func CanonicalAttestationDigest(p *AttestationProof) (string, []byte, error) {
	if p == nil {
		return "", nil, errors.New("nil attestation proof")
	}
	if p.Version == "" {
		return "", nil, errors.New("missing version")
	}
	if p.Subject == "" || p.Issuer == "" || p.Statement == "" {
		return "", nil, errors.New("missing mandatory fields")
	}
	// Build a map omitting signature-bearing fields to avoid circularity.
	m := map[string]interface{}{
		"ver":   p.Version,
		"stmt":  p.Statement,
		"sub":   p.Subject,
		"iss":   p.Issuer,
		"iat":   p.IssuedAt.UTC().Format(time.RFC3339Nano),
	}
	if !p.ExpiresAt.IsZero() { m["exp"] = p.ExpiresAt.UTC().Format(time.RFC3339Nano) }
	if p.Nonce != "" { m["nonce"] = p.Nonce }
	if p.RawPOAChainHash != "" { m["chain_hash"] = p.RawPOAChainHash }
	if p.RawPOAChainAlgo != "" { m["chain_hash_alg"] = p.RawPOAChainAlgo }
	// Deterministic ordering by sorting keys then encoding manually.
	keys := make([]string, 0, len(m))
	for k := range m { keys = append(keys, k) }
	sort.Strings(keys)
	// Manual minimal JSON encoding to guarantee ordering.
	buf := make([]byte, 0, 256)
	buf = append(buf, '{')
	for i, k := range keys {
		// Marshal value using json for safety (strings/time strings already sanitized).
		vb, err := json.Marshal(m[k])
		if err != nil { return "", nil, fmt.Errorf("marshal value %s: %w", k, err) }
		// Append key:
		buf = append(buf, '"')
		buf = append(buf, k...)
		buf = append(buf, '"', ':')
		buf = append(buf, vb...)
		if i < len(keys)-1 { buf = append(buf, ',') }
	}
	buf = append(buf, '}')
	dig := sha256.Sum256(buf)
	return hex.EncodeToString(dig[:]), buf, nil
}
