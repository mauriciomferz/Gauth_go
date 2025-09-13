/*
Package gauth provides a comprehensive authentication and authorization framework for Go applications.

# Overview

GAuth is designed to provide secure, extensible, and production-ready authentication and authorization
capabilities for modern distributed systems. The library emphasizes:

- Type safety and clear interfaces
- Modular, composable components
- Clean separation of concerns
- Production-ready resilience patterns
- Comprehensive audit logging
- Event-driven architecture with strongly typed metadata

# Core Components

 1. Token Management (pkg/token)
    Handles secure token operations with strong typing:

    tokenService := token.NewService(token.Config{
    SigningMethod: token.RS256,
    SigningKey:    privateKey,
    Store:        store.NewRedisStore(redisOpts),
    })

    // Issue a token
    token, err := tokenService.Issue(ctx, &token.Token{
    Type:    token.Access,
    Subject: "user123",
    Scopes:  []string{"read", "write"},
    })

 2. Authentication (pkg/auth)
    Provides authentication methods and flows:

    authService := auth.NewService(auth.Config{
    TokenService:  tokenService,
    EventHandler: events.NewPublisher(),
    })

    // Protect an endpoint
    http.HandleFunc("/api", authService.Middleware(handler))

 3. Storage (pkg/store)
    Flexible storage backends with encryption support:

    // Memory store for development
    store := store.NewMemoryStore()

    // Redis store for production
    store := store.NewRedisStore(redisOpts)

    // Add encryption
    store = store.NewEncryptedStore(store, encKey)

 4. Events (pkg/events)
    Type-safe event system for auditing and monitoring:

    publisher := events.NewPublisher()
    publisher.Subscribe(events.LogHandler)
    publisher.Subscribe(events.MetricsHandler)

 5. Resilience (pkg/resilience)
    Reliability patterns for distributed systems:

    cb := resilience.NewCircuitBreaker(circuitConfig)
    retry := resilience.NewRetry(retryConfig)
    timeout := resilience.NewTimeout(timeoutConfig)

# Package Organization

The library is organized into focused, well-documented packages:

pkg/

	auth/       - Core authentication functionality
	    providers/   - Authentication providers (Basic, OAuth2, etc.)
	    tokens/     - Token generation and validation
	    claims/     - Claims handling and validation
	    doc.go     - Package documentation

	util/       - Common utility functions and types
	    time_range.go - Type-safe time range functionality
	    doc.go     - Package documentation

	authz/      - Authorization and access control
	    policy/     - Policy definition and evaluation
	    roles/      - Role-based access control
	    scope/      - Scope-based authorization
	    doc.go     - Package documentation

	token/      - Token management and storage
	    store/      - Token storage implementations
	    encryption/ - Token encryption and security
	    rotation/   - Key rotation management
	    doc.go     - Package documentation

	events/     - Event system for auditing/monitoring
	    bus/        - Event publishing and subscription
	    handlers/   - Event handler implementations
	    doc.go     - Package documentation

	resilience/ - Reliability patterns
	    circuit/    - Circuit breaker implementation
	    retry/      - Retry strategies
	    timeout/    - Timeout handlers
	    doc.go     - Package documentation

internal/     - Implementation details

	circuit/    - Circuit breaker internals
	rate/       - Rate limiting algorithms
	events/     - Event system implementation
	audit/      - Audit logging internals

examples/    - Standalone example applications

	    basic/     - Simple authentication flows
	    advanced/  - Complex usage patterns
	    patterns/  - Resilience pattern examples
		  middleware.go - HTTP middleware
		  oauth2.go - OAuth2 integration

		store/      - Storage implementations
		  memory.go - In-memory store
		  redis.go  - Redis store
		  types.go  - Store interfaces

		events/     - Event system
		  publisher.go - Event publishing
		  handlers.go - Event handlers
		  types.go   - Event types

		resilience/ - Reliability patterns
		  circuit.go - Circuit breaker
		  retry.go  - Retry logic
		  timeout.go - Timeout handling

# Getting Started

1. Basic Token Management:

	import (
		"github.com/Gimel-Foundation/gauth/pkg/token"
		"github.com/Gimel-Foundation/gauth/pkg/store"
	)

	// Create token service
	service := token.NewService(token.Config{
		SigningMethod: token.RS256,
		SigningKey:    privateKey,
		Store:        store.NewMemoryStore(),
	})

	// Issue token
	token, err := service.Issue(ctx, &token.Token{
		Type:    token.Access,
		Subject: "user123",
		Scopes:  []string{"read", "write"},
	})

2. HTTP Authentication:

	import "github.com/Gimel-Foundation/gauth/pkg/auth"

	// Create auth middleware
	auth := auth.NewMiddleware(auth.Config{
		TokenService: tokenService,
	})

	// Protect routes
	http.Handle("/api", auth.RequireToken(handler))

3. Event Monitoring:

	import "github.com/Gimel-Foundation/gauth/pkg/events"

	// Create event publisher
	pub := events.NewPublisher()
	pub.Subscribe(events.LogHandler)
	pub.Subscribe(events.MetricsHandler)

# Design Principles

1. Type Safety
  - Strong types for all public interfaces
  - No map[string]interface{} in public APIs
  - Proper error types and handling

2. Clear Boundaries
  - Well-defined package responsibilities
  - Minimal cross-package dependencies
  - Clean interfaces for extension

3. Production Ready
  - Built-in monitoring
  - Resilience patterns
  - Performance optimizations

4. Developer Friendly
  - Consistent interfaces
  - Good documentation
  - Helpful examples

# Contributing

See CONTRIBUTING.md for guidelines on:
- Code organization
- Documentation standards
- Testing requirements
- Review process

# Security

See SECURITY.md for:
- Security policy
- Vulnerability reporting
- Security best practices
*/
package gauth
