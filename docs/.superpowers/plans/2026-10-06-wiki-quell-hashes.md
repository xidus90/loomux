# Wiki-Quell-Hashes im Tor — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eine Spur `lint/wiki-sources` in `loomux check`, die veraltete Registerzeilen und `sources:`-Pins zitierter Dateien meldet, und ein Befehl `loomux wiki repin`, mit dem ein Mensch beide im Zweig nachzieht.

**Architecture:** Eine reine Prüffunktion `wiki.SourceDrift` liest nur Dateien im Baum (`_identities.tsv` der Projektwurzel, die Wiki-Seiten, die zitierten Dateien). `wiki.Repin` nutzt dieselbe Ladefunktion, rechnet Endhashes im Speicher über die Zitatkette und ändert Seiten per Zeilen-Edit. `hooks.WikiSourceJobs` hängt die Prüfung als eigene Spur (Stack `wiki-sources`) an `loomux check` und an `hooks.GateLanes`, nicht an das Stop-Gate. Der Wächter verweigert `wiki repin` Agenten.

**Tech Stack:** Go 1.26, Standardbibliothek; vorhandene Pakete `internal/brain/identity`, `internal/brain/wiki`, `internal/verify`, `internal/hooks`, `internal/cli`, `internal/config`.

**Spec:** `docs/.superpowers/specs/2026-10-06-wiki-quell-hashes-design.md` (mit Nachtrag „nicht im Stop-Gate“). Implementierer lesen Spec und Plan.

## Global Constraints

- Coverage 100 % je Funktion; ein Ausschluss nur mit `//coverage:exempt <grund>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst. Kompilierte Regexe auf Paketebene sind erlaubt (Muster: `config.unsafeInScope`).
- Code, Bezeichner, Kommentare, Fehlermeldungen, Commits englisch; Doku-Prosa unter `docs/de` und das Wiki deutsch, `docs/en` englisch.
- Commits nach Conventional Commits, Scope = Code-Bereich (`wiki`, `verify`, `hooks`, `cli`), kein Verweis auf Plan, Spec oder Task. Autor ist der Nutzer; keine `Co-Authored-By`-Zeile.
- Commit-Nachrichten per Datei (`git commit -F <datei>`), Ausgabe des Commits in eine Datei im Scratchpad, nie nach `/dev/null`.
- Kein Agent schreibt `.loomux/config.toml`. Kein Agent pusht.
- Kein Agent fährt `loomux wiki repin` (ab Task 6 verweigert es der Wächter; vorher gilt es als Regel).
- Prozesse nie nach Namen beenden; nur eigene per PID. Kein `python -`/Heredoc an einen Interpreter. Kein Hintergrundprozess bleibt laufen, wenn ein Task fertig ist.
- Jede neue Testzeile läuft vor dem Code rot (bei neuen Symbolen gegen Stubs, die das alte Verhalten liefern); der Bericht zeigt Befehl und `--- FAIL`-Ausgabe.
- Mutationsrunde je neuer Funktion per `go test -overlay` (Pfade in Windows-Form `C:/…`, nie zusammen mit `-cover`): jede Teilbedingung einzeln streichen, jede Fehlerprüfung auf „jeder Fehler“ weiten; Mutanten müssen bauen. Je Mutant eine Ausgabedatei `t<N>-mut-<n>.txt` im Scratchpad.
- Gate: `sh ci/gate.sh` bzw. das pre-commit-Tor. Ein Torlauf schreibt seine Ausgabe erst ganz in eine Datei.

## Review Focus

1. **Seiten mit CRLF:** Eine Seite, die mit `\r\n` gespeichert ist, muss `repin` mit CRLF zurückschreiben, und die Kaskade rechnet ihren Hash mit LF-Faltung — Test in Task 2 (Zeilen-Edit) und Task 3 (Kaskade über eine CRLF-Seite).
2. **Werte mit Anführungszeichen, ohne, mit Kommentar:** `content_hash: "sha256:…"`, `'sha256:…'`, unquotiert, `revision: 3 # x` — die ersten drei bleiben in ihrer Form, der Kommentar führt zur Ablehnung statt zu stillem Verlust — Task 2.
3. **Ein Pin auf einen fremden Scope oder ein unvollständiger Eintrag in derselben Seite** wie ein veralteter Pin: `repin` ändert nur den eigenen, vollständigen Eintrag und lässt den Rest byte-gleich — Task 3.
4. **Ein Wiki, das nicht unter `docs/wiki` liegt** (`[layout] wiki = "wiki"`): Die Spur findet es über `wikiDirFor` — Task 5.
5. **Ein Projekt mit `[modules] brain = false`**: keine Spur, auch wenn Register und Pins veraltet sind — Task 4 und Task 5.

---

### Task 1: `identity.HashBytes` und die Prüfung `wiki.SourceDrift`

**Files:**
- Modify: `internal/brain/identity/identity.go` (Funktion `ContentHash`, Zeile 68–79)
- Modify: `internal/brain/identity/identity_test.go`
- Create: `internal/brain/wiki/sources.go`
- Create: `internal/brain/wiki/sources_test.go`

**Interfaces:**
- Consumes: `identity.ReadIdentities(path string) (map[string]identity.Identity, error)` (fehlende Datei → leere Map), `identity.Identity{DocID, Relative, ContentHash string; Revision int}`, `wiki.pageFrom(content []byte, filePath, wikiRoot string) *WikiPage`, `wiki.IsScaffoldFile(name string) bool`, `WikiPage.Sources []Source`, `WikiPage.BrokenFrontmatter *string`, `Source{ID, Resource, DocID, ContentHash string; Revision *int}`.
- Produces:
  - `func identity.HashBytes(data []byte) string` — SHA-256 mit CRLF→LF, Form `sha256:<hex>`.
  - `type wiki.SourceFinding struct { Rule, Page, Message string }`
  - Konstanten `wiki.RuleRowMissing = "row-missing"`, `RuleSourceMissing = "source-missing"`, `RuleDocID = "doc-id"`, `RuleRegisterStale = "register-stale"`, `RulePinStale = "pin-stale"`, `RuleCycle = "cycle"`
  - `func wiki.SourceDrift(projectRoot, wikiRoot, scope string) ([]SourceFinding, error)`
  - paketintern für Task 3: `type sourceWorld`, `type sourcePage`, `type pin`, `func loadSources(projectRoot, wikiRoot, scope string) (*sourceWorld, error)`, `func (w *sourceWorld) drift() ([]SourceFinding, error)`, `func (w *sourceWorld) fileHash(relative string) (string, error)`, Seam `var readSource = os.ReadFile`.

- [ ] **Step 1: Failing test für `HashBytes`**

In `internal/brain/identity/identity_test.go` ergänzen:

```go
// HashBytes is ContentHash without the file: the same fold of CRLF to LF,
// so a repin that hashes bytes it has not written yet agrees with a later
// read of the file.
func TestHashBytesFoldsCRLFLikeContentHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(path, []byte("a\r\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fromFile, err := ContentHash(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := HashBytes([]byte("a\r\nb\n")); got != fromFile {
		t.Fatalf("HashBytes %s, ContentHash %s", got, fromFile)
	}
	if got := HashBytes([]byte("a\nb\n")); got != fromFile {
		t.Fatalf("LF and CRLF must hash alike: %s vs %s", got, fromFile)
	}
	if HashBytes([]byte("a\rb\n")) == fromFile {
		t.Fatal("a lone CR is no line end and must not fold")
	}
}
```

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/brain/identity/ -run TestHashBytesFoldsCRLFLikeContentHash -count=1`
Expected: Build-Fehler `undefined: HashBytes`. Danach einen Stub `func HashBytes(data []byte) string { return "" }` einsetzen und erneut laufen lassen: `--- FAIL` mit `HashBytes , ContentHash sha256:…`.

- [ ] **Step 3: `HashBytes` implementieren, `ContentHash` darauf stützen**

```go
// ContentHash computes the SHA-256 digest of file bytes with CRLF normalized to LF.
// Normalizing line endings ensures checkouts on different platforms (Windows vs Linux)
// do not invalidate content hashes or trigger spurious modification reports (Spec 5.6).
func ContentHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return HashBytes(data), nil
}

// HashBytes is ContentHash for bytes already in hand.
func HashBytes(data []byte) string {
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	sum := sha256.Sum256(normalized)
	return fmt.Sprintf("sha256:%x", sum)
}
```

- [ ] **Step 4: Grün**

Run: `go test ./internal/brain/identity/ -count=1`
Expected: `ok`.

- [ ] **Step 5: Testwelt und erste Prüfung schreiben (rot)**

`internal/brain/wiki/sources_test.go`:

```go
package wiki

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
)

const testScope = "project/x"

// testPin is one sources[] entry a test page carries.
type testPin struct {
	path, docID, hash string
	revision         int
}

// page renders a wiki page with the given pins, in the form the bundle
// writes: block list, two-space entries, the hash in double quotes.
func page(title string, pins ...testPin) string {
	var b strings.Builder
	b.WriteString("---\ntype: Topic\ntitle: " + title + "\ndescription: d\n")
	if len(pins) > 0 {
		b.WriteString("sources:\n")
		for i, p := range pins {
			b.WriteString("  - id: s" + string(rune('0'+i)) + "\n")
			b.WriteString("    resource: brain://" + testScope + "/" + p.path + "\n")
			b.WriteString("    doc_id: " + p.docID + "\n")
			b.WriteString("    content_hash: \"" + p.hash + "\"\n")
			b.WriteString("    revision: " + itoa(p.revision) + "\n")
		}
	}
	b.WriteString("---\n\n# " + title + "\n")
	return b.String()
}

func itoa(n int) string { return strconv.Itoa(n) }

// project lays files out by project-relative path and writes the register
// from rows; no rows means no register file.
func project(t *testing.T, files map[string]string, rows ...identity.Identity) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		writeAt(t, root, rel, body)
	}
	if len(rows) > 0 {
		register := map[string]identity.Identity{}
		for _, r := range rows {
			register[r.Relative] = r
		}
		writeAt(t, root, "_identities.tsv", identity.RenderIdentities(register))
	}
	return root
}

func writeAt(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hashOf(s string) string { return identity.HashBytes([]byte(s)) }

func drift(t *testing.T, root string) []SourceFinding {
	t.Helper()
	got, err := SourceDrift(root, filepath.Join(root, "docs", "wiki"), testScope)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// A project where every pin matches its row and every row its file has
// nothing to report: the state master is in when this lane arrives.
func TestSourceDriftIsSilentWhenEverythingMatches(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(doc), 5}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 5})
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}
```

Dazu `strconv` in die Imports. Run: `go test ./internal/brain/wiki/ -run TestSourceDrift -count=1` → Build-Fehler `undefined: SourceDrift`. Dann `sources.go` mit Stub (`func SourceDrift(...) ([]SourceFinding, error) { return nil, nil }` plus Typ und Konstanten) anlegen: Dieser erste Test ist mit dem Stub grün (er prüft Schweigen) — das ist erwartet; die folgenden Tests sind gegen den Stub rot.

- [ ] **Step 6: Die übrigen Prüftests schreiben (gegen den Stub rot)**

In `sources_test.go` ergänzen. Revisionen und Hashes benachbarter Felder sind paarweise verschieden; der erwartete Befund steht nie zufällig an erster Stelle der Lesereihenfolge.

```go
// Stelle 1: a cited file changed and nobody ran reindex. Pin and row still
// agree, so only the register is stale -- the case that left 22 pins behind.
func TestSourceDriftReportsAFileAheadOfItsRow(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/b.md": page("B", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	want := []SourceFinding{{RuleRegisterStale, "docs/de/a.md", "changed since its register row (revision 3); cited by topics/b.md, topics/t.md"}}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// Stelle 2: the register moved on (a reindex in the registry's checkout),
// the page did not.
func TestSourceDriftReportsAPinBehindItsRow(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(now), Revision: 4})
	want := []SourceFinding{{RulePinStale, "topics/t.md", "sources[0]: docs/de/a.md pinned at revision 3, the register row is at revision 4"}}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// Same revision, another hash: still behind, and the message says what
// differs, since the numbers alone would read as equal.
func TestSourceDriftReportsAPinWithAnotherHashAtTheSameRevision(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf("# other\n"), 6}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 6})
	want := []SourceFinding{{RulePinStale, "topics/t.md", "sources[0]: docs/de/a.md pinned with another content_hash than its register row (revision 6)"}}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// Findings 1 to 3 stand alone: a pin that has one gets no other.
func TestSourceDriftReportsWhatRepinCannotFix(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md": doc,
		"docs/wiki/topics/t.md": page("T",
			testPin{"docs/de/new.md", "01N", hashOf(doc), 1},
			testPin{"docs/de/gone.md", "01G", hashOf(doc), 2},
			testPin{"docs/de/a.md", "01WRONG", hashOf("# x\n"), 9}),
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 7},
		identity.Identity{DocID: "01G", Relative: "docs/de/gone.md", ContentHash: hashOf(doc), Revision: 2},
	)
	want := []SourceFinding{
		{RuleDocID, "topics/t.md", "sources[2]: doc_id 01WRONG, the register has 01A for docs/de/a.md"},
		{RuleRowMissing, "topics/t.md", "sources[0]: docs/de/new.md is not in the register"},
		{RuleSourceMissing, "topics/t.md", "sources[1]: docs/de/gone.md does not exist"},
	}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// A pin of another area, an incomplete entry (house/source-incomplete owns
// it) and a page whose frontmatter does not load (the okf axis owns it) are
// none of this check's business; neither is a changed file nobody cites.
func TestSourceDriftLeavesOtherAxesAndScopesAlone(t *testing.T) {
	doc, uncited := "# A\n", "# U\n"
	foreign := "---\ntype: Topic\ntitle: F\ndescription: d\nsources:\n  - id: f\n    resource: brain://shared/kb/x.md\n    doc_id: 01F\n    content_hash: \"sha256:00\"\n    revision: 1\n---\n"
	incomplete := "---\ntype: Topic\ntitle: I\ndescription: d\nsources:\n  - id: i\n    resource: brain://project/x/docs/de/a.md\n    doc_id: 01A\n---\n"
	broken := "---\ntitle: [unclosed\n---\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/de/u.md":          uncited,
		"docs/wiki/topics/f.md": foreign,
		"docs/wiki/topics/i.md": incomplete,
		"docs/wiki/topics/z.md": broken,
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 2},
		identity.Identity{DocID: "01U", Relative: "docs/de/u.md", ContentHash: hashOf("# U old\n"), Revision: 8},
	)
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// A CRLF checkout of a file whose row was hashed with LF is no change.
func TestSourceDriftFoldsCRLF(t *testing.T) {
	root := project(t, map[string]string{
		"docs/de/a.md":          "# A\r\nline\r\n",
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf("# A\nline\n"), 4}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf("# A\nline\n"), Revision: 4})
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// Pages that pin each other can never both be current; a page that pins
// itself neither. One finding per cycle, naming its pages.
func TestSourceDriftReportsCitationCycles(t *testing.T) {
	a := page("A", testPin{"docs/wiki/topics/b.md", "01B", "sha256:b", 2})
	b := page("B", testPin{"docs/wiki/topics/a.md", "01A", "sha256:a", 3})
	self := page("S", testPin{"docs/wiki/topics/s.md", "01S", "sha256:s", 4})
	root := project(t, map[string]string{
		"docs/wiki/topics/a.md": a,
		"docs/wiki/topics/b.md": b,
		"docs/wiki/topics/s.md": self,
	},
		identity.Identity{DocID: "01A", Relative: "docs/wiki/topics/a.md", ContentHash: "sha256:a", Revision: 3},
		identity.Identity{DocID: "01B", Relative: "docs/wiki/topics/b.md", ContentHash: "sha256:b", Revision: 2},
		identity.Identity{DocID: "01S", Relative: "docs/wiki/topics/s.md", ContentHash: "sha256:s", Revision: 4},
	)
	var cycles []SourceFinding
	for _, f := range drift(t, root) {
		if f.Rule == RuleCycle {
			cycles = append(cycles, f)
		}
	}
	want := []SourceFinding{
		{RuleCycle, "topics/a.md", "pages cite each other: topics/a.md, topics/b.md"},
		{RuleCycle, "topics/s.md", "pages cite each other: topics/s.md"},
	}
	if !reflect.DeepEqual(cycles, want) {
		t.Fatalf("cycles %+v\nwant %+v", cycles, want)
	}
}

// A chain without a cycle is no cycle: a synthesis citing a topic citing a doc.
func TestSourceDriftFindsNoCycleInAChain(t *testing.T) {
	doc := "# A\n"
	topic := page("T", testPin{"docs/de/a.md", "01A", hashOf(doc), 2})
	root := project(t, map[string]string{
		"docs/de/a.md":              doc,
		"docs/wiki/topics/t.md":     topic,
		"docs/wiki/syntheses/s.md":  page("S", testPin{"docs/wiki/topics/t.md", "01T", hashOf(topic), 5}),
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 2},
		identity.Identity{DocID: "01T", Relative: "docs/wiki/topics/t.md", ContentHash: hashOf(topic), Revision: 5},
	)
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// Nothing to judge: no scope, no wiki, no register.
func TestSourceDriftWithoutScopeWikiOrRegister(t *testing.T) {
	root := project(t, map[string]string{
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", "sha256:a", 1}),
	})
	wikiRoot := filepath.Join(root, "docs", "wiki")
	for _, c := range []struct{ wiki, scope string }{{wikiRoot, ""}, {"", testScope}, {wikiRoot, testScope}} {
		got, err := SourceDrift(root, c.wiki, c.scope)
		if err != nil || got != nil {
			t.Fatalf("%+v: findings %+v, err %v", c, got, err)
		}
	}
}

// A lane that cannot read is the lane's failure, never its silence.
func TestSourceDriftReportsWhatItCannotRead(t *testing.T) {
	root := project(t, map[string]string{"docs/de/a.md": "# A\n"},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: "sha256:a", Revision: 1})
	if _, err := SourceDrift(root, filepath.Join(root, "docs", "missing"), testScope); err == nil {
		t.Fatal("a wiki root that does not exist must fail")
	}
	writeAt(t, root, "_identities.tsv", "doc_id\tpfad\tcontent_hash\trevision\nbroken\n")
	if _, err := SourceDrift(root, filepath.Join(root, "docs"), testScope); err == nil {
		t.Fatal("a register that does not read must fail")
	}
}

// Reading a page or a cited file can fail for reasons the tree cannot
// provoke on every platform; the seam stands in for them.
func TestSourceDriftReportsAReadFailure(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(doc), 2}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 2})
	boom := errors.New("denied")
	for _, target := range []string{"t.md", "a.md"} {
		saved := readSource
		readSource = func(path string) ([]byte, error) {
			if filepath.Base(path) == target {
				return nil, boom
			}
			return os.ReadFile(path)
		}
		_, err := SourceDrift(root, filepath.Join(root, "docs", "wiki"), testScope)
		readSource = saved
		if !errors.Is(err, boom) {
			t.Fatalf("%s: err %v", target, err)
		}
	}
}
```

Dazu `errors` in die Imports.

- [ ] **Step 7: Rot sehen**

Run: `go test ./internal/brain/wiki/ -run TestSourceDrift -count=1`
Expected: `--- FAIL` für jeden Test außer `IsSilentWhenEverythingMatches`, `LeavesOtherAxesAndScopesAlone`, `FoldsCRLF`, `FindsNoCycleInAChain` und `WithoutScopeWikiOrRegister`, die gegen den Stub schweigen. Diese fünf prüfen Schweigen und werden in der Mutationsrunde (Step 10) belegt.

- [ ] **Step 8: `sources.go` implementieren**

```go
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
```

Hinweis zu `index[v], low[v] = len(index), len(index)`: Go wertet die rechte Seite vor der Zuweisung aus, beide bekommen dieselbe Zahl, und die Map wächst danach um eins. Das ist der Tarjan-Zähler ohne eigene Variable.

- [ ] **Step 9: Grün**

Run: `go test ./internal/brain/wiki/ ./internal/brain/identity/ -count=1`
Expected: `ok` für beide. Dann Coverage nur dieser Dateien im echten Baum:
`go test ./internal/brain/wiki/ -count=1 -coverprofile=<scratchpad>/t1.out` und `go tool cover -func=<scratchpad>/t1.out | grep -E "sources.go"` → jede Funktion 100.0 %.

- [ ] **Step 10: Mutationsrunde**

Je Mutant ein Overlay (`go test -overlay <scratchpad>/t1-mut-<n>.json ./internal/brain/wiki/ -run TestSourceDrift -count=1`), Ausgabe nach `t1-mut-<n>.txt`. Mindestens (ein Boden, keine Runde — jede Teilbedingung einzeln):
1. `scope == "" ||` streichen
2. `len(register) == 0` → `false`
3. `IsScaffoldFile(path)` streichen
4. `page.BrokenFrontmatter != nil` → `false`
5. je eine der vier Bedingungen im Pin-Filter streichen (4 Mutanten)
6. `errors.Is(err, fs.ErrNotExist)` → `err != nil`
7. `continue` nach `RuleRowMissing` streichen
8. `!slices.Contains(citers…)` → `true` (Seite doppelt genannt)
9. `*p.source.Revision != row.Revision` → `false`
10. `hashes[path] != row.ContentHash` → `true`
11. `len(members) > 1 || self` → `len(members) > 1`
12. `low[v] != index[v]` → `false`
Jeder Mutant baut und stirbt an einer benannten Assertion; ein Überlebender bekommt einen Test, der ihn tötet.

- [ ] **Step 11: Commit**

```bash
git add internal/brain/identity/identity.go internal/brain/identity/identity_test.go internal/brain/wiki/sources.go internal/brain/wiki/sources_test.go
git commit -F <scratchpad>/msg-t1.txt > <scratchpad>/commit-t1.log 2>&1
```

Nachricht in `msg-t1.txt`:

```
feat(wiki): check the source pins against the register

SourceDrift reads the register at the project root, the wiki pages and
the cited files, and reports a row behind its file, a pin behind its row,
pins repin cannot fix and pages that pin each other. It reads the tree
alone, so a linked worktree and CI see what the main checkout sees.
```

---

### Task 2: Zeilen-Edit `repinText`

**Files:**
- Create: `internal/brain/wiki/repin.go` (nur `pinValue`, `repinText` und Hilfen)
- Create: `internal/brain/wiki/repin_test.go`

**Interfaces:**
- Consumes: nichts aus Task 1 außer dem Paket.
- Produces:
  - `type pinValue struct { hash string; revision int }`
  - `func repinText(text string, updates map[string]pinValue) (string, error)` — setzt in jedem `sources:`-Eintrag, dessen `doc_id` in `updates` steht, die Werte von `content_hash` und `revision`; sonst bleibt jedes Byte.

- [ ] **Step 1: Tests schreiben**

`internal/brain/wiki/repin_test.go`:

```go
package wiki

import (
	"strings"
	"testing"
)

const repinPage = "---\n" +
	"type: Topic\n" +
	"title: T\n" +
	"description: Eine lange Beschreibung, die eine YAML-Bibliothek bei achtzig Zeichen umbrechen würde, hier aber stehen bleibt.\n" +
	"sources:\n" +
	"  - id: a\n" +
	"    resource: brain://project/x/docs/de/a.md\n" +
	"    doc_id: 01A\n" +
	"    content_hash: \"sha256:aaa\"\n" +
	"    revision: 3\n" +
	"  - id: b\n" +
	"    resource: brain://project/x/docs/de/b.md\n" +
	"    doc_id: 01B\n" +
	"    content_hash: 'sha256:bbb'\n" +
	"    revision: 5\n" +
	"  - id: c\n" +
	"    resource: brain://project/x/docs/de/c.md\n" +
	"    doc_id: \"01C\"\n" +
	"    content_hash: sha256:ccc\n" +
	"    revision: 7\n" +
	"---\n\n# T\nrevision: 3 in the body stays\n"

// Only the two values of the named entries change; quotes, order, the long
// description and the body stay byte for byte.
func TestRepinTextChangesOnlyTheTwoValues(t *testing.T) {
	got, err := repinText(repinPage, map[string]pinValue{
		"01A": {"sha256:a2", 4},
		"01C": {"sha256:c2", 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.NewReplacer(
		"content_hash: \"sha256:aaa\"\n    revision: 3", "content_hash: \"sha256:a2\"\n    revision: 4",
		"content_hash: sha256:ccc\n    revision: 7", "content_hash: sha256:c2\n    revision: 8",
	).Replace(repinPage)
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// A list written at column 0, as YAML allows under a mapping key, is read
// with its own indent.
func TestRepinTextReadsAListAtColumnZero(t *testing.T) {
	text := "---\nsources:\n- id: a\n  doc_id: 01A\n  content_hash: \"sha256:aaa\"\n  revision: 3\n---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:a2", 4}})
	if err != nil {
		t.Fatal(err)
	}
	if want := "---\nsources:\n- id: a\n  doc_id: 01A\n  content_hash: \"sha256:a2\"\n  revision: 4\n---\n"; got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// Single quotes stay single quotes.
func TestRepinTextKeepsSingleQuotes(t *testing.T) {
	got, err := repinText(repinPage, map[string]pinValue{"01B": {"sha256:b2", 6}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "    content_hash: 'sha256:b2'\n    revision: 6\n") {
		t.Fatalf("got\n%s", got)
	}
}

// A page saved with CRLF keeps CRLF on every line, the changed ones too.
func TestRepinTextKeepsCRLF(t *testing.T) {
	crlf := strings.ReplaceAll(repinPage, "\n", "\r\n")
	got, err := repinText(crlf, map[string]pinValue{"01A": {"sha256:a2", 4}})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.ReplaceAll(strings.Replace(repinPage, "content_hash: \"sha256:aaa\"\n    revision: 3", "content_hash: \"sha256:a2\"\n    revision: 4", 1), "\n", "\r\n")
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// Values equal to the update leave the page as it is.
func TestRepinTextWithoutChangeIsIdentity(t *testing.T) {
	got, err := repinText(repinPage, map[string]pinValue{"01A": {"sha256:aaa", 3}})
	if err != nil || got != repinPage {
		t.Fatalf("err %v, changed: %v", err, got != repinPage)
	}
}

// Every form it cannot edit safely is refused rather than rewritten.
func TestRepinTextRefusesWhatItCannotEditSafely(t *testing.T) {
	cases := map[string]string{
		"no frontmatter":     "# T\n",
		"unclosed":           "---\ntitle: T\n",
		"flow list":          "---\nsources: [{doc_id: 01A, content_hash: x, revision: 1}]\n---\n",
		"no sources":         "---\ntitle: T\n---\n",
		"missing revision":   "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n---\n",
		"missing hash":       "---\nsources:\n  - doc_id: 01A\n    revision: 1\n---\n",
		"duplicate doc_id":   "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1\n  - doc_id: 01A\n    content_hash: \"y\"\n    revision: 2\n---\n",
		"comment":            "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1 # pinned by hand\n---\n",
		"block scalar":       "---\nsources:\n  - doc_id: 01A\n    content_hash: >\n      x\n    revision: 1\n---\n",
		"unterminated quote": "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\n      y\"\n    revision: 1\n---\n",
		"entry not found":    "---\nsources:\n  - doc_id: 01Z\n    content_hash: \"x\"\n    revision: 1\n---\n",
	}
	for name, text := range cases {
		if _, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 2}}); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
}

// A nested key of the same name deeper in an entry is not the entry's field.
func TestRepinTextIgnoresDeeperKeys(t *testing.T) {
	text := "---\nsources:\n  - doc_id: 01A\n    meta:\n      revision: 99\n    content_hash: \"x\"\n    revision: 1\n---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 2}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "      revision: 99\n") || !strings.Contains(got, "    revision: 2\n") {
		t.Fatalf("got\n%s", got)
	}
}
```

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/brain/wiki/ -run TestRepinText -count=1` → Build-Fehler `undefined: repinText`. Stub `func repinText(text string, _ map[string]pinValue) (string, error) { return text, nil }` und `type pinValue struct{ hash string; revision int }` anlegen, erneut laufen: `--- FAIL` in `ChangesOnlyTheTwoValues`, `ReadsAListAtColumnZero`, `KeepsSingleQuotes`, `KeepsCRLF`, `RefusesWhatItCannotEditSafely`, `IgnoresDeeperKeys`. `WithoutChangeIsIdentity` ist gegen den Stub grün (prüft Identität); die Mutationsrunde belegt ihn.

- [ ] **Step 3: Implementieren**

In `internal/brain/wiki/repin.go`:

```go
package wiki

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
		if seen[docID] {
			return fmt.Errorf("doc_id %s stands twice", docID)
		}
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
			line, err := setValue(lines[at], next)
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
// it cannot read as one plain or quoted scalar on this line is refused.
func setValue(line, next string) (string, error) {
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
	}
	return m[1] + m[2] + ":" + m[3] + next + line[len(body(line)):], nil
}
```

Hinweis zum Fall „comment“: `revision: 1 # pinned by hand` liest YAML als `1`; ein Neuschreiben des Werts hätte den Kommentar verloren, darum Ablehnung.

- [ ] **Step 4: Grün und Coverage**

Run: `go test ./internal/brain/wiki/ -run TestRepinText -count=1` → `ok`. Coverage von `repin.go` im echten Baum, jede Funktion 100.0 %. Fehlt ein Zweig (etwa `len(old) < 2` bei einem Wert `"`), einen Fall in `TestRepinTextRefusesWhatItCannotEditSafely` ergänzen: `"lone quote": "---\nsources:\n  - doc_id: 01A\n    content_hash: \"\n    revision: 1\n---\n"`.

- [ ] **Step 5: Mutationsrunde**

Mindestens: `body(lines[0]) != "---"` → `false`; `end == len(lines)` → `false`; `strings.TrimSpace(m[1]) != ""` → `false`; je ein Glied der `stop`-Schleifenbedingung streichen (3); `len(m[1]) == entryIndent` streichen; `len(m[1]) == entryIndent+2` streichen; `seen[docID]` → `false`; `!seen[docID]`-Schlussprüfung streichen; `strings.Contains(old, " #")` → `false`; `old[len(old)-1] != old[0]` → `false`; Quote-Erhalt (`next = string(old[0]) + …`) streichen; Zeilenende-Erhalt (`+ line[len(body(line)):]`) durch `+ "\n"` ersetzen. Ausgaben `t2-mut-<n>.txt`.

- [ ] **Step 6: Commit**

`msg-t2.txt`:

```
feat(wiki): edit the values of a source pin in place

repinText changes content_hash and revision of the sources entries it is
given and leaves every other byte of the page alone, quotes and line ends
included. A YAML round trip would refold and requote every page it
touched; a form it cannot edit that way is refused.
```

---

### Task 3: `wiki.Repin`

**Files:**
- Modify: `internal/brain/wiki/repin.go`
- Modify: `internal/brain/wiki/repin_test.go`

**Interfaces:**
- Consumes: aus Task 1 `loadSources`, `(*sourceWorld).drift`, `(*sourceWorld).fileHash`, `sourcePage`, `pin`, `registerName`, `readSource`, `identity.HashBytes`; aus Task 2 `repinText`, `pinValue`.
- Produces:
  - `type RowAdvance struct { Path string; From, To int }`
  - `type RepinResult struct { Rows []RowAdvance; Pages []string }` (Pages wiki-relativ, sortiert; Rows nach Path sortiert)
  - `type RepinRefusal struct { Findings []SourceFinding }` mit `Error() string`
  - `func Repin(projectRoot, wikiRoot, scope string) (RepinResult, error)`
  - Seam `var writeSource = os.WriteFile`

- [ ] **Step 1: Tests schreiben**

In `repin_test.go` ergänzen (Imports: `errors`, `os`, `path/filepath`, `reflect`, `identity`):

```go
func repin(t *testing.T, root string) RepinResult {
	t.Helper()
	got, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// The chain the wiki has: a doc, a topic citing it, a synthesis citing the
// topic. The doc changed; one run moves all three rows by exactly one, pins
// both pages, and leaves the check silent.
func TestRepinCarriesAChangeThroughThreeLinks(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	topic := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	synthesis := page("S", testPin{"docs/wiki/topics/t.md", "01T", hashOf(topic), 5})
	root := project(t, map[string]string{
		"docs/de/a.md":             now,
		"docs/wiki/topics/t.md":    topic,
		"docs/wiki/syntheses/s.md": synthesis,
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3},
		identity.Identity{DocID: "01T", Relative: "docs/wiki/topics/t.md", ContentHash: hashOf(topic), Revision: 5},
		identity.Identity{DocID: "01S", Relative: "docs/wiki/syntheses/s.md", ContentHash: hashOf(synthesis), Revision: 9},
	)
	got := repin(t, root)
	want := RepinResult{
		Rows:  []RowAdvance{{"docs/de/a.md", 3, 4}, {"docs/wiki/topics/t.md", 5, 6}},
		Pages: []string{"syntheses/s.md", "topics/t.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result %+v\nwant %+v", got, want)
	}
	if findings := drift(t, root); len(findings) != 0 {
		t.Fatalf("after repin: %+v", findings)
	}
	register, err := identity.ReadIdentities(filepath.Join(root, "_identities.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	// The synthesis is cited by nobody: its row stays as it was although
	// its pin changed it.
	if row := register["docs/wiki/syntheses/s.md"]; row.Revision != 9 || row.ContentHash != hashOf(synthesis) {
		t.Fatalf("uncited row moved: %+v", row)
	}
	if again := repin(t, root); len(again.Rows)+len(again.Pages) != 0 {
		t.Fatalf("a second run changed %+v", again)
	}
}

// After a reindex in the registry's checkout the rows are current and only
// the pins are behind: repin writes the pins and no row. Two pages cite the
// same file, so its hash is taken once for both.
func TestRepinMovesPinsToCurrentRows(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
		"docs/wiki/topics/u.md": page("U", testPin{"docs/de/a.md", "01A", hashOf(old), 2}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(now), Revision: 4})
	before := read(t, root, "_identities.tsv")
	got := repin(t, root)
	if len(got.Rows) != 0 || !reflect.DeepEqual(got.Pages, []string{"topics/t.md", "topics/u.md"}) {
		t.Fatalf("result %+v", got)
	}
	if read(t, root, "_identities.tsv") != before {
		t.Fatal("the register was rewritten")
	}
	if !strings.Contains(read(t, root, "docs/wiki/topics/t.md"), "content_hash: \""+hashOf(now)+"\"\n    revision: 4\n") {
		t.Fatalf("page %s", read(t, root, "docs/wiki/topics/t.md"))
	}
}

// A page that changed by hand and is cited moves its own row, and its
// citers follow; a CRLF page keeps CRLF and hashes folded.
func TestRepinFollowsAHandEditOfACitedCRLFPage(t *testing.T) {
	topicOld := strings.ReplaceAll(page("T"), "\n", "\r\n")
	topicNow := strings.ReplaceAll(page("T")+"more\n", "\n", "\r\n")
	root := project(t, map[string]string{
		"docs/wiki/topics/t.md":    topicNow,
		"docs/wiki/syntheses/s.md": strings.ReplaceAll(page("S", testPin{"docs/wiki/topics/t.md", "01T", hashOf(topicOld), 2}), "\n", "\r\n"),
	}, identity.Identity{DocID: "01T", Relative: "docs/wiki/topics/t.md", ContentHash: hashOf(topicOld), Revision: 2})
	got := repin(t, root)
	if !reflect.DeepEqual(got.Rows, []RowAdvance{{"docs/wiki/topics/t.md", 2, 3}}) {
		t.Fatalf("rows %+v", got.Rows)
	}
	synthesis := read(t, root, "docs/wiki/syntheses/s.md")
	if strings.Count(synthesis, "\r\n") != strings.Count(synthesis, "\n") {
		t.Fatalf("CRLF lost: %q", synthesis)
	}
	if findings := drift(t, root); len(findings) != 0 {
		t.Fatalf("after repin: %+v", findings)
	}
}

// A page with a foreign pin and an incomplete entry beside the stale one:
// only the stale entry changes.
func TestRepinTouchesOnlyOwnCompleteEntries(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	text := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	text = strings.Replace(text, "---\n\n# T", "  - id: f\n    resource: brain://shared/kb/x.md\n    doc_id: 01F\n    content_hash: \"sha256:f\"\n    revision: 1\n  - id: i\n    resource: brain://project/x/docs/de/a.md\n---\n\n# T", 1)
	root := project(t, map[string]string{"docs/de/a.md": now, "docs/wiki/topics/t.md": text},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	repin(t, root)
	want := strings.Replace(text, "content_hash: \""+hashOf(old)+"\"\n    revision: 3", "content_hash: \""+hashOf(now)+"\"\n    revision: 4", 1)
	if got := read(t, root, "docs/wiki/topics/t.md"); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// What repin may not fix stops it before any write: the findings 1-3 and 6,
// and a page it cannot edit safely, even when another page is fixable.
func TestRepinRefusesBeforeItWrites(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	fixable := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	cases := map[string]string{
		"row missing":  page("U", testPin{"docs/de/new.md", "01N", "sha256:n", 1}),
		"not editable": strings.Replace(page("U", testPin{"docs/de/a.md", "01A", hashOf(old), 3}), "revision: 3", "revision: 3 # hand", 1),
	}
	for name, other := range cases {
		root := project(t, map[string]string{"docs/de/a.md": now, "docs/wiki/topics/t.md": fixable, "docs/wiki/topics/u.md": other},
			identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
		register := read(t, root, "_identities.tsv")
		_, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
		if err == nil {
			t.Fatalf("%s: no refusal", name)
		}
		if read(t, root, "docs/wiki/topics/t.md") != fixable || read(t, root, "_identities.tsv") != register {
			t.Fatalf("%s: something was written", name)
		}
	}
	var refusal *RepinRefusal
	root := project(t, map[string]string{"docs/de/a.md": now, "docs/wiki/topics/t.md": fixable, "docs/wiki/topics/u.md": cases["row missing"]},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	_, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
	if !errors.As(err, &refusal) || len(refusal.Findings) != 1 || refusal.Findings[0].Rule != RuleRowMissing {
		t.Fatalf("err %v", err)
	}
	if refusal.Error() != "1 finding(s) repin does not fix" {
		t.Fatalf("message %q", refusal.Error())
	}
}

// Nothing to repin: no register, or nothing behind.
func TestRepinWithNothingToDo(t *testing.T) {
	doc := "# A\n"
	empty := project(t, map[string]string{"docs/wiki/topics/t.md": page("T")})
	current := project(t, map[string]string{"docs/de/a.md": doc, "docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(doc), 2})},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 2})
	for _, root := range []string{empty, current} {
		if got := repin(t, root); len(got.Rows)+len(got.Pages) != 0 {
			t.Fatalf("result %+v", got)
		}
	}
}

// A write that fails stops the run and names what was written before it.
func TestRepinReportsAWriteFailure(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	boom := errors.New("disk full")
	saved := writeSource
	writeSource = func(path string, data []byte, perm os.FileMode) error {
		if filepath.Base(path) == "_identities.tsv" {
			return boom
		}
		return os.WriteFile(path, data, perm)
	}
	defer func() { writeSource = saved }()
	got, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
	if !errors.Is(err, boom) || !reflect.DeepEqual(got.Pages, []string{"topics/t.md"}) || len(got.Rows) != 0 {
		t.Fatalf("result %+v, err %v", got, err)
	}
	// A page that cannot be written stops the run before the register.
	fresh := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	register := read(t, fresh, "_identities.tsv")
	writeSource = func(string, []byte, os.FileMode) error { return boom }
	got, err = Repin(fresh, filepath.Join(fresh, "docs", "wiki"), testScope)
	if !errors.Is(err, boom) || len(got.Pages)+len(got.Rows) != 0 || read(t, fresh, "_identities.tsv") != register {
		t.Fatalf("result %+v, err %v", got, err)
	}
}

// A cited file that reads for the check and fails when the plan reads it
// again stops the run before any write.
func TestRepinReportsAFailureWhilePlanning(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	text := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	root := project(t, map[string]string{"docs/de/a.md": now, "docs/wiki/topics/t.md": text},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	boom := errors.New("gone between two reads")
	reads := 0
	saved := readSource
	readSource = func(path string) ([]byte, error) {
		if filepath.Base(path) == "a.md" {
			reads++
			if reads > 1 {
				return nil, boom
			}
		}
		return os.ReadFile(path)
	}
	defer func() { readSource = saved }()
	if _, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope); !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
	if read(t, root, "docs/wiki/topics/t.md") != text {
		t.Fatal("a page was written")
	}
}

// Load and drift failures reach the caller unchanged.
func TestRepinReportsAReadFailure(t *testing.T) {
	root := project(t, map[string]string{"docs/de/a.md": "# A\n"},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: "sha256:a", Revision: 1})
	if _, err := Repin(root, filepath.Join(root, "docs", "missing"), testScope); err == nil {
		t.Fatal("a wiki root that does not exist must fail")
	}
	writeAt(t, root, "docs/wiki/topics/t.md", page("T", testPin{"docs/de/a.md", "01A", "sha256:a", 1}))
	boom := errors.New("denied")
	saved := readSource
	readSource = func(path string) ([]byte, error) {
		if filepath.Base(path) == "a.md" {
			return nil, boom
		}
		return os.ReadFile(path)
	}
	defer func() { readSource = saved }()
	if _, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope); !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
}
```

- [ ] **Step 2: Rot sehen**

Stubs: `type RowAdvance struct{ Path string; From, To int }`, `type RepinResult struct{ Rows []RowAdvance; Pages []string }`, `type RepinRefusal struct{ Findings []SourceFinding }` mit `func (r *RepinRefusal) Error() string { return "" }`, `var writeSource = os.WriteFile`, `func Repin(projectRoot, wikiRoot, scope string) (RepinResult, error) { return RepinResult{}, nil }`.
Run: `go test ./internal/brain/wiki/ -run TestRepin -count=1`
Expected: `--- FAIL` in `CarriesAChangeThroughThreeLinks`, `MovesPinsToCurrentRows`, `FollowsAHandEditOfACitedCRLFPage`, `TouchesOnlyOwnCompleteEntries`, `RefusesBeforeItWrites`, `ReportsAWriteFailure`, `ReportsAFailureWhilePlanning`, `ReportsAReadFailure`. `WithNothingToDo` ist gegen den Stub grün; die Mutationsrunde belegt ihn.

- [ ] **Step 3: Implementieren**

In `repin.go` ergänzen (Imports: `os`, `path/filepath`, `slices`, `identity`):

```go
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
			return RepinResult{}, fmt.Errorf("%s: %w", p.relative, err)
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
	if len(p.pins) > 0 {
		updates := map[string]pinValue{}
		for _, pn := range p.pins {
			digest, err := r.finalHash(pn.path)
			if err != nil {
				return "", err
			}
			updates[pn.source.DocID] = pinValue{hash: digest, revision: r.finalRevision(pn.path, digest)}
		}
		var err error
		if text, err = repinText(text, updates); err != nil {
			return "", err
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
```

Prüfpunkte gegen die Spec beim Implementieren:
- `r.hash` enthält genau die zitierten Pfade (nur `finalHash` füllt es, nur über Pins), also rückt nur eine zitierte Zeile vor (Entscheidung 2).
- `finalRevision` misst gegen `r.world.register`, das Register wie gelesen: höchstens +1 je Lauf.
- Seiten werden in Lesereihenfolge (`world.pages`, lexikalisch) geschrieben, `result.Pages` danach sortiert.

- [ ] **Step 4: Grün und Coverage**

Run: `go test ./internal/brain/wiki/ -count=1` → `ok`; Coverage von `repin.go` und `sources.go` je Funktion 100.0 % (im echten Baum gemessen).

- [ ] **Step 5: Mutationsrunde**

Mindestens: `f.Rule != RuleRegisterStale && f.Rule != RulePinStale` je Glied streichen (2); `len(blocking) > 0` → `false`; Memo in `final` (`done`-Zweig) streichen — muss bauen und darf nur Laufzeit ändern, dann als äquivalent begründen oder töten; `isPage`-Zweig → immer `fileHash` (CRLF-Test muss sterben); `digest != row.ContentHash` in `finalRevision` → `true`; `r.text[...] == string(p.raw)` → `false`; `digest == row.ContentHash` in `write` → `false`; `len(rows) > 0` → `true`; Reihenfolge Seiten/Register tauschen. Ausgaben `t3-mut-<n>.txt`.

- [ ] **Step 6: Commit**

`msg-t3.txt`:

```
feat(wiki): repin the register and the pins of a branch

Repin moves the register rows of the cited files and every pin of the
project's own scope to the files as they are, through the whole chain of
citing pages, in the tree it is given rather than the registry's
checkout. A row moves by one at most per run, an uncited row stays, and
everything it may not fix is refused before the first write.
```

---

### Task 4: Stack `wiki-sources` in der Konfiguration

**Files:**
- Modify: `internal/verify/schema.go` (`StackNames` Zeile 28–31, `ParseConfig` Zeile 120–137, `parseStack` Zeile 248–253)
- Modify: `internal/verify/effective.go` (Zeile 44–48)
- Modify: `internal/verify/schema_test.go` (Zeilen 31–32, 184, 296–330)

**Interfaces:**
- Consumes: nichts aus Task 1–3.
- Produces: `eff.Config.Stacks["wiki-sources"]["lint"].Lane.Off` ist `true` bei `[verify.wiki-sources] lint = false` und bei `[modules] brain = false`.

- [ ] **Step 1: Tests anpassen und ergänzen (rot)**

In `schema_test.go`:
- Zeile 184: die erwartete Liste wird `"go,python,typescript,vue,svelte,css,html,gdscript,cpp,shell,sql,rust,wiki,wiki-sources,project"`.
- In der Tabelle der Ablehnungen (Zeilen 31–32) zwei Zeilen ergänzen:

```go
		{"[verify.wiki-sources]\nlint = \"x\"", "[verify.wiki-sources].lint can only be false"},
		{"[verify.wiki-sources]\ntest = false", "[verify.wiki-sources].lint can only be false"},
```

- Neuer Test:

```go
// The source lane has its own switch, so it can go without the bundle lint
// going too, and the bundle lint's switch leaves it alone.
func TestTheSourceLaneHasItsOwnSwitch(t *testing.T) {
	cfg, err := parse(t, "[verify.wiki-sources]\nlint = false")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Stacks["wiki-sources"]["lint"].Lane.Off || cfg.Stacks["wiki"]["lint"].Lane.Off {
		t.Fatalf("stacks %+v", cfg.Stacks)
	}
	cfg, err = parse(t, "[verify.wiki]\nlint = false")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Stacks["wiki-sources"]["lint"].Lane.Off {
		t.Fatal("[verify.wiki] lint = false switched the source lane off")
	}
}
```

- Im bestehenden Brain-Test (ab Zeile 296) je Fall zusätzlich prüfen:

```go
		if !cfg.Stacks["wiki-sources"]["lint"].Lane.Off {
			t.Fatalf("%q: brain = false must turn lint/wiki-sources off", src)
		}
```

  und im Fall „brain an“ (Zeile ~327) zusätzlich `if _, ok := cfg.Stacks["wiki-sources"]; ok { t.Fatalf(...) }`.

- In `effective_test.go` (oder wo `Effective` getestet wird; per `grep -n "func TestEffective\|BuildEffective\|Resolve(" internal/verify/*_test.go` finden) einen Test, der zeigt, dass `eff.Stacks` keinen Eintrag `wiki-sources` bekommt, auch mit `[verify.wiki-sources] lint = false`:

```go
func TestTheSourceStackHasNoLanesToLayer(t *testing.T) {
	cfg, err := parse(t, "[verify.wiki-sources]\nlint = false")
	if err != nil {
		t.Fatal(err)
	}
	eff, err := Resolve(cfg, presetsFor(t), detect.Facts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := eff.Stacks["wiki-sources"]; ok || slices.Contains(eff.Active, "wiki-sources") {
		t.Fatalf("stacks %+v, active %v", eff.Stacks, eff.Active)
	}
}
```

  Der Test gehört nach `internal/verify/effective_test.go` (dort stehen `presetsFor` und die `Resolve`-Tests; `parse` kommt aus `schema_test.go` im selben Paket).

Run: `go test ./internal/verify/ -count=1` → `--- FAIL` (unbekannter Stack, falsche Liste, Brain-Schalter).

- [ ] **Step 2: Implementieren**

`schema.go`:

```go
func StackNames() []string {
	return []string{"go", "python", "typescript", "vue", "svelte", "css", "html",
		"gdscript", "cpp", "shell", "sql", "rust", "wiki", "wiki-sources", "project"}
}
```

In `parseStack` die Bedingung für die Wiki-Stacks erweitern:

```go
		case slices.Contains(Kinds(), key):
			// The wiki stacks have no tools of their own; each one lane can
			// only be switched off.
			if (stack == "wiki" || stack == "wiki-sources") && (key != "lint" || v != false) {
				return fmt.Errorf("[verify.%s].lint can only be false", stack)
			}
```

In `ParseConfig`:

```go
	if !modules.Brain {
		// The wiki lanes are the brain module's lanes. Off, each takes the
		// very value `[verify.<stack>] lint = false` leaves, so the edit lane,
		// the check chain and the stop gate ask no second question.
		for _, stack := range []string{"wiki", "wiki-sources"} {
			if cfg.Stacks[stack] == nil {
				cfg.Stacks[stack] = map[string]Override{}
			}
			cfg.Stacks[stack]["lint"] = laneOff()
		}
	}
```

`effective.go`:

```go
	for _, stack := range StackNames() {
		// The wiki lanes are the hooks' business; they have no tools to layer.
		if stack == "wiki" || stack == "wiki-sources" {
			continue
		}
```

- [ ] **Step 3: Grün**

Run: `go test ./internal/verify/ ./internal/hooks/ ./internal/cli/ -count=1 > <scratchpad>/t4.txt 2>&1` → alle `ok`. Fällt ein anderer Test, der die Stackliste oder den Fehlertext von `[verify.wiki]` festhält (`grep -rn "can only be false\|StackNames\|stacks are:" internal testdata`), ihn anpassen; ein Golden unter `testdata/` mit dem alten Text nicht von Hand ändern, sondern im Bericht nennen.

- [ ] **Step 4: Mutationsrunde**

`stack == "wiki-sources"` in `parseStack` streichen; die Schleife in `ParseConfig` auf `"wiki"` allein; die Ausnahme in `effective.go` auf `"wiki"` allein. Ausgaben `t4-mut-<n>.txt`.

- [ ] **Step 5: Commit**

`msg-t4.txt`:

```
feat(verify): give the wiki source lane a switch of its own

[verify.wiki-sources] lint = false turns the lane off without the
bundle lint, and [modules] brain = false turns off both, as it turned
off the bundle lint before.
```

---

### Task 5: Spur `lint/wiki-sources` in `loomux check` und `GateLanes`

**Files:**
- Modify: `internal/hooks/wikigate.go`
- Modify: `internal/hooks/wikigate_test.go`
- Modify: `internal/hooks/lanestates.go` (Zeile 49)
- Modify: `internal/hooks/lanestates_test.go`
- Modify: `internal/cli/check.go` (Zeile 232–234)
- Modify: `internal/cli/check_test.go`

**Interfaces:**
- Consumes: `wiki.SourceDrift`, `wiki.SourceFinding`, `wiki.Rule*` (Task 1); `eff.Config.Stacks["wiki-sources"]` (Task 4); vorhandene `stacksWithWiki`, `wikiDirFor`, `config.ReadManifest`.
- Produces: `func hooks.WikiSourceJobs(eff verify.Effective, facts detect.Facts, root string, kinds []string) []verify.Job`; `func sourceLane(root, wikiRoot string, drift func(projectRoot, wikiRoot, scope string) ([]wiki.SourceFinding, error)) (string, error)`.

- [ ] **Step 1: Tests schreiben (rot)**

In `wikigate_test.go` (Imports ergänzen: `identity`, `wiki`, `fmt`):

```go
// sourceProject is wikiProject with a register and a cited doc whose row
// is behind it: the lane must be red.
func sourceProject(t *testing.T, manifest string) string {
	t.Helper()
	old, now := "# A\n", "# A changed\n"
	pinned := fmt.Sprintf("---\ntitle: T\ntype: concept\ndescription: d\nsources:\n  - id: a\n    resource: brain://project/x/docs/de/a.md\n    doc_id: 01A\n    content_hash: \"%s\"\n    revision: 3\n---\n\n# T\n", identity.HashBytes([]byte(old)))
	root, _ := wikiProject(t, manifest, pinned)
	if err := os.MkdirAll(filepath.Join(root, "docs", "de"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		filepath.Join("docs", "de", "a.md"): now,
		"_identities.tsv": identity.RenderIdentities(map[string]identity.Identity{
			"docs/de/a.md": {DocID: "01A", Relative: "docs/de/a.md", ContentHash: identity.HashBytes([]byte(old)), Revision: 3},
		}),
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// Its own lane key, so probation holds it apart from the bundle lint.
func TestWikiSourceJobsAreALaneOfTheirOwn(t *testing.T) {
	root := sourceProject(t, "")
	eff, facts := loadedFor(t, root)
	jobs := WikiSourceJobs(eff, facts, root, []string{"lint"})
	if len(jobs) != 1 || jobs[0].Name != "lint/wiki-sources" || verify.LaneKey(jobs[0]) != "lint/wiki-sources@." {
		t.Fatalf("jobs %+v", jobs)
	}
	out, err := jobs[0].Fn()
	if err == nil || !strings.Contains(out, "  • [wiki-sources:register-stale] docs/de/a.md: changed since its register row (revision 3); cited by page.md\n") ||
		!strings.Contains(out, "a human runs `loomux wiki repin`") {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// The same three conditions, with the lane's own switch, and the brain
// module's.
func TestWikiSourceJobsOnlyForLintAWikiAndTheirSwitch(t *testing.T) {
	root := sourceProject(t, "")
	eff, facts := loadedFor(t, root)
	if jobs := WikiSourceJobs(eff, facts, root, []string{"test"}); jobs != nil {
		t.Fatalf("jobs without lint: %+v", jobs)
	}
	plain := t.TempDir()
	plainEff, plainFacts := loadedFor(t, plain)
	if jobs := WikiSourceJobs(plainEff, plainFacts, plain, []string{"lint"}); jobs != nil {
		t.Fatalf("jobs without a wiki: %+v", jobs)
	}
	for _, manifest := range []string{"[verify.wiki-sources]\nlint = false\n", "[modules]\nbrain = false\n"} {
		off := sourceProject(t, manifest)
		offEff, offFacts := loadedFor(t, off)
		if jobs := WikiSourceJobs(offEff, offFacts, off, []string{"lint"}); jobs != nil {
			t.Fatalf("%q: jobs with the lane off: %+v", manifest, jobs)
		}
	}
	// The bundle lint's switch is not this lane's.
	on := sourceProject(t, "[verify.wiki]\nlint = false\n")
	onEff, onFacts := loadedFor(t, on)
	if jobs := WikiSourceJobs(onEff, onFacts, on, []string{"lint"}); len(jobs) != 1 {
		t.Fatalf("jobs %+v", jobs)
	}
}

// A wiki laid out elsewhere is found where the manifest says.
func TestWikiSourceJobsFollowTheLayout(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"kb\"\n")
	if err := os.MkdirAll(filepath.Join(root, "kb"), 0o755); err != nil {
		t.Fatal(err)
	}
	pinned := "---\ntitle: T\ntype: concept\ndescription: d\nsources:\n  - id: a\n    resource: brain://project/x/a.md\n    doc_id: 01A\n    content_hash: \"sha256:old\"\n    revision: 3\n---\n"
	for name, body := range map[string]string{
		filepath.Join("kb", "t.md"): pinned,
		"a.md":                      "# A\n",
		"_identities.tsv":           "doc_id\tpfad\tcontent_hash\trevision\n01A\ta.md\tsha256:old\t3\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	eff, facts := loadedFor(t, root)
	jobs := WikiSourceJobs(eff, facts, root, []string{"lint"})
	if len(jobs) != 1 {
		t.Fatalf("jobs %+v", jobs)
	}
	if out, err := jobs[0].Fn(); err == nil || !strings.Contains(out, "[wiki-sources:register-stale] a.md:") {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// The report names, per rule present, what to do; a lane that cannot read
// fails rather than stays silent; no findings is green.
func TestSourceLaneReport(t *testing.T) {
	findings := []wiki.SourceFinding{
		{Rule: wiki.RuleRowMissing, Page: "t.md", Message: "m1"},
		{Rule: wiki.RuleCycle, Page: "t.md", Message: "m2"},
		{Rule: wiki.RulePinStale, Page: "t.md", Message: "m3"},
	}
	out, err := sourceLane(t.TempDir(), "wiki", func(string, string, string) ([]wiki.SourceFinding, error) { return findings, nil })
	if err == nil || err.Error() != "the wiki sources have 3 finding(s)" {
		t.Fatalf("err %v", err)
	}
	for _, want := range []string{
		"  • [wiki-sources:row-missing] t.md: m1\n",
		"  row-missing: repin adds no register row; a newly cited file gets one from `loomux reindex` in the registry's checkout (see `loomux wiki repin` in the CLI reference)\n",
		"  cycle: pages that pin each other can never both be current; drop one of the pins\n",
		"  register-stale, pin-stale: check the citing pages; then a human runs `loomux wiki repin`\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("out %q lacks %q", out, want)
		}
	}
	if strings.Contains(out, "source-missing, doc-id") {
		t.Fatalf("a hint for a rule not present: %q", out)
	}
	boom := errors.New("denied")
	if _, err := sourceLane(t.TempDir(), "wiki", func(string, string, string) ([]wiki.SourceFinding, error) { return nil, boom }); !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
	if out, err := sourceLane(t.TempDir(), "wiki", func(string, string, string) ([]wiki.SourceFinding, error) { return nil, nil }); out != "" || err != nil {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// The scope comes from the manifest; without one the drift is asked with
// none, and a manifest that does not read counts as none, as the lint reads
// it.
func TestSourceLaneTakesTheScopeFromTheManifest(t *testing.T) {
	var asked []string
	spy := func(_ string, _ string, scope string) ([]wiki.SourceFinding, error) {
		asked = append(asked, scope)
		return nil, nil
	}
	declared := t.TempDir()
	writeManifest(t, declared, "[area]\nscope = \"project/x\"\n")
	broken := t.TempDir()
	writeManifest(t, broken, "[area\n")
	for _, root := range []string{declared, t.TempDir(), broken} {
		if _, err := sourceLane(root, "wiki", spy); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(asked, []string{"project/x", "", ""}) {
		t.Fatalf("asked %q", asked)
	}
}
```

Imports zusätzlich `reflect`. In `lanestates_test.go` die Erwartung von `TestGateLanesNameTheWikiAndTheProjectLanes` (Zeile 42–48) erweitern; die Spur steht bei jedem Projekt mit Wiki in der Liste, auch ohne Register (sie ist dort grün), und `-` sortiert vor `@`:

```go
func TestGateLanesNameTheWikiAndTheProjectLanes(t *testing.T) {
	root, _ := wikiProject(t, "[verify.project]\nlint = \"echo hi\"\n", cleanPage)
	got, err := GateLanes(root)
	if err != nil || !slices.Equal(got, []string{"lint/project@.", "lint/wiki-sources@.", "lint/wiki@."}) {
		t.Fatalf("%v %v", got, err)
	}
	// Its own switch takes it out of the list and leaves the bundle lint.
	off, _ := wikiProject(t, "[verify.wiki-sources]\nlint = false\n", cleanPage)
	if got, err := GateLanes(off); err != nil || !slices.Equal(got, []string{"lint/wiki@."}) {
		t.Fatalf("%v %v", got, err)
	}
}
```

In `internal/cli/check_test.go` den Ende-zu-Ende-Fall über die echte Befehlskette:

```go
// The lane runs in `loomux check lint` beside the bundle lint, red on a
// stale row, green once the register matches.
func TestCheckLintRunsTheSourceLane(t *testing.T) {
	root := t.TempDir()
	old, now := "# A\n", "# A changed\n"
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"docs/wiki\"\n")
	writeFile(t, filepath.Join(root, "docs", "de", "a.md"), now)
	writeFile(t, filepath.Join(root, "docs", "wiki", "t.md"), "---\ntitle: T\ntype: concept\ndescription: d\nsources:\n  - id: a\n    resource: brain://project/x/docs/de/a.md\n    doc_id: 01A\n    content_hash: \""+identity.HashBytes([]byte(old))+"\"\n    revision: 3\n---\n\n# T\n")
	register := func(hash string) {
		writeFile(t, filepath.Join(root, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\n01A\tdocs/de/a.md\t"+hash+"\t3\n")
	}
	register(identity.HashBytes([]byte(old)))
	code, out, _ := run("check", "lint", "--root", root)
	if code == 0 || !strings.Contains(out, "lint/wiki-sources: failed") || !strings.Contains(out, "[wiki-sources:register-stale] docs/de/a.md") {
		t.Fatalf("code %d, out %q", code, out)
	}
	register(identity.HashBytes([]byte(now)))
	writeFile(t, filepath.Join(root, "docs", "wiki", "t.md"), "---\ntitle: T\ntype: concept\ndescription: d\nsources:\n  - id: a\n    resource: brain://project/x/docs/de/a.md\n    doc_id: 01A\n    content_hash: \""+identity.HashBytes([]byte(now))+"\"\n    revision: 3\n---\n\n# T\n")
	if code, out, _ := run("check", "lint", "--root", root, "-v"); code != 0 || !strings.Contains(out, "lint/wiki-sources: ok") {
		t.Fatalf("code %d, out %q", code, out)
	}
}
```

Wie `lint/wiki` im Bericht steht („`lint/wiki: ok [in-process]`“, siehe Ausgabe von `check lint -v` am Anfang dieser Arbeit), so steht auch die neue Spur; die Zeichenfolgen `: failed` und `: ok` gegen die echte Ausgabe des ersten roten Laufs prüfen und bei Abweichung den Test an die Ausgabe anpassen, nicht umgekehrt. Ist `writeFile` in `wiki_test.go` mit anderer Signatur definiert, dessen Form nehmen. Falls `check lint` in einem Projekt ohne Go-Stack an einer anderen Spur scheitert (`go`-Preset ohne Go), im Test zusätzlich prüfen, dass die Spur `lint/wiki-sources` rot bzw. ok berichtet, und den Exit-Code nur am Vergleich rot/grün der Spur festmachen.

Run: `go test ./internal/hooks/ ./internal/cli/ -run "WikiSource|SourceLane|GateLanes|CheckLintRunsTheSourceLane" -count=1` → Build-Fehler `undefined: WikiSourceJobs`; mit Stubs (`WikiSourceJobs` gibt `nil`, `sourceLane` gibt `"", nil`) `--- FAIL`.

- [ ] **Step 2: Implementieren**

`wikigate.go` (Imports: `config`):

```go
// WikiSourceJobs is the lane of the wiki's sources: the pins of the
// project's own scope against the register at the root, and the register
// against the files they cite. Its own stack, so its own lane key: in
// probation apart from the bundle lint, and switched off on its own with
// [verify.wiki-sources] lint = false.
//
// In `loomux check` and the gate's lane list, never at the turn end: only a
// human turns it green (`loomux wiki repin`), and a lane an agent cannot fix
// would hold every turn end for nothing. An agent meets it at the commit.
func WikiSourceJobs(eff verify.Effective, facts detect.Facts, root string, kinds []string) []verify.Job {
	if !slices.Contains(kinds, "lint") || eff.Config.Stacks["wiki-sources"]["lint"].Lane.Off {
		return nil
	}
	if !slices.Contains(stacksWithWiki(facts.Stacks, root), "wiki") {
		return nil
	}
	wikiRoot := filepath.Join(root, filepath.FromSlash(wikiDirFor(root)))
	return []verify.Job{{
		Name: "lint/wiki-sources", Kind: "lint", Stack: "wiki-sources", Area: ".", Origin: "in-process", Dir: root, After: -1,
		Fn: func() (string, error) { return sourceLane(root, wikiRoot, wiki.SourceDrift) },
	}}
}

// sourceHints says per rule what to do, in the order of the report.
var sourceHints = []struct{ rules, hint string }{
	{"row-missing", "repin adds no register row; a newly cited file gets one from `loomux reindex` in the registry's checkout (see `loomux wiki repin` in the CLI reference)"},
	{"source-missing, doc-id", "correct the pin by hand"},
	{"cycle", "pages that pin each other can never both be current; drop one of the pins"},
	{"register-stale, pin-stale", "check the citing pages; then a human runs `loomux wiki repin`"},
}

// sourceLane reports the findings the way lintBundle reports the bundle's,
// then one line per kind of finding present saying what to do. The drift
// arrives as a parameter for the failure no tree provokes.
func sourceLane(root, wikiRoot string, drift func(projectRoot, wikiRoot, scope string) ([]wiki.SourceFinding, error)) (string, error) {
	// A manifest that does not read counts as none, as the lint reads it;
	// the write barrier is what refuses a broken one.
	scope := ""
	if manifest, err := config.ReadManifest(root); err == nil {
		scope = manifest.Scope
	}
	findings, err := drift(root, wikiRoot, scope)
	if err != nil {
		return "", fmt.Errorf("checking the wiki sources: %w", err)
	}
	if len(findings) == 0 {
		return "", nil
	}
	var report strings.Builder
	present := map[string]bool{}
	for _, f := range findings {
		fmt.Fprintf(&report, "  • [wiki-sources:%s] %s: %s\n", f.Rule, f.Page, f.Message)
		present[f.Rule] = true
	}
	for _, h := range sourceHints {
		for _, rule := range strings.Split(h.rules, ", ") {
			if present[rule] {
				fmt.Fprintf(&report, "  %s: %s\n", h.rules, h.hint)
				break
			}
		}
	}
	return report.String(), fmt.Errorf("the wiki sources have %d finding(s)", len(findings))
}
```

`internal/cli/check.go`, nach Zeile 234:

```go
	jobs = append(jobs, hooks.WikiGateJobs(eff, facts, root, kinds)...)
	// check stop replays the stop gate, which leaves the source lane out.
	if request != "stop" {
		jobs = append(jobs, hooks.WikiSourceJobs(eff, facts, root, kinds)...)
	}
```

Dazu in `check_test.go` am Ende von `TestCheckLintRunsTheSourceLane` (nach dem roten Lauf, vor dem Nachziehen des Registers):

```go
	// The replay of the stop gate leaves the lane out, as the hook does.
	if _, out, _ := run("check", "stop", "--root", root); strings.Contains(out, "lint/wiki-sources") {
		t.Fatalf("check stop ran the source lane: %q", out)
	}
```

Läuft `check stop` in diesem Testprojekt (kein git) nicht bis zum Bericht, den Lauf in einem `git init`-Projekt fahren (Muster in `check_test.go` per grep nach `"check", "stop"` suchen) und das im Bericht nennen.

`internal/hooks/lanestates.go`, nach Zeile 49:

```go
	jobs = append(jobs, WikiGateJobs(eff, facts, root, kinds)...)
	jobs = append(jobs, WikiSourceJobs(eff, facts, root, kinds)...)
```

`internal/hooks/stop.go` bleibt unverändert (Spec-Nachtrag). Den Kommentar über `GateLanes` („the lanes of every kind as a check plans them, and the wiki's“) auf „and the wiki's two“ anpassen.

- [ ] **Step 3: Grün, Coverage, ganze Suite**

Run: `go test ./internal/hooks/ ./internal/cli/ -count=1 > <scratchpad>/t5.txt 2>&1` → `ok`. Stop-Gate-Tests müssen unverändert grün sein (sie dürfen die neue Spur nicht sehen). Coverage von `wikigate.go` je Funktion 100.0 %.

- [ ] **Step 4: Mutationsrunde**

`!slices.Contains(kinds, "lint")` streichen; `Lane.Off` → `false`; die Wiki-Bedingung streichen; `err == nil` beim Manifest → `true` (Scope eines kaputten Manifests); `len(findings) == 0` → `false`; `break` in der Hinweisschleife streichen (doppelte Hinweiszeile muss einen Test töten — sonst Test ergänzen, der genau eine Zeile `register-stale, pin-stale:` zählt bei beiden Regeln); `WikiSourceJobs` in `lanestates.go` streichen; in `check.go` streichen; `request != "stop"` → `true`. Ausgaben `t5-mut-<n>.txt`.

- [ ] **Step 5: Commit**

`msg-t5.txt`:

```
feat(check): run the wiki source check as a lane of its own

lint/wiki-sources runs in loomux check and stands in the gate's lane
list, with a key of its own so probation holds it apart from the bundle
lint. It stays out of the stop gate: only a human turns it green, and a
lane an agent cannot fix would hold every turn end.
```

---

### Task 6: `loomux wiki repin` und die Wächterregel

**Files:**
- Modify: `internal/cli/wikicmd.go`
- Modify: `internal/cli/wiki_test.go` (oder die Testdatei der `wiki`-Befehle; per `grep -ln "wikiRetype\|\"wiki\", \"retype\"" internal/cli/*_test.go` finden)
- Modify: `internal/hooks/guard.go` (`programWrites` ab Zeile 669, Ablehnungstext Zeile 320)
- Modify: `internal/hooks/guard_test.go` (`TestTheGuardRefusesCommandsThatWriteTheConfiguration` ab Zeile 826, `TestCheckToolNamesTheConfigurationReason` ab Zeile 1195)

**Interfaces:**
- Consumes: `wiki.Repin`, `wiki.RepinResult`, `wiki.RowAdvance`, `wiki.RepinRefusal` (Task 3); `wiki.Root(projectRoot string) string`; `projectRoot(flag string) (string, error)` in `internal/cli/wiki.go`; `config.ReadManifest`, `config.ErrNoManifest`.
- Produces: Befehl `loomux wiki repin [--root <pfad>]`; Wächter verweigert ihn Agenten.

- [ ] **Step 1: CLI-Tests (rot)**

```go
// repin from the command line: what it moved on stdout, nothing to do said
// as such, a refusal with exit 1 and the findings, a usage error with 2.
func TestWikiRepin(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"docs/wiki\"\n")
	writeFile(t, filepath.Join(root, "docs", "de", "a.md"), now)
	writeFile(t, filepath.Join(root, "docs", "wiki", "t.md"), "---\ntitle: T\ntype: concept\ndescription: d\nsources:\n  - id: a\n    resource: brain://project/x/docs/de/a.md\n    doc_id: 01A\n    content_hash: \""+identity.HashBytes([]byte(old))+"\"\n    revision: 3\n---\n")
	writeFile(t, filepath.Join(root, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\n01A\tdocs/de/a.md\t"+identity.HashBytes([]byte(old))+"\t3\n")
	code, out, errOut := run("wiki", "repin", "--root", root)
	if code != 0 || out != "repinned: t.md\nregister: docs/de/a.md revision 3 -> 4\n" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if code, out, _ := run("wiki", "repin", "--root", root); code != 0 || out != "nothing to repin\n" {
		t.Fatalf("second run: code %d, out %q", code, out)
	}
	writeFile(t, filepath.Join(root, "docs", "wiki", "u.md"), "---\ntitle: U\ntype: concept\ndescription: d\nsources:\n  - id: n\n    resource: brain://project/x/docs/de/new.md\n    doc_id: 01N\n    content_hash: \"sha256:n\"\n    revision: 1\n---\n")
	code, out, errOut = run("wiki", "repin", "--root", root)
	if code != 1 || out != "" || !strings.Contains(errOut, "  • [wiki-sources:row-missing] u.md: sources[0]: docs/de/new.md is not in the register\n") ||
		!strings.Contains(errOut, "loomux wiki repin: 1 finding(s) repin does not fix\n") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if code, _, _ := run("wiki", "repin", "extra"); code != 2 {
		t.Fatalf("a positional argument: code %d", code)
	}
	if code, _, _ := run("wiki", "repin", "--nope"); code != 2 {
		t.Fatalf("an unknown flag: code %d", code)
	}
}

// Without a manifest or a wiki there is nothing to repin; a manifest that
// does not read is an error.
func TestWikiRepinWithoutAnArea(t *testing.T) {
	bare := t.TempDir()
	if code, out, _ := run("wiki", "repin", "--root", bare); code != 0 || out != "nothing to repin\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	broken := t.TempDir()
	writeFile(t, filepath.Join(broken, ".loomux", "config.toml"), "[area\n")
	if code, _, errOut := run("wiki", "repin", "--root", broken); code != 1 || !strings.HasPrefix(errOut, "loomux wiki repin: ") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// A failure that is no refusal is one line and exit 1; what was written
// before it is still printed.
func TestWikiRepinReportsAFailure(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"docs/wiki\"\n")
	writeFile(t, filepath.Join(root, "docs", "wiki", "t.md"), "---\ntitle: T\n---\n")
	writeFile(t, filepath.Join(root, "_identities.tsv"), "doc_id\tpfad\nbroken\n")
	if code, _, errOut := run("wiki", "repin", "--root", root); code != 1 || !strings.HasPrefix(errOut, "loomux wiki repin: ") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}
```

Zusätzlich die Meldung bei leerem `wiki`: `run("wiki")` → `stderr` enthält `needs init, types, retype or repin`. Imports: `identity`.

- [ ] **Step 2: Wächtertests (rot)**

In `TestTheGuardRefusesCommandsThatWriteTheConfiguration` in die Liste `refused`:

```go
		// wiki repin declares the citing pages still correct, which is
		// a human's decision as an approval is.
		"loomux wiki repin",
		"loomux wiki repin --root .",
		"go run ./cmd/loomux wiki repin",
		"cd x && loomux wiki repin",
```

in die Liste der erlaubten Zeilen (ab Zeile 1140):

```go
		"loomux wiki repin --help",
		"loomux wiki repin -h",
		"loomux wiki retype --scope s --from a --to b",
		"loomux wiki types",
		"loomux wiki init --scope s",
		"loomux wiki",
```

und in `TestCheckToolNamesTheConfigurationReason`:

```go
	got = checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux wiki repin"}, config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "wiki repin declares the citing pages still correct") }) {
		t.Fatalf("wiki repin reasons %v", got)
	}
```

Run: `go test ./internal/hooks/ -run "TestTheGuardRefusesCommandsThatWriteTheConfiguration|TestCheckToolNamesTheConfigurationReason" -count=1` → `--- FAIL` (wird nicht verweigert).

- [ ] **Step 3: CLI implementieren**

`wikicmd.go`:

```go
// wikiCommand is `loomux wiki init|types|retype|repin`: the three wiki tools
// of the Python reference -- `brain wiki init`, `brain types` and `brain
// retype` -- and repin, which the reference does not have.
```

Im `switch`: `case "repin": return wikiRepin(args[1:], stdout, stderr)`; Meldung bei leerem Aufruf: `error: 'loomux wiki' needs init, types, retype or repin`.

```go
// wikiRepin is `loomux wiki repin`: the register and the pins of the tree at
// --root moved to the files as they are (wiki.Repin). Unlike its neighbours
// it reads no registry: it works on the tree it is given, a linked worktree
// included. The guard keeps it from agents.
func wikiRepin(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("wiki repin", flag.ContinueOnError)
	flags.SetOutput(stderr)
	rootFlag := flags.String("root", "", "project root; the working directory when empty")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "error: unrecognized arguments: %s\n", flags.Arg(0))
		return 2
	}
	fail := func(err error) int {
		fmt.Fprintf(stderr, "loomux wiki repin: %v\n", err)
		return 1
	}
	root, err := projectRoot(*rootFlag)
	if err != nil {
		return fail(err)
	}
	scope := ""
	manifest, err := config.ReadManifest(root)
	switch {
	case err == nil:
		scope = manifest.Scope
	case !errors.Is(err, config.ErrNoManifest):
		return fail(err)
	}
	result, err := wiki.Repin(root, wiki.Root(root), scope)
	for _, page := range result.Pages {
		fmt.Fprintf(stdout, "repinned: %s\n", page)
	}
	for _, row := range result.Rows {
		fmt.Fprintf(stdout, "register: %s revision %d -> %d\n", row.Path, row.From, row.To)
	}
	var refusal *wiki.RepinRefusal
	if errors.As(err, &refusal) {
		for _, f := range refusal.Findings {
			fmt.Fprintf(stderr, "  • [wiki-sources:%s] %s: %s\n", f.Rule, f.Page, f.Message)
		}
	}
	if err != nil {
		return fail(err)
	}
	if len(result.Pages)+len(result.Rows) == 0 {
		fmt.Fprintln(stdout, "nothing to repin")
	}
	return 0
}
```

Imports: `errors`. Vor dem Schreiben prüfen, ob `config.ReadManifest` ohne Datei wirklich `config.ErrNoManifest` (ggf. gewickelt) liefert: `grep -n "ErrNoManifest" internal/config/manifest.go`; wenn es ein anderes Signal gibt (etwa `nil, nil`), den `switch` darauf umstellen und den Test `TestWikiRepinWithoutAnArea` behalten.

- [ ] **Step 4: Wächter implementieren**

`guard.go`, in `programWrites` vor dem schließenden `}` des `switch`:

```go
	case "wiki":
		// repin declares every page citing a changed source still correct,
		// an approval for all of them at once; init, types and retype do
		// not decide that. A lone --help or -h only reads.
		return len(args) > 1 && args[1] == "repin" &&
			(len(args) != 3 || (args[2] != "--help" && args[2] != "-h"))
```

Ablehnungstext (Zeile 320) erweitern, sodass er den Test-Teilstring enthält:

```go
					reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads, merge-hook install and remove write executable hooks into repositories, convert and fetch write into an area's inbox, which the write barrier keeps from agents, and wiki repin declares the citing pages still correct; a human runs them. An agent proposes a change with `loomux config set|unset … --propose`, which a human applies, and names `loomux wiki repin` for the human to run")
```

Danach `grep -rn "convert and fetch write into an area's inbox, which" internal docs testdata flows` — jede Stelle (auch aufgenommene `stderr`-Erwartungen unter `testdata/`; ein Golden dort nicht von Hand ändern, sondern im Bericht nennen), die den alten Wortlaut festhält, anpassen (Tests: der Teilstring `convert and fetch write into an area's inbox` bleibt enthalten).

Prüfen, dass `programWrites` überhaupt für `wiki` erreicht wird: `programArgs` liefert die Wörter nach dem Programmnamen; `args[0]` ist `wiki`. Der Fall `loomux wiki repin --root .` hat `len(args) == 4` und ist damit verweigert; `loomux wiki repin --help` hat `len(args) == 3` und ist frei.

- [ ] **Step 5: Grün, Coverage**

Run: `go test ./internal/cli/ ./internal/hooks/ -count=1 > <scratchpad>/t6.txt 2>&1` → `ok`. Coverage `wikiRepin` und `programWrites` 100.0 %.

- [ ] **Step 6: Mutationsrunde**

`flags.NArg() > 0` → `false`; `!errors.Is(err, config.ErrNoManifest)` → `true`; `errors.As`-Zweig streichen; `len(...) == 0` → `true`; im Wächter `args[1] == "repin"` → `true` (dann muss `wiki types` als verweigert einen Test töten), `len(args) != 3` → `true`. Ausgaben `t6-mut-<n>.txt`.

- [ ] **Step 7: Commit**

`msg-t6.txt`:

```
feat(cli): add loomux wiki repin, and keep it from agents

wiki repin runs wiki.Repin on the tree at --root and prints what it moved
on, or the findings it refuses. The guard refuses it to an agent:
repinning declares every citing page still correct, which is a human's
decision as an approval is.
```

---

### Task 7: Doku, und der erste echte Lauf

> **Der Controller fährt diesen Task selbst, nicht ein Implementierer.** Sobald die Doku geändert ist, ist das Tor rot, und der einzige grüne Weg ist ein Befehl, den nur ein Mensch fährt. Ein Implementierer vor einem roten Tor ist genau der, der `_identities.tsv` oder Pins von Hand ändern würde; die Wächterregel aus Task 6 steckt bis nach dem Merge nicht im `bin/loomux.exe` des Haupt-Checkouts, das die Hooks rufen. Vor der Bitte um `repin`: `git diff --stat` zeigt nur `docs/**` und `README*`.

**Files:**
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md`
- Modify: `docs/wiki/topics/brain-maintenance.md`, `docs/wiki/topics/wiki-schicht.md`
- Modify: `README.md`, `README.de.md` (Befehlsliste neben `wiki retype`, Zeile ~213)
- Modify (durch `repin`, nicht von Hand): `_identities.tsv` und das Frontmatter der Seiten, die `docs/de/cli-reference.md`, `docs/de/configuration.md` (und ggf. `docs/de/hooks.md`) zitieren

**Interfaces:**
- Consumes: alles aus Task 1–6.
- Produces: nichts für spätere Tasks.

- [ ] **Step 1: Grep nach Stellen, die das Verhalten beschreiben**

Run: `grep -rn "lint/wiki\|wiki retype\|\[verify.wiki\]\|lint/wiki@\|wiki init, types\|needs init, types" --include=*.md . | grep -v "^./docs/.superpowers"` → Liste in den Bericht. Jede Stelle einzeln entscheiden: betrifft sie die neue Spur oder `repin`? Besonders:
- `docs/*/cli-reference.md` Abschnitt „Order“ bei `loomux check` (en Zeile ~74): `lint/wiki` ist nicht mehr „die eine Ausnahme“; beide In-Prozess-Spuren kommen zuletzt, `lint/wiki-sources` nach `lint/wiki`.
- `docs/*/cli-reference.md` beim Stop-Gate (en Zeile ~423) und bei `loomux check stop`: ausdrücklich ergänzen, dass `lint/wiki-sources` dort **nicht** läuft.
- `docs/*/configuration.md` bei `wiki knows only lint = false` (en Zeile ~183), bei den Spurschlüsseln (en Zeile ~549: „The project-wide lanes are `lint/wiki@.` …“) und beim Modulschalter `brain` (en Zeile ~612).

- [ ] **Step 2: Texte schreiben**

`docs/en/cli-reference.md`, nach dem Abschnitt `loomux wiki retype`:

```markdown
#### `loomux wiki repin [--root <path>]`
Moves the register and the source pins of the tree at `--root` (default: the working directory) to the files as they are: each register row of a cited file whose hash changed gets the new hash and its revision plus one, and every `sources:` entry of the project's own scope (`brain://<scope>/…`) gets its row's hash and revision. A pinned page's own hash changes with its pins, so the run follows the chain of citing pages to the end. A row nobody cites stays as it is; a row moves by one at most per run.

- **What it declares**: whoever runs it says that the pages citing a changed file are still correct. It is `approve --reject` for every such page at once, without a case. The guard refuses it to an agent; an agent checks the pages and names the command.
- **The tree, not the registry**: it reads and writes `_identities.tsv` at the project root and the pages of the project's wiki, never the registry's checkout, so it works in a linked worktree.
- **Edits in place**: only the values of `content_hash:` and `revision:` change; quotes, line ends and every other byte stay. A page whose entry it cannot edit that way (a flow list, a missing line, a doc_id twice, a comment or block value) is refused.
- **Refusals** (exit `1`, nothing written): the findings `row-missing`, `source-missing`, `doc-id` and `cycle` of the [source lane](#…) are printed as the lane prints them, then `loomux wiki repin: <n> finding(s) repin does not fix`. A newly cited file needs its row from `loomux reindex` first.
- **Output**: `repinned: <page>` per page, `register: <path> revision <old> -> <new>` per row, or `nothing to repin`. Without `[area]`, a wiki or a register at the root it does nothing.
- **After a merge conflict** in `_identities.tsv` or in pins: run it again.
- **Exit codes**: `0`; `1` for a refusal or an error; `2` for a usage error.
```

Den Anker `(#…)` auf den Abschnitt setzen, der die Spur beschreibt (Step 2, nächster Block). Bei `loomux check` (Abschnitt „Order“) ergänzen:

```markdown
  `lint/wiki-sources` comes after it, also in this process: the source pins of the project's own scope against `_identities.tsv` at the root, and that register against the files the pins cite. Findings: `row-missing`, `source-missing`, `doc-id`, `register-stale` (a cited file changed since its row), `pin-stale` (a pin behind its row), `cycle` (pages that pin each other). The last line of a red run says what to do per kind; for `register-stale` and `pin-stale` it is `loomux wiki repin`, run by a human. It reads the tree alone: pins of other areas, read-only areas (their register lives in the state directory) and projects without a register at the root are not judged. Not in the stop gate.
```

`docs/en/configuration.md` bei der Stack-Liste:

```markdown
- `wiki-sources` knows only `lint = false`, which switches off `lint/wiki-sources`, the check of the wiki's source pins in `loomux check`, and nothing else. `[verify.wiki] lint = false` leaves it running. It has a lane key of its own (`lint/wiki-sources@.`), so it serves its probation apart from `lint/wiki`.
```

bei den Spurschlüsseln: „The project-wide lanes are `lint/wiki@.`, `lint/wiki-sources@.` and `<kind>/project@.`.“; beim Modulschalter `brain`: „The lanes `lint/wiki` and `lint/wiki-sources` are off everywhere …“.

Die deutschen Fassungen (`docs/de/…`) mit demselben Inhalt in deutscher Prosa, Bezeichner unverändert, an den gleichnamigen Stellen; Ankerlinks auf die deutschen Überschriften.

`README.md` Befehlsliste: `loomux wiki repin [--root P]        # move the register and the source pins to the files as they are (a human runs it)`; `README.de.md` entsprechend deutsch.

Wiki (deutsch): In `topics/brain-maintenance.md` unter „Erkennung“ einen Absatz: Die Spur `lint/wiki-sources` erkennt beim Commit, was `reconcile` sonst erst im nächsten Lauf sähe; `loomux wiki repin` ist der Weg im Zweig, wenn der Mensch die Seiten für richtig erklärt; im Haupt-Checkout bleibt `reconcile`/`approve` der Weg mit Fall und Review. In `topics/wiki-schicht.md` unter „Die Wiki-Werkzeuge“ eine Zeile zu `wiki repin` und unter „Lint“ eine Zeile zur Spur. Kein `status`-Feld ändern; keine Seite löschen.

- [ ] **Step 3: Doku committen? Nein — erst der Lauf**

Die Änderungen an `docs/de/cli-reference.md`, `docs/de/configuration.md` (und ggf. `docs/de/hooks.md`) machen die Spur rot: Diese Dateien sind zitiert, loomux hat keine Armed-Datei. Vor dem Commit:

Run: `go run ./cmd/loomux check lint -v > <scratchpad>/t7-before.txt 2>&1` → erwartet: `lint/wiki-sources: failed` mit `register-stale` für jede geänderte zitierte Datei. Den Auszug in den Bericht.

- [ ] **Step 4: Ein Mensch fährt `repin`**

Der Agent fährt `repin` nicht (der Wächter verweigert es). Er liest die Seiten, die die geänderten Dateien zitieren (`grep -rln "docs/de/cli-reference.md\|docs/de/configuration.md" docs/wiki`), prüft, ob ihre Aussagen durch die Doku-Änderung falsch werden (die Änderung fügt nur die neue Spur und den neuen Befehl hinzu; erwartet: keine Seite wird falsch), nennt das Ergebnis und bittet den Nutzer, im Worktree auszuführen:

```bash
go -C "C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/admiring-zhukovsky-faf0b9" run ./cmd/loomux wiki repin
```

Danach liest der Agent die Wirkung selbst: `git status --short`, `git diff --stat`, und `go run ./cmd/loomux check lint -v > <scratchpad>/t7-after.txt 2>&1` → `lint/wiki-sources: ok`. Zusätzlich die Probe aus der Spec (`t7-sources.py`, `reg-vs-file.py` im Scratchpad) erneut: 0 Abweichungen bei den zitierten Zeilen.

- [ ] **Step 5: Commit**

```bash
git add docs/en/cli-reference.md docs/de/cli-reference.md docs/en/configuration.md docs/de/configuration.md docs/wiki README.md README.de.md _identities.tsv
git commit -F <scratchpad>/msg-t7.txt > <scratchpad>/commit-t7.log 2>&1
```

`msg-t7.txt`:

```
docs: describe the wiki source lane and loomux wiki repin

The register and the pins of the pages citing the changed German docs
were moved on with loomux wiki repin, the first run of the new lane on
this repository.
```

Der Commit läuft durch das pre-commit-Tor mit der neuen Spur; grün ist der Ende-zu-Ende-Beleg.

---

## Abschluss

- Ganze Suite und Tor einmal auf dem Endstand: `sh ci/gate.sh > <scratchpad>/gate-final.txt 2>&1`.
- Nach dem Merge: Die Hooks rufen `bin/loomux.exe` im Haupt-Checkout. Wächterregel und Spur greifen in Agenten-Sitzungen erst, wenn der Nutzer dort `master` holt und das Binary neu baut (`go -C "C:/Users/micro/Documents/#GIT/loomux" build -o bin/loomux.exe ./cmd/loomux`). In die Übergabe aufnehmen.
- Abschluss-Review des ganzen Zweigs gegen Spec und Plan.
- Commits nach Thema gruppieren (`release-pr`-Skill); Label `release:minor`, Changelog-Block:

```
## Changelog

### Added
- `lint/wiki-sources` in `loomux check`: the wiki's source pins of the project's own scope against `_identities.tsv` at the root, and that register against the files they cite. Own switch `[verify.wiki-sources] lint = false`; in probation until its first green run where `.loomux/armed.toml` exists; not in the stop gate.
- `loomux wiki repin`: moves the register rows of changed cited files and the pins of the citing pages to the files as they are, in the tree it is given. The guard refuses it to agents.
```
