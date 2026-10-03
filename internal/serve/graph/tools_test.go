package graph_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/gitenv"
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

	callersSym   string
	callersOpts  query.CallersOptions
	callersAns   query.CallersAnswer
	callersErr   error
	callersPanic any

	skeletonFile  string
	skeletonOpts  query.SkeletonOptions
	skeletonAns   query.SkeletonAnswer
	skeletonErr   error
	skeletonPanic any

	grepPat   string
	grepOpts  query.GrepOptions
	grepAns   query.GrepAnswer
	grepErr   error
	grepPanic any

	mapOpts  query.MapOptions
	mapAns   query.MapAnswer
	mapErr   error
	mapPanic any

	blasted    bool
	blastOpts  query.BlastOptions
	blastAns   query.BlastAnswer
	blastErr   error
	blastPanic any
}

func (r *recorder) deps(registryDir string) servegraph.Deps {
	return servegraph.Deps{
		RegistryDir: registryDir,
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
		Callers: func(root, symbol string, opts query.CallersOptions) (query.CallersAnswer, []string, error) {
			if r.callersPanic != nil {
				panic(r.callersPanic)
			}
			r.root, r.callersSym, r.callersOpts = root, symbol, opts
			return r.callersAns, r.notes, r.callersErr
		},
		Skeleton: func(root, file string, opts query.SkeletonOptions) (query.SkeletonAnswer, []string, error) {
			if r.skeletonPanic != nil {
				panic(r.skeletonPanic)
			}
			r.root, r.skeletonFile, r.skeletonOpts = root, file, opts
			return r.skeletonAns, r.notes, r.skeletonErr
		},
		Grep: func(root, pattern string, opts query.GrepOptions) (query.GrepAnswer, []string, error) {
			if r.grepPanic != nil {
				panic(r.grepPanic)
			}
			r.root, r.grepPat, r.grepOpts = root, pattern, opts
			return r.grepAns, r.notes, r.grepErr
		},
		Map: func(root string, opts query.MapOptions) (query.MapAnswer, []string, error) {
			if r.mapPanic != nil {
				panic(r.mapPanic)
			}
			r.root, r.mapOpts = root, opts
			return r.mapAns, r.notes, r.mapErr
		},
		Blast: func(root string, opts query.BlastOptions) (query.BlastAnswer, []string, error) {
			if r.blastPanic != nil {
				panic(r.blastPanic)
			}
			r.blasted, r.root, r.blastOpts = true, root, opts
			return r.blastAns, r.notes, r.blastErr
		},
	}
}

func sameDir(t *testing.T, got, want string) {
	t.Helper()
	if filepath.Clean(filepath.FromSlash(got)) != filepath.Clean(want) {
		t.Errorf("root = %q, want %q", got, want)
	}
}

func TestTheSevenToolsAreListed(t *testing.T) {
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
	want := "graph_blast,graph_check_freshness,graph_file_api,graph_find_all,graph_find_code,graph_repo_map,graph_trace_calls"
	if strings.Join(got, ",") != want {
		t.Errorf("tools = %v, want %v", got, want)
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
	return servegraph.Deps{RegistryDir: dir, Ask: query.Ask, Check: query.Check}, root
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

func TestFileApi(t *testing.T) {
	dir, _, _ := registry(t)

	// Missing scope
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_file_api",
		map[string]any{"file": "main.go"})
	if !isError || text != "graph_file_api requires a scope" {
		t.Fatalf("want scope required, got %q", text)
	}

	// Missing file
	text, isError = call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_file_api",
		map[string]any{"scope": "project/open"})
	if !isError || text != "graph_file_api requires a file" {
		t.Fatalf("want file required, got %q", text)
	}

	// Unknown scope
	text, isError = call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_file_api",
		map[string]any{"scope": "project/nowhere", "file": "main.go"})
	if !isError {
		t.Fatal("want error for unknown scope")
	}

	// Success with notes and progress
	sink := newProgressSink()
	r := &recorder{notes: []string{"rebuilt graph"}}
	session := connectWith(t, privacy.ChannelLocal, r.deps(dir), sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_file_api",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"scope": "project/open", "file": "main.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text = res.Content[0].(*mcp.TextContent).Text
	if res.IsError || !strings.Contains(text, "rebuilt graph") {
		t.Fatalf("unexpected result: %q, isError=%v", text, res.IsError)
	}
	if sink.next(t) != "rebuilt graph" {
		t.Fatal("progress notification missing")
	}

	// Cloud suppresses notes
	rCloud := &recorder{notes: []string{"rebuilt graph"}}
	text, isError = call(t, connect(t, privacy.ChannelCloud, rCloud.deps(dir)), "graph_file_api",
		map[string]any{"scope": "project/open", "file": "main.go"})
	if isError || strings.Contains(text, "rebuilt graph") {
		t.Fatalf("cloud must suppress notes, got %q", text)
	}

	// Error handling: local vs cloud (cloudFailure and ErrNoGraph)
	rErr := &recorder{skeletonErr: errors.New("read failed")}
	text, isError = call(t, connect(t, privacy.ChannelLocal, rErr.deps(dir)), "graph_file_api",
		map[string]any{"scope": "project/open", "file": "main.go"})
	if !isError || !strings.Contains(text, "read failed") {
		t.Fatalf("local must report error, got %q", text)
	}

	text, isError = call(t, connect(t, privacy.ChannelCloud, rErr.deps(dir)), "graph_file_api",
		map[string]any{"scope": "project/open", "file": "main.go"})
	if !isError || text != cloudFailure {
		t.Fatalf("cloud must report cloudFailure, got %q", text)
	}

	rNoGraph := &recorder{skeletonErr: query.ErrNoGraph}
	text, isError = call(t, connect(t, privacy.ChannelCloud, rNoGraph.deps(dir)), "graph_file_api",
		map[string]any{"scope": "project/open", "file": "main.go"})
	if !isError || text != query.ErrNoGraph.Error() {
		t.Fatalf("cloud must report ErrNoGraph text, got %q", text)
	}

	// Panic handling: local vs cloud
	rPanic := &recorder{skeletonPanic: "boom"}
	text, isError = call(t, connect(t, privacy.ChannelLocal, rPanic.deps(dir)), "graph_file_api",
		map[string]any{"scope": "project/open", "file": "main.go"})
	if !isError || !strings.Contains(text, "boom") {
		t.Fatalf("local panic must contain boom, got %q", text)
	}

	text, isError = call(t, connect(t, privacy.ChannelCloud, rPanic.deps(dir)), "graph_file_api",
		map[string]any{"scope": "project/open", "file": "main.go"})
	if !isError || text != cloudPanic {
		t.Fatalf("cloud panic must be cloudPanic, got %q", text)
	}
}

func TestTraceCalls(t *testing.T) {
	dir, _, _ := registry(t)

	// Missing scope
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_trace_calls",
		map[string]any{"symbol": "Main"})
	if !isError || text != "graph_trace_calls requires a scope" {
		t.Fatalf("want scope required, got %q", text)
	}

	// Missing symbol
	text, isError = call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_trace_calls",
		map[string]any{"scope": "project/open"})
	if !isError || text != "graph_trace_calls requires a symbol" {
		t.Fatalf("want symbol required, got %q", text)
	}

	// Unknown scope
	text, isError = call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_trace_calls",
		map[string]any{"scope": "project/nowhere", "symbol": "Main"})
	if !isError {
		t.Fatal("want error for unknown scope")
	}

	// Execution: default dir (in), direction out, depths
	tests := []struct {
		args    map[string]any
		wantDir blast.Direction
		wantDep blast.Depth
		wantIn  string
	}{
		{
			args:    map[string]any{"scope": "project/open", "symbol": "Main"},
			wantDir: blast.In,
			wantDep: 1,
		},
		{
			args:    map[string]any{"scope": "project/open", "symbol": "Main", "direction": "out", "depth": "all", "in": "pkg/"},
			wantDir: blast.Out,
			wantDep: blast.All,
			wantIn:  "pkg/",
		},
		{
			args:    map[string]any{"scope": "project/open", "symbol": "Main", "depth": "Full"},
			wantDir: blast.In,
			wantDep: blast.All,
		},
		{
			args:    map[string]any{"scope": "project/open", "symbol": "Main", "direction": "in", "depth": "2"},
			wantDir: blast.In,
			wantDep: 2,
		},
		{
			args:    map[string]any{"scope": "project/open", "symbol": "Main", "depth": float64(3)},
			wantDir: blast.In,
			wantDep: 3,
		},
		{
			args:    map[string]any{"scope": "project/open", "symbol": "Main", "depth": float64(0)},
			wantDir: blast.In,
			wantDep: 1,
		},
		{
			args:    map[string]any{"scope": "project/open", "symbol": "Main", "depth": "invalid"},
			wantDir: blast.In,
			wantDep: 1,
		},
	}

	for _, tc := range tests {
		r := &recorder{}
		text, isError = call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_trace_calls", tc.args)
		if isError {
			t.Fatalf("unexpected error: %q", text)
		}
		if r.callersOpts.Direction != tc.wantDir {
			t.Errorf("direction = %v, want %v", r.callersOpts.Direction, tc.wantDir)
		}
		if r.callersOpts.Depth != tc.wantDep {
			t.Errorf("depth = %v, want %v", r.callersOpts.Depth, tc.wantDep)
		}
		if r.callersOpts.In != tc.wantIn {
			t.Errorf("in = %v, want %v", r.callersOpts.In, tc.wantIn)
		}
	}

	// Notes on local vs cloud
	sink := newProgressSink()
	rNotes := &recorder{notes: []string{"drift note"}}
	session := connectWith(t, privacy.ChannelLocal, rNotes.deps(dir), sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_trace_calls",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"scope": "project/open", "symbol": "Main"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text = res.Content[0].(*mcp.TextContent).Text
	if res.IsError || !strings.Contains(text, "drift note") {
		t.Fatalf("local must contain note, got %q", text)
	}
	if sink.next(t) != "drift note" {
		t.Fatal("progress notification missing")
	}

	rCloud := &recorder{notes: []string{"drift note"}}
	text, isError = call(t, connect(t, privacy.ChannelCloud, rCloud.deps(dir)), "graph_trace_calls",
		map[string]any{"scope": "project/open", "symbol": "Main"})
	if isError || strings.Contains(text, "drift note") {
		t.Fatalf("cloud must suppress notes, got %q", text)
	}

	// Error handling: local vs cloud
	rErr := &recorder{callersErr: errors.New("trace failed")}
	text, isError = call(t, connect(t, privacy.ChannelLocal, rErr.deps(dir)), "graph_trace_calls",
		map[string]any{"scope": "project/open", "symbol": "Main"})
	if !isError || !strings.Contains(text, "trace failed") {
		t.Fatalf("local must report error, got %q", text)
	}

	text, isError = call(t, connect(t, privacy.ChannelCloud, rErr.deps(dir)), "graph_trace_calls",
		map[string]any{"scope": "project/open", "symbol": "Main"})
	if !isError || text != cloudFailure {
		t.Fatalf("cloud must report cloudFailure, got %q", text)
	}

	// Panic handling: local vs cloud
	rPanic := &recorder{callersPanic: "panic trace"}
	text, isError = call(t, connect(t, privacy.ChannelLocal, rPanic.deps(dir)), "graph_trace_calls",
		map[string]any{"scope": "project/open", "symbol": "Main"})
	if !isError || !strings.Contains(text, "panic trace") {
		t.Fatalf("local panic must contain message, got %q", text)
	}

	text, isError = call(t, connect(t, privacy.ChannelCloud, rPanic.deps(dir)), "graph_trace_calls",
		map[string]any{"scope": "project/open", "symbol": "Main"})
	if !isError || text != cloudPanic {
		t.Fatalf("cloud panic must be cloudPanic, got %q", text)
	}
}

func TestGraphBlastWithoutAScopeIsRefused(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_blast", map[string]any{"base": "HEAD~1"})
	if !isError || text != "graph_blast requires a scope" || r.blasted {
		t.Errorf("got %q (isError %v, blasted %v)", text, isError, r.blasted)
	}
}

func TestGraphBlastPassesBaseDepthAndKeep(t *testing.T) {
	dir, open, _ := registry(t)
	r := &recorder{}
	_, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_blast",
		map[string]any{"scope": "project/open", "base": "main", "depth": 0.5})
	if isError || !r.blasted {
		t.Fatalf("isError %v, blasted %v", isError, r.blasted)
	}
	sameDir(t, r.root, open)
	o := r.blastOpts
	if o.Base != "main" || o.Depth != 1 || o.Cached || o.NoRefresh {
		t.Errorf("opts = %+v, want base main, depth 1, no cached, refresh on", o)
	}
	if o.Keep == nil || o.Keep("secrets/x.go") || !o.Keep("lib.go") {
		t.Error("Keep must refuse what the never glob covers and keep the rest")
	}

	r = &recorder{}
	call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_blast",
		map[string]any{"scope": "project/open", "depth": "all"})
	if r.blastOpts.Base != "" || r.blastOpts.Depth != blast.All {
		t.Errorf("opts = %+v, want no base and the whole closure", r.blastOpts)
	}
}

func TestGraphBlastAnswersWithTheReportAndTheNotesInFront(t *testing.T) {
	dir, _, _ := registry(t)
	ans := query.BlastAnswer{Range: "main...HEAD", Hidden: 2}
	want := strings.TrimSuffix(query.BlastReport(ans), "\n")

	r := &recorder{blastAns: ans, notes: []string{"rebuilt the graph"}}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_blast",
		map[string]any{"scope": "project/open"})
	if isError || text != "rebuilt the graph\n"+want {
		t.Errorf("local: got %q (isError %v)", text, isError)
	}

	r = &recorder{blastAns: ans, notes: []string{"rebuilt the graph"}}
	text, isError = call(t, connect(t, privacy.ChannelCloud, r.deps(dir)), "graph_blast",
		map[string]any{"scope": "project/open"})
	if isError || text != want {
		t.Errorf("cloud: got %q (isError %v), want the report without notes", text, isError)
	}
}

func TestGraphBlastFailures(t *testing.T) {
	dir, _, _ := registry(t)
	gitErr := errors.New("git diff: fatal: not a git repository: " + dir)
	cases := []struct {
		name    string
		channel privacy.Channel
		err     error
		want    string
	}{
		{"local git failure", privacy.ChannelLocal, gitErr, gitErr.Error()},
		{"cloud git failure", privacy.ChannelCloud, gitErr, cloudFailure},
		{"cloud no graph", privacy.ChannelCloud, fmt.Errorf("blast: %w", query.ErrNoGraph), query.ErrNoGraph.Error()},
		{"local bad base", privacy.ChannelLocal, fmt.Errorf("%w, got %q", query.ErrBadBase, "--output=x"),
			`--base must name a revision, got "--output=x"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &recorder{blastErr: c.err}
			text, isError := call(t, connect(t, c.channel, r.deps(dir)), "graph_blast",
				map[string]any{"scope": "project/open"})
			if !isError || text != c.want {
				t.Errorf("got %q (isError %v), want %q", text, isError, c.want)
			}
		})
	}

	r := &recorder{blastPanic: "secrets/x.go: boom"}
	text, isError := call(t, connect(t, privacy.ChannelCloud, r.deps(dir)), "graph_blast",
		map[string]any{"scope": "project/open"})
	if !isError || text != cloudPanic {
		t.Errorf("cloud panic: got %q (isError %v)", text, isError)
	}
}

func TestGraphBlastOutsideGitIsAnErrorEndToEnd(t *testing.T) {
	// The area has a graph but no repository: an error, not an empty report.
	deps, _ := hashRepo(t)
	deps.Blast = query.Blast
	text, isError := call(t, connect(t, privacy.ChannelLocal, deps), "graph_blast",
		map[string]any{"scope": "project/hash"})
	if !isError || !strings.Contains(text, "not a git repository") {
		t.Errorf("local: got %q (isError %v), want git's failure", text, isError)
	}
	text, isError = call(t, connect(t, privacy.ChannelCloud, deps), "graph_blast",
		map[string]any{"scope": "project/hash"})
	if !isError || text != cloudFailure {
		t.Errorf("cloud: got %q (isError %v), want %q", text, isError, cloudFailure)
	}
}

// A clean tree in a one-commit repository has nothing to compare with, and
// the local answer says so in its own words, not in git's.
func TestGraphBlastWithoutHistoryNamesItLocally(t *testing.T) {
	deps, root := hashRepo(t)
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "."}, {"commit", "-qm", "init"},
	} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gitenv.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	deps.Blast = query.Blast
	text, isError := call(t, connect(t, privacy.ChannelLocal, deps), "graph_blast",
		map[string]any{"scope": "project/hash"})
	if !isError || !strings.HasSuffix(text, "nothing to compare: HEAD has no parent commit") {
		t.Errorf("local: got %q (isError %v), want ErrNoHistory's text", text, isError)
	}
}

func TestFindAll(t *testing.T) {
	dir, _, _ := registry(t)

	// Missing scope
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_find_all",
		map[string]any{"pattern": "needle"})
	if !isError || text != "graph_find_all requires a scope" {
		t.Fatalf("want scope required, got %q", text)
	}

	// Missing pattern
	text, isError = call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_find_all",
		map[string]any{"scope": "project/open"})
	if !isError || text != "graph_find_all requires a pattern" {
		t.Fatalf("want pattern required, got %q", text)
	}

	// Unknown scope
	text, isError = call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_find_all",
		map[string]any{"scope": "project/nowhere", "pattern": "needle"})
	if !isError {
		t.Fatal("want error for unknown scope")
	}

	// Options forwarding
	r := &recorder{}
	text, isError = call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_all",
		map[string]any{"scope": "project/open", "pattern": "needle", "in": "src/", "ignore_case": true, "fixed": true})
	if isError {
		t.Fatalf("unexpected error: %q", text)
	}
	if r.grepPat != "needle" || r.grepOpts.In != "src/" || !r.grepOpts.IgnoreCase || !r.grepOpts.Fixed {
		t.Errorf("grep opts mismatch: pat=%v, opts=%+v", r.grepPat, r.grepOpts)
	}

	// Notes on local vs cloud
	sink := newProgressSink()
	rNotes := &recorder{notes: []string{"grep note"}}
	session := connectWith(t, privacy.ChannelLocal, rNotes.deps(dir), sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_find_all",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"scope": "project/open", "pattern": "needle"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text = res.Content[0].(*mcp.TextContent).Text
	if res.IsError || !strings.Contains(text, "grep note") {
		t.Fatalf("local must contain note, got %q", text)
	}
	if sink.next(t) != "grep note" {
		t.Fatal("progress notification missing")
	}

	rCloud := &recorder{notes: []string{"grep note"}}
	text, isError = call(t, connect(t, privacy.ChannelCloud, rCloud.deps(dir)), "graph_find_all",
		map[string]any{"scope": "project/open", "pattern": "needle"})
	if isError || strings.Contains(text, "grep note") {
		t.Fatalf("cloud must suppress notes, got %q", text)
	}

	// Error handling: local vs cloud
	rErr := &recorder{grepErr: errors.New("grep error")}
	text, isError = call(t, connect(t, privacy.ChannelLocal, rErr.deps(dir)), "graph_find_all",
		map[string]any{"scope": "project/open", "pattern": "needle"})
	if !isError || !strings.Contains(text, "grep error") {
		t.Fatalf("local must report error, got %q", text)
	}

	text, isError = call(t, connect(t, privacy.ChannelCloud, rErr.deps(dir)), "graph_find_all",
		map[string]any{"scope": "project/open", "pattern": "needle"})
	if !isError || text != cloudFailure {
		t.Fatalf("cloud must report cloudFailure, got %q", text)
	}

	// Panic handling: local vs cloud
	rPanic := &recorder{grepPanic: "panic grep"}
	text, isError = call(t, connect(t, privacy.ChannelLocal, rPanic.deps(dir)), "graph_find_all",
		map[string]any{"scope": "project/open", "pattern": "needle"})
	if !isError || !strings.Contains(text, "panic grep") {
		t.Fatalf("local panic must contain message, got %q", text)
	}

	text, isError = call(t, connect(t, privacy.ChannelCloud, rPanic.deps(dir)), "graph_find_all",
		map[string]any{"scope": "project/open", "pattern": "needle"})
	if !isError || text != cloudPanic {
		t.Fatalf("cloud panic must be cloudPanic, got %q", text)
	}
}

func TestRepoMap(t *testing.T) {
	dir, _, _ := registry(t)

	// Missing scope
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_repo_map",
		map[string]any{})
	if !isError || text != "graph_repo_map requires a scope" {
		t.Fatalf("want scope required, got %q", text)
	}

	// Unknown scope
	text, isError = call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_repo_map",
		map[string]any{"scope": "project/nowhere"})
	if !isError {
		t.Fatal("want error for unknown scope")
	}

	// Options: max_dirs provided vs default
	r1 := &recorder{}
	text, isError = call(t, connect(t, privacy.ChannelLocal, r1.deps(dir)), "graph_repo_map",
		map[string]any{"scope": "project/open", "max_dirs": float64(5)})
	if isError {
		t.Fatalf("unexpected error: %q", text)
	}
	if r1.mapOpts.MaxDirs != 5 {
		t.Errorf("maxDirs = %v, want 5", r1.mapOpts.MaxDirs)
	}

	r2 := &recorder{}
	text, isError = call(t, connect(t, privacy.ChannelLocal, r2.deps(dir)), "graph_repo_map",
		map[string]any{"scope": "project/open"})
	if isError {
		t.Fatalf("unexpected error: %q", text)
	}
	if r2.mapOpts.MaxDirs != 16 {
		t.Errorf("maxDirs = %v, want 16", r2.mapOpts.MaxDirs)
	}

	// Notes on local vs cloud
	sink := newProgressSink()
	rNotes := &recorder{notes: []string{"map note"}}
	session := connectWith(t, privacy.ChannelLocal, rNotes.deps(dir), sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_repo_map",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"scope": "project/open"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text = res.Content[0].(*mcp.TextContent).Text
	if res.IsError || !strings.Contains(text, "map note") {
		t.Fatalf("local must contain note, got %q", text)
	}
	if sink.next(t) != "map note" {
		t.Fatal("progress notification missing")
	}

	rCloud := &recorder{notes: []string{"map note"}}
	text, isError = call(t, connect(t, privacy.ChannelCloud, rCloud.deps(dir)), "graph_repo_map",
		map[string]any{"scope": "project/open"})
	if isError || strings.Contains(text, "map note") {
		t.Fatalf("cloud must suppress notes, got %q", text)
	}

	// Error handling: local vs cloud
	rErr := &recorder{mapErr: errors.New("map error")}
	text, isError = call(t, connect(t, privacy.ChannelLocal, rErr.deps(dir)), "graph_repo_map",
		map[string]any{"scope": "project/open"})
	if !isError || !strings.Contains(text, "map error") {
		t.Fatalf("local must report error, got %q", text)
	}

	text, isError = call(t, connect(t, privacy.ChannelCloud, rErr.deps(dir)), "graph_repo_map",
		map[string]any{"scope": "project/open"})
	if !isError || text != cloudFailure {
		t.Fatalf("cloud must report cloudFailure, got %q", text)
	}

	// Panic handling: local vs cloud
	rPanic := &recorder{mapPanic: "panic map"}
	text, isError = call(t, connect(t, privacy.ChannelLocal, rPanic.deps(dir)), "graph_repo_map",
		map[string]any{"scope": "project/open"})
	if !isError || !strings.Contains(text, "panic map") {
		t.Fatalf("local panic must contain message, got %q", text)
	}

	text, isError = call(t, connect(t, privacy.ChannelCloud, rPanic.deps(dir)), "graph_repo_map",
		map[string]any{"scope": "project/open"})
	if !isError || text != cloudPanic {
		t.Fatalf("cloud panic must be cloudPanic, got %q", text)
	}
}

func TestParseDepthFloorsAndNeverGoesBelowOne(t *testing.T) {
	cases := []struct {
		in   any
		want blast.Depth
	}{
		{0.5, 1}, {2.7, 2}, {3.0, 3}, {-4.0, 1}, {"all", blast.All}, {"FULL", blast.All},
		{"2", 2}, {"0", 1}, {"x", 1}, {nil, 1}, {true, 1},
	}
	for _, c := range cases {
		if got := servegraph.ParseDepth(c.in); got != c.want {
			t.Errorf("parseDepth(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
