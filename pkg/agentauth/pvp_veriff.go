package agentauth

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// VeriffPVPClient implements PowerVerificationPoint using Veriff
type VeriffPVPClient struct {
	apiKey    string
	apiSecret string
	client    *http.Client
}

// NewVeriffPVPClient creates a new Veriff client
func NewVeriffPVPClient(apiKey, apiSecret string) *VeriffPVPClient {
	return &VeriffPVPClient{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// VerifyIdentityProof verifies identity using Veriff API
func (c *VeriffPVPClient) VerifyIdentityProof(ctx context.Context, request *IdentityProofRequest) (*IdentityProofResult, error) {
	// Veriff typically uses a session decision webhook or a decision endpoint.
	// For this integration, we assume we check a session status.

	sessionIDVal, ok := request.ProofData["session_id"]
	if !ok {
		return nil, fmt.Errorf("invalid proof format: missing session_id in ProofData")
	}
	sessionID, ok := sessionIDVal.(string)
	if !ok {
		return nil, fmt.Errorf("invalid proof format: session_id must be a string")
	}

	_ = sessionID // Prevent unused variable error for now

	// Placeholder for actual API call
	// req, _ := http.NewRequest("GET", "https://stationapi.veriff.com/v1/sessions/"+sessionID+"/decision", nil)
	// req.Header.Set("X-AUTH-CLIENT", c.apiKey)
	// ... calculate signature ...

	// Proactively returning error until fully implemented to avoid misleading behavior
	return nil, fmt.Errorf("veriff verification not fully implemented - requires signature calculation")
}
