/*
Package errors provides a standardized error handling system for GAuth.

This package defines structured error types with error codes, detailed information, and
helper methods for creating and manipulating errors. The core types include:

  - ErrorCode: Enumerated error codes for different categories of errors
  - Error: The main error type with context information
  - ErrorDetails: Structured details about an error occurrence

Usage example:

	// Create a new error
	err := errors.New(errors.ErrInvalidToken, "Token has invalid format")

	// Add additional context
	err = err.WithAdditionalInfo("token_id", tokenID)

	// Add request information
	err = err.WithRequestInfo(requestID, clientID, "/api/token", "POST", "192.168.1.1")

	// Check error type
	if errors.IsAuthError(err) {
	    // Handle authentication error
	}

This package helps ensure consistent error handling and reporting throughout the GAuth
authentication framework.
*/
package errors
