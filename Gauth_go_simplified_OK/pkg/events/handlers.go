package events

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogHandler writes events to a log file
type LogHandler struct {
	logger *log.Logger
}

// NewLogHandler creates a new log handler
func NewLogHandler(path string) (*LogHandler, error) {
	// Validate path to prevent directory traversal attacks
	if filepath.IsAbs(path) {
		cleanPath := filepath.Clean(path)
		if !strings.HasPrefix(cleanPath, filepath.Clean(filepath.Dir(path))) {
			return nil, fmt.Errorf("invalid log file path: potential directory traversal")
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &LogHandler{
		logger: log.New(file, "", log.LstdFlags),
	}, nil
}

// Handle implements EventHandler
func (h *LogHandler) Handle(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("failed to marshal event: %v", err)
		return
	}
	h.logger.Printf("%s", data)
}

// MetricsHandler sends events to the metrics system
type MetricsHandler struct {
	collector MetricsCollector
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(collector MetricsCollector) *MetricsHandler {
	return &MetricsHandler{
		collector: collector,
	}
}

// Handle implements EventHandler
func (h *MetricsHandler) Handle(event Event) {
	switch event.Type {
	case EventTypeAuth:
		h.collector.RecordAuthEvent(event)
	case EventTypeAuthz:
		h.collector.RecordAuthzEvent(event)
	case EventTypeToken:
		h.collector.RecordTokenEvent(event)
	}
}

// BufferedHandler buffers events before processing
type BufferedHandler struct {
	handler    EventHandler
	buffer     []Event
	bufferSize int
	flushTime  time.Duration
	done       chan struct{}
}

// NewBufferedHandler creates a new buffered handler
func NewBufferedHandler(handler EventHandler, bufferSize int, flushTime time.Duration) *BufferedHandler {
	h := &BufferedHandler{
		handler:    handler,
		buffer:     make([]Event, 0, bufferSize),
		bufferSize: bufferSize,
		flushTime:  flushTime,
		done:       make(chan struct{}),
	}

	go h.flushLoop()
	return h
}

// Handle implements EventHandler
func (h *BufferedHandler) Handle(event Event) {
	h.buffer = append(h.buffer, event)
	if len(h.buffer) >= h.bufferSize {
		h.flush()
	}
}

func (h *BufferedHandler) flush() {
	if len(h.buffer) == 0 {
		return
	}

	for _, event := range h.buffer {
		h.handler.Handle(event)
	}
	h.buffer = h.buffer[:0]
}

func (h *BufferedHandler) flushLoop() {
	ticker := time.NewTicker(h.flushTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.flush()
		case <-h.done:
			h.flush()
			return
		}
	}
}

// Close flushes remaining events and stops the flush loop
func (h *BufferedHandler) Close() {
	close(h.done)
}

// FilterHandler filters events based on criteria
type FilterHandler struct {
	handler EventHandler
	filter  func(Event) bool
}

// NewFilterHandler creates a new filter handler
func NewFilterHandler(handler EventHandler, filter func(Event) bool) *FilterHandler {
	return &FilterHandler{
		handler: handler,
		filter:  filter,
	}
}

// Handle implements EventHandler
func (h *FilterHandler) Handle(event Event) {
	if h.filter(event) {
		h.handler.Handle(event)
	}
}

// ChainHandler chains multiple handlers together
type ChainHandler struct {
	handlers []EventHandler
}

// NewChainHandler creates a new chain handler
func NewChainHandler(handlers ...EventHandler) *ChainHandler {
	return &ChainHandler{
		handlers: handlers,
	}
}

// Handle implements EventHandler
func (h *ChainHandler) Handle(event Event) {
	for _, handler := range h.handlers {
		handler.Handle(event)
	}
}

// AsyncHandler handles events asynchronously
type AsyncHandler struct {
	handler EventHandler
	queue   chan Event
	done    chan struct{}
}

// NewAsyncHandler creates a new async handler
func NewAsyncHandler(handler EventHandler, queueSize int) *AsyncHandler {
	h := &AsyncHandler{
		handler: handler,
		queue:   make(chan Event, queueSize),
		done:    make(chan struct{}),
	}

	go h.processLoop()
	return h
}

// Handle implements EventHandler
func (h *AsyncHandler) Handle(event Event) {
	select {
	case h.queue <- event:
	default:
		log.Printf("event queue full, dropping event: %v", event)
	}
}

func (h *AsyncHandler) processLoop() {
	for {
		select {
		case event := <-h.queue:
			h.handler.Handle(event)
		case <-h.done:
			return
		}
	}
}

// Close stops the async handler
func (h *AsyncHandler) Close() {
	close(h.done)
}
