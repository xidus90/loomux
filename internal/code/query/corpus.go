package query

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/trigram"
)

// CorpusStats is what the search corpus selection came to. It is separate from
// Stats, which counts the graph corpus: the two sets differ on purpose.
type CorpusStats struct {
	Files         int
	Ignored       int
	SkippedSize   int
	SkippedBinary int
	Unreadable    int
	Indexed       bool
	// Digest is the corpus digest the record and the trigram index carry.
	Digest [32]byte
}

// writeCorpus selects the search corpus, hashes it, and writes its record and
// trigram index.
//
// bodies are the bytes the extraction already read, keyed by repo-relative
// path: a file that is both a graph file and a search file is hashed once. A
// file only the search corpus wants is read here, once.
//
// Nothing in here fails the build. The graph and its sidecars are what a
// question needs; a corpus that cannot be selected or written costs the next
// grep its index and says why.
//
// walked, when not nil, receives the listing.
func writeCorpus(root string, maxFilesize int64, bodies map[string][]byte, walked *corpus.Listing, notice func(string)) CorpusStats {
	var st CorpusStats
	rules, _, err := loadIgnore(root)
	if err != nil {
		notice(fmt.Sprintf("search corpus skipped: %v", err))
		return st
	}
	matcher := matcherFor(rules)
	listing, err := walkCorpus(root, matcher)
	if err != nil {
		notice(fmt.Sprintf("search corpus skipped: %v", err))
		return st
	}
	if walked != nil {
		*walked = listing
	}
	files, ignored, skippedSize := corpus.SearchCorpus(listing, maxFilesize)
	st.Ignored, st.SkippedSize = ignored, skippedSize
	st.Files = len(files)

	prints := make(map[string]corpus.Print, len(files))
	indexed := make([]trigram.File, 0, len(files))
	for _, f := range files {
		data, ok := bodies[f.Rel]
		if !ok {
			data, err = readFile(f.Abs)
			if err != nil {
				// Unreadable is skipped, counted and recorded with an empty
				// hash: the only way to learn it is readable again is to try.
				st.Unreadable++
				prints[f.Rel] = corpus.Print{Size: f.Size, MTime: f.MTime}
				continue
			}
		}
		sum := sha256.Sum256(data)
		binary := corpus.IsBinary(data)
		if binary {
			st.SkippedBinary++
		}
		prints[f.Rel] = corpus.Print{
			Size: f.Size, MTime: f.MTime,
			Hash: hex.EncodeToString(sum[:]), Binary: binary,
		}
		if !binary {
			indexed = append(indexed, trigram.File{Rel: f.Rel, Data: data})
		}
	}

	rec := corpus.NewRecord(rules, prints, maxFilesize)
	st.Digest = corpus.Digest(rec)
	rec.Ignored, rec.SkippedSize, rec.SkippedBinary = st.Ignored, st.SkippedSize, st.SkippedBinary
	counters := trigram.Counters{
		Ignored: st.Ignored, SkippedSize: st.SkippedSize, SkippedBinary: st.SkippedBinary,
		IgnoreCase: rec.IgnoreCase,
	}
	// The index goes first: a record never names an index that was not
	// written, so a failed write leaves the old pair standing.
	if err := writeIndex(root, st.Digest, all.Version(), rec.Settings, counters, indexed); err != nil {
		notice(fmt.Sprintf("trigram index not written, the old record stays: %v", err))
		return st
	}
	st.Indexed = true
	if err := corpus.Write(root, rec); err != nil {
		noticeRecord(notice, "search corpus record", err)
	}
	return st
}

// matcherFor keeps a nil *corpus.Rules from reaching Walk as a non-nil
// interface, which would call a method on a nil pointer.
func matcherFor(rules *corpus.Rules) corpus.Matcher {
	if rules == nil {
		return nil
	}
	return rules
}

// loadIgnore and walkCorpus are seams: the ignore load fails only when git
// refuses `ls-files`, and the walk fails only when the root vanishes between
// reading the record and walking it. Both arms announce and let the build
// stand; the tests stand in for the day a real tree produces them.
var (
	loadIgnore = corpus.Load
	walkCorpus = corpus.Walk
	// writeIndex fails on a disk that refuses the file, or on Windows when a
	// reader keeps the old index mapped past the replace budget.
	writeIndex = trigram.Write
)

// readFile is the read the extraction and the corpus selection both go
// through, so a test can count that a file shared by the two is read once.
var readFile = os.ReadFile
