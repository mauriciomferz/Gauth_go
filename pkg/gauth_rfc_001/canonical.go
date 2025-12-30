package gauth_rfc_001

// Canonical POA digest implementation.
//
// Rationale:
// A delegation (PowerOfAttorney) needs a stable, minimal, mutation‑resistant
// representation for digital signatures. Fields that are mutable post‑issuance
// (Status, UpdatedAt) MUST be excluded to avoid invalidating the signature when
// operational state changes (revocation, expiry synchronization). Immutable or
// issuance‑defining fields are included. Ordering and formatting are strictly
// defined to guarantee determinism across platforms and Go versions.
//
// Included fields (in order):
//   id, version, grantor, grantee, scope (sorted), restrictions (sorted keys, always present),
//   weights (sorted keys when present), taxonomy (agent_type, sector, action_class when Version>=3 and non-empty),
//   valid_from (RFC3339 UTC), valid_until (RFC3339 UTC), created_at (RFC3339 UTC)
// Excluded fields: Status (mutable), UpdatedAt (mutable), jurisdiction, witnesses, attestations, revocation fields (mutable legal/evidentiary metadata), any future dynamic metadata.
// Domain separation: a constant prefix prevents cross‑protocol hash reuse.
// Weighted / threshold multi-signature mode enables a V2 domain which incorporates
// threshold and sorted weight mapping into the domain prefix to guarantee digest differentiation when
// aggregation semantics change (mitigating replay/confusion between single and aggregated signature contexts).

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

const poaDigestDomainV1 = "GAUTH_AAP001_POA_V1\n" // trailing newline as delimiter (legacy)
// V3 domain adds taxonomy sentinel to prevent cross-version digest collision when new fields introduced.
// Format extension: GAUTH_AAP001_POA_V3|tax=1 (weights/threshold logic still uses V2 multi-sig domain when threshold>1).
// V2 multi-sig domain (threshold + weights binding). Stable ordering of weights keys.
// Format: GAUTH_AAP001_POA_V2|thr=<threshold>|w=<signer1>=<w1>,<signer2>=<w2>,...\n

// CanonicalPOADigest returns (digestHex, canonicalJSON, error) for the supplied POA.
// digestHex = SHA256( domain || canonicalJSON ) in lowercase hex.
//
//nolint:gocyclo // Canonical digest generation with field normalization
func CanonicalPOADigest(p *PowerOfAttorney) (string, []byte, error) {
	if p == nil {
		return "", nil, fmt.Errorf("nil PowerOfAttorney")
	}

	// Defensive copy & normalize times to UTC (RFC3339 formatting later)
	vf := p.ValidFrom.UTC()
	vu := p.ValidUntil.UTC()
	ca := p.CreatedAt.UTC()

	// Scope: copy & sort for determinism
	scope := make([]string, len(p.Scope))
	copy(scope, p.Scope)
	sort.Strings(scope)

	// Restrictions: collect keys & sort; always include object (may be empty)
	rKeys := make([]string, 0, len(p.Restrictions))
	for k := range p.Restrictions {
		rKeys = append(rKeys, k)
	}
	sort.Strings(rKeys)

	// Build canonical JSON manually for strict ordering & minimal encoding (no spaces)
	var buf bytes.Buffer
	buf.WriteByte('{')
	// id / version / grantor / grantee
	writeJSONStringField(&buf, "id", p.ID, true)
	writeJSONStringField(&buf, "version", fmt.Sprintf("%d", p.Version), false)
	writeJSONStringField(&buf, "grantor", p.Grantor, false)
	writeJSONStringField(&buf, "grantee", p.Grantee, false)
	// scope array (prepend comma because previous field used helper which does not leave a trailing comma)
	buf.WriteByte(',')
	buf.WriteString("\"scope\":")
	buf.WriteByte('[')
	for i, s := range scope {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeJSONStringRaw(&buf, s)
	}
	buf.WriteByte(']')
	buf.WriteByte(',')
	// restrictions object (always present)
	buf.WriteString("\"restrictions\":")
	buf.WriteByte('{')
	for i, k := range rKeys {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeJSONStringRaw(&buf, k)
		buf.WriteByte(':')
		writeJSONStringRaw(&buf, p.Restrictions[k])
	}
	buf.WriteByte('}')
	// weights object (optional; deterministic ordering)
	if len(p.Weights) > 0 {
		wKeys := make([]string, 0, len(p.Weights))
		for k := range p.Weights {
			wKeys = append(wKeys, k)
		}
		sort.Strings(wKeys)
		buf.WriteByte(',')
		buf.WriteString("\"weights\":")
		buf.WriteByte('{')
		for i, k := range wKeys {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJSONStringRaw(&buf, k)
			buf.WriteByte(':')
			writeJSONStringRaw(&buf, fmt.Sprintf("%d", p.Weights[k]))
		}
		buf.WriteByte('}')
	}
	// taxonomy fields (Version >=3) included only when non-empty to avoid inflating legacy POAs.
	if p.Version >= 3 {
		includedAny := false
		// agent_type, sector, action_class in fixed order
		if p.AgentType != "" || p.Sector != "" || p.ActionClass != "" {
			buf.WriteByte(',')
			buf.WriteString("\"taxonomy\":")
			buf.WriteByte('{')
			// We emit keys only if value non-empty for minimal encoding; order fixed.
			firstField := true
			if p.AgentType != "" {
				writeJSONStringRaw(&buf, "agent_type")
				buf.WriteByte(':')
				writeJSONStringRaw(&buf, p.AgentType)
				firstField = false
				includedAny = true
			}
			if p.Sector != "" {
				if !firstField {
					buf.WriteByte(',')
				}
				writeJSONStringRaw(&buf, "sector")
				buf.WriteByte(':')
				writeJSONStringRaw(&buf, p.Sector)
				firstField = false
				includedAny = true
			}
			if p.ActionClass != "" {
				if !firstField {
					buf.WriteByte(',')
				}
				writeJSONStringRaw(&buf, "action_class")
				buf.WriteByte(':')
				writeJSONStringRaw(&buf, p.ActionClass)
				includedAny = true
			}
			buf.WriteByte('}')
		}
		_ = includedAny // future metrics hook if needed
	}
	// Hierarchical fields (Version >=4): parent_poa_id, parent_digest (if present), depth.
	if p.Version >= 4 {
		// Emit object only if any hierarchical linkage present OR always? Decide: always emit for structural binding even for root (depth=0).
		buf.WriteByte(',')
		buf.WriteString("\"hierarchy\":")
		buf.WriteByte('{')
		// parent_poa_id (empty for root): include value to guarantee root canonical differs from pre-v4 digests.
		writeJSONStringRaw(&buf, "parent_poa_id")
		buf.WriteByte(':')
		writeJSONStringRaw(&buf, p.ParentPOAID)
		buf.WriteByte(',')
		writeJSONStringRaw(&buf, "parent_digest")
		buf.WriteByte(':')
		writeJSONStringRaw(&buf, p.ParentDigest)
		buf.WriteByte(',')
		writeJSONStringRaw(&buf, "depth")
		buf.WriteByte(':')
		writeJSONStringRaw(&buf, fmt.Sprintf("%d", p.Depth))
		buf.WriteByte('}')
	}
	// times (writeJSONStringField will prepend comma automatically)
	writeJSONStringField(&buf, "valid_from", vf.Format(time.RFC3339), false)
	writeJSONStringField(&buf, "valid_until", vu.Format(time.RFC3339), false)
	writeJSONStringField(&buf, "created_at", ca.Format(time.RFC3339), false)
	buf.WriteByte('}')

	canonical := buf.Bytes()
	// Domain selection hierarchy:
	//   V4 hierarchical sentinel when Version>=4 and not multi-sig (single-sig taxonomy/hierarchy path)
	//   V2 multi-sig domain when threshold>1 (overrides hierarchical/taxonomy to retain stable multi-sig semantics)
	//   V3 taxonomy sentinel when Version>=3 (non hierarchical/non multi-sig)
	//   V1 legacy otherwise.
	domain := poaDigestDomainV1
	if p.Version >= 4 && !(p.Threshold > 1 && len(p.Signers) > 0) {
		// Hierarchical+taxonomy (if taxonomy present). Sentinel indicates presence of hierarchy fields in canonical JSON.
		domain = "GAUTH_AAP001_POA_V4|hier=1\n"
	} else if p.Version >= 3 && !(p.Threshold > 1 && len(p.Signers) > 0) {
		// Taxonomy-only upgrade path (non multi-sig & non hierarchical)
		domain = "GAUTH_AAP001_POA_V3|tax=1\n"
	}
	if p.Threshold > 1 && len(p.Signers) > 0 { // multi-sig overrides other domains
		weightParts := []string{}
		if len(p.Weights) > 0 {
			for k, v := range p.Weights {
				weightParts = append(weightParts, fmt.Sprintf("%s=%d", k, v))
			}
			sort.Strings(weightParts)
		}
		domain = fmt.Sprintf("GAUTH_AAP001_POA_V2|thr=%d|w=%s\n", p.Threshold, strings.Join(weightParts, ","))
	}
	h := sha256.Sum256(append([]byte(domain), canonical...))
	return hex.EncodeToString(h[:]), canonical, nil
}

// writeJSONStringField writes a string field with given name/value; if first==false it prefixes with a comma.
func writeJSONStringField(buf *bytes.Buffer, name, value string, first bool) {
	if !first {
		buf.WriteByte(',')
	}
	writeJSONStringRaw(buf, name)
	buf.WriteByte(':')
	writeJSONStringRaw(buf, value)
}

// writeJSONStringRaw writes a JSON string (without surrounding field name) escaping quotes & backslashes minimally.
func writeJSONStringRaw(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\':
			buf.WriteByte('\\')
			buf.WriteByte(c)
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		default:
			if c < 0x20 {
				// Control chars -> \u00XX
				buf.WriteString(fmt.Sprintf("\\u%04x", c))
			} else {
				buf.WriteByte(c)
			}
		}
	}
	buf.WriteByte('"')
}
