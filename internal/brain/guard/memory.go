package guard

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The agents' memory is one of the places outside the registered trees that
// the barrier leaves open; the others are a case's `proposal.md` and the
// session scratchpad (scratchpad.go) (spec
// 2026-09-13-schranke-memory-offen). Without it an agent under this barrier
// can remember nothing, in any project, for any user. The trees are derived
// from the home directory instead of registered, so no machine needs a line of
// configuration for them.

// userHome and claudeConfigDir are the two inputs, as variables so a test can
// set both without touching the machine it runs on.
var (
	userHome        = os.UserHomeDir
	claudeConfigDir = func() string { return os.Getenv("CLAUDE_CONFIG_DIR") }
)

// antigravityRoots are the three Antigravity homes found under ~/.gemini on
// 2026-09-13. antigravity-backup lies beside them and is left shut on
// purpose: the spec opens these three roots and no other.
var antigravityRoots = []string{"antigravity", "antigravity-cli", "antigravity-ide"}

// memoryBases resolves where the memory trees start: Claude Code's projects
// directory, or "" where none can be named, and every Antigravity home.
//
// A base that does not resolve is left out instead of refused. The exemption
// may only ever open, so its failure has to close nothing that was open
// before -- the barrier then decides exactly as it did without it.
//
// A home or CLAUDE_CONFIG_DIR that is not absolute counts as unset: a
// relative one would be anchored at the hook's working directory, whatever
// repository that is. An unset CLAUDE_CONFIG_DIR falls back to
// <home>/.claude, so a relative one does as well. On Windows the same test
// also drops a rooted path without a drive, `\Users\x\.claude`, which would
// land on the working directory's drive -- a closed tree, never an opened one.
func memoryBases() (claude string, antigravity []string) {
	home, err := userHome()
	if err != nil || !filepath.IsAbs(home) {
		home = ""
	}
	config := claudeConfigDir()
	if !filepath.IsAbs(config) {
		config = ""
	}
	if config == "" && home != "" {
		config = filepath.Join(home, ".claude")
	}
	if config != "" {
		if resolved, err := resolvePath(filepath.Join(config, "projects")); err == nil {
			claude = resolved
		}
	}
	if home == "" {
		return claude, nil
	}
	for _, name := range antigravityRoots {
		if resolved, err := resolvePath(filepath.Join(home, ".gemini", name)); err == nil {
			antigravity = append(antigravity, resolved)
		}
	}
	return claude, antigravity
}

// isMemory reports whether one resolved path lies in an agent's memory.
//
// Components are counted, the way `inside` counts them, so `memory-alt` is no
// `memory` and a name one directory too deep or too shallow matches nothing.
// Only paths below a tree count, never the tree's own directory: a write
// target is a file, and without that rule `brain/t.md` would pass as a
// conversation named `t.md`.
func isMemory(resolved, claude string, antigravity []string) bool {
	target := spelled(resolved)
	if claude != "" {
		if rest, ok := below(target, claude); ok && len(rest) >= 3 && rest[1] == "memory" {
			return true
		}
	}
	for _, root := range antigravity {
		rest, ok := below(target, root)
		if !ok || len(rest) < 2 {
			continue
		}
		if rest[0] == "knowledge" || (rest[0] == "brain" && len(rest) >= 3) {
			return true
		}
	}
	return false
}

// below answers the spelled components of target past base, and whether base
// is a component prefix of target at all.
func below(target []string, base string) ([]string, bool) {
	parts := spelled(base)
	if len(parts) > len(target) || !slices.Equal(target[:len(parts)], parts) {
		return nil, false
	}
	return target[len(parts):], true
}

// memoryShown names the memory trees for a refusal, or "" where there are
// none, so a refused writer learns where it may write.
func memoryShown(claude string, antigravity []string) string {
	shown := []string{}
	if claude != "" {
		shown = append(shown, filepath.Join(claude, "*", "memory"))
	}
	for _, root := range antigravity {
		shown = append(shown, filepath.Join(root, "knowledge"), filepath.Join(root, "brain", "*"))
	}
	return strings.Join(shown, ", ")
}
