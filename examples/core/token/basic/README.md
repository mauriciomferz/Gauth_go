# Basic Token Management Example

This example demonstrates the core token management features:

1. Token creation with typed metadata
2. Secure token signing (RSA)
3. Token validation
4. Token storage and retrieval
5. Token filtering
6. Token revocation

## Features Demonstrated

- Strongly typed token metadata
- RSA signature verification
- Token filtering by multiple criteria
- Safe token revocation
- Clean error handling

## Running the Example

```bash
go run main.go
```

## Expected Output

```
Issued token: eyJhbGciOiJSUzI1N...

Token validated successfully

Found 1 matching tokens

Token validation failed as expected: token has been revoked
```

## Code Organization

The example demonstrates:

1. Setting up the token service:
   - RSA key generation
   - Service configuration
   - Store configuration

2. Creating a token with metadata:
   - Basic claims (sub, iss, aud)
   - Scopes for authorization
   - Device information
   - Application context
   - Classification (tags/labels)

3. Token operations:
   - Issuing
   - Validation
   - Filtering
   - Revocation

## Key Concepts

### Type Safety

The example uses strongly typed structures instead of generic maps:

```go
token.Metadata{
    Device: &token.DeviceInfo{
        ID:        "device123",
        UserAgent: "ExampleApp/1.0",
        Platform:  "iOS",
        Version:   "15.0",
    },
    Labels: map[string]string{
        "environment": "production",
    },
    Tags: []string{"mobile"},
}
```

### Token Filtering

Demonstrates flexible token querying:

```go
filter := &token.Filter{
    Types:    []token.Type{token.Access},
    Subject:  "user123",
    Active:   true,
    Tags:     []string{"mobile"},
    Labels:   map[string]string{
        "environment": "production",
    },
}
```

### Proper Revocation

Shows correct token revocation with metadata:

```go
token.RevocationStatus{
    RevokedAt: time.Now(),
    Reason:    "user logout",
    RevokedBy: "example-app",
}