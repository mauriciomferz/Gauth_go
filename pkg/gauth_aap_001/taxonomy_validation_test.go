package gauth_aap_001

import "testing"

// TestValidateTaxonomyPositive ensures allowed values pass.
func TestValidateTaxonomyPositive(t *testing.T) {
	p := &PowerOfAttorney{Version: 3, AgentType: "human", Sector: "finance", ActionClass: "read_ops"}
	if err := ValidateTaxonomy(p); err != nil {
		t.Fatalf("expected taxonomy valid: %v", err)
	}
	// Empty fields also valid
	p2 := &PowerOfAttorney{Version: 3}
	if err := ValidateTaxonomy(p2); err != nil {
		t.Fatalf("empty taxonomy should be valid: %v", err)
	}
}

// TestValidateTaxonomyNegative ensures invalid enumeration values are rejected.
func TestValidateTaxonomyNegative(t *testing.T) {
	cases := []PowerOfAttorney{
		{Version: 3, AgentType: "alien"},
		{Version: 3, Sector: "galactic"},
		{Version: 3, ActionClass: "root_access"},
	}
	for i, c := range cases {
		if err := ValidateTaxonomy(&c); err == nil {
			t.Fatalf("case %d expected invalid taxonomy", i)
		}
	}
	// Version <3 should ignore invalid values (backward compatibility)
	legacy := &PowerOfAttorney{Version: 2, AgentType: "alien"}
	if err := ValidateTaxonomy(legacy); err != nil {
		t.Fatalf("legacy version should ignore taxonomy validation: %v", err)
	}
}
