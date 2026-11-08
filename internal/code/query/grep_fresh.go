package query

import (
	"encoding/hex"
	"fmt"
	"os"
	"slices"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/code/spanidx"
)

// overlayLimit is how many files may move, per drift, before grep rebuilds
// instead of reading the moved files beside the old index.
const overlayLimit = 200

// buildWith, trackedSet and writeRecord are seams: the tests count the
// rebuilds and the git calls grep makes, and a record write fails only on a
// broken disk.
var (
	buildWith   = BuildWith
	trackedSet  = corpus.TrackedSet
	writeRecord = corpus.Write
)

// overlayFits reports whether the drift is small file content only, so the
// old index still answers for every file that did not move. A moved git index
// fits when it still tracks the same files; that check is the only git call,
// and it runs last.
func overlayFits(s grepState, root string) bool {
	d := s.corpusDrift
	if d == nil || d.Settings || d.Config || len(d.Sources) > 0 || s.graphDrift == nil {
		return false
	}
	// Count leaves the index flag out here: it is not a file.
	if len(d.Changed)+len(d.Added)+len(d.Removed) > overlayLimit || s.graphDrift.Count() > overlayLimit {
		return false
	}
	if !lexicon.Usable(root) {
		return false
	}
	if d.Index {
		tracked, err := trackedSet(root)
		return err == nil && slices.Equal(tracked, s.rec.Tracked)
	}
	return true
}

// takeOverlay reports whether grep answers from the overlay. When the git
// index moved but tracks the same files, the record takes its new stamp, so
// the next call skips the git call; a failed write costs only that. A record
// that changed since the probe belongs to a concurrent rebuild and is left
// alone: the old file list with the new stamp would no longer match its index.
//
// ponytail: compare-then-write leaves a window of one file write; a rebuild
// landing in it is undone until the next drift. Take the ask rebuild lock
// here if that ever shows.
func takeOverlay(s grepState, root string, say func(string)) bool {
	moved := s.corpusDrift != nil && s.corpusDrift.Index
	var stamp corpus.Stamp
	if moved {
		// Stat before git lists the index: a change in between leaves the
		// record behind the index, never ahead of it.
		stamp = gitIndexStamp(s.rec)
	}
	if !overlayFits(s, root) {
		return false
	}
	if moved && unchanged(root, s.rec) {
		rec := s.rec
		rec.Stamp = stamp
		if err := writeRecord(root, rec); err != nil {
			say(fmt.Sprintf("search corpus record not restamped: %v", err))
		}
	}
	return true
}

// unchanged reports whether the record on disk is still the one the probe read.
func unchanged(root string, rec corpus.Record) bool {
	now, ok := readRecord(root)
	return ok && now.Digest == rec.Digest && now.Stamp == rec.Stamp
}

// gitIndexStamp is the stamp of the git index the record names, the empty
// stamp when it is gone, which is how the probe reads a missing index.
func gitIndexStamp(rec corpus.Record) corpus.Stamp {
	info, err := os.Stat(corpus.GitAbs(rec, rec.GitPaths.Index))
	if err != nil {
		return corpus.Stamp{}
	}
	return corpus.Stamp{Size: info.Size(), MTime: info.ModTime().UnixNano()}
}

// movedFiles are the files the old index and the old symbols no longer
// describe, in either drift.
func movedFiles(s grepState) map[string]bool {
	moved := map[string]bool{}
	for _, p := range slices.Concat(s.corpusDrift.Changed, s.corpusDrift.Added, s.graphDrift.Changed, s.graphDrift.Added) {
		moved[p] = true
	}
	return moved
}

// grepState is what one walk tells grep about the search corpus and the graph.
// corpusDrift is nil without a record; graphDrift is nil when the graph's
// fingerprint is missing or from another extractor.
type grepState struct {
	rec         corpus.Record
	hasRecord   bool
	rules       *corpus.Rules
	listing     corpus.Listing
	corpusDrift *corpus.Drift
	graphDrift  *freshness.Drift
}

// probeGrep reads the record and walks the tree once; the graph's file set is
// a filter over the same listing.
func probeGrep(root string, maxFilesize int64) (grepState, error) {
	var s grepState
	var err error
	s.rec, s.hasRecord = readRecord(root)
	if s.hasRecord {
		s.rules = corpus.RulesFromRecord(s.rec)
	} else if s.rules, _, err = loadIgnore(root); err != nil {
		return s, err
	}
	if s.listing, err = walkCorpus(root, matcherFor(s.rules)); err != nil {
		return s, err
	}
	if s.hasRecord {
		s.corpusDrift = corpus.Compare(s.rec, s.listing, s.rules, corpus.Settings{MaxFilesize: maxFilesize})
	}
	// Over a given listing the probe has nothing left to fail on; should it
	// ever, its nil drift reads as unknown and rebuilds.
	s.graphDrift, _ = freshness.ProbeFiles(root, all.Version(), sourceset.FromListing(s.listing))
	return s, nil
}

// needsRebuild names why the graph or the search corpus must be rebuilt, or
// returns "" when both are current. The graph's reasons are ask.Refresh's.
func needsRebuild(s grepState, root string) string {
	if reason := graphReason(s, root); reason != "" {
		return reason
	}
	switch {
	case !s.hasRecord:
		return "no search corpus record, rebuilding"
	case !s.corpusDrift.Clean():
		return fmt.Sprintf("search corpus moved (%d entries), rebuilding", s.corpusDrift.Count())
	}
	return ""
}

// onlySettings reports a drift in nothing but the settings, which a flag for
// one call explains without a rebuild.
func onlySettings(d corpus.Drift) bool {
	return d.Settings && d.Count() == 1
}

func graphReason(s grepState, root string) string {
	switch {
	case s.graphDrift == nil:
		return "no freshness record, building the graph"
	case !s.graphDrift.Clean():
		return fmt.Sprintf("%d files moved, rebuilding the graph", s.graphDrift.Count())
	case !lexicon.Usable(root):
		return "no ask index, rebuilding the graph"
	}
	return ""
}

// rebuildLimit is the size limit a rebuild writes into the record. A flag for
// one call never does: with one, the record keeps its own limit, or gets the
// default when there is none.
func rebuildLimit(s grepState, flag *int64, resolved int64) int64 {
	switch {
	case flag == nil:
		return resolved
	case s.hasRecord:
		return s.rec.MaxFilesize
	}
	return corpus.DefaultMaxFilesize
}

// rebuildGrep rebuilds graph and corpus and returns the state of the new
// record, with the listing the build walked. ok is false when the rebuild did
// not run or left no indexed corpus; s still describes the tree then.
func rebuildGrep(root string, s grepState, reason string, maxFilesize int64, say func(string)) (grepState, bool) {
	var listing corpus.Listing
	indexed := false
	_, err := ask.Refresh(root, all.Version(), func() error {
		_, stats, err := buildWith(root, BuildOptions{MaxFilesize: &maxFilesize, Listing: &listing}, say)
		indexed = stats.Corpus.Indexed
		return err
	}, ask.RefreshOptions{Force: reason, Notice: say})
	if err != nil {
		say(fmt.Sprintf("search corpus not rebuilt, answering from the disk: %v", err))
		return s, false
	}
	if !indexed {
		return s, false
	}
	rec, ok := readRecord(root)
	if !ok {
		return s, false
	}
	return grepState{rec: rec, hasRecord: true, rules: corpus.RulesFromRecord(rec), listing: listing}, true
}

// recordDigest decodes the digest the record stores; ok is false when it is
// not one.
func recordDigest(rec corpus.Record) (digest [32]byte, ok bool) {
	raw, err := hex.DecodeString(rec.Digest)
	if err != nil || len(raw) != len(digest) {
		return digest, false
	}
	return [32]byte(raw), true
}

// symbolSearch is one search with the given symbols.
type symbolSearch func(spans func(string) []model.SymbolSpan, inDegree func(model.NodeID) int) (grep.Result, error)

// withSymbols runs search with the symbols of the spans sidecar the record's
// digest binds, or with the graph's when there is no such sidecar. The sidecar
// is closed before it returns.
func withSymbols(root string, rec corpus.Record, search symbolSearch) (grep.Result, error) {
	if digest, ok := recordDigest(rec); ok {
		if r, err := spanidx.Open(root, digest); err == nil {
			defer r.Close()
			return search(r.Spans, r.InDegree)
		}
	}
	g, err := readGraph(root)
	if err != nil {
		return grep.Result{}, err
	}
	spans := model.FileSpans(g)
	return search(func(p string) []model.SymbolSpan { return spans[p] }, blast.New(g).InDegree)
}
