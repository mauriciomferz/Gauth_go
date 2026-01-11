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

// MexicoIdentityConnector handles Mexican identity verification
// Supports CURP, RFC, INE, Passport
type MexicoIdentityConnector struct {
	config     *MexicoConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// MexicoConnectorConfig configuration for Mexican identity connector
type MexicoConnectorConfig struct {
	// RENAPO (birth certificate registry) configuration
	RENAPOURL    string `validate:"required,url"`
	RENAPOAPIKey string `validate:"required"`

	// SAT (tax authority) configuration
	SATURL    string `validate:"url"`
	SATAPIKey string

	// INE (voter registration) configuration
	INEURL    string `validate:"url"`
	INEAPIKey string

	// Timeouts
	RequestTimeout time.Duration
}

// CURPRequest CURP validation request
type CURPRequest struct {
	CURP        string `json:"curp" validate:"required,len=18"`
	Name        string `json:"name"`
	DateOfBirth string `json:"date_of_birth"`
}

// CURPResponse CURP validation response
type CURPResponse struct {
	Valid           bool   `json:"valid"`
	CURP            string `json:"curp"`
	Name            string `json:"name,omitempty"`
	Gender          string `json:"gender"` // H (Hombre), M (Mujer)
	DateOfBirth     string `json:"date_of_birth"`
	StateOfBirth    string `json:"state_of_birth"` // 2-letter code
	CheckDigitValid bool   `json:"check_digit_valid"`
	Error           string `json:"error,omitempty"`
}

// RFCRequest RFC validation request
type RFCRequest struct {
	RFC          string `json:"rfc" validate:"required"`
	Name         string `json:"name"`
	TaxpayerType string `json:"taxpayer_type" validate:"omitempty,oneof=person company"`
}

// RFCResponse RFC validation response
type RFCResponse struct {
	Valid           bool   `json:"valid"`
	RFC             string `json:"rfc"`
	Name            string `json:"name,omitempty"`
	TaxpayerType    string `json:"taxpayer_type"` // Person, Company
	Status          string `json:"status"`        // Active, Inactive, Suspended
	CheckDigitValid bool   `json:"check_digit_valid"`
	Error           string `json:"error,omitempty"`
}

// INERequest INE (voter ID) validation request
type INERequest struct {
	INENumber   string `json:"ine_number" validate:"required"`
	CIC         string `json:"cic" validate:"required"` // Clave de Identificación del Ciudadano
	OCR         string `json:"ocr,omitempty"`           // Optical Character Recognition code
	FirstName   string `json:"first_name" validate:"required"`
	LastName    string `json:"last_name" validate:"required"`
	DateOfBirth string `json:"date_of_birth" validate:"required"`
}

// INEResponse INE validation response
type INEResponse struct {
	Valid        bool       `json:"valid"`
	INENumber    string     `json:"ine_number"`
	CIC          string     `json:"cic"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	CURP         string     `json:"curp,omitempty"`
	DateOfBirth  string     `json:"date_of_birth"`
	Gender       string     `json:"gender"`
	Address      *MXAddress `json:"address,omitempty"`
	IssueDate    string     `json:"issue_date"`
	ExpiryDate   string     `json:"expiry_date"`
	EmissionYear string     `json:"emission_year"`
	Section      string     `json:"section,omitempty"` // Sección electoral
	Error        string     `json:"error,omitempty"`
}

// MXAddress Mexican address structure
type MXAddress struct {
	Calle          string `json:"calle"`                     // Street
	NumeroExterior string `json:"numero_exterior"`           // Exterior number
	NumeroInterior string `json:"numero_interior,omitempty"` // Interior number
	Colonia        string `json:"colonia"`                   // Neighborhood
	Municipio      string `json:"municipio"`                 // Municipality
	Estado         string `json:"estado"`                    // State
	CodigoPostal   string `json:"codigo_postal"`             // Postal code (5 digits)
	Pais           string `json:"pais"`                      // Country
}

// PassportRequest passport validation request
type MXPassportRequest struct {
	PassportNumber string `json:"passport_number" validate:"required"`
	FirstName      string `json:"first_name" validate:"required"`
	LastName       string `json:"last_name" validate:"required"`
	DateOfBirth    string `json:"date_of_birth" validate:"required"`
	Nationality    string `json:"nationality" validate:"required"`
}

// PassportResponse passport validation response
type MXPassportResponse struct {
	Valid            bool   `json:"valid"`
	PassportNumber   string `json:"passport_number"`
	GivenNames       string `json:"given_names"`
	Surname          string `json:"surname"`
	CURP             string `json:"curp,omitempty"`
	DateOfBirth      string `json:"date_of_birth"`
	Gender           string `json:"gender"`
	Nationality      string `json:"nationality"`
	PlaceOfBirth     string `json:"place_of_birth"`
	DateOfIssue      string `json:"date_of_issue"`
	DateOfExpiry     string `json:"date_of_expiry"`
	IssuingAuthority string `json:"issuing_authority"`
	Error            string `json:"error,omitempty"`
}

// NewMexicoIdentityConnector creates a new Mexican identity connector
func NewMexicoIdentityConnector(config *MexicoConnectorConfig) (*MexicoIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &MexicoIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// ValidateCURP validates Mexican CURP (Clave Única de Registro de Población)
// Format: 18 characters (AAAA######HHHHHH##)
// Example: GOTJ901015HDFRRL09
func (mc *MexicoIdentityConnector) ValidateCURP(ctx context.Context, req *CURPRequest) (*CURPResponse, error) {
	// Validate request
	if err := mc.validator.Struct(req); err != nil {
		return &CURPResponse{Valid: false, Error: err.Error()}, nil
	}

	curp := strings.ToUpper(strings.TrimSpace(req.CURP))

	// Validate format (18 alphanumeric characters)
	if !regexp.MustCompile(`^[A-Z]{4}\d{6}[HM][A-Z]{5}[A-Z0-9]\d$`).MatchString(curp) {
		return &CURPResponse{
			Valid: false,
			Error: "Invalid CURP format",
		}, nil
	}

	// Extract components
	// Positions 0-3: First surname letter, first name letter, date
	year := curp[4:6]
	month := curp[6:8]
	day := curp[8:10]
	gender := string(curp[10]) // H or M
	state := curp[11:13]

	// Determine century
	yearInt, _ := strconv.Atoi(year)
	var century string
	if yearInt >= 0 && yearInt <= 25 {
		century = "20"
	} else {
		century = "19"
	}

	dateOfBirth := fmt.Sprintf("%s%s-%s-%s", century, year, month, day)

	// Validate check digit
	checkDigitValid := mc.validateCURPCheckDigit(curp)

	response := &CURPResponse{
		Valid:           checkDigitValid,
		CURP:            curp,
		Name:            req.Name,
		Gender:          gender,
		DateOfBirth:     dateOfBirth,
		StateOfBirth:    state,
		CheckDigitValid: checkDigitValid,
	}

	if !checkDigitValid {
		response.Error = "Invalid CURP check digit"
	}

	return response, nil
}

// ValidateRFC validates Mexican RFC (Registro Federal de Contribuyentes)
// Formats:
// - Person: 13 characters (AAAA######XXX)
// - Company: 12 characters (AAA######XXX)
func (mc *MexicoIdentityConnector) ValidateRFC(ctx context.Context, req *RFCRequest) (*RFCResponse, error) {
	// Validate request
	if err := mc.validator.Struct(req); err != nil {
		return &RFCResponse{Valid: false, Error: err.Error()}, nil
	}

	rfc := strings.ToUpper(strings.TrimSpace(req.RFC))

	// Determine RFC type and validate format
	var taxpayerType string
	var checkDigitValid bool

	switch len(rfc) {
	case 13:
		// Person RFC: 4 letters + 6 digits + 3 alphanumeric
		if !regexp.MustCompile(`^[A-Z&Ñ]{4}\d{6}[A-Z0-9]{3}$`).MatchString(rfc) {
			return &RFCResponse{
				Valid: false,
				Error: "Invalid person RFC format",
			}, nil
		}
		taxpayerType = "Person"
		checkDigitValid = mc.validateRFCCheckDigit(rfc)
	case 12:
		// Company RFC: 3 letters + 6 digits + 3 alphanumeric
		if !regexp.MustCompile(`^[A-Z&Ñ]{3}\d{6}[A-Z0-9]{3}$`).MatchString(rfc) {
			return &RFCResponse{
				Valid: false,
				Error: "Invalid company RFC format",
			}, nil
		}
		taxpayerType = "Company"
		checkDigitValid = mc.validateRFCCheckDigit(rfc)
	default:
		return &RFCResponse{
			Valid: false,
			Error: "Invalid RFC length (must be 12 or 13 characters)",
		}, nil
	}

	response := &RFCResponse{
		Valid:           checkDigitValid,
		RFC:             rfc,
		Name:            req.Name,
		TaxpayerType:    taxpayerType,
		Status:          "Active",
		CheckDigitValid: checkDigitValid,
	}

	if !checkDigitValid {
		response.Error = "Invalid RFC check digit"
	}

	return response, nil
}

//nolint:misspell // VerifyINE verifies Mexican INE (Instituto Nacional Electoral) voter ID
func (mc *MexicoIdentityConnector) VerifyINE(ctx context.Context, req *INERequest) (*INEResponse, error) {
	// Validate request
	if err := mc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate INE number format (13 digits)
	if !regexp.MustCompile(`^\d{13}$`).MatchString(req.INENumber) {
		return &INEResponse{
			Valid: false,
			Error: "Invalid INE number format (must be 13 digits)",
		}, nil
	}

	// Validate CIC format (9 digits)
	if !regexp.MustCompile(`^\d{9}$`).MatchString(req.CIC) {
		return &INEResponse{
			Valid: false,
			Error: "Invalid CIC format (must be 9 digits)",
		}, nil
	}

	// In production, this would verify with INE database
	response := &INEResponse{
		Valid:        true,
		INENumber:    req.INENumber,
		CIC:          req.CIC,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		DateOfBirth:  req.DateOfBirth,
		IssueDate:    "2020-01-15",
		ExpiryDate:   "2030-01-15",
		EmissionYear: "2020",
	}

	return response, nil
}

// VerifyPassport verifies Mexican passport
func (mc *MexicoIdentityConnector) VerifyPassport(ctx context.Context, req *MXPassportRequest) (*MXPassportResponse, error) {
	// Validate request
	if err := mc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	passportNumber := strings.ToUpper(strings.TrimSpace(req.PassportNumber))

	// Validate passport number format (letter + 8 digits)
	if !regexp.MustCompile(`^[A-Z]\d{8}$`).MatchString(passportNumber) {
		return &MXPassportResponse{
			Valid: false,
			Error: "Invalid passport number format",
		}, nil
	}

	// In production, this would verify with SRE (Secretaría de Relaciones Exteriores)
	response := &MXPassportResponse{
		Valid:            true,
		PassportNumber:   passportNumber,
		GivenNames:       req.FirstName,
		Surname:          req.LastName,
		DateOfBirth:      req.DateOfBirth,
		Nationality:      "MEX",
		DateOfIssue:      "2020-01-15",
		DateOfExpiry:     "2030-01-15",
		IssuingAuthority: "México",
	}

	return response, nil
}

// Helper methods

func (mc *MexicoIdentityConnector) validateCURPCheckDigit(curp string) bool {
	// CURP check digit calculation
	checkDigitMap := "0123456789ABCDEFGHIJKLMNÑOPQRSTUVWXYZ"
	sum := 0

	for i := 0; i < 17; i++ {
		char := curp[i]
		var value int
		if char >= '0' && char <= '9' {
			value = int(char - '0')
		} else {
			value = strings.IndexByte(checkDigitMap, char)
		}
		sum += value * (18 - i)
	}

	remainder := sum % 10
	expectedCheck := (10 - remainder) % 10

	actualCheck, _ := strconv.Atoi(string(curp[17]))

	return expectedCheck == actualCheck
}

func (mc *MexicoIdentityConnector) validateRFCCheckDigit(rfc string) bool {
	// RFC check digit calculation
	checkDigitMap := "0123456789ABCDEFGHIJKLMN&OPQRSTUVWXYZ Ñ"
	sum := 0

	// Get the base RFC (without check digit)
	baseRFC := rfc[:len(rfc)-1]

	for i, char := range baseRFC {
		value := strings.IndexRune(checkDigitMap, char)
		if value == -1 {
			return false
		}
		sum += value * (len(baseRFC) + 1 - i)
	}

	remainder := sum % 11
	var expectedCheck string
	if remainder == 0 {
		expectedCheck = "0"
	} else {
		checkValue := 11 - remainder
		if checkValue == 10 {
			expectedCheck = "A"
		} else {
			expectedCheck = strconv.Itoa(checkValue)
		}
	}

	actualCheck := string(rfc[len(rfc)-1])

	return expectedCheck == actualCheck
}

func (mc *MexicoIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

var _ = (*MexicoIdentityConnector).generateCacheKey

// GetMetrics returns connector metrics
func (mc *MexicoIdentityConnector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return map[string]interface{}{
		"connector": "mexico_identity",
	}
}

// Close closes the connector and cleans up resources
func (mc *MexicoIdentityConnector) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return nil
}
