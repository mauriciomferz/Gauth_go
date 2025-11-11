package web

import (
	"fmt"
	"os"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth/mocks"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// InitRFC0111FromEnv initializes RFC-0111 components based on environment variables.
// This is a web-server specific helper that can create mock services and configure persistence.
//
// Environment variables:
//   - GAUTH_RFC0111_ENABLED: Set to "1" to enable RFC-0111 functionality
//   - GAUTH_RFC0111_USE_MOCKS: Set to "1" to use mock external services (default: 1)
//   - GAUTH_TOKEN_STORE: "postgres" or "memory" (default: memory)
//   - DB_HOST: PostgreSQL host (default: localhost)
//   - DB_PORT: PostgreSQL port (default: 5432)
//   - DB_NAME: PostgreSQL database name (default: gauth)
//   - DB_USER: PostgreSQL user (default: gauth)
//   - DB_PASSWORD: PostgreSQL password (default: gauth_password)
//   - DB_SSLMODE: PostgreSQL SSL mode (default: disable)
//
// Returns nil if RFC-0111 is not enabled.
// Returns an ExtendedTokenStore configured based on GAUTH_TOKEN_STORE.
func InitRFC0111FromEnv() (*gauth.RFC0111Components, gauth.ExtendedTokenStore, error) {
	// Check if RFC-0111 is enabled
	if os.Getenv("GAUTH_RFC0111_ENABLED") != "1" {
		return nil, nil, nil
	}

	// Determine whether to use mocks (default: yes)
	useMocks := os.Getenv("GAUTH_RFC0111_USE_MOCKS") != "0"

	if !useMocks {
		return nil, nil, fmt.Errorf("RFC-0111: real external service implementations not yet available, set GAUTH_RFC0111_USE_MOCKS=1 or unset")
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
		return nil, nil, fmt.Errorf("RFC-0111 initialization failed: %w", err)
	}

	// Initialize token store based on GAUTH_TOKEN_STORE environment variable
	tokenStoreType := os.Getenv("GAUTH_TOKEN_STORE")
	if tokenStoreType == "" {
		tokenStoreType = "memory" // default
	}

	var tokenStore gauth.ExtendedTokenStore
	switch tokenStoreType {
	case "postgres":
		// Build PostgreSQL DSN from environment variables
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "gauth"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "gauth"
		}
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			password = "gauth_password"
		}
		sslmode := os.Getenv("DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}

		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)

		// Create PostgreSQL token store
		pgStore, err := gauth.NewPostgresExtendedTokenStoreFromDSN(dsn)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize PostgreSQL token store: %w", err)
		}
		tokenStore = pgStore
		fmt.Fprintf(os.Stderr, "[RFC-0111] Using PostgreSQL token store (host=%s, db=%s)\n", host, dbname)

	case "memory":
		tokenStore = gauth.NewMemoryExtendedTokenStore()
		fmt.Fprintf(os.Stderr, "[RFC-0111] Using in-memory token store\n")

	default:
		return nil, nil, fmt.Errorf("unknown token store type: %s (supported: memory, postgres)", tokenStoreType)
	}

	return components, tokenStore, nil
}
