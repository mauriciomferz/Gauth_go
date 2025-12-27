// Package mcp provides HTTP Server-Sent Events (SSE) transport implementation for MCP protocol
package mcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// SSE connection settings
	sseReadTimeout     = 60 * time.Second
	sseReconnectDelay  = 1 * time.Second
	sseMaxReconnects   = 5
	sseHeartbeatPeriod = 30 * time.Second
)

// SSETransport implements MCP protocol over HTTP Server-Sent Events
type SSETransport struct {
	url     string
	headers http.Header
	client  *http.Client

	// Connection state
	conn         io.ReadCloser
	connMu       sync.RWMutex
	connected    bool
	reconnecting bool
	lastEventID  string

	// Message handling
	receiveCh chan []byte
	errorCh   chan error

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Callbacks
	onConnect    func()
	onDisconnect func(error)
	onMessage    func([]byte)

	// Metrics
	messagesReceived int64
	reconnectCount   int
	lastError        error
	lastErrorTime    time.Time
}

// sseEvent represents a Server-Sent Event
type sseEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// NewSSETransport creates a new SSE transport
func NewSSETransport(url string, headers http.Header) *SSETransport {
	ctx, cancel := context.WithCancel(context.Background())

	if headers == nil {
		headers = http.Header{}
	}
	headers.Set("Accept", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")

	return &SSETransport{
		url:       url,
		headers:   headers,
		client:    &http.Client{Timeout: 0}, // No timeout for streaming
		receiveCh: make(chan []byte, 100),
		errorCh:   make(chan error, 10),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Connect establishes SSE connection with auto-reconnect
func (t *SSETransport) Connect(ctx context.Context) error {
	return t.connectWithRetry(ctx, 0)
}

// connectWithRetry attempts to connect with exponential backoff
func (t *SSETransport) connectWithRetry(ctx context.Context, attempt int) error {
	t.connMu.Lock()
	if t.reconnecting && attempt > sseMaxReconnects {
		t.connMu.Unlock()
		return fmt.Errorf("max reconnection attempts (%d) exceeded", sseMaxReconnects)
	}
	t.reconnecting = true
	t.connMu.Unlock()

	// Exponential backoff for reconnection
	if attempt > 0 {
		// #nosec G115
		delay := sseReconnectDelay * time.Duration(1<<uint(attempt-1))
		if delay > reconnectMaxDelay {
			delay = reconnectMaxDelay
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", t.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for key, values := range t.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add Last-Event-ID if available (for resume)
	if t.lastEventID != "" {
		req.Header.Set("Last-Event-ID", t.lastEventID)
	}

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		t.lastError = err
		t.lastErrorTime = time.Now()

		// Retry with exponential backoff
		if attempt < sseMaxReconnects {
			return t.connectWithRetry(ctx, attempt+1)
		}
		return fmt.Errorf("failed to connect to %s: %w", t.url, err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		t.lastError = err
		t.lastErrorTime = time.Now()

		if attempt < sseMaxReconnects {
			return t.connectWithRetry(ctx, attempt+1)
		}
		return err
	}

	// Verify content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		_ = resp.Body.Close()
		return fmt.Errorf("unexpected content type: %s", contentType)
	}

	t.connMu.Lock()
	t.conn = resp.Body
	t.connected = true
	t.reconnecting = false
	if attempt > 0 {
		t.reconnectCount++
	}
	t.connMu.Unlock()

	// Start goroutine for reading events
	t.wg.Add(1)
	go t.readPump()

	// Notify connection established
	if t.onConnect != nil {
		t.onConnect()
	}

	return nil
}

// readPump handles incoming SSE events
func (t *SSETransport) readPump() {
	defer t.wg.Done()
	defer t.handleDisconnect(nil)

	t.connMu.RLock()
	conn := t.conn
	t.connMu.RUnlock()

	if conn == nil {
		return
	}

	scanner := bufio.NewScanner(conn)
	scanner.Split(bufio.ScanLines)

	var currentEvent sseEvent

	for scanner.Scan() {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		// Empty line indicates end of event
		if line == "" {
			if currentEvent.Data != "" {
				t.processEvent(&currentEvent)
			}
			currentEvent = sseEvent{}
			continue
		}

		// Ignore comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		field := parts[0]
		value := strings.TrimPrefix(parts[1], " ")

		switch field {
		case "id":
			currentEvent.ID = value
			t.lastEventID = value

		case "event":
			currentEvent.Event = value

		case "data":
			if currentEvent.Data != "" {
				currentEvent.Data += "\n"
			}
			currentEvent.Data += value

		case "retry":
			// Parse retry time (milliseconds)
			// Could be used to adjust reconnection delay
		}
	}

	if err := scanner.Err(); err != nil {
		// Use select to avoid panic if channel is closed
		select {
		case t.errorCh <- fmt.Errorf("scanner error: %w", err):
		case <-t.ctx.Done():
			// Transport is closing, ignore error
		}
	}
}

// processEvent processes a complete SSE event
func (t *SSETransport) processEvent(event *sseEvent) {
	// Handle different event types
	switch event.Event {
	case "message", "":
		// Default event type is "message"
		data := []byte(event.Data)

		t.messagesReceived++

		// Send raw bytes to receive channel
		select {
		case t.receiveCh <- data:
			if t.onMessage != nil {
				t.onMessage(data)
			}
		case <-t.ctx.Done():
			return
		}

	case "heartbeat":
		// Heartbeat event, no action needed

	case "error":
		// Use select to avoid panic if channel is closed
		select {
		case t.errorCh <- fmt.Errorf("server error: %s", event.Data):
		case <-t.ctx.Done():
			// Transport is closing, ignore error
		}
	}
}

// Send is not supported for SSE (read-only transport)
func (t *SSETransport) Send(ctx context.Context, message []byte) error {
	return fmt.Errorf("SSE transport is read-only, cannot send messages")
}

// Receive receives a message from the SSE stream
func (t *SSETransport) Receive(ctx context.Context) ([]byte, error) {
	select {
	case msg := <-t.receiveCh:
		return msg, nil
	case err := <-t.errorCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.ctx.Done():
		return nil, fmt.Errorf("transport closed")
	}
}

// Close closes the SSE connection
func (t *SSETransport) Close() error {
	t.cancel()

	t.connMu.Lock()
	if t.conn != nil {
		_ = t.conn.Close()
		t.conn = nil
	}
	t.connected = false
	t.connMu.Unlock()

	// Wait for goroutines to finish
	t.wg.Wait()

	// Close channels
	close(t.receiveCh)
	close(t.errorCh)

	return nil
}

// handleDisconnect handles connection loss and triggers reconnection
func (t *SSETransport) handleDisconnect(err error) {
	t.connMu.Lock()
	wasConnected := t.connected
	t.connected = false
	if t.conn != nil {
		_ = t.conn.Close()
		t.conn = nil
	}
	t.connMu.Unlock()

	if !wasConnected {
		return
	}

	// Notify disconnect
	if t.onDisconnect != nil {
		t.onDisconnect(err)
	}

	// Attempt reconnection
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		if err := t.connectWithRetry(t.ctx, 1); err != nil {
			// Use select to avoid panic if channel is closed
			select {
			case t.errorCh <- fmt.Errorf("reconnection failed: %w", err):
			case <-t.ctx.Done():
				// Transport is closing, ignore error
			}
		}
	}()
}

// IsConnected returns true if the transport is connected
func (t *SSETransport) IsConnected() bool {
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	return t.connected
}

// SetOnConnect sets the connection callback
func (t *SSETransport) SetOnConnect(fn func()) {
	t.onConnect = fn
}

// SetOnDisconnect sets the disconnection callback
func (t *SSETransport) SetOnDisconnect(fn func(error)) {
	t.onDisconnect = fn
}

// SetOnMessage sets the message callback
func (t *SSETransport) SetOnMessage(fn func([]byte)) {
	t.onMessage = fn
}

// GetMetrics returns transport metrics
func (t *SSETransport) GetMetrics() map[string]interface{} {
	t.connMu.RLock()
	defer t.connMu.RUnlock()

	return map[string]interface{}{
		"connected":         t.connected,
		"reconnecting":      t.reconnecting,
		"messages_received": t.messagesReceived,
		"reconnect_count":   t.reconnectCount,
		"last_event_id":     t.lastEventID,
		"last_error":        t.lastError,
		"last_error_time":   t.lastErrorTime,
	}
}
