package hooks

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// An 8.3 alias matches the long name Windows may have made it from: the
// stem's first letters (two at least) or two letters and a four-digit hex
// hash, a ~ and digits, and the first three letters of the extension.
func TestAliasMatchesTheLongNameItMayStandFor(t *testing.T) {
	for _, row := range []struct {
		alias, long string
		want        bool
	}{
		{"LOOMUX~1", ".loomux", true},
		{"loomux~1", ".loomux", true},
		{"LOOMUX~12", ".loomux", true},
		{"LOOMU~10", ".loomux", true},
		{"LO~1", ".loomux", true},
		{"LO1A2B~1", ".loomux", true},
		{"LO1A2G~1", ".loomux", false},
		{"XY1A2B~1", ".loomux", false},
		{"L~1", ".loomux", false},
		{"LOOMUY~1", ".loomux", false},
		{"LOOMUX~", ".loomux", false},
		{"LOOMUX~X", ".loomux", false},
		{"LOOMUX1", ".loomux", false},
		{"CONFIG~1.TOM", "config.toml", true},
		{"CONFIG~1.TO", "config.toml", false},
		{"CONFIG~1", "config.toml", false},
		{"CONFIG~1.TXT", "config.toml", false},
		{"LOOMUX~1.TOM", ".loomux", false},
		{"PACKAG~1.JSO", "package-lock.json", true},
		{"CREDEN~1.JSO", "credentials.json", true},
		{"MYFILE~1.TXT", "my file.txt", true},
	} {
		if got := aliasMatches(row.alias, row.long); got != row.want {
			t.Errorf("aliasMatches(%q, %q) = %v, want %v", row.alias, row.long, got, row.want)
		}
	}
}

// A path is also spelled with each element's trailing dots and blanks gone,
// which Windows drops, and with each 8.3 alias put back as the protected
// name it may stand for.
func TestLexicalSpellingsFoldWhatWindowsFolds(t *testing.T) {
	elements := []string{".loomux", "config.toml"}
	for rel, want := range map[string][]string{
		".loomux./config.toml. ": {".loomux./config.toml", ".loomux/config.toml", ".loomux/config.toml. "},
		"LOOMUX~1/CONFIG~1.TOM":  {".loomux/CONFIG~1.TOM", ".loomux/config.toml", "LOOMUX~1/config.toml"},
		"docs/a.":                {"docs/a"},
		"./../x":                 nil,
		"a/.../b":                nil,
		".loomux/config.toml":    nil,
		"LOOMUX~1.":              {".loomux", "LOOMUX~1"},
	} {
		got := lexicalSpellings(rel, elements)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("%q: %q, want %q", rel, got, want)
		}
	}
	// Every element with a second form doubles the spellings, up to the cap.
	if got := lexicalSpellings("LOOMUX~1/LOOMUX~2/LOOMUX~3/LOOMUX~4/LOOMUX~5", elements); len(got) != maxLexical-1 {
		t.Errorf("%d spellings, want the cap %d without the path itself", len(got), maxLexical)
	}
}

// The elements an alias may stand for are the literal ones: a glob's
// wildcard element names no file.
func TestProtectedElementsAreLiteral(t *testing.T) {
	got := newJudge(t.TempDir(), config.Policy{}).protectedElements()
	if !slices.Contains(got, ".loomux") || !slices.Contains(got, "config.toml") {
		t.Fatalf("elements %q", got)
	}
	for _, e := range got {
		if strings.ContainsAny(e, "*?[") {
			t.Errorf("element %q holds a wildcard", e)
		}
	}
}

// The guard judges those spellings in the default mode, for a shell line
// and for a writing tool alike, without asking the file system.
func TestTheDefaultModeReadsShortNamesAndTrailingDots(t *testing.T) {
	root := t.TempDir()
	for _, line := range []string{
		"echo x > LOOMUX~1/config.toml",
		"echo x > .loomux/CONFIG~1.TOM",
		"echo x > .loomux/config.toml.",
		"rm -rf LOOMUX~1",
	} {
		if got := checkTool(root, "Bash", command(line), config.Policy{}); !slices.Contains(got, manifestReason) {
			t.Errorf("%q: reasons %q, want the manifest's", line, got)
		}
	}
	for _, path := range []string{"LOOMUX~1/config.toml", ".loomux/config.toml. "} {
		got := checkTool(root, "Write", map[string]any{"file_path": path, "content": "x"}, config.Policy{})
		if !slices.Contains(got, manifestReason) {
			t.Errorf("Write %q: reasons %q, want the manifest's", path, got)
		}
	}
	// Strict mode adds them to what it resolves: an alias of a file that is
	// not there yet resolves to nothing the rules know.
	strict := config.Policy{Strict: true}
	if got := checkTool(root, "Write", map[string]any{"file_path": ".loomux/CONFIG~1.TOM", "content": "x"}, strict); !slices.Contains(got, manifestReason) {
		t.Errorf("strict Write of an alias: reasons %q, want the manifest's", got)
	}
	for _, line := range []string{"echo x > PROGRA~1/x", "echo x > docs/a.", "echo x > .loomux/CONFIG~1.TXT"} {
		if got := checkTool(root, "Bash", command(line), config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}
