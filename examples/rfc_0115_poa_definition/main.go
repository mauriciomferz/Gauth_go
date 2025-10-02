// Official GiFo-RFC-0115 PoA-Definition Implementation Demo
//
// Copyright (c) 2025 Gimel Foundation gGmbH i.G.
// Licensed under Apache 2.0
//
// Demonstrates the complete Power-of-Attorney Credential Definition (PoA-Definition)
// structure as specified in GiFo-RFC-0115 by Dr. Götz G. Wehberg
//
// Digital Supply Institute - Standards Track Document
// Obsoletes: - 15. September 2025

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/poa"
)

func main() {
	fmt.Println("=== GiFo-RFC-0115 PoA-Definition Implementation Demo ===")
	fmt.Println("Digital Supply Institute")
	fmt.Println("Category: Standards Track")
	fmt.Println("Obsoletes: - 15. September 2025")
	fmt.Println()
	fmt.Println("Gimel Foundation gGmbH i.G., www.GimelFoundation.com")
	fmt.Println("Operated by Gimel Technologies GmbH")
	fmt.Println("MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert")
	fmt.Println("Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com")
	fmt.Println()

	// Create a comprehensive PoA-Definition example
	poaDefinition := createExamplePoADefinition()

	// Convert to JSON for demonstration
	jsonData, err := json.MarshalIndent(poaDefinition, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling PoA-Definition: %v", err)
	}

	fmt.Println("Complete RFC-0115 PoA-Definition Structure:")
	fmt.Println("==========================================")
	fmt.Println(string(jsonData))
	fmt.Println()

	// Demonstrate type safety
	fmt.Println("Type Safety Demonstration:")
	fmt.Println("========================")
	fmt.Printf("Principal Type: %s\n", poaDefinition.Parties.Principal.Type)
	fmt.Printf("Organization Type: %s\n", poaDefinition.Parties.Principal.Organization.Type)
	fmt.Printf("Authorization Type: %s representation\n", poaDefinition.Authorization.AuthorizationType.RepresentationType)
	fmt.Printf("Signature Type: %s\n", poaDefinition.Authorization.AuthorizationType.SignatureType)
	fmt.Printf("Validity Period: %s to %s\n", 
		poaDefinition.Requirements.ValidityPeriod.StartTime.Format("2006-01-02"),
		poaDefinition.Requirements.ValidityPeriod.EndTime.Format("2006-01-02"))
	fmt.Println()

	// Demonstrate RFC-0115 compliance validation
	fmt.Println("RFC-0115 Compliance Validation:")
	fmt.Println("==============================")
	
	// Create RFC-0115 compliant configuration
	config := poa.CreateRFC0115CompliantConfig()
	if err := poa.ValidateRFC0115Compliance(config); err != nil {
		log.Fatalf("RFC-0115 compliance validation failed: %v", err)
	}
	fmt.Println("✅ RFC-0115 exclusions validated (Web3, AI operators, DNA identities excluded)")
	
	// Validate the PoA-Definition itself
	if err := poa.ValidatePoADefinition(poaDefinition); err != nil {
		log.Fatalf("PoA-Definition validation failed: %v", err)
	}
	fmt.Println("✅ PoA-Definition structure validated for RFC-0115 compliance")
	fmt.Println("✅ Mandatory exclusions enforced (Section 2)")
	fmt.Println("✅ Official Gimel Foundation gGmbH i.G. attribution")
	fmt.Println()

	fmt.Println("✅ RFC-0115 PoA-Definition implementation successfully demonstrated")
}

func createExamplePoADefinition() *poa.PoADefinition {
	// Create validity period
	startTime := time.Now()
	endTime := startTime.AddDate(1, 0, 0) // Valid for 1 year

	return &poa.PoADefinition{
		// A. Parties (RFC-0115 Section 3.A)
		Parties: poa.Parties{
			Principal: poa.Principal{
				Type:     poa.PrincipalTypeOrganization,
				Identity: "gimel-foundation-ggmbh-ig",
				Organization: &poa.Organization{
					Type:                poa.OrgTypeNonProfit,
					Name:                "Gimel Foundation gGmbH i.G.",
					RegisterEntry:       "Siegburg HRB 18660",
					ManagingDirector:    "Bjørn Baunbæk, Dr. Götz G. Wehberg",
					RegisteredAuthority: true,
				},
			},
			Representative: &poa.Representative{
				ClientOwner: &poa.ClientOwnerInfo{
					Name:                      "Daniel Hartert",
					RegisteredPowerOfAttorney: true,
					CommercialRegisterEntry:   true,
				},
			},
			AuthorizedClient: poa.AuthorizedClient{
				Type:              poa.ClientTypeLLM,
				Identity:          "gauth-ai-client-v1",
				Version:           "1.0.0",
				OperationalStatus: "active",
			},
		},

		// B. Type and Scope of Authorization (RFC-0115 Section 3.B)
		Authorization: poa.AuthorizationScope{
			AuthorizationType: poa.AuthorizationType{
				RepresentationType: poa.RepresentationSole,
				Restrictions:       []string{"Financial transactions require dual approval"},
				SubProxyAuthority:  false,
				SignatureType:      poa.SignatureSingle,
			},
			ApplicableSectors: []poa.IndustrySector{
				poa.SectorInformationComm,
				poa.SectorProfessional,
				poa.SectorFinancialInsurance,
			},
			ApplicableRegions: []poa.GeographicScope{
				{Type: poa.GeoTypeNational, Identifier: "DE"}, // Germany
				{Type: poa.GeoTypeRegional, Identifier: "EU"}, // European Union
			},
			AuthorizedActions: poa.AuthorizedActions{
				Transactions: []poa.Transaction{
					poa.TransactionLoan,
					poa.TransactionPurchase,
				},
				Decisions: []poa.Decision{
					poa.DecisionFinancial,
					poa.DecisionStrategic,
					poa.DecisionInfoSharing,
				},
				NonPhysicalActions: []poa.NonPhysicalAction{
					poa.ActionResearching,
					poa.ActionBrainstorming,
				},
			},
		},

		// C. Requirements (RFC-0115 Section 3.C)
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime:             startTime,
				EndTime:               endTime,
				AutoRenewalConditions: []string{"Subject to annual review and approval"},
				TerminationConditions: []string{"30 days written notice", "Material breach of terms"},
			},
			FormalRequirements: poa.FormalRequirements{
				NotarialCertification:  true,
				IDVerificationRequired: true,
				DigitalSignatures:      true,
			},
			PowerLimits: poa.PowerLimits{
				PowerLevels: []string{
					"Level 1: Information sharing and basic research",
					"Level 2: Strategic recommendations with approval",
				},
				InteractionBounds: []string{
					"Cannot initiate financial transactions > €10,000 without dual approval",
					"Must maintain audit trail for all decisions",
				},
				ToolLimitations: []string{
					"No access to production financial systems",
					"Read-only access to customer data",
				},
				QuantumResistance:  true,
				ExplicitExclusions: []string{
					"Web3/blockchain technology for extended tokens (RFC-0115 Section 2)",
					"AI-controlled AI deployment lifecycle (RFC-0115 Section 2)",
					"AI-based authorization compliance tracking (RFC-0115 Section 2)",
					"AI quality assurance systems (RFC-0115 Section 2)",
					"DNA-based identities or genetic data biometrics (RFC-0115 Section 2)",
					"AI tracking of DNA identity quality (RFC-0115 Section 2)",
					"AI-based identity theft risk tracking (RFC-0115 Section 2)",
					"Legal representation in court proceedings",
					"Personnel hiring/firing decisions",
					"Regulatory filings and submissions",
				},
			},
			RightsObligations: poa.RightsObligations{
				ReportingDuties: []string{
					"Weekly activity reports to designated supervisor",
					"Immediate notification of any security incidents",
				},
				LiabilityRules: []string{
					"Limited liability within authorized scope",
					"Full liability for unauthorized actions",
				},
				CompensationRules: []string{
					"No compensation for AI client operations",
					"Standard corporate indemnification applies",
				},
			},
			SpecialConditions: poa.SpecialConditions{
				ConditionalEffectiveness: []string{
					"Effective only during business hours (9-17 CET)",
					"Suspended during declared system maintenance windows",
				},
				ImmediateNotification: []string{
					"Security breaches or attempted unauthorized access",
					"System failures affecting decision-making capabilities",
				},
			},
			DeathIncapacityRules: poa.DeathIncapacityRules{
				ContinuationOnDeath:    false,
				IncapacityInstructions: "Power of attorney immediately revoked upon incapacity of any managing director",
			},
			SecurityCompliance: poa.SecurityCompliance{
				CommunicationProtocols: []string{
					"TLS 1.3 for all communications",
					"End-to-end encryption for sensitive data",
				},
				SecurityProperties: []string{
					"Multi-factor authentication required",
					"Zero-trust architecture compliance",
					"Quantum-resistant cryptographic algorithms",
				},
				ComplianceInfo: []string{
					"GDPR compliant data handling",
					"ISO 27001 security management",
					"German regulatory compliance (BaFin)",
				},
				UpdateMechanism: "Automated security updates with 24h approval window",
			},
			JurisdictionLaw: poa.JurisdictionLaw{
				Language:            "German",
				GoverningLaw:        "German Federal Law",
				PlaceOfJurisdiction: "Königswinter, Germany",
				AttachedDocuments: []string{
					"Gimel Foundation Articles of Association",
					"Corporate Power of Attorney Certificate",
					"Technical Implementation Specifications",
				},
			},
			ConflictResolution: poa.ConflictResolution{
				ArbitrationJurisdiction: "German Arbitration Institute (DIS), Cologne",
			},
		},
	}
}