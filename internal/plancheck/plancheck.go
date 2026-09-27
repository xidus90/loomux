// Package plancheck holds the migration plan to the fusion spec. The plan
// follows the spec and never contradicts it (AGENTS.md): the English and the
// German stage table list the same stages with the same state and priority,
// each state matches the spec's stage tables, each priority its table of the
// order of the open stages, and the diagram draws every stage once, in the
// class and with the priority its row has.
package plancheck

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
)

var (
	boldWord = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`\*\*([^*]+)\*\*`) })
	nodeLine = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^\s*\w+\["(\S[^"]*)"\]:::(\w+)`) })
	nodeRank = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`· P(\d+)`) })
	stageTop = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^G?\d+`) })
)

// A row is one stage of a plan's stage table.
type row struct {
	id, class, priority string
}

// A node is one stage the plan's diagram draws.
type node struct {
	id, class, rank string
}

// Check compares the English and the German plan with each other and with the
// spec and returns one line per disagreement; none means they agree.
func Check(en, de, spec string) []string {
	rowsEN, err := stages(en)
	if err != nil {
		return []string{"en: " + err.Error()}
	}
	rowsDE, err := stages(de)
	if err != nil {
		return []string{"de: " + err.Error()}
	}
	var out []string
	if len(rowsEN) != len(rowsDE) {
		out = append(out, fmt.Sprintf("en lists %d stages, de %d", len(rowsEN), len(rowsDE)))
	}
	for i := 0; i < min(len(rowsEN), len(rowsDE)); i++ {
		if rowsEN[i] != rowsDE[i] {
			out = append(out, fmt.Sprintf("row %d: en %v, de %v", i+1, rowsEN[i], rowsDE[i]))
		}
	}
	state, priority := states(spec), priorities(spec)
	for _, r := range rowsEN {
		out = append(out, againstSpec(r, state, priority)...)
	}
	out = append(out, againstDiagram("en", rowsEN, diagram(en))...)
	return append(out, againstDiagram("de", rowsDE, diagram(de))...)
}

// againstSpec holds one row to the spec: its stage and state exist there, and
// its priority is the spec's, a dash for a stage that is done or only proposed.
func againstSpec(r row, state, priority map[string]string) []string {
	var out []string
	if s, ok := state[r.id]; !ok {
		out = append(out, fmt.Sprintf("%s: the spec lists no such stage", r.id))
	} else if s != r.class {
		out = append(out, fmt.Sprintf("%s: the plan says %s, the spec %s", r.id, r.class, s))
	}
	want := "—"
	if r.class != "done" && r.class != "proposed" {
		p, ok := priority[r.id]
		if !ok {
			p, ok = priority[stageTop().FindString(r.id)]
		}
		if !ok || p == "—" {
			return append(out, fmt.Sprintf("%s: open, but the spec gives it no priority", r.id))
		}
		want = p
	}
	if r.priority != want {
		out = append(out, fmt.Sprintf("%s: priority %q, the spec says %q", r.id, r.priority, want))
	}
	return out
}

// againstDiagram holds the diagram of one language to its stage table.
func againstDiagram(lang string, rows []row, nodes []node) []string {
	drawn := map[string]string{"done": "done", "built": "partial", "open": "planned", "proposed": "planned"}
	byID := map[string]row{}
	for _, r := range rows {
		byID[r.id] = r
	}
	var out []string
	seen := map[string]int{}
	for _, n := range nodes {
		seen[n.id]++
		r, ok := byID[n.id]
		if !ok {
			out = append(out, fmt.Sprintf("%s: the diagram draws %s, which the table lacks", lang, n.id))
			continue
		}
		if drawn[r.class] != n.class {
			out = append(out, fmt.Sprintf("%s: %s is drawn %s, the table says %s", lang, n.id, n.class, r.class))
		}
		if n.rank != "" && n.rank != r.priority {
			out = append(out, fmt.Sprintf("%s: %s is drawn with P%s, the table says %s", lang, n.id, n.rank, r.priority))
		}
	}
	for _, r := range rows {
		if seen[r.id] != 1 {
			out = append(out, fmt.Sprintf("%s: %s is drawn %d times", lang, r.id, seen[r.id]))
		}
	}
	return out
}

// stages reads the stage table of a plan: its first column names the stage in
// bold, the second its state and the last its priority.
func stages(plan string) ([]row, error) {
	for _, t := range tables(plan) {
		if !headed(t[0], []string{"Stage", "Stufe"}, []string{"Status", "Stand"}) {
			continue
		}
		var rows []row
		for _, c := range t[1:] {
			words := bold(c[0])
			if len(words) == 0 {
				return nil, fmt.Errorf("row %q names no stage in bold", c[0])
			}
			priority := ""
			if f := strings.Fields(c[len(c)-1]); len(f) > 0 {
				priority = f[0]
			}
			rows = append(rows, row{id: words[0], class: classOf(c[1]), priority: priority})
		}
		return rows, nil
	}
	return nil, fmt.Errorf("no stage table")
}

// states reads the state of every stage the spec's stage tables list.
func states(spec string) map[string]string {
	out := map[string]string{}
	for _, t := range tables(spec) {
		if !headed(t[0], []string{"Stufe", "Teilstufe"}, []string{"Stand"}) {
			continue
		}
		for _, c := range t[1:] {
			if words := bold(c[0]); len(words) > 0 && len(c) > 1 {
				out[words[0]] = classOf(c[1])
			}
		}
	}
	return out
}

// priorities reads the spec's table of the order of the open stages: each
// stage named in bold in its second column has the priority of its first.
func priorities(spec string) map[string]string {
	out := map[string]string{}
	for _, t := range tables(spec) {
		if !headed(t[0], []string{"Prio"}, []string{"Stufe"}) {
			continue
		}
		for _, c := range t[1:] {
			if len(c) < 2 {
				continue
			}
			for _, w := range bold(c[1]) {
				out[w] = c[0]
			}
		}
	}
	return out
}

// diagram reads the nodes of the mermaid blocks of a plan; a node's label
// starts with its stage and may name its priority as "· P<n>".
func diagram(plan string) []node {
	var out []node
	inside := false
	for line := range strings.SplitSeq(plan, "\n") {
		switch trimmed := strings.TrimSpace(line); {
		case trimmed == "```mermaid":
			inside = true
		case trimmed == "```":
			inside = false
		case inside:
			m := nodeLine().FindStringSubmatch(line)
			if m == nil {
				continue
			}
			n := node{id: strings.Fields(m[1])[0], class: m[2]}
			if r := nodeRank().FindStringSubmatch(m[1]); r != nil {
				n.rank = r[1]
			}
			out = append(out, n)
		}
	}
	return out
}

// classOf reduces a state cell to what the two documents must agree on.
func classOf(state string) string {
	for _, c := range []struct{ prefix, class string }{
		{"✅", "done"}, {"🚧", "built"}, {"offen", "open"}, {"open", "open"},
		{"vorgeschlagen", "proposed"}, {"proposed", "proposed"}, {"➗", "split"},
	} {
		if strings.HasPrefix(state, c.prefix) {
			return c.class
		}
	}
	return "unknown"
}

// headed reports whether a table's header starts with one of first and then
// one of second.
func headed(header, first, second []string) bool {
	return len(header) > 1 && slices.Contains(first, header[0]) && slices.Contains(second, header[1])
}

// bold returns the words of a cell set in bold, in order.
func bold(cell string) []string {
	var words []string
	for _, m := range boldWord().FindAllStringSubmatch(cell, -1) {
		words = append(words, m[1])
	}
	return words
}

// tables returns every table of a markdown text as rows of cells, the header
// first and the separator line left out.
func tables(text string) [][][]string {
	var all [][][]string
	var current [][]string
	for line := range strings.SplitSeq(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			if current != nil {
				all = append(all, current)
				current = nil
			}
			continue
		}
		if c := cells(line); !strings.HasPrefix(c[0], "---") {
			current = append(current, c)
		}
	}
	if current != nil {
		all = append(all, current)
	}
	return all
}

// cells splits one table line into its trimmed cells; an escaped pipe stays
// inside its cell.
func cells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(strings.TrimSuffix(line, "|"), "|")
	parts := strings.Split(strings.ReplaceAll(line, `\|`, "\x00"), "|")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(strings.ReplaceAll(p, "\x00", `\|`))
	}
	return parts
}
