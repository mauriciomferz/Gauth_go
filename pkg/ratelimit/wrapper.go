package ratelimit

// AsAlgorithm wraps a rate limiting implementation to ensure it satisfies the Algorithm interface
func AsAlgorithm(algo Algorithm) Algorithm {
	return algo
}

type NewAlgorithm func(*Config) Algorithm

func WrapTokenBucket(config *Config) Algorithm {
	return AsAlgorithm(NewTokenBucket(config))
}

func WrapSlidingWindow(config *Config) Algorithm {
	return AsAlgorithm(NewSlidingWindow(config))
}
