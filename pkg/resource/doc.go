/*
Package resource provides internal types and functionality for resource management in GAuth.

This package defines various resource types and operations that can be performed on them.
Resources represent protected entities in the system that are subject to authentication
and authorization checks. Unlike the public pkg/resources package, this internal package
implements the core functionality and data structures.

Key types in this package:

  - Resource: The main resource type that contains identification and metadata
  - ResourceConfig: Strongly-typed configuration for resources with type safety
  - ResourceType: Enumerated resource types (API, Service, Database, etc.)
  - RateLimit: Type-safe configuration for rate limiting on resources
  - Attributes: Strongly typed resource attributes
  - Hierarchy: Resource parent-child relationship management
  - Scopes: Resource access scope definitions

Usage example:

	// Create a new API resource with typed configuration
	config := &ResourceConfig{
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
*/
package resource
