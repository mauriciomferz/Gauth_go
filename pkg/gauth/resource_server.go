package gauth

import (
	"time"

	"github.com/Gimel-Foundation/gauth/internal/errors"
)

// SetRateLimit enables basic in-memory rate limiting for demonstration.
func (s *ResourceServer) SetRateLimit(requests int, per time.Duration) {
	if s.rateLimiter == nil {
		s.rateLimiter = &simpleRateLimiter{
			limit: requests,
			window: per,
			counts: make(map[string][]time.Time),
		}
	} else {
		s.rateLimiter.limit = requests
		s.rateLimiter.window = per
	}
}

// simpleRateLimiter is a naive in-memory rate limiter (per subject)
type simpleRateLimiter struct {
	limit  int
	window time.Duration
	counts map[string][]time.Time
}

func (rl *simpleRateLimiter) Allow(subject string) bool {
       now := time.Now()
       times := rl.counts[subject]
       // Remove timestamps outside the window
       cutoff := now.Add(-rl.window)
       var filtered []time.Time
       for _, t := range times {
	       if t.After(cutoff) {
		       filtered = append(filtered, t)
	       }
       }
       if len(filtered) >= rl.limit {
	       rl.counts[subject] = filtered
	       return false
       }
       rl.counts[subject] = append(filtered, now)
       return true
}

// ResourceServer represents a server that provides protected resources
type ResourceServer struct {
	name        string
	auth        *GAuth
	rateLimiter *simpleRateLimiter
}






// NewResourceServer creates a new resource server instance
func NewResourceServer(name string, auth *GAuth) *ResourceServer {
	return &ResourceServer{
		name: name,
		auth: auth,
	}
}

// ProcessTransaction processes a transaction with the given token
func (s *ResourceServer) ProcessTransaction(tx TransactionDetails, token string) (string, error) {
	// Validate token
	tokenData, err := s.auth.ValidateToken(token)
	if err != nil {
		return "", err
	}

       // Rate limiting: use subject as key
       subject := tokenData.OwnerID
       if s.rateLimiter != nil && !s.rateLimiter.Allow(subject) {
	       return "", errors.New(errors.ErrRateLimitExceeded, "rate limit exceeded for subject")
       }

	// Check if token has required scope
	hasScope := false
	for _, scope := range tokenData.Scope {
		if scope == "transaction:execute" {
			hasScope = true
			break
		}
	}
       if !hasScope {
	       return "", errors.New(errors.ErrInsufficientScope, "token lacks transaction:execute scope")
       }

	// In a real implementation, this would process the transaction
	// For this example, we just return a success message
	return "Transaction processed successfully", nil
}
