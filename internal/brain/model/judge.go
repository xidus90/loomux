package model

import (
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// functionWords is the reference's closed list (judge.py:34-40): common
// German function words, every one with an umlaut twice, with it and
// transcribed. It tells German from English; it identifies no language.
var functionWords = []string{
	"der", "die", "das", "den", "dem", "des", "ein", "eine", "einer", "einem", "einen", "und", "oder", "aber", "nicht",
	"ist", "sind", "war", "waren", "wird", "werden", "wurde", "hat", "haben", "in", "im", "an", "auf", "aus", "bei", "mit",
	"von", "zu", "zur", "zum", "fuer", "für", "ueber", "über", "unter", "durch", "gegen", "ohne", "nach", "vor", "seit",
	"sich", "als", "wenn", "weil", "dass", "es", "sie", "er", "wir", "man", "jede", "jeder", "jedes", "kein", "keine", "nur",
	"schon", "noch", "auch", "dann", "damit", "deshalb", "wobei", "welche", "welcher", "mittels",
}

// FunctionWords is a copy of the list, so a caller cannot change the judge.
func FunctionWords() []string { return slices.Clone(functionWords) }

// wordRun is Python's `[\wÄÖÜäöüß]+`: RE2's \w is ASCII, so the letters,
// digits and the underscore of every script are spelled out.
var wordRun = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[\p{L}\p{N}_]+`) })

// hyphenated is `\b[\wÄÖÜäöüß]+(?:-[\wÄÖÜäöüß]+)+\b` without the
// boundaries: a leftmost, greedy run of word characters starts and ends
// where Python's \b stands.
var hyphenated = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`[\p{L}\p{N}_]+(?:-[\p{L}\p{N}_]+)+`)
})

// IsGerman needs two different function words; the umlaut never counts.
// `in`, `die` and `man` occur in English and Dutch sentences too, two
// different ones practically never.
func IsGerman(text string) bool {
	seen := map[string]bool{}
	for _, w := range wordRun().FindAllString(text, -1) {
		seen[pyLower(w)] = true
	}
	hits := 0
	for _, w := range functionWords {
		if seen[w] {
			hits++
		}
	}
	return hits >= 2
}

// pyLower is Python's `str.lower()` as far as it decides membership in the
// function words. Python lowers with the full mapping, strings.ToLower with
// the simple one; of the runes Python 3.14 knows, they part only at U+0130,
// which Python turns into `i` and a combining dot and Go into a bare `i`, so
// `İN` would read as `in`. Python's final sigma stays unmapped: no function
// word holds a sigma.
func pyLower(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "\u0130", "i\u0307"))
}

// ChoppedWords are hyphenated words a model assembled from fragments, by two
// signals (judge.py:69-117): the parts push together into a common word
// while no part but the first is capitalised, or a capitalised word holds a
// short part rarer than a word part would be.
func ChoppedWords(text string) []string {
	var hits []string
	for _, w := range hyphenated().FindAllString(text, -1) {
		parts := strings.Split(w, "-")
		onlyFirstCapital := !slices.ContainsFunc(parts[1:], startsUpper)
		cutApart := onlyFirstCapital && commonEnough(strings.Join(parts, ""))
		fragmented := startsUpper(w) && slices.ContainsFunc(parts, func(part string) bool {
			return utf8.RuneCountInString(part) <= partMax && fragment(part)
		})
		if cutApart || fragmented {
			hits = append(hits, w)
		}
	}
	return hits
}

// startsUpper is Python's `s[:1].isupper()`, which reads the property
// Uppercase: Lu and Other_Uppercase. unicode.IsUpper reads Lu alone and
// misses the Roman numerals U+2160 to U+216F, which a part can hold.
func startsUpper(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.In(r, unicode.Upper, unicode.Other_Uppercase)
}

// IsOneSentence is exactly one sentence end, standing at the very end.
func IsOneSentence(text string) bool {
	tightened := pytext.Strip(text)
	if tightened == "" {
		return false
	}
	return strings.Count(tightened, ".")+strings.Count(tightened, "!")+strings.Count(tightened, "?") == 1 &&
		strings.ContainsAny(tightened[len(tightened)-1:], ".!?")
}

// WordCount is the number of word runs.
func WordCount(text string) int { return len(wordRun().FindAllString(text, -1)) }

// readsBack says whether the sentence comes back unchanged out of a file
// head, read by yaml.v3 as loomux reads every head (`index.ParseFrontmatter`).
// The reference asks PyYAML. On every sentence of the battery the two agree;
// they part on bare scalars YAML 1.1 resolves otherwise (`yes`, `1:20`,
// `0o17`), which never pass IsGerman and so never reach this judge. A tab is
// refused first: PyYAML refuses it anywhere in a plain scalar, yaml.v3 keeps
// one inside the sentence, and apply's PyYAML port would refuse the head.
func readsBack(sentence string) bool {
	if strings.Contains(sentence, "\t") {
		return false
	}
	var loaded map[string]any
	if err := yaml.Unmarshal([]byte("description: "+sentence+"\n"), &loaded); err != nil {
		return false
	}
	value, ok := loaded["description"].(string)
	return ok && len(loaded) == 1 && value == sentence
}
