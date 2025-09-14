# Package Organization Guide

This document outlines the organization and responsibilities of each package in the GAuth library.

## Core Packages (`pkg/`)

### `auth/`
Authentication functionality and token management.
```go
auth/
├── claims/       // Claims handling and validation
├── providers/    // Authentication providers (Basic, OAuth2, etc.)
├── tokens/       // Token generation and validation
└── doc.go       // Package documentation
```

### `authz/`
Authorization and permission management.
```go
authz/
├── policy/      // Policy definition and evaluation
├── roles/       // Role-based access control
└── scope/       // Scope-based authorization
```

### `token/`
Token operations and storage.
```go
token/
├── store/       // Token storage implementations
├── encryption/  // Token encryption/decryption
└── rotation/    // Key rotation management
```

### `audit/`
Audit logging and security events.
```go
audit/
├── events/      // Event definitions and handlers
├── storage/     // Audit log storage
└── export/      // Audit log export formats
```

## Internal Packages (`internal/`)

### `circuit/`
Circuit breaker implementation.
```go
circuit/
├── breaker.go   // Circuit breaker core
└── monitor.go   // State monitoring
```

### `rate/`
Rate limiting implementation (per-user via OwnerID and per-client, type-safe APIs).
```go
rate/
├── limiter.go    // Rate limiter core
└── algorithms/   // Rate limiting algorithms
```

### `resilience/`
Resilience patterns.
```go
resilience/
├── retry/       // Retry strategies
├── timeout/     // Timeout handlers
└── bulkhead/    // Concurrency control
```

### `events/`
Event system implementation.
```go
events/
├── bus.go       // Event bus
├── handlers/    // Event handlers
└── types.go     // Event type definitions
```

## Support Packages

### `resources/`
Shared resources and constants.
```go
resources/
├── messages.go  // User-facing messages
└── errors.go    // Error definitions
```

### `metrics/`
Monitoring and metrics.
```go
metrics/
├── prometheus/  // Prometheus metrics
└── statsd/      // StatsD metrics
```

## Package Dependencies

1. Core Package Dependencies:
```
auth → token → encryption
  ↓      ↓
authz → audit
```

2. Support Package Dependencies:
```
All Packages → resources
All Packages → metrics (optional)
```

3. Internal Package Dependencies:
```
circuit → events
rate → metrics
resilience → circuit, rate
```

## Best Practices

1. **Package Organization**
   - Keep packages focused and cohesive
   - Use internal packages for implementation details
   - Export only necessary types and functions

2. **Dependencies**
   - Minimize cross-package dependencies
   - Use interfaces for decoupling
   - Avoid circular dependencies

3. **Documentation**
   - Include doc.go in each package
   - Document all exported symbols
   - Provide usage examples

4. **Testing**
   - Keep tests alongside code
   - Use table-driven tests
   - Include benchmarks for critical paths

## Guidelines for New Packages

When adding a new package:

1. **Package Location**
   - Use `pkg/` for public API
   - Use `internal/` for implementation details

2. **Package Structure**
   - Create doc.go first
   - Define public interfaces
   - Implement core functionality
   - Add tests

3. **Documentation**
   - Document package purpose
   - Include usage examples
   - List dependencies

4. **Testing**
   - Add unit tests
   - Add integration tests if needed
   - Include benchmarks for performance-critical code