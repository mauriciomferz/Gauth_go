package poa

import (
	"strings"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/aap/errs"
)

func TestErrorTaxonomy(t *testing.T) {
	// Test standard error construction
	err := NewError(ErrCodeValidation, "invalid input")

	// Verify code mapping
	if err.Code != string(errs.ErrInvalidRequest) {
		t.Errorf("Expected code %q, got %q", errs.ErrInvalidRequest, err.Code)
	}

	// Verify conversion to RFCError
	aapErr := err.AsRFCError()
	if aapErr.Code != errs.ErrInvalidRequest {
		t.Errorf("Expected RFC code %q, got %q", errs.ErrInvalidRequest, aapErr.Code)
	}
	if aapErr.Message != "invalid input" {
		t.Errorf("Expected message %q, got %q", "invalid input", aapErr.Message)
	}

	// Test wrapping
	cause := NewError("internal", "db fail")
	wrapped := WrapError(ErrCodeInternal, "operation failed", cause)

	if wrapped.Unwrap() != cause {
		t.Error("Unwrap mismatch")
	}

	// Verify formatting
	msg := wrapped.Error()
	if !strings.Contains(msg, "operation failed") || !strings.Contains(msg, "db fail") {
		t.Errorf("Error string %q does not contain expected messages", msg)
	}
}
