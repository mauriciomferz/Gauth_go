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
	truliooStatusMatch = "match"
)

// TruliooProvider implements USIdentityAPIProvider for Trulioo GlobalGateway API
// API Documentation: https://developer.trulioo.com/docs
type TruliooProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	name       string
}

// TruliooConfig holds configuration for Trulioo provider
type TruliooConfig struct {
	APIKey     string
	BaseURL    string // Default: https://gateway.trulioo.com
	Timeout    time.Duration
	MaxRetries int
}

// NewTruliooProvider creates a new Trulioo API provider
func NewTruliooProvider(config *TruliooConfig) *TruliooProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://gateway.trulioo.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &TruliooProvider{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		name: "trulioo",
	}
}

// =============================================================================
// API Request/Response Structures
// =============================================================================

// TruliooVerifyRequest is the request sent to Trulioo API
type TruliooVerifyRequest struct {
	AcceptTruliooTermsAndConditions bool              `json:"AcceptTruliooTermsAndConditions"`
	Demo                            bool              `json:"Demo,omitempty"`
	CleansedAddress                 bool              `json:"CleansedAddress,omitempty"`
	ConfigurationName               string            `json:"ConfigurationName"`
	CountryCode                     string            `json:"CountryCode"`
	DataFields                      TruliooDataFields `json:"DataFields"`
}

type TruliooDataFields struct {
	PersonInfo    TruliooPersonInfo    `json:"PersonInfo,omitempty"`
	Location      TruliooLocation      `json:"Location,omitempty"`
	Communication TruliooCommunication `json:"Communication,omitempty"`
	Document      TruliooDocument      `json:"Document,omitempty"`
	NationalIds   []TruliooNationalID  `json:"NationalIds,omitempty"`
}

type TruliooPersonInfo struct {
	FirstGivenName string `json:"FirstGivenName,omitempty"`
	FirstSurName   string `json:"FirstSurName,omitempty"`
	DayOfBirth     int    `json:"DayOfBirth,omitempty"`
	MonthOfBirth   int    `json:"MonthOfBirth,omitempty"`
	YearOfBirth    int    `json:"YearOfBirth,omitempty"`
}

type TruliooLocation struct {
	BuildingNumber    string `json:"BuildingNumber,omitempty"`
	StreetName        string `json:"StreetName,omitempty"`
	City              string `json:"City,omitempty"`
	StateProvinceCode string `json:"StateProvinceCode,omitempty"`
	PostalCode        string `json:"PostalCode,omitempty"`
	Country           string `json:"Country,omitempty"`
}

type TruliooCommunication struct {
	EmailAddress string `json:"EmailAddress,omitempty"`
	Telephone    string `json:"Telephone,omitempty"`
}

type TruliooDocument struct {
	DocumentType   string `json:"DocumentType,omitempty"`
	DocumentNumber string `json:"DocumentNumber,omitempty"`
	ExpirationDate string `json:"ExpirationDate,omitempty"`
	IssuingCountry string `json:"IssuingCountry,omitempty"`
}

type TruliooNationalID struct {
	Type   string `json:"Type,omitempty"`
	Number string `json:"Number,omitempty"`
}

// TruliooVerifyResponse is the response from Trulioo API
type TruliooVerifyResponse struct {
	TransactionID string            `json:"TransactionID"`
	RecordStatus  string            `json:"RecordStatus"`
	MatchStatus   string            `json:"MatchStatus,omitempty"`
	Errors        []TruliooError    `json:"Errors,omitempty"`
	InputFields   TruliooDataFields `json:"InputFields,omitempty"`
	Record        TruliooRecord     `json:"Record,omitempty"`
	CountryCode   string            `json:"CountryCode"`
}

type TruliooError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type TruliooRecord struct {
	TransactionRecordID string                    `json:"TransactionRecordID"`
	RecordStatus        string                    `json:"RecordStatus"`
	DatasourceResults   []TruliooDataSourceResult `json:"DatasourceResults,omitempty"`
}

type TruliooDataSourceResult struct {
	DatasourceName   string               `json:"DatasourceName"`
	DatasourceFields []TruliooFieldResult `json:"DatasourceFields,omitempty"`
	AppendedFields   []TruliooFieldResult `json:"AppendedFields,omitempty"`
	Errors           []TruliooError       `json:"Errors,omitempty"`
	FieldGroups      []TruliooFieldGroup  `json:"FieldGroups,omitempty"`
}

type TruliooFieldResult struct {
	FieldName string `json:"FieldName"`
	Status    string `json:"Status"`
}

type TruliooFieldGroup struct {
	GroupName string               `json:"GroupName"`
	Fields    []TruliooFieldResult `json:"Fields,omitempty"`
}

// =============================================================================
// USIdentityAPIProvider Implementation
// =============================================================================

// VerifyDocument implements USIdentityAPIProvider
func (t *TruliooProvider) VerifyDocument(ctx context.Context, req interface{}) (*external.IdentityVerificationResult, error) {
	switch r := req.(type) {
	case *external.PassportVerificationRequest:
		return t.verifyPassport(ctx, r)
	case *external.DLVerificationRequest:
		return t.verifyDriverLicense(ctx, r)
	case *external.StateIDVerificationRequest:
		return t.verifyStateID(ctx, r)
	default:
		return nil, fmt.Errorf("unsupported request type: %T", req)
	}
}

// ValidateSSN implements USIdentityAPIProvider
func (t *TruliooProvider) ValidateSSN(ctx context.Context, req *external.SSNValidationRequest) (*external.SSNValidationResult, error) {
	startTime := time.Now()

	// Build Trulioo request
	truliooReq := TruliooVerifyRequest{
		AcceptTruliooTermsAndConditions: true,
		ConfigurationName:               "Identity Verification",
		CountryCode:                     "US",
		DataFields: TruliooDataFields{
			PersonInfo: TruliooPersonInfo{
				FirstGivenName: req.FirstName,
				FirstSurName:   req.LastName,
				DayOfBirth:     req.DateOfBirth.Day(),
				MonthOfBirth:   int(req.DateOfBirth.Month()),
				YearOfBirth:    req.DateOfBirth.Year(),
			},
			NationalIds: []TruliooNationalID{
				{
					Type:   "SSN",
					Number: req.SSN,
				},
			},
		},
	}

	// Make API request
	resp, err := t.makeRequest(ctx, "POST", "/v2/verifications/verify", truliooReq)
	if err != nil {
		return nil, fmt.Errorf("trulioo API request failed: %w", err)
	}

	// Parse response
	var truliooResp TruliooVerifyResponse
	if err := json.Unmarshal(resp, &truliooResp); err != nil {
		return nil, fmt.Errorf("failed to parse trulioo response: %w", err)
	}

	// Convert to SSNValidationResult
	result := &external.SSNValidationResult{
		Valid:                 truliooResp.RecordStatus == truliooStatusMatch,
		ValidationLevel:       req.ValidationLevel,
		FormatValid:           true, // Trulioo validates format
		ProviderName:          t.name,
		ProviderTransactionID: truliooResp.TransactionID,
		ProcessingTimeMs:      time.Since(startTime).Milliseconds(),
		ValidationTimestamp:   time.Now(),
	}

	// Calculate confidence score based on match status
	if truliooResp.RecordStatus == truliooStatusMatch {
		result.ConfidenceScore = 0.95
	} else if truliooResp.RecordStatus == "nomatch" {
		result.ConfidenceScore = 0.1
	} else {
		result.ConfidenceScore = 0.5
	}

	// Check individual fields from datasource results
	for _, datasource := range truliooResp.Record.DatasourceResults {
		for _, field := range datasource.DatasourceFields {
			switch field.FieldName {
			case "FirstGivenName", "FirstSurName":
				if result.NameMatch == nil {
					result.NameMatch = &external.NameMatchResult{
						Matched: field.Status == truliooStatusMatch,
						Score:   calculateMatchScore(field.Status),
					}
				}
			case "DayOfBirth", "MonthOfBirth", "YearOfBirth":
				if result.DOBMatch == nil {
					result.DOBMatch = &external.DOBMatchResult{
						Matched: field.Status == truliooStatusMatch,
					}
				}
			}
		}
	}

	// Mask SSN in response
	if len(req.SSN) >= 9 {
		result.SSN = fmt.Sprintf("XXX-XX-%s", req.SSN[len(req.SSN)-4:])
	}

	// Add errors if any
	if len(truliooResp.Errors) > 0 {
		for _, err := range truliooResp.Errors {
			result.Errors = append(result.Errors, external.VerificationError{
				Code:     err.Code,
				Message:  err.Message,
				Severity: "error",
			})
		}
	}

	return result, nil
}

// GetProviderName implements USIdentityAPIProvider
func (t *TruliooProvider) GetProviderName() string {
	return t.name
}

// GetSupportedDocumentTypes implements USIdentityAPIProvider
func (t *TruliooProvider) GetSupportedDocumentTypes() []external.DocumentType {
	return []external.DocumentType{
		external.DocumentTypePassport,
		external.DocumentTypeDriverLicense,
		external.DocumentTypeStateID,
	}
}

// =============================================================================
// Private Methods
// =============================================================================

func (t *TruliooProvider) verifyPassport(ctx context.Context, req *external.PassportVerificationRequest) (*external.IdentityVerificationResult, error) {
	startTime := time.Now()

	// Build Trulioo request
	truliooReq := TruliooVerifyRequest{
		AcceptTruliooTermsAndConditions: true,
		ConfigurationName:               "Identity Verification",
		CountryCode:                     req.Nationality,
		DataFields: TruliooDataFields{
			PersonInfo: TruliooPersonInfo{
				FirstGivenName: req.FirstName,
				FirstSurName:   req.LastName,
				DayOfBirth:     req.DateOfBirth.Day(),
				MonthOfBirth:   int(req.DateOfBirth.Month()),
				YearOfBirth:    req.DateOfBirth.Year(),
			},
			Document: TruliooDocument{
				DocumentType:   "Passport",
				DocumentNumber: req.PassportNumber,
				ExpirationDate: req.ExpirationDate.Format("2006-01-02"),
				IssuingCountry: req.Nationality,
			},
		},
	}

	// Make API request
	resp, err := t.makeRequest(ctx, "POST", "/v2/verifications/verify", truliooReq)
	if err != nil {
		return nil, fmt.Errorf("trulioo API request failed: %w", err)
	}

	// Parse and convert response
	return t.parseVerificationResponse(resp, external.DocumentTypePassport, "", startTime)
}

func (t *TruliooProvider) verifyDriverLicense(ctx context.Context, req *external.DLVerificationRequest) (*external.IdentityVerificationResult, error) {
	startTime := time.Now()

	// Build Trulioo request
	truliooReq := TruliooVerifyRequest{
		AcceptTruliooTermsAndConditions: true,
		ConfigurationName:               "Identity Verification",
		CountryCode:                     "US",
		DataFields: TruliooDataFields{
			PersonInfo: TruliooPersonInfo{
				FirstGivenName: req.FirstName,
				FirstSurName:   req.LastName,
				DayOfBirth:     req.DateOfBirth.Day(),
				MonthOfBirth:   int(req.DateOfBirth.Month()),
				YearOfBirth:    req.DateOfBirth.Year(),
			},
			Location: TruliooLocation{
				StateProvinceCode: req.State,
				Country:           "US",
			},
			Document: TruliooDocument{
				DocumentType:   "DriversLicence",
				DocumentNumber: req.LicenseNumber,
				ExpirationDate: req.ExpirationDate.Format("2006-01-02"),
				IssuingCountry: "US",
			},
		},
	}

	// Add address if provided
	if req.Address != nil {
		truliooReq.DataFields.Location.StreetName = req.Address.StreetAddress1
		truliooReq.DataFields.Location.City = req.Address.City
		truliooReq.DataFields.Location.PostalCode = req.Address.ZipCode
	}

	// Make API request
	resp, err := t.makeRequest(ctx, "POST", "/v2/verifications/verify", truliooReq)
	if err != nil {
		return nil, fmt.Errorf("trulioo API request failed: %w", err)
	}

	// Parse and convert response
	result, err := t.parseVerificationResponse(resp, external.DocumentTypeDriverLicense, req.State, startTime)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *TruliooProvider) verifyStateID(ctx context.Context, req *external.StateIDVerificationRequest) (*external.IdentityVerificationResult, error) {
	startTime := time.Now()

	// Build Trulioo request
	truliooReq := TruliooVerifyRequest{
		AcceptTruliooTermsAndConditions: true,
		ConfigurationName:               "Identity Verification",
		CountryCode:                     "US",
		DataFields: TruliooDataFields{
			PersonInfo: TruliooPersonInfo{
				FirstGivenName: req.FirstName,
				FirstSurName:   req.LastName,
				DayOfBirth:     req.DateOfBirth.Day(),
				MonthOfBirth:   int(req.DateOfBirth.Month()),
				YearOfBirth:    req.DateOfBirth.Year(),
			},
			Location: TruliooLocation{
				StateProvinceCode: req.State,
				Country:           "US",
			},
			Document: TruliooDocument{
				DocumentType:   "NationalID",
				DocumentNumber: req.IDNumber,
				ExpirationDate: req.ExpirationDate.Format("2006-01-02"),
				IssuingCountry: "US",
			},
		},
	}

	// Add address if provided
	if req.Address != nil {
		truliooReq.DataFields.Location.StreetName = req.Address.StreetAddress1
		truliooReq.DataFields.Location.City = req.Address.City
		truliooReq.DataFields.Location.PostalCode = req.Address.ZipCode
	}

	// Make API request
	resp, err := t.makeRequest(ctx, "POST", "/v2/verifications/verify", truliooReq)
	if err != nil {
		return nil, fmt.Errorf("trulioo API request failed: %w", err)
	}

	// Parse and convert response
	result, err := t.parseVerificationResponse(resp, external.DocumentTypeStateID, req.State, startTime)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *TruliooProvider) parseVerificationResponse(
	respBody []byte,
	docType external.DocumentType,
	state string,
	startTime time.Time,
) (*external.IdentityVerificationResult, error) {
	var truliooResp TruliooVerifyResponse
	if err := json.Unmarshal(respBody, &truliooResp); err != nil {
		return nil, fmt.Errorf("failed to parse trulioo response: %w", err)
	}

	// Determine verification status
	verified := truliooResp.RecordStatus == truliooStatusMatch

	// Determine verification level
	verificationLevel := external.VerificationLevelBasic
	if verified {
		verificationLevel = external.VerificationLevelStandard
	}

	// Build verification checks from datasource results
	checks := &external.VerificationChecks{}
	confidenceScore := 0.0

	for _, datasource := range truliooResp.Record.DatasourceResults {
		for _, field := range datasource.DatasourceFields {
			score := calculateMatchScore(field.Status)
			confidenceScore += score

			checkResult := &external.CheckResult{
				Status: convertTruliooStatus(field.Status),
				Score:  score,
			}

			switch field.FieldName {
			case "DocumentNumber":
				checks.DocumentAuthenticity = checkResult
			case "ExpirationDate":
				checks.DocumentExpiration = checkResult
			case "FirstGivenName", "FirstSurName":
				if checks.NameMatch == nil {
					checks.NameMatch = checkResult
				}
			case "DayOfBirth", "MonthOfBirth", "YearOfBirth":
				if checks.DOBMatch == nil {
					checks.DOBMatch = checkResult
				}
			case "StreetName", "City", "PostalCode":
				if checks.AddressMatch == nil {
					checks.AddressMatch = checkResult
				}
			}
		}
	}

	// Normalize confidence score (0-1 range)
	if len(truliooResp.Record.DatasourceResults) > 0 {
		totalFields := 0
		for _, datasource := range truliooResp.Record.DatasourceResults {
			totalFields += len(datasource.DatasourceFields)
		}
		if totalFields > 0 {
			confidenceScore = confidenceScore / float64(totalFields)
		}
	}

	result := &external.IdentityVerificationResult{
		Verified:              verified,
		VerificationLevel:     verificationLevel,
		ConfidenceScore:       confidenceScore,
		DocumentType:          docType,
		DocumentState:         state,
		Checks:                checks,
		ProviderName:          t.name,
		ProviderTransactionID: truliooResp.TransactionID,
		VerificationTimestamp: time.Now(),
		ProcessingTimeMs:      time.Since(startTime).Milliseconds(),
	}

	// Add errors if any
	if len(truliooResp.Errors) > 0 {
		for _, err := range truliooResp.Errors {
			result.Errors = append(result.Errors, external.VerificationError{
				Code:     err.Code,
				Message:  err.Message,
				Severity: "error",
			})
		}
	}

	// Extract verified identity from input fields
	if truliooResp.InputFields.PersonInfo.FirstGivenName != "" {
		result.VerifiedIdentity = &external.VerifiedIdentity{
			FirstName: truliooResp.InputFields.PersonInfo.FirstGivenName,
			LastName:  truliooResp.InputFields.PersonInfo.FirstSurName,
			DateOfBirth: time.Date(
				truliooResp.InputFields.PersonInfo.YearOfBirth,
				time.Month(truliooResp.InputFields.PersonInfo.MonthOfBirth),
				truliooResp.InputFields.PersonInfo.DayOfBirth,
				0, 0, 0, 0, time.UTC,
			),
		}
	}

	return result, nil
}

func (t *TruliooProvider) makeRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	url := t.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+t.apiKey) // Trulioo uses Basic auth with API key
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := t.httpClient.Do(req)
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
		return nil, fmt.Errorf("trulioo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func convertTruliooStatus(status string) external.CheckStatus {
	switch status {
	case truliooStatusMatch:
		return external.CheckStatusPassed
	case "nomatch":
		return external.CheckStatusFailed
	case "missing":
		return external.CheckStatusNotPerformed
	default:
		return external.CheckStatusWarning
	}
}

func calculateMatchScore(status string) float64 {
	switch status {
	case truliooStatusMatch:
		return 1.0
	case "nomatch":
		return 0.0
	case "missing":
		return 0.5
	default:
		return 0.5
	}
}
