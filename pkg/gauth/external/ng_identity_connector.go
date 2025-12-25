package external

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

// NigeriaIdentityConnector handles Nigerian identity verification
// Supports NIN, BVN, Passport, Driver's License, Voter's Card
type NigeriaIdentityConnector struct {
	config     *NigeriaConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// NigeriaConnectorConfig configuration for Nigerian identity connector
type NigeriaConnectorConfig struct {
	// NIMC (National Identity Management Commission) configuration
	NIMCURL    string `validate:"required,url"`
	NIMCAPIKey string `validate:"required"`

	// BVN (Bank Verification Number) configuration
	BVNURL    string `validate:"url"`
	BVNAPIKey string

	// FRSC (Federal Road Safety Corps) configuration
	FRSCURL    string `validate:"url"`
	FRSCAPIKey string

	// Timeouts
	RequestTimeout time.Duration
}

// NINRequest NIN validation request
type NINRequest struct {
	NIN         string `json:"nin" validate:"required,len=11"`
	FirstName   string `json:"first_name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"date_of_birth"`
}

// NINResponse NIN validation response
type NINResponse struct {
	Valid            bool       `json:"valid"`
	NIN              string     `json:"nin"`
	FirstName        string     `json:"first_name"`
	MiddleName       string     `json:"middle_name,omitempty"`
	Surname          string     `json:"surname"`
	DateOfBirth      string     `json:"date_of_birth"`
	Gender           string     `json:"gender"` // Male, Female
	PlaceOfBirth     string     `json:"place_of_birth,omitempty"`
	StateOfOrigin    string     `json:"state_of_origin,omitempty"` // 36 states + FCT
	LGA              string     `json:"lga,omitempty"`             // Local Government Area
	Address          *NGAddress `json:"address,omitempty"`
	Photo            string     `json:"photo,omitempty"` // Base64 encoded
	RegistrationDate string     `json:"registration_date,omitempty"`
	Error            string     `json:"error,omitempty"`
}

// BVNRequest BVN validation request
type BVNRequest struct {
	BVN         string `json:"bvn" validate:"required,len=11"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	PhoneNumber string `json:"phone_number"`
}

// BVNResponse BVN validation response
type BVNResponse struct {
	Valid            bool       `json:"valid"`
	BVN              string     `json:"bvn"`
	FirstName        string     `json:"first_name"`
	MiddleName       string     `json:"middle_name,omitempty"`
	LastName         string     `json:"last_name"`
	DateOfBirth      string     `json:"date_of_birth"`
	Gender           string     `json:"gender"`
	PhoneNumber      string     `json:"phone_number"`
	Email            string     `json:"email,omitempty"`
	Address          *NGAddress `json:"address,omitempty"`
	LGA              string     `json:"lga,omitempty"`
	StateOfOrigin    string     `json:"state_of_origin,omitempty"`
	NIN              string     `json:"nin,omitempty"` // Linked NIN
	EnrollmentBank   string     `json:"enrollment_bank,omitempty"`
	EnrollmentBranch string     `json:"enrollment_branch,omitempty"`
	RegistrationDate string     `json:"registration_date,omitempty"`
	WatchListed      bool       `json:"watch_listed"`
	Error            string     `json:"error,omitempty"`
}

// DriverLicenseRequest driver's license validation request
type NGDriverLicenseRequest struct {
	LicenseNumber string `json:"license_number" validate:"required"`
	FirstName     string `json:"first_name" validate:"required"`
	Surname       string `json:"surname" validate:"required"`
	DateOfBirth   string `json:"date_of_birth" validate:"required"`
}

// DriverLicenseResponse driver's license validation response
type NGDriverLicenseResponse struct {
	Valid         bool       `json:"valid"`
	LicenseNumber string     `json:"license_number"`
	FirstName     string     `json:"first_name"`
	Surname       string     `json:"surname"`
	DateOfBirth   string     `json:"date_of_birth"`
	Address       *NGAddress `json:"address,omitempty"`
	IssueDate     string     `json:"issue_date"`
	ExpiryDate    string     `json:"expiry_date"`
	LicenseClass  []string   `json:"license_class"` // A, B, C, D, E, F, G, H
	StateOfIssue  string     `json:"state_of_issue"`
	Error         string     `json:"error,omitempty"`
}

// NGAddress Nigerian address structure
type NGAddress struct {
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	LGA           string `json:"lga"`                   // Local Government Area
	State         string `json:"state"`                 // 36 states + FCT
	PostalCode    string `json:"postal_code,omitempty"` // 6 digits
	Country       string `json:"country"`
}

// PassportRequest passport validation request
type NGPassportRequest struct {
	PassportNumber string `json:"passport_number" validate:"required"`
	Surname        string `json:"surname" validate:"required"`
	GivenNames     string `json:"given_names" validate:"required"`
	DateOfBirth    string `json:"date_of_birth" validate:"required"`
	Nationality    string `json:"nationality" validate:"required"`
}

// PassportResponse passport validation response
type NGPassportResponse struct {
	Valid            bool   `json:"valid"`
	PassportNumber   string `json:"passport_number"`
	Surname          string `json:"surname"`
	GivenNames       string `json:"given_names"`
	DateOfBirth      string `json:"date_of_birth"`
	Gender           string `json:"gender"`
	PlaceOfBirth     string `json:"place_of_birth"`
	Nationality      string `json:"nationality"`
	DateOfIssue      string `json:"date_of_issue"`
	DateOfExpiry     string `json:"date_of_expiry"`
	IssuingAuthority string `json:"issuing_authority"`
	PassportType     string `json:"passport_type"` // Ordinary, Official, Diplomatic
	Error            string `json:"error,omitempty"`
}

// NewNigeriaIdentityConnector creates a new Nigerian identity connector
func NewNigeriaIdentityConnector(config *NigeriaConnectorConfig) (*NigeriaIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &NigeriaIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// ValidateNIN validates Nigerian National Identification Number
// Format: 11 digits
func (nc *NigeriaIdentityConnector) ValidateNIN(ctx context.Context, req *NINRequest) (*NINResponse, error) {
	// Validate request
	if err := nc.validator.Struct(req); err != nil {
		return &NINResponse{Valid: false, Error: err.Error()}, nil
	}

	nin := strings.TrimSpace(req.NIN)

	// Validate format (11 digits)
	if !regexp.MustCompile(`^\d{11}$`).MatchString(nin) {
		return &NINResponse{
			Valid: false,
			Error: "Invalid NIN format (must be 11 digits)",
		}, nil
	}

	// In production, this would verify with NIMC database
	response := &NINResponse{
		Valid:            true,
		NIN:              nin,
		FirstName:        req.FirstName,
		Surname:          req.Surname,
		DateOfBirth:      req.DateOfBirth,
		RegistrationDate: "2020-01-15",
	}

	return response, nil
}

// ValidateBVN validates Nigerian Bank Verification Number
// Format: 11 digits
func (nc *NigeriaIdentityConnector) ValidateBVN(ctx context.Context, req *BVNRequest) (*BVNResponse, error) {
	// Validate request
	if err := nc.validator.Struct(req); err != nil {
		return &BVNResponse{Valid: false, Error: err.Error()}, nil
	}

	bvn := strings.TrimSpace(req.BVN)

	// Validate format (11 digits)
	if !regexp.MustCompile(`^\d{11}$`).MatchString(bvn) {
		return &BVNResponse{
			Valid: false,
			Error: "Invalid BVN format (must be 11 digits)",
		}, nil
	}

	// In production, this would verify with CBN (Central Bank of Nigeria)
	response := &BVNResponse{
		Valid:            true,
		BVN:              bvn,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		DateOfBirth:      req.DateOfBirth,
		PhoneNumber:      req.PhoneNumber,
		WatchListed:      false,
		RegistrationDate: "2020-01-15",
	}

	return response, nil
}

// VerifyDriverLicense verifies Nigerian driver's license
func (nc *NigeriaIdentityConnector) VerifyDriverLicense(ctx context.Context, req *NGDriverLicenseRequest) (*NGDriverLicenseResponse, error) {
	// Validate request
	if err := nc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	licenseNumber := strings.ToUpper(strings.TrimSpace(req.LicenseNumber))

	// Validate license number format (3 letters + 8 digits + 2 letters)
	if !regexp.MustCompile(`^[A-Z]{3}\d{8}[A-Z]{2}$`).MatchString(licenseNumber) {
		return &NGDriverLicenseResponse{
			Valid: false,
			Error: "Invalid license number format",
		}, nil
	}

	// In production, this would verify with FRSC database
	response := &NGDriverLicenseResponse{
		Valid:         true,
		LicenseNumber: licenseNumber,
		FirstName:     req.FirstName,
		Surname:       req.Surname,
		DateOfBirth:   req.DateOfBirth,
		IssueDate:     "2020-01-15",
		ExpiryDate:    "2025-01-15",
		LicenseClass:  []string{"B", "C"}, // Private and commercial light vehicles
		StateOfIssue:  "Lagos",
	}

	return response, nil
}

// VerifyPassport verifies Nigerian passport
func (nc *NigeriaIdentityConnector) VerifyPassport(ctx context.Context, req *NGPassportRequest) (*NGPassportResponse, error) {
	// Validate request
	if err := nc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	passportNumber := strings.ToUpper(strings.TrimSpace(req.PassportNumber))

	// Validate passport number format (letter + 8 digits)
	if !regexp.MustCompile(`^[A-Z]\d{8}$`).MatchString(passportNumber) {
		return &NGPassportResponse{
			Valid: false,
			Error: "Invalid passport number format",
		}, nil
	}

	// In production, this would verify with Nigeria Immigration Service
	response := &NGPassportResponse{
		Valid:            true,
		PassportNumber:   passportNumber,
		Surname:          req.Surname,
		GivenNames:       req.GivenNames,
		DateOfBirth:      req.DateOfBirth,
		Nationality:      "NGA",
		DateOfIssue:      "2020-01-15",
		DateOfExpiry:     "2030-01-15",
		IssuingAuthority: "Nigeria",
		PassportType:     "Ordinary",
	}

	return response, nil
}

// Helper methods

func (nc *NigeriaIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (nc *NigeriaIdentityConnector) GetMetrics() map[string]interface{} {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	return map[string]interface{}{
		"connector": "nigeria_identity",
	}
}

// Close closes the connector and cleans up resources
func (nc *NigeriaIdentityConnector) Close() error {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	return nil
}
