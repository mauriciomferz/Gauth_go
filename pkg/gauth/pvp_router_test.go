// Package gauth - PVP Router Tests
package gauth

import (
	"context"
	"testing"
	"time"
)

// MockPVP is a mock implementation of PowerVerificationPoint for testing
type MockPVP struct {
	result *IdentityProofResult
	err    error
}

func (m *MockPVP) VerifyIdentityProof(ctx context.Context, request *IdentityProofRequest) (*IdentityProofResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &IdentityProofResult{
		Valid:      true,
		SubjectID:  request.SubjectID,
		Identity:   "Test Identity",
		VerifiedAt: time.Now(),
		TrustLevel: "substantial",
	}, nil
}

// TestNewPVPRouter tests router creation
func TestNewPVPRouter(t *testing.T) {
	defaultPVP := &MockPVP{
		result: &IdentityProofResult{
			Valid:      true,
			SubjectID:  "default_user",
			Identity:   "Default PVP",
			TrustLevel: "low",
		},
	}
	
	router := NewPVPRouter(defaultPVP)
	if router == nil {
		t.Fatal("Expected router, got nil")
	}
	if router.defaultPVP != defaultPVP {
		t.Error("Default PVP not set correctly")
	}
}

// TestPVPRouter_RegisterPVP tests PVP registration
func TestPVPRouter_RegisterPVP(t *testing.T) {
	router := NewPVPRouter(nil)
	
	oidcPVP := &MockPVP{
		result: &IdentityProofResult{
			Valid:      true,
			SubjectID:  "oidc_user",
			Identity:   "OIDC Identity",
			TrustLevel: "high",
		},
	}
	
	router.RegisterPVP([]string{"oidc_id_token", "oidc_external"}, oidcPVP)
	
	// Verify registration
	pvp, found := router.GetPVP("oidc_id_token")
	if !found {
		t.Error("Expected to find oidc_id_token PVP")
	}
	if pvp != oidcPVP {
		t.Error("Retrieved PVP does not match registered PVP")
	}
	
	pvp, found = router.GetPVP("oidc_external")
	if !found {
		t.Error("Expected to find oidc_external PVP")
	}
	if pvp != oidcPVP {
		t.Error("Retrieved PVP does not match registered PVP")
	}
}

// TestPVPRouter_VerifyIdentityProof tests proof verification routing
func TestPVPRouter_VerifyIdentityProof(t *testing.T) {
	ctx := context.Background()
	
	// Create mock PVPs
	oidcPVP := &MockPVP{
		result: &IdentityProofResult{
			Valid:      true,
			SubjectID:  "oidc_user",
			Identity:   "OIDC User",
			TrustLevel: "high",
		},
	}
	
	eidasPVP := &MockPVP{
		result: &IdentityProofResult{
			Valid:      true,
			SubjectID:  "eidas_user",
			Identity:   "eIDAS User",
			TrustLevel: "substantial",
		},
	}
	
	defaultPVP := &MockPVP{
		result: &IdentityProofResult{
			Valid:      true,
			SubjectID:  "default_user",
			Identity:   "Default User",
			TrustLevel: "low",
		},
	}
	
	router := NewPVPRouter(defaultPVP)
	router.RegisterPVP([]string{"oidc_id_token", "oidc_external"}, oidcPVP)
	router.RegisterPVP([]string{"eIDAS"}, eidasPVP)
	
	tests := []struct {
		name            string
		request         *IdentityProofRequest
		expectedIdentity string
		expectedTrust    string
		expectError      bool
	}{
		{
			name: "route to OIDC PVP",
			request: &IdentityProofRequest{
				SubjectID:    "oidc_user",
				IdentityType: "natural_person",
				ProofMethod:  "oidc_id_token",
				ProofData: map[string]interface{}{
					"id_token": "eyJhbGc...",
					"audience": "client123",
				},
			},
			expectedIdentity: "OIDC User",
			expectedTrust:    "high",
			expectError:      false,
		},
		{
			name: "route to eIDAS PVP",
			request: &IdentityProofRequest{
				SubjectID:    "eidas_user",
				IdentityType: "natural_person",
				ProofMethod:  "eIDAS",
				ProofData: map[string]interface{}{
					"certificate": "...",
				},
			},
			expectedIdentity: "eIDAS User",
			expectedTrust:    "substantial",
			expectError:      false,
		},
		{
			name: "route to default PVP for unregistered method",
			request: &IdentityProofRequest{
				SubjectID:    "default_user",
				IdentityType: "natural_person",
				ProofMethod:  "government_id",
				ProofData: map[string]interface{}{
					"id_number": "123456",
				},
			},
			expectedIdentity: "Default User",
			expectedTrust:    "low",
			expectError:      false,
		},
		{
			name: "nil request",
			request: nil,
			expectError: true,
		},
		{
			name: "empty proof method",
			request: &IdentityProofRequest{
				SubjectID:    "user",
				IdentityType: "natural_person",
				ProofMethod:  "",
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := router.VerifyIdentityProof(ctx, tt.request)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if result == nil {
				t.Error("Expected result, got nil")
				return
			}
			
			if result.Identity != tt.expectedIdentity {
				t.Errorf("Expected identity '%s', got '%s'", tt.expectedIdentity, result.Identity)
			}
			
			if result.TrustLevel != tt.expectedTrust {
				t.Errorf("Expected trust level '%s', got '%s'", tt.expectedTrust, result.TrustLevel)
			}
		})
	}
}

// TestPVPRouter_VerifyIdentityProof_NoDefaultPVP tests behavior without default PVP
func TestPVPRouter_VerifyIdentityProof_NoDefaultPVP(t *testing.T) {
	ctx := context.Background()
	router := NewPVPRouter(nil)
	
	oidcPVP := &MockPVP{
		result: &IdentityProofResult{
			Valid:      true,
			SubjectID:  "oidc_user",
			Identity:   "OIDC User",
			TrustLevel: "high",
		},
	}
	
	router.RegisterPVP([]string{"oidc_id_token"}, oidcPVP)
	
	// Test with unregistered proof method and no default PVP
	request := &IdentityProofRequest{
		SubjectID:    "user",
		IdentityType: "natural_person",
		ProofMethod:  "unknown_method",
		ProofData:    map[string]interface{}{},
	}
	
	result, err := router.VerifyIdentityProof(ctx, request)
	if err == nil {
		t.Error("Expected error for unregistered proof method without default PVP")
	}
	if result != nil {
		t.Error("Expected nil result for error case")
	}
}

// TestPVPRouter_GetSupportedProofMethods tests supported methods retrieval
func TestPVPRouter_GetSupportedProofMethods(t *testing.T) {
	router := NewPVPRouter(nil)
	
	oidcPVP := &MockPVP{}
	eidasPVP := &MockPVP{}
	
	router.RegisterPVP([]string{"oidc_id_token", "oidc_external"}, oidcPVP)
	router.RegisterPVP([]string{"eIDAS", "government_id"}, eidasPVP)
	
	methods := router.GetSupportedProofMethods()
	
	expectedMethods := map[string]bool{
		"oidc_id_token":  true,
		"oidc_external":  true,
		"eIDAS":          true,
		"government_id":  true,
	}
	
	if len(methods) != len(expectedMethods) {
		t.Errorf("Expected %d methods, got %d", len(expectedMethods), len(methods))
	}
	
	for _, method := range methods {
		if !expectedMethods[method] {
			t.Errorf("Unexpected method: %s", method)
		}
	}
}

// TestPVPRouter_ConcurrentAccess tests thread-safe operation
func TestPVPRouter_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	router := NewPVPRouter(nil)
	
	oidcPVP := &MockPVP{
		result: &IdentityProofResult{
			Valid:      true,
			SubjectID:  "concurrent_user",
			Identity:   "Concurrent User",
			TrustLevel: "substantial",
		},
	}
	
	// Register PVP concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			router.RegisterPVP([]string{"oidc_id_token"}, oidcPVP)
			done <- true
		}()
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify concurrently
	for i := 0; i < 10; i++ {
		go func() {
			request := &IdentityProofRequest{
				SubjectID:    "concurrent_user",
				IdentityType: "natural_person",
				ProofMethod:  "oidc_id_token",
				ProofData:    map[string]interface{}{},
			}
			_, err := router.VerifyIdentityProof(ctx, request)
			if err != nil {
				t.Errorf("Concurrent verification failed: %v", err)
			}
			done <- true
		}()
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
}
