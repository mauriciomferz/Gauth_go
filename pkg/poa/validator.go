package poa

// PoAValidator defines the interface for Power of Attorney validation.
type PoAValidator interface {
	Validate(poA *PowerOfAttorney) ([]ValidationWarning, error)
}

// PowerOfAttorney is a placeholder for the PoA structure.
type PowerOfAttorney struct {
	ID         string
	Parties    []string
	Scope      string
	Conditions map[string]interface{} // Conditional/special fields
	RawJSON    []byte                 // Raw JSON for CBOR/streaming
}

// ValidationWarning represents a warning or informational message from validation.
type ValidationWarning struct {
	Code    string
	Message string
}

// ValidatorRegistry allows modular registration of PoA validators.
type ValidatorRegistry struct {
	validators map[string]PoAValidator
}

func NewValidatorRegistry() *ValidatorRegistry {
	return &ValidatorRegistry{validators: make(map[string]PoAValidator)}
}

func (r *ValidatorRegistry) Register(name string, v PoAValidator) {
	r.validators[name] = v
}

func (r *ValidatorRegistry) Get(name string) PoAValidator {
	return r.validators[name]
}

// ConditionalInterpreter evaluates conditional/special fields in PoA.
type ConditionalInterpreter interface {
	Evaluate(conditions map[string]interface{}, context map[string]interface{}) (bool, error)
}

// AuditMetrics records validation and PoA lifecycle metrics.
type AuditMetrics interface {
	RecordValidation(poA *PowerOfAttorney, warnings []ValidationWarning, err error)
}

// CBORCodec encodes/decodes PoA to/from CBOR for compact/streaming support.
type CBORCodec interface {
	Encode(poA *PowerOfAttorney) ([]byte, error)
	Decode(data []byte) (*PowerOfAttorney, error)
}

// RawPOAExposer exposes RawPOA in verification results.
type RawPOAExposer interface {
	Expose(poA *PowerOfAttorney) ([]byte, error)
}

// Default implementations (stubs)
type DefaultConditionalInterpreter struct{}

func (i *DefaultConditionalInterpreter) Evaluate(conditions map[string]interface{}, context map[string]interface{}) (bool, error) {
	// TODO: Implement real conditional logic
	return true, nil
}

type DefaultAuditMetrics struct{}

func (m *DefaultAuditMetrics) RecordValidation(poA *PowerOfAttorney, warnings []ValidationWarning, err error) {
	// TODO: Implement real metrics recording
}

type DefaultCBORCodec struct{}

func (c *DefaultCBORCodec) Encode(poA *PowerOfAttorney) ([]byte, error) {
	// TODO: Implement CBOR encoding
	return poA.RawJSON, nil
}

func (c *DefaultCBORCodec) Decode(data []byte) (*PowerOfAttorney, error) {
	// TODO: Implement CBOR decoding
	return &PowerOfAttorney{RawJSON: data}, nil
}

type DefaultRawPOAExposer struct{}

func (e *DefaultRawPOAExposer) Expose(poA *PowerOfAttorney) ([]byte, error) {
	// TODO: Implement RawPOA exposure logic
	return poA.RawJSON, nil
}
