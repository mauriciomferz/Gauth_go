package external

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

// CanadaIdentityConnector handles Canadian identity verification
// Supports SIN, Driver's License, Passport, Provincial Health Cards
type CanadaIdentityConnector struct {
	config     *CanadaConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// CanadaConnectorConfig configuration for Canadian identity connector
type CanadaConnectorConfig struct {
	// Service Canada configuration
	ServiceCanadaURL string `validate:"required,url"`
	ServiceCanadaKey string `validate:"required"`

	// Provincial services configuration
	ProvincialServicesURL string `validate:"url"`
	ProvincialAPIKey      string

	// IRCC (Immigration) configuration
	IRCCURL    string `validate:"url"`
	IRCCAPIKey string

	// Timeouts
	RequestTimeout time.Duration
}

// SINRequest SIN (Social Insurance Number) validation request
type SINRequest struct {
	SIN         string `json:"sin" validate:"required,len=9"`
	Name        string `json:"name"`
	DateOfBirth string `json:"date_of_birth"`
}

// SINResponse SIN validation response
type SINResponse struct {
	Valid           bool   `json:"valid"`
	SIN             string `json:"sin"`  // Masked: ***-***-123
	Type            string `json:"type"` // Permanent, Temporary, Business
	CheckDigitValid bool   `json:"check_digit_valid"`
	Error           string `json:"error,omitempty"`
}

// DriverLicenseRequest driver's license validation request for Canada
type CADriverLicenseRequest struct {
	LicenseNumber string `json:"license_number" validate:"required"`
	Name          string `json:"name" validate:"required"`
	DateOfBirth   string `json:"date_of_birth" validate:"required"`
	Province      string `json:"province" validate:"required,oneof=ON QC BC AB MB SK NS NB NL PE NT YT NU"`
}

// DriverLicenseResponse driver's license validation response for Canada
type CADriverLicenseResponse struct {
	Valid            bool       `json:"valid"`
	LicenseNumber    string     `json:"license_number"`
	Name             string     `json:"name"`
	DateOfBirth      string     `json:"date_of_birth"`
	Province         string     `json:"province"`
	Address          *CAAddress `json:"address,omitempty"`
	IssueDate        string     `json:"issue_date"`
	ExpiryDate       string     `json:"expiry_date"`
	LicenseClass     string     `json:"license_class"` // G, G2, M, M2, A, D, etc.
	Conditions       string     `json:"conditions,omitempty"`
	Endorsements     []string   `json:"endorsements,omitempty"`
	IssuinagentAuthority string     `json:"issuing_authority,omitempty"`
	Error            string     `json:"error,omitempty"`
}

// CAAddress Canadian address structure
type CAAddress struct {
	StreetNumber string `json:"street_number"`
	StreetName   string `json:"street_name"`
	Unit         string `json:"unit,omitempty"`
	City         string `json:"city"`
	Province     string `json:"province"`    // 2-letter code
	PostalCode   string `json:"postal_code"` // A1A 1A1
	Country      string `json:"country"`
}

// CAPassportRequest passport verification request for Canada
type CAPassportRequest struct {
	PassportNumber string `json:"passport_number" validate:"required"`
	FirstName      string `json:"first_name" validate:"required"`
	LastName       string `json:"last_name" validate:"required"`
	DateOfBirth    string `json:"date_of_birth" validate:"required"`
	Nationality    string `json:"nationality,omitempty"`
	Gender         string `json:"gender,omitempty"`
}

// CAPassportResponse passport verification response for Canada
type CAPassportResponse struct {
	Valid            bool   `json:"valid"`
	PassportNumber   string `json:"passport_number"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	DateOfBirth      string `json:"date_of_birth"`
	Nationality      string `json:"nationality,omitempty"`
	Gender           string `json:"gender,omitempty"`
	IssueDate        string `json:"issue_date,omitempty"`
	ExpiryDate       string `json:"expiry_date,omitempty"`
	IssuinagentAuthority string `json:"issuing_authority,omitempty"`
	Error            string `json:"error,omitempty"`
}

// HealthCardRequest provincial health card validation request
type HealthCardRequest struct {
	HealthNumber string `json:"health_number" validate:"required"`
	Province     string `json:"province" validate:"required,oneof=ON QC BC AB MB SK NS NB NL PE"`
	VersionCode  string `json:"version_code,omitempty"`
	LastName     string `json:"last_name" validate:"required"`
	FirstName    string `json:"first_name" validate:"required"`
	DateOfBirth  string `json:"date_of_birth" validate:"required"`
}

// HealthCardResponse health card validation response
type HealthCardResponse struct {
	Valid        bool   `json:"valid"`
	HealthNumber string `json:"health_number"`
	Province     string `json:"province"`
	VersionCode  string `json:"version_code,omitempty"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	DateOfBirth  string `json:"date_of_birth"`
	Gender       string `json:"gender"`
	ExpiryDate   string `json:"expiry_date,omitempty"`
	Error        string `json:"error,omitempty"`
}

// NewCanadaIdentityConnector creates a new Canadian identity connector
func NewCanadaIdentityConnector(config *CanadaConnectorConfig) (*CanadaIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &CanadaIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// ValidateSIN validates Canadian SIN (Social Insurance Number)
// Format: 9 digits (123-456-789)
// First digit determines type: 1-7 permanent, 9 temporary, 0 business
func (cc *CanadaIdentityConnector) ValidateSIN(ctx context.Context, req *SINRequest) (*SINResponse, error) {
	// Validate request
	if err := cc.validator.Struct(req); err != nil {
		return &SINResponse{Valid: false, Error: err.Error()}, nil
	}

	// Remove formatting
	sin := strings.ReplaceAll(req.SIN, "-", "")
	sin = strings.ReplaceAll(sin, " ", "")
	sin = strings.TrimSpace(sin)

	// Validate format (9 digits)
	if !regexp.MustCompile(`^\d{9}$`).MatchString(sin) {
		return &SINResponse{
			Valid: false,
			Error: "Invalid SIN format (must be 9 digits)",
		}, nil
	}

	// Determine SIN type from first digit
	firstDigit := sin[0]
	var sinType string
	switch {
	case firstDigit >= '1' && firstDigit <= '7':
		sinType = "Permanent"
	case firstDigit == '9':
		sinType = "Temporary"
	case firstDigit == '0':
		sinType = "Business"
	default:
		return &SINResponse{
			Valid: false,
			Error: "Invalid SIN first digit",
		}, nil
	}

	// Validate check digit using Luhn algorithm
	checkDigitValid := cc.validateSINCheckDigit(sin)

	// Mask SIN
	maskedSIN := fmt.Sprintf("***-***-%s", sin[6:9])

	response := &SINResponse{
		Valid:           checkDigitValid,
		SIN:             maskedSIN,
		Type:            sinType,
		CheckDigitValid: checkDigitValid,
	}

	if !checkDigitValid {
		response.Error = "Invalid SIN check digit"
	}

	return response, nil
}

// VerifyDriverLicense verifies Canadian driver's license
func (cc *CanadaIdentityConnector) VerifyDriverLicense(ctx context.Context, req *CADriverLicenseRequest) (*CADriverLicenseResponse, error) {
	// Validate request
	if err := cc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate license number format (varies by province)
	if !cc.validateLicenseFormat(req.LicenseNumber, req.Province) {
		return &CADriverLicenseResponse{
			Valid: false,
			Error: fmt.Sprintf("Invalid license number format for province %s", req.Province),
		}, nil
	}

	// In production, this would:
	// 1. Verify with provincial motor vehicle department
	// 2. Check license status and suspensions
	// 3. Validate demerit points

	// Mock response for demonstration
	response := &CADriverLicenseResponse{
		Valid:         true,
		LicenseNumber: req.LicenseNumber,
		Name:          req.Name,
		DateOfBirth:   req.DateOfBirth,
		IssueDate:     "2020-01-15",
		ExpiryDate:    "2025-01-15",
		LicenseClass:  "G", // Full license
	}

	return response, nil
}

// VerifyPassport verifies Canadian passport
func (cc *CanadaIdentityConnector) VerifyPassport(ctx context.Context, req *CAPassportRequest) (*CAPassportResponse, error) {
	// Validate request
	if err := cc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	passportNumber := strings.ToUpper(strings.TrimSpace(req.PassportNumber))

	// Validate passport number format (2 letters + 6 digits)
	if !regexp.MustCompile(`^[A-Z]{2}\d{6}$`).MatchString(passportNumber) {
		return &CAPassportResponse{
			Valid: false,
			Error: "Invalid passport number format",
		}, nil
	}

	// In production, this would verify with IRCC (Immigration, Refugees and Citizenship Canada)
	response := &CAPassportResponse{
		Valid:            true,
		PassportNumber:   passportNumber,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		DateOfBirth:      req.DateOfBirth,
		Nationality:      "CAN",
		Gender:           req.Gender,
		IssueDate:        "2020-01-15",
		ExpiryDate:       "2030-01-15",
		IssuinagentAuthority: "Canada",
	}

	return response, nil
}

// VerifyHealthCard verifies provincial health card
func (cc *CanadaIdentityConnector) VerifyHealthCard(ctx context.Context, req *HealthCardRequest) (*HealthCardResponse, error) {
	// Validate request
	if err := cc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate health number format (varies by province)
	if !cc.validateHealthCardFormat(req.HealthNumber, req.Province) {
		return &HealthCardResponse{
			Valid: false,
			Error: fmt.Sprintf("Invalid health card format for province %s", req.Province),
		}, nil
	}

	// In production, this would verify with provincial health ministry
	response := &HealthCardResponse{
		Valid:        true,
		HealthNumber: req.HealthNumber,
		Province:     req.Province,
		VersionCode:  req.VersionCode,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		DateOfBirth:  req.DateOfBirth,
	}

	return response, nil
}

// Helper methods

func (cc *CanadaIdentityConnector) validateSINCheckDigit(sin string) bool {
	// Luhn algorithm (same as credit cards)
	sum := 0
	for i := 0; i < 9; i++ {
		digit, _ := strconv.Atoi(string(sin[i]))

		// Double every second digit
		if i%2 == 1 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	return sum%10 == 0
}

func (cc *CanadaIdentityConnector) validateLicenseFormat(licenseNumber, province string) bool {
	// License formats vary by province
	patterns := map[string]string{
		"ON": `^[A-Z]\d{14}$`,    // 1 letter + 14 digits
		"QC": `^[A-Z]\d{13}$`,    // 1 letter + 13 digits
		"BC": `^\d{7}$`,          // 7 digits
		"AB": `^\d{6}-\d{3}$`,    // 6 digits - 3 digits
		"MB": `^[A-Z0-9]{1,12}$`, // Alphanumeric, 1-12 chars
		"SK": `^\d{8}$`,          // 8 digits
		"NS": `^[A-Z]{5}\d{9}$`,  // 5 letters + 9 digits
		"NB": `^\d{7}$`,          // 7 digits
		"NL": `^[A-Z]\d{9}$`,     // 1 letter + 9 digits
		"PE": `^\d{1,6}$`,        // 1-6 digits
		"NT": `^\d{6}$`,          // 6 digits
		"YT": `^\d{6}$`,          // 6 digits
		"NU": `^\d{6}$`,          // 6 digits
	}

	pattern, ok := patterns[province]
	if !ok {
		return false
	}

	matched, _ := regexp.MatchString(pattern, licenseNumber)
	return matched
}

func (cc *CanadaIdentityConnector) validateHealthCardFormat(healthNumber, province string) bool {
	// Health card formats vary by province
	patterns := map[string]string{
		"ON": `^\d{10}$`,        // 10 digits
		"QC": `^[A-Z]{4}\d{8}$`, // 4 letters + 8 digits
		"BC": `^\d{10}$`,        // 10 digits
		"AB": `^\d{9}$`,         // 9 digits
		"MB": `^\d{9}$`,         // 9 digits
		"SK": `^\d{9}$`,         // 9 digits
		"NS": `^\d{10}$`,        // 10 digits
		"NB": `^\d{9}$`,         // 9 digits
		"NL": `^[A-Z0-9]{12}$`,  // 12 alphanumeric
		"PE": `^\d{9}$`,         // 9 digits
	}

	pattern, ok := patterns[province]
	if !ok {
		return false
	}

	matched, _ := regexp.MatchString(pattern, healthNumber)
	return matched
}

func (cc *CanadaIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (cc *CanadaIdentityConnector) GetMetrics() map[string]interface{} {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return map[string]interface{}{
		"connector": "canada_identity",
	}
}

// Close closes the connector and cleans up resources
func (cc *CanadaIdentityConnector) Close() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return nil
}
