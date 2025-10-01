// Package handlers provides strongly typed event handlers for the events package.
//
// This package contains various event handlers that process events in different ways:
// - LoggingHandler: Logs events to the standard logger
// - MetricsHandler: Collects metrics from events
// - AuditHandler: Stores events for audit purposes
// - MultiHandler: Combines multiple handlers
// - FilteredHandler: Filters events before passing them to the underlying handler
//
// Each handler is now implemented in its own file for better organization.
package handlers
