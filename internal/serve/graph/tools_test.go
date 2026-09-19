package graph_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/query"
	servegraph "github.com/xidus90/loomux/internal/serve/graph"
)

// connect wires a server with the two tools to an in-memory client. No socket,
// no port, no waiting.
func connect(t *testing.T, channel privacy.Channel, deps servegraph.Deps) *mcp.ClientSession {
	t.Helper()
	return connectWith(t, channel, deps, nil)
}

// connectWith is connect with client options, so that a test can listen for the
// progress notifications the notes become.
func connectWith(t *testing.T, channel privacy.Channel, deps servegraph.Deps, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "loomux", Version: "test"}, nil)
	servegraph.Register(server, channel, deps)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	// Connect returns the server session too; the test needs only the error.
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, opts)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// progressSink collects the progress notifications of one session.
type progressSink struct {
	messages chan string
}

func newProgressSink() *progressSink {
	return &progressSink{messages: make(chan string, 8)}
}

func (s *progressSink) options() *mcp.ClientOptions {
	return &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			s.messages <- req.Params.Message
		},
	}
}

// next waits for one notification. A notification travels on its own, so it can
// arrive after the answer did.
func (s *progressSink) next(t *testing.T) string {
	t.Helper()
	select {
	case message := <-s.messages:
		return message
	case <-time.After(5 * time.Second):
		t.Fatal("no progress notification arrived")
		return ""
	}
}

// registry writes a registry with two areas into a fresh directory: open,
// which every channel sees, and private, which is local_only. open's manifest
// declares one never glob.
func registry(t *testing.T) (dir, open, private string) {
	t.Helper()
	dir = t.TempDir()
	open, private = t.TempDir(), t.TempDir()
	manifests := map[string]string{
		open:    "[area]\nscope = \"project/open\"\n\n[privacy]\nnever = [\"secrets/**\"]\n",
		private: "[area]\nscope = \"project/private\"\n\n[privacy]\nmode = \"local_only\"\n",
	}
	for root, body := range manifests {
		path := filepath.Join(root, ".loomux", "config.toml")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	reg := "[[area]]\nscope = \"project/open\"\npath = " + strconv.Quote(filepath.ToSlash(open)) + "\n\n" +
		"[[area]]\nscope = \"project/private\"\npath = " + strconv.Quote(filepath.ToSlash(private)) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte(reg), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, open, private
}

// call runs one tool and returns its text and error flag.
func call(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return res.Content[0].(*mcp.TextContent).Text, res.IsError
}

// recorder is a fake Ask and Check that remembers what reached it.
type recorder struct {
	root       string
	question   string
	opts       query.AskOptions
	asked      bool
	checked    bool
	answer     ask.Answer
	notes      []string
	askErr     error
	drift      query.Drift
	checkErr   error
	askPanic   any
	checkPanic any
}

func (r *recorder) deps(registryDir string) servegraph.Deps {
	return servegraph.Deps{
		RegistryDir: registryDir,
		LegacyDir:   registryDir,
		Ask: func(root, question string, opts query.AskOptions) (ask.Answer, []string, error) {
			if r.askPanic != nil {
				panic(r.askPanic)
			}
			r.asked, r.root, r.question, r.opts = true, root, question, opts
			return r.answer, r.notes, r.askErr
		},
		Check: func(root string) (query.Drift, error) {
			if r.checkPanic != nil {
				panic(r.checkPanic)
			}
			r.checked, r.root = true, root
			return r.drift, r.checkErr
		},
	}
}

func sameDir(t *testing.T, got, want string) {
	t.Helper()
	if filepath.Clean(filepath.FromSlash(got)) != filepath.Clean(want) {
		t.Errorf("root = %q, want %q", got, want)
	}
}

func TestTheTwoToolsAreListed(t *testing.T) {
	dir, _, _ := registry(t)
	session := connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir))
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	got := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	// Sorted by the SDK, see TestTheFiveToolsAreListed in serve/brain.
	if strings.Join(got, ",") != "graph_check_freshness,graph_find_code" {
		t.Errorf("tools = %v", got)
	}
}

func TestFindCodeWithoutAScopeIsRefused(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code", map[string]any{"query": "x"})
	if !isError || text != "graph_find_code requires a scope" || r.asked {
		t.Errorf("got %q (isError %v, asked %v)", text, isError, r.asked)
	}
}

func TestFindCodeWithoutAQueryIsRefused(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code", map[string]any{"scope": "project/open"})
	if !isError || text != "graph_find_code requires a query" || r.asked {
		t.Errorf("got %q (isError %v, asked %v)", text, isError, r.asked)
	}
}

func TestFindCodeReachesTheAreaRootWithGraftsDefaults(t *testing.T) {
	dir, open, _ := registry(t)
	r := &recorder{}
	call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "cache"})
	sameDir(t, r.root, open)
	if r.question != "cache" {
		t.Errorf("question = %q", r.question)
	}
	o := r.opts
	if o.Limit != 5 || !o.Source || o.Full || o.In != "" || o.NoRefresh || o.Keep == nil {
		t.Errorf("opts = %+v, want limit 5, source on, full off, refresh on, a Keep", o)
	}
}

func TestFindCodePassesLimitFullAndIn(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	session := connect(t, privacy.ChannelLocal, r.deps(dir))
	call(t, session, "graph_find_code", map[string]any{
		"scope": "project/open", "query": "q", "limit": 3, "full": true, "in": "lib",
	})
	if r.opts.Limit != 3 || !r.opts.Full || r.opts.In != "lib" {
		t.Errorf("opts = %+v", r.opts)
	}
	for _, bad := range []any{0, -2, "seven"} {
		call(t, session, "graph_find_code", map[string]any{"scope": "project/open", "query": "q", "limit": bad})
		if r.opts.Limit != 5 {
			t.Errorf("limit %v became %d, want 5", bad, r.opts.Limit)
		}
	}
}

func TestKeepIsTheAreasNeverGlobs(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if r.opts.Keep("secrets/a.go") {
		t.Error("secrets/a.go lies under the never glob and must be refused")
	}
	if !r.opts.Keep("lib/a.go") {
		t.Error("lib/a.go must be admitted")
	}
}

func TestALocalOnlyAreaDoesNotExistOnTheCloudChannel(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	session := connect(t, privacy.ChannelCloud, r.deps(dir))
	for _, tool := range []string{"graph_find_code", "graph_check_freshness"} {
		text, isError := call(t, session, tool, map[string]any{"scope": "project/private", "query": "q"})
		if !isError || !strings.HasPrefix(text, "unknown scope 'project/private'") {
			t.Errorf("%s: got %q (isError %v)", tool, text, isError)
		}
		if strings.Contains(text, "project/private;") || strings.Contains(text, ", project/private") {
			t.Errorf("%s: the list of known scopes must not name the hidden area: %q", tool, text)
		}
	}
	if r.asked || r.checked {
		t.Error("a hidden area must be refused before anything is read")
	}
}

func TestALocalOnlyAreaIsVisibleOnTheLocalChannel(t *testing.T) {
	dir, _, private := registry(t)
	r := &recorder{}
	call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/private", "query": "q"})
	sameDir(t, r.root, private)
}

func TestAllIsNotAScope(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "all", "query": "q"})
	if !isError || !strings.HasPrefix(text, "unknown scope 'all'") || r.asked {
		t.Errorf("got %q (isError %v, asked %v)", text, isError, r.asked)
	}
}

func TestNotesComeBeforeTheAnswerAndAsProgress(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{answer: ask.Answer{Note: "the answer"}, notes: []string{"3 files moved, rebuilding the graph"}}
	sink := newProgressSink()
	session := connectWith(t, privacy.ChannelLocal, r.deps(dir), sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_find_code",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"scope": "project/open", "query": "q"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text := res.Content[0].(*mcp.TextContent).Text; text != "3 files moved, rebuilding the graph\nthe answer" {
		t.Errorf("text = %q", text)
	}
	if got := sink.next(t); got != "3 files moved, rebuilding the graph" {
		t.Errorf("progress = %q", got)
	}
}

func TestNoGraphIsAnErrorForTheModel(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{askErr: query.ErrNoGraph}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if !isError || !strings.Contains(text, "loomux graph build") {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

// drifted is the drift query.Check finds in a repository with lib/a.go and
// secrets/k.go after the named files change their bodies. A hand-built
// query.Drift records no node paths, and Only hides every id it cannot place,
// so a test that wants a visible id takes one from a real Check.
func drifted(t *testing.T, change ...string) query.Drift {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":       "module example.com/drift\n",
		"lib/a.go":     "package lib\n\nfunc A() {}\n",
		"secrets/k.go": "package secrets\n\nfunc K() {}\n",
	}
	write := func(rel, body string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for rel, body := range files {
		write(rel, body)
	}
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range change {
		write(rel, strings.Replace(files[rel], "{}", "{ _ = 1 }", 1))
	}
	d, err := query.Check(root)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCheckFreshnessRunsWithoutARefreshAndDriftIsNoError(t *testing.T) {
	dir, open, _ := registry(t)
	r := &recorder{drift: drifted(t, "lib/a.go", "secrets/k.go")}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if isError {
		t.Fatalf("drift is a report, not an error: %q", text)
	}
	sameDir(t, r.root, open)
	if !strings.Contains(text, "lib/a.go#A") || strings.Contains(text, "secrets/k.go") {
		t.Errorf("text %q must show lib and hide secrets", text)
	}
	if !strings.Contains(text, "2 more under the area's never globs") {
		t.Errorf("text %q must say that drift was hidden: the file node and K", text)
	}
	if r.asked {
		t.Error("check_freshness must not go through Ask, which refreshes")
	}
}

func TestTheCloudChannelHearsNoCountOfHiddenDrift(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{drift: drifted(t, "secrets/k.go")}
	text, isError := call(t, connect(t, privacy.ChannelCloud, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if isError {
		t.Fatalf("got an error: %q", text)
	}
	if strings.Contains(text, "never globs") || strings.Contains(text, "secrets/") {
		t.Errorf("the cloud channel must learn nothing about hidden drift: %q", text)
	}
	if !strings.Contains(text, "DRIFT") {
		t.Errorf("hidden drift is still drift, never OK: %q", text)
	}
}

func TestCheckFreshnessOnAMissingGraphIsText(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{drift: query.Drift{Missing: true}}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if isError || !strings.Contains(text, "NO GRAPH") {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestCheckFreshnessReportsAReadFailure(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{checkErr: errors.New("disk on fire")}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if !isError || text != "disk on fire" {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestCheckFreshnessWithoutAScopeIsRefused(t *testing.T) {
	dir, _, _ := registry(t)
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_check_freshness", nil)
	if !isError || text != "graph_check_freshness requires a scope" {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestAPanicBecomesAnErrorResult(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{askPanic: "boom", checkPanic: "bang"}
	session := connect(t, privacy.ChannelLocal, r.deps(dir))
	text, isError := call(t, session, "graph_find_code", map[string]any{"scope": "project/open", "query": "q"})
	if !isError || !strings.Contains(text, "boom") {
		t.Errorf("find_code: got %q (isError %v)", text, isError)
	}
	text, isError = call(t, session, "graph_check_freshness", map[string]any{"scope": "project/open"})
	if !isError || !strings.Contains(text, "bang") {
		t.Errorf("check_freshness: got %q (isError %v)", text, isError)
	}
}

func TestAPanicTellsTheCloudChannelNothingOfItsValue(t *testing.T) {
	// A panic value can be anything the failing code held -- a path, a line
	// of a hidden file -- so the cloud reads a fixed text instead of it.
	dir, _, _ := registry(t)
	r := &recorder{askPanic: "secrets/k.go: boom", checkPanic: "secrets/k.go: bang"}
	session := connect(t, privacy.ChannelCloud, r.deps(dir))
	for _, tool := range []string{"graph_find_code", "graph_check_freshness"} {
		text, isError := call(t, session, tool, map[string]any{"scope": "project/open", "query": "q"})
		if !isError || text != cloudPanic {
			t.Errorf("%s: got %q (isError %v), want %q", tool, text, isError, cloudPanic)
		}
	}
}

func TestABrokenRegistryIsAnErrorResult(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte("not toml ["), 0o600); err != nil {
		t.Fatal(err)
	}
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if !isError || text == "" {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestArgumentsThatAreNotAnObjectEndEmpty(t *testing.T) {
	// The twin of the serve/brain test of that name: broken arguments are a
	// missing scope, not an outage.
	dir, _, _ := registry(t)
	session := connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir))
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "graph_check_freshness", Arguments: []any{1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text := res.Content[0].(*mcp.TextContent).Text; !res.IsError || text != "graph_check_freshness requires a scope" {
		t.Errorf("got %q (isError %v)", text, res.IsError)
	}
}

func TestWithoutAProgressTokenTheNotesLapseButStayInTheText(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{notes: []string{"note 1"}, askErr: errors.New("graph unreadable")}
	sink := newProgressSink()
	session := connectWith(t, privacy.ChannelLocal, r.deps(dir), sink.options())
	// Two calls on one session, the second one with a token: what arrives first
	// tells whether the first call's note lapsed, and waiting for a message that
	// does come beats waiting a while for one that does not.
	text, isError := call(t, session, "graph_find_code", map[string]any{"scope": "project/open", "query": "q"})
	if !isError || text != "note 1\ngraph unreadable" {
		t.Errorf("got %q (isError %v); a failure keeps its notes in front", text, isError)
	}
	r.notes = []string{"note 2"}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_find_code",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"scope": "project/open", "query": "q"},
	}); err != nil {
		t.Fatalf("second CallTool: %v", err)
	}
	if got := sink.next(t); got != "note 2" {
		t.Errorf("the first note to arrive is %q; the untokened call did not lapse", got)
	}
}

func TestALimitOfOneIsTakenAsGiven(t *testing.T) {
	// One is the smallest count that means something; the fallback to 5 is
	// for what lies below it.
	dir, _, _ := registry(t)
	r := &recorder{}
	call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q", "limit": 1})
	if r.opts.Limit != 1 {
		t.Errorf("limit 1 became %d", r.opts.Limit)
	}
}

func TestAnAnswerWithoutNotesIsTheReportAlone(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{answer: ask.Answer{Note: "no match for q"}}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if isError || text != "no match for q" {
		t.Errorf("got %q (isError %v), want the report with nothing in front", text, isError)
	}
}

func TestABrokenRegistryIsNotCalledAnUnknownScope(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte("not toml ["), 0o600); err != nil {
		t.Fatal(err)
	}
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if !isError || text == privacy.UnknownScope("project/open", nil).Error() {
		t.Errorf("got %q (isError %v), want the registry's own error", text, isError)
	}
	if !strings.Contains(text, "registry.toml") || !strings.Contains(text, "not valid TOML") {
		t.Errorf("text %q must name the registry file and its parse error", text)
	}
}

func TestABrokenRegistryIsTheFixedTextOnTheCloudChannel(t *testing.T) {
	// The registry's own error names its absolute path on this machine, and
	// a manifest's error names an area's directory; the cloud reads neither.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte("not toml ["), 0o600); err != nil {
		t.Fatal(err)
	}
	session := connect(t, privacy.ChannelCloud, (&recorder{}).deps(dir))
	for _, tool := range []string{"graph_find_code", "graph_check_freshness"} {
		text, isError := call(t, session, tool, map[string]any{"scope": "project/open", "query": "q"})
		if !isError || text != cloudFailure {
			t.Errorf("%s: got %q (isError %v), want %q", tool, text, isError, cloudFailure)
		}
	}
}

// hashRepo is one registered area whose never glob covers a file whose name
// starts with '#', with a graph built over the real query layer.
func hashRepo(t *testing.T) (servegraph.Deps, string) {
	t.Helper()
	dir, root := t.TempDir(), t.TempDir()
	files := map[string]string{
		".loomux/config.toml": "[area]\nscope = \"project/hash\"\n\n[privacy]\nnever = [\"#*\"]\n",
		"go.mod":              "module example.com/hash\n",
		"#gen.go":             "package main\n\n// Generate makes the secret table.\nfunc Generate() {}\n",
		"lib.go":              "package main\n\n// Generator is the open one.\nfunc Generator() { Generate() }\n",
	}
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	reg := "[[area]]\nscope = \"project/hash\"\npath = " + strconv.Quote(filepath.ToSlash(root)) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte(reg), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatal(err)
	}
	return servegraph.Deps{RegistryDir: dir, LegacyDir: dir, Ask: query.Ask, Check: query.Check}, root
}

func TestANeverGlobHidesAFileWhoseNameStartsWithAHashEndToEnd(t *testing.T) {
	deps, root := hashRepo(t)
	session := connect(t, privacy.ChannelLocal, deps)

	text, isError := call(t, session, "graph_find_code", map[string]any{"scope": "project/hash", "query": "generate"})
	if isError || !strings.Contains(text, "lib.go") {
		t.Fatalf("got %q (isError %v), want lib.go found", text, isError)
	}
	if strings.Contains(text, "#gen.go") {
		t.Errorf("find_code shows a file under the never glob:\n%s", text)
	}

	if err := os.WriteFile(filepath.Join(root, "#gen.go"), []byte("package main\n\nfunc Generate() { _ = 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	text, _ = call(t, session, "graph_check_freshness", map[string]any{"scope": "project/hash"})
	if strings.Contains(text, "#gen.go") || !strings.Contains(text, "DRIFT") {
		t.Errorf("check_freshness must report DRIFT and hide #gen.go:\n%s", text)
	}
}

// none waits a short while and fails if a progress notification arrives. A
// notification that is sent arrives within milliseconds, so the wait is long
// enough to see one and short enough not to slow the suite.
func (s *progressSink) none(t *testing.T) {
	t.Helper()
	select {
	case message := <-s.messages:
		t.Errorf("progress %q arrived; the cloud channel must hear no refresh note", message)
	case <-time.After(300 * time.Millisecond):
	}
}

// hiddenFailure is the kind of error a rebuild or a read produces: it names
// a file under the never globs, its line and its token.
var hiddenFailure = errors.New("parse secrets/gen.go: secrets/gen.go:3:1: expected declaration, found oops")

const cloudFailure = "the graph could not be read on this channel; ask on the local channel for details"

const cloudPanic = "internal error; ask on the local channel for details"

func TestTheCloudChannelHearsNoRefreshNote(t *testing.T) {
	// A refresh note counts files under the never globs too ("3 files moved,
	// rebuilding the graph"), so on the cloud channel it goes nowhere: not
	// into the text and not up as progress.
	dir, _, _ := registry(t)
	r := &recorder{answer: ask.Answer{Note: "the answer"}, notes: []string{"3 files moved, rebuilding the graph"}}
	sink := newProgressSink()
	session := connectWith(t, privacy.ChannelCloud, r.deps(dir), sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_find_code",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"scope": "project/open", "query": "q"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text := res.Content[0].(*mcp.TextContent).Text; res.IsError || text != "the answer" {
		t.Errorf("got %q (isError %v), want the answer alone", text, res.IsError)
	}
	sink.none(t)
}

func TestACloudFailureOfFindCodeIsTheFixedText(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{notes: []string{"rebuild failed: " + hiddenFailure.Error()}, askErr: hiddenFailure}
	text, isError := call(t, connect(t, privacy.ChannelCloud, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if !isError || text != cloudFailure {
		t.Errorf("got %q (isError %v), want %q", text, isError, cloudFailure)
	}
}

func TestACloudFailureOfCheckFreshnessIsTheFixedText(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{checkErr: hiddenFailure}
	text, isError := call(t, connect(t, privacy.ChannelCloud, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if !isError || text != cloudFailure {
		t.Errorf("got %q (isError %v), want %q", text, isError, cloudFailure)
	}
}

func TestNoGraphKeepsItsOwnTextOnTheCloudChannel(t *testing.T) {
	// ErrNoGraph names no path, and it tells the caller what to do.
	dir, _, _ := registry(t)
	r := &recorder{askErr: fmt.Errorf("ask: %w", query.ErrNoGraph)}
	text, isError := call(t, connect(t, privacy.ChannelCloud, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if !isError || text != query.ErrNoGraph.Error() {
		t.Errorf("got %q (isError %v), want the no-graph text alone, without what wrapped it", text, isError)
	}
}

func TestALocalFailureKeepsItsDetail(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{askErr: hiddenFailure, checkErr: hiddenFailure}
	session := connect(t, privacy.ChannelLocal, r.deps(dir))
	for _, tool := range []string{"graph_find_code", "graph_check_freshness"} {
		text, isError := call(t, session, tool, map[string]any{"scope": "project/open", "query": "q"})
		if !isError || text != hiddenFailure.Error() {
			t.Errorf("%s: got %q (isError %v), want the error itself", tool, text, isError)
		}
	}
}

func TestFindCodeChecksTheQueryBeforeTheScope(t *testing.T) {
	// Both arguments are refused before anything is resolved: an unknown
	// scope without a query is a missing query, not an unknown scope.
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/nowhere"})
	if !isError || text != "graph_find_code requires a query" || r.asked {
		t.Errorf("got %q (isError %v, asked %v)", text, isError, r.asked)
	}
}

func TestFindCodeWithNeitherArgumentAsksForTheScope(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code", map[string]any{})
	if !isError || text != "graph_find_code requires a scope" || r.asked {
		t.Errorf("got %q (isError %v, asked %v); the scope is checked first", text, isError, r.asked)
	}
}
