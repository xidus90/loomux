package web

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/config/target"
	"github.com/xidus90/loomux/internal/hooks"
)

// pathRuleNote stands at every path rule: the rule is a write barrier and
// nothing more.
const pathRuleNote = "blocks writes, not reads"

// moduleOffNote stands over the keys of a module that [modules] switches off.
const moduleOffNote = "module off — these keys have no effect"

// rule is one entry of a rule list and where it comes from.
type rule struct {
	What, Reason, Origin, Note string
	ReadOnly                   bool
	// Index is the position in the list of the file the page is about, what
	// a removal names; a rule of another origin has none to name.
	Index int
}

// row is one key of the settings page.
type row struct {
	ID, Doc, Value, Default, Origin string
	// Items are the elements of a list of tables that has no rule view.
	Items []string
	Rules []rule
	// Problem is why a rule list could not be shown in full.
	Problem string
	// RawOnly marks a table the form has no field for.
	RawOnly bool
	// Section is the dotted name of a list of rules or allowed lines, which
	// a form appends to; Match says its entries hold globs, not a regex.
	Section string
	Match   bool
	// Editable rows have a form: Input is the value as it is typed and
	// Choices, when there are any, the only values a select offers.
	Editable bool
	Input    string
	Choices  []string
	// Resettable rows have a line in the file that a reset removes.
	Resettable bool
}

// group is the keys of one module.
type group struct {
	Module string
	Off    bool
	Note   string
	Rows   []row
}

// settingsView is the data of the settings page.
type settingsView struct {
	ID, Title, Root string
	Missing         bool
	Query           string
	Modified        bool
	// Error is why the file could not be read; the page then offers the raw
	// text, where it can be repaired.
	Error  string
	Groups []group
	// Unknown are the keys of the file that no row above describes; the raw
	// text is where they are read and changed.
	Unknown []string
	// Notice is why the last action on this page did not happen.
	Notice    string
	Proposals []proposalRow
	// Registered and LLMConfig are the llm_config switch of a project.
	Registered, LLMConfig bool
}

// moduleOrder is the order the page lists the modules in.
var moduleOrder = []schema.Module{schema.Base, schema.Hooks, schema.Brain, schema.Graph}

// settings shows the keys of one target: a project, or the machine.
func (s *server) settings(w http.ResponseWriter, r *http.Request) {
	s.showSettings(w, r, nil)
}

// showSettings is the settings page. problem is why an action that ends
// here did not happen.
func (s *server) showSettings(w http.ResponseWriter, r *http.Request, problem error) {
	id := r.PathValue("id")
	tgt, project, ok := s.target(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	v := settingsView{
		ID:         id,
		Title:      "Global settings",
		Missing:    project.Missing,
		Query:      r.URL.Query().Get("q"),
		Modified:   r.URL.Query().Get("modified") == "1",
		Registered: project.Registered,
		LLMConfig:  project.LLMConfig,
	}
	if !tgt.Global {
		v.Title, v.Root = project.Name, project.Root
	}
	text, err := tgt.Read()
	var entries []schema.Entry
	if err == nil {
		entries, err = tgt.Entries(text)
	}
	// A file that cannot be read gets the raw text to repair it; proposals
	// are shown against a file that can.
	if err != nil {
		v.Error = err.Error()
	} else {
		var modulesProblem error
		v.Groups, modulesProblem = s.groups(tgt, text, entries, v)
		// The entries parsed, so the text is valid TOML and this cannot fail.
		unknown, _ := tgt.Unknown(text)
		v.Unknown = slices.DeleteFunc(unknown, func(id string) bool { return !v.keepsText(id) })
		views, err := tgt.Proposals()
		problem = errors.Join(problem, modulesProblem, err)
		v.Proposals = proposalRows(views)
	}
	if problem != nil {
		v.Notice = problem.Error()
	}
	s.page(w, r, "settings", v)
}

// groups lays the entries that pass the filters out by module. A [modules]
// that does not read leaves every module shown as on, and the error comes
// back to be said on the page: the file is valid TOML, but the page cannot
// tell which modules it switches off.
func (s *server) groups(tgt target.Target, text string, entries []schema.Entry, v settingsView) ([]group, error) {
	modules := config.AllModules()
	var problem error
	if !tgt.Global {
		read, err := config.ReadModules(v.Root)
		if err == nil {
			modules = read
		} else {
			problem = fmt.Errorf("every module is shown as on: %w", err)
		}
	}
	off := map[schema.Module]bool{schema.Hooks: !modules.Hooks, schema.Brain: !modules.Brain, schema.Graph: !modules.Graph}
	var out []group
	for _, module := range moduleOrder {
		g := group{Module: string(module), Off: off[module]}
		if g.Off {
			g.Note = moduleOffNote
		}
		for _, e := range entries {
			if e.Key.Module == module && v.keeps(e) {
				g.Rows = append(g.Rows, s.rowOf(tgt, text, e))
			}
		}
		if len(g.Rows) > 0 {
			out = append(out, g)
		}
	}
	return out, problem
}

// keeps says whether the filters of the page let an entry through.
func (v settingsView) keeps(e schema.Entry) bool {
	if v.Modified && e.Origin != schema.Set {
		return false
	}
	q := strings.ToLower(v.Query)
	return strings.Contains(strings.ToLower(e.Key.ID()), q) || strings.Contains(strings.ToLower(e.Key.Doc), q)
}

// keepsText says whether the search lets a key the schema does not know
// through. The file holds it, so "only modified" never hides it.
func (v settingsView) keepsText(id string) bool {
	return strings.Contains(strings.ToLower(id), strings.ToLower(v.Query))
}

// shownValue is what a human reads for an entry. A preset row has no value
// of its own; the schema default mirrors what the presets fill in, and an
// empty column would read as "unset".
func shownValue(e schema.Entry) string {
	if e.Origin == schema.Preset {
		return e.Key.Default
	}
	return e.Value
}

// rowOf is one entry as the page shows it.
func (s *server) rowOf(tgt target.Target, text string, e schema.Entry) row {
	k := e.Key
	r := row{ID: k.ID(), Doc: k.Doc, Default: k.Default, Origin: string(e.Origin)}
	if k.Kind == schema.TableList {
		r.Section, r.Match = k.Section, k.Section == "policy.paths.rules"
	}
	switch {
	case k.Kind == schema.TableList && strings.HasPrefix(k.Section, "policy."):
		var err error
		if r.Rules, err = s.rules(tgt, text, k.Section == "policy.paths.rules"); err != nil {
			r.Problem = err.Error()
		}
	case k.Kind == schema.TableList:
		// Each block is one line of Value.
		if e.Count > 0 {
			r.Items = strings.Split(e.Value, "\n")
		}
	case k.Kind == schema.Table:
		r.RawOnly = true
		r.Value = shownValue(e)
	default:
		r.Value = shownValue(e)
		// A family nobody named yet has no key to write: its id carries
		// the star.
		r.Editable = !k.Named()
		r.Input = e.Input
		r.Resettable = e.Origin == schema.Set
		switch k.Kind {
		case schema.Enum:
			r.Choices = k.Choices
		case schema.Bool:
			r.Choices = []string{"true", "false"}
		}
	}
	return r
}

// source is the rules one origin contributes.
type source struct {
	origin   string
	policy   config.Policy
	readOnly bool
}

// rules lists a rule list in the order the guard applies it: built-in, then
// the machine's, then the project's. What a project page does not own is
// read-only. A source that does not read costs its own rules only: the error
// comes back with the rules of the others, so that one broken file does not
// hide the rest of what the guard applies.
func (s *server) rules(tgt target.Target, text string, paths bool) ([]rule, error) {
	sources := []source{{"built-in", config.Policy{Paths: hooks.BuiltinPathRules(), Commands: hooks.BuiltinCommandRules()}, true}}
	own := "global"
	var problem error
	if !tgt.Global {
		global, err := config.ReadGlobalPolicy(s.options.StateDir)
		if err == nil {
			sources = append(sources, source{"global", global, true})
		}
		problem = err
		own = "project"
	}
	mine, err := config.ParsePolicy(tgt.Path, []byte(text))
	if err == nil {
		sources = append(sources, source{own, mine, false})
	}
	problem = errors.Join(problem, err)

	var out []rule
	for _, src := range sources {
		if paths {
			for i, p := range src.policy.Paths {
				out = append(out, rule{What: strings.Join(p.Match, ", "), Reason: p.Reason, Origin: src.origin, Note: pathRuleNote, ReadOnly: src.readOnly, Index: i})
			}
			continue
		}
		for i, c := range src.policy.Commands {
			out = append(out, rule{What: c.Source, Reason: c.Reason, Origin: src.origin, ReadOnly: src.readOnly, Index: i})
		}
	}
	return out, problem
}
