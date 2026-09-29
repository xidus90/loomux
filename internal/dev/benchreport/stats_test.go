package benchreport

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestMedianOfAnOddCountIsTheMiddle(t *testing.T) {
	if got := Median([]float64{5, 1, 3}); got != 3 {
		t.Fatalf("Median = %v, want 3", got)
	}
}

func TestMedianOfAnEvenCountAveragesTheMiddleTwo(t *testing.T) {
	if got := Median([]float64{4, 1, 2, 3}); got != 2.5 {
		t.Fatalf("Median = %v, want 2.5", got)
	}
}

func TestMedianOfNothingIsZeroAndLeavesTheInputUnsorted(t *testing.T) {
	in := []float64{3, 1, 2}
	Median(in)
	if !slices.Equal(in, []float64{3, 1, 2}) {
		t.Fatalf("input reordered to %v", in)
	}
	if Median(nil) != 0 {
		t.Fatal("Median(nil) != 0")
	}
}

func TestMeanAveragesEveryValueAndTakesZeroForNone(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"none", nil, 0},
		{"one", []float64{7}, 7},
		// Unsorted input: the order of the runs does not change the mean.
		{"spread", []float64{10, 0, 5}, 5},
		{"even", []float64{1, 2, 3, 4}, 2.5},
		// The case that tells the mean from the median: a skewed series has
		// the mean 4 and the median 2.
		{"skewed", []float64{1, 2, 9}, 4},
	}
	for _, c := range cases {
		if got := Mean(c.in); got != c.want {
			t.Errorf("%s: Mean(%v) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

func TestMSIsTheDivisionTheOldRendererDid(t *testing.T) {
	d := 89138123 * time.Nanosecond
	if MS(d) != float64(d)/float64(time.Millisecond) {
		t.Fatal("MS differs from float64(d)/1e6")
	}
}

func TestSummarizeKeepsColdApartFromTheWarmRuns(t *testing.T) {
	got := Summarize("keyword", 900, []float64{12, 10, 11})
	want := Timing{Name: "keyword", ColdMS: 900, WarmMS: []float64{12, 10, 11},
		MedianMS: 11, MinMS: 10, MaxMS: 12}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSummarizeOfNoWarmRunsLeavesTheSpreadAtZero(t *testing.T) {
	got := Summarize("cold only", 7, nil)
	if got.MedianMS != 0 || got.MinMS != 0 || got.MaxMS != 0 || got.ColdMS != 7 {
		t.Fatalf("got %+v", got)
	}
}

func TestRowSpellsEveryTimeWithOneDecimal(t *testing.T) {
	row := Timing{Name: "name", ColdMS: 1.23, MedianMS: 3.44, MinMS: 1, MaxMS: 5}.Row()
	if row != "| name | 1.2 ms | 3.4 ms | 1.0 ms | 5.0 ms |" {
		t.Fatalf("Row = %q", row)
	}
}
