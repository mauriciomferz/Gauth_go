package policy

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/mauriciomferz/Gauth_go/pkg/policy"
)

// policyChainPersist is the schema for persisted policy chains.
type policyChainPersist struct {
	Bundles  []policy.Bundle `json:"bundles"`
	Checksum string          `json:"checksum"`
}

// loadState loads the policy chain from disk into the handler's registry.
func (h *Handler) loadState(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}
	var pc policyChainPersist
	if err := json.Unmarshal(b, &pc); err != nil {
		return err
	}
	if pc.Checksum != "" {
		rawBundles, err := json.Marshal(pc.Bundles)
		if err != nil {
			return fmt.Errorf("checksum marshal: %w", err)
		}
		sum := sha256.Sum256(rawBundles)
		if fmt.Sprintf("%x", sum[:]) != pc.Checksum {
			return errors.New("policy persistence checksum mismatch")
		}
	}

	// Rebuild registry
	reg := policy.NewRegistry()
	for _, stored := range pc.Bundles {
		if _, err := reg.AddBundle(policy.Bundle{ID: stored.ID, Version: stored.Version, Policies: stored.Policies}); err != nil {
			return fmt.Errorf("reappend error: %w", err)
		}
	}
	// Verify continuity
	expected := 1
	for _, b := range reg.ChainWithVersions() {
		if b.Version != expected {
			return fmt.Errorf("continuity break expected=%d got=%d", expected, b.Version)
		}
		expected++
	}

	h.Registry = reg
	h.Engine = policy.NewChainEngine(h.Registry)
	return nil
}

// saveState persists the current registry state to disk.
func (h *Handler) saveState(path string) error {
	if h.Registry == nil {
		return errors.New("nil registry")
	}
	pc := policyChainPersist{Bundles: []policy.Bundle{}}
	for _, hash := range h.Registry.ChainHashes() {
		if b := h.Registry.FindByHash(hash); b != nil {
			pc.Bundles = append(pc.Bundles, *b)
		}
	}
	rawBundles, err := json.Marshal(pc.Bundles)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(rawBundles)
	pc.Checksum = fmt.Sprintf("%x", sum[:])

	enc, err := json.Marshal(pc)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, enc, 0o600); err != nil {
		return err
	}
	// Verify write by fsync? Skipping for brevity in non-critical path, but good practice.
	return os.Rename(tmp, path)
}
