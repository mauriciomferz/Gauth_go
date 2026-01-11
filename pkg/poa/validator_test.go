package poa

import (
	"bytes"
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
	poa := &PowerOfAttorney{
		ID: "123", Parties: []string{"alice", "service-x"}, Scope: "txn",
		Conditions: map[string]interface{}{"limit": 5},
	}
	data, err := codec.Encode(poa)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty CBOR bytes")
	}
	// Deterministic encoding: second encode should match first.
	data2, err := codec.Encode(poa)
	if err != nil {
		t.Fatalf("second encode error: %v", err)
	}
	if !bytes.Equal(data, data2) {
		t.Fatalf("expected deterministic CBOR encoding identical across calls")
	}
	poa2, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if poa2.ID != poa.ID || poa2.Scope != poa.Scope {
		t.Fatalf("decoded core fields mismatch")
	}
	if len(poa2.Parties) != len(poa.Parties) {
		t.Fatalf("decoded parties length mismatch")
	}
	// numeric may decode as uint64/int64/float64 depending on codec settings; normalize via JSON number semantics
	limitVal := poa2.Conditions["limit"]
	switch v := limitVal.(type) {
	case int:
		if v != 5 {
			t.Fatalf("conditions limit mismatch int %d", v)
		}
	case int64:
		if v != 5 {
			t.Fatalf("conditions limit mismatch int64 %d", v)
		}
	case uint64:
		if v != 5 {
			t.Fatalf("conditions limit mismatch uint64 %d", v)
		}
	case float64:
		if v != 5 {
			t.Fatalf("conditions limit mismatch float64 %f", v)
		}
	default:
		t.Fatalf("unexpected numeric type %T", v)
	}
	if len(poa2.RawJSON) == 0 {
		t.Fatalf("expected RawJSON populated")
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

// FuzzCBORCodec exercises Encode/Decode for random PoA fields to uncover panics or state corruption.
func FuzzCBORCodec(f *testing.F) {
	seed := []struct{ id, scope string }{{"seed1", "scopeA"}, {"seed2", "scopeB"}}
	for _, s := range seed {
		f.Add(s.id, s.scope)
	}
	codec := &DefaultCBORCodec{}
	f.Fuzz(func(t *testing.T, id string, scope string) {
		po := &PowerOfAttorney{ID: id, Scope: scope, Parties: []string{"alice", "svc"}}
		data, err := codec.Encode(po)
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}
		if len(data) == 0 {
			t.Fatalf("empty encoding")
		}
		back, err := codec.Decode(data)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if back.ID != po.ID || back.Scope != po.Scope {
			t.Fatalf("roundtrip mismatch")
		}
	})
}
