// Package gauth provides public resource types and interfaces.
package gauth

import "fmt"

// ResourceType represents the type of a resource
type ResourceType string

const (
	// ResourceTypeAPI represents an API resource
	ResourceTypeAPI ResourceType = "api"
	// ResourceTypeService represents a service resource
	ResourceTypeService ResourceType = "service"
	// ResourceTypeEndpoint represents an endpoint resource
	ResourceTypeEndpoint ResourceType = "endpoint"
	// ResourceTypeData represents a data resource
	ResourceTypeData ResourceType = "data"
	// ResourceTypeFile represents a file resource
	ResourceTypeFile ResourceType = "file"
)

// ResourceStatus represents the status of a resource
type ResourceStatus string

const (
	// ResourceStatusActive indicates an active resource
	ResourceStatusActive ResourceStatus = "active"
	// ResourceStatusInactive indicates an inactive resource
	ResourceStatusInactive ResourceStatus = "inactive"
	// ResourceStatusDeprecated indicates a deprecated resource
	ResourceStatusDeprecated ResourceStatus = "deprecated"
	// ResourceStatusMaintenance indicates a resource under maintenance
	ResourceStatusMaintenance ResourceStatus = "maintenance"
)

// ResourceAccess represents resource access levels
type ResourceAccess string

const (
	// ResourceAccessPublic indicates a public resource
	ResourceAccessPublic ResourceAccess = "public"
	// ResourceAccessProtected indicates a protected resource
	ResourceAccessProtected ResourceAccess = "protected"
	// ResourceAccessPrivate indicates a private resource
	ResourceAccessPrivate ResourceAccess = "private"
	// ResourceAccessInternal indicates an internal resource
	ResourceAccessInternal ResourceAccess = "internal"
)

// ResourceConfig represents resource configuration
type ResourceConfig struct {
	// Core settings
	ID          string         `json:"id"`          // Resource identifier
	Type        ResourceType   `json:"type"`        // Resource type
	Name        string         `json:"name"`        // Resource name
	Description string         `json:"description"` // Resource description
	Version     string         `json:"version"`     // Resource version
	Status      ResourceStatus `json:"status"`      // Resource status

	// Access control
	OwnerID     string         `json:"owner_id"`     // Resource owner
	AccessLevel ResourceAccess `json:"access_level"` // Access level
	Scopes      []string       `json:"scopes"`       // Required scopes

	// Routing
	Path    string   `json:"path"`    // Resource path
	Methods []string `json:"methods"` // Allowed methods

	// Deployment
	Region      string `json:"region"`      // Geographic region
	Environment string `json:"environment"` // Deploy environment

	// Rate limiting
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"` // Rate limits

	// Extensions
	Tags     []string          `json:"tags"`     // Resource tags
	Metadata map[string]string `json:"metadata"` // Custom metadata
}

// NewResource creates a new resource with the given ID and type
func NewResource(id string, typ ResourceType) *ResourceConfig {
	return &ResourceConfig{
		ID:          id,
		Type:        typ,
		Status:      ResourceStatusActive,
		AccessLevel: ResourceAccessProtected,
		Methods:     make([]string, 0),
		Tags:        make([]string, 0),
		Metadata:    make(map[string]string),
	}
}

// Validate checks if the resource configuration is valid
func (r *ResourceConfig) Validate() error {
	// Simple validation for example purposes
	if r.ID == "" {
		return fmt.Errorf("resource ID is required")
	}
	if r.Name == "" {
		return fmt.Errorf("resource name is required")
	}
	return nil
}

// IsActive returns true if the resource is active
func (r *ResourceConfig) IsActive() bool {
	return r.Status == ResourceStatusActive
}

// IsPublic returns true if the resource is public
func (r *ResourceConfig) IsPublic() bool {
	return r.AccessLevel == ResourceAccessPublic
}

// RequiresScope checks if the resource requires a specific scope
func (r *ResourceConfig) RequiresScope(scope string) bool {
	for _, s := range r.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// RequiresAnyScope checks if the resource requires any of the given scopes
func (r *ResourceConfig) RequiresAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if r.RequiresScope(scope) {
			return true
		}
	}
	return false
}

// RequiresAllScopes checks if the resource requires all given scopes
func (r *ResourceConfig) RequiresAllScopes(scopes ...string) bool {
	for _, scope := range scopes {
		if !r.RequiresScope(scope) {
			return false
		}
	}
	return true
}

// AllowsMethod checks if the resource allows a specific HTTP method
func (r *ResourceConfig) AllowsMethod(method string) bool {
	for _, m := range r.Methods {
		if m == method {
			return true
		}
	}
	return false
}
