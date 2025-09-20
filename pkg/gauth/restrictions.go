package gauth

import "time"

// Restriction defines constraints and requirements for authentication or authorization.
//
// Use the Properties field for all new restriction logic. The Value field is deprecated and exists only for legacy migration.
//
// Example usage:
//   // Create a time-based restriction
//   start := time.Now()
//   end := start.Add(2 * time.Hour)
//   r := Restriction{
//       Type:       "time",
//       Enforced:   true,
//       Properties: NewProperties(),
//   }
//   r.Properties.SetTime("start", start)
//   r.Properties.SetTime("end", end)
//
// See restrictions_util.go for helper constructors.
type Restriction struct {
	// Type of restriction (e.g., "ip", "time", "rate", etc.)
	Type string `json:"type"`

	// Deprecated: Value is for legacy plugin/migration flexibility only. Use Properties for all new code.
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

	// Properties holds additional type-specific configuration (preferred for all new code)
	Properties *Properties `json:"properties,omitempty"`
}
