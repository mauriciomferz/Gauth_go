package resources

// ErrorMessages contains all error message templates used in the application
var ErrorMessages = map[string]string{
	"ServiceUnavailable": "Service %s is currently unavailable",
	"CircuitOpen":        "Circuit breaker is open for service %s",
	"RateLimitExceeded":  "Rate limit exceeded for service %s",
	"DependencyFailed":   "Dependency %s failed: %v",
	"TimeoutExceeded":    "Request timeout exceeded for service %s",
	"InvalidConfig":      "Invalid configuration for service %s: %v",
}

// StatusMessages contains all status message templates
var StatusMessages = map[string]string{
	"ServiceStarting":    "Starting service %s",
	"ServiceReady":       "Service %s is ready",
	"ServiceDegraded":    "Service %s is in degraded state",
	"ServiceStopping":    "Stopping service %s",
	"CircuitStateChange": "Circuit breaker state changed from %s to %s",
}

// LogMessages contains all log message templates
var LogMessages = map[string]string{
	"RequestReceived":  "Received request for service %s",
	"RequestCompleted": "Completed request for service %s in %v",
	"RequestFailed":    "Request failed for service %s: %v",
	"MetricRecorded":   "Recorded metric for service %s: %v",
	"ConfigChanged":    "Configuration changed for service %s",
}
