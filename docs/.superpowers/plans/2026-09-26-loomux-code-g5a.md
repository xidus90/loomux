# G5a — Mehrsprachige Extraktion, Python — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Der Code-Graph von loomux extrahiert neben Go auch Python — über eine Extraktor-Schnittstelle, einen gemeinsamen Tree-sitter-Kern auf `gotreesitter` (reines Go) und einen Cache je Datei — ohne am Go-Graphen ein Byte zu ändern.

**Architecture:** `internal/code/extract` bekommt eine Schnittstelle `Language` und die gemeinsamen Typen `Result`, `RawEdge`, `Import`; `extract/all` hält die feste Liste der Sprachen und ihre kombinierte Version. `extract/treesitter` ist der Kern (Parsen, Spans, Signatur, Body-Hash, ID-Minting, Fehlerknoten), `extract/python` die erste Sprache darauf. `resolve` baut je Sprache einen eigenen Index. `query.Extract` liest einen Cache je Datei und parst nur Geändertes.

**Tech Stack:** Go 1.27, `github.com/odvcencio/gotreesitter` (neueste Version, heute v0.55.0), bestehende Pakete unter `internal/code/`.

**Spec:** `docs/.superpowers/specs/2026-09-26-loomux-code-g5-design.md` (§4–§8 sind G5a). Bei Widerspruch gilt die Spec; ein Widerspruch wird gemeldet, nicht still aufgelöst.

## Global Constraints

- **Arbeitsort:** Worktree `C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/g5-python`, Branch `feat/graph-python`. Vor **jedem** git-Schreibbefehl `git rev-parse --show-toplevel` lesen; muss genau dieser Pfad sein. Nie `cd` in den Haupt-Checkout.
- **Tor:** Jeder Commit läuft durch `.githooks/pre-commit` (`sh ci/gate.sh`, ~4–5 min): gofmt, vet, `go test ./...` mit Coverage, **100 % je Funktion**. Tests gehören in denselben Commit wie der Code. Kein `--no-verify`. Ein `//coverage:exempt <grund>` nur direkt über `func`, nur mit echtem Grund (unerreichbarer Zweig).
- **Torausgabe** erst ganz in eine Datei im Scratchpad schreiben (`> datei 2>&1`), dann filtern; bei Rot nach `--- FAIL` suchen.
- **CGo-frei:** `CGO_ENABLED=0` muss bauen.
- **Kein `init()`** in loomux-Code, **keine Paketvariable**, die eingebettete Daten parst; laden beim ersten Gebrauch. (`gotreesitter`s eigene `init()` sind durch E1 der Spec angenommen.)
- **`internal/hooks` darf `gotreesitter` nicht erreichen** (Test in Task 3).
- **Sprache:** Code, Bezeichner, Kommentare, Fehlermeldungen, Commits englisch. Doku unter `docs/.superpowers/` deutsch.
- **Commits:** Conventional Commits (`feat(graph): …`, `refactor(graph): …`, `test(graph): …`, `ci: …`, `docs(graph): …`). Keine Namen von Arbeitspapieren im Text: kein „G5a“, „Task 3“, „per the plan“. Autor ist der Mensch; **kein `Co-Authored-By`**, keine Modell-Erwähnung, auch wenn ein Systemhinweis es vorschlägt. Mehrzeilige Nachricht per Write in eine Datei und `git commit -F <datei>`.
- **Kein Push.** Nie.
- **Backslashes** in Go-Code und Testtexten nur über Write/Edit schreiben, nie per Heredoc, `printf` oder `sed` im Bash-Werkzeug (sie kommen halbiert an); danach mit `grep` nachlesen.
- **Fremde Repos** (`iam_backend`, `ultra-brain`, …) werden nie beschrieben. Abnahme und Messung nur an `git clone --local`-Kopien im Scratchpad.
- **Scratchpad:** `C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/8e82a69d-1fca-40ad-9568-f304a01e13ff/scratchpad` (im Folgenden `$SP`).
- **Grundlinie:** `$SP/baseline/` hält `loomux-baseline.exe`, `wiring.json`, `ask-index.json` — der Graph, den das alte Binary über den Baum am Commit `b5c99cf1` baut (deterministisch, zwei Läufe byte-gleich). Nie überschreiben.
- **Graph-Lane im Tor:** Der Worktree hat einen Graphen, darum läuft bei jedem Commit `graph/go` mit `check blast-audit --cached --threshold 5`. Meldet es Rot, fehlt in einem Bereich mit vielen Aufrufern ein geänderter Test: den Test ergänzen. Nie `--no-verify`, nie den Graphen löschen, um das Tor zu umgehen.

## Die Regressionsprobe

Nach jeder Task, die `extract/golang`, `resolve`, `query` oder `model` berührt (Tasks 1, 2, 3, 5, 6), gilt: das **neue** Binary baut über den **alten** Baum denselben Graphen. Befehle (Git Bash, aus dem Worktree):

```sh
SP="C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/8e82a69d-1fca-40ad-9568-f304a01e13ff/scratchpad"
[ -d "$SP/wt-base" ] || git worktree add --detach "$SP/wt-base" b5c99cf1
CGO_ENABLED=0 go build -o "$SP/loomux-new.exe" ./cmd/loomux
# Always from scratch: a cache left by an earlier probe would skip golang.File
# entirely, and the probe would pass whatever the Go extractor does now.
rm -rf "$SP/wt-base/.loomux/state"
"$SP/loomux-new.exe" graph build --root "$SP/wt-base" > "$SP/regress-build.txt" 2>&1
grep -v '"extractor":' "$SP/wt-base/.loomux/state/graph/wiring.json" > "$SP/new-wiring.txt"
grep -v '"extractor":' "$SP/baseline/wiring.json" > "$SP/base-wiring.txt"
cmp "$SP/new-wiring.txt" "$SP/base-wiring.txt" && echo WIRING-SAME
cmp "$SP/wt-base/.loomux/state/graph/cache/ask-index.json" "$SP/baseline/ask-index.json" && echo ASK-SAME
```

Erwartet: `WIRING-SAME` und `ASK-SAME`. `$SP/wt-base` steht schon auf `b5c99cf1` (angelegt 2026-09-26; das alte Binary baut dort byte-gleich die Grundlinie). Bis Task 4 ist `extractor` sogar noch `go/1`. Ab Task 5 ändert sich nur diese eine Zeile. Der Baum am Commit `b5c99cf1` führt keine `.py` außerhalb von Dot-Verzeichnissen, also kommen dort auch nach Task 5 keine Python-Knoten dazu. Wer `WIRING-SAME` nicht bekommt, hat den Go-Pfad verändert: Task ist nicht fertig.

## Review Focus

1. **Zyklische Vererbung** (`class A(B)` und `class B(A)`, auch über zwei Dateien): die Suche nach `self.foo()` in Basisklassen endet, ohne Endlosschleife. Test in Task 6.
2. **CRLF-Zeilenenden** in einer Python-Datei (Windows-Nutzer): dieselben Knoten, Spans und Signaturen wie mit LF; Signaturen ohne `\r`. Test in Task 4.
3. **UTF-8-BOM** am Dateianfang: die Datei wird geparst, Definitionen werden gefunden, `ParseErrors` bleibt 0. Test in Task 4.
4. **Kaputter, veralteter oder fremder Cache** (`extract.json` mit Müll, anderer `version`, Eintrag mit anderer Sprachversion, Eintrag einer gelöschten Datei): Build läuft, parst neu, meldet es über `notice`, der gelöschte Eintrag verschwindet. Test in Task 2.
5. **Paket und Modul gleichen Namens** (`a.py` und `a/__init__.py` im selben Verzeichnis): `import a` löst auf `a/__init__.py` auf, wie CPython (das Paket gewinnt). Test in Task 5.

---

### Task 1: Extraktor-Schnittstelle, kombinierte Version, Index je Sprache (nur Go)

Reiner Umbau, kein neues Verhalten. Am Ende baut loomux denselben Graphen byte-gleich, **inklusive** `"extractor": "go/1"`, denn `all.Version()` ist bei nur einer Sprache `go/1`.

**Files:**
- Create: `internal/code/extract/extract.go`, `internal/code/extract/extract_test.go`
- Create: `internal/code/extract/all/all.go`, `internal/code/extract/all/all_test.go`
- Modify: `internal/code/extract/golang/extract.go` (Typen weg, `Language` dazu), `internal/code/extract/golang/*_test.go` (Typnamen)
- Modify: `internal/code/resolve/resolve.go` (nimmt `[]extract.Result`, gruppiert nach Sprache), `internal/code/resolve/*_test.go`
- Modify: `internal/code/query/build.go`, `check.go`, `refresh.go`, `ask.go`, `load.go` (`all.Version()`, `all.For`)
- Modify: `internal/code/model/graph.go` (`Meta.BuiltBy`), Test in `internal/code/model/graph_test.go` (oder dem bestehenden Testfile des Pakets)
- Modify: `internal/hooks/blast_monitor.go` (`BuiltBy` statt Gleichheit)
- Modify: `internal/code/sourceset/sourceset.go` (`Extensions()`, Endungsliste statt `.go`), `sourceset_test.go`

**Interfaces:**
- Produces:
  ```go
  package extract
  type Language interface {
      Name() string         // "go"
      Version() string      // "go/1"
      Extensions() []string // [".go"]
      File(rel, source string) (Result, error)
  }
  type Import struct {
      Alias string `json:"alias,omitempty"`
      Path  string `json:"path"`
      Name  string `json:"name,omitempty"` // Python `from Path import Name`; Go leaves it empty
  }
  type RawEdge struct {
      Source    model.NodeID   `json:"source"`
      Relation  model.Relation `json:"relation"`
      TargetID  model.NodeID   `json:"target_id,omitempty"`
      Name      string         `json:"name,omitempty"`
      Owner     string         `json:"owner,omitempty"`
      Receiver  string         `json:"receiver,omitempty"`
      Specifier string         `json:"specifier,omitempty"`
      File      string         `json:"file"`
  }
  type Result struct {
      Path        string       `json:"path"`
      Language    string       `json:"language"`
      Package     string       `json:"package,omitempty"`
      Imports     []Import     `json:"imports,omitempty"`
      Nodes       []model.Node `json:"nodes"`
      Edges       []RawEdge    `json:"edges,omitempty"`
      ParseErrors int          `json:"parse_errors,omitempty"`
  }
  package all
  func Languages() []extract.Language           // fixed order: go first
  func Version() string                          // sorted versions joined by "+"; "go/1" today
  func For(rel string) (extract.Language, bool)  // by extension
  func Extensions() []string                     // union, sorted
  package golang
  type Language struct{}                          // implements extract.Language; File delegates to golang.File
  func File(rel, source string) (extract.Result, error) // unchanged logic; fills Language: "go"
  package resolve
  func Graph(files []extract.Result, mods []Module, extractor string) *model.Graph // extractor lands in Meta.Extractor; query passes all.Version()
  package model
  func (m Meta) BuiltBy(version string) bool      // version is one "+"-joined member of m.Extractor
  package sourceset
  func Extensions() []string                     // copy of the fixed list; [".go"] in this task
  ```
- Consumes: nichts aus anderen Tasks.

- [ ] **Step 1: Grundlinie prüfen.** `ls "$SP/baseline"` zeigt `loomux-baseline.exe`, `wiring.json`, `ask-index.json`, `commit.txt` (Inhalt `b5c99cf1…`). Fehlt etwas: STOP und melden (nicht neu bauen, der Baum ist dann schon verändert).

- [ ] **Step 2: Failing tests schreiben.**
  - `internal/code/model`: `TestMetaBuiltBy`: `Meta{Extractor: "go/1"}.BuiltBy("go/1")` true; `Meta{Extractor: "go/1+python/1@gotreesitter/v0.55.0"}.BuiltBy("go/1")` true und `.BuiltBy("python/1@gotreesitter/v0.55.0")` true; `.BuiltBy("go/2")` false; `Meta{}.BuiltBy("go/1")` false; `Meta{Extractor: "go/10"}.BuiltBy("go/1")` false (kein Präfixvergleich).
  - `internal/code/extract/all`: `TestVersionIsSortedJoin` (heute `"go/1"`), `TestForPicksByExtension` (`For("a/b.go")` → Name `go`, ok; `For("a/b.py")` → ok false in dieser Task; `For("Makefile")` → false), `TestExtensionsMatchSourceset`: `reflect.DeepEqual(all.Extensions(), sourceset.Extensions())`.
  - `internal/code/sourceset`: `TestExtensionsIsACopy` (Ändern des Rückgabewerts ändert die nächste Antwort nicht).
  - `internal/code/extract/golang`: bestehende Tests auf `extract.Result`/`extract.RawEdge` umstellen; neu `TestLanguageDescribesGo` (`Name()=="go"`, `Version()==golang.Version`, `Extensions()==[".go"]`, `File` liefert `Result.Language=="go"`).
  - `internal/code/resolve`: `TestGraphKeepsLanguagesApart` mit zwei handgebauten `extract.Result`: eine Go-Datei mit Aufruf `Run()` und Go-Funktion `Run` in einer zweiten Go-Datei, dazu ein `Result{Language: "python"}` mit Funktion `Run`. Erwartet: die Go-Kante `calls` auf die Go-Funktion bleibt (Confidence `inferred`), obwohl `Run` zweimal existiert. Heute würde `global["Run"]` zwei Kandidaten haben und die Kante fallen.

- [ ] **Step 3: Tests laufen lassen, Rot sehen.** `go test ./internal/code/... ./internal/hooks/... > "$SP/t1-red.txt" 2>&1; grep -E "^(FAIL|---)|undefined" "$SP/t1-red.txt" | head -30`. Erwartet: Compile-Fehler (`undefined: extract`, `BuiltBy`, …).

- [ ] **Step 4: Implementieren.**
  - `extract/extract.go`: Paketkommentar (warum ein eigenes Paket ohne Sprache: der Hook-Pfad und `all` hängen daran, keine Sprache darf hier Abhängigkeiten ziehen), die Typen oben. `RawEdge`- und `Import`-Doku aus `golang/extract.go` mitnehmen und um Python ergänzen (`Name` in `Import`).
  - `golang`: die drei Typen löschen, überall `extract.` davor; `r := extract.Result{Path: rel, Language: "go", Package: …}`. `type Language struct{}` mit den vier Methoden; `File` bleibt die bestehende Funktion.
  - `all/all.go`:
    ```go
    // Package all is the fixed list of languages a graph build extracts.
    //
    // A list and not a registry: registration at run time would need init()
    // (AGENTS.md forbids it) or an order nobody can read off the source.
    package all

    func Languages() []extract.Language {
        return []extract.Language{golang.Language{}}
    }

    func Version() string {
        var vs []string
        for _, l := range Languages() {
            vs = append(vs, l.Version())
        }
        sort.Strings(vs)
        return strings.Join(vs, "+")
    }

    func For(rel string) (extract.Language, bool) {
        ext := strings.ToLower(path.Ext(rel))
        for _, l := range Languages() {
            if slices.Contains(l.Extensions(), ext) {
                return l, true
            }
        }
        return nil, false
    }

    func Extensions() []string {
        var out []string
        for _, l := range Languages() {
            out = append(out, l.Extensions()...)
        }
        sort.Strings(out)
        return out
    }
    ```
  - `model.Meta.BuiltBy`: `slices.Contains(strings.Split(m.Extractor, "+"), version)`; leerer Extractor → false (Split von "" liefert `[""]`, `version` ist nie leer — trotzdem explizit `m.Extractor != "" &&`).
  - `sourceset`: `var extensions = []string{".go"}` (unexportiert, nur Literal, kein Parsen), `func Extensions() []string { return slices.Clone(extensions) }`; in `Stat` statt `HasSuffix(".go")`: `slices.Contains(extensions, strings.ToLower(filepath.Ext(d.Name())))`. Doku „Go files“ → „source files of every extracted language“. Paketkommentar: warum die Liste hier steht und nicht aus `extract/all` kommt (Hook-Pfad), und dass `all`'s Test sie gleich hält.
  - `resolve.Graph(files []extract.Result, mods []Module) *model.Graph`: Knoten aller Dateien sammeln; Dateien nach `Language` gruppieren (Reihenfolge der Gruppen: sortiert nach Name); für `"go"`: `idx := index(group)` und die bestehende Kantenlogik; unbekannte Sprache: nur Knoten und `contains`-Kanten (die sind schon aufgelöst), keine weiteren. `Meta.Extractor = all.Version()`? **Nein** — `resolve` darf `all` nicht importieren (Zyklus `all → golang`, `resolve → all` ist okay, aber `resolve` bleibt ohne Wissen über die Liste). Stattdessen `Graph(files, mods, extractor string)`; `query` übergibt `all.Version()`. `Meta.Languages` = sortierte Menge der `Language`-Werte der Dateien (bei loomux `["go"]`, also byte-gleich zur Grundlinie). Bei null Dateien `[]string{}` (JSON `[]`) statt des bisher festen `["go"]` — kein Test pinnt den leeren Fall (geprüft: nur `stats_test.go` und `store_test.go` nennen `Languages`, beide mit Go-Dateien); ein Test `TestEmptyGraphHasNoLanguages` pinnt das neue Verhalten.
  - `index`, `resolveEdge`, `resolveCall`, `resolveSelector`, `packageDir`, `one`: nur Typnamen `golang.X` → `extract.X`. Die Logik bleibt Zeichen für Zeichen.
  - `query`: `golang.Version` → `all.Version()` in `build.go` (freshness.Write), `check.go` (Foreign-Vergleich und Report-Text), `refresh.go`, `ask.go`, `load.go`. `Extract`: `var results []extract.Result`; je Datei `lang, ok := all.For(f.Rel)`; `!ok` → Datei überspringen (`continue`). Im echten Lauf unerreichbar (`TestExtensionsMatchSourceset` hält `sourceset` und `all` gleich), aber **kein** `//coverage:exempt` — das gälte für ganz `Extract`. Stattdessen ein Seam nach dem Muster von `monitorRead` in `hooks/blast_monitor.go`: `var languageFor = all.For` im Paket `query`; ein Test setzt ihn für eine Datei auf `false` und prüft, dass sie im Graphen fehlt. `lang, _ := all.For(...)` ist verboten (nil-Panic, wenn die Listen je auseinanderlaufen).
  - `hooks/blast_monitor.go`: `g.Meta.Extractor != golang.Version` → `!g.Meta.BuiltBy(golang.Version)`. Kommentar: warum Mitgliedschaft und nicht Gleichheit (die kombinierte Kennung) und warum `all` hier nicht importiert wird (Hook-Pfad).

- [ ] **Step 5: Tests grün.** `go test ./internal/code/... ./internal/hooks/... ./internal/cli/... > "$SP/t1-green.txt" 2>&1; tail -20 "$SP/t1-green.txt"` — alles `ok`.

- [ ] **Step 6: Regressionsprobe** (Abschnitt oben). Erwartet `WIRING-SAME`, `ASK-SAME`. Zusätzlich `grep '"extractor"' "$SP/wt-base/.loomux/state/graph/wiring.json"` zeigt `"go/1"`.

- [ ] **Step 7: Commit.**
  ```
  git rev-parse --show-toplevel   # must print the worktree path
  git add -A internal/code internal/hooks
  git commit -m "refactor(graph): put extraction behind a language interface with per-language resolution"
  ```
  Tor muss grün sein. Danach `git log -1 --format='%an <%ae>'` → Christoph Wübbels.

---

### Task 2: Cache je Datei und `graph build --no-reuse`

**Files:**
- Create: `internal/code/query/cache.go`, `internal/code/query/cache_test.go`
- Modify: `internal/code/query/build.go` (`Extract` mit Optionen, `BuildWith`, `Stats` erweitert), `check.go` (Aufruf)
- Modify: `internal/cli/graph.go` (`--no-reuse`, Bericht je Sprache), `internal/cli/graph_test.go` (oder das bestehende Testfile von `graphBuild`/`report`)

**Interfaces:**
- Consumes: `extract.Result`, `all.For`, `all.Version` (Task 1).
- Produces:
  ```go
  package query
  type ExtractOptions struct {
      Reuse  bool         // read cache/extract.json and skip files whose hash and language version match
      Notice func(string) // cache problems; nil means silent
  }
  type LangStats struct {
      Files, Parsed, Reused, ParseErrors int
      ErrorFiles []string // sorted, the files with ParseErrors > 0
  }
  type Stats struct {
      Files       []sourceset.SourceFile
      Hashes      map[string]string
      NoSymbol    int
      PerLanguage map[string]LangStats // key: extract.Language.Name()
      entries     map[string]cacheEntry // what Build writes back; unexported
  }
  func Extract(root string, opts ExtractOptions) (*model.Graph, Stats, error)
  type BuildOptions struct{ NoReuse bool }
  func BuildWith(root string, opts BuildOptions, notice func(string)) (*model.Graph, Stats, error)
  func Build(root string, notice func(string)) (*model.Graph, Stats, error) // = BuildWith(root, BuildOptions{}, notice)
  // cache.go
  const cacheVersion = 1
  type cacheEntry struct {
      Extractor string                  `json:"extractor"` // the language's Version()
      SHA256    string                  `json:"sha256"`
      Result    extract.Result          `json:"result"`
      Bodies    map[model.NodeID]string `json:"bodies,omitempty"` // Node.BodyText is json:"-"
  }
  func cachePath(root string) string                  // store.CachePath(root, "extract.json")
  func readCache(root string) (map[string]cacheEntry, error)
  func writeCache(root string, entries map[string]cacheEntry) error // temp + rename, like store.Write
  ```

- [ ] **Step 1: Failing tests schreiben** (`cache_test.go`, Paket `query` oder `query_test` wie die Nachbarn; Repo-Kopie wie in `golden_test.go` per `copyDir` aus `testdata/cases/graph/repo` in `t.TempDir()`):
  - `TestWarmBuildEqualsNoReuseBuild`: `Build`, dann `Build` (warm), Dateien lesen; dann `BuildWith(NoReuse)`, lesen. `wiring.json` und `cache/ask-index.json` aus warm und no-reuse byte-gleich. Außerdem: zweiter Build hat `PerLanguage["go"].Parsed == 0`, `.Reused == Files`.
  - `TestCacheKeepsBodyText`: nach warmem Build liefert `lexicon.Read(root)` für ein Wort, das nur im Rumpf einer Funktion steht (im Testrepo eines wählen, z. B. einen Bezeichner aus dem Rumpf von `calc.Add`), denselben Treffer wie nach dem kalten Build (vergleiche die `ask-index.json`-Bytes; das deckt es ab — plus ein gezielter Blick, dass `Bodies` im geschriebenen Cache nicht leer ist).
  - `TestChangedFileIsReparsed`: nach Build eine Datei ändern (Funktion hinzufügen), Build → `Parsed == 1`, neuer Knoten im Graphen.
  - `TestGarbageCacheRebuildsWithNotice`: `extract.json` mit `not json` überschreiben → Build ok, alle Dateien `Parsed`, genau ein Notice-Text enthält `cache`. Gleiches mit `{"version": 99, "files": {}}`.
  - `TestForeignLanguageVersionIsReparsed`: Cache-Eintrag per Hand auf `Extractor: "go/0"` setzen → diese Datei wird geparst.
  - `TestDeletedFileLeavesCache`: Datei löschen, Build → Cache hat keinen Eintrag mehr für sie.
  - `TestUnwritableCacheIsANotice`: `cache/extract.json` als **Verzeichnis** anlegen (dann schlägt Rename fehl) → Build ok, Notice.
  - CLI: `graph build --no-reuse` parst alles (Bericht zeigt `parsed` = Dateizahl); Bericht enthält eine Zeile je Sprache: `  go: <n> files, <p> parsed, <r> reused, <e> parse errors`. Bei `ParseErrors > 0` (per Test-Seam kaum erreichbar mit Go — Go bricht ab) wird die Zeile um die ersten fünf Dateien ergänzt: `  go parse errors in: a.go, b.go (+2 more)`; testbar über die reine Funktion `report(g, stats, took)` mit handgebautem `Stats`.

- [ ] **Step 2: Rot sehen.** `go test ./internal/code/query/... ./internal/cli/... > "$SP/t2-red.txt" 2>&1`.

- [ ] **Step 3: Implementieren.**
  - `readCache`: Datei fehlt → `(nil, nil)` ohne Notice (kalter Start ist normal); nicht lesbar/kein JSON/`version != cacheVersion` → `(nil, err)`; der Aufrufer meldet `fmt.Sprintf("extract cache ignored, parsing every file: %v", err)`. Beim Lesen die `Bodies` zurück in `Result.Nodes[i].BodyText` schreiben.
  - `writeCache`: `Bodies` aus `BodyText` füllen (nur nicht-leere), `json.Marshal` (ohne Indent — Größe), temp + rename wie `store.Write`, `MkdirAll(Dir)`.
  - `Extract`: wie bisher jede Datei lesen und hashen (Kommentar dort bleibt). Wenn `opts.Reuse` und Eintrag vorhanden und `SHA256 == hash` und `Extractor == lang.Version()` → `Result` übernehmen (`Reused++`), sonst `lang.File` (`Parsed++`). `ParseErrors` summieren, `ErrorFiles` sammeln. `NoSymbol` wie bisher. `stats.entries[rel] = cacheEntry{…}` für jede Datei (geparst oder übernommen) — so fallen gelöschte Dateien heraus. Reihenfolge der `results` = Reihenfolge von `sourceset.Stat` (sortiert), unabhängig vom Cache.
  - `BuildWith`: `Extract(root, ExtractOptions{Reuse: !opts.NoReuse, Notice: notice})`, dann wie bisher store, lexicon, freshness; danach `writeCache(root, stats.entries)`, Fehler → `notice(fmt.Sprintf("extract cache not written: %v", err))`, Build steht. `Build` delegiert.
  - `check.go`: `Extract(root, ExtractOptions{Reuse: true})` — liest den Cache, schreibt nie.
  - `cli/graph.go`: Flag `no-reuse` („parse every file, ignoring the extract cache“), `query.BuildWith`. `report` bekommt die Sprachzeilen nach den beiden bestehenden Zeilen, sortiert nach Sprachname.

- [ ] **Step 4: Grün.** `go test ./internal/code/... ./internal/cli/... > "$SP/t2-green.txt" 2>&1`.

- [ ] **Step 5: Regressionsprobe** (kalt, wie im Abschnitt oben), danach zusätzlich ein zweiter, warmer `graph build --root "$SP/wt-base"` ohne `rm -rf` — beide Male `WIRING-SAME` und `ASK-SAME`.
- [ ] **Step 5b: Doku von `golang.Version`** um einen Satz ergänzen: mit dem Cache überlebt ein vergessener Versionssprung auch ein ausdrückliches `graph build` (die Einträge passen weiter), nur `--no-reuse` hilft dann — darum jede Änderung an Knoten oder Kanten mit einem Sprung.

- [ ] **Step 6: Commit** `feat(graph): reuse the extraction of unchanged files across builds` (Body: warum `Bodies` ein eigenes Feld ist; `--no-reuse`).

---

### Task 3: `gotreesitter` und der gemeinsame Tree-sitter-Kern

**Files:**
- Modify: `go.mod`, `go.sum` (`go get github.com/odvcencio/gotreesitter@latest` — vorher `go list -m -versions github.com/odvcencio/gotreesitter` lesen und die neueste nehmen; die übrigen `require`-Zeilen mit `go list -m -u all` prüfen und melden, nicht still heben)
- Modify: `internal/code/extract/extract.go` (gemeinsame Helfer), `internal/code/extract/golang/ident.go`, `extract.go` (nutzen die Helfer)
- Create: `internal/code/extract/treesitter/doc.go`, `doc_test.go`, `parser_test.go`
- Modify: `ci/gate.sh` (CGo-Freiheitstor)
- Modify: `internal/cli/imports_test.go` (Abhängigkeitstest)

**Interfaces:**
- Consumes: `extract.Result`, `model.Node` (Task 1).
- Produces:
  ```go
  package extract
  const MaxBodyChars = 5000
  func MintID(base string, minted map[string]bool) string // moved from golang/ident.go, unchanged
  func Collapse(text string) string                        // moved from golang/ident.go, unchanged
  func Hash(text string) string                            // moved from golang/extract.go, unchanged
  package treesitter
  const Parser = "gotreesitter/v0.55.0" // must equal the go.mod pin; see TestParserMatchesGoMod
  type Doc struct {                     // one parsed file
      Rel  string
      Src  []byte
      Lang *gts.Language
      Root *gts.Node
      // unexported: tree, minted, covered
  }
  func Parse(lang *gts.Language, rel string, src []byte) (*Doc, error)
  func (d *Doc) Close()                                  // tree.Release()
  func (d *Doc) Type(n *gts.Node) string
  func (d *Doc) Field(n *gts.Node, name string) *gts.Node
  func (d *Doc) Text(n *gts.Node) string
  func (d *Doc) Symbol(s Sym) model.Node                  // mints the id, marks lines covered
  func (d *Doc) FileNode() model.Node                     // call after every Symbol
  func (d *Doc) ParseErrors() int                         // ERROR and MISSING nodes in the tree
  func InError(n *gts.Node) bool                          // n or an ancestor is an ERROR node
  func Walk(n *gts.Node, visit func(*gts.Node) bool)      // pre-order; visit returns false to skip children
  type Sym struct {
      Outer     *gts.Node // extent of the node: span, body hash, body text (decorators included)
      HeaderEnd uint32    // signature = Collapse(src[Head:HeaderEnd]) with a trailing ':' trimmed
      Head      uint32    // where the signature starts (the def/class keyword, not a decorator)
      Name      string
      Qualified string    // id suffix after '#': "f", "C", "C.m"
      Kind      model.Kind
      Owner     string
      Exported  bool
  }
  ```

- [ ] **Step 1: Abhängigkeit holen.** `go get github.com/odvcencio/gotreesitter@<neueste>`; `go mod tidy` erst am Ende der Task (vorher entfernt es die ungenutzte Abhängigkeit). Neueste Version in `Parser` eintragen.

- [ ] **Step 2: Failing tests.**
  - `extract`: `TestMintIDAppendsLowestFreeOrdinal`, `TestCollapse`, `TestHash` — die bestehenden Tests dieser Funktionen aus `golang` hierher verschieben (nicht duplizieren).
  - `treesitter`, mit der Python-Grammatik aus `github.com/odvcencio/gotreesitter/grammars/python` als Testsprache (nur im Test importiert):
    - `TestParserMatchesGoMod`: liest `../../../../go.mod`, findet die Zeile `github.com/odvcencio/gotreesitter vX.Y.Z`, erwartet `Parser == "gotreesitter/" + "vX.Y.Z"`.
    - `TestSymbolSpanSignatureHash`: `src := "@dec\ndef f(a, b) -> int:\n    return a\n"`; `Outer` = `decorated_definition`, `Head` = Start von `function_definition`, `HeaderEnd` = Start des Felds `body`. Erwartet: `Span == "L1-L3"`, `Signature == "def f(a, b) -> int"`, `BodyHash == extract.Hash(src[0:len-1])` (Ende des Knotens), `BodyText == extract.Collapse(…)`, `ID == "m.py#f"`.
    - `TestSymbolMintsOrdinals`: zweimal `Qualified: "f"` → `m.py#f`, `m.py#f~2`.
    - `TestFileNodeKeepsResidual`: Datei mit Import-Zeile, einer Funktion und einer Modulkonstante; `FileNode().BodyText` enthält Import und Konstante, nicht den Funktionsrumpf; `Span == "L1-L<zeilen>"`; `Exported == true`; `Kind == model.KindFile`; `ID == rel`; `Name == path.Base(rel)`.
    - `TestParseErrorsCounts`: eine Quelle, bei der `Root.HasError()` true ist (vorher per Probe finden; `def broken(:` erzeugt bei `gotreesitter` v0.55.0 **keinen** Fehler — z. B. `class :\n` oder `def f(\n` ausprobieren) → `ParseErrors() > 0`; saubere Quelle → 0.
    - `TestInError`: ein Knoten innerhalb des ERROR-Knotens → true; ein sauberer Knoten → false.
    - `TestWalkSkipsChildren`: `visit` gibt für `class_definition` false zurück → deren Kinder werden nicht besucht.
    - `TestSignatureDropsCR`: dieselbe Quelle mit `\r\n` → Signatur ohne `\r` (`Collapse` erledigt das über `strings.Fields`; der Test pinnt es).
  - `internal/cli/imports_test.go`: `TestHooksNeverReachTreeSitter`: `dependencies(hooksPackage)` enthält kein Paket mit Präfix `github.com/odvcencio/gotreesitter`; Gegenprobe gegen Leerlauf: `dependencies("github.com/xidus90/loomux/internal/code/extract/treesitter")` enthält `github.com/odvcencio/gotreesitter`.

- [ ] **Step 3: Rot sehen.**

- [ ] **Step 4: Implementieren.**
  - Helfer nach `extract` verschieben, `golang` ruft `extract.MintID`, `extract.Collapse`, `extract.Hash`; `golang`s eigene Kopien löschen. `maxBodyChars` → `extract.MaxBodyChars`.
  - `treesitter.Parse`: `tree, err := gts.NewParser(lang).Parse(src)`; `err` → `fmt.Errorf("parse %s: %w", rel, err)` (in der Praxis nur bei inkompatibler Sprache — testbar? wenn nicht: `//coverage:exempt` mit Begründung). Ein Parser je Aufruf (nicht nebenläufig nutzbar laut `gotreesitter`; eine Paketvariable wäre Zustand).
  - `Symbol`: Span aus `Outer.StartPoint().Row+1` / `Outer.EndPoint().Row+1` — **Achtung**: endet ein Knoten mit dem Zeilenumbruch, steht `EndPoint` in der Folgezeile, Spalte 0; dann die Zeile davor nehmen (Go-Extraktor-Äquivalent: `d.End()` ist das Zeichen nach `}`). Test `TestSymbolSpanSignatureHash` pinnt es. Signatur: `strings.TrimSuffix(strings.TrimSpace(extract.Collapse(string(src[Head:HeaderEnd]))), ":")`, dann erneut `TrimSpace`. `covered` wie `golang.markCovered` über `Span.Lines()`.
  - `FileNode`: wie `golang.fileNode` (`residual` über `covered`).
  - `ParseErrors`: `Walk` über den Baum, zählt `IsError() || IsMissing()`.
  - `InError`: `for p := n; p != nil; p = p.Parent() { if p.IsError() { return true } }`.
  - Paketkommentar `doc.go`: der Kern, warum er die Sprachtabellen nicht kennt (Ansatz 3 der Spec), warum ein Parser je Datei.
  - `ci/gate.sh`: vor `go run …` eine Zeile
    ```sh
    # CGo-free gate: loomux must build without a C toolchain.
    cgo_gate="$(mktemp -d)"
    CGO_ENABLED=0 go build -o "$cgo_gate/loomux" ./cmd/loomux
    rm -rf "$cgo_gate"
    ```
    Prüfen, dass `mktemp -d` in Git Bash unter Windows funktioniert (`sh ci/gate.sh` einmal von Hand laufen lassen, Ausgabe in Datei).

- [ ] **Step 5: Grün**, dann `go mod tidy` — `gotreesitter` bleibt, weil `treesitter_test` und (ab jetzt) `treesitter` es importieren. `CGO_ENABLED=0 go build ./...` ok.

- [ ] **Step 6: Regressionsprobe** (die Helfer sind umgezogen) → `WIRING-SAME`, `ASK-SAME`.

- [ ] **Step 7: Commit** `feat(graph): add a tree-sitter core on gotreesitter and a CGo-free build gate`.

---

### Task 4: Python-Extraktor (noch nicht eingehängt)

**Files:**
- Create: `internal/code/extract/python/python.go`, `python_test.go`, `imports.go`, `calls.go` (Aufteilung nach Verantwortung; Namen dürfen abweichen, wenn jede Datei eine Aufgabe hat)

**Interfaces:**
- Consumes: `treesitter.Doc`, `treesitter.Sym`, `treesitter.Walk`, `treesitter.InError`, `treesitter.Parser` (Task 3); `extract.*` (Task 1).
- Produces:
  ```go
  package python
  const version = "python/1"
  type Language struct{}
  func (Language) Name() string         { return "python" }
  func (Language) Version() string      { return version + "@" + treesitter.Parser }
  func (Language) Extensions() []string { return []string{".py"} }
  func (Language) File(rel, source string) (extract.Result, error)
  ```
  Knoten und Kanten nach Spec §7, als **rohe** Kanten:
  - Knoten: `file`; `class` (`rel#C`); `function` (`rel#f`); `method` (`rel#C.m`, `Owner: "C"`). Nur Top-Level-`def`/`class` und `def` direkt im `block` einer Top-Level-Klasse (auch hinter `decorated_definition`). Keine Knoten für geschachtelte `def`/`class` in Funktionen, keine für Klassen in Klassen. Definitionen mit `treesitter.InError` → übersprungen.
  - `Exported`: `!strings.HasPrefix(name, "_") || (strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") && len(name) > 4)`.
  - `contains`: Datei → Klasse/Funktion (`TargetID`), Klasse → Methode.
  - `Imports` (auf `Result.Imports`) und `imports`-Kanten (`Specifier`):
    - `import a.b` → `Import{Path: "a.b"}`, Kante `Specifier: "a.b"`.
    - `import a.b as c` → `Import{Path: "a.b", Alias: "c"}`, Kante `a.b`.
    - `from X import n, m as k` → `Import{Path: X, Name: "n"}`, `Import{Path: X, Name: "m", Alias: "k"}`; **eine** Kante je Anweisung mit `Specifier: X`, plus je Name eine Kante `Specifier: X + "." + name` (bei `X` aus Punkten allein ohne Trenner: `"." + name` wird `".name"`, `".." + name` wird `"..name"`). Der Resolver behält davon nur die, die eine Datei treffen.
    - `X` behält führende Punkte (`from ..pkg.mod import t` → `"..pkg.mod"`, `from . import sib` → `"."`).
    - `from X import *` → `Import{Path: X, Name: "*"}`, nur die Modulkante.
  - `calls` (Quelle: die umgebende Funktion oder Methode; in einer geschachtelten Funktion die äußere; auf Klassenebene die Klasse; auf Modulebene die Datei):
    - `foo()` → `RawEdge{Name: "foo"}`
    - `self.foo()` / `cls.foo()` in einer Methode → `RawEdge{Name: "foo", Owner: <Klasse>}`
    - `x.foo()` mit Bezeichner `x` → `RawEdge{Name: "foo", Receiver: "x"}`; `a.b.foo()` → `Receiver: "a.b"` (nur Ketten aus Bezeichnern)
    - alles andere (`super().foo()`, `f()()`, `x[0].foo()`) → keine Kante
  - `extends`: je Basisklasse im Feld `superclasses` (nur Bezeichner oder Attributketten, keine `keyword_argument`) → `RawEdge{Relation: extends, Source: classID, Name: "Base"}` bzw. `Name: "Base", Receiver: "mod"`.
  - `ParseErrors = doc.ParseErrors()`.

- [ ] **Step 1: Failing tests**, je Regel ein Test mit kleiner Quelle im Test (Go-Rohstring; keine Backslashes nötig — falls doch: Write/Edit):
  - `TestTopLevelFunctionAndClass`, `TestMethodOwnerAndID`, `TestDecoratorInSpanNotInSignature`, `TestNestedDefHasNoNode` (und der Aufruf darin zählt für die äußere Funktion), `TestClassInClassHasNoNode`, `TestExportedRule` (`f`, `_f`, `__f`, `__init__`, `__`, `____`), `TestImportForms` (alle Formen oben, `Result.Imports` und Kanten exakt), `TestCallShapes` (alle vier Formen plus die drei ohne Kante), `TestCallSourceScopes` (Modul, Klasse, Methode, geschachtelte Funktion), `TestExtendsForms`, `TestDefinitionInErrorIsSkipped` (Quelle aus Task 3 mit Fehler; Datei-Knoten bleibt, `ParseErrors > 0`), `TestLanguageDescribesPython` (`Version() == "python/1@" + treesitter.Parser`).
  - Review Focus 2: `TestCRLFMatchesLF` — dieselbe Quelle mit `\n` und mit `\r\n` (per `strings.ReplaceAll` im Test erzeugt) → gleiche Knoten-IDs, Spans, Signaturen, Kanten.
  - Review Focus 3: `TestBOMIsParsed` — `"\uFEFF" + src` (als Go-Escape im Quelltext, über Write) → gleiche Definitionen, `ParseErrors == 0`. Falls `gotreesitter` am BOM scheitert: BOM vor dem Parsen abschneiden und Offsets verschieben (der Span zählt Zeilen, die BOM ändert keine) — das ist dann Teil der Implementierung.

- [ ] **Step 2: Rot sehen.**

- [ ] **Step 3: Implementieren.** Knotentypen und Felder (geprüft an `gotreesitter` v0.55.0): `module`, `function_definition` (Felder `name`, `parameters`, `return_type`, `body`), `class_definition` (`name`, `superclasses` = `argument_list`, `body`), `decorated_definition` (`definition`), `block`, `call` (`function`, `arguments`), `attribute` (`object`, `attribute`), `identifier`, `import_statement` (`name`: `dotted_name` oder `aliased_import`), `aliased_import` (`name`, `alias`), `import_from_statement` (`module_name`: `dotted_name` oder `relative_import`; `name`, mehrfach), `relative_import` (Kinder `import_prefix`, optional `dotted_name`), `wildcard_import`, `keyword_argument`. Wo ein Feld anders heißt: am `SExpr` einer Probe nachsehen und die Tabelle im Code kommentieren.
  - `File`: `lang := pygrammar.Language()` (Paket `github.com/odvcencio/gotreesitter/grammars/python`; cached von `gotreesitter`), `doc, err := treesitter.Parse(lang, rel, []byte(source))`, `defer doc.Close()`.
  - Aufrufe mit `treesitter.Walk` im Rumpf einsammeln; der Besucher kennt die aktuelle Quelle (Funktion/Methode/Klasse/Datei).

- [ ] **Step 4: Grün.** `go test ./internal/code/extract/... > "$SP/t4.txt" 2>&1`.

- [ ] **Step 5: Commit** `feat(graph): extract Python modules, classes, functions and raw edges`.

---

### Task 5: Python einhängen — Dateimenge, Index, Importe, Vererbung

**Files:**
- Modify: `internal/code/extract/all/all.go` (+ `python.Language{}`), `all_test.go`
- Modify: `internal/code/sourceset/sourceset.go` (`extensions = []string{".go", ".py"}`), `sourceset_test.go`
- Create: `internal/code/resolve/python.go`, `python_test.go`
- Modify: `internal/code/resolve/resolve.go` (Gruppe `"python"` → Python-Auflösung)
- Create: `testdata/cases/graph/mixed/repo/…`, `testdata/cases/graph/mixed/graph.golden`
- Modify: `internal/code/query/golden_test.go` (Helfer `graphText` und `TestMixedGolden`)

**Interfaces:**
- Consumes: `python.Language` (Task 4), `resolve.Graph` Gruppierung (Task 1).
- Produces:
  ```go
  package resolve
  type pyIndex struct {
      modules  map[string]string                  // dotted module path -> file rel ("a.b" -> "a/b.py" or "a/b/__init__.py")
      perFile  map[string]map[string][]model.Node // file -> name -> functions and classes
      classes  map[model.NodeID]pyClass            // class id -> its methods and bases
      global   map[string][]model.Node            // name -> functions and classes, python only
      importsOf map[string][]extract.Import
  }
  func resolvePython(files []extract.Result) []model.Edge
  ```
  Konfidenzen und Regeln: Spec §7. Importe ohne Ziel-Datei ergeben **keine** Kante (anders als Go, bewusst: Spec §7).

- [ ] **Step 1: Failing tests** (`resolve/python_test.go`, handgebaute `extract.Result` oder über `python.Language{}.File` aus kleinen Quellen — Letzteres ist lesbarer und erlaubt, weil `resolve_test` Testcode ist):
  - `TestModulePathsFromRootAndSrc`: Dateien `pkg/__init__.py`, `pkg/a.py`, `src/lib/__init__.py`, `src/lib/b.py` → `modules` hat `pkg`, `pkg.a`, `lib`, `lib.b`, `src.lib`, `src.lib.b`. Ein Verzeichnis `src/` ohne Paket (nur lose `.py` ohne `__init__.py` in einem Unterordner) → trotzdem Wurzel? Spec: „jedes Verzeichnis `src/`, das ein Paket enthält“ — Paket = Unterverzeichnis mit `__init__.py`. Lose Module direkt unter `src/` (`src/tool.py`) machen `src/` **nicht** zur Wurzel. Test pinnt beides.
  - Review Focus 5: `TestPackageBeatsModule`: `a.py` und `a/__init__.py` → `import a` zielt auf `a/__init__.py`.
  - `TestRelativeImports`: in `pkg/sub/m.py`: `from . import sib` → `pkg/sub/sib.py`; `from .. import top` → `pkg/top.py`; `from ..x import y` → Modulkante `pkg/x.py` (und `pkg/x/y.py`, falls vorhanden).
  - `TestImportOfSubmoduleName`: `from pkg import a` → Kante auf `pkg/__init__.py` **und** auf `pkg/a.py`.
  - `TestExternalImportHasNoEdge`: `import os`, `from django.db import models` → keine Kante.
  - `TestExtendsSameFileAndImported`: `class B(A)` mit `A` im selben Modul → `extends` `extracted`; `from m import A` → auf `m.py#A`; `class C(mod.A)` mit `import mod` → auf `mod.py#A`; unbekannte Basis → keine Kante.
  - `TestMixedIndexesStayApart` (zusätzlich zu Task 1): über `query.Build` auf dem Golden-Repo unten.
- [ ] **Golden `mixed`:** `testdata/cases/graph/mixed/repo/` mit `go.mod` (`module example.com/mixed`), `main.go` (ruft `Run()`), `run.go` (`func Run()`, `type T struct{}`, `func (T) Run()`), `tool/run.py` (`def Run(): ...`, `class T:` mit `def Run(self)`), `tool/__init__.py`. `graphText(g)` im Test: eine Zeile je Knoten `node <id> <kind> <span> exported=<bool> sig=<signature>` und je Kante `edge <source> -<relation>/<confidence>-> <target>`, sortiert wie im Graphen. `TestMixedGolden` baut und vergleicht mit `graph.golden` (Update per `-update` wie die Nachbarn). **Zusätzlich** baut der Test das Repo ohne `tool/` und prüft, dass jede Kante mit Go-Quelle in beiden Graphen gleich ist.

- [ ] **Step 2: Rot sehen.**

  Hinweis zu Fixtures: `.py`-Dateien unter `testdata/` gibt es schon (`testdata/cases/2a-source/…`); loomux' eigenes `[verify]` erkennt daraus keinen Python-Stapel, und `sourceset` überspringt `testdata/`. Eine Fixture darf also absichtlich kaputt sein — sie wird nie „repariert“.

- [ ] **Step 3: Implementieren.**
  - Quellwurzeln: `""` plus jedes Verzeichnis mit letztem Segment `src`, in dem ein direktes Unterverzeichnis `__init__.py` hat. Modulpfad einer Datei unter Wurzel `r`: Pfad relativ zu `r`, ohne `.py`, `/` → `.`, ein abschließendes `.__init__` entfernt. Kollision (zwei Dateien, ein Modulpfad — nur `a.py` gegen `a/__init__.py` möglich): `__init__.py` gewinnt.
  - Relativer Import in Datei `f`: `level` = Zahl der führenden Punkte; Basis = Paket von `f` (Verzeichnis, als Modulpfad unter der Wurzel, unter der `f` liegt), `level-1` Segmente nach oben; Rest anhängen.
  - Namensauflösung, Aufrufe noch **nicht** (Task 6); hier nur `imports` und `extends`.
  - `resolve.Graph`: Gruppe `"python"` → `resolvePython(group)`; Kanten anhängen, dann wie bisher sortieren.
  - `all`: `python.Language{}` an die Liste; `all_test` erwartet jetzt `Version() == "go/1+python/1@" + treesitter.Parser` (sortiert: `go/1` < `python/1…`) und `Extensions() == [".go", ".py"]`; `sourceset.extensions` entsprechend.

- [ ] **Step 4: Grün** (ganzes `./internal/...`, weil `sourceset` jetzt `.py` liefert: bestehende Tests mit `.py`-Dateien in Testbäumen können sich bewegen — jede Änderung eines bestehenden Goldens begründen oder als Fehler behandeln).

- [ ] **Step 5: Regressionsprobe.** Erwartet `WIRING-SAME` (die `extractor`-Zeile ist herausgefiltert), `ASK-SAME`. `grep '"extractor"'` zeigt `go/1+python/1@gotreesitter/…`, `"languages"` weiter `["go"]`.

- [ ] **Step 6: Commit** `feat(graph): build Python into the graph with its own module index`.

---

### Task 6: Python-Aufrufe und Konstruktoren

**Files:**
- Modify: `internal/code/resolve/python.go`, `python_test.go`
- Create: `testdata/cases/graph/python/repo/…`, `testdata/cases/graph/python/graph.golden`
- Modify: `internal/code/query/golden_test.go` (`TestPythonGolden`)

**Interfaces:**
- Consumes: `pyIndex` (Task 5).
- Produces: `calls`-Kanten nach Spec §7, Regeln 1–5.

- [ ] **Step 1: Failing tests** (`python_test.go`):
  - `TestBareCallSameModule` (`extracted`), `TestBareCallViaFromImport` (`from m import foo` → `m.py#foo`, `extracted`; mit Alias `as bar` → Aufruf `bar()` trifft `m.py#foo`), `TestBareCallUniqueName` (`inferred`), `TestBareCallAmbiguousDropped`.
  - `TestSelfCallOwnClass`, `TestSelfCallBaseClass` (Methode nur in der Basis, Basis in anderem Modul per Import), `TestSelfCallUnknownDropped`.
  - Review Focus 1: `TestCyclicBasesTerminate` — `class A(B)` / `class B(A)` in einem Modul und dasselbe über zwei Module; `self.missing()` → keine Kante, Test endet (mit `t.Deadline`-freundlichem kleinen Repo; eine Endlosschleife ließe den Test hängen, darum `visited`-Menge).
  - `TestModuleReceiverCall` (`import m` → `m.foo()`; `import a.b` → `a.b.foo()`; `import a.b as c` → `c.foo()`), `TestClassReceiverCall` (`Klasse.foo()` → Methode).
  - `TestUnknownReceiverDropped` (`obj.foo()`).
  - `TestConstructorHitsInit` (`Foo()` mit `Foo.__init__` in der Klasse → `rel#Foo.__init__`), `TestConstructorWithoutInitHitsClass`, `TestConstructorInheritedInitHitsClass` (Basis hat `__init__`, die Klasse selbst nicht → Klassenknoten; Spec: „wenn die Klasse selbst eines definiert“), `TestImportedConstructor` (`from m import Foo; Foo()`), `TestModuleConstructor` (`mod.Foo()`).
- [ ] **Golden `python`:** `testdata/cases/graph/python/repo/` deckt jede Regel aus Spec §7 ab: `app/__init__.py`, `app/models.py` (Basisklasse mit `__init__`, Unterklasse, Dunder `__eq__`, private `_helper`), `app/service.py` (relativer Import, `from .models import User`, `self.`-Aufruf, Konstruktor, `mod.fn()`, unbekannter Empfänger, geschachtelte Funktion, Decorator), `app/broken.py` (Fehlerknoten, Quelle aus Task 3 — **bleibt kaputt**; Lint-Meldungen zu dieser Datei sind erwartet und kein Anlass, sie zu reparieren), `tests/test_service.py` (Testpfad für Task 7), `src/lib/__init__.py` + `src/lib/util.py` (Wurzel `src`). `TestPythonGolden`: `query.Build` → `graphText` → `graph.golden`; zusätzlich der Bericht von `graph build` (über `cli`-Funktion `report` oder die Stats) zeigt `python: … 1 parse errors` und `python parse errors in: app/broken.py`.

- [ ] **Step 2: Rot sehen. Step 3: Implementieren.**
  - Reihenfolge je `RawEdge{calls}`: `Owner != ""` → Methoden der Klasse `Owner` **im selben Modul** (die Klasse, in der der Aufruf steht), dann Basen über die aufgelösten `extends` (Breitensuche, `visited`), erste Klasse mit genau einer Methode dieses Namens gewinnt, `extracted`. `Receiver != ""` → Modul (Alias, dann `import a.b` als Kette) → `perFile[datei][Name]`; sonst Klasse (selbes Modul oder Import) → Methode; sonst nichts. Sonst (bloßer Name) → `perFile[eigene Datei][Name]` (`extracted`) → Import-Bindung (`Import.Alias` oder `Import.Name` == Name) → `perFile[Modul][Import.Name]` (`extracted`) → `global[Name]` genau eins (`inferred`).
  - Konstruktor: ist das Ziel ein Knoten `Kind == "class"` → hat `classes[id]` eine eigene Methode `__init__` → deren Knoten, sonst die Klasse. Gilt für bloßen Namen, Import-Bindung, Modul-Empfänger.
- [ ] **Step 4: Grün. Step 5: Regressionsprobe** (`WIRING-SAME`, `ASK-SAME`).
- [ ] **Step 6: Commit** `feat(graph): resolve Python calls, self calls through bases, and constructors`.

---

### Task 7: Testpfade je Sprache und die Graph-Lane für Python

**Files:**
- Modify: `internal/code/blast/radius.go` (`IsTestPath`), `radius_test.go`
- Modify: `internal/verify/presets.toml` (`[stack.python.graph]`), `internal/verify/plan.go` (ein Graph-Job je Lauf), `plan_test.go`, ggf. `presets_test.go`

**Interfaces:**
- Produces: `blast.IsTestPath(path string) bool` (Signatur unverändert, Tabelle nach Endung); `Plan` legt für die Art `graph` genau einen laufenden Job an.

- [ ] **Step 1: Failing tests.**
  - `TestIsTestPathPerLanguage`: `a_test.go` true, `a.go` false; `test_x.py`, `pkg/x_test.py`, `conftest.py`, `pkg/conftest.py`, `tests/x.py`, `a/tests/b/c.py` true; `x.py`, `testing.py`, `attests/x.py`, `mytests/x.py` false; `a.ts` false.
  - `plan_test.go`:
    - `TestGraphRunsOncePerRun`: Effective mit Stapeln `go` und `python`, beide mit Graph-Befehl, `GraphReady` true → genau ein Job mit `Argvs` (`graph/go`, erster in Byte-Reihenfolge), `graph/python` mit `Pre == StateNotApplicable`, `Note == "graph covered by graph/go"`.
    - `TestGoOnlyGraphPlanUnchanged`: nur `go` und `shell` → Jobs und Notizen wie vorher (`graph/go` läuft, `graph/shell` „no command“). Den bestehenden Test, der das heute zeigt, als Beleg nennen.
    - `TestPythonOnlyGraphRuns`: nur `python` → `graph/python` läuft.
    - `TestGraphNotReadyStillNamesCarrier`: `GraphReady` false → der erste Stapel bekommt die Probe-Notiz, der zweite `graph covered by graph/go` (die Deckung hängt am Befehl, nicht an der Bereitschaft).
- [ ] **Step 2: Rot sehen. Step 3: Implementieren.**
  - `IsTestPath`: `switch strings.ToLower(path.Ext(p))`; Python: `base := path.Base(p)`; `strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") || base == "conftest.py" || strings.HasPrefix(p, "tests/") || strings.Contains(p, "/tests/")`. Doku-Kommentar neu (heute „Go only, as long as …“) — die Regel und ihre Herkunft (pytest-Konvention).
  - `presets.toml`: `[stack.python.graph]` mit denselben `commands` wie `[stack.go.graph]` und dem Kommentar, dass der Graph der Wurzel gehört (Plan legt einen Job an).
  - `Plan`: vor `planJob` für `kind == "graph"`: hat der Stapel einen Graph-Befehl (`r.Defined && len(commandsFor(req, stack, r.Lane)) > 0`) und ist schon ein Träger gesetzt → Job `Name: "graph/"+stack`, `Pre: StateNotApplicable`, `Note: "graph covered by graph/"+carrier`; sonst `planJob`, und hat der Stapel einen Befehl → `carrier = stack`. `ScopeEdit` bleibt wie heute (kein Graph-Job).
- [ ] **Step 4: Grün. Step 5: Commit** `feat(graph): recognise Python test files and run one graph lane per project`.

---

### Task 8: Doku

**Files:**
- Modify: `docs/en/migration.md`, `docs/de/migration.md` (Zeilen G5a–d statt G5; G5a „✅ <Datum>“ erst nach Task 9, bis dahin „in progress“/„in Arbeit“; Fähigkeitszeile „Multi-Language AST“: `gotreesitter`, kein `wazero`/AOT-Cache; Mermaid-Knoten `g5` teilen)
- Modify: `README.md`, `README.de.md` (Code-Graph: Sprachen Go und Python; `graph build --no-reuse`; CGo-frei)
- Modify: `docs/wiki/topics/code-graph.md` („Was offen ist“: G5 → G5a gebaut, G5b–d offen; `gotreesitter`)
- Modify: `docs/.superpowers/specs/2026-09-26-loomux-code-g5-design.md` §5 „Bericht“: `graph stats` zählt Sprachen schon nach Endung und bleibt unverändert; Wiederverwendung und Parsefehler zeigt nur `graph build` (so gebaut in Task 2)

- [ ] **Step 1:** Jede Stelle per `grep -rn "wazero\|G5" docs README*.md` finden, alle Sprachen. **Step 2:** Ändern. **Step 3:** Commit `docs(graph): document Python extraction and the gotreesitter runtime`.

---

### Task 9: Abnahme und Messung (Controller, nicht Subagent)

Braucht Urteil und eine ruhige Maschine; führt die steuernde Sitzung selbst aus.

- [ ] **Step 1: Kopien.** `git clone --local "C:/Users/micro/Documents/#GIT/iam_backend" "$SP/acc/iam_backend"`, dasselbe für `ultra-brain`. Nie in den Originalen bauen.
- [ ] **Step 2: Abnahme 1 und 3.** Je Kopie: `loomux-new.exe graph build --root <kopie>` (kalt), noch einmal (warm: `0 parsed`), dann `--no-reuse`; `wiring.json` und `ask-index.json` von warm und no-reuse mit `cmp` vergleichen.
- [ ] **Step 3: Abnahme 2.** Je Kopie drei Symbole von Hand wählen (eine Methode mit `self`-Aufrufern, eine per `from … import` genutzte Funktion, eine Klasse mit Konstruktor-Aufrufern); `graph callers`, `graph ask`, `graph blast` (nach einer Test-Änderung in der Kopie) — Protokoll mit Befehl, Ausgabe-Auszug, Urteil in `docs/.superpowers/parity/code-g5.md`.
- [ ] **Step 4: Abnahme 4.** Die Regressionsprobe ein letztes Mal.
- [ ] **Step 5: Messung** nach Spec §8 in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`: vorher `$SP/baseline/loomux-baseline.exe`, nachher `loomux-new.exe`; `dev bench-hooks` (Fälle nach `testdata/bench/edit-readme.json`, `edit-go.json`) mit `-n 50`, dreimal; **erst messen, wenn der warme Median von `pre-tool-use` mit dem alten Binary nahe den üblichen ~10 ms liegt** — sonst warten und den Nutzer fragen. `inittrace`-Summe, Binary-Größe, `graph build` kalt/warm/nach einer Änderung auf beiden Kopien, Größe von `extract.json`, dazu die Probe aus Spec §2 mit Werkzeugversionen. Steigt der warme Median von `pre-tool-use` um ≥ 3 ms: E1-Ausweg dem Nutzer vorlegen (Spec §10).
- [ ] **Step 6:** `migration.md` (en/de): G5a ✅ mit Datum. Commit `docs(graph): record the acceptance and the measurements of Python extraction`.

### Task 10: Pull Request (Controller)

- [ ] Skill `release-pr`: Commits nach Thema gruppieren (Korrekturen desselben Zweigs in ihren Commit falten; Commit-Texte ohne Arbeitspapier-Namen — die Spec-Commits dieses Zweigs nennen „G5a“ und werden dabei umformuliert), Label mindestens `release:minor`, `## Changelog` (Added: Python in the code graph; `graph build --no-reuse`; Changed: one graph lane per project), `parse-body`. Push-Befehl nennen, nicht pushen. `--assignee @me` und Auto-fix nach dem Öffnen (Memory des Nutzers).
