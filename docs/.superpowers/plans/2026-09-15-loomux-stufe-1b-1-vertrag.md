# Vertrag für den Plan der Stufe 1b-1 (Kopf, Regeln, Schnittstellen)

> **Stand vor der Ausarbeitung.** Maßgeblich ist der Plan `2026-09-15-loomux-stufe-1b-1.md`. Wo ein Task davon
> abweicht, nennt er es („Abweichungen vom Vertrag") oder folgt einer Regel aus
> `2026-09-15-loomux-stufe-1b-1-rulings.md`. Überholt sind unter anderem:
> - die Fallzahl (71 statt 59);
> - die Usage-Wortlaute von Task 10 (argparse-Formen gemessen);
> - die MCP-Regel des Fake-qmd (nummerierte Snippets, `search_error`);
> - der Wortlaut von `ErrBaselineRed` und das Urteil `not compiled` (Task 13);
> - `NewHTTPSession` in Task 12 (`HTTPSession{URL: …}`);
> - die README-Pflege (nur Task 16).

Dieser Text ist die gemeinsame Grundlage aller Task-Entwürfe. Ein Bezeichner aus einem
Interfaces-Block wird nie umbenannt; ein zusätzlicher Helfer steht als Ergänzung unter
„Produces" des eigenen Tasks. Wo dieser Vertrag und die Spec sich widersprechen, gilt die Spec —
der Widerspruch ist dann ein Befund im Entwurf.

Spec: `docs/.superpowers/specs/2026-09-15-loomux-stufe-1b-1-design.md` (Commit `311d5b2`).
Fakten: `.superpowers/sdd/1b-1-facts/{moved-packages,search,status,loomux-now}.md`.
Quellen, nur lesen: `C:/Users/micro/Documents/#GIT/loomux-src/ub` (Go `pkg/…`, Python `src/brain/…`,
`tools/go_mutants.py`). Python der Referenz: `uv run --project C:/Users/micro/Documents/#GIT/loomux-src/ub python -c "…"`.

---

## Planköpfe (werden wörtlich übernommen)

# loomux Stufe 1b-1 — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux brain search|catalog|read|neighbors|status` mit belegter Parität zur Python-Referenz von ultra-brain, dazu `loomux dev mutants` und die Mutationsrunde der Stufe.

**Architecture:** ultra-brains Go-Pakete `catalog`, `reader`, `graph`, `privacy`, `index/identity.go` und beide qmd-Ports ziehen mit Tests nach `internal/brain/*` und werden dort test-first auf das Verhalten der Python-Referenz gebracht. Neu entstehen `internal/brain/pytext` (Pythons Text-, Pfad- und Zeitschreibweisen, von fünf Paketen gebraucht), `internal/brain/status`, der Befehl `loomux brain`, ein Fake-qmd für deterministische Aufzeichnungen und `loomux dev mutants` über `go test -overlay`. `brain/*` liest befristet ultra-brains Zustandsverzeichnis und die alten Manifestnamen.

**Tech Stack:** Go ≥ 1.25 (`go 1.25.0` in `go.mod`, gebaut mit Go 1.27), `github.com/BurntSushi/toml v1.6.0`, `golang.org/x/sys v0.18.0`, `gopkg.in/yaml.v3 v3.0.1`, neu `golang.org/x/text v0.38.0` (NFC und casefold; offline im Modulcache, lässt `go 1.25.0` stehen — gemessen 2026-09-15).

**Spec:** `docs/.superpowers/specs/2026-09-15-loomux-stufe-1b-1-design.md`

## Global Constraints

- **Wo gearbeitet wird:** Task 0 legt es fest. Im Hauptcheckout `C:\Users\micro\Documents\#GIT\loomux` nur mit ausdrücklichem Ja des Nutzers auf `master`; sonst Worktree `C:\Users\micro\Documents\#GIT\loomux-sdd-1b1` auf Branch `sdd-1b-1`, dessen Pfad der Mensch vorher in die loomux-Registry einträgt (sonst verweigert die Schreibschranke des Piloten jeden Write dort). Edit/Write-Werkzeuge verweigern Pfade unter `%TEMP%` außerhalb des Sitzungs-Scratchpads; temporäre Dateien schreibt Go-Code (`t.TempDir()`, `os.WriteFile`) oder die Shell.
- **Quellen nur lesen:** `C:\Users\micro\Documents\#GIT\loomux-src\ub` (ultra-brain, Tag `loomux-1a-source`, `3cc72d2`). Nie aus einem laufenden Checkout kopieren.
- **Modulpfad** `github.com/xidus90/loomux`; Pakete unter `internal/`. Einstiege: `cmd/loomux` und die Entwicklerhilfe `internal/dev/fakeqmd/qmd` (nur für Aufzeichnungen gebaut, nie ausgeliefert).
- **Sprache:** Code, Bezeichner, Kommentare, Fehlermeldungen, Commit-Nachrichten englisch; Pläne, Specs und `docs/.superpowers/parity/` deutsch; `docs/en/*.md` und `docs/de/*.md` sagen dasselbe.
- **Coverage 100 % je Funktion**, Tor `loomux dev covergate` im Pre-Commit; eine Ausnahme nur mit `//coverage:exempt <reason>` in der Zeile direkt über `func`, der Grund nennt, was ohne Eingriff ins Betriebssystem unerreichbar ist.
- **Umzugsregel:** Code und Tests wortgleich bis auf Paketnamen, Importpfade, die im Task genannten Literale und Verhaltenskorrekturen und neue Tests für 100 %. Jede weitere Änderung ist ein Befund im Task-Bericht. „Nach Python" heißt: die Abweichung wird test-first auf das Referenzverhalten gebracht, die erwarteten Werte sind an der Referenz gemessen (`uv run --project <ub> python -c …`), nicht aus dem Gedächtnis geschrieben.
- **Referenz im Zweifel:** Wo die Python-Referenz (`src/brain/core.py`, `cli.py`, `identity.py`, `privacy.py`, `walk.py`, `manifest.py`, `registry.py`, `search/qmd.py`, `maintenance/reconcile.py`) Verhalten oder Wortlaut hat, gilt es, auch gegen ultra-brains Go-Fassung. Jede Abweichung, die loomux behält, ist eine Zeile in `docs/.superpowers/parity/stufe-1b-1.md`; der Task, der sie schafft, nennt sie in seinem Bericht, Task 12 trägt sie ein.
- **Exit-Codes `loomux brain`:** 0 Erfolg (auch `no matches`), 1 Laufzeitfehler mit `error: {grund}` auf stderr und ohne stdout, 2 Usage-Fehler.
- **Nie beschreiben:** `.loomux/config.toml` (jede), `%LOCALAPPDATA%\loomux\registry.toml`, `%LOCALAPPDATA%\brain\**`, der qmd-Index. Tests bauen ihre Welten in `t.TempDir()`.
- **Kein Test startet qmd, eine GPU, `uv` oder die Python-Referenz.** Im Tor laufen nur Stubs: `httptest` für den MCP-Port, der `Runner`-Seam für den CLI-Port, `search.FakePort`, `fakeqmd` im Prozess.
- **Startzeit:** kein `init()`. Umgezogene Paketvariablen mit `regexp.MustCompile` bleiben wortgleich (Task 15 misst sie mit `inittrace`); **neuer** Code kompiliert Regexe über `sync.OnceValue` beim ersten Gebrauch, wie 1a Task 10 es entschieden hat.
- **Abhängigkeiten:** `brain/*` benutzt `config` und einander; `cli` ruft `brain/*`; `hooks` importiert kein neues `brain/*`-Paket.
- **Git:** kein Push, kein `gh`. Commits unter der Nutzeridentität ohne `Co-Authored-By` und ohne Werbezeile, Nachricht englisch über `git commit -F <datei>`. Vor jedem Commit `git branch --show-current`, `git rev-parse --short HEAD` und `git diff --cached --stat` lesen. Nur eigene Dateien stagen (`git add <pfade>`, nie `git add .`). Das Tor verweigert ungestagte oder ungetrackte `*.go`, `go.mod`, `go.sum`, `testdata`, `.githooks`. Nach jedem Commit `git log -1 --format='%an <%ae>'`.
- **Ein Shell-Befehl je Aufruf**, keine langen `&&`-Ketten.
- **Subagenten** mit `model: "opus"` und ausdrücklich gesetztem `effort`, ein Implementierer je Task.

## Umzugsverfahren (gilt für die Tasks 3–8)

(Kopie des 1a-Verfahrens, `docs/.superpowers/plans/2026-09-14-loomux-stufe-1a.md:844-876`, mit
`SRC=/c/Users/micro/Documents/#GIT/loomux-src/ub` und `LOOMUX` = Arbeitsort aus Task 0. Schritt 1
kopiert **nur die im Task genannten Dateien**, nicht das ganze Verzeichnis. Schritt 4 ohne
`go mod tidy`-Versionssenkung; `golang.org/x/text` holt Task 2.)

---

## Dateistruktur nach Stufe 1b-1

| Pfad | Verantwortung | Herkunft | Task |
|---|---|---|---|
| `internal/config/legacy.go` | befristet: ultra-brains Zustandsverzeichnis (bis Stufe 3), Manifest-Altnamen (bis Stufe 4) | neu | 1 |
| `internal/brain/pytext/` | Pythons `repr`, `splitlines`, `strip`, `read_text`, `isoformat`/`fromisoformat`, `str(Path)`, NFC, casefold | neu | 2 |
| `internal/brain/privacy/` | Kanal, Sichtbarkeit aller Bereiche, Enthaltensein, `never`-Globs, Prüfzentrum | ub `pkg/privacy` | 3 |
| `internal/brain/identity/` | `_identities.tsv` lesen | ub `pkg/index/identity.go` | 4 |
| `internal/brain/graph/` | `graph.json`, Nachbarn | ub `pkg/graph` `model.go`, `neighbors.go`, `read.go` | 5 |
| `internal/brain/reader/` | Dokument und Abschnitt | ub `pkg/reader` | 6 |
| `internal/brain/catalog/` | Wurzel- und Bereichskatalog | ub `pkg/catalog` `area.go`, `root.go` | 7 |
| `internal/brain/search/` | MCP- und CLI-Port, Suche, Befunde, Reconcile-Stempel | ub `pkg/search` + neu `stamp.go` | 8 |
| `internal/brain/status/` | `status`-Zeilen | neu | 9 |
| `internal/cli/brain.go`, `internal/cli/brainargs.go` | `loomux brain …` | neu | 10 |
| `internal/dev/fakeqmd/`, `internal/dev/fakeqmd/qmd/` | Fake-qmd aus einer Fixture: CLI-Antworten und MCP-Handler | neu | 11 |
| `internal/dev/recordcase/`, `internal/dev/importcases/`, `internal/cli/dev.go` | Programmform, Umgebung, CRLF; Faltung in jedem Bereichsverzeichnis | geändert | 11 |
| `testdata/cases/1b-1-worlds/`, `…/1b-1-source/`, `…/1b-1/`, `…/1b-1-map.toml`, `internal/cli/cases_1b1_test.go` | Fallkorpus | neu | 12 |
| `docs/.superpowers/parity/stufe-1b-1.md` | Abweichungsliste | neu | 12 |
| `internal/dev/mutants/`, `internal/cli/dev.go` | `loomux dev mutants` | neu | 13 |
| `testdata/bench/1b-1-brain.json`, `docs/{en,de}/benchmarks.md` | Messung | neu/geändert | 15 |

**Nicht umgezogen:** ub `pkg/catalog/catalog.go`, `pkg/graph/render.go`, `pkg/graph/synth.go`, `pkg/index` außer `identity.go`, `pkg/mcp`, `pkg/wiki`.

---

## Task 0: Arbeitsort (Mensch)

Keine Schnittstellen. Schritte: Arbeitsort wählen (Worktree empfohlen); für den Worktree
`git -C <loomux> worktree add -b sdd-1b-1 <pfad>` und den Registry-Block, den der Mensch an
`%LOCALAPPDATA%\loomux\registry.toml` anhängt:
```toml
[[area]]
scope     = "project/loomux-sdd-1b1"
path      = "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1"
workspace = true
```
Prüfen: `bin/loomux.exe` im Worktree bauen (`go build -o bin/loomux.exe ./cmd/loomux`), `git config core.hooksPath .githooks`, `sh .githooks/pre-commit` grün vor Task 1.

## Task 1: Befristete Ausnahme in `internal/config`

**Files:** Create `internal/config/legacy.go`, `internal/config/legacy_test.go`. Modify `internal/config/manifest.go` nur so weit, dass `ReadManifest` und die neue Funktion einen gemeinsamen internen Leser über eine Namensliste teilen (kein Verhaltenswechsel von `ReadManifest`, belegt durch dessen unveränderte Tests).

**Produces:**
```go
// LegacyBrainDirUntilStage3 is ultra-brain's state directory. brain/* reads the artefacts of
// read-only areas and the reconcile stamp from it until stage 3 moves reconcile to Go.
func LegacyBrainDirUntilStage3() string // LOOMUX_LEGACY_BRAIN_DIR; else %LOCALAPPDATA%\brain (home\AppData\Local\brain); POSIX $XDG_STATE_HOME/brain, else home/.local/state/brain
// ReadAreaManifestUntilStage4 reads an area's manifest under the name loomux writes, or under
// one of ultra-brain's names until stage 4 moves the hosts: .loomux/config.toml, else
// .ultra-brain/config.toml, else .brain.toml. Only brain/* calls it.
func ReadAreaManifestUntilStage4(dir string) (*Manifest, error)
```
**Consumes:** `Manifest`, `ErrNoManifest`, `manifestNames`, `defaultStateDir` (registry.go; die Legacy-Wahl darf `filepath.Join(filepath.Dir(defaultStateDir(...)), "brain")` rechnen oder `defaultStateDir` um einen Namensparameter erweitern — dann bleibt `StateDir` bitgleich, belegt durch dessen Tests).

**Regeln:**
- Vorrang wie Pythons `registry.manifest_path` erweitert um den loomux-Namen: der erste Name, der als **Datei** existiert. Keiner da → `fmt.Errorf("%s: %w (%s)", dir, ErrNoManifest, <die drei Namen mit ", ">)` in der Schreibweise, die `ReadManifest` für seine Namensliste nutzt.
- Wie Pythons `read_manifest`: ein Manifest ohne nicht-leeres `[area] scope` → `fmt.Errorf("%s: [area] scope is required and must be a non-empty string", path)`. Die übrigen Prüfungen, die Python hat und loomux nicht (`[wiki] types`, `[maintenance]`, `[model]`, `[layout] wiki`-Formregeln), sind eine Paritätszeile, kein Code.
- Kommentar an beiden Funktionen nennt Ablaufstufe und Grund.
- Tests: jede Plattformwahl per Parameter (nicht per `runtime.GOOS`), `LOOMUX_LEGACY_BRAIN_DIR` per `t.Setenv`, Vorrang der drei Namen, Verzeichnis statt Datei unter einem Namen, Altmanifest mit `[privacy] mode = "local_only"` und `never`, fehlendes `[area] scope`, kaputtes TOML nennt die Datei, `ReadManifest` findet eine `.brain.toml` weiterhin nicht.

## Task 2: `internal/brain/pytext` — Pythons Schreibweisen

**Files:** Create `internal/brain/pytext/{repr.go,lines.go,text.go,isotime.go,path.go,unicode.go}` mit Tests. Modify `go.mod`, `go.sum` (`go get golang.org/x/text@v0.38.0`, danach `go.mod` prüfen: `go 1.25.0` unverändert, kein `toolchain`-Eintrag).

**Produces:**
```go
package pytext
func Repr(s string) string                       // repr(str): Quote-Wahl, \\ \' \t \n \r, nicht Druckbares als \xhh \uhhhh \Uhhhhhhhh
func SplitLines(s string) []string               // str.splitlines(): \n \r \r\n \v \f \x1c \x1d \x1e \x85 \u2028 \u2029; "" → leer; kein leeres Endstück
func IsSpace(r rune) bool                        // str.isspace()
func Strip(s string) string                      // str.strip()
func RStrip(s string) string                     // str.rstrip()
func ReadText(path string) (string, error)       // Path.read_text(encoding="utf-8"): streng UTF-8, universelle Zeilenenden (\r\n und einzelnes \r → \n), BOM bleibt
func IsoFormat(t time.Time) string               // datetime.isoformat() eines zonenbehafteten Werts: YYYY-MM-DDTHH:MM:SS[.ffffff]±HH:MM[:SS]
func ParseAwareIsoFormat(s string) (time.Time, bool) // datetime.fromisoformat mit tzinfo; false für naiv, unlesbar oder eine Form außerhalb der gemessenen Teilmenge
func PathString(goos, p string) string           // str(Path(p)): windows wie PureWindowsPath (/ → \, // und . zusammengefasst, Endtrenner fort, .. bleibt), sonst wie PurePosixPath
func NFC(s string) string                        // unicodedata.normalize("NFC", s)
func CaseFold(s string) string                   // str.casefold()
```
Fehlerwortlaut `ReadText` für ungültiges UTF-8: `fmt.Errorf("%s: not valid UTF-8", path)` (Python bricht dort mit Traceback ab — Paritätszeile). Andere Lesefehler unverändert (`os.ReadFile`).

**Regeln:** Jede Funktion bekommt einen Tabellentest, dessen erwartete Werte **an Python 3.14 der Referenz gemessen** sind; der Task legt die Messbefehle und ihre Ausgabe in den Bericht. Mindestens: `repr` von `x`, `it's`, `say "hi"`, `both ' and "`, `a\tb`, `\x00`, `\x7f`, `é`, `\u200b`, `\U0001f600`; `splitlines` aller Trenner und von `"a\n"`, `"\n"`, `""`; `isspace` über den ganzen Unicode-Bereich (Python listet, Go vergleicht); `isoformat` mit und ohne Mikrosekunden, `Z`-Eingabe → `+00:00`, `+02:00` bleibt, 3-stelliger Bruch → 6 Stellen; `fromisoformat`-Annahme/Ablehnung für die Schreibform des Writers, `Z`, Bruch 1–6 Stellen, naiv, nur Datum, Müll — jede Form, die Python annimmt und Go nicht, ist eine Paritätszeile; `str(Path)` für `C:/a/b`, `C:/a//b/./c/`, `C:/a/../b`, `//server/share/x`; NFC von NFD-`é`; casefold `ß`, `ẞ`, `ﬁ`, `Σ`.

## Task 3: `internal/brain/privacy` — Umzug, Sichtbarkeit, Glob-Matcher nach Python

**Files:** Move ub `pkg/privacy/{channel,containment,readable}.go` + `privacy_test.go` → `internal/brain/privacy/` (Umzugsverfahren; `internal/testlock` gibt es in loomux). Create `internal/brain/privacy/areas.go` + `areas_test.go`, `glob_test.go`.

**Produces:**
```go
type Channel string
const ( ChannelLocal Channel = "local"; ChannelCloud Channel = "cloud" )
func ParseChannel(s string) (Channel, error)                                  // unverändert
func VisibleManifest(dir string, ch Channel) (*config.Manifest, bool, error)  // GEÄNDERT: config.ReadAreaManifestUntilStage4; jeder Fehler (auch kein Manifest) → error
func IsVisible(manifest *config.Manifest, ch Channel) bool                     // unverändert
type VisibleArea struct { Area config.Area; Manifest *config.Manifest }        // NEU
func VisibleAreas(registryDir, legacyDir, scope string, ch Channel) ([]VisibleArea, error) // NEU: core._visible_areas
func Single(areas []VisibleArea, scope string) (VisibleArea, error)             // NEU: core._single
func UnknownScope(scope string, areas []VisibleArea) error                      // NEU: core._unknown_scope
func Contained(scope, relative string) (string, error)                          // GEÄNDERT: core._contained
func MatchesGlobs(patterns []string, relative string) bool                      // GEÄNDERT: privacy.matches_globs
func IsReadable(manifest *config.Manifest, relative string) bool                // unverändert
func ReviewExcludes(manifest *config.Manifest) ([]string, error)                // GEÄNDERT: walk.review_excludes
```
**Consumes:** Task 1 `config.ReadAreaManifestUntilStage4`, `config.ReadRegistry`, `config.ManifestDir`; Task 2 `pytext.Repr`, `pytext.NFC`, `pytext.CaseFold`.

**Regeln:**
- `VisibleAreas` liest die Registry aus `registryDir`, je Bereich das Manifest aus `config.ManifestDir(area, legacyDir)`; der erste Fehler bricht ab (wie Python für **jeden** registrierten Bereich). `scope == "all"` → alle sichtbaren in Registry-Reihenfolge; sonst die mit gleichem Scope, keiner → `UnknownScope`.
- `UnknownScope`: `unknown scope {Repr(scope)}; known scopes are: {sortierte sichtbare Scopes mit ", "}`.
- `Contained` nach `core._contained`: `\` → `/`; verweigert posix-absolut, Windows-absolut, jedes Laufwerk (jedes **eine Zeichen** vor `:`, gezählt in Runen), `..` als Segment; Meldung `{scope}/{relative} leaves the area` mit dem rohen `relative`; Rückgabe wie `PurePosixPath.as_posix()` (`""` und `.` → `.`, `//`, `./` und Endtrenner zusammengefasst).
- `MatchesGlobs` nach `PurePosixPath(nfc(relative).casefold()).full_match(nfc(pattern).casefold())`: beide Seiten normalisiert wie `PurePosixPath`; Muster übersetzt wie `glob.translate(pattern, recursive=True, include_hidden=True, seps="/")` samt `fnmatch._translate` (Klassen `[a-z]`, `[!x]`) — die Quelle ist die der Referenz-Python (`python -c "import glob,fnmatch,inspect; print(inspect.getsource(glob.translate)); print(inspect.getsource(fnmatch._translate))"`), nicht ein Gedächtnisbild. Kein `\`-Umbau (Python macht keinen). Regex über `sync.OnceValue` je Muster ist nicht nötig; ein Cache ist verboten, Kompilieren je Aufruf ist erlaubt (Mengen sind klein). Tabellentest mit an Python gemessenen Booleans: `a/**/b`~`a/b`, `**/x.md`~`x.md`, `a/**`~`a`, `review/**`~`review`, `[a-z]*.md`~`x.md`, `[!a]*.md`~`a.md`, `*.md`~`dir/x.md`, `*`~`.hidden`, `ß.md`~`SS.md`, NFD~NFC, `a//b/./c`~`a/b/c`.
- `ReviewExcludes` nach `walk.review_excludes` (walk.py:96-113): kein `TrimSpace`; Ablehnung nur, wenn `PurePosixPath(review).parts` leer ist, Wortlaut mit `Repr`.
- `privacy_test.go` schreibt `.brain.toml`/`.ultra-brain/config.toml` — die Namen werden jetzt gelesen; ein fehlendes Manifest ist jetzt ein Fehler (Test umstellen), ein gesperrtes `.loomux/config.toml` neben einer offenen `.brain.toml` folgt dem Vorrang aus Task 1 (erster existierender Name, also Fehler).
- Paritätszeilen: fehlendes Manifest meldet loomux mit eigenem Wortlaut statt `[Errno 2] No such file or directory: '…'`.

## Task 4: `internal/brain/identity` — Umzug nach Python

**Files:** Move ub `pkg/index/identity.go` + seinen Test → `internal/brain/identity/` (Paket `identity`). Enthält `identity.go` weitere Exporte (Schreiber, Hash), ziehen sie wortgleich mit und werden auf 100 % gehoben; der Bericht nennt sie.

**Produces:**
```go
package identity
const IdentitiesHeader = "doc_id\tpfad\tcontent_hash\trevision"
type Identity struct { DocID, Relative, ContentHash string; Revision int }
func ReadIdentities(path string) (map[string]Identity, error) // fehlt die Datei → leere Map, kein Fehler
```
**Consumes:** Task 2 `pytext.ReadText`, `pytext.SplitLines`, `pytext.Strip`, `pytext.Repr`.

**Regeln nach `identity.py`:** Existenz über `os.Stat` (fehlt → leer; anderer Stat-Fehler → Fehler); Inhalt über `ReadText`; `SplitLines`, erste Zeile ungeprüft übersprungen; Zeile mit `Strip(line) == ""` übersprungen; genau vier `\t`-Felder, sonst `{path}: line {n}: expected 4 tab-separated fields, found {k}` (n ab 2); Revision nur ASCII-Ziffern, sonst `{path}: line {n}: revision {Repr(rev)} is not a number`; doppelter Pfad → letzte Zeile gewinnt. Paritätszeile: Revision aus Nicht-ASCII-Ziffern (Python `isdigit`+`int`) und ungültiges UTF-8 (Python Traceback).

## Task 5: `internal/brain/graph` — Umzug, Wortlaute und Formprüfung nach Python

**Files:** Move ub `pkg/graph/{model,neighbors,read}.go` + `neighbors_test.go`, `read_test.go` → `internal/brain/graph/`. Nicht: `render.go`, `synth.go` und ihre Tests; ein Test in `read_test.go`/`neighbors_test.go`, der sie braucht, entfällt mit Bericht.

**Produces:**
```go
type Node struct{ ID string `json:"id"`; Title string `json:"title"`; Tags []string `json:"tags"` }
type Edge struct{ From string `json:"from"`; To string `json:"to"` }
type LinksInfo struct{ Total int `json:"total"`; Resolved int `json:"resolved"`; Dropped map[string]int `json:"dropped"` }
type Graph struct{ Scope string `json:"scope"`; Nodes []Node `json:"nodes"`; Edges []Edge `json:"edges"`; Links LinksInfo `json:"links"` }
type Document struct{ Relative string; Title string; Tags []string; Links []string }
var ErrNotIndexed = errors.New("never indexed; run `brain reindex`")   // Literal geändert
func Neighbors(g *Graph, relative string) ([]string, []string)
func RenderNeighbors(incoming, outgoing []string) string
func ReadGraph(area config.Area, stateDir string) (*Graph, error)     // stateDir = Legacy-Verzeichnis
func ParseGraph(data []byte, path string) (*Graph, error)
```
**Consumes:** Task 2 `pytext.ReadText`; `config.ManifestDir`.

**Regeln nach `core._graph`/`_check_shape`/`_edges`/`_links`:** alle Meldungen mit Backticks (`delete it and run `brain reindex``); fehlende Datei → `{scope}: never indexed; run `brain reindex`` (Python prüft `exists()`); Lesen über `ReadText`; ein Wurzelwert, der kein Objekt ist, → `graph is missing edges, links`; Zählwerte nur ganzzahlige JSON-Zahlen (`3.0`, `1.5`, `1e2` → `the link counts are not numbers`; mit `json.Decoder.UseNumber`). Paritätszeilen: Wahrheitswerte als Zählwert (Python nimmt sie), Nicht-Zeichenketten in `from`/`to` (Python wandelt per `str()`), Wortlaut des JSON-Fehlers in Klammern, Wurzel-Array oder -Zeichenkette mit Schlüsselnamen.

## Task 6: `internal/brain/reader` — Umzug, Lesen und Abschnitte nach Python

**Files:** Move ub `pkg/reader/{read,section}.go` + `reader_test.go` → `internal/brain/reader/`.

**Produces:**
```go
func ReadDocument(area config.Area, manifest *config.Manifest, relative, section string, ch privacy.Channel, stateDir string) (string, error) // section "" = ganze Datei
func ExtractSection(content, heading string) (string, error)
```
**Consumes:** Task 2 `pytext.ReadText`, `pytext.SplitLines`, `pytext.Strip`, `pytext.RStrip`, `pytext.Repr`; Task 3 `privacy.Contained`, `privacy.IsReadable`, `privacy.MatchesGlobs`, `privacy.ReviewExcludes`.

**Regeln nach `core.read`/`_section`:** Reihenfolge Enthaltensein → `never` (`{scope}/{inside} is excluded by [privacy] never`) → Prüfzentrum nur auf `cloud` (`{scope}/{inside} is the review centre; refused on the cloud channel`) → `ReadText(area.Path/inside)` → Abschnitt. `_section` wörtlich: `SplitLines`; Überschrift = Zeile beginnt mit `#` und `Strip(TrimLeft(line, "#")) == heading`; Ebene = Zahl führender `#`; Ende = nächste Zeile mit `#` und Ebene ≤; Ergebnis `RStrip(join("\n")) + "\n"`; fehlt sie → `no section titled {Repr(heading)}`. Paritätszeilen: `--section ""` (Python sucht eine leere Überschrift), `.`/leerer Pfad (Python liest das Verzeichnis und scheitert mit `Errno 13`).

## Task 7: `internal/brain/catalog` — Umzug

**Files:** Move ub `pkg/catalog/{area,root}.go` + `area_test.go`, `root_test.go` → `internal/brain/catalog/`. Nicht: `catalog.go`, `catalog_test.go`.

**Produces:**
```go
func AreaArtifactDir(area config.Area, stateDir string) string
func ReadAreaCatalog(area config.Area, stateDir string) (string, error) // GEÄNDERT: pytext.ReadText
func RenderRootCatalog(areas []config.Area) string                      // "# brain\n\n" + je Bereich nach Scope sortiert "* [s](brain://s/)\n"
```
**Consumes:** Task 2 `pytext.ReadText`. **Paritätszeile:** fehlendes `index.md` meldet loomux mit dem Go-Wortlaut.

## Task 8: `internal/brain/search` — Ports, Suche, Befunde, Stempel

**Files:** Move ub `pkg/search/{port,http,mcp,daemon,daemon_windows,daemon_other,launcher,qmd,search,fake}.go` + alle Tests (`port_test`, `http_test`, `mcp_test`, `daemon_test`, `launcher_test`, `qmd_test`, `search_test`, `visibility_test`) → `internal/brain/search/`. Create `stamp.go` + `stamp_test.go`.

**Produces (ub-API, Änderungen benannt):**
```go
// port.go, http.go, fake.go: wie ub (Profile, ProfileKeyword/Fast/Full, SearchHit, SearchPort, Session, HTTPSession, NewHTTPSession, DefaultHost, DefaultPort, …, FakePort, NewFakePort, ScriptedSearch, ScriptedIndexed, ScriptedPending, SearchCall)
// mcp.go
const WarmingNotice = "starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm."
func WithNotice(fn func(message string)) QmdMcpOption                           // NEU
func DefaultConnectWith(port int, launcher func(string) ([]string, error), spawner DaemonSpawner, waitTimeout time.Duration, notice func(string)) ConnectFunc // GEÄNDERT: notice (nil erlaubt) höchstens einmal je Port, direkt nach einem erfolgreichen Start des Daemons, vor dem Warten
func DefaultConnect(port int, notice func(string)) ConnectFunc                   // GEÄNDERT
// daemon.go
func StartDaemonWith(env map[string]string, port int, launcher func(string) ([]string, error), spawner DaemonSpawner) error // GEÄNDERT: ist QMD_LLAMA_GPU oder QMD_FORCE_CPU in der Umgebung gesetzt (os.LookupEnv), wird keine Backbone-Variable angehängt
// launcher.go: Wortlaut cannot find {Repr(name)} on PATH; node-Suche wie Python (node.exe neben dem Shim, sonst LookPath("node"), eine .cmd/.bat wird verweigert)
// qmd.go
func DefaultRunner(argv []string) ([]byte, []byte, int, error) // GEÄNDERT: startet Launcher(argv[0]) + argv[1:], Umgebung os.Environ() (CUDA-Vorgabe hängt nichts an)
//   (*QmdPort).invoke: ein Launcher-Fehler geht unverändert weiter (Python: SearchUnavailable aus launcher), ein Startfehler als "cannot run {argv0}: {err}", ein Exit ≠ 0 als "{argv0} exited with {code}: {pytext.Strip(stderr)}"
// search.go
type SearchAnswer struct{ Hits []SearchHit; Findings []string }
const NoMatches = "no matches"
func CollectionName(scope string) string
func ExecuteSearch(query, scope string, profile Profile, n int, channel privacy.Channel, port SearchPort, registryDir, legacyDir string, now time.Time) (*SearchAnswer, error) // GEÄNDERT
func FormatSearch(answer *SearchAnswer) string                                   // GEÄNDERT: Snippet über pytext.SplitLines
// stamp.go (neu)
const ReconcileInterval = 24 * time.Hour
const ReconcileAdvice = "run `brain reconcile`"
func ReadLastRun(stateDir string) (time.Time, bool, error) // <stateDir>/maintenance/last-run.txt; fehlt, kein UTF-8, unlesbar als ISO oder naiv → false, nil; andere Lesefehler → error
func Stale(stamp, now time.Time) bool                      // now.Sub(stamp) >= ReconcileInterval
func StaleReconcile(stateDir string, now time.Time) ([]string, error) // der Suchbefund oder nichts
```
**Consumes:** Task 2 (`Repr`, `SplitLines`, `Strip`, `ReadText`, `IsoFormat`, `ParseAwareIsoFormat`); Task 3 (`VisibleAreas`, `IsReadable`, `Channel`); Task 4 (`identity.ReadIdentities`); `config.ManifestDir`.

**Regeln:**
- `ExecuteSearch` nach `core.search`/`_ask`/`_assemble`: `VisibleAreas(registryDir, legacyDir, scope, channel)`; keine → leere Antwort ohne Befunde; Collections sortiert; zweiter Versuch bei leerer Antwort, dann Befund (Wortlaut core.py:125-128 mit `profile`); Register **aller** sichtbaren Bereiche vor dem ersten Treffer lesen (`<ManifestDir(area, legacyDir)>/_identities.tsv`, Fehler → Abbruch); je Treffer unbekannte Collection oder `never` → gezählt; nicht im Register → Befund `{scope}/{relative}: hit is not in the register; reindex to catch up`; kein Abbruch bei `n`, am Ende `ordered[:n]`; Zurückgehaltene-Befund (core.py:190-194); Stempelbefund aus `StaleReconcile(legacyDir, now)` zuletzt. `n <= 0 → 5` entfällt (die CLI verweigert).
- `FormatSearch` nach `_print_search`: ohne Treffer `no matches\n`; je Treffer `brain://{scope}/{relative}:{line}  {score:.0%}  {title}\n`, jede `SplitLines`-Zeile als `    {line}\n`, dann `\n`. Kein Rückfall auf `Collection` bei leerem Scope.
- `StaleReconcile`: nur zonenbehafteter Stempel und `Stale` → ``the last full reconciliation was {IsoFormat}, more than 24 hours ago: a source may have changed without this answer knowing (run `brain reconcile`)``.
- `WithNotice` test-first über Spawner-Stub: Notiz genau einmal, nur wenn gestartet wurde, nicht bei erreichbarem Daemon.
- Paritätszeilen: 1 (Rangfolge, `fast` ohne Erweiterung), 2 (`keyword`-Score und -Zeile), 5 (Aufwärm-Hinweis), 6 (Backbone), Wortlaut `within 1m0s` gegen Pythons `within 60s`.

## Task 9: `internal/brain/status` — neu nach `core.status`

**Files:** Create `internal/brain/status/{status.go,status_test.go}`.

**Produces:**
```go
package status
func Lines(ch privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) ([]string, error)
```
**Consumes:** Task 2 (`IsoFormat`, `PathString`, `NFC`); Task 3 (`VisibleAreas`, `IsReadable`, `MatchesGlobs`); Task 4 (`ReadIdentities`); Task 5 (`ReadGraph`); Task 8 (`ReadLastRun`, `Stale`, `ReconcileAdvice`, `CollectionName`, `SearchPort`, `FakePort` in Tests); `config.ManifestDir`.

**Regeln:** jede Zeilenart L1–L8 mit Wortlaut, Bedingung, Reihenfolge und Sortierung aus `status.md` §2–§6 und core.py:328-532: L1 immer; je sichtbarem Bereich in Registry-Reihenfolge L2 (≥ 2 Include-Globs), L3 (`os.Stat(area.Path)` scheitert → `{scope}: {PathString(runtime.GOOS, area.Path)} does not exist; skipped`, weiter), L4 (`graph.json` fehlt im Artefaktverzeichnis → weiter), L5 (`total != 0 && resolved*2 < total`, `dropped` nach Schlüssel sortiert, leer → `()`), L6 (unsere = sortierte Register-Pfade ohne `never` und `unsearched`; ihre = `NFC` von `port.Indexed(CollectionName(scope))`, gerufen auch bei leerem „unsere"; erste drei, `, …` mit U+2026 bei mehr als drei; jeder Fehler von `Indexed` → L6b mit `{err}`); dann L7 über **alle** sichtbaren Bereiche (auch übersprungene), nach Hash-Zeichenkette sortiert, Pfade darin sortiert; dann L8 einmal (`NotYetSearchable`, 0 → keine Zeile, Fehler → L8b). Fehler aus Registry, Manifest, Register, Graph brechen ab. Je Zeilenart ein Test mit `FakePort` und Welt in `t.TempDir()`; die L6b-Zeile mit eingebettetem Zeilenumbruch aus dem gemessenen qmd-Text wird wörtlich geprüft.

## Task 10: `loomux brain` in `internal/cli`

**Files:** Create `internal/cli/brain.go`, `internal/cli/brainargs.go`, `internal/cli/brain_test.go`, `internal/cli/brainargs_test.go`. Modify `internal/cli/commands.go` (`"brain": brainCommand,`).

**Produces:**
```go
func brainCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int
var brainSearchPort = func(notice func(string)) search.SearchPort { return search.NewQmdMcpPort(search.WithNotice(notice)) }
var brainStatusPort = func() search.SearchPort { return &search.QmdPort{Executable: "qmd", Runner: search.DefaultRunner} }
var brainNow = time.Now
// brainargs.go: ein argparse-ähnlicher Leser, Form nach Wahl des Tasks, mit Tests für jede Fehlerart
```
**Consumes:** Task 1 (`LegacyBrainDirUntilStage3`), `config.StateDir`; Task 3 (`ParseChannel`, `VisibleAreas`, `Single`, `Contained`); Task 5 (`ReadGraph`, `Neighbors`, `RenderNeighbors`); Task 6 (`ReadDocument`); Task 7 (`ReadAreaCatalog`, `RenderRootCatalog`); Task 8 (`ExecuteSearch`, `FormatSearch`, `Profile*`); Task 9 (`status.Lines`).

**Regeln:**
- Unterbefehle und Formen wie `cli.py:471-542`: `search <query> [--scope all] [--profile keyword|fast|full, Vorgabe fast] [-n N, Vorgabe 5]`, `catalog [--scope all]`, `read <relative> --scope S [--section T]`, `neighbors <relative> --scope S`, `status`; alle mit `--channel local|cloud` (Vorgabe `local`). Flags und Positionsargument in beliebiger Reihenfolge; `--flag value` und `--flag=value`; `-n 3` und `-n3`; `--` beendet Optionen. Nicht unterstützt (Paritätszeile): Abkürzungen wie `--prof`, `--state-dir`.
- Usage-Fehler → stderr `usage: loomux brain <sub> …` und `loomux brain <sub>: error: {argparse-Wortlaut}`, Exit 2: `argument --channel: invalid choice: 'x' (choose from local, cloud)`, `argument --profile: invalid choice: …`, `argument -n: must be at least 1, got N`, `argument -n: invalid _at_least_one value: 'x'`, `the following arguments are required: --scope` bzw. `query`/`relative`, `unrecognized arguments: …`. Ohne Unterbefehl `loomux brain: subcommand required`, unbekannt `loomux brain: unknown subcommand "x"`, beide Exit 2 (Form von `loomux dev`).
- Laufzeitfehler → `error: {err}` auf stderr, Exit 1, kein stdout (Ausgabe erst schreiben, wenn die Antwort steht).
- `catalog` wie `core.catalog` + `print(end="")`; `read` wie `core.read` + `print(end="")` (Scope über `VisibleAreas("all")` + `Single`); `neighbors` wie `core.neighbors` (`Single` → `Contained` → `ReadGraph(area, legacyDir)`) + `RenderNeighbors`; `search` → `FormatSearch` auf stdout, danach je Befund `note: {befund}` auf stderr; der Aufwärm-Hinweis geht über `brainSearchPort(func(m string){ fmt.Fprintf(stderr, "note: %s\n", m) })`; `status` → `status.Lines(ch, brainStatusPort(), …)` je Zeile mit `\n`.
- Registry aus `config.StateDir()`, Legacy aus `config.LegacyBrainDirUntilStage3()`, Zeit aus `brainNow()`. Ein Port mit `Close() error` wird nach der Suche geschlossen.
- Tests: jede Form, jeder Usage-Fehler mit exaktem stderr, jeder Laufzeitfehlerweg, `search` mit gesetztem `brainSearchPort` (FakePort) inklusive Befunden und Hinweis, `status` mit `brainStatusPort` (FakePort).

## Task 11: Werkzeuge für den 1b-1-Korpus — Fake-qmd, Programmform, Faltung

**Files:** Create `internal/dev/fakeqmd/{fakeqmd.go,fakeqmd_test.go}`, `internal/dev/fakeqmd/qmd/main.go`. Modify `internal/dev/recordcase/recordcase.go` (+Test), `internal/dev/importcases/importcases.go` (+Test), `internal/cli/dev.go` (+`dev_test.go`).

**Produces:**
```go
package fakeqmd
const FixtureName = "qmd-fixture.json" // liegt in der Wurzel einer Welt
const FixtureEnv = "LOOMUX_FAKE_QMD_FIXTURE"
type Hit struct {
	Collection string  `json:"collection"`
	Relative   string  `json:"relative"`
	Line       int     `json:"line"`
	Score      float64 `json:"score"`
	DocID      string  `json:"docid"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet"`
}
type Fixture struct {
	Collections map[string][]string `json:"collections"`  // qmd ls <c>
	Pending     int                 `json:"pending"`      // qmd status
	StatusError string              `json:"status_error"` // nicht leer: qmd status endet mit Exit 1 und diesem stderr
	Hits        []Hit               `json:"hits"`         // search|vsearch|query --json und MCP query
}
func Load(path string) (*Fixture, error)                                // fehlt die Datei → leere Fixture
func (f *Fixture) RunCLI(args []string, stdout, stderr io.Writer) int
func (f *Fixture) MCPHandler() http.Handler
func Main(args []string, getenv func(string) string, stdout, stderr io.Writer) int // Fixture aus FixtureEnv, dann RunCLI; ohne Variable Exit 2
// internal/dev/fakeqmd/qmd/main.go: package main; func main() { os.Exit(fakeqmd.Main(os.Args[1:], os.Getenv, os.Stdout, os.Stderr)) } mit //coverage:exempt

// recordcase.Spec — neue Felder
//   Argv        []string // Programm und führende Argumente; ersetzt das erste Token von Cmd; schließt Exe aus
//   Env         []string // KEY=VALUE, {{WORLD}} wird durch die gestagte Welt ersetzt
//   PathPrepend string   // Verzeichnis vor PATH des aufgezeichneten Prozesses
// immer: BRAIN_STATE_DIR=<welt>, PYTHONUTF8=1; stdout "\r\n" → "\n" vor Normalize
// importcases.TranslateWorld(dir string) error — GEÄNDERT: faltet zusätzlich in jedem Verzeichnis, das ein `path` in dir/registry.toml als "{{WORLD}}/…" nennt
```
**Consumes:** `cases.SplitCommand`, `cases.WorldToken`, `cases.Normalize`; `search`-Formen aus Task 8 (MCP-Antwortform `translateReply`, CLI-Parser `qmd.py:_parse`).

**Regeln:**
- CLI der Fixture: `ls <c>` → je Pfad eine Zeile ` 1 B  Jan  1 00:00  qmd://<c>/<rel>`; unbekannte Collection → Exit 1, stdout leer, stderr `Collection not found: <c>\nRun 'qmd ls' to see available collections.\n` (gemessen). `status` → `Documents\n  Pending:  N need embedding (run 'qmd embed')\n` bei N > 0, sonst ohne die Pending-Zeile; `StatusError` → Exit 1. `search|vsearch|query <q> --json -n N [-c c]…` → JSON-Array `{"file":"qmd://c/rel","line":…,"score":…,"docid":…,"title":…,"snippet":…}` der Treffer in den genannten Collections, Fixture-Reihenfolge, höchstens N. Unbekannter Befehl → Exit 2.
- MCP-Handler: POST JSON-RPC; `initialize` → `result` mit `protocolVersion "2025-06-18"`; `tools/call` `query` → `result.structuredContent.results` der Treffer aus `arguments.collections`, höchstens `arguments.limit`, `file` = `c/rel`; alles andere → JSON-RPC-Fehler. Antwort als einfaches JSON (der Port liest auch SSE, braucht es nicht).
- `dev record-case`: neue Flags `--argv "<programm und argumente>"` (über `SplitCommand`), `--env KEY=VALUE` (wiederholbar), `--path-prepend DIR`; genau eins von `--exe`/`--argv`; Pflichtmeldung `loomux dev record-case: --exe or --argv, --cmd, --world and --out are required`, beide → `loomux dev record-case: --exe and --argv exclude each other`.
- Die 1a-Aufzeichnungen bleiben gültig: `TestRecordedCasesOfStage1a` grün, `1a-source` unberührt.

## Task 12: Der Fallkorpus der Stufe 1b-1

**Files:** Create `testdata/cases/1b-1-worlds/**`, `testdata/cases/1b-1-source/**`, `testdata/cases/1b-1/**`, `testdata/cases/1b-1-map.toml`, `internal/cli/cases_1b1_test.go`, `docs/.superpowers/parity/stufe-1b-1.md`. Modify `testdata/cases/README.md`.

**Produces:** `func TestRecordedCasesOfStage1b1(t *testing.T)` mit festgenagelter Fallzahl.
**Consumes:** Task 10 (`brainSearchPort`, `brainStatusPort`), Task 11 (`fakeqmd.Load`, `FixtureName`, `RunCLI`, `MCPHandler`, record-case-Flags, `TranslateWorld`), Task 8 (`NewQmdMcpPort`, `WithConnect`, `WithCLI`, `NewHTTPSession`, `QmdPort`).

**Regeln:**
- Fake-qmd bauen: `go build -o "$TEMP/loomux-fakeqmd/qmd.exe" ./internal/dev/fakeqmd/qmd`.
- Aufzeichnen je Fall: `bin/loomux.exe dev record-case --argv "uv run --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --path-prepend "$TEMP/loomux-fakeqmd" --cmd "brain-mcp <befehl> …" --world testdata/cases/1b-1-worlds/<welt> --out testdata/cases/1b-1-source/<verb>/<fall> --notes "…" [--compare message]`.
- Welten: Registry `registry.toml` mit Pfaden `{{WORLD}}/…`; schreibbare Bereiche als `{{WORLD}}/repo-*` mit `.ultra-brain/config.toml` oder `.brain.toml`, `_identities.tsv`, `graph.json`, `index.md` und Dokumenten; schreibgeschützte unter `areas/<flat>/`; Stempel `maintenance/last-run.txt` weit in der Vergangenheit (`2000-01-01T00:00:00+00:00`) oder Zukunft (`2999-01-01T00:00:00+00:00`); `qmd-fixture.json` in der Wurzel.
- Fälle: je Befehl Erfolg, jede Verweigerung und jeder Fehlerweg aus der Spec („Fehlerverhalten"); `status` je Zeilenart L1a/b/c … L8a/b; `search --profile full` als Daten (Treffer mit mehrzeiligem Snippet, ein `never`-Treffer, einer nicht im Register, veralteter Stempel; zweimal leer → `no matches`), `fast` und `keyword` als Meldung; `cloud`-Kanal mit `local_only`-Bereich.
- Map: `[[command]] from = "brain-mcp " to = "loomux brain "`.
- Suite: je Fall `t.Chdir(dir)`, `t.Setenv("LOOMUX_STATE_DIR", dir)`, `t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)`; Fixture aus `dir/qmd-fixture.json`; `httptest.NewServer(fixture.MCPHandler())`; `brainSearchPort` → MCP-Port gegen den Server mit CLI-Runner aus `fixture.RunCLI`; `brainStatusPort` → `QmdPort` mit demselben Runner; beide Seams nach dem Fall zurückgesetzt.
- Rot lesen wie in 1a Task 14 Step 6; Paritätsliste in der Form von `stufe-1a.md` mit den sechs Zeilen der Spec und allen, die die Tasks 1–11 gemeldet haben; jede ohne Freigabe (Freigaben gehören dem Menschen).

## Task 13: `loomux dev mutants`

**Files:** Create `internal/dev/mutants/{mutants.go,generate.go,round.go}` mit Tests. Modify `internal/cli/dev.go` (+`dev_test.go`).

**Produces:**
```go
package mutants
type Mutant struct { Family string; Path string; Line int; Was, Now string }
func Generate(path string, source []byte) []Mutant // Familien a1–a4 nach ub/tools/go_mutants.py
type Outcome int
const ( Passed Outcome = iota; Failed; BuildFailed; TimedOut )
type TestFunc func(pkg, overlay string) (Outcome, error) // overlay "" = unverändert
func GoTest(root string) TestFunc                        // go test [-overlay …] -count=1 -failfast -timeout 60s ./<pkg>/; //coverage:exempt nur hier
type Options struct { Packages []string; Root string; Only string; Family string; Workers int }
type Summary struct { Killed, Survived, NotCompiled, NoMutant int; Survivors []Mutant }
var ErrBaselineRed = errors.New("the package suite is red before any mutant")
func Round(opts Options, test TestFunc, w io.Writer) (Summary, error)
```
**Regeln:** Mutationsregeln wörtlich aus `tools/go_mutants.py` (Regex, Operatorliste, `_split_top`, `_columns`, Ausschluss von `for`); Bedeutung von `no mutant` wie dort; Bericht je Mutant `killed`/`SURVIVED`/`no mutant`/`not compiled` (Wortlaut des Skripts, wo es einen hat), Summen, Liste der Überlebenden; `TimedOut` zählt als getötet; Overlay-JSON `{"Replace": {"<absolut>": "<temp>"}}` in `os.MkdirTemp`, aufgeräumt auch bei Fehler; `Workers` Vorgabe `runtime.NumCPU()/2`, mindestens 1. CLI: `loomux dev mutants <paket>… [--only DATEI] [--family a1|a2|a3|a4] [--workers N]`; `ErrBaselineRed` → Exit 2; Exit 0 nach vollständiger Runde. Tests zählen die Mutanten eines Fixture-Quelltexts exakt (Zahlen an `go_mutants.py` gemessen) und fahren eine Runde mit gestubbtem `TestFunc`.

## Task 14: Die Mutationsrunde der Stufe

**Files:** Modify `docs/.superpowers/parity/stufe-1b-1.md`; Tests in den betroffenen Paketen.

**Regeln:** `loomux dev mutants` über `internal/brain/guard`, `internal/hooks`, `internal/config`, `internal/cases`, `internal/brain/wiki` und alle neuen `internal/brain/*`-Pakete; Protokoll nach `$TEMP`; je Überlebendem ein nachgereichter Test (rot mit dem Mutanten gezeigt, grün ohne) oder eine Zeile, warum der Code den Unterschied nicht sehen kann.

## Task 15: Messen

**Files:** Create `testdata/bench/1b-1-brain.json`. Modify `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `docs/.superpowers/parity/stufe-1b-1.md`.

**Regeln:** Schritt 1 (Mensch): die Bereiche aus `%LOCALAPPDATA%\brain\registry.toml` in `%LOCALAPPDATA%\loomux\registry.toml` übernehmen — der Controller legt die Vorlage vor. Dann `loomux dev bench-hooks testdata/bench/1b-1-brain.json -n 20` über `bin/loomux.exe brain search` je Profil (warm; kalt nach beendetem Daemon) und `brain status`; `GODEBUG=inittrace=1` vor dem Einzug (Binary aus dem Commit vor Task 3) und danach; Zielwert `brain search --profile fast` warm ≤ 150 ms mit eigens ausgewiesenem Anteil fürs Lesen der Register; Live-Vergleich `status` und eine `full`-Suche: Python (`uv run --project <ub> brain-mcp …`, kein brain-Daemon aktiv) und loomux in derselben Minute, nebeneinander in der Paritätsliste. Eintrag in beiden Sprachen im Format des letzten Eintrags.

## Task 16: Rauchtest und Abschluss

**Files:** Modify `docs/.superpowers/parity/stufe-1b-1.md` (Abschnitt „Pilot").

**Regeln:** die fünf Befehle gegen die echte Registry je einmal, Ergebnisse in „Pilot"; Abschlussprüfung gegen die fünf Punkte „Stufe 1b-1 ist fertig, wenn"; Freigaben gehören dem Menschen.
