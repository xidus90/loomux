package linecov

import (
	"reflect"
	"testing"
)

func TestGodotCfgReadsAPackedStringArrayAsAList(t *testing.T) {
	cfg := "[nano_coverage]\ngeneral/ignore_paths=PackedStringArray(\"a\", \"b\")\n"
	vals, found, err := GodotCfg([]byte(cfg), "nano_coverage")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if got := vals["general/ignore_paths"]; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("got %q", got)
	}
}

func TestGodotCfgKeepsAnEmptyStringOnlyForOutsideCallers(t *testing.T) {
	cfg := []byte("[g]\r\nk=\"\"\r\nj=\"x\"\r\n")
	vals, found, err := GodotCfgKeepEmpty(cfg, "g")
	if err != nil || !found {
		t.Fatal(found, err)
	}
	if got, ok := vals["k"]; !ok || !reflect.DeepEqual(got, []string{""}) || !reflect.DeepEqual(vals["j"], []string{"x"}) {
		t.Fatalf("got %q", vals)
	}
	for _, read := range []func() (Values, bool, error){
		func() (Values, bool, error) { return GodotCfg(cfg, "g") },
		func() (Values, bool, error) { return readGodotCfg(cfg, Where{Section: "g"}) },
	} {
		vals, found, err := read()
		if err != nil || !found || len(vals["k"]) != 0 {
			t.Fatalf("the gate must keep dropping empty values: %q %v %v", vals, found, err)
		}
	}
}

func TestNanoMatcherReadsAPatternLikeNanoCoverage(t *testing.T) {
	m, err := NanoMatcher("**/*_test.gd")
	if err != nil {
		t.Fatal(err)
	}
	if !m("test/a_test.gd") || m("test/a.gd") {
		t.Fatal("pattern matched the wrong files")
	}
	if _, err := NanoMatcher("[a"); err == nil {
		t.Fatal("want an error for a broken pattern")
	}
}
