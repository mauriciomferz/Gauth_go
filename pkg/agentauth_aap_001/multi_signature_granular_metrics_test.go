package agentauth_aap_001

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	cr "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// sign helper
func msSign(t *testing.T, poa *PowerOfAttorney, priv ed25519.PrivateKey, keyID string) *POASignature {
	dig, canon, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("canonical digest: %v", err)
	}
	sig := ed25519.Sign(priv, canon)
	return &POASignature{Algorithm: algEd25519, KeyID: keyID, DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig)}
}

type msGranProvider struct{ pubs map[string]ed25519.PublicKey }

func (p *msGranProvider) PublicKey(k string) ([]byte, string, error) {
	if pk, ok := p.pubs[k]; ok {
		return pk, algEd25519, nil
	}
	return nil, "", fmt.Errorf("unknown key %s", k)
}
func (p *msGranProvider) ActiveSigner() (cr.Signer, error) {
	return nil, fmt.Errorf("no active signer")
}
func (p *msGranProvider) VerifyWith(msg, sig []byte, keyID string) error {
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

// TestMultiSignatureGranularCounters verifies granular counters increment along distinct failure paths.
func TestMultiSignatureGranularCounters(t *testing.T) {
	// Ensure weighted mode disabled so threshold logic uses simple counts
	t.Setenv("AGENTAUTH_MULTI_SIG_WEIGHTS", "")
	mem := imetrics.NewMemory()
	// Generate keys for alice,bob
	pubA, privA, _ := ed25519.GenerateKey(rand.Reader)
	pubB, privB, _ := ed25519.GenerateKey(rand.Reader)
	hashA := sha256.Sum256(pubA)
	keyIDA := hex.EncodeToString(hashA[:6])
	hashB := sha256.Sum256(pubB)
	keyIDB := hex.EncodeToString(hashB[:6])
	provider := &msGranProvider{pubs: map[string]ed25519.PublicKey{keyIDA: pubA, keyIDB: pubB}}
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "allow_create", Subject: "*", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(nil, ma, WithMetrics(mem), WithKeyProvider(provider), WithStrictAuthenticity())

	// SUCCESS path threshold=2
	poaSuccess := &PowerOfAttorney{ID: "g1", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob"}, Threshold: 2, MultiSignatures: map[string]*POASignature{}}
	poaSuccess.MultiSignatures["alice"] = msSign(t, poaSuccess, privA, keyIDA)
	poaSuccess.MultiSignatures["bob"] = msSign(t, poaSuccess, privB, keyIDB)
	if err := svc.verifyMultiSignatures(poaSuccess); err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if poaSuccess.SatisfiedSignatures != 2 {
		t.Fatalf("expected satisfied signatures=2 got %d", poaSuccess.SatisfiedSignatures)
	}

	// FAILURE: threshold not met
	poaFail := &PowerOfAttorney{ID: "g2", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob", "carol"}, Threshold: 3, MultiSignatures: map[string]*POASignature{}}
	poaFail.MultiSignatures["alice"] = msSign(t, poaFail, privA, keyIDA)
	poaFail.MultiSignatures["bob"] = msSign(t, poaFail, privB, keyIDB)
	if err := svc.verifyMultiSignatures(poaFail); err == nil {
		t.Fatalf("expected threshold failure")
	}

	snap := mem.SnapshotEx()
	// Structural failure expected because len(MultiSignatures) (2) < threshold (3), triggers structural failure path not threshold failure.
	if snap.MultiSignatureVerifications != 1 {
		t.Fatalf("expected 1 success verification got %d", snap.MultiSignatureVerifications)
	}
	if snap.MultiSignatureStructuralFailures != 1 {
		t.Fatalf("expected 1 structural failure got %d", snap.MultiSignatureStructuralFailures)
	}
	if snap.MultiSignatureThresholdFailures != 0 {
		t.Fatalf("expected 0 threshold failures got %d", snap.MultiSignatureThresholdFailures)
	}
}
