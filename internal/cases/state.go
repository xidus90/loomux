package cases

import (
	"bytes"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Normalizer rewrites one side of a world_after comparison before the two are
// held against each other. world is the case's recorded world, the state the
// run started from; tree is the side to rewrite, keyed by slash-separated path.
// Both sides go through the same normalizer, so what it folds away is exactly
// what the comparison no longer sees -- and nothing else.
type Normalizer func(world, tree map[string][]byte) map[string][]byte

// The tokens NormalizeState writes, one per value no two runs agree on.
const (
	nowToken   = "{{NOW}}"
	todayToken = "{{TODAY}}"
	mtimeToken = "{{MTIME}}"
	docIDToken = "{{DOCID}}"
)

const (
	lastRunPath    = "maintenance/last-run.txt"
	identitiesBase = "_identities.tsv"
)

// runStamp is the one spelling of the pass's `now` both writers produce:
// Python's `datetime.isoformat()` of a UTC instant, whose fraction is six
// digits or, when it is zero, absent. Anything else is not tokenized.
var runStamp = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})T\d{2}:\d{2}:\d{2}(\.\d{6})?\+00:00$`)

// statsPath is the stat cache of one area's pass, `_stat_path`.
var statsPath = regexp.MustCompile(`^maintenance/[^/]+/stats\.tsv$`)

// nanoseconds is an `st_mtime_ns` of this era: nineteen digits, from 2001 to
// 2286. A value in seconds or milliseconds is a writer that no longer writes
// what the reference writes -- and a reference pass after it would hash every
// source anew -- so it is not folded.
var nanoseconds = regexp.MustCompile(`^\d{19}$`)

// docID is the shape `NewDocID` and `new_doc_id` mint: 26 characters of the
// Crockford alphabet.
var docID = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// NormalizeState is the normalization of stage 3a's file worlds, as narrow as
// the three things it folds:
//
//   - The pass's `now`. `maintenance/last-run.txt` holds it, and wherever that
//     exact stamp stands in a file -- a case's `created`, a package's
//     `generated.at` -- it becomes {{NOW}}. The day it names becomes {{TODAY}}
//     where a case id carries it, `<segment>-<day>-<four hex digits>`, in a
//     path and in a file alike. A stamp of any other shape is not a stamp of
//     the reference's shape, and it stays to fail the comparison.
//   - The modification time in a pass's stat cache,
//     `maintenance/<collection>/stats.tsv`: it is the time the world was
//     staged, which differs between the recording and every replay. Only a
//     nanosecond count in that one column is folded; the path and the size
//     stay.
//   - A doc id an index run minted. A register row whose id the world did not
//     already hold gets {{DOCID}}, and the rows of such a register are sorted:
//     the register is ordered by id, so where a random id lands is as random as
//     the id. A register without a minted id keeps its order to the byte. An
//     id minted twice -- on more than one row of the tree's registers -- is
//     not folded: a doc id is unique across a vault, and the fold would hide
//     exactly the duplicate.
//
// Nothing else is touched. The catalogs carry no time at all, and the case
// directories and files differ from run to run only in the day and the stamp.
func NormalizeState(world, tree map[string][]byte) map[string][]byte {
	minted := mintedOnce(tree, knownDocIDs(world))
	out := make(map[string][]byte, len(tree))
	stamp, day := runStampOf(tree)
	for name, data := range tree {
		if stamp != "" {
			data = bytes.ReplaceAll(data, []byte(stamp), []byte(nowToken))
			name = foldDay(name, day)
			data = []byte(foldDay(string(data), day))
		}
		switch {
		case statsPath.MatchString(name):
			data = foldMTimes(data)
		case path.Base(name) == identitiesBase:
			data = foldMinted(data, minted)
		}
		out[name] = data
	}
	return out
}

// runStampOf is the pass's stamp and its day, or two empty strings when the
// tree holds no stamp of the reference's shape.
func runStampOf(tree map[string][]byte) (string, string) {
	match := runStamp.FindStringSubmatch(strings.TrimSuffix(string(tree[lastRunPath]), "\n"))
	if match == nil {
		return "", ""
	}
	return match[0], match[1]
}

// foldDay puts {{TODAY}} in place of day wherever a case id carries it.
func foldDay(text, day string) string {
	pattern := regexp.MustCompile(`-` + regexp.QuoteMeta(day) + `-([0-9a-f]{4})\b`)
	return pattern.ReplaceAllString(text, "-"+todayToken+"-$1")
}

// foldMTimes replaces the second column of every row that holds a nanosecond
// count there; the header and any other row stay as they are.
func foldMTimes(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i := 1; i < len(lines); i++ {
		fields := strings.Split(lines[i], "\t")
		if len(fields) == 3 && nanoseconds.MatchString(fields[1]) {
			fields[1] = mtimeToken
			lines[i] = strings.Join(fields, "\t")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// foldMinted tokenizes every id of minted, and sorts the rows after the header
// when it tokenized one.
func foldMinted(data []byte, minted map[string]bool) []byte {
	text := string(data)
	body, trailing := strings.CutSuffix(text, "\n")
	lines := strings.Split(body, "\n")
	folded := false
	for i := 1; i < len(lines); i++ {
		id, rest, found := strings.Cut(lines[i], "\t")
		if found && minted[id] {
			lines[i] = docIDToken + "\t" + rest
			folded = true
		}
	}
	if !folded {
		return data
	}
	sort.Strings(lines[1:])
	out := strings.Join(lines, "\n")
	if trailing {
		out += "\n"
	}
	return []byte(out)
}

// mintedOnce is every well-formed id of the tree's registers that known does
// not hold and that stands on exactly one row of them.
func mintedOnce(tree map[string][]byte, known map[string]bool) map[string]bool {
	rows := map[string]int{}
	for name, data := range tree {
		if path.Base(name) != identitiesBase {
			continue
		}
		for _, line := range strings.Split(string(data), "\n")[1:] {
			id, _, found := strings.Cut(line, "\t")
			if found && docID.MatchString(id) && !known[id] {
				rows[id]++
			}
		}
	}
	once := map[string]bool{}
	for id, count := range rows {
		if count == 1 {
			once[id] = true
		}
	}
	return once
}

// knownDocIDs is every id a register of the world already holds.
func knownDocIDs(world map[string][]byte) map[string]bool {
	known := map[string]bool{}
	for name, data := range world {
		if path.Base(name) != identitiesBase {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			id, _, _ := strings.Cut(line, "\t")
			known[id] = true
		}
	}
	return known
}
