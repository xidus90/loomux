// Package mcptools holds the tool list that serve and the bridge share.
//
// One list in one place: the bridge answers tools/list on its own so that a
// cold qmd start cannot land inside a host's handshake, and a list that drifted
// from serve's would be the worst bug this layer could have.
//
// No loomux code names a protocol revision: the SDK negotiates one upward with
// the host and speaks the one it negotiates downward to serve.
package mcptools

import (
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/config"
)

// CacheTTL and CacheScope let a host cache tools/list. Twelve static tools make
// that free.
const (
	CacheTTL   = 5 * time.Minute
	CacheScope = "public"
)

var (
	once  sync.Once
	tools []*mcp.Tool
)

// brainCount is how many of tools are the brain's; the graph's follow them.
const brainCount = 5

// Brain are the five knowledge tools in their canonical order, the ones
// serve/brain answers.
func Brain() []*mcp.Tool {
	once.Do(build)
	return tools[:brainCount:brainCount]
}

// Graph are the code-graph tools, the ones serve/graph answers.
func Graph() []*mcp.Tool {
	once.Do(build)
	return tools[brainCount:len(tools):len(tools)]
}

// Tools are every tool, as the bridge lists them and serve registers them:
// the brain's five, then the graph's seven.
//
// Built on first use rather than in a package variable: the start floor of
// every loomux invocation, the per-edit hook included, is measured, and a
// schema built at init would be paid by callers that never speak MCP.
//
// The result is read-only by contract, and nothing in the type enforces it:
// Tools, Brain and Graph hand out views of one slice on every call, and the
// scope and relative schemas are one map shared across the tools that take
// them. That shared identity is the point -- it is what lets serve and the
// bridge hold one object instead of two that can drift -- but it also means a
// caller that writes into the slice or into a schema corrupts every other
// reader in the process.
func Tools() []*mcp.Tool {
	once.Do(build)
	return tools
}

// For is the tool list a project with these modules offers: a tool of a
// module that is off is not listed, so the host never sees it and a call to
// it is an unknown tool.
func For(m config.Modules) []*mcp.Tool {
	// Empty, never nil: bridge.Options reads nil as "every tool".
	offered := make([]*mcp.Tool, 0, len(Tools()))
	if m.Brain {
		offered = append(offered, Brain()...)
	}
	if m.Graph {
		offered = append(offered, Graph()...)
	}
	return offered
}

func build() {
	scope := map[string]any{
		"type":        "string",
		"description": "area to ask, or 'all'",
	}
	relative := map[string]any{"type": "string"}
	// One area, never "all": a graph belongs to one repository, and answering
	// across several is federation, which this stage does not have.
	areaScope := map[string]any{"type": "string", "description": "the registered area whose code to ask"}
	tools = []*mcp.Tool{
		{
			Name:        "brain_search",
			Description: "Search the visible areas.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"scope": scope,
					// fast is purely vectorial, full the hybrid chain, keyword
					// BM25 (search/port.py). The prototype's fast/balanced/deep
					// never existed in the reference.
					"profile": map[string]any{"type": "string", "enum": []string{"fast", "full", "keyword"}},
					// Ten, not the command line's five: parity here is with the
					// MCP front (daemon/tools.py), not with cli.py.
					"n": map[string]any{"type": "integer", "default": 10},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "brain_catalog",
			Description: "The root catalog, or one area's.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"scope": scope},
			},
		},
		{
			Name:        "brain_read",
			Description: "Exactly one file, optionally one section of it.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":    scope,
					"relative": relative,
					"section":  map[string]any{"type": "string"},
				},
				"required": []string{"scope", "relative"},
			},
		},
		{
			Name:        "brain_neighbors",
			Description: "Incoming and outgoing links of one page.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":    scope,
					"relative": relative,
				},
				"required": []string{"scope", "relative"},
			},
		},
		{
			Name:        "brain_status",
			Description: "What to know before trusting an answer.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name: "graph_find_code",
			Description: "Query one area's code graph in plain words. Returns ranked " +
				"symbols with exact file:line spans and the relevant source inlined.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope": areaScope,
					"query": map[string]any{"type": "string", "description": "what you want to understand, in plain words"},
					"limit": map[string]any{"type": "integer", "description": "max results (default 5)"},
					"full":  map[string]any{"type": "boolean", "description": "inline whole definition spans instead of the capped excerpt"},
					"in":    map[string]any{"type": "string", "description": "narrow to nodes under this path prefix, filtered before scoring"},
				},
				"required": []string{"scope", "query"},
			},
		},
		{
			Name:        "graph_file_api",
			Description: "Inspect the symbols, signatures, and types declared in one file without reading the whole file body.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope": areaScope,
					"file":  map[string]any{"type": "string", "description": "path to the file, or a unique basename"},
				},
				"required": []string{"scope", "file"},
			},
		},
		{
			Name:        "graph_trace_calls",
			Description: "Trace callers or callees of a symbol along call, reference, and inheritance edges.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":     areaScope,
					"symbol":    map[string]any{"type": "string", "description": "symbol name, Class.method, pkg.Fn, or file path"},
					"direction": map[string]any{"type": "string", "enum": []string{"in", "out"}, "description": "'in' for callers/dependents, 'out' for callees/dependencies"},
					"depth":     map[string]any{"description": "depth limit as integer or 'all' for full transitive closure"},
					"in":        map[string]any{"type": "string", "description": "narrow to nodes under this path prefix"},
				},
				"required": []string{"scope", "symbol"},
			},
		},
		{
			Name:        "graph_find_all",
			Description: "Find all occurrences of a string or regex across indexed files, grouped by enclosing symbol and ranked by symbol coupling (inDegree).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":       areaScope,
					"pattern":     map[string]any{"type": "string", "description": "literal string or regular expression"},
					"in":          map[string]any{"type": "string", "description": "narrow to files under this path prefix"},
					"ignore_case": map[string]any{"type": "boolean", "description": "case-insensitive search"},
					"fixed":       map[string]any{"type": "boolean", "description": "treat pattern as literal string instead of regex"},
				},
				"required": []string{"scope", "pattern"},
			},
		},
		{
			Name:        "graph_repo_map",
			Description: "Generate a compact overview of repository structure, directory clusters, hubs, and hotspots.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":    areaScope,
					"max_dirs": map[string]any{"type": "integer", "description": "maximum number of directories to list (default 16)"},
				},
				"required": []string{"scope"},
			},
		},
		{
			Name:        "graph_blast",
			Description: "Show what a change reaches: the symbols a git diff touches, their callers, and whether a test that reaches them changed too.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope": areaScope,
					"base":  map[string]any{"type": "string", "description": "compare base...HEAD; empty compares the working tree with HEAD, or the last commit when the tree is clean"},
					"depth": map[string]any{"description": "depth limit as integer or 'all' for full transitive closure"},
				},
				"required": []string{"scope"},
			},
		},
		{
			Name:        "graph_check_freshness",
			Description: "Report whether one area's code graph is in sync with the code (drift check).",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"scope": areaScope},
				"required":   []string{"scope"},
			},
		},
	}
}
