// Official GiFo-RFC-0111 Implementation Demo
//
// Copyright (c) 2025 Gimel Foundation gGmbH i.G.
// Licensed under Apache 2.0
//
// Demonstrates the complete GAuth 1.0 Authorization Framework
// as specified in GiFo-RFC-0111 by Dr. Götz G. Wehberg

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/rfc0111"
)

func main() {
	fmt.Println("=== GiFo-RFC-0111 GAuth 1.0 Authorization Framework Demo ===")
	fmt.Println("Digital Supply Institute")
	fmt.Println("ISBN: 978-3-00-084039-5")
	fmt.Println("Category: Standards Track")
	fmt.Println()
	fmt.Println("Gimel Foundation gGmbH i.G., www.GimelFoundation.com")
	fmt.Println("Operated by Gimel Technologies GmbH")
	fmt.Println("MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert")
	fmt.Println("Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com")
	fmt.Println()

	// Demonstrate RFC-0111 compliant configuration
	fmt.Println("1. RFC-0111 Compliance Validation:")
	fmt.Println("==================================")
	
	config := createRFC0111Config()
	if err := rfc0111.ValidateRFC0111Compliance(config); err != nil {
		log.Fatalf("RFC-0111 compliance violation: %v", err)
	}
	fmt.Println("✅ RFC-0111 Exclusions validated (Web3, AI operators, DNA identities excluded)")
	fmt.Println()

	// Demonstrate core RFC-0111 structures
	fmt.Println("2. Core RFC-0111 Authorization Framework:")
	fmt.Println("=========================================")
	
	// Create Resource Owner (as per RFC-0111 Section 3)
	resourceOwner := createRFC0111ResourceOwner()
	fmt.Printf("Resource Owner: %s (%s)\n", resourceOwner.Identity.Subject, resourceOwner.Type)
	
	// Create Client (AI system as per RFC-0111 Section 3)
	client := createRFC0111Client()
	fmt.Printf("AI Client: %s (%s)\n", client.Identity.AgentID, client.Type)
	
	// Create Extended Token (as per RFC-0111 Section 3)
	extendedToken := createRFC0111ExtendedToken()
	fmt.Printf("Extended Token: %s (valid until %s)\n", 
		extendedToken.TokenID, 
		extendedToken.ValidUntil.Format("2006-01-02 15:04"))
	fmt.Println()

	// Demonstrate P*P Architecture (RFC-0111 Section 3)
	fmt.Println("3. Power*Point (P*P) Architecture:")
	fmt.Println("==================================")
	
	pdp := createRFC0111PowerDecisionPoint()
	fmt.Printf("Power Decision Point: %s (Owner: %s)\n", pdp.ID, pdp.Owner.Identity.Subject)
	
	pip := createRFC0111PowerInformationPoint()
	fmt.Printf("Power Information Point: %s (%d data sources)\n", pip.ID, len(pip.DataSources))
	
	pvp := createRFC0111PowerVerificationPoint()
	fmt.Printf("Power Verification Point: %s (Trust Service: %s)\n", 
		pvp.ID, pvp.TrustServiceProvider)
	fmt.Println()

	// Demonstrate Authorization Request (RFC-0111 Section 6)
	fmt.Println("4. RFC-0111 Authorization Flow:")
	fmt.Println("===============================")
	
	request := createRFC0111Request()
	fmt.Printf("Authorization Request: %s (%s)\n", request.RequestID, request.RequestType)
	fmt.Printf("Requested Action: %s\n", request.Action.Description)
	
	grant := createRFC0111AuthorizationGrant()
	fmt.Printf("Authorization Grant: %s (%s)\n", grant.GrantID, grant.GrantType)
	fmt.Println()

	// Demonstrate Power Management
	fmt.Println("5. Power Management (RFC-0111 Compliant):")
	fmt.Println("=========================================")
	
	power := createRFC0111GrantedPower()
	fmt.Printf("Granted Power: %s (%s)\n", power.PowerID, power.PowerType)
	fmt.Printf("Geographic Scope: %d regions\n", len(power.Scope.Geographic))
	fmt.Printf("Restrictions: %d restrictions\n", len(power.Restrictions))
	fmt.Println()

	// Show JSON serialization for complete structure
	fmt.Println("6. Complete RFC-0111 Structure (JSON):")
	fmt.Println("======================================")
	
	completeStructure := map[string]interface{}{
		"rfc_version": "GiFo-RFC-0111",
		"isbn": "978-3-00-084039-5",
		"resource_owner": resourceOwner,
		"client": client,
		"extended_token": extendedToken,
		"authorization_grant": grant,
		"granted_power": power,
		"pdp_architecture": pdp,
		"compliance": map[string]bool{
			"web3_excluded": config.ExcludeWeb3,
			"ai_operators_excluded": config.ExcludeAIOperators,
			"dna_identities_excluded": config.ExcludeDNAIdentities,
		},
	}
	
	jsonData, err := json.MarshalIndent(completeStructure, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling RFC-0111 structure: %v", err)
	}
	
	// Show first 1000 characters to avoid overwhelming output
	jsonStr := string(jsonData)
	if len(jsonStr) > 1000 {
		fmt.Println(jsonStr[:1000] + "...")
		fmt.Println("[Structure truncated for display - complete implementation available]")
	} else {
		fmt.Println(jsonStr)
	}
	fmt.Println()

	fmt.Println("✅ GiFo-RFC-0111 GAuth 1.0 Authorization Framework demonstration complete")
	fmt.Println("✅ All mandatory exclusions enforced (Section 2)")
	fmt.Println("✅ Complete P*P Architecture implemented")
	fmt.Println("✅ Official Gimel Foundation gGmbH i.G. attribution")
	fmt.Println()
	fmt.Println("For production use, implement concrete service with:")
	fmt.Println("- Real cryptographic implementations")
	fmt.Println("- Proper identity verification")
	fmt.Println("- Commercial register integration")  
	fmt.Println("- Notarization services")
	fmt.Println("- Compliance tracking")
}

func createRFC0111Config() *rfc0111.RFC0111Config {
	return &rfc0111.RFC0111Config{
		AuthorizationServerURL: "https://auth.gimelfoundation.com",
		TrustServiceProvider:   "Gimel Foundation Trust Services",
		RequireNotarization:    true,
		MaxDelegationDepth:     3,
		DefaultTokenValidity:   24 * time.Hour,
		AuditingEnabled:        true,
		ComplianceTrackingEnabled: true,
		
		// RFC-0111 Section 2: Mandatory exclusions for open source
		ExcludeWeb3:         true, // No blockchain/web3 tokens
		ExcludeAIOperators:  true, // No AI controlling the entire process
		ExcludeDNAIdentities: true, // No DNA-based identities
	}
}

func createRFC0111ResourceOwner() *rfc0111.RFC0111ResourceOwner {
	return &rfc0111.RFC0111ResourceOwner{
		ID:   "gimel-foundation-resource-owner",
		Type: rfc0111.RFC0111ResourceOwnerTypeOrganization,
		Identity: rfc0111.RFC0111VerifiedIdentity{
			Subject:          "Gimel Foundation gGmbH i.G.",
			IdentityProvider: "Commercial Register Siegburg",
			VerificationLevel: rfc0111.RFC0111VerificationLevelHigh,
			Attributes: map[string]interface{}{
				"registration": "HRB 18660",
				"address": "Hardtweg 31, D-53639 Königswinter",
				"managing_directors": "Bjørn Baunbæk, Dr. Götz G. Wehberg",
				"chairman": "Daniel Hartert",
			},
			VerifiedAt: time.Now(),
		},
		Authorization: rfc0111.RFC0111ResourceOwnerAuth{
			StatutoryAuthority:   true,
			RegisteredAuthority:  true,
			NotarizationLevel:   rfc0111.RFC0111NotarizationFull,
		},
		Powers: []rfc0111.RFC0111GrantedPower{
			{
				PowerID:   "gimel-foundation-corporate-power",
				PowerType: rfc0111.RFC0111PowerTypeTransaction,
				ValidFrom: time.Now(),
				ValidUntil: time.Now().AddDate(1, 0, 0), // 1 year validity
				Revocable: true,
			},
		},
	}
}

func createRFC0111Client() *rfc0111.RFC0111Client {
	return &rfc0111.RFC0111Client{
		ID:   "gauth-demo-ai-client",
		Type: rfc0111.RFC0111ClientTypeDigitalAgent,
		Identity: rfc0111.RFC0111ClientIdentity{
			AgentID:       "gauth-agent-v1.0",
			SystemVersion: "1.0.0",
			Capabilities:  []string{"transaction", "decision", "audit"},
			TrustLevel:    rfc0111.RFC0111TrustLevelStandard,
			CertificationLevel: rfc0111.RFC0111CertificationStandard,
		},
		Owner: rfc0111.RFC0111ClientOwner{
			ID: "gimel-foundation-client-owner",
			Identity: rfc0111.RFC0111VerifiedIdentity{
				Subject:          "Gimel Foundation gGmbH i.G.",
				IdentityProvider: "Commercial Register Siegburg",
				VerificationLevel: rfc0111.RFC0111VerificationLevelHigh,
				VerifiedAt: time.Now(),
			},
		},
		Capabilities: []rfc0111.RFC0111ClientCapability{
			rfc0111.RFC0111CapabilityTransaction,
			rfc0111.RFC0111CapabilityDecision,
			rfc0111.RFC0111CapabilityAction,
		},
		Status:  rfc0111.RFC0111ClientStatusActive,
		Version: "1.0.0",
	}
}

func createRFC0111ExtendedToken() *rfc0111.RFC0111ExtendedToken {
	return &rfc0111.RFC0111ExtendedToken{
		TokenID:    "rfc0111-token-" + fmt.Sprintf("%d", time.Now().Unix()),
		ClientID:   "gauth-demo-ai-client",
		ResourceID: "gimel-foundation-resource",
		Scope: rfc0111.RFC0111AuthorizationScope{
			Resources:    []string{"commercial_registry", "corporate_documents"},
			Actions:      []string{"read", "verify", "audit"},
			Transactions: []string{"verification_request", "compliance_check"},
			Geographic: []rfc0111.RFC0111GeographicScope{
				{Type: "country", Identifier: "DE"},
				{Type: "region", Identifier: "EU"},
			},
			Temporal: &rfc0111.RFC0111TemporalScope{
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(24 * time.Hour),
				TimeZone:   "Europe/Berlin",
				BusinessHours: &rfc0111.RFC0111BusinessHours{
					Days:  []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
					Start: "09:00",
					End:   "17:00",
				},
			},
		},
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour),
		Revoked:    false,
	}
}

func createRFC0111Request() *rfc0111.RFC0111Request {
	return &rfc0111.RFC0111Request{
		RequestID:   "rfc0111-request-" + fmt.Sprintf("%d", time.Now().Unix()),
		ClientID:    "gauth-demo-ai-client",
		ResourceID:  "gimel-foundation-resource",
		RequestType: rfc0111.RFC0111RequestTypeTransaction,
		Action: rfc0111.RFC0111RequestedAction{
			Type:        "verification",
			Description: "Verify commercial registry entry for Gimel Foundation gGmbH i.G.",
			Parameters: map[string]interface{}{
				"registry": "Siegburg HRB 18660",
				"verification_type": "statutory_authority",
			},
			ExpectedOutcome: "Verified statutory authority for RFC-0111 compliance",
		},
		Urgency: rfc0111.RFC0111RequestUrgencyNormal,
	}
}

func createRFC0111AuthorizationGrant() *rfc0111.RFC0111AuthorizationGrant {
	return &rfc0111.RFC0111AuthorizationGrant{
		GrantID:         "rfc0111-grant-" + fmt.Sprintf("%d", time.Now().Unix()),
		ClientID:        "gauth-demo-ai-client",
		ResourceOwnerID: "gimel-foundation-resource-owner",
		GrantType:       rfc0111.RFC0111GrantTypeAuthorizationCode,
		ValidFrom:       time.Now(),
		ValidUntil:      time.Now().Add(1 * time.Hour),
		Conditions: []rfc0111.RFC0111GrantCondition{
			{
				ConditionID: "audit-required",
				Type:        rfc0111.RFC0111ConditionTypeAudit,
				Description: "All actions must be audited for RFC-0111 compliance",
				Required:    true,
			},
		},
	}
}

func createRFC0111GrantedPower() *rfc0111.RFC0111GrantedPower {
	return &rfc0111.RFC0111GrantedPower{
		PowerID:   "rfc0111-power-" + fmt.Sprintf("%d", time.Now().Unix()),
		PowerType: rfc0111.RFC0111PowerTypeDecision,
		Scope: rfc0111.RFC0111AuthorizationScope{
			Actions: []string{"verify_identity", "check_compliance", "audit_transaction"},
			Geographic: []rfc0111.RFC0111GeographicScope{
				{Type: "country", Identifier: "DE"},
				{Type: "region", Identifier: "EU"},
			},
			Monetary: &rfc0111.RFC0111MonetaryScope{
				Currency:   "EUR",
				MaxAmount:  10000.00,
				DailyLimit: 50000.00,
			},
		},
		Restrictions: []rfc0111.RFC0111PowerRestriction{
			{
				RestrictionID: "business-hours-only",
				Type:          rfc0111.RFC0111RestrictionTypeTemporal,
				Description:   "Actions only allowed during business hours (9-17 CET)",
			},
			{
				RestrictionID: "dual-approval-required",
				Type:          rfc0111.RFC0111RestrictionTypeApproval,
				Description:   "Transactions >€5000 require dual approval",
				RequiredApproval: true,
			},
		},
		DelegationRules: []rfc0111.RFC0111DelegationRule{
			{
				RuleID:   "limited-delegation",
				Type:     rfc0111.RFC0111DelegationTypeLimited,
				MaxDepth: 2,
				AllowedTypes: []rfc0111.RFC0111ClientType{
					rfc0111.RFC0111ClientTypeDigitalAgent,
					rfc0111.RFC0111ClientTypeAgenticAI,
				},
				RequiresApproval: true,
			},
		},
		ValidFrom: time.Now(),
		ValidUntil: time.Now().AddDate(0, 6, 0), // 6 months validity
		Revocable: true,
	}
}

func createRFC0111PowerDecisionPoint() *rfc0111.RFC0111PowerDecisionPoint {
	return &rfc0111.RFC0111PowerDecisionPoint{
		ID: "gimel-foundation-pdp",
		Owner: rfc0111.RFC0111ClientOwner{
			ID: "gimel-foundation-client-owner",
			Identity: rfc0111.RFC0111VerifiedIdentity{
				Subject:          "Gimel Foundation gGmbH i.G.",
				IdentityProvider: "Commercial Register Siegburg",
				VerificationLevel: rfc0111.RFC0111VerificationLevelHigh,
				VerifiedAt: time.Now(),
			},
		},
		Policies: []rfc0111.RFC0111AuthorizationPolicy{
			{
				PolicyID: "rfc0111-compliance-policy",
				Name:     "RFC-0111 Compliance Policy",
				Version:  "1.0",
				Rules: []rfc0111.RFC0111PolicyRule{
					{
						RuleID:    "exclude-web3",
						Condition: "technology_type != 'web3'",
						Effect:    rfc0111.RFC0111PolicyEffectAllow,
					},
					{
						RuleID:    "exclude-ai-operators",
						Condition: "ai_control_level != 'full_process'",
						Effect:    rfc0111.RFC0111PolicyEffectAllow,
					},
				},
				Priority:   1,
				Enabled:    true,
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().AddDate(1, 0, 0),
			},
		},
	}
}

func createRFC0111PowerInformationPoint() *rfc0111.RFC0111PowerInformationPoint {
	return &rfc0111.RFC0111PowerInformationPoint{
		ID: "gimel-foundation-pip",
		DataSources: []rfc0111.RFC0111InformationSource{
			{
				SourceID:     "commercial-register-siegburg",
				Type:         rfc0111.RFC0111SourceTypeCommercialRegister,
				URL:          "https://commercial-register.siegburg.de",
				Capabilities: []string{"verify_registration", "check_authority"},
				TrustLevel:   rfc0111.RFC0111TrustLevelHigh,
			},
			{
				SourceID:     "gimel-foundation-identity",
				Type:         rfc0111.RFC0111SourceTypeIdentityProvider,
				URL:          "https://identity.gimelfoundation.com",
				Capabilities: []string{"verify_identity", "check_authorization"},
				TrustLevel:   rfc0111.RFC0111TrustLevelCertified,
			},
		},
		Capabilities: []string{"identity_verification", "authority_validation", "compliance_checking"},
	}
}

func createRFC0111PowerVerificationPoint() *rfc0111.RFC0111PowerVerificationPoint {
	return &rfc0111.RFC0111PowerVerificationPoint{
		ID:                   "gimel-foundation-pvp",
		TrustServiceProvider: "Gimel Foundation Trust Services",
		VerificationMethods: []rfc0111.RFC0111VerificationMethod{
			{
				MethodID:    "commercial-register-verification",
				Type:        rfc0111.RFC0111VerificationTypeCertificate,
				Description: "Commercial register certificate verification",
				TrustLevel:  rfc0111.RFC0111TrustLevelHigh,
				Required:    true,
			},
			{
				MethodID:    "notary-verification",
				Type:        rfc0111.RFC0111VerificationTypeNotary,
				Description: "Notarized document verification",
				TrustLevel:  rfc0111.RFC0111TrustLevelCertified,
				Required:    false,
			},
		},
	}
}