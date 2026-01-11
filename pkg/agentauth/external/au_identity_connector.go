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

// AustraliaIdentityConnector handles Australian identity verification
// Supports myGovID, Medicare, Driver's License, Passport
type AustraliaIdentityConnector struct {
	config     *AustraliaConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// AustraliaConnectorConfig configuration for Australian identity connector
type AustraliaConnectorConfig struct {
	// myGovID configuration
	MyGovIDURL      string `validate:"required,url"`
	MyGovIDClientID string `validate:"required"`
	MyGovIDSecret   string `validate:"required"`

	// Document Verification Service (DVS) configuration
	DVSURL        string `validate:"url"`
	DVSAccessCode string

	// Medicare verification
	MedicareURL    string `validate:"url"`
	MedicareAPIKey string

	// Timeouts
	RequestTimeout time.Duration
}

// MyGovIDAuthRequest myGovID authentication request
type MyGovIDAuthRequest struct {
	IdentityStrength    string   `json:"identity_strength" validate:"required,oneof=IP1 IP2 IP3"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
}

// MyGovIDAuthResponse myGovID authentication response
type MyGovIDAuthResponse struct {
	Success          bool              `json:"success"`
	SessionID        string            `json:"session_id"`
	IdentityStrength string            `json:"identity_strength"`
	UserInfo         *MyGovIDUserInfo  `json:"user_info"`
	Attributes       map[string]string `json:"attributes"`
	Error            string            `json:"error,omitempty"`
}

// MyGovIDUserInfo user information from myGovID
type MyGovIDUserInfo struct {
	MyGovID       string     `json:"mygov_id"`
	GivenName     string     `json:"given_name"`
	FamilyName    string     `json:"family_name"`
	MiddleName    string     `json:"middle_name,omitempty"`
	DateOfBirth   string     `json:"date_of_birth"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	PhoneNumber   string     `json:"phone_number"`
	Address       *AUAddress `json:"address"`
}

// AUAddress Australian address structure
type AUAddress struct {
	StreetAddress string `json:"street_address"`
	Suburb        string `json:"suburb"`
	State         string `json:"state"` // NSW, VIC, QLD, SA, WA, TAS, NT, ACT
	Postcode      string `json:"postcode"`
	Country       string `json:"country"`
}

// MedicareCardRequest Medicare card validation request
type MedicareCardRequest struct {
	CardNumber  string `json:"card_number" validate:"required,len=10"`
	IRN         string `json:"irn" validate:"required,len=1"` // Individual Reference Number (1-9)
	CardColor   string `json:"card_color" validate:"omitempty,oneof=green blue yellow"`
	FamilyName  string `json:"family_name" validate:"required"`
	FirstName   string `json:"first_name" validate:"required"`
	DateOfBirth string `json:"date_of_birth" validate:"required"`
	ExpiryDate  string `json:"expiry_date"`
}

// MedicareCardResponse Medicare card validation response
type MedicareCardResponse struct {
	Valid       bool       `json:"valid"`
	CardNumber  string     `json:"card_number"`
	IRN         string     `json:"irn"`
	CardColor   string     `json:"card_color"`
	Name        string     `json:"name"`
	DateOfBirth string     `json:"date_of_birth"`
	ExpiryDate  string     `json:"expiry_date"`
	Address     *AUAddress `json:"address,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// DriverLicenseRequest driver's license validation request for Australia
type AUDriverLicenseRequest struct {
	LicenseNumber string `json:"license_number" validate:"required"`
	Name          string `json:"name" validate:"required"`
	DateOfBirth   string `json:"date_of_birth" validate:"required"`
	State         string `json:"state" validate:"required,oneof=NSW VIC QLD SA WA TAS NT ACT"`
	CardNumber    string `json:"card_number,omitempty"` // For states that use card numbers
}

// DriverLicenseResponse driver's license validation response for Australia
type AUDriverLicenseResponse struct {
	Valid            bool       `json:"valid"`
	LicenseNumber    string     `json:"license_number"`
	Name             string     `json:"name"`
	DateOfBirth      string     `json:"date_of_birth"`
	State            string     `json:"state"`
	CardNumber       string     `json:"card_number,omitempty"`
	Address          *AUAddress `json:"address"`
	IssueDate        string     `json:"issue_date"`
	ExpiryDate       string     `json:"expiry_date"`
	LicenseClass     string     `json:"license_class"` // C, LR, MR, HR, HC, MC, etc.
	Conditions       string     `json:"conditions,omitempty"`
	IssuingAuthority string     `json:"issuing_authority,omitempty"`
	Error            string     `json:"error,omitempty"`
}

// PassportRequest passport verification request for Australia
type AUPassportRequest struct {
	PassportNumber string `json:"passport_number" validate:"required"`
	FirstName      string `json:"first_name" validate:"required"`
	LastName       string `json:"last_name" validate:"required"`
	DateOfBirth    string `json:"date_of_birth" validate:"required"`
	Nationality    string `json:"nationality,omitempty"`
	Gender         string `json:"gender,omitempty"`
}

// PassportResponse passport verification response for Australia
type AUPassportResponse struct {
	Valid            bool   `json:"valid"`
	PassportNumber   string `json:"passport_number"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	DateOfBirth      string `json:"date_of_birth"`
	Nationality      string `json:"nationality,omitempty"`
	Gender           string `json:"gender,omitempty"`
	IssueDate        string `json:"issue_date,omitempty"`
	ExpiryDate       string `json:"expiry_date,omitempty"`
	IssuingAuthority string `json:"issuing_authority,omitempty"`
	Error            string `json:"error,omitempty"`
}

// NewAustraliaIdentityConnector creates a new Australian identity connector
func NewAustraliaIdentityConnector(config *AustraliaConnectorConfig) (*AustraliaIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &AustraliaIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// AuthenticateMyGovID authenticates using myGovID
// Identity Proofing levels: IP1 (basic), IP2 (standard), IP3 (strong)
func (ac *AustraliaIdentityConnector) AuthenticateMyGovID(
	ctx context.Context,
	req *MyGovIDAuthRequest,
) (*MyGovIDAuthResponse, error) {
	// Validate request
	if err := ac.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate identity strength
	validStrengths := map[string]bool{"IP1": true, "IP2": true, "IP3": true}
	if !validStrengths[req.IdentityStrength] {
		return &MyGovIDAuthResponse{
			Success: false,
			Error:   "Invalid identity strength (must be IP1, IP2, or IP3)",
		}, nil
	}

	// In production, this would:
	// 1. Redirect user to myGovID login
	// 2. Perform OIDC authentication
	// 3. Receive identity claims
	// 4. Verify identity strength

	// Mock response for demonstration
	response := &MyGovIDAuthResponse{
		Success:          true,
		SessionID:        fmt.Sprintf("mygov_%d", time.Now().Unix()),
		IdentityStrength: req.IdentityStrength,
		UserInfo: &MyGovIDUserInfo{
			MyGovID:       fmt.Sprintf("MYGOV%d", time.Now().Unix()),
			GivenName:     "John",
			FamilyName:    "Smith",
			DateOfBirth:   "1990-01-15",
			Email:         "john.smith@example.com.au",
			EmailVerified: true,
		},
		Attributes: make(map[string]string),
	}

	return response, nil
}

// ValidateMedicareCard validates Australian Medicare card
func (ac *AustraliaIdentityConnector) ValidateMedicareCard(
	ctx context.Context,
	req *MedicareCardRequest,
) (*MedicareCardResponse, error) {
	// Validate request
	if err := ac.validator.Struct(req); err != nil {
		return &MedicareCardResponse{Valid: false, Error: err.Error()}, nil
	}

	cardNumber := strings.TrimSpace(req.CardNumber)
	irn := strings.TrimSpace(req.IRN)

	// Validate card number format (10 digits)
	if !regexp.MustCompile(`^\d{10}$`).MatchString(cardNumber) {
		return &MedicareCardResponse{
			Valid: false,
			Error: "Invalid Medicare card number format (must be 10 digits)",
		}, nil
	}

	// Validate IRN (Individual Reference Number: 1-9)
	if !regexp.MustCompile(`^[1-9]$`).MatchString(irn) {
		return &MedicareCardResponse{
			Valid: false,
			Error: "Invalid IRN (must be 1-9)",
		}, nil
	}

	// Validate card number check digit
	if !ac.validateMedicareCheckDigit(cardNumber) {
		return &MedicareCardResponse{
			Valid: false,
			Error: "Invalid Medicare card number check digit",
		}, nil
	}

	// In production, this would verify with Services Australia
	response := &MedicareCardResponse{
		Valid:       true,
		CardNumber:  cardNumber,
		IRN:         irn,
		CardColor:   req.CardColor,
		Name:        fmt.Sprintf("%s %s", req.FirstName, req.FamilyName),
		DateOfBirth: req.DateOfBirth,
		ExpiryDate:  req.ExpiryDate,
	}

	return response, nil
}

// VerifyDriverLicense verifies Australian driver's license
func (ac *AustraliaIdentityConnector) VerifyDriverLicense(
	ctx context.Context,
	req *AUDriverLicenseRequest,
) (*AUDriverLicenseResponse, error) {
	// Validate request
	if err := ac.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate license number format (varies by state)
	if !ac.validateLicenseNumberFormat(req.LicenseNumber, req.State) {
		return &AUDriverLicenseResponse{
			Valid: false,
			Error: fmt.Sprintf("Invalid license number format for state %s", req.State),
		}, nil
	}

	// In production, this would:
	// 1. Verify with state transport authority
	// 2. Check license status and demerit points
	// 3. Validate license classes

	// Mock response for demonstration
	response := &AUDriverLicenseResponse{
		Valid:         true,
		LicenseNumber: req.LicenseNumber,
		Name:          req.Name,
		DateOfBirth:   req.DateOfBirth,
		IssueDate:     "2020-01-15",
		ExpiryDate:    "2030-01-15",
		LicenseClass:  "C", // Car license
	}

	return response, nil
}

// VerifyPassport verifies Australian passport
func (ac *AustraliaIdentityConnector) VerifyPassport(ctx context.Context, req *AUPassportRequest) (*AUPassportResponse, error) {
	// Validate request
	if err := ac.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	passportNumber := strings.ToUpper(strings.TrimSpace(req.PassportNumber))

	// Validate passport number format (1 or 2 letters + 7 digits)
	if !regexp.MustCompile(`^[A-Z]{1,2}\d{7}$`).MatchString(passportNumber) {
		return &AUPassportResponse{
			Valid: false,
			Error: "Invalid passport number format",
		}, nil
	}

	// In production, this would verify with Department of Foreign Affairs and Trade
	response := &AUPassportResponse{
		Valid:            true,
		PassportNumber:   passportNumber,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		DateOfBirth:      req.DateOfBirth,
		Nationality:      "AUS",
		Gender:           req.Gender,
		IssueDate:        "2020-01-15",
		ExpiryDate:       "2030-01-15",
		IssuingAuthority: "Commonwealth of Australia",
	}

	return response, nil
}

// Helper methods

func (ac *AustraliaIdentityConnector) validateMedicareCheckDigit(cardNumber string) bool {
	// Medicare card uses weighted sum algorithm
	// Weights: 1, 3, 7, 9, 1, 3, 7, 9 for first 8 digits
	weights := []int{1, 3, 7, 9, 1, 3, 7, 9}
	sum := 0

	for i := 0; i < 8; i++ {
		digit := int(cardNumber[i] - '0')
		sum += digit * weights[i]
	}

	checkDigit := sum % 10
	actualCheckDigit := int(cardNumber[8] - '0')

	return checkDigit == actualCheckDigit
}

func (ac *AustraliaIdentityConnector) validateLicenseNumberFormat(licenseNumber, state string) bool {
	// License number formats vary by state
	patterns := map[string]string{
		"NSW": `^\d{8}$`,         // 8 digits
		"VIC": `^\d{10}$`,        // 10 digits
		"QLD": `^\d{8,9}$`,       // 8-9 digits
		"SA":  `^[A-Z]\d{6}$`,    // 1 letter + 6 digits
		"WA":  `^\d{7}$`,         // 7 digits
		"TAS": `^\d{8}$`,         // 8 digits
		"NT":  `^\d{6,8}$`,       // 6-8 digits
		"ACT": `^[A-Z]{2}\d{6}$`, // 2 letters + 6 digits
	}

	pattern, ok := patterns[state]
	if !ok {
		return false
	}

	matched, _ := regexp.MatchString(pattern, licenseNumber)
	return matched
}

func (ac *AustraliaIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

var _ = (*AustraliaIdentityConnector).generateCacheKey

// GetMetrics returns connector metrics
func (ac *AustraliaIdentityConnector) GetMetrics() map[string]interface{} {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	return map[string]interface{}{
		"connector": "australia_identity",
	}
}

// Close closes the connector and cleans up resources
func (ac *AustraliaIdentityConnector) Close() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return nil
}
