package test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
)

// resolverBoth implements PublicKeyResolver for both Ed25519 and ECDSA.
type resolverBoth struct{ ed ed25519.PublicKey; ec *ecdsa.PublicKey }
func (r *resolverBoth) FindByID(id string) *notary.PublicKeyRecord {
    switch id {
    case "ed1": return &notary.PublicKeyRecord{Ed25519: r.ed}
    case "ec1": return &notary.PublicKeyRecord{ECDSA: r.ec}
    default: return nil
    }
}

// resolverEd implements PublicKeyResolver for only Ed25519 signer.
type resolverEd struct{ ed ed25519.PublicKey }
func (r *resolverEd) FindByID(id string) *notary.PublicKeyRecord { if id == "ed1" { return &notary.PublicKeyRecord{Ed25519: r.ed} }; return nil }

// TestRotationV2MixedAlgVerification exercises verification across Ed25519 + ECDSA-P256.
func TestRotationV2MixedAlgVerification(t *testing.T) {
    // Build config with two signers different algs.
    cfg := &notary.WeightsConfig{SchemaVersion:1, ActiveKeySetID:"mixed-set", ThresholdWeight:70,
        Signers: []struct{ID string `json:"id"`; Alg string `json:"alg,omitempty"`; Weight int `json:"weight"`}{
            {ID:"ed1", Alg:"ED25519", Weight:30}, {ID:"ec1", Alg:"ECDSA-P256", Weight:40},
        }, AlgorithmSuite: []string{"ed25519","ecdsa-p256"}}
    art, err := notary.BuildArtifactFromConfig(cfg, "", time.Now())
    if err != nil { t.Fatalf("artifact build: %v", err) }
    // Generate keys
    pubEd, privEd, _ := ed25519.GenerateKey(rand.Reader)
    privEc, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil { t.Fatalf("ecdsa key: %v", err) }
    // Attach signatures
    if err := notary.AttachEd25519Signature(&art, privEd, "ed1", "ED25519", 30); err != nil { t.Fatalf("attach ed25519: %v", err) }
    if err := notary.AttachECDSASignature(&art, privEc, "ec1", "ECDSA-P256", 40); err != nil { t.Fatalf("attach ecdsa: %v", err) }
    // Resolver with both public keys
    rs := &resolverBoth{ed: pubEd, ec: &privEc.PublicKey}
    verified, _, failures := notary.VerifyArtifactSignatures(&art, rs)
    if verified != 70 { t.Fatalf("expected verified weight 70 got %d failures=%v", verified, failures) }
    if len(failures) != 0 { t.Fatalf("unexpected failures: %v", failures) }
}

// TestRotationV2MixedAlgFailure ensures invalid signature reduces verified weight and records failure.
func TestRotationV2MixedAlgFailure(t *testing.T) {
    cfg := &notary.WeightsConfig{SchemaVersion:1, ActiveKeySetID:"mixed-set", ThresholdWeight:50,
        Signers: []struct{ID string `json:"id"`; Alg string `json:"alg,omitempty"`; Weight int `json:"weight"`}{
            {ID:"ed1", Alg:"ED25519", Weight:30}, {ID:"ec1", Alg:"ECDSA-P256", Weight:20},
        }, AlgorithmSuite: []string{"ed25519","ecdsa-p256"}}
    art, err := notary.BuildArtifactFromConfig(cfg, "", time.Now())
    if err != nil { t.Fatalf("artifact build: %v", err) }
    // Good Ed25519 key
    pubEd, privEd, _ := ed25519.GenerateKey(rand.Reader)
    if err := notary.AttachEd25519Signature(&art, privEd, "ed1", "ED25519", 30); err != nil { t.Fatalf("attach ed25519: %v", err) }
    // Fake ECDSA signature (random bytes) to trigger failure
    fake := make([]byte, 64)
    if _, err := rand.Read(fake); err != nil {
        t.Fatalf("rand.Read failed: %v", err)
    }
    // Assign directly (since ECDSA signer entry exists already)
    for i := range art.Signers { if art.Signers[i].ID == "ec1" { art.Signers[i].Signature = base64.RawURLEncoding.EncodeToString(fake) } }
    rs := &resolverEd{ed: pubEd}
    verified, _, failures := notary.VerifyArtifactSignatures(&art, rs)
    if verified != 30 { t.Fatalf("expected verified weight 30 got %d", verified) }
    // Expect at least one failure reason referencing public_key_not_found or signature_invalid.
    if len(failures) == 0 { t.Fatalf("expected failures") }
    // Ensure ECDSA signer remains present but unverified
    foundEc := false
    for _, s := range art.Signers { if s.ID == "ec1" { foundEc = true; break } }
    if !foundEc { t.Fatalf("missing ec1 signer post verification") }
}
