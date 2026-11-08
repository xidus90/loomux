package grep

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/corpus"
)

// scanLines is the serial line scan the parallel one has to reproduce: split
// at '\n', drop one trailing '\r', match each line on its own.
func scanLines(re *regexp.Regexp, data []byte) (lines []int, texts []string) {
	if corpus.IsBinary(data) {
		return nil, nil
	}
	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if re.MatchString(line) {
			lines = append(lines, i+1)
			texts = append(texts, line)
		}
	}
	return lines, texts
}

func TestScanEqualsLineScan(t *testing.T) {
	patterns := []struct {
		pattern string
		opts    Options
	}{
		{"func Search", Options{}},
		{"Search|Grep", Options{}},
		{"func search", Options{IgnoreCase: true}},
		{`[A-Z]{3}\w+`, Options{}},
		{"^$", Options{}},
		{`\Afoo`, Options{}},
		{`foo\z`, Options{}},
		{"foo$", Options{}},
		{`(?s)a.b`, Options{}},
		{"[^x]", Options{}},
		{"func Search", Options{Fixed: true}},
	}
	files := map[string]string{
		"lf":         "package x\nfunc Search() {}\nfoo\n\nGrep ABCdef\n",
		"crlf":       "func Search()\r\nfoo\r\n\r\nx\r\nGREP\r\n",
		"no newline": "foo\nfunc search\nfoo",
		"trailing":   "\n\nfoo\na\nb\n",
		"binary":     "func Search\x00\nfoo\n",
		"empty":      "",
		"one x":      "x",
		"two a line": "Search or Grep\nfoo foo\n",
	}
	for _, p := range patterns {
		sc, err := compile(p.pattern, p.opts)
		if err != nil {
			t.Fatalf("%s: %v", p.pattern, err)
		}
		for name, text := range files {
			got := sc.file([]byte(text))
			lines, texts := scanLines(sc.re, []byte(text))
			if !reflect.DeepEqual(got.lines, lines) || !reflect.DeepEqual(got.texts, texts) {
				t.Errorf("%s on %s: got %v %q, want %v %q", p.pattern, name, got.lines, got.texts, lines, texts)
			}
		}
	}
}

func TestScanMarksUnreadableAndBinaryFiles(t *testing.T) {
	sc, err := compile("x", Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := sc.all([]string{"gone", "blob", "text"}, func(p string) ([]byte, error) {
		switch p {
		case "gone":
			return nil, errors.New("gone")
		case "blob":
			return []byte("x\x00"), nil
		}
		return []byte("x\n"), nil
	})
	if !got[0].unreadable || !got[1].binary || got[2].unreadable || got[2].binary || len(got[2].lines) != 1 {
		t.Errorf("scanned = %+v", got)
	}
}

func TestSearchOrderIsStableUnderParallelScan(t *testing.T) {
	const maxHits = 7
	files := make([]string, 200)
	content := make(map[string]string, len(files))
	for i := range files {
		files[i] = fmt.Sprintf("f%03d.go", i)
		content[files[i]] = strings.Repeat("hit\nmiss\n", i%3)
	}
	read := func(p string) ([]byte, error) { return []byte(content[p]), nil }

	// The serial reference: hits in file order, the first maxHits counted.
	var want []Group
	counted, truncated := 0, 0
	for _, p := range files {
		lines, texts := scanLines(regexp.MustCompile("hit"), []byte(content[p]))
		for j := range lines {
			if counted >= maxHits {
				truncated++
				continue
			}
			counted++
			if len(want) == 0 || want[len(want)-1].Path != p {
				want = append(want, Group{Path: p})
			}
			want[len(want)-1].Hits = append(want[len(want)-1].Hits, Hit{Line: lines[j], Text: texts[j]})
		}
	}

	for run := 0; run < 20; run++ {
		res, err := Search(nil, nil, nil, "hit", Options{Files: files, MaxHits: maxHits}, read)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(res.Groups, want) || res.TruncatedHits != truncated || res.TotalHits != maxHits {
			t.Fatalf("run %d: groups %+v truncated %d, want %+v truncated %d", run, res.Groups, res.TruncatedHits, want, truncated)
		}
	}
}
