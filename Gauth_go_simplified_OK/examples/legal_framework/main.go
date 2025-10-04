package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

// Example compliance record for demonstration
type ComplianceRecord struct {
	ID            string `json:"id"`
	Framework     string `json:"framework"`
	RequiredLevel string `json:"required_level"`
	Status        string `json:"status"`
}

func main() {
	fmt.Println("🏛️ Legal Framework Integration Demo")
	fmt.Println("===================================")

	// Create RFC compliant service which includes legal framework validation
	rfcService, err := auth.NewRFCCompliantService("demo-issuer", "demo-audience")
	if err != nil {
		log.Fatalf("Failed to create RFC service: %v", err)
	}

	// Create a power of attorney request with legal framework compliance
	fmt.Println("\n1. Creating Power of Attorney Request with Legal Framework...")
	poaRequest := auth.PowerOfAttorneyRequest{
		ClientID:     "legal-client-123",
		ResponseType: "code",
		Scope:        []string{"legal-transactions", "document-signing"},
		RedirectURI:  "https://legal-app.example.com/callback",
		State:        "legal-state-456",
		PowerType:    "legal_power_of_attorney",
		PrincipalID:  "principal-legal-entity",
		AIAgentID:    "legal-ai-agent-789",
		Jurisdiction: "EU",
		LegalBasis:   "GDPR Article 6(1)(a)",
	}

	// Test authorization with legal framework compliance
	fmt.Println("\n2. Testing Authorization with Legal Framework...")
	ctx := context.Background()
	gauthResponse, err := rfcService.AuthorizeGAuth(ctx, poaRequest)
	if err != nil {
		fmt.Printf("❌ Legal framework authorization failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Legal authorization successful!\n")
	fmt.Printf("   Authorization Code: %s\n", gauthResponse.AuthorizationCode)
	fmt.Printf("   Legal Compliance: %s\n", gauthResponse.LegalCompliance)
	fmt.Printf("   Audit Record ID: %s\n", gauthResponse.AuditRecordID)

	// Create a PoA definition with comprehensive legal framework
	fmt.Println("\n3. Creating Comprehensive PoA Definition...")
	poaDefinition := auth.PoADefinition{
		Principal: auth.Principal{
			Type:     auth.PrincipalTypeOrganization,
			Identity: "Legal Corp Inc.",
			Organization: &auth.Organization{
				Type:                auth.OrgTypeCommercial,
				Name:                "Legal Corporation Inc.",
				RegisterEntry:       "REG-12345-EU",
				ManagingDirector:    "John Legal Director",
				RegisteredAuthority: true,
			},
		},
		Client: auth.ClientAI{
			Type:              auth.ClientTypeLLM,
			Identity:          "legal-ai-agent-789",
			Version:           "2.1.0",
			OperationalStatus: "active",
		},
		AuthorizationType: auth.AuthorizationType{
			RepresentationType:     auth.RepresentationSole,
			RestrictionsExclusions: []string{"no_financial_transfers", "no_asset_disposal"},
			SubProxyAuthority:      false,
			SignatureType:          auth.SignatureSingle,
		},
		ScopeDefinition: auth.ScopeDefinition{
			ApplicableSectors: []auth.IndustrySector{auth.SectorFinancial, auth.SectorProfessional},
			ApplicableRegions: []auth.GeographicScope{
				{Type: auth.GeoTypeRegional, Identifier: "EU", Description: "European Union"},
			},
			AuthorizedActions: auth.AuthorizedActions{
				Decisions:          []auth.DecisionType{auth.DecisionLegal, auth.DecisionInformation},
				NonPhysicalActions: []auth.NonPhysicalAction{auth.ActionResearch, auth.ActionSharing},
			},
		},
		Requirements: auth.Requirements{
			JurisdictionLaw: auth.JurisdictionLaw{
				Language:            "en",
				GoverningLaw:        "EU Digital Services Act",
				PlaceOfJurisdiction: "European Union",
				AttachedDocuments:   []string{"legal-framework-v1.pdf", "compliance-cert.pdf"},
			},
			SecurityCompliance: auth.SecurityCompliance{
				CommunicationProtocols: []string{"HTTPS", "TLS1.3"},
				SecurityProperties:     []string{"encryption", "audit_logging", "legal_compliance"},
				ComplianceInfo:         []string{"GDPR", "eIDAS 2.0", "RFC-0111", "RFC-0115"},
				UpdateMechanism:        "automatic_with_legal_review",
			},
		},
	}

	// Show the created PoA definition structure
	fmt.Println("\n4. PoA Definition Created Successfully!")
	fmt.Printf("   Principal Type: %s\n", poaDefinition.Principal.Type)
	fmt.Printf("   Client AI Type: %s\n", poaDefinition.Client.Type)
	fmt.Printf("   Authorization Type: %s\n", poaDefinition.AuthorizationType.RepresentationType)
	fmt.Printf("   Applicable Sectors: %v\n", poaDefinition.ScopeDefinition.ApplicableSectors)
	fmt.Printf("   Governing Law: %s\n", poaDefinition.Requirements.JurisdictionLaw.GoverningLaw)

	// Show compliance information
	fmt.Println("\n📋 Legal Framework Compliance:")
	fmt.Printf("   • Jurisdiction: %s\n", poaDefinition.Requirements.JurisdictionLaw.PlaceOfJurisdiction)
	fmt.Printf("   • Language: %s\n", poaDefinition.Requirements.JurisdictionLaw.Language)
	fmt.Printf("   • Security Protocols: %v\n", poaDefinition.Requirements.SecurityCompliance.CommunicationProtocols)
	fmt.Printf("   • Compliance Standards: %v\n", poaDefinition.Requirements.SecurityCompliance.ComplianceInfo)

	fmt.Println("\n✅ Legal Framework Integration Demo Completed Successfully!")
}
