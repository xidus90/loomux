// Package fakeqmd stands in for qmd while the corpus of stage 1b-1 is recorded
// and replayed. One fixture per world says which documents each collection
// holds, how many wait for embedding and what a search finds; the package
// answers qmd's command line and its MCP query tool from it, so the Python
// reference and loomux ask the same engine and no model or GPU is involved.
package fakeqmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// FixtureName is the fixture's file name in the root of a world.
const FixtureName = "qmd-fixture.json"

// FixtureEnv names the fixture the fake qmd binary answers from.
const FixtureEnv = "LOOMUX_FAKE_QMD_FIXTURE"

// mcpDefaultLimit is what qmd 2.8.3's MCP query returned without a limit in
// the spike of 2026-09-15: ten hits.
const mcpDefaultLimit = 10

// Hit is one search result of a fixture, in the engine's ranking order.
type Hit struct {
	Collection string  `json:"collection"`
	Relative   string  `json:"relative"`
	Line       int     `json:"line"`
	Score      float64 `json:"score"`
	DocID      string  `json:"docid"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet"`
}

// Fixture is the engine's whole state as a world declares it.
type Fixture struct {
	Collections map[string][]string `json:"collections"`  // qmd ls <c>
	Pending     int                 `json:"pending"`      // qmd status
	StatusError string              `json:"status_error"` // not empty: qmd status exits 1 with this stderr
	SearchError string              `json:"search_error"` // not empty: every search exits 1 with this stderr, MCP query is a JSON-RPC error
	Hits        []Hit               `json:"hits"`         // search|vsearch|query --json and MCP query
}

// Load reads a fixture. A world without one has an engine that knows nothing.
func Load(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Fixture{}, nil
	}
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	// A misspelt key would leave the engine silently empty, and a recording
	// would pin that emptiness as the reference's answer.
	decoder.DisallowUnknownFields()
	fixture := &Fixture{}
	if err := decoder.Decode(fixture); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return fixture, nil
}

// RunCLI answers the qmd command lines both CLI ports run and refuses every
// other one with exit code 2.
func (f *Fixture) RunCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fakeqmd: subcommand required")
		return 2
	}
	switch args[0] {
	case "ls":
		return f.list(args[1:], stdout, stderr)
	case "status":
		return f.status(args[1:], stdout, stderr)
	case "search", "vsearch", "query":
		return f.search(args[1:], stdout, stderr)
	case "update", "embed":
		// The two calls the index commands make. qmd 2.8.3 takes no argument
		// for either as brain and loomux call them, and their output is read
		// by neither, so the fake says nothing.
		if len(args) != 1 {
			fmt.Fprintf(stderr, "fakeqmd: %s takes no arguments\n", args[0])
			return 2
		}
		return 0
	}
	fmt.Fprintf(stderr, "fakeqmd: unknown subcommand %q\n", args[0])
	return 2
}

// list prints a collection as qmd ls does. Both readers take only the qmd://
// location from a row, so size and date are fixed.
func (f *Fixture) list(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "fakeqmd: ls takes exactly one collection")
		return 2
	}
	relatives, ok := f.Collections[args[0]]
	if !ok {
		// qmd 2.8.3, measured 2026-09-15: exit 1, nothing on stdout, two lines on stderr.
		fmt.Fprintf(stderr, "Collection not found: %s\nRun 'qmd ls' to see available collections.\n", args[0])
		return 1
	}
	if len(relatives) == 0 {
		// qmd 2.8.3 for a collection without documents (dist/cli/qmd.js:1491).
		fmt.Fprintf(stdout, "No files found in collection: %s\n", args[0])
		return 0
	}
	for _, relative := range relatives {
		fmt.Fprintf(stdout, " 1 B  Jan  1 00:00  qmd://%s/%s\n", args[0], relative)
	}
	return 0
}

// status prints the Documents block of qmd status, reduced to the line both
// readers look for; qmd leaves that line out when nothing is pending.
func (f *Fixture) status(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "fakeqmd: status takes no arguments")
		return 2
	}
	if f.StatusError != "" {
		fmt.Fprint(stderr, f.StatusError)
		return 1
	}
	fmt.Fprintln(stdout, "Documents")
	if f.Pending > 0 {
		fmt.Fprintf(stdout, "  Pending:  %d need embedding (run 'qmd embed')\n", f.Pending)
	}
	return 0
}

// cliHit is one row of qmd's --json output, keys in qmd's order.
type cliHit struct {
	DocID   string  `json:"docid"`
	Score   float64 `json:"score"`
	File    string  `json:"file"`
	Line    int     `json:"line"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet,omitempty"`
}

// search prints the hits of the named collections as qmd --json does: an
// array, indented by two, empty as [] and never as null.
func (f *Fixture) search(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fakeqmd: a query is required")
		return 2
	}
	limit, collections, ok := searchFlags(args[1:], stderr)
	if !ok {
		return 2
	}
	if f.SearchError != "" {
		fmt.Fprint(stderr, f.SearchError)
		return 1
	}
	rows := []cliHit{}
	for _, hit := range f.hitsIn(collections, limit) {
		rows = append(rows, cliHit{
			DocID:   hit.DocID,
			Score:   hit.Score,
			File:    "qmd://" + hit.Collection + "/" + hit.Relative,
			Line:    hit.Line,
			Title:   hit.Title,
			Snippet: hit.Snippet,
		})
	}
	data, _ := json.MarshalIndent(rows, "", "  ")
	fmt.Fprintf(stdout, "%s\n", data)
	return 0
}

// searchFlags reads the switches both CLI ports pass: --json, -n N and any
// number of -c COLLECTION. The fake answers in JSON only, so --json and -n are
// required.
func searchFlags(args []string, stderr io.Writer) (int, []string, bool) {
	limit, asJSON := 0, false
	var collections []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" {
			asJSON = true
			continue
		}
		if (arg != "-n" && arg != "-c") || i+1 == len(args) {
			fmt.Fprintf(stderr, "fakeqmd: unexpected argument %q\n", arg)
			return 0, nil, false
		}
		i++
		if arg == "-c" {
			collections = append(collections, args[i])
			continue
		}
		n, err := strconv.Atoi(args[i])
		if err != nil || n < 1 {
			fmt.Fprintf(stderr, "fakeqmd: -n needs a positive number, got %q\n", args[i])
			return 0, nil, false
		}
		limit = n
	}
	if !asJSON || limit == 0 {
		fmt.Fprintln(stderr, "fakeqmd: search needs --json and -n")
		return 0, nil, false
	}
	return limit, collections, true
}

// hitsIn keeps the fixture's order as the engine's ranking. No collection
// named means every collection, as qmd's MCP query answered in the spike.
func (f *Fixture) hitsIn(collections []string, limit int) []Hit {
	var found []Hit
	for _, hit := range f.Hits {
		if len(found) == limit {
			break
		}
		if len(collections) == 0 || slices.Contains(collections, hit.Collection) {
			found = append(found, hit)
		}
	}
	return found
}

// addLineNumbers numbers every line of a snippet from start on, as qmd 2.8.3
// does on the MCP path only (dist/store.js addLineNumbers, called at
// dist/mcp/server.js:301): split at \n, an empty snippet is one empty line.
func addLineNumbers(text string, start int) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strconv.Itoa(start+i) + ": " + line
	}
	return strings.Join(lines, "\n")
}

// rpcRequest is the part of a JSON-RPC call the handler reads.
type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params struct {
		Name      string `json:"name"`
		Arguments struct {
			Collections []string `json:"collections"`
			Limit       *int     `json:"limit"`
		} `json:"arguments"`
	} `json:"params"`
}

// mcpHit is one entry of the query tool's structuredContent.results.
type mcpHit struct {
	DocID   string  `json:"docid"`
	File    string  `json:"file"`
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	Line    int     `json:"line"`
	Snippet string  `json:"snippet"`
}

// MCPHandler answers the two calls loomux's MCP port makes, the initialize
// handshake and the query tool. Every other call gets a JSON-RPC error, so a
// port that asks for more fails loudly in the corpus instead of reading
// nothing.
func (f *Fixture) MCPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			writeJSON(w, rpcError(nil, -32600, "fakeqmd: only POST is served"))
			return
		}
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, rpcError(nil, -32700, "fakeqmd: "+err.Error()))
			return
		}
		switch {
		case req.Method == "initialize":
			writeJSON(w, rpcResult(req.ID, map[string]any{
				"protocolVersion": "2025-06-18",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "fakeqmd", "version": "0"},
			}))
		case req.Method == "tools/call" && req.Params.Name == "query" && f.SearchError != "":
			// The engine is reachable and fails the search itself.
			writeJSON(w, rpcError(req.ID, -32603, f.SearchError))
		case req.Method == "tools/call" && req.Params.Name == "query":
			limit := mcpDefaultLimit
			if req.Params.Arguments.Limit != nil {
				limit = *req.Params.Arguments.Limit
			}
			results := []mcpHit{}
			for _, hit := range f.hitsIn(req.Params.Arguments.Collections, limit) {
				results = append(results, mcpHit{
					DocID:   hit.DocID,
					File:    hit.Collection + "/" + hit.Relative,
					Title:   hit.Title,
					Score:   hit.Score,
					Line:    hit.Line,
					Snippet: addLineNumbers(hit.Snippet, hit.Line),
				})
			}
			writeJSON(w, rpcResult(req.ID, map[string]any{
				"structuredContent": map[string]any{"results": results},
			}))
		default:
			writeJSON(w, rpcError(req.ID, -32601, fmt.Sprintf("fakeqmd: no answer for %s %s", req.Method, req.Params.Name)))
		}
	})
}

func rpcResult(id json.RawMessage, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func rpcError(id json.RawMessage, code int, message string) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}}
}

// writeJSON ends a reply; a client that hung up has nobody left to tell.
func writeJSON(w io.Writer, v any) {
	_ = json.NewEncoder(w).Encode(v)
}

// CallLogName is the file beside the fixture that every answered `update` and
// `embed` is appended to, one command line per line.
//
// Those two calls change the engine and print nothing either side reads, so
// without a trace a command that never made them would answer exactly like
// one that did. Written into the world, the trace is part of what a recording
// pins and a replay has to reproduce. Reads leave none: they change nothing,
// and a trace of them would give every read-only recording a world_after.
const CallLogName = "qmd-calls.log"

// Run answers one command line from the fixture at path and logs it beside
// the fixture when it was a writing call the engine accepted.
func Run(path string, args []string, stdout, stderr io.Writer) int {
	fixture, err := Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "fakeqmd: %v\n", err)
		return 2
	}
	code := fixture.RunCLI(args, stdout, stderr)
	if code != 0 || (args[0] != "update" && args[0] != "embed") {
		return code
	}
	if err := appendCall(filepath.Join(filepath.Dir(path), CallLogName), args); err != nil {
		fmt.Fprintf(stderr, "fakeqmd: %v\n", err)
		return 2
	}
	return 0
}

// appendCall adds one command line to the log. The write's own error is left
// to Close: a file opened for appending takes a line or fails to flush it.
func appendCall(path string, args []string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	fmt.Fprintln(file, strings.Join(args, " "))
	return file.Close()
}

// Main is the fake qmd binary: it answers one command line from the fixture
// FixtureEnv names.
func Main(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	path := getenv(FixtureEnv)
	if path == "" {
		fmt.Fprintf(stderr, "fakeqmd: %s is not set\n", FixtureEnv)
		return 2
	}
	return Run(path, args, stdout, stderr)
}
