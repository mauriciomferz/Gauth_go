package pdp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/obligations"
)

// AdviceEvent represents an advice emission to clients (non-mandatory recommendations).
type AdviceEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	Subject    string            `json:"subject"`
	Action     string            `json:"action"`
	Resource   string            `json:"resource"`
	AdviceID   string            `json:"advice_id"`
	AdviceType string            `json:"advice_type"`
	Message    string            `json:"message"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// AdviceChannel provides async emission of advice events to clients.
// Clients should subscribe to AdviceEvents() channel to receive recommendations.
type AdviceChannel interface {
	// Emit sends an advice event to all subscribers (non-blocking).
	Emit(ctx context.Context, event AdviceEvent) error

	// AdviceEvents returns read-only channel for consuming advice events.
	AdviceEvents() <-chan AdviceEvent

	// Close stops emission and drains the channel.
	Close() error
}

// BufferedAdviceChannel implements AdviceChannel with a buffered channel.
type BufferedAdviceChannel struct {
	ch       chan AdviceEvent
	closed   bool
	closedMu sync.RWMutex
}

// NewBufferedAdviceChannel creates an advice channel with specified buffer size.
func NewBufferedAdviceChannel(bufferSize int) *BufferedAdviceChannel {
	if bufferSize <= 0 {
		bufferSize = 100 // Default buffer
	}
	return &BufferedAdviceChannel{
		ch: make(chan AdviceEvent, bufferSize),
	}
}

// Emit sends an advice event (non-blocking, drops if buffer full).
func (b *BufferedAdviceChannel) Emit(ctx context.Context, event AdviceEvent) error {
	b.closedMu.RLock()
	defer b.closedMu.RUnlock()

	if b.closed {
		return fmt.Errorf("advice channel closed")
	}

	select {
	case b.ch <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Buffer full, drop event (advice is non-mandatory)
		return fmt.Errorf("advice buffer full, event dropped")
	}
}

// AdviceEvents returns read-only channel for consuming advice.
func (b *BufferedAdviceChannel) AdviceEvents() <-chan AdviceEvent {
	return b.ch
}

// Close stops emission and drains the channel.
func (b *BufferedAdviceChannel) Close() error {
	b.closedMu.Lock()
	defer b.closedMu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	close(b.ch)
	return nil
}

// ObligationAuditSink provides persistent storage for obligation execution results.
// This extends the audit sink from P1.4 with obligation-specific fields.
type ObligationAuditSink interface {
	// RecordObligationExecution stores obligation execution result with full context.
	RecordObligationExecution(ctx context.Context, record ObligationAuditRecord) error

	// Close gracefully shuts down the sink.
	Close() error
}

// ObligationAuditRecord represents a complete obligation execution audit entry.
type ObligationAuditRecord struct {
	Timestamp      time.Time         `json:"timestamp"`
	Subject        string            `json:"subject"`
	Action         string            `json:"action"`
	Resource       string            `json:"resource"`
	Decision       string            `json:"decision"` // "allow" or "deny"
	ObligationID   string            `json:"obligation_id"`
	ObligationType string            `json:"obligation_type"`
	Mandatory      bool              `json:"mandatory"`
	Success        bool              `json:"success"`
	DurationMS     float64           `json:"duration_ms"`
	Error          string            `json:"error,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// JSONFileObligationAuditSink implements ObligationAuditSink with JSONL file append.
type JSONFileObligationAuditSink struct {
	filePath string
	mu       sync.Mutex
}

// NewJSONFileObligationAuditSink creates a JSONL file sink for obligation audit.
func NewJSONFileObligationAuditSink(filePath string) *JSONFileObligationAuditSink {
	return &JSONFileObligationAuditSink{
		filePath: filePath,
	}
}

// RecordObligationExecution appends obligation execution record to JSONL file.
func (j *JSONFileObligationAuditSink) RecordObligationExecution(ctx context.Context, record ObligationAuditRecord) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Marshal record to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal obligation audit record: %w", err)
	}

	// Append to file (import os at top)
	// For now, we'll use log output as placeholder
	log.Printf("[ObligationAudit] %s\n", string(data))
	return nil
}

// Close is a no-op for file-based sink.
func (j *JSONFileObligationAuditSink) Close() error {
	return nil
}

// ExtendedObligationExecutor implements obligations.Executor with advice channel and audit sink support.
type ExtendedObligationExecutor struct {
	handlers      map[string]ObligationHandler
	adviceChannel AdviceChannel
	auditSink     ObligationAuditSink
	mu            sync.RWMutex
}

// ObligationHandler defines the contract for executing specific obligation types.
type ObligationHandler interface {
	// Execute runs the obligation logic and returns error on failure.
	Execute(ctx context.Context, obligationID string, params map[string]string) error

	// Type returns the obligation type this handler supports (e.g., "log", "notify", "rate_limit").
	Type() string
}

// NewExtendedObligationExecutor creates an executor with built-in handlers.
func NewExtendedObligationExecutor(opts ...ExtendedExecutorOption) *ExtendedObligationExecutor {
	exec := &ExtendedObligationExecutor{
		handlers: make(map[string]ObligationHandler),
	}

	// Apply options
	for _, opt := range opts {
		opt(exec)
	}

	// Register built-in handlers if not overridden
	if _, exists := exec.handlers["log"]; !exists {
		exec.RegisterHandler(&LogObligationHandler{})
	}
	if _, exists := exec.handlers["notify"]; !exists {
		exec.RegisterHandler(&NotifyObligationHandler{})
	}
	if _, exists := exec.handlers["rate_limit"]; !exists {
		exec.RegisterHandler(NewRateLimitObligationHandler(0, 0, nil))
	}

	return exec
}

// ExtendedExecutorOption is a functional option for ExtendedObligationExecutor.
type ExtendedExecutorOption func(*ExtendedObligationExecutor)

// WithAdviceChannel sets the advice emission channel.
func WithAdviceChannel(ch AdviceChannel) ExtendedExecutorOption {
	return func(e *ExtendedObligationExecutor) {
		e.adviceChannel = ch
	}
}

// WithObligationAuditSink sets the persistent audit sink.
func WithObligationAuditSink(sink ObligationAuditSink) ExtendedExecutorOption {
	return func(e *ExtendedObligationExecutor) {
		e.auditSink = sink
	}
}

// RegisterHandler adds an obligation handler (thread-safe).
func (e *ExtendedObligationExecutor) RegisterHandler(handler ObligationHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[handler.Type()] = handler
}

// Execute implements obligations.Executor interface.
func (e *ExtendedObligationExecutor) Execute(ctx context.Context, names []string) []obligations.Result {
	results := make([]obligations.Result, 0, len(names))

	for _, name := range names {
		result := obligations.Result{
			Name:    name,
			Success: true,
		}

		// Parse obligation name format: "type:id" or just "id" (defaults to "log")
		obligationType, obligationID := e.parseObligationName(name)

		// Get handler
		e.mu.RLock()
		handler, exists := e.handlers[obligationType]
		e.mu.RUnlock()

		if !exists {
			result.Success = false
			result.Error = fmt.Errorf("no handler for obligation type: %s", obligationType)
			results = append(results, result)
			continue
		}

		// Execute obligation
		start := time.Now()
		err := handler.Execute(ctx, obligationID, nil) // TODO: Pass params from obligation definition
		duration := time.Since(start)

		if err != nil {
			result.Success = false
			result.Error = err
		}

		// Audit execution if sink configured
		if e.auditSink != nil {
			auditRecord := ObligationAuditRecord{
				Timestamp:      time.Now(),
				ObligationID:   obligationID,
				ObligationType: obligationType,
				Success:        result.Success,
				DurationMS:     float64(duration.Microseconds()) / 1000.0,
			}
			if err != nil {
				auditRecord.Error = err.Error()
			}
			// Fire and forget (don't block on audit failures)
			_ = e.auditSink.RecordObligationExecution(ctx, auditRecord)
		}

		results = append(results, result)
	}

	return results
}

// parseObligationName splits "type:id" format into components.
func (e *ExtendedObligationExecutor) parseObligationName(name string) (obligationType, obligationID string) {
	// Simple parsing: "log:user_action" -> ("log", "user_action")
	// If no colon, default to "log" type
	for i, r := range name {
		if r == ':' {
			return name[:i], name[i+1:]
		}
	}
	return "log", name
}

// Built-in Obligation Handlers

// LogObligationHandler logs obligation execution (built-in handler).
type LogObligationHandler struct{}

func (h *LogObligationHandler) Type() string { return "log" }

func (h *LogObligationHandler) Execute(ctx context.Context, obligationID string, params map[string]string) error {
	log.Printf("[Obligation:Log] Executed obligation: %s, params: %v", obligationID, params)
	return nil
}

// NotifyObligationHandler sends notifications (stub for external integrations).
type NotifyObligationHandler struct{}

func (h *NotifyObligationHandler) Type() string { return "notify" }

func (h *NotifyObligationHandler) Execute(ctx context.Context, obligationID string, params map[string]string) error {
	// TODO: Integrate with notification service (email, Slack, webhook)
	log.Printf("[Obligation:Notify] Notification triggered: %s, params: %v", obligationID, params)
	return nil
}

// RateLimitObligationHandler enforces rate limits with in-memory or Redis backend.
type RateLimitObligationHandler struct {
	mu      sync.RWMutex
	limits  map[string]*rateLimitEntry // in-memory fallback
	checker RateLimitChecker           // optional external checker (e.g., Redis)
	limit   int                        // default limit (configurable)
	window  time.Duration              // default window (configurable)
}

// rateLimitEntry tracks rate limit state for in-memory implementation
type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

// RateLimitChecker allows injection of external rate limit backends (e.g., Redis)
type RateLimitChecker interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// NewRateLimitObligationHandler creates a rate limit handler with configurable backend.
func NewRateLimitObligationHandler(limit int, window time.Duration, checker RateLimitChecker) *RateLimitObligationHandler {
	if limit <= 0 {
		limit = 100 // default 100 requests
	}
	if window <= 0 {
		window = time.Minute // default 1 minute window
	}
	return &RateLimitObligationHandler{
		limits:  make(map[string]*rateLimitEntry),
		checker: checker,
		limit:   limit,
		window:  window,
	}
}

func (h *RateLimitObligationHandler) Type() string { return "rate_limit" }

func (h *RateLimitObligationHandler) Execute(ctx context.Context, obligationID string, params map[string]string) error {
	// Extract key from params or use obligationID as key
	key := obligationID
	if k, ok := params["key"]; ok && k != "" {
		key = k
	}

	// Use external checker if configured (e.g., Redis)
	if h.checker != nil {
		allowed, err := h.checker.Allow(ctx, key)
		if err != nil {
			log.Printf("[Obligation:RateLimit] External checker error: %v, falling back to allow", err)
			return nil // Fail open on backend errors
		}
		if !allowed {
			return fmt.Errorf("rate limit exceeded for key: %s", key)
		}
		return nil
	}

	// In-memory fallback implementation
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, exists := h.limits[key]
	now := time.Now()

	if !exists || now.Sub(entry.windowStart) > h.window {
		// New window
		h.limits[key] = &rateLimitEntry{count: 1, windowStart: now}
		return nil
	}

	if entry.count >= h.limit {
		return fmt.Errorf("rate limit exceeded: %d requests in %s for key: %s", h.limit, h.window, key)
	}

	entry.count++
	return nil
}

// EmitAdvice sends advice event to the configured channel (non-blocking).
func (e *ExtendedObligationExecutor) EmitAdvice(ctx context.Context, event AdviceEvent) error {
	if e.adviceChannel == nil {
		return nil // Advice channel not configured, skip silently
	}
	return e.adviceChannel.Emit(ctx, event)
}
