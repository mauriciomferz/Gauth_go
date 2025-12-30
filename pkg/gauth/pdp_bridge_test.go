// Package gauth - PDP Bridge Tests
package gauth

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/pdp"
)

// mockPDPEngine implements pdp.Engine for testing
type mockPDPEngine struct {
	allowDecision bool
	returnError   error
}

func (m *mockPDPEngine) Evaluate(ctx context.Context, req pdp.Request) (pdp.Decision, error) {
	if m.returnError != nil {
		return pdp.Decision{}, m.returnError
	}

	return pdp.Decision{
		Allow:    m.allowDecision,
		Reason:   "test decision",
		Policies: []string{"test-policy"},
	}, nil
}

func (m *mockPDPEngine) Metrics() pdp.MetricsSnapshot {
	return pdp.MetricsSnapshot{}
}

func TestPDPBridge_EvaluatePolicy(t *testing.T) {
	tests := []struct {
		name          string
		allowDecision bool
		request       interface{}
		wantAllow     bool
		wantError     bool
	}{
		{
			name:          "allow decision with token request",
			allowDecision: true,
			request: &ExtendedTokenRequest{
				GrantID: "test-grant",
				Scope:   []string{"read", "write"},
			},
			wantAllow: true,
			wantError: false,
		},
		{
			name:          "deny decision with token request",
			allowDecision: false,
			request: &ExtendedTokenRequest{
				GrantID: "test-grant",
				Scope:   []string{"read", "write"},
			},
			wantAllow: false,
			wantError: false,
		},
		{
			name:          "allow decision with auth request",
			allowDecision: true,
			request: &ExtendedAuthorizationRequest{
				AuthorizationRequest: &AuthorizationRequest{
					ClientID: "test-client",
					Scopes:   []string{"read"},
				},
				RequestedActions: []string{"read"},
				RequestTime:      time.Now(),
			},
			wantAllow: true,
			wantError: false,
		},
		{
			name:          "allow decision with grant",
			allowDecision: true,
			request: &ExtendedAuthorizationGrant{
				AuthorizationGrant: &AuthorizationGrant{
					GrantID:  "test-grant",
					ClientID: "test-client",
				},
				ResourceOwnerID: "resource-owner-001",
				IssuerID:        "issuer-001",
				IssuedAt:        time.Now(),
			},
			wantAllow: true,
			wantError: false,
		},
		{
			name:          "allow decision with map request",
			allowDecision: true,
			request: map[string]interface{}{
				"subject":  "user-123",
				"action":   "read",
				"resource": "document-456",
			},
			wantAllow: true,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &mockPDPEngine{
				allowDecision: tt.allowDecision,
			}
			bridge := NewPDPBridge(engine)

			allow, err := bridge.EvaluatePolicy(context.Background(), tt.request)

			if (err != nil) != tt.wantError {
				t.Errorf("EvaluatePolicy() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if allow != tt.wantAllow {
				t.Errorf("EvaluatePolicy() = %v, want %v", allow, tt.wantAllow)
			}
		})
	}
}

func TestPDPBridge_ConvertTokenRequest(t *testing.T) {
	bridge := &PDPBridge{}

	req := &ExtendedTokenRequest{
		GrantID:          "grant-123",
		Scope:            []string{"read", "write"},
		JurisdictionCode: "US",
		ClientOwnerInfo: &ClientOwnerInfo{
			OwnerID:   "owner-001",
			OwnerType: "organization",
		},
		ResourceOwnerInfo: &ResourceOwnerInfo{
			OwnerID: "resource-owner-001",
		},
		RequestedActions: []string{"read"},
	}

	pdpReq := bridge.convertTokenRequest(req)

	if pdpReq.Subject != "owner-001" {
		t.Errorf("Expected subject 'owner-001', got '%s'", pdpReq.Subject)
	}
	if pdpReq.Action != "read" {
		t.Errorf("Expected action 'read', got '%s'", pdpReq.Action)
	}
	if pdpReq.Resource != "resource-owner-001" {
		t.Errorf("Expected resource 'resource-owner-001', got '%s'", pdpReq.Resource)
	}
	if pdpReq.Attributes["grant_id"] != "grant-123" {
		t.Errorf("Expected grant_id 'grant-123', got '%s'", pdpReq.Attributes["grant_id"])
	}
	if pdpReq.Attributes["jurisdiction"] != "US" {
		t.Errorf("Expected jurisdiction 'US', got '%s'", pdpReq.Attributes["jurisdiction"])
	}
}

func TestPDPBridge_ConvertAuthRequest(t *testing.T) {
	bridge := &PDPBridge{}

	req := &ExtendedAuthorizationRequest{
		AuthorizationRequest: &AuthorizationRequest{
			ClientID: "client-001",
			Scopes:   []string{"read", "write"},
		},
		RequestedActions: []string{"execute"},
		RequestTime:      time.Now(),
		Restrictions: []PowerRestriction{
			{RestrictionType: "time_bound"},
		},
	}

	pdpReq := bridge.convertAuthRequest(req)

	if pdpReq.Subject != "client-001" {
		t.Errorf("Expected subject 'client-001', got '%s'", pdpReq.Subject)
	}
	if pdpReq.Action != "execute" {
		t.Errorf("Expected action 'execute', got '%s'", pdpReq.Action)
	}
	if pdpReq.Attributes["client_id"] != "client-001" {
		t.Errorf("Expected client_id 'client-001', got '%s'", pdpReq.Attributes["client_id"])
	}
	if pdpReq.Attributes["restrictions_count"] != "1" {
		t.Errorf("Expected restrictions_count '1', got '%s'", pdpReq.Attributes["restrictions_count"])
	}
}

func TestPDPBridge_ConvertGrantRequest(t *testing.T) {
	bridge := &PDPBridge{}

	now := time.Now()
	grant := &ExtendedAuthorizationGrant{
		AuthorizationGrant: &AuthorizationGrant{
			GrantID:  "grant-456",
			ClientID: "client-002",
		},
		ResourceOwnerID: "owner-002",
		IssuerID:        "issuer-002",
		IssuedAt:        now,
	}

	pdpReq := bridge.convertGrantRequest(grant)

	if pdpReq.Subject != "client-002" {
		t.Errorf("Expected subject 'client-002', got '%s'", pdpReq.Subject)
	}
	if pdpReq.Action != "use_grant" {
		t.Errorf("Expected action 'use_grant', got '%s'", pdpReq.Action)
	}
	if pdpReq.Resource != "owner-002" {
		t.Errorf("Expected resource 'owner-002', got '%s'", pdpReq.Resource)
	}
	if pdpReq.Attributes["grant_id"] != "grant-456" {
		t.Errorf("Expected grant_id 'grant-456', got '%s'", pdpReq.Attributes["grant_id"])
	}
	if pdpReq.Time != now {
		t.Errorf("Expected time %v, got %v", now, pdpReq.Time)
	}
}

func TestPDPBridge_ConvertMapRequest(t *testing.T) {
	bridge := &PDPBridge{}

	req := map[string]interface{}{
		"subject":    "user-789",
		"action":     "delete",
		"resource":   "file-123",
		"ip_address": "192.168.1.100",
		"department": "engineering",
	}

	pdpReq := bridge.convertMapRequest(req)

	if pdpReq.Subject != "user-789" {
		t.Errorf("Expected subject 'user-789', got '%s'", pdpReq.Subject)
	}
	if pdpReq.Action != "delete" {
		t.Errorf("Expected action 'delete', got '%s'", pdpReq.Action)
	}
	if pdpReq.Resource != "file-123" {
		t.Errorf("Expected resource 'file-123', got '%s'", pdpReq.Resource)
	}
	if pdpReq.Attributes["ip_address"] != "192.168.1.100" {
		t.Errorf("Expected ip_address '192.168.1.100', got '%s'", pdpReq.Attributes["ip_address"])
	}
	if pdpReq.Attributes["department"] != "engineering" {
		t.Errorf("Expected department 'engineering', got '%s'", pdpReq.Attributes["department"])
	}
}
