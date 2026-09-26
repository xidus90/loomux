// Package fakeollama stands in for Ollama where the Python reference and
// loomux are recorded and replayed against the same answers. It runs as a
// process of its own, because the reference speaks HTTP to it, and in the
// replay test as a handler on the same fixed port.
package fakeollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// Fixture is the one answer every request gets.
type Fixture struct {
	Status   int     `json:"status"`
	Response *string `json:"response"`
	Body     string  `json:"body"`
}

// Load reads a fixture.
func Load(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &Fixture{}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if f.Status == 0 {
		f.Status = http.StatusOK
	}
	return f, nil
}

type request struct {
	Model   string `json:"model"`
	Stream  bool   `json:"stream"`
	Think   bool   `json:"think"`
	Options struct {
		Temperature float64 `json:"temperature"`
		NumCtx      int     `json:"num_ctx"`
	} `json:"options"`
}

// Handler answers every request from the fixture and writes one line per
// request to log, prompt left out: the prompt carries the case id and the
// day, and the lines must read alike in the recording and the replay.
func (f *Fixture) Handler(log io.Writer) http.Handler {
	var mu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got request
		_ = json.NewDecoder(r.Body).Decode(&got)
		mu.Lock()
		fmt.Fprintf(log, "%s %s model=%s temperature=%s num_ctx=%d stream=%t think=%t\n",
			r.Method, r.URL.Path, got.Model, strconv.FormatFloat(got.Options.Temperature, 'g', -1, 64),
			got.Options.NumCtx, got.Stream, got.Think)
		mu.Unlock()
		w.WriteHeader(f.Status)
		if f.Response != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"response": *f.Response})
			return
		}
		_, _ = w.Write([]byte(f.Body))
	})
}

// Serve listens on addr until ctx ends.
func Serve(ctx context.Context, addr string, f *Fixture, log io.Writer) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: f.Handler(log)}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	// Serve ends only with an error; the one the closing above causes is the
	// regular end. Any other is a listener that broke, which no test provokes,
	// so it shares the return with the regular end instead of a branch.
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return err
}
