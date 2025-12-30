// Package oidc provides structured logging for OIDC operations.
package oidc

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// LogLevel represents the logging level.
type LogLevel string

const (
	// LogLevelDebug enables debug-level logging.
	LogLevelDebug LogLevel = "debug"

	// LogLevelInfo enables info-level logging.
	LogLevelInfo LogLevel = "info"

	// LogLevelWarn enables warn-level logging.
	LogLevelWarn LogLevel = "warn"

	// LogLevelError enables error-level logging.
	LogLevelError LogLevel = "error"

	// LogLevelFatal enables fatal-level logging.
	LogLevelFatal LogLevel = "fatal"
)

// LogFormat represents the log output format.
type LogFormat string

const (
	// LogFormatJSON outputs logs in JSON format.
	LogFormatJSON LogFormat = "json"

	// LogFormatConsole outputs logs in human-readable console format.
	LogFormatConsole LogFormat = "console"
)

// ContextKey is the type for context keys used in logging.
type ContextKey string

const (
	// ContextKeyCorrelationID is the context key for correlation IDs.
	ContextKeyCorrelationID ContextKey = "correlation_id"

	// ContextKeyClientID is the context key for client IDs.
	ContextKeyClientID ContextKey = "client_id"

	// ContextKeyUserID is the context key for user IDs.
	ContextKeyUserID ContextKey = "user_id"

	// ContextKeySessionID is the context key for session IDs.
	ContextKeySessionID ContextKey = "session_id"

	// ContextKeyGrantType is the context key for grant types.
	ContextKeyGrantType ContextKey = "grant_type"

	// ContextKeyEndpoint is the context key for endpoint names.
	ContextKeyEndpoint ContextKey = "endpoint"
)

// LoggerConfig configures the structured logger.
type LoggerConfig struct {
	// Level is the minimum log level to output.
	Level LogLevel

	// Format is the output format (json or console).
	Format LogFormat

	// Output is the writer to send logs to (defaults to os.Stdout).
	Output io.Writer

	// IncludeTimestamp determines whether to include timestamps.
	IncludeTimestamp bool

	// IncludeCaller determines whether to include caller information.
	IncludeCaller bool

	// RedactSensitiveData determines whether to redact tokens, secrets, etc.
	RedactSensitiveData bool

	// SensitiveFields is a list of field names to redact.
	SensitiveFields []string
}

// DefaultLoggerConfig returns the default logger configuration.
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:               LogLevelInfo,
		Format:              LogFormatJSON,
		Output:              os.Stdout,
		IncludeTimestamp:    true,
		IncludeCaller:       false,
		RedactSensitiveData: true,
		SensitiveFields: []string{
			"password",
			"secret",
			"token",
			"access_token",
			"refresh_token",
			"id_token",
			"authorization",
			"client_secret",
			"code",
			"device_code",
			"user_code",
		},
	}
}

// Logger provides structured logging with context support.
type Logger struct {
	logger          zerolog.Logger
	config          LoggerConfig
	sensitiveFields map[string]bool
}

// NewLogger creates a new structured logger.
func NewLogger(config LoggerConfig) *Logger {
	// Set output
	output := config.Output
	if output == nil {
		output = os.Stdout
	}

	// Configure zerolog
	var zlog zerolog.Logger

	if config.Format == LogFormatConsole {
		zlog = zerolog.New(zerolog.ConsoleWriter{Out: output, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Logger()
	} else {
		zlog = zerolog.New(output).With().Timestamp().Logger()
	}

	// Set log level
	switch config.Level {
	case LogLevelDebug:
		zlog = zlog.Level(zerolog.DebugLevel)
	case LogLevelInfo:
		zlog = zlog.Level(zerolog.InfoLevel)
	case LogLevelWarn:
		zlog = zlog.Level(zerolog.WarnLevel)
	case LogLevelError:
		zlog = zlog.Level(zerolog.ErrorLevel)
	case LogLevelFatal:
		zlog = zlog.Level(zerolog.FatalLevel)
	default:
		zlog = zlog.Level(zerolog.InfoLevel)
	}

	// Add caller information if requested
	if config.IncludeCaller {
		zlog = zlog.With().Caller().Logger()
	}

	// Build sensitive fields map
	sensitiveFields := make(map[string]bool)
	for _, field := range config.SensitiveFields {
		sensitiveFields[field] = true
	}

	return &Logger{
		logger:          zlog,
		config:          config,
		sensitiveFields: sensitiveFields,
	}
}

// WithContext creates a logger with context fields.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	zlog := l.logger

	// Add correlation ID if present
	if correlationID := ctx.Value(ContextKeyCorrelationID); correlationID != nil {
		if id, ok := correlationID.(string); ok {
			zlog = zlog.With().Str("correlation_id", id).Logger()
		}
	}

	// Add client ID if present
	if clientID := ctx.Value(ContextKeyClientID); clientID != nil {
		if id, ok := clientID.(string); ok {
			zlog = zlog.With().Str("client_id", id).Logger()
		}
	}

	// Add user ID if present
	if userID := ctx.Value(ContextKeyUserID); userID != nil {
		if id, ok := userID.(string); ok {
			zlog = zlog.With().Str("user_id", id).Logger()
		}
	}

	// Add session ID if present
	if sessionID := ctx.Value(ContextKeySessionID); sessionID != nil {
		if id, ok := sessionID.(string); ok {
			zlog = zlog.With().Str("session_id", id).Logger()
		}
	}

	// Add grant type if present
	if grantType := ctx.Value(ContextKeyGrantType); grantType != nil {
		if gt, ok := grantType.(string); ok {
			zlog = zlog.With().Str("grant_type", gt).Logger()
		}
	}

	// Add endpoint if present
	if endpoint := ctx.Value(ContextKeyEndpoint); endpoint != nil {
		if ep, ok := endpoint.(string); ok {
			zlog = zlog.With().Str("endpoint", ep).Logger()
		}
	}

	return &Logger{
		logger:          zlog,
		config:          l.config,
		sensitiveFields: l.sensitiveFields,
	}
}

// WithFields creates a logger with additional fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zlog := l.logger

	for key, value := range fields {
		// Redact sensitive fields
		if l.config.RedactSensitiveData && l.sensitiveFields[key] {
			value = "[REDACTED]"
		}

		switch v := value.(type) {
		case string:
			zlog = zlog.With().Str(key, v).Logger()
		case int:
			zlog = zlog.With().Int(key, v).Logger()
		case int64:
			zlog = zlog.With().Int64(key, v).Logger()
		case float64:
			zlog = zlog.With().Float64(key, v).Logger()
		case bool:
			zlog = zlog.With().Bool(key, v).Logger()
		case time.Duration:
			zlog = zlog.With().Dur(key, v).Logger()
		case error:
			zlog = zlog.With().Err(v).Logger()
		default:
			zlog = zlog.With().Interface(key, v).Logger()
		}
	}

	return &Logger{
		logger:          zlog,
		config:          l.config,
		sensitiveFields: l.sensitiveFields,
	}
}

// Debug logs a debug-level message.
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug-level message.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Info logs an info-level message.
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs a formatted info-level message.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Warn logs a warn-level message.
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs a formatted warn-level message.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Error logs an error-level message.
func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf logs a formatted error-level message.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// ErrorWithErr logs an error-level message with an error object.
func (l *Logger) ErrorWithErr(err error, msg string) {
	l.logger.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal-level message and exits.
func (l *Logger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal-level message and exits.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal().Msgf(format, args...)
}

// LogRequest logs an HTTP request.
func (l *Logger) LogRequest(method, path string, statusCode int, duration time.Duration) {
	l.logger.Info().
		Str("method", method).
		Str("path", path).
		Int("status", statusCode).
		Dur("duration_ms", duration).
		Msg("HTTP request")
}

// LogTokenOperation logs a token operation.
func (l *Logger) LogTokenOperation(operation, grantType string, success bool, duration time.Duration) {
	event := l.logger.Info()
	if !success {
		event = l.logger.Error()
	}

	event.
		Str("operation", operation).
		Str("grant_type", grantType).
		Bool("success", success).
		Dur("duration_ms", duration).
		Msg("Token operation")
}

// LogDeviceFlowOperation logs a device flow operation.
func (l *Logger) LogDeviceFlowOperation(operation, status string, duration time.Duration) {
	l.logger.Info().
		Str("operation", operation).
		Str("status", status).
		Dur("duration_ms", duration).
		Msg("Device flow operation")
}

// LogPAROperation logs a PAR operation.
func (l *Logger) LogPAROperation(operation string, success bool, duration time.Duration) {
	event := l.logger.Info()
	if !success {
		event = l.logger.Error()
	}

	event.
		Str("operation", operation).
		Bool("success", success).
		Dur("duration_ms", duration).
		Msg("PAR operation")
}

// LogStorageOperation logs a storage operation.
func (l *Logger) LogStorageOperation(operation, backend string, success bool, duration time.Duration) {
	event := l.logger.Debug()
	if !success {
		event = l.logger.Warn()
	}

	event.
		Str("operation", operation).
		Str("backend", backend).
		Bool("success", success).
		Dur("duration_ms", duration).
		Msg("Storage operation")
}

// LogRateLimitCheck logs a rate limit check.
func (l *Logger) LogRateLimitCheck(endpoint, result string, remaining int) {
	l.logger.Debug().
		Str("endpoint", endpoint).
		Str("result", result).
		Int("remaining", remaining).
		Msg("Rate limit check")
}

// LoAgentAuthenticationAttempt logs an authentication attempt.
func (l *Logger) LoAgentAuthenticationAttempt(provider, userID string, success bool) {
	event := l.logger.Info()
	if !success {
		event = l.logger.Warn()
	}

	event.
		Str("provider", provider).
		Str("user_id", userID).
		Bool("success", success).
		Msg("Authentication attempt")
}

// LogError logs an error with context.
func (l *Logger) LogError(errorType, message string, err error) {
	l.logger.Error().
		Str("error_type", errorType).
		Err(err).
		Msg(message)
}

// GlobalLogger is the global logger instance.
var GlobalLogger *Logger

func init() {
	// Initialize with default configuration
	GlobalLogger = NewLogger(DefaultLoggerConfig())
}

// SetGlobalLogger sets the global logger instance.
func SetGlobalLogger(logger *Logger) {
	GlobalLogger = logger
}

// Debug logs a debug message using the global logger.
func Debug(msg string) {
	GlobalLogger.Debug(msg)
}

// Debugf logs a formatted debug message using the global logger.
func Debugf(format string, args ...interface{}) {
	GlobalLogger.Debugf(format, args...)
}

// Info logs an info message using the global logger.
func Info(msg string) {
	GlobalLogger.Info(msg)
}

// Infof logs a formatted info message using the global logger.
func Infof(format string, args ...interface{}) {
	GlobalLogger.Infof(format, args...)
}

// Warn logs a warn message using the global logger.
func Warn(msg string) {
	GlobalLogger.Warn(msg)
}

// Warnf logs a formatted warn message using the global logger.
func Warnf(format string, args ...interface{}) {
	GlobalLogger.Warnf(format, args...)
}

// Error logs an error message using the global logger.
func Error(msg string) {
	GlobalLogger.Error(msg)
}

// Errorf logs a formatted error message using the global logger.
func Errorf(format string, args ...interface{}) {
	GlobalLogger.Errorf(format, args...)
}

// ErrorWithErr logs an error with an error object using the global logger.
func ErrorWithErr(err error, msg string) {
	GlobalLogger.ErrorWithErr(err, msg)
}

// Fatal logs a fatal message using the global logger.
func Fatal(msg string) {
	GlobalLogger.Fatal(msg)
}

// Fatalf logs a formatted fatal message using the global logger.
func Fatalf(format string, args ...interface{}) {
	GlobalLogger.Fatalf(format, args...)
}

// WithContext creates a logger with context fields using the global logger.
func WithContext(ctx context.Context) *Logger {
	return GlobalLogger.WithContext(ctx)
}

// WithFields creates a logger with additional fields using the global logger.
func WithFields(fields map[string]interface{}) *Logger {
	return GlobalLogger.WithFields(fields)
}
