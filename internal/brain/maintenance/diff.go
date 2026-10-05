package maintenance

// The source diff of a review case: Python's `difflib` as far as
// `unified_diff` reaches, and no further.
//
// A hand-rolled diff would have been the cheaper answer and the wrong one.
// The hunks rendered here become the `D` segments of `package.md`, a proposal
// may only claim what it can quote out of one of them verbatim, and the
// evidence check of stage 3b looks the quote up by the segment's number. So
// the boundary between two hunks and the bytes inside one are an interface,
// not a matter of taste -- and `difflib.SequenceMatcher` does not compute a
// longest common subsequence. It takes the longest matching *block* and
// recurses on both sides of it, which for the same input yields different
// boundaries than an LCS, and from 200 lines up it additionally drops
// "popular" lines from the index (`autojunk`), which moves them again.
//
// The port is therefore line for line, and it is pinned: testdata/hunks.golden.json
// came out of the reference's own `_hunks`; the script that wrote it is in the
// archive release archive/parity-recordings.
//
// What is deliberately **not** ported: `isjunk`. `unified_diff` builds its
// matcher with `SequenceMatcher(None, a, b)`, so `bjunk` is empty for every
// call this package makes, and the four junk-extension loops of
// `find_longest_match` could never take a step. Writing them would be four
// branches no input reaches. `autojunk` is a different matter and is ported:
// it is on by default and a 200-line source is ordinary.

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// pyLines is `_lines` (`reconcile.py:869-885`): split at `\n` and nowhere
// else, keeping the separator.
//
// Not pytext.SplitLines, and the docstring there says why: it breaks on a
// lone `\r` too, so a source carrying one would gain a "line" ending without
// `\n` -- and the no-newline marker would then land in the middle of the
// file. Only a real `\n` ends a line here, which is the boundary
// `identity.ContentHash` and `decode` use as well.
func pyLines(text string) []string {
	parts := strings.Split(text, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts[:len(parts)-1] {
		lines = append(lines, part+"\n")
	}
	if last := parts[len(parts)-1]; last != "" {
		// No trailing newline: the remainder is a line of its own, and the
		// only one the marker may ever apply to.
		lines = append(lines, last)
	}
	return lines
}

// opcode is one `(tag, i1, i2, j1, j2)` of `SequenceMatcher.get_opcodes`.
type opcode struct {
	tag            string
	i1, i2, j1, j2 int
}

// match is one `(a index, b index, size)` of `get_matching_blocks`.
type match struct{ ai, bj, size int }

// matcher is `difflib.SequenceMatcher` over two slices of lines, with
// `isjunk` fixed at None and `autojunk` at its default.
type matcher struct {
	a, b []string
	b2j  map[string][]int
}

// newMatcher builds the reverse index `__chain_b` builds, popular lines
// included.
//
// `autojunk` is the half of `__chain_b` that has teeth here: from 200 lines
// of b upwards, every line occurring more than `len(b)/100 + 1` times is
// struck from the index, so `find_longest_match` can no longer anchor on it.
// A file of a few hundred lines with a run of blank ones reaches that, and
// without the rule the hunks of such a file come out elsewhere.
//
// The struck lines are **not** junk in the sense the extension loops mean:
// `bjunk` stays empty, and a struck line still extends a match found beside
// it. Folding the two sets together would be the easy mistake here.
func newMatcher(a, b []string) *matcher {
	b2j := map[string][]int{}
	for j, line := range b {
		b2j[line] = append(b2j[line], j)
	}
	if len(b) >= 200 {
		ntest := len(b)/100 + 1
		for line, indices := range b2j {
			if len(indices) > ntest {
				delete(b2j, line)
			}
		}
	}
	return &matcher{a: a, b: b, b2j: b2j}
}

// findLongestMatch is `find_longest_match` without its junk arms: the longest
// block of equal lines in `a[alo:ahi]` against `b[blo:bhi]`, earliest in a
// and, among those, earliest in b.
func (m *matcher) findLongestMatch(alo, ahi, blo, bhi int) match {
	besti, bestj, bestsize := alo, blo, 0
	j2len := map[int]int{}
	for i := alo; i < ahi; i++ {
		newj2len := map[int]int{}
		for _, j := range m.b2j[m.a[i]] {
			if j < blo {
				continue
			}
			if j >= bhi {
				break
			}
			k := j2len[j-1] + 1
			newj2len[j] = k
			if k > bestsize {
				besti, bestj, bestsize = i-k+1, j-k+1, k
			}
		}
		j2len = newj2len
	}
	// The block is grown outwards over lines the index no longer holds --
	// see newMatcher on why a struck line still counts here.
	for besti > alo && bestj > blo && m.a[besti-1] == m.b[bestj-1] {
		besti, bestj, bestsize = besti-1, bestj-1, bestsize+1
	}
	for besti+bestsize < ahi && bestj+bestsize < bhi && m.a[besti+bestsize] == m.b[bestj+bestsize] {
		bestsize++
	}
	return match{ai: besti, bj: bestj, size: bestsize}
}

// matchingBlocks is `get_matching_blocks`: the longest block, then the same
// question again to its left and to its right, sorted and with adjacent
// blocks collapsed, closed by the empty sentinel at the end of both slices.
func (m *matcher) matchingBlocks() []match {
	queue := [][4]int{{0, len(m.a), 0, len(m.b)}}
	var blocks []match
	for len(queue) > 0 {
		box := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		found := m.findLongestMatch(box[0], box[1], box[2], box[3])
		if found.size == 0 {
			continue
		}
		blocks = append(blocks, found)
		if box[0] < found.ai && box[2] < found.bj {
			queue = append(queue, [4]int{box[0], found.ai, box[2], found.bj})
		}
		if found.ai+found.size < box[1] && found.bj+found.size < box[3] {
			queue = append(queue, [4]int{found.ai + found.size, box[1], found.bj + found.size, box[3]})
		}
	}
	// By the index in a alone, where Python sorts the whole triple. The two
	// orders cannot disagree: every box is split into one strictly to the left
	// of the block and one strictly to its right, in a *and* in b, so no two
	// blocks ever begin at the same index in a and the remaining two members
	// of the triple never get a say.
	slices.SortFunc(blocks, func(x, y match) int { return x.ai - y.ai })
	var collapsed []match
	var run match
	for _, block := range blocks {
		if run.ai+run.size == block.ai && run.bj+run.size == block.bj {
			run.size += block.size
			continue
		}
		if run.size != 0 {
			collapsed = append(collapsed, run)
		}
		run = block
	}
	if run.size != 0 {
		collapsed = append(collapsed, run)
	}
	return append(collapsed, match{ai: len(m.a), bj: len(m.b)})
}

// opcodes is `get_opcodes`: the blocks turned into the five tags, with the
// gaps between them named.
func (m *matcher) opcodes() []opcode {
	var out []opcode
	i, j := 0, 0
	for _, block := range m.matchingBlocks() {
		tag := ""
		switch {
		case i < block.ai && j < block.bj:
			tag = "replace"
		case i < block.ai:
			tag = "delete"
		case j < block.bj:
			tag = "insert"
		}
		if tag != "" {
			out = append(out, opcode{tag: tag, i1: i, i2: block.ai, j1: j, j2: block.bj})
		}
		i, j = block.ai+block.size, block.bj+block.size
		if block.size != 0 {
			out = append(out, opcode{tag: "equal", i1: block.ai, i2: i, j1: block.bj, j2: j})
		}
	}
	return out
}

// groupedOpcodes is `get_grouped_opcodes`: the opcodes cut into groups of up
// to n lines of context each, which is what makes two distant changes two
// hunks instead of one.
func (m *matcher) groupedOpcodes(n int) [][]opcode {
	codes := m.opcodes()
	if len(codes) == 0 {
		// Both sides empty. Python invents one equal opcode here so the
		// generator has something to walk; the group it builds is a lone
		// `equal` and is dropped again at the end, so the answer stays empty
		// -- but without it the two fixups below would index an empty list.
		codes = []opcode{{tag: "equal", i1: 0, i2: 1, j1: 0, j2: 1}}
	}
	if first := codes[0]; first.tag == "equal" {
		codes[0] = opcode{tag: "equal", i1: max(first.i1, first.i2-n), i2: first.i2,
			j1: max(first.j1, first.j2-n), j2: first.j2}
	}
	if last := codes[len(codes)-1]; last.tag == "equal" {
		codes[len(codes)-1] = opcode{tag: "equal", i1: last.i1, i2: min(last.i2, last.i1+n),
			j1: last.j1, j2: min(last.j2, last.j1+n)}
	}
	var groups [][]opcode
	var group []opcode
	for _, code := range codes {
		// A long unchanged stretch ends the group: n lines of it close the
		// hunk, n more open the next one, and everything between is left out.
		if code.tag == "equal" && code.i2-code.i1 > n+n {
			group = append(group, opcode{tag: "equal", i1: code.i1, i2: min(code.i2, code.i1+n),
				j1: code.j1, j2: min(code.j2, code.j1+n)})
			groups = append(groups, group)
			group = nil
			code = opcode{tag: "equal", i1: max(code.i1, code.i2-n), i2: code.i2,
				j1: max(code.j1, code.j2-n), j2: code.j2}
		}
		group = append(group, code)
	}
	if len(group) > 0 && !(len(group) == 1 && group[0].tag == "equal") {
		groups = append(groups, group)
	}
	return groups
}

// unifiedDiff is `difflib.unified_diff` with its defaults: three lines of
// context, `\n` as the line terminator and no file dates. Every line carries
// its own terminator, exactly as the generator yields them.
func unifiedDiff(a, b []string, fromFile, toFile string, n int) []string {
	var out []string
	for _, group := range newMatcher(a, b).groupedOpcodes(n) {
		if len(out) == 0 {
			out = append(out, "--- "+fromFile+"\n", "+++ "+toFile+"\n")
		}
		first, last := group[0], group[len(group)-1]
		out = append(out, fmt.Sprintf("@@ -%s +%s @@\n",
			formatRangeUnified(first.i1, last.i2), formatRangeUnified(first.j1, last.j2)))
		for _, code := range group {
			if code.tag == "equal" {
				for _, line := range a[code.i1:code.i2] {
					out = append(out, " "+line)
				}
				continue
			}
			if code.tag == "replace" || code.tag == "delete" {
				for _, line := range a[code.i1:code.i2] {
					out = append(out, "-"+line)
				}
			}
			if code.tag == "replace" || code.tag == "insert" {
				for _, line := range b[code.j1:code.j2] {
					out = append(out, "+"+line)
				}
			}
		}
	}
	return out
}

// formatRangeUnified is `_format_range_unified`: one number for a range of
// one line, `start,length` otherwise -- and an empty range begins at the line
// before it, which is what makes a new file read `-0,0`.
func formatRangeUnified(start, stop int) string {
	beginning := start + 1
	length := stop - start
	if length == 1 {
		return strconv.Itoa(beginning)
	}
	if length == 0 {
		beginning--
	}
	return strconv.Itoa(beginning) + "," + strconv.Itoa(length)
}

// noBaseline and noNewline are `_NO_BASELINE` and `_NO_NEWLINE`. Both go into
// `package.md` verbatim, so both stay spelt the way the reference spells them
// -- the German line included (AGENTS.md:28 exempts text that has to match the
// reference byte for byte). It is read by a person and by the reviewing model,
// never by code.
const (
	noBaseline = "# ohne verifizierten Vorzustand: das Paket führt nur den neuen Stand"
	noNewline  = `\ No newline at end of file`
)

// hunksOf is `_hunks`: one changed source as unified-diff hunks, each one
// carrying the file header so it reads on its own.
//
// Where no baseline could be verified, the single hunk is the whole new file
// and noBaseline stands above it: the reviewer must not be left to assume a
// previous state the package cannot show.
func hunksOf(item Changed) []string {
	var before []string
	lead := "/dev/null"
	if item.Baseline != nil {
		before = pyLines(*item.Baseline)
		lead = item.Relative + " (HEAD)"
	}
	var lines []string
	for _, line := range unifiedDiff(before, pyLines(item.Text), lead, item.Relative, 3) {
		if strings.HasSuffix(line, "\n") {
			lines = append(lines, line)
			continue
		}
		// difflib never emits git's no-newline marker, so a file without a
		// trailing newline would read as one that has it -- and a proposal
		// quoting that last line verbatim would never match the source again.
		lines = append(lines, line+"\n"+noNewline+"\n")
	}
	if len(lines) == 0 {
		// Nothing moved: there is no header to take. Python slices `lines[:2]`
		// off the empty list and splits the empty string, which yields the
		// same nothing; Go would index past the end.
		return nil
	}
	header := lines[0] + lines[1]
	if item.Baseline == nil {
		header = noBaseline + "\n" + header
	}
	var out []string
	for _, part := range splitAtHunkHeads(strings.Join(lines[2:], "")) {
		// TrimSuffix and not a trim of all whitespace: the closing fence of
		// the package brings its own line break, while a diff line of "+"
		// alone -- an added empty line -- must keep the one belonging to it.
		out = append(out, strings.TrimSuffix(header+part, "\n"))
	}
	return out
}

// splitAtHunkHeads cuts text before every line that opens a hunk, which is
// `_HUNK = re.compile(r"^(?=@@ )", re.MULTILINE)` and its `split`.
//
// A loop over the line starts and not a regexp: RE2 has no lookahead, and a
// pattern that consumed the `@@ ` would eat the very bytes the hunk needs.
// The cut is safe against a source line that itself begins with `@@`, because
// every line of a diff body arrives prefixed with a space, a `+` or a `-`.
//
// Python's split puts an empty piece in front of the first match and `if part`
// drops it again; nothing empty is produced here, so there is nothing to drop.
func splitAtHunkHeads(text string) []string {
	var parts []string
	last := 0
	for at := 0; at < len(text); {
		if at > last && strings.HasPrefix(text[at:], "@@ ") {
			parts = append(parts, text[last:at])
			last = at
		}
		newline := strings.IndexByte(text[at:], '\n')
		if newline < 0 {
			break
		}
		at += newline + 1
	}
	if last < len(text) {
		parts = append(parts, text[last:])
	}
	return parts
}
