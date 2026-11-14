package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/config/target"
)

// The seams for the file system calls whose failure a test cannot provoke
// on a real disk: a proposal file that cannot be removed and a write that
// fails after the file was read.
var (
	removeProposalFile = func(t target.Target, id string) error { return t.Remove(id) }
	applyWrite         = func(t target.Target, before, next string) error { return t.Write(before, next) }
)

// say prints to a human's terminal. Keys, values and diffs come from an
// agent, so every control byte is shown escaped rather than sent.
func say(w io.Writer, format string, args ...any) {
	fmt.Fprint(w, clean(fmt.Sprintf(format, args...)))
}

// clean escapes every control character but the line break and the tab,
// and every byte that is no UTF-8: an escape sequence in a proposal would
// otherwise move the cursor, recolour or retitle the terminal it is read in.
func clean(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '\n' || r == '\t':
			out.WriteRune(r)
		case r == utf8.RuneError && size == 1:
			fmt.Fprintf(&out, `\x%02x`, s[i])
		case unicode.IsControl(r):
			// Every control character is below U+00A0.
			fmt.Fprintf(&out, `\x%02x`, r)
		default:
			out.WriteString(s[i : i+size])
		}
		i += size
	}
	return out.String()
}

// configPropose is set and unset with --propose: the same computation and
// the same refusals, but the change is stored for a human instead of written.
func configPropose(t target.Target, p target.Proposal, stdout, stderr io.Writer) int {
	text, err := t.Read()
	var diff string
	if err == nil {
		_, diff, err = t.Resolve(text, p)
	}
	if err != nil {
		say(stderr, "loomux config: %v\n", err)
		return 1
	}
	if diff == "" {
		fmt.Fprintln(stderr, "loomux config: already so; nothing proposed")
		return 0
	}
	id, err := t.Store(time.Now(), p)
	if err != nil {
		say(stderr, "loomux config: %v\n", err)
		return 1
	}
	say(stdout, "%s", diff)
	fmt.Fprintf(stdout, "proposal %s stored; a human applies it with `loomux config apply %s`\n", id, id)
	return 0
}

func proposalHint(n int) string {
	if n == 1 {
		return "1 proposal open — loomux config proposals"
	}
	return fmt.Sprintf("%d proposals open — loomux config proposals", n)
}

// configProposals shows each proposal with the diff it would make to the
// file as it is now, each on its own: under --all the later ones meet the
// earlier ones' result instead. A file that does not read is named and the
// others are still shown.
func configProposals(t target.Target, asJSON bool, stdout, stderr io.Writer) int {
	all, err := t.Proposals()
	if err != nil {
		say(stderr, "loomux config: %v\n", err)
		return 1
	}
	code := 0
	views := []target.ProposalView{}
	for _, v := range all {
		if v.Unreadable {
			say(stderr, "loomux config: %s\n", v.Error)
			code = 1
			continue
		}
		views = append(views, v)
	}
	if asJSON {
		// JSON escapes the C0 controls itself but passes DEL and the C1
		// range as they are; a reader who cats the output meets them raw.
		for i, v := range views {
			views[i] = target.ProposalView{Proposal: target.Proposal{ID: clean(v.ID), Op: clean(v.Op), Key: clean(v.Key), Input: clean(v.Input), Created: clean(v.Created), Target: clean(v.Target)}, Diff: clean(v.Diff), Error: clean(v.Error)}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(views)
		return code
	}
	for _, v := range views {
		say(stdout, "%s  %s  %s\n", v.ID, v.Created, v.Proposal.Describe())
		switch {
		case v.Error != "":
			say(stdout, "  refused now: %s\n", v.Error)
		case v.Diff == "":
			fmt.Fprintln(stdout, "  already so; apply removes it")
		default:
			say(stdout, "%s", v.Diff)
		}
	}
	return code
}

// configApply is the human's half: each proposal is computed again against
// the file as it is now, shown, confirmed and written the way `config set`
// writes, and only then removed. A proposal that no longer holds stays.
func configApply(t target.Target, id string, all, yes bool, stdin io.Reader, stderr io.Writer) int {
	ids, err := t.Choose(id, all)
	if err != nil {
		say(stderr, "loomux config: %v\n", err)
		return 1
	}
	if len(ids) == 0 {
		fmt.Fprintln(stderr, "loomux config: no proposals open")
		return 0
	}
	// One reader for every question: a reader per question would buffer
	// the answers meant for the next ones and lose them.
	answers := bufio.NewReader(stdin)
	code := 0
	for _, id := range ids {
		if !applyOne(t, id, yes, answers, stderr) {
			code = 1
		}
	}
	return code
}

// applyOne answers false when the run must end in 1: the proposal no longer
// holds, or its file outlived a change that was made.
func applyOne(t target.Target, id string, yes bool, answers *bufio.Reader, stderr io.Writer) bool {
	pending, err := t.Prepare(id)
	if err != nil {
		say(stderr, "loomux config: %v; the proposal stays\n", err)
		return false
	}
	p, diff := pending.Proposal, pending.Diff
	if diff == "" {
		if err := removeProposalFile(t, id); err != nil {
			say(stderr, "loomux config: proposal %s is already so, but its file could not be removed: %v\n", id, err)
			return false
		}
		say(stderr, "loomux config: proposal %s (%s) is already so; removed\n", id, p.Describe())
		return true
	}
	say(stderr, "proposal %s: %s\n%s", id, p.Describe(), diff)
	if !yes {
		fmt.Fprint(stderr, "write these changes? [y/N] ")
		answer, _ := answers.ReadString('\n')
		if !confirmed(answer) {
			say(stderr, "loomux config: declined; proposal %s stays\n", id)
			return true
		}
	}
	if err := applyWrite(t, pending.Text, pending.Next); err != nil {
		say(stderr, "loomux config: %v; nothing written, the proposal stays\n", err)
		return false
	}
	say(stderr, "loomux config: wrote %s\n", t.Path)
	if hint := writtenHint(t, p.Key); hint != "" {
		fmt.Fprintln(stderr, hint)
	}
	if err := removeProposalFile(t, id); err != nil {
		say(stderr, "loomux config: the change is written, but the proposal file could not be removed: %v; remove it with `loomux config reject %s`\n", err, id)
		return false
	}
	return true
}

func configReject(t target.Target, id string, all bool, stderr io.Writer) int {
	ids, err := t.Choose(id, all)
	for _, id := range ids {
		err = errors.Join(err, t.Reject(id))
	}
	if err != nil {
		say(stderr, "loomux config: %v\n", err)
		return 1
	}
	if len(ids) == 0 {
		fmt.Fprintln(stderr, "loomux config: no proposals open")
		return 0
	}
	say(stderr, "loomux config: rejected %s\n", strings.Join(ids, ", "))
	return 0
}
