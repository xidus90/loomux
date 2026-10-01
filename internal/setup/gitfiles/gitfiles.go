// Package gitfiles holds the git files loomux writes into a host project:
// the three hooks and the lines its .gitignore needs.
package gitfiles

import "strings"

// gitignoreComment heads the lines WithGitignore appends.
const gitignoreComment = "# loomux: local state and backups"

// prePush keeps agents and hurried humans off the default branch; it follows
// .githooks/pre-push of loomux itself, extended by main.
const prePush = `#!/bin/sh
# loomux pre-push hook: nobody but a human pushes to the default branch, and
# a human reads this before overriding it.
while read -r local_ref local_sha remote_ref remote_sha; do
	case "$remote_ref" in
	refs/heads/main | refs/heads/master)
		echo "pre-push: pushing to ${remote_ref#refs/heads/} is refused; open a pull request" >&2
		exit 1
		;;
	esac
done
exit 0
`

// Hooks are the three git hooks of a host project, keyed by file name.
// binary is the path of the loomux binary, with or without one pair of
// surrounding double quotes; the scripts quote it themselves.
func Hooks(binary string) map[string]string {
	b := binary
	if len(b) >= 2 && strings.HasPrefix(b, `"`) && strings.HasSuffix(b, `"`) {
		b = b[1 : len(b)-1]
	}
	return map[string]string{
		"pre-commit": preCommit(b),
		"commit-msg": "#!/bin/sh\n" +
			"# loomux commit-msg hook: the commit rules of [commit].\n" +
			`exec "` + b + `" check commit-msg "$1"` + "\n",
		"pre-push": prePush,
	}
}

// preCommitMarker opens the comment line of the pre-commit hook init writes;
// Upgrade knows its own older hook by it.
const preCommitMarker = "# loomux pre-commit hook:"

// preCommit is the pre-commit hook of a host project for the binary b: the
// gate with --arm, which enters the lanes a green run found ok and stages the
// file itself. The script stages nothing: only the command knows whether the
// commit takes the whole index, and a `git add` here would pull the file
// into a commit of paths.
func preCommit(b string) string {
	return "#!/bin/sh\n" +
		preCommitMarker + " the check chain of .loomux/config.toml.\n" +
		`exec "` + b + `" check precommit --arm` + "\n"
}

// Upgrade is the pre-commit hook init wrote before lanes could be armed --
// the shebang, the marker line and the one call `exec "<binary>" check
// precommit`, nothing else -- in today's form for the same binary. False for
// every other text: a hook with a line of its own is the project's.
func Upgrade(text string) (string, bool) {
	lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\n")
	if len(lines) != 3 || lines[0] != "#!/bin/sh" || !strings.HasPrefix(lines[1], preCommitMarker) {
		return "", false
	}
	b, ok := strings.CutPrefix(lines[2], `exec "`)
	if !ok {
		return "", false
	}
	b, ok = strings.CutSuffix(b, `" check precommit`)
	if !ok {
		return "", false
	}
	return preCommit(b), true
}

// IsLoomuxPreCommit says whether a hook's text carries the marker line of the
// pre-commit hook init writes, in the old form or today's, with or without
// lines a human added: the sign that loomux was set up in this project.
func IsLoomuxPreCommit(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, preCommitMarker) {
			return true
		}
	}
	return false
}

// RunsAGate says whether an existing hook already runs a check chain: a
// loomux, ultraloom or ulguard call, or ci/gate.sh. Comment lines do not
// count, so a hook that merely mentions a chain is not taken for one.
func RunsAGate(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		for _, marker := range []string{"loomux", "ultraloom", "ulguard", "ci/gate.sh"} {
			if strings.Contains(line, marker) {
				return true
			}
		}
	}
	return false
}

// GitignoreLines are the lines a host project's .gitignore needs; the state
// directory also holds the backups of files the installer replaces.
func GitignoreLines() []string {
	return []string{"/.loomux/state/"}
}

// WithGitignore returns text with the missing GitignoreLines appended under
// one comment line, and whether it changed. A line counts as present with or
// without its leading and trailing slash. The file's line ending is kept.
func WithGitignore(text string) (string, bool) {
	present := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		present[strings.Trim(strings.TrimSpace(line), "/")] = true
	}
	var missing []string
	for _, line := range GitignoreLines() {
		if !present[strings.Trim(line, "/")] {
			missing = append(missing, line)
		}
	}
	if len(missing) == 0 {
		return text, false
	}
	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	var b strings.Builder
	b.WriteString(text)
	if text != "" && !strings.HasSuffix(text, "\n") {
		b.WriteString(eol)
	}
	b.WriteString(gitignoreComment + eol)
	for _, line := range missing {
		b.WriteString(line + eol)
	}
	return b.String(), true
}
