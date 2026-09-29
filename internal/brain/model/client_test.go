package model

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/config"
)

func settingsFor(endpoint string) config.ModelSettings {
	return config.ModelSettings{Enabled: true, Endpoint: endpoint, Name: "m", Temperature: 0.2, Roles: map[string]bool{"propose": true}}
}

// isWarmUp tells the warm-up from a question: only the warm-up caps the
// answer with num_predict.
func isWarmUp(body map[string]any) bool {
	options, _ := body["options"].(map[string]any)
	_, capped := options["num_predict"]
	return capped
}

// afterWarm answers the client's first request, its warm-up, and hands every
// later one, the questions, to h: a test of how a question fails must not
// fail at the warm-up instead.
func afterWarm(h http.HandlerFunc) http.HandlerFunc {
	var seen atomic.Int32
	return func(w http.ResponseWriter, r *http.Request) {
		if seen.Add(1) == 1 {
			_, _ = io.Copy(io.Discard, r.Body)
			_, _ = w.Write([]byte(`{"response":""}`))
			return
		}
		h(w, r)
	}
}

func TestGuardEndpointRefusesAnythingOffTheLoopback(t *testing.T) {
	for endpoint, want := range map[string]string{
		" http://localhost:11434":               "spaces or control characters",
		"http://local\thost:1":                  "spaces or control characters",
		"http://localhost:1/\x7f":               "spaces or control characters",
		"ftp://localhost/x":                     "must use http or https",
		"//127.0.0.1:11434":                     "must use http or https",
		"http://localhost:notaport":             "port must be a number",
		"http://localhost:70000":                "port must be a number",
		"http://localhost:99999999999999999999": "port must be a number",
		"http://192.0.2.1:11434":                "must stay on the loopback",
		"http://[::1]:11434@evil.example.com":   "is not a readable address",
		"http://127.0.0.2:11434":                "must stay on the loopback",
	} {
		err := GuardEndpoint(endpoint)
		if err == nil || !strings.Contains(err.Error(), want) || !strings.HasPrefix(err.Error(), "[model] endpoint") {
			t.Errorf("%q: %v, want %q", endpoint, err, want)
		}
	}
}

// Review Focus 2: Python reads the host lower-cased.
func TestGuardEndpointLetsTheLoopbackThrough(t *testing.T) {
	for _, endpoint := range []string{"http://127.0.0.1:11434", "https://localhost", "http://[::1]:11434/", "http://LOCALHOST:11434", "http://localhost:65535"} {
		if err := GuardEndpoint(endpoint); err != nil {
			t.Errorf("%q: %v", endpoint, err)
		}
	}
}

func TestAskPostsTheReferencesPayload(t *testing.T) {
	var got map[string]any
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"response":"answer"}`))
	}))
	defer server.Close()
	client, err := NewClient(settingsFor(server.URL + "//"))
	if err != nil {
		t.Fatal(err)
	}
	text, ok := client.Ask(context.Background(), "frage")
	if !ok || text != "answer" || path != "POST /api/generate" {
		t.Fatalf("%q %v %s", text, ok, path)
	}
	options, _ := got["options"].(map[string]any)
	if got["model"] != "m" || got["prompt"] != "frage" || got["stream"] != false || got["think"] != false ||
		options["temperature"] != 0.2 || options["num_ctx"] != float64(8192) || len(got) != 5 {
		t.Fatalf("%v", got)
	}
}

func TestAskFormatSendsTheSchemaAndAskSendsNone(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		bodies = append(bodies, got)
		_ = json.NewEncoder(w).Encode(map[string]string{"response": "x"})
	}))
	defer server.Close()
	client, err := NewClient(settingsFor(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	client.Ask(context.Background(), "p")
	client.AskFormat(context.Background(), "p", map[string]any{"type": "object"})
	if len(bodies) != 3 {
		t.Fatalf("%d requests, want the warm-up and two questions", len(bodies))
	}
	if _, present := bodies[1]["format"]; present {
		t.Fatal("Ask sent a format")
	}
	if format, _ := bodies[2]["format"].(map[string]any); format["type"] != "object" {
		t.Fatalf("AskFormat sent %v", bodies[2]["format"])
	}
}

// Ollama loads a model on the first request that names it, and reloads it
// when num_ctx differs from how it was loaded; that took 28-30 s on the first
// question and ran into the question's limit. So the first question is
// preceded by a request that loads the model with the questions' own options
// and asks for a single token.
func TestTheFirstQuestionWarmsTheModelFirst(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		bodies = append(bodies, got)
		_, _ = w.Write([]byte(`{"response":"answer"}`))
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if text, ok := client.Ask(context.Background(), "frage"); !ok || text != "answer" {
		t.Fatalf("%q %v", text, ok)
	}
	if len(bodies) != 2 {
		t.Fatalf("%d requests, want the warm-up and the question", len(bodies))
	}
	warm, question := bodies[0], bodies[1]
	options, _ := warm["options"].(map[string]any)
	if warm["model"] != "m" || warm["stream"] != false || warm["think"] != false || warm["prompt"] == "frage" ||
		options["temperature"] != 0.2 || options["num_ctx"] != float64(8192) || options["num_predict"] != float64(1) ||
		len(options) != 3 || len(warm) != 5 {
		t.Fatalf("warm-up %v", warm)
	}
	if prompt, _ := warm["prompt"].(string); prompt == "" || len(prompt) > 40 {
		t.Fatalf("warm-up prompt %q", prompt)
	}
	if isWarmUp(question) || question["prompt"] != "frage" {
		t.Fatalf("question %v", question)
	}
}

func TestASecondQuestionSendsNoSecondWarmUp(t *testing.T) {
	var warmUps, questions atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if isWarmUp(got) {
			warmUps.Add(1)
		} else {
			questions.Add(1)
		}
		_, _ = w.Write([]byte(`{"response":"x"}`))
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	client.Ask(context.Background(), "p")
	client.AskFormat(context.Background(), "p", map[string]any{"type": "object"})
	if warmUps.Load() != 1 || questions.Load() != 2 {
		t.Fatalf("%d warm-ups, %d questions", warmUps.Load(), questions.Load())
	}
}

// Loading can outlast the question's limit; the warm-up has none of its own.
// As Ollama does, the server spends the load on whichever request comes
// first.
func TestAWarmUpLongerThanTheQuestionLimitDoesNotFailTheQuestion(t *testing.T) {
	var seen atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if seen.Add(1) == 1 {
			time.Sleep(time.Second)
		}
		_, _ = w.Write([]byte(`{"response":"answer"}`))
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	client.http.Timeout = 300 * time.Millisecond
	if text, ok := client.Ask(context.Background(), "p"); !ok || text != "answer" {
		t.Fatalf("%q %v", text, ok)
	}
}

// A failed warm-up is an outage: the question is not sent, and it is not
// tried again on this client, so a dead Ollama costs one attempt per client
// rather than one per question.
func TestAFailedWarmUpSendsNoQuestionAndIsNotRetried(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"response":"x"}`))
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	for range 2 {
		if text, ok := client.Ask(context.Background(), "p"); ok || text != "" {
			t.Fatalf("%q %v", text, ok)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("%d requests, want the one warm-up", requests.Load())
	}
}

// Without a limit of its own the warm-up ends with the caller's context.
func TestTheCallersContextEndsAWarmUp(t *testing.T) {
	var questions atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if !isWarmUp(got) {
			questions.Add(1)
			_, _ = w.Write([]byte(`{"response":"x"}`))
			return
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, ok := client.Ask(ctx, "p"); ok || questions.Load() != 0 {
		t.Fatalf("answered %v after %d questions", ok, questions.Load())
	}
}

func TestAskCountsEveryOutageAsNoAnswer(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"500":         func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) },
		"not json":    func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) },
		"not object":  func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`["x"]`)) },
		"no response": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"done":true}`)) },
		"not string":  func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"response":3}`)) },
	} {
		server := httptest.NewServer(afterWarm(handler))
		client, _ := NewClient(settingsFor(server.URL))
		if text, ok := client.Ask(context.Background(), "p"); ok || text != "" {
			t.Errorf("%s: %q %v", name, text, ok)
		}
		server.Close()
	}
}

// The status alone decides: each refused status carries a body that would
// otherwise be an answer, and 299 is still 2xx, as urllib counts it.
func TestTheStatusDecidesAgainstAnAnswerInTheBody(t *testing.T) {
	for status, want := range map[int]bool{200: true, 299: true, 300: false, 500: false} {
		server := httptest.NewServer(afterWarm(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"response":"x"}`))
		}))
		client, _ := NewClient(settingsFor(server.URL))
		if _, ok := client.Ask(context.Background(), "p"); ok != want {
			t.Errorf("%d: answered %v, want %v", status, ok, want)
		}
		server.Close()
	}
}

// A 101 is the one status below 200 Go's client hands back as final; its
// body is the connection, which here carries an answer.
func TestASwitchOfProtocolsIsNoAnswer(t *testing.T) {
	server := httptest.NewServer(afterWarm(func(w http.ResponseWriter, _ *http.Request) {
		conn, _, _ := w.(http.Hijacker).Hijack()
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: x\r\n\r\n{\"response\":\"x\"}"))
		_ = conn.Close()
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if _, ok := client.Ask(context.Background(), "p"); ok {
		t.Fatal("a 101 answered")
	}
}

func TestAnEmptyResponseIsAnAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"response":""}`))
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if text, ok := client.Ask(context.Background(), "p"); !ok || text != "" {
		t.Fatalf("%q %v", text, ok)
	}
}

// Review Focus 3.
func TestARedirectIsNoAnswerAndNoSecondRequest(t *testing.T) {
	var foreign atomic.Int32
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		foreign.Add(1)
		_, _ = w.Write([]byte(`{"response":"leaked"}`))
	}))
	defer elsewhere.Close()
	server := httptest.NewServer(afterWarm(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/api/generate", http.StatusFound)
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if _, ok := client.Ask(context.Background(), "p"); ok || foreign.Load() != 0 {
		t.Fatalf("ok %v, foreign %d", ok, foreign.Load())
	}
}

// Go's ProxyFromEnvironment never proxies a loopback address anyway, so a
// request through a proxy server would pass with the default transport as
// well; what is held here is the transport itself.
func TestTheClientTakesNoProxyFromTheEnvironment(t *testing.T) {
	client, _ := NewClient(settingsFor("http://127.0.0.1:1"))
	if transport, ok := client.http.Transport.(*http.Transport); !ok || transport.Proxy != nil {
		t.Fatal("the transport may take a proxy")
	}
}

// serve builds a client per area and pass; a kept-alive socket per client
// would pile up for the life of the process.
func TestTheClientKeepsNoIdleConnection(t *testing.T) {
	client, _ := NewClient(settingsFor("http://127.0.0.1:1"))
	if transport, ok := client.http.Transport.(*http.Transport); !ok || !transport.DisableKeepAlives {
		t.Fatal("the transport keeps idle connections")
	}
}

func TestABodyCutShortIsNoAnswer(t *testing.T) {
	server := httptest.NewServer(afterWarm(func(w http.ResponseWriter, _ *http.Request) {
		// Raw on the hijacked connection: a body that promises 100 bytes
		// and ends after a whole JSON object, so only the short read can
		// refuse it.
		conn, _, _ := w.(http.Hijacker).Hijack()
		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{\"response\":\"x\"}"))
		_ = conn.Close()
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if _, ok := client.Ask(context.Background(), "p"); ok {
		t.Fatal("half a body answered")
	}
}

// The warm-up leaves the redirect rule as it found it: a 302 is a failed
// warm-up, and the address it names is never asked.
func TestARedirectedWarmUpFollowsNothing(t *testing.T) {
	var foreign atomic.Int32
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		foreign.Add(1)
		_, _ = w.Write([]byte(`{"response":"leaked"}`))
	}))
	defer elsewhere.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/api/generate", http.StatusFound)
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if _, ok := client.Ask(context.Background(), "p"); ok || foreign.Load() != 0 {
		t.Fatalf("ok %v, foreign %d", ok, foreign.Load())
	}
}

func TestAskGivesUpAfterItsTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(afterWarm(func(w http.ResponseWriter, _ *http.Request) { <-release }))
	defer server.Close()
	defer close(release)
	client, _ := NewClient(settingsFor(server.URL))
	client.http.Timeout = 50 * time.Millisecond
	if _, ok := client.Ask(context.Background(), "p"); ok {
		t.Fatal("a hanging server answered")
	}
}

func TestNewClientUsesTheReferencesTimeouts(t *testing.T) {
	client, _ := NewClient(settingsFor("http://127.0.0.1:1"))
	if client.http.Timeout != 30*time.Second || connectTimeout != 2*time.Second {
		t.Fatal(client.http.Timeout, connectTimeout)
	}
	if _, err := NewClient(settingsFor("http://192.0.2.1")); err == nil {
		t.Fatal("a client was built for a foreign host")
	}
}

func TestANobodyListeningIsNoAnswer(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	endpoint := server.URL
	server.Close()
	client, _ := NewClient(settingsFor(endpoint))
	if _, ok := client.Ask(context.Background(), "p"); ok {
		t.Fatal("a closed port answered")
	}
}
