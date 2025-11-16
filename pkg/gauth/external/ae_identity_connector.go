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

// UAEIdentityConnector handles UAE identity verification
// Supports UAE Pass, Emirates ID, MOFA authentication
type UAEIdentityConnector struct {
	config     *UAEConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// UAEConnectorConfig configuration for UAE identity connector
type UAEConnectorConfig struct {
	// UAE Pass configuration
	UAEPassURL         string `validate:"required,url"`
	UAEPassClientID    string `validate:"required"`
	UAEPassClientSecret string `validate:"required"`
	
	// Emirates ID verification
	EmiratesIDURL      string `validate:"url"`
	EmiratesIDAPIKey   string
	
	// MOFA (Ministry of Foreign Affairs) configuration
	MOFAURL            string `validate:"url"`
	MOFAAPIKey         string
	
	// Timeouts
	RequestTimeout     time.Duration
}

// UAEPassAuthRequest UAE Pass authentication request
type UAEPassAuthRequest struct {
	ServiceID          string   `json:"service_id" validate:"required"`
	AssuranceLevel     string   `json:"assurance_level" validate:"required,oneof=low substantial high"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	RedirectURI        string   `json:"redirect_uri" validate:"required,url"`
	State              string   `json:"state" validate:"required"`
	UseOAuth2          bool     `json:"use_oauth2"`
}

// UAEPassAuthResponse UAE Pass authentication response
type UAEPassAuthResponse struct {
	Success            bool              `json:"success"`
	AccessToken        string            `json:"access_token"`
	TokenType          string            `json:"token_type"`
	ExpiresIn          int               `json:"expires_in"`
	UserInfo           *UAEPassUserInfo  `json:"user_info"`
	AssuranceLevel     string            `json:"assurance_level"`
	Error              string            `json:"error,omitempty"`
}

// UAEPassUserInfo user information from UAE Pass
type UAEPassUserInfo struct {
	UUID               string     `json:"uuid"` // Unique identifier
	EmiratesID         string     `json:"emirates_id"`
	FullName           string     `json:"full_name"`
	FullNameArabic     string     `json:"full_name_arabic"`
	FirstName          string     `json:"first_name"`
	FirstNameArabic    string     `json:"first_name_arabic"`
	LastName           string     `json:"last_name"`
	LastNameArabic     string     `json:"last_name_arabic"`
	DateOfBirth        string     `json:"date_of_birth"`
	Gender             string     `json:"gender"`
	Nationality        string     `json:"nationality"`
	Email              string     `json:"email"`
	Mobile             string     `json:"mobile"`
	IDCardNumber       string     `json:"id_card_number"`
	IDExpiryDate       string     `json:"id_expiry_date"`
	Address            *UAEAddress `json:"address"`
}

// UAEAddress UAE address structure
type UAEAddress struct {
	AddressLine1       string `json:"address_line1"`
	AddressLine2       string `json:"address_line2"`
	City               string `json:"city"`
	Emirate            string `json:"emirate"`
	POBox              string `json:"po_box"`
	Country            string `json:"country"`
}

// EmiratesIDRequest Emirates ID verification request
type EmiratesIDRequest struct {
	EmiratesID         string `json:"emirates_id" validate:"required"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	DateOfBirth        string `json:"date_of_birth"`
	UseNFC             bool   `json:"use_nfc"`
}

// EmiratesIDResponse Emirates ID verification response
type EmiratesIDResponse struct {
	Valid              bool              `json:"valid"`
	EmiratesID         string            `json:"emirates_id"`
	IDCardNumber       string            `json:"id_card_number"`
	CardType           string            `json:"card_type"` // citizen, resident
	IssueDate          string            `json:"issue_date"`
	ExpiryDate         string            `json:"expiry_date"`
	Status             string            `json:"status"` // active, expired, cancelled
	UserInfo           *EmiratesIDInfo   `json:"user_info"`
	Error              string            `json:"error,omitempty"`
}

// EmiratesIDInfo user information from Emirates ID
type EmiratesIDInfo struct {
	EmiratesID         string `json:"emirates_id"`
	FullName           string `json:"full_name"`
	FullNameArabic     string `json:"full_name_arabic"`
	DateOfBirth        string `json:"date_of_birth"`
	Gender             string `json:"gender"`
	Nationality        string `json:"nationality"`
	PlaceOfBirth       string `json:"place_of_birth"`
	Emirate            string `json:"emirate"`
	CardType           string `json:"card_type"`
}

// MOFAAuthRequest MOFA authentication request
type MOFAAuthRequest struct {
	ServiceCode        string `json:"service_code" validate:"required"`
	EmiratesID         string `json:"emirates_id" validate:"required"`
	ServiceType        string `json:"service_type" validate:"required,oneof=attestation legalization authentication"`
}

// MOFAAuthResponse MOFA authentication response
type MOFAAuthResponse struct {
	Success            bool              `json:"success"`
	SessionID          string            `json:"session_id"`
	ServiceCode        string            `json:"service_code"`
	Status             string            `json:"status"`
	UserInfo           *MOFAUserInfo     `json:"user_info"`
	Error              string            `json:"error,omitempty"`
}

// MOFAUserInfo user information from MOFA
type MOFAUserInfo struct {
	EmiratesID         string `json:"emirates_id"`
	FullName           string `json:"full_name"`
	Nationality        string `json:"nationality"`
	ServiceHistory     []string `json:"service_history"`
}

// NewUAEIdentityConnector creates a new UAE identity connector
func NewUAEIdentityConnector(config *UAEConnectorConfig) (*UAEIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	
	connector := &UAEIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}
	
	return connector, nil
}

// AuthenticateUAEPass authenticates using UAE Pass
func (uc *UAEIdentityConnector) AuthenticateUAEPass(ctx context.Context, req *UAEPassAuthRequest) (*UAEPassAuthResponse, error) {
	// Validate request
	if err := uc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate assurance level (aligns with eIDAS)
	if req.AssuranceLevel != "low" && req.AssuranceLevel != "substantial" && req.AssuranceLevel != "high" {
		return &UAEPassAuthResponse{
			Success: false,
			Error:   "Invalid assurance level (must be low, substantial, or high)",
		}, nil
	}
	
	// In production, this would:
	// 1. Generate OAuth2/OIDC authorization request
	// 2. Redirect to UAE Pass
	// 3. Receive callback with authorization code
	// 4. Exchange code for access token
	// 5. Fetch user info
	
	// Mock response for demonstration
	response := &UAEPassAuthResponse{
		Success:        true,
		AccessToken:    fmt.Sprintf("uaepass_%d", time.Now().Unix()),
		TokenType:      "Bearer",
		ExpiresIn:      3600,
		AssuranceLevel: req.AssuranceLevel,
		UserInfo: &UAEPassUserInfo{
			UUID:           fmt.Sprintf("UAE_%d", time.Now().Unix()),
			EmiratesID:     "784-1990-1234567-1",
			FullName:       "Mohammed Ahmed",
			FullNameArabic: "محمد أحمد",
			FirstName:      "Mohammed",
			LastName:       "Ahmed",
			DateOfBirth:    "1990-01-15",
			Gender:         "M",
			Nationality:    "ARE",
			Email:          "mohammed.ahmed@example.ae",
			Mobile:         "+971501234567",
		},
	}
	
	return response, nil
}

// VerifyEmiratesID verifies Emirates ID
// Format: 784-YYYY-NNNNNNN-C
// - 784: UAE country code
// - YYYY: Year of birth
// - NNNNNNN: Sequential number
// - C: Check digit
func (uc *UAEIdentityConnector) VerifyEmiratesID(ctx context.Context, req *EmiratesIDRequest) (*EmiratesIDResponse, error) {
	// Validate request
	if err := uc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate Emirates ID format
	valid, components := uc.validateEmiratesIDFormat(req.EmiratesID)
	if !valid {
		return &EmiratesIDResponse{
			Valid: false,
			Error: "Invalid Emirates ID format (expected: 784-YYYY-NNNNNNN-C)",
		}, nil
	}
	
	// In production, this would:
	// 1. Read Emirates ID chip (if NFC enabled)
	// 2. Call Federal Authority for Identity and Citizenship (ICA) API
	// 3. Verify card status
	// 4. Extract biometric data
	
	// Mock response for demonstration
	response := &EmiratesIDResponse{
		Valid:        true,
		EmiratesID:   req.EmiratesID,
		IDCardNumber: components["full"],
		CardType:     "resident",
		IssueDate:    "2020-01-15",
		ExpiryDate:   "2025-01-15",
		Status:       "active",
		UserInfo: &EmiratesIDInfo{
			EmiratesID:     req.EmiratesID,
			FullName:       "Mohammed Ahmed",
			FullNameArabic: "محمد أحمد",
			DateOfBirth:    "1990-01-15",
			Gender:         "M",
			Nationality:    "ARE",
			Emirate:        "Dubai",
			CardType:       "resident",
		},
	}
	
	return response, nil
}

// AuthenticateMOFA authenticates with MOFA (Ministry of Foreign Affairs)
func (uc *UAEIdentityConnector) AuthenticateMOFA(ctx context.Context, req *MOFAAuthRequest) (*MOFAAuthResponse, error) {
	// Validate request
	if err := uc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Verify Emirates ID format first
	emiratesReq := &EmiratesIDRequest{
		EmiratesID: req.EmiratesID,
	}
	emiratesResp, err := uc.VerifyEmiratesID(ctx, emiratesReq)
	if err != nil || !emiratesResp.Valid {
		return &MOFAAuthResponse{
			Success: false,
			Error:   "Invalid Emirates ID",
		}, nil
	}
	
	// In production, this would:
	// 1. Connect to MOFA authentication system
	// 2. Verify service code
	// 3. Check attestation/legalization status
	// 4. Return service history
	
	// Mock response for demonstration
	response := &MOFAAuthResponse{
		Success:     true,
		SessionID:   fmt.Sprintf("mofa_%d", time.Now().Unix()),
		ServiceCode: req.ServiceCode,
		Status:      "authenticated",
		UserInfo: &MOFAUserInfo{
			EmiratesID:  req.EmiratesID,
			FullName:    "Mohammed Ahmed",
			Nationality: "ARE",
			ServiceHistory: []string{
				"Document attestation (2024-06-15)",
				"Certificate legalization (2024-03-10)",
			},
		},
	}
	
	return response, nil
}

// Helper methods

func (uc *UAEIdentityConnector) validateEmiratesIDFormat(emiratesID string) (bool, map[string]string) {
	// Remove spaces and validate format: 784-YYYY-NNNNNNN-C
	emiratesID = strings.ReplaceAll(emiratesID, " ", "")
	
	pattern := regexp.MustCompile(`^784-(\d{4})-(\d{7})-(\d)$`)
	matches := pattern.FindStringSubmatch(emiratesID)
	
	if matches == nil {
		return false, nil
	}
	
	components := map[string]string{
		"full":       emiratesID,
		"country":    "784",
		"year":       matches[1],
		"number":     matches[2],
		"checkdigit": matches[3],
	}
	
	// Validate year (reasonable range: 1900-current year)
	var year int
	fmt.Sscanf(components["year"], "%d", &year)
	currentYear := time.Now().Year()
	if year < 1900 || year > currentYear {
		return false, nil
	}
	
	return true, components
}

func (uc *UAEIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (uc *UAEIdentityConnector) GetMetrics() map[string]interface{} {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	
	return map[string]interface{}{
		"connector": "uae_identity",
	}
}

// Close closes the connector and cleans up resources
func (uc *UAEIdentityConnector) Close() error {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	return nil
}
