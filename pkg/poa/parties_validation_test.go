package poa

import (
	"testing"
)

func TestPartiesStructureValidation_RFC115_C1(t *testing.T) {
	// Test Principal Validation
	t.Run("Principal", func(t *testing.T) {
		p := Principal{Type: "Organization", Identity: "did:example:123"}
		// Basic check since Principal struct seems to lack explicit Validate() method in poa.go
		// but ValidatePoADefinition checks Identity.
		def := PoADefinition{
			Parties: Parties{
				Principal:        Principal{}, // Empty
				AuthorizedClient: AuthorizedClient{Identity: "client1"},
			},
		}
		if err := ValidatePoADefinition(def); err == nil {
			t.Error("Expected error for empty Principal Identity")
		}

		def.Parties.Principal = p
		// Should pass principal check (fails later on other things or passes if minimal valid)
		// We fix other parts to isolate principal check if possible, or just check error message.
	})

	// Test AuthorizedClient Validation
	t.Run("AuthorizedClient", func(t *testing.T) {
		validClient := AuthorizedClient{
			TypeEnum:        ClientTypeLLM,
			Identity:        "did:example:client",
			Version:         "1.0",
			StatusEnum:      OperationalStatusActive,
			ModelAttributes: &ModelAttributes{Architecture: "Transformer"},
		}
		if err := validClient.Validate(); err != nil {
			t.Errorf("Valid client failed validation: %v", err)
		}

		// Missing Identity
		invalid := validClient
		invalid.Identity = ""
		if err := invalid.Validate(); err == nil {
			t.Error("Expected error for missing Identity")
		}

		// Invalid Type
		invalid = validClient
		invalid.TypeEnum = "InvalidType"
		if err := invalid.Validate(); err == nil {
			t.Error("Expected error for invalid ClientType")
		}

		// Missing ModelAttributes for LLM
		invalid = validClient
		invalid.ModelAttributes = nil
		if err := invalid.Validate(); err == nil {
			t.Error("Expected error for missing ModelAttributes for LLM")
		}

		// Missing Team for AgenticAI
		agentic := validClient
		agentic.TypeEnum = ClientTypeAgenticAI
		agentic.ModelAttributes = nil // Not strictly required for AgenticAI in switch, but let's see code.
		// AgenticAI checks TeamComposition
		if err := agentic.Validate(); err == nil {
			t.Error("Expected error for missing TeamComposition for AgenticAI")
		}
	})

	// Test Representative Validation
	t.Run("Representative", func(t *testing.T) {
		validRep := Representative{
			Identity:          "did:example:rep",
			LegalRelationship: RelationshipOwner,
		}
		if err := validRep.Validate(); err != nil {
			t.Errorf("Valid representative failed validation: %v", err)
		}

		// Missing Identity
		invalid := validRep
		invalid.Identity = ""
		if err := invalid.Validate(); err == nil {
			t.Error("Expected error for missing Identity")
		}

		// Invalid Relationship
		invalid = validRep
		invalid.LegalRelationship = "InvalidRel"
		if err := invalid.Validate(); err == nil {
			t.Error("Expected error for invalid LegalRelationship")
		}

		// Broken Authorization Chain
		withChain := validRep
		withChain.AuthorizationChain = []AuthorizationLink{
			{FromParty: "A", ToParty: "B", Scope: "read"},
			{FromParty: "C", ToParty: "D", Scope: "read"}, // Broken link B != C
		}
		// Representative.Validate calls ValidateAuthorizationChain only?
		// Actually Representative.Validate logic for chain in poa.go:
		// loops and checks fields. It does NOT call ValidateAuthorizationChain function explicitly in the code I saw?
		// Let's check poa.go content again.
		// Representative.Validate checks fields of links.
		// ValidateAuthorizationChain is a separate function.

		// Let's test ValidateAuthorizationChain separately
		if err := ValidateAuthorizationChain(withChain.AuthorizationChain); err == nil {
			t.Error("Expected error for broken authorization chain")
		}
	})
}
