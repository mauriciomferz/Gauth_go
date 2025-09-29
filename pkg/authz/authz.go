package authz

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Authorize determines if a subject can perform an action on a resource
func (a *memoryAuthorizer) Authorize(ctx context.Context, subject Subject, action Action, resource Resource) (*Decision, error) {
	req := &AccessRequest{
		Subject:  subject,
		Action:   action,
		Resource: resource,
	}
	resp, err := a.IsAllowed(ctx, req)
	if err != nil {
		return nil, err
	}
	return &Decision{
		Allowed:   resp.Allowed,
		Reason:    resp.Reason,
		Policy:    resp.PolicyID,
		Timestamp: time.Now(),
	}, nil
}

// Permission represents an action that can be performed on a resource
type Permission string

// Resource represents a protected resource in the system

// Role represents a collection of permissions
type Role string

// Resource represents a protected resource in the system
// type Resource string

// Role represents a collection of permissions
// type Role string

// Subject represents an entity (user, service) that requests access
// type Subject string

// Action represents the type of operation being attempted
// type Action string

// Policy defines access rules for resources
// type Policy struct {
//       ID         string               // Unique identifier for the policy
//       Version    string               // Policy version for tracking changes
//       Effect     string               // Allow or Deny
//       Subjects   []Subject            // Who this policy applies to
//       Resources  []Resource           // What resources this policy protects
//       Actions    []Action             // What actions are covered
//       Conditions map[string]Condition // Additional conditions that must be met
//       Priority   int                  // Policy priority (higher takes precedence)
//       Metadata   map[string]string    // Additional policy metadata
// }

// Condition represents a function that evaluates additional access requirements
// type Condition interface {
//       // Evaluate returns true if the condition is met
//       Evaluate(ctx context.Context, request *AccessRequest) (bool, error)
// }

// memoryAuthorizer implements Authorizer with in-memory storage
type memoryAuthorizer struct {
	policies     sync.Map // map[string]*Policy
	roles        sync.Map // map[Role][]Permission
	assignments  sync.Map // map[Subject][]Role
	_conditions  sync.Map // map[string]Condition - reserved for dynamic conditions
}

// NewMemoryAuthorizer creates a new in-memory authorizer
func NewMemoryAuthorizer() Authorizer {
	return &memoryAuthorizer{}
}

func (a *memoryAuthorizer) AddPolicy(_ context.Context, policy *Policy) error {
	if policy == nil || policy.ID == "" {
		return errors.New("policy ID is required")
	}
	if _, exists := a.policies.LoadOrStore(policy.ID, policy); exists {
		return fmt.Errorf("policy %s already exists", policy.ID)
	}
	return nil
}

func (a *memoryAuthorizer) RemovePolicy(_ context.Context, policyID string) error {
	if _, exists := a.policies.LoadAndDelete(policyID); !exists {
		return fmt.Errorf("policy %s not found", policyID)
	}
	return nil
}

func (a *memoryAuthorizer) UpdatePolicy(_ context.Context, policy *Policy) error {
	if policy.ID == "" {
		return errors.New("policy ID is required")
	}

	a.policies.Store(policy.ID, policy)
	return nil
}

func (a *memoryAuthorizer) GetPolicy(_ context.Context, policyID string) (*Policy, error) {
	if val, ok := a.policies.Load(policyID); ok {
		return val.(*Policy), nil
	}
	return nil, fmt.Errorf("policy %s not found", policyID)
}

func (a *memoryAuthorizer) ListPolicies(_ context.Context) ([]*Policy, error) {
	var policies []*Policy
	a.policies.Range(func(_, value interface{}) bool {
		policy := value.(*Policy)
		policies = append(policies, policy)
		return true
	})
	return policies, nil
}

func (a *memoryAuthorizer) IsAllowed(ctx context.Context, request *AccessRequest) (*AccessResponse, error) {
	var matchingPolicies []*Policy

	// Collect all applicable policies
	a.policies.Range(func(_, value interface{}) bool {
		policy := value.(*Policy)

		// Check if policy applies to this request
		if a.policyApplies(policy, request) {
			matchingPolicies = append(matchingPolicies, policy)
		}
		return true
	})

	// Sort policies by priority
	sortPoliciesByPriority(matchingPolicies)

	// Evaluate policies in order
	for _, policy := range matchingPolicies {
		allowed, reason := a.evaluatePolicy(ctx, policy, request)
		if allowed || policy.Effect == "deny" {
			return &AccessResponse{
				Allowed:     allowed,
				Reason:      reason,
				PolicyID:    policy.ID,
				Annotations: make(map[string]string),
			}, nil
		}
	}

	// Default deny if no policies match
	return &AccessResponse{
		Allowed:     false,
		Reason:      "no matching policies found",
		Annotations: make(map[string]string),
	}, nil
}

func (a *memoryAuthorizer) AddRole(_ context.Context, role Role, permissions []Permission) error {
	if _, exists := a.roles.LoadOrStore(role, permissions); exists {
		return fmt.Errorf("role %s already exists", role)
	}
	return nil
}

func (a *memoryAuthorizer) RemoveRole(_ context.Context, role Role) error {
	if _, exists := a.roles.LoadAndDelete(role); !exists {
		return fmt.Errorf("role %s not found", role)
	}
	return nil
}

func (a *memoryAuthorizer) AssignRole(_ context.Context, subject Subject, role Role) error {
	// Check if role exists
	if _, exists := a.roles.Load(role); !exists {
		return fmt.Errorf("role %s not found", role)
	}

	var roles []Role
	if val, ok := a.assignments.Load(subject); ok {
		roles = val.([]Role)
		// Check if role is already assigned
		for _, r := range roles {
			if r == role {
				return nil
			}
		}
	}

	roles = append(roles, role)
	a.assignments.Store(subject, roles)
	return nil
}

func (a *memoryAuthorizer) UnassignRole(ctx context.Context, subject Subject, role Role) error {
	if val, ok := a.assignments.Load(subject); ok {
		roles := val.([]Role)
		for i, r := range roles {
			if r == role {
				roles = append(roles[:i], roles[i+1:]...)
				if len(roles) == 0 {
					a.assignments.Delete(subject)
				} else {
					a.assignments.Store(subject, roles)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("role %s not assigned to subject %s", role, subject)
}

func (a *memoryAuthorizer) GetRoles(ctx context.Context, subject Subject) ([]Role, error) {
	if val, ok := a.assignments.Load(subject); ok {
		return val.([]Role), nil
	}
	return []Role{}, nil
}

// Helper functions

func (a *memoryAuthorizer) policyApplies(policy *Policy, request *AccessRequest) bool {
	return subjectMatches(policy.Subjects, request.Subject) &&
		resourceMatches(policy.Resources, request.Resource) &&
		actionMatches(policy.Actions, request.Action)
}

func (a *memoryAuthorizer) evaluatePolicy(ctx context.Context, policy *Policy, request *AccessRequest) (bool, string) {
	// Evaluate all conditions
	for name, condition := range policy.Conditions {
		allowed, err := condition.Evaluate(ctx, request)
		if err != nil {
			return false, fmt.Sprintf("condition %s evaluation failed: %v", name, err)
		}
		if !allowed {
			return false, fmt.Sprintf("condition %s not met", name)
		}
	}

	if policy.Effect == "allow" {
		return true, "policy allows access"
	}
	return false, "policy denies access"
}

func subjectMatches(policySubjects []Subject, requestSubject Subject) bool {
	if len(policySubjects) == 0 {
		return true
	}
	for _, subject := range policySubjects {
		if subject.ID == "*" || subject.ID == requestSubject.ID {
			return true
		}
	}
	return false
}

func resourceMatches(policyResources []Resource, requestResource Resource) bool {
	if len(policyResources) == 0 {
		return true
	}
	for _, resource := range policyResources {
		if resource.ID == "*" || resource.ID == requestResource.ID {
			return true
		}
		// Handle path pattern matching
		if resourcePathMatches(resource.ID, requestResource.ID) {
			return true
		}
	}
	return false
}

// resourcePathMatches checks if a resource pattern matches a resource path
func resourcePathMatches(pattern, path string) bool {
	// Handle wildcard patterns like "/docs/*" matching "/docs/secret"
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}
	// Handle exact wildcards
	if pattern == "*" {
		return true
	}
	return false
}

func actionMatches(policyActions []Action, requestAction Action) bool {
	if len(policyActions) == 0 {
		return true
	}
	for _, action := range policyActions {
		if action.Name == "*" || action.Name == requestAction.Name {
			return true
		}
	}
	return false
}

func sortPoliciesByPriority(_ []*Policy) {
	// Implement sorting by priority (higher priority first)
}
