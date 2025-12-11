// Package a2a implements Agent-to-Agent Authorization Profile (draft-liu-oauth-a2a-profile).
package a2a

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// AgentIdentity represents an AI agent's identity.
type AgentIdentity struct {
	ID           string            `json:"agent_id"`
	Type         string            `json:"agent_type"` // e.g., "orchestrator", "worker", "verifier"
	OwnerID      string            `json:"owner_id,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// CallHop represents a single hop in an A2A call chain.
type CallHop struct {
	Caller    AgentIdentity `json:"caller"`
	Callee    AgentIdentity `json:"callee"`
	Timestamp time.Time     `json:"timestamp"`
	Action    string        `json:"action"`
	RequestID string        `json:"request_id,omitempty"`
	Signature string        `json:"signature,omitempty"` // Signature of this hop
	PrevHash  string        `json:"prev_hash,omitempty"` // Hash of previous hop (blockchain-style)
}

// A2ACallContext represents the full context of an agent interaction chain.
type A2ACallContext struct {
	ChainID   string        `json:"chain_id"`
	Initiator AgentIdentity `json:"initiator"` // Original user or root agent
	Hops      []CallHop     `json:"hops"`
}

// ComputeHopHash computes the hash of a hop including the previous hash.
func (h *CallHop) ComputeHash() string {
	// Create a copy without signature for hashing if needed, or hash specific fields
	// For simplicity, hash canonical JSON of struct (excluding signature if it signs the hash)
	// Here we try to chain hash: PrevHash + Caller + Callee + Action + Timestamp
	data := h.PrevHash + h.Caller.ID + h.Callee.ID + h.Action + h.Timestamp.Format(time.RFC3339Nano)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// A2ATransactionToken represents the token exchanged between agents.
// This would typically be embedded in a JWT "a2a" claim.
type A2ATransactionToken struct {
	Context   A2ACallContext `json:"context"`
	IssuedAt  time.Time      `json:"iat"`
	ExpiresAt time.Time      `json:"exp"`
}
