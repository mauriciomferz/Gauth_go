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

// SpainIdentityConnector handles Spanish identity verification
// Supports Cl@ve, DNI electrónico, NIE, FNMT certificates
type SpainIdentityConnector struct {
	config     *SpainConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// SpainConnectorConfig configuration for Spanish identity connector
type SpainConnectorConfig struct {
	// Cl@ve configuration
	ClaveURL          string `validate:"required,url"`
	ClaveClientID     string `validate:"required"`
	ClaveClientSecret string `validate:"required"`

	// FNMT (Fábrica Nacional de Moneda y Timbre) configuration
	FNMTURL    string `validate:"url"`
	FNMTAPIKey string

	// DNI electrónico configuration
	DNIeURL    string `validate:"url"`
	DNIeAPIKey string

	// Timeouts
	RequestTimeout time.Duration
}

// ClaveAuthRequest Cl@ve authentication request
type ClaveAuthRequest struct {
	ServiceID           string   `json:"service_id" validate:"required"`
	ClaveLevel          string   `json:"clave_level" validate:"required,oneof=low substantial high"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
	State               string   `json:"state" validate:"required"`
}

// ClaveAuthResponse Cl@ve authentication response
type ClaveAuthResponse struct {
	Success    bool              `json:"success"`
	SessionID  string            `json:"session_id"`
	ClaveLevel string            `json:"clave_level"`
	UserInfo   *ClaveUserInfo    `json:"user_info"`
	Attributes map[string]string `json:"attributes"`
	Error      string            `json:"error,omitempty"`
}

// ClaveUserInfo user information from Cl@ve
type ClaveUserInfo struct {
	ID            string     `json:"id"` // Unique identifier
	FirstName     string     `json:"first_name"`
	FirstSurname  string     `json:"first_surname"`
	SecondSurname string     `json:"second_surname"`
	DNI           string     `json:"dni"` // Spanish national ID
	NIE           string     `json:"nie"` // Foreigner ID number
	DateOfBirth   string     `json:"date_of_birth"`
	Email         string     `json:"email"`
	MobilePhone   string     `json:"mobile_phone"`
	Address       *ESAddress `json:"address"`
}

// ESAddress Spanish address structure
type ESAddress struct {
	Street              string `json:"street"`
	Number              string `json:"number"`
	Floor               string `json:"floor"`
	Door                string `json:"door"`
	PostalCode          string `json:"postal_code"`
	City                string `json:"city"`
	Province            string `json:"province"`
	AutonomousCommunity string `json:"autonomous_community"`
	Country             string `json:"country"`
}

// DNIValidationRequest DNI/NIE validation request
type DNIValidationRequest struct {
	DocumentNumber string `json:"document_number" validate:"required"`
	DocumentType   string `json:"document_type" validate:"required,oneof=DNI NIE"`
	FirstName      string `json:"first_name"`
	FirstSurname   string `json:"first_surname"`
	DateOfBirth    string `json:"date_of_birth"`
}

// DNIValidationResponse DNI/NIE validation response
type DNIValidationResponse struct {
	Valid          bool   `json:"valid"`
	DocumentNumber string `json:"document_number"`
	DocumentType   string `json:"document_type"`
	ControlLetter  string `json:"control_letter"`
	IssueDate      string `json:"issue_date"`
	ExpiryDate     string `json:"expiry_date"`
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
}

// DNIeVerificationRequest DNI electrónico verification request
type DNIeVerificationRequest struct {
	DNINumber       string `json:"dni_number" validate:"required"`
	CertificateData []byte `json:"certificate_data" validate:"required"`
	PIN             string `json:"pin" validate:"required"`
	UseNFC          bool   `json:"use_nfc"`
}

// DNIeVerificationResponse DNI electrónico verification response
type DNIeVerificationResponse struct {
	Valid            bool          `json:"valid"`
	DNINumber        string        `json:"dni_number"`
	ChipVerified     bool          `json:"chip_verified"`
	CertificateValid bool          `json:"certificate_valid"`
	UserInfo         *DNIeUserInfo `json:"user_info"`
	Error            string        `json:"error,omitempty"`
}

// DNIeUserInfo user information from DNI electrónico
type DNIeUserInfo struct {
	DNI           string `json:"dni"`
	Name          string `json:"name"`
	FirstSurname  string `json:"first_surname"`
	SecondSurname string `json:"second_surname"`
	DateOfBirth   string `json:"date_of_birth"`
	PlaceOfBirth  string `json:"place_of_birth"`
	Gender        string `json:"gender"`
	Nationality   string `json:"nationality"`
	IssueDate     string `json:"issue_date"`
	ExpiryDate    string `json:"expiry_date"`
}

// FNMTCertificateRequest FNMT certificate verification request
type FNMTCertificateRequest struct {
	CertificateData []byte `json:"certificate_data" validate:"required"`
	SerialNumber    string `json:"serial_number"`
}

// FNMTCertificateResponse FNMT certificate verification response
type FNMTCertificateResponse struct {
	Valid           bool   `json:"valid"`
	SerialNumber    string `json:"serial_number"`
	Subject         string `json:"subject"`
	Issuer          string `json:"issuer"`
	NotBefore       string `json:"not_before"`
	NotAfter        string `json:"not_after"`
	CertificateType string `json:"certificate_type"` // person, company, seal
	Status          string `json:"status"`           // valid, revoked, expired
	Error           string `json:"error,omitempty"`
}

// NewSpainIdentityConnector creates a new Spanish identity connector
func NewSpainIdentityConnector(config *SpainConnectorConfig) (*SpainIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &SpainIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// AuthenticateClave authenticates using Cl@ve (Clave Permanente)
func (sc *SpainIdentityConnector) AuthenticateClave(ctx context.Context, req *ClaveAuthRequest) (*ClaveAuthResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate Cl@ve level (maps to eIDAS levels)
	if req.ClaveLevel != "low" && req.ClaveLevel != "substantial" && req.ClaveLevel != "high" {
		return &ClaveAuthResponse{
			Success: false,
			Error:   "Invalid Cl@ve level (must be low, substantial, or high)",
		}, nil
	}

	// In production, this would:
	// 1. Generate SAML AuthnRequest
	// 2. Redirect to Cl@ve identity provider
	// 3. Receive SAML Response
	// 4. Validate assertion

	// Mock response for demonstration
	response := &ClaveAuthResponse{
		Success:    true,
		SessionID:  fmt.Sprintf("clave_%d", time.Now().Unix()),
		ClaveLevel: req.ClaveLevel,
		UserInfo: &ClaveUserInfo{
			ID:            fmt.Sprintf("CL_%d", time.Now().Unix()),
			FirstName:     "Juan",
			FirstSurname:  "García",
			SecondSurname: "López",
			DNI:           "12345678Z",
			Email:         "juan.garcia@example.es",
		},
		Attributes: make(map[string]string),
	}

	return response, nil
}

// ValidateDNI validates Spanish DNI or NIE
// DNI format: 8 digits + 1 control letter (e.g., 12345678Z)
// NIE format: 1 letter (X, Y, Z) + 7 digits + 1 control letter (e.g., X1234567L)
func (sc *SpainIdentityConnector) ValidateDNI(ctx context.Context, req *DNIValidationRequest) (*DNIValidationResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return &DNIValidationResponse{Valid: false, Error: err.Error()}, nil
	}

	docNum := strings.ToUpper(strings.TrimSpace(req.DocumentNumber))

	var valid bool
	var controlLetter string

	switch req.DocumentType {
	case "DNI":
		valid, controlLetter = sc.validateDNINumber(docNum)
	case "NIE":
		valid, controlLetter = sc.validateNIENumber(docNum)
	default:
		return &DNIValidationResponse{
			Valid: false,
			Error: "Invalid document type (must be DNI or NIE)",
		}, nil
	}

	if !valid {
		return &DNIValidationResponse{
			Valid:          false,
			DocumentNumber: docNum,
			DocumentType:   req.DocumentType,
			Error:          "Invalid document number or control letter",
		}, nil
	}

	response := &DNIValidationResponse{
		Valid:          true,
		DocumentNumber: docNum,
		DocumentType:   req.DocumentType,
		ControlLetter:  controlLetter,
		Status:         "valid",
	}

	return response, nil
}

// VerifyDNIe verifies DNI electrónico (electronic national ID card)
func (sc *SpainIdentityConnector) VerifyDNIe(
	ctx context.Context,
	req *DNIeVerificationRequest,
) (*DNIeVerificationResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate DNI format first
	dniReq := &DNIValidationRequest{
		DocumentNumber: req.DNINumber,
		DocumentType:   "DNI",
	}
	dniResp, err := sc.ValidateDNI(ctx, dniReq)
	if err != nil || !dniResp.Valid {
		return &DNIeVerificationResponse{
			Valid: false,
			Error: "Invalid DNI number",
		}, nil
	}

	// In production, this would:
	// 1. Read DNI electrónico chip (via NFC or card reader)
	// 2. Verify PIN
	// 3. Extract certificate
	// 4. Validate certificate chain
	// 5. Read personal data

	// Mock response for demonstration
	response := &DNIeVerificationResponse{
		Valid:            true,
		DNINumber:        req.DNINumber,
		ChipVerified:     true,
		CertificateValid: true,
		UserInfo: &DNIeUserInfo{
			DNI:           req.DNINumber,
			Name:          "Juan",
			FirstSurname:  "García",
			SecondSurname: "López",
			DateOfBirth:   "1980-01-15",
			Gender:        "M",
			Nationality:   "ESP",
			IssueDate:     "2020-01-15",
			ExpiryDate:    "2030-01-15",
		},
	}

	return response, nil
}

// VerifyFNMTCertificate verifies FNMT digital certificate
func (sc *SpainIdentityConnector) VerifyFNMTCertificate(
	ctx context.Context,
	req *FNMTCertificateRequest,
) (*FNMTCertificateResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// In production, this would:
	// 1. Parse X.509 certificate
	// 2. Validate certificate chain
	// 3. Check OCSP/CRL status
	// 4. Extract subject information

	// Mock response for demonstration
	response := &FNMTCertificateResponse{
		Valid:           true,
		SerialNumber:    req.SerialNumber,
		Subject:         "CN=Juan García López, SERIALNUMBER=12345678Z",
		Issuer:          "CN=FNMT Clase 2 CA",
		NotBefore:       "2020-01-15",
		NotAfter:        "2025-01-15",
		CertificateType: "person",
		Status:          "valid",
	}

	return response, nil
}

// Helper methods

func (sc *SpainIdentityConnector) validateDNINumber(dni string) (bool, string) {
	// DNI format: 8 digits + 1 control letter
	if !regexp.MustCompile(`^\d{8}[A-Z]$`).MatchString(dni) {
		return false, ""
	}

	// Extract number and letter
	number := dni[0:8]
	letter := dni[8:9]

	// Calculate control letter
	var num int
	_, _ = fmt.Sscanf(number, "%d", &num) // Best effort parsing; will be 0 if invalid

	letters := "TRWAGMYFPDXBNJZSQVHLCKE"
	expectedLetter := string(letters[num%23])

	return letter == expectedLetter, expectedLetter
}

func (sc *SpainIdentityConnector) validateNIENumber(nie string) (bool, string) {
	// NIE format: 1 letter (X, Y, Z) + 7 digits + 1 control letter
	if !regexp.MustCompile(`^[XYZ]\d{7}[A-Z]$`).MatchString(nie) {
		return false, ""
	}

	// Replace first letter with digit for calculation
	firstLetter := nie[0:1]
	number := nie[1:8]
	letter := nie[8:9]

	var prefix string
	switch firstLetter {
	case "X":
		prefix = "0"
	case "Y":
		prefix = "1"
	case "Z":
		prefix = "2"
	default:
		return false, ""
	}

	// Calculate control letter
	fullNumber := prefix + number
	var num int
	_, _ = fmt.Sscanf(fullNumber, "%d", &num) // Best effort parsing; will be 0 if invalid

	letters := "TRWAGMYFPDXBNJZSQVHLCKE"
	expectedLetter := string(letters[num%23])

	return letter == expectedLetter, expectedLetter
}

func (sc *SpainIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

var _ = (*SpainIdentityConnector).generateCacheKey

// GetMetrics returns connector metrics
func (sc *SpainIdentityConnector) GetMetrics() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return map[string]interface{}{
		"connector": "spain_identity",
	}
}

// Close closes the connector and cleans up resources
func (sc *SpainIdentityConnector) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return nil
}
