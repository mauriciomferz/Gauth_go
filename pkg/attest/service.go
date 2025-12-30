package attest

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/notary"
	internalCrypto "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

const (
	sigModeEdDSA = "eddsa"
)

// AttestationService provides construction + signing helpers for model limits attestation.
// Extraction target for logic currently embedded in web/server_clean.go (RB7).
// It intentionally avoids importing web types to prevent cycles; callers supply/consume
// simple DTO structs (canonical JSON already prepared).
// AttestationService provides construction + signing helpers for model limits attestation.
// Extraction target for logic currently embedded in web/server_clean.go (RB7).
// It intentionally avoids importing web types to prevent cycles; callers supply/consume
// simple DTO structs (canonical JSON already prepared).
type AttestationService struct {
	keyProvider internalCrypto.KeyProvider
}

// Option configures the AttestationService
type Option func(*AttestationService)

// WithKeyProvider injects a crypto provider for signing operations
func WithKeyProvider(kp internalCrypto.KeyProvider) Option {
	return func(s *AttestationService) {
		s.keyProvider = kp
	}
}

// NewAttestationService constructs a new service (future: inject replay store, notarizer, tracer).
func NewAttestationService(opts ...Option) *AttestationService {
	s := &AttestationService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// BuildUnsigned attaches snapshot metadata & nonce; caller provides base struct pointer with fields populated.
// Returns canonical JSON bytes ready for signing.
func (s *AttestationService) BuildUnsigned(canonicalJSON []byte, nonce string) ([]byte, string) {
	if nonce == "" {
		nonce = randomNonce(14)
	}
	// Simple domain separation via returning nonce (caller embeds before marshal if needed).
	return canonicalJSON, nonce
}

// SignDomainSeparated performs domain-separated signing of raw canonical JSON bytes with provided prefix.
// Public method to allow direct usage without intermediate struct binding.
func (s *AttestationService) SignDomainSeparated(prefix string, raw []byte) ([]byte, string, error) {
	kp := s.keyProvider

	if kp == nil {
		return nil, "", errors.New("no_key_provider")
	}
	signer, err := kp.ActiveSigner()
	if err != nil {
		return nil, "", err
	}
	msg := append([]byte(prefix), raw...)
	sig, err := signer.Sign(msg)
	if err != nil {
		return nil, "", err
	}
	return sig, signer.KeyID(), nil
}

// SignDomainSeparatedB64 returns base64 signature form.
func (s *AttestationService) SignDomainSeparatedB64(prefix string, raw []byte) (string, string, error) {
	sig, kid, err := s.SignDomainSeparated(prefix, raw)
	if err != nil {
		return "", "", err
	}
	return base64.RawStdEncoding.EncodeToString(sig), kid, nil
}

// StampNotarization placeholder performing external notarization (future injection).
func (s *AttestationService) StampNotarization(hash string) (provider string, latency time.Duration, err error) {
	return "noop", 0, nil
}

// ModelLimitsNotarizeSignResult captures signature + optional notarization output for model limits attestations.
type ModelLimitsNotarizeSignResult struct {
	Signature    string // base64 raw url signature (empty if signing disabled or error)
	SigKid       string // key id used (empty if signing disabled)
	SigMode      string // algorithm label (eddsa)
	Notarization *struct {
		Provider       string  `json:"provider"`
		Timestamp      string  `json:"timestamp"`
		LatencySeconds float64 `json:"latency_seconds"`
		Success        bool    `json:"success"`
	} `json:"notarization,omitempty"`
	// DomainSignature and DomainPrefix added for migration phase 2 (optional dual signature to enable
	// domain separation without breaking legacy consumers). Present only if env AGENTAUTH_ATTEST_DOMAIN_PREFIX is set
	// at signing time and raw signing enabled.
	DomainSignature string `json:"domain_signature,omitempty"`
	DomainPrefix    string `json:"domain_prefix,omitempty"`
}

// NotarizeAndSignModelLimits performs (optional) notarization + domain-separated signing over the provided unsigned
// attestation JSON representation. The caller is responsible for marshaling the unsigned attestation structure
// excluding signature-bearing fields and passing in contextual hashes. Returns a result struct; failures in
// notarization or signing do not produce hard errors (best-effort) unless active key material is missing.
// seed format: attest|<snapshotHash>|<auditHead>|<anchorHead> (sha256)
func (s *AttestationService) NotarizeAndSignModelLimits(unsignedJSON []byte, snapshotHash, auditHead, anchorHead string, enableNotarization, enableSigning bool, notarizer notary.Notarizer) (ModelLimitsNotarizeSignResult, error) {
	res := ModelLimitsNotarizeSignResult{}
	// Compute combined hash for potential notarization.
	combinedSeed := fmt.Sprintf("attest|%s|%s|%s", snapshotHash, auditHead, anchorHead)
	combinedHashSum := sha256.Sum256([]byte(combinedSeed))
	combinedHash := fmt.Sprintf("sha256:%x", combinedHashSum[:])
	if enableNotarization && notarizer != nil && snapshotHash != "" {
		if receipt, err := notarizer.Notarize(combinedHash); err == nil {
			res.Notarization = &struct {
				Provider       string  `json:"provider"`
				Timestamp      string  `json:"timestamp"`
				LatencySeconds float64 `json:"latency_seconds"`
				Success        bool    `json:"success"`
			}{Provider: receipt.Provider, Timestamp: receipt.Timestamp, LatencySeconds: receipt.LatencySeconds, Success: receipt.Success}
		}
	}
	if !enableSigning {
		return res, nil
	}

	kp := s.keyProvider

	if kp == nil {
		return res, errors.New("no_key_provider")
	}

	signer, err := kp.ActiveSigner()
	if err != nil {
		return res, err
	}

	primaryPrefix := AttestationDomainPrefix
	msg := append([]byte(primaryPrefix), unsignedJSON...)
	sigBytes, err := signer.Sign(msg)
	if err != nil {
		return res, err
	}
	if len(sigBytes) > 0 {
		res.Signature = base64.RawStdEncoding.EncodeToString(sigBytes)
		res.SigKid = signer.KeyID()
		res.SigMode = sigModeEdDSA
	}

	// Optional dual domain signature.
	if dp := os.Getenv("AGENTAUTH_ATTEST_DOMAIN_PREFIX"); dp != "" && len(sigBytes) > 0 {
		dmsg := append([]byte(dp), unsignedJSON...)
		if dsig, derr := signer.Sign(dmsg); derr == nil && len(dsig) > 0 {
			res.DomainSignature = base64.RawStdEncoding.EncodeToString(dsig)
			res.DomainPrefix = dp
		}
	}
	return res, nil
}

// MarshalUnsignedModelLimits produces deterministic unsigned JSON for signing from a generic object representing
// the attestation without signature-bearing fields. It is a convenience wrapper used by callers extracting embedded logic.
func (s *AttestationService) MarshalUnsignedModelLimits(v interface{}) ([]byte, error) {
	// We rely on standard json.Marshal for this phase; future canonicalization can be injected if necessary.
	return json.Marshal(v)
}

// CanonicalizeModelLimitsUnsigned produces a deterministic JSON encoding of the unsigned attestation.
// It purposely strips all signature-bearing fields (signature, sig_kid, sig_mode, domain_signature, domain_prefix)
// if the passed value is a map or struct containing those keys/fields. This allows tests and callers to reuse
// the exact same canonicalization logic and avoid drift when new optional signature fields are introduced.
// Future enhancement: switch to explicit field ordering with a custom encoder if Go's map iteration order
// (already deterministic for struct fields) becomes insufficient.
func (s *AttestationService) CanonicalizeModelLimitsUnsigned(raw []byte) ([]byte, error) {
	// Attempt fast-path: unmarshal into generic map to drop fields, then re-marshal.
	var gen map[string]any
	if err := json.Unmarshal(raw, &gen); err != nil {
		return nil, err
	}
	delete(gen, "signature")
	delete(gen, "sig_kid")
	delete(gen, "sig_mode")
	delete(gen, "domain_signature")
	delete(gen, "domain_prefix")
	return json.Marshal(gen)
}

// randomNonce local minimal entropy helper (duplicated to avoid import cycle with web package)
func randomNonce(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	// naive timestamp-mix fallback; cryptographic randomness not required for demo nonce
	for i := range b {
		b[i] = chars[int(time.Now().UnixNano()+int64(i))%len(chars)]
	}
	return string(b)
}

// Deprecated functions removed.
