// Package compat holds the list of runtime, framework and tool versions
// loomux claims to support, and loads it.
package compat

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/verify"
)

//go:embed compat.toml
var listText []byte

// Addon is a framework from verify's frameworks.toml at a pinned version.
type Addon struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Entry is one supported combination: a runtime version, the addons and tools
// run with it, and the fixture project and kinds of check it is run with.
type Entry struct {
	Stack   string            `json:"stack"`
	Runtime string            `json:"runtime"`
	Version string            `json:"version"`
	Fixture string            `json:"fixture"`
	Kinds   []string          `json:"kinds"`
	Addons  []Addon           `json:"addons,omitempty"`
	Tools   map[string]string `json:"tools,omitempty"`
	// Line is the 1-based line of the entry's header in compat.toml.
	Line int `toml:"-" json:"-"`
}

// LoadEntry reads one entry from a JSON file, as a scout probe is written,
// and holds it to the rules of the list.
func LoadEntry(path string) (Entry, error) {
	var e Entry
	err := readJSON(path, &e, func() error { return check(e, references(), true) })
	return e, err
}

// LoadProbes reads a JSON list of entries, each held to the same rules.
func LoadProbes(path string) ([]Entry, error) {
	var es []Entry
	err := readJSON(path, &es, func() error {
		for _, e := range es {
			if err := check(e, references(), true); err != nil {
				return fmt.Errorf("%s: %w", e.Key(), err)
			}
		}
		return nil
	})
	return es, err
}

// readJSON decodes the file strictly into v and then validates it.
func readJSON(path string, v any, valid func() error) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := valid(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// Key identifies an entry: <stack>/<runtime>-<version>/<fixture>.
func (e Entry) Key() string {
	return fmt.Sprintf("%s/%s-%s/%s", e.Stack, e.Runtime, e.Version, e.Fixture)
}

// Load reads the embedded list in file order, once.
var Load = sync.OnceValues(func() ([]Entry, error) { return parse(listText, "compat.toml") })

var runtimeOf = map[string]string{"gdscript": "godot", "python": "python", "go": "go"}

var (
	versionRe = regexp.MustCompile(`^\d+\.\d+$`)
	headerRe  = regexp.MustCompile(`^\s*\[\[\s*entry\s*\]\]`)
)

// parse refuses anything that could pass for supported but is not.
func parse(data []byte, path string) ([]Entry, error) {
	if _, err := decodeWatches(data, path); err != nil {
		return nil, err
	}
	return decode(data, path, true)
}

// decode is parse; without strict it does not require the fixtures to exist,
// which an older list read from git may name after they were removed.
func decode(data []byte, path string, strict bool) ([]Entry, error) {
	var doc struct {
		Entry []Entry          `toml:"entry"`
		Watch []map[string]any `toml:"watch"` // checked by decodeWatches
	}
	md, err := toml.Decode(string(data), &doc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	if un := md.Undecoded(); len(un) > 0 {
		key := un[0][len(un[0])-1]
		return nil, fmt.Errorf("%s: unknown key %q", place(path, keyLine(lines, key)), key)
	}
	var headers []int
	for i, l := range lines {
		if headerRe.MatchString(l) {
			headers = append(headers, i+1)
		}
	}
	refs := references()
	seen := map[string]bool{}
	for i := range doc.Entry {
		e := &doc.Entry[i]
		e.Line = headers[i]
		at := place(path, e.Line)
		if err := check(*e, refs, strict); err != nil {
			return nil, fmt.Errorf("%s: %w", at, err)
		}
		if seen[e.Key()] {
			return nil, fmt.Errorf("%s: duplicate entry %s", at, e.Key())
		}
		seen[e.Key()] = true
	}
	return doc.Entry, nil
}

// refs is what an entry's addons and tools are checked against.
type refs struct {
	frameworks []verify.Framework
	tools      map[string][]string
}

// references loads both from verify; its embedded data is validated by
// verify's own tests, so a failure here is a broken build, not an input error.
//
//coverage:exempt unreachable error branches of embedded data
func references() refs {
	frameworks, err := verify.LoadFrameworks()
	if err != nil {
		panic(err)
	}
	r := refs{frameworks: frameworks, tools: map[string][]string{}}
	for stack := range runtimeOf {
		if r.tools[stack], err = verify.ToolsOf(stack); err != nil {
			panic(err)
		}
	}
	return r
}

func check(e Entry, refs refs, strict bool) error {
	if e.Fixture == "" {
		return fmt.Errorf("entry has no fixture")
	}
	want, ok := runtimeOf[e.Stack]
	if !ok {
		return fmt.Errorf("stack %q; stacks are gdscript, python, go", e.Stack)
	}
	if e.Runtime != want {
		return fmt.Errorf("runtime %q; %s runs on %s", e.Runtime, e.Stack, want)
	}
	if !versionRe.MatchString(e.Version) {
		return fmt.Errorf("version %q is not MAJOR.MINOR", e.Version)
	}
	if len(e.Kinds) == 0 {
		return fmt.Errorf("entry has no kinds")
	}
	for _, k := range e.Kinds {
		if !slices.Contains(verify.Kinds(), k) {
			return fmt.Errorf("kind %q; kinds are %s", k, strings.Join(verify.Kinds(), ", "))
		}
	}
	for _, a := range e.Addons {
		if !slices.ContainsFunc(refs.frameworks, func(f verify.Framework) bool { return f.Stack == e.Stack && f.Name == a.Name }) {
			return fmt.Errorf("addon %q is no framework of %s", a.Name, e.Stack)
		}
	}
	tools := refs.tools[e.Stack]
	for name := range e.Tools {
		if !slices.Contains(tools, name) {
			return fmt.Errorf("tool %q is not one %s brings (%s)", name, e.Stack, strings.Join(tools, ", "))
		}
	}
	if _, err := fs.Stat(fixtures, "testdata/"+e.Fixture); strict && err != nil {
		return fmt.Errorf("fixture %q has no folder under testdata/", e.Fixture)
	}
	return nil
}

// place names the file and, when a line is known, the line.
func place(path string, line int) string {
	if line == 0 {
		return path
	}
	return fmt.Sprintf("%s:%d", path, line)
}

// keyLine is the 1-based line that assigns key, or 0 when none does (the key
// sits inside an inline table).
func keyLine(lines []string, key string) int {
	re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `\s*=`)
	for i, l := range lines {
		if re.MatchString(l) {
			return i + 1
		}
	}
	return 0
}

// A nested go.mod makes Go refuse the folder, so the go-module fixture keeps its
// module file as go.mod.txt; whoever copies the fixture out renames it.
//
//go:embed all:testdata
var fixtures embed.FS

// Fixtures is the tree of fixture projects, one folder per Entry.Fixture.
func Fixtures() fs.FS {
	sub, _ := fs.Sub(fixtures, "testdata") // the path is valid and constant
	return sub
}
