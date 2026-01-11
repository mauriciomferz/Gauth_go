package harnesslib

import (
	"os"
	"testing"
)

func TestScanFileHeadings(t *testing.T) {
	// Create a temporary markdown file.
	content := "# Title One\n## Sub Section\nPlain text\n### Deep Heading\n"
	f, err := os.CreateTemp(t.TempDir(), "rfc.md")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if _, err2 := f.WriteString(content); err2 != nil {
		t.Fatalf("write: %v", err2)
	}
	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}

	clauses, err := ScanFile(f.Name(), "0111")
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if len(clauses) != 3 {
		t.Fatalf("expected 3 clauses, got %d", len(clauses))
	}
	// Ensure IDs are lowercased and hyphenated.
	expected := []string{"0111:title-one", "0111:sub-section", "0111:deep-heading"}
	for i, id := range expected {
		if clauses[i].ID != id {
			t.Errorf("clause %d id mismatch: want %s got %s", i, id, clauses[i].ID)
		}
	}
	// Line numbers sanity: ascending and non-zero.
	prev := 0
	for _, c := range clauses {
		if c.LineFrom < prev {
			t.Errorf("line order violation: %d after %d", c.LineFrom, prev)
		}
		prev = c.LineFrom
	}
}
