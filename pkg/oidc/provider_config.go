// Package oidc implements OpenID Connect support for GAuth.
// This file provides external provider configuration management.
package oidc

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// ProviderConfig represents an external OIDC provider configuration.
// This enables GAuth to act as an OIDC relying party for external providers
// like Google, Okta, or Azure AD.
type ProviderConfig struct {
	// ID is a unique identifier for this provider (e.g., "google", "okta-prod")
	ID string `json:"id" yaml:"id"`

	// Name is a human-readable name (e.g., "Google", "Okta Production")
	Name string `json:"name" yaml:"name"`

	// IssuerURL is the provider's issuer URL (must match iss claim in tokens)
	IssuerURL string `json:"issuer_url" yaml:"issuer_url"`

	// ClientID is the OAuth2/OIDC client identifier
	ClientID string `json:"client_id" yaml:"client_id"`

	// ClientSecret is the OAuth2/OIDC client secret
	// WARNING: Must be stored securely, never logged
	ClientSecret string `json:"client_secret,omitempty" yaml:"client_secret,omitempty"`

	// Scopes are the OIDC scopes to request (typically: openid, profile, email)
	Scopes []string `json:"scopes" yaml:"scopes"`

	// ClaimMappings maps provider-specific claims to GAuth claims
	// Example: {"sub": "user_id", "email": "email", "name": "full_name"}
	ClaimMappings map[string]string `json:"claim_mappings" yaml:"claim_mappings"`

	// TenantID is used for multi-tenant providers like Azure AD
	TenantID string `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty"`

	// DefaultTrustLevel is the trust level to use if not specified in token
	// Valid values: "low", "substantial", "high"
	DefaultTrustLevel string `json:"default_trust_level" yaml:"default_trust_level"`

	// TrustMapping maps provider-specific authentication context to GAuth trust levels
	// Example: {"mfa": "high", "password": "substantial"}
	TrustMapping map[string]string `json:"trust_mapping,omitempty" yaml:"trust_mapping,omitempty"`

	// Metadata contains provider-specific metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Enabled indicates whether this provider is active
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// Validate checks if the provider configuration is valid.
func (p *ProviderConfig) Validate() error {
	if p.ID == "" {
		return errors.New("provider ID is required")
	}

	if p.Name == "" {
		return errors.New("provider name is required")
	}

	if p.IssuerURL == "" {
		return errors.New("issuer URL is required")
	}

	// Validate issuer URL format
	issuerURL, err := url.Parse(p.IssuerURL)
	if err != nil {
		return fmt.Errorf("invalid issuer URL: %w", err)
	}

	if issuerURL.Scheme != "https" {
		return errors.New("issuer URL must use HTTPS")
	}

	if p.ClientID == "" {
		return errors.New("client ID is required")
	}

	if p.ClientSecret == "" {
		return errors.New("client secret is required")
	}

	// Validate scopes
	if len(p.Scopes) == 0 {
		return errors.New("at least one scope is required")
	}

	hasOpenIDScope := false
	for _, scope := range p.Scopes {
		if scope == "openid" {
			hasOpenIDScope = true
			break
		}
	}

	if !hasOpenIDScope {
		return errors.New("scopes must include 'openid'")
	}

	// Validate default trust level
	if p.DefaultTrustLevel != "" {
		validLevels := map[string]bool{"low": true, "substantial": true, "high": true}
		if !validLevels[p.DefaultTrustLevel] {
			return fmt.Errorf("invalid default trust level: %s (must be low, substantial, or high)", p.DefaultTrustLevel)
		}
	}

	return nil
}

// GetDiscoveryURL returns the OIDC discovery endpoint URL for this provider.
func (p *ProviderConfig) GetDiscoveryURL() string {
	issuer := strings.TrimSuffix(p.IssuerURL, "/")
	return issuer + "/.well-known/openid-configuration"
}

// ProviderRegistry manages external OIDC provider configurations.
type ProviderRegistry interface {
	// Register adds a new provider configuration
	Register(cfg ProviderConfig) error

	// Get retrieves a provider configuration by ID
	Get(id string) (*ProviderConfig, error)

	// List returns all registered provider configurations
	List() []ProviderConfig

	// ListEnabled returns only enabled provider configurations
	ListEnabled() []ProviderConfig

	// Update modifies an existing provider configuration
	Update(id string, cfg ProviderConfig) error

	// Delete removes a provider configuration
	Delete(id string) error

	// Enable activates a provider
	Enable(id string) error

	// Disable deactivates a provider
	Disable(id string) error
}

// InMemoryProviderRegistry implements ProviderRegistry using in-memory storage.
// This is suitable for development and testing. Production deployments should
// use a persistent storage backend.
type InMemoryProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]ProviderConfig
}

// NewInMemoryProviderRegistry creates a new in-memory provider registry.
func NewInMemoryProviderRegistry() *InMemoryProviderRegistry {
	return &InMemoryProviderRegistry{
		providers: make(map[string]ProviderConfig),
	}
}

// Register adds a new provider configuration.
func (r *InMemoryProviderRegistry) Register(cfg ProviderConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid provider configuration: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[cfg.ID]; exists {
		return fmt.Errorf("provider with ID %s already exists", cfg.ID)
	}

	r.providers[cfg.ID] = cfg
	return nil
}

// Get retrieves a provider configuration by ID.
func (r *InMemoryProviderRegistry) Get(id string) (*ProviderConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, exists := r.providers[id]
	if !exists {
		return nil, fmt.Errorf("provider with ID %s not found", id)
	}

	// Return a copy to prevent external modifications
	cfgCopy := cfg
	return &cfgCopy, nil
}

// List returns all registered provider configurations.
func (r *InMemoryProviderRegistry) List() []ProviderConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	configs := make([]ProviderConfig, 0, len(r.providers))
	for _, cfg := range r.providers {
		configs = append(configs, cfg)
	}

	return configs
}

// ListEnabled returns only enabled provider configurations.
func (r *InMemoryProviderRegistry) ListEnabled() []ProviderConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	configs := make([]ProviderConfig, 0)
	for _, cfg := range r.providers {
		if cfg.Enabled {
			configs = append(configs, cfg)
		}
	}

	return configs
}

// Update modifies an existing provider configuration.
func (r *InMemoryProviderRegistry) Update(id string, cfg ProviderConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid provider configuration: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[id]; !exists {
		return fmt.Errorf("provider with ID %s not found", id)
	}

	// Ensure ID doesn't change
	cfg.ID = id
	r.providers[id] = cfg
	return nil
}

// Delete removes a provider configuration.
func (r *InMemoryProviderRegistry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[id]; !exists {
		return fmt.Errorf("provider with ID %s not found", id)
	}

	delete(r.providers, id)
	return nil
}

// Enable activates a provider.
func (r *InMemoryProviderRegistry) Enable(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, exists := r.providers[id]
	if !exists {
		return fmt.Errorf("provider with ID %s not found", id)
	}

	cfg.Enabled = true
	r.providers[id] = cfg
	return nil
}

// Disable deactivates a provider.
func (r *InMemoryProviderRegistry) Disable(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, exists := r.providers[id]
	if !exists {
		return fmt.Errorf("provider with ID %s not found", id)
	}

	cfg.Enabled = false
	r.providers[id] = cfg
	return nil
}
