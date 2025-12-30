// Official AAP001 Implementation Demo (Beta Demonstration)
//
// Copyright (c) 2025 AgentAuth Community
// Licensed under Apache 2.0
//
// ⚠️  BETA DEMONSTRATION NOTICE
// This example is part of a beta demonstration implementation and is **NOT production ready**.
// It omits enforceable signature ceremonies, tamper‑evident audit storage, jurisdictional
// nuance handling, and verified identity assurance workflows required for regulated use.
// See DISCLAIMER.md and docs/DEPRECATION_TIMELINE.md for lifecycle & removal roadmap.
//
// Demonstrates the complete AgentAuth 1.0 Authorization Framework
// as specified in AAP001 by The AgentAuth Community

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001"
)

// DemoResult captures key outcomes for testing & introspection while keeping
// main() focused on user-oriented printing.
type DemoResult struct {
	DelegationID   string
	ValidUntil     time.Time
	InvalidAction  bool
	PostRevokeFail bool
	JSONTruncated  bool
}

// scenarioParams allows tests to exercise all branches of the original demo logic.
// It is intentionally internal (unexported) to keep public API surface minimal.
type scenarioParams struct {
	grantor              string
	grantee              string
	allowCreatePolicy    bool
	allowRevokePolicy    bool
	allowInvalidAction   bool                         // if true, admin:delete will be allowed (triggers failure path)
	performRevoke        bool                         // if false, skip revoke
	postRevokeValidation bool                         // if true, attempt post-revoke validation
	largeJSON            bool                         // if true, force JSON truncation path
	modifyConfig         func(*agentauth_aap_001.AAP001Config) // optional mutator to trigger config errors
}

// runDemoInternal executes the scenario described by params and returns DemoResult or error explaining failure.
func runDemoInternal(p scenarioParams) (*DemoResult, error) {
	cfg := createAAP001Config()
	if p.modifyConfig != nil {
		p.modifyConfig(cfg)
	}
	if err := agentauth_aap_001.ValidateAAP001Compliance(cfg); err != nil {
		return nil, fmt.Errorf("config invalid: %w", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	if p.allowCreatePolicy {
		authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: p.grantor, Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	}
	if p.allowRevokePolicy {
		authorizer.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: p.grantor, Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	}
	if p.allowInvalidAction {
		authorizer.AddPolicy(authz.Policy{ID: "allow-admin-delete", Subject: p.grantor, Resource: "poa", Actions: []string{"admin:delete"}, Effect: authz.Allow})
	}
	svc := agentauth_aap_001.NewService(auditLogger, authorizer)
	req := agentauth_aap_001.DelegationRequest{Grantor: p.grantor, Grantee: p.grantee, Scope: []string{"transaction:execute", "account:read"}, Duration: 2 * time.Hour}
	delegation, err := svc.CreateDelegation(req)
	if err != nil {
		return nil, fmt.Errorf("create delegation failed: %w", err)
	}
	if err := svc.ValidateDelegation(delegation.POA.ID, p.grantee, "transaction:execute"); err != nil {
		return nil, fmt.Errorf("validate allowed failed: %w", err)
	}
	invalid := false
	if err := svc.ValidateDelegation(delegation.POA.ID, p.grantee, "admin:delete"); err != nil {
		invalid = true
		if p.allowInvalidAction { // expected success path but got failure
			return nil, fmt.Errorf("expected admin:delete to be allowed")
		}
	} else if !p.allowInvalidAction { // action unexpectedly allowed
		return nil, fmt.Errorf("expected invalid action rejection")
	}
	framework := agentauth_aap_001.AgentAuth10Framework{AuthServer: cfg.AuthorizationServerURL, Clients: []string{p.grantee}}
	meta, _ := framework.ToJSON()
	out := map[string]interface{}{"delegation": delegation.POA, "auth_token": delegation.AuthToken, "framework": json.RawMessage(meta)}
	if p.largeJSON {
		big := map[string]int{}
		for i := 0; i < 1200; i++ {
			big[fmt.Sprintf("k%d", i)] = i
		}
		out["large"] = big
	}
	blob, _ := json.MarshalIndent(out, "", "  ")
	truncated := len(blob) > 900
	postRevokeFail := false
	if p.performRevoke {
		if err := svc.RevokeDelegation(delegation.POA.ID, p.grantor); err != nil {
			return nil, fmt.Errorf("revoke failed: %w", err)
		}
		if p.postRevokeValidation {
			if err := svc.ValidateDelegation(delegation.POA.ID, p.grantee, "transaction:execute"); err != nil {
				postRevokeFail = true
			} else {
				return nil, fmt.Errorf("expected rejection after revocation")
			}
		}
	}
	return &DemoResult{DelegationID: delegation.POA.ID, ValidUntil: delegation.POA.ValidUntil, InvalidAction: invalid, PostRevokeFail: postRevokeFail, JSONTruncated: truncated}, nil
}

// RunDemo executes the original demo sequence and returns a structured result.
func RunDemo() (*DemoResult, error) {
	return runDemoInternal(scenarioParams{
		grantor:              "principal@example.com",
		grantee:              "agent@example.com",
		allowCreatePolicy:    true,
		allowRevokePolicy:    true,
		allowInvalidAction:   false,
		performRevoke:        true,
		postRevokeValidation: true,
		largeJSON:            false,
	})
}

// RunDemoWithLargeJSON creates an oversized metadata snapshot to force the
// JSON truncation branch. It reuses the same logic but inflates the output.
func RunDemoWithLargeJSON() (*DemoResult, error) {
	return runDemoInternal(scenarioParams{
		grantor:              "principal@example.com",
		grantee:              "agent@example.com",
		allowCreatePolicy:    true,
		allowRevokePolicy:    true,
		allowInvalidAction:   false,
		performRevoke:        true,
		postRevokeValidation: true,
		largeJSON:            true,
	})
}

// RunDemoExpectFailure manipulates policies to trigger an invalid action NOT being rejected (expected error path).
func RunDemoExpectFailure() error {
	_, err := runDemoInternal(scenarioParams{
		grantor:              "principal@example.com",
		grantee:              "agent@example.com",
		allowCreatePolicy:    true,
		allowRevokePolicy:    true,
		allowInvalidAction:   true, // triggers failure via unexpected allowance
		performRevoke:        true,
		postRevokeValidation: true,
	})
	if err == nil {
		return fmt.Errorf("expected invalid action rejection but was allowed")
	}
	return err
}

func main() {
	fmt.Println("=== AAP001 AgentAuth 1.0 Authorization Framework Demo (Beta) ===")
	fmt.Println("This walkthrough uses the current aap001 service API (delegations) instead of deprecated deep struct graph.")
	fmt.Println()
	res, err := RunDemo()
	if err != nil {
		log.Fatalf("demo run failed: %v", err)
	}
	fmt.Printf("Delegation %s valid until %s\n", res.DelegationID, res.ValidUntil.Format(time.RFC3339))
	if res.InvalidAction {
		fmt.Println("Invalid action rejection verified ✅")
	}
	if res.PostRevokeFail {
		fmt.Println("Post-revoke validation failure verified ✅")
	}
	if res.JSONTruncated {
		fmt.Println("Snapshot JSON truncated due to size ✅")
	} else {
		fmt.Println("Snapshot JSON not truncated ✅")
	}
	fmt.Println("\nAll AAP001 demo steps completed (beta demonstration – non-production).")
}

func createAAP001Config() *agentauth_aap_001.AAP001Config {
	return &agentauth_aap_001.AAP001Config{
		AuthorizationServerURL:    "https://auth.example.com",
		TrustServiceProvider:      "AgentAuth Community Trust Services",
		RequireNotarization:       true,
		MaxDelegationDepth:        3,
		DefaultTokenValidity:      24 * time.Hour,
		AuditingEnabled:           true,
		ComplianceTrackingEnabled: true,

		// AAP001 Section 2: Mandatory exclusions for open source
		ExcludeWeb3:          true, // No blockchain/web3 tokens
		ExcludeAIOperators:   true, // No AI controlling the entire process
		ExcludeDNAIdentities: true, // No DNA-based identities
	}
}
