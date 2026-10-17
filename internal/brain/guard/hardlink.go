package guard

import "fmt"

// HardLinked is the refusal of a write to a file that has more than one
// name, or "" where the path names one file of one name, a directory or
// nothing yet.
//
// A hard link is not a link the resolver could follow: both names are the
// file itself, so `ln ../B/.loomux/config.toml notes.txt` turns every rule
// about names into a rule about one of them. Where the other names lie is
// not asked. Posix keeps no list of them, and on Windows each would need the
// whole judgement of both stages, including the policy of a project this
// hook does not know -- an unanswerable name is refused anyway, so the
// count alone decides. Nothing an agent edits keeps a second name: pnpm and
// uv link their stores into node_modules and .venv, where a write changes
// every project that shares the store.
//
// The policy in `internal/hooks` asks the same question of each path a
// call writes, so both stages refuse in the same words.
func HardLinked(path string) string {
	links, err := linkCount(path)
	switch {
	case err != nil && !stopsResolving(err):
		return fmt.Sprintf("%s: loomux cannot count this file's links, so it refuses: %v", path, err)
	case links > 1:
		return fmt.Sprintf("%s: this file has %d links, and a write through one "+
			"name changes the file under every other, wherever it lies; no tool "+
			"may write a file with more than one link", path, links)
	}
	return ""
}
