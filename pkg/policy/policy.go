package policy

// Engine defines the interface for evaluating authorization & delegation policies.
// Implementations might wrap OPA (Rego) or a CEL interpreter.
type Engine interface {
	// EvaluateAuthorization returns a decision (allow/deny) and an optional reason code.
	EvaluateAuthorization(input AuthzInput) (Decision, error)
	// EvaluateDelegation validates a proposed delegation request under current policies.
	EvaluateDelegation(input DelegationInput) (Decision, error)
	// Reload forces a reload of policy sources (filesystem, bundle, remote, etc.).
	Reload() error
}

// Decision captures the outcome of a policy evaluation.
type Decision struct {
	Allow      bool
	ReasonCode string            // machine-readable denial / rationale code
	Metadata   map[string]string // optional additional data (e.g. policy version)
}

// AuthzInput is the structured context passed to the policy engine for resource authorization decisions.
type AuthzInput struct {
	Subject    string            // user / service principal id
	Action     string            // verb (read, write, delegate, etc.)
	Resource   string            // resource identifier
	Scopes     []string          // granted scopes
	Attributes map[string]string // arbitrary contextual attributes
}

// DelegationInput represents a request to delegate authority or rights.
type DelegationInput struct {
	PrincipalID  string
	DelegateID   string
	Scopes       []string
	Jurisdiction string
	ValidityDays int
}

// StubEngine is a no-op implementation used during early hardening phases.
// It allows all operations while recording last inputs for inspection.
type StubEngine struct {
	LastAuthzInput      *AuthzInput
	LastDelegationInput *DelegationInput
	PolicyVersion       string
}

func NewStubEngine() *StubEngine {
	return &StubEngine{PolicyVersion: "stub-0"}
}

func (e *StubEngine) EvaluateAuthorization(input AuthzInput) (Decision, error) {
	e.LastAuthzInput = &input
	return Decision{Allow: true, ReasonCode: "ALLOW_STUB", Metadata: map[string]string{"policyVersion": e.PolicyVersion}}, nil
}

func (e *StubEngine) EvaluateDelegation(input DelegationInput) (Decision, error) {
	e.LastDelegationInput = &input
	return Decision{Allow: true, ReasonCode: "ALLOW_STUB", Metadata: map[string]string{"policyVersion": e.PolicyVersion}}, nil
}

func (e *StubEngine) Reload() error { return nil }
