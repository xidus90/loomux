package linecov

import (
	"reflect"
	"testing"
)

func TestDetectFindsLCOVAfterABOMAndBlankLines(t *testing.T) {
	for _, content := range []string{"\xef\xbb\xbf\n  TN:\nSF:a\n", "SF:a\nDA:1,1\n"} {
		if f := Detect([]byte(content)); f == nil || f.Name() != "lcov" {
			t.Errorf("%q: %v", content, f)
		}
	}
	for _, content := range []string{"", "hello\nSF:a\n", "\x00\x01"} {
		if f := Detect([]byte(content)); f != nil {
			t.Errorf("%q detected as %s", content, f.Name())
		}
	}
}

func TestLCOVReadsLinesByPositionAndReportsWhatItCannotRead(t *testing.T) {
	report := "TN:\r\nSF:res://core/a.gd\r\nDA:3,0\r\nDA:4,2,abc123\r\nFN:1,f\r\nLF:2\r\nend_of_record\r\n" +
		"DA:9,1\nSF:core\\b.gd\nDA:x,1\nDA:0,1\nDA:5,1\nDA:5,2\nend_of_record\n"
	recs, problems := lcovFormat{}.Read([]byte(report), ReadContext{})
	want := []Record{
		{Path: "res://core/a.gd", Lines: map[int]uint64{3: 0, 4: 2}},
		{Path: `core\b.gd`, Lines: map[int]uint64{5: 3}},
	}
	if !reflect.DeepEqual(recs, want) {
		t.Fatalf("%+v", recs)
	}
	wantProblems := []string{`core\b.gd: unreadable record "DA:x,1"`, `core\b.gd: unreadable record "DA:0,1"`}
	if !reflect.DeepEqual(problems, wantProblems) {
		t.Fatalf("%q", problems)
	}
}

func TestLCOVAcceptsLineOneAndRefusesALineNumberTooLargeToHold(t *testing.T) {
	recs, problems := lcovFormat{}.Read([]byte("SF:a\nDA:1,1\nDA:99999999999999999999,1\n"), ReadContext{})
	if !reflect.DeepEqual(recs, []Record{{Path: "a", Lines: map[int]uint64{1: 1}}}) || len(problems) != 1 {
		t.Fatalf("%+v %q", recs, problems)
	}
}

func TestLCOVReadsTheFirstRecordAfterABOM(t *testing.T) {
	recs, problems := lcovFormat{}.Read([]byte("\xef\xbb\xbfSF:a.c\nDA:1,1\nend_of_record\n"), ReadContext{})
	if !reflect.DeepEqual(recs, []Record{{Path: "a.c", Lines: map[int]uint64{1: 1}}}) || len(problems) != 0 {
		t.Fatalf("%+v %q", recs, problems)
	}
}

func TestLCOVRefusesAHitCountTooLargeToHold(t *testing.T) {
	_, problems := lcovFormat{}.Read([]byte("SF:a\nDA:1,99999999999999999999999\n"), ReadContext{})
	if len(problems) != 1 {
		t.Fatalf("%q", problems)
	}
}
