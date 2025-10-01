// AccessRequest represents a request to perform an action on a resource

package authz

import (
	"context"
	"time"
)

// Subject represents an entity requesting access (RFC111: power-of-attorney grantee, e.g. AI, user, or service)
type Subject struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Roles      []string          `json:"roles"`
	Attributes map[string]string `json:"attributes"`
	Groups     []string          `json:"groups"`
}

// Resource represents a protected resource (RFC111: object of action/decision, e.g. data, service, asset)
type Resource struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Owner      string            `json:"owner"`
	Attributes map[string]string `json:"attributes"`
	Tags       []string          `json:"tags"`
}

// Action represents an operation on a resource (RFC111: operation/transaction/decision to be authorized)
type Action struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

// Policy represents an authorization policy (RFC111: formalizes power-of-attorney, scope, restrictions, and conditions)
type Policy struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Effect determines whether the policy allows or denies
	Effect Effect `json:"effect"`

	// Who this policy applies to
	Subjects []Subject `json:"subjects"`

	// What resources this policy protects
	Resources []Resource `json:"resources"`

	// What actions are covered
	Actions []Action `json:"actions"`

	// Conditions that must be satisfied (named)
	Conditions map[string]Condition `json:"conditions"`

	// Priority determines policy evaluation order
	Priority int `json:"priority"`

	// Status of the policy
	Status string `json:"status"`
}

// AccessRequest represents a request to perform an action on a resource
// (RFC111: credentialized request, e.g. for transaction, decision, or action)
type AccessRequest struct {
	Subject  Subject           `json:"subject"`
	Resource Resource          `json:"resource"`
	Action   Action            `json:"action"`
	Context  map[string]string `json:"context,omitempty"`
}

// AccessResponse represents the result of an access check (RFC111: result of PDP
// evaluation, with annotations for audit/compliance)
type AccessResponse struct {
	Allowed     bool              `json:"allowed"`
	Reason      string            `json:"reason"`
	PolicyID    string            `json:"policy_id,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Effect represents the policy effect (RFC111: allow/deny decision)
type Effect string

const (
	// Allow grants access
	Allow Effect = "allow"

	// Deny blocks access
	Deny Effect = "deny"
)

// Condition represents a policy condition interface (RFC111: additional requirements for power-of-attorney, e.g. time, IP, role)
type Condition interface {
	Evaluate(ctx context.Context, request *AccessRequest) (bool, error)
}

// Decision represents an authorization decision (RFC111: PDP output, includes reason, policy, timestamp)
type Decision struct {
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason"`
	Policy    string    `json:"policy"`
	Timestamp time.Time `json:"timestamp"`
}

// Authorizer evaluates authorization requests (RFC111: PDP interface, central authority for all decisions)
type Authorizer interface {
	// Authorize determines if a subject can perform an action on a resource
	Authorize(ctx context.Context, subject Subject, action Action, resource Resource) (*Decision, error)

	// AddPolicy adds or updates a policy
	AddPolicy(ctx context.Context, policy *Policy) error

	// RemovePolicy removes a policy
	RemovePolicy(ctx context.Context, policyID string) error

	// ListPolicies returns all policies
	ListPolicies(ctx context.Context) ([]*Policy, error)
}

// PolicyStore manages policy persistence (RFC111: PAP/PIP, manages policy lifecycle and retrieval)
type PolicyStore interface {
	// Store stores a policy
	Store(ctx context.Context, policy *Policy) error

	// Get retrieves a policy by ID
	Get(ctx context.Context, id string) (*Policy, error)

	// Delete removes a policy
	Delete(ctx context.Context, id string) error

	// List returns all policies
	List(ctx context.Context) ([]*Policy, error)
}

// PolicyEvaluator evaluates policies (RFC111: PDP logic, evaluates policy against request)
type PolicyEvaluator interface {
	// Evaluate evaluates a policy against a request
	Evaluate(ctx context.Context, policy *Policy, subject Subject, action Action, resource Resource) (*Decision, error)
}

// AuthzError represents authorization specific errors (RFC111: error handling for compliance and audit)
type Error struct {
	Code    string // Error code
	Message string // Error message
	Details string // Additional details
}

func (e *Error) Error() string {
	return "authz: " + e.Code + ": " + e.Message
}
