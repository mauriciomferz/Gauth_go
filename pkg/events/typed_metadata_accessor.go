package events

import (
	"fmt"
	"strings"
	"time"
)

// TypedEventData is an interface that can be implemented by
// structures that want to be converted to and from event metadata
type TypedEventData interface {
	// ToMetadata converts the structure to event metadata
	ToMetadata() *Metadata

	// FromMetadata populates the structure from event metadata
	// Returns error if metadata is missing required fields or has invalid types
	FromMetadata(m *Metadata) error
}

// WithTypedMetadata is a helper function to add typed data to an event
func WithTypedMetadata(e Event, data TypedEventData) Event {
	if data == nil {
		return e
	}
	e.Metadata = data.ToMetadata()
	return e
}

// TypedMetadataAccessor provides helper methods for accessing structured data
// from event metadata
type TypedMetadataAccessor struct {
	metadata *Metadata
	prefix   string
}

// NewTypedMetadataAccessor creates a new accessor for the given metadata
func NewTypedMetadataAccessor(metadata *Metadata, prefix string) *TypedMetadataAccessor {
	if prefix != "" && !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}

	return &TypedMetadataAccessor{
		metadata: metadata,
		prefix:   prefix,
	}
}

// GetString gets a string value with the given field name
func (a *TypedMetadataAccessor) GetString(field string) (string, error) {
	if a.metadata == nil {
		return "", fmt.Errorf("metadata is nil")
	}
	val, ok := a.metadata.GetString(a.prefix + field)
	if !ok {
		return "", fmt.Errorf("string field '%s' not found", a.prefix+field)
	}
	return val, nil
}

// GetInt gets an integer value with the given field name
func (a *TypedMetadataAccessor) GetInt(field string) (int, error) {
	if a.metadata == nil {
		return 0, fmt.Errorf("metadata is nil")
	}
	val, ok := a.metadata.GetInt(a.prefix + field)
	if !ok {
		return 0, fmt.Errorf("int field '%s' not found", a.prefix+field)
	}
	return val, nil
}

// GetBool gets a boolean value with the given field name
func (a *TypedMetadataAccessor) GetBool(field string) (bool, error) {
	if a.metadata == nil {
		return false, fmt.Errorf("metadata is nil")
	}
	val, ok := a.metadata.GetBool(a.prefix + field)
	if !ok {
		return false, fmt.Errorf("bool field '%s' not found", a.prefix+field)
	}
	return val, nil
}

// GetFloat gets a float value with the given field name
func (a *TypedMetadataAccessor) GetFloat(field string) (float64, error) {
	if a.metadata == nil {
		return 0, fmt.Errorf("metadata is nil")
	}
	val, ok := a.metadata.GetFloat(a.prefix + field)
	if !ok {
		return 0, fmt.Errorf("float field '%s' not found", a.prefix+field)
	}
	return val, nil
}

// GetTime gets a time value with the given field name
func (a *TypedMetadataAccessor) GetTime(field string) (time.Time, error) {
	if a.metadata == nil {
		return time.Time{}, fmt.Errorf("metadata is nil")
	}
	val, err := a.metadata.GetTime(a.prefix + field)
	if err != nil {
		return time.Time{}, fmt.Errorf("time field '%s' not found: %v", a.prefix+field, err)
	}
	return val, nil
}

// SetString sets a string value with the given field name
func (a *TypedMetadataAccessor) SetString(field string, value string) {
	if a.metadata == nil {
		return
	}
	a.metadata.SetString(a.prefix+field, value)
}

// SetInt sets an integer value with the given field name
func (a *TypedMetadataAccessor) SetInt(field string, value int) {
	if a.metadata == nil {
		return
	}
	a.metadata.SetInt(a.prefix+field, value)
}

// SetBool sets a boolean value with the given field name
func (a *TypedMetadataAccessor) SetBool(field string, value bool) {
	if a.metadata == nil {
		return
	}
	a.metadata.SetBool(a.prefix+field, value)
}

// SetFloat sets a float value with the given field name
func (a *TypedMetadataAccessor) SetFloat(field string, value float64) {
	if a.metadata == nil {
		return
	}
	a.metadata.SetFloat(a.prefix+field, value)
}

// SetTime sets a time value with the given field name
func (a *TypedMetadataAccessor) SetTime(field string, value time.Time) {
	if a.metadata == nil {
		return
	}
	a.metadata.SetTime(a.prefix+field, value)
}
