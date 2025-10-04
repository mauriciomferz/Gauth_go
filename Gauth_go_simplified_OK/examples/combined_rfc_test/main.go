// Simple Combined RFC Test
package main

import (
	"fmt"
	"log"

	"github.com/Gimel-Foundation/gauth/pkg/rfc"
)

func main() {
	fmt.Println("🧪 Testing Combined RFC-0111 & RFC-0115 Implementation")
	fmt.Println("══════════════════════════════════════════════════════")

	// Test RFC-0111 Configuration
	fmt.Println("\n📋 Testing RFC-0111 Configuration...")
	rfc0111Config := rfc.CreateRFC0111Config()

	if err := rfc.ValidateRFC0111Config(rfc0111Config); err != nil {
		log.Fatalf("❌ RFC-0111 validation failed: %v", err)
	}
	fmt.Println("✅ RFC-0111 configuration validated successfully")

	// Test RFC-0115 Configuration
	fmt.Println("\n📋 Testing RFC-0115 Configuration...")
	rfc0115Config := rfc.CreateRFC0115PoADefinition()

	if err := rfc.ValidateRFC0115PoADefinition(*rfc0115Config); err != nil {
		log.Fatalf("❌ RFC-0115 validation failed: %v", err)
	}
	fmt.Println("✅ RFC-0115 configuration validated successfully")

	// Test Combined Configuration
	fmt.Println("\n📋 Testing Combined RFC Configuration...")
	combinedConfig := rfc.CreateCombinedRFCConfig()

	if err := rfc.ValidateCombinedRFCConfig(combinedConfig); err != nil {
		log.Fatalf("❌ Combined RFC validation failed: %v", err)
	}
	fmt.Println("✅ Combined RFC configuration validated successfully")

	// Display Exclusions Compliance
	fmt.Println("\n🔒 RFC-0111 Exclusions Compliance:")
	fmt.Printf("  🚫 Web3/Blockchain: %v\n", combinedConfig.RFC0111.Exclusions.Web3Blockchain.Prohibited)
	fmt.Printf("  🚫 AI Operators: %v\n", combinedConfig.RFC0111.Exclusions.AIOperators.Prohibited)
	fmt.Printf("  🚫 DNA Identities: %v\n", combinedConfig.RFC0111.Exclusions.DNABasedIdentities.Prohibited)
	fmt.Printf("  🚫 Decentralized Auth: %v\n", combinedConfig.RFC0111.Exclusions.DecentralizedAuth.Prohibited)

	// Display Integration Status
	fmt.Println("\n🤝 Integration Status:")
	fmt.Printf("  🔗 Integration Level: %s\n", combinedConfig.IntegrationLevel)
	fmt.Printf("  📦 Combined Version: %s\n", combinedConfig.CombinedVersion)

	if combinedConfig.RFC0115 != nil {
		fmt.Printf("  ✅ RFC-0115 PoA Definition: Included\n")
		fmt.Printf("  🤖 Authorized Client Type: %s\n", combinedConfig.RFC0115.Parties.AuthorizedClient.Type)
		fmt.Printf("  🏗️ GAuth Integration: %s role\n", combinedConfig.RFC0115.GAuthContext.PPArchitectureRole)
		fmt.Printf("  🔒 Exclusions Compliant: %v\n", combinedConfig.RFC0115.GAuthContext.ExclusionsCompliant)
	}

	fmt.Println("\n🎉 Combined RFC Implementation Test Completed Successfully!")
	fmt.Println("═══════════════════════════════════════════════════════════")
}
