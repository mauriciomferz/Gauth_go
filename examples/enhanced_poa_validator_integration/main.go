package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	cr "github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001"
)

// EnhancedPoAValidatorExample demonstrates real Service integration with enhanced PoA validator
// showing warning collection, daily limits, and metrics throughout the delegation lifecycle
func main() {
	ctx := context.Background()

	fmt.Println("🚀 Enhanced PoA Validator Service Integration Example")
	fmt.Println("=====================================================")

	// 1. Set up enhanced validator components
	fmt.Println("\n📋 Step 1: Setting up enhanced validation components...")

	// Create temporary directory for daily limits persistence
	tempDir := "/tmp/enhanced_poa_validator_service_example"
	dbPath := filepath.Join(tempDir, "daily_limits.json")

	// Initialize daily limit store with persistent storage
	store, err := agentauth_aap_001.NewBoltDailyLimitStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to create daily limit store: %v", err)
	}
	fmt.Printf("✅ Daily limit store initialized: %s\n", dbPath)

	// Initialize conditional expression engine
	engine := agentauth_aap_001.NewSimpleConditionalEngine()
	fmt.Println("✅ Conditional expression engine initialized")

	// Initialize metrics recorder
	metrics := agentauth_aap_001.NewInMemoryValidationMetrics()
	fmt.Println("✅ Validation metrics recorder initialized")

	// Create enhanced validator with all components
	enhancedValidator := agentauth_aap_001.NewEnhancedPoAValidator(
		agentauth_aap_001.WithDailyLimitStore(store),
		agentauth_aap_001.WithConditionalEngine(engine),
		agentauth_aap_001.WithMetricsRecorder(metrics),
	)
	fmt.Println("✅ Enhanced PoA validator created")

	// 2. Create AAP001 service with enhanced validator integration
	fmt.Println("\n🔧 Step 2: Creating AAP001 service with enhanced validator...")

	// Create key provider for signatures
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		log.Fatalf("Failed to create key provider: %v", err)
	}

	// Create audit logger
	auditLogger := audit.NewMemoryLogger(nil)

	// Create authorizer with policies
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "allow-alice",
		Subject:  "alice@company.com",
		Resource: "*",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})
	authorizer.AddPolicy(authz.Policy{
		ID:       "allow-corporate",
		Subject:  "corporate@company.com",
		Resource: "*",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})

	// Create service with enhanced validator integrated
	svc := agentauth_aap_001.NewService(auditLogger, authorizer,
		agentauth_aap_001.WithEnhancedValidator(enhancedValidator),
		agentauth_aap_001.WithSignerProvider(kp.ActiveSigner),
	)
	fmt.Println("✅ AAP001 service created with enhanced validation")

	// 3. Demonstrate delegation creation with warning collection
	fmt.Println("\n🧪 Step 3: Creating delegations with enhanced validation...")

	// Scenario 1: High-value delegation that triggers warning
	fmt.Println("\n💰 Scenario 1: High-value financial delegation...")
	req1 := agentauth_aap_001.DelegationRequest{
		Grantor:  "alice@company.com",
		Grantee:  "bob@company.com",
		Scope:    []string{"transaction:withdraw"},
		Duration: 24 * time.Hour,
		Restrictions: map[string]string{
			"currency":   "USD",
			"max_amount": "1500000", // High amount triggers warning
		},
	}

	resp1, err := svc.CreateDelegationCtx(ctx, req1)
	if err != nil {
		log.Fatalf("CreateDelegationCtx failed: %v", err)
	}

	fmt.Printf("   ✅ Delegation created: %s\n", resp1.POA.ID)
	fmt.Printf("   📊 Warnings collected: %d\n", len(resp1.Warnings))
	for _, warning := range resp1.Warnings {
		fmt.Printf("      - %s: %s [%s]\n", warning.Code, warning.Message, warning.Severity)
	}

	// Scenario 2: Delegation with daily limits
	fmt.Println("\n� Scenario 2: Delegation with daily limit tracking...")
	req2 := agentauth_aap_001.DelegationRequest{
		Grantor:  "corporate@company.com",
		Grantee:  "agent@company.com",
		Scope:    []string{"transaction:payment"},
		Duration: 7 * 24 * time.Hour,
		Restrictions: map[string]string{
			"currency":         "USD",
			"max_amount":       "5000",
			"max_daily_amount": "10000",
		},
	}

	resp2, err := svc.CreateDelegationCtx(ctx, req2)
	if err != nil {
		log.Fatalf("CreateDelegationCtx failed: %v", err)
	}

	fmt.Printf("   ✅ Delegation created: %s\n", resp2.POA.ID)
	fmt.Printf("   📊 Warnings: %d\n", len(resp2.Warnings))

	// Simulate daily usage to trigger approaching limit warning
	today := time.Now().Format("2006-01-02")
	if err := store.IncrementDailyUsage(resp2.POA.ID, today, 8500); err != nil {
		log.Fatalf("Failed to increment daily usage: %v", err)
	}
	fmt.Println("   💸 Simulated usage: $8,500 (85% of daily limit)")

	// Validate again to see approaching limit warning
	result := enhancedValidator.ValidateWithResult(ctx, &resp2.POA)
	fmt.Printf("   ⚠️  Warnings after usage: %d\n", len(result.Warnings))
	for _, warning := range result.Warnings {
		fmt.Printf("      - %s: %s\n", warning.Code, warning.Message)
	}

	// 4. Display final metrics
	fmt.Println("\n📈 Step 4: Validation metrics summary...")
	summary := metrics.GetMetricsSummary()
	fmt.Printf("   Total Validations: %d\n", summary["total_validations"])
	fmt.Printf("   Daily Limit Checks: %d\n", summary["daily_limit_checks"])

	successCounts := summary["success_counts"].(map[string]int)
	fmt.Println("   Success Counts:")
	for key, count := range successCounts {
		fmt.Printf("      - %s: %d\n", key, count)
	}

	warningCounts := summary["warning_counts"].(map[string]int)
	if len(warningCounts) > 0 {
		fmt.Println("   Warning Counts:")
		for key, count := range warningCounts {
			fmt.Printf("      - %s: %d\n", key, count)
		}
	}

	fmt.Println("\n✅ Enhanced PoA Validator Service Integration complete!")
	fmt.Println("   - Warning collection: ✓")
	fmt.Println("   - Daily limit tracking: ✓")
	fmt.Println("   - Metrics recording: ✓")
	fmt.Println("   - Service integration: ✓")
}
