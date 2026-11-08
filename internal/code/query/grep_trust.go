package query

import (
	"fmt"
	"os"
	"time"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/spanidx"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/code/trigram"
)

// statIndex is a seam: the index was just mapped, so a stat fails only on a
// disk that is going away.
var statIndex = os.Stat

// trustedIndex is the trigram index and the symbol sidecar that grep answers
// from without looking at the tree.
type trustedIndex struct {
	ix    *trigram.Index
	spans *spanidx.Reader
	built string
}

// openTrusted maps the index and its sidecar and checks that they belong
// together and to this binary and these settings. The error says why not; the
// caller then takes the checked path.
func openTrusted(root string, maxFilesize int64) (*trustedIndex, error) {
	ix, err := trigram.Read(root)
	if err != nil {
		return nil, err
	}
	t, err := bindTrusted(root, ix, maxFilesize)
	if err != nil {
		ix.Close()
		return nil, err
	}
	return t, nil
}

func bindTrusted(root string, ix *trigram.Index, maxFilesize int64) (*trustedIndex, error) {
	if ix.Extractor() != all.Version() {
		return nil, fmt.Errorf("the index is from another extractor version (%s)", ix.Extractor())
	}
	if string(ix.Settings()) != string(corpus.CanonicalSettings(maxFilesize, ix.Counters().IgnoreCase)) {
		return nil, fmt.Errorf("the index was built with other settings (%s)", ix.Settings())
	}
	info, err := statIndex(store.CachePath(root, "trigram.idx"))
	if err != nil {
		return nil, err
	}
	spans, err := spanidx.Open(root, ix.Digest())
	if err != nil {
		return nil, err
	}
	return &trustedIndex{ix: ix, spans: spans, built: info.ModTime().UTC().Format(time.RFC3339)}, nil
}

func (t *trustedIndex) close() {
	t.spans.Close()
	t.ix.Close()
}

// plan is the search over the index's own file table and counters. The table
// holds no binary file, so the scope counts add the ones the header counts.
func (t *trustedIndex) plan(pattern string, opts GrepOptions) grepPlan {
	c := t.ix.Counters()
	return grepPlan{
		base: grep.Result{
			Pattern: pattern, Ignored: c.Ignored, SkippedSize: c.SkippedSize, SkippedBinary: c.SkippedBinary,
		},
		paths:      t.ix.Files(),
		candidates: narrow(t.ix, pattern, opts, nil),
		index:      "trusted",
		built:      t.built,
		uncounted:  c.SkippedBinary,
		symbols: func(search symbolSearch) (grep.Result, error) {
			return search(t.spans.Spans, t.spans.InDegree)
		},
	}
}

// narrow is the set of files the index says may match, plus extra, or nil
// when the pattern has no literal condition to narrow by.
func narrow(ix *trigram.Index, pattern string, opts GrepOptions, extra map[string]bool) map[string]bool {
	sets, ok, err := trigram.Literals(pattern, opts.Fixed, opts.IgnoreCase)
	if err != nil || !ok {
		return nil
	}
	var candidates map[string]bool
	if found, used := ix.Candidates(sets); used {
		candidates = make(map[string]bool, len(found)+len(extra))
		for _, p := range found {
			candidates[p] = true
		}
		for p := range extra {
			candidates[p] = true
		}
	}
	return candidates
}
