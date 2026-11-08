package query

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/trigram"
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
	// MaxFilesize is the resolved search corpus limit; nil means the default
	// and 0 means unlimited.
	MaxFilesize *int64
	// Channel is "local" (or empty) or "cloud". On the cloud channel the
	// project's privacy declaration filters the files, and Keep, when the
	// caller supplies one, is that predicate already.
	Channel string
	// Glob narrows the corpus to paths matching the pattern; empty means every
	// path.
	Glob string
}

// Grep answers a pattern over the search corpus, from the trigram index when
// it is current and from a scan when it is not.
//
// Both paths search the same file list, the one corpus.SearchCorpus selects:
// a graph file and a README match the same pattern, and the index can only be
// trusted because a scan of the same list is its oracle.
// readRecord is a seam: a rebuild can leave no record behind only when a
// concurrent run rewrites the cache, which no test tree produces. The tests of
// this package stand in for that day.
var readRecord = corpus.Read

func Grep(root, pattern string, opts GrepOptions) (grep.Result, []string, error) {
	g, notes, err := loadGraph(root, opts.NoRefresh)
	if err != nil {
		return grep.Result{}, notes, err
	}
	maxFilesize, err := corpus.ResolveMaxFilesize(opts.MaxFilesize, nil)
	if err != nil {
		return grep.Result{}, notes, err
	}
	say := func(s string) { notes = append(notes, s) }

	// The record and its matcher answer the common case without git: walk
	// once, compare, keep the listing. Only drift pays for Load.
	rec, hasRecord := readRecord(root)
	var rules *corpus.Rules
	var listing corpus.Listing
	trusted := false
	rebuild := func(reason string) bool {
		_, rerr := ask.Refresh(root, all.Version(), func() error {
			_, _, buildErr := BuildWith(root, BuildOptions{MaxFilesize: &maxFilesize}, nil)
			return buildErr
		}, ask.RefreshOptions{Force: reason, Notice: say})
		if rerr != nil {
			say(fmt.Sprintf("search corpus not rebuilt, answering from the disk: %v", rerr))
			return false
		}
		rec, hasRecord = readRecord(root)
		if !hasRecord {
			return false
		}
		rules = corpus.RulesFromRecord(rec)
		walked, werr := walkCorpus(root, matcherFor(rules))
		if werr != nil {
			err = werr
			return false
		}
		listing = walked
		trusted = true
		return true
	}
	if hasRecord {
		rules = corpus.RulesFromRecord(rec)
		listing, err = walkCorpus(root, matcherFor(rules))
		if err != nil {
			return grep.Result{}, notes, err
		}
		drift := corpus.Compare(rec, listing, rules, corpus.Settings{MaxFilesize: maxFilesize})
		onlySettings := drift.Settings && !drift.Index && !drift.Config &&
			len(drift.Changed) == 0 && len(drift.Added) == 0 &&
			len(drift.Removed) == 0 && len(drift.Sources) == 0
		switch {
		case drift.Clean():
			trusted = true
		case onlySettings && opts.MaxFilesize != nil:
			say("settings differ from index, scanning with the flag value")
		case opts.NoRefresh:
			// Report the stale index and scan below.
		default:
			rebuild(fmt.Sprintf("search corpus moved (%d entries), rebuilding", drift.Count()))
			if err != nil {
				return grep.Result{}, notes, err
			}
		}
	} else {
		rules, _, err = loadIgnore(root)
		if err != nil {
			return grep.Result{}, notes, err
		}
		listing, err = walkCorpus(root, matcherFor(rules))
		if err != nil {
			return grep.Result{}, notes, err
		}
		if !opts.NoRefresh {
			rebuild("no search corpus record, rebuilding")
			if err != nil {
				return grep.Result{}, notes, err
			}
		}
	}

	files, ignored, skippedSize := corpus.SearchCorpus(listing, maxFilesize)
	if trusted && hasRecord {
		ignored, skippedSize = rec.Ignored, rec.SkippedSize
	}
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Rel
	}

	ignoreCase := rules != nil && rules.IgnoreCase()
	if hasRecord {
		ignoreCase = rec.IgnoreCase
	}
	settings := corpus.CanonicalSettings(maxFilesize, ignoreCase)

	// What the corpus left out, for the report. On a trusted record the
	// counters come from it, so scan and index path report equal numbers.
	res := grep.Result{Pattern: pattern, Ignored: ignored, SkippedSize: skippedSize}
	if trusted && hasRecord {
		res.SkippedBinary = rec.SkippedBinary
	} else if hasRecord {
		for _, print := range rec.Files {
			if print.Binary {
				res.SkippedBinary++
			}
		}
	}

	channel, err := grepChannel(opts.Channel)
	if err != nil {
		return res, notes, err
	}
	keep := opts.Keep
	if keep == nil && channel == privacy.ChannelCloud {
		keep, err = cloudKeep(root)
		if err != nil {
			return res, notes, err
		}
	}

	// A prefix under a hidden path is answered empty with a note: the error
	// "prefix not indexed" would tell the caller whether the path exists.
	if channel == privacy.ChannelCloud && keep != nil && opts.In != "" {
		prefix := lexicon.NormalizePrefix(opts.In)
		under, readable := 0, 0
		for _, p := range paths {
			if !lexicon.UnderPrefix(p, prefix) {
				continue
			}
			under++
			if keep(p) {
				readable++
			}
		}
		if under > 0 && readable == 0 {
			res.Index = "scan"
			return res, append(notes, "no file under this prefix is readable on the cloud channel"), nil
		}
	}

	scopePaths := paths
	if glob := globFilter(opts.Glob); glob != nil {
		filtered := make([]string, 0, len(paths))
		for _, p := range paths {
			if glob(p) {
				filtered = append(filtered, p)
			}
		}
		paths = filtered
	}

	index := "none"
	var candidates map[string]bool
	ix, ixErr := trigram.Read(root)
	switch {
	case ixErr == nil:
		// The candidates are copies; the mapping must not outlive the call, or
		// the next build could not replace the file on Windows.
		defer ix.Close()
		if !trusted || !hasRecord || ix.Extractor() != all.Version() ||
			string(ix.Settings()) != string(settings) || ix.Digest() != corpus.Digest(rec) {
			index = "stale"
			break
		}
		// A pattern without a literal condition cannot be narrowed, but the
		// index is still current: that is a scan, not a stale index.
		index = "scan"
		if sets, ok, lerr := trigram.Literals(pattern, opts.Fixed, opts.IgnoreCase); lerr == nil && ok {
			if found, used := ix.Candidates(sets); used {
				candidates = make(map[string]bool, len(found))
				for _, p := range found {
					candidates[p] = true
				}
				index = "used"
			}
		}
	case errors.Is(ixErr, trigram.ErrNoIndex):
		index = "none"
	default:
		index = "stale"
	}
	if glob := globFilter(opts.Glob); glob != nil && candidates != nil {
		for p := range candidates {
			if !glob(p) {
				delete(candidates, p)
			}
		}
	}

	x := blast.New(g)
	spans := model.FileSpans(g)
	reader := func(p string) ([]byte, error) {
		if keep != nil && !keep(p) {
			return nil, os.ErrPermission
		}
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	}

	found, err := grep.Search(g, x, spans, pattern, grep.Options{
		IgnoreCase: opts.IgnoreCase,
		Fixed:      opts.Fixed,
		In:         opts.In,
		MaxHits:    opts.MaxHits,
		Files:      paths,
		All:        scopePaths,
		Candidates: candidates,
		Index:      index,
	}, reader)
	if err != nil {
		return found, notes, err
	}
	found.Ignored = res.Ignored
	found.SkippedSize = res.SkippedSize
	found.SkippedBinary = res.SkippedBinary
	return found, notes, nil
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
