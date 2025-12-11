package a2a

import (
	"errors"
	"time"
)

// ChainBuilder helps construct valid A2A call chains.
type ChainBuilder struct {
	context *A2ACallContext
}

// NewChainBuilder starts a new call chain.
func NewChainBuilder(chainID string, initiator AgentIdentity) *ChainBuilder {
	return &ChainBuilder{
		context: &A2ACallContext{
			ChainID:   chainID,
			Initiator: initiator,
			Hops:      make([]CallHop, 0),
		},
	}
}

// AddHop adds a new call execution hop to the chain.
func (b *ChainBuilder) AddHop(caller, callee AgentIdentity, action string) error {
	prevHash := ""
	if len(b.context.Hops) > 0 {
		// Calculate hash of the last hop
		lastHop := b.context.Hops[len(b.context.Hops)-1]
		prevHash = lastHop.ComputeHash()
	} else {
		// First hop links to initiator (conceptually) or ChainID
		prevHash = b.context.ChainID
	}

	hop := CallHop{
		Caller:    caller,
		Callee:    callee,
		Timestamp: time.Now().UTC(),
		Action:    action,
		PrevHash:  prevHash,
	}

	// In a real implementation, caller would sign the hop here.
	// hop.Signature = sign(hop)

	b.context.Hops = append(b.context.Hops, hop)
	return nil
}

// Build returns the constructed context.
func (b *ChainBuilder) Build() *A2ACallContext {
	return b.context
}

// ChainValidator validates A2A call chains.
type ChainValidator struct{}

// Validate checks chain integrity and consistency.
func (v *ChainValidator) Validate(ctx *A2ACallContext) error {
	if ctx.ChainID == "" {
		return errors.New("missing chain ID")
	}
	if ctx.Initiator.ID == "" {
		return errors.New("missing initiator")
	}

	for i, hop := range ctx.Hops {
		expectedPrevHash := ""
		if i == 0 {
			expectedPrevHash = ctx.ChainID
		} else {
			expectedPrevHash = ctx.Hops[i-1].ComputeHash()
		}

		if hop.PrevHash != expectedPrevHash {
			return errors.New("broken chain link at hop " + string(rune(i)))
		}

		// Validate signatures here if implemented
	}

	return nil
}
