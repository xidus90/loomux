package web

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"strings"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/config/target"
	"github.com/xidus90/loomux/internal/flow/runs"
	"github.com/xidus90/loomux/internal/selfupdate"
	"github.com/xidus90/loomux/internal/setup"
	"github.com/xidus90/loomux/internal/setup/hostfile"
)

// tiles are the dashboard's tiles in their fixed order; a new tile is one
// more entry here.
func tiles() []Tile {
	return []Tile{
		{ID: "projects", Title: "Projects", Render: renderProjects},
		{ID: "proposals", Title: "Config proposals", Render: renderProposals},
		{ID: "flow-runs", Title: "Waiting flow runs", Render: renderFlowRuns},
		{ID: "review-cases", Title: "Review cases", Render: renderReviewCases},
		{ID: "index", Title: "Index state", Render: renderIndex},
		{ID: "version", Title: "Version", Render: renderVersion},
	}
}

// tileHTML runs the named block of the dashboard page. The text is what
// html/template wrote, escaped for where each value stands; nothing else
// makes a template.HTML here.
func tileHTML(name string, data any) (template.HTML, error) {
	var out bytes.Buffer
	if err := pages()["dashboard"].ExecuteTemplate(&out, name, data); err != nil {
		return "", err
	}
	return template.HTML(out.String()), nil
}

// existing is the projects whose configuration file is there: the ones the
// tiles that read a project have something to read in.
func existing(projects []Project) []Project {
	var out []Project
	for _, p := range projects {
		if !p.Missing {
			out = append(out, p)
		}
	}
	return out
}

// projectRow is a project as the projects tile lists it.
type projectRow struct {
	Project
	// Hooks says whether the hook entries stand in the host files, and is
	// empty where the interface was given no way to read the project.
	Hooks string
	// Binary says what is wrong with the binary the entries call, empty when
	// nothing is.
	Binary string
	// Problem is why the project's configuration or setup could not be read.
	Problem string
}

type projectsTile struct {
	CSRF string
	Rows []projectRow
}

// renderProjects lists every project with what its row can say: the
// project whose configuration does not parse names that in its own row, and
// the others are listed as always.
func renderProjects(s *server, csrf string) (template.HTML, error) {
	projects, err := Projects(s.options.StateDir)
	if err != nil {
		return "", err
	}
	data := projectsTile{CSRF: csrf}
	for _, p := range projects {
		row := projectRow{Project: p}
		if !p.Missing {
			row.Hooks, row.Binary, row.Problem = s.hooksOf(p)
		}
		data.Rows = append(data.Rows, row)
	}
	return tileHTML("tile_projects", data)
}

// hooksOf reads the project's configuration and, with Gather, whether its
// hook entries stand.
func (s *server) hooksOf(p Project) (hooks, binary, problem string) {
	tgt := target.Project(p.Root)
	text, err := tgt.Read()
	if err == nil {
		_, err = tgt.Entries(text)
	}
	var problems []string
	if err != nil {
		problems = append(problems, err.Error())
	}
	if s.options.Gather != nil {
		facts, err := s.options.Gather(p.Root)
		if err == nil {
			var stand bool
			if stand, err = hooksStand(facts); err == nil {
				binary = binaryNote(facts)
				hooks = "missing"
				if stand {
					hooks = "set up"
				}
			}
		}
		if err != nil {
			problems = append(problems, err.Error())
		}
	}
	return hooks, binary, strings.Join(problems, "; ")
}

// binaryNote says what is wrong with the binary the hook entries call: it is
// not there, or the installed machine-wide one is older than this interface.
// A development build names no version to compare with.
func binaryNote(f setup.Facts) string {
	switch {
	case !f.BinaryThere:
		return "binary missing"
	case f.Binary == hostfile.Canonical && f.Installed != "" && selfupdate.IsVersion(f.Version) &&
		!selfupdate.AtLeast(f.Installed, f.Version):
		return "binary " + f.Installed + " is older than " + f.Version
	}
	return ""
}

// hooksStand says whether every hook file of the project's hosts already holds
// the entries init would write: merging them adds nothing. The merge is only
// asked, never written.
func hooksStand(f setup.Facts) (bool, error) {
	read := setup.ProjectReader(f.Root)
	for _, host := range f.Hosts {
		before, _, err := read(hostfile.Path(host))
		if err != nil {
			return false, err
		}
		merged, err := hostfile.Merge(host, before, hostfile.Entries(host, f.Binary))
		if err != nil {
			return false, err
		}
		if len(merged.Added) > 0 {
			return false, nil
		}
	}
	return true, nil
}

// proposalsRow is one target that has open proposals, or whose proposals could
// not be listed.
type proposalsRow struct {
	ID, Title string
	// Count is the open proposals, Unreadable those among them whose file
	// does not read.
	Count, Unreadable int
	Problem           string
}

// renderProposals counts the open proposals of every target. The tile needs
// the number, and which of them are broken, and not the diffs: resolving every
// proposal of every project on each visit of the start page would be the
// settings page's work.
func renderProposals(s *server, _ string) (template.HTML, error) {
	projects, err := Projects(s.options.StateDir)
	if err != nil {
		return "", err
	}
	rows := []proposalsRow{{ID: globalID, Title: "Global settings"}}
	targets := []target.Target{target.Global(s.options.StateDir)}
	for _, p := range existing(projects) {
		rows = append(rows, proposalsRow{ID: p.ID, Title: p.Name})
		targets = append(targets, target.Project(p.Root))
	}
	// Only a target with something to say stays in the list.
	var listed []proposalsRow
	for i, tgt := range targets {
		ids, err := tgt.IDs()
		rows[i].Count = len(ids)
		for _, id := range ids {
			if _, loadErr := tgt.Load(id); loadErr != nil {
				rows[i].Unreadable++
			}
		}
		if err != nil {
			rows[i].Problem = err.Error()
		}
		if rows[i].Count > 0 || rows[i].Problem != "" {
			listed = append(listed, rows[i])
		}
	}
	return tileHTML("tile_proposals", listed)
}

// waitingRow is one run at a gate, or one project whose runs cannot be read.
type waitingRow struct {
	ProjectID, Project string
	Run, Flow, Node    string
	Question           string
	Problem            string
}

func renderFlowRuns(s *server, _ string) (template.HTML, error) {
	projects, err := Projects(s.options.StateDir)
	if err != nil {
		return "", err
	}
	var rows []waitingRow
	for _, p := range existing(projects) {
		waiting, err := runs.WaitingRuns(p.Root)
		if err != nil {
			rows = append(rows, waitingRow{ProjectID: p.ID, Project: p.Name,
				Problem: fmt.Sprintf("%s cannot be read as a folder of runs: %v", runs.Dir, err)})
		}
		for _, w := range waiting {
			row := waitingRow{ProjectID: p.ID, Project: p.Name, Run: w.ID}
			switch {
			case w.Err != nil:
				row.Problem = w.Err.Error()
			case w.Gate != nil:
				row.Node, row.Question = w.Gate.Node, strings.TrimRight(w.Gate.Question, "\n")
				if w.Marker != nil {
					row.Flow = w.Marker.Flow
				}
			}
			rows = append(rows, row)
		}
	}
	return tileHTML("tile_flow_runs", rows)
}

type casesTile struct {
	// None says why there is nothing to list: no review centre is a state of
	// a machine, not a failure.
	None   string
	Cases  []maintenance.ListedCase
	Broken []string
}

func renderReviewCases(s *server, _ string) (template.HTML, error) {
	listed, broken, err := maintenance.ListCases(s.options.StateDir)
	switch {
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, maintenance.ErrNoReviewCentre):
		return tileHTML("tile_review_cases", casesTile{None: "No area declares a review centre."})
	case err != nil:
		return "", err
	}
	return tileHTML("tile_review_cases", casesTile{Cases: listed, Broken: broken})
}

// indexRow is one project's code graph.
type indexRow struct {
	ID, Name string
	State    string
	Problem  string
}

type indexTile struct {
	Rows []indexRow
	// Search is what `brain status` says; Problem is why it could not.
	Search, SearchProblem string
}

func renderIndex(s *server, _ string) (template.HTML, error) {
	projects, err := Projects(s.options.StateDir)
	if err != nil {
		return "", err
	}
	var data indexTile
	for _, p := range existing(projects) {
		row := indexRow{ID: p.ID, Name: p.Name}
		// Probing only reads: rebuilding a graph is not what opening a page
		// should do.
		drift, err := freshness.Probe(p.Root, all.Version())
		switch {
		case err != nil:
			row.Problem = err.Error()
		case drift == nil:
			row.State = "not built yet"
		case drift.Clean():
			row.State = "fresh"
		default:
			row.State = fmt.Sprintf("%d files changed since the last build", drift.Count())
		}
		data.Rows = append(data.Rows, row)
	}
	if s.options.SearchStatus == nil {
		data.SearchProblem = "this interface was given no way to ask the search"
	} else if data.Search, err = s.options.SearchStatus(s.options.StateDir); err != nil {
		data.SearchProblem = err.Error()
	}
	return tileHTML("tile_index", data)
}

type versionTile struct {
	Running, Channel string
	Update           string
	Problems         []string
}

func renderVersion(s *server, _ string) (template.HTML, error) {
	data := versionTile{Running: s.version, Channel: "stable"}
	beta, err := selfupdate.ReadChannel(s.options.StateDir)
	if err != nil {
		data.Problems = append(data.Problems, err.Error())
	}
	if beta {
		data.Channel = "beta"
	}
	status, err := selfupdate.ReadStatus(s.options.StateDir)
	switch {
	case err != nil:
		data.Problems = append(data.Problems, err.Error())
	case status == nil:
		data.Update = "no update check has run yet"
	case status.Result == selfupdate.Failed:
		data.Update = fmt.Sprintf("the last check, %s, failed: %s", status.CheckedAt.Format("2006-01-02 15:04"), status.Error)
	default:
		data.Update = fmt.Sprintf("the last check, %s: %s", status.CheckedAt.Format("2006-01-02 15:04"), status.Result)
		if status.Version != "" {
			data.Update += ", " + status.Version
		}
	}
	return tileHTML("tile_version", data)
}
