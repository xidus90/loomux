package query

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
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

// readRecord is a seam: a rebuild can leave no record behind only when a
// concurrent run rewrites the cache, which no test tree produces. The tests of
// this package stand in for that day.
var readRecord = corpus.Read

// Grep answers a pattern over the search corpus, from the trigram index when
// it is current and from a scan when it is not.
//
// Both paths search the same file list, the one corpus.SearchCorpus selects:
// a graph file and a README match the same pattern, and the index can only be
// trusted because a scan of the same list is its oracle.
func Grep(root, pattern string, opts GrepOptions) (grep.Result, []string, error) {
	if _, err := os.Stat(store.WiringPath(root)); errors.Is(err, os.ErrNotExist) {
		return grep.Result{}, nil, ErrNoGraph
	}
	maxFilesize, err := corpus.ResolveMaxFilesize(opts.MaxFilesize, nil)
	if err != nil {
		return grep.Result{}, nil, err
	}
	var notes []string
	say := func(s string) { notes = append(notes, s) }

	// One walk answers both the corpus and the graph; only drift pays for git.
	s, err := probeGrep(root, maxFilesize)
	if err != nil {
		return grep.Result{}, notes, err
	}
	// Trust is the corpus's alone: the graph lends symbols, not files.
	trusted := s.hasRecord && s.corpusDrift.Clean()
	reason := needsRebuild(s, root)
	switch {
	case reason == "":
	case s.hasRecord && opts.MaxFilesize != nil && onlySettings(*s.corpusDrift) && graphReason(s, root) == "":
		say("settings differ from index, scanning with the flag value")
	case opts.NoRefresh:
		// Report the stale index and scan below.
	default:
		if rebuilt, ok := rebuildGrep(root, s, reason, maxFilesize, say); ok {
			s, trusted = rebuilt, true
		}
	}
	rec, hasRecord, rules, listing := s.rec, s.hasRecord, s.rules, s.listing

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
		digest, digestOK := recordDigest(rec)
		if !trusted || !digestOK || ix.Extractor() != all.Version() ||
			string(ix.Settings()) != string(settings) || ix.Digest() != digest {
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

	reader := func(p string) ([]byte, error) {
		if keep != nil && !keep(p) {
			return nil, os.ErrPermission
		}
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	}

	found, err := withSymbols(root, rec, func(spans func(string) []model.SymbolSpan, inDegree func(model.NodeID) int) (grep.Result, error) {
		return grep.Search(nil, nil, nil, pattern, grep.Options{
			IgnoreCase: opts.IgnoreCase,
			Fixed:      opts.Fixed,
			In:         opts.In,
			MaxHits:    opts.MaxHits,
			Files:      paths,
			All:        scopePaths,
			Candidates: candidates,
			Index:      index,
			Spans:      spans,
			InDegree:   inDegree,
		}, reader)
	})
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
