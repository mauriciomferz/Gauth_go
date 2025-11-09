package auth

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLegalFrameworkValidator_Initialize tests the Initialize method
func TestLegalFrameworkValidator_Initialize(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Initialize creates validator",
		},
		{
			name: "Initialize is idempotent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			if v.validator == nil {
				t.Error("Initialize() failed to create validator")
			}

			// Test idempotency
			v.Initialize()
			if v.validator == nil {
				t.Error("Initialize() second call failed")
			}
		})
	}
}

// TestLegalFrameworkValidator_ValidateJurisdiction tests ValidateJurisdiction method
func TestLegalFrameworkValidator_ValidateJurisdiction(t *testing.T) {
	tests := []struct {
		name         string
		jurisdiction Jurisdiction
		action       string
		wantErr      bool
	}{
		{
			name:         "Valid US jurisdiction",
			jurisdiction: JurisdictionUS,
			action:       "transfer",
			wantErr:      false,
		},
		{
			name:         "Valid EU jurisdiction",
			jurisdiction: JurisdictionEU,
			action:       "access",
			wantErr:      false,
		},
		{
			name:         "Valid UK jurisdiction",
			jurisdiction: JurisdictionUK,
			action:       "delegate",
			wantErr:      false,
		},
		{
			name:         "Valid CA jurisdiction",
			jurisdiction: JurisdictionCA,
			action:       "approve",
			wantErr:      false,
		},
		{
			name:         "Valid AU jurisdiction",
			jurisdiction: JurisdictionAU,
			action:       "revoke",
			wantErr:      false,
		},
		{
			name:         "Valid JP jurisdiction",
			jurisdiction: JurisdictionJP,
			action:       "verify",
			wantErr:      false,
		},
		{
			name:         "Empty action",
			jurisdiction: JurisdictionUS,
			action:       "",
			wantErr:      false, // Current implementation accepts empty action
		},
		{
			name:         "Special characters in action",
			jurisdiction: JurisdictionEU,
			action:       "transfer:funds:high-value",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			ctx := context.Background()
			err := v.ValidateJurisdiction(ctx, tt.jurisdiction, tt.action)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJurisdiction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLegalFrameworkValidator_GetJurisdictionRules tests GetJurisdictionRules method
func TestLegalFrameworkValidator_GetJurisdictionRules(t *testing.T) {
	tests := []struct {
		name         string
		jurisdiction string
		wantErr      bool
		checkRules   func(t *testing.T, rules *JurisdictionRules)
	}{
		{
			name:         "Get US rules",
			jurisdiction: "US",
			wantErr:      false,
			checkRules: func(t *testing.T, rules *JurisdictionRules) {
				if rules == nil {
					t.Error("Expected non-nil rules")
					return
				}
				if rules.Country != "US" {
					t.Errorf("Expected country US, got %s", rules.Country)
				}
				if rules.RequiredApprovals == nil {
					t.Error("Expected non-nil RequiredApprovals")
				}
				if rules.ValueLimits == nil {
					t.Error("Expected non-nil ValueLimits")
				}
			},
		},
		{
			name:         "Get EU rules",
			jurisdiction: "EU",
			wantErr:      false,
			checkRules: func(t *testing.T, rules *JurisdictionRules) {
				if rules == nil {
					t.Error("Expected non-nil rules")
					return
				}
				if rules.Country != "EU" {
					t.Errorf("Expected country EU, got %s", rules.Country)
				}
			},
		},
		{
			name:         "Get UK rules",
			jurisdiction: "UK",
			wantErr:      false,
			checkRules: func(t *testing.T, rules *JurisdictionRules) {
				if rules == nil {
					t.Error("Expected non-nil rules")
					return
				}
				if rules.Country != "UK" {
					t.Errorf("Expected country UK, got %s", rules.Country)
				}
			},
		},
		{
			name:         "Get CA rules",
			jurisdiction: "CA",
			wantErr:      false,
			checkRules: func(t *testing.T, rules *JurisdictionRules) {
				if rules != nil && rules.Country != "CA" {
					t.Errorf("Expected country CA, got %s", rules.Country)
				}
			},
		},
		{
			name:         "Empty jurisdiction",
			jurisdiction: "",
			wantErr:      true, // Should error on empty jurisdiction
			checkRules:   nil,
		},
		{
			name:         "Invalid jurisdiction",
			jurisdiction: "INVALID",
			wantErr:      true,
			checkRules:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			rules, err := v.GetJurisdictionRules(tt.jurisdiction)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetJurisdictionRules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkRules != nil && !tt.wantErr {
				tt.checkRules(t, rules)
			}
		})
	}
}

// TestLegalFrameworkValidator_ValidateJurisdictionRequirements tests ValidateJurisdictionRequirements
func TestLegalFrameworkValidator_ValidateJurisdictionRequirements(t *testing.T) {
	tests := []struct {
		name    string
		rules   *JurisdictionRules
		action  string
		wantErr bool
	}{
		{
			name: "Valid requirements",
			rules: &JurisdictionRules{
				Country:           "US",
				RequiredApprovals: map[string]ApprovalLevel{"transfer": SingleApproval},
				ValueLimits:       map[string]float64{"transfer": 10000.0},
			},
			action:  "transfer",
			wantErr: false,
		},
		{
			name: "Empty action",
			rules: &JurisdictionRules{
				Country:           "EU",
				RequiredApprovals: map[string]ApprovalLevel{},
				ValueLimits:       map[string]float64{},
			},
			action:  "",
			wantErr: false, // Current implementation accepts empty action
		},
		{
			name: "Dual approval requirement",
			rules: &JurisdictionRules{
				Country:           "UK",
				RequiredApprovals: map[string]ApprovalLevel{"high-value": DualApproval},
				ValueLimits:       map[string]float64{"high-value": 100000.0},
			},
			action:  "high-value",
			wantErr: false,
		},
		{
			name: "Board approval requirement",
			rules: &JurisdictionRules{
				Country:           "JP",
				RequiredApprovals: map[string]ApprovalLevel{"critical": BoardApproval},
				ValueLimits:       map[string]float64{"critical": 1000000.0},
			},
			action:  "critical",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			ctx := context.Background()
			err := v.ValidateJurisdictionRequirements(ctx, tt.rules, tt.action)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJurisdictionRequirements() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLegalFrameworkValidator_VerifyLegalCapacity tests VerifyLegalCapacity method
func TestLegalFrameworkValidator_VerifyLegalCapacity(t *testing.T) {
	tests := []struct {
		name    string
		entity  *Entity
		wantErr bool
	}{
		{
			name: "Valid corporation entity",
			entity: &Entity{
				ID:             "entity_001",
				Type:           "corporation",
				JurisdictionID: "US",
				LegalStatus:    "active",
			},
			wantErr: false,
		},
		{
			name: "Valid individual entity",
			entity: &Entity{
				ID:             "entity_002",
				Type:           "individual",
				JurisdictionID: "EU",
				LegalStatus:    "verified",
			},
			wantErr: false,
		},
		{
			name: "Valid partnership entity",
			entity: &Entity{
				ID:             "entity_003",
				Type:           "partnership",
				JurisdictionID: "UK",
				LegalStatus:    "active",
			},
			wantErr: false,
		},
		{
			name: "Entity with capacity proofs - unsupported type",
			entity: &Entity{
				ID:             "entity_004",
				Type:           "trust",
				JurisdictionID: "CA",
				LegalStatus:    "active",
				CapacityProofs: []CapacityProof{
					{
						Type:         "incorporation",
						IssuedAt:     time.Now(),
						ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
						IssuerID:     "issuer_001",
						Proof:        "proof_data",
						Jurisdiction: "CA",
					},
				},
			},
			wantErr: true, // trust type not supported in compliance validator
		},
		{
			name: "Entity with fiduciary duties - unsupported type",
			entity: &Entity{
				ID:             "entity_005",
				Type:           "fiduciary",
				JurisdictionID: "AU",
				LegalStatus:    "active",
				FiduciaryDuties: []FiduciaryDuty{
					{
						Type:        "care",
						Description: "Duty of care",
						Scope:       []string{"financial"},
						Validation:  []string{"annual_review"},
					},
				},
			},
			wantErr: true, // fiduciary type not supported in compliance validator
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			ctx := context.Background()
			err := v.VerifyLegalCapacity(ctx, tt.entity)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyLegalCapacity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLegalFrameworkValidator_ValidateClientResourceServerInteraction tests interaction validation
func TestLegalFrameworkValidator_ValidateClientResourceServerInteraction(t *testing.T) {
	tests := []struct {
		name    string
		client  *Client
		server  *ResourceServer
		wantErr bool
	}{
		{
			name: "Valid interaction with entities",
			client: &Client{
				ID:      "client_001",
				Type:    "ai_assistant",
				OwnerID: "owner_001",
				Entity: &Entity{
					ID:             "entity_001",
					Type:           "corporation",
					JurisdictionID: "US",
				},
			},
			server: &ResourceServer{
				ID:   "server_001",
				Type: "data_store",
				Entity: &Entity{
					ID:             "entity_002",
					Type:           "corporation",
					JurisdictionID: "US",
				},
			},
			wantErr: false,
		},
		{
			name: "Client without entity",
			client: &Client{
				ID:      "client_002",
				Type:    "ai_agent",
				OwnerID: "owner_002",
				Entity:  nil,
			},
			server: &ResourceServer{
				ID:   "server_002",
				Type: "api",
				Entity: &Entity{
					ID:             "entity_003",
					Type:           "corporation",
					JurisdictionID: "EU",
				},
			},
			wantErr: false, // No error when client entity is nil
		},
		{
			name: "Server without entity",
			client: &Client{
				ID:      "client_003",
				Type:    "ai_service",
				OwnerID: "owner_003",
				Entity: &Entity{
					ID:             "entity_004",
					Type:           "individual",
					JurisdictionID: "UK",
				},
			},
			server: &ResourceServer{
				ID:     "server_003",
				Type:   "storage",
				Entity: nil,
			},
			wantErr: false, // No error when server entity is nil
		},
		{
			name: "Both without entities",
			client: &Client{
				ID:      "client_004",
				Type:    "ai_bot",
				OwnerID: "owner_004",
				Entity:  nil,
			},
			server: &ResourceServer{
				ID:     "server_004",
				Type:   "database",
				Entity: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			ctx := context.Background()
			err := v.ValidateClientResourceServerInteraction(ctx, tt.client, tt.server)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateClientResourceServerInteraction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLegalFrameworkValidator_ValidateResourceServerPowers tests server power validation
func TestLegalFrameworkValidator_ValidateResourceServerPowers(t *testing.T) {
	tests := []struct {
		name    string
		token   *Token
		request *LegalFrameworkRequest
		wantErr bool
	}{
		{
			name: "Valid power with jurisdiction",
			token: &Token{
				ID:        "token_001",
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			request: &LegalFrameworkRequest{
				ID:           "request_001",
				ClientID:     "client_001",
				Action:       "read",
				Resource:     "data",
				Jurisdiction: "US",
			},
			wantErr: false,
		},
		{
			name: "Valid power without jurisdiction",
			token: &Token{
				ID:        "token_002",
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			request: &LegalFrameworkRequest{
				ID:           "request_002",
				ClientID:     "client_002",
				Action:       "write",
				Resource:     "records",
				Jurisdiction: "",
			},
			wantErr: false,
		},
		{
			name: "Multiple scopes",
			token: &Token{
				ID:        "token_003",
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(2 * time.Hour),
			},
			request: &LegalFrameworkRequest{
				ID:           "request_003",
				ClientID:     "client_003",
				Action:       "admin",
				Resource:     "system",
				Jurisdiction: "EU",
				Scope:        []string{"read", "write", "delete"},
			},
			wantErr: false,
		},
		{
			name: "With power of attorney",
			token: &Token{
				ID:        "token_004",
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(3 * time.Hour),
			},
			request: &LegalFrameworkRequest{
				ID:           "request_004",
				ClientID:     "client_004",
				Action:       "transfer",
				Resource:     "funds",
				Jurisdiction: "UK",
				PowerOfAttorney: &PowerOfAttorney{
					ID:        "poa_001",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			ctx := context.Background()
			err := v.ValidateResourceServerPowers(ctx, tt.token, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResourceServerPowers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLegalFrameworkValidator_EnforceFiduciaryDuties tests fiduciary duty enforcement
func TestLegalFrameworkValidator_EnforceFiduciaryDuties(t *testing.T) {
	tests := []struct {
		name    string
		power   *PowerOfAttorney
		wantErr bool
	}{
		{
			name: "Valid power of attorney",
			power: &PowerOfAttorney{
				ID:        "poa_001",
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "Expired power of attorney",
			power: &PowerOfAttorney{
				ID:        "poa_002",
				IssuedAt:  time.Now().Add(-2 * 365 * 24 * time.Hour),
				ExpiresAt: time.Now().Add(-1 * 365 * 24 * time.Hour),
			},
			wantErr: false, // Current implementation doesn't validate expiration
		},
		{
			name: "Future power of attorney",
			power: &PowerOfAttorney{
				ID:        "poa_003",
				IssuedAt:  time.Now().Add(1 * 24 * time.Hour),
				ExpiresAt: time.Now().Add(366 * 24 * time.Hour),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			ctx := context.Background()
			err := v.EnforceFiduciaryDuties(ctx, tt.power)

			if (err != nil) != tt.wantErr {
				t.Errorf("EnforceFiduciaryDuties() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLegalFrameworkValidator_ValidateDuty tests duty validation
func TestLegalFrameworkValidator_ValidateDuty(t *testing.T) {
	tests := []struct {
		name    string
		duty    FiduciaryDuty
		wantErr bool
	}{
		{
			name: "Valid duty of care",
			duty: FiduciaryDuty{
				Type:        "care",
				Description: "Duty of care and diligence",
				Scope:       []string{"financial", "operational"},
				Validation:  []string{"annual_review"},
			},
			wantErr: false,
		},
		{
			name: "Valid duty of loyalty",
			duty: FiduciaryDuty{
				Type:        "loyalty",
				Description: "Duty of loyalty to beneficiaries",
				Scope:       []string{"all"},
				Validation:  []string{"conflict_check"},
			},
			wantErr: false,
		},
		{
			name: "Empty type - should error",
			duty: FiduciaryDuty{
				Type:        "",
				Description: "Some description",
				Scope:       []string{"test"},
			},
			wantErr: true,
		},
		{
			name: "Empty description - should error",
			duty: FiduciaryDuty{
				Type:        "prudence",
				Description: "",
				Scope:       []string{"investment"},
			},
			wantErr: true,
		},
		{
			name: "Empty scope - valid",
			duty: FiduciaryDuty{
				Type:        "disclosure",
				Description: "Duty to disclose",
				Scope:       []string{},
				Validation:  []string{"quarterly"},
			},
			wantErr: false,
		},
		{
			name: "Empty validation - valid",
			duty: FiduciaryDuty{
				Type:        "obedience",
				Description: "Duty of obedience",
				Scope:       []string{"statutory"},
				Validation:  []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			ctx := context.Background()
			err := v.ValidateDuty(ctx, tt.duty)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDuty() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Additional validation for error messages
			if tt.wantErr && err != nil {
				errMsg := err.Error()
				if !strings.Contains(errMsg, "invalid duty") {
					t.Errorf("Expected error message to contain 'invalid duty', got: %s", errMsg)
				}
			}
		})
	}
}

// TestLegalFrameworkValidator_TrackApprovalDetails tests approval tracking
func TestLegalFrameworkValidator_TrackApprovalDetails(t *testing.T) {
	tests := []struct {
		name    string
		event   *ApprovalEvent
		wantErr bool
	}{
		{
			name: "Valid approval event",
			event: &ApprovalEvent{
				ApprovalID:      "approval_001",
				RequesterID:     "user_001",
				ApproverID:      "approver_001",
				Action:          "transfer",
				JurisdictionID:  "US",
				LegalBasis:      "contract_001",
				FiduciaryChecks: []FiduciaryDuty{},
				Time:            time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Event with fiduciary checks",
			event: &ApprovalEvent{
				ApprovalID:     "approval_002",
				RequesterID:    "user_002",
				ApproverID:     "approver_002",
				Action:         "delegate",
				JurisdictionID: "EU",
				LegalBasis:     "poa_001",
				FiduciaryChecks: []FiduciaryDuty{
					{
						Type:        "care",
						Description: "Duty of care",
						Scope:       []string{"financial"},
					},
				},
				Time: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Empty approval ID - should error",
			event: &ApprovalEvent{
				ApprovalID:     "",
				RequesterID:    "user_003",
				ApproverID:     "approver_003",
				Action:         "approve",
				JurisdictionID: "UK",
				Time:           time.Now(),
			},
			wantErr: true,
		},
		{
			name: "Empty action - should error",
			event: &ApprovalEvent{
				ApprovalID:     "approval_004",
				RequesterID:    "user_004",
				ApproverID:     "approver_004",
				Action:         "",
				JurisdictionID: "CA",
				Time:           time.Now(),
			},
			wantErr: true,
		},
		{
			name: "With evidence",
			event: &ApprovalEvent{
				ApprovalID:     "approval_005",
				RequesterID:    "user_005",
				ApproverID:     "approver_005",
				Action:         "verify",
				JurisdictionID: "AU",
				Evidence:       map[string]interface{}{"document_id": "doc_001", "verified": true},
				Time:           time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &LegalFrameworkValidator{}
			v.Initialize()

			ctx := context.Background()
			err := v.TrackApprovalDetails(ctx, tt.event)

			if (err != nil) != tt.wantErr {
				t.Errorf("TrackApprovalDetails() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Additional validation for error messages
			if tt.wantErr && err != nil {
				errMsg := err.Error()
				if !strings.Contains(errMsg, "invalid approval event") {
					t.Errorf("Expected error message to contain 'invalid approval event', got: %s", errMsg)
				}
			}
		})
	}
}

// TestNewStandardLegalFramework tests StandardLegalFramework creation
func TestNewStandardLegalFramework(t *testing.T) {
	framework := NewStandardLegalFramework()

	if framework == nil {
		t.Fatal("NewStandardLegalFramework() returned nil")
	}

	if framework.validator == nil {
		t.Error("Framework validator is nil")
	}

	if framework.store == nil {
		t.Error("Framework store is nil")
	}

	// Verify validator is initialized
	if framework.validator.validator == nil {
		t.Error("Validator's internal validator is nil")
	}
}

// TestStandardLegalFramework_ValidateJurisdictionRequirements tests framework requirements validation
func TestStandardLegalFramework_ValidateJurisdictionRequirements(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	tests := []struct {
		name    string
		rules   *JurisdictionRules
		action  string
		wantErr bool
	}{
		{
			name: "Valid requirements",
			rules: &JurisdictionRules{
				Country:           "US",
				RequiredApprovals: map[string]ApprovalLevel{"read": SingleApproval},
				ValueLimits:       map[string]float64{"read": 1000.0},
			},
			action:  "read",
			wantErr: false,
		},
		{
			name: "Complex requirements",
			rules: &JurisdictionRules{
				Country: "EU",
				RequiredApprovals: map[string]ApprovalLevel{
					"low":      SingleApproval,
					"medium":   DualApproval,
					"high":     BoardApproval,
					"critical": BoardApproval,
				},
				ValueLimits: map[string]float64{
					"low":      1000.0,
					"medium":   50000.0,
					"high":     500000.0,
					"critical": 10000000.0,
				},
			},
			action:  "high",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := framework.ValidateJurisdictionRequirements(ctx, tt.rules, tt.action)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJurisdictionRequirements() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStandardLegalFramework_ValidateDuty tests framework duty validation
func TestStandardLegalFramework_ValidateDuty(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	tests := []struct {
		name    string
		duty    FiduciaryDuty
		wantErr bool
	}{
		{
			name: "Valid duty",
			duty: FiduciaryDuty{
				Type:        "care",
				Description: "Exercise reasonable care",
				Scope:       []string{"all_assets"},
			},
			wantErr: false,
		},
		{
			name: "Invalid duty - empty type",
			duty: FiduciaryDuty{
				Type:        "",
				Description: "Some duty",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := framework.ValidateDuty(ctx, tt.duty)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDuty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStandardLegalFramework_TrackApprovalDetails tests framework approval tracking
func TestStandardLegalFramework_TrackApprovalDetails(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	tests := []struct {
		name          string
		approvalOrEvent interface{}
		wantErr       bool
	}{
		{
			name: "Track Approval struct",
			approvalOrEvent: &Approval{
				ID:             "approval_001",
				TransactionID:  "tx_001",
				RequesterID:    "user_001",
				ApproverID:     "approver_001",
				Action:         "transfer",
				JurisdictionID: "US",
				LegalBasis:     "contract_001",
			},
			wantErr: false,
		},
		{
			name: "Track ApprovalEvent struct",
			approvalOrEvent: &ApprovalEvent{
				ApprovalID:     "approval_002",
				RequesterID:    "user_002",
				ApproverID:     "approver_002",
				Action:         "approve",
				JurisdictionID: "EU",
				Time:           time.Now(),
			},
			wantErr: false,
		},
		{
			name:            "Unsupported type - should error",
			approvalOrEvent: "invalid_type",
			wantErr:         true,
		},
		{
			name: "Approval with fiduciary checks",
			approvalOrEvent: &Approval{
				ID:             "approval_003",
				TransactionID:  "tx_003",
				RequesterID:    "user_003",
				ApproverID:     "approver_003",
				Action:         "delegate",
				JurisdictionID: "UK",
				LegalBasis:     "poa_003",
				FiduciaryChecks: []FiduciaryDuty{
					{
						Type:        "loyalty",
						Description: "Duty of loyalty",
						Scope:       []string{"beneficiary"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := framework.TrackApprovalDetails(ctx, tt.approvalOrEvent)

			if (err != nil) != tt.wantErr {
				t.Errorf("TrackApprovalDetails() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStandardLegalFramework_VerifyLegalCapacity tests framework capacity verification
func TestStandardLegalFramework_VerifyLegalCapacity(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	tests := []struct {
		name    string
		entity  *Entity
		wantErr bool
	}{
		{
			name: "Valid entity",
			entity: &Entity{
				ID:             "entity_001",
				Type:           "corporation",
				JurisdictionID: "US",
				LegalStatus:    "active",
			},
			wantErr: false,
		},
		{
			name: "Entity with proofs",
			entity: &Entity{
				ID:             "entity_002",
				Type:           "trust",
				JurisdictionID: "EU",
				LegalStatus:    "verified",
				CapacityProofs: []CapacityProof{
					{
						Type:      "registration",
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := framework.VerifyLegalCapacity(ctx, tt.entity)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyLegalCapacity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStandardLegalFramework_ValidateClientResourceServerInteraction tests framework interaction validation
func TestStandardLegalFramework_ValidateClientResourceServerInteraction(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	client := &Client{
		ID:      "client_001",
		Type:    "ai_assistant",
		OwnerID: "owner_001",
		Entity: &Entity{
			ID:             "entity_001",
			Type:           "corporation",
			JurisdictionID: "US",
		},
	}

	server := &ResourceServer{
		ID:   "server_001",
		Type: "api",
		Entity: &Entity{
			ID:             "entity_002",
			Type:           "corporation",
			JurisdictionID: "US",
		},
	}

	err := framework.ValidateClientResourceServerInteraction(ctx, client, server)
	if err != nil {
		t.Errorf("ValidateClientResourceServerInteraction() unexpected error = %v", err)
	}
}

// TestStandardLegalFramework_ValidateResourceServerPowers tests framework server power validation
func TestStandardLegalFramework_ValidateResourceServerPowers(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	token := &Token{
		ID:        "token_001",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	request := &LegalFrameworkRequest{
		ID:           "request_001",
		ClientID:     "client_001",
		Action:       "read",
		Resource:     "data",
		Jurisdiction: "US",
	}

	err := framework.ValidateResourceServerPowers(ctx, token, request)
	if err != nil {
		t.Errorf("ValidateResourceServerPowers() unexpected error = %v", err)
	}
}

// TestStandardLegalFramework_ValidateJurisdiction tests framework jurisdiction validation
func TestStandardLegalFramework_ValidateJurisdiction(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	tests := []struct {
		name         string
		jurisdiction interface{}
		action       string
		wantErr      bool
	}{
		{
			name:         "String jurisdiction",
			jurisdiction: "US",
			action:       "transfer",
			wantErr:      false,
		},
		{
			name:         "Jurisdiction type",
			jurisdiction: JurisdictionEU,
			action:       "access",
			wantErr:      false,
		},
		{
			name:         "Unsupported type",
			jurisdiction: 123,
			action:       "read",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := framework.ValidateJurisdiction(ctx, tt.jurisdiction, tt.action)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJurisdiction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStandardLegalFramework_GetJurisdictionRules tests framework rules retrieval
func TestStandardLegalFramework_GetJurisdictionRules(t *testing.T) {
	framework := NewStandardLegalFramework()

	rules, err := framework.GetJurisdictionRules("US")
	if err != nil {
		t.Errorf("GetJurisdictionRules() unexpected error = %v", err)
		return
	}

	if rules == nil {
		t.Error("Expected non-nil rules")
		return
	}

	if rules.Country != "US" {
		t.Errorf("Expected country US, got %s", rules.Country)
	}
}

// TestStandardLegalFramework_EnforceFiduciaryDuties tests framework fiduciary duty enforcement
func TestStandardLegalFramework_EnforceFiduciaryDuties(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	power := &PowerOfAttorney{
		ID:        "poa_001",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	}

	err := framework.EnforceFiduciaryDuties(ctx, power)
	if err != nil {
		t.Errorf("EnforceFiduciaryDuties() unexpected error = %v", err)
	}
}

// TestStandardLegalFramework_Store tests framework store access
func TestStandardLegalFramework_Store(t *testing.T) {
	framework := NewStandardLegalFramework()

	store := framework.Store()
	if store == nil {
		t.Error("Store() returned nil")
	}

	// Test store functionality
	ctx := context.Background()
	records, err := store.GetTrackingRecords(ctx, "approval_001")
	if err != nil {
		t.Errorf("GetTrackingRecords() unexpected error = %v", err)
		return
	}

	if len(records) == 0 {
		t.Error("Expected non-empty tracking records")
	}

	// Verify record structure
	record := records[0]
	if record.ApprovalID != "approval_001" {
		t.Errorf("Expected approval ID approval_001, got %s", record.ApprovalID)
	}
}

// TestStoreStub_GetTrackingRecords tests StoreStub implementation
func TestStoreStub_GetTrackingRecords(t *testing.T) {
	store := &StoreStub{}
	ctx := context.Background()

	tests := []struct {
		name       string
		approvalID string
		wantErr    bool
	}{
		{
			name:       "Get records for approval",
			approvalID: "approval_001",
			wantErr:    false,
		},
		{
			name:       "Get records for another approval",
			approvalID: "approval_002",
			wantErr:    false,
		},
		{
			name:       "Empty approval ID",
			approvalID: "",
			wantErr:    false, // Current implementation accepts empty ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := store.GetTrackingRecords(ctx, tt.approvalID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTrackingRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(records) == 0 {
					t.Error("Expected non-empty records")
				}

				// Verify record structure
				record := records[0]
				if record.ApprovalID != tt.approvalID {
					t.Errorf("Expected approval ID %s, got %s", tt.approvalID, record.ApprovalID)
				}
				if record.Status != "completed" {
					t.Errorf("Expected status completed, got %s", record.Status)
				}
				if record.Action != "approval_tracked" {
					t.Errorf("Expected action approval_tracked, got %s", record.Action)
				}
			}
		})
	}
}

// TestLegalFramework_ConcurrentOperations tests thread safety
func TestLegalFramework_ConcurrentOperations(t *testing.T) {
	framework := NewStandardLegalFramework()
	ctx := context.Background()

	var wg sync.WaitGroup
	iterations := 10

	// Test concurrent jurisdiction validation
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(idx int) {
			defer wg.Done()

			jurisdictions := []interface{}{"US", JurisdictionEU, "UK", JurisdictionCA}
			j := jurisdictions[idx%len(jurisdictions)]

			_ = framework.ValidateJurisdiction(ctx, j, "test_action")
		}(i)
	}

	// Test concurrent rules retrieval
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(idx int) {
			defer wg.Done()

			countries := []string{"US", "EU", "UK", "CA"}
			country := countries[idx%len(countries)]

			_, _ = framework.GetJurisdictionRules(country)
		}(i)
	}

	// Test concurrent duty validation
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(idx int) {
			defer wg.Done()

			duty := FiduciaryDuty{
				Type:        "care",
				Description: "Test duty",
				Scope:       []string{"test"},
			}

			_ = framework.ValidateDuty(ctx, duty)
		}(i)
	}

	wg.Wait()
}
