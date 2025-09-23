# Code Organization Guide for GAuth Contributors

This document explains the organization of the GAuth codebase to help new contributors understand where to make changes and how to maintain code quality.

## Core Principles

1. **Clear Separation of Concerns**: Each package has a specific responsibility
2. **Type Safety**: Replacing `map[string]interface{}` with typed structures
3. **Modular Design**: Breaking large files into smaller, focused components
4. **Consistent API**: Providing a consistent interface for consumers
5. **Backward Compatibility**: Supporting both old and new APIs during transition

## Directory Structure

GAuth follows a standard Go project layout with the following structure:

```
gauth/
├── cmd/          # Command-line tools and executables
├── docs/         # Documentation files
├── examples/     # Example applications using GAuth
├── internal/     # Internal implementation details
│   ├── audit/      # Audit logging
│   ├── circuit/    # Circuit breaker
│   ├── errors/     # Error handling
│   ├── events/     # Event management
│   ├── rate/       # Rate limiting internals
│   ├── ratelimit/  # Rate limiting implementation
│   ├── resilience/ # Resilience patterns
│   └── security/   # Security utilities
├── pkg/          # Public API packages
│   ├── auth/       # Authentication
│   ├── authz/      # Authorization
│   ├── errors/     # Public error types
│   ├── events/     # Event system
│   ├── gauth/      # Main GAuth API
│   ├── rate/       # Rate limiting API
│   ├── resilience/ # Resilience patterns API
│   └── token/      # Token management
├── test/         # Integration and benchmark tests
├── go.mod        # Go modules definition
├── go.sum        # Go modules checksum
├── Makefile      # Build automation
└── README.md     # Project overview
```

## Package Organization

### Public API Packages (`pkg/`)

These packages form the public API of GAuth. Changes to these packages should maintain backward compatibility:

- **pkg/auth**: Core authentication functionality
  - User authentication
  - Credential validation
  - Multi-factor authentication

- **pkg/authz**: Authorization and policy enforcement
  - Policy definition and evaluation
  - Role-based access control
  - Attribute-based access control

- **pkg/token**: Token creation and validation
  - JWT support
  - PASETO support
  - Token signing and verification

- **pkg/store**: Token storage backends
  - Memory store (for development)
  - Redis store (for production)
  - Interface for custom stores

- **pkg/events**: Event system
  - Typed events with structured metadata
  - Event emission and subscription
  - Event filtering and routing

- **pkg/errors**: Error handling
  - Strongly-typed errors
  - Error codes and categories
  - Error context and details

- **pkg/gauth**: Main package integrating all components
  - High-level API
  - Configuration
  - Builder patterns

### Internal Implementation (`internal/`)

These packages contain implementation details that aren't part of the public API:

- **internal/resource**: Resource management
  - Resource definition
  - Resource configuration
  - Resource tagging

- **internal/ratelimit**: Rate limiting
  - Token bucket implementation
  - Distributed rate limiting
  - Rate limit configuration

- **internal/circuit**: Circuit breaker patterns
  - Circuit state management
  - Failure detection
  - Fallback mechanisms

- **internal/errors**: Internal error handling
  - Error creation
  - Error wrapping
  - Error conversion

## Type Safety and Modularization

GAuth emphasizes type safety throughout the codebase. We avoid using `map[string]interface{}` in favor of strongly-typed structures, and we break large files into smaller, focused components.

### Modularization Strategy

#### From Monolithic to Modular

We've broken down large files into smaller, focused components:

1. **Before**: `legal_framework.go` (1140+ lines)
2. **After**: 
   - `legal_framework_types.go`: Core type definitions
   - `legal_framework_impl.go`: Implementation logic

#### Rate Limiting Example

The rate limiting functionality has been restructured:

1. **Before**: `ratelimit.go` (553 lines) in root directory
2. **After**:
   - `internal/ratelimit/ratelimiter.go`: Core implementation
   - `internal/ratelimit/client_ratelimiter.go`: Client-specific implementation
   - `internal/ratelimit/adaptive_ratelimiter.go`: Adaptive implementation
   - `internal/ratelimit/http_middleware.go`: HTTP integration
   - `pkg/rate/ratelimit.go`: Public API

### Type-Safe Patterns

1. **Helper Methods Pattern**

Instead of directly accessing `map[string]interface{}`, we've added helper methods:

```go
// Before
val, ok := params["limit"].(int)
if !ok {
    // Type assertion failed
}

// After
val, err := condition.GetIntParam("limit")
if err != nil {
    // Handle error
}
```

2. **Event Metadata**

Instead of:
```go
type Event struct {
    Metadata map[string]interface{}
}
```

We use:
```go
type Metadata struct {
    Values map[string]MetadataValue
}

type MetadataValue struct {
    Type string
    Data interface{}
}

// With helper methods
func (m *Metadata) GetString(key string) (string, bool)
func (m *Metadata) SetString(key, value string)
// etc.
```

3. **Resource Configuration**

Instead of:
```go
type Resource struct {
    Config map[string]interface{}
}
```

We use:
```go
type Resource struct {
    Config *ResourceConfig
}

type ResourceConfig struct {
    Settings map[string]ConfigValue
}

type ConfigValue struct {
    Type string
    Data interface{}
}

// With helper methods
func (r *Resource) GetConfigString(key string) (string, bool)
func (r *Resource) SetConfigString(key, value string)
// etc.
```

4. **Error Details**

Instead of:
```go
type Error struct {
    Details map[string]interface{}
}
```

We use:
```go
type Error struct {
    Details *ErrorDetails
}

type ErrorDetails struct {
    Timestamp time.Time
    RequestID string
    UserID string
    // Other specific fields
    AdditionalInfo map[string]string // Typed map
}
```

### Backward Compatibility

For a smooth transition, we provide conversion methods:

```go
// Convert from typed structure to map
func (m *UserMetadata) ToMap() map[string]interface{} {
    return map[string]interface{}{
        "user": m.Username,
        "roles": m.Roles,
        "active": m.Active,
    }
}

// Convert from map to typed structure
func UserMetadataFromMap(m map[string]interface{}) (*UserMetadata, error) {
    // Implementation with validation
}
```

## Testing Standards

All packages should have corresponding test files following these guidelines:

1. Unit tests for all exported functions and types
2. Table-driven tests for functions with multiple code paths
3. Mocks or stubs for external dependencies
4. Integration tests for interactions between components
5. Benchmark tests for performance-critical code

## Documentation Standards

Documentation in GAuth follows these guidelines:

1. All exported types, functions, and constants have descriptive comments
2. Package-level documentation in `doc.go` files
3. Examples for common use cases
4. Consistent formatting and style

## Best Practices for Contributors

1. **Create Focused Files**: Break down large files into smaller, focused components
2. **Maintain Type Safety**: Use strongly-typed structures instead of generic maps
3. **Add Helper Methods**: Provide type-safe access to data
4. **Ensure Backward Compatibility**: Add conversion methods between typed structures and maps
5. **Separate Concerns**: Keep packages focused on specific responsibilities
6. **Follow Go Idioms**: Adhere to standard Go practices and conventions
7. **Write Tests**: Add tests for all new functionality
8. **Document Changes**: Update documentation to reflect your changes
9. **Consider Performance**: Be mindful of performance implications, especially in core components
10. **Provide Examples**: Create example code showing how to use the API

## Migration Strategy

For existing components:

1. **Identify Large Files**: Look for files over 500 lines that need modularization
2. **Extract Types**: Move type definitions to separate files
3. **Add Helper Methods**: Implement type-safe accessors for data
4. **Implement Backward Compatibility**: Create conversion functions between old and new APIs
5. **Update Tests**: Ensure tests cover both old and new APIs
6. **Update Documentation**: Document the new structure and provide migration guidance
7. **Create Examples**: Demonstrate the new APIs through example code