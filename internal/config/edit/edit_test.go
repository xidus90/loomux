package edit

import (
	"errors"
	"strings"
	"testing"
)

func TestSetAddsANamedTableThatIsNotThereYet(t *testing.T) {
	text := "[agent]\ndefault = \"w\"\n\n[agent.models.w]\nprovider = \"claude\"\n"
	got, err := Set(text, "agent.models.gemini", "provider", `"agy"`)
	if err != nil {
		t.Fatal(err)
	}
	want := text + "\n[agent.models.gemini]\nprovider = \"agy\"\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	got, err = Set(got, "agent.roles", "reviewer", `"gemini"`)
	if err != nil || !strings.HasSuffix(got, "[agent.roles]\nreviewer = \"gemini\"\n") {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestSetReplacesAValueAndKeepsItsComment(t *testing.T) {
	in := "# head\n[commit]\nlanguage = \"en\"  # prose is German\nthreshold = 2\n"
	got, err := Set(in, "commit", "language", `"de"`)
	if err != nil {
		t.Fatal(err)
	}
	want := "# head\n[commit]\nlanguage = \"de\"  # prose is German\nthreshold = 2\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestSetReplacesAMultiLineList(t *testing.T) {
	in := "[index]\ninclude = [\n  \"a\",  # first\n  \"b]\",\n]\nexclude = []\n"
	got, err := Set(in, "index", "include", `["c"]`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[index]\ninclude = [\"c\"]\nexclude = []\n" {
		t.Fatalf("got\n%s", got)
	}
}

func TestSetAddsAKeyAfterTheLastKeyOfItsSection(t *testing.T) {
	in := "[commit]\nlanguage = \"en\"\n\n# next\n[layout]\nwiki = \"docs\"\n"
	got, _ := Set(in, "commit", "threshold", "3")
	want := "[commit]\nlanguage = \"en\"\nthreshold = 3\n\n# next\n[layout]\nwiki = \"docs\"\n"
	if got != want {
		t.Fatalf("got\n%s", got)
	}
}

func TestSetAddsAMissingSectionAtTheEnd(t *testing.T) {
	got, _ := Set("[area]\nscope = \"s\"\n", "modules", "graph", "false")
	if got != "[area]\nscope = \"s\"\n\n[modules]\ngraph = false\n" {
		t.Fatalf("got\n%s", got)
	}
}

func TestSetTreatsASubTableAsItsOwnSection(t *testing.T) {
	in := "[verify]\ntimeout = \"600s\"\n[verify.go.lint]\ncommands = []\n"
	got, _ := Set(in, "verify", "max_parallel", "4")
	want := "[verify]\ntimeout = \"600s\"\nmax_parallel = 4\n[verify.go.lint]\ncommands = []\n"
	if got != want {
		t.Fatalf("got\n%s", got)
	}
}

func TestSetRefusesWhatItCannotPlace(t *testing.T) {
	for name, in := range map[string]string{
		"dotted key":   "commit.language = \"en\"\n",
		"inline table": "commit = { language = \"en\" }\n",
		"twice":        "[commit]\nthreshold = 1\n[commit]\nlanguage = \"en\"\n",
		"key twice":    "[commit]\nlanguage = \"en\"\nlanguage = \"de\"\n",
		"quoted key":   "[commit]\n\"language\" = \"en\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Set(in, "commit", "language", `"de"`); !errors.Is(err, ErrAmbiguous) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestSetKeepsCRLF(t *testing.T) {
	got, _ := Set("[commit]\r\nlanguage = \"en\"\r\n", "commit", "language", `"de"`)
	if got != "[commit]\r\nlanguage = \"de\"\r\n" {
		t.Fatalf("%q", got)
	}
}

// A file whose lines end in both ways, as one edited by two editors: every
// line keeps its own ending, and no key is lost to a line glued to the one
// before it.
func TestSetKeepsMixedLineEndings(t *testing.T) {
	text := "[commit]\r\nthreshold = 2\nlanguage = \"en\"\r\n"
	got, err := Set(text, "commit", "threshold", "3")
	if err != nil || got != "[commit]\r\nthreshold = 3\nlanguage = \"en\"\r\n" {
		t.Fatalf("Set = %q, %v", got, err)
	}
	got, err = Set(text, "commit", "language", `"de"`)
	if err != nil || got != "[commit]\r\nthreshold = 2\nlanguage = \"de\"\n" {
		t.Fatalf("Set = %q, %v", got, err)
	}
	got, err = Remove(text, "commit", "threshold")
	if err != nil || got != "[commit]\r\nlanguage = \"en\"\r\n" {
		t.Fatalf("Remove = %q, %v", got, err)
	}
}

// Notepad saves a CRLF file without an ending after the last line; that line
// has no \r to give, and the file is still CRLF.
func TestSetKeepsCRLFWithoutAFinalLineEnd(t *testing.T) {
	got, err := Set("[commit]\r\nlanguage = \"en\"\r\nthreshold = 2", "modules", "graph", "false")
	if err != nil || got != "[commit]\r\nlanguage = \"en\"\r\nthreshold = 2\r\n\r\n[modules]\r\ngraph = false\r\n" {
		t.Fatalf("Set = %q, %v", got, err)
	}
	// One line and no ending at all names no CRLF: new lines get \n.
	got, err = Set("[commit]", "commit", "threshold", "2")
	if err != nil || got != "[commit]\nthreshold = 2\n" {
		t.Fatalf("Set = %q, %v", got, err)
	}
}

// TestSetReadsStringsAndCommentsInsideAValue covers what the value scanners
// must skip: an escaped quote inside a string, and a bracket in a comment of
// a multi-line list.
func TestSetReadsStringsAndCommentsInsideAValue(t *testing.T) {
	in := "[commit]\nlanguage = \"e\\\"n # no\" # real\n"
	got, err := Set(in, "commit", "language", `"de"`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[commit]\nlanguage = \"de\" # real\n" {
		t.Fatalf("%q", got)
	}
	in = "[index]\ninclude = [ # [\n  \"a\\\"]\",\n  'b]',\n]\nexclude = []\n"
	got, err = Set(in, "index", "include", `[]`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[index]\ninclude = []\nexclude = []\n" {
		t.Fatalf("%q", got)
	}
}

// TestSetLeavesDottedKeysOfOtherSectionsAlone: a dotted key under another
// table cannot belong to the one being changed.
func TestSetLeavesDottedKeysOfOtherSectionsAlone(t *testing.T) {
	in := "[layout]\nwiki.path = \"docs\"\n[commit]\nlanguage = \"en\"\n"
	got, err := Set(in, "commit", "language", `"de"`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[layout]\nwiki.path = \"docs\"\n[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("%q", got)
	}
	// [layout] commit.language is layout.commit.language: no key of [commit].
	if _, err := Set("[layout]\ncommit.language = \"en\"\n", "commit", "language", `"de"`); err != nil {
		t.Fatalf("a dotted key of another table: got %v", err)
	}
}

// TestSetRefusesKeysThatReachIntoTheSection: a dotted key or an inline table
// under a parent table spells keys of a sub-table, so a new [verify.go]
// header would define that table twice.
func TestSetRefusesKeysThatReachIntoTheSection(t *testing.T) {
	for name, in := range map[string]string{
		"dotted key":   "[verify]\ngo.lint = []\n",
		"inline table": "[verify]\ngo = { lint = [] }\n",
		"plain value":  "[verify]\ngo = 1\n",
		"inline in it": "[verify.go]\nlint = { commands = [] }\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Set(in, "verify.go", "x", "1"); !errors.Is(err, ErrAmbiguous) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

// TestSetRefusesLinesOfNoKnownForm: a header or key spelled with spaces or
// quotes is TOML this scan does not read, and guessing past it would put the
// change into the wrong table.
func TestSetRefusesLinesOfNoKnownForm(t *testing.T) {
	for name, in := range map[string]string{
		"spaced header":  "[ commit ]\nlanguage = \"en\"\n",
		"quoted header":  "[\"commit\"]\nlanguage = \"en\"\n",
		"spaced dotted":  "[commit]\nlanguage . x = 1\n",
		"quoted dotted":  "[commit]\nlanguage.\"x\" = 1\n",
		"spaced section": "[a . b]\nx = 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Set(in, "commit", "language", `"de"`); !errors.Is(err, ErrAmbiguous) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

// TestSetRefusesMultiLineStrings: the lines inside a multi-line string are
// text, not headers and keys, and this scan cannot tell them apart.
func TestSetRefusesMultiLineStrings(t *testing.T) {
	for name, in := range map[string]string{
		"basic":          "[area]\nreason = \"\"\"\nfirst\n[commit]\n\"\"\"\n",
		"literal":        "[area]\nreason = '''\nfirst\n[commit]\n'''\n",
		"dotted key":     "[area]\na.reason = \"\"\"\nfirst\n\"\"\"\n",
		"in a list":      "[area]\nreasons = [\n  \"\"\"\nfirst\n[commit]\n\"\"\",\n]\n",
		"after a string": "[area]\nreason = [\"a\\\"\", '''\nb''']\n",
		// Only the opening line gives these away: the closing line reads as
		// a comment, so the scan must refuse where the string starts.
		"closed in a comment":            "[area]\nreason = \"\"\"\n[commit]\n# end\"\"\"\n",
		"dotted key closed in a comment": "[area]\na.reason = \"\"\"\n[commit]\n# end\"\"\"\n",
		"after an even string":           "[area]\nreason = [\"ab\", '''x''']\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Set(in, "commit", "language", `"de"`); !errors.Is(err, ErrAmbiguous) {
				t.Fatalf("got %v", err)
			}
		})
	}
	// Three quotes in a comment open nothing.
	in := "[commit]\nlanguage = \"en\" # \"\"\" \n"
	if _, err := Set(in, "commit", "language", `"de"`); err != nil {
		t.Fatalf("a comment: got %v", err)
	}
}

func TestSetFillsAnEmptyFile(t *testing.T) {
	got, _ := Set("", "modules", "graph", "false")
	if got != "[modules]\ngraph = false\n" {
		t.Fatalf("%q", got)
	}
}

func TestRemoveDropsTheLineAndAnEmptySection(t *testing.T) {
	got, _ := Remove("[area]\nscope = \"s\"\n\n[modules]\ngraph = false\n", "modules", "graph")
	if got != "[area]\nscope = \"s\"\n" {
		t.Fatalf("%q", got)
	}
	got, _ = Remove("[commit]\nlanguage = \"de\"\nthreshold = 3\n", "commit", "language")
	if got != "[commit]\nthreshold = 3\n" {
		t.Fatalf("%q", got)
	}
	got, _ = Remove("[modules]\ngraph = false\n", "modules", "graph")
	if got != "" {
		t.Fatalf("the last key leaves an empty file: %q", got)
	}
}

func TestRemoveLeavesAMissingKeyAlone(t *testing.T) {
	in := "[commit]\nthreshold = 3\n"
	got, err := Remove(in, "commit", "language")
	if err != nil || got != in {
		t.Fatalf("%q, %v", got, err)
	}
}

func TestRemoveRefusesWhatItCannotPlace(t *testing.T) {
	for name, in := range map[string]string{
		"dotted key": "commit.language = \"en\"\n",
		"twice":      "[commit]\nthreshold = 1\n[commit]\nlanguage = \"en\"\n",
		"key twice":  "[commit]\nlanguage = \"en\"\nlanguage = \"de\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Remove(in, "commit", "language"); !errors.Is(err, ErrAmbiguous) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestAppendBlockAddsATableListEntry(t *testing.T) {
	got := AppendBlock("[area]\nscope = \"s\"\n", "policy.commands.rules",
		[][2]string{{"regex", `'pip\s+install'`}, {"reason", `"uv, never pip."`}})
	want := "[area]\nscope = \"s\"\n\n[[policy.commands.rules]]\nregex = 'pip\\s+install'\nreason = \"uv, never pip.\"\n"
	if got != want {
		t.Fatalf("got\n%s", got)
	}
	if got := AppendBlock("", "x", nil); got != "[[x]]\n" {
		t.Fatalf("an empty file gets the block alone, got %q", got)
	}
}

// TestSetSkipsATableListOfTheSameName: [[commit]] entries are not the
// [commit] table, so a key goes into a new [commit] instead.
func TestSetSkipsATableListOfTheSameName(t *testing.T) {
	got, err := Set("[[commit]]\nlanguage = \"en\"\n", "commit", "threshold", "3")
	if err != nil {
		t.Fatal(err)
	}
	if got != "[[commit]]\nlanguage = \"en\"\n\n[commit]\nthreshold = 3\n" {
		t.Fatalf("%q", got)
	}
	got, err = Set("[[commit]]\nlanguage = \"en\"\n", "commit", "language", `"de"`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[[commit]]\nlanguage = \"en\"\n\n[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("a key of a list entry is not the table's: %q", got)
	}
}

// TestSetReadsEscapesOnlyInBasicStrings: a backslash escapes in "…" and is
// a plain character in '…'. Misread either way, a string ends in the wrong
// place, and what follows it — three quotes, a bracket, a # — is misjudged.
func TestSetReadsEscapesOnlyInBasicStrings(t *testing.T) {
	for name, c := range map[string]struct{ in, want string }{
		"three quotes inside a basic string": {
			"[area]\nnote = \"x'''a\\\"'''\"\n[commit]\nlanguage = \"en\"\n",
			"[area]\nnote = \"x'''a\\\"'''\"\n[commit]\nlanguage = \"de\"\n",
		},
		"a literal string ending in a backslash": {
			"[commit]\nlanguage = 'C:\\' # a '''comment'''\n",
			"[commit]\nlanguage = \"de\" # a '''comment'''\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Set(c.in, "commit", "language", `"de"`)
			if err != nil || got != c.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
	// A list whose strings end wrongly would seem to stay open and swallow
	// the next key, which Set would then add a second time.
	in := "[index]\ninclude = ['a\\', \"bc\"]\nexclude = []\n"
	got, err := Set(in, "index", "exclude", `["x"]`)
	if err != nil || got != "[index]\ninclude = ['a\\', \"bc\"]\nexclude = [\"x\"]\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

// TestSetSurvivesAListLeftOpenAtTheEnd: an unclosed [ runs to the last line
// and no further.
func TestSetSurvivesAListLeftOpenAtTheEnd(t *testing.T) {
	got, err := Set("[area]\nk = [\n", "commit", "language", `"de"`)
	if err != nil || got != "[area]\nk = [\n\n[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

// TestRemoveKeepsTheLineBeforeAnEmptiedSection: only a blank line goes with
// the header; a key or comment right above it is the human's.
func TestRemoveKeepsTheLineBeforeAnEmptiedSection(t *testing.T) {
	got, err := Remove("[area]\nscope = \"s\"\n[modules]\ngraph = false\n", "modules", "graph")
	if err != nil || got != "[area]\nscope = \"s\"\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

// TestRemoveKeepsAHeaderThatStillHoldsComments: without its header a
// comment in the section would read as one of the section above.
func TestRemoveKeepsAHeaderThatStillHoldsComments(t *testing.T) {
	for in, want := range map[string]string{
		"[commit]\nlanguage = \"de\"\n\n[verify]\n# a note\nbudget = 5\n":                "[commit]\nlanguage = \"de\"\n\n[verify]\n# a note\n",
		"[verify]\nbudget = 5\n# trailing note\n\n[commit]\nlanguage = \"de\"\n":         "[verify]\n# trailing note\n\n[commit]\nlanguage = \"de\"\n",
		"[commit]\nlanguage = \"de\"\n\n[verify]\nbudget = 5\n\n[area]\nscope = \"s\"\n": "[commit]\nlanguage = \"de\"\n\n[area]\nscope = \"s\"\n",
		"[verify]\n\nbudget = 5\n\n": "",
	} {
		got, err := Remove(in, "verify", "budget")
		if err != nil || got != want {
			t.Errorf("Remove(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

// TestRemoveTakesATopLevelKey: a key above every header has no header to
// drop with it.
func TestRemoveTakesATopLevelKey(t *testing.T) {
	got, err := Remove("k = 1\n", "", "k")
	if err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}

// A byte order mark is no line of the document: the scan reads past it, and
// every edit hands it back where it stood.
func TestAByteOrderMarkIsKept(t *testing.T) {
	const bom = "\uFEFF"
	got, err := Set(bom+"[commit]\nlanguage = \"en\"\n", "commit", "language", `"de"`)
	if err != nil || got != bom+"[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("Set = %q, %v", got, err)
	}
	got, err = Set(bom+"threshold = 1\n", "commit", "threshold", "2")
	if err != nil || got != bom+"threshold = 1\n\n[commit]\nthreshold = 2\n" {
		t.Fatalf("Set on a top-level key = %q, %v", got, err)
	}
	got, err = Remove(bom+"# c\n[commit]\nthreshold = 1\n", "commit", "threshold")
	if err != nil || got != bom+"# c\n" {
		t.Fatalf("Remove = %q, %v", got, err)
	}
	got = AppendBlock(bom+"[commit]\n", "commit.allow", [][2]string{{"regex", `"x"`}})
	if got != bom+"[commit]\n\n[[commit.allow]]\nregex = \"x\"\n" {
		t.Fatalf("AppendBlock = %q", got)
	}
	if got = AppendBlock(bom, "commit.allow", nil); got != bom+"[[commit.allow]]\n" {
		t.Fatalf("AppendBlock on a bare mark = %q", got)
	}
}

// Only the value is replaced: the indentation and the spacing around = are
// the human's.
func TestSetKeepsTheShapeOfTheLineItReplaces(t *testing.T) {
	for in, want := range map[string]string{
		"[commit]\n\tthreshold=1\n":            "[commit]\n\tthreshold=2\n",
		"[commit]\n  threshold   =   1  # c\n": "[commit]\n  threshold   =   2  # c\n",
		"[commit]\nthreshold =\t1\n":           "[commit]\nthreshold =\t2\n",
		"[commit]\n  threshold = [\n  1,\n]\n": "[commit]\n  threshold = 2\n",
	} {
		got, err := Set(in, "commit", "threshold", "2")
		if err != nil || got != want {
			t.Errorf("Set(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}
