package auth

import (
	"context"
	"fmt"
	"time"
)

// LegalFrameworkRequest is the canonical request type for legal framework operations
// (moved here to ensure visibility to all code in the package)
type LegalFrameworkRequest struct {
	ID              string
	ClientID        string
	Action          string
	Resource        string
	Scope           []string
	Timestamp       time.Time
	Jurisdiction    string
	Metadata        map[string]interface{}
	ResourceServer  *ResourceServer
	PowerOfAttorney *PowerOfAttorney
	Entity          *Entity
}

// --- Shared types for main/test compatibility ---

// Duplicate definitions removed

// ... existing code continues ...

type ApprovalEvent struct {
	ID              string
	TransactionID   string
	Time            time.Time
	ApprovalID      string
	RequesterID     string
	ApproverID      string
	Action          string
	JurisdictionID  string
	LegalBasis      string
	FiduciaryChecks []FiduciaryDuty
	FiduciaryDuties []FiduciaryDuty
	Evidence        interface{}
}

type Approval = ApprovalEvent

type Store interface {
	GetTrackingRecords(ctx interface{}, approvalID string) ([]TrackingRecord, error)
}

type ClientAuthorization struct {
	Client    *Client
	Server    *ResourceServer
	Timestamp time.Time
	Scope     []string
}

type Token struct {
	ID        string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Value     string
	Scopes    []string
	Audience  string
	Issuer    string
	Type      string
}

type ServerAuthorization struct {
	Token   *Token
	Request interface{}
}

type TrackingRecord struct{}

type Transaction struct {
	ID        string
	GrantID   string
	Type      string
	Status    string
	Timestamp time.Time
	Details   map[string]interface{}
}

type DelegationLink struct {
	FromID       string
	ToID         string
	Type         string
	Level        int
	Time         time.Time
	Entity       *Entity
	Jurisdiction string
	Power        *PowerOfAttorney
}

type ComplianceAction struct {
	Name         string
	RequesterID  string
	ApproverID   string
	Jurisdiction string
	LegalBasis   string
	Checks       []string
	Evidence     interface{}
}

func (f *StandardLegalFramework) Store() interface{} {
	return f.store
}

// --- Shared types for main/test compatibility ---

// --- Shared types for main/test compatibility ---

// --- Shared types moved from stubs for main/test compatibility ---

// LegalFrameworkTypes.go
// Core type definitions for RFC111 compliance

// Condition represents a condition that must be evaluated to determine access

// ConditionParameters defines the allowed parameters for a Condition.
type ConditionParameters struct {
	StringParams map[string]string
	IntParams    map[string]int
	BoolParams   map[string]bool
	// Add more typed fields as needed for your use cases
}

type Condition struct {
	Type       string
	Rule       string
	Parameters ConditionParameters
}

// NewCondition creates a new Condition with typed parameters

func NewCondition(condType, rule string, parameters ConditionParameters) *Condition {
	return &Condition{
		Type:       condType,
		Rule:       rule,
		Parameters: parameters,
	}
}

// GetStringParam retrieves a string parameter with error handling

func (c *Condition) GetStringParam(key string) (string, error) {
	if val, ok := c.Parameters.StringParams[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("parameter %s not found", key)
}

// GetIntParam retrieves an int parameter with error handling

func (c *Condition) GetIntParam(key string) (int, error) {
	if val, ok := c.Parameters.IntParams[key]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("parameter %s not found", key)
}

// GetBoolParam retrieves a bool parameter with error handling

func (c *Condition) GetBoolParam(key string) (bool, error) {
	if val, ok := c.Parameters.BoolParams[key]; ok {
		return val, nil
	}
	return false, fmt.Errorf("parameter %s not found", key)
}

// Constraint represents a restriction on a value
type Constraint struct {
	Type       string
	Limit      interface{}
	Validation string
}

// NewConstraint creates a new Constraint
func NewConstraint(constraintType string, limit interface{}, validation string) *Constraint {
	return &Constraint{
		Type:       constraintType,
		Limit:      limit,
		Validation: validation,
	}
}

// GetIntLimit returns the limit as an integer
func (c *Constraint) GetIntLimit() (int, error) {
	if intVal, ok := c.Limit.(int); ok {
		return intVal, nil
	}
	if floatVal, ok := c.Limit.(float64); ok {
		return int(floatVal), nil
	}
	return 0, fmt.Errorf("limit is not an integer")
}

// GetStringLimit returns the limit as a string
func (c *Constraint) GetStringLimit() (string, error) {
	if strVal, ok := c.Limit.(string); ok {
		return strVal, nil
	}
	return "", fmt.Errorf("limit is not a string")
}

// ValueLimit represents a monetary limit with currency and time period
type ValueLimit struct {
	Currency string
	Amount   float64
	Period   string
}

// NewValueLimit creates a new ValueLimit
func NewValueLimit(currency string, amount float64, period string) *ValueLimit {
	return &ValueLimit{
		Currency: currency,
		Amount:   amount,
		Period:   period,
	}
}

// DataSource represents a source of data for authorization decisions
type DataSource struct {
	ID          string
	Type        string
	Endpoint    string
	Credentials *Credentials
}

// PowerInformationPoint represents a component that provides data for authorization decisions
type PowerInformationPoint struct {
	DataSources []DataSource
	Cache       *DataCache
	Updates     chan DataUpdate
}

// DataCache represents a cache for authorization data
type DataCache struct {
	Entries map[string]DataCacheEntry
	TTL     time.Duration
}

// DataCacheEntry represents an entry in the data cache
type DataCacheEntry struct {
	Key       string
	Value     interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// GetValueString retrieves the value as a string with error handling
func (e *DataCacheEntry) GetValueString() (string, error) {
	if strVal, ok := e.Value.(string); ok {
		return strVal, nil
	}
	return "", fmt.Errorf("value is not a string")
}

// GetValueInt retrieves the value as an int with error handling
func (e *DataCacheEntry) GetValueInt() (int, error) {
	if intVal, ok := e.Value.(int); ok {
		return intVal, nil
	}
	if floatVal, ok := e.Value.(float64); ok {
		return int(floatVal), nil
	}
	return 0, fmt.Errorf("value is not an integer")
}

// GetValueBool retrieves the value as a bool with error handling
func (e *DataCacheEntry) GetValueBool() (bool, error) {
	if boolVal, ok := e.Value.(bool); ok {
		return boolVal, nil
	}
	return false, fmt.Errorf("value is not a boolean")
}

// IsExpired checks if the entry has expired
func (e *DataCacheEntry) IsExpired() bool {
	return time.Since(e.Timestamp) > e.TTL
}

// DataUpdate represents an update to authorization data
type DataUpdate struct {
	Source string
	Key    string
	Value  interface{}
}

// PowerAdministrationPoint represents a component that administers policies
type PowerAdministrationPoint struct {
	Policies       []Policy
	Administrators []string
	AuditLog       []AdminAction
}

// PowerVerificationPoint represents a component that verifies credentials
type PowerVerificationPoint struct {
	TrustAnchors []TrustAnchor
	CertStore    *CertificateStore
	Validators   []Validator
}

// EnforcementPoint represents a component that enforces access control decisions
type EnforcementPoint struct {
	Rules    []EnforcementRule
	Handlers []Handler
	Audit    *AuditLog
}

// ValidationRuleParameters defines allowed parameters for a ValidationRule.
type ValidationRuleParameters struct {
	StringParams map[string]string
	IntParams    map[string]int
	BoolParams   map[string]bool
	// Add more typed fields as needed
}

type ValidationRule struct {
	Name       string
	Predicate  string
	Parameters ValidationRuleParameters
}

// GetStringParam retrieves a string parameter with error handling
func (r *ValidationRule) GetStringParam(key string) (string, error) {
	if val, ok := r.Parameters.StringParams[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("parameter %s not found", key)
}

// GetIntParam retrieves an int parameter with error handling
func (r *ValidationRule) GetIntParam(key string) (int, error) {
	if val, ok := r.Parameters.IntParams[key]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("parameter %s not found", key)
}

// GetBoolParam retrieves a bool parameter with error handling
func (r *ValidationRule) GetBoolParam(key string) (bool, error) {
	if val, ok := r.Parameters.BoolParams[key]; ok {
		return val, nil
	}
	return false, fmt.Errorf("parameter %s not found", key)
}

// Authority represents an entity with authority to make decisions
type Authority struct {
	ID        string
	Type      string
	Level     int
	Powers    []Power
	Delegator string
}

// DecisionRule represents a rule for making access control decisions
type DecisionRule struct {
	Condition Condition
	Effect    string
	Priority  int
}

// ApprovalStep represents a step in an approval process
type ApprovalStep struct {
	Type      string
	Approvers []string
	Threshold int
	Timeout   time.Duration
}

// Policy represents an access control policy
type Policy struct {
	ID         string
	Name       string
	Parameters ValidationRuleParameters
	Rules      []DecisionRule
	Version    string
	Created    time.Time
}

func NewValidationRule(name, predicate string, parameters ValidationRuleParameters) *ValidationRule {
	return &ValidationRule{
		Name:       name,
		Predicate:  predicate,
		Parameters: parameters,
	}
}

// AdminAction represents an administrative action
type AdminAction struct {
	Admin     string
	Action    string
	Resource  string
	Timestamp time.Time
}

// TrustAnchor represents a trust anchor for credential verification
type TrustAnchor struct {
	ID         string
	Type       string
	PublicKey  string
	Issuer     string
	ValidFrom  time.Time
	ValidUntil time.Time
}
type CertificateStore struct {
	Certificates   map[string]Certificate
	RevocationList []string
}

// Certificate represents a certificate for authentication
type Certificate struct {
	ID         string
	Subject    string
	Issuer     string
	PublicKey  string
	ValidFrom  time.Time
	ValidUntil time.Time
}

// ValidatorParameters defines allowed parameters for a Validator.
type ValidatorParameters struct {
	StringParams map[string]string
	IntParams    map[string]int
	BoolParams   map[string]bool
	// Add more typed fields as needed
}

// Validator represents a validator for credentials
type Validator struct {
	ID         string
	Type       string
	Parameters ValidatorParameters
}

// EnforcementRule represents a rule for enforcing access control
type EnforcementRule struct {
	ID        string
	Resource  string
	Action    string
	Condition Condition
}

// Handler represents a handler for access control decisions
type Handler struct {
	ID         string
	Type       string
	Priority   int
	Callback   func(context.Context, Decision) error
	Attributes DecisionAttributes
}

// AuditLog represents an audit log
type AuditLog struct {
	Entries []AuditEntry
	MaxSize int
}

// AddEntry adds an entry to the audit log
func (a *AuditLog) AddEntry(entry AuditEntry) {
	a.Entries = append(a.Entries, entry)
	// Maintain max size
	if a.MaxSize > 0 && len(a.Entries) > a.MaxSize {
		// Remove oldest entries
		excess := len(a.Entries) - a.MaxSize
		a.Entries = a.Entries[excess:]
	}
}

// AuditEntry represents an entry in the audit log
type AuditEntry struct {
	ID        string
	Timestamp time.Time
	Actor     string
	Action    string
	Resource  string
	Decision  string
	Reason    string
}

// Credentials represents authentication credentials
type Credentials struct {
	Type       string
	ID         string
	Secret     string
	Expiration time.Time
}

// IsExpired checks if the credentials have expired
func (c *Credentials) IsExpired() bool {
	return !c.Expiration.IsZero() && time.Now().After(c.Expiration)
}

// Power represents a capability granted to an authority
type Power struct {
	Type       string
	Resource   string
	Actions    []string
	Conditions []Condition
}

// HasAction checks if the power grants a specific action
func (p *Power) HasAction(action string) bool {
	for _, a := range p.Actions {
		if a == "*" || a == action {
			return true
		}
	}
	return false
}

// Decision represents an access control decision

// DecisionAttributes defines the allowed attributes for a Decision.
type DecisionAttributes struct {
	StringAttrs map[string]string
	IntAttrs    map[string]int
	BoolAttrs   map[string]bool
	// Add more typed fields as needed for your use cases
}

type Decision struct {
	Effect     string
	Resource   string
	Action     string
	Subject    string
	Timestamp  time.Time
	Attributes DecisionAttributes
}

// NewDecision creates a new Decision
func NewDecision(effect, resource, action, subject string, attributes DecisionAttributes) *Decision {
	return &Decision{
		Effect:     effect,
		Resource:   resource,
		Action:     action,
		Subject:    subject,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}
}

// GetStringAttribute retrieves a string attribute with error handling
func (d *Decision) GetStringAttribute(key string) (string, error) {
	if val, ok := d.Attributes.StringAttrs[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("attribute %s not found", key)
}

// --- Stubbed methods for StandardLegalFramework ---
func (f *StandardLegalFramework) TrackApprovalDetails(ctx interface{}, event *ApprovalEvent) error {
	return nil
}
func (f *StandardLegalFramework) VerifyLegalCapacity(ctx interface{}, entity interface{}) error {
	return nil
}
func (f *StandardLegalFramework) ValidateClientResourceServerInteraction(ctx interface{}, client interface{}, server interface{}) error {
	return nil
}
func (f *StandardLegalFramework) ValidateResourceServerPowers(ctx interface{}, token interface{}, request interface{}) error {
	return nil
}
func (f *StandardLegalFramework) ValidateJurisdiction(ctx interface{}, jurisdiction interface{}, action interface{}) error {
	return nil
}
func (f *StandardLegalFramework) GetJurisdictionRules(jurisdiction string) (*JurisdictionRules, error) {
	return &JurisdictionRules{}, nil
}
func (f *StandardLegalFramework) ValidateJurisdictionRequirements(ctx interface{}, rules *JurisdictionRules, action string) error {
	return nil
}
func (f *StandardLegalFramework) ValidateDuty(ctx interface{}, duty interface{}) error { return nil }
func (f *StandardLegalFramework) EnforceFiduciaryDuties(ctx interface{}, power interface{}) error {
	return nil
}
