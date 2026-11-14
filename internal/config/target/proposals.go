package target

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/xidus90/loomux/internal/config/edit"
)

// linkFile is the seam for the hard link that claims a proposal's name: a
// file system without hard links is not one a test can provoke on a real
// disk.
var linkFile = os.Link

// Proposal is one change an agent asked for and a human has not yet applied.
// It holds what was asked, never a diff: a diff is computed against the file
// as it is whenever one is shown, so a stale or forged one has nowhere to be.
type Proposal struct {
	ID      string `json:"id"`
	Op      string `json:"op"`
	Key     string `json:"key"`
	Input   string `json:"input"`
	Created string `json:"created"`
	Target  string `json:"target"`
}

// ProposalView is a proposal as the proposals listing shows it: with the
// diff it would make to the current file, or why it no longer holds.
type ProposalView struct {
	Proposal
	Diff  string `json:"diff"`
	Error string `json:"error,omitempty"`
	// Unreadable marks a proposal whose file did not load: Error says why,
	// and nothing else in the view is known.
	Unreadable bool `json:"-"`
}

// Describe is the proposal as one line: the operation, the key and, for a
// set, the value.
func (p Proposal) Describe() string {
	if p.Op == "set" {
		return fmt.Sprintf("set %s %s", Word(p.Key), Word(p.Input))
	}
	return Word(p.Op) + " " + Word(p.Key)
}

// Word is a key or a value as one unmistakable word: as typed when it is
// one, quoted with Go escapes when it is empty, holds a blank or a byte a
// terminal would act on.
func Word(s string) string {
	if s != "" && !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || !strconv.IsPrint(r) || r == '"' }) {
		return s
	}
	return strconv.Quote(s)
}

// dir sits in the state directory next to the file it concerns: the
// project's under .loomux/state, which the policy keeps agents' own writes
// out of, and the machine-wide one beside the global config.toml.
func (t Target) dir() string {
	if t.Global {
		return filepath.Join(filepath.Dir(t.Path), "config", "proposals")
	}
	return filepath.Join(filepath.Dir(t.Path), "state", "config", "proposals")
}

func (t Target) name() string {
	if t.Global {
		return "global"
	}
	return "project"
}

func (t Target) file(id string) string {
	return filepath.Join(t.dir(), id+".json")
}

// Resolve is the text a proposal would make of text now, and the diff
// between the two: empty when it is already so.
func (t Target) Resolve(text string, p Proposal) (next, diff string, err error) {
	switch p.Op {
	case "set":
		next, err = t.Change(text, p.Key, p.Input)
	case "unset":
		next, err = t.Unset(text, p.Key)
	default:
		err = fmt.Errorf("unknown operation %s", Word(p.Op))
	}
	if err != nil {
		return "", "", err
	}
	return next, edit.Diff(text, next), nil
}

// Store gives the proposal the first free id of its second, stamps it with
// the time and this target's name, and writes it complete under that name.
func (t Target) Store(now time.Time, p Proposal) (string, error) {
	now = now.UTC()
	p.Created = now.Format(time.RFC3339)
	p.Target = t.name()
	return storeProposal(t.dir(), now, p)
}

// storeProposal writes the proposal under the first free name of its
// second. The link is what makes the name the writer's own: it fails on a
// name another writer took in the meantime, where a rename would silently
// replace that proposal.
func storeProposal(dir string, now time.Time, p Proposal) (string, error) {
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

// IDs are the open proposals by name, oldest first: the name starts with the
// time it was made. A directory that is not there holds none.
func (t Target) IDs() ([]string, error) {
	entries, err := os.ReadDir(t.dir())
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

// OpenCount counts for a hint beside a list; a directory that cannot be read
// shows no hint rather than failing the listing it adorns.
func (t Target) OpenCount() int {
	ids, _ := t.IDs()
	return len(ids)
}

// Load reads one proposal. The file name is its id, whatever the file says:
// that is the name a human typed or saw listed.
func (t Target) Load(id string) (Proposal, error) {
	var p Proposal
	data, err := readFile(t.file(id))
	if err == nil {
		err = json.Unmarshal(data, &p)
	}
	if err != nil {
		return p, fmt.Errorf("proposal %s: %w", id, err)
	}
	p.ID = id
	return p, nil
}

// Choose resolves one id or all to the ids to act on. An id is also a file
// name, so one that could leave the directory is simply not found.
func (t Target) Choose(id string, all bool) ([]string, error) {
	ids, err := t.IDs()
	if err != nil || all {
		return ids, err
	}
	if !slices.Contains(ids, id) {
		return nil, fmt.Errorf("no open proposal %q; see `loomux config proposals`", id)
	}
	return []string{id}, nil
}

// Proposals shows each open proposal with the diff it would make to the
// file as it is now, each on its own: the later ones do not meet the earlier
// ones' result. A proposal file that does not read comes back as a view
// holding the reason, so one broken file hides no other.
func (t Target) Proposals() ([]ProposalView, error) {
	ids, err := t.IDs()
	if err != nil {
		return nil, err
	}
	text, err := t.Read()
	if err != nil {
		return nil, err
	}
	views := []ProposalView{}
	for _, id := range ids {
		p, err := t.Load(id)
		if err != nil {
			views = append(views, ProposalView{Proposal: Proposal{ID: id}, Error: err.Error(), Unreadable: true})
			continue
		}
		v := ProposalView{Proposal: p}
		if _, v.Diff, err = t.Resolve(text, p); err != nil {
			v.Error = err.Error()
		}
		views = append(views, v)
	}
	return views, nil
}

// Pending is a proposal computed against the file as it is now: Text is the
// file, Next what it would become and Diff the difference, empty when the
// file is already so.
type Pending struct {
	Proposal
	Text, Next, Diff string
}

// Prepare computes one proposal again against the current file. A proposal
// that no longer holds is an error naming it.
func (t Target) Prepare(id string) (Pending, error) {
	p, err := t.Load(id)
	if err != nil {
		return Pending{}, err
	}
	text, err := t.Read()
	if err != nil {
		return Pending{}, err
	}
	next, diff, err := t.Resolve(text, p)
	if err != nil {
		return Pending{}, fmt.Errorf("proposal %s (%s): %w", id, p.Describe(), err)
	}
	return Pending{Proposal: p, Text: text, Next: next, Diff: diff}, nil
}

// Apply computes the proposal again against the file as it is now, writes
// it and only then removes the proposal; one that is already so is just
// removed. One that no longer holds stays, with the reason.
func (t Target) Apply(id string) error {
	pending, err := t.Prepare(id)
	if err != nil {
		return err
	}
	return t.ApplyPending(pending)
}

// ApplyPending writes a proposal prepared earlier and removes it. A caller
// that showed pending.Diff to a human applies this very value, so that what
// is written is what was shown; a file that changed since Prepare read it is
// not overwritten.
func (t Target) ApplyPending(pending Pending) error {
	if pending.Diff != "" {
		if err := t.Write(pending.Text, pending.Next); err != nil {
			return err
		}
	}
	return t.Remove(pending.ID)
}

// Reject removes a proposal without applying it.
func (t Target) Reject(id string) error {
	return t.Remove(id)
}

// Remove deletes the file of one proposal.
func (t Target) Remove(id string) error {
	return os.Remove(t.file(id))
}
