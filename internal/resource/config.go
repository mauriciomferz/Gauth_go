package resource

import "sync"

// Config represents a configuration for a resource
type Config struct {
	mu     sync.RWMutex
	values map[string]interface{}
}

// NewResourceConfig creates a new empty resource configuration
func NewResourceConfig() *Config {
	return &Config{
		values: make(map[string]interface{}),
	}
}

// GetString retrieves a string value from the configuration
func (c *Config) GetString(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

// SetString sets a string value in the configuration
func (c *Config) SetString(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// GetInt retrieves an integer value from the configuration
func (c *Config) GetInt(key string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[key]; ok {
		if i, ok := val.(int); ok {
			return i, true
		}
	}
	return 0, false
}

// SetInt sets an integer value in the configuration
func (c *Config) SetInt(key string, value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// GetFloat retrieves a float value from the configuration
func (c *Config) GetFloat(key string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[key]; ok {
		if f, ok := val.(float64); ok {
			return f, true
		}
	}
	return 0, false
}

// SetFloat sets a float value in the configuration
func (c *Config) SetFloat(key string, value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// GetBool retrieves a boolean value from the configuration
func (c *Config) GetBool(key string) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[key]; ok {
		if b, ok := val.(bool); ok {
			return b, true
		}
	}
	return false, false
}

// SetBool sets a boolean value in the configuration
func (c *Config) SetBool(key string, value bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// GetMap retrieves a map value from the configuration
func (c *Config) GetMap(key string) (map[string]interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m, true
		}
	}
	return nil, false
}

// SetMap sets a map value in the configuration
func (c *Config) SetMap(key string, value map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// GetSlice retrieves a slice value from the configuration
func (c *Config) GetSlice(key string) ([]interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[key]; ok {
		if s, ok := val.([]interface{}); ok {
			return s, true
		}
	}
	return nil, false
}

// SetSlice sets a slice value in the configuration
func (c *Config) SetSlice(key string, value []interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// Has checks if a key exists in the configuration
func (c *Config) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.values[key]
	return exists
}

// Remove removes a key from the configuration
func (c *Config) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.values, key)
}

// GetKeys returns all keys in the configuration
func (c *Config) GetKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	return keys
}
