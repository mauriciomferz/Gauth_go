// Package resource provides types and functionality for resource management.
package resource

import (
	"time"
)

// Type represents a resource type
type Type string

const (
	// TypeAPI represents an API resource
	TypeAPI Type = "api"
	// TypeService represents a service resource
	TypeService Type = "service"
	// TypeEndpoint represents an endpoint resource
	TypeEndpoint Type = "endpoint"
	// TypeData represents a data resource
	TypeData Type = "data"
	// TypeFile represents a file resource
	TypeFile Type = "file"
)

// Status represents a resource status
type Status string

const (
	// StatusActive indicates an active resource
	StatusActive Status = "active"
	// StatusInactive indicates an inactive resource
	StatusInactive Status = "inactive"
	// StatusDeprecated indicates a deprecated resource
	StatusDeprecated Status = "deprecated"
	// StatusMaintenance indicates a resource under maintenance
	StatusMaintenance Status = "maintenance"
)

// AccessLevel represents a resource access level
type AccessLevel string

const (
	// AccessPublic indicates a public resource
	AccessPublic AccessLevel = "public"
	// AccessProtected indicates a protected resource
	AccessProtected AccessLevel = "protected"
	// AccessPrivate indicates a private resource
	AccessPrivate AccessLevel = "private"
	// AccessInternal indicates an internal resource
	AccessInternal AccessLevel = "internal"
)

// Resource represents a protected resource with strongly typed fields
type Resource struct {
	// Core fields
	ID          string `json:"id"`          // Unique resource identifier
	Type        Type   `json:"type"`        // Resource type
	Name        string `json:"name"`        // Resource name
	Description string `json:"description"` // Resource description
	Version     string `json:"version"`     // Resource version
	Status      Status `json:"status"`      // Current status

	// Access control
	OwnerID     string      `json:"owner_id"`     // Resource owner
	AccessLevel AccessLevel `json:"access_level"` // Access level
	Scopes      []string    `json:"scopes"`       // Required scopes

	// Routing
	Path    string   `json:"path"`    // Resource path/endpoint
	Methods []string `json:"methods"` // Allowed HTTP methods

	// Availability
	Region      string `json:"region"`      // Geographic region
	Environment string `json:"environment"` // Deployment environment

	// Rate limiting
	RateLimit *RateLimit `json:"rate_limit,omitempty"` // Rate limiting config

	// Timestamps
	CreatedAt time.Time `json:"created_at"` // Creation timestamp
	UpdatedAt time.Time `json:"updated_at"` // Last update timestamp

	// Additional metadata
	Tags     []string          `json:"tags"`     // Resource tags
	Metadata map[string]string `json:"metadata"` // Custom metadata
	Config   *ResourceConfig   `json:"config"`   // Configuration
}

// RateLimit represents rate limiting configuration
type RateLimit struct {
	// RequestsPerSecond is the maximum requests per second
	RequestsPerSecond int `json:"requests_per_second"`

	// BurstSize is the maximum burst size
	BurstSize int `json:"burst_size"`

	// WindowSize is the time window in seconds
	WindowSize int `json:"window_size"`
}

// NewResource creates a new resource
func NewResource(id string, typ Type) *Resource {
	now := time.Now()
	return &Resource{
		ID:          id,
		Type:        typ,
		Status:      StatusActive,
		AccessLevel: AccessProtected,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        make([]string, 0),
		Metadata:    make(map[string]string),
		Config:      NewResourceConfig(),
	}
}

// Validate validates the resource configuration
func (r *Resource) Validate() error {
	// Add validation logic
	return nil
}
