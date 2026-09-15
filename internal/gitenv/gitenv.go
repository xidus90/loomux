// Package gitenv keeps git's own environment out of ultraloom's git calls.
//
// Its own package because two programs need the same answer: ulinit reads
// facts about a project, and ulguard will do the same for a worktree. A second
// copy of the list would drift in exactly the entry that matters. The list is
// ultra-brain's, because it cuts identity and configuration as well as the
// repository pointers.
package gitenv

import (
	"os"
	"strings"
)

// Location is what git or a caller around us exports that decides which
// repository a git child acts on, how it is configured and whom it writes as
// -- among them what git hands the hooks this project runs, the client-side
// pre-commit and commit-msg -- and what has to be taken back out before a
// child of ours starts.
//
// Deliberately not "every hook": a server-side hook gets more, and there this
// strip would be actively wrong. git 2.54.0's `git-receive-pack` documentation
// (QUARANTINE ENVIRONMENT) has incoming objects in a temporary store that is
// migrated only after `pre-receive` has finished, and the name
// GIT_QUARANTINE_PATH stands in that version's `git.exe`; taking the object
// store pointers out inside such a hook hides exactly the objects the push is
// about. `githooks` adds GIT_PUSH_OPTION_COUNT and GIT_PUSH_OPTION_<n> for
// `pre-receive` and `update`, which this list does not name, and
// gitnamespaces has GIT_NAMESPACE where a namespace is in use, which it does
// name -- stripped inside such a hook, the push would leave its namespace.
// Nothing in ultraloom runs a server-side hook; whoever reuses this list for
// one has to weigh it again.
//
// The repository, index and object store entries *outrank* the directory a
// command is given to work in: a git call with its working directory set to
// one repository still answers about the one these point at. Measured on
// 2026-09-07 out of a worktree, where GIT_DIR is absolute: `git rev-parse
// --absurd-flag` in a scratch directory came back a success, and reading an
// unset setting answered `.githooks`. In the main checkout the exported value
// is the relative `.git`, which resolves inside the scratch repository by luck
// and hides the whole effect.
//
// A named list rather than a GIT_ prefix cut: GIT_EDITOR, GIT_TERMINAL_PROMPT
// and GIT_TRACE are the user's settings and none of our business. Identity and
// configuration are on it all the same, because a child that inherits them
// writes as someone else or into a config it was never pointed at: on
// 2026-08-28 a test's `git config user.name` in ultra-brain landed in the real
// `.git/config` through an inherited GIT_DIR and signed 709 commits.
var Location = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_COMMON_DIR",
	"GIT_NAMESPACE",
	"GIT_PREFIX",
	"GIT_CEILING_DIRECTORIES",
	"GIT_GRAFT_FILE",
	"GIT_CONFIG",
	"GIT_CONFIG_PARAMETERS",
	"GIT_CONFIG_COUNT",
	"GIT_CONFIG_GLOBAL",
	"GIT_CONFIG_SYSTEM",
	"GIT_AUTHOR_NAME",
	"GIT_AUTHOR_EMAIL",
	"GIT_AUTHOR_DATE",
	"GIT_COMMITTER_NAME",
	"GIT_COMMITTER_EMAIL",
	"GIT_COMMITTER_DATE",
	"GIT_ATTR_SOURCE",
	"GIT_ATTR_NOSYSTEM",
	"GIT_LITERAL_PATHSPECS",
	"GIT_GLOB_PATHSPECS",
	"GIT_NOGLOB_PATHSPECS",
	"GIT_ICASE_PATHSPECS",
	"GIT_REFLOG_ACTION",
}

// numbered names are GIT_CONFIG_KEY_<n> and GIT_CONFIG_VALUE_<n>, whose count is open.
var numbered = []string{"GIT_CONFIG_KEY_", "GIT_CONFIG_VALUE_"}

// Clean returns parent without the variables in Location and the numbered
// config pairs.
//
// Takes the parent environment rather than reading it, so the decision is
// testable without a process to inherit from. Entries are "NAME=value" as
// os.Environ spells them; anything without a separator is passed through
// untouched, because a name we cannot read is not a name we can match.
//
// The match is case-sensitive, and `src/ultraloom/gitenv.py` is not --
// os.environ upper-cases its keys on Windows, measured on 2026-09-08: after
// `os.environ["git_dir"] = "x"` the only key that reads back is GIT_DIR, so a
// lowercase spelling is stripped there and passed through here. Git writes the
// uppercase spelling, so nothing has ever produced the difference.
//
// It is written down because the case was the *only* difference for as long as
// the two lists were the same list. They are not: Location is ultra-brain's
// wider one, which cuts identity and configuration as well as the repository
// pointers, so the Python module is no mirror image of this one any more and
// nothing here may be read off it.
func Clean(parent []string) []string {
	cleaned := make([]string, 0, len(parent))
	for _, entry := range parent {
		name, _, found := strings.Cut(entry, "=")
		if found && (contains(Location, name) || hasAnyPrefix(name, numbered)) {
			continue
		}
		cleaned = append(cleaned, entry)
	}
	return cleaned
}

// Environ is Clean over this process's own environment.
func Environ() []string {
	return Clean(os.Environ())
}

func contains(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

func hasAnyPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
