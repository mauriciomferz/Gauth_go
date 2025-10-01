// Package audit provides types and functionality for audit logging.
package audit

import "time"

// Type represents the type of audit event
type Type string

const (
	// TypeAuth represents authentication events
	TypeAuth Type = "auth"
	// TypeToken represents token-related events
	TypeToken Type = "token"
	// TypeTransaction represents transaction events
	TypeTransaction Type = "transaction"
	// TypeSystem represents system events
	TypeSystem Type = "system"
)

// Action represents an audited action
type Action string

const (
	// ActionCreate represents a creation event
	ActionCreate Action = "create"
	// ActionRead represents a read event
	ActionRead Action = "read"
	// ActionUpdate represents an update event
	ActionUpdate Action = "update"
	// ActionDelete represents a deletion event
	ActionDelete Action = "delete"
	// ActionAuthenticate represents an authentication event
	ActionAuthenticate Action = "authenticate"
	// ActionAuthorize represents an authorization event
	ActionAuthorize Action = "authorize"
)

// Status represents the outcome of an audited event
type Status string

const (
	// StatusSuccess indicates successful completion
	StatusSuccess Status = "success"
	// StatusFailure indicates a failure
	StatusFailure Status = "failure"
	// StatusError indicates an error condition
	StatusError Status = "error"
	// StatusDenied indicates access denial
	StatusDenied Status = "denied"
)

// Event represents an audit event with strongly typed fields
type Event struct {
	// Core fields
	ID        string    `json:"id"`        // Unique event identifier
	Type      Type      `json:"type"`      // Event type
	Action    Action    `json:"action"`    // Action performed
	Status    Status    `json:"status"`    // Event outcome
	Timestamp time.Time `json:"timestamp"` // When event occurred

	// Actor information
	ActorID   string `json:"actor_id"`   // Who performed the action
	ActorType string `json:"actor_type"` // Type of actor (user, system, etc)
	ClientID  string `json:"client_id"`  // Client application ID
	IPAddress string `json:"ip_address"` // Source IP address
	UserAgent string `json:"user_agent"` // User agent string

	// Target information
	ResourceID   string `json:"resource_id"`   // Resource being acted upon
	ResourceType string `json:"resource_type"` // Type of resource

	// Context information
	RequestID   string `json:"request_id"`  // Associated request ID
	SessionID   string `json:"session_id"`  // Associated session ID
	TraceID     string `json:"trace_id"`    // Distributed tracing ID
	Environment string `json:"environment"` // Runtime environment
	Location    string `json:"location"`    // Geographic location

	// Additional information
	Message   string   `json:"message"`    // Event description
	Details   string   `json:"details"`    // Additional details
	ErrorCode string   `json:"error_code"` // Error code if failed
	Tags      []string `json:"tags"`       // Event tags

	// Typed metadata uses string values for consistency
	Metadata map[string]string `json:"metadata"` // Additional metadata
}

// Validate ensures the audit event is valid
func (e *Event) Validate() error {
	// Add validation logic
	return nil
}

// IsSuccess returns true if the event was successful
func (e *Event) IsSuccess() bool {
	return e.Status == StatusSuccess
}

// IsFailure returns true if the event failed
func (e *Event) IsFailure() bool {
	return e.Status == StatusFailure || e.Status == StatusError
}

// IsDenied returns true if access was denied
func (e *Event) IsDenied() bool {
	return e.Status == StatusDenied
}

// GetMetadata returns both standard and custom metadata
func (e *Event) GetMetadata() map[string]string {
	metadata := map[string]string{
		"type":        string(e.Type),
		"action":      string(e.Action),
		"status":      string(e.Status),
		"actor_id":    e.ActorID,
		"resource_id": e.ResourceID,
		"client_id":   e.ClientID,
		"environment": e.Environment,
	}

	// Add custom metadata
	for k, v := range e.Metadata {
		metadata[k] = v
	}

	return metadata
}
