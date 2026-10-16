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
