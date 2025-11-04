package config

import (
	"os"
	"testing"
)

func TestDefaultCascadeConfig(t *testing.T) {
	config := DefaultCascadeConfig()

	if config.Enabled {
		t.Error("default config should be disabled")
	}
	if config.Mode != CascadeModeSuspend {
		t.Errorf("expected default mode %v, got %v", CascadeModeSuspend, config.Mode)
	}
	if config.MaxDepth != 10 {
		t.Errorf("expected default max depth 10, got %d", config.MaxDepth)
	}
	if config.BatchSize != 100 {
		t.Errorf("expected default batch size 100, got %d", config.BatchSize)
	}
}

func TestLoadCascadeConfigFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected CascadeConfig
	}{
		{
			name:     "defaults when no env vars set",
			envVars:  map[string]string{},
			expected: DefaultCascadeConfig(),
		},
		{
			name: "enabled with revoke mode",
			envVars: map[string]string{
				"GAUTH_CASCADE_PARENT_REVOCATION": "1",
				"GAUTH_CASCADE_MODE":              "revoke",
				"GAUTH_CASCADE_MAX_DEPTH":         "5",
				"GAUTH_CASCADE_BATCH_SIZE":        "50",
			},
			expected: CascadeConfig{
				Enabled:   true,
				Mode:      CascadeModeRevoke,
				MaxDepth:  5,
				BatchSize: 50,
			},
		},
		{
			name: "enabled with suspend mode",
			envVars: map[string]string{
				"GAUTH_CASCADE_PARENT_REVOCATION": "true",
				"GAUTH_CASCADE_MODE":              "suspend",
			},
			expected: CascadeConfig{
				Enabled:   true,
				Mode:      CascadeModeSuspend,
				MaxDepth:  10,
				BatchSize: 100,
			},
		},
		{
			name: "enabled with notify mode",
			envVars: map[string]string{
				"GAUTH_CASCADE_PARENT_REVOCATION": "yes",
				"GAUTH_CASCADE_MODE":              "notify",
				"GAUTH_CASCADE_MAX_DEPTH":         "0", // unlimited
			},
			expected: CascadeConfig{
				Enabled:   true,
				Mode:      CascadeModeNotify,
				MaxDepth:  0,
				BatchSize: 100,
			},
		},
		{
			name: "invalid mode falls back to default",
			envVars: map[string]string{
				"GAUTH_CASCADE_PARENT_REVOCATION": "1",
				"GAUTH_CASCADE_MODE":              "invalid",
			},
			expected: CascadeConfig{
				Enabled:   true,
				Mode:      CascadeModeSuspend, // fallback to default
				MaxDepth:  10,
				BatchSize: 100,
			},
		},
		{
			name: "invalid numbers fall back to defaults",
			envVars: map[string]string{
				"GAUTH_CASCADE_PARENT_REVOCATION": "1",
				"GAUTH_CASCADE_MAX_DEPTH":         "invalid",
				"GAUTH_CASCADE_BATCH_SIZE":        "-1",
			},
			expected: CascadeConfig{
				Enabled:   true,
				Mode:      CascadeModeSuspend,
				MaxDepth:  10,  // fallback
				BatchSize: 100, // fallback (negative rejected)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv := []string{
				"GAUTH_CASCADE_PARENT_REVOCATION",
				"GAUTH_CASCADE_MODE",
				"GAUTH_CASCADE_MAX_DEPTH",
				"GAUTH_CASCADE_BATCH_SIZE",
			}
			for _, key := range clearEnv {
				os.Unsetenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Cleanup after test
			t.Cleanup(func() {
				for _, key := range clearEnv {
					os.Unsetenv(key)
				}
			})

			config := LoadCascadeConfigFromEnv()

			if config.Enabled != tt.expected.Enabled {
				t.Errorf("expected Enabled %v, got %v", tt.expected.Enabled, config.Enabled)
			}
			if config.Mode != tt.expected.Mode {
				t.Errorf("expected Mode %v, got %v", tt.expected.Mode, config.Mode)
			}
			if config.MaxDepth != tt.expected.MaxDepth {
				t.Errorf("expected MaxDepth %d, got %d", tt.expected.MaxDepth, config.MaxDepth)
			}
			if config.BatchSize != tt.expected.BatchSize {
				t.Errorf("expected BatchSize %d, got %d", tt.expected.BatchSize, config.BatchSize)
			}
		})
	}
}

func TestCascadeConfigMethods(t *testing.T) {
	tests := []struct {
		name      string
		config    CascadeConfig
		valid     bool
		should    bool
		unlimited bool
	}{
		{
			name: "valid revoke mode",
			config: CascadeConfig{
				Enabled:   true,
				Mode:      CascadeModeRevoke,
				MaxDepth:  5,
				BatchSize: 50,
			},
			valid:     true,
			should:    true,
			unlimited: false,
		},
		{
			name: "valid suspend mode",
			config: CascadeConfig{
				Enabled:   true,
				Mode:      CascadeModeSuspend,
				MaxDepth:  0, // unlimited
				BatchSize: 100,
			},
			valid:     true,
			should:    true,
			unlimited: true,
		},
		{
			name: "valid notify mode",
			config: CascadeConfig{
				Enabled:   true,
				Mode:      CascadeModeNotify,
				MaxDepth:  10,
				BatchSize: 25,
			},
			valid:     true,
			should:    true,
			unlimited: false,
		},
		{
			name: "invalid mode",
			config: CascadeConfig{
				Enabled:   true,
				Mode:      "invalid",
				MaxDepth:  10,
				BatchSize: 100,
			},
			valid:     false,
			should:    false,
			unlimited: false,
		},
		{
			name: "disabled config",
			config: CascadeConfig{
				Enabled:   false,
				Mode:      CascadeModeRevoke,
				MaxDepth:  10,
				BatchSize: 100,
			},
			valid:     true,
			should:    false,
			unlimited: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if valid := tt.config.IsValidMode(); valid != tt.valid {
				t.Errorf("expected IsValidMode %v, got %v", tt.valid, valid)
			}
			if should := tt.config.ShouldProcessCascade(); should != tt.should {
				t.Errorf("expected ShouldProcessCascade %v, got %v", tt.should, should)
			}
			if unlimited := tt.config.IsUnlimitedDepth(); unlimited != tt.unlimited {
				t.Errorf("expected IsUnlimitedDepth %v, got %v", tt.unlimited, unlimited)
			}
		})
	}
}
