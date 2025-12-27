//go:build go1.18

package anchor

import (
	"os"
	"testing"
)

// FuzzAnchorPersistence tests the EnablePersistence logic (which loads JSON) for robustness.
func FuzzAnchorPersistence(f *testing.F) {
	// Seed with valid JSON
	f.Add([]byte(`{
		"anchors": [
			{"hash": "abc", "anchored_at": "2023-01-01T00:00:00Z", "provider": "memory"}
		]
	}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpfile, err := os.CreateTemp("", "anchor_fuzz_*.json")
		if err != nil {
			return
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(data); err != nil {
			return
		}
		if err := tmpfile.Close(); err != nil {
			return
		}

		m := NewMemoryAnchor()
		_ = m.EnablePersistence(tmpfile.Name())
	})
}
