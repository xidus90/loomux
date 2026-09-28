package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// tagsServer answers /api/tags with status and body.
func tagsServer(t *testing.T, status int, body string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/tags" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func clientAt(t *testing.T, endpoint string) *Client {
	t.Helper()
	client, err := NewClient(settingsFor(endpoint))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

const tags = `{"models":[` +
	`{"name":"gemma3:latest","model":"gemma3:latest"},` +
	`{"name":"qwen","model":"qwen:7b"},` +
	`{"name":"hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL","model":"hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL"},` +
	`{"name":"localhost:5000/team/model","model":"localhost:5000/team/model"}]}`

func TestHasFindsTheModelByNameOrModel(t *testing.T) {
	client := clientAt(t, tagsServer(t, 200, tags)+"/")
	for name, want := range map[string]bool{
		"gemma3":        true, // Ollama adds :latest
		"gemma3:latest": true,
		"gemma3:2b":     false,
		"qwen:7b":       true, // the model field
		"qwen":          true, // the name field
		"hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL": true,
		"hf.co/unsloth/gemma-4-E4B-it-qat-GGUF":            false, // a tag must match exactly
		"hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q8":      false,
		"localhost:5000/team/model:latest":                 true, // a port is no tag
		"Gemma3":                                           true, // Ollama compares in any case
		"QWEN:7B":                                          true,
		"llama":                                            false,
	} {
		got, err := client.Has(context.Background(), name)
		if err != nil || got != want {
			t.Errorf("%q: %v %v, want %v", name, got, err, want)
		}
	}
}

func TestHasCallsAnUnreadableAnswerAnError(t *testing.T) {
	for name, endpoint := range map[string]string{
		"500":        tagsServer(t, 500, tags),
		"not json":   tagsServer(t, 200, "<html>"),
		"no models":  tagsServer(t, 200, `{"models":3}`),
		"redirected": tagsServer(t, 302, ""),
	} {
		got, err := clientAt(t, endpoint).Has(context.Background(), "gemma3")
		if err == nil || got || errors.Is(err, ErrUnreachable) {
			t.Errorf("%s: %v %v", name, got, err)
		}
	}
}

// Only a connection that fails is an Ollama that is not running.
func TestHasCallsAClosedPortUnreachable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	endpoint := server.URL
	server.Close()
	if got, err := clientAt(t, endpoint).Has(context.Background(), "gemma3"); !errors.Is(err, ErrUnreachable) || got {
		t.Fatalf("%v %v", got, err)
	}
}

// pullServer streams lines as NDJSON after status and records the request.
func pullServer(t *testing.T, status int, lines ...string) (string, *map[string]any) {
	t.Helper()
	got := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/pull" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(status)
		for _, line := range lines {
			_, _ = fmt.Fprintln(w, line)
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(server.Close)
	return server.URL, &got
}

type step struct {
	status           string
	completed, total int64
}

func TestPullReportsEveryLineAndEndsAtSuccess(t *testing.T) {
	endpoint, request := pullServer(t, 200,
		`{"status":"pulling manifest"}`,
		`{"status":"pulling abc","digest":"sha256:abc","total":100,"completed":40}`,
		``,
		`{"status":"pulling abc","digest":"sha256:abc","total":100,"completed":100}`,
		`{"status":"verifying sha256 digest"}`,
		`{"status":"success"}`,
	)
	var steps []step
	err := clientAt(t, endpoint).Pull(context.Background(), "gemma3", func(status string, completed, total int64) {
		steps = append(steps, step{status, completed, total})
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []step{{"pulling manifest", 0, 0}, {"pulling abc", 40, 100}, {"pulling abc", 100, 100},
		{"verifying sha256 digest", 0, 0}, {"success", 0, 0}}
	if fmt.Sprint(steps) != fmt.Sprint(want) {
		t.Errorf("steps = %v", steps)
	}
	if (*request)["model"] != "gemma3" || (*request)["stream"] != true || len(*request) != 2 {
		t.Errorf("request = %v", *request)
	}
}

func TestPullFailsWithoutSuccess(t *testing.T) {
	for name, want := range map[string]struct {
		status int
		lines  []string
		err    string
	}{
		"error line":  {200, []string{`{"status":"pulling manifest"}`, `{"error":"pull model manifest: file does not exist"}`}, "file does not exist"},
		"cut short":   {200, []string{`{"status":"pulling manifest"}`}, "ended without success"},
		"not json":    {200, []string{`<html>`}, "unreadably"},
		"not 2xx":     {404, []string{`{"error":"pull model manifest: file does not exist"}`}, "404 Not Found: pull model manifest: file does not exist"},
		"bare 500":    {500, []string{`<html>`}, "answered 500 Internal Server Error"},
		"redirected":  {302, nil, "302"},
		"empty error": {200, []string{`{"error":""}`}, "ollama pull failed"},
	} {
		endpoint, _ := pullServer(t, want.status, want.lines...)
		err := clientAt(t, endpoint).Pull(context.Background(), "gemma3", func(string, int64, int64) {})
		if err == nil || !strings.Contains(err.Error(), want.err) {
			t.Errorf("%s: %v, want %q", name, err, want.err)
		}
	}
}

// switchingServer answers every request with 101, the one status below 200
// that Go's client hands back as an answer, and hangs up.
func switchingServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		_, _ = buf.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: x\r\n\r\n")
		_ = buf.Flush()
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// 2xx is the whole range: 299 is an answer, 101 below it is not.
func TestOnlyA2xxStatusIsAnAnswer(t *testing.T) {
	if got, err := clientAt(t, tagsServer(t, 299, tags)).Has(context.Background(), "gemma3"); err != nil || !got {
		t.Errorf("Has at 299: %v %v", got, err)
	}
	endpoint, _ := pullServer(t, 299, `{"status":"success"}`)
	if err := clientAt(t, endpoint).Pull(context.Background(), "gemma3", func(string, int64, int64) {}); err != nil {
		t.Errorf("Pull at 299: %v", err)
	}
	switching := switchingServer(t)
	if _, err := clientAt(t, switching).Has(context.Background(), "gemma3"); err == nil || !strings.Contains(err.Error(), "GET /api/tags answered 101") {
		t.Errorf("Has at 101: %v", err)
	}
	if err := clientAt(t, switching).Pull(context.Background(), "gemma3", func(string, int64, int64) {}); err == nil || !strings.Contains(err.Error(), "POST /api/pull answered 101") {
		t.Errorf("Pull at 101: %v", err)
	}
}

// A JSON body without an error says nothing either: no colon, no empty
// reason after the status.
func TestAPullRefusalWithAnEmptyErrorNamesTheStatusOnly(t *testing.T) {
	endpoint, _ := pullServer(t, 500, `{}`)
	err := clientAt(t, endpoint).Pull(context.Background(), "gemma3", func(string, int64, int64) {})
	if err == nil || err.Error() != "POST /api/pull answered 500 Internal Server Error" {
		t.Fatalf("%v", err)
	}
}

func TestPullFailsAtAClosedPort(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	endpoint := server.URL
	server.Close()
	if err := clientAt(t, endpoint).Pull(context.Background(), "gemma3", func(string, int64, int64) {}); err == nil {
		t.Fatal("a closed port pulled")
	}
}

func TestPullStopsWhenTheContextEnds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The server notices a closed connection only once the body is read.
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = fmt.Fprintln(w, `{"status":"pulling abc","total":100,"completed":1}`)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	err := clientAt(t, server.URL).Pull(ctx, "gemma3", func(string, int64, int64) { cancel() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// A download of several GB takes longer than any total limit Ask may keep.
func TestPullHasNoTotalTimeout(t *testing.T) {
	client := clientAt(t, "http://127.0.0.1:11434")
	if client.pull.Timeout != 0 || client.pull.Transport != client.http.Transport || client.pull.CheckRedirect == nil {
		t.Fatalf("pull client = %+v", client.pull)
	}
}
