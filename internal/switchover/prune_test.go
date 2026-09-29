package switchover

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const fixtureSettings = `{
  "permissions": {"allow": ["Bash(git status:*)"]},
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "uv run ultraloom hook session-start"}], "ultraLoomOwned": true}
    ],
    "PreToolUse": [
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "ulguard --root x"}], "ultraLoomOwned": true},
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "loomux hook pre-tool-use --host claude"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "brain wiki-gate"}, {"type": "command", "command": "my-own-check"}]}
    ]
  },
  "enabledPlugins": {"pyright-lsp@claude-plugins-official": true}
}`

const fixtureAfter = `{
  "permissions": {"allow": ["Bash(git status:*)"]},
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          {
            "type": "command",
            "command": "loomux hook pre-tool-use --host claude"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "brain wiki-gate"
          },
          {
            "type": "command",
            "command": "my-own-check"
          }
        ]
      }
    ]
  },
  "enabledPlugins": {"pyright-lsp@claude-plugins-official": true}
}`

var fixtureNeedles = []string{"ulguard", "brain wiki-gate", "ultraloom hook"}

// realSettings has the shape of a settings.json written by the old
// installer: two-space indent, every hook group marked ultraLoomOwned.
const realSettings = `{
  "permissions": {
    "allow": [
      "WebSearch",
      "Bash(git log:*)"
    ],
    "additionalDirectories": []
  },
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "uv run --project \"${CLAUDE_PROJECT_DIR}/.ultraloom/vendor/ultraloom\" ultraloom hook session-start --root \"${CLAUDE_PROJECT_DIR}\"",
            "timeout": 20
          }
        ],
        "ultraLoomOwned": true
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Write|Edit|NotebookEdit|Bash|PowerShell",
        "hooks": [
          {
            "type": "command",
            "command": "ulguard --root \"${CLAUDE_PROJECT_DIR}\"",
            "timeout": 10
          }
        ],
        "ultraLoomOwned": true
      }
    ],
    "Stop": [
      {
        "matcher": "wiki",
        "hooks": [
          {
            "type": "command",
            "command": "brain wiki-gate --root \"${CLAUDE_PROJECT_DIR}\"",
            "timeout": 120
          }
        ],
        "ultraLoomOwned": true
      }
    ]
  },
  "enabledPlugins": {
    "pyright-lsp@claude-plugins-official": true
  }
}
`

// run prunes and fails the test on an error.
func run(t *testing.T, in string, needles []string) (out string, removed, kept []string) {
	t.Helper()
	b, removed, kept, err := PruneHooks([]byte(in), needles)
	if err != nil {
		t.Fatalf("PruneHooks: %v", err)
	}
	return string(b), removed, kept
}

func TestPruneHooksDropsWholeOldGroupsAndKeepsTheRest(t *testing.T) {
	out, removed, kept := run(t, fixtureSettings, fixtureNeedles)
	if strings.Contains(out, "ulguard") || strings.Contains(out, "ultraloom hook") {
		t.Errorf("an old group is still there:\n%s", out)
	}
	if !strings.Contains(out, "loomux hook pre-tool-use") {
		t.Errorf("the new hook is gone:\n%s", out)
	}
	wantRemoved := []string{"SessionStart: uv run ultraloom hook session-start", "PreToolUse: ulguard --root x"}
	if !reflect.DeepEqual(removed, wantRemoved) {
		t.Errorf("removed = %q, want %q", removed, wantRemoved)
	}
	// The Stop group mixes an old command with the user's own: it stays.
	if !strings.Contains(out, "my-own-check") || !reflect.DeepEqual(kept, []string{"Stop: brain wiki-gate"}) {
		t.Errorf("mixed group: kept = %q", kept)
	}
	if strings.Contains(out, `"SessionStart"`) {
		t.Errorf("an empty event is still there:\n%s", out)
	}
}

func TestPruneHooksWritesExactlyTheExpectedDocument(t *testing.T) {
	out, _, _ := run(t, fixtureSettings, fixtureNeedles)
	if out != fixtureAfter {
		t.Errorf("out:\n%s\nwant:\n%s", out, fixtureAfter)
	}
}

func TestPruneHooksLeavesEverythingOutsideHooksByteForByte(t *testing.T) {
	out, _, _ := run(t, fixtureSettings, fixtureNeedles)
	head := fixtureSettings[:strings.Index(fixtureSettings, `"hooks"`)]
	tail := fixtureSettings[strings.LastIndex(fixtureSettings, `  "enabledPlugins"`)-len(",\n"):]
	if !strings.HasPrefix(out, head) || !strings.HasSuffix(out, tail) {
		t.Errorf("the bytes around hooks changed:\n%s", out)
	}
	if strings.Index(out, `"PreToolUse"`) > strings.Index(out, `"Stop"`) {
		t.Errorf("event order changed:\n%s", out)
	}
}

func TestPruneHooksKeepsTheOrderOfEventsAsInTheFile(t *testing.T) {
	// The order is neither alphabetical nor the order of the removals.
	in := `{"hooks": {"Zeta": [{"hooks": [{"command": "own"}]}], "Alpha": [{"hooks": [{"command": "old"}]}, {"hooks": [{"command": "own2"}]}], "Mid": [{"hooks": [{"command": "own3"}]}]}}`
	out, _, _ := run(t, in, []string{"old"})
	z, a, m := strings.Index(out, `"Zeta"`), strings.Index(out, `"Alpha"`), strings.Index(out, `"Mid"`)
	if !(z >= 0 && z < a && a < m) {
		t.Errorf("event order changed (Zeta %d, Alpha %d, Mid %d):\n%s", z, a, m, out)
	}
}

func TestPruneHooksChangesNothingWhenNothingMatches(t *testing.T) {
	for name, n := range map[string][]string{"no needles": nil, "no match": {"zzz"}, "empty needle": {""}} {
		out, removed, kept := run(t, fixtureSettings, n)
		if out != fixtureSettings || len(removed) != 0 || len(kept) != 0 {
			t.Errorf("%s: changed %v, removed %q, kept %q", name, out != fixtureSettings, removed, kept)
		}
	}
	plain := `{"permissions": {}}`
	if out, removed, kept := run(t, plain, fixtureNeedles); out != plain || len(removed) != 0 || len(kept) != 0 {
		t.Errorf("a file without hooks changed: %q", out)
	}
}

func TestPruneHooksIgnoresAnEmptyNeedleNextToARealOne(t *testing.T) {
	// An empty needle is a substring of every command; it must not match.
	in := `{"hooks": {"Stop": [{"hooks": [{"command": "own"}]}, {"hooks": [{"command": "old"}]}]}}`
	_, removed, _ := run(t, in, []string{"", "old"})
	if !reflect.DeepEqual(removed, []string{"Stop: old"}) {
		t.Errorf("removed = %q", removed)
	}
}

func TestPruneHooksReportsAMixedGroupEvenWhenNothingIsRemoved(t *testing.T) {
	in := `{"hooks": {"Stop": [{"hooks": [{"command": "own"}, {"command": "old thing"}, {"command": "old other"}]}]}}`
	out, removed, kept := run(t, in, []string{"old"})
	if out != in || len(removed) != 0 {
		t.Errorf("changed %v, removed %q", out != in, removed)
	}
	// The first matching command, not the first command.
	if !reflect.DeepEqual(kept, []string{"Stop: old thing"}) {
		t.Errorf("kept = %q", kept)
	}
}

func TestPruneHooksRemovesAGroupOnlyWhenEveryCommandMatches(t *testing.T) {
	in := `{"hooks": {"Stop": [
	  {"hooks": [{"command": "old a"}, {"command": "old b"}]},
	  {"hooks": [{"command": "old c"}, {"command": "own"}]},
	  {"hooks": [{"command": "own"}, {"command": "old d"}]}
	]}}`
	out, removed, kept := run(t, in, []string{"old"})
	if !reflect.DeepEqual(removed, []string{"Stop: old a; old b"}) {
		t.Errorf("removed = %q", removed)
	}
	if !reflect.DeepEqual(kept, []string{"Stop: old c", "Stop: old d"}) {
		t.Errorf("kept = %q", kept)
	}
	if strings.Contains(out, "old a") || !strings.Contains(out, "old c") || !strings.Contains(out, "old d") {
		t.Errorf("wrong groups removed:\n%s", out)
	}
}

func TestPruneHooksMatchesASubstringOfTheCommandNotTheWholeCommand(t *testing.T) {
	in := `{"hooks": {"Stop": [{"hooks": [{"command": "/x/bin/ulguard --root y"}]}]}}`
	_, removed, _ := run(t, in, []string{"ulguard"})
	if len(removed) != 1 {
		t.Errorf("removed = %q", removed)
	}
	// A needle longer than the command never matches.
	if _, removed, _ = run(t, in, []string{"/x/bin/ulguard --root y and more"}); len(removed) != 0 {
		t.Errorf("removed = %q", removed)
	}
}

func TestPruneHooksCountsACommandOnceWhateverNumberOfNeedlesItHas(t *testing.T) {
	// One command that contains two needles is one matching command.
	in := `{"hooks": {"Stop": [{"hooks": [{"command": "ulguard and brain wiki-gate"}]}]}}`
	_, removed, kept := run(t, in, fixtureNeedles)
	if !reflect.DeepEqual(removed, []string{"Stop: ulguard and brain wiki-gate"}) || len(kept) != 0 {
		t.Errorf("removed = %q, kept = %q", removed, kept)
	}
}

func TestPruneHooksKeepsTheOrderOfTheGroupsThatStay(t *testing.T) {
	in := "{\n  \"hooks\": " + `{"Stop": [
	  {"hooks": [{"command": "first own"}]},
	  {"hooks": [{"command": "old"}]},
	  {"hooks": [{"command": "second own"}]}
	]}` + "\n}\n"
	out, _, _ := run(t, in, []string{"old"})
	if a, b := strings.Index(out, "first own"), strings.Index(out, "second own"); a < 0 || a > b {
		t.Errorf("group order changed:\n%s", out)
	}
}

func TestPruneHooksDoesNotReportAGroupItCannotRead(t *testing.T) {
	// A hook whose command is no string makes the group unreadable: it is
	// neither removed nor reported, even though another command matches.
	in := `{"hooks": {"Stop": [{"hooks": [{"command": "old"}, {"command": 5}]}]}}`
	out, removed, kept := run(t, in, []string{"old"})
	if out != in || len(removed) != 0 || len(kept) != 0 {
		t.Errorf("changed %v, removed %q, kept %q", out != in, removed, kept)
	}
}

func TestPruneHooksNeverRemovesWhatItCannotRead(t *testing.T) {
	cases := map[string]string{
		"a group without commands":       `{"hooks": []}`,
		"a group without a hooks list":   `{"matcher": "old"}`,
		"a group that is not an object":  `"old"`,
		"a hooks list that is no list":   `{"hooks": "old"}`,
		"a command that is no string":    `{"hooks": [{"command": 5}]}`,
		"an entry without a command":     `{"hooks": [{"type": "prompt", "prompt": "old"}]}`,
		"a command next to a prompt one": `{"hooks": [{"command": "old"}, {"type": "prompt"}]}`,
	}
	for name, group := range cases {
		in := `{"hooks": {"Stop": [` + group + `]}}`
		out, removed, _ := run(t, in, []string{"old"})
		if out != in || len(removed) != 0 {
			t.Errorf("%s: removed %q, changed %v", name, removed, out != in)
		}
	}
}

func TestPruneHooksKeepsAnEventThatIsNotAListOrWasEmptyAlready(t *testing.T) {
	in := "{\n  \"hooks\": " + `{"A": "old", "B": [], "C": {"hooks": [{"command": "old"}]}, "D": [{"hooks": [{"command": "old"}]}]}` + "\n}\n"
	out, removed, _ := run(t, in, []string{"old"})
	if !reflect.DeepEqual(removed, []string{"D: old"}) {
		t.Errorf("removed = %q", removed)
	}
	// Indented, so an untouched value is still the file's own text.
	for _, want := range []string{`"A": "old"`, `"B": []`, `"C": {`} {
		if !strings.Contains(out, want) {
			t.Errorf("%s is gone:\n%s", want, out)
		}
	}
	if strings.Contains(out, `"D"`) {
		t.Errorf("D lost its only group and must go:\n%s", out)
	}
}

func TestPruneHooksListsEveryRemovedGroupOfAnEventInOrder(t *testing.T) {
	in := `{"hooks": {"Stop": [
	  {"hooks": [{"command": "old 1"}]},
	  {"hooks": [{"command": "own"}]},
	  {"hooks": [{"command": "old 2"}]}
	], "Start": [{"hooks": [{"command": "old 3"}]}]}}`
	out, removed, _ := run(t, in, []string{"old"})
	if !reflect.DeepEqual(removed, []string{"Stop: old 1", "Stop: old 2", "Start: old 3"}) {
		t.Errorf("removed = %q", removed)
	}
	if !strings.Contains(out, "own") || strings.Contains(out, "old") {
		t.Errorf("out:\n%s", out)
	}
}

func TestPruneHooksLeavesAnEmptyHooksObjectWhenEverythingGoes(t *testing.T) {
	out, removed, _ := run(t, realSettings, fixtureNeedles)
	if len(removed) != 3 {
		t.Errorf("removed = %q", removed)
	}
	i := strings.Index(realSettings, `"hooks": `) + len(`"hooks": `)
	j := strings.Index(realSettings, ",\n  \"enabledPlugins\"")
	want := realSettings[:i] + "{}" + realSettings[j:]
	if out != want {
		t.Errorf("out:\n%s\nwant:\n%s", out, want)
	}
}

func TestPruneHooksIsIdempotent(t *testing.T) {
	once, _, _ := run(t, fixtureSettings, fixtureNeedles)
	twice, removed, kept := run(t, once, fixtureNeedles)
	if twice != once || len(removed) != 0 || len(kept) != 1 {
		t.Errorf("second run changed %v, removed %q, kept %q", twice != once, removed, kept)
	}
}

func TestPruneHooksFollowsTheIndentOfTheFile(t *testing.T) {
	body := func(unit string) string {
		return "{\n" + unit + `"a": 1,` + "\n" + unit + `"hooks": {"Stop": [{"hooks": [{"command": "old"}]}, {"hooks": [{"command": "own"}]}]}` + "\n}\n"
	}
	want := func(unit string) string {
		return "{\n" + unit + `"a": 1,` + "\n" + unit + `"hooks": {` + "\n" +
			unit + unit + `"Stop": [` + "\n" +
			unit + unit + unit + "{\n" +
			unit + unit + unit + unit + `"hooks": [` + "\n" +
			unit + unit + unit + unit + unit + "{\n" +
			unit + unit + unit + unit + unit + unit + `"command": "own"` + "\n" +
			unit + unit + unit + unit + unit + "}\n" +
			unit + unit + unit + unit + "]\n" +
			unit + unit + unit + "}\n" +
			unit + unit + "]\n" +
			unit + "}\n}\n"
	}
	for name, unit := range map[string]string{"two spaces": "  ", "four spaces": "    ", "tab": "\t"} {
		if out, _, _ := run(t, body(unit), []string{"old"}); out != want(unit) {
			t.Errorf("%s: out:\n%q\nwant:\n%q", name, out, want(unit))
		}
	}
}

func TestPruneHooksWritesACompactBlockIntoACompactFile(t *testing.T) {
	in := `{"a":1,"hooks":{"Stop":[{"hooks":[{"command":"old"}]},{"hooks": [ {"command": "own"} ]}]},"b":2}`
	out, _, _ := run(t, in, []string{"old"})
	want := `{"a":1,"hooks":{"Stop":[{"hooks":[{"command":"own"}]}]},"b":2}`
	if out != want {
		t.Errorf("out = %s, want %s", out, want)
	}
}

func TestPruneHooksKeepsTheLineEndingsOfACRLFFile(t *testing.T) {
	in := strings.ReplaceAll(fixtureSettings, "\n", "\r\n")
	out, _, _ := run(t, in, fixtureNeedles)
	want := strings.ReplaceAll(fixtureAfter, "\n", "\r\n")
	if out != want {
		t.Errorf("out = %q\nwant %q", out, want)
	}
	if strings.Count(out, "\n") != strings.Count(out, "\r\n") {
		t.Errorf("a bare LF is left in the file")
	}
	// A CRLF file that has no hooks change stays as it was.
	same, _, _ := run(t, in, []string{"zzz"})
	if same != in {
		t.Errorf("an unchanged CRLF file was rewritten")
	}
}

func TestPruneHooksTakesTheLineEndingOfTheLineTheKeyStandsOn(t *testing.T) {
	// The line before "hooks" ends in CRLF, the rest of the file in LF.
	crlfBefore := "{\n  \"a\": 1,\r\n" + `  "hooks": {"S": [{"hooks": [{"command": "old"}]}, {"hooks": [{"command": "own"}]}]}` + "\n}\n"
	out, _, _ := run(t, crlfBefore, []string{"old"})
	block := out[strings.Index(out, `"hooks"`):strings.LastIndex(out, "}\n}")]
	if strings.Count(block, "\n") == 0 || strings.Count(block, "\n") != strings.Count(block, "\r\n") {
		t.Errorf("the block does not use CRLF:\n%q", out)
	}
	// The line before "hooks" ends in LF, the rest of the file in CRLF.
	lfBefore := "{\r\n  \"a\": 1,\n" + `  "hooks": {"S": [{"hooks": [{"command": "old"}]}, {"hooks": [{"command": "own"}]}]}` + "\r\n}\r\n"
	out, _, _ = run(t, lfBefore, []string{"old"})
	block = out[strings.Index(out, `"hooks"`):strings.LastIndex(out, "}\r\n}")]
	if strings.Count(block, "\n") == 0 || strings.Contains(block, "\r") {
		t.Errorf("the block does not use LF:\n%q", out)
	}
	if !strings.HasPrefix(out, "{\r\n  \"a\": 1,\n") || !strings.HasSuffix(out, "\r\n}\r\n") {
		t.Errorf("the bytes around the block changed:\n%q", out)
	}
}

func TestPruneHooksIndentsAValueWhoseLineStartsAtTheFileStart(t *testing.T) {
	// The whitespace before the key is on the first line, or on the second
	// with only an LF before it: there is no line ending to look at.
	body := `{"hooks": {"S": [{"hooks": [{"command": "old"}]}, {"hooks": [{"command": "own"}]}]}}`
	for name, in := range map[string]string{"first line": "  " + body, "second line": "\n  " + body} {
		out, removed, _ := run(t, in, []string{"old"})
		if len(removed) != 1 || strings.Contains(out, "\r") || !strings.Contains(out, "\n            \"command\": \"own\"") {
			t.Errorf("%s: removed %q, out %q", name, removed, out)
		}
	}
}

func TestPruneHooksDoesNotAddCRToAnLFFile(t *testing.T) {
	out, _, _ := run(t, fixtureSettings, fixtureNeedles)
	if strings.Contains(out, "\r") {
		t.Errorf("a CR appeared in an LF file")
	}
}

func TestPruneHooksFindsTheRealHooksKeyAmongLookAlikes(t *testing.T) {
	// The text "hooks" occurs in a value before and after the real key, in
	// a nested object before it, and after multi-byte text, so the start of
	// the value has to be counted in bytes.
	in := "{\n" +
		`  "ä": "ü \"hooks\": {\"Stop\": []}",` + "\n" +
		`  "nested": {"hooks": {"Stop": [{"hooks": [{"command": "old"}]}]}},` + "\n" +
		`  "日本語": ["hooks", "€"],` + "\n" +
		`  "hooks": {"Stop": [{"hooks": [{"command": "old"}]}, {"hooks": [{"command": "own"}]}]},` + "\n" +
		`  "z": "hooks"` + "\n}\n"
	out, removed, _ := run(t, in, []string{"old"})
	if !reflect.DeepEqual(removed, []string{"Stop: old"}) {
		t.Errorf("removed = %q", removed)
	}
	head := in[:strings.Index(in, `  "hooks": {"Stop"`)]
	tail := ",\n  \"z\": \"hooks\"\n}\n"
	if !strings.HasPrefix(out, head) || !strings.HasSuffix(out, tail) {
		t.Errorf("bytes around hooks changed:\n%s", out)
	}
	if !strings.Contains(out, `"nested": {"hooks": {"Stop": [{"hooks": [{"command": "old"}]}]}}`) {
		t.Errorf("the nested object was touched:\n%s", out)
	}
	if strings.Count(out, `"command": "old"`) != 1 {
		t.Errorf("out:\n%s", out)
	}
}

func TestPruneHooksTakesTheLastOfTwoHooksKeys(t *testing.T) {
	in := `{"hooks": {"Stop": [{"hooks": [{"command": "old"}]}]}, "hooks": {"Stop": [{"hooks": [{"command": "old2"}]}, {"hooks": [{"command": "own"}]}]}}`
	out, removed, _ := run(t, in, []string{"old"})
	if !reflect.DeepEqual(removed, []string{"Stop: old2"}) {
		t.Errorf("removed = %q", removed)
	}
	if !strings.HasPrefix(out, `{"hooks": {"Stop": [{"hooks": [{"command": "old"}]}]}, "hooks": `) {
		t.Errorf("the first hooks key changed:\n%s", out)
	}
}

func TestPruneHooksKeepsTheTextOfWhatItKeeps(t *testing.T) {
	// The escapes are built from parts: the editor turns a literal backslash-u
	// sequence into the character.
	esc := `\` + `u00e4`
	amp := `\` + `u0026`
	in := "{\n  \"hooks\": " + `{"Stö&p<x>": [
	  {"hooks": [{"command": "old"}]},
	  {"hooks": [{"command": "ä` + esc + ` \"q\" </b>", "timeout": 1.50, "n": 1e3}], "x": "` + amp + `"}
	]}` + "\n}\n"
	out, _, _ := run(t, in, []string{"old"})
	for _, want := range []string{`"Stö&p<x>": [`, `"ä` + esc + ` \"q\" </b>"`, `"timeout": 1.50`, `"n": 1e3`, `"x": "` + amp + `"`} {
		if !strings.Contains(out, want) {
			t.Errorf("%s changed:\n%s", want, out)
		}
	}
}

func TestPruneHooksRefusesWhatIsNotJSON(t *testing.T) {
	const (
		syntax = iota // the decoder's own message
		notObject
		trailing
	)
	cases := map[string]struct {
		in   string
		kind int
	}{
		"empty":                   {``, syntax},
		"broken key":              {`{1: 2}`, syntax},
		"broken value":            {`{"a":`, syntax},
		"wrong closing bracket":   {`{]`, syntax},
		"unterminated number":     {`{"a": 1`, syntax},
		"unterminated object":     {`{"a": "x"`, syntax},
		"broken inside the hooks": {`{"hooks": {"Stop": [}}`, syntax},
		"not an object":           {`[]`, notObject},
		"a scalar":                {`1`, notObject},
		"hooks is a list":         {`{"hooks": []}`, notObject},
		"hooks is null":           {`{"hooks": null}`, notObject},
		"hooks is a string":       {`{"hooks": "x"}`, notObject},
		"trailing data":           {`{"a": 1} x`, trailing},
		"a second document":       {`{"a": 1} {}`, trailing},
		"trailing brace":          {`{"a": 1} }`, trailing},
	}
	for name, c := range cases {
		out, removed, kept, err := PruneHooks([]byte(c.in), fixtureNeedles)
		if err == nil {
			t.Errorf("%s: no error", name)
			continue
		}
		if out != nil || removed != nil || kept != nil {
			t.Errorf("%s: results next to an error: %q %q %q", name, out, removed, kept)
		}
		if got := errors.Is(err, errNotObject); got != (c.kind == notObject) {
			t.Errorf("%s: not-an-object = %v for %q", name, got, err)
		}
		if got := errors.Is(err, errTrailing); got != (c.kind == trailing) {
			t.Errorf("%s: trailing = %v for %q", name, got, err)
		}
	}
}

func TestPruneHooksLooksNoFurtherWithoutNeedles(t *testing.T) {
	// With nothing to look for the file is not even read.
	for _, in := range []string{`{`, `not json`, ``} {
		out, removed, kept, err := PruneHooks([]byte(in), nil)
		if err != nil || string(out) != in || removed != nil || kept != nil {
			t.Errorf("%q: %q, %v, %v, %v", in, out, removed, kept, err)
		}
	}
}
