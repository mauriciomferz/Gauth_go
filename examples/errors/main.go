package main

import (
	"fmt"
	"log"
	"net/http"

	"os"

	"github.com/Gimel-Foundation/gauth/pkg/errors"
)

func main() {
	// Example: Handle token validation error
	if err := validateToken("invalid-token"); err != nil {
		log.Printf("Error: %v", err)
		if authErr, ok := err.(*errors.Error); ok {
			handleStructuredError(authErr)
		}
		os.Exit(1)
	}

	// Example: Handle rate limit error
	if err := checkRateLimits("client123"); err != nil {
		log.Printf("Error: %v", err)
		if authErr, ok := err.(*errors.Error); ok {
			handleStructuredError(authErr)
		}
		os.Exit(1)
	}

	log.Println("All operations completed successfully")
}

func validateToken(token string) error {
	// Simulate token validation failure
	if token != "valid-token" {
		err := errors.New(errors.ErrInvalidToken, "The token provided is malformed or invalid")
		err = err.WithSource(errors.SourceToken)
		err = err.WithRequestInfo("req-123", "client-456", "user-789")
		err = err.WithHTTPInfo("/api/resource", "GET", http.StatusUnauthorized, "192.168.1.1")
		err = err.AddInfo("token_hint", "Check token format and signature")
		return err
	}
	return nil
}

func checkRateLimits(clientID string) error {
	// Simulate rate limit being exceeded
	if clientID == "client123" {
		baseErr := fmt.Errorf("rate limit of 100 requests per minute exceeded")
		err := errors.New(errors.ErrRateLimited, "API rate limit exceeded")
		err = err.WithSource(errors.SourceRateLimiting)
		err = err.WithCause(baseErr)
		err = err.WithRequestInfo("req-456", clientID, "")
		err = err.WithHTTPInfo("/api/resource", "POST", http.StatusTooManyRequests, "192.168.1.1")
		err = err.AddInfo("retry_after", "60")
		return err
	}
	return nil
}

func handleStructuredError(err *errors.Error) {
	// Example of how to use the structured error for different purposes:

	// 1. Logging with context
	log.Printf("[%s] %s (Source: %s, Code: %s)",
		err.Details.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		err.Message,
		err.Source,
		err.Code)

	// 2. Error response formatting
	if err.Details.HTTPStatusCode > 0 {
		errorResponse := map[string]interface{}{
			"error":             string(err.Code),
			"error_description": err.Message,
		}

		// Add retry information for rate limiting
		if err.Code == errors.ErrRateLimited {
			if retryAfter, ok := err.Details.AdditionalInfo["retry_after"]; ok {
				errorResponse["retry_after"] = retryAfter
			}
		}

		// In a real application, we would return this as JSON
		log.Printf("Would respond with HTTP %d: %+v",
			err.Details.HTTPStatusCode, errorResponse)
	}

	// 3. Metrics and monitoring
	log.Printf("Would increment metric: errors_total{code=%q,source=%q}",
		err.Code, err.Source)

	// 4. Special handling for specific errors
	switch err.Code {
	case errors.ErrTokenExpired:
		log.Println("Would trigger token refresh flow")
	case errors.ErrRateLimited:
		log.Println("Would implement exponential backoff")
	case errors.ErrServerError:
		log.Println("Would trigger alert to on-call engineer")
	}
}
