package poa

import (
	"regexp"
	"strings"
	"testing"
)

// ScopeMatcher defines interface for checking if a request scope is allowed by held scope.
type ScopeMatcher interface {
	Matches(heldScope, requestedScope string) bool
}

// DefaultScopeMatcher implements simple wildcard matching and regex.
// Supports:
// - Exact match: "a:b" == "a:b"
// - Wildcard: "a:*" matches "a:b", "a:c"
// - Regex (if prefixed with "regex:"): "regex:^a:\d+$" matches "a:123"
type DefaultScopeMatcher struct{}

func (m *DefaultScopeMatcher) Matches(held, requested string) bool {
	// 1. Direct equality
	if held == requested {
		return true
	}

	// 2. Regex support
	if strings.HasPrefix(held, "regex:") {
		pattern := strings.TrimPrefix(held, "regex:")
		matched, _ := regexp.MatchString(pattern, requested)
		return matched
	}

	// 3. Simple Wildcard support (standard "glob" style for colons)
	// "resource:*" matches "resource:item"
	// "resource:*:read" matches "resource:123:read"
	// We convert wildcard to regex for simplicity.
	// Escape meta chars except '*'
	quoted := regexp.QuoteMeta(held)
	// Replace \* with .*? (non-greedy match) or [^:]+ for path segments?
	// AAP-002 usually implies hierarchy. Let's assume '*' matches anything in that segment.
	// For "resource:*" -> "resource:.*"
	regexStr := "^" + strings.ReplaceAll(quoted, "\\*", ".*") + "$"
	matched, _ := regexp.MatchString(regexStr, requested)
	return matched
}

func ValidateScopeNarrowing(heldScopes []string, requestedScope string) bool {
	matcher := &DefaultScopeMatcher{}
	for _, held := range heldScopes {
		if matcher.Matches(held, requestedScope) {
			return true
		}
	}
	return false
}

func TestScopeNarrowing_RFC115_C2(t *testing.T) {
	tests := []struct {
		name    string
		held    []string
		request string
		want    bool
	}{
		{
			name:    "Exact Match",
			held:    []string{"file:read"},
			request: "file:read",
			want:    true,
		},
		{
			name:    "Wildcard Suffix",
			held:    []string{"file:*"},
			request: "file:write",
			want:    true,
		},
		{
			name:    "Wildcard Middle",
			held:    []string{"file:*:read"},
			request: "file:report:read",
			want:    true,
		},
		{
			name:    "Regex Match Number",
			held:    []string{`regex:^order:\d+:view$`},
			request: "order:12345:view",
			want:    true,
		},
		{
			name:    "Regex Mismatch",
			held:    []string{`regex:^order:\d+:view$`},
			request: "order:abc:view",
			want:    false,
		},
		{
			name:    "No Match",
			held:    []string{"file:read"},
			request: "file:write",
			want:    false,
		},
		{
			name:    "Empty Held",
			held:    []string{},
			request: "file:read",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateScopeNarrowing(tt.held, tt.request)
			if got != tt.want {
				t.Errorf("ValidateScopeNarrowing() = %v, want %v", got, tt.want)
			}
		})
	}
}
