package attest

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	internalCrypto "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
)

// simpleFakeRegistry provides a minimal KeyFinder for fuzzing.
type simpleFakeRegistry struct{ pub ed25519.PublicKey }
func (s *simpleFakeRegistry) FindByID(id string) *internalCrypto.Key { return &internalCrypto.Key{Public: s.pub, ID: "fuzz"} }

// fuzzReplayStrategy stores last N nonces (in-memory ring) to exercise replay logic without unbounded growth.
type fuzzReplayStrategy struct{ seen map[string]struct{} }
func (f *fuzzReplayStrategy) Seen(nonce string) bool { _, ok := f.seen[nonce]; return ok }
func (f *fuzzReplayStrategy) Record(nonce string) { if len(f.seen) > 2048 { for k := range f.seen { delete(f.seen, k); break } }; f.seen[nonce] = struct{}{} }

// FuzzVerifyModelLimitsAttestation exercises attestation verification over arbitrary JSON inputs.
// Seeds include minimally valid and domain-signature variants.
func FuzzVerifyModelLimitsAttestation(f *testing.F) {
    pub, priv, _ := ed25519.GenerateKey(nil)
    // Build minimal valid attestation seed
    base := Attestation{Success: true, Configured: true, SigKid: "fuzz", SigMode: "eddsa"}
    base.Snapshot.Hash = "sha256:deadbeef"; base.Snapshot.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)
    unsignedStruct := struct { Success bool `json:"success"`; Configured bool `json:"configured"`; Snapshot struct { Hash string `json:"hash"`; GeneratedAt string `json:"generated_at"` } `json:"snapshot"` }{Success: base.Success, Configured: base.Configured, Snapshot: base.Snapshot}
    rawUnsigned, _ := json.Marshal(unsignedStruct)
    primaryMsg := append([]byte(AttestationDomainPrefix), rawUnsigned...)
    sig := ed25519.Sign(priv, primaryMsg)
    base.Signature = base64.RawStdEncoding.EncodeToString(sig)
    validSeed, _ := json.Marshal(base)
    f.Add(validSeed)
    // Domain signature variant
    base.DomainPrefix = "GAUTH_FUZZ_PREFIX:"
    dmsg := append([]byte(base.DomainPrefix), rawUnsigned...)
    dsig := ed25519.Sign(priv, dmsg)
    base.DomainSignature = base64.RawStdEncoding.EncodeToString(dsig)
    dualSeed, _ := json.Marshal(base)
    f.Add(dualSeed)
    // Missing nonce seed (should exercise nonce_missing path when replay strategy expects nonce presence)
    noNonce := base; noNonce.Nonce = ""; missingNonceSeed, _ := json.Marshal(noNonce); f.Add(missingNonceSeed)
    // Invalid primary signature base64 seed
    badSig := base; badSig.Signature = "@@not_base64@@"; badSigB, _ := json.Marshal(badSig); f.Add(badSigB)
    // Inconsistent notarization (Success=false)
    inc := base; inc.Notarization = &struct { Provider string `json:"provider"`; Timestamp string `json:"timestamp"`; LatencySeconds float64 `json:"latency_seconds"`; Success bool `json:"success"` }{Provider: "mem", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), LatencySeconds: 0.001, Success: false}
    incSeed, _ := json.Marshal(inc); f.Add(incSeed)
    reg := &simpleFakeRegistry{pub: pub}
    replay := &fuzzReplayStrategy{seen: map[string]struct{}{}}
    f.Fuzz(func(t *testing.T, data []byte) {
        // Attempt to interpret fuzz data as Attestation
        var att Attestation
        if json.Unmarshal(data, &att) != nil {
            return
        }
        // Limit size of variable fields to avoid extreme memory consumption
        if len(att.DomainPrefix) > 256 { att.DomainPrefix = att.DomainPrefix[:256] }
        VerifyModelLimitsAttestation(&att, reg, replay, time.Now())
    })
}
