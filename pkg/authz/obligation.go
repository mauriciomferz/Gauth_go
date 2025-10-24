package authz

// Obligation represents an action to be performed after a policy decision (e.g., logging, notification, resource update).
type Obligation struct {
	ID        string            // Unique identifier
	Type      string            // e.g., "log", "notify", "update"
	Params    map[string]string // Arbitrary parameters
	Mandatory bool              // If true, failure may flip an allow decision to deny
}

// Advice represents a non-mandatory recommendation to the client after a policy decision.
type Advice struct {
	ID     string
	Type   string
	Params map[string]string
}

// ObligationExecutor executes obligations and persists audit records.
type ObligationExecutor interface {
	Execute(obligation Obligation, context map[string]interface{}) error
	PersistAudit(obligation Obligation, context map[string]interface{}, result error) error
}

// DefaultObligationExecutor is a stub implementation for demo/testing.
type DefaultObligationExecutor struct{}

func (e *DefaultObligationExecutor) Execute(obligation Obligation, context map[string]interface{}) error {
	// TODO: Implement real execution logic (logging, notification, etc.)
	return nil
}

func (e *DefaultObligationExecutor) PersistAudit(obligation Obligation, context map[string]interface{}, result error) error {
	// TODO: Persist audit record (to DB, file, etc.)
	return nil
}
