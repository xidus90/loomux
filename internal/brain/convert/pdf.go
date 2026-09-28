package convert

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/programs"
)

// scanThreshold is the least a page must hold to count as text: measured
// text pages held 1931 to 2593 characters (pdf.py:15-18). Measured there on
// pypdf's output, and again on Poppler 25.07.0's: scan pages held 0
// characters, text pages at least 142.
const scanThreshold = 100

// pdfTimeout bounds one pdftotext call; the reference had none, and a
// hanging PDF must not hold the whole batch.
const pdfTimeout = 2 * time.Minute

// Tools is the seam to the programs convert and fetch start.
type Tools struct {
	Run  func(child.Spec) child.Result
	Look func(string) (string, error)
}

// SystemTools starts the real programs from the PATH.
func SystemTools() Tools { return Tools{Run: child.Run, Look: exec.LookPath} }

// toolError is a program that is missing or the wrong one: every PDF of the
// run is left for a person, with the command that fixes it.
type toolError struct{ msg string }

func (e *toolError) Error() string { return e.msg }

// extraction is what one PDF yields: the washed text of the pages kept, ""
// where none was, and how many pages there were and were skipped as scans.
type extraction struct {
	text           string
	pages, skipped int
}

// pdftotext is Poppler's program, found and checked once per run.
type pdftotext struct {
	tools Tools
	once  sync.Once
	exe   string
	err   error
}

func newPDFToText(t Tools) *pdftotext { return &pdftotext{tools: t} }

// resolve finds pdftotext and takes it only when `-v` names Poppler: xpdf
// writes another text under the same name, and two builds on one machine
// would rewrite every PDF target from the other shell. The exit code of -v
// decides nothing once it said something: xpdf 4.06 ends it with 99 and
// writes to stdout, Poppler 25.07.0 with 0 to stderr (measured 2026-09-26),
// so xpdf is the wrong program, not a missing one. A -v that did not start,
// ran out of time or said nothing is named as such, so the user learns what
// went wrong with the program they have.
func (p *pdftotext) resolve() (string, error) {
	p.once.Do(func() {
		install := programs.Install("pdftotext")
		exe, err := p.tools.Look("pdftotext")
		if err != nil {
			p.err = &toolError{"pdftotext is not on PATH; install it with: " + install}
			return
		}
		res := p.tools.Run(child.Spec{Argv: []string{exe, "-v"}, Timeout: pdfTimeout})
		said := res.Stdout + res.Stderr
		var why string
		switch {
		case res.Err != nil:
			why = fmt.Sprintf("%s -v did not start: %s", exe, oneLine(res.Err))
		case res.TimedOut:
			why = fmt.Sprintf("%s -v took longer than %s", exe, pdfTimeout)
		case firstLine(said) == "":
			why = fmt.Sprintf("%s -v exited %d and said nothing", exe, res.Code)
		case !strings.Contains(said, "Poppler"):
			why = fmt.Sprintf("%s is not Poppler's pdftotext (%s)", exe, firstLine(said))
		default:
			p.exe = exe
			return
		}
		p.err = &toolError{why + "; install Poppler with: " + install}
	})
	return p.exe, p.err
}

// oneLine is an error on one line: child.Run joins the program's path to a
// start failure with a line break, which would split a skipped: line.
func oneLine(err error) string {
	return strings.ReplaceAll(err.Error(), "\n", ": ")
}

// firstLine is the first line a program said. pdftotext writes -v and its
// errors with CRLF under Windows, xpdf and Poppler alike (measured
// 2026-09-26); child.Run turns that into LF, but the seam promises nothing,
// and the CR must not end up inside a skipped: line.
func firstLine(said string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(said), "\n")
	return strings.TrimSpace(first)
}

// extract is this PDF's text, each page held to the scan threshold on its
// own: three text pages among seven scans keep their three.
func (p *pdftotext) extract(path string) (extraction, error) {
	exe, err := p.resolve()
	if err != nil {
		return extraction{}, err
	}
	res := p.tools.Run(child.Spec{
		Argv:    []string{exe, "-layout", "-enc", "UTF-8", "-eol", "unix", filepath.Base(path), "-"},
		Dir:     filepath.Dir(path),
		Timeout: pdfTimeout,
	})
	switch {
	case res.Err != nil:
		return extraction{}, fmt.Errorf("pdftotext did not start: %s", oneLine(res.Err))
	case res.TimedOut:
		return extraction{}, fmt.Errorf("pdftotext took longer than %s", pdfTimeout)
	case res.Code != 0 && firstLine(res.Stderr) == "":
		return extraction{}, fmt.Errorf("pdftotext exited %d and said nothing", res.Code)
	case res.Code != 0:
		return extraction{}, fmt.Errorf("pdftotext exited %d: %s", res.Code, firstLine(res.Stderr))
	// child.Run already puts U+FFFD in place of a broken byte; this holds
	// any other seam to the same text.
	case !utf8.ValidString(res.Stdout):
		return extraction{}, fmt.Errorf("pdftotext wrote no UTF-8")
	}
	pages := splitPages(res.Stdout)
	var kept []string
	for _, page := range pages {
		if utf8.RuneCountInString(pytext.Strip(page)) >= scanThreshold {
			kept = append(kept, wash(page))
		}
	}
	return extraction{text: strings.Join(kept, "\n\n"), pages: len(pages), skipped: len(pages) - len(kept)}, nil
}

// splitPages cuts pdftotext's output at its form feeds; the one after the
// last page ends no page.
func splitPages(out string) []string {
	if out == "" {
		return nil
	}
	pages := strings.Split(out, "\f")
	if pages[len(pages)-1] == "" {
		pages = pages[:len(pages)-1]
	}
	return pages
}

var spaceRuns = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[ \t]{2,}`) })

// wash pulls the layout's floods of whitespace into one character and
// blank-separated lines into paragraphs (pdf.py:53-66).
func wash(page string) string {
	var paragraphs, current []string
	for _, line := range pytext.SplitLines(page) {
		line = pytext.Strip(spaceRuns().ReplaceAllString(line, " "))
		switch {
		case line != "":
			current = append(current, line)
		case len(current) > 0:
			paragraphs = append(paragraphs, strings.Join(current, " "))
			current = nil
		}
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, strings.Join(current, " "))
	}
	return strings.Join(paragraphs, "\n\n")
}
