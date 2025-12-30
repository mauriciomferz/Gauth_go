package agentauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// DuplicateKeyError indicates a duplicate JSON key encountered while parsing.
type DuplicateKeyError struct {
	Key string
}

func (e *DuplicateKeyError) Error() string { return fmt.Sprintf("duplicate key: %s", e.Key) }

// ParseClaims parses a raw JSON object representing token claims, enforcing:
//   - duplicate key rejection
//   - maximum nesting depth (default 10)
//   - object top-level only (must begin with '{')
//
// Returns map[string]any with json.Number preserved for numbers.
func ParseClaims(data []byte) (map[string]any, error) {
	// Use custom byteReader implementing io.ByteReader to reduce allocation overhead.
	dec := json.NewDecoder(bytesReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return nil, errors.New("claims must be a JSON object")
	}
	claims := make(map[string]any)
	seen := make(map[string]struct{})
	for dec.More() {
		tKey, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tKey.(string)
		if !ok {
			return nil, errors.New("non-string key in claims object")
		}
		if _, exists := seen[key]; exists {
			return nil, &DuplicateKeyError{Key: key}
		}
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		if err := checkDepth(v, 0, 10); err != nil {
			return nil, fmt.Errorf("depth validation for key %s: %w", key, err)
		}
		claims[key] = v
		seen[key] = struct{}{}
	}
	// consume closing '}'
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	// Ensure no trailing non-whitespace content
	if dec.More() {
		return nil, errors.New("unexpected trailing content after claims object")
	}
	return claims, nil
}

// checkDepth ensures nesting depth does not exceed maxDepth.
func checkDepth(v any, current, maxDepth int) error {
	if current > maxDepth {
		return fmt.Errorf("max depth %d exceeded", maxDepth)
	}
	switch tv := v.(type) {
	case map[string]any:
		for _, inner := range tv {
			if err := checkDepth(inner, current+1, maxDepth); err != nil {
				return err
			}
		}
	case []any:
		for _, inner := range tv {
			if err := checkDepth(inner, current+1, maxDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

// bytesReader isolates *bytes.Reader creation to avoid importing bytes in tests inadvertently.
func bytesReader(b []byte) *byteReader { return &byteReader{b: b} }

// Minimal reader implementing io.Reader + io.ByteReader for json.Decoder; avoids extra allocations.
type byteReader struct {
	b []byte
	i int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
func (r *byteReader) ReadByte() (byte, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	c := r.b[r.i]
	r.i++
	return c, nil
}
