package main

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
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
	cfg := createRFC0111Config()
	cfg.MaxDelegationDepth = 0
	if err := rfc0111.ValidateRFC0111Compliance(cfg); err == nil {
		t.Fatalf("expected config validation failure for depth=0")
	}
	// Scenario 2: Missing revoke permission (simulate by crafting custom function locally)
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create-only", Subject: "principal@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := rfc0111.NewService(auditLogger, authorizer)
	req := rfc0111.DelegationRequest{Grantor: "principal@example.com", Grantee: "agent@example.com", Scope: []string{"transaction:execute"}, Duration: time.Hour}
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
