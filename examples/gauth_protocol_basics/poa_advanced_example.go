// poa_advanced_example.go
// Title: Advanced POA (RFC 111) Scenarios
// Description: Demonstrates negative and positive cases for RFC 111 Power of Attorney validation, including invalid jurisdiction, disallowed scope, missing fields, and a valid advanced POA.

package main

import (
	"context"
	"fmt"

	auth "github.com/mauriciomferz/AgentAuth/pkg/auth"
)

// Advanced RFC 111 Power of Attorney (POA) scenario demonstrating negative cases and compliance checks.

//nolint:unused // Example function for documentation purposes
func runAdvancedPOAExample() {
	ctx := context.Background()
	svc := auth.NewRFCCompliantService()

	// 1. Invalid jurisdiction (should fail)
	invalidJurisdictionReq := auth.PowerOfAttorneyRequest{
		ClientID:     "demo-client",
		ResponseType: "code",
		Scope:        "ai_power_of_attorney,financial_transactions",
		RedirectURI:  "https://app.example.com/callback",
		PowerType:    "financial_transactions",
		PrincipalID:  "user-123",
		AIAgentID:    "ai-agent-1",
		Jurisdiction: "ZZ", // Invalid
		LegalBasis:   "power_of_attorney_act_2025",
	}
	_, err := svc.AuthorizePowerOfAttorney(ctx, invalidJurisdictionReq)
	if err != nil {
		fmt.Println("[NEGATIVE] Invalid jurisdiction correctly rejected:", err)
	}

	// 2. Disallowed scope (should fail)
	disallowedScopeReq := invalidJurisdictionReq
	disallowedScopeReq.Jurisdiction = "US"
	disallowedScopeReq.Scope = "nuclear_launch_codes"
	_, err = svc.AuthorizePowerOfAttorney(ctx, disallowedScopeReq)
	if err != nil {
		fmt.Println("[NEGATIVE] Disallowed scope correctly rejected:", err)
	}

	// 3. Missing required fields (should fail)
	missingFieldsReq := auth.PowerOfAttorneyRequest{
		ClientID:     "",
		ResponseType: "code",
		Scope:        "ai_power_of_attorney",
		RedirectURI:  "https://app.example.com/callback",
		PowerType:    "",
		PrincipalID:  "",
		AIAgentID:    "ai-agent-1",
		Jurisdiction: "US",
		LegalBasis:   "power_of_attorney_act_2025",
	}
	_, err = svc.AuthorizePowerOfAttorney(ctx, missingFieldsReq)
	if err != nil {
		fmt.Println("[NEGATIVE] Missing fields correctly rejected:", err)
	}

	// 4. Valid advanced POA (should succeed)
	advancedReq := auth.PowerOfAttorneyRequest{
		ClientID:     "enterprise-client-42",
		ResponseType: "code",
		Scope:        "ai_power_of_attorney,financial_transactions,compliance_audit",
		RedirectURI:  "https://enterprise.example.com/callback",
		PowerType:    "compliance_audit",
		PrincipalID:  "org-789",
		AIAgentID:    "ai-auditor-2",
		Jurisdiction: "EU",
		LegalBasis:   "eu_poa_regulation_2025",
	}
	resp, err := svc.AuthorizePowerOfAttorney(ctx, advancedReq)
	if err != nil {
		fmt.Println("[UNEXPECTED] Advanced POA request failed:", err)
		return
	}
	fmt.Println("[POSITIVE] Advanced POA granted! Authorization Code:", resp.AuthorizationCode)
	fmt.Println("Legal Compliance:", resp.LegalCompliance)
	fmt.Println("Audit Record ID:", resp.AuditRecordID)
}
