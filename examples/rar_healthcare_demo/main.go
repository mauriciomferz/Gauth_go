package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// Healthcare AI Scenario:
// A "RadiologyAI" agent (Client) has a Power of Attorney from "Dr. Smith" (Principal)
// to access "Radiology Dept" resources.
// The agent attempts to get a token specifically for "Patient 123's X-Rays".

func main() {
	fmt.Println("=== Healthcare AI RAR Demo ===")

	validator := gauth.NewRARValidator()

	// 2. Define the PoADefinition (Correct Structure)
	// Dr. Smith grants RadiologyAI access to "dept:radiology:images:*"

	poaDef := &poa.PoADefinition{
		Parties: poa.Parties{
			Principal: poa.Principal{
				Identity: "did:web:hospital.com:dr-smith",
				Type:     "Organization",
			},
			AuthorizedClient: poa.AuthorizedClient{
				Identity: "did:web:ai-vendors.com:radiology-bot",
				Type:     "DigitalAgent",
			},
		},
		Authorization: poa.AuthorizationScope{
			AuthorizedActions: poa.AuthorizedActions{
				NonPhysicalActions: []poa.ActionTypeNonPhysical{"read", "analyze"},
			},
		},
		Requirements: poa.Requirements{
			PowerLimits: poa.PowerLimits{
				InteractionBounds: []string{"dept:radiology:images:*"},
			},
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(24 * time.Hour),
			},
		},
	}

	fmt.Printf("[1] PoA Defined: Principal=%s, Agent=%s, Bounds=%v\n",
		poaDef.Parties.Principal.Identity, poaDef.Parties.AuthorizedClient.Identity, poaDef.Requirements.PowerLimits.InteractionBounds)

	// 3. Define the Rich Authorization Request (RAR)
	details := []gauth.AuthorizationDetail{
		{
			Type:       "healthcare_record",
			Locations:  []string{"dept:radiology:images:patient-123"},
			Actions:    []string{"analyze"},
			DataTypes:  []string{"dicom"},
			Identifier: "patient-123",
		},
	}

	rarBytes, _ := json.MarshalIndent(details, "", "  ")
	fmt.Printf("[2] Agent requests Token for specific details:\n%s\n", string(rarBytes))

	// 4. Validate
	fmt.Println("[3] Validating Request against PoA...")
	err := validator.ValidateAuthorizationDetails(poaDef, details)
	if err != nil {
		log.Fatalf("❌ Validation Failed: %v", err)
	}
	fmt.Println("✅ Access Granted! The specific request is within the general PoA scope.")

	// 5. Negative Test
	fmt.Println("\n[4] Negative Test: Agent requests Oncology records...")
	badDetails := []gauth.AuthorizationDetail{
		{
			Type:      "healthcare_record",
			Locations: []string{"dept:oncology:records:patient-999"}, // Mismatch
			Actions:   []string{"analyze"},
		},
	}

	err = validator.ValidateAuthorizationDetails(poaDef, badDetails)
	if err == nil {
		log.Fatal("❌ FAILURE: Should have denied access to Oncology dept")
	}
	fmt.Printf("✅ Correctly Denied: %v\n", err)

	fmt.Println("\n=== Demo Complete ===")
}
