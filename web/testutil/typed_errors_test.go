package testutil

import "testing"

func TestUnknownCapabilityTypedError(t *testing.T) {
	_, err := ParseCapabilityRegistry(CapAlphaUnknownMapping)
	if err == nil {
		t.Fatalf("expected error for unknown capability mapping")
	}
	te, ok := err.(*CapabilityRegistryError)
	if !ok {
		t.Fatalf("expected CapabilityRegistryError type; got %T", err)
	}
	if te.Kind != RegistryErrorUnknownCapability {
		t.Fatalf("expected kind RegistryErrorUnknownCapability; got %v", te.Kind)
	}
	if te.CapabilityID != "cap.unknown" {
		t.Fatalf("expected CapabilityID 'cap.unknown'; got %s", te.CapabilityID)
	}
	if te.Action != "delegation:create" {
		t.Fatalf("expected Action 'delegation:create'; got %s", te.Action)
	}
}

func TestEmptyActionMappingTypedError(t *testing.T) {
	// Synthesize a registry with an empty mapping by editing a valid fixture.
	raw := `{"schema_version":1,"capabilities":[{"id":"cap.alpha","version":"1.0","stable":true}],"action_mappings":{"alpha:act":[]}}`
	_, err := ParseCapabilityRegistry(raw)
	if err == nil {
		t.Fatalf("expected error for empty action mapping")
	}
	te, ok := err.(*CapabilityRegistryError)
	if !ok {
		t.Fatalf("expected CapabilityRegistryError type; got %T", err)
	}
	if te.Kind != RegistryErrorEmptyActionMapping {
		t.Fatalf("expected kind RegistryErrorEmptyActionMapping; got %v", te.Kind)
	}
	if te.Action != "alpha:act" {
		t.Fatalf("expected Action 'alpha:act'; got %s", te.Action)
	}
	if te.CapabilityID != "" {
		t.Fatalf("expected empty CapabilityID for empty mapping error; got %s", te.CapabilityID)
	}
}
