// Package tsa provides a prototype HTTP client abstraction for a timestamp
// authority style anchoring service (demo only; no production security).
package tsa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// Receipt represents a timestamp authority style receipt for an anchored hash.
type Receipt struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Provider  string    `json:"provider"`
	Proof     []byte    `json:"proof,omitempty"`
	Version   int       `json:"version"`
}

// Client defines minimal TSA client operations.
type Client interface {
	Submit(ctx context.Context, hash string) (Receipt, error)
	Verify(ctx context.Context, r Receipt) error
}

// HTTPClient is a prototype TSA client using a simple HTTP JSON API.
// POST /tsa/anchor {"hash":"..."} => {hash,timestamp,provider,proof,version}
// POST /tsa/verify {receipt} => {"ok":true}
// This is a stub for integration; production code would include auth, TLS pinning, etc.
type HTTPClient struct {
	BaseURL    string
	HTTP       *http.Client
	ProviderID string
	Timeout    time.Duration
}

func NewHTTPClient(baseURL, providerID string) *HTTPClient {
	return &HTTPClient{BaseURL: baseURL, ProviderID: providerID, HTTP: &http.Client{Timeout: 10 * time.Second}, Timeout: 10 * time.Second}
}

func (c *HTTPClient) Submit(ctx context.Context, hash string) (Receipt, error) {
	if hash == "" {
		return Receipt{}, errors.New("empty hash")
	}
	reqBody := map[string]string{"hash": hash}
	b, mErr := json.Marshal(reqBody)
	if mErr != nil {
		return Receipt{}, mErr
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/tsa/anchor", bytesReader(b))
	if err != nil {
		return Receipt{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Receipt{}, err
	}
	defer func() {
		// Best effort close; capture error if needed.
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return Receipt{}, errors.New("unexpected status")
	}
	var r Receipt
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return Receipt{}, err
	}
	return r, nil
}

func (c *HTTPClient) Verify(ctx context.Context, r Receipt) error {
	if r.Hash == "" {
		return errors.New("invalid receipt")
	}
	b, mErr := json.Marshal(r)
	if mErr != nil {
		return mErr
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/tsa/verify", bytesReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected status")
	}
	var vr struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		return err
	}
	if !vr.OK {
		return errors.New("verification failed")
	}
	return nil
}

// bytesReader wraps byte slice without importing bytes everywhere.
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
