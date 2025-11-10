package registry

import (
	"context"
	"testing"
	"time"
)

// TestMockCommercialRegisterService_VerifyRegistration tests registration verification
func TestMockCommercialRegisterService_VerifyRegistration(t *testing.T) {
	svc := NewMockCommercialRegisterService()
	ctx := context.Background()

	t.Run("Valid German GmbH registration", func(t *testing.T) {
		req := &RegistrationVerificationRequest{
			EntityName:         "Test Technologies GmbH",
			RegistrationNumber: "HRB12345",
			Jurisdiction:       "DE",
			EntityType:         "GmbH",
		}

		result, err := svc.VerifyRegistration(ctx, req)
		if err != nil {
			t.Fatalf("VerifyRegistration() unexpected error: %v", err)
		}

		if !result.Verified {
			t.Errorf("Verified = %v, want true", result.Verified)
		}

		if result.RegistrationNumber != "HRB 12345" {
			t.Errorf("RegistrationNumber = %v, want HRB 12345", result.RegistrationNumber)
		}

		if result.EntityName != "Test Technologies GmbH" {
			t.Errorf("EntityName = %v, want Test Technologies GmbH", result.EntityName)
		}

		if result.EntityType != "GmbH" {
			t.Errorf("EntityType = %v, want GmbH", result.EntityType)
		}

		if result.Jurisdiction != "DE" {
			t.Errorf("Jurisdiction = %v, want DE", result.Jurisdiction)
		}

		if result.Status != "active" {
			t.Errorf("Status = %v, want active", result.Status)
		}

		if result.RegisterName != "Handelsregister" {
			t.Errorf("RegisterName = %v, want Handelsregister", result.RegisterName)
		}

		if result.VerificationMethod != "mock_registry_api" {
			t.Errorf("VerificationMethod = %v, want mock_registry_api", result.VerificationMethod)
		}
	})

	t.Run("Valid UK Limited Company registration", func(t *testing.T) {
		req := &RegistrationVerificationRequest{
			EntityName:         "Test Technologies Ltd",
			RegistrationNumber: "12345678",
			Jurisdiction:       "GB",
			EntityType:         "Ltd",
		}

		result, err := svc.VerifyRegistration(ctx, req)
		if err != nil {
			t.Fatalf("VerifyRegistration() unexpected error: %v", err)
		}

		if !result.Verified {
			t.Errorf("Verified = %v, want true", result.Verified)
		}

		if result.Jurisdiction != "GB" {
			t.Errorf("Jurisdiction = %v, want GB", result.Jurisdiction)
		}

		if result.RegisterName != "Companies House" {
			t.Errorf("RegisterName = %v, want Companies House", result.RegisterName)
		}

		// EntityType returns LegalForm which is "Private Limited Company" for UK
		if result.EntityType == "" {
			t.Error("EntityType is empty")
		}
	})

	t.Run("Invalid registration - not found", func(t *testing.T) {
		req := &RegistrationVerificationRequest{
			EntityName:         "Non-Existent Company",
			RegistrationNumber: "INVALID999",
			Jurisdiction:       "DE",
		}

		result, err := svc.VerifyRegistration(ctx, req)
		if err != nil {
			t.Fatalf("VerifyRegistration() unexpected error: %v", err)
		}

		if result.Verified {
			t.Error("Verified = true, want false for non-existent company")
		}
	})

	t.Run("Missing registration number", func(t *testing.T) {
		req := &RegistrationVerificationRequest{
			EntityName:   "Some Company",
			Jurisdiction: "DE",
			// Missing RegistrationNumber
		}

		result, err := svc.VerifyRegistration(ctx, req)
		if err != nil {
			t.Fatalf("VerifyRegistration() unexpected error: %v", err)
		}

		// Mock returns unverified result, not error
		if result.Verified {
			t.Error("Verified = true, want false for missing registration number")
		}
	})

	t.Run("Missing jurisdiction", func(t *testing.T) {
		req := &RegistrationVerificationRequest{
			EntityName:         "Test Company",
			RegistrationNumber: "HRB12345",
			// Missing Jurisdiction
		}

		result, err := svc.VerifyRegistration(ctx, req)
		if err != nil {
			t.Fatalf("VerifyRegistration() unexpected error: %v", err)
		}

		// Mock returns unverified result, not error
		if result.Verified {
			t.Error("Verified = true, want false for missing jurisdiction")
		}
	})
}

// TestMockCommercialRegisterService_VerifyAuthorizedRepresentative tests representative verification
func TestMockCommercialRegisterService_VerifyAuthorizedRepresentative(t *testing.T) {
	svc := NewMockCommercialRegisterService()
	ctx := context.Background()

	t.Run("Valid managing director", func(t *testing.T) {
		req := &RepresentativeVerificationRequest{
			RepresentativeName: "Dr. Max Müller",
			EntityRegistration: "HRB12345",
			Jurisdiction:       "DE",
			AuthorityType:      "managing_director",
		}

		result, err := svc.VerifyAuthorizedRepresentative(ctx, req)
		if err != nil {
			t.Fatalf("VerifyAuthorizedRepresentative() unexpected error: %v", err)
		}

		if !result.Verified {
			t.Errorf("Verified = %v, want true", result.Verified)
		}

		if result.Position != "Geschäftsführer" {
			t.Errorf("Position = %v, want Geschäftsführer", result.Position)
		}

		if result.AuthorityType != "managing_director" {
			t.Errorf("AuthorityType = %v, want managing_director", result.AuthorityType)
		}

		if result.AuthorityScope != "unlimited" {
			t.Errorf("AuthorityScope = %v, want unlimited", result.AuthorityScope)
		}

		if result.SignatureAuthority != "sole" {
			t.Errorf("SignatureAuthority = %v, want sole", result.SignatureAuthority)
		}
	})

	t.Run("Valid Prokura holder", func(t *testing.T) {
		req := &RepresentativeVerificationRequest{
			RepresentativeName: "Erika Musterfrau",
			EntityRegistration: "HRB12345",
			Jurisdiction:       "DE",
			AuthorityType:      "prokura",
		}

		result, err := svc.VerifyAuthorizedRepresentative(ctx, req)
		if err != nil {
			t.Fatalf("VerifyAuthorizedRepresentative() unexpected error: %v", err)
		}

		if !result.Verified {
			t.Errorf("Verified = %v, want true", result.Verified)
		}

		if result.AuthorityType != "prokura" {
			t.Errorf("AuthorityType = %v, want prokura", result.AuthorityType)
		}

		if result.Position != "Prokuristin" {
			t.Errorf("Position = %v, want Prokuristin", result.Position)
		}
	})

	t.Run("Invalid representative - not found", func(t *testing.T) {
		req := &RepresentativeVerificationRequest{
			RepresentativeName: "Unknown Person",
			EntityRegistration: "HRB12345",
			Jurisdiction:       "DE",
			AuthorityType:      "unknown_type",
		}

		result, err := svc.VerifyAuthorizedRepresentative(ctx, req)
		if err != nil {
			t.Fatalf("VerifyAuthorizedRepresentative() unexpected error: %v", err)
		}

		if result.Verified {
			t.Error("Verified = true, want false for unknown representative")
		}
	})

	t.Run("Invalid entity registration", func(t *testing.T) {
		req := &RepresentativeVerificationRequest{
			RepresentativeName: "Dr. Max Müller",
			EntityRegistration: "INVALID999",
			Jurisdiction:       "DE",
			AuthorityType:      "managing_director",
		}

		result, err := svc.VerifyAuthorizedRepresentative(ctx, req)
		if err != nil {
			t.Fatalf("VerifyAuthorizedRepresentative() unexpected error: %v", err)
		}

		if result.Verified {
			t.Error("Verified = true, want false for invalid entity")
		}
	})

	t.Run("Missing required fields", func(t *testing.T) {
		req := &RepresentativeVerificationRequest{
			RepresentativeName: "Dr. Max Müller",
			// Missing EntityRegistration, Jurisdiction, AuthorityType
		}

		result, err := svc.VerifyAuthorizedRepresentative(ctx, req)
		if err != nil {
			t.Fatalf("VerifyAuthorizedRepresentative() unexpected error: %v", err)
		}

		// Mock returns unverified result, not error
		if result.Verified {
			t.Error("Verified = true, want false for missing fields")
		}
	})
}

// TestMockCommercialRegisterService_VerifyProkura tests Prokura verification
func TestMockCommercialRegisterService_VerifyProkura(t *testing.T) {
	svc := NewMockCommercialRegisterService()
	ctx := context.Background()

	t.Run("Valid Einzelprokura", func(t *testing.T) {
		req := &ProkuraVerificationRequest{
			ProkuraHolder:      "Erika Musterfrau",
			EntityRegistration: "HRB12345",
			Jurisdiction:       "DE",
			ProkuraType:        "einzelprokura",
		}

		result, err := svc.VerifyProkura(ctx, req)
		if err != nil {
			t.Fatalf("VerifyProkura() unexpected error: %v", err)
		}

		if !result.Verified {
			t.Errorf("Verified = %v, want true", result.Verified)
		}

		if result.ProkuraType != "einzelprokura" {
			t.Errorf("ProkuraType = %v, want einzelprokura", result.ProkuraType)
		}

		if result.Scope != "all_business_transactions" {
			t.Errorf("Scope = %v, want all_business_transactions", result.Scope)
		}

		if result.JointRepresentation {
			t.Error("JointRepresentation = true, want false for Einzelprokura")
		}

		if result.Status != "active" {
			t.Errorf("Status = %v, want active", result.Status)
		}
	})

	t.Run("Non-existent entity", func(t *testing.T) {
		req := &ProkuraVerificationRequest{
			ProkuraHolder:      "Klaus Weber",
			EntityRegistration: "HRB67890",
			Jurisdiction:       "DE",
			ProkuraType:        "gesamtprokura",
		}

		result, err := svc.VerifyProkura(ctx, req)
		if err != nil {
			t.Fatalf("VerifyProkura() unexpected error: %v", err)
		}

		if result.Verified {
			t.Error("Expected unverified result for non-existent entity")
		}
	})

	t.Run("Invalid Prokura - not found", func(t *testing.T) {
		req := &ProkuraVerificationRequest{
			ProkuraHolder:      "Unknown Person",
			EntityRegistration: "HRB12345",
			Jurisdiction:       "DE",
			ProkuraType:        "einzelprokura",
		}

		result, err := svc.VerifyProkura(ctx, req)
		if err != nil {
			t.Fatalf("VerifyProkura() unexpected error: %v", err)
		}

		if result.Verified {
			t.Error("Verified = true, want false for unknown Prokura holder")
		}
	})

	t.Run("Revoked Prokura", func(t *testing.T) {
		req := &ProkuraVerificationRequest{
			ProkuraHolder:      "Revoked Holder",
			EntityRegistration: "HRB99999",
			Jurisdiction:       "DE",
			ProkuraType:        "einzelprokura",
		}

		result, err := svc.VerifyProkura(ctx, req)
		if err != nil {
			t.Fatalf("VerifyProkura() unexpected error: %v", err)
		}

		// Mock should return verified=true but status=revoked for this test
		if result.Verified && result.Status == "revoked" {
			// This is acceptable - entity exists but is revoked
			return
		}

		// Otherwise it should not be verified
		if result.Verified {
			t.Error("Verified = true, want false for revoked Prokura")
		}
	})

	t.Run("Missing required fields", func(t *testing.T) {
		req := &ProkuraVerificationRequest{
			ProkuraHolder: "Anna Schmidt",
			// Missing EntityRegistration, Jurisdiction, ProkuraType
		}

		result, err := svc.VerifyProkura(ctx, req)
		if err != nil {
			t.Fatalf("VerifyProkura() unexpected error: %v", err)
		}

		// Mock returns unverified result, not error
		if result.Verified {
			t.Error("Verified = true, want false for missing fields")
		}
	})
}

// TestMockCommercialRegisterService_GetEntityDetails tests entity details retrieval
func TestMockCommercialRegisterService_GetEntityDetails(t *testing.T) {
	svc := NewMockCommercialRegisterService()
	ctx := context.Background()

	t.Run("Valid German GmbH details", func(t *testing.T) {
		details, err := svc.GetEntityDetails(ctx, "HRB12345", "DE")
		if err != nil {
			t.Fatalf("GetEntityDetails() unexpected error: %v", err)
		}

		if details.RegistrationNumber != "HRB 12345" {
			t.Errorf("RegistrationNumber = %v, want HRB 12345", details.RegistrationNumber)
		}

		if details.EntityName != "Test Technologies GmbH" {
			t.Errorf("EntityName = %v, want Test Technologies GmbH", details.EntityName)
		}

		if details.LegalForm != "GmbH" {
			t.Errorf("LegalForm = %v, want GmbH", details.LegalForm)
		}

		if details.Status != "active" {
			t.Errorf("Status = %v, want active", details.Status)
		}

		if details.Capital == nil {
			t.Error("Capital is nil, expected capital information")
		} else {
			if details.Capital.RegisteredCapital != 25000 {
				t.Errorf("RegisteredCapital = %v, want 25000", details.Capital.RegisteredCapital)
			}
			if details.Capital.Currency != "EUR" {
				t.Errorf("Currency = %v, want EUR", details.Capital.Currency)
			}
		}

		if len(details.ManagingDirectors) == 0 {
			t.Error("ManagingDirectors is empty, expected at least one director")
		}

		if details.RegisteredAddress.Country != "DE" {
			t.Errorf("Country = %v, want DE", details.RegisteredAddress.Country)
		}
	})

	t.Run("Valid UK Limited Company details", func(t *testing.T) {
		details, err := svc.GetEntityDetails(ctx, "12345678", "GB")
		if err != nil {
			t.Fatalf("GetEntityDetails() unexpected error: %v", err)
		}

		if details.RegistrationNumber != "12345678" {
			t.Errorf("RegistrationNumber = %v, want 12345678", details.RegistrationNumber)
		}

		if details.EntityName == "" {
			t.Error("EntityName is empty")
		}

		if details.LegalForm == "" {
			t.Error("LegalForm is empty")
		}

		if details.RegisteredAddress.Country != "GB" {
			t.Errorf("Country = %v, want GB", details.RegisteredAddress.Country)
		}
	})

	t.Run("Invalid registration - not found", func(t *testing.T) {
		details, err := svc.GetEntityDetails(ctx, "INVALID999", "DE")
		if err == nil {
			t.Error("Expected error for invalid registration, got nil")
		}

		if details != nil {
			t.Errorf("Expected nil details for error case, got %v", details)
		}
	})

	t.Run("Missing registration ID", func(t *testing.T) {
		details, err := svc.GetEntityDetails(ctx, "", "DE")
		if err == nil {
			t.Error("Expected error for missing registration ID, got nil")
		}

		if details != nil {
			t.Errorf("Expected nil details for error case, got %v", details)
		}
	})

	t.Run("Missing jurisdiction", func(t *testing.T) {
		details, err := svc.GetEntityDetails(ctx, "HRB12345", "")
		if err == nil {
			t.Error("Expected error for missing jurisdiction, got nil")
		}

		if details != nil {
			t.Errorf("Expected nil details for error case, got %v", details)
		}
	})
}

// TestMockCommercialRegisterService_GetAuthorizedSignatories tests signatory retrieval
func TestMockCommercialRegisterService_GetAuthorizedSignatories(t *testing.T) {
	svc := NewMockCommercialRegisterService()
	ctx := context.Background()

	t.Run("Valid signatories for German GmbH", func(t *testing.T) {
		signatories, err := svc.GetAuthorizedSignatories(ctx, "HRB12345", "DE")
		if err != nil {
			t.Fatalf("GetAuthorizedSignatories() unexpected error: %v", err)
		}

		if len(signatories) == 0 {
			t.Error("Signatories is empty, expected at least one signatory")
		}

		// Check for managing director
		foundDirector := false
		for _, sig := range signatories {
			if sig.AuthorityType == "managing_director" {
				foundDirector = true
				if sig.SignatureAuthority == "" {
					t.Error("SignatureAuthority is empty for managing director")
				}
				if sig.Position == "" {
					t.Error("Position is empty for managing director")
				}
			}
		}

		if !foundDirector {
			t.Error("No managing director found in signatories")
		}
	})

	t.Run("Valid signatories for UK Limited Company", func(t *testing.T) {
		signatories, err := svc.GetAuthorizedSignatories(ctx, "12345678", "GB")
		if err != nil {
			t.Fatalf("GetAuthorizedSignatories() unexpected error: %v", err)
		}

		if len(signatories) == 0 {
			t.Error("Signatories is empty, expected at least one signatory")
		}

		// Check for director
		foundDirector := false
		for _, sig := range signatories {
			if sig.Position == "Director" || sig.AuthorityType == "managing_director" {
				foundDirector = true
			}
		}

		if !foundDirector {
			t.Error("No director found in signatories")
		}
	})

	t.Run("Invalid registration - not found", func(t *testing.T) {
		signatories, err := svc.GetAuthorizedSignatories(ctx, "INVALID999", "DE")
		if err == nil {
			t.Error("Expected error for invalid registration, got nil")
		}

		if signatories != nil {
			t.Errorf("Expected nil signatories for error case, got %v", signatories)
		}
	})

	t.Run("Missing registration ID", func(t *testing.T) {
		signatories, err := svc.GetAuthorizedSignatories(ctx, "", "DE")
		if err == nil {
			t.Error("Expected error for missing registration ID, got nil")
		}

		if signatories != nil {
			t.Errorf("Expected nil signatories for error case, got %v", signatories)
		}
	})
}

// TestEntityDetails_Validation tests entity details structure validation
func TestEntityDetails_Validation(t *testing.T) {
	now := time.Now()

	t.Run("Valid entity details", func(t *testing.T) {
		details := &EntityDetails{
			RegistrationNumber: "HRB 12345",
			EntityName:         "Test GmbH",
			LegalForm:          "GmbH",
			RegisteredAddress: Address{
				Street:     "Teststr. 1",
				City:       "Berlin",
				PostalCode: "10115",
				Country:    "DE",
			},
			RegistrationDate: now.Add(-365 * 24 * time.Hour),
			Status:           "active",
			Capital: &CapitalInfo{
				RegisteredCapital: 25000,
				PaidUpCapital:     25000,
				Currency:          "EUR",
			},
			ManagingDirectors: []Signatory{
				{
					Name:               "Test Director",
					Position:           "Geschäftsführer",
					AuthorityType:      "managing_director",
					SignatureAuthority: "sole",
					AppointmentDate:    now.Add(-365 * 24 * time.Hour),
					ValidFrom:          now.Add(-365 * 24 * time.Hour),
				},
			},
			LastUpdated: now,
		}

		if details.RegistrationNumber == "" {
			t.Error("RegistrationNumber is empty")
		}

		if details.EntityName == "" {
			t.Error("EntityName is empty")
		}

		if details.Status != "active" {
			t.Errorf("Status = %v, want active", details.Status)
		}

		if len(details.ManagingDirectors) == 0 {
			t.Error("ManagingDirectors is empty")
		}
	})

	t.Run("Entity with multiple directors", func(t *testing.T) {
		details := &EntityDetails{
			RegistrationNumber: "HRB 67890",
			EntityName:         "Multi-Director GmbH",
			LegalForm:          "GmbH",
			ManagingDirectors: []Signatory{
				{
					Name:          "Director 1",
					AuthorityType: "managing_director",
				},
				{
					Name:          "Director 2",
					AuthorityType: "managing_director",
				},
			},
		}

		if len(details.ManagingDirectors) != 2 {
			t.Errorf("ManagingDirectors count = %d, want 2", len(details.ManagingDirectors))
		}
	})

	t.Run("Entity with Prokura holders", func(t *testing.T) {
		details := &EntityDetails{
			RegistrationNumber: "HRB 11111",
			EntityName:         "Prokura Test GmbH",
			LegalForm:          "GmbH",
			AuthorizedSignatories: []Signatory{
				{
					Name:               "Prokurist 1",
					Position:           "Prokurist",
					AuthorityType:      "prokura",
					SignatureAuthority: "sole",
				},
			},
		}

		if len(details.AuthorizedSignatories) == 0 {
			t.Error("AuthorizedSignatories is empty")
		}

		if details.AuthorizedSignatories[0].AuthorityType != "prokura" {
			t.Errorf("AuthorityType = %v, want prokura", details.AuthorizedSignatories[0].AuthorityType)
		}
	})
}

// Benchmark tests
func BenchmarkMockCommercialRegisterService_VerifyRegistration(b *testing.B) {
	svc := NewMockCommercialRegisterService()
	ctx := context.Background()

	req := &RegistrationVerificationRequest{
		EntityName:         "Test Technologies GmbH",
		RegistrationNumber: "HRB 12345",
		Jurisdiction:       "DE",
		EntityType:         "GmbH",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.VerifyRegistration(ctx, req)
	}
}

func BenchmarkMockCommercialRegisterService_VerifyAuthorizedRepresentative(b *testing.B) {
	svc := NewMockCommercialRegisterService()
	ctx := context.Background()

	req := &RepresentativeVerificationRequest{
		RepresentativeName: "Dr. Max Müller",
		EntityRegistration: "HRB 12345",
		Jurisdiction:       "DE",
		AuthorityType:      "managing_director",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.VerifyAuthorizedRepresentative(ctx, req)
	}
}

func BenchmarkMockCommercialRegisterService_GetEntityDetails(b *testing.B) {
	svc := NewMockCommercialRegisterService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetEntityDetails(ctx, "HRB12345", "DE")
	}
}
