package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/auth"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001"
)

// Composite demo flow:
// 1. Issue a demo JWT token with scopes.
// 2. Build authorization request using token claims (roles + scopes + env conditions).
// 3. Create a POA delegation under AAP001 service (allowed by role & scopes).
// 4. Revoke the POA and demonstrate subsequent validation failure due to revocation chain.
func main() {
	ctx := context.Background()

	// --- Token issuance (demo) ---
	jwtSvc, _ := auth.NewProperJWTService("issuer-demo", "aud-demo")
	tokenStr, _ := jwtSvc.CreateToken("alice", []string{"docs:read", "delegation:create"}, time.Minute*10)
	claims, _ := jwtSvc.ValidateToken(tokenStr)

	// --- Authorization setup ---
	az := authz.NewMemoryAuthorizer()
	// Policies leveraging roles & required scopes & ABAC conditions
	az.AddPolicy(authz.Policy{ID: "create-poa-role", Roles: []string{"grantor"}, Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow, RequiredScopes: []string{"delegation:create"}, Conditions: []authz.Condition{{Key: "env", Operator: "equals", Values: []string{"prod"}}}})
	// Subject-based policy for AAP001 internal authorization (since service does not propagate roles/scopes yet)
	az.AddPolicy(authz.Policy{ID: "create-poa-subject", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	az.AddPolicy(authz.Policy{ID: "revoke-poa", Subject: "alice", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow, RequiredScopes: []string{"delegation:create"}})
	az.AddPolicy(authz.Policy{ID: "revoke-poa-subject", Subject: "alice", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})

	// Construct request context with roles and scopes
	req := authz.Request{Subject: claims.UserID, Resource: "poa", Action: "create_delegation", Context: map[string]string{
		"roles":  "grantor", // user has grantor role
		"scopes": "docs:read delegation:create",
		"env":    "prod",
	}}
	decision, _ := az.Authorize(ctx, req)
	fmt.Println("Create delegation authorized?", decision.Allow, decision.Reason)
	if !decision.Allow {
		return
	}

	// --- AAP001 service with revocation chain ---
	svc := gauth_rfc_001.NewService(audit.NewMemoryLogger(nil), az)
	delResp, err := svc.CreateDelegationCtx(ctx, gauth_rfc_001.DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Restrictions: map[string]string{"classification": "public"}, Duration: time.Hour})
	if err != nil {
		fmt.Println("Delegation create failed:", err)
		return
	}
	fmt.Println("Delegation created:", delResp.POA.ID)

	// validate before revocation
	if err := svc.ValidateDelegationCtx(ctx, delResp.POA.ID, "bob", "read"); err != nil {
		fmt.Println("Validate failed:", err)
		return
	} else {
		fmt.Println("Delegation validated pre-revoke")
	}

	// revoke
	if err := svc.RevokeDelegationCtx(ctx, delResp.POA.ID, "alice"); err != nil {
		fmt.Println("Revocation failed:", err)
		return
	} else {
		fmt.Println("Delegation revoked")
	}

	// validate after revocation (expected failure)
	if err := svc.ValidateDelegationCtx(ctx, delResp.POA.ID, "bob", "read"); err != nil {
		fmt.Println("Post-revoke validation (expected failure):", err)
	} else {
		fmt.Println("Unexpected success post-revoke")
	}
}
