package agentauth

import (
	"fmt"
	"os"

	"github.com/mauriciomferz/AgentAuth/pkg/pdp"
)

// AAP001Config holds the configuration for AAP001 components
type AAP001Config struct {
	// Enabled controls whether AAP001 functionality is active
	Enabled bool

	// UseMocks controls whether to use mock external services
	// If false, real implementations must be provided
	UseMocks bool

	// Component interfaces (can be real or mock implementations)
	PVPClient           PowerVerificationPoint
	PIPClient           PIPClient
	CommercialRegClient CommercialRegisterClient

	// Storage and validators
	SubscriptionStore   SubscriptionStore
	AuthChainValidator  *AuthorizationChainValidator
	FormalReqValidator  *FormalRequirementsValidator
	ComplianceValidator *ComplianceValidator

	// Flow managers
	SubscriptionManager *SubscriptionFlowManager
	ComplianceTracker   ComplianceTracker
}

// AAP001Components holds initialized AAP001 components
type AAP001Components struct {
	SubscriptionStore   SubscriptionStore
	SubscriptionManager *SubscriptionFlowManager
	ComplianceTracker   ComplianceTracker
	ComplianceValidator *ComplianceValidator
	AuthChainValidator  *AuthorizationChainValidator
	FormalReqValidator  *FormalRequirementsValidator
	PVPClient           PowerVerificationPoint
	PIPClient           PIPClient
	CommercialRegClient CommercialRegisterClient
}

// InitAAP001FromEnv initializes AAP001 components based on environment variables.
// This is a convenience function for web server integration.
// Deprecated: Use internal/config.Load() and explicit initialization instead.
// Environment variables:
//   - AGENTAUTH_AAP001_ENABLED: Set to "1" to enable AAP001 functionality
//   - AGENTAUTH_AAP001_USE_MOCKS: Set to "1" to use mock external services (default)
//
// When enabled with mocks, this function creates:
//   - Mock external service clients (PVP, PIP, Commercial Register)
//   - In-memory subscription storage
//   - All required validators
//   - Subscription flow manager
//   - Compliance tracker
func InitAAP001FromEnv() (*AAP001Components, error) {
	// Check if AAP001 is enabled
	if os.Getenv("AGENTAUTH_AAP001_ENABLED") != "1" {
		return nil, nil
	}

	// Determine whether to use mocks (default: yes)
	useMocks := true
	if os.Getenv("AGENTAUTH_AAP001_USE_MOCKS") == "0" {
		useMocks = false
	}

	if !useMocks {
		return nil, fmt.Errorf("AAP001: real external service implementations not yet available, set AGENTAUTH_AAP001_USE_MOCKS=1")
	}

	return InitAAP001WithMocks()
}

// InitAAP001WithMocks initializes AAP001 components using mock external services.
// This is suitable for development, testing, and demonstrations.
func InitAAP001WithMocks() (*AAP001Components, error) {
	// Import is in this file to avoid circular dependency with mocks package
	// Instead, we'll use interface types and let callers provide mock implementations
	// For now, this returns an error requiring explicit setup
	return nil, fmt.Errorf("AAP001: use InitAAP001WithComponents to provide mock implementations")
}

// createDefaultPDPEngine creates a PDP engine with default policies
// This provides basic policy evaluation capability for AAP001 compliance
func createDefaultPDPEngine() pdp.Engine {
	// Create engine with deny-overrides combining strategy
	// This means any deny decision will override allow decisions
	strategy := pdp.DenyOverridesStrategy{}
	engine := pdp.NewInMemoryEngine(strategy)

	// Enable optional features
	engine.WithObligationFailureDenies(true)

	// Add default policies for AAP001 compliance
	// Policy 1: Allow authenticated requests with valid authorization chains
	engine.AddPolicy(pdp.Policy{
		ID:      "AAP001-allow-valid-chain",
		Subjects: []string{"*"}, // Apply to all subjects
		Rules: []pdp.Rule{
			{
				ID:        "allow-read-write",
				Actions:   []string{"read", "write", "execute", "authorize", "access"},
				Resources: []string{"*"}, // Apply to all resources
				Effect:    "allow",
			},
		},
		Metadata: map[string]string{
			"description": "Allow requests with valid authorization chains",
			"aap":         "AAP001",
		},
	})

	// Policy 2: Default deny for unknown actions
	engine.AddPolicy(pdp.Policy{
		ID:      "AAP001-default-deny",
		Subjects: []string{"*"},
		Rules: []pdp.Rule{
			{
				ID:        "deny-unknown",
				Actions:   []string{"delete", "admin"},
				Resources: []string{"*"},
				Effect:    "deny",
			},
		},
		Metadata: map[string]string{
			"description": "Deny dangerous actions by default",
			"aap":         "AAP001",
		},
	})

	return engine
}

// InitAAP001WithComponents initializes AAP001 using provided components.
// This gives full control over which implementations to use (mock or real).
func InitAAP001WithComponents(
	pvpClient PowerVerificationPoint,
	pipClient PIPClient,
	commercialRegClient CommercialRegisterClient,
) (*AAP001Components, error) {

	if pvpClient == nil {
		return nil, fmt.Errorf("AAP001: pvpClient is required")
	}
	if pipClient == nil {
		return nil, fmt.Errorf("AAP001: pipClient is required")
	}
	if commercialRegClient == nil {
		return nil, fmt.Errorf("AAP001: commercialRegClient is required")
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

	// Create PDP engine with default policy store
	pdpEngine := createDefaultPDPEngine()
	pdpBridge := NewPDPBridge(pdpEngine)

	complianceValidator := NewComplianceValidator(
		authChainValidator,
		pipClient,
		pdpBridge, // PDPClient with actual PDP engine
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

	return &AAP001Components{
		SubscriptionStore:   subscriptionStore,
		SubscriptionManager: subscriptionManager,
		ComplianceTracker:   complianceTracker,
		ComplianceValidator: complianceValidator,
		AuthChainValidator:  authChainValidator,
		FormalReqValidator:  formalReqValidator,
		PVPClient:           pvpClient,
		PIPClient:           pipClient,
		CommercialRegClient: commercialRegClient,
	}, nil
}
