package main

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth_aap_001"
)

func TestRunDemoWithLargeJSON(t *testing.T) {
	res, err := RunDemoWithLargeJSON()
	if err != nil {
		t.Fatalf("RunDemoWithLargeJSON error: %v", err)
	}
	if !res.JSONTruncated {
		t.Fatalf("expected JSONTruncated true")
	}
}

func TestRunDemoExpectFailure(t *testing.T) {
	err := RunDemoExpectFailure()
	if err == nil {
		t.Fatalf("expected failure error, got nil")
	}
}

// Additional negative scenarios to cover error return branches indirectly.
func TestRunDemoFailureScenarios(t *testing.T) {
	// Scenario 1: Invalid config (delegation depth 0)
	cfg := createAAP001Config()
	cfg.MaxDelegationDepth = 0
	if err := gauth_aap_001.ValidateAAP001Compliance(cfg); err == nil {
		t.Fatalf("expected config validation failure for depth=0")
	}
	// Scenario 2: Missing revoke permission (simulate by crafting custom function locally)
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create-only", Subject: "principal@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := gauth_aap_001.NewService(auditLogger, authorizer)
	req := gauth_aap_001.DelegationRequest{Grantor: "principal@example.com", Grantee: "agent@example.com", Scope: []string{"transaction:execute"}, Duration: time.Hour}
	delegation, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("unexpected create failure: %v", err)
	}
	// Validate allowed action
	if err := svc.ValidateDelegation(delegation.POA.ID, req.Grantee, "transaction:execute"); err != nil {
		t.Fatalf("unexpected validate failure: %v", err)
	}
	// Attempt revoke should fail due to missing permission.
	if err := svc.RevokeDelegation(delegation.POA.ID, req.Grantor); err == nil {
		t.Fatalf("expected revoke failure without permission")
	}
}
