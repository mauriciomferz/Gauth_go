package authz

import (
	"testing"
	"time"
)

// in-memory store implementation for test
type memStore struct{ policies []Policy }

func (m *memStore) Load() ([]Policy, error) { return m.policies, nil }
func (m *memStore) LastModified() time.Time { return time.Now() }

func TestRegexPreValidationInvalidPattern(t *testing.T) {
	store := &memStore{policies: []Policy{
		{
			ID:       "bad",
			Subject:  "u",
			Resource: "r",
			Actions:  []string{"a"},
			Effect:   Allow,
			Conditions: []Condition{{
				Key:      "x",
				Operator: "regex",
				Values:   []string{"(unclosed"},
			}},
		},
	}}
	pa, err := NewPersistentAuthorizer(store, 1*time.Hour)
	if err != nil {
		t.Fatalf("new persistent authorizer error: %v", err)
	}
	// initial load performed in constructor; capture metrics snapshot
	snap := pa.GetMetricsSnapshot()
	if snap.RegexCompileErrors == 0 {
		t.Fatalf("expected compile error metric for invalid pattern")
	}
	if snap.RegexCompiles != 0 {
		t.Fatalf("expected no successful compiles, got %d", snap.RegexCompiles)
	}
}
