package search

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	ColdAttempts = 3
)

// ConnectFunc connects to the search daemon session.
type ConnectFunc func(env map[string]string) (Session, error)

// QmdMcpOption configures a QmdMcpPort.
type QmdMcpOption func(*QmdMcpPort)

// WithConnect sets a custom daemon connect function.
func WithConnect(fn ConnectFunc) QmdMcpOption {
	return func(p *QmdMcpPort) {
		p.connect = fn
	}
}

// WithColdAttempts sets max cold start attempts.
func WithColdAttempts(attempts int) QmdMcpOption {
	return func(p *QmdMcpPort) {
		p.attempts = attempts
	}
}

// WithCLI sets the CLI SearchPort for maintenance delegation.
func WithCLI(cli SearchPort) QmdMcpOption {
	return func(p *QmdMcpPort) {
		p.cli = cli
	}
}

// WithBackbone sets the compute backbone.
func WithBackbone(backbone Backbone) QmdMcpOption {
	return func(p *QmdMcpPort) {
		p.backbone = backbone
		p.env = BackboneEnv(backbone)
	}
}

// WithPort sets the daemon port.
func WithPort(port int) QmdMcpOption {
	return func(p *QmdMcpPort) {
		p.port = port
	}
}

// QmdMcpPort implements SearchPort over a warm qmd daemon via MCP.
type QmdMcpPort struct {
	backbone Backbone
	env      map[string]string
	port     int
	connect  ConnectFunc
	cli      SearchPort
	attempts int

	session Session
	mu      sync.Mutex
}

// DefaultConnectWith returns a ConnectFunc using the given launcher, spawner, and timeout.
func DefaultConnectWith(port int, launcher func(string) ([]string, error), spawner DaemonSpawner, waitTimeout time.Duration) ConnectFunc {
	return func(env map[string]string) (Session, error) {
		session := NewHTTPSession(port)
		if !session.Reachable() {
			if err := StartDaemonWith(env, port, launcher, spawner); err != nil {
				return nil, err
			}
			if err := session.WaitUntilReachable(waitTimeout); err != nil {
				return nil, err
			}
		}
		return session, nil
	}
}

// DefaultConnect returns a ConnectFunc that connects to a local daemon on port, starting one if needed.
func DefaultConnect(port int) ConnectFunc {
	return DefaultConnectWith(port, Launcher, DefaultSpawner, 60*time.Second)
}

// NewQmdMcpPort creates a new QmdMcpPort.
func NewQmdMcpPort(opts ...QmdMcpOption) *QmdMcpPort {
	p := &QmdMcpPort{
		backbone: DefaultBackbone,
		env:      BackboneEnv(DefaultBackbone),
		port:     DefaultPort,
		attempts: ColdAttempts,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.connect == nil {
		p.connect = DefaultConnect(p.port)
	}
	if p.cli == nil {
		p.cli = &QmdPort{Executable: "qmd"}
	}
	return p
}

// Search executes a search query.
func (p *QmdMcpPort) Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error) {
	args := formatArguments(query, collections, profile, n)
	p.mu.Lock()
	defer p.mu.Unlock()

	reply, err := p.ask(args)
	if err != nil {
		return nil, err
	}
	return translateReply(reply, collections), nil
}

func (p *QmdMcpPort) ask(args map[string]any) (map[string]any, error) {
	var failures []string
	for i := 0; i < p.attempts; i++ {
		session := p.session
		if session == nil {
			var err error
			session, err = p.connect(p.env)
			if err != nil {
				failures = append(failures, err.Error())
				p.letGo()
				continue
			}
			p.session = session
		}

		reply, err := session.Call("query", args)
		if err != nil {
			failures = append(failures, err.Error())
			p.letGo()
			continue
		}
		return reply, nil
	}
	return nil, fmt.Errorf("the search engine did not answer in %d attempts: %s", p.attempts, strings.Join(failures, "; "))
}

func (p *QmdMcpPort) letGo() {
	if p.session != nil {
		_ = p.session.Close()
		p.session = nil
	}
}

// Close closes any open session.
func (p *QmdMcpPort) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.letGo()
	return nil
}

// Indexed returns indexed relative paths for collection.
func (p *QmdMcpPort) Indexed(collection string) ([]string, error) {
	return p.cli.Indexed(collection)
}

// Refresh triggers an index update.
func (p *QmdMcpPort) Refresh(collections []string) error {
	return p.cli.Refresh(collections)
}

// NotYetSearchable returns the number of pending documents.
func (p *QmdMcpPort) NotYetSearchable() (int, error) {
	return p.cli.NotYetSearchable()
}

// Embed triggers embedding generation.
func (p *QmdMcpPort) Embed(collections []string) error {
	return p.cli.Embed(collections)
}

func formatArguments(query string, collections []string, profile Profile, n int) map[string]any {
	cols := make([]string, len(collections))
	copy(cols, collections)

	args := map[string]any{
		"limit":       n,
		"collections": cols,
	}

	switch profile {
	case ProfileFast:
		args["searches"] = []map[string]any{
			{"type": "vec", "query": query},
		}
		args["rerank"] = false
	case ProfileKeyword:
		args["searches"] = []map[string]any{
			{"type": "lex", "query": query},
		}
		args["rerank"] = false
	default:
		args["query"] = query
		args["rerank"] = true
	}
	return args
}

func translateReply(reply map[string]any, collections []string) []SearchHit {
	sc, ok := reply["structuredContent"].(map[string]any)
	if !ok {
		return nil
	}
	results, ok := sc["results"].([]any)
	if !ok {
		return nil
	}

	hits := make([]SearchHit, 0, len(results))
	for _, res := range results {
		resMap, ok := res.(map[string]any)
		if !ok {
			continue
		}
		filePath, _ := resMap["file"].(string)
		collection, relative := splitPath(filePath, collections)

		line := 1
		if lineVal, ok := resMap["line"]; ok && lineVal != nil {
			switch v := lineVal.(type) {
			case int:
				line = v
			case float64:
				line = int(v)
			}
			if line < 1 {
				line = 1
			}
		}

		title, _ := resMap["title"].(string)
		snippet, _ := resMap["snippet"].(string)
		score := 0.0
		if scoreVal, ok := resMap["score"].(float64); ok {
			score = scoreVal
		}
		docID, _ := resMap["docid"].(string)

		hits = append(hits, SearchHit{
			Collection: collection,
			Relative:   relative,
			Line:       line,
			Title:      title,
			Snippet:    snippet,
			Score:      score,
			ContentKey: docID,
		})
	}
	return hits
}

func splitPath(path string, collections []string) (string, string) {
	head, rest, found := strings.Cut(path, "/")
	if found {
		for _, col := range collections {
			if head == col {
				return head, rest
			}
		}
	}
	defaultCol := ""
	if len(collections) > 0 {
		defaultCol = collections[0]
	}
	return defaultCol, path
}
