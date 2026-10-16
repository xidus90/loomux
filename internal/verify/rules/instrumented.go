package rules

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/verify/linecov"
)

// instrumentationProfiles is a variable so a test can fail the profile load.
var instrumentationProfiles = linecov.LoadProfiles

// Instrumented reports every file whose first line is the marker of a coverage
// tool the project uses: the tool rewrote the source on disk and a killed run
// never restored it. The tool is detected on disk (its settings file) in the
// root and in every directory below it holding a settings file, for the files
// below that directory: a Godot project under godot/ counts for godot/**. The
// first line comes from the file's Content, so a staged version is judged as
// staged. Files of a layer with checked = false count too.
func Instrumented(files []File, root string) ([]Finding, error) {
	ps, err := instrumentationProfiles()
	if err != nil {
		return nil, err
	}
	tree, err := linecov.ScanTree(root)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, dir := range settingsDirs(tree.Files, ps) {
		// An unreadable settings file would leave the tool undetected and the
		// check silent: fail instead. Notes (ambiguous matches) stay dropped.
		active, finds, _ := linecov.Activate(&linecov.Tree{Root: filepath.Join(tree.Root, dir)}, nil, ps)
		if len(finds) > 0 {
			finds[0].Path = path.Join(dir, finds[0].Path)
			return nil, fmt.Errorf("instrumentation check: %s", finds[0])
		}
		for _, a := range active {
			marker := a.Profile.Instrumented
			if marker == "" {
				continue
			}
			for _, f := range files {
				below := dir == "." || strings.HasPrefix(f.Rel, dir+"/")
				if below && slices.Contains(a.Profile.Extensions, path.Ext(f.Rel)) && firstLine(f.Content) == marker {
					out = append(out, Finding{Path: f.Rel, Text: "still holds " + a.Profile.Name + " instrumentation; restore the sources before committing"})
				}
			}
		}
	}
	slices.SortFunc(out, func(a, b Finding) int { return strings.Compare(a.Path, b.Path) })
	// Nested projects using the same tool both see a file below the inner one.
	return slices.Compact(out), nil
}

// settingsDirs is the root (".") and every directory below it holding a file
// named like a settings file of one of the profiles.
func settingsDirs(tree []string, ps []linecov.Profile) []string {
	dirs := []string{"."}
	for _, rel := range tree {
		dir, base := path.Dir(rel), path.Base(rel)
		if dir == "." || slices.Contains(dirs, dir) {
			continue
		}
		if slices.ContainsFunc(ps, func(p linecov.Profile) bool {
			return slices.ContainsFunc(p.Settings, func(sf linecov.SettingsFile) bool {
				ok, _ := path.Match(sf.File, base)
				return ok
			})
		}) {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// firstLine is the first line without BOM, line ending and surrounding blanks.
func firstLine(content []byte) string {
	first, _, _ := bytes.Cut(bytes.TrimPrefix(content, []byte("\xef\xbb\xbf")), []byte("\n"))
	return string(bytes.TrimSpace(first))
}
