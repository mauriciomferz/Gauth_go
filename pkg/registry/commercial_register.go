// Package registry - Commercial Register Integration per AAP001 Steps II & VII
package registry

import (
	"context"
	"fmt"
	"time"
)

// CommercialRegisterService provides integration with commercial/business registers
// AAP001 Requirement (Step II & VII): Authorization server must verify authorization
// through commercial register or equivalent authoritative source
type CommercialRegisterService interface {
	// VerifyRegistration verifies an entity is registered in the commercial register
	VerifyRegistration(ctx context.Context, req *RegistrationVerificationRequest) (*RegistrationVerificationResult, error)

	// VerifyAuthorizedRepresentative verifies a person's authority to represent an entity
	VerifyAuthorizedRepresentative(
		ctx context.Context, req *RepresentativeVerificationRequest,
	) (*RepresentativeVerificationResult, error)

	// VerifyProkura verifies Prokura (German power of attorney) registration
	VerifyProkura(ctx context.Context, req *ProkuraVerificationRequest) (*ProkuraVerificationResult, error)

	// GetEntityDetails retrieves complete entity details from register
	GetEntityDetails(ctx context.Context, registrationID, jurisdiction string) (*EntityDetails, error)

	// GetAuthorizedSignatories retrieves list of authorized signatories
	GetAuthorizedSignatories(ctx context.Context, registrationID, jurisdiction string) ([]Signatory, error)
}

// RegistrationVerificationRequest contains details for registration verification
type RegistrationVerificationRequest struct {
	EntityName         string `json:"entity_name"`
	RegistrationNumber string `json:"registration_number"`
	Jurisdiction       string `json:"jurisdiction"`          // ISO 3166-1 alpha-2
	EntityType         string `json:"entity_type,omitempty"` // "AG", "GmbH", "Ltd", "Inc", etc.
	RegistrationDate   string `json:"registration_date,omitempty"`
}

// RegistrationVerificationResult contains verification results
type RegistrationVerificationResult struct {
	Verified           bool      `json:"verified"`
	RegistrationNumber string    `json:"registration_number"`
	EntityName         string    `json:"entity_name"`
	EntityType         string    `json:"entity_type"`
	Jurisdiction       string    `json:"jurisdiction"`
	RegisterName       string    `json:"register_name"` // e.g., "Handelsregister", "Companies House"
	RegistrationDate   time.Time `json:"registration_date"`
	Status             string    `json:"status"` // "active", "dissolved", "suspended"
	VerificationDate   time.Time `json:"verification_date"`
	VerificationMethod string    `json:"verification_method"`
	VerificationRef    string    `json:"verification_ref,omitempty"`
	RegisterURL        string    `json:"register_url,omitempty"`
}

// RepresentativeVerificationRequest contains details for representative verification
type RepresentativeVerificationRequest struct {
	RepresentativeName string `json:"representative_name"`
	RepresentativeID   string `json:"representative_id,omitempty"` // National ID, passport, etc.
	EntityRegistration string `json:"entity_registration"`
	Jurisdiction       string `json:"jurisdiction"`
	AuthorityType      string `json:"authority_type"` // "managing_director", "prokura", "authorized_signatory"
}

// RepresentativeVerificationResult contains representative verification results
type RepresentativeVerificationResult struct {
	Verified           bool      `json:"verified"`
	RepresentativeName string    `json:"representative_name"`
	Position           string    `json:"position"`
	AuthorityType      string    `json:"authority_type"`
	AuthorityScope     string    `json:"authority_scope"` // "unlimited", "limited", "joint"
	AppointmentDate    time.Time `json:"appointment_date"`
	ValidFrom          time.Time `json:"valid_from"`
	ValidUntil         time.Time `json:"valid_until,omitempty"`
	SignatureAuthority string    `json:"signature_authority"` // "sole", "joint", "collective"
	RegisterEntry      string    `json:"register_entry,omitempty"`
	VerificationDate   time.Time `json:"verification_date"`
	VerificationRef    string    `json:"verification_ref,omitempty"`
	Limitations        []string  `json:"limitations,omitempty"`
}

// ProkuraVerificationRequest contains details for Prokura verification
type ProkuraVerificationRequest struct {
	ProkuraHolder      string `json:"prokura_holder"`
	EntityRegistration string `json:"entity_registration"`
	Jurisdiction       string `json:"jurisdiction"`
	ProkuraType        string `json:"prokura_type"` // "einzelprokura", "gesamtprokura"
}

// ProkuraVerificationResult contains Prokura verification results
type ProkuraVerificationResult struct {
	Verified            bool      `json:"verified"`
	ProkuraHolder       string    `json:"prokura_holder"`
	ProkuraType         string    `json:"prokura_type"`
	GrantDate           time.Time `json:"grant_date"`
	RegisterEntryDate   time.Time `json:"register_entry_date"`
	Scope               string    `json:"scope"` // "all_business_transactions", "limited"
	Limitations         []string  `json:"limitations,omitempty"`
	JointRepresentation bool      `json:"joint_representation"`
	JointPartners       []string  `json:"joint_partners,omitempty"`
	Status              string    `json:"status"` // "active", "revoked"
	VerificationDate    time.Time `json:"verification_date"`
	VerificationRef     string    `json:"verification_ref,omitempty"`
}

// EntityDetails contains complete entity information from register
type EntityDetails struct {
	RegistrationNumber    string        `json:"registration_number"`
	EntityName            string        `json:"entity_name"`
	LegalForm             string        `json:"legal_form"`
	RegisteredAddress     Address       `json:"registered_address"`
	RegistrationDate      time.Time     `json:"registration_date"`
	Status                string        `json:"status"`
	Capital               *CapitalInfo  `json:"capital,omitempty"`
	ManagingDirectors     []Signatory   `json:"managing_directors"`
	AuthorizedSignatories []Signatory   `json:"authorized_signatories"`
	Shareholders          []Shareholder `json:"shareholders,omitempty"`
	BusinessPurpose       string        `json:"business_purpose,omitempty"`
	LastUpdated           time.Time     `json:"last_updated"`
}

// Address represents a physical address
type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	State      string `json:"state,omitempty"`
}

// CapitalInfo represents company capital information
type CapitalInfo struct {
	RegisteredCapital float64 `json:"registered_capital"`
	PaidUpCapital     float64 `json:"paid_up_capital"`
	Currency          string  `json:"currency"`
}

// Signatory represents an authorized signatory
type Signatory struct {
	Name               string    `json:"name"`
	Position           string    `json:"position"`
	AuthorityType      string    `json:"authority_type"`      // "managing_director", "prokura", "authorized_signatory"
	SignatureAuthority string    `json:"signature_authority"` // "sole", "joint", "collective"
	AppointmentDate    time.Time `json:"appointment_date"`
	ValidFrom          time.Time `json:"valid_from"`
	ValidUntil         time.Time `json:"valid_until,omitempty"`
	Limitations        []string  `json:"limitations,omitempty"`
}

// Shareholder represents a shareholder
type Shareholder struct {
	Name            string  `json:"name"`
	SharePercentage float64 `json:"share_percentage"`
	ShareType       string  `json:"share_type,omitempty"`
}

// MockCommercialRegisterService provides a mock implementation for testing
type MockCommercialRegisterService struct {
	registrations   map[string]*EntityDetails
	representatives map[string]*RepresentativeVerificationResult
	prokuras        map[string]*ProkuraVerificationResult
	verifyDelay     time.Duration
}

// NewMockCommercialRegisterService creates a new mock service
func NewMockCommercialRegisterService() *MockCommercialRegisterService {
	mock := &MockCommercialRegisterService{
		registrations:   make(map[string]*EntityDetails),
		representatives: make(map[string]*RepresentativeVerificationResult),
		prokuras:        make(map[string]*ProkuraVerificationResult),
		verifyDelay:     100 * time.Millisecond,
	}

	// Pre-populate with test data
	mock.seedTestData()

	return mock
}

// seedTestData populates mock with test data
func (m *MockCommercialRegisterService) seedTestData() {
	// German GmbH example
	m.registrations["HRB12345-DE"] = &EntityDetails{
		RegistrationNumber: "HRB 12345",
		EntityName:         "Test Technologies GmbH",
		LegalForm:          "GmbH",
		RegisteredAddress: Address{
			Street:     "Teststraße 1",
			City:       "München",
			PostalCode: "80331",
			Country:    "DE",
			State:      "Bayern",
		},
		RegistrationDate: time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		Status:           "active",
		Capital: &CapitalInfo{
			RegisteredCapital: 25000,
			PaidUpCapital:     25000,
			Currency:          "EUR",
		},
		ManagingDirectors: []Signatory{
			{
				Name:               "Dr. Max Mustermann",
				Position:           "Geschäftsführer",
				AuthorityType:      "managing_director",
				SignatureAuthority: "sole",
				AppointmentDate:    time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
				ValidFrom:          time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
			},
		},
		AuthorizedSignatories: []Signatory{
			{
				Name:               "Erika Musterfrau",
				Position:           "Prokuristin",
				AuthorityType:      "prokura",
				SignatureAuthority: "sole",
				AppointmentDate:    time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC),
				ValidFrom:          time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		BusinessPurpose: "Entwicklung und Vertrieb von Software",
		LastUpdated:     time.Now().Add(-24 * time.Hour),
	}

	// UK Ltd example
	m.registrations["12345678-GB"] = &EntityDetails{
		RegistrationNumber: "12345678",
		EntityName:         "Test Technologies Ltd",
		LegalForm:          "Private Limited Company",
		RegisteredAddress: Address{
			Street:     "123 Test Street",
			City:       "London",
			PostalCode: "SW1A 1AA",
			Country:    "GB",
		},
		RegistrationDate: time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC),
		Status:           "active",
		ManagingDirectors: []Signatory{
			{
				Name:               "John Smith",
				Position:           "Director",
				AuthorityType:      "managing_director",
				SignatureAuthority: "sole",
				AppointmentDate:    time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC),
				ValidFrom:          time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		BusinessPurpose: "Software development and consulting",
		LastUpdated:     time.Now().Add(-48 * time.Hour),
	}
}

// VerifyRegistration verifies entity registration
func (m *MockCommercialRegisterService) VerifyRegistration(
	ctx context.Context, req *RegistrationVerificationRequest,
) (*RegistrationVerificationResult, error) {
	// Simulate API delay
	time.Sleep(m.verifyDelay)

	key := fmt.Sprintf("%s-%s", req.RegistrationNumber, req.Jurisdiction)
	entity, exists := m.registrations[key]

	if !exists {
		return &RegistrationVerificationResult{
			Verified:           false,
			VerificationDate:   time.Now(),
			VerificationMethod: "mock_registry_api",
		}, nil
	}

	return &RegistrationVerificationResult{
		Verified:           true,
		RegistrationNumber: entity.RegistrationNumber,
		EntityName:         entity.EntityName,
		EntityType:         entity.LegalForm,
		Jurisdiction:       req.Jurisdiction,
		RegisterName:       getRegisterName(req.Jurisdiction),
		RegistrationDate:   entity.RegistrationDate,
		Status:             entity.Status,
		VerificationDate:   time.Now(),
		VerificationMethod: "mock_registry_api",
		VerificationRef:    fmt.Sprintf("VERIFY-%d", time.Now().Unix()),
		RegisterURL:        fmt.Sprintf("https://mock-register.example/%s", entity.RegistrationNumber),
	}, nil
}

// VerifyAuthorizedRepresentative verifies representative authority
func (m *MockCommercialRegisterService) VerifyAuthorizedRepresentative(
	ctx context.Context, req *RepresentativeVerificationRequest,
) (*RepresentativeVerificationResult, error) {
	time.Sleep(m.verifyDelay)

	key := fmt.Sprintf("%s-%s", req.EntityRegistration, req.Jurisdiction)
	entity, exists := m.registrations[key]

	if !exists {
		return &RepresentativeVerificationResult{
			Verified:         false,
			VerificationDate: time.Now(),
		}, nil
	}

	// Search in managing directors
	for _, director := range entity.ManagingDirectors {
		if director.Name == req.RepresentativeName || req.AuthorityType == "managing_director" {
			return &RepresentativeVerificationResult{
				Verified:           true,
				RepresentativeName: director.Name,
				Position:           director.Position,
				AuthorityType:      director.AuthorityType,
				AuthorityScope:     "unlimited",
				AppointmentDate:    director.AppointmentDate,
				ValidFrom:          director.ValidFrom,
				ValidUntil:         director.ValidUntil,
				SignatureAuthority: director.SignatureAuthority,
				RegisterEntry:      entity.RegistrationNumber,
				VerificationDate:   time.Now(),
				VerificationRef:    fmt.Sprintf("REP-VERIFY-%d", time.Now().Unix()),
			}, nil
		}
	}

	// Search in authorized signatories
	for _, signatory := range entity.AuthorizedSignatories {
		if signatory.Name == req.RepresentativeName {
			return &RepresentativeVerificationResult{
				Verified:           true,
				RepresentativeName: signatory.Name,
				Position:           signatory.Position,
				AuthorityType:      signatory.AuthorityType,
				AuthorityScope:     determineScope(signatory),
				AppointmentDate:    signatory.AppointmentDate,
				ValidFrom:          signatory.ValidFrom,
				ValidUntil:         signatory.ValidUntil,
				SignatureAuthority: signatory.SignatureAuthority,
				RegisterEntry:      entity.RegistrationNumber,
				VerificationDate:   time.Now(),
				VerificationRef:    fmt.Sprintf("REP-VERIFY-%d", time.Now().Unix()),
				Limitations:        signatory.Limitations,
			}, nil
		}
	}

	return &RepresentativeVerificationResult{
		Verified:         false,
		VerificationDate: time.Now(),
	}, nil
}

// VerifyProkura verifies Prokura registration
func (m *MockCommercialRegisterService) VerifyProkura(
	ctx context.Context, req *ProkuraVerificationRequest,
) (*ProkuraVerificationResult, error) {
	time.Sleep(m.verifyDelay)

	key := fmt.Sprintf("%s-%s", req.EntityRegistration, req.Jurisdiction)
	entity, exists := m.registrations[key]

	if !exists || req.Jurisdiction != "DE" {
		return &ProkuraVerificationResult{
			Verified:         false,
			VerificationDate: time.Now(),
		}, nil
	}

	// Search for Prokura holder
	for _, signatory := range entity.AuthorizedSignatories {
		if signatory.AuthorityType == "prokura" && signatory.Name == req.ProkuraHolder {
			return &ProkuraVerificationResult{
				Verified:            true,
				ProkuraHolder:       signatory.Name,
				ProkuraType:         determineProkuraType(signatory),
				GrantDate:           signatory.AppointmentDate,
				RegisterEntryDate:   signatory.ValidFrom,
				Scope:               "all_business_transactions",
				Limitations:         signatory.Limitations,
				JointRepresentation: signatory.SignatureAuthority == "joint",
				Status:              "active",
				VerificationDate:    time.Now(),
				VerificationRef:     fmt.Sprintf("PROKURA-VERIFY-%d", time.Now().Unix()),
			}, nil
		}
	}

	return &ProkuraVerificationResult{
		Verified:         false,
		VerificationDate: time.Now(),
	}, nil
}

// GetEntityDetails retrieves entity details
func (m *MockCommercialRegisterService) GetEntityDetails(
	ctx context.Context, registrationID, jurisdiction string,
) (*EntityDetails, error) {
	time.Sleep(m.verifyDelay)

	key := fmt.Sprintf("%s-%s", registrationID, jurisdiction)
	entity, exists := m.registrations[key]

	if !exists {
		return nil, fmt.Errorf("entity not found: %s", registrationID)
	}

	return entity, nil
}

// GetAuthorizedSignatories retrieves authorized signatories
func (m *MockCommercialRegisterService) GetAuthorizedSignatories(
	ctx context.Context, registrationID, jurisdiction string,
) ([]Signatory, error) {
	time.Sleep(m.verifyDelay)

	entity, err := m.GetEntityDetails(ctx, registrationID, jurisdiction)
	if err != nil {
		return nil, err
	}

	signatories := make([]Signatory, 0)
	signatories = append(signatories, entity.ManagingDirectors...)
	signatories = append(signatories, entity.AuthorizedSignatories...)

	return signatories, nil
}

// Helper functions

func getRegisterName(jurisdiction string) string {
	registers := map[string]string{
		"DE": "Handelsregister",
		"GB": "Companies House",
		"US": "State Business Registry",
		"FR": "Registre du Commerce et des Sociétés",
		"IT": "Registro delle Imprese",
		"ES": "Registro Mercantil",
	}

	if name, ok := registers[jurisdiction]; ok {
		return name
	}
	return "Commercial Register"
}

func determineScope(signatory Signatory) string {
	if len(signatory.Limitations) == 0 {
		return "unlimited"
	}
	return "limited"
}

func determineProkuraType(signatory Signatory) string {
	if signatory.SignatureAuthority == "sole" {
		return "einzelprokura"
	}
	return "gesamtprokura"
}

// AddTestEntity adds a test entity to the mock registry
func (m *MockCommercialRegisterService) AddTestEntity(registrationID, jurisdiction string, entity *EntityDetails) {
	key := fmt.Sprintf("%s-%s", registrationID, jurisdiction)
	m.registrations[key] = entity
}
