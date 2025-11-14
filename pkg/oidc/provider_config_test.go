package oidc

import (
	"testing"
)

func TestProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProviderConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			cfg: ProviderConfig{
				ID:                "google",
				Name:              "Google",
				IssuerURL:         "https://accounts.google.com",
				ClientID:          "test-client-id",
				ClientSecret:      "test-client-secret",
				Scopes:            []string{"openid", "profile", "email"},
				DefaultTrustLevel: "substantial",
				Enabled:           true,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			cfg: ProviderConfig{
				Name:         "Google",
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid"},
			},
			wantErr: true,
			errMsg:  "provider ID is required",
		},
		{
			name: "missing name",
			cfg: ProviderConfig{
				ID:           "google",
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid"},
			},
			wantErr: true,
			errMsg:  "provider name is required",
		},
		{
			name: "missing issuer URL",
			cfg: ProviderConfig{
				ID:           "google",
				Name:         "Google",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid"},
			},
			wantErr: true,
			errMsg:  "issuer URL is required",
		},
		{
			name: "non-HTTPS issuer URL",
			cfg: ProviderConfig{
				ID:           "google",
				Name:         "Google",
				IssuerURL:    "http://accounts.google.com",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid"},
			},
			wantErr: true,
			errMsg:  "issuer URL must use HTTPS",
		},
		{
			name: "invalid issuer URL format",
			cfg: ProviderConfig{
				ID:           "google",
				Name:         "Google",
				IssuerURL:    "://invalid-url",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid"},
			},
			wantErr: true,
			errMsg:  "invalid issuer URL",
		},
		{
			name: "missing client ID",
			cfg: ProviderConfig{
				ID:           "google",
				Name:         "Google",
				IssuerURL:    "https://accounts.google.com",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid"},
			},
			wantErr: true,
			errMsg:  "client ID is required",
		},
		{
			name: "missing client secret",
			cfg: ProviderConfig{
				ID:        "google",
				Name:      "Google",
				IssuerURL: "https://accounts.google.com",
				ClientID:  "test-client-id",
				Scopes:    []string{"openid"},
			},
			wantErr: true,
			errMsg:  "client secret is required",
		},
		{
			name: "missing scopes",
			cfg: ProviderConfig{
				ID:           "google",
				Name:         "Google",
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{},
			},
			wantErr: true,
			errMsg:  "at least one scope is required",
		},
		{
			name: "missing openid scope",
			cfg: ProviderConfig{
				ID:           "google",
				Name:         "Google",
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"profile", "email"},
			},
			wantErr: true,
			errMsg:  "scopes must include 'openid'",
		},
		{
			name: "invalid trust level",
			cfg: ProviderConfig{
				ID:                "google",
				Name:              "Google",
				IssuerURL:         "https://accounts.google.com",
				ClientID:          "test-client-id",
				ClientSecret:      "test-client-secret",
				Scopes:            []string{"openid"},
				DefaultTrustLevel: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid default trust level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestProviderConfig_GetDiscoveryURL(t *testing.T) {
	tests := []struct {
		name      string
		issuerURL string
		want      string
	}{
		{
			name:      "Google",
			issuerURL: "https://accounts.google.com",
			want:      "https://accounts.google.com/.well-known/openid-configuration",
		},
		{
			name:      "issuer with trailing slash",
			issuerURL: "https://accounts.google.com/",
			want:      "https://accounts.google.com/.well-known/openid-configuration",
		},
		{
			name:      "Okta",
			issuerURL: "https://dev-12345.okta.com",
			want:      "https://dev-12345.okta.com/.well-known/openid-configuration",
		},
		{
			name:      "Azure AD",
			issuerURL: "https://login.microsoftonline.com/tenant-id/v2.0",
			want:      "https://login.microsoftonline.com/tenant-id/v2.0/.well-known/openid-configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ProviderConfig{IssuerURL: tt.issuerURL}
			got := cfg.GetDiscoveryURL()
			if got != tt.want {
				t.Errorf("GetDiscoveryURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInMemoryProviderRegistry_Register(t *testing.T) {
	registry := NewInMemoryProviderRegistry()

	validCfg := ProviderConfig{
		ID:           "google",
		Name:         "Google",
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Scopes:       []string{"openid", "profile", "email"},
		Enabled:      true,
	}

	// Register valid configuration
	err := registry.Register(validCfg)
	if err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}

	// Try to register duplicate
	err = registry.Register(validCfg)
	if err == nil {
		t.Error("Register() expected error for duplicate ID, got nil")
	}

	// Register invalid configuration
	invalidCfg := ProviderConfig{
		ID: "invalid",
		// Missing required fields
	}
	err = registry.Register(invalidCfg)
	if err == nil {
		t.Error("Register() expected error for invalid config, got nil")
	}
}

func TestInMemoryProviderRegistry_Get(t *testing.T) {
	registry := NewInMemoryProviderRegistry()

	cfg := ProviderConfig{
		ID:           "google",
		Name:         "Google",
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Scopes:       []string{"openid"},
		Enabled:      true,
	}

	// Register provider
	if err := registry.Register(cfg); err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}

	// Get existing provider
	got, err := registry.Get("google")
	if err != nil {
		t.Fatalf("Get() unexpected error = %v", err)
	}

	if got.ID != cfg.ID || got.Name != cfg.Name {
		t.Errorf("Get() = %+v, want %+v", got, cfg)
	}

	// Get non-existent provider
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Get() expected error for non-existent provider, got nil")
	}
}

func TestInMemoryProviderRegistry_List(t *testing.T) {
	registry := NewInMemoryProviderRegistry()

	// Empty registry
	list := registry.List()
	if len(list) != 0 {
		t.Errorf("List() expected empty list, got %d items", len(list))
	}

	// Register multiple providers
	providers := []ProviderConfig{
		{
			ID:           "google",
			Name:         "Google",
			IssuerURL:    "https://accounts.google.com",
			ClientID:     "test-client-id-1",
			ClientSecret: "test-secret-1",
			Scopes:       []string{"openid"},
			Enabled:      true,
		},
		{
			ID:           "okta",
			Name:         "Okta",
			IssuerURL:    "https://dev.okta.com",
			ClientID:     "test-client-id-2",
			ClientSecret: "test-secret-2",
			Scopes:       []string{"openid"},
			Enabled:      false,
		},
	}

	for _, p := range providers {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register() unexpected error = %v", err)
		}
	}

	list = registry.List()
	if len(list) != 2 {
		t.Errorf("List() expected 2 items, got %d", len(list))
	}
}

func TestInMemoryProviderRegistry_ListEnabled(t *testing.T) {
	registry := NewInMemoryProviderRegistry()

	providers := []ProviderConfig{
		{
			ID:           "google",
			Name:         "Google",
			IssuerURL:    "https://accounts.google.com",
			ClientID:     "test-client-id-1",
			ClientSecret: "test-secret-1",
			Scopes:       []string{"openid"},
			Enabled:      true,
		},
		{
			ID:           "okta",
			Name:         "Okta",
			IssuerURL:    "https://dev.okta.com",
			ClientID:     "test-client-id-2",
			ClientSecret: "test-secret-2",
			Scopes:       []string{"openid"},
			Enabled:      false,
		},
		{
			ID:           "azure",
			Name:         "Azure AD",
			IssuerURL:    "https://login.microsoftonline.com/tenant/v2.0",
			ClientID:     "test-client-id-3",
			ClientSecret: "test-secret-3",
			Scopes:       []string{"openid"},
			Enabled:      true,
		},
	}

	for _, p := range providers {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register() unexpected error = %v", err)
		}
	}

	enabled := registry.ListEnabled()
	if len(enabled) != 2 {
		t.Errorf("ListEnabled() expected 2 items, got %d", len(enabled))
	}

	for _, p := range enabled {
		if !p.Enabled {
			t.Errorf("ListEnabled() returned disabled provider: %s", p.ID)
		}
	}
}

func TestInMemoryProviderRegistry_Update(t *testing.T) {
	registry := NewInMemoryProviderRegistry()

	original := ProviderConfig{
		ID:           "google",
		Name:         "Google",
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "old-client-id",
		ClientSecret: "old-secret",
		Scopes:       []string{"openid"},
		Enabled:      true,
	}

	if err := registry.Register(original); err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}

	// Update existing provider
	updated := original
	updated.ClientID = "new-client-id"
	updated.Name = "Google Updated"

	err := registry.Update("google", updated)
	if err != nil {
		t.Fatalf("Update() unexpected error = %v", err)
	}

	got, _ := registry.Get("google")
	if got.ClientID != "new-client-id" || got.Name != "Google Updated" {
		t.Errorf("Update() did not update provider correctly")
	}

	// Update non-existent provider
	err = registry.Update("nonexistent", updated)
	if err == nil {
		t.Error("Update() expected error for non-existent provider, got nil")
	}

	// Update with invalid configuration
	invalid := ProviderConfig{
		ID: "google",
		// Missing required fields
	}
	err = registry.Update("google", invalid)
	if err == nil {
		t.Error("Update() expected error for invalid config, got nil")
	}
}

func TestInMemoryProviderRegistry_Delete(t *testing.T) {
	registry := NewInMemoryProviderRegistry()

	cfg := ProviderConfig{
		ID:           "google",
		Name:         "Google",
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Scopes:       []string{"openid"},
		Enabled:      true,
	}

	if err := registry.Register(cfg); err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}

	// Delete existing provider
	err := registry.Delete("google")
	if err != nil {
		t.Fatalf("Delete() unexpected error = %v", err)
	}

	// Verify deleted
	_, err = registry.Get("google")
	if err == nil {
		t.Error("Get() expected error after delete, got nil")
	}

	// Delete non-existent provider
	err = registry.Delete("nonexistent")
	if err == nil {
		t.Error("Delete() expected error for non-existent provider, got nil")
	}
}

func TestInMemoryProviderRegistry_EnableDisable(t *testing.T) {
	registry := NewInMemoryProviderRegistry()

	cfg := ProviderConfig{
		ID:           "google",
		Name:         "Google",
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Scopes:       []string{"openid"},
		Enabled:      false,
	}

	if err := registry.Register(cfg); err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}

	// Enable provider
	err := registry.Enable("google")
	if err != nil {
		t.Fatalf("Enable() unexpected error = %v", err)
	}

	got, _ := registry.Get("google")
	if !got.Enabled {
		t.Error("Enable() did not enable provider")
	}

	// Disable provider
	err = registry.Disable("google")
	if err != nil {
		t.Fatalf("Disable() unexpected error = %v", err)
	}

	got, _ = registry.Get("google")
	if got.Enabled {
		t.Error("Disable() did not disable provider")
	}

	// Enable non-existent provider
	err = registry.Enable("nonexistent")
	if err == nil {
		t.Error("Enable() expected error for non-existent provider, got nil")
	}

	// Disable non-existent provider
	err = registry.Disable("nonexistent")
	if err == nil {
		t.Error("Disable() expected error for non-existent provider, got nil")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
