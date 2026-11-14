package target

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// projectTarget is a project whose configuration file holds text.
func projectTarget(t *testing.T, text string) (string, Target) {
	t.Helper()
	root := t.TempDir()
	tgt := Project(root)
	writeFile(t, tgt.Path, text)
	return root, tgt
}

// A file that cannot be read is reported as the read error, not as a file
// that changed since it was read.
func TestWriteConfigReportsTheReadErrorOfAnUnreadableFile(t *testing.T) {
	_, tgt := projectTarget(t, "")
	if err := os.Remove(tgt.Path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(tgt.Path, 0o755); err != nil {
		t.Fatal(err)
	}
	err := tgt.Write("[commit]\nthreshold = 2\n", "x")
	var pe *fs.PathError
	if !errors.As(err, &pe) || strings.Contains(err.Error(), "changed since it was read") {
		t.Fatalf("err = %v, want the read error", err)
	}
}

// A directory that cannot be made is reported as that, not as the failed
// temporary file the write would try next.
func TestWriteConfigReportsADirectoryItCannotMake(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux"), "")
	tgt := Project(root)
	err := tgt.Write("", "x")
	var pe *fs.PathError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v, want a path error", err)
	}
	// Where the reader already refuses a path under a file (ENOTDIR on
	// POSIX), the read is the step that fails; Windows reads it as absent.
	readRefuses := runtime.GOOS != "windows" && pe.Op == "open"
	if pe.Op != "mkdir" && !readRefuses {
		t.Fatalf("err = %v (op %q), want the failed mkdir", err, pe.Op)
	}
}

func TestWriteConfigRefusesATextItWasNotMadeFrom(t *testing.T) {
	_, tgt := projectTarget(t, "[commit]\nthreshold = 5\n")
	err := tgt.Write("[commit]\nthreshold = 2\n", "[commit]\nthreshold = 3\n")
	if err == nil || !strings.Contains(err.Error(), "changed since it was read; nothing written") {
		t.Fatalf("err = %v, want a refusal", err)
	}
	// A file that went missing since it was read changed too.
	os.Remove(tgt.Path)
	if err := tgt.Write("[commit]\nthreshold = 5\n", "x"); err == nil {
		t.Fatal("a vanished file was written over")
	}
	if _, err := os.Stat(tgt.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat = %v, want the file still missing", err)
	}
}

func TestWriteConfigReportsAFileItCannotRead(t *testing.T) {
	_, tgt := projectTarget(t, "")
	os.Remove(tgt.Path)
	os.Mkdir(tgt.Path, 0o755)
	if err := tgt.Write("", "x"); err == nil {
		t.Fatal("no error for a configuration that is a directory")
	}
}

// Two writes made from the same text must not both land: the second would
// take the first one's change back unseen.
func TestWriteConfigLetsOnlyOneOfTwoWritesFromTheSameTextLand(t *testing.T) {
	before := "[commit]\nthreshold = 2\n"
	_, tgt := projectTarget(t, before)
	real := replaceText
	t.Cleanup(func() { replaceText = real })
	replaceText = func(path, text string) error {
		time.Sleep(50 * time.Millisecond)
		return real(path, text)
	}
	var landed atomic.Int32
	var wg sync.WaitGroup
	for _, next := range []string{"[commit]\nthreshold = 3\n", "[commit]\nthreshold = 4\n"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tgt.Write(before, next) == nil {
				landed.Add(1)
			}
		}()
	}
	wg.Wait()
	if landed.Load() != 1 {
		t.Fatalf("%d writes landed, want 1", landed.Load())
	}
}

func TestSetReadFileForTestServesEveryReadUntilItIsRestored(t *testing.T) {
	_, project := projectTarget(t, "[commit]\nlanguage = \"de\"\n")
	restore := SetReadFileForTest(func(string) ([]byte, error) { return []byte("[commit]\n"), nil })
	if text, err := project.Read(); err != nil || text != "[commit]\n" {
		t.Errorf("replaced: %q, %v", text, err)
	}
	restore()
	if text, err := project.Read(); err != nil || text != "[commit]\nlanguage = \"de\"\n" {
		t.Errorf("restored: %q, %v", text, err)
	}
}

func TestUnknownNamesTheKeysOfThisTargetsFileItDoesNotKnow(t *testing.T) {
	_, project := projectTarget(t, "")
	got, err := project.Unknown("[commit]\nlanguage = \"de\"\n[dotnet]\nsdk = \"8\"\n")
	if err != nil || len(got) != 1 || got[0] != "dotnet" {
		t.Errorf("project: %v, %v", got, err)
	}
	got, err = Global(t.TempDir()).Unknown("[model]\nname = \"x\"\n[commit]\nlanguage = \"de\"\n")
	if err != nil || len(got) != 1 || got[0] != "commit" {
		t.Errorf("global: %v, %v", got, err)
	}
}
