// Package gauth - Power Administration Point (PAP) Types
// RFC-0111 Section 3.1 - P*P Architecture
package gauth

import (
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
)

// PolicyType represents the type of authorization policy
type PolicyType string

const (
	// PolicyTypePoA - Power of Attorney policy
	PolicyTypePoA PolicyType = "poa"

	// PolicyTypeAuthorizationChain - Authorization chain policy
	PolicyTypeAuthorizationChain PolicyType = "authorization_chain"

	// PolicyTypeScope - Scope restriction policy
	PolicyTypeScope PolicyType = "scope"

	// PolicyTypeRestriction - Power restriction policy
	PolicyTypeRestriction PolicyType = "restriction"

	// PolicyTypeCompliance - Compliance requirement policy
	PolicyTypeCompliance PolicyType = "compliance"
)

// PolicyStatus represents the lifecycle status of a policy
type PolicyStatus string

const (
	// PolicyStatusDraft - Policy is being created/edited
	PolicyStatusDraft PolicyStatus = "draft"

	// PolicyStatusActive - Policy is active and enforced
	PolicyStatusActive PolicyStatus = "active"

	// PolicyStatusSuspended - Policy is temporarily disabled
	PolicyStatusSuspended PolicyStatus = "suspended"

	// PolicyStatusRevoked - Policy is permanently revoked
	PolicyStatusRevoked PolicyStatus = "revoked"

	// PolicyStatusExpired - Policy has expired
	PolicyStatusExpired PolicyStatus = "expired"
)

// AuthorizationPolicy represents a policy administered by PAP
type AuthorizationPolicy struct {
	// Policy identification
	PolicyID      string       `json:"policy_id"`
	PolicyType    PolicyType   `json:"policy_type"`
	PolicyVersion int          `json:"policy_version"`
	PolicyName    string       `json:"policy_name"`
	Description   string       `json:"description,omitempty"`
	Status        PolicyStatus `json:"status"`

	// Policy owner (Owner's Authorizer)
	CreatedBy        string `json:"created_by"`
	OwnersAuthorizer string `json:"owners_authorizer"`
	ClientOwner      string `json:"client_owner"`
	OrganizationID   string `json:"organization_id,omitempty"`

	// Policy content
	PolicyRules  PolicyRules          `json:"policy_rules"`
	Scope        *PolicyScope         `json:"scope,omitempty"`
	Restrictions []PowerRestriction   `json:"restrictions,omitempty"`
	PoATemplate  *poa.PowerOfAttorney `json:"poa_template,omitempty"`

	// Lifecycle timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`

	// Versioning
	PreviousVersion *string `json:"previous_version,omitempty"`
	ChangeLog       string  `json:"change_log,omitempty"`

	// Enforcement tracking
	EnforcementCount int64      `json:"enforcement_count"`
	LastEnforcedAt   *time.Time `json:"last_enforced_at,omitempty"`
	ViolationCount   int64      `json:"violation_count"`

	// Metadata
	Tags     []string               `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyRules defines the rules within a policy
type PolicyRules struct {
	// Allowed actions
	AllowedActions []string `json:"allowed_actions,omitempty"`

	// Denied actions (takes precedence over allowed)
	DeniedActions []string `json:"denied_actions,omitempty"`

	// Resource patterns
	ResourcePatterns []string `json:"resource_patterns,omitempty"`

	// Attribute-based conditions
	Conditions []PolicyCondition `json:"conditions,omitempty"`

	// Time-based rules
	TimeRestrictions *TimeRestrictions `json:"time_restrictions,omitempty"`

	// Value limits
	ValueLimits *ValueLimits `json:"value_limits,omitempty"`
}

// PolicyScope defines the scope of policy application
type PolicyScope struct {
	// Geographic scope
	Countries []string `json:"countries,omitempty"`
	Regions   []string `json:"regions,omitempty"`

	// Sector scope
	Sectors []string `json:"sectors,omitempty"`

	// Entity scope
	Entities []string `json:"entities,omitempty"`

	// Client scope
	ClientIDs []string `json:"client_ids,omitempty"`
}

// PolicyCondition represents a condition that must be met
type PolicyCondition struct {
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"` // eq, ne, gt, lt, in, contains, matches
	Value     interface{} `json:"value"`
	Required  bool        `json:"required"`
}

// TimeRestrictions defines time-based policy restrictions
type TimeRestrictions struct {
	// Valid time windows
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`

	// Allowed time windows (e.g., business hours)
	AllowedTimeWindows []TimeWindow `json:"allowed_time_windows,omitempty"`

	// Timezone for time calculations
	Timezone string `json:"timezone,omitempty"`
}

// Note: TimeWindow and ValueLimits types are defined in advanced_claims.go and external_integrations.go

// PolicyCreateRequest represents a request to create a new policy
type PolicyCreateRequest struct {
	PolicyType       PolicyType             `json:"policy_type"`
	PolicyName       string                 `json:"policy_name"`
	Description      string                 `json:"description,omitempty"`
	ClientOwner      string                 `json:"client_owner"`
	OwnersAuthorizer string                 `json:"owners_authorizer"`
	PolicyRules      PolicyRules            `json:"policy_rules"`
	Scope            *PolicyScope           `json:"scope,omitempty"`
	Restrictions     []PowerRestriction     `json:"restrictions,omitempty"`
	PoATemplate      *poa.PowerOfAttorney   `json:"poa_template,omitempty"`
	ExpiresAt        *time.Time             `json:"expires_at,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyUpdateRequest represents a request to update an existing policy
type PolicyUpdateRequest struct {
	PolicyID     string                  `json:"policy_id"`
	PolicyName   *string                 `json:"policy_name,omitempty"`
	Description  *string                 `json:"description,omitempty"`
	PolicyRules  *PolicyRules            `json:"policy_rules,omitempty"`
	Scope        *PolicyScope            `json:"scope,omitempty"`
	Restrictions *[]PowerRestriction     `json:"restrictions,omitempty"`
	ExpiresAt    *time.Time              `json:"expires_at,omitempty"`
	Tags         *[]string               `json:"tags,omitempty"`
	Metadata     *map[string]interface{} `json:"metadata,omitempty"`
	ChangeLog    string                  `json:"change_log"`
}

// PolicySearchCriteria represents criteria for searching policies
type PolicySearchCriteria struct {
	PolicyTypes      []PolicyType   `json:"policy_types,omitempty"`
	Statuses         []PolicyStatus `json:"statuses,omitempty"`
	ClientOwner      string         `json:"client_owner,omitempty"`
	OwnersAuthorizer string         `json:"owners_authorizer,omitempty"`
	Tags             []string       `json:"tags,omitempty"`
	SearchText       string         `json:"search_text,omitempty"`
	CreatedAfter     *time.Time     `json:"created_after,omitempty"`
	CreatedBefore    *time.Time     `json:"created_before,omitempty"`
	Limit            int            `json:"limit,omitempty"`
	Offset           int            `json:"offset,omitempty"`
}

// PolicySearchResult represents the result of a policy search
type PolicySearchResult struct {
	Policies   []*AuthorizationPolicy `json:"policies"`
	TotalCount int                    `json:"total_count"`
	Limit      int                    `json:"limit"`
	Offset     int                    `json:"offset"`
}

// PolicyValidationResult represents the result of policy validation
type PolicyValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// PolicyEnforcementEvent represents a policy enforcement event
type PolicyEnforcementEvent struct {
	EventID    string    `json:"event_id"`
	PolicyID   string    `json:"policy_id"`
	ClientID   string    `json:"client_id"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	Allowed    bool      `json:"allowed"`
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Violations []string  `json:"violations,omitempty"`
}

// PolicyStatistics represents statistics for a policy
type PolicyStatistics struct {
	PolicyID               string     `json:"policy_id"`
	EnforcementCount       int64      `json:"enforcement_count"`
	AllowedCount           int64      `json:"allowed_count"`
	DeniedCount            int64      `json:"denied_count"`
	ViolationCount         int64      `json:"violation_count"`
	LastEnforcedAt         *time.Time `json:"last_enforced_at,omitempty"`
	AverageEnforcementTime float64    `json:"average_enforcement_time_ms"`
}
