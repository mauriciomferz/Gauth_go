package main

import (
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth_aap_001"
	"github.com/mauriciomferz/AgentAuth/pkg/testutil"
)

// Covers branch where revoke is skipped entirely.
func TestScenarioNoRevoke(t *testing.T) {
	res, err := runDemoInternal(scenarioParams{grantor: "nr@example.com", grantee: "g@example.com", allowCreatePolicy: true, allowRevokePolicy: true, performRevoke: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostRevokeFail {
		t.Fatalf("post revoke flag should be false when revoke skipped")
	}
}

// Covers branch with revoke but no post-revoke validation attempt.
func TestScenarioRevokeNoValidation(t *testing.T) {
	res, err := runDemoInternal(scenarioParams{grantor: "rv@example.com", grantee: "g@example.com", allowCreatePolicy: true, allowRevokePolicy: true, performRevoke: true, postRevokeValidation: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostRevokeFail {
		t.Fatalf("post revoke fail should be false when not validated")
	}
}

func TestRunDemoExpectFailureMessage(t *testing.T) {
	err := RunDemoExpectFailure()
	if err == nil {
		t.Fatalf("expected failure error, got nil")
	}
	// Accept either phrase used in failure paths.
	if !strings.Contains(err.Error(), "expected invalid action") && !strings.Contains(err.Error(), "expected admin:delete") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestExecDelegationScenarioAllowedForbiddenMessage(t *testing.T) {
	err := ExecDelegationScenarioAllowedForbidden()
	if err == nil {
		t.Fatalf("expected sentinel error, got nil")
	}
	if !strings.Contains(err.Error(), "forbidden action unexpectedly allowed") && !strings.Contains(err.Error(), "expected forbidden action") {
		t.Fatalf("unexpected sentinel error message: %v", err)
	}
}

// Zero duration expiry edge case.
func TestExpiryZeroDuration(t *testing.T) {
	fc := testutil.NewFakeClock(time.Now())
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create-zero", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := gauth_aap_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer).WithClock(fc.Now)
	req := gauth_aap_001.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: 0}
	if _, err := svc.CreateDelegation(req); err == nil {
		t.Fatalf("expected create failure for zero duration")
	}
}

// Negative duration edge case.
func TestExpiryNegativeDuration(t *testing.T) {
	fc := testutil.NewFakeClock(time.Now())
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create-neg", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := gauth_aap_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer).WithClock(fc.Now)
	req := gauth_aap_001.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: -1 * time.Minute}
	if _, err := svc.CreateDelegation(req); err == nil {
		t.Fatalf("expected create failure for negative duration")
	}
}
