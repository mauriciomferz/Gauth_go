package auth
import (
	"context"
	"fmt"
	"time"
)

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
	Request *LegalFrameworkRequest
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
	FromID        string
	ToID          string
	Type          string
	Level         int
	Time          time.Time
	Entity        *Entity
	Jurisdiction  string
	Power         *Power
}

type ComplianceAction struct {
	Name         string
	RequesterID  string
	ApproverID   string
	Jurisdiction string
	LegalBasis   string
}

// ...existing code...
// (Copy all remaining valid type and function definitions from the current file here)
// ...existing code...

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
