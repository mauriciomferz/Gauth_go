// Package main indexes RFC clause definitions across markdown files into a single
// JSON artifact used by the conformance harness for clause-to-test mapping.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/rfc"
	"github.com/pkg/errors"
)

func main() {
	out := flag.String("out", "docs/rfc/rfc_clause_index.json", "output file path")
	base := flag.String("dir", "docs/rfc", "directory containing RFC markdown files")
	flag.Parse()

	files := []struct{ Name, ID string }{
		{"gauth_aap_001.md", "0111"},
		{"aap002.md", "0115"},
	}

	var all []rfc.Clause
	for _, f := range files {
		path := filepath.Join(*base, f.Name)
		clauses, err := rfc.ParseRFCFile(path, f.ID)
		if err != nil {
			log.Fatalf("parse %s: %v", path, errors.WithStack(err))
		}
		all = append(all, clauses...)
	}

	idx := rfc.Index{GeneratedAt: time.Now().UTC(), Clauses: all}
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		log.Fatalf("marshal index: %v", err)
	}
	if err := os.WriteFile(*out, b, 0o600); err != nil {
		log.Fatalf("write index: %v", err)
	}
	fmt.Printf("wrote clause index: %s (clauses=%d)\n", *out, len(all))
}
