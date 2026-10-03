package serve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// Upkeep is the daily reconciliation, carried by the serve process that
// happens to be running (`daemon/server.py`, `Upkeep`). Nothing else can own
// "daily": serve starts on a client's request, and a week without a client
// would be a week without a pass. So when the last full pass is more than a
// day old it runs before the first request is answered, and every interval
// after that for as long as the process lives -- no schedule, nothing left
// behind when the process ends.
//
// The gate covers the first pass only, and it holds every tool, the ones that
// need no model included: an answer built on knowledge nobody has checked is
// what it prevents. Later passes run in the background.
//
// It runs `reconcile` and never `reindex`. An index run writes the register
// the scan holds the disk against, so one that went first would let a change
// pass the review gate without becoming a case.
type Upkeep struct {
	lookup   config.ArtifactLookup
	run      func(context.Context, time.Time) (*maintenance.Report, error)
	now      func() time.Time
	interval time.Duration
	after    func(time.Duration) <-chan time.Time

	settled chan struct{}
	once    sync.Once

	mu      sync.Mutex
	failure string
	failed  bool
	report  *maintenance.Report
}

// NewUpkeep is the upkeep of the registry in registryDir, the state directory
// its artefacts are read from too.
func NewUpkeep(registryDir string) *Upkeep {
	lookup := config.ArtifactLookup{Primary: registryDir}
	return &Upkeep{
		lookup: lookup,
		run: func(ctx context.Context, now time.Time) (*maintenance.Report, error) {
			return reconcileRegistered(ctx, lookup, now)
		},
		now:      func() time.Time { return time.Now().UTC() },
		interval: search.ReconcileInterval,
		after:    time.After,
		settled:  make(chan struct{}),
	}
}

// Settled says whether the first pass is behind, which a caller asks before it
// waits.
func (u *Upkeep) Settled() bool {
	select {
	case <-u.settled:
		return true
	default:
		return false
	}
}

// CaughtUp holds the caller until the first pass has settled, run or failed,
// or until ctx ends.
func (u *Upkeep) CaughtUp(ctx context.Context) error {
	select {
	case <-u.settled:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// KeepUp catches up now, then once per interval, until ctx ends.
func (u *Upkeep) KeepUp(ctx context.Context) {
	for {
		u.catchUp(ctx)
		select {
		case <-ctx.Done():
			return
		case <-u.after(u.interval):
		}
	}
}

// catchUp runs one pass when one is owed. Its failure is recorded, never
// raised and never a reason to stop answering: serve runs no index, so there
// is nothing a failed pass could let past the review gate. A tick that ran or
// found nothing owed clears an earlier failure; one that found nothing owed
// forgets the report as well, because the stamp moved without this process --
// somebody reconciled by hand -- and the cases it names may be decided by now.
// The gate opens either way: an unsettled upkeep is a serve that answers
// nobody.
func (u *Upkeep) catchUp(ctx context.Context) {
	defer u.once.Do(func() { close(u.settled) })
	report, err := u.pass(ctx)
	u.mu.Lock()
	defer u.mu.Unlock()
	if err != nil {
		u.failure, u.failed = err.Error(), true
		return
	}
	u.report, u.failure, u.failed = report, "", false
}

func (u *Upkeep) pass(ctx context.Context) (*maintenance.Report, error) {
	stamp, ok, err := search.ReadLastRun(u.lookup.Primary)
	if err != nil {
		return nil, err
	}
	now := u.now()
	if ok && now.Sub(stamp) < u.interval {
		return nil, nil
	}
	return u.run(ctx, now)
}

// reconcileRegistered is one full pass over the registry, or nothing to do
// when there is no registry or no area in it: an unregistered state directory
// is serve's ordinary first minute, not a defect worth a failure line.
func reconcileRegistered(ctx context.Context, lookup config.ArtifactLookup, now time.Time) (*maintenance.Report, error) {
	if info, err := os.Stat(filepath.Join(lookup.Primary, "registry.toml")); err != nil || !info.Mode().IsRegular() {
		return nil, nil
	}
	areas, err := config.ReadRegistry(lookup.Primary)
	if err != nil || len(areas) == 0 {
		return nil, err
	}
	report, err := maintenance.ReconcileContext(ctx, areas, lookup, now)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// Trailer is what every answer of the tool named cmd carries beyond itself:
// `status` the whole listing, every other tool one headline marked with `! `,
// and every tool but `search` -- whose own findings carry it already -- the
// line that the last full pass has aged.
func (u *Upkeep) Trailer(cmd string, ch privacy.Channel) ([]string, error) {
	if cmd == "status" {
		return u.Notes(ch)
	}
	lines, err := u.headline(ch)
	if err != nil {
		return nil, err
	}
	if cmd != "search" {
		stale, err := search.StaleReconcile(u.lookup.Primary, u.now())
		if err != nil {
			return nil, err
		}
		lines = append(lines, stale...)
	}
	for i, line := range lines {
		lines[i] = "! " + line
	}
	return lines, nil
}

// Notes are what the last pass produced, for the reader of `status`, and
// everything in them passes the channel gate first: the pass runs over the
// whole registry, and a case line would otherwise name an area the channel
// may not know exists. A failure and a set of cases do not exclude each other;
// cases opened by an earlier pass still wait for a person after a later one
// failed.
//
// The registry is asked only when there is something to gate. The reference
// asks it on every answer; an upkeep with nothing to say then turned an
// answer that needs no registry into a registry error.
func (u *Upkeep) Notes(ch privacy.Channel) ([]string, error) {
	n, err := u.gated(ch)
	if err != nil {
		return nil, err
	}
	var lines []string
	if n.failed != "" {
		lines = append(lines, n.failed)
	}
	lines = append(lines, n.cases...)
	return append(lines, n.unreadable...), nil
}

// gatedNotes are the three kinds of line Notes is made of, kept apart for the
// headline, which counts cases and skipped files as what they are.
type gatedNotes struct {
	failed            string
	cases, unreadable []string
}

// gated is Notes before it is joined.
func (u *Upkeep) gated(ch privacy.Channel) (gatedNotes, error) {
	u.mu.Lock()
	quiet := !u.failed && (u.report == nil || len(u.report.Cases) == 0 && len(u.report.Unreadable) == 0)
	u.mu.Unlock()
	if quiet {
		return gatedNotes{}, nil
	}
	visible, err := privacy.VisibleAreas(u.lookup.Primary, "all", ch)
	if err != nil {
		return gatedNotes{}, err
	}
	seen := map[string]config.Area{}
	for _, entry := range visible {
		seen[entry.Area.Scope] = entry.Area
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	var n gatedNotes
	if u.failed {
		n.failed = u.failedLine(ch)
	}
	if u.report != nil {
		for _, c := range u.report.Cases {
			if _, ok := seen[c.Area]; ok {
				n.cases = append(n.cases, fmt.Sprintf(
					"the daily reconciliation opened a case for %s/%s; `loomux cases` lists it", c.Area, c.Target))
			}
		}
		for _, path := range u.report.Unreadable {
			if line, ok := unreadableLine(path, ch, seen); ok {
				n.unreadable = append(n.unreadable, line)
			}
		}
	}
	return n, nil
}

// failedLine says why the pass failed, but the cause only on the local
// channel: an error's text names host paths more often than not, and the
// cloud reader could not act on them anyway.
func (u *Upkeep) failedLine(ch privacy.Channel) string {
	tail := ": " + u.failure
	if ch != privacy.ChannelLocal {
		tail = " (the cause is named on the local channel)"
	}
	return "the daily reconciliation failed" + tail + "; sources changed since the last pass " +
		"are not yet cases -- fix the cause, then run `loomux reconcile`"
}

// headline is the same news in one line, for every answer but `status`: the
// failure when there is one, else a count of each kind the channel may see --
// a skipped case file is no open case. Never the list, which is read on
// purpose under `status`.
func (u *Upkeep) headline(ch privacy.Channel) ([]string, error) {
	n, err := u.gated(ch)
	if err != nil {
		return nil, err
	}
	if n.failed != "" {
		return []string{n.failed}, nil
	}
	var counts []string
	if len(n.cases) > 0 {
		counts = append(counts, fmt.Sprintf("%d open case(s)", len(n.cases)))
	}
	if len(n.unreadable) > 0 {
		counts = append(counts, fmt.Sprintf("%d unreadable case file(s)", len(n.unreadable)))
	}
	if len(counts) == 0 {
		return nil, nil
	}
	return []string{strings.Join(counts, " and ") + " from the daily reconciliation; `status` lists them"}, nil
}

// unreadableLine is one skipped case file, addressed by what the channel may
// be told. Two owners have to clear the gate: the case's collection must be a
// scope the channel sees, and the file must lie under the path of an area it
// sees, because the review centre may belong to a hidden area whatever the
// case inside it is about. Off the local channel the address is
// `collection/case id`, which is how `loomux case` names it.
func unreadableLine(path string, ch privacy.Channel, seen map[string]config.Area) (string, bool) {
	caseDir := filepath.Dir(path)
	collection := filepath.Base(filepath.Dir(caseDir))
	known := false
	for scope := range seen {
		known = known || search.CollectionName(scope) == collection
	}
	if !known {
		return "", false
	}
	under := false
	for _, area := range seen {
		under = under || withinFolded(path, area.Path)
	}
	if !under {
		return "", false
	}
	where := path
	if ch != privacy.ChannelLocal {
		where = collection + "/" + filepath.Base(caseDir)
	}
	return "unreadable case file, skipped by the daily reconciliation: " + where, true
}

// withinFolded is `PurePath.is_relative_to` on Windows: part by part and
// without regard to case.
func withinFolded(path, dir string) bool {
	p := strings.Split(strings.ToLower(filepath.ToSlash(filepath.Clean(path))), "/")
	d := strings.Split(strings.ToLower(filepath.ToSlash(filepath.Clean(dir))), "/")
	if len(p) < len(d) {
		return false
	}
	for i := range d {
		if p[i] != d[i] {
			return false
		}
	}
	return true
}
