package verify

import (
	"strings"
	"testing"
)

func TestTheMatrixLoads(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	var gdunit *Framework
	for i := range fs {
		if fs[i].Stack == "gdscript" && fs[i].Name == "gdUnit4" {
			gdunit = &fs[i]
		}
	}
	if gdunit == nil {
		t.Fatal("no row for gdUnit4")
	}
	if gdunit.Kind != "test" || gdunit.Found == "" || gdunit.Carries == "" || gdunit.Source == "" {
		t.Fatalf("%#v", *gdunit)
	}
	if !gdunit.Proven {
		t.Error("gdUnit4 on Godot 4 is the one line this repository has run itself")
	}
}

func TestADuplicateRowIsALoadError(t *testing.T) {
	// the last duplicate must not win silently.
	body := "[[framework]]\nstack=\"gdscript\"\nname=\"GUT\"\nkind=\"test\"\nfound=\"plugin.cfg:GUT\"\ncarries=\"3.x\"\nsource=\"https://example.com\"\n"
	_, err := parseFrameworks([]byte(body+body), "frameworks.toml")
	if err == nil || !strings.Contains(err.Error(), "gdscript/GUT") {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(err.Error(), "frameworks.toml:8") {
		t.Errorf("error should name the second row's line: %v", err)
	}
}

func TestAnUnknownKeyOrKindIsALoadError(t *testing.T) {
	// a misspelt key or kind must not look configured.
	for _, body := range []string{
		"[[framework]]\nstack=\"gdscript\"\nname=\"X\"\nkidn=\"test\"\n",
		"[[framework]]\nstack=\"gdscript\"\nname=\"X\"\nkind=\"typecheck\"\nfound=\"a:b\"\ncarries=\"c\"\nsource=\"d\"\n",
	} {
		if _, err := parseFrameworks([]byte(body), "frameworks.toml"); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}

func TestAMissingProvenIsFalse(t *testing.T) {
	// an omitted proven is false, never a promise.
	body := "[[framework]]\nstack=\"gdscript\"\nname=\"X\"\nkind=\"test\"\nfound=\"a:b\"\ncarries=\"c\"\nsource=\"d\"\n"
	fs, err := parseFrameworks([]byte(body), "frameworks.toml")
	if err != nil || len(fs) != 1 || fs[0].Proven {
		t.Fatalf("%#v %v", fs, err)
	}
}

func TestMalformedTOMLAndMissingFieldsAreLoadErrors(t *testing.T) {
	row := "[[framework]]\nstack=\"s\"\nname=\"n\"\nkind=\"test\"\nfound=\"a:b\"\ncarries=\"c\"\nsource=\"d\"\n"
	for name, body := range map[string]string{
		"syntax":  "[[framework\n",
		"stack":   strings.Replace(row, "stack=\"s\"\n", "", 1),
		"name":    strings.Replace(row, "name=\"n\"\n", "", 1),
		"carries": strings.Replace(row, "carries=\"c\"\n", "", 1),
		"source":  strings.Replace(row, "source=\"d\"\n", "", 1),
	} {
		_, err := parseFrameworks([]byte(body), "frameworks.toml")
		if err == nil || !strings.HasPrefix(err.Error(), "frameworks.toml") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestRowsInOddButValidFormsDoNotPanic(t *testing.T) {
	inline := "framework = [{stack = \"s\", name = \"n\", kind = \"test\", found = \"a:b\", carries = \"c\", source = \"d\"}]\n"
	commented := "[[framework]] # note\nstack=\"s\"\nname=\"n\"\nkind=\"test\"\nfound=\"a:b\"\ncarries=\"c\"\nsource=\"d\"\n"
	for name, body := range map[string]string{"inline": inline, "commented": commented} {
		fs, err := parseFrameworks([]byte(body), "frameworks.toml")
		if err != nil || len(fs) != 1 {
			t.Errorf("%s: %v %v", name, fs, err)
		}
	}
	// No header to point at for an inline row, so the error names the file only.
	_, err := parseFrameworks([]byte(strings.Replace(inline, "test", "x", 1)), "frameworks.toml")
	if err == nil || !strings.HasPrefix(err.Error(), "frameworks.toml: ") {
		t.Errorf("%v", err)
	}
}

func TestAMisspeltTableAndPrefixKeysGetTheRightLine(t *testing.T) {
	_, err := parseFrameworks([]byte("# c\n[[framwork]]\nstack=\"s\"\n"), "frameworks.toml")
	if err == nil || !strings.Contains(err.Error(), "frameworks.toml:2: unknown key \"framwork\"") {
		t.Errorf("%v", err)
	}
	// "s" must not match the line of "stack".
	body := "[[framework]]\nstack=\"s\"\nname=\"n\"\nkind=\"test\"\nfound=\"a:b\"\ncarries=\"c\"\nsource=\"d\"\ns = 1\n"
	_, err = parseFrameworks([]byte(body), "frameworks.toml")
	if err == nil || !strings.Contains(err.Error(), "frameworks.toml:8: unknown key \"s\"") {
		t.Errorf("%v", err)
	}
}

func TestOnlyTheGodot4RowsAreProven(t *testing.T) {
	want := map[string]bool{"gdUnit4": true, "NanoCoverage": true, "gdtoolkit4": true, "GUT": false, "gdtoolkit3": false, "godot-code-coverage": false}
	fs, err := LoadFrameworks()
	if err != nil || len(fs) != len(want) {
		t.Fatalf("%d rows, %v", len(fs), err)
	}
	for _, f := range fs {
		if p, ok := want[f.Name]; !ok || p != f.Proven {
			t.Errorf("%s: proven=%v", f.Name, f.Proven)
		}
	}
}

func TestAnUnknownKeyInAnInlineTableStillFails(t *testing.T) {
	// The key starts no line, so no line can be named.
	body := "framework = [{stack = \"s\", kidn = \"x\"}]\n"
	_, err := parseFrameworks([]byte(body), "frameworks.toml")
	if err == nil || !strings.Contains(err.Error(), "kidn") {
		t.Fatalf("%v", err)
	}
}

func TestAnUnknownKeyNamesItsRow(t *testing.T) {
	body := "[[framework]]\nstack=\"s\"\nname=\"n\"\nkidn=\"test\"\n"
	_, err := parseFrameworks([]byte(body), "frameworks.toml")
	if err == nil || !strings.Contains(err.Error(), "kidn") || !strings.Contains(err.Error(), "frameworks.toml:4") {
		t.Fatalf("%v", err)
	}
}

func TestARowWithoutAPatternLoads(t *testing.T) {
	// A lane that detects nothing leaves its rows nothing to repeat.
	body := "[[framework]]\nstack=\"s\"\nname=\"n\"\nkind=\"lint\"\ncarries=\"c\"\nsource=\"d\"\n"
	fs, err := parseFrameworks([]byte(body), "frameworks.toml")
	if err != nil || len(fs) != 1 || fs[0].Found != "" {
		t.Fatalf("%#v %v", fs, err)
	}
}

func TestThePageReaderNamesWhatItCannotRead(t *testing.T) {
	// A page out of shape must say where, instead of taking the test down
	// with it.
	for name, page := range map[string]struct{ text, want string }{
		"no header":   {"|---|---|\n", "p.md:1: a separator row with nothing above it to be its header"},
		"short row":   {"| Framework | Kind |\n|---|---|\n| gdUnit4 | test |\n", "p.md:3: 2 cells, but the framework table has five columns"},
		"long row":    {"| Framework |\n|---|\n| a | b | c | d | e | f |\n", "p.md:3: 6 cells, but the framework table has five columns"},
		"other table": {"| Lane | Kind |\n|---|---|\n| lint | lint |\n", ""},
	} {
		rows, complaints := frameworkRowsIn("p.md", page.text, " (unproven)")
		if page.want == "" {
			if len(rows) != 0 || len(complaints) != 0 {
				t.Errorf("%s: %v %v", name, rows, complaints)
			}
			continue
		}
		if len(complaints) != 1 || complaints[0] != page.want {
			t.Errorf("%s: %v", name, complaints)
		}
	}
}
