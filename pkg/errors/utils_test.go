package errors

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"
	"testing"
)

func TestWithStack(t *testing.T) {
	err := New(ErrServerError, "Test error with stack")
	err = err.WithStack()

	// Check that stack trace is added
	if err.Details == nil {
		t.Error("Details should not be nil after WithStack")
		return
	}

	stackTrace, ok := err.Details.AdditionalInfo["stack_trace"]
	if !ok {
		t.Error("Stack trace should be added to AdditionalInfo")
		return
	}

	// Check that stack trace contains this function's name
	if !strings.Contains(stackTrace, "TestWithStack") {
		t.Errorf("Stack trace should contain test function name, got: %s", stackTrace)
	}

	// Check that stack trace contains file information
	if !strings.Contains(stackTrace, ".go:") {
		t.Errorf("Stack trace should contain file information, got: %s", stackTrace)
	}
}

func TestWithFields(t *testing.T) {
	err := New(ErrServerError, "Test error with fields")

	fields := map[string]string{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
	}

	err = err.WithFields(fields)

	// Check that fields are added
	if err.Details == nil {
		t.Error("Details should not be nil after WithFields")
		return
	}

	for k, v := range fields {
		got, ok := err.Details.AdditionalInfo[k]
		if !ok {
			t.Errorf("Field %s should be present in AdditionalInfo", k)
			continue
		}

		if got != v {
			t.Errorf("Field %s value should be %s, got %s", k, v, got)
		}
	}
}

func TestWithContext(t *testing.T) {
	err := New(ErrServerError, "Test error with context")

	// Create context with values
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "test-req-123")
	ctx = context.WithValue(ctx, "user_id", "test-user-456")

	err = err.WithContext(ctx)

	// Check that context values are extracted
	if err.Details == nil {
		t.Error("Details should not be nil after WithContext")
		return
	}

	if err.Details.RequestID != "test-req-123" {
		t.Errorf("RequestID should be extracted from context, got %s", err.Details.RequestID)
	}

	if err.Details.UserID != "test-user-456" {
		t.Errorf("UserID should be extracted from context, got %s", err.Details.UserID)
	}
}

func TestIsAuthError(t *testing.T) {
	// Test with auth error
	err1 := New(ErrInvalidToken, "Invalid token")
	if !IsAuthError(err1) {
		t.Errorf("ErrInvalidToken should be detected as auth error")
	}

	err2 := New(ErrTokenExpired, "Token expired")
	if !IsAuthError(err2) {
		t.Errorf("ErrTokenExpired should be detected as auth error")
	}

	// Test with non-auth error
	err3 := New(ErrRateLimited, "Rate limited")
	if IsAuthError(err3) {
		t.Errorf("ErrRateLimited should not be detected as auth error")
	}

	// Test with standard error
	err4 := stderrors.New("standard error")
	if IsAuthError(err4) {
		t.Errorf("Standard error should not be detected as auth error")
	}
}

func TestIsRateLimitError(t *testing.T) {
	// Test with rate limit error
	err1 := New(ErrRateLimited, "Rate limited")
	if !IsRateLimitError(err1) {
		t.Errorf("ErrRateLimited should be detected as rate limit error")
	}

	// Test with non-rate limit error
	err2 := New(ErrInvalidToken, "Invalid token")
	if IsRateLimitError(err2) {
		t.Errorf("ErrInvalidToken should not be detected as rate limit error")
	}

	// Test with standard error
	err3 := stderrors.New("standard error")
	if IsRateLimitError(err3) {
		t.Errorf("Standard error should not be detected as rate limit error")
	}
}

func TestGetRetryAfter(t *testing.T) {
	// Test with retry-after present
	err1 := New(ErrRateLimited, "Rate limited")
	err1 = err1.AddInfo("retry_after", "60")

	retryAfter, ok := GetRetryAfter(err1)
	if !ok {
		t.Error("GetRetryAfter should return true when retry_after is present")
	}
	if retryAfter != 60 {
		t.Errorf("GetRetryAfter should return 60, got %d", retryAfter)
	}

	// Test with invalid retry-after
	err2 := New(ErrRateLimited, "Rate limited")
	err2 = err2.AddInfo("retry_after", "invalid")

	_, ok = GetRetryAfter(err2)
	if ok {
		t.Error("GetRetryAfter should return false when retry_after is invalid")
	}

	// Test with missing retry-after
	err3 := New(ErrRateLimited, "Rate limited")
	_, ok = GetRetryAfter(err3)
	if ok {
		t.Error("GetRetryAfter should return false when retry_after is missing")
	}

	// Test with standard error
	err4 := stderrors.New("standard error")
	_, ok = GetRetryAfter(err4)
	if ok {
		t.Error("GetRetryAfter should return false for standard error")
	}
}

func TestNewHTTPError(t *testing.T) {
	// Create a test HTTP response
	req, _ := http.NewRequest("GET", "http://example.com/api/resource", nil)
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Request:    req,
		Header:     http.Header{},
	}
	resp.Header.Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
	resp.Header.Set("Retry-After", "60")

	// Test JSON response body
	body := []byte(`{"error":"invalid_token","error_description":"The access token is invalid"}`)

	err := NewHTTPError(resp, body)

	// Check error code mapping
	if err.Code != ErrInvalidToken {
		t.Errorf("Expected error code %s, got %s", ErrInvalidToken, err.Code)
	}

	// Check error message from JSON
	if err.Message != "The access token is invalid" {
		t.Errorf("Expected message from JSON, got %s", err.Message)
	}

	// Check HTTP details
	if err.Details.HTTPStatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, err.Details.HTTPStatusCode)
	}

	// Check headers were added as additional info
	wwwAuth, ok := err.Details.AdditionalInfo["www-authenticate"]
	if !ok || wwwAuth != "Bearer error=\"invalid_token\"" {
		t.Errorf("Expected WWW-Authenticate header in additional info")
	}
}
