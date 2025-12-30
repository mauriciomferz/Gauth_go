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

	"github.com/mauriciomferz/AgentAuth/internal/aap"
	"github.com/pkg/errors"
)

func main() {
	out := flag.String("out", "docs/aap/aap_clause_index.json", "output file path")
	base := flag.String("dir", "docs/aap", "directory containing RFC markdown files")
	flag.Parse()

	files := []struct{ Name, ID string }{
		{"agentauth_aap_001.md", "0111"},
		{"aap002.md", "0115"},
	}

	var all []aap.Clause
	for _, f := range files {
		path := filepath.Join(*base, f.Name)
		clauses, err := aap.ParseRFCFile(path, f.ID)
		if err != nil {
			log.Fatalf("parse %s: %v", path, errors.WithStack(err))
		}
		all = append(all, clauses...)
	}

	idx := aap.Index{GeneratedAt: time.Now().UTC(), Clauses: all}
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		log.Fatalf("marshal index: %v", err)
	}
	if err := os.WriteFile(*out, b, 0o600); err != nil {
		log.Fatalf("write index: %v", err)
	}
	fmt.Printf("wrote clause index: %s (clauses=%d)\n", *out, len(all))
}
