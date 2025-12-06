package audit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/common"
)

// DatabaseLogger implements audit logging to the database
type DatabaseLogger struct {
	mu            sync.RWMutex
	repo          *Repository
	logger        common.Logger
	eventQueue    chan *AuditEvent
	done          chan struct{}
	wg            sync.WaitGroup
	droppedEvents int64
}

// NewDatabaseLogger creates a new database-backed audit logger
func NewDatabaseLogger(repo *Repository, logger common.Logger) *DatabaseLogger {
	if logger == nil {
		logger = common.NewSimpleLogger()
	}
	dl := &DatabaseLogger{
		repo:       repo,
		logger:     logger,
		eventQueue: make(chan *AuditEvent, 5000), // Large buffer for DB spikes
		done:       make(chan struct{}),
	}
	dl.wg.Add(1)
	go dl.processEvents()
	return dl
}

// processEvents runs in background to persist events
func (dl *DatabaseLogger) processEvents() {
	defer dl.wg.Done()

	// Batch processing parameters
	const batchSize = 100
	const flushInterval = 500 * time.Millisecond

	batch := make([]*AuditEvent, 0, batchSize)
	timer := time.NewTimer(flushInterval)
	defer timer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// Create a separate context with timeout for DB write
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := dl.repo.CreateEventsBulk(ctx, batch)
		cancel()

		if err != nil {
			dl.logger.Errorf("Failed to persist audit batch of %d events: %v", len(batch), err)
			// In a robust system, we might retry or fallback to file
		}

		// Reset batch
		// batch = batch[:0] // Reuse slice optimization? Safer to make new in case of async issues with pointers?
		// Since we passed pointers to CreateEventsBulk, and it's done, we can reuse.
		batch = make([]*AuditEvent, 0, batchSize)
	}

	for {
		select {
		case event := <-dl.eventQueue:
			batch = append(batch, event)
			if len(batch) >= batchSize {
				flush()
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(flushInterval)
			}
		case <-timer.C:
			flush()
			timer.Reset(flushInterval)
		case <-dl.done:
			// Drain queue
			for {
				select {
				case event := <-dl.eventQueue:
					batch = append(batch, event)
				default:
					flush()
					return
				}
			}
		}
	}
}

// Log logs an audit event
func (dl *DatabaseLogger) Log(ctx context.Context, entry interface{}) error {
	var event *AuditEvent

	switch e := entry.(type) {
	case *AuditEvent:
		event = e
	case *Event:
		// Convert memory Event to DB AuditEvent
		event = &AuditEvent{
			TenantID:   "default", // Default tenant
			Timestamp:  e.Timestamp,
			EventType:  string(e.Type),
			Category:   "authz",   // Infer?
			UserID:     e.Subject, // map Subject to UserID
			Action:     e.Action,
			ResourceID: e.Object, // map Object to ResourceID
			Status:     e.Result,
			// ... other mappings
			Severity: "info",
		}
		// Marshal metadata to state/changes if needed, or simple mapping
		if e.Metadata != nil {
			event.BeforeState = e.Metadata // Simply dumping metadata here for now
		}
	default:
		return nil // Unknown type
	}

	if event.ID == "" {
		event.ID = generateID()
	}

	select {
	case dl.eventQueue <- event:
		return nil
	default:
		atomic.AddInt64(&dl.droppedEvents, 1)
		dl.logger.Warnf("Audit database queue full, dropped event %s", event.ID)
		return nil
	}
}

// Close stops the processor
func (dl *DatabaseLogger) Close() error {
	close(dl.done)
	dl.wg.Wait()
	return nil
}
