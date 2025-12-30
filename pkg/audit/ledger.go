package audit

import (
	"context"
	"errors"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/common"
)

// LedgerEntry is maintained for backward compatibility with earlier direct hash chain usage.
// It is derived from Event instances stored in MemoryLogger.
type LedgerEntry struct {
	Index     int       // sequential position
	Timestamp time.Time // event time (UTC)
	Actor     string    // mapped from Event.Subject
	Action    string    // Event.Action
	Resource  string    // Event.Object
	// Details is DEPRECATED. Use Event.Metadata["details"] instead. Will be removed in a future revision.
	// It is preserved temporarily to avoid breaking existing tests; new code should rely on metadata.
	Details  string
	PrevHash string // Event.PrevHash
	Hash     string // Event.Hash
}

// Ledger now wraps MemoryLogger to avoid duplicate hash chain logic.
type Ledger struct {
	logger *MemoryLogger
	sealed bool
}

// NewLedger constructs a Ledger over a fresh MemoryLogger (simple logger omitted to avoid cycles).
// NewLedger constructs a Ledger with a default simple logger.
func NewLedger() *Ledger { return NewLedgerWithLogger(&common.SimpleLogger{}) }

// NewLedgerWithLogger allows providing a custom logger (e.g., NoopLogger for benchmarks/tests).
func NewLedgerWithLogger(l common.Logger) *Ledger { return &Ledger{logger: NewMemoryLogger(l)} }

// Append records a new action as an audit.Event and returns a projected LedgerEntry.
func (l *Ledger) Append(actor, action, resource, details string) (LedgerEntry, error) {
	if l.sealed {
		return LedgerEntry{}, errors.New("ledger sealed")
	}
	ev := NewEvent(EventTypeAuthorization, action, ResultSuccess)
	ev.Subject = actor
	ev.Object = resource
	if details != "" {
		if ev.Metadata == nil {
			ev.Metadata = map[string]interface{}{}
		}
		ev.Metadata["details"] = details
	}
	if err := l.logger.Log(context.Background(), ev); err != nil {
		return LedgerEntry{}, err
	}

	// Allow async audit processing to complete
	time.Sleep(50 * time.Millisecond)

	// Project last event to LedgerEntry
	events, _ := l.logger.Query(context.Background(), nil)
	e := events[len(events)-1]
	le := LedgerEntry{
		Index:     e.ChainIndex,
		Timestamp: e.Timestamp,
		Actor:     e.Subject,
		Action:    e.Action,
		Resource:  e.Object,
		PrevHash:  e.PrevHash,
		Hash:      e.Hash,
	}
	if e.Metadata != nil {
		if d, ok := e.Metadata["details"].(string); ok {
			le.Details = d
		}
	}
	return le, nil
}

// Verify delegates to MemoryLogger.VerifyChain.
func (l *Ledger) Verify() error { return l.logger.VerifyChain() }

// Seal prevents further appends.
func (l *Ledger) Seal() { l.sealed = true }

// Snapshot returns a projection of all events as LedgerEntry slice.
func (l *Ledger) Snapshot() []LedgerEntry {
	evs, _ := l.logger.Query(context.Background(), nil)
	out := make([]LedgerEntry, len(evs))
	for i, e := range evs {
		le := LedgerEntry{
			Index:     e.ChainIndex,
			Timestamp: e.Timestamp,
			Actor:     e.Subject,
			Action:    e.Action,
			Resource:  e.Object,
			PrevHash:  e.PrevHash,
			Hash:      e.Hash,
		}
		if e.Metadata != nil {
			if d, ok := e.Metadata["details"].(string); ok {
				le.Details = d
			}
		}
		out[i] = le
	}
	return out
}

// Underlying exposes the wrapped MemoryLogger (testing / migration aid).
func (l *Ledger) Underlying() *MemoryLogger { return l.logger }
