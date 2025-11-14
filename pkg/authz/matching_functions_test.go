package authz

import (
	"testing"
)

// TestMatchesResource tests the matchesResource function for various patterns
func TestMatchesResource(t *testing.T) {
	ma := NewMemoryAuthorizer()

	tests := []struct {
		name     string
		resource string
		pattern  string
		expected bool
	}{
		// Wildcard tests
		{
			name:     "wildcard matches any resource",
			resource: "documents/secret/file.txt",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "wildcard matches empty string",
			resource: "",
			pattern:  "*",
			expected: true,
		},

		// Exact match tests
		{
			name:     "exact match success",
			resource: "documents/file.txt",
			pattern:  "documents/file.txt",
			expected: true,
		},
		{
			name:     "exact match failure - different resource",
			resource: "documents/file.txt",
			pattern:  "documents/other.txt",
			expected: false,
		},
		{
			name:     "exact match failure - case sensitive",
			resource: "documents/FILE.txt",
			pattern:  "documents/file.txt",
			expected: false,
		},
		{
			name:     "exact match empty strings",
			resource: "",
			pattern:  "",
			expected: true,
		},

		// Prefix wildcard tests
		{
			name:     "prefix wildcard matches direct child",
			resource: "documents/file.txt",
			pattern:  "documents/*",
			expected: true,
		},
		{
			name:     "prefix wildcard matches nested path",
			resource: "documents/secret/confidential/file.txt",
			pattern:  "documents/*",
			expected: true,
		},
		{
			name:     "prefix wildcard matches exact prefix",
			resource: "documents/",
			pattern:  "documents/*",
			expected: true,
		},
		{
			name:     "prefix wildcard does not match different path",
			resource: "files/document.txt",
			pattern:  "documents/*",
			expected: false,
		},
		{
			name:     "prefix wildcard does not match similar prefix",
			resource: "documents_backup/file.txt",
			pattern:  "documents/*",
			expected: false,
		},
		{
			name:     "prefix wildcard empty prefix matches all",
			resource: "anything/here",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "prefix wildcard with partial match",
			resource: "api/v1/users/123",
			pattern:  "api/v1/*",
			expected: true,
		},
		{
			name:     "prefix wildcard no match - shorter resource",
			resource: "api",
			pattern:  "api/v1/*",
			expected: false,
		},
		{
			name:     "prefix wildcard matches resource starting with prefix",
			resource: "s3://bucket/key",
			pattern:  "s3://*",
			expected: true,
		},
		{
			name:     "prefix wildcard matches empty string after prefix",
			resource: "prefix",
			pattern:  "prefix*",
			expected: true,
		},

		// Edge cases
		{
			name:     "resource longer than pattern",
			resource: "very/long/path/to/resource",
			pattern:  "short",
			expected: false,
		},
		{
			name:     "pattern longer than resource",
			resource: "short",
			pattern:  "very/long/pattern",
			expected: false,
		},
		{
			name:     "special characters in exact match",
			resource: "file-name_with.special$chars",
			pattern:  "file-name_with.special$chars",
			expected: true,
		},
		{
			name:     "special characters in prefix wildcard",
			resource: "file-name_with.special$chars/subpath",
			pattern:  "file-name_with.special$chars/*",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ma.matchesResource(tt.resource, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesResource(%q, %q) = %v, expected %v",
					tt.resource, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestMatchesSubject tests the matchesSubject function
func TestMatchesSubject(t *testing.T) {
	ma := NewMemoryAuthorizer()

	tests := []struct {
		name     string
		subject  string
		pattern  string
		expected bool
	}{
		// Wildcard tests
		{
			name:     "wildcard matches any subject",
			subject:  "user@example.com",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "wildcard matches empty string",
			subject:  "",
			pattern:  "*",
			expected: true,
		},

		// Exact match tests
		{
			name:     "exact match success",
			subject:  "user@example.com",
			pattern:  "user@example.com",
			expected: true,
		},
		{
			name:     "exact match failure",
			subject:  "user@example.com",
			pattern:  "admin@example.com",
			expected: false,
		},
		{
			name:     "exact match case sensitive",
			subject:  "User@Example.com",
			pattern:  "user@example.com",
			expected: false,
		},
		{
			name:     "exact match empty strings",
			subject:  "",
			pattern:  "",
			expected: true,
		},
		{
			name:     "exact match with special characters",
			subject:  "service-account-123",
			pattern:  "service-account-123",
			expected: true,
		},
		{
			name:     "exact match with UUID",
			subject:  "550e8400-e29b-41d4-a716-446655440000",
			pattern:  "550e8400-e29b-41d4-a716-446655440000",
			expected: true,
		},

		// Non-match tests
		{
			name:     "different subjects do not match",
			subject:  "alice",
			pattern:  "bob",
			expected: false,
		},
		{
			name:     "partial match does not succeed",
			subject:  "user@example.com",
			pattern:  "user@example",
			expected: false,
		},
		{
			name:     "subject longer than pattern",
			subject:  "very-long-subject-name",
			pattern:  "short",
			expected: false,
		},
		{
			name:     "pattern longer than subject",
			subject:  "short",
			pattern:  "very-long-pattern-name",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ma.matchesSubject(tt.subject, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesSubject(%q, %q) = %v, expected %v",
					tt.subject, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestMatchesAction tests the matchesAction function
func TestMatchesAction(t *testing.T) {
	ma := NewMemoryAuthorizer()

	tests := []struct {
		name     string
		action   string
		actions  []string
		expected bool
	}{
		// Wildcard tests
		{
			name:     "wildcard in actions matches any action",
			action:   "read",
			actions:  []string{"*"},
			expected: true,
		},
		{
			name:     "wildcard matches empty action",
			action:   "",
			actions:  []string{"*"},
			expected: true,
		},
		{
			name:     "wildcard in list with other actions",
			action:   "delete",
			actions:  []string{"read", "write", "*"},
			expected: true,
		},

		// Single action match tests
		{
			name:     "exact match with single action",
			action:   "read",
			actions:  []string{"read"},
			expected: true,
		},
		{
			name:     "no match with single action",
			action:   "delete",
			actions:  []string{"read"},
			expected: false,
		},

		// Multiple actions tests
		{
			name:     "match first action in list",
			action:   "read",
			actions:  []string{"read", "write", "delete"},
			expected: true,
		},
		{
			name:     "match middle action in list",
			action:   "write",
			actions:  []string{"read", "write", "delete"},
			expected: true,
		},
		{
			name:     "match last action in list",
			action:   "delete",
			actions:  []string{"read", "write", "delete"},
			expected: true,
		},
		{
			name:     "no match in multiple actions",
			action:   "execute",
			actions:  []string{"read", "write", "delete"},
			expected: false,
		},

		// Empty actions tests
		{
			name:     "empty actions list does not match",
			action:   "read",
			actions:  []string{},
			expected: false,
		},
		{
			name:     "empty action against empty list",
			action:   "",
			actions:  []string{},
			expected: false,
		},
		{
			name:     "empty action matches in list",
			action:   "",
			actions:  []string{"", "read"},
			expected: true,
		},

		// Case sensitivity tests
		{
			name:     "case sensitive - no match",
			action:   "Read",
			actions:  []string{"read"},
			expected: false,
		},
		{
			name:     "case sensitive - exact match",
			action:   "READ",
			actions:  []string{"READ"},
			expected: true,
		},

		// Special characters tests
		{
			name:     "action with colon",
			action:   "s3:GetObject",
			actions:  []string{"s3:GetObject", "s3:PutObject"},
			expected: true,
		},
		{
			name:     "action with hyphen",
			action:   "get-data",
			actions:  []string{"get-data"},
			expected: true,
		},
		{
			name:     "action with underscore",
			action:   "list_users",
			actions:  []string{"list_users", "create_user"},
			expected: true,
		},

		// Duplicate actions tests
		{
			name:     "duplicate actions in list",
			action:   "read",
			actions:  []string{"read", "write", "read"},
			expected: true,
		},

		// Long action lists
		{
			name:     "match in long list",
			action:   "action5",
			actions:  []string{"action1", "action2", "action3", "action4", "action5", "action6", "action7"},
			expected: true,
		},
		{
			name:     "no match in long list",
			action:   "action99",
			actions:  []string{"action1", "action2", "action3", "action4", "action5"},
			expected: false,
		},

		// Edge cases
		{
			name:     "action with spaces (no match expected)",
			action:   "read file",
			actions:  []string{"read", "file"},
			expected: false,
		},
		{
			name:     "action with spaces exact match",
			action:   "read file",
			actions:  []string{"read file"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ma.matchesAction(tt.action, tt.actions)
			if result != tt.expected {
				t.Errorf("matchesAction(%q, %v) = %v, expected %v",
					tt.action, tt.actions, result, tt.expected)
			}
		})
	}
}
