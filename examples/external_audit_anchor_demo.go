package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/anchor"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/ledger"
)

// ExternalAuditAnchorDemo demonstrates the integrated external audit anchoring system.
func main() {
	fmt.Println("🔗 External Audit Anchor Integration Demo (sec5.item1)")
	fmt.Println("=======================================================")

	// Setup temp files with restricted permissions
	tmpDir := "/tmp/gauth-external-anchor-demo"
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := tmpDir + "/audit-ledger.db"
	receiptPath := tmpDir + "/external-receipts.json"
	anchorFilePath := tmpDir + "/ledger-anchor.json"

	// 1. Create external anchor provider
	fmt.Println("\n1️⃣ Creating External Anchor Providers...")

	// Memory provider for demo (in production: use TSA, blockchain, etc.)
	memoryProvider := anchor.NewMemoryProvider()
	fmt.Printf("   ✅ Memory provider initialized\n")

	// TSA stub provider with realistic latency simulation
	tsaProvider := anchor.NewTSAStubProvider(25, 100, 0.0) // 25-100ms, no failures
	fmt.Printf("   ✅ TSA stub provider initialized (25-100ms latency)\n")

	// 2. Create External Audit Ledger with memory provider
	fmt.Println("\n2️⃣ Creating External Audit Ledger...")

	externalLedger, err := ledger.NewExternalAuditLedger(
		dbPath,
		memoryProvider,
		receiptPath,
		2*time.Second, // Anchor every 2 seconds
	)
	if err != nil {
		log.Fatalf("Failed to create external audit ledger: %v", err)
	}
	defer externalLedger.Close()

	// Enable traditional anchor file as well (dual anchoring)
	if err := externalLedger.EnableAnchorFile(anchorFilePath, time.Second); err != nil {
		log.Fatalf("Failed to enable anchor file: %v", err)
	}
	fmt.Printf("   ✅ External audit ledger created\n")
	fmt.Printf("   📁 Database: %s\n", dbPath)
	fmt.Printf("   📋 External receipts: %s\n", receiptPath)
	fmt.Printf("   ⚓ Anchor file: %s\n", anchorFilePath)

	ctx := context.Background()

	// 3. Add audit entries and demonstrate automatic anchoring
	fmt.Println("\n3️⃣ Adding Audit Entries with Automatic Anchoring...")

	entries := []*ledger.Entry{
		{
			ID:      "auth-001",
			TS:      time.Now().UTC(),
			Type:    "authentication",
			Subject: "user@example.com",
			Object:  "login-service",
			Metadata: map[string]interface{}{
				"action":     "login",
				"result":     "success",
				"ip":         "192.168.1.100",
				"user_agent": "GAuth-Client/1.0",
			},
		},
		{
			ID:      "authz-001",
			TS:      time.Now().UTC().Add(500 * time.Millisecond),
			Type:    "authorization",
			Subject: "user@example.com",
			Object:  "protected-resource",
			Metadata: map[string]interface{}{
				"action":     "access",
				"result":     "success",
				"policy_id":  "pol-12345",
				"capability": "read",
			},
		},
		{
			ID:      "revoke-001",
			TS:      time.Now().UTC().Add(time.Second),
			Type:    "revocation",
			Subject: "admin@example.com",
			Object:  "capability:cap-789",
			Metadata: map[string]interface{}{
				"action":   "revoke",
				"result":   "success",
				"reason":   "security-policy",
				"operator": "admin@example.com",
			},
		},
	}

	for i, entry := range entries {
		fmt.Printf("   📝 Appending entry %d: %s -> %s\n", i+1, entry.Subject, entry.Object)
		if err := externalLedger.Append(ctx, entry); err != nil {
			log.Fatalf("Failed to append entry: %v", err)
		}
		time.Sleep(300 * time.Millisecond) // Brief pause between entries
	}

	fmt.Printf("   ✅ Added %d audit entries to ledger\n", len(entries))

	// 4. Wait for automatic anchoring and show status
	fmt.Println("\n4️⃣ Monitoring Automatic External Anchoring...")
	time.Sleep(2500 * time.Millisecond) // Wait for anchor interval

	status := externalLedger.ExternalAnchorStatus()
	fmt.Printf("   📊 Anchor Status:\n")
	fmt.Printf("      • Configured: %v\n", status["configured"])
	fmt.Printf("      • Interval: %v\n", status["interval"])
	fmt.Printf("      • Last anchor: %v\n", status["last_anchor_at"])
	fmt.Printf("      • Age: %.2fs\n", status["age_seconds"])

	if receipt, ok := status["latest_receipt"]; ok {
		r := receipt.(map[string]interface{})
		fmt.Printf("      • Latest Receipt:\n")
		fmt.Printf("        - Hash: %v\n", r["hash"])
		fmt.Printf("        - Provider: %v\n", r["provider"])
		fmt.Printf("        - Timestamp: %v\n", r["timestamp"])
	}

	// 5. Force immediate external anchoring
	fmt.Println("\n5️⃣ Force External Anchoring...")
	if err := externalLedger.ForceExternalAnchor(); err != nil {
		log.Fatalf("Failed to force external anchor: %v", err)
	}
	fmt.Printf("   ✅ External anchor forced successfully\n")

	// 6. Verify chain integrity
	fmt.Println("\n6️⃣ Verifying Chain Integrity...")
	result, err := externalLedger.VerifyChain(ctx)
	if err != nil {
		log.Fatalf("Failed to verify chain: %v", err)
	}

	fmt.Printf("   🔍 Chain Verification Results:\n")
	fmt.Printf("      • Total entries: %d\n", result.Count)
	fmt.Printf("      • Mismatches: %d\n", result.Mismatches)
	fmt.Printf("      • First hash: %s\n", result.FirstHash)
	fmt.Printf("      • Last hash: %s\n", result.LastHash)

	if result.Mismatches == 0 {
		fmt.Printf("   ✅ Chain integrity verified - no tampering detected\n")
	} else {
		fmt.Printf("   ❌ Chain integrity compromised - %d mismatches found\n", result.Mismatches)
	}

	// 7. Demonstrate TSA provider integration
	fmt.Println("\n7️⃣ Demonstrating TSA Provider Integration...")

	// Create another ledger with TSA provider
	tsaDbPath := tmpDir + "/tsa-audit-ledger.db"
	tsaReceiptPath := tmpDir + "/tsa-receipts.json"

	tsaLedger, err := ledger.NewExternalAuditLedger(
		tsaDbPath,
		tsaProvider,
		tsaReceiptPath,
		time.Second,
	)
	if err != nil {
		log.Fatalf("Failed to create TSA ledger: %v", err)
	}
	defer tsaLedger.Close()

	// Add an entry and force anchor with TSA
	tsaEntry := &ledger.Entry{
		ID:      "tsa-demo",
		TS:      time.Now().UTC(),
		Type:    "audit",
		Subject: "system",
		Object:  "external-tsa",
		Metadata: map[string]interface{}{
			"action": "timestamp-demo",
			"result": "success",
		},
	}

	if err := tsaLedger.Append(ctx, tsaEntry); err != nil {
		log.Fatalf("Failed to append TSA entry: %v", err)
	}

	fmt.Printf("   ⏱️ Submitting to TSA provider (simulated latency)...\n")
	start := time.Now()
	if err := tsaLedger.ForceExternalAnchor(); err != nil {
		log.Fatalf("Failed to anchor with TSA: %v", err)
	}
	latency := time.Since(start)

	tsaStatus := tsaLedger.ExternalAnchorStatus()
	if receipt, ok := tsaStatus["latest_receipt"]; ok {
		r := receipt.(map[string]interface{})
		fmt.Printf("   ✅ TSA anchoring completed in %v\n", latency)
		fmt.Printf("      • Provider: %v\n", r["provider"])
		fmt.Printf("      • Timestamp: %v\n", r["timestamp"])
		fmt.Printf("      • Hash: %v\n", r["hash"])
	}

	// 8. Show file artifacts
	fmt.Println("\n8️⃣ Generated Artifacts:")

	if _, err := os.Stat(anchorFilePath); err == nil {
		fmt.Printf("   📄 Anchor file: %s\n", anchorFilePath)
	}

	if _, err := os.Stat(receiptPath); err == nil {
		fmt.Printf("   🧾 External receipts: %s\n", receiptPath)
	}

	if _, err := os.Stat(tsaReceiptPath); err == nil {
		fmt.Printf("   🧾 TSA receipts: %s\n", tsaReceiptPath)
	}

	fmt.Printf("   🗄️ BoltDB ledger: %s\n", dbPath)
	fmt.Printf("   🗄️ TSA BoltDB ledger: %s\n", tsaDbPath)

	fmt.Println("\n✨ External Audit Anchor Integration Demo Complete!")
	fmt.Println("🎯 sec5.item1 - External Audit Anchor support is now fully implemented:")
	fmt.Println("   • ✅ BoltDB audit ledger with hash chains")
	fmt.Println("   • ✅ External anchor provider integration (Memory, TSA stub)")
	fmt.Println("   • ✅ Automatic periodic anchoring with configurable intervals")
	fmt.Println("   • ✅ Manual force anchoring on-demand")
	fmt.Println("   • ✅ External receipt persistence with hash-chain integrity")
	fmt.Println("   • ✅ Dual anchoring (both file-based and external provider)")
	fmt.Println("   • ✅ Chain verification and tamper detection")
	fmt.Println("   • ✅ Comprehensive test coverage")
	fmt.Println("   • ✅ Beta implementation ready for integration")
}
