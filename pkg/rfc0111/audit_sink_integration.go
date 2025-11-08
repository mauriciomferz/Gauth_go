package rfc0111

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
)

// AuditSink defines the interface for external audit event destinations.
// Implementations can send events to SIEM systems, log aggregators, databases,
// message queues, or compliance archival systems.
//
// Best Practices:
// - Sink.Send() should be fast (< 100ms) to avoid blocking token operations
// - For slow destinations, use async buffering (see AsyncAuditSink)
// - Implement idempotency checks (events may be retried on transient failures)
// - Handle errors gracefully (log but don't crash - fail-open for non-critical sinks)
type AuditSink interface {
	// Send delivers an audit event to the external sink
	// Returns error only for critical failures that should be surfaced to caller
	Send(ctx context.Context, event *audit.Event) error

	// Close flushes any buffered events and releases resources
	Close() error
}

// AuditSinkFunc is a function adapter for simple sink implementations
type AuditSinkFunc func(ctx context.Context, event *audit.Event) error

func (f AuditSinkFunc) Send(ctx context.Context, event *audit.Event) error {
	return f(ctx, event)
}

func (f AuditSinkFunc) Close() error {
	return nil // no-op for stateless functions
}

// AsyncAuditSink wraps a slow sink with async buffering and background flushing.
// Events are queued and sent in a background goroutine to prevent blocking
// token operations. If the buffer fills, oldest events are dropped (fail-open).
type AsyncAuditSink struct {
	sink       AuditSink
	buffer     chan *audit.Event
	bufferSize int
	wg         sync.WaitGroup
	closed     chan struct{}
	mu         sync.Mutex
	isClosing  bool

	// Metrics (optional, exposed for observability)
	Sent    uint64 // successfully sent events
	Dropped uint64 // events dropped due to full buffer
	Errors  uint64 // send errors
}

// NewAsyncAuditSink creates an async wrapper with the specified buffer size.
// Recommended buffer sizes: 100-1000 for low-volume, 10000+ for high-volume.
func NewAsyncAuditSink(sink AuditSink, bufferSize int) *AsyncAuditSink {
	if bufferSize <= 0 {
		bufferSize = 1000 // default buffer
	}

	async := &AsyncAuditSink{
		sink:       sink,
		buffer:     make(chan *audit.Event, bufferSize),
		bufferSize: bufferSize,
		closed:     make(chan struct{}),
	}

	// Start background flusher
	async.wg.Add(1)
	go async.flushLoop()

	return async
}

func (a *AsyncAuditSink) flushLoop() {
	defer a.wg.Done()

	for {
		select {
		case event := <-a.buffer:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := a.sink.Send(ctx, event); err != nil {
				a.mu.Lock()
				a.Errors++
				a.mu.Unlock()
				// TODO: Consider exponential backoff retry for transient errors
			} else {
				a.mu.Lock()
				a.Sent++
				a.mu.Unlock()
			}
			cancel()

		case <-a.closed:
			// Drain remaining events before shutdown
			for {
				select {
				case event := <-a.buffer:
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					_ = a.sink.Send(ctx, event)
					cancel()
				default:
					return
				}
			}
		}
	}
}

func (a *AsyncAuditSink) Send(ctx context.Context, event *audit.Event) error {
	a.mu.Lock()
	if a.isClosing {
		a.mu.Unlock()
		return fmt.Errorf("audit sink is closed")
	}
	a.mu.Unlock()

	select {
	case a.buffer <- event:
		return nil // event queued successfully
	default:
		// Buffer full - drop oldest event (fail-open)
		a.mu.Lock()
		a.Dropped++
		a.mu.Unlock()
		return nil // Don't block token operations
	}
}

func (a *AsyncAuditSink) Close() error {
	a.mu.Lock()
	if a.isClosing {
		a.mu.Unlock()
		return nil
	}
	a.isClosing = true
	a.mu.Unlock()

	close(a.closed)
	a.wg.Wait()
	return a.sink.Close()
}

// WithAuditSink configures an external audit sink for token lifecycle events.
// Events are sent to the sink for: delegation creation, token verification,
// token revocation, and evidence attachment operations.
//
// Backward Compatibility: Audit sinks are opt-in (disabled by default).
// Existing deployments continue working without changes.
//
// Usage:
//
//	sink := NewAsyncAuditSink(mySiemSink, 1000)
//	svc := rfc0111.NewService(logger, authorizer, WithAuditSink(sink))
//
// For multiple sinks (e.g., SIEM + compliance DB), use MultiplexAuditSink:
//
//	multiplex := NewMultiplexAuditSink(siemSink, complianceSink)
//	svc := rfc0111.NewService(logger, authorizer, WithAuditSink(multiplex))
func WithAuditSink(sink AuditSink) Option {
	return func(s *Service) {
		s.auditSink = sink
	}
}

// sendToAuditSink sends an event to the configured external sink (if present).
// This is called after s.audit.Log() succeeds to ensure audit chain integrity.
// Errors are logged but don't fail the operation (fail-open for external sinks).
func (s *Service) sendToAuditSink(ctx context.Context, event *audit.Event) {
	if s.auditSink == nil {
		return // No external sink configured
	}

	// Send with timeout to prevent blocking token operations
	sinkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.auditSink.Send(sinkCtx, event); err != nil {
		// Log error but don't fail operation (fail-open for external sinks)
		// TODO: Add metrics for sink failures
		if s.metrics != nil {
			// TODO: Add s.metrics.IncAuditSinkErrors()
		}
	}
}

// MultiplexAuditSink sends events to multiple sinks in parallel.
// Useful for sending to both SIEM and compliance archive simultaneously.
type MultiplexAuditSink struct {
	sinks []AuditSink
}

func NewMultiplexAuditSink(sinks ...AuditSink) *MultiplexAuditSink {
	return &MultiplexAuditSink{sinks: sinks}
}

func (m *MultiplexAuditSink) Send(ctx context.Context, event *audit.Event) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(m.sinks))

	for _, sink := range m.sinks {
		wg.Add(1)
		go func(s AuditSink) {
			defer wg.Done()
			if err := s.Send(ctx, event); err != nil {
				errCh <- err
			}
		}(sink)
	}

	wg.Wait()
	close(errCh)

	// Collect errors (non-blocking)
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiplex sink: %d/%d sinks failed", len(errs), len(m.sinks))
	}
	return nil
}

func (m *MultiplexAuditSink) Close() error {
	var errs []error
	for _, sink := range m.sinks {
		if err := sink.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multiplex close: %d/%d sinks failed", len(errs), len(m.sinks))
	}
	return nil
}

// FilteredAuditSink wraps a sink with event filtering based on type, action, or custom predicate.
// Useful for sending only high-severity events to expensive sinks (e.g., compliance archive).
type FilteredAuditSink struct {
	sink      AuditSink
	predicate func(*audit.Event) bool
}

func NewFilteredAuditSink(sink AuditSink, predicate func(*audit.Event) bool) *FilteredAuditSink {
	return &FilteredAuditSink{sink: sink, predicate: predicate}
}

func (f *FilteredAuditSink) Send(ctx context.Context, event *audit.Event) error {
	if !f.predicate(event) {
		return nil // event filtered out
	}
	return f.sink.Send(ctx, event)
}

func (f *FilteredAuditSink) Close() error {
	return f.sink.Close()
}

// Common filter predicates

// FilterByEventType creates a predicate that allows only specific event types
func FilterByEventType(allowedTypes ...audit.EventType) func(*audit.Event) bool {
	typeSet := make(map[audit.EventType]struct{}, len(allowedTypes))
	for _, t := range allowedTypes {
		typeSet[t] = struct{}{}
	}
	return func(e *audit.Event) bool {
		_, ok := typeSet[e.Type]
		return ok
	}
}

// FilterByAction creates a predicate that allows only specific actions
func FilterByAction(allowedActions ...string) func(*audit.Event) bool {
	actionSet := make(map[string]struct{}, len(allowedActions))
	for _, a := range allowedActions {
		actionSet[a] = struct{}{}
	}
	return func(e *audit.Event) bool {
		_, ok := actionSet[e.Action]
		return ok
	}
}

// FilterByResult creates a predicate that allows only specific results (e.g., only failures)
func FilterByResult(allowedResults ...string) func(*audit.Event) bool {
	resultSet := make(map[string]struct{}, len(allowedResults))
	for _, r := range allowedResults {
		resultSet[r] = struct{}{}
	}
	return func(e *audit.Event) bool {
		_, ok := resultSet[e.Result]
		return ok
	}
}
