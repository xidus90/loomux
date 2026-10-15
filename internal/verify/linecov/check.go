package linecov

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/verify/gocover"
)

// Options of one run. Reports are files, directories or globs relative to
// the working directory; none means the profiles' default locations.
type Options struct {
	Root    string
	Reports []string
}

// Result is what the run found and what it could not check.
type Result struct {
	Findings []Finding
	Notes    []string
}

// readFile is a seam for the one read error a test cannot make on disk.
var readFile = os.ReadFile

// statFile is the same kind of seam for a source that vanishes between read and stat.
var statFile = os.Stat

// readDir is the seam for a report directory that stats but cannot be listed.
var readDir = os.ReadDir

// Check reads the reports, resolves their paths, activates the profiles and
// applies the rules. An error means the run could not start at all.
func Check(o Options) (Result, error) {
	ps, err := LoadProfiles()
	if err != nil {
		return Result{}, err
	}
	tree, err := ScanTree(o.Root)
	if err != nil {
		return Result{}, err
	}
	files, looked, finds, err := reportFiles(o.Reports, tree.Root, ps)
	if err != nil {
		return Result{}, err
	}
	if len(files) == 0 {
		if len(finds) == 0 {
			finds = []Finding{{"", 0, "no coverage report found; looked in " + strings.Join(looked, ", ")}}
		}
		return Result{Findings: finds}, nil
	}
	var notes, readRel []string
	var module string
	if gomod, err := os.ReadFile(filepath.Join(tree.Root, "go.mod")); err == nil {
		if module, err = gocover.ModulePath(gomod); err != nil {
			notes = append(notes, err.Error()+"; Go profile paths cannot be resolved")
		}
	}
	rep := Report{}
	var newest time.Time
	outside, nested := 0, 0
	for _, f := range files {
		shown := display(tree.Root, f)
		content, err := readFile(f)
		if err != nil {
			finds = append(finds, Finding{shown, 0, "unreadable: " + err.Error()})
			continue
		}
		if st, err := os.Stat(f); err == nil && st.ModTime().After(newest) {
			newest = st.ModTime()
		}
		format := Detect(content)
		if format == nil {
			finds = append(finds, Finding{shown, 0, "unknown coverage format"})
			continue
		}
		readRel = append(readRel, shown)
		recs, problems := format.Read(content, ReadContext{Module: module})
		for _, p := range problems {
			finds = append(finds, Finding{shown, 0, p})
		}
		for _, rec := range recs {
			switch res := tree.Resolve(rec); res.Outcome {
			case Resolved:
				rep.File(res.Rel)
				for n, h := range rec.Lines {
					rep.Add(res.Rel, n, h)
				}
			case Outside:
				outside++
			case Nested:
				nested++
			case Missing:
				finds = append(finds, Finding{res.Rel, 0, "is in the report but not on disk"})
			case Ambiguous:
				finds = append(finds, Finding{res.Rel, 0, "ambiguous source: " + strings.Join(res.Candidates, ", ")})
			}
		}
	}
	act, actFinds, actNotes := Activate(tree, readRel, ps)
	finds, notes = append(finds, actFinds...), append(notes, actNotes...)
	if outside > 0 {
		notes = append(notes, fmt.Sprintf("%d records outside the root were skipped", outside))
	}
	if nested > 0 {
		notes = append(notes, fmt.Sprintf("%d records in other checkouts were skipped", nested))
	}
	for _, a := range act {
		if !scopedFor(act, a) {
			notes = append(notes, a.Profile.Name+": no scope in its settings; files absent from the report are not checked")
		}
	}
	if rep.Lines() == 0 {
		finds = append(finds, Finding{"", 0, "the reports hold no line records"})
		return Result{Findings: SortFindings(finds), Notes: notes}, nil
	}
	finds = append(finds, Gate(Input{
		Report: rep, Tree: tree, Active: act, Natives: Natives(ps, act), Newest: newest,
		Read: func(rel string) ([]byte, error) {
			return os.ReadFile(filepath.Join(tree.Root, filepath.FromSlash(rel)))
		},
		ModTime: func(rel string) (time.Time, error) {
			st, err := statFile(filepath.Join(tree.Root, filepath.FromSlash(rel)))
			if err != nil {
				return time.Time{}, err
			}
			return st.ModTime(), nil
		},
	})...)
	return Result{Findings: SortFindings(finds), Notes: notes}, nil
}

// reportFiles expands the given reports, or the default locations of every
// profile; looked lists what was tried, for the finding that nothing was
// found, and finds the report directories that could not be listed.
func reportFiles(given []string, root string, ps []Profile) (files, looked []string, finds []Finding, err error) {
	if len(given) == 0 {
		for _, p := range ps {
			for _, r := range p.Reports {
				if slices.Contains(looked, r) {
					continue
				}
				looked = append(looked, r)
				if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(r))); err == nil && st.Mode().IsRegular() {
					files = append(files, filepath.Join(root, filepath.FromSlash(r)))
				}
			}
		}
		return files, looked, nil, nil
	}
	for _, g := range given {
		looked = append(looked, g)
		if st, err := os.Stat(g); err == nil && st.IsDir() {
			entries, err := readDir(g)
			if err != nil {
				finds = append(finds, Finding{display(root, g), 0, "unreadable: " + err.Error()})
			}
			for _, e := range entries {
				if e.Type().IsRegular() {
					files = append(files, filepath.Join(g, e.Name()))
				}
			}
			continue
		}
		matches, err := filepath.Glob(g)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("--report %q: %w", g, err)
		}
		files = append(files, matches...)
	}
	return files, looked, finds, nil
}

// scopedFor says whether an active profile with a scope, a itself or
// another, measures one of a's extensions. When another one does, the
// project uses that tool, and a's missing scope is no gap.
func scopedFor(act []Active, a Active) bool {
	return slices.ContainsFunc(act, func(b Active) bool {
		return b.Scoped() && slices.ContainsFunc(b.Profile.Extensions, func(e string) bool { return slices.Contains(a.Profile.Extensions, e) })
	})
}

// display names a report relative to the root when it lies below it.
func display(root, f string) string {
	abs, _ := filepath.Abs(f)
	if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(f)
}
