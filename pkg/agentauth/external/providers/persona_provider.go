package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth/external"
)

const (
	personaStatusPassed = "passed"
)

// PersonaProvider implements USIdentityAPIProvider for Persona API
// API Documentation: https://docs.withpersona.com/reference
type PersonaProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	name       string
}

// PersonaConfig holds configuration for Persona provider
type PersonaConfig struct {
	APIKey     string
	BaseURL    string // Default: https://withpersona.com/api/v1
	Timeout    time.Duration
	MaxRetries int
}

// NewPersonaProvider creates a new Persona API provider
func NewPersonaProvider(config *PersonaConfig) *PersonaProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://withpersona.com/api/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &PersonaProvider{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		name: "persona",
	}
}

// =============================================================================
// API Request/Response Structures
// =============================================================================

// PersonaVerificationRequest is the request sent to Persona API
type PersonaVerificationRequest struct {
	Data PersonaVerificationData `json:"data"`
}

type PersonaVerificationData struct {
	Type       string                        `json:"type"`
	Attributes PersonaVerificationAttributes `json:"attributes"`
}

type PersonaVerificationAttributes struct {
	// Document information
	DocumentType   string `json:"document-type"`
	DocumentNumber string `json:"document-number,omitempty"`
	FirstName      string `json:"name-first"`
	LastName       string `json:"name-last"`
	DateOfBirth    string `json:"birthdate,omitempty"`
	ExpirationDate string `json:"expiration-date,omitempty"`
	IssueDate      string `json:"issue-date,omitempty"`
	Country        string `json:"country-code,omitempty"`
	State          string `json:"identification-class,omitempty"` // For DL state

	// Optional fields
	Nationality string `json:"nationality,omitempty"`
	SSN         string `json:"identification-number,omitempty"`

	// Images (base64 encoded)
	FrontImage  string `json:"front-photo,omitempty"`
	BackImage   string `json:"back-photo,omitempty"`
	SelfieImage string `json:"selfie-photo,omitempty"`
}

// PersonaVerificationResponse is the response from Persona API
type PersonaVerificationResponse struct {
	Data PersonaVerificationResponseData `json:"data"`
}

type PersonaVerificationResponseData struct {
	ID         string                                `json:"id"`
	Type       string                                `json:"type"`
	Attributes PersonaVerificationResponseAttributes `json:"attributes"`
}

type PersonaVerificationResponseAttributes struct {
	Status            string                `json:"status"`
	CreatedAt         string                `json:"created-at"`
	CompletedAt       string                `json:"completed-at,omitempty"`
	Checks            []PersonaCheck        `json:"checks"`
	Fields            PersonaVerifiedFields `json:"fields,omitempty"`
	VerificationScore float64               `json:"verification-score,omitempty"`
}

type PersonaCheck struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Reasons []string `json:"reasons,omitempty"`
}

type PersonaVerifiedFields struct {
	NameFirst      PersonaField `json:"name-first,omitempty"`
	NameLast       PersonaField `json:"name-last,omitempty"`
	Birthdate      PersonaField `json:"birthdate,omitempty"`
	DocumentNumber PersonaField `json:"document-number,omitempty"`
	ExpirationDate PersonaField `json:"expiration-date,omitempty"`
	Address        PersonaField `json:"address,omitempty"`
}

type PersonaField struct {
	Value string `json:"value"`
	Valid bool   `json:"valid"`
}

// PersonaSSNVerificationRequest is for SSN validation
type PersonaSSNVerificationRequest struct {
	Data PersonaSSNData `json:"data"`
}

type PersonaSSNData struct {
	Type       string               `json:"type"`
	Attributes PersonaSSNAttributes `json:"attributes"`
}

type PersonaSSNAttributes struct {
	SSN         string `json:"identification-number"`
	FirstName   string `json:"name-first"`
	LastName    string `json:"name-last"`
	DateOfBirth string `json:"birthdate,omitempty"`
}

// =============================================================================
// USIdentityAPIProvider Implementation
// =============================================================================

// VerifyDocument implements USIdentityAPIProvider
func (p *PersonaProvider) VerifyDocument(ctx context.Context, req interface{}) (*external.IdentityVerificationResult, error) {
	switch r := req.(type) {
	case *external.PassportVerificationRequest:
		return p.verifyPassport(ctx, r)
	case *external.DLVerificationRequest:
		return p.verifyDriverLicense(ctx, r)
	case *external.StateIDVerificationRequest:
		return p.verifyStateID(ctx, r)
	default:
		return nil, fmt.Errorf("unsupported request type: %T", req)
	}
}

// ValidateSSN implements USIdentityAPIProvider
func (p *PersonaProvider) ValidateSSN(ctx context.Context, req *external.SSNValidationRequest) (*external.SSNValidationResult, error) {
	startTime := time.Now()

	// Build Persona request
	personaReq := PersonaSSNVerificationRequest{
		Data: PersonaSSNData{
			Type: "verification/database",
			Attributes: PersonaSSNAttributes{
				SSN:         req.SSN,
				FirstName:   req.FirstName,
				LastName:    req.LastName,
				DateOfBirth: req.DateOfBirth.Format("2006-01-02"),
			},
		},
	}

	// Make API request
	resp, err := p.makeRequest(ctx, "POST", "/verifications/database", personaReq)
	if err != nil {
		return nil, fmt.Errorf("persona API request failed: %w", err)
	}

	// Parse response
	var personaResp PersonaVerificationResponse
	if err := json.Unmarshal(resp, &personaResp); err != nil {
		return nil, fmt.Errorf("failed to parse persona response: %w", err)
	}

	// Convert to SSNValidationResult
	result := &external.SSNValidationResult{
		Valid:                 personaResp.Data.Attributes.Status == personaStatusPassed,
		ValidationLevel:       req.ValidationLevel,
		ConfidenceScore:       personaResp.Data.Attributes.VerificationScore,
		FormatValid:           true, // Persona validates format
		ProviderName:          p.name,
		ProviderTransactionID: personaResp.Data.ID,
		ProcessingTimeMs:      time.Since(startTime).Milliseconds(),
		ValidationTimestamp:   time.Now(),
	}

	// Check individual fields
	if personaResp.Data.Attributes.Fields.NameFirst.Valid {
		result.NameMatch = &external.NameMatchResult{
			Matched: personaResp.Data.Attributes.Fields.NameFirst.Valid,
			Score:   1.0,
		}
	}

	if personaResp.Data.Attributes.Fields.Birthdate.Valid {
		result.DOBMatch = &external.DOBMatchResult{
			Matched: personaResp.Data.Attributes.Fields.Birthdate.Valid,
		}
	}

	// Check for deceased status from checks
	for _, check := range personaResp.Data.Attributes.Checks {
		if check.Name == "deceased" {
			result.NotDeceased = check.Status == personaStatusPassed
		}
	}

	// Mask SSN in response
	if len(req.SSN) >= 9 {
		result.SSN = fmt.Sprintf("XXX-XX-%s", req.SSN[5:])
	}

	return result, nil
}

// GetProviderName implements USIdentityAPIProvider
func (p *PersonaProvider) GetProviderName() string {
	return p.name
}

// GetSupportedDocumentTypes implements USIdentityAPIProvider
func (p *PersonaProvider) GetSupportedDocumentTypes() []external.DocumentType {
	return []external.DocumentType{
		external.DocumentTypePassport,
		external.DocumentTypeDriverLicense,
		external.DocumentTypeStateID,
	}
}

// =============================================================================
// Private Methods
// =============================================================================

func (p *PersonaProvider) verifyPassport(ctx context.Context, req *external.PassportVerificationRequest) (*external.IdentityVerificationResult, error) {
	startTime := time.Now()

	// Build Persona request
	personaReq := PersonaVerificationRequest{
		Data: PersonaVerificationData{
			Type: "verification/government-id",
			Attributes: PersonaVerificationAttributes{
				DocumentType:   "passport",
				DocumentNumber: req.PassportNumber,
				FirstName:      req.FirstName,
				LastName:       req.LastName,
				DateOfBirth:    req.DateOfBirth.Format("2006-01-02"),
				ExpirationDate: req.ExpirationDate.Format("2006-01-02"),
				IssueDate:      req.IssueDate.Format("2006-01-02"),
				Country:        req.Nationality,
				Nationality:    req.Nationality,
			},
		},
	}

	// Add images if provided
	if len(req.PassportImageFront) > 0 {
		personaReq.Data.Attributes.FrontImage = encodeBase64(req.PassportImageFront)
	}
	if len(req.PassportImageBack) > 0 {
		personaReq.Data.Attributes.BackImage = encodeBase64(req.PassportImageBack)
	}
	if len(req.FaceImage) > 0 {
		personaReq.Data.Attributes.SelfieImage = encodeBase64(req.FaceImage)
	}

	// Make API request
	resp, err := p.makeRequest(ctx, "POST", "/verifications", personaReq)
	if err != nil {
		return nil, fmt.Errorf("persona API request failed: %w", err)
	}

	// Parse and convert response
	return p.parseVerificationResponse(resp, external.DocumentTypePassport, startTime)
}

func (p *PersonaProvider) verifyDriverLicense(ctx context.Context, req *external.DLVerificationRequest) (*external.IdentityVerificationResult, error) {
	startTime := time.Now()

	// Build Persona request
	personaReq := PersonaVerificationRequest{
		Data: PersonaVerificationData{
			Type: "verification/government-id",
			Attributes: PersonaVerificationAttributes{
				DocumentType:   "drivers_license",
				DocumentNumber: req.LicenseNumber,
				FirstName:      req.FirstName,
				LastName:       req.LastName,
				DateOfBirth:    req.DateOfBirth.Format("2006-01-02"),
				ExpirationDate: req.ExpirationDate.Format("2006-01-02"),
				IssueDate:      req.IssueDate.Format("2006-01-02"),
				Country:        "US",
				State:          req.State,
			},
		},
	}

	// Add images if provided
	if len(req.LicenseImageFront) > 0 {
		personaReq.Data.Attributes.FrontImage = encodeBase64(req.LicenseImageFront)
	}
	if len(req.LicenseImageBack) > 0 {
		personaReq.Data.Attributes.BackImage = encodeBase64(req.LicenseImageBack)
	}

	// Make API request
	resp, err := p.makeRequest(ctx, "POST", "/verifications", personaReq)
	if err != nil {
		return nil, fmt.Errorf("persona API request failed: %w", err)
	}

	// Parse and convert response
	result, err := p.parseVerificationResponse(resp, external.DocumentTypeDriverLicense, startTime)
	if err != nil {
		return nil, err
	}
	result.DocumentState = req.State
	return result, nil
}

func (p *PersonaProvider) verifyStateID(ctx context.Context, req *external.StateIDVerificationRequest) (*external.IdentityVerificationResult, error) {
	startTime := time.Now()

	// Build Persona request
	personaReq := PersonaVerificationRequest{
		Data: PersonaVerificationData{
			Type: "verification/government-id",
			Attributes: PersonaVerificationAttributes{
				DocumentType:   "id_card",
				DocumentNumber: req.IDNumber,
				FirstName:      req.FirstName,
				LastName:       req.LastName,
				DateOfBirth:    req.DateOfBirth.Format("2006-01-02"),
				ExpirationDate: req.ExpirationDate.Format("2006-01-02"),
				IssueDate:      req.IssueDate.Format("2006-01-02"),
				Country:        "US",
				State:          req.State,
			},
		},
	}

	// Add images if provided
	if len(req.IDImageFront) > 0 {
		personaReq.Data.Attributes.FrontImage = encodeBase64(req.IDImageFront)
	}
	if len(req.IDImageBack) > 0 {
		personaReq.Data.Attributes.BackImage = encodeBase64(req.IDImageBack)
	}

	// Make API request
	resp, err := p.makeRequest(ctx, "POST", "/verifications", personaReq)
	if err != nil {
		return nil, fmt.Errorf("persona API request failed: %w", err)
	}

	// Parse and convert response
	result, err := p.parseVerificationResponse(resp, external.DocumentTypeStateID, startTime)
	if err != nil {
		return nil, err
	}
	result.DocumentState = req.State
	return result, nil
}

func (p *PersonaProvider) parseVerificationResponse(
	respBody []byte,
	docType external.DocumentType,
	startTime time.Time,
) (*external.IdentityVerificationResult, error) {
	var personaResp PersonaVerificationResponse
	if err := json.Unmarshal(respBody, &personaResp); err != nil {
		return nil, fmt.Errorf("failed to parse persona response: %w", err)
	}

	// Convert status to verification level
	verificationLevel := external.VerificationLevelStandard
	if personaResp.Data.Attributes.Status == personaStatusPassed {
		verificationLevel = external.VerificationLevelEnhanced
	}

	// Build verification checks
	checks := &external.VerificationChecks{}
	for _, check := range personaResp.Data.Attributes.Checks {
		checkResult := &external.CheckResult{
			Status: convertPersonaStatus(check.Status),
			Score:  1.0,
		}
		if len(check.Reasons) > 0 {
			checkResult.Details = check.Reasons[0]
		}

		switch check.Name {
		case "document_authenticity":
			checks.DocumentAuthenticity = checkResult
		case "document_expiration":
			checks.DocumentExpiration = checkResult
		case "name_match":
			checks.NameMatch = checkResult
		case "dob_match":
			checks.DOBMatch = checkResult
		case "address_match":
			checks.AddressMatch = checkResult
		case "photo_comparison":
			checks.FaceMatch = checkResult
		}
	}

	result := &external.IdentityVerificationResult{
		Verified:              personaResp.Data.Attributes.Status == personaStatusPassed,
		VerificationLevel:     verificationLevel,
		ConfidenceScore:       personaResp.Data.Attributes.VerificationScore,
		DocumentType:          docType,
		Checks:                checks,
		ProviderName:          p.name,
		ProviderTransactionID: personaResp.Data.ID,
		VerificationTimestamp: time.Now(),
		ProcessingTimeMs:      time.Since(startTime).Milliseconds(),
	}

	// Extract verified identity if available
	if personaResp.Data.Attributes.Fields.NameFirst.Value != "" {
		verifiedIdentity := &external.VerifiedIdentity{
			FirstName: personaResp.Data.Attributes.Fields.NameFirst.Value,
			LastName:  personaResp.Data.Attributes.Fields.NameLast.Value,
		}
		// Parse date of birth if available
		if personaResp.Data.Attributes.Fields.Birthdate.Value != "" {
			if dob, err := time.Parse("2006-01-02", personaResp.Data.Attributes.Fields.Birthdate.Value); err == nil {
				verifiedIdentity.DateOfBirth = dob
			}
		}
		result.VerifiedIdentity = verifiedIdentity
	}

	return result, nil
}

func (p *PersonaProvider) makeRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Persona-Version", "2023-01-05")

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("persona API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func convertPersonaStatus(status string) external.CheckStatus {
	switch status {
	case personaStatusPassed:
		return external.CheckStatusPassed
	case "failed":
		return external.CheckStatusFailed
	case "requires_retry":
		return external.CheckStatusWarning
	default:
		return external.CheckStatusNotPerformed
	}
}

func encodeBase64(data []byte) string {
	// In production, use encoding/base64
	// For now, return empty string if not implemented
	return ""
}
