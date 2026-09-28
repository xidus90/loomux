package model

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// zipfData is generate.py's table, packed. It is unpacked on first use, never
// at start: the hook path must not pay for a judge it never calls.
//
//go:embed zipf/de.txt.gz
var zipfData []byte

//go:embed zipf/NOTICE.md
var zipfNotice string

// ZipfNotice is the license notice of the embedded table; NOTICE.md, which
// the release ships beside the binaries, quotes it.
func ZipfNotice() string { return zipfNotice }

// partMax is the longest part the fragment signal looks at (judge.py:56).
const partMax = 5

type zipfTable struct {
	common map[string]bool    // Zipf >= 3.0, no digit run
	mid    map[string]bool    // 2.5 <= Zipf < 3.0, no digit run
	digits map[string]float64 // keys with a digit run, raw frequency
}

var loadedZipf = sync.OnceValue(func() *zipfTable {
	return mustZipf(zipfData)
})

// mustZipf unpacks and reads the embedded table.
func mustZipf(data []byte) *zipfTable {
	text, err := unpackZipf(data)
	if err != nil {
		panic(err)
	}
	table, err := parseZipf(text)
	if err != nil {
		panic(err)
	}
	return table
}

func unpackZipf(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func parseZipf(text []byte) (*zipfTable, error) {
	table := &zipfTable{common: map[string]bool{}, mid: map[string]bool{}, digits: map[string]float64{}}
	section := ""
	lines := bufio.NewScanner(bytes.NewReader(text))
	for lines.Scan() {
		line := lines.Text()
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case line == "[common]" || line == "[mid]" || line == "[digits]":
			section = line
		case section == "[common]":
			table.common[line] = true
		case section == "[mid]":
			table.mid[line] = true
		case section == "[digits]":
			key, raw, ok := strings.Cut(line, "\t")
			freq, err := strconv.ParseFloat(raw, 64)
			if !ok || err != nil {
				return nil, fmt.Errorf("zipf table: bad digits line %q", line)
			}
			table.digits[key] = freq
		default:
			return nil, fmt.Errorf("zipf table: %q stands outside a section", line)
		}
	}
	// A line past the scanner's limit ends the loop like the end of the
	// text; without this check the table would be cut short silently.
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("zipf table: %w", err)
	}
	return table, nil
}

// band is what the table knows about a word: 2 at Zipf 3.0 or more, 1 at 2.5
// or more, 0 below. A word without a token is 0, like wordfreq's minimum; a
// word of several tokens takes the lowest band among them, because wordfreq
// combines them as 1/f = 1/f1 + 1/f2 + ..., which lies below every one of
// them (parity list: several tokens).
func band(word string) int {
	tokens := zipfTokens(word)
	if len(tokens) == 0 {
		return 0
	}
	table := loadedZipf()
	lowest := 2
	for _, token := range tokens {
		lowest = min(lowest, tokenBand(table, token))
	}
	return lowest
}

// zipfTokens is wordfreq's tokenize for German as far as a part of the judge
// can reach it: casefold(NFC(word)), cut at every rune outside the regex
// module's \w. That drops superscript, subscript and fraction digits, which
// Python's re keeps in a part (`CO₂` is the token `co`). The parity list
// holds where this parts from wordfreq: Alphabetic taken as L, Nl and M,
// punctuation between digits, and the scripts written without spaces.
func zipfTokens(word string) []string {
	return strings.FieldsFunc(pytext.CaseFold(pytext.NFC(word)), func(r rune) bool {
		return !wordRune(r)
	})
}

// wordRune is the regex module's \w: Alphabetic, Mark, Decimal_Number,
// Connector_Punctuation and Join_Control, with Alphabetic taken as L and Nl.
func wordRune(r rune) bool {
	return unicode.In(r, unicode.L, unicode.Nl, unicode.M, unicode.Nd, unicode.Pc) || r == 0x200c || r == 0x200d
}

// tokenBand is what the table knows about one token.
func tokenBand(table *zipfTable, token string) int {
	smashed := smashNumbers(token)
	if raw, ok := table.digits[smashed]; ok {
		freq := raw
		if smashed != token {
			freq *= digitFreq(token)
		}
		// _word_frequency sums 1/f over the tokens and inverts the sum; for
		// one token that round trip still moves some values by the last bit,
		// and the rounding after it can see the difference.
		freq = 1 / (1 / freq)
		return bandOf(zipfOf(freq))
	}
	switch {
	case smashed != token:
		return 0
	case table.common[token]:
		return 2
	case table.mid[token]:
		return 1
	}
	return 0
}

func bandOf(zipf float64) int {
	switch {
	case zipf >= 3.0:
		return 2
	case zipf >= 2.5:
		return 1
	}
	return 0
}

// commonEnough says whether a word pushed together is a real word
// (judge.py:45-48).
func commonEnough(word string) bool { return band(word) == 2 }

// fragment says whether a short part is rarer than a word part would be
// (judge.py:49-51). The judge asks it for parts of at most partMax
// characters, counted as written.
func fragment(part string) bool { return band(part) == 0 }

// log10 is Python's math.log(x, 10), which divides two natural logarithms
// and can differ from math.Log10 in the last bit.
func log10(x float64) float64 { return math.Log(x) / math.Log(10) }

// zipfOf is word_frequency's rounding and freq_to_zipf's, as
// zipf_frequency(word, lang, min_zipf=0) calls them.
func zipfOf(freq float64) float64 {
	freq = math.Max(freq, 1e-9)
	lead := math.Floor(-log10(freq))
	freq = pyRound(freq, int(lead)+3)
	return pyRound(log10(freq)+9, 2)
}

// pyRound is Python's round(x, n) for n >= 0: the exact binary value rounded
// to n decimals, an exact half to even.
func pyRound(x float64, n int) float64 {
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	return rounded
}

// The constants of wordfreq/numbers.py.
var digitFreqs = [10]float64{0.009, 0.300, 0.175, 0.124, 0.096, 0.078, 0.066, 0.057, 0.050, 0.045}

const (
	yearLogPeak   = -1.9185
	notYearProb   = 0.1
	referenceYear = 2019
	plateauWidth  = 20
)

func pow10(x float64) float64 { return math.Pow(10, x) }

// Digits are ASCII here, where wordfreq's `\d` takes every script's (parity
// list: digits only ASCII). The period and comma stay in the pattern as
// wordfreq has them, though zipfTokens never leaves one in a token.
var multiDigit = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[0-9][0-9.,]+`) })
var pureDigit = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[0-9]+`) })

// smashNumbers is wordfreq's: every digit of a run of two or more
// characters becomes 0.
func smashNumbers(text string) string {
	return multiDigit().ReplaceAllStringFunc(text, func(run string) string {
		return strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return '0'
			}
			return r
		}, run)
	})
}

// digitFreq is wordfreq's digit_freq.
func digitFreq(text string) float64 {
	freq := 1.0
	for _, run := range multiDigit().FindAllString(text, -1) {
		for _, digits := range pureDigit().FindAllString(run, -1) {
			if len(digits) == 4 {
				freq *= yearFreq(digits)
			} else {
				freq *= benfordFreq(digits)
			}
		}
	}
	return freq
}

func benfordFreq(digits string) float64 {
	return digitFreqs[digits[0]-'0'] / pow10(float64(len(digits)-1))
}

// yearFreq is wordfreq's year_freq. Each product is converted to float64 on
// its own: Go may fuse a multiply and an add into one rounding, CPython never
// does, and the conversion forbids the fusion.
func yearFreq(digits string) float64 {
	year, _ := strconv.Atoi(digits)
	var logFreq float64
	switch {
	case year <= referenceYear:
		logFreq = yearLogPeak - float64(0.0083*float64(referenceYear-year))
	case year <= referenceYear+plateauWidth:
		logFreq = yearLogPeak
	default:
		logFreq = yearLogPeak - float64(0.2*float64(year-(referenceYear+plateauWidth)))
	}
	return pow10(logFreq) + float64(notYearProb*benfordFreq(digits))
}
