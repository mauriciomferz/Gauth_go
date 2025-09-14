package audit

import "time"

// Entry represents a single audit log entry.
type Metadata map[string]string

// Entry represents a single audit log entry.
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

// Filter represents filtering options for searching audit entries.
type Filter struct {
	ActorIDs  []string
	Types     []string
	Actions   []string
	Results   []string
	ChainID   string
	Tags      []string
	Metadata  []MetadataFilter
	TimeRange *TimeRange
	Limit     int
	Offset    int
}

type MetadataFilter struct {
	Key      string
	Value    interface{}
	Operator string // "eq", "ne", "contains"
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

// ActionTokenGenerate is a stub for audit action constant
const ActionTokenGenerate = "token_generate"

// ActionTokenValidate is a stub for audit action constant
const ActionTokenValidate = "token_validate"

// ActionTokenRevoke is a stub for audit action constant
const ActionTokenRevoke = "token_revoke"

// Event is a stub for audit.Event compatibility
// (matches the struct used in jwt.go)
type Event struct {
	Type     string
	Action   string
	ActorID  string
	Result   string
	Metadata map[string]string
}
