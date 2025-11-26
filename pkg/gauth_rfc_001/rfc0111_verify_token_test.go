package gauth_rfc_001

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	cr "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// helper to create service with key provider and signer
func newServiceWithSigner(t *testing.T) (*Service, cr.Signer) {
	t.Helper()
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("key provider: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "grantor", Resource: "*", Actions: []string{"create_delegation", "revoke_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))
	return svc, func() cr.Signer { s, _ := kp.ActiveSigner(); return s }()
}

func issueDelegation(t *testing.T, svc *Service, grantor, grantee string, dur time.Duration) *DelegationResponse {
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: grantor, Grantee: grantee, Scope: []string{"read", "write"}, Duration: dur})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	return resp
}

func TestVerifyTokenValid(t *testing.T) {
	svc, _ := newServiceWithSigner(t)
	resp := issueDelegation(t, svc, "grantor", "grantee", time.Minute)
	// Basic verify
	res, err := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if res.DelegationID != resp.POA.ID || res.Expired || res.Revoked {
		t.Fatalf("unexpected result %#v", res)
	}
}

func TestVerifyTokenExpired(t *testing.T) {
	svc, _ := newServiceWithSigner(t)
	svc.WithClock(func() time.Time { return time.Unix(0, 0) })
	resp := issueDelegation(t, svc, "grantor", "grantee", time.Second)
	// Advance clock beyond expiry
	svc.WithClock(func() time.Time { return time.Unix(10, 0) })
	res, err := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
	if err == nil || res == nil || !res.Expired {
		t.Fatalf("expected expired error got res=%#v err=%v", res, err)
	}
}

func TestVerifyTokenRevoked(t *testing.T) {
	svc, _ := newServiceWithSigner(t)
	resp := issueDelegation(t, svc, "grantor", "grantee", time.Minute)
	if err := svc.RevokeDelegation(resp.POA.ID, "grantor"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	res, err := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
	if err == nil || res == nil || !res.Revoked {
		t.Fatalf("expected revoked error res=%#v err=%v", res, err)
	}
}

func TestVerifyTokenMissingPOA(t *testing.T) {
	svc, _ := newServiceWithSigner(t)
	// Token from delegation then delete from repo
	resp := issueDelegation(t, svc, "grantor", "grantee", time.Minute)
	// simulate removal
	delete(svc.repo.(*memoryRepository).store, resp.POA.ID)
	_, err := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
	if err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestVerifyTokenDigestTamper(t *testing.T) {
	svc, _ := newServiceWithSigner(t)
	resp := issueDelegation(t, svc, "grantor", "grantee", time.Minute)
	// Tamper stored POA (change scope) causing digest mismatch
	stored, _ := svc.repo.Get(resp.POA.ID)
	stored.Scope = []string{"different"}
	_, err := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
	if err == nil {
		t.Fatalf("expected integrity failure on digest mismatch")
	}
}

func TestVerifyTokenSubjectMismatch(t *testing.T) {
	svc, _ := newServiceWithSigner(t)
	resp := issueDelegation(t, svc, "grantor", "grantee", time.Minute)
	_, err := svc.VerifyToken(WithSubject(context.Background(), "other"), resp.AuthToken)
	if err == nil {
		t.Fatalf("expected unauthorized subject mismatch")
	}
}

// Signature verification should succeed when keyProvider supplies public key.
func TestVerifyTokenSignatureVerified(t *testing.T) {
	svc, signer := newServiceWithSigner(t)
	resp := issueDelegation(t, svc, "grantor", "grantee", time.Minute)
	// Inject signature manually to stored POA
	poa, _ := svc.repo.Get(resp.POA.ID)
	dig, canon, derr := CanonicalPOADigest(poa)
	if derr != nil {
		t.Fatalf("digest: %v", derr)
	}
	sig, serr := signer.Sign(canon)
	if serr != nil {
		t.Fatalf("sign: %v", serr)
	}
	poa.Signature = &POASignature{Algorithm: signer.Algorithm(), KeyID: signer.KeyID(), DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig), Canonical: canon}
	res, err := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
	if err != nil {
		t.Fatalf("verify with signature soft skip err=%v", err)
	}
	if !res.SignatureValid {
		t.Fatalf("expected signature verified")
	}
}
