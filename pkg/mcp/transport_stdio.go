package mcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// StdioTransport implements Transport using stdio (stdin/stdout)
// This is the most common transport for MCP servers launched as subprocesses
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	reader *bufio.Reader
	mu     sync.Mutex
	closed bool
}

// NewStdioTransport creates a new stdio transport by launching an MCP server process
func NewStdioTransport(ctx context.Context, command string, args ...string) (*StdioTransport, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	// Get pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start MCP server process: %w", err)
	}

	transport := &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		reader: bufio.NewReader(stdout),
		closed: false,
	}

	// Start stderr reader (log errors)
	go transport.readStderr()

	return transport, nil
}

// Send sends a message to the MCP server via stdin
func (t *StdioTransport) Send(ctx context.Context, message []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	// Write message with newline delimiter
	if _, err := t.stdin.Write(message); err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}
	if _, err := t.stdin.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// Receive receives a message from the MCP server via stdout
func (t *StdioTransport) Receive(ctx context.Context) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, fmt.Errorf("transport is closed")
	}

	// Read line (JSON-RPC messages are newline-delimited)
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdout: %w", err)
	}

	return line, nil
}

// Close terminates the MCP server process and closes all pipes
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true

	// Close stdin first to signal server to shutdown
	if err := t.stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited
		t.stdout.Close()
		t.stderr.Close()
		return err
	case <-context.Background().Done():
		// Timeout - force kill
		t.cmd.Process.Kill()
		t.stdout.Close()
		t.stderr.Close()
		return fmt.Errorf("MCP server process did not exit gracefully")
	}
}

// readStderr reads stderr output from MCP server (for logging)
func (t *StdioTransport) readStderr() {
	reader := bufio.NewReader(t.stderr)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				// Log error reading stderr
				fmt.Printf("MCP stderr error: %v\n", err)
			}
			return
		}
		// Log stderr output (TODO: integrate with proper logging)
		fmt.Printf("MCP server stderr: %s", line)
	}
}
