/*
Package gauth implements the GAuth authorization framework for AI power-of-attorney, as defined in GiFo-RFC-0111 and GiFo-RFC-0115.

Copyright (c) 2025 Gimel Foundation gGmbH i.G.
Licensed under Apache 2.0

Gimel Foundation gGmbH i.G., www.GimelFoundation.com
Operated by Gimel Technologies GmbH
MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert
Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com

GAuth enables AI systems to act on behalf of humans or organizations, with explicit,
verifiable, and auditable power-of-attorney flows. Features complete RFC-0115 PoA-Definition
implementation with type safety and compliance with modern standards.

Key Features:
- Typed APIs for grants, tokens, and events (no map[string]interface{} in public APIs)
- Centralized authorization and delegation, as required by RFC111
- Audit logging and compliance with RFC111 exclusions and flows
- Modular, extensible structure for easy integration and contribution

Getting Started:
- See the README.md for onboarding and entry points
- Explore examples/ and cmd/demo/ for runnable demos
- Review core types and APIs in pkg/gauth/

For more, see the full RFC111 specification and the package documentation in this file.

License:
GAuth is licensed under Apache 2.0. See LICENSE for details. OAuth, OpenID Connect,
and MCP licenses apply to their respective components.

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
		// store/      - [REMOVED] Token storage implementations (use pkg/token instead)
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

	// store/      - [REMOVED] Storage implementations (use pkg/token instead)
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

3. Development Prototype
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

# RFC111 Compliance

This package implements the GiFo-RfC 0111 (GAuth) standard for AI power-of-attorney, delegation, and auditability.
All protocol roles, flows, and exclusions are respected. See https://gimelfoundation.com for the full RFC.

# Advanced Usage: Delegation & Attestation

GAuth supports advanced delegation and attestation flows:

  - Delegation: Grant power-of-attorney to an AI or agent, with explicit scope, restrictions, and validity.
  - Attestation: Require notary/witness or versioned attestation for high-assurance delegation.

Example:

	tok := token.NewDelegatedToken("ai-agent-123", gauth.DelegationOptions{
	    Principal: "owner-456",
	    Scope:     "sign_contract",
	    Restrictions: gauth.Restrictions{MaxValue: 10000},
	    Attestation:  gauth.Attestation{Notary: "notary-xyz", Version: "v2"},
	    ValidUntil:   time.Now().Add(24 * time.Hour),
	})
	// Store the token using the canonical GAuth API, e.g.:
	// err := svc.RequestToken(...)

See LIBRARY.md and examples/ for more advanced flows.
*/
package gauth
