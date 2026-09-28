package model

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"os"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// zipfSum is the sha256 of the unpacked table generate.py wrote; a changed
// table is a changed judge, and the battery below would not know.
const zipfSum = "c63dd53b5aa754e0570a4a056666bdab739b6e4212823b104a869b728e9e862d"

func TestTheZipfTableIsTheGeneratedOne(t *testing.T) {
	reader, err := gzip.NewReader(bytes.NewReader(zipfData))
	if err != nil {
		t.Fatal(err)
	}
	text, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(text)
	if hex.EncodeToString(sum[:]) != zipfSum {
		t.Fatal("de.txt.gz is not the table generate.py writes")
	}
}

type battery struct {
	Words []struct {
		Word   string   `json:"word"`
		Zipf   float64  `json:"zipf"`
		Tokens []string `json:"tokens"`
	} `json:"words"`
	DigitFreq map[string]float64 `json:"digit_freq"`
}

func readBattery(t *testing.T) battery {
	t.Helper()
	data, err := os.ReadFile("testdata/zipf-battery.json")
	if err != nil {
		t.Fatal(err)
	}
	var b battery
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	return b
}

// TestTheTableAnswersWhatWordfreqAnswers holds the band and both questions the
// judge asks against wordfreq's own number for every word of the battery,
// however many tokens wordfreq reads in it. The thresholds are written out
// here rather than taken from bandOf, so that a wrong threshold there cannot
// hide.
func TestTheTableAnswersWhatWordfreqAnswers(t *testing.T) {
	seen := map[string]bool{}
	for _, row := range readBattery(t).Words {
		seen[row.Word] = true
		want := 0
		switch {
		case row.Zipf >= 3.0:
			want = 2
		case row.Zipf >= 2.5:
			want = 1
		}
		if _, excepted := bandExceptions[row.Word]; excepted {
			if band(row.Word) == want {
				t.Errorf("band(%q) agrees with wordfreq; drop it from bandExceptions", row.Word)
			}
			continue
		}
		if got := band(row.Word); got != want {
			t.Errorf("band(%q) = %d, wordfreq says %.2f (tokens %q)", row.Word, got, row.Zipf, row.Tokens)
		}
		if got, want := commonEnough(row.Word), row.Zipf >= 3.0; got != want {
			t.Errorf("commonEnough(%q) = %v, wordfreq says %.2f", row.Word, got, row.Zipf)
		}
		if utf8.RuneCountInString(row.Word) <= partMax {
			if got, want := fragment(row.Word), row.Zipf < 2.5; got != want {
				t.Errorf("fragment(%q) = %v, wordfreq says %.2f", row.Word, got, row.Zipf)
			}
		}
	}
	for word := range bandExceptions {
		if !seen[word] {
			t.Errorf("bandExceptions names %q, which the battery does not hold", word)
		}
	}
}

// TestTheSplitIsWordfreqsTokenizer holds zipfTokens against the tokens
// wordfreq's tokenize recorded for every word of the battery.
func TestTheSplitIsWordfreqsTokenizer(t *testing.T) {
	seen := map[string]bool{}
	for _, row := range readBattery(t).Words {
		seen[row.Word] = true
		got, want := zipfTokens(row.Word), row.Tokens
		if len(want) == 0 {
			want = nil
		}
		same := slices.Equal(got, want)
		if _, excepted := splitExceptions[row.Word]; excepted {
			if same {
				t.Errorf("zipfTokens(%q) agrees with wordfreq; drop it from splitExceptions", row.Word)
			}
			continue
		}
		if !same {
			t.Errorf("zipfTokens(%q) = %q, wordfreq reads %q", row.Word, got, want)
		}
	}
	for word := range splitExceptions {
		if !seen[word] {
			t.Errorf("splitExceptions names %q, which the battery does not hold", word)
		}
	}
}

// splitExceptions are the battery's words zipfTokens splits differently from
// wordfreq, each with the reason; the rows stand in the parity list.
var splitExceptions = map[string]string{
	"3,5":   "wordfreq keeps a comma between digits in one token (UAX #29, WB11/12); the judge's part regex never yields a comma",
	"1.000": "wordfreq keeps a period between digits in one token (UAX #29, WB11/12); the judge's part regex never yields a period",
	"0,0,0": "wordfreq keeps a comma between digits in one token (UAX #29, WB11/12); the judge's part regex never yields a comma",
}

// bandExceptions are the battery's words whose band may differ from
// wordfreq's, each with the reason. The judge's parts are runs of Python's
// re \w, so none of them reaches the judge; the rows stand in the parity list.
var bandExceptions = map[string]string{
	"1.000": "wordfreq keeps a period or comma between digits in one token and multiplies by digit_freq; band splits there, and the judge's part regex never yields a period",
}

// TestDigitFreqIsWordfreqs holds the Benford and year estimate against
// wordfreq's own values. math.Pow and C's pow may part in the last bit, so
// the comparison allows one part in 1e15.
func TestDigitFreqIsWordfreqs(t *testing.T) {
	for run, want := range readBattery(t).DigitFreq {
		if got := digitFreq(run); math.Abs(got-want) > want*1e-15 {
			t.Errorf("digitFreq(%q) = %v, wordfreq says %v", run, got, want)
		}
	}
}

func TestZipfNoticeNamesTheLicenseAndTheSource(t *testing.T) {
	notice := ZipfNotice()
	for _, want := range []string{"CC BY-SA 4.0", "Copyright 2022 Robyn Speer", "wordfreq", "Google Books Ngrams"} {
		if !strings.Contains(notice, want) {
			t.Errorf("the notice does not name %q", want)
		}
	}
}

func TestParseZipfRefusesWhatGenerateNeverWrites(t *testing.T) {
	for _, text := range []string{
		"word before any section\n",
		"[digits]\nno-tab\n",
		"[digits]\n00\tnot-a-number\n",
		"[unknown]\nx\n",
		// longer than bufio.Scanner's 64 KiB: the scanner stops with an error
		"[common]\n" + strings.Repeat("x", 70000) + "\n",
	} {
		if _, err := parseZipf([]byte(text)); err == nil {
			t.Errorf("parseZipf(%q) took it", text)
		}
	}
}

func TestMustZipfPanicsOnABrokenTable(t *testing.T) {
	var packed bytes.Buffer
	writer := gzip.NewWriter(&packed)
	if _, err := writer.Write([]byte("x\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"no gzip": []byte("plain"), "word outside a section": packed.Bytes()} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("mustZipf took a broken table")
				}
			}()
			mustZipf(data)
		})
	}
}

func TestUnpackRefusesWhatIsNoGzip(t *testing.T) {
	if _, err := unpackZipf([]byte("plain")); err == nil {
		t.Fatal("plain bytes unpacked")
	}
}

func TestPyRoundRoundsLikePython(t *testing.T) {
	for _, c := range []struct {
		x    float64
		n    int
		want float64
	}{
		{2.675, 2, 2.67}, // the binary value lies below the half
		{0.125, 2, 0.12}, // an exact half goes to even
		{0.375, 2, 0.38},
		{1.23456e-7, 9, 1.23e-7},
	} {
		if got := pyRound(c.x, c.n); got != c.want {
			t.Errorf("pyRound(%v, %d) = %v, want %v", c.x, c.n, got, c.want)
		}
	}
}

func TestSmashNumbersZeroesRunsAndLeavesSingleDigits(t *testing.T) {
	for in, want := range map[string]string{"2026": "0000", "g4": "g4", "3,5": "0,0", "x86": "x00", "a1b": "a1b"} {
		if got := smashNumbers(in); got != want {
			t.Errorf("smashNumbers(%q) = %q, want %q", in, got, want)
		}
	}
}
