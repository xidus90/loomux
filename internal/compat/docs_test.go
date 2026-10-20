package compat

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/verify"
)

// cell renders a list of values as one table cell; "—" means none.
func cell(parts []string) string {
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, ", ")
}

// rowOf is the line the support page carries for an entry.
func rowOf(e Entry) string {
	var addons, tools []string
	for _, a := range e.Addons {
		addons = append(addons, a.Name+" "+a.Version)
	}
	for name, v := range e.Tools {
		tools = append(tools, name+" "+v)
	}
	slices.Sort(tools)
	return fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |",
		e.Stack, e.Runtime, e.Version, e.Fixture, cell(addons), cell(tools), cell(e.Kinds))
}

// tableRows returns the data rows of the table under the heading, without
// its header and separator lines.
func tableRows(body, heading string) []string {
	var rows []string
	in, lines := false, 0
	for _, l := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "## "):
			in, lines = l == heading, 0
		case in && strings.HasPrefix(l, "|"):
			if lines++; lines > 2 {
				rows = append(rows, l)
			}
		}
	}
	return rows
}

// The support page lists exactly the entries of the list, cell for cell, in
// both languages and in the order of the list.
func TestTheSupportPageListsEveryEntry(t *testing.T) {
	es, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, e := range es {
		want = append(want, rowOf(e))
	}
	for lang, heading := range map[string]string{"en": "## Verified combinations", "de": "## Geprüfte Kombinationen"} {
		page := filepath.Join("..", "..", "docs", lang, "support.md")
		body, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		if got := tableRows(string(body), heading); !slices.Equal(got, want) {
			t.Errorf("%s: table under %q is\n%s\nwant\n%s", page, heading, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
}

// A framework row is proven exactly when the list runs it: an entry carries
// its addon, or (for gdtoolkitN) pins the gdtoolkit tool at major N.
func TestProvenFollowsTheList(t *testing.T) {
	es, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	fs, err := verify.LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		ran := false
		for _, e := range es {
			for _, a := range e.Addons {
				ran = ran || a.Name == f.Name
			}
			if major, ok := strings.CutPrefix(f.Name, "gdtoolkit"); ok {
				ran = ran || strings.HasPrefix(e.Tools["gdtoolkit"], major+".")
			}
		}
		if ran != f.Proven {
			t.Errorf("%s: proven = %v, but the list runs it = %v", f.Name, f.Proven, ran)
		}
	}
}
