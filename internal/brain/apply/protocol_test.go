package apply

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// protocolGoldens is testdata/protocol.json, written by calling `_safe`,
// `_append_log`, `_append_audit` and `_append` of the Python reference.
type protocolGoldens struct {
	Times []string `json:"times"`
	Safe  []struct {
		In  string `json:"in"`
		Out string `json:"out"`
	} `json:"safe"`
	Log []struct {
		Time    int    `json:"time"`
		Target  string `json:"target"`
		Applied int    `json:"applied"`
		Case    string `json:"case"`
		Out     string `json:"out"`
	} `json:"log"`
	Audit []struct {
		Time       int      `json:"time"`
		Target     string   `json:"target"`
		Case       string   `json:"case"`
		Claims     int      `json:"claims"`
		Complaints []string `json:"complaints"`
		Decided    string   `json:"decided"`
		Changed    string   `json:"changed"`
		Out        string   `json:"out"`
	} `json:"audit"`
	Append []struct {
		Existing string `json:"existing"`
		Block    string `json:"block"`
		Out      string `json:"out"`
	} `json:"append"`
	Notes struct {
		Moved   string `json:"moved"`
		Refused string `json:"refused"`
		Amend   string `json:"amend"`
	} `json:"notes"`
}

func loadProtocolGoldens(t *testing.T) (protocolGoldens, []time.Time) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "protocol.json"))
	if err != nil {
		t.Fatal(err)
	}
	var goldens protocolGoldens
	if err := json.Unmarshal(raw, &goldens); err != nil {
		t.Fatal(err)
	}
	times := make([]time.Time, len(goldens.Times))
	for i, text := range goldens.Times {
		parsed, err := time.Parse(time.RFC3339Nano, text)
		if err != nil {
			t.Fatal(err)
		}
		times[i] = parsed
	}
	return goldens, times
}

func TestNotesAreTheReferenceTexts(t *testing.T) {
	goldens, _ := loadProtocolGoldens(t)
	for _, pair := range [][2]string{
		{MovedNote, goldens.Notes.Moved},
		{RefusedNote, goldens.Notes.Refused},
		{AmendNote, goldens.Notes.Amend},
	} {
		if pair[0] != pair[1] {
			t.Errorf("note %q, reference %q", pair[0], pair[1])
		}
	}
}

func TestSafeMatchesTheReference(t *testing.T) {
	goldens, _ := loadProtocolGoldens(t)
	for _, c := range goldens.Safe {
		if got := Safe(c.In); got != c.Out {
			t.Errorf("Safe(%q) = %q, reference %q", c.In, got, c.Out)
		}
	}
}

func TestSafeIsAFixedPoint(t *testing.T) {
	goldens, _ := loadProtocolGoldens(t)
	for _, c := range goldens.Safe {
		once := Safe(c.In)
		if twice := Safe(once); twice != once {
			t.Errorf("Safe(Safe(%q)) = %q, want %q", c.In, twice, once)
		}
	}
}

func TestSafeReplacesInvalidUTF8OnePerByte(t *testing.T) {
	// Python never sees invalid UTF-8 in a str; Go ranges over each bad byte
	// as U+FFFD, which is not a letter or number and so becomes one `·`.
	if got := Safe("a\xff\xfeb"); got != "a··b" || !utf8.ValidString(got) {
		t.Errorf("Safe = %q", got)
	}
}

func TestLogLineMatchesTheReference(t *testing.T) {
	goldens, times := loadProtocolGoldens(t)
	for _, c := range goldens.Log {
		if got := LogLine(times[c.Time], c.Target, c.Applied, c.Case); got != c.Out {
			t.Errorf("LogLine = %q, reference %q", got, c.Out)
		}
	}
}

func TestLogLineDatesInUTC(t *testing.T) {
	// The reference is always handed `datetime.now(UTC)`.
	local := time.Date(2026, 1, 1, 0, 30, 0, 0, time.FixedZone("x", 2*3600))
	if got := LogLine(local, "p", 1, "c"); !strings.HasPrefix(got, "- 2025-12-31 ") {
		t.Errorf("LogLine = %q", got)
	}
}

func TestRenderAuditMatchesTheReference(t *testing.T) {
	goldens, times := loadProtocolGoldens(t)
	for _, c := range goldens.Audit {
		entry := AuditEntry{
			Now:        times[c.Time],
			Target:     c.Target,
			CaseID:     c.Case,
			Claims:     make([]string, c.Claims),
			Complaints: c.Complaints,
			Decided:    c.Decided,
			Changed:    c.Changed,
		}
		if got := RenderAudit(entry); got != c.Out {
			t.Errorf("RenderAudit = %q, reference %q", got, c.Out)
		}
	}
}

func TestAppendMatchesTheReference(t *testing.T) {
	goldens, _ := loadProtocolGoldens(t)
	for _, c := range goldens.Append {
		if got := Append(c.Existing, c.Block); got != c.Out {
			t.Errorf("Append(%q, %q) = %q, reference %q", c.Existing, c.Block, got, c.Out)
		}
	}
}

func TestLogInsertPutsTheNewestFirst(t *testing.T) {
	// OKF §9: "a flat list of date-grouped entries, newest first", each day
	// under a `## YYYY-MM-DD` heading. The day is taken in UTC, as LogLine's.
	now := time.Date(2026, 9, 25, 0, 30, 0, 0, time.FixedZone("x", 2*3600))
	const head = "# Protokoll\n\n> Vorspann.\n\n"
	const line = "- 2026-09-24 — `neu.md`\n"
	cases := []struct {
		name, existing, want string
	}{
		{"empty", "", "## 2026-09-24\n\n" + line},
		{"no entry yet", head, head + "## 2026-09-24\n\n" + line},
		{"no entry yet, no trailing newline", "# Protokoll", "# Protokoll\n\n## 2026-09-24\n\n" + line},
		{"an older day",
			head + "## 2026-09-20\n\n- alt\n",
			head + "## 2026-09-24\n\n" + line + "\n## 2026-09-20\n\n- alt\n"},
		{"the same day",
			head + "## 2026-09-24\n\n- früher\n\n## 2026-09-20\n\n- alt\n",
			head + "## 2026-09-24\n\n" + line + "- früher\n\n## 2026-09-20\n\n- alt\n"},
		{"the same day, checked out with CRLF",
			"# P\r\n\r\n## 2026-09-24\r\n\r\n- früher\r\n",
			"# P\r\n\r\n## 2026-09-24\n\n" + line + "- früher\r\n"},
		{"lines without headings",
			head + "- 2026-09-17 — alt\n",
			head + "## 2026-09-24\n\n" + line + "\n- 2026-09-17 — alt\n"},
		{"star entries without headings",
			head + "* alt\n",
			head + "## 2026-09-24\n\n" + line + "\n* alt\n"},
	}
	for _, c := range cases {
		if got := LogInsert(c.existing, now, line); got != c.want {
			t.Errorf("%s: LogInsert =\n%q\nwant\n%q", c.name, got, c.want)
		}
	}
}

func TestUnrecordedComparesTheLastNote(t *testing.T) {
	cases := []struct {
		last, note string
		want       bool
	}{
		{"", MovedNote, true},
		{MovedNote, MovedNote, false},
		{RefusedNote, AmendNote, true},
		{AmendNote, RefusedNote, true},
		{"", "", false},
	}
	for _, c := range cases {
		if got := Unrecorded(c.last, c.note); got != c.want {
			t.Errorf("Unrecorded(%q, %q) = %v, want %v", c.last, c.note, got, c.want)
		}
	}
}
