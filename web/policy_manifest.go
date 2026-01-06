package web

// RB4 Signed Policy Manifest implementation.
// Exposes /api/v1/policy/manifest returning a signed, hash-addressed snapshot of capability governance.

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/internal/capability"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
)

// manifestCanonical represents the unsigned core used for hashing & signature (excludes generated_at & signature fields).
type manifestCanonical struct {
	SchemaVersion         int              `json:"schema_version"`
	Capabilities          []manifestCap    `json:"capabilities"`
	ActionMatrix          []manifestAction `json:"action_matrix"`
	RegistryHash          string           `json:"registry_hash"`
	RegistryPrevHash      string           `json:"registry_prev_hash,omitempty"`
	RegistryLastChangedAt string           `json:"registry_last_changed_at,omitempty"`
	CapabilityCount       int              `json:"capability_count"`
	ActionCount           int              `json:"action_count"`
}

type manifestCap struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	Stable          bool     `json:"stable"`
	DeprecatedAfter string   `json:"deprecated_after,omitempty"`
	SunsetAfter     string   `json:"sunset_after,omitempty"`
	Versions        []string `json:"versions,omitempty"`
}

type manifestAction struct {
	Action   string   `json:"action"`
	Required []string `json:"required"`
}

// buildPolicyManifest constructs the canonical manifest data from server state.
func (s *BetaServer) buildPolicyManifest() (manifestCanonical, []byte, string, error) {
	caps := capability.DefaultRegistry().List()
	sort.Slice(caps, func(i, j int) bool { return caps[i].ID < caps[j].ID })
	mcaps := make([]manifestCap, 0, len(caps))
	for _, c := range caps {
		entry := manifestCap{ID: c.ID, Version: c.Version, Stable: c.Stable}
		if c.DeprecatedAfter != "" {
			entry.DeprecatedAfter = c.DeprecatedAfter
		}
		if c.SunsetAfter != "" {
			entry.SunsetAfter = c.SunsetAfter
		}
		if len(c.Versions) > 0 {
			entry.Versions = c.Versions
		}
		mcaps = append(mcaps, entry)
	}
	// Action matrix from capabilitiesHandler
	actionMappings := s.capabilitiesHandler.GetActionMappings()
	actKeys := make([]string, 0, len(actionMappings))
	for k := range actionMappings {
		actKeys = append(actKeys, k)
	}
	sort.Strings(actKeys)
	actions := make([]manifestAction, 0, len(actKeys))
	for _, a := range actKeys {
		req := append([]string{}, actionMappings[a]...)
		sort.Strings(req)
		actions = append(actions, manifestAction{Action: a, Required: req})
	}
	// Build canonical struct
	canon := manifestCanonical{
		SchemaVersion:    1,
		Capabilities:     mcaps,
		ActionMatrix:     actions,
		RegistryHash:     s.GetCapabilityRegistryHash(),
		RegistryPrevHash: s.CapabilityPrevRegistryHash(),
		RegistryLastChangedAt: func() string {
			if !s.CapabilityRegistryChangeAt().IsZero() {
				return s.CapabilityRegistryChangeAt().Format(time.RFC3339)
			}
			return ""
		}(),
		CapabilityCount: len(mcaps),
		ActionCount:     len(actions),
	}
	raw, err := json.Marshal(canon)
	if err != nil {
		return manifestCanonical{}, nil, "", err
	}
	sum := sha256.Sum256(raw)
	manifestHash := fmt.Sprintf("sha256:%x", sum[:])
	return canon, raw, manifestHash, nil
}

// registerPolicyManifest mounts the /api/v1/policy/manifest endpoint.
func (s *BetaServer) registerPolicyManifest() {
	if s.router == nil {
		fmt.Fprintf(os.Stderr, "[policy_manifest] error: s.router is nil, cannot register routes\n")
		return
	}
	s.router.GET("/api/v1/policy/manifest", func(c *gin.Context) {
		canon, raw, hash, err := s.buildPolicyManifest()
		if err != nil {
			respondError(c, 500, "manifest_build_failed", "build_failed", "policy manifest build failed", "AAP001:policy_manifest", err.Error())
			return
		}
		// Signing prerequisites - use s.keyProvider if available, fallback to global
		kp := s.keyProvider

		if os.Getenv("AGENTAUTH_TOKEN_SIG_MODE") != sigModeEdDSA || kp == nil {
			respondError(c, 500, "signing_unavailable", "signing_unavailable", "active eddsa key unavailable", "AAP001:policy_manifest", nil)
			return
		}
		// RB6: use signer interface for agility
		signer, sErr := kp.ActiveSigner()
		if sErr != nil {
			respondError(c, 500, "signing_unavailable", "signing_unavailable", "active eddsa key unavailable", "AAP001:policy_manifest", sErr.Error())
			return
		}
		msg := append([]byte("AGENTAUTH_POLICY_MANIFEST:"), raw...)
		sigBytes, sErr := signer.Sign(msg)
		if sErr != nil {
			respondError(c, 500, "signing_unavailable", "signing_unavailable", "active eddsa key unavailable", "AAP001:policy_manifest", sErr.Error())
			return
		}
		sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)
		d := sha256.Sum256(raw)
		etag := fmt.Sprintf("W/\"%s\"", hex.EncodeToString(d[:]))
		c.Header("Cache-Control", "max-age=60")
		c.Header("ETag", etag)
		if inm := c.GetHeader("If-None-Match"); inm != "" && inm == etag {
			c.Status(http.StatusNotModified)
			return
		}
		payload := gin.H{
			"schema_version":     canon.SchemaVersion,
			"generated_at":       time.Now().UTC().Format(time.RFC3339Nano),
			"capabilities":       canon.Capabilities,
			"action_matrix":      canon.ActionMatrix,
			"registry_hash":      canon.RegistryHash,
			"registry_prev_hash": canon.RegistryPrevHash,
			"registry_last_changed_at": func() any {
				if canon.RegistryLastChangedAt != "" {
					return canon.RegistryLastChangedAt
				}
				return nil
			}(),
			"manifest_hash":    hash,
			"capability_count": canon.CapabilityCount,
			"action_count":     canon.ActionCount,
			"signature":        sigB64,
			"sig_kid": func() string {
				if rk, ok := signer.(interface{ KeyID() string }); ok {
					return rk.KeyID()
				}
				return ""
			}(),
			"sig_mode": sigModeEdDSA,
			"etag":     etag,
		}
		c.JSON(200, payload)
		if m, ok := s.metrics.(interface{ IncPolicyManifestEmitted() }); ok {
			m.IncPolicyManifestEmitted()
		}
	})
}

// Metrics interface extension (optional counters) - implement no-op if underlying metrics doesn't support.
var _ = metrics.Metrics(nil)
