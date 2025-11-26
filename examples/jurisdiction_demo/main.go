package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/jurisdiction"
)

func main() {
	fmt.Println("=== Jurisdiction-Specific Enforcement Demo ===")

	// Create enforcement engine
	integration := jurisdiction.NewServerIntegration()

	// Set up audit callback
	var auditLog []jurisdiction.EnforcementDecision
	integration.SetAuditCallback(func(decision jurisdiction.EnforcementDecision) {
		auditLog = append(auditLog, decision)
		fmt.Printf("📋 AUDIT: Request %s - Decision: %v, Jurisdiction: %s\n",
			decision.RequestID, decision.Allowed, decision.Jurisdiction)
	})

	ctx := context.Background()

	fmt.Println("🌍 Scenario 1: US Trade Execution (within value limit)")
	fmt.Println("─────────────────────────────────────────────")
	result1, _ := integration.EnforceJurisdiction(ctx,
		"trader@wallstreet.com",
		"trading-system",
		"trade_execution",
		map[string]interface{}{
			"jurisdiction": "US",
			"entity_type":  "corporation",
			"value":        5000000.0, // $5M - within $10M limit
		})
	printDecision(result1, "US Trade within limit")

	fmt.Println("\n🚫 Scenario 2: US Trade Execution (exceeds value limit)")
	fmt.Println("─────────────────────────────────────────────")
	result2, _ := integration.EnforceJurisdiction(ctx,
		"trader@wallstreet.com",
		"trading-system",
		"trade_execution",
		map[string]interface{}{
			"jurisdiction": "US",
			"entity_type":  "corporation",
			"value":        15000000.0, // $15M - exceeds $10M limit
		})
	printDecision(result2, "US Trade exceeds limit")

	fmt.Println("\n🇪🇺 Scenario 3: EU GDPR Data Processing (with consent)")
	fmt.Println("─────────────────────────────────────────────")
	result3, _ := integration.EnforceJurisdiction(ctx,
		"analyst@eurobank.eu",
		"customer-data",
		"gdpr_data_processing",
		map[string]interface{}{
			"jurisdiction": "EU",
			"entity_type":  "corporation",
			"gdpr_consent": true,
		})
	printDecision(result3, "EU GDPR with consent")

	fmt.Println("\n🚫 Scenario 4: EU GDPR Data Processing (without consent)")
	fmt.Println("─────────────────────────────────────────────")
	result4, _ := integration.EnforceJurisdiction(ctx,
		"analyst@eurobank.eu",
		"customer-data",
		"gdpr_data_processing",
		map[string]interface{}{
			"jurisdiction": "EU",
			"entity_type":  "corporation",
			"gdpr_consent": false,
		})
	printDecision(result4, "EU GDPR without consent")

	fmt.Println("\n🌐 Scenario 5: Cross-Border Data Transfer (EU to UK - allowed)")
	fmt.Println("─────────────────────────────────────────────")
	result5, _ := integration.EnforceJurisdiction(ctx,
		"compliance@eubank.com",
		"customer-data",
		"personal_data_transfer",
		map[string]interface{}{
			"jurisdiction":             "EU",
			"entity_type":              "corporation",
			"destination_jurisdiction": "UK",
		})
	printDecision(result5, "EU to UK cross-border")

	fmt.Println("\n🚫 Scenario 6: Cross-Border Data Transfer (EU to US - blocked)")
	fmt.Println("─────────────────────────────────────────────")
	result6, _ := integration.EnforceJurisdiction(ctx,
		"compliance@eubank.com",
		"customer-data",
		"personal_data_transfer",
		map[string]interface{}{
			"jurisdiction":             "EU",
			"entity_type":              "corporation",
			"destination_jurisdiction": "US",
		})
	printDecision(result6, "EU to US cross-border")

	fmt.Println("\n🛡️  Scenario 7: EU Data Residency (personal data leaving EU)")
	fmt.Println("─────────────────────────────────────────────")
	result7, _ := integration.EnforceJurisdiction(ctx,
		"admin@eubank.com",
		"data-export",
		"data_export",
		map[string]interface{}{
			"jurisdiction":             "EU",
			"entity_type":              "corporation",
			"destination_jurisdiction": "US",
			"data_type":                "personal_data",
		})
	printDecision(result7, "EU data residency violation")

	fmt.Println("\n🚫 Scenario 8: EU Blocked Action (unrestricted data export)")
	fmt.Println("─────────────────────────────────────────────")
	result8, _ := integration.EnforceJurisdiction(ctx,
		"admin@eubank.com",
		"data-system",
		"unrestricted_data_export",
		map[string]interface{}{
			"jurisdiction": "EU",
			"entity_type":  "corporation",
		})
	printDecision(result8, "EU blocked action")

	fmt.Println("\n🇺🇸 Scenario 9: US CCPA Data Processing (no opt-out)")
	fmt.Println("─────────────────────────────────────────────")
	result9, _ := integration.EnforceJurisdiction(ctx,
		"analyst@usbank.com",
		"customer-data",
		"ccpa_data_processing",
		map[string]interface{}{
			"jurisdiction": "US",
			"entity_type":  "corporation",
			"ccpa_opt_out": false,
		})
	printDecision(result9, "US CCPA no opt-out")

	fmt.Println("\n🚫 Scenario 10: US CCPA Data Processing (with opt-out)")
	fmt.Println("─────────────────────────────────────────────")
	result10, _ := integration.EnforceJurisdiction(ctx,
		"analyst@usbank.com",
		"customer-data",
		"ccpa_data_processing",
		map[string]interface{}{
			"jurisdiction": "US",
			"entity_type":  "corporation",
			"ccpa_opt_out": true,
		})
	printDecision(result10, "US CCPA with opt-out")

	// Wait for audit callbacks
	time.Sleep(200 * time.Millisecond)

	// Display comprehensive metrics
	fmt.Println("\n📊 Enforcement Metrics Summary")
	fmt.Println("═════════════════════════════════════════════")
	metrics := integration.GetMetrics()
	fmt.Printf("Total Enforcements:      %d\n", metrics.TotalEnforcements)
	if metrics.TotalEnforcements > 0 {
		fmt.Printf("Allowed:                 %d (%.1f%%)\n",
			metrics.AllowedCount,
			float64(metrics.AllowedCount)/float64(metrics.TotalEnforcements)*100)
		fmt.Printf("Denied:                  %d (%.1f%%)\n",
			metrics.DeniedCount,
			float64(metrics.DeniedCount)/float64(metrics.TotalEnforcements)*100)
	} else {
		fmt.Printf("Allowed:                 %d (0.0%%)\n", metrics.AllowedCount)
		fmt.Printf("Denied:                  %d (0.0%%)\n", metrics.DeniedCount)
	}
	fmt.Printf("Average Latency:         %.2f ms\n", metrics.AverageLatencyMs)
	fmt.Printf("Cross-Border Attempts:   %d\n", metrics.CrossBorderAttempts)
	fmt.Printf("Cross-Border Denials:    %d\n", metrics.CrossBorderDenials)
	fmt.Printf("Data Residency Violations: %d\n", metrics.DataResidencyViolations)

	fmt.Println("\n📈 Jurisdiction Breakdown:")
	for jurisdiction, count := range metrics.JurisdictionBreakdown {
		fmt.Printf("  %s: %d enforcements\n", jurisdiction, count)
	}

	fmt.Println("\n🚨 Violation Types:")
	for violationType, count := range metrics.ViolationsByType {
		fmt.Printf("  %s: %d\n", violationType, count)
	}

	fmt.Printf("\n✅ Audit Log: %d entries recorded\n", len(auditLog))

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nJurisdiction-specific enforcement (BETA) is operational!")
	fmt.Println("The beta system successfully enforces:")
	fmt.Println("  ✓ GDPR compliance (EU)")
	fmt.Println("  ✓ CCPA compliance (US)")
	fmt.Println("  ✓ Cross-border data transfer rules")
	fmt.Println("  ✓ Data residency requirements")
	fmt.Println("  ✓ Value limits and approval levels")
	fmt.Println("  ✓ Blocked actions per jurisdiction")
	fmt.Println("  ✓ Real-time enforcement with audit trails")
	fmt.Println("\nNote: This is a BETA implementation for testing and validation.")
}

func printDecision(decision *jurisdiction.EnforcementDecision, scenario string) {
	if decision.Allowed {
		fmt.Printf("✅ ALLOWED (%s)\n", scenario)
	} else {
		fmt.Printf("❌ DENIED (%s)\n", scenario)
	}

	fmt.Printf("   Request ID: %s\n", decision.RequestID)
	fmt.Printf("   Jurisdiction: %s\n", decision.Jurisdiction)

	if len(decision.AppliedRules) > 0 {
		fmt.Printf("   Applied Rules: %v\n", decision.AppliedRules)
	}

	if len(decision.RequiredApprovals) > 0 {
		fmt.Printf("   Required Approvals: %v\n", decision.RequiredApprovals)
	}

	if len(decision.ValueLimits) > 0 {
		fmt.Printf("   Value Limits: %v\n", decision.ValueLimits)
	}

	if len(decision.Violations) > 0 {
		fmt.Printf("   ⚠️  Violations:\n")
		for _, violation := range decision.Violations {
			fmt.Printf("      - %s\n", violation)
		}
	}

	if len(decision.Warnings) > 0 {
		fmt.Printf("   ⚠️  Warnings: %v\n", decision.Warnings)
	}

	fmt.Printf("   Latency: %v\n", decision.EnforcementLatency)
}
