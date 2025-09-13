package events

import (
	"fmt"
	// "time" // Removed unused import
)

// SetStringSlice sets a slice of strings as multiple metadata entries
// using a prefix and index pattern: prefix.0, prefix.1, prefix.2, etc.
func (m *Metadata) SetStringSlice(key string, values []string) {
	for i, v := range values {
		m.SetString(fmt.Sprintf("%s.%d", key, i), v)
	}
	m.SetInt(fmt.Sprintf("%s.count", key), len(values))
}

// GetStringSlice retrieves a slice of strings from multiple metadata entries
func (m *Metadata) GetStringSlice(key string) ([]string, bool) {
	countKey := fmt.Sprintf("%s.count", key)
	count, ok := m.GetInt(countKey)
	if !ok {
		return nil, false
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		val, ok := m.GetString(fmt.Sprintf("%s.%d", key, i))
		if !ok {
			return nil, false
		}
		result[i] = val
	}

	return result, true
}
