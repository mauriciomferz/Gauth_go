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

// KenyaIdentityConnector handles Kenyan identity verification
// Supports National ID, Huduma Namba, Passport, Driver's License
type KenyaIdentityConnector struct {
	config     *KenyaConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// KenyaConnectorConfig configuration for Kenyan identity connector
type KenyaConnectorConfig struct {
	// IPRS (Integrated Population Registration System) configuration
	IPRSURL           string `validate:"required,url"`
	IPRSAPIKey        string `validate:"required"`
	
	// NTSA (National Transport and Safety Authority) configuration
	NTSAURL           string `validate:"url"`
	NTSAAPIKey        string
	
	// Huduma Number configuration
	HudumaURL         string `validate:"url"`
	HudumaAPIKey      string
	
	// Timeouts
	RequestTimeout    time.Duration
}

// NationalIDRequest National ID validation request
type NationalIDRequest struct {
	IDNumber    string `json:"id_number" validate:"required"`
	FirstName   string `json:"first_name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"date_of_birth"`
}

// NationalIDResponse National ID validation response
type NationalIDResponse struct {
	Valid            bool       `json:"valid"`
	IDNumber         string     `json:"id_number"`
	FirstName        string     `json:"first_name"`
	MiddleName       string     `json:"middle_name,omitempty"`
	Surname          string     `json:"surname"`
	DateOfBirth      string     `json:"date_of_birth"`
	Gender           string     `json:"gender"` // Male, Female
	PlaceOfBirth     string     `json:"place_of_birth,omitempty"`
	District         string     `json:"district,omitempty"`
	Address          *KEAddress `json:"address,omitempty"`
	SerialNumber     string     `json:"serial_number,omitempty"`
	DateOfIssue      string     `json:"date_of_issue,omitempty"`
	Error            string     `json:"error,omitempty"`
}

// HudumaNambaRequest Huduma Namba validation request
type HudumaNambaRequest struct {
	HudumaNamba string `json:"huduma_namba" validate:"required"`
	FirstName   string `json:"first_name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"date_of_birth"`
}

// HudumaNambaResponse Huduma Namba validation response
type HudumaNambaResponse struct {
	Valid            bool       `json:"valid"`
	HudumaNamba      string     `json:"huduma_namba"`
	FirstName        string     `json:"first_name"`
	MiddleName       string     `json:"middle_name,omitempty"`
	Surname          string     `json:"surname"`
	DateOfBirth      string     `json:"date_of_birth"`
	Gender           string     `json:"gender"`
	NationalID       string     `json:"national_id,omitempty"` // Linked National ID
	Address          *KEAddress `json:"address,omitempty"`
	RegistrationDate string     `json:"registration_date,omitempty"`
	Error            string     `json:"error,omitempty"`
}

// DriverLicenseRequest driver's license validation request
type KEDriverLicenseRequest struct {
	LicenseNumber   string `json:"license_number" validate:"required"`
	FirstName       string `json:"first_name" validate:"required"`
	Surname         string `json:"surname" validate:"required"`
	DateOfBirth     string `json:"date_of_birth" validate:"required"`
}

// DriverLicenseResponse driver's license validation response
type KEDriverLicenseResponse struct {
	Valid           bool       `json:"valid"`
	LicenseNumber   string     `json:"license_number"`
	FirstName       string     `json:"first_name"`
	Surname         string     `json:"surname"`
	DateOfBirth     string     `json:"date_of_birth"`
	Address         *KEAddress `json:"address,omitempty"`
	IssueDate       string     `json:"issue_date"`
	ExpiryDate      string     `json:"expiry_date"`
	LicenseClass    []string   `json:"license_class"` // A, A1, B, B1, C, C1, D, D1, E, BCE
	BloodGroup      string     `json:"blood_group,omitempty"`
	Error           string     `json:"error,omitempty"`
}

// KEAddress Kenyan address structure
type KEAddress struct {
	StreetAddress string `json:"street_address"`
	Building      string `json:"building,omitempty"`
	City          string `json:"city"`
	County        string `json:"county"` // 47 counties
	PostalCode    string `json:"postal_code"` // 5 digits
	Country       string `json:"country"`
}

// PassportRequest passport validation request
type KEPassportRequest struct {
	PassportNumber  string `json:"passport_number" validate:"required"`
	Surname         string `json:"surname" validate:"required"`
	GivenNames      string `json:"given_names" validate:"required"`
	DateOfBirth     string `json:"date_of_birth" validate:"required"`
	Nationality     string `json:"nationality" validate:"required"`
}

// PassportResponse passport validation response
type KEPassportResponse struct {
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

// NewKenyaIdentityConnector creates a new Kenyan identity connector
func NewKenyaIdentityConnector(config *KenyaConnectorConfig) (*KenyaIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	
	connector := &KenyaIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}
	
	return connector, nil
}

// ValidateNationalID validates Kenyan National ID number
// Format: 7-8 digits (older cards) or 8 digits (newer cards)
func (kc *KenyaIdentityConnector) ValidateNationalID(ctx context.Context, req *NationalIDRequest) (*NationalIDResponse, error) {
	// Validate request
	if err := kc.validator.Struct(req); err != nil {
		return &NationalIDResponse{Valid: false, Error: err.Error()}, nil
	}
	
	idNumber := strings.TrimSpace(req.IDNumber)
	
	// Validate format (7-8 digits)
	if !regexp.MustCompile(`^\d{7,8}$`).MatchString(idNumber) {
		return &NationalIDResponse{
			Valid: false,
			Error: "Invalid National ID format (must be 7-8 digits)",
		}, nil
	}
	
	// In production, this would verify with IPRS database
	response := &NationalIDResponse{
		Valid:        true,
		IDNumber:     idNumber,
		FirstName:    req.FirstName,
		Surname:      req.Surname,
		DateOfBirth:  req.DateOfBirth,
		DateOfIssue:  "2020-01-15",
	}
	
	return response, nil
}

// ValidateHudumaNamba validates Kenyan Huduma Namba (unique personal identifier)
// Format: Variable length alphanumeric
func (kc *KenyaIdentityConnector) ValidateHudumaNamba(ctx context.Context, req *HudumaNambaRequest) (*HudumaNambaResponse, error) {
	// Validate request
	if err := kc.validator.Struct(req); err != nil {
		return &HudumaNambaResponse{Valid: false, Error: err.Error()}, nil
	}
	
	hudumaNamba := strings.ToUpper(strings.TrimSpace(req.HudumaNamba))
	
	// Validate format (alphanumeric, minimum 8 characters)
	if !regexp.MustCompile(`^[A-Z0-9]{8,}$`).MatchString(hudumaNamba) {
		return &HudumaNambaResponse{
			Valid: false,
			Error: "Invalid Huduma Namba format",
		}, nil
	}
	
	// In production, this would verify with Huduma Kenya database
	response := &HudumaNambaResponse{
		Valid:            true,
		HudumaNamba:      hudumaNamba,
		FirstName:        req.FirstName,
		Surname:          req.Surname,
		DateOfBirth:      req.DateOfBirth,
		RegistrationDate: "2020-01-15",
	}
	
	return response, nil
}

// VerifyDriverLicense verifies Kenyan driver's license
func (kc *KenyaIdentityConnector) VerifyDriverLicense(ctx context.Context, req *KEDriverLicenseRequest) (*KEDriverLicenseResponse, error) {
	// Validate request
	if err := kc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	licenseNumber := strings.ToUpper(strings.TrimSpace(req.LicenseNumber))
	
	// Validate license number format (variable format)
	if len(licenseNumber) < 8 {
		return &KEDriverLicenseResponse{
			Valid: false,
			Error: "Invalid license number format",
		}, nil
	}
	
	// In production, this would verify with NTSA database
	response := &KEDriverLicenseResponse{
		Valid:         true,
		LicenseNumber: licenseNumber,
		FirstName:     req.FirstName,
		Surname:       req.Surname,
		DateOfBirth:   req.DateOfBirth,
		IssueDate:     "2020-01-15",
		ExpiryDate:    "2025-01-15",
		LicenseClass:  []string{"B", "BCE"}, // Light motor vehicles
	}
	
	return response, nil
}

// VerifyPassport verifies Kenyan passport
func (kc *KenyaIdentityConnector) VerifyPassport(ctx context.Context, req *KEPassportRequest) (*KEPassportResponse, error) {
	// Validate request
	if err := kc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	passportNumber := strings.ToUpper(strings.TrimSpace(req.PassportNumber))
	
	// Validate passport number format (letter + 7 digits or similar patterns)
	if !regexp.MustCompile(`^[A-Z]\d{7}$`).MatchString(passportNumber) {
		return &KEPassportResponse{
			Valid: false,
			Error: "Invalid passport number format",
		}, nil
	}
	
	// In production, this would verify with Directorate of Immigration Services
	response := &KEPassportResponse{
		Valid:            true,
		PassportNumber:   passportNumber,
		Surname:          req.Surname,
		GivenNames:       req.GivenNames,
		DateOfBirth:      req.DateOfBirth,
		Nationality:      "KEN",
		DateOfIssue:      "2020-01-15",
		DateOfExpiry:     "2030-01-15",
		IssuingAuthority: "Kenya",
		PassportType:     "Ordinary",
	}
	
	return response, nil
}

// Helper methods

func (kc *KenyaIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (kc *KenyaIdentityConnector) GetMetrics() map[string]interface{} {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	
	return map[string]interface{}{
		"connector": "kenya_identity",
	}
}

// Close closes the connector and cleans up resources
func (kc *KenyaIdentityConnector) Close() error {
	kc.mu.Lock()
	defer kc.mu.Unlock()
	return nil
}
