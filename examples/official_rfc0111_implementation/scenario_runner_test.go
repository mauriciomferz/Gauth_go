package main

import (
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
)

func TestScenarioRunnerSuccessMinimal(t *testing.T) {
	res, err := runDemoInternal(scenarioParams{grantor: "p@example.com", grantee: "g@example.com", allowCreatePolicy: true, allowRevokePolicy: true, performRevoke: true, postRevokeValidation: true})
	if err != nil || res.DelegationID == "" || !res.PostRevokeFail || !res.InvalidAction { // invalid action should be rejected
		t.Fatalf("unexpected result: %+v err=%v", res, err)
	}
}

func TestScenarioRunnerInvalidActionAllowed(t *testing.T) {
	_, err := runDemoInternal(scenarioParams{grantor: "p2@example.com", grantee: "g2@example.com", allowCreatePolicy: true, allowRevokePolicy: true, allowInvalidAction: true, performRevoke: true, postRevokeValidation: true})
	if err == nil {
		t.Fatalf("expected error when invalid action is allowed")
	}
}

func TestScenarioRunnerMissingCreatePolicy(t *testing.T) {
	_, err := runDemoInternal(scenarioParams{grantor: "x@example.com", grantee: "y@example.com", allowRevokePolicy: true})
	if err == nil {
		t.Fatalf("expected create failure without create policy")
	}
}

func TestScenarioRunnerMissingRevokePolicy(t *testing.T) {
	// perform revoke should succeed only if revoke policy exists; we expect error.
	_, err := runDemoInternal(scenarioParams{grantor: "r@example.com", grantee: "g@example.com", allowCreatePolicy: true, performRevoke: true, postRevokeValidation: true})
	if err == nil {
		t.Fatalf("expected revoke failure without revoke policy")
	}
}

func TestScenarioRunnerConfigMutationFailure(t *testing.T) {
	_, err := runDemoInternal(scenarioParams{grantor: "cm@example.com", grantee: "cg@example.com", allowCreatePolicy: true, modifyConfig: func(cfg *rfc0111.RFC0111Config) { cfg.MaxDelegationDepth = 0 }})
	if err == nil {
		t.Fatalf("expected config validation failure")
	}
}

func TestScenarioRunnerLargeJSON(t *testing.T) {
	res, err := runDemoInternal(scenarioParams{grantor: "lj@example.com", grantee: "gl@example.com", allowCreatePolicy: true, allowRevokePolicy: true, performRevoke: true, postRevokeValidation: true, largeJSON: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.JSONTruncated {
		t.Fatalf("expected JSON truncation")
	}
}
