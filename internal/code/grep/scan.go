package grep

import (
	"bytes"
	"fmt"
	"regexp"
	"regexp/syntax"
	"runtime"
	"slices"
	"sync"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/trigram"
)

// scanner is one pattern compiled for the file scan.
type scanner struct {
	// re matches one line, with '\n' and a trailing '\r' cut off.
	re *regexp.Regexp
	// whole is the (?m) form of re for a single pass over a file; nil when a
	// match could cross a line or anchor the text instead of the line.
	whole *regexp.Regexp
	// lits is the pattern's literal condition from trigram.Literals,
	// ASCII-lowercased: some conjunction has to be in a matching line. nil
	// when the pattern has none.
	lits [][][]byte
	// fold says the pattern folds case somewhere: an ASCII literal then also
	// matches non-ASCII bytes (the Kelvin sign for k), which the lowered copy
	// cannot see.
	fold bool
	// max is how many matching lines of one file keep their text: no more
	// than that can be counted from one file, so the rest only add to more.
	max int
}

// scanned is what the scan found in one file.
type scanned struct {
	unreadable, binary bool
	lines              []int
	texts              []string
	// more counts the matching lines past the scanner's max.
	more int
}

// compile turns a grep pattern into its scanner, refusing what RE2 cannot do.
func compile(pattern string, opts Options) (*scanner, error) {
	p := pattern
	if opts.Fixed {
		p = regexp.QuoteMeta(pattern)
	} else if form := unsupported(pattern); form != "" {
		return nil, fmt.Errorf("%s not supported in RE2 regex: %s", form, pattern)
	}
	if opts.IgnoreCase {
		p = "(?i)" + p
	}
	// Without OneLine, ^ and $ parse as line anchors, as in the (?m) form;
	// the error is the one regexp.Compile would return for p.
	tree, err := syntax.Parse(p, syntax.Perl&^syntax.OneLine)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}
	sc := &scanner{re: regexp.MustCompile(p), fold: folds(tree), max: opts.MaxHits}
	if sc.max <= 0 {
		sc.max = DefaultMaxHits
	}
	if noNewline(tree) {
		sc.whole = regexp.MustCompile("(?m)" + p)
	}
	// The pattern compiled, so Literals cannot refuse it.
	sets, ok, _ := trigram.Literals(pattern, opts.Fixed, opts.IgnoreCase)
	if ok {
		for _, set := range sets {
			var lits [][]byte
			for _, lit := range set {
				lits = append(lits, asciiLower([]byte(lit)))
			}
			sc.lits = append(sc.lits, lits)
		}
	}
	return sc, nil
}

// folds reports whether any node of re matches case-insensitively.
func folds(re *syntax.Regexp) bool {
	if re.Flags&syntax.FoldCase != 0 {
		return true
	}
	for _, sub := range re.Sub {
		if folds(sub) {
			return true
		}
	}
	return false
}

// noNewline reports whether re can never match a '\n' and holds no text
// anchor: then a line matches exactly when a (?m) match over the whole file
// starts in it. \A and \z anchor each line in the line scan, the file here.
func noNewline(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpAnyChar, syntax.OpBeginText, syntax.OpEndText:
		return false
	case syntax.OpCharClass:
		for i := 0; i+1 < len(re.Rune); i += 2 {
			if re.Rune[i] <= '\n' && '\n' <= re.Rune[i+1] {
				return false
			}
		}
	case syntax.OpLiteral:
		return !slices.Contains(re.Rune, '\n')
	}
	for _, sub := range re.Sub {
		if !noNewline(sub) {
			return false
		}
	}
	return true
}

// all scans files on one worker per CPU. Each result lands at its file's
// index, so the caller merges them in file order whatever finished first.
func (s *scanner) all(files []string, read FileReader) []scanned {
	out := make([]scanned, len(files))
	next := make(chan int)
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				data, err := read(files[i])
				if err != nil {
					out[i] = scanned{unreadable: true}
					continue
				}
				out[i] = s.file(data)
			}
		}()
	}
	for i := range files {
		next <- i
	}
	close(next)
	wg.Wait()
	return out
}

// file finds the matching lines of one file's content.
func (s *scanner) file(data []byte) scanned {
	if corpus.IsBinary(data) {
		// A binary file is not in the index and not in the corpus; the scan
		// skips it the same way so both paths agree.
		return scanned{binary: true}
	}
	var lower []byte
	if s.lits != nil && !(s.fold && hasNonASCII(data)) {
		lower = asciiLower(data)
		if !s.holds(lower) {
			return scanned{}
		}
	}
	// A '\r' before '\n' would hide a (?m) $ match; such a file goes by line.
	if s.whole != nil && bytes.IndexByte(data, '\r') < 0 {
		return s.oneRun(data)
	}
	var out scanned
	start := 0
	for n := 1; start <= len(data); n++ {
		end := bytes.IndexByte(data[start:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += start
		}
		if lower == nil || s.holds(lower[start:end]) {
			line := bytes.TrimSuffix(data[start:end], []byte{'\r'})
			if s.re.Match(line) {
				s.keep(&out, n, line)
			}
		}
		start = end + 1
	}
	return out
}

// oneRun maps the matches of one pass of s.whole over data to their lines.
func (s *scanner) oneRun(data []byte) scanned {
	var out scanned
	line, pos, last := 1, 0, 0
	for _, loc := range s.whole.FindAllIndex(data, -1) {
		line += bytes.Count(data[pos:loc[0]], []byte{'\n'})
		pos = loc[0]
		if line == last {
			continue
		}
		last = line
		start := bytes.LastIndexByte(data[:loc[0]], '\n') + 1
		end := bytes.IndexByte(data[loc[0]:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += loc[0]
		}
		s.keep(&out, line, data[start:end])
	}
	return out
}

// keep adds one matching line to out, its text only while out holds fewer
// than s.max: the copy is what a broad pattern would otherwise pile up.
func (s *scanner) keep(out *scanned, line int, text []byte) {
	if len(out.lines) >= s.max {
		out.more++
		return
	}
	out.lines = append(out.lines, line)
	out.texts = append(out.texts, string(text))
}

// holds reports whether some conjunction of s.lits is entirely in b.
func (s *scanner) holds(b []byte) bool {
	for _, set := range s.lits {
		all := true
		for _, lit := range set {
			if !bytes.Contains(b, lit) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

func asciiLower(b []byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return out
}

func hasNonASCII(b []byte) bool {
	for _, c := range b {
		if c >= 0x80 {
			return true
		}
	}
	return false
}
