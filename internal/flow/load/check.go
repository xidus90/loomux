package load

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/expr"
	"github.com/xidus90/loomux/internal/flow/tmpl"
)

// stage is one of the checks after the declarations. They share a signature so
// that Load can walk them in order and stop at the first one with findings.
type stage func(file string, made *draft, catalog *flow.Catalog) Findings

// stage2 asks the catalog for every node kind.
func stage2(file string, made *draft, catalog *flow.Catalog) Findings {
	var found Findings
	for _, node := range made.graph.Nodes {
		// command is in the format and not in this build. Named apart, its
		// author reads that the kind is known and waiting, not misspelled.
		if node.Kind == "command" {
			found.add(file, "node %q has kind %q; command nodes are planned but not built", node.Name, node.Kind)
			continue
		}
		block, ok := catalog.Block(node.Kind)
		if !ok {
			found.add(file, "node %q has kind %q; known kinds: %s",
				node.Name, node.Kind, strings.Join(catalog.Kinds(), ", "))
			continue
		}
		for _, problem := range block.Check(node) {
			found.add(file, "node %q: %s", node.Name, problem)
		}
	}
	return found
}

// textFolders is where each kind of text lives. A fixed place per kind is
// what lets a project overlay a bundled flow's text file by file.
var textFolders = map[string]string{"instruction": "instructions/", "question": "questions/"}

// textPath says what is wrong with a text's path, or "" when nothing is.
func textPath(t flow.Text) string {
	if strings.Contains(t.Path, `\`) {
		return fmt.Sprintf("%s %q must use / between folders", t.Key, t.Path)
	}
	folder, known := textFolders[t.Key]
	if !known {
		folder = "instructions/"
	}
	if !fs.ValidPath(t.Path) || !strings.HasPrefix(t.Path, folder) {
		return fmt.Sprintf("%s %q must be a file under %s", t.Key, t.Path, folder)
	}
	return ""
}

// stage3 opens every text a node renders, once its path keeps to its folder.
func stage3(file string, made *draft, catalog *flow.Catalog) Findings {
	var found Findings
	for _, node := range made.graph.Nodes {
		block, _ := catalog.Block(node.Kind) // stage 2 cleared every kind
		for _, text := range block.Texts(node) {
			if text.Path == "" {
				continue // an inline text has no file to open
			}
			if problem := textPath(text); problem != "" {
				found.add(file, "node %q: %s", node.Name, problem)
				continue
			}
			if _, err := flow.ReadText(made.graph.Texts, text); err != nil {
				found.add(file, "node %q: %v", node.Name, err)
			}
		}
	}
	return found
}

// stage4 reads every name a text, a condition or a cap uses.
//
// It is the one check that writes: a condition and a cap are parsed here, and
// parsing them is what turns them into the Condition and the Cap a run needs.
// Doing it twice would let the checked text and the executed one drift apart.
func stage4(file string, made *draft, catalog *flow.Catalog) Findings {
	graph := made.graph
	fieldTypes := types(graph.State)
	paramTypes := types(graph.Params)
	vocabulary := expr.Names{Fields: fieldTypes, Params: paramTypes, Predicates: catalog.Predicates()}
	var found Findings
	for _, node := range graph.Nodes {
		block, _ := catalog.Block(node.Kind) // stage 2 cleared every kind
		for _, text := range block.Texts(node) {
			raw, _ := flow.ReadText(graph.Texts, text) // stage 3 opened every text
			template, err := tmpl.Parse(string(raw))
			if err != nil {
				found.add(file, "node %q %s: %v", node.Name, text.Key, err)
				continue
			}
			// Existence is all a placeholder needs: tmpl shows all four types,
			// a list as one "- " line per entry, so no type can be wrong here.
			for _, name := range template.Names() {
				_, isField := fieldTypes[name]
				_, isParam := paramTypes[name]
				if !isField && !isParam {
					found.add(file, "node %q %s names %q; known fields: %s",
						node.Name, text.Key, name, list(fieldTypes))
				}
			}
		}
	}
	for i, edge := range graph.Edges {
		if edge.Text == "" {
			continue
		}
		condition, err := expr.ParseCondition(edge.Text, vocabulary)
		if err != nil {
			found.add(file, "edge %s→%s: %v", edge.From, edge.To, err)
			continue
		}
		graph.Edges[i].When = condition
	}
	for i, node := range graph.Nodes {
		limit, err := expr.ParseCap(node.Raw["max_visits"], paramTypes)
		if err != nil {
			found.add(file, "node %q: %v", node.Name, err)
			continue
		}
		graph.Nodes[i].MaxVisits = limit
	}
	return found
}

// stage5 holds every written field against [state].
//
// That also settles the rule that no two blocks write one field with different
// types: both are held against the same declaration, so two that disagree with
// each other cannot both agree with it.
func stage5(file string, made *draft, catalog *flow.Catalog) Findings {
	var found Findings
	for _, node := range made.graph.Nodes {
		block, _ := catalog.Block(node.Kind) // stage 2 cleared every kind
		writes := block.Writes(node)
		for _, name := range sortedNames(writes) {
			field, ok := made.graph.State[name]
			if !ok {
				found.add(file, "node %q writes %q, which [state] does not declare", node.Name, name)
				continue
			}
			if writes[name] != field.Type {
				found.add(file, "node %q writes %q as %s, but [state] declares it as %s",
					node.Name, name, writes[name], field.Type)
			}
		}
	}
	return found
}

// stage6 is the five graph rules, and it stops inside itself.
//
// Rules 3 to 5 walk a graph that rules 1 and 2 declared whole. Run anyway, a
// start that does not exist would report every node as unreachable on top, and
// the reader would look for a dozen mistakes instead of the one there is.
func stage6(file string, made *draft, _ *flow.Catalog) Findings {
	graph := made.graph
	byName := make(map[string]flow.Node, len(graph.Nodes))
	out := make(map[string][]flow.Edge, len(graph.Nodes))
	for _, node := range graph.Nodes {
		byName[node.Name] = node
	}
	for _, edge := range graph.Edges {
		out[edge.From] = append(out[edge.From], edge)
	}
	var found Findings
	if _, ok := byName[graph.Start]; !ok {
		found.add(file, "start node %q does not exist", graph.Start)
	}
	for _, edge := range graph.Edges {
		if _, ok := byName[edge.From]; !ok {
			found.add(file, "edge from unknown node %q", edge.From)
			continue
		}
		if _, ok := byName[edge.To]; !ok && edge.To != flow.End {
			found.add(file, "edge from %q to unknown node %q", edge.From, edge.To)
		}
	}
	if len(found) > 0 {
		return found
	}
	// Reachability before the rest: an island with no edges at all is
	// unreachable *and* has no way out, and "no outgoing edge" would send its
	// author looking for a missing edge instead of a missing path to the node.
	seen := reachable(graph.Start, out)
	var lost []string
	for name := range byName {
		if !seen[name] {
			lost = append(lost, name)
		}
	}
	if len(lost) > 0 {
		slices.Sort(lost)
		found.add(file, "unreachable node(s): %s", strings.Join(lost, ", "))
	}
	// An error edge is not a way out: a node whose only edge is the fallback
	// would run once and then have nowhere to go when it succeeded.
	for _, node := range graph.Nodes {
		if !slices.ContainsFunc(out[node.Name], func(edge flow.Edge) bool { return !edge.OnError }) {
			found.add(file, "node %q has no outgoing edge", node.Name)
		}
	}
	// A back edge is allowed, an unbounded loop is not. A cap over a parameter
	// counts as more than one visit whatever the parameter turns out to be:
	// the runner holds every cap against the resolved parameters at start.
	for _, name := range sortedNames(byName) {
		limit := byName[name].MaxVisits
		if limit.Param == "" && limit.Add <= 1 && onCycle(name, out) {
			found.add(file, "node %q sits on a cycle but allows one visit; raise its max_visits", name)
		}
	}
	return found
}

func reachable(start string, out map[string][]flow.Edge) map[string]bool {
	seen := make(map[string]bool, len(out))
	pending := []string{start}
	for len(pending) > 0 {
		name := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[name] || name == flow.End {
			continue
		}
		seen[name] = true
		for _, edge := range out[name] {
			pending = append(pending, edge.To)
		}
	}
	return seen
}

func onCycle(start string, out map[string][]flow.Edge) bool {
	seen := make(map[string]bool, len(out))
	var pending []string
	for _, edge := range out[start] {
		pending = append(pending, edge.To)
	}
	for len(pending) > 0 {
		name := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if name == start {
			return true
		}
		if seen[name] || name == flow.End {
			continue
		}
		seen[name] = true
		for _, edge := range out[name] {
			pending = append(pending, edge.To)
		}
	}
	return false
}

func types(fields map[string]flow.Field) map[string]flow.Type {
	kinds := make(map[string]flow.Type, len(fields))
	for name, field := range fields {
		kinds[name] = field.Type
	}
	return kinds
}
