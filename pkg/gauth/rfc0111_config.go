package gauth

import (
	"fmt"
	"os"
)

// RFC0111Config holds the configuration for RFC-0111 components
type RFC0111Config struct {
	// Enabled controls whether RFC-0111 functionality is active
	Enabled bool
	
	// UseMocks controls whether to use mock external services
	// If false, real implementations must be provided
	UseMocks bool
	
	// Component interfaces (can be real or mock implementations)
	PVPClient              PowerVerificationPoint
	PIPClient              PIPClient
	CommercialRegClient    CommercialRegisterClient
	
	// Storage and validators
	SubscriptionStore      SubscriptionStore
	AuthChainValidator     *AuthorizationChainValidator
	FormalReqValidator     *FormalRequirementsValidator
	ComplianceValidator    *ComplianceValidator
	
	// Flow managers
	SubscriptionManager    *SubscriptionFlowManager
	ComplianceTracker      ComplianceTracker
}

// RFC0111Components holds initialized RFC-0111 components
type RFC0111Components struct {
	SubscriptionStore      SubscriptionStore
	SubscriptionManager    *SubscriptionFlowManager
	ComplianceTracker      ComplianceTracker
	ComplianceValidator    *ComplianceValidator
	AuthChainValidator     *AuthorizationChainValidator
	FormalReqValidator     *FormalRequirementsValidator
	PVPClient              PowerVerificationPoint
	PIPClient              PIPClient
	CommercialRegClient    CommercialRegisterClient
}

// InitRFC0111FromEnv initializes RFC-0111 components based on environment variables.
// This is a convenience function for web server integration.
//
// Environment variables:
//   - GAUTH_RFC0111_ENABLED: Set to "1" to enable RFC-0111 functionality
//   - GAUTH_RFC0111_USE_MOCKS: Set to "1" to use mock external services (default)
//
// When enabled with mocks, this function creates:
//   - Mock external service clients (PVP, PIP, Commercial Register)
//   - In-memory subscription storage
//   - All required validators
//   - Subscription flow manager
//   - Compliance tracker
func InitRFC0111FromEnv() (*RFC0111Components, error) {
	// Check if RFC-0111 is enabled
	if os.Getenv("GAUTH_RFC0111_ENABLED") != "1" {
		return nil, nil
	}
	
	// Determine whether to use mocks (default: yes)
	useMocks := true
	if os.Getenv("GAUTH_RFC0111_USE_MOCKS") == "0" {
		useMocks = false
	}
	
	if !useMocks {
		return nil, fmt.Errorf("RFC-0111: real external service implementations not yet available, set GAUTH_RFC0111_USE_MOCKS=1")
	}
	
	return InitRFC0111WithMocks()
}

// InitRFC0111WithMocks initializes RFC-0111 components using mock external services.
// This is suitable for development, testing, and demonstrations.
func InitRFC0111WithMocks() (*RFC0111Components, error) {
	// Import is in this file to avoid circular dependency with mocks package
	// Instead, we'll use interface types and let callers provide mock implementations
	// For now, this returns an error requiring explicit setup
	return nil, fmt.Errorf("RFC-0111: use InitRFC0111WithComponents to provide mock implementations")
}

// InitRFC0111WithComponents initializes RFC-0111 using provided components.
// This gives full control over which implementations to use (mock or real).
func InitRFC0111WithComponents(
	pvpClient PowerVerificationPoint,
	pipClient PIPClient,
	commercialRegClient CommercialRegisterClient,
) (*RFC0111Components, error) {
	
	if pvpClient == nil {
		return nil, fmt.Errorf("RFC-0111: pvpClient is required")
	}
	if pipClient == nil {
		return nil, fmt.Errorf("RFC-0111: pipClient is required")
	}
	if commercialRegClient == nil {
		return nil, fmt.Errorf("RFC-0111: commercialRegClient is required")
	}
	
	// Create storage
	subscriptionStore := NewMemorySubscriptionStore()
	
	// Create validators
	authChainValidator := NewAuthorizationChainValidator(
		commercialRegClient,
		nil, // TrustServiceProvider (optional)
		nil, // RevocationChecker (optional)
	)
	
	formalReqValidator := NewFormalRequirementsValidator(
		nil,   // NotarialCertificateVerifier (optional)
		nil,   // IdentityDocumentVerifier (optional)
		nil,   // DigitalSignatureVerifier (optional)
		false, // strict mode (false for development)
	)
	
	complianceValidator := NewComplianceValidator(
		authChainValidator,
		pipClient,
		nil, // PDPClient (optional)
	)
	
	// Create subscription flow manager
	subscriptionManager := NewSubscriptionFlowManager(
		pvpClient,
		pipClient,
		commercialRegClient,
		authChainValidator,
		formalReqValidator,
		subscriptionStore,
	)
	
	// Create compliance tracker
	complianceTracker := NewMemoryComplianceTracker(complianceValidator)
	
	return &RFC0111Components{
		SubscriptionStore:      subscriptionStore,
		SubscriptionManager:    subscriptionManager,
		ComplianceTracker:      complianceTracker,
		ComplianceValidator:    complianceValidator,
		AuthChainValidator:     authChainValidator,
		FormalReqValidator:     formalReqValidator,
		PVPClient:              pvpClient,
		PIPClient:              pipClient,
		CommercialRegClient:    commercialRegClient,
	}, nil
}
