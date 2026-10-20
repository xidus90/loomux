package compat

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
)

// LaneResult is how one lane of a check ended: its name (<kind>/<stack>),
// its verify state and what it printed.
type LaneResult struct{ Lane, State, Output string }

// RunOptions is one run of an entry: the work directory its copies go into,
// the directory holding one folder per addon name, and the check that runs
// kinds at a root. Rescan has Godot import a gdscript project before its
// lanes run. Entry, when set, is run instead of the entry named Key; it need
// not be in the list.
type RunOptions struct {
	Key, Work, Addons string
	Entry             *Entry
	Check             func(root string, kinds []string) ([]LaneResult, error)
	Rescan            func(root string) error
}

// Report is the verdict on one entry; each line is a deviation, its first
// line naming lane, state and wanted state, the rest the tail of its output.
type Report struct {
	Entry string   `json:"entry"`
	Green bool     `json:"green"`
	Lines []string `json:"lines"`
}

// tailLines is how much of a red lane's output a report line keeps.
const tailLines = 20

// addonDirs are the folders under addons/ the real addons install into.
var addonDirs = map[string]string{"GUT": "gut", "gdUnit4": "gdUnit4", "NanoCoverage": "nano_coverage_godot"}

// buildGraph is query.Build; a test stands in for a build that fails.
var buildGraph = query.Build

var featuresRe = regexp.MustCompile(`(config/features=PackedStringArray\(")\d+\.\d+`)

// Run runs the entry key for real: every kind but graph must end ok, the
// graph must hold the fixture's edges, and each break must turn exactly its
// kind red.
func Run(o RunOptions) (Report, error) {
	if o.Entry != nil {
		return runEntry(*o.Entry, o)
	}
	entries, _ := Load() // the embedded list is validated by the tests
	i := slices.IndexFunc(entries, func(e Entry) bool { return e.Key() == o.Key })
	if i < 0 {
		return Report{}, fmt.Errorf("unknown entry %q", o.Key)
	}
	return runEntry(entries[i], o)
}

func runEntry(e Entry, o RunOptions) (Report, error) {
	expect, err := LoadExpect(e.Fixture)
	if err != nil {
		return Report{}, err
	}
	work, _ := filepath.Abs(o.Work)
	work = cmp.Or(work, o.Work)
	kinds := slices.DeleteFunc(slices.Clone(e.Kinds), func(k string) bool { return k == "graph" })
	r := Report{Entry: e.Key(), Lines: []string{}}

	dir := filepath.Join(work, "entry")
	lines, err := runCopy(e, o, dir, kinds, nil)
	if err != nil {
		return Report{}, err
	}
	r.Lines = append(r.Lines, lines...)
	if slices.Contains(e.Kinds, "graph") {
		r.Lines = append(r.Lines, graphLines(dir, expect.Edges)...)
	}
	for i, b := range expect.Breaks {
		lines, err := runCopy(e, o, filepath.Join(work, fmt.Sprintf("break-%d-%s", i+1, b.Kind)), kinds, &b)
		if err != nil {
			return Report{}, err
		}
		for _, l := range lines {
			r.Lines = append(r.Lines, "break "+b.Kind+": "+l)
		}
	}
	r.Green = len(r.Lines) == 0
	return r, nil
}

// runCopy lays out a fresh work copy at dir, with b applied when not nil,
// checks kinds there and answers the deviations.
func runCopy(e Entry, o RunOptions, dir string, kinds []string, b *Break) ([]string, error) {
	if err := prepare(e, o, dir, b); err != nil {
		return nil, err
	}
	var lines []string
	if e.Stack == "gdscript" {
		if err := o.Rescan(dir); err != nil {
			lines = append(lines, "rescan: "+err.Error())
		}
	}
	results, err := o.Check(dir, kinds)
	if err != nil {
		return nil, err
	}
	broken := ""
	if b != nil {
		broken = b.Kind
	}
	return append(lines, judge(results, kinds, broken)...), nil
}

// prepare copies the fixture with the entry's addons, engine version and
// tool pins, and applies b.
func prepare(e Entry, o RunOptions, dir string, b *Break) error {
	if err := errors.Join(os.RemoveAll(dir), CopyFixture(e.Fixture, dir)); err != nil {
		return err
	}
	for _, a := range e.Addons {
		sub, ok := addonDirs[a.Name]
		if !ok {
			return fmt.Errorf("addon %s: no folder known under addons/", a.Name)
		}
		dst := filepath.Join(dir, "addons", sub)
		if err := errors.Join(os.RemoveAll(dst), os.CopyFS(dst, os.DirFS(filepath.Join(o.Addons, a.Name)))); err != nil {
			return fmt.Errorf("addon %s: %w", a.Name, err)
		}
	}
	// verify.Line matches the engine by major.minor, and one Godot 4 fixture
	// serves several versions; Godot 3 projects carry no features line.
	if e.Stack == "gdscript" {
		project := filepath.Join(dir, "project.godot")
		data, err := os.ReadFile(project)
		if err != nil {
			return err
		}
		// The read just succeeded; a failed write leaves the lanes to say so.
		os.WriteFile(project, featuresRe.ReplaceAll(data, []byte("${1}"+e.Version)), 0o644)
	}
	return errors.Join(pinPython(dir, e), pinTools(dir, e), applyBreak(dir, b))
}

// pinPython has uv run the entry's Python, not the newest it finds, and
// binds the entry's tools as uv constraints: a preset variant may run a tool
// with a bare `uv run`, which no {tool:} placeholder reaches, and the
// fixture's dependency groups leave the tools unpinned.
func pinPython(dir string, e Entry) error {
	if e.Stack != "python" {
		return nil
	}
	project := filepath.Join(dir, "pyproject.toml")
	old, err := os.ReadFile(project)
	if err != nil {
		return err
	}
	var reqs []string
	for _, name := range slices.Sorted(maps.Keys(e.Tools)) {
		reqs = append(reqs, strconv.Quote(requirement(name, e.Tools[name])))
	}
	constraints := "\n[tool.uv]\nconstraint-dependencies = [" + strings.Join(reqs, ", ") + "]\n"
	return errors.Join(
		os.WriteFile(filepath.Join(dir, ".python-version"), []byte(e.Version+"\n"), 0o644),
		os.WriteFile(project, append(old, constraints...), 0o644))
}

// requirement is name pinned to spec in PEP 440 form, as verify fills a
// {tool:} placeholder: mypy==2.4.*, or mypy>=2 where spec brings its operator.
func requirement(name, spec string) string {
	if strings.IndexAny(spec, "=<>!~") == 0 {
		return name + spec
	}
	return name + "==" + spec
}

// pinTools appends the entry's tools as [verify.<stack>.tools] to the work
// copy's config.
func pinTools(dir string, e Entry) error {
	if len(e.Tools) == 0 {
		return nil
	}
	cfg := filepath.Join(dir, ".loomux", "config.toml")
	old, _ := os.ReadFile(cfg) // no fixture has one yet
	pins := fmt.Sprintf("\n[verify.%s.tools]\n", e.Stack)
	for _, name := range slices.Sorted(maps.Keys(e.Tools)) {
		pins += fmt.Sprintf("%s = %q\n", name, e.Tools[name])
	}
	return errors.Join(os.MkdirAll(filepath.Dir(cfg), 0o755), os.WriteFile(cfg, append(old, pins...), 0o644))
}

func applyBreak(dir string, b *Break) error {
	if b == nil {
		return nil
	}
	file := filepath.Join(dir, filepath.FromSlash(b.File))
	return errors.Join(os.MkdirAll(filepath.Dir(file), 0o755), os.WriteFile(file, []byte(b.Content), 0o644))
}

// judge holds each lane to ok, and each kind to a lane that ran. In a run of
// broken's break, its kind must end failed and a lane behind it may be
// blocked; only a red lane blocks, and any other red is a line of its own.
func judge(results []LaneResult, kinds []string, broken string) []string {
	var lines []string
	for _, k := range kinds {
		if !slices.ContainsFunc(results, func(r LaneResult) bool { return kindOf(r) == k }) {
			lines = append(lines, k+": no lane ran")
		}
	}
	for _, r := range results {
		want := "ok"
		if kindOf(r) == broken {
			want = "failed"
		}
		if r.State == want || (broken != "" && want == "ok" && r.State == "blocked") {
			continue
		}
		line := fmt.Sprintf("%s: %s, want %s", r.Lane, r.State, want)
		if tail := tail(r.Output); tail != "" {
			line += "\n" + tail
		}
		lines = append(lines, line)
	}
	return lines
}

func kindOf(r LaneResult) string {
	kind, _, _ := strings.Cut(r.Lane, "/")
	return kind
}

func tail(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	return strings.Join(lines[max(0, len(lines)-tailLines):], "\n")
}

// graphLines builds the graph of dir and names each edge it lacks.
func graphLines(dir string, edges [][3]string) []string {
	g, _, err := buildGraph(dir, func(string) {})
	if err != nil {
		return []string{"graph: " + err.Error()}
	}
	var lines []string
	for _, edge := range edges {
		if !slices.ContainsFunc(g.Edges, func(x model.Edge) bool {
			return string(x.Source) == edge[0] && string(x.Relation) == edge[1] && string(x.Target) == edge[2]
		}) {
			lines = append(lines, "graph: no edge "+strings.Join(edge[:], " "))
		}
	}
	return lines
}

// Markdown is the report for a job summary: the verdict as a heading, each
// line an item with its output in a code block.
func (r Report) Markdown() string {
	verdict := "red"
	if r.Green {
		verdict = "green"
	}
	out := fmt.Sprintf("### %s: %s\n", r.Entry, verdict)
	if len(r.Lines) > 0 {
		out += "\n"
	}
	for _, l := range r.Lines {
		head, rest, _ := strings.Cut(l, "\n")
		out += "- " + head + "\n"
		if rest != "" {
			out += "\n  ```\n  " + strings.ReplaceAll(rest, "\n", "\n  ") + "\n  ```\n"
		}
	}
	return out
}

// AddonSource says where an addon comes from: its repository, the folder of
// the addon inside the clone (or the release asset), and the asset to
// download instead of the clone when the repository does not hold the built
// addon. Tag is "v" + the addon's version.
type AddonSource struct {
	Name  string `json:"name"`
	Repo  string `json:"repo"`
	Tag   string `json:"tag"`
	Dir   string `json:"dir"`
	Asset string `json:"asset"`
}

// Description is what CI installs for an entry.
type Description struct {
	Stack   string        `json:"stack"`
	Runtime string        `json:"runtime"`
	Version string        `json:"version"`
	Addons  []AddonSource `json:"addons"`
}

// addonSources are the repositories of the addons; Asset is a format with
// the version. NanoCoverage's repository holds no built extension, only its
// release zip does.
var addonSources = map[string]AddonSource{
	"GUT":          {Repo: "https://github.com/bitwes/Gut", Dir: "addons/gut"},
	"gdUnit4":      {Repo: "https://github.com/godot-gdunit-labs/gdUnit4", Dir: "addons/gdUnit4"},
	"NanoCoverage": {Repo: "https://github.com/IgorBayerl/nano-coverage-godot", Dir: "addons/nano_coverage_godot", Asset: "nano_coverage_godot_v%s.zip"},
}

// Describe names the runtime and addon sources of the entry key.
func Describe(key string) (Description, error) {
	entries, _ := Load() // the embedded list is validated by the tests
	i := slices.IndexFunc(entries, func(e Entry) bool { return e.Key() == key })
	if i < 0 {
		return Description{}, fmt.Errorf("unknown entry %q", key)
	}
	return DescribeEntry(entries[i])
}

// DescribeEntry names the runtime and addon sources of e.
func DescribeEntry(e Entry) (Description, error) {
	d := Description{Stack: e.Stack, Runtime: e.Runtime, Version: e.Version, Addons: []AddonSource{}}
	for _, a := range e.Addons {
		s, ok := addonSources[a.Name]
		if !ok {
			return Description{}, fmt.Errorf("addon %s: no source known", a.Name)
		}
		s.Name, s.Tag = a.Name, "v"+a.Version
		if s.Asset != "" {
			s.Asset = fmt.Sprintf(s.Asset, a.Version)
		}
		d.Addons = append(d.Addons, s)
	}
	return d, nil
}
