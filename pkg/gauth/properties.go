// Package gauth provides the core authentication and authorization framework
package gauth

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Properties represents a collection of property values, keyed by string.
type Properties struct {
	values map[string]PropertyValue
}

// PropertyValue represents a strongly typed value for restriction properties.
type PropertyValue struct {
	StringValue string    `json:"string_value,omitempty"`
	IntValue    int       `json:"int_value,omitempty"`
	Int64Value  int64     `json:"int64_value,omitempty"`
	FloatValue  float64   `json:"float_value,omitempty"`
	BoolValue   bool      `json:"bool_value,omitempty"`
	TimeValue   string    `json:"time_value,omitempty"` // RFC3339 string
	Type        string    `json:"type"`
}

// NewStringProperty creates a new string property value
func NewStringProperty(value string) PropertyValue {
	return PropertyValue{
		Type:        "string",
		StringValue: value,
	}
}

// NewIntProperty creates a new integer property value
func NewIntProperty(value int) PropertyValue {
	return PropertyValue{
		Type:     "int",
		IntValue: value,
	}
}

// NewInt64Property creates a new int64 property value
func NewInt64Property(value int64) PropertyValue {
	return PropertyValue{
		Type:       "int64",
		Int64Value: value,
	}
}

// NewFloatProperty creates a new float property value
func NewFloatProperty(value float64) PropertyValue {
	return PropertyValue{
		Type:      "float",
		FloatValue: value,
	}
}

// NewBoolProperty creates a new boolean property value
func NewBoolProperty(value bool) PropertyValue {
	return PropertyValue{
		Type:     "bool",
		BoolValue: value,
	}
}

// NewTimeProperty creates a new time property value
func NewTimeProperty(value time.Time) PropertyValue {
	return PropertyValue{
		Type:      "time",
		TimeValue: value.Format(time.RFC3339),
	}
}

// ToString converts the property value to string
func (pv PropertyValue) ToString() string {
	switch pv.Type {
	case "string":
		return pv.StringValue
	case "int":
		return strconv.Itoa(pv.IntValue)
	case "int64":
		return strconv.FormatInt(pv.Int64Value, 10)
	case "float":
		return strconv.FormatFloat(pv.FloatValue, 'f', -1, 64)
	case "bool":
		return strconv.FormatBool(pv.BoolValue)
	case "time":
		return pv.TimeValue
	}
	return ""
}

// ToInt converts the property value to an integer
func (pv PropertyValue) ToInt() (int, error) {
	switch pv.Type {
	case "int":
		return pv.IntValue, nil
	case "int64":
		return int(pv.Int64Value), nil
	case "float":
		return int(pv.FloatValue), nil
	case "string":
		return strconv.Atoi(pv.StringValue)
	}
	return 0, fmt.Errorf("cannot convert %s to int", pv.Type)
}

func (pv PropertyValue) ToInt64() (int64, error) {
	switch pv.Type {
	case "int64":
		return pv.Int64Value, nil
	case "int":
		return int64(pv.IntValue), nil
	case "float":
		return int64(pv.FloatValue), nil
	case "string":
		return strconv.ParseInt(pv.StringValue, 10, 64)
	}
	return 0, fmt.Errorf("cannot convert %s to int64", pv.Type)
}

// ToFloat converts the property value to a float
func (pv PropertyValue) ToFloat() (float64, error) {
	switch pv.Type {
	case "float":
		return pv.FloatValue, nil
	case "int":
		return float64(pv.IntValue), nil
	case "int64":
		return float64(pv.Int64Value), nil
	case "string":
		return strconv.ParseFloat(pv.StringValue, 64)
	}
	return 0, fmt.Errorf("cannot convert %s to float", pv.Type)
}

// ToBool converts the property value to a boolean
func (pv PropertyValue) ToBool() (bool, error) {
	switch pv.Type {
	case "bool":
		return pv.BoolValue, nil
	case "string":
		return strconv.ParseBool(pv.StringValue)
	}
	return false, fmt.Errorf("cannot convert %s to bool", pv.Type)
}

// ToTime converts the property value to time.Time
func (pv PropertyValue) ToTime() (time.Time, error) {
	switch pv.Type {
	case "time":
		return time.Parse(time.RFC3339, pv.TimeValue)
	case "string":
		return time.Parse(time.RFC3339, pv.StringValue)
	}
	return time.Time{}, fmt.Errorf("cannot convert %s to time", pv.Type)
}

func (pv PropertyValue) MarshalJSON() ([]byte, error) {
	type Alias PropertyValue
	return json.Marshal((Alias)(pv))
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (pv *PropertyValue) UnmarshalJSON(data []byte) error {
	type Alias PropertyValue
	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	*pv = PropertyValue(*aux)
	return nil
}

// NewProperties creates a new empty properties collection
func NewProperties() *Properties {
	return &Properties{
		values: make(map[string]PropertyValue),
	}
}

// Deprecated: PropertiesFromMap creates a new Properties instance from a map.
// This function exists only for migration from legacy code using map[string]interface{}.
// Use strongly-typed Properties methods instead.
func PropertiesFromMap(m map[string]interface{}) *Properties {
	p := NewProperties()
	for k, v := range m {
		switch val := v.(type) {
		case string:
			p.SetString(k, val)
		case int:
			p.SetInt(k, val)
		case int64:
			p.SetInt64(k, val)
		case float64:
			p.SetFloat(k, val)
		case bool:
			p.SetBool(k, val)
		case time.Time:
			p.SetTime(k, val)
		default:
			p.SetString(k, fmt.Sprintf("%v", val))
		}
	}
	return p
}

// Set sets a property value
func (p *Properties) Set(key string, value PropertyValue) {
	p.values[key] = value
}

// Get gets a property value
func (p *Properties) Get(key string) (PropertyValue, bool) {
	v, ok := p.values[key]
	return v, ok
}

// SetString sets a string property value
func (p *Properties) SetString(key string, value string) {
	p.Set(key, NewStringProperty(value))
}

// SetInt sets an integer property value
func (p *Properties) SetInt(key string, value int) {
	p.Set(key, NewIntProperty(value))
}

// SetInt64 sets an int64 property value
func (p *Properties) SetInt64(key string, value int64) {
	p.Set(key, NewInt64Property(value))
}

// SetFloat sets a float property value
func (p *Properties) SetFloat(key string, value float64) {
	p.Set(key, NewFloatProperty(value))
}

// SetBool sets a boolean property value
func (p *Properties) SetBool(key string, value bool) {
	p.Set(key, NewBoolProperty(value))
}

// SetTime sets a time property value
func (p *Properties) SetTime(key string, value time.Time) {
	p.Set(key, NewTimeProperty(value))
}

// GetString gets a string property value
func (p *Properties) GetString(key string) (string, bool) {
	v, ok := p.Get(key)
	if !ok {
		return "", false
	}
	return v.ToString(), true
}

// GetInt gets an integer property value
func (p *Properties) GetInt(key string) (int, bool) {
	v, ok := p.Get(key)
	if !ok {
		return 0, false
	}
	i, err := v.ToInt()
	return i, err == nil
}

// GetInt64 gets an int64 property value
func (p *Properties) GetInt64(key string) (int64, bool) {
	v, ok := p.Get(key)
	if !ok {
		return 0, false
	}
	i, err := v.ToInt64()
	return i, err == nil
}

// GetFloat gets a float property value
func (p *Properties) GetFloat(key string) (float64, bool) {
	v, ok := p.Get(key)
	if !ok {
		return 0, false
	}
	f, err := v.ToFloat()
	return f, err == nil
}

// GetBool gets a boolean property value
func (p *Properties) GetBool(key string) (bool, bool) {
	v, ok := p.Get(key)
	if !ok {
		return false, false
	}
	b, err := v.ToBool()
	return b, err == nil
}

// GetTime gets a time property value
func (p *Properties) GetTime(key string) (time.Time, bool) {
	v, ok := p.Get(key)
	if !ok {
		return time.Time{}, false
	}
	t, err := v.ToTime()
	return t, err == nil
}

// Keys returns all keys in the properties
func (p *Properties) Keys() []string {
	keys := make([]string, 0, len(p.values))
	for k := range p.values {
		keys = append(keys, k)
	}
	return keys
}

// Delete removes a property
func (p *Properties) Delete(key string) {
	delete(p.values, key)
}

// Len returns the number of properties
func (p *Properties) Len() int {
	return len(p.values)
}

// MarshalJSON implements the json.Marshaler interface
func (p *Properties) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.values)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (p *Properties) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	p.values = make(map[string]PropertyValue)
	for k, v := range rawMap {
		var pv PropertyValue
		if err := json.Unmarshal(v, &pv); err != nil {
			return err
		}
		p.values[k] = pv
	}
	return nil
}

// ToMap converts the properties to a map[string]interface{} for legacy compatibility
func (p *Properties) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range p.values {
		switch v.Type {
		case "string":
			result[k] = v.StringValue
		case "int":
			result[k] = v.IntValue
		case "int64":
			result[k] = v.Int64Value
		case "float":
			result[k] = v.FloatValue
		case "bool":
			result[k] = v.BoolValue
		case "time":
			parsed, err := time.Parse(time.RFC3339, v.TimeValue)
			if err == nil {
				result[k] = parsed
			} else {
				result[k] = v.TimeValue
			}
		default:
			result[k] = nil
		}
	}
	return result
}
