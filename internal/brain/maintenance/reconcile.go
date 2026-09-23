package maintenance

// The reconciliation: turning a changed raw source into a review case.
//
// Two stages, and `Scan` holds both: `stat` first -- modification time and
// size against a cache of the last clean reading -- and a content hash only
// for the suspects that survive it. Report counts both, so "the filter is two
// staged" is a number a test can read rather than a claim in a comment.
//
// One case per page, never per source: the decision under review is about the
// text of one page, and three cases for one page would produce three proposals
// contradicting each other. Changed sources are therefore collected across
// *all* areas first and only then matched against every wiki -- a source of
// one area may well be cited by the wiki of another.
//
// Not built here, on purpose: `source_missing`. A vanished path is the index
// run's rename detection to judge, and reading it as a change would raise a
// case for a file that merely moved.
//
// Two things this file does **not** do that the reference does, both of them
// recorded in docs/.superpowers/parity/stufe-3a.md:
//
//   - **Nobody is asked for a proposal.** `_proposers` builds one Ollama
//     proposer per `local_only` area; the local model is stage 4. A closed
//     area's case therefore carries `Manual` and the note that says why no
//     proposal lies beside it, which is exactly the state the field semantics
//     were written for.
//   - **The merge trigger asks nobody either.** `_merge_cases` is here, behind
//     the source cases and through the same `landCase`, but the case it lands
//     carries no proposal for the same reason a source case does not.
//
// The original is `src/brain/maintenance/reconcile.py`.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/brain/vcs"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// The file names one case directory may hold beside `case.toml`, and the note
// a closed area's case carries.
//
// localOnlyNote is `_LOCAL_ONLY_NOTE` and stays spelt to the letter: it is
// written into `case.toml`, which the Python side reads back, so AGENTS.md:28
// exempts it the way it exempts the segment labels of package.go.
const (
	proposalName    = "proposal.md"
	supersededName  = "superseded-proposal.md"
	caseName        = "case.toml"
	packageName     = "package.md"
	lastRunRelative = "maintenance/last-run.txt"
	localOnlyNote   = "manual review: this area is local_only, so no skill path is offered (spec 5)"
	localOnlyMode   = "local_only"
)

// mergePathsHeading and mergeSubjectsHeading are `_MERGE_PATHS` and
// `_MERGE_SUBJECTS`. German and spelt to the letter, for the reason
// localOnlyNote is: they are written into `package.md`, which the Python side
// reads back and which `absorbable` searches for them, so AGENTS.md:28 exempts
// them the way it exempts the segment labels of package.go.
const (
	mergePathsHeading    = "# geänderte Dateipfade des Merges — nur Namen, kein Dateiinhalt"
	mergeSubjectsHeading = "# Commit-Betreffzeilen des Merges — je Commit der erste Absatz (git %s)"
)

// openRealizations is `_OPEN_REALIZATIONS`: the two values that say a page
// still promises something, which is the whole question a merge case asks --
// is what it promised now built?
var openRealizations = []string{"planned", "in_progress"}

// ErrNoReviewCentre is the one failure of a pass that a caller has to tell
// apart from every other.
//
// `reindex` catches up on the gate before it indexes, and where there is no
// gate there is nothing to walk around -- so it goes on. Every other
// refusal here says the pass broke, which is a reason to stop.
var ErrNoReviewCentre = errors.New("no area declares [layout] review; there is nowhere to put a case")

// Report is what one pass looked at, what it had to hash, and what it
// produced.
type Report struct {
	Checked, Hashed int
	Cases           []Case

	// Unreadable are the case files that could not be read. Not an abort: a
	// case is a readable file in the vault and gets hand-edited, so one typo
	// must not stop the pass for every other area -- but it must not vanish
	// either, or the broken case sits there while a second one opens beside it.
	Unreadable []string
}

// Reconcile compares every registered source against its register entry and
// raises the cases the changes call for.
//
// Both directories come in through lookup and neither is read from the
// environment, for the reason config.ResolvedAreaDir states: `internal/serve`
// promises everything hangs off the state directory it was handed, and a
// lookup that asked StateDir() itself would break that for every caller.
func Reconcile(areas []config.Area, lookup config.ArtifactLookup, now time.Time) (Report, error) {
	return ReconcileContext(context.Background(), areas, lookup, now)
}

// ReconcileContext is Reconcile under a context that can end it. It is asked
// after every area's scan, the last of them standing before the first write,
// so a pass that ends early has only read: no case, no stamp.
func ReconcileContext(ctx context.Context, areas []config.Area, lookup config.ArtifactLookup, now time.Time) (Report, error) {
	manifests, err := manifestsOf(areas, lookup)
	if err != nil {
		return Report{}, err
	}
	root, err := reviewRootOf(areas, manifests)
	if err != nil {
		return Report{}, err
	}
	checked, hashed := 0, 0
	changed := map[string]Changed{}
	for _, area := range areas {
		manifest := manifests[area.Scope]
		if manifest == nil {
			continue
		}
		areaChecked, areaHashed, areaChanged, err := Scan(area, manifest, lookup.Primary, lookup.Fallback)
		if err != nil {
			return Report{}, err
		}
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		checked += areaChecked
		hashed += areaHashed
		for docID, item := range areaChanged {
			changed[docID] = item
		}
	}
	var broken []string
	raw, err := sourceCases(areas, manifests, root, changed, now, &broken)
	if err != nil {
		return Report{}, err
	}
	// After the source cases, never before: a merge case defers to a standing
	// source case for the same page, and it can only see one that is already
	// on disk.
	merged, err := mergeCases(areas, manifests, root, lookup, now, &broken)
	if err != nil {
		return Report{}, err
	}
	if err := writeLastRun(lookup, now); err != nil {
		return Report{}, err
	}
	sort.Strings(broken)
	return Report{
		Checked: checked, Hashed: hashed, Cases: dedupedByID(append(raw, merged...)),
		Unreadable: slices.Compact(broken),
	}, nil
}

// dedupedByID is the report's case list with the first version of every id and
// no other.
//
// The guard is the merge trigger's, not the source one's: a source case is
// raised once per area and target, and CaseID takes both into its digest -- so
// within that producer two cases can never share an id. Two merge events of
// one range, recorded under two spellings of one repository, do: both resolve
// to the same area, and the second absorbs the case the first landed and hands
// it back unchanged.
//
// The first version wins because it is the one that was landed; the second is
// that very case read back off the disk, and both describe one directory.
func dedupedByID(raw []Case) []Case {
	seen := map[string]bool{}
	var cases []Case
	for _, item := range raw {
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		cases = append(cases, item)
	}
	return cases
}

// ReviewRoot is the vault's one review centre, for callers that only want to
// find a case.
//
// A thin composition rather than a second derivation: `[layout] review` has
// exactly one reader here, and a command that assembled the path from the
// manifest itself would be the second place a rename has to reach.
func ReviewRoot(areas []config.Area, lookup config.ArtifactLookup) (string, error) {
	manifests, err := manifestsOf(areas, lookup)
	if err != nil {
		return "", err
	}
	return reviewRootOf(areas, manifests)
}

// writeLastRun records when the last full pass ran. Its reader is
// search.ReadLastRun, behind the staleness warning of search and of status;
// nothing in this package reads it back, and the catch-up before an index run
// does not ask it -- that pass runs every time.
func writeLastRun(lookup config.ArtifactLookup, now time.Time) error {
	return writeIfChanged(lookup.WritePath(filepath.FromSlash(lastRunRelative)), pytext.IsoFormat(now)+"\n")
}

// manifestsOf is every area's declaration, skipping the ones that have not
// declared themselves.
//
// Registering an area before it writes a declaration is normal, and such an
// area simply carries nothing to compare. A declaration that exists and does
// not read is the opposite: reconciling on a guess about which sources an area
// holds would raise cases for the wrong files.
//
// The declaration is looked for under ResolvedAreaDir and not under the area,
// because a read-only area keeps its artefacts -- the declaration among them
// -- in the state directory.
func manifestsOf(areas []config.Area, lookup config.ArtifactLookup) (map[string]*config.Manifest, error) {
	found := map[string]*config.Manifest{}
	for _, area := range areas {
		manifest, err := config.ReadAreaManifestUntilStage4(
			config.ResolvedAreaDir(area, lookup.Primary, lookup.Fallback))
		if errors.Is(err, config.ErrNoManifest) {
			continue
		}
		if err != nil {
			return nil, err
		}
		found[area.Scope] = manifest
	}
	return found, nil
}

// reviewRootOf is the one review centre, found by the declaration that names
// it.
//
// The vault identifies itself rather than being named again in the registry:
// `[layout] review` is already the field the indexer's exclusion reads
// (`privacy.ReviewExcludes`), so a second declaration could only ever
// contradict it.
func reviewRootOf(areas []config.Area, manifests map[string]*config.Manifest) (string, error) {
	var roots []string
	for _, area := range areas {
		manifest := manifests[area.Scope]
		if manifest == nil || manifest.LayoutReview == "" {
			continue
		}
		root, err := declaredReview(area, manifest)
		if err != nil {
			return "", err
		}
		roots = append(roots, root)
	}
	if len(roots) == 0 {
		return "", ErrNoReviewCentre
	}
	if len(roots) > 1 {
		return "", fmt.Errorf("%s and %s both declare [layout] review; two review centres are none",
			roots[0], roots[1])
	}
	return roots[0], nil
}

// declaredReview is one area's review centre, checked before it is believed.
//
// This path is the ground of the write barrier's one exemption (`brain guard`,
// internal/brain/guard), so a value reaching out of the area would make every
// file named proposal.md on the disk writable. A workspace area's own
// declaration lies inside the tree the agent may write, so that value is
// reachable input.
//
// Two tests and not one, because filepath.Join is not pathlib's `/`. Python
// joins an absolute or rooted value by *replacing* the area's root, so
// `is_relative_to` catches it; Join concatenates `C:/aside` onto the area and
// the result would sit comfortably inside it. The rooted test is therefore
// made here in so many words, and the containment test after it catches `..`
// on the way out, which no type check would.
//
// Python additionally refuses a `[layout] review` that is not a string;
// Manifest.LayoutReview is typed, so there is nothing left to refuse.
//
// The containment is decided by **guard's own resolver** and by nothing else,
// and that is the whole point of the import. `[layout] review` has two
// readers: the barrier, which grants its one exemption over this path, and
// this function, which writes case files into it. Two readers of one field
// that answer differently are worse than either answer -- measured on
// 2026-09-20 with a junction out of the area, the barrier withdrew the
// exemption ("") while this side accepted the place and a write landed
// outside. `filepath.EvalSymlinks` does not close that: Windows reports a
// junction as `ModeIrregular` rather than a link, so EvalSymlinks hands it
// back unchanged and only `os.Readlink` follows it. guard's `finalPath` does
// follow it, together with 8.3 aliases and the case of every component, and
// it resolves a path that does not exist yet -- which is the ordinary state
// of a review centre before the first case.
//
// It also settles what `supersede` may delete: a spent case directory is
// removed with `os.RemoveAll`, and under a junction that removal lands in the
// target. Since the target now has to lie inside the area, it cannot land
// anywhere else.
func declaredReview(area config.Area, manifest *config.Manifest) (string, error) {
	declared := manifest.LayoutReview
	refusal := fmt.Errorf("%s: [layout] review must stay inside the area, found %q", area.Scope, declared)
	if rootedValue(declared) {
		return "", refusal
	}
	root := filepath.Join(area.Path, filepath.FromSlash(declared))
	resolvedArea, areaErr := guard.ResolvePath(area.Path)
	resolvedRoot, rootErr := guard.ResolvePath(root)
	// A value this side cannot resolve joins the containment failure, the way
	// guard folds the same three into one "" (guard.go, `reviewCentre`): all
	// three mean "this value cannot be shown to stay inside the area", and a
	// place that was never resolved is no place to put a case.
	//
	// Two of the three operands have a case in this suite -- the containment
	// and a drive-relative area, which fails `areaErr`. `rootErr` has none.
	// No spelling reaches it alone: root is the area plus one relative step,
	// so a spelling that breaks it breaks the area first. What does reach it
	// is the file system below the area -- links on the review path that lead
	// in a circle (`errLinkCycle`), or a file-system error the resolver does
	// not take as a reason to step up one component (`stopsResolving`). It is
	// written out because it is the same answer, not because a test reaches
	// it.
	if areaErr != nil || rootErr != nil || !guard.IsRelativeTo(resolvedRoot, resolvedArea) {
		return "", refusal
	}
	// The unresolved path is what comes back, the way `_declared_review`
	// answers `area.path / declared`: the resolution is the check, not the
	// answer, and a reader looking for a case should find the place the
	// manifest names.
	return root, nil
}

// rootedValue says whether a declared value names a place of its own rather
// than a step below the area: a leading separator of either kind, or a volume
// name. `filepath.IsAbs` is not enough on Windows, where a rooted path without
// a drive is not called absolute although joining one still replaces the root.
func rootedValue(value string) bool {
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\`) {
		return true
	}
	return filepath.VolumeName(value) != ""
}

// sourceCases groups the changed sources by the page derived from them, one
// case each.
func sourceCases(
	areas []config.Area,
	manifests map[string]*config.Manifest,
	reviewRoot string,
	changed map[string]Changed,
	now time.Time,
	broken *[]string,
) ([]Case, error) {
	var cases []Case
	for _, area := range areas {
		manifest := manifests[area.Scope]
		if manifest == nil || !isDirectory(area.WikiPath) {
			continue
		}
		index, err := Dependents(area.WikiPath)
		if err != nil {
			return nil, err
		}
		affected := map[string][]Changed{}
		// Sorted by doc id, so the sources of one case arrive in the order
		// `case.toml` renders them in whichever way the map was filled.
		for _, docID := range sortedDocIDs(changed) {
			for _, target := range index[docID] {
				affected[target] = append(affected[target], changed[docID])
			}
		}
		for _, target := range sortedTargets(affected) {
			landed, err := landCase(
				area, manifest, reviewRoot, target, affected[target], now, broken,
				"source_change", "source_changed", nil)
			if err != nil {
				return nil, err
			}
			cases = append(cases, landed)
		}
	}
	return cases, nil
}

// mergeCases turns every recorded merge into one case per page it may have
// realised.
//
// One case per candidate page, not one per merge: the page is the unit under
// review, and a single case naming five pages would invite one proposal
// editing five of them.
//
// Nothing here may take the pass down that is not about the vault's own state.
// A merge is recorded by a shell hook that knows nothing about the registry,
// so an event pointing at a deleted throwaway worktree, at a repository behind
// a shared `core.hooksPath`, or at an area that has since been deregistered is
// normal input, not a defect -- each of those is skipped with the event kept.
// Four do come back as an error, counted off against the calls below: the log
// that is not text, a wiki page that cannot be read (`candidates`), a case that
// could not be written, and a drop that could not be recorded. All four say the
// pass broke rather than that this one merge is unreadable.
func mergeCases(
	areas []config.Area,
	manifests map[string]*config.Manifest,
	reviewRoot string,
	lookup config.ArtifactLookup,
	now time.Time,
	broken *[]string,
) ([]Case, error) {
	events, err := ReadEvents(lookup.Primary, lookup.Fallback)
	if err != nil {
		return nil, err
	}
	repositories := byRepository(areas, manifests)
	var cases []Case
	for _, event := range events {
		found, ok := repositories[commonOf(event.Repo)]
		if !ok {
			continue
		}
		evidence, err := mergeEvidence(event)
		if err != nil {
			// The range is unreadable -- garbage-collected, or the work tree
			// is gone. `continue` and nothing else: keeping the event costs a
			// re-read next pass, while dropping it here, or handing the error
			// up, would lose the merge for good. The drop below must not be
			// reached on this path.
			continue
		}
		deferred, err := landMerge(found, reviewRoot, evidence, now, broken, &cases)
		if err != nil {
			return nil, err
		}
		if deferred {
			continue
		}
		// Dropped even when nothing landed: no page of this area was open, so
		// this merge has nothing left to produce on any later pass, and
		// keeping it would raise the same nothing forever.
		if err := DropEvent(lookup.Primary, event); err != nil {
			return nil, err
		}
	}
	return cases, nil
}

// repositoryArea is one registered area together with the declaration a case
// of it is written from.
type repositoryArea struct {
	area     config.Area
	manifest *config.Manifest
}

// byRepository is the areas keyed by the git directory their working tree
// shares.
//
// An event's repository is the working tree the merge ran in, which for a
// linked worktree is not the area's path at all -- and may not exist any more.
// The common directory is the only value that identifies a repository across
// its worktrees, so the match is made on that.
//
// First area wins where two share one repository: both would get the same
// evidence, and raising the same merge twice for two wikis of one checkout is
// noise rather than an extra finding.
func byRepository(
	areas []config.Area, manifests map[string]*config.Manifest,
) map[string]repositoryArea {
	found := map[string]repositoryArea{}
	for _, area := range areas {
		manifest := manifests[area.Scope]
		if manifest == nil || !isDirectory(area.WikiPath) {
			continue
		}
		common := commonOf(area.Path)
		if common == "" {
			continue
		}
		if _, taken := found[common]; !taken {
			found[common] = repositoryArea{area: area, manifest: manifest}
		}
	}
	return found
}

// commonOf is the shared git directory of directory, or "" for every way there
// is none.
//
// Both failures fold into the same empty answer, the way `_common` folds both
// into `None`: git ran and refused, because the place is no repository; and
// git could not be run at all, because the directory is gone or git is not
// installed. Neither is a reason to abort a whole reconciliation -- an event
// naming a throwaway worktree that has since been removed is ordinary input --
// so the error is answered here and never handed up. A caller that passed it
// on would take a whole pass down over one unplaceable merge.
//
// The empty answer doubles as the key no area can have: byRepository refuses
// to key on it, so an unresolvable event looks up nothing.
func commonOf(directory string) string {
	common, err := vcs.CommonDirectory(directory)
	if err != nil {
		return ""
	}
	return common
}

// mergeEvidence is the whole evidence of a merge: changed paths and commit
// subjects.
//
// Two git reads and no third. `git show` is absent by construction, so the
// package cannot carry source text however the range is shaped -- which is
// what keeps "no reconciliation against code contents" holding under this
// trigger.
//
// An empty range yields no block rather than an empty one: a heading over
// nothing reads as a list that was lost, not as a list that is empty.
func mergeEvidence(event MergeEvent) ([]string, error) {
	// Both reads happen and only then are they judged, the way the reference
	// calls both and checks the pair afterwards. Written as one check because
	// it is one answer: either the range can be read or it cannot, and no
	// object name is known that `git diff` reads while `git log` refuses it.
	// The two candidates were tried on 2026-09-20 with git 2.54.0: a tree name
	// is taken by both, and a blob name is taken by `git log` here while other
	// builds refuse it to `git diff` -- which makes it a vector for the first
	// operand, never for the second. The second stands because it is the same
	// answer, not because a test reaches it.
	paths, pathsErr := vcs.ChangedPaths(event.Repo, event.First, event.Last)
	subjects, subjectsErr := vcs.CommitSubjects(event.Repo, event.First, event.Last)
	if pathsErr != nil || subjectsErr != nil {
		return nil, errors.Join(pathsErr, subjectsErr)
	}
	var blocks []string
	if len(paths) > 0 {
		sorted := append([]string(nil), paths...)
		// Code point order, which for UTF-8 is byte order: Python's `sorted`
		// and sort.Strings agree on every string either of them can hold.
		sort.Strings(sorted)
		blocks = append(blocks, strings.Join(append([]string{mergePathsHeading}, sorted...), "\n"))
	}
	if len(subjects) > 0 {
		// Unsorted, and that is the reference's doing: the subjects arrive in
		// git's own order, which is the order the commits landed in.
		blocks = append(blocks, strings.Join(append([]string{mergeSubjectsHeading}, subjects...), "\n"))
	}
	return blocks, nil
}

// landMerge lands a case per open candidate, or defers the whole event; it
// reports which.
//
// **Any** standing case for a candidate page defers the event, whatever its
// trigger, unless that case already carries exactly this evidence. One case
// per page allows no second one beside it, and landing over the standing one
// would throw its contents away: a verified source diff and any proposal
// written against it for a source case, and a *different* merge's paths and
// subjects for a merge case. Neither may be swallowed, so the event waits
// until the standing case is decided and lands afterwards -- noise before lost
// knowledge.
//
// The exception is what makes the event droppable at all. A case that already
// carries this evidence was raised by this very event -- the recovery path
// after a run that landed the cases and died before DropEvent -- or by a merge
// that said exactly the same thing. Either way there is nothing left to add,
// so it is absorbed rather than deferred; without it an event would be blocked
// forever by its own cases.
//
// Decide first, land second. A loop that landed as it went would leave a
// deferred event with some of its cases already written, and those cases would
// then block it on every later pass.
func landMerge(
	found repositoryArea,
	reviewRoot string,
	evidence []string,
	now time.Time,
	broken *[]string,
	cases *[]Case,
) (bool, error) {
	scope := search.CollectionName(found.area.Scope)
	targets, err := candidates(found.area)
	if err != nil {
		return false, err
	}
	deferred := false
	for _, target := range targets {
		// Every candidate is asked and not only up to the first that refuses,
		// the way the reference builds its whole dictionary before the `any`:
		// the lookup records an unreadable case file on its way, and a loop
		// that broke off would hand the caller a shorter list of them.
		directory, standing := standingCase(reviewRoot, scope, target, broken)
		if !absorbable(directory, standing, evidence) {
			deferred = true
		}
	}
	if deferred {
		return true, nil
	}
	for _, target := range targets {
		// `due` out of the closed vocabulary: nothing about a source changed,
		// the page has simply become due for the one question this trigger
		// asks -- is what it promised now built?
		landed, err := landCase(
			found.area, found.manifest, reviewRoot, target, nil, now, broken,
			"merge", "due", evidence)
		if err != nil {
			return false, err
		}
		*cases = append(*cases, landed)
	}
	return false, nil
}

// absorbable says whether landing this evidence on the standing case would
// lose nothing.
//
// A free page is absorbable trivially. An occupied one only when the case is a
// merge case whose package already quotes every block of this evidence -- the
// trigger check matters on its own, because an empty range makes the block
// test vacuous and must not be allowed to land on a source case.
//
// The package is asked rather than `case.toml`, because the range that
// produced it is nowhere in the schema: `sources` is empty for every merge
// case, so two different merges of one page are indistinguishable there. The
// format is not extended for this -- the evidence *is* the identity, and it is
// already written down.
//
// A package that cannot be read is no proof either and reads as not
// absorbable. Python asks `is_file()` first and would raise on a file it
// cannot read; both answers mean the same thing here -- nothing on disk shows
// that this case carries this evidence.
func absorbable(directory string, standing *Case, evidence []string) bool {
	if standing == nil {
		return true
	}
	if standing.Trigger != "merge" {
		return false
	}
	raw, err := readFileFn(filepath.Join(directory, packageName))
	if err != nil {
		return false
	}
	rendered := pageText(raw)
	for _, block := range evidence {
		if !strings.Contains(rendered, block) {
			return false
		}
	}
	return true
}

// candidates is the pages of area that still promise something.
//
// Deliberately the whole open set of the area and not the pages the merge's
// paths point at: a conflicted merge writes no event at all, so the range is a
// narrowing convenience and never the definition. Making it the definition
// would give the two kinds of merge two different candidate sets, and the
// richer one the smaller.
//
// Sorted by the relative path as a string, where the reference sorts the paths
// themselves. The two disagree exactly where Dependents records they do -- `-`
// sorts below `/` -- and here the order decides nothing: every candidate gets
// its own case, and each case's id is taken from its own target.
func candidates(area config.Area) ([]string, error) {
	var open []string
	err := walkPages(area.WikiPath, func(page *wiki.WikiPage) error {
		if page.Realization != nil && slices.Contains(openRealizations, *page.Realization) {
			open = append(open, page.Relative)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(open)
	return open, nil
}

// isDirectory says whether a registered wiki is there to be read. An area that
// names none, and one whose wiki has not been laid down yet, answer the same
// nothing -- neither derives a page from a source.
func isDirectory(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func sortedDocIDs(changed map[string]Changed) []string {
	out := make([]string, 0, len(changed))
	for docID := range changed {
		out = append(out, docID)
	}
	sort.Strings(out)
	return out
}

func sortedTargets(affected map[string][]Changed) []string {
	out := make([]string, 0, len(affected))
	for target := range affected {
		out = append(out, target)
	}
	sort.Strings(out)
	return out
}

// landCase writes -- or leaves standing -- the one case for target, and
// answers it.
//
// The second producer of cases -- the merge trigger -- enters here through
// trigger, state and evidence; the detection above is none of its business.
// `evidence` replaces the source diffs as the `D` segments of the package, and
// a merge passes no sources at all.
//
// `weight` is `change` for both producers, and that is a decision rather than
// an oversight: the vocabulary is closed at `change | time`, a merge is an
// event and not the passing of a due date, and the ordering these values feed
// has not been measured yet. Inventing a third value here would change the
// format ahead of the measurement that would justify it.
func landCase(
	area config.Area,
	manifest *config.Manifest,
	reviewRoot, target string,
	sources []Changed,
	now time.Time,
	broken *[]string,
	trigger, state string,
	evidence []string,
) (Case, error) {
	scope := search.CollectionName(area.Scope)
	states := sourceStates(sources)
	standingDir, standing := standingCase(reviewRoot, scope, target, broken)
	if standing != nil && slices.Equal(standing.Sources, states) {
		// Nothing moved since the case was opened: leaving it standing keeps
		// its created, its id, and any proposal already written for it.
		return withCurrentMode(standingDir, *standing, manifest)
	}
	if standing != nil && standing.Trigger == "merge" && trigger == "source_change" {
		// A standing merge case carries evidence out of an event that is
		// already consumed: the paths and subjects of that merge exist nowhere
		// else, so overwriting it with a source case would destroy them for
		// good. The merge case stays standing, the source change stays
		// unadvanced in the register, and a later pass -- once the merge case
		// is decided -- raises it then.
		return withCurrentMode(standingDir, *standing, manifest)
	}
	identifier := CaseID(area.Scope, target, now)
	directory := CaseDir(reviewRoot, scope, identifier)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return Case{}, err
	}
	superseded, err := supersede(standingDir, standing != nil, directory)
	if err != nil {
		return Case{}, err
	}
	page := filepath.Join(area.WikiPath, filepath.FromSlash(target))
	targetHash, err := contentHashFn(page)
	if err != nil {
		return Case{}, err
	}
	closed := manifest.PrivacyMode == localOnlyMode
	landed := Case{
		ID:      identifier,
		Area:    area.Scope,
		Target:  target,
		State:   state,
		Trigger: trigger,
		Weight:  "change",
		Created: now,
		Sources: states,

		TargetHash:         targetHash,
		SupersededProposal: superseded,
		// Here already: the area's mode is settled before anyone was asked,
		// and exactly that makes it the switch.
		LocalOnly: closed,
		// No local model is asked before stage 4, so both of these are settled
		// here as well -- where the reference waits for an answer from it. No
		// proposal means a manual case, and the note says why one is missing.
		Manual: closed,
		Note:   noteFor(manifest),
	}
	segments, err := segmentsOf(sources, page, evidence)
	if err != nil {
		return Case{}, err
	}
	// Built before the writes because the reference hands the proposer exactly
	// the package the checker reads later. RenderPackage takes id and created
	// off the case, and neither moves between here and the write.
	packageText := RenderPackage(landed, segments)
	if _, err := WriteCase(filepath.Join(directory, caseName), landed); err != nil {
		return Case{}, err
	}
	if err := writeIfChanged(filepath.Join(directory, packageName), packageText); err != nil {
		return Case{}, err
	}
	return landed, nil
}

// sourceStates is the state of every source the case was opened over, sorted
// by doc id so two runs render `case.toml` byte for byte alike.
func sourceStates(sources []Changed) []SourceState {
	states := make([]SourceState, 0, len(sources))
	for _, item := range sources {
		states = append(states, SourceState{
			DocID: item.DocID, Revision: item.Revision, ContentHash: item.ContentHash,
		})
	}
	slices.SortFunc(states, func(x, y SourceState) int { return strings.Compare(x.DocID, y.DocID) })
	return states
}

// noteFor is the reason a case carries no proposal.
//
// One wording and not the reference's two: the second of them names a spent
// attempt at the local model, and stage 3a makes none. It returns with the
// local model in stage 4.
func noteFor(manifest *config.Manifest) string {
	if manifest.PrivacyMode != localOnlyMode {
		return ""
	}
	return localOnlyNote
}

// withCurrentMode is a standing case whose LocalOnly follows today's
// declaration.
//
// The one field that is brought up to date, and its job is the reason why: it
// is the switch at which the reader withholds the package. Should an area flip
// from a cloud mode to local_only after the case was opened, false would keep
// standing there and the source diff of a closed area would go out past the
// display. Cases written before the field get the same treatment -- they lack
// it, and a missing field reads as false.
//
// Manual and Note are deliberately **not** derived anew. They record what
// happened when the case was opened; a new verdict would need a second
// question, and a spent attempt is not asked twice. An area flipping the other
// way therefore still carries the old note -- named and accepted, because the
// note only explains what happened back then.
//
// It writes only when something changes: WriteCase renders the file the same
// way every time, and a pass without a change must touch no file.
func withCurrentMode(directory string, standing Case, manifest *config.Manifest) (Case, error) {
	current := manifest.PrivacyMode == localOnlyMode
	if standing.LocalOnly == current {
		return standing, nil
	}
	standing.LocalOnly = current
	if _, err := WriteCase(filepath.Join(directory, caseName), standing); err != nil {
		return Case{}, err
	}
	return standing, nil
}

// standingCase is the open case for target, looked up by target rather than by
// id: CaseID carries the day, so the same page reconciled on two days would
// otherwise open a second case beside the first instead of replacing it.
//
// A directory of the review centre that holds no `case.toml` is none of this
// function's business -- a reviewer may keep notes of their own beside a case.
//
// A case file that cannot be read is skipped, not fatal, and not silent: the
// pass goes on for every other area and the caller is handed the path, so a
// second case opening beside the broken one is at least explicable. Python
// skips only its own CaseError here and lets an unreadable file take the pass
// down; ReadCase answers one error type for both, and the forgiving reading is
// the one that matches what this list is for.
func standingCase(reviewRoot, scope, target string, broken *[]string) (string, *Case) {
	directory := filepath.Join(reviewRoot, scope)
	// Sorted by name, which is what os.ReadDir promises. A directory that is
	// not there yet is the ordinary state of a first pass and no failure.
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", nil
	}
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name(), caseName)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		read, err := ReadCase(path)
		if err != nil {
			*broken = append(*broken, path)
			continue
		}
		if read.Target == target {
			return filepath.Dir(path), &read
		}
	}
	return "", nil
}

// supersede carries a written proposal out of a discarded case and drops the
// rest.
//
// A proposal about a source state that no longer exists is worthless and must
// not be decided on -- but discarding it silently would lose work the reviewer
// already did, so it moves aside under a name of its own and is recorded in
// the new case. An earlier record moves along with it: it is the only trace
// left that somebody once worked here.
//
// The whole directory goes, not the files this package knows by name: the
// vault is open, and a reviewer may well have dropped a folder of their own
// notes beside the case.
func supersede(oldDir string, standing bool, directory string) (string, error) {
	if !standing {
		return "", nil
	}
	target := filepath.Join(directory, supersededName)
	for _, name := range []string{proposalName, supersededName} {
		data, err := os.ReadFile(filepath.Join(oldDir, name))
		if err != nil {
			continue
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return "", err
		}
		break
	}
	if oldDir != directory {
		if err := os.RemoveAll(oldDir); err != nil {
			return "", err
		}
	} else if err := os.Remove(filepath.Join(oldDir, proposalName)); err != nil &&
		!errors.Is(err, fs.ErrNotExist) {
		// Twice on one day the new case lands in the very directory the old
		// one had, so there is nothing to remove but the spent proposal.
		return "", err
	}
	if _, err := os.Stat(target); err != nil {
		return "", nil
	}
	return supersededName, nil
}

// segmentsOf is the quotable units of the package: the diffs, the page, the
// citations.
//
// One `D` segment per hunk, each carrying the file header so it reads on its
// own; one `W` segment per blank-line-separated paragraph of the page, its
// frontmatter left out, because the frontmatter repeats what the `Q` segments
// already carry and a proposal could otherwise satisfy the evidence binding by
// quoting a doc id instead of prose.
//
// A merge case hands its evidence in as `evidence` and brings no sources at
// all: its `D` segments are the changed paths and the commit subjects. They
// keep the `D` kind rather than gaining one of their own, because the kind
// order is the checker's contract for what a package may contain and a new
// kind would have to be agreed there first -- while the role is the same in
// both cases: this is what happened outside the wiki, and it is what a
// proposal may quote.
func segmentsOf(sources []Changed, page string, evidence []string) ([]Segment, error) {
	sorted := append([]Changed(nil), sources...)
	slices.SortFunc(sorted, func(x, y Changed) int { return strings.Compare(x.DocID, y.DocID) })
	hunks := append([]string(nil), evidence...)
	var citations []Citation
	for _, item := range sorted {
		hunks = append(hunks, hunksOf(item)...)
		citations = append(citations, Citation{DocID: item.DocID, Resource: item.Relative})
	}
	raw, err := readFileFn(page)
	if err != nil {
		return nil, err
	}
	var paragraphs []string
	for _, block := range strings.Split(bodyOf(pageText(raw)), "\n\n") {
		if stripped := pytext.Strip(block); stripped != "" {
			paragraphs = append(paragraphs, stripped)
		}
	}
	return BuildPackage(hunks, paragraphs, citations), nil
}

// pageText is the page as Python's `read_text(errors="replace")` reads it:
// decoded the way `decode` decodes a source, and then with the lone carriage
// return folded as well.
//
// That second fold is the one difference to a `D` segment, and it is the
// reference's and not an oversight: a diff is drawn against a content hash and
// has to leave the lone `\r` standing, while the page is only ever read as
// prose here. Universal newlines is what `open()` in text mode does.
func pageText(raw []byte) string {
	return strings.ReplaceAll(decode(raw), "\r", "\n")
}

// bodyOf is the page without its frontmatter.
//
// A plain split rather than the wiki reader: nothing here needs the parsed
// values, and a page with a broken block must still yield its body rather than
// refuse -- the same stance Dependents takes.
func bodyOf(text string) string {
	if !strings.HasPrefix(text, "---\n") {
		return text
	}
	at := strings.Index(text[3:], "\n---")
	if at == -1 {
		return text
	}
	// Only `\n` is cut off the front, never all whitespace: a body opening
	// with an indented line keeps its indentation.
	return strings.TrimLeft(text[3+at+4:], "\n")
}

// writeIfChanged is `write_if_changed`: an untouched file keeps its own
// timestamp, and the swap is the one lock.ReplaceText performs.
//
// os.ReadFile and not pytext.ReadText, for the reason WriteCase states: a
// standing file whose bytes differ only in their line endings would compare
// equal and never be rewritten.
func writeIfChanged(path, text string) error {
	if standing, err := os.ReadFile(path); err == nil && string(standing) == text {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return lock.ReplaceText(path, text)
}
