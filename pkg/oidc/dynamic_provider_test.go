package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDynamicProviderService_RegisterProvider tests basic provider registration.
func TestDynamicProviderService_RegisterProvider(t *testing.T) {
	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	ctx := context.Background()

	t.Run("register_without_discovery", func(t *testing.T) {
		cfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:                "test-provider",
				Name:              "Test Provider",
				IssuerURL:         "https://test.example.com",
				ClientID:          "test-client",
				ClientSecret:      "test-secret",
				Scopes:            []string{"openid", "profile"},
				DefaultTrustLevel: "substantial",
				Enabled:           true,
			},
			AutoDiscover: false,
		}

		result, err := service.RegisterProvider(ctx, cfg)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "test-provider", result.ProviderID)
		assert.False(t, result.DiscoveryFetched)

		// Verify provider is registered
		provider, err := registry.Get("test-provider")
		require.NoError(t, err)
		assert.Equal(t, "Test Provider", provider.Name)
	})

	t.Run("register_duplicate_fails", func(t *testing.T) {
		cfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:                "duplicate-provider",
				Name:              "Duplicate",
				IssuerURL:         "https://dup.example.com",
				ClientID:          "dup-client",
				ClientSecret:      "dup-secret",
				Scopes:            []string{"openid"},
				DefaultTrustLevel: "low",
				Enabled:           true,
			},
		}

		// Register first time
		_, err := service.RegisterProvider(ctx, cfg)
		require.NoError(t, err)

		// Try to register again
		_, err = service.RegisterProvider(ctx, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("register_invalid_config", func(t *testing.T) {
		cfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:   "invalid",
				Name: "Invalid Provider",
				// Missing required fields
			},
		}

		result, err := service.RegisterProvider(ctx, cfg)
		assert.Error(t, err)
		assert.False(t, result.Success)
	})
}

// TestDynamicProviderService_RegisterWithDiscovery tests provider registration with auto-discovery.
func TestDynamicProviderService_RegisterWithDiscovery(t *testing.T) {
	// Create mock discovery server with TLS
	var serverURL string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			discoveryDoc := DiscoveryMetadata{
				Issuer:                 serverURL,
				AuthorizationEndpoint:  serverURL + "/authorize",
				TokenEndpoint:          serverURL + "/token",
				JwksURI:                serverURL + "/jwks",
				ScopesSupported:        []string{"openid", "profile", "email"},
				ResponseTypesSupported: []string{"code"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(discoveryDoc)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	// Use server's client that trusts its certificate
	service.httpClient = server.Client()
	ctx := context.Background()

	t.Run("register_with_successful_discovery", func(t *testing.T) {
		cfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:                "discovery-provider",
				Name:              "Discovery Provider",
				IssuerURL:         serverURL,
				ClientID:          "client-id",
				ClientSecret:      "client-secret",
				Scopes:            []string{"openid"},
				DefaultTrustLevel: "substantial",
				Enabled:           true,
			},
			AutoDiscover: true,
		}

		result, err := service.RegisterProvider(ctx, cfg)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.DiscoveryFetched)
		assert.NotNil(t, result.Metadata)
		assert.Equal(t, serverURL, result.Metadata.Issuer)

		// Verify metadata was stored
		provider, err := registry.Get("discovery-provider")
		require.NoError(t, err)
		assert.NotNil(t, provider.Metadata)
		assert.Contains(t, provider.Metadata, "discovery")
	})

	t.Run("register_with_failed_discovery_continues", func(t *testing.T) {
		cfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:                "failed-discovery",
				Name:              "Failed Discovery",
				IssuerURL:         "https://nonexistent.example.com",
				ClientID:          "client-id",
				ClientSecret:      "client-secret",
				Scopes:            []string{"openid"},
				DefaultTrustLevel: "low",
				Enabled:           true,
			},
			AutoDiscover: true,
		}

		result, err := service.RegisterProvider(ctx, cfg)
		require.NoError(t, err) // Should succeed despite discovery failure
		assert.True(t, result.Success)
		assert.False(t, result.DiscoveryFetched)
		assert.Contains(t, result.Error, "discovery failed")
	})
}

// TestDynamicProviderService_UpdateProvider tests provider updates.
func TestDynamicProviderService_UpdateProvider(t *testing.T) {
	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	ctx := context.Background()

	// Register initial provider
	initialCfg := DynamicProviderConfig{
		ProviderConfig: ProviderConfig{
			ID:                "update-test",
			Name:              "Initial Name",
			IssuerURL:         "https://initial.example.com",
			ClientID:          "initial-client",
			ClientSecret:      "initial-secret",
			Scopes:            []string{"openid"},
			DefaultTrustLevel: "low",
			Enabled:           true,
		},
	}

	_, err := service.RegisterProvider(ctx, initialCfg)
	require.NoError(t, err)

	t.Run("update_provider_success", func(t *testing.T) {
		updatedCfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:                "update-test",
				Name:              "Updated Name",
				IssuerURL:         "https://initial.example.com",
				ClientID:          "updated-client",
				ClientSecret:      "updated-secret",
				Scopes:            []string{"openid", "profile"},
				DefaultTrustLevel: "substantial",
				Enabled:           true,
			},
		}

		err := service.UpdateProvider(ctx, "update-test", updatedCfg)
		require.NoError(t, err)

		// Verify update
		provider, err := registry.Get("update-test")
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", provider.Name)
		assert.Equal(t, "updated-client", provider.ClientID)
		assert.Equal(t, "substantial", provider.DefaultTrustLevel)
	})

	t.Run("update_nonexistent_provider", func(t *testing.T) {
		cfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:                "nonexistent",
				Name:              "Nonexistent",
				IssuerURL:         "https://test.example.com",
				ClientID:          "client",
				ClientSecret:      "secret",
				Scopes:            []string{"openid"},
				DefaultTrustLevel: "low",
				Enabled:           true,
			},
		}

		err := service.UpdateProvider(ctx, "nonexistent", cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestDynamicProviderService_DeleteProvider tests provider deletion.
func TestDynamicProviderService_DeleteProvider(t *testing.T) {
	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	ctx := context.Background()

	// Register provider
	cfg := DynamicProviderConfig{
		ProviderConfig: ProviderConfig{
			ID:                "delete-test",
			Name:              "Delete Test",
			IssuerURL:         "https://delete.example.com",
			ClientID:          "delete-client",
			ClientSecret:      "delete-secret",
			Scopes:            []string{"openid"},
			DefaultTrustLevel: "low",
			Enabled:           true,
		},
		TenantID: "tenant-1",
	}

	_, err := service.RegisterProvider(ctx, cfg)
	require.NoError(t, err)

	t.Run("delete_provider_success", func(t *testing.T) {
		err := service.DeleteProvider(ctx, "delete-test")
		require.NoError(t, err)

		// Verify deletion
		_, err = registry.Get("delete-test")
		assert.Error(t, err)

		// Verify tenant association cleaned up
		tenants := service.GetRegisteredTenants()
		assert.NotContains(t, tenants, "tenant-1")
	})

	t.Run("delete_nonexistent_provider", func(t *testing.T) {
		err := service.DeleteProvider(ctx, "nonexistent")
		assert.Error(t, err)
	})
}

// TestDynamicProviderService_TenantIsolation tests multi-tenant provider isolation.
func TestDynamicProviderService_TenantIsolation(t *testing.T) {
	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	ctx := context.Background()

	// Register providers for different tenants
	tenant1Cfg := DynamicProviderConfig{
		ProviderConfig: ProviderConfig{
			ID:                "tenant1-provider",
			Name:              "Tenant 1 Provider",
			IssuerURL:         "https://tenant1.example.com",
			ClientID:          "tenant1-client",
			ClientSecret:      "tenant1-secret",
			Scopes:            []string{"openid"},
			DefaultTrustLevel: "substantial",
			Enabled:           true,
		},
		TenantID: "tenant-1",
	}

	tenant2Cfg := DynamicProviderConfig{
		ProviderConfig: ProviderConfig{
			ID:                "tenant2-provider",
			Name:              "Tenant 2 Provider",
			IssuerURL:         "https://tenant2.example.com",
			ClientID:          "tenant2-client",
			ClientSecret:      "tenant2-secret",
			Scopes:            []string{"openid"},
			DefaultTrustLevel: "high",
			Enabled:           true,
		},
		TenantID: "tenant-2",
	}

	_, err := service.RegisterProvider(ctx, tenant1Cfg)
	require.NoError(t, err)

	_, err = service.RegisterProvider(ctx, tenant2Cfg)
	require.NoError(t, err)

	t.Run("get_tenant_specific_provider", func(t *testing.T) {
		provider, err := service.GetProviderForTenant("tenant-1", "tenant1-provider")
		require.NoError(t, err)
		assert.Equal(t, "Tenant 1 Provider", provider.Name)

		// Tenant 2 should not have access to tenant 1's provider
		_, err = service.GetProviderForTenant("tenant-2", "tenant1-provider")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not available")
	})

	t.Run("list_providers_for_tenant", func(t *testing.T) {
		tenant1Providers := service.ListProvidersForTenant("tenant-1")
		assert.Len(t, tenant1Providers, 1)
		assert.Equal(t, "tenant1-provider", tenant1Providers[0].ID)

		tenant2Providers := service.ListProvidersForTenant("tenant-2")
		assert.Len(t, tenant2Providers, 1)
		assert.Equal(t, "tenant2-provider", tenant2Providers[0].ID)
	})

	t.Run("get_tenant_providers", func(t *testing.T) {
		tenant1IDs := service.GetTenantProviders("tenant-1")
		assert.Contains(t, tenant1IDs, "tenant1-provider")

		tenant2IDs := service.GetTenantProviders("tenant-2")
		assert.Contains(t, tenant2IDs, "tenant2-provider")
	})

	t.Run("validate_provider_access", func(t *testing.T) {
		err := service.ValidateProviderAccess("tenant-1", "tenant1-provider")
		assert.NoError(t, err)

		err = service.ValidateProviderAccess("tenant-2", "tenant1-provider")
		assert.Error(t, err)
	})
}

// TestDynamicProviderService_AssociateProvider tests tenant-provider associations.
func TestDynamicProviderService_AssociateProvider(t *testing.T) {
	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	ctx := context.Background()

	// Register a global provider
	cfg := DynamicProviderConfig{
		ProviderConfig: ProviderConfig{
			ID:                "global-provider",
			Name:              "Global Provider",
			IssuerURL:         "https://global.example.com",
			ClientID:          "global-client",
			ClientSecret:      "global-secret",
			Scopes:            []string{"openid"},
			DefaultTrustLevel: "substantial",
			Enabled:           true,
		},
	}

	_, err := service.RegisterProvider(ctx, cfg)
	require.NoError(t, err)

	t.Run("associate_provider_with_tenant", func(t *testing.T) {
		err := service.AssociateProviderWithTenant("tenant-alpha", "global-provider")
		require.NoError(t, err)

		// Verify association
		providers := service.GetTenantProviders("tenant-alpha")
		assert.Contains(t, providers, "global-provider")
	})

	t.Run("disassociate_provider_from_tenant", func(t *testing.T) {
		err := service.AssociateProviderWithTenant("tenant-beta", "global-provider")
		require.NoError(t, err)

		err = service.DisassociateProviderFromTenant("tenant-beta", "global-provider")
		require.NoError(t, err)

		// Verify disassociation
		providers := service.GetTenantProviders("tenant-beta")
		assert.NotContains(t, providers, "global-provider")
	})

	t.Run("associate_nonexistent_provider", func(t *testing.T) {
		err := service.AssociateProviderWithTenant("tenant-gamma", "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestDynamicProviderService_RefreshDiscovery tests discovery metadata refresh.
func TestDynamicProviderService_RefreshDiscovery(t *testing.T) {
	// Create mock discovery server with TLS
	callCount := 0
	var serverURL string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			callCount++
			discoveryDoc := DiscoveryMetadata{
				Issuer:        serverURL,
				TokenEndpoint: serverURL + "/token",
				JwksURI:       serverURL + fmt.Sprintf("/jwks-v%d", callCount),
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(discoveryDoc)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	service.httpClient = server.Client()
	ctx := context.Background()

	// Register provider with discovery
	cfg := DynamicProviderConfig{
		ProviderConfig: ProviderConfig{
			ID:                "refresh-test",
			Name:              "Refresh Test",
			IssuerURL:         serverURL,
			ClientID:          "client",
			ClientSecret:      "secret",
			Scopes:            []string{"openid"},
			DefaultTrustLevel: "low",
			Enabled:           true,
		},
		AutoDiscover: true,
	}

	_, err := service.RegisterProvider(ctx, cfg)
	require.NoError(t, err)

	t.Run("refresh_discovery_metadata", func(t *testing.T) {
		// Refresh discovery
		metadata, err := service.RefreshDiscoveryMetadata(ctx, "refresh-test")
		require.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Contains(t, metadata.JwksURI, "/jwks-v")

		// Verify updated in provider
		provider, err := registry.Get("refresh-test")
		require.NoError(t, err)
		assert.Contains(t, provider.Metadata, "discovery_refreshed_at")
	})
}

// TestDynamicProviderService_Statistics tests provider statistics.
func TestDynamicProviderService_Statistics(t *testing.T) {
	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	ctx := context.Background()

	// Register multiple providers
	for i := 1; i <= 5; i++ {
		cfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:                fmt.Sprintf("provider-%d", i),
				Name:              fmt.Sprintf("Provider %d", i),
				IssuerURL:         fmt.Sprintf("https://provider%d.example.com", i),
				ClientID:          fmt.Sprintf("client-%d", i),
				ClientSecret:      fmt.Sprintf("secret-%d", i),
				Scopes:            []string{"openid"},
				DefaultTrustLevel: "substantial",
				Enabled:           i <= 3, // First 3 enabled, last 2 disabled
			},
			TenantID: fmt.Sprintf("tenant-%d", (i-1)/2), // Distribute across tenants
		}

		_, err := service.RegisterProvider(ctx, cfg)
		require.NoError(t, err)
	}

	stats := service.GetProviderStatistics()
	assert.Equal(t, 5, stats["total_providers"])
	assert.Equal(t, 3, stats["enabled_providers"])
	assert.Equal(t, 2, stats["disabled_providers"])
	assert.True(t, stats["multi_tenant_enabled"].(bool))
}

// TestDynamicProviderService_ConcurrentOperations tests thread safety.
func TestDynamicProviderService_ConcurrentOperations(t *testing.T) {
	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	ctx := context.Background()

	done := make(chan bool, 30)

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		go func(idx int) {
			cfg := DynamicProviderConfig{
				ProviderConfig: ProviderConfig{
					ID:                fmt.Sprintf("concurrent-%d", idx),
					Name:              fmt.Sprintf("Concurrent %d", idx),
					IssuerURL:         fmt.Sprintf("https://concurrent%d.example.com", idx),
					ClientID:          fmt.Sprintf("client-%d", idx),
					ClientSecret:      fmt.Sprintf("secret-%d", idx),
					Scopes:            []string{"openid"},
					DefaultTrustLevel: "low",
					Enabled:           true,
				},
				TenantID: fmt.Sprintf("tenant-%d", idx%3),
			}

			_, _ = service.RegisterProvider(ctx, cfg)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(idx int) {
			_ = service.ListProvidersForTenant(fmt.Sprintf("tenant-%d", idx%3))
			_ = service.GetTenantProviders(fmt.Sprintf("tenant-%d", idx%3))
			done <- true
		}(i)
	}

	// Concurrent tenant associations
	for i := 0; i < 10; i++ {
		go func(idx int) {
			_ = service.AssociateProviderWithTenant(
				fmt.Sprintf("tenant-%d", idx%3),
				fmt.Sprintf("concurrent-%d", idx),
			)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 30; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Verify no data corruption
	stats := service.GetProviderStatistics()
	assert.GreaterOrEqual(t, stats["total_providers"].(int), 10)
}

// TestDynamicProviderService_GetRegisteredTenants tests tenant listing.
func TestDynamicProviderService_GetRegisteredTenants(t *testing.T) {
	registry := NewInMemoryProviderRegistry()
	service := NewDynamicProviderService(registry)
	ctx := context.Background()

	// Initially no tenants
	tenants := service.GetRegisteredTenants()
	assert.Empty(t, tenants)

	// Register providers for different tenants
	for _, tenant := range []string{"tenant-a", "tenant-b", "tenant-c"} {
		cfg := DynamicProviderConfig{
			ProviderConfig: ProviderConfig{
				ID:                fmt.Sprintf("%s-provider", tenant),
				Name:              fmt.Sprintf("%s Provider", tenant),
				IssuerURL:         fmt.Sprintf("https://%s.example.com", tenant),
				ClientID:          fmt.Sprintf("%s-client", tenant),
				ClientSecret:      fmt.Sprintf("%s-secret", tenant),
				Scopes:            []string{"openid"},
				DefaultTrustLevel: "substantial",
				Enabled:           true,
			},
			TenantID: tenant,
		}

		_, err := service.RegisterProvider(ctx, cfg)
		require.NoError(t, err)
	}

	// Verify tenants registered
	tenants = service.GetRegisteredTenants()
	assert.Len(t, tenants, 3)
	assert.Contains(t, tenants, "tenant-a")
	assert.Contains(t, tenants, "tenant-b")
	assert.Contains(t, tenants, "tenant-c")
}
