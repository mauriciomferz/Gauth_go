# Type Safety in GAuth

This document explains GAuth's approach to type safety and how we've moved away from using `map[string]interface{}` in favor of strongly-typed structures.

## The Problem with `map[string]interface{}`

Using `map[string]interface{}` in a library like GAuth has several drawbacks:

1. **Lack of Type Safety**: No compile-time type checking, leading to potential runtime errors
2. **Poor Developer Experience**: Unclear what keys/values are expected
3. **No Autocompletion**: IDE support is limited without type information
4. **Error-Prone**: Easy to misspell keys or use incorrect value types
5. **Performance Impact**: Type assertions and reflections can impact performance
6. **Documentation Burden**: Requires extensive documentation to explain expected structure

## Our Solution: Typed Structures

GAuth has moved to a type-safe approach using the following patterns:

### 1. Event Metadata

The `Metadata` type provides a strongly-typed container for event data:

```go
// Instead of map[string]interface{}
type Metadata struct {
    Values map[string]MetadataValue
}

type MetadataValue struct {
    Type string      // "string", "int", "bool", etc.
    Data interface{} // The actual value
}
```

With helper methods for type-safe access:

```go
func (m *Metadata) GetString(key string) (string, bool)
func (m *Metadata) GetInt(key string) (int, bool)
func (m *Metadata) GetBool(key string) (bool, bool)
func (m *Metadata) GetFloat(key string) (float64, bool)
func (m *Metadata) GetTime(key string) (time.Time, bool)

func (m *Metadata) SetString(key, value string)
func (m *Metadata) SetInt(key string, value int)
// etc.
```

Example usage:

```go
// Creating metadata
metadata := events.NewMetadata()
metadata.SetString("user_id", "user123")
metadata.SetInt("login_attempts", 3)
metadata.SetBool("successful", true)

// Using metadata
if userID, ok := metadata.GetString("user_id"); ok {
    // Use userID safely
}

if attempts, ok := metadata.GetInt("login_attempts"); ok && attempts > 5 {
    // Handle too many attempts
}
```

### 2. Authorization Context

The `Context` type in the authorization package provides type safety for contextual information:

```go
type Context struct {
    Values map[string]ContextValue
}

type ContextValue struct {
    Type string
    Data interface{}
}
```

With helper methods:

```go
func (c *Context) GetString(key string) (string, bool)
func (c *Context) SetString(key, value string)
// etc.
```

Example usage:

```go
// Creating an access request with context
request := authz.NewAccessRequest(
    authz.Subject{ID: "user123", Type: "user"},
    authz.Resource{ID: "doc456", Type: "document"},
    authz.Action{Name: "read"},
)

request.WithStringValue("department", "engineering")
request.WithBoolValue("emergency", true)
```

### 3. Resource Configuration

The `ResourceConfig` type provides type safety for resource configuration:

```go
type ResourceConfig struct {
    Settings map[string]ConfigValue
}

type ConfigValue struct {
    Type string
    Data interface{}
}
```

With helper methods:

```go
func (r *Resource) GetConfigString(key string) (string, bool)
func (r *Resource) SetConfigString(key, value string)
// etc.
```

Example usage:

```go
resource := resource.NewResource("api-gateway", resource.TypeAPI)
resource.SetConfigString("version", "v2")
resource.SetConfigInt("max_connections", 1000)
resource.SetConfigBool("public", true)

// Type-safe access
if version, ok := resource.GetConfigString("version"); ok {
    fmt.Println("API version:", version)
}
```

### 4. Error Details

The `ErrorDetails` structure provides type safety for error context:

```go
type ErrorDetails struct {
    Timestamp     time.Time
    RequestID     string
    ClientID      string
    UserID        string
    ResourceID    string
    IPAddress     string
    Path          string
    Method        string
    AdditionalInfo map[string]interface{}
}
```

With helper methods:

```go
func (e *Error) WithRequestInfo(requestID, clientID, path, method, ipAddress string) *Error
func (e *Error) WithUserID(userID string) *Error
func (e *Error) WithResourceID(resourceID string) *Error
func (e *Error) WithAdditionalInfo(key string, value interface{}) *Error
```

Example usage:

```go
err := errors.New(errors.ErrInvalidToken, "Token signature verification failed")
err = err.WithRequestInfo(requestID, clientID, "/api/resource", "GET", ipAddress)
err = err.WithUserID(userID)
err = err.WithAdditionalInfo("token_id", tokenID)
```

## Benefits of Type-Safe Structures

1. **Compile-Time Safety**: Catch errors at compile time rather than runtime
2. **Better Developer Experience**: Clear API with IDE autocompletion
3. **Self-Documenting**: Structure makes clear what data is expected
4. **Improved Performance**: Avoid excessive type assertions
5. **Maintainability**: Easier to evolve the codebase over time

## Backward Compatibility

For backward compatibility, we provide conversion methods:

```go
// Convert type-safe metadata to map
func (m *Metadata) ToMap() map[string]interface{}

// Create metadata from map
func MetadataFromMap(data map[string]interface{}) *Metadata
```

## Guidelines for Contributors

When contributing to GAuth:

1. **Avoid `map[string]interface{}`** in public APIs
2. **Create typed structures** for data that needs to be flexible
3. **Provide helper methods** for type-safe access
4. **Add conversion methods** for backward compatibility if needed
5. **Document structure expectations** clearly