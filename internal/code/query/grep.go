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
	// MaxFilesize is the flag's search corpus limit; nil means the project's
	// [grep] table, else the default, and 0 means unlimited.
	MaxFilesize *int64
	// Channel is "local" (or empty) or "cloud". On the cloud channel the
	// project's privacy declaration filters the files, and Keep, when the
	// caller supplies one, is that predicate already.
	Channel string
	// Glob narrows the corpus to paths matching the pattern; empty means every
	// path.
	Glob string
	// TrustIndex answers from the trigram index and its symbol sidecar
	// without walking the tree. An index that cannot be trusted falls back to
	// the checked path with a note.
	TrustIndex bool
}

// readRecord is a seam: a rebuild can leave no record behind only when a
// concurrent run rewrites the cache, which no test tree produces. The tests of
// this package stand in for that day.
var readRecord = corpus.Read

// Grep answers a pattern over the search corpus, from the trigram index when
// it is current, from the index plus a read of the moved files when a few
// files' content drifted, and from a scan otherwise.
//
// Both paths search the same file list, the one corpus.SearchCorpus selects:
// a graph file and a README match the same pattern, and the index can only be
// trusted because a scan of the same list is its oracle.
func Grep(root, pattern string, opts GrepOptions) (grep.Result, []string, error) {
	if _, err := os.Stat(store.WiringPath(root)); errors.Is(err, os.ErrNotExist) {
		return grep.Result{}, nil, ErrNoGraph
	}
	maxFilesize, err := resolveMaxFilesize(root, opts.MaxFilesize)
	if err != nil {
		return grep.Result{}, nil, err
	}
	var notes []string
	say := func(s string) { notes = append(notes, s) }

	if opts.TrustIndex {
		plan, err := trustedPlan(root, maxFilesize, pattern, opts)
		if err == nil {
			res, err := searchPlan(root, pattern, opts, plan, say)
			return res, notes, err
		}
		say("index not trusted: " + err.Error())
	}

	// One walk answers both the corpus and the graph; only drift pays for git.
	s, err := probeGrep(root, maxFilesize, true)
	if err != nil {
		return grep.Result{}, notes, err
	}
	// Trust is the corpus's alone: the graph lends symbols, not files.
	trusted := s.hasRecord && s.corpusDrift.Clean()
	reason := needsRebuild(s, root)
	// The overlay answers from the old index plus a read of the moved files.
	var moved map[string]bool
	switch {
	case reason == "":
	case takeOverlay(s, root, say):
		moved = movedFiles(s)
	case s.hasRecord && opts.MaxFilesize != nil && onlySettings(*s.corpusDrift) && graphReason(s, root) == "":
		say("settings differ from index, scanning with the flag value")
	case opts.NoRefresh:
		// Report the stale index and scan below.
	default:
		limit := rebuildLimit(s, opts.MaxFilesize, maxFilesize)
		if rebuilt, ok := rebuildGrep(root, s, reason, limit, say); ok {
			s, trusted = rebuilt, limit == maxFilesize
			if !trusted {
				say("settings differ from index, scanning with the flag value")
			}
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

	index := "none"
	var candidates map[string]bool
	ix, ixErr := trigram.Read(root)
	switch {
	case ixErr == nil:
		digest, digestOK := recordDigest(rec)
		if (!trusted && moved == nil) || !digestOK || ix.Extractor() != all.Version() ||
			string(ix.Settings()) != string(settings) || ix.Digest() != digest {
			index = "stale"
			break
		}
		// A pattern without a literal condition cannot be narrowed, but the
		// index is still current: that is a scan, not a stale index.
		index = "scan"
		candidates = narrow(ix, pattern, opts, moved)
		if candidates != nil {
			index = "used"
		}
		if moved != nil {
			index = "overlay"
		}
	case errors.Is(ixErr, trigram.ErrNoIndex):
		index = "none"
	default:
		index = "stale"
	}
	// The candidates are copies. The scan that follows has no bound, and a
	// build waiting to replace the index on Windows has about 2 s.
	if ixErr == nil {
		ix.Close()
	}

	digest, digestOK := recordDigest(rec)
	found, err := searchPlan(root, pattern, opts, grepPlan{
		base:       res,
		paths:      paths,
		candidates: candidates,
		index:      index,
		symbols: func(search symbolSearch) (grep.Result, error) {
			return withSymbols(root, digest, digestOK, func(oldSpans func(string) []model.SymbolSpan, inDegree func(model.NodeID) int) (grep.Result, error) {
				// A moved file's old spans would name the wrong symbol: none is better.
				spans := func(p string) []model.SymbolSpan {
					if moved[p] {
						return nil
					}
					return oldSpans(p)
				}
				return search(spans, inDegree)
			})
		},
	}, say)
	return found, notes, err
}

// grepPlan is what one search needs once the corpus, the index and the
// symbols are settled, however the caller got there.
type grepPlan struct {
	// base carries the pattern and the counters of what the corpus left out.
	base grep.Result
	// paths is the corpus; candidates, when not nil, narrows it.
	paths      []string
	candidates map[string]bool
	index      string
	// built is when the index was written, set on a trusted answer only.
	built string
	// uncounted is how many corpus files the paths leave out (the binaries
	// of a trusted index's file table), so the scope counts match a scan's.
	// Under --in or --glob it overcounts, as the table cannot place them.
	uncounted int
	symbols   func(symbolSearch) (grep.Result, error)
}

// searchPlan applies the channel, the prefix and the glob to the plan and
// runs the search. A search error returns what the search found until then.
func searchPlan(root, pattern string, opts GrepOptions, plan grepPlan, say func(string)) (grep.Result, error) {
	res, paths, candidates := plan.base, plan.paths, plan.candidates
	channel, err := grepChannel(opts.Channel)
	if err != nil {
		return res, err
	}
	keep := opts.Keep
	if keep == nil && channel == privacy.ChannelCloud {
		keep, err = cloudKeep(root)
		if err != nil {
			return res, err
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
			say("no file under this prefix is readable on the cloud channel")
			return res, nil
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

	if glob := globFilter(opts.Glob); glob != nil {
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

	found, err := plan.symbols(func(spans func(string) []model.SymbolSpan, inDegree func(model.NodeID) int) (grep.Result, error) {
		return grep.Search(nil, nil, nil, pattern, grep.Options{
			IgnoreCase: opts.IgnoreCase,
			Fixed:      opts.Fixed,
			In:         opts.In,
			MaxHits:    opts.MaxHits,
			Files:      paths,
			All:        scopePaths,
			Candidates: candidates,
			Index:      plan.index,
			Spans:      spans,
			InDegree:   inDegree,
		}, reader)
	})
	if err != nil {
		return found, err
	}
	found.Ignored = res.Ignored
	found.SkippedSize = res.SkippedSize
	found.SkippedBinary = res.SkippedBinary
	found.IndexBuilt = plan.built
	found.FilesInScope += plan.uncounted
	if candidates == nil {
		found.Candidates += plan.uncounted
	}
	return found, nil
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
