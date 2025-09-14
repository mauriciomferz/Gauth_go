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
	// RefundTransaction represents a refund transaction
	RefundTransaction TransactionType = "refund"
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

// TransactionDetails represents the details of a transaction with strong typing
type TransactionDetails struct {
	// ID uniquely identifies the transaction
	ID string `json:"id"`

	// Type indicates the transaction type
	Type TransactionType `json:"type"`

	// Status indicates the current transaction status
	Status TransactionStatus `json:"status"`

	// ClientID identifies the client making the transaction
	ClientID string `json:"client_id"`

	// ResourceID identifies the resource being accessed
	ResourceID string `json:"resource_id"`

	// Scopes contains required access scopes
	Scopes []string `json:"scopes"`

	// Amount represents the transaction amount
	Amount float64 `json:"amount"`

	// Currency represents the transaction currency
	Currency string `json:"currency"`

	// Timestamp records when the transaction occurred
	Timestamp time.Time `json:"timestamp"`

	// Source represents the transaction source
	Source string `json:"source,omitempty"`

	// Destination represents the transaction destination
	Destination string `json:"destination,omitempty"`

	// Description provides additional context
	Description string `json:"description,omitempty"`

	// Reference contains external reference numbers
	Reference string `json:"reference,omitempty"`

	// CustomMetadata allows for extensibility while maintaining type safety
	// for common fields above
	CustomMetadata map[string]string `json:"custom_metadata,omitempty"`
}

// Validate performs validation on the transaction details
func (t *TransactionDetails) Validate() error {
	// Add validation logic here
	return nil
}

// IsMonetary returns true if this is a monetary transaction
func (t *TransactionDetails) IsMonetary() bool {
	return t.Type == PaymentTransaction || t.Type == TransferTransaction
}

// RequiresAuthorization returns true if this transaction requires authorization
func (t *TransactionDetails) RequiresAuthorization() bool {
	return t.Type == PaymentTransaction || t.Type == AuthorizationTransaction
}

// GetMetadata returns both standard and custom metadata as a map
func (t *TransactionDetails) GetMetadata() map[string]string {
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
	for k, v := range t.CustomMetadata {
		metadata[k] = v
	}

	return metadata
}
