// Simple Combined RFC Test
// Demonstrates validation and compliance checks for RFC-0111 & RFC-0115 configurations.
// Shows how to create, validate, and inspect combined RFC configuration for AgentAuth.

package main

import (
	"fmt"
	"log"

	"github.com/mauriciomferz/AgentAuth/pkg/rfc"
)

func main() {
	fmt.Println("🧪 Testing Combined RFC-0111 & RFC-0115 Implementation")
	fmt.Println("══════════════════════════════════════════════════════")

	// Test Combined Configuration using existing helper
	fmt.Println("\n📋 Testing Combined RFC Configuration...")
	combinedConfig := rfc.CreateCombinedRFCConfig()

	if err := rfc.ValidateCombinedRFCConfig(combinedConfig); err != nil {
		log.Fatalf("❌ Combined RFC validation failed: %v", err)
	}
	fmt.Println("✅ Combined RFC configuration validated successfully")

	// Display Exclusions Compliance (structured fields)
	fmt.Println("\n� RFC-0111 Exclusions Compliance:")
	ex := combinedConfig.AAP001.Exclusions
	fmt.Printf("  🚫 Web3 Blockchain: prohibited=%v license_required=%v\n", ex.Web3Blockchain.Prohibited, ex.Web3Blockchain.LicenseRequired)
	fmt.Printf("  � AI Operators: prohibited=%v license_required=%v\n", ex.AIOperators.Prohibited, ex.AIOperators.LicenseRequired)
	fmt.Printf("  🚫 DNA Based Identities: prohibited=%v license_required=%v\n", ex.DNABasedIdentities.Prohibited, ex.DNABasedIdentities.LicenseRequired)
	fmt.Printf("  🚫 Decentralized Auth: prohibited=%v license_required=%v\n", ex.DecentralizedAuth.Prohibited, ex.DecentralizedAuth.LicenseRequired)
	fmt.Printf("  � Enforcement Level: %s\n", ex.EnforcementLevel)

	// Display Integration Status
	fmt.Println("\n🤝 Integration Status:")
	fmt.Printf("  🔗 Integration Level: %s\n", combinedConfig.IntegrationLevel)
	fmt.Printf("  📦 Combined Version: %s\n", combinedConfig.CombinedVersion)

	fmt.Printf("  ✅ RFC-0115 PoA Definition: Included\n")
	// Create detailed PoA definition and show nested data
	poaDef := rfc.CreateDefaultPoADefinition(combinedConfig.AAP002.PoADefinition)
	fmt.Printf("  🤖 Authorized Client Type: %s\n", poaDef.Parties.AuthorizedClient.Type)
	fmt.Printf("  🏗️ AgentAuth Integration: %s role\n", poaDef.AgentAuthContext.PPArchitectureRole)
	fmt.Printf("  🔒 Exclusions Compliant: %v\n", poaDef.AgentAuthContext.ExclusionsCompliant)

	fmt.Println("\n🎉 Combined RFC Implementation Test Completed Successfully!")
	fmt.Println("═══════════════════════════════════════════════════════════")
}
