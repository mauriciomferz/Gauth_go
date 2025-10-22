package token

// NewServiceForBenchmarks creates a service instance for benchmarking
// This function provides the exact signature expected by the benchmark tests
func NewServiceForBenchmarks(config Config, store *MemoryStore) *Service {
	return &Service{
		store:     store,
		blacklist: &MemoryBlacklist{blacklist: make(map[string]string)},
		config:    &config,
	}
}
