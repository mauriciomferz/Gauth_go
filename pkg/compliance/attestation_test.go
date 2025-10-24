package compliance

import (
	"testing"
)

func TestAttestationPipeline_IngestAndVerify(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	pipeline := NewAttestationPipeline()

	a := Attestation{
		ID:           "attest-1",
		FlowID:       "flow-1",
		Jurisdiction: JurisdictionUS,
		Entity:       EntityTypeCorporation,
		Proof:        []byte("dummy-proof"),
		Timestamp:    1690000000,
		Verified:     false,
	}

	pipeline.IngestAttestation(a)
	if len(pipeline.ListAttestations()) != 1 {
		t.Fatalf("expected 1 attestation, got %d", len(pipeline.ListAttestations()))
	}

	verified := pipeline.VerifyAttestation(a.ID, validator)
	if !verified {
		t.Fatalf("attestation should verify for valid jurisdiction")
	}

	att := pipeline.attestations[a.ID]
	if !att.Verified {
		t.Fatalf("attestation.Verified should be true after verification")
	}
}

func TestAttestationPipeline_VerifyInvalidJurisdiction(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	pipeline := NewAttestationPipeline()

	a := Attestation{
		ID:           "attest-2",
		FlowID:       "flow-2",
		Jurisdiction: Jurisdiction("INVALID"),
		Entity:       EntityTypeCorporation,
		Proof:        []byte("dummy-proof"),
		Timestamp:    1690000001,
		Verified:     false,
	}

	pipeline.IngestAttestation(a)
	verified := pipeline.VerifyAttestation(a.ID, validator)
	if verified {
		t.Fatalf("attestation should not verify for invalid jurisdiction")
	}
}
