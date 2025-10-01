# Distributed Token Management Example

This example demonstrates the token management capabilities of the GAuth library, specifically:

1. Distributed token storage with Redis
2. Token validation with configurable rules
3. Token lifecycle management
4. Access control and scoping

## Prerequisites

- Redis server running locally on default port (6379)
- Go 1.16 or later

## Features Demonstrated

1. **Token Creation and Storage**
   - Creating tokens with metadata
   - Storing tokens in Redis
   - Token value to ID mapping

2. **Token Validation**
   - Issuer validation
   - Audience validation
   - Scope requirements
   - Time-based validation (expiry, not-before)
   - Clock skew handling

3. **Token Management**
   - Listing tokens by filters
   - Token revocation
   - Revocation verification

4. **Error Handling**
   - Validation failures
   - Storage errors
   - Not found scenarios

## Running the Example

1. Start Redis:
   ```sh
   docker run -d -p 6379:6379 redis
   ```

2. Run the example:
   ```sh
   go run main.go
   ```

## Expected Output

The example will show:
1. Token creation and storage
2. Token retrieval and validation
3. Token listing with filters
4. Token revocation
5. Validation failure cases

## Code Structure

- `main.go`: Example implementation
- Uses `github.com/Gimel-Foundation/gauth/pkg/token` package
- Demonstrates all major token management features

## Key Concepts

### Token Storage

```go
store, err := token.NewRedisStore(token.RedisConfig{
    Addresses:  []string{"localhost:6379"},
    KeyPrefix:  "example:",
    DefaultTTL: time.Hour * 24,
})
```

### Token Validation

```go
validator := token.NewValidationChain(token.ValidationConfig{
    AllowedIssuers:   []string{"example-service"},
    AllowedAudiences: []string{"example-app"},
    RequiredScopes:   []string{"read"},
    ClockSkew:       time.Minute,
}, nil)
```

### Token Creation

```go
newToken := &token.Token{
    ID:      "example-token",
    Subject: "user123",
    Issuer:  "example-service",
    Scopes:  []string{"read", "write"},
    // ... additional fields
}
```

### Token Management

```go
// List tokens
tokens, err := store.List(ctx, token.Filter{
    Subject: "user123",
    Type:    token.AccessToken,
})

// Revoke token
err := store.Revoke(ctx, tokenID, "user logout")
```

## Error Handling

The example demonstrates handling of various error cases:
- `ErrTokenNotFound`
- `ErrTokenExpired`
- `ErrInvalidIssuer`
- `ErrInsufficientScope`
- `ErrTokenRevoked`

## Best Practices

1. Always close the store when done:
   ```go
   defer store.Close()
   ```

2. Use proper validation configuration:
   ```go
   ValidationConfig{
       ClockSkew: time.Minute, // Allow for time differences
       // ... other settings
   }
   ```

3. Include proper token metadata:
   ```go
   token.Metadata{
       DeviceInfo: &token.DeviceInfo{
           ID:        "device123",
           UserAgent: "Example/1.0",
           IPAddress: "192.168.1.1",
       },
   }
   ```

4. Handle all errors appropriately:
   ```go
   if err != nil {
       log.Fatalf("Operation failed: %v", err)
   }
   ```

## Next Steps

1. Implement custom validation rules
2. Add monitoring and metrics
3. Implement token encryption
4. Add cache layer for performance
5. Implement token rotation