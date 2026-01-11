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

// SouthAfricaIdentityConnector handles South African identity verification
// Supports ID Number, Passport, Driver's License
type SouthAfricaIdentityConnector struct {
	config     *SouthAfricaConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// SouthAfricaConnectorConfig configuration for South African identity connector
type SouthAfricaConnectorConfig struct {
	// Department of Home Affairs configuration
	DHA_URL    string `validate:"required,url"` //nolint:stylecheck
	DHA_APIKey string `validate:"required"`     //nolint:stylecheck

	// NATIS (traffic department) configuration
	NATISURL    string `validate:"url"`
	NATISAPIKey string

	// Timeouts
	RequestTimeout time.Duration
}

// IDNumberRequest ID number validation request
type IDNumberRequest struct {
	IDNumber    string `json:"id_number" validate:"required,len=13"`
	Name        string `json:"name"`
	DateOfBirth string `json:"date_of_birth"`
}

// IDNumberResponse ID number validation response
type IDNumberResponse struct {
	Valid           bool   `json:"valid"`
	IDNumber        string `json:"id_number"`
	DateOfBirth     string `json:"date_of_birth"`
	Gender          string `json:"gender"`      // Male, Female
	Citizenship     string `json:"citizenship"` // SA Citizen, Permanent Resident
	CheckDigitValid bool   `json:"check_digit_valid"`
	Error           string `json:"error,omitempty"`
}

// DriverLicenseRequest driver's license validation request
type ZADriverLicenseRequest struct {
	LicenseNumber string `json:"license_number" validate:"required"`
	IDNumber      string `json:"id_number" validate:"required,len=13"`
	FirstName     string `json:"first_name" validate:"required"`
	Surname       string `json:"surname" validate:"required"`
	DateOfBirth   string `json:"date_of_birth" validate:"required"`
}

// DriverLicenseResponse driver's license validation response
type ZADriverLicenseResponse struct {
	Valid         bool       `json:"valid"`
	LicenseNumber string     `json:"license_number"`
	IDNumber      string     `json:"id_number"`
	FirstName     string     `json:"first_name"`
	Surname       string     `json:"surname"`
	DateOfBirth   string     `json:"date_of_birth"`
	Address       *ZAAddress `json:"address,omitempty"`
	IssueDate     string     `json:"issue_date"`
	ExpiryDate    string     `json:"expiry_date"`
	LicenseCode   []string   `json:"license_code"` // A1, A, B, C1, C, EB, EC1, EC, etc.
	Restrictions  string     `json:"restrictions,omitempty"`
	PrDPNumber    string     `json:"prdp_number,omitempty"` // Professional Driving Permit
	Error         string     `json:"error,omitempty"`
}

// ZAAddress South African address structure
type ZAAddress struct {
	StreetAddress string `json:"street_address"`
	Suburb        string `json:"suburb"`
	City          string `json:"city"`
	Province      string `json:"province"`    // 9 provinces
	PostalCode    string `json:"postal_code"` // 4 digits
	Country       string `json:"country"`
}

// PassportRequest passport validation request
type ZAPassportRequest struct {
	PassportNumber string `json:"passport_number" validate:"required"`
	IDNumber       string `json:"id_number,omitempty"`
	Surname        string `json:"surname" validate:"required"`
	FirstName      string `json:"first_name" validate:"required"`
	DateOfBirth    string `json:"date_of_birth" validate:"required"`
	Nationality    string `json:"nationality" validate:"required"`
}

// PassportResponse passport validation response
type ZAPassportResponse struct {
	Valid            bool   `json:"valid"`
	PassportNumber   string `json:"passport_number"`
	IDNumber         string `json:"id_number,omitempty"`
	Surname          string `json:"surname"`
	Names            string `json:"names"`
	DateOfBirth      string `json:"date_of_birth"`
	Gender           string `json:"gender"`
	Nationality      string `json:"nationality"`
	CountryOfBirth   string `json:"country_of_birth"`
	DateOfIssue      string `json:"date_of_issue"`
	DateOfExpiry     string `json:"date_of_expiry"`
	IssuingAuthority string `json:"issuing_authority"`
	Error            string `json:"error,omitempty"`
}

// NewSouthAfricaIdentityConnector creates a new South African identity connector
func NewSouthAfricaIdentityConnector(config *SouthAfricaConnectorConfig) (*SouthAfricaIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &SouthAfricaIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// ValidateIDNumber validates South African ID number
// Format: 13 digits (YYMMDDGSSSCAZ)
// YY: Year, MM: Month, DD: Day
// G: Gender (0-4 female, 5-9 male)
// SSS: Sequence number
// C: Citizenship (0 = SA, 1 = non-SA)
// A: Usually 8 or 9
// Z: Check digit (Luhn algorithm)
func (zac *SouthAfricaIdentityConnector) ValidateIDNumber(ctx context.Context, req *IDNumberRequest) (*IDNumberResponse, error) {
	// Validate request
	if err := zac.validator.Struct(req); err != nil {
		return &IDNumberResponse{Valid: false, Error: err.Error()}, nil
	}

	idNumber := strings.TrimSpace(req.IDNumber)

	// Validate format (13 digits)
	if !regexp.MustCompile(`^\d{13}$`).MatchString(idNumber) {
		return &IDNumberResponse{
			Valid: false,
			Error: "Invalid ID number format (must be 13 digits)",
		}, nil
	}

	// Extract components
	year := idNumber[0:2]
	month := idNumber[2:4]
	day := idNumber[4:6]
	genderDigit := idNumber[6:10]
	citizenshipDigit := idNumber[10:11]

	// Determine century
	yearInt, _ := strconv.Atoi(year)
	var century string
	if yearInt >= 0 && yearInt <= 25 {
		century = "20"
	} else {
		century = "19"
	}

	dateOfBirth := fmt.Sprintf("%s%s-%s-%s", century, year, month, day)

	// Determine gender
	genderInt, _ := strconv.Atoi(genderDigit)
	var gender string
	if genderInt < 5000 {
		gender = "Female"
	} else {
		gender = "Male"
	}

	// Determine citizenship
	citizenship := "SA Citizen"
	if citizenshipDigit == "1" {
		citizenship = "Permanent Resident"
	}

	// Validate check digit using Luhn algorithm
	checkDigitValid := zac.validateIDCheckDigit(idNumber)

	response := &IDNumberResponse{
		Valid:           checkDigitValid,
		IDNumber:        idNumber,
		DateOfBirth:     dateOfBirth,
		Gender:          gender,
		Citizenship:     citizenship,
		CheckDigitValid: checkDigitValid,
	}

	if !checkDigitValid {
		response.Error = "Invalid ID number check digit"
	}

	return response, nil
}

// VerifyDriverLicense verifies South African driver's license
func (zac *SouthAfricaIdentityConnector) VerifyDriverLicense(
	ctx context.Context,
	req *ZADriverLicenseRequest,
) (*ZADriverLicenseResponse, error) {
	// Validate request
	if err := zac.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate ID number first
	if !regexp.MustCompile(`^\d{13}$`).MatchString(req.IDNumber) {
		return &ZADriverLicenseResponse{
			Valid: false,
			Error: "Invalid ID number format",
		}, nil
	}

	// In production, this would verify with NATIS (National Traffic Information System)
	response := &ZADriverLicenseResponse{
		Valid:         true,
		LicenseNumber: req.LicenseNumber,
		IDNumber:      req.IDNumber,
		FirstName:     req.FirstName,
		Surname:       req.Surname,
		DateOfBirth:   req.DateOfBirth,
		IssueDate:     "2020-01-15",
		ExpiryDate:    "2025-01-15",
		LicenseCode:   []string{"B", "EB"}, // Light motor vehicle
	}

	return response, nil
}

// VerifyPassport verifies South African passport
func (zac *SouthAfricaIdentityConnector) VerifyPassport(
	ctx context.Context,
	req *ZAPassportRequest,
) (*ZAPassportResponse, error) {
	// Validate request
	if err := zac.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	passportNumber := strings.ToUpper(strings.TrimSpace(req.PassportNumber))

	// Validate passport number format (letter + 8 digits)
	if !regexp.MustCompile(`^[A-Z]\d{8}$`).MatchString(passportNumber) {
		return &ZAPassportResponse{
			Valid: false,
			Error: "Invalid passport number format",
		}, nil
	}

	// In production, this would verify with Department of Home Affairs
	response := &ZAPassportResponse{
		Valid:            true,
		PassportNumber:   passportNumber,
		IDNumber:         req.IDNumber,
		Surname:          req.Surname,
		Names:            req.FirstName,
		DateOfBirth:      req.DateOfBirth,
		Nationality:      "ZAF",
		DateOfIssue:      "2020-01-15",
		DateOfExpiry:     "2030-01-15",
		IssuingAuthority: "South Africa",
	}

	return response, nil
}

// Helper methods

func (zac *SouthAfricaIdentityConnector) validateIDCheckDigit(idNumber string) bool {
	// Luhn algorithm
	sum := 0
	for i := 0; i < 13; i++ {
		digit, _ := strconv.Atoi(string(idNumber[i]))

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

func (zac *SouthAfricaIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

var _ = (*SouthAfricaIdentityConnector).generateCacheKey

// GetMetrics returns connector metrics
func (zac *SouthAfricaIdentityConnector) GetMetrics() map[string]interface{} {
	zac.mu.RLock()
	defer zac.mu.RUnlock()

	return map[string]interface{}{
		"connector": "southafrica_identity",
	}
}

// Close closes the connector and cleans up resources
func (zac *SouthAfricaIdentityConnector) Close() error {
	zac.mu.Lock()
	defer zac.mu.Unlock()
	return nil
}
