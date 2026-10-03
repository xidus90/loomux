package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config/schema"
)

// isolateConfigState points every directory a config run may read outside
// its --root at temporary ones.
func isolateConfigState(t *testing.T) {
	t.Helper()
	for _, name := range []string{"LOOMUX_STATE_DIR", "XDG_CONFIG_HOME", "LOCALAPPDATA"} {
		t.Setenv(name, t.TempDir())
	}
}

// A --propose switched off is named as such even when a later flag breaks
// the parse: the subcommand is taken before the flags are read, so the
// write is known to be one.
func TestConfigNamesASwitchedOffProposeBeforeABrokenParse(t *testing.T) {
	isolateConfigState(t)
	root := configRoot(t, "")
	code, _, errOut := runConfig(t, "", "set", "commit.language", "de", "--propose=false", "--bogus", "--root", root)
	if code != 2 || !strings.Contains(errOut, "--propose given and switched off") {
		t.Fatalf("code %d, stderr %q; want the switched-off refusal", code, errOut)
	}
}

// The project file is named by its short name in a parse error, not by the
// absolute path the global file is named by.
func TestConfigNamesTheProjectFileByItsShortName(t *testing.T) {
	isolateConfigState(t)
	root := configRoot(t, "[commit\n")
	code, _, errOut := runConfig(t, "", "list", "--root", root)
	if code != 1 || !strings.HasPrefix(errOut, "loomux config: "+schema.ProjectFile+": not valid TOML") {
		t.Fatalf("code %d, stderr %q; want the short name", code, errOut)
	}
	if strings.Contains(errOut, root) {
		t.Fatalf("stderr %q names the root %q", errOut, root)
	}
}

func TestShortenKeepsAValueOfExactlyTheListWidth(t *testing.T) {
	value := strings.Repeat("ä", listWidth)
	if got := shorten(value); got != value {
		t.Fatalf("shorten(%d runes) = %q, want it whole", listWidth, got)
	}
	longer := value + "b"
	if got := shorten(longer); got != strings.Repeat("ä", listWidth-1)+"…" {
		t.Fatalf("shorten(%d runes) = %q", listWidth+1, got)
	}
}

func TestConfigListCountsTwoTableListEntries(t *testing.T) {
	isolateConfigState(t)
	root := configRoot(t, "[[commit.allow]]\nregex = \"x\"\nreason = \"y\"\n\n[[commit.allow]]\nregex = \"z\"\nreason = \"w\"\n")
	code, out, _ := runConfig(t, "", "list", "--root", root)
	if code != 0 || !strings.Contains(out, "2 entries") || strings.Contains(out, "1 entry") {
		t.Fatalf("code %d, list:\n%s", code, out)
	}
}

// A write without a hint ends with the line that names the file: no empty
// line after it.
func TestConfigSetWithoutAHintEndsWithTheWrittenLine(t *testing.T) {
	isolateConfigState(t)
	root := configRoot(t, "")
	code, _, errOut := runConfig(t, "", "set", "commit.threshold", "4", "--yes", "--root", root)
	want := "loomux config: wrote " + filepath.Join(root, ".loomux", "config.toml") + "\n"
	if code != 0 || !strings.HasSuffix(errOut, want) {
		t.Fatalf("code %d, stderr %q; want it to end with %q", code, errOut, want)
	}
}

// A file that cannot be read is reported as the read error, not as a file
// that changed since it was read.
func TestWriteConfigReportsTheReadErrorOfAnUnreadableFile(t *testing.T) {
	_, target := projectTarget(t, "")
	if err := os.Remove(target.path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target.path, 0o755); err != nil {
		t.Fatal(err)
	}
	err := writeConfig(target, "[commit]\nthreshold = 2\n", "x")
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
	target, err := resolveConfigTarget(root, false)
	if err != nil {
		t.Fatal(err)
	}
	err = writeConfig(target, "", "x")
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
