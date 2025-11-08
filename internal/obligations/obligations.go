// Package obligations provides obligation and advice processing for authorization decisions (RFC 0111 sec2.item3).
package obligations

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Obligation represents a mandatory action that must be performed.
type Obligation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "log", "notify", "transform", "validate"
	Action      string                 `json:"action"`
	Target      string                 `json:"target,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Priority    int                    `json:"priority"` // Higher = more important
	CreatedAt   time.Time              `json:"created_at"`
	Status      string                 `json:"status"` // "pending", "fulfilled", "failed"
	FulfilledAt *time.Time             `json:"fulfilled_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// Advice represents an optional recommendation.
type Advice struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // "recommendation", "warning", "info"
	Message    string                 `json:"message"`
	Severity   string                 `json:"severity"` // "low", "medium", "high"
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ExecutionResult represents the outcome of obligation/advice processing.
type ExecutionResult struct {
	ObligationID string                 `json:"obligation_id,omitempty"`
	AdviceID     string                 `json:"advice_id,omitempty"`
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
	ExecutedAt   time.Time              `json:"executed_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Handler processes obligations and advice.
type Handler interface {
	// ExecuteObligation fulfills a mandatory obligation.
	ExecuteObligation(ctx context.Context, obligation *Obligation) (*ExecutionResult, error)

	// ProcessAdvice handles advisory recommendations.
	ProcessAdvice(ctx context.Context, advice *Advice) (*ExecutionResult, error)

	// RegisterObligationHandler registers a handler for a specific obligation type.
	RegisterObligationHandler(obligationType string, handler ObligationHandler) error

	// RegisterAdviceHandler registers a handler for a specific advice type.
	RegisterAdviceHandler(adviceType string, handler AdviceHandler) error
}

// ObligationHandler executes specific obligation logic.
type ObligationHandler func(ctx context.Context, obligation *Obligation) error

// AdviceHandler processes specific advice logic.
type AdviceHandler func(ctx context.Context, advice *Advice) error

// DefaultHandler provides basic obligation and advice processing.
type DefaultHandler struct {
	obligationHandlers map[string]ObligationHandler
	adviceHandlers     map[string]AdviceHandler
}

// NewDefaultHandler creates a handler with standard obligation/advice processors.
func NewDefaultHandler() *DefaultHandler {
	handler := &DefaultHandler{
		obligationHandlers: make(map[string]ObligationHandler),
		adviceHandlers:     make(map[string]AdviceHandler),
	}

	// Register default obligation handlers
	if err := handler.RegisterObligationHandler("log", logObligation); err != nil {
		panic(fmt.Sprintf("failed to register log obligation handler: %v", err))
	}
	if err := handler.RegisterObligationHandler("notify", notifyObligation); err != nil {
		panic(fmt.Sprintf("failed to register notify obligation handler: %v", err))
	}
	if err := handler.RegisterObligationHandler("validate", validateObligation); err != nil {
		panic(fmt.Sprintf("failed to register validate obligation handler: %v", err))
	}

	// Register default advice handlers
	if err := handler.RegisterAdviceHandler("recommendation", logAdvice); err != nil {
		panic(fmt.Sprintf("failed to register recommendation advice handler: %v", err))
	}
	if err := handler.RegisterAdviceHandler("warning", logAdvice); err != nil {
		panic(fmt.Sprintf("failed to register warning advice handler: %v", err))
	}
	if err := handler.RegisterAdviceHandler("info", logAdvice); err != nil {
		panic(fmt.Sprintf("failed to register info advice handler: %v", err))
	}

	return handler
}

// ExecuteObligation fulfills an obligation.
func (h *DefaultHandler) ExecuteObligation(ctx context.Context, obligation *Obligation) (*ExecutionResult, error) {
	handler, exists := h.obligationHandlers[obligation.Type]
	if !exists {
		return &ExecutionResult{
			ObligationID: obligation.ID,
			Success:      false,
			Message:      fmt.Sprintf("No handler registered for obligation type: %s", obligation.Type),
			ExecutedAt:   time.Now(),
		}, fmt.Errorf("unknown obligation type: %s", obligation.Type)
	}

	err := handler(ctx, obligation)
	if err != nil {
		return &ExecutionResult{
			ObligationID: obligation.ID,
			Success:      false,
			Message:      err.Error(),
			ExecutedAt:   time.Now(),
		}, err
	}

	return &ExecutionResult{
		ObligationID: obligation.ID,
		Success:      true,
		Message:      fmt.Sprintf("Obligation %s fulfilled successfully", obligation.Type),
		ExecutedAt:   time.Now(),
	}, nil
}

// ProcessAdvice handles advisory recommendations.
func (h *DefaultHandler) ProcessAdvice(ctx context.Context, advice *Advice) (*ExecutionResult, error) {
	handler, exists := h.adviceHandlers[advice.Type]
	if !exists {
		// Advice is optional, so missing handler is not an error
		log.Printf("No handler for advice type %s, skipping", advice.Type)
		return &ExecutionResult{
			AdviceID:   advice.ID,
			Success:    true,
			Message:    "Advice skipped (no handler)",
			ExecutedAt: time.Now(),
		}, nil
	}

	err := handler(ctx, advice)
	if err != nil {
		// Advice failures are logged but don't fail the operation
		log.Printf("Advice processing failed (non-critical): %v", err)
		return &ExecutionResult{
			AdviceID:   advice.ID,
			Success:    false,
			Message:    err.Error(),
			ExecutedAt: time.Now(),
		}, nil
	}

	return &ExecutionResult{
		AdviceID:   advice.ID,
		Success:    true,
		Message:    fmt.Sprintf("Advice %s processed successfully", advice.Type),
		ExecutedAt: time.Now(),
	}, nil
}

// RegisterObligationHandler adds a handler for a specific obligation type.
func (h *DefaultHandler) RegisterObligationHandler(obligationType string, handler ObligationHandler) error {
	h.obligationHandlers[obligationType] = handler
	return nil
}

// RegisterAdviceHandler adds a handler for a specific advice type.
func (h *DefaultHandler) RegisterAdviceHandler(adviceType string, handler AdviceHandler) error {
	h.adviceHandlers[adviceType] = handler
	return nil
}

// Default handlers

func logObligation(ctx context.Context, obligation *Obligation) error {
	log.Printf("[OBLIGATION] %s: action=%s, target=%s, params=%v",
		obligation.Type, obligation.Action, obligation.Target, obligation.Parameters)
	return nil
}

func notifyObligation(ctx context.Context, obligation *Obligation) error {
	// Stub: In production, this would send notifications (email, webhook, etc.)
	log.Printf("[NOTIFY] Notification obligation: %s -> %s", obligation.Action, obligation.Target)
	return nil
}

func validateObligation(ctx context.Context, obligation *Obligation) error {
	// Stub: In production, this would perform validation checks
	log.Printf("[VALIDATE] Validation obligation: %s with params %v", obligation.Action, obligation.Parameters)
	return nil
}

func logAdvice(ctx context.Context, advice *Advice) error {
	log.Printf("[ADVICE/%s] %s: %s (severity: %s)",
		advice.Type, advice.ID, advice.Message, advice.Severity)
	return nil
}
