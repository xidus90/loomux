package benchsearch

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/index"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// Options are one run as the command line asked for it.
type Options struct {
	Scope        string // "" = default "knowledge"
	ScopeSet     bool   // --scope given
	Profile      search.Profile
	Channel      privacy.Channel
	Out          string // "" = default
	Questions    string // "" = default
	Corpus       string // "" or a stand directory
	Latency      bool
	LatencyQuery string
	Repeat       int
}

// Deps are what a run reaches outside itself.
type Deps struct {
	StateDir, FallbackDir string                                    // the real state
	Daemon                func() search.SearchPort                  // everyday: the service
	CLI                   func(index string) search.SearchPort      // corpus: QmdPort with --index
	QmdVersion            func() string                             // never fails: "unknown" instead
	Models                func(configPath string) map[string]string // index.Models
	Loomux                string
	Now                   func() time.Time // the stamp and the reconcile clock
	Clock                 func() time.Time // for timings
	Random                func() string
	Warn                  func(string)
	// Backbone is what a qmd process this run starts computes on: the user's
	// QMD_LLAMA_GPU or QMD_FORCE_CPU where one is set, else the machine's
	// [search] backbone.
	Backbone string
}

// daemonStarter is a daemon port that knows whether it started the daemon
// (search.QmdMcpPort does).
type daemonStarter interface{ StartedDaemon() bool }

// allAreas is the scope that measures every registered area.
const allAreas = "all"

// Bench runs one measurement and returns the markdown printed on success.
// Every refusal is a Problems or a plain error; the caller prints each line
// as "error: <line>" and exits 1.
//
// A corpus run registers the stand in a throwaway state and a named qmd
// index, and removes both on every return: the report is on disk by then,
// and the caller prints it afterwards. Ctrl+C runs no defer; the next corpus
// run sweeps what it left.
func Bench(o Options, d Deps) (string, error) {
	if err := refuseATwiceToldCorpus(o); err != nil {
		return "", err
	}
	// Once per run: taken again for the head, the report could name a minute
	// its file name does not.
	stamp := benchreport.Stamp(d.Now())
	c := chain{scope: o.Scope, channel: o.Channel, stateDir: d.StateDir, fallback: d.FallbackDir, now: d.Now}
	if c.scope == "" && !o.ScopeSet {
		c.scope = "knowledge"
	}
	if c.channel == "" {
		c.channel = privacy.ChannelLocal
	}
	profile := o.Profile
	if profile == "" {
		profile = search.ProfileFast
	}
	var prepared *Prepared
	if o.Corpus != "" {
		// Checked before anything is written: an unsound stand leaves no
		// throwaway state and no qmd index behind.
		stand, err := filepath.Abs(o.Corpus)
		if err == nil {
			err = CheckCorpus(stand)
		}
		if err != nil {
			return "", err
		}
		lockDir := filepath.Join(d.StateDir, "bench")
		SweepStale(lockDir, d.Warn)
		p, cleanup, err := PrepareCorpus(stand, lockDir, d.Random, d.Warn)
		defer cleanup()
		if err != nil {
			return "", err
		}
		prepared = p
		c.scope, c.stateDir, c.fallback = p.Scope, p.StateDir, ""
	}
	registered, err := config.ReadRegistry(c.stateDir)
	if err != nil {
		return "", err
	}
	areas, err := benchAreas(registered, c.scope)
	if err != nil {
		return "", err
	}
	out, err := benchOut(o.Out, areas)
	if err != nil {
		return "", err
	}
	mdPath, jsPath, err := benchreport.Targets(out, "bench-"+stamp+"-"+string(profile))
	if err != nil {
		return "", err
	}
	questionSet := o.Questions
	switch {
	case prepared != nil:
		questionSet = filepath.Join(prepared.Stand, "questions.yaml")
	case questionSet == "":
		questionSet = filepath.Join(out, "questions.yaml")
	}
	questions, err := LoadQuestions(questionSet, DefaultShape())
	if err != nil {
		return "", err
	}
	if prepared == nil && !o.ScopeSet {
		// The default scope only finds the question set; what is measured is
		// where its answers lie.
		if c.scope, err = pointedScope(registered, questions); err != nil {
			return "", err
		}
		areas = selectAreas(registered, c.scope)
	}
	portName, modelsFrom := "daemon", index.QmdConfigPath()
	if prepared != nil {
		portName, modelsFrom = "cli", index.QmdConfigPathFor(prepared.Index)
		c.port = d.CLI(prepared.Index)
		err = indexCorpus(c.port, prepared.Scope, profile)
	} else {
		c.port = d.Daemon()
	}
	if err != nil {
		return "", err
	}
	listings, err := listed(c.port, areas)
	if err != nil {
		return "", err
	}
	readScope, relative, ok := firstDocument(listings)
	if !ok {
		// An empty index answers every question with a clean miss, and the
		// report would pass for a measurement.
		return "", fmt.Errorf("the engine holds no document for %s, so there is nothing to measure "+
			"against; index the area before measuring", strings.Join(slices.Sorted(maps.Keys(areas)), ", "))
	}
	var findings []string
	outcomes, err := RunQuality(questions, c.ask(profile, areas, &findings), d.Clock)
	if err != nil {
		return "", err
	}
	var latency *Latency
	if o.Latency {
		latency, err = c.latency(readScope, relative, o.LatencyQuery, o.Repeat, d.Clock)
		if err != nil {
			return "", err
		}
	}
	env := benchreport.Current(d.Loomux)
	env.Qmd, env.Models, env.Profile, env.Port = d.QmdVersion(), d.Models(modelsFrom), string(profile), portName
	// A daemon found running keeps the backbone of whoever started it; the
	// command line, and a daemon this run started, run on the one resolved here.
	env.Backbone = "unknown"
	if starter, ok := c.port.(daemonStarter); prepared != nil || ok && starter.StartedDaemon() {
		env.Backbone = d.Backbone
	}
	run := Run{Stamp: stamp, Scope: c.scope, Profile: string(profile), Environment: env, QuestionSet: questionSet,
		Outcomes: outcomes, Findings: findings, Latency: latency}
	for _, paths := range listings {
		run.Documents += len(paths)
	}
	if prepared != nil {
		run.Corpus = prepared.Stand
	}
	text := Markdown(run)
	// Measured durations are finite, so the report always encodes.
	report, _ := Envelope(run)
	payload, _ := report.JSON()
	if err := benchreport.WriteBoth(mdPath, jsPath, []byte(text), payload); err != nil {
		return "", err
	}
	return text, nil
}

// indexCorpus builds the corpus's index. The engine has never seen the
// stand: its collection holds nothing until update and embed have run.
//
// qmd embed can exit 0 with most documents still without vectors; fast and
// full would then measure an index that is not there. keyword reads no
// vectors and skips the check, which costs a command line of its own.
func indexCorpus(port search.SearchPort, scope string, profile search.Profile) error {
	collections := []string{search.CollectionName(scope)}
	if err := port.Refresh(collections); err != nil {
		return err
	}
	if err := port.Embed(collections); err != nil || profile == search.ProfileKeyword {
		return err
	}
	pending, err := port.NotYetSearchable()
	if err != nil {
		return err
	}
	if pending > 0 {
		return fmt.Errorf("%d documents of the corpus are not embedded after qmd embed; a %s report "+
			"would measure an unembedded index", pending, profile)
	}
	return nil
}

// refuseATwiceToldCorpus refuses what a stand already answers. Overridden
// silently, a report would name a question set nobody asked for.
func refuseATwiceToldCorpus(o Options) error {
	switch {
	case o.Corpus == "":
		return nil
	case o.ScopeSet || o.Questions != "":
		return errors.New("--corpus names its own scope and question set; drop --scope/--questions")
	case o.Latency:
		// The throwaway state has no catalog to time.
		return errors.New("--latency measures the everyday chain; drop it with --corpus")
	case o.Out == "":
		// The default would be a folder inside the read-only stand.
		return errors.New("--corpus has no measurement folder of its own; name one with --out")
	}
	return nil
}

// benchAreas is every measured area's directory by scope. With "all" the
// hits of one run come from several areas, and each resolves against its own.
func benchAreas(registered []config.Area, scope string) (map[string]string, error) {
	areas := selectAreas(registered, scope)
	if scope != allAreas && len(areas) == 0 {
		return nil, fmt.Errorf("no area named %q in the registry", scope)
	}
	return areas, nil
}

// selectAreas is the directory by scope of every registered area scope names.
func selectAreas(registered []config.Area, scope string) map[string]string {
	areas := map[string]string{}
	for _, area := range registered {
		if scope == allAreas || area.Scope == scope {
			areas[area.Scope] = area.Path
		}
	}
	return areas
}

// pointedScope is the one area every expect of the questions lies in, or
// all when they lie in several. An expect outside every area could never be
// found, and the run would report a silent miss.
func pointedScope(registered []config.Area, questions []Question) (string, error) {
	pointed := map[string]bool{}
	for _, q := range questions {
		// Areas nest (a craft inside the knowledge area): the deepest one
		// holding the note owns it, whatever the registry's order. Two
		// areas on one directory are equally deep; the registry allows
		// them, and the first registered owns the note.
		owner, depth := "", -1
		for _, area := range registered {
			if d := len(filepath.Clean(area.Path)); d > depth && inside(q.Expect, area.Path) {
				owner, depth = area.Scope, d
			}
		}
		if depth < 0 {
			return "", fmt.Errorf("question %s expects %s, which lies in no registered area; name the area with --scope", q.ID, q.Expect)
		}
		pointed[owner] = true
	}
	if len(pointed) != 1 {
		return allAreas, nil
	}
	return slices.Collect(maps.Keys(pointed))[0], nil
}

// inside tells whether path is dir or lies below it, each step compared the
// way the engine's answers are.
func inside(path, dir string) bool {
	for p := filepath.Clean(path); ; p = filepath.Dir(p) {
		if sameFile(p, dir) {
			return true
		}
		if filepath.Dir(p) == p {
			return false
		}
	}
}

// benchOut is the measurement folder. The default exists only where one
// area is measured.
func benchOut(out string, areas map[string]string) (string, error) {
	if out != "" {
		return out, nil
	}
	if len(areas) != 1 {
		return "", fmt.Errorf("a run over %d areas has no measurement folder of its own — name one with --out", len(areas))
	}
	for _, path := range areas {
		out = filepath.Join(path, "98 Messung")
	}
	return out, nil
}

// listed is what the engine holds of every area, by scope. The engine knows
// collections, not scopes.
func listed(port search.SearchPort, areas map[string]string) (map[string][]string, error) {
	listings := map[string][]string{}
	for _, scope := range slices.Sorted(maps.Keys(areas)) {
		paths, err := port.Indexed(search.CollectionName(scope))
		if err != nil {
			return nil, err
		}
		listings[scope] = paths
	}
	return listings, nil
}

// firstDocument is the least (scope, relative) the engine holds. Read
// latency follows the document's size, so the choice is by name, never by
// the engine's listing order; ok is false for an empty index.
func firstDocument(listings map[string][]string) (string, string, bool) {
	for _, scope := range slices.Sorted(maps.Keys(listings)) {
		if paths := listings[scope]; len(paths) > 0 {
			return scope, slices.Min(paths), true
		}
	}
	return "", "", false
}

// chain is the search chain above the port as everyday use asks it: with
// the privacy filter, the register and the reconcile stamp.
type chain struct {
	port               search.SearchPort
	scope              string
	channel            privacy.Channel
	stateDir, fallback string
	now                func() time.Time
}

func (c chain) search(query string, profile search.Profile, n int) (*search.SearchAnswer, error) {
	return search.ExecuteSearch(query, c.scope, profile, n, c.channel, c.port, c.stateDir, c.fallback, c.now())
}

// ask reduces the chain to ranked absolute paths among the first ten. What
// the chain says about its answer goes to findings, named by the query:
// without it a miss cannot be told from a search that never ran.
func (c chain) ask(profile search.Profile, areas map[string]string, findings *[]string) Ask {
	return func(query string) ([]string, error) {
		answer, err := c.search(query, profile, 10)
		if err != nil {
			return nil, err
		}
		for _, finding := range answer.Findings {
			*findings = append(*findings, query+": "+finding)
		}
		ranked := make([]string, 0, len(answer.Hits))
		for _, hit := range answer.Hits {
			ranked = append(ranked, filepath.Join(areas[hit.Scope], filepath.FromSlash(hit.Relative)))
		}
		return ranked, nil
	}
}

// latency times catalog, read and the three profiles. The query is probed
// once, untimed, first: one that finds nothing would time the chain's
// retry on an empty answer instead of a search. The keyword probe proves
// the keyword and full rows, not the purely vectorial fast row.
func (c chain) latency(readScope, relative, query string, repeat int, clock func() time.Time) (*Latency, error) {
	probe, err := c.search(query, search.ProfileKeyword, 5)
	if err != nil {
		return nil, err
	}
	if len(probe.Hits) == 0 {
		return nil, fmt.Errorf("the latency query %q finds nothing in %q; every timed search would "+
			"measure the empty-result retry instead of a search — name a query that hits with --latency-query", query, c.scope)
	}
	answering := func(req answer.Request) func() error {
		req.Channel = c.channel
		return func() error {
			_, _, err := answer.RunWith(answer.Ports{}, req, c.stateDir, c.fallback, nil)
			return err
		}
	}
	searching := func(profile search.Profile) func() error {
		return func() error {
			_, err := c.search(query, profile, 5)
			return err
		}
	}
	timings, err := MeasureLatency([]Operation{
		{"catalog", answering(answer.Request{Command: "catalog", Scope: c.scope})},
		{"read", answering(answer.Request{Command: "read", Scope: readScope, Query: relative})},
		{"keyword", searching(search.ProfileKeyword)},
		{"fast", searching(search.ProfileFast)},
		{"full", searching(search.ProfileFull)},
	}, repeat, clock)
	if err != nil {
		return nil, err
	}
	// Scope-qualified: across areas a bare relative path names as many
	// files as there are areas.
	return &Latency{Document: readScope + "/" + relative, Query: query, Timings: timings}, nil
}
