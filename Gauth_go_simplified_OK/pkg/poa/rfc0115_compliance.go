// Package poa provides RFC-0115 compliance validation
package poa

import (
	"fmt"
	"strings"
)

// RFC0115Config represents RFC-0115 compliance configuration
type RFC0115Config struct {
	EnableWeb3Integration bool     `json:"enable_web3_integration"`
	EnableAIOperators     bool     `json:"enable_ai_operators"`
	EnableDNAIdentities   bool     `json:"enable_dna_identities"`
	AllowedAITypes        []string `json:"allowed_ai_types"`
	RequiredExclusions    []string `json:"required_exclusions"`
	MandatoryCompliance   bool     `json:"mandatory_compliance"`
}

// ValidateRFC0115Compliance validates RFC-0115 compliance according to Section 2
func ValidateRFC0115Compliance(config RFC0115Config) error {
	var violations []string

	// Section 2: Limitations on the right to make derivative works (Exclusions)

	// Check Web3 exclusion
	if config.EnableWeb3Integration {
		violations = append(violations, "RFC-0115 Section 2: Web3/blockchain technology integration is prohibited")
	}

	// Check AI operators exclusion
	if config.EnableAIOperators {
		violations = append(violations, "RFC-0115 Section 2: AI operators controlling AI deployment lifecycle are prohibited")
	}

	// Check DNA-based identities exclusion
	if config.EnableDNAIdentities {
		violations = append(violations, "RFC-0115 Section 2: DNA-based identities/genetic data are prohibited")
	}

	// Validate mandatory exclusions are enforced
	requiredExclusions := []string{
		"web3_blockchain_technology",
		"ai_controlled_lifecycle",
		"ai_authorization_compliance_tracking",
		"ai_quality_assurance",
		"dna_based_identities",
		"genetic_data_biometrics",
		"ai_dna_identity_tracking",
		"ai_identity_theft_risk_tracking",
	}

	for _, required := range requiredExclusions {
		found := false
		for _, configured := range config.RequiredExclusions {
			if strings.EqualFold(configured, required) {
				found = true
				break
			}
		}
		if !found {
			violations = append(violations, fmt.Sprintf("RFC-0115 Section 2: Missing mandatory exclusion: %s", required))
		}
	}

	// Check that PoA-Definition is used only with GAuth
	if !config.MandatoryCompliance {
		violations = append(violations, "RFC-0115 Section 2: PoA-Definition Must Not be used in contexts other than GAuth")
	}

	if len(violations) > 0 {
		return fmt.Errorf("RFC-0115 compliance violations:\n- %s", strings.Join(violations, "\n- "))
	}

	return nil
}

// CreateRFC0115CompliantConfig creates a compliant RFC-0115 configuration
func CreateRFC0115CompliantConfig() RFC0115Config {
	return RFC0115Config{
		EnableWeb3Integration: false,
		EnableAIOperators:     false,
		EnableDNAIdentities:   false,
		AllowedAITypes: []string{
			"llm",
			"digital_agent",
			"agentic_ai",
			"humanoid_robot",
		},
		RequiredExclusions: []string{
			"web3_blockchain_technology",
			"ai_controlled_lifecycle",
			"ai_authorization_compliance_tracking",
			"ai_quality_assurance",
			"dna_based_identities",
			"genetic_data_biometrics",
			"ai_dna_identity_tracking",
			"ai_identity_theft_risk_tracking",
		},
		MandatoryCompliance: true,
	}
}

// ValidatePoADefinition validates a PoA-Definition for RFC-0115 compliance
func ValidatePoADefinition(poa *PoADefinition) error {
	var violations []string

	// Validate authorized client types are compliant
	allowedTypes := map[ClientType]bool{
		ClientTypeLLM:           true,
		ClientTypeDigitalAgent:  true,
		ClientTypeAgenticAI:     true,
		ClientTypeHumanoidRobot: true,
		ClientTypeOther:         true,
	}

	if !allowedTypes[poa.Parties.AuthorizedClient.Type] {
		violations = append(violations, fmt.Sprintf("Invalid authorized client type: %s", poa.Parties.AuthorizedClient.Type))
	}

	// Validate that exclusions are not violated in the PoA structure
	if poa.Requirements.SecurityCompliance.UpdateMechanism != "" {
		// Check for prohibited AI-controlled update mechanisms
		prohibited := []string{"ai_controlled", "blockchain_based", "web3_integrated", "dna_verified"}
		for _, p := range prohibited {
			if strings.Contains(strings.ToLower(poa.Requirements.SecurityCompliance.UpdateMechanism), p) {
				violations = append(violations, fmt.Sprintf("Prohibited update mechanism detected: %s", p))
			}
		}
	}

	// Validate formal requirements compliance
	if len(poa.Requirements.PowerLimits.ExplicitExclusions) == 0 {
		violations = append(violations, "RFC-0115 requires explicit exclusions to be defined")
	}

	if len(violations) > 0 {
		return fmt.Errorf("PoA-Definition RFC-0115 compliance violations:\n- %s", strings.Join(violations, "\n- "))
	}

	return nil
}
