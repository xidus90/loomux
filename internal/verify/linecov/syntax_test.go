package linecov

import (
	"reflect"
	"testing"
)

func TestINIReadsASectionWithListsAndContinuations(t *testing.T) {
	cfg := "# top\nfilter = src/\nfilter = lib/\n[run]\nSource = src,\n    pkg\nomit =\n    src/gen/*\n    */tests/*\n; note\nbranch: true\n[report]\nomit = other\n"
	vals, found, err := readINI([]byte(cfg), Where{Section: "run"})
	want := Values{"source": {"src", "pkg"}, "omit": {"src/gen/*", "*/tests/*"}, "branch": {"true"}}
	if err != nil || !found || !reflect.DeepEqual(vals, want) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
	top, found, err := readINI([]byte(cfg), Where{})
	if err != nil || !found || !reflect.DeepEqual(top, Values{"filter": {"src/", "lib/"}}) {
		t.Fatalf("%v %v %v", top, found, err)
	}
	if _, found, _ := readINI([]byte(cfg), Where{Section: "coverage:run"}); found {
		t.Fatal("found a section that is not there")
	}
}

func TestINIRefusesWhatConfigparserRefuses(t *testing.T) {
	for _, cfg := range []string{"[run\nx = 1\n", "  orphan\n", "no assignment\n", "=value\n"} {
		if _, _, err := readINI([]byte(cfg), Where{Section: "run"}); err == nil {
			t.Errorf("%q: no error", cfg)
		}
	}
}

func TestGodotCfgReadsPackedArraysStringsAndBooleans(t *testing.T) {
	cfg := "config_version=5\n\n[application]\nconfig/name=\"space\"\n\n[nano_coverage]\n; comment\n" +
		"general/ignore_paths=PackedStringArray(\"**/*_test.gd\", \"res://harness/(old)/**\",\n\"res://x\\\"y\")\n" +
		"general/ignore_addons=false\ngeneral/report_dir=\"res://coverage-report\"\ngeneral/odd=\"a\\q\"\n"
	vals, found, err := readGodotCfg([]byte(cfg), Where{Section: "nano_coverage"})
	want := Values{
		"general/ignore_paths":  {"**/*_test.gd", "res://harness/(old)/**", `res://x"y`},
		"general/ignore_addons": {"false"},
		"general/report_dir":    {"res://coverage-report"},
		"general/odd":           {`a\q`}, // an escape Go cannot unquote stays as written
	}
	if err != nil || !found || !reflect.DeepEqual(vals, want) {
		t.Fatalf("%q %v %v", vals, found, err)
	}
}

func TestSyntaxesRegistersEveryReaderUnderItsName(t *testing.T) {
	got := syntaxes()
	for _, name := range []string{"ini", "godotcfg"} {
		if got[name] == nil {
			t.Errorf("no syntax %q", name)
		}
	}
	vals, found, err := got["ini"]([]byte("[a]\nk = v\n"), Where{Section: "a"})
	if err != nil || !found || !reflect.DeepEqual(vals, Values{"k": {"v"}}) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
}

func TestGodotCfgSkipsMultiLineDictionariesAndArrays(t *testing.T) {
	cfg := "[input]\n\ncamera_focus_ship={\n\"deadzone\": 0.5,\n\"events\": [Object(InputEventKey,\"resource_local_to_scene\":false,\"keycode\":0)]\n}\n" +
		"list=[\n1,\n2\n]\nnote=\"two\nlines\"\n\n[nano_coverage]\ngeneral/ignore_addons=true\n"
	vals, found, err := readGodotCfg([]byte(cfg), Where{Section: "nano_coverage"})
	if err != nil || !found || !reflect.DeepEqual(vals, Values{"general/ignore_addons": {"true"}}) {
		t.Fatalf("%q %v %v", vals, found, err)
	}
	if _, _, err := readGodotCfg([]byte("[a]\nk=\"open\n"), Where{Section: "a"}); err == nil {
		t.Fatal("an unterminated string was read")
	}
}

func TestGodotCfgReportsAMissingSection(t *testing.T) {
	_, found, err := readGodotCfg([]byte("[application]\nk=1\n"), Where{Section: "nano_coverage"})
	if err != nil || found {
		t.Fatalf("%v %v", found, err)
	}
}

func TestGodotCfgRefusesBrokenFiles(t *testing.T) {
	for _, cfg := range []string{"[a\n", "[a]\nkey\n", "[a]\nk=PackedStringArray(\"x\"\n"} {
		if _, _, err := readGodotCfg([]byte(cfg), Where{Section: "a"}); err == nil {
			t.Errorf("%q: no error", cfg)
		}
	}
}
