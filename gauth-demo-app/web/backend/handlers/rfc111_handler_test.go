package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

func setupRFC111TestRouter(t *testing.T) (*gin.Engine, *RFC111Handler) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test configuration
	config := viper.New()
	config.Set("redis.addr", "localhost:6379")
	config.Set("redis.password", "")
	config.Set("redis.db", 0)
	config.Set("auth.jwt_secret", "test_secret_key")
	config.Set("auth.token_expiration", "3600s")
	config.Set("audit.log_level", "info")
	config.Set("audit.storage", "memory")

	// Create test logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce test output

	// Create RFC111 handler
	handler, err := NewRFC111Handler(config, logger)
	require.NoError(t, err, "Failed to create RFC111 handler")

	// Setup router
	router := gin.New()
	handler.RegisterRoutes(router)

	return router, handler
}

func TestRFC111Handler_ProcessRFC111Authorization(t *testing.T) {
	router, _ := setupRFC111TestRouter(t)

	tests := []struct {
		name           string
		payload        interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "Valid RFC111 Authorization Request",
			payload: services.RFC111AuthorizationRequest{
				ClientID:              "demo_ai_client",
				ResponseType:          "code",
				Scope:                 []string{"ai_power_of_attorney", "legal_framework", "financial_transactions"},
				RedirectURI:           "http://localhost:3000/callback",
				Jurisdiction:          "US",
				LegalBasis:            "power_of_attorney_act_2024",
				ComplianceRequirement: "rfc111_compliant",
				VerificationLevel:     "highest",
				PowerOfAttorney: &services.RFC111PowerOfAttorney{
					ID:               "poa_123",
					PrincipalID:      "user123",
					AgentID:          "ai_assistant_v2",
					PowerType:        "financial_transactions",
					Scope:            []string{"sign_contracts", "manage_investments", "authorize_payments"},
					EffectiveDate:    time.Now().Add(-time.Hour),           // Effective 1 hour ago
					ExpirationDate:   time.Now().Add(time.Hour * 24 * 365), // Expires in 1 year
					ComplianceStatus: "compliant",
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "code")
				assert.Contains(t, response, "legal_validation")
				assert.Contains(t, response, "compliance_status")

				// Check code format
				code, exists := response["code"].(string)
				assert.True(t, exists)
				assert.Contains(t, code, "rfc111_auth_")

				// Check legal validation
				legalValidation, exists := response["legal_validation"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, true, legalValidation["valid"])
				assert.Equal(t, "US", legalValidation["jurisdiction_id"])

				// Check compliance status
				complianceStatus, exists := response["compliance_status"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "compliant", complianceStatus["status"])
			},
		},
		{
			name: "RFC111 Authorization with AI Agent Context",
			payload: services.RFC111AuthorizationRequest{
				ClientID:              "ai_agent_financial",
				ResponseType:          "code",
				Scope:                 []string{"ai_power_of_attorney", "portfolio_management"},
				RedirectURI:           "http://localhost:3000/callback",
				Jurisdiction:          "US",
				LegalBasis:            "corporate_power_of_attorney_act_2024",
				ComplianceRequirement: "ai_agent",
				VerificationLevel:     "enhanced",
				DelegationContext: &services.DelegationContext{
					PrincipalID:     "cfo_jane_smith",
					DelegationType:  "corporate_financial_authority",
					DelegationScope: []string{"portfolio_management", "risk_assessment"},
					ChainDepth:      1,
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "code")
				assert.Contains(t, response, "compliance_status")

				// Check compliance for AI agent
				complianceStatus, exists := response["compliance_status"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "compliant", complianceStatus["status"])
				assert.Contains(t, complianceStatus["compliance_level"].(string), "rfc111")
			},
		},
		{
			name: "RFC111 Authorization with Geographic Restrictions",
			payload: services.RFC111AuthorizationRequest{
				ClientID:              "regional_ai_client",
				ResponseType:          "code",
				Scope:                 []string{"regional_authorization", "geographic_compliance"},
				RedirectURI:           "http://localhost:3000/callback",
				Jurisdiction:          "EU",
				LegalBasis:            "eu_power_of_attorney_directive",
				ComplianceRequirement: "geographic_restricted",
				VerificationLevel:     "highest",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "legal_validation")

				legalValidation, exists := response["legal_validation"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "EU", legalValidation["jurisdiction_id"])
			},
		},
		{
			name:           "Invalid Request - Missing Required Fields",
			payload:        map[string]interface{}{"client_id": "test_client"},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "error")
				assert.Equal(t, "invalid_request", response["error"])
			},
		},
		{
			name: "Invalid Request - Empty Scope",
			payload: services.RFC111AuthorizationRequest{
				ClientID:     "test_client",
				ResponseType: "code",
				Scope:        []string{},
				RedirectURI:  "http://localhost:3000/callback",
				Jurisdiction: "US",
				LegalBasis:   "test_basis",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal payload to JSON
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/rfc111/authorize", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run custom checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestRFC111Handler_ExchangeAuthorizationCode(t *testing.T) {
	router, _ := setupRFC111TestRouter(t)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "Valid Token Exchange",
			payload: map[string]interface{}{
				"grant_type":    "authorization_code",
				"code":          "rfc111_auth_123456789",
				"redirect_uri":  "http://localhost:3000/callback",
				"client_id":     "demo_ai_client",
				"client_secret": "demo_secret",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "access_token")
				assert.Contains(t, response, "token_type")
				assert.Contains(t, response, "expires_in")
				assert.Contains(t, response, "refresh_token")
				assert.Contains(t, response, "scope")
				assert.Contains(t, response, "compliance_level")

				// Check token format
				accessToken := response["access_token"].(string)
				assert.Contains(t, accessToken, "enhanced_token_")

				// Check token type
				assert.Equal(t, "Bearer", response["token_type"])

				// Check scope includes RFC111 compliance
				scope := response["scope"].(string)
				assert.Contains(t, scope, "rfc111_compliant")
				assert.Contains(t, scope, "power_of_attorney")

				// Check compliance level
				complianceLevel := response["compliance_level"].(string)
				assert.Equal(t, "rfc111_rfc115_full", complianceLevel)
			},
		},
		{
			name: "Invalid Grant Type",
			payload: map[string]interface{}{
				"grant_type":   "invalid_grant",
				"code":         "test_code",
				"redirect_uri": "http://localhost:3000/callback",
				"client_id":    "test_client",
			},
			expectedStatus: http.StatusOK, // Current implementation accepts any grant type for demo
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "access_token")
			},
		},
		{
			name:           "Missing Required Fields",
			payload:        map[string]interface{}{"grant_type": "authorization_code"},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "error")
				assert.Equal(t, "invalid_request", response["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal payload to JSON
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/rfc111/token", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run custom checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestRFC111Handler_GetLegalFrameworkInfo(t *testing.T) {
	router, _ := setupRFC111TestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Get General Legal Framework Info",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "supported_jurisdictions")
				assert.Contains(t, response, "legal_bases")
				assert.Contains(t, response, "compliance_levels")
				assert.Contains(t, response, "verification_methods")

				// Check supported jurisdictions
				jurisdictions, exists := response["supported_jurisdictions"].([]interface{})
				assert.True(t, exists)
				assert.Contains(t, jurisdictions, "US")
				assert.Contains(t, jurisdictions, "EU")

				// Check legal bases
				legalBases, exists := response["legal_bases"].([]interface{})
				assert.True(t, exists)
				assert.Contains(t, legalBases, "corporate_power_of_attorney")
				assert.Contains(t, legalBases, "ai_delegation_authority")

				// Check compliance levels
				complianceLevels, exists := response["compliance_levels"].([]interface{})
				assert.True(t, exists)
				assert.Contains(t, complianceLevels, "rfc111_compliant")
			},
		},
		{
			name:           "Get Jurisdiction-Specific Info",
			queryParams:    "?jurisdiction=US",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "jurisdiction_specific")

				jurisdictionInfo, exists := response["jurisdiction_specific"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "US", jurisdictionInfo["jurisdiction"])
				assert.Contains(t, jurisdictionInfo, "regulatory_context")
				assert.Contains(t, jurisdictionInfo, "required_documents")
				assert.Contains(t, jurisdictionInfo, "attestation_requirements")

				// Check attestation requirements
				attestationReqs, exists := jurisdictionInfo["attestation_requirements"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "enhanced", attestationReqs["minimum_level"])
				assert.Equal(t, float64(2), attestationReqs["required_attesters"])
			},
		},
		{
			name:           "Get EU Jurisdiction Info",
			queryParams:    "?jurisdiction=EU",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				jurisdictionInfo, exists := response["jurisdiction_specific"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "EU", jurisdictionInfo["jurisdiction"])

				regulatoryContext := jurisdictionInfo["regulatory_context"].(string)
				assert.Equal(t, "EU_legal_framework", regulatoryContext)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			url := "/api/v1/rfc111/legal-framework" + tt.queryParams
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run custom checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestRFC111Handler_ValidateLegalFramework(t *testing.T) {
	router, _ := setupRFC111TestRouter(t)

	tests := []struct {
		name           string
		payload        services.LegalFrameworkRequest
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "Valid Legal Framework Validation",
			payload: services.LegalFrameworkRequest{
				ClientID:     "demo_client",
				Action:       "authorize",
				Scope:        []string{"power_of_attorney", "legal_framework"},
				Jurisdiction: "US",
				Timestamp:    time.Now(),
				Metadata: map[string]interface{}{
					"legal_basis":            "corporate_power_of_attorney",
					"verification_level":     "highest",
					"compliance_requirement": "rfc111_compliant",
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["valid"])
				assert.Equal(t, "US", response["jurisdiction_id"])
				assert.Equal(t, "corporate_power_of_attorney", response["legal_basis"])
				assert.Equal(t, "rfc111_compliant", response["compliance_level"])
				assert.Contains(t, response, "validated_at")
				assert.Contains(t, response, "validation_id")
				assert.Equal(t, "jurisdiction_US_compliant", response["regulatory_context"])
			},
		},
		{
			name: "EU Legal Framework Validation",
			payload: services.LegalFrameworkRequest{
				ClientID:     "eu_client",
				Action:       "authorize",
				Scope:        []string{"gdpr_compliant", "eu_power_of_attorney"},
				Jurisdiction: "EU",
				Timestamp:    time.Now(),
				Metadata: map[string]interface{}{
					"legal_basis":            "eu_power_directive",
					"verification_level":     "enhanced",
					"compliance_requirement": "gdpr_rfc111_compliant",
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["valid"])
				assert.Equal(t, "EU", response["jurisdiction_id"])
				assert.Equal(t, "eu_power_directive", response["legal_basis"])
				assert.Equal(t, "jurisdiction_EU_compliant", response["regulatory_context"])
			},
		},
		{
			name:           "Invalid Request Format",
			payload:        services.LegalFrameworkRequest{},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "error")
				assert.Equal(t, "invalid_request", response["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal payload to JSON
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/rfc111/legal-framework/validate", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run custom checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestRFC111Handler_CreateAdvancedDelegation(t *testing.T) {
	router, _ := setupRFC111TestRouter(t)

	tests := []struct {
		name           string
		payload        services.DelegationRequest
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "Valid Advanced Delegation Creation",
			payload: services.DelegationRequest{
				PrincipalID: "corp_ceo_123",
				DelegateID:  "ai_agent_v2",
				PowerType:   "advanced_financial_delegation",
				Scope:       []string{"contract_signing", "investment_decisions", "regulatory_compliance"},
				Restrictions: &services.Restrictions{
					AmountLimits:   map[string]float64{"USD": 100000},
					GeoConstraints: []string{"US", "EU", "CA"},
					TimeWindows: []services.TimeWindow{
						{
							StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
							EndTime:   time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
						},
					},
				},
				AttestationRequirement: &services.AttestationRequirement{
					Type:           "digital_signature",
					Level:          "enhanced",
					MultiSignature: true,
					Attesters:      []string{"notary_public", "legal_counsel"},
				},
				ValidityPeriod: &services.ValidityPeriod{
					StartTime: time.Now(),
					EndTime:   time.Now().Add(time.Hour * 24 * 90), // 90 days
				},
				Jurisdiction: "US",
				LegalBasis:   "corporate_power_delegation_act_2024",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "delegation_id")
				assert.Contains(t, response, "enhanced_token")
				assert.Contains(t, response, "compliance_status")
				assert.Contains(t, response, "verification_proof")
				assert.Contains(t, response, "status")
				assert.Equal(t, "active", response["status"])

				// Check enhanced token
				enhancedToken, exists := response["enhanced_token"].(map[string]interface{})
				assert.True(t, exists)
				assert.Contains(t, enhancedToken, "id")
				assert.Equal(t, "delegated_bearer", enhancedToken["type"])

				// Check compliance status
				complianceStatus, exists := response["compliance_status"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "compliant", complianceStatus["status"])

				// Check verification proof
				verificationProof, exists := response["verification_proof"].(map[string]interface{})
				assert.True(t, exists)
				assert.Contains(t, verificationProof, "proof_id")
				assert.Equal(t, "highest", verificationProof["trust_level"])
			},
		},
		{
			name: "Advanced Delegation with Multi-Signature",
			payload: services.DelegationRequest{
				PrincipalID: "board_chair_456",
				DelegateID:  "governance_ai",
				PowerType:   "governance_authority",
				Scope:       []string{"board_decisions", "shareholder_communications"},
				AttestationRequirement: &services.AttestationRequirement{
					Type:           "multi_signature",
					Level:          "highest",
					MultiSignature: true,
					Attesters:      []string{"board_secretary", "legal_counsel", "compliance_officer"},
				},
				ValidityPeriod: &services.ValidityPeriod{
					StartTime: time.Now(),
					EndTime:   time.Now().Add(time.Hour * 24 * 30), // 30 days
				},
				Jurisdiction: "US",
				LegalBasis:   "corporate_governance_act_2024",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "attestations")

				attestations, exists := response["attestations"].([]interface{})
				assert.True(t, exists)
				assert.Greater(t, len(attestations), 0)

				// Check first attestation
				if len(attestations) > 0 {
					attestation := attestations[0].(map[string]interface{})
					assert.Contains(t, attestation, "type")
					assert.Contains(t, attestation, "trust_level")
				}
			},
		},
		{
			name:           "Invalid Delegation Request",
			payload:        services.DelegationRequest{},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "error")
				assert.Equal(t, "invalid_request", response["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal payload to JSON
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/rfc115/delegation", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run custom checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestRFC111Handler_GetDelegation(t *testing.T) {
	router, _ := setupRFC111TestRouter(t)

	tests := []struct {
		name           string
		delegationID   string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Get Existing Delegation",
			delegationID:   "delegation_123",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "delegation_123", response["delegation_id"])
				assert.Contains(t, response, "principal_id")
				assert.Contains(t, response, "delegate_id")
				assert.Contains(t, response, "power_type")
				assert.Contains(t, response, "scope")
				assert.Contains(t, response, "status")
				assert.Contains(t, response, "compliance_status")

				// Check compliance status
				complianceStatus, exists := response["compliance_status"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "compliant", complianceStatus["status"])
				assert.Equal(t, "rfc111_rfc115_full", complianceStatus["compliance_level"])

				// Check restrictions
				restrictions, exists := response["restrictions"].(map[string]interface{})
				assert.True(t, exists)
				assert.Contains(t, restrictions, "amount_limits")
			},
		},
		{
			name:           "Get Non-existent Delegation",
			delegationID:   "nonexistent",
			expectedStatus: http.StatusOK, // Current implementation returns mock data
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "nonexistent", response["delegation_id"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			url := fmt.Sprintf("/api/v1/rfc115/delegation/%s", tt.delegationID)
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run custom checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestRFC111Handler_GetComplianceStatus(t *testing.T) {
	router, _ := setupRFC111TestRouter(t)

	tests := []struct {
		name           string
		clientID       string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Get Compliance Status for Active Client",
			clientID:       "demo_ai_client",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "demo_ai_client", response["client_id"])
				assert.Contains(t, response, "compliance_status")
				assert.Contains(t, response, "active_delegations")
				assert.Contains(t, response, "active_tokens")
				assert.Contains(t, response, "risk_assessment")
				assert.Contains(t, response, "audit_summary")

				// Check overall compliance status
				complianceStatus, exists := response["compliance_status"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "compliant", complianceStatus["overall_status"])

				// Check RFC111 compliance
				rfc111Compliance, exists := complianceStatus["rfc111_compliance"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "compliant", rfc111Compliance["status"])
				assert.Equal(t, true, rfc111Compliance["legal_framework_validated"])
				assert.Equal(t, true, rfc111Compliance["power_of_attorney_verified"])

				// Check RFC115 compliance
				rfc115Compliance, exists := complianceStatus["rfc115_compliance"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "compliant", rfc115Compliance["status"])
				assert.Equal(t, true, rfc115Compliance["attestation_verified"])

				// Check risk assessment
				riskAssessment, exists := response["risk_assessment"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "low", riskAssessment["risk_level"])

				// Check audit summary
				auditSummary, exists := response["audit_summary"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, float64(157), auditSummary["total_events"])
				assert.Equal(t, float64(0), auditSummary["compliance_violations"])
			},
		},
		{
			name:           "Get Compliance Status for AI Agent",
			clientID:       "ai_agent_financial",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "ai_agent_financial", response["client_id"])

				complianceStatus, exists := response["compliance_status"].(map[string]interface{})
				assert.True(t, exists)
				assert.Equal(t, "compliant", complianceStatus["overall_status"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			url := fmt.Sprintf("/api/v1/compliance/status/%s", tt.clientID)
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run custom checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

// Test unimplemented endpoints return proper error responses
func TestRFC111Handler_UnimplementedEndpoints(t *testing.T) {
	router, _ := setupRFC111TestRouter(t)

	unimplementedEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/api/v1/rfc115/delegation/test_id"},
		{http.MethodDelete, "/api/v1/rfc115/delegation/test_id"},
		{http.MethodPost, "/api/v1/rfc115/attestation"},
		{http.MethodGet, "/api/v1/rfc115/attestation/test_id"},
		{http.MethodPost, "/api/v1/rfc115/verification"},
		{http.MethodPost, "/api/v1/tokens/enhanced/test_id/refresh"},
		{http.MethodDelete, "/api/v1/tokens/enhanced/test_id"},
		{http.MethodPost, "/api/v1/tokens/enhanced/test_id/delegate"},
		{http.MethodGet, "/api/v1/tokens/enhanced/test_id/chain"},
	}

	for _, endpoint := range unimplementedEndpoints {
		t.Run(fmt.Sprintf("%s %s", endpoint.method, endpoint.path), func(t *testing.T) {
			var req *http.Request
			if endpoint.method == http.MethodPost || endpoint.method == http.MethodPut {
				req = httptest.NewRequest(endpoint.method, endpoint.path, bytes.NewBufferString("{}"))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(endpoint.method, endpoint.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should return not implemented
			assert.Equal(t, http.StatusNotImplemented, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "error")
			assert.Equal(t, "Method not implemented yet", response["error"])
		})
	}
}
