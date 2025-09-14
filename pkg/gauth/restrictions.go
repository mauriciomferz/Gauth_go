package gauth

import "time"

// Restriction defines constraints and requirements for authentication or authorization
type Restriction struct {
	// Type of restriction (e.g., "ip", "time", "rate", etc.)
	Type string `json:"type"`

	// Value of the restriction (e.g., IP range, time window, etc.)
	// NOTE: interface{} is used here only for backend plugin/migration flexibility.
	// All public APIs use type-safe alternatives. Do not expose in new APIs.
	Value interface{} `json:"value"`

	// ValidFrom defines when the restriction starts being active
	ValidFrom *time.Time `json:"valid_from,omitempty"`

	// ValidUntil defines when the restriction expires
	ValidUntil *time.Time `json:"valid_until,omitempty"`

	// Description provides human-readable information about the restriction
	Description string `json:"description,omitempty"`

	// Enforced indicates whether the restriction is actively enforced
	Enforced bool `json:"enforced"`

	// StrictMode determines if violation of this restriction should fail hard
	StrictMode bool `json:"strict_mode"`

	// Properties holds additional type-specific configuration
	Properties *Properties `json:"properties,omitempty"`
}
