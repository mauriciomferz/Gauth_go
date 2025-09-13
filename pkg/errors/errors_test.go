package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestErrorCreation(t *testing.T) {
	// Test basic error creation
	err := New(ErrInvalidToken, "Invalid token format")
	if err.Code != ErrInvalidToken {
		t.Errorf("Expected code %s, got %s", ErrInvalidToken, err.Code)
	}
	if err.Message != "Invalid token format" {
		t.Errorf("Expected message 'Invalid token format', got '%s'", err.Message)
	}
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
	if err.Details.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestErrorMethods(t *testing.T) {
	// Test method chaining and field setting
	baseErr := stderrors.New("underlying error")
	err := New(ErrServerError, "Server error occurred")
	err = err.WithSource(SourceToken)
	err = err.WithCause(baseErr)
	err = err.WithRequestInfo("req-123", "client-456", "user-789")
	err = err.WithHTTPInfo("/api/resource", "GET", http.StatusInternalServerError, "192.168.1.1")
	err = err.AddInfo("retry_after", "60")

	// Check fields
	if err.Source != SourceToken {
		t.Errorf("Expected source %s, got %s", SourceToken, err.Source)
	}
	if err.Cause != baseErr {
		t.Errorf("Expected cause to be set correctly")
	}
	if err.Details.RequestID != "req-123" {
		t.Errorf("Expected RequestID 'req-123', got '%s'", err.Details.RequestID)
	}
	if err.Details.ClientID != "client-456" {
		t.Errorf("Expected ClientID 'client-456', got '%s'", err.Details.ClientID)
	}
	if err.Details.UserID != "user-789" {
		t.Errorf("Expected UserID 'user-789', got '%s'", err.Details.UserID)
	}
	if err.Details.Path != "/api/resource" {
		t.Errorf("Expected Path '/api/resource', got '%s'", err.Details.Path)
	}
	if err.Details.Method != "GET" {
		t.Errorf("Expected Method 'GET', got '%s'", err.Details.Method)
	}
	if err.Details.HTTPStatusCode != http.StatusInternalServerError {
		t.Errorf("Expected HTTPStatusCode %d, got %d", http.StatusInternalServerError, err.Details.HTTPStatusCode)
	}
	if err.Details.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IPAddress '192.168.1.1', got '%s'", err.Details.IPAddress)
	}
	if v, ok := err.Details.AdditionalInfo["retry_after"]; !ok || v != "60" {
		t.Errorf("Expected AdditionalInfo['retry_after'] = '60', got '%s'", v)
	}
}

func TestErrorString(t *testing.T) {
	// Test without cause
	err1 := New(ErrRateLimited, "Rate limit exceeded")
	expected1 := "rate_limited: Rate limit exceeded"
	if err1.Error() != expected1 {
		t.Errorf("Expected '%s', got '%s'", expected1, err1.Error())
	}

	// Test with cause
	cause := fmt.Errorf("underlying cause")
	err2 := New(ErrRateLimited, "Rate limit exceeded").WithCause(cause)
	expected2 := "rate_limited: Rate limit exceeded: underlying cause"
	if err2.Error() != expected2 {
		t.Errorf("Expected '%s', got '%s'", expected2, err2.Error())
	}
}

func TestWithDetails(t *testing.T) {
	// Create an error with initial details
	err := New(ErrInvalidToken, "Invalid token")
	originalTime := err.Details.Timestamp

	// Wait a moment to ensure timestamps would differ
	time.Sleep(10 * time.Millisecond)

	// Replace with new details
	newDetails := &ErrorDetails{
		RequestID:      "new-req-id",
		ClientID:       "new-client-id",
		UserID:         "new-user-id",
		AdditionalInfo: map[string]string{"key": "value"},
	}

	err = err.WithDetails(newDetails)

	// Check fields
	if err.Details.RequestID != "new-req-id" {
		t.Errorf("Expected RequestID 'new-req-id', got '%s'", err.Details.RequestID)
	}
	if err.Details.ClientID != "new-client-id" {
		t.Errorf("Expected ClientID 'new-client-id', got '%s'", err.Details.ClientID)
	}
	if err.Details.UserID != "new-user-id" {
		t.Errorf("Expected UserID 'new-user-id', got '%s'", err.Details.UserID)
	}

	// Check timestamp preservation
	if err.Details.Timestamp != originalTime {
		t.Error("Expected original timestamp to be preserved")
	}

	// Test with nil details
	err = err.WithDetails(nil)
	if err.Details == nil {
		t.Error("WithDetails(nil) should not set Details to nil")
	}
}
