package authz

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
)

// Obligation failure taxonomy
var (
	ErrObligationTimeout    = fmt.Errorf("obligation execution timed out")
	ErrObligationNetwork    = fmt.Errorf("obligation network error")
	ErrObligationPermission = fmt.Errorf("obligation permission denied")
	ErrObligationInternal   = fmt.Errorf("obligation internal error")
)

// Obligation represents an action to be performed after a policy decision (e.g., logging, notification, resource update).
// ... (rest of structs)
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

// AuditObligationExecutor implements ObligationExecutor by persisting results to pkg/audit.
type AuditObligationExecutor struct {
	Audit audit.Logger
}

// Execute executes the obligation and calls PersistAudit automatically.
func (e *AuditObligationExecutor) Execute(ob Obligation, ctx map[string]interface{}) error {
	// In a real system, we'd dispatch based on ob.Type (e.g., "log", "email", "webhook").
	// For this implementation, we simulate success unless a param says "fail".
	var err error
	if ob.Params != nil && ob.Params["simulate_failure"] == "true" {
		err = ErrObligationInternal
	}

	// Persist the result to the audit ledger
	e.PersistAudit(ob, ctx, err)

	return err
}

// PersistAudit writes the obligation execution result to the audit log.
func (e *AuditObligationExecutor) PersistAudit(ob Obligation, ctx map[string]interface{}, result error) error {
	if e.Audit == nil {
		return nil
	}

	status := "success"
	resStr := "success"
	if result != nil {
		status = "failure"
		resStr = result.Error()
	}

	// Create audit event metadata
	metadata := make(map[string]interface{})
	for k, v := range ob.Params {
		metadata["param_"+k] = v
	}
	for k, v := range ctx {
		metadata["ctx_"+k] = v
	}
	metadata["mandatory"] = ob.Mandatory
	if result != nil {
		metadata["error"] = resStr
	}

	event := &audit.Event{
		Type:      audit.EventTypeObligation,
		Timestamp: time.Now().UTC(),
		Subject:   fmt.Sprintf("%v", ctx["request_subject"]),
		Object:    ob.ID,
		Action:    ob.Type,
		Result:    status,
		Metadata:  metadata,
		Severity:  "info",
	}
	if ob.Mandatory && result != nil {
		event.Severity = "warning"
	}

	return e.Audit.Log(context.Background(), event)
}

// DefaultObligationExecutor is a legacy stub implementation.
type DefaultObligationExecutor struct{}

func (e *DefaultObligationExecutor) Execute(obligation Obligation, context map[string]interface{}) error {
	return nil
}

func (e *DefaultObligationExecutor) PersistAudit(obligation Obligation, context map[string]interface{}, result error) error {
	status := "success"
	if result != nil {
		status = "failure"
	}
	fmt.Printf("[AUDIT] Obligation %s type=%s status=%s\n", obligation.ID, obligation.Type, status)
	return nil
}
