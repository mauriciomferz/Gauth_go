package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

// TestCompleteAuthorizationFlow tests basic RFC compliance
func TestCompleteAuthorizationFlow(t *testing.T) {
	// Setup test environment
	ctx := context.Background()
	framework := setupTestFramework(t)

	// Create a simple GAuth request for testing
	request := auth.GAuthRequest{
		ClientID:     "test-client",
		ResponseType: "code",
		Scope:        []string{"test_scope"},
		RedirectURI:  "https://test.example.com/callback",
		PowerType:    "test_power",
		PrincipalID:  "test_principal",
		AIAgentID:    "test_agent",
		Jurisdiction: "US",
		LegalBasis:   "test_basis",
		PoADefinition: auth.PoADefinition{
			Principal: auth.Principal{
				Identity: "test_principal",
				Type:     auth.PrincipalTypeIndividual,
			},
		},
	}

	// Test authorization
	response, err := framework.AuthorizeGAuth(ctx, request)
	require.NoError(t, err, "GAuth authorization should succeed")
	require.NotEmpty(t, response.AuthorizationCode, "Authorization code should not be empty")
}

// Helper functions for test setup
func setupTestFramework(_ *testing.T) *auth.RFCCompliantService {
	service, err := auth.NewRFCCompliantService("test-issuer", "test-audience")
	if err != nil {
		panic(err)
	}
	return service
}
