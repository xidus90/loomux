package query

import (
	"encoding/hex"
	"fmt"

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

// rebuildGrep rebuilds graph and corpus and returns the state of the new
// record, with the listing the build walked. ok is false when the rebuild did
// not run or left no indexed corpus; s still describes the tree then.
func rebuildGrep(root string, s grepState, reason string, maxFilesize int64, say func(string)) (grepState, bool) {
	var listing corpus.Listing
	indexed := false
	_, err := ask.Refresh(root, all.Version(), func() error {
		_, stats, err := BuildWith(root, BuildOptions{MaxFilesize: &maxFilesize, Listing: &listing}, say)
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
