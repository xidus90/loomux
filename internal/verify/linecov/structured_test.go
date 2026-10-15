package linecov

import (
	"reflect"
	"strings"
	"testing"
)

func TestTOMLReadsADottedTable(t *testing.T) {
	doc := "[tool.coverage.run]\nsource = [\"src\", \"pkg\"]\nbranch = true\nparallel = 4\nmixed = [\"a\", [\"b\"]]\n[tool.coverage.run.paths]\nx = [\"a\"]\n"
	vals, found, err := readTOML([]byte(doc), Where{Table: "tool.coverage.run"})
	want := Values{"source": {"src", "pkg"}, "branch": {"true"}, "parallel": {"4"}, "mixed": {"a", "b"}}
	if err != nil || !found || !reflect.DeepEqual(vals, want) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
	if _, found, err := readTOML([]byte("[project]\nname = \"x\"\n"), Where{Table: "tool.coverage.run"}); found || err != nil {
		t.Fatal("found a table that is not there")
	}
	if _, _, err := readTOML([]byte("[broken"), Where{Table: "run"}); err == nil {
		t.Fatal("no error")
	}
}

func TestJSONReadsTheRootOrAKeyPath(t *testing.T) {
	doc := `{"name":"app","nyc":{"include":["src/**"],"all":true,"n":1.5}}`
	vals, found, err := readJSON([]byte(doc), Where{Path: "nyc"})
	if err != nil || !found || !reflect.DeepEqual(vals, Values{"include": {"src/**"}, "all": {"true"}, "n": {"1.5"}}) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
	root, found, err := readJSON([]byte(`{"exclude":"x"}`), Where{})
	if err != nil || !found || !reflect.DeepEqual(root, Values{"exclude": {"x"}}) {
		t.Fatalf("%v %v %v", root, found, err)
	}
	if _, found, _ := readJSON([]byte(`{"nyc":3}`), Where{Path: "nyc"}); found {
		t.Fatal("a scalar is not a section")
	}
	if _, _, err := readJSON([]byte(`{`), Where{}); err == nil {
		t.Fatal("no error")
	}
}

func TestXMLReadsElementsAndAttributesBelowAPath(t *testing.T) {
	doc := `<?xml version="1.0"?><phpunit><source>
	  <include><directory suffix=".php">src</directory><file>bootstrap.php</file></include>
	  <exclude><directory>src/Legacy</directory></exclude>
	</source></phpunit>`
	vals, found, err := readXML([]byte(doc), Where{Path: "phpunit/source"})
	want := Values{
		"include/directory":        {"src"},
		"include/directory@suffix": {".php"},
		"include/file":             {"bootstrap.php"},
		"exclude/directory":        {"src/Legacy"},
	}
	if err != nil || !found || !reflect.DeepEqual(vals, want) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
	for _, path := range []string{"other/source", "phpunit/coverage"} {
		if _, found, _ := readXML([]byte(doc), Where{Path: path}); found {
			t.Errorf("%s found", path)
		}
	}
	if _, _, err := readXML([]byte("<a><b></a>"), Where{Path: "a"}); err == nil {
		t.Fatal("no error")
	}
	if _, _, err := readXML([]byte("  "), Where{Path: "a"}); err == nil {
		t.Fatal("an empty document was read")
	}
}

func TestXMLWantsExactlyOneRoot(t *testing.T) {
	for _, doc := range []string{"<a/><b/>", "<!-- none -->"} {
		_, _, err := readXML([]byte(doc), Where{Path: "a"})
		if err == nil || !strings.Contains(err.Error(), "exactly one root") {
			t.Errorf("%q: %v", doc, err)
		}
	}
}

func TestXMLReadsTextOnlyFromLeaves(t *testing.T) {
	vals, found, err := readXML([]byte("<r><a>stray<b>x</b></a></r>"), Where{Path: "r"})
	if err != nil || !found || !reflect.DeepEqual(vals, Values{"a/b": {"x"}}) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
}

func TestXMLMergesEveryElementOnThePathAndSplitsLists(t *testing.T) {
	doc := `<RunSettings><DataCollectionRunSettings><DataCollectors>
	  <DataCollector><Configuration><ExcludeByFile>**/gen/*.cs,**/obj/**</ExcludeByFile></Configuration></DataCollector>
	  <DataCollector><Configuration><ExcludeByFile>x.cs</ExcludeByFile></Configuration></DataCollector>
	</DataCollectors></DataCollectionRunSettings></RunSettings>`
	vals, found, err := readXML([]byte(doc), Where{Path: "RunSettings/DataCollectionRunSettings/DataCollectors/DataCollector/Configuration"})
	if err != nil || !found || !reflect.DeepEqual(vals, Values{"ExcludeByFile": {"**/gen/*.cs", "**/obj/**", "x.cs"}}) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
}
