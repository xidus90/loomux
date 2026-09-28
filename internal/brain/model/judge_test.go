package model

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/brain/index"
)

const glm = "Die Scheiben erfordern durch Prüfkriterien die Ferti-Stellung um das Pro-jekt fort-zu-set-zen"

func TestGermanIsMeasuredAtFunctionWordsNeverAtUmlauts(t *testing.T) {
	for _, text := range []string{
		"Der Bau ist in acht Scheiben zerlegt.",
		"Die Scheiben definieren Fertigkriterien für verschiedene Bauabschnitte.",
		"Die Scheiben definieren Fertigkriterien fuer verschiedene Bauabschnitte.",
		glm,
	} {
		if !IsGerman(text) {
			t.Errorf("IsGerman(%q) = false", text)
		}
	}
	for _, text := range []string{"The build is split into eight slices.", "De bouw is in acht plakken verdeeld.", "Data in tables"} {
		if IsGerman(text) {
			t.Errorf("IsGerman(%q) = true", text)
		}
	}
}

// Python lowers the dotted capital I to `i` and a combining dot, Go to a bare
// `i`; with Go's reading `İN` would be the function word `in`.
func TestADottedCapitalINeverLowersIntoAFunctionWord(t *testing.T) {
	if IsGerman("İN DER Stadt.") {
		t.Error("İN counted as the function word in")
	}
	if !IsGerman("IN DER Stadt.") {
		t.Error("IN no longer counts as the function word in")
	}
}

func TestAChoppedWordIsFound(t *testing.T) {
	for _, word := range []string{"Pro-jekt", "fort-zu-set-zen", "Ferti-Stellung"} {
		if got := ChoppedWords("Ein Satz mit " + word + " darin."); !slices.Equal(got, []string{word}) {
			t.Errorf("ChoppedWords(%q) = %q", word, got)
		}
	}
	if got := ChoppedWords(glm); !slices.Equal(got, []string{"Ferti-Stellung", "Pro-jekt", "fort-zu-set-zen"}) {
		t.Errorf("the glm sentence: %q", got)
	}
}

func TestAnOrdinaryHyphenatedWordIsNotChopped(t *testing.T) {
	for _, word := range []string{"E-Mail", "qmd-Profil", "Python-nach-Go-Migration", "Know-how", "Fertig-Kriterium", "Ende-Zustand", "Ende-zu-Ende-Verschlüsselung"} {
		if got := ChoppedWords("Ein Satz mit " + word + " darin."); len(got) != 0 {
			t.Errorf("ChoppedWords(%q) = %q", word, got)
		}
	}
	for _, word := range []string{"Teilsystem-Scheiben", "Wiki-Suchkette"} {
		if got := ChoppedWords("Das " + word + " steht bereit."); len(got) != 0 {
			t.Errorf("a rare compound %q counted as chopped: %q", word, got)
		}
	}
}

// Python's isupper() takes Other_Uppercase as well as Lu; of the runes a part
// can hold, that adds the Roman numerals U+2160 to U+216F.
func TestARomanNumeralStartsUpperAsInPython(t *testing.T) {
	for _, s := range []string{"Ⅻ", "Ⅰx", "Äpfel", "E"} {
		if !startsUpper(s) {
			t.Errorf("startsUpper(%q) = false", s)
		}
	}
	for _, s := range []string{"ⅻ", "ǅ", "e", "1", "_", ""} {
		if startsUpper(s) {
			t.Errorf("startsUpper(%q) = true", s)
		}
	}
}

func TestOneSentenceEndsOnceAndAtTheEnd(t *testing.T) {
	for text, want := range map[string]bool{
		"Ein Satz.": true, "Ein Satz. Noch einer.": false, "Ein Satz. Und Text danach": false, "Kein Ende": false, "   ": false,
	} {
		if got := IsOneSentence(text); got != want {
			t.Errorf("IsOneSentence(%q) = %v", text, got)
		}
	}
}

func TestWordCountCountsAWordWithUmlautsAsOne(t *testing.T) {
	if n := WordCount("Für größere Bereiche gilt das auch."); n != 6 {
		t.Errorf("got %d", n)
	}
	if n := WordCount("Der Bau ist in acht Scheiben zerlegt."); n != 7 {
		t.Errorf("got %d", n)
	}
}

func TestTheFunctionWordsAreTheReferencesSeventyTwo(t *testing.T) {
	words := FunctionWords()
	if len(words) != 72 || !slices.Contains(words, "für") || !slices.Contains(words, "fuer") || !slices.Contains(words, "mittels") {
		t.Fatalf("%d words: %q", len(words), words)
	}
	words[0] = "changed"
	if FunctionWords()[0] == "changed" {
		t.Fatal("FunctionWords hands out its own list")
	}
}

// PyYAML separates tokens with spaces only and refuses a tab anywhere in a
// plain scalar; yaml.v3 keeps one inside the sentence as text. A head with
// it would pass here and be refused by apply's PyYAML port.
func TestASentenceWithATabNeverReadsBack(t *testing.T) {
	for _, sentence := range []string{
		"Der Tabulator\tist im Satz.", "Der Tabulator\t ist im Satz.", "\tDer Tabulator ist im Satz.", "Der Tabulator ist im Satz.\t",
	} {
		if readsBack(sentence) {
			t.Errorf("readsBack(%q) = true", sentence)
		}
	}
}

// judgeRow is one sentence of the battery with the reference judges' verdicts.
type judgeRow struct {
	Text        string   `json:"text"`
	IsGerman    bool     `json:"is_german"`
	Chopped     []string `json:"chopped"`
	OneSentence bool     `json:"one_sentence"`
	WordCount   int      `json:"word_count"`
	ReadsBack   bool     `json:"reads_back"`
}

func readJudgeBattery(t *testing.T) []judgeRow {
	t.Helper()
	data, err := os.ReadFile("testdata/judge-battery.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []judgeRow
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestTheJudgesAgreeWithTheReference(t *testing.T) {
	for _, row := range readJudgeBattery(t) {
		if got := IsGerman(row.Text); got != row.IsGerman {
			t.Errorf("IsGerman(%q) = %v", row.Text, got)
		}
		if got := ChoppedWords(row.Text); !slices.Equal(got, row.Chopped) {
			t.Errorf("ChoppedWords(%q) = %q, reference %q", row.Text, got, row.Chopped)
		}
		if got := IsOneSentence(row.Text); got != row.OneSentence {
			t.Errorf("IsOneSentence(%q) = %v", row.Text, got)
		}
		if got := WordCount(row.Text); got != row.WordCount {
			t.Errorf("WordCount(%q) = %d, reference %d", row.Text, got, row.WordCount)
		}
		if got := readsBack(row.Text); got != row.ReadsBack {
			t.Errorf("readsBack(%q) = %v, PyYAML %v", row.Text, got, row.ReadsBack)
		}
	}
}

// comesBackOutOfAHead says whether text, written as the description of a
// head, comes back unchanged out of index.ParseFrontmatter, the reader loomux
// reads every head with.
func comesBackOutOfAHead(t *testing.T, text string) {
	t.Helper()
	meta, err := index.ParseFrontmatter("---\ndescription: "+text+"\n---\n", true)
	if err != nil {
		t.Errorf("the head with %q: %v", text, err)
		return
	}
	if got := meta["description"]; got != text || len(meta) != 1 {
		t.Errorf("the head with %q reads back as %q", text, meta)
	}
}

// TestEverySentenceTheJudgeReadsBackComesBackOutOfAHead holds the premise of
// readsBack: it judges a head the way loomux's head reader reads one. Every
// sentence of the battery that passes it comes back out of a real head.
func TestEverySentenceTheJudgeReadsBackComesBackOutOfAHead(t *testing.T) {
	for _, row := range readJudgeBattery(t) {
		if row.ReadsBack {
			comesBackOutOfAHead(t, row.Text)
		}
	}
}

// TestABareScalarPartsFromPyYAMLOnlyWhereIsGermanRefuses holds examples of
// the scalars YAML 1.1 (PyYAML) and yaml.v3 resolve differently, with
// PyYAML's verdict: e.g. bools, sexagesimal numbers, `=` and `<<` are
// strings to yaml.v3, and `0o17`, `1e3` and `-.5` numbers. The list is not
// complete: `2001-12-14 21:59:43.10 -5` (a PyYAML timestamp) and `1.5e3` (a
// yaml.v3 float) deviate as well. Each such pattern matches a whole scalar
// without two separate words, so IsGerman refuses it and describe never
// reaches readsBack; the rows stand in the parity list.
func TestABareScalarPartsFromPyYAMLOnlyWhereIsGermanRefuses(t *testing.T) {
	for text, pyyaml := range map[string]bool{
		"yes": false, "on": false, "no": false, "off": false, "Yes": false, "NO": false,
		"1:20": false, "1:20.": false, "=": false, "<<": false,
		"0o17": true, "1e3": true, "-.5": true,
	} {
		got := readsBack(text)
		if got == pyyaml {
			t.Errorf("readsBack(%q) = %v as PyYAML; drop it from the list and the parity list", text, got)
		}
		if IsGerman(text) {
			t.Errorf("IsGerman(%q) = true; the deviation reaches describe", text)
		}
		if got {
			comesBackOutOfAHead(t, text)
		}
	}
}
