package godotsuite

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/verify/linecov"
)

var past = time.Date(2024, 3, 4, 5, 6, 8, 0, time.UTC) // even second: FAT keeps two

func gdProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := project(t, files)
	for rel := range files {
		if err := os.Chtimes(filepath.Join(root, rel), past, past); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func mtimeOf(t *testing.T, p string) time.Time {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.ModTime().Round(time.Second)
}

func TestSettleAcceptsARewriteWithTheSameContentAndRestoresItsMtime(t *testing.T) {
	root := gdProject(t, map[string]string{"core/a.gd": "extends Node\n", "readme.md": "x"})
	snap, err := Take(root)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "core", "a.gd")
	if err := os.WriteFile(p, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("extends Node\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if mtimeOf(t, p).Equal(past) {
		t.Fatal("precondition: the rewrite must have moved the mtime")
	}
	if f := snap.Settle(root); len(f) != 0 {
		t.Fatalf("findings %q", f)
	}
	if got := mtimeOf(t, p); !got.Equal(past) {
		t.Fatalf("mtime %v, want %v", got, past)
	}
}

func TestSettleReportsAFileWithOtherContent(t *testing.T) {
	root := gdProject(t, map[string]string{"core/a.gd": "one\n"})
	snap, _ := Take(root)
	os.WriteFile(filepath.Join(root, "core", "a.gd"), []byte("two\n"), 0o644)
	want := []string{"core/a.gd differs after restoring the instrumentation"}
	if f := snap.Settle(root); !reflect.DeepEqual(f, want) {
		t.Fatalf("got %q", f)
	}
}

func TestSettleReportsADeletedFile(t *testing.T) {
	root := gdProject(t, map[string]string{"core/a.gd": "one\n"})
	snap, _ := Take(root)
	os.Remove(filepath.Join(root, "core", "a.gd"))
	want := []string{"core/a.gd differs after restoring the instrumentation"}
	if f := snap.Settle(root); !reflect.DeepEqual(f, want) {
		t.Fatalf("got %q", f)
	}
}

func TestSettleListsFindingsSortedByPath(t *testing.T) {
	root := gdProject(t, map[string]string{"b.gd": "1", "a.gd": "1", "c.gd": "1"})
	snap, _ := Take(root)
	for _, n := range []string{"b.gd", "a.gd"} {
		os.WriteFile(filepath.Join(root, n), []byte("2"), 0o644)
	}
	f := snap.Settle(root)
	if len(f) != 2 || f[0][:4] != "a.gd" || f[1][:4] != "b.gd" {
		t.Fatalf("got %q", f)
	}
}

func TestSettleDoesNotNormaliseLineEndings(t *testing.T) {
	// The engine restores byte for byte, so a CRLF/LF flip is a real change.
	root := gdProject(t, map[string]string{"a.gd": "x\r\ny\r\n"})
	snap, _ := Take(root)
	os.WriteFile(filepath.Join(root, "a.gd"), []byte("x\ny\n"), 0o644)
	if f := snap.Settle(root); len(f) != 1 {
		t.Fatalf("got %q", f)
	}
}

func TestSettleAcceptsAnUnchangedFileWithABOM(t *testing.T) {
	root := gdProject(t, map[string]string{"a.gd": "\xef\xbb\xbfextends Node\r\n"})
	snap, err := Take(root)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "a.gd"), []byte("\xef\xbb\xbfextends Node\r\n"), 0o644)
	if f := snap.Settle(root); len(f) != 0 {
		t.Fatalf("findings %q", f)
	}
}

func TestTakeOnlyVisitsWhatWalkGDVisits(t *testing.T) {
	root := gdProject(t, map[string]string{"a.gd": "1", "addons/x.gd": "1", "m.md": "1"})
	snap, _ := Take(root)
	if len(snap) != 1 {
		t.Fatalf("got %d entries", len(snap))
	}
	if _, ok := snap["a.gd"]; !ok {
		t.Fatal("a.gd missing")
	}
}

func TestTakeFailsOnAMissingRoot(t *testing.T) {
	if _, err := Take(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("want an error")
	}
}

// instrumentOnDisk leaves the state a run that was killed while the sources
// were rewritten leaves behind: the addon's backups, its manifest, and the
// marker in every source it names.
func instrumentOnDisk(t *testing.T, root, dir string, rels ...string) {
	t.Helper()
	var files []map[string]string
	for _, rel := range rels {
		p := filepath.Join(root, filepath.FromSlash(rel))
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(src)
		bkp := dir + "/backup/" + rel
		write(t, filepath.Join(root, filepath.FromSlash(bkp)), string(src))
		write(t, p, "# __NANO_COVERAGE_INSTRUMENTED__\n"+string(src))
		files = append(files, map[string]string{"path": "res://" + rel,
			"backup_path": "res://" + bkp, "original_hash": "sha256:" + hex.EncodeToString(sum[:])})
	}
	body, err := json.Marshal(map[string]any{"version": 1, "files": files})
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, filepath.FromSlash(dir), manifestName), string(body))
}

func gdWithManifest(t *testing.T, body string) string {
	t.Helper()
	root := project(t, map[string]string{"project.godot": body, "core/a.gd": "extends Node\n"})
	instrumentOnDisk(t, root, ".nano_coverage", "core/a.gd")
	return root
}

func TestRestoreManifestPutsTheSourcesBackAndDropsTheBackups(t *testing.T) {
	root := gdWithManifest(t, "config_version=5\n")
	f, err := RestoreManifest(root)
	if err != nil || len(f) != 0 {
		t.Fatalf("%v %q", err, f)
	}
	if body, _ := os.ReadFile(filepath.Join(root, "core", "a.gd")); string(body) != "extends Node\n" {
		t.Fatalf("source %q", body)
	}
	if _, err := os.Stat(filepath.Join(root, ".nano_coverage")); err == nil {
		t.Fatal(".nano_coverage stays")
	}
}

func TestRestoreManifestReadsTheProjectsBackupDir(t *testing.T) {
	root := project(t, map[string]string{
		"project.godot": "[nano_coverage]\ngeneral/backup_dir=\"res://build/nano\"\n",
		"core/a.gd":     "extends Node\n"})
	instrumentOnDisk(t, root, "build/nano", "core/a.gd")
	if f, err := RestoreManifest(root); err != nil || len(f) != 0 {
		t.Fatalf("%v %q", err, f)
	}
	if body, _ := os.ReadFile(filepath.Join(root, "core", "a.gd")); string(body) != "extends Node\n" {
		t.Fatalf("source %q", body)
	}
}

func TestRestoreManifestLeavesASourceSomebodyPutBackAlone(t *testing.T) {
	root := gdWithManifest(t, "config_version=5\n")
	p := filepath.Join(root, "core", "a.gd")
	write(t, p, "extends Node # mine\n")
	if f, err := RestoreManifest(root); err != nil || len(f) != 0 {
		t.Fatalf("%v %q", err, f)
	}
	if body, _ := os.ReadFile(p); string(body) != "extends Node # mine\n" {
		t.Fatalf("source %q", body)
	}
}

func TestRestoreManifestKeepsTheManifestWhenABackupIsGone(t *testing.T) {
	root := gdWithManifest(t, "config_version=5\n")
	if err := os.Remove(filepath.Join(root, ".nano_coverage", "backup", "core", "a.gd")); err != nil {
		t.Fatal(err)
	}
	f, err := RestoreManifest(root)
	want := "res://core/a.gd: its backup is gone, res://.nano_coverage/backup/core/a.gd"
	if err != nil || !reflect.DeepEqual(f, []string{want}) {
		t.Fatalf("%v %q", err, f)
	}
	if _, err := os.Stat(filepath.Join(root, ".nano_coverage", manifestName)); err != nil {
		t.Fatal("the manifest went, so no later run can try again")
	}
}

func TestRestoreManifestRestoresABackupThatDoesNotMatchItsHashAndSaysSo(t *testing.T) {
	root := gdWithManifest(t, "config_version=5\n")
	bkp := filepath.Join(root, ".nano_coverage", "backup", "core", "a.gd")
	write(t, bkp, "extends Node\r\n")
	f, err := RestoreManifest(root)
	want := "res://core/a.gd: its backup does not match the hash the manifest records"
	if err != nil || !reflect.DeepEqual(f, []string{want}) {
		t.Fatalf("%v %q", err, f)
	}
	if body, _ := os.ReadFile(filepath.Join(root, "core", "a.gd")); string(body) != "extends Node\r\n" {
		t.Fatalf("source %q", body)
	}
}

func TestRestoreManifestReportsASourceItCannotWrite(t *testing.T) {
	root := gdWithManifest(t, "config_version=5\n")
	p := filepath.Join(root, "core", "a.gd")
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := RestoreManifest(root)
	if err != nil || len(f) != 1 || !strings.HasPrefix(f[0], "res://core/a.gd cannot be written back:") {
		t.Fatalf("%v %q", err, f)
	}
}

func TestRestoreManifestReportsAReadOnlySource(t *testing.T) {
	root := gdWithManifest(t, "config_version=5\n")
	p := filepath.Join(root, "core", "a.gd")
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	f, err := RestoreManifest(root)
	if err != nil || len(f) != 1 || !strings.HasPrefix(f[0], "res://core/a.gd cannot be written back:") {
		t.Fatalf("%v %q", err, f)
	}
}

// A manifest is a file like any other: a path in it that is not a res:// path
// inside the project is never written to, however it is spelled.
func TestRestoreManifestRefusesAPathOutsideTheProject(t *testing.T) {
	for _, bad := range []string{"user://core/a.gd", "res://../core/a.gd", "res://x/../../core/a.gd"} {
		root := gdWithManifest(t, "config_version=5\n")
		p := filepath.Join(root, ".nano_coverage", manifestName)
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		write(t, p, strings.ReplaceAll(string(body), "res://core/a.gd", bad))
		f, err := RestoreManifest(root)
		want := "the manifest names a path outside the project: " + bad
		if err != nil || !reflect.DeepEqual(f, []string{want}) {
			t.Fatalf("%s: %v %q", bad, err, f)
		}
	}
}

func TestRestoreManifestReportsAManifestThatDoesNotRead(t *testing.T) {
	root := gdWithManifest(t, "config_version=5\n")
	write(t, filepath.Join(root, ".nano_coverage", manifestName), "{")
	f, err := RestoreManifest(root)
	if err != nil || len(f) != 1 || !strings.HasPrefix(f[0], manifestName+" of the instrumentation does not read:") {
		t.Fatalf("%v %q", err, f)
	}
}

func TestRestoreManifestDoesNothingWithoutAManifest(t *testing.T) {
	root := project(t, map[string]string{"project.godot": "config_version=5\n", "core/a.gd": "extends Node\n"})
	if f, err := RestoreManifest(root); err != nil || f != nil {
		t.Fatalf("%v %q", err, f)
	}
}

func TestRestoreManifestFails(t *testing.T) {
	t.Run("no project", func(t *testing.T) {
		if _, err := RestoreManifest(filepath.Join(t.TempDir(), "gone")); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("no project.godot", func(t *testing.T) {
		if _, err := RestoreManifest(t.TempDir()); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("unreadable project.godot", func(t *testing.T) {
		root := project(t, map[string]string{"project.godot": "[nano_coverage\n"})
		if _, err := RestoreManifest(root); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("backup_dir outside the project", func(t *testing.T) {
		for _, bad := range []string{"/tmp/nano", "res://../nano"} {
			root := project(t, map[string]string{"project.godot": "[nano_coverage]\ngeneral/backup_dir=\"" + bad + "\"\n"})
			if _, err := RestoreManifest(root); err == nil {
				t.Fatalf("%s: want an error", bad)
			}
		}
	})
	t.Run("unreadable manifest", func(t *testing.T) {
		root := gdWithManifest(t, "config_version=5\n")
		p := filepath.Join(root, ".nano_coverage", manifestName)
		os.Remove(p)
		os.Mkdir(p, 0o755)
		if _, err := RestoreManifest(root); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("no marker profile", func(t *testing.T) {
		root := gdWithManifest(t, "config_version=5\n")
		loadProfiles = func() ([]linecov.Profile, error) { return nil, nil }
		defer func() { loadProfiles = linecov.LoadProfiles }()
		if _, err := RestoreManifest(root); err == nil {
			t.Fatal("want an error")
		}
	})
}

// link makes dir a directory link to target: a symlink where the system
// grants one, else a junction, which Windows grants without privilege.
func link(t *testing.T, dir, target string) {
	t.Helper()
	if os.Symlink(target, dir) == nil {
		return
	}
	if runtime.GOOS != "windows" {
		t.Skip("no symlink here")
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", dir, target).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J %s %s: %v\n%s", dir, target, err, out)
	}
}

// A link inside the project passes every textual check on a res:// path;
// what it points to outside is still never written to.
func TestRestoreManifestDoesNotWriteThroughALinkOutOfTheProject(t *testing.T) {
	root := project(t, map[string]string{"project.godot": "config_version=5\n"})
	outside := t.TempDir()
	write(t, filepath.Join(outside, "a.gd"), "extends Node\n")
	link(t, filepath.Join(root, "core"), outside)
	instrumentOnDisk(t, root, ".nano_coverage", "core/a.gd")
	f, err := RestoreManifest(root)
	if err != nil || len(f) != 1 || !strings.HasPrefix(f[0], "res://core/a.gd cannot be written back:") {
		t.Fatalf("%v %q", err, f)
	}
	if body, _ := os.ReadFile(filepath.Join(outside, "a.gd")); !strings.HasPrefix(string(body), "# __NANO_COVERAGE_INSTRUMENTED__\n") {
		t.Fatalf("written outside the project: %q", body)
	}
}

func TestRestoreManifestDoesNotReadOrRemoveThroughALinkedBackupDir(t *testing.T) {
	root := project(t, map[string]string{"project.godot": "config_version=5\n", "core/a.gd": "extends Node\n"})
	outside := t.TempDir()
	link(t, filepath.Join(root, ".nano_coverage"), outside)
	instrumentOnDisk(t, root, ".nano_coverage", "core/a.gd")
	if _, err := RestoreManifest(root); err == nil {
		t.Fatal("want an error")
	}
	for _, p := range []string{manifestName, filepath.Join("backup", "core", "a.gd")} {
		if _, err := os.Stat(filepath.Join(outside, p)); err != nil {
			t.Fatalf("removed outside the project: %s", p)
		}
	}
}
