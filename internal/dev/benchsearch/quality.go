package benchsearch

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// top is the rank a hit must reach: the expected source among the first three.
const top = 3

// Ask runs one query against the search chain and returns the ranked
// absolute paths of what it found.
type Ask func(query string) ([]string, error)

// Outcome is one question of a quality run.
type Outcome struct {
	Question  Question
	Rank      int // 0: not found
	Hit       bool
	ElapsedMS float64
}

// RunQuality asks every question in the set's order and measures one rank
// each, never a score: a high score is no proof of truth. A search error
// ends the run, so a partial run cannot pass for a whole one.
func RunQuality(qs []Question, ask Ask, clock func() time.Time) ([]Outcome, error) {
	outcomes := make([]Outcome, 0, len(qs))
	for _, q := range qs {
		started := clock()
		ranked, err := ask(q.Query)
		if err != nil {
			return nil, err
		}
		elapsed := benchreport.MS(clock().Sub(started))
		rank := rankOf(q.Expect, ranked)
		outcomes = append(outcomes, Outcome{Question: q, Rank: rank, Hit: rank > 0 && rank <= top, ElapsedMS: elapsed})
	}
	return outcomes, nil
}

// rankOf is the one-based position of expect among ranked, 0 when absent.
func rankOf(expect string, ranked []string) int {
	for i, candidate := range ranked {
		if sameFile(expect, candidate) {
			return i + 1
		}
	}
	return 0
}

// sameFile tells whether two paths name one file. The question set carries
// the spelling somebody typed, the engine the one on disk. Spelling settles
// most pairs cheaply; a junction or an 8.3 short name shares no spelling with
// its long form, and only the file system can tell. A path that cannot be
// read names no file this run could have found.
func sameFile(a, b string) bool {
	if ca, cb := filepath.Clean(a), filepath.Clean(b); ca == cb || (runtime.GOOS == "windows" && strings.EqualFold(ca, cb)) {
		return true
	}
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ia, ib)
}
