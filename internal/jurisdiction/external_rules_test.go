package jurisdiction

import (
	"os"
	"testing"
)

// TestExternalRulesLoad verifies that external jurisdiction rules file overrides defaults when provided.
func TestExternalRulesLoad(t *testing.T) {
	// Create temporary rules file
	f, err := os.CreateTemp(t.TempDir(), "jurisdiction_rules_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer f.Close()
	json := `{
	  "jurisdictions": [
	    {
	      "jurisdiction": "UNITED_STATES",
	      "strict_mode": true,
	      "allowed_actions": ["transfer"],
	      "blocked_actions": ["high_value_transfer"],
	      "cross_border_rules": {"transfer": ["CANADA"]},
	      "data_residency_rules": {"personal_data": true}
	    }
	  ]
	}`
	if _, err := f.WriteString(json); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Set env to point to file
	os.Setenv("GAUTH_JURISDICTION_RULES_PATH", f.Name())
	defer os.Unsetenv("GAUTH_JURISDICTION_RULES_PATH")

	eng := NewEnforcementEngine()
	enf, err := eng.GetJurisdictionEnforcement("UNITED_STATES")
	if err != nil {
		t.Fatalf("enforcement fetch failed: %v", err)
	}
	if !enf.StrictMode {
		t.Fatalf("expected strict_mode true from external file")
	}
	if !enf.AllowedActions["transfer"] || enf.AllowedActions["high_value_transfer"] {
		t.Fatalf("allowed actions not applied correctly: %+v", enf.AllowedActions)
	}
	if !enf.BlockedActions["high_value_transfer"] {
		t.Fatalf("blocked action missing")
	}
	if enf.CrossBorderRules["transfer"][0] != "CANADA" {
		t.Fatalf("cross border rule not loaded: %+v", enf.CrossBorderRules)
	}
	if !enf.DataResidencyRules["personal_data"] {
		t.Fatalf("data residency rule missing")
	}
}
