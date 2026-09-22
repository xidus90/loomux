package maintenance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// caseStates and caseTriggers are `STATES` and `TRIGGERS`. A misspelt state or
// trigger must fail loudly rather than fall back to a guess.
var (
	caseStates   = []string{"current", "source_changed", "due", "source_missing", "in_review"}
	caseTriggers = []string{"source_change", "merge"}
)

// SourceState is one source a case was opened over, in the shape reconcile
// compares against on the next pass.
type SourceState struct {
	DocID       string
	Revision    int
	ContentHash string
}

// Case is one review case, the file a person later decides.
//
// Note, SupersededProposal and PromptVersion are strings and no pointers:
// "" is the absent key, the way config.Area spells an absent `wiki`. The
// renderer leaves an empty one out, so a case that never had a note keeps the
// file it had before the field existed.
type Case struct {
	ID         string
	Area       string
	Target     string
	TargetHash string
	State      string
	Trigger    string
	Weight     string
	Created    time.Time
	Sources    []SourceState

	// Note and SupersededProposal are set when a case was rebuilt while a
	// proposal already stood, or when the target moved beneath it. Both are
	// recorded rather than dropped: prose in a file no code reads is no record.
	Note               string
	SupersededProposal string

	// Manual says there is no skill path for this case, whatever the reason.
	// It never said "closed area" on its own, and a closed area with a local
	// proposal carries false here, so it says nothing about where the case may
	// travel either.
	Manual bool

	// LocalOnly is the area's privacy mode, regardless of whether a proposal
	// came about: the switch that keeps a source diff away from a cloud model
	// must hang neither on a substring someone can reword nor on a mark that an
	// answer from the local model clears.
	LocalOnly bool

	// PromptVersion names which prompt produced the proposal beside this file,
	// set only when one came about. A proposal stays readable a second time for
	// as long as the reader knows the version that asked for it.
	PromptVersion string
}

// CaseID builds a deterministic id so a repeated pass never opens the same
// case twice (`case_id`).
//
// The digest is taken over area and target together, not over target alone --
// otherwise two areas reviewing an identically named file on the same day
// would collide on one case id.
func CaseID(area, target string, now time.Time) string {
	lastSegment := area
	if cut := strings.LastIndex(area, "/"); cut >= 0 {
		lastSegment = area[cut+1:]
	}
	digest := sha256.Sum256([]byte(area + "\n" + target))
	return fmt.Sprintf("%s-%s-%s", lastSegment, now.Format("2006-01-02"), hex.EncodeToString(digest[:])[:4])
}

// CaseDir is `<review root>/<scope>/<case>/`, the directory one case owns.
func CaseDir(reviewRoot, scope, id string) string {
	return filepath.Join(reviewRoot, scope, id)
}

// WriteCase renders the case in a fixed field order and swaps the file in,
// and answers whether anything actually changed (`write_case`).
//
// The fixed order is what makes two writes of one case byte-identical, which
// the merge-into-index step depends on to skip unchanged cases. A TOML encoder
// sorts its keys and could not hold that order, so the rendering is done here.
//
// The answer is read by the step that reports a case file as an uncommitted
// change when a run breaks off. A second identical write must not land in that
// report -- it would send the reader looking for a change git will not show
// them.
//
// The state and the trigger are held to the same closed vocabularies ReadCase
// holds them to. `write_case` does not check them, and `Case.state` is a plain
// `str` there rather than a `Literal` -- but that is an omission of the
// reference, not a contract: without the check this writes a file its own
// reader refuses, and a case is decided by a person reading that file. It
// touches nothing that is rendered, so the byte parity of an accepted case is
// untouched.
func WriteCase(path string, c Case) (bool, error) {
	for _, vocabulary := range []struct {
		key     string
		value   string
		allowed []string
	}{
		{"state", c.State, caseStates},
		{"trigger", c.Trigger, caseTriggers},
	} {
		if !slices.Contains(vocabulary.allowed, vocabulary.value) {
			return false, outsideVocabulary(path, vocabulary.key, vocabulary.value, vocabulary.allowed)
		}
	}
	if c.Created.IsZero() {
		// Python refuses a `created` without a zone here (`write_case`),
		// because a naive value would run through TOML's local-datetime form
		// and come back reinterpreted in whatever zone the reader runs in.
		// Go has no naive form to refuse: every time.Time carries a Location
		// and pytext.IsoFormat always writes an offset. What is left in that
		// slot is the weaker guard of a case nobody gave a point in time --
		// the read side below is where the zone rule still has teeth.
		return false, fmt.Errorf("%s: created must name a point in time, found the zero value", path)
	}
	text := renderCase(c)
	// os.ReadFile and not pytext.ReadText: that one folds CRLF into LF, and a
	// standing file whose bytes differ only in its line endings would compare
	// equal and never be rewritten. An unreadable file counts as changed, and
	// the write below then surfaces whatever is really wrong with it.
	if standing, err := os.ReadFile(path); err == nil && string(standing) == text {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := lock.ReplaceText(path, text); err != nil {
		return false, err
	}
	return true, nil
}

// renderCase is the file's fixed order: the eight mandatory keys, then the
// optional ones, then one `[[sources]]` block per source.
//
// A flag is written only when it is true, so the file of an ordinary case is
// exactly the one it was before the flag existed.
func renderCase(c Case) string {
	var out strings.Builder
	for _, field := range []struct{ key, value string }{
		{"id", c.ID},
		{"area", c.Area},
		{"target", c.Target},
		{"target_hash", c.TargetHash},
		{"state", c.State},
		{"trigger", c.Trigger},
		{"weight", c.Weight},
	} {
		fmt.Fprintf(&out, "%s = %s\n", field.key, config.QuoteTOML(field.value))
	}
	// pytext.IsoFormat and not time.RFC3339: Python writes `+00:00` where Go
	// writes `Z`, and a case file that differed in that one character would be
	// rewritten, and reported as changed, on every pass across the two sides.
	fmt.Fprintf(&out, "created = %s\n", pytext.IsoFormat(c.Created))
	for _, field := range []struct{ key, value string }{
		{"note", c.Note},
		{"superseded_proposal", c.SupersededProposal},
	} {
		if field.value != "" {
			fmt.Fprintf(&out, "%s = %s\n", field.key, config.QuoteTOML(field.value))
		}
	}
	for _, flag := range []struct {
		key string
		set bool
	}{
		{"manual", c.Manual},
		{"local_only", c.LocalOnly},
	} {
		if flag.set {
			fmt.Fprintf(&out, "%s = true\n", flag.key)
		}
	}
	if c.PromptVersion != "" {
		fmt.Fprintf(&out, "prompt_version = %s\n", config.QuoteTOML(c.PromptVersion))
	}
	for _, source := range c.Sources {
		fmt.Fprintf(&out, "\n[[sources]]\ndoc_id = %s\nrevision = %d\ncontent_hash = %s\n",
			config.QuoteTOML(source.DocID), source.Revision, config.QuoteTOML(source.ContentHash))
	}
	return out.String()
}

// ReadCase reads one case file and refuses one it cannot use whole
// (`read_case`).
//
// No skipping and no defaulting: a case whose state or trigger is unknown, or
// whose stamp carries no zone, would be decided by a person on what the file
// seems to say rather than on what it says.
func ReadCase(path string) (Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Case{}, err
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		return Case{}, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	return caseReading{path: path, data: document}.read()
}

// caseReading carries the file name into every refusal, so a person who has to
// repair a case file is told which one.
type caseReading struct {
	path string
	data map[string]any
}

// read assembles the case, in the order the refusals are cheapest to
// understand in: the two closed vocabularies, then the stamp, then the rest.
func (r caseReading) read() (Case, error) {
	c := Case{}
	var err error
	if c.State, err = r.member("state", caseStates); err != nil {
		return Case{}, err
	}
	if c.Trigger, err = r.member("trigger", caseTriggers); err != nil {
		return Case{}, err
	}
	if c.Created, err = r.created(); err != nil {
		return Case{}, err
	}
	if c.Sources, err = r.sources(); err != nil {
		return Case{}, err
	}
	for _, field := range []struct {
		key    string
		into   *string
		wanted func(string) (string, error)
	}{
		{"id", &c.ID, r.required},
		{"area", &c.Area, r.required},
		{"target", &c.Target, r.required},
		{"target_hash", &c.TargetHash, r.required},
		{"weight", &c.Weight, r.required},
		{"note", &c.Note, r.optional},
		{"superseded_proposal", &c.SupersededProposal, r.optional},
		{"prompt_version", &c.PromptVersion, r.optional},
	} {
		if *field.into, err = field.wanted(field.key); err != nil {
			return Case{}, err
		}
	}
	for _, flag := range []struct {
		key  string
		into *bool
	}{
		{"manual", &c.Manual},
		{"local_only", &c.LocalOnly},
	} {
		if *flag.into, err = r.flag(flag.key); err != nil {
			return Case{}, err
		}
	}
	return c, nil
}

// member is a required string out of a closed vocabulary.
func (r caseReading) member(key string, allowed []string) (string, error) {
	value, err := r.required(key)
	if err != nil {
		return "", err
	}
	if !slices.Contains(allowed, value) {
		return "", outsideVocabulary(r.path, key, value, allowed)
	}
	return value, nil
}

// outsideVocabulary is the one wording both sides use for a value outside a
// closed set, so a case refused on writing and one refused on reading name the
// same defect in the same words.
func outsideVocabulary(path, key, value string, allowed []string) error {
	return fmt.Errorf("%s: %s must be one of %s, found %q", path, key, strings.Join(allowed, ", "), value)
}

// required is `_require_str`: the key has to be there and carry a non-empty
// string.
//
// Two statements rather than one condition joined by `||`: Go measures
// statements, so a single `if` would report both arms covered as soon as
// either of them had been taken once.
func (r caseReading) required(key string) (string, error) {
	value, ok := r.data[key].(string)
	if !ok {
		return "", r.errorf("%s must be a non-empty string, found %s", key, caseType(r.data[key]))
	}
	if value == "" {
		return "", r.errorf("%s must be a non-empty string, found an empty one", key)
	}
	return value, nil
}

// optional answers "" for an absent key and refuses a present one that is no
// string.
//
// Python hands a non-string `note` through untyped; a Go string field has no
// room for that, and the refusal is the honest reading of the two.
func (r caseReading) optional(key string) (string, error) {
	value, present := r.data[key]
	if !present {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", r.errorf("%s must be a string, found %s", key, caseType(value))
	}
	return text, nil
}

// flag is `_flag`: an absent flag is false, and anything but a boolean is a
// defect rather than a truth.
//
// A hand-edited `local_only = "ja"` would be truthy in a looser reader -- in
// the field that decides whether a source diff may reach a cloud model, the
// silent reading has to be the strict one.
func (r caseReading) flag(key string) (bool, error) {
	value, present := r.data[key]
	if !present {
		return false, nil
	}
	set, ok := value.(bool)
	if !ok {
		return false, r.errorf("%s must be a boolean, found %s", key, caseType(value))
	}
	return set, nil
}

// created is the stamp, refused unless it carries a zone of its own.
func (r caseReading) created() (time.Time, error) {
	value, ok := r.data["created"].(time.Time)
	if !ok {
		return time.Time{}, r.errorf("created must be a TOML datetime, found %s", caseType(r.data["created"]))
	}
	if zoneless(value) {
		return time.Time{}, r.errorf("created must carry an offset, found %s", value.Format("2006-01-02T15:04:05"))
	}
	return value, nil
}

// zoneless says whether a decoded date or time brought no offset of its own.
//
// This is where Python's refusal of a naive `created` lands (`read_case`): a
// hand-edited case.toml is the normal case, not the exception, and tomllib
// answers a naive datetime for TOML's local date-time and a plain date or time
// object for the two shorter forms. The decoder here answers a time.Time for
// all three and marks them by their Location: third_party/toml/internal/tz.go
// builds the fixed zones "datetime-local", "date-local" and "time-local" at
// whatever offset this machine happens to run at, which is exactly the silent
// reinterpretation Python refuses. A stamp that does carry an offset never
// lands in one of them: "Z" reads as UTC, "+HH:MM" as an unnamed fixed zone,
// and, where the offset happens to equal this machine's, as time.Local -- the
// parser hands RFC 3339 to time.ParseInLocation in the local zone
// (third_party/toml/parse.go:349). None of the three carries one of the names
// above.
func zoneless(value time.Time) bool {
	switch value.Location().String() {
	case "datetime-local", "date-local", "time-local":
		return true
	}
	return false
}

// sources is the `[[sources]]` array, each entry checked like the top level.
func (r caseReading) sources() ([]SourceState, error) {
	entries, err := r.sourceEntries()
	if err != nil {
		return nil, err
	}
	var sources []SourceState
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, r.errorf("sources must be an array of [[sources]] tables, found %s in it", caseType(raw))
		}
		source, err := r.source(entry)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

// source is one entry of the array.
func (r caseReading) source(entry map[string]any) (SourceState, error) {
	inner := caseReading{path: r.path, data: entry}
	docID, err := inner.required("doc_id")
	if err != nil {
		return SourceState{}, err
	}
	revision, err := inner.requiredInt("revision")
	if err != nil {
		return SourceState{}, err
	}
	contentHash, err := inner.required("content_hash")
	if err != nil {
		return SourceState{}, err
	}
	return SourceState{DocID: docID, Revision: revision, ContentHash: contentHash}, nil
}

// requiredInt is `_require_int`. Python excludes bool there explicitly,
// because a Python bool passes `isinstance(value, int)`; nothing of that kind
// is needed here, since a decoded TOML boolean is a bool and fails the
// assertion to int64 on its own.
func (r caseReading) requiredInt(key string) (int, error) {
	value, ok := r.data[key].(int64)
	if !ok {
		return 0, r.errorf("%s must be an integer, found %s", key, caseType(r.data[key]))
	}
	return int(value), nil
}

// sourceEntries is the `sources` value in either shape the decoder answers
// with: []map[string]any when every element is a table, []any otherwise.
func (r caseReading) sourceEntries() ([]any, error) {
	switch value := r.data["sources"].(type) {
	case nil:
		return nil, nil
	case []map[string]any:
		entries := make([]any, len(value))
		for i, entry := range value {
			entries[i] = entry
		}
		return entries, nil
	case []any:
		return value, nil
	default:
		return nil, r.errorf("sources must be an array of [[sources]] tables, found %s", caseType(value))
	}
}

// errorf names the file first, the way the rest of this binary refuses one.
// No error type of its own: nothing decides anything on a case failure beyond
// reporting it, and `CaseError` would be a type with no caller.
func (r caseReading) errorf(format string, args ...any) error {
	return fmt.Errorf("%s: %s", r.path, fmt.Sprintf(format, args...))
}

// caseType names a decoded value by its TOML type, the vocabulary a person who
// wrote the file can find in it. An absent key is no type at all.
func caseType(value any) string {
	switch value.(type) {
	case nil:
		return "no value"
	case string:
		return "a string"
	case int64:
		return "an integer"
	case float64:
		return "a float"
	case bool:
		return "a boolean"
	case time.Time:
		return "a datetime"
	case map[string]any:
		return "a table"
	}
	return "an array"
}
