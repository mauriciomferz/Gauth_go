# Basic Authentication Example

This example demonstrates basic authentication using GAuth with type-safe implementations:

## Features

1. **Type-Safe Authentication**
   - Strongly typed credentials
   - Type-safe metadata handling
   - Clear error types

2. **Event Handling**
   - Event publishing
   - Audit logging
   - Type-safe event data

3. **Token Management**
   - RSA token signing
   - Token validation
   - Token metadata

## Running the Example

```bash
go run main.go
```

## Expected Output

```
Authentication successful. Token: eyJhbGciOiJSUzI1...

Token validated. Subject: testuser, Scopes: [read write]
```

## Code Structure

### 1. Service Setup

```go
// Create event publisher
publisher := events.NewPublisher()
publisher.Subscribe(&events.LogHandler{})

// Create token service
tokenService := token.NewService(token.Config{
    SigningMethod:    token.RS256,
    SigningKey:       privateKey,
    ValidityPeriod:   time.Hour,
    DefaultScopes:    []string{"read"},
})

// Create auth service
authService := auth.NewService(auth.Config{
    TokenService: tokenService,
    Provider:    auth.NewBasicProvider(validateCredentials),
    EventHandler: publisher,
})
```

### 2. Type-Safe Authentication

```go
creds := auth.Credentials{
    Username: "testuser",
    Password: "testpass",
    Metadata: &auth.AuthMetadata{
        Device: &token.DeviceInfo{
            ID:        "device123",
            UserAgent: "ExampleApp/1.0",
            Platform:  "iOS",
        },
        ClientID: "example-app",
        Scopes:   []string{"read", "write"},
    },
}
```

### 3. Error Handling

```go
token, err := authService.Authenticate(ctx, creds)
if err != nil {
    switch err {
    case auth.ErrInvalidCredentials:
        // Handle invalid credentials
    case auth.ErrRateLimitExceeded:
        // Handle rate limiting
    default:
        // Handle other errors
    }
}
```

## Extension Points

1. **Custom Authentication**
   - Implement custom providers
   - Add MFA support
   - Custom validation logic

2. **Event Handling**
   - Add custom event handlers
   - Implement metrics collection
   - Add security monitoring

3. **Token Customization**
   - Add custom metadata
   - Implement token rotation
   - Add revocation support