// Package model is the local model: a client that never leaves the loopback,
// the gate in front of it, the roles `propose`, `describe` and `place`, and
// the judges a sentence of `describe` has to pass. It follows
// `src/brain/model/` of the reference.
//
// Two kinds of failure are kept strictly apart. A misconfiguration -- an
// address off the loopback -- is an error, because a fallback would hide it
// and the next run would speak outwards again. An outage -- Ollama down, too
// slow, answering nonsense -- is no answer, and no answer means a manual case
// everywhere, never the cloud.
package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// Connect short, read long: 2 s tell "Ollama is not running" from "Ollama is
// computing", and 30 s carry one question to a loaded model. Loading is not
// in them: the warm-up before a client's first question takes it, with no
// limit of its own (28-30 s were measured for loading plus the first question
// against Ollama 0.34 on CUDA, where the reference measured 6.1 s at worst).
const (
	connectTimeout = 2 * time.Second
	totalTimeout   = 30 * time.Second
	numCtx         = 8192
)

// warmPrompt is what the warm-up asks; its one token of answer is dropped.
const warmPrompt = "Reply with OK."

// GuardEndpoint refuses every address that is not the loopback, in the
// reference's order (`client.py:59-109`). The host is compared as a string,
// lower-cased as Python's `hostname` reads it; nothing is resolved.
func GuardEndpoint(endpoint string) error {
	for _, r := range endpoint {
		if r <= ' ' || r == 0x7f {
			return fmt.Errorf("[model] endpoint must not contain spaces or control characters, found %s", pytext.Repr(endpoint))
		}
	}
	parts, err := url.Parse(endpoint)
	if err != nil {
		if strings.Contains(err.Error(), "invalid port") {
			return fmt.Errorf("[model] endpoint port must be a number, found %s", pytext.Repr(endpoint))
		}
		return fmt.Errorf("[model] endpoint is not a readable address, found %s", pytext.Repr(endpoint))
	}
	if parts.Scheme != "http" && parts.Scheme != "https" {
		return fmt.Errorf("[model] endpoint must use http or https, found %s", pytext.Repr(endpoint))
	}
	// url.Parse takes any run of digits; Python's `port` raises outside
	// 0..65535, and that is a misconfiguration, not an outage.
	if port := parts.Port(); port != "" {
		if n, err := strconv.Atoi(port); err != nil || n > 65535 {
			return fmt.Errorf("[model] endpoint port must be a number, found %s", pytext.Repr(endpoint))
		}
	}
	switch strings.ToLower(parts.Hostname()) {
	case "127.0.0.1", "localhost", "::1":
		return nil
	}
	return fmt.Errorf("[model] endpoint must stay on the loopback (127.0.0.1, ::1, localhost), found %s", pytext.Repr(endpoint))
}

// Client asks one Ollama on the loopback.
type Client struct {
	settings config.ModelSettings
	http     *http.Client
	// pull shares http's transport and redirect rule but has no total
	// timeout: a download of several GB outlasts any limit a question keeps,
	// and so may loading the model, which the warm-up does through it.
	pull *http.Client
	base string // the endpoint without its trailing slashes
	url  string
	// warm runs the warm-up once per client; ready is its outcome, kept for
	// the client's life.
	warm  sync.Once
	ready bool
}

// NewClient guards the endpoint and builds the client. No proxy is taken
// from the environment -- the request would go through a foreign machine
// while the guard saw only the entered string -- and no redirect is
// followed: the guard checks the entered address, not one a 302 names.
// No connection is kept alive: serve builds a client per area and pass, and
// each client's idle socket would stay open for the life of the process.
func NewClient(s config.ModelSettings) (*Client, error) {
	if err := GuardEndpoint(s.Endpoint); err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy:             nil,
		DialContext:       (&net.Dialer{Timeout: connectTimeout}).DialContext,
		DisableKeepAlives: true,
	}
	noRedirect := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	base := strings.TrimRight(s.Endpoint, "/")
	return &Client{
		settings: s,
		base:     base,
		url:      base + "/api/generate",
		http:     &http.Client{Transport: transport, Timeout: totalTimeout, CheckRedirect: noRedirect},
		pull:     &http.Client{Transport: transport, CheckRedirect: noRedirect},
	}, nil
}

type generateRequest struct {
	Model   string          `json:"model"`
	Prompt  string          `json:"prompt"`
	Stream  bool            `json:"stream"`
	Think   bool            `json:"think"`
	Options generateOptions `json:"options"`
	// Format is the output schema the endpoint enforces (`place` only); it
	// stands behind options, where the reference's payload appends it.
	Format any `json:"format,omitempty"`
}

type generateOptions struct {
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
	// NumPredict caps the answer; only the warm-up sets it, so a question's
	// body stays as the reference sends it.
	NumPredict *int `json:"num_predict,omitempty"`
}

// Ask is AskFormat without a schema.
func (c *Client) Ask(ctx context.Context, prompt string) (string, bool) {
	return c.AskFormat(ctx, prompt, nil)
}

// AskFormat is one question to the model; a format other than nil goes out
// as the schema the endpoint holds the answer to. false stands for every
// outage: no connection, a timeout, a status that is not 2xx (a redirect
// included), a body that is not a JSON object with a string `response`, and
// a warm-up that failed, on this call or an earlier one of the same client.
// An empty string is an answer; the role judges it.
func (c *Client) AskFormat(ctx context.Context, prompt string, format any) (string, bool) {
	c.warm.Do(func() { c.ready = c.warmUp(ctx) })
	if !c.ready {
		return "", false
	}
	// A struct of strings, bools, numbers and a schema of maps and slices
	// always marshals, and the URL passed the guard, so neither of the next
	// two calls can fail.
	body, _ := json.Marshal(generateRequest{
		Model: c.settings.Name, Prompt: prompt,
		Options: generateOptions{Temperature: c.settings.Temperature, NumCtx: numCtx},
		Format:  format,
	})
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return "", false
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", false
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", false
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", false
	}
	object, _ := decoded.(map[string]any)
	text, ok := object["response"].(string)
	return text, ok
}

// warmUp loads the model with the options the questions use and has it
// answer one token, so that the first question meets a loaded model. Ollama
// loads a model on the first request that names it and loads it again when
// num_ctx differs from the loaded one, so the warm-up names both.
//
// It has no total limit -- it goes through pull, like a download -- but the
// same loopback transport and redirect rule, and ends with ctx. Only a 2xx
// status is a warm model; the body is not read for an answer. A failure is
// not retried by the client: the run that asked goes on without the model,
// and an Ollama that is down costs one attempt per client, not one per
// question.
func (c *Client) warmUp(ctx context.Context) bool {
	one := 1
	// As in AskFormat, neither the body nor the request can fail to build.
	body, _ := json.Marshal(generateRequest{
		Model: c.settings.Name, Prompt: warmPrompt,
		Options: generateOptions{Temperature: c.settings.Temperature, NumCtx: numCtx, NumPredict: &one},
	})
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := c.pull.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode >= 200 && response.StatusCode <= 299
}
