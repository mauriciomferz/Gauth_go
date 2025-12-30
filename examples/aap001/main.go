// Package main demonstrates AAP001 integration with mock external services
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth/mocks"
)

func main() {
	fmt.Println("AAP001 Integration Example with Mock Services")
	fmt.Println("================================================")
	fmt.Println()

	// 1. Create mock external services
	fmt.Println("1. Creating mock external services...")
	pvpClient := mocks.NewMockPowerVerificationPoint()
	pipClient := mocks.NewMockPIPClient()
	commercialRegClient := mocks.NewMockCommercialRegisterClient()
	fmt.Println("   ✓ Mock services created")
	fmt.Println()

	// 2. Create storage
	fmt.Println("2. Creating subscription storage...")
	subscriptionStore := agentauth.NewMemorySubscriptionStore()
	fmt.Println("   ✓ In-memory storage initialized")
	fmt.Println()

	// 3. Create validators (with mock dependencies)
	fmt.Println("3. Creating validators...")
	// Note: Validators need their own dependencies - creating simple mocks
	authChainValidator := agentauth.NewAuthorizationChainValidator(
		commercialRegClient,
		nil, // TrustServiceProvider (not needed for basic demo)
		nil, // RevocationChecker (not needed for basic demo)
	)
	formalReqValidator := agentauth.NewFormalRequirementsValidator(
		nil,   // NotarialCertificateVerifier
		nil,   // IdentityDocumentVerifier
		nil,   // DigitalSignatureVerifier
		false, // strict mode off for demo
	)
	complianceValidator := agentauth.NewComplianceValidator(
		authChainValidator,
		pipClient,
		nil, // PDPClient (not needed for basic demo)
	)
	fmt.Println("   ✓ Validators created")
	fmt.Println()

	// 4. Create subscription manager
	fmt.Println("4. Creating subscription flow manager...")
	subscriptionManager := agentauth.NewSubscriptionFlowManager(
		pvpClient,
		pipClient,
		commercialRegClient,
		authChainValidator,
		formalReqValidator,
		subscriptionStore,
	)
	fmt.Println("   ✓ Subscription manager ready")
	fmt.Println()

	// 5. Create compliance tracker
	fmt.Println("5. Creating compliance tracker...")
	complianceTracker := agentauth.NewMemoryComplianceTracker(complianceValidator)
	fmt.Println("   ✓ Compliance tracker initialized")
	fmt.Println()

	// 6. Initiate a subscription (Step I)
	fmt.Println("6. Initiating subscription (Step I)...")
	ctx := context.Background()
	subscription, err := subscriptionManager.InitiateSubscription(ctx)
	if err != nil {
		log.Fatalf("Failed to initiate subscription: %v", err)
	}
	fmt.Printf("   ✓ Subscription created: %s\n", subscription.ID)
	fmt.Printf("   ✓ Status: %s\n\n", subscription.Status)

	// 7. Test identity verification with mock PVP
	fmt.Println("7. Testing identity verification with mock PVP...")
	identityProof, err := pvpClient.VerifyIdentityProof(ctx, &agentauth.IdentityProofRequest{
		SubjectID:     "test_subject_001",
		IdentityType:  "natural_person",
		ProofMethod:   "eIDAS",
		RequiredLevel: "high",
	})
	if err != nil {
		log.Fatalf("Identity verification failed: %v", err)
	}
	fmt.Printf("   ✓ Identity verified: %s\n", identityProof.Identity)
	fmt.Printf("   ✓ Trust level: %s\n", identityProof.TrustLevel)
	fmt.Printf("   ✓ PVP call count: %d\n\n", pvpClient.CallCount)

	// 8. Test client info retrieval with mock PIP
	fmt.Println("8. Testing client info retrieval with mock PIP...")
	clientInfo, err := pipClient.GetClientInfo(ctx, "test_client_001")
	if err != nil {
		log.Fatalf("Failed to get client info: %v", err)
	}
	fmt.Printf("   ✓ Client name: %s\n", clientInfo.ClientName)
	fmt.Printf("   ✓ Client active: %v\n", clientInfo.Active)
	fmt.Printf("   ✓ PIP call count: %d\n\n", pipClient.GetClientInfoCallCount)

	// 9. Test company verification with mock commercial register
	fmt.Println("9. Testing company verification with mock commercial register...")
	companyInfo, err := commercialRegClient.VerifyCompany(ctx, "DE", "HRB12345")
	if err != nil {
		log.Fatalf("Company verification failed: %v", err)
	}
	fmt.Printf("   ✓ Company: %s\n", companyInfo.LegalName)
	fmt.Printf("   ✓ Jurisdiction: %s\n", companyInfo.Jurisdiction)
	fmt.Printf("   ✓ Status: %s\n", companyInfo.Status)
	fmt.Printf("   ✓ Commercial register call count: %d\n\n", commercialRegClient.VerifyCompanyCallCount)

	// 10. Get storage statistics
	fmt.Println("10. Checking storage statistics...")
	stats := subscriptionStore.GetStats()
	fmt.Printf("   ✓ Total subscriptions: %d\n", stats["total"])
	fmt.Println("   ✓ Storage initialized and operational")
	fmt.Println()

	// 11. Get compliance tracker statistics
	fmt.Println("11. Checking compliance tracker statistics...")
	trackerStats := complianceTracker.GetStats()
	fmt.Printf("   ✓ Compliance tracker initialized\n")
	fmt.Printf("   ✓ Statistics available: %d metrics\n\n", len(trackerStats))

	// Success!
	fmt.Println("================================================")
	fmt.Println("✓ All AAP001 components working successfully!")
	fmt.Println("================================================")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("- Execute Steps II-VIII for complete subscription")
	fmt.Println("- Execute authorization flow (Steps a-i)")
	fmt.Println("- Integrate with REST API")
	fmt.Println("- Replace mocks with real external service implementations")
}
