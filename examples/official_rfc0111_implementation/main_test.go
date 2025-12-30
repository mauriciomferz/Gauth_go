package main

import (
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth_aap_001"
	"github.com/mauriciomferz/Gauth_go/pkg/testutil"
)

func TestDelegationLifecycle(t *testing.T) {
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "test-allow-create-delegation",
		Subject:  "alice@example.com",
		Resource: "poa",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})
	authorizer.AddPolicy(authz.Policy{
		ID:       "test-allow-revoke-delegation",
		Subject:  "alice@example.com",
		Resource: "*",
		Actions:  []string{"revoke_delegation"},
		Effect:   authz.Allow,
	})
	svc := gauth_aap_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)
	req := gauth_aap_001.DelegationRequest{
		Grantor:      "alice@example.com",
		Grantee:      "bob@example.com",
		Scope:        []string{"transaction:execute", "account:read"},
		Restrictions: map[string]string{"max_amount": "1000.00"},
		Duration:     2 * time.Hour,
	}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}
	if resp.POA.Grantor != req.Grantor || resp.POA.Grantee != req.Grantee {
		t.Fatalf("Delegation party mismatch: %+v", resp.POA)
	}
	if err := svc.ValidateDelegation(resp.POA.ID, req.Grantee, "transaction:execute"); err != nil {
		t.Fatalf("ValidateDelegation allowed action failed: %v", err)
	}
	if err := svc.ValidateDelegation(resp.POA.ID, req.Grantee, "forbidden:admin"); err == nil {
		t.Fatalf("Expected validation failure for forbidden action")
	}
	if err := svc.RevokeDelegation(resp.POA.ID, req.Grantor); err != nil {
		t.Fatalf("RevokeDelegation failed: %v", err)
	}
	if err := svc.ValidateDelegation(resp.POA.ID, req.Grantee, "transaction:execute"); err == nil {
		t.Fatalf("Expected rejection after revocation")
	}
}

func TestUnauthorizedRevocation(t *testing.T) {
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "test-allow-create-delegation-unauth-revoke",
		Subject:  "alice@example.com",
		Resource: "poa",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})
	authorizer.AddPolicy(authz.Policy{
		ID:       "test-allow-revoke-delegation-alice",
		Subject:  "alice@example.com",
		Resource: "*",
		Actions:  []string{"revoke_delegation"},
		Effect:   authz.Allow,
	})
	svc := gauth_aap_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)
	req := gauth_aap_001.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: 1 * time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}
	if err := svc.RevokeDelegation(resp.POA.ID, "mallory@example.com"); err == nil {
		t.Fatalf("expected revocation to be unauthorized")
	}
	if err := svc.RevokeDelegation(resp.POA.ID, "alice@example.com"); err != nil {
		t.Fatalf("expected alice to revoke successfully: %v", err)
	}
}

func TestCreateDelegationUnauthorized(t *testing.T) {
	authorizer := authz.NewMemoryAuthorizer()
	svc := gauth_aap_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)
	req := gauth_aap_001.DelegationRequest{Grantor: "carol@example.com", Grantee: "dave@example.com", Scope: []string{"transaction:execute"}, Duration: 30 * time.Minute}
	if _, err := svc.CreateDelegation(req); err == nil {
		t.Fatalf("expected create delegation to be unauthorized without policy")
	}
}

func TestDelegationExpiry(t *testing.T) {
	fc := testutil.NewFakeClock(time.Now())
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	authorizer.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: "alice@example.com", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	svc := gauth_aap_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer).WithClock(fc.Now)
	req := gauth_aap_001.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: 2 * time.Minute}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	fc.Advance(3 * time.Minute)
	if err := svc.ValidateDelegation(resp.POA.ID, req.Grantee, "transaction:execute"); err == nil {
		t.Fatalf("expected expiry validation failure")
	}
}

func TestDelegationWrongGrantee(t *testing.T) {
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	authorizer.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: "alice@example.com", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	svc := gauth_aap_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)
	req := gauth_aap_001.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: 10 * time.Minute}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := svc.ValidateDelegation(resp.POA.ID, "mallory@example.com", "transaction:execute"); err == nil {
		t.Fatalf("expected grantee mismatch error")
	}
}

func TestAuditEventsCount(t *testing.T) {
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	authorizer.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: "alice@example.com", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	svc := gauth_aap_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)
	req := gauth_aap_001.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: 5 * time.Minute}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := svc.ValidateDelegation(resp.POA.ID, req.Grantee, "transaction:execute"); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if err := svc.RevokeDelegation(resp.POA.ID, req.Grantor); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
	evs := svc.AuditEvents()
	if len(evs) < 3 {
		t.Fatalf("expected at least 3 audit events (create, validate, revoke), got %d", len(evs))
	}
	want := map[string]bool{"create_delegation": false, "validate_delegation": false, "revoke_delegation": false}
	for _, ev := range evs {
		want[ev.Action] = true
	}
	for act, seen := range want {
		if !seen {
			t.Fatalf("expected audit event action %s to be present; events=%v", act, evs)
		}
	}
}

func TestDelegationScenarioAllowedForbidden(t *testing.T) {
	err := ExecDelegationScenarioAllowedForbidden()
	if err == nil {
		t.Fatalf("expected sentinel error for allowed forbidden action")
	}
}

func TestExpiryScenarioNoAdvance(t *testing.T) {
	ok, err := ExecExpiryScenarioNoAdvance()
	if err != nil || !ok {
		t.Fatalf("expected success before expiry, err=%v ok=%v", err, ok)
	}
}

func TestCoverageHelpersScenario(t *testing.T) {
	res, err := ExecDelegationScenario()
	if err != nil {
		t.Fatalf("ExecDelegationScenario error: %v", err)
	}
	if !res.Validated || !res.InvalidAction || !res.Revoked || !res.PostRevokeFail {
		t.Fatalf("unexpected scenario result: %+v", res)
	}
	if res.AuditEvents < 3 {
		t.Fatalf("expected >=3 audit events, got %d", res.AuditEvents)
	}
	ok, err := ExecExpiryScenario()
	if err != nil {
		t.Fatalf("ExecExpiryScenario error: %v", err)
	}
	if !ok {
		t.Fatalf("expiry scenario did not enforce expiry")
	}
}
