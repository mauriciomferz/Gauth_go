/*
Package errors provides standardized error handling for authentication and authorization operations.

This package implements a comprehensive error system that provides:

  - Strongly typed error definitions
  - HTTP status code mapping
  - Error categorization
  - Localized error messages
  - Security context preservation
  - Error wrapping and unwrapping
  - Detailed error logging
  - Error code standardization
  - Client-friendly error responses

The error types preserve security context while providing appropriate information
to clients without leaking sensitive details.
*/
package errors
