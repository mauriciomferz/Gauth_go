package resource

// SetString sets a string value in the configuration
func (c *ResourceConfig) SetString(key, value string) {
	c.Settings[key] = ConfigValue{Type: "string", Data: value}
}

// SetInt sets an int value in the configuration
func (c *ResourceConfig) SetInt(key string, value int) {
	c.Settings[key] = ConfigValue{Type: "int", Data: value}
}

// SetFloat sets a float64 value in the configuration
func (c *ResourceConfig) SetFloat(key string, value float64) {
	c.Settings[key] = ConfigValue{Type: "float", Data: value}
}

// SetBool sets a bool value in the configuration
func (c *ResourceConfig) SetBool(key string, value bool) {
	c.Settings[key] = ConfigValue{Type: "bool", Data: value}
}

// SetMap sets a map[string]interface{} value in the configuration
func (c *ResourceConfig) SetMap(key string, value map[string]interface{}) {
	c.Settings[key] = ConfigValue{Type: "map", Data: value}
}

// GetMap retrieves a map[string]interface{} value from the configuration
func (c *ResourceConfig) GetMap(key string) (map[string]interface{}, bool) {
	if val, ok := c.Settings[key]; ok && val.Type == "map" {
		 if m, ok := val.Data.(map[string]interface{}); ok {
			  return m, true
		 }
	}
	return nil, false
}

// SetSlice sets a []interface{} value in the configuration
func (c *ResourceConfig) SetSlice(key string, value []interface{}) {
	c.Settings[key] = ConfigValue{Type: "slice", Data: value}
}

// GetSlice retrieves a []interface{} value from the configuration
func (c *ResourceConfig) GetSlice(key string) ([]interface{}, bool) {
	if val, ok := c.Settings[key]; ok && val.Type == "slice" {
		 if s, ok := val.Data.([]interface{}); ok {
			  return s, true
		 }
	}
	return nil, false
}

// Has checks if a key exists in the configuration
func (c *ResourceConfig) Has(key string) bool {
	_, ok := c.Settings[key]
	return ok
}

// Remove deletes a key from the configuration
func (c *ResourceConfig) Remove(key string) {
	delete(c.Settings, key)
}

// ConfigValue represents a typed value for resource configuration
type ConfigValue struct {
	Type string
	Data interface{}
}

// ResourceConfig represents a structured configuration for a resource
type ResourceConfig struct {
	// Settings holds configuration settings
	Settings map[string]ConfigValue
}

// NewResourceConfig creates a new empty resource configuration
func NewResourceConfig() *ResourceConfig {
	return &ResourceConfig{
		Settings: make(map[string]ConfigValue),
	}
}

// GetString retrieves a string value from the configuration
func (c *ResourceConfig) GetString(key string) (string, bool) {
	if val, ok := c.Settings[key]; ok && val.Type == "string" {
		if str, ok := val.Data.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetInt retrieves an integer value from the configuration
func (c *ResourceConfig) GetInt(key string) (int, bool) {
	if val, ok := c.Settings[key]; ok && val.Type == "int" {
		if i, ok := val.Data.(int); ok {
			return i, true
		}
	}
	return 0, false
}

// GetFloat retrieves a float value from the configuration
func (c *ResourceConfig) GetFloat(key string) (float64, bool) {
	if val, ok := c.Settings[key]; ok && val.Type == "float" {
		if f, ok := val.Data.(float64); ok {
			return f, true
		}
	}
	return 0, false
}

// GetBool retrieves a boolean value from the configuration
func (c *ResourceConfig) GetBool(key string) (bool, bool) {
	if val, ok := c.Settings[key]; ok && val.Type == "bool" {
		if b, ok := val.Data.(bool); ok {
			return b, true
		}
	}
	return false, false
}
