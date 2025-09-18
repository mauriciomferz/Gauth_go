package auth

import "time"

// LegalFrameworkRequest is the canonical request type for legal framework operations


type Client struct {
	ID           string
	Type         string
	OwnerID      string
	Entity       *Entity
	Capabilities []string
}

type Entity struct {
	ID             string
	Type           string
	JurisdictionID string
	LegalStatus    string
	CapacityProofs []CapacityProof
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

func NewMemoryStore() interface{}                  { return &StoreStub{} }
func NewStandardVerificationSystem() interface{}  { return struct{}{} }
func NewStandardCommercialRegister() interface{}  { return struct{}{} }