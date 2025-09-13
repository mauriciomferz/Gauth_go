// Package audit provides comprehensive audit logging functionality for security and compliance.
//
// The audit package offers:
//   - Structured audit logging with rich metadata
//   - Multiple storage backends (memory, file, database)
//   - Filtering and search capabilities
//   - Compliance-focused features (tamper detection, forward secrecy)
//   - Integration with monitoring systems
//
// Basic usage:
//
//	logger := audit.NewLogger(audit.Config{
//	    Storage: audit.NewFileStorage("/var/log/gauth/audit.log"),
//	    MaxRetention: 90 * 24 * time.Hour,
//	    BatchSize:    1000,
//	})
//
//	entry := audit.NewEntry(audit.TypeAuth).
//	    WithActor("user123", audit.ActorUser).
//	    WithAction(audit.ActionLogin).
//	    WithTarget("webapp").
//	    WithResult(audit.ResultSuccess)
//
//	logger.Log(ctx, entry)
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/util"
	"github.com/google/uuid"
)

// Type represents the category of an audit event
type Type string

const (
	TypeAuth     Type = "auth"     // Authentication events
	TypeToken    Type = "token"    // Token management events
	TypeResource Type = "resource" // Resource access events
	TypeAdmin    Type = "admin"    // Administrative actions
	TypeSystem   Type = "system"   // System-level events
)

// Action represents the specific action being audited
type Action string

const (
	// Authentication actions
	ActionLogin         Action = "login"
	ActionLogout        Action = "logout"
	ActionPasswordReset Action = "password_reset"
	ActionMFAEnable     Action = "mfa_enable"
	ActionMFADisable    Action = "mfa_disable"

	// Token actions
	ActionTokenCreate   Action = "token_create"
	ActionTokenValidate Action = "token_validate"
	ActionTokenRevoke   Action = "token_revoke"
	ActionTokenRotate   Action = "token_rotate"

	// Resource actions
	ActionResourceAccess Action = "resource_access"
	ActionResourceModify Action = "resource_modify"
	ActionResourceDelete Action = "resource_delete"

	// Administrative actions
	ActionUserCreate   Action = "user_create"
	ActionUserModify   Action = "user_modify"
	ActionUserDelete   Action = "user_delete"
	ActionPolicyModify Action = "policy_modify"
	ActionConfigChange Action = "config_change"
)

// Result represents the outcome of the audited action
type Result string

const (
	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
	ResultDenied  Result = "denied"
	ResultError   Result = "error"
)

// Level represents the severity level of the audit entry
type Level string

const (
	LevelInfo     Level = "info"
	LevelWarning  Level = "warning"
	LevelError    Level = "error"
	LevelCritical Level = "critical"
)

// ActorType represents the type of entity performing the action
type ActorType string

const (
	ActorUser    ActorType = "user"
	ActorService ActorType = "service"
	ActorSystem  ActorType = "system"
)

// Entry represents a single audit log entry
type Entry struct {
	// Core fields
	ID        string    `json:"id"`        // Unique entry identifier
	Type      Type      `json:"type"`      // Entry type
	Action    Action    `json:"action"`    // Specific action
	Result    Result    `json:"result"`    // Action result
	Level     Level     `json:"level"`     // Severity level
	Timestamp time.Time `json:"timestamp"` // When the action occurred
	ChainID   string    `json:"chain_id"`  // Groups related entries
	PrevHash  string    `json:"prev_hash"` // Hash of previous entry (tamper detection)

	// Actor information
	ActorID    string    `json:"actor_id"`    // Who performed the action
	ActorType  ActorType `json:"actor_type"`  // Type of actor
	ActorName  string    `json:"actor_name"`  // Display name of actor
	SessionID  string    `json:"session_id"`  // Associated session
	ClientIP   string    `json:"client_ip"`   // Client IP address
	ClientInfo string    `json:"client_info"` // User agent/client details

	// Target information
	TargetID      string                 `json:"target_id"`      // Affected resource
	TargetType    string                 `json:"target_type"`    // Type of resource
	TargetName    string                 `json:"target_name"`    // Display name of resource
	TargetChanges map[string]interface{} `json:"target_changes"` // What changed

	// Context
	Location string            `json:"location"` // Where the action occurred
	TraceID  string            `json:"trace_id"` // For distributed tracing
	Tags     []string          `json:"tags"`     // Searchable tags
	Metadata map[string]string `json:"metadata"` // Additional context
	Error    string            `json:"error"`    // Error details if any
}

// Storage defines the interface for audit log storage
type Storage interface {
	// Store saves an audit entry
	Store(ctx context.Context, entry *Entry) error

	// Search retrieves entries matching the filter
	Search(ctx context.Context, filter *Filter) ([]*Entry, error)

	// GetByID retrieves a specific entry
	GetByID(ctx context.Context, id string) (*Entry, error)

	// GetChain retrieves all entries in a chain
	GetChain(ctx context.Context, chainID string) ([]*Entry, error)

	// Cleanup removes entries older than retention period
	Cleanup(ctx context.Context, before time.Time) error
}

// Filter defines criteria for searching audit entries
type Filter struct {
	Types       []Type       // Filter by entry types
	Actions     []Action     // Filter by actions
	Results     []Result     // Filter by results
	Levels      []Level      // Filter by severity
	ActorIDs    []string     // Filter by actors
	ActorTypes  []ActorType  // Filter by actor types
	TargetIDs   []string     // Filter by targets
	TargetTypes []string     // Filter by target types
	TimeRange   *TimeRange   // Filter by time range
	Tags        []string     // Filter by tags
	Metadata    []MetaFilter // Filter by metadata
	ChainID     string       // Filter by chain
	Limit       int          // Maximum results
	Offset      int          // Skip first n results
}

// TimeRange alias to util.TimeRange for backward compatibility
type TimeRange = util.TimeRange

// MetaFilter defines metadata matching criteria
type MetaFilter struct {
	Key      string
	Value    string
	Operator string // eq, ne, contains, etc.
}

// Config holds configuration for the audit logger
type Config struct {
	Storage      Storage       // Storage backend
	MaxRetention time.Duration // How long to keep entries
	BatchSize    int           // Batch size for storage operations
	Async        bool          // Whether to log asynchronously
	BufferSize   int           // Size of async buffer
	ErrorHandler func(error)   // Handler for async errors
}

// Logger provides thread-safe audit logging capabilities
type Logger struct {
	storage  Storage
	config   Config
	mu       sync.RWMutex
	buffer   []*Entry
	lastHash string
	done     chan struct{}
}

// NewLogger creates a new audit logger
func NewLogger(config Config) *Logger {
	if config.BatchSize <= 0 {
		config.BatchSize = 1000
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 10000
	}

	l := &Logger{
		storage: config.Storage,
		config:  config,
		buffer:  make([]*Entry, 0, config.BatchSize),
		done:    make(chan struct{}),
	}

	if config.Async {
		go l.processBuffer()
	}

	return l
}

// NewEntry creates a new audit entry
func NewEntry(typ Type) *Entry {
	return &Entry{
		ID:        uuid.New().String(),
		Type:      typ,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// WithActor sets actor information
func (e *Entry) WithActor(id string, typ ActorType) *Entry {
	e.ActorID = id
	e.ActorType = typ
	return e
}

// WithAction sets the action
func (e *Entry) WithAction(action Action) *Entry {
	e.Action = action
	return e
}

// WithResult sets the result
func (e *Entry) WithResult(result Result) *Entry {
	e.Result = result
	return e
}

// WithTarget sets target information
func (e *Entry) WithTarget(id string, typ string) *Entry {
	e.TargetID = id
	e.TargetType = typ
	return e
}

// WithError sets error information
func (e *Entry) WithError(err error) *Entry {
	if err != nil {
		e.Error = err.Error()
		e.Result = ResultError
		e.Level = LevelError
	}
	return e
}

// WithMetadata adds metadata
func (e *Entry) WithMetadata(key, value string) *Entry {
	e.Metadata[key] = value
	return e
}

// calculateHash generates a hash of the entry for tamper detection
func (e *Entry) calculateHash() string {
	h := sha256.New()
	fmt.Fprintf(h, "%s:%s:%s:%s:%d:%s",
		e.ID, e.ActorID, e.Action, e.TargetID,
		e.Timestamp.UnixNano(), e.PrevHash)
	return hex.EncodeToString(h.Sum(nil))
}

// Log records a new audit entry
func (l *Logger) Log(ctx context.Context, entry *Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Set previous hash for tamper detection
	entry.PrevHash = l.lastHash
	hash := entry.calculateHash()
	l.lastHash = hash

	if l.config.Async {
		// Add to buffer for async processing
		l.buffer = append(l.buffer, entry)
		if len(l.buffer) >= l.config.BatchSize {
			l.flush(ctx)
		}
		return nil
	}

	return l.storage.Store(ctx, entry)
}

// processBuffer handles async processing of buffered entries
func (l *Logger) processBuffer() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			if len(l.buffer) > 0 {
				l.flush(context.Background())
			}
			l.mu.Unlock()
		case <-l.done:
			return
		}
	}
}

// flush writes buffered entries to storage
func (l *Logger) flush(ctx context.Context) {
	if len(l.buffer) == 0 {
		return
	}

	// TODO: Implement batch storage operation
	for _, entry := range l.buffer {
		if err := l.storage.Store(ctx, entry); err != nil {
			if l.config.ErrorHandler != nil {
				l.config.ErrorHandler(err)
			}
		}
	}

	l.buffer = l.buffer[:0]
}

// Search searches for audit entries
func (l *Logger) Search(ctx context.Context, filter *Filter) ([]*Entry, error) {
	return l.storage.Search(ctx, filter)
}

// GetChain retrieves all entries in a chain
func (l *Logger) GetChain(ctx context.Context, chainID string) ([]*Entry, error) {
	return l.storage.GetChain(ctx, chainID)
}

// Close stops async processing and flushes remaining entries
func (l *Logger) Close() error {
	if l.config.Async {
		close(l.done)
		l.mu.Lock()
		l.flush(context.Background())
		l.mu.Unlock()
	}
	return nil
}
