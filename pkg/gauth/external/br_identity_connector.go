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

// BrazilIdentityConnector handles Brazilian identity verification
// Supports CPF, CNH, e-CPF, Gov.br
type BrazilIdentityConnector struct {
	config     *BrazilConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// BrazilConnectorConfig configuration for Brazilian identity connector
type BrazilConnectorConfig struct {
	// Gov.br configuration
	GovBrURL          string `validate:"required,url"`
	GovBrClientID     string `validate:"required"`
	GovBrSecret       string `validate:"required"`
	
	// Receita Federal (Federal Revenue) configuration
	ReceitaURL        string `validate:"url"`
	ReceitaAPIKey     string
	
	// DETRAN (DMV) configuration
	DETRANURL         string `validate:"url"`
	DETRANAPIKey      string
	
	// Timeouts
	RequestTimeout    time.Duration
}

// GovBrAuthRequest Gov.br authentication request
type GovBrAuthRequest struct {
	ServiceID           string   `json:"service_id" validate:"required"`
	TrustLevel          string   `json:"trust_level" validate:"required,oneof=1 2 3"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
}

// GovBrAuthResponse Gov.br authentication response
type GovBrAuthResponse struct {
	Success    bool              `json:"success"`
	SessionID  string            `json:"session_id"`
	TrustLevel string            `json:"trust_level"`
	UserInfo   *GovBrUserInfo    `json:"user_info"`
	Attributes map[string]string `json:"attributes"`
	Error      string            `json:"error,omitempty"`
}

// GovBrUserInfo user information from Gov.br
type GovBrUserInfo struct {
	Sub             string      `json:"sub"` // Subject identifier
	CPF             string      `json:"cpf"` // Masked
	Name            string      `json:"name"`
	SocialName      string      `json:"social_name,omitempty"` // Nome social
	DateOfBirth     string      `json:"date_of_birth"`
	PhoneNumber     string      `json:"phone_number"`
	PhoneVerified   bool        `json:"phone_verified"`
	Email           string      `json:"email"`
	EmailVerified   bool        `json:"email_verified"`
	Address         *BRAddress  `json:"address,omitempty"`
	Picture         string      `json:"picture,omitempty"`
}

// BRAddress Brazilian address structure
type BRAddress struct {
	Logradouro   string `json:"logradouro"` // Street
	Numero       string `json:"numero"` // Number
	Complemento  string `json:"complemento,omitempty"` // Complement
	Bairro       string `json:"bairro"` // Neighborhood
	Municipio    string `json:"municipio"` // City
	UF           string `json:"uf"` // State (2 letters)
	CEP          string `json:"cep"` // Postal code (12345-678)
	Pais         string `json:"pais"` // Country
}

// CPFRequest CPF validation request
type CPFRequest struct {
	CPF         string `json:"cpf" validate:"required"`
	Name        string `json:"name"`
	DateOfBirth string `json:"date_of_birth"`
}

// CPFResponse CPF validation response
type CPFResponse struct {
	Valid            bool   `json:"valid"`
	CPF              string `json:"cpf"` // Formatted: 123.456.789-01
	Name             string `json:"name,omitempty"`
	Status           string `json:"status"` // Regular, Suspended, Canceled, Null
	CheckDigitsValid bool   `json:"check_digits_valid"`
	Error            string `json:"error,omitempty"`
}

// CNHRequest CNH (driver's license) validation request
type CNHRequest struct {
	CNHNumber       string `json:"cnh_number" validate:"required,len=11"`
	RegistrationNumber string `json:"registration_number" validate:"required"` // Número de registro
	SecurityCode    string `json:"security_code"` // Código de segurança
	Name            string `json:"name" validate:"required"`
	DateOfBirth     string `json:"date_of_birth" validate:"required"`
}

// CNHResponse CNH validation response
type CNHResponse struct {
	Valid              bool       `json:"valid"`
	CNHNumber          string     `json:"cnh_number"`
	RegistrationNumber string     `json:"registration_number"`
	Name               string     `json:"name"`
	DateOfBirth        string     `json:"date_of_birth"`
	CPF                string     `json:"cpf,omitempty"`
	Category           string     `json:"category"` // A, B, AB, C, D, E
	IssueDate          string     `json:"issue_date"`
	ExpiryDate         string     `json:"expiry_date"`
	FirstLicenseDate   string     `json:"first_license_date"`
	IssuingState       string     `json:"issuing_state"` // UF
	Status             string     `json:"status"` // Active, Suspended, Expired
	Points             int        `json:"points,omitempty"` // Demerit points
	Address            *BRAddress `json:"address,omitempty"`
	Error              string     `json:"error,omitempty"`
}

// ECPFRequest e-CPF (digital certificate) request
type ECPFRequest struct {
	Certificate     string `json:"certificate" validate:"required"` // Base64 encoded
	Password        string `json:"password" validate:"required"`
	ValidateChain   bool   `json:"validate_chain"`
}

// ECPFResponse e-CPF validation response
type ECPFResponse struct {
	Valid            bool              `json:"valid"`
	CPF              string            `json:"cpf"`
	Name             string            `json:"name"`
	Email            string            `json:"email,omitempty"`
	SerialNumber     string            `json:"serial_number"`
	IssuerDN         string            `json:"issuer_dn"`
	SubjectDN        string            `json:"subject_dn"`
	NotBefore        string            `json:"not_before"`
	NotAfter         string            `json:"not_after"`
	CertificateType  string            `json:"certificate_type"` // A1, A3
	CertificateChain []string          `json:"certificate_chain,omitempty"`
	Error            string            `json:"error,omitempty"`
}

// NewBrazilIdentityConnector creates a new Brazilian identity connector
func NewBrazilIdentityConnector(config *BrazilConnectorConfig) (*BrazilIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	
	connector := &BrazilIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}
	
	return connector, nil
}

// AuthenticateGovBr authenticates using Gov.br
// Trust levels: 1 (bronze), 2 (silver), 3 (gold)
func (bc *BrazilIdentityConnector) AuthenticateGovBr(ctx context.Context, req *GovBrAuthRequest) (*GovBrAuthResponse, error) {
	// Validate request
	if err := bc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate trust level
	if req.TrustLevel != "1" && req.TrustLevel != "2" && req.TrustLevel != "3" {
		return &GovBrAuthResponse{
			Success: false,
			Error:   "Invalid trust level (must be 1, 2, or 3)",
		}, nil
	}
	
	// In production, this would:
	// 1. Redirect to Gov.br OAuth 2.0 / OIDC
	// 2. User authenticates with appropriate method
	// 3. Receive user claims at requested trust level
	
	// Mock response for demonstration
	response := &GovBrAuthResponse{
		Success:    true,
		SessionID:  fmt.Sprintf("govbr_%d", time.Now().Unix()),
		TrustLevel: req.TrustLevel,
		UserInfo: &GovBrUserInfo{
			Sub:           fmt.Sprintf("sub_%d", time.Now().Unix()),
			CPF:           "***.456.789-**",
			Name:          "João da Silva",
			DateOfBirth:   "1990-01-15",
			Email:         "joao.silva@example.com.br",
			EmailVerified: true,
		},
		Attributes: make(map[string]string),
	}
	
	return response, nil
}

// ValidateCPF validates Brazilian CPF (Cadastro de Pessoas Físicas)
// Format: 11 digits (123.456.789-01)
func (bc *BrazilIdentityConnector) ValidateCPF(ctx context.Context, req *CPFRequest) (*CPFResponse, error) {
	// Validate request
	if err := bc.validator.Struct(req); err != nil {
		return &CPFResponse{Valid: false, Error: err.Error()}, nil
	}
	
	// Remove formatting
	cpf := strings.ReplaceAll(req.CPF, ".", "")
	cpf = strings.ReplaceAll(cpf, "-", "")
	cpf = strings.TrimSpace(cpf)
	
	// Validate format (11 digits)
	if !regexp.MustCompile(`^\d{11}$`).MatchString(cpf) {
		return &CPFResponse{
			Valid: false,
			Error: "Invalid CPF format (must be 11 digits)",
		}, nil
	}
	
	// Check for known invalid CPFs (all same digit)
	if regexp.MustCompile(`^(\d)\1{10}$`).MatchString(cpf) {
		return &CPFResponse{
			Valid: false,
			Error: "Invalid CPF (all digits are the same)",
		}, nil
	}
	
	// Validate check digits
	checkDigitsValid := bc.validateCPFCheckDigits(cpf)
	
	// Format CPF
	formattedCPF := fmt.Sprintf("%s.%s.%s-%s", cpf[0:3], cpf[3:6], cpf[6:9], cpf[9:11])
	
	response := &CPFResponse{
		Valid:            checkDigitsValid,
		CPF:              formattedCPF,
		Name:             req.Name,
		Status:           "Regular",
		CheckDigitsValid: checkDigitsValid,
	}
	
	if !checkDigitsValid {
		response.Error = "Invalid CPF check digits"
	}
	
	return response, nil
}

// VerifyCNH verifies Brazilian CNH (Carteira Nacional de Habilitação)
func (bc *BrazilIdentityConnector) VerifyCNH(ctx context.Context, req *CNHRequest) (*CNHResponse, error) {
	// Validate request
	if err := bc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	cnhNumber := strings.TrimSpace(req.CNHNumber)
	
	// Validate format (11 digits)
	if !regexp.MustCompile(`^\d{11}$`).MatchString(cnhNumber) {
		return &CNHResponse{
			Valid: false,
			Error: "Invalid CNH number format (must be 11 digits)",
		}, nil
	}
	
	// In production, this would:
	// 1. Verify with DENATRAN (National Traffic Department)
	// 2. Check DETRAN state database
	// 3. Validate security code
	// 4. Check demerit points
	
	// Mock response for demonstration
	response := &CNHResponse{
		Valid:              true,
		CNHNumber:          cnhNumber,
		RegistrationNumber: req.RegistrationNumber,
		Name:               req.Name,
		DateOfBirth:        req.DateOfBirth,
		Category:           "AB", // Car and motorcycle
		IssueDate:          "2020-01-15",
		ExpiryDate:         "2030-01-15",
		FirstLicenseDate:   "2010-01-15",
		IssuingState:       "SP",
		Status:             "Active",
		Points:             0,
	}
	
	return response, nil
}

// VerifyECPF verifies e-CPF digital certificate
func (bc *BrazilIdentityConnector) VerifyECPF(ctx context.Context, req *ECPFRequest) (*ECPFResponse, error) {
	// Validate request
	if err := bc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// In production, this would:
	// 1. Decode certificate from Base64
	// 2. Verify password
	// 3. Validate certificate chain (ICP-Brasil)
	// 4. Check revocation status (LCR/OCSP)
	// 5. Extract CPF from certificate
	
	// Mock response for demonstration
	response := &ECPFResponse{
		Valid:           true,
		CPF:             "123.456.789-01",
		Name:            "João da Silva",
		SerialNumber:    "1234567890ABCDEF",
		IssuerDN:        "CN=AC Certisign RFB G5,O=Certisign Certificadora Digital,C=BR",
		SubjectDN:       "CN=João da Silva:12345678901,O=ICP-Brasil,C=BR",
		NotBefore:       "2020-01-01T00:00:00Z",
		NotAfter:        "2023-01-01T00:00:00Z",
		CertificateType: "A3",
	}
	
	return response, nil
}

// Helper methods

func (bc *BrazilIdentityConnector) validateCPFCheckDigits(cpf string) bool {
	// Calculate first check digit
	sum := 0
	for i := 0; i < 9; i++ {
		digit, _ := strconv.Atoi(string(cpf[i]))
		sum += digit * (10 - i)
	}
	remainder := sum % 11
	var checkDigit1 int
	if remainder < 2 {
		checkDigit1 = 0
	} else {
		checkDigit1 = 11 - remainder
	}
	
	actualCheckDigit1, _ := strconv.Atoi(string(cpf[9]))
	if checkDigit1 != actualCheckDigit1 {
		return false
	}
	
	// Calculate second check digit
	sum = 0
	for i := 0; i < 10; i++ {
		digit, _ := strconv.Atoi(string(cpf[i]))
		sum += digit * (11 - i)
	}
	remainder = sum % 11
	var checkDigit2 int
	if remainder < 2 {
		checkDigit2 = 0
	} else {
		checkDigit2 = 11 - remainder
	}
	
	actualCheckDigit2, _ := strconv.Atoi(string(cpf[10]))
	return checkDigit2 == actualCheckDigit2
}

func (bc *BrazilIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (bc *BrazilIdentityConnector) GetMetrics() map[string]interface{} {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	return map[string]interface{}{
		"connector": "brazil_identity",
	}
}

// Close closes the connector and cleans up resources
func (bc *BrazilIdentityConnector) Close() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return nil
}
