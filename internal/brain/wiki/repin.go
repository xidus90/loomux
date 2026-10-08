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
