package poa

import (
	"testing"
	"time"
)

func TestValidatePoADefinition_Success(t *testing.T) {
	def := PoADefinition{
		Parties: Parties{
			Principal:        Principal{Identity: "org1", Type: PrincipalTypeOrganization},
			AuthorizedClient: AuthorizedClient{Identity: "agent1", Type: ClientTypeLLM},
		},
		Requirements: Requirements{ValidityPeriod: ValidityPeriod{StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}},
	}
	if err := ValidatePoADefinition(def); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestValidatePoADefinition_FailPrincipal(t *testing.T) {
	def := PoADefinition{
		Parties: Parties{
			Principal:        Principal{Identity: "", Type: PrincipalTypeOrganization},
			AuthorizedClient: AuthorizedClient{Identity: "agent1", Type: ClientTypeLLM},
		},
		Requirements: Requirements{ValidityPeriod: ValidityPeriod{StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}},
	}
	if err := ValidatePoADefinition(def); err == nil {
		t.Fatalf("expected error for missing principal identity")
	}
}

func TestValidatePoADefinition_FailAuthorizedClient(t *testing.T) {
	def := PoADefinition{
		Parties: Parties{
			Principal:        Principal{Identity: "org1", Type: PrincipalTypeOrganization},
			AuthorizedClient: AuthorizedClient{Identity: "", Type: ClientTypeLLM},
		},
		Requirements: Requirements{ValidityPeriod: ValidityPeriod{StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}},
	}
	if err := ValidatePoADefinition(def); err == nil {
		t.Fatalf("expected error for missing authorized client identity")
	}
}

func TestValidatePoADefinition_FailValidity(t *testing.T) {
	def := PoADefinition{
		Parties: Parties{
			Principal:        Principal{Identity: "org1", Type: PrincipalTypeOrganization},
			AuthorizedClient: AuthorizedClient{Identity: "agent1", Type: ClientTypeLLM},
		},
		Requirements: Requirements{ValidityPeriod: ValidityPeriod{StartTime: time.Now().Add(time.Hour), EndTime: time.Now()}},
	}
	if err := ValidatePoADefinition(def); err == nil {
		t.Fatalf("expected error for inverted validity period")
	}
}
