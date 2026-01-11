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

// KoreaIdentityConnector handles South Korean identity verification
// Supports i-PIN, Resident Registration Number, Alien Registration Card
type KoreaIdentityConnector struct {
	config     *KoreaConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// KoreaConnectorConfig configuration for Korean identity connector
type KoreaConnectorConfig struct {
	// i-PIN configuration
	IPINServiceURL string `validate:"required,url"`
	IPINClientID   string `validate:"required"`
	IPINSecret     string `validate:"required"`

	// PASS (Mobile authentication) configuration
	PASSServiceURL string `validate:"url"`
	PASSClientID   string
	PASSSecret     string

	// Government24 configuration
	Gov24URL    string `validate:"url"`
	Gov24APIKey string

	// Timeouts
	RequestTimeout time.Duration
}

// IPINAuthRequest i-PIN authentication request
type IPINAuthRequest struct {
	ServiceID           string   `json:"service_id" validate:"required"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
	UsePASS             bool     `json:"use_pass"` // Use mobile PASS authentication
}

// IPINAuthResponse i-PIN authentication response
type IPINAuthResponse struct {
	Success    bool              `json:"success"`
	SessionID  string            `json:"session_id"`
	UserInfo   *IPINUserInfo     `json:"user_info"`
	Attributes map[string]string `json:"attributes"`
	Error      string            `json:"error,omitempty"`
}

// IPINUserInfo user information from i-PIN
type IPINUserInfo struct {
	IPIN         string     `json:"i_pin"` // Unique identifier
	CI           string     `json:"ci"`    // Connecting Information (88 chars)
	DI           string     `json:"di"`    // Duplication Information (64 chars)
	Name         string     `json:"name"`
	NameEnglish  string     `json:"name_english,omitempty"`
	Gender       string     `json:"gender"`
	Nationality  string     `json:"nationality"`
	DateOfBirth  string     `json:"date_of_birth"`
	MobileNumber string     `json:"mobile_number"`
	Email        string     `json:"email"`
	Address      *KRAddress `json:"address,omitempty"`
	IsForeigner  bool       `json:"is_foreigner"`
}

// KRAddress Korean address structure
type KRAddress struct {
	JibunAddress  string `json:"jibun_address"` // Traditional address
	RoadAddress   string `json:"road_address"`  // New road name address
	PostalCode    string `json:"postal_code"`
	DetailAddress string `json:"detail_address,omitempty"`
	ExtraAddress  string `json:"extra_address,omitempty"`
}

// RRNRequest Resident Registration Number validation request
type RRNRequest struct {
	RRN         string `json:"rrn" validate:"required,len=13"`
	Name        string `json:"name"`
	DateOfBirth string `json:"date_of_birth"`
}

// RRNResponse Resident Registration Number validation response
type RRNResponse struct {
	Valid           bool   `json:"valid"`
	RRN             string `json:"rrn"` // Masked format
	DateOfBirth     string `json:"date_of_birth"`
	Gender          string `json:"gender"`
	RegionCode      string `json:"region_code"`
	CheckDigitValid bool   `json:"check_digit_valid"`
	Error           string `json:"error,omitempty"`
}

// ARCRequest Alien Registration Card validation request
type ARCRequest struct {
	CardNumber  string `json:"card_number" validate:"required,len=13"`
	Name        string `json:"name" validate:"required"`
	Nationality string `json:"nationality" validate:"required"`
	DateOfBirth string `json:"date_of_birth"`
	VisaType    string `json:"visa_type"`
}

// ARCResponse Alien Registration Card validation response
type ARCResponse struct {
	Valid           bool       `json:"valid"`
	CardNumber      string     `json:"card_number"` // Masked
	Name            string     `json:"name"`
	NameKorean      string     `json:"name_korean,omitempty"`
	Nationality     string     `json:"nationality"`
	DateOfBirth     string     `json:"date_of_birth"`
	Gender          string     `json:"gender"`
	VisaType        string     `json:"visa_type"`
	IssueDate       string     `json:"issue_date"`
	ExpiryDate      string     `json:"expiry_date"`
	Address         *KRAddress `json:"address,omitempty"`
	CheckDigitValid bool       `json:"check_digit_valid"`
	Error           string     `json:"error,omitempty"`
}

// PASSAuthRequest PASS (mobile) authentication request
type PASSAuthRequest struct {
	ServiceID           string   `json:"service_id" validate:"required"`
	TelecomProvider     string   `json:"telecom_provider" validate:"required,oneof=SKT KT LGU+ MVNO"`
	RequestedAttributes []string `json:"requested_attributes" validate:"required,min=1"`
	Purpose             string   `json:"purpose" validate:"required"`
	ReturnURL           string   `json:"return_url" validate:"required,url"`
}

// PASSAuthResponse PASS authentication response
type PASSAuthResponse struct {
	Success    bool              `json:"success"`
	SessionID  string            `json:"session_id"`
	UserInfo   *PASSUserInfo     `json:"user_info"`
	Attributes map[string]string `json:"attributes"`
	Error      string            `json:"error,omitempty"`
}

// PASSUserInfo user information from PASS
type PASSUserInfo struct {
	CI              string `json:"ci"` // Connecting Information
	DI              string `json:"di"` // Duplication Information
	Name            string `json:"name"`
	DateOfBirth     string `json:"date_of_birth"`
	Gender          string `json:"gender"`
	Nationality     string `json:"nationality"`
	MobileNumber    string `json:"mobile_number"`
	TelecomProvider string `json:"telecom_provider"`
}

// NewKoreaIdentityConnector creates a new Korean identity connector
func NewKoreaIdentityConnector(config *KoreaConnectorConfig) (*KoreaIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &KoreaIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// AuthenticateIPIN authenticates using i-PIN
func (kc *KoreaIdentityConnector) AuthenticateIPIN(ctx context.Context, req *IPINAuthRequest) (*IPINAuthResponse, error) {
	// Validate request
	if err := kc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// In production, this would:
	// 1. Redirect to i-PIN service
	// 2. User authenticates with i-PIN or PASS
	// 3. Receive CI/DI and personal information
	// 4. Return verified identity

	// Mock response for demonstration
	response := &IPINAuthResponse{
		Success:   true,
		SessionID: fmt.Sprintf("ipin_%d", time.Now().Unix()),
		UserInfo: &IPINUserInfo{
			IPIN:        fmt.Sprintf("IPIN%d", time.Now().Unix()),
			CI:          strings.Repeat("A", 88), // 88 character CI
			DI:          strings.Repeat("B", 64), // 64 character DI
			Name:        "김철수",
			NameEnglish: "Kim Chul-soo",
			Gender:      "M",
			Nationality: "KR",
			DateOfBirth: "1990-01-15",
			IsForeigner: false,
		},
		Attributes: make(map[string]string),
	}

	return response, nil
}

// ValidateRRN validates Korean Resident Registration Number (주민등록번호)
// Format: YYMMDD-GXXXXXX (13 digits)
// G: Gender/Century (1/2: 1900s, 3/4: 2000s, 5/6: 1900s foreigner, 7/8: 2000s foreigner)
func (kc *KoreaIdentityConnector) ValidateRRN(ctx context.Context, req *RRNRequest) (*RRNResponse, error) {
	// Validate request
	if err := kc.validator.Struct(req); err != nil {
		return &RRNResponse{Valid: false, Error: err.Error()}, nil
	}

	rrn := strings.ReplaceAll(strings.TrimSpace(req.RRN), "-", "")

	// Validate format (13 digits)
	if !regexp.MustCompile(`^\d{13}$`).MatchString(rrn) {
		return &RRNResponse{
			Valid: false,
			Error: "Invalid RRN format (must be 13 digits)",
		}, nil
	}

	// Extract components
	year := rrn[0:2]
	month := rrn[2:4]
	day := rrn[4:6]
	genderCode := rrn[6:7]
	regionCode := rrn[7:11]

	// Determine century and gender
	var century string
	var gender string
	switch genderCode {
	case "1", "2":
		century = "19"
	case "3", "4":
		century = "20"
	case "5", "6":
		century = "19" // Foreigner
	case "7", "8":
		century = "20" // Foreigner
	case "9", "0":
		century = "18"
	default:
		return &RRNResponse{
			Valid: false,
			Error: "Invalid gender code",
		}, nil
	}

	// Odd: Male, Even: Female
	genderInt, _ := strconv.Atoi(genderCode)
	if genderInt%2 == 1 || genderInt == 0 {
		gender = "M"
	} else {
		gender = "F"
	}

	// Validate check digit
	checkDigitValid := kc.validateRRNCheckDigit(rrn)

	// Mask RRN for response (show only first 6 digits)
	maskedRRN := rrn[0:6] + "-*******"

	response := &RRNResponse{
		Valid:           checkDigitValid,
		RRN:             maskedRRN,
		DateOfBirth:     fmt.Sprintf("%s%s-%s-%s", century, year, month, day),
		Gender:          gender,
		RegionCode:      regionCode,
		CheckDigitValid: checkDigitValid,
	}

	if !checkDigitValid {
		response.Error = "Invalid check digit"
	}

	return response, nil
}

// VerifyARC verifies Alien Registration Card (외국인등록증)
func (kc *KoreaIdentityConnector) VerifyARC(ctx context.Context, req *ARCRequest) (*ARCResponse, error) {
	// Validate request
	if err := kc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	cardNumber := strings.ReplaceAll(strings.TrimSpace(req.CardNumber), "-", "")

	// Validate format (13 digits, starts with 5-8)
	if !regexp.MustCompile(`^[5-8]\d{12}$`).MatchString(cardNumber) {
		return &ARCResponse{
			Valid: false,
			Error: "Invalid ARC number format",
		}, nil
	}

	// Validate check digit (same algorithm as RRN)
	checkDigitValid := kc.validateRRNCheckDigit(cardNumber)

	// Extract gender from 7th digit
	genderCode := cardNumber[6:7]
	genderInt, _ := strconv.Atoi(genderCode)
	var gender string
	if genderInt%2 == 1 {
		gender = "M"
	} else {
		gender = "F"
	}

	// Mask card number
	maskedNumber := cardNumber[0:6] + "-*******"

	// In production, this would verify with Immigration Office database
	response := &ARCResponse{
		Valid:           checkDigitValid,
		CardNumber:      maskedNumber,
		Name:            req.Name,
		Nationality:     req.Nationality,
		DateOfBirth:     req.DateOfBirth,
		Gender:          gender,
		VisaType:        req.VisaType,
		IssueDate:       "2020-01-15",
		ExpiryDate:      "2025-01-15",
		CheckDigitValid: checkDigitValid,
	}

	if !checkDigitValid {
		response.Error = "Invalid check digit"
	}

	return response, nil
}

// AuthenticatePASS authenticates using mobile PASS
func (kc *KoreaIdentityConnector) AuthenticatePASS(ctx context.Context, req *PASSAuthRequest) (*PASSAuthResponse, error) {
	// Validate request
	if err := kc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// In production, this would:
	// 1. Initiate mobile authentication
	// 2. User approves on mobile device
	// 3. Receive CI/DI and verified information

	// Mock response for demonstration
	response := &PASSAuthResponse{
		Success:   true,
		SessionID: fmt.Sprintf("pass_%d", time.Now().Unix()),
		UserInfo: &PASSUserInfo{
			CI:              strings.Repeat("A", 88),
			DI:              strings.Repeat("B", 64),
			Name:            "김철수",
			DateOfBirth:     "1990-01-15",
			Gender:          "M",
			Nationality:     "KR",
			MobileNumber:    "+821012345678",
			TelecomProvider: req.TelecomProvider,
		},
		Attributes: make(map[string]string),
	}

	return response, nil
}

// Helper methods

func (kc *KoreaIdentityConnector) validateRRNCheckDigit(rrn string) bool {
	// RRN/ARC check digit algorithm
	// Weights: 2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5
	weights := []int{2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5}
	sum := 0

	for i := 0; i < 12; i++ {
		digit, _ := strconv.Atoi(string(rrn[i]))
		sum += digit * weights[i]
	}

	checkDigit := (11 - (sum % 11)) % 10
	actualCheckDigit, _ := strconv.Atoi(string(rrn[12]))

	return checkDigit == actualCheckDigit
}

func (kc *KoreaIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

var _ = (*KoreaIdentityConnector).generateCacheKey

// GetMetrics returns connector metrics
func (kc *KoreaIdentityConnector) GetMetrics() map[string]interface{} {
	kc.mu.RLock()
	defer kc.mu.RUnlock()

	return map[string]interface{}{
		"connector": "korea_identity",
	}
}

// Close closes the connector and cleans up resources
func (kc *KoreaIdentityConnector) Close() error {
	kc.mu.Lock()
	defer kc.mu.Unlock()
	return nil
}
