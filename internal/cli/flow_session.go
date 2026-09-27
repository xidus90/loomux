package cli

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/model"
	"github.com/xidus90/loomux/internal/flow/runner"
	"github.com/xidus90/loomux/internal/flow/runs"
)

// flowSession is one run as the runner needs it: where it lives, the graph it
// walks, where that graph came from and what it was started with.
type flowSession struct {
	root     string
	id       string
	graph    *flow.Graph
	catalog  *flow.Catalog
	agent    config.Agent
	found    load.Found
	params   flow.Params
	baseline *flow.Baseline
	model    model.Model
}

// flowProject is what every command reads before a flow: the [agent] and
// [flow] tables, and the blocks this build knows.
type flowProject struct {
	agent    config.Agent
	settings config.FlowSettings
	catalog  *flow.Catalog
}

// readFlowProject reads the project at root for a flow command.
func readFlowProject(root string, deps flowDeps) (flowProject, error) {
	agent, err := config.ReadAgent(root)
	if err != nil {
		return flowProject{}, err
	}
	settings, err := config.ReadFlowSettings(root)
	if err != nil {
		return flowProject{}, err
	}
	catalog, err := flow.NewCatalog(deps.Blocks, nil)
	if err != nil {
		return flowProject{}, err
	}
	return flowProject{agent: agent, settings: settings, catalog: catalog}, nil
}

// openFlow finds a flow and loads it, for a command that has nothing to check
// in between.
func openFlow(root, name string, options map[string]string, deps flowDeps) (flowSession, error) {
	project, found, err := findFlow(root, name, deps)
	if err != nil {
		return flowSession{}, err
	}
	return project.loadFlow(root, found, options)
}

// findFlow reads the project and finds a flow by name -- [flow] default when
// the name is empty -- without loading it. What Find ignored of the project
// is said on stderr: the command goes on with the bundled flow.
func findFlow(root, name string, deps flowDeps) (flowProject, load.Found, error) {
	project, err := readFlowProject(root, deps)
	if err != nil {
		return flowProject{}, load.Found{}, err
	}
	if name == "" {
		if name, err = flowDefault(root, project.settings, deps.Bundled); err != nil {
			return flowProject{}, load.Found{}, err
		}
	}
	found, err := load.Find(root, deps.Bundled, project.settings, name)
	if err != nil {
		return flowProject{}, load.Found{}, err
	}
	for _, warning := range found.Warnings {
		flowWarn(deps, warning)
	}
	return project, found, nil
}

// loadFlow loads a found flow and lays the options a run was started with
// over its parameters.
func (p flowProject) loadFlow(root string, found load.Found, options map[string]string) (flowSession, error) {
	graph, err := load.Load(found, p.catalog)
	if err != nil {
		return flowSession{}, err
	}
	params, err := load.Params(graph, options)
	if err != nil {
		return flowSession{}, err
	}
	return flowSession{root: root, graph: graph, catalog: p.catalog, agent: p.agent, found: found, params: params}, nil
}

// flowDefault is the flow `loomux flow run` starts without a name. A default
// that names no flow is load.MissingDefault's refusal: Find's "no flow named"
// would point at a name nobody typed.
func flowDefault(root string, settings config.FlowSettings, bundled fs.FS) (string, error) {
	if settings.Default == "" {
		return "", fmt.Errorf("no flow named and [flow] default is unset; known flows: %s", flowNames(load.Names(root, bundled)))
	}
	return settings.Default, load.MissingDefault(root, bundled, settings)
}

// needsBaseline says whether a node of the flow measures against the commit a
// run starts from.
func (s flowSession) needsBaseline() bool {
	for _, node := range s.graph.Nodes {
		// The load has found a block for every kind.
		block, _ := s.catalog.Block(node.Kind)
		if block.NeedsBaseline() {
			return true
		}
	}
	return false
}

// resolve is the model an agent node of this flow runs on. config.ReadAgent
// refuses a binding or a default that names no model, so the role chain
// resolves for every node.
func (s flowSession) resolve(node flow.Node) model.Resolved {
	resolved, _ := model.Resolve(s.agent, node.Role, s.graph.Role)
	return resolved
}

// modelFor finds the model the flow's agent nodes ask, before this command
// writes anything: a provider without an adapter is a refusal now rather than
// a failed node in the journal. A flow without agent nodes needs none. A run
// holds one model in its Env, so a flow asks one provider.
func (s flowSession) modelFor(deps flowDeps) (model.Model, error) {
	var providers []string
	for _, node := range s.graph.Nodes {
		if node.Kind != "agent" {
			continue
		}
		if provider := s.resolve(node).Provider; !slices.Contains(providers, provider) {
			providers = append(providers, provider)
		}
	}
	switch len(providers) {
	case 0:
		return nil, nil
	case 1:
		return deps.Models(providers[0])
	default:
		slices.Sort(providers)
		return nil, fmt.Errorf("flow %q asks %s; a run holds one model, so a flow asks one provider",
			s.graph.Name, strings.Join(providers, " and "))
	}
}

// walker is the runner over this session's journal.
func (s flowSession) walker(deps flowDeps, replay bool) *runner.Runner {
	return runner.New(runner.Options{
		Graph:   s.graph,
		Catalog: s.catalog,
		Journal: flowJournal{path: runs.JournalPath(s.root, s.id)},
		Env: flow.Env{
			Root:     s.root,
			Params:   s.params,
			Baseline: s.baseline,
			Model:    s.model,
			Agent:    s.agent,
		},
		Clock:  deps.Clock,
		Warn:   func(text string) { flowWarn(deps, text) },
		Replay: replay,
	})
}

// flowFinish prints how a run ended, with the flow it ran and where that came
// from, and turns it into the exit code: 0 done, 3 paused, a block's own
// code, and 1 for every other failure.
func flowFinish(deps flowDeps, s flowSession, result runner.Result, err error) int {
	if err != nil {
		return flowRefuse(deps, err)
	}
	fmt.Fprintf(deps.Stdout, "run %s (%s, %s): %s\n", s.id, s.found.Name, flowOrigin(s.found), result.Status)
	for _, text := range []string{result.Question, result.Detail} {
		if text != "" {
			// A question read from a file ends in a newline of its own;
			// flow.ReadText has dropped the carriage returns before it.
			fmt.Fprintln(deps.Stdout, strings.TrimRight(text, "\n"))
		}
	}
	switch {
	case result.Status == "paused":
		return flowExitPaused
	case result.Status == "done":
		return flowExitOK
	case result.ExitCode != nil:
		return *result.ExitCode
	default:
		return flowExitFailed
	}
}
