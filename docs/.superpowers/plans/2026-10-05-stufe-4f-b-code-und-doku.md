# Stufe 4f PR B: Code und Doku Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Code und Doku von loomux nennen `ultraloom` und `ultra-brain` nicht mehr, außer im festen Restbestand unten. `loomux area check` und `loomux dev switchover` fallen weg, ebenso jede Erkennung von Altinstallationen.

**Architecture:** Zwölf Tasks, je ein Commit oder wenige Commits, ein Thema je Commit. Zuerst fallen die Befehle und Erkenner, die Funktion tragen (Task 2–8), dann die Namen in Kommentaren, Tests und Doku (Task 9–11). Vor dem ersten Löschen stehen die Proben aus der Spec in der Akte (Task 1). Am Ende prüft ein Grep gegen eine eingefrorene Restliste (Task 12). Kein Code entsteht neu; die Arbeit ist Löschen und Umbenennen.

**Tech Stack:** Go 1.26, Git Bash, `loomux` aus diesem Baum (`go run ./cmd/loomux …`).

**Spec:** `docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md`, Abschnitt „PR B: Code und Doku“; Fusions-Spec „#24 im Einzelnen“ und Nachträge #32, #34. Die Akte ist `docs/.superpowers/parity/stufe-4f.md`.

**Zweig und Basis:** `refactor/drop-predecessor-references`, gestapelt auf `docs/predecessor-equality` bei `49230aab` (PR A, xidus90/loomux#81, gefaltet und gepusht, auf `origin/master` v7.2.0). Wird PR A vor dem Merge noch einmal umgeschrieben, setzt dieser Zweig mit `git rebase --onto <neues PR-A-HEAD> 49230aab` neu auf.

## Global Constraints

- Suchliste aus #24, als Muster für jeden Grep dieses Plans:
  `ultraloom|ultra-brain|ultra_brain|UltraBrain|ulguard|ulinit|ulflow|brain-mcp|brain guard|ultraLoomOwned|ultraloom-wiki-guard|\.brain\.toml|LegacyBrainDir|ReadAreaManifestUntil|specs-ul|specs-ub|plans-ub|plans-ul|bench-ub` (mit `-i`).
- Nicht angefasst in diesem PR, sondern in PR C: `testdata/cases/**` außer den elf `index.yml` aus Task 6, `internal/dev/importcases`, `internal/dev/recordcase` (samt `dev record-case` und `dev record-mcp-case`), ihre Tests in `internal/cli/dev_test.go` und `internal/cli/devmcp_test.go`, die Toleranzlisten der Wiedergabetests (`internal/cli/cases_*_test.go`, z. B. `cases_3a_test.go:95`).
- Benannte Ausnahmen, die unverändert bleiben: `docs/{en,de}/benchmarks.md`, `testdata/bench/1a-hooks.json`, `testdata/bench/search/v1/baseline/*`, die Herkunftsspalte von `docs/*/migration.md`, `docs/.superpowers/**`, `CHANGELOG.md` (siehe Entscheidung E1).
- `.loomux/config.toml` schreibt kein Agent (AGENTS.md). Die Kommentaränderung in Task 11 führt ein Mensch aus.
- Coverage 100 % je Funktion. Fällt eine Funktion weg, fallen ihre Tests mit; bleibt ein Helfer ohne Aufrufer, fällt er auch (der Lint meldet `unused`).
- Kommentare, Fehlermeldungen und Commits englisch; Akte und Spec deutsch. Commit-Texte nennen kein Arbeitspapier, keine Stufe, keinen Task.
- Commits: Conventional Commits. Ein Wegfall eines Befehls trägt `!`. Kein `Co-Authored-By` auf ein Modell.
- Jede lange Ausgabe (Tor, ganze Testsuite) geht erst ganz in eine Datei im Scratchpad (`> "$SCRATCH/<name>.txt" 2>&1`), dann wird gefiltert. Bei Rot zuerst nach `--- FAIL` suchen.
- Kein Hintergrundprozess bleibt nach dem Ende eines Tasks liegen.
- Kein Subagent ruft `reindex`, `convert`, `fetch`, `area add` oder `reconcile` gegen den echten Zustand oder das echte Projekt auf. Proben laufen mit `LOOMUX_STATE_DIR` und `XDG_CONFIG_HOME` auf leeren Ordnern im Scratchpad oder als Go-Test über `run(...)`. Der Wächter verweigert einige dieser Befehle ohnehin.
- `dev import-cases` wird in diesem Zweig **nicht** aufgerufen: ein Re-Import aus `3a-source`/`3b-source` brächte die Zeilen zurück, die Task 6 aus den `index.yml` nimmt.

## Offene Entscheidungen (vor Task 0, vom Nutzer)

Der Plan nimmt für jede eine Annahme. Fällt die Antwort anders aus, ändert sich nur der genannte Task.

- **E1 — Changelog.** Erledigt mit PR A (xidus90/loomux#81, `49230aab`): Die Ausnahme umfasst fünf Zeilen, benannt nach Eintrag (7.0.1, 7.0.0, 4.2.2, 2.5.0, 2.3.0). Der Changelog wird nicht angefasst.
- **E2 — Wiki-Quellen.** Die `sources:`-Zeilen in `docs/wiki/**` zeigen in die Archive, und der Dateiname des Archivs trägt selbst `ultra-brain` (`specs-ub/2026-08-18-ultra-brain-architektur-design.md`). Nach (h) werden sie erst mit dem letzten Folgeprojekt umgehängt. Die Spiegel unter `internal/brain/apply/testdata/frontmatter/w-*.md` sind Kopien solcher Seiten. **Annahme:** Diese Zeilen bleiben bis (h). Nur die Prosa der Seiten wird neu gefasst. Betrifft Task 11 und die Ausnahmeliste von PR C.
- **E3 — `docs/wiki/log.md`.** Das Log ist eine Chronik wie `benchmarks.md`, mit fünf Treffern (`:47`, `:67`, `:93`, `:97`, `:107`). **Annahme:** Das Log wird eine benannte Ausnahme; alte Einträge bleiben, neue nennen die Altprojekte nicht. Betrifft Task 0 und Task 11.
- **E4 — Roadmap-Zeile „The hint beside a declaration without `[area]`“** (`README.md:180`, `README.de.md:180`). Sie ist eine Folgezeile von #28: Eine `.loomux/config.toml` ohne `[area]` neben einem Altmanifest bekommt keinen Hinweis. Mit Task 2 erkennt loomux kein Altmanifest mehr, und die Zeile hat keinen Gegenstand. **Annahme:** Sie fällt aus beiden READMEs, und der Nachtrag hält das fest. Betrifft Task 0 und Task 2.

## Korrekturen der Spec am Code (gehen in Nachtrag #35, Task 0)

- **(b) „Schutz von `.ultraloom/vendor`“ in `internal/hooks/worktree.go`:** Im Code gibt es keine Sonderregel für diesen Pfad. `.ultraloom/vendor` steht nur in Kommentaren (`:78`, `:282-283`, `:414-416`, `:496`) und in Testfixturen (`.loomux/vendor/ultraloom` in `worktree_test.go`, `.ultraloom/vendor` in `internal/worktree/mirror/mirrorcfg_test.go`). `standsInside` und `leadsInto` schützen jeden gespiegelten Pfad. Würden sie gelöscht, könnte das Aufräumen am Sitzungsende über eine Junction den Haupt-Checkout treffen. **Darum:** Die Logik bleibt, die Kommentare bekommen ein neutrales Beispiel (`.tools/godot`, das die Tests schon nutzen) (Task 9), die Fixturen neutrale Pfade (Task 10).
- **(i) „`loomux config … --propose`“ für den Kommentar in `.loomux/config.toml:58-59`:** `loomux config` kennt nur Schlüsseloperationen (`internal/cli/config.go:59`: `list`, `get`, `set`, `unset`, `proposals`, `apply`, `reject`). Einen Kommentar kann man damit nicht vorschlagen. **Darum:** Der Plan nennt dem Menschen die zwei neuen Zeilen, und er ändert sie von Hand (Task 11). Die Fixturkopie `internal/setup/testdata/loomux/loomux-config.toml` ändert der Agent (Task 9). Kein Test bindet sie an die echte Datei; sie weicht schon bei `:85` ab.
- **(a) „wird mit genau diesem Fall … geprüft“:** Ein bleibender Test mit `.brain.toml` als Eingabe widerspräche (d). **Darum:** Jeder der elf Tests, die heute den Hinweis erwarten, wird einmal auf die neue Erwartung umgestellt und grün gefahren; die Ausgabe kommt in die Akte. Danach wird er gelöscht, weil er nur noch den Fall „kein Manifest“ wiederholt (Task 2).

## Review Focus

1. **Ein Projekt, dessen `.githooks/pre-commit` noch `ulguard` ruft.** `RunsAGate` erkennt es nach Task 5 nicht mehr als Tor. `init` muss den Hook trotzdem behalten und nicht überschreiben; nur die Notiz wechselt von „runs a gate already“ zu „a hook of the project is already there“. Eigentümer: Task 5, Schritt 1.
2. **Ein registrierter Bereich mit nur `.brain.toml`, gefahren über die CLI-Leser** (`brain status`, `brain catalog`, `lint --scope all`, `convert`, `reconcile`), nicht nur über die Unit-Tests. Erwartet wird dieselbe Antwort wie für einen Bereich ohne jede Datei. Eigentümer: Task 2, Schritt 6.
3. **`hook status` auf einer `settings.json` mit alten Einträgen.** Der Bericht über die loomux-Ereignisse muss vollständig bleiben; der Abschnitt über Altlasten fällt ersatzlos, und das Wort „Wiki:“ steht ohne Präfix. Eigentümer: Task 4.
4. **Ein Bereich, in dem noch ein Ordner `.ultra-brain/` mit `*.md` liegt.** Ohne den Ausschluss aus `AlwaysExcludes` indexiert `reindex` dessen Markdown. (`**/.brain.toml` fällt folgenlos, weil das Muster `**/*.md` es nie trifft.) Ein bleibender Test bräuchte den Altnamen als Eingabe. Darum gilt die Probe an der echten Registry als Schutz: Eigentümer ist Task 1, Step 2.
5. **Der Wächter nach dem Wegfall der Regel zu `dev switchover prune-hooks`.** Zwischen Basis und HEAD darf sich nur genau diese Zeilenfamilie von „verweigert“ auf „durch“ drehen. `dev bench`, `dev mutants`, `doc.exe dev …` im strikten Modus und die anderen Befehlsregeln bleiben, wie sie sind. Eigentümer: Task 3, Schritt 4.

## Restbestand nach PR B (eingefroren)

Nach Task 11 trifft der Grep der Suchliste über `git ls-files` nur noch hier. Jeder andere Treffer ist ein Befund.

| Ort | Grund | geht mit |
|---|---|---|
| `testdata/cases/**` | Aufzeichnungen und übersetzte Fälle | PR C |
| `internal/dev/importcases/**`, `internal/dev/recordcase/**` | Werkzeuge der Aufzeichnung | PR C |
| `internal/cli/dev_test.go` (Zeilen zu `record-case`/`import-cases`), `internal/cli/devmcp_test.go` | Tests dieser Werkzeuge | PR C |
| `internal/cli/cases_*_test.go`: Toleranzlisten mit Pfaden der Aufnahme (z. B. `cases_3a_test.go:95`) | spiegeln die Aufnahme | PR C (Ausnahme oder eigene Erwartung) |
| `internal/setup/templates/templates_test.go:55`, `:147` | Verbotslisten alter Namen für ausgelieferte Vorlagen; ein kleiner Vorläufer des Tor-Tests | PR C (der Tor-Test löst sie ab) |
| `docs/{en,de}/benchmarks.md`, `testdata/bench/1a-hooks.json`, `testdata/bench/search/v1/baseline/*` | Chronik | bleibt |
| `docs/*/migration.md`: Herkunftsspalte und Beschreibungen, die die Migration erzählen (#35) | Plan | Ende des Plans |
| `CHANGELOG.md`, je eine Zeile in 7.0.1, 7.0.0, 4.2.2, 2.5.0, 2.3.0 | Geschichte (E1) | bleibt |
| `docs/.superpowers/**` | Arbeitspapiere und Archive | (h) |
| `_identities.tsv` (Zeilen der Archive) | Register folgt `reindex` | (h) |
| `docs/wiki/**`: Archivverweise (`sources:`-Zeilen, im Text zitierte Archivpfade), `docs/wiki/log.md` | E2, E3, #35 | (h) bzw. bleibt |
| `internal/brain/apply/testdata/frontmatter/w-*.md` | Spiegel der Wiki-Quellen (E2) | (h) |
| Roadmap-Zeile W2 „Brain web app“ (`ultra-brain/web`) in beiden READMEs | laufendes Folgeprojekt | Web |
| Zeilen von Roadmap und Migrationsplan zu `ulflow` | laufendes Folgeprojekt | Flow |

---

### Task 0: Nachtrag #35 in der Fusions-Spec

Der Nutzer hat E2–E4 nicht einzeln entschieden, sondern mit dem Start der Umsetzung die Annahmen des Plans angenommen (2026-10-05). So steht es auch im Nachtrag.

**Files:**
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` (Nachtragstabelle hinter #34; Liste „Die benannten Ausnahmen“ unter „#24 im Einzelnen“; Zeile #15)
- Modify: `docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md` (Abschnitt PR B, Punkt (b) und (i); Abschnitt PR C, Ausnahmeliste)

- [ ] **Step 1: Nachtrag #35 schreiben**

Eine Zeile in der Form von #34. Inhalt: die Annahmen zu E2–E4 und die drei Korrekturen oben mit ihren Belegzeilen. Spalte „Stand“: „Annahmen des Plans, vom Nutzer mit dem Start der Umsetzung angenommen (2026-10-05)“.

- [ ] **Step 2: #15 ergänzen (#32 „Ergänzen“)**

In Zeile #15, Spalte 2 hinter „`scripts/install.ps1` und `install.sh`: Bauen in `~/go/bin`“ einfügen: „; ebenso `scripts/install.{sh,ps1}` von ultra-brain“.

- [ ] **Step 3: Ausnahmeliste unter „#24 im Einzelnen“ und in der 4f-Spec nachziehen**

Nach E2 und E3 kommen `docs/wiki/log.md` und die `sources:`-Zeilen in Archive (samt ihren Spiegeln unter `internal/brain/apply/testdata/frontmatter/`) zu den Ausnahmen. In der 4f-Spec, Abschnitt PR B, (b): „der Schutz von `.ultraloom/vendor` …“ wird „die Kommentare und Fixturen zu `.ultraloom/vendor` (die Schutzlogik bleibt, #35)“. Bei (i) kommt „von Hand durch einen Menschen (#35)“ hinzu. Die Ausnahmeliste in PR C, Punkt 4, wird gleich gefasst.

- [ ] **Step 4: Plancheck**

Run: `go test ./internal/plancheck/ > "$SCRATCH/t0.txt" 2>&1; tail -3 "$SCRATCH/t0.txt"`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md
git commit -F "$SCRATCH/msg-t0.txt"
```
Nachricht (per Write nach `$SCRATCH/msg-t0.txt`): `docs: settle the leftovers of removing predecessor references from code and docs`.

---

### Task 1: Proben vor dem Löschen

Kein Code. Drei Proben, ihre Ausgaben kommen in die Akte.

**Files:**
- Modify: `docs/.superpowers/parity/stufe-4f.md` (neuer Abschnitt „4. Vor PR B“)

- [ ] **Step 1: Registry lesen**

```bash
REG="${LOOMUX_STATE_DIR:-$LOCALAPPDATA/loomux}/registry.toml"
ls -la "$REG" > "$SCRATCH/t1-reg.txt" 2>&1; cat "$REG" >> "$SCRATCH/t1-reg.txt"
```
Liegt dort keine Registry, gilt der Pfad, den `defaultStateDir` in `internal/config/registry.go` für Windows bildet; ihn dort nachlesen, nicht raten. Der Pfad, der gilt, kommt in die Akte.

- [ ] **Step 2: Je Bereich die Manifeste und Altordner zählen**

Für jeden `path` der Registry (ein Skript per Write nach `$SCRATCH/t1-areas.sh`, Pfade aus der Registry):

```bash
for p in <pfade aus der Registry>; do
  printf '%s\t' "$p"
  [ -f "$p/.loomux/config.toml" ] && grep -q '^\[area\]' "$p/.loomux/config.toml" && printf 'area ' || printf 'NO-AREA '
  [ -f "$p/.brain.toml" ] && printf '.brain.toml '
  [ -f "$p/.ultra-brain/config.toml" ] && printf '.ultra-brain/config.toml '
  [ -d "$p/.ultra-brain" ] && printf '.ultra-brain/:%s-md ' "$(find "$p/.ultra-brain" -name '*.md' | wc -l)"
  echo
done > "$SCRATCH/t1-areas.txt" 2>&1
```

Expected: Jeder Bereich, der nicht als Arbeitsbereich ohne `[area]` geführt ist, zeigt `area`. Kein Bereich hat ein Altmanifest ohne `area`. Kein Bereich hat einen `.ultra-brain/`-Ordner mit `*.md`.
Weicht ein Bereich ab: **anhalten** und dem Nutzer vorlegen. Diese Probe ist die Vorbedingung der Spec vor dem Merge. Ein solcher Bereich würde nach Task 2 still manifestlos und nach Task 6 mitindexiert.

- [ ] **Step 3: `reconcile` mit unerwartetem Argument (#32, vor dem Wegfall des Korpus von `claude/scheibe-9b`)**

`reconcileCommand` (`internal/cli/maintenance.go:32-37`) prüft mit `refusesArguments` vor jedem Zugriff auf den Zustand. Die Probe läuft trotzdem gegen einen leeren Zustand:
Run: `mkdir -p "$SCRATCH/t1-state" && LOOMUX_STATE_DIR="$SCRATCH/t1-state" go run ./cmd/loomux reconcile unexpected-arg > "$SCRATCH/t1-reconcile.txt" 2>&1; echo "exit $?" >> "$SCRATCH/t1-reconcile.txt"`
Expected: `exit 2` und eine Meldung von `refusesArguments` auf stderr; `$SCRATCH/t1-state` bleibt leer. Bei einem anderen Exit: anhalten, vorlegen.

- [ ] **Step 4: Akte schreiben**

Abschnitt „4. Vor PR B“ mit drei Unterabschnitten (Registry, Bereiche, `reconcile`). Je Aussage der Befehl und die Ausgabe (Tabelle der Bereiche aus Step 2, ohne private Pfadteile, die nicht schon in der Akte stehen).

- [ ] **Step 5: Commit**

```bash
git add docs/.superpowers/parity/stufe-4f.md
git commit -F "$SCRATCH/msg-t1.txt"
```
Nachricht: `docs: record that no registered area depends on an old manifest`.

---

### Task 2: `area check` und die Erkennung alter Manifeste fallen weg — Klasse (a)

**Files:**
- Delete: `internal/cli/areacheck.go`, `internal/cli/areacheck_test.go`
- Modify: `internal/cli/area.go` (Nutzungszeile `:22`, Zweig `:56`, Doku-Kommentar `:53`)
- Modify: `internal/config/areadeclaration.go` (ganz, siehe Step 3)
- Modify: `internal/config/areadeclaration_test.go`
- Modify: `internal/config/manifest_test.go:477` (Kommentar „`area check` names the key as ignored“: der Satzteil fällt)
- Probe, dann Delete (je der eine Test): `internal/brain/apply/resolve_test.go:123-135`, `internal/brain/check/house/undeclared_test.go:28ff`, `internal/brain/check/run/undeclared_test.go:29-41`, `internal/brain/convert/undeclared_test.go:27ff`, `internal/brain/maintenance/undeclared_test.go:49ff`, `internal/brain/privacy/areas_test.go:272ff`, `internal/brain/wiki/undeclared_test.go:26ff`, `internal/cli/brain_test.go:492-505`, `internal/cli/lintsweep_test.go:57ff`, `internal/cli/lintsweep_undeclared_test.go:29ff`
- Docs: `docs/{en,de}/cli-reference.md` (Überschrift „Upkeep/Pflege“ `en:681`/`de:704` ohne `area check`; Abschnitt `#### loomux area check` `en:725-732`/`de:748-755` fällt), `docs/{en,de}/getting-started.md:204-208`/`:206-209`, `README.md:248`, `README.de.md:249`, `docs/wiki/topics/datenmodell-und-bereiche.md:48-57`, nach E4 die Roadmap-Zeile `README.md:180`/`README.de.md:180`

**Interfaces:**
- Produces: `config.ReadAreaDeclaration(dir string) (*Manifest, error)` gibt für ein Verzeichnis ohne reguläre `.loomux/config.toml` immer `fmt.Errorf("%s: %w (%s)", dir, ErrNoManifest, name)`. `config.IsUndeclared(err error) bool` ist `errors.Is(err, ErrNoManifest) || errors.Is(err, ErrNoArea)`. `ErrOldManifest`, `oldManifestError` und `OldManifestNames` gibt es nicht mehr.

- [ ] **Step 1: Die elf Tests auf die neue Erwartung umstellen (Probe, Teil 1: RED)**

In jedem der elf Tests aus der Liste oben wird die Erwartung „Meldung mit `an old manifest lies there`“ ersetzt durch „dieselbe Antwort wie für ein leeres Verzeichnis“. Konkret ruft der Test den Leser zweimal auf, einmal mit `.brain.toml` und einmal ohne jede Datei, und vergleicht die beiden Antworten (Fehlertext nach `strings.ReplaceAll(dir, …)`, Befunde, Manifest). Beispiel für `internal/brain/check/run/undeclared_test.go`:

```go
func TestAreaManifestTakesAnOldManifestAsNone(t *testing.T) {
	old := t.TempDir()
	if err := os.WriteFile(filepath.Join(old, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	lookup := config.ArtifactLookup{Primary: t.TempDir()}
	m1, f1 := areaManifest(config.Area{Scope: "project/p", Path: old}, lookup)
	m2, f2 := areaManifest(config.Area{Scope: "project/p", Path: empty}, lookup)
	if m1 != nil || m2 != nil || len(f1) != len(f2) {
		t.Fatalf("old %+v %+v, empty %+v %+v", m1, f1, m2, f2)
	}
}
```

`internal/brain/apply/resolve_test.go:120-136` läuft über `config.OldManifestNames()`, das Step 3 löscht. Darum bekommt der Test in diesem Step die Namen als Literal: `for _, name := range []string{filepath.Join(".ultra-brain", "config.toml"), ".brain.toml"}`. Sonst baut Step 4 nicht.

In `internal/config/areadeclaration_test.go` werden `TestReadAreaDeclarationNamesTheOldManifest…` (Zeilen 88–112) und die Zeile `{oldManifestError{…}, false}` in `TestIsUndeclaredTakesExactlyTheTwoAbsences` (`:176`) entsprechend umgestellt: `.brain.toml` allein ergibt genau `dir + ": no manifest found (.loomux/config.toml)"`, und `IsUndeclared` ist `true`.

Run: `go test ./internal/config/ ./internal/brain/... ./internal/cli/ -run 'Undeclared|OldManifest|Old' > "$SCRATCH/t2-red.txt" 2>&1; grep -E -- '--- FAIL|^ok|^FAIL' "$SCRATCH/t2-red.txt"`
Expected: Jeder der elf Tests ist `--- FAIL`, weil der heutige Code den Hinweis gibt.

- [ ] **Step 2: Befehl und Zweig entfernen**

`internal/cli/areacheck.go` und `areacheck_test.go` löschen. In `internal/cli/area.go` den Zweig `if len(args) > 0 && args[0] == "check" { … }` (`:56-58`) entfernen. `areaUsage` endet danach mit `"[--merge-branch B] [--privacy M] [--no-reindex] [-y|--yes]"`, ohne die zweite Zeile. Der Doku-Kommentar von `areaCommand` lautet „It has one subcommand, `add`; …“. `area check x` fällt danach in den vorhandenen Zweig `args[0] != "add"`: Nutzung auf stderr, Exit 2. Der Test in `internal/cli/area_test.go` (bzw. der Datei, in der die Nutzungstests von `area` stehen):

```go
func TestAreaCheckIsGone(t *testing.T) {
	code, out, errOut := run("area", "check", t.TempDir())
	if code != 2 || out != "" || !strings.HasPrefix(errOut, "usage: loomux area add") || strings.Contains(errOut, "check") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}
```
RED: am Stand vor diesem Step gibt `area check <leeres Verzeichnis>` einen Bericht und nicht Exit 2.

- [ ] **Step 3: `areadeclaration.go` vereinfachen**

```go
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ReadAreaDeclaration reads the declaration of the area whose manifest lies in
// dir: `.loomux/config.toml`, checked whole by ReadDeclaration.
//
// Two answers are no defect of the file, and callers tell them apart: no
// regular file of that name is ErrNoManifest, a file without [area] -- policy
// only -- is ErrNoArea.
func ReadAreaDeclaration(dir string) (*Manifest, error) {
	name := filepath.Join(".loomux", "config.toml")
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
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

Danach `git grep -n "OldManifestNames\|ErrOldManifest\|oldManifestError"`: Kein Treffer darf bleiben. Kommentare an den zehn Aufrufern, die den Hinweis erwähnen (z. B. `internal/brain/check/house/federation.go:340`), werden mitgezogen.

- [ ] **Step 4: GREEN**

Run: `go test ./internal/config/ ./internal/brain/... ./internal/cli/ > "$SCRATCH/t2-green.txt" 2>&1; grep -E -- '--- FAIL|^ok|^FAIL' "$SCRATCH/t2-green.txt"`
Expected: alle `ok`.

- [ ] **Step 5: Probe in die Akte, dann die elf Tests löschen**

Die Ausgabe von `go test -v -run '<die elf Testnamen>'` über die betroffenen Pakete kommt nach `$SCRATCH/t2-probe.txt`. Ein Absatz in `parity/stufe-4f.md`, Abschnitt 4, hält fest: je Aufrufer der Testname, „wie ein leeres Verzeichnis“, grün. Danach werden die elf Tests gelöscht. Sie wiederholen nur noch den Fall „kein Manifest“, den die Nachbartests ohnehin halten. Mit ihnen fallen die Fixturen mit `.brain.toml` aus `areadeclaration_test.go` (`:43`, `:55`, `:75`, `:94-96`, `:127`). Wo ein solcher Test ein anderes Verhalten prüft (z. B. `:43`, kaputtes TOML), bekommt er `.loomux/config.toml` statt `.brain.toml`.

- [ ] **Step 6: Ende-zu-Ende über die CLI (Review Focus 2)**

Kein Shell-Lauf: Der Wächter verweigert `convert` und `area add` aus Bash, und eine echte Registry darf nicht entstehen. Stattdessen eine Wegwerf-Testdatei `internal/cli/zz_oldmanifest_probe_test.go`. Sie nutzt die vorhandenen Welt-Helfer aus `brain_test.go` (Registry im temporären `LOOMUX_STATE_DIR`, `writeFile`, `run`) und registriert zwei Bereiche: einen mit nur `.brain.toml`, einen leeren. Dann ruft sie je Befehl `run(...)` für beide auf: `brain status`, `brain catalog <scope>`, `lint --scope <scope>`, `reconcile`. `convert` ist über seinen Unit-Test aus Step 1 schon geprobt. Sie druckt Exit, stdout und stderr beider Läufe mit `t.Logf`.
Run: `go test ./internal/cli/ -run OldManifestProbe -v > "$SCRATCH/t2-e2e.txt" 2>&1`
Expected: Je Befehl sind beide Antworten gleich, bis auf Scope und Pfad. Die Tabelle kommt in die Akte, **die Testdatei wird danach gelöscht** und nicht committet.

- [ ] **Step 7: Mutanten**

Je per `go test -overlay` (Pfade in der Form `C:/…`), jeder Mutant muss bauen:
1. `IsUndeclared` ohne `|| errors.Is(err, ErrNoArea)`
2. `IsUndeclared` ohne `errors.Is(err, ErrNoManifest) ||`
3. `ReadAreaDeclaration` mit `err != nil &&` statt `err != nil ||`

Expected: Jeder wird rot; im Bericht stehen die tötende Testzeile und „gebaut: ja“.

- [ ] **Step 8: Doku**

Abschnitte und Zeilen aus der Dateiliste entfernen bzw. neu fassen. `getting-started` (en/de): Der Absatz über ein Altmanifest fällt ganz. Das Wiki (`datenmodell-und-bereiche.md:48-57`) nennt nur `.loomux/config.toml` und keinen alten Namen mehr. Nach E4 fällt die Roadmap-Zeile in beiden READMEs. Danach `git grep -n "area check" -- docs README.md README.de.md internal`: Es bleibt nur `CHANGELOG.md`.

- [ ] **Step 9: Commit**

```bash
git add -A internal/cli internal/config internal/brain docs README.md README.de.md
git status --short > "$SCRATCH/t2-status.txt"; cat "$SCRATCH/t2-status.txt"
git commit -F "$SCRATCH/msg-t2.txt" > "$SCRATCH/t2-commit.txt" 2>&1; tail -5 "$SCRATCH/t2-commit.txt"
```
Nachricht: `feat(config)!: read no old area manifest and drop loomux area check`, Rumpf: „A directory with only `.brain.toml` or `.ultra-brain/config.toml` now answers like one without a manifest. BREAKING CHANGE: `loomux area check` is gone.“ Vor dem `git add -A` prüfen, dass `git status` nur Dateien dieses Tasks zeigt.

---

### Task 3: `loomux dev switchover` fällt weg (#34)

**Files:**
- Delete: `internal/switchover/` (ganz), `internal/cli/switchover.go`, `internal/cli/switchover_test.go`
- Modify: `internal/cli/dev.go:71` (Eintrag `"switchover"`), Nutzungstext von `dev`, falls er den Befehl nennt
- Modify: `internal/hooks/guard.go:320` (Begründungstext), `:697-703` (Fall `"dev"` in der Befehlsregel)
- Modify: `internal/hooks/guard_test.go:833-834`, `:1008-1029`, `:1182-1199`, `:1239-1243`; `internal/hooks/guardholes_test.go:38`; `internal/hooks/guardstrict_test.go:33`
- Docs: `docs/{en,de}/cli-reference.md` (Wächterliste `en:258-269`/`de:268-277`, Abschnitt `### loomux dev switchover` `en:1130-1200`/`de:1164-1230` bis zum nächsten `###`), `docs/en/configuration.md:575-577`, `docs/de/configuration.md:591-593`, `README.md:332-333`, `README.de.md:339-340`, `docs/wiki/topics/schreibschranke.md:41`

- [ ] **Step 1: Vorher-Aufnahme des Wächters**

Eine Batterie per Write nach `$SCRATCH/t3-battery.txt`, eine Zeile je Befehl. Sie enthält alle Zeilen aus `guard_test.go:1008-1029` und `:1182-1199` und aus `guardstrict_test.go:33`, dazu `loomux dev bench hooks`, `loomux dev mutants ./internal/x`, `loomux init --yes`, `loomux config set a b`, `loomux area add x`, `loomux merge-hook install`. Ein kleines Go-Testprogramm im Scratchpad oder ein Overlay-Test liest jede Zeile, ruft `checkTool(root, "Bash", {"command": line}, policy)` einmal mit `config.Policy{}` und einmal mit `Strict: true` und schreibt `verweigert|durch` je Zeile nach `$SCRATCH/t3-before.txt`. Diese Aufnahme läuft am Stand vor Step 2.

- [ ] **Step 2: Code entfernen**

`internal/switchover/`, `internal/cli/switchover.go`, `switchover_test.go` löschen und den Eintrag in `devCommands` streichen. In `guard.go` fällt der Fall `case "dev":` samt Kommentar. Im Begründungstext `:320` fällt der Satzteil „, dev switchover prune-hooks removes hook entries from a settings file“. In den Tests fallen die Zeilen, die `prune-hooks` als verweigert erwarten, und der Test `:1239-1243` auf seine Begründung. Die Zeilen in `:1182-1199`, die durchgehen sollen, bleiben, weil sie weiter durchgehen. Ein `dev`-Befehl, der verweigert erwartet wird, wandert nicht in diese Liste: Er fällt.

- [ ] **Step 3: Tests grün**

Run: `go test ./internal/hooks/ ./internal/cli/ > "$SCRATCH/t3-test.txt" 2>&1; grep -E -- '--- FAIL|^ok|^FAIL' "$SCRATCH/t3-test.txt"`
Expected: `ok` für beide.

- [ ] **Step 4: Differenzprobe (Review Focus 5)**

Dieselbe Batterie am neuen Stand nach `$SCRATCH/t3-after.txt`, dann `diff "$SCRATCH/t3-before.txt" "$SCRATCH/t3-after.txt"`.
Expected: Es wechseln nur Zeilen, die `switchover` und `prune-hooks` enthalten, und nur von `verweigert` auf `durch`. Wechselt eine andere Zeile, oder eine Zeile von `durch` auf `verweigert`, ist das ein Befund: anhalten.
Den Diff in den Bericht übernehmen.

- [ ] **Step 5: Doku**

Abschnitte entfernen, Wächterlisten und den zitierten Begründungstext in beiden `cli-reference.md` wortgleich zu `guard.go:320` nachziehen. In `configuration.md` (en/de) fällt der Satz über das Umstellungsskript. Dass „ohne Datei jede Lane scharf“ ist, steht dort weiterhin, nur ohne das Skript. Danach `git grep -n -i "switchover" -- docs/en docs/de docs/wiki README.md README.de.md internal cmd`: Es bleiben nur Treffer in `migration.md` und `benchmarks.md`.

- [ ] **Step 6: Commit**

Nachricht: `feat(dev)!: drop dev switchover`, Rumpf: „The switch-over from the old tools is done everywhere; its render and prune-hooks subcommands and the guard rule for prune-hooks go with it. BREAKING CHANGE: `loomux dev switchover` is gone.“

---

### Task 4: `hook status` meldet keine abgelösten Hooks mehr — Klasse (b), (c)

**Files:**
- Modify: `internal/hooks/status.go` (`knownLegacyHooks` `:20-38`, `LegacyFinding` `:40-44`, der Teil von `auditSettings`, der Befunde sammelt, `:73-95`, der Berichtsteil `:172-195`, die Wiki-Zeilen `:136`, `:138`)
- Modify: `internal/hooks/status_test.go` (`:79-87`, `:151`, `:190-193`, `:272-274`, `:314-316` und jeder Test, der den Altlastenteil prüft)
- Docs: `git grep -n -i "legacy hook\|Redundant or obsolete\|UltraBrain Wiki\|superseded" -- docs README.md README.de.md` und jeden Treffer nachziehen

**Interfaces:**
- Produces: `auditSettings(root string) map[string]bool` (nur noch die installierten loomux-Ereignisse). Bleibt der Name passend, bleibt er; sonst `installedEvents`.

- [ ] **Step 1: Tests umstellen (RED)**

`status_test.go`: Die Erwartungen `UltraBrain Wiki: Inactive / Disabled` und `UltraBrain Wiki: Active` werden `Wiki: Inactive / Disabled` und `Wiki: Active`. Ein neuer Test hält Review Focus 3 fest: Er schreibt eine `settings.json` mit loomux-Einträgen für `PreToolUse` und `Stop` und daneben einem fremden Eintrag `{"command": "other-tool hook stop"}`. Erwartet werden die beiden loomux-Ereignisse im Bericht und die Abwesenheit von `legacy`, `superseded` und `other-tool` im Bericht.

Run: `go test ./internal/hooks/ -run Status > "$SCRATCH/t4-red.txt" 2>&1; grep -- '--- FAIL' "$SCRATCH/t4-red.txt"`
Expected: Die Wiki-Tests sind rot. Der neue Test ist rot, weil der heutige Bericht den Abschnitt `[OK] No obsolete or redundant legacy hooks found.` druckt.

- [ ] **Step 2: Code**

Tabelle, Typ, Befundsammlung und den Block `if len(findings) == 0 { … } else { … }` (`:186-193`) löschen. Die Überschrift `:169` wird `" Hook Audit (.claude/settings.json)"`. Der neue Test aus Step 1 prüft auch die Abwesenheit von `Redundancy`. Die beiden Wiki-Zeilen lauten:

```go
		fmt.Fprintf(stdout, "Wiki: Active (Bundle Directory: '%s')\n", wikiDir)
	} else {
		fmt.Fprintln(stdout, "Wiki: Inactive / Disabled (default)")
```

Tests, die nur die Tabelle prüften (`:79-87`, `:272-274`, `:314-316` samt ihren Funktionen), fallen.

- [ ] **Step 3: GREEN, Lint**

Run: `go test ./internal/hooks/ > "$SCRATCH/t4.txt" 2>&1; tail -3 "$SCRATCH/t4.txt"; go vet ./internal/hooks/`
Expected: `ok`, kein `vet`-Befund.

- [ ] **Step 4: Commit**

Nachricht: `feat(hooks): report no superseded hook entries in hook status`, Rumpf: „The wiki line reads `Wiki:`.“

---

### Task 5: Setup und Prüfstand erkennen keine Altinstallation mehr — (b), #34

**Files:**
- Modify: `internal/setup/gitfiles/gitfiles.go:90-100` (Kommentar und Markerliste)
- Modify: `internal/setup/gitfiles/gitfiles_test.go:152-153`
- Modify: `internal/setup/plan_test.go:160-175` (`TestAnExistingHookIsKeptAndNamed`)
- Modify: `internal/cases/gitworld.go:60` (`"/.ultraloom/"` aus `excluded`)
- Modify: `internal/cases/gitworld_test.go:71`, `:91`

- [ ] **Step 1: Tests umstellen (RED)**

`gitfiles_test.go`: Die Fälle `{"ulguard", …, true}` und `{"ultraloom", …, true}` werden `{"foreign", "#!/bin/sh\nother-guard --root .\n", false}` und `{"loomux", "#!/bin/sh\nloomux check precommit\n", true}`, falls der zweite nicht schon in der Tabelle steht. `plan_test.go`, Review Focus 1: `.githooks/pre-commit` bekommt `"#!/bin/sh\nother-guard check\n"`, und die erwartete Notiz wird `".githooks/pre-commit: kept; a hook of the project is already there"`. Daneben bleibt die Prüfung, dass der Hook nicht umgeschrieben wird (`changeOf(p, path)` ist `false`). Einen zweiten Fall mit `"#!/bin/sh\nsh ci/gate.sh\n"` und der Notiz `kept; it runs a gate already` dazunehmen, wenn es ihn noch nicht gibt.

Diese Tests sind schon am heutigen Code grün, weil `other-guard` nie ein Marker war. Einen roten Lauf ohne Altnamen gibt es für das Streichen eines Namens nicht. Kein Compile-Fehler und kein anderer Test wird als RED gemeldet. Der Nachweis ist ein einmaliger Overlay-Test, der nicht committet wird. Er prüft `RunsAGate("ulguard --root .")` und `RunsAGate("uv run ultraloom check precommit")`: vor Step 2 beide `true`, nach Step 2 beide `false`. Die beiden Ausgaben kommen in den Bericht.

- [ ] **Step 2: Code**

```go
// RunsAGate says whether an existing hook already runs a check chain: a
// loomux call or ci/gate.sh. Comment lines do not count, so a hook that
// merely mentions a chain is not taken for one.
func RunsAGate(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		for _, marker := range []string{"loomux", "ci/gate.sh"} {
```

`gitworld.go:60`: `"/.ultraloom/"` fällt aus `excluded`. Die Fixture in `gitworld_test.go:71`/`:91` nimmt `.loomux/state.json` (das bleibt ausgeschlossen) statt `.ultraloom/state.json`. Prüft der Test genau den Ausschluss, nennt er `.loomux/`.

- [ ] **Step 3: GREEN und Mutanten**

Run: `go test ./internal/setup/... ./internal/cases/ > "$SCRATCH/t5.txt" 2>&1; grep -E -- '--- FAIL|^ok|^FAIL' "$SCRATCH/t5.txt"`
Expected: alle `ok`.
Mutanten per Overlay: (1) `"ci/gate.sh"` aus der Markerliste; (2) `strings.HasPrefix(line, "#")` zu `false`; (3) `"/.loomux/"` aus `excluded`. Jeder wird rot.

- [ ] **Step 4: Wiedergabe der Fälle**

Run: `go test ./internal/cli/ -run 'TestCases|TestRecordedCases' > "$SCRATCH/t5-cases.txt" 2>&1; grep -E -- '--- FAIL|^ok|^FAIL' "$SCRATCH/t5-cases.txt"`
Expected: `ok`. Ein rotes `2c`/`2b` hieße, dass eine übersetzte Welt doch `.ultraloom/` trägt. Dann anhalten und vorlegen, nicht den Ausschluss zurücknehmen.

- [ ] **Step 5: Commit**

Nachricht: `feat(setup): take only loomux and ci/gate.sh calls as a gate in an existing hook`, Rumpf: „The case bench no longer excludes `.ultraloom/`; no translated world carries it.“

---

### Task 6: `AlwaysExcludes` ohne die alten Manifestnamen (#34)

**Files:**
- Modify: `internal/brain/index/walk.go:25-26`
- Modify: `internal/brain/index/walk_test.go:129-130`
- Modify (elf Dateien, je zwei Zeilenpaare): `testdata/cases/3a/reindex/{area-without-include,cases-opened,globs,missing-manifest,no-review-centre,nothing-open}/world_after/xdg/qmd/index.yml`, `testdata/cases/3b/approve/{amend,no-repo,rebase,success}/world_after/xdg/qmd/index.yml`

Die Liste ist per `git ls-files 'testdata/cases/3a/*' 'testdata/cases/3b/*' | grep index.yml | xargs grep -l "ultra-brain"` zu bestätigen. Sind es mehr oder weniger als elf, gilt die Liste aus dem Befehl, und die Zahl kommt in den Bericht.

Der Pin dieser Änderung ist die Wiedergabe von 3a und 3b, kein neuer Test mit Altnamen. `cases_3a_test.go:241-249` und `cases_3b_test.go:170ff` dekodieren jede `formatOnly`-Datei, darunter `xdg/qmd/index.yml`, und vergleichen den Inhalt mit `world_after`. Fehlen die zwei Muster nur auf einer Seite, wird der Fall rot.

- [ ] **Step 1: Code**

Die zwei Zeilen `"**/.brain.toml",` und `"**/.ultra-brain/**",` aus `AlwaysExcludes` streichen, ebenso die zwei Erwartungen in `walk_test.go:129-130`.

- [ ] **Step 2: RED über die Wiedergabe**

Run: `go test ./internal/cli/ -run 'TestCases3a|TestCases3b' > "$SCRATCH/t6-red.txt" 2>&1; grep -E -- '--- FAIL|differs in content' "$SCRATCH/t6-red.txt"`
Expected: Genau die Fälle mit einer `index.yml` aus der Liste sind rot, mit `xdg/qmd/index.yml differs in content`. Die Zahl der roten Fälle kommt in den Bericht und muss der Zahl der Dateien entsprechen.

- [ ] **Step 3: Die übersetzten `index.yml` anpassen**

Per Skript (Write nach `$SCRATCH/t6-yml.py`, mit `newline="\n"` beim Schreiben, ein `assert` je Datei, dass genau zwei Paare fallen):

```python
import pathlib, sys
for p in sys.argv[1:]:
    path = pathlib.Path(p)
    text = path.read_text(encoding="utf-8")
    drop = ["    - '**/.brain.toml'\n", "    - '**/.ultra-brain/**'\n"]
    assert all(text.count(d) == 2 for d in drop), (p, [text.count(d) for d in drop])
    for d in drop:
        text = text.replace(d, "")
    path.write_text(text, encoding="utf-8", newline="\n")
```

Vorher die Einrückung in einer Datei per `sed -n 14,18p` lesen und die Suchtexte im Skript daran ausrichten. Nach dem Lauf `git diff --stat testdata/cases`: genau die Dateien der Liste, je vier gelöschte Zeilen.

- [ ] **Step 4: GREEN, Wiedergabe**

Run: `go test ./internal/brain/index/ ./internal/cli/ > "$SCRATCH/t6.txt" 2>&1; grep -E -- '--- FAIL|^ok|^FAIL' "$SCRATCH/t6.txt"`
Expected: `ok` für beide. Die Suiten `TestCases3a` und `TestCases3b` laufen dabei mit.

- [ ] **Step 5: Commit**

Nachricht: `feat(index): stop excluding the old manifest names from every area`.

---

### Task 7: `NeighbourWiki` fällt weg (#32)

**Files:**
- Modify: `internal/detect/edges.go` (`:31-63` und `ofFamily` `:65-71`, falls ohne weiteren Aufrufer; die Imports `io/fs` und `strings`, falls verwaist)
- Modify: `internal/detect/edges_test.go` (alle `TestNeighbourWiki…`, `:49-120`, samt Helfern wie `closedFS`, falls verwaist)

- [ ] **Step 1: Aufrufer prüfen**

Run: `git grep -n "NeighbourWiki\|ofFamily\|neighbour_repo" -- '*.go'`
Expected: Treffer nur in `edges.go` und `edges_test.go`. Die Tests `internal/brain/wiki/root_test.go:97/106` heißen nur ähnlich und bleiben.

- [ ] **Step 2: Löschen, bauen, testen**

Run: `go vet ./internal/detect/ && go test ./internal/detect/ > "$SCRATCH/t7.txt" 2>&1; tail -2 "$SCRATCH/t7.txt"`
Expected: `ok`.

- [ ] **Step 3: Commit**

Nachricht: `refactor(detect): drop NeighbourWiki, which nothing calls`.

---

### Task 8: Ausgelieferte Vorlage ohne Altnamen (#34)

**Files:**
- Modify: `internal/setup/templates/files/skills/brain-review/SKILL.md:86`
- Test: `internal/setup/templates/templates_test.go:147` (die Liste `old` prüft schon `brain-mcp`, `ultraloom`, `ulguard`)

- [ ] **Step 1: RED**

In `templates_test.go:147` kommt `"ultra-brain"` zur Liste `old`.
Run: `go test ./internal/setup/templates/ > "$SCRATCH/t8-red.txt" 2>&1; grep -- '--- FAIL' "$SCRATCH/t8-red.txt"`
Expected: rot, mit `SKILL.md`.

- [ ] **Step 2: Vorlage**

`case: ultra-brain-2026-08-27-a4f2` wird `case: notes-2026-08-27-a4f2`. Die Form `<scope-slug>-<datum>-<hex>` bleibt.

- [ ] **Step 3: GREEN, Golden-Dateien**

Run: `go test ./internal/setup/... > "$SCRATCH/t8.txt" 2>&1; grep -E -- '--- FAIL|^ok' "$SCRATCH/t8.txt"`
Expected: `ok`. Hält eine Golden-Datei unter `internal/setup/testdata/` die alte Zeile fest, wird sie mitgezogen. Vorher `git grep -n "ultra-brain-2026" -- internal/setup` lesen.

- [ ] **Step 4: Commit**

Nachricht: `docs(setup): use a neutral case id in the brain-review skill`.

---

### Task 9: Kommentare, Paketdoku und Fixturkommentar — Klasse (c)

**Files:** alle Zeilen aus Anhang A, Gruppe „Task 9“, dazu die Kommentare in `internal/hooks/worktree.go` (`:78`, `:282-283`, `:414-416`, `:496`) und `internal/setup/testdata/loomux/loomux-config.toml:58-59`.

Regeln, in dieser Reihenfolge angewandt:
1. **Reine Herkunft fällt.** „`frontmatter.go is moved from ultra-brain's pkg/maintenance/frontmatter.go.`“: Der Satz fällt. Steht danach etwas über das Original, wird es „The reference …“.
2. **Ein Verweis auf das Verhalten der Python-Referenz** wird „the reference“: „`ultra-brain's pkg/guard/guard.go:293`“ wird „the reference's write barrier“. Zeilennummern in fremden Repos fallen, weil der Leser sie nicht mehr öffnen kann.
3. **Ein Messbeleg mit Altnamen** („Measured against ultra-brain's real registration …“) wird ohne den Namen gefasst („Measured against a real registration …“). Datum und Zahl bleiben.
4. **Paketdoku** `internal/gitenv/gitenv.go:1-6`: „Package gitenv keeps git's own environment out of loomux's git calls.“ Der Absatz über zwei Programme fällt, ebenso der Satz über ultra-brain (`:6`). `:32` und `:48` werden nach Regel 1–3 gefasst.
5. **Worktree-Kommentare:** `.ultraloom/vendor` wird `.tools/godot` (ein Pfad, den die Tests schon spiegeln), „the pinned runtime every other hook needs“ wird „a mirrored directory the main checkout needs“.
6. **Fixturkommentar** `loomux-config.toml:58-59`:
   ```toml
   # Laden kompiliert; was nicht kompiliert, ist ein Fehler, damit eine Regel
   # nicht still ausfällt.
   ```
7. **Was den Sinn verliert, wenn der Name fällt, fällt ganz.** Eine Zeile, die nur sagt, dass ein Altwerkzeug etwas anders machte, ohne dass der Code darauf baut, fällt.

Ausgenommen sind die Toleranzlisten der Wiedergabetests (Restbestand). Die Doku-Kommentare über den Replay-Tests (`cases_1b1_test.go:16`, `cases_2a_test.go:46` usw.) werden nach Regel 2 gefasst, z. B. „TestCases2a replays the recordings of the reference's `check` against loomux“.

- [ ] **Step 1: Gruppe nach Paket abarbeiten**

Je Paket die Zeilen aus Anhang A öffnen, nach den Regeln fassen und `go vet ./<paket>/` fahren. Nach jeder Regel 1 oder 7 den Absatz darüber und darunter lesen, damit kein Satz ohne Bezug stehen bleibt („The original is …“ nach gestrichenem „moved from“).

- [ ] **Step 2: Rest-Grep**

Run: `git grep -n -i -E '<Suchliste>' -- 'internal/*.go' ':!internal/dev/importcases' ':!internal/dev/recordcase' ':!*_test.go'`
Expected: kein Treffer.

Run: `git grep -n -i -E '<Suchliste>' -- 'internal/*_test.go' ':!internal/dev/importcases' ':!internal/dev/recordcase' ':!internal/cli/dev_test.go' ':!internal/cli/devmcp_test.go' | grep -E ':\s*//'`
Expected: nur Kommentarzeilen über den Toleranzlisten der Wiedergabetests, die auf eine Aufnahme zeigen. Jede steht im Bericht.

- [ ] **Step 3: Tor**

Run: `sh ci/gate.sh > "$SCRATCH/t9-gate.txt" 2>&1; tail -15 "$SCRATCH/t9-gate.txt"`
Expected: grün.

- [ ] **Step 4: Commit**

Nachricht: `docs: name no predecessor project in comments and package docs`.

---

### Task 10: Testeingaben mit neutralen Namen — Klasse (d)

**Files:** alle Zeilen aus Anhang A, Gruppe „Task 10“, dazu `internal/hooks/worktree_test.go` und `internal/worktree/mirror/mirrorcfg_test.go`.

Ersetzungen, je Datei einheitlich:

| alt | neu | Grund |
|---|---|---|
| `project/ultra-brain` | `project/side-notes` | Bindestrich bleibt (Sammlungsname `project-side-notes`), Sortierung bleibt zwischen `knowledge` und `space` |
| `ultra-brain.md` (Wegweiser) | `side-notes.md` | folgt dem Scope |
| `ulguard` | `other-guard` | fremder Hook, kein Marker |
| `brain guard` | `notes-guard` | fremder Hook |
| `uv run ultraloom hook …`, `.ultraloom/vendor/ultraloom` | `uv run other-hooks hook …`, `.tools/vendor/other-hooks` | fremder Hook |
| `.loomux/vendor/ultraloom`, `.ultraloom/vendor` | `.loomux/vendor/tool`, `.tools/vendor` | gespiegelter Pfad |
| `ULTRALOOM_GITENV_PROBE` | `LOOMUX_GITENV_PROBE` | |
| `ulinit` (Variable in `merge_test.go:572`) | `olderMatcher` | |
| `case: ultra-brain-2026-08-27-a4f2` (`internal/brain/evidence/testdata/proposal-*.md`) | `case: notes-2026-08-27-a4f2` | wie Task 8 |
| `ultraloom`/`ultra-brain` in der Herkunftsspalte von `plancheck_test.go:43,45` | `alpha`/`beta` | nur geparst |

**Löschen statt umbenennen:** Tests, die nur zeigen, dass ein Altname gewöhnlich ist. Das betrifft `internal/brain/guard/decide_test.go:489-525` (`TestTheOldManifestNamesAreOrdinaryFiles`) und den Kommentar `:528-536`, `internal/config/manifest_test.go:199-205`, `internal/detect/detect_test.go:384-396` und den `.brain.toml`-Eintrag in `internal/lock/replacedir_test.go:19`. Bei diesem letzten wird nur der Eintrag ersetzt, durch eine beliebige zweite Datei, wenn der Test zwei Dateien braucht. Vorher je Test lesen, ob er noch etwas anderes prüft. Wenn ja, bleibt er mit neutralem Namen.

- [ ] **Step 1: Ersetzen, je Paket testen**

Nach jedem Paket: `go test ./<paket>/ > "$SCRATCH/t10-<paket>.txt" 2>&1; tail -2 …`. Ein roter Test nach einer reinen Umbenennung zeigt eine Abhängigkeit vom Namen, etwa Sortierung oder Länge. Sie wird im Bericht genannt und mit einem passenden neutralen Namen gelöst, nicht mit einer geänderten Erwartung.

- [ ] **Step 2: Rest-Grep**

Run: `git grep -n -i -E '<Suchliste>' -- 'internal/*_test.go' 'internal/**/testdata/**' ':!internal/dev/importcases' ':!internal/dev/recordcase' ':!internal/cli/dev_test.go' ':!internal/cli/devmcp_test.go' ':!internal/brain/apply/testdata/frontmatter'`
Expected: nur Toleranzzeilen der Wiedergabetests und die zwei Verbotslisten in `templates_test.go` (Restbestand). Jede steht im Bericht.

- [ ] **Step 3: Volle Suite**

Run: `go test ./... > "$SCRATCH/t10-all.txt" 2>&1; grep -E -- '--- FAIL|^FAIL' "$SCRATCH/t10-all.txt"`
Expected: leer.

- [ ] **Step 4: Commit**

Nachricht: `test: give test inputs neutral names`.

---

### Task 11: Doku, Roadmap, Migrationsplan — Klassen (f), (g), (i), #32

**Files:**
- `AGENTS.md:3-5`
- `LICENSE.md:5`
- `README.md`, `README.de.md`: `:143-144` (Absatz zum Migrationsplan), `:392`/`:399` (Linktext der Fusions-Spec), Roadmap-Zeilen „Register revisions only from review“/„Registerrevisionen nur aus der Prüfung“, „Brain web app“/„Brain-Web-App“ (W2), „Flow runtime“/„Flow-Laufzeit“
- `docs/{en,de}/cli-reference.md` (übrige Zeilen aus Anhang A, Gruppe „Task 11“), `docs/{en,de}/hooks.md` (`en:422-431`, `de:445-455`)
- `docs/wiki/**`: die Prosa-Zeilen aus Anhang A, Gruppe „Task 11“; `sources:`-Zeilen und `log.md` bleiben (E2, E3)
- `docs/{en,de}/migration.md`: Zeile **4f**, Spalte „Stand“
- `.loomux/config.toml:58-59`: **Mensch**

- [ ] **Step 1: `AGENTS.md` und `LICENSE.md`**

`AGENTS.md`, Kopf:
```markdown
# loomux

One Go binary for the hook path, the check chain and the knowledge system of
a project. Design: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`.
```
`LICENSE.md:5`: `Required Notice: Copyright 2026 Christoph Wübbels (https://github.com/xidus90/loomux)`.

- [ ] **Step 2: Referenzdoku en/de**

Regeln wie Task 9. Beispiele für `cli-reference.md`:
- „answer as ultra-brain's `brain-mcp` does“ → „answer as the reference does“
- „Four commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 3a“ → „Four top-level commands since stage 3a“
- „It replaces ultraloom's `ulinit`, …“ → Der Satz fällt.

`hooks.md` (en/de): „a loomux command that replaced ulinit's under …“ wird „a loomux command under an older matcher …“, und „an entry that still runs `ulguard` is not ours“ wird „an entry of another tool is not ours“. Beide Sprachen bekommen denselben Inhalt, die Dateinamen bleiben gleich.

- [ ] **Step 3: Wiki-Prosa**

Je Seite aus Anhang A: Prosa ohne Altnamen („Die sechs Grundsätze aus ultra-brain“ → „Die sechs Grundsätze des Wissenssystems“). `description:` in der Frontmatter zählt als Prosa. `topics/index.md:6` wird von Hand an die neue Beschreibung von `architektur-grundsaetze.md` angeglichen. Kein `reindex` auf das echte Projekt: Der Bericht vermerkt, dass der nächste `reindex` eines Menschen die Zeile so wiedergeben soll. `sources:`-Zeilen und `log.md` bleiben unverändert. Danach `go run ./cmd/loomux lint --scope project/loomux > "$SCRATCH/t11-lint.txt" 2>&1`: keine neuen Befunde gegenüber dem Lauf vor diesem Step (vorher aufnehmen).

- [ ] **Step 4: README**

- `:143-144` (en): „Where each migration stage and each capability carried over from the predecessor projects stands — …“. Der Absatz in `README.de.md:143` bekommt denselben Inhalt.
- `:392`/`:399`: Der Linktext wird „Fusion design“/„Fusions-Design“.
- **Register revisions only from review** bekommt ans Ende der Beschreibung die Regel aus #32 als Teil dessen, was die Zeile baut: „; reconcile moves the hash of a register row that no page cites forward without raising its revision“. Die deutsche Zeile bekommt denselben Satz: „; `reconcile` schiebt den Hash einer Registerzeile, die keine Seite zitiert, vor, ohne ihre Revision zu erhöhen“. Es ist eine Zusage der Zeile und kein Befund über den heutigen Code.
- **Brain web app (W2)** bekommt in der Beschreibung: „, the cross-area edges (`edges-cross.json`) and in the browser a page reader, full-text search and the review centre“.
- **Flow runtime** bekommt: „Flow B also takes a diagnosis run without a model (`run --no-model`)“.

- [ ] **Step 5: Migrationsplan**

Zeile 4f (en), Ende der Spalte „Stand“: „Left: PR B, PR C and the restart“ wird „PR B has removed the references from code and docs: `area check` and `dev switchover` are gone, as is the detection of old installations. Left: PR C and the restart“. Die deutsche Zeile bekommt denselben Inhalt. Status bleibt `open`.
Run: `go test ./internal/plancheck/ > "$SCRATCH/t11-plan.txt" 2>&1; tail -2 "$SCRATCH/t11-plan.txt"`
Expected: `ok`.

- [ ] **Step 6: `.loomux/config.toml` — dem Menschen nennen**

Der Agent schreibt die Datei nicht. Im Bericht steht für den Menschen: Zeilen 58–59 von `.loomux/config.toml` ersetzen durch
```toml
# Laden kompiliert; was nicht kompiliert, ist ein Fehler, damit eine Regel
# nicht still ausfällt.
```
und dann ohne `--no-verify` committen. Dieser Commit steht vor dem Push; Task 12 prüft ihn.

- [ ] **Step 7: Commits**

Drei Commits:
- `docs: describe loomux without its predecessor projects` (AGENTS, LICENSE, Referenzdoku, Wiki, README-Absätze)
- `docs(roadmap): add the leftovers of the reference repositories to their rows`
- `docs(migration): note that code and docs no longer name the predecessors`

---

### Task 12: Abschluss

- [ ] **Step 1: Rest-Grep gegen den Restbestand**

Run: `git grep -n -i -E '<Suchliste>' > "$SCRATCH/t12-grep.txt"; cut -d: -f1 "$SCRATCH/t12-grep.txt" | sort -u > "$SCRATCH/t12-files.txt"; wc -l "$SCRATCH/t12-files.txt"`
Jede Datei wird einer Zeile der Tabelle „Restbestand nach PR B“ zugeordnet (Skript per Write: Pfadmuster der Tabelle gegen die Liste). Expected: keine Datei ohne Zuordnung. Bei `docs/wiki/**` muss jede Trefferzeile eine `sources:`-Zeile (`resource: brain://project/loomux/docs/.superpowers/…`) oder aus `log.md` sein. Bei `README*.md` muss sie aus der W2-Zeile oder einer `ulflow`-Zeile sein.

- [ ] **Step 2: `.loomux/config.toml`**

Run: `git grep -n "ultraloom" -- .loomux/config.toml`
Expected: kein Treffer. Steht der Treffer noch da, hat der Mensch Task 11, Step 6 noch nicht ausgeführt: dem Nutzer nennen und warten.

- [ ] **Step 3: Tor und volle Suite auf dem Endstand**

Run: `sh ci/gate.sh > "$SCRATCH/t12-gate.txt" 2>&1; tail -20 "$SCRATCH/t12-gate.txt"`
Expected: grün, Coverage 100 % je Funktion.

- [ ] **Step 4: Binary neu bauen**

```bash
go build -o bin/loomux.new.exe ./cmd/loomux
go run ./cmd/loomux dev swap-binary --dir bin
```
Dann `bin/loomux.exe area check .`: Expected Nutzung von `area add` auf stderr, Exit 2. `bin/loomux.exe dev switchover`: Expected Exit 2 und die Meldung, die `devCommand` für einen unbekannten Unterbefehl gibt (`internal/cli/dev.go`, hinter `devCommands[args[0]]`); ihren Wortlaut dort nachlesen.

- [ ] **Step 5: PR-Rumpf vorbereiten (für `release-pr`)**

Per Write nach `$SCRATCH/pr-body.md`:

```markdown
Release: major — `loomux area check` and `loomux dev switchover` are removed.

## Changelog

### Removed
- `loomux area check`: the switch-over from the old area manifests is done.
- `loomux dev switchover render` and `prune-hooks`, together with the guard rule for `prune-hooks`.
- `loomux hook status` no longer lists superseded hook entries of the old tools.

### Changed
- A directory with only `.brain.toml` or `.ultra-brain/config.toml` is answered like one without a manifest; the hint that pointed to `area check` is gone.
- The index no longer excludes `**/.brain.toml` and `**/.ultra-brain/**` in every area.
- `loomux init` takes only a `loomux` or `ci/gate.sh` call in an existing pre-commit hook as a gate; any other hook is still kept and named.
- `loomux hook status` prints `Wiki:` for the wiki line.
- The `brain-review` skill template uses a neutral case id.
```

Label `release:major`. Den Push nennt `release-pr`, ein Mensch führt ihn aus.

- [ ] **Step 6: Commits gruppieren**

`release-pr` aufrufen. Vor dem Falten je Fixup `git show --stat` gegen die Dateien des Zielcommits halten. Nach dem Falten: `git diff <alter HEAD> HEAD` muss leer sein, und `git rebase <basis> --exec "go vet ./..."` muss grün sein.

---

## Anhang A: Inventur der Treffer, nach Task geordnet

Erzeugt am 2026-10-05 an `6e64a7e7` mit
`git grep -n -i -E '<Suchliste>' -- . ':!testdata/cases' ':!docs/.superpowers' ':!docs/*/benchmarks.md' ':!docs/*/migration.md' ':!testdata/bench' ':!internal/dev/importcases' ':!internal/dev/recordcase' ':!internal/switchover' ':!internal/cli/switchover*' ':!internal/cli/areacheck*'`.
Zeilen sind auf 260 Zeichen gekürzt. Die Zeilennummern gelten an der Basis; nach jedem Task verschieben sie sich, darum vor jedem Task neu greppen und den Anhang nur als Liste der Dateien und Stellen lesen. „Rest“ ist der Restbestand.

### Task 2 (30 Zeilen)

```text
README.de.md:180:| **Der Hinweis neben einer Deklaration ohne `[area]`** | Eine `.loomux/config.toml` ohne `[area]` neben einer alten `.brain.toml` oder `.ultra-brain/config.toml` bekommt `ErrNoArea` ohne Verweis auf `loomux area check`; die im Altmanifest geb
README.md:180:| **The hint beside a declaration without `[area]`** | A `.loomux/config.toml` without `[area]` next to an old `.brain.toml` or `.ultra-brain/config.toml` is answered with `ErrNoArea` and no pointer to `loomux area check`, so the declaration left
docs/de/cli-reference.md:749:Ein Bericht über die Bereichsmanifeste eines Verzeichnisses, für den, der den Inhalt einer alten `.ultra-brain/config.toml` oder `.brain.toml` nach `.loomux/config.toml` überträgt: bei einer Umstellung der Agent, der die neue K
docs/de/cli-reference.md:751:- **Dateien**: `.loomux/config.toml`, `.ultra-brain/config.toml` und `.brain.toml`, in der Reihenfolge, in der der Bericht sie probiert; ein Name, der keine reguläre Datei ist, wird übergangen.
docs/de/getting-started.md:206:  nur noch `.ultra-brain/config.toml` oder `.brain.toml` trägt, wird abgelehnt,
docs/en/cli-reference.md:726:A report on the area manifests of one directory, for whoever carries the content of an old `.ultra-brain/config.toml` or `.brain.toml` over into `.loomux/config.toml`: in a switch-over the agent that prepares the new configuration,
docs/en/cli-reference.md:728:- **Files**: `.loomux/config.toml`, `.ultra-brain/config.toml` and `.brain.toml`, in the order the report tries them; a name that is not a regular file is skipped.
docs/en/getting-started.md:204:  carries only `.ultra-brain/config.toml` or `.brain.toml` is refused, and the
internal/brain/apply/resolve_test.go:27:// where Python's fixture writes `.brain.toml`.
internal/brain/check/house/undeclared_test.go:32:	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n"), 0o644); err != nil {
internal/brain/check/run/undeclared_test.go:33:	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n"), 0o644); err != nil {
internal/brain/convert/undeclared_test.go:32:	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n\n[privacy]\nmode = \"local_only\"\n"), 0o644); err != nil {
internal/brain/maintenance/undeclared_test.go:53:	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n"), 0o644); err != nil {
internal/brain/privacy/areas_test.go:280:	writeFile(t, filepath.Join(ws, ".brain.toml"), "[area]\nscope = \"project/ws\"\n")
internal/brain/wiki/undeclared_test.go:30:	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"p\"\n"), 0o644); err != nil {
internal/cli/brain_test.go:275:// ultra-brain's state directory is read no more, whatever
internal/cli/brain_test.go:500:	writeFile(t, filepath.Join(w.area, ".brain.toml"), "[area]\nscope = \"project/a\"\n")
internal/cli/brain_test.go:502:	want := "an old manifest lies there (.brain.toml): `loomux area check " + filepath.ToSlash(w.area) + "` shows what to carry over"
internal/cli/lintsweep_test.go:62:	writeFile(t, filepath.Join(oldWiki, "..", ".brain.toml"), "[area]\nscope = \"project/old\"\n")
internal/cli/lintsweep_undeclared_test.go:33:	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n"), 0o644); err != nil {
internal/config/areadeclaration.go:10:// OldManifestNames are ultra-brain's two names for an area declaration, in
internal/config/areadeclaration.go:15:	return []string{filepath.Join(".ultra-brain", "config.toml"), ".brain.toml"}
internal/config/areadeclaration_test.go:43:	declareIn(t, dir, ".brain.toml", "[area\n")
internal/config/areadeclaration_test.go:55:	declareIn(t, dir, ".brain.toml", "[area]\nscope = \"old\"\n")
internal/config/areadeclaration_test.go:75:	for _, name := range []string{filepath.Join(".loomux", "config.toml"), ".brain.toml"} {
internal/config/areadeclaration_test.go:88:// one ultra-brain asked first is named.
internal/config/areadeclaration_test.go:94:		{[]string{".brain.toml"}, ".brain.toml"},
internal/config/areadeclaration_test.go:95:		{[]string{filepath.Join(".ultra-brain", "config.toml")}, filepath.Join(".ultra-brain", "config.toml")},
internal/config/areadeclaration_test.go:96:		{[]string{".brain.toml", filepath.Join(".ultra-brain", "config.toml")}, filepath.Join(".ultra-brain", "config.toml")},
internal/config/areadeclaration_test.go:127:	declareIn(t, dir, ".brain.toml", "[area]\nscope = \"old\"\n")
```

### Task 3 (2 Zeilen)

```text
docs/de/cli-reference.md:1165:Die zwei Stücke Code der Umstellung eines Projekts von den alten Werkzeugen (`ultraloom`, `ultra-brain`) auf loomux: ein Agent bereitet vor, ein Mensch führt aus. `loomux dev switchover` allein zeigt die Unterbefehle und endet m
docs/en/cli-reference.md:1131:The two pieces of code of the switch-over of a project from the old tools (`ultraloom`, `ultra-brain`) to loomux: an agent prepares, a human runs. `loomux dev switchover` alone prints the subcommands and exits `2`; an unknown subc
```

### Task 4 (21 Zeilen)

```text
internal/hooks/status.go:24:	{"ulguard", "superseded by 'loomux hook pre-tool-use' and 'loomux hook post-tool-use'"},
internal/hooks/status.go:25:	{"brain guard", "merged into 'loomux hook pre-tool-use'"},
internal/hooks/status.go:30:	// The whole command line, not just the event: `ultraloom hook stop` as a
internal/hooks/status.go:33:	{"ultraloom hook stop", "superseded by 'loomux hook stop'"},
internal/hooks/status.go:34:	{"ultraloom hook subagent-start", "superseded by 'loomux hook subagent-start'"},
internal/hooks/status.go:35:	{"ultraloom hook subagent-stop", "superseded by 'loomux hook subagent-stop'"},
internal/hooks/status.go:36:	{"generate_index.py", "superseded by ultra-brain catalog/reindex and wiki-gate"},
internal/hooks/status.go:136:		fmt.Fprintf(stdout, "UltraBrain Wiki: Active (Bundle Directory: '%s')\n", wikiDir)
internal/hooks/status.go:138:		fmt.Fprintln(stdout, "UltraBrain Wiki: Inactive / Disabled (default)")
internal/hooks/status_test.go:79:	// and that is a loomux subcommand. `ulguard` is itself listed as superseded
internal/hooks/status_test.go:81:	// binary this very report calls gone. The fixture configures no `ulguard`
internal/hooks/status_test.go:86:	if strings.Contains(out, "ulguard") {
internal/hooks/status_test.go:87:		t.Fatalf("no reason may name ulguard as a successor, got %s", out)
internal/hooks/status_test.go:151:		"UltraBrain Wiki: Inactive / Disabled",
internal/hooks/status_test.go:190:	if strings.Contains(out, "UltraBrain Wiki: Inactive") {
internal/hooks/status_test.go:193:	for _, want := range []string{"UltraBrain Wiki: Active", "'notes'", "*.md (in notes)"} {
internal/hooks/status_test.go:272:			"Stop": [{"hooks": [{"command": "ultraloom hook stop"}]}],
internal/hooks/status_test.go:273:			"SubagentStart": [{"hooks": [{"command": "ultraloom hook subagent-start"}]}],
internal/hooks/status_test.go:274:			"SubagentStop": [{"hooks": [{"command": "ultraloom hook subagent-stop"}]}]
internal/hooks/status_test.go:314:			"SubagentStop": [{"hooks": [{"command": "ultraloom hook subagent-stop"}]}],
internal/hooks/status_test.go:316:			"Stop": [{"hooks": [{"command": "ultraloom hook stop"}]}]
```

### Task 5 (8 Zeilen)

```text
internal/cases/gitworld.go:60:var excluded = []string{"/git.toml", "/faketool.json", "/.origin.git/", "/.ultraloom/", "/.loomux/", "/.claude/"}
internal/cases/gitworld_test.go:71:		".ultraloom/state.json": `{"base": "{{COMMIT:1}}", "head": "{{COMMIT:2}}"}`,
internal/cases/gitworld_test.go:91:	state, _ := os.ReadFile(filepath.Join(dir, ".ultraloom", "state.json"))
internal/setup/gitfiles/gitfiles.go:91:// loomux, ultraloom or ulguard call, or ci/gate.sh. Comment lines do not
internal/setup/gitfiles/gitfiles.go:99:		for _, marker := range []string{"loomux", "ultraloom", "ulguard", "ci/gate.sh"} {
internal/setup/gitfiles/gitfiles_test.go:152:		{"ulguard", "#!/bin/sh\nulguard --root .\n", true},
internal/setup/gitfiles/gitfiles_test.go:153:		{"ultraloom", "#!/bin/sh\nuv run ultraloom check precommit\n", true},
internal/setup/plan_test.go:164:		".githooks/pre-commit": "#!/bin/sh\nulguard check\n",
```

### Task 6 (4 Zeilen)

```text
internal/brain/index/walk.go:25:	"**/.brain.toml",
internal/brain/index/walk.go:26:	"**/.ultra-brain/**",
internal/brain/index/walk_test.go:129:		".brain.toml",               // ALWAYS_EXCLUDES
internal/brain/index/walk_test.go:130:		".ultra-brain/config.toml",  // ALWAYS_EXCLUDES
```

### Task 7 (3 Zeilen)

```text
internal/detect/edges.go:37:// ultraloom, which merely stands in the same parent directory, had iam_wiki
internal/detect/edges_test.go:107:// Measured on 2026-09-10: ultraloom, which only stands beside iam_wiki in the
internal/detect/edges_test.go:112:	for _, project := range []string{"ultraloom", "space", "iamx", "iamx_backend"} {
```

### Task 8 (1 Zeilen)

```text
internal/setup/templates/files/skills/brain-review/SKILL.md:86:case: ultra-brain-2026-08-27-a4f2
```

### Task 9 (107 Zeilen)

```text
internal/brain/apply/frontmatter.go:3:// frontmatter.go is moved from ultra-brain's pkg/maintenance/frontmatter.go.
internal/brain/apply/patch.go:4:// patch.go is moved from ultra-brain's pkg/maintenance/patch.go. The original
internal/brain/apply/resolve.go:80:// nearest first. Python's marker is `.brain.toml`; loomux's is
internal/brain/catalog/python_test.go:16:// ultra-brain tag loomux-1a-source) on 2026-09-15.
internal/brain/check/house/bundle.go:254:// over ultra-brain's `docs/wiki`, not one of the 83 targets that reach this lookup is a
internal/brain/check/house/bundle_test.go:184:	// pages of ultra-brain's `docs/wiki`. "Keine andere Seite ... zeigt hierher" is the
internal/brain/check/house/bundle_test.go:449:	// than the pages inside them (`ultra-brain/docs/wiki/index.md`), and a directory is
internal/brain/check/house/federation.go:134:// slashes, so `brain://project/ultra-brain/...` splits into host
internal/brain/check/house/federation.go:135:// `project` and path `/ultra-brain/...`. Netloc plus path is everything
internal/brain/check/house/federation.go:348:// registration in ultra-brain before the repair: a broken `.brain.toml` at the vault
internal/brain/check/house/federation.go:349:// root reported `project/ultra-brain`, an area the signpost names
internal/brain/check/house/federation.go:397:// `project/ultra-brain` is looked for as `ultra-brain.md`.
internal/brain/check/house/federation_test.go:51:// (`ultra-brain/docs/.superpowers/specs/2026-08-24-wiki-verbund-design.md:167-173`).
internal/brain/check/house/federation_test.go:349:	// registry: `project/ultra-brain` keeps its wiki in its own
internal/brain/check/house/federation_test.go:675:	// Measured against ultra-brain's real registration before this fixture was
internal/brain/check/house/federation_test.go:676:	// written: a broken `.brain.toml` at the vault root reported
internal/brain/check/house/federation_test.go:677:	// `project/ultra-brain`, which the signpost names correctly.
internal/brain/check/house/page.go:72:// measured, not feared: `ultra-brain/docs/wiki/_schema.md:27` documents the conflict
internal/brain/check/house/page.go:117:// (`ultra-brain/docs/.superpowers/specs/2026-09-01-pruefkatalog-in-go-design.md:156`),
internal/brain/check/house/page.go:119:// (`ultra-brain/docs/.superpowers/specs/2026-08-23-scheibe-3-wiki-schicht-design.md:178`).
internal/brain/check/house/page_test.go:477:	// case is not hypothetical: `ultra-brain/docs/wiki/_schema.md:27` documents the
internal/brain/check/okf/soft.go:137:// ultra-brain's `docs/wiki` -- and 0 carry a single footnote marker. A rule
internal/brain/check/okf/soft_test.go:333:	// and `log.md` (`ultra-brain/docs/wiki/index.md:12-14`). A directory holding
internal/brain/check/run/run.go:430://     what the write barrier already does (ultra-brain's `pkg/guard/guard.go:293`),
internal/brain/check/run/run.go:449:// the barrier gives for reading it once (ultra-brain's `pkg/guard/guard.go:298-300`).
internal/brain/check/run/run.go:479:			// the barrier insists on at ultra-brain's `pkg/guard/guard.go:334`.
internal/brain/check/run/run_test.go:489:	// itself. ultra-brain's `pkg/guard/guard.go:293` walks the parents for the
internal/brain/check/run/run_test.go:615:	// insists on at ultra-brain's `pkg/guard/guard.go:334`. With no registration to
internal/brain/check/run/targets_test.go:11:// the shape the registry is in right now for `project/ultra-brain`, whose
internal/brain/evidence/boundaries_test.go:5:// `read_package` in src/brain/maintenance/evidence.py, ultra-brain 3cc72d2)
internal/brain/evidence/evidence.go:9:// Moved from ultra-brain's pkg/maintenance/evidence.go. The original is
internal/brain/evidence/reference_test.go:3:// The tests of ultra-brain's tests/maintenance/test_evidence.py that have no
internal/brain/guard/bench_test.go:10:// a registry of real size -- the ~40 ms `brain guard` spent above the Go
internal/brain/guard/decide_test.go:489:// loomux reads one manifest. ultra-brain's legacy `.brain.toml` and its
internal/brain/guard/decide_test.go:490:// `.ultra-brain/config.toml` are ordinary names here, and a barrier that
internal/brain/guard/decide_test.go:528:// The barrier used to pin `<bundle>/.ultra-brain/hooks/wiki_guard.py`: the
internal/brain/guard/decide_test.go:533:// `brain guard` and installs no file (src/brain/init.py: "A bare command,
internal/brain/guard/decide_test.go:535:// 6bceba4, and no `.ultra-brain/hooks` directory exists in any registered
internal/brain/guard/guard.go:44:// repository and to its virtual environment; `brain guard` is wired the
internal/brain/guard/manifest.go:9:// ultra-brain had a second, legacy spelling; loomux cut it on 2026-09-14.
internal/brain/guard/path.go:202:// from ultra-brain along with `anchor`.
internal/brain/index/qmdconfig.go:338:// PruneCollections removes collections created by ultra-brain that are no longer registered.
internal/brain/maintenance/reconcile.go:315:// This path is the ground of the write barrier's one exemption (`brain guard`,
internal/brain/privacy/channel.go:42:// ultra-brain's Go gate answered an absent declaration as visible and hid the
internal/brain/pytext/repr.go:2:// ultra-brain reference does: repr, splitlines, strip, read_text,
internal/brain/pytext/url_test.go:6:// for each input on 2026-09-23, run through `uv run python` in ultra-brain.
internal/brain/reader/python_test.go:17:// (brain.core._section and Path.read_text, Python 3.14.7, ultra-brain tag
internal/brain/wiki/root.go:14:// ultra-brain's FindWikiPath read [wiki] path and [area] wiki from .brain.toml;
internal/brain/wiki/scaffold_test.go:53:// src/brain/wiki/scaffold.py at ultra-brain 3cc72d2; the script is
internal/brain/wiki/sweep_test.go:91:	// testdata/sweep/python.golden is what `lint_bundle` of ultra-brain at
internal/cli/brain.go:19:// brainCommand is `loomux brain`: the five read commands of brain-mcp
internal/cli/brainargs_test.go:101:// ones brain-mcp reports from its top-level parser.
internal/cli/braincheck.go:18:// of the check run. Its reference is the Go binary of ultra-brain, not the
internal/cli/cases_1b1_test.go:16:// TestRecordedCasesOfStage1b1 replays the recordings of brain-mcp against
internal/cli/cases_2a_test.go:46:// TestCases2a replays the recordings of `ultraloom check` against loomux
internal/cli/cases_2b_test.go:26:// TestCases2b replays the recordings of `ultraloom commit-msg` against loomux
internal/cli/cases_2c_test.go:28:// TestCases2c replays the recordings of ultraloom's session hooks against
internal/cli/cases_3a_test.go:181:// TestCases3a replays the recordings of brain-mcp's reconcile, reindex, embed
internal/cli/cases_3b_test.go:98:// TestCases3b replays the recordings of brain-mcp's cases, case and approve
internal/cli/cases_4a2_test.go:77:// TestCases4a2 replays the recorded cases of `brain-mcp hook` against
internal/cli/cases_4c1_test.go:137:// TestCases4c1 replays the recordings of brain-mcp's reconcile over areas
internal/cli/cases_4d_test.go:83:// TestCases4d replays the recordings of brain-mcp's convert. The reference
internal/cli/check_test.go:409:// Without paths the check reads the working directory, as ulinit did.
internal/cli/mergehook.go:41:// mergeHookCommand is `loomux merge-hook`, `brain-mcp hook` of the reference
internal/config/manifest.go:41:// The one manifest name. loomux cut the two spellings of ultra-brain
internal/config/manifest.go:42:// (`.ultra-brain/config.toml`, `.brain.toml`) on 2026-09-14;
internal/config/manifest.go:438:// `.brain.toml` under `%LOCALAPPDATA%\brain\areas\` (counted by listing
internal/config/manifest_test.go:427:	// counted by listing `%LOCALAPPDATA%\brain\areas\*\.brain.toml`.
internal/config/policy.go:73:// Every expression is compiled here. ulguard dropped the compile error and a
internal/config/registry_test.go:205:	// (project/ultra-brain, project/space, project/ecoflow) and one of those
internal/detect/detect_test.go:384:// A .loomux/config.toml without a wiki decides, and an old .brain.toml beside
internal/dev/benchcases/cases.go:152:// ultraloom) reject a payload without session_id before doing any work, and
internal/dev/benchhooks/benchhooks.go:169:// failure -- `brain guard` ends with 2 on a repository it does not know,
internal/dev/mutants/mutants.go:2:// mutants its suite does not notice. It ports ultra-brain's
internal/gitenv/gitenv.go:1:// Package gitenv keeps git's own environment out of ultraloom's git calls.
internal/gitenv/gitenv.go:3:// Its own package because two programs need the same answer: ulinit reads
internal/gitenv/gitenv.go:4:// facts about a project, and ulguard will do the same for a worktree. A second
internal/gitenv/gitenv.go:6:// ultra-brain's, because it cuts identity and configuration as well as the
internal/gitenv/gitenv.go:32:// Nothing in ultraloom runs a server-side hook; whoever reuses this list for
internal/gitenv/gitenv.go:48:// 2026-08-28 a test's `git config user.name` in ultra-brain landed in the real
internal/gitwork/gitwork.go:6:// the log of what was added. The Python original is src/ultraloom/worktree.py.
internal/hooks/post_edit.go:268:// bundle. What stood here read `bundle` out of ultraloom's
internal/hooks/post_edit.go:269:// .ultraloom/answers.toml line by line -- a file loomux does not write, and a
internal/hooks/wikigate.go:23:// refuses every code-only commit is no gate. ultraloom installed drift only
internal/hooks/wikigate.go:27:// In `loomux check` and the stop gate alike: ultraloom ran the gate as a Stop
internal/hooks/worktree.go:78:// writes: `ulguard hook session-start` writes at session start -- as
internal/hooks/worktree.go:282:// `<worktree>/.ultraloom` would make this remove the main checkout's own
internal/hooks/worktree.go:283:// `.ultraloom/vendor` -- the pinned runtime every other hook needs, taken out
internal/hooks/worktree.go:414:// followed and nothing else, so a junction at `<dir>/.ultraloom` makes
internal/hooks/worktree.go:415:// `<dir>/.ultraloom/vendor` read and remove the reparse point of
internal/hooks/worktree.go:416:// `<main>/.ultraloom/vendor` -- the pinned runtime, taken out by the very
internal/hooks/worktree.go:496:// path is a path and not only a name: `.ultraloom/vendor` names a target two
internal/hooks/worktree_test.go:571:// pinned runtime every other ultraloom hook needs, gone silently, and `link`
internal/hooks/worktree_test.go:983:// pinned runtime every other ultraloom hook needs out of the main checkout.
internal/hosts/claude_test.go:37:// and the refusal names why. Checked against src/ultraloom/hooks/payload.py --
internal/hosts/codex.go:29:// and ultra-brain has already paid for that shape once: on 2026-09-06 its
internal/hosts/hostio.go:12:// for Antigravity, the way `brain guard` does it -- and it cannot work here:
internal/sessions/sessions.go:9:// ultraloom is no longer a writer here -- it keeps its own state under
internal/sessions/sessions.go:10:// `.ultraloom/hooks/`, a different directory.
internal/sessions/sessions.go:31:// src/ultraloom/hooks, and no hook calls `Forget`), so a file older than
internal/sessions/sessions_test.go:238:// same reasoning as ultraloom/hooks/state.py's own path builder.
internal/setup/hostfile/merge.go:226:// matcher changed, such as one of ours under ulinit's, and a block Merge
internal/setup/hostfile/merge_test.go:328:// Carried over from ulinit's settings merge, where it guarded the same
internal/setup/hostfile/merge_test.go:570:	// A command of ours under ulinit's matcher, from before MultiEdit joined
internal/setup/testdata/loomux/loomux-config.toml:58:# Laden kompiliert; was nicht kompiliert, ist ein Fehler (in ultraloom fiel
internal/worktree/junction/junction.go:70:// one place in ultraloom's own code that could.
internal/worktree/topo/worktreetopo.go:8:// `C:/Users/micro/Documents/#GIT/ultraloom/.git` for the first and
```

### Task 10 (102 Zeilen)

```text
internal/brain/check/house/federation_test.go:67:		{Scope: "project/ultra-brain", Path: "u", WikiPath: "u"},
internal/brain/check/house/federation_test.go:71:			cites("a.md", "brain://project/ultra-brain/topics/y"),
internal/brain/check/house/federation_test.go:358:		{Scope: "project/ultra-brain", Path: elsewhere,
internal/brain/check/house/federation_test.go:361:	hub := filepath.Join(areaPath, "91 P", "ultra-brain.md")
internal/brain/check/house/federation_test.go:381:		{Scope: "project/ultra-brain", Path: elsewhere,
internal/brain/check/house/federation_test.go:384:	below := filepath.Join(areaPath, "91 P", "ultra-brain.md", "d.md")
internal/brain/check/house/federation_test.go:403:		{Scope: "project/ultra-brain", Path: elsewhere,
internal/brain/check/house/federation_test.go:406:	hub := filepath.Join(areaPath, "91 P", "ultra-brain.md")
internal/brain/check/house/federation_test.go:414:	if strings.Contains(got[0], "ultra-brain.md") {
internal/brain/check/house/federation_test.go:686:		{Scope: "project/ultra-brain", Path: elsewhere,
internal/brain/check/house/federation_test.go:838:		{Scope: "project/ultra-brain", Path: elsewhere,
internal/brain/check/house/federation_test.go:918:	areas := []config.Area{post, {Scope: "project/ultra-brain", Path: elsewhere,
internal/brain/check/house/federation_test.go:921:		"knowledge": {catalog(linkTo(wikiPath, filepath.Join(areaPath, "91 P", "ultra-brain.md")))},
internal/brain/convert/model_test.go:236:	elsewhere := w.area("project/ultra-brain", "", false)
internal/brain/convert/model_test.go:237:	m := serveModel(t, w, "project/ultra-brain")
internal/brain/convert/model_test.go:240:	if !slices.Equal(out.Suggested, []string{"video.txt.md: belongs in project/ultra-brain, left in the inbox"}) {
internal/brain/convert/model_test.go:247:	if len(place) != 1 || !strings.Contains(place[0], "\n- knowledge\n- project/ultra-brain\n") || !strings.Contains(place[0], "[00:00] Hallo zusammen.") {
internal/brain/evidence/testdata/proposal-good.md:2:case: ultra-brain-2026-08-27-a4f2
internal/brain/evidence/testdata/proposal-invented.md:2:case: ultra-brain-2026-08-27-a4f2
internal/brain/evidence/testdata/proposal-two-claims.md:2:case: ultra-brain-2026-08-27-a4f2
internal/brain/guard/decide_test.go:497:		".brain.toml",
internal/brain/guard/decide_test.go:498:		filepath.Join(".ultra-brain", "config.toml"),
internal/brain/model/place_test.go:14:const placeAnswer = `{"scope": "project/ultra-brain", "grund": "Es geht um den Indexer."}`
internal/brain/model/place_test.go:32:	if got, ok := place(t, placeAnswer, "project/ultra-brain", "space"); !ok || got != "project/ultra-brain" {
internal/brain/model/place_test.go:40:		"project/ultra-brain",
internal/brain/model/place_test.go:42:		`["project/ultra-brain"]`,
internal/brain/model/place_test.go:45:		if got, ok := place(t, answer, "project/ultra-brain"); ok {
internal/brain/model/place_test.go:58:	answer := func(grund string) string { return `{"scope": "project/ultra-brain", "grund": ` + grund + `}` }
internal/brain/model/place_test.go:59:	if got, ok := place(t, answer(nested(9999)), "project/ultra-brain"); !ok || got != "project/ultra-brain" {
internal/brain/model/place_test.go:63:		if got, ok := place(t, answer(grund), "project/ultra-brain"); ok {
internal/brain/model/place_test.go:71:	allRoles(t, placeAnswer, &bodies).Place(context.Background(), "text", []string{"project/ultra-brain"})
internal/brain/model/place_test.go:87:	allRoles(t, placeAnswer, &bodies).Place(context.Background(), strings.Repeat("A", 1799)+"B"+strings.Repeat("C", 500), []string{"project/ultra-brain", "space"})
internal/brain/model/place_test.go:89:	if !strings.Contains(prompt, "\n- project/ultra-brain\n- space\n") {
internal/brain/search/search_test.go:104:		{"project/ultra-brain", "project-ultra-brain"},
internal/brain/search/search_test.go:173:	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"))
internal/brain/search/search_test.go:174:	writeRegistry(t, dir, areaEntry("knowledge", knowledge)+areaEntry("project/ultra-brain", ub))
internal/brain/search/search_test.go:178:	want := "unknown scope 'project/missing'; known scopes are: knowledge, project/ultra-brain"
internal/brain/search/search_test.go:189:	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"), "docs/intro.md")
internal/brain/search/search_test.go:191:	writeRegistry(t, dir, areaEntry("project/ultra-brain", ub)+areaEntry("knowledge", knowledge))
internal/brain/search/search_test.go:198:				{Collection: "project-ultra-brain", Relative: "docs/intro.md", Line: 1, Title: "Introduction", Snippet: "First line\nSecond line", Score: 0.85},
internal/brain/search/search_test.go:208:	if !reflect.DeepEqual(requestedCols, []string{"knowledge", "project-ultra-brain"}) {
internal/brain/search/search_test.go:209:		t.Errorf("expected sorted collections [knowledge project-ultra-brain], got %v", requestedCols)
internal/brain/search/search_test.go:214:	if answer.Hits[0].Scope != "project/ultra-brain" || answer.Hits[1].Scope != "knowledge" {
internal/brain/search/search_test.go:226:	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"), "docs/intro.md")
internal/brain/search/search_test.go:231:	writeRegistry(t, dir, areaEntry("project/ultra-brain", ub)+areaEntry("knowledge", knowledge))
internal/brain/search/search_test.go:237:			return []search.SearchHit{{Collection: "project-ultra-brain", Relative: "docs/intro.md", Line: 1, Title: "Introduction", Snippet: "Hello", Score: 0.9}}, nil
internal/brain/search/search_test.go:241:	answer, err := search.ExecuteSearch("hello", "project/ultra-brain", search.ProfileFull, 3, privacy.ChannelLocal, port, dir, searchNow)
internal/brain/search/search_test.go:245:	if !reflect.DeepEqual(requestedCols, []string{"project-ultra-brain"}) {
internal/brain/search/search_test.go:246:		t.Errorf("expected [project-ultra-brain], got %v", requestedCols)
internal/brain/search/search_test.go:248:	if len(answer.Hits) != 1 || answer.Hits[0].Scope != "project/ultra-brain" || len(answer.Findings) != 0 {
internal/brain/search/search_test.go:249:		t.Errorf("expected 1 hit with scope project/ultra-brain and no findings, got %+v %v", answer.Hits, answer.Findings)
internal/brain/search/search_test.go:532:				{Scope: "project/ultra-brain", Relative: "docs/intro.md", Line: 10, Title: "Introduction", Score: 0.854, Snippet: "@@ -1,3 @@ (header)\n# Introduction\n\nSome detail"},
internal/brain/search/search_test.go:538:		expected := "brain://project/ultra-brain/docs/intro.md:10  85%  Introduction\n" +
internal/brain/search/search_test.go:552:	head := "brain://project/ultra-brain/docs/intro.md:10  85%  Introduction\n"
internal/brain/search/search_test.go:562:			hit := search.SearchHit{Scope: "project/ultra-brain", Relative: "docs/intro.md", Line: 10, Title: "Introduction", Score: 0.854, Snippet: tc.snippet}
internal/cases/case_test.go:28:	writeCaseFile(t, dir, "cmd", []byte("brain guard write.txt\n"))
internal/cases/case_test.go:45:	if c.Cmd != "brain guard write.txt" {
internal/cases/case_test.go:46:		t.Errorf("expected cmd 'brain guard write.txt', got %q", c.Cmd)
internal/cases/runner_test.go:83:	tokens, err := cases.SplitCommand(`brain guard "path with spaces" 'single quote'`)
internal/cases/runner_test.go:152:	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "brain guard"}
internal/cli/benchcompare_test.go:286:    "SessionStart": [{"hooks": [{"type": "command", "command": "uv run ultraloom hook session-start --root \"${CLAUDE_PROJECT_DIR}\""}]}],
internal/cli/benchcompare_test.go:288:      {"matcher": "Write|Edit|Bash", "hooks": [{"type": "command", "command": "ulguard --root \"${CLAUDE_PROJECT_DIR}\""}]},
internal/cli/benchcompare_test.go:289:      {"matcher": "", "hooks": [{"type": "command", "command": "brain guard"}]}
internal/config/manifest_test.go:199:	write(t, filepath.Join(dir, ".brain.toml"), "[area]\nscope = \"brain\"\n")
internal/config/manifest_test.go:200:	if err := os.MkdirAll(filepath.Join(dir, ".ultra-brain"), 0o755); err != nil {
internal/config/manifest_test.go:203:	write(t, filepath.Join(dir, ".ultra-brain", "config.toml"), "[area]\nscope = \"ultra-brain\"\n")
internal/config/manifest_test.go:429:	owned := Area{Scope: "project/ultra-brain", Path: "P"}
internal/detect/detect_test.go:390:		".brain.toml":         {Data: []byte("[area]\nwiki = true\n")},
internal/detect/detect_test.go:394:		t.Fatalf("mode = %q: .brain.toml was read although .loomux/config.toml exists", facts.WikiMode)
internal/dev/benchcases/cases_test.go:12:    "SessionStart": [{"hooks": [{"type": "command", "command": "uv run --project \"${CLAUDE_PROJECT_DIR}/.ultraloom/vendor/ultraloom\" ultraloom hook session-start --root \"${CLAUDE_PROJECT_DIR}\""}]}],
internal/dev/benchcases/cases_test.go:14:      {"matcher": "Write|Edit|Bash", "hooks": [{"type": "command", "command": "ulguard --root \"${CLAUDE_PROJECT_DIR}\""}]},
internal/dev/benchcases/cases_test.go:16:      {"matcher": "", "hooks": [{"type": "command", "command": "brain guard"}]}
internal/dev/benchcases/cases_test.go:42:	if got := pre.Steps[0].Argv; !reflect.DeepEqual(got, []string{"ulguard", "--root", root}) {
internal/gitenv/gitenv_test.go:80:	t.Setenv("ULTRALOOM_GITENV_PROBE", "1")
internal/gitenv/gitenv_test.go:86:	if !slices.Contains(environ, "ULTRALOOM_GITENV_PROBE=1") {
internal/hooks/pretool_test.go:100:		t.Fatalf("ulguard answered 1 here; loomux refuses, got %d", code)
internal/hooks/worktree_test.go:151:	mkdirAll(t, filepath.Join(main, ".loomux", "vendor", "ultraloom"))
internal/hooks/worktree_test.go:158:	for _, relative := range []string{".tools/godot", ".loomux/vendor/ultraloom"} {
internal/hooks/worktree_test.go:541:	mkdirAll(t, filepath.Join(main, ".loomux", "vendor", "ultraloom"))
internal/hooks/worktree_test.go:555:	if _, err := os.Stat(filepath.Join(main, ".loomux", "vendor", "ultraloom")); err != nil {
internal/hooks/worktree_test.go:959:	mkdirAll(t, filepath.Join(main, ".loomux", "vendor", "ultraloom"))
internal/hooks/worktree_test.go:972:	if _, err := os.Stat(filepath.Join(main, ".loomux", "vendor", "ultraloom")); err != nil {
internal/lock/replacedir_test.go:19:		".brain.toml":     "[area]\nscope = \"" + mark + "\"\n",
internal/plancheck/plancheck_test.go:43:		"| **1a** | ✅ | ultraloom | pilot | — | — |",
internal/plancheck/plancheck_test.go:45:		"| **4d** | open | ultra-brain | convert | 4a | 3 |",
internal/setup/hostfile/merge_test.go:53:		 "hooks":[{"type":"command","command":"ulguard --root \"${CLAUDE_PROJECT_DIR}\"","timeout":10}]},
internal/setup/hostfile/merge_test.go:55:		 "hooks":[{"type":"command","command":"brain guard","timeout":15}]}
internal/setup/hostfile/merge_test.go:74:		`ulguard --root "${CLAUDE_PROJECT_DIR}"`,
internal/setup/hostfile/merge_test.go:75:		"brain guard",
internal/setup/hostfile/merge_test.go:165:	existing := []byte(`{"wiki-guard":{"PreToolUse":[{"matcher":"write_to_file","hooks":[{"type":"command","command":"brain guard"}]}]}}`)
internal/setup/hostfile/merge_test.go:177:	if guard := commands(t, got.Merged, "wiki-guard", "PreToolUse"); !reflect.DeepEqual(guard, []string{"brain guard"}) {
internal/setup/hostfile/merge_test.go:222:		"            \"command\": \"brain guard\"\n          }\n        ]\n      }\n    ]\n  }"
internal/setup/hostfile/merge_test.go:572:	ulinit := `"PreToolUse": [{"matcher": "Write|Edit|NotebookEdit|Bash|PowerShell", "hooks": [{"type": "command", "command": ` + strconvQuote(pre) + `}]}]`
internal/setup/hostfile/merge_test.go:573:	got, err = Merge(claude, []byte(`{"hooks": {`+all(ulinit)+`}}`), Entries(claude, b))
internal/setup/hostfile/table_test.go:107:		`ulguard --root "${CLAUDE_PROJECT_DIR}"`,
internal/setup/hostfile/table_test.go:108:		"brain guard",
internal/setup/plan_test.go:152:	settings := `{"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "ulguard start"}]}]}}`
internal/setup/plan_test.go:156:	if !ok || !strings.Contains(c.After, "ulguard start") || !hasNote(p, ".claude/settings.json: SessionStart/ keeps a hook of the project") {
internal/setup/templates/templates_test.go:55:	for _, unwanted := range []string{"GEMINI.md", ".agents/", "uv", "ulguard", ".ultraloom", "shim"} {
internal/setup/templates/templates_test.go:147:		for _, old := range []string{"uv run", "brain-mcp", "ultraloom", "ulguard"} {
internal/worktree/mirror/mirrorcfg_test.go:24:	write(t, root, "[worktree]\nmirror = [\".tools\", \".ultraloom/vendor\"]\n")
internal/worktree/mirror/mirrorcfg_test.go:30:	if !slices.Equal(got, []string{".tools", ".ultraloom/vendor"}) {
```

### Task 11 (58 Zeilen)

```text
.loomux/config.toml:58:# Laden kompiliert; was nicht kompiliert, ist ein Fehler (in ultraloom fiel
AGENTS.md:4:that `ultraloom` and `ultra-brain` provided separately. Design:
LICENSE.md:5:Required Notice: Copyright 2026 Christoph Wübbels (https://github.com/xidus90/ultra-brain)
README.de.md:143:Wo jede Migrationsstufe und jede aus ultraloom und ultra-brain übernommene
README.de.md:399:- [Fusions-Design: Ultraloom & Ultra-Brain](docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md)
README.md:143:Where each migration stage and each capability carried over from ultraloom
README.md:144:and ultra-brain stands — origin, status, dependencies and priority, with a map
README.md:392:- [Fusion Design: Ultraloom & Ultra-Brain](docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md)
docs/de/cli-reference.md:626:Die fünf Datenbefehle lesen die Bereiche der einen Registry (`registry.toml` in `LOOMUX_STATE_DIR` oder dessen Plattformvorgabe) und antworten wie `brain-mcp` von ultra-brain; ein aufgezeichneter Fallkorpus (`testdata/cases/1b-1`)
docs/de/cli-reference.md:671:Seit Stufe 3c. Ein aufgezeichneter Fallkorpus (`testdata/cases/3c`) hält sie an der Referenz: `brain check` am Go-Binär von ultra-brain, denn eine Python-Form gibt es nicht, die übrigen an `brain-mcp`. Registry und Erklärungen 
docs/de/cli-reference.md:706:Vier Befehle der `brain`-CLI von ultra-brain, seit Stufe 3a Befehle auf oberster Ebene von loomux; ein aufgezeichneter Fallkorpus (`testdata/cases/3a`) hält sie an der Python-Referenz. Sie lesen und schreiben allein im Zustandsver
docs/de/cli-reference.md:757:Zwei Befehle der `brain`-CLI von ultra-brain, seit Stufe 4d Befehle der obersten Ebene von loomux; ein aufgezeichneter Fallsatz (`testdata/cases/4d`, 29 Fälle) hält `convert` an der Python-Referenz, und eine Aufnahme der eigenen 
docs/de/cli-reference.md:797:Der Hook, der `reconcile` einen gelandeten Merge meldet, in jedem Repository eines Bereichs, dessen Manifest `[maintenance] on_merge = true` sagt. `brain-mcp hook` von ultra-brain unter neuem Namen, weil `hook` hier der Namensraum 
docs/de/cli-reference.md:801:- **`status`** nennt den Zustand jedes Repositorys: `installed`, `missing` (gemerkt, Datei weg), `moved` (gemerkt, die Datei steht, aber `core.hooksPath` hat sich geändert und git sucht woanders; `install` schreibt ihn dorthin, wo
docs/de/cli-reference.md:810:Drei Befehle der `brain`-CLI von ultra-brain, seit Stufe 3b Befehle auf oberster Ebene von loomux; ein aufgezeichneter Fallkorpus (`testdata/cases/3b`) hält sie an der Python-Referenz, `approve` samt den Dateien, die es schreibt, 
docs/de/cli-reference.md:1488:beim Namen und schreibt nur, was ein Mensch bestätigt. Es ersetzt `ulinit`
docs/de/cli-reference.md:1489:aus ultraloom, `scripts/install.ps1` und die Hook-Hälfte von `brain init`.
docs/de/cli-reference.md:1633:  eigener Eintrag von Claude Code unter ulinits Matcher bekommt ebenso einen
docs/de/hooks.md:445:loomux-Befehl, der ulinits unter `Write|Edit|NotebookEdit|Bash|PowerShell` in
docs/de/hooks.md:454:`ulguard` ruft, ist nicht unserer: `init` fügt den ganzen loomux-Block
docs/de/hooks.md:455:daneben ein und überlässt ulguard dir.
docs/en/cli-reference.md:603:The five data commands read the areas of the one registry (`registry.toml` in `LOOMUX_STATE_DIR` or its platform default) and answer as ultra-brain's `brain-mcp` does; a recorded case corpus (`testdata/cases/1b-1`) holds them to it
docs/en/cli-reference.md:648:Since stage 3c. A recorded case corpus (`testdata/cases/3c`) holds them to the reference: `brain check` to ultra-brain's Go binary, since there is no Python form, the others to `brain-mcp`. Registry and declarations come from `LOOM
docs/en/cli-reference.md:683:Four commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 3a; a recorded case corpus (`testdata/cases/3a`) holds them to the Python reference. They read from and write to loomux's state directory alone.
docs/en/cli-reference.md:734:Two commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 4d; a recorded case corpus (`testdata/cases/4d`, 29 cases) holds `convert` to the Python reference, and a recording of Poppler's own output holds t
docs/en/cli-reference.md:774:The hook that tells `reconcile` a merge has landed, in every repository of an area whose manifest says `[maintenance] on_merge = true`. ultra-brain's `brain-mcp hook` under a new name, since `hook` is the namespace of the host hook
docs/en/cli-reference.md:778:- **`status`** names each repository's state: `installed`, `missing` (remembered, file gone), `moved` (remembered, the file stands, but `core.hooksPath` changed and git looks elsewhere; `install` writes it where git looks now), `no
docs/en/cli-reference.md:787:Three commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 3b; a recorded case corpus (`testdata/cases/3b`) holds them to the Python reference, `approve` together with the files it writes and the commit i
docs/en/cli-reference.md:1431:writes only what a human approves. It replaces ultraloom's `ulinit`,
docs/en/cli-reference.md:1565:  Claude Code entry of ours under ulinit's matcher gets one for `MultiEdit`
docs/en/hooks.md:422:or a hand edit gave it — a loomux command that replaced ulinit's under
docs/en/hooks.md:430:entry that still runs `ulguard` is not ours: init adds the whole loomux block
docs/en/hooks.md:431:beside it and leaves ulguard to you.
docs/wiki/_schema.md:55:den in ultra-brain `brain wiki init` anlegte; loomux kennt den Befehl nicht,
docs/wiki/entities/brain-daemon.md:4:description: Der langlebige Prozess, der in ultra-brain Index und Modelle hielt — und warum er unter Windows über WMI startete.
docs/wiki/entities/brain-daemon.md:30:bei Störungen“ beschreiben den Daemon von ultra-brain bis zum Umzug am
docs/wiki/entities/brain-daemon.md:41:Der Klient von ultra-brain startete den Daemon deshalb über WMI — der Prozess
docs/wiki/log.md:47:  Abnahme“; die Scheiben von ultra-brain durch die Stufen von loomux ersetzt,
docs/wiki/log.md:67:  IDs stammten aus dem Register von `ultra-brain`, und die Quellen unter
docs/wiki/log.md:93:  Quelle: `docs/.superpowers/bench-ub/entscheidung-46.md`.
docs/wiki/log.md:97:  `docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md`.
docs/wiki/log.md:107:- 2026-09-16 — Bündel aus `ultra-brain` übernommen: 24 Inhaltsseiten und das
docs/wiki/syntheses/warum-fast-die-vorgabe-bleibt.md:41:`docs/.superpowers/bench-ub/entscheidung-46.md`; die Einordnung in
docs/wiki/topics/architektur-grundsaetze.md:4:description: Die sechs Grundsätze aus ultra-brain, die Grundsätze der Fusion, die Arbeitsteilung zwischen Code, KI und Mensch und die vier Fehlerstellen.
docs/wiki/topics/architektur-grundsaetze.md:22:Aus dem Architektur-Design ultra-brain übernommen.
docs/wiki/topics/architektur-grundsaetze.md:115:Architektur-Design ultra-brain und die Fusions-Spec.
docs/wiki/topics/brain-maintenance.md:25:Die Pflegeschicht von ultra-brain ist mit **Stufe 3** nach loomux gezogen,
docs/wiki/topics/brain-maintenance.md:177:Architektur-Design ultra-brain; Design von Stufe 3 in loomux.
docs/wiki/topics/datenmodell-und-bereiche.md:48:Das Manifest `.brain.toml` beschreibt **nur, was der eigene Bereich mitbringt** —
docs/wiki/topics/datenmodell-und-bereiche.md:53:`.brain.toml` ist der Name bis zum Umzug am 2026-09-16. loomux liest seit dem
docs/wiki/topics/datenmodell-und-bereiche.md:123:Architektur-Design ultra-brain.
docs/wiki/topics/datenschutz-und-kanaele.md:75:Quelle: Architektur-Design ultra-brain; der
docs/wiki/topics/index.md:6:* [Grundsätze und Vertrauenskette](architektur-grundsaetze.md) - Die sechs Grundsätze aus ultra-brain, die Grundsätze der Fusion, die Arbeitsteilung zwischen Code, KI und Mensch und die vier Fehlerstellen.
docs/wiki/topics/scheiben-und-abnahme.md:22:Paritätsakte. Die Herkunft: ultra-brain zerlegte seinen Bau in acht
docs/wiki/topics/scheiben-und-abnahme.md:71:| G5a | ✅ | Extraktor-Schnittstelle, Tree-sitter-Kern auf `gotreesitter`, Python; abgenommen an `iam_backend` und `ultra-brain` |
docs/wiki/topics/scheiben-und-abnahme.md:125:die Fusions-Spec und, für die Herkunft, das Architektur-Design ultra-brain.
docs/wiki/topics/suche-und-profile.md:155:Quelle: Architektur-Design ultra-brain.
docs/wiki/topics/wiki-schicht.md:162:Quellen: Architektur-Design ultra-brain; Design von Stufe 3 in loomux.
```

### Rest (87 Zeilen)

```text
CHANGELOG.md:27:- `loomux dev import-cases` no longer leaves a half-translated world (untranslated `.brain.toml`, missing `.loomux/config.toml`) when a file is briefly locked on Windows; the previous world is kept instead.
CHANGELOG.md:34:- Area declarations are read from `.loomux/config.toml` alone; `.brain.toml` and `.ultra-brain/config.toml` are no longer read. An area that still carries only an old manifest is refused with a pointer to `loomux area check`, which shows what t
CHANGELOG.md:193:- `loomux init` appends a block for the tools an own hook entry under an older matcher lacks, `manage_task` for Antigravity and `MultiEdit` for a Claude Code entry of ours under ulinit's matcher, so an upgrade guards them without a hand edit.
CHANGELOG.md:478:- `loomux brain` commands and `loomux serve` read the artefacts of read-only areas and the reconcile stamp from loomux's state directory first and fall back to ultra-brain's directory while nothing lies there.
CHANGELOG.md:500:- `loomux hook status` reports each hook event loomux serves and flags the `ultraloom hook stop` and subagent hooks as superseded.
README.de.md:169:| **Brain-Web-App** | Das Second Brain im Browser: Markdown-Editor, ADR-Katalog, Wissensgraph (Komponenten aus `ultra-brain/web`) | W2 | W1 | 5 |
README.md:169:| **Brain web app** | The second brain in the browser: markdown editor, ADR catalog, knowledge graph (components from `ultra-brain/web`) | W2 | W1 | 5 |
_identities.tsv:22:01M39G4J1401E7S4ZZQTWHNWVS	docs/.superpowers/plans-ub/2026-08-20-scheibe-2a-entscheidungen.md	sha256:0c7bfd21611ad0bb511986c9e98735c0d54fa3f1c259308f856978faa913606c	1
_identities.tsv:24:01M39G4J140FT8R61JBX8GHSGQ	docs/.superpowers/specs-ub/2026-08-25-typkatalog-und-migration-design.md	sha256:7e76602161a6f458818ca6ca76ca31fad4386937fd95b1279f44ad3a990d5726	1
_identities.tsv:27:01M39G4J1411FDN1CQW1EH893W	docs/.superpowers/specs-ul/2026-08-25-commit-sprachpruefung-design.md	sha256:5333b5bc12eb20fa670aae5f4f80ff9cd7d57b823d4e146c341b371394e5ca67	1
_identities.tsv:32:01M39G4J142G2EGP08ZS43SCNX	docs/.superpowers/specs-ul/2026-08-28-installer-kern-design.md	sha256:428f1e61cb5de2753da13a25874663a6f643a6922c59ad75faad7119fa9e4a76	1
_identities.tsv:34:01M39G4J1442WTWSH1VRCYEEDX	docs/.superpowers/plans-ub/2026-08-20-scheibe-2a-abnahme.md	sha256:71c8d0e5c0196b026a5b96125297b8f7662829501f8e1f2e8efebc36065a5027	1
_identities.tsv:37:01M39G4J1455FB5WR2GWQ7J2TV	docs/.superpowers/specs-ul/2026-08-25-policy-baukasten-design.md	sha256:d9594eb0a72bcc40725b32584d6f36fa1870ed358ef3daf894a08decfa057778	1
_identities.tsv:39:01M39G4J145RS6BD2T0F9X0XDR	docs/.superpowers/plans-ub/2026-08-20-scheibe-2a-suchkette.md	sha256:5dbd29fd127681cc63433ce94492a6f6a029e3d555339811c33c00dd70436507	1
_identities.tsv:42:01M39G4J146PHJMJRKF36X1YME	docs/.superpowers/plans-ub/2026-08-19-scheibe-1-indexer.md	sha256:1c38fd1e30390ffb31ab13083d09006e6f6a49996776312c3f781da0eb09d90e	1
_identities.tsv:43:01M39G4J146Y6S4BQ1PXBG7P32	docs/.superpowers/specs-ub/2026-08-21-pruefkorpus-design.md	sha256:2d659e52e90d1c4bf32a2cbfad420d7b53eb5efb7937d92a9776f6963a7bb940	1
_identities.tsv:44:01M39G4J146YN3QXHXNST9W7M1	docs/.superpowers/specs-ul/2026-08-25-sitzungs-hooks-payloads.md	sha256:612c029ffee7fd8f8730488c84ce20be60147b54aa4044a151ea0aad94cc862b	1
_identities.tsv:45:01M39G4J14766JJKC2EXJQP1MR	docs/.superpowers/plans-ub/2026-08-21-scheibe-2c1-daemon.md	sha256:a034db4372598772a762d81b2a12f0df815ce2fed764ae3b3f19c70b89976b10	1
_identities.tsv:51:01M39G4J148BJM914C85W1AP1X	docs/.superpowers/specs-ub/2026-09-06-produkt-design.md	sha256:8982aab4e8015816ba3c3c49ac9d4bdf1ab36291f46ca81543d3c5e5f52b3c83	1
_identities.tsv:52:01M39G4J148KZB6NB2WPYHCXAX	docs/.superpowers/specs-ul/2026-08-22-pruefkette-reihenfolge-design.md	sha256:c6f3f7b4e1ee4c8c21838efa536d8cf45b47f5a0a19de9087088de556c6b2db1	1
_identities.tsv:55:01M39G4J149QEKEEFJXRT76JRX	docs/.superpowers/bench-ub/entscheidung-46.md	sha256:649561472a12de08af8a5f5c0ef7df190aff3bf3a539ab89d12a022396d89adf	1
_identities.tsv:58:01M39G4J14AGHVYCB236STY4DY	docs/.superpowers/plans-ub/2026-08-21-pruefkorpus-v1.md	sha256:2a9358ddace4085a0ce389fda921b968fa17964e5688a741f194b50be41c301d	1
_identities.tsv:65:01M39G4J14CK311B66GRSAK7HQ	docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md	sha256:2440a49696a1f61737a94fe79d27093fbe48f043fa75d8097a6a809a42b99df3	1
_identities.tsv:71:01M39G4J14DEDFG7R8R2Y5M9P3	docs/.superpowers/plans-ub/2026-08-20-scheibe-2a-nacharbeit.md	sha256:ebcb1159c3da4e93e645839802bc486b279cd57749d0346394a1cc24137d0edf	1
_identities.tsv:72:01M39G4J14DVPR809VPSCXVNTJ	docs/.superpowers/specs-ub/2026-09-05-scheibe-6-lokales-modell-design.md	sha256:a9c5425f8176bec6b04ce61a16ada32bcf0e6bea932e5afa98aa9de2df548e44	1
_identities.tsv:74:01M39G4J14E1Q40012BSGH3PZC	docs/.superpowers/plans-ub/2026-08-18-scheibe-0-fundament.md	sha256:f927f47a11773eae3c57d9a8416882a735e6dcf1f990ef69c222c77a5bd797eb	1
_identities.tsv:84:01M39G4J14HP7S98F0MRHBHKKM	docs/.superpowers/plans-ub/2026-08-22-scheibe-2c2-mcp-fronten.md	sha256:179c1cc297ff0f27e92be6b41d84f1bed023cb975c6af624d49b5befc3c9c0aa	1
_identities.tsv:92:01M39G4J14PGZ3VB85HYNNFDT3	docs/.superpowers/specs-ub/2026-09-04-python-nach-go-migration-design.md	sha256:f22699fb984c52f10c8a1fe3f7122a2682ea7bff3a5870df8cb8f1d7f9ea1c63	1
_identities.tsv:94:01M39G4J14QJC2Q6SCQJT866TY	docs/.superpowers/plans-ub/2026-08-21-scheibe-2b-messwerk.md	sha256:e82b5213f0a5b0e3570941b3f6ae5b19d908c10846bc2d8067f5fde92ce65586	1
_identities.tsv:96:01M39G4J14QX83YZ6H311PKVSC	docs/.superpowers/specs-ul/2026-09-10-go-hooks-drei-hosts-design.md	sha256:9a6a9b4600f886c5d773b3b673129c47f377d1eef94879c2bdd1859027fd4699	1
_identities.tsv:99:01M39G4J14S3NZ5MEP32DD5RM5	docs/.superpowers/specs-ul/2026-08-21-teilprojekt-2-backlog.md	sha256:8179a7bc3ca79c4e153a9ec6dce6de6657dd0d5e8c2e12ed7ff049e543bc2e1f	1
_identities.tsv:103:01M39G4J14TVHGCDDYZ0Q9GKXY	docs/.superpowers/specs-ul/2026-09-10-antigravity-hook-messung.md	sha256:8fff143b723413b41c392e28ad1e2d2a5a3b482730afc695d23305688ae88e9f	1
_identities.tsv:108:01M39G4J14WYA2XEQ20WNXHCG9	docs/.superpowers/specs-ul/2026-09-10-wiki-flottenstandard-design.md	sha256:69138dbebac800eeb70664ad2537b9e6a823964dbc088d89b5664c58f0017fe7	1
_identities.tsv:109:01M39G4J14X1FZ6HN4EPDF6RV2	docs/.superpowers/specs-ul/2026-09-07-worktree-mirror-design.md	sha256:dd1025351bc41afda381fa44035f1142c91160a415e62fa7284367818e7c7584	1
_identities.tsv:110:01M39G4J14Y9A98XSR9XXPGYNF	docs/.superpowers/specs-ub/2026-08-27-scheibe-5-brain-maintenance-design.md	sha256:de834ed94c677d0a259a45b88142f11c8edc00848ca64f51f7654af8bf34d93d	1
_identities.tsv:112:01M39G4J14YMS69JRMEV8J6V73	docs/.superpowers/specs-ub/2026-09-13-schranke-memory-offen-design.md	sha256:4393775cb4351b561f5a7f0d5d5cae63ec90ae6e55ab663450c92ea5f536bed1	1
docs/wiki/entities/brain-daemon.md:10:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/entities/okf.md:8:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/entities/qmd.md:8:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/syntheses/gegenpruefung-vor-jeder-designempfehlung.md:9:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/syntheses/warum-fast-die-vorgabe-bleibt.md:9:    resource: brain://project/loomux/docs/.superpowers/bench-ub/entscheidung-46.md
docs/wiki/syntheses/warum-fast-die-vorgabe-bleibt.md:14:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/topics/abnahmen-und-echte-umgebung.md:9:    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-20-scheibe-2a-abnahme.md
docs/wiki/topics/abnahmen-und-echte-umgebung.md:14:    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-19-scheibe-1-indexer.md
docs/wiki/topics/architektur-grundsaetze.md:9:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/topics/brain-maintenance.md:9:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/topics/datenmodell-und-bereiche.md:10:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/topics/datenschutz-und-kanaele.md:10:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/topics/scheiben-und-abnahme.md:9:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/topics/suche-und-profile.md:10:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
docs/wiki/topics/suche-und-profile.md:15:    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-18-scheibe-0-fundament.md
docs/wiki/topics/wiki-schicht.md:10:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-entity.in.md:10:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-entity.out.md:11:  resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-long-description.in.md:9:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-long-description.out.md:10:  resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-quotes-umlauts.in.md:10:    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-21-scheibe-2b-messwerk.md
internal/brain/apply/testdata/frontmatter/w-quotes-umlauts.out.md:11:  resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-21-scheibe-2b-messwerk.md
internal/brain/apply/testdata/frontmatter/w-source.in.md:3:title: Architektur-Design ultra-brain
internal/brain/apply/testdata/frontmatter/w-source.in.md:8:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-source.in.md:22:> Die Datei `docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md`
internal/brain/apply/testdata/frontmatter/w-source.out.md:3:title: Architektur-Design ultra-brain
internal/brain/apply/testdata/frontmatter/w-source.out.md:9:  resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-source.out.md:28:> Die Datei `docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md`
internal/brain/apply/testdata/frontmatter/w-synthesis.in.md:9:    resource: brain://project/loomux/docs/.superpowers/bench-ub/entscheidung-46.md
internal/brain/apply/testdata/frontmatter/w-synthesis.in.md:14:    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-synthesis.in.md:41:`docs/.superpowers/bench-ub/entscheidung-46.md`; die Einordnung in
internal/brain/apply/testdata/frontmatter/w-synthesis.out.md:10:  resource: brain://project/loomux/docs/.superpowers/bench-ub/entscheidung-46.md
internal/brain/apply/testdata/frontmatter/w-synthesis.out.md:15:  resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
internal/brain/apply/testdata/frontmatter/w-synthesis.out.md:47:`docs/.superpowers/bench-ub/entscheidung-46.md`; die Einordnung in
internal/brain/apply/testdata/frontmatter/w-topic.in.md:9:    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-20-scheibe-2a-abnahme.md
internal/brain/apply/testdata/frontmatter/w-topic.in.md:14:    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-19-scheibe-1-indexer.md
internal/brain/apply/testdata/frontmatter/w-topic.out.md:10:  resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-20-scheibe-2a-abnahme.md
internal/brain/apply/testdata/frontmatter/w-topic.out.md:15:  resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-19-scheibe-1-indexer.md
internal/cli/cases_3a_test.go:95:			"missing file in actual: repo-new/.ultra-brain/config.toml",
internal/cli/dev_test.go:66:	code, _, errOut := run("dev", "record-case", "--cmd", "ulguard {{WORLD}}")
internal/cli/dev_test.go:106:		"--exe", filepath.Join(t.TempDir(), "gone.exe"), "--cmd", "ulguard x",
internal/cli/dev_test.go:114:	code, _, errOut := run("dev", "record-case", "--exe", "brain.exe", "--argv", "uv run brain-mcp",
internal/cli/dev_test.go:115:		"--cmd", "brain-mcp status", "--world", t.TempDir(), "--out", t.TempDir())
internal/cli/dev_test.go:123:		"--cmd", "brain-mcp status", "--world", t.TempDir(), "--out", t.TempDir())
internal/cli/dev_test.go:171:	for name, content := range map[string]string{"cmd": "ulguard --root {{WORLD}}\n", "exit": "2\n", "stdout": ""} {
internal/cli/dev_test.go:177:	if err := os.WriteFile(mapFile, []byte("[[command]]\nfrom = \"ulguard --root {{WORLD}}\"\nto   = \"loomux hook pre-tool-use --host claude --root {{WORLD}}\"\n"), 0o644); err != nil {
internal/cli/dev_test.go:202:	if err := os.WriteFile(good, []byte("[[command]]\nfrom = \"ulguard\"\nto = \"loomux\"\n"), 0o644); err != nil {
internal/cli/dev_test.go:938:	for name, content := range map[string]string{"cmd": "ultraloom check all --root {{WORLD}}\n", "exit": "1\n", "stdout": ""} {
internal/cli/dev_test.go:944:	if err := os.WriteFile(mapFile, []byte("[[command]]\nfrom = \"ultraloom check \"\nto = \"loomux check \"\n"), 0o644); err != nil {
internal/cli/devmcp_test.go:54:		"--argv", "uv run brain-mcp", "--tool", "catalog", "--arguments", `{"scope":"notes"}`,
internal/cli/devmcp_test.go:60:	if strings.Join(seen.Argv, " ") != "uv run brain-mcp" {
```

