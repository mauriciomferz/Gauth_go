package rate

import "errors"

var (
	// ErrLimitExceeded is returned when the rate limit is exceeded
	ErrLimitExceeded = errors.New("rate limit exceeded")

	// ErrInvalidLimit is returned when a limit configuration is invalid
	ErrInvalidLimit = errors.New("invalid rate limit")

	// ErrInvalidWindow is returned when a time window configuration is invalid
	ErrInvalidWindow = errors.New("invalid time window")

	// ErrInvalidBurst is returned when a burst size configuration is invalid
	ErrInvalidBurst = errors.New("invalid burst size")

	// ErrInvalidPrecision is returned when a precision configuration is invalid
	ErrInvalidPrecision = errors.New("invalid precision")

	// ErrStoreRequired is returned when a required store is not provided
	ErrStoreRequired = errors.New("store is required")

	// ErrStoreFailed is returned when a store operation fails
	ErrStoreFailed = errors.New("store operation failed")

	// ErrInvalidKey is returned when a key is invalid
	ErrInvalidKey = errors.New("invalid key")

	// ErrInvalidCount is returned when a count is invalid
	ErrInvalidCount = errors.New("invalid count")

	// ErrClosed is returned when operating on a closed limiter
	ErrClosed = errors.New("limiter is closed")
)
