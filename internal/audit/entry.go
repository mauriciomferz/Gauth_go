// Package audit provides audit logging functionality built on top of the events package.
package audit

import (
	"time"

	"github.com/Gimel-Foundation/gauth/internal/events"
)

// AuditType represents audit log entry types
type AuditType string

const (
	// AuditTypeAuth represents authentication audit entries
	AuditTypeAuth AuditType = "auth"
	// AuditTypeToken represents token-related audit entries
	AuditTypeToken AuditType = "token"
	// AuditTypeResource represents resource access audit entries
	AuditTypeResource AuditType = "resource"
	// AuditTypeAdmin represents administrative audit entries
	AuditTypeAdmin AuditType = "admin"
)

// Level represents audit log entry severity
type Level string

const (
	// LevelInfo represents informational entries
	LevelInfo Level = "info"
	// LevelWarning represents warning entries
	LevelWarning Level = "warning"
	// LevelError represents error entries
	LevelError Level = "error"
	// LevelCritical represents critical entries
	LevelCritical Level = "critical"
)

// Entry represents an audit log entry
type Entry struct {
	// Core fields
	ID        string    `json:"id"`        // Unique entry identifier
	Type      AuditType `json:"type"`      // Entry type
	Level     Level     `json:"level"`     // Entry severity level
	Timestamp time.Time `json:"timestamp"` // When entry was created

	// Actor information
	ActorID   string `json:"actor_id"`   // Who performed the action
	ActorType string `json:"actor_type"` // Type of actor (user, system)
	SessionID string `json:"session_id"` // Associated session

	// Action details
	Action   string `json:"action"`   // Action performed
	Resource string `json:"resource"` // Resource acted upon
	Result   string `json:"result"`   // Action result

	// Context information
	RequestID string `json:"request_id"` // Associated request
	TraceID   string `json:"trace_id"`   // Distributed tracing ID
	Source    string `json:"source"`     // Origin of the action

	// Event reference
	Event *events.Event `json:"event"` // Associated system event

	// Additional details
	Message  string            `json:"message"`  // Human-readable message
	Metadata map[string]string `json:"metadata"` // Additional context
}

// NewEntry creates a new audit entry from an event
func NewEntry(event *events.Event) *Entry {
	entry := &Entry{
		ID:        event.Subject,
		Timestamp: event.Timestamp,
		Event:     event,
		Metadata:  make(map[string]string),
	}

	// Map event type to audit type and level
	switch event.Type {
	case events.AuthSuccess:
		entry.Type = AuditTypeAuth
		entry.Level = LevelInfo
		if details, ok := event.Details.(*events.AuthEventDetails); ok {
			entry.ActorID = details.ClientID
			entry.Message = "Authentication successful"
			entry.Action = "authenticate"
		}
	case events.AuthFailure:
		entry.Type = AuditTypeAuth
		entry.Level = LevelWarning
		if details, ok := event.Details.(*events.AuthEventDetails); ok {
			entry.ActorID = details.ClientID
			entry.Message = "Authentication failed"
			entry.Action = "authenticate"
			entry.Result = "failure"
			entry.Metadata["error_code"] = details.ErrorCode
			entry.Metadata["error_detail"] = details.ErrorDetail
		}
	case events.TokenIssued:
		entry.Type = AuditTypeToken
		entry.Level = LevelInfo
		if details, ok := event.Details.(*events.TokenEventDetails); ok {
			entry.Message = "Token issued"
			entry.Action = "issue"
			entry.Metadata["token_type"] = details.TokenType
			entry.Metadata["expires_in"] = string(details.ExpiresIn)
		}
	case events.TokenRevoked:
		entry.Type = AuditTypeToken
		entry.Level = LevelInfo
		if details, ok := event.Details.(*events.TokenEventDetails); ok {
			entry.Message = "Token revoked"
			entry.Action = "revoke"
			entry.Metadata["reason_code"] = details.ReasonCode
		}
	}

	return entry
}

// WithActor sets actor information
func (e *Entry) WithActor(id, typ string) *Entry {
	e.ActorID = id
	e.ActorType = typ
	return e
}

// WithSession sets session information
func (e *Entry) WithSession(sessionID string) *Entry {
	e.SessionID = sessionID
	return e
}

// WithRequest sets request context
func (e *Entry) WithRequest(requestID, traceID string) *Entry {
	e.RequestID = requestID
	e.TraceID = traceID
	return e
}

// WithResource sets resource information
func (e *Entry) WithResource(resource string) *Entry {
	e.Resource = resource
	return e
}

// WithMessage sets a custom message
func (e *Entry) WithMessage(msg string) *Entry {
	e.Message = msg
	return e
}

// AddMetadata adds metadata key-value pairs
func (e *Entry) AddMetadata(key, value string) *Entry {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// String returns a human-readable representation
func (e *Entry) String() string {
	return e.Message
}
