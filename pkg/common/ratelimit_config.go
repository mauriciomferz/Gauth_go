// Package common provides shared types for the GAuth project.
package common

// RateLimitConfig represents rate limiting configuration.
type RateLimitConfig struct {
	RequestsPerSecond int `json:"requests_per_second"` // Maximum requests per second
	BurstSize         int `json:"burst_size"`          // Maximum burst size
	WindowSize        int `json:"window_size"`         // Time window in seconds
}
