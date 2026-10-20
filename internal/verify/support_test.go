package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// supportKinds are the columns of the support page that the presets decide,
// in page order after Stack and the detection column.
var supportKinds = []string{"lint", "types", "test", "coverage", "graph", "rules"}

// The support page lists every stack a project may configure, except the
// ones that check no code of a language (wiki, wiki-sources, project), and
// marks a kind as missing exactly where the presets bring no lane for it: a
// page that promised a lane the binary does not ship, or hid one it does,
// would answer the one question it exists for wrongly.
func TestTheSupportPageMatchesThePresets(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, s := range StackNames() {
		if !slices.Contains([]string{"wiki", "wiki-sources", "project"}, s) {
			want = append(want, s)
		}
	}
	for _, lang := range []string{"en", "de"} {
		page := filepath.Join("..", "..", "docs", lang, "support.md")
		body, err := os.ReadFile(page)
		if err != nil {
			t.Errorf("%s: %v", page, err)
			continue
		}
		rows, complaints := supportRowsIn(page, string(body))
		for _, c := range complaints {
			t.Error(c)
		}
		var got []string
		for _, r := range rows {
			got = append(got, r[0])
			lanes := p.Stacks[r[0]].Lanes
			for i, kind := range supportKinds {
				l, ok := lanes[kind]
				ships := ok && (len(l.Commands) > 0 || len(l.OnFile) > 0)
				if cell := r[2+i]; (cell == "—") == ships {
					t.Errorf("%s: %s %s is %q, but the presets %s", page, r[0], kind, cell, map[bool]string{true: "ship a lane", false: "ship none"}[ships])
				}
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: stacks %v, want %v in this order", page, got, want)
		}
	}
}

func TestTheSupportPageReaderNamesWhatItCannotRead(t *testing.T) {
	for name, page := range map[string]struct{ text, want string }{
		"no header":         {"|---|\n", "p.md:1: a separator row with nothing above it to be its header"},
		"short row":         {"| Stack | lint |\n|---|---|\n| go | a |\n", "p.md:3: 2 cells, but the support table has 9 columns"},
		"other table":       {"| Lane | Kind |\n|---|---|\n| lint | lint |\n", ""},
		"other stack table": {"| Stack | Runtime |\n|---|---|\n| go | go |\n", ""},
	} {
		rows, complaints := supportRowsIn("p.md", page.text)
		if page.want == "" {
			if len(rows) != 0 || len(complaints) != 0 {
				t.Errorf("%s: %v %v", name, rows, complaints)
			}
			continue
		}
		if len(complaints) != 1 || complaints[0] != page.want {
			t.Errorf("%s: %v", name, complaints)
		}
	}
}

// supportRowsIn returns the body rows of the page's support table, the one
// whose header opens with "Stack" and names the lint column (the
// verified-combinations table opens with "Stack" too), beside the complaints
// of everything it could not read.
func supportRowsIn(page, text string) ([][]string, []string) {
	var rows [][]string
	var complaints []string
	var header []string
	body := false
	for i, line := range strings.Split(text, "\n") {
		at := fmt.Sprintf("%s:%d", page, i+1)
		if !strings.HasPrefix(line, "|") {
			header, body = nil, false
			continue
		}
		cells := pageCells(line)
		switch {
		case strings.HasPrefix(cells[0], "---"):
			if header == nil {
				complaints = append(complaints, at+": a separator row with nothing above it to be its header")
				continue
			}
			header, body = nil, header[0] == "Stack" && slices.Contains(header, "lint")
		case !body:
			header = cells
		case len(cells) != 3+len(supportKinds):
			complaints = append(complaints, fmt.Sprintf("%s: %d cells, but the support table has %d columns", at, len(cells), 3+len(supportKinds)))
		default:
			rows = append(rows, cells)
		}
	}
	return rows, complaints
}
