package testutil

import (
	"context"
	"sync"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/common"
)

// AuditRecorder wraps MemoryLogger to expose recorded events for assertions
type AuditRecorder struct {
	Logger *audit.MemoryLogger
	mu     sync.RWMutex
	events []audit.Event
}

func NewAuditRecorder(base common.Logger) *AuditRecorder {
	ml := audit.NewMemoryLogger(base)
	return &AuditRecorder{Logger: ml, events: make([]audit.Event, 0)}
}

// Log overrides MemoryLogger.Log to also copy event into slice
func (ar *AuditRecorder) Log(ctx context.Context, entry interface{}) error {
	if err := ar.Logger.Log(ctx, entry); err != nil {
		return err
	}
	// After underlying logger appended, copy the last element if type matched
	if _, ok := entry.(*audit.Event); ok {
		ar.mu.Lock()
		if l := len(ar.MemoryLoggerEvents()); l > 0 { // copy only new tail
			ar.events = append(ar.events, ar.MemoryLoggerEvents()[l-1])
		}
		ar.mu.Unlock()
	}
	return nil
}

// MemoryLoggerEvents exposes a copy of underlying memory logger events (reflection avoidance)
func (ar *AuditRecorder) MemoryLoggerEvents() []audit.Event {
	res, err := ar.Logger.Query(context.Background(), nil)
	if err != nil || len(res) == 0 {
		return []audit.Event{}
	}
	out := make([]audit.Event, len(res))
	for i, ev := range res {
		out[i] = *ev
	}
	return out
}

func (ar *AuditRecorder) Events() []audit.Event {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	out := make([]audit.Event, len(ar.events))
	copy(out, ar.events)
	return out
}
