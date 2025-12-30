package agentauth_aap_001

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	cr "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// helper to produce signer key id
func genKeyID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:6])
}

// createSignature signs canonical POA digest and returns POASignature
func createSignature(t *testing.T, poa *PowerOfAttorney, keyID string, priv ed25519.PrivateKey) *POASignature {
	t.Helper()
	dig, canon, derr := CanonicalPOADigest(poa)
	if derr != nil {
		t.Fatalf("canonical digest failed: %v", derr)
	}
	sig := ed25519.Sign(priv, canon)
	return &POASignature{Algorithm: algEd25519, KeyID: keyID, DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig)}
}

// mock provider exposing a map of public keys.
type msMetricsProvider struct{ pubs map[string]ed25519.PublicKey }

func (p *msMetricsProvider) PublicKey(keyID string) ([]byte, string, error) {
	pk, ok := p.pubs[keyID]
	if !ok {
		return nil, "", ErrUnknownKey(keyID)
	}
	return pk, algEd25519, nil
}

// ActiveSigner returns nil to indicate no issuance signing capability for this provider in tests.
func (p *msMetricsProvider) ActiveSigner() (cr.Signer, error) {
	return nil, fmt.Errorf("no active signer")
}

// VerifyWith implements KeyProvider extended interface; delegates to ed25519.Verify
func (p *msMetricsProvider) VerifyWith(msg, sig []byte, keyID string) error {
	pk, ok := p.pubs[keyID]
	if !ok {
		return fmt.Errorf("unknown key %s", keyID)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length")
	}
	if !ed25519.Verify(pk, msg, sig) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

// lightweight error type for missing key
type ErrUnknownKey string

func (e ErrUnknownKey) Error() string { return "unknown key " + string(e) }

// TestMultiSignatureMetricsSuccessAndFailure verifies counters increment on success vs threshold failure.
func TestMultiSignatureMetricsSuccessAndFailure(t *testing.T) {
	os.Unsetenv("AGENTAUTH_MULTI_SIG_WEIGHTS")
	mem := imetrics.NewMemory()
	// Generate three keys
	pubA, privA, _ := ed25519.GenerateKey(rand.Reader)
	pubB, privB, _ := ed25519.GenerateKey(rand.Reader)
	pubC, _, _ := ed25519.GenerateKey(rand.Reader)
	keyIDA := genKeyID(pubA)
	keyIDB := genKeyID(pubB)
	keyIDC := genKeyID(pubC)
	provider := &msMetricsProvider{pubs: map[string]ed25519.PublicKey{keyIDA: pubA, keyIDB: pubB, keyIDC: pubC}}
	svc := NewService(nil, authzAllowAll{}, WithMetrics(mem), WithKeyProvider(provider), WithStrictAuthenticity())

	// SUCCESS path: Threshold 2 with two valid signatures
	poaSuccess := &PowerOfAttorney{ID: "ms1", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob", "carol"}, Threshold: 2, MultiSignatures: map[string]*POASignature{}}
	poaSuccess.MultiSignatures["alice"] = createSignature(t, poaSuccess, keyIDA, privA)
	poaSuccess.MultiSignatures["bob"] = createSignature(t, poaSuccess, keyIDB, privB)
	if err := svc.verifyMultiSignatures(poaSuccess); err != nil {
		t.Fatalf("expected success got %v", err)
	}

	// FAILURE path: Threshold 3 but only two signatures -> insufficient threshold
	poaFail := &PowerOfAttorney{ID: "ms2", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob", "carol"}, Threshold: 3, MultiSignatures: map[string]*POASignature{}}
	poaFail.MultiSignatures["alice"] = createSignature(t, poaFail, keyIDA, privA)
	poaFail.MultiSignatures["bob"] = createSignature(t, poaFail, keyIDB, privB)
	if err := svc.verifyMultiSignatures(poaFail); err == nil {
		t.Fatalf("expected threshold failure")
	}

	snap := mem.SnapshotEx()
	if snap.MultiSignatureVerifications != 1 {
		t.Fatalf("expected 1 multi-signature verification success, got %d", snap.MultiSignatureVerifications)
	}
	if snap.MultiSignatureVerificationFailures != 1 {
		t.Fatalf("expected 1 multi-signature verification failure, got %d", snap.MultiSignatureVerificationFailures)
	}
}
