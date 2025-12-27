// Package mcp provides WebSocket transport implementation for MCP protocol
package mcp

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// WebSocket connection settings
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 * 1024 // 1MB

	// Reconnection settings
	maxReconnectAttempts = 5
	reconnectBaseDelay   = 1 * time.Second
	reconnectMaxDelay    = 30 * time.Second
)

// WebSocketTransport implements MCP protocol over WebSocket
type WebSocketTransport struct {
	url     string
	headers http.Header
	conn    *websocket.Conn
	connMu  sync.RWMutex

	// Message handling
	sendCh    chan []byte
	receiveCh chan []byte
	errorCh   chan error

	// Lifecycle
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	connected    bool
	reconnecting bool

	// Callbacks
	onConnect    func()
	onDisconnect func(error)
	onMessage    func([]byte)

	// Metrics
	messagesSent     int64
	messagesReceived int64
	reconnectCount   int
	lastError        error
	lastErrorTime    time.Time
}

// NewWebSocketTransport creates a new WebSocket transport
func NewWebSocketTransport(url string, headers http.Header) *WebSocketTransport { // #nosec G115
	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketTransport{
		url:       url,
		headers:   headers,
		sendCh:    make(chan []byte, 100),
		receiveCh: make(chan []byte, 100),
		errorCh:   make(chan error, 10),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Connect establishes WebSocket connection with auto-reconnect
func (t *WebSocketTransport) Connect(ctx context.Context) error {
	return t.connectWithRetry(ctx, 0)
}

// connectWithRetry attempts to connect with exponential backoff
func (t *WebSocketTransport) connectWithRetry(ctx context.Context, attempt int) error {
	t.connMu.Lock()
	if t.reconnecting && attempt > maxReconnectAttempts {
		t.connMu.Unlock()
		return fmt.Errorf("max reconnection attempts (%d) exceeded", maxReconnectAttempts)
	}
	t.reconnecting = true
	t.connMu.Unlock()

	// Exponential backoff for reconnection
	if attempt > 0 {
		// #nosec G115
		delay := reconnectBaseDelay * time.Duration(1<<uint(attempt-1))
		if delay > reconnectMaxDelay {
			delay = reconnectMaxDelay
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	// Establish WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
	}

	conn, _, err := dialer.DialContext(ctx, t.url, t.headers)
	if err != nil {
		t.lastError = err
		t.lastErrorTime = time.Now()

		// Retry with exponential backoff
		if attempt < maxReconnectAttempts {
			return t.connectWithRetry(ctx, attempt+1)
		}
		return fmt.Errorf("failed to connect to %s: %w", t.url, err)
	}

	// Configure connection
	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait)) // Best effort deadline
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait)) // Best effort deadline
		return nil
	})

	t.connMu.Lock()
	t.conn = conn
	t.connected = true
	t.reconnecting = false
	if attempt > 0 {
		t.reconnectCount++
	}
	t.connMu.Unlock()

	// Start goroutines for read/write/ping
	t.wg.Add(3)
	go t.readPump()
	go t.writePump()
	go t.pingPump()

	// Notify connection established
	if t.onConnect != nil {
		t.onConnect()
	}

	return nil
}

// readPump handles incoming messages
func (t *WebSocketTransport) readPump() {
	defer t.wg.Done()
	defer t.handleDisconnect(nil)

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		t.connMu.RLock()
		conn := t.conn
		t.connMu.RUnlock()

		if conn == nil {
			return
		}

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				t.errorCh <- fmt.Errorf("websocket read error: %w", err)
			}
			return
		}

		if messageType != websocket.TextMessage {
			continue
		}

		t.messagesSent++

		// Send raw bytes to receive channel
		select {
		case t.receiveCh <- data:
			if t.onMessage != nil {
				t.onMessage(data)
			}
		case <-t.ctx.Done():
			return
		}
	}
}

// writePump handles outgoing messages
func (t *WebSocketTransport) writePump() {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return

		case data := <-t.sendCh:
			t.connMu.RLock()
			conn := t.conn
			t.connMu.RUnlock()

			if conn == nil {
				continue
			}

			_ = conn.SetWriteDeadline(time.Now().Add(writeWait)) // Best effort deadline

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				t.errorCh <- fmt.Errorf("write error: %w", err)
				return
			}

			t.messagesReceived++
		}
	}
}

// pingPump sends periodic ping messages
func (t *WebSocketTransport) pingPump() {
	defer t.wg.Done()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return

		case <-ticker.C:
			t.connMu.RLock()
			conn := t.conn
			t.connMu.RUnlock()

			if conn == nil {
				continue
			}

			_ = conn.SetWriteDeadline(time.Now().Add(writeWait)) // Best effort deadline
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send sends a message over the WebSocket
func (t *WebSocketTransport) Send(ctx context.Context, message []byte) error {
	select {
	case t.sendCh <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-t.ctx.Done():
		return fmt.Errorf("transport closed")
	}
}

// Receive receives a message from the WebSocket
func (t *WebSocketTransport) Receive(ctx context.Context) ([]byte, error) {
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

// Close closes the WebSocket connection
func (t *WebSocketTransport) Close() error {
	t.cancel()

	t.connMu.Lock()
	if t.conn != nil {
		// Send close message
		_ = t.conn.SetWriteDeadline(time.Now().Add(writeWait))                                                          // Best effort deadline
		_ = t.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")) // Best effort close
		t.conn.Close()
		t.conn = nil
	}
	t.connected = false
	t.connMu.Unlock()

	// Wait for goroutines to finish
	t.wg.Wait()

	// Close channels
	close(t.sendCh)
	close(t.receiveCh)
	close(t.errorCh)

	return nil
}

// handleDisconnect handles connection loss and triggers reconnection
func (t *WebSocketTransport) handleDisconnect(err error) {
	t.connMu.Lock()
	wasConnected := t.connected
	t.connected = false
	if t.conn != nil {
		t.conn.Close()
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
	go func() {
		if err := t.connectWithRetry(t.ctx, 1); err != nil {
			t.errorCh <- fmt.Errorf("reconnection failed: %w", err)
		}
	}()
}

// IsConnected returns true if the transport is connected
func (t *WebSocketTransport) IsConnected() bool {
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	return t.connected
}

// SetOnConnect sets the connection callback
func (t *WebSocketTransport) SetOnConnect(fn func()) {
	t.onConnect = fn
}

// SetOnDisconnect sets the disconnection callback
func (t *WebSocketTransport) SetOnDisconnect(fn func(error)) {
	t.onDisconnect = fn
}

// SetOnMessage sets the message callback
func (t *WebSocketTransport) SetOnMessage(fn func([]byte)) {
	t.onMessage = fn
}

// GetMetrics returns transport metrics
func (t *WebSocketTransport) GetMetrics() map[string]interface{} {
	t.connMu.RLock()
	defer t.connMu.RUnlock()

	return map[string]interface{}{
		"connected":         t.connected,
		"reconnecting":      t.reconnecting,
		"messages_sent":     t.messagesSent,
		"messages_received": t.messagesReceived,
		"reconnect_count":   t.reconnectCount,
		"last_error":        t.lastError,
		"last_error_time":   t.lastErrorTime,
	}
}
