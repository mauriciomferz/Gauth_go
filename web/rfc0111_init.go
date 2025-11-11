package web

import (
	"fmt"
	"os"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth/mocks"
)

// InitRFC0111FromEnv initializes RFC-0111 components based on environment variables.
// This is a web-server specific helper that can create mock services.
//
// Environment variables:
//   - GAUTH_RFC0111_ENABLED: Set to "1" to enable RFC-0111 functionality
//   - GAUTH_RFC0111_USE_MOCKS: Set to "1" to use mock external services (default: 1)
//
// Returns nil if RFC-0111 is not enabled.
func InitRFC0111FromEnv() (*gauth.RFC0111Components, error) {
	// Check if RFC-0111 is enabled
	if os.Getenv("GAUTH_RFC0111_ENABLED") != "1" {
		return nil, nil
	}

	// Determine whether to use mocks (default: yes)
	useMocks := os.Getenv("GAUTH_RFC0111_USE_MOCKS") != "0"

	if !useMocks {
		return nil, fmt.Errorf("RFC-0111: real external service implementations not yet available, set GAUTH_RFC0111_USE_MOCKS=1 or unset")
	}

	// Create mock external services
	pvpClient := mocks.NewMockPowerVerificationPoint()
	pipClient := mocks.NewMockPIPClient()
	commercialRegClient := mocks.NewMockCommercialRegisterClient()

	// Initialize RFC-0111 with mocks
	components, err := gauth.InitRFC0111WithComponents(
		pvpClient,
		pipClient,
		commercialRegClient,
	)
	if err != nil {
		return nil, fmt.Errorf("RFC-0111 initialization failed: %w", err)
	}

	return components, nil
}
