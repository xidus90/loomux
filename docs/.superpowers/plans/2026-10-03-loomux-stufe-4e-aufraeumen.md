# Stufe 4e: der Aufräum-PR — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Lese-Rückfälle aus der Zeit neben ultra-brain verschwinden aus loomux: kein zweites Zustandsverzeichnis mehr, keine alten Manifestnamen im brain-Leser, kein `Manifest.Lanes`, und die Ratschläge nennen loomux' eigene Befehle.

**Architecture:** Ein neuer Leser `config.ReadAreaDeclaration(dir)` liest nur `.loomux/config.toml` und unterscheidet `ErrNoManifest` (keine Datei, mit Hinweis auf ein daliegendes Altmanifest) von `ErrNoArea` (Policy ohne `[area]`); `config.IsUndeclared` fasst beide für die Aufrufer zusammen, die „nichts deklariert“ hinnehmen. Danach fallen `LegacyBrainDirUntilStage3`, `ArtifactLookup.Fallback`, `ResolvedAreaDir` und jeder durchgereichte `fallbackDir`; `moveStock` schrumpft auf `recoverStock` (nur noch `lock.Recover`). Die Ratschläge wechseln auf `loomux reindex|reconcile|embed`, und die Fälle 1b-1/1b-2 folgen über Umschreibregeln des Imports, nicht von Hand.

**Tech Stack:** Go 1.x (Repo-Stand), `github.com/BurntSushi/toml` (third_party), die Fallsuiten unter `internal/cli/cases*_test.go`, `loomux dev import-cases`.

**Spec:** `docs/.superpowers/specs/2026-10-03-loomux-stufe-4e-aufraeumen-design.md` (freigegeben 2026-10-03). Nutzerentscheide vom 2026-10-03 darüber hinaus: (a) Spec freigegeben; (b) die sechs offenen Entscheidungen aus `parity/artefakte-nach-lebensdauer.md` **blockieren** 4e ✅ — dieser PR setzt 4e nicht auf ✅, sondern zieht den Text „offen“ der Zeile 4e nach; (c) dass `maintenance`, `lint`, `convert` und `reconcile` einen Arbeitsbereich ohne `[area]` auslassen, ist ein eigener späterer PR — hier nur eine Roadmap-Folgezeile. PR #67 (`fix(brain): skip a workspace without an area`, `ec3c48c9`) ist gemergt; die Vorbedingung aus Entscheidung 9 gilt.

## Global Constraints

- Zweig `refactor/stage-4e-cleanup`, Basis `origin/master` `ec3c48c9`. Niemand arbeitet auf `master`; kein Agent pusht.
- Coverage 100 % je Funktion; Ausnahme nur mit `//coverage:exempt <Grund>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Code, Kommentare, Fehlermeldungen, Commits englisch; Arbeitspapiere unter `docs/.superpowers/` deutsch; `docs/en` und `docs/de` mit gleichen Dateinamen.
- Commits: Conventional Commits, kein Arbeitspapier im Text (kein „4e“, „stage“, „task“, „plan“), Autor der Mensch, **keine** Modell-Zeile, kein `Co-Authored-By`. Mehrzeilige Nachrichten per Write in eine Scratch-Datei und `git commit -F <datei>`; die Ausgabe jedes Commits in eine Scratch-Datei lenken, nie verwerfen.
- Kein Agent schreibt `.loomux/config.toml`, ruft `loomux init`, `config set|apply` oder `area add` gegen echte Projekte oder das echte Zustandsverzeichnis. Die Registry ändert nur der Mensch (Task 0).
- Testwelten setzen `LOOMUX_STATE_DIR` **und** `XDG_CONFIG_HOME` auf die Welt (Memory `isolate-qmd-config-in-test-worlds`).
- Aufgezeichnete Fälle unter `testdata/cases/<stufe>/` werden nie von Hand geändert; sie entstehen per `loomux dev import-cases`.
- Release-Label: `release:major`. Die Commits der Tasks 2 und 3 tragen `!` und einen Footer `BREAKING CHANGE:` — `.ultra-brain/config.toml` und `.brain.toml` werden nicht mehr gelesen (Konfigurationsformat), `LOOMUX_LEGACY_BRAIN_DIR` entfällt (dokumentierte Umgebungsvariable).
- Overlay-Pfade für `go test -overlay` immer in Windows-Form mit Vorwärtsschrägstrichen (`C:/…`); ein Mutant, der nicht baut, ist BADMUTANT, nie „getötet“.
- `internal/cli` braucht ~75–110 s für die volle Suite: Mutationsrunden dort nur als Handrunde mit gezieltem `-run`.
- Jeder Task endet mit einem grünen `sh ci/gate.sh` vor dem Commit (das pre-commit-Tor fährt es ohnehin).

## Review Focus

1. **Eine Workspace-Konfiguration, die ganz fehlt** (nicht nur ohne `[area]`): `VisibleAreas` muss sie genauso auslassen. Test: `TestVisibleAreasLeavesOutAWorkspaceWithoutAnyConfigFile` (Task 2).
2. **Ein Altmanifest neben einer fehlenden `.loomux/config.toml`** in einem registrierten Bereich: die Meldung nennt die Datei und `loomux area check`, bleibt aber `ErrNoManifest` (sonst ändern die Aufrufer, die `IsUndeclared` fragen, ihr Verhalten). Test: `TestReadAreaDeclarationNamesAnOldManifestBesideTheMissingOne` (Task 1), CLI-Sicht in `TestBrainNamesAnOldManifestBesideTheMissingOne` (Task 2).
3. **Eine gesperrte `.loomux/config.toml` mit `local_only`**: Fehler, nie „nicht deklariert“ — sonst öffnet ein Rückfall auf Defaults ein geschlossenes Gebiet. Test: `TestReadAreaDeclarationPassesOnALockedFile` (Task 1) und die verbliebene Zeile in `TestVisibleManifest` (Task 2).
4. **Ein `index`-Lauf, der mitten im Tausch stirbt, während `approve` läuft**: der Bestand liegt als Aside, kein Ziel; `approve` muss ihn zurücklegen statt neben ihm zu schreiben. Test: `TestApprovePutsAnAsideStockBackBeforeItAdvancesTheRegister` (Task 3).
5. **`LOOMUX_LEGACY_BRAIN_DIR` bleibt in einer Nutzerumgebung gesetzt**: loomux darf daraus nichts mehr lesen, weder Bestand noch Stempel. Test: `TestBrainReadsNothingFromTheOldStateDirectory` (Task 3) und `TestNewArtifactLookupTakesTheStateDirectoryAlone` (Task 3).

## Was die Spec behauptet und der Code an HEAD nicht hält

Geprobt am 2026-10-03 in zwei Scratch-Worktrees (`714d4b27`), Ergebnis vor dem Schreiben dieses Plans:

1. **Keine Brücke nötig, keine freigegebene Abweichung.** Die Spec zählt Altnamen in `1b-1-worlds` 28, `1b-2-worlds` 25, `3a-worlds` 21, … und folgert, „fast jede Fallsuite“ falle ohne Übersetzung beim Bereitstellen. Die `*-worlds`-Bäume sind aber **Aufnahmequellen**; die Suiten stellen `testdata/cases/<stufe>/<verb>/<name>/world` bereit, und die sind beim Import längst übersetzt. In allen Lauf-Bäumen aller zwölf Suiten liegen genau **zwei** Altmanifeste:

   | Datei | Wirkung |
   |---|---|
   | `3a/area-add/known-scope/world_after/repo-new/.ultra-brain/config.toml` | Erwartete Datei, die loomux nicht schreibt; steht als „missing file“ in `expected3a`. Kein Leser. |
   | `4d/convert/no-registry/world/vault/.brain.toml` | Der Fall endet an der fehlenden Registry, bevor ein Manifest gelesen wird. |

   Gemessen: mit `ReadAreaManifestUntilStage4` auf nur `.loomux/config.toml` und `LegacyBrainDirUntilStage3() == ""` laufen `TestRecordedCasesOfStage1a`, `…1b1`, `TestRecordedMCPCasesOfStage1b2`, `TestCases2a|2b|2c|3a|3b|3c|4a2|4c1|4d` alle grün (`go test ./internal/cli -run 'TestRecorded|Cases' -count=1`, 42 s). Die Fälle mit absichtlich kaputtem Manifest (`missing-manifest`, `manifest-without-scope`, `manifest-wrong-type`) tragen bereits `.loomux/config.toml` oder gar keine Datei und vergleichen `outcome` bzw. leeres stdout; ihre Wortlaute in `loomuxWording` (`no manifest found`, `[area] scope must be …`) bleiben. **Übersetzer beim Bereitstellen: 0 Fälle. Freigegebene Abweichungen: 0 Fälle.** Der Abschnitt „Aufgezeichnete Fälle“ der Spec entfällt ersatzlos; `TranslateWorld` bleibt nur als Importwerkzeug.
2. **Ratschläge: 12 Fälle, nicht 10.** Neben den zehn der Spec trägt `brain-status/backlog` in 1b-1 (`stdout`) und 1b-2 (`result`) `run \`brain embed\`` (`internal/brain/status/status.go:234`), das die Spec nicht nennt. Der Import reproduziert beide Korpora heute byte-gleich (geprobt: Re-Import mit den Karten an HEAD, `git status` leer). Weg: drei `[[stdout]]`-Regeln für 1b-1, drei neue `[[result]]`-Regeln für 1b-2 (der MCP-Import kennt noch keine; Task 5 baut sie). Nach Re-Import ändern sich genau 6 + 6 Dateien, alle Suiten grün (geprobt).
3. **Entscheidung 7, Test „Aside, kein Ziel, `approve` → eingeräumt“ kann so nicht eintreten.** `resolveSources` liest das Register **vor** `advanceRegisterLocked`; liegt der Bestand schon vorher als Aside, findet `approve` die Quelle nicht und verweigert, bevor es schreibt. Das Fenster, das `recoverStock` schließt, ist ein `index`, der **nach** dem Lesen und **vor** dem Fortschreiben stirbt. Der Test in Task 3 legt den Aside deshalb während des Seitenschreibens an (Naht `replaceText`); ohne `lock.Recover` scheitert `AdvanceRegister` an der fehlenden Datei (geprobt: Overlay-Mutant „`recoverStock` gibt `nil`“ macht ihn rot).
4. **Entscheidung 8, „beide bleiben Fehler“, gilt nur für `VisibleAreas`.** Sieben Aufrufer nehmen „nichts deklariert“ heute hin (`nearestVault`, `hubFolder`, `areaManifest`, `convert.Areas`, `maintenance.Manifests`, `DeclaredTypesIn`, `sweepContext`); für sie war eine Policy-Datei ohne `[area]` bisher `ErrNoManifest`. Sie fragen künftig `config.IsUndeclared(err)` und behandeln `ErrNoArea` genau wie vorher — verhaltensgleich. Ohne das würde jede Policy-Datei dort zum harten Fehler.
5. **Der Workspace-Fix aus PR #67 bricht ohne Nachzug.** `VisibleAreas` lässt einen Workspace nur bei `ErrNoManifest` aus; die `iam_*` tragen eine `.loomux/config.toml` ohne `[area]`, die der neue Leser mit `ErrNoArea` beantwortet. Geprobt: `TestVisibleAreasLeavesOutAWorkspaceWithoutADeclaration` wird rot. Erzeuger und Verbraucher gehören in denselben Task (Task 2).
6. **`ResolvedAreaDir` hat 18 Aufrufer, aber `fallbackDir` reicht weiter, als die Liste der Spec zeigt:** `answer.RunFor`/`RunWith` und der Funktionstyp `func(answer.Request, string, string, func(string))` in `serve.Options.Answer` und `servebrain.Deps.Answer`, `servegraph.Deps.LegacyDir`, `benchsearch.chain.fallback`, und im Benchmark `LOOMUX_BENCH_LEGACY` (`internal/brain/search/bench_test.go`). Die vollständige Tabelle steht in Task 3 und ist kompiliert.
7. **`ArtifactLookup` bleibt als Hülle** (Spec Entscheidung 2 überlässt es dem Plan): 40 Signaturen tragen `config.ArtifactLookup`; die Hülle mit nur `Primary` ist der kleinere Diff. `Resolve` und `WritePath` antworten danach gleich; `apply.registerRead`/`registerWrite` werden zu einem `registerOf`.
8. **Doku-Zeilennummern der Spec stimmen nicht durchweg** (`cli-reference.md:735` ist `convert`, nicht `area check`). Der Plan nennt Stellen per Suchtext.
9. **`OldManifestNames` statt einer Hilfe mit Stufennamen.** Die Namensliste lebt in `internal/config` (`config` kann `internal/cli` nicht importieren); `area check` nimmt sie von dort. Eine Quelle, wie die Spec will, nur am anderen Ende.

## Dateien

| Datei | Verantwortung | Task |
|---|---|---|
| `internal/config/areadeclaration.go` (neu) | `ReadAreaDeclaration`, `IsUndeclared`, `OldManifestNames` | 1 |
| `internal/config/areadeclaration_test.go` (neu) | Tests dazu | 1 |
| `internal/config/legacy.go`, `legacy_test.go` | erst ohne `ReadAreaManifestUntilStage4` (2), dann gelöscht (3) | 2, 3 |
| 11 brain-/cli-/setup-Aufrufer (Liste Task 2) | neuer Leser, `IsUndeclared` | 2 |
| `internal/cli/areacheck.go` | eigene Auswahl `chosenManifest` | 2 |
| `internal/config/artifacts.go`, `artifacts_test.go` | Hülle ohne Rückfall | 3 |
| `internal/brain/apply/stock.go`, `stock_test.go`, `resolve.go`, `approve.go`, `approve_test.go`, `place_test.go` | `recoverStock`, `registerOf` | 3 |
| ~25 Produktdateien (Tabelle Task 3), 42 Testdateien | `fallbackDir` entfällt | 3 |
| `internal/config/manifest.go`, `manifest_test.go` | `Lanes`, `LaneConfig`, `Check` entfallen | 4 |
| `internal/dev/importcases/importcases.go`, `mcp.go`, `result_test.go` (neu) | `[[result]]` | 5 |
| `internal/brain/graph/read.go`, `search/stamp.go`, `status/status.go` + Tests, `testdata/cases/1b-1-map.toml`, `1b-2-map.toml`, 12 Fälle | Ratschläge | 6 |
| `docs/en|de/*`, `README*.md`, `docs/wiki/topics/datenmodell-und-bereiche.md`, `testdata/cases/README.md`, Arbeitspapiere | Doku, Akten | 2, 3, 6, 7, 8 |

---

### Task 0: Registry-Schritt (Mensch) und Vorher-Messung

Der Mensch ändert die Registry vor dem Merge; die Vorher-Messung braucht nur ein Binär von `origin/master` und blockiert keinen der folgenden Tasks.

**Files:** keine im Repo (Scratch und `%LOCALAPPDATA%\loomux\registry.toml`, nur vom Menschen).

- [ ] **Step 1: Registry-Schritt dem Menschen vorlegen und auf Bestätigung warten.** Es gibt keinen Befehl (`loomux area` kennt nur `add` und `check`). Wortlaut an den Menschen:

  > In `C:/Users/micro/AppData/Local/loomux/registry.toml` bitte von Hand, in einem Durchgang:
  > 1. die ganzen `[[area]]`-Blöcke mit `scope = "project/obsidian-ai"`, `scope = "project/ultra-brain"` und `scope = "project/ultraloom"` löschen;
  > 2. in den Blöcken `project/iam-backend`, `project/iam-frontend`, `project/iam-workers` die Zeile `wiki = "C:/Users/micro/Documents/#GIT/iam_wiki"` löschen (gewollt ist die Form, die `init --brain=none` schreibt: `scope`, `path`, `workspace = true`).
  >
  > Danach dürfen `%LOCALAPPDATA%\loomux\areas\project-obsidian-ai\` (samt `.lock`), die `.lock` von `project-ultra-brain` und `project-ultraloom` und die Reste `project-iam-wiki.alt`, `project-space.alt` gelöscht werden; nichts liest sie.

- [ ] **Step 2: Nur lesend prüfen.**

  Run: `loomux brain catalog --scope all > <scratch>/after-registry-catalog.txt 2>&1; echo $?` und `loomux brain status > <scratch>/after-registry-status.txt 2>&1; echo $?`
  Expected: beide Exit 0; keiner der drei entfernten Scopes in der Ausgabe; `grep -c 'readonly' %LOCALAPPDATA%/loomux/registry.toml` → 0.

- [ ] **Step 3: Vorher-Binär und Messbasis in einem Scratch-Worktree.**

  ```bash
  S=<scratch>/measure
  git worktree add --detach "$S/before" origin/master > "$S/wt-before.log" 2>&1
  ```

  Expected: Worktree auf `ec3c48c9` (oder dem dann aktuellen `origin/master`, Hash im Benchmark-Eintrag notieren).

- [ ] **Step 4: Vorher messen (kalt, dann fünf warm).** Gemessen werden die drei Benchmarks in `internal/brain/search/bench_test.go`, die nur lesen: `BenchmarkVisibleAreasOfTheRealRegistry`, `BenchmarkRegistersOfTheRealRegistry` und `BenchmarkExecuteSearchWithoutTheEngine` (die ganze `ExecuteSearch` mit ersetzter Maschine).

  ```bash
  cd "$S/before"
  LOOMUX_BENCH_REGISTRY="$LOCALAPPDATA/loomux" LOOMUX_BENCH_LEGACY="$LOCALAPPDATA/brain" \
    go test ./internal/brain/search -run '^$' -bench 'RealRegistry|ExecuteSearchWithoutTheEngine' -benchtime=1x -count=1 > "$S/before-cold.txt" 2>&1
  LOOMUX_BENCH_REGISTRY="$LOCALAPPDATA/loomux" LOOMUX_BENCH_LEGACY="$LOCALAPPDATA/brain" \
    go test ./internal/brain/search -run '^$' -bench 'RealRegistry|ExecuteSearchWithoutTheEngine' -count=5 > "$S/before-warm.txt" 2>&1
  ```

  Expected: drei Benchmarks je Lauf, kein `--- SKIP`, kein `FAIL`. Datum und Uhrzeit notieren. Erwartung für den Eintrag (ehrlich): nach dem Registry-Schritt gibt es keinen read-only-Bereich mehr, `VisibleAreas` ändert sich durch den PR nicht messbar; gespart wird nur das `os.Stat(primary)`, das `ArtifactLookup.Resolve` vor jeder Antwort machte (Stempel, Sammlungsliste). Ein Unterschied unter dem Rauschen ist das erwartete Ergebnis, kein Befund.

- [ ] **Step 5: Den Vorher-Worktree stehen lassen bis Task 7, dann entfernen** (`git worktree remove "$S/before"`, `git worktree prune`).

---

### Task 1: Der neue Leser `config.ReadAreaDeclaration`

**Files:**
- Create: `internal/config/areadeclaration.go`
- Create: `internal/config/areadeclaration_test.go`

**Interfaces:**
- Consumes: `config.ReadDeclaration(path string) (*Manifest, error)`, `config.ErrNoManifest`, `config.ErrNoArea`.
- Produces: `func ReadAreaDeclaration(dir string) (*Manifest, error)`, `func IsUndeclared(err error) bool`, `func OldManifestNames() []string` (in dieser Reihenfolge: `filepath.Join(".ultra-brain", "config.toml")`, `".brain.toml"`).

- [ ] **Step 1: Stubs mit dem alten Verhalten anlegen**, damit der rote Lauf an Assertions scheitert und nicht am Build. `internal/config/areadeclaration.go`:

  ```go
  package config

  import (
  	"errors"
  	"path/filepath"
  )

  func OldManifestNames() []string {
  	return []string{filepath.Join(".ultra-brain", "config.toml"), ".brain.toml"}
  }

  func ReadAreaDeclaration(dir string) (*Manifest, error) { return ReadAreaManifestUntilStage4(dir) }

  func IsUndeclared(err error) bool { return errors.Is(err, ErrNoManifest) }
  ```

- [ ] **Step 2: Die Tests schreiben.** `internal/config/areadeclaration_test.go`:

  ```go
  package config

  import (
  	"errors"
  	"os"
  	"path/filepath"
  	"strings"
  	"testing"

  	"github.com/xidus90/loomux/internal/testlock"
  )

  // declareIn writes body under name below dir.
  func declareIn(t *testing.T, dir, name, body string) {
  	t.Helper()
  	path := filepath.Join(dir, name)
  	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
  		t.Fatal(err)
  	}
  	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
  		t.Fatal(err)
  	}
  }

  func TestReadAreaDeclarationReadsTheLoomuxName(t *testing.T) {
  	dir := t.TempDir()
  	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"project/a\"\n[privacy]\nmode = \"local_only\"\n")
  	got, err := ReadAreaDeclaration(dir)
  	if err != nil || got.Scope != "project/a" || got.PrivacyMode != "local_only" {
  		t.Fatalf("got %+v, %v", got, err)
  	}
  	if got.Path != filepath.Join(dir, ".loomux", "config.toml") {
  		t.Fatalf("path %q", got.Path)
  	}
  }

  // An old name beside the new one changes nothing: the new one is the
  // declaration, whatever the old one says.
  func TestReadAreaDeclarationIgnoresAnOldNameBesideTheNewOne(t *testing.T) {
  	dir := t.TempDir()
  	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"new\"\n")
  	declareIn(t, dir, ".brain.toml", "[area\n")
  	got, err := ReadAreaDeclaration(dir)
  	if err != nil || got.Scope != "new" {
  		t.Fatalf("got %+v, %v", got, err)
  	}
  }

  // A policy-only file is ErrNoArea and no longer sends the reader on to an
  // old name, which once declared the area in its place.
  func TestReadAreaDeclarationAnswersAPolicyOnlyFileWithErrNoArea(t *testing.T) {
  	dir := t.TempDir()
  	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[verify]\n")
  	declareIn(t, dir, ".brain.toml", "[area]\nscope = \"old\"\n")
  	got, err := ReadAreaDeclaration(dir)
  	if !errors.Is(err, ErrNoArea) || errors.Is(err, ErrNoManifest) || got != nil {
  		t.Fatalf("got %+v, %v; want ErrNoArea alone", got, err)
  	}
  }

  func TestReadAreaDeclarationWithoutAFileIsErrNoManifestWithoutAHint(t *testing.T) {
  	dir := t.TempDir()
  	_, err := ReadAreaDeclaration(dir)
  	want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + ")"
  	if !errors.Is(err, ErrNoManifest) || err.Error() != want {
  		t.Fatalf("got %v; want %q", err, want)
  	}
  }

  // A directory under the name declares nothing, and neither does one under an
  // old name: the hint names files only.
  func TestReadAreaDeclarationTakesOnlyRegularFiles(t *testing.T) {
  	dir := t.TempDir()
  	for _, name := range []string{filepath.Join(".loomux", "config.toml"), ".brain.toml"} {
  		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
  			t.Fatal(err)
  		}
  	}
  	_, err := ReadAreaDeclaration(dir)
  	if !errors.Is(err, ErrNoManifest) || strings.Contains(err.Error(), "old manifest") {
  		t.Fatalf("got %v; want ErrNoManifest without a hint", err)
  	}
  }

  // Each old name alone is named in the hint, together with the command that
  // shows what to carry over; the error stays ErrNoManifest. With both, the
  // one ultra-brain asked first is named.
  func TestReadAreaDeclarationNamesAnOldManifestBesideTheMissingOne(t *testing.T) {
  	for _, tc := range []struct {
  		names []string
  		named string
  	}{
  		{[]string{".brain.toml"}, ".brain.toml"},
  		{[]string{filepath.Join(".ultra-brain", "config.toml")}, filepath.Join(".ultra-brain", "config.toml")},
  		{[]string{".brain.toml", filepath.Join(".ultra-brain", "config.toml")}, filepath.Join(".ultra-brain", "config.toml")},
  	} {
  		dir := t.TempDir()
  		for _, name := range tc.names {
  			declareIn(t, dir, name, "[area]\nscope = \"old\"\n")
  		}
  		_, err := ReadAreaDeclaration(dir)
  		want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + "); an old manifest lies there (" +
  			tc.named + "): `loomux area check " + dir + "` shows what to carry over"
  		if !errors.Is(err, ErrNoManifest) || err.Error() != want {
  			t.Fatalf("%v: got %v\nwant %s", tc.names, err, want)
  		}
  	}
  }

  // A file that is there and does not read is the reader's error, never an
  // absence -- not even beside an old name that would read.
  func TestReadAreaDeclarationPassesOnABrokenFile(t *testing.T) {
  	dir := t.TempDir()
  	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area\n")
  	declareIn(t, dir, ".brain.toml", "[area]\nscope = \"old\"\n")
  	_, err := ReadAreaDeclaration(dir)
  	if err == nil || IsUndeclared(err) || !strings.Contains(err.Error(), "not valid TOML") {
  		t.Fatalf("got %v; want the parse error", err)
  	}
  }

  // A file that is there and cannot be opened is an error naming it, never an
  // absence: taken for one, a locked `local_only` declaration would open its
  // area to the defaults.
  func TestReadAreaDeclarationPassesOnALockedFile(t *testing.T) {
  	dir := t.TempDir()
  	path := filepath.Join(dir, ".loomux", "config.toml")
  	declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"local_only\"\n")
  	testlock.Lock(t, path)
  	_, err := ReadAreaDeclaration(dir)
  	if err == nil || IsUndeclared(err) || !strings.HasPrefix(err.Error(), path+": cannot be read: ") {
  		t.Fatalf("got %v; want the read error naming %s", err, path)
  	}
  }

  // The post-merge hook asks this reader for consent, the default branch
  // included.
  func TestReadAreaDeclarationCarriesTheMergeConsent(t *testing.T) {
  	for _, row := range []struct {
  		maintenance string
  		onMerge     bool
  		branch      string
  	}{
  		{"[maintenance]\non_merge = true\nbranch = \"trunk\"\n", true, "trunk"},
  		{"[maintenance]\non_merge = true\n", true, DefaultMergeBranch},
  		{"", false, DefaultMergeBranch},
  	} {
  		dir := t.TempDir()
  		declareIn(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"a\"\n"+row.maintenance)
  		m, err := ReadAreaDeclaration(dir)
  		if err != nil || m.OnMerge != row.onMerge || m.MergeBranch != row.branch {
  			t.Fatalf("%q: got %+v, %v", row.maintenance, m, err)
  		}
  	}
  }

  func TestIsUndeclaredTakesExactlyTheTwoAbsences(t *testing.T) {
  	for _, tc := range []struct {
  		err  error
  		want bool
  	}{
  		{ErrNoManifest, true},
  		{ErrNoArea, true},
  		{errors.New("x: " + ErrNoManifest.Error()), false},
  		{errors.New("not valid TOML"), false},
  		{nil, false},
  	} {
  		if got := IsUndeclared(tc.err); got != tc.want {
  			t.Fatalf("IsUndeclared(%v) = %v, want %v", tc.err, got, tc.want)
  		}
  	}
  }
  ```

- [ ] **Step 3: Rot laufen lassen.**

  Run: `go test ./internal/config/ -run 'ReadAreaDeclaration|IsUndeclared' -count=1 > <scratch>/t1-red.log 2>&1`
  Expected (geprobt): `--- FAIL` genau für `TestReadAreaDeclarationAnswersAPolicyOnlyFileWithErrNoArea`, `TestReadAreaDeclarationWithoutAFileIsErrNoManifestWithoutAHint`, `TestReadAreaDeclarationNamesAnOldManifestBesideTheMissingOne`, `TestIsUndeclaredTakesExactlyTheTwoAbsences`; die übrigen sechs sind grün (gleiches Verhalten wie der alte Leser). Den Abschnitt mit den vier `--- FAIL` in den Bericht.

- [ ] **Step 4: Die Implementierung einsetzen.** `internal/config/areadeclaration.go` ganz ersetzen:

  ```go
  package config

  import (
  	"errors"
  	"fmt"
  	"os"
  	"path/filepath"
  )

  // OldManifestNames are ultra-brain's two names for an area declaration, in
  // the order its reader asked them. loomux reads neither; they are named so
  // that `area check` can show what to carry over and ReadAreaDeclaration can
  // point there. They go when `area check` goes.
  func OldManifestNames() []string {
  	return []string{filepath.Join(".ultra-brain", "config.toml"), ".brain.toml"}
  }

  // ReadAreaDeclaration reads the declaration of the area whose manifest lies in
  // dir: `.loomux/config.toml`, checked whole by ReadDeclaration.
  //
  // Two answers are no defect of the file, and callers tell them apart: no
  // regular file of that name is ErrNoManifest, a file without [area] -- policy
  // only -- is ErrNoArea. Where an old manifest lies beside a missing one, the
  // error says so and names `area check`, since a reader that ignores it
  // silently would leave the area undeclared without a word.
  func ReadAreaDeclaration(dir string) (*Manifest, error) {
  	name := filepath.Join(".loomux", "config.toml")
  	path := filepath.Join(dir, name)
  	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
  		for _, old := range OldManifestNames() {
  			if info, err := os.Stat(filepath.Join(dir, old)); err == nil && info.Mode().IsRegular() {
  				return nil, fmt.Errorf("%s: %w (%s); an old manifest lies there (%s): `loomux area check %s` shows what to carry over",
  					dir, ErrNoManifest, name, old, dir)
  			}
  		}
  		return nil, fmt.Errorf("%s: %w (%s)", dir, ErrNoManifest, name)
  	}
  	return ReadDeclaration(path)
  }

  // IsUndeclared says that err is one of the two answers of ReadAreaDeclaration
  // that mean "this directory declares no area": no manifest, or one without
  // [area]. Every other error is a declaration that is there and does not read.
  func IsUndeclared(err error) bool {
  	return errors.Is(err, ErrNoManifest) || errors.Is(err, ErrNoArea)
  }
  ```

- [ ] **Step 5: Grün laufen lassen, Coverage.**

  Run: `go test ./internal/config/ -count=1 -coverprofile=<scratch>/t1.out > <scratch>/t1.log 2>&1; go tool cover -func=<scratch>/t1.out | grep areadeclaration`
  Expected: `ok`; `OldManifestNames`, `ReadAreaDeclaration`, `IsUndeclared` je 100.0 % (geprobt).

- [ ] **Step 6: Mutationsrunde** (Paket `config` ist schnell; je Mutant eine Kopie von `areadeclaration.go` per Overlay, `-run 'ReadAreaDeclaration|IsUndeclared' -count=1`). Pflicht-Mutanten und der Test, der sie tötet:

  | Mutant | tötet |
  |---|---|
  | `!info.Mode().IsRegular()` → `false` | `TestReadAreaDeclarationTakesOnlyRegularFiles` |
  | Hinweisschleife: `info.Mode().IsRegular()` → `true` | `TestReadAreaDeclarationTakesOnlyRegularFiles` |
  | `OldManifestNames` Reihenfolge vertauscht | `…NamesAnOldManifestBesideTheMissingOne` (dritte Zeile) |
  | `%w` → `%v` im Hinweis | `…NamesAnOldManifest…` (`errors.Is`) |
  | `\|\| errors.Is(err, ErrNoArea)` gestrichen | `TestIsUndeclaredTakesExactlyTheTwoAbsences` |
  | `return ReadDeclaration(path)` → `return nil, ErrNoArea` | `…ReadsTheLoomuxName` |

  Überlebende mit Begründung in `docs/.superpowers/parity/stufe-4e.md`, Abschnitt „Überlebende Mutanten“, neuer Absatz „Die Runde für den Aufräum-PR (2026-10-…)“.

- [ ] **Step 7: Commit.** Nachricht per Write nach `<scratch>/msg1.txt`:

  ```
  feat(config): read an area declaration under loomux's name alone

  ReadAreaDeclaration reads .loomux/config.toml and nothing else. A missing
  file is ErrNoManifest, and where an old .ultra-brain/config.toml or
  .brain.toml lies beside it the error names that file and `loomux area
  check`. A file without [area] is ErrNoArea. IsUndeclared takes both, for
  the readers that accept an area that declares nothing.
  ```

  Run: `git add internal/config/areadeclaration.go internal/config/areadeclaration_test.go && git commit -F <scratch>/msg1.txt > <scratch>/commit1.log 2>&1; echo $?`

---

### Task 2: Jeder Leser auf `.loomux/config.toml`; `area check` wählt selbst

**Files:**
- Modify: `internal/brain/apply/resolve.go` (`nearestVault`), `internal/brain/check/house/federation.go` (`hubFolder`), `internal/brain/check/run/run.go` (`areaManifest`, `fileRoot`, `fileRoot`s zweiter Leser), `internal/brain/convert/run.go` (`Areas`), `internal/brain/index/reindex.go` (`indexArea`), `internal/brain/maintenance/reconcile.go` (`Manifests`), `internal/brain/privacy/channel.go` (`VisibleManifest`), `internal/brain/privacy/areas.go` (`VisibleAreas`), `internal/brain/wiki/census.go` (`DeclaredTypesIn`), `internal/cli/lintsweep.go` (`sweepContext`), `internal/setup/facts.go` (Zeile `declared, err := …`), `internal/cli/areacheck.go`
- Modify: `internal/config/legacy.go` (nur `manifestNamesUntilStage4` und `ReadAreaManifestUntilStage4` raus), `internal/config/legacy_test.go`
- Test: `internal/brain/privacy/areas_test.go`, `privacy_test.go`, `nesting_test.go`, `internal/brain/search/visibility_test.go`, `search_test.go`, `nesting_test.go`, `internal/brain/apply/resolve_test.go`, `stock_test.go`, `internal/brain/index/reindex_test.go`, `staging_test.go`, `internal/brain/check/run/run_test.go`, `internal/brain/answer/nesting_test.go`, `internal/brain/status/status_test.go`, `internal/brain/wiki/census_test.go`, `internal/brain/maintenance/world_test.go`, `internal/cli/lintsweep_test.go`, `mergehook_test.go`, `wikicmd_test.go`, `brain_test.go`, `internal/lock/replacedir_test.go`
- Docs: `docs/en|de/getting-started.md` (Absatz „Every `loomux brain` command fails with `no manifest found`“), `docs/wiki/topics/datenmodell-und-bereiche.md` (Absatz ab „`.brain.toml` ist der Name bis zum Umzug“)

**Interfaces:**
- Consumes: `config.ReadAreaDeclaration`, `config.IsUndeclared`, `config.OldManifestNames` (Task 1).
- Produces: keine neuen Namen; `config.ReadAreaManifestUntilStage4` existiert danach nicht mehr. `cli.chosenManifest(root string) string` (paketintern).

- [ ] **Step 1: Den Workspace-Rotlauf vorbereiten.** Nur `internal/brain/privacy/channel.go` umstellen:

  ```go
  	manifest, err := config.ReadAreaDeclaration(dir)
  ```

  (statt `config.ReadAreaManifestUntilStage4(dir)` in `VisibleManifest`). Und den Gegenfall an `internal/brain/privacy/areas_test.go` anhängen:

  ```go
  // A workspace without any config file is left out the same way: both
  // answers of "no declaration here" mean no brain area.
  func TestVisibleAreasLeavesOutAWorkspaceWithoutAnyConfigFile(t *testing.T) {
  	registryDir, legacyDir := workspaceWorld(t, true, "[verify]")
  	ws := filepath.Join(filepath.Dir(registryDir), "workspace")
  	if err := os.Remove(filepath.Join(ws, ".loomux", "config.toml")); err != nil {
  		t.Fatal(err)
  	}
  	got, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
  	if err != nil || scopesOf(got) != "project/a" {
  		t.Fatalf("got %q, %v", scopesOf(got), err)
  	}
  }
  ```

- [ ] **Step 2: Rot.**

  Run: `go test ./internal/brain/privacy/ -run 'Workspace' -count=1 > <scratch>/t2-red.log 2>&1`
  Expected (geprobt): `--- FAIL: TestVisibleAreasLeavesOutAWorkspaceWithoutADeclaration` mit `…\.loomux\config.toml: the configuration declares no [area]`; `…WithoutAnyConfigFile` grün.

- [ ] **Step 3: Die Workspace-Regel nachziehen** — in `internal/brain/privacy/areas.go`:

  ```go
  		if area.Workspace && config.IsUndeclared(err) {
  ```

  (statt `errors.Is(err, config.ErrNoManifest)`; der Import `errors` entfällt, wenn er sonst unbenutzt ist). Den Doc-Kommentar von `VisibleAreas` ab „The manifest of each area comes from config.ResolvedAreaDir“ bleibt bis Task 3; den Satz „The one exception is a workspace entry without a declaration“ lassen.

- [ ] **Step 4: Grün.**

  Run: `go test ./internal/brain/privacy/ -run 'Workspace' -count=1`
  Expected: `ok`.

- [ ] **Step 5: Die übrigen Aufrufer umstellen.** In jeder Datei der Liste oben:
  - `config.ReadAreaManifestUntilStage4(` → `config.ReadAreaDeclaration(` (12 Stellen: `apply/resolve.go:90`, `federation.go:372`, `run.go:349,465,503`, `convert/run.go:38`, `reindex.go:167`, `reconcile.go:270`, `channel.go:53` schon erledigt, `census.go:136`, `lintsweep.go:93`, `setup/facts.go:213`);
  - `errors.Is(err, config.ErrNoManifest)` → `config.IsUndeclared(err)` an den sieben duldenden Stellen (`apply/resolve.go:91`, `federation.go:374`, `run.go:351`, `convert/run.go:40` als `case config.IsUndeclared(err):`, `reconcile.go:272`, `census.go:137`, `lintsweep.go:95` als `case config.IsUndeclared(err):`);
  - danach unbenutzte `"errors"`-Importe entfernen (geprobt: `privacy/areas.go`, `wiki/census.go`, `check/house/federation.go`, `apply/resolve.go`, `check/run/run.go`, `cli/lintsweep.go`).

  `internal/setup/facts.go` bleibt bei `f.OnMerge = err == nil && declared.OnMerge` — eine Policy-Datei ohne `[area]` stimmte schon vorher keinem Hook zu.

- [ ] **Step 6: `area check` bekommt seine Auswahl.** In `internal/cli/areacheck.go` die Variable `manifestNames` ersetzen und `chosenManifest` dahinter einfügen:

  ```go
  // manifestNames are the files this command reports on: loomux's one name,
  // then ultra-brain's two in the order its reader asked them.
  var manifestNames = append([]string{filepath.Join(".loomux", "config.toml")}, config.OldManifestNames()...)

  // chosenManifest is the name a reader that still knew the old names took:
  // the first regular file that declares an area. A `.loomux/config.toml`
  // without [area] gives way to the next name; an old name without [area], or
  // any name that does not read, chooses none. loomux reads only the first
  // name; this line tells which file carried the declaration before.
  func chosenManifest(root string) string {
  	for _, name := range manifestNames {
  		path := filepath.Join(root, name)
  		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
  			continue
  		}
  		_, err := config.ReadDeclaration(path)
  		if errors.Is(err, config.ErrNoArea) && name == manifestNames[0] {
  			continue
  		}
  		if err != nil {
  			return ""
  		}
  		return name
  	}
  	return ""
  }
  ```

  und in `areaCheck` die drei Zeilen ab `chosen := ""` ersetzen durch:

  ```go
  	chosen := chosenManifest(root)
  ```

- [ ] **Step 7: Den alten Leser löschen.** In `internal/config/legacy.go` `manifestNamesUntilStage4` samt Kommentar und `ReadAreaManifestUntilStage4` samt Kommentar entfernen, die dann unbenutzten Importe `errors`, `fmt`, `strings` auch. In `internal/config/legacy_test.go` die Funktionen `TestTheThreeNamesAreTriedInOrder` bis einschließlich `TestALockedFirstNameIsAnErrorEvenBesideAnOpenLegacyName` und `TestEveryNameCarriesTheMergeConsent` löschen (Gesperrt und Zustimmung decken die Tests aus Task 1), den dann unbenutzten Import `internal/testlock` auch. `legacyWrite`, `TestReadManifestStillKnowsOnlyTheLoomuxName` und `TestReadManifestStillAcceptsAManifestWithoutAScope` bleiben bis Task 3.

  Run: `go build ./... && go vet ./... > <scratch>/vet2.log 2>&1; echo $?`
  Expected: `go vet` meldet nur noch `stock_test.go:55` und `maintenance/world_test.go:213` (`undefined: config.ReadAreaManifestUntilStage4`); beide auf `config.ReadAreaDeclaration` umstellen, die Fehlermeldung in `world_test.go` mit. Danach Exit 0.

- [ ] **Step 8: Die Testwelten auf den einen Namen umstellen.** Skript per Write nach `<scratch>/rename_fixtures.py`:

  ```python
  # Renames old manifest names in test fixtures to loomux's one name.
  import pathlib, sys

  root = pathlib.Path(sys.argv[1])
  for rel in sys.argv[2:]:
      p = root / rel
      s = p.read_bytes().decode("utf-8")
      before = s
      s = s.replace(', ".brain.toml")', ', ".loomux", "config.toml")')
      s = s.replace('filepath.Join(".ultra-brain", "config.toml")', 'filepath.Join(".loomux", "config.toml")')
      s = s.replace(', ".ultra-brain", "config.toml")', ', ".loomux", "config.toml")')
      s = s.replace('manifestName: ".brain.toml"', 'manifestName: ".loomux/config.toml"')
      s = s.replace('manifestName: ".ultra-brain/config.toml"', 'manifestName: ".loomux/config.toml"')
      if s != before:
          p.write_bytes(s.encode("utf-8"))
          print("changed", rel)
  ```

  Run: `python <scratch>/rename_fixtures.py . internal/brain/answer/nesting_test.go internal/brain/apply/resolve_test.go internal/brain/apply/stock_test.go internal/brain/check/run/run_test.go internal/brain/index/reindex_test.go internal/brain/index/staging_test.go internal/brain/privacy/areas_test.go internal/brain/privacy/nesting_test.go internal/brain/privacy/privacy_test.go internal/brain/search/nesting_test.go internal/brain/search/search_test.go internal/brain/search/visibility_test.go internal/brain/status/status_test.go internal/brain/wiki/census_test.go internal/cli/lintsweep_test.go internal/cli/mergehook_test.go internal/cli/wikicmd_test.go`
  Expected: 17 Zeilen `changed …` (geprobt). `internal/lock/replacedir_test.go` bleibt: dort ist `.brain.toml` nur ein Dateiname im kopierten Bestand. `areacheck_test.go`, `cases_3a_test.go`, `switchover_test.go`, `importcases/*_test.go`, `guard/decide_test.go`, `index/walk_test.go`, `detect_test.go`, `switchover/render_test.go` behalten ihre Altnamen absichtlich (sie prüfen Altnamen als Daten).

- [ ] **Step 9: Was das Skript nicht sieht, von Hand** (alle geprobt):
  - `internal/brain/index/reindex_test.go` `setupTestArea`: `os.MkdirAll(areaDir, 0o755)` → `os.MkdirAll(filepath.Join(areaDir, ".loomux"), 0o755)`.
  - `internal/brain/search/search_test.go` `writeArea`: `os.MkdirAll(dir, 0o750)` → `os.MkdirAll(filepath.Join(dir, ".loomux"), 0o750)`.
  - `internal/brain/apply/stock_test.go:50` und `internal/brain/index/staging_test.go:58`: in der Namensliste `".brain.toml"` → `filepath.Join(".loomux", "config.toml")`; `staging_test.go:364` ebenso.
  - `internal/brain/apply/resolve_test.go`: `TestResolveFindsAVaultUnderEveryNameOfTheDeclaration` ersetzen durch

    ```go
    // A vault declares itself under `.loomux/config.toml` alone: one that still
    // carries only an old name marks no vault, and the walk finds none.
    func TestResolveFindsNoVaultUnderAnOldName(t *testing.T) {
    	for _, name := range config.OldManifestNames() {
    		t.Run(name, func(t *testing.T) {
    			vault, casePath, areas := newVault(t)
    			if err := os.RemoveAll(filepath.Join(vault, ".loomux")); err != nil {
    				t.Fatal(err)
    			}
    			writeFile(t, filepath.Join(vault, name), vaultManifest(testReview))
    			_, err := resolve(casePath, caseOf("knowledge"), areas)
    			if err == nil || !strings.Contains(err.Error(), "no area declaration above this case") {
    				t.Fatalf("got %v; want no vault found", err)
    			}
    		})
    	}
    }
    ```

    und in `TestResolveWalksPastAPolicyOnlyConfig` den Untertest `"beside .brain.toml"` löschen (nach dem Skript schreibt er zweimal dieselbe Datei und prüft nichts mehr); im Kommentar darüber „and at the vault root it gives way to the `.brain.toml` beside it“ streichen.
  - `internal/brain/privacy/privacy_test.go` `TestVisibleManifest`: die drei Zeilen `"closed config.toml that cannot be read beside an open .brain.toml"`, `"closed .loomux/config.toml that cannot be read beside an open .brain.toml"`, `"the loomux name before an open .brain.toml"` durch **eine** ersetzen —

    ```go
    		{"a .loomux/config.toml that cannot be read", func(t *testing.T, dir string) {
    			path := filepath.Join(dir, ".loomux", "config.toml")
    			writeFile(t, path, closed)
    			testlock.Lock(t, path)
    		}, false, false, false, true},
    ```

    und im Kommentar über dem Test „The names are read in the order config.ReadAreaManifestUntilStage4 gives them: … which declares nothing.“ ersetzen durch „The declaration is `.loomux/config.toml` alone, readable or not; one without an [area] table declares nothing.“
  - `internal/brain/search/visibility_test.go` `closedWorld`: im `sealed`-Zweig die Zeile, die die offene Datei schreibt (`put(filepath.Join(dir, ".loomux", "config.toml"), "[area]\nscope = \"zz-sealed\"\n"+open)` nach dem Skript), löschen; der Kommentar oben wird „With sealed it adds "zz-sealed", whose `local_only` `.loomux/config.toml` cannot be read.“
  - `internal/brain/privacy/areas_test.go` `TestVisibleAreasStillRefusesAnEntryWithoutDeclarationThatIsNoWorkspace`: `errors.Is(err, config.ErrNoManifest)` → `errors.Is(err, config.ErrNoArea)`, Meldung „want config.ErrNoArea“.
  - `internal/cli/brain_test.go` `TestBrainReadersStillRefuseAnAreaWithoutDeclaration`: `strings.Contains(errOut, "no manifest found")` → `strings.Contains(errOut, "declares no [area]")`.

- [ ] **Step 10: Die CLI-Sicht des Hinweises testen** — an `internal/cli/brain_test.go` anhängen:

  ```go
  // A registered area that still carries only an old manifest is refused with
  // the new reader's error, and the error says where the old one lies and which
  // command shows what to carry over.
  func TestBrainNamesAnOldManifestBesideTheMissingOne(t *testing.T) {
  	w := brainWorld(t, "", map[string]string{"index.md": "# project/a\n"})
  	if err := os.Remove(filepath.Join(w.area, ".loomux", "config.toml")); err != nil {
  		t.Fatal(err)
  	}
  	writeFile(t, filepath.Join(w.area, ".brain.toml"), "[area]\nscope = \"project/a\"\n")
  	code, out, errOut := run("brain", "catalog", "--scope", "all")
  	want := "an old manifest lies there (.brain.toml): `loomux area check " + filepath.ToSlash(w.area) + "` shows what to carry over"
  	if code != 1 || out != "" || !strings.Contains(errOut, want) {
  		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
  	}
  }
  ```

- [ ] **Step 11: Alle Tests.**

  Run: `go test ./... -count=1 > <scratch>/t2.log 2>&1; grep -E '^(--- FAIL|FAIL)' <scratch>/t2.log`
  Expected: keine Zeile. Die Fallsuiten sind darin (geprobt grün).

- [ ] **Step 12: Kommentare, die den alten Leser nennen, nachziehen** (`git grep -n "ReadAreaManifestUntilStage4\|until stage 4\|ultra-brain's two names\|\.brain\.toml" -- 'internal/**/*.go' ':!*_test.go'`): `apply/resolve.go:79-85` (nur noch `.loomux/config.toml` markiert einen Vault), `check/run/run.go:328-340` (der Absatz „That reader and not `config.ReadManifest`, because the areas on this machine still declare themselves as `.brain.toml` …“ wird ein Satz: der Leser ist `config.ReadAreaDeclaration`, strenger als `ReadManifest`, und beides kommt als `manifest-unreadable` an), `run.go:378`, `privacy/channel.go:45-46`, `config/manifest.go:43` („loomux cut the two spellings …“ bleibt, nur der Verweis auf `ReadAreaManifestUntilStage4` entfällt, falls vorhanden). `index/walk.go:26` (`**/.ultra-brain/**` als Ausschluss) bleibt: das Verzeichnis kann noch in Repos liegen.

- [ ] **Step 13: Doku dieses Tasks.** `docs/en/getting-started.md` im Punkt „Every `loomux brain` command fails with `no manifest found`“ den Halbsatz „the commands also accept the legacy manifests `.ultra-brain/config.toml` and `.brain.toml` until the clean-up pull request of stage 4e removes them“ ersetzen durch „an area that still carries only `.ultra-brain/config.toml` or `.brain.toml` is refused, and the message names the old file and `loomux area check <path>`, which shows what to carry over into `.loomux/config.toml`“; `docs/de/getting-started.md` gleichlautend deutsch. `docs/wiki/topics/datenmodell-und-bereiche.md`: den Satz „loomux liest `.loomux/config.toml` vor den beiden Altnamen … sonst gilt weiter das Altmanifest —“ ersetzen durch „loomux liest seit dem Aufräum-PR nur `.loomux/config.toml`; eine Datei ohne `[area]` deklariert keinen Bereich, ein danebenliegendes Altmanifest nennt die Fehlermeldung samt `loomux area check`.“ (nicht löschen, `status` der Seite unverändert).

- [ ] **Step 14: Mutationsrunde (Hand, Overlay).** Je Aufrufer der Mutant „`config.IsUndeclared(err)` → `errors.Is(err, config.ErrNoManifest)`“ (die alte Regel) gegen die Tests des Pakets; `chosenManifest` mit den Mutanten „`&& name == manifestNames[0]` gestrichen“, „`continue` → `return \"\"`“, „`return name` → `return \"\"`“ gegen `-run 'TestAreaCheck|TestClassifyKeys|TestFlattenKeys|TestLegacyHints'`; `VisibleAreas`: „`IsUndeclared` → `errors.Is(err, config.ErrNoArea)`“ muss `TestVisibleAreasLeavesOutAWorkspaceWithoutAnyConfigFile` töten. Ein Aufrufer, dessen Mutant überlebt, bekommt einen Test mit einer Policy-Datei ohne `[area]` an seiner Stelle, oder die Begründung in „Überlebende Mutanten“.

- [ ] **Step 15: Commit** (`<scratch>/msg2.txt`):

  ```
  refactor(brain)!: read area declarations under .loomux/config.toml alone

  Every brain reader takes the declaration from config.ReadAreaDeclaration.
  .ultra-brain/config.toml and .brain.toml are no longer read; an area that
  carries only one of them is refused with a message that names it and
  `loomux area check`. A .loomux/config.toml without [area] no longer hands
  the decision to an old name. The readers that accept an undeclared area
  accept both answers, and a workspace without [area] is still left out.
  `area check` keeps its three names and picks the chosen file itself.

  BREAKING CHANGE: .ultra-brain/config.toml and .brain.toml no longer declare
  an area; carry their content into .loomux/config.toml.
  ```

  Run: `git add -A internal docs/en/getting-started.md docs/de/getting-started.md docs/wiki/topics/datenmodell-und-bereiche.md && git commit -F <scratch>/msg2.txt > <scratch>/commit2.log 2>&1; echo $?`

---

### Task 3: Kein zweites Zustandsverzeichnis mehr

**Files:**
- Delete: `internal/config/legacy.go`, `internal/config/legacy_test.go`
- Rewrite: `internal/config/artifacts.go`, `internal/config/artifacts_test.go`, `internal/brain/apply/stock.go`, `internal/brain/apply/stock_test.go`
- Modify: Produktdateien der Tabelle unten; `internal/brain/apply/resolve.go`, `approve.go`; `internal/config/manifest_test.go` (zwei Tests ziehen ein)
- Test: 42 Testdateien (Liste Step 9)
- Docs: `docs/en|de/cli-reference.md`, `docs/en|de/configuration.md`, `testdata/cases/README.md`, `docs/wiki/topics/datenmodell-und-bereiche.md`

**Interfaces:**
- Consumes: `config.ManifestDir(area Area, stateDir string) string` (unverändert).
- Produces: `type ArtifactLookup struct{ Primary string }` mit `Resolve`, `WritePath`, `NewArtifactLookup()`; `apply.recoverStock(area config.Area, lookup config.ArtifactLookup) error`; `apply.registerOf(area config.Area, lookup config.ArtifactLookup) string`; die Signaturen der Tabelle in Step 6. Entfallen: `config.LegacyBrainDirUntilStage3`, `config.ResolvedAreaDir`, `ArtifactLookup.Fallback`, `serve.Options.LegacyDir`, `servebrain.Deps.LegacyDir`, `servegraph.Deps.LegacyDir`, `benchsearch.Deps.FallbackDir`, `apply.moveStock`, `copyStock`, `stagingDir`, `swapDir`, `readStock`, `beforeSwap`, `registerRead`, `registerWrite`.

- [ ] **Step 1: Die roten Tests schreiben.** `internal/config/artifacts_test.go` ganz ersetzen:

  ```go
  package config_test

  import (
  	"os"
  	"path/filepath"
  	"testing"

  	"github.com/xidus90/loomux/internal/config"
  )

  // A state file is read and written in the one state directory, whether it
  // lies there or not; a copy elsewhere is never the answer.
  func TestArtifactLookupAnswersTheStateDirectoryAlone(t *testing.T) {
  	primary := t.TempDir()
  	lookup := config.ArtifactLookup{Primary: primary}
  	relative := filepath.Join("maintenance", "last-run.txt")
  	want := filepath.Join(primary, relative)
  	if got := lookup.Resolve(relative); got != want {
  		t.Fatalf("Resolve = %q, want %q", got, want)
  	}
  	if got := lookup.WritePath(relative); got != want {
  		t.Fatalf("WritePath = %q, want %q", got, want)
  	}
  }

  // LOOMUX_LEGACY_BRAIN_DIR names nothing any more: the lookup takes the state
  // directory and asks no second place, even where only that one holds the file.
  func TestNewArtifactLookupTakesTheStateDirectoryAlone(t *testing.T) {
  	primary, old := t.TempDir(), t.TempDir()
  	t.Setenv(config.StateDirEnv, primary)
  	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", old)
  	relative := filepath.Join("maintenance", "last-run.txt")
  	if err := os.MkdirAll(filepath.Join(old, "maintenance"), 0o755); err != nil {
  		t.Fatal(err)
  	}
  	if err := os.WriteFile(filepath.Join(old, relative), []byte("2999-01-01T00:00:00+00:00\n"), 0o644); err != nil {
  		t.Fatal(err)
  	}
  	lookup := config.NewArtifactLookup()
  	if lookup.Primary != primary {
  		t.Fatalf("NewArtifactLookup = %+v, want the state directory %q", lookup, primary)
  	}
  	if got, want := lookup.Resolve(relative), filepath.Join(primary, relative); got != want {
  		t.Fatalf("Resolve = %q, want %q", got, want)
  	}
  }
  ```

  Und an `internal/cli/brain_test.go` (hinter `TestBrainReadsAReadOnlyAreaFromTheLegacyDirectory`):

  ```go
  // ultra-brain's state directory is read no more, whatever
  // LOOMUX_LEGACY_BRAIN_DIR names: not a read-only area's stock, which is then
  // missing, and not the reconcile stamp, which is then never written.
  func TestBrainReadsNothingFromTheOldStateDirectory(t *testing.T) {
  	state, old, area := t.TempDir(), t.TempDir(), t.TempDir()
  	t.Setenv("LOOMUX_STATE_DIR", state)
  	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", old)
  	writeFile(t, filepath.Join(state, "registry.toml"),
  		"[[area]]\nscope = \"project/r\"\npath = "+strconv.Quote(filepath.ToSlash(area))+"\nreadonly = true\n")
  	stock := filepath.Join(old, "areas", "project-r")
  	writeFile(t, filepath.Join(stock, ".loomux", "config.toml"), "[area]\nscope = \"project/r\"\n")
  	writeFile(t, filepath.Join(stock, "index.md"), "# project/r\n")
  	writeFile(t, filepath.Join(old, "maintenance", "last-run.txt"), "2999-01-01T00:00:00+00:00\n")

  	code, out, errOut := run("brain", "catalog", "--scope", "project/r")
  	missing := filepath.Join(state, "areas", "project-r") + ": no manifest found"
  	if code != 1 || out != "" || !strings.Contains(errOut, missing) {
  		t.Fatalf("catalog: code %d\nout %q\nerr %q", code, out, errOut)
  	}

  	writeFile(t, filepath.Join(state, "registry.toml"), "")
  	stubBrainStatusPort(t, search.NewFakePort())
  	code, out, errOut = run("brain", "status")
  	if code != 0 || !strings.HasPrefix(out, "last reconcile: never;") {
  		t.Fatalf("status: code %d\nout %q\nerr %q", code, out, errOut)
  	}
  }
  ```

- [ ] **Step 2: Rot.**

  Run: `go test ./internal/cli/ -run TestBrainReadsNothingFromTheOldStateDirectory -count=1 > <scratch>/t3-red.log 2>&1; go test ./internal/config/ -run 'ArtifactLookup' -count=1 >> <scratch>/t3-red.log 2>&1`
  Expected (geprobt für die CLI-Zeile): `--- FAIL: TestBrainReadsNothingFromTheOldStateDirectory … catalog: code 0 out "# project/r\n"`; `--- FAIL: TestNewArtifactLookupTakesTheStateDirectoryAlone` (der Rückfall liefert die Datei aus `old`). `TestArtifactLookupAnswersTheStateDirectoryAlone` ist an HEAD schon grün (kein Fallback gesetzt) und bleibt der Wächter der neuen Hülle.

- [ ] **Step 3: `artifacts.go` ersetzen, `legacy.go` löschen.**

  ```go
  package config

  import "path/filepath"

  // ArtifactLookup names the state files of the maintenance layer: the stamp,
  // the merge events, the record of qmd collections. They live in one state
  // directory, the one this run was handed; a lookup never reads from anywhere
  // else, so that `internal/serve`'s promise holds -- everything hangs off the
  // state directory it was given.
  type ArtifactLookup struct {
  	Primary string
  }

  // NewArtifactLookup takes the state directory this machine declares.
  func NewArtifactLookup() ArtifactLookup {
  	return ArtifactLookup{Primary: StateDir()}
  }

  // Resolve is the place relative is to be read from.
  func (l ArtifactLookup) Resolve(relative string) string {
  	return filepath.Join(l.Primary, relative)
  }

  // WritePath is the place relative is to be written to.
  func (l ArtifactLookup) WritePath(relative string) string {
  	return filepath.Join(l.Primary, relative)
  }
  ```

  Run: `git rm -q internal/config/legacy.go internal/config/legacy_test.go`. Die beiden Tests `TestReadManifestStillKnowsOnlyTheLoomuxName` und `TestReadManifestStillAcceptsAManifestWithoutAScope` (Paket `config`) vorher nach `internal/config/manifest_test.go` übernehmen, `legacyWrite(…)` dabei durch `write(t, filepath.Join(dir, name), content)` mit `os.MkdirAll` davor oder durch `declareIn` aus Task 1 ersetzen.

- [ ] **Step 4: `ResolvedAreaDir` → `ManifestDir` an allen 18 Stellen** (geprobt, ein Lauf):

  ```bash
  F=$(git grep -l "ResolvedAreaDir(" -- '*.go' ':!*_test.go' ':!internal/config/artifacts.go')
  sed -i -E 's/config\.ResolvedAreaDir\(([a-zA-Z.]+), lookup\.Primary, lookup\.Fallback\)/config.ManifestDir(\1, lookup.Primary)/g; s/config\.ResolvedAreaDir\(([a-zA-Z.]+), (stateDir|registryDir), fallbackDir\)/config.ManifestDir(\1, \2)/g' $F
  git grep -n "ResolvedAreaDir(" -- '*.go' ':!*_test.go'
  ```

  Expected: keine Zeile mehr.

- [ ] **Step 5: `stock.go` ersetzen, `registerOf` zusammenführen.** `internal/brain/apply/stock.go` ganz:

  ```go
  package apply

  import (
  	"github.com/xidus90/loomux/internal/config"
  	"github.com/xidus90/loomux/internal/lock"
  )

  // recoverDir is lock.Recover, as a variable so that a test can make it fail.
  var recoverDir = lock.Recover

  // recoverStock finishes a swap a killed `index` run left half-done in a
  // read-only area's stock, before the first write into it.
  //
  // `index` publishes that stock with lock.ReplaceDir, which puts the old one
  // aside for an instant. Killed in that instant, it leaves the aside and no
  // target. A register written now would create the target with that one file,
  // and the next Recover would find a target and delete the aside -- the
  // area's catalog, graph and declaration with it. So the aside goes back
  // first, as `index` does before it reads.
  //
  // A writable area keeps its stock in its own tree, which no swap touches.
  // Outside the barrier by design: it renames below the state directory, never
  // into the vault or a wiki, which is what `place.gate` measures.
  func recoverStock(area config.Area, lookup config.ArtifactLookup) error {
  	if !area.ReadOnly {
  		return nil
  	}
  	return recoverDir(config.ManifestDir(area, lookup.Primary))
  }
  ```

  In `approve.go` `moveStock(area, a.o.Lookup)` → `recoverStock(area, a.o.Lookup)`. In `resolve.go` `registerRead` und `registerWrite` durch eine Funktion ersetzen und beide Namen per `sed -i 's/registerRead(/registerOf(/g; s/registerWrite(/registerOf(/g' internal/brain/apply/*.go` umbenennen:

  ```go
  // registerOf is `area_artifact_dir(area, state_dir) / _REGISTER`: a writable
  // area's register lies in its tree, a read-only one's in the state
  // directory, where `index` publishes that area's stock too. It is read and
  // written at the same place.
  func registerOf(area config.Area, lookup config.ArtifactLookup) string {
  	return filepath.Join(config.ManifestDir(area, lookup.Primary), registerName)
  }
  ```

  `sourceFile.readFrom` und `.register` bleiben beide (gleicher Wert, kleinerer Diff).

- [ ] **Step 6: Die durchgereichten Parameter entfernen.** Die folgende Tabelle ist die vollständige, kompilierte Liste (alt → neu, je Datei, als Ersetzung im Text). Am bequemsten per Skript `<scratch>/drop_params.py` mit genau diesen Paaren (`s.replace(old, new)`, je Paar `MISSING` melden, wenn es nicht trifft):

  | Datei | alt | neu |
  |---|---|---|
  | `internal/brain/search/stamp.go` | `func ReadLastRun(stateDir, fallbackDir string)` | `func ReadLastRun(stateDir string)` |
  | | `config.ArtifactLookup{Primary: stateDir, Fallback: fallbackDir}` | `config.ArtifactLookup{Primary: stateDir}` |
  | | `func StaleReconcile(stateDir, fallbackDir string, now time.Time)` | `func StaleReconcile(stateDir string, now time.Time)` |
  | | `ReadLastRun(stateDir, fallbackDir)` | `ReadLastRun(stateDir)` |
  | `internal/brain/search/search.go` | `port SearchPort, registryDir, fallbackDir string, now time.Time)` | `port SearchPort, registryDir string, now time.Time)` |
  | | `privacy.VisibleAreas(registryDir, fallbackDir, scope, channel)` | `privacy.VisibleAreas(registryDir, scope, channel)` |
  | | `assemble(hits, areas, registryDir, fallbackDir, n)` | `assemble(hits, areas, registryDir, n)` |
  | | `StaleReconcile(registryDir, fallbackDir, now)` | `StaleReconcile(registryDir, now)` |
  | | `areas []privacy.VisibleArea, stateDir, fallbackDir string, n int)` | `areas []privacy.VisibleArea, stateDir string, n int)` |
  | `internal/brain/privacy/areas.go` | `func VisibleAreas(registryDir, fallbackDir, scope string, ch Channel)` | `func VisibleAreas(registryDir, scope string, ch Channel)` |
  | `internal/brain/status/status.go` | `port search.SearchPort, registryDir, fallbackDir string, now time.Time)` | `port search.SearchPort, registryDir string, now time.Time)` |
  | | `lastReconcile(registryDir, fallbackDir, now)` | `lastReconcile(registryDir, now)` |
  | | `privacy.VisibleAreas(registryDir, fallbackDir, "all", ch)` | `privacy.VisibleAreas(registryDir, "all", ch)` |
  | | `graph.ReadGraph(area, registryDir, fallbackDir)` | `graph.ReadGraph(area, registryDir)` |
  | | `unfindable(visible, port, registryDir, fallbackDir)` | `unfindable(visible, port, registryDir)` |
  | | `sharedHashes(areas, registryDir, fallbackDir)` | `sharedHashes(areas, registryDir)` |
  | | `func lastReconcile(stateDir, fallbackDir string, now time.Time)` | `func lastReconcile(stateDir string, now time.Time)` |
  | | `search.ReadLastRun(stateDir, fallbackDir)` | `search.ReadLastRun(stateDir)` |
  | | `func registerPath(area config.Area, stateDir, fallbackDir string)` | `func registerPath(area config.Area, stateDir string)` |
  | | `port search.SearchPort, stateDir, fallbackDir string)` | `port search.SearchPort, stateDir string)` |
  | | `registerPath(area, stateDir, fallbackDir)` | `registerPath(area, stateDir)` |
  | | `areas []privacy.VisibleArea, stateDir, fallbackDir string)` | `areas []privacy.VisibleArea, stateDir string)` |
  | | `registerPath(visible.Area, stateDir, fallbackDir)` | `registerPath(visible.Area, stateDir)` |
  | `internal/brain/graph/read.go` | `func ReadGraph(area config.Area, stateDir, fallbackDir string)` | `func ReadGraph(area config.Area, stateDir string)` |
  | `internal/brain/catalog/area.go` | `func ReadAreaCatalog(area config.Area, stateDir, fallbackDir string)` | `func ReadAreaCatalog(area config.Area, stateDir string)` |
  | `internal/brain/convert/run.go` | `func Areas(areas []config.Area, stateDir, fallbackDir string)` | `func Areas(areas []config.Area, stateDir string)` |
  | `internal/brain/index/reindex.go` | `func Reindex(registryPath, stateDir, fallbackDir string, port search.SearchPort)` | `func Reindex(registryPath, stateDir string, port search.SearchPort)` |
  | | `ReindexWithOutput(registryPath, stateDir, fallbackDir, port, os.Stderr)` | `ReindexWithOutput(registryPath, stateDir, port, os.Stderr)` |
  | | `func ReindexWithOutput(registryPath, stateDir, fallbackDir string, port search.SearchPort, stderr io.Writer)` | `func ReindexWithOutput(registryPath, stateDir string, port search.SearchPort, stderr io.Writer)` |
  | | `indexArea(area, areas, stateDir, fallbackDir, stderr)` | `indexArea(area, areas, stateDir, stderr)` |
  | | `config.ArtifactLookup{Primary: stateDir, Fallback: fallbackDir}` | `config.ArtifactLookup{Primary: stateDir}` |
  | | `\tstateDir, fallbackDir string,\n` | `\tstateDir string,\n` |
  | `internal/brain/maintenance/events.go` | `func EventsPath(stateDir, fallbackDir string)` | `func EventsPath(stateDir string)` |
  | | `config.ArtifactLookup{Primary: stateDir, Fallback: fallbackDir}` | `config.ArtifactLookup{Primary: stateDir}` |
  | | `func ReadEvents(stateDir, fallbackDir string)` | `func ReadEvents(stateDir string)` |
  | | `dropped(stateDir, fallbackDir)` | `dropped(stateDir)` |
  | | `EventsPath(stateDir, fallbackDir)` | `EventsPath(stateDir)` |
  | | `func dropped(stateDir, fallbackDir string)` | `func dropped(stateDir string)` |
  | `internal/brain/maintenance/scan.go` | `\tstateDir, fallbackDir string,\n` | `\tstateDir string,\n` |
  | `internal/brain/maintenance/reconcile.go` | `Scan(area, manifest, lookup.Primary, lookup.Fallback)` | `Scan(area, manifest, lookup.Primary)` |
  | | `ReadEvents(lookup.Primary, lookup.Fallback)` | `ReadEvents(lookup.Primary)` |
  | `internal/brain/answer/answer.go` | `func Run(req Request, registryDir, fallbackDir string, notice func(string))` | `func Run(req Request, registryDir string, notice func(string))` |
  | | `RunWith(DefaultPorts(), req, registryDir, fallbackDir, notice)` | `RunWith(DefaultPorts(), req, registryDir, notice)` |
  | | `func(Request, string, string, func(string)) (string, []string, error)` | `func(Request, string, func(string)) (string, []string, error)` |
  | | `return func(req Request, registryDir, fallbackDir string, notice func(string))` | `return func(req Request, registryDir string, notice func(string))` |
  | | `RunWith(DefaultPortsWith(stateDir, opts...), req, registryDir, fallbackDir, notice)` | `RunWith(DefaultPortsWith(stateDir, opts...), req, registryDir, notice)` |
  | | `func RunWith(ports Ports, req Request, registryDir, fallbackDir string, notice func(string))` | `func RunWith(ports Ports, req Request, registryDir string, notice func(string))` |
  | | `search(ports, req, registryDir, fallbackDir, notice)` | `search(ports, req, registryDir, notice)` |
  | | `catalog(req.Scope, req.Channel, registryDir, fallbackDir)` | `catalog(req.Scope, req.Channel, registryDir)` |
  | | `read(req.Query, req.Scope, req.Section, req.Channel, registryDir, fallbackDir)` | `read(req.Query, req.Scope, req.Section, req.Channel, registryDir)` |
  | | `neighbors(req.Query, req.Scope, req.Channel, registryDir, fallbackDir)` | `neighbors(req.Query, req.Scope, req.Channel, registryDir)` |
  | | `status(ports, req.Channel, registryDir, fallbackDir)` | `status(ports, req.Channel, registryDir)` |
  | | `func search(ports Ports, req Request, registryDir, fallbackDir string, notice func(string))` | `func search(ports Ports, req Request, registryDir string, notice func(string))` |
  | | `port, registryDir, fallbackDir, ports.Now())` | `port, registryDir, ports.Now())` |
  | | `func catalog(scope string, channel privacy.Channel, registryDir, fallbackDir string)` | `func catalog(scope string, channel privacy.Channel, registryDir string)` |
  | | `privacy.VisibleAreas(registryDir, fallbackDir, "all", channel)` | `privacy.VisibleAreas(registryDir, "all", channel)` |
  | | `braincatalog.ReadAreaCatalog(area.Area, registryDir, fallbackDir)` | `braincatalog.ReadAreaCatalog(area.Area, registryDir)` |
  | | `func area(scope string, channel privacy.Channel, registryDir, fallbackDir string)` | `func area(scope string, channel privacy.Channel, registryDir string)` |
  | | `func read(relative, scope, section string, channel privacy.Channel, registryDir, fallbackDir string)` | `func read(relative, scope, section string, channel privacy.Channel, registryDir string)` |
  | | `area(scope, channel, registryDir, fallbackDir)` | `area(scope, channel, registryDir)` |
  | | `func neighbors(relative, scope string, channel privacy.Channel, registryDir, fallbackDir string)` | `func neighbors(relative, scope string, channel privacy.Channel, registryDir string)` |
  | | `graph.ReadGraph(found.Area, registryDir, fallbackDir)` | `graph.ReadGraph(found.Area, registryDir)` |
  | | `func status(ports Ports, channel privacy.Channel, registryDir, fallbackDir string)` | `func status(ports Ports, channel privacy.Channel, registryDir string)` |
  | | `brainstatus.Lines(channel, ports.Status(), registryDir, fallbackDir, ports.Now())` | `brainstatus.Lines(channel, ports.Status(), registryDir, ports.Now())` |
  | `internal/dev/benchsearch/run.go` | `StateDir, FallbackDir string` (Feld in `Deps`) | `StateDir string` |
  | | `stateDir: d.StateDir, fallback: d.FallbackDir, now: d.Now}` | `stateDir: d.StateDir, now: d.Now}` |
  | | `c.scope, c.stateDir, c.fallback = p.Scope, p.StateDir, ""` | `c.scope, c.stateDir = p.Scope, p.StateDir` |
  | | `\tstateDir, fallback string\n` (Feld in `chain`) | `\tstateDir string\n` |
  | | `c.port, c.stateDir, c.fallback, c.now())` | `c.port, c.stateDir, c.now())` |
  | | `answer.RunWith(answer.Ports{}, req, c.stateDir, c.fallback, nil)` | `answer.RunWith(answer.Ports{}, req, c.stateDir, nil)` |
  | `internal/serve/serve.go` | `\tLegacyDir   string\n` (in `Options`) | — |
  | | `NewUpkeep(opts.RegistryDir, opts.LegacyDir)` | `NewUpkeep(opts.RegistryDir)` |
  | | `\t\tLegacyDir:   opts.LegacyDir,\n` (zweimal) | — |
  | | `func(answer.Request, string, string, func(string)) (string, []string, error)` | `func(answer.Request, string, func(string)) (string, []string, error)` |
  | `internal/serve/upkeep.go` | `func NewUpkeep(registryDir, legacyDir string) *Upkeep {` | `func NewUpkeep(registryDir string) *Upkeep {` |
  | | `config.ArtifactLookup{Primary: registryDir, Fallback: legacyDir}` | `config.ArtifactLookup{Primary: registryDir}` |
  | | `u.lookup.Primary, u.lookup.Fallback` (dreimal) | `u.lookup.Primary` |
  | `internal/serve/brain/tools.go` | `\tLegacyDir   string\n` | — |
  | | `deps.Answer(request, deps.RegistryDir, deps.LegacyDir, notice)` | `deps.Answer(request, deps.RegistryDir, notice)` |
  | | `func(answer.Request, string, string, func(string)) (string, []string, error)` | `func(answer.Request, string, func(string)) (string, []string, error)` |
  | `internal/serve/graph/tools.go` | `\tLegacyDir   string\n` | — |
  | | `privacy.VisibleAreas(deps.RegistryDir, deps.LegacyDir, "all", channel)` | `privacy.VisibleAreas(deps.RegistryDir, "all", channel)` |
  | `internal/cli/brain.go` | die drei Kommentarzeilen „The registry is loomux's; artefacts …“ und `registryDir, fallbackDir := config.StateDir(), config.LegacyBrainDirUntilStage3()` | `registryDir := config.StateDir()` |
  | | `answer.RunWith(brainPorts(), req, registryDir, fallbackDir, func(message string) {` | `answer.RunWith(brainPorts(), req, registryDir, func(message string) {` |
  | `internal/cli/dev.go` | `StateDir:    stateDir,` + `FallbackDir: config.LegacyBrainDirUntilStage3(),` | `StateDir: stateDir,` |
  | `internal/cli/serve.go` | `LegacyDir:   config.LegacyBrainDirUntilStage3(),` | — |
  | `internal/cli/convert.go` | `convert.Areas(areas, lookup.Primary, lookup.Fallback)` (zweimal) | `convert.Areas(areas, lookup.Primary)` |
  | `internal/cli/index.go` | `index.ReindexWithOutput(path, lookup.Primary, lookup.Fallback, brainPorts().Status(), stderr)` | `index.ReindexWithOutput(path, lookup.Primary, brainPorts().Status(), stderr)` |

  Danach `gofmt -w internal/` (Feldausrichtung).

  Run: `go build ./... > <scratch>/b3.log 2>&1; echo $?`
  Expected: 0 (geprobt).

- [ ] **Step 7: Kommentare, die den Rückfall beschreiben** (`git grep -n "fallbackDir\|fallback\b\|legacy directory\|ultra-brain's state\|loomux migrate\|until stage 3\|LOOMUX_LEGACY_BRAIN_DIR" -- 'internal/**/*.go' ':!*_test.go'`): `catalog/area.go:11-13`, `graph/read.go:26`, `index/reindex.go:91-99` (der ganze Absatz „Both directories are arguments …“ wird „stateDir is the one place anything is read from and written to; it is an argument for the reason `internal/serve` gives …“), `index/reindex.go:160-162` („The declaration is read where the stock lies today …“), `index/staging.go:27,49`, `maintenance/events.go:61-68,127-130`, `maintenance/reconcile.go:108-112,264-266`, `maintenance/scan.go:69`, `privacy/areas.go:31-36`, `privacy/channel.go:47-51`, `search/search.go:40`, `search/stamp.go:16-27` (nur die Rückfallsätze; der Ratschlag selbst ist Task 6), `status/status.go:25-29`, `lock/replacedir.go:33-52` (Zeile 35 nennt `config.ResolvedAreaDir` → `config.ManifestDir`; der Absatz „It is tolerable only because config.ResolvedAreaDir still falls back …“ wird: „An absent target is a missing declaration for a read-only area, and every brain reader refuses it; Recover, which `index` calls before it reads and `approve` before it writes, closes that window.“), `cli/maintenance.go:26`, `config/registry.go:36-38` (dort zudem `project/space` und `project/iam-wiki` richtig beschreiben oder das Beispiel streichen), `apply/stock.go` ist neu. Kein Kommentar nennt danach ein Ende „bis Stufe …“.

- [ ] **Step 8: Die Tests von `apply`.** `internal/brain/apply/stock_test.go` ganz ersetzen:

  ```go
  package apply

  import (
  	"errors"
  	"path/filepath"
  	"testing"

  	"github.com/xidus90/loomux/internal/config"
  	"github.com/xidus90/loomux/internal/lock"
  )

  // A swap a killed `index` left half-done is finished: the aside goes back
  // whole, and nothing of it is lost.
  func TestRecoverStockPutsAnAsideBack(t *testing.T) {
  	lookup := config.ArtifactLookup{Primary: t.TempDir()}
  	ro := config.Area{Scope: "project/ro", Path: t.TempDir(), ReadOnly: true}
  	target := config.ManifestDir(ro, lookup.Primary)
  	writeFile(t, filepath.Join(target+lock.AsideSuffix, "catalog.tsv"), "stock\n")
  	if err := recoverStock(ro, lookup); err != nil {
  		t.Fatal(err)
  	}
  	if readFile(t, filepath.Join(target, "catalog.tsv")) != "stock\n" {
  		t.Fatal("the aside did not go back")
  	}
  	absent(t, target+lock.AsideSuffix)
  }

  // A writable area is never swapped: a directory beside its tree that is
  // named like an aside is someone else's, and stays.
  func TestRecoverStockLeavesAWritableAreaAlone(t *testing.T) {
  	lookup := config.ArtifactLookup{Primary: t.TempDir()}
  	area := config.Area{Scope: "w", Path: t.TempDir()}
  	aside := area.Path + lock.AsideSuffix
  	writeFile(t, filepath.Join(aside, "kept.md"), "x")
  	if err := recoverStock(area, lookup); err != nil {
  		t.Fatal(err)
  	}
  	if !isFile(filepath.Join(aside, "kept.md")) {
  		t.Fatal("a writable area's neighbour was touched")
  	}
  }

  func TestRecoverStockPassesOnTheFailure(t *testing.T) {
  	boom := errors.New("boom")
  	seam(t, &recoverDir, func(string) error { return boom })
  	ro := config.Area{Scope: "project/ro", Path: t.TempDir(), ReadOnly: true}
  	if err := recoverStock(ro, config.ArtifactLookup{Primary: t.TempDir()}); !errors.Is(err, boom) {
  		t.Fatalf("want the failure passed on, got %v", err)
  	}
  }
  ```

  In `approve_test.go` `addReadOnlyArea` und `TestAReadOnlyAreasRegisterIsWrittenToTheNewPlace` ersetzen durch:

  ```go
  // addReadOnlyArea registers a read-only area whose stock lies in the state
  // directory -- a register and a catalog beside it -- and adds its one source
  // to the case. registered is the hash the register holds; it returns the
  // register and the source's hash.
  func (v *appVault) addReadOnlyArea(t *testing.T, registered string) (register, digest string) {
  	t.Helper()
  	code := filepath.Join(v.base, "ro")
  	source := filepath.Join(code, "doc.md")
  	writeFile(t, source, "Text\n")
  	digest = hashOf(t, source)
  	ro := config.Area{Scope: "project/ro", Path: code, ReadOnly: true}
  	stock := config.ManifestDir(ro, v.lookup.Primary)
  	register = writeRegister(t, stock, identity.Identity{DocID: "01RO", Relative: "doc.md", ContentHash: registered, Revision: 4})
  	writeFile(t, filepath.Join(stock, "catalog.tsv"), "stock\n")
  	v.areas = append(v.areas, ro)
  	v.editCase(t, func(c *maintenance.Case) {
  		c.Sources = append(c.Sources, maintenance.SourceState{DocID: "01RO", Revision: 4, ContentHash: digest})
  	})
  	return register, digest
  }

  // A read-only area's register is advanced where it lies, in the state
  // directory, and the rest of its stock stays beside it.
  func TestAReadOnlyAreasRegisterIsAdvancedInTheStateDirectory(t *testing.T) {
  	v := newAppVault(t)
  	register, digest := v.addReadOnlyArea(t, "sha256:old")
  	v.mustApprove(t)
  	identities, err := identity.ReadIdentities(register)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got := identities["doc.md"]; got.Revision != 5 || got.ContentHash != digest {
  		t.Fatalf("register row = %+v", got)
  	}
  	if readFile(t, filepath.Join(filepath.Dir(register), "catalog.tsv")) != "stock\n" {
  		t.Fatal("the stock beside the register changed")
  	}
  }

  // An `index` killed mid-swap after this approval read the register leaves
  // the stock aside and no target. The approval puts it back before it
  // advances the register: the register is advanced inside the whole stock,
  // and the aside is not left for a later Recover to delete.
  func TestApprovePutsAnAsideStockBackBeforeItAdvancesTheRegister(t *testing.T) {
  	v := newAppVault(t)
  	register, digest := v.addReadOnlyArea(t, "sha256:old")
  	stock := filepath.Dir(register)
  	write := replaceText
  	seam(t, &replaceText, func(path, text string) error {
  		if filepath.Base(path) == filepath.Base(appTarget) {
  			if err := os.Rename(stock, stock+lock.AsideSuffix); err != nil {
  				t.Fatal(err)
  			}
  		}
  		return write(path, text)
  	})
  	v.mustApprove(t)
  	identities, err := identity.ReadIdentities(register)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got := identities["doc.md"]; got.Revision != 5 || got.ContentHash != digest {
  		t.Fatalf("register row = %+v", got)
  	}
  	if readFile(t, filepath.Join(stock, "catalog.tsv")) != "stock\n" {
  		t.Fatal("the stock that lay aside is lost")
  	}
  	absent(t, stock+lock.AsideSuffix)
  }
  ```

  In `TestAFailureMidWayReportsWhatWasWritten` die Zeile `{"move stock", …}` umbauen: Name `"recover stock"`, Naht `seam(t, &recoverDir, func(string) error { return broken })`, Erwartung `[]string{page, registerName}` unverändert. In `place_test.go` (`TestNoWritePrimitiveIsCalledOutsideTheBarrier`) den Block ab „The third tier writes below the state directory …“ bis vor `offenders :=` ersetzen durch:

  ```go
  	// The third tier renames below the state directory and never into the
  	// vault or a wiki, which is all the gate measures: recoverStock puts a
  	// read-only area's stock back from aside before a register is written
  	// there, and derives its target itself.
  	for _, call := range []string{"recoverDir", "lock.Recover"} {
  		primitives[call] = map[string]bool{"recoverStock": true}
  	}
  ```

  In `resolve_test.go`: `config.ResolvedAreaDir(ro, lookup.Primary, lookup.Fallback)` → `config.ManifestDir(ro, lookup.Primary)`; `TestResolveSourcesWritesAFallbackRegisterToTheNewPlace` samt Kommentar löschen.

  Run: `go vet ./internal/brain/apply/ && go test ./internal/brain/apply/ -count=1 -coverprofile=<scratch>/ap.out > <scratch>/ap.log 2>&1`
  Expected: `ok`, 100.0 % (geprobt).

- [ ] **Step 9: Die übrigen Tests, vom Übersetzer geführt.** Regeln, in dieser Reihenfolge:
  - **R1** Jede Zeile `t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", …)` in `internal/cli/*_test.go` löschen, außer in `TestBrainReadsNothingFromTheOldStateDirectory`; in `config_mutants_test.go:19` und `config_proposals_mutants_test.go:18` den Namen aus der Liste nehmen. Skript `<scratch>/cli_tests.py` (geprobt) mit `re.sub(r'(?m)^[ \t]*t\.Setenv\("LOOMUX_LEGACY_BRAIN_DIR", [^\n]*\)\n', "", s)` und `'"LOOMUX_STATE_DIR", "LOOMUX_LEGACY_BRAIN_DIR", '` → `'"LOOMUX_STATE_DIR", '`; danach die neue Testfunktion wieder mit ihren beiden `Setenv` prüfen.
  - **R2** `internal/cli/cases_1b2_test.go`: `LegacyDir:   dir,` löschen; `answerFrom` gibt `func(answer.Request, string, func(string)) (string, []string, error)` zurück und ruft `answer.RunWith(ports, req, registryDir, notice)`.
  - **R3** `internal/cli/serve_test.go:100-102` (Prüfung auf `seen.LegacyDir`) löschen; `area_test.go` Kommentar „a state directory, a legacy directory and qmd's configuration“ → „a state directory and qmd's configuration“.
  - **R4** `internal/cli/brain_test.go`: `brainWorldDirs` ohne `legacy`; `brainReadOnlyWorld` legt die Artefakte unter `w.state/areas/project-r` ab; die beiden Stempel-Zeilen schreiben nach `w.state`; `TestBrainSearchJudgesTheLegacyStampAtBrainNow` → `TestBrainSearchJudgesTheStampAtBrainNow`, `TestBrainReadsAReadOnlyAreaFromTheLegacyDirectory` → `…FromTheStateDirectory`; `brainNewGraph` und `TestBrainPrefersTheNewStateDirOverTheLegacyOne` löschen (ersetzt durch den neuen Test).
  - **R5** Jeder Aufruf aus der Tabelle in Step 6 verliert in Tests dasselbe Argument; Testhelfer, die `(registryDir, legacyDir string)` zurückgeben (`privacy` `buildWorld`, `workspaceWorld`, `search` `closedWorld` u. a.), geben nur noch das Zustandsverzeichnis zurück. Ein Test, der einen Bestand **nur** im Altverzeichnis anlegt, legt ihn im Zustandsverzeichnis an. `search/bench_test.go`: `benchDirs` liest nur `LOOMUX_BENCH_REGISTRY` und gibt einen Wert zurück, der Skip-Text nennt nur noch diese Variable, und `BenchmarkRegistersOfTheRealRegistry` ruft `config.ManifestDir(visible.Area, registryDir)` statt `config.ResolvedAreaDir(…)` (der `sed` in Step 4 lässt Testdateien aus).
  - **R6** Diese Tests prüfen den Rückfall selbst und werden gelöscht: `TestReindexReadsTheCollectionRecordFromTheLegacyDirectory`, `TestReadEventsReadsTheFallbackWhenThePrimaryHasNothing`, `TestEventsPathPrefersThePrimary`, `TestDropEventHidesAnEventReadFromTheFallback`, `TestExecuteSearch_AReadOnlyAreaReadsItsRegisterFromTheLegacyDirectory`, `TestMoveStock*`, `TestCopyStockRefusesASourceThatIsNotThere`; in `index/staging_test.go` jede Welt, die einen read-only-Bestand in `legacyDir` anlegt, legt ihn unter `config.ManifestDir(area, stateDir)` an (die Tests, die den Tausch und `Recover` prüfen, bleiben).

  Betroffen (geprobt, `git grep -c` an HEAD): `internal/brain/apply/{approve,place,resolve,stock}_test.go`, `check/house/federation_test.go`, `check/run/run_test.go`, `convert/run_test.go`, `index/{reindex,staging}_test.go`, `maintenance/{events,reconcile,world}_test.go`, `privacy/{areas,nesting}_test.go`, `search/{bench,search}_test.go`, `catalog/area_test.go`, `graph/read_python_test.go`, `status/status_test.go`, `answer/answer_test.go`, `internal/cli/{area,brain,braincheck,cases_1b1,cases_1b2,cases_3a,cases_3b,cases_3c,cases_4a2,cases_4c1,cases_4d,config_mutants,config_proposals_mutants,convert,index,maintenance,mergehook,serve}_test.go`, `internal/config/artifacts_test.go`, `internal/dev/benchsearch/{run,corpusrun}_test.go`, `internal/serve/{control,serve,serve_internal,update}_test.go`, `internal/serve/brain/tools_test.go`, `internal/serve/graph/tools_test.go`.

  Run: `go vet ./... > <scratch>/vet3.log 2>&1; echo $?` bis 0, dann `go test ./... -count=1 > <scratch>/t3.log 2>&1; grep -E '^(--- FAIL|FAIL)' <scratch>/t3.log`
  Expected: keine Zeile; die zwölf Fallsuiten grün (geprobt für `internal/cli` samt aller Suiten nach R1–R4).

- [ ] **Step 10: Mutationsrunde.** `recoverStock`: „`return recoverDir(…)` → `_ = recoverDir; return nil`“ tötet `TestApprovePutsAnAsideStockBackBeforeItAdvancesTheRegister` (geprobt), „`if !area.ReadOnly` → `if false && !area.ReadOnly`“ tötet `TestRecoverStockLeavesAWritableAreaAlone` (geprobt); `ArtifactLookup.Resolve`/`WritePath`: „`l.Primary` → `\"\"`“ gegen `-run ArtifactLookup`; `NewArtifactLookup`: „`StateDir()` → `os.Getenv(\"LOOMUX_LEGACY_BRAIN_DIR\")`“ tötet `TestNewArtifactLookupTakesTheStateDirectoryAlone`. Overlay-JSON z. B.:

  ```bash
  W=$(cygpath -m "$PWD"); M=$(cygpath -m <scratch>/stock_mut.go)
  printf '{"Replace":{"%s/internal/brain/apply/stock.go":"%s"}}' "$W" "$M" > <scratch>/ov.json
  go test -overlay <scratch>/ov.json ./internal/brain/apply/ -run 'TestApprovePutsAnAsideStock|TestRecoverStock' -count=1
  ```

- [ ] **Step 11: Doku dieses Tasks.**
  - `docs/en/cli-reference.md` / `docs/de/cli-reference.md`: den Punkt `LOOMUX_LEGACY_BRAIN_DIR` in der Liste der Umgebungsvariablen (Zeile 32) löschen; im Absatz „The five data commands read …“ (en ~600, de ~623) die Sätze ab „A read-only area's artefacts … are read from loomux's state directory first …“ bis „… is read through `.ultra-brain/config.toml` or `.brain.toml`.“ ersetzen durch „A read-only area's artefacts (`index.md`, `graph.json`, `_identities.tsv`) and the reconcile stamp are read from loomux's state directory, under `areas/<flat scope>` and `maintenance/`; an area directory whose `.loomux/config.toml` is missing or has no `[area]` table is not declared.“; im Upkeep-Abschnitt („They write only to loomux's state directory and read the legacy one as the fallback described above“ und „**Environment**: … `LOOMUX_LEGACY_BRAIN_DIR` is the fallback and is never written“) den Rückfall streichen; im `convert`-Abschnitt („**Environment**: the registry and the area declarations come from `LOOMUX_STATE_DIR`, with `LOOMUX_LEGACY_BRAIN_DIR` as the fallback“) ebenso. Den `area check`-Abschnitt nicht ändern.
  - `docs/en/configuration.md` / `docs/de/configuration.md`: in der Tabelle der Orte die beiden Zusätze „; falls back to `%LOCALAPPDATA%\brain\…` for reading until stage 4e“ streichen; den Absatz unter der Tabelle ab „`LOOMUX_STATE_DIR` overrides the state directory and `LOOMUX_LEGACY_BRAIN_DIR` …“ ersetzen durch „`LOOMUX_STATE_DIR` overrides the state directory; there is no command-line flag for it.“
  - `testdata/cases/README.md` Zeilen 29, 51 („both `LOOMUX_STATE_DIR` and `LOOMUX_LEGACY_BRAIN_DIR`“ → „`LOOMUX_STATE_DIR`“) und 310 (die Aufzählung der gesetzten Variablen ohne `LOOMUX_LEGACY_BRAIN_DIR`).
  - `docs/wiki/topics/datenmodell-und-bereiche.md`: „und lässt Artefakte und Reconcile-Stempel bis Stufe 3 unter `LOOMUX_LEGACY_BRAIN_DIR`“ → „Artefakte und Reconcile-Stempel liegen im selben Zustandsverzeichnis; ein zweites liest loomux nicht mehr.“
  - Grep danach: `git grep -n "LOOMUX_LEGACY_BRAIN_DIR\|LegacyBrainDir\|ResolvedAreaDir" -- ':!docs/.superpowers' ':!docs/en/benchmarks.md' ':!docs/de/benchmarks.md' ':!CHANGELOG.md'` → nur noch `TestBrainReadsNothingFromTheOldStateDirectory` und `TestNewArtifactLookupTakesTheStateDirectoryAlone`.

- [ ] **Step 12: Commit** (`<scratch>/msg3.txt`):

  ```
  refactor(brain)!: read artefacts from loomux's state directory alone

  The fallback to ultra-brain's state directory is gone: ArtifactLookup
  answers the one state directory, a read-only area's stock is read where
  `index` writes it, and every fallbackDir parameter from the commands down to
  serve and the bench goes with it. approve no longer copies an old stock
  over; it only puts an aside back that a killed `index` left, before it
  advances the register.

  BREAKING CHANGE: LOOMUX_LEGACY_BRAIN_DIR is no longer read; artefacts,
  stamps and the record of qmd collections come from LOOMUX_STATE_DIR alone.
  ```

  Run: `git add -A internal testdata/cases/README.md docs && git commit -F <scratch>/msg3.txt > <scratch>/commit3.log 2>&1; echo $?`

---

### Task 4: `Manifest.Lanes` entfällt

**Files:**
- Modify: `internal/config/manifest.go`, `internal/config/manifest_test.go`

**Interfaces:**
- Produces: `config.Manifest` ohne Feld `Lanes`; `config.LaneConfig` entfällt.

- [ ] **Step 1: Den Wächtertest schreiben** (er ist an HEAD grün und muss es bleiben: kein Leser fragt `[check]`, und eine alte Datei mit dem Schlüssel liest weiter) — an `internal/config/manifest_test.go`:

  ```go
  // `[check] lanes` is read by nobody, and a manifest that still carries it --
  // as a list or as a table -- reads as one that does not: unknown keys are not
  // judged here. `area check` names the key as ignored.
  func TestReadManifestPassesOverAnOldCheckLanesKey(t *testing.T) {
  	for _, lanes := range []string{"[check]\nlanes = [\"gofmt\"]\n", "[check.lanes]\ngofmt = \"gofmt -l .\"\n"} {
  		dir := t.TempDir()
  		if err := os.WriteFile(manifestIn(t, dir), []byte("[area]\nscope = \"project/test\"\n\n"+lanes), 0o644); err != nil {
  			t.Fatal(err)
  		}
  		m, err := ReadManifest(dir)
  		if err != nil || m.Scope != "project/test" {
  			t.Fatalf("%q: got %+v, %v", lanes, m, err)
  		}
  	}
  }
  ```

  Run: `go test ./internal/config/ -run TestReadManifestPassesOverAnOldCheckLanesKey -count=1` → `ok`.

- [ ] **Step 2: `TestReadManifestCheckLanes` und `TestReadManifestCheckLanesTable` löschen.**

- [ ] **Step 3: Entfernen** (geprobt): in `manifest.go` den Typ `LaneConfig` samt Kommentar, das Feld `Lanes []LaneConfig` in `Manifest`, das Drahtfeld `Check struct { Lanes toml.Primitive … } \`toml:"check"\``, den ganzen Block `var lanes []LaneConfig … }` in `readManifestAmong`, die Zuweisung `Lanes: lanes,` und den Import `sort`; `meta, err := toml.Decode(string(data), &file)` wird

  ```go
  		if _, err := toml.Decode(string(data), &file); err != nil {
  			return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
  		}
  ```

  Run: `gofmt -w internal/config/manifest.go && go vet ./... > <scratch>/vet4.log 2>&1; echo $?; go test ./internal/config/ ./internal/cli/ -run 'ReadManifest|AreaCheck|ClassifyKeys' -count=1`
  Expected: 0, beide `ok` (geprobt; `TestClassifyKeysCallsALanesTableIgnoredWithAHint` bleibt grün).

- [ ] **Step 4: Mutation.** Kein neuer Code mit Regel; der Wächtertest ist gegen den Mutanten „`toml.Decode` → `toml.DecodeStrict`-artiger Abbruch bei unbekanntem Schlüssel“ nicht nötig. Keine Runde, im Bericht so vermerken.

- [ ] **Step 5: Commit** (`<scratch>/msg4.txt`):

  ```
  refactor(config): drop the unread [check] lanes of a manifest

  Manifest.Lanes and LaneConfig had no reader since the check chain moved to
  [verify]. A manifest that still carries [check] lanes reads as before;
  `area check` reports the key as ignored.
  ```

---

### Task 5: Der MCP-Import schreibt das Ergebnis nach Regeln um

**Files:**
- Modify: `internal/dev/importcases/importcases.go` (Feld `Result`), `internal/dev/importcases/mcp.go`
- Create: `internal/dev/importcases/result_test.go`

**Interfaces:**
- Produces: `importcases.Mapping.Result []Rule \`toml:"result"\``; Karten können `[[result]]`-Tabellen tragen. Paketintern `var readRecorded = os.ReadFile`.

- [ ] **Step 1: Tests schreiben.** `internal/dev/importcases/result_test.go`:

  ```go
  package importcases

  import (
  	"errors"
  	"os"
  	"path/filepath"
  	"strings"
  	"testing"
  )

  // resultRecording writes one recorded MCP case under from whose result text
  // is text, as the recorder indents it.
  func resultRecording(t *testing.T, from, text string) {
  	t.Helper()
  	dir := filepath.Join(from, "brain-status", "advice")
  	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
  		t.Fatal(err)
  	}
  	files := map[string]string{
  		"call":   `{"tool":"status","arguments":{},"channel":"cloud"}`,
  		"result": "{\n  \"isError\": false,\n  \"text\": \"" + text + "\"\n}\n",
  	}
  	for name, body := range files {
  		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
  			t.Fatal(err)
  		}
  	}
  }

  func statusRule() Mapping {
  	return Mapping{
  		Tools:  []Rule{{From: "status", To: "brain_status"}},
  		Result: []Rule{{From: "run `brain reindex`", To: "run `loomux reindex`"}, {From: "brain embed", To: "loomux embed"}},
  	}
  }

  // Every rule applies, every occurrence, and the bytes around them stay as the
  // recorder wrote them -- indentation and the escaped backslash included.
  func TestImportMCPRewritesTheResultByEveryRule(t *testing.T) {
  	from, to := t.TempDir(), t.TempDir()
  	resultRecording(t, from, `a\\b: run `+"`brain reindex`"+`; run `+"`brain reindex`"+`; run `+"`brain embed`")
  	if err := ImportMCP(from, to, statusRule()); err != nil {
  		t.Fatal(err)
  	}
  	got, err := os.ReadFile(filepath.Join(to, "brain-status", "advice", "result"))
  	if err != nil {
  		t.Fatal(err)
  	}
  	want := "{\n  \"isError\": false,\n  \"text\": \"a\\\\b: run `loomux reindex`; run `loomux reindex`; run `loomux embed`\"\n}\n"
  	if string(got) != want {
  		t.Fatalf("result\n%q\nwant\n%q", got, want)
  	}
  }

  // Without a rule the result is the recording, byte for byte.
  func TestImportMCPKeepsAResultNoRuleNames(t *testing.T) {
  	from, to := t.TempDir(), t.TempDir()
  	resultRecording(t, from, "run `brain reindex`")
  	m := statusRule()
  	m.Result = nil
  	if err := ImportMCP(from, to, m); err != nil {
  		t.Fatal(err)
  	}
  	got, err := os.ReadFile(filepath.Join(to, "brain-status", "advice", "result"))
  	if err != nil {
  		t.Fatal(err)
  	}
  	if want := "{\n  \"isError\": false,\n  \"text\": \"run `brain reindex`\"\n}\n"; string(got) != want {
  		t.Fatalf("result %q, want %q", got, want)
  	}
  }

  func TestImportMCPReportsAResultItCannotRead(t *testing.T) {
  	from, to := t.TempDir(), t.TempDir()
  	resultRecording(t, from, "x")
  	boom := errors.New("boom")
  	saved := readRecorded
  	readRecorded = func(string) ([]byte, error) { return nil, boom }
  	t.Cleanup(func() { readRecorded = saved })
  	if err := ImportMCP(from, to, statusRule()); !errors.Is(err, boom) {
  		t.Fatalf("got %v, want the read failure", err)
  	}
  }

  // A `result` the target holds as a directory: the copy leaves it alone, and
  // the one writer of that file cannot write over it.
  func TestImportMCPReportsAResultItCannotWrite(t *testing.T) {
  	from, to := t.TempDir(), t.TempDir()
  	resultRecording(t, from, "x")
  	if err := os.MkdirAll(filepath.Join(to, "brain-status", "advice", "result"), 0o755); err != nil {
  		t.Fatal(err)
  	}
  	err := ImportMCP(from, to, statusRule())
  	if err == nil || !strings.Contains(err.Error(), filepath.Join("advice", "result")) {
  		t.Fatalf("got %v; want the blocked result file named", err)
  	}
  }
  ```

  Damit der rote Lauf nicht am Build scheitert, zuerst nur das Feld in `Mapping` (Step 3, erster Teil) und `var readRecorded = os.ReadFile` in `mcp.go` anlegen.

- [ ] **Step 2: Rot.**

  Run: `go test ./internal/dev/importcases/ -run 'Result' -count=1 > <scratch>/t5-red.log 2>&1`
  Expected: `--- FAIL: TestImportMCPRewritesTheResultByEveryRule` (Ergebnis unverändert), `--- FAIL: TestImportMCPReportsAResultItCannotRead` (Naht ungenutzt, `err == nil`); `…ItCannotWrite` und `…KeepsAResultNoRuleNames` sind grün (die Kopie scheitert am Verzeichnis mit derselben Pfadangabe bzw. kopiert unverändert) und bleiben Wächter. Geprobt: genau diese zwei `--- FAIL`, `got <nil>, want the read failure`.

- [ ] **Step 3: Implementieren.** In `importcases.go` hinter `Stdout`:

  ```go
  	// Result rewrites the recorded result of an MCP case the way Stdout
  	// rewrites a command's output: every occurrence of From becomes To, in
  	// the file as the recorder wrote it.
  	Result []Rule `toml:"result"`
  ```

  und den Doc-Kommentar von `Mapping` auf „[[command]], [[tool]], [[exit]], [[stdout]] and [[result]] tables“ erweitern. In `mcp.go` vor `ImportMCP`:

  ```go
  // readRecorded reads a recorded file the import rewrites, as a variable so
  // that a test can make the read fail.
  var readRecorded = os.ReadFile
  ```

  den Doc-Kommentar von `ImportMCP` beginnen mit „The result is rewritten by the [[result]] rules of m, each a deviation of the parity list, the way [[stdout]] rewrites a command's output. Beside that one thing is rewritten: the tool's name.“ und im Rumpf die Kopie ersetzen durch:

  ```go
  		// The two files this import rewrites are not copied first: one writer
  		// per file keeps the copy from being the one that fails.
  		if err := copyTree(c.Path, out, map[string]bool{"call": true, "result": true}); err != nil {
  			return err
  		}
  		recorded, err := readRecorded(filepath.Join(c.Path, "result"))
  		if err != nil {
  			return err
  		}
  		// As bytes and not decoded: the recorder's indentation and its
  		// escapes stay as they were, as they do for the call's arguments.
  		if err := os.WriteFile(filepath.Join(out, "result"), rewriteStdout(recorded, m.Result), 0o644); err != nil {
  			return err
  		}
  ```

- [ ] **Step 4: Grün und Coverage.**

  Run: `go test ./internal/dev/importcases/ -count=1 -coverprofile=<scratch>/ic.out > <scratch>/t5.log 2>&1; go tool cover -func=<scratch>/ic.out | grep mcp.go`
  Expected: `ok`; `ImportMCP` 100.0 % (geprobt).

- [ ] **Step 5: Unverändertes Korpus.** Der Re-Import von 1b-2 mit der Karte an HEAD (noch ohne `[[result]]`) muss byte-gleich bleiben:

  Run: `go run ./cmd/loomux dev import-cases --mcp --map testdata/cases/1b-2-map.toml --from testdata/cases/1b-2-source --to testdata/cases/1b-2 > <scratch>/reimport5.log 2>&1; git status --short testdata`
  Expected: leer (geprobt).

- [ ] **Step 6: Mutation.** „`rewriteStdout(recorded, m.Result)` → `recorded`“ tötet `…RewritesTheResultByEveryRule`; „`\"result\": true` aus der Skip-Liste“ — die Kopie schreibt dann zuerst, der eigene Schreiber überschreibt: Überlebender, Begründung „ein Schreiber je Datei ist eine Konvention, kein Verhalten“ in „Überlebende Mutanten“.

- [ ] **Step 7: Commit** (`<scratch>/msg5.txt`):

  ```
  feat(dev): rewrite a recorded MCP result by [[result]] rules on import

  import-cases --mcp copied a recorded result unchanged. A map can now carry
  [[result]] rules that replace text in it, as [[stdout]] does for a command's
  output, so that a deviation of loomux's wording stays reproducible on every
  re-import instead of being edited into the corpus by hand.
  ```

---

### Task 6: Ratschläge nennen loomux' Befehle

**Files:**
- Modify: `internal/brain/graph/read.go` (12 Meldungen + `ErrNotIndexed`), `internal/brain/search/stamp.go` (`ReconcileAdvice` + Kommentar), `internal/brain/status/status.go:61,234`
- Test: `internal/brain/graph/read_python_test.go`, `read_test.go`, `internal/brain/search/search_test.go`, `stamp_test.go`, `internal/brain/status/status_test.go`, `internal/cli/brain_test.go`, `internal/serve/upkeep_test.go`
- Modify: `testdata/cases/1b-1-map.toml`, `testdata/cases/1b-2-map.toml`; neu importiert: 6 Fälle in `testdata/cases/1b-1`, 6 in `1b-2`
- Docs: `docs/en|de/cli-reference.md` (Punkt „Advice“, die Beispiele in `brain neighbors`/`brain status`), `testdata/cases/README.md` (Zeilen `1b-1-map.toml`, `1b-2-map.toml`), `docs/.superpowers/parity/stufe-1b-1.md` (Zeile in der Abweichungstabelle), `stufe-1b-2.md` (neuer Abschnitt)

**Interfaces:**
- Consumes: `importcases.Mapping.Result` (Task 5).
- Produces: `graph.ErrNotIndexed` = `"never indexed; run \`loomux reindex\`"`, `search.ReconcileAdvice` = `"run \`loomux reconcile\`"`.

- [ ] **Step 1: Die Erwartungen in den Tests zuerst ändern** (rot): in den sieben Testdateien `` `brain reindex` `` → `` `loomux reindex` ``, `` `brain reconcile` `` → `` `loomux reconcile` ``, `` `brain embed` `` → `` `loomux embed` `` (geprobt mit `sed -i 's/`brain reindex`/`loomux reindex`/g; s/`brain reconcile`/`loomux reconcile`/g; s/`brain embed`/`loomux embed`/g'`). `TestReconcileConstantsAreThePythonOnes` heißt danach `TestReconcileConstants` und sein Kommentar sagt, der Ratschlag sei loomux' eigener (Abweichung in `stufe-1b-1.md`).

- [ ] **Step 2: Rot.**

  Run: `go test ./internal/brain/graph/ ./internal/brain/search/ ./internal/brain/status/ ./internal/serve/ -count=1 > <scratch>/t6-red.log 2>&1; grep -c -- '--- FAIL' <scratch>/t6-red.log`
  Expected: > 0, je Paket `FAIL` (geprobt: 25 Tests in diesen vier Paketen plus 5 in `internal/cli`).

- [ ] **Step 3: Die Meldungen ändern** (geprobt): `sed -i 's/run `brain reindex`/run `loomux reindex`/g' internal/brain/graph/read.go internal/brain/status/status.go`, `sed -i 's/run `brain embed`/run `loomux embed`/' internal/brain/status/status.go`, und in `stamp.go` `const ReconcileAdvice = "run \`loomux reconcile\`"`. Danach mit `grep -c` je Datei prüfen: `read.go` 12, `status.go` 2, `stamp.go` 1. Den Kommentar über `ReconcileAdvice` (`stamp.go:18-22`) neu schreiben: der Text ist loomux' eigener Ratschlag; die aufgezeichneten Fälle halten ihn über die Regeln der Importkarten.

- [ ] **Step 4: Die Karten.** An `testdata/cases/1b-1-map.toml` anhängen:

  ```toml

  # Deviation "Ratschläge nennen loomux" in parity/stufe-1b-1.md: the reference
  # advises its own commands, loomux advises its own.
  [[stdout]]
  from = "run `brain reindex`"
  to   = "run `loomux reindex`"

  [[stdout]]
  from = "run `brain reconcile`"
  to   = "run `loomux reconcile`"

  [[stdout]]
  from = "run `brain embed`"
  to   = "run `loomux embed`"
  ```

  und an `testdata/cases/1b-2-map.toml` dieselben drei Regeln als `[[result]]`, Kommentar mit `parity/stufe-1b-2.md`. Den Kopfkommentar von `1b-1-map.toml` („Stage 1b-1 translates one command prefix.“) um „and rewrites the advice“ ergänzen.

- [ ] **Step 5: Neu importieren.**

  ```bash
  go build -o <scratch>/lx.exe ./cmd/loomux
  <scratch>/lx.exe dev import-cases --map testdata/cases/1b-1-map.toml --from testdata/cases/1b-1-source --to testdata/cases/1b-1 > <scratch>/i1.log 2>&1; echo $?
  <scratch>/lx.exe dev import-cases --mcp --map testdata/cases/1b-2-map.toml --from testdata/cases/1b-2-source --to testdata/cases/1b-2 > <scratch>/i2.log 2>&1; echo $?
  git status --short testdata
  ```

  Expected (geprobt): beide 0; geändert genau `1b-1/brain-status/{backlog,never-indexed,stamp-missing,stamp-stale,vault-cloud,vault-local}/stdout` und `1b-2/brain-neighbors/{broken-graph,never-indexed}/result`, `1b-2/brain-status/{backlog,broken-graph,never-indexed,vault-cloud}/result`, dazu die beiden Karten.

- [ ] **Step 6: Grün.**

  Run: `go test ./... -count=1 > <scratch>/t6.log 2>&1; grep -E '^(--- FAIL|FAIL)' <scratch>/t6.log`
  Expected: keine Zeile (geprobt, samt `TestRecordedCasesOfStage1b1` und `TestRecordedMCPCasesOfStage1b2`).

- [ ] **Step 7: Akten und Doku.**
  - `docs/.superpowers/parity/stufe-1b-1.md`, Tabelle „Fall / Bereich | Alt | Neu | Begründung | Freigabe“, neue Zeile: `| Ratschläge nennen loomux (\`brain-status/{backlog,never-indexed,stamp-missing,stamp-stale,vault-cloud,vault-local}\`) | \`run \`brain reindex\`\`, \`run \`brain reconcile\`\`, \`run \`brain embed\`\` | \`run \`loomux reindex\`\`, … | Die Befehle von ultra-brain gibt es nach der Umstellung nicht mehr; der Ratschlag nennt den Befehl, der ihn erfüllt. Getragen von drei \`[[stdout]]\`-Regeln in \`1b-1-map.toml\`, beim Re-Import reproduzierbar | Spec Aufräum-PR, 2026-10-03 |`.
  - `docs/.superpowers/parity/stufe-1b-2.md`: neuer Abschnitt „## 8. Ratschläge nennen loomux“ mit den sechs Fällen, den drei `[[result]]`-Regeln und dem Hinweis, dass `brain-status/backlog` den in der Spec fehlenden `brain embed`-Ratschlag trägt.
  - `docs/en/cli-reference.md` Punkt „**Advice**: the messages still name `brain reindex` …“ → „**Advice**: the messages name loomux's own commands, `loomux reindex`, `loomux reconcile` and `loomux embed`; the recorded cases of 1b-1 and 1b-2 carry the reference's wording through rewrite rules of their import maps.“; die Beispiele (``<scope>: never indexed; run `brain reindex` ``, ``last reconcile: never; run `brain reconcile` ``, ``… older than 24 h, run `brain reconcile` ``) auf `loomux` umstellen; `docs/de/cli-reference.md` gleichlautend.
  - `testdata/cases/README.md`: Zeile `1b-1-map.toml` („One rule: `brain-mcp ` → `loomux brain `.“) → „One command rule, `brain-mcp ` → `loomux brain `, and three `[[stdout]]` rules putting `loomux reindex`, `loomux reconcile` and `loomux embed` in place of the reference's advice.“; Zeile `1b-2-map.toml` um „and three `[[result]]` rules for the same advice“ ergänzen.

- [ ] **Step 8: Mutation.** Je Meldung genügt der Wortlauttest; Mutant „`ReconcileAdvice` zurück auf `brain reconcile`“ tötet `TestRecordedCasesOfStage1b1` (`brain-status/stamp-missing`) und `TestLinesNameAMissingStampNever`.

- [ ] **Step 9: Commit** (`<scratch>/msg6.txt`):

  ```
  fix(brain): name loomux's commands in the advice of status and the graph reader

  A missing or broken graph, an old reconcile stamp and an embedding backlog
  advised `brain reindex`, `brain reconcile` and `brain embed`, the commands
  of the tool loomux replaced. They now name loomux reindex, reconcile and
  embed. The recorded cases follow through rewrite rules of their import
  maps, not by hand.
  ```

  Run: `git add -A internal testdata docs && git commit -F <scratch>/msg6.txt > <scratch>/commit6.log 2>&1; echo $?`

---

### Task 7: Nachher messen und eintragen

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (nur anhängen)

- [ ] **Step 1: Nachher messen** — im Zweig, gleiche Befehle wie Task 0 Step 4, ohne `LOOMUX_BENCH_LEGACY`:

  ```bash
  LOOMUX_BENCH_REGISTRY="$LOCALAPPDATA/loomux" go test ./internal/brain/search -run '^$' -bench 'RealRegistry|ExecuteSearchWithoutTheEngine' -benchtime=1x -count=1 > <scratch>/measure/after-cold.txt 2>&1
  LOOMUX_BENCH_REGISTRY="$LOCALAPPDATA/loomux" go test ./internal/brain/search -run '^$' -bench 'RealRegistry|ExecuteSearchWithoutTheEngine' -count=5 > <scratch>/measure/after-warm.txt 2>&1
  ```

  Expected: drei Benchmarks, kein `SKIP`. Ist der Registry-Schritt (Task 0 Step 1) noch nicht erledigt, steht das im Eintrag, und die Messung wird nach ihm wiederholt.

- [ ] **Step 2: Eintrag** in beiden Sprachen, am Ende, Form wie die Einträge vom 2026-10-02: Überschrift `## 2026-10-0X HH:MM — Reading Without the Old State Directory` / `## … — Lesen ohne das alte Zustandsverzeichnis`; was gemessen wurde (die drei Benchmarks, reale Registry, nur lesend); Basis (`origin/master` Hash aus Task 0) gegen Änderung (Zweig-HEAD); kalt (`-benchtime=1x`) und warm (Median aus fünf); die Zahlen als Tabelle; „Key Findings“: kein read-only-Bereich mehr nach dem Registry-Schritt, darum ändert sich `VisibleAreas` nicht; gespart ist das `os.Stat` in `Resolve` für Stempel und Sammlungsliste; Unterschiede unter dem Rauschen werden so benannt. Historische Einträge bleiben wörtlich.

- [ ] **Step 3: Den Vorher-Worktree entfernen** (`git worktree remove <scratch>/measure/before`, `git worktree prune`, `git worktree list` lesen).

- [ ] **Step 4: Commit** (`<scratch>/msg7.txt`): `docs(benchmarks): measure the brain readers without the old state directory`.

---

### Task 8: Migrationsplan, Roadmap, Fusions-Spec, Akten

**Files:**
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, `docs/en/migration.md`, `docs/de/migration.md`, `README.md`, `README.de.md`, `docs/.superpowers/parity/stufe-3a.md`, `docs/.superpowers/parity/stufe-4e.md`

- [ ] **Step 1: Fusions-Spec zuerst.** In der Nachtragstabelle hinter #27 eine Zeile `| 28 | — | Der Aufräum-PR der Stufe 4e … |` mit den Entscheidungen der Spec 6 (keine Aside-Auflösung, die Regel read-only → `<state>/areas/<scope>` bleibt wegen `dev bench search`), 9 (Workspace ohne `[area]` ist kein brain-Bereich, PR #67), 11 (Auflagen aus `stufe-3a.md`: Ratschläge getragen, Asides und read-only-Deklarationen entfallen, `merge-events.done.tsv`/`qmd-collections.json` ausdrücklich fallengelassen) und 12 (Wellenbefunde 1, 2, 4, 5 als Roadmap-Folgezeilen), dazu Nutzerentscheid (b) (die sechs Entscheidungen aus `artefakte-nach-lebensdauer.md` blockieren 4e ✅) und (c) (Arbeitsbereich ohne `[area]` in `maintenance`, `lint`, `convert`, `reconcile` auslassen: eigener PR); Spalte Begründung mit den Messungen dieses Plans (zwei Altmanifeste in Lauf-Bäumen, alle Suiten ohne Rückfall grün, 12 Ratschlagsfälle); Status „freigegeben 2026-10-03 mit der Spec `2026-10-03-loomux-stufe-4e-aufraeumen-design.md`“. In Zeile #18 (`Manifest.Lanes` bleibt geparst, bis `loomux migrate`) und in der Klassentabelle Zeile „(a) Lese-Rückfälle“ den Vermerk „erledigt mit dem Aufräum-PR (Nachtrag #28)“ anhängen; #25 („Aside-Auflösung“) und #26 („bleibt, weil `#Obsidian/AI` …“) bekommen „überholt durch #28: `#Obsidian/AI` ist aus der Registry genommen, die Auflösung wird nicht gebaut“. Die Prio-Spalte der Stufentabelle bleibt unverändert.

- [ ] **Step 2: `migration.md` en/de, Zeile 4e.** Status bleibt `open` / `offen` (Nutzerentscheid b). Im Inhaltstext den Satz ab „Open: five loomux findings …“ bis „… still need a carrier.“ ersetzen durch (en): „Open: the six decisions of `parity/artefakte-nach-lebensdauer.md`, which the user set on 2026-10-03 to block 4e ✅; four findings of the wave as roadmap rows (fusion spec #28). Done with the clean-up pull request (fusion spec #28): `LegacyBrainDirUntilStage3` and `LOOMUX_LEGACY_BRAIN_DIR`, `ReadAreaManifestUntilStage4` with the old manifest names (an area that carries only one is refused with a pointer to `loomux area check`) and `Manifest.Lanes` are gone; the advice names `loomux reindex|reconcile|embed`; of the four obligations of `parity/stufe-3a.md` the advice is carried, the asides and the read-only declarations fell away with the read-only areas, and carrying `merge-events.done.tsv` and `qmd-collections.json` over is dropped.“; de gleichlautend. Den Satz „After the wave one clean-up pull request: …“ streichen. Die Fähigkeitszeilen, die `LOOMUX_LEGACY_BRAIN_DIR` oder den Rückfall als Stand nennen (`git grep -n "legacy directory\|LOOMUX_LEGACY" docs/en/migration.md docs/de/migration.md`), nachziehen (3a-Zeile: „the legacy directory only read“ → „until the clean-up pull request the legacy directory was read as a fallback“).

  Run: `go test ./internal/plancheck/ -count=1`
  Expected: `ok` (Status und Prio unverändert; der Inhalt wird nicht geprüft).

- [ ] **Step 3: Roadmap.** In `README.md` unter „Coming“ und `README.de.md` unter „Kommt“ fünf Zeilen in der Form der vorhandenen: (1) „Leave a workspace without `[area]` out of `maintenance`, `lint`, `convert` and `reconcile` too — today they take it as an area that declares nothing“ (Nutzerentscheid c); (2)–(5) die Wellenbefunde 1, 2, 4, 5 aus `parity/stufe-4e.md`, Messung 10 (Profilart ohne Prüfgegenstand endet mit Exit 1; `verify.profiles` nicht setzbar; Repo-Wurzel als Wiki abgelehnt; `dev bench cases` ohne Hook), je mit Stufe „4e-Folge“, Abhängigkeit „—“ und Prio, wie die Fusions-Spec #28 sie vergibt (vorher dort festlegen; ohne Festlegung dort nicht erfinden).

- [ ] **Step 4: `parity/stufe-3a.md`.** Die drei Abschnitte „### Stufe 4: `migrate` muss `merge-events.done.tsv` und `qmd-collections.json` mitnehmen“, „### Umstieg: die Ratschläge `brain reindex` und `brain reconcile` ziehen mit um“, „### Stufe 4: der abwesende Bereichsordner nach einem erschlagenen Tausch“ und „### Stufe 4: `guard` muss die Deklaration eines read-only-Bereichs finden“ bekommen am Ende je einen Absatz „**Abgeschlossen 2026-10-0X** mit dem Aufräum-PR (Spec `2026-10-03-loomux-stufe-4e-aufraeumen-design.md`, Entscheidung 11): …“ mit dem jeweiligen Ausgang (getragen / entfällt / fallengelassen); nichts löschen. In „## Abweichungen“ eine Zeile `| Mitnehmen von \`merge-events.done.tsv\` und \`qmd-collections.json\` | … | Fallengelassen | Die Welle lief ohne; \`reindex\` baut die Sammlungen neu | Nutzer 2026-10-03 |`; die Zeilen „Zustandsort“ und „`qmd-collections.json` (S5)“ bekommen „Rückfall entfernt mit dem Aufräum-PR“.

- [ ] **Step 5: `parity/stufe-4e.md`.** Im Abschnitt „Überlebende Mutanten“ den Satz „Die Runde für den Aufräum-PR (Stück C) steht noch aus …“ durch die Runde dieses PRs ersetzen (Tabellen aus den Tasks 1–6: je Funktion Mutanten, getötet, überlebt mit Begründung, BADMUTANT, Methode „Handrunde per `go test -overlay`, gezielte `-run`“). Unter „Messungen“ eine „### 11. Der Aufräum-PR (2026-10-0X)“ mit den Zählungen aus „Was die Spec behauptet …“ dieses Plans (zwei Altmanifeste in Lauf-Bäumen, 0 Übersetzer, 0 Abweichungen, 12 Ratschlagsfälle) und dem Verweis auf den Benchmark-Eintrag.

- [ ] **Step 6: Abschlussgrep** über alles Markdown außer Arbeitspapieren, `testdata` und den Benchmark-Chroniken:

  Run: `git grep -n "LOOMUX_LEGACY_BRAIN_DIR\|LegacyBrainDirUntilStage3\|ReadAreaManifestUntilStage4\|Manifest.Lanes\|brain reindex\|brain reconcile\|brain embed\|falls back to .*brain" -- '*.md' ':!docs/.superpowers' ':!testdata' ':!docs/en/benchmarks.md' ':!docs/de/benchmarks.md' ':!CHANGELOG.md'`
  Expected: nur `docs/wiki/syntheses/gegenpruefung-vor-jeder-designempfehlung.md:33` (Chronik eines damaligen Stands, bleibt) — jede andere Zeile nachziehen.

- [ ] **Step 7: Commit** (`<scratch>/msg8.txt`): `docs(migration): record the removal of the read fallbacks and what still blocks the stage` — Rumpf: welche Zeilen sich änderten, ohne Arbeitspapiere im Kopf zu nennen.

- [ ] **Step 8: PR vorbereiten** mit dem Skill `release-pr`: Label `release:major`, Rumpf mit `Release: major — old manifest names and LOOMUX_LEGACY_BRAIN_DIR are no longer read`, `## Changelog` mit `### Removed` (die zwei Altnamen, die Umgebungsvariable, `[check] lanes`), `### Changed` (Meldung bei Altmanifest, Ratschläge), `### Added` (`[[result]]` in `dev import-cases`). Den Push-Befehl dem Menschen nennen; nicht pushen. Vor dem Merge: Task 0 Step 1 bestätigt, danach Selbstnutzung `loomux brain catalog --scope all` und `loomux brain status` an der echten Registry, beide Exit 0.

---

## Selbstprüfung

- Spec-Abdeckung: Entscheidungen 2 (Task 3), 3 (Tasks 1, 2), 4 (Task 4), 5 (Task 2), 6 (Task 3: Rückfallteil weg, `ManifestDir`-Regel bleibt), 7 (Task 3, Test nach Korrektur 3), 8 (Tasks 1, 2), 9 (vorher gemergt, Nachzug Task 2), 10 (Tasks 5, 6), 11 (Task 8 Step 4), 12 (Task 8 Step 3), Registry-Schritt (Task 0), Messen (Tasks 0, 7), Doku (Tasks 2, 3, 6, 8). Fertig-Bedingungen: 4e bleibt `offen` (Nutzerentscheid b).
- Platzhalter: keine; die Testanpassungen in Task 3 Step 9 sind Regeln samt Dateiliste für eine übersetzergeführte Signaturänderung, die Signaturen selbst stehen vollständig in Step 6.
- Typen: `ReadAreaDeclaration`, `IsUndeclared`, `OldManifestNames`, `recoverStock`, `registerOf`, `Mapping.Result`, `readRecorded` heißen in allen Tasks gleich.
- Kompiliert und geprobt (Scratch-Worktrees auf `714d4b27`): alle Go-Blöcke der Tasks 1–6 und die Tabelle in Task 3 Step 6. Produktseite: `go build ./...` und `go vet` nach Task 2 und nach Task 3. Testseite nach Task 3 übersetzt und gelaufen nur `internal/config`, `internal/brain/apply`, `internal/cli` (vollständig, samt aller Fallsuiten) und für Task 5 `internal/dev/importcases`; die Testdateien von `privacy`, `search`, `status`, `index`, `maintenance`, `serve`, `benchsearch`, `check/*`, `answer`, `catalog`, `graph` sind nach der Signaturänderung **nicht** übersetzt worden — R5/R6 in Task 3 Step 9 sind Regeln, keine geprobte Liste. Nach Task 2 lief `go test ./...` bis auf die in Step 9 genannten Einzelfälle grün. Rotläufe gemessen: Task 1, Task 2 (Workspace), Task 3 (CLI-Test), Task 5, Task 6. Die Mutanten von `recoverStock` in Task 3 Step 10 per Overlay. Task 4 hat keine Mutationsrunde (nur Entfernen).
