package fakeqmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

// world is a fixture with a listed, an empty and an unlisted collection, a
// backlog, and hits in two collections.
func world() *Fixture {
	return &Fixture{
		Collections: map[string][]string{"repo-a": {"notes/a b.md", "x.md"}, "empty": {}},
		Pending:     3,
		Hits: []Hit{
			{Collection: "repo-a", Relative: "notes/a b.md", Line: 3, Score: 0.75, DocID: "#abc123", Title: "A", Snippet: "one\ntwo"},
			{Collection: "repo-b", Relative: "b.md", Line: 1, Score: 0.5, DocID: "#def456", Title: "B"},
			{Collection: "repo-a", Relative: "x.md", Line: 7, Score: 0.25, DocID: "#789abc", Title: "X", Snippet: "x"},
		},
	}
}

func runCLI(f *Fixture, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := f.RunCLI(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestLoadReadsAFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), FixtureName)
	body := `{"collections":{"repo-a":["x.md"]},"pending":2,"status_error":"locked","search_error":"down",` +
		`"hits":[{"collection":"repo-a","relative":"x.md","line":4,"score":0.5,"docid":"#a1","title":"T","snippet":"s"}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := &Fixture{
		Collections: map[string][]string{"repo-a": {"x.md"}},
		Pending:     2,
		StatusError: "locked",
		SearchError: "down",
		Hits:        []Hit{{Collection: "repo-a", Relative: "x.md", Line: 4, Score: 0.5, DocID: "#a1", Title: "T", Snippet: "s"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadWithoutAFileIsAnEngineThatKnowsNothing(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), FixtureName))
	if err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runCLI(f, "ls", "repo-a"); code != 1 || out != "" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestLoadRefusesWhatIsNoFixture(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err == nil {
		t.Error("a directory: want error")
	}
	for name, body := range map[string]string{
		"broken.json":   `{"pending":`,
		"misspelt.json": `{"collection":{}}`,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), path) {
			t.Errorf("%s: want an error naming the file, got %v", name, err)
		}
	}
}

func TestListPrintsTheCollectionAsQmdDoes(t *testing.T) {
	code, out, errOut := runCLI(world(), "ls", "repo-a")
	want := " 1 B  Jan  1 00:00  qmd://repo-a/notes/a b.md\n 1 B  Jan  1 00:00  qmd://repo-a/x.md\n"
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestListOfAnEmptyCollectionSaysSo(t *testing.T) {
	code, out, _ := runCLI(world(), "ls", "empty")
	if code != 0 || out != "No files found in collection: empty\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestListOfAnUnknownCollectionFailsAsQmdDoes(t *testing.T) {
	code, out, errOut := runCLI(world(), "ls", "repo-z")
	want := "Collection not found: repo-z\nRun 'qmd ls' to see available collections.\n"
	if code != 1 || out != "" || errOut != want {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestStatusReportsTheBacklog(t *testing.T) {
	f := world()
	if code, out, _ := runCLI(f, "status"); code != 0 || out != "Documents\n  Pending:  3 need embedding (run 'qmd embed')\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	f.Pending = 0
	if code, out, _ := runCLI(f, "status"); code != 0 || out != "Documents\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestStatusFailsWithTheScriptedError(t *testing.T) {
	f := world()
	f.StatusError = "index is locked\n"
	code, out, errOut := runCLI(f, "status")
	if code != 1 || out != "" || errOut != "index is locked\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestSearchPrintsTheHitsAsQmdJSON(t *testing.T) {
	code, out, errOut := runCLI(world(), "query", "q", "--json", "-n", "1", "-c", "repo-a")
	want := `[
  {
    "docid": "#abc123",
    "score": 0.75,
    "file": "qmd://repo-a/notes/a b.md",
    "line": 3,
    "title": "A",
    "snippet": "one\ntwo"
  }
]
`
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestSearchKeepsTheFixtureOrderAcrossTheNamedCollections(t *testing.T) {
	var rows []cliHit
	code, out, _ := runCLI(world(), "vsearch", "q", "-c", "repo-a", "--json", "-c", "repo-b", "-n", "5")
	if err := json.Unmarshal([]byte(out), &rows); code != 0 || err != nil {
		t.Fatalf("code %d, err %v, out %q", code, err, out)
	}
	var files []string
	for _, row := range rows {
		files = append(files, row.File)
	}
	want := []string{"qmd://repo-a/notes/a b.md", "qmd://repo-b/b.md", "qmd://repo-a/x.md"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("files %v", files)
	}
	if strings.Count(out, `"snippet"`) != 2 {
		t.Errorf("an empty snippet is left out, as qmd does: %s", out)
	}
}

func TestSearchWithoutACollectionSearchesAllAndStopsAtN(t *testing.T) {
	var rows []cliHit
	_, out, _ := runCLI(world(), "search", "q", "--json", "-n", "2")
	if err := json.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 2 || rows[1].File != "qmd://repo-b/b.md" {
		t.Fatalf("err %v, rows %+v", err, rows)
	}
}

func TestSearchWithoutHitsPrintsAnEmptyArray(t *testing.T) {
	if code, out, _ := runCLI(world(), "search", "q", "--json", "-n", "5", "-c", "nowhere"); code != 0 || out != "[]\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// An engine that fails the search fails every search verb, and still refuses
// a call neither port makes before it fails.
func TestSearchFailsWithTheScriptedError(t *testing.T) {
	f := world()
	f.SearchError = "the index is locked\n"
	for _, verb := range []string{"search", "vsearch", "query"} {
		code, out, errOut := runCLI(f, verb, "q", "--json", "-n", "5", "-c", "repo-a")
		if code != 1 || out != "" || errOut != "the index is locked\n" {
			t.Errorf("%s: code %d, out %q, err %q", verb, code, out, errOut)
		}
	}
	if code, _, errOut := runCLI(f, "search", "q", "-n", "5"); code != 2 || !strings.HasPrefix(errOut, "fakeqmd: ") {
		t.Errorf("unexpected call: code %d, err %q", code, errOut)
	}
}

func TestRunCLIRefusesACallNeitherPortMakes(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"collection", "add"},
		{"embed", "-c", "repo-a"},
		{"update", "repo-a"},
		{"ls"},
		{"ls", "a", "b"},
		{"status", "--json"},
		{"search"},
		{"search", "q", "-n", "5"},
		{"search", "q", "--json"},
		{"search", "q", "--json", "-n"},
		{"search", "q", "--json", "-n", "0"},
		{"search", "q", "--json", "-n", "x"},
		{"search", "q", "--json", "-n", "5", "--full"},
	} {
		code, out, errOut := runCLI(world(), args...)
		if code != 2 || out != "" || !strings.HasPrefix(errOut, "fakeqmd: ") {
			t.Errorf("%q: code %d, out %q, err %q", args, code, out, errOut)
		}
	}
}

// post sends one JSON-RPC body and decodes the reply.
func post(t *testing.T, url, body string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var reply map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, reply
}

// queryResults digs structuredContent.results out of a query reply.
func queryResults(reply map[string]any) []any {
	result, _ := reply["result"].(map[string]any)
	content, _ := result["structuredContent"].(map[string]any)
	results, _ := content["results"].([]any)
	return results
}

func TestMCPHandlerAnswersTheHandshake(t *testing.T) {
	server := httptest.NewServer(world().MCPHandler())
	defer server.Close()
	_, reply := post(t, server.URL+"/mcp", `{"jsonrpc":"2.0","id":7,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	result, _ := reply["result"].(map[string]any)
	if reply["id"] != float64(7) || result["protocolVersion"] != "2025-06-18" {
		t.Fatalf("reply %v", reply)
	}
}

// qmd 2.8.3 numbers every snippet line on the MCP path, starting at the hit's
// line (dist/mcp/server.js:301, addLineNumbers in dist/store.js); the CLI's
// --json leaves the snippet alone.
func TestMCPHandlerAnswersTheQueryTool(t *testing.T) {
	server := httptest.NewServer(world().MCPHandler())
	defer server.Close()
	_, reply := post(t, server.URL, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"query","arguments":{"query":"q","collections":["repo-a"],"limit":1}}}`)
	results := queryResults(reply)
	if len(results) != 1 {
		t.Fatalf("reply %v", reply)
	}
	want := map[string]any{"docid": "#abc123", "file": "repo-a/notes/a b.md", "title": "A", "score": 0.75, "line": float64(3), "snippet": "3: one\n4: two"}
	if !reflect.DeepEqual(results[0], want) {
		t.Fatalf("hit %v", results[0])
	}
	_, reply = post(t, server.URL, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"query","arguments":{"query":"q","collections":["repo-b"]}}}`)
	results = queryResults(reply)
	if len(results) != 1 {
		t.Fatalf("reply %v", reply)
	}
	if hit, _ := results[0].(map[string]any); hit["snippet"] != "1: " {
		t.Fatalf("an empty snippet is numbered too: %v", results[0])
	}
}

func TestMCPHandlerGivesTenHitsWithoutALimit(t *testing.T) {
	f := &Fixture{}
	for i := range 12 {
		f.Hits = append(f.Hits, Hit{Collection: "c", Relative: fmt.Sprintf("%d.md", i), DocID: "#0"})
	}
	server := httptest.NewServer(f.MCPHandler())
	defer server.Close()
	_, reply := post(t, server.URL, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"query","arguments":{"query":"q"}}}`)
	if results := queryResults(reply); len(results) != 10 {
		t.Fatalf("reply %v", reply)
	}
}

// The engine answers the handshake and fails the search, as a qmd whose index
// breaks under the query does.
func TestMCPHandlerFailsTheQueryWithTheScriptedError(t *testing.T) {
	f := world()
	f.SearchError = "the index is locked\n"
	server := httptest.NewServer(f.MCPHandler())
	defer server.Close()
	_, reply := post(t, server.URL, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"query","arguments":{"query":"q"}}}`)
	rpcErr, _ := reply["error"].(map[string]any)
	if reply["id"] != float64(4) || reply["result"] != nil || rpcErr["code"] != float64(-32603) || rpcErr["message"] != "the index is locked\n" {
		t.Fatalf("reply %v", reply)
	}
	_, reply = post(t, server.URL, `{"jsonrpc":"2.0","id":5,"method":"initialize","params":{}}`)
	if _, ok := reply["result"].(map[string]any); !ok {
		t.Fatalf("handshake %v", reply)
	}
}

func TestMCPHandlerRefusesEveryOtherCall(t *testing.T) {
	server := httptest.NewServer(world().MCPHandler())
	defer server.Close()
	for body, code := range map[string]float64{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get","arguments":{}}}`: -32601,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`:                                        -32601,
		`{`: -32700,
	} {
		_, reply := post(t, server.URL, body)
		rpcErr, _ := reply["error"].(map[string]any)
		if rpcErr["code"] != code || !strings.HasPrefix(rpcErr["message"].(string), "fakeqmd: ") {
			t.Errorf("%s: reply %v", body, reply)
		}
	}
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET: status %d", resp.StatusCode)
	}
}

// mcpPort is loomux's MCP port, connected to a fake served at url.
func mcpPort(url string) *search.QmdMcpPort {
	return search.NewQmdMcpPort(search.WithConnect(func(map[string]string) (search.Session, error) {
		return &search.HTTPSession{URL: url}, nil
	}))
}

// cliPort is loomux's CLI port, answered by f through the Runner seam as the
// fake binary answers it.
func cliPort(f *Fixture) *search.QmdPort {
	return &search.QmdPort{Executable: "qmd", Runner: func(argv []string) ([]byte, []byte, int, error) {
		var out, errb bytes.Buffer
		code := f.RunCLI(argv[1:], &out, &errb)
		return out.Bytes(), errb.Bytes(), code, nil
	}}
}

// The fake is only worth its fixture if loomux's own ports read it: the MCP
// port over HTTP, the CLI port through its Runner seam. The MCP snippets
// arrive numbered; the port hands them on as the CLI port does, so this test
// also holds the port's removal of qmd's line prefix.
func TestLoomuxsPortsReadTheFake(t *testing.T) {
	f := world()
	server := httptest.NewServer(f.MCPHandler())
	defer server.Close()
	want := []search.SearchHit{
		{Collection: "repo-a", Relative: "notes/a b.md", Line: 3, Title: "A", Snippet: "one\ntwo", Score: 0.75, ContentKey: "#abc123"},
		{Collection: "repo-a", Relative: "x.md", Line: 7, Title: "X", Snippet: "x", Score: 0.25, ContentKey: "#789abc"},
	}
	hits, err := mcpPort(server.URL).Search("q", []string{"repo-a"}, search.ProfileFull, 5)
	if err != nil || !reflect.DeepEqual(hits, want) {
		t.Fatalf("MCP: err %v, hits %+v", err, hits)
	}

	cli := cliPort(f)
	if hits, err := cli.Search("q", []string{"repo-a"}, search.ProfileKeyword, 5); err != nil || !reflect.DeepEqual(hits, want) {
		t.Fatalf("CLI search: err %v, hits %+v", err, hits)
	}
	if indexed, err := cli.Indexed("repo-a"); err != nil || !reflect.DeepEqual(indexed, []string{"notes/a b.md", "x.md"}) {
		t.Fatalf("indexed %v, err %v", indexed, err)
	}
	if indexed, err := cli.Indexed("empty"); err != nil || len(indexed) != 0 {
		t.Fatalf("empty: indexed %v, err %v", indexed, err)
	}
	if pending, err := cli.NotYetSearchable(); err != nil || pending != 3 {
		t.Fatalf("pending %d, err %v", pending, err)
	}
	// The same bytes the Python reference builds from the same stderr.
	_, err = cli.Indexed("repo-z")
	if err == nil || err.Error() != "qmd exited with 1: Collection not found: repo-z\nRun 'qmd ls' to see available collections." {
		t.Fatalf("err %v", err)
	}
}

// A failing engine fails both ports with its own words, never with an empty
// answer.
func TestLoomuxsPortsFailWhenTheFakeFailsTheSearch(t *testing.T) {
	f := world()
	f.SearchError = "the index is locked\n"
	server := httptest.NewServer(f.MCPHandler())
	defer server.Close()
	hits, err := mcpPort(server.URL).Search("q", []string{"repo-a"}, search.ProfileFull, 5)
	if err == nil || hits != nil || !strings.Contains(err.Error(), "the index is locked") {
		t.Fatalf("MCP: hits %+v, err %v", hits, err)
	}
	// The same bytes the Python reference builds from the same stderr.
	hits, err = cliPort(f).Search("q", []string{"repo-a"}, search.ProfileFull, 5)
	if err == nil || hits != nil || err.Error() != "qmd exited with 1: the index is locked" {
		t.Fatalf("CLI: hits %+v, err %v", hits, err)
	}
}

func TestMainNeedsTheFixtureVariable(t *testing.T) {
	var out, errb bytes.Buffer
	code := Main([]string{"status"}, func(string) string { return "" }, &out, &errb)
	if code != 2 || errb.String() != "fakeqmd: LOOMUX_FAKE_QMD_FIXTURE is not set\n" {
		t.Fatalf("code %d, err %q", code, errb.String())
	}
}

func TestMainAnswersFromTheNamedFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), FixtureName)
	if err := os.WriteFile(path, []byte(`{"pending":4}`), 0o644); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == FixtureEnv {
			return path
		}
		return ""
	}
	var out, errb bytes.Buffer
	if code := Main([]string{"status"}, getenv, &out, &errb); code != 0 || out.String() != "Documents\n  Pending:  4 need embedding (run 'qmd embed')\n" {
		t.Fatalf("code %d, out %q, err %q", code, out.String(), errb.String())
	}
}

func TestMainReportsAFixtureItCannotRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), FixtureName)
	if err := os.WriteFile(path, []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Main([]string{"status"}, func(string) string { return path }, &out, &errb)
	if code != 2 || !strings.HasPrefix(errb.String(), "fakeqmd: "+path) {
		t.Fatalf("code %d, err %q", code, errb.String())
	}
}

func TestRunCLIAnswersTheTwoWritingCalls(t *testing.T) {
	for _, verb := range []string{"update", "embed"} {
		if code, out, errOut := runCLI(world(), verb); code != 0 || out != "" || errOut != "" {
			t.Errorf("%s: code %d, out %q, err %q", verb, code, out, errOut)
		}
	}
}

func TestRunLogsTheWritingCallsBesideTheFixture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FixtureName)
	if err := os.WriteFile(path, []byte(`{"pending":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"update"}, {"status"}, {"embed"}, {"embed", "-c", "x"}} {
		var out, errb bytes.Buffer
		Run(path, args, &out, &errb)
	}
	got, err := os.ReadFile(filepath.Join(dir, CallLogName))
	if err != nil {
		t.Fatal(err)
	}
	// A read is no change to the engine and a refused call changed nothing.
	if string(got) != "update\nembed\n" {
		t.Fatalf("log %q", got)
	}
}

func TestRunReportsALogItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FixtureName)
	// A directory where the log belongs cannot be appended to.
	if err := os.Mkdir(filepath.Join(dir, CallLogName), 0o755); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := Run(path, []string{"update"}, &out, &errb); code != 2 || !strings.HasPrefix(errb.String(), "fakeqmd: ") {
		t.Fatalf("code %d, err %q", code, errb.String())
	}
}
