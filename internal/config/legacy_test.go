package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
