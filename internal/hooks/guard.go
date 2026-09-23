// Package hooks holds what loomux runs from a harness lifecycle event: the
// pre-tool-use guard that answers with the project's policy and the global
// write barrier, the post-tool-use lanes, the session start, the status report
// and the worktree mirror.
package hooks

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
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

// Built-in rules that protect secrets, stop gate controls, and lock files.
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
	// The literal below is the second copy of sessions.StateDir; a rule is a
	// verbatim glob here, so the two are kept in step by hand.
	{Match: []string{".loomux/state/hooks/**"}, Reason: "the stop gate's own controls are not written by the party it gates"},
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
	manifest := manifestWriteSource()
	return []config.CommandRule{{
		Regex:  regexp.MustCompile(`(^|\s)git\s+push(\s|$)`),
		Source: `(^|\s)git\s+push(\s|$)`,
		Reason: "Whether commits reach the remote is a human's decision.",
	}, {
		Regex:  regexp.MustCompile(manifest),
		Source: manifest,
		Reason: ".loomux/config.toml: the manifest is where the barrier reads its own limits, so no shell command may write it",
	}}
})

// manifestWriteSource is the expression for a shell line that writes the
// manifest: the barrier refuses it to every writing tool, and a shell line is
// the same write by another road.
//
// It reads command text, not a file system, so it is a net with known holes
// rather than a proof: a path held in a variable (`> "$M"`), a program that
// opens the file itself (`python -c …`) and a command spelled through an alias
// the list does not know all pass. It errs the other way where it cannot tell:
// a `>` inside a quoted string, or the manifest named as the source of a
// `mv` or an `Out-File -InputObject`, is refused. One rule rather than one per
// form, because `checkTool` names a reason once per matching rule and a line
// like `tee M > M` would otherwise say the same thing twice.
//
// Every form stays inside one segment of the line -- nothing between the
// command and the manifest crosses `;`, `|`, `&` or a line break -- so a write
// elsewhere on the line and a read of the manifest do not add up to a refusal.
func manifestWriteSource() string {
	const (
		// The manifest as one shell word: quoted or not, under any directory,
		// or glued to a parameter (`of=`, `-FilePath:`); either slash, any case.
		name = `['"]?(?:[^\s;|&'"<>]*[/\\=:])?\.loomux[/\\]+config\.toml['"]?`
		word = name + `(?:[\s;|&),]|$)`
		last = name + `\s*(?:[;|&\n)]|$)`
		// A command at the head of a segment, behind `sudo` and friends and
		// under any directory; `$(` opens a segment as well.
		head = `(?:^|[;|&({\n])\s*(?:(?:sudo|command|exec|nohup)\s+)*(?:[^\s;|&]*[/\\])?`
		exe  = `(?:\.exe)?\s`
		// The rest of the segment up to the word that names the manifest.
		rest = `(?:[^;|&\n]*[\s(,])?`
		// `sed -i`, `-i.bak`, `-Ei`, `--in-place`; case-sensitive, because
		// `perl -I` is an include path.
		inPlace = `(?-i:-[A-Za-z]*i\S*|--in-place\S*)`
	)
	forms := []string{
		// A redirect into it: `>`, `>>`, `>|`, `2>`, `&>`.
		`>[>|!]?\s*` + word,
		// Commands that write or remove every file they name.
		head + `(?:tee|tee-object|set-content|add-content|out-file|clear-content|ac|` +
			`mv|move|move-item|mi|rename-item|ren|rni|rm|del|erase|remove-item|ri|truncate)` +
			exe + rest + word,
		// Commands that write only their destination.
		head + `(?:cp|copy|copy-item|cpi|install)` + exe + rest + last,
		head + `(?:cp|copy|copy-item|cpi|install)` + exe + `(?:[^;|&\n]*\s)?-dest\w*[:\s]\s*` + word,
		head + `dd` + exe + `(?:[^;|&\n]*\s)?of=` + word,
		head + `git\s+(?:-\S+\s+)*(?:mv|rm|checkout|restore)\s` + rest + word,
		// An in-place edit, with the flag before or after the file.
		head + `(?:sed|perl)` + exe + `(?:[^;|&\n]*\s)?` + inPlace + `\s` + rest + word,
		head + `(?:sed|perl)` + exe + rest + word + `(?:[^;|&\n]*\s)?` + inPlace,
		// .NET from PowerShell.
		`\[(?:system\.)?io\.file\]::(?:write|append|create|delete|move|replace)\w*\s*\(` + rest + word,
	}
	return `(?i)` + strings.Join(forms, "|")
}

// commandTools are the tools whose "command" argument is a shell line.
var commandTools = map[string]bool{"Bash": true, "PowerShell": true}

// matchGlob matches a slash-separated path against a glob pattern supporting
// `**`, and answers an error for a pattern it cannot read.
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
// slash, which are matched against the base name, still reach it. Those
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
			for _, rule := range append(builtinPathRules, policy.Paths...) {
				for _, glob := range rule.Match {
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
		}
	}
	if commandTools[tool] {
		if line, ok := input["command"].(string); ok {
			for _, rule := range append(builtinCommands(), policy.Commands...) {
				if rule.Regex.MatchString(line) {
					reasons = append(reasons, rule.Reason)
				}
			}
		}
	}
	return reasons
}
