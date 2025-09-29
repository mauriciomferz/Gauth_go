/*
Package resource provides internal types and functionality for resource management in GAuth.

This package defines various resource types and operations that can be performed on them.
Resources represent protected entities in the system that are subject to authentication
and authorization checks. Unlike the public pkg/resources package, this internal package
implements the core functionality and data structures.

Key types in this package:

  - Resource: The main resource type that contains identification and metadata
  - Config: Strongly-typed configuration for resources with type safety
  - ResourceType: Enumerated resource types (API, Service, Database, etc.)
  - RateLimit: Type-safe configuration for rate limiting on resources
  - Attributes: Strongly typed resource attributes
  - Hierarchy: Resource parent-child relationship management
  - Scopes: Resource access scope definitions

Usage example:

	// Create a new API resource with typed configuration
	config := &Config{
		Version:        "v1",
		Public:         true,
		MaxConnections: 100,
		RateLimit: &RateLimit{
			RequestsPerMinute: 1000,
			BurstSize:        50,
		},
	}
	resource := resource.NewResource("payment-api", resource.TypeAPI, config)

	// Add strongly typed metadata
	resource.SetAttributes(&Attributes{
		Owner:       "platform-team",
		Department:  "finance",
		Environment: "production",
		Tags:        []string{"critical", "payment"},
	})

The Resource type offers a flexible yet type-safe way to define protected resources
in your authentication system, avoiding the use of map[string]interface{} for improved
type safety and developer experience.

This package provides internal implementation details for the public pkg/resources package,
following the principle of exposing clean interfaces in the public API while keeping
complex implementation details internal. This separation ensures:

1. Public API stability
2. Implementation flexibility
3. Proper encapsulation
4. Maintainable code organization

Relationship to public packages:
- pkg/resources: Public API exposing resource management functionality
- internal/resource: Implementation details and data structures
- pkg/auth: Uses resources for authentication decisions
- pkg/authz: Uses resources for authorization policies
*/
package resource
