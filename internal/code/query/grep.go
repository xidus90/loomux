package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/model"
)

// GrepAnswer aliases grep.Result.
type GrepAnswer = grep.Result

// GrepOptions configures the grep search.
type GrepOptions struct {
	IgnoreCase bool
	Fixed      bool
	In         string
	MaxHits    int
	NoRefresh  bool
	Keep       func(path string) bool
}

// Grep orchestrates a symbol-coupled regex grep across the codebase.
func Grep(root, pattern string, opts GrepOptions) (grep.Result, []string, error) {
	g, notes, err := loadGraph(root, opts.NoRefresh)
	if err != nil {
		return grep.Result{}, notes, err
	}

	x := blast.New(g)
	spans := model.FileSpans(g)

	reader := func(p string) ([]byte, error) {
		if opts.Keep != nil && !opts.Keep(p) {
			return nil, os.ErrPermission
		}
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	}

	grepOpts := grep.Options{
		IgnoreCase: opts.IgnoreCase,
		Fixed:      opts.Fixed,
		In:         opts.In,
		MaxHits:    opts.MaxHits,
	}

	res, err := grep.Search(g, x, spans, pattern, grepOpts, reader)
	return res, notes, err
}

// GrepReport formats grep.Result into a human-readable CLI report.
func GrepReport(r grep.Result) string {
	if r.TotalHits == 0 {
		return fmt.Sprintf("no matches for %q\n", r.Pattern)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d matches across %d files for %q:\n", r.TotalHits, r.FilesSearched, r.Pattern)
	for _, grp := range r.Groups {
		if grp.Symbol != nil {
			fmt.Fprintf(&b, "\n%s  %s (%s, inDegree=%d):\n", grp.Path, grp.Symbol.Name, grp.Symbol.Kind, grp.InDegree)
		} else {
			fmt.Fprintf(&b, "\n%s (file level):\n", grp.Path)
		}
		for _, h := range grp.Hits {
			fmt.Fprintf(&b, "  L%-4d | %s\n", h.Line, h.Text)
		}
	}
	if r.TruncatedHits > 0 {
		fmt.Fprintf(&b, "\n... and %d more hits truncated\n", r.TruncatedHits)
	}
	return b.String()
}
