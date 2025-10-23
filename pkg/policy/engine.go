package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Effect enumerates policy decision effect.
type Effect string

const (
	Allow Effect = "allow"
	Deny  Effect = "deny"
)

// Rule is a minimal condition/action match element.
type Rule struct {
	Actions   []string          `json:"actions"`
	Resources []string          `json:"resources"`
	Expr      string            `json:"expr,omitempty"` // optional expression (AND-only) over attributes/time
	Effect    Effect            `json:"effect"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// Policy groups rules under an ID and optional subject match (simple RBAC/ABAC hybrid).
type Policy struct {
	ID       string            `json:"id"`
	Subjects []string          `json:"subjects"` // direct subject identifiers or wildcard "*"
	Rules    []Rule            `json:"rules"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// Bundle is a set of policies with provenance hash chain fields.
type Bundle struct {
	ID       string    `json:"id"`
	Version  int       `json:"version"` // monotonically increasing revision number
	Policies []Policy  `json:"policies"`
	Created  time.Time `json:"created"`
	PrevHash string    `json:"prev_hash"`
	Hash     string    `json:"hash"`
}

// Decision captures evaluation outcome and provenance.
type EvalDecision struct {
	Allow         bool     `json:"allow"`
	Deny          bool     `json:"deny"`
	Matched       []string `json:"matched,omitempty"`
	DeniedBy      []string `json:"denied_by,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	BundleHash    string   `json:"bundle_hash,omitempty"`
	ChainHead     string   `json:"chain_head,omitempty"`     // current registry head hash
	PolicyChain   []string `json:"policy_chain,omitempty"`   // hashes from genesis to head
	PolicyVersion int      `json:"policy_version,omitempty"` // effective bundle version
}

// Request represents an authorization query to the engine.
type EvalRequest struct {
	Subject  string            `json:"subject"`
	Action   string            `json:"action"`
	Resource string            `json:"resource"`
	Attrs    map[string]string `json:"attrs,omitempty"` // subject / resource / environment flattened attributes
	Now      time.Time         `json:"now"`
}

// Registry maintains an ordered chain of bundles with integrity verification.
type Registry struct {
	bundles      []Bundle
	headOverride *Bundle // when rollback applied, points to historical bundle treated as active
}

func NewRegistry() *Registry { return &Registry{bundles: make([]Bundle, 0)} }

// AddBundle appends a bundle computing its hash and linking previous.
func (r *Registry) AddBundle(b Bundle) (Bundle, error) {
	if b.ID == "" {
		return Bundle{}, errors.New("bundle id required")
	}
	b.Created = time.Now().UTC()
	if len(r.bundles) > 0 {
		b.PrevHash = r.bundles[len(r.bundles)-1].Hash
	}
	// Assign monotonically increasing version if not explicitly set (>0 accepted)
	if b.Version == 0 {
		if len(r.bundles) == 0 {
			b.Version = 1
		} else {
			b.Version = r.bundles[len(r.bundles)-1].Version + 1
		}
	}
	// Reset any previous rollback override when appending new head (forward progression)
	r.headOverride = nil
	h, err := hashBundle(b)
	if err != nil {
		return Bundle{}, err
	}
	b.Hash = h
	r.bundles = append(r.bundles, b)
	return b, nil
}

// Head returns latest bundle (nil if empty).
func (r *Registry) Head() *Bundle {
	if r.headOverride != nil {
		return r.headOverride
	}
	if len(r.bundles) == 0 {
		return nil
	}
	return &r.bundles[len(r.bundles)-1]
}

// ChainHashes returns ordered list of bundle hashes.
func (r *Registry) ChainHashes() []string {
	if r.headOverride == nil {
		out := make([]string, len(r.bundles))
		for i, b := range r.bundles {
			out[i] = b.Hash
		}
		return out
	}
	// When rolled back, still return full chain (historical visibility)
	out := make([]string, len(r.bundles))
	for i, b := range r.bundles {
		out[i] = b.Hash
	}
	return out
}

// VerifyChain recalculates hashes & link correctness.
func (r *Registry) VerifyChain() error {
	for i, b := range r.bundles {
		h, err := hashBundle(b)
		if err != nil {
			return err
		}
		if h != b.Hash {
			return fmt.Errorf("bundle hash mismatch at index %d", i)
		}
		if i == 0 && b.PrevHash != "" {
			return fmt.Errorf("genesis bundle has prev hash")
		}
		if i > 0 && b.PrevHash != r.bundles[i-1].Hash {
			return fmt.Errorf("broken prev hash link at %d", i)
		}
	}
	return nil
}

// Rollback sets headOverride to the bundle with matching version. Does not mutate historical bundles.
// Returns error if version not found. Rollback is idempotent if same version requested.
func (r *Registry) Rollback(version int) error {
	for i := range r.bundles {
		if r.bundles[i].Version == version {
			r.headOverride = &r.bundles[i]
			return nil
		}
	}
	return fmt.Errorf("rollback version %d not found", version)
}

// ActiveVersion returns the version of current effective head (after rollback if any).
func (r *Registry) ActiveVersion() int {
	h := r.Head()
	if h == nil {
		return 0
	}
	return h.Version
}

// ChainWithVersions returns slice of (version, hash) for introspection APIs.
func (r *Registry) ChainWithVersions() []struct {
	Version int
	Hash    string
} {
	out := make([]struct {
		Version int
		Hash    string
	}, len(r.bundles))
	for i, b := range r.bundles {
		out[i] = struct {
			Version int
			Hash    string
		}{b.Version, b.Hash}
	}
	return out
}

// --- Diff Support ---
// PolicyDiff summarizes differences between two bundles (from -> to) at policy granularity.
// A policy is considered "changed" when its rule set or subjects differ (hash comparison of canonical JSON form).
// Added: policies present in 'to' but not in 'from'. Removed: present in 'from' but not in 'to'. Changed: IDs present in both with differing canonical representations.
type PolicyDiff struct {
	FromVersion int      `json:"from_version"`
	ToVersion   int      `json:"to_version"`
	Added       []Policy `json:"added"`
	Removed     []Policy `json:"removed"`
	Changed     []struct {
		ID   string `json:"id"`
		From Policy `json:"from"`
		To   Policy `json:"to"`
	} `json:"changed"`
	Unchanged   []Policy `json:"unchanged"`
	FromHash    string   `json:"from_hash"`
	ToHash      string   `json:"to_hash"`
	ChainHead   string   `json:"chain_head"`
	PolicyChain []string `json:"policy_chain"`
}

// canonicalPolicy serializes a policy deterministically for diff comparisons (without meta ordering guarantees beyond JSON marshal field order).
func canonicalPolicy(p Policy) string {
	// Sort rules deterministically based on Expr then Actions then Resources for stable diff hash.
	sorted := p
	sort.Slice(sorted.Rules, func(i, j int) bool {
		// primary key: Expr; fallback to concatenated actions/resources for stability
		if sorted.Rules[i].Expr != sorted.Rules[j].Expr {
			return sorted.Rules[i].Expr < sorted.Rules[j].Expr
		}
		aI := strings.Join(sorted.Rules[i].Actions, ":")
		aJ := strings.Join(sorted.Rules[j].Actions, ":")
		if aI != aJ {
			return aI < aJ
		}
		rI := strings.Join(sorted.Rules[i].Resources, ":")
		rJ := strings.Join(sorted.Rules[j].Resources, ":")
		return rI < rJ
	})
	// Sort subjects for stable comparison
	subj := make([]string, len(sorted.Subjects))
	copy(subj, sorted.Subjects)
	sort.Strings(subj)
	sorted.Subjects = subj
	enc, err := json.Marshal(sorted)
	if err != nil {
		// Should not happen (static types); return empty to avoid panics while surfacing anomaly via logging hook if any.
		return ""
	}
	return string(enc)
}

// Diff computes a PolicyDiff between two bundle versions. Returns error if versions not found or identical versions requested.
// When fromVersion == 0 it defaults to active version (after rollback). When toVersion == 0 it defaults to head version.
func (r *Registry) Diff(fromVersion, toVersion int) (PolicyDiff, error) {
	if len(r.bundles) == 0 {
		return PolicyDiff{}, errors.New("empty policy chain")
	}
	// Resolve defaults
	if fromVersion == 0 {
		fromVersion = r.ActiveVersion()
	}
	if toVersion == 0 {
		h := r.Head()
		if h != nil {
			toVersion = h.Version
		}
	}
	if fromVersion == toVersion {
		return PolicyDiff{}, errors.New("diff requires distinct versions")
	}
	var fromB, toB *Bundle
	for i := range r.bundles {
		if r.bundles[i].Version == fromVersion {
			fromB = &r.bundles[i]
		}
		if r.bundles[i].Version == toVersion {
			toB = &r.bundles[i]
		}
	}
	if fromB == nil || toB == nil {
		return PolicyDiff{}, fmt.Errorf("one or both versions not found (from=%d to=%d)", fromVersion, toVersion)
	}
	// Build maps for membership & canonical hashes
	fromMap := map[string]Policy{}
	toMap := map[string]Policy{}
	fromCanon := map[string]string{}
	toCanon := map[string]string{}
	for _, p := range fromB.Policies {
		fromMap[p.ID] = p
		fromCanon[p.ID] = canonicalPolicy(p)
	}
	for _, p := range toB.Policies {
		toMap[p.ID] = p
		toCanon[p.ID] = canonicalPolicy(p)
	}
	var added, removed, unchanged []Policy
	var changed []struct {
		ID   string `json:"id"`
		From Policy `json:"from"`
		To   Policy `json:"to"`
	}
	// Detect removed & changed / unchanged
	for id, fp := range fromMap {
		tp, ok := toMap[id]
		if !ok {
			removed = append(removed, fp)
			continue
		}
		if fromCanon[id] != toCanon[id] {
			changed = append(changed, struct {
				ID   string `json:"id"`
				From Policy `json:"from"`
				To   Policy `json:"to"`
			}{ID: id, From: fp, To: tp})
		} else {
			unchanged = append(unchanged, fp)
		}
	}
	// Detect added
	for id, tp := range toMap {
		if _, ok := fromMap[id]; !ok {
			added = append(added, tp)
		}
	}
	// Sort output slices deterministically by policy ID
	sort.Slice(added, func(i, j int) bool { return added[i].ID < added[j].ID })
	sort.Slice(removed, func(i, j int) bool { return removed[i].ID < removed[j].ID })
	sort.Slice(unchanged, func(i, j int) bool { return unchanged[i].ID < unchanged[j].ID })
	sort.Slice(changed, func(i, j int) bool { return changed[i].ID < changed[j].ID })
	head := r.Head()
	diff := PolicyDiff{FromVersion: fromVersion, ToVersion: toVersion, Added: added, Removed: removed, Changed: changed, Unchanged: unchanged, FromHash: fromB.Hash, ToHash: toB.Hash}
	if head != nil {
		diff.ChainHead = head.Hash
	}
	diff.PolicyChain = r.ChainHashes()
	return diff, nil
}

// FindByHash returns the bundle matching the given hash (linear scan acceptable for demo).
func (r *Registry) FindByHash(hash string) *Bundle {
	for i := range r.bundles {
		if r.bundles[i].Hash == hash {
			return &r.bundles[i]
		}
	}
	return nil
}

// ValidateBundle performs minimal schema validation (no empty policy IDs, rules, or effects).
func ValidateBundle(b Bundle) error {
	if b.ID == "" {
		return errors.New("bundle id required")
	}
	if len(b.Policies) == 0 {
		return errors.New("bundle must contain at least one policy")
	}
	for _, p := range b.Policies {
		if p.ID == "" {
			return fmt.Errorf("policy id required")
		}
		if len(p.Subjects) == 0 {
			return fmt.Errorf("policy %s must have at least one subject", p.ID)
		}
		if len(p.Rules) == 0 {
			return fmt.Errorf("policy %s must contain at least one rule", p.ID)
		}
		for ri, r := range p.Rules {
			if len(r.Actions) == 0 {
				return fmt.Errorf("policy %s rule[%d] missing actions", p.ID, ri)
			}
			if len(r.Resources) == 0 {
				return fmt.Errorf("policy %s rule[%d] missing resources", p.ID, ri)
			}
			if r.Effect != Allow && r.Effect != Deny {
				return fmt.Errorf("policy %s rule[%d] invalid effect", p.ID, ri)
			}
		}
	}
	return nil
}

// Engine evaluates requests against the head bundle (if any).
type ChainEngine struct {
	reg *Registry
}

func NewChainEngine(reg *Registry) *ChainEngine { return &ChainEngine{reg: reg} }

// Evaluate performs policy evaluation with deny-overrides semantics.
func (e *ChainEngine) Evaluate(ctx context.Context, req EvalRequest) (EvalDecision, error) {
	head := e.reg.Head()
	if head == nil {
		return EvalDecision{Allow: false, Reason: "no policy bundles"}, nil
	}
	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}
	dec := EvalDecision{BundleHash: head.Hash, ChainHead: head.Hash, PolicyChain: e.reg.ChainHashes(), PolicyVersion: head.Version}
	// Iterate policies
	for _, p := range head.Policies {
		if !subjectMatch(p.Subjects, req.Subject) {
			continue
		}
		for _, r := range p.Rules {
			if !stringSliceMatch(r.Actions, req.Action) {
				continue
			}
			if !stringSliceMatch(r.Resources, req.Resource) {
				continue
			}
			if r.Expr != "" {
				ok, err := evalExpr(r.Expr, req.Attrs, req.Now)
				if err != nil {
					return EvalDecision{}, err
				}
				if !ok {
					continue
				}
			}
			if r.Effect == Deny {
				dec.Deny = true
				dec.DeniedBy = append(dec.DeniedBy, p.ID)
			} else {
				dec.Matched = append(dec.Matched, p.ID)
			}
		}
	}
	if dec.Deny {
		dec.Allow = false
		dec.Reason = "denied by policy"
	} else if len(dec.Matched) > 0 {
		dec.Allow = true
		dec.Reason = "allowed"
	} else {
		dec.Allow = false
		dec.Reason = "no matching policy"
	}
	return dec, nil
}

// Helpers
func subjectMatch(subjects []string, sub string) bool {
	for _, s := range subjects {
		if s == "*" || strings.EqualFold(s, sub) {
			return true
		}
	}
	return false
}

func stringSliceMatch(list []string, value string) bool {
	for _, v := range list {
		if v == "*" || v == value {
			return true
		}
	}
	return false
}

// hashBundle produces a canonical hash of bundle excluding Hash field itself.
func hashBundle(b Bundle) (string, error) {
	// canonical ordering of policies and rules for deterministic hash
	sort.Slice(b.Policies, func(i, j int) bool { return b.Policies[i].ID < b.Policies[j].ID })
	for pi := range b.Policies {
		sort.Slice(b.Policies[pi].Rules, func(i, j int) bool { return b.Policies[pi].Rules[i].Expr < b.Policies[pi].Rules[j].Expr })
	}
	tmp := struct {
		ID       string    `json:"id"`
		Version  int       `json:"version"`
		Policies []Policy  `json:"policies"`
		Created  time.Time `json:"created"`
		PrevHash string    `json:"prev_hash"`
	}{b.ID, b.Version, b.Policies, b.Created, b.PrevHash}
	data, err := json.Marshal(tmp)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// --- Expression Evaluator (extended) ---
// Grammar (simple recursive descent with top-level splitting respecting parentheses):
//   expr        := orExpr
//   orExpr      := andExpr ( '||' andExpr )*
//   andExpr     := unaryExpr ( '&&' unaryExpr )*
//   unaryExpr   := '!'* primary
//   primary     := clause | '(' orExpr ')'
//   clause      := key '==' value
//                | key ('>'|'<'|'>='|'<=' ) number
//                | key 'in' [v1,v2,...]
//                | time_between("HH:MM","HH:MM")
// Extended operators: logical OR (||), AND (&&), unary NOT (!), parentheses for grouping.

func evalExpr(expr string, attrs map[string]string, now time.Time) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	} // empty treated as true (rule-level guard handles presence)
	return parseOr(expr, attrs, now)
}

func parseOr(s string, attrs map[string]string, now time.Time) (bool, error) {
	parts := splitTopWithParens(s, "||")
	var any bool
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := parseAnd(p, attrs, now)
		if err != nil {
			return false, err
		}
		if v {
			any = true
			break
		}
	}
	return any, nil
}

func parseAnd(s string, attrs map[string]string, now time.Time) (bool, error) {
	parts := splitTopWithParens(s, "&&")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := parseUnary(p, attrs, now)
		if err != nil {
			return false, err
		}
		if !v {
			return false, nil
		}
	}
	return true, nil
}

func parseUnary(s string, attrs map[string]string, now time.Time) (bool, error) {
	negCount := 0
	for strings.HasPrefix(strings.TrimSpace(s), "!") {
		negCount++
		s = strings.TrimSpace(s)[1:]
	}
	v, err := parsePrimary(strings.TrimSpace(s), attrs, now)
	if err != nil {
		return false, err
	}
	if negCount%2 == 1 {
		return !v, nil
	}
	return v, nil
}

func parsePrimary(s string, attrs map[string]string, now time.Time) (bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return true, nil
	}
	// Parenthesized group
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") && matchingParens(s) {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		return parseOr(inner, attrs, now)
	}
	return evalClause(s, attrs, now)
}

// splitTopWithParens splits s by delim at top level ignoring delimiters inside parentheses.
func splitTopWithParens(s, delim string) []string {
	var parts []string
	depth := 0
	last := 0
	i := 0
	for i < len(s) {
		if s[i] == '(' {
			depth++
			i++
			continue
		}
		if s[i] == ')' {
			if depth > 0 {
				depth--
			}
			i++
			continue
		}
		if depth == 0 && strings.HasPrefix(s[i:], delim) {
			parts = append(parts, s[last:i])
			i += len(delim)
			last = i
			continue
		}
		i++
	}
	parts = append(parts, s[last:])
	return parts
}

// matchingParens returns true if outer parentheses match and encompass entire string without imbalance.
func matchingParens(s string) bool {
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return false
	}
	depth := 0
	for i, ch := range s {
		if ch == '(' {
			depth++
		}
		if ch == ')' {
			depth--
			if depth == 0 && i != len(s)-1 {
				return false
			}
		}
		if depth < 0 {
			return false
		}
	}
	return depth == 0
}

func evalClause(clause string, attrs map[string]string, now time.Time) (bool, error) {
	// time_between("15:04","22:00")
	if strings.HasPrefix(clause, "time_between") {
		i := strings.Index(clause, "(")
		j := strings.LastIndex(clause, ")")
		if i < 0 || j < 0 {
			return false, fmt.Errorf("invalid time_between syntax")
		}
		inside := clause[i+1 : j]
		segs := splitCSV(inside)
		if len(segs) != 2 {
			return false, fmt.Errorf("time_between requires 2 params")
		}
		layout := "15:04"
		start, err := time.Parse(layout, trimQuotes(segs[0]))
		if err != nil {
			return false, err
		}
		end, err := time.Parse(layout, trimQuotes(segs[1]))
		if err != nil {
			return false, err
		}
		// Compare only clock time in UTC
		cur := now.UTC()
		curClock := time.Date(0, 1, 1, cur.Hour(), cur.Minute(), 0, 0, time.UTC)
		sClock := time.Date(0, 1, 1, start.Hour(), start.Minute(), 0, 0, time.UTC)
		eClock := time.Date(0, 1, 1, end.Hour(), end.Minute(), 0, 0, time.UTC)
		if sClock.Before(eClock) {
			return (curClock.Equal(sClock) || curClock.After(sClock)) && (curClock.Before(eClock) || curClock.Equal(eClock)), nil
		}
		// overnight window
		return !(curClock.After(eClock) && curClock.Before(sClock)), nil
	}
	// equality or in operator
	if strings.Contains(clause, " in ") {
		segs := strings.SplitN(clause, " in ", 2)
		key := strings.TrimSpace(segs[0])
		list := strings.TrimSpace(segs[1])
		if !strings.HasPrefix(list, "[") || !strings.HasSuffix(list, "]") {
			return false, fmt.Errorf("invalid in list syntax")
		}
		list = strings.Trim(list, "[]")
		opts := splitCSV(list)
		val := attrs[key]
		for _, o := range opts {
			if val == trimQuotes(o) {
				return true, nil
			}
		}
		return false, nil
	}
	if strings.Contains(clause, "==") {
		segs := strings.SplitN(clause, "==", 2)
		key := strings.TrimSpace(segs[0])
		want := trimQuotes(strings.TrimSpace(segs[1]))
		return attrs[key] == want, nil
	}
	// numeric comparisons
	for _, op := range []string{">=", "<=", ">", "<"} {
		if strings.Contains(clause, op) {
			segs := strings.SplitN(clause, op, 2)
			key := strings.TrimSpace(segs[0])
			rhs := strings.TrimSpace(segs[1])
			valStr := attrs[key]
			if valStr == "" {
				return false, nil
			}
			lhs, err1 := strconv.ParseFloat(valStr, 64)
			rhsVal, err2 := strconv.ParseFloat(trimQuotes(rhs), 64)
			if err1 != nil || err2 != nil {
				return false, fmt.Errorf("numeric parse error in clause: %s", clause)
			}
			switch op {
			case ">":
				return lhs > rhsVal, nil
			case "<":
				return lhs < rhsVal, nil
			case ">=":
				return lhs >= rhsVal, nil
			case "<=":
				return lhs <= rhsVal, nil
			}
		}
	}
	return false, fmt.Errorf("unsupported clause: %s", clause)
}

func splitCSV(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		t := strings.TrimSpace(r)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func trimQuotes(s string) string { return strings.Trim(s, "\"'") }
