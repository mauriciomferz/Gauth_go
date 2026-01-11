package agentauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// StripePVPClient implements PowerVerificationPoint using Stripe Identity
type StripePVPClient struct {
	apiKey string
	client *http.Client
}

// NewStripePVPClient creates a new Stripe Identity client
func NewStripePVPClient(apiKey string) *StripePVPClient {
	return &StripePVPClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// StripeVerificationSession represents a subset of Stripe Identity Verification Session object
type StripeVerificationSession struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	// Add other fields as needed
}

// VerifyIdentityProof verifies identity using Stripe Identity API
func (c *StripePVPClient) VerifyIdentityProof(ctx context.Context, request *IdentityProofRequest) (*IdentityProofResult, error) {
	// In a real implementation, 'request.Proof' would likely contain the VerificationSession ID
	// or we would be creating a session here.
	// request.ProofData should contain the 'session_id'
	sessionIDVal, ok := request.ProofData["session_id"]
	if !ok {
		return nil, fmt.Errorf("invalid proof format: missing session_id in ProofData")
	}
	sessionID, ok := sessionIDVal.(string)
	if !ok {
		return nil, fmt.Errorf("invalid proof format: session_id must be a string")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.stripe.com/v1/identity/verification_sessions/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.apiKey, "")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe api request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stripe api returned non-200 status: %d", resp.StatusCode)
	}

	var session StripeVerificationSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	valid := session.Status == "verified"

	return &IdentityProofResult{
		Valid:      valid,
		SubjectID:  request.SubjectID,
		Identity:   fmt.Sprintf("stripe:%s", session.ID),
		TrustLevel: request.RequiredLevel, // Assuming level matches if verified
		VerifiedAt: time.Now(),
	}, nil
}
