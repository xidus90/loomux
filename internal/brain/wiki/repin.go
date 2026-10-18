package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/brain/identity"
)

// pinValue is what repin writes into one sources[] entry.
type pinValue struct {
	hash     string
	revision int
}

var (
	sourcesKey = regexp.MustCompile(`^sources:(.*)$`)
	entryStart = regexp.MustCompile(`^( *)- `)
	entryField = regexp.MustCompile(`^( *(?:- )?)(doc_id|content_hash|revision):(\s*)(.*)$`)
	flowKey    = regexp.MustCompile(`^\s*(doc_id|content_hash|revision):`)
)

// repinText sets content_hash and revision of every sources[] entry whose
// doc_id has an update and leaves every other byte alone: a YAML round trip
// would refold the description and requote the hashes of every page it
// touches. What it cannot edit that way it refuses.
func repinText(text string, updates map[string]pinValue) (string, error) {
	lines := strings.SplitAfter(text, "\n")
	if body(lines[0]) != "---" {
		return "", fmt.Errorf("no frontmatter")
	}
	end := 1
	for end < len(lines) && body(lines[end]) != "---" {
		end++
	}
	if end == len(lines) {
		return "", fmt.Errorf("frontmatter is not closed")
	}
	start := -1
	for i := 1; i < end; i++ {
		if m := sourcesKey.FindStringSubmatch(body(lines[i])); m != nil {
			if strings.TrimSpace(m[1]) != "" {
				return "", fmt.Errorf("sources is not a block list")
			}
			start = i + 1
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("no sources list")
	}
	stop := start
	for stop < end && (strings.HasPrefix(lines[stop], " ") || strings.HasPrefix(lines[stop], "- ") || strings.TrimSpace(lines[stop]) == "") {
		stop++
	}
	entryIndent := -1
	seen := map[string]bool{}
	fields := map[string]int{}
	flush := func() error {
		id, ok := fields["doc_id"]
		if !ok {
			return nil
		}
		docID := unquote(value(lines[id]))
		update, wanted := updates[docID]
		if !wanted {
			return nil
		}
		// A doc_id that stands twice cites one file twice: both entries
		// get the same values.
		seen[docID] = true
		for _, key := range []string{"content_hash", "revision"} {
			at, ok := fields[key]
			if !ok {
				return fmt.Errorf("the entry of doc_id %s has no %s line", docID, key)
			}
			next := update.hash
			if key == "revision" {
				next = strconv.Itoa(update.revision)
			}
			line, err := setValue(lines[at], lines[at+1:], next)
			if err != nil {
				return fmt.Errorf("the entry of doc_id %s: %w", docID, err)
			}
			lines[at] = line
		}
		return nil
	}
	for i := start; i < stop; i++ {
		line := body(lines[i])
		if m := entryStart.FindStringSubmatch(line); m != nil && (entryIndent < 0 || len(m[1]) == entryIndent) {
			if err := flush(); err != nil {
				return "", err
			}
			entryIndent = len(m[1])
			fields = map[string]int{}
			if strings.HasPrefix(line[len(m[0]):], "{") {
				last, err := repinFlow(lines, i, stop, len(m[0]), updates, seen)
				if err != nil {
					return "", err
				}
				i = last
				continue
			}
		}
		if m := entryField.FindStringSubmatch(line); m != nil && entryIndent >= 0 && len(m[1]) == entryIndent+2 {
			fields[m[2]] = i
		}
	}
	if err := flush(); err != nil {
		return "", err
	}
	for docID := range updates {
		if !seen[docID] {
			return "", fmt.Errorf("no sources entry with doc_id %s", docID)
		}
	}
	return strings.Join(lines, ""), nil
}

// repinFlow edits the sources entry written as a flow mapping that opens at
// column open of lines[first], however many of the lines up to stop it
// spans, and returns the index of the line it closes on. The closed entry
// is put back into lines[first] and the lines it consumed are emptied, so
// that the join of lines is the page with only the two values changed.
func repinFlow(lines []string, first, stop, open int, updates map[string]pinValue, seen map[string]bool) (int, error) {
	text := strings.Join(lines[first:stop], "")
	fields, end, err := flowFields(text, open)
	if err != nil {
		return 0, err
	}
	last := first
	for consumed := len(lines[first]); consumed <= end; consumed += len(lines[last]) {
		last++
	}
	entry := text[:len(text)-len(strings.Join(lines[last+1:stop], ""))]
	id, ok := fields["doc_id"]
	if ok {
		docID := unquote(text[id[0]:id[1]])
		if update, wanted := updates[docID]; wanted {
			seen[docID] = true
			at := map[string][2]int{}
			for _, key := range []string{"content_hash", "revision"} {
				span, ok := fields[key]
				if !ok {
					return 0, fmt.Errorf("the entry of doc_id %s has no %s", docID, key)
				}
				at[key] = span
			}
			// The later value first, so that the offset of the earlier one holds.
			keys := []string{"content_hash", "revision"}
			if at["revision"][0] < at["content_hash"][0] {
				keys[0], keys[1] = keys[1], keys[0]
			}
			for _, key := range slices.Backward(keys) {
				next := update.hash
				if key == "revision" {
					next = strconv.Itoa(update.revision)
				}
				span := at[key]
				old := entry[span[0]:span[1]]
				if q := old[0]; q == '"' || q == '\'' {
					next = string(q) + next + string(q)
				}
				entry = entry[:span[0]] + next + entry[span[1]:]
			}
		}
	}
	lines[first] = entry
	for i := first + 1; i <= last; i++ {
		lines[i] = ""
	}
	return last, nil
}

// flowFields reads the flow mapping whose `{` stands at text[open] and
// returns the value spans of its own doc_id, content_hash and revision and
// the offset of its closing `}`. Nested mappings and lists are read through,
// not into. A value it could not rewrite as one scalar on one line, a
// comment and a mapping that does not close are refused.
func flowFields(text string, open int) (map[string][2]int, int, error) {
	fields := map[string][2]int{}
	item := open + 1
	depth := 0
	take := func(to int) error {
		m := flowKey.FindStringSubmatchIndex(text[item:to])
		if m == nil {
			return nil
		}
		key := text[item+m[2] : item+m[3]]
		from, until := item+m[1], to
		for from < until && strings.ContainsRune(" \t\r\n", rune(text[from])) {
			from++
		}
		for until > from && strings.ContainsRune(" \t\r\n", rune(text[until-1])) {
			until--
		}
		v := text[from:until]
		switch {
		case v == "" || strings.Contains(v, "\n"):
			return fmt.Errorf("%s is not a one-line value", key)
		case (v[0] == '"' || v[0] == '\'') && (len(v) < 2 || v[len(v)-1] != v[0]):
			return fmt.Errorf("%s is not one quoted value", key)
		}
		fields[key] = [2]int{from, until}
		return nil
	}
	for i := open + 1; i < len(text); i++ {
		switch c := text[i]; c {
		case '"', '\'':
			j := i + 1
			for ; j < len(text); j++ {
				if c == '"' && text[j] == '\\' {
					j++
				} else if text[j] == c {
					if c == '\'' && j+1 < len(text) && text[j+1] == '\'' {
						j++
						continue
					}
					break
				}
			}
			if j >= len(text) {
				return nil, 0, fmt.Errorf("a quoted value in a flow entry is not closed")
			}
			i = j
		case '#':
			if strings.ContainsRune(" \t\r\n", rune(text[i-1])) {
				return nil, 0, fmt.Errorf("a flow entry carries a comment")
			}
		case '{', '[':
			depth++
		case '}', ']':
			if depth > 0 {
				depth--
				continue
			}
			if err := take(i); err != nil {
				return nil, 0, err
			}
			return fields, i, nil
		case ',':
			if depth == 0 {
				if err := take(i); err != nil {
					return nil, 0, err
				}
				item = i + 1
			}
		}
	}
	return nil, 0, fmt.Errorf("a flow entry is not closed")
}

// body is a line without its line end.
func body(line string) string {
	return strings.TrimRight(line, "\r\n")
}

// value is the raw value of a `key: value` line.
func value(line string) string {
	return entryField.FindStringSubmatch(body(line))[4]
}

func unquote(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}

// setValue writes next into a field line in the quotes the old value used,
// keeping the indent, the spacing after the colon and the line end. A value
// it cannot read as one plain or quoted scalar on this line is refused; so is
// a plain one that the following lines continue (they are indented deeper
// than the key, blank lines in between do not end it). following always ends
// in the closing fence, which is neither blank nor indented.
func setValue(line string, following []string, next string) (string, error) {
	m := entryField.FindStringSubmatch(body(line))
	old := strings.TrimRight(m[4], " \t")
	switch {
	case old == "" || old[0] == '|' || old[0] == '>':
		return "", fmt.Errorf("%s is not a one-line value", m[2])
	case strings.Contains(old, " #"):
		return "", fmt.Errorf("%s carries a comment", m[2])
	case old[0] == '"' || old[0] == '\'':
		if len(old) < 2 || old[len(old)-1] != old[0] {
			return "", fmt.Errorf("%s is not a one-line value", m[2])
		}
		next = string(old[0]) + next + string(old[0])
	default:
		for _, f := range following {
			if strings.TrimSpace(f) == "" {
				continue
			}
			if len(f)-len(strings.TrimLeft(f, " ")) > len(m[1]) {
				return "", fmt.Errorf("%s continues on the next line", m[2])
			}
			break
		}
	}
	return m[1] + m[2] + ":" + m[3] + next + line[len(body(line)):], nil
}

// RowAdvance is one register row repin moved on.
type RowAdvance struct {
	Path     string
	From, To int
}

// RepinResult is what repin wrote: the rows moved on and the pages pinned
// anew (wiki-relative).
type RepinResult struct {
	Rows  []RowAdvance
	Pages []string
}

// RepinRefusal is repin's answer to findings it may not advance: a row it
// cannot mint, a file that is gone, a doc_id that does not match, a cycle.
type RepinRefusal struct {
	Findings []SourceFinding
}

func (r *RepinRefusal) Error() string {
	return fmt.Sprintf("%d finding(s) repin does not fix", len(r.Findings))
}

// writeSource writes pages and the register; a test replaces it to reach a
// failed write.
var writeSource = os.WriteFile

// Repin moves the register rows of the cited files and the pins of the
// project's own scope to the files as they are, in the tree at projectRoot:
// the branch's own register, not the registry's checkout. Whoever runs it
// declares the citing pages still correct. A row moves by one at most in a
// run, measured against the register as it was read, and a row nobody cites
// is left as it is. It refuses before any write what it may not fix.
func Repin(projectRoot, wikiRoot, scope string) (RepinResult, error) {
	world, err := loadSources(projectRoot, wikiRoot, scope)
	if err != nil || world == nil {
		return RepinResult{}, err
	}
	findings, err := world.drift()
	if err != nil {
		return RepinResult{}, err
	}
	var blocking []SourceFinding
	for _, f := range findings {
		if f.Rule != RuleRegisterStale && f.Rule != RulePinStale {
			blocking = append(blocking, f)
		}
	}
	if len(blocking) > 0 {
		return RepinResult{}, &RepinRefusal{Findings: blocking}
	}
	plan := &repinPlan{world: world, byPath: map[string]*sourcePage{}, text: map[string]string{}, hash: map[string]string{}}
	for _, p := range world.pages {
		plan.byPath[p.repoPath] = p
	}
	for _, p := range world.pages {
		if _, err := plan.final(p); err != nil {
			return RepinResult{}, err
		}
	}
	return plan.write()
}

// repinPlan computes every page's final text and every cited file's final
// hash in memory; cycles were refused, so the recursion ends.
type repinPlan struct {
	world  *sourceWorld
	byPath map[string]*sourcePage
	text   map[string]string // repoPath -> final text
	hash   map[string]string // repoPath -> final hash
}

func (r *repinPlan) final(p *sourcePage) (string, error) {
	if text, done := r.text[p.repoPath]; done {
		return text, nil
	}
	text := string(p.raw)
	updates := map[string]pinValue{}
	for _, pn := range p.pins {
		digest, err := r.finalHash(pn.path)
		if err != nil {
			return "", err
		}
		rev := r.finalRevision(pn.path, digest)
		// A pin that is current is not edited, so that a page the check
		// never named is not refused for the form of its sources block.
		if pn.source.ContentHash != digest || *pn.source.Revision != rev {
			updates[pn.source.DocID] = pinValue{hash: digest, revision: rev}
		}
	}
	if len(updates) > 0 {
		var err error
		// The page that cannot be edited is the one to name, however it was
		// reached: through its own turn or through a page that cites it.
		if text, err = repinText(text, updates); err != nil {
			return "", fmt.Errorf("%s: %w", p.relative, err)
		}
	}
	r.text[p.repoPath] = text
	return text, nil
}

func (r *repinPlan) finalHash(path string) (string, error) {
	if digest, done := r.hash[path]; done {
		return digest, nil
	}
	var digest string
	if p, isPage := r.byPath[path]; isPage {
		text, err := r.final(p)
		if err != nil {
			return "", err
		}
		digest = identity.HashBytes([]byte(text))
	} else {
		var err error
		if digest, err = r.world.fileHash(path); err != nil {
			return "", err
		}
	}
	r.hash[path] = digest
	return digest, nil
}

func (r *repinPlan) finalRevision(path, digest string) int {
	row := r.world.register[path]
	if digest != row.ContentHash {
		return row.Revision + 1
	}
	return row.Revision
}

// write writes the changed pages first and the register last, so that a
// failure leaves a register that still names the old state of what it
// could not reach.
func (r *repinPlan) write() (RepinResult, error) {
	var result RepinResult
	for _, p := range r.world.pages {
		if r.text[p.repoPath] == string(p.raw) {
			continue
		}
		if err := writeSource(p.path, []byte(r.text[p.repoPath]), 0o644); err != nil {
			return result, fmt.Errorf("writing %s: %w", p.relative, err)
		}
		result.Pages = append(result.Pages, p.relative)
	}
	register := map[string]identity.Identity{}
	for path, row := range r.world.register {
		register[path] = row
	}
	var rows []RowAdvance
	for path, digest := range r.hash {
		row := r.world.register[path]
		if digest == row.ContentHash {
			continue
		}
		next := r.finalRevision(path, digest)
		rows = append(rows, RowAdvance{Path: path, From: row.Revision, To: next})
		row.ContentHash, row.Revision = digest, next
		register[path] = row
	}
	if len(rows) > 0 {
		target := filepath.Join(r.world.projectRoot, registerName)
		if err := writeSource(target, []byte(identity.RenderIdentities(register)), 0o644); err != nil {
			return result, fmt.Errorf("writing %s: %w", registerName, err)
		}
	}
	slices.SortFunc(rows, func(a, b RowAdvance) int { return strings.Compare(a.Path, b.Path) })
	slices.Sort(result.Pages)
	result.Rows = rows
	return result, nil
}
