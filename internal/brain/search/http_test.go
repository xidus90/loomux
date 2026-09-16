package search_test

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/search"
)

func TestHTTPSession_PlainJSON(t *testing.T) {
	var capturedMethod string
	var capturedParams map[string]any
	var capturedContentType string
	var capturedAccept string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		capturedAccept = r.Header.Get("Accept")

		body, _ := io.ReadAll(r.Body)
		var rpcReq struct {
			Jsonrpc string         `json:"jsonrpc"`
			ID      int64          `json:"id"`
			Method  string         `json:"method"`
			Params  map[string]any `json:"params"`
		}
		_ = json.Unmarshal(body, &rpcReq)
		capturedMethod = rpcReq.Method
		capturedParams = rpcReq.Params

		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      rpcReq.ID,
			"result": map[string]any{
				"structuredContent": map[string]any{
					"results": []any{
						map[string]any{"file": "vault/doc.md", "score": 0.9},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	session := &search.HTTPSession{
		URL: ts.URL,
	}

	res, err := session.Call("query", map[string]any{"query": "hello"})
	if err != nil {
		t.Fatalf("unexpected Call error: %v", err)
	}

	if capturedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", capturedContentType)
	}
	if !strings.Contains(capturedAccept, "application/json") || !strings.Contains(capturedAccept, "text/event-stream") {
		t.Errorf("expected Accept application/json, text/event-stream, got %q", capturedAccept)
	}
	if capturedMethod != "tools/call" {
		t.Errorf("expected method 'tools/call', got %q", capturedMethod)
	}
	if capturedParams["name"] != "query" {
		t.Errorf("expected param name 'query', got %v", capturedParams["name"])
	}

	sc, ok := res["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent in result, got: %v", res)
	}
	results, ok := sc["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 result in structuredContent, got: %v", sc)
	}
}

func TestHTTPSession_SSEStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\r\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"answer\":\"streamed-ok\"}}\r\n\r\n"))
	}))
	defer ts.Close()

	session := &search.HTTPSession{
		URL: ts.URL,
	}

	res, err := session.Call("test", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected Call error with SSE: %v", err)
	}
	if res["answer"] != "streamed-ok" {
		t.Errorf("expected answer 'streamed-ok', got %v", res["answer"])
	}
}

func TestHTTPSession_Handshake(t *testing.T) {
	var capturedMethod string
	var capturedParams map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var rpcReq struct {
			Jsonrpc string         `json:"jsonrpc"`
			ID      int64          `json:"id"`
			Method  string         `json:"method"`
			Params  map[string]any `json:"params"`
		}
		_ = json.Unmarshal(body, &rpcReq)
		capturedMethod = rpcReq.Method
		capturedParams = rpcReq.Params

		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      rpcReq.ID,
			"result": map[string]any{
				"capabilities": map[string]any{},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	session := &search.HTTPSession{
		URL: ts.URL,
	}

	err := session.Handshake()
	if err != nil {
		t.Fatalf("unexpected Handshake error: %v", err)
	}
	if capturedMethod != "initialize" {
		t.Errorf("expected method 'initialize', got %q", capturedMethod)
	}
	if capturedParams["protocolVersion"] != "2025-06-18" {
		t.Errorf("expected protocolVersion '2025-06-18', got %v", capturedParams["protocolVersion"])
	}
	clientInfo, ok := capturedParams["clientInfo"].(map[string]any)
	if !ok || clientInfo["name"] != "brain" {
		t.Errorf("expected clientInfo name 'brain', got %v", clientInfo)
	}
}

func TestHTTPSession_ErrorEnvelope(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]any{
				"code":    -32600,
				"message": "Invalid Request",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	session := &search.HTTPSession{
		URL: ts.URL,
	}

	_, err := session.Call("bad_tool", nil)
	if err == nil {
		t.Fatal("expected error when qmd returns JSON-RPC error, got nil")
	}
	if !strings.Contains(err.Error(), "qmd refused tools/call") {
		t.Errorf("expected 'qmd refused' error, got: %v", err)
	}
}

func TestHTTPSession_HTTPStatusError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal daemon error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	session := &search.HTTPSession{
		URL: ts.URL,
	}

	_, err := session.Call("tool", nil)
	if err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected 'status 500' error, got: %v", err)
	}
}

func TestHTTPSession_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json at all"))
	}))
	defer ts.Close()

	session := &search.HTTPSession{
		URL: ts.URL,
	}

	_, err := session.Call("tool", nil)
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
}

func TestHTTPSession_Reachability(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]any{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	session := &search.HTTPSession{
		Host:         host,
		Port:         port,
		URL:          ts.URL,
		PollInterval: 10 * time.Millisecond,
	}

	if !session.Reachable() {
		t.Errorf("expected session to be reachable when server is up")
	}

	if err := session.WaitUntilReachable(time.Second); err != nil {
		t.Errorf("unexpected error from WaitUntilReachable: %v", err)
	}

	// Close server to make port unreachable
	ts.Close()

	if session.Reachable() {
		t.Errorf("expected session to be unreachable after server closed")
	}

	err = session.WaitUntilReachable(30 * time.Millisecond)
	if err == nil {
		t.Errorf("expected error from WaitUntilReachable on closed port, got nil")
	}
}

func TestHTTPSession_Reachable_HandshakeFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "handshake rejected", http.StatusForbidden)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	session := &search.HTTPSession{
		Host: host,
		Port: port,
		URL:  ts.URL,
	}

	if session.Reachable() {
		t.Errorf("expected session to be unreachable when handshake fails")
	}
}

func TestHTTPSession_Close(t *testing.T) {
	session := search.NewHTTPSession(0)
	if err := session.Close(); err != nil {
		t.Errorf("unexpected error on Close: %v", err)
	}
}

func TestHTTPSession_DefaultSession(t *testing.T) {
	session := &search.HTTPSession{}
	// Empty host and port will default to localhost:8765
	if session.Reachable() {
		t.Log("daemon happened to be reachable on localhost:8765")
	}
	// Wait until reachable with 1ms deadline and 0 poll interval
	err := session.WaitUntilReachable(1 * time.Millisecond)
	if err == nil {
		t.Log("daemon happened to answer within 1ms")
	}
}

func TestHTTPSession_NonMapResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "not-a-map",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	session := &search.HTTPSession{
		URL: ts.URL,
	}
	res, err := session.Call("tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected empty map for non-map result, got %v", res)
	}
}

func TestHTTPSession_ClientDoError(t *testing.T) {
	session := &search.HTTPSession{
		URL: "http://127.0.0.1:1/mcp",
	}
	_, err := session.Call("tool", nil)
	if err == nil {
		t.Fatal("expected client error on unreachable port, got nil")
	}
}

func TestHTTPSession_InvalidURL(t *testing.T) {
	session := &search.HTTPSession{
		URL: "http://invalid\x7furl",
	}
	_, err := session.Call("tool", nil)
	if err == nil {
		t.Fatal("expected error with control char in URL, got nil")
	}
}

func TestHTTPSession_SSEWithEmptyOrInvalidLines(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\r\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"hello\":\"world\"}}\r\ndata: {invalid json\r\ndata:\r\n\r\n"))
	}))
	defer ts.Close()

	session := &search.HTTPSession{
		URL: ts.URL,
	}

	res, err := session.Call("test", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["hello"] != "world" {
		t.Errorf("expected hello: world, got: %v", res)
	}
}
