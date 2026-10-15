package linecov

import (
	"reflect"
	"testing"
)

const jacocoReport = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<!DOCTYPE report PUBLIC "-//JACOCO//DTD Report 1.1//EN" "report.dtd">
<report name="app">
  <package name="com/x">
    <class name="com/x/Calc" sourcefilename="Calc.java"/>
    <sourcefile name="Calc.java">
      <line nr="3" mi="0" ci="2" mb="0" cb="0"/>
      <line nr="4" mi="3" ci="0" mb="0" cb="0"/>
    </sourcefile>
  </package>
  <package name="">
    <sourcefile name="Main.kt"><line nr="1" mi="0" ci="1"/></sourcefile>
  </package>
</report>`

func TestJaCoCoReadsCoveredInstructionsPerLineAndAsksForASuffixSearch(t *testing.T) {
	if f := Detect([]byte(jacocoReport)); f == nil || f.Name() != "jacoco" {
		t.Fatalf("%v", f)
	}
	recs, problems := jacocoFormat{}.Read([]byte(jacocoReport), ReadContext{})
	want := []Record{
		{Path: "com/x/Calc.java", Suffix: true, Lines: map[int]uint64{3: 2, 4: 0}},
		{Path: "Main.kt", Suffix: true, Lines: map[int]uint64{1: 1}},
	}
	if len(problems) != 0 || !reflect.DeepEqual(recs, want) {
		t.Fatalf("%+v %q", recs, problems)
	}
}

func TestJaCoCoReportsBrokenXMLAndUnreadableLines(t *testing.T) {
	if _, problems := (jacocoFormat{}).Read([]byte("<report><package>"), ReadContext{}); len(problems) != 1 {
		t.Fatalf("%q", problems)
	}
	bad := `<report><package name="p"><sourcefile name="A.java"><line nr="0" ci="1"/><line nr="2" ci="x"/></sourcefile></package></report>`
	recs, problems := jacocoFormat{}.Read([]byte(bad), ReadContext{})
	if len(problems) != 2 || len(recs) != 1 || len(recs[0].Lines) != 0 {
		t.Fatalf("%+v %q", recs, problems)
	}
}
