// Package model is the local model: a client that never leaves the loopback,
// the gate in front of it, and the role `propose`. It follows
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
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// Connect short, read long: 2 s tell "Ollama is not running" from "Ollama is
// computing", 30 s carry a cold start (the reference measured 6.1 s at worst).
const (
	connectTimeout = 2 * time.Second
	totalTimeout   = 30 * time.Second
	numCtx         = 8192
)

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
	// timeout: a download of several GB outlasts any limit a question keeps.
	pull *http.Client
	base string // the endpoint without its trailing slashes
	url  string
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
}

type generateOptions struct {
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
}

// Ask is one question to the model. false stands for every outage: no
// connection, a timeout, a status that is not 2xx (a redirect included), a
// body that is not a JSON object with a string `response`. An empty string
// is an answer; the role judges it.
func (c *Client) Ask(ctx context.Context, prompt string) (string, bool) {
	// A struct of strings, bools and numbers always marshals, and the URL
	// passed the guard, so neither of the next two calls can fail.
	body, _ := json.Marshal(generateRequest{
		Model: c.settings.Name, Prompt: prompt,
		Options: generateOptions{Temperature: c.settings.Temperature, NumCtx: numCtx},
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
