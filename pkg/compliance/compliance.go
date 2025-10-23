package compliance

import (
	"context"
	"errors"
	"sync"
)

// DataClass enumerates categories for data classification.
type DataClass string

const (
	DataClassPersonal    DataClass = "personal"
	DataClassOperational DataClass = "operational"
	DataClassCrypto      DataClass = "cryptographic"
)

var (
	ErrFlowNotFound = errors.New("flow not found")
)

// Flow models a data flow between a source and destination component.
type Flow struct {
	ID          string
	Source      string
	Destination string
	DataTypes   []DataClass
	Purpose     string
	Retention   string // human-readable policy reference
	// Legal compliance fields
	Jurisdiction     Jurisdiction  `json:"jurisdiction,omitempty"`
	RequiredApproval ApprovalLevel `json:"required_approval,omitempty"`
	LegallyApproved  bool          `json:"legally_approved,omitempty"`
}

// Registry stores declared data flows (in-memory stub).
type Registry struct {
	mu             sync.RWMutex
	flows          map[string]Flow
	legalValidator *LegalFrameworkValidator
}

func NewRegistry() *Registry { 
	return &Registry{
		flows:          map[string]Flow{},
		legalValidator: NewLegalFrameworkValidator(),
	}
}

func (r *Registry) Register(f Flow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flows[f.ID] = f
}

// RegisterWithLegalValidation registers a flow after legal compliance validation.
func (r *Registry) RegisterWithLegalValidation(ctx context.Context, f Flow, action string) error {
	if f.Jurisdiction != "" {
		if err := r.legalValidator.ValidateJurisdiction(ctx, f.Jurisdiction, action); err != nil {
			return err
		}
		f.LegallyApproved = true
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flows[f.ID] = f
	return nil
}

func (r *Registry) List() []Flow {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Flow, 0, len(r.flows))
	for _, f := range r.flows {
		out = append(out, f)
	}
	return out
}

// GetLegalValidator returns the legal framework validator
func (r *Registry) GetLegalValidator() *LegalFrameworkValidator {
	return r.legalValidator
}

// ValidateFlowCompliance validates a flow against legal requirements
func (r *Registry) ValidateFlowCompliance(ctx context.Context, flowID string, jurisdiction Jurisdiction) error {
	r.mu.RLock()
	flow, exists := r.flows[flowID]
	r.mu.RUnlock()
	
	if !exists {
		return ErrFlowNotFound
	}
	
	// Determine action based on data types
	action := "data_flow"
	for _, dataType := range flow.DataTypes {
		if dataType == DataClassPersonal {
			action = "personal_data_flow"
			break
		}
		if dataType == DataClassCrypto {
			action = "crypto_data_flow"
			break
		}
	}
	
	return r.legalValidator.ValidateJurisdiction(ctx, jurisdiction, action)
}

// RetentionPolicy placeholder for future structured enforcement.
type RetentionPolicy struct {
	Name        string
	Description string
	TTLDays     int
}
