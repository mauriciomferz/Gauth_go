package gauth_aap_001

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	cr "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// fixedClock returns a constant timestamp for deterministic canonical fields.
func fixedClock() time.Time { return time.Unix(1730000000, 0).UTC() }

func TestAlgorithmAgility_Ed25519AndECDSA(t *testing.T) {
	authorizer := authz.NewMemoryAuthorizer()
	// Grant create delegation permission for grantor g1
	authorizer.AddPolicy(authz.Policy{ID: "p1", Subject: "g1", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	// Service A: Ed25519
	svcEd := NewService(audit.NewMemoryLogger(nil), authorizer, WithInMemoryAlgorithm(cr.AlgoEd25519))
	svcEd.WithClock(fixedClock)
	// Service B: ECDSA P-256
	svcEc := NewService(audit.NewMemoryLogger(nil), authorizer, WithInMemoryAlgorithm(cr.AlgoECDSAP256))
	svcEc.WithClock(fixedClock)

	req := DelegationRequest{Grantor: "g1", Grantee: "u1", Scope: []string{"read"}, Duration: time.Hour}
	respEd, err := svcEd.CreateDelegationCtx(context.Background(), req)
	if err != nil {
		t.Fatalf("ed25519 issuance failed: %v", err)
	}
	respEc, err := svcEc.CreateDelegationCtx(context.Background(), req)
	if err != nil {
		t.Fatalf("ecdsa issuance failed: %v", err)
	}

	if respEd.POA.Signature == nil || respEc.POA.Signature == nil {
		t.Fatalf("missing signatures")
	}
	if respEd.POA.Signature.Algorithm != cr.AlgoEd25519 {
		t.Fatalf("expected ed25519 algorithm recorded")
	}
	if respEc.POA.Signature.Algorithm != cr.AlgoECDSAP256 {
		t.Fatalf("expected ecdsa-p256 algorithm recorded")
	}

	// Canonical digest invariance per-algorithm (each must match its recorded digest).
	digEd, _, derrEd := CanonicalPOADigest(&respEd.POA)
	if derrEd != nil || digEd != respEd.POA.Signature.DigestHex {
		t.Fatalf("ed25519 canonical digest mismatch")
	}
	digEc, _, derrEc := CanonicalPOADigest(&respEc.POA)
	if derrEc != nil || digEc != respEc.POA.Signature.DigestHex {
		t.Fatalf("ecdsa canonical digest mismatch")
	}

	// Verify signatures via registry dispatch using the service verify helper (internal path).
	if err := svcEd.verifyPOASignature(&respEd.POA); err != nil {
		t.Fatalf("ed25519 verification path failed: %v", err)
	}
	if err := svcEc.verifyPOASignature(&respEc.POA); err != nil {
		t.Fatalf("ecdsa verification path failed: %v", err)
	}

	// Negative tamper test: modify canonical byte (scope ordering) and expect failure.
	// For simplicity adjust scope for ECDSA POA.
	tampered := respEc.POA
	tampered.Scope = []string{"write"}
	digTampered, _, _ := CanonicalPOADigest(&tampered)
	if digTampered == respEc.POA.Signature.DigestHex {
		t.Fatalf("tampered digest unexpectedly matches original")
	}
	// Manually dispatch verification expecting failure.
	sigBytes, _ := base64.StdEncoding.DecodeString(respEc.POA.Signature.SigBase64)
	_ = sigBytes // only for potential future raw checks
	if err := svcEc.verifyPOASignature(&tampered); err == nil {
		t.Fatalf("expected verification failure for tampered POA")
	}
}
