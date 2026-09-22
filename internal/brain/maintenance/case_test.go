package maintenance_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// plainCase is the case every test starts from: the eight mandatory fields,
// no flag, no note, no source.
func plainCase() maintenance.Case {
	return maintenance.Case{
		ID: "a", Area: "project/a", Target: "a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC),
	}
}

// writeCase writes into a fresh directory and fails the test on an error.
func writeCase(t *testing.T, c maintenance.Case) (string, bool) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "case.toml")
	changed, err := maintenance.WriteCase(path, c)
	if err != nil {
		t.Fatalf("WriteCase: %v", err)
	}
	return path, changed
}

// caseFile puts a hand-written case.toml where ReadCase will look for it.
func caseFile(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "case.toml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestCaseIDIsDeterministic(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	first := maintenance.CaseID("project/loomux", "docs/wiki/a.md", now)
	second := maintenance.CaseID("project/loomux", "docs/wiki/a.md", now)
	if first != second {
		t.Fatalf("%q != %q", first, second)
	}
	if !strings.HasPrefix(first, "loomux-2026-09-19-") {
		t.Fatalf("id = %q", first)
	}
}

// The digest runs over area and target together: two areas reviewing an
// identically named file would collide on one id on the same day otherwise.
//
// The two areas share their last segment on purpose. That segment is the id's
// prefix, so two areas that differ in it already differ in the prefix and the
// digest is never reached -- a test built that way passes even when the digest
// is taken over the target alone, which is the collision `case_id` names.
func TestCaseIDSeparatesTwoAreasWithTheSameLastSegment(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	a := maintenance.CaseID("vault/loomux", "a.md", now)
	b := maintenance.CaseID("project/loomux", "a.md", now)
	if a == b {
		t.Fatalf("two areas produced one id: %q", a)
	}
}

// An area without a slash is its own last segment, the way `rsplit` answers.
func TestCaseIDTakesAnAreaWithoutASlashWhole(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	if id := maintenance.CaseID("loomux", "a.md", now); !strings.HasPrefix(id, "loomux-2026-09-19-") {
		t.Fatalf("id = %q", id)
	}
}

// A slash at the very front is still the last one: `rsplit("/", 1)` cuts
// there and keeps what follows, so the prefix is the name without it.
func TestCaseIDCutsAtALeadingSlash(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	if id := maintenance.CaseID("/loomux", "a.md", now); !strings.HasPrefix(id, "loomux-2026-09-19-") {
		t.Fatalf("id = %q", id)
	}
}

func TestCaseDirJoinsScopeAndCase(t *testing.T) {
	got := maintenance.CaseDir(filepath.Join("vault", "95 Pruefzentrum"), "project/a", "a-2026-09-19-abcd")
	want := filepath.Join("vault", "95 Pruefzentrum", "project/a", "a-2026-09-19-abcd")
	if got != want {
		t.Fatalf("CaseDir = %q, want %q", got, want)
	}
}

func TestWriteCaseRoundTrips(t *testing.T) {
	want := maintenance.Case{
		ID: "loomux-2026-09-19-abcd", Area: "project/loomux",
		Target: "docs/wiki/a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC),
		Sources: []maintenance.SourceState{{DocID: "d1", Revision: 3, ContentHash: "sha256:bb"}},
	}
	path, _ := writeCase(t, want)
	got, err := maintenance.ReadCase(path)
	if err != nil {
		t.Fatalf("ReadCase: %v", err)
	}
	if got.ID != want.ID || got.Target != want.Target || len(got.Sources) != 1 {
		t.Fatalf("case = %+v", got)
	}
	if got.Sources[0] != want.Sources[0] {
		t.Fatalf("source = %+v", got.Sources[0])
	}
	if !got.Created.Equal(want.Created) {
		t.Fatalf("created = %v, want %v", got.Created, want.Created)
	}
}

// Every optional field set: the second half of the round trip, and the only
// test that reaches each `if` in the renderer.
func TestWriteCaseRoundTripsEveryOptionalField(t *testing.T) {
	want := plainCase()
	want.Note = "the target moved beneath it"
	want.SupersededProposal = "proposal-2026-09-18.md"
	want.Manual = true
	want.LocalOnly = true
	want.PromptVersion = "v3"
	want.Sources = []maintenance.SourceState{{DocID: "d1", Revision: 3, ContentHash: "sha256:bb"}}
	path, _ := writeCase(t, want)
	got, err := maintenance.ReadCase(path)
	if err != nil {
		t.Fatalf("ReadCase: %v", err)
	}
	if got.Note != want.Note || got.SupersededProposal != want.SupersededProposal {
		t.Fatalf("case = %+v", got)
	}
	if !got.Manual || !got.LocalOnly || got.PromptVersion != want.PromptVersion {
		t.Fatalf("case = %+v", got)
	}
}

// The field order is fixed, so two writes of one case are byte-identical and
// so a case file stays comparable with the one Python writes.
func TestWriteCaseRendersTheFixedOrder(t *testing.T) {
	c := plainCase()
	c.Note = "n"
	c.SupersededProposal = "p.md"
	c.Manual = true
	c.LocalOnly = true
	c.PromptVersion = "v3"
	c.Sources = []maintenance.SourceState{{DocID: "d1", Revision: 3, ContentHash: "sha256:bb"}}
	path, _ := writeCase(t, c)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := strings.Join([]string{
		`id = "a"`,
		`area = "project/a"`,
		`target = "a.md"`,
		`target_hash = "sha256:aa"`,
		`state = "source_changed"`,
		`trigger = "source_change"`,
		`weight = "change"`,
		`created = 2026-09-19T10:00:00+00:00`,
		`note = "n"`,
		`superseded_proposal = "p.md"`,
		`manual = true`,
		`local_only = true`,
		`prompt_version = "v3"`,
		``,
		`[[sources]]`,
		`doc_id = "d1"`,
		`revision = 3`,
		`content_hash = "sha256:bb"`,
		``,
	}, "\n")
	if string(raw) != want {
		t.Fatalf("rendered:\n%s\nwant:\n%s", raw, want)
	}
}

// `created` is written as Python's `isoformat()` spells it, not as RFC 3339
// does: a UTC stamp reads `+00:00` there and `Z` here, and the difference
// alone would report every case Python wrote as changed.
func TestWriteCaseRendersTheOffsetLikePython(t *testing.T) {
	c := plainCase()
	c.Created = time.Date(2026, 9, 19, 10, 0, 0, 0, time.FixedZone("", 2*3600))
	path, _ := writeCase(t, c)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), "created = 2026-09-19T10:00:00+02:00\n") {
		t.Fatalf("rendered:\n%s", raw)
	}
}

// A flag that is false is not in the file, so an ordinary case keeps the file
// it had before the flag existed.
func TestWriteCaseOmitsFalseFlags(t *testing.T) {
	path, _ := writeCase(t, plainCase())
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, key := range []string{"manual", "local_only", "note", "prompt_version", "sources"} {
		if strings.Contains(string(raw), key) {
			t.Fatalf("rendered %q for a plain case:\n%s", key, raw)
		}
	}
}

// Strings are escaped the way a TOML basic string demands: a note carries
// prose this package does not author.
func TestWriteCaseEscapesANoteWithANewline(t *testing.T) {
	c := plainCase()
	c.Note = "two\nlines\x7f"
	path, _ := writeCase(t, c)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), `note = "two\nlines\u007f"`) {
		t.Fatalf("rendered:\n%s", raw)
	}
	got, err := maintenance.ReadCase(path)
	if err != nil {
		t.Fatalf("ReadCase: %v", err)
	}
	if got.Note != c.Note {
		t.Fatalf("note = %q", got.Note)
	}
}

// A second identical write is no change: the report of an interrupted run
// would otherwise send the reader after a change git will not show them.
func TestWriteCaseReportsNoChangeTwice(t *testing.T) {
	path, changed := writeCase(t, plainCase())
	if !changed {
		t.Fatal("the first write reported no change")
	}
	second, err := maintenance.WriteCase(path, plainCase())
	if err != nil {
		t.Fatalf("WriteCase: %v", err)
	}
	if second {
		t.Fatal("the second identical write reported a change")
	}
}

func TestWriteCaseReportsAChangedCase(t *testing.T) {
	path, _ := writeCase(t, plainCase())
	c := plainCase()
	c.State = "in_review"
	changed, err := maintenance.WriteCase(path, c)
	if err != nil {
		t.Fatalf("WriteCase: %v", err)
	}
	if !changed {
		t.Fatal("a changed case reported no change")
	}
}

// The case directory is made on the way: a case is the first file in it.
func TestWriteCaseMakesTheCaseDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project", "a-2026-09-19-abcd", "case.toml")
	if _, err := maintenance.WriteCase(path, plainCase()); err != nil {
		t.Fatalf("WriteCase: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat: %v", err)
	}
}

func TestWriteCaseFailsWhenTheDirectoryCannotBeMade(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "project")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := maintenance.WriteCase(filepath.Join(blocker, "case.toml"), plainCase()); err == nil {
		t.Fatal("WriteCase: want an error for a file where the directory belongs")
	}
}

func TestWriteCaseFailsWhenTheTargetIsADirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.toml")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if _, err := maintenance.WriteCase(path, plainCase()); err == nil {
		t.Fatal("WriteCase: want an error for a directory in the file's place")
	}
}

// The zero value is refused, because nothing else can be: Go has no naive
// datetime to refuse the way Python does.
func TestWriteCaseRefusesACreatedNobodySet(t *testing.T) {
	c := plainCase()
	c.Created = time.Time{}
	path := filepath.Join(t.TempDir(), "case.toml")
	if _, err := maintenance.WriteCase(path, c); err == nil {
		t.Fatal("WriteCase: want an error for a zero-value created")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("WriteCase wrote the file it refused")
	}
}

// The absence itself comes back, not a complaint about the fields of an empty
// document: `read_case` ends in the FileNotFoundError of `read_text`.
func TestReadCaseFailsOnAMissingFile(t *testing.T) {
	_, err := maintenance.ReadCase(filepath.Join(t.TempDir(), "case.toml"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want the missing file", err)
	}
}

func TestReadCaseFailsOnBrokenTOML(t *testing.T) {
	if _, err := maintenance.ReadCase(caseFile(t, "id = ")); err == nil {
		t.Fatal("ReadCase: want an error for a file that is no TOML")
	}
}

// A misspelt state must fail loudly rather than fall back to a guess.
func TestReadCaseRefusesAnUnknownState(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()), `"source_changed"`, `"reviewed"`, 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for an unknown state")
	}
}

// A state that is not there is a missing field, not a value outside the
// vocabulary -- `_require_str` speaks before the membership test does.
func TestReadCaseNamesAMissingState(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()), `state = "source_changed"`+"\n", "", 1)
	_, err := maintenance.ReadCase(caseFile(t, text))
	if err == nil || !strings.Contains(err.Error(), "state must be a non-empty string") {
		t.Fatalf("err = %v, want the missing state named", err)
	}
}

func TestReadCaseRefusesAnUnknownTrigger(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()), `trigger = "source_change"`, `trigger = "cron"`, 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for an unknown trigger")
	}
}

// A hand-edited case.toml is the normal case, not the exception, and a
// date-time without an offset comes back in the reader's own zone.
func TestReadCaseRefusesADateTimeWithoutAnOffset(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()),
		"created = 2026-09-19T10:00:00+00:00", "created = 2026-09-19T10:00:00", 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a date-time without an offset")
	}
}

// A plain date is the form tomllib answers with a `date` object for, which
// Python's `isinstance(created, datetime)` refuses.
func TestReadCaseRefusesAPlainDate(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()),
		"created = 2026-09-19T10:00:00+00:00", "created = 2026-09-19", 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a plain date")
	}
}

// A bare time is the third of the three zoneless forms, and it has to be
// nailed down on its own: the three names live in one `case` clause, which Go
// counts as one statement and reports covered as soon as any of them was hit.
func TestReadCaseRefusesAPlainTime(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()),
		"created = 2026-09-19T10:00:00+00:00", "created = 10:00:00", 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a plain time")
	}
}

func TestReadCaseRefusesACreatedThatIsNoDateTime(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()),
		"created = 2026-09-19T10:00:00+00:00", `created = "yesterday"`, 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a created that is no date-time")
	}
}

func TestReadCaseRefusesAMissingKey(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()), `weight = "change"`+"\n", "", 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a missing weight")
	}
}

func TestReadCaseRefusesAKeyOfTheWrongType(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()), `weight = "change"`, "weight = 3", 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a weight that is no string")
	}
}

// Empty is its own refusal, not the same `if` as the wrong type: Go measures
// statements, so one condition would report both arms covered from one test.
func TestReadCaseRefusesAnEmptyKey(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()), `weight = "change"`, `weight = ""`, 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for an empty weight")
	}
}

// A hand-edited `local_only = "ja"` would be truthy in a looser reader; in
// the field that decides whether a source diff may reach a cloud model the
// silent reading has to be the strict one.
func TestReadCaseRefusesAFlagThatIsNoBoolean(t *testing.T) {
	text := rendered(t, plainCase()) + "local_only = \"ja\"\n"
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a flag that is no boolean")
	}
}

func TestReadCaseRefusesANoteThatIsNoString(t *testing.T) {
	text := rendered(t, plainCase()) + "note = 3\n"
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a note that is no string")
	}
}

func TestReadCaseRefusesSourcesThatAreNoArray(t *testing.T) {
	text := rendered(t, plainCase()) + "sources = 3\n"
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for sources that are no array")
	}
}

func TestReadCaseRefusesASourceThatIsNoTable(t *testing.T) {
	text := rendered(t, plainCase()) + "sources = [1, 2]\n"
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a source that is no table")
	}
}

func TestReadCaseRefusesARevisionThatIsNoInteger(t *testing.T) {
	c := plainCase()
	c.Sources = []maintenance.SourceState{{DocID: "d1", Revision: 3, ContentHash: "sha256:bb"}}
	text := strings.Replace(rendered(t, c), "revision = 3", `revision = "3"`, 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a revision that is no integer")
	}
}

// An inline array of tables decodes as []any, not []map[string]any -- the
// second of the two shapes the decoder answers with.
func TestReadCaseTakesAnInlineSourceArray(t *testing.T) {
	text := rendered(t, plainCase()) +
		"sources = [{ doc_id = \"d1\", revision = 3, content_hash = \"sha256:bb\" }]\n"
	got, err := maintenance.ReadCase(caseFile(t, text))
	if err != nil {
		t.Fatalf("ReadCase: %v", err)
	}
	if len(got.Sources) != 1 || got.Sources[0].Revision != 3 {
		t.Fatalf("sources = %+v", got.Sources)
	}
}

// rendered is what WriteCase puts in the file, so a refusal test can change
// one key of a file that is otherwise correct.
func rendered(t *testing.T, c maintenance.Case) string {
	t.Helper()
	path, _ := writeCase(t, c)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(raw)
}

// A state that is no string fails on the same rule every required key does,
// before the closed vocabulary is ever consulted.
func TestReadCaseRefusesAStateThatIsNoString(t *testing.T) {
	text := strings.Replace(rendered(t, plainCase()), `state = "source_changed"`, "state = 3", 1)
	if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
		t.Fatal("ReadCase: want an error for a state that is no string")
	}
}

// Each key of a source is checked like a key of the case itself, and each of
// the three has to be able to refuse on its own.
func TestReadCaseRefusesAnIncompleteSource(t *testing.T) {
	for _, dropped := range []string{"doc_id", "revision", "content_hash"} {
		lines := []string{
			`doc_id = "d1"`, `revision = 3`, `content_hash = "sha256:bb"`,
		}
		var kept []string
		for _, line := range lines {
			if !strings.HasPrefix(line, dropped+" ") {
				kept = append(kept, line)
			}
		}
		text := rendered(t, plainCase()) + "\n[[sources]]\n" + strings.Join(kept, "\n") + "\n"
		if _, err := maintenance.ReadCase(caseFile(t, text)); err == nil {
			t.Fatalf("ReadCase: want an error for a source without %q", dropped)
		}
	}
}

// A refusal names the TOML type it found, in the vocabulary of the file rather
// than of Go -- the person who has to repair the file wrote TOML, not Go.
//
// `created` carries the string case, because a string is the one value
// `weight` accepts; every other value goes through `weight`.
func TestReadCaseNamesTheTypeItFound(t *testing.T) {
	for _, testCase := range []struct{ key, was, value, want string }{
		{"created", "2026-09-19T10:00:00+00:00", `"x"`, "a string"},
		{"weight", `"change"`, "3", "an integer"},
		{"weight", `"change"`, "1.5", "a float"},
		{"weight", `"change"`, "true", "a boolean"},
		{"weight", `"change"`, "2026-09-19T10:00:00+00:00", "a datetime"},
		{"weight", `"change"`, "{ a = 1 }", "a table"},
		{"weight", `"change"`, `["a"]`, "an array"},
	} {
		text := strings.Replace(rendered(t, plainCase()),
			testCase.key+" = "+testCase.was, testCase.key+" = "+testCase.value, 1)
		_, err := maintenance.ReadCase(caseFile(t, text))
		if err == nil {
			t.Fatalf("ReadCase: want an error for %s = %s", testCase.key, testCase.value)
		}
		if !strings.Contains(err.Error(), testCase.want) {
			t.Fatalf("error for %s = %s is %q, want %q in it", testCase.key, testCase.value, err, testCase.want)
		}
	}
}

// A case whose state or trigger is outside the closed vocabulary is refused
// before anything is written: the alternative is a file its own reader turns
// down, and nobody can decide a case they cannot read.
func TestWriteCaseRefusesAValueOutsideTheVocabulary(t *testing.T) {
	for _, testCase := range []struct{ field, value string }{
		{"state", "reviewed"},
		{"trigger", "cron"},
	} {
		c := plainCase()
		switch testCase.field {
		case "state":
			c.State = testCase.value
		case "trigger":
			c.Trigger = testCase.value
		}
		path := filepath.Join(t.TempDir(), "case.toml")
		_, err := maintenance.WriteCase(path, c)
		if err == nil {
			t.Fatalf("WriteCase: want an error for %s = %q", testCase.field, testCase.value)
		}
		if !strings.Contains(err.Error(), testCase.field+" must be one of ") {
			t.Fatalf("error for %s = %q is %q", testCase.field, testCase.value, err)
		}
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("WriteCase wrote the file it refused for %s", testCase.field)
		}
	}
}
