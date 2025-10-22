package common

import (
	"fmt"
	"time"
)

// Config represents common configuration options
type Config struct {
	Debug       bool          `json:"debug" yaml:"debug"`
	LogLevel    string        `json:"log_level" yaml:"log_level"`
	Environment string        `json:"environment" yaml:"environment"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Debug:       false,
		LogLevel:    "info",
		Environment: "development",
		Timeout:     30 * time.Second,
	}
}

// Response represents a common response structure
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewSuccessResponse creates a successful response
func NewSuccessResponse(data interface{}, message string) *Response {
	return &Response{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(err error, message string) *Response {
	return &Response{
		Success: false,
		Message: message,
		Error:   err.Error(),
	}
}

// Logger interface for common logging
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// SimpleLogger is a basic logger implementation
type SimpleLogger struct{}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{}
}

func (l *SimpleLogger) Debug(args ...interface{}) {
	fmt.Println("[DEBUG]", fmt.Sprint(args...))
}

func (l *SimpleLogger) Info(args ...interface{}) {
	fmt.Println("[INFO]", fmt.Sprint(args...))
}

func (l *SimpleLogger) Warn(args ...interface{}) {
	fmt.Println("[WARN]", fmt.Sprint(args...))
}

func (l *SimpleLogger) Error(args ...interface{}) {
	fmt.Println("[ERROR]", fmt.Sprint(args...))
}

func (l *SimpleLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

func (l *SimpleLogger) Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func (l *SimpleLogger) Warnf(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

func (l *SimpleLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// Validator interface for common validation
type Validator interface {
	Validate(interface{}) error
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int `json:"requests_per_second" yaml:"requests_per_second"`
	BurstSize         int `json:"burst_size" yaml:"burst_size"`
	WindowSize        int `json:"window_size" yaml:"window_size"`
}

// Utils contains common utility functions
type Utils struct{}

// NewUtils creates a new utils instance
func NewUtils() *Utils {
	return &Utils{}
}

// GenerateID generates a simple ID (placeholder implementation)
func (u *Utils) GenerateID() string {
	return fmt.Sprintf("id_%d", time.Now().UnixNano())
}
