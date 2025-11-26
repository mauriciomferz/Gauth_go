package revocation

import (
	"fmt"
	"log"
	"os"
)

// Logger interface for dependency injection
type Logger interface {
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
}

// SimpleLogger implements the Logger interface using standard log package
type SimpleLogger struct {
	prefix string
	logger *log.Logger
}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger(prefix string) Logger {
	return &SimpleLogger{
		prefix: prefix,
		logger: log.New(os.Stdout, fmt.Sprintf("[%s] ", prefix), log.LstdFlags|log.Lmicroseconds),
	}
}

func (l *SimpleLogger) Info(msg string) {
	l.logger.Printf("INFO: %s", msg)
}

func (l *SimpleLogger) Infof(format string, args ...interface{}) {
	l.logger.Printf("INFO: "+format, args...)
}

func (l *SimpleLogger) Warn(msg string) {
	l.logger.Printf("WARN: %s", msg)
}

func (l *SimpleLogger) Warnf(format string, args ...interface{}) {
	l.logger.Printf("WARN: "+format, args...)
}

func (l *SimpleLogger) Error(msg string) {
	l.logger.Printf("ERROR: %s", msg)
}

func (l *SimpleLogger) Errorf(format string, args ...interface{}) {
	l.logger.Printf("ERROR: "+format, args...)
}
