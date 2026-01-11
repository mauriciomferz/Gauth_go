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

// JapanIdentityConnector handles Japanese identity verification
// Supports My Number Card, Residence Card, Driver's License, Individual Number
type JapanIdentityConnector struct {
	config     *JapanConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// JapanConnectorConfig configuration for Japanese identity connector
type JapanConnectorConfig struct {
	// My Number Card configuration
	MyNumberURL      string `validate:"required,url"`
	MyNumberClientID string `validate:"required"`
	MyNumberSecret   string `validate:"required"`

	// JPKI (Japan Public Key Infrastructure) configuration
	JPKIAuthURL string `validate:"url"`
	JPKICertURL string `validate:"url"`

	// eKYC API configuration
	EKYCServiceURL string `validate:"url"`
	EKYCAPIKey     string

	// Timeouts
	RequestTimeout time.Duration
}

// MyNumberAuthRequest My Number Card authentication request
type MyNumberAuthRequest struct {
	CardNumber          string   `json:"card_number" validate:"required"`
	PIN                 string   `json:"pin" validate:"required,len=4"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	UseJPKI             bool     `json:"use_jpki"` // Use JPKI certificate
}

// MyNumberAuthResponse My Number Card authentication response
type MyNumberAuthResponse struct {
	Success    bool              `json:"success"`
	SessionID  string            `json:"session_id"`
	UserInfo   *MyNumberUserInfo `json:"user_info"`
	Attributes map[string]string `json:"attributes"`
	JPKICert   *JPKICertificate  `json:"jpki_cert,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// MyNumberUserInfo user information from My Number Card
type MyNumberUserInfo struct {
	IndividualNumber string     `json:"individual_number"` // 12-digit My Number
	Name             *JPName    `json:"name"`
	DateOfBirth      string     `json:"date_of_birth"`
	Gender           string     `json:"gender"`
	Address          *JPAddress `json:"address"`
	CardIssueDate    string     `json:"card_issue_date"`
	CardExpiryDate   string     `json:"card_expiry_date"`
	PhotoData        string     `json:"photo_data,omitempty"` // Base64 encoded
}

// JPName Japanese name structure (Kanji, Hiragana, Romaji)
type JPName struct {
	FamilyNameKanji  string `json:"family_name_kanji"`
	GivenNameKanji   string `json:"given_name_kanji"`
	FamilyNameKana   string `json:"family_name_kana"`
	GivenNameKana    string `json:"given_name_kana"`
	FamilyNameRomaji string `json:"family_name_romaji"`
	GivenNameRomaji  string `json:"given_name_romaji"`
}

// JPAddress Japanese address structure
type JPAddress struct {
	PostalCode   string `json:"postal_code"`
	Prefecture   string `json:"prefecture"`
	City         string `json:"city"`
	Town         string `json:"town"`
	BuildingName string `json:"building_name,omitempty"`
	RoomNumber   string `json:"room_number,omitempty"`
}

// JPKICertificate JPKI certificate information
type JPKICertificate struct {
	Type            string `json:"type"` // "auth" or "sign"
	SerialNumber    string `json:"serial_number"`
	IssuerDN        string `json:"issuer_dn"`
	SubjectDN       string `json:"subject_dn"`
	NotBefore       string `json:"not_before"`
	NotAfter        string `json:"not_after"`
	PublicKey       string `json:"public_key"`
	CertificateData string `json:"certificate_data"` // Base64 encoded
}

// IndividualNumberRequest Individual Number (My Number) validation request
type IndividualNumberRequest struct {
	IndividualNumber string  `json:"individual_number" validate:"required,len=12"`
	Name             *JPName `json:"name"`
	DateOfBirth      string  `json:"date_of_birth"`
}

// IndividualNumberResponse Individual Number validation response
type IndividualNumberResponse struct {
	Valid            bool   `json:"valid"`
	IndividualNumber string `json:"individual_number"`
	CheckDigitValid  bool   `json:"check_digit_valid"`
	Error            string `json:"error,omitempty"`
}

// ResidenceCardRequest residence card validation request
type ResidenceCardRequest struct {
	CardNumber      string `json:"card_number" validate:"required"`
	Nationality     string `json:"nationality" validate:"required"`
	DateOfBirth     string `json:"date_of_birth"`
	ResidenceStatus string `json:"residence_status"`
}

// ResidenceCardResponse residence card validation response
type ResidenceCardResponse struct {
	Valid            bool       `json:"valid"`
	CardNumber       string     `json:"card_number"`
	Name             *JPName    `json:"name"`
	Nationality      string     `json:"nationality"`
	DateOfBirth      string     `json:"date_of_birth"`
	Gender           string     `json:"gender"`
	ResidenceStatus  string     `json:"residence_status"`
	ExpiryDate       string     `json:"expiry_date"`
	Address          *JPAddress `json:"address"`
	WorkRestrictions string     `json:"work_restrictions,omitempty"`
	Error            string     `json:"error,omitempty"`
}

// JPDriverLicenseRequest driver's license validation request for Japan
type JPDriverLicenseRequest struct {
	LicenseNumber     string  `json:"license_number" validate:"required,len=12"`
	Name              *JPName `json:"name" validate:"required"`
	DateOfBirth       string  `json:"date_of_birth" validate:"required"`
	IssuingPrefecture string  `json:"issuing_prefecture"`
}

// JPDriverLicenseResponse driver's license validation response for Japan
type JPDriverLicenseResponse struct {
	Valid            bool       `json:"valid"`
	LicenseNumber    string     `json:"license_number"`
	Name             *JPName    `json:"name"`
	DateOfBirth      string     `json:"date_of_birth"`
	Address          *JPAddress `json:"address"`
	IssueDate        string     `json:"issue_date"`
	ExpiryDate       string     `json:"expiry_date"`
	LicenseTypes     []string   `json:"license_types"` // e.g., "普通", "大型", "二輪"
	Conditions       string     `json:"conditions,omitempty"`
	IssuingAuthority string     `json:"issuing_authority"`
	Error            string     `json:"error,omitempty"`
}

// NewJapanIdentityConnector creates a new Japanese identity connector
func NewJapanIdentityConnector(config *JapanConnectorConfig) (*JapanIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &JapanIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// AuthenticateMyNumber authenticates using My Number Card
func (jc *JapanIdentityConnector) AuthenticateMyNumber(
	ctx context.Context,
	req *MyNumberAuthRequest,
) (*MyNumberAuthResponse, error) {
	// Validate request
	if err := jc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate card number format (alphanumeric)
	if !regexp.MustCompile(`^[A-Z0-9]{8,20}$`).MatchString(req.CardNumber) {
		return &MyNumberAuthResponse{
			Success: false,
			Error:   "Invalid card number format",
		}, nil
	}

	// Validate PIN (4 digits)
	if !regexp.MustCompile(`^\d{4}$`).MatchString(req.PIN) {
		return &MyNumberAuthResponse{
			Success: false,
			Error:   "Invalid PIN format (must be 4 digits)",
		}, nil
	}

	// In production, this would:
	// 1. Read My Number Card chip via NFC or card reader
	// 2. Verify PIN
	// 3. Extract certificate and personal data
	// 4. Optionally retrieve JPKI certificate

	// Mock response for demonstration
	response := &MyNumberAuthResponse{
		Success:   true,
		SessionID: fmt.Sprintf("mynumber_%d", time.Now().Unix()),
		UserInfo: &MyNumberUserInfo{
			IndividualNumber: "123456789012",
			Name: &JPName{
				FamilyNameKanji:  "山田",
				GivenNameKanji:   "太郎",
				FamilyNameKana:   "やまだ",
				GivenNameKana:    "たろう",
				FamilyNameRomaji: "Yamada",
				GivenNameRomaji:  "Taro",
			},
			DateOfBirth:    "1990-01-15",
			Gender:         "M",
			CardIssueDate:  "2020-01-01",
			CardExpiryDate: "2030-01-01",
		},
		Attributes: make(map[string]string),
	}

	if req.UseJPKI {
		response.JPKICert = &JPKICertificate{
			Type:         "auth",
			SerialNumber: "1234567890",
			IssuerDN:     "CN=JPKI,O=JPKI,C=JP",
			SubjectDN:    "CN=Yamada Taro,O=Individual,C=JP",
			NotBefore:    "2020-01-01T00:00:00Z",
			NotAfter:     "2030-01-01T00:00:00Z",
		}
	}

	return response, nil
}

// ValidateIndividualNumber validates Japanese Individual Number (My Number)
// Format: 12 digits with check digit using modulo 11 algorithm
func (jc *JapanIdentityConnector) ValidateIndividualNumber(
	ctx context.Context,
	req *IndividualNumberRequest,
) (*IndividualNumberResponse, error) {
	// Validate request
	if err := jc.validator.Struct(req); err != nil {
		return &IndividualNumberResponse{Valid: false, Error: err.Error()}, nil
	}

	number := strings.TrimSpace(req.IndividualNumber)

	// Validate format (12 digits)
	if !regexp.MustCompile(`^\d{12}$`).MatchString(number) {
		return &IndividualNumberResponse{
			Valid: false,
			Error: "Invalid Individual Number format (must be 12 digits)",
		}, nil
	}

	// Validate check digit (last digit) using modulo 11 algorithm
	checkDigitValid := jc.validateMyNumberCheckDigit(number)

	response := &IndividualNumberResponse{
		Valid:            checkDigitValid,
		IndividualNumber: number,
		CheckDigitValid:  checkDigitValid,
	}

	if !checkDigitValid {
		response.Error = "Invalid check digit"
	}

	return response, nil
}

// VerifyResidenceCard verifies Japanese residence card (在留カード)
func (jc *JapanIdentityConnector) VerifyResidenceCard(
	ctx context.Context,
	req *ResidenceCardRequest,
) (*ResidenceCardResponse, error) {
	// Validate request
	if err := jc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate card number format (2 letters + 8 digits + 1 check letter)
	if !regexp.MustCompile(`^[A-Z]{2}\d{8}[A-Z]$`).MatchString(req.CardNumber) {
		return &ResidenceCardResponse{
			Valid: false,
			Error: "Invalid residence card number format",
		}, nil
	}

	// In production, this would:
	// 1. Verify card with Immigration Services Agency database
	// 2. Check residence status and expiry
	// 3. Validate work restrictions

	// Mock response for demonstration
	response := &ResidenceCardResponse{
		Valid:      true,
		CardNumber: req.CardNumber,
		Name: &JPName{
			FamilyNameRomaji: "Smith",
			GivenNameRomaji:  "John",
		},
		Nationality:      req.Nationality,
		DateOfBirth:      req.DateOfBirth,
		Gender:           "M",
		ResidenceStatus:  "Engineer/Specialist in Humanities/International Services",
		ExpiryDate:       "2028-01-15",
		WorkRestrictions: "None",
	}

	return response, nil
}

// VerifyDriverLicense verifies Japanese driver's license
func (jc *JapanIdentityConnector) VerifyDriverLicense(
	ctx context.Context,
	req *JPDriverLicenseRequest,
) (*JPDriverLicenseResponse, error) {
	// Validate request
	if err := jc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	licenseNumber := strings.TrimSpace(req.LicenseNumber)

	// Validate format (12 digits)
	if !regexp.MustCompile(`^\d{12}$`).MatchString(licenseNumber) {
		return &JPDriverLicenseResponse{
			Valid: false,
			Error: "Invalid license number format (must be 12 digits)",
		}, nil
	}

	// Extract prefecture code (first 2 digits)
	prefectureCode := licenseNumber[0:2]

	// In production, this would:
	// 1. Verify with National Police Agency database
	// 2. Check license status and violations
	// 3. Validate license types and conditions

	// Mock response for demonstration
	response := &JPDriverLicenseResponse{
		Valid:            true,
		LicenseNumber:    licenseNumber,
		Name:             req.Name,
		DateOfBirth:      req.DateOfBirth,
		IssueDate:        "2020-01-15",
		ExpiryDate:       "2027-01-15",
		IssuingAuthority: fmt.Sprintf("Prefecture %s Public Safety Commission", prefectureCode),
	}

	return response, nil
}

// Helper methods

func (jc *JapanIdentityConnector) validateMyNumberCheckDigit(number string) bool {
	// My Number uses modulo 11 check digit algorithm
	// P_n = 11 - ((Q_n + 6) % 11)
	// Where Q_n = Σ(P_i × Q_i) for i = 1 to 11
	// Q_i values: 6, 5, 4, 3, 2, 7, 6, 5, 4, 3, 2

	weights := []int{6, 5, 4, 3, 2, 7, 6, 5, 4, 3, 2}
	sum := 0

	for i := 0; i < 11; i++ {
		digit, _ := strconv.Atoi(string(number[i]))
		sum += digit * weights[i]
	}

	calculatedCheck := 11 - ((sum + 6) % 11)
	if calculatedCheck >= 10 {
		calculatedCheck = 0
	}

	actualCheck, _ := strconv.Atoi(string(number[11]))

	return calculatedCheck == actualCheck
}

func (jc *JapanIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

var _ = (*JapanIdentityConnector).generateCacheKey

// GetMetrics returns connector metrics
func (jc *JapanIdentityConnector) GetMetrics() map[string]interface{} {
	jc.mu.RLock()
	defer jc.mu.RUnlock()

	return map[string]interface{}{
		"connector": "japan_identity",
	}
}

// Close closes the connector and cleans up resources
func (jc *JapanIdentityConnector) Close() error {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	return nil
}
