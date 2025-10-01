# Type-Safe Token Usage Example

This example demonstrates the use of strongly-typed token management in GAuth. It shows how to:

1. Create and configure a type-safe token store
2. Generate tokens with proper typing
3. Add structured metadata
4. Perform type-safe token operations
5. Use filters for token querying
6. Handle token rotation securely

## Key Features

### 1. Strong Type Safety

Instead of using `map[string]interface{}`, the example uses proper types:

```go
type Token struct {
    ID        string
    Type      Type
    Value     string
    IssuedAt  time.Time
    ExpiresAt time.Time
    // ... other fields
}
```

### 2. Structured Metadata

Metadata is strongly typed and extensible:

```go
type Metadata struct {
    Device     *DeviceInfo
    AppID      string
    AppVersion string
    Labels     map[string]string
    Tags       []string
    Attributes map[string][]string
}
```

### 3. Type-Safe Operations

All operations use proper types:

```go
// Saving a token
store.Save(ctx, key, token)

// Filtering tokens
filter := token.Filter{
    Types:   []token.Type{token.Access},
    Subject: "user-123",
    Active:  true,
}
```

### 4. Error Handling

Proper error types for different scenarios:

```go
type ValidationError struct {
    Code    string
    Message string
}
```

## Running the Example

```bash
go run main.go
```

## Key Concepts Demonstrated

1. **Token Configuration**
   - Proper algorithm types
   - Validation settings
   - Scope configuration

2. **Token Creation**
   - Type-safe fields
   - Proper time handling
   - Structured metadata

3. **Token Operations**
   - CRUD operations
   - Token rotation
   - Token validation

4. **Metadata Management**
   - Device information
   - Application context
   - Labels and tags

5. **Error Handling**
   - Type-specific errors
   - Validation failures
   - Operation errors

## Best Practices

1. **Type Safety**
   - Use proper types instead of interface{}
   - Define clear structures
   - Validate at compile time

2. **Token Management**
   - Proper expiration handling
   - Secure token rotation
   - Clear validation rules

3. **Metadata Organization**
   - Structured device info
   - Clear attribute patterns
   - Extensible design

4. **Error Handling**
   - Specific error types
   - Clear error messages
   - Proper error wrapping

## Further Reading

- [Token Package Documentation](../../../pkg/token/doc.go)
- [Token Types Documentation](../../../pkg/token/types.go)
- [Token Store Interface](../../../pkg/token/store.go)