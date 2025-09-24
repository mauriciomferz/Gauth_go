package authz

import "fmt"

// Annotation represents a typed value for access response annotations
// Annotation represents a typed value for access response annotations.
// Only one field should be non-zero at a time.
type Annotation struct {
	StringValue string  `json:"string_value,omitempty"`
	IntValue    int     `json:"int_value,omitempty"`
	FloatValue  float64 `json:"float_value,omitempty"`
	BoolValue   bool    `json:"bool_value,omitempty"`
	Type        string  `json:"type"`
}

// Annotations represents structured annotations for access responses
type Annotations struct {
	// Values holds the annotation values
	Values map[string]Annotation `json:"values"`
}

// NewAnnotations creates a new empty annotations structure
func NewAnnotations() *Annotations {
	return &Annotations{
		Values: make(map[string]Annotation),
	}
}

// GetString retrieves a string annotation
func (a *Annotations) GetString(key string) (string, bool) {
	if val, ok := a.Values[key]; ok && val.Type == "string" {
		return val.StringValue, true
	}
	return "", false
}

// GetInt retrieves an integer annotation
func (a *Annotations) GetInt(key string) (int, bool) {
	if val, ok := a.Values[key]; ok && val.Type == "int" {
		return val.IntValue, true
	}
	return 0, false
}

// GetFloat retrieves a float annotation
func (a *Annotations) GetFloat(key string) (float64, bool) {
	if val, ok := a.Values[key]; ok && val.Type == "float" {
		return val.FloatValue, true
	}
	return 0, false
}

// GetBool retrieves a boolean annotation
func (a *Annotations) GetBool(key string) (bool, bool) {
	if val, ok := a.Values[key]; ok && val.Type == "bool" {
		return val.BoolValue, true
	}
	return false, false
}

// SetString sets a string annotation
func (a *Annotations) SetString(key, value string) {
	a.Values[key] = Annotation{
		Type:        "string",
		StringValue: value,
	}
}

// SetInt sets an integer annotation
func (a *Annotations) SetInt(key string, value int) {
	a.Values[key] = Annotation{
		Type:     "int",
		IntValue: value,
	}
}

// SetFloat sets a float annotation
func (a *Annotations) SetFloat(key string, value float64) {
	a.Values[key] = Annotation{
		Type:       "float",
		FloatValue: value,
	}
}

// SetBool sets a boolean annotation
func (a *Annotations) SetBool(key string, value bool) {
	a.Values[key] = Annotation{
		Type:      "bool",
		BoolValue: value,
	}
}

// Has checks if an annotation exists
func (a *Annotations) Has(key string) bool {
	_, exists := a.Values[key]
	return exists
}

// Remove removes an annotation
func (a *Annotations) Remove(key string) {
	delete(a.Values, key)
}

// GetKeys returns all annotation keys
func (a *Annotations) GetKeys() []string {
	keys := make([]string, 0, len(a.Values))
	for k := range a.Values {
		keys = append(keys, k)
	}
	return keys
}

// ToMap converts annotations to a map for backward compatibility
func (a *Annotations) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range a.Values {
		switch v.Type {
		case "string":
			result[k] = v.StringValue
		case "int":
			result[k] = v.IntValue
		case "float":
			result[k] = v.FloatValue
		case "bool":
			result[k] = v.BoolValue
		default:
			result[k] = nil
		}
	}
	return result
}

// FromMap converts a map to annotations for backward compatibility
func AnnotationsFromMap(data map[string]interface{}) *Annotations {
	annotations := NewAnnotations()
	for k, v := range data {
		switch val := v.(type) {
		case string:
			annotations.SetString(k, val)
		case int:
			annotations.SetInt(k, val)
		case float64:
			annotations.SetFloat(k, val)
		case bool:
			annotations.SetBool(k, val)
		default:
			// For any other type, store as string representation
			annotations.SetString(k, fmt.Sprintf("%v", val))
		}
	}
	return annotations
}
