//go:build windows

package hooks

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"golang.org/x/sys/windows"
)

// shortName is the 8.3 alias the file system keeps for path, or "" when this
// volume keeps none; asked through GetShortPathName, not through the resolver
// the guard uses.
func shortName(t *testing.T, path string) string {
	t.Helper()
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, windows.MAX_PATH)
	length, err := windows.GetShortPathName(wide, &buffer[0], uint32(len(buffer)))
	if err != nil || length == 0 || int(length) > len(buffer) {
		t.Skipf("no short name for %q: %v", path, err)
	}
	short := windows.UTF16ToString(buffer[:length])
	if strings.EqualFold(short, path) {
		return ""
	}
	return short
}

// Windows opens example. as example; strict mode resolves it, the default
// mode drops the dot without asking the file system.
func TestStrictModeResolvesATrailingDot(t *testing.T) {
	catalog(t, "example")
	root := project(t)
	mkfile(t, root, ".loomux/flows/example/flow.toml")
	line := "echo x > .loomux/flows/example./flow.toml"
	if got := checkTool(root, "Bash", command(line), config.Policy{}); !slices.Equal(got, []string{bundledWant("example")}) {
		t.Fatalf("default: reasons %q", got)
	}
	if got := checkTool(root, "Bash", command(line), strictPolicy); !slices.Equal(got, []string{bundledWant("example")}) {
		t.Fatalf("strict: reasons %q", got)
	}
	lands := "in strict mode loomux refuses a write to .loomux./con$X: the expansion may land on a protected path"
	if got := checkTool(root, "Bash", command("echo x > .loomux./con$X"), strictPolicy); !slices.Equal(got, []string{lands}) {
		t.Fatalf("strict expansion behind a trailing dot: reasons %q", got)
	}
	write := map[string]any{"file_path": ".loomux/flows/example/flow.toml "}
	if got := checkTool(root, "Write", write, strictPolicy); !slices.Equal(got, []string{bundledWant("example")}) {
		t.Fatalf("strict write with a trailing blank: reasons %q", got)
	}
}

// A target the file system cannot resolve, and a root it cannot, are
// refused in strict mode; a device name is no place to resolve.
func TestStrictModeRefusesWhatItCannotResolve(t *testing.T) {
	root := project(t)
	got := checkTool(root, "Write", map[string]any{"file_path": "//server/share"}, strictPolicy)
	if len(got) != 1 || !strings.HasPrefix(got[0], "loomux cannot resolve //server/share, so it refuses in strict mode: ") {
		t.Errorf("an unresolvable target: reasons %q", got)
	}
	if _, err := newJudge("C:relative", strictPolicy).resolvedSpellings([]string{"x"}); err == nil || !strings.HasPrefix(err.Error(), "loomux cannot resolve the project root C:relative") {
		t.Errorf("an unresolvable root: %v", err)
	}
	if got := checkTool(root, "Bash", command("echo x > nul 2> con"), strictPolicy); len(got) != 0 {
		t.Errorf("a device: reasons %q", got)
	}
}

func TestStrictModeResolvesAShortName(t *testing.T) {
	root := project(t)
	short := shortName(t, filepath.Join(root, ".loomux"))
	if short == "" {
		t.Skip("this volume keeps no 8.3 names")
	}
	// The default mode reads the alias without the file system.
	line := "echo x > " + filepath.Base(short) + "/config.toml"
	if got := checkTool(root, "Bash", command(line), config.Policy{}); !slices.Equal(got, []string{manifestReason}) {
		t.Fatalf("default: reasons %q", got)
	}
	if got := checkTool(root, "Bash", command(line), strictPolicy); !slices.Equal(got, []string{manifestReason}) {
		t.Fatalf("strict: reasons %q", got)
	}
	expansion := filepath.Base(short) + "/con$X"
	lands := "in strict mode loomux refuses a write to " + expansion + ": the expansion may land on a protected path"
	if got := checkTool(root, "Bash", command("echo x > "+expansion), strictPolicy); !slices.Equal(got, []string{lands}) {
		t.Fatalf("strict expansion behind a short name: reasons %q", got)
	}
}
