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

// ItalyIdentityConnector handles Italian identity verification
// Supports SPID, CIE, Codice Fiscale, eIDAS
type ItalyIdentityConnector struct {
	config     *ItalyConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// ItalyConnectorConfig configuration for Italian identity connector
type ItalyConnectorConfig struct {
	// SPID configuration
	SPIDMetadataURL   string `validate:"required,url"`
	SPIDAggregatorURL string `validate:"url"`
	ServiceProviderID string `validate:"required"`
	
	// CIE (Carta d'Identità Elettronica) configuration
	CIEURL          string `validate:"url"`
	CIEClientID     string
	CIEClientSecret string
	
	// eIDAS Node configuration
	EIDASNodeURL string `validate:"url"`
	
	// Timeouts
	RequestTimeout time.Duration
}

// SPIDAuthRequest SPID authentication request
type SPIDAuthRequest struct {
	ServiceProviderID   string   `json:"service_provider_id" validate:"required"`
	SPIDLevel           string   `json:"spid_level" validate:"required,oneof=1 2 3"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
}

// SPIDAuthResponse SPID authentication response
type SPIDAuthResponse struct {
	Success          bool              `json:"success"`
	AssertionID      string            `json:"assertion_id"`
	SPIDLevel        string            `json:"spid_level"`
	IdentityProvider string            `json:"identity_provider"`
	UserInfo         *SPIDUserInfo     `json:"user_info"`
	Attributes       map[string]string `json:"attributes"`
	Error            string            `json:"error,omitempty"`
}

// SPIDUserInfo user information from SPID
type SPIDUserInfo struct {
	SpidCode      string     `json:"spid_code"` // Unique identifier
	Name          string     `json:"name"`
	FamilyName    string     `json:"family_name"`
	FiscalNumber  string     `json:"fiscal_number"` // Codice Fiscale
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	MobilePhone   string     `json:"mobile_phone"`
	DateOfBirth   string     `json:"date_of_birth"`
	Gender        string     `json:"gender"`
	PlaceOfBirth  string     `json:"place_of_birth"`
	CountyOfBirth string     `json:"county_of_birth"`
	Address       *ITAddress `json:"address"`
	DigitalAddress string    `json:"digital_address"` // PEC (Posta Elettronica Certificata)
}

// ITAddress Italian address structure
type ITAddress struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	City       string `json:"city"`
	Province   string `json:"province"`
	PostalCode string `json:"postal_code"`
	Region     string `json:"region"`
	Country    string `json:"country"`
}

// CodiceFiscaleRequest Codice Fiscale validation request
type CodiceFiscaleRequest struct {
	CodiceFiscale string `json:"codice_fiscale" validate:"required,len=16"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	DateOfBirth   string `json:"date_of_birth"`
	Gender        string `json:"gender" validate:"omitempty,oneof=M F"`
}

// CodiceFiscaleResponse Codice Fiscale validation response
type CodiceFiscaleResponse struct {
	Valid            bool   `json:"valid"`
	CodiceFiscale    string `json:"codice_fiscale"`
	LastName         string `json:"last_name"` // First 3 consonants
	FirstName        string `json:"first_name"` // First 3 consonants
	YearOfBirth      string `json:"year_of_birth"`
	MonthOfBirth     string `json:"month_of_birth"`
	DayOfBirth       string `json:"day_of_birth"`
	Gender           string `json:"gender"` // Encoded in day
	PlaceOfBirth     string `json:"place_of_birth"` // Cadastral code
	ControlCharacter string `json:"control_character"`
	Error            string `json:"error,omitempty"`
}

// CIEAuthRequest CIE authentication request
type CIEAuthRequest struct {
	ServiceID           string   `json:"service_id" validate:"required"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
	UseMobileApp        bool     `json:"use_mobile_app"` // Use CieID mobile app
}

// CIEAuthResponse CIE authentication response
type CIEAuthResponse struct {
	Success    bool              `json:"success"`
	SessionID  string            `json:"session_id"`
	UserInfo   *CIEUserInfo      `json:"user_info"`
	Attributes map[string]string `json:"attributes"`
	Error      string            `json:"error,omitempty"`
}

// CIEUserInfo user information from CIE
type CIEUserInfo struct {
	DocumentNumber string     `json:"document_number"`
	Name           string     `json:"name"`
	FamilyName     string     `json:"family_name"`
	FiscalCode     string     `json:"fiscal_code"`
	DateOfBirth    string     `json:"date_of_birth"`
	PlaceOfBirth   string     `json:"place_of_birth"`
	Gender         string     `json:"gender"`
	Citizenship    string     `json:"citizenship"`
	Address        *ITAddress `json:"address"`
	IssueDate      string     `json:"issue_date"`
	ExpiryDate     string     `json:"expiry_date"`
}

// NewItalyIdentityConnector creates a new Italian identity connector
func NewItalyIdentityConnector(config *ItalyConnectorConfig) (*ItalyIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	
	connector := &ItalyIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}
	
	return connector, nil
}

// AuthenticateSPID authenticates using SPID (Sistema Pubblico di Identità Digitale)
func (ic *ItalyIdentityConnector) AuthenticateSPID(ctx context.Context, req *SPIDAuthRequest) (*SPIDAuthResponse, error) {
	// Validate request
	if err := ic.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate SPID level
	if req.SPIDLevel != "1" && req.SPIDLevel != "2" && req.SPIDLevel != "3" {
		return &SPIDAuthResponse{
			Success: false,
			Error:   "Invalid SPID level (must be 1, 2, or 3)",
		}, nil
	}
	
	// In production, this would:
	// 1. Generate SAML AuthnRequest
	// 2. Redirect user to SPID identity provider
	// 3. Receive SAML Response
	// 4. Validate assertion
	
	// Mock response for demonstration
	response := &SPIDAuthResponse{
		Success:          true,
		AssertionID:      fmt.Sprintf("spid_%d", time.Now().Unix()),
		SPIDLevel:        req.SPIDLevel,
		IdentityProvider: "PosteID", // Example: Poste Italiane
		UserInfo: &SPIDUserInfo{
			SpidCode:     fmt.Sprintf("SPID_%d", time.Now().Unix()),
			Name:         "Mario",
			FamilyName:   "Rossi",
			FiscalNumber: "RSSMRA80A01H501U",
			Email:        "mario.rossi@example.it",
			DateOfBirth:  "1980-01-01",
			Gender:       "M",
		},
		Attributes: make(map[string]string),
	}
	
	return response, nil
}

// ValidateCodiceFiscale validates Italian Codice Fiscale (tax code)
// Format: RSSMRA80A01H501U (16 characters)
// - 3 chars: Last name consonants
// - 3 chars: First name consonants
// - 2 digits: Year of birth
// - 1 letter: Month of birth (A=Jan, B=Feb, ..., T=Dec)
// - 2 digits: Day of birth (01-31 for males, 41-71 for females)
// - 4 chars: Municipality code (Belfiore code)
// - 1 char: Control character
func (ic *ItalyIdentityConnector) ValidateCodiceFiscale(ctx context.Context, req *CodiceFiscaleRequest) (*CodiceFiscaleResponse, error) {
	// Validate request
	if err := ic.validator.Struct(req); err != nil {
		return &CodiceFiscaleResponse{Valid: false, Error: err.Error()}, nil
	}
	
	cf := strings.ToUpper(strings.TrimSpace(req.CodiceFiscale))
	
	// Validate format (16 alphanumeric characters)
	if !regexp.MustCompile(`^[A-Z]{6}\d{2}[A-Z]\d{2}[A-Z]\d{3}[A-Z]$`).MatchString(cf) {
		return &CodiceFiscaleResponse{
			Valid: false,
			Error: "Invalid Codice Fiscale format (must be 16 alphanumeric characters)",
		}, nil
	}
	
	// Extract components
	lastName := cf[0:3]
	firstName := cf[3:6]
	year := cf[6:8]
	monthCode := cf[8:9]
	day := cf[9:11]
	placeCode := cf[11:15]
	controlChar := cf[15:16]
	
	// Decode month (A=Jan, B=Feb, ..., T=Dec)
	monthMap := map[string]string{
		"A": "01", "B": "02", "C": "03", "D": "04",
		"E": "05", "H": "06", "L": "07", "M": "08",
		"P": "09", "R": "10", "S": "11", "T": "12",
	}
	month, monthValid := monthMap[monthCode]
	if !monthValid {
		return &CodiceFiscaleResponse{
			Valid: false,
			Error: "Invalid month code",
		}, nil
	}
	
	// Decode day and gender
	// Males: 01-31, Females: 41-71 (day + 40)
	var dayInt int
	_, _ = fmt.Sscanf(day, "%d", &dayInt) // Best effort parsing; will be 0 if invalid
	gender := "M"
	if dayInt > 40 {
		gender = "F"
		dayInt -= 40
	}
	
	if dayInt < 1 || dayInt > 31 {
		return &CodiceFiscaleResponse{
			Valid: false,
			Error: "Invalid day",
		}, nil
	}
	
	// Validate control character
	calculatedControl := ic.calculateCodiceFiscaleControl(cf[0:15])
	if controlChar != calculatedControl {
		return &CodiceFiscaleResponse{
			Valid: false,
			Error: fmt.Sprintf("Invalid control character (expected %s, got %s)", calculatedControl, controlChar),
		}, nil
	}
	
	response := &CodiceFiscaleResponse{
		Valid:            true,
		CodiceFiscale:    cf,
		LastName:         lastName,
		FirstName:        firstName,
		YearOfBirth:      year,
		MonthOfBirth:     month,
		DayOfBirth:       fmt.Sprintf("%02d", dayInt),
		Gender:           gender,
		PlaceOfBirth:     placeCode,
		ControlCharacter: controlChar,
	}
	
	return response, nil
}

// AuthenticateCIE authenticates using CIE (Carta d'Identità Elettronica)
func (ic *ItalyIdentityConnector) AuthenticateCIE(ctx context.Context, req *CIEAuthRequest) (*CIEAuthResponse, error) {
	// Validate request
	if err := ic.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// In production, this would:
	// 1. Read CIE chip via NFC or mobile app
	// 2. Verify PIN
	// 3. Extract certificate and personal data
	// 4. Validate certificate chain
	
	// Mock response for demonstration
	response := &CIEAuthResponse{
		Success:   true,
		SessionID: fmt.Sprintf("cie_%d", time.Now().Unix()),
		UserInfo: &CIEUserInfo{
			DocumentNumber: "CA12345AB",
			Name:           "Mario",
			FamilyName:     "Rossi",
			FiscalCode:     "RSSMRA80A01H501U",
			DateOfBirth:    "1980-01-01",
			Gender:         "M",
			Citizenship:    "ITA",
			IssueDate:      "2020-01-15",
			ExpiryDate:     "2030-01-15",
		},
		Attributes: make(map[string]string),
	}
	
	return response, nil
}

// Helper methods

func (ic *ItalyIdentityConnector) calculateCodiceFiscaleControl(first15 string) string {
	// Odd position values (0-indexed, so positions 0, 2, 4, ...)
	oddValues := map[rune]int{
		'0': 1, '1': 0, '2': 5, '3': 7, '4': 9, '5': 13, '6': 15, '7': 17, '8': 19, '9': 21,
		'A': 1, 'B': 0, 'C': 5, 'D': 7, 'E': 9, 'F': 13, 'G': 15, 'H': 17, 'I': 19, 'J': 21,
		'K': 2, 'L': 4, 'M': 18, 'N': 20, 'O': 11, 'P': 3, 'Q': 6, 'R': 8, 'S': 12, 'T': 14,
		'U': 16, 'V': 10, 'W': 22, 'X': 25, 'Y': 24, 'Z': 23,
	}
	
	// Even position values (0-indexed, so positions 1, 3, 5, ...)
	evenValues := map[rune]int{
		'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
		'A': 0, 'B': 1, 'C': 2, 'D': 3, 'E': 4, 'F': 5, 'G': 6, 'H': 7, 'I': 8, 'J': 9,
		'K': 10, 'L': 11, 'M': 12, 'N': 13, 'O': 14, 'P': 15, 'Q': 16, 'R': 17, 'S': 18, 'T': 19,
		'U': 20, 'V': 21, 'W': 22, 'X': 23, 'Y': 24, 'Z': 25,
	}
	
	sum := 0
	for i, char := range first15 {
		if i%2 == 0 {
			sum += oddValues[char]
		} else {
			sum += evenValues[char]
		}
	}
	
	// Control character
	controlChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return string(controlChars[sum%26])
}

func (ic *ItalyIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (ic *ItalyIdentityConnector) GetMetrics() map[string]interface{} {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	
	return map[string]interface{}{
		"connector": "italy_identity",
	}
}

// Close closes the connector and cleans up resources
func (ic *ItalyIdentityConnector) Close() error {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	return nil
}
