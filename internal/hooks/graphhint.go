package hooks

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/shellwords"
)

// graphHint is the refusal that points a session's first grep over code at
// the code graph, once per session and agent. Everything on its way lets the
// call through instead: no payload, the switch off, no symbol, no graph
// where the shell stands, no session, or a mark it cannot write -- a hint
// without its mark would come back on every repeat.
func graphHint(object map[string]any, root string, policy config.Policy) (string, bool) {
	if object == nil || policy.GraphHintOff {
		return "", false
	}
	tool, input := guard.Call(object)
	session, _ := object["session_id"].(string)
	agent, _ := object["agent_id"].(string)
	if graphQuery(tool, input) {
		// An agent that already asks the graph needs no hint: the query spends
		// it. A mark that cannot be written changes nothing here.
		_, _ = sessions.MarkOnce(root, session, agent, "graph-hint")
		return "", false
	}
	symbol, ok := graphHintSymbol(tool, input)
	if !ok {
		return "", false
	}
	cwd, _ := object["cwd"].(string)
	if cwd == "" {
		cwd = root
	}
	if !graphAt(cwd) {
		return "", false
	}
	if first, err := sessions.MarkOnce(root, session, agent, "graph-hint"); !first || err != nil {
		return "", false
	}
	return graphHintReason(symbol), true
}

// graphQuery reports whether a call asks the code graph: a graph tool of
// loomux's MCP server, or `loomux graph …` in a shell the guard reads.
func graphQuery(tool string, input map[string]any) bool {
	if strings.HasPrefix(tool, "mcp__loomux__graph_") {
		return true
	}
	found := false
	// PowerShell's call operator & needs no care: the segments cut there.
	eachCommand(tool, input, func(words []string, _ bool) bool {
		found = len(words) > 1 && programName(words[0]) == "loomux" && words[1] == "graph"
		return found
	})
	return found
}

// eachCommand hands every command of a shell call to fn as its words, with
// whether it reads a pipe, until fn answers true. A command it cannot split
// is skipped.
func eachCommand(tool string, input map[string]any, fn func(words []string, piped bool) bool) {
	shell, found := commandTools[tool]
	if !found {
		return
	}
	values, _ := commandLines(shell, input)
	for _, value := range values {
		at := 0
		for _, segment := range splitSegments(value, true) {
			// A segment right behind a single | reads the pipe, not files.
			piped := at > 0 && value[at-1] == '|' && (at < 2 || value[at-2] != '|')
			at += len(segment) + 1
			if tool == "PowerShell" {
				// A backslash is a path separator to PowerShell, not an escape:
				// doubled, it survives the POSIX split as written.
				segment = strings.ReplaceAll(segment, `\`, `\\`)
			}
			words, err := shellwords.Split(segment)
			if err != nil {
				continue
			}
			if fn(words, piped) {
				return
			}
		}
	}
}

// programName is a command's program without its folder or .exe.
func programName(word string) string {
	return strings.TrimSuffix(strings.ToLower(path.Base(strings.ReplaceAll(word, `\`, "/"))), ".exe")
}

// hintOnly reports whether a tool reaches the pre-tool hook only for the
// graph hint: the Grep tool and loomux's graph tools.
func hintOnly(tool string) bool {
	return tool == "Grep" || strings.HasPrefix(tool, "mcp__loomux__graph_")
}

// graphAt reports whether a built graph covers dir: one at dir or above.
func graphAt(dir string) bool {
	for {
		if _, err := os.Stat(store.WiringPath(dir)); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// graphHintReason is the hint's own wording: it must not read as a policy
// refusal, which an agent takes as "never do this".
func graphHintReason(symbol string) string {
	return "loomux: this repository has a code graph that answers who calls what in one step.\n" +
		"Instead of grepping for `" + symbol + "`, run `loomux graph callers " + symbol + "` (callers; add `--direction out` for callees) or `loomux graph blast` (what a change reaches).\n" +
		"This hint comes once per session: running the same command again goes through."
}

// graphHintSymbol answers whether a call searches code for a symbol the code
// graph knows, and which: the Grep tool, or grep, rg, git grep or
// Select-String in a shell the guard reads. The rules are the agent spike's
// (docs/wiki/syntheses/graph-hinweis-im-waechter.md): a name in a definition or
// call shape, or a bare capitalised or mixedCase name, searched over code and
// not over docs, logs or configuration.
func graphHintSymbol(tool string, input map[string]any) (string, bool) {
	if tool == "Grep" {
		pattern, _ := input["pattern"].(string)
		var scope searchScope
		if s, _ := input["path"].(string); s != "" {
			scope.paths = append(scope.paths, s)
		}
		if s, _ := input["glob"].(string); s != "" {
			scope.filters = append(scope.filters, s)
		}
		if s, _ := input["type"].(string); s != "" {
			scope.filters = append(scope.filters, "*."+s)
		}
		// The Grep tool searches the working tree when it names no scope.
		if !scope.code(true) {
			return "", false
		}
		return hintSymbol(pattern)
	}
	var symbol string
	eachCommand(tool, input, func(words []string, piped bool) bool {
		pattern, scope, ok := searchOf(words)
		// A shell search with no scope reads its standard input.
		if piped || !ok || !scope.code(false) {
			return false
		}
		symbol, ok = hintSymbol(pattern)
		return ok
	})
	return symbol, symbol != ""
}

// searchScope is where a search looks: the paths it names, and the globs and
// types that filter them.
type searchScope struct {
	paths, filters []string
}

// code reports whether a search runs over code. A filter decides when there
// is one, since `rg -t md .` searches no code whatever the path; otherwise any
// path that is not plainly docs, logs or configuration counts. With neither,
// the search reads what unscoped means for its tool.
func (s searchScope) code(unscoped bool) bool {
	scope := s.filters
	if len(scope) == 0 {
		scope = s.paths
	}
	if len(scope) == 0 {
		return unscoped
	}
	for _, one := range scope {
		for _, alt := range braces(one) {
			if !docScope(alt) {
				return true
			}
		}
	}
	return false
}

// braces expands one level of a glob's braces: `*.{md,txt}` is `*.md` and
// `*.txt`.
func braces(glob string) []string {
	open, end := strings.IndexByte(glob, '{'), strings.IndexByte(glob, '}')
	if open < 0 || end < open {
		return []string{glob}
	}
	var out []string
	for _, alt := range strings.Split(glob[open+1:end], ",") {
		out = append(out, glob[:open]+alt+glob[end+1:])
	}
	return out
}

// searchOf reads a grep, rg, git grep or Select-String command: its pattern
// and the paths, globs and types it searches.
func searchOf(words []string) (pattern string, scope searchScope, ok bool) {
	if len(words) == 0 {
		return "", scope, false
	}
	program := programName(words[0])
	args := words[1:]
	switch {
	case program == "select-string":
		return selectString(args)
	case program == "git" && len(args) > 0 && args[0] == "grep":
		args = args[1:]
	case program != "grep" && program != "rg" && program != "egrep":
		return "", scope, false
	}
	havePattern := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			continue
		case a == "-e" || a == "--regexp":
			if i+1 < len(args) {
				pattern, havePattern = args[i+1], true
				i++
			}
		case strings.HasPrefix(a, "--include=") || strings.HasPrefix(a, "--glob="):
			scope.filters = append(scope.filters, a[strings.IndexByte(a, '=')+1:])
		case strings.HasPrefix(a, "--type="):
			scope.filters = append(scope.filters, "*."+a[len("--type="):])
		case a == "-t" || a == "--type" || a == "-g" || a == "--glob" || a == "--include":
			if i+1 < len(args) {
				value := args[i+1]
				if a == "-t" || a == "--type" {
					value = "*." + value
				}
				scope.filters = append(scope.filters, value)
				i++
			}
		case slices.Contains(valueFlags, a):
			i++
		case strings.HasPrefix(a, "-"):
		case !havePattern:
			pattern, havePattern = a, true
		default:
			scope.paths = append(scope.paths, a)
		}
	}
	return pattern, scope, havePattern
}

// valueFlags are the grep and rg flags whose value is the next word.
var valueFlags = []string{"-A", "-B", "-C", "-m", "--max-count", "-f", "--file", "-j", "--threads"}

// selectString reads PowerShell's Select-String: -Pattern and -Path or
// -LiteralPath, by name or, for the pattern, as the first bare argument.
func selectString(args []string) (pattern string, scope searchScope, ok bool) {
	for i := 0; i < len(args); i++ {
		name := strings.ToLower(args[i])
		switch {
		case (name == "-pattern" || name == "-path" || name == "-literalpath") && i+1 < len(args):
			if name == "-pattern" {
				pattern, ok = args[i+1], true
			} else {
				scope.paths = append(scope.paths, args[i+1])
			}
			i++
		case !strings.HasPrefix(name, "-") && !ok:
			pattern, ok = args[i], true
		}
	}
	return pattern, scope, ok
}

// docScope reports whether one scope names docs, logs or configuration: a
// docs or log folder anywhere on its path, or such a file's extension.
func docScope(s string) bool {
	s = strings.ToLower(strings.ReplaceAll(s, `\`, "/"))
	for _, segment := range strings.Split(s, "/") {
		switch segment {
		case "docs", "doc", "log", "logs":
			return true
		}
	}
	switch path.Ext(strings.TrimRight(s, "*/")) {
	case ".md", ".log", ".json", ".toml", ".yaml", ".yml", ".txt":
		return true
	}
	return false
}

var (
	definedName = regexp.MustCompile(`(?:func\s+(?:\\?\([^)]*\\?\)\s*)?|type\s+)([A-Za-z_]\w*)`)
	calledName  = regexp.MustCompile(`([A-Za-z_]\w*)\\*\(`)
	bareName    = regexp.MustCompile(`^[A-Za-z_]\w*$`)
	shapedName  = regexp.MustCompile(`^[A-Z]|[a-z][A-Z]`)
)

// notSymbols are words a search names that no symbol answers.
var notSymbols = []string{"func", "type", "return", "nil", "true", "false", "err", "string", "error", "struct", "interface", "main", "test"}

// hintSymbol is the symbol a search pattern hunts: a definition first, then a
// call, then a bare capitalised or mixedCase name of an alternation.
func hintSymbol(pattern string) (string, bool) {
	var found []string
	for _, m := range definedName.FindAllStringSubmatch(pattern, -1) {
		found = append(found, m[1])
	}
	for _, m := range calledName.FindAllStringSubmatch(pattern, -1) {
		found = append(found, m[1])
	}
	for _, part := range strings.Split(pattern, "|") {
		part = strings.Trim(strings.ReplaceAll(part, `\b`, ""), `^$\'" `)
		if bareName.MatchString(part) && shapedName.MatchString(part) && strings.ToUpper(part) != part {
			found = append(found, part)
		}
	}
	for _, name := range found {
		if len(name) >= 4 && !slices.Contains(notSymbols, name) {
			// `freshness.Probe(` names its package; keeping it picks the one
			// Probe the search meant. A one-letter receiver names no package.
			qualified := regexp.MustCompile(`([a-z][a-z0-9_]+)\\?\.` + regexp.QuoteMeta(name) + `\b`)
			if m := qualified.FindStringSubmatch(pattern); m != nil {
				return m[1] + "." + name, true
			}
			return name, true
		}
	}
	return "", false
}
