package auth

import "fmt"

// ClaimType represents the type of a claim value
type ClaimType interface {
	~string | ~int64 | ~float64 | ~bool | []string | []int64 | []float64
}

// ClaimValue represents a strongly typed claim value
type ClaimValue struct {
	Type    string
	String  string
	Int     int64
	Float   float64
	Bool    bool
	Strings []string
	Ints    []int64
	Floats  []float64
}

// Claims represents a map of strongly typed claims
type Claims struct {
	values map[string]ClaimValue
}

// NewClaims creates a new Claims instance
func NewClaims() *Claims {
	return &Claims{
		values: make(map[string]ClaimValue),
	}
}

// Set sets a claim value with type checking
func (c *Claims) Set(key string, value interface{}) error {
	switch v := value.(type) {
	case string:
		c.values[key] = ClaimValue{Type: "string", String: v}
	case int64:
		c.values[key] = ClaimValue{Type: "int", Int: v}
	case float64:
		c.values[key] = ClaimValue{Type: "float", Float: v}
	case bool:
		c.values[key] = ClaimValue{Type: "bool", Bool: v}
	case []string:
		c.values[key] = ClaimValue{Type: "strings", Strings: v}
	case []int64:
		c.values[key] = ClaimValue{Type: "ints", Ints: v}
	case []float64:
		c.values[key] = ClaimValue{Type: "floats", Floats: v}
	default:
		return fmt.Errorf("unsupported claim type for key %s", key)
	}
	return nil
}

// GetString gets a string claim value
func (c *Claims) GetString(key string) (string, error) {
	if v, ok := c.values[key]; ok && v.Type == "string" {
		return v.String, nil
	}
	return "", fmt.Errorf("claim %s not found or not a string", key)
}

// GetInt gets an int64 claim value
func (c *Claims) GetInt(key string) (int64, error) {
	if v, ok := c.values[key]; ok && v.Type == "int" {
		return v.Int, nil
	}
	return 0, fmt.Errorf("claim %s not found or not an int", key)
}

// GetFloat gets a float64 claim value
func (c *Claims) GetFloat(key string) (float64, error) {
	if v, ok := c.values[key]; ok && v.Type == "float" {
		return v.Float, nil
	}
	return 0, fmt.Errorf("claim %s not found or not a float", key)
}

// GetBool gets a boolean claim value
func (c *Claims) GetBool(key string) (bool, error) {
	if v, ok := c.values[key]; ok && v.Type == "bool" {
		return v.Bool, nil
	}
	return false, fmt.Errorf("claim %s not found or not a bool", key)
}

// GetStringSlice gets a string slice claim value
func (c *Claims) GetStringSlice(key string) ([]string, error) {
	if v, ok := c.values[key]; ok && v.Type == "strings" {
		return v.Strings, nil
	}
	return nil, fmt.Errorf("claim %s not found or not a string slice", key)
}

// GetIntSlice gets an int64 slice claim value
func (c *Claims) GetIntSlice(key string) ([]int64, error) {
	if v, ok := c.values[key]; ok && v.Type == "ints" {
		return v.Ints, nil
	}
	return nil, fmt.Errorf("claim %s not found or not an int slice", key)
}

// GetFloatSlice gets a float64 slice claim value
func (c *Claims) GetFloatSlice(key string) ([]float64, error) {
	if v, ok := c.values[key]; ok && v.Type == "floats" {
		return v.Floats, nil
	}
	return nil, fmt.Errorf("claim %s not found or not a float slice", key)
}

// Has checks if a claim exists
func (c *Claims) Has(key string) bool {
	_, ok := c.values[key]
	return ok
}

// Keys returns all claim keys
func (c *Claims) Keys() []string {
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	return keys
}

// Delete removes a claim
func (c *Claims) Delete(key string) {
	delete(c.values, key)
}

// Clear removes all claims
func (c *Claims) Clear() {
	c.values = make(map[string]ClaimValue)
}

// Len returns the number of claims
func (c *Claims) Len() int {
	return len(c.values)
}
