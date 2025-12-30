package rate

import (
	"github.com/mauriciomferz/AgentAuth/pkg/rate"
)

// Re-export all types and functions from pkg/rate
type (
	Config    = rate.Config
	Limiter   = rate.Limiter
	Algorithm = rate.Algorithm
)

// Re-export constants
const (
	TokenBucket   = rate.TokenBucket
	SlidingWindow = rate.SlidingWindow
	FixedWindow   = rate.FixedWindow
)

// Re-export functions
var (
	DefaultConfig   = rate.DefaultConfig
	NewLimiter      = rate.NewLimiter
	WrapTokenBucket = rate.WrapTokenBucket
	Demo            = rate.Demo
)
