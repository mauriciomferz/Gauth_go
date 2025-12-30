package poa

import (
	"strings"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/rfc/errs"
)

func TestErrorTaxonomy(t *testing.T) {
	// Test standard error construction
	err := NewError(ErrCodeValidation, "invalid input")

	// Verify code mapping
	if err.Code != string(errs.ErrInvalidRequest) {
		t.Errorf("Expected code %q, got %q", errs.ErrInvalidRequest, err.Code)
	}

	// Verify conversion to RFCError
	rfcErr := err.AsRFCError()
	if rfcErr.Code != errs.ErrInvalidRequest {
		t.Errorf("Expected RFC code %q, got %q", errs.ErrInvalidRequest, rfcErr.Code)
	}
	if rfcErr.Message != "invalid input" {
		t.Errorf("Expected message %q, got %q", "invalid input", rfcErr.Message)
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
