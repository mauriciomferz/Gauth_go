// Package audit/entry.go: RFC111 Compliance Mapping
//
// This file implements the audit entry structure and builder as required by RFC111:
//   - Type-safe, explicit audit entry struct for all protocol events
//   - Builder methods for actor, action, target, and result fields
//   - All audit entries are structured and verifiable
//
// Relevant RFC111 Sections:
//   - Section 6: How GAuth works (audit, event, compliance)
//   - Section 7: Benefits (verifiability, auditability)
//
// Compliance:
//   - All fields are explicit and type-safe (no ambiguous types)
//   - Audit entries are structured, verifiable, and cover all protocol steps
//   - No exclusions (Web3, DNA, decentralized auth) are present
//   - See README and docs/ for full protocol mapping
//
// License: Apache 2.0 (see LICENSE file)

package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// NewEntry creates a new audit Entry with the given type.
func NewEntry(typ string) *Entry {
	return &Entry{
		ID:        generateID(),
		Type:      typ,
		Timestamp: time.Now(),
		Metadata:  make(Metadata),
		Tags:      []string{},
	}
}

// WithActor sets the actor ID and type.
func (e *Entry) WithActor(id, typ string) *Entry {
	e.ActorID = id
	e.ActorType = typ
	return e
}

// WithAction sets the action.
func (e *Entry) WithAction(action string) *Entry {
	e.Action = action
	return e
}

// WithTarget sets the target ID and type.
func (e *Entry) WithTarget(id, typ string) *Entry {
	e.TargetID = id
	e.TargetType = typ
	return e
}

// WithResult sets the result.
func (e *Entry) WithResult(result string) *Entry {
	e.Result = result
	return e
}

// WithMetadata adds a key-value pair to metadata.
func (e *Entry) WithMetadata(key string, value string) *Entry {
	e.Metadata[key] = value
	return e
}

// calculateHash computes a hash of the entry's core fields.
func (e *Entry) calculateHash() string {
	h := sha256.New()
	h.Write([]byte(e.ID))
	h.Write([]byte(e.Type))
	h.Write([]byte(e.Action))
	h.Write([]byte(e.Result))
	h.Write([]byte(e.ActorID))
	h.Write([]byte(e.TargetID))
	h.Write([]byte(e.Timestamp.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// generateID creates a pseudo-unique ID for entries (for demo/testing).
func generateID() string {
	return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
}

// Common constants for test compatibility
const (
	TypeAuth             = "auth"
	TypeToken            = "token"
	TypeResource         = "resource"
	ActorUser            = "user"
	ActionLogin          = "login"
	ActionResourceAccess = "resource_access"
	ResultSuccess        = "success"
)
