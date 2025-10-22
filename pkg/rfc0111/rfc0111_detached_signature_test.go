package rfc0111

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	cr "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/crypto"
)

// TestDetachedSignatureIssuanceAndVerification ensures that when feature flags are enabled the
// envelope V2 carries detached signature fields which verify successfully.
func TestDetachedSignatureIssuanceAndVerification(t *testing.T) {
    t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
    t.Setenv("GAUTH_DETACHED_SIGNATURE", "1")

    kp, err := cr.NewInMemoryEd25519Provider()
    if err != nil {
        t.Fatalf("key provider: %v", err)
    }
    auditLogger := audit.NewMemoryLogger(nil)
    authorizer := authz.NewMemoryAuthorizer()
    authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "grantor", Resource: "*", Actions: []string{"create_delegation", "revoke_delegation"}, Effect: authz.Allow})
    svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))

    resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{"read"}, Duration: time.Minute})
    if err != nil {
        t.Fatalf("create delegation: %v", err)
    }
    // Verify token; detached signature should be validated.
    vres, verr := svc.VerifyToken(WithSubject(context.Background(), "grantee"), resp.AuthToken)
    if verr != nil {
        t.Fatalf("verify token: %v", verr)
    }
    if !vres.DetachedSignatureValid {
        t.Fatalf("expected detached signature to verify: %#v", vres)
    }
}

// TestDetachedSignatureTamper modifies stored POA after issuance to trigger integrity failure
// (digest mismatch) – ensures tamper is detected (error returned).
func TestDetachedSignatureTamper(t *testing.T) {
    t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
    t.Setenv("GAUTH_DETACHED_SIGNATURE", "1")
    kp, err := cr.NewInMemoryEd25519Provider()
    if err != nil { t.Fatalf("key provider: %v", err) }
    auditLogger := audit.NewMemoryLogger(nil)
    authorizer := authz.NewMemoryAuthorizer()
    authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "g", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
    svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))
    resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "g", Grantee: "u", Scope: []string{"read"}, Duration: time.Minute})
    if err != nil { t.Fatalf("create delegation: %v", err) }
    // Tamper stored POA (alter scope -> digest mismatch)
    stored, _ := svc.repo.Get(resp.POA.ID)
    stored.Scope = []string{"tampered"}
    // Expect integrity failure
    if _, vErr := svc.VerifyToken(WithSubject(context.Background(), "u"), resp.AuthToken); vErr == nil {
        t.Fatalf("expected integrity failure on tamper")
    }
}

// TestDetachedSignatureDisabled ensures absence when flag disabled (control path).
func TestDetachedSignatureDisabled(t *testing.T) {
    t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
    t.Setenv("GAUTH_DETACHED_SIGNATURE", "") // ensure disabled even if parent test process exported
    // GAUTH_DETACHED_SIGNATURE intentionally NOT set
    kp, err := cr.NewInMemoryEd25519Provider()
    if err != nil { t.Fatalf("key provider: %v", err) }
    auditLogger := audit.NewMemoryLogger(nil)
    authorizer := authz.NewMemoryAuthorizer()
    authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "g", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
    svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))
    resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "g", Grantee: "u", Scope: []string{"read"}, Duration: time.Minute})
    if err != nil { t.Fatalf("create delegation: %v", err) }
    res, vErr := svc.VerifyToken(WithSubject(context.Background(), "u"), resp.AuthToken)
    if vErr != nil { t.Fatalf("verify: %v", vErr) }
    if res.DetachedSignatureValid { t.Fatalf("expected no detached signature validation when feature disabled") }
}
