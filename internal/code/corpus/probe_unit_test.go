package corpus

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRulesFromRecordRebuildsEveryKind(t *testing.T) {
	rec := Record{
		GitPaths: GitPaths{Root: t.TempDir()},
		Tracked:  []string{"sub/deep/a.go"},
		Sources: []Source{
			{Rel: ".gitignore", Kind: "gitignore", Body: []byte("*.log\n")},
			{Rel: "sub/.gitignore", Kind: "gitignore", Body: []byte("*.tmp\n")},
			{Rel: ".git/info/exclude", Kind: "exclude", Body: []byte("!keep\n")},
			{Rel: "global-ignore", Kind: "global", Body: []byte("*.bak\n")},
		},
	}
	r := rulesFromRecord(rec)
	if !r.tracked["sub/deep/a.go"] || !r.trackedDirs["sub"] || !r.trackedDirs["sub/deep"] {
		t.Fatalf("tracked state = %v %v", r.tracked, r.trackedDirs)
	}
	if len(r.dirs[""]) != 1 || len(r.dirs["sub"]) != 1 {
		t.Fatalf("dirs = %v", r.dirs)
	}
	if len(r.exclude) != 1 || len(r.global) != 1 {
		t.Fatalf("exclude = %v, global = %v", r.exclude, r.global)
	}
	if !r.loaded[""] || !r.loaded["sub"] {
		t.Fatalf("loaded = %v", r.loaded)
	}
}

func TestSourceAbs(t *testing.T) {
	rec := Record{GitPaths: GitPaths{Root: filepath.Join(t.TempDir(), "repo")}}
	if got, want := sourceAbs(rec, "sub/.gitignore"), filepath.Join(rec.GitPaths.Root, "sub", ".gitignore"); got != want {
		t.Fatalf("sourceAbs = %q, want %q", got, want)
	}
	abs := filepath.Join(t.TempDir(), "outside", "ignore")
	if got := sourceAbs(rec, abs); got != filepath.Clean(abs) {
		t.Fatalf("sourceAbs(absolute) = %q, want %q", got, abs)
	}
}

func TestRelForAbs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	rec := Record{GitPaths: GitPaths{Root: root}}
	if got := relForAbs(rec, filepath.Join(root, "sub", "ignore")); got != "sub/ignore" {
		t.Fatalf("relForAbs inside = %q", got)
	}
	outside := filepath.Join(t.TempDir(), "ignore")
	if got := relForAbs(rec, outside); got != filepath.ToSlash(outside) {
		t.Fatalf("relForAbs outside = %q", got)
	}
	if got := relForAbs(Record{}, outside); got != filepath.ToSlash(outside) {
		t.Fatalf("relForAbs without root = %q", got)
	}
}

func TestDedupeSorted(t *testing.T) {
	if got := dedupeSorted(nil); got != nil {
		t.Fatalf("dedupeSorted(nil) = %v", got)
	}
	if got := dedupeSorted([]string{"b", "a", "b", "a"}); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("dedupeSorted = %v", got)
	}
}

func TestChangedSourcesFindsNewConfiguredFiles(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "extra-ignore")
	if err := os.WriteFile(inside, []byte("*.x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "global-ignore")
	if err := os.WriteFile(outside, []byte("*.y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := Record{GitPaths: GitPaths{Root: root, Exclude: inside, Global: outside}}
	got := changedSources(rec)
	if !slices.Contains(got, "extra-ignore") {
		t.Fatalf("changedSources = %v, want the new exclude file", got)
	}
	if !slices.Contains(got, filepath.ToSlash(outside)) {
		t.Fatalf("changedSources = %v, want the new global file", got)
	}
}

func TestChangedSourcesReadError(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, "ignore")
	if err := os.WriteFile(name, []byte("*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := Record{
		GitPaths: GitPaths{Root: root},
		Sources:  []Source{{Rel: "ignore"}},
	}
	old := readFile
	readFile = func(string) ([]byte, error) { return nil, errors.New("boom") }
	defer func() { readFile = old }()
	if got := changedSources(rec); !slices.Equal(got, []string{"ignore"}) {
		t.Fatalf("changedSources = %v, want the unreadable source", got)
	}
}

func TestHashFileReadError(t *testing.T) {
	old := readFile
	readFile = func(string) ([]byte, error) { return nil, errors.New("boom") }
	defer func() { readFile = old }()
	if _, err := hashFile("x"); err == nil {
		t.Fatal("hashFile accepted an unreadable file")
	}
}
