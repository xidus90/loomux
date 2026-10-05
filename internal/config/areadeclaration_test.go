package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

// declareIn writes body under name below dir.
func declareIn(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadAreaDeclarationReadsTheLoomuxName(t *testing.T) {
	dir := t.TempDir()
	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"project/a\"\n[privacy]\nmode = \"local_only\"\n")
	got, err := ReadAreaDeclaration(dir)
	if err != nil || got.Scope != "project/a" || got.PrivacyMode != "local_only" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if got.Path != filepath.Join(dir, ".loomux", "config.toml") {
		t.Fatalf("path %q", got.Path)
	}
}

// A policy-only file is ErrNoArea, not ErrNoManifest.
func TestReadAreaDeclarationAnswersAPolicyOnlyFileWithErrNoArea(t *testing.T) {
	dir := t.TempDir()
	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[verify]\n")
	got, err := ReadAreaDeclaration(dir)
	if !errors.Is(err, ErrNoArea) || errors.Is(err, ErrNoManifest) || got != nil {
		t.Fatalf("got %+v, %v; want ErrNoArea alone", got, err)
	}
}

func TestReadAreaDeclarationWithoutAFileIsErrNoManifestWithoutAHint(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadAreaDeclaration(dir)
	want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + ")"
	if !errors.Is(err, ErrNoManifest) || err.Error() != want {
		t.Fatalf("got %v; want %q", err, want)
	}
}

// A directory under the name declares nothing.
func TestReadAreaDeclarationTakesOnlyRegularFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := ReadAreaDeclaration(dir)
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("got %v; want ErrNoManifest", err)
	}
}

// A file that is there and does not read is the reader's error, never an
// absence.
func TestReadAreaDeclarationPassesOnABrokenFile(t *testing.T) {
	dir := t.TempDir()
	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area\n")
	_, err := ReadAreaDeclaration(dir)
	if err == nil || IsUndeclared(err) || !strings.Contains(err.Error(), "not valid TOML") {
		t.Fatalf("got %v; want the parse error", err)
	}
}

// A file that is there and cannot be opened is an error naming it, never an
// absence: taken for one, a locked `local_only` declaration would open its
// area to the defaults.
func TestReadAreaDeclarationPassesOnALockedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".loomux", "config.toml")
	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"local_only\"\n")
	testlock.Lock(t, path)
	_, err := ReadAreaDeclaration(dir)
	if err == nil || IsUndeclared(err) || !strings.HasPrefix(err.Error(), path+": cannot be read: ") {
		t.Fatalf("got %v; want the read error naming %s", err, path)
	}
}

// The post-merge hook asks this reader for consent, the default branch
// included.
func TestReadAreaDeclarationCarriesTheMergeConsent(t *testing.T) {
	for _, row := range []struct {
		maintenance string
		onMerge     bool
		branch      string
	}{
		{"[maintenance]\non_merge = true\nbranch = \"trunk\"\n", true, "trunk"},
		{"[maintenance]\non_merge = true\n", true, DefaultMergeBranch},
		{"", false, DefaultMergeBranch},
	} {
		dir := t.TempDir()
		declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"a\"\n"+row.maintenance)
		m, err := ReadAreaDeclaration(dir)
		if err != nil || m.OnMerge != row.onMerge || m.MergeBranch != row.branch {
			t.Fatalf("%q: got %+v, %v", row.maintenance, m, err)
		}
	}
}

func TestIsUndeclaredTakesExactlyTheTwoAbsences(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{ErrNoManifest, true},
		{ErrNoArea, true},
		{fmt.Errorf("x: %w", ErrNoManifest), true},
		{errors.New("x: " + ErrNoManifest.Error()), false},
		{errors.New("not valid TOML"), false},
		{nil, false},
	} {
		if got := IsUndeclared(tc.err); got != tc.want {
			t.Fatalf("IsUndeclared(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
