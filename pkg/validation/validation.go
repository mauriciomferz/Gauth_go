package validation

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Common validation literals.
const requiredValue = "required"

// Claims represents a minimal token claims structure used for validation scaffolding.
type Claims struct {
	Subject   string
	Issuer    string
	Audience  []string
	ExpiresAt time.Time
	Scopes    []string
}

// InputError aggregates multiple field validation failures.
type InputError struct {
	FieldErrors map[string]string
}

func (e *InputError) Error() string { return fmt.Sprintf("validation failed: %v", e.FieldErrors) }

// Validator defines validation operations for inputs & claims.
type Validator interface {
	ValidateClaims(c Claims) error
	ValidateString(name, value string, min, max int) error
}

// BasicValidator is a simple implementation used in early phases.
type BasicValidator struct{}

func NewBasicValidator() *BasicValidator { return &BasicValidator{} }

func (v *BasicValidator) ValidateClaims(c Claims) error {
	fe := map[string]string{}
	if c.Subject == "" {
		fe["subject"] = requiredValue
	}
	if c.Issuer == "" {
		fe["issuer"] = requiredValue
	}
	if len(c.Audience) == 0 {
		fe["audience"] = "must have at least one"
	}
	if time.Until(c.ExpiresAt) <= 0 {
		fe["expiresAt"] = "must be in the future"
	}
	if len(fe) > 0 {
		return &InputError{FieldErrors: fe}
	}
	return nil
}

func (v *BasicValidator) ValidateString(name, value string, min, max int) error {
	l := len(value)
	if l < min || (max > 0 && l > max) {
		return fmt.Errorf("%s length %d outside range [%d,%d]", name, l, min, max)
	}
	return nil
}

// SafeSlice safely truncates a string to at most n runes (not bytes) with ellipsis indication.
func SafeSlice(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

// CombineErrors merges multiple errors into one (very simple implementation).
func CombineErrors(errs ...error) error {
	var parts []string
	for _, e := range errs {
		if e != nil {
			parts = append(parts, e.Error())
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return errors.New(strings.Join(parts, "; "))
}
