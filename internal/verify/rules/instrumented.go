package rules

import (
	"bytes"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/verify/linecov"
)

// instrumentationProfiles is a variable so a test can fail the profile load.
var instrumentationProfiles = linecov.LoadProfiles

// Instrumented reports every file whose first line is the marker of a coverage
// tool the project uses: the tool rewrote the source on disk and a killed run
// never restored it. The tool is detected on disk below root (its settings
// file); the first line comes from the file's Content, so a staged version is
// judged as staged. Files of a layer with checked = false count too.
func Instrumented(files []File, root string) ([]Finding, error) {
	ps, err := instrumentationProfiles()
	if err != nil {
		return nil, err
	}
	tree, err := linecov.ScanTree(root)
	if err != nil {
		return nil, err
	}
	// An unreadable settings file would leave the tool undetected and the
	// check silent: fail instead. Notes (ambiguous matches) stay dropped.
	active, finds, _ := linecov.Activate(tree, nil, ps)
	if len(finds) > 0 {
		return nil, fmt.Errorf("instrumentation check: %s", finds[0])
	}
	var out []Finding
	for _, a := range active {
		marker := a.Profile.Instrumented
		if marker == "" {
			continue
		}
		for _, f := range files {
			if slices.Contains(a.Profile.Extensions, path.Ext(f.Rel)) && firstLine(f.Content) == marker {
				out = append(out, Finding{Path: f.Rel, Text: "still holds " + a.Profile.Name + " instrumentation; restore the sources before committing"})
			}
		}
	}
	slices.SortFunc(out, func(a, b Finding) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

// firstLine is the first line without BOM, line ending and surrounding blanks.
func firstLine(content []byte) string {
	first, _, _ := bytes.Cut(bytes.TrimPrefix(content, []byte("\xef\xbb\xbf")), []byte("\n"))
	return string(bytes.TrimSpace(first))
}
