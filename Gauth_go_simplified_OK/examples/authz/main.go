package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Gimel-Foundation/gauth/pkg/authz"
)

func main() {
	runRBAC()
	runPolicy()
	runABAC()
	runDistributed()
	fmt.Println("\nAll authorization examples complete.")
}

func runRBAC() {
	fmt.Println("\n=== RBAC Example ===")
	ctx := context.Background()
	az := authz.NewMemoryAuthorizer()

	// Represent roles as policies
	adminPolicy := &authz.Policy{
		ID:        "admin-role",
		Effect:    authz.Allow,
		Subjects:  []authz.Subject{{ID: "admin1", Roles: []string{"admin"}}},
		Resources: []authz.Resource{{ID: "users"}},
		Actions:   []authz.Action{{Name: "read"}, {Name: "write"}},
	}
	userPolicy := &authz.Policy{
		ID:        "user-role",
		Effect:    authz.Allow,
		Subjects:  []authz.Subject{{ID: "user123", Roles: []string{"user"}}},
		Resources: []authz.Resource{{ID: "users"}},
		Actions:   []authz.Action{{Name: "read"}},
	}
	if err := az.AddPolicy(ctx, adminPolicy); err != nil {
		log.Fatalf("AddPolicy admin: %v", err)
	}
	if err := az.AddPolicy(ctx, userPolicy); err != nil {
		log.Fatalf("AddPolicy user: %v", err)
	}

	subject := authz.Subject{ID: "user123", Roles: []string{"user"}}
	resource := authz.Resource{ID: "users"}
	action := authz.Action{Name: "read"}

	decision, err := az.Authorize(ctx, subject, action, resource)
	if err != nil {
		log.Fatalf("Authorize: %v", err)
	}
	fmt.Printf("RBAC: Access allowed: %v (reason: %s)\n", decision.Allowed, decision.Reason)
}

func runPolicy() {
	fmt.Println("\n=== Policy-Based Example ===")
	ctx := context.Background()
	az := authz.NewMemoryAuthorizer()

	// Define a policy that allows user123 to read documents
	policy := &authz.Policy{
		ID:        "policy1",
		Effect:    authz.Allow,
		Subjects:  []authz.Subject{{ID: "user123"}},
		Resources: []authz.Resource{{ID: "documents"}},
		Actions:   []authz.Action{{Name: "read"}},
	}
	if err := az.AddPolicy(ctx, policy); err != nil {
		log.Fatalf("AddPolicy: %v", err)
	}

	subject := authz.Subject{ID: "user123"}
	resource := authz.Resource{ID: "documents"}
	action := authz.Action{Name: "read"}

	decision, err := az.Authorize(ctx, subject, action, resource)
	if err != nil {
		log.Fatalf("Authorize: %v", err)
	}
	fmt.Printf("Policy: Access allowed: %v (reason: %s)\n", decision.Allowed, decision.Reason)
}

func runABAC() {
	fmt.Println("\n=== ABAC Example ===")
	ctx := context.Background()
	az := authz.NewMemoryAuthorizer()

	// Define an ABAC policy: allow any user from department "engineering" to read any technical doc
	policy := &authz.Policy{
		ID:        "abac-policy-1",
		Effect:    authz.Allow,
		Subjects:  []authz.Subject{{ID: "*", Type: "user"}}, // Wildcard subject
		Resources: []authz.Resource{{ID: "*", Type: "document", Attributes: map[string]string{"category": "technical"}}},
		Actions:   []authz.Action{{Name: "read"}},
		Conditions: map[string]authz.Condition{
			"department": attrEquals("department", "engineering"),
		},
	}
	if err := az.AddPolicy(ctx, policy); err != nil {
		log.Fatalf("AddPolicy ABAC: %v", err)
	}

	// Subject with matching attribute
	subject := authz.Subject{
		ID:         "alice",
		Type:       "user",
		Attributes: map[string]string{"department": "engineering"},
	}
	resource := authz.Resource{
		ID:         "doc-123",
		Type:       "document",
		Attributes: map[string]string{"category": "technical"},
	}
	action := authz.Action{Name: "read"}

	decision, err := az.Authorize(ctx, subject, action, resource)
	if err != nil {
		log.Fatalf("Authorize ABAC: %v", err)
	}
	fmt.Printf("ABAC: Access allowed for engineering: %v (reason: %s)\n", decision.Allowed, decision.Reason)

	// Subject with non-matching attribute
	subject2 := authz.Subject{
		ID:         "bob",
		Type:       "user",
		Attributes: map[string]string{"department": "marketing"},
	}
	decision2, err := az.Authorize(ctx, subject2, action, resource)
	if err != nil {
		log.Fatalf("Authorize ABAC: %v", err)
	}
	fmt.Printf("ABAC: Access allowed for marketing: %v (reason: %s)\n", decision2.Allowed, decision2.Reason)
}

// attrEquals returns a Condition that checks if a subject attribute equals a value
func attrEquals(attr, value string) authz.Condition {
	return attrEqualsCondition{attr: attr, value: value}
}

type attrEqualsCondition struct {
	attr  string
	value string
}

func (c attrEqualsCondition) Evaluate(ctx context.Context, req *authz.AccessRequest) (bool, error) {
	if req.Subject.Attributes == nil {
		return false, nil
	}
	return req.Subject.Attributes[c.attr] == c.value, nil
}
func runDistributed() {
	fmt.Println("\n=== Distributed Example ===")
	fmt.Println("Distributed authorization demo is a stub. Configure Redis and nodes for a real test.")
}
