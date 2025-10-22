package rfc0111

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"testing"
	"time"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	cr "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/crypto"
)

// weightedKeyProvider exposes public keys only (no active signer needed for verification path).
type weightedKeyProvider struct{ pubs map[string]ed25519.PublicKey }

func (p *weightedKeyProvider) ActiveSigner() (cr.Signer, error) { return nil, nil }
func (p *weightedKeyProvider) PublicKey(keyID string) ([]byte, string, error) {
	pk, ok := p.pubs[keyID]
	if !ok {
		return nil, "", cr.ErrUnknownKey
	}
	return pk, cr.AlgoEd25519, nil
}

func (p *weightedKeyProvider) VerifyWith(msg, sig []byte, keyID string) error {
	pk, ok := p.pubs[keyID]
	if !ok {
		return cr.ErrUnknownKey
	}
	if len(sig) != ed25519.SignatureSize {
		return ErrInvalidSignatureLength
	}
	if !ed25519.Verify(pk, msg, sig) {
		return ErrInvalidSignature
	}
	return nil
}

// local errors for clarity
var (
	ErrInvalidSignatureLength = rfcErr("invalid signature length")
	ErrInvalidSignature       = rfcErr("invalid signature")
)

type rfcErr string

func (e rfcErr) Error() string { return string(e) }

// mkSig helper
func mkSig(t *testing.T, poa *PowerOfAttorney, priv ed25519.PrivateKey, keyID string) *POASignature {
	dig, canon, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("canonical digest: %v", err)
	}
	sig := ed25519.Sign(priv, canon)
	return &POASignature{Algorithm: cr.AlgoEd25519, KeyID: keyID, DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig)}
}

func genKeyIDWeighted(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:6])
}

// TestMultiSignatureWeightedMode exercises GAUTH_MULTI_SIG_WEIGHTS parsing & success/failure semantics.
// Scenario:
//
//	Signers: alice,bob,carol with weights alice=2,bob=1,carol=3 (total=6). Threshold interpretations:
//	 * Success case threshold=3 with signatures from alice (2) + bob (1) => cumulative weight 3 >= 3
//	 * Failure case threshold=5 with signatures from alice (2) + bob (1) => cumulative weight 3 < 5
//
// Structural guard relaxed (len(signatures) < threshold allowed) in weighted mode.
func TestMultiSignatureWeightedMode(t *testing.T) {
	// Enable weighted mode
	os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", "alice=2,bob=1,carol=3,dave=4")
	mem := imetrics.NewMemory()
	// Generate keys
	pubA, privA, _ := ed25519.GenerateKey(rand.Reader)
	pubB, privB, _ := ed25519.GenerateKey(rand.Reader)
	pubC, _, _ := ed25519.GenerateKey(rand.Reader)
	keyIDA := genKeyIDWeighted(pubA)
	keyIDB := genKeyIDWeighted(pubB)
	keyIDC := genKeyIDWeighted(pubC)
	kp := &weightedKeyProvider{pubs: map[string]ed25519.PublicKey{keyIDA: pubA, keyIDB: pubB, keyIDC: pubC}}
	svc := NewService(nil, authzAllowAll{}, WithMetrics(mem), WithKeyProvider(kp), WithStrictAuthenticity())

	// SUCCESS: threshold=3 (weight) with alice + bob (weight=3). Signers list length >= threshold to satisfy structural validation.
	poaSuccess := &PowerOfAttorney{ID: "w1", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob", "carol", "dave"}, Threshold: 3, MultiSignatures: map[string]*POASignature{}}
	poaSuccess.MultiSignatures["alice"] = mkSig(t, poaSuccess, privA, keyIDA)
	poaSuccess.MultiSignatures["bob"] = mkSig(t, poaSuccess, privB, keyIDB)
	if err := svc.verifyMultiSignatures(poaSuccess); err != nil {
		t.Fatalf("weighted success expected got error: %v", err)
	}
	if poaSuccess.SatisfiedWeight < 3 {
		t.Fatalf("expected satisfied weight >=3 got %d", poaSuccess.SatisfiedWeight)
	}

	// FAILURE: threshold=4 with only alice+bob weight=3 (<4). Additional signers carol,dave omitted (no signatures) to test weight failure.
	poaFail := &PowerOfAttorney{ID: "w2", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob", "carol", "dave"}, Threshold: 4, MultiSignatures: map[string]*POASignature{}}
	poaFail.MultiSignatures["alice"] = mkSig(t, poaFail, privA, keyIDA)
	poaFail.MultiSignatures["bob"] = mkSig(t, poaFail, privB, keyIDB)
	errFail := svc.verifyMultiSignatures(poaFail)
	if errFail == nil {
		t.Fatalf("expected weighted insufficient failure")
	}

	snap := mem.SnapshotEx()
	if snap.MultiSignatureVerifications == 0 {
		t.Fatalf("expected at least 1 verification success (got %d successes)", snap.MultiSignatureVerifications)
	}
	if snap.MultiSignatureWeightFailures == 0 {
		t.Fatalf("expected weight failure counter increment (weight_failures=%d struct_fail=%d threshold_fail=%d verification_failures=%d err=%v)", snap.MultiSignatureWeightFailures, snap.MultiSignatureStructuralFailures, snap.MultiSignatureThresholdFailures, snap.MultiSignatureVerificationFailures, errFail)
	}
}
