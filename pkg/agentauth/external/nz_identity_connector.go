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

// NZIdentityConnector handles New Zealand identity verification
// Supports RealMe, Driver's License, Passport
type NZIdentityConnector struct {
	config     *NZConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// NZConnectorConfig configuration for New Zealand identity connector
type NZConnectorConfig struct {
	// RealMe configuration
	RealMeURL      string `validate:"required,url"`
	RealMeClientID string `validate:"required"`
	RealMeSecret   string `validate:"required"`

	// Document verification
	NZTAServiceURL string `validate:"url"` // NZ Transport Agency
	DIAServiceURL  string `validate:"url"` // Department of Internal Affairs

	// Timeouts
	RequestTimeout time.Duration
}

// RealMeAuthRequest RealMe authentication request
type RealMeAuthRequest struct {
	ServiceProvider     string   `json:"service_provider" validate:"required"`
	StrengthOfIdentity  string   `json:"strength_of_identity" validate:"required,oneof=low moderate substantial high"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
}

// RealMeAuthResponse RealMe authentication response
type RealMeAuthResponse struct {
	Success            bool              `json:"success"`
	SessionID          string            `json:"session_id"`
	StrengthOfIdentity string            `json:"strength_of_identity"`
	UserInfo           *RealMeUserInfo   `json:"user_info"`
	Attributes         map[string]string `json:"attributes"`
	Error              string            `json:"error,omitempty"`
}

// RealMeUserInfo user information from RealMe
type RealMeUserInfo struct {
	FIT                 string     `json:"fit"` // Federated Identity Token
	FirstName           string     `json:"first_name"`
	MiddleName          string     `json:"middle_name,omitempty"`
	Surname             string     `json:"surname"`
	DateOfBirth         string     `json:"date_of_birth"`
	Gender              string     `json:"gender"`
	Email               string     `json:"email"`
	EmailVerified       bool       `json:"email_verified"`
	PhoneNumber         string     `json:"phone_number"`
	Address             *NZAddress `json:"address"`
	VerifiedCredentials []string   `json:"verified_credentials"` // Documents used for identity verification
}

// NZAddress New Zealand address structure
type NZAddress struct {
	UnitType     string `json:"unit_type,omitempty"` // Flat, Unit, etc.
	UnitNumber   string `json:"unit_number,omitempty"`
	FloorNumber  string `json:"floor_number,omitempty"`
	StreetNumber string `json:"street_number"`
	StreetName   string `json:"street_name"`
	StreetType   string `json:"street_type,omitempty"` // Road, Street, Avenue, etc.
	Suburb       string `json:"suburb"`
	City         string `json:"city"`
	Postcode     string `json:"postcode"`
	Country      string `json:"country"`
}

// DriverLicenseRequest driver's license validation request
type NZDriverLicenseRequest struct {
	LicenseNumber string `json:"license_number" validate:"required"`
	Version       string `json:"version" validate:"required"` // 2-digit version number
	FirstName     string `json:"first_name" validate:"required"`
	Surname       string `json:"surname" validate:"required"`
	DateOfBirth   string `json:"date_of_birth" validate:"required"`
}

// DriverLicenseResponse driver's license validation response
type NZDriverLicenseResponse struct {
	Valid           bool       `json:"valid"`
	LicenseNumber   string     `json:"license_number"`
	Version         string     `json:"version"`
	FirstName       string     `json:"first_name"`
	MiddleName      string     `json:"middle_name,omitempty"`
	Surname         string     `json:"surname"`
	DateOfBirth     string     `json:"date_of_birth"`
	Address         *NZAddress `json:"address"`
	IssueDate       string     `json:"issue_date"`
	ExpiryDate      string     `json:"expiry_date"`
	LicenseClasses  []string   `json:"license_classes"`        // Class 1, 2, 3, 4, 5, 6
	Endorsements    []string   `json:"endorsements,omitempty"` // D (Dangerous goods), P (Passenger), etc.
	Conditions      string     `json:"conditions,omitempty"`
	PhotoCardNumber string     `json:"photo_card_number,omitempty"`
	Error           string     `json:"error,omitempty"`
}

// PassportRequest passport validation request
type NZPassportRequest struct {
	PassportNumber string `json:"passport_number" validate:"required"`
	FirstName      string `json:"first_name" validate:"required"`
	Surname        string `json:"surname" validate:"required"`
	DateOfBirth    string `json:"date_of_birth" validate:"required"`
	Nationality    string `json:"nationality" validate:"required"`
}

// PassportResponse passport validation response
type NZPassportResponse struct {
	Valid            bool   `json:"valid"`
	PassportNumber   string `json:"passport_number"`
	GivenNames       string `json:"given_names"`
	Surname          string `json:"surname"`
	DateOfBirth      string `json:"date_of_birth"`
	Gender           string `json:"gender"`
	Nationality      string `json:"nationality"`
	PlaceOfBirth     string `json:"place_of_birth"`
	DateOfIssue      string `json:"date_of_issue"`
	DateOfExpiry     string `json:"date_of_expiry"`
	IssuinagentAuthority string `json:"issuing_authority"`
	Error            string `json:"error,omitempty"`
}

// NewNZIdentityConnector creates a new New Zealand identity connector
func NewNZIdentityConnector(config *NZConnectorConfig) (*NZIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &NZIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// AuthenticateRealMe authenticates using RealMe
// Identity strengths: low, moderate, substantial, high
func (nc *NZIdentityConnector) AuthenticateRealMe(ctx context.Context, req *RealMeAuthRequest) (*RealMeAuthResponse, error) {
	// Validate request
	if err := nc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate strength of identity
	validStrengths := map[string]bool{"low": true, "moderate": true, "substantial": true, "high": true}
	if !validStrengths[req.StrengthOfIdentity] {
		return &RealMeAuthResponse{
			Success: false,
			Error:   "Invalid strength of identity",
		}, nil
	}

	// In production, this would:
	// 1. Redirect to RealMe login
	// 2. Perform SAML 2.0 authentication
	// 3. Receive verified identity at requested assurance level
	// 4. Return FIT (Federated Identity Token)

	// Mock response for demonstration
	response := &RealMeAuthResponse{
		Success:            true,
		SessionID:          fmt.Sprintf("realme_%d", time.Now().Unix()),
		StrengthOfIdentity: req.StrengthOfIdentity,
		UserInfo: &RealMeUserInfo{
			FIT:                 fmt.Sprintf("FIT%d", time.Now().Unix()),
			FirstName:           "John",
			Surname:             "Smith",
			DateOfBirth:         "1990-01-15",
			Gender:              "M",
			Email:               "john.smith@example.co.nz",
			EmailVerified:       true,
			PhoneNumber:         "+64211234567",
			VerifiedCredentials: []string{"NZ Driver License", "NZ Passport"},
		},
		Attributes: make(map[string]string),
	}

	return response, nil
}

// VerifyDriverLicense verifies New Zealand driver's license
func (nc *NZIdentityConnector) VerifyDriverLicense(ctx context.Context, req *NZDriverLicenseRequest) (*NZDriverLicenseResponse, error) {
	// Validate request
	if err := nc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	licenseNumber := strings.ToUpper(strings.TrimSpace(req.LicenseNumber))

	// Validate license number format (2 letters + 6 digits)
	if !regexp.MustCompile(`^[A-Z]{2}\d{6}$`).MatchString(licenseNumber) {
		return &NZDriverLicenseResponse{
			Valid: false,
			Error: "Invalid license number format (must be 2 letters + 6 digits)",
		}, nil
	}

	// Validate version number (2 digits)
	if !regexp.MustCompile(`^\d{2}$`).MatchString(req.Version) {
		return &NZDriverLicenseResponse{
			Valid: false,
			Error: "Invalid version number (must be 2 digits)",
		}, nil
	}

	// In production, this would:
	// 1. Verify with NZTA (New Zealand Transport Agency)
	// 2. Check license status and demerit points
	// 3. Validate license classes and endorsements

	// Mock response for demonstration
	response := &NZDriverLicenseResponse{
		Valid:          true,
		LicenseNumber:  licenseNumber,
		Version:        req.Version,
		FirstName:      req.FirstName,
		Surname:        req.Surname,
		DateOfBirth:    req.DateOfBirth,
		IssueDate:      "2020-01-15",
		ExpiryDate:     "2030-01-15",
		LicenseClasses: []string{"1"}, // Full car license
	}

	return response, nil
}

// VerifyPassport verifies New Zealand passport
func (nc *NZIdentityConnector) VerifyPassport(ctx context.Context, req *NZPassportRequest) (*NZPassportResponse, error) {
	// Validate request
	if err := nc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	passportNumber := strings.ToUpper(strings.TrimSpace(req.PassportNumber))

	// Validate passport number format (2 letters + 6 digits or 1 letter + 7 digits)
	if !regexp.MustCompile(`^([A-Z]{2}\d{6}|[A-Z]\d{7})$`).MatchString(passportNumber) {
		return &NZPassportResponse{
			Valid: false,
			Error: "Invalid passport number format",
		}, nil
	}

	// In production, this would:
	// 1. Verify with DIA (Department of Internal Affairs)
	// 2. Check passport validity and status
	// 3. Validate against border control records

	// Mock response for demonstration
	response := &NZPassportResponse{
		Valid:            true,
		PassportNumber:   passportNumber,
		GivenNames:       req.FirstName,
		Surname:          req.Surname,
		DateOfBirth:      req.DateOfBirth,
		Nationality:      "NZL",
		DateOfIssue:      "2020-01-15",
		DateOfExpiry:     "2030-01-15",
		IssuinagentAuthority: "New Zealand",
	}

	return response, nil
}

// Helper methods

func (nc *NZIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (nc *NZIdentityConnector) GetMetrics() map[string]interface{} {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	return map[string]interface{}{
		"connector": "newzealand_identity",
	}
}

// Close closes the connector and cleans up resources
func (nc *NZIdentityConnector) Close() error {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	return nil
}
