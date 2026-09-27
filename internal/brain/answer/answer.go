// Package answer holds the brain answer itself, apart from the command line
// that used to own it: `loomux serve` answers the same five commands, and a
// package under internal/cli would close an import cycle around it.
package answer

import (
	"fmt"
	"io"
	"strings"
	"time"

	braincatalog "github.com/xidus90/loomux/internal/brain/catalog"
	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/reader"
	brainsearch "github.com/xidus90/loomux/internal/brain/search"
	brainstatus "github.com/xidus90/loomux/internal/brain/status"
	"github.com/xidus90/loomux/internal/config"
)

// Request is one command and its arguments. Query carries the single
// positional: the search query for search, the relative path for read and
// neighbors. They never occur together, so there is no second field for it --
// do not add a Relative.
type Request struct {
	Command string // "search", "catalog", "read", "neighbors", "status"
	Query   string // search: the query; read/neighbors: the relative path
	Scope   string
	Profile string
	Count   int
	Section string
	Channel privacy.Channel
}

// Ports are the engines an answer asks and the clock it judges a reconcile
// stamp against. They are an argument, not package state: every caller that
// replaces one does so for the length of one call, and two callers in one
// process -- the command line and a serve listener -- never see each other's.
type Ports struct {
	// Search is the engine a search asks. notice hears it when this call had
	// to start the daemon first.
	Search func(notice func(string)) brainsearch.SearchPort
	// Status is the engine a status asks.
	Status func() brainsearch.SearchPort
	// Now is the clock a reconcile stamp is judged against.
	Now func() time.Time
}

// DefaultPorts are the engines of the running system as the command line
// wants them: the machine's backbone from the state directory of this run,
// nothing else said about the qmd port beyond its own defaults.
func DefaultPorts() Ports {
	return DefaultPortsWith(config.StateDir())
}

// DefaultPortsWith are those engines with the qmd port built to the caller's
// order: a search asks the warm qmd daemon over MCP, a status the qmd command
// line, because the daemon has no `ls` and counts its backlog with another
// model.
//
// The options are there for the caller that is not the command line. serve
// knows its own state directory, and it hands in the lock of that directory
// and its own attempt count; left to the global default, a service on a state
// directory of its own would lock another file than the one it means and start
// a second daemon beside the running one.
//
// Both engines run on the [search] backbone of stateDir's config.toml, read
// each time a port is built: a daemon that died under a long-lived serve comes
// back on the backbone the file says now. A file that does not read makes
// either port Unavailable, so only what asks the engine fails.
//
// A function rather than a variable: nothing may hand out a shared value that
// one caller can write into, and building three closures costs nothing.
func DefaultPortsWith(stateDir string, opts ...brainsearch.QmdMcpOption) Ports {
	return Ports{
		Search: func(notice func(string)) brainsearch.SearchPort {
			backbone, err := brainsearch.ConfiguredBackbone(stateDir)
			if err != nil {
				return brainsearch.Unavailable(err)
			}
			return brainsearch.NewQmdMcpPort(append([]brainsearch.QmdMcpOption{
				brainsearch.WithNotice(notice),
				brainsearch.WithBackbone(backbone),
			}, opts...)...)
		},
		Status: func() brainsearch.SearchPort {
			backbone, err := brainsearch.ConfiguredBackbone(stateDir)
			if err != nil {
				return brainsearch.Unavailable(err)
			}
			return &brainsearch.QmdPort{Executable: "qmd", Backbone: backbone}
		},
		Now: time.Now,
	}
}

// Run answers one command: the text for the reader, the search findings as
// notes, or the error.
//
// notice hears the warming hint when the search had to start the engine. It is
// not a note: notes come back with the answer, the hint happens while
// connecting, and a caller that has no stderr -- serve -- turns it into a
// progress notification instead.
//
// The registry is loomux's; artefacts of read-only areas and the stamp stay in
// ultra-brain's state directory until stage 3, which is why both directories
// are named.
func Run(req Request, registryDir, fallbackDir string, notice func(string)) (string, []string, error) {
	return RunWith(DefaultPorts(), req, registryDir, fallbackDir, notice)
}

// RunFor is Run for a caller that has something to say about the qmd port: it
// gives back an answer function of the shape serve keeps, with its state
// directory and the options baked in. The ports are built per call, as Run
// builds them, so that two answers never share a session.
func RunFor(stateDir string, opts ...brainsearch.QmdMcpOption) func(Request, string, string, func(string)) (string, []string, error) {
	return func(req Request, registryDir, fallbackDir string, notice func(string)) (string, []string, error) {
		return RunWith(DefaultPortsWith(stateDir, opts...), req, registryDir, fallbackDir, notice)
	}
}

// RunWith is Run against engines of the caller's choosing. It is what a test of
// either caller uses; production calls Run and takes the default ports.
func RunWith(ports Ports, req Request, registryDir, fallbackDir string, notice func(string)) (string, []string, error) {
	if notice == nil {
		notice = func(string) {}
	}
	switch req.Command {
	case "search":
		return search(ports, req, registryDir, fallbackDir, notice)
	case "catalog":
		text, err := catalog(req.Scope, req.Channel, registryDir, fallbackDir)
		return text, nil, err
	case "read":
		text, err := read(req.Query, req.Scope, req.Section, req.Channel, registryDir, fallbackDir)
		return text, nil, err
	case "neighbors":
		text, err := neighbors(req.Query, req.Scope, req.Channel, registryDir, fallbackDir)
		return text, nil, err
	case "status":
		text, err := status(ports, req.Channel, registryDir, fallbackDir)
		return text, nil, err
	}
	return "", nil, fmt.Errorf("unknown command: %s", req.Command)
}

// search is core.search plus _print_search: the hits for stdout, the findings
// as notes.
func search(ports Ports, req Request, registryDir, fallbackDir string, notice func(string)) (string, []string, error) {
	port := ports.Search(notice)
	if closer, ok := port.(io.Closer); ok {
		// Let go of the session whatever the answer was; the answer does not
		// depend on how letting go went.
		defer closer.Close()
	}
	found, err := brainsearch.ExecuteSearch(req.Query, req.Scope, brainsearch.Profile(req.Profile),
		req.Count, req.Channel, port, registryDir, fallbackDir, ports.Now())
	if err != nil {
		return "", nil, err
	}
	return brainsearch.FormatSearch(found), found.Findings, nil
}

// catalog is core.catalog: the root catalog of the visible areas, or the
// index.md of one of them.
func catalog(scope string, channel privacy.Channel, registryDir, fallbackDir string) (string, error) {
	areas, err := privacy.VisibleAreas(registryDir, fallbackDir, "all", channel)
	if err != nil {
		return "", err
	}
	if scope == "all" {
		plain := make([]config.Area, len(areas))
		for i, visible := range areas {
			plain[i] = visible.Area
		}
		return braincatalog.RenderRootCatalog(plain), nil
	}
	area, err := privacy.Single(areas, scope)
	if err != nil {
		return "", err
	}
	return braincatalog.ReadAreaCatalog(area.Area, registryDir, fallbackDir)
}

// area is the one visible area a read or a neighbour query names; every area
// of the registry is checked first, as core._visible_areas("all") does.
func area(scope string, channel privacy.Channel, registryDir, fallbackDir string) (privacy.VisibleArea, error) {
	areas, err := privacy.VisibleAreas(registryDir, fallbackDir, "all", channel)
	if err != nil {
		return privacy.VisibleArea{}, err
	}
	return privacy.Single(areas, scope)
}

// read is core.read.
func read(relative, scope, section string, channel privacy.Channel, registryDir, fallbackDir string) (string, error) {
	found, err := area(scope, channel, registryDir, fallbackDir)
	if err != nil {
		return "", err
	}
	return reader.ReadDocument(found.Area, found.Manifest, relative, section, channel)
}

// neighbors is core.neighbors plus render_neighbors: containment before the
// graph is read, so a path leaving the area is refused even where no graph
// exists.
func neighbors(relative, scope string, channel privacy.Channel, registryDir, fallbackDir string) (string, error) {
	found, err := area(scope, channel, registryDir, fallbackDir)
	if err != nil {
		return "", err
	}
	inside, err := privacy.Contained(scope, relative)
	if err != nil {
		return "", err
	}
	g, err := graph.ReadGraph(found.Area, registryDir, fallbackDir)
	if err != nil {
		return "", err
	}
	incoming, outgoing := graph.Neighbors(g, inside)
	return graph.RenderNeighbors(incoming, outgoing), nil
}

// status is core.status plus _print_status: one line each.
func status(ports Ports, channel privacy.Channel, registryDir, fallbackDir string) (string, error) {
	lines, err := brainstatus.Lines(channel, ports.Status(), registryDir, fallbackDir, ports.Now())
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	return b.String(), nil
}
