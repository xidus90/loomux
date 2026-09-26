package maintenance_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitenv"
)

// world is the ground a maintenance test stands on: the state directory
// everything is written to, the fallback directory the old stock is still read
// from, and the areas registered in them.
//
// It is written for several areas from the first test that needs one, because
// Reconcile works on a whole vault: a second helper beside this one would mean
// two answers to "what does a registered area look like", and the pair would
// drift the moment one of them learned a new key.
type world struct {
	StateDir string
	Fallback string

	// Root is the directory the areas hang off, so a test can add a second
	// one beside the first without inventing a place for it.
	Root string

	Areas     []config.Area
	Manifests map[string]*config.Manifest

	// docIDs counts across the whole world and not per area. A real doc id is
	// unique over every area a vault holds, and a counter that restarted per
	// area would give the first file of the second area the key the first
	// area's first file already has -- so anything that merges two areas'
	// findings by doc id would drop one of them without a word.
	docIDs int
}

// areaWorld is a world with exactly one area, hoisted into fields so the
// common case reads as `w.Area` rather than `w.Areas[0]`.
type areaWorld struct {
	*world
	Area     config.Area
	Manifest *config.Manifest
}

// areaOptions is everything a test may vary about one area. The zero value is
// the ordinary case: a writable area named `project/one`, no repository.
type areaOptions struct {
	Scope string

	// Files are laid down under the area's path, keyed by a slash-separated
	// relative path.
	Files map[string]string

	// Registered are the files the register is built over. Left out -- a nil
	// slice -- means every file of Files, which is the state a finished
	// `reindex` leaves behind. An explicit empty slice registers nothing,
	// which is a world worth building: an area whose register was never
	// written.
	Registered []string

	ReadOnly bool

	// Git lays the files down as a repository with one commit before the
	// register is written, so HEAD and the register agree to begin with.
	Git bool

	// Include is `[index] include`. The default covers the extensions the
	// tests of this package use; the manifest default (`**/*.md`) would make
	// a `.go` fixture invisible, and `**/*` would index the manifest itself,
	// which `AlwaysExcludes` does not yet know about for `.loomux`.
	Include []string

	// Review is `[layout] review`, the one declaration that makes an area the
	// vault's review centre. Left out, the area declares none -- which is the
	// state ErrNoReviewCentre is about, so it has to stay the default.
	Review string

	// Wiki is `[layout] wiki` and, joined onto the area, the registered
	// WikiPath. Left out, the area names no wiki at all and raises no source
	// case, because nothing derives from its sources.
	Wiki string

	// PrivacyMode is `[privacy] mode`. Left out, the manifest's own default
	// applies, which is `manual_cloud`.
	PrivacyMode string

	// Declaration is appended to the written declaration as it stands: a
	// table the options above do not spell out, such as the area's [model].
	Declaration string

	// Cites are the wiki pages whose frontmatter names sources, keyed by the
	// page's path under the area and holding the sources' paths. The pages are
	// written after the doc ids are handed out and before the register is
	// hashed, because a page has to carry the very id the register gave its
	// source -- that pair is the whole of the derivation index.
	Cites map[string][]string

	// Realizations is `realization` per wiki page, keyed the way Cites is. A
	// page named here and not in Cites is written all the same, because the
	// merge trigger asks only what a page still promises and never which
	// source it came from -- a page with no sources at all is an ordinary
	// candidate.
	Realizations map[string]string

	// NoManifest lays the files down and writes no declaration: an area
	// registered before it declared itself, which reconcile skips.
	NoManifest bool

	// BrokenManifest writes a declaration that is not valid TOML, which
	// reconcile refuses over rather than skipping.
	BrokenManifest bool
}

// newWorld is an empty vault: two directories and no area yet.
func newWorld(t *testing.T) *world {
	t.Helper()
	root := t.TempDir()
	return &world{
		StateDir:  filepath.Join(root, "state"),
		Fallback:  filepath.Join(root, "legacy"),
		Root:      filepath.Join(root, "areas"),
		Manifests: map[string]*config.Manifest{},
	}
}

// newArea is the one-area world the scan tests stand on: a register whose
// hashes match the files, which is the state a change is a change against.
func newArea(t *testing.T, files map[string]string) areaWorld {
	t.Helper()
	return newAreaWith(t, areaOptions{Files: files})
}

func newAreaWith(t *testing.T, opts areaOptions) areaWorld {
	t.Helper()
	w := newWorld(t)
	area, manifest := w.addArea(t, opts)
	return areaWorld{world: w, Area: area, Manifest: manifest}
}

// addArea lays down one area: its files, its declaration, and the register
// `reindex` would have left behind.
func (w *world) addArea(t *testing.T, opts areaOptions) (config.Area, *config.Manifest) {
	t.Helper()
	scope := opts.Scope
	if scope == "" {
		scope = "project/one"
	}
	include := opts.Include
	if len(include) == 0 {
		include = []string{"**/*.go", "**/*.md", "**/*.txt"}
	}

	path := filepath.Join(w.Root, filepath.FromSlash(scope))
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	for name, content := range opts.Files {
		writeUnder(t, path, name, content)
	}

	registered := opts.Registered
	if registered == nil {
		registered = sortedKeys(opts.Files)
	}
	// The ids first, the citing pages second and the hashes third: a page has
	// to name the id the register gave its source, and its own row has to
	// describe the page as it stands after that name went in.
	ids := w.assignDocIDs(registered)
	for _, page := range wikiPagesOf(opts) {
		writeUnder(t, path, page, citingPage(page, opts.Cites[page], ids, opts.Realizations[page]))
	}

	// Where the declaration goes is `ResolvedAreaDir` and not the area: a
	// read-only area keeps every artefact -- the declaration among them -- in
	// the state directory, and a helper that always wrote it into the tree
	// would let a reader that looks in the wrong place pass. Measured: with
	// the declaration in the tree, a reconcile that reads `area.Path` skips
	// every read-only area **without a word**, and nothing in this suite says
	// so.
	declarationDir := config.ResolvedAreaDir(
		config.Area{Scope: scope, Path: path, ReadOnly: opts.ReadOnly}, w.StateDir, w.Fallback)
	switch {
	case opts.NoManifest:
	case opts.BrokenManifest:
		writeUnder(t, declarationDir, ".loomux/config.toml", "[area\nscope = \n")
	default:
		writeUnder(t, declarationDir, ".loomux/config.toml", declaration(scope, include, opts))
	}

	if opts.Git {
		initRepo(t, path)
		gitRun(t, path, "add", "--all")
		gitCommit(t, path, "first")
	}

	area := config.Area{Scope: scope, Path: path, ReadOnly: opts.ReadOnly}
	if opts.Wiki != "" {
		// Absolute, the way the registry holds it: `index/walk.go` takes the
		// relative step from Area.Path to Area.WikiPath itself.
		area.WikiPath = filepath.Join(path, filepath.FromSlash(opts.Wiki))
	}
	var manifest *config.Manifest
	if !opts.NoManifest && !opts.BrokenManifest {
		read, err := config.ReadAreaManifestUntilStage4(declarationDir)
		if err != nil {
			t.Fatalf("ReadAreaManifestUntilStage4: %v", err)
		}
		manifest = read
		w.Manifests[scope] = read
	}
	w.writeRegister(t, area, ids)

	w.Areas = append(w.Areas, area)
	return area, manifest
}

// assignDocIDs hands one key per registered path out of the world's counter,
// in code point order so a repeated build of the same world agrees with itself.
func (w *world) assignDocIDs(relatives []string) map[string]string {
	ids := map[string]string{}
	sorted := append([]string(nil), relatives...)
	sort.Strings(sorted)
	for _, relative := range sorted {
		w.docIDs++
		ids[relative] = docID(w.docIDs)
	}
	return ids
}

// wikiPagesOf is every page an area declares, in code point order: the citing
// ones and the ones that only carry a realization. One page may be both, and
// then it is written once.
func wikiPagesOf(opts areaOptions) []string {
	pages := sortedKeys(opts.Cites)
	for _, page := range sortedKeys(opts.Realizations) {
		if _, cited := opts.Cites[page]; !cited {
			pages = append(pages, page)
		}
	}
	sort.Strings(pages)
	return pages
}

// citingPage is a wiki page whose frontmatter names its sources by the doc id
// the register gave them, and says what it still promises. `Dependents` and
// the merge trigger read nothing else off a page, so the rest is there for a
// person opening the fixture.
//
// An empty `sources` list is left out rather than written empty: the wiki
// reader judges a page by the keys its block carries, and a key standing over
// nothing is a different page than one that never named a source.
func citingPage(page string, sources []string, ids map[string]string, realization string) string {
	var out strings.Builder
	title := strings.TrimSuffix(filepath.Base(page), ".md")
	out.WriteString("---\ntype: note\ntitle: " + title + "\n")
	if realization != "" {
		out.WriteString("realization: " + realization + "\n")
	}
	if len(sources) > 0 {
		out.WriteString("sources:\n")
	}
	for _, source := range sources {
		out.WriteString("  - resource: " + source + "\n    doc_id: " + ids[source] + "\n")
	}
	out.WriteString("---\n\n# " + title + "\n\nWhat the sources say.\n")
	return out.String()
}

// Events writes the merge log the `post-merge` hook would have left behind:
// one tab-separated line per event, in the state directory every reader of
// this stage prefers.
func (w *world) Events(t *testing.T, events ...maintenance.MergeEvent) {
	t.Helper()
	var out strings.Builder
	for _, event := range events {
		out.WriteString(strings.Join([]string{
			event.Repo, event.First, event.Last, event.Branch, pytext.IsoFormat(event.At),
		}, "\t") + "\n")
	}
	writeUnder(t, w.StateDir, "maintenance/merge-events.tsv", out.String())
}

// Event is one landed merge of an area, the range running from the commit
// before HEAD to HEAD. The caller owes the area a second commit; a repository
// with one commit has no range.
func (w *world) Event(t *testing.T, scope string) maintenance.MergeEvent {
	t.Helper()
	return maintenance.MergeEvent{
		Repo:   filepath.Join(w.Root, filepath.FromSlash(scope)),
		First:  w.Revision(t, scope, "HEAD~1"),
		Last:   w.Revision(t, scope, "HEAD"),
		Branch: "feature",
		At:     w.Now(),
	}
}

// Revision is the object name a revision spells out in an area's repository.
func (w *world) Revision(t *testing.T, scope, spec string) string {
	t.Helper()
	return gitOutput(t, filepath.Join(w.Root, filepath.FromSlash(scope)), "rev-parse", spec)
}

// CommitFile lays a file down in an area's repository and commits it, so a
// range with two ends exists.
func (w *world) CommitFile(t *testing.T, scope, relative, content, message string) {
	t.Helper()
	root := filepath.Join(w.Root, filepath.FromSlash(scope))
	writeUnder(t, root, relative, content)
	gitRun(t, root, "add", "--all")
	gitCommit(t, root, message)
}

// Dropped is the log of consumed events, or "" while nothing was consumed.
func (w *world) Dropped(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(w.StateDir, "maintenance", "merge-events.done.tsv"))
	if err != nil {
		return ""
	}
	return string(raw)
}

// writeRegister is the register `reindex` writes: one row per registered path,
// its digest taken from the file as it lies now, revision 1.
func (w *world) writeRegister(t *testing.T, area config.Area, ids map[string]string) {
	t.Helper()
	identities := map[string]identity.Identity{}
	for relative, id := range ids {
		digest, err := identity.ContentHash(filepath.Join(area.Path, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("ContentHash %s: %v", relative, err)
		}
		identities[relative] = identity.Identity{
			DocID:       id,
			Relative:    relative,
			ContentHash: digest,
			Revision:    1,
		}
	}
	w.writeRegisterText(t, area, identity.RenderIdentities(identities))
}

// Change rewrites one file of one area, which is what makes the next scan see
// it as changed.
//
// The caller owes content of a different length than what stood there: the
// stat cache compares modification time and size, and a same-size rewrite
// within the clock's resolution would hide behind it -- the file would never
// be hashed again and the change would go unnoticed.
func (w *world) Change(t *testing.T, scope, relative, content string) {
	t.Helper()
	writeUnder(t, filepath.Join(w.Root, filepath.FromSlash(scope)), relative, content)
}

// Lookup is the pair of directories every reader of this stage is handed:
// written to the first, read from the first and, as long as nothing lies
// there, from the second.
func (w *world) Lookup() config.ArtifactLookup {
	return config.ArtifactLookup{Primary: w.StateDir, Fallback: w.Fallback}
}

// Now is the point in time the tests reconcile at. Fixed and not the clock: a
// case id carries the day, and a run started a second before midnight would
// open a second case beside the first.
func (w *world) Now() time.Time {
	return time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)
}

// writeRegisterText puts a register where this area's artefacts live --
// `ResolvedAreaDir`, not the area, because a read-only area keeps them in the
// state directory and a test that wrote them into the tree would prove the
// wrong thing.
func (w *world) writeRegisterText(t *testing.T, area config.Area, text string) {
	t.Helper()
	dir := config.ResolvedAreaDir(area, w.StateDir, w.Fallback)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_identities.tsv"), []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// docID is a register key that looks like the real thing -- 26 characters of
// the Crockford alphabet -- and is unique across the whole world, the way a
// real one is unique across a whole vault.
func docID(n int) string {
	digits := ""
	for n > 0 {
		digits = string(identity.CrockfordAlphabet[n%32]) + digits
		n /= 32
	}
	return strings.Repeat("0", 26-len(digits)) + digits
}

func declaration(scope string, include []string, opts areaOptions) string {
	quoted := make([]string, len(include))
	for i, glob := range include {
		quoted[i] = config.QuoteTOML(glob)
	}
	wiki := opts.Wiki
	if wiki == "" {
		wiki = "wiki"
	}
	out := "[area]\nscope = " + config.QuoteTOML(scope) + "\n\n" +
		"[layout]\nwiki = " + config.QuoteTOML(wiki) + "\n"
	if opts.Review != "" {
		out += "review = " + config.QuoteTOML(opts.Review) + "\n"
	}
	if opts.PrivacyMode != "" {
		out += "\n[privacy]\nmode = " + config.QuoteTOML(opts.PrivacyMode) + "\n"
	}
	out += "\n[index]\ninclude = [" + strings.Join(quoted, ", ") + "]\n"
	if opts.Declaration != "" {
		out += "\n" + opts.Declaration
	}
	return out
}

func sortedKeys[V any](entries map[string]V) []string {
	out := make([]string, 0, len(entries))
	for name := range entries {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func writeUnder(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// requireGit stands in front of every case that builds a repository, for the
// reason vcs_test.go states: without git on the PATH a missing baseline and a
// disagreeing one look alike, and half the cases would pass for the wrong
// reason.
//
// Written out here rather than borrowed: Go cannot import another package's
// test code, and vcs_test.go's own helper says the maintenance package gets
// one of its own.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

// initRepo turns a laid-down area into a repository. `core.autocrlf=false` is
// not tidiness: git for Windows sets it globally, and with it on, a fixture
// committed with CRLF would arrive in the object store already folded -- so
// the case that proves the baseline folds CRLF itself would pass without the
// folding ever happening.
func initRepo(t *testing.T, root string) {
	t.Helper()
	requireGit(t)
	gitRun(t, root, "init", "-q")
	gitRun(t, root, "config", "core.autocrlf", "false")
	writeUnder(t, root, ".gitattributes", "* -text\n")
}

// gitCommit carries the identity as `-c` on the call, so nothing of it is left
// behind in the fixture, and `commit.gpgsign=false` keeps a machine whose user
// signs by default from asking for a key.
func gitCommit(t *testing.T, root, message string) {
	t.Helper()
	gitRun(t, root,
		"-c", "user.name=Test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"commit", "-q", "-m", message)
}

// gitOutput is gitRun for a call whose answer is wanted rather than only its
// success -- the object names a merge event's range is spelt in.
func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = gitenv.Environ()
	out, err := command.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = gitenv.Environ()
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
