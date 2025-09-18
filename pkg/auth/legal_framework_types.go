package auth

import (
    "context"
    "fmt"
    "time"
)

type ValidationRule struct {
    Name       string
    Predicate  string
    Parameters ValidationRuleParameters
}

func (r *ValidationRule) GetStringParam(key string) (string, error) {
    if val, ok := r.Parameters.StringParams[key]; ok {
        return val, nil
    }
    return "", fmt.Errorf("parameter %s not found", key)
}

func (r *ValidationRule) GetIntParam(key string) (int, error) {
    if val, ok := r.Parameters.IntParams[key]; ok {
        return val, nil
    }
    return 0, fmt.Errorf("parameter %s not found", key)
}

func (r *ValidationRule) GetBoolParam(key string) (bool, error) {
    if val, ok := r.Parameters.BoolParams[key]; ok {
        return val, nil
    }
    return false, fmt.Errorf("parameter %s not found", key)
}

type Authority struct {
    ID        string
    Type      string
    Level     int
    Powers    []Power
    Delegator string
}

type DecisionRule struct {
    Condition Condition
    Effect    string
    Priority  int
}

type ApprovalStep struct {
    Type      string
    Approvers []string
    Threshold int
    Timeout   time.Duration
}

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

type AdminAction struct {
    Admin     string
    Action    string
    Resource  string
    Timestamp time.Time
}

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

type Certificate struct {
    ID         string
    Subject    string
    Issuer     string
    PublicKey  string
    ValidFrom  time.Time
    ValidUntil time.Time
}

type ValidationRuleParameters struct {
    StringParams map[string]string
    IntParams    map[string]int
    BoolParams   map[string]bool
}

type ValidatorParameters struct {
    StringParams map[string]string
    IntParams    map[string]int
    BoolParams   map[string]bool
}

type Validator struct {
    ID         string
    Type       string
    Parameters ValidatorParameters
}

type EnforcementRule struct {
    ID        string
    Resource  string
    Action    string
    Condition Condition
}

type Handler struct {
    ID         string
    Type       string
    Priority   int
    Callback   func(context.Context, Decision) error
    Attributes DecisionAttributes
}

type AuditLog struct {
    Entries []AuditEntry
    MaxSize int
}

func (a *AuditLog) AddEntry(entry AuditEntry) {
    a.Entries = append(a.Entries, entry)
    if a.MaxSize > 0 && len(a.Entries) > a.MaxSize {
        excess := len(a.Entries) - a.MaxSize
        a.Entries = a.Entries[excess:]
    }
}

type AuditEntry struct {
    ID        string
    Timestamp time.Time
    Actor     string
    Action    string
    Resource  string
    Decision  string
    Reason    string
}

type Credentials struct {
    Type       string
    ID         string
    Secret     string
    Expiration time.Time
}

func (c *Credentials) IsExpired() bool {
    return !c.Expiration.IsZero() && time.Now().After(c.Expiration)
}

type Power struct {
    Type       string
    Resource   string
    Actions    []string
    Conditions []Condition
}

func (p *Power) HasAction(action string) bool {
    for _, a := range p.Actions {
        if a == "*" || a == action {
            return true
        }
    }
    return false
}

type DecisionAttributes struct {
    StringAttrs map[string]string
    IntAttrs    map[string]int
    BoolAttrs   map[string]bool
}

type Decision struct {
    Effect     string
    Resource   string
    Action     string
    Subject    string
    Timestamp  time.Time
    Attributes DecisionAttributes
}

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

func (d *Decision) GetStringAttribute(key string) (string, error) {
    if val, ok := d.Attributes.StringAttrs[key]; ok {
        return val, nil
    }
    return "", fmt.Errorf("attribute %s not found", key)
}

type StandardLegalFramework struct {
    store    interface{}
    register interface{}
    verifier interface{}
}
