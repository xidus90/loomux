package hooks

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/verify"
)

// maxSeenLines is how much of the stop gate's last report session start
// passes on: enough to say what is wrong, not a context of findings.
const maxSeenLines = 40

// probationLines are what session start tells an agent about the gate: one
// line per lane in probation, and what the stop gate last found in them while
// HEAD and the armed lanes are what they were then. The stop gate ends a turn over such findings
// with 0, where no host shows its stderr, so this is where they are heard.
// Nothing without the file or without a lane in probation.
func probationLines(root string) []string {
	states, exists, err := ReadLaneStates(root)
	if err != nil {
		return []string{"loomux: " + err.Error()}
	}
	if !exists || len(states.Probation) == 0 {
		return nil
	}
	var lines []string
	for _, key := range states.Probation {
		lines = append(lines, "loomux: lane "+key+" is in probation: what it finds warns and fails no gate until a green commit arms it")
	}
	// The newest stand of any session, which may be about a commit since
	// left: only one about HEAD now says anything about this tree, and only
	// under the lanes armed now -- a gate arm since then changes what the
	// lanes in probation are, and no commit need follow it.
	seen, found := sessions.LastSeen(root)
	if head, err := gitwork.Head(root); !found || err != nil || head != seen.Head {
		return lines
	}
	if armed, _ := verify.ReadArmed(root); !slices.Equal(seen.Armed, armed.Keys) {
		return lines
	}
	report := strings.Split(strings.TrimSuffix(seen.Report, "\n"), "\n")
	lines = append(lines, "loomux: at the last turn end that checked this commit, the lanes in probation reported:")
	lines = append(lines, report[:min(len(report), maxSeenLines)]...)
	if rest := len(report) - maxSeenLines; rest > 0 {
		lines = append(lines, fmt.Sprintf("loomux: %d more lines; `loomux check stop` shows all", rest))
	}
	return lines
}

// preCommitHook is the hook git runs before a commit of root, by path and
// text; false where there is none or no repository to hold one.
func preCommitHook(root string) (path, text string, found bool) {
	dir, err := gitwork.HooksDir(root)
	if err != nil {
		return "", "", false
	}
	path = filepath.Join(dir, "pre-commit")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	return filepath.ToSlash(path), string(data), true
}

// renderProbation is the status report's section on the armed lanes. A
// project without the file gets none: its report reads as it always has.
func renderProbation(w io.Writer, root string) {
	states, exists, err := ReadLaneStates(root)
	if err == nil && !exists {
		return
	}
	fmt.Fprintln(w, "\n--------------------------------------------------------------------------------")
	fmt.Fprintf(w, " Lane Probation (%s)\n", verify.ArmedFile)
	fmt.Fprintln(w, "--------------------------------------------------------------------------------")
	if err != nil {
		fmt.Fprintf(w, " [WARN] %v\n", err)
		return
	}
	for _, key := range states.Armed {
		fmt.Fprintf(w, " [ARMED] %s\n", key)
	}
	for _, key := range states.Probation {
		fmt.Fprintf(w, " [PROBATION] %s: warns only until a green commit arms it\n", key)
	}
	for _, key := range states.Orphans {
		fmt.Fprintf(w, " [ORPHAN] %s: no lane answers to this entry\n", key)
	}
	switch path, text, found := preCommitHook(root); {
	case !found:
		fmt.Fprintf(w, " [WARN] no pre-commit hook arms lanes: %s\n", verify.HowToArm)
	case !verify.HookArms(text):
		fmt.Fprintf(w, " [WARN] %s does not arm lanes: %s\n", path, verify.HowToArm)
	}
	if states.Ignored {
		fmt.Fprintf(w, " [WARN] %s is ignored by git: it reaches no commit and holds on this machine only\n", verify.ArmedFile)
	}
}
