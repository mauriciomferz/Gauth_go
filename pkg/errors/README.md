# GAuth Error Handling System

This package provides a comprehensive error handling system for the GAuth framework, offering structured error information for better debugging, logging, metrics, and client responses.

## Features

- **Structured Errors**: All errors include error codes, descriptive messages, and detailed context
- **Error Sources**: Track where errors originated from (authentication, authorization, token, etc.)
- **Chain of Causes**: Support for wrapping underlying errors while maintaining context
- **HTTP Integration**: Built-in support for HTTP status codes and request details
- **Request Tracking**: Include request IDs, client IDs, and user information with errors
- **Extensible**: Add custom information to errors as needed

## Usage Examples

### Basic Error Creation

```go
import "github.com/Gimel-Foundation/gauth/pkg/errors"

func validateToken(token string) error {
    if !isValid(token) {
        return autherrors.New(autherrors.ErrInvalidToken, "The token is invalid or malformed")
    }
    return nil
}
```

### Adding Context to Errors

```go
func processAuthRequest(req Request) error {
    err := validateToken(req.Token)
    if err != nil {
        // Convert standard errors to structured errors if needed
        var authErr *errors.Error
        if !stdErrors.As(err, &authErr) {
            authErr = errors.New(errors.ErrServerError, "Authentication failed")
                .WithCause(err)
        }
        
        // Add request context
        return authErr.WithRequestInfo(
            req.RequestID,
            req.ClientID, 
            req.UserID,
        ).WithHTTPInfo(
            req.Path,
            req.Method,
            http.StatusUnauthorized,
            req.IPAddress,
        )
    }
    return nil
}
```

### Error Handling

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    err := processAuthRequest(parseRequest(r))
    if err != nil {
        // Check for structured errors
        var authErr *errors.Error
        if stdErrors.As(err, &authErr) {
            // Handle structured error
            code := http.StatusInternalServerError
            if authErr.Details != nil && authErr.Details.HTTPStatusCode > 0 {
                code = authErr.Details.HTTPStatusCode
            }
            
            // Create error response
            resp := map[string]interface{}{
                "error":             string(authErr.Code),
                "error_description": authErr.Message,
            }
            
            // Add any additional information
            if authErr.Details != nil {
                for k, v := range authErr.Details.AdditionalInfo {
                    resp[k] = v
                }
            }
            
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(code)
            json.NewEncoder(w).Encode(resp)
            return
        }
        
        // Handle standard errors
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "server_error",
            "error_description": "An unexpected error occurred",
        })
        return
    }
    
    // Process successful request...
}
```

## Predefined Error Codes

The package includes common OAuth 2.0 and authorization-related error codes:

- `ErrTokenExpired`: The token has expired
- `ErrInvalidToken`: The token is invalid or malformed
- `ErrInsufficientScope`: The token lacks required scopes
- `ErrRateLimited`: Rate limit has been exceeded
- `ErrInvalidRequest`: The request is malformed
- `ErrInvalidClient`: Client authentication failed
- `ErrInvalidGrant`: The authorization grant is invalid
- `ErrUnauthorizedClient`: The client is not authorized
- `ErrInvalidScope`: The requested scope is invalid
- `ErrServerError`: An internal server error occurred
- `ErrTemporarilyUnavailable`: Service is temporarily unavailable

## Error Sources

Track where errors originated from:

- `SourceAuthentication`: Authentication system
- `SourceAuthorization`: Authorization system
- `SourceToken`: Token validation/creation
- `SourceStorage`: Storage systems
- `SourceRateLimiting`: Rate limiting system
- `SourceCircuitBreaker`: Circuit breaker
- `SourceValidation`: Input validation
- `SourceProtocol`: Protocol handling
- `SourceResourceServer`: Resource server

## Best Practices

1. **Create Specific Errors**: Use the most specific error code that applies
2. **Add Context**: Include request IDs, client IDs, and other context
3. **Wrap Causes**: Use WithCause to preserve underlying error information
4. **Consistent HTTP Status Codes**: Map error codes to appropriate HTTP status codes
5. **Log Structured Errors**: Use the structured information for comprehensive logging