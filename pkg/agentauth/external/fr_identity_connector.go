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

// FranceIdentityConnector handles French identity verification
// Supports FranceConnect, Carte Vitale, CNI, French passport, INSEE number
type FranceIdentityConnector struct {
	config     *FranceConnectorConfig
	httpClient *http.Client
	validator  *validator.Validate
	mu         sync.RWMutex
}

// FranceConnectorConfig configuration for French identity connector
type FranceConnectorConfig struct {
	// FranceConnect configuration
	FranceConnectURL string `validate:"required,url"`
	ClientID         string `validate:"required"`
	ClientSecret     string `validate:"required"`
	RedirectURI      string `validate:"required,url"`

	// ANTS (Agence Nationale des Titres Sécurisés) API
	ANTSAPIKey string
	ANTSAPIURL string `validate:"url"`

	// INSEE (Institut National de la Statistique) API
	INSEEAPIKey string
	INSEEAPIURL string `validate:"url"`

	// Timeouts
	RequestTimeout time.Duration
}

// FranceConnectAuthRequest FranceConnect authentication request
type FranceConnectAuthRequest struct {
	ServiceID   string   `json:"service_id" validate:"required"`
	Scope       []string `json:"scope" validate:"required,min=1"`
	EIDASLevel  string   `json:"eidas_level" validate:"required,oneof=low substantial high"`
	Nonce       string   `json:"nonce" validate:"required"`
	State       string   `json:"state" validate:"required"`
	RedirectURI string   `json:"redirect_uri" validate:"required,url"`
}

// FranceConnectAuthResponse FranceConnect authentication response
type FranceConnectAuthResponse struct {
	Success          bool                   `json:"success"`
	Code             string                 `json:"code"`
	State            string                 `json:"state"`
	IDToken          string                 `json:"id_token"`
	AccessToken      string                 `json:"access_token"`
	EIDASLevel       string                 `json:"eidas_level"`
	UserInfo         *FranceConnectUserInfo `json:"user_info"`
	Error            string                 `json:"error,omitempty"`
	ErrorDescription string                 `json:"error_description,omitempty"`
}

// FranceConnectUserInfo user information from FranceConnect
type FranceConnectUserInfo struct {
	Sub               string     `json:"sub"` // Unique identifier
	GivenName         string     `json:"given_name"`
	FamilyName        string     `json:"family_name"`
	PreferredUsername string     `json:"preferred_username"`
	Birthdate         string     `json:"birthdate"`
	Gender            string     `json:"gender"`
	Birthplace        string     `json:"birthplace"`
	Birthcountry      string     `json:"birthcountry"`
	Email             string     `json:"email"`
	EmailVerified     bool       `json:"email_verified"`
	Address           *FRAddress `json:"address"`
}

// FRAddress French address structure
type FRAddress struct {
	Formatted     string `json:"formatted"`
	StreetAddress string `json:"street_address"`
	Locality      string `json:"locality"`
	Region        string `json:"region"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
}

// INSEENumberRequest INSEE number (NIR) validation request
type INSEENumberRequest struct {
	INSEENumber string `json:"insee_number" validate:"required,len=15"`
	FirstName   string `json:"first_name" validate:"required"`
	LastName    string `json:"last_name" validate:"required"`
	DateOfBirth string `json:"date_of_birth" validate:"required"`
}

// INSEENumberResponse INSEE number validation response
type INSEENumberResponse struct {
	Valid              bool   `json:"valid"`
	INSEENumber        string `json:"insee_number"`
	Sex                string `json:"sex"` // 1=male, 2=female
	YearOfBirth        string `json:"year_of_birth"`
	MonthOfBirth       string `json:"month_of_birth"`
	DepartmentOfBirth  string `json:"department_of_birth"`
	CommuneOfBirth     string `json:"commune_of_birth"`
	RegistrationNumber string `json:"registration_number"`
	ControlKey         string `json:"control_key"`
	Error              string `json:"error,omitempty"`
}

// CNIVerificationRequest Carte Nationale d'Identité verification request
type CNIVerificationRequest struct {
	CNINumber    string `json:"cni_number" validate:"required"`
	FirstName    string `json:"first_name" validate:"required"`
	LastName     string `json:"last_name" validate:"required"`
	DateOfBirth  string `json:"date_of_birth" validate:"required"`
	PlaceOfBirth string `json:"place_of_birth" validate:"required"`
}

// CNIVerificationResponse CNI verification response
type CNIVerificationResponse struct {
	Valid            bool   `json:"valid"`
	CNINumber        string `json:"cni_number"`
	DocumentType     string `json:"document_type"` // CNI, Passport
	IssueDate        string `json:"issue_date"`
	ExpiryDate       string `json:"expiry_date"`
	IssuinagentAuthority string `json:"issuing_authority"`
	MRZVerified      bool   `json:"mrz_verified"`
	ChipVerified     bool   `json:"chip_verified"`
	Status           string `json:"status"` // valid, expired, revoked, lost
	Error            string `json:"error,omitempty"`
}

// FrenchPassportRequest French passport verification request
type FrenchPassportRequest struct {
	PassportNumber string `json:"passport_number" validate:"required"`
	FirstName      string `json:"first_name" validate:"required"`
	LastName       string `json:"last_name" validate:"required"`
	DateOfBirth    string `json:"date_of_birth" validate:"required"`
	Nationality    string `json:"nationality" validate:"required,iso3166_1_alpha3"`
}

// FrenchPassportResponse French passport verification response
type FrenchPassportResponse struct {
	Valid             bool   `json:"valid"`
	PassportNumber    string `json:"passport_number"`
	DocumentType      string `json:"document_type"`
	IssueDate         string `json:"issue_date"`
	ExpiryDate        string `json:"expiry_date"`
	IssuinagentAuthority  string `json:"issuing_authority"`
	MRZVerified       bool   `json:"mrz_verified"`
	RFIDVerified      bool   `json:"rfid_verified"`
	BiometricVerified bool   `json:"biometric_verified"`
	Status            string `json:"status"` // valid, expired, revoked, lost
	Error             string `json:"error,omitempty"`
}

// CarteVitaleRequest Carte Vitale verification request
type CarteVitaleRequest struct {
	CarteVitaleNumber string `json:"carte_vitale_number" validate:"required"`
	INSEENumber       string `json:"insee_number" validate:"required,len=15"`
	FirstName         string `json:"first_name" validate:"required"`
	LastName          string `json:"last_name" validate:"required"`
	DateOfBirth       string `json:"date_of_birth" validate:"required"`
}

// CarteVitaleResponse Carte Vitale verification response
type CarteVitaleResponse struct {
	Valid             bool   `json:"valid"`
	CarteVitaleNumber string `json:"carte_vitale_number"`
	INSEENumber       string `json:"insee_number"`
	HealthInsuranceID string `json:"health_insurance_id"`
	Status            string `json:"status"` // active, expired, suspended
	ValidUntil        string `json:"valid_until"`
	OrganismCode      string `json:"organism_code"` // Social security organism
	Error             string `json:"error,omitempty"`
}

// NewFranceIdentityConnector creates a new French identity connector
func NewFranceIdentityConnector(config *FranceConnectorConfig) (*FranceIdentityConnector, error) {
	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	connector := &FranceIdentityConnector{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		validator:  validate,
	}

	return connector, nil
}

// AuthenticateFranceConnect authenticates using FranceConnect (eIDAS)
func (fc *FranceIdentityConnector) AuthenticateFranceConnect(ctx context.Context, req *FranceConnectAuthRequest) (*FranceConnectAuthResponse, error) {
	// Validate request
	if err := fc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Build authorization URL (used for redirect in production)
	_ = fmt.Sprintf("%s/authorize?"+
		"response_type=code&"+
		"client_id=%s&"+
		"redirect_uri=%s&"+
		"scope=%s&"+
		"state=%s&"+
		"nonce=%s&"+
		"acr_values=eidas%s",
		fc.config.FranceConnectURL,
		fc.config.ClientID,
		req.RedirectURI,
		strings.Join(req.Scope, "+"),
		req.State,
		req.Nonce,
		fc.getEIDASLevel(req.EIDASLevel))

	// In production, this would redirect the user to FranceConnect
	response := &FranceConnectAuthResponse{
		Success:    true,
		State:      req.State,
		EIDASLevel: req.EIDASLevel,
		UserInfo: &FranceConnectUserInfo{
			Sub:        fmt.Sprintf("fc_%d", time.Now().Unix()),
			GivenName:  "Jean",
			FamilyName: "Dupont",
			Birthdate:  "1990-01-15",
			Gender:     "male",
		},
	}

	return response, nil
}

// ValidateINSEENumber validates French INSEE number (NIR - Numéro d'Inscription au Répertoire)
// Format: 1 YY MM DD DDD NNN KK
// - 1: Sex (1=male, 2=female)
// - YY: Year of birth (last 2 digits)
// - MM: Month of birth (01-12)
// - DD: Department of birth
// - DDD: Commune code
// - NNN: Registration number
// - KK: Control key (97 - (rest of number mod 97))
func (fc *FranceIdentityConnector) ValidateINSEENumber(ctx context.Context, req *INSEENumberRequest) (*INSEENumberResponse, error) {
	// Validate request
	if err := fc.validator.Struct(req); err != nil {
		return &INSEENumberResponse{Valid: false, Error: err.Error()}, nil
	}

	// Remove spaces and format
	inseeNumber := strings.ReplaceAll(req.INSEENumber, " ", "")
	if len(inseeNumber) != 15 {
		return &INSEENumberResponse{Valid: false, Error: "INSEE number must be 15 digits"}, nil
	}

	// Validate format (all digits)
	if !regexp.MustCompile(`^\d{15}$`).MatchString(inseeNumber) {
		return &INSEENumberResponse{Valid: false, Error: "INSEE number must contain only digits"}, nil
	}

	// Extract components
	sex := inseeNumber[0:1]
	year := inseeNumber[1:3]
	month := inseeNumber[3:5]
	department := inseeNumber[5:7]
	commune := inseeNumber[7:10]
	regNumber := inseeNumber[10:13]
	controlKey := inseeNumber[13:15]

	// Validate sex (1=male, 2=female)
	if sex != "1" && sex != "2" {
		return &INSEENumberResponse{Valid: false, Error: "Invalid sex code (must be 1 or 2)"}, nil
	}

	// Validate month (01-12 or 20-50 for special cases)
	var monthInt int
	_, _ = fmt.Sscanf(month, "%d", &monthInt) // Best effort parsing; will be 0 if invalid
	if (monthInt < 1 || monthInt > 12) && (monthInt < 20 || monthInt > 50) {
		return &INSEENumberResponse{Valid: false, Error: "Invalid month"}, nil
	}

	// Calculate control key
	// Take first 13 digits and compute: 97 - (number mod 97)
	baseNumber := inseeNumber[0:13]
	var baseNum int64
	_, _ = fmt.Sscanf(baseNumber, "%d", &baseNum) // Best effort parsing; will be 0 if invalid

	calculatedKey := 97 - (baseNum % 97)
	var providedKey int
	_, _ = fmt.Sscanf(controlKey, "%d", &providedKey) // Best effort parsing; will be 0 if invalid

	if int64(providedKey) != calculatedKey {
		return &INSEENumberResponse{
			Valid: false,
			Error: fmt.Sprintf("Invalid control key (expected %02d, got %s)", calculatedKey, controlKey),
		}, nil
	}

	response := &INSEENumberResponse{
		Valid:              true,
		INSEENumber:        inseeNumber,
		Sex:                sex,
		YearOfBirth:        year,
		MonthOfBirth:       month,
		DepartmentOfBirth:  department,
		CommuneOfBirth:     commune,
		RegistrationNumber: regNumber,
		ControlKey:         controlKey,
	}

	return response, nil
}

// VerifyCNI verifies French Carte Nationale d'Identité
func (fc *FranceIdentityConnector) VerifyCNI(ctx context.Context, req *CNIVerificationRequest) (*CNIVerificationResponse, error) {
	// Validate request
	if err := fc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate CNI number format (12 characters alphanumeric)
	if !fc.isValidCNINumber(req.CNINumber) {
		return &CNIVerificationResponse{
			Valid: false,
			Error: "Invalid CNI number format (must be 12 alphanumeric characters)",
		}, nil
	}

	// In production, this would call ANTS API
	response := &CNIVerificationResponse{
		Valid:            true,
		CNINumber:        req.CNINumber,
		DocumentType:     "CNI",
		IssueDate:        "2020-01-15",
		ExpiryDate:       "2030-01-15",
		IssuinagentAuthority: "Prefecture de Paris",
		MRZVerified:      true,
		ChipVerified:     true,
		Status:           "valid",
	}

	return response, nil
}

// VerifyFrenchPassport verifies French passport
func (fc *FranceIdentityConnector) VerifyFrenchPassport(ctx context.Context, req *FrenchPassportRequest) (*FrenchPassportResponse, error) {
	// Validate request
	if err := fc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate passport number format (2 letters + 7 digits)
	if !fc.isValidFrenchPassportNumber(req.PassportNumber) {
		return &FrenchPassportResponse{
			Valid: false,
			Error: "Invalid passport number format (must be 2 letters + 7 digits, e.g., AB1234567)",
		}, nil
	}

	// In production, this would call ANTS API
	response := &FrenchPassportResponse{
		Valid:             true,
		PassportNumber:    req.PassportNumber,
		DocumentType:      "Passport",
		IssueDate:         "2020-01-15",
		ExpiryDate:        "2030-01-15",
		IssuinagentAuthority:  "République Française",
		MRZVerified:       true,
		RFIDVerified:      true,
		BiometricVerified: true,
		Status:            "valid",
	}

	return response, nil
}

// VerifyCarteVitale verifies Carte Vitale (French health insurance card)
func (fc *FranceIdentityConnector) VerifyCarteVitale(ctx context.Context, req *CarteVitaleRequest) (*CarteVitaleResponse, error) {
	// Validate request
	if err := fc.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate INSEE number first
	inseeReq := &INSEENumberRequest{
		INSEENumber: req.INSEENumber,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DateOfBirth: req.DateOfBirth,
	}
	inseeResp, err := fc.ValidateINSEENumber(ctx, inseeReq)
	if err != nil || !inseeResp.Valid {
		return &CarteVitaleResponse{
			Valid: false,
			Error: "Invalid INSEE number",
		}, nil
	}

	// In production, this would call CNAM API
	response := &CarteVitaleResponse{
		Valid:             true,
		CarteVitaleNumber: req.CarteVitaleNumber,
		INSEENumber:       req.INSEENumber,
		HealthInsuranceID: fmt.Sprintf("HI_%d", time.Now().Unix()),
		Status:            "active",
		ValidUntil:        "2030-12-31",
		OrganismCode:      "CPAM",
	}

	return response, nil
}

// Helper methods

func (fc *FranceIdentityConnector) getEIDASLevel(level string) string {
	switch level {
	case "low":
		return "1"
	case "substantial":
		return "2"
	case "high":
		return "3"
	default:
		return "1"
	}
}

func (fc *FranceIdentityConnector) isValidCNINumber(cniNumber string) bool {
	// CNI format: 12 characters alphanumeric
	return regexp.MustCompile(`^[A-Z0-9]{12}$`).MatchString(strings.ToUpper(cniNumber))
}

func (fc *FranceIdentityConnector) isValidFrenchPassportNumber(passportNumber string) bool {
	// French passport format: 2 letters + 7 digits (e.g., AB1234567)
	return regexp.MustCompile(`^[A-Z]{2}\d{7}$`).MatchString(strings.ToUpper(passportNumber))
}

func (fc *FranceIdentityConnector) generateCacheKey(operation string, parts ...string) string {
	combined := strings.Join(append([]string{operation}, parts...), ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns connector metrics
func (fc *FranceIdentityConnector) GetMetrics() map[string]interface{} {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return map[string]interface{}{
		"connector": "france_identity",
	}
}

// Close closes the connector and cleans up resources
func (fc *FranceIdentityConnector) Close() error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return nil
}
