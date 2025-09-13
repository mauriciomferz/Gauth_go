// Package events provides typed metadata structures for events
package events

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Clear removes all keys from the metadata, including read-only values
func (m *Metadata) Clear() {
	if m.values == nil {
		return
	}
	for k := range m.values {
		delete(m.values, k)
	}
}

// Size returns the number of items in the metadata
func (m *Metadata) Size() int {
	if m.values == nil {
		return 0
	}
	return len(m.values)
}

// Has returns true if the key exists in the metadata
func (m *Metadata) Has(key string) bool {
	if m.values == nil {
		return false
	}
	_, exists := m.values[key]
	return exists
}

// Delete removes a key from the metadata if it exists and is not read-only
func (m *Metadata) Delete(key string) {
	if m.values == nil {
		return
	}
	if existingVal, exists := m.values[key]; exists && !existingVal.ReadOnly {
		delete(m.values, key)
	}
}

// MetadataValue represents a strongly typed value in event metadata
type MetadataValue struct {
	Type     string      `json:"type"`
	Value    interface{} `json:"value"`
	ReadOnly bool        `json:"read_only,omitempty"`
}

// NewStringValue creates a new string metadata value
func NewStringValue(value string) MetadataValue {
	return MetadataValue{
		Type:  "string",
		Value: value,
	}
}

// NewIntValue creates a new integer metadata value
func NewIntValue(value int) MetadataValue {
	return MetadataValue{
		Type:  "int",
		Value: value,
	}
}

// NewInt64Value creates a new int64 metadata value
func NewInt64Value(value int64) MetadataValue {
	return MetadataValue{
		Type:  "int64",
		Value: value,
	}
}

// NewFloatValue creates a new float metadata value
func NewFloatValue(value float64) MetadataValue {
	return MetadataValue{
		Type:  "float",
		Value: value,
	}
}

// NewBoolValue creates a new boolean metadata value
func NewBoolValue(value bool) MetadataValue {
	return MetadataValue{
		Type:  "bool",
		Value: value,
	}
}

// NewTimeValue creates a new time metadata value
func NewTimeValue(value time.Time) MetadataValue {
	return MetadataValue{
		Type:  "time",
		Value: value.Format(time.RFC3339),
	}
}

// NewReadOnlyValue creates a read-only value
func NewReadOnlyValue(value MetadataValue) MetadataValue {
	value.ReadOnly = true
	return value
}

// ToString converts the metadata value to string
func (mv MetadataValue) ToString() string {
	switch mv.Type {
	case "string":
		if s, ok := mv.Value.(string); ok {
			return s
		}
	case "int":
		if i, ok := mv.Value.(int); ok {
			return strconv.Itoa(i)
		}
	case "int64":
		if i, ok := mv.Value.(int64); ok {
			return strconv.FormatInt(i, 10)
		}
	case "float":
		if f, ok := mv.Value.(float64); ok {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
	case "bool":
		if b, ok := mv.Value.(bool); ok {
			return strconv.FormatBool(b)
		}
	case "time":
		if t, ok := mv.Value.(string); ok {
			return t
		}
	}
	return fmt.Sprintf("%v", mv.Value)
}

// ToInt converts the metadata value to an integer
func (mv MetadataValue) ToInt() (int, error) {
	switch mv.Type {
	case "int":
		if i, ok := mv.Value.(int); ok {
			return i, nil
		}
	case "string":
		if s, ok := mv.Value.(string); ok {
			return strconv.Atoi(s)
		}
	}
	return 0, fmt.Errorf("cannot convert %s to int", mv.Type)
}

// ToBool converts the metadata value to a boolean
func (mv MetadataValue) ToBool() (bool, error) {
	switch mv.Type {
	case "bool":
		if b, ok := mv.Value.(bool); ok {
			return b, nil
		}
	case "string":
		if s, ok := mv.Value.(string); ok {
			return strconv.ParseBool(s)
		}
	}
	return false, fmt.Errorf("cannot convert %s to bool", mv.Type)
}

// ToFloat converts the metadata value to a float
func (mv MetadataValue) ToFloat() (float64, error) {
	switch mv.Type {
	case "float":
		if f, ok := mv.Value.(float64); ok {
			return f, nil
		}
	case "int":
		if i, ok := mv.Value.(int); ok {
			return float64(i), nil
		}
	case "string":
		if s, ok := mv.Value.(string); ok {
			return strconv.ParseFloat(s, 64)
		}
	}
	return 0, fmt.Errorf("cannot convert %s to float", mv.Type)
}

// ToTime converts the metadata value to time.Time
func (mv MetadataValue) ToTime() (time.Time, error) {
	switch mv.Type {
	case "time":
		if s, ok := mv.Value.(string); ok {
			return time.Parse(time.RFC3339, s)
		}
	}
	return time.Time{}, fmt.Errorf("cannot convert %s to time", mv.Type)
}

// MarshalJSON implements json.Marshaler
func (mv MetadataValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type     string      `json:"type"`
		Value    interface{} `json:"value"`
		ReadOnly bool        `json:"read_only,omitempty"`
	}{
		Type:     mv.Type,
		Value:    mv.Value,
		ReadOnly: mv.ReadOnly,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (mv *MetadataValue) UnmarshalJSON(data []byte) error {
	var temp struct {
		Type     string          `json:"type"`
		Value    json.RawMessage `json:"value"`
		ReadOnly bool            `json:"read_only"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	mv.Type = temp.Type
	mv.ReadOnly = temp.ReadOnly

	switch temp.Type {
	case "string":
		var s string
		if err := json.Unmarshal(temp.Value, &s); err != nil {
			return err
		}
		mv.Value = s
	case "int":
		var i int
		if err := json.Unmarshal(temp.Value, &i); err != nil {
			return err
		}
		mv.Value = i
	case "int64":
		var i int64
		if err := json.Unmarshal(temp.Value, &i); err != nil {
			return err
		}
		mv.Value = i
	case "float":
		var f float64
		if err := json.Unmarshal(temp.Value, &f); err != nil {
			return err
		}
		mv.Value = f
	case "bool":
		var b bool
		if err := json.Unmarshal(temp.Value, &b); err != nil {
			return err
		}
		mv.Value = b
	case "time":
		var s string
		if err := json.Unmarshal(temp.Value, &s); err != nil {
			return err
		}
		mv.Value = s
	default:
		var v interface{}
		if err := json.Unmarshal(temp.Value, &v); err != nil {
			return err
		}
		mv.Value = v
	}

	return nil
}

// Metadata represents a collection of strongly typed metadata values
type Metadata struct {
	values map[string]MetadataValue
}

// NewMetadata creates a new empty metadata collection
func NewMetadata() *Metadata {
	return &Metadata{
		values: make(map[string]MetadataValue),
	}
}

// Set adds or updates a value in the metadata
func (m *Metadata) Set(key string, value MetadataValue) {
	if m.values == nil {
		m.values = make(map[string]MetadataValue)
	}

	// Don't update if existing value is read-only
	if existingVal, exists := m.values[key]; exists && existingVal.ReadOnly {
		return
	}

	m.values[key] = value
}

// SetString sets a string value
func (m *Metadata) SetString(key, value string) {
	m.Set(key, NewStringValue(value))
}

// SetInt sets an integer value
func (m *Metadata) SetInt(key string, value int) {
	m.Set(key, NewIntValue(value))
}

// SetInt64 sets an int64 value
func (m *Metadata) SetInt64(key string, value int64) {
	m.Set(key, NewInt64Value(value))
}

// SetFloat sets a float value
func (m *Metadata) SetFloat(key string, value float64) {
	m.Set(key, NewFloatValue(value))
}

// SetBool sets a boolean value
func (m *Metadata) SetBool(key string, value bool) {
	m.Set(key, NewBoolValue(value))
}

// SetTime sets a time value
func (m *Metadata) SetTime(key string, value time.Time) {
	m.Set(key, NewTimeValue(value))
}

// SetReadOnly sets a read-only value
func (m *Metadata) SetReadOnly(key string, value MetadataValue) {
	m.Set(key, NewReadOnlyValue(value))
}

// Get retrieves a value from metadata
func (m *Metadata) Get(key string) (MetadataValue, bool) {
	if m.values == nil {
		return MetadataValue{}, false
	}
	val, exists := m.values[key]
	return val, exists
}

// GetString retrieves a string value
func (m *Metadata) GetString(key string) (string, bool) {
	if val, exists := m.Get(key); exists && val.Type == "string" {
		if s, ok := val.Value.(string); ok {
			return s, true
		}
	}
	return "", false
}

// GetInt retrieves an int value
func (m *Metadata) GetInt(key string) (int, bool) {
	if val, exists := m.Get(key); exists && val.Type == "int" {
		if i, ok := val.Value.(int); ok {
			return i, true
		}
	}
	return 0, false
}

// GetBool retrieves a bool value
func (m *Metadata) GetBool(key string) (bool, bool) {
	if val, exists := m.Get(key); exists && val.Type == "bool" {
		if b, ok := val.Value.(bool); ok {
			return b, true
		}
	}
	return false, false
}

// GetFloat retrieves a float value
func (m *Metadata) GetFloat(key string) (float64, bool) {
	if val, exists := m.Get(key); exists {
		f, err := val.ToFloat()
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

// GetTime retrieves a time.Time value
func (m *Metadata) GetTime(key string) (time.Time, error) {
	if val, exists := m.Get(key); exists {
		return val.ToTime()
	}
	return time.Time{}, fmt.Errorf("key not found")
}

// Keys returns all keys in the metadata
func (m *Metadata) Keys() []string {
	if m.values == nil {
		return []string{}
	}

	keys := make([]string, 0, len(m.values))
	for k := range m.values {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of metadata items
func (m *Metadata) Len() int {
	if m.values == nil {
		return 0
	}
	return len(m.values)
}

// MarshalJSON implements json.Marshaler
func (m *Metadata) MarshalJSON() ([]byte, error) {
	if m.values == nil {
		return json.Marshal(map[string]MetadataValue{})
	}
	return json.Marshal(m.values)
}

// UnmarshalJSON implements json.Unmarshaler
func (m *Metadata) UnmarshalJSON(data []byte) error {
	m.values = make(map[string]MetadataValue)

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	for k, v := range rawMap {
		var mv MetadataValue
		if err := json.Unmarshal(v, &mv); err != nil {
			// If not a proper MetadataValue, try as a simple value
			var simpleVal interface{}
			if err := json.Unmarshal(v, &simpleVal); err != nil {
				return err
			}

			switch val := simpleVal.(type) {
			case string:
				mv = NewStringValue(val)
			case float64: // JSON numbers decode as float64
				if val == float64(int(val)) {
					mv = NewIntValue(int(val))
				} else {
					mv = NewFloatValue(val)
				}
			case bool:
				mv = NewBoolValue(val)
			default:
				// Handle as string
				mv = NewStringValue(fmt.Sprintf("%v", val))
			}
		}

		m.values[k] = mv
	}

	return nil
}
