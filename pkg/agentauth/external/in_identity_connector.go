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

// IndiaIdentityConnector handles Indian identity verification
// Supports Aadhaar, PAN, DigiLocker, e-KYC
type IndiaIdentityConnector struct {
	config     *IndiaConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// IndiaConnectorConfig configuration for Indian identity connector
type IndiaConnectorConfig struct {
	// Aadhaar configuration
	AadhaarAuthURL    string `validate:"required,url"`
	AadhaarAUA        string `validate:"required"` // Authentication User Agency code
	AadhaarASA        string `validate:"required"` // Authentication Service Agency code
	AadhaarLicenseKey string `validate:"required"`

	// DigiLocker configuration
	DigiLockerURL      string `validate:"url"`
	DigiLockerClientID string
	DigiLockerSecret   string

	// e-KYC configuration
	EKYCServiceURL string `validate:"url"`
	EKYCAPIKey     string

	// PAN verification
	PANVerificationURL string `validate:"url"`
	PANAPIKey          string

	// Timeouts
	RequestTimeout time.Duration
}

// AadhaarAuthRequest Aadhaar authentication request
type AadhaarAuthRequest struct {
	AadhaarNumber       string   `json:"aadhaar_number" validate:"required,len=12"`
	AuthType            string   `json:"auth_type" validate:"required,oneof=OTP Biometric Iris"`
	OTP                 string   `json:"otp,omitempty"`
	BiometricData       string   `json:"biometric_data,omitempty"` // Base64 encoded
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	Consent             bool     `json:"consent" validate:"required"`
}

// AadhaarAuthResponse Aadhaar authentication response
type AadhaarAuthResponse struct {
	Success       bool             `json:"success"`
	TransactionID string           `json:"transaction_id"`
	Timestamp     string           `json:"timestamp"`
	UserInfo      *AadhaarUserInfo `json:"user_info"`
	ErrorCode     string           `json:"error_code,omitempty"`
	Error         string           `json:"error,omitempty"`
}

// AadhaarUserInfo user information from Aadhaar
type AadhaarUserInfo struct {
	AadhaarNumber  string     `json:"aadhaar_number"` // Masked (XXXX-XXXX-1234)
	Name           string     `json:"name"`
	DateOfBirth    string     `json:"date_of_birth"`
	Gender         string     `json:"gender"` // M/F/O (Other)
	Email          string     `json:"email"`
	EmailVerified  bool       `json:"email_verified"`
	Mobile         string     `json:"mobile"`
	MobileVerified bool       `json:"mobile_verified"`
	Address        *INAddress `json:"address"`
	Photo          string     `json:"photo,omitempty"` // Base64 encoded
}

// INAddress Indian address structure
type INAddress struct {
	CareOf      string `json:"care_of,omitempty"` // c/o
	House       string `json:"house,omitempty"`
	Street      string `json:"street,omitempty"`
	Landmark    string `json:"landmark,omitempty"`
	Locality    string `json:"locality"`
	VTC         string `json:"vtc"` // Village/Town/City
	SubDistrict string `json:"sub_district,omitempty"`
	District    string `json:"district"`
	State       string `json:"state"`
	PostOffice  string `json:"post_office,omitempty"`
	Pincode     string `json:"pincode"`
	Country     string `json:"country"`
}

// PANRequest PAN card validation request
type PANRequest struct {
	PANNumber   string `json:"pan_number" validate:"required,len=10"`
	Name        string `json:"name"`
	DateOfBirth string `json:"date_of_birth"`
}

// PANResponse PAN card validation response
type PANResponse struct {
	Valid       bool   `json:"valid"`
	PANNumber   string `json:"pan_number"`
	Name        string `json:"name"`
	Category    string `json:"category"` // Individual, Company, HUF, etc.
	Status      string `json:"status"`   // Active, Inactive, etc.
	DateOfBirth string `json:"date_of_birth,omitempty"`
	Error       string `json:"error,omitempty"`
}

// DigiLockerAuthRequest DigiLocker authentication request
type DigiLockerAuthRequest struct {
	RequestedDocuments []string `json:"requested_documents" validate:"required,min=1"`
	Purpose            string   `json:"purpose" validate:"required"`
	ReturnURL          string   `json:"return_url" validate:"required,url"`
}

// DigiLockerAuthResponse DigiLocker authentication response
type DigiLockerAuthResponse struct {
	Success   bool                 `json:"success"`
	SessionID string               `json:"session_id"`
	UserInfo  *DigiLockerUserInfo  `json:"user_info"`
	Documents []DigiLockerDocument `json:"documents"`
	Error     string               `json:"error,omitempty"`
}

// DigiLockerUserInfo user information from DigiLocker
type DigiLockerUserInfo struct {
	DigiLockerID  string `json:"digilocker_id"`
	Name          string `json:"name"`
	DateOfBirth   string `json:"date_of_birth"`
	Gender        string `json:"gender"`
	AadhaarNumber string `json:"aadhaar_number"` // Masked
}

// DigiLockerDocument document from DigiLocker
type DigiLockerDocument struct {
	DocumentType string `json:"document_type"` // PAN, Aadhaar, DrivingLicense, etc.
	DocumentID   string `json:"document_id"`
	IssuerName   string `json:"issuer_name"`
	IssueDate    string `json:"issue_date"`
	ExpiryDate   string `json:"expiry_date,omitempty"`
	URI          string `json:"uri"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
}

// EKYCRequest e-KYC request
type EKYCRequest struct {
	AadhaarNumber string `json:"aadhaar_number" validate:"required,len=12"`
	OTP           string `json:"otp" validate:"required"`
	Consent       bool   `json:"consent" validate:"required"`
}

// EKYCResponse e-KYC response with full demographic and biometric data
type EKYCResponse struct {
	Success       bool             `json:"success"`
	TransactionID string           `json:"transaction_id"`
	KYCData       *AadhaarUserInfo `json:"kyc_data"`
	Error         string           `json:"error,omitempty"`
}

// NewIndiaIdentityConnector creates a new Indian identity connector
func NewIndiaIdentityConnector(config *IndiaConnectorConfig) (*IndiaIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &IndiaIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// AuthenticateAadhaar authenticates using Aadhaar
// Auth types: OTP, Biometric (fingerprint), Iris
func (ic *IndiaIdentityConnector) AuthenticateAadhaar(ctx context.Context, req *AadhaarAuthRequest) (*AadhaarAuthResponse, error) {
	// Validate request
	if err := ic.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate Aadhaar number format (12 digits)
	if !regexp.MustCompile(`^\d{12}$`).MatchString(req.AadhaarNumber) {
		return &AadhaarAuthResponse{
			Success: false,
			Error:   "Invalid Aadhaar number format (must be 12 digits)",
		}, nil
	}

	// Validate Aadhaar number check digit (Verhoeff algorithm)
	if !ic.validateAadhaarCheckDigit(req.AadhaarNumber) {
		return &AadhaarAuthResponse{
			Success: false,
			Error:   "Invalid Aadhaar number check digit",
		}, nil
	}

	// Validate consent
	if !req.Consent {
		return &AadhaarAuthResponse{
			Success: false,
			Error:   "User consent is required for Aadhaar authentication",
		}, nil
	}

	// Validate auth type specific data
	switch req.AuthType {
	case "OTP":
		if req.OTP == "" {
			return &AadhaarAuthResponse{
				Success: false,
				Error:   "OTP is required for OTP authentication",
			}, nil
		}
	case "Biometric", "Iris":
		if req.BiometricData == "" {
			return &AadhaarAuthResponse{
				Success: false,
				Error:   "Biometric data is required",
			}, nil
		}
	}

	// In production, this would:
	// 1. Connect to UIDAI (Unique Identification Authority of India)
	// 2. Perform authentication via CIDR (Central Identities Data Repository)
	// 3. Receive demographic and biometric verification result

	// Mask Aadhaar number for response
	maskedAadhaar := "XXXX-XXXX-" + req.AadhaarNumber[8:]

	// Mock response for demonstration
	response := &AadhaarAuthResponse{
		Success:       true,
		TransactionID: fmt.Sprintf("aadhaar_%d", time.Now().Unix()),
		Timestamp:     time.Now().Format(time.RFC3339),
		UserInfo: &AadhaarUserInfo{
			AadhaarNumber:  maskedAadhaar,
			Name:           "Rajesh Kumar",
			DateOfBirth:    "1990-01-15",
			Gender:         "M",
			Email:          "rajesh.kumar@example.in",
			EmailVerified:  true,
			Mobile:         "+91-9876543210",
			MobileVerified: true,
		},
	}

	return response, nil
}

// ValidatePAN validates Indian PAN (Permanent Account Number)
// Format: ABCDE1234F (5 letters + 4 digits + 1 letter)
func (ic *IndiaIdentityConnector) ValidatePAN(ctx context.Context, req *PANRequest) (*PANResponse, error) {
	// Validate request
	if err := ic.validator.Struct(req); err != nil {
		return &PANResponse{Valid: false, Error: err.Error()}, nil
	}

	pan := strings.ToUpper(strings.TrimSpace(req.PANNumber))

	// Validate format (5 letters + 4 digits + 1 letter)
	if !regexp.MustCompile(`^[A-Z]{5}\d{4}[A-Z]$`).MatchString(pan) {
		return &PANResponse{
			Valid: false,
			Error: "Invalid PAN format (must be 5 letters + 4 digits + 1 letter)",
		}, nil
	}

	// 4th character indicates PAN holder category
	category := ""
	switch pan[3] {
	case 'P':
		category = "Individual"
	case 'C':
		category = "Company"
	case 'H':
		category = "HUF (Hindu Undivided Family)"
	case 'A':
		category = "AOP (Association of Persons)"
	case 'B':
		category = "BOI (Body of Individuals)"
	case 'G':
		category = "Government"
	case 'J':
		category = "Artificial Juridical Person"
	case 'L':
		category = "Local Authority"
	case 'F':
		category = "Firm/Partnership"
	case 'T':
		category = "Trust"
	default:
		category = "Unknown"
	}

	// In production, this would verify with Income Tax Department
	response := &PANResponse{
		Valid:       true,
		PANNumber:   pan,
		Name:        req.Name,
		Category:    category,
		Status:      "Active",
		DateOfBirth: req.DateOfBirth,
	}

	return response, nil
}

// AuthenticateDigiLocker authenticates using DigiLocker
func (ic *IndiaIdentityConnector) AuthenticateDigiLocker(ctx context.Context, req *DigiLockerAuthRequest) (*DigiLockerAuthResponse, error) {
	// Validate request
	if err := ic.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// In production, this would:
	// 1. Redirect to DigiLocker OAuth
	// 2. User authorizes document access
	// 3. Retrieve requested documents
	// 4. Return document URIs and metadata

	// Mock response for demonstration
	response := &DigiLockerAuthResponse{
		Success:   true,
		SessionID: fmt.Sprintf("digilocker_%d", time.Now().Unix()),
		UserInfo: &DigiLockerUserInfo{
			DigiLockerID:  fmt.Sprintf("DL%d", time.Now().Unix()),
			Name:          "Rajesh Kumar",
			DateOfBirth:   "1990-01-15",
			Gender:        "M",
			AadhaarNumber: "XXXX-XXXX-1234",
		},
		Documents: []DigiLockerDocument{
			{
				DocumentType: "PAN",
				DocumentID:   "ABCDE1234F",
				IssuerName:   "Income Tax Department",
				IssueDate:    "2015-01-01",
				URI:          "digilocker://documents/pan/ABCDE1234F",
				MimeType:     "application/pdf",
				Size:         102400,
			},
		},
	}

	return response, nil
}

// PerformEKYC performs e-KYC using Aadhaar OTP
func (ic *IndiaIdentityConnector) PerformEKYC(ctx context.Context, req *EKYCRequest) (*EKYCResponse, error) {
	// Validate request
	if err := ic.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate Aadhaar number
	if !regexp.MustCompile(`^\d{12}$`).MatchString(req.AadhaarNumber) {
		return &EKYCResponse{
			Success: false,
			Error:   "Invalid Aadhaar number format",
		}, nil
	}

	// Validate consent
	if !req.Consent {
		return &EKYCResponse{
			Success: false,
			Error:   "User consent is required for e-KYC",
		}, nil
	}

	// In production, this would:
	// 1. Verify OTP with UIDAI
	// 2. Retrieve full KYC data (demographics + address + photo)
	// 3. Return digitally signed e-KYC XML

	// Mask Aadhaar number
	maskedAadhaar := "XXXX-XXXX-" + req.AadhaarNumber[8:]

	// Mock response for demonstration
	response := &EKYCResponse{
		Success:       true,
		TransactionID: fmt.Sprintf("ekyc_%d", time.Now().Unix()),
		KYCData: &AadhaarUserInfo{
			AadhaarNumber:  maskedAadhaar,
			Name:           "Rajesh Kumar",
			DateOfBirth:    "1990-01-15",
			Gender:         "M",
			Email:          "rajesh.kumar@example.in",
			EmailVerified:  true,
			Mobile:         "+91-9876543210",
			MobileVerified: true,
			Address: &INAddress{
				House:    "123",
				Street:   "MG Road",
				Locality: "Koramangala",
				VTC:      "Bangalore",
				District: "Bangalore Urban",
				State:    "Karnataka",
				Pincode:  "560034",
				Country:  "India",
			},
		},
	}

	return response, nil
}

// Helper methods

func (ic *IndiaIdentityConnector) validateAadhaarCheckDigit(aadhaar string) bool {
	// Aadhaar uses Verhoeff algorithm for check digit
	// This is a simplified validation - production should use full Verhoeff

	// Verhoeff multiplication table
	d := [][]int{
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		{1, 2, 3, 4, 0, 6, 7, 8, 9, 5},
		{2, 3, 4, 0, 1, 7, 8, 9, 5, 6},
		{3, 4, 0, 1, 2, 8, 9, 5, 6, 7},
		{4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
		{5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
		{6, 5, 9, 8, 7, 1, 0, 4, 3, 2},
		{7, 6, 5, 9, 8, 2, 1, 0, 4, 3},
		{8, 7, 6, 5, 9, 3, 2, 1, 0, 4},
		{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
	}

	// Permutation table
	p := [][]int{
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		{1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
		{5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
		{8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
		{9, 4, 5, 3, 1, 2, 6, 8, 7, 0},
		{4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
		{2, 7, 9, 3, 8, 0, 6, 4, 1, 5},
		{7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
	}

	c := 0
	for i := 0; i < len(aadhaar); i++ {
		digit := int(aadhaar[len(aadhaar)-1-i] - '0')
		c = d[c][p[i%8][digit]]
	}

	return c == 0
}

func (ic *IndiaIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (ic *IndiaIdentityConnector) GetMetrics() map[string]interface{} {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	return map[string]interface{}{
		"connector": "india_identity",
	}
}

// Close closes the connector and cleans up resources
func (ic *IndiaIdentityConnector) Close() error {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	return nil
}
