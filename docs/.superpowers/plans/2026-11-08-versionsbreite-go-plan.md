# Versionsbreite Go und Pre-Releases im Scout – Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Der Code-Graph liest jede gepflegte Go-Version mit richtigen Kanten,
der Compat-Scout prüft Pre-Releases von Go, Python und Godot vorab und meldet
Versionen, deren Pflege upstream endet.

**Architecture:** Alles sitzt im bestehenden Kompatibilitätssystem
(`internal/compat`, `compat.toml`, `compat.yml`, `compat-scout.yml`,
`ci/compat-*.sh`). Der Scout liefert drei Arten von Funden: `new` (wie heute),
`pre` (Pre-Release, nur Probe und Issue, nie Listeneintrag) und `eol` (Issue
ohne Probe). Go-Syntax je Version prüfen eigene Einträge nur mit der Art
`graph`.

**Tech Stack:** Go (`go/parser`), GitHub Actions, POSIX sh, `gh`, `jq`,
endoflife.date API v1.

**Spec:** `docs/.superpowers/specs/2026-11-08-versionsbreite-go-design.md`

## Global Constraints

- Branch: neuer Branch `feat/compat-version-breadth` von `origin/master`. Er
  trägt den Norm-Hunk aus `AGENTS.md` (Rules, „Every language the code graph
  reads …“), die Spec und diesen Plan. Nichts davon auf
  `feat/dsh-loomux-plugin` committen.
- TDD je Task. Coverage 100 % je Funktion; `//coverage:exempt <reason>` nur
  direkt über `func` und mit Begründung.
- Code, Kommentare, Commits englisch; Doku deutsch und englisch in Parität
  (`docs/en/*` ↔ `docs/de/*`, gleiche Dateinamen).
- Conventional Commits, Scope = Codebereich (`compat`, `extract`, `ci`), nie
  ein Arbeitspapier im Commit-Text. Kein Modell als Mitautor.
- Nur ein Mensch pusht. CI-Änderungen gelten erst nach dem Lauf auf GitHub als
  belegt (Task 9).
- Pre-Releases gelangen nie in `compat.toml` und nie in den Pflichtcheck
  `compat` eines Pull Requests.
- Stabile Go-Einträge bauen loomux mit der Toolchain aus `go.mod`
  (`toolchain go1.27.3`); nur Läufe mit einer Go-Version über dieser
  Toolchain bauen mit der eigenen.
- Ende der Pflege: Quelle `https://endoflife.date/api/v1/products/<go|python|godot>`,
  Feld `isEol` je Release (`name` = MAJOR.MINOR). Gepflegt geprüft am
  2026-11-08 (Repo nicht archiviert, letzter Push am selben Tag). Nur
  Laufzeiten, nicht Addons und Tools.
- Ein Austrage-PR hat mindestens `release:patch` und nennt im
  `## Changelog` unter `Removed` die letzte loomux-Version, die die Version
  noch unterstützt.
- Wiki: jede Änderung an `docs/wiki/` endet mit einem Eintrag in
  `docs/wiki/log.md`; meldet `lint/wiki-sources` `register-stale` oder
  `pin-stale`, gilt das Verfahren aus `AGENTS.md` vor `loomux wiki repin`.
- Roadmap: der erste Commit trägt die Zeile „Versionsbreite: Pre-Releases und
  Ende der Pflege im Compat-Scout, Go-Syntax je Version“ unter „Coming“ in
  `README.md` und `README.de.md` ein; Task 8 nimmt sie heraus.

## Review Focus

1. **Ein Pre-Release einer Version, die die Liste schon stabil führt**
   (z. B. `1.27rc3` neben `1.27`): kein Fund. Test in Task 2.
2. **Mehrere Pre-Releases derselben nächsten Version** (`1.28rc1`, `1.28rc2`):
   nur der neueste wird Fund; das Issue zu `rc1` schließt der Report. Tests in
   Task 2 und Task 4.
3. **endoflife.date nicht erreichbar oder Laufzeit dort unbekannt:** Scout
   bricht mit Fehler ab, statt stillschweigend „alles gepflegt“ zu melden.
   Test in Task 3.
4. **Ein Syntax-Eintrag (`kinds = ["graph"]`) wird nie zur Scout-Linie** und
   erzeugt keinen doppelten Fund. Test in Task 7.
5. **Stabile Probe einer Go-Version über `go.mod`** (1.28 wird stabil, Liste
   führt 1.27): baut mit der eigenen Toolchain, nicht grün ohne Prüfung.
   Test in Task 5.

---

### Task 1: Upstream-Antworten mit Pre-Release-Kennung

**Files:**
- Modify: `internal/compat/scout.go` (`Upstream`, `goSource`, `godotSource`,
  `parseReleases`, `HTTPUpstream.Releases`)
- Test: `internal/compat/scout_test.go` (`fakeUp`, `TestParseReleases`,
  `TestHTTPUpstream`)
- Modify: `README.md`, `README.de.md` (Roadmap-Zeile, siehe Global Constraints)

**Interfaces:**
- Produces: `type Release struct { Version string; Pre bool }`;
  `Upstream.Releases(source string) ([]Release, error)`. `Version` ohne
  führendes `v`/`go`/`Python `; bei Godot ohne `-stable`, Pre-Releases mit
  Kennung (`4.8-beta2`).
- Produces: `goSource = "https://go.dev/dl/?mode=json&include=all"`,
  `godotSource = "https://api.github.com/repos/godotengine/godot-builds/releases?per_page=100"`.

- [ ] **Step 1:** Echte Antworten holen und gekürzt als Testdaten in
  `scout_test.go` ablegen (je zwei bis drei Releases, mindestens ein
  Pre-Release): `curl -s "https://go.dev/dl/?mode=json&include=all"`, die
  python.org-Quelle aus `pythonSource`, die godot-builds-Releases. Prüfen und
  im Test kommentieren, ob die Go-Antwort RCs als `stable: false` führt.
- [ ] **Step 2: Failing tests**
  - `TestParseReleasesMarksPreReleases`: Go-Body mit `go1.28rc1`
    (`stable:false`) und `go1.27.3` ergibt
    `[]Release{{"1.28rc1", true}, {"1.27.3", false}}`; Python-Body mit
    `Python 3.16.0a1` (`pre_release:true`) ergibt `{"3.16.0a1", true}`;
    godot-builds mit `4.8-beta2` (`prerelease:true`) und `4.7-stable` ergibt
    `{"4.8-beta2", true}, {"4.7", false}`; Drafts fallen weg.
  - PyPI und Addon-Releases liefern weiter nur stabile mit `Pre:false`
    (bestehende Erwartungen übernehmen).
- [ ] **Step 3:** Run `go test ./internal/compat/ -run 'ParseReleases|HTTPUpstream'` → FAIL.
- [ ] **Step 4:** `Release` einführen, `parseReleases` und `Scout` auf
  `[]Release` umstellen; `Scout` filtert vorerst `Pre` heraus, sodass alle
  bestehenden Scout-Tests unverändert grün bleiben.
- [ ] **Step 5:** Run `go test ./internal/compat/...` → PASS.
- [ ] **Step 6: Commit** `refactor(compat): read pre-release flags from the upstream sources`

### Task 2: Pre-Release-Funde für Laufzeiten

**Files:**
- Modify: `internal/compat/scout.go`, `internal/compat/compat.go` (`Entry`,
  `Key`, `check`)
- Test: `internal/compat/scout_test.go`, `internal/compat/compat_test.go`

**Interfaces:**
- Consumes: `Release` (Task 1).
- Produces: `Entry.Pre string` mit Tags `toml:"-" json:"pre,omitempty"`:
  die volle Pre-Release-Version (`1.28rc1`, `3.16.0a1`, `4.8-beta2`);
  `Entry.Version` bleibt MAJOR.MINOR (`1.28`). `Key()` hängt bei gesetztem
  `Pre` `-<Pre>` an die Laufzeit an (`go/go-1.28rc1/go-module`).
- Produces: `Finding.Kind string` (`json:"kind"`), Werte `"new"`, `"pre"`,
  `"eol"`. `IssueTitle` für `pre`: `compat: <line> <Pre> is out`.
  `Finding.Probe` wird `*Entry` mit `json:"probe,omitempty"` (nil bei eol).
  Ein `pre`-Fund hat leeres `TOML`: ein nur am RC belegter Listeneintrag
  gehört nicht ins Issue.
- Produces: `func parsePre(s string) (ver []int, ok bool)`: liest
  MAJOR.MINOR aus Go `1.28rc1`, Python `3.16.0a1|b2|rc1`, Godot
  `4.8-beta2|-rc1|-dev3`; anderes → `ok=false`.

- [ ] **Step 1: Failing tests**
  - `TestScoutFindsNewestPreReleaseOfNextVersion`: Liste Go 1.26, 1.27;
    Upstream `1.28rc1`, `1.28rc2` (Pre), `1.27.3` → genau ein Fund,
    `Kind:"pre"`, `Probe.Version:"1.28"`, `Probe.Pre:"1.28rc2"`, Titel
    `compat: go-module/go 1.28rc2 is out`.
  - `TestScoutIgnoresPreReleaseOfListedVersion` (Review Focus 1): Upstream
    `1.27rc3` bei gelistetem 1.27 → kein Fund.
  - `TestScoutPrefersStableOverPreRelease`: Upstream `1.28` stabil und
    `1.28rc2` → ein Fund `Kind:"new"`, Version `1.28`.
  - `TestScoutPreReleaseOnlyForRuntimes`: Addon- und Tool-Linien ergeben aus
    Pre-Releases keinen Fund (bestehender Test
    `TestScoutPatchAndPreReleaseNeverCount` deckt Addons/Tools weiter ab).
  - Python (`3.16.0a1` bei Liste bis 3.15) und Godot (`4.8-beta2` bei Liste
    bis 4.7) je ein Fund mit passendem `Pre`.
  - `TestListRefusesPre`: ein `pre = "…"` in `compat.toml` ist
    `unknown key "pre"`; `LoadEntry` einer Probe mit `"pre":"1.28rc1"`
    gelingt.
- [ ] **Step 2:** Run `go test ./internal/compat/ -run 'Scout|List|LoadEntry'` → FAIL.
- [ ] **Step 3:** In `scoutGroup` für `kind == "runtime"` den neuesten
  Pre-Release nehmen, dessen MAJOR.MINOR die Regeln `newMinor`/`newMajor`
  erfüllt, nur wenn kein stabiler Fund derselben MAJOR.MINOR existiert.
  `check` lässt `Pre` nur zu, wenn `parsePre(e.Pre)` dieselbe MAJOR.MINOR wie
  `e.Version` liefert.
- [ ] **Step 4:** Run `go test ./internal/compat/...` → PASS.
- [ ] **Step 5: Commit** `feat(compat): scout pre-releases of Go, Python and Godot`

### Task 3: Funde zum Ende der Pflege

**Files:**
- Modify: `internal/compat/scout.go` (neue Quelle `eolSource`, `Scout`)
- Test: `internal/compat/scout_test.go`
- Modify: `docs/en/support.md`, `docs/de/support.md` (Abschnitt zur Liste:
  wann eine Version herausfällt, Changelog-Satz)

**Interfaces:**
- Produces: `const eolSource = "https://endoflife.date/api/v1/products/%s"`;
  `Upstream.EOL(runtime string) (map[string]bool, error)`: MAJOR.MINOR →
  `isEol`; `HTTPUpstream` liest `result.releases[].{name,isEol}`.
- Produces: Fund `Kind:"eol"`, `Line:"<runtime>"`, `Version:"<MAJOR.MINOR>"`,
  kein `Probe`, leeres `TOML`; Titel
  `compat: <runtime> <MAJOR.MINOR> is past end of support`.
- Produces: `Resolved(entries, title)` ist für einen eol-Titel genau dann
  wahr, wenn kein Eintrag diese Laufzeit-Version führt.

- [ ] **Step 1: Failing tests**
  - `TestScoutReportsEndOfSupport`: Liste Python 3.10 und 3.11, EOL-Antwort
    `{"3.10":true,"3.11":false}` → genau ein eol-Fund für 3.10.
  - `TestScoutEOLIsPerRuntimeNotPerFixture`: Godot 4.5 in zwei Fixtures,
    EOL 4.5 → ein Fund, nicht zwei.
  - `TestScoutFailsWhenEOLUnknown` (Review Focus 3): Quelle liefert Fehler,
    oder eine gelistete Version fehlt in der Antwort → `Scout` gibt Fehler mit
    der Laufzeit und Version zurück.
  - `TestResolvedEOL`: Titel `compat: python 3.10 is past end of support` ist
    `open` bei Liste mit 3.10, `resolved` ohne.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implementieren; `fakeUp` um eine EOL-Map erweitern.
- [ ] **Step 4:** Run `go test ./internal/compat/...` → PASS.
- [ ] **Step 5:** Doku in `support.md` (en/de): Liste folgt endoflife.date;
  Austrage-PR mit `## Changelog`/`Removed` und Satz „<runtime> <version> is no
  longer supported; v<x.y.z> is the last release that supports it.“
- [ ] **Step 6: Commit** `feat(compat): report runtime versions past end of support`

### Task 4: Report – überholte Pre-Release-Issues und eol-Issues

**Files:**
- Modify: `internal/compat/scout.go` (`Superseded`), `internal/cli/compat.go`
  (`devCompatScout`), `ci/compat-report.sh`, `.github/workflows/compat-scout.yml`
- Test: `internal/compat/scout_test.go`, `internal/cli/compat_test.go`
  (bestehende Datei der CLI-Tests; sonst neben `compat.go` anlegen)
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (neue Flag),
  `docs/en/releases.md`, `docs/de/releases.md` (Austrage-PR ≥ `release:patch`)

**Interfaces:**
- Consumes: `Finding.Kind`, eol-Titel (Tasks 2, 3).
- Produces: `func Superseded(title string, findings []Finding) bool`: wahr
  für einen pre-Titel, den kein aktueller Fund trägt.
- Produces: CLI `loomux dev compat scout --resolved <title> [--findings <file>]`:
  gibt `resolved` aus, wenn `Resolved` oder (mit `--findings`) `Superseded`
  zutrifft.

- [ ] **Step 1: Failing tests**
  - `TestSupersededPreRelease` (Review Focus 2): `…go 1.28rc1 is out` ist
    überholt, wenn die Funde nur `…1.28rc2…` tragen, und wenn die Funde leer
    sind; nicht überholt, wenn der eigene Titel dabei ist; ein `new`- oder
    eol-Titel ist nie überholt.
  - CLI-Test: `--resolved` mit `--findings` auf eine Datei mit rc2-Fund gibt
    für den rc1-Titel `resolved` aus.
- [ ] **Step 2:** Run → FAIL. **Step 3:** Implementieren. **Step 4:** Run → PASS.
- [ ] **Step 5:** `compat-report.sh`: `--findings "$findings"` an
  `--resolved` geben; Body je `kind`: `new` wie heute; `pre` sagt, dass ein
  Pre-Release vorab geprüft wird und nichts in `compat.toml` gehört; `eol`
  ohne Probe-Abschnitt, mit dem Satz aus Task 3 und der aktuellen Version aus
  `gh release list --exclude-pre-releases --limit 1 --json tagName`.
  `compat-scout.yml`: `probes` nur aus Funden mit `kind != "eol"`
  (`jq -c '[.[] | select(.kind != "eol") | .probe]'`).
- [ ] **Step 6:** `sh -n ci/compat-report.sh`; shellcheck, falls das Repo es
  in `ci.yml` fährt.
- [ ] **Step 7:** Doku `cli-reference.md` (en/de) für `--findings`,
  `releases.md` (en/de) für das Austragen.
- [ ] **Step 8: Commit** `feat(compat): close superseded pre-release issues and file end-of-support issues`

### Task 5: Proben installieren Pre-Releases und bauen mit eigener Toolchain

**Files:**
- Modify: `internal/compat/run.go` (`Describe`), `ci/compat-setup.sh`,
  `.github/workflows/compat.yml`
- Test: `internal/compat/describe_test.go`

**Interfaces:**
- Consumes: `Entry.Pre` (Task 2).
- Produces: `Describe` gibt zusätzlich `"install":"<Version>"` aus: `Pre`,
  falls gesetzt, sonst `Version`, je Laufzeit in der Schreibweise des
  Installers (Go für `actions/setup-go`, Python für `uv python install`,
  Godot als godot-builds-Tag ohne `-stable`-Zwang). Die Schreibweisen vorher
  in den Docs von `actions/setup-go`, uv und godot-builds nachlesen und als
  Testfälle festhalten.
- Produces: `compat-setup.sh` schreibt `install=…` und `rebuild=true|false`
  nach `GITHUB_OUTPUT`; `rebuild` ist `true` genau dann, wenn die Laufzeit
  `go` ist und die Version über der Toolchain aus `go.mod` liegt.

- [ ] **Step 1: Failing tests** in `describe_test.go`:
  `install` für `{go 1.28, Pre 1.28rc1}`, `{python 3.16, Pre 3.16.0a1}`,
  `{godot 4.8, Pre 4.8-beta2}` und für stabile Einträge gleich `Version`.
- [ ] **Step 2:** Run → FAIL. **Step 3:** Implementieren. **Step 4:** Run → PASS.
- [ ] **Step 5:** `compat-setup.sh`: `install_godot` nimmt `${install}` und
  hängt `-stable` nur bei stabilen an; Python installiert `${install}`;
  `rebuild` per Vergleich mit der `toolchain`-Zeile aus `go.mod` (Review
  Focus 5: auch für eine stabile Probe über `go.mod`).
- [ ] **Step 6:** `compat.yml`: zweites `setup-go` nutzt
  `steps.setup.outputs.install`; neuer Schritt
  `if: steps.setup.outputs.rebuild == 'true'` →
  `GOTOOLCHAIN=local go build -o bin/loomux.exe ./cmd/loomux`.
- [ ] **Step 7:** `sh -n ci/compat-setup.sh`; lokal
  `sh ci/compat-setup.sh bin/loomux.exe <key> <tmp>` mit `PROBE` einer
  Go-RC-Probe gegen ein Fake-`GITHUB_OUTPUT` → `rebuild=true`.
- [ ] **Step 8: Commit** `ci(compat): install pre-release runtimes and build probes above go.mod with their own toolchain`

### Task 6: Go-Extraktor liest die Sprachmittel von 1.26 und 1.27

**Files:**
- Modify: `internal/code/extract/golang/extract.go`, `scope.go`, `ident.go`
  (nur wo ein Test rot wird)
- Test: `internal/code/extract/golang/edges_test.go`, neue Testdaten
  `testdata/versions.go.txt`

**Interfaces:**
- Produces: keine neue API; Knoten-IDs bleiben `<rel>#<Name>` bzw.
  `<rel>#<Typ>.<Methode>`.

- [ ] **Step 0:** Jede Form unten gegen `https://go.dev/ref/spec` (Composite
  literals, Method declarations, Type parameter declarations) und die Release
  Notes `go.dev/doc/go1.26`, `go.dev/doc/go1.27` prüfen; die Notizen hier
  stammen aus einer Zusammenfassung. Eine Form, die die Spec nicht trägt,
  entfällt.
- [ ] **Step 1: Failing tests** gegen `versions.go.txt`, je Sprachmittel
  eine Erwartung im Stil von `hasEdge`:
  - Go 1.26 `new(expr)`: `func f() *int { return new(g()) }` → `calls`-Kante
    `f` → `g`.
  - Go 1.26 selbstbezüglicher Typparameter
    `type Adder[A Adder[A]] interface{ Add(A) A }` und
    `func algo[A Adder[A]](x, y A) A { return x.Add(y) }` → Knoten `Adder`,
    `algo`; kein Fehler, keine Kante auf einen Phantomknoten `A`.
  - Go 1.27 generische Methode
    `func (s *Stack[T]) Map[U any](f func(T) U) *Stack[U]` → Knoten
    `Stack.Map`; ein Aufruf `s.Map[int](h)` aus `use` ergibt `calls` von
    `use` mit Name `Map` und Empfänger `s` (wie bei nicht-generischen
    Methoden).
  - Go 1.27 Feldselektor-Schlüssel `T{a.b: v()}` → `calls` auf `v`.
  - Go 1.27 Typinferenz in Zuweisung `var h func(int) int = id` → keine
    Kante, kein Fehler (Wertreferenz, wie bisher; `TestFileEmitsNeither…`).
- [ ] **Step 2:** Run `go test ./internal/code/extract/golang/` → FAIL wo
  der Extraktor die Form nicht kennt; Tests, die sofort grün sind, bleiben als
  Regressionstests.
- [ ] **Step 3:** Extraktor nachziehen (Typparameterliste einer Methode wie
  bei `ft.TypeParams` in `extract.go:337` binden; Aufruf mit Typargumenten am
  Selektor schälen wie `IndexListExpr` in `ident.go`).
- [ ] **Step 4:** Run `go test ./internal/code/...` → PASS.
- [ ] **Step 5:** `docs/wiki/topics/code-graph.md`, Abschnitt „Wie der Graph
  entsteht“, Go-Punkt: gelesen werden die gepflegten Go-Versionen, geprüft je
  Version über die Kompatibilitätsliste, Pre-Releases vorab ohne Gate; Eintrag
  in `docs/wiki/log.md`.
- [ ] **Step 6: Commit** `feat(extract): read Go 1.26 and 1.27 syntax with its edges`

### Task 7: Syntax-Einträge je Go-Version in der Liste

**Files:**
- Create: `internal/compat/testdata/go-syntax-1.26/` und
  `internal/compat/testdata/go-syntax-1.27/` (je `go.mod.txt` mit
  `go 1.26` bzw. `go 1.27`, eine `.go`-Datei aus den Formen von Task 6,
  `expect.toml` mit `stacks = ["go"]` und den Kanten)
- Modify: `internal/compat/compat.toml`, `internal/compat/scout.go`
  (Gruppierung), `internal/compat/compat_test.go`,
  `docs/en/support.md`, `docs/de/support.md` (Tabelle; `docs_test.go` hält sie
  zellgenau gegen die Liste)

**Interfaces:**
- Consumes: Extraktor aus Task 6.
- Produces: Einträge `stack="go" runtime="go" version="1.26"
  fixture="go-syntax-1.26" kinds=["graph"]` und dasselbe für 1.27.
- Produces: Regel „Syntax-Eintrag“ = Eintrag mit `Kinds` genau `["graph"]`;
  `Scout` bildet aus ihm keine Linie.

- [ ] **Step 1: Failing tests**
  - `TestScoutSkipsSyntaxEntries` (Review Focus 4): Liste `go-module` 1.27 und
    `go-syntax-1.27`; Upstream 1.28 → genau ein Fund mit Linie
    `go-module/go`.
  - `TestEveryGoVersionHasASyntaxEntry`: für jede Go-Version der geladenen
    Liste existiert ein Syntax-Eintrag mit Fixture `go-syntax-<version>`.
  - `TestSyntaxFixtureNeedsItsVersion`: jedes `go-syntax-<N>`-Fixture
    (ohne Imports) wird mit `go/types` zweimal geprüft:
    `types.Config{GoVersion: "go<N>"}` meldet keinen Fehler,
    `GoVersion: "go<N-1>"` mindestens einen. `go/parser` allein unterscheidet
    die Versionen nicht, erst die Typprüfung. Eine Form, die `go/types` nicht
    nach Version sperrt, fliegt aus dem Fixture (sie bleibt Regressionstest in
    Task 6).
  - `TestListGoVersionsNotAboveGoModToolchain`: höchste Go-Version der Liste
    ≤ MAJOR.MINOR der `toolchain`-Zeile aus `../../go.mod`.
  - `layer1_test.go`/`run_test.go`-Stil: `Run` des Syntax-Eintrags 1.27 ist
    grün, und eine fehlende Kante in `expect.toml` macht ihn rot.
- [ ] **Step 2:** Run → FAIL. **Step 3:** Fixtures, Einträge, Gruppierung,
  Tabelle in `support.md` (en/de). **Step 4:** Run `go test ./internal/compat/...` → PASS.
- [ ] **Step 5: Commit** `test(compat): pin the syntax of every listed Go version`

### Task 8: Wiki und Roadmap abschließen

**Files:**
- Modify: `docs/wiki/topics/code-graph.md` (falls Task 6 nicht schon alles
  sagt), `docs/wiki/log.md`, `README.md`, `README.de.md`

- [ ] **Step 1:** Roadmap-Zeile in beiden READMEs ersetzen durch
  „Syntax je Version für Python und Godot“ unter „Coming“ (nächster Schritt:
  Python-Session); der erledigte Teil steht in `support.md` und
  `code-graph.md`.
- [ ] **Step 2:** `loomux check stop` bzw. `lint/wiki`, `lint/wiki-sources`
  grün; bei `pin-stale` das Verfahren aus `AGENTS.md`.
- [ ] **Step 3:** `go test ./...` → PASS; `go vet ./...` sauber.
- [ ] **Step 4: Commit** `docs: describe the compatibility scout's pre-release and end-of-support findings`

### Task 9: Abnahme auf GitHub (Mensch)

- [ ] **Step 1:** Mensch pusht `feat/compat-version-breadth` und öffnet den
  PR (Skill `release-pr`; Label `release:minor`).
- [ ] **Step 2:** Mensch startet `compat-scout` per `workflow_dispatch`.
  Erwartet: kein Go-Pre-Release-Fund, solange es keinen RC gibt; ein
  Python-Pre-Release-Fund, falls `3.16.0a*` erschienen ist; keine eol-Funde
  (Stand 2026-11-08 sind alle gelisteten Versionen bei endoflife.date
  gepflegt, Godot 3.6 eingeschlossen).
- [ ] **Step 3:** Mensch startet `compat` per `workflow_dispatch` mit
  `entries` = die beiden Syntax-Einträge → grün auf Linux und Windows.
- [ ] **Step 4:** Ergebnis der Läufe im PR festhalten; erst dann gilt der Plan
  als erledigt.
