package web

// RB3 Discovery Endpoint implementation.
// Provides a cacheable snapshot of governance + cryptographic configuration for clients.
// Path: GET /api/v1/discovery
// Cache-Control: max-age=30
// ETag (weak) computed over stable canonical JSON excluding generated_at.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// registerRB3Discovery mounts the RB3 discovery endpoint. Invoked from NewBetaServer after other
// subsystems initialized so capability & rotation hashes are available.
func (s *BetaServer) registerRB3Discovery() {
	s.router.GET("/api/v1/discovery", func(c *gin.Context) {
		// --- Build token_algorithms slice mirroring well-known logic ---
		legacyAlg := algHS256
		algs := []string{legacyAlg}
		jwtEnabled := os.Getenv("GAUTH_USE_JWT_LIB") == "1"
		if os.Getenv("GAUTH_TOKEN_SIG_MODE") == sigModeEdDSA { // advertise EdDSA when enabled
			algs = append(algs, "EdDSA")
		}
		if jwtEnabled {
			jwtAlg := os.Getenv("GAUTH_JWT_ALG")
			if jwtAlg == "" {
				jwtAlg = algRS256
			}
			algs = append(algs, jwtAlg)
		}
		// Ensure deterministic ordering (except legacy first for backward semantics)
		if len(algs) > 1 {
			tail := append([]string{}, algs[1:]...)
			sort.Strings(tail)
			algs = append([]string{legacyAlg}, tail...)
		}
		// Digest domains static list (exposed previously in well-known)
		digestDomains := []string{"GAUTH_RFC0111_POA_V1", "GAUTH_RFC0111_POA_V2", "GAUTH_RFC0111_POA_V3|tax=1"}
		taxonomySupported := true // RB2 added unconditional taxonomy support; future toggle could gate this.
		activeDigestDomain := "GAUTH_RFC0111_POA_V1"
		poaVersionCurrent := 1
		if taxonomySupported { // prefer taxonomy domain for single-sig issuance baseline
			activeDigestDomain = "GAUTH_RFC0111_POA_V3|tax=1"
			poaVersionCurrent = 3
		}
		// Replay strict mode if durable WAL configured for token issuance replay store.
		replayStrict := false
		if s.replayStore != nil && s.replayStore.IsDurable() { // check via exported method
			replayStrict = true
		}
		// Rotation ledger head hash (optional)
		rotHead := ""
		if s.rotationLedger != nil {
			rotHead = s.rotationLedger.HeadHash()
		}
		// Capability registry hash
		capHash := s.GetCapabilityRegistryHash()
		// Core payload excluding generated_at for ETag computation.
		// Max delegation depth (RB12) dynamic env inspection each request; empty or invalid => omitted / 0.
		var maxDepthVal any
		if raw := os.Getenv("GAUTH_MAX_DELEGATION_DEPTH"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil && v > 0 {
				maxDepthVal = v
			}
		}
		core := map[string]any{
			"schema_version":       1,
			"digest_domains":       digestDomains,
			"active_digest_domain": activeDigestDomain,
			"token_algorithms":     algs,
			"replay_strict_mode":   replayStrict,
			"poa_version_current":  poaVersionCurrent,
			"capabilities_hash":    capHash,
			"rotation_tip_hash":    rotHead,
			"taxonomy_supported":   taxonomySupported,
			"max_delegation_depth": maxDepthVal,
			"revocation_signing_alg_values_supported": func() []string {
				if os.Getenv("GAUTH_TOKEN_SIG_MODE") == sigModeEdDSA {
					return []string{"EdDSA"}
				}
				return []string{}
			}(),
		}
		canonical, _ := json.Marshal(core)
		sum := sha256.Sum256(canonical)
		etag := fmt.Sprintf("W/\"%s\"", hex.EncodeToString(sum[:]))
		c.Header("Cache-Control", "max-age=30")
		c.Header("ETag", etag)
		if inm := c.GetHeader("If-None-Match"); inm != "" && inm == etag {
			c.Status(http.StatusNotModified)
			return
		}
		// Final payload with timestamp & etag surface (etag duplicated inside body for debugging convenience)
		core["generated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		core["etag"] = etag
		c.JSON(200, core)
	})
}
