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

// An old name beside the new one changes nothing: the new one is the
// declaration, whatever the old one says.
func TestReadAreaDeclarationIgnoresAnOldNameBesideTheNewOne(t *testing.T) {
	dir := t.TempDir()
	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"new\"\n")
	declareIn(t, dir, ".brain.toml", "[area\n")
	got, err := ReadAreaDeclaration(dir)
	if err != nil || got.Scope != "new" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// A policy-only file is ErrNoArea and no longer sends the reader on to an
// old name, which once declared the area in its place.
func TestReadAreaDeclarationAnswersAPolicyOnlyFileWithErrNoArea(t *testing.T) {
	dir := t.TempDir()
	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[verify]\n")
	declareIn(t, dir, ".brain.toml", "[area]\nscope = \"old\"\n")
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

// A directory under the name declares nothing, and neither does one under an
// old name: the hint names files only.
func TestReadAreaDeclarationTakesOnlyRegularFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{filepath.Join(".loomux", "config.toml"), ".brain.toml"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_, err := ReadAreaDeclaration(dir)
	if !errors.Is(err, ErrNoManifest) || strings.Contains(err.Error(), "old manifest") {
		t.Fatalf("got %v; want ErrNoManifest without a hint", err)
	}
}

// Each old name alone is named in the hint, together with the command that
// shows what to carry over; the error stays ErrNoManifest. With both, the
// one ultra-brain asked first is named.
func TestReadAreaDeclarationNamesAnOldManifestBesideTheMissingOne(t *testing.T) {
	for _, tc := range []struct {
		names []string
		named string
	}{
		{[]string{".brain.toml"}, ".brain.toml"},
		{[]string{filepath.Join(".ultra-brain", "config.toml")}, filepath.Join(".ultra-brain", "config.toml")},
		{[]string{".brain.toml", filepath.Join(".ultra-brain", "config.toml")}, filepath.Join(".ultra-brain", "config.toml")},
	} {
		dir := t.TempDir()
		for _, name := range tc.names {
			declareIn(t, dir, name, "[area]\nscope = \"old\"\n")
		}
		_, err := ReadAreaDeclaration(dir)
		want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + "); an old manifest lies there (" +
			tc.named + "): `loomux area check " + dir + "` shows what to carry over"
		if !errors.Is(err, ErrNoManifest) || err.Error() != want {
			t.Fatalf("%v: got %v\nwant %s", tc.names, err, want)
		}
		if !errors.Is(err, ErrOldManifest) || IsUndeclared(err) {
			t.Fatalf("%v: %v must be ErrOldManifest and not undeclared", tc.names, err)
		}
	}
}

// Without an old name beside it the missing file is no ErrOldManifest.
func TestReadAreaDeclarationWithoutAnOldNameIsNoOldManifest(t *testing.T) {
	_, err := ReadAreaDeclaration(t.TempDir())
	if errors.Is(err, ErrOldManifest) || !IsUndeclared(err) {
		t.Fatalf("got %v; want a plain absence", err)
	}
}

// A file that is there and does not read is the reader's error, never an
// absence -- not even beside an old name that would read.
func TestReadAreaDeclarationPassesOnABrokenFile(t *testing.T) {
	dir := t.TempDir()
	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area\n")
	declareIn(t, dir, ".brain.toml", "[area]\nscope = \"old\"\n")
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
		{oldManifestError{fmt.Errorf("x: %w", ErrNoManifest)}, false},
		{errors.New("x: " + ErrNoManifest.Error()), false},
		{errors.New("not valid TOML"), false},
		{nil, false},
	} {
		if got := IsUndeclared(tc.err); got != tc.want {
			t.Fatalf("IsUndeclared(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
