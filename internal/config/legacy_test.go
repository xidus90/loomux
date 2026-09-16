package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

func TestTheLegacyWindowsDirIsLocalAppDataBrain(t *testing.T) {
	got := legacyBrainDirUntilStage3For("windows", stub(map[string]string{"LOCALAPPDATA": `D:\state`}), `C:\Users\x`)
	if want := filepath.Join(`D:\state`, "brain"); got != want {
		t.Errorf("legacyBrainDirUntilStage3For = %q, want %q", got, want)
	}
}

func TestTheLegacyWindowsDirFallsBackToTheHome(t *testing.T) {
	got := legacyBrainDirUntilStage3For("windows", stub(map[string]string{}), `C:\Users\x`)
	if want := filepath.Join(`C:\Users\x`, "AppData", "Local", "brain"); got != want {
		t.Errorf("legacyBrainDirUntilStage3For = %q, want %q", got, want)
	}
}

func TestTheLegacyDirElsewhereFollowsXdgStateHome(t *testing.T) {
	got := legacyBrainDirUntilStage3For("linux", stub(map[string]string{"XDG_STATE_HOME": "/s"}), "/home/x")
	if want := filepath.Join("/s", "brain"); got != want {
		t.Errorf("legacyBrainDirUntilStage3For = %q, want %q", got, want)
	}
}

func TestTheLegacyDirElsewhereWithoutXdgIsLocalStateBrain(t *testing.T) {
	got := legacyBrainDirUntilStage3For("linux", stub(map[string]string{}), "/home/x")
	if want := filepath.Join("/home/x", ".local", "state", "brain"); got != want {
		t.Errorf("legacyBrainDirUntilStage3For = %q, want %q", got, want)
	}
}

func TestTheLegacyEnvironmentVariableBeatsThePlatform(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
	if got := LegacyBrainDirUntilStage3(); got != dir {
		t.Errorf("LegacyBrainDirUntilStage3() = %q, want %q", got, dir)
	}
}

func TestTheLegacyDirIgnoresLoomuxStateDir(t *testing.T) {
	// The two directories are two answers: a registry in loomux's and the
	// artefacts in ultra-brain's. Following LOOMUX_STATE_DIR here would read
	// the artefacts of read-only areas from a directory nothing writes.
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", "")
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	t.Setenv("LOCALAPPDATA", `D:\state`)
	t.Setenv("XDG_STATE_HOME", "/state")
	home, _ := os.UserHomeDir()
	if want := legacyBrainDirUntilStage3For(runtime.GOOS, os.Getenv, home); LegacyBrainDirUntilStage3() != want {
		t.Errorf("LegacyBrainDirUntilStage3() = %q, want %q", LegacyBrainDirUntilStage3(), want)
	}
}

func TestTheLegacyDirPassesTheRealHomeOn(t *testing.T) {
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", "")
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("XDG_STATE_HOME", "")
	home, _ := os.UserHomeDir()
	got := LegacyBrainDirUntilStage3()
	if want := legacyBrainDirUntilStage3For(runtime.GOOS, os.Getenv, home); got != want {
		t.Errorf("LegacyBrainDirUntilStage3() = %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, home) {
		t.Errorf("LegacyBrainDirUntilStage3() = %q does not sit under the home directory %q", got, home)
	}
}

// legacyWrite writes content to name below dir and creates the directories
// the name needs.
func legacyWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTheThreeNamesAreTriedInOrder(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"loomux\"\n")
	legacyWrite(t, dir, filepath.Join(".ultra-brain", "config.toml"), "[area]\nscope = \"ultra-brain\"\n")
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
	for _, step := range []struct {
		want   string
		remove string
	}{
		{want: "loomux", remove: ".loomux"},
		{want: "ultra-brain", remove: ".ultra-brain"},
		{want: "brain"},
	} {
		m, err := ReadAreaManifestUntilStage4(dir)
		if err != nil {
			t.Fatal(err)
		}
		if m.Scope != step.want {
			t.Errorf("Scope = %q, want %q", m.Scope, step.want)
		}
		if step.remove != "" {
			if err := os.RemoveAll(filepath.Join(dir, step.remove)); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestADirectoryUnderANameIsSkipped(t *testing.T) {
	// `manifest_path` asks `is_file()`, so a directory of that name is not
	// the manifest and the next name is asked.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".ultra-brain", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
	m, err := ReadAreaManifestUntilStage4(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Scope != "brain" {
		t.Errorf("Scope = %q, want %q", m.Scope, "brain")
	}
}

func TestALegacyManifestCarriesItsPrivacy(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, ".brain.toml",
		"[area]\nscope = \"project/closed\"\n\n[privacy]\nmode = \"local_only\"\nnever = [\"secret/**\", \"*.pem\"]\n")
	m, err := ReadAreaManifestUntilStage4(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.PrivacyMode != "local_only" {
		t.Errorf("PrivacyMode = %q, want local_only", m.PrivacyMode)
	}
	if strings.Join(m.NeverGlobs, ",") != "secret/**,*.pem" {
		t.Errorf("NeverGlobs = %v, want [secret/** *.pem]", m.NeverGlobs)
	}
}

func TestALegacyManifestWithoutAScopeIsRefused(t *testing.T) {
	// Wording and order of `read_manifest` (src/brain/manifest.py:21-24),
	// measured: a missing table, a missing key and an empty string all give
	// this one message, and it comes before the complaint about the mode.
	for name, body := range map[string]string{
		"no area table":  "[privacy]\nmode = \"local_only\"\n",
		"no scope key":   "[area]\nname = \"x\"\n",
		"empty scope":    "[area]\nscope = \"\"\n",
		"and a bad mode": "[privacy]\nmode = \"bogus\"\n",
	} {
		dir := t.TempDir()
		legacyWrite(t, dir, ".brain.toml", body)
		_, err := ReadAreaManifestUntilStage4(dir)
		want := filepath.Join(dir, ".brain.toml") + ": [area] scope is required and must be a non-empty string"
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", name, err, want)
		}
	}
}

func TestAPolicyOnlyLoomuxManifestDeclaresNothingForBrain(t *testing.T) {
	// A .loomux/config.toml without an [area] table is policy only and
	// declares nothing, as the write barrier reads it (stage 1a, R7a): the
	// next name is asked, and the privacy of the policy file is not the
	// area's.
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), "[privacy]\nmode = \"local_only\"\n")
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
	m, err := ReadAreaManifestUntilStage4(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Scope != "brain" || m.PrivacyMode != "manual_cloud" {
		t.Errorf("Scope, PrivacyMode = %q, %q; want brain, manual_cloud", m.Scope, m.PrivacyMode)
	}
}

func TestAPolicyOnlyLoomuxManifestAloneIsErrNoManifest(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), "[layout]\nwiki = \"docs/wiki\"\n")
	_, err := ReadAreaManifestUntilStage4(dir)
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("err = %v, want ErrNoManifest", err)
	}
	want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + ", " +
		filepath.Join(".ultra-brain", "config.toml") + ", .brain.toml)"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestAnAreaTableWithoutAScopeInTheLoomuxManifestIsRefused(t *testing.T) {
	// An [area] table is a declaration, so an empty one is a scope error for
	// the loomux file and does not hand the decision to the .brain.toml
	// beside it.
	for name, body := range map[string]string{
		"empty table": "[area]\n",
		"empty scope": "[area]\nscope = \"\"\n",
	} {
		dir := t.TempDir()
		legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), body)
		legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
		_, err := ReadAreaManifestUntilStage4(dir)
		want := filepath.Join(dir, ".loomux", "config.toml") + ": [area] scope is required and must be a non-empty string"
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", name, err, want)
		}
	}
}

func TestABlankScopeIsAScope(t *testing.T) {
	// `not scope` is false for " ", measured: read_manifest accepts it.
	dir := t.TempDir()
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \" \"\n")
	m, err := ReadAreaManifestUntilStage4(dir)
	if err != nil || m.Scope != " " {
		t.Fatalf("ReadAreaManifestUntilStage4 = %+v, %v; want scope \" \"", m, err)
	}
}

func TestBrokenLegacyTomlNamesTheFile(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".ultra-brain", "config.toml"), "[area\nscope = \"k\"\n")
	_, err := ReadAreaManifestUntilStage4(dir)
	prefix := filepath.Join(dir, ".ultra-brain", "config.toml") + ": not valid TOML: "
	if err == nil || !strings.HasPrefix(err.Error(), prefix) {
		t.Fatalf("err = %v, want the prefix %q", err, prefix)
	}
}

func TestNoNameAtAllIsErrNoManifestNamingAllThree(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadAreaManifestUntilStage4(dir)
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("err = %v, want ErrNoManifest", err)
	}
	want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + ", " +
		filepath.Join(".ultra-brain", "config.toml") + ", .brain.toml)"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestALockedFirstNameIsAnErrorEvenBesideAnOpenLegacyName(t *testing.T) {
	// The first name that is a regular file decides. A locked
	// .loomux/config.toml must not hand the decision to a .brain.toml next to
	// it, which might open an area the locked file closes.
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"local_only\"\n")
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"k\"\n")
	testlock.Lock(t, filepath.Join(dir, ".loomux", "config.toml"))
	_, err := ReadAreaManifestUntilStage4(dir)
	if err == nil || errors.Is(err, ErrNoManifest) {
		t.Fatalf("err = %v, want a read error", err)
	}
	if !strings.HasPrefix(err.Error(), filepath.Join(dir, ".loomux", "config.toml")+": cannot be read: ") {
		t.Errorf("err = %q does not name the locked file", err)
	}
}

func TestReadManifestStillKnowsOnlyTheLoomuxName(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
	legacyWrite(t, dir, filepath.Join(".ultra-brain", "config.toml"), "[area]\nscope = \"ultra-brain\"\n")
	_, err := ReadManifest(dir)
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("ReadManifest err = %v, want ErrNoManifest", err)
	}
	if want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + ")"; err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestReadManifestStillAcceptsAManifestWithoutAScope(t *testing.T) {
	// The scope rule belongs to the brain reader alone; the check chain read
	// such a manifest before this stage and still does.
	dir := t.TempDir()
	write(t, manifestIn(t, dir), "[privacy]\nmode = \"local_only\"\n")
	m, err := ReadManifest(dir)
	if err != nil || m.Scope != "" || m.PrivacyMode != "local_only" {
		t.Fatalf("ReadManifest = %+v, %v", m, err)
	}
}
