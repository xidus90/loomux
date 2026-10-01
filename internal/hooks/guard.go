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
// runs that never look at a command line. What a shell line writes is judged
// by the path rules (shellWrites), not here.
var builtinCommands = sync.OnceValue(func() []config.CommandRule {
	return []config.CommandRule{{
		Regex:  regexp.MustCompile(`(^|\s)git\s+push(\s|$)`),
		Source: `(^|\s)git\s+push(\s|$)`,
		Reason: "Whether commits reach the remote is a human's decision.",
	}}
})

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
// own, and answers every reason it found, each once: a caller that wants to
// say why it refuses needs all of them, not the first. A writing tool's
// targets and the targets of a shell line go through the same path check; a
// shell line is also read by the command rules and for loomux's own commands.
func checkTool(root, tool string, input map[string]any, policy config.Policy) []string {
	var reasons []string
	var targets []shellTarget
	j := newJudge(root, policy)
	if guard.IsWritingTool(tool) {
		for _, target := range guard.WriteTargets(input) {
			targets = append(targets, shellTarget{path: target})
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
				found, unknown := shellWrites(root, line)
				targets = append(targets, found...)
				if policy.Strict {
					reasons = append(reasons, j.strictReasons(found, unknown)...)
				}
				if writesConfiguration(line, policy.Strict) {
					reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads, merge-hook install and remove write executable hooks into repositories, dev switchover prune-hooks removes hook entries from a settings file, and convert and fetch write into an area's inbox, which the write barrier keeps from agents; a human runs them. An agent proposes a change with `loomux config set|unset … --propose`, which a human applies")
				}
				if answersAGate(line, policy.Strict) {
					reasons = append(reasons, "a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer \"…\"` themselves")
				}
			}
		}
	}
	reasons = append(reasons, j.reasons(targets)...)
	return uniqueReasons(reasons)
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
// inbox, a repository's hooks or a settings file's hook entries, past every
// path rule. It is a function rather than a CommandRule
// because "init without --dry-run" needs a lookahead that RE2 lacks. It
// reads words, not a file system, so it is a net with holes; readings states
// what it guarantees and what passes in the default mode. With anyProgram, in
// strict mode, every program knownProgram does not name counts as loomux.
func writesConfiguration(line string, anyProgram bool) bool {
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
				if readingWrites(words, exempt, anyProgram) {
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
		// A caret escapes the next character for cmd (con^fig); dropping it
		// adds the reading cmd runs, and never removes a refusal.
		strings.NewReplacer("^", ""),
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
func readingWrites(words []string, plain, anyProgram bool) bool {
	if wordsWriteConfiguration(words, plain, anyProgram) {
		return true
	}
	// A call behind a brace is no direct call, so its flag exempts nothing.
	for i, w := range words {
		if (w == "{" || w == "}") && wordsWriteConfiguration(words[i+1:], false, anyProgram) {
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
func wordsWriteConfiguration(words []string, plain, anyProgram bool) bool {
	head := len(words)
	read := readPrefixes(words)
	if anyProgram && slices.ContainsFunc(read.named, func(call []string) bool { return programWrites(call, false, true) }) {
		return true
	}
	if len(read.program) == 0 {
		return false
	}
	return programWrites(read.program, plain && len(read.program) == head, anyProgram)
}

// programWrites is wordsWriteConfiguration for words that start with the
// program, exempt when a reading flag may exempt the call.
func programWrites(words []string, exempt, anyProgram bool) bool {
	if startsLoomux(words) {
		return true
	}
	found, ok := programArgs(words, anyProgram)
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
	case "dev":
		// switchover prune-hooks rewrites the settings file it is pointed at
		// in-process, past every path rule: it would take the guard's own
		// hook entries out of a project the write barrier closes. It has no
		// reading form, so no flag exempts it. The script a human runs calls
		// it; every other dev command passes.
		return len(args) > 2 && args[1] == "switchover" && args[2] == "prune-hooks"
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
// wrappers that run the word after them -- with their flags, read as
// wrapperFlags reads them.
func dropPrefixes(words []string) []string {
	return readPrefixes(words).program
}

// prefixes is what runs in front of a program, as readPrefixes reads it.
type prefixes struct {
	// program is the words from the program on.
	program []string
	// spawns says whether a wrapper runs the program in a process of its
	// own, where a cd, pushd or popd moves no shell, or fails to run it.
	spawns bool
	// dir is the folder env -C or sudo -D runs the program in, relative to
	// the shell's; "" for the shell's own.
	dir string
	// named are the words from each wrapper named by a path on: a file there
	// may be any program, a copied loomux too, which strict mode judges.
	named [][]string
}

// note keeps words as a call of their own when the wrapper at their head is
// named by a path.
func (p *prefixes) note(words []string) {
	if strings.ContainsAny(words[0], `/\`) {
		p.named = append(p.named, words)
	}
}

// wrapped reads the flags of the wrapper name at the head of words, notes
// the folder they set and whether the wrapper is a program of its own rather
// than the shell's builtin, and says how many words they take.
func (p *prefixes) wrapped(name string, words []string, builtin bool) int {
	n, dir := wrapperFlags(name, words)
	p.dir = followed(p.dir, dir)
	p.spawns = p.spawns || !builtin
	return n
}

// readPrefixes is dropPrefixes with what the prefixes do to the program.
func readPrefixes(words []string) (read prefixes) {
	for len(words) > 0 {
		w := words[0]
		n := 1
		isRedirect, bare := redirection(w)
		// base matches the shell's own words by the exact word, never a path
		// to a file of that name; name matches an external wrapper, which
		// several come as <name>.exe (Git for Windows ships env, xargs,
		// nice, stdbuf, nohup, winpty and timeout; Windows has sudo.exe,
		// cmd.exe and timeout.exe), and trimming .exe off a name is safe.
		base := baseName(w)
		if slices.Contains(shellWords, base) && base != strings.ToLower(w) {
			// A file named like a word of the shell is a program like any
			// other, a copied loomux too.
			base = ""
		}
		name := verbOf(w)
		switch {
		case isRedirect:
			if bare {
				n = 2
			}
		case strings.Contains(w, "=") && !strings.HasPrefix(w, "-"):
		case base == "{" || base == "!" || base == "if" || base == "then" || base == "else" ||
			base == "elif" || base == "while" || base == "until" || base == "do" ||
			base == "coproc" || base == "try" || base == "catch" || base == "finally" ||
			w == ".":
			// A lone . is PowerShell's dot-source operator, a sibling of &.
		case base == "function":
			// The name, then the body.
			n = 2
		case base == "command":
			// The shell builtin, by the exact word; its -p, -v take no value.
			n += flagCount(words[1:])
		case base == "exec" || base == "time":
			// The shell's own builtins, by the exact word; /usr/bin/time is
			// GNU's, exec -a and time -o take a value.
			n += read.wrapped(base, words[1:], true)
		case name == "timeout":
			read.note(words)
			// The duration comes before the program.
			n += read.wrapped(name, words[1:], false) + 1
		case name != "exec" && wrapperValues[name] != nil || slices.Contains(spawnNoValueWrappers, name):
			// exec is only the shell's: a file named exec, exec.exe, is a
			// program.
			// The external wrappers: those with value flags, and winpty,
			// setsid, chronic, nohup, which have none. All run the program.
			read.note(words)
			read.spawns = true
			n += read.wrapped(name, words[1:], false)
			if name == "sudo" && n < len(words) && words[n] == "run" {
				// Sudo for Windows runs what follows its run, after flags
				// of its own.
				n++
				n += read.wrapped(name, words[n:], false)
			}
		case name == "cmd":
			read.note(words)
			read.spawns = true
			// Every switch up to /c, /k or /r, which the command follows,
			// glued to the switch (/cDIR) or the next word.
			for n < len(words) && len(words[n]) > 1 && words[n][0] == '/' {
				tail, isRun := cmdRunTail(words[n])
				n++
				if isRun {
					if tail != "" {
						words = append([]string{tail}, words[n:]...)
						n = 0
					}
					break
				}
			}
		default:
			read.program = words
			return read
		}
		words = words[min(n, len(words)):]
	}
	read.program = words
	return read
}

// cmdRunTail reads a cmd switch word: isRun whether it carries the run switch
// /c, /k or /r (also behind earlier glued switches, /d/c), and tail the
// command glued after it, "" when the command is the next word.
func cmdRunTail(word string) (tail string, isRun bool) {
	lower := strings.ToLower(word)
	best := -1
	for _, s := range []string{"/c", "/k", "/r"} {
		if at := strings.Index(lower, s); at >= 0 && (best < 0 || at < best) {
			best = at
		}
	}
	if best < 0 {
		return "", false
	}
	return word[best+2:], true
}

// shellWords are the reserved words and builtins dropPrefixes skips: the
// shell reads them only as the word itself, never as a path to a file.
var shellWords = []string{"{", "!", "if", "then", "else", "elif", "while", "until", "do", "coproc",
	"try", "catch", "finally", "function", "command", "exec", "time"}

// flagCount is how many words at the head of words are flags of a wrapper
// none of whose flags takes a value, a redirection among them included.
func flagCount(words []string) int {
	n := pastRedirections(words, 0)
	for n < len(words) && len(words[n]) > 1 && words[n][0] == '-' {
		n = pastRedirections(words, n+1)
	}
	return min(n, len(words))
}

// wrapperValues names, for each wrapper whose flags dropPrefixes reads with
// wrapperFlags, the flags that take a value: bash's exec -a, GNU time's format
// and output file, and the coreutils, findutils and sudo flags.
var wrapperValues = map[string][]string{
	"sudo": {"-u", "-g", "-C", "-D", "-h", "-p", "-r", "-R", "-t", "-U", "-T",
		"--user", "--group", "--close-from", "--chdir", "--chroot", "--host", "--prompt", "--role",
		"--type", "--other-user", "--command-timeout"},
	// -S is left out: its value is the command line itself.
	"env": {"-u", "-C", "-a", "--unset", "--chdir", "--argv0"},
	// -l, -i and -e, --max-lines, --replace and --eof take a value only
	// glued to them.
	"xargs": {"-n", "-L", "-P", "-s", "-I", "-d", "-E", "-a",
		"--max-args", "--max-procs", "--max-chars", "--delimiter", "--arg-file", "--process-slot-var"},
	"nice":    {"-n", "--adjustment"},
	"timeout": {"-s", "-k", "--signal", "--kill-after"},
	"exec":    {"-a"},
	"time":    {"-f", "-o", "--format", "--output"},
	"stdbuf":  {"-i", "-o", "-e", "--input", "--output", "--error"},
	"ionice":  {"-c", "-n", "-p", "-P", "-u", "--class", "--classdata", "--pid", "--pgid", "--uid"},
	// unbuffer passes its arguments to Expect's spawn, whose -ignore, -open
	// and -leaveopen take the next word; -p takes none.
	"unbuffer": {"-ignore", "-open", "-leaveopen"},
}

// spawnNoValueWrappers are the external wrappers that run the program and
// whose own flags take no value: winpty, setsid, chronic and nohup.
var spawnNoValueWrappers = []string{"winpty", "setsid", "chronic", "nohup"}

// chdirFlags are the flags that set the folder a wrapper runs its program in.
var chdirFlags = map[string][]string{"env": {"-C", "--chdir"}, "sudo": {"-D", "--chdir"}}

// wrapperFlags is how many words at the head of words are the wrapper's
// flags, a flag's separate value included (flagValue), and the folder a flag
// of chdirFlags sets. A redirection among them counts as well, also between a
// flag and its value: the shell takes it out of the words before the wrapper
// reads them. -- ends them and counts; env's lone - is its -i.
func wrapperFlags(wrapper string, words []string) (n int, dir string) {
	n = pastRedirections(words, 0)
	for n < len(words) {
		w := words[n]
		if w == "--" {
			return n + 1, dir
		}
		if (len(w) < 2 || w[0] != '-') && (wrapper != "env" || w != "-") {
			break
		}
		n++
		name, value, next := flagValue(wrapper, w)
		if next {
			n = pastRedirections(words, n)
			if n < len(words) {
				value = words[n]
			}
			n++
		}
		if slices.Contains(chdirFlags[wrapper], name) {
			dir = value
		}
		n = pastRedirections(words, n)
	}
	return min(n, len(words)), dir
}

// flagValue reads flag the way getopt reads the wrapper's flags: name is the
// flag of wrapperValues it is or abbreviates, "" for one that takes no value;
// value is its value when glued to it, and next says whether the next word
// is its value instead. A long option counts by its name or by any prefix of
// it (--sig for --signal), with its value after = or in the next word; a short
// one alone or last in a bundle (-Hu root) takes the next word, one before
// the last the rest of its own (-uroot).
func flagValue(wrapper, flag string) (name, value string, next bool) {
	values := wrapperValues[wrapper]
	if slices.Contains(values, flag) {
		return flag, "", true
	}
	if strings.HasPrefix(flag, "--") {
		long, glued, hasValue := strings.Cut(flag, "=")
		at := slices.IndexFunc(values, func(v string) bool { return strings.HasPrefix(v, long) })
		if at < 0 {
			return "", "", false
		}
		return values[at], glued, !hasValue
	}
	for i := 1; i < len(flag); i++ {
		if letter := "-" + flag[i:i+1]; slices.Contains(values, letter) {
			return letter, flag[i+1:], i == len(flag)-1
		}
	}
	return "", "", false
}

// pastRedirections is the index of the first word from n on that is no
// redirection or the target of a bare one.
func pastRedirections(words []string, n int) int {
	for n < len(words) {
		isRedirect, bare := redirection(words[n])
		if !isRedirect {
			return n
		}
		n++
		if bare {
			n++
		}
	}
	return n
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
	if verbOf(words[0]) != "go" || len(words) < 2 || words[1] != "run" {
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

// knownTools are programs whose subcommands share loomux's names (git
// config set, gh config set, npm init, terraform init). With the verb table
// and readVerbs they are the programs strict mode does not take for a renamed
// loomux, by their bare name only: a path to a file of that name may be
// anything.
var knownTools = []string{"git", "gh", "go", "npm", "npx", "pnpm", "yarn", "cargo", "uv", "uvx",
	"poetry", "pip", "pip3", "docker", "dotnet", "terraform", "kubectl", "helm", "make", "cmake"}

// knownProgram says whether word is the bare name of a program strict mode
// leaves to its name.
func knownProgram(word string) bool {
	if strings.ContainsAny(word, `/\`) {
		return false
	}
	name := verbOf(word)
	for _, list := range [][]string{knownTools, readVerbs, everyFileWrites, everyFileRemoves,
		moveVerbs, renameVerbs, copyVerbs, otherWriteVerbs} {
		if slices.Contains(list, name) {
			return true
		}
	}
	return false
}

// programArgs are the arguments loomux would get from words: loomuxArgs,
// and in strict mode the arguments of every program knownProgram does not
// name -- a copied or renamed binary (doc.exe flow resume … --answer) is
// loomux by what it is told, not by what it is called.
func programArgs(words []string, anyProgram bool) ([]string, bool) {
	if args, ok := loomuxArgs(words); ok || !anyProgram || knownProgram(words[0]) {
		return args, ok
	}
	return words[1:], true
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
