[![Go CI](https://github.com/mauriciomferz/Gauth_go/actions/workflows/go-ci.yml/badge.svg)](https://github.com/mauriciomferz/Gauth_go/actions/workflows/go-ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/mauriciomferz/Gauth_go)](https://goreportcard.com/report/github.com/mauriciomferz/Gauth_go) [![codecov](https://codecov.io/gh/mauriciomferz/Gauth_go/branch/main/graph/badge.svg)](https://codecov.io/gh/mauriciomferz/Gauth_go)

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


## Quick Start

Here’s a minimal example to get started with GAuth:

```go
import (
    "github.com/mauriciomferz/Gauth_go/pkg/gauth"
    "github.com/mauriciomferz/Gauth_go/pkg/token"
)

func main() {
    config := &gauth.Config{
        AuthServerURL:     "https://auth.example.com",
        ClientID:          "your-client-id",
        ClientSecret:      "your-secret",
        Scopes:            []string{"read", "write"},
        TokenConfig: &token.Config{
            SigningMethod: token.HS256,
            SigningKey:    []byte("your-signing-key"),
        },

This diagram summarizes the main system components and their interactions. For a much more detailed, layered architecture—including the Public API, Core Services, Storage, and Integration layers—see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). That document also includes:

- Layered architecture diagrams (ASCII and Mermaid)
- Sequence diagrams for token and resource flows
- Explanations of Power*Point (P*P) roles (PEP, PDP, PIP, PAP, PVP)
- Data flow and extension points
- Security, monitoring, and best practices

Refer to [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full technical breakdown and visual diagrams.
    }
    auth, err := gauth.New(config, nil)
    if err != nil {
        panic(err)
    }
    // Use auth to initiate authorization, request tokens, etc.
}
```

For more advanced usage, see the `examples/` directory.

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
7. **Code Organization**: Library code and examples are clearly separated.

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

## Architecture Diagram

Below is a high-level architecture diagram of GAuth:

```
+-------------------+        +-------------------+        +-------------------+
|    Client App     | <----> |   GAuth Service   | <----> |   Token Store     |
+-------------------+        +-------------------+        +-------------------+
        |                          |                              |
        |   (OAuth2/OIDC/MCP)      |   (Audit/Event/Policy)       |
        v                          v                              v
+-------------------+        +-------------------+        +-------------------+
|   Resource/API    | <----> |   Audit Logger    | <----> |   Event System    |
+-------------------+        +-------------------+        +-------------------+
```

For a detailed diagram, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).


## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for full guidelines. Here’s a summary of the most important points:

- **Development Setup:**
    1. Clone the repository
    2. Install dependencies: `go mod download`
    3. Run tests: `go test ./...`
    4. Try examples: `cd examples/basic && go run main.go`

- **Code Organization:**
    - Library code lives in `/pkg` (public API), `/internal` (private), `/examples` (demos), `/cmd` (CLI/demo apps)
    - Each package uses `doc.go` for documentation, `interfaces.go` for public interfaces, and focused implementation files
    - Tests are in `_test.go` files

- **Code Style:**
    - Follow standard Go conventions and use `gofmt`
    - All exported functions/types must have GoDoc comments
    - Use type-safe APIs (avoid `map[string]interface{}` in public APIs)
    - Write meaningful commit messages and include tests for new features

- **Pull Requests:**
    - Fork the repo, create a branch from `main`, and submit a PR via GitHub Flow
    - Update documentation and tests as needed
    - Ensure all CI checks pass before requesting review
    - Update `README.md` and `CHANGELOG.md` for interface changes
    - PRs require sign-off from two maintainers

- **Issue Reporting:**
    - Use [GitHub Issues](https://github.com/Gimel-Foundation/gauth/issues) to report bugs or request features
    - Provide clear, actionable details in your report

- **Community:**
    - Join discussions, ask questions, and help review PRs
    - Be respectful and constructive (see Code of Conduct if present)

For more, see [CONTRIBUTING.md](CONTRIBUTING.md), [GETTING_STARTED.md](docs/GETTING_STARTED.md), and [LIBRARY.md](LIBRARY.md).

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


## More Usage Examples

GAuth provides a comprehensive set of real-world, runnable examples in the [`examples/`](examples/) directory. These cover everything from basic authentication to advanced delegation, restrictions, and batch operations. Below are some highlights and new advanced scenarios:

### Basic Authentication Example

```go
import (
    "github.com/mauriciomferz/Gauth_go/pkg/gauth"
    "github.com/mauriciomferz/Gauth_go/pkg/token"
)

func main() {
    config := &gauth.Config{
        AuthServerURL:     "https://auth.example.com",
        ClientID:          "example-client",
        ClientSecret:      "example-secret",
        Scopes:            []string{"transaction:execute", "read", "write"},
        TokenConfig: &token.Config{SigningMethod: token.RS256},
    }
    auth, err := gauth.New(config, nil)
    if err != nil {
        panic(err)
    }
    // Initiate authorization
    grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
        ClientID: "example-client",
        Scopes:   []string{"payment:execute"},
    })
    if err != nil {
        panic(err)
    }
    // Use grant.GrantID to request tokens, etc.
}
```

See [`examples/basic/main.go`](examples/basic/main.go) for a full runnable example and comments.

### Advanced: Multi-Scope, Restrictions, and Batch Processing

```go
// ...imports...

func main() {
    // ...config setup...
    // Multi-scope authorization
    grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{
        ClientID: "advanced-client",
        Scopes:   []string{"read", "write", "transaction:execute"},
    })
    // Token request with restrictions
    tokenResp, _ := auth.RequestToken(gauth.TokenRequest{
        GrantID: grant.GrantID,
        Scope:   []string{"read", "write", "transaction:execute"},
        Restrictions: []gauth.Restriction{
            {Type: "ip_range", Value: "192.168.1.0/24"},
            {Type: "time_window", Value: "business_hours"},
        },
    })
    // Batch transaction processing
    transactions := []gauth.TransactionDetails{
        {Type: "payment", Amount: 100.0, ResourceID: "resource-1"},
        {Type: "transfer", Amount: 50.0, ResourceID: "resource-2"},
    }
    for _, tx := range transactions {
        _, err := server.ProcessTransaction(tx, tokenResp.Token)
        if err != nil {
            // handle error
        }
    }
}
```

See [`examples/advanced/main.go`](examples/advanced/main.go) for a full runnable example and comments.

### Edge Case: Custom Restrictions and Delegation

```go
// ...imports...

func main() {
    // Custom restriction: Only allow transactions above a certain amount during business hours
    grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{
        ClientID: "custom-client",
        Scopes:   []string{"transaction:execute"},
        Restrictions: []gauth.Restriction{
            {Type: "min_amount", Value: "100.00"},
            {Type: "time_window", Value: "business_hours"},
        },
    })
    // Delegation: Grant access to a sub-agent
    delegatedGrant, _ := auth.DelegateAuthorization(gauth.DelegationRequest{
        GrantID: grant.GrantID,
        DelegateTo: "sub-agent-123",
        Scopes: []string{"transaction:execute"},
    })
    // Use delegatedGrant.GrantID for sub-agent actions
}
```

See [`examples/advanced_delegation_attestation/main.go`](examples/advanced_delegation_attestation/main.go) for a full example of delegation and attestation flows.

For more, browse the [`examples/`](examples/) directory for topics such as:
- Typed events and metadata
- Error handling and propagation
- Audit logging
- Rate limiting
- Resilience patterns
- API gateway integration

Each example is runnable and demonstrates a specific feature or integration pattern.