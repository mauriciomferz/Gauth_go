package attest

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	internalCrypto "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
)

// AttestationService provides construction + signing helpers for model limits attestation.
// Extraction target for logic currently embedded in web/server_clean.go (RB7).
// It intentionally avoids importing web types to prevent cycles; callers supply/consume
// simple DTO structs (canonical JSON already prepared).
type AttestationService struct{}

// NewAttestationService constructs a new service (future: inject replay store, notarizer, tracer).
func NewAttestationService() *AttestationService { return &AttestationService{} }

// BuildUnsigned attaches snapshot metadata & nonce; caller provides base struct pointer with fields populated.
// Returns canonical JSON bytes ready for signing.
func (s *AttestationService) BuildUnsigned(canonicalJSON []byte, nonce string) ([]byte, string) {
    if nonce == "" { nonce = randomNonce(14) }
    // Simple domain separation via returning nonce (caller embeds before marshal if needed).
    return canonicalJSON, nonce
}

// SignDomainSeparated performs domain-separated signing of raw canonical JSON bytes with provided prefix.
// Public method to allow direct usage without intermediate struct binding.
func (s *AttestationService) SignDomainSeparated(prefix string, raw []byte) ([]byte, string, error) {
    return SignDomainSeparated(prefix, raw)
}

// SignDomainSeparatedB64 returns base64 signature form.
func (s *AttestationService) SignDomainSeparatedB64(prefix string, raw []byte) (string, string, error) {
    return SignDomainSeparatedB64(prefix, raw)
}

// StampNotarization placeholder performing external notarization (future injection).
func (s *AttestationService) StampNotarization(hash string) (provider string, latency time.Duration, err error) {
    return "noop", 0, nil
}

// ModelLimitsNotarizeSignResult captures signature + optional notarization output for model limits attestations.
type ModelLimitsNotarizeSignResult struct {
    Signature    string  // base64 raw url signature (empty if signing disabled or error)
    SigKid       string  // key id used (empty if signing disabled)
    SigMode      string  // algorithm label (eddsa)
    Notarization *struct {
        Provider       string  `json:"provider"`
        Timestamp      string  `json:"timestamp"`
        LatencySeconds float64 `json:"latency_seconds"`
        Success        bool    `json:"success"`
    } `json:"notarization,omitempty"`
    // DomainSignature and DomainPrefix added for migration phase 2 (optional dual signature to enable
    // domain separation without breaking legacy consumers). Present only if env GAUTH_ATTEST_DOMAIN_PREFIX is set
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
    // Prefer agile global rotating signer (RB6). Falls back to direct registry access for backward compatibility.
    signer := internalCrypto.GlobalRotatingSigner()
    // KeyID accessor (optional) for rotating signer.
    type keyIDProvider interface{ KeyID() string }
    primaryPrefix := AttestationDomainPrefix
    if signer != nil {
        msg := append([]byte(primaryPrefix), unsignedJSON...)
        sigBytes, err := signer.Sign(msg)
        if err == nil && len(sigBytes) > 0 {
            res.Signature = base64.RawStdEncoding.EncodeToString(sigBytes)
            if kidp, ok := signer.(keyIDProvider); ok { res.SigKid = kidp.KeyID() }
            res.SigMode = "eddsa" // agility retains eddsa label for now
        }
        // Optional dual domain signature.
        if dp := os.Getenv("GAUTH_ATTEST_DOMAIN_PREFIX"); dp != "" && len(sigBytes) > 0 {
            dmsg := append([]byte(dp), unsignedJSON...)
            if dsig, derr := signer.Sign(dmsg); derr == nil && len(dsig) > 0 {
                res.DomainSignature = base64.RawStdEncoding.EncodeToString(dsig)
                res.DomainPrefix = dp
            }
        }
        // If signer failed to produce signature fall through to legacy path for clarity.
        if res.Signature != "" { return res, nil }
    }
    // Legacy direct registry path.
    if internalCrypto.GlobalEdDSARegistry == nil || internalCrypto.GlobalEdDSARegistry.Active() == nil {
        return res, errors.New("no_active_key")
    }
    active := internalCrypto.GlobalEdDSARegistry.Active()
    if len(active.Private) != ed25519.PrivateKeySize { return res, errors.New("no_private_material") }
    primaryMsg := append([]byte(primaryPrefix), unsignedJSON...)
    primarySig := ed25519.Sign(active.Private, primaryMsg)
    res.Signature = base64.RawStdEncoding.EncodeToString(primarySig)
    res.SigKid = active.ID
    res.SigMode = "eddsa"
    if dp := os.Getenv("GAUTH_ATTEST_DOMAIN_PREFIX"); dp != "" {
        dualMsg := append([]byte(dp), unsignedJSON...)
        dsig := ed25519.Sign(active.Private, dualMsg)
        res.DomainSignature = base64.RawStdEncoding.EncodeToString(dsig)
        res.DomainPrefix = dp
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

// SignAttestation performs domain-separated signing of a model limits attestation using global signer agility.
// If global signer unavailable, falls back to direct ed25519 using active registry key.
// Returns a mutated copy with signature fields populated.
// SignDomainSeparated signs raw canonical JSON bytes with provided domain prefix.
// Returns signature bytes and active key id. Base64 encoding left to caller for flexibility.
func SignDomainSeparated(prefix string, raw []byte) ([]byte, string, error) {
    if internalCrypto.GlobalEdDSARegistry == nil || internalCrypto.GlobalEdDSARegistry.Active() == nil {
        return nil, "", errors.New("no_active_key")
    }
    active := internalCrypto.GlobalEdDSARegistry.Active()
    if len(active.Private) != ed25519.PrivateKeySize { return nil, "", errors.New("no_private_material") }
    msg := append([]byte(prefix), raw...)
    if sig, err := internalCrypto.SignWithGlobal(msg); err == nil {
        return sig, active.ID, nil
    }
    // Fallback direct signature
    sig := ed25519.Sign(active.Private, msg)
    return sig, active.ID, nil
}

// SignDomainSeparatedB64 helper returning base64 raw url encoded signature string.
func SignDomainSeparatedB64(prefix string, raw []byte) (string, string, error) {
    sig, kid, err := SignDomainSeparated(prefix, raw)
    if err != nil { return "", "", err }
    return base64.RawStdEncoding.EncodeToString(sig), kid, nil
}
