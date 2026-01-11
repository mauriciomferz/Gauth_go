package agentauth_aap_001

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	cr "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// helper creates an ed25519 signature over canonical digest of poa using provided private key & keyID
func signPOA(t *testing.T, poa *PowerOfAttorney, keyID string, priv ed25519.PrivateKey) *POASignature {
	t.Helper()
	dig, canon, derr := CanonicalPOADigest(poa)
	if derr != nil {
		t.Fatalf("canonical digest failed: %v", derr)
	}
	sig := ed25519.Sign(priv, canon)
	return &POASignature{Algorithm: algEd25519, KeyID: keyID, DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig)}
}

// testProvider implements crypto.KeyProvider for tests (local, multi key support)
type testProvider struct {
	active cr.Signer
	pubs   map[string]ed25519.PublicKey
}

func (p *testProvider) ActiveSigner() (cr.Signer, error) {
	if p.active == nil {
		return nil, fmt.Errorf("no active")
	}
	return p.active, nil
}
func (p *testProvider) PublicKey(keyID string) ([]byte, string, error) {
	pk, ok := p.pubs[keyID]
	if !ok {
		return nil, "", fmt.Errorf("unknown key %s", keyID)
	}
	return pk, cr.AlgoEd25519, nil
}

func (p *testProvider) VerifyWith(msg, sig []byte, keyID string) error {
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

type testSigner struct {
	keyID string
	priv  ed25519.PrivateKey
}

func (s *testSigner) KeyID() string                   { return s.keyID }
func (s *testSigner) Algorithm() string               { return cr.AlgoEd25519 }
func (s *testSigner) Public() []byte                  { return s.priv.Public().(ed25519.PublicKey) }
func (s *testSigner) Sign(msg []byte) ([]byte, error) { return ed25519.Sign(s.priv, msg), nil }
func (s *testSigner) Verify(msg, sig []byte) bool {
	pub := s.priv.Public().(ed25519.PublicKey)
	return ed25519.Verify(pub, msg, sig)
}

func genKey(t *testing.T) (priv ed25519.PrivateKey, pub ed25519.PublicKey, keyID string) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	h := sha256.Sum256(pub)
	keyID = hex.EncodeToString(h[:6])
	return priv, pub, keyID
}

func TestVerifyMultiSignaturesThresholdSuccess(t *testing.T) {
	t.Setenv("AGENTAUTH_MULTI_SIG_WEIGHTS", "")
	poa := &PowerOfAttorney{ID: "t1", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: nowUTC(), ValidUntil: nowUTC().Add(time.Hour), CreatedAt: nowUTC(), Signers: []string{"alice", "bob", "carol"}, Threshold: 2, MultiSignatures: map[string]*POASignature{}}
	// Build three providers (simulate distinct signer keys) - capture priv keys & keyIDs
	privA, pubA, keyIDA := genKey(t)
	privB, pubB, keyIDB := genKey(t)
	_, pubC, keyIDC := genKey(t)
	prov := &testProvider{pubs: map[string]ed25519.PublicKey{keyIDA: pubA, keyIDB: pubB, keyIDC: pubC}, active: &testSigner{keyID: keyIDA, priv: privA}}
	svc := NewService(nil, authzAllowAll{}, WithKeyProvider(prov), WithStrictAuthenticity())
	poa.MultiSignatures["alice"] = signPOA(t, poa, keyIDA, privA)
	poa.MultiSignatures["bob"] = signPOA(t, poa, keyIDB, privB)
	if err := svc.verifyMultiSignatures(poa); err != nil {
		t.Fatalf("expected success got error: %v", err)
	}
}

func TestVerifyMultiSignaturesThresholdInsufficient(t *testing.T) {
	t.Setenv("AGENTAUTH_MULTI_SIG_WEIGHTS", "")
	poa := &PowerOfAttorney{ID: "t2", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: nowUTC(), ValidUntil: nowUTC().Add(time.Hour), CreatedAt: nowUTC(), Signers: []string{"alice", "bob", "carol"}, Threshold: 3, MultiSignatures: map[string]*POASignature{}}
	privA, pubA, keyIDA := genKey(t)
	privB, pubB, keyIDB := genKey(t)
	_, pubC, keyIDC := genKey(t)
	prov := &testProvider{pubs: map[string]ed25519.PublicKey{keyIDA: pubA, keyIDB: pubB, keyIDC: pubC}, active: &testSigner{keyID: keyIDA, priv: privA}}
	svc := NewService(nil, authzAllowAll{}, WithKeyProvider(prov), WithStrictAuthenticity())
	poa.MultiSignatures["alice"] = signPOA(t, poa, keyIDA, privA)
	poa.MultiSignatures["bob"] = signPOA(t, poa, keyIDB, privB)
	if err := svc.verifyMultiSignatures(poa); err == nil {
		t.Fatalf("expected insufficient threshold error")
	}
}

func TestVerifyMultiSignaturesInvalidSignature(t *testing.T) {
	t.Setenv("AGENTAUTH_MULTI_SIG_WEIGHTS", "")
	poa := &PowerOfAttorney{ID: "t3", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: nowUTC(), ValidUntil: nowUTC().Add(time.Hour), CreatedAt: nowUTC(), Signers: []string{"alice", "bob"}, Threshold: 2, MultiSignatures: map[string]*POASignature{}}
	privA, pubA, keyIDA := genKey(t)
	privB, pubB, keyIDB := genKey(t)
	prov := &testProvider{pubs: map[string]ed25519.PublicKey{keyIDA: pubA, keyIDB: pubB}, active: &testSigner{keyID: keyIDA, priv: privA}}
	svc := NewService(nil, authzAllowAll{}, WithKeyProvider(prov), WithStrictAuthenticity())
	poa.MultiSignatures["alice"] = signPOA(t, poa, keyIDA, privA)
	// tamper second signature: sign altered canonical (scope mutated)
	originalScope := poa.Scope
	poa.Scope = []string{"write"}
	digTamper, canonTamper, derr := CanonicalPOADigest(poa)
	if derr != nil {
		t.Fatalf("canonical digest failed tamper: %v", derr)
	}
	sigBad := ed25519.Sign(privB, canonTamper)
	poa.Scope = originalScope // revert after tamper signature created
	poa.MultiSignatures["bob"] = &POASignature{Algorithm: algEd25519, KeyID: keyIDB, DigestHex: digTamper, SigBase64: base64.StdEncoding.EncodeToString(sigBad)}
	if err := svc.verifyMultiSignatures(poa); err == nil {
		t.Fatalf("expected invalid signature digest mismatch error")
	}
}

// nowUTC helper
func nowUTC() time.Time { return time.Now().UTC() }

// authzAllowAll stub
type authzAllowAll struct{}

func (a authzAllowAll) Authorize(ctx context.Context, r authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: true}, nil
}

func (a authzAllowAll) GetPermissions(ctx context.Context, subject string) ([]authz.Permission, error) {
	return []authz.Permission{{Resource: "*", Actions: []string{"*"}, Granted: true}}, nil
}
