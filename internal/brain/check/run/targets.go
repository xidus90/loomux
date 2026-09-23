package run

import (
	"fmt"
	"os"

	"github.com/xidus90/loomux/internal/config"
)

// Targets answers the areas a check run may read, and refuses the ones it
// could only pretend to read.
//
// It is the Go twin of `_lint_targets` (`src/brain/cli.py:1466-1496`),
// and it sits beside the three widths rather than inside them for the
// reason that function gives: `Path.rglob` on a missing directory yields
// no pages at all, "and every rule would then read that silence as a
// clean bundle rather than as a bundle it never saw". `CheckBundle`
// cannot say so itself -- it answers findings, and this is not a finding
// but the statement that no run took place, which is what
// `TestABundleThatCannotBeWalkedIsNotADefect` asserts of it and what the
// exit code 2 of `cmd/brain` is for.
//
// A finding was the alternative and was rejected twice over: it would add
// a thirty-second name to a catalog the design counts at thirty-one
// (design 5.3), and it would put a broken environment among rules that
// all describe a defect in a page or a bundle. Python raises here rather
// than reporting, and raising is what exit 2 already means in this
// binary.
//
// The asymmetry between the two paths is Python's and deliberate: naming
// an area without a wiki is a mistake worth a message, skipping one in
// the sweep is not, because most areas will never carry a bundle.
func Targets(areas []config.Area, scope string) ([]config.Area, error) {
	if scope == "all" {
		var out []config.Area
		for _, area := range areas {
			// An area that declares no wiki is passed over without a
			// word; only a declared one is held to existing.
			if area.WikiPath == "" {
				continue
			}
			if err := existing(area); err != nil {
				return nil, err
			}
			out = append(out, area)
		}
		return out, nil
	}
	for _, area := range areas {
		if area.Scope != scope {
			continue
		}
		if area.WikiPath == "" {
			return nil, fmt.Errorf(
				"area %q declares no wiki path; "+
					"add `wiki = ...` to its entry", scope)
		}
		if err := existing(area); err != nil {
			return nil, err
		}
		return []config.Area{area}, nil
	}
	return nil, fmt.Errorf("no area named %q in the registry", scope)
}

// existing is `_existing` of `src/brain/cli.py:1478-1484`: the wiki path
// must be a directory, not merely a name that resolves. A plain file
// passes an existence test and fails this one, as `Path.is_dir` does --
// walking a file yields the same silence a missing directory does.
//
// The message is Python's, with one word changed: it names `loomux wiki
// init`, the command that answers it here, where Python names `brain wiki
// init`. Advice naming a command this binary does not have would send the
// reader nowhere; the refusals are compared on exit and stdout, and the
// message goes to stderr.
func existing(area config.Area) error {
	info, err := os.Stat(area.WikiPath)
	if err == nil && info.IsDir() {
		return nil
	}
	return fmt.Errorf(
		"area %q has no wiki at %s; "+
			"run `loomux wiki init --scope %s` first",
		area.Scope, area.WikiPath, area.Scope)
}
