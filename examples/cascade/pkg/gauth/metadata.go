// Package gauth provides core types and interfaces for the GAuth authentication system.
package gauth

import "time"

// MetadataHolder is an interface for types that can provide metadata.
type MetadataHolder interface {
	// GetMetadata returns the metadata map
	GetMetadata() map[string]string
	// ValidateMetadata validates the metadata contents
	ValidateMetadata() error
}

// TransactionMetadata represents strongly typed metadata for transactions.
type TransactionMetadata struct {
	// Currency represents the transaction currency code (e.g., USD, EUR)
	Currency string `json:"currency,omitempty"`
	// Amount represents the transaction amount
	Amount float64 `json:"amount,omitempty"`
	// Method represents the transaction method (e.g., credit_card, transfer)
	Method string `json:"method,omitempty"`
	// Source represents the transaction source (e.g., account ID)
	Source string `json:"source,omitempty"`
	// Destination represents the transaction destination (e.g., account ID)
	Destination string `json:"destination,omitempty"`
	// Category represents the transaction category
	Category string `json:"category,omitempty"`
	// Timestamp represents when the transaction was initiated
	Timestamp time.Time `json:"timestamp,omitempty"`
	// Status represents the transaction status
	Status string `json:"status,omitempty"`
	// Reference represents an external reference number
	Reference string `json:"reference,omitempty"`
	// Description provides additional transaction details
	Description string `json:"description,omitempty"`
}

// TokenMetadata represents strongly typed metadata for tokens.
type TokenMetadata struct {
	// Issuer represents the token issuer
	Issuer string `json:"issuer,omitempty"`
	// Subject represents the token subject
	Subject string `json:"subject,omitempty"`
	// Audience represents the intended token audience
	Audience []string `json:"audience,omitempty"`
	// IssuedAt represents token issuance time
	IssuedAt time.Time `json:"issued_at,omitempty"`
	// NotBefore represents token validity start time
	NotBefore time.Time `json:"not_before,omitempty"`
	// ExpiresAt represents token expiration time
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	// JTI represents the unique token identifier
	JTI string `json:"jti,omitempty"`
	// SessionID represents associated session identifier
	SessionID string `json:"session_id,omitempty"`
	// DeviceID represents associated device identifier
	DeviceID string `json:"device_id,omitempty"`
}

// AuditMetadata represents strongly typed metadata for audit events.
type AuditMetadata struct {
	// Actor represents who performed the action
	Actor string `json:"actor,omitempty"`
	// Action represents what was done
	Action string `json:"action,omitempty"`
	// Resource represents what was acted upon
	Resource string `json:"resource,omitempty"`
	// Timestamp represents when the action occurred
	Timestamp time.Time `json:"timestamp,omitempty"`
	// Status represents the action outcome
	Status string `json:"status,omitempty"`
	// IPAddress represents the source IP
	IPAddress string `json:"ip_address,omitempty"`
	// UserAgent represents the client user agent
	UserAgent string `json:"user_agent,omitempty"`
	// Location represents the geographic location
	Location string `json:"location,omitempty"`
	// Reason provides context for the action
	Reason string `json:"reason,omitempty"`
}

// ResourceMetadata represents strongly typed metadata for resources.
type ResourceMetadata struct {
	// Owner represents the resource owner
	Owner string `json:"owner,omitempty"`
	// Type represents the resource type
	Type string `json:"type,omitempty"`
	// Tags represents resource tags
	Tags []string `json:"tags,omitempty"`
	// CreatedAt represents creation time
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt represents last update time
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	// Version represents resource version
	Version string `json:"version,omitempty"`
	// Status represents resource status
	Status string `json:"status,omitempty"`
	// Region represents geographic region
	Region string `json:"region,omitempty"`
	// Environment represents deployment environment
	Environment string `json:"environment,omitempty"`
}

// ExtensibleMetadata allows for custom metadata fields while
// maintaining type safety for known fields.
type ExtensibleMetadata struct {
	// Known fields with proper types
	Actor     string    `json:"actor,omitempty"`
	Action    string    `json:"action,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Status    string    `json:"status,omitempty"`

	// Custom fields can be added here
	Custom map[string]string `json:"custom,omitempty"`
}

// Validate ensures the metadata meets requirements
func (m *TransactionMetadata) Validate() error {
	// Add validation logic
	return nil
}

func (m *TokenMetadata) Validate() error {
	// Add validation logic
	return nil
}

func (m *AuditMetadata) Validate() error {
	// Add validation logic
	return nil
}

func (m *ResourceMetadata) Validate() error {
	// Add validation logic
	return nil
}

func (m *ExtensibleMetadata) Validate() error {
	// Add validation logic
	return nil
}
