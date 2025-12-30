package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	cr "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// TestMandatoryDetachedSignature_Enforced verifies that when GAUTH_REQUIRE_DETACHED_SIGNATURE
// is set, tokens without detached signatures are rejected.
func TestMandatoryDetachedSignature_Enforced(t *testing.T) {
	t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	t.Setenv("GAUTH_DETACHED_SIGNATURE", "") // intentionally disabled
	t.Setenv("GAUTH_REQUIRE_DETACHED_SIGNATURE", "1")

	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("key provider: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "grantor", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))

	// Create delegation WITHOUT detached signature (detached signature flag disabled)
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}

	// Verify token should FAIL because detached signature is required but missing
	_, verr := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
	if verr == nil {
		t.Fatalf("expected verification failure when detached signature is required but missing")
	}
}

// TestMandatoryDetachedSignature_Present verifies that when GAUTH_REQUIRE_DETACHED_SIGNATURE
// is set, tokens WITH detached signatures are accepted.
func TestMandatoryDetachedSignature_Present(t *testing.T) {
	t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	t.Setenv("GAUTH_DETACHED_SIGNATURE", "1") // enabled
	t.Setenv("GAUTH_REQUIRE_DETACHED_SIGNATURE", "1")

	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("key provider: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "grantor", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))

	// Create delegation WITH detached signature
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}

	// Verify token should SUCCEED because detached signature is present and valid
	vres, verr := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
	if verr != nil {
		t.Fatalf("verify token: %v", verr)
	}
	if !vres.DetachedSignatureValid {
		t.Fatalf("expected detached signature to verify: %#v", vres)
	}
}

// TestMandatoryDetachedSignature_NotRequired verifies backward compatibility when
// GAUTH_REQUIRE_DETACHED_SIGNATURE is NOT set (default behavior).
func TestMandatoryDetachedSignature_NotRequired(t *testing.T) {
	t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	t.Setenv("GAUTH_DETACHED_SIGNATURE", "") // disabled
	t.Setenv("GAUTH_REQUIRE_DETACHED_SIGNATURE", "")

	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("key provider: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "g", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))

	// Create delegation WITHOUT detached signature
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "g", Grantee: "u", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}

	// Verify token should SUCCEED (backward compatible mode - signatures optional)
	res, verr := svc.VerifyToken(WithSubject(context.Background(), "u"), resp.AuthToken)
	if verr != nil {
		t.Fatalf("verify: %v", verr)
	}
	// DetachedSignatureValid should be false (no signature present)
	if res.DetachedSignatureValid {
		t.Fatalf("expected no detached signature validation when feature disabled")
	}
}

// TestMandatoryDetachedSignature_V1EnvelopeRejected verifies that when
// GAUTH_REQUIRE_DETACHED_SIGNATURE is set, V1 envelopes (which cannot have detached signatures)
// are rejected.
func TestMandatoryDetachedSignature_V1EnvelopeRejected(t *testing.T) {
	t.Setenv("GAUTH_POA_ENVELOPE_V2", "") // V1 envelope
	t.Setenv("GAUTH_REQUIRE_DETACHED_SIGNATURE", "1")

	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("key provider: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "g", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))

	// Create delegation with V1 envelope
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "g", Grantee: "u", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}

	// Verify token should FAIL (V1 envelope cannot have detached signature)
	_, verr := svc.VerifyToken(WithSubject(context.Background(), "u"), resp.AuthToken)
	if verr == nil {
		t.Fatalf("expected verification failure for V1 envelope when detached signature is required")
	}
}

// TestMandatoryDetachedSignature_FailClosedMetrics verifies that mandatory enforcement
// failures are properly instrumented in metrics.
func TestMandatoryDetachedSignature_FailClosedMetrics(t *testing.T) {
	t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	t.Setenv("GAUTH_DETACHED_SIGNATURE", "")
	t.Setenv("GAUTH_REQUIRE_DETACHED_SIGNATURE", "1")

	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("key provider: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "g", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})

	// Use metrics-enabled service
	memMetrics := metrics.NewMemory()
	svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp), WithMetrics(memMetrics))

	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "g", Grantee: "u", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}

	// Baseline metrics
	baselineSnapshot := memMetrics.SnapshotEx()
	baselineFailures := baselineSnapshot.SignatureVerificationFailures

	// Attempt verification (should fail due to missing required signature)
	_, _ = svc.VerifyToken(WithSubject(context.Background(), "u"), resp.AuthToken)

	// Check metrics incremented
	newSnapshot := memMetrics.SnapshotEx()
	newFailures := newSnapshot.SignatureVerificationFailures
	if newFailures <= baselineFailures {
		t.Fatalf("expected signature verification failure metric to increment: baseline=%d, new=%d", baselineFailures, newFailures)
	}
}
