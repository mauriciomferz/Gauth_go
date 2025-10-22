package harnesslib

import (
	"strings"
	"testing"
)

func TestSymbolLocationsMarkdownDedup(t *testing.T) {
	r := Report{GeneratedAt: "2025-01-01T00:00:00Z", Evidence: map[string][]string{
		"Foo": {"a.go:10", "a.go:10", "b.go:20"},
		"Bar": {"x.go:5"},
	}}
	md := r.ToSymbolLocationsMarkdown()
	// Expect dedup count for Foo to be 2
	if !contains(md, "| Foo | 2 |") {
		t.Fatalf("expected Foo dedup count 2, got markdown: %s", md)
	}
	// Ensure duplicate location not repeated more than once
	if countSubstring(md, "a.go:10") != 1 {
		t.Fatalf("expected a.go:10 to appear once after dedup, got markdown: %s", md)
	}
}

func TestSymbolLocationsOrdering(t *testing.T) {
	r := Report{GeneratedAt: "2025-01-01T00:00:00Z", Evidence: map[string][]string{
		"Zeta":  {"z.go:1"},
		"Alpha": {"a.go:1"},
	}}
	md := r.ToSymbolLocationsMarkdown()
	// Alpha should appear before Zeta in table ordering
	aIdx := strings.Index(md, "| Alpha |")
	zIdx := strings.Index(md, "| Zeta |")
	if aIdx == -1 || zIdx == -1 || aIdx > zIdx {
		t.Fatalf("expected Alpha before Zeta; aIdx=%d zIdx=%d\n%s", aIdx, zIdx, md)
	}
}

// helpers
func contains(s, sub string) bool { return strings.Contains(s, sub) }

func countSubstring(s, sub string) int {
	c, idx := 0, 0
	for {
		i := strings.Index(s[idx:], sub)
		if i == -1 {
			break
		}
		c++
		idx += i + len(sub)
	}
	return c
}
