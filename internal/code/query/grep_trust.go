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

// trustedPlan maps the index, checks that it and its symbol sidecar belong
// together and to this binary and these settings, and returns the search over
// the index's own file table. The error says why not; the caller then takes
// the checked path.
//
// Neither mapping outlives the call: the plan holds copies, and the sidecar
// is mapped again for the merge. A rebuild that removes it in between leaves
// the symbols to the graph, as on the checked path.
func trustedPlan(root string, maxFilesize int64, pattern string, opts GrepOptions) (grepPlan, error) {
	ix, err := trigram.Read(root)
	if err != nil {
		return grepPlan{}, err
	}
	defer ix.Close()
	built, err := bindTrusted(root, ix, maxFilesize)
	if err != nil {
		return grepPlan{}, err
	}
	c, digest := ix.Counters(), ix.Digest()
	return grepPlan{
		base: grep.Result{
			Pattern: pattern, Ignored: c.Ignored, SkippedSize: c.SkippedSize, SkippedBinary: c.SkippedBinary,
		},
		paths:      ix.Files(),
		candidates: narrow(ix, pattern, opts, nil),
		index:      "trusted",
		built:      built,
		// The table holds no binary file, so the scope counts add the ones
		// the header counts.
		uncounted: c.SkippedBinary,
		symbols: func(search symbolSearch) (grep.Result, error) {
			return withSymbols(root, digest, true, search)
		},
	}, nil
}

// bindTrusted returns when the index was written, or why it is not trusted.
func bindTrusted(root string, ix *trigram.Index, maxFilesize int64) (string, error) {
	if ix.Extractor() != all.Version() {
		return "", fmt.Errorf("the index is from another extractor version (%s)", ix.Extractor())
	}
	if string(ix.Settings()) != string(corpus.CanonicalSettings(maxFilesize, ix.Counters().IgnoreCase)) {
		return "", fmt.Errorf("the index was built with other settings (%s)", ix.Settings())
	}
	info, err := statIndex(store.CachePath(root, "trigram.idx"))
	if err != nil {
		return "", err
	}
	spans, err := spanidx.Open(root, ix.Digest())
	if err != nil {
		return "", err
	}
	spans.Close()
	return info.ModTime().UTC().Format(time.RFC3339), nil
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
