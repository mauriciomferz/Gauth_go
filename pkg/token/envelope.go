package token

import (
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
)

// Envelope is a draft structured token payload (Milestone 2A scaffold).
// Not yet wired into issuance; future versions will sign + serialize this.
type Envelope struct {
	Version         string            `json:"ver"`
	KeyID           string            `json:"kid,omitempty"`
	DelegationID    string            `json:"delegation_id"`
	Grantor         string            `json:"grantor"`
	Grantee         string            `json:"grantee"`
	Scope           []string          `json:"scope"`
	Restrictions    map[string]string `json:"restrictions,omitempty"`
	Status          string            `json:"status,omitempty"`
	IssuedAt        time.Time         `json:"iat"`
	ExpiresAt       time.Time         `json:"exp"`
	IssuanceChain   string            `json:"iss_chain_tip,omitempty"`
	RevocationChain string            `json:"rev_chain_tip,omitempty"`
	JTI             string            `json:"jti,omitempty"` // unique token identifier (replay protection)
}

// EnvelopeV2 introduces explicit canonical POA digest and satisfied multi-signature metadata.
// It embeds a minimal subset of POA fields required for validation while allowing future
// expansion (e.g., policy versioning) without breaking legacy consumers. Tokens issued
// with GAUTH_POA_ENVELOPE_V2=1 will serialize this structure instead of Envelope.
type EnvelopeV2 struct {
	Version             string            `json:"ver"`
	KeyID               string            `json:"kid,omitempty"`
	DelegationID        string            `json:"delegation_id"`
	Grantor             string            `json:"grantor"`
	Grantee             string            `json:"grantee"`
	Scope               []string          `json:"scope"`
	Restrictions        map[string]string `json:"restrictions,omitempty"`
	Status              string            `json:"status,omitempty"`
	IssuedAt            time.Time         `json:"iat"`
	ExpiresAt           time.Time         `json:"exp"`
	IssuanceChain       string            `json:"iss_chain_tip,omitempty"`
	RevocationChain     string            `json:"rev_chain_tip,omitempty"`
	JTI                 string            `json:"jti,omitempty"`
	CanonicalDigest     string            `json:"canonical_digest,omitempty"` // hex digest of canonical POA representation
	SatisfiedWeight     int               `json:"satisfied_weight,omitempty"`
	SatisfiedSignatures int               `json:"satisfied_signatures,omitempty"`
	// PoAVersion records the schema version of embedded POA serialization (e.g. "poa/v1").
	PoAVersion string `json:"poa_version,omitempty"`
	// RawPOA is a deterministic JSON serialization of the PowerOfAttorney used to derive CanonicalDigest.
	// Provided for auditing and external verification replay. Omitted if size exceeds policy limits.
	RawPOA string `json:"raw_poa,omitempty"`
	// RawPOAChain embeds a minimal CBOR-like streaming representation (length-prefixed map sequence or
	// indefinite-length CBOR array) of the delegation chain snapshot at issuance (prototype: single POA item).
	// Base64 encoded; omitted if embedding disabled or size exceeds limits. Uses hash algorithm negotiated
	// via GAUTH_RAW_POA_CHAIN_HASH_ALGO (default sha256) for PrevHash continuity when multiple items present.
	RawPOAChain string `json:"raw_poa_chain,omitempty"`
	// RawPOAChainAlgo records hashing algorithm used when computing chain continuity ("sha256", "blake2b256", "sha3_256").
	RawPOAChainAlgo string `json:"raw_poa_chain_algo,omitempty"`
	// Detached signature (optional, feature gated by GAUTH_DETACHED_SIGNATURE=1). This is an Ed25519 (or future) signature
	// over the canonical POA JSON bytes (the exact same bytes whose SHA-256 hex is CanonicalDigest). The intent is to
	// provide a publicly verifiable integrity proof decoupled from the embedded (or absent) POA signature object.
	DetachedSignature    string `json:"detached_sig,omitempty"`
	DetachedSignatureAlg string `json:"detached_sig_alg,omitempty"`
	DetachedSignatureKid string `json:"detached_sig_kid,omitempty"`
	// AdvancedClaims provides comprehensive token metadata including typ semantic enforcement, claims set metadata,
	// and extensible restrictions. When present, VerifyToken enforces typ-specific validation rules (e.g. delegation
	// tokens must have valid PoA reference, capability tokens must specify supported operations). Omitted for
	// backward compatibility with tokens issued before P2.10 (sec1.item2 integration).
	AdvancedClaims *gauth.AdvancedClaims `json:"advanced_claims,omitempty"`
}
