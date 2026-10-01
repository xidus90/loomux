package hooks

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// The header lines of a patch name every file it changes, in either form
// and with either line end; /dev/null names none, and a time stamp after a
// tab is no part of the name.
func TestPatchPathsReadsEveryHeader(t *testing.T) {
	text := strings.Join([]string{
		"From 1 Mon Sep 17 00:00:00 2001",
		"---",
		"diff --git a/x.go b/y.go",
		"rename from x.go",
		"rename to y.go",
		"copy to z.go",
		"--- a/x.go",
		"+++ b/y.go",
		"--- /dev/null",
		"+++ new.txt\t2026-10-01 12:00:00",
		`+++ "b/with space.txt"`,
		"@@ -1 +1 @@",
	}, "\r\n")
	want := []string{"a/x.go", "b/y.go", "x.go", "y.go", "z.go", "new.txt", "b/with space.txt"}
	if got := patchPaths(text); !slices.Equal(got, want) {
		t.Fatalf("paths %q, want %q", got, want)
	}
	if got := patchPaths("+++ \n--- /dev/null\n+++ \"\"\n"); len(got) != 0 {
		t.Fatalf("a header without a name: %q", got)
	}
}

// -pN drops N leading elements; without -p every level counts, the base
// name included, as patch may take any of them.
func TestStrippedPathsDropLeadingElements(t *testing.T) {
	for _, row := range []struct {
		raw   string
		strip int
		want  []string
	}{
		{"a/b/c.go", 1, []string{"b/c.go"}},
		{"a/b/c.go", 0, []string{"a/b/c.go"}},
		{"a/b/c.go", 3, nil},
		{"a/b/c.go", -1, []string{"a/b/c.go", "b/c.go", "c.go"}},
		{"c.go", -1, []string{"c.go"}},
	} {
		if got := strippedPaths(row.raw, row.strip); !slices.Equal(got, row.want) {
			t.Errorf("%q -p%d: %q, want %q", row.raw, row.strip, got, row.want)
		}
	}
}

// patch, git apply and git am write the files their patch names, read
// from disk where the line names the patch, or from the heredoc the line
// holds; patch also writes the file it names and its -o.
func TestAPatchWritesTheFilesItNames(t *testing.T) {
	root := holesWorld(t)
	mkfile(t, root, "sub/keep.txt")
	for line, want := range map[string][]string{
		"git apply p.diff":                       {"w:.loomux/config.toml"},
		"git apply -R --check p.diff":            {"w:.loomux/config.toml"},
		"git apply ok.diff":                      {"w:src/a.go"},
		"git apply -p0 ok.diff":                  {"w:a/src/a.go", "w:b/src/a.go"},
		"git apply --directory=.loomux q.diff":   {"w:.loomux/config.toml"},
		"git apply --directory .loomux q.diff":   {"w:.loomux/config.toml"},
		"git apply --exclude x -p 1 ok.diff":     {"w:src/a.go"},
		"git -C . apply ok.diff":                 {"w:src/a.go"},
		"git am p.patch":                         {"w:.loomux/config.toml"},
		"git am < p.patch":                       {"w:.loomux/config.toml"},
		"git am":                                 nil,
		"patch .loomux/config.toml < p.diff":     {"w:.loomux/config.toml"},
		"patch -o out.toml a < ok.diff":          {"w:a", "w:out.toml"},
		"patch --output=out.toml a ok.diff":      {"w:a", "w:out.toml"},
		"patch -p1 < p.diff":                     {"w:.loomux/config.toml"},
		"patch -p 1 -i p.diff":                   {"w:.loomux/config.toml"},
		"patch --strip=1 --input=p.diff":         {"w:.loomux/config.toml"},
		"patch --strip 1 --input p.diff":         {"w:.loomux/config.toml"},
		"patch -d sub -p1 -i ../p.diff":          {"w:sub/.loomux/config.toml"},
		"patch --directory=sub -p1 -i ../p.diff": {"w:sub/.loomux/config.toml"},
		"patch -r rej.txt -p1 < ok.diff":         {"w:rej.txt", "w:src/a.go"},
		"patch < p.diff":                         {"w:.loomux/config.toml", "w:a/.loomux/config.toml", "w:b/.loomux/config.toml", "w:config.toml"},
		"patch":                                  nil,
		"patch -p1 <p.diff":                      {"w:.loomux/config.toml"},
		"patch -d sub -p1 < p.diff":              {"w:sub/.loomux/config.toml"},
		"git apply --directoryx ok.diff":         {"w:src/a.go"},
		"git apply ok.diff\n+++ b/x.txt":         {"w:src/a.go"},
		"cat <<'EOF'\n+++ b/x.txt\nEOF":          nil,
		"git apply <<'EOF'\n--- a/.loomux/config.toml\n+++ b/.loomux/config.toml\nEOF": {"w:.loomux/config.toml", "w:a/.loomux/config.toml", "w:b/.loomux/config.toml", "w:config.toml"},
	} {
		targets, _ := shellWrites(root, line)
		if got := spelled(targets); !slices.Equal(got, want) {
			t.Errorf("%q: targets %q, want %q", line, got, want)
		}
	}
}

// A patch the guard cannot read is refused with a reason of its own: the
// files it would change are unknown.
func TestAPatchTheGuardCannotReadRefuses(t *testing.T) {
	root := holesWorld(t)
	for _, line := range []string{"git apply missing.diff", "patch -p1 -i missing.diff", "patch -p1 < missing.diff", "git am src"} {
		got := checkTool(root, "Bash", command(line), config.Policy{})
		if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "cannot read the patch") }) {
			t.Errorf("%q: reasons %q, want the unreadable patch", line, got)
		}
	}
	if got := checkTool(root, "Bash", command("git apply ok.diff"), config.Policy{}); len(got) != 0 {
		t.Errorf("a patch of an unkept file: reasons %q", got)
	}
}
