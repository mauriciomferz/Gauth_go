package ratelimit

// AsAlgorithm wraps a rate limiting implementation to ensure it satisfies the Algorithm interface
func AsAlgorithm(algo Algorithm) Algorithm {
	return algo
}

// NewAlgorithm is a constructor function type that creates an Algorithm implementation
type NewAlgorithm func(*Config) Algorithm

// WrapTokenBucket wraps TokenBucket constructor to return Algorithm interface
func WrapTokenBucket(config *Config) Algorithm {
	return AsAlgorithm(NewTokenBucket(config))
}

// WrapSlidingWindow wraps SlidingWindow constructor to return Algorithm interface
func WrapSlidingWindow(config *Config) Algorithm {
	return AsAlgorithm(NewSlidingWindow(config))
}
