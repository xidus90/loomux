package wiki

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/identity"
)

// SourceFinding is one finding of the source check: a pin of the project's
// own scope that no longer matches the register, a register row that no
// longer matches its file, or pages that pin each other. Page is the
// wiki-relative page, or for register-stale the project-relative file.
type SourceFinding struct {
	Rule    string
	Page    string
	Message string
}

// The rules of the source check. Repin advances what register-stale and
// pin-stale name and refuses the rest.
const (
	RuleRowMissing    = "row-missing"
	RuleSourceMissing = "source-missing"
	RuleDocID         = "doc-id"
	RuleRegisterStale = "register-stale"
	RulePinStale      = "pin-stale"
	RuleCycle         = "cycle"
)

// registerName is the register a writable area keeps at its root
// (`config.ManifestDir`); the one `wiki init` lays into a bundle is scaffold.
const registerName = "_identities.tsv"

// readSource reads pages and cited files; a test replaces it to reach the
// failures no tree provokes on every platform.
var readSource = os.ReadFile

// pin is one complete sources[] entry of the project's own scope.
type pin struct {
	index  int
	source Source
	path   string // project-relative, forward slashes
}

type sourcePage struct {
	relative string // wiki-relative, as findings name it
	repoPath string // project-relative, as the register names it
	path     string
	raw      []byte
	pins     []pin
}

type sourceWorld struct {
	projectRoot string
	register    map[string]identity.Identity
	pages       []*sourcePage
}

// SourceDrift checks the pins of the wiki at wikiRoot against the register
// at projectRoot and the register against the files the pins cite. It reads
// the tree alone, never the registry, so it answers the same in a linked
// worktree, in CI and in the checkout the registry names. Without a scope, a
// wiki or a register there is nothing to judge.
func SourceDrift(projectRoot, wikiRoot, scope string) ([]SourceFinding, error) {
	world, err := loadSources(projectRoot, wikiRoot, scope)
	if err != nil || world == nil {
		return nil, err
	}
	return world.drift()
}

func loadSources(projectRoot, wikiRoot, scope string) (*sourceWorld, error) {
	if scope == "" || wikiRoot == "" {
		return nil, nil
	}
	register, err := identity.ReadIdentities(filepath.Join(projectRoot, registerName))
	if err != nil {
		return nil, err
	}
	if len(register) == 0 {
		return nil, nil
	}
	prefix := "brain://" + scope + "/"
	world := &sourceWorld{projectRoot: projectRoot, register: register}
	err = filepath.WalkDir(wikiRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") || IsScaffoldFile(path) {
			return nil
		}
		raw, err := readSource(path)
		if err != nil {
			return err
		}
		page := pageFrom(raw, path, wikiRoot)
		if page.BrokenFrontmatter != nil {
			return nil
		}
		// Rel cannot fail here: path came out of a walk below wikiRoot, and
		// the wiki is a directory of the project or its neighbour on the same
		// volume.
		repo, _ := filepath.Rel(projectRoot, path)
		sp := &sourcePage{relative: page.Relative, repoPath: filepath.ToSlash(repo), path: path, raw: raw}
		for i, s := range page.Sources {
			if !strings.HasPrefix(s.Resource, prefix) || strings.TrimSpace(s.DocID) == "" ||
				strings.TrimSpace(s.ContentHash) == "" || s.Revision == nil {
				continue
			}
			sp.pins = append(sp.pins, pin{index: i, source: s, path: strings.TrimPrefix(s.Resource, prefix)})
		}
		world.pages = append(world.pages, sp)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return world, nil
}

// fileHash hashes a cited file as the register does; "" means it is gone.
func (w *sourceWorld) fileHash(relative string) (string, error) {
	data, err := readSource(filepath.Join(w.projectRoot, filepath.FromSlash(relative)))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return identity.HashBytes(data), nil
}

func (w *sourceWorld) drift() ([]SourceFinding, error) {
	var findings []SourceFinding
	hashes := map[string]string{}
	citers := map[string][]string{}
	for _, page := range w.pages {
		for _, p := range page.pins {
			label := fmt.Sprintf("sources[%d]", p.index)
			row, ok := w.register[p.path]
			if !ok {
				findings = append(findings, SourceFinding{RuleRowMissing, page.relative,
					fmt.Sprintf("%s: %s is not in the register", label, p.path)})
				continue
			}
			digest, seen := hashes[p.path]
			if !seen {
				var err error
				if digest, err = w.fileHash(p.path); err != nil {
					return nil, err
				}
				hashes[p.path] = digest
			}
			if digest == "" {
				findings = append(findings, SourceFinding{RuleSourceMissing, page.relative,
					fmt.Sprintf("%s: %s does not exist", label, p.path)})
				continue
			}
			if strings.TrimSpace(p.source.DocID) != row.DocID {
				findings = append(findings, SourceFinding{RuleDocID, page.relative,
					fmt.Sprintf("%s: doc_id %s, the register has %s for %s", label, p.source.DocID, row.DocID, p.path)})
				continue
			}
			if !slices.Contains(citers[p.path], page.relative) {
				citers[p.path] = append(citers[p.path], page.relative)
			}
			switch {
			case *p.source.Revision != row.Revision:
				findings = append(findings, SourceFinding{RulePinStale, page.relative,
					fmt.Sprintf("%s: %s pinned at revision %d, the register row is at revision %d", label, p.path, *p.source.Revision, row.Revision)})
			case p.source.ContentHash != row.ContentHash:
				findings = append(findings, SourceFinding{RulePinStale, page.relative,
					fmt.Sprintf("%s: %s pinned with another content_hash than its register row (revision %d)", label, p.path, row.Revision)})
			}
		}
	}
	for path, pages := range citers {
		row := w.register[path]
		if hashes[path] != row.ContentHash {
			findings = append(findings, SourceFinding{RuleRegisterStale, path,
				fmt.Sprintf("changed since its register row (revision %d); cited by %s", row.Revision, strings.Join(pages, ", "))})
		}
	}
	findings = append(findings, w.cycles()...)
	slices.SortFunc(findings, func(a, b SourceFinding) int {
		return cmp.Or(cmp.Compare(a.Page, b.Page), cmp.Compare(a.Rule, b.Rule), cmp.Compare(a.Message, b.Message))
	})
	return findings, nil
}

// cycles finds the pages that pin each other (Tarjan's strongly connected
// components over the pins between pages), one finding per cycle.
func (w *sourceWorld) cycles() []SourceFinding {
	byPath := map[string]*sourcePage{}
	for _, p := range w.pages {
		byPath[p.repoPath] = p
	}
	index, low := map[string]int{}, map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out []SourceFinding
	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = len(index), len(index)
		stack = append(stack, v)
		onStack[v] = true
		self := false
		for _, p := range byPath[v].pins {
			if byPath[p.path] == nil {
				continue
			}
			if p.path == v {
				self = true
			}
			if _, seen := index[p.path]; !seen {
				visit(p.path)
				low[v] = min(low[v], low[p.path])
			} else if onStack[p.path] {
				low[v] = min(low[v], index[p.path])
			}
		}
		if low[v] != index[v] {
			return
		}
		var members []string
		for {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[top] = false
			members = append(members, byPath[top].relative)
			if top == v {
				break
			}
		}
		if len(members) > 1 || self {
			slices.Sort(members)
			out = append(out, SourceFinding{RuleCycle, members[0], "pages cite each other: " + strings.Join(members, ", ")})
		}
	}
	for _, p := range w.pages {
		if _, seen := index[p.repoPath]; !seen {
			visit(p.repoPath)
		}
	}
	return out
}
