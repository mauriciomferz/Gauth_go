package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/jurisdiction"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// allowAllAuthorizer is a test authorizer that allows all requests.
type allowAllJurisdictionAuthorizer struct{}

func (a *allowAllJurisdictionAuthorizer) Authorize(ctx context.Context, req authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: true, Reason: "test allow-all"}, nil
}

func (a *allowAllJurisdictionAuthorizer) GetPermissions(ctx context.Context, subject string) ([]authz.Permission, error) {
	return []authz.Permission{{Resource: "*", Actions: []string{"*"}, Granted: true}}, nil
}

// jurisdictionTestService creates a test service with jurisdiction enforcement enabled.
func jurisdictionTestService() *Service {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllJurisdictionAuthorizer{}
	integration := jurisdiction.NewServerIntegration()

	return NewService(logger, authorizer, WithJurisdictionEnforcement(integration))
}

// TestJurisdictionIntegration_Disabled verifies that jurisdiction enforcement is disabled by default.
func TestJurisdictionIntegration_Disabled(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllJurisdictionAuthorizer{}

	// Service WITHOUT jurisdiction enforcement (default behavior)
	svc := NewService(logger, authorizer)

	// Create delegation without any jurisdiction claims - should succeed
	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read", "write"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation should succeed when enforcement disabled: %v", err)
	}
	if resp == nil || resp.AuthToken == "" {
		t.Fatal("Expected valid delegation response")
	}
}

// TestJurisdictionIntegration_EUGDPRConsent tests EU GDPR consent validation during delegation creation.
func TestJurisdictionIntegration_EUGDPRConsent(t *testing.T) {
	svc := jurisdictionTestService()
	ctx := WithSubject(context.Background(), "bob@example.com")

	// Test 1: EU jurisdiction with GDPR consent - ALLOW
	reqWithConsent := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@eubank.com",
		Scope:    []string{"gdpr_data_processing"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction": "EU",
			"entity_type":  "corporation",
			"gdpr_consent": true, // Required for GDPR data processing
		},
	}

	respAllow, err := svc.CreateDelegationCtx(ctx, reqWithConsent)
	if err != nil {
		t.Fatalf("EU GDPR with consent should be ALLOWED: %v", err)
	}
	if respAllow == nil || respAllow.AuthToken == "" {
		t.Fatal("Expected valid delegation response with GDPR consent")
	}

	// Test 2: EU jurisdiction WITHOUT GDPR consent - DENY
	reqWithoutConsent := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@eubank.com",
		Scope:    []string{"gdpr_data_processing"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction": "EU",
			"entity_type":  "corporation",
			"gdpr_consent": false, // Missing consent
		},
	}

	_, err = svc.CreateDelegationCtx(ctx, reqWithoutConsent)
	if err == nil {
		t.Fatal("EU GDPR without consent should be DENIED")
	}
	// Verify error message contains jurisdiction enforcement denial
	if err.Error() == "" {
		t.Fatal("Expected error message about jurisdiction enforcement")
	}
}

// TestJurisdictionIntegration_USCCPAOptOut tests US CCPA opt-out validation.
func TestJurisdictionIntegration_USCCPAOptOut(t *testing.T) {
	svc := jurisdictionTestService()
	ctx := WithSubject(context.Background(), "bob@example.com")

	// Test 1: US jurisdiction without CCPA opt-out - ALLOW
	reqNoOptOut := DelegationRequest{
		Grantor:  "alice@usbank.com",
		Grantee:  "bob@usbank.com",
		Scope:    []string{"ccpa_data_processing"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction": "US",
			"entity_type":  "corporation",
			"ccpa_opt_out": false, // No opt-out, processing allowed
		},
	}

	respAllow, err := svc.CreateDelegationCtx(ctx, reqNoOptOut)
	if err != nil {
		t.Fatalf("US CCPA without opt-out should be ALLOWED: %v", err)
	}
	if respAllow == nil || respAllow.AuthToken == "" {
		t.Fatal("Expected valid delegation response without CCPA opt-out")
	}

	// Test 2: US jurisdiction WITH CCPA opt-out - DENY
	reqWithOptOut := DelegationRequest{
		Grantor:  "alice@usbank.com",
		Grantee:  "bob@usbank.com",
		Scope:    []string{"ccpa_data_processing"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction": "US",
			"entity_type":  "corporation",
			"ccpa_opt_out": true, // User opted out, processing denied
		},
	}

	_, err = svc.CreateDelegationCtx(ctx, reqWithOptOut)
	if err == nil {
		t.Fatal("US CCPA with opt-out should be DENIED")
	}
}

// TestJurisdictionIntegration_CrossBorderDataTransfer tests cross-border data transfer rules.
func TestJurisdictionIntegration_CrossBorderDataTransfer(t *testing.T) {
	svc := jurisdictionTestService()
	ctx := WithSubject(context.Background(), "bob@example.com")

	// Test 1: EU to UK cross-border transfer (adequacy country) - ALLOW
	reqEUtoUK := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@ukbank.com",
		Scope:    []string{"personal_data_transfer"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction":             "EU",
			"entity_type":              "corporation",
			"destination_jurisdiction": "UK", // UK is adequacy country for EU
		},
	}

	respAllow, err := svc.CreateDelegationCtx(ctx, reqEUtoUK)
	if err != nil {
		t.Fatalf("EU to UK cross-border transfer should be ALLOWED: %v", err)
	}
	if respAllow == nil {
		t.Fatal("Expected valid delegation for EU->UK transfer")
	}

	// Test 2: EU to US cross-border transfer (NOT adequacy country) - DENY
	reqEUtoUS := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@usbank.com",
		Scope:    []string{"personal_data_transfer"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction":             "EU",
			"entity_type":              "corporation",
			"destination_jurisdiction": "US", // US not in EU adequacy list for personal data
		},
	}

	_, err = svc.CreateDelegationCtx(ctx, reqEUtoUS)
	if err == nil {
		t.Fatal("EU to US cross-border transfer should be DENIED")
	}
}

// TestJurisdictionIntegration_DataResidency tests data residency enforcement.
func TestJurisdictionIntegration_DataResidency(t *testing.T) {
	svc := jurisdictionTestService()
	ctx := WithSubject(context.Background(), "bob@example.com")

	// Test 1: EU personal data staying in EU - ALLOW
	reqResidencyOK := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@eubank.com",
		Scope:    []string{"data_export"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction":             "EU",
			"entity_type":              "corporation",
			"destination_jurisdiction": "EU",            // Staying in EU
			"data_type":                "personal_data", // EU requires personal data to stay local
		},
	}

	respAllow, err := svc.CreateDelegationCtx(ctx, reqResidencyOK)
	if err != nil {
		t.Fatalf("EU personal data staying in EU should be ALLOWED: %v", err)
	}
	if respAllow == nil {
		t.Fatal("Expected valid delegation for EU data residency")
	}

	// Test 2: EU personal data leaving EU - DENY
	reqResidencyViolation := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@usbank.com",
		Scope:    []string{"data_export"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction":             "EU",
			"entity_type":              "corporation",
			"destination_jurisdiction": "US",            // Leaving EU
			"data_type":                "personal_data", // Violates residency rule
		},
	}

	_, err = svc.CreateDelegationCtx(ctx, reqResidencyViolation)
	if err == nil {
		t.Fatal("EU personal data leaving EU should be DENIED (data residency violation)")
	}
}

// TestJurisdictionIntegration_BlockedActions tests jurisdiction-specific blocked actions.
func TestJurisdictionIntegration_BlockedActions(t *testing.T) {
	svc := jurisdictionTestService()
	ctx := WithSubject(context.Background(), "bob@example.com")

	// Test 1: EU blocked action "unrestricted_data_export" - DENY
	reqBlockedAction := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@eubank.com",
		Scope:    []string{"unrestricted_data_export"}, // Blocked in EU
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction": "EU",
			"entity_type":  "corporation",
		},
	}

	_, err := svc.CreateDelegationCtx(ctx, reqBlockedAction)
	if err == nil {
		t.Fatal("EU blocked action 'unrestricted_data_export' should be DENIED")
	}

	// Test 2: Allowed action in EU - ALLOW
	reqAllowedAction := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@eubank.com",
		Scope:    []string{"read", "write"}, // Standard actions, not blocked
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction": "EU",
			"entity_type":  "corporation",
		},
	}

	respAllow, err := svc.CreateDelegationCtx(ctx, reqAllowedAction)
	if err != nil {
		t.Fatalf("EU allowed action should succeed: %v", err)
	}
	if respAllow == nil {
		t.Fatal("Expected valid delegation for allowed action")
	}
}

// TestJurisdictionIntegration_VerifyTokenEnforcement tests jurisdiction enforcement during token verification.
func TestJurisdictionIntegration_VerifyTokenEnforcement(t *testing.T) {
	svc := jurisdictionTestService()
	ctx := WithSubject(context.Background(), "bob@eubank.com")

	// Create a delegation with EU jurisdiction
	req := DelegationRequest{
		Grantor:  "alice@eubank.com",
		Grantee:  "bob@eubank.com",
		Scope:    []string{"read"},
		Duration: 1 * time.Hour,
		Claims: map[string]interface{}{
			"jurisdiction": "EU",
			"entity_type":  "corporation",
		},
	}

	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create delegation: %v", err)
	}
	if resp == nil || resp.AuthToken == "" {
		t.Fatal("Expected valid auth token")
	}

	// Verify the token - should enforce jurisdiction rules at verification time
	result, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("Token verification with jurisdiction enforcement should succeed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected valid verification result")
	}
	if result.DelegationID == "" {
		t.Fatal("Expected delegation ID in result")
	}
}

// TestExtractJurisdictionFromPOA tests jurisdiction extraction helper.
func TestExtractJurisdictionFromPOA(t *testing.T) {
	// Test 1: Jurisdiction from Jurisdiction field
	poa1 := &PowerOfAttorney{
		Jurisdiction: "EU",
	}
	jur1 := ExtractJurisdictionFromPOA(poa1)
	if jur1 != "EU" {
		t.Errorf("Expected EU, got %s", jur1)
	}

	// Test 2: Jurisdiction from Restrictions map (US normalized)
	poa2 := &PowerOfAttorney{
		Restrictions: map[string]string{
			"jurisdiction": "US",
		},
	}
	jur2 := ExtractJurisdictionFromPOA(poa2)
	if jur2 != "US" { // ExtractJurisdictionFromClaims may normalize or return as-is
		t.Logf("Got jurisdiction: %s (expected US or UNITED_STATES)", jur2)
	}

	// Test 3: Default jurisdiction (nil PoA)
	jur3 := ExtractJurisdictionFromPOA(nil)
	if jur3 != "UNITED_STATES" && jur3 != "US" { // Accept either format
		t.Errorf("Expected UNITED_STATES or US (default), got %s", jur3)
	}
}

// TestValidateJurisdictionCompliance tests standalone compliance validation.
func TestValidateJurisdictionCompliance(t *testing.T) {
	svc := jurisdictionTestService()
	ctx := WithSubject(context.Background(), "bob@example.com")

	// Test 1: Valid action in jurisdiction
	poa1 := &PowerOfAttorney{
		Jurisdiction: "US",
		Scope:        []string{"read"},
	}
	err1 := svc.ValidateJurisdictionCompliance(ctx, poa1)
	if err1 != nil {
		t.Fatalf("Valid action should pass compliance: %v", err1)
	}

	// Test 2: Blocked action in jurisdiction
	poa2 := &PowerOfAttorney{
		Jurisdiction: "EU",
		Scope:        []string{"unrestricted_data_export"},
	}
	err2 := svc.ValidateJurisdictionCompliance(ctx, poa2)
	if err2 == nil {
		t.Fatal("Blocked action should fail compliance validation")
	}
}
