# GAuth Examples Guide

This document provides an overview of the examples included in the GAuth codebase, highlighting how they demonstrate best practices and showcase the library's capabilities.

## Example Structure

Each example in the `examples/` directory follows a consistent structure:

1. **README.md**: Explains the purpose of the example and key concepts
2. **main.go**: Contains the example code with extensive comments
3. **Additional files**: As needed for more complex examples

## Example Categories

### 1. Core Functionality Examples

- **basic**: Simple authentication and authorization flow
- **auth**: Authentication examples (local, OAuth, OIDC)
- **authz**: Authorization examples (RBAC, ABAC)
- **token**: Token creation, validation, and management

### 2. Integration Examples

- **custom_server**: Integrating GAuth with a custom HTTP server
- **microservices**: Using GAuth in a microservices architecture
- **gateway**: Implementing API gateway patterns with GAuth

### 3. Advanced Pattern Examples

- **rate**: Rate limiting patterns and configurations
- **resilience**: Building resilient authentication services
- **errors**: Error handling and custom error types
- **events**: Event-driven authentication and authorization

### 4. Domain-Specific Examples

- **legal_framework**: Implementing regulatory frameworks
- **cascade**: Cascading authentication and authorization
- **audit**: Comprehensive audit logging

## Featured Examples

### Legal Framework Example

The `examples/legal_framework` example demonstrates how to implement a regulatory framework for financial services using GAuth's type-safe components:

```go
// Create a new legal framework
framework := auth.NewLegalFramework(
    "financial-services-framework",
    "Financial Services Regulatory Framework",
    "1.0.0",
)

// Add policies, authorities, and rules
// ...

// Make authorization decisions
decision, err := framework.Authorize(ctx, "customer-123", "account-456", "transfer")
```

Key features demonstrated:
- Type-safe structures for policies and rules
- Helper methods for data access
- Integration with audit logging
- Regulatory compliance patterns

### Rate Limiting Example

The `examples/rate` example shows how to use the different rate limiting implementations:

```go
// Basic rate limiter
limiter := ratelimit.NewRateLimiter(ratelimit.RateLimiterConfig{
    RequestLimit:  100,
    ResetInterval: time.Minute,
})

// HTTP middleware integration
app.Use(ratelimit.HTTPRateLimitMiddleware(config))
```

Key features demonstrated:
- Different rate limiting strategies
- HTTP integration
- Configuration options
- Monitoring and metrics

## Running Examples

Each example can be run using:

```bash
cd examples/<example-name>
go run main.go
```

For examples that require external dependencies (like Redis), refer to the example's README.md for setup instructions.

## Creating New Examples

When creating new examples:

1. Follow the established structure
2. Include a comprehensive README.md
3. Add extensive comments to the code
4. Demonstrate best practices and type safety
5. Keep examples focused on specific use cases
6. Include setup instructions for any dependencies

## Example Best Practices

1. **Real-world Scenarios**: Base examples on realistic use cases
2. **Progressive Complexity**: Start simple and build up
3. **Complete Code**: Ensure examples can run standalone
4. **Error Handling**: Demonstrate proper error handling
5. **Type Safety**: Showcase type-safe patterns
6. **Documentation**: Include clear explanations in comments

## Suggested Example Workflow

When using examples to learn GAuth:

1. Start with the basic example
2. Move to domain-specific examples that match your use case
3. Explore integration examples for your architecture
4. Study advanced patterns for specific requirements