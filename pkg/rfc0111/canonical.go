package rfc0111

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
//   weights (sorted keys when present), valid_from (RFC3339 UTC), valid_until (RFC3339 UTC), created_at (RFC3339 UTC)
// Excluded fields: Status (mutable), UpdatedAt (mutable), any future dynamic metadata.
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

const poaDigestDomainV1 = "GAUTH_RFC0111_POA_V1\n" // trailing newline as delimiter (legacy)
// V2 multi-sig domain (threshold + weights binding). Stable ordering of weights keys.
// Format: GAUTH_RFC0111_POA_V2|thr=<threshold>|w=<signer1>=<w1>,<signer2>=<w2>,...\n

// CanonicalPOADigest returns (digestHex, canonicalJSON, error) for the supplied POA.
// digestHex = SHA256( domain || canonicalJSON ) in lowercase hex.
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
		for k := range p.Weights { wKeys = append(wKeys, k) }
		sort.Strings(wKeys)
		buf.WriteByte(',')
		buf.WriteString("\"weights\":")
		buf.WriteByte('{')
		for i, k := range wKeys {
			if i > 0 { buf.WriteByte(',') }
			writeJSONStringRaw(&buf, k)
			buf.WriteByte(':')
			writeJSONStringRaw(&buf, fmt.Sprintf("%d", p.Weights[k]))
		}
		buf.WriteByte('}')
	}
	// times (writeJSONStringField will prepend comma automatically)
	writeJSONStringField(&buf, "valid_from", vf.Format(time.RFC3339), false)
	writeJSONStringField(&buf, "valid_until", vu.Format(time.RFC3339), false)
	writeJSONStringField(&buf, "created_at", ca.Format(time.RFC3339), false)
	buf.WriteByte('}')

	canonical := buf.Bytes()
	// Domain selection: V2 if threshold>1 (multi-signature context) else V1.
	domain := poaDigestDomainV1
	if p.Threshold > 1 && len(p.Signers) > 0 {
		weightParts := []string{}
		if len(p.Weights) > 0 {
			for k, v := range p.Weights {
				weightParts = append(weightParts, fmt.Sprintf("%s=%d", k, v))
			}
			sort.Strings(weightParts)
		}
		domain = fmt.Sprintf("GAUTH_RFC0111_POA_V2|thr=%d|w=%s\n", p.Threshold, strings.Join(weightParts, ","))
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
