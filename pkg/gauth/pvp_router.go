// Package gauth - Power Verification Point Router
// Routes identity proof requests to the appropriate PVP implementation based on proof method
package gauth

import (
	"context"
	"fmt"
	"sync"
)

// PVPRouter routes identity proof requests to appropriate PVP implementations
type PVPRouter struct {
	pvps      map[string]PowerVerificationPoint
	mu        sync.RWMutex
	defaultPVP PowerVerificationPoint
}

// NewPVPRouter creates a new PVP router
func NewPVPRouter(defaultPVP PowerVerificationPoint) *PVPRouter {
	return &PVPRouter{
		pvps:      make(map[string]PowerVerificationPoint),
		defaultPVP: defaultPVP,
	}
}

// RegisterPVP registers a PVP for specific proof methods
// Parameters:
//   - proofMethods: List of proof methods this PVP handles
//   - pvp: The PowerVerificationPoint implementation
func (r *PVPRouter) RegisterPVP(proofMethods []string, pvp PowerVerificationPoint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for _, method := range proofMethods {
		r.pvps[method] = pvp
	}
}

// VerifyIdentityProof implements PowerVerificationPoint interface
// Routes requests to the appropriate PVP based on proof method
func (r *PVPRouter) VerifyIdentityProof(ctx context.Context, request *IdentityProofRequest) (*IdentityProofResult, error) {
	if request == nil {
		return nil, fmt.Errorf("identity proof request is required")
	}
	
	if request.ProofMethod == "" {
		return nil, fmt.Errorf("proof method is required")
	}
	
	// Look up PVP for proof method
	r.mu.RLock()
	pvp, found := r.pvps[request.ProofMethod]
	r.mu.RUnlock()
	
	if !found {
		// Try default PVP if available
		if r.defaultPVP != nil {
			return r.defaultPVP.VerifyIdentityProof(ctx, request)
		}
		
		return nil, fmt.Errorf("no PVP registered for proof method: %s", request.ProofMethod)
	}
	
	return pvp.VerifyIdentityProof(ctx, request)
}

// GetPVP returns the PVP for a specific proof method
func (r *PVPRouter) GetPVP(proofMethod string) (PowerVerificationPoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	pvp, found := r.pvps[proofMethod]
	return pvp, found
}

// GetSupportedProofMethods returns all registered proof methods
func (r *PVPRouter) GetSupportedProofMethods() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	methods := make([]string, 0, len(r.pvps))
	for method := range r.pvps {
		methods = append(methods, method)
	}
	
	return methods
}
