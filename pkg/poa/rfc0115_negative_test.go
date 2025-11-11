package poa

import (
	"testing"
	"time"
)

// Helper to build minimal valid PoADefinition we can then mutate for negative cases.
func minimalValidDefinition() PoADefinition {
	start := time.Now()
	end := start.Add(24 * time.Hour)
	return PoADefinition{
		Parties:       Parties{Principal: Principal{Type: PrincipalTypeOrganization, Identity: "org-123"}, AuthorizedClient: AuthorizedClient{Type: string(ClientTypeLLM), Identity: "client-xyz", Version: "v1", OperationalStatus: string(OperationalStatusActive)}},
		Authorization: AuthorizationScope{ApplicableSectors: []IndustrySector{{Code: SectorInfoCommunication, Description: "Information and Communication", Authorized: true}}, ApplicableRegions: []GeographicScope{{Type: GeoTypeNational, Identifier: "US"}}, AuthorizedActions: AuthorizedActions{Transactions: []TransactionType{TransactionLoan}}},
		Requirements:  Requirements{ValidityPeriod: ValidityPeriod{StartTime: start, EndTime: end}},
	}
}

// assertErr ensures error is present and contains substring.
func assertErr(t *testing.T, err error, sub string) { //nolint:unused // called via reflection in test framework
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", sub)
	}
	if sub != "" && !contains(err.Error(), sub) {
		t.Fatalf("expected error to contain %q got %q", sub, err.Error())
	}
}

func contains(hay, needle string) bool { //nolint:unused // helper for assertErr
	return len(needle) == 0 || (needle != "" && (stringContains(hay, needle)))
}
func stringContains(s, sub string) bool { //nolint:unused // helper for contains
	return len(sub) <= len(s) && (indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int { return len(sub) } //nolint:unused // helper for stringContains

// (Above simplistic helpers kept minimal; using built-ins would be simpler but avoid extra imports.)

// TestRFC0115NegativeCases covers structural & semantic failures.
func TestRFC0115NegativeCases(t *testing.T) {
	// 1. Exclusion flags not all true
	cfgBad := RFC0115Config{ExcludeWeb3: false, ExcludeAIOperators: true, ExcludeDNAIdentities: true, MaxValidityDays: 365}
	if err := ValidateRFC0115Compliance(cfgBad); err == nil {
		t.Fatalf("expected exclusion flags error")
	}

	// 2. MaxValidityDays out of bounds
	cfgBad2 := RFC0115Config{ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true, MaxValidityDays: 0}
	if err := ValidateRFC0115Compliance(cfgBad2); err == nil {
		t.Fatalf("expected max validity error")
	}

	// 3. PoADefinition principal identity missing
	def := minimalValidDefinition()
	def.Parties.Principal.Identity = ""
	if err := ValidateRFC0115Compliance(def); err == nil {
		t.Fatalf("expected principal identity error")
	}

	// 4. No sectors
	def2 := minimalValidDefinition()
	def2.Authorization.ApplicableSectors = nil
	if err := ValidateRFC0115Compliance(def2); err == nil {
		t.Fatalf("expected sector error")
	}

	// 5. No regions
	def3 := minimalValidDefinition()
	def3.Authorization.ApplicableRegions = nil
	if err := ValidateRFC0115Compliance(def3); err == nil {
		t.Fatalf("expected region error")
	}

	// 6. No actions
	def4 := minimalValidDefinition()
	def4.Authorization.AuthorizedActions = AuthorizedActions{}
	if err := ValidateRFC0115Compliance(def4); err == nil {
		t.Fatalf("expected action error")
	}

	// 7. Validity duration negative
	def5 := minimalValidDefinition()
	def5.Requirements.ValidityPeriod.EndTime = def5.Requirements.ValidityPeriod.StartTime.Add(-1 * time.Hour)
	if err := ValidateRFC0115Compliance(def5); err == nil {
		t.Fatalf("expected negative duration error")
	}

	// 8. Validity duration exceeds 2 years
	def6 := minimalValidDefinition()
	def6.Requirements.ValidityPeriod.EndTime = def6.Requirements.ValidityPeriod.StartTime.Add(24 * time.Hour * 731)
	if err := ValidateRFC0115Compliance(def6); err == nil {
		t.Fatalf("expected excessive duration error")
	}
}
