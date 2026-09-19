// Package brain registers the five knowledge tools on an MCP server.
//
// It is the thinnest possible layer: it maps one tool call to one answer.Run
// and turns the three return values into the three things MCP has for them.
package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/mcptools"
)

// Deps are what the tools need. A test replaces Answer and needs no registry.
//
// Answer is required, not optional: a nil one panics on the first tool call.
// There is no fallback to answer.Run here, because a fallback would only be
// reachable with a registry on disk; the caller that wires serve passes it.
type Deps struct {
	Answer      func(answer.Request, string, string, func(string)) (string, []string, error)
	RegistryDir string
	LegacyDir   string
}

// Register adds the five tools to server. The channel is the listener's, never
// an argument: a channel a caller can name is a claim.
func Register(server *mcp.Server, channel privacy.Channel, deps Deps) {
	for _, tool := range mcptools.Brain() {
		server.AddTool(tool, handler(tool.Name, channel, deps))
	}
}

func handler(name string, channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		cmd := command(name)
		request := answer.Request{
			Command: cmd,
			Query:   str(args, positional(cmd)),
			Scope:   scope(args),
			Profile: profile(args),
			Count:   count(args),
			Section: str(args, "section"),
			Channel: channel,
		}
		notice := func(message string) { report(ctx, req, message) }
		out, notes, err := refuse(args, cmd)
		if err == nil {
			out, notes, err = deps.Answer(request, deps.RegistryDir, deps.LegacyDir, notice)
		}
		if err != nil {
			// What the core refuses is content for the model, not an outage:
			// the model can act on "unknown scope", and reporting it as a
			// protocol error would take that chance away.
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}
		// Both ways, on purpose: as a progress note, which reaches a host that
		// sent a token while the answer is still being built, and in the text,
		// which reaches every host. See withFindings.
		for _, note := range notes {
			report(ctx, req, note)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: withFindings(cmd, forMCP(cmd, out), notes)}},
		}, nil
	}
}

// withFindings puts a search's findings into the answer, where the reference
// puts them.
//
// A finding degrades an answer without invalidating it -- the index is stale,
// the engine answered empty twice -- and it is the reader's only warning that
// the hits below are worth less than they look. The reference writes them into
// the text, one `! {finding}` line under the hits (daemon/tools.py::_search).
//
// loomux sent them as progress notes alone, and a progress note **without a
// progress token from the host has no recipient**: report drops it, which
// TestWithoutAProgressTokenTheHintsLapse pins. A host that sends no token --
// and nothing obliges one to -- therefore lost a warning the reference always
// shows. That is not a difference in presentation, it is a lost warning.
//
// So both: the note still goes up, so a host with a token sees it early rather
// than with the answer, which is better than the reference; and the text
// carries it too, so no host is without it.
//
// search alone, because it is the only answer whose notes are findings about
// that answer. What a note means for the other four is not this function's to
// guess.
func withFindings(command, text string, notes []string) string {
	if command != "search" || len(notes) == 0 {
		return text
	}
	lines := make([]string, 0, len(notes)+1)
	lines = append(lines, text)
	for _, note := range notes {
		lines = append(lines, "! "+note)
	}
	return strings.Join(lines, "\n")
}

// report sends one line upwards as progress, never to stderr: no host reads our
// stderr. Without a progress token from the host the line lapses, and that is
// right -- a message with no recipient is not one.
func report(ctx context.Context, req *mcp.CallToolRequest, message string) {
	token := req.Params.GetProgressToken()
	if token == nil {
		return
	}
	// A failed notification must not fail the answer: the answer stands either
	// way, and the host asked for the answer, not for the commentary.
	_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token,
		Message:       message,
	})
}

// arguments unpacks the raw call arguments. A call without arguments and a call
// with broken ones both end as an empty map: the schema already told the host
// what is required, and a parse error here would be an outage, which this is
// not. Empty arguments need no branch of their own -- unmarshalling nothing
// fails, and the nil map that leaves reads like an empty one.
func arguments(req *mcp.CallToolRequest) map[string]any {
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return map[string]any{}
	}
	return args
}

// command turns brain_search into search. The family prefix is for the host's
// flat tool list; the answer knows the bare names.
func command(name string) string {
	return strings.TrimPrefix(name, "brain_")
}

func str(args map[string]any, key string) string {
	value, ok := args[key].(string)
	if !ok {
		return ""
	}
	return value
}

// positional names the argument that carries the answer's one positional: the
// query for search, the relative path for read and neighbors. The command
// decides, never the caller -- nothing on this path validates arguments against
// the schema, so a read that also carries a query must still read the relative.
func positional(command string) string {
	switch command {
	case "search":
		return "query"
	case "read", "neighbors":
		return "relative"
	}
	return ""
}

// profiles are the three the engine knows, in the order the schema lists them.
// The first is the default, which is what makes the order more than cosmetic.
func profiles() []string { return []string{"fast", "full", "keyword"} }

// profile reads which search to run, and falls back to the cheap one.
//
// The default belongs here for the reason scope's does: the command line gets
// it from its parser (brainargs.go, `fallback: string(search.ProfileFast)`),
// and a tool call has no parser. Without it the empty string travelled all the
// way down -- answer.search hands `Profile(req.Profile)` straight to the port
// (answer.go), and an unknown profile falls into the `default` arm of both
// ports: `query` with `rerank: true` over MCP (search/mcp.go) and the `query`
// subcommand on the command line (search/qmd.go). Both of those are the **full**
// profile. So `brain_search` without a profile ran a rerank search where the
// reference runs fast (daemon/tools.py: `arguments.get("profile",
// Profile.FAST.value)`) and where loomux's own command line runs fast too --
// the slowest of the three, silently, on the front a model uses most.
func profile(args map[string]any) string {
	if given := str(args, "profile"); given != "" {
		return given
	}
	return profiles()[0]
}

// refuse turns back a call the answer must never see, and says nothing about
// one it may.
//
// Nothing below this enforces a schema. The host is told which arguments are
// required and which values a profile may take, and a well-behaved one obeys --
// but no layer on this path holds a call against its schema, and both gaps were
// measured rather than feared:
//
//   - `brain_search` without a query searched for the empty string and came
//     back "no matches", which reads to a model as "nothing found" for a
//     question nobody asked.
//   - `brain_search` with an unknown profile searched anyway, on whatever the
//     engine makes of a word it does not know.
//
// The command line is refused both by its parser; the reference refuses both in
// daemon/tools.py, by reading `arguments["query"]` and by building
// `Profile(...)`. This is the same refusal in loomux's own words.
//
// scope is deliberately not in here although the schema marks it required for
// read and neighbors: the reference defaults it to "all" for every tool and
// then fails on the unknown scope, and `scope` above does the same.
func refuse(args map[string]any, command string) (string, []string, error) {
	if name := positional(command); name != "" && str(args, name) == "" {
		return "", nil, fmt.Errorf("%s is required", name)
	}
	profile := str(args, "profile")
	if profile == "" || slices.Contains(profiles(), profile) {
		return "", nil, nil
	}
	return "", nil, fmt.Errorf("invalid profile %q; choose from %s", profile, strings.Join(profiles(), ", "))
}

// scope reads the area to ask, and falls back to every visible one.
//
// The default lives here and not in answer.Run, because the two fronts reach
// the same answer from opposite directions: the command line gets "all" from
// its parser (brainParserFor), and a tool call has no parser at all. Without
// it, `brain_catalog` with no arguments -- the root catalog, the most ordinary
// call this server has -- asked for the area named by the empty string and was
// refused. The reference does the same in one line
// (daemon/tools.py::_dispatch: arguments.get("scope", "all")).
func scope(args map[string]any) string {
	if given := str(args, "scope"); given != "" {
		return given
	}
	return "all"
}

// forMCP is the answer as a tool call carries it, which is not quite the answer
// as a terminal carries it.
//
// Three of the five answers are built by a formatter that ends its last line,
// because stdout wants a line there. A CallToolResult is not a stream of lines:
// the reference builds the same three by joining, and its text ends with the
// last character of the last line (daemon/tools.py::render_neighbors and
// _search, and "\n".join(core.status(...))). One terminator, removed here and
// nowhere else, is the whole difference.
//
// `read` and `catalog` are not in the list and must not be: what they hand back
// is a document, its own trailing newline included, on both fronts.
func forMCP(command, text string) string {
	switch command {
	case "search", "status", "neighbors":
		return strings.TrimSuffix(text, "\n")
	}
	return text
}

// mcpResultCount is the MCP front's n (daemon/tools.py:167), where the command
// line's parser gives 5 (brainargs.go).
const mcpResultCount = 10

// count reads n. JSON numbers arrive as float64.
//
// The default lives here for the reason scope's and profile's do: nothing on
// this path applies the schema's `default: 10`. Both fronts register through
// the untyped AddTool, which stores no resolved schema, and applySchema --
// the one call that validates arguments and fills defaults -- is reached only
// from the generic mcp.AddTool[In,Out]. The reference therefore carries its
// ten twice, in the schema (daemon/tools.py:53) and in the code
// (daemon/tools.py:167); loomux now carries it in mcptools and here.
//
// Zero is not "not given": search.ExecuteSearch truncates the hits to n, so a
// zero discards every one of them and the model reads "no matches" as an
// answer. An explicit `n: 0` still travels as it is -- what the reference does
// with it is unrecorded.
func count(args map[string]any) int {
	value, ok := args["n"].(float64)
	if !ok {
		return mcpResultCount
	}
	return int(value)
}
