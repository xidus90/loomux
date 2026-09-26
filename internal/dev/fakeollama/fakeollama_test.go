package fakeollama

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T, text string) *Fixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ollama-fixture.json")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func post(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(body)))
	return w
}

func TestTheFakeAnswersAndLogsWithoutThePrompt(t *testing.T) {
	var log bytes.Buffer
	w := post(t, fixture(t, `{"response":"hallo"}`).Handler(&log), `{"model":"m","prompt":"geheim","stream":false,"think":false,"options":{"temperature":0,"num_ctx":8192}}`)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"response":"hallo"}` {
		t.Fatalf("%d %q", w.Code, w.Body.String())
	}
	if log.String() != "POST /api/generate model=m temperature=0 num_ctx=8192 stream=false think=false\n" {
		t.Fatalf("%q", log.String())
	}
}

func TestTheFakeCanAnswerNonsense(t *testing.T) {
	w := post(t, fixture(t, `{"status":500,"body":"<html>"}`).Handler(&bytes.Buffer{}), `{}`)
	if w.Code != 500 || w.Body.String() != "<html>" {
		t.Fatalf("%d %q", w.Code, w.Body.String())
	}
}

func TestLoadRefusesABrokenFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.json")
	_ = os.WriteFile(path, []byte("{"), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("a broken fixture loaded")
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("a missing fixture loaded")
	}
}

func TestServeEndsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Serve(ctx, "127.0.0.1:0", fixture(t, `{"response":"x"}`), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
}

func TestServeReportsAnAddressItCannotListenOn(t *testing.T) {
	if err := Serve(context.Background(), "127.0.0.1:-1", fixture(t, `{}`), &bytes.Buffer{}); err == nil {
		t.Fatal("listened on port -1")
	}
}

// lockedBuffer is written by the server's goroutine and read by the test's.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Serve cannot tell which port 127.0.0.1:0 got, so the request goes to a port
// the test reserved and released just before. The context ends once the one
// answer is in: ended from the handler, closing the server could cut the
// answer off.
func TestServeAnswersOverHTTPUntilItsContextEnds(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := probe.Addr().String()
	probe.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := &lockedBuffer{}
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, addr, fixture(t, `{"response":"hallo"}`), log) }()
	body := `{"model":"m","prompt":"p","stream":false,"think":false,"options":{"temperature":0.5,"num_ctx":8192}}`
	var resp *http.Response
	for deadline := time.Now().Add(5 * time.Second); ; {
		resp, err = http.Post("http://"+addr+"/api/generate", "application/json", strings.NewReader(body))
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not end with its context")
	}
	if got := log.String(); got != "POST /api/generate model=m temperature=0.5 num_ctx=8192 stream=false think=false\n" {
		t.Fatalf("%q", got)
	}
}
