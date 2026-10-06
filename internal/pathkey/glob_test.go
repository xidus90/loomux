package pathkey

import "testing"

// A leading **/ stands for any directory, the root included, and is anchored
// at an element: my.loomux is no .loomux.
func TestGlobReadsALeadingDoubleStarAsAnyDirectory(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"**/.loomux/config.toml", ".loomux/config.toml", true},
		{"**/.loomux/config.toml", "../sibling/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "/repo/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "C:/repo/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "my.loomux/config.toml", false},
		{"**/.loomux/config.toml", ".loomux/config.toml.bak", false},
		{"**/.loomux/state/runs/**", "x/.loomux/state/runs", true},
		{"**/.loomux/state/runs/**", "x/.loomux/state/runs/0001.jsonl", true},
		{"**/.loomux/state/runs/**", ".loomux/state/runsx/0001.jsonl", false},
	} {
		got, err := Glob(row.pattern, row.name)
		if err != nil || got != row.want {
			t.Errorf("Glob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
	if _, err := Glob("**/foo/*[x", "a/foo/b"); err == nil {
		t.Fatal("a bad class behind **/ must still be an error")
	}
}

// A pattern with a slash ending in /** takes the directory and all below it;
// one without a slash looks at the file name in any directory.
func TestGlobOfDirectoriesAndNames(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"docs/**", "docs", true},
		{"docs/**", "docs/de/a.md", true},
		{"docs/**", "docsx/a.md", false},
		{"*.md", "docs/de/ä.md", true},
		{"*.md", "a.mdx", false},
		{"README.md", "sub/README.md", true},
	} {
		if got, err := Glob(row.pattern, row.name); err != nil || got != row.want {
			t.Errorf("Glob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
}
