// Combined RFC-0111 & RFC-0115 Implementation Demo
//
// This example demonstrates the unified implementation of:
// - GiFo-RFC-0111: The GAuth 1.0 Authorization Framework (ISBN: 978-3-00-084039-5)
// - GiFo-RFC-0115: Power-of-Attorney Credential Definition (PoA-Definition)
//
// Copyright (c) 2025 Gimel Foundation gGmbH i.G.
// Licensed under Apache 2.0
//
// Official Gimel Foundation Implementation
// Gimel Foundation gGmbH i.G., www.GimelFoundation.com
// Operated by Gimel Technologies GmbH
// MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartett
// Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/mauriciomferz/Gauth_go/pkg/rfc"
)

func displayRFC0111Exclusions(ex rfc.RFC0111Exclusions) {
	fmt.Printf("  🚫 Web3 Blockchain: prohibited=%v license_required=%v\n", ex.Web3Blockchain.Prohibited, ex.Web3Blockchain.LicenseRequired)
	fmt.Printf("  🚫 AI Operators: prohibited=%v license_required=%v\n", ex.AIOperators.Prohibited, ex.AIOperators.LicenseRequired)
	fmt.Printf("  🚫 DNA Based Identities: prohibited=%v license_required=%v\n", ex.DNABasedIdentities.Prohibited, ex.DNABasedIdentities.LicenseRequired)
	fmt.Printf("  🚫 Decentralized Auth: prohibited=%v license_required=%v\n", ex.DecentralizedAuth.Prohibited, ex.DecentralizedAuth.LicenseRequired)
	fmt.Printf("  🔐 Enforcement Level: %s\n", ex.EnforcementLevel)
}

func main() {
	fmt.Println("🚀 Combined RFC-0111 & RFC-0115 Implementation Demo")
	fmt.Println("═══════════════════════════════════════════════════")

	// Create combined RFC configuration
	fmt.Println("\n📋 Creating Combined RFC Configuration...")
	combinedConfig := rfc.CreateCombinedRFCConfig()

	// Validate the combined configuration
	fmt.Println("\n🔍 Validating Combined RFC Configuration...")
	if err := rfc.ValidateCombinedRFCConfig(combinedConfig); err != nil {
		log.Fatalf("❌ Combined RFC validation failed: %v", err)
	}
	fmt.Println("✅ Combined RFC configuration validated successfully")

	// Display RFC-0111 compliance
	fmt.Println("\n🔒 RFC-0111 Exclusions Compliance:")
	displayRFC0111Exclusions(combinedConfig.RFC0111.Exclusions)

	// Display RFC-0111 PP Architecture
	fmt.Println("\n🏗️ RFC-0111 Power*Point Architecture:")
	displayPPArchitecture(combinedConfig.RFC0111.PPArchitecture)

	// Display RFC-0115 PoA Definition
	if combinedConfig.RFC0115 != nil {
		fmt.Println("\n📄 RFC-0115 Power-of-Attorney Definition:")
		poaDefinition := rfc.CreateDefaultPoADefinition(combinedConfig.RFC0115.PoADefinition)
		displayPoADefinition(poaDefinition)
	}

	// Display Integration Status
	fmt.Println("\n🤝 RFC Integration Status:")
	displayIntegrationStatus(*combinedConfig)

	// JSON Serialization Test
	fmt.Println("\n💾 JSON Serialization Test:")
	jsonData, err := json.MarshalIndent(combinedConfig, "", "  ")
	if err != nil {
		log.Fatalf("❌ JSON serialization failed: %v", err)
	}

	fmt.Printf("✅ Combined configuration serialized successfully (%d bytes)\n", len(jsonData))

	// Create specific AI client configurations
	fmt.Println("\n🤖 AI Client Configurations:")
	demonstrateAIClientConfigs()
	fmt.Println("\n🎉 Combined RFC Implementation Demo Completed Successfully!")
	fmt.Println("════════════════════════════════════════════════════════")
}

func displayPPArchitecture(pp rfc.RFC0111PPArchitecture) {
	fmt.Printf("  🛡️ PEP (Power Enforcement Point):\n")
	fmt.Printf("    - Supply Side: %s (%s)\n", pp.PEP.SupplySide.Entity, pp.PEP.SupplySide.Status)
	fmt.Printf("    - Demand Side: %s (%s)\n", pp.PEP.DemandSide.Entity, pp.PEP.DemandSide.Status)

	fmt.Printf("  🎯 PDP (Power Decision Point): %s\n", pp.PDP.PrimaryPDP)
	fmt.Printf("  📊 PIP (Power Information Point): %s\n", pp.PIP.AuthorizationServer)
	fmt.Printf("  🔧 PAP (Power Administration Point): %s\n", pp.PAP.ClientOwnerAuthorizer)
	fmt.Printf("  ✅ PVP (Power Verification Point): %s\n", pp.PVP.TrustServiceProvider)
}

func displayPoADefinition(poa rfc.RFC0115PoADefinition) {
	fmt.Printf("  👤 Principal: %s (%s)\n",
		poa.Parties.Principal.Identity, poa.Parties.Principal.Type)

	if poa.Parties.Principal.Organization != nil {
		fmt.Printf("    - Organization: %s (%s)\n",
			poa.Parties.Principal.Organization.Name, poa.Parties.Principal.Organization.Type)
		fmt.Printf("    - Register Entry: %s\n", poa.Parties.Principal.Organization.RegisterEntry)
	}

	fmt.Printf("  🤖 Authorized Client: %s (%s)\n",
		poa.Parties.AuthorizedClient.Identity, poa.Parties.AuthorizedClient.Type)
	fmt.Printf("    - Status: %s\n", poa.Parties.AuthorizedClient.OperationalStatus)

	fmt.Printf("  🌍 Geographic Scope: %d regions\n", len(poa.Authorization.ApplicableRegions))
	for _, region := range poa.Authorization.ApplicableRegions {
		fmt.Printf("    - %s: %s (%s)\n", region.Name, region.Identifier, region.Type)
	}

	fmt.Printf("  🏭 Industry Sectors: %d sectors\n", len(poa.Authorization.ApplicableSectors))

	fmt.Printf("  🔗 GAuth Integration:\n")
	fmt.Printf("    - PP Role: %s\n", poa.GAuthContext.PPArchitectureRole)
	fmt.Printf("    - Exclusions Compliant: %v\n", poa.GAuthContext.ExclusionsCompliant)
	fmt.Printf("    - AI Governance Level: %s\n", poa.GAuthContext.AIGovernanceLevel)
}

func displayIntegrationStatus(config rfc.CombinedRFCConfig) {
	fmt.Printf("  🔗 Integration Level: %s\n", config.IntegrationLevel)
	fmt.Printf("  📦 Combined Version: %s\n", config.CombinedVersion)

	fmt.Printf("  🔄 Compatibility Matrix:\n")
	for component, version := range config.Compatibility {
		fmt.Printf("    - %s: %s\n", component, version)
	}
}

func demonstrateAIClientConfigs() {
	// Digital Agent Configuration
	fmt.Println("  🤖 Digital Agent Configuration:")
	digitalAgent := createDigitalAgentConfig()
	fmt.Printf("    - Type: %s\n", digitalAgent.Type)
	fmt.Printf("    - Identity: %s\n", digitalAgent.Identity)
	fmt.Printf("    - Autonomy Level: %s\n", digitalAgent.AutonomyLevel)
	fmt.Printf("    - Capabilities: %v\n", digitalAgent.AICapabilities)

	// Agentic AI Configuration
	fmt.Println("  🤖🤖 Agentic AI Team Configuration:")
	agenticAI := createAgenticAIConfig()
	fmt.Printf("    - Type: %s\n", agenticAI.Type)
	fmt.Printf("    - Identity: %s\n", agenticAI.Identity)
	fmt.Printf("    - Autonomy Level: %s\n", agenticAI.AutonomyLevel)
	fmt.Printf("    - Capabilities: %v\n", agenticAI.AICapabilities)

	// Humanoid Robot Configuration
	fmt.Println("  🤖👤 Humanoid Robot Configuration:")
	humanoidRobot := createHumanoidRobotConfig()
	fmt.Printf("    - Type: %s\n", humanoidRobot.Type)
	fmt.Printf("    - Identity: %s\n", humanoidRobot.Identity)
	fmt.Printf("    - Autonomy Level: %s\n", humanoidRobot.AutonomyLevel)
	fmt.Printf("    - Capabilities: %v\n", humanoidRobot.AICapabilities)
}

func createDigitalAgentConfig() rfc.RFC0111Client {
	return rfc.RFC0111Client{
		Type:     rfc.RFC0111ClientTypeDigitalAgent,
		Identity: "digital_agent_v1_0",
		AICapabilities: []string{
			"natural_language_processing",
			"decision_making",
			"transaction_processing",
			"communication",
			"reasoning",
		},
		AutonomyLevel:  "supervised",
		RequestTypes:   []string{"transactions", "decisions", "actions", "communications"},
		ComplianceMode: "strict_rfc_0111",
	}
}

func createAgenticAIConfig() rfc.RFC0111Client {
	return rfc.RFC0111Client{
		Type:     rfc.RFC0111ClientTypeAgenticAI,
		Identity: "agentic_ai_team_v1_0",
		AICapabilities: []string{
			"multi_agent_coordination",
			"distributed_decision_making",
			"collaborative_reasoning",
			"task_delegation",
			"team_communication",
			"consensus_building",
		},
		AutonomyLevel:  "semi_autonomous",
		RequestTypes:   []string{"complex_transactions", "strategic_decisions", "coordinated_actions"},
		ComplianceMode: "enterprise_rfc_0111",
	}
}

func createHumanoidRobotConfig() rfc.RFC0111Client {
	return rfc.RFC0111Client{
		Type:     rfc.RFC0111ClientTypeHumanoidRobot,
		Identity: "humanoid_robot_v2_1",
		AICapabilities: []string{
			"physical_interaction",
			"spatial_reasoning",
			"human_robot_interaction",
			"motor_control",
			"sensory_processing",
			"safety_protocols",
		},
		AutonomyLevel:  "supervised_physical",
		RequestTypes:   []string{"physical_actions", "safety_decisions", "interaction_protocols"},
		ComplianceMode: "safety_critical_rfc_0111",
	}
}
