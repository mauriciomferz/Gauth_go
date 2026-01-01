package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth/external"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// PolicyClientConfig holds configuration for the Policy API client.
type PolicyClientConfig struct {
	BaseURL        string
	Timeout        time.Duration
	CircuitBreaker *external.CircuitBreaker
}

// PolicyClient interacts with the AgentAuth Policy API.
type PolicyClient struct {
	config     *PolicyClientConfig
	httpClient *http.Client
}

// NewPolicyClient creates a new policy client.
func NewPolicyClient(config *PolicyClientConfig) *PolicyClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.CircuitBreaker == nil {
		config.CircuitBreaker = external.NewCircuitBreaker(5, config.Timeout, 60*time.Second)
	}
	return &PolicyClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// ProvenanceResponse models the API response from /api/v1/policy/provenance.
type ProvenanceResponse struct {
	Success            bool                       `json:"success"`
	HeadHash           string                     `json:"head_hash"`
	Chain              []string                   `json:"chain"`
	Verified           bool                       `json:"verified"`
	VerificationError  string                     `json:"verification_error,omitempty"`
	QueriedHash        string                     `json:"queried_hash,omitempty"`
	Length             int                        `json:"length"`
	RevocationSnapshot *delegation.SignedTreeHead `json:"revocation_snapshot,omitempty"`
}

// GetProvenance retrieves the current policy chain status and revocation snapshot.
// Optionally filters by specific hash.
func (c *PolicyClient) GetProvenance(ctx context.Context, hash string) (*ProvenanceResponse, error) {
	url := fmt.Sprintf("%s/api/v1/policy/provenance", c.config.BaseURL)
	if hash != "" {
		url += fmt.Sprintf("?hash=%s", hash)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	var resp *http.Response
	err = c.config.CircuitBreaker.Call(func() error {
		var rErr error
		resp, rErr = c.httpClient.Do(req)
		return rErr
	})
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result ProvenanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
