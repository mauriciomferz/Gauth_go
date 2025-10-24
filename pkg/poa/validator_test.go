package poa

import (
	"testing"
)

func TestValidatorRegistry_RegisterAndGet(t *testing.T) {
	reg := NewValidatorRegistry()
	stub := &StubValidator{}
	reg.Register("stub", stub)
	v := reg.Get("stub")
	if v == nil {
		t.Fatalf("expected validator, got nil")
	}
}

type StubValidator struct{}

func (s *StubValidator) Validate(poA *PowerOfAttorney) ([]ValidationWarning, error) {
	return []ValidationWarning{{Code: "stub", Message: "ok"}}, nil
}

func TestConditionalInterpreter_Evaluate(t *testing.T) {
	ci := &DefaultConditionalInterpreter{}
	ok, err := ci.Evaluate(map[string]interface{}{"cond": true}, map[string]interface{}{"ctx": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true, got false")
	}
}

func TestCBORCodec_EncodeDecode(t *testing.T) {
	codec := &DefaultCBORCodec{}
	poa := &PowerOfAttorney{ID: "123", RawJSON: []byte(`{"id":"123"}`)}
	data, err := codec.Encode(poa)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	poa2, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(poa2.RawJSON) != string(poa.RawJSON) {
		t.Fatalf("expected %s got %s", poa.RawJSON, poa2.RawJSON)
	}
}

func TestRawPOAExposer_Expose(t *testing.T) {
	exposer := &DefaultRawPOAExposer{}
	poa := &PowerOfAttorney{RawJSON: []byte(`{"id":"123"}`)}
	out, err := exposer.Expose(poa)
	if err != nil {
		t.Fatalf("expose error: %v", err)
	}
	if string(out) != string(poa.RawJSON) {
		t.Fatalf("expected %s got %s", poa.RawJSON, out)
	}
}

func TestAuditMetrics_RecordValidation(t *testing.T) {
	metrics := &DefaultAuditMetrics{}
	poa := &PowerOfAttorney{ID: "123"}
	metrics.RecordValidation(poa, []ValidationWarning{{Code: "stub", Message: "ok"}}, nil)
	// No error expected, stub implementation
}
