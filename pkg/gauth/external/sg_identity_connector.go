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

// SingaporeIdentityConnector handles Singapore identity verification
// Supports SingPass, NRIC/FIN, MyInfo, CorpPass
type SingaporeIdentityConnector struct {
	config     *SingaporeConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// SingaporeConnectorConfig configuration for Singapore identity connector
type SingaporeConnectorConfig struct {
	// SingPass configuration
	SingPassURL       string `validate:"required,url"`
	SingPassClientID  string `validate:"required"`
	SingPassSecret    string `validate:"required"`
	
	// MyInfo configuration
	MyInfoURL         string `validate:"url"`
	MyInfoClientID    string
	MyInfoSecret      string
	
	// CorpPass configuration
	CorpPassURL       string `validate:"url"`
	CorpPassClientID  string
	CorpPassSecret    string
	
	// Timeouts
	RequestTimeout    time.Duration
}

// SingPassAuthRequest SingPass authentication request
type SingPassAuthRequest struct {
	EServiceID          string   `json:"e_service_id" validate:"required"`
	AuthLevel           string   `json:"auth_level" validate:"required,oneof=L0 L2"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
}

// SingPassAuthResponse SingPass authentication response
type SingPassAuthResponse struct {
	Success    bool              `json:"success"`
	SessionID  string            `json:"session_id"`
	AuthLevel  string            `json:"auth_level"`
	UserInfo   *SingPassUserInfo `json:"user_info"`
	Attributes map[string]string `json:"attributes"`
	Error      string            `json:"error,omitempty"`
}

// SingPassUserInfo user information from SingPass
type SingPassUserInfo struct {
	NRIC        string      `json:"nric"` // S/T/F/G prefix + 7 digits + check letter
	UEN         string      `json:"uen,omitempty"` // For businesses
	Name        string      `json:"name"`
	Sex         string      `json:"sex"`
	Race        string      `json:"race"`
	Nationality string      `json:"nationality"`
	DateOfBirth string      `json:"date_of_birth"`
	Email       string      `json:"email"`
	MobileNo    string      `json:"mobile_no"`
	RegAddress  *SGAddress  `json:"reg_address"`
	MailAddress *SGAddress  `json:"mail_address,omitempty"`
}

// SGAddress Singapore address structure
type SGAddress struct {
	Block      string `json:"block,omitempty"`
	Building   string `json:"building,omitempty"`
	Floor      string `json:"floor,omitempty"`
	Unit       string `json:"unit,omitempty"`
	Street     string `json:"street"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// NRICRequest NRIC/FIN validation request
type NRICRequest struct {
	NRIC        string `json:"nric" validate:"required,len=9"`
	Name        string `json:"name"`
	DateOfBirth string `json:"date_of_birth"`
}

// NRICResponse NRIC/FIN validation response
type NRICResponse struct {
	Valid           bool   `json:"valid"`
	NRIC            string `json:"nric"`
	Type            string `json:"type"` // "citizen", "pr", "foreigner", "other"
	CheckLetterValid bool  `json:"check_letter_valid"`
	Error           string `json:"error,omitempty"`
}

// MyInfoRequest MyInfo data retrieval request
type MyInfoRequest struct {
	UINFIN            string   `json:"uinfin" validate:"required"` // NRIC/FIN
	Attributes        []string `json:"attributes" validate:"required,min=1"`
	Purpose           string   `json:"purpose" validate:"required"`
	AuthorizationCode string   `json:"authorization_code" validate:"required"`
}

// MyInfoResponse MyInfo data retrieval response
type MyInfoResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error,omitempty"`
}

// CorpPassAuthRequest CorpPass authentication request (for businesses)
type CorpPassAuthRequest struct {
	EntityID            string   `json:"entity_id" validate:"required"` // UEN
	EServiceID          string   `json:"e_service_id" validate:"required"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
}

// CorpPassAuthResponse CorpPass authentication response
type CorpPassAuthResponse struct {
	Success    bool                 `json:"success"`
	SessionID  string               `json:"session_id"`
	EntityInfo *CorpPassEntityInfo  `json:"entity_info"`
	UserInfo   *CorpPassUserInfo    `json:"user_info"`
	Attributes map[string]string    `json:"attributes"`
	Error      string               `json:"error,omitempty"`
}

// CorpPassEntityInfo entity (company) information from CorpPass
type CorpPassEntityInfo struct {
	UEN         string `json:"uen"` // Unique Entity Number
	EntityName  string `json:"entity_name"`
	EntityType  string `json:"entity_type"`
	RegStatus   string `json:"reg_status"`
}

// CorpPassUserInfo user information from CorpPass
type CorpPassUserInfo struct {
	NRIC       string   `json:"nric"`
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	MobileNo   string   `json:"mobile_no"`
	Roles      []string `json:"roles"` // Corporate roles
}

// NewSingaporeIdentityConnector creates a new Singapore identity connector
func NewSingaporeIdentityConnector(config *SingaporeConnectorConfig) (*SingaporeIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	
	connector := &SingaporeIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}
	
	return connector, nil
}

// AuthenticateSingPass authenticates using SingPass
// Auth levels: L0 (basic), L2 (2FA with OneKey/SMS)
func (sc *SingaporeIdentityConnector) AuthenticateSingPass(ctx context.Context, req *SingPassAuthRequest) (*SingPassAuthResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate auth level
	if req.AuthLevel != "L0" && req.AuthLevel != "L2" {
		return &SingPassAuthResponse{
			Success: false,
			Error:   "Invalid auth level (must be L0 or L2)",
		}, nil
	}
	
	// In production, this would:
	// 1. Redirect to SingPass login
	// 2. Perform SAML 2.0 authentication
	// 3. Receive user attributes
	// 4. Verify 2FA if L2
	
	// Mock response for demonstration
	response := &SingPassAuthResponse{
		Success:   true,
		SessionID: fmt.Sprintf("singpass_%d", time.Now().Unix()),
		AuthLevel: req.AuthLevel,
		UserInfo: &SingPassUserInfo{
			NRIC:        "S1234567D",
			Name:        "Tan Ah Kow",
			Sex:         "M",
			Race:        "Chinese",
			Nationality: "SG",
			DateOfBirth: "1990-01-15",
			Email:       "ahkow@example.sg",
			MobileNo:    "+6591234567",
		},
		Attributes: make(map[string]string),
	}
	
	return response, nil
}

// ValidateNRIC validates Singapore NRIC/FIN
// Format: S/T/F/G + 7 digits + check letter
// S/T: Singapore Citizens and Permanent Residents
// F/G: Foreigners
func (sc *SingaporeIdentityConnector) ValidateNRIC(ctx context.Context, req *NRICRequest) (*NRICResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return &NRICResponse{Valid: false, Error: err.Error()}, nil
	}
	
	nric := strings.ToUpper(strings.TrimSpace(req.NRIC))
	
	// Validate format (S/T/F/G/M + 7 digits + check letter)
	if !regexp.MustCompile(`^[STFGM]\d{7}[A-Z]$`).MatchString(nric) {
		return &NRICResponse{
			Valid: false,
			Error: "Invalid NRIC/FIN format",
		}, nil
	}
	
	// Determine type
	prefix := string(nric[0])
	idType := ""
	switch prefix {
	case "S", "T":
		idType = "citizen_pr"
	case "F", "G":
		idType = "foreigner"
	case "M":
		idType = "other"
	}
	
	// Validate check letter
	checkLetterValid := sc.validateNRICCheckLetter(nric)
	
	response := &NRICResponse{
		Valid:            checkLetterValid,
		NRIC:             nric,
		Type:             idType,
		CheckLetterValid: checkLetterValid,
	}
	
	if !checkLetterValid {
		response.Error = "Invalid check letter"
	}
	
	return response, nil
}

// RetrieveMyInfo retrieves personal data from MyInfo
func (sc *SingaporeIdentityConnector) RetrieveMyInfo(ctx context.Context, req *MyInfoRequest) (*MyInfoResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate UINFIN format
	if !regexp.MustCompile(`^[STFGM]\d{7}[A-Z]$`).MatchString(req.UINFIN) {
		return &MyInfoResponse{
			Success: false,
			Error:   "Invalid UINFIN format",
		}, nil
	}
	
	// In production, this would:
	// 1. Verify authorization code
	// 2. Retrieve attributes from MyInfo API
	// 3. Return requested person data
	
	// Mock response for demonstration
	response := &MyInfoResponse{
		Success: true,
		Data: map[string]interface{}{
			"uinfin":    req.UINFIN,
			"name":      "Tan Ah Kow",
			"sex":       "M",
			"race":      "Chinese",
			"dob":       "1990-01-15",
			"email":     "ahkow@example.sg",
			"mobileno":  "+6591234567",
		},
	}
	
	return response, nil
}

// AuthenticateCorpPass authenticates using CorpPass (for businesses)
func (sc *SingaporeIdentityConnector) AuthenticateCorpPass(ctx context.Context, req *CorpPassAuthRequest) (*CorpPassAuthResponse, error) {
	// Validate request
	if err := sc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Validate UEN format (9-10 alphanumeric characters)
	if !regexp.MustCompile(`^[0-9A-Z]{9,10}$`).MatchString(req.EntityID) {
		return &CorpPassAuthResponse{
			Success: false,
			Error:   "Invalid UEN format",
		}, nil
	}
	
	// In production, this would:
	// 1. Redirect to CorpPass login
	// 2. Perform SAML 2.0 authentication
	// 3. Receive entity and user information
	// 4. Verify corporate roles
	
	// Mock response for demonstration
	response := &CorpPassAuthResponse{
		Success:   true,
		SessionID: fmt.Sprintf("corppass_%d", time.Now().Unix()),
		EntityInfo: &CorpPassEntityInfo{
			UEN:        req.EntityID,
			EntityName: "Example Pte Ltd",
			EntityType: "Local Company",
			RegStatus:  "Live",
		},
		UserInfo: &CorpPassUserInfo{
			NRIC:     "S1234567D",
			Name:     "Tan Ah Kow",
			Email:    "ahkow@example.com.sg",
			MobileNo: "+6591234567",
			Roles:    []string{"Admin", "Authorized Personnel"},
		},
		Attributes: make(map[string]string),
	}
	
	return response, nil
}

// Helper methods

func (sc *SingaporeIdentityConnector) validateNRICCheckLetter(nric string) bool {
	// NRIC check letter algorithm
	prefix := nric[0]
	digits := nric[1:8]
	checkLetter := nric[8]
	
	// Weights: 2, 7, 6, 5, 4, 3, 2
	weights := []int{2, 7, 6, 5, 4, 3, 2}
	sum := 0
	
	for i, weight := range weights {
		digit := int(digits[i] - '0')
		sum += digit * weight
	}
	
	// Different offset for different prefix types
	var offset int
	switch prefix {
	case 'T', 'G', 'M':
		offset = 4
	default:
		offset = 0
	}
	
	sum += offset
	
	// Check letter mapping
	var checkLetters string
	switch prefix {
	case 'S', 'T':
		checkLetters = "JZIHGFEDCBA"
	case 'F', 'G':
		checkLetters = "XWUTRQPNMLK"
	case 'M':
		checkLetters = "KXWUTRQPNMLK"
	default:
		return false
	}
	
	remainder := sum % 11
	expectedLetter := checkLetters[remainder]
	
	return checkLetter == byte(expectedLetter)
}

func (sc *SingaporeIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (sc *SingaporeIdentityConnector) GetMetrics() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	return map[string]interface{}{
		"connector": "singapore_identity",
	}
}

// Close closes the connector and cleans up resources
func (sc *SingaporeIdentityConnector) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return nil
}
