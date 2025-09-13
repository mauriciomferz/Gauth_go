// Package gauth provides the core authentication and authorization framework
package gauth

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// PropertyValue represents a strongly typed value for restriction properties
type PropertyValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// NewStringProperty creates a new string property value
func NewStringProperty(value string) PropertyValue {
	return PropertyValue{
		Type:  "string",
		Value: value,
	}
}

// NewIntProperty creates a new integer property value
func NewIntProperty(value int) PropertyValue {
	return PropertyValue{
		Type:  "int",
		Value: value,
	}
}

// NewInt64Property creates a new int64 property value
func NewInt64Property(value int64) PropertyValue {
	return PropertyValue{
		Type:  "int64",
		Value: value,
	}
}

// NewFloatProperty creates a new float property value
func NewFloatProperty(value float64) PropertyValue {
	return PropertyValue{
		Type:  "float",
		Value: value,
	}
}

// NewBoolProperty creates a new boolean property value
func NewBoolProperty(value bool) PropertyValue {
	return PropertyValue{
		Type:  "bool",
		Value: value,
	}
}

// NewTimeProperty creates a new time property value
func NewTimeProperty(value time.Time) PropertyValue {
	return PropertyValue{
		Type:  "time",
		Value: value.Format(time.RFC3339),
	}
}

// ToString converts the property value to string
func (pv PropertyValue) ToString() string {
	switch pv.Type {
	case "string":
		if s, ok := pv.Value.(string); ok {
			return s
		}
	case "int":
		if i, ok := pv.Value.(int); ok {
			return strconv.Itoa(i)
		}
	case "int64":
		if i, ok := pv.Value.(int64); ok {
			return strconv.FormatInt(i, 10)
		}
	case "float":
		if f, ok := pv.Value.(float64); ok {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
	case "bool":
		if b, ok := pv.Value.(bool); ok {
			return strconv.FormatBool(b)
		}
	case "time":
		if t, ok := pv.Value.(string); ok {
			return t
		}
	}
	return fmt.Sprintf("%v", pv.Value)
}

// ToInt converts the property value to an integer
func (pv PropertyValue) ToInt() (int, error) {
	switch pv.Type {
	case "int":
		if i, ok := pv.Value.(int); ok {
			return i, nil
		}
	case "int64":
		if i, ok := pv.Value.(int64); ok {
			return int(i), nil
		}
	case "float":
		if f, ok := pv.Value.(float64); ok {
			return int(f), nil
		}
	case "string":
		if s, ok := pv.Value.(string); ok {
			i, err := strconv.Atoi(s)
			return i, err
		}
	}
	return 0, fmt.Errorf("cannot convert %s to int", pv.Type)
}

// ToInt64 converts the property value to an int64
func (pv PropertyValue) ToInt64() (int64, error) {
	switch pv.Type {
	case "int64":
		if i, ok := pv.Value.(int64); ok {
			return i, nil
		}
	case "int":
		if i, ok := pv.Value.(int); ok {
			return int64(i), nil
		}
	case "float":
		if f, ok := pv.Value.(float64); ok {
			return int64(f), nil
		}
	case "string":
		if s, ok := pv.Value.(string); ok {
			return strconv.ParseInt(s, 10, 64)
		}
	}
	return 0, fmt.Errorf("cannot convert %s to int64", pv.Type)
}

// ToBool converts the property value to a boolean
func (pv PropertyValue) ToBool() (bool, error) {
	switch pv.Type {
	case "bool":
		if b, ok := pv.Value.(bool); ok {
			return b, nil
		}
	case "string":
		if s, ok := pv.Value.(string); ok {
			return strconv.ParseBool(s)
		}
	}
	return false, fmt.Errorf("cannot convert %s to bool", pv.Type)
}

// ToFloat converts the property value to a float
func (pv PropertyValue) ToFloat() (float64, error) {
	switch pv.Type {
	case "float":
		if f, ok := pv.Value.(float64); ok {
			return f, nil
		}
	case "int":
		if i, ok := pv.Value.(int); ok {
			return float64(i), nil
		}
	case "string":
		if s, ok := pv.Value.(string); ok {
			return strconv.ParseFloat(s, 64)
		}
	}
	return 0, fmt.Errorf("cannot convert %s to float", pv.Type)
}

// ToTime converts the property value to a time.Time
func (pv PropertyValue) ToTime() (time.Time, error) {
	switch pv.Type {
	case "time":
		if s, ok := pv.Value.(string); ok {
			return time.Parse(time.RFC3339, s)
		}
	case "string":
		if s, ok := pv.Value.(string); ok {
			return time.Parse(time.RFC3339, s)
		}
	}
	return time.Time{}, fmt.Errorf("cannot convert %s to time", pv.Type)
}

// MarshalJSON implements the json.Marshaler interface
func (pv PropertyValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}{
		Type:  pv.Type,
		Value: pv.Value,
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (pv *PropertyValue) UnmarshalJSON(data []byte) error {
	type tempStruct struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}
	var temp tempStruct
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	pv.Type = temp.Type
	switch temp.Type {
	case "string":
		var s string
		if err := json.Unmarshal(temp.Value, &s); err != nil {
			return err
		}
		pv.Value = s
	case "int":
		var i int
		if err := json.Unmarshal(temp.Value, &i); err != nil {
			return err
		}
		pv.Value = i
	case "int64":
		var i int64
		if err := json.Unmarshal(temp.Value, &i); err != nil {
			return err
		}
		pv.Value = i
	case "float":
		var f float64
		if err := json.Unmarshal(temp.Value, &f); err != nil {
			return err
		}
		pv.Value = f
	case "bool":
		var b bool
		if err := json.Unmarshal(temp.Value, &b); err != nil {
			return err
		}
		pv.Value = b
	case "time":
		var s string
		if err := json.Unmarshal(temp.Value, &s); err != nil {
			return err
		}
		pv.Value = s
	default:
		var v interface{}
		if err := json.Unmarshal(temp.Value, &v); err != nil {
			return err
		}
		pv.Value = v
	}

	return nil
}

// Properties represents a collection of strongly typed property values
type Properties struct {
	values map[string]PropertyValue
}

// NewProperties creates a new empty properties collection
func NewProperties() *Properties {
	return &Properties{
		values: make(map[string]PropertyValue),
	}
}

// FromMap creates a new Properties instance from a map
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
			if s, ok := v.Value.(string); ok {
				result[k] = s
			}
		case "int":
			if i, ok := v.Value.(int); ok {
				result[k] = i
			}
		case "int64":
			if i, ok := v.Value.(int64); ok {
				result[k] = i
			}
		case "float":
			if f, ok := v.Value.(float64); ok {
				result[k] = f
			}
		case "bool":
			if b, ok := v.Value.(bool); ok {
				result[k] = b
			}
		case "time":
			if t, ok := v.Value.(string); ok {
				parsed, err := time.Parse(time.RFC3339, t)
				if err == nil {
					result[k] = parsed
				} else {
					result[k] = t
				}
			}
		default:
			result[k] = v.Value
		}
	}
	return result
}
