package authz

import (
	"fmt"
	"time"
)

// ContextValue represents a typed value for authorization context
// ContextValue represents a typed value for authorization context.
// Only one field should be non-zero at a time.
type ContextValue struct {
	StringValue string  `json:"string_value,omitempty"`
	IntValue    int     `json:"int_value,omitempty"`
	FloatValue  float64 `json:"float_value,omitempty"`
	BoolValue   bool    `json:"bool_value,omitempty"`
	TimeValue   string  `json:"time_value,omitempty"`
	Type        string  `json:"type"`
}

// Context represents a structured authorization context
type Context struct {
	// Values holds the context values
	Values map[string]ContextValue `json:"values"`

	// Timestamp when the context was created
	Timestamp time.Time `json:"timestamp"`
}

// NewContext creates a new empty authorization context
func NewContext() *Context {
	return &Context{
		Values:    make(map[string]ContextValue),
		Timestamp: time.Now(),
	}
}

// GetString retrieves a string value from the context
func (c *Context) GetString(key string) (string, bool) {
	if val, ok := c.Values[key]; ok && val.Type == "string" {
		return val.StringValue, true
	}
	return "", false
}

// GetInt retrieves an integer value from the context
func (c *Context) GetInt(key string) (int, bool) {
	if val, ok := c.Values[key]; ok && val.Type == "int" {
		return val.IntValue, true
	}
	return 0, false
}

// GetFloat retrieves a float value from the context
func (c *Context) GetFloat(key string) (float64, bool) {
	if val, ok := c.Values[key]; ok && val.Type == "float" {
		return val.FloatValue, true
	}
	return 0, false
}

// GetBool retrieves a boolean value from the context
func (c *Context) GetBool(key string) (bool, bool) {
	if val, ok := c.Values[key]; ok && val.Type == "bool" {
		return val.BoolValue, true
	}
	return false, false
}

// GetTime retrieves a time value from the context
func (c *Context) GetTime(key string) (time.Time, bool) {
	if val, ok := c.Values[key]; ok && val.Type == "time" {
		t, err := time.Parse(time.RFC3339, val.TimeValue)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// SetString sets a string value in the context
func (c *Context) SetString(key, value string) {
	c.Values[key] = ContextValue{
		Type:        "string",
		StringValue: value,
	}
}

// SetInt sets an integer value in the context
func (c *Context) SetInt(key string, value int) {
	c.Values[key] = ContextValue{
		Type:     "int",
		IntValue: value,
	}
}

// SetFloat sets a float value in the context
func (c *Context) SetFloat(key string, value float64) {
	c.Values[key] = ContextValue{
		Type:       "float",
		FloatValue: value,
	}
}

// SetBool sets a boolean value in the context
func (c *Context) SetBool(key string, value bool) {
	c.Values[key] = ContextValue{
		Type:      "bool",
		BoolValue: value,
	}
}

// SetTime sets a time value in the context
func (c *Context) SetTime(key string, value time.Time) {
	c.Values[key] = ContextValue{
		Type:      "time",
		TimeValue: value.Format(time.RFC3339),
	}
}

// Has checks if a key exists in the context
func (c *Context) Has(key string) bool {
	_, exists := c.Values[key]
	return exists
}

// Remove removes a key from the context
func (c *Context) Remove(key string) {
	delete(c.Values, key)
}

// GetKeys returns all keys in the context
func (c *Context) GetKeys() []string {
	keys := make([]string, 0, len(c.Values))
	for k := range c.Values {
		keys = append(keys, k)
	}
	return keys
}

// ToMap converts the context to a map for backward compatibility
func (c *Context) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range c.Values {
		switch v.Type {
		case "string":
			result[k] = v.StringValue
		case "int":
			result[k] = v.IntValue
		case "float":
			result[k] = v.FloatValue
		case "bool":
			result[k] = v.BoolValue
		case "time":
			t, err := time.Parse(time.RFC3339, v.TimeValue)
			if err == nil {
				result[k] = t
			} else {
				result[k] = v.TimeValue
			}
		default:
			result[k] = nil
		}
	}
	return result
}

// Deprecated: FromMap converts a map to a context for backward compatibility.
// This function exists only for migration from legacy code using map[string]interface{}.
// Use strongly-typed Context methods instead.
func FromMap(data map[string]interface{}) *Context {
	ctx := NewContext()
	for k, v := range data {
		switch val := v.(type) {
		case string:
			ctx.SetString(k, val)
		case int:
			ctx.SetInt(k, val)
		case float64:
			ctx.SetFloat(k, val)
		case bool:
			ctx.SetBool(k, val)
		case time.Time:
			ctx.SetTime(k, val)
		default:
			// For any other type, store as string representation
			ctx.SetString(k, fmt.Sprintf("%v", val))
		}
	}
	return ctx
}
