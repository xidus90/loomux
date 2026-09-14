package guard

import (
	"os"
	"path/filepath"
)

// The scratchpad of a Claude Code session is the third place outside the
// registered trees the barrier leaves open (loomux spec, parity list of stage
// 1a). Claude Code hands every session a directory
// <temp>/claude/<project>/<session>/scratchpad for its throwaway files; the
// barrier refused it on 2026-09-14 and the session had nowhere else to put them.

var tempDir = os.TempDir

// scratchpadBase resolves <temp>/claude, or "" where temp is not absolute or
// does not resolve. Like the memory bases it may only ever open.
func scratchpadBase() string {
	temp := tempDir()
	if !filepath.IsAbs(temp) {
		return ""
	}
	resolved, err := resolvePath(filepath.Join(temp, "claude"))
	if err != nil {
		return ""
	}
	return resolved
}

// isScratchpad reports whether a resolved target lies below
// <base>/<project>/<session>/scratchpad/, counted in components.
func isScratchpad(resolved, base string) bool {
	if base == "" {
		return false
	}
	rest, ok := below(spelled(resolved), base)
	return ok && len(rest) >= 4 && rest[2] == "scratchpad"
}
