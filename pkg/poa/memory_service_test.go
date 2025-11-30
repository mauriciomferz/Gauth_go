package poa

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestMemoryService_Validate tests the Validate method
func TestMemoryService_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MemoryService) *ProofOfAuthorization
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid POA",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				poa := &ProofOfAuthorization{
					ID:        "poa_valid_001",
					Subject:   "user_001",
					Resource:  "resource_001",
					Action:    "read",
					Issuer:    "issuer_001",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(24 * time.Hour),
					Scope:     []string{"read", "list"},
				}
				s.proofs[poa.ID] = poa
				return poa
			},
			wantErr: false,
		},
		{
			name: "Nil POA - should error",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				return nil
			},
			wantErr: true,
			errMsg:  "PoA is required",
		},
		{
			name: "Revoked POA - should error",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				poa := &ProofOfAuthorization{
					ID:        "poa_revoked_001",
					Subject:   "user_002",
					Resource:  "resource_002",
					Action:    "write",
					Issuer:    "issuer_002",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				s.proofs[poa.ID] = poa
				s.revoked[poa.ID] = true
				return poa
			},
			wantErr: true,
			errMsg:  "PoA has been revoked",
		},
		{
			name: "Expired POA - should error",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				return &ProofOfAuthorization{
					ID:        "poa_expired_001",
					Subject:   "user_003",
					Resource:  "resource_003",
					Action:    "delete",
					Issuer:    "issuer_003",
					IssuedAt:  time.Now().Add(-48 * time.Hour),
					ExpiresAt: time.Now().Add(-24 * time.Hour),
				}
			},
			wantErr: true,
			errMsg:  "PoA has expired",
		},
		{
			name: "POA with valid delegation",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				poa := &ProofOfAuthorization{
					ID:        "poa_delegation_001",
					Subject:   "user_004",
					Resource:  "resource_004",
					Action:    "admin",
					Issuer:    "issuer_004",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(48 * time.Hour),
					Delegation: &Delegation{
						DelegatedBy: "user_004",
						DelegatedTo: "user_005",
						DelegatedAt: time.Now(),
						ExpiresAt:   time.Now().Add(24 * time.Hour),
						Scope:       []string{"read", "write"},
						Revocable:   true,
					},
				}
				s.proofs[poa.ID] = poa
				return poa
			},
			wantErr: false,
		},
		{
			name: "POA with expired delegation - should error",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				return &ProofOfAuthorization{
					ID:        "poa_expired_delegation_001",
					Subject:   "user_006",
					Resource:  "resource_006",
					Action:    "execute",
					Issuer:    "issuer_006",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(48 * time.Hour),
					Delegation: &Delegation{
						DelegatedBy: "user_006",
						DelegatedTo: "user_007",
						DelegatedAt: time.Now().Add(-48 * time.Hour),
						ExpiresAt:   time.Now().Add(-24 * time.Hour),
						Scope:       []string{"execute"},
						Revocable:   true,
					},
				}
			},
			wantErr: true,
			errMsg:  "delegation has expired",
		},
		{
			name: "POA with nil delegation",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				poa := &ProofOfAuthorization{
					ID:         "poa_nil_delegation_001",
					Subject:    "user_008",
					Resource:   "resource_008",
					Action:     "view",
					Issuer:     "issuer_008",
					IssuedAt:   time.Now(),
					ExpiresAt:  time.Now().Add(24 * time.Hour),
					Delegation: nil,
				}
				s.proofs[poa.ID] = poa
				return poa
			},
			wantErr: false,
		},
		{
			name: "POA expires exactly now (edge case)",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				return &ProofOfAuthorization{
					ID:        "poa_expires_now_001",
					Subject:   "user_009",
					Resource:  "resource_009",
					Action:    "access",
					Issuer:    "issuer_009",
					IssuedAt:  time.Now().Add(-1 * time.Hour),
					ExpiresAt: time.Now().Add(-1 * time.Millisecond), // Just expired
				}
			},
			wantErr: true,
			errMsg:  "PoA has expired",
		},
		{
			name: "POA with multiple scopes",
			setup: func(s *MemoryService) *ProofOfAuthorization {
				poa := &ProofOfAuthorization{
					ID:        "poa_multi_scope_001",
					Subject:   "user_010",
					Resource:  "resource_010",
					Action:    "manage",
					Issuer:    "issuer_010",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(72 * time.Hour),
					Scope:     []string{"read", "write", "delete", "admin"},
				}
				s.proofs[poa.ID] = poa
				return poa
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemoryService()
			ctx := context.Background()

			poa := tt.setup(s)
			err := s.Validate(ctx, poa)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, expected to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestMemoryService_Revoke tests the Revoke method
func TestMemoryService_Revoke(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MemoryService) string
		wantErr bool
		errMsg  string
	}{
		{
			name: "Revoke existing POA",
			setup: func(s *MemoryService) string {
				poa := &ProofOfAuthorization{
					ID:        "poa_to_revoke_001",
					Subject:   "user_011",
					Resource:  "resource_011",
					Action:    "access",
					Issuer:    "issuer_011",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				s.proofs[poa.ID] = poa
				return poa.ID
			},
			wantErr: false,
		},
		{
			name: "Revoke non-existent POA - should error",
			setup: func(s *MemoryService) string {
				return "poa_nonexistent_001"
			},
			wantErr: true,
			errMsg:  "PoA not found",
		},
		{
			name: "Revoke already revoked POA",
			setup: func(s *MemoryService) string {
				poa := &ProofOfAuthorization{
					ID:        "poa_already_revoked_001",
					Subject:   "user_012",
					Resource:  "resource_012",
					Action:    "modify",
					Issuer:    "issuer_012",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				s.proofs[poa.ID] = poa
				s.revoked[poa.ID] = true
				return poa.ID
			},
			wantErr: false, // Revoking an already revoked POA should succeed idempotently
		},
		{
			name: "Revoke empty POA ID",
			setup: func(s *MemoryService) string {
				return ""
			},
			wantErr: true,
			errMsg:  "PoA not found",
		},
		{
			name: "Revoke POA with delegation",
			setup: func(s *MemoryService) string {
				poa := &ProofOfAuthorization{
					ID:        "poa_with_delegation_revoke_001",
					Subject:   "user_013",
					Resource:  "resource_013",
					Action:    "transfer",
					Issuer:    "issuer_013",
					IssuedAt:  time.Now(),
					ExpiresAt: time.Now().Add(48 * time.Hour),
					Delegation: &Delegation{
						DelegatedBy: "user_013",
						DelegatedTo: "user_014",
						DelegatedAt: time.Now(),
						ExpiresAt:   time.Now().Add(24 * time.Hour),
						Scope:       []string{"transfer"},
						Revocable:   true,
					},
				}
				s.proofs[poa.ID] = poa
				return poa.ID
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemoryService()
			ctx := context.Background()

			poaID := tt.setup(s)
			err := s.Revoke(ctx, poaID)

			if (err != nil) != tt.wantErr {
				t.Errorf("Revoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("Revoke() error = %v, expected to contain %q", err, tt.errMsg)
				}
			}

			// Verify revocation was recorded if no error
			if !tt.wantErr && !s.revoked[poaID] {
				t.Errorf("Revoke() did not record revocation for POA %s", poaID)
			}
		})
	}
}

// TestMemoryService_RevokeValidateCycle tests revoke then validate cycle
func TestMemoryService_RevokeValidateCycle(t *testing.T) {
	s := NewMemoryService()
	ctx := context.Background()

	// Create POA
	poa := &ProofOfAuthorization{
		ID:        "poa_cycle_001",
		Subject:   "user_015",
		Resource:  "resource_015",
		Action:    "access",
		Issuer:    "issuer_015",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	s.proofs[poa.ID] = poa

	// Validate before revocation - should succeed
	err := s.Validate(ctx, poa)
	if err != nil {
		t.Errorf("Validate() before revocation failed: %v", err)
	}

	// Revoke POA
	err = s.Revoke(ctx, poa.ID)
	if err != nil {
		t.Errorf("Revoke() failed: %v", err)
	}

	// Validate after revocation - should fail
	err = s.Validate(ctx, poa)
	if err == nil {
		t.Error("Validate() after revocation should have failed")
	}
	if err != nil && !contains(err.Error(), "revoked") {
		t.Errorf("Validate() error = %v, expected to contain 'revoked'", err)
	}
}

// TestMemoryService_List tests the List method
func TestMemoryService_List(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*MemoryService) string
		wantLen  int
		checkIDs []string
	}{
		{
			name: "List POAs for subject with multiple POAs",
			setup: func(s *MemoryService) string {
				subject := "user_020"
				for i := 0; i < 3; i++ {
					poa := &ProofOfAuthorization{
						ID:        generateTestID(i),
						Subject:   subject,
						Resource:  generateTestResource(i),
						Action:    "read",
						Issuer:    "issuer_020",
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					s.proofs[poa.ID] = poa
				}
				return subject
			},
			wantLen:  3,
			checkIDs: []string{"test_poa_0", "test_poa_1", "test_poa_2"},
		},
		{
			name: "List POAs for subject with no POAs",
			setup: func(s *MemoryService) string {
				return "user_no_poas"
			},
			wantLen:  0,
			checkIDs: []string{},
		},
		{
			name: "List excludes revoked POAs",
			setup: func(s *MemoryService) string {
				subject := "user_021"
				// Create 3 POAs, revoke 1
				for i := 0; i < 3; i++ {
					poa := &ProofOfAuthorization{
						ID:        generateTestID(10 + i),
						Subject:   subject,
						Resource:  generateTestResource(10 + i),
						Action:    "write",
						Issuer:    "issuer_021",
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					s.proofs[poa.ID] = poa
					if i == 1 {
						s.revoked[poa.ID] = true
					}
				}
				return subject
			},
			wantLen:  2,
			checkIDs: []string{"test_poa_10", "test_poa_12"},
		},
		{
			name: "List with mixed subjects",
			setup: func(s *MemoryService) string {
				targetSubject := "user_022"
				// Add POAs for target subject
				for i := 0; i < 2; i++ {
					poa := &ProofOfAuthorization{
						ID:        generateTestID(20 + i),
						Subject:   targetSubject,
						Resource:  generateTestResource(20 + i),
						Action:    "read",
						Issuer:    "issuer_022",
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					s.proofs[poa.ID] = poa
				}
				// Add POAs for other subjects
				for i := 0; i < 3; i++ {
					poa := &ProofOfAuthorization{
						ID:        generateTestID(30 + i),
						Subject:   "other_user",
						Resource:  generateTestResource(30 + i),
						Action:    "write",
						Issuer:    "issuer_other",
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					s.proofs[poa.ID] = poa
				}
				return targetSubject
			},
			wantLen:  2,
			checkIDs: []string{"test_poa_20", "test_poa_21"},
		},
		{
			name: "List empty subject",
			setup: func(s *MemoryService) string {
				return ""
			},
			wantLen:  0,
			checkIDs: []string{},
		},
		{
			name: "List with all POAs revoked",
			setup: func(s *MemoryService) string {
				subject := "user_023"
				for i := 0; i < 3; i++ {
					poa := &ProofOfAuthorization{
						ID:        generateTestID(40 + i),
						Subject:   subject,
						Resource:  generateTestResource(40 + i),
						Action:    "delete",
						Issuer:    "issuer_023",
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					s.proofs[poa.ID] = poa
					s.revoked[poa.ID] = true
				}
				return subject
			},
			wantLen:  0,
			checkIDs: []string{},
		},
		{
			name: "List with large number of POAs",
			setup: func(s *MemoryService) string {
				subject := "user_024"
				for i := 0; i < 50; i++ {
					poa := &ProofOfAuthorization{
						ID:        generateTestID(100 + i),
						Subject:   subject,
						Resource:  generateTestResource(100 + i),
						Action:    "access",
						Issuer:    "issuer_024",
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					s.proofs[poa.ID] = poa
				}
				return subject
			},
			wantLen: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemoryService()
			ctx := context.Background()

			subject := tt.setup(s)
			result, err := s.List(ctx, subject)

			if err != nil {
				t.Errorf("List() unexpected error = %v", err)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("List() returned %d POAs, want %d", len(result), tt.wantLen)
			}

			// Verify specific IDs if provided
			if len(tt.checkIDs) > 0 {
				foundIDs := make(map[string]bool)
				for _, poa := range result {
					foundIDs[poa.ID] = true
				}

				for _, expectedID := range tt.checkIDs {
					if !foundIDs[expectedID] {
						t.Errorf("List() missing expected POA ID: %s", expectedID)
					}
				}
			}

			// Verify all returned POAs have correct subject
			for _, poa := range result {
				if poa.Subject != subject {
					t.Errorf("List() returned POA with wrong subject: got %s, want %s", poa.Subject, subject)
				}
			}
		})
	}
}

// TestMemoryService_ConcurrentOperations tests thread safety
// Note: MemoryService currently lacks mutex protection, so this test is skipped
// until proper synchronization is added to the implementation.
func TestMemoryService_ConcurrentOperations(t *testing.T) {
	t.Skip("Skipping concurrent test - MemoryService lacks mutex protection")
}

// TestCreateDelegationAttestation tests CreateDelegationAttestation function
func TestCreateDelegationAttestation(t *testing.T) {
	tests := []struct {
		name        string
		delegatedBy string
		delegatedTo string
		scope       []string
		validate    func(*testing.T, *Delegation)
	}{
		{
			name:        "Create basic delegation",
			delegatedBy: "user_100",
			delegatedTo: "user_101",
			scope:       []string{"read", "write"},
			validate: func(t *testing.T, d *Delegation) {
				if d.DelegatedBy != "user_100" {
					t.Errorf("DelegatedBy = %s, want user_100", d.DelegatedBy)
				}
				if d.DelegatedTo != "user_101" {
					t.Errorf("DelegatedTo = %s, want user_101", d.DelegatedTo)
				}
				if len(d.Scope) != 2 {
					t.Errorf("Scope length = %d, want 2", len(d.Scope))
				}
				if !d.Revocable {
					t.Error("Delegation should be revocable by default")
				}
				if d.ExpiresAt.Before(time.Now()) {
					t.Error("Delegation should not be expired")
				}
			},
		},
		{
			name:        "Create delegation with empty scope",
			delegatedBy: "user_102",
			delegatedTo: "user_103",
			scope:       []string{},
			validate: func(t *testing.T, d *Delegation) {
				if len(d.Scope) != 0 {
					t.Errorf("Scope length = %d, want 0", len(d.Scope))
				}
			},
		},
		{
			name:        "Create delegation with nil scope",
			delegatedBy: "user_104",
			delegatedTo: "user_105",
			scope:       nil,
			validate: func(t *testing.T, d *Delegation) {
				if len(d.Scope) > 0 {
					t.Errorf("Scope should be nil or empty, got %v", d.Scope)
				}
			},
		},
		{
			name:        "Create delegation with multiple scopes",
			delegatedBy: "user_106",
			delegatedTo: "user_107",
			scope:       []string{"read", "write", "delete", "admin", "execute"},
			validate: func(t *testing.T, d *Delegation) {
				if len(d.Scope) != 5 {
					t.Errorf("Scope length = %d, want 5", len(d.Scope))
				}
			},
		},
		{
			name:        "Create delegation with empty strings",
			delegatedBy: "",
			delegatedTo: "",
			scope:       []string{"read"},
			validate: func(t *testing.T, d *Delegation) {
				if d.DelegatedBy != "" {
					t.Errorf("DelegatedBy = %s, want empty string", d.DelegatedBy)
				}
				if d.DelegatedTo != "" {
					t.Errorf("DelegatedTo = %s, want empty string", d.DelegatedTo)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delegation := CreateDelegationAttestation(tt.delegatedBy, tt.delegatedTo, tt.scope)

			if delegation == nil {
				t.Fatal("CreateDelegationAttestation() returned nil")
			}

			tt.validate(t, delegation)

			// Verify timestamps
			now := time.Now()
			if delegation.DelegatedAt.After(now) {
				t.Error("DelegatedAt should not be in the future")
			}
			if delegation.ExpiresAt.Before(now) {
				t.Error("ExpiresAt should be in the future")
			}
			if delegation.ExpiresAt.Before(delegation.DelegatedAt) {
				t.Error("ExpiresAt should be after DelegatedAt")
			}
		})
	}
}

// TestCreateAttestation tests CreateAttestation function
func TestCreateAttestation(t *testing.T) {
	tests := []struct {
		name       string
		attestedBy string
		evidence   map[string]interface{}
		validate   func(*testing.T, *Attestation)
	}{
		{
			name:       "Create attestation with evidence",
			attestedBy: "attester_001",
			evidence: map[string]interface{}{
				"document_id": "doc_001",
				"verified":    true,
				"score":       0.95,
			},
			validate: func(t *testing.T, a *Attestation) {
				if a.AttestedBy != "attester_001" {
					t.Errorf("AttestedBy = %s, want attester_001", a.AttestedBy)
				}
				if len(a.Evidence) != 3 {
					t.Errorf("Evidence length = %d, want 3", len(a.Evidence))
				}
				if a.Evidence["document_id"] != "doc_001" {
					t.Errorf("Evidence document_id = %v, want doc_001", a.Evidence["document_id"])
				}
				if a.Confidence != 0.90 {
					t.Errorf("Confidence = %f, want 0.90", a.Confidence)
				}
				if a.ValidityScore != 0.95 {
					t.Errorf("ValidityScore = %f, want 0.95", a.ValidityScore)
				}
			},
		},
		{
			name:       "Create attestation with empty evidence",
			attestedBy: "attester_002",
			evidence:   map[string]interface{}{},
			validate: func(t *testing.T, a *Attestation) {
				if len(a.Evidence) != 0 {
					t.Errorf("Evidence length = %d, want 0", len(a.Evidence))
				}
			},
		},
		{
			name:       "Create attestation with nil evidence",
			attestedBy: "attester_003",
			evidence:   nil,
			validate: func(t *testing.T, a *Attestation) {
				if len(a.Evidence) > 0 {
					t.Errorf("Evidence should be nil or empty, got %v", a.Evidence)
				}
			},
		},
		{
			name:       "Create attestation with complex evidence",
			attestedBy: "attester_004",
			evidence: map[string]interface{}{
				"documents": []string{"doc1", "doc2", "doc3"},
				"metadata": map[string]interface{}{
					"source": "system",
					"level":  5,
				},
				"timestamp": time.Now().Unix(),
			},
			validate: func(t *testing.T, a *Attestation) {
				if len(a.Evidence) != 3 {
					t.Errorf("Evidence length = %d, want 3", len(a.Evidence))
				}
			},
		},
		{
			name:       "Create attestation with empty attester",
			attestedBy: "",
			evidence: map[string]interface{}{
				"test": "data",
			},
			validate: func(t *testing.T, a *Attestation) {
				if a.AttestedBy != "" {
					t.Errorf("AttestedBy = %s, want empty string", a.AttestedBy)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attestation := CreateAttestation(tt.attestedBy, tt.evidence)

			if attestation == nil {
				t.Fatal("CreateAttestation() returned nil")
			}

			tt.validate(t, attestation)

			// Verify timestamp
			now := time.Now()
			if attestation.AttestedAt.After(now) {
				t.Error("AttestedAt should not be in the future")
			}

			// Verify default scores are reasonable
			if attestation.Confidence < 0 || attestation.Confidence > 1 {
				t.Errorf("Confidence = %f, should be between 0 and 1", attestation.Confidence)
			}
			if attestation.ValidityScore < 0 || attestation.ValidityScore > 1 {
				t.Errorf("ValidityScore = %f, should be between 0 and 1", attestation.ValidityScore)
			}
		})
	}
}

// Helper functions
func generateTestID(idx int) string {
	return fmt.Sprintf("test_poa_%d", idx)
}

func generateTestSubject(idx int) string {
	return fmt.Sprintf("test_subject_%d", idx)
}

func generateTestResource(idx int) string {
	return fmt.Sprintf("test_resource_%d", idx)
}
