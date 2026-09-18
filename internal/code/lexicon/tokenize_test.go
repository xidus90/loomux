package lexicon_test

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/lexicon"
)

func TestTokenizeSplitsCamelCaseAndLowercases(t *testing.T) {
	got := lexicon.Tokenize("resolveGoImport")
	want := []string{"resolve", "go", "import"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenizeSplitsOnEveryNonAlphanumeric(t *testing.T) {
	got := lexicon.Tokenize("internal/code/extract/golang_test.go")
	want := []string{"internal", "code", "extract", "golang", "test", "go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenizeDropsSingleCharactersAndStopWords(t *testing.T) {
	// Length must be greater than one, and the stop list is 32 words.
	got := lexicon.Tokenize("how does the a x cache get used")
	want := []string{"cache"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v -- stop words and single characters carry no intent", got, want)
	}
}

func TestTokenizeIsASCIIOnlyByDesign(t *testing.T) {
	// Byte-for-byte Graft's regex: the camel split is [a-z0-9][A-Z], the
	// separator is [^a-z0-9]. unicode.IsUpper would be the nicer Go version and
	// would produce different tokens than the golden values on an identifier
	// with an umlaut.
	got := lexicon.Tokenize("großeZahl")
	want := []string{"gro", "zahl"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenizeHandlesDigitsInsideAName(t *testing.T) {
	got := lexicon.Tokenize("sha256Sum")
	want := []string{"sha256", "sum"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenizeOfEmptyTextIsEmpty(t *testing.T) {
	if got := lexicon.Tokenize(""); len(got) != 0 {
		t.Fatalf("got %v, want nothing", got)
	}
}

func TestCountsSumsRepeats(t *testing.T) {
	got := lexicon.Counts([]string{"cache", "get", "cache"})
	if got["cache"] != 2 || got["get"] != 1 {
		t.Fatalf("got %v, want cache twice", got)
	}
}

func TestStopWordsAreThirtyTwo(t *testing.T) {
	// Counted against index-file.ts:28-33, not taken from a comment. The number
	// is here so a later edit to the list is a deliberate edit.
	if n := lexicon.StopWordCount(); n != 32 {
		t.Fatalf("got %d stop words, want 32", n)
	}
}
