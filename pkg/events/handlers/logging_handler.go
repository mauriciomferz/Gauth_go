// Package handlers provides event handlers for the GAuth events system
package handlers

import (
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/events"
)

// LoggingHandler logs events to the standard logger
type LoggingHandler struct {
	// Options
	IncludeTimestamp bool
	IncludeMetadata  bool
	LogLevel         string
}

// NewLoggingHandler creates a new logging handler with default settings
func NewLoggingHandler() *LoggingHandler {
	return &LoggingHandler{
		IncludeTimestamp: true,
		IncludeMetadata:  true,
		LogLevel:         "info",
	}
}

// Handle implements the EventHandler interface
func (h *LoggingHandler) Handle(event events.Event) {
	// Basic log format
	logMsg := string(event.Type) + ":" + event.Action + " - " + event.Message

	// Add status if present
	if event.Status != "" {
		logMsg += " [" + event.Status + "]"
	}

	// Add timestamp if enabled
	if h.IncludeTimestamp {
		logMsg = event.Timestamp.Format(time.RFC3339) + " " + logMsg
	}

	// Add subject and resource if present
	if event.Subject != "" {
		logMsg += " Subject=" + event.Subject
	}

	if event.Resource != "" {
		logMsg += " Resource=" + event.Resource
	}

	// Add error if present
	if event.Error != "" {
		logMsg += " Error=" + event.Error
	}

	// Add metadata if enabled and present
	if h.IncludeMetadata && event.Metadata != nil && event.Metadata.Len() > 0 {
		logMsg += " Metadata={"
		keys := event.Metadata.Keys()
		for i, k := range keys {
			if i > 0 {
				logMsg += ", "
			}
			if value, exists := event.Metadata.Get(k); exists {
				logMsg += k + ":" + value.ToString()
			}
		}
		logMsg += "}"
	}

	// Log with appropriate level
	switch h.LogLevel {
	case "debug":
		log.Printf("DEBUG: %s", logMsg)
	case "info":
		log.Printf("INFO: %s", logMsg)
	case "warn":
		log.Printf("WARN: %s", logMsg)
	case "error":
		log.Printf("ERROR: %s", logMsg)
	default:
		log.Print(logMsg)
	}
}
