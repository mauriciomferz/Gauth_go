// Package transaction provides internal transaction handling functionality.
package transaction

import "time"

// Type represents a transaction type
type Type string

const (
	// TypePayment represents a payment transaction
	TypePayment Type = "payment"
	// TypeTransfer represents a transfer transaction
	TypeTransfer Type = "transfer"
	// TypeAuthorization represents an authorization transaction
	TypeAuthorization Type = "authorization"
	// TypeValidation represents a validation transaction
	TypeValidation Type = "validation"
)

// Status represents a transaction status
type Status string

const (
	// StatusPending indicates a pending transaction
	StatusPending Status = "pending"
	// StatusSuccess indicates a successful transaction
	StatusSuccess Status = "success"
	// StatusFailed indicates a failed transaction
	StatusFailed Status = "failed"
	// StatusCancelled indicates a cancelled transaction
	StatusCancelled Status = "cancelled"
)

// Details represents internal transaction details
type Details struct {
	// Core fields
	ID         string   `json:"id"`          // Unique transaction identifier
	Type       Type     `json:"type"`        // Type of transaction
	Status     Status   `json:"status"`      // Current status
	ClientID   string   `json:"client_id"`   // Client initiating the transaction
	ResourceID string   `json:"resource_id"` // Resource being accessed
	Scopes     []string `json:"scopes"`      // Required scopes

	// Financial details
	Amount   float64 `json:"amount"`   // Transaction amount
	Currency string  `json:"currency"` // Transaction currency

	// Transaction context
	Source      string `json:"source"`      // Source of transaction
	Destination string `json:"destination"` // Destination of transaction
	Description string `json:"description"` // Additional context
	Reference   string `json:"reference"`   // External reference

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`   // When transaction was created
	UpdatedAt   time.Time  `json:"updated_at"`   // When transaction was last updated
	CompletedAt *time.Time `json:"completed_at"` // When transaction completed (if done)

	// Metadata allows for extension while maintaining type safety
	Metadata map[string]string `json:"metadata"` // Additional metadata with string values
}

// Validate performs validation on the transaction details
func (d *Details) Validate() error {
	// Add validation logic
	return nil
}

// IsMonetary returns true if this is a monetary transaction
func (d *Details) IsMonetary() bool {
	return d.Type == TypePayment || d.Type == TypeTransfer
}

// RequiresAuthorization returns true if this transaction requires authorization
func (d *Details) RequiresAuthorization() bool {
	return d.Type == TypePayment || d.Type == TypeAuthorization
}

// GetMetadata returns both standard and custom metadata as a map
func (d *Details) GetMetadata() map[string]string {
	metadata := map[string]string{
		"type":        string(d.Type),
		"status":      string(d.Status),
		"client_id":   d.ClientID,
		"resource_id": d.ResourceID,
	}

	// Add non-empty optional fields
	if d.Source != "" {
		metadata["source"] = d.Source
	}
	if d.Destination != "" {
		metadata["destination"] = d.Destination
	}
	if d.Reference != "" {
		metadata["reference"] = d.Reference
	}

	// Add custom metadata
	for k, v := range d.Metadata {
		metadata[k] = v
	}

	return metadata
}
