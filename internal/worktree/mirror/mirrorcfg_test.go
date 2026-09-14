package mirror

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func write(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".loomux", "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMirrorReadsTheConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	write(t, root, "[worktree]\nmirror = [\".tools\", \".ultraloom/vendor\"]\n")

	got, err := Mirror(root)
	if err != nil {
		t.Fatalf("Mirror: %v", err)
	}
	if !slices.Equal(got, []string{".tools", ".ultraloom/vendor"}) {
		t.Fatalf("Mirror = %q, want the two configured paths in order", got)
	}
}

// The three ways of having nothing to do, all of them an empty answer and no
// error: a hook that fires in every project must not report any of them.
func TestTheThreeNoOpCases(t *testing.T) {
	t.Run("no config file", func(t *testing.T) {
		got, err := Mirror(t.TempDir())
		if err != nil || len(got) != 0 {
			t.Fatalf("Mirror = %q, %v; want empty and nil", got, err)
		}
	})
	t.Run("config without the section", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "[verify]\nlint = \"ruff check .\"\n")
		got, err := Mirror(root)
		if err != nil || len(got) != 0 {
			t.Fatalf("Mirror = %q, %v; want empty and nil", got, err)
		}
	})
	t.Run("section without the key", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "[worktree]\n")
		got, err := Mirror(root)
		if err != nil || len(got) != 0 {
			t.Fatalf("Mirror = %q, %v; want empty and nil", got, err)
		}
	})
}

// A path that climbs out of the project would junction something outside it.
//
// The entries go into TOML literal strings on purpose: in a basic string
// `C:\absolute` is an invalid escape, so the parser would reject the line and
// the test would pass without the path check ever running.
//
// `C:\absolute` is guarded rather than merely annotated as Windows-specific:
// on POSIX `filepath.IsLocal` reads the backslash as an ordinary byte, so the
// entry is one legal filename there and `Mirror` accepts it. Unguarded, the
// case would not record a platform difference -- it would fail.
//
// The empty entry belongs here for the same reason the others do -- it is
// refused -- even though nothing about it climbs anywhere.
func TestMirrorRefusesAPathThatLeavesTheProject(t *testing.T) {
	entries := []string{"../elsewhere", "/absolute", ".tools/../..", ""}
	if runtime.GOOS == "windows" {
		entries = append(entries, "C:\\absolute")
	}
	for _, entry := range entries {
		root := t.TempDir()
		write(t, root, "[worktree]\nmirror = ['"+entry+"']\n")
		if _, err := Mirror(root); err == nil {
			t.Fatalf("%q was accepted", entry)
		}
	}
}

// Damage is an error, never an empty answer: read as "nothing to mirror", a
// broken file would switch the whole mechanism off without a word.
func TestBrokenTomlIsAnError(t *testing.T) {
	root := t.TempDir()
	write(t, root, "[worktree\nmirror = ")
	if _, err := Mirror(root); err == nil {
		t.Fatal("broken toml was read as an empty answer")
	}
}

func TestAMirrorThatIsNotAListOfStringsIsAnError(t *testing.T) {
	root := t.TempDir()
	write(t, root, "[worktree]\nmirror = \".tools\"\n")
	if _, err := Mirror(root); err == nil {
		t.Fatal("a string was accepted where a list belongs")
	}
}

// Unreadable is not the same as absent: only a missing file means "nothing to
// mirror", and a config.toml that cannot be read has to say so.
func TestAnUnreadableConfigIsAnError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Mirror(root); err == nil {
		t.Fatal("a config.toml that is not a file was read as an empty answer")
	}
}
