package rfc0111

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	cr "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/crypto"
)

// helper constructing service with signer + key provider and mandatory signatures
func newStrictService(t *testing.T) (*Service, cr.Signer) {
	t.Helper()
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("key provider: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "grantor", Resource: "*", Actions: []string{"create_delegation", "revoke_delegation"}, Effect: authz.Allow})
	signerFn := kp.ActiveSigner
	svc := NewService(auditLogger, authorizer, WithSignerProvider(signerFn), WithKeyProvider(kp), WithMandatorySignatures(), WithStrictAuthenticity())
	s, _ := signerFn()
	return svc, s
}

// issueSignedDelegation creates a delegation and ensures it is signed.
func issueSignedDelegation(t *testing.T, svc *Service, grantor, grantee string) *PowerOfAttorney {
	t.Helper()
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: grantor, Grantee: grantee, Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	poa, _ := svc.repo.Get(resp.POA.ID)
	if poa.Signature == nil {
		t.Fatalf("expected signature produced")
	}
	return poa
}

// TestSignatureNegativeCases exercises failure modes of verifyPOASignature.
func TestSignatureNegativeCases(t *testing.T) {
	svc, signer := newStrictService(t)
	poa := issueSignedDelegation(t, svc, "grantor", "grantee")
	// Baseline success
	if err := svc.verifyPOASignature(poa); err != nil {
		t.Fatalf("baseline verification failed: %v", err)
	}

	// 1. Tamper digest -> expect integrity failure digest mismatch
	origDig := poa.Signature.DigestHex
	poa.Signature.DigestHex = "deadbeef"
	if err := svc.verifyPOASignature(poa); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch integrity failure got %v", err)
	}
	poa.Signature.DigestHex = origDig

	// 2. Wrong key: generate new key pair and sign canonical with different key id
	dig, canon, derr := CanonicalPOADigest(poa)
	if derr != nil {
		t.Fatalf("canonical: %v", derr)
	}
	// produce alternate key signer
	altKP, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("alt provider: %v", err)
	}
	altSigner, _ := altKP.ActiveSigner()
	altSig, serr := altSigner.Sign(canon)
	if serr != nil {
		t.Fatalf("alt sign: %v", serr)
	}
	poa.Signature = &POASignature{Algorithm: altSigner.Algorithm(), KeyID: altSigner.KeyID(), DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(altSig), Canonical: canon}
	// Service does not know alt key -> strictAuthenticity should raise integrity failure public key missing.
	err = svc.verifyPOASignature(poa)
	if err == nil || !strings.Contains(err.Error(), "signature public key missing") {
		t.Fatalf("expected integrity failure missing public key got %v", err)
	}

	// Restore valid signature
	sig, serr2 := signer.Sign(canon)
	if serr2 != nil {
		t.Fatalf("sign restore: %v", serr2)
	}
	poa.Signature = &POASignature{Algorithm: signer.Algorithm(), KeyID: signer.KeyID(), DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig), Canonical: canon}

	// 3. Corrupt signature bytes (truncate) -> generic signature verification failed via registry (encoding error)
	poa.Signature.SigBase64 = poa.Signature.SigBase64[:10]
	if err := svc.verifyPOASignature(poa); err == nil || !strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("expected integrity failure signature verification failed got %v", err)
	}

	// 4. Modify canonical content secretly (simulate mutation after signing) -> signature verification failed
	// We simulate by recomputing digest for mutated POA but not updating signature bytes (so digest mismatch already tested). Instead mutate CreatedAt and keep old digest.
	sigValidPOA := issueSignedDelegation(t, svc, "grantor", "grantee2")
	if err := svc.verifyPOASignature(sigValidPOA); err != nil {
		t.Fatalf("pre mutate verify: %v", err)
	}
	sigValidPOA.CreatedAt = sigValidPOA.CreatedAt.Add(time.Hour) // canonical excludes UpdatedAt but includes created_at so this changes digest
	// Now verification should fail at digest mismatch path
	if err := svc.verifyPOASignature(sigValidPOA); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch after created_at mutation got %v", err)
	}
}

// (No custom RFC error extraction needed; we assert via substring matching in error messages.)
