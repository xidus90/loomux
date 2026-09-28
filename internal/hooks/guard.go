// Package hooks holds what loomux runs from a harness lifecycle event: the
// pre-tool-use guard that answers with the project's policy and the global
// write barrier, the post-tool-use lanes, the session start, the status report
// and the worktree mirror.
package hooks

import (
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/shellwords"
)

const (
	ExitOK       = 0
	ExitInternal = 1
	ExitDenied   = 2
)

type HookPayload struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

// manifestReason refuses an agent the manifest, to a writing tool and to a
// shell line alike.
const manifestReason = ".loomux/config.toml: the manifest is where the barrier reads its own limits, so no agent may write it"

// Built-in rules that protect secrets, the manifest, the stop gate's controls,
// the run files and lock files.
var builtinPathRules = []config.PathRule{
	{Match: []string{".env"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".env.*"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"*.pem"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"*.key"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"id_rsa*"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"*.p12"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".npmrc"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".pypirc"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"credentials.json"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".aws/**"}, Reason: "secrets are not written by an agent"},
	// The constant the gate itself stats (stop.go), not a second copy of the
	// path: a marker the guard spelled differently would be an open door.
	{Match: []string{NoVerifyMarker}, Reason: "the stop gate's own controls are not written by the party it gates"},
	// loomux's own files under any directory: a write into a sibling
	// worktree's .loomux, or one named by its absolute path, is the same
	// write as into this project's.
	{Match: []string{"**/.loomux/config.toml"}, Reason: manifestReason},
	// The literal below is the second copy of sessions.StateDir; a rule is a
	// verbatim glob here, so the two are kept in step by hand.
	{Match: []string{"**/.loomux/state/hooks/**"}, Reason: "the stop gate's own controls are not written by the party it gates"},
	// The second copy of runs.Dir, kept in step by hand like the one above.
	{Match: []string{"**/.loomux/state/runs/**"}, Reason: runFilesReason},
	{Match: []string{"uv.lock"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"poetry.lock"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"package-lock.json"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"pnpm-lock.yaml"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"yarn.lock"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"Cargo.lock"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"go.sum"}, Reason: "lock files are written by their package manager, not by hand"},
}

// builtinCommands compiles on first use rather than at load: this binary hangs
// on every tool call, and a package variable would pay for the expression in
// runs that never look at a command line.
var builtinCommands = sync.OnceValue(func() []config.CommandRule {
	const (
		// The manifest as one shell word: quoted or not, under any directory,
		// or glued to a parameter (`of=`, `-FilePath:`); either slash, any case.
		manifestName = `['"]?(?:[^\s;|&'"<>]*[/\\=:])?\.loomux[/\\]+config\.toml['"]?`
		// The runs folder or any file in it, as the same kind of word.
		runsName = `['"]?(?:[^\s;|&'"<>]*[/\\=:])?\.loomux[/\\]+state[/\\]+runs(?:[/\\][^\s;|&'"<>]*)?['"]?`
		// What a removal may name to take the runs with it: the state
		// folder above them, or a glob in their place.
		runsGone = `['"]?(?:[^\s;|&'"<>]*[/\\=:])?\.loomux[/\\]+state(?:[/\\]+(?:runs|` + globName + `)(?:[/\\][^\s;|&'"<>]*)?)?[/\\]*['"]?`
	)
	manifest, runFiles := writeSource(manifestName, manifestName), writeSource(runsName, runsGone)
	return []config.CommandRule{{
		Regex:  regexp.MustCompile(`(^|\s)git\s+push(\s|$)`),
		Source: `(^|\s)git\s+push(\s|$)`,
		Reason: "Whether commits reach the remote is a human's decision.",
	}, {
		Regex:  regexp.MustCompile(manifest),
		Source: manifest,
		Reason: ".loomux/config.toml: the manifest is where the barrier reads its own limits, so no shell command may write it",
	}, {
		Regex:  regexp.MustCompile(runFiles),
		Source: runFiles,
		Reason: runFilesReason,
	}}
})

// globName is one folder name that holds a glob character, and so may stand
// for any name in its place.
const globName = `[^\s;|&'"<>/\\]*[*?\[][^\s;|&'"<>/\\]*`

// writeSource is the expression for a shell line that writes or removes the
// file or folder the expression name spells as one shell word: the manifest,
// which the barrier refuses to every writing tool, and the files a path rule
// keeps from an agent. A shell line is the same write by another road.
//
// removed is the wider word the removing verbs are read against -- rm, del,
// erase, Remove-Item and its aliases, rmdir, rd, git rm, git clean and
// [IO.Directory]::Delete: it adds what takes name with it, the folder above
// or a glob in its place. Every other verb reads name alone, so that an agent
// may still put a flow of its own under .loomux/flows.
//
// It reads command text, not a file system, so it is a net with known holes
// rather than a proof: a path held in a variable (`> "$M"`), a program that
// opens the file itself (`python -c …`) and a command spelled through an alias
// the list does not know all pass. So do, by that choice, a copy or a move
// into the folder above that overwrites a kept folder (`cp -r x/example
// .loomux/flows/`, `mv x/example .loomux/flows/`, `cp -r x/runs
// .loomux/state/`) and a mv of the folder above itself; a removal of .loomux
// (`rm -rf .loomux`), which takes the manifest as well, or of a glob one
// level up (`rm -r .loomux/*`, `rm -r .loomux/flow*`, `rm -r .loomux/state*`);
// a git clean without a path (`git clean -fdX`), which takes ignored run
// files; and a glob in a fixed segment of the path, which the shell expands
// onto the kept file (`.loomux/sta*/runs/…`, `.loomux/state/r*/…` in a
// write, `.loomux/config.tom?`) -- a glob is read only in a flow's own
// segment under .loomux/flows and in a removal in place of the runs folder.
// .loomux/state/hooks has no shell rule of its own; a removal of
// .loomux/state is refused only for the runs in it. It errs the other way
// where it cannot tell: a `>` inside a quoted string, the file named as the
// source of a `mv` or an `Out-File -InputObject`, or a glob that would miss
// it, is refused. One rule rather than one per form, because `checkTool`
// names a reason once per matching rule and a line like `tee M > M` would
// otherwise say the same thing twice.
//
// Every form stays inside one segment of the line -- nothing between the
// command and the file crosses `;`, `|`, `&` or a line break -- so a write
// elsewhere on the line and a read of the file do not add up to a refusal.
func writeSource(name, removed string) string {
	word := name + `(?:[\s;|&),]|$)`
	last := name + `\s*(?:[;|&\n)]|$)`
	gone := removed + `(?:[\s;|&),]|$)`
	const (
		// A command at the head of a segment, behind `sudo` and friends and
		// under any directory; `$(` opens a segment as well.
		head = `(?:^|[;|&({\n])\s*(?:(?:sudo|command|exec|nohup)\s+)*(?:[^\s;|&]*[/\\])?`
		exe  = `(?:\.exe)?\s`
		// The rest of the segment up to the word that names the file.
		rest = `(?:[^;|&\n]*[\s(,])?`
		// `sed -i`, `-i.bak`, `-Ei`, `--in-place`; case-sensitive, because
		// `perl -I` is an include path.
		inPlace = `(?-i:-[A-Za-z]*i\S*|--in-place\S*)`
	)
	forms := []string{
		// A redirect into it: `>`, `>>`, `>|`, `2>`, `&>`.
		`>[>|!]?\s*` + word,
		// Commands that write every file they name.
		head + `(?:tee|tee-object|set-content|add-content|out-file|clear-content|ac|` +
			`mv|move|move-item|mi|rename-item|ren|rni|truncate)` + exe + rest + word,
		// Commands that remove every file or folder they name, a folder with
		// all it holds.
		head + `(?:rm|del|erase|remove-item|ri|rmdir|rd)` + exe + rest + gone,
		// Commands that write only their destination.
		head + `(?:cp|copy|copy-item|cpi|install)` + exe + rest + last,
		head + `(?:cp|copy|copy-item|cpi|install)` + exe + `(?:[^;|&\n]*\s)?-dest\w*[:\s]\s*` + word,
		head + `dd` + exe + `(?:[^;|&\n]*\s)?of=` + word,
		head + `git\s+(?:-\S+\s+)*(?:mv|checkout|restore)\s` + rest + word,
		head + `git\s+(?:-\S+\s+)*(?:rm|clean)\s` + rest + gone,
		// An in-place edit, with the flag before or after the file.
		head + `(?:sed|perl)` + exe + `(?:[^;|&\n]*\s)?` + inPlace + `\s` + rest + word,
		head + `(?:sed|perl)` + exe + rest + word + `(?:[^;|&\n]*\s)?` + inPlace,
		// .NET from PowerShell.
		`\[(?:system\.)?io\.(?:file|directory)\]::(?:write|append|create|delete|move|replace)\w*\s*\(` + rest + word,
		`\[(?:system\.)?io\.directory\]::delete\w*\s*\(` + rest + gone,
	}
	return `(?i)` + strings.Join(forms, "|")
}

// commandTool says how the guard finds the shell lines in a call to a tool
// that runs them. Every zero value is the closed case: a tool added with
// nothing set refuses a call without a line and has what it carries judged
// as typing, whole lines only.
//
// Antigravity's run_command sends CommandLine (measured with agy 1.2.11 on
// 2026-09-25); command_line is the other spelling agy.exe carries. agy
// 1.2.11 types a line into a task run_command left open with manage_task,
// Action send_input and the line under Input (measured on 2026-09-25); the
// model wrote the line end itself, so Input reaches the terminal as it
// stands. send_command_input is the older tool for the same, which agy.exe
// still carries; its argument name is not measured, Input is the guess.
// manage_task's schema in agy 1.2.11 names list, status, kill and
// send_input; the first three carry no line.
type commandTool struct {
	keys         []string // argument names the line may stand under, matched without case
	lineOptional bool     // a call without a line passes: Claude's shells, whose tool always carries one
	whole        bool     // the value is a whole command line, not keystrokes typed into an open terminal
	quiet        []string // Actions whose calls carry no line
}

// commandTools are the tools that run a shell line; every line found is
// judged, for WriteTargets' reason.
var commandTools = map[string]commandTool{
	"Bash":               {keys: []string{"command"}, lineOptional: true, whole: true},
	"PowerShell":         {keys: []string{"command"}, lineOptional: true, whole: true},
	"run_command":        {keys: []string{"CommandLine", "command_line"}, whole: true},
	"send_command_input": {keys: []string{"Input"}},
	"manage_task":        {keys: []string{"Input"}, quiet: []string{"list", "status", "kill"}},
}

// commandLines is every non-empty string a call to tool carries under one of
// its keys, in any case, in the sorted order of the call's keys so that the
// reasons come out the same on every run. The values are found before the
// Action is read: a quiet Action does not stop a line it carries from being
// judged. A call is answered with ok false when a key holds a value that is
// no string, which might be the line the tool runs, and when it carries none
// and is not quiet: a line the guard cannot find would switch off every
// command rule without a word. The strings found are judged either way.
func commandLines(tool commandTool, input map[string]any) (values []string, ok bool) {
	readable := true
	for _, key := range slices.Sorted(maps.Keys(input)) {
		if !slices.ContainsFunc(tool.keys, func(name string) bool { return strings.EqualFold(name, key) }) {
			continue
		}
		switch value := input[key].(type) {
		case string:
			// An empty value types and runs nothing.
			if value != "" {
				values = append(values, value)
			}
		default:
			readable = false
		}
	}
	if !readable {
		return values, false
	}
	if len(values) > 0 {
		return values, true
	}
	return nil, quietAction(tool, input)
}

// quietAction says whether a call names an Action that carries no line.
// Every key spelled action counts, and each must name a quiet Action as a
// string: a second spelling with another Action, or a value that is no
// string, might be the one the tool obeys.
func quietAction(tool commandTool, input map[string]any) bool {
	named := false
	for key, value := range input {
		if !strings.EqualFold(key, "action") {
			continue
		}
		// A value that is no string reads as "", which no quiet list holds.
		if action, _ := value.(string); !slices.Contains(tool.quiet, action) {
			return false
		}
		named = true
	}
	return named
}

// typedLines is the lines a terminal would run from what an agent types into
// an open task, or the reason the guard cannot tell. Only whole lines without
// control characters can be judged: a fragment may be finished by the next
// call, and a backspace, an escape sequence, a tab an interactive shell
// completes at, or another key the terminal binds edits the line after the
// guard has read it. A line ending in a backslash or a backtick is not whole
// either: bash and PowerShell continue it on the next line.
func typedLines(name, value string) (lines []string, refusal string) {
	const rule = "loomux judges what an agent types into a task only as whole lines without control characters; this "
	if !strings.HasSuffix(value, "\n") && !strings.HasSuffix(value, "\r") {
		return nil, rule + name + " input does not end its line, so it refuses"
	}
	if strings.ContainsFunc(value, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\r' }) {
		return nil, rule + name + " input carries a control character, so it refuses"
	}
	lines = strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == '\r' })
	// A backslash or a backtick at a line end continues the line in bash or
	// PowerShell: the next line finishes it, after the guard has read it.
	for _, line := range lines {
		if strings.HasSuffix(line, `\`) || strings.HasSuffix(line, "`") {
			return nil, rule + name + " input does not end its line, so it refuses"
		}
	}
	return lines, ""
}

// matchGlob matches a slash-separated path against a glob pattern supporting
// `**`, and answers an error for a pattern it cannot read. A leading `**/`
// stands for any directory, the root included.
//
// It is `path.Match`, not `filepath.Match`: the path is slash-separated on
// every platform, and on Windows `filepath.Match` separates on `\` only, so a
// `*` there ran across a `/`.
//
// The error is not dropped, and that is the point. `config.ReadPolicy` refuses
// a malformed glob at load, but its check -- `path.Match(glob, "")` --
// stops at the first chunk that does not match an empty name, so a bad class in
// a later chunk (`foo/*[x`) still arrives here. Treating that as "no match"
// made the rule protect nothing without a word; the caller turns it into a
// refusal instead.
func matchGlob(pattern, name string) (bool, error) {
	// A leading **/ is any directory, the root included: the rest is tried
	// against the name and against every tail of it that starts after a
	// slash, so an element is matched whole.
	if rest, anywhere := strings.CutPrefix(pattern, "**/"); anywhere {
		for {
			if matched, err := matchGlob(rest, name); err != nil || matched {
				return matched, err
			}
			slash := strings.IndexByte(name, '/')
			if slash < 0 {
				return false, nil
			}
			name = name[slash+1:]
		}
	}
	if pattern == name {
		return true, nil
	}
	// Direct wildcard suffix like .aws/**
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if name == prefix || strings.HasPrefix(name, prefix+"/") {
			return true, nil
		}
	}
	if strings.Contains(pattern, "/") {
		return path.Match(pattern, name)
	}
	// For patterns without slashes (e.g. *.pem or .env.* or uv.lock)
	// they match either at the root or base name depending on rule semantics
	return path.Match(pattern, path.Base(name))
}

// relativePath names a target the way a rule spells one: relative to the
// project root, with forward slashes.
//
// A target that will not relativise -- another volume, or a path outside the
// root -- is matched as the absolute path it is, and that is a deliberate
// half-answer rather than a fallback that works. Every rule carrying a slash
// (`.aws/**`, `.loomux/no-verify`) stops matching such a target, because the
// absolute path does not begin where the rule does; only the rules without a
// slash, which are matched against the base name, still reach it -- and the
// rules under `**/`, which match loomux's own files under any directory. Those
// targets are the write barrier's to decide, and it does: it resolves the path
// and compares it with the registered trees, which is the question "is this
// file even in this project" asked properly. The policy is about paths in the
// project, so it answers about those and leaves the rest where the answer is.
func relativePath(raw, root string) string {
	raw = filepath.Clean(raw)
	if !filepath.IsAbs(raw) {
		return filepath.ToSlash(raw)
	}
	rel, err := filepath.Rel(root, raw)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(raw)
	}
	return filepath.ToSlash(rel)
}

// checkTool judges one tool call against the built-in rules and the project's
// own, and answers every reason it found: a caller that wants to say why it
// refuses needs all of them, not the first.
func checkTool(root, tool string, input map[string]any, policy config.Policy) []string {
	var reasons []string
	if guard.IsWritingTool(tool) {
		for _, target := range guard.WriteTargets(input) {
			rel := relativePath(target, root)
			// loomux's own rules fold case, as the barrier does for the
			// manifest: Windows and macOS keep .LOOMUX/State/hooks and
			// .loomux/state/hooks as one folder. A project's rules match as the
			// project spelled them.
			reasons = append(reasons, pathReasons(builtinPathRules, rel, true)...)
			reasons = append(reasons, pathReasons(policy.Paths, rel, false)...)
			reasons = append(reasons, flowFolderReasons(root, rel)...)
		}
	}
	if shell, found := commandTools[tool]; found {
		values, ok := commandLines(shell, input)
		if !ok && !shell.lineOptional {
			reasons = append(reasons, "loomux found no command line in this "+tool+" call, so it cannot judge it and refuses")
		}
		for _, value := range values {
			lines := []string{value}
			if !shell.whole {
				var refusal string
				if lines, refusal = typedLines(tool, value); refusal != "" {
					reasons = append(reasons, refusal)
					continue
				}
			}
			for _, line := range lines {
				for _, rule := range append(builtinCommands(), policy.Commands...) {
					if rule.Regex.MatchString(line) {
						reasons = append(reasons, rule.Reason)
					}
				}
				// The flow folder rule reads the config, so only a line that
				// could name such a folder pays for it.
				if strings.Contains(strings.ToLower(line), "flows") {
					if rule, ok := flowFolderCommand(root); ok && rule.Regex.MatchString(line) {
						reasons = append(reasons, rule.Reason)
					}
				}
				if writesConfiguration(line) {
					reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads, merge-hook install and remove write executable hooks into repositories, and convert and fetch write into an area's inbox, which the write barrier keeps from agents; a human runs them. An agent proposes a change with `loomux config set|unset … --propose`, which a human applies")
				}
				if answersAGate(line) {
					reasons = append(reasons, "a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer \"…\"` themselves")
				}
			}
		}
	}
	return reasons
}

// pathReasons is the reason of every rule that matches rel, with the rule and
// rel both in lower case when fold is set.
func pathReasons(rules []config.PathRule, rel string, fold bool) []string {
	if fold {
		rel = strings.ToLower(rel)
	}
	var reasons []string
	for _, rule := range rules {
		for _, glob := range rule.Match {
			if fold {
				glob = strings.ToLower(glob)
			}
			matched, err := matchGlob(glob, rel)
			if err != nil {
				// A rule nobody can evaluate is a rule nobody can trust,
				// and the call it would have judged goes no further.
				reasons = append(reasons, fmt.Sprintf(
					"loomux cannot read the glob %q of the rule %q, so it refuses: %v",
					glob, rule.Reason, err))
				break
			}
			if matched {
				reasons = append(reasons, rule.Reason)
				break
			}
		}
	}
	return reasons
}

// writesConfiguration says whether a shell line runs a loomux command that
// writes .loomux/config.toml or the global config in-process, or an area's
// inbox, past every path rule. It is a function rather than a CommandRule
// because "init without --dry-run" needs a lookahead that RE2 lacks. It
// reads words, not a file system, so it is a net with holes; readings states
// what it guarantees and what passes.
func writesConfiguration(line string) bool {
	// Judged on the line as written, before any rewrite: a continuation or
	// an escape the rewrites resolve is already no plain line.
	plain := plainLine(line)
	for _, variant := range lineVariants(line) {
		// The quote-blind cut also breaks inside a wrapper's quoted inner
		// command and puts its loomux at the head of a segment, while the
		// wrapper reads that string once more. Such a segment finds a call
		// but exempts nothing; only one whose bounds lie outside quotes may.
		aware := splitSegments(variant, true)
		for _, segment := range segments(variant) {
			exempt := plain && slices.Contains(aware, segment)
			for _, words := range readings(segment) {
				if readingWrites(words, exempt) {
					return true
				}
			}
		}
	}
	return false
}

// lineVariants are the line as written and the line as each shell would join
// it before reading words. The line as written stays among them because a
// continuation of one shell is none to the other: `\` at a line end continues
// in bash but not in PowerShell, which runs the first line on its own. The
// joined line then loses its backticks, the PowerShell escape character, so
// loomux con`fig is read as config; a bash substitution in backticks is
// already cut apart by segments in the line as written.
//
// Each rewrite applies to every variant made before it, so every combination
// is judged -- a backtick escape inside a block with glued braces as well as
// either alone. A rewrite that changes nothing adds no variant, which keeps a
// plain line at one.
func lineVariants(line string) []string {
	out := []string{line}
	rewrites := []*strings.Replacer{
		strings.NewReplacer("\\\r\n", "", "\\\n", "", "`\r\n", " ", "`\n", " "),
		strings.NewReplacer("`", ""),
		// A brace glued to a word ({loomux init}, try{) still opens or
		// closes a block; set apart, it becomes the lone word readingWrites
		// looks behind.
		strings.NewReplacer("{", " { ", "}", " } "),
	}
	for _, r := range rewrites {
		for _, v := range out {
			if w := r.Replace(v); !slices.Contains(out, w) {
				out = append(out, w)
			}
		}
	}
	return out
}

// readings returns the ways a segment may be read, and the rule refuses when
// any of them writes, so a reading can add a refusal but never remove one.
//
// A backslash is a shell escape to bash and a path separator to PowerShell.
// The bash reading is the strict split as written, which resolves escapes
// (con\fig, "say \"hi") and is dropped when it fails. The PowerShell reading
// turns backslashes into slashes and splits tolerantly: it never fails, and
// wherever a strict split of that text would succeed the two agree. The
// field reading splits at blanks and trims the quotes off each word, honouring
// no quote at all: after an earlier escaped quote both other readings group
// the wrong text, but the tail of a quote-blind segment cut at the ( of
// "C:\Program Files (x86)\…" still starts with the program's path.
//
// Guaranteed, together with segments: a program named loomux, loomux.exe or
// a path ending in either, quoted or not, with any bytes in the quoted path,
// is read as the program of some segment, and an unclosed quote or a stray
// escape never makes a segment pass. Not guaranteed: an alias, a program held
// in a variable, a command inside a string (sh -c "loomux init", pwsh -c
// ...), and, after any earlier escaped \" or \' on the line, a quoted program
// path whose part after its last break character ( ) & ; | holds a blank: the
// field reading then starts that segment inside the path, as in
// `echo "a \" b"; "C:\Program Files (x86)\My Tools\loomux.exe" init`.
func readings(segment string) [][]string {
	var out [][]string
	if words, err := shellwords.Split(segment); err == nil {
		out = append(out, words)
	}
	fields := strings.Fields(segment)
	for i, w := range fields {
		fields[i] = strings.Trim(w, `"'`)
	}
	return append(out, tolerantWords(strings.ReplaceAll(segment, `\`, "/")), fields)
}

// tolerantWords splits like a shell without escapes and never fails: ' and "
// group, and a quote left open runs to the end of the segment.
func tolerantWords(s string) []string {
	var words []string
	var word strings.Builder
	inWord := false
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0 && c == quote:
			quote = 0
		case quote != 0:
			word.WriteByte(c)
		case c == '"' || c == '\'':
			quote, inWord = c, true
		case c == ' ' || c == '\t' || c == '\r':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteByte(c)
			inWord = true
		}
	}
	if inWord {
		words = append(words, word.String())
	}
	return words
}

// readingWrites judges one reading of a segment from its head and from every
// word after a lone { or }. Braces are no segment breaks, because ${VAR}
// holds them, yet a block opens a command: the body of try { … } catch { … }
// or of a function sits there, behind a word no break precedes.
func readingWrites(words []string, plain bool) bool {
	if wordsWriteConfiguration(words, plain) {
		return true
	}
	// A call behind a brace is no direct call, so its flag exempts nothing.
	for i, w := range words {
		if (w == "{" || w == "}") && wordsWriteConfiguration(words[i+1:], false) {
			return true
		}
	}
	return false
}

// wordsWriteConfiguration judges one reading of a segment. An exempting flag
// counts only when the whole line is plain and loomux is the segment's first
// word: a wrapper may read the words once more (cmd resolves ^, %X%, !X! and
// " inside what the shell passed on as one quoted word), and the guard does
// not model that second reading.
func wordsWriteConfiguration(words []string, plain bool) bool {
	head := len(words)
	words = dropPrefixes(words)
	if len(words) == 0 {
		return false
	}
	exempt := plain && len(words) == head
	if startsLoomux(words) {
		return true
	}
	found, ok := loomuxArgs(words)
	if !ok || len(found) == 0 {
		return false
	}
	// A block's closing brace glued to the last word (--dry-run}) is no part
	// of it; the reading where braces stand apart judges the same line.
	args := make([]string, len(found))
	for i, a := range found {
		args[i] = strings.TrimRight(a, "})")
	}
	switch args[0] {
	case "init":
		return !exempt || !flagOn(args, "--dry-run", "--detect-only")
	case "config":
		// config takes --root and --global before its subcommand as well;
		// the subcommand is judged where it stands.
		args = append(args[:1:1], skipTargetFlags(args[1:])...)
		// What reads is named, and everything else writes: a subcommand
		// added later is refused until it is listed here.
		if len(args) > 1 {
			switch args[1] {
			case "list", "get", "proposals":
				return false
			case "set", "unset":
				return !exempt || !onlyProposes(args[2:])
			}
		}
		return len(args) != 2 || (args[1] != "--help" && args[1] != "-h")
	case "convert", "fetch":
		// Both write into an area's inbox, which the write barrier keeps
		// from agents; a lone --help or -h only reads.
		return len(args) != 2 || (args[1] != "--help" && args[1] != "-h")
	case "area":
		return len(args) > 1 && args[1] == "add"
	case "merge-hook":
		// install and remove put executable hooks into other repositories;
		// status reads and record is what the installed hook itself calls.
		return len(args) > 1 && (args[1] == "install" || args[1] == "remove")
	}
	return false
}

// startsLoomux says whether words run Start-Process, start or saps with
// loomux anywhere among the arguments. The arguments travel as one
// PowerShell list the words cannot judge, and the program may stand behind
// any parameter, so such a call counts as every call a rule looks for.
func startsLoomux(words []string) bool {
	switch baseName(words[0]) {
	case "start-process", "start", "saps":
		return slices.ContainsFunc(words[1:], func(w string) bool {
			// -FilePath:loomux.exe carries the program behind the colon.
			if strings.HasPrefix(w, "-") {
				w = w[strings.IndexByte(w, ':')+1:]
			}
			return isLoomux(strings.Trim(w, `"'`))
		})
	}
	return false
}

// skipTargetFlags drops the flags that choose config's file -- --root with
// its value, --root=…, --global -- from the front of args. Any other flag
// stops it: the words after it are judged as they stand, and a bare flag
// there refuses.
func skipTargetFlags(args []string) []string {
	for len(args) > 0 {
		name, _, glued := strings.Cut(strings.TrimLeft(args[0], "-"), "=")
		switch {
		case !strings.HasPrefix(args[0], "-"):
			return args
		case name == "global" && !glued:
			args = args[1:]
		case name == "root" && glued:
			args = args[1:]
		case name == "root" && len(args) > 1:
			args = args[2:]
		default:
			return args
		}
	}
	return args
}

// onlyProposes says whether config set or unset stores a proposal instead of
// writing: --propose stands as a flag of its own (see flagOn).
func onlyProposes(args []string) bool {
	return flagOn(args, "--propose", "-propose")
}

// flagOn says whether one of the spellings stands among the arguments as a
// word of its own, with nothing that could take it back. It trusts the words
// only on a plainLine, where each word is what the program receives.
//
// A comment ends the words. Past a -- a word is a value the parse hands on,
// so a flag there counts for nothing. A redirection does not end the
// arguments -- `> out --x` still passes --x -- so from the first one on only
// redirections and their targets may follow. A -name=… of the same flag
// refuses, since the last one wins and it may say false.
func flagOn(args []string, spellings ...string) bool {
	for i, a := range args {
		if strings.HasPrefix(a, "#") {
			args = args[:i]
			break
		}
	}
	for i, a := range args {
		if redirects(a) {
			if !onlyRedirections(args[i:]) {
				return false
			}
			args = args[:i]
			break
		}
	}
	on := false
	for _, a := range args {
		if a == "--" {
			break
		}
		name := strings.TrimLeft(a, "-")
		for _, s := range spellings {
			if name != a && strings.HasPrefix(name, strings.TrimLeft(s, "-")+"=") {
				return false
			}
			on = on || a == s
		}
	}
	return on
}

// onlyRedirections says whether words are redirections alone, each with its
// target: glued (>out, 2>&1) or as the next word (> out).
func onlyRedirections(words []string) bool {
	for i := 0; i < len(words); i++ {
		if !redirects(words[i]) {
			return false
		}
		target := strings.TrimLeft(strings.TrimLeft(words[i], "0123456789&"), "<>")
		if target == "" {
			i++
		}
	}
	return true
}

// plainLine says whether a line holds only words a shell passes on as
// written, so the words the guard reads are the words the program gets. It
// is an allowlist, because every list of what expands (brace expansion,
// globs, $, backticks, PowerShell sub-expressions and splats, cmd's %X%)
// has kept growing.
//
// Outside quotes a line may hold plainByte, the breaks ; | & and line
// breaks, the redirections < and >, and a # that opens a comment at the
// start of a word. Inside double quotes only plainByte and # may stand, which
// leaves out $, the backtick and \; a # there is a byte of the word to both
// shells, and a word that begins with it only ends the flags flagOn reads,
// which can refuse more but never less. Inside single quotes any ASCII byte may
// stand but the breaks ; | & ( ) < >, cmd's ^ % ! and #; a byte beyond ASCII
// refuses, because PowerShell also ends a single quoted string at a
// typographic quote (’). A quote left open refuses.
func plainLine(line string) bool {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c >= 0x80:
			return false
		case quote == '\'':
			if c == '\'' {
				quote = 0
			} else if strings.IndexByte(";|&()<>^%!#", c) >= 0 {
				// Neither bash nor PowerShell reads these here, but a
				// wrapper that gets the string may: as a break for an inner
				// command, or as cmd's escape and expansions. A quoted #
				// reaches the program as a word, which flagOn would take
				// for the start of a comment.
				return false
			}
		case quote == '"':
			if c == '"' {
				quote = 0
			} else if !plainByte(c) && c != '#' {
				return false
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '#' && (i == 0 || strings.IndexByte(" \t\r\n;|&", line[i-1]) >= 0):
			end := strings.IndexByte(line[i:], '\n')
			if end < 0 {
				return true
			}
			i += end
		case c == '&' && !pairedAmpersand(line, i):
			return false
		case !plainByte(c) && strings.IndexByte(";|&\r\n<>", c) < 0:
			return false
		}
	}
	return quote == 0
}

// pairedAmpersand says whether the & at i is part of && or of a redirection
// (2>&1, &>). A lone & is PowerShell's call operator or bash's background
// job; either puts a loomux call behind something that is not a plain break.
func pairedAmpersand(line string, i int) bool {
	return i > 0 && strings.IndexByte("&<>", line[i-1]) >= 0 ||
		i+1 < len(line) && strings.IndexByte("&<>", line[i+1]) >= 0
}

// plainByte is a byte no shell reads as anything but itself inside a word:
// letters, digits, blanks and . _ / : = , + -. The % of cmd's %X% is left
// out, and so is everything that globs, expands or escapes.
func plainByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		strings.IndexByte(" \t._/:=,+-", c) >= 0
}

// redirects says whether a word opens a redirection: >, <, their doubled
// forms and a descriptor or & before them (2>, &>, 2>&1, <<<).
func redirects(word string) bool {
	rest := strings.TrimLeft(word, "0123456789&")
	return strings.HasPrefix(rest, "<") || strings.HasPrefix(rest, ">")
}

// segments cuts a line where a new command may start, and answers the pieces
// of two cuts together, because each one alone lets something through.
//
// Both break at ; | & (so && and || too), line breaks, ( ) for subshells
// and $( ), and the backtick of a bash substitution; the PowerShell call
// operator & is a break as well, which leaves the program it calls at the head
// of the next segment. The & of >& and <& is none. Braces are not breaks,
// because ${VAR} holds one; a { that opens a group is a word of its own and
// dropPrefixes skips it.
//
//   - The quote-blind cut breaks inside strings too. It keeps a command that a
//     misread quote would hide: a bash-escaped \" pairs wrongly under a cut
//     that honours no escapes, and a real ; after it would vanish. It also
//     refuses `echo "x; loomux init"`, a false positive kept on purpose.
//   - The quote-aware cut breaks only outside ' and " and gives every quoted
//     program path an uncut segment: "C:\R&D Tools\loomux.exe" and
//     "C:\Program Files (x86)\…" stay whole. It honours no escapes, so a
//     backslash never moves a break and one run on the line as written serves
//     the rewritten reading as well; a quote left open runs to the end.
//
// A line without quotes cuts the same both ways, so only then is the second
// cut skipped.
func segments(line string) []string {
	out := splitSegments(line, false)
	if strings.ContainsAny(line, `"'`) {
		out = append(out, splitSegments(line, true)...)
	}
	return out
}

// splitSegments is one cut of segments, in a single pass without allocation
// beyond the result.
func splitSegments(line string, quoteAware bool) []string {
	var out []string
	start := 0
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case quoteAware && (c == '"' || c == '\''):
			quote = c
		case c == '&' && i > 0 && (line[i-1] == '>' || line[i-1] == '<'),
			c == '|' && i > 0 && line[i-1] == '>':
			// >& and <& duplicate a descriptor, and >| writes over a file
			// noclobber keeps; cutting there would leave the 1 of 2>&1 in
			// front of the program, or the target of >| as a program.
		case strings.IndexByte(";|&\n()`", c) >= 0:
			out = append(out, line[start:i])
			start = i + 1
		}
	}
	return append(out, line[start:])
}

// dropPrefixes strips what runs in front of the real program: the VAR=value
// assignments a shell applies to its environment, redirections with their
// target, the shell's reserved words, the { that opens a group, and the
// wrappers that run the word after them -- with their flags, and the separate
// value of the flags wrapperTakesValue names.
func dropPrefixes(words []string) []string {
	for len(words) > 0 {
		w := words[0]
		n := 1
		isRedirect, bare := redirection(w)
		switch base := baseName(w); {
		case isRedirect:
			if bare {
				n = 2
			}
		case strings.Contains(w, "=") && !strings.HasPrefix(w, "-"):
		case base == "{" || base == "!" || base == "if" || base == "then" || base == "else" ||
			base == "elif" || base == "while" || base == "until" || base == "do" ||
			base == "coproc" || base == "try" || base == "catch" || base == "finally":
		case base == "function":
			// The name, then the body.
			n = 2
		case base == "sudo" || base == "env" || base == "xargs":
			n += wrapperFlags(base, words[1:])
		case base == "command" || base == "exec" || base == "nohup" || base == "time":
			n += flagCount(words[1:])
		case base == "nice":
			if len(words) > 2 && words[1] == "-n" {
				n = 3 + flagCount(words[3:])
			} else {
				n += flagCount(words[1:])
			}
		case base == "timeout":
			// The duration comes before the program.
			n += wrapperFlags(base, words[1:]) + 1
		case base == "cmd" || base == "cmd.exe":
			// Every switch up to /c or /k, which the command follows.
			for n < len(words) && len(words[n]) > 1 && words[n][0] == '/' {
				n++
				if f := strings.ToLower(words[n-1]); f == "/c" || f == "/k" {
					break
				}
			}
		default:
			return words
		}
		words = words[min(n, len(words)):]
	}
	return words
}

// flagCount is how many words at the head of words are flags.
func flagCount(words []string) int {
	n := 0
	for n < len(words) && len(words[n]) > 1 && words[n][0] == '-' {
		n++
	}
	return n
}

// wrapperFlags is how many words at the head of words are the wrapper's
// flags, a flag's separate value included; -- ends them and counts.
func wrapperFlags(wrapper string, words []string) int {
	n := 0
	for n < len(words) && len(words[n]) > 1 && words[n][0] == '-' {
		if words[n] == "--" {
			return n + 1
		}
		if wrapperTakesValue(wrapper, words[n]) {
			n++
		}
		n++
	}
	return min(n, len(words))
}

// wrapperTakesValue names the flags of sudo, env, xargs and timeout whose
// value is the next word.
func wrapperTakesValue(wrapper, flag string) bool {
	switch wrapper {
	case "sudo":
		return slices.Contains([]string{"-u", "-g", "-C", "-D", "-h", "-p", "-r", "-R", "-t", "-U", "-T",
			"--user", "--group", "--close-from", "--chdir", "--chroot", "--host", "--prompt", "--role",
			"--type", "--other-user", "--command-timeout"}, flag)
	case "env":
		// -S is left out: its value is the command line itself.
		return slices.Contains([]string{"-u", "-C", "--unset", "--chdir"}, flag)
	case "xargs":
		return slices.Contains([]string{"-n", "-L", "-P", "-s", "-I", "-d", "-E", "-a",
			"--max-args", "--max-lines", "--max-procs", "--max-chars", "--delimiter", "--arg-file"}, flag)
	}
	return slices.Contains([]string{"-s", "-k", "--signal", "--kill-after"}, flag)
}

// redirection says whether w is a redirection (>out, 2>/dev/null, <, 2>&1)
// and whether its target is the next word rather than glued to it.
func redirection(w string) (isRedirect, bare bool) {
	i := 0
	for i < len(w) && w[i] >= '0' && w[i] <= '9' {
		i++
	}
	if i == len(w) || (w[i] != '<' && w[i] != '>') {
		return false, false
	}
	for i < len(w) && strings.IndexByte("<>|&", w[i]) >= 0 {
		i++
	}
	return true, i == len(w)
}

// baseName is the last path element of w in lower case. It is cut by hand at
// either slash so the answer does not depend on the platform the guard runs
// on.
func baseName(w string) string {
	return strings.ToLower(w[strings.LastIndexAny(w, `/\`)+1:])
}

// isLoomux says whether w names the loomux program: by name or by a path
// ending in loomux or loomux.exe.
func isLoomux(w string) bool {
	base := baseName(w)
	return base == "loomux" || base == "loomux.exe"
}

// loomuxArgs returns the arguments after the program when the program is
// loomux, or go run of its main package.
func loomuxArgs(words []string) ([]string, bool) {
	if isLoomux(words[0]) {
		return words[1:], true
	}
	if baseName(words[0]) != "go" || len(words) < 2 || words[1] != "run" {
		return nil, false
	}
	rest := words[2:]
	for len(rest) > 0 && strings.HasPrefix(rest[0], "-") {
		flag := rest[0]
		rest = rest[1:]
		if !strings.Contains(flag, "=") && goFlagTakesValue(strings.TrimLeft(flag, "-")) && len(rest) > 0 {
			rest = rest[1:]
		}
	}
	if len(rest) > 0 && isLoomuxPackage(rest[0]) {
		return rest[1:], true
	}
	return nil, false
}

// goFlagTakesValue names the build flags of go run whose value is the next
// word; every other flag is a switch.
func goFlagTakesValue(name string) bool {
	switch name {
	case "C", "tags", "ldflags", "gcflags", "asmflags", "gccgoflags", "mod", "modfile",
		"exec", "toolexec", "overlay", "pgo", "p", "pkgdir", "buildmode", "compiler",
		"installsuffix", "coverpkg", "covermode", "o":
		return true
	}
	return false
}

// isLoomuxPackage says whether a go run argument is loomux's main package:
// cmd/loomux or its main.go, relative or under a module path, at any version.
// It compares whole path elements, so mycmd/loomux is another package.
func isLoomuxPackage(p string) bool {
	p = strings.ToLower(p)
	if at := strings.LastIndexByte(p, '@'); at >= 0 {
		p = p[:at]
	}
	p = strings.TrimSuffix(strings.TrimSuffix(p, "/"), "/main.go")
	p = strings.TrimPrefix(p, "./")
	return p == "cmd/loomux" || strings.HasSuffix(p, "/cmd/loomux")
}
