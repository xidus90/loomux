package search

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ColdAttempts = 3
)

// WarmingNotice is the brain daemon's word for a search that had to start the engine
// (daemon/server.py). The Python command line never says it; loomux says it once per port.
const WarmingNotice = "starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm."

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

// WithNotice hands the default connect a function that hears WarmingNotice when the port had
// to start the daemon. It has no effect beside WithConnect.
func WithNotice(fn func(message string)) QmdMcpOption {
	return func(p *QmdMcpPort) {
		p.notice = fn
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
	notice   func(message string)

	session Session
	mu      sync.Mutex
}

// DefaultConnectWith returns a ConnectFunc using the given launcher, spawner, and timeout.
// notice, when not nil, hears WarmingNotice at most once for the returned ConnectFunc: right
// after a daemon start succeeded, before the wait for it.
func DefaultConnectWith(port int, launcher func(string) ([]string, error), spawner DaemonSpawner, waitTimeout time.Duration, notice func(string)) ConnectFunc {
	var once sync.Once
	return func(env map[string]string) (Session, error) {
		session := NewHTTPSession(port)
		if !session.Reachable() {
			if err := StartDaemonWith(env, port, launcher, spawner); err != nil {
				return nil, err
			}
			if notice != nil {
				once.Do(func() { notice(WarmingNotice) })
			}
			if err := session.WaitUntilReachable(waitTimeout); err != nil {
				return nil, err
			}
		}
		return session, nil
	}
}

// DefaultConnect returns a ConnectFunc that connects to a local daemon on port, starting one if needed.
func DefaultConnect(port int, notice func(string)) ConnectFunc {
	return DefaultConnectWith(port, Launcher, DefaultSpawner, 60*time.Second, notice)
}

// connectDefault is the connect NewQmdMcpPort falls back to; a test replaces it to see what
// the port hands on without starting a daemon.
var connectDefault = DefaultConnect

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
		p.connect = connectDefault(p.port, p.notice)
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
				// A connection that never came is not a daemon that stumbled: the
				// reference connects outside the retried block (qmd_mcp.py `_ask`), so a
				// failed connect leaves at once. Retrying it would spawn one detached
				// daemon per attempt and wait out the connect timeout three times over.
				return nil, unanswered(append(failures, err.Error()))
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
	return nil, unanswered(failures)
}

// unanswered is what a caller hears when the engine gave no reply: the attempts that were
// actually spent, and what each of them ran into.
func unanswered(failures []string) error {
	return fmt.Errorf("the search engine did not answer in %d attempts: %s", len(failures), strings.Join(failures, "; "))
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
		rawLine, numbered := 0, false
		if lineVal, ok := resMap["line"]; ok && lineVal != nil {
			switch v := lineVal.(type) {
			case int:
				line = v
				rawLine, numbered = v, true
			case float64:
				line = int(v)
				rawLine, numbered = int(v), true
			}
			if line < 1 {
				line = 1
			}
		}

		title, _ := resMap["title"].(string)
		snippet, _ := resMap["snippet"].(string)
		if numbered {
			snippet = withoutLineNumbers(snippet, rawLine)
		}
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

// withoutLineNumbers takes off the numbers qmd's daemon puts in front of every snippet line of a
// query answer (addLineNumbers in mcp/server.js: each "\n"-separated part i begins with
// "<line+i>: "). qmd query --json, the output the Python reference reads, has none. A snippet
// in which a single part lacks its own number is handed on as it came.
func withoutLineNumbers(snippet string, line int) string {
	parts := strings.Split(snippet, "\n")
	for i, part := range parts {
		prefix := strconv.Itoa(line+i) + ": "
		if !strings.HasPrefix(part, prefix) {
			return snippet
		}
		parts[i] = part[len(prefix):]
	}
	return strings.Join(parts, "\n")
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
