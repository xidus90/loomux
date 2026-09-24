// Package graph registers the code-graph tools on an MCP server.
//
// The same thin layer as serve/brain: one tool call, one query call, and the
// results turned into what MCP has for them. The area comes from the registry
// through the channel's eyes, never from a path the caller names.
package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/mcptools"
)

// Deps are what the tools need. A test replaces them and needs no
// tree; serve passes query's functions.
type Deps struct {
	RegistryDir string
	LegacyDir   string
	Ask         func(root, question string, opts query.AskOptions) (ask.Answer, []string, error)
	Check       func(root string) (query.Drift, error)
	Callers     func(root, symbol string, opts query.CallersOptions) (query.CallersAnswer, []string, error)
	Skeleton    func(root, file string, opts query.SkeletonOptions) (query.SkeletonAnswer, []string, error)
	Grep        func(root, pattern string, opts query.GrepOptions) (query.GrepAnswer, []string, error)
	Map         func(root string, opts query.MapOptions) (query.MapAnswer, []string, error)
	Blast       func(root string, opts query.BlastOptions) (query.BlastAnswer, []string, error)
}

// mcpLimit is the reference's MCP default (src/mcp/tools.ts), not the command
// line's 8.
const mcpLimit = 5

// Register adds the graph tools to server. The channel is the listener's.
func Register(server *mcp.Server, channel privacy.Channel, deps Deps) {
	handlers := map[string]mcp.ToolHandler{
		"graph_find_code":       findCode(channel, deps),
		"graph_file_api":        fileApi(channel, deps),
		"graph_trace_calls":     traceCalls(channel, deps),
		"graph_find_all":        findAll(channel, deps),
		"graph_repo_map":        repoMap(channel, deps),
		"graph_blast":           graphBlast(channel, deps),
		"graph_check_freshness": checkFreshness(channel, deps),
	}
	for _, tool := range mcptools.Graph() {
		server.AddTool(tool, guarded(channel, handlers[tool.Name]))
	}
}

func findCode(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		scope, question := str(args, "scope"), str(args, "query")
		// Both arguments are refused before the scope is resolved.
		if scope == "" {
			return failure("graph_find_code requires a scope"), nil
		}
		if question == "" {
			return failure("graph_find_code requires a query"), nil
		}
		area, refusal := resolve("graph_find_code", channel, deps, scope)
		if refusal != nil {
			return refusal, nil
		}
		answer, notes, err := deps.Ask(area.Area.Path, question, query.AskOptions{
			Limit: limit(args), In: str(args, "in"), Source: true, Full: flag(args, "full"),
			Keep: readable(area.Manifest),
		})
		if channel == privacy.ChannelCloud {
			// A refresh note counts every file the refresh saw, those under
			// the never globs too, and a failed rebuild's note quotes the
			// parse error of a hidden file. The cloud channel hears none of
			// them, neither in the text nor as progress.
			notes = nil
		}
		for _, note := range notes {
			report(ctx, req, note)
		}
		if err != nil {
			return failure(withNotes(notes, errorText(channel, err))), nil
		}
		return success(withNotes(notes, strings.TrimSuffix(query.AskReport(answer), "\n"))), nil
	}
}

func fileApi(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		scope, file := str(args, "scope"), str(args, "file")
		if scope == "" {
			return failure("graph_file_api requires a scope"), nil
		}
		if file == "" {
			return failure("graph_file_api requires a file"), nil
		}
		area, refusal := resolve("graph_file_api", channel, deps, scope)
		if refusal != nil {
			return refusal, nil
		}
		answer, notes, err := deps.Skeleton(area.Area.Path, file, query.SkeletonOptions{
			Keep: readable(area.Manifest),
		})
		if channel == privacy.ChannelCloud {
			notes = nil
		}
		for _, note := range notes {
			report(ctx, req, note)
		}
		if err != nil {
			return failure(withNotes(notes, errorText(channel, err))), nil
		}
		return success(withNotes(notes, strings.TrimSuffix(query.SkeletonReport(answer), "\n"))), nil
	}
}

func traceCalls(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		scope, symbol := str(args, "scope"), str(args, "symbol")
		if scope == "" {
			return failure("graph_trace_calls requires a scope"), nil
		}
		if symbol == "" {
			return failure("graph_trace_calls requires a symbol"), nil
		}
		area, refusal := resolve("graph_trace_calls", channel, deps, scope)
		if refusal != nil {
			return refusal, nil
		}
		var dir blast.Direction
		if str(args, "direction") == "out" {
			dir = blast.Out
		} else {
			dir = blast.In
		}
		answer, notes, err := deps.Callers(area.Area.Path, symbol, query.CallersOptions{
			Direction: dir,
			Depth:     parseDepth(args["depth"]),
			In:        str(args, "in"),
			Keep:      readable(area.Manifest),
		})
		if channel == privacy.ChannelCloud {
			notes = nil
		}
		for _, note := range notes {
			report(ctx, req, note)
		}
		if err != nil {
			return failure(withNotes(notes, errorText(channel, err))), nil
		}
		return success(withNotes(notes, strings.TrimSuffix(query.CallersReport(answer), "\n"))), nil
	}
}

func findAll(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		scope, pattern := str(args, "scope"), str(args, "pattern")
		if scope == "" {
			return failure("graph_find_all requires a scope"), nil
		}
		if pattern == "" {
			return failure("graph_find_all requires a pattern"), nil
		}
		area, refusal := resolve("graph_find_all", channel, deps, scope)
		if refusal != nil {
			return refusal, nil
		}
		answer, notes, err := deps.Grep(area.Area.Path, pattern, query.GrepOptions{
			In:         str(args, "in"),
			IgnoreCase: flag(args, "ignore_case"),
			Fixed:      flag(args, "fixed"),
			Keep:       readable(area.Manifest),
		})
		if channel == privacy.ChannelCloud {
			notes = nil
		}
		for _, note := range notes {
			report(ctx, req, note)
		}
		if err != nil {
			return failure(withNotes(notes, errorText(channel, err))), nil
		}
		return success(withNotes(notes, strings.TrimSuffix(query.GrepReport(answer), "\n"))), nil
	}
}

func repoMap(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		scope := str(args, "scope")
		if scope == "" {
			return failure("graph_repo_map requires a scope"), nil
		}
		area, refusal := resolve("graph_repo_map", channel, deps, scope)
		if refusal != nil {
			return refusal, nil
		}
		maxDirs := 16
		if n, ok := args["max_dirs"].(float64); ok && n >= 1 {
			maxDirs = int(n)
		}
		answer, notes, err := deps.Map(area.Area.Path, query.MapOptions{
			MaxDirs: maxDirs,
			Keep:    readable(area.Manifest),
		})
		if channel == privacy.ChannelCloud {
			notes = nil
		}
		for _, note := range notes {
			report(ctx, req, note)
		}
		if err != nil {
			return failure(withNotes(notes, errorText(channel, err))), nil
		}
		return success(withNotes(notes, strings.TrimSuffix(query.MapReport(answer), "\n"))), nil
	}
}

// graphBlast has no refusal of its own before resolve: scope is its only
// required argument, and resolve refuses an empty one with the same text.
// git runs in the area's root; the never globs reach query as Keep, which
// counts a refused changed file or hit instead of naming it.
func graphBlast(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		area, refusal := resolve("graph_blast", channel, deps, str(args, "scope"))
		if refusal != nil {
			return refusal, nil
		}
		answer, notes, err := deps.Blast(area.Area.Path, query.BlastOptions{
			Base:  str(args, "base"),
			Depth: parseDepth(args["depth"]),
			Keep:  readable(area.Manifest),
		})
		if channel == privacy.ChannelCloud {
			notes = nil
		}
		for _, note := range notes {
			report(ctx, req, note)
		}
		if err != nil {
			return failure(withNotes(notes, errorText(channel, err))), nil
		}
		return success(withNotes(notes, strings.TrimSuffix(query.BlastReport(answer), "\n"))), nil
	}
}

// checkFreshness reports drift without refreshing first: a refresh would make
// it answer about a graph it had just repaired, which is always OK
// (NO_REFRESH_TOOLS in src/mcp/tools.ts).
func checkFreshness(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		area, refusal := resolve("graph_check_freshness", channel, deps, str(arguments(req), "scope"))
		if refusal != nil {
			return refusal, nil
		}
		drift, err := deps.Check(area.Area.Path)
		if err != nil {
			return failure(errorText(channel, err)), nil
		}
		drift = drift.Only(readable(area.Manifest))
		if channel == privacy.ChannelCloud {
			// The count alone tells a remote model that something under the
			// never globs changed. Locally that is the user's own business;
			// on the cloud channel it is a disclosure. The report still says
			// DRIFT rather than OK, because OK would be a lie about the graph.
			// So a DRIFT with no visible entry still tells the cloud that
			// something hidden changed, only not what or how much; the user
			// accepted that on 2026-09-19 (spec section 3.4): DRIFT, never OK.
			drift.Hidden = 0
		}
		return success(strings.TrimSuffix(query.CheckReport(drift), "\n")), nil
	}
}

// resolve turns a scope into the one area the channel may see, before anything
// is read. A hidden area and an unknown one get the same answer: telling a
// cloud caller that an area exists would disclose what local_only hides.
//
// The visible areas are asked for under "all", so that every error VisibleAreas
// can still return is one of the configuration -- a registry or a manifest that
// does not read, named by its absolute path -- and goes through errorText like
// any other read failure. The unknown scope is Single's answer, over the same
// visible areas VisibleAreas would have named.
func resolve(tool string, channel privacy.Channel, deps Deps, scope string) (privacy.VisibleArea, *mcp.CallToolResult) {
	if scope == "" {
		return privacy.VisibleArea{}, failure(tool + " requires a scope")
	}
	areas, err := privacy.VisibleAreas(deps.RegistryDir, deps.LegacyDir, "all", channel)
	if err != nil {
		return privacy.VisibleArea{}, failure(errorText(channel, err))
	}
	area, err := privacy.Single(areas, scope)
	if err != nil {
		return privacy.VisibleArea{}, failure(err.Error())
	}
	return area, nil
}

// cloudFailure is what the cloud channel reads of every read failure but a
// missing graph: of the configuration, of Ask and of Check.
const cloudFailure = "the graph could not be read on this channel; ask on the local channel for details"

// errorText is what a caller on channel reads of a read failure. Locally it is
// the error itself. On the cloud channel it is a fixed text: a parse error
// names a hidden file with its line and token, and a registry, manifest or
// graph read error names an absolute path on this machine. A missing graph keeps its own text,
// which names neither and says what to do; whatever it came wrapped in stays
// behind.
func errorText(channel privacy.Channel, err error) string {
	switch {
	case channel != privacy.ChannelCloud:
		return err.Error()
	case errors.Is(err, query.ErrNoGraph):
		return query.ErrNoGraph.Error()
	}
	return cloudFailure
}

// readable is the area's never globs as the predicate query takes.
func readable(manifest *config.Manifest) func(string) bool {
	return func(path string) bool { return privacy.IsReadable(manifest, path) }
}

// withNotes puts the notes in front of the text, as the reference does
// (`${note}\n${res.text}`): a rebuild note explains the answer, so it comes
// first. serve/brain appends its findings instead, because a finding devalues
// an answer; two kinds of note, two places.
func withNotes(notes []string, text string) string {
	if len(notes) == 0 {
		return text
	}
	return strings.Join(notes, "\n") + "\n" + text
}

// cloudPanic is what the cloud channel reads of a panic.
const cloudPanic = "internal error; ask on the local channel for details"

// guarded turns a panic into a tool error. The reference promises the same
// (callTool never throws); a panic here must not take the listener down.
//
// Locally the error carries the panic value. The cloud channel reads a fixed
// text: the value is whatever the failing code held, and that may be a path or
// a line under the never globs.
func guarded(channel privacy.Channel, next mcp.ToolHandler) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (res *mcp.CallToolResult, err error) {
		defer func() {
			if r := recover(); r != nil {
				text := fmt.Sprintf("internal error: %v", r)
				if channel == privacy.ChannelCloud {
					text = cloudPanic
				}
				res, err = failure(text), nil
			}
		}()
		return next(ctx, req)
	}
}

func success(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func failure(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

// limit reads the count, and falls back to the reference's MCP default for a
// missing, non-numeric or below-one one. JSON numbers arrive as float64.
func limit(args map[string]any) int {
	if n, ok := args["limit"].(float64); ok && n >= 1 {
		return int(n)
	}
	return mcpLimit
}

func flag(args map[string]any, key string) bool {
	value, _ := args[key].(bool)
	return value
}

// arguments, str and report are serve/brain's, repeated rather than shared:
// three small functions do not earn a package, and a shared one would couple
// two tool families that otherwise know nothing of each other.
func arguments(req *mcp.CallToolRequest) map[string]any {
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return map[string]any{}
	}
	return args
}

func str(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func report(ctx context.Context, req *mcp.CallToolRequest, message string) {
	token := req.Params.GetProgressToken()
	if token == nil {
		return
	}
	_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token,
		Message:       message,
	})
}

func parseDepth(v any) blast.Depth {
	switch val := v.(type) {
	case string:
		if strings.EqualFold(val, "all") || strings.EqualFold(val, "full") {
			return blast.All
		}
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return blast.Depth(n)
		}
	case float64:
		if n := int(math.Floor(val)); n >= 1 {
			return blast.Depth(n)
		}
	}
	return 1
}
