package errors

import "errors"

// Circuit breaker errors
var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)
