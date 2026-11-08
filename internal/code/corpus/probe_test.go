package corpus

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// written builds the corpus of root and stores its record.
func written(t *testing.T, root string, maxFilesize int64) Record {
	t.Helper()
	rec := buildCorpus(t, root, maxFilesize)
	if err := Write(root, rec); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return rec
}

// bump sets a file's modification time into the future so a probe on any
// filesystem sees a difference.
func bump(t *testing.T, name string) {
	t.Helper()
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(name, future, future); err != nil {
		t.Fatalf("chtimes %s: %v", name, err)
	}
}

func TestProbeUnknownWithoutRecord(t *testing.T) {
	d, err := Probe(t.TempDir(), Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil || d != nil {
		t.Fatalf("Probe without a record = %v, %v; want nil, nil", d, err)
	}
}

func TestProbeCleanWithoutReads(t *testing.T) {
	root := newRepo(t, map[string]string{
		".gitignore": "*.log\n",
		"keep.go":    "package x\n",
		"a.log":      "ignored\n",
		"note.md":    "text\n",
	}, nil, nil)
	written(t, root, DefaultMaxFilesize)

	old := readFile
	var read []string
	readFile = func(name string) ([]byte, error) {
		read = append(read, name)
		return old(name)
	}
	defer func() { readFile = old }()

	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !d.Clean() {
		t.Fatalf("probe found drift in an unchanged tree: %+v", d)
	}
	// The record and its binary mirror: the second is the first's cheap form.
	for _, name := range read {
		clean := filepath.Clean(name)
		if clean != filepath.Clean(RecordPath(root)) && clean != filepath.Clean(mirrorPath(RecordPath(root))) {
			t.Fatalf("the clean probe read %s, want only the record or its mirror", name)
		}
	}
}

func TestProbeDriftOnContent(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	written(t, root, DefaultMaxFilesize)
	name := filepath.Join(root, "keep.go")
	if err := os.WriteFile(name, []byte("package y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bump(t, name)
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !slices.Equal(d.Changed, []string{"keep.go"}) || d.Count() != 1 {
		t.Fatalf("drift = %+v, want one changed file", d)
	}
}

func TestProbeEmptyHashNeverTrusted(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	rec := written(t, root, DefaultMaxFilesize)
	print := rec.Files["keep.go"]
	print.Hash = ""
	rec.Files["keep.go"] = print
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !slices.Equal(d.Changed, []string{"keep.go"}) {
		t.Fatalf("drift = %+v, want the file with the empty hash", d)
	}
}

func TestProbeAddedAndRemoved(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	written(t, root, DefaultMaxFilesize)
	if err := os.WriteFile(filepath.Join(root, "new.md"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "keep.go")); err != nil {
		t.Fatal(err)
	}
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !slices.Equal(d.Added, []string{"new.md"}) || !slices.Equal(d.Removed, []string{"keep.go"}) {
		t.Fatalf("drift = %+v", d)
	}
}

func TestProbeSettingsDrift(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	written(t, root, DefaultMaxFilesize)
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize - 1})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !d.Settings {
		t.Fatalf("drift = %+v, want a settings drift", d)
	}
}

func TestProbeConfigChangeIsDrift(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	rec := written(t, root, DefaultMaxFilesize)
	name := rec.GitPaths.Config
	config, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, append(config, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !d.Config {
		t.Fatalf("drift = %+v, want a config drift", d)
	}
}

func TestProbeIndexStampChangeIsDrift(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, []string{"keep.go"}, nil)
	rec := written(t, root, DefaultMaxFilesize)
	bump(t, rec.GitPaths.Index)
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !d.Index {
		t.Fatalf("drift = %+v, want an index drift", d)
	}
	if d.Count() != 1 {
		t.Fatalf("drift = %+v, want exactly one reason", d)
	}
}

func TestProbeNoGitWithoutStampChange(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	written(t, root, DefaultMaxFilesize)
	t.Setenv("PATH", "")
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !d.Clean() {
		t.Fatalf("probe needs git or found drift: %+v", d)
	}
}

func TestProbeIgnoreSourceChange(t *testing.T) {
	root := newRepo(t, map[string]string{
		".gitignore": "*.log\n",
		"keep.go":    "package x\n",
	}, nil, nil)
	written(t, root, DefaultMaxFilesize)
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.log\nextra.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !slices.Contains(d.Sources, ".gitignore") {
		t.Fatalf("drift = %+v, want .gitignore among the sources", d)
	}
}

func TestProbeNewIgnoreFile(t *testing.T) {
	root := newRepo(t, map[string]string{"sub/drop.md": "x\n"}, nil, nil)
	written(t, root, DefaultMaxFilesize)
	if err := os.WriteFile(filepath.Join(root, "sub", ".gitignore"), []byte("drop.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !slices.Contains(d.Sources, "sub/.gitignore") {
		t.Fatalf("drift = %+v, want the new ignore file", d)
	}
}

func TestProbeDeletedIgnoreFile(t *testing.T) {
	root := newRepo(t, map[string]string{
		".gitignore": "a.log\n",
		"a.log":      "x\n",
		"keep.go":    "package x\n",
	}, nil, nil)
	written(t, root, DefaultMaxFilesize)
	if err := os.Remove(filepath.Join(root, ".gitignore")); err != nil {
		t.Fatal(err)
	}
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !slices.Contains(d.Sources, ".gitignore") {
		t.Fatalf("drift = %+v, want the deleted ignore file", d)
	}
}

func TestProbeOutsideRepository(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, _, err := Load(root)
	if err != nil || rules != nil {
		t.Fatalf("Load outside a repository = %v, %v", rules, err)
	}
	listing, err := Walk(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	files, _, _ := SearchCorpus(listing, DefaultMaxFilesize)
	prints := map[string]Print{}
	for _, f := range files {
		prints[f.Rel] = Print{Size: f.Size, MTime: f.MTime}
	}
	if err := Write(root, NewRecord(nil, prints, DefaultMaxFilesize)); err != nil {
		t.Fatal(err)
	}
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if d.Clean() {
		t.Fatal("the hash placeholder should have been checked")
	}
	if !slices.Equal(d.Changed, []string{"note.md"}) {
		t.Fatalf("drift = %+v", d)
	}
}

func TestProbeUnreadableSearchFileIsNotDrift(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	rec := written(t, root, DefaultMaxFilesize)
	print := rec.Files["keep.go"]
	print.Hash = ""
	rec.Files["keep.go"] = print
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}
	old := readFile
	readFile = func(name string) ([]byte, error) {
		if filepath.Base(name) == "keep.go" {
			return nil, errors.New("boom")
		}
		return old(name)
	}
	defer func() { readFile = old }()
	d, err := Probe(root, Settings{MaxFilesize: DefaultMaxFilesize})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if len(d.Changed) != 0 {
		t.Fatalf("drift = %+v, want the unreadable file left alone", d)
	}
}

func TestRulesFromRecordOutsideRepository(t *testing.T) {
	if RulesFromRecord(Record{}) != nil {
		t.Fatal("RulesFromRecord of a record without a root is not nil")
	}
}

func TestRulesFromRecordWalksLikeLoad(t *testing.T) {
	root := newRepo(t, map[string]string{
		".gitignore":        "*.log\n",
		"keep.go":           "package x\n",
		"nested/.gitignore": "*.tmp\n",
		"nested/keep.txt":   "text\n",
		"nested/drop.tmp":   "tmp\n",
		"a.log":             "ignored\n",
	}, []string{"keep.go"}, nil)
	rec := written(t, root, DefaultMaxFilesize)
	rules := RulesFromRecord(rec)
	if rules == nil {
		t.Fatal("RulesFromRecord returned nil")
	}
	listing, err := Walk(root, rules)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if d := Compare(rec, listing, rules, Settings{MaxFilesize: DefaultMaxFilesize}); !d.Clean() {
		t.Fatalf("Compare of a freshly written record = %+v, want clean", d)
	}
}

func TestCompareAbsoluteGitPaths(t *testing.T) {
	root := newRepo(t, map[string]string{
		"keep.go": "package x\n",
		"a.log":   "ignored\n",
	}, []string{"keep.go"}, nil)
	rec := written(t, root, DefaultMaxFilesize)
	rules := RulesFromRecord(rec)
	listing, err := Walk(root, rules)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	abs := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	rec.GitPaths.Index = abs(".git/index")
	rec.GitPaths.Config = abs(".git/config")
	rec.GitPaths.Exclude = abs(".git/info/exclude")
	if d := Compare(rec, listing, rules, Settings{MaxFilesize: DefaultMaxFilesize}); !d.Clean() {
		t.Fatalf("Compare with absolute git paths = %+v, want clean", d)
	}
}

func TestGitAbsResolvesRecordPaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	rec := Record{GitPaths: GitPaths{Root: root}}
	if got := GitAbs(rec, ""); got != "" {
		t.Fatalf("empty = %q, want empty", got)
	}
	absolute := filepath.Join(t.TempDir(), "index")
	if got := GitAbs(rec, absolute); got != filepath.Clean(absolute) {
		t.Fatalf("absolute = %q, want %q", got, filepath.Clean(absolute))
	}
	if got, want := GitAbs(rec, filepath.Join(".git", "index")), filepath.Join(root, ".git", "index"); got != want {
		t.Fatalf("relative = %q, want %q", got, want)
	}
}
