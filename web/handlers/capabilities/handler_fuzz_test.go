//go:build go1.18

package capabilities

import (
	"os"
	"testing"
)

// FuzzCapabilityLoader tests the LoadFromFile logic by feeding it fuzzed JSON bytes via a temporary file.
func FuzzCapabilityLoader(f *testing.F) {
	// Seed with valid JSON
	f.Add([]byte(`{
		"schema_version": 1,
		"capabilities": [
			{"id": "cap1", "version": "1.0", "stable": true}
		],
		"action_mappings": {
			"read": ["cap1"]
		}
	}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Use a temporary file since LoadFromFile expects a path
		tmpfile, err := os.CreateTemp("", "cap_fuzz_*.json")
		if err != nil {
			return
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()

		if _, err := tmpfile.Write(data); err != nil {
			return
		}
		if err := tmpfile.Close(); err != nil {
			return
		}

		h := NewHandler()
		_ = h.LoadFromFile(tmpfile.Name())
	})
}
