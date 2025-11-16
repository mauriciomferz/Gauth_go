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

// SaudiIdentityConnector handles Saudi Arabia identity verification
// Supports Absher, Iqama, Muqeem, MOCI authentication
type SaudiIdentityConnector struct {
	config     *SaudiConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// SaudiConnectorConfig configuration for Saudi Arabia identity connector
type SaudiConnectorConfig struct {
	// Absher platform configuration
	AbsherURL          string `validate:"required,url"`
	AbsherClientID     string `validate:"required"`
	AbsherClientSecret string `validate:"required"`
	
	// Muqeem (expatriate services) configuration
	MuqeemURL          string `validate:"url"`
	MuqeemAPIKey       string
	
	// MOCI (Ministry of Commerce) configuration
	MOCIURL            string `validate:"url"`
	MOCIAPIKey         string
	
	// Timeouts
	RequestTimeout     time.Duration
}

// AbsherAuthRequest Absher authentication request
type AbsherAuthRequest struct {
	ServiceID          string   `json:"service_id" validate:"required"`
	IDType             string   `json:"id_type" validate:"required,oneof=national iqama"` // National ID (citizens) or Iqama (residents)
	IDNumber           string   `json:"id_number" validate:"required"`
	DateOfBirth        string   `json:"date_of_birth" validate:"required"`
	RequestedServices  []string `json:"requested_services" validate:"required,min=1"`
	RedirectURL        string   `json:"redirect_url" validate:"required,url"`
}

// AbsherAuthResponse Absher authentication response
type AbsherAuthResponse struct {
	Success            bool              `json:"success"`
	SessionID          string            `json:"session_id"`
	AccessToken        string            `json:"access_token"`
	TokenType          string            `json:"token_type"`
	ExpiresIn          int               `json:"expires_in"`
	UserInfo           *AbsherUserInfo   `json:"user_info"`
	AvailableServices  []string          `json:"available_services"`
	Error              string            `json:"error,omitempty"`
}

// AbsherUserInfo user information from Absher
type AbsherUserInfo struct {
	ID                 string          `json:"id"` // Unique identifier
	IDType             string          `json:"id_type"` // national, iqama
	IDNumber           string          `json:"id_number"`
	FullName           string          `json:"full_name"`
	FullNameArabic     string          `json:"full_name_arabic"`
	FirstName          string          `json:"first_name"`
	FatherName         string          `json:"father_name"`
	GrandfatherName    string          `json:"grandfather_name"`
	FamilyName         string          `json:"family_name"`
	DateOfBirth        string          `json:"date_of_birth"`
	PlaceOfBirth       string          `json:"place_of_birth"`
	Gender             string          `json:"gender"`
	Nationality        string          `json:"nationality"`
	MaritalStatus      string          `json:"marital_status"`
	Religion           string          `json:"religion"`
	Email              string          `json:"email"`
	Mobile             string          `json:"mobile"`
	Address            *SaudiAddress   `json:"address"`
}

// SaudiAddress Saudi Arabia address structure
type SaudiAddress struct {
	District           string `json:"district"`
	Street             string `json:"street"`
	BuildingNumber     string `json:"building_number"`
	AdditionalNumber   string `json:"additional_number"`
	PostalCode         string `json:"postal_code"`
	City               string `json:"city"`
	Region             string `json:"region"`
	Country            string `json:"country"`
}

// IqamaRequest Iqama (residence permit) verification request
type IqamaRequest struct {
	IqamaNumber        string `json:"iqama_number" validate:"required"`
	DateOfBirth        string `json:"date_of_birth"`
	Nationality        string `json:"nationality"`
}

// IqamaResponse Iqama verification response
type IqamaResponse struct {
	Valid              bool            `json:"valid"`
	IqamaNumber        string          `json:"iqama_number"`
	IssueDate          string          `json:"issue_date"`
	ExpiryDate         string          `json:"expiry_date"`
	Status             string          `json:"status"` // active, expired, cancelled
	Profession         string          `json:"profession"`
	Sponsor            string          `json:"sponsor"` // Kafeel
	SponsorID          string          `json:"sponsor_id"`
	BorderNumber       string          `json:"border_number"`
	UserInfo           *IqamaUserInfo  `json:"user_info"`
	Error              string          `json:"error,omitempty"`
}

// IqamaUserInfo user information from Iqama
type IqamaUserInfo struct {
	IqamaNumber        string `json:"iqama_number"`
	FullName           string `json:"full_name"`
	FullNameArabic     string `json:"full_name_arabic"`
	DateOfBirth        string `json:"date_of_birth"`
	Gender             string `json:"gender"`
	Nationality        string `json:"nationality"`
	Religion           string `json:"religion"`
	MaritalStatus      string `json:"marital_status"`
	Profession         string `json:"profession"`
}

// MuqeemRequest Muqeem verification request
type MuqeemRequest struct {
	ServiceType        string `json:"service_type" validate:"required,oneof=status transfer exit_reentry final_exit"`
	IqamaNumber        string `json:"iqama_number" validate:"required"`
	SponsorID          string `json:"sponsor_id"`
}

// MuqeemResponse Muqeem verification response
type MuqeemResponse struct {
	Success            bool              `json:"success"`
	ServiceType        string            `json:"service_type"`
	IqamaNumber        string            `json:"iqama_number"`
	Status             string            `json:"status"`
	PermissionStatus   string            `json:"permission_status"`
	ValidUntil         string            `json:"valid_until"`
	Details            map[string]string `json:"details"`
	Error              string            `json:"error,omitempty"`
}

// MOCIAuthRequest MOCI authentication request
type MOCIAuthRequest struct {
	ServiceCode        string `json:"service_code" validate:"required"`
	CommercialRegNo    string `json:"commercial_reg_no"`
	IDNumber           string `json:"id_number"`
}

// MOCIAuthResponse MOCI authentication response
type MOCIAuthResponse struct {
	Success            bool              `json:"success"`
	SessionID          string            `json:"session_id"`
	ServiceCode        string            `json:"service_code"`
	CompanyInfo        *MOCICompanyInfo  `json:"company_info"`
	Error              string            `json:"error,omitempty"`
}

// MOCICompanyInfo company information from MOCI
type MOCICompanyInfo struct {
	CommercialRegNo    string   `json:"commercial_reg_no"`
	CompanyName        string   `json:"company_name"`
	CompanyNameArabic  string   `json:"company_name_arabic"`
	LegalForm          string   `json:"legal_form"`
	IssueDate          string   `json:"issue_date"`
	ExpiryDate         string   `json:"expiry_date"`
	Status             string   `json:"status"`
	City               string   `json:"city"`
	Activities         []string `json:"activities"`
}

// NewSaudiIdentityConnector creates a new Saudi Arabia identity connector
func NewSaudiIdentityConnector(config *SaudiConnectorConfig) (*SaudiIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	
	connector := &SaudiIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}
	
	return connector, nil
}

// AuthenticateAbsher authenticates using Absher platform
func (sc *SaudiIdentityConnector) AuthenticateAbsher(ctx context.Context, req *AbsherAuthRequest) (*AbsherAuthResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate ID format
	var valid bool
	if req.IDType == "national" {
		valid = sc.validateNationalIDFormat(req.IDNumber)
	} else if req.IDType == "iqama" {
		valid = sc.validateIqamaFormat(req.IDNumber)
	}
	
	if !valid {
		return &AbsherAuthResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid %s ID format", req.IDType),
		}, nil
	}
	
	// In production, this would:
	// 1. Connect to Absher OAuth2 API
	// 2. Authenticate user credentials
	// 3. Request user permissions for services
	// 4. Return access token and user info
	
	// Mock response for demonstration
	response := &AbsherAuthResponse{
		Success:   true,
		SessionID: fmt.Sprintf("absher_%d", time.Now().Unix()),
		AccessToken: fmt.Sprintf("token_%d", time.Now().Unix()),
		TokenType: "Bearer",
		ExpiresIn: 3600,
		UserInfo: &AbsherUserInfo{
			ID:             fmt.Sprintf("SA_%d", time.Now().Unix()),
			IDType:         req.IDType,
			IDNumber:       req.IDNumber,
			FullName:       "Abdullah Mohammed",
			FullNameArabic: "عبدالله محمد",
			FirstName:      "Abdullah",
			FatherName:     "Mohammed",
			DateOfBirth:    "1985-05-20",
			Gender:         "M",
			Nationality:    "SAU",
			Email:          "abdullah.mohammed@example.sa",
			Mobile:         "+966501234567",
		},
		AvailableServices: req.RequestedServices,
	}
	
	return response, nil
}

// VerifyIqama verifies Iqama (residence permit)
// Format: 10 digits, starting with 1 or 2
// - 1xxxxxxxxx: Individual Iqama
// - 2xxxxxxxxx: Family Iqama
func (sc *SaudiIdentityConnector) VerifyIqama(ctx context.Context, req *IqamaRequest) (*IqamaResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate Iqama format
	if !sc.validateIqamaFormat(req.IqamaNumber) {
		return &IqamaResponse{
			Valid: false,
			Error: "Invalid Iqama number format (must be 10 digits starting with 1 or 2)",
		}, nil
	}
	
	// In production, this would:
	// 1. Call Ministry of Interior (MOI) API
	// 2. Verify Iqama status
	// 3. Check expiry date
	// 4. Verify sponsor information
	// 5. Return detailed Iqama info
	
	// Mock response for demonstration
	response := &IqamaResponse{
		Valid:        true,
		IqamaNumber:  req.IqamaNumber,
		IssueDate:    "2020-01-15",
		ExpiryDate:   "2025-01-15",
		Status:       "active",
		Profession:   "Engineer",
		Sponsor:      "ABC Company",
		SponsorID:    "7001234567",
		BorderNumber: "B12345678",
		UserInfo: &IqamaUserInfo{
			IqamaNumber:    req.IqamaNumber,
			FullName:       "Ahmed Hassan",
			FullNameArabic: "أحمد حسن",
			DateOfBirth:    "1990-03-25",
			Gender:         "M",
			Nationality:    "EGY",
			Profession:     "Engineer",
		},
	}
	
	return response, nil
}

// CheckMuqeemStatus checks Muqeem (expatriate services) status
func (sc *SaudiIdentityConnector) CheckMuqeemStatus(ctx context.Context, req *MuqeemRequest) (*MuqeemResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Verify Iqama first
	iqamaReq := &IqamaRequest{
		IqamaNumber: req.IqamaNumber,
	}
	iqamaResp, err := sc.VerifyIqama(ctx, iqamaReq)
	if err != nil || !iqamaResp.Valid {
		return &MuqeemResponse{
			Success: false,
			Error:   "Invalid Iqama number",
		}, nil
	}
	
	// In production, this would:
	// 1. Connect to Muqeem platform
	// 2. Check service status (transfer, exit-reentry, final exit)
	// 3. Verify sponsor permissions
	// 4. Return service status
	
	// Mock response for demonstration
	response := &MuqeemResponse{
		Success:          true,
		ServiceType:      req.ServiceType,
		IqamaNumber:      req.IqamaNumber,
		Status:           "active",
		PermissionStatus: "approved",
		ValidUntil:       "2025-06-30",
		Details: map[string]string{
			"service":      req.ServiceType,
			"sponsor":      "ABC Company",
			"border_count": "2",
		},
	}
	
	return response, nil
}

// AuthenticateMOCI authenticates with MOCI (Ministry of Commerce)
func (sc *SaudiIdentityConnector) AuthenticateMOCI(ctx context.Context, req *MOCIAuthRequest) (*MOCIAuthResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// In production, this would:
	// 1. Connect to MOCI system
	// 2. Verify commercial registration
	// 3. Check company status
	// 4. Return company details
	
	// Mock response for demonstration
	response := &MOCIAuthResponse{
		Success:     true,
		SessionID:   fmt.Sprintf("moci_%d", time.Now().Unix()),
		ServiceCode: req.ServiceCode,
		CompanyInfo: &MOCICompanyInfo{
			CommercialRegNo:   req.CommercialRegNo,
			CompanyName:       "Saudi Tech Solutions",
			CompanyNameArabic: "الحلول التقنية السعودية",
			LegalForm:         "LLC",
			IssueDate:         "2015-06-10",
			ExpiryDate:        "2025-06-10",
			Status:            "active",
			City:              "Riyadh",
			Activities: []string{
				"Information Technology",
				"Software Development",
			},
		},
	}
	
	return response, nil
}

// Helper methods

func (sc *SaudiIdentityConnector) validateNationalIDFormat(idNumber string) bool {
	// Saudi National ID: 10 digits, starting with 1
	return regexp.MustCompile(`^1\d{9}$`).MatchString(idNumber)
}

func (sc *SaudiIdentityConnector) validateIqamaFormat(iqamaNumber string) bool {
	// Iqama: 10 digits, starting with 1 or 2
	return regexp.MustCompile(`^[12]\d{9}$`).MatchString(iqamaNumber)
}

func (sc *SaudiIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (sc *SaudiIdentityConnector) GetMetrics() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	return map[string]interface{}{
		"connector": "saudi_identity",
	}
}

// Close closes the connector and cleans up resources
func (sc *SaudiIdentityConnector) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return nil
}
