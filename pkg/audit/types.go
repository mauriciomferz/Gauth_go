package audit

import "time"

// Entry represents a single audit log entry.

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
