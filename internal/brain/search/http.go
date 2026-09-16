package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultHost      = "localhost"
	DefaultPort      = 8765
	HandshakeTimeout = 5 * time.Second
	ProbeTimeout     = 500 * time.Millisecond
	QueryTimeout     = 300 * time.Second
)

// Session abstracts one connection to a running search daemon.
type Session interface {
	Call(name string, arguments map[string]any) (map[string]any, error)
	Close() error
}

// HTTPSession implements Session over streamable HTTP MCP transport.
type HTTPSession struct {
	Host         string
	Port         int
	URL          string
	HTTPClient   *http.Client
	PollInterval time.Duration

	mu sync.Mutex
	id int64
}

// NewHTTPSession constructs an HTTPSession for the given port.
func NewHTTPSession(port int) *HTTPSession {
	if port <= 0 {
		port = DefaultPort
	}
	return &HTTPSession{
		Host: DefaultHost,
		Port: port,
		URL:  fmt.Sprintf("http://%s:%d/mcp", DefaultHost, port),
	}
}

// Reachable returns true if the daemon is listening and successfully responds to handshake.
func (s *HTTPSession) Reachable() bool {
	if !s.portOpen(ProbeTimeout) {
		return false
	}
	if err := s.Handshake(); err != nil {
		return false
	}
	return true
}

func (s *HTTPSession) portOpen(timeout time.Duration) bool {
	host := s.Host
	if host == "" {
		host = DefaultHost
	}
	port := s.Port
	if port == 0 {
		port = DefaultPort
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (s *HTTPSession) url() string {
	if s.URL != "" {
		return s.URL
	}
	host := s.Host
	if host == "" {
		host = DefaultHost
	}
	port := s.Port
	if port <= 0 {
		port = DefaultPort
	}
	return fmt.Sprintf("http://%s:%d/mcp", host, port)
}

// WaitUntilReachable polls until Reachable returns true or timeout expires.
func (s *HTTPSession) WaitUntilReachable(timeout time.Duration) error {
	pollInterval := s.PollInterval
	if pollInterval <= 0 {
		pollInterval = 250 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	for {
		if s.Reachable() {
			return nil
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("no qmd daemon answered on %s within %v", s.url(), timeout)
}

// Handshake initializes the MCP session with the daemon.
func (s *HTTPSession) Handshake() error {
	_, err := s.postWithTimeout("initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "brain",
			"version": "0",
		},
	}, HandshakeTimeout)
	return err
}

// Call executes an MCP tool call on the daemon.
func (s *HTTPSession) Call(name string, arguments map[string]any) (map[string]any, error) {
	return s.postWithTimeout("tools/call", map[string]any{
		"name":      name,
		"arguments": arguments,
	}, QueryTimeout)
}

// Close closes the session.
func (s *HTTPSession) Close() error {
	return nil
}

func (s *HTTPSession) postWithTimeout(method string, params map[string]any, timeout time.Duration) (map[string]any, error) {
	s.mu.Lock()
	s.id++
	reqID := s.id
	s.mu.Unlock()

	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      reqID,
		"method":  method,
		"params":  params,
	})

	targetURL := s.url()

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	respBytes, _ := io.ReadAll(resp.Body)
	reply, err := decodeReply(respBytes)
	if err != nil {
		return nil, err
	}

	if rpcErr, ok := reply["error"]; ok && rpcErr != nil {
		return nil, fmt.Errorf("qmd refused %s: %v", method, rpcErr)
	}

	result, ok := reply["result"].(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return result, nil
}

func decodeReply(body []byte) (map[string]any, error) {
	lines := strings.Split(string(body), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimRight(lines[i], "\r")
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			var decoded map[string]any
			if err := json.Unmarshal([]byte(data), &decoded); err != nil {
				continue
			}
			return decoded, nil
		}
	}
	var plain map[string]any
	if err := json.Unmarshal(body, &plain); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return plain, nil
}
