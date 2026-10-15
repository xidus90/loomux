package linecov

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Every format and every profile brings a fixture, so an entry cannot be
// added untested. A tool fixture is a small project: its settings file, a
// source carrying each of the profile's markers with a reason, and a report
// at a default location or as report.* beside it. want.txt lists the
// findings the fixture must produce, one per line; absent means none.
//
// A fixture proves that settings and markers take effect, not only that they
// exist. Its report lists the lines the markers exempt with zero hits on
// purpose (a real tool drops them): were a marker not read, they would be
// "not covered". A file only one exclude setting removes is reported with an
// uncovered line, and an in-scope file the report lacks must be found absent.
// Each marker without a reason is a finding in want.txt.
func TestEveryFormatHasAFixtureItReadsCleanly(t *testing.T) {
	for _, f := range formats() {
		matches, _ := filepath.Glob(filepath.Join("testdata", "formats", f.Name(), "report.*"))
		if len(matches) != 1 {
			t.Errorf("%s: want one testdata/formats/%s/report.*", f.Name(), f.Name())
			continue
		}
		content, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatal(err)
		}
		if got := Detect(content); got == nil || got.Name() != f.Name() {
			t.Errorf("%s: detected as %v", f.Name(), got)
		}
		recs, problems := f.Read(content, ReadContext{Module: "example.com/m"})
		if len(recs) == 0 || len(problems) != 0 {
			t.Errorf("%s: %d records, problems %q", f.Name(), len(recs), problems)
		}
	}
}

func TestEveryProfileHasAFixtureProjectThatPasses(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		src := filepath.Join("testdata", "tools", p.Name)
		if _, err := os.Stat(src); err != nil {
			t.Errorf("%s: no fixture project at %s", p.Name, src)
			continue
		}
		// A copy, written in one go: a checkout writes the fixture's files in
		// any order and at any pace, and the freshness rule would see that.
		dir := t.TempDir()
		if err := os.CopyFS(dir, os.DirFS(src)); err != nil {
			t.Fatal(err)
		}
		reports, _ := filepath.Glob(filepath.Join(dir, "report.*"))
		res, err := Check(Options{Root: dir, Reports: reports})
		if err != nil {
			t.Fatalf("%s: %v", p.Name, err)
		}
		var got, want []string
		for _, f := range res.Findings {
			got = append(got, f.String())
		}
		if w, _ := os.ReadFile(filepath.Join(dir, "want.txt")); strings.TrimSpace(string(w)) != "" {
			want = strings.Split(strings.TrimSpace(string(w)), "\n")
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: findings %q, want %q", p.Name, got, want)
		}
		tree, err := ScanTree(dir)
		if err != nil {
			t.Fatal(err)
		}
		act, _, _ := Activate(tree, nil, []Profile{p})
		if len(p.Settings) > 0 && (len(act) != 1 || act[0].Settings == "") {
			t.Errorf("%s: the fixture's settings file was not read", p.Name)
		}
		for _, spec := range p.Markers {
			m, _ := spec.compile("")
			if !fixtureHas(t, tree, p, m) {
				t.Errorf("%s: no fixture source carries %s", p.Name, spec.Label)
			}
		}
		// A configured marker is compiled from the value the fixture's own
		// settings give its key, and a source must carry it.
		vals, _, _, _ := readSettings(tree.Root, p)
		for _, spec := range p.MarkerKeys {
			if len(vals[spec.Key]) == 0 {
				t.Errorf("%s: the fixture's settings do not set %s", p.Name, spec.Key)
			}
			for _, v := range vals[spec.Key] {
				m, err := spec.compile(v)
				if err != nil {
					t.Fatal(err)
				}
				if !fixtureHas(t, tree, p, m) {
					t.Errorf("%s: no fixture source carries %s", p.Name, m.Label)
				}
			}
		}
	}
}

func fixtureHas(t *testing.T, tree *Tree, p Profile, m Marker) bool {
	for _, rel := range tree.Files {
		if !slices.Contains(p.Extensions, filepath.Ext(rel)) {
			continue
		}
		src, err := os.ReadFile(filepath.Join(tree.Root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		// Markers match line by line, as the gate applies them.
		for _, line := range strings.Split(string(src), "\n") {
			if m.Re.MatchString(line) {
				return true
			}
		}
	}
	return false
}
