package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/config/edit"
)

// The seams for the file system calls whose failure a test cannot provoke
// on a real disk: a file system without hard links, a proposal file that
// cannot be removed and a write that fails after the file was read.
var (
	linkFile           = os.Link
	removeProposalFile = os.Remove
	applyWrite         = writeConfig
)

// proposal is one change an agent asked for and a human has not yet applied.
// It holds what was asked, never a diff: a diff is computed against the file
// as it is whenever one is shown, so a stale or forged one has nowhere to be.
type proposal struct {
	ID      string `json:"id"`
	Op      string `json:"op"`
	Key     string `json:"key"`
	Input   string `json:"input"`
	Created string `json:"created"`
	Target  string `json:"target"`
}

// proposalView is a proposal as `config proposals --json` shows it: with the
// diff it would make to the current file, or why it no longer holds.
type proposalView struct {
	proposal
	Diff  string `json:"diff"`
	Error string `json:"error,omitempty"`
}

// proposalDir sits in the state directory next to the file it concerns: the
// project's under .loomux/state, which the policy keeps agents' own writes
// out of, and the machine-wide one beside the global config.toml.
func (t configTarget) proposalDir() string {
	if t.global {
		return filepath.Join(filepath.Dir(t.path), "config", "proposals")
	}
	return filepath.Join(filepath.Dir(t.path), "state", "config", "proposals")
}

func (t configTarget) targetName() string {
	if t.global {
		return "global"
	}
	return "project"
}

// compute is the change a proposal stands for, against text as it is now.
func (t configTarget) compute(text string, p proposal) (string, error) {
	switch p.Op {
	case "set":
		return proposeChange(t, text, p.Key, p.Input)
	case "unset":
		return proposeUnset(t, text, p.Key)
	}
	return "", fmt.Errorf("unknown operation %s", word(p.Op))
}

// change is the text a proposal would make of text now, and the diff
// between the two: empty when it is already so.
func (t configTarget) change(text string, p proposal) (next, diff string, err error) {
	next, err = t.compute(text, p)
	if err != nil {
		return "", "", err
	}
	return next, edit.Diff(text, next), nil
}

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

// word is a key or a value as one unmistakable word: as typed when it is
// one, quoted with Go escapes when it is empty, holds a blank or a byte a
// terminal would act on.
func word(s string) string {
	if s != "" && !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || !strconv.IsPrint(r) || r == '"' }) {
		return s
	}
	return strconv.Quote(s)
}

// configPropose is set and unset with --propose: the same computation and
// the same refusals, but the change is stored for a human instead of written.
func configPropose(t configTarget, p proposal, stdout, stderr io.Writer) int {
	text, err := t.read()
	var diff string
	if err == nil {
		_, diff, err = t.change(text, p)
	}
	if err != nil {
		say(stderr, "loomux config: %v\n", err)
		return 1
	}
	if diff == "" {
		fmt.Fprintln(stderr, "loomux config: already so; nothing proposed")
		return 0
	}
	now := time.Now().UTC()
	p.Created = now.Format(time.RFC3339)
	p.Target = t.targetName()
	id, err := storeProposal(t.proposalDir(), now, p)
	if err != nil {
		say(stderr, "loomux config: %v\n", err)
		return 1
	}
	say(stdout, "%s", diff)
	fmt.Fprintf(stdout, "proposal %s stored; a human applies it with `loomux config apply %s`\n", id, id)
	return 0
}

// storeProposal gives the proposal the first free id of its second and
// writes it complete under that name. The link is what makes the name the
// writer's own: it fails on a name another writer took in the meantime,
// where a rename would silently replace that proposal.
func storeProposal(dir string, now time.Time, p proposal) (string, error) {
	// A directory that cannot be made fails the temporary file below, with
	// the path that could not be made in its message.
	_ = os.MkdirAll(dir, 0o755)
	stamp := now.Format("20060102T150405Z")
	for n := 1; ; n++ {
		p.ID = fmt.Sprintf("%s-%03d", stamp, n)
		data, _ := json.MarshalIndent(p, "", "  ")
		data = append(data, '\n')
		temporary, err := writeTemporaryProposal(dir, data)
		if err != nil {
			return "", err
		}
		final := filepath.Join(dir, p.ID+".json")
		err = linkFile(temporary, final)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			// FAT and some network shares have no hard links. An exclusive
			// create claims the name as surely, at the price of a moment in
			// which a reader finds the file still empty.
			err = createExclusive(final, data)
		}
		os.Remove(temporary)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return p.ID, err
	}
}

// writeTemporaryProposal writes data to a file of its own in dir.
//
//coverage:exempt the Write and Close error arms need a write to a freshly created temporary file to fail (a full disk); the CreateTemp arm is covered
func writeTemporaryProposal(dir string, data []byte) (string, error) {
	file, err := os.CreateTemp(dir, "proposal.*.tmp")
	if err != nil {
		return "", err
	}
	_, err = file.Write(data)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

// createExclusive writes data to a file that must not exist yet.
//
//coverage:exempt the Write and Close error arms need a write to a freshly created file to fail (a full disk); the OpenFile arms, free and taken, are covered
func createExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}

// proposalIDs are the open proposals by name, oldest first: the name starts
// with the time it was made. A directory that is not there holds none.
func (t configTarget) proposalIDs() ([]string, error) {
	entries, err := os.ReadDir(t.proposalDir())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if id, ok := strings.CutSuffix(e.Name(), ".json"); ok && !e.IsDir() {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

// openProposals counts for the hint in list and the form; a directory that
// cannot be read shows no hint rather than failing the listing it adorns.
func (t configTarget) openProposals() int {
	ids, _ := t.proposalIDs()
	return len(ids)
}

func proposalHint(n int) string {
	if n == 1 {
		return "1 proposal open — loomux config proposals"
	}
	return fmt.Sprintf("%d proposals open — loomux config proposals", n)
}

// loadProposal reads one proposal. The file name is its id, whatever the
// file says: that is the name a human typed or saw listed.
func (t configTarget) loadProposal(id string) (proposal, error) {
	var p proposal
	data, err := os.ReadFile(t.proposalFile(id))
	if err == nil {
		err = json.Unmarshal(data, &p)
	}
	if err != nil {
		return p, fmt.Errorf("proposal %s: %w", id, err)
	}
	p.ID = id
	return p, nil
}

func (t configTarget) proposalFile(id string) string {
	return filepath.Join(t.proposalDir(), id+".json")
}

// chosenProposals resolves <id> or --all to the ids to act on. An id is also
// a file name, so one that could leave the directory is simply not found.
func (t configTarget) chosenProposals(id string, all bool) ([]string, error) {
	ids, err := t.proposalIDs()
	if err != nil || all {
		return ids, err
	}
	if !slices.Contains(ids, id) {
		return nil, fmt.Errorf("no open proposal %q; see `loomux config proposals`", id)
	}
	return []string{id}, nil
}

// configProposals shows each proposal with the diff it would make to the
// file as it is now, each on its own: under --all the later ones meet the
// earlier ones' result instead. A file that does not read is named and the
// others are still shown.
func configProposals(t configTarget, asJSON bool, stdout, stderr io.Writer) int {
	ids, err := t.proposalIDs()
	var text string
	if err == nil {
		text, err = t.read()
	}
	if err != nil {
		say(stderr, "loomux config: %v\n", err)
		return 1
	}
	code := 0
	views := []proposalView{}
	for _, id := range ids {
		p, err := t.loadProposal(id)
		if err != nil {
			say(stderr, "loomux config: %v\n", err)
			code = 1
			continue
		}
		v := proposalView{proposal: p}
		if _, v.Diff, err = t.change(text, p); err != nil {
			v.Error = err.Error()
		}
		views = append(views, v)
	}
	if asJSON {
		// JSON escapes the C0 controls itself but passes DEL and the C1
		// range as they are; a reader who cats the output meets them raw.
		for i, v := range views {
			views[i] = proposalView{proposal{clean(v.ID), clean(v.Op), clean(v.Key), clean(v.Input), clean(v.Created), clean(v.Target)}, clean(v.Diff), clean(v.Error)}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(views)
		return code
	}
	for _, v := range views {
		say(stdout, "%s  %s  %s\n", v.ID, v.Created, describeProposal(v.proposal))
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

func describeProposal(p proposal) string {
	if p.Op == "set" {
		return fmt.Sprintf("set %s %s", word(p.Key), word(p.Input))
	}
	return word(p.Op) + " " + word(p.Key)
}

// configApply is the human's half: each proposal is computed again against
// the file as it is now, shown, confirmed and written the way `config set`
// writes, and only then removed. A proposal that no longer holds stays.
func configApply(t configTarget, id string, all, yes bool, stdin io.Reader, stderr io.Writer) int {
	ids, err := t.chosenProposals(id, all)
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
func applyOne(t configTarget, id string, yes bool, answers *bufio.Reader, stderr io.Writer) bool {
	p, err := t.loadProposal(id)
	var text, next, diff string
	if err == nil {
		text, err = t.read()
	}
	if err == nil {
		next, diff, err = t.change(text, p)
		if err != nil {
			err = fmt.Errorf("proposal %s (%s): %w", id, describeProposal(p), err)
		}
	}
	if err != nil {
		say(stderr, "loomux config: %v; the proposal stays\n", err)
		return false
	}
	if diff == "" {
		if err := removeProposalFile(t.proposalFile(id)); err != nil {
			say(stderr, "loomux config: proposal %s is already so, but its file could not be removed: %v\n", id, err)
			return false
		}
		say(stderr, "loomux config: proposal %s (%s) is already so; removed\n", id, describeProposal(p))
		return true
	}
	say(stderr, "proposal %s: %s\n%s", id, describeProposal(p), diff)
	if !yes {
		fmt.Fprint(stderr, "write these changes? [y/N] ")
		answer, _ := answers.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			say(stderr, "loomux config: declined; proposal %s stays\n", id)
			return true
		}
	}
	if err := applyWrite(t, next); err != nil {
		say(stderr, "loomux config: %v; nothing written, the proposal stays\n", err)
		return false
	}
	say(stderr, "loomux config: wrote %s\n", t.path)
	if err := removeProposalFile(t.proposalFile(id)); err != nil {
		say(stderr, "loomux config: the change is written, but the proposal file could not be removed: %v; remove it with `loomux config reject %s`\n", err, id)
		return false
	}
	return true
}

func configReject(t configTarget, id string, all bool, stderr io.Writer) int {
	ids, err := t.chosenProposals(id, all)
	for _, id := range ids {
		err = errors.Join(err, os.Remove(t.proposalFile(id)))
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
