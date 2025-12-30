package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// FilePolicySource loads policies from a JSON file.
type FilePolicySource struct {
	path string
	mu   sync.Mutex
}

// NewFilePolicySource creates a new source reading from path.
func NewFilePolicySource(path string) *FilePolicySource {
	return &FilePolicySource{path: path}
}

func (s *FilePolicySource) Fetch(ctx context.Context) ([]authz.Policy, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty if file not exists yet? Or error?
			// Let's error to be safe.
			return nil, "", err
		}
		return nil, "", err
	}

	// Compute hash as version
	h := sha256.New()
	h.Write(data)
	version := hex.EncodeToString(h.Sum(nil))

	var policies []authz.Policy
	if err := json.Unmarshal(data, &policies); err != nil {
		return nil, "", fmt.Errorf("invalid policy JSON: %w", err)
	}

	return policies, version, nil
}

// UpdateFile is a helper for testing to atomic writes
func (s *FilePolicySource) UpdateFile(policies []authz.Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(policies)
	return os.WriteFile(s.path, b, 0o644)
}
