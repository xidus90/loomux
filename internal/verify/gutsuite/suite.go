// Package gutsuite runs a Godot project's GUT suite headless, for Godot 3
// (GUT 7) and Godot 4 (GUT 9), and judges it by GUT's JUnit report, never by
// the engine's exit code alone: Godot 4 without its import exits 0 and
// writes no report, Godot 3 without its class list exits 0 with a report of
// no case. Both are findings here.
package gutsuite

import (
	"cmp"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify"
)

// Options is one run.
type Options struct {
	// Checkout holds the state directory; the project root when empty.
	Root, Checkout, Godot, RunID string
	Start                        func(child.Spec) child.Result
}

// Result is what the run found; no finding is a green run.
type Result struct {
	Findings, Notes []string
}

const cmdln = "addons/gut/gut_cmdln.gd"

// defaultDirs are where GUT looks without a .gutconfig.json, as far as a
// project has them.
var defaultDirs = []string{"res://test", "res://tests"}

// gutConfig is the part of .gutconfig.json that says which scripts run.
type gutConfig struct {
	Dirs           []string `json:"dirs"`
	Prefix         *string  `json:"prefix"`
	Suffix         *string  `json:"suffix"`
	IncludeSubdirs *bool    `json:"include_subdirs"`
}

// rule is which files GUT runs: in dirs, named prefix…suffix, below them too
// with subdirs.
type rule struct {
	dirs           []string
	prefix, suffix string
	subdirs        bool
	config         bool // from .gutconfig.json
}

func ruleOf(root string) (rule, error) {
	r := rule{dirs: defaultDirs, prefix: "test_", suffix: ".gd", subdirs: true}
	data, err := os.ReadFile(filepath.Join(root, ".gutconfig.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return r, nil
	}
	var c gutConfig
	if err == nil {
		err = json.Unmarshal(data, &c)
	}
	if err != nil {
		return rule{}, fmt.Errorf(".gutconfig.json cannot be read: %w", err)
	}
	// With a config, GUT includes subdirectories only when it says so (GUT 7
	// and 9: _include_subdirectories = false); the true above is loomux's own
	// -ginclude_subdirs, which a config replaces.
	r.config, r.subdirs = true, false
	if len(c.Dirs) > 0 {
		r.dirs = c.Dirs
	}
	if c.Prefix != nil {
		r.prefix = *c.Prefix
	}
	if c.Suffix != nil {
		r.suffix = *c.Suffix
	}
	if c.IncludeSubdirs != nil {
		r.subdirs = *c.IncludeSubdirs
	}
	return r, nil
}

// Scripts are the test scripts GUT will run, root-relative and sorted, and
// the directories it looks in.
func Scripts(root string) (scripts, dirs []string, err error) {
	r, err := ruleOf(root)
	if err != nil {
		return nil, nil, err
	}
	scripts, _ = r.scripts(root)
	return scripts, r.dirs, nil
}

// scripts finds the files of r and the directories of r that exist.
func (r rule) scripts(root string) (scripts, present []string) {
	for _, dir := range r.dirs {
		base := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(dir, "res://")))
		if info, err := os.Stat(base); err != nil || !info.IsDir() {
			continue
		}
		present = append(present, dir)
		_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, _ error) error {
			switch {
			case d.IsDir() && p != base && !r.subdirs:
				return filepath.SkipDir
			case d.IsDir():
				return nil
			case strings.HasPrefix(d.Name(), r.prefix) && strings.HasSuffix(d.Name(), r.suffix):
				rel, _ := filepath.Rel(root, p)
				scripts = append(scripts, filepath.ToSlash(rel))
			}
			return nil
		})
	}
	slices.Sort(scripts)
	return slices.Compact(scripts), present
}

// Run runs the suite once and judges its report.
func Run(o Options) (Result, error) {
	if _, err := os.Stat(filepath.Join(o.Root, filepath.FromSlash(cmdln))); err != nil {
		return Result{Findings: []string{"GUT is not installed: " + cmdln + " is missing"}}, nil
	}
	r, err := ruleOf(o.Root)
	if err != nil {
		return Result{}, err
	}
	planned, present := r.scripts(o.Root)
	if len(planned) == 0 {
		return Result{Findings: []string{fmt.Sprintf("no GUT test script (%s*%s) under %s", r.prefix, r.suffix, strings.Join(r.dirs, ", "))}}, nil
	}
	state := filepath.Join(cmp.Or(o.Checkout, o.Root), ".loomux", "state", "gut-suite", o.RunID)
	if err := os.MkdirAll(state, 0o755); err != nil {
		return Result{}, err
	}
	report := filepath.Join(state, "junit.xml")
	window := "--headless"
	if _, godot3 := verify.ClassListMissing(o.Root); godot3 {
		window = "--no-window"
	}
	argv := []string{o.Godot, window, "--path", o.Root, "-s", "res://" + cmdln}
	if r.config {
		argv = append(argv, "-gconfig=res://.gutconfig.json")
	} else {
		argv = append(argv, "-gdir="+strings.Join(present, ","), "-ginclude_subdirs", "-gprefix="+r.prefix)
	}
	argv = append(argv, "-gexit", "-gjunit_xml_file="+report)
	start := o.Start
	if start == nil {
		start = child.Run
	}
	ran := start(child.Spec{Dir: o.Root, Argv: argv})
	findings := verdict(report, planned, ran)
	if len(findings) == 0 {
		_ = os.RemoveAll(state)
	}
	return Result{Findings: findings}, nil
}

type junitReport struct {
	Suites []struct {
		Name  string `xml:"name,attr"`
		Cases []struct {
			Name     string `xml:"name,attr"`
			Failures []struct {
				Text string `xml:",chardata"`
			} `xml:"failure"`
		} `xml:"testcase"`
	} `xml:"testsuite"`
}

// verdict reads GUT's report against the planned scripts and the engine's
// result.
func verdict(report string, planned []string, ran child.Result) []string {
	output := tail(ran.Stdout + ran.Stderr)
	data, err := os.ReadFile(report)
	if err != nil {
		return []string{"GUT wrote no report: " + output}
	}
	var rep junitReport
	if err := xml.Unmarshal(data, &rep); err != nil {
		return []string{"GUT's report cannot be read: " + err.Error()}
	}
	var findings []string
	seen := map[string]bool{}
	cases := 0
	for _, s := range rep.Suites {
		name := strings.TrimPrefix(s.Name, "res://")
		seen[name] = true
		for _, c := range s.Cases {
			cases++
			for _, f := range c.Failures {
				findings = append(findings, name+" "+c.Name+": "+firstLine(f.Text))
			}
		}
	}
	if cases == 0 {
		return []string{"GUT ran no test: " + output}
	}
	for _, p := range planned {
		// GUT names an inner test class's suite <script>.<Class>.
		inner := func(name string) bool { return strings.HasPrefix(name, p+".") }
		if !seen[p] && !slices.ContainsFunc(slices.Collect(maps.Keys(seen)), inner) {
			findings = append(findings, p+": planned, but not in GUT's report")
		}
	}
	if len(findings) == 0 && ran.Code != 0 {
		findings = append(findings, fmt.Sprintf("the engine exited with %d: %s", ran.Code, output))
	}
	return findings
}

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}

// tail is the last lines of the engine's output, enough to say why.
func tail(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > 5 {
		lines = lines[len(lines)-5:]
	}
	return strings.Join(lines, " | ")
}
