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

// SwedenIdentityConnector handles Swedish identity verification
// Supports BankID, personnummer validation, eIDAS
type SwedenIdentityConnector struct {
	config     *SwedenConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// SwedenConnectorConfig configuration for Swedish identity connector
type SwedenConnectorConfig struct {
	// BankID configuration
	BankIDURL          string `validate:"required,url"`
	BankIDClientID     string `validate:"required"`
	BankIDClientSecret string `validate:"required"`
	BankIDCertPath     string // Path to BankID certificate
	
	// eIDAS Node configuration
	EIDASNodeURL       string `validate:"url"`
	
	// Timeouts
	RequestTimeout     time.Duration
}

// BankIDAuthRequest BankID authentication request
type BankIDAuthRequest struct {
	PersonalNumber     string `json:"personal_number" validate:"required,len=12"`
	UseMobileBankID    bool   `json:"use_mobile_bankid"`
	UserVisibleData    string `json:"user_visible_data,omitempty"`
	UserNonVisibleData string `json:"user_non_visible_data,omitempty"`
	EndUserIP          string `json:"end_user_ip" validate:"required,ip"`
}

// BankIDAuthResponse BankID authentication response
type BankIDAuthResponse struct {
	Success            bool              `json:"success"`
	OrderRef           string            `json:"order_ref"`
	AutoStartToken     string            `json:"auto_start_token"`
	QRStartToken       string            `json:"qr_start_token"`
	QRStartSecret      string            `json:"qr_start_secret"`
	Status             string            `json:"status"` // pending, complete, failed
	HintCode           string            `json:"hint_code,omitempty"`
	CompletionData     *BankIDCompletion `json:"completion_data,omitempty"`
	Error              string            `json:"error,omitempty"`
}

// BankIDCompletion completion data from BankID
type BankIDCompletion struct {
	User               *BankIDUser       `json:"user"`
	Device             *BankIDDevice     `json:"device"`
	Cert               *BankIDCert       `json:"cert"`
	Signature          string            `json:"signature"`
	OcspResponse       string            `json:"ocsp_response"`
}

// BankIDUser user information from BankID
type BankIDUser struct {
	PersonalNumber     string `json:"personal_number"`
	Name               string `json:"name"`
	GivenName          string `json:"given_name"`
	Surname            string `json:"surname"`
}

// BankIDDevice device information
type BankIDDevice struct {
	IPAddress          string `json:"ip_address"`
}

// BankIDCert certificate information
type BankIDCert struct {
	NotBefore          string `json:"not_before"`
	NotAfter           string `json:"not_after"`
}

// BankIDSignRequest BankID signing request
type BankIDSignRequest struct {
	PersonalNumber     string `json:"personal_number" validate:"required,len=12"`
	UseMobileBankID    bool   `json:"use_mobile_bankid"`
	UserVisibleData    string `json:"user_visible_data" validate:"required"`
	UserNonVisibleData string `json:"user_non_visible_data,omitempty"`
	EndUserIP          string `json:"end_user_ip" validate:"required,ip"`
}

// BankIDSignResponse BankID signing response
type BankIDSignResponse struct {
	Success            bool              `json:"success"`
	OrderRef           string            `json:"order_ref"`
	AutoStartToken     string            `json:"auto_start_token"`
	QRStartToken       string            `json:"qr_start_token"`
	QRStartSecret      string            `json:"qr_start_secret"`
	Status             string            `json:"status"`
	CompletionData     *BankIDCompletion `json:"completion_data,omitempty"`
	Error              string            `json:"error,omitempty"`
}

// PersonnummerRequest personnummer validation request
type PersonnummerRequest struct {
	Personnummer       string `json:"personnummer" validate:"required"`
	IncludeCentury     bool   `json:"include_century"`
}

// PersonnummerResponse personnummer validation response
type PersonnummerResponse struct {
	Valid              bool   `json:"valid"`
	Personnummer       string `json:"personnummer"`
	Century            string `json:"century"`
	Year               string `json:"year"`
	Month              string `json:"month"`
	Day                string `json:"day"`
	BirthNumber        string `json:"birth_number"`
	CheckDigit         string `json:"check_digit"`
	Age                int    `json:"age"`
	Gender             string `json:"gender"` // male, female
	CoordinationNumber bool   `json:"coordination_number"` // +60 to day for immigrants
	Error              string `json:"error,omitempty"`
}

// NewSwedenIdentityConnector creates a new Swedish identity connector
func NewSwedenIdentityConnector(config *SwedenConnectorConfig) (*SwedenIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	
	connector := &SwedenIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}
	
	return connector, nil
}

// AuthenticateBankID authenticates using BankID
func (sc *SwedenIdentityConnector) AuthenticateBankID(ctx context.Context, req *BankIDAuthRequest) (*BankIDAuthResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate personnummer first
	pnrReq := &PersonnummerRequest{
		Personnummer: req.PersonalNumber,
	}
	pnrResp, err := sc.ValidatePersonnummer(ctx, pnrReq)
	if err != nil || !pnrResp.Valid {
		return &BankIDAuthResponse{
			Success: false,
			Error:   "Invalid personnummer",
		}, nil
	}
	
	// In production, this would:
	// 1. Call BankID API /auth endpoint
	// 2. Return orderRef and autoStartToken
	// 3. Poll /collect endpoint for completion
	// 4. Validate signature and OCSP response
	
	// Mock response for demonstration
	response := &BankIDAuthResponse{
		Success:        true,
		OrderRef:       fmt.Sprintf("order_%d", time.Now().Unix()),
		AutoStartToken: fmt.Sprintf("token_%d", time.Now().Unix()),
		QRStartToken:   fmt.Sprintf("qr_%d", time.Now().Unix()),
		QRStartSecret:  generateSecret(),
		Status:         "complete",
		CompletionData: &BankIDCompletion{
			User: &BankIDUser{
				PersonalNumber: req.PersonalNumber,
				Name:           "Erik Svensson",
				GivenName:      "Erik",
				Surname:        "Svensson",
			},
			Device: &BankIDDevice{
				IPAddress: req.EndUserIP,
			},
			Cert: &BankIDCert{
				NotBefore: "2020-01-01",
				NotAfter:  "2025-01-01",
			},
			Signature:    "mock_signature",
			OcspResponse: "mock_ocsp",
		},
	}
	
	return response, nil
}

// SignWithBankID signs data using BankID
func (sc *SwedenIdentityConnector) SignWithBankID(ctx context.Context, req *BankIDSignRequest) (*BankIDSignResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate personnummer
	pnrReq := &PersonnummerRequest{
		Personnummer: req.PersonalNumber,
	}
	pnrResp, err := sc.ValidatePersonnummer(ctx, pnrReq)
	if err != nil || !pnrResp.Valid {
		return &BankIDSignResponse{
			Success: false,
			Error:   "Invalid personnummer",
		}, nil
	}
	
	// In production, this would:
	// 1. Call BankID API /sign endpoint
	// 2. Return orderRef
	// 3. Poll /collect endpoint for completion
	// 4. Return signature
	
	// Mock response for demonstration
	response := &BankIDSignResponse{
		Success:        true,
		OrderRef:       fmt.Sprintf("sign_%d", time.Now().Unix()),
		AutoStartToken: fmt.Sprintf("token_%d", time.Now().Unix()),
		Status:         "complete",
		CompletionData: &BankIDCompletion{
			User: &BankIDUser{
				PersonalNumber: req.PersonalNumber,
				Name:           "Erik Svensson",
			},
			Signature: "mock_signature_" + req.UserVisibleData,
		},
	}
	
	return response, nil
}

// ValidatePersonnummer validates Swedish personnummer (personal identity number)
// Format: YYYYMMDD-XXXX or YYMMDD-XXXX
// - YYYYMMDD or YYMMDD: Date of birth
// - XXX: Birth number (odd = male, even = female)
// - X: Check digit (Luhn algorithm)
// Coordination numbers: Day + 60 for immigrants without Swedish birth
func (sc *SwedenIdentityConnector) ValidatePersonnummer(ctx context.Context, req *PersonnummerRequest) (*PersonnummerResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return &PersonnummerResponse{Valid: false, Error: err.Error()}, nil
	}
	
	// Remove separators and spaces
	pnr := strings.ReplaceAll(req.Personnummer, "-", "")
	pnr = strings.ReplaceAll(pnr, "+", "")
	pnr = strings.TrimSpace(pnr)
	
	// Validate length (10 or 12 digits)
	if len(pnr) != 10 && len(pnr) != 12 {
		return &PersonnummerResponse{
			Valid: false,
			Error: "Personnummer must be 10 or 12 digits",
		}, nil
	}
	
	// Validate all digits
	if !regexp.MustCompile(`^\d+$`).MatchString(pnr) {
		return &PersonnummerResponse{
			Valid: false,
			Error: "Personnummer must contain only digits",
		}, nil
	}
	
	// Extract components
	var century, year, month, day, birthNum, checkDigit string
	var isCoordination bool
	
	if len(pnr) == 12 {
		century = pnr[0:2]
		year = pnr[2:4]
		month = pnr[4:6]
		day = pnr[6:8]
		birthNum = pnr[8:11]
		checkDigit = pnr[11:12]
	} else {
		// 10 digits - assume current century
		currentYear := time.Now().Year()
		var y int
		fmt.Sscanf(pnr[0:2], "%d", &y)
		if y > currentYear%100 {
			century = fmt.Sprintf("%d", (currentYear/100)-1)
		} else {
			century = fmt.Sprintf("%d", currentYear/100)
		}
		year = pnr[0:2]
		month = pnr[2:4]
		day = pnr[4:6]
		birthNum = pnr[6:9]
		checkDigit = pnr[9:10]
	}
	
	// Check for coordination number (day + 60)
	var dayInt int
	fmt.Sscanf(day, "%d", &dayInt)
	if dayInt > 60 {
		isCoordination = true
		dayInt -= 60
		day = fmt.Sprintf("%02d", dayInt)
	}
	
	// Validate month (01-12)
	var monthInt int
	fmt.Sscanf(month, "%d", &monthInt)
	if monthInt < 1 || monthInt > 12 {
		return &PersonnummerResponse{
			Valid: false,
			Error: "Invalid month",
		}, nil
	}
	
	// Validate day (01-31)
	if dayInt < 1 || dayInt > 31 {
		return &PersonnummerResponse{
			Valid: false,
			Error: "Invalid day",
		}, nil
	}
	
	// Validate check digit using Luhn algorithm
	testNumber := year + month + day + birthNum
	if !sc.validateLuhn(testNumber, checkDigit) {
		return &PersonnummerResponse{
			Valid: false,
			Error: "Invalid check digit",
		}, nil
	}
	
	// Determine gender (odd = male, even = female)
	var birthNumInt int
	fmt.Sscanf(birthNum, "%d", &birthNumInt)
	gender := "female"
	if birthNumInt%2 == 1 {
		gender = "male"
	}
	
	// Calculate age
	birthYear, _ := time.Parse("2006", century+year)
	age := time.Now().Year() - birthYear.Year()
	
	response := &PersonnummerResponse{
		Valid:              true,
		Personnummer:       req.Personnummer,
		Century:            century,
		Year:               year,
		Month:              month,
		Day:                day,
		BirthNumber:        birthNum,
		CheckDigit:         checkDigit,
		Age:                age,
		Gender:             gender,
		CoordinationNumber: isCoordination,
	}
	
	return response, nil
}

// Helper methods

func (sc *SwedenIdentityConnector) validateLuhn(number string, checkDigit string) bool {
	// Luhn algorithm for Swedish personnummer
	sum := 0
	for i, char := range number {
		digit := int(char - '0')
		if i%2 == 0 {
			// Double every other digit
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	
	// Calculate check digit
	calculatedCheck := (10 - (sum % 10)) % 10
	expectedCheck := int(checkDigit[0] - '0')
	
	return calculatedCheck == expectedCheck
}

func generateSecret() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (sc *SwedenIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (sc *SwedenIdentityConnector) GetMetrics() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	return map[string]interface{}{
		"connector": "sweden_identity",
	}
}

// Close closes the connector and cleans up resources
func (sc *SwedenIdentityConnector) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return nil
}
