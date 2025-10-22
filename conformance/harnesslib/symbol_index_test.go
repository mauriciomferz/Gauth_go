package harnesslib

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBuildSymbolIndexCoreSymbols ensures the symbol index includes key implementation symbols.
func TestBuildSymbolIndexCoreSymbols(t *testing.T) {
	// Determine repo root (same logic as Analyze chooses). Start at current working dir and ascend until go.mod found.
	cwd, _ := filepath.Abs(".")
	root := cwd
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		parent := filepath.Dir(root)
		if _, err2 := os.Stat(filepath.Join(parent, "go.mod")); err2 == nil {
			root = parent
		} else {
			gp := filepath.Dir(parent)
			if _, err3 := os.Stat(filepath.Join(gp, "go.mod")); err3 == nil {
				root = gp
			}
		}
	}

	symbols, errs := BuildSymbolIndex(root)
	if len(errs) > 0 {
		t.Logf("encountered parse/read errors: %v", errs)
	}
	// Required symbols list may grow; start with two representative ones.
	required := []string{"CanonicalPOADigest", "PowerOfAttorney"}
	for _, r := range required {
		if _, ok := symbols[r]; !ok {
			t.Errorf("expected symbol %s to be indexed", r)
		}
	}
	// Basic sanity: indexing should discover more than trivial harness files.
	if len(symbols) < 30 { // adjusted threshold (tests excluded by default)
		t.Errorf("unexpectedly few symbols indexed: %d", len(symbols))
	}
}
