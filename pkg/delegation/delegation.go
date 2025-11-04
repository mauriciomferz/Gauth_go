package delegation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc"
)

// Delegation status constants
const (
	StatusActive           = "active"
	StatusSuspended        = "suspended"
	StatusTerminated       = "terminated"
	StatusPending          = "pending"
	StatusPartiallyRevoked = "partially_revoked" // scope reduced without full termination
)

// Delegation represents a POA/delegation grant from Subject to Delegate over Scope until ExpiresAt.
// Chain fields (PrevHash, Hash) allow sequencing and tamper detection similar to policy bundles.
type Delegation struct {
	ID        string            `json:"id"`
	Subject   string            `json:"subject"`  // original rights holder
	Delegate  string            `json:"delegate"` // entity receiving delegated rights
	Scope     map[string]string `json:"scope"`    // key-value scope constraints (resource, action, etc.)
	IssuedAt  time.Time         `json:"issued_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	PrevHash  string            `json:"prev_hash"`
	Hash      string            `json:"hash"`
	Status    string            `json:"status,omitempty"` // active, suspended, terminated, pending
}

// Revocation represents a future ability to invalidate a Delegation before expiry.
// Placeholder only; not consulted by Chain.VerifyChain yet.
type Revocation struct {
	DelegationHash string    `json:"delegation_hash"`
	Reason         string    `json:"reason"`
	RevokedAt      time.Time `json:"revoked_at"`
}

// TODO: Implement RevocationChain with hash linkage + Verify, and integrate into authorization path.

// Chain maintains ordered delegations.
type Chain struct{ items []Delegation }

// currentMaxDelegationDepth parses GAUTH_MAX_DELEGATION_DEPTH each call to allow dynamic test changes.
// Invalid or empty values disable enforcement (return 0).
func currentMaxDelegationDepth() int64 {
	raw := os.Getenv("GAUTH_MAX_DELEGATION_DEPTH")
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func NewChain() *Chain { return &Chain{items: make([]Delegation, 0)} }

// Append adds a new delegation computing its hash and linking previous.
func (c *Chain) Append(d Delegation) (Delegation, error) {
	if d.ID == "" {
		return Delegation{}, errors.New("delegation id required")
	}
	if d.Subject == "" || d.Delegate == "" {
		return Delegation{}, errors.New("subject and delegate required")
	}
	if d.ExpiresAt.Before(time.Now().UTC()) {
		return Delegation{}, errors.New("cannot append expired delegation")
	}
	if d.Status == "" {
		d.Status = StatusActive
	}
	if !validDelegationStatus(d.Status) {
		return Delegation{}, errors.New("invalid status")
	}
	d.IssuedAt = time.Now().UTC()
	// Enforce max depth (chain length) if configured (>0)
	if md := currentMaxDelegationDepth(); md > 0 {
		newDepth := int64(len(c.items) + 1) // depth defined as number of chain entries after append
		if newDepth > md {
			return Delegation{}, rfc.New(rfc.ErrDelegationDepthExceeded, fmt.Sprintf("delegation depth %d exceeds max %d", newDepth, md))
		}
	}
	if len(c.items) > 0 {
		d.PrevHash = c.items[len(c.items)-1].Hash
	}
	h, err := hashDelegation(d)
	if err != nil {
		return Delegation{}, err
	}
	d.Hash = h
	c.items = append(c.items, d)
	return d, nil
}

// AppendWithMetrics is an optional helper allowing instrumentation of depth exceeded events.
func (c *Chain) AppendWithMetrics(d Delegation, m metrics.Metrics) (Delegation, error) {
	if d.ID == "" {
		return Delegation{}, errors.New("delegation id required")
	}
	d.IssuedAt = time.Now().UTC()
	if md := currentMaxDelegationDepth(); md > 0 {
		newDepth := int64(len(c.items) + 1)
		if newDepth > md {
			if m != nil {
				m.IncDelegationDepthExceeded()
			}
			return Delegation{}, rfc.New(rfc.ErrDelegationDepthExceeded, fmt.Sprintf("delegation depth %d exceeds max %d", newDepth, md))
		}
	}
	if len(c.items) > 0 {
		d.PrevHash = c.items[len(c.items)-1].Hash
	}
	h, err := hashDelegation(d)
	if err != nil {
		return Delegation{}, err
	}
	d.Hash = h
	c.items = append(c.items, d)
	if m != nil {
		m.SetMaxObservedDelegationDepth(len(c.items))
	}
	return d, nil
}

// Head returns most recent delegation (nil if empty).
func (c *Chain) Head() *Delegation {
	if len(c.items) == 0 {
		return nil
	}
	return &c.items[len(c.items)-1]
}

// VerifyChain ensures hash integrity and linkage plus non-expired entries.
func (c *Chain) VerifyChain() error {
	for i, d := range c.items {
		h, err := hashDelegation(d)
		if err != nil {
			return err
		}
		if h != d.Hash {
			return fmt.Errorf("delegation hash mismatch at %d", i)
		}
		if i == 0 && d.PrevHash != "" {
			return errors.New("genesis delegation has prev hash")
		}
		if i > 0 && d.PrevHash != c.items[i-1].Hash {
			return fmt.Errorf("broken prev hash link at %d", i)
		}
		if time.Now().UTC().After(d.ExpiresAt) {
			return fmt.Errorf("delegation expired at %d", i)
		}
	}
	return nil
}

// ValidateScopeNarrowing ensures a new delegation does not widen scope relative to parent.
// Returns error if widened.
func ValidateScopeNarrowing(parent, child Delegation) error {
	// Simple rule: every key in child must exist in parent and value must be equal (or more restrictive if numeric range, omitted here).
	for k, v := range child.Scope {
		pv, ok := parent.Scope[k]
		if !ok {
			return fmt.Errorf("scope key %s not present in parent", k)
		}
		if pv != v {
			return fmt.Errorf("scope key %s value widened or changed (parent=%s child=%s)", k, pv, v)
		}
	}
	return nil
}

// validDelegationStatus reports whether status value is supported.
func validDelegationStatus(s string) bool {
	switch s {
	case StatusActive, StatusSuspended, StatusTerminated, StatusPending, StatusPartiallyRevoked:
		return true
	default:
		return false
	}
}

// ValidateDelegationStatusTransition ensures prohibited transitions (terminated->active, terminated->suspended).
func ValidateDelegationStatusTransition(old, new string) error {
	if !validDelegationStatus(old) || !validDelegationStatus(new) {
		return errors.New("invalid status value")
	}
	if old == new {
		return nil
	}
	if old == StatusTerminated && new != StatusTerminated {
		return errors.New("terminated delegations cannot transition")
	}
	if old == StatusActive && new == StatusPending {
		return errors.New("cannot revert to pending from active")
	}
	// partial revocation rules:
	// Allow active|suspended -> partially_revoked (scope reduction event)
	// Disallow transitions out of partially_revoked except termination (one-way narrowing)
	if old == StatusPartiallyRevoked && new != StatusPartiallyRevoked && new != StatusTerminated {
		return errors.New("partially_revoked delegations can only terminate")
	}
	if (old == StatusActive || old == StatusSuspended) && new == StatusPartiallyRevoked {
		return nil
	}
	return nil
}

func hashDelegation(d Delegation) (string, error) {
	tmp := struct {
		ID        string            `json:"id"`
		Subject   string            `json:"subject"`
		Delegate  string            `json:"delegate"`
		Scope     map[string]string `json:"scope"`
		IssuedAt  time.Time         `json:"issued_at"`
		ExpiresAt time.Time         `json:"expires_at"`
		PrevHash  string            `json:"prev_hash"`
	}{d.ID, d.Subject, d.Delegate, d.Scope, d.IssuedAt, d.ExpiresAt, d.PrevHash}
	data, err := json.Marshal(tmp)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}
