package security
package security

import (
	"os"
	"strings"
	"testing"
)

func TestStartupValidator_JWTSigningKey(t *testing.T) {
	tests := []struct {
		name           string
		keyValue       string
		productionMode bool
		expectError    bool
		expectWarning  bool
	}{
		{
			name:           "missing key - critical error",
			keyValue:       "",
			productionMode: true,
			expectError:    true,
		},
		{
			name:           "weak key dev-please-change",
			keyValue:       "dev-please-change",
			productionMode: true,
			expectError:    true,
		},
		{
			name:           "weak key dev-signing-key",
			keyValue:       "dev-signing-key-change-in-production",
			productionMode: true,
			expectError:    true,
		},
		{
			name:           "short key in production",
			keyValue:       "short123",
			productionMode: true,
			expectWarning:  true,
		},
		{
			name:           "strong key in production",
			keyValue:       "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0",
			productionMode: true,
			expectError:    false,
			expectWarning:  false,
		},
		{
			name:           "weak key in dev - allowed",
			keyValue:       "dev-please-change",
			productionMode: false,
			expectError:    true, // Still error even in dev
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			os.Setenv("GAUTH_JWT_SIGNING_KEY", tt.keyValue)
			defer os.Unsetenv("GAUTH_JWT_SIGNING_KEY")

			validator := NewStartupValidator(tt.productionMode)
			err := validator.ValidateAll()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
			if tt.expectWarning && len(validator.GetWarnings()) == 0 {
				t.Error("Expected warning but got none")
			}
		})
	}
}

func TestStartupValidator_ProductionMode(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		expectError   bool
		expectWarning bool
	}{
		{
			name: "dev index enabled in production",
			envVars: map[string]string{
				"GAUTH_JWT_SIGNING_KEY": "strong-key-12345678901234567890",
				"GAUTH_DEV_INDEX":       "1",
			},
			expectError: true,
		},
		{
			name: "mock PVP in production",
			envVars: map[string]string{
				"GAUTH_JWT_SIGNING_KEY": "strong-key-12345678901234567890",
				"GAUTH_PVP_PROVIDER":    "mock",
			},
			expectWarning: true,
		},
		{
			name: "rate limiting disabled in production",
			envVars: map[string]string{
				"GAUTH_JWT_SIGNING_KEY":     "strong-key-12345678901234567890",
				"GAUTH_RATE_LIMIT_ENABLED": "false",
			},
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			validator := NewStartupValidator(true) // Production mode
			err := validator.ValidateAll()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if tt.expectWarning && len(validator.GetWarnings()) == 0 {
				t.Error("Expected warning but got none")
			}
		})
	}
}

func TestURIValidator_SSRFProtection(t *testing.T) {
	validator := NewURIValidator()

	tests := []struct {
		name        string
		uri         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid https URI",
			uri:         "https://api.example.com/resource",
			expectError: false,
		},
		{
			name:        "file URI blocked",
			uri:         "file:///etc/passwd",
			expectError: true,
			errorMsg:    "scheme",
		},
		{
			name:        "localhost blocked",
			uri:         "https://localhost:8080/admin",
			expectError: true,
			errorMsg:    "localhost",
		},
		{
			name:        "127.0.0.1 blocked",
			uri:         "https://127.0.0.1/internal",
			expectError: true,
			errorMsg:    "localhost",
		},
		{
			name:        "AWS metadata blocked",
			uri:         "http://169.254.169.254/latest/meta-data/",
			expectError: true,
			errorMsg:    "metadata",
		},
		{
			name:        "private IP blocked",
			uri:         "https://192.168.1.1/admin",
			expectError: true,
			errorMsg:    "private IP",
		},
		{
			name:        "private IP 10.x blocked",
			uri:         "https://10.0.0.1/internal",
			expectError: true,
			errorMsg:    "private IP",
		},
		{
			name:        "GCP metadata blocked",
			uri:         "http://metadata.google.internal/computeMetadata/v1/",
			expectError: true,
			errorMsg:    "metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateURI(tt.uri)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for URI %s but got none", tt.uri)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for URI %s: %v", tt.uri, err)
			}
			if tt.expectError && err != nil && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
				t.Errorf("Error message should contain '%s', got: %v", tt.errorMsg, err)
			}
		})
	}
}

func TestURIValidator_CustomSchemes(t *testing.T) {
	// Allow mcp:// scheme for testing
	validator := NewURIValidatorWithSchemes([]string{"mcp", "https"})

	tests := []struct {
		name        string
		uri         string
		expectError bool
	}{
		{
			name:        "mcp scheme allowed",
			uri:         "mcp://server/resource",
			expectError: false,
		},
		{
			name:        "https still allowed",
			uri:         "https://api.example.com/data",
			expectError: false,
		},
		{
			name:        "http not allowed",
			uri:         "http://api.example.com/data",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateURI(tt.uri)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for URI %s but got none", tt.uri)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for URI %s: %v", tt.uri, err)
			}
		})
	}
}

func TestProductionModeDetector(t *testing.T) {
	tests := []struct {
		name             string
		envVars          map[string]string
		expectProduction bool
	}{
		{
			name: "GAUTH_ENV=production",
			envVars: map[string]string{
				"GAUTH_ENV": "production",
			},
			expectProduction: true,
		},
		{
			name: "GAUTH_MODE=prod",
			envVars: map[string]string{
				"GAUTH_MODE": "prod",
			},
			expectProduction: true,
		},
		{
			name: "development mode",
			envVars: map[string]string{
				"GAUTH_ENV":      "development",
				"GAUTH_DEV_MODE": "true",
			},
			expectProduction: false,
		},
		{
			name: "no env vars - defaults to dev",
			envVars: map[string]string{
				"GAUTH_DEV_INDEX": "1",
			},
			expectProduction: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars first
			clearEnvVars := []string{"GAUTH_ENV", "GAUTH_MODE", "GAUTH_DEV_MODE", "GAUTH_DEV_INDEX", "GAUTH_PORT"}
			for _, key := range clearEnvVars {
				os.Unsetenv(key)
			}

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := ProductionModeDetector()
			if result != tt.expectProduction {
				t.Errorf("Expected production=%v, got %v", tt.expectProduction, result)
			}
		})
	}
}
