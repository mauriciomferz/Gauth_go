package authz

import (
	"fmt"
)

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
	status := "success"
	if result != nil {
		status = "failure"
	}
	// TODO: Integrate with pkg/audit for durable storage. For now, log to stdout.
	fmt.Printf("[AUDIT] Obligation %s type=%s status=%s\n", obligation.ID, obligation.Type, status)
	return nil
}
