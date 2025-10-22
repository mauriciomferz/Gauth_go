package rfc

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// NormativeStatement represents a single MUST/SHOULD/etc line extracted from a block.
type NormativeStatement struct {
	Level string `json:"level"`
	Text  string `json:"text"`
	Line  int    `json:"line"`
}

// Clause represents a section or fragment of an RFC markdown file.
type Clause struct {
	RFC                 string               `json:"rfc"`
	SectionID           string               `json:"section_id"`
	FragmentID          string               `json:"fragment_id"`
	Title               string               `json:"title"`
	NormativeStatements []NormativeStatement `json:"normative_statements"`
	RawBlock            string               `json:"raw_block"`
	BlockHash           string               `json:"block_hash"`
	SourceFile          string               `json:"source_file"`
}

// Index is the top-level JSON output structure.
type Index struct {
	GeneratedAt time.Time `json:"generated_at"`
	Clauses     []Clause  `json:"clauses"`
}

var normativeRegex = regexp.MustCompile(`\b(MUST NOT|SHOULD NOT|MUST|SHOULD|MAY|REQUIRED)\b`)

// ParseRFCFile ingests a markdown RFC file and produces clause entries.
// Simplistic heuristic: headings starting with "##" start a new clause block.
func ParseRFCFile(path string, rfcID string) ([]Clause, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var clauses []Clause
	var currentLines []string
	var currentTitle string
	var sectionCounter int
	lineNo := 0
	var lineOffsets []int

	flush := func() {
		if len(currentLines) == 0 {
			return
		}
		raw := strings.Join(currentLines, "\n")
		h := sha256.Sum256([]byte("GAUTH-RFC-BLOCK:" + raw))
		clause := Clause{
			RFC:        rfcID,
			SectionID:  intToSection(sectionCounter),
			FragmentID: intToSection(sectionCounter),
			Title:      currentTitle,
			RawBlock:   raw,
			BlockHash:  "sha256-" + hex.EncodeToString(h[:]),
			SourceFile: path[strings.LastIndex(path, "/")+1:],
		}
		// Extract normative statements
		startLine := lineOffsets[0]
		for i, l := range currentLines {
			if normativeRegex.MatchString(l) {
				clause.NormativeStatements = append(clause.NormativeStatements, NormativeStatement{
					Level: normativeRegex.FindString(l),
					Text:  strings.TrimSpace(l),
					Line:  startLine + i,
				})
			}
		}
		clauses = append(clauses, clause)
		currentLines = nil
		lineOffsets = nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++
		// Normalize trailing whitespace
		line = strings.TrimRight(line, " \t")
		if strings.HasPrefix(line, "## ") { // new clause
			flush()
			sectionCounter++
			currentTitle = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentLines = []string{line}
			lineOffsets = []int{lineNo}
			continue
		}
		if currentLines != nil {
			currentLines = append(currentLines, line)
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return clauses, nil
}

func intToSection(i int) string {
	return fmt.Sprintf("%d", i)
}
