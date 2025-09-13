// Package gauth provides types and interfaces for transactions.
package gauth

import "time"

// TransactionType represents the type of transaction
type TransactionType string

const (
	// PaymentTransaction represents a payment transaction
	PaymentTransaction TransactionType = "payment"
	// TransferTransaction represents a transfer transaction
	TransferTransaction TransactionType = "transfer"
	// AuthorizationTransaction represents an authorization transaction
	AuthorizationTransaction TransactionType = "authorization"
	// ValidationTransaction represents a validation transaction
	ValidationTransaction TransactionType = "validation"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	// TransactionPending indicates a pending transaction
	TransactionPending TransactionStatus = "pending"
	// TransactionSuccess indicates a successful transaction
	TransactionSuccess TransactionStatus = "success"
	// TransactionFailed indicates a failed transaction
	TransactionFailed TransactionStatus = "failed"
	// TransactionCancelled indicates a cancelled transaction
	TransactionCancelled TransactionStatus = "cancelled"
)

// Transaction represents a strongly-typed resource transaction.
type Transaction struct {
	// Core fields
	ID         string            `json:"id"`          // Unique transaction identifier
	Type       TransactionType   `json:"type"`        // Type of transaction
	Status     TransactionStatus `json:"status"`      // Current status
	ClientID   string            `json:"client_id"`   // Client initiating the transaction
	ResourceID string            `json:"resource_id"` // Resource being accessed
	Scopes     []string          `json:"scopes"`      // Required scopes

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

// Validate performs validation on the transaction
func (t *Transaction) Validate() error {
	// Add validation logic
	return nil
}

// IsMonetary returns true if this is a monetary transaction
func (t *Transaction) IsMonetary() bool {
	return t.Type == PaymentTransaction || t.Type == TransferTransaction
}

// RequiresAuthorization returns true if this transaction requires authorization
func (t *Transaction) RequiresAuthorization() bool {
	return t.Type == PaymentTransaction || t.Type == AuthorizationTransaction
}

// GetMetadata returns both standard and custom metadata as a map
func (t *Transaction) GetMetadata() map[string]string {
	metadata := map[string]string{
		"type":        string(t.Type),
		"status":      string(t.Status),
		"client_id":   t.ClientID,
		"resource_id": t.ResourceID,
	}

	// Add non-empty optional fields
	if t.Source != "" {
		metadata["source"] = t.Source
	}
	if t.Destination != "" {
		metadata["destination"] = t.Destination
	}
	if t.Reference != "" {
		metadata["reference"] = t.Reference
	}

	// Add custom metadata
	for k, v := range t.Metadata {
		metadata[k] = v
	}

	return metadata
}
