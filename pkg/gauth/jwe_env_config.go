// Package gauth - JWE Environment Configuration
// Provides environment variable-based configuration for production deployments
package gauth

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// JWEConfigFromEnv creates JWE configuration from environment variables
// Supports the following environment variables:
//
// GAUTH_JWE_ENABLED - Enable/disable JWE encryption (default: true)
// GAUTH_JWE_ALGORITHM - Key encryption algorithm (default: RSA-OAEP-256)
// GAUTH_JWE_ENCRYPTION - Content encryption algorithm (default: A256GCM)
// GAUTH_JWE_PUBLIC_KEY - Path to RSA public key PEM file (required for RSA)
// GAUTH_JWE_PRIVATE_KEY - Path to RSA private key PEM file (required for RSA)
// GAUTH_JWE_KEY_ID - Key identifier for key rotation (default: gauth-prod-YYYY-MM)
// GAUTH_JWE_KEY_DIR - Directory containing multiple keys (optional, for key registry)
// GAUTH_JWE_ROTATION_DAYS - Key rotation interval in days (default: 365)
// GAUTH_JWE_COMPRESSION - Enable compression (default: true)
//
// Example usage:
//
//	export GAUTH_JWE_ENABLED=true
//	export GAUTH_JWE_PUBLIC_KEY=/etc/gauth/keys/public.pem
//	export GAUTH_JWE_PRIVATE_KEY=/etc/gauth/keys/private.pem
//	export GAUTH_JWE_KEY_ID=gauth-prod-2025-11
func JWEConfigFromEnv() (*JWEConfig, error) {
	config := DefaultJWEConfig()

	// Parse GAUTH_JWE_ENABLED
	if envEnabled := os.Getenv("GAUTH_JWE_ENABLED"); envEnabled != "" {
		enabled, err := strconv.ParseBool(envEnabled)
		if err != nil {
			return nil, fmt.Errorf("invalid GAUTH_JWE_ENABLED value '%s': %w", envEnabled, err)
		}
		config.Enabled = enabled
	}

	// If disabled, return early
	if !config.Enabled {
		return config, nil
	}

	// Parse GAUTH_JWE_ALGORITHM
	if envAlgorithm := os.Getenv("GAUTH_JWE_ALGORITHM"); envAlgorithm != "" {
		algorithm := strings.ToUpper(envAlgorithm)
		if algorithm != "RSA-OAEP-256" && algorithm != "A256KW" {
			return nil, fmt.Errorf("unsupported GAUTH_JWE_ALGORITHM: %s (supported: RSA-OAEP-256, A256KW)", algorithm)
		}
		config.Algorithm = algorithm
	}

	// Parse GAUTH_JWE_ENCRYPTION
	if envEncryption := os.Getenv("GAUTH_JWE_ENCRYPTION"); envEncryption != "" {
		encryption := strings.ToUpper(envEncryption)
		if encryption != "A256GCM" && encryption != "A128GCM" {
			return nil, fmt.Errorf("unsupported GAUTH_JWE_ENCRYPTION: %s (supported: A256GCM, A128GCM)", encryption)
		}
		config.Encryption = encryption
	}

	// Parse key paths (required for RSA-OAEP-256)
	if envPublicKey := os.Getenv("GAUTH_JWE_PUBLIC_KEY"); envPublicKey != "" {
		config.PublicKeyPath = envPublicKey
	}

	if envPrivateKey := os.Getenv("GAUTH_JWE_PRIVATE_KEY"); envPrivateKey != "" {
		config.PrivateKeyPath = envPrivateKey
	}

	// Parse GAUTH_JWE_KEY_ID
	if envKeyID := os.Getenv("GAUTH_JWE_KEY_ID"); envKeyID != "" {
		config.KeyID = envKeyID
	}

	// Parse GAUTH_JWE_ROTATION_DAYS
	if envRotationDays := os.Getenv("GAUTH_JWE_ROTATION_DAYS"); envRotationDays != "" {
		rotationDays, err := strconv.Atoi(envRotationDays)
		if err != nil {
			return nil, fmt.Errorf("invalid GAUTH_JWE_ROTATION_DAYS value '%s': %w", envRotationDays, err)
		}
		if rotationDays < 1 {
			return nil, fmt.Errorf("GAUTH_JWE_ROTATION_DAYS must be positive (got %d)", rotationDays)
		}
		config.KeyRotationDays = rotationDays
	}

	// Validate the final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid JWE configuration from environment: %w", err)
	}

	return config, nil
}

// JWEConfigFromEnvWithDefaults creates JWE configuration with sensible defaults
// Falls back to development config if environment variables are not set
func JWEConfigFromEnvWithDefaults() (*JWEConfig, error) {
	// Check if any JWE environment variables are set
	hasEnvConfig := os.Getenv("GAUTH_JWE_ENABLED") != "" ||
		os.Getenv("GAUTH_JWE_PUBLIC_KEY") != "" ||
		os.Getenv("GAUTH_JWE_PRIVATE_KEY") != ""

	if hasEnvConfig {
		return JWEConfigFromEnv()
	}

	// Fall back to development config
	return DevelopmentJWEConfig(), nil
}

// ValidateEnvironment checks that all required environment variables are set
// for production deployment with JWE encryption
func ValidateEnvironment() []string {
	var errors []string

	// Check if JWE is enabled
	enabled := os.Getenv("GAUTH_JWE_ENABLED")
	if enabled == "" {
		errors = append(errors, "GAUTH_JWE_ENABLED not set (default: true)")
	} else if enabled != "true" && enabled != "false" {
		errors = append(errors, fmt.Sprintf("GAUTH_JWE_ENABLED must be 'true' or 'false' (got: %s)", enabled))
	}

	// If explicitly disabled, no further checks needed
	if enabled == "false" {
		return errors
	}

	// Check algorithm
	algorithm := os.Getenv("GAUTH_JWE_ALGORITHM")
	if algorithm == "" {
		algorithm = "RSA-OAEP-256" // default
	}

	// For RSA algorithms, check key paths
	if algorithm == "RSA-OAEP-256" || algorithm == "" {
		publicKey := os.Getenv("GAUTH_JWE_PUBLIC_KEY")
		if publicKey == "" {
			errors = append(errors, "GAUTH_JWE_PUBLIC_KEY not set (required for RSA-OAEP-256)")
		} else {
			// Check file exists
			if _, err := os.Stat(publicKey); err != nil {
				errors = append(errors, fmt.Sprintf("GAUTH_JWE_PUBLIC_KEY file not found: %s", publicKey))
			}
		}

		privateKey := os.Getenv("GAUTH_JWE_PRIVATE_KEY")
		if privateKey == "" {
			errors = append(errors, "GAUTH_JWE_PRIVATE_KEY not set (required for RSA-OAEP-256)")
		} else {
			// Check file exists
			if _, err := os.Stat(privateKey); err != nil {
				errors = append(errors, fmt.Sprintf("GAUTH_JWE_PRIVATE_KEY file not found: %s", privateKey))
			}
		}
	}

	// Check key ID
	keyID := os.Getenv("GAUTH_JWE_KEY_ID")
	if keyID == "" {
		errors = append(errors, "GAUTH_JWE_KEY_ID not set (recommended for key rotation)")
	}

	// Check rotation days (optional, but validate if set)
	if rotationDays := os.Getenv("GAUTH_JWE_ROTATION_DAYS"); rotationDays != "" {
		if days, err := strconv.Atoi(rotationDays); err != nil {
			errors = append(errors, fmt.Sprintf("GAUTH_JWE_ROTATION_DAYS must be a number (got: %s)", rotationDays))
		} else if days < 1 {
			errors = append(errors, fmt.Sprintf("GAUTH_JWE_ROTATION_DAYS must be positive (got: %d)", days))
		}
	}

	return errors
}

// PrintEnvironmentHelp prints help text for JWE environment variables
func PrintEnvironmentHelp() {
	fmt.Println("GAuth JWE Environment Variables:")
	fmt.Println()
	fmt.Println("  GAUTH_JWE_ENABLED         Enable/disable JWE encryption (true/false, default: true)")
	fmt.Println("  GAUTH_JWE_ALGORITHM       Key encryption algorithm (RSA-OAEP-256/A256KW, default: RSA-OAEP-256)")
	fmt.Println("  GAUTH_JWE_ENCRYPTION      Content encryption (A256GCM/A128GCM, default: A256GCM)")
	fmt.Println("  GAUTH_JWE_PUBLIC_KEY      Path to RSA public key PEM file (required for RSA)")
	fmt.Println("  GAUTH_JWE_PRIVATE_KEY     Path to RSA private key PEM file (required for RSA)")
	fmt.Println("  GAUTH_JWE_KEY_ID          Key identifier for rotation (e.g., gauth-prod-2025-11)")
	fmt.Println("  GAUTH_JWE_KEY_DIR         Directory containing multiple keys (for key registry)")
	fmt.Println("  GAUTH_JWE_ROTATION_DAYS   Key rotation interval in days (default: 365)")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  export GAUTH_JWE_ENABLED=true")
	fmt.Println("  export GAUTH_JWE_PUBLIC_KEY=/etc/gauth/keys/public.pem")
	fmt.Println("  export GAUTH_JWE_PRIVATE_KEY=/etc/gauth/keys/private.pem")
	fmt.Println("  export GAUTH_JWE_KEY_ID=gauth-prod-2025-11")
	fmt.Println()
}
