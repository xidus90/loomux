package query

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/diff"
	"github.com/xidus90/loomux/internal/code/model"
)

// BlastOptions picks the change and how far its radius reaches.
type BlastOptions struct {
	Base      string
	Cached    bool
	Depth     blast.Depth // <= 0 and not All means 1
	NoRefresh bool
	Keep      func(path string) bool
}

// BlastAnswer is the blast radius of the change git names in Range.
type BlastAnswer struct {
	Range string `json:"range"`
	blast.Report
	Hidden int `json:"hidden,omitempty"`
}

// ErrBaseAndCached is a request for two different changes at once.
var ErrBaseAndCached = errors.New("--base and --cached exclude each other")

// ErrBadBase is a base git would read as an option instead of a revision.
var ErrBadBase = errors.New("--base must name a revision")

// ErrNoHistory is a change git has nothing to compare with: no HEAD, or a
// clean tree whose HEAD has no parent to fall back to.
var ErrNoHistory = errors.New("nothing to compare")

// diffPrefixes pins the header paths the parser reads, whatever
// diff.noprefix or diff.mnemonicPrefix the user configured.
var diffPrefixes = []string{"--src-prefix=a/", "--dst-prefix=b/"}

// Blast is the blast radius of a change as git sees it: the index against
// HEAD with Cached, Base...HEAD with Base, otherwise the working tree against
// HEAD and, when that is clean, the last commit.
func Blast(root string, opts BlastOptions) (BlastAnswer, []string, error) {
	ans, _, notes, err := blastWith(root, opts)
	return ans, notes, err
}

// blastWith is Blast that also hands back the graph it read, so a caller
// that needs the graph again does not read it a second time.
func blastWith(root string, opts BlastOptions) (BlastAnswer, *model.Graph, []string, error) {
	if opts.Base != "" && opts.Cached {
		return BlastAnswer{}, nil, nil, ErrBaseAndCached
	}
	// A base git would parse as an option could write files (--output).
	if strings.HasPrefix(opts.Base, "-") {
		return BlastAnswer{}, nil, nil, fmt.Errorf("%w, got %q", ErrBadBase, opts.Base)
	}
	g, notes, err := loadGraph(root, opts.NoRefresh)
	if err != nil {
		return BlastAnswer{}, nil, notes, err
	}
	var flags []string
	rng, rev := "working tree against HEAD", "HEAD"
	switch {
	case opts.Cached:
		flags, rng = []string{"--cached"}, "index against HEAD"
	case opts.Base != "":
		rng = opts.Base + "...HEAD"
		rev = rng
	}
	files, err := changedFiles(root, flags, rev)
	// Asked only once git failed: a missing HEAD is the one failure with a
	// message of its own, and the probe costs a git call. A probe that fails
	// itself says why better than the diff: outside a repository git diff
	// answers with its usage.
	if err != nil {
		switch ok, perr := hasRevision(root, "HEAD"); {
		case perr != nil:
			err = perr
		case !ok:
			err = fmt.Errorf("%w: no HEAD to compare with", ErrNoHistory)
		}
	}
	if err == nil && len(files) == 0 && opts.Base == "" && !opts.Cached {
		ok, perr := hasRevision(root, "HEAD~1")
		if perr != nil {
			return BlastAnswer{}, nil, notes, perr
		}
		if !ok {
			return BlastAnswer{}, nil, notes, fmt.Errorf("%w: HEAD has no parent commit", ErrNoHistory)
		}
		rng, rev = "HEAD~1...HEAD", "HEAD~1...HEAD"
		files, err = changedFiles(root, nil, rev)
	}
	if err != nil {
		return BlastAnswer{}, nil, notes, err
	}
	ans := BlastAnswer{Range: rng}
	if opts.Keep != nil {
		kept := files[:0]
		for _, f := range files {
			if opts.Keep(f.Path) {
				kept = append(kept, f)
			} else {
				ans.Hidden++
			}
		}
		files = kept
	}
	depth := opts.Depth
	if depth <= 0 && depth != blast.All {
		depth = 1
	}
	ans.Report = blast.Radius(g, blast.New(g), files, depth)
	if opts.Keep != nil {
		hits := ans.Hits[:0]
		for _, h := range ans.Hits {
			if opts.Keep(h.Node.Path) {
				hits = append(hits, h)
			} else {
				ans.Hidden++
			}
		}
		ans.Hits = hits
		// A test path is a path like any hit's: refused, it is not named.
		for i := range ans.Areas {
			var tests []string
			for _, p := range ans.Areas[i].Tests {
				if opts.Keep(p) {
					tests = append(tests, p)
				} else {
					ans.Hidden++
				}
			}
			ans.Areas[i].Tests = tests
		}
	}
	return ans, g, notes, nil
}

// hasRevision says whether rev names a commit. Only rev-parse's exit 1 means
// it does not; any other failure (no repository, a refused safe.directory, a
// broken one) is git's, and returned as git said it.
func hasRevision(root, rev string) (bool, error) {
	_, err := gitOutput(root, "rev-parse", "--verify", "-q", rev+"^{commit}")
	var exit *exec.ExitError
	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return false, nil
	}
	return false, err
}

// changedFiles asks git twice: which files, then which lines. The revision
// comes after --end-of-options, so git never reads it as an option.
func changedFiles(root string, flags []string, rev string) ([]diff.File, error) {
	tail := append(append(flags, "--end-of-options"), rev)
	args := append([]string{"diff", "--relative", "-M", "--name-status", "-z"}, diffPrefixes...)
	names, err := gitOutput(root, append(args, tail...)...)
	if err != nil {
		return nil, err
	}
	files, err := diff.ParseNameStatus(bytes.NewReader(names))
	if err != nil || len(files) == 0 {
		return files, err
	}
	// Each flag overrides a setting of the user's that would change the
	// patch: a hunk fused under diff.interHunkContext spans lines nobody
	// changed and seeds the symbols between them.
	args = append([]string{"diff", "--relative", "-M", "--unified=0", "--inter-hunk-context=0",
		"--no-color", "--no-ext-diff", "--no-textconv"}, diffPrefixes...)
	patch, err := gitOutput(root, append(args, tail...)...)
	if err != nil {
		return nil, err
	}
	return files, diff.ApplyHunks(files, bytes.NewReader(patch))
}

// BlastReport formats a BlastAnswer one fact per line; empty sections are
// left out.
func BlastReport(a BlastAnswer) string {
	var b strings.Builder
	fmt.Fprintf(&b, "blast radius: %s\n", a.Range)
	if len(a.Areas) == 0 && len(a.Deleted) == 0 && len(a.Unindexed) == 0 {
		b.WriteString("no change\n")
	}
	for _, ar := range a.Areas {
		fmt.Fprintf(&b, "%s [%s]\n", ar.Path, ar.Signal)
		for _, s := range ar.Seeds {
			fmt.Fprintf(&b, "  seed %s in-degree %d\n", seedName(s.Node), s.InDegree)
		}
		if len(ar.Tests) > 0 {
			fmt.Fprintf(&b, "  tests: %s\n", strings.Join(ar.Tests, ", "))
		}
	}
	if len(a.Hits) > 0 {
		b.WriteString("reached:\n")
	}
	for _, h := range a.Hits {
		fmt.Fprintf(&b, "  %s (%s, depth %d, %s) in %s:%s from %s\n",
			h.Node.Name, h.Node.Kind, h.Depth, h.Relation, h.Node.Path, h.Node.Span, strings.Join(h.From, ", "))
	}
	for _, p := range a.Unindexed {
		fmt.Fprintf(&b, "not indexed: %s\n", p)
	}
	if len(a.Evidence) > 0 {
		b.WriteString("evidence:\n")
	}
	for _, e := range a.Evidence {
		fmt.Fprintf(&b, "  %s in %s\n", e.Node.Name, e.Node.Path)
		for _, l := range e.Lines {
			fmt.Fprintf(&b, "    %s\n", l)
		}
		if e.More > 0 {
			fmt.Fprintf(&b, "    … +%d lines\n", e.More)
		}
	}
	if a.MoreEvidence > 0 {
		fmt.Fprintf(&b, "  … %d more symbols\n", a.MoreEvidence)
	}
	// Deleted files close the report: nothing of theirs is left to reach.
	for _, p := range a.Deleted {
		fmt.Fprintf(&b, "deleted: %s\n", p)
	}
	if a.Hidden > 0 {
		fmt.Fprintf(&b, "hidden: %d\n", a.Hidden)
	}
	return b.String()
}

// seedName is a symbol with its kind and span; a file stands for itself.
func seedName(n *model.Node) string {
	if n.Kind == model.KindFile {
		return n.Path + " (file)"
	}
	return fmt.Sprintf("%s (%s) %s", n.Name, n.Kind, n.Span)
}
