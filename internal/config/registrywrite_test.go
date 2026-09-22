package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRegistrySortsByScope(t *testing.T) {
	text := RenderRegistry([]Area{
		{Scope: "project/zeta", Path: "C:/z"},
		{Scope: "project/alpha", Path: "C:/a"},
	})
	if strings.Index(text, "project/alpha") > strings.Index(text, "project/zeta") {
		t.Fatalf("areas are not sorted:\n%s", text)
	}
}

// The file byte for byte: one empty line between two tables, none in front of
// the first and none after the last -- the shape `area-add/new-area` recorded
// from the reference.
func TestRenderRegistrySeparatesTablesByOneEmptyLine(t *testing.T) {
	text := RenderRegistry([]Area{
		{Scope: "project/b", Path: "C:/b"},
		{Scope: "project/a", Path: "C:/a"},
	})
	want := "[[area]]\nscope = \"project/a\"\npath = \"C:/a\"\n\n" +
		"[[area]]\nscope = \"project/b\"\npath = \"C:/b\"\n"
	if text != want {
		t.Fatalf("RenderRegistry =\n%q\nwant\n%q", text, want)
	}
}

// A flag that is false is not written: the file of an ordinary area stays the
// one it was before the flag existed.
func TestRenderRegistryOmitsFalseFlags(t *testing.T) {
	text := RenderRegistry([]Area{{Scope: "project/a", Path: "C:/a"}})
	for _, key := range []string{"readonly", "signpost", "shared", "workspace", "wiki"} {
		if strings.Contains(text, key) {
			t.Fatalf("rendered %q for a plain area:\n%s", key, text)
		}
	}
}

// QuoteTOML escapes what a TOML basic string forbids raw, the way the Python
// side's `_quote` does, so the two renderings stay comparable byte for byte.
func TestQuoteTOMLEscapesWhatTOMLForbids(t *testing.T) {
	for _, testCase := range []struct {
		value string
		want  string
	}{
		{"C:/a", `"C:/a"`},
		{"a\\b", `"a\\b"`},
		{`a"b`, `"a\"b"`},
		{"a\b\t\n\f\rb", `"a\b\t\n\f\rb"`},
		{"a\x07b", `"a\u0007b"`},
		{"a\x7fb", `"a\u007fb"`},
		// The space is the first character that is no control character, and
		// it stays raw: "92 Engineering" is a path of this machine's registry.
		{"92 Engineering", `"92 Engineering"`},
	} {
		if got := QuoteTOML(testCase.value); got != testCase.want {
			t.Fatalf("QuoteTOML(%q) = %s, want %s", testCase.value, got, testCase.want)
		}
	}
}

// Every field survives the round trip, flags and wiki included, and a path
// carrying the characters TOML forbids raw comes back the way it went in.
func TestWriteRegistryRoundTripsEveryField(t *testing.T) {
	stateDir := t.TempDir()
	want := []Area{
		{Scope: "project/loomux", Path: "C:/Users/x/loomux", WikiPath: "docs/wiki"},
		{
			Scope:     "project/space",
			Path:      "C:/Users/x/space\ttabbed\\slashed\"quoted\x07",
			WikiPath:  "wiki",
			ReadOnly:  true,
			Signpost:  true,
			Shared:    true,
			Workspace: true,
		},
	}
	if err := WriteRegistry(stateDir, want); err != nil {
		t.Fatalf("WriteRegistry: %v", err)
	}
	got, err := ReadRegistry(stateDir)
	if err != nil {
		t.Fatalf("ReadRegistry: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("read %d areas, wrote %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("area %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// A registry the reader would reject is not written at all: the standing file
// keeps its bytes, and no temporary file is left behind.
func TestWriteRegistryRefusesWhatTheReaderWouldReject(t *testing.T) {
	stateDir := t.TempDir()
	first := []Area{{Scope: "project/a", Path: "C:/a"}}
	if err := WriteRegistry(stateDir, first); err != nil {
		t.Fatalf("WriteRegistry: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(stateDir, registryName))
	if err != nil {
		t.Fatalf("reading the standing registry: %v", err)
	}
	broken := []Area{
		{Scope: "project/a", Path: "C:/a", Signpost: true},
		{Scope: "project/b", Path: "C:/b", Signpost: true},
	}
	if err := WriteRegistry(stateDir, broken); err == nil {
		t.Fatal("WriteRegistry: want an error for two signposts")
	}
	after, err := os.ReadFile(filepath.Join(stateDir, registryName))
	if err != nil {
		t.Fatalf("the standing registry did not survive: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("the registry changed:\n%s\nwant\n%s", after, before)
	}
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != registryName && entry.Name() != registryLockName {
			t.Fatalf("the refused write left %q behind", entry.Name())
		}
	}
}

func TestAddAreaAppends(t *testing.T) {
	stateDir := t.TempDir()
	if err := WriteRegistry(stateDir, []Area{{Scope: "project/a", Path: "C:/a"}}); err != nil {
		t.Fatalf("WriteRegistry: %v", err)
	}
	if err := AddArea(stateDir, Area{Scope: "project/b", Path: "C:/b"}); err != nil {
		t.Fatalf("AddArea: %v", err)
	}
	got, err := ReadRegistry(stateDir)
	if err != nil {
		t.Fatalf("ReadRegistry: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("read %d areas, want 2", len(got))
	}
}

// A second area of the same scope is an error, not an overwrite.
func TestAddAreaRefusesADuplicateScope(t *testing.T) {
	stateDir := t.TempDir()
	area := Area{Scope: "project/a", Path: "C:/a"}
	if err := AddArea(stateDir, area); err != nil {
		t.Fatalf("AddArea: %v", err)
	}
	err := AddArea(stateDir, Area{Scope: "project/a", Path: "C:/other"})
	if err == nil || !strings.Contains(err.Error(), "project/a") {
		t.Fatalf("AddArea: want an error naming the scope, got %v", err)
	}
}

// On an empty machine AddArea writes the first registry.
func TestAddAreaCreatesTheRegistry(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	if err := AddArea(stateDir, Area{Scope: "project/a", Path: "C:/a"}); err != nil {
		t.Fatalf("AddArea: %v", err)
	}
	if _, err := ReadRegistry(stateDir); err != nil {
		t.Fatalf("ReadRegistry: %v", err)
	}
}

// A registry that is there but unreadable stops AddArea instead of being
// taken for an empty machine and overwritten.
func TestAddAreaRefusesAnUnreadableRegistry(t *testing.T) {
	stateDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stateDir, registryName), []byte("not toml ["), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := AddArea(stateDir, Area{Scope: "project/a", Path: "C:/a"}); err == nil {
		t.Fatal("AddArea: want an error for a registry that does not parse")
	}
}

// A state directory that cannot be made stops both entry points before they
// take a lock.
func TestRegistryWritesRefuseAnUnmakeableStateDir(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	stateDir := filepath.Join(blocker, "state")
	if err := WriteRegistry(stateDir, []Area{{Scope: "project/a", Path: "C:/a"}}); err == nil {
		t.Fatal("WriteRegistry: want an error for a state directory under a file")
	}
	if err := AddArea(stateDir, Area{Scope: "project/a", Path: "C:/a"}); err == nil {
		t.Fatal("AddArea: want an error for a state directory under a file")
	}
}

// A lock that cannot be opened stops the write; a directory of that name is
// the cheapest way to say so.
func TestRegistryWritesRefuseAnUnopenableLock(t *testing.T) {
	stateDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(stateDir, registryLockName), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := WriteRegistry(stateDir, []Area{{Scope: "project/a", Path: "C:/a"}}); err == nil {
		t.Fatal("WriteRegistry: want an error for a lock that cannot be opened")
	}
}

// The swap itself can fail after the reader has accepted the text.
func TestWriteRegistryReportsAFailedSwap(t *testing.T) {
	stateDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(stateDir, registryName), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := WriteRegistry(stateDir, []Area{{Scope: "project/a", Path: "C:/a"}}); err == nil {
		t.Fatal("WriteRegistry: want an error when the swap cannot happen")
	}
}
