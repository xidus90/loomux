package ask

import (
	"math"
	"testing"

	"github.com/xidus90/loomux/internal/code/lexicon"
)

func TestIDFFallsAsATermSpreads(t *testing.T) {
	// idf = log(1 + n/(1+df)). A term in one document of a hundred
	// discriminates; a term in every document does not.
	rare := idf(1, 100)
	common := idf(100, 100)
	if !(rare > common) {
		t.Fatalf("rare = %v, common = %v, want rare to weigh more", rare, common)
	}
	want := math.Log(1 + 100.0/2.0)
	if math.Abs(rare-want) > 1e-12 {
		t.Errorf("idf(1, 100) = %v, want %v", rare, want)
	}
}

func TestIDFOfAnUnseenTermIsTheCorpusWideValue(t *testing.T) {
	// df = 0: no document holds it, so the weight is the largest the corpus
	// allows rather than a division by zero.
	if got := idf(0, 10); math.Abs(got-math.Log(1+10.0)) > 1e-12 {
		t.Fatalf("idf(0, 10) = %v", got)
	}
}

func TestWeightsReadEveryQueryTermAgainstTheCorpus(t *testing.T) {
	ix := &lexicon.Index{DocCount: 10, DF: map[string]int{"cache": 4}}

	got := weights(map[string]int{"cache": 1, "unseen": 2}, ix)

	if len(got) != 2 {
		t.Fatalf("weights = %v, want one entry per query term", got)
	}
	if math.Abs(got["cache"]-math.Log(1+10.0/5.0)) > 1e-12 {
		t.Errorf("weights[cache] = %v, want idf(4, 10)", got["cache"])
	}
	// A term the corpus never saw has no key in this DF -- the map is there,
	// the key is not -- and a lookup of a missing key yields the value type's
	// zero. So df = 0, the corpus-wide weight, and not a lookup failure a
	// caller would have to notice.
	if math.Abs(got["unseen"]-math.Log(1+10.0)) > 1e-12 {
		t.Errorf("weights[unseen] = %v, want idf(0, 10)", got["unseen"])
	}
}

func TestWeightsReadACorpusWithoutADocumentFrequencyMap(t *testing.T) {
	// A nil DF is the other road to the same zero, and a different one: no map
	// at all rather than a map without the key. An Index decoded from a sidecar
	// with no "df" member, or zero-valued in a caller, has to weigh every term
	// at the corpus-wide value instead of panicking -- reading a nil map is
	// legal in Go, writing one is not, and weights only reads.
	ix := &lexicon.Index{DocCount: 10}

	got := weights(map[string]int{"cache": 1}, ix)

	if math.Abs(got["cache"]-math.Log(1+10.0)) > 1e-12 {
		t.Errorf("weights[cache] = %v, want idf(0, 10)", got["cache"])
	}
}

func TestOverlapWeighsEachSharedTermByItsIDF(t *testing.T) {
	query := map[string]int{"cache": 1, "get": 1}
	doc := map[string]int{"cache": 2}
	weights := map[string]float64{"cache": 3, "get": 9}

	// 1 * 2 * 3 -- the query count, the document count, the term's weight. The
	// term the document lacks contributes nothing, however heavy.
	if got := overlap(newQuestion(query), doc, weights); math.Abs(got-6) > 1e-12 {
		t.Fatalf("overlap = %v, want 6", got)
	}
}

func TestOverlapMultipliesByTheQueryCount(t *testing.T) {
	// 3 * 2 * 5 = 30 -- a word the question asked for three times against a
	// document that says it twice. The query count is a factor and not a flag:
	// dropping it would leave 2 * 5 = 10.
	got := overlap(newQuestion(map[string]int{"cache": 3}), map[string]int{"cache": 2},
		map[string]float64{"cache": 5})
	if math.Abs(got-30) > 1e-12 {
		t.Fatalf("overlap = %v, want 30", got)
	}
}

func TestBM25HoldsK1AndBToTheirValues(t *testing.T) {
	query := map[string]int{"cache": 1}

	// Both cases run at a length ratio other than 1, because a ratio of exactly
	// 1 collapses (1 - b + b*ratio) to 1 and hides b entirely.
	//
	// Repeated term, ratio 6/4 = 1.5:
	//   norm  = 1.2 * (1 - 0.75 + 0.75*1.5) = 1.2 * 1.375 = 1.65
	//   score = 1 * 3 * 2.2 / (3 + 1.65)    = 6.6 / 4.65
	repeated := bm25(newQuestion(query), map[string]int{"cache": 3},
		map[string]float64{"cache": 1}, 6, 4)
	if math.Abs(repeated-1.4193548387096774) > 1e-12 {
		t.Errorf("repeated = %v, want 6.6/4.65 -- k1 = 1.2 and b = 0.75", repeated)
	}

	// Long body at the same average, ratio 10/4 = 2.5, and an idf of 2:
	//   norm  = 1.2 * (1 - 0.75 + 0.75*2.5) = 1.2 * 2.125 = 2.55
	//   score = 2 * 1 * 2.2 / (1 + 2.55)    = 4.4 / 3.55
	long := bm25(newQuestion(query), map[string]int{"cache": 1},
		map[string]float64{"cache": 2}, 10, 4)
	if math.Abs(long-1.2394366197183098) > 1e-12 {
		t.Errorf("long = %v, want 4.4/3.55 -- k1 = 1.2 and b = 0.75", long)
	}
}

func TestBM25SaturatesARepeatedTerm(t *testing.T) {
	query := map[string]int{"cache": 1}
	weights := map[string]float64{"cache": 1}

	once := bm25(newQuestion(query), map[string]int{"cache": 1}, weights, 1, 1)
	tenTimes := bm25(newQuestion(query), map[string]int{"cache": 10}, weights, 10, 1)
	// k1 = 1.2 saturates: ten occurrences are worth more than one and nowhere
	// near ten times as much. Raw term frequency is what BM25 exists to avoid.
	if !(tenTimes > once) {
		t.Fatalf("ten = %v, once = %v", tenTimes, once)
	}
	if tenTimes > once*3 {
		t.Errorf("ten = %v against once = %v: a repeat must saturate", tenTimes, once)
	}
}

func TestBM25PenalizesALongBody(t *testing.T) {
	query := map[string]int{"cache": 1}
	weights := map[string]float64{"cache": 1}

	short := bm25(newQuestion(query), map[string]int{"cache": 1}, weights, 5, 100)
	long := bm25(newQuestion(query), map[string]int{"cache": 1}, weights, 500, 100)
	// b = 0.75 normalizes by length, so a long definition does not win on bulk.
	if !(short > long) {
		t.Fatalf("short = %v, long = %v, want the short body to score higher", short, long)
	}
}

func TestBM25OfAnEmptyCorpusIsZeroAndNotNaN(t *testing.T) {
	got := bm25(newQuestion(map[string]int{"x": 1}), map[string]int{"x": 1}, map[string]float64{"x": 1}, 0, 0)
	if math.IsNaN(got) || math.IsInf(got, 0) {
		t.Fatalf("got %v, want a finite value: an empty corpus must not poison a score", got)
	}
}

func TestLexicalWeighsNameThreeAndPathTwo(t *testing.T) {
	weights := map[string]float64{"cache": 1}
	query := map[string]int{"cache": 1}

	byName := lexical(newQuestion(query), lexicon.Doc{Name: map[string]int{"cache": 1}}, weights, 1)
	byPath := lexical(newQuestion(query), lexicon.Doc{Path: map[string]int{"cache": 1}}, weights, 1)
	byBody := lexical(newQuestion(query), lexicon.Doc{Body: map[string]int{"cache": 1}}, weights, 1)

	// 3 and 2 are not to be rounded and not to be invented: they decide whether
	// a name hit beats a path hit.
	if math.Abs(byName-3) > 1e-12 {
		t.Errorf("name score = %v, want 3", byName)
	}
	if math.Abs(byPath-2) > 1e-12 {
		t.Errorf("path score = %v, want 2", byPath)
	}
	if !(byName > byPath && byPath > byBody) {
		t.Errorf("name %v, path %v, body %v -- want that order", byName, byPath, byBody)
	}
}

func TestLexicalOfANonMatchIsZero(t *testing.T) {
	got := lexical(newQuestion(map[string]int{"nothing": 1}),
		lexicon.Doc{Name: map[string]int{"cache": 1}},
		map[string]float64{"nothing": 5}, 1)
	if got != 0 {
		t.Fatalf("got %v, want 0 -- a document the query never touched scores nothing", got)
	}
}

// The three terms below are summed in a deliberately brittle arrangement: at
// 1e16 the spacing of a float64 is 2, so 1e16+1 rounds back to 1e16 and the
// two small terms only survive if they are added to each other BEFORE the
// large one arrives. Sorted ascending by term name -- "aaa", "bbb", "zzz" --
// they are, and the sum is exactly 1e16+2. A range over the map does not hand
// them over sorted in every run, and such a run sums to 1e16 -- restoring the
// two unordered loops made these assertions fail at repeat 3 and 11, measured.
// That is the whole defect: a query's terms were summed
// in the iteration order of a map, which Go randomizes per range, so the same
// query could produce different last bits between two runs and break a tie on
// noise instead of on the node id -- while an Answer promises descending
// score, ascending id. pagerank.Rank guards the same arithmetic the same way.
//
// The order is now newQuestion's, built once per query rather than per sum, and
// these two tests reach the sums only through it -- so what they discriminate
// is newQuestion's sort. Dropping it leaves them summing in map order again,
// which is what they measure.
var (
	orderSensitiveQuery = map[string]int{"aaa": 1, "bbb": 1, "zzz": 1}
	orderSensitiveIDF   = map[string]float64{"aaa": 1, "bbb": 1, "zzz": 1e16}
	orderSensitiveSum   = 1e16 + 2
)

// orderSensitiveRuns is how often each assertion repeats. A single range can
// happen to hand the terms over in sorted order; fifty in a row cannot.
const orderSensitiveRuns = 50

func TestOverlapSumsTheTermsInSortedOrder(t *testing.T) {
	doc := map[string]int{"aaa": 1, "bbb": 1, "zzz": 1}
	for i := 0; i < orderSensitiveRuns; i++ {
		if got := overlap(newQuestion(orderSensitiveQuery), doc, orderSensitiveIDF); got != orderSensitiveSum {
			t.Fatalf("run %d: overlap = %v, want the sorted-order sum %v", i, got, orderSensitiveSum)
		}
	}
}

func TestBM25SumsTheTermsInSortedOrder(t *testing.T) {
	body := map[string]int{"aaa": 1, "bbb": 1, "zzz": 1}
	// bodyLen == avgLen makes norm 1.2 and the BM25 factor 1*(1.2+1)/(1+1.2)
	// exactly 1, so each term contributes its idf and nothing else.
	for i := 0; i < orderSensitiveRuns; i++ {
		if got := bm25(newQuestion(orderSensitiveQuery), body, orderSensitiveIDF, 1, 1); got != orderSensitiveSum {
			t.Fatalf("run %d: bm25 = %v, want the sorted-order sum %v", i, got, orderSensitiveSum)
		}
	}
}
