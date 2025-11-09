// Package policy provides policy management and storage for authorization policies.
//
// This package implements policy persistence, policy adapters for different
// authorization systems, and policy registry management. It serves as the
// policy administration point (PAP) in the GAuth architecture.
//
// # Policy Storage
//
// The package supports multiple storage backends:
//   - Memory - Fast, non-persistent (testing/development)
//   - File - JSON/YAML file persistence
//   - Database - SQL/NoSQL persistence (future)
//
// # File-Based Policy Store
//
// Using file-based policy storage:
//
//	store, err := policy.NewFilePolicyStore("policies.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Add policy
//	p := &policy.Policy{
//	    ID:       "read-docs",
//	    Version:  1,
//	    Effect:   policy.EffectAllow,
//	    Subjects: []string{"user:alice"},
//	    Resources: []string{"document:*"},
//	    Actions:  []string{"read"},
//	}
//	err = store.Add(p)
//
//	// List policies
//	policies, err := store.List()
//
//	// Get specific policy
//	p, err := store.Get("read-docs")
//
//	// Remove policy
//	err = store.Remove("read-docs")
//
// # Watch for Policy Changes
//
// Monitor policy file for changes:
//
//	store, _ := policy.NewFilePolicyStore("policies.json")
//
//	// Start watching for changes
//	err := store.Start(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Policies are automatically reloaded when file changes
//
// # Policy Registry
//
// Manage multiple policy sources:
//
//	registry := policy.NewRegistry()
//
//	// Register policy sources
//	registry.Register("file", fileStore)
//	registry.Register("database", dbStore)
//	registry.Register("remote", remoteStore)
//
//	// List all policies from all sources
//	allPolicies := registry.ListAll()
//
//	// Get policy by ID (searches all sources)
//	p, source, err := registry.Get("policy-id")
//
// # Policy Validation
//
// Validate policies before storage:
//
//	validator := policy.NewValidator()
//
//	err := validator.Validate(p)
//	if err != nil {
//	    // Handle validation errors
//	    var validationErr *policy.ValidationError
//	    if errors.As(err, &validationErr) {
//	        for _, issue := range validationErr.Issues {
//	            fmt.Printf("Field: %s, Issue: %s\n", issue.Field, issue.Message)
//	        }
//	    }
//	}
//
// Validation checks:
//   - Required fields (ID, Effect)
//   - Valid effect values (Allow/Deny)
//   - Pattern syntax
//   - Expression syntax
//   - Obligation structure
//
// # Authorizer Adapter
//
// Adapt policy store to authorizer interface:
//
//	store := policy.NewFilePolicyStore("policies.json")
//	adapter := policy.NewAuthorizerAdapter(store)
//
//	// Use adapter with authorizer
//	decision, err := adapter.Authorize(request)
//
// The adapter automatically:
//   - Loads policies from store
//   - Evaluates policies against requests
//   - Handles policy updates
//   - Implements authorization interface
//
// # Policy Versioning
//
// Track policy versions for audit and rollback:
//
//	// Create versioned policy
//	p := &policy.Policy{
//	    ID:      "my-policy",
//	    Version: 2,  // Increment version on changes
//	    // ... other fields
//	}
//
//	// Store maintains version history
//	versions, err := store.GetVersions("my-policy")
//
//	// Rollback to previous version
//	err = store.Rollback("my-policy", 1)
//
// # Policy Templates
//
// Use templates for common policy patterns:
//
//	// RBAC template
//	template := policy.RBACTemplate{
//	    Role:     "admin",
//	    Resource: "system:*",
//	    Actions:  []string{"read", "write", "delete"},
//	}
//	p := template.Build()
//
//	// ABAC template
//	template := policy.ABACTemplate{
//	    SubjectAttrs: map[string]string{
//	        "department": "engineering",
//	    },
//	    ResourceAttrs: map[string]string{
//	        "classification": "internal",
//	    },
//	    Actions: []string{"read"},
//	}
//	p := template.Build()
//
// # Policy Import/Export
//
// Import policies from external sources:
//
//	// Import from JSON
//	policies, err := policy.ImportJSON(jsonBytes)
//	for _, p := range policies {
//	    store.Add(p)
//	}
//
//	// Export to JSON
//	policies, _ := store.List()
//	jsonBytes, err := policy.ExportJSON(policies)
//
// Supported formats:
//   - JSON
//   - YAML
//   - Cedar (AWS authorization language)
//
// # Performance Considerations
//
//   - Use memory stores for high-frequency lookups
//   - Enable file watching for automatic policy reloads
//   - Cache compiled policy expressions
//   - Batch policy operations when possible
//   - Use policy versioning for safe updates
//
// # Security Best Practices
//
//   - Validate all policies before storage
//   - Implement access control for policy management
//   - Audit all policy changes
//   - Use version control for policy files
//   - Review policies regularly for over-permissions
//   - Test policies before deploying to production
//   - Implement policy approval workflows
//
// # Thread Safety
//
// All PolicyStore and Registry methods are safe for concurrent use.
// File-based stores use file locking to prevent corruption.
//
// # Test Coverage
//
// This package has 76.9% test coverage, including:
//   - Policy CRUD operations
//   - File watching and auto-reload
//   - Policy validation
//   - Authorizer adapter
//   - Version management
//   - Registry operations
//   - Error path coverage
package policy
