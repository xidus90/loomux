package maintenance

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/index"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/brain/vcs"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// statHeader is `_STAT_HEADER` (`reconcile.py:130`), spelt out to the letter
// although no reader ever looks at it: `_read_stats` drops the first line by
// position (`[1:]`) and readStats does the same. What the letters decide is
// the bytes -- both writers compare the whole file against what stands before
// they swap, so a second spelling would make the two tools rewrite the cache
// on every alternating run, and would not match the golden either.
const statHeader = "pfad\tmtime_ns\tsize"

// identitiesName is the register one area's `reindex` leaves behind.
const identitiesName = "_identities.tsv"

// The three reads of one source, as seams. Each of them can only fail if the
// file changes between the walk that listed it and the moment it is read --
// a race no input reaches, and the alternative to a seam is exempting the
// whole of Scan from the coverage rule over three lines. `index/reindex.go`
// keeps `readDocFn` and `replaceDirFn` the same way.
var (
	statFn        = os.Stat
	contentHashFn = identity.ContentHash
	readFileFn    = os.ReadFile
)

// Changed is a source whose content no longer matches the register.
type Changed struct {
	DocID       string
	Relative    string
	Revision    int
	ContentHash string
	Text        string

	// Baseline is the committed version, but only where its hash proved to be
	// the one the register describes. nil means no diff can be honestly drawn.
	//
	// A pointer and not a plain string, because a committed empty file is a
	// baseline that happens to be empty and the package prints a real diff
	// against it, while "there is none" makes it print its no-baseline note
	// and take the whole new file as one hunk (`reconcile.py:899-915`).
	// `vcs.ShowBlob` goes out of its way to keep those two apart; folding them
	// together here would throw that away at the first caller.
	Baseline *string
}

// Scan is stage one and two over one area's registered sources: which of them
// still match the register, and what the ones that do not now say.
//
// Both directories are arguments and neither is read from the environment, for
// the reason `config.ResolvedAreaDir` states -- a lookup that asked StateDir()
// itself would break `internal/serve`'s promise that everything hangs off the
// state directory it was handed.
func Scan(
	area config.Area,
	manifest *config.Manifest,
	stateDir, fallbackDir string,
) (int, int, map[string]Changed, error) {
	register := filepath.Join(config.ResolvedAreaDir(area, stateDir, fallbackDir), identitiesName)
	// Refused, not taken for empty: an unreadable register read as empty would
	// report an area without a single source, which reads as "all is well".
	identities, err := identity.ReadIdentities(register)
	if err != nil {
		return 0, 0, nil, err
	}

	// No nested areas, the way `_scan` calls `find_files` with its default
	// (`reconcile.py:372`), although `reconcile` does hold the whole
	// registration (`reconcile.py:194`). It costs nothing: only keys of the
	// register are ever looked up here, and the register was written by a walk
	// that had already carved the neighbours out -- so a path the wider walk
	// adds is one nothing asks for (`TestScanIgnoresAnUnregisteredFile`).
	files, err := index.FindFiles(area, manifest, nil)
	if err != nil {
		return 0, 0, nil, err
	}
	root := filepath.Clean(area.Path)
	present := make(map[string]string, len(files))
	for _, file := range files {
		// The error is dropped the way `index.FindFiles` drops it over the
		// same two paths: file came out of a walk of root, so the two share a
		// volume and Rel has nothing left to refuse.
		relative, _ := filepath.Rel(root, file)
		present[filepath.ToSlash(relative)] = file
	}

	cache := readStats(statPath(stateDir, area.Scope))
	fresh := map[string]stamp{}
	changed := map[string]Changed{}
	checked := 0
	hashed := 0

	for _, relative := range sortedRegister(identities) {
		entry := identities[relative]
		path, ok := present[relative]
		if !ok {
			// A vanished path is `source_missing`, which the index run's
			// rename detection decides on -- not a change of content.
			continue
		}
		checked++
		info, statErr := statFn(path)
		if statErr != nil {
			return 0, 0, nil, statErr
		}
		now := stamp{mtimeNs: info.ModTime().UnixNano(), size: info.Size()}
		if cache[relative] == now {
			fresh[relative] = now
			continue
		}
		hashed++
		digest, hashErr := contentHashFn(path)
		if hashErr != nil {
			return 0, 0, nil, hashErr
		}
		if digest == entry.ContentHash {
			fresh[relative] = now
			continue
		}
		raw, readErr := readFileFn(path)
		if readErr != nil {
			return 0, 0, nil, readErr
		}
		changed[entry.DocID] = Changed{
			DocID:    entry.DocID,
			Relative: relative,
			Revision: entry.Revision,
			// Decoded through `decode`, never a reader that folds more: both
			// sides of the diff have to be folded by the same rule.
			ContentHash: digest,
			Text:        decode(raw),
			Baseline:    baselineOf(area.Path, relative, entry.ContentHash),
		}
	}

	// Only the fresh are written, and that is the point of the cache rather
	// than an oversight: a changed source left out is hashed again on the next
	// run and reported again, while one carried in would match its own stamp
	// from then on and never be reported a second time.
	if err := writeStats(statPath(stateDir, area.Scope), fresh); err != nil {
		return 0, 0, nil, err
	}
	return checked, hashed, changed, nil
}

// sortedRegister is `sorted(identities.items())`: the order decides which of
// two refusals a reader sees first, and an order that came from a map would
// make that depend on the run.
func sortedRegister(identities map[string]identity.Identity) []string {
	out := make([]string, 0, len(identities))
	for relative := range identities {
		out = append(out, relative)
	}
	sort.Strings(out)
	return out
}

// baselineOf is the previous version of a source, but only once it has proved
// to be one (`_baseline`, `reconcile.py:410-433`).
//
// The package wants a source diff and "what changed, with a citation"; neither
// is deliverable without the older text. Git holds it -- the register keeps a
// hash, never the content, and a shadow copy in the state directory would be a
// second register.
//
// Believing HEAD blindly would be worse than having nothing: a hand commit
// between two approvals, or a history that knows this path in some third
// state, would yield a diff describing a change that never happened. So the
// committed bytes are folded the way `identity.ContentHash` folds a file --
// CRLF to LF -- hashed, and compared against the register entry. Only an exact
// match is a baseline; everything else is nil, and the package then says so out
// loud.
//
// The one error `vcs.ShowBlob` answers, `ErrOutsideRepository`, cannot arrive
// here: relative is a key of the walk's own output and so lies inside the area.
// It is folded into "no baseline" all the same, because an unreadable version
// is an ordinary answer and never a reason to abort a whole reconciliation.
func baselineOf(directory, relative, expected string) *string {
	blob, err := vcs.ShowBlob(directory, relative)
	if err != nil || blob == nil {
		return nil
	}
	normalised := bytes.ReplaceAll(blob, []byte("\r\n"), []byte("\n"))
	if fmt.Sprintf("sha256:%x", sha256.Sum256(normalised)) != expected {
		return nil
	}
	text := decode(blob)
	return &text
}

// decode is bytes as text, folded exactly the way `identity.ContentHash` folds
// them (`_decode`, `reconcile.py:436-449`).
//
// A reader with universal newlines folds something else: it turns a lone CR
// into LF as well, while `ContentHash` leaves it standing on purpose. Decoding
// the two sides of a diff by different rules makes the package claim a change
// to a line nobody touched, and puts git's "no newline" marker in the middle of
// the file where that lone CR sits.
//
// A fabricated change is the worse failure of the two: it is quotable, so the
// evidence check of spec 7 would pass a claim about something that never
// happened -- which is exactly what that check exists to prevent.
//
// `pytext.ReadText` is the reader that cannot be used here: it folds the lone
// CR and refuses bytes that are not UTF-8, where Python's `errors="replace"`
// carries them through as U+FFFD.
func decode(raw []byte) string {
	folded := bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	if utf8.Valid(folded) {
		return string(folded)
	}
	var out strings.Builder
	for i := 0; i < len(folded); {
		if r, size := utf8.DecodeRune(folded[i:]); r != utf8.RuneError || size > 1 {
			out.Write(folded[i : i+size])
			i += size
			continue
		}
		// One U+FFFD per maximal subpart, not per byte: `errors="replace"`
		// consumes the longest prefix that could still have become a
		// character, so a truncated three-byte sequence is one replacement
		// there and would be two under `strings.ToValidUTF8`, which collapses
		// a whole run into one instead.
		out.WriteRune(utf8.RuneError)
		i += maximalSubpart(folded[i:])
	}
	return out.String()
}

// maximalSubpart is the length of the ill-formed sequence at the front of raw:
// the longest prefix that is still a prefix of some well-formed sequence, and
// at least one byte (Unicode 16.0, table 3-7).
//
// Not read off the standard and hoped for: all sixteen byte strings the tests
// of this file claim an answer for were put to CPython 3.14 on 2026-09-20
// through `raw.decode("utf-8", errors="replace")`, and every one of them
// answered what is claimed here.
func maximalSubpart(raw []byte) int {
	first := raw[0]
	var total int
	var low, high byte
	switch {
	case first >= 0xC2 && first <= 0xDF:
		total, low, high = 2, 0x80, 0xBF
	case first == 0xE0:
		total, low, high = 3, 0xA0, 0xBF
	case first >= 0xE1 && first <= 0xEC:
		total, low, high = 3, 0x80, 0xBF
	case first == 0xED:
		total, low, high = 3, 0x80, 0x9F
	case first >= 0xEE && first <= 0xEF:
		total, low, high = 3, 0x80, 0xBF
	case first == 0xF0:
		total, low, high = 4, 0x90, 0xBF
	case first >= 0xF1 && first <= 0xF3:
		total, low, high = 4, 0x80, 0xBF
	case first == 0xF4:
		total, low, high = 4, 0x80, 0x8F
	default:
		// A continuation byte on its own, an overlong lead (C0, C1) or a lead
		// beyond the range Unicode still has characters in (F5 and up): none
		// of them is a prefix of anything well-formed.
		return 1
	}
	if len(raw) < 2 || raw[1] < low || raw[1] > high {
		return 1
	}
	for i := 2; i < total; i++ {
		if len(raw) <= i || raw[i] < 0x80 || raw[i] > 0xBF {
			return i
		}
	}
	return total
}

// stamp is one source's last clean reading: change either half and the file is
// hashed again.
type stamp struct {
	mtimeNs int64
	size    int64
}

// statPath is `_stat_path` (`reconcile.py:991-993`), written out here rather
// than taken from the area's artefact directory: the cache is this layer's own
// and belongs to the state directory even for an area that keeps everything
// else in its own tree.
func statPath(stateDir, scope string) string {
	return filepath.Join(stateDir, "maintenance", search.CollectionName(scope), "stats.tsv")
}

// readStats is the last clean reading per path. A damaged line is dropped, not
// fatal, and a file that cannot be read at all is an empty cache.
//
// The file is a throwaway cache, so the worst a bad line can cost is one extra
// hash -- refusing over it would take the whole run down for a file the run
// could simply rebuild (`_read_stats`, `reconcile.py:996-1001`).
//
// Two places where that costs a hash rather than the run, and both differ from
// the reference. Bytes that are not UTF-8 are **not** checked here: they travel
// into the key, which then matches no real path, so the line is spent and the
// good lines around it still work -- where `read_text` raises and takes the
// whole run down. And a file that exists but cannot be opened answers an empty
// cache, where Python's `read_text` raises the `PermissionError` on.
func readStats(path string) map[string]stamp {
	stats := map[string]stamp{}
	data, err := os.ReadFile(path)
	if err != nil {
		return stats
	}
	for i, line := range pytext.SplitLines(string(data)) {
		if i == 0 {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			continue
		}
		mtime, ok := parseStamp(fields[1])
		if !ok {
			continue
		}
		size, ok := parseStamp(fields[2])
		if !ok {
			continue
		}
		stats[fields[0]] = stamp{mtimeNs: mtime, size: size}
	}
	return stats
}

// parseStamp is `isdigit()` and `int()` in one: digits only, so a sign is no
// number here, and a value this side cannot hold is a damaged line rather than
// a wrapped stamp that could match a real file by accident. An empty field
// needs no guard of its own -- it has no non-digit to fail on and falls to
// ParseInt, which refuses it, exactly as `"".isdigit()` is False.
func parseStamp(field string) (int64, bool) {
	for i := 0; i < len(field); i++ {
		if field[i] < '0' || field[i] > '9' {
			return 0, false
		}
	}
	value, err := strconv.ParseInt(field, 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// writeStats puts the cache down the way `_write_stats` does, through the
// write-on-change of `write_if_changed`: an untouched cache keeps its own
// timestamps, and the swap is the one `lock.ReplaceText` performs.
func writeStats(path string, stats map[string]stamp) error {
	text := renderStats(stats)
	// os.ReadFile and not pytext.ReadText, for the reason WriteCase states: a
	// standing file whose bytes differ only in their line endings would
	// compare equal and never be rewritten.
	if standing, err := os.ReadFile(path); err == nil && string(standing) == text {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return lock.ReplaceText(path, text)
}

// renderStats is the file's bytes: the header, then one line per path in code
// point order, then a closing newline.
func renderStats(stats map[string]stamp) string {
	lines := make([]string, 0, len(stats)+1)
	lines = append(lines, statHeader)
	relatives := make([]string, 0, len(stats))
	for relative := range stats {
		relatives = append(relatives, relative)
	}
	// sort.Strings orders by byte, `sorted()` by code point; for UTF-8 the two
	// are the same order, which is what lets a golden file pin it.
	sort.Strings(relatives)
	for _, relative := range relatives {
		entry := stats[relative]
		lines = append(lines, relative+"\t"+
			strconv.FormatInt(entry.mtimeNs, 10)+"\t"+
			strconv.FormatInt(entry.size, 10))
	}
	return strings.Join(lines, "\n") + "\n"
}
