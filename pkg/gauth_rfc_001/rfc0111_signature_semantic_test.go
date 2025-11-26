package gauth_rfc_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	cr "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// signerProviderStub returns a valid signer via in-memory ed25519 provider.
func signerProviderStub() (cr.Signer, error) {
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		return nil, err
	}
	return kp.ActiveSigner()
}

// failingSignerProvider simulates signer acquisition failure.
func failingSignerProvider() (cr.Signer, error) { return nil, context.Canceled }

func setupAllowAuthorizer() *authz.MemoryAuthorizer {
	mem := authz.NewMemoryAuthorizer()
	// Add allow policy for create_delegation on resource "poa".
	pol := authz.Policy{ID: "p_allow_create", Subject: "*", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow}
	mem.AddPolicy(pol)
	return mem
}

func TestMandatorySignatureSuccess(t *testing.T) {
	auditLogger := audit.NewMemoryLogger(nil)
	authzMem := setupAllowAuthorizer()
	svc := NewService(auditLogger, authzMem, WithSignerProvider(signerProviderStub), WithMandatorySignatures())
	req := DelegationRequest{Grantor: "a@example.com", Grantee: "b@example.com", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD"}, Duration: time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.POA.Signature == nil {
		t.Fatalf("expected signature present")
	}
}

func TestMandatorySignatureFailureAbort(t *testing.T) {
	auditLogger := audit.NewMemoryLogger(nil)
	authzMem := setupAllowAuthorizer()
	svc := NewService(auditLogger, authzMem, WithSignerProvider(failingSignerProvider), WithMandatorySignatures())
	req := DelegationRequest{Grantor: "x@example.com", Grantee: "y@example.com", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD"}, Duration: time.Hour}
	_, err := svc.CreateDelegation(req)
	if err == nil {
		t.Fatalf("expected error when signer unavailable")
	}
}

func TestAdvancedSemanticValidatorRules(t *testing.T) {
	auditLogger := audit.NewMemoryLogger(nil)
	authzMem := setupAllowAuthorizer()
	// Use AdvancedPoAValidator: currency requirement & 30d cap enforced here now.
	svc := NewService(auditLogger, authzMem, WithSemanticValidator(AdvancedPoAValidator{}))
	// Self delegation non-wildcard should fail
	bad := DelegationRequest{Grantor: "same@example.com", Grantee: "same@example.com", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD"}, Duration: time.Hour}
	if _, err := svc.CreateDelegation(bad); err == nil {
		t.Fatalf("expected self-delegation rejection")
	}
	// Missing currency for transaction scope
	missingCur := DelegationRequest{Grantor: "a@example.com", Grantee: "b@example.com", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{}, Duration: time.Hour}
	if _, err := svc.CreateDelegation(missingCur); err == nil {
		t.Fatalf("expected currency restriction requirement failure")
	}
	// Over 30d cap
	overCap := DelegationRequest{Grantor: "a@example.com", Grantee: "b@example.com", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD"}, Duration: 31 * 24 * time.Hour}
	if _, err := svc.CreateDelegation(overCap); err == nil {
		t.Fatalf("expected duration cap failure")
	}
	// Wildcard self delegation should be rejected under Advanced (unless GAUTH_ALLOW_WILDCARD=1)
	deniedSelf := DelegationRequest{Grantor: "same@example.com", Grantee: "same@example.com", Scope: []string{"*"}, Restrictions: map[string]string{}, Duration: time.Hour}
	if _, err := svc.CreateDelegation(deniedSelf); err == nil {
		t.Fatalf("expected wildcard rejection in advanced mode")
	}
}
