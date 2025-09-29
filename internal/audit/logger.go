// Package audit provides interfaces for audit logging.
package audit

import (
	"context"

	"github.com/Gimel-Foundation/gauth/internal/events"
)

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	// LogEvent logs a system event with audit context
	LogEvent(ctx context.Context, event *events.Event) error

	// LogEntry logs an audit entry directly
	LogEntry(ctx context.Context, entry *Entry) error

	// Query searches audit logs with filters
	Query(ctx context.Context, filter *Filter) ([]*Entry, error)
}

// Filter defines criteria for querying audit logs
type Filter struct {
	// Types filters by entry types
	Types []Type

	// Levels filters by severity levels
	Levels []Level

	// ActorID filters by actor
	ActorID string

	// Resource filters by resource
	Resource string

	// TimeRange specifies a time range
	TimeRange *TimeRange
}

// TimeRange represents a time range for filtering
type TimeRange struct {
	// Start is the start time (inclusive)
	Start int64

	// End is the end time (exclusive)
	End int64
}

// Option configures a logger
type Option func(*Options)

// Options holds logger configuration
type Options struct {
	// BufferSize is the size of the event buffer
	BufferSize int

	// AsyncWrite enables asynchronous writes
	AsyncWrite bool

	// FilterFunc filters entries before logging
	FilterFunc func(*Entry) bool
}

// WithBufferSize sets the buffer size
func WithBufferSize(size int) Option {
	return func(o *Options) {
		o.BufferSize = size
	}
}

// WithAsyncWrite enables async writing
func WithAsyncWrite(async bool) Option {
	return func(o *Options) {
		o.AsyncWrite = async
	}
}

// WithFilter adds an entry filter
func WithFilter(f func(*Entry) bool) Option {
	return func(o *Options) {
		o.FilterFunc = f
	}
}

// DefaultOptions returns default options
func DefaultOptions() *Options {
	return &Options{
		BufferSize: 1000,
		AsyncWrite: true,
	}
}
