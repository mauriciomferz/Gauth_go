package gauth

import (
	"github.com/Gimel-Foundation/gauth/internal/resource"
)

// RateLimitConfig defines rate limiting configuration for resources.
type RateLimitConfig struct {
	RequestsPerSecond int `json:"requests_per_second"` // Maximum requests per second
	BurstSize         int `json:"burst_size"`          // Maximum burst size
	WindowSize        int `json:"window_size"`         // Time window in seconds
}

// ResourceType represents the type of a resource
type ResourceType string

const (
	// ResourceTypeAPI represents an API resource
	ResourceTypeAPI = ResourceType(resource.TypeAPI)
	// ResourceTypeService represents a service resource
	ResourceTypeService = ResourceType(resource.TypeService)
	// ResourceTypeEndpoint represents an endpoint resource
	ResourceTypeEndpoint = ResourceType(resource.TypeEndpoint)
	// ResourceTypeData represents a data resource
	ResourceTypeData = ResourceType(resource.TypeData)
	// ResourceTypeFile represents a file resource
	ResourceTypeFile = ResourceType(resource.TypeFile)
)

// ResourceStatus represents the status of a resource
type ResourceStatus string

const (
	// ResourceStatusActive indicates an active resource
	ResourceStatusActive = ResourceStatus(resource.StatusActive)
	// ResourceStatusInactive indicates an inactive resource
	ResourceStatusInactive = ResourceStatus(resource.StatusInactive)
	// ResourceStatusDeprecated indicates a deprecated resource
	ResourceStatusDeprecated = ResourceStatus(resource.StatusDeprecated)
	// ResourceStatusMaintenance indicates a resource under maintenance
	ResourceStatusMaintenance = ResourceStatus(resource.StatusMaintenance)
)

// ResourceAccess represents resource access levels
type ResourceAccess string

const (
	// ResourceAccessPublic indicates a public resource
	ResourceAccessPublic = ResourceAccess(resource.AccessPublic)
	// ResourceAccessProtected indicates a protected resource
	ResourceAccessProtected = ResourceAccess(resource.AccessProtected)
	// ResourceAccessPrivate indicates a private resource
	ResourceAccessPrivate = ResourceAccess(resource.AccessPrivate)
	// ResourceAccessInternal indicates an internal resource
	ResourceAccessInternal = ResourceAccess(resource.AccessInternal)
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
	// Convert to internal type and validate
	internal := &resource.Resource{
		ID:          r.ID,
		Type:        resource.Type(r.Type),
		Name:        r.Name,
		Description: r.Description,
		Version:     r.Version,
		Status:      resource.Status(r.Status),
		OwnerID:     r.OwnerID,
		AccessLevel: resource.AccessLevel(r.AccessLevel),
		Scopes:      r.Scopes,
		Path:        r.Path,
		Methods:     r.Methods,
		Region:      r.Region,
		Environment: r.Environment,
		Tags:        r.Tags,
		Metadata:    r.Metadata,
	}

	if r.RateLimit != nil {
		internal.RateLimit = &resource.RateLimit{
			RequestsPerSecond: r.RateLimit.RequestsPerSecond,
			BurstSize:         r.RateLimit.BurstSize,
			WindowSize:        r.RateLimit.WindowSize,
		}
	}

	return internal.Validate()
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
