// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: MIT

package gauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// SecureJSONParser provides security-hardened JSON parsing with depth limits,
// size validation, and UTF-8 validation to prevent DOS attacks and memory exhaustion.
// Completes sec1.item3 (Robust JSON parsing) by adding explicit security controls
// around encoding/json standard library.
type SecureJSONParser struct {
	// MaxDepth limits JSON nesting depth to prevent stack overflow attacks (default: 32)
	MaxDepth int
	// MaxSize limits JSON payload size in bytes to prevent memory exhaustion (default: 1MB)
	MaxSize int
	// StrictUnknownFields rejects JSON with unknown fields when enabled
	StrictUnknownFields bool
	// ValidateUTF8 enforces UTF-8 validation on all string values
	ValidateUTF8 bool
}

// DefaultSecureParser returns a SecureJSONParser with recommended security defaults
func DefaultSecureParser() *SecureJSONParser {
	return &SecureJSONParser{
		MaxDepth:            32,          // Recommended max nesting depth
		MaxSize:             1024 * 1024, // 1MB max payload
		StrictUnknownFields: false,       // Backward compatible default
		ValidateUTF8:        true,        // Always validate UTF-8
	}
}

// ParseSecure parses JSON data with security hardening:
// - Depth limit validation (prevent deeply nested JSON DOS)
// - Size validation (prevent memory exhaustion)
// - UTF-8 validation (reject invalid unicode)
// - Optional strict unknown field rejection
//
// Returns error if any security constraint is violated.
func (p *SecureJSONParser) ParseSecure(data []byte, v interface{}) error {
	// Size validation
	if len(data) > p.MaxSize {
		return fmt.Errorf("JSON payload exceeds max size: %d > %d bytes", len(data), p.MaxSize)
	}

	// UTF-8 validation (entire payload must be valid UTF-8)
	if p.ValidateUTF8 && !utf8.Valid(data) {
		return fmt.Errorf("invalid UTF-8 in JSON payload")
	}

	// Depth validation (count max nesting level)
	depth := computeMaxDepth(data)
	if depth > p.MaxDepth {
		return fmt.Errorf("JSON nesting depth exceeds limit: %d > %d", depth, p.MaxDepth)
	}

	// Create decoder with optional strict mode
	decoder := json.NewDecoder(bytes.NewReader(data))
	if p.StrictUnknownFields {
		decoder.DisallowUnknownFields()
	}

	// Decode JSON
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("JSON decode failed: %w", err)
	}

	return nil
}

// computeMaxDepth calculates the maximum nesting depth of JSON data.
// This is a simple state machine parser that tracks bracket/brace depth without full parsing.
func computeMaxDepth(data []byte) int {
	maxDepth := 0
	currentDepth := 0
	inString := false
	escaped := false

	for i := 0; i < len(data); i++ {
		c := data[i]

		// Handle string escapes
		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		// Handle string boundaries
		if c == '"' {
			inString = !inString
			continue
		}

		// Only track depth outside strings
		if !inString {
			switch c {
			case '{', '[':
				currentDepth++
				if currentDepth > maxDepth {
					maxDepth = currentDepth
				}
			case '}', ']':
				currentDepth--
				if currentDepth < 0 {
					// Malformed JSON (closing bracket without opening)
					return maxDepth
				}
			}
		}
	}

	return maxDepth
}

// ValidateJSONSecurity performs comprehensive security validation on JSON data
// without full parsing. Returns error if any security constraint is violated.
// This is a fast pre-check before calling ParseSecure.
func ValidateJSONSecurity(data []byte, maxDepth, maxSize int) error {
	// Size check
	if len(data) > maxSize {
		return fmt.Errorf("JSON payload exceeds max size: %d > %d bytes", len(data), maxSize)
	}

	// UTF-8 validation
	if !utf8.Valid(data) {
		return fmt.Errorf("invalid UTF-8 in JSON payload")
	}

	// Depth check
	depth := computeMaxDepth(data)
	if depth > maxDepth {
		return fmt.Errorf("JSON nesting depth exceeds limit: %d > %d", depth, maxDepth)
	}

	return nil
}
