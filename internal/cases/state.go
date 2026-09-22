package cases

import (
	"bytes"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
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
	userToken  = "{{USER}}"
	shaToken   = "{{SHA}}"
)

const (
	lastRunPath    = "maintenance/last-run.txt"
	identitiesBase = "_identities.tsv"
	auditBase      = "audit.md"
	logBase        = "log.md"
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

// auditHeader is the head of a block `apply._append_audit` writes,
// "## <now> — <target> (Fall `<id>`)"; the first group is the stamp. It and the
// three patterns below compile on first use, which keeps the package's init
// inside the allocation budget cmd/loomux holds every start to.
var auditHeader = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile("^## (\\S+) — .+ \\(Fall `[^`]*`\\)$") })

// reviewer is the one shape of a reviewer `cli._reviewer` hands on: the account
// running the command behind `human:`. A quote ends it, for the spelling YAML
// quotes.
var reviewer = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`human:[^\s'"]+`) })

// verifiedBy is a `by:` line of a `verified` entry, as PyYAML dumps it.
var verifiedBy = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?m)^([ \t-]*by: ['"]?)human:[^\s'"]+`) })

// commitLine is the line `approve` reports its commit on, with the full SHA git
// printed; any other length is not one git printed.
var commitLine = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?m)^committet als [0-9a-f]{40}$`) })

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
// And the four an approval of stage 3b writes:
//
//   - The reviewer. `human:<account>` becomes `human:{{USER}}` in every
//     `audit.md` and on the `by:` lines of a page's frontmatter; a reviewer of
//     any other shape stays.
//   - The approval's `now`. It writes no last-run.txt; its stamp is the one in
//     the header of an audit block the world's `audit.md` did not hold. Where
//     the tree holds exactly one such stamp of the reference's shape, and the
//     world holds it nowhere, it becomes {{NOW}} in every file that differs from
//     the world -- the new block, the page's `generated.at` and `verified[].at`,
//     quoted or not. Older blocks cannot carry it and stay byte-equal.
//   - That stamp's day, in the new line of `log.md`, `- <day> — …`, and only
//     there.
//   - The commit, on stdout: `committet als <40 hex digits>` becomes
//     `committet als {{SHA}}`.
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
	foldApproval(world, tree, out)
	return out
}

// foldApproval applies the four folds of an approval to out, the tree after
// the folds of a pass. It decides from the tree as the run left it: what is
// new is what the world did not hold.
func foldApproval(world, tree, out map[string][]byte) {
	approved := approvalStampOf(world, tree)
	for name, data := range out {
		base := path.Base(name)
		if approved != "" && !bytes.Equal(tree[name], world[name]) {
			data = bytes.ReplaceAll(data, []byte(approved), []byte(nowToken))
			if base == logBase {
				data = foldLogDay(data, world[name], approved[:len("2006-01-02")])
			}
		}
		switch {
		case name == stdoutKey:
			data = commitLine().ReplaceAll(data, []byte("committet als "+shaToken))
		case base == auditBase:
			data = reviewer().ReplaceAll(data, []byte("human:"+userToken))
		case base != logBase && path.Ext(base) == ".md":
			data = foldFrontmatterReviewer(data)
		}
		out[name] = data
	}
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

// approvalStampOf is the stamp of the one run whose audit block is new, or ""
// when there is none, more than one, one of another shape, or one the world
// already spells somewhere.
func approvalStampOf(world, tree map[string][]byte) string {
	found := ""
	for name, data := range tree {
		if path.Base(name) != auditBase {
			continue
		}
		old := lineSet(world[name])
		for _, line := range strings.Split(string(data), "\n") {
			match := auditHeader().FindStringSubmatch(line)
			if match == nil || old[line] {
				continue
			}
			if found != "" && found != match[1] {
				return ""
			}
			found = match[1]
		}
	}
	if !runStamp.MatchString(found) {
		return ""
	}
	for _, data := range world {
		if bytes.Contains(data, []byte(found)) {
			return ""
		}
	}
	return found
}

// foldLogDay puts {{TODAY}} in place of day at the head of every log line of
// data that old does not hold.
func foldLogDay(data, old []byte, day string) []byte {
	known := lineSet(old)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if rest, ok := strings.CutPrefix(line, "- "+day+" — "); ok && !known[line] {
			lines[i] = "- " + todayToken + " — " + rest
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// foldFrontmatterReviewer tokenizes the reviewer on the `by:` lines of the
// frontmatter of a page; the body stays as it is.
func foldFrontmatterReviewer(data []byte) []byte {
	text := string(data)
	rest, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return data
	}
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return data
	}
	head := verifiedBy().ReplaceAllString(rest[:end], "${1}human:"+userToken)
	return []byte("---\n" + head + rest[end:])
}

// lineSet is the set of data's lines.
func lineSet(data []byte) map[string]bool {
	set := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		set[line] = true
	}
	return set
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
