package harnesslib

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

type Clause struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	RFC      string `json:"aap"`
	LineFrom int    `json:"line_from"`
	LineTo   int    `json:"line_to"`
}

var headingRx = regexp.MustCompile(`^#{1,6} +(.+)`)

func ScanFile(path, rfc string) ([]Clause, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var clauses []Clause
	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		text := scanner.Text()
		m := headingRx.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		id := rfc + ":" + strings.ToLower(strings.ReplaceAll(strings.TrimSpace(m[1]), " ", "-"))
		clauses = append(clauses, Clause{ID: id, Title: m[1], RFC: rfc, LineFrom: line, LineTo: line})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return clauses, nil
}
