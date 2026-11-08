package trigram

import (
	"encoding/binary"
	"reflect"
	"testing"
)

// candidatesCorpus is the corpus of the candidate tests: two files sharing a
// prefix, one matching only a later trigram, one matching nothing.
func candidatesCorpus() []File {
	return []File{
		{Rel: "a.txt", Data: []byte("abcde")},
		{Rel: "b.txt", Data: []byte("abcxx")},
		{Rel: "c.txt", Data: []byte("zzzzz")},
		{Rel: "d.txt", Data: []byte("bcd")},
	}
}

// candidatesIndex builds the corpus' index and fails the test if it cannot.
func candidatesIndex(t *testing.T) *Index {
	t.Helper()
	ix, err := decode(buildTest(t, candidatesCorpus()))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return ix
}

// TestCandidatesIntersect pins that all trigrams of one literal have to be
// present: only a.txt has both abc and bcd.
func TestCandidatesIntersect(t *testing.T) {
	ix := candidatesIndex(t)
	got, used := ix.Candidates([][]string{{"abcd"}})
	if !used {
		t.Fatal("Candidates(abcd) reported no condition")
	}
	if want := []string{"a.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Candidates(abcd) = %v, want %v", got, want)
	}
}

// TestDisjunctionUnion pins that alternatives are a union: abc or bcd.
func TestDisjunctionUnion(t *testing.T) {
	ix := candidatesIndex(t)
	got, used := ix.Candidates([][]string{{"abc"}, {"bcd"}})
	if !used {
		t.Fatal("Candidates reported no condition")
	}
	if want := []string{"a.txt", "b.txt", "d.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Candidates = %v, want %v", got, want)
	}
}

// TestNoConditionUsesAll pins that an empty condition is "every file", not
// "no file".
func TestNoConditionUsesAll(t *testing.T) {
	ix := candidatesIndex(t)
	got, used := ix.Candidates(nil)
	if used {
		t.Fatal("Candidates(nil) reported a condition")
	}
	if want := []string{"a.txt", "b.txt", "c.txt", "d.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Candidates(nil) = %v, want %v", got, want)
	}
}

// TestMissingGramEmpty pins that a trigram no file carries means no match is
// possible, and that this is still a used condition.
func TestMissingGramEmpty(t *testing.T) {
	ix := candidatesIndex(t)
	got, used := ix.Candidates([][]string{{"qrs"}})
	if !used {
		t.Fatal("Candidates(qrs) reported no condition")
	}
	if len(got) != 0 {
		t.Fatalf("Candidates(qrs) = %v, want none", got)
	}
}

// TestEmptyConjunctionUsesAll pins that one alternative without literals makes
// the whole disjunction unconstrained.
func TestEmptyConjunctionUsesAll(t *testing.T) {
	ix := candidatesIndex(t)
	got, used := ix.Candidates([][]string{{}})
	if used {
		t.Fatal("Candidates({{}}) reported a condition")
	}
	if len(got) != len(candidatesCorpus()) {
		t.Fatalf("Candidates({{}}) = %v, want every file", got)
	}
}

// TestCandidatesShortLiteralUsesAll pins that a literal shorter than a trigram
// cannot narrow anything.
func TestCandidatesShortLiteralUsesAll(t *testing.T) {
	ix := candidatesIndex(t)
	if got, used := ix.Candidates([][]string{{"ab"}}); used || len(got) != 4 {
		t.Fatalf("Candidates(ab) = %v, %v; want all files, no condition", got, used)
	}
}

// TestCandidatesFallBackOnACorruptPosting pins that a broken posting list
// later in the conjunction turns the lookup into "search everything" instead
// of a silently short candidate set: bcd's ids get a file id the table lacks.
func TestCandidatesFallBackOnACorruptPosting(t *testing.T) {
	raw := buildTest(t, candidatesCorpus())
	gram, _ := Pack("bcd")
	table := headerSize() + fileTableSize(candidatesCorpus())
	gramCount := gramCountOf(raw)
	for i := 0; i < gramCount; i++ {
		entry := table + i*16
		if binary.LittleEndian.Uint32(raw[entry:]) != gram {
			continue
		}
		off := table + gramCount*16 + int(binary.LittleEndian.Uint64(raw[entry+4:]))
		binary.LittleEndian.PutUint32(raw[off:], 9)
	}
	ix := readRaw(t, raw)
	got, used := ix.Candidates([][]string{{"abcd"}})
	if used {
		t.Fatalf("Candidates(abcd) = %v used a broken posting list", got)
	}
	if want := ix.Files(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Candidates(abcd) = %v, want every file %v", got, want)
	}
}

// TestIntersectIDs pins the two-pointer intersection, all three comparisons
// included.
func TestIntersectIDs(t *testing.T) {
	got := intersectIDs([]uint32{2, 5, 9}, []uint32{1, 5, 7})
	if want := []uint32{5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("intersectIDs = %v, want %v", got, want)
	}
	if got := intersectIDs(nil, []uint32{1}); got == nil || len(got) != 0 {
		t.Fatalf("intersectIDs(nil, [1]) = %v, want empty", got)
	}
}

// TestMergeIDs pins the ordered union with duplicate removal.
func TestMergeIDs(t *testing.T) {
	got := mergeIDs([]uint32{1, 5, 9}, []uint32{1, 5, 7})
	if want := []uint32{1, 5, 7, 9}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeIDs = %v, want %v", got, want)
	}
	if got := mergeIDs(nil, []uint32{3}); !reflect.DeepEqual(got, []uint32{3}) {
		t.Fatalf("mergeIDs(nil, [3]) = %v, want [3]", got)
	}
	if got := mergeIDs([]uint32{3}, nil); !reflect.DeepEqual(got, []uint32{3}) {
		t.Fatalf("mergeIDs([3], nil) = %v, want [3]", got)
	}
}
