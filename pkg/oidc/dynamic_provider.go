// Package oidc - Dynamic Provider Management
// Implements runtime provider registration with discovery and multi-tenant support
package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// DynamicProviderService manages runtime provider registration and discovery.
type DynamicProviderService struct {
	registry        ProviderRegistry
	httpClient      *http.Client
	mu              sync.RWMutex
	tenantProviders map[string]map[string]bool    // tenant -> provider IDs
	discoveryMeta   map[string]*DiscoveryMetadata // provider ID -> discovery metadata
}

// DynamicProviderConfig extends ProviderConfig with dynamic registration options.
type DynamicProviderConfig struct {
	ProviderConfig

	// TenantID associates this provider with a specific tenant
	// Empty string means provider is available to all tenants
	TenantID string `json:"tenant_id,omitempty"`

	// AutoDiscover enables automatic discovery of provider metadata
	AutoDiscover bool `json:"auto_discover,omitempty"`

	// DiscoveryURL overrides the default discovery endpoint
	DiscoveryURL string `json:"discovery_url,omitempty"`

	// RefreshInterval for discovery metadata (0 = no refresh)
	RefreshInterval time.Duration `json:"refresh_interval,omitempty"`

	// Tags for organizing and filtering providers
	Tags []string `json:"tags,omitempty"`
}

// DiscoveryMetadata represents OIDC discovery document metadata.
type DiscoveryMetadata struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserinfoEndpoint                 string   `json:"userinfo_endpoint,omitempty"`
	JwksURI                          string   `json:"jwks_uri"`
	ScopesSupported                  []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported           []string `json:"response_types_supported,omitempty"`
	SubjectTypesSupported            []string `json:"subject_types_supported,omitempty"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported,omitempty"`
}

// ProviderRegistrationResult contains the result of a provider registration.
type ProviderRegistrationResult struct {
	ProviderID       string             `json:"provider_id"`
	Success          bool               `json:"success"`
	DiscoveryFetched bool               `json:"discovery_fetched,omitempty"`
	Metadata         *DiscoveryMetadata `json:"metadata,omitempty"`
	Error            string             `json:"error,omitempty"`
	RegisteredAt     time.Time          `json:"registered_at"`
}

// NewDynamicProviderService creates a new dynamic provider service.
func NewDynamicProviderService(registry ProviderRegistry) *DynamicProviderService {
	return &DynamicProviderService{
		registry:        registry,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		tenantProviders: make(map[string]map[string]bool),
		discoveryMeta:   make(map[string]*DiscoveryMetadata),
	}
}

// RegisterProvider registers a new provider with optional auto-discovery.
func (s *DynamicProviderService) RegisterProvider(ctx context.Context, cfg DynamicProviderConfig) (*ProviderRegistrationResult, error) {
	result := &ProviderRegistrationResult{
		ProviderID:   cfg.ID,
		RegisteredAt: time.Now(),
	}

	// Auto-discover metadata if requested
	if cfg.AutoDiscover {
		metadata, err := s.fetchDiscoveryMetadata(ctx, cfg)
		if err != nil {
			result.Error = fmt.Sprintf("discovery failed: %v", err)
			// Continue with registration even if discovery fails
		} else {
			result.DiscoveryFetched = true
			result.Metadata = metadata

			// Update configuration with discovered metadata
			if cfg.Metadata == nil {
				cfg.Metadata = make(map[string]interface{})
			}
			cfg.Metadata["discovery"] = metadata
			cfg.Metadata["jwks_uri"] = metadata.JwksURI
			cfg.Metadata["authorization_endpoint"] = metadata.AuthorizationEndpoint
			cfg.Metadata["token_endpoint"] = metadata.TokenEndpoint

			// Store the discovery metadata
			s.mu.Lock()
			s.discoveryMeta[cfg.ID] = metadata
			s.mu.Unlock()
		}
	}

	// Register the provider
	if err := s.registry.Register(cfg.ProviderConfig); err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	// Associate with tenant if specified
	if cfg.TenantID != "" {
		s.associateProviderWithTenant(cfg.TenantID, cfg.ID)
	}

	result.Success = true
	return result, nil
}

// UpdateProvider updates an existing provider configuration.
func (s *DynamicProviderService) UpdateProvider(ctx context.Context, providerID string, cfg DynamicProviderConfig) error {
	// Re-fetch discovery if requested
	if cfg.AutoDiscover {
		metadata, err := s.fetchDiscoveryMetadata(ctx, cfg)
		if err != nil {
			return fmt.Errorf("discovery failed during update: %w", err)
		}

		// Update metadata
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata["discovery"] = metadata
		cfg.Metadata["jwks_uri"] = metadata.JwksURI

		// Store the discovery metadata
		s.mu.Lock()
		s.discoveryMeta[providerID] = metadata
		s.mu.Unlock()
	}

	return s.registry.Update(providerID, cfg.ProviderConfig)
}

// DeleteProvider removes a provider and cleans up tenant associations.
func (s *DynamicProviderService) DeleteProvider(ctx context.Context, providerID string) error {
	// Remove from registry
	if err := s.registry.Delete(providerID); err != nil {
		return err
	}

	// Clean up tenant associations
	s.mu.Lock()
	defer s.mu.Unlock()

	for tenant, providers := range s.tenantProviders {
		delete(providers, providerID)
		if len(providers) == 0 {
			delete(s.tenantProviders, tenant)
		}
	}

	return nil
}

// GetProviderForTenant retrieves a provider configuration for a specific tenant.
func (s *DynamicProviderService) GetProviderForTenant(tenantID string, providerID string) (*ProviderConfig, error) {
	// Check if provider is associated with this tenant
	s.mu.RLock()
	providers, tenantExists := s.tenantProviders[tenantID]
	s.mu.RUnlock()

	if tenantExists {
		if !providers[providerID] {
			return nil, fmt.Errorf("provider %s not available for tenant %s", providerID, tenantID)
		}
	}

	// Get provider from registry
	return s.registry.Get(providerID)
}

// ListProvidersForTenant returns all providers available to a tenant.
func (s *DynamicProviderService) ListProvidersForTenant(tenantID string) []ProviderConfig {
	s.mu.RLock()
	providerIDs, tenantExists := s.tenantProviders[tenantID]
	s.mu.RUnlock()

	if !tenantExists {
		// Return all enabled global providers
		return s.registry.ListEnabled()
	}

	// Return tenant-specific providers
	var configs []ProviderConfig
	for providerID := range providerIDs {
		if cfg, err := s.registry.Get(providerID); err == nil {
			if cfg.Enabled {
				configs = append(configs, *cfg)
			}
		}
	}

	return configs
}

// AssociateProviderWithTenant associates a provider with a specific tenant.
func (s *DynamicProviderService) AssociateProviderWithTenant(tenantID string, providerID string) error {
	// Verify provider exists
	if _, err := s.registry.Get(providerID); err != nil {
		return fmt.Errorf("provider %s not found: %w", providerID, err)
	}

	s.associateProviderWithTenant(tenantID, providerID)
	return nil
}

// DisassociateProviderFromTenant removes the association between a provider and tenant.
func (s *DynamicProviderService) DisassociateProviderFromTenant(tenantID string, providerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if providers, exists := s.tenantProviders[tenantID]; exists {
		delete(providers, providerID)
		if len(providers) == 0 {
			delete(s.tenantProviders, tenantID)
		}
	}

	return nil
}

// RefreshDiscoveryMetadata refreshes the discovery metadata for a provider.
func (s *DynamicProviderService) RefreshDiscoveryMetadata(ctx context.Context, providerID string) (*DiscoveryMetadata, error) {
	// Get provider configuration
	cfg, err := s.registry.Get(providerID)
	if err != nil {
		return nil, err
	}

	// Fetch discovery metadata
	dynamicCfg := DynamicProviderConfig{
		ProviderConfig: *cfg,
		AutoDiscover:   true,
	}

	metadata, err := s.fetchDiscoveryMetadata(ctx, dynamicCfg)
	if err != nil {
		return nil, err
	}

	// Update provider metadata
	if cfg.Metadata == nil {
		cfg.Metadata = make(map[string]interface{})
	}
	cfg.Metadata["discovery"] = metadata
	cfg.Metadata["discovery_refreshed_at"] = time.Now()

	if err := s.registry.Update(providerID, *cfg); err != nil {
		return nil, fmt.Errorf("failed to update provider after discovery refresh: %w", err)
	}

	// Store the discovery metadata
	s.mu.Lock()
	s.discoveryMeta[providerID] = metadata
	s.mu.Unlock()

	return metadata, nil
}

// GetTenantProviders returns the provider IDs associated with a tenant.
func (s *DynamicProviderService) GetTenantProviders(tenantID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers, exists := s.tenantProviders[tenantID]
	if !exists {
		return []string{}
	}

	providerIDs := make([]string, 0, len(providers))
	for providerID := range providers {
		providerIDs = append(providerIDs, providerID)
	}

	return providerIDs
}

// GetRegisteredTenants returns all tenants with provider associations.
func (s *DynamicProviderService) GetRegisteredTenants() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tenants := make([]string, 0, len(s.tenantProviders))
	for tenant := range s.tenantProviders {
		tenants = append(tenants, tenant)
	}

	return tenants
}

// associateProviderWithTenant is the internal method to associate a provider with a tenant.
func (s *DynamicProviderService) associateProviderWithTenant(tenantID string, providerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tenantProviders[tenantID]; !exists {
		s.tenantProviders[tenantID] = make(map[string]bool)
	}

	s.tenantProviders[tenantID][providerID] = true
}

// fetchDiscoveryMetadata fetches OIDC discovery metadata from the provider.
func (s *DynamicProviderService) fetchDiscoveryMetadata(ctx context.Context, cfg DynamicProviderConfig) (*DiscoveryMetadata, error) {
	// Determine discovery URL
	discoveryURL := cfg.DiscoveryURL
	if discoveryURL == "" {
		discoveryURL = cfg.GetDiscoveryURL()
	}

	// Check if we already have discovery metadata
	s.mu.RLock()
	if existing, ok := s.discoveryMeta[cfg.ID]; ok {
		s.mu.RUnlock()
		return existing, nil
	}
	s.mu.RUnlock()

	// Fetch discovery document
	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned status %d", resp.StatusCode)
	}

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read discovery response: %w", err)
	}

	var metadata DiscoveryMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	// Validate issuer matches
	if metadata.Issuer != cfg.IssuerURL {
		return nil, fmt.Errorf("discovery issuer %s does not match configured issuer %s", metadata.Issuer, cfg.IssuerURL)
	}

	return &metadata, nil
}

// ValidateProviderAccess checks if a tenant has access to a provider.
func (s *DynamicProviderService) ValidateProviderAccess(tenantID string, providerID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If no tenant-specific providers are configured, allow access to all
	if len(s.tenantProviders) == 0 {
		return nil
	}

	// Check tenant-specific providers
	providers, tenantExists := s.tenantProviders[tenantID]
	if !tenantExists {
		return fmt.Errorf("tenant %s has no provider associations", tenantID)
	}

	if !providers[providerID] {
		return fmt.Errorf("tenant %s does not have access to provider %s", tenantID, providerID)
	}

	return nil
}

// GetProviderStatistics returns statistics about registered providers.
func (s *DynamicProviderService) GetProviderStatistics() map[string]interface{} {
	allProviders := s.registry.List()
	enabledProviders := s.registry.ListEnabled()

	s.mu.RLock()
	tenantCount := len(s.tenantProviders)
	s.mu.RUnlock()

	return map[string]interface{}{
		"total_providers":      len(allProviders),
		"enabled_providers":    len(enabledProviders),
		"disabled_providers":   len(allProviders) - len(enabledProviders),
		"tenant_count":         tenantCount,
		"multi_tenant_enabled": tenantCount > 0,
	}
}
