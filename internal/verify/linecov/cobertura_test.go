package linecov

import (
	"reflect"
	"testing"
)

const coberturaReport = `<?xml version="1.0" ?>
<!DOCTYPE coverage SYSTEM "http://cobertura.sourceforge.net/xml/coverage-04.dtd">
<coverage line-rate="0.5" version="7.6.1">
  <sources><source> src </source><source>C:\work\app\lib</source></sources>
  <packages><package name="p"><classes>
    <class name="A" filename="app.py"><methods/><lines>
      <line number="1" hits="3"/><line number="2" hits="0"/>
    </lines></class>
    <class name="A.Inner" filename="app.py"><lines><line number="2" hits="1"/></lines></class>
  </classes></package></packages>
</coverage>`

func TestDetectTellsCoberturaByItsRootElement(t *testing.T) {
	if f := Detect([]byte(coberturaReport)); f == nil || f.Name() != "cobertura" {
		t.Fatalf("%v", f)
	}
	for _, s := range []string{"<report/>", "<coverage", "<!-- only a comment -->", "plain", `text before <coverage line-rate="1"/>`} {
		if f := Detect([]byte(s)); f != nil && f.Name() == "cobertura" {
			t.Errorf("%q detected as cobertura", s)
		}
	}
}

// Clover's root element is coverage too; only Cobertura's children or its
// line-rate attribute tell the two apart.
func TestDetectTellsCoberturaFromClover(t *testing.T) {
	for _, row := range []struct {
		doc  string
		want bool
	}{
		{`<coverage line-rate="1"/>`, true},
		{`<coverage><sources/></coverage>`, true},
		{`<coverage version="1"><!-- c --><packages/></coverage>`, true},
		{`<coverage generated="1" clover="4.4"><project timestamp="1"><file name="a.js"><line num="1" count="0" type="stmt"/></file></project></coverage>`, false},
		{`<coverage><project><packages/></project></coverage>`, false},
		{`<coverage><project>`, false},
		{`<coverage>`, false},
	} {
		if got := Detect([]byte(row.doc)); (got != nil && got.Name() == "cobertura") != row.want {
			t.Errorf("%s: %v", row.doc, got)
		}
	}
}

func TestCoberturaReadsEveryClassWithTheReportSources(t *testing.T) {
	recs, problems := coberturaFormat{}.Read([]byte(coberturaReport), ReadContext{})
	bases := []string{"src", `C:\work\app\lib`}
	want := []Record{
		{Path: "app.py", Bases: bases, Lines: map[int]uint64{1: 3, 2: 0}},
		{Path: "app.py", Bases: bases, Lines: map[int]uint64{2: 1}},
	}
	if len(problems) != 0 || !reflect.DeepEqual(recs, want) {
		t.Fatalf("%+v %q", recs, problems)
	}
}

func TestCoberturaReportsBrokenXMLAndUnreadableLines(t *testing.T) {
	if _, problems := (coberturaFormat{}).Read([]byte("<coverage><packages>"), ReadContext{}); len(problems) != 1 {
		t.Fatalf("%q", problems)
	}
	bad := `<coverage><packages><package><classes><class filename="a.py"><lines><line number="x" hits="1"/><line number="2" hits="-1"/></lines></class></classes></package></packages></coverage>`
	recs, problems := coberturaFormat{}.Read([]byte(bad), ReadContext{})
	if len(problems) != 2 || len(recs) != 1 || len(recs[0].Lines) != 0 {
		t.Fatalf("%+v %q", recs, problems)
	}
}
