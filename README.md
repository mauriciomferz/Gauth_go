# GAuth: AI Power-of-Attorney Authorization Framework

GAuth enables AI systems to act on behalf of humans or organizations, with explicit, verifiable, and auditable power-of-attorney flows. Built on OAuth, OpenID Connect, and MCP, GAuth is designed for open source, extensibility, and compliance with RFC111.

---

## Who is this for?
- Developers integrating AI with sensitive actions or decisions
- Security architects and compliance teams
- Anyone needing transparent, auditable AI authorization


## Where to Start
- Library code: `pkg/gauth/`
- Examples: All runnable examples are now isolated in their own `main.go` under `examples/` or `examples/<category>/cmd/`.
- See `examples/README.md` for a directory of example topics.
- Package docs: `pkg/gauth/doc.go`

## RFC111 Compliance
- Implements the GiFo-RfC 0111 (GAuth) standard for AI power-of-attorney, delegation, and auditability.
- All protocol roles, flows, and exclusions are respected. See https://gimelfoundation.com for the full RFC.


## Features

- **Type Safety**: All public APIs use explicit, strongly-typed structures (no map[string]interface{} in public APIs)
- **Authentication**: Token-based authentication with JWT and PASETO support
- **Authorization**: Fine-grained, policy-based access control (RBAC, ABAC, PBAC)
- **Audit Logging**: Comprehensive, structured audit trails for all actions
- **Extensible Storage**: Memory, Redis, and database backends
- **Resilience**: Circuit breaking and retry mechanisms
- **Observability**: Metrics, distributed tracing, and event-driven architecture
- **Modernized Examples**: All examples are now up-to-date with the latest API and are runnable in isolation.


## Getting Started

1. Review the runnable examples in `examples/` or `examples/<category>/cmd/` for minimal and advanced integrations.
2. See `pkg/gauth/` for core types and APIs.
3. Extend or customize by implementing your own token store, audit logger, or event types.


## Manual Testing
See `MANUAL_TESTING.md` for suggestions.

---

## Breaking Changes (2025 Migration)

- All example logic with a `main` function is now in its own `main.go` under `examples/` or `examples/<category>/cmd/`.
- All obsolete and duplicate example files have been removed.
- All examples are refactored for the new API and type-safe signatures.
- See `docs/IMPROVEMENTS.md` for a summary of codebase modernization.

## License
- GAuth: Apache 2.0 (see LICENSE)
- OAuth, OpenID Connect: Apache 2.0
- MCP: MIT

---
For more, see the RFC111 specification and the package-level documentation in `doc.go`.
    AuthServerURL: "https://auth.example.com",
    ClientID:     "client-123",
    ClientSecret: "secret-456",
    RateLimit: gauth.RateLimitConfig{
        RequestsPerSecond: 100,
        WindowSize:       60,
    },

// Example: Per-user (OwnerID) rate limiting
server := gauth.NewResourceServer("resource", auth)
server.SetRateLimit(10, time.Minute) // 10 requests per minute per OwnerID

// In ProcessTransaction, rate limiting is enforced per token OwnerID
})

// Use in your application
token, err := auth.Authenticate(ctx, credentials)
if err != nil {
    // Handle error
}
```

## Documentation

- [Getting Started](docs/GETTING_STARTED.md): Quick introduction to GAuth
- [Code Organization](docs/CODE_ORGANIZATION.md): Understanding the codebase structure
- [Type Safety](docs/TYPE_SAFETY.md): How GAuth ensures type safety
- [Architecture](docs/ARCHITECTURE.md): Design and architecture of GAuth
- [Development](docs/DEVELOPMENT.md): Contributing to GAuth
- [Patterns Guide](docs/PATTERNS_GUIDE.md): Common patterns and best practices
- [Event System](docs/EVENT_SYSTEM.md): Understanding the event system
- [Testing](docs/TESTING.md): Testing GAuth and your application

## Package Structure

- `pkg/`: Public API packages
  - `auth/`: Authentication functionality
  - `authz/`: Authorization and policy enforcement
  - `token/`: Token creation and validation
  - `store/`: Token and session storage backends
  - `events/`: Event system with typed metadata
  - `errors/`: Error handling with typed details
  - `audit/`: Audit logging with compliance features
  - `rate/`: Rate limiting with multiple algorithms
  - `resilience/`: Circuit breaking and retry patterns
  - `resources/`: Resource management system
  - `monitoring/`: Metrics and distributed tracing
  - `metrics/`: Prometheus and custom metrics
  - `mesh/`: Service mesh integration
  - `util/`: Common utilities and helpers
  - `gauth/`: Main integration package

- `internal/`: Implementation details
  - `resource/`: Resource management with typed configurations
  - `ratelimit/`: Rate limiting implementation
  - `circuit/`: Circuit breaking patterns
  - `errors/`: Internal error types

- `examples/`: Usage examples
  - `basic/`: Simple authentication flow
  - `rate/`: Rate limiting examples
  - `resilient/`: Resilience patterns
  - `advanced/`: Advanced usage patterns
  - `typed_events/`: Using strongly typed event structures
  - `errors/`: Error handling and propagation
  - `audit/`: Audit logging implementation
  - `cache/`: Token caching strategies
  - `gateway/`: API gateway integration
  - `events/`: Event system with filtering and subscription

## Demos and Examples

Explore the `/demo` and `/cmd/demo` directories for runnable demonstration applications and usage examples:

- `/demo/main.go`: End-to-end protocol demo (authorization, token, transaction, audit)
- `/cmd/demo/main.go` and `/cmd/demo/improved_main.go`: CLI demo apps showing advanced flows

To run a demo:

```sh
# Run the main demo
cd demo
go run main.go

# Or run the CLI demo
cd cmd/demo
go run main.go
```

These demos use only the public, type-safe APIs and are a great starting point for learning and extension.

## Design Philosophy

1. **Clear Entry Points**: Each package has well-defined interfaces and documentation.
2. **Extensibility**: Core functionality is interface-based for easy customization.
3. **Type Safety**: Strong typing with typed structures instead of `interface{}`.
4. **Resilience**: Built-in patterns for reliable operation.
5. **Developer Experience**: Comprehensive examples and documentation.
6. **Modularity**: Common utilities are centralized and reusable across packages.

## Typed Structures

GAuth uses strongly typed structures instead of `map[string]interface{}` for better type safety, IDE support, and code clarity:

```go
// Instead of this:
metadata := map[string]interface{}{
    "user_id": "user123",
    "method": "password",
    "timestamp": time.Now(),
}

// Use this:
authEvent := &AuthenticationEvent{
    User: UserMetadata{
        UserID:   "user123",
        Username: "johndoe",
        Email:    "john@example.com",
    },
    Auth: AuthenticationMetadata{
        Method:    "password",
        Timestamp: time.Now(),
        SourceIP:  "192.168.1.1",
    },
}

// Type-safe access without type assertions
fmt.Println(authEvent.User.Username)  // No type assertions needed
```

See the [typed_events example](examples/typed_events) for a complete demonstration.
7. **Code Organization**: Library code and examples are clearly separated.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

1. Clone the repository
2. Install dependencies: `go mod download`
3. Run tests: `go test ./...`
4. Try examples: `cd examples/basic && go run main.go`

### Code Organization

Each package follows these principles:
- `doc.go`: Package documentation
- `interfaces.go`: Public interfaces
- Implementation files: Focused, single-responsibility
- `_test.go`: Comprehensive tests

## Type Safety

GAuth emphasizes type safety throughout the codebase. Instead of using `map[string]interface{}`, we provide strongly-typed structures with helper methods:

```go
// Instead of using map[string]interface{}:
metadata := make(map[string]interface{})
metadata["user_id"] = "user123"
metadata["logged_in"] = true
metadata["login_count"] = 42

// We use typed structures with helper methods:
metadata := events.NewMetadata()
metadata.SetString("user_id", "user123")
metadata.SetBool("logged_in", true)
metadata.SetInt("login_count", 42)

// Type-safe access:
if userID, ok := metadata.GetString("user_id"); ok {
    // Use userID safely
}
```

Read more about our approach to type safety in [Type Safety](docs/TYPE_SAFETY.md).

## Examples

See the `examples/` directory for:
- Basic authentication flows
- Rate limiting implementations
- Resilience patterns
- Custom extensions
- Typed structures demo (showing type-safe alternatives to map[string]interface{})

## Testing

Run the full test suite:
```bash
go test ./...
```

Run specific tests:
```bash
go test ./pkg/gauth -run TestAuth
go test ./internal/rate -run TestRateLimit
```

## License

MIT License - see [LICENSE](LICENSE)