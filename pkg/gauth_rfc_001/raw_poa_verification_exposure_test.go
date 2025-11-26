package gauth_rfc_001

import (
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// helper to build service with envelope v2 + embedding flags
// (helper removed; direct env usage per test keeps isolation)

// TestRawPOAExposedWhenEmbedded ensures verification result surfaces RawPOA & PoAVersion.
func TestRawPOAExposedWhenEmbedded(t *testing.T) {
	t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	t.Setenv("GAUTH_EMBED_FULL_POA", "1")
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer())
	// authorize basic create_delegation for grantor
	authzMem := svc.authz.(*authz.MemoryAuthorizer)
	authzMem.AddPolicy(authz.Policy{Subject: "g1", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	req := DelegationRequest{Grantor: "g1", Grantee: "g2", Scope: []string{"a"}, Restrictions: map[string]string{"x": "y"}, Duration: time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("CreateDelegation error: %v", err)
	}
	ctx := WithSubject(testCtx(), "g2")
	vr, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken error: %v", err)
	}
	if vr.RawPOA == "" {
		t.Fatalf("expected RawPOA populated when embedding enabled")
	}
	if vr.PoAVersion != "poa/v1" {
		t.Fatalf("expected poa/v1, got %s", vr.PoAVersion)
	}
	if !strings.Contains(vr.RawPOA, "\"grantor\":\"g1\"") {
		t.Fatalf("RawPOA does not contain grantor field")
	}
}

// TestRawPOAAbsentWhenDisabled ensures RawPOA empty when embedding flag off.
func TestRawPOAAbsentWhenDisabled(t *testing.T) {
	t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	t.Setenv("GAUTH_EMBED_FULL_POA", "0")
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer())
	authzMem := svc.authz.(*authz.MemoryAuthorizer)
	authzMem.AddPolicy(authz.Policy{Subject: "g1", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	req := DelegationRequest{Grantor: "g1", Grantee: "g2", Scope: []string{"a"}, Restrictions: map[string]string{"x": "y"}, Duration: time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("CreateDelegation error: %v", err)
	}
	ctx := WithSubject(testCtx(), "g2")
	vr, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken error: %v", err)
	}
	if vr.RawPOA != "" {
		t.Fatalf("expected RawPOA empty when embedding disabled")
	}
	if vr.PoAVersion != "" {
		t.Fatalf("expected PoAVersion empty when embedding disabled")
	}
}

// TestRawPOAOmittedWhenSizeExceeded ensures omission when canonical JSON exceeds cap.
func TestRawPOAOmittedWhenSizeExceeded(t *testing.T) {
	t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	t.Setenv("GAUTH_EMBED_FULL_POA", "1")
	// Set a tiny max size to force omission (canonical JSON will exceed this)
	t.Setenv("GAUTH_MAX_RAW_POA_BYTES", "10")
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer())
	authzMem := svc.authz.(*authz.MemoryAuthorizer)
	authzMem.AddPolicy(authz.Policy{Subject: "g1", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	req := DelegationRequest{Grantor: "g1", Grantee: "g2", Scope: []string{"scope_action"}, Restrictions: map[string]string{"currency": "USD"}, Duration: time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("CreateDelegation error: %v", err)
	}
	ctx := WithSubject(testCtx(), "g2")
	vr, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken error: %v", err)
	}
	if vr.RawPOA != "" {
		t.Fatalf("expected RawPOA omitted due to size cap")
	}
	if vr.PoAVersion != "" {
		t.Fatalf("expected PoAVersion omitted when RawPOA omitted")
	}
}
