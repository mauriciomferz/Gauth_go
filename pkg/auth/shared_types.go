package auth

import "time"

// Only keep unique types or stubs required for test compatibility

// Minimal TrackingRecord type for compatibility
type TrackingRecord struct{}

// Minimal stubs for types required by the auth package

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

type Entity struct {
	ID             string
	Type           string
	JurisdictionID string
	LegalStatus    string
	CapacityProofs []CapacityProof
	FiduciaryDuties []FiduciaryDuty
}

type Client struct {
	ID           string
	Type         string
	OwnerID      string
	Entity       *Entity
	Capabilities []string
}

type ResourceServer struct {
	ID     string
	Type   string
	Entity *Entity
	Scopes []string
}