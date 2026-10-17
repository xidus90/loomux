package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Every row's pattern must match inside its own fixture: a row that claims a
// detection nobody can reproduce is worse than no row.
func TestEveryFrameworkRowHasAFixtureItFinds(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		dir := filepath.Join("testdata", "frameworks", f.Stack, f.Name)
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("%s/%s: no fixture at %s", f.Stack, f.Name, dir)
			continue
		}
		if f.Found == "" {
			// Nothing to find: the fixture of a row without a pattern only
			// shows what a project driving that framework looks like.
			continue
		}
		if !HasTests(dir, []string{f.Found}) {
			t.Errorf("%s/%s: %q matches nothing under %s", f.Stack, f.Name, f.Found, dir)
		}
	}
}

// Two frameworks of one kind in one stack must be told apart, or a project
// carrying one would be reported as carrying both. Different kinds may overlap.
func TestFrameworkRowsOfOneKindDoNotMatchEachOthersFixtures(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		dir := filepath.Join("testdata", "frameworks", f.Stack, f.Name)
		for _, other := range fs {
			// A row without a pattern matches nothing anywhere, so it can
			// neither be confused with a sibling nor confuse one.
			if other.Found == "" || other.Name == f.Name || other.Stack != f.Stack || other.Kind != f.Kind {
				continue
			}
			if HasTests(dir, []string{other.Found}) {
				t.Errorf("%s/%s: pattern %q also matches the fixture of %s (%s)", other.Stack, other.Name, other.Found, f.Name, dir)
			}
		}
	}
}

// The lint lane runs on any project of its stack, so the two gdtoolkit rows
// name no pattern. That exception belongs to lint alone: a third row without
// one has to be argued for, not merged.
func TestOnlyTheTwoLintRowsCarryNoPattern(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	var bare []string
	for _, f := range fs {
		if f.Found == "" {
			bare = append(bare, f.Stack+"/"+f.Name)
		}
	}
	slices.Sort(bare)
	if want := []string{"gdscript/gdtoolkit3", "gdscript/gdtoolkit4"}; !slices.Equal(bare, want) {
		t.Errorf("rows without a pattern are %v, not %v", bare, want)
	}
}

// A lane detects a framework exactly where the matrix calls the row proven: a
// proven row's pattern has to be the lane's own, or the data and the pages
// describe a detection loomux does not perform, and an unproven row's pattern
// must be in no lane, or the two files and both pages stop being true the
// moment someone wires one up without running it.
func TestALanesPatternsAreTheProvenRowsAndNoOthers(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.Found == "" {
			continue
		}
		lane := p.Stacks[f.Stack].Lanes[f.Kind]
		if wired := slices.Contains(lane.WhenFound, f.Found); wired != f.Proven {
			t.Errorf("%s/%s: proven = %v, but [stack.%s.%s].when_found %v carrying %q = %v",
				f.Stack, f.Name, f.Proven, f.Stack, f.Kind, lane.WhenFound, f.Found, wired)
		}
	}
}

// The one column that legitimately differs between the pages is "carries":
// the German page translates it.
var carriesDE = map[string]string{
	"Godot 4.3 to 4.8": "Godot 4.3 bis 4.8",
	"GUT 7.x carries Godot 3, 9.x carries Godot 4; the pattern does not tell them apart": "GUT 7.x trägt Godot 3, 9.x trägt Godot 4; das Muster unterscheidet sie nicht",
	"documented for Godot 3.5, unconfirmed on 3.6":                                       "auf Godot 3.5 dokumentiert, 3.6 unbestätigt",
}

// unprovenMarkers are the suffixes a page puts after the name of an unproven
// row, in the language of that page. A slice, not a map: two pages reported in
// a different order on every run are two pages nobody can diff.
var unprovenMarkers = []struct{ lang, marker string }{
	{"en", " (unproven)"},
	{"de", " (unerprobt)"},
}

// The page and the data say the same thing, in both languages: every table row
// of the page is a data row and every data row of the stack is a table row,
// cell for cell. Prose never stands in for a row.
func TestEveryFrameworkRowIsDocumented(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	for _, stack := range stacksIn(fs) {
		for _, l := range unprovenMarkers {
			page := filepath.Join("..", "..", "docs", l.lang, "stacks", stack+".md")
			body, err := os.ReadFile(page)
			if err != nil {
				t.Errorf("%s: every stack of the matrix needs a page: %v", page, err)
				continue
			}
			checkFrameworkPage(t, page, string(body), l.marker, l.lang, stack, fs)
		}
	}
}

// checkFrameworkPage holds one page against the rows of one stack.
func checkFrameworkPage(t *testing.T, page, body, marker, lang, stack string, fs []Framework) {
	t.Helper()
	parsed, complaints := frameworkRowsIn(page, body, marker)
	for _, c := range complaints {
		t.Error(c)
	}
	rows := map[string]pageRow{}
	for _, r := range parsed {
		if _, twice := rows[r.name]; twice {
			t.Errorf("%s: %s has two table rows", page, r.name)
		}
		rows[r.name] = r
	}
	bullets := bulletNames(body, marker)
	want := map[string]bool{}
	for _, f := range fs {
		if f.Stack != stack {
			continue
		}
		want[f.Name] = true
		if !f.Proven && !slices.Contains(bullets, f.Name) {
			t.Errorf("%s: %s is unproven and no bullet says what is still open", page, f.Name)
		}
		r, ok := rows[f.Name]
		if !ok {
			t.Errorf("%s: %s has no table row", page, f.Name)
			continue
		}
		carries := f.Carries
		if tr, ok := carriesDE[carries]; ok && lang == "de" {
			carries = tr
		}
		if r.unproven == f.Proven {
			t.Errorf("%s: %s unproven marker is %v, data say proven = %v", page, f.Name, r.unproven, f.Proven)
		}
		for col, got := range map[string][2]string{
			"kind":    {r.kind, f.Kind},
			"found":   {r.found, f.Found},
			"carries": {r.carries, carries},
			"source":  {r.source, f.Source},
		} {
			if got[0] != got[1] {
				t.Errorf("%s: %s %s is %q, data say %q", page, f.Name, col, got[0], got[1])
			}
		}
	}
	for name := range rows {
		if !want[name] {
			t.Errorf("%s names %s, which the matrix does not have for %s", page, name, stack)
		}
	}
	for _, name := range bullets {
		if !want[name] {
			t.Errorf("%s keeps open work for %s, which the matrix does not have for %s", page, name, stack)
		}
	}
}

// stacksIn gives the distinct stacks of the matrix, in the order the data name
// them.
func stacksIn(fs []Framework) []string {
	var out []string
	for _, f := range fs {
		if !slices.Contains(out, f.Stack) {
			out = append(out, f.Stack)
		}
	}
	return out
}

type pageRow struct {
	name, kind, found, carries, source string
	unproven                           bool
}

// frameworkRowsIn returns the body rows of the page's framework table beside
// the complaints of everything it could not read. A header is the row above a
// separator, and the framework table is the one whose header opens with
// "Framework", so another table on the page stays untouched.
func frameworkRowsIn(page, text, marker string) ([]pageRow, []string) {
	var rows []pageRow
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
			header, body = nil, header[0] == "Framework"
		case !body:
			header = cells
		case len(cells) != 5:
			// An exact count, so a sixth cell cannot ride along unread.
			complaints = append(complaints, fmt.Sprintf("%s: %d cells, but the framework table has five columns", at, len(cells)))
		default:
			name, unproven := strings.CutSuffix(cells[0], marker)
			rows = append(rows, pageRow{
				name: name, unproven: unproven,
				kind: cells[1], found: strings.Trim(cells[2], "`"),
				carries: cells[3], source: cells[4],
			})
		}
	}
	return rows, complaints
}

// pageCells splits one table line into its trimmed cells.
func pageCells(line string) []string {
	cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// bulletNames gives the rows the page's list of open work names: a bullet
// carries the marker the table puts after an unproven name.
func bulletNames(text, marker string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "- ")
		if !ok {
			continue
		}
		if name, _, ok := strings.Cut(rest, marker+":"); ok {
			out = append(out, name)
		}
	}
	return out
}
