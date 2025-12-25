package gauth

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// PVPFactory creates appropriate PVP implementation based on configuration
type PVPFactory struct {
	productionMode bool
}

// NewPVPFactory creates a new PVP factory
func NewPVPFactory(productionMode bool) *PVPFactory {
	return &PVPFactory{
		productionMode: productionMode,
	}
}

// CreatePVP creates a PowerVerificationPoint based on configuration
func (f *PVPFactory) CreatePVP() (PowerVerificationPoint, error) {
	provider := strings.ToLower(os.Getenv("GAUTH_PVP_PROVIDER"))

	// In production mode, block mock provider
	if f.productionMode && (provider == "" || provider == "mock") {
		return nil, fmt.Errorf(
			"production mode requires real identity verification provider. " +
				"Set GAUTH_PVP_PROVIDER to one of: stripe, veriff, idemia, onfido, jumio")
	}

	switch provider {
	case "stripe":
		return f.createStripePVP()
	case "veriff":
		return f.createVeriffPVP()
	case "idemia":
		return f.createIdemiaPVP()
	case "onfido":
		return f.createOnfidoPVP()
	case "jumio":
		return f.createJumioPVP()
	case "mock", "":
		if f.productionMode {
			return nil, fmt.Errorf("mock PVP not allowed in production")
		}
		// Development/testing mode: use mock
		return NewMockPVPWithWarning(), nil
	default:
		return nil, fmt.Errorf("unknown PVP provider: %s", provider)
	}
}

// createStripePVP creates Stripe Identity verification client
func (f *PVPFactory) createStripePVP() (PowerVerificationPoint, error) {
	apiKey := os.Getenv("STRIPE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("STRIPE_API_KEY not set - required for Stripe Identity provider")
	}

	return NewStripePVPClient(apiKey), nil
}

// createVeriffPVP creates Veriff verification client
func (f *PVPFactory) createVeriffPVP() (PowerVerificationPoint, error) {
	apiKey := os.Getenv("VERIFF_API_KEY")
	apiSecret := os.Getenv("VERIFF_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("VERIFF_API_KEY and VERIFF_API_SECRET required for Veriff provider")
	}

	return NewVeriffPVPClient(apiKey, apiSecret), nil
}

// createIdemiaPVP creates Idemia verification client
func (f *PVPFactory) createIdemiaPVP() (PowerVerificationPoint, error) {
	apiKey := os.Getenv("IDEMIA_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("IDEMIA_API_KEY required for Idemia provider")
	}

	return NewGenericPVPStub("Idemia", "https://www.idemia.com/identity-verification"), nil
}

// createOnfidoPVP creates Onfido verification client
func (f *PVPFactory) createOnfidoPVP() (PowerVerificationPoint, error) {
	apiToken := os.Getenv("ONFIDO_API_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("ONFIDO_API_TOKEN required for Onfido provider")
	}

	return NewGenericPVPStub("Onfido", "https://documentation.onfido.com/"), nil
}

// createJumioPVP creates Jumio verification client
func (f *PVPFactory) createJumioPVP() (PowerVerificationPoint, error) {
	apiToken := os.Getenv("JUMIO_API_TOKEN")
	apiSecret := os.Getenv("JUMIO_API_SECRET")

	if apiToken == "" || apiSecret == "" {
		return nil, fmt.Errorf("JUMIO_API_TOKEN and JUMIO_API_SECRET required for Jumio provider")
	}

	return NewGenericPVPStub("Jumio", "https://www.jumio.com/developers/"), nil
}

// MockPVPWithWarning wraps mock PVP with production warning
type MockPVPWithWarning struct {
	delegate PowerVerificationPoint
}

// NewMockPVPWithWarning creates a mock PVP that logs warnings
func NewMockPVPWithWarning() *MockPVPWithWarning {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "╔════════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stderr, "║ WARNING: USING MOCK IDENTITY VERIFICATION (PVP)                ║\n")
	fmt.Fprintf(os.Stderr, "║                                                                 ║\n")
	fmt.Fprintf(os.Stderr, "║ This is a DEVELOPMENT-ONLY mock that accepts any identity      ║\n")
	fmt.Fprintf(os.Stderr, "║ proof without real verification.                               ║\n")
	fmt.Fprintf(os.Stderr, "║                                                                 ║\n")
	fmt.Fprintf(os.Stderr, "║ NEVER USE IN PRODUCTION                                        ║\n")
	fmt.Fprintf(os.Stderr, "║                                                                 ║\n")
	fmt.Fprintf(os.Stderr, "║ Set GAUTH_PVP_PROVIDER to: stripe, veriff, idemia, onfido,    ║\n")
	fmt.Fprintf(os.Stderr, "║ or jumio for production deployments.                           ║\n")
	fmt.Fprintf(os.Stderr, "╚════════════════════════════════════════════════════════════════╝\n")
	fmt.Fprintf(os.Stderr, "\n")

	// Import from mocks package would happen here
	// For now, return inline mock
	return &MockPVPWithWarning{
		delegate: &inlineMockPVP{},
	}
}

// VerifyIdentityProof delegates to mock with warning logging
func (m *MockPVPWithWarning) VerifyIdentityProof(
	ctx context.Context,
	request *IdentityProofRequest,
) (*IdentityProofResult, error) {
	return m.delegate.VerifyIdentityProof(ctx, request)
}

// inlineMockPVP is a simple inline mock for when mocks package isn't available
type inlineMockPVP struct{}

func (m *inlineMockPVP) VerifyIdentityProof(
	ctx context.Context,
	request *IdentityProofRequest,
) (*IdentityProofResult, error) {
	// Import actual mock implementation
	// Using fallback inline version to avoid circular dependency
	return &IdentityProofResult{
		Valid:      true,
		SubjectID:  request.SubjectID,
		Identity:   fmt.Sprintf("mock_identity_%s", request.SubjectID),
		TrustLevel: request.RequiredLevel,
	}, nil
}
