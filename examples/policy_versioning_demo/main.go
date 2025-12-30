package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/policy"
	pkgpolicy "github.com/mauriciomferz/AgentAuth/pkg/policy"
)

func main() {
	fmt.Println("=== Policy Versioning & Rollback Demo (BETA) ===")
	fmt.Println("This is a BETA implementation for testing and validation")
	fmt.Println()

	// Initialize registry and version manager
	registry := pkgpolicy.NewRegistry()
	versionManager := policy.NewPolicyVersionManager(registry)
	ctx := context.Background()

	// Track audit events
	var auditEvents []policy.VersionAuditEvent
	versionManager.SetAuditCallback(func(event policy.VersionAuditEvent) {
		auditEvents = append(auditEvents, event)
		fmt.Printf("📋 Audit: %s | Version: %d | Success: %v\n", event.EventType, event.Version, event.Success)
		if event.Error != "" {
			fmt.Printf("   Error: %s\n", event.Error)
		}
	})

	// Scenario 1: Create initial version (1.0.0)
	fmt.Println("\n--- Scenario 1: Create Baseline Version 1.0.0 ---")
	bundle1 := pkgpolicy.Bundle{
		ID: "policy-bundle-v1",
		Policies: []pkgpolicy.Policy{
			{
				ID:       "read-policy",
				Subjects: []string{"alice", "bob"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"/documents/*"},
						Effect:    pkgpolicy.Allow,
					},
				},
			},
		},
	}

	metadata1 := policy.PolicyVersionMetadata{
		SemanticVersion: policy.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Name:            "Initial Release",
		Description:     "Baseline policy with read-only access",
		Author:          "security-team",
		RollbackAllowed: true,
		Tags:            []string{"baseline", "beta"},
	}

	v1, err := versionManager.CreateVersion(ctx, bundle1, metadata1)
	if err != nil {
		panic(fmt.Sprintf("Failed to create v1: %v", err))
	}
	fmt.Printf("✅ Created: %s (v%d) - Hash: %s\n", v1.Name, v1.BundleVersion, v1.Hash[:8])
	fmt.Printf("   Semantic Version: %s\n", v1.SemanticVersion.String())
	fmt.Printf("   Auto-activated: %v\n", v1.ActivatedAt != nil)

	// Scenario 2: Create backward-compatible version (1.1.0)
	fmt.Println("\n--- Scenario 2: Create Backward-Compatible Version 1.1.0 ---")
	bundle2 := pkgpolicy.Bundle{
		ID: "policy-bundle-v2",
		Policies: []pkgpolicy.Policy{
			{
				ID:       "read-policy",
				Subjects: []string{"alice", "bob"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"/documents/*"},
						Effect:    pkgpolicy.Allow,
					},
				},
			},
			{
				ID:       "write-policy",
				Subjects: []string{"alice"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"write"},
						Resources: []string{"/documents/*"},
						Effect:    pkgpolicy.Allow,
					},
				},
			},
		},
	}

	metadata2 := policy.PolicyVersionMetadata{
		SemanticVersion: policy.SemanticVersion{Major: 1, Minor: 1, Patch: 0},
		Name:            "Add Write Permissions",
		Description:     "Added write policy for alice - backward compatible",
		Author:          "security-team",
		RollbackAllowed: true,
		Tags:            []string{"enhancement", "beta"},
	}

	v2, err := versionManager.CreateVersion(ctx, bundle2, metadata2)
	if err != nil {
		panic(fmt.Sprintf("Failed to create v2: %v", err))
	}
	fmt.Printf("✅ Created: %s (v%d) - Hash: %s\n", v2.Name, v2.BundleVersion, v2.Hash[:8])
	fmt.Printf("   Semantic Version: %s\n", v2.SemanticVersion.String())
	fmt.Printf("   Backward Compatible: %v\n", v2.BackwardCompatible)
	if v2.ImpactAnalysis != nil {
		fmt.Printf("   Impact: +%d policies, ~%d modified, -%d removed | Risk: %s\n",
			v2.ImpactAnalysis.PoliciesAdded,
			v2.ImpactAnalysis.PoliciesModified,
			v2.ImpactAnalysis.PoliciesRemoved,
			v2.ImpactAnalysis.RiskLevel)
	}

	// Scenario 3: Activate Version 2
	fmt.Println("\n--- Scenario 3: Activate Version 1.1.0 ---")
	err = versionManager.ActivateVersion(ctx, 2, "admin-user")
	if err != nil {
		panic(fmt.Sprintf("Failed to activate v2: %v", err))
	}
	fmt.Println("✅ Activation successful")

	// Scenario 4: Create breaking change version (2.0.0)
	fmt.Println("\n--- Scenario 4: Create Breaking Change Version 2.0.0 ---")
	bundle3 := pkgpolicy.Bundle{
		ID: "policy-bundle-v3",
		Policies: []pkgpolicy.Policy{
			{
				ID:       "read-policy-v2",
				Subjects: []string{"alice"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"read", "write", "delete"},
						Resources: []string{"/documents/*"},
						Effect:    pkgpolicy.Allow,
					},
				},
			},
		},
	}

	metadata3 := policy.PolicyVersionMetadata{
		SemanticVersion:   policy.SemanticVersion{Major: 2, Minor: 0, Patch: 0},
		Name:              "Consolidated Permissions",
		Description:       "Major: Removed bob, consolidated all permissions for alice",
		Author:            "security-team",
		RequiredApprovals: []string{"security-lead", "compliance-officer"},
		RollbackAllowed:   true,
		Tags:              []string{"breaking-change", "beta"},
	}

	v3, err := versionManager.CreateVersion(ctx, bundle3, metadata3)
	if err != nil {
		panic(fmt.Sprintf("Failed to create v3: %v", err))
	}
	fmt.Printf("✅ Created: %s (v%d) - Hash: %s\n", v3.Name, v3.BundleVersion, v3.Hash[:8])
	fmt.Printf("   Semantic Version: %s\n", v3.SemanticVersion.String())
	fmt.Printf("   Backward Compatible: %v\n", v3.BackwardCompatible)
	if len(v3.ValidationErrors) > 0 {
		fmt.Printf("   ⚠️  Validation Errors: %v\n", v3.ValidationErrors)
	}
	if v3.ImpactAnalysis != nil {
		fmt.Printf("   Impact: +%d policies, -%d removed | Risk: %s\n",
			v3.ImpactAnalysis.PoliciesAdded,
			v3.ImpactAnalysis.PoliciesRemoved,
			v3.ImpactAnalysis.RiskLevel)
	}
	fmt.Printf("   Required Approvals: %v\n", v3.RequiredApprovals)

	// Scenario 5: Try to activate without approvals (should fail)
	fmt.Println("\n--- Scenario 5: Attempt Activation Without Approvals ---")
	err = versionManager.ActivateVersion(ctx, 3, "admin-user")
	if err != nil {
		fmt.Printf("❌ Expected Failure: %v\n", err)
	} else {
		fmt.Println("⚠️  Unexpected: Activation succeeded without approvals")
	}

	// Scenario 6: Approve the version
	fmt.Println("\n--- Scenario 6: Obtain Required Approvals ---")
	for _, approver := range v3.RequiredApprovals {
		err = versionManager.ApproveVersion(3, approver)
		if err != nil {
			panic(fmt.Sprintf("Failed to get approval from %s: %v", approver, err))
		}
		fmt.Printf("✅ Approval obtained: %s\n", approver)
	}

	// Scenario 7: Activate with approvals
	fmt.Println("\n--- Scenario 7: Activate Version 2.0.0 After Approvals ---")
	err = versionManager.ActivateVersion(ctx, 3, "admin-user")
	if err != nil {
		panic(fmt.Sprintf("Failed to activate v3: %v", err))
	}
	fmt.Println("✅ Activation successful")

	// Scenario 8: Deprecate old version
	fmt.Println("\n--- Scenario 8: Deprecate Version 1.0.0 ---")
	sunsetDate := time.Now().AddDate(0, 3, 0) // 3 months from now
	err = versionManager.DeprecateVersion(ctx, 1, "Superseded by 2.0.0", &sunsetDate, "compliance-team")
	if err != nil {
		panic(fmt.Sprintf("Failed to deprecate v1: %v", err))
	}
	fmt.Printf("✅ Version 1 deprecated (sunset: %s)\n", sunsetDate.Format("2006-01-02"))

	// Scenario 9: Try rollback to Version 2 (will fail - major version boundary)
	fmt.Println("\n--- Scenario 9: Attempt Rollback Across Major Version (Should Fail) ---")
	err = versionManager.RollbackVersion(ctx, 2, "admin-user", "Production incident - reverting breaking changes")
	if err != nil {
		fmt.Printf("❌ Expected Safety Failure: %v\n", err)
		fmt.Println("   Rollback blocked: Cannot rollback across major version boundaries (2.0.0 -> 1.1.0)")
		fmt.Println("   This is a security feature to prevent unsafe state transitions")
	} else {
		fmt.Println("⚠️  Unexpected: Rollback succeeded across major version boundary")
	}

	// Scenario 10: Create a patch version 2.0.1 and rollback to 2.0.0 (safe)
	fmt.Println("\n--- Scenario 10: Create Patch Version 2.0.1 and Safe Rollback ---")
	time.Sleep(100 * time.Millisecond)

	bundle4 := pkgpolicy.Bundle{
		ID: "policy-bundle-v4",
		Policies: []pkgpolicy.Policy{
			{
				ID:       "read-policy-v2",
				Subjects: []string{"alice"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"read", "write", "delete"},
						Resources: []string{"/documents/*", "/reports/*"}, // Added /reports/*
						Effect:    pkgpolicy.Allow,
					},
				},
			},
		},
	}

	metadata4 := policy.PolicyVersionMetadata{
		SemanticVersion: policy.SemanticVersion{Major: 2, Minor: 0, Patch: 1},
		Name:            "Resource Expansion",
		Description:     "Patch: Added /reports/ to allowed resources",
		Author:          "security-team",
		RollbackAllowed: true,
		Tags:            []string{"patch", "beta"},
	}

	v4, err := versionManager.CreateVersion(ctx, bundle4, metadata4)
	if err != nil {
		panic(fmt.Sprintf("Failed to create v4: %v", err))
	}
	fmt.Printf("✅ Created: %s (v%d) - Hash: %s\n", v4.Name, v4.BundleVersion, v4.Hash[:8])
	fmt.Printf("   Semantic Version: %s\n", v4.SemanticVersion.String())

	// Activate v4
	err = versionManager.ActivateVersion(ctx, 4, "admin-user")
	if err != nil {
		panic(fmt.Sprintf("Failed to activate v4: %v", err))
	}
	fmt.Printf("✅ Activated Version 4 (2.0.1)\n")

	// Now rollback to v3 (2.0.0) - safe within same major version
	fmt.Println("\n--- Scenario 11: Safe Rollback Within Major Version (2.0.1 -> 2.0.0) ---")
	err = versionManager.RollbackVersion(ctx, 3, "admin-user", "Rollback patch changes")
	if err != nil {
		panic(fmt.Sprintf("Failed to rollback: %v", err))
	}
	fmt.Println("✅ Rollback successful (within major version boundary)")
	fmt.Printf("   Active Version: %d\n", versionManager.GetActiveVersion())

	// Summary
	fmt.Println("\n=== Summary (BETA Implementation) ===")
	fmt.Printf("Total Versions Created: %d\n", len(versionManager.ListVersions()))
	fmt.Printf("Active Version: %d\n", versionManager.GetActiveVersion())
	fmt.Printf("Total Audit Events: %d\n", len(auditEvents))

	// Export metadata
	metadata, err := versionManager.ExportMetadata()
	if err != nil {
		panic(fmt.Sprintf("Failed to export metadata: %v", err))
	}
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	fmt.Printf("\n📦 Exported Metadata:\n%s\n", metadataJSON)

	fmt.Println("\n✅ Demo completed successfully")
	fmt.Println("⚠️  Remember: This is a BETA implementation for testing purposes")
}
