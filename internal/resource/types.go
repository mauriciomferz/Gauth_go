// Package resource provides types and functionality for resource management.
package resource

import (
	"fmt"
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

// IsActive checks if the resource is active
func (r *Resource) IsActive() bool {
	return r.Status == StatusActive
}

// IsPublic checks if the resource is public
func (r *Resource) IsPublic() bool {
	return r.AccessLevel == AccessPublic
}

// RequiresScope checks if the resource requires a specific scope
func (r *Resource) RequiresScope(scope string) bool {
	for _, s := range r.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// RequiresAnyScope checks if the resource requires any of the given scopes
func (r *Resource) RequiresAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if r.RequiresScope(scope) {
			return true
		}
	}
	return false
}

// RequiresAllScopes checks if the resource requires all given scopes
func (r *Resource) RequiresAllScopes(scopes ...string) bool {
	for _, scope := range scopes {
		if !r.RequiresScope(scope) {
			return false
		}
	}
	return true
}

// AllowsMethod checks if the resource allows a specific HTTP method
func (r *Resource) AllowsMethod(method string) bool {
	for _, m := range r.Methods {
		if m == method {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the resource
func (r *Resource) AddTag(tag string) {
	for _, t := range r.Tags {
		if t == tag {
			return
		}
	}
	r.Tags = append(r.Tags, tag)
	r.UpdatedAt = time.Now()
}

// RemoveTag removes a tag from the resource
func (r *Resource) RemoveTag(tag string) {
	for i, t := range r.Tags {
		if t == tag {
			r.Tags = append(r.Tags[:i], r.Tags[i+1:]...)
			r.UpdatedAt = time.Now()
			return
		}
	}
}

// SetMetadata sets a metadata value
func (r *Resource) SetMetadata(key, value string) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]string)
	}
	r.Metadata[key] = value
	r.UpdatedAt = time.Now()
}

// GetMetadata gets a metadata value
func (r *Resource) GetMetadata(key string) string {
	if r.Metadata == nil {
		return ""
	}
	return r.Metadata[key]
}

// SetConfig sets a configuration value
func (r *Resource) SetConfig(key string, value interface{}) {
	if r.Config == nil {
		r.Config = NewResourceConfig()
	}

	// Determine the type of value and store it appropriately
	switch v := value.(type) {
	case string:
		r.Config.SetString(key, v)
	case int:
		r.Config.SetInt(key, v)
	case float64:
		r.Config.SetFloat(key, v)
	case bool:
		r.Config.SetBool(key, v)
	case []interface{}:
		r.Config.SetSlice(key, v)
	case map[string]interface{}:
		r.Config.SetMap(key, v)
	default:
		// For any other type, convert to string if possible
		r.Config.SetString(key, fmt.Sprintf("%v", value))
	}

	r.UpdatedAt = time.Now()
}

// GetConfig gets a configuration value
func (r *Resource) GetConfig(key string) interface{} {
	if r.Config == nil {
		return nil
	}

	// Try each type in sequence
	if val, ok := r.Config.GetString(key); ok {
		return val
	}
	if val, ok := r.Config.GetInt(key); ok {
		return val
	}
	if val, ok := r.Config.GetFloat(key); ok {
		return val
	}
	if val, ok := r.Config.GetBool(key); ok {
		return val
	}
	if val, ok := r.Config.GetMap(key); ok {
		return val
	}
	if val, ok := r.Config.GetSlice(key); ok {
		return val
	}

	return nil
}

// GetConfigString gets a string configuration value
func (r *Resource) GetConfigString(key string) (string, bool) {
	if r.Config == nil {
		return "", false
	}
	return r.Config.GetString(key)
}

// GetConfigInt gets an int configuration value
func (r *Resource) GetConfigInt(key string) (int, bool) {
	if r.Config == nil {
		return 0, false
	}
	return r.Config.GetInt(key)
}

// GetConfigFloat gets a float configuration value
func (r *Resource) GetConfigFloat(key string) (float64, bool) {
	if r.Config == nil {
		return 0, false
	}
	return r.Config.GetFloat(key)
}

// GetConfigBool gets a bool configuration value
func (r *Resource) GetConfigBool(key string) (bool, bool) {
	if r.Config == nil {
		return false, false
	}
	return r.Config.GetBool(key)
}

// SetConfigString sets a string configuration value
func (r *Resource) SetConfigString(key string, value string) {
	if r.Config == nil {
		r.Config = NewResourceConfig()
	}
	r.Config.SetString(key, value)
	r.UpdatedAt = time.Now()
}

// SetConfigInt sets an int configuration value
func (r *Resource) SetConfigInt(key string, value int) {
	if r.Config == nil {
		r.Config = NewResourceConfig()
	}
	r.Config.SetInt(key, value)
	r.UpdatedAt = time.Now()
}

// SetConfigFloat sets a float configuration value
func (r *Resource) SetConfigFloat(key string, value float64) {
	if r.Config == nil {
		r.Config = NewResourceConfig()
	}
	r.Config.SetFloat(key, value)
	r.UpdatedAt = time.Now()
}

// SetConfigBool sets a bool configuration value
func (r *Resource) SetConfigBool(key string, value bool) {
	if r.Config == nil {
		r.Config = NewResourceConfig()
	}
	r.Config.SetBool(key, value)
	r.UpdatedAt = time.Now()
}
