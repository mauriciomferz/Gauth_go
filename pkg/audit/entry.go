package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Entry represents a single audit log entry.
type Metadata map[string]string

type Entry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Action    string    `json:"action"`
	Result    string    `json:"result"`
	ActorID   string    `json:"actor_id"`
	ChainID   string    `json:"chain_id,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Metadata  Metadata  `json:"metadata,omitempty"`
	// Fields for SQL storage compatibility
	Level         string   `json:"level,omitempty"`
	PrevHash      string   `json:"prev_hash,omitempty"`
	ActorType     string   `json:"actor_type,omitempty"`
	ActorName     string   `json:"actor_name,omitempty"`
	SessionID     string   `json:"session_id,omitempty"`
	ClientIP      string   `json:"client_ip,omitempty"`
	ClientInfo    string   `json:"client_info,omitempty"`
	TargetID      string   `json:"target_id,omitempty"`
	TargetType    string   `json:"target_type,omitempty"`
	TargetName    string   `json:"target_name,omitempty"`
	TargetChanges Metadata `json:"target_changes,omitempty"`
	Location      string   `json:"location,omitempty"`
	TraceID       string   `json:"trace_id,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// NewEntry creates a new audit Entry with the given type.
func NewEntry(typ string) *Entry {
	return &Entry{
		ID:        generateID(),
		Type:      typ,
		Timestamp: time.Now(),
		Metadata:  make(Metadata),
		Tags:      []string{},
	}
}

// WithActor sets the actor ID and type.
func (e *Entry) WithActor(id, typ string) *Entry {
	e.ActorID = id
	e.ActorType = typ
	return e
}

// WithAction sets the action.
func (e *Entry) WithAction(action string) *Entry {
	e.Action = action
	return e
}

// WithTarget sets the target ID and type.
func (e *Entry) WithTarget(id, typ string) *Entry {
	e.TargetID = id
	e.TargetType = typ
	return e
}

// WithResult sets the result.
func (e *Entry) WithResult(result string) *Entry {
	e.Result = result
	return e
}

// WithMetadata adds a key-value pair to metadata.
func (e *Entry) WithMetadata(key string, value string) *Entry {
	e.Metadata[key] = value
	return e
}

// CalculateHash computes a hash of the entry's core fields.
func (e *Entry) CalculateHash() string {
	h := sha256.New()
	h.Write([]byte(e.ID))
	h.Write([]byte(e.Type))
	h.Write([]byte(e.Action))
	h.Write([]byte(e.Result))
	h.Write([]byte(e.ActorID))
	h.Write([]byte(e.TargetID))
	h.Write([]byte(e.Timestamp.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// generateID creates a pseudo-unique ID for entries (for demo/testing).
func generateID() string {
	return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
}

// Common constants for test compatibility
const (
	TypeAuth             = "auth"
	TypeToken            = "token"
	TypeResource         = "resource"
	ActorUser            = "user"
	ActionLogin          = "login"
	ActionResourceAccess = "resource_access"
	ResultSuccess        = "success"
)
