// Package mirror reads which directories a worktree must not own itself.
//
// Its own package because the answer is needed before anything else can run
// and by more than one subcommand -- and because it is the only place that
// decides what a configured path may look like.
package mirror

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// file is the shape this package reads out of config.toml, and nothing else.
// Every other table in that file belongs to internal/config; an unknown one is
// no concern of ours, which is what the pointer expresses: absent stays absent.
type file struct {
	Worktree *struct {
		Mirror []string `toml:"mirror"`
	} `toml:"worktree"`
}

// Mirror returns the configured paths, in the order the file names them.
//
// Three ways of having nothing to do, all of them an empty list and no error:
// no `config.toml`, a `config.toml` without `[worktree]`, and a `[worktree]`
// without `mirror`. A hook that fires in every project on the machine meets
// all three constantly, and reporting any of them as a failure would train
// its user to ignore it.
//
// Damage is the opposite case and *is* an error. Read as "nothing to mirror",
// a broken file would switch the mechanism off silently -- and the thing being
// switched off is what puts `.ultraloom/vendor` in place, so the next symptom
// would be every other hook failing for an unrelated-looking reason.
func Mirror(root string) ([]string, error) {
	path := filepath.Join(root, ".loomux", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var parsed file
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if parsed.Worktree == nil || len(parsed.Worktree.Mirror) == 0 {
		return nil, nil
	}
	cleaned := make([]string, 0, len(parsed.Worktree.Mirror))
	for _, entry := range parsed.Worktree.Mirror {
		relative, err := inside(entry)
		if err != nil {
			return nil, fmt.Errorf("%s: [worktree].mirror: %w", path, err)
		}
		cleaned = append(cleaned, relative)
	}
	return cleaned, nil
}

// inside refuses a path that does not stay below the project.
//
// Checked here rather than at the call site: this is the one entry a project
// controls, and the thing built from it is a link into another directory.
//
// `filepath.IsLocal` is the only test applied, and it is the platform's own
// answer to this question -- but what it guarantees is not the same
// everywhere. On every platform it rejects an empty path, an absolute or
// rooted one, and any `..` segment. Only on Windows does it also reject a
// reserved device name and read a backslash as a separator: on POSIX a
// backslash is an ordinary byte, so `C:\absolute` is one legal filename there
// and passes. A project that writes Windows paths into a config read by a
// Linux worktree therefore gets a directory with an odd name, not a refusal.
//
// The empty-path branch below is not a gap in `IsLocal`, which rejects ""
// already; it exists so the message names the cause instead of quoting an
// empty string back at the reader.
func inside(entry string) (string, error) {
	if entry == "" {
		return "", fmt.Errorf("an empty path names nothing")
	}
	native := filepath.FromSlash(entry)
	if !filepath.IsLocal(native) {
		return "", fmt.Errorf("%q leaves the project", entry)
	}
	return filepath.ToSlash(filepath.Clean(native)), nil
}
