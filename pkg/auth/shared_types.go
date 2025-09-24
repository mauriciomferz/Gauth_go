package auth

import "time"

type GrantCondition struct {
	Type       string
	Constraint string
}

type LegalFrameworkAuthorizationGrant struct {
	ID         string
	RequestID  string
	GrantorID  string
	Scope      []string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	Conditions []GrantCondition
}

type Client struct {
	ID           string
	Type         string
	OwnerID      string
	Entity       *Entity
	Capabilities []string
}

type Entity struct {
	ID              string
	Type            string
	JurisdictionID  string
	LegalStatus     string
	CapacityProofs  []CapacityProof
	FiduciaryDuties []FiduciaryDuty
}

type ResourceServer struct {
	ID     string
	Type   string
	Entity *Entity
	Scopes []string
}

type CapacityProof struct {
	Type         string
	IssuedAt     time.Time
	ExpiresAt    time.Time
	IssuerID     string
	Proof        string
	Jurisdiction string
	Entity       *Entity
}

type FiduciaryDuty struct {
	Type        string
	Description string
	Scope       []string
	Validation  []string
}

type StoreStub struct{}

func (s *StoreStub) GetTrackingRecords(ctx interface{}, approvalID string) ([]TrackingRecord, error) {
	return []TrackingRecord{}, nil
}

func NewMemoryStore() interface{}                { return &StoreStub{} }
func NewStandardVerificationSystem() interface{} { return struct{}{} }
func NewStandardCommercialRegister() interface{} { return struct{}{} }

type StandardLegalFramework struct {
	verifier interface{}
	store    interface{}
	register interface{}
}

func NewStandardLegalFramework(store, verifier, register interface{}) *StandardLegalFramework {
	return &StandardLegalFramework{store: store, verifier: verifier, register: register}
}
