package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mauriciomferz/AgentAuth/pkg/auth"
)

// truncate safely shortens a string to at most n characters and appends ... if truncated.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func main() {
	passed := 0
	failed := 0
	fmt.Println("🧪 AgentAuth RFC Implementation - Functional Validation Test")
	fmt.Println("========================================================")
	fmt.Println("Testing ACTUAL implementation (not stubs)")
	fmt.Println("")

	ctx := context.Background()

	// Create RFC-compliant service
	rfcService := auth.NewRFCCompliantService()
	fmt.Println("✅ RFC service created")
	// Test 1: RFC 111 with VALID request
	fmt.Println("\n🧪 Test 1: RFC 111 Valid Power-of-Attorney Request")

	validPOARequest := auth.PowerOfAttorneyRequest{
		ClientID:     "ai_trading_bot",
		ResponseType: "code",
		Scope:        "ai_power_of_attorney,financial_transactions",
		RedirectURI:  "https://app.example.com/callback",
		PowerType:    "financial_transactions",
		PrincipalID:  "corp_ceo_123",
		AIAgentID:    "ai_trading_assistant_v2",
		Jurisdiction: "US",
		LegalBasis:   "power_of_attorney_act_2025",
	}

	poaResponse, err := rfcService.AuthorizePowerOfAttorney(ctx, validPOARequest)
	if err != nil {
		fmt.Printf("❌ Valid request failed: %v\n", err)
		failed++
	} else {
		fmt.Printf("✅ Valid request succeeded:\n")
		if len(poaResponse.AuthorizationCode) >= 20 {
			fmt.Printf("   Authorization Code: %s...\n", poaResponse.AuthorizationCode[:20])
		} else {
			fmt.Printf("   Authorization Code: %s\n", poaResponse.AuthorizationCode)
		}
		fmt.Printf("   Legal Compliance: %v\n", poaResponse.LegalCompliance)
		fmt.Printf("   Audit Record: %s\n", poaResponse.AuditRecordID)
		passed++
	}

	// Test 2: RFC 111 with INVALID jurisdiction (should fail)
	fmt.Println("\n🧪 Test 2: RFC 111 Invalid Jurisdiction (Should Fail)")
	fmt.Println("=" + strings.Repeat("=", 55))

	invalidJurisdictionRequest := validPOARequest
	invalidJurisdictionRequest.Jurisdiction = "INVALID_COUNTRY"

	_, err = rfcService.AuthorizePowerOfAttorney(ctx, invalidJurisdictionRequest)
	if err != nil {
		fmt.Printf("✅ Correctly rejected invalid jurisdiction: %v\n", err)
		passed++
	} else {
		fmt.Printf("❌ Failed to reject invalid jurisdiction\n")
		failed++
	}

	// Test 3: RFC 111 with INVALID AI capabilities (should fail)
	fmt.Println("\n🧪 Test 3: RFC 111 Invalid AI Capabilities (Should Fail)")
	fmt.Println("=" + strings.Repeat("=", 56))

	invalidAIRequest := validPOARequest
	invalidAIRequest.Scope = validAIInvalidCapability(validPOARequest.Scope, "nuclear_launch_codes")

	_, err = rfcService.AuthorizePowerOfAttorney(ctx, invalidAIRequest)
	if err != nil {
		fmt.Printf("✅ Correctly rejected invalid AI capabilities: %v\n", err)
		passed++
	} else {
		fmt.Printf("❌ Failed to reject invalid AI capabilities\n")
		failed++
	}

	// Test 4: RFC 115 Valid Delegation
	fmt.Println("\n🧪 Test 4: RFC 115 Valid Advanced Delegation")
	fmt.Println("=" + strings.Repeat("=", 42))
	validDelegationRequest := auth.DelegationRequest{
		PrincipalID:    "corp_ceo_123",
		DelegateID:     "trusted_delegate_456",
		ValidityPeriod: auth.ValidityPeriod{Days: 30},
		AttestationRequirement: auth.AttestationRequirement{
			Attesters: []string{"compliance_officer"},
		},
	}
	delegationResp, err := rfcService.CreateAdvancedDelegation(ctx, validDelegationRequest)
	if err != nil {
		fmt.Printf("❌ Valid delegation request failed: %v\n", err)
		failed++
	} else {
		fmt.Printf("✅ Valid delegation succeeded:\n")
		fmt.Printf("   Delegation ID: %s\n", truncate(delegationResp.DelegationID, 20))
		fmt.Printf("   Status: %s\n", delegationResp.Status)
		fmt.Printf("   Valid Until: %s\n", delegationResp.ValidUntil.Format("2006-01-02"))
		fmt.Printf("   Attestations: %v\n", delegationResp.Attestations)
		passed++
	}

	// Test 5: RFC 115 Invalid delegation period (should fail)
	fmt.Println("\n🧪 Test 5: RFC 115 Invalid Delegation Period (Should Fail)")
	fmt.Println("=" + strings.Repeat("=", 57))
	invalidPeriodRequest := validDelegationRequest
	invalidPeriodRequest.ValidityPeriod.Days = 0
	_, err = rfcService.CreateAdvancedDelegation(ctx, invalidPeriodRequest)
	if err != nil {
		fmt.Printf("✅ Correctly rejected invalid delegation period: %v\n", err)
		passed++
	} else {
		fmt.Printf("❌ Failed to reject invalid delegation period\n")
		failed++
	}

	// Test 6: RFC 115 Insufficient attestation (should fail)
	fmt.Println("\n🧪 Test 6: RFC 115 Insufficient Attestation (Should Fail)")
	fmt.Println("=" + strings.Repeat("=", 52))
	insufficientAttestationRequest := validDelegationRequest
	insufficientAttestationRequest.AttestationRequirement.Attesters = nil
	_, err = rfcService.CreateAdvancedDelegation(ctx, insufficientAttestationRequest)
	if err != nil {
		fmt.Printf("✅ Correctly rejected insufficient attestation: %v\n", err)
		passed++
	} else {
		fmt.Printf("❌ Failed to reject insufficient attestation\n")
		failed++
	}

	// Summary
	fmt.Println("\n🎯 FUNCTIONAL TEST SUMMARY")
	fmt.Println("========================")
	fmt.Printf("Passed: %d  Failed: %d\n", passed, failed)
	if failed == 0 {
		fmt.Println("✅ All implemented validation tests passed")
	} else {
		fmt.Println("❌ Some validation tests failed")
	}
	fmt.Println("")
	fmt.Println("📌 Notes:")
	fmt.Println("   - Delegation tests are placeholders pending full RFC 115 implementation")
	fmt.Println("   - Only jurisdiction & capability negative cases executed")
}

// validAIInvalidCapability returns the original scope string with an injected clearly invalid capability token.
func validAIInvalidCapability(original string, bad string) string {
	parts := strings.Split(original, ",")
	// ensure trimming and uniqueness not strictly required here
	parts = append(parts, bad)
	return strings.Join(parts, ",")
}
