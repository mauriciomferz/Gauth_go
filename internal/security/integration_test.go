package security_test

import (
	"context"
	"testing"

	"github.com/mauriciomferz/AgentAuth/internal/security"
	"github.com/mauriciomferz/AgentAuth/pkg/mcp"
)

// TestForgedTokenRejection ensures forged tokens are rejected
func TestForgedTokenRejection(t *testing.T) {
	// This test validates that using weak/known signing keys
	// would allow token forgery - demonstrating the vulnerability

	t.Run("weak signing key allows forgery", func(t *testing.T) {
		// Setup weak key
		t.Setenv("AGENTAUTH_JWT_SIGNING_KEY", "dev-please-change")

		// Validator should catch this
		validator := security.NewStartupValidator(true)
		err := validator.ValidateAll()

		if err == nil {
			t.Fatal("Expected validation to fail with weak signing key")
		}

		if len(validator.GetErrors()) == 0 {
			t.Fatal("Expected errors for weak signing key")
		}
	})

	t.Run("strong signing key passes", func(t *testing.T) {
		// Setup strong key
		t.Setenv("AGENTAUTH_JWT_SIGNING_KEY", "strong-random-key-1234567890abcdefghijklmnopqrstuvwxyz")

		validator := security.NewStartupValidator(true)
		err := validator.ValidateAll()

		if err != nil {
			t.Fatalf("Strong key should pass validation: %v", err)
		}
	})
}

// TestSSRFPrevention validates MCP client blocks SSRF vectors
func TestSSRFPrevention(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expectError bool
		description string
	}{
		{
			name:        "file scheme blocked",
			uri:         "file:///etc/passwd",
			expectError: true,
			description: "Local file access must be blocked",
		},
		{
			name:        "localhost blocked",
			uri:         "https://localhost/admin",
			expectError: true,
			description: "Localhost access must be blocked",
		},
		{
			name:        "127.0.0.1 blocked",
			uri:         "https://127.0.0.1/internal",
			expectError: true,
			description: "Loopback IP must be blocked",
		},
		{
			name:        "AWS metadata blocked",
			uri:         "http://169.254.169.254/latest/meta-data/iam/credentials",
			expectError: true,
			description: "Cloud metadata endpoint must be blocked",
		},
		{
			name:        "private IP 192.168.x blocked",
			uri:         "https://192.168.1.100/api",
			expectError: true,
			description: "Private IP ranges must be blocked",
		},
		{
			name:        "private IP 10.x blocked",
			uri:         "https://10.0.0.1/internal",
			expectError: true,
			description: "RFC1918 private ranges must be blocked",
		},
		{
			name:        "GCP metadata blocked",
			uri:         "http://metadata.google.internal/computeMetadata/v1/",
			expectError: true,
			description: "GCP metadata must be blocked",
		},
		{
			name:        "valid https allowed",
			uri:         "https://api.example.com/resource",
			expectError: false,
			description: "Legitimate external HTTPS should work",
		},
		{
			name:        "valid mcp allowed",
			uri:         "mcp://server/resource",
			expectError: false,
			description: "MCP protocol should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create MCP client with mock transport
			transport := &mockTransport{}
			client := mcp.NewMCPClient("test-server", "test", transport)

			// Attempt to read resource
			_, err := client.ReadResource(context.Background(), tt.uri)

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}

			if tt.expectError && err != nil {
				// Verify error message indicates SSRF protection
				errMsg := err.Error()
				if !containsAny(errMsg, []string{"SSRF", "blocked", "validation", "scheme", "private", "metadata"}) {
					t.Errorf("Error should indicate SSRF protection, got: %v", err)
				}
			}
		})
	}
}

// TestIdentityVerificationEnforcement validates PVP requirements
func TestIdentityVerificationEnforcement(t *testing.T) {
	t.Run("mock PVP blocked in production", func(t *testing.T) {
		t.Setenv("AGENTAUTH_PVP_PROVIDER", "mock")
		t.Setenv("AGENTAUTH_JWT_SIGNING_KEY", "strong-key-12345678901234567890123456")

		validator := security.NewStartupValidator(true) // Production mode
		err := validator.ValidateAll()

		// Should have warning about mock PVP
		warnings := validator.GetWarnings()
		foundPVPWarning := false
		for _, w := range warnings {
			if containsAny(w, []string{"PVP", "mock", "identity", "verification"}) {
				foundPVPWarning = true
				break
			}
		}

		if !foundPVPWarning && err == nil {
			t.Error("Expected warning or error about mock PVP in production")
		}
	})

	t.Run("real PVP provider required in production", func(t *testing.T) {
		t.Setenv("AGENTAUTH_PVP_PROVIDER", "stripe")
		t.Setenv("AGENTAUTH_JWT_SIGNING_KEY", "strong-key-12345678901234567890123456")

		validator := security.NewStartupValidator(true)
		err := validator.ValidateAll()

		// Should pass validation (implementation is separate concern)
		if err != nil {
			t.Errorf("Real PVP provider should pass validation: %v", err)
		}
	})
}

// TestDebugEndpointsBlocked validates debug features are disabled in production
func TestDebugEndpointsBlocked(t *testing.T) {
	t.Run("AGENTAUTH_DEV_INDEX blocked in production", func(t *testing.T) {
		t.Setenv("AGENTAUTH_DEV_INDEX", "1")
		t.Setenv("AGENTAUTH_JWT_SIGNING_KEY", "strong-key-12345678901234567890123456")

		validator := security.NewStartupValidator(true) // Production mode
		err := validator.ValidateAll()

		if err == nil {
			t.Fatal("Expected error when AGENTAUTH_DEV_INDEX=1 in production")
		}

		if !containsAny(err.Error(), []string{"DEV_INDEX", "debug", "development"}) {
			t.Errorf("Error should mention DEV_INDEX, got: %v", err)
		}
	})

	t.Run("AGENTAUTH_DEV_MODE blocked in production", func(t *testing.T) {
		t.Setenv("AGENTAUTH_DEV_MODE", "true")
		t.Setenv("AGENTAUTH_JWT_SIGNING_KEY", "strong-key-12345678901234567890123456")

		validator := security.NewStartupValidator(true)
		err := validator.ValidateAll()

		if err == nil {
			t.Fatal("Expected error when AGENTAUTH_DEV_MODE=true in production")
		}

		if !containsAny(err.Error(), []string{"DEV_MODE", "development"}) {
			t.Errorf("Error should mention DEV_MODE, got: %v", err)
		}
	})

	t.Run("debug settings allowed in development", func(t *testing.T) {
		t.Setenv("AGENTAUTH_DEV_INDEX", "1")
		t.Setenv("AGENTAUTH_DEV_MODE", "true")
		t.Setenv("AGENTAUTH_JWT_SIGNING_KEY", "dev-key-for-testing")

		validator := security.NewStartupValidator(false) // Development mode
		// Note: dev key will still fail even in dev mode
		// This is intentional - weak keys are never acceptable
		err := validator.ValidateAll()

		// Should fail due to weak key, not dev settings
		if err != nil && containsAny(err.Error(), []string{"DEV_INDEX", "DEV_MODE"}) {
			t.Error("Dev settings should be allowed in development mode")
		}
	})
}

// TestProductionModeDetection validates mode detection logic
func TestProductionModeDetection(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		expectProd bool
	}{
		{
			name: "AGENTAUTH_ENV=production",
			envVars: map[string]string{
				"AGENTAUTH_ENV": "production",
			},
			expectProd: true,
		},
		{
			name: "AGENTAUTH_MODE=prod",
			envVars: map[string]string{
				"AGENTAUTH_MODE": "prod",
			},
			expectProd: true,
		},
		{
			name: "development with dev index",
			envVars: map[string]string{
				"AGENTAUTH_DEV_INDEX": "1",
			},
			expectProd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env
			clearKeys := []string{"AGENTAUTH_ENV", "AGENTAUTH_MODE", "AGENTAUTH_DEV_MODE", "AGENTAUTH_DEV_INDEX"}
			for _, k := range clearKeys {
				t.Setenv(k, "")
			}

			// Set test env
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			isProd := security.ProductionModeDetector()
			if isProd != tt.expectProd {
				t.Errorf("Expected production=%v, got %v", tt.expectProd, isProd)
			}
		})
	}
}

// Helper functions

type mockTransport struct{}

func (m *mockTransport) Send(ctx context.Context, message []byte) error {
	return nil
}

func (m *mockTransport) Receive(ctx context.Context) ([]byte, error) {
	// Return a valid JSONRPC response with matching ID
	return []byte(`{"jsonrpc": "2.0", "id": 1, "result": {"contents": [{"uri": "test", "text": "test"}]}}`), nil
}

func (m *mockTransport) Close() error {
	return nil
}

func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	// Case-insensitive contains
	s = toLower(s)
	substr = toLower(substr)
	return len(s) >= len(substr) && (s == substr || stringContains(s, substr))
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
