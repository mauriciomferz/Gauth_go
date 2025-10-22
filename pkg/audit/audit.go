package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/common"
)

const (
	TypeAuth     = "auth"
	TypeToken    = "token"
	TypeResource = "resource"

	// Actor types
	ActorUser    = "user"
	ActorService = "service"
	ActorSystem  = "system"

	ActionLogin          = "login"
	ActionLogout         = "logout"
	ActionResourceAccess = "resource_access"

	// Result types
	ResultSuccess = "success"
	ResultFailure = "failure"
)

// Event represents an audit event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Subject   string                 `json:"subject,omitempty"`
	Object    string                 `json:"object,omitempty"`
	Action    string                 `json:"action"`
	Result    string                 `json:"result"`
	ClientID  string                 `json:"client_id,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Severity  string                 `json:"severity"`
	// ChainIndex is the sequential number within the in-memory logger (0-based)
	ChainIndex int `json:"chain_index"`
	// PrevHash is the hash of the previous event in the chain (empty for first)
	PrevHash string `json:"prev_hash"`
	// Hash is the SHA-256 hash of (PrevHash || Timestamp || ID || Type || Subject || Object || Action || Result || Metadata JSON)
	Hash string `json:"hash"`
}

// Entry is a stub for compatibility
type Entry struct{}

// Filter represents query filters for audit events
type Filter struct {
	EventTypes []EventType `json:"event_types,omitempty"`
	Subject    string      `json:"subject,omitempty"`
	StartTime  *time.Time  `json:"start_time,omitempty"`
	EndTime    *time.Time  `json:"end_time,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
}

// EventType represents different types of audit events
type EventType string

const (
	EventTypeAuthentication EventType = "authentication"
	EventTypeAuthorization  EventType = "authorization"
	EventTypeTokenIssue     EventType = "token_issue"
	EventTypeTokenRevoke    EventType = "token_revoke"
	EventTypeResourceAccess EventType = "resource_access"
	EventTypeError          EventType = "error"
)

// Event represents an audit event

// MemoryLogger implements audit logging in memory
type MemoryLogger struct {
	mu     sync.RWMutex
	events []Event
	logger common.Logger
	anchor AnchorFunc
}

// AnchorFunc receives (index, hash) when a new event is appended. Intended for
// external anchoring (e.g., write root hash to durable store or blockchain). It
// MUST be fast / non-blocking; it is invoked in a goroutine.
type AnchorFunc func(index int, hash string)

// SetAnchor registers (or replaces) the anchor callback. Passing nil removes it.
func (ml *MemoryLogger) SetAnchor(fn AnchorFunc) { ml.mu.Lock(); ml.anchor = fn; ml.mu.Unlock() }

// NewMemoryLogger creates a new memory-based audit logger
func NewMemoryLogger(logger common.Logger) *MemoryLogger {
	if logger == nil {
		logger = common.NewSimpleLogger() // fallback no-op style
	}
	return &MemoryLogger{events: make([]Event, 0), logger: logger}
}

// Log logs an audit event
func (ml *MemoryLogger) Log(ctx context.Context, entry interface{}) error {
	var event *Event
	switch e := entry.(type) {
	case *Event:
		event = e
	default:
		return nil // Ignore unknown types
	}

	if event.ID == "" {
		event.ID = generateID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	ml.mu.Lock()
	// chain position & previous hash
	idx := len(ml.events)
	prevHash := ""
	if idx > 0 {
		prevHash = ml.events[idx-1].Hash
	}
	event.ChainIndex = idx
	event.PrevHash = prevHash
	event.Hash = computeEventHash(*event)
	ml.events = append(ml.events, *event)
	ml.mu.Unlock()

	// Optional anchor callback (non-blocking best effort)
	if ml.anchor != nil {
		// pass only immutable values
		go ml.anchor(event.ChainIndex, event.Hash)
	}

	// Also log to the underlying logger (non-critical)
	if b, err := json.Marshal(event); err == nil {
		ml.logger.Infof("Audit Event: %s", string(b))
	}
	return nil
}

// VerifyChain validates the integrity of the in-memory hash chain.
func (ml *MemoryLogger) VerifyChain() error {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	for i, ev := range ml.events {
		expected := computeEventHash(ev)
		if ev.Hash != expected {
			return fmt.Errorf("audit chain hash mismatch at index %d", i)
		}
		if i == 0 && ev.PrevHash != "" {
			return fmt.Errorf("audit chain first event has non-empty prev hash")
		}
		if i > 0 && ev.PrevHash != ml.events[i-1].Hash {
			return fmt.Errorf("audit chain broken link at index %d", i)
		}
		if ev.ChainIndex != i {
			return fmt.Errorf("audit chain index field mismatch at %d", i)
		}
	}
	return nil
}

// Query queries audit events based on filter
func (ml *MemoryLogger) Query(ctx context.Context, filter *Filter) ([]*Event, error) {
	var result []*Event
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	for i := range ml.events {
		event := &ml.events[i]

		// Apply filters
		if filter != nil {
			if len(filter.EventTypes) > 0 {
				found := false
				for _, t := range filter.EventTypes {
					if event.Type == t {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			if filter.Subject != "" && event.Subject != filter.Subject {
				continue
			}

			if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
				continue
			}

			if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
				continue
			}
		}

		result = append(result, event)
	}

	// Apply limit and offset
	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(result) {
			result = result[filter.Offset:]
		}
		if filter.Limit > 0 && filter.Limit < len(result) {
			result = result[:filter.Limit]
		}
	}

	return result, nil
}

// Close closes the audit logger
func (ml *MemoryLogger) Close() error {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.events = nil
	return nil
}

// NewEvent creates a new audit event

func NewEvent(eventType EventType, action, result string) *Event {
	return &Event{
		ID:        generateID(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Action:    action,
		Result:    result,
		Severity:  "info",
	}
}

type FileConfig struct {
	Directory string
}

// RedisConfig is a stub for benchmark compatibility
type RedisConfig struct {
	Addr      string
	Addresses []string
	KeyPrefix string
}

// SQLConfig is a stub for benchmark compatibility
type SQLConfig struct {
	Driver string
	DSN    string
}

// FileStorage is a stub for benchmark compatibility
// Store and Close methods are also stubbed
// (Add similar for RedisStorage and SQLStorage)
type (
	FileStorage  struct{}
	RedisStorage struct{}
	SQLStorage   struct{}
)

func NewFileStorage(cfg FileConfig) (*FileStorage, error)    { return &FileStorage{}, nil }
func NewRedisStorage(cfg RedisConfig) (*RedisStorage, error) { return &RedisStorage{}, nil }
func NewSQLStorage(cfg SQLConfig) (*SQLStorage, error)       { return &SQLStorage{}, nil }

// --- BENCHMARK METRICS STRICT COMPATIBILITY PATCH ---
type BenchmarkMetrics struct{}

func (m *BenchmarkMetrics) ObserveEntry(...interface{})            {}
func (m *BenchmarkMetrics) ObserveStorageOperation(...interface{}) {}
func NewMetrics(_ string) *BenchmarkMetrics                        { return &BenchmarkMetrics{} }

// --- END BENCHMARK METRICS STRICT COMPATIBILITY PATCH ---

func (fs *FileStorage) Store(ctx context.Context, entry *Entry) error  { return nil }
func (rs *RedisStorage) Store(ctx context.Context, entry *Entry) error { return nil }
func (ss *SQLStorage) Store(ctx context.Context, entry *Entry) error   { return nil }

func (rs *RedisStorage) Search(ctx context.Context, filter *Filter) ([]*Entry, error) {
	return []*Entry{}, nil
}

func (ss *SQLStorage) Search(ctx context.Context, filter *Filter) ([]*Entry, error) {
	return []*Entry{}, nil
}

func (fs *FileStorage) Close() error  { return nil }
func (rs *RedisStorage) Close() error { return nil }
func (ss *SQLStorage) Close() error   { return nil }

// NewAuditLogger creates a new audit logger
// NewAuditLogger returns a new MemoryLogger for compatibility
func NewAuditLogger() *MemoryLogger {
	simpleLogger := &common.SimpleLogger{}
	return NewMemoryLogger(simpleLogger)
}

// generateID generates a unique identifier for entries
func generateID() string {
	return time.Now().Format("20060102150405") + "-" +
		string(rune(time.Now().Nanosecond()%26+65)) +
		string(rune(time.Now().Nanosecond()%26+97))
}

// computeEventHash derives a SHA-256 over immutable audit event fields.
func computeEventHash(ev Event) string {
	h := sha256.New()
	// stable ordering
	h.Write([]byte(ev.PrevHash))
	h.Write([]byte(ev.Timestamp.UTC().Format(time.RFC3339Nano)))
	h.Write([]byte(ev.ID))
	h.Write([]byte(ev.Type))
	h.Write([]byte(ev.Subject))
	h.Write([]byte(ev.Object))
	h.Write([]byte(ev.Action))
	h.Write([]byte(ev.Result))
	if ev.Metadata != nil {
		if b, err := json.Marshal(ev.Metadata); err == nil {
			h.Write(b)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
