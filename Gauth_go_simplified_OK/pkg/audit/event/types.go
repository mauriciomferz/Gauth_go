// Package event defines the core event types and interfaces for audit logging.
package event

import (
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// Type represents the type of event
type Type uint32

const (
	TypeUnknown Type = iota
	TypeAuth         // Authentication events
	TypeAuthz        // Authorization events
	TypeToken        // Token management events
	TypeRate         // Rate limiting events
	TypeCircuit      // Circuit breaker events
	TypeAudit        // Audit events
)

// Result represents the result of an event
type Result uint8

const (
	ResultUnknown Result = iota
	ResultSuccess
	ResultFailure
	ResultDenied
	ResultError
)

const (
	unknownString = "Unknown"
)

// Actor represents an entity that triggered the event
type Actor struct {
	ID        string            // Unique identifier
	Type      string            // Type of actor (user, service, etc.)
	Metadata  map[string]string // Additional actor metadata
	IPAddress string            // Source IP if applicable
}

// Resource represents the target of an event
type Resource struct {
	ID       string            // Resource identifier
	Type     string            // Resource type
	Name     string            // Human readable name
	Metadata map[string]string // Additional resource metadata
}

// Event represents a complete audit event
type Event struct {
	ID          string            // Unique event ID
	Type        Type              // Event type
	Timestamp   time.Time         // When the event occurred
	Actor       Actor             // Who/what triggered the event
	Action      string            // What was attempted
	Resource    Resource          // What was targeted
	Result      Result            // Outcome
	Metadata    map[string]string // Additional event metadata
	ChainID     string            // For linking related events
	Tags        []string          // For categorization
	Duration    time.Duration     // How long the action took
	ErrorDetail string            // Details if Result is Error
}

// Builder provides a fluent interface for creating events
type Builder struct {
	event Event
}

// NewEvent creates a new event builder
func NewEvent(typ Type) *Builder {
	return &Builder{
		event: Event{
			Type:      typ,
			Timestamp: time.Now(),
			Metadata:  make(map[string]string),
		},
	}
}

// WithActor sets the event actor
func (b *Builder) WithActor(id string, typ string) *Builder {
	b.event.Actor = Actor{
		ID:       id,
		Type:     typ,
		Metadata: make(map[string]string),
	}
	return b
}

// WithResource sets the event resource
func (b *Builder) WithResource(id string, typ string, name string) *Builder {
	b.event.Resource = Resource{
		ID:       id,
		Type:     typ,
		Name:     name,
		Metadata: make(map[string]string),
	}
	return b
}

// WithAction sets the event action
func (b *Builder) WithAction(action string) *Builder {
	b.event.Action = action
	return b
}

// WithResult sets the event result
func (b *Builder) WithResult(result Result) *Builder {
	b.event.Result = result
	return b
}

// WithMetadata adds metadata to the event
func (b *Builder) WithMetadata(key, value string) *Builder {
	b.event.Metadata[key] = value
	return b
}

// WithChainID sets the event chain ID
func (b *Builder) WithChainID(chainID string) *Builder {
	b.event.ChainID = chainID
	return b
}

// WithTags adds tags to the event
func (b *Builder) WithTags(tags ...string) *Builder {
	b.event.Tags = append(b.event.Tags, tags...)
	return b
}

// WithError sets the error details
func (b *Builder) WithError(err error) *Builder {
	if err != nil {
		b.event.Result = ResultError
		b.event.ErrorDetail = err.Error()
	}
	return b
}

// Build creates the final event
func (b *Builder) Build() *Event {
	if b.event.ID == "" {
		b.event.ID = token.NewID()
	}
	return &b.event
}

// String implements fmt.Stringer for Type
func (t Type) String() string {
	switch t {
	case TypeAuth:
		return "Auth"
	case TypeAuthz:
		return "Authz"
	case TypeToken:
		return "Token"
	case TypeRate:
		return "Rate"
	case TypeCircuit:
		return "Circuit"
	case TypeAudit:
		return "Audit"
	default:
		return unknownString
	}
}

// String implements fmt.Stringer for Result
func (r Result) String() string {
	switch r {
	case ResultSuccess:
		return "Success"
	case ResultFailure:
		return "Failure"
	case ResultDenied:
		return "Denied"
	case ResultError:
		return "Error"
	default:
		return unknownString
	}
}
