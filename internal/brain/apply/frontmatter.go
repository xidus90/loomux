package apply

// The reference is `_advance` and `_advance_register` in
// `src/brain/maintenance/apply.py`. It renders through PyYAML's `safe_dump`
// and writes `datetime.isoformat()`, where yaml.v3 and `time.RFC3339` would
// differ. pyyaml.go (the load) and pyyaml_emit.go (the dump) make the bytes
// the same.

import (
	"errors"
	"regexp"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/pytext"
)

// advanceBlock is `apply._FRONTMATTER` (apply.py:224): it decides where the
// body starts. documentBlock is `document._FRONTMATTER` (document.py:10): it
// decides what YAML is read. The reference uses both, and where they
// disagree -- a closing `---` with trailing blanks -- it reads no YAML at all
// and keeps the body, so the old frontmatter is gone. That is reproduced.
var (
	advanceBlock  = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?s)\A---\n(.*?)\n---[ \t]*\n`) })
	documentBlock = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n`) })
)

// SourceUpdate is the new state of one source a case names. Revision is the
// case's revision of the source, not the page's: the entry gets Revision+1,
// as `state.revision + 1` in `_advance`.
type SourceUpdate struct {
	DocID       string
	ContentHash string
	Revision    int
}

// IsoFormat is Python's `datetime.isoformat()` of t in UTC:
// `2026-09-22T08:16:27.936837+00:00`, the fraction only when the
// microseconds are not zero, never `Z`.
func IsoFormat(t time.Time) string {
	return pytext.IsoFormat(t.UTC())
}

// AdvanceFrontmatter writes the new source state, the generation time and a
// `verified` entry into the frontmatter of page, as `_advance` does. The
// caller hands in the page folded to LF, as `approve` does before patching.
// Every `sources[]` entry whose `doc_id` reads, as Python's `str()`, like an
// update's DocID gets that update's hash and revision; `generated.at` is
// set; `{by: reviewer, at: now}` is appended to `verified`. The frontmatter
// is written again as `yaml.safe_dump(sort_keys=False, allow_unicode=True,
// default_flow_style=False)` writes it, so comments and quoting styles do
// not survive, and the body stays byte for byte.
func AdvanceFrontmatter(page string, updates []SourceUpdate, reviewer string, now time.Time) (string, error) {
	block := advanceBlock().FindStringIndex(page)
	if block == nil {
		return "", errors.New("the target page has no frontmatter to advance")
	}
	meta := newDict()
	if match := documentBlock().FindStringSubmatch(page); match != nil {
		loaded, err := loadFrontmatter(match[1])
		if err != nil {
			return "", err
		}
		meta = loaded
	}
	stamp := IsoFormat(now)
	advanceSourceEntries(meta, updates)
	if generated := meta.get("generated"); generated != nil && generated.kind == pyDict {
		generated.set("at", newStr(stamp))
	} else {
		at := newDict()
		at.set("at", newStr(stamp))
		meta.set("generated", at)
	}
	entry := newDict()
	entry.set("by", newStr(reviewer))
	entry.set("at", newStr(stamp))
	if verified := meta.get("verified"); verified != nil && verified.kind == pyList {
		verified.items = append(verified.items, entry)
	} else {
		meta.set("verified", &pyValue{kind: pyList, items: []*pyValue{entry}})
	}
	return "---\n" + dumpYAML(meta) + "---\n" + page[block[1]:], nil
}

// AdvanceSources is the half of AdvanceFrontmatter a rejection needs: the
// `sources[]` entries the updates name get their hash and revision+1, and
// nothing else changes -- no `generated`, no `verified`, since a rejected
// page was neither regenerated nor confirmed. A page without frontmatter or
// without a matching entry is returned as it is, with false: rewriting it
// would only reformat YAML nobody asked to touch. So is a page on which the
// two frontmatter patterns disagree, where AdvanceFrontmatter would drop the
// old frontmatter rather than read it.
func AdvanceSources(page string, updates []SourceUpdate) (string, bool, error) {
	block := advanceBlock().FindStringIndex(page)
	match := documentBlock().FindStringSubmatch(page)
	if block == nil || match == nil {
		return page, false, nil
	}
	meta, err := loadFrontmatter(match[1])
	if err != nil {
		return "", false, err
	}
	if !advanceSourceEntries(meta, updates) {
		return page, false, nil
	}
	return "---\n" + dumpYAML(meta) + "---\n" + page[block[1]:], true, nil
}

// advanceSourceEntries moves every `sources[]` entry of meta whose `doc_id`
// reads, as Python's `str()`, like an update's DocID to that update's hash
// and revision+1, and reports whether any entry matched.
func advanceSourceEntries(meta *pyValue, updates []SourceUpdate) bool {
	fresh := make(map[string]SourceUpdate, len(updates))
	for _, update := range updates {
		fresh[update.DocID] = update
	}
	entries := meta.get("sources")
	if entries == nil || entries.kind != pyList {
		return false
	}
	changed := false
	for _, entry := range entries.items {
		if entry.kind != pyDict {
			continue
		}
		id, ok := pythonStr(entry.get("doc_id"))
		state, found := fresh[id]
		if ok && found {
			entry.set("content_hash", newStr(state.ContentHash))
			entry.set("revision", newInt(state.Revision+1))
			changed = true
		}
	}
	return changed
}

// AdvanceRegister is the text `_advance_register` hands to `_write(...,
// scaffold=True)` for one register: the register at tsvPath, read and never
// written here, with every source of caseStates moved on. caseStates maps a
// register-relative path to the case's state of that source, un-advanced:
// its row becomes that state with the revision one past it, the same
// `state.revision + 1` AdvanceFrontmatter writes to the page. The rest of
// the register is rendered as `render_identities` renders it.
//
// The caller writes the text through the barrier, whose write-if-changed
// decides whether anything changed, as `_write` does in Python.
func AdvanceRegister(tsvPath string, caseStates map[string]identity.Identity) (string, error) {
	identities, err := identity.ReadIdentities(tsvPath)
	if err != nil {
		return "", err
	}
	for relative, caseState := range caseStates {
		caseState.Revision++
		identities[relative] = caseState
	}
	return identity.RenderIdentities(identities), nil
}
