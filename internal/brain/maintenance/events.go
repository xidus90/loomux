// Package maintenance holds the reconciliation layer's own state: the review
// case a person later decides (case.go), and the log of landed merges here,
// which is the second trigger a case can come from.
//
// Of the log, only the reading and the forgetting are here. The one writer of the log is a
// POSIX sh `post-merge` hook that records the repository, the commit range and
// the branch and ends with an unconditional `exit 0` -- it may not block a
// merge, may not fail it and may not start anything that outlives it, so it
// appends one line and leaves. Installing that hook is stage 4; a Go
// `RecordEvent` would today have no caller but its own test.
//
// The original is `src/brain/maintenance/merge_events.py`.
package maintenance

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// eventFields and keyFields are `_FIELDS` and `_KEY_FIELDS`: a line of the log
// carries five tab-separated fields, a line of the dropped file three.
const (
	eventFields = 5
	keyFields   = 3
)

// eventsRelative and droppedRelative are `events_path` and `_dropped_path`,
// beside the reconciliation's own state because that is what consumes them.
var (
	eventsRelative  = filepath.Join("maintenance", "merge-events.tsv")
	droppedRelative = filepath.Join("maintenance", "merge-events.done.tsv")
)

// MergeEvent is one landed merge, in the facts a case can be built from.
//
// Repo is a string and no path type: it comes off a text file the hook wrote,
// and the reader must not silently normalise what git reported. There is no
// scope and no area here either -- which area a merge belongs to is derived
// from the repository at reconciliation time (`reconcile._resolve_event`), and
// a field for it would be a second, ageing copy of that answer.
type MergeEvent struct {
	Repo   string
	First  string
	Last   string
	Branch string
	At     time.Time
}

// Key is what makes two records the same merge: the range, in one repository
// (`MergeEvent.key`). The branch and the stamp are deliberately out of it --
// git runs the hook again for a re-merge, and the second line differs in the
// stamp alone while saying the same thing.
func (e MergeEvent) Key() [3]string {
	return [3]string{e.Repo, e.First, e.Last}
}

// EventsPath is where the log is read from: stateDir first and, as long as
// nothing lies there, fallbackDir -- ultra-brain's, whose hook writes into it.
//
// Both directories are arguments and neither is read from the environment, so
// that a caller's state directory is the only one this reads (internal/serve's
// promise, and the shape `search.ReadLastRun` takes for the stamp next door).
func EventsPath(stateDir, fallbackDir string) string {
	return config.ArtifactLookup{Primary: stateDir, Fallback: fallbackDir}.Resolve(eventsRelative)
}

// ReadEvents is every merge still waiting for a case, each range only once.
//
// Two merges of the same range are one event, and the first line wins: a
// second case over the same commits would say the same thing twice, and the
// later stamp is the re-run rather than the landing.
//
// A line that does not parse is skipped rather than raised over -- too few
// fields, too many, or a stamp that is no stamp. The file is written by a
// shell hook that must never fail, so a truncated line is a thing that
// happens, and taking the whole reconciliation down over it would turn a lost
// merge into a lost run. Bytes that are not UTF-8 are the one failure that
// does come back as an error: Python ends in a UnicodeDecodeError there, and
// answering "no merge ever happened" would be the worse lie.
func ReadEvents(stateDir, fallbackDir string) ([]MergeEvent, error) {
	seen, err := dropped(stateDir, fallbackDir)
	if err != nil {
		return nil, err
	}
	lines, err := readLines(EventsPath(stateDir, fallbackDir))
	if err != nil {
		return nil, err
	}
	var events []MergeEvent
	for _, line := range lines {
		event, ok := parseEvent(line)
		if !ok {
			continue
		}
		if seen[event.Key()] {
			continue
		}
		seen[event.Key()] = true
		events = append(events, event)
	}
	return events, nil
}

// DropEvent forgets one event, once its case exists -- only here, and never on
// reading: an event dropped before the case is written is a merge nobody will
// ever hear about again.
//
// Written as an append to a second file rather than as a deletion from the
// log. The hook appends to the log from inside a merge, with no lock anybody
// could take, and a read-modify-write here would overwrite whatever landed in
// between -- which loses a merge for good. Both writers only ever append, so
// neither can lose the other's line, and both files grow monotonically at
// roughly a hundred bytes per merge. Nothing compacts them.
//
// The price is a reversed failure direction: losing the dropped file raises
// every merge ever recorded a second time, where losing the log under a
// deleting design would merely forget what was already done. Duplicated cases
// are noise a reviewer rejects; a forgotten merge is knowledge nobody gets
// back, so the noisy direction is the chosen one.
//
// No fallbackDir: writing happens only to the new place. The reading pair of
// this stage then prefers it, so the drops a Python `brain reconcile` wrote
// into the old directory stop counting from the first drop here on -- every
// merge it had already handled resurfaces once and is dropped again. Stage 4's
// `migrate` carries the file over; until then that is the self-healing
// direction of the two.
func DropEvent(stateDir string, event MergeEvent) error {
	key := event.Key()
	path := config.ArtifactLookup{Primary: stateDir}.WritePath(droppedRelative)
	return appendLine(path, strings.Join(key[:], "\t")+"\n")
}

// dropped is the set of ranges whose case exists (`_dropped`). A line that is
// no triple is ignored, for the reason a malformed log line is.
func dropped(stateDir, fallbackDir string) (map[[3]string]bool, error) {
	path := config.ArtifactLookup{Primary: stateDir, Fallback: fallbackDir}.Resolve(droppedRelative)
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}
	keys := make(map[[3]string]bool, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) == keyFields {
			keys[[3]string{fields[0], fields[1], fields[2]}] = true
		}
	}
	return keys, nil
}

// parseEvent is `_parse_event`: five fields and a stamp that reads, or no
// event at all. An unreadable stamp makes the whole line a broken one rather
// than an event without a time -- the stamp is the last field the hook writes,
// so a half-written line ends exactly there.
func parseEvent(line string) (MergeEvent, bool) {
	fields := strings.Split(line, "\t")
	if len(fields) != eventFields {
		return MergeEvent{}, false
	}
	at, ok := pytext.ParseAwareIsoFormat(fields[4])
	if !ok {
		return MergeEvent{}, false
	}
	return MergeEvent{
		Repo: fields[0], First: fields[1], Last: fields[2], Branch: fields[3], At: at,
	}, true
}

// readLines is `_lines`: the file's non-empty lines, or nothing at all when it
// is not a file. `is_file()` and not `exists()`, so a directory in its place is
// an absent log rather than a failed run.
func readLines(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil
	}
	if !info.Mode().IsRegular() {
		return nil, nil
	}
	text, err := pytext.ReadText(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range pytext.SplitLines(text) {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// appendLine is `_append`: the directory made, the line added, nothing read
// first. The open mode is the whole point -- two writers, no lock, and an
// append is the one write neither can lose for the other.
func appendLine(path, line string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return writeAndClose(file, line)
}

// writeAndClose puts one line into an open file and closes it either way.
//
// Split out of appendLine so that the failed write has a caller a test can
// reach: appendLine opens the handle itself, and a handle it opened for
// appending does not refuse the write. A handle opened for reading does.
func writeAndClose(file *os.File, line string) error {
	if _, err := file.WriteString(line); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
