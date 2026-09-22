package hooks

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/verify"
)

// DefaultStopBudget is how long the gate's chain may take in all: under the
// 300 s its settings entry grants, with room for git and the report. The
// budget belongs to the scope, as it does for post-edit.
const DefaultStopBudget = 270 * time.Second

// MaxBlocks is how many turn ends in a row the gate holds before it lets one
// go. It guards against a loop, and a loop is blocks in a row: a green run in
// between ends the row.
const MaxBlocks = 3

// NoVerifyMarker is the file a human sets to let turns end unchecked. The
// write barrier keeps it out of an agent's reach (guard.go).
const NoVerifyMarker = ".loomux/no-verify"

// StopEnv is what a stop run needs from outside: what starts a tool and
// finds it, which binary {loomux} names, the budget, the clock, and whether
// Godot has imported a project. A nil ImportReady asks the disk.
type StopEnv struct {
	Start       func(child.Spec) child.Result
	Look        func(string) (string, error)
	Loomux      string
	Budget      time.Duration
	Now         func() time.Time
	ImportReady func(dir string) bool
}

// The seams no world can provoke: a plan that fails, and git failing on a
// repository it just answered about.
var (
	stopPlan = verify.Plan
	stopTree = gitwork.ContentTree
)

// errNoRepository says the root is outside every tree git measures; the gate
// then runs the chain every time.
var errNoRepository = errors.New("no repository")

// Stop checks that everything since the last green pass is green before a
// turn ends, with the real tools.
func Stop(stdin io.Reader, stderr io.Writer, root, hostName string, budget time.Duration) int {
	loomux, err := editExecutable()
	if err != nil {
		loomux = "loomux"
	}
	return RunStop(stdin, stderr, root, hostName, StopEnv{
		Start: child.Run, Look: exec.LookPath, Loomux: loomux, Budget: budget, Now: time.Now, ImportReady: verify.ImportReady,
	})
}

// RunStop is the stop gate: 0 lets the turn end, 2 holds it with the reason
// on stderr, 1 is a gate that could not judge and holds nothing. The order is
// payload, findings of the subagents, counter, marker, tree, config, chain.
func RunStop(stdin io.Reader, stderr io.Writer, root, hostName string, env StopEnv) int {
	say := func(format string, a ...any) { fmt.Fprintf(stderr, "loomux hook stop: "+format+"\n", a...) }
	host, err := hosts.ParseHost(hostName)
	if err != nil {
		say("%v", err)
		return ExitInternal
	}
	payload, err := hosts.Read(host, stdin)
	if err != nil {
		say("%v", err)
		return ExitInternal
	}
	id := payload.SessionID
	if id == "" {
		// No shared fallback file: two sessions would count each other's
		// blocks (stop.py:102-109).
		say("payload carries no session_id")
		return ExitInternal
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}

	state := sessions.ReadState(root, id)
	save := func() {
		if err := sessions.WriteState(root, id, state); err != nil {
			say("%v", err)
		}
	}
	// First the findings: what moved while a subagent ran is for the main
	// agent to see whether or not anything is verified.
	found := printFindings(stderr, root, id)

	if state.Blocks >= MaxBlocks {
		// The give-up turn ends, and stderr at exit 0 reaches nobody: the
		// findings it printed stay on disk, and the next turn end, with the
		// counter back at 0, delivers them again and holds for them.
		say("gave up after %d consecutive blocks; base stays at %s. Fix the lanes or set %s.", MaxBlocks, shortSHA(state.Base), NoVerifyMarker)
		state.Blocks = 0
		save()
		return ExitOK
	}

	// A finding whose file stays behind arrives again at the next turn end,
	// and a findings hold counts nothing on its own, so nothing would bound
	// that loop -- not even the marker, which does not skip findings. It
	// counts, and the give-up arm above ends the row.
	stuck := len(found) > 0 && !clearFindings(stderr, root, id, found)
	if stuck {
		state.Blocks++
		save()
	}
	// One turn end is one block whatever holds it: a red chain or a git
	// failure beside a stuck finding is not a second one.
	countBlock := func() {
		if !stuck {
			state.Blocks++
			save()
		}
	}
	// A green pass ends a row of blocks -- but not a row of turn ends held by
	// a finding the gate cannot clear away, which the lanes say nothing about.
	passed := func() {
		if !stuck {
			state.Blocks = 0
		}
		save()
	}
	// Findings hold the turn whatever else happens below: once cleared,
	// the next turn end has nothing left to deliver.
	holdForFindings := len(found) > 0
	end := func(code int) int {
		if holdForFindings && code != ExitDenied {
			return ExitDenied
		}
		return code
	}
	// The marker skips the chain, not the findings.
	if _, err := os.Stat(filepath.Join(root, NoVerifyMarker)); err == nil {
		return end(ExitOK)
	}

	head, tree, err := stopTrees(stderr, root, state)
	switch {
	case errors.Is(err, errNoRepository):
	case err != nil:
		say("%v", err)
		countBlock()
		return ExitDenied
	case tree == state.Green:
		// A tree found green before is a green pass, and ends a row of
		// blocks as a green chain does. Written only when that changes
		// something: the no-op turn end stays one that writes nothing.
		if !stuck && state.Blocks != 0 {
			passed()
		}
		return end(ExitOK)
	}

	facts := detect.Detect(os.DirFS(root))
	eff, err := editLoad(root, facts)
	if err != nil {
		say("%v", err)
		return end(ExitInternal)
	}
	// Built in: a config can replace the stop profile but not remove it.
	kinds, _ := verify.ExpandProfile(eff.Config, "stop")
	runID := verify.NewRunID(env.Now(), os.Getpid())
	ready := env.ImportReady
	if ready == nil {
		ready = verify.ImportReady
	}
	jobs, err := stopPlan(eff, verify.Request{Kinds: kinds, Scope: verify.ScopeCheck}, verify.PlanEnv{
		Root: root, Loomux: env.Loomux, RunID: runID, HasTests: verify.HasTests, ImportReady: ready,
	})
	if err != nil {
		say("%v", err)
		return end(ExitInternal)
	}
	jobs = append(jobs, WikiGateJobs(eff, facts, root, kinds)...)
	if err := verify.PrepareCover(root); err != nil {
		say("%v", err)
		return end(ExitInternal)
	}
	outs := verify.Run(jobs, verify.RunOptions{
		Scope: verify.ScopeCheck, MaxParallel: eff.Config.MaxParallel, Timeout: eff.Config.Timeout,
		Budget: env.Budget, Start: env.Start, Look: env.Look, Now: env.Now,
	})
	code := stopVerdict(stderr, kinds, outs)
	if err := verify.CleanCover(root, runID, code == ExitOK); err != nil {
		say("cleaning coverage files: %v", err)
	}
	switch code {
	case ExitDenied:
		countBlock()
		return ExitDenied
	case ExitOK:
		state.Base, state.Green = head, tree
		passed()
	}
	return end(code)
}

// stopTrees answers where HEAD stands and what tree the work is, or
// errNoRepository. A tree equal to the base's is nothing new; it is returned
// as the green tree's twin so the caller has one comparison to make.
func stopTrees(stderr io.Writer, root string, state sessions.SessionState) (head, tree string, err error) {
	// Every refusal gitwork.Head has is one of the two -- a root git ignores
	// and a root no working tree covers -- and both mean the same here: there
	// is no tree to measure, so the chain runs every time.
	if head, err = gitwork.Head(root); err != nil {
		return "", "", errNoRepository
	}
	base := state.Base
	if base == "" && head != "" {
		fmt.Fprintf(stderr, "loomux hook stop: no base commit for this session; measuring from HEAD, so what this session committed stays unseen\n")
		base = head
	}
	baseTree := gitwork.EmptyTree
	if base != "" {
		if baseTree, err = gitwork.TreeOf(root, base); err != nil {
			// A base that amend, rebase and gc took away is no git failure
			// to hold the turn over: measure from HEAD, and the next green
			// run sets a new base (plan, Nachtrag 8).
			fmt.Fprintf(stderr, "loomux hook stop: base %s is gone; measuring from HEAD\n", shortSHA(base))
			baseTree = headTree(root, head)
		}
	}
	// The copied index goes to the system's temp directory, not to the state:
	// a state directory that cannot be written must not become a git failure
	// that holds every turn.
	tree, err = stopTree(root, os.TempDir())
	if err != nil {
		return "", "", err
	}
	if tree == baseTree {
		// Nothing since the base: the same answer as a green tree.
		return head, state.Green, nil
	}
	return head, tree, nil
}

// headTree is the tree HEAD holds, and the empty tree where there is no
// commit to hold one.
//
//coverage:exempt TreeOf cannot refuse a commit rev-parse resolved a moment ago; the empty tree is what an unborn repository measures against
func headTree(root, head string) string {
	if head == "" {
		return gitwork.EmptyTree
	}
	tree, err := gitwork.TreeOf(root, head)
	if err != nil {
		return gitwork.EmptyTree
	}
	return tree
}

// stopVerdict writes the red lanes and says what they mean for the turn:
// red holds it, a budget that ran out or a kind with nothing to check leaves
// it unjudged, anything else passes.
func stopVerdict(stderr io.Writer, kinds []string, outs []verify.Outcome) int {
	var red []verify.Outcome
	for _, o := range outs {
		if verify.Red(o.State, verify.ScopeCheck) {
			red = append(red, o)
		}
	}
	if len(red) > 0 {
		// Only the red lanes: what reaches the agent's context is what it
		// has to fix.
		verify.WriteCheck(stderr, red, false)
		return ExitDenied
	}
	if slices.ContainsFunc(outs, func(o verify.Outcome) bool { return o.State == verify.StateBudget }) {
		fmt.Fprintln(stderr, "loomux hook stop: not everything was verified; raise --budget or shrink the stop profile")
		return ExitInternal
	}
	if _, notes := verify.CheckVerdict(kinds, outs); len(notes) > 0 {
		for _, note := range notes {
			fmt.Fprintf(stderr, "loomux hook stop: %s\n", note)
		}
		fmt.Fprintln(stderr, "loomux hook stop: nothing was verified for these kinds; the base stays")
		return ExitInternal
	}
	return ExitOK
}

// printFindings writes what stopped subagents left, prefixed with each
// agent's id, and returns it for clearFindings. It clears nothing: the
// give-up turn prints findings it must not take away.
func printFindings(stderr io.Writer, root, sessionID string) []sessions.Finding {
	found := sessions.Findings(root, sessionID)
	for _, f := range found {
		for _, line := range f.Lines {
			fmt.Fprintf(stderr, "subagent %s: %s\n", f.AgentID, line)
		}
	}
	return found
}

// clearFindings removes what printFindings delivered and reports whether
// every file is gone: one that stays is delivered again at the next turn
// end, and the caller counts that as a block.
func clearFindings(stderr io.Writer, root, sessionID string, found []sessions.Finding) bool {
	cleared := true
	for _, f := range found {
		if err := clearFinding(root, sessionID, f.AgentID, len(f.Lines)); err != nil {
			fmt.Fprintf(stderr, "loomux hook stop: %v\n", err)
			cleared = false
		}
	}
	return cleared
}

// readAgent is the seam for the file a subagent rewrites between the delivery
// and the clear -- the interleaving this stage cannot provoke from outside.
var readAgent = sessions.ReadAgent

// clearFinding takes the `delivered` lines this turn printed out of one
// subagent's file, and nothing else.
//
// It decides against the file as it stands here, not against the copy
// printFindings read: the same agent's subagent-stop may have appended a
// line in between, and that line has not been delivered. Findings are
// appended oldest first, so what was delivered is the front of the list and
// whatever tail arrived since stays. A list shorter than the count cannot
// happen unless something else rewrote the file, and then everything counts
// as delivered.
//
// This narrows the window to the moment between the re-read and the rename;
// a write that lands exactly there still loses a line. That is the honest
// limit without a lock, and a lock over a file two processes touch is not
// something this stage builds.
//
// A file that still carries a snapshot belongs to a subagent running again
// under the same id: it keeps the snapshot, because removing it would leave
// that run with nothing to compare against at its stop.
func clearFinding(root, sessionID, agentID string, delivered int) error {
	file, found := readAgent(root, sessionID, agentID)
	if !found {
		// Gone already, or a path no read reaches: a removal answers nil for
		// the first and names what stands in the way for the second, which is
		// a finding that arrives again and counts.
		return sessions.RemoveAgent(root, sessionID, agentID)
	}
	var rest []string
	if len(file.Finding) > delivered {
		rest = file.Finding[delivered:]
	}
	if len(rest) == 0 && file.Snapshot == nil {
		return sessions.RemoveAgent(root, sessionID, agentID)
	}
	file.Finding = rest
	return writeAgent(root, sessionID, agentID, file)
}

func shortSHA(sha string) string {
	if sha == "" {
		return "no commit"
	}
	return sha[:min(12, len(sha))]
}
