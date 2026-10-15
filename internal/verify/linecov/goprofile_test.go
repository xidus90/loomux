package linecov

import (
	"reflect"
	"testing"
)

const goReport = "mode: set\n" +
	"example.com/m/a.go:3.10,5.2 2 1\n" +
	"example.com/m/a.go:5.2,7.3 1 0\n" +
	"example.com/m/sub/b.go:1.1,1.9 1 0\n" +
	"other.org/x/c.go:1.1,2.2 1 1\n" +
	"other.org/x/c.go:4.1,4.2 1 1\n" +
	"garbage\n" +
	"example.com/m/a.go:9.1,8.1 1 1\n"

func TestGoProfileSpreadsBlocksOverLinesAndKeepsTheHighestCount(t *testing.T) {
	if f := Detect([]byte(goReport)); f == nil || f.Name() != "goprofile" {
		t.Fatalf("%v", f)
	}
	recs, problems := goProfileFormat{}.Read([]byte(goReport), ReadContext{Module: "example.com/m"})
	want := []Record{
		{Path: "a.go", Lines: map[int]uint64{3: 1, 4: 1, 5: 1, 6: 0, 7: 0}},
		{Path: "sub/b.go", Lines: map[int]uint64{1: 0}},
	}
	if !reflect.DeepEqual(recs, want) {
		t.Fatalf("%+v", recs)
	}
	wantProblems := []string{
		"other.org/x/c.go: outside module example.com/m",
		`unreadable record "garbage"`,
		`unreadable record "example.com/m/a.go:9.1,8.1 1 1"`,
	}
	if !reflect.DeepEqual(problems, wantProblems) {
		t.Fatalf("%q", problems)
	}
}

func TestGoProfileSkipsBlankLinesBeforeTheModeLine(t *testing.T) {
	recs, problems := goProfileFormat{}.Read([]byte("\n\r\nmode: set\nexample.com/m/a.go:1.1,1.9 1 1\n"), ReadContext{Module: "example.com/m"})
	if !reflect.DeepEqual(recs, []Record{{Path: "a.go", Lines: map[int]uint64{1: 1}}}) || len(problems) != 0 {
		t.Fatalf("%+v %q", recs, problems)
	}
}

func TestGoProfileNeedsAModule(t *testing.T) {
	recs, problems := goProfileFormat{}.Read([]byte(goReport), ReadContext{})
	if recs != nil || len(problems) != 1 {
		t.Fatalf("%+v %q", recs, problems)
	}
}
