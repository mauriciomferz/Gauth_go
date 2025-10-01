// Package events provides a unified event system for GAuth
// This file defines builder methods for the Event type
package events

import (
	"time"
)

// WithType sets the event type
func (e Event) WithType(eventType EventType) Event {
	e.Type = eventType
	return e
}

// WithAction sets the event action
func (e Event) WithAction(action string) Event {
	e.Action = action
	return e
}

// WithActionEnum sets the event action from an EventAction enum
func (e Event) WithActionEnum(action EventAction) Event {
	e.Action = string(action)
	return e
}

// WithStatus sets the event status
func (e Event) WithStatus(status string) Event {
	e.Status = status
	return e
}

// WithStatusEnum sets the event status from an EventStatus enum
func (e Event) WithStatusEnum(status EventStatus) Event {
	e.Status = string(status)
	return e
}

// WithSubject adds a subject to the event
func (e Event) WithSubject(subject string) Event {
	e.Subject = subject
	return e
}

// WithResource adds a resource to the event
func (e Event) WithResource(resource string) Event {
	e.Resource = resource
	return e
}

// WithMessage adds a message to the event
func (e Event) WithMessage(message string) Event {
	e.Message = message
	return e
}

// WithError adds an error to the event
func (e Event) WithError(err error) Event {
	if err != nil {
		e.Error = err.Error()
		e.Status = string(StatusError)
	}
	return e
}

// WithStringMetadata adds a string metadata to the event
func (e Event) WithStringMetadata(key, value string) Event {
	if e.Metadata == nil {
		e.Metadata = NewMetadata()
	}
	e.Metadata.SetString(key, value)
	return e
}

// WithIntMetadata adds an integer metadata to the event
func (e Event) WithIntMetadata(key string, value int) Event {
	if e.Metadata == nil {
		e.Metadata = NewMetadata()
	}
	e.Metadata.SetInt(key, value)
	return e
}

// WithBoolMetadata adds a boolean metadata to the event
func (e Event) WithBoolMetadata(key string, value bool) Event {
	if e.Metadata == nil {
		e.Metadata = NewMetadata()
	}
	e.Metadata.SetBool(key, value)
	return e
}

// MergeMetadata merges the given metadata with existing metadata
// MergeMetadata is removed: use type-safe setters instead
func (e Event) MergeMetadata(_ map[string]interface{}) Event {
	panic("MergeMetadata is removed: use type-safe setters instead")
}

// WithTimeMetadata adds a time metadata to the event
func (e Event) WithTimeMetadata(key string, value time.Time) Event {
	if e.Metadata == nil {
		e.Metadata = NewMetadata()
	}
	e.Metadata.SetTime(key, value)
	return e
}

// WithTypedMetadata adds a strongly typed metadata value
func (e Event) WithTypedMetadata(key string, value MetadataValue) Event {
	if e.Metadata == nil {
		e.Metadata = NewMetadata()
	}
	e.Metadata.Set(key, value)
	return e
}
