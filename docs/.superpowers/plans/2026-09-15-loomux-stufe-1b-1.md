# loomux Stufe 1b-1 — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux brain search|catalog|read|neighbors|status` mit belegter Parität zur Python-Referenz von ultra-brain, dazu `loomux dev mutants` und die Mutationsrunde der Stufe.

**Architecture:** ultra-brains Go-Pakete ziehen mit Tests nach `internal/brain/*` und werden dort test-first auf das Verhalten der Python-Referenz gebracht. Es sind `catalog`, `reader`, `graph`, `privacy`, `index/identity.go` und beide qmd-Ports.

Neu entstehen:
- `internal/brain/pytext`: Pythons Text-, Pfad- und Zeitschreibweisen, die fünf Pakete brauchen;
- `internal/brain/status`;
- der Befehl `loomux brain`;
- ein Fake-qmd für deterministische Aufzeichnungen;
- `loomux dev mutants` über `go test -overlay`.

`brain/*` liest befristet ultra-brains Zustandsverzeichnis und die alten Manifestnamen.

**Tech Stack:**
- Go ≥ 1.25 (`go 1.25.0` in `go.mod`). Das Tor verlangt Go 1.27: Die `pytext`-Tabellen sind gegen Unicode 17.0.0 gemessen, und ein anderer Stand bricht den Tabellentest mit klarer Meldung ab.
- `github.com/BurntSushi/toml v1.6.0`, `golang.org/x/sys v0.18.0`, `gopkg.in/yaml.v3 v3.0.1`.
- Neu `golang.org/x/text v0.38.0` für NFC und casefold. Das Modul liegt offline im Modulcache und lässt `go 1.25.0` stehen (gemessen am 2026-09-15).

**Spec:** `docs/.superpowers/specs/2026-09-15-loomux-stufe-1b-1-design.md`

**Planunterlagen:** `docs/.superpowers/plans/2026-09-15-loomux-stufe-1b-1-vertrag.md` (Schnittstellenvertrag, aus dem die Tasks entworfen sind) und `docs/.superpowers/plans/2026-09-15-loomux-stufe-1b-1-rulings.md` (Regeln R1–R36 zur Reparatur der Entwürfe). „Vertrag", „Ruling Rn" und Befundnummern „Fn" in den Tasks verweisen darauf. Rangfolge: Spec vor Task, Task vor Rulings, Rulings vor Vertrag.

## Global Constraints

**Arbeitsort und Quellen**
- **Wo gearbeitet wird:** Task 0 legt es fest.
  - Empfohlen ist der Worktree `C:\Users\micro\Documents\#GIT\loomux-sdd-1b1` auf Branch `sdd-1b-1`. Den Pfad trägt der Mensch vorher in die loomux-Registry ein, und die Claude-Sitzung startet dort. Sonst verweigert die Schreibschranke des Piloten jeden Write, oder das Binary des Hauptcheckouts urteilt.
  - Der Hauptcheckout auf `master` kommt nur mit ausdrücklichem Ja des Nutzers in Frage.
  - Edit/Write-Werkzeuge verweigern Pfade unter `%TEMP%` außerhalb des Sitzungs-Scratchpads. Temporäre Dateien schreibt Go-Code (`t.TempDir()`, `os.WriteFile`) oder die Shell.
- **Quellen nur lesen:** `C:\Users\micro\Documents\#GIT\loomux-src\ub` (ultra-brain, Tag `loomux-1a-source`, `3cc72d2`). Nie aus einem laufenden Checkout kopieren. Die Referenz läuft nur mit `uv run --no-sync --project …` und `PYTHONDONTWRITEBYTECODE=1`, damit weder `.venv` noch `.pyc` im Quell-Worktree entstehen.
- **Modulpfad** `github.com/xidus90/loomux`; Pakete liegen unter `internal/`. Einstiege sind `cmd/loomux` und die Entwicklerhilfe `internal/dev/fakeqmd/qmd`, die nur für Aufzeichnungen gebaut und nie ausgeliefert wird.

**Sprache und Dokumentation**
- Code, Bezeichner, Kommentare, Fehlermeldungen und Commit-Nachrichten sind englisch. Pläne, Specs und `docs/.superpowers/parity/` sind deutsch.
- `docs/en/*.md` und `docs/de/*.md` sagen dasselbe. Die READMEs pflegt Task 16; `docs/*/cli-reference.md` pflegen Task 13 (`dev mutants`) und Task 16 (`brain`).

**Coverage, Umzug und Referenz**
- **Coverage 100 % je Funktion.** Das Tor ist `loomux dev covergate` im Pre-Commit. Eine Ausnahme gibt es nur mit `//coverage:exempt <reason>` in der Zeile direkt über `func`; der Grund nennt, was ohne Eingriff ins Betriebssystem unerreichbar ist.
- **Umzugsregel:** Code und Tests bleiben wortgleich. Erlaubt sind nur:
  - Paketnamen und Importpfade,
  - die im Task genannten Literale und Verhaltenskorrekturen,
  - neue Tests für 100 %.

  Jede weitere Änderung ist ein Befund im Task-Bericht. „Nach Python" heißt: Die Abweichung wird test-first auf das Referenzverhalten gebracht, und die erwarteten Werte sind an der Referenz gemessen, nicht aus dem Gedächtnis geschrieben.
- **Referenz im Zweifel:** Wo die Python-Referenz Verhalten oder Wortlaut hat, gilt es, auch gegen ultra-brains Go-Fassung. Gemeint sind `src/brain/core.py`, `cli.py`, `identity.py`, `privacy.py`, `walk.py`, `manifest.py`, `registry.py`, `search/qmd.py` und `maintenance/reconcile.py`. Jede Abweichung, die loomux behält, wird eine Zeile in `docs/.superpowers/parity/stufe-1b-1.md`: Der Task, der sie schafft, nennt sie in seinem Bericht, Task 12 trägt sie ein, Freigaben gibt nur der Mensch.
- **Exit-Codes `loomux brain`:**
  - 0 bei Erfolg, auch bei `no matches`;
  - 1 bei Laufzeitfehlern, mit `error: {grund}` auf stderr und ohne stdout;
  - 2 bei Usage-Fehlern.

**Tabu und Tests**
- **Nie beschreiben:**
  - `.loomux/config.toml` (jede Datei dieses Namens),
  - `%LOCALAPPDATA%\loomux\registry.toml`,
  - `%LOCALAPPDATA%\brain\**`,
  - den qmd-Index.

  Tests bauen ihre Welten in `t.TempDir()`. Registry-Einträge und Pfadregeln legt der Controller dem Menschen als Vorlage vor.
- **Kein Test startet qmd, eine GPU, `uv` oder die Python-Referenz.** Im Tor laufen nur Stubs: `httptest` für den MCP-Port, der `Runner`-Seam für den CLI-Port, `search.FakePort` und `fakeqmd` im Prozess. Die Ausnahme ist der `go`-Befehl selbst gegen ein Probemodul in `t.TempDir()`, wie `TestDevRecordCaseRecordsABinary` es seit 1a tut.
- **Startzeit:** Es gibt kein `init()`.
  - Umgezogene Paketvariablen mit `regexp.MustCompile` bleiben wortgleich; Task 15 misst sie mit `inittrace`.
  - **Neuer** Code kompiliert Regexe über `sync.OnceValue` beim ersten Gebrauch oder je Aufruf, wie 1a Task 10 es entschieden hat.
- **Abhängigkeiten:**
  - `brain/*` benutzt `config` und einander.
  - `cli` ruft `brain/*`.
  - `internal/dev/*` darf `internal/brain/pytext` benutzen.
  - `hooks` importiert kein neues `brain/*`-Paket.

**Git und Arbeitsweise**
- **Git:**
  - Kein Push, kein `gh`.
  - Commits laufen unter der Nutzeridentität, ohne `Co-Authored-By` und ohne Werbezeile. Die Nachricht ist englisch und geht über `git commit -F <datei>`.
  - Jeder Commit-Abschnitt folgt dieser Reihenfolge:
    1. `git add <genau die Dateien des Tasks>`, nie `git add .`;
    2. `sh .githooks/pre-commit`;
    3. `git branch --show-current`, `git rev-parse --short HEAD` und `git diff --cached --stat` lesen;
    4. `git commit -F <datei>`;
    5. `git log -1 --format='%an <%ae>'` und `git log -1 --format=%B` prüfen: keine `Co-Authored-By`-Zeile.

    Das Tor verweigert ungestagte oder ungetrackte `*.go`, `go.mod`, `go.sum`, `testdata` und `.githooks`; darum wird vor dem Tor gestagt.
- **Ein Shell-Befehl je Aufruf**, keine langen `&&`-Ketten.
- **Subagenten** laufen mit `model: "opus"` und ausdrücklich gesetztem `effort`, ein Implementierer je Task.

## Dateistruktur nach Stufe 1b-1

| Pfad | Verantwortung | Herkunft | Task |
|---|---|---|---|
| `internal/config/legacy.go`, `internal/config/manifest.go` | befristet: ultra-brains Zustandsverzeichnis (bis Stufe 3), Manifest-Altnamen (bis Stufe 4) | neu, geändert | 1 |
| `internal/brain/pytext/` | Pythons `repr`, `splitlines`, `strip`, `read_text`, `isoformat`/`fromisoformat`, `str(Path)`, NFC, casefold | neu | 2 |
| `internal/brain/privacy/` | Kanal, Sichtbarkeit aller Bereiche, Enthaltensein, `never`-Globs nach `full_match`, Prüfzentrum | ub `pkg/privacy` + neu `areas.go`, `glob.go` | 3 |
| `internal/brain/identity/` | `_identities.tsv` lesen | ub `pkg/index/identity.go` | 4 |
| `internal/brain/graph/` | `graph.json`, Nachbarn | ub `pkg/graph` `model.go`, `neighbors.go`, `read.go` | 5 |
| `internal/brain/reader/` | Dokument und Abschnitt | ub `pkg/reader` | 6 |
| `internal/brain/catalog/` | Wurzel- und Bereichskatalog | ub `pkg/catalog` `area.go`, `root.go` | 7 |
| `internal/brain/search/` | MCP- und CLI-Port, Suche, Befunde, Reconcile-Stempel | ub `pkg/search` + neu `stamp.go` | 8 |
| `internal/brain/status/` | `status`-Zeilen | neu | 9 |
| `internal/cli/brain.go`, `internal/cli/brainargs.go` | `loomux brain …` mit argparse-gleichen Formen | neu | 10 |
| `internal/dev/fakeqmd/`, `internal/dev/fakeqmd/qmd/` | Fake-qmd aus einer Fixture: CLI-Antworten und MCP-Handler | neu | 11 |
| `internal/dev/recordcase/`, `internal/dev/importcases/`, `internal/cli/dev.go` | Programmform, Umgebung, CRLF; Faltung in jedem Registry-Pfad | geändert | 11 |
| `testdata/cases/1b-1-worlds/`, `…/1b-1-source/`, `…/1b-1/`, `…/1b-1-map.toml`, `internal/cli/cases_1b1_test.go` | Fallkorpus | neu | 12 |
| `docs/.superpowers/parity/stufe-1b-1.md` | Abweichungsliste | neu | 12, 14, 15, 16 |
| `internal/dev/mutants/`, `internal/cli/dev.go`, `docs/{en,de}/cli-reference.md` | `loomux dev mutants` | neu, geändert | 13 |
| `<paket>/mutation_test.go` | Tests für Überlebende der Runde | neu | 14 |
| `testdata/bench/1b-1-brain.json`, `internal/brain/search/bench_test.go`, `docs/{en,de}/benchmarks.md` | Messung | neu, geändert | 15 |
| `README.md`, `README.de.md`, `docs/{en,de}/cli-reference.md` | Nutzerdoku für `loomux brain` | geändert | 16 |

**Nicht umgezogen:** ub `pkg/catalog/catalog.go`, `pkg/graph/render.go`, `pkg/graph/synth.go`, `pkg/index` außer `identity.go`, `pkg/mcp`, `pkg/wiki`.

## Umzugsverfahren (gilt für die Tasks 3–8)

Jeder Umzug folgt denselben Schritten; die Tasks nennen Quelle, Ziel, Paketname und die geänderten Stellen und schreiben die Befehle für ihre Dateien aus. `$LOOMUX` und `$SRC` kommen aus Task 0.

| alter Importpfad | neuer Importpfad |
|---|---|
| `github.com/xidus90/ultra-brain/pkg/config` | `github.com/xidus90/loomux/internal/config` |
| `github.com/xidus90/ultra-brain/internal/testlock` | `github.com/xidus90/loomux/internal/testlock` |
| `github.com/xidus90/ultra-brain/pkg/privacy` | `github.com/xidus90/loomux/internal/brain/privacy` |
| `github.com/xidus90/ultra-brain/pkg/graph` | `github.com/xidus90/loomux/internal/brain/graph` |
| `github.com/xidus90/ultra-brain/pkg/reader` | `github.com/xidus90/loomux/internal/brain/reader` |
| `github.com/xidus90/ultra-brain/pkg/catalog` | `github.com/xidus90/loomux/internal/brain/catalog` |
| `github.com/xidus90/ultra-brain/pkg/search` | `github.com/xidus90/loomux/internal/brain/search` |
| `github.com/xidus90/ultra-brain/pkg/index` (nur `identity.go`) | `github.com/xidus90/loomux/internal/brain/identity` |

1. **Nur die im Task genannten Dateien kopieren**, Tests eingeschlossen, je Datei ein Aufruf:
   ```bash
   mkdir -p "$LOOMUX/<ziel>"
   ```
   ```bash
   cp "$SRC/<quelle>/<datei>" "$LOOMUX/<ziel>/"
   ```
2. **Paketnamen setzen**, nur wenn er sich ändert, in Nicht-Test- und Testdateien:
   ```bash
   sed -i 's/^package <alt>$/package <neu>/; s/^package <alt>_test$/package <neu>_test/' "$LOOMUX/<ziel>"/*.go
   ```
3. **Importpfade umschreiben** nach der Tabelle oben, danach Paketbezeichner, falls sich der Paketname geändert hat. Vorher prüfen, dass nur die frisch kopierten Dateien alte Pfade tragen:
   ```bash
   grep -rl 'github.com/xidus90/ultra-brain/' "$LOOMUX" --include=*.go
   ```
   Erwartet: genau die Dateien aus Schritt 1, weil die vorherigen Tasks committet sind.
4. **Formatieren und prüfen:**
   - `gofmt -w "$LOOMUX/<ziel>"`, dann `go vet ./<ziel>/...` und `go test ./<ziel>/...`. Das muss grün sein, bevor irgendetwas anderes geändert wird.
   - Kein `go mod tidy` in Umzügen: `golang.org/x/text` holt Task 2, und kein umgezogenes Paket braucht ein weiteres Fremdmodul.
   - **Ausnahme Task 3:** Nach dem Importumbau sind genau fünf Untertests von `TestVisibleManifest` rot, weil der umgezogene `VisibleManifest` noch `config.ReadManifest` ruft, das nur `.loomux/config.toml` kennt. Task 3 behebt das als ersten Schritt nach dem Umzug.
5. **Die im Task genannten Literale und Verhaltenskorrekturen ändern**, test-first. Tests, die das alte Verhalten prüfen, werden im selben Schritt umgestellt.
6. **Coverage heben:**
   ```bash
   go test ./<ziel>/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
   ```
   ```bash
   go tool cover -func="$TEMP/pkg.out"
   ```
   Erwartet: Jede Funktion des Pakets steht bei 100.0%. Für eine Zeile darunter zeigt `go tool cover -html="$TEMP/pkg.out" -o "$TEMP/pkg.html"` die roten Blöcke. Je Block entsteht ein Test, der genau ihn ausführt: erst rot sehen, dann grün. Ist ein Block ohne Eingriff ins Betriebssystem nicht erreichbar, bekommt die Funktion `//coverage:exempt <reason>`; der Grund nennt, was fehlen müsste, damit der Block läuft.
7. **Stagen, Tor, Commit** in der Reihenfolge der Global Constraints.

---

### Task 0: Arbeitsort (Mensch, oder mit ausdrücklichem Ja im Chat)

Dieser Task legt `$LOOMUX` erst fest: Steps 2–4 laufen darum mit absoluten Pfaden (`git -C` mit dem ausgeschriebenen Pfad), ab Step 5 läuft jeder Befehl in `$LOOMUX`. Voraussetzung: keine; kein Task davor.

**Files:** keine im Repo. Außerhalb: ein neuer Worktree `C:\Users\micro\Documents\#GIT\loomux-sdd-1b1` (Branch `sdd-1b-1`) und ein angehängter Block in `%LOCALAPPDATA%\loomux\registry.toml` — den schreibt **nur der Mensch** (Global Constraints: „Nie beschreiben").

**Interfaces:** keine. Dieser Task legt nur die zwei Namen fest, auf die alle weiteren Tasks sich beziehen:

| Name | Worktree (empfohlen) | Hauptcheckout (nur mit ausdrücklichem Ja des Nutzers) |
|---|---|---|
| `LOOMUX` | `/c/Users/micro/Documents/#GIT/loomux-sdd-1b1` | `/c/Users/micro/Documents/#GIT/loomux` |
| Branch | `sdd-1b-1` | `master` |
| `SRC` | `/c/Users/micro/Documents/#GIT/loomux-src/ub` | dasselbe |

Jeder Befehl der Tasks 1–16 läuft mit `$LOOMUX` als Arbeitsverzeichnis. Der Hauptcheckout trägt heute uncommittete Änderungen des Nutzers an `README.md`, `README.de.md`, `docs/de/benchmarks.md`, `docs/en/benchmarks.md`; ein Worktree nimmt sie nicht mit und stört sie nicht — ein weiterer Grund für den Worktree.

**Vorab gemessen (2026-09-15):**

| Frage | Befund |
|---|---|
| `git -C <loomux> rev-parse --short master` | `e4bf7b5` (nach dem Spec-Commit `311d5b2` nur Doku: `README*.md`, `docs/*/benchmarks.md`; `go.mod` unverändert) |
| Branch `sdd-1b-1`, Pfad `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1` | beide noch nicht vorhanden |
| `git -C <loomux> config --get core.hooksPath` | `.githooks` — steht in der gemeinsamen Repo-Konfiguration, gilt also auch im Worktree; ein relativer `hooksPath` wird gegen die Wurzel des Worktrees aufgelöst |
| `git -C "$SRC" rev-parse --short HEAD`, `git -C "$SRC" status --porcelain` | `3cc72d2`, keine Ausgabe |
| `go version` | `go version go1.27.0 windows/amd64` |
| `PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c "import sys; print(sys.version.split()[0])"` | `3.14.7` |
| `.gitignore` | `/bin/`, `/coverage.out`, `*.test`, `/.loomux/state/` — ein frischer Worktree hat also kein `bin/loomux.exe` |
| `.claude/settings.json` | alle drei Hooks rufen `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe"`; das Binary des Verzeichnisses, in dem die Claude-Sitzung **startet**, entscheidet über jeden Write |
| Schranke heute, Write nach `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/internal/probe.go` | Exit 2; die Ausgabe beginnt mit `{"decision": "deny", "reason": "C:\\Users\\micro\\Documents\\#GIT\\loomux-sdd-1b1\\internal\\probe.go lies outside every writable tree; writing is allowed only below: ` und zählt danach die Bäume der Registry dieser Maschine auf |
| dieselbe Probe gegen eine Kopie der Registry mit dem Block unten (`LOOMUX_STATE_DIR` auf die Kopie) | Exit 0, keine Ausgabe |
| dieselbe Kopie, Write nach `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/.loomux/config.toml` | Exit 2; der `reason` nennt den Pfad der Datei und endet mit `\.loomux\config.toml: the manifest is where the barrier reads its own limits, so no writing tool may touch it` |
| `go version` gegen die Torvoraussetzung | Go 1.27 ist Voraussetzung des Tors: `TestTheTablesAreTheOnesMeasured` (Task 2) verlangt die Unicode-17.0.0-Tabellen, die erst Go 1.27 mitbringt; `go.mod` erlaubt mit `go 1.25.0` ältere Werkzeugketten, an denen das Tor rot wird |

Der Registry-Block braucht kein `wiki`: `workspace = true` öffnet den ganzen Baum (gemessen oben), und `checkInbox` (`internal/brain/guard/registry.go:218`) überspringt einen Bereich, dessen Manifest noch keine Datei ist. Der Scope `project/loomux-sdd-1b1` ergibt ein anderes Zustandsverzeichnis als `project/loomux`, die Doppelprüfung der Registry schlägt also nicht an.

- [ ] **Step 1: Arbeitsort wählen**

Der Controller fragt den Nutzer: Worktree (empfohlen) oder Hauptcheckout auf `master`. Ohne ausdrückliches Ja zum Hauptcheckout gilt der Worktree. Beim Hauptcheckout entfallen Step 3 und 4 und in Step 2 der Befehl `git branch --list sdd-1b-1`; die übrigen Prüfungen von Step 2 (Stand von `master`, `$SRC`, Go 1.27) laufen auch dort. Danach weiter mit Step 5, `LOOMUX=/c/Users/micro/Documents/#GIT/loomux`.

- [ ] **Step 2: Vorbedingungen prüfen**

```bash
git -C "/c/Users/micro/Documents/#GIT/loomux" status --short --branch
```
Expected: erste Zeile `## master`; höchstens die vier Doku-Dateien des Nutzers als ` M`. Keine `*.go`-Datei, kein `go.mod`.
```bash
git -C "/c/Users/micro/Documents/#GIT/loomux" branch --list sdd-1b-1
```
Expected: keine Ausgabe.
```bash
git -C "/c/Users/micro/Documents/#GIT/loomux-src/ub" rev-parse --short HEAD
```
Expected: `3cc72d2`.
```bash
go version | grep -E '^go version go1\.27(\.[0-9]+)? '
```
Expected: genau eine Trefferzeile, heute `go version go1.27.0 windows/amd64`. Kein Treffer: anhalten, der Mensch installiert Go 1.27 — mit einer älteren Werkzeugkette wird das Tor ab Task 2 rot (`pytext tables are measured against Unicode 17.0.0 (Go 1.27)`).

- [ ] **Step 3: Worktree anlegen** (Mensch, oder der Controller nach ausdrücklichem Ja)

```bash
git -C "/c/Users/micro/Documents/#GIT/loomux" worktree add -b sdd-1b-1 "/c/Users/micro/Documents/#GIT/loomux-sdd-1b1"
```
Expected: Exit 0.
```bash
test "$(git -C "/c/Users/micro/Documents/#GIT/loomux-sdd-1b1" rev-parse --short HEAD)" = "$(git -C "/c/Users/micro/Documents/#GIT/loomux" rev-parse --short master)"
```
Expected: keine Ausgabe, Exit 0 — der Worktree steht auf dem Stand von `master`, gleich welcher Commit das beim Anlegen ist. Exit 1: anhalten, der Worktree ist nicht von `master` abgezweigt.

- [ ] **Step 4: Registry-Eintrag (nur Mensch)**

Der Mensch hängt an `%LOCALAPPDATA%\loomux\registry.toml` (`C:\Users\micro\AppData\Local\loomux\registry.toml`) an:
```toml
[[area]]
scope     = "project/loomux-sdd-1b1"
path      = "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1"
workspace = true
```
Kein Agent schreibt diese Datei, auch nicht auf Bitte.

Zuerst prüfen, dass das Binary des Hauptcheckouts da ist (es ist git-ignoriert und kann fehlen):
```bash
test -x "/c/Users/micro/Documents/#GIT/loomux/bin/loomux.exe"
```
Expected: keine Ausgabe, Exit 0. Exit 1: anhalten; der Mensch baut es im Hauptcheckout mit `go build -o bin/loomux.exe ./cmd/loomux` und startet diesen Schritt neu.

Prüfen (liest nur: `internal/hooks/pretool.go` und `guard.go` rufen weder `sessions` noch die Worktree-Verknüpfung, und `internal/brain/guard` enthält kein `os.WriteFile`/`MkdirAll`/`OpenFile`/`Create`; unter `.loomux/state/hooks/` schreiben nur `SessionStart` und `loomux worktree link|unlink|remove`):
```bash
printf '%s' '{"session_id":"probe","hook_event_name":"PreToolUse","cwd":"C:/Users/micro/Documents/#GIT/loomux-sdd-1b1","tool_name":"Write","tool_input":{"file_path":"C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/internal/probe.go","content":"x"}}' | "/c/Users/micro/Documents/#GIT/loomux/bin/loomux.exe" hook pre-tool-use --host claude --root "C:/Users/micro/Documents/#GIT/loomux"
```
Expected: keine Ausgabe, Exit 0. Vor dem Eintrag war es Exit 2 mit einem `reason`, der `lies outside every writable tree; writing is allowed only below: ` enthält (gemessen); kommt diese Meldung noch, steht der Block nicht in der Datei oder der Pfad ist anders geschrieben.

- [ ] **Step 5: Pilot-Binary im Arbeitsort bauen**

Ab hier läuft jeder Befehl mit `$LOOMUX` als Arbeitsverzeichnis.
```bash
go build -o bin/loomux.exe ./cmd/loomux
```
Expected: keine Ausgabe, Exit 0; `bin/loomux.exe` existiert (git-ignoriert).

- [ ] **Step 6: Hooks des Repos scharf schalten**

```bash
git config core.hooksPath .githooks
```
Expected: keine Ausgabe. Der Wert stand schon (gemeinsame Konfiguration); der Befehl ist idempotent und bleibt, damit ein frischer Klon denselben Weg geht wie `AGENTS.md` ihn beschreibt.
```bash
git config --get core.hooksPath
```
Expected: `.githooks`.

- [ ] **Step 7: Tor grün vor Task 1**

```bash
git branch --show-current
```
Expected: `sdd-1b-1` (Hauptcheckout: `master`).
```bash
sh .githooks/pre-commit
```
Expected: Exit 0 — keine Zeile `pre-commit: inputs differ from the index`, keine `gofmt:`-Liste, `go vet` still, `go test` ohne `FAIL`, `covergate` ohne `not covered:`, danach `bin/loomux.new.exe` gebaut und von `dev swap-binary` an die Stelle von `bin/loomux.exe` gesetzt. Scheitert das Tor hier, ist der Stand von `master` rot: kein Task beginnt, der Befund geht an den Menschen.

- [ ] **Step 8: Sitzung im Arbeitsort starten (Mensch)**

Die Claude-Sitzung, die die Tasks 1–16 fährt, startet **in `$LOOMUX`**. Sonst ist `${CLAUDE_PROJECT_DIR}` der Hauptcheckout, dessen `bin/loomux.exe` jeden Write beurteilt und von den Commits im Worktree nie neu gebaut wird; die Warnung der Session-Start-Hook über ein veraltetes Binary beträfe dann das falsche Verzeichnis.

- [ ] **Step 9: Bericht**

Der Bericht nennt: gewählten Arbeitsort, `LOOMUX`, Branch, `git rev-parse --short HEAD`, das Ergebnis der Schrankenprobe aus Step 4 (Exit-Code) und den Exit-Code von `sh .githooks/pre-commit`.

Paritätszeilen: keine — dieser Task ändert kein Verhalten.

---

### Task 1: Befristete Ausnahme in `internal/config`

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Task 0 ist abgeschlossen (Task 0 committet nichts; einen Task davor mit Commit gibt es nicht).

**Files:**
- Create: `internal/config/legacy.go`
- Test: `internal/config/legacy_test.go` (neu)
- Modify: `internal/config/manifest.go` — nur `ReadManifest` (Zeilen 137–211 am Stand `311d5b2`): die Schleife zieht in den internen Leser `readManifestAmong` um; der Rest der Datei bleibt wortgleich.

**Interfaces:**
- Consumes: `Manifest`, `ErrNoManifest`, `manifestNames` (`manifest.go:38`, `:43`), `defaultStateDir(goos string, getenv func(string) string, home string) string` (`registry.go:166`); in Tests `stub(env map[string]string) func(string) string` (`registry_test.go:271`), `write(t, path, content)` und `manifestIn(t, dir)` (`manifest_test.go:180`, `:189`), `testlock.Lock(t, path)` (`internal/testlock`).
- Produces (Vertrag, wörtlich):
```go
// LegacyBrainDirUntilStage3 is ultra-brain's state directory. brain/* reads the artefacts of
// read-only areas and the reconcile stamp from it until stage 3 moves reconcile to Go.
func LegacyBrainDirUntilStage3() string // LOOMUX_LEGACY_BRAIN_DIR; else %LOCALAPPDATA%\brain (home\AppData\Local\brain); POSIX $XDG_STATE_HOME/brain, else home/.local/state/brain
// ReadAreaManifestUntilStage4 reads an area's manifest under the name loomux writes, or under
// one of ultra-brain's names until stage 4 moves the hosts: .loomux/config.toml, else
// .ultra-brain/config.toml, else .brain.toml. Only brain/* calls it.
func ReadAreaManifestUntilStage4(dir string) (*Manifest, error)
```
- Produces (Ergänzungen, alle unexportiert; jeder Name der befristeten Ausnahme trägt seinen Ablauf, damit eine Suche nach `UntilStage` in Stufe 3 und 4 alles findet):
```go
const legacyBrainDirEnvUntilStage3 = "LOOMUX_LEGACY_BRAIN_DIR"
var manifestNamesUntilStage4 = []string{filepath.Join(".loomux", "config.toml"), filepath.Join(".ultra-brain", "config.toml"), ".brain.toml"}
func legacyBrainDirUntilStage3For(goos string, getenv func(string) string, home string) string // filepath.Join(filepath.Dir(defaultStateDir(goos, getenv, home)), "brain")
func readManifestAmong(dir string, names []string, requireScope bool) (*Manifest, error) // shared by ReadManifest (manifestNames, false) and ReadAreaManifestUntilStage4 (manifestNamesUntilStage4, true); requireScope expires with stage 4
```
Kein anderer Task nennt einen dieser unexportierten Namen; exportiert bleiben nur die beiden Vertragsfunktionen.

**Unberührt:** `registry.go` wird nicht angefasst: `legacyBrainDirUntilStage3For` rechnet über `filepath.Dir(defaultStateDir(goos, getenv, home))`, `StateDir` bleibt damit bitgleich (belegt durch die unveränderten Tests in `registry_test.go`).

**Gemessen an der Referenz (Python 3.14.7, `PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c` mit `read_manifest(Path)` auf Fixture-Dateien):**

| Manifestinhalt | `read_manifest` | loomux nach diesem Task |
|---|---|---|
| `[area]` ohne `scope` | `{path}: [area] scope is required and must be a non-empty string` | gleich |
| `scope = ""` | dieselbe Meldung | gleich |
| nur `[privacy] mode = "bogus"` (kein Scope, falscher Modus) | dieselbe Meldung (Scope wird vor dem Modus geprüft) | gleich |
| `scope = " "` | angenommen, Scope `' '` | gleich |
| `scope = 3` | dieselbe Scope-Meldung | `{path}: not valid TOML: toml: line 2 (last key "area.scope"): incompatible types: TOML value has type int64; destination has type string` — Paritätszeile |
| BOM vor `[area]` | `{path}: not valid TOML: Invalid statement (at line 1, column 1)` | angenommen — Paritätszeile |
| `[area` (kaputt) | `{path}: not valid TOML: Expected ']' at the end of a table declaration (at line 1, column 6)` | `{path}: not valid TOML: ` + Wortlaut von BurntSushi toml — Paritätszeile |
| ungültiges UTF-8 im Wert | `UnicodeDecodeError` als Traceback (fängt `_parse` nicht) | `{path}: not valid TOML: toml: line 2 (last key "area.scope"): invalid UTF-8 byte: 0xff` — Paritätszeile |

Namensvorrang Python (`registry.manifest_path`, `registry.py:132-138`): `.ultra-brain/config.toml`, wenn `is_file()`, sonst `.brain.toml` ohne weitere Prüfung.

**`.loomux/config.toml` ohne `[area]` (keine Referenz, Python kennt den Namen nicht):** Die Datei ist dann nur Policy und deklariert nichts, wie die Schranke sie seit Stufe 1a liest (Guard-Regel R7a, `internal/brain/guard/manifest.go:74`, Paritätsliste 1a). `ReadAreaManifestUntilStage4` fragt darum den nächsten Namen; gibt es keinen, folgt `ErrNoManifest` mit allen drei Namen. Eine vorhandene `[area]`-Tabelle ohne oder mit leerem `scope` ist dagegen der Scope-Fehler dieser Datei, auch neben einer gültigen `.brain.toml`. Für die beiden Altnamen gilt das nicht: `read_manifest` verweigert sie ohne Scope, gleich ob `[area]` fehlt. Gemessen im Wegwerfmodul: `meta.IsDefined("area")` von BurntSushi toml ist für ein leeres `[area]` wahr und für eine Datei nur mit `[privacy]` oder `[layout]` falsch (die Tests in Step 3 unterscheiden beides).

- [ ] **Step 1: Den Leser aus `ReadManifest` herausziehen (Refactoring unter den bestehenden Tests)**

In `internal/config/manifest.go`, Funktion `ReadManifest`:

vorher (`manifest.go:137-140`):
```go
// ReadManifest reads the manifest of the area rooted at repoRoot.
func ReadManifest(repoRoot string) (*Manifest, error) {
	for _, name := range manifestNames {
		path := filepath.Join(repoRoot, name)
```
nachher:
```go
// ReadManifest reads the manifest of the area rooted at repoRoot.
func ReadManifest(repoRoot string) (*Manifest, error) {
	return readManifestAmong(repoRoot, manifestNames)
}

// readManifestAmong reads the first of names below dir that is a regular
// file.
func readManifestAmong(dir string, names []string) (*Manifest, error) {
	for _, name := range names {
		path := filepath.Join(dir, name)
```

vorher (`manifest.go:209-210`):
```go
	return nil, fmt.Errorf("%s: %w (%s)", repoRoot, ErrNoManifest,
		strings.Join(manifestNames, ", "))
```
nachher:
```go
	return nil, fmt.Errorf("%s: %w (%s)", dir, ErrNoManifest,
		strings.Join(names, ", "))
```
Alles zwischen den beiden Stellen (der Kommentarblock über `os.ReadFile`, Lesen, Dekodieren, Modusprüfung, Lanes, Rückgabe) bleibt wortgleich.

- [ ] **Step 2: Bestehende Tests grün sehen**

```bash
go test ./internal/config/ -count=1
```
Expected: `ok  	github.com/xidus90/loomux/internal/config`. Kein Test wurde geändert; das belegt, dass `ReadManifest` sich verhält wie vorher.

- [ ] **Step 3: Failing tests schreiben** — `internal/config/legacy_test.go`

```go
package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

func TestTheLegacyWindowsDirIsLocalAppDataBrain(t *testing.T) {
	got := legacyBrainDirUntilStage3For("windows", stub(map[string]string{"LOCALAPPDATA": `D:\state`}), `C:\Users\x`)
	if want := filepath.Join(`D:\state`, "brain"); got != want {
		t.Errorf("legacyBrainDirUntilStage3For = %q, want %q", got, want)
	}
}

func TestTheLegacyWindowsDirFallsBackToTheHome(t *testing.T) {
	got := legacyBrainDirUntilStage3For("windows", stub(map[string]string{}), `C:\Users\x`)
	if want := filepath.Join(`C:\Users\x`, "AppData", "Local", "brain"); got != want {
		t.Errorf("legacyBrainDirUntilStage3For = %q, want %q", got, want)
	}
}

func TestTheLegacyDirElsewhereFollowsXdgStateHome(t *testing.T) {
	got := legacyBrainDirUntilStage3For("linux", stub(map[string]string{"XDG_STATE_HOME": "/s"}), "/home/x")
	if want := filepath.Join("/s", "brain"); got != want {
		t.Errorf("legacyBrainDirUntilStage3For = %q, want %q", got, want)
	}
}

func TestTheLegacyDirElsewhereWithoutXdgIsLocalStateBrain(t *testing.T) {
	got := legacyBrainDirUntilStage3For("linux", stub(map[string]string{}), "/home/x")
	if want := filepath.Join("/home/x", ".local", "state", "brain"); got != want {
		t.Errorf("legacyBrainDirUntilStage3For = %q, want %q", got, want)
	}
}

func TestTheLegacyEnvironmentVariableBeatsThePlatform(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
	if got := LegacyBrainDirUntilStage3(); got != dir {
		t.Errorf("LegacyBrainDirUntilStage3() = %q, want %q", got, dir)
	}
}

func TestTheLegacyDirIgnoresLoomuxStateDir(t *testing.T) {
	// The two directories are two answers: a registry in loomux's and the
	// artefacts in ultra-brain's. Following LOOMUX_STATE_DIR here would read
	// the artefacts of read-only areas from a directory nothing writes.
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", "")
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	t.Setenv("LOCALAPPDATA", `D:\state`)
	t.Setenv("XDG_STATE_HOME", "/state")
	home, _ := os.UserHomeDir()
	if want := legacyBrainDirUntilStage3For(runtime.GOOS, os.Getenv, home); LegacyBrainDirUntilStage3() != want {
		t.Errorf("LegacyBrainDirUntilStage3() = %q, want %q", LegacyBrainDirUntilStage3(), want)
	}
}

func TestTheLegacyDirPassesTheRealHomeOn(t *testing.T) {
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", "")
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("XDG_STATE_HOME", "")
	home, _ := os.UserHomeDir()
	got := LegacyBrainDirUntilStage3()
	if want := legacyBrainDirUntilStage3For(runtime.GOOS, os.Getenv, home); got != want {
		t.Errorf("LegacyBrainDirUntilStage3() = %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, home) {
		t.Errorf("LegacyBrainDirUntilStage3() = %q does not sit under the home directory %q", got, home)
	}
}

// legacyWrite writes content to name below dir and creates the directories
// the name needs.
func legacyWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTheThreeNamesAreTriedInOrder(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"loomux\"\n")
	legacyWrite(t, dir, filepath.Join(".ultra-brain", "config.toml"), "[area]\nscope = \"ultra-brain\"\n")
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
	for _, step := range []struct {
		want   string
		remove string
	}{
		{want: "loomux", remove: ".loomux"},
		{want: "ultra-brain", remove: ".ultra-brain"},
		{want: "brain"},
	} {
		m, err := ReadAreaManifestUntilStage4(dir)
		if err != nil {
			t.Fatal(err)
		}
		if m.Scope != step.want {
			t.Errorf("Scope = %q, want %q", m.Scope, step.want)
		}
		if step.remove != "" {
			if err := os.RemoveAll(filepath.Join(dir, step.remove)); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestADirectoryUnderANameIsSkipped(t *testing.T) {
	// `manifest_path` asks `is_file()`, so a directory of that name is not
	// the manifest and the next name is asked.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".ultra-brain", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
	m, err := ReadAreaManifestUntilStage4(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Scope != "brain" {
		t.Errorf("Scope = %q, want %q", m.Scope, "brain")
	}
}

func TestALegacyManifestCarriesItsPrivacy(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, ".brain.toml",
		"[area]\nscope = \"project/closed\"\n\n[privacy]\nmode = \"local_only\"\nnever = [\"secret/**\", \"*.pem\"]\n")
	m, err := ReadAreaManifestUntilStage4(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.PrivacyMode != "local_only" {
		t.Errorf("PrivacyMode = %q, want local_only", m.PrivacyMode)
	}
	if strings.Join(m.NeverGlobs, ",") != "secret/**,*.pem" {
		t.Errorf("NeverGlobs = %v, want [secret/** *.pem]", m.NeverGlobs)
	}
}

func TestALegacyManifestWithoutAScopeIsRefused(t *testing.T) {
	// Wording and order of `read_manifest` (src/brain/manifest.py:21-24),
	// measured: a missing table, a missing key and an empty string all give
	// this one message, and it comes before the complaint about the mode.
	for name, body := range map[string]string{
		"no area table":  "[privacy]\nmode = \"local_only\"\n",
		"no scope key":   "[area]\nname = \"x\"\n",
		"empty scope":    "[area]\nscope = \"\"\n",
		"and a bad mode": "[privacy]\nmode = \"bogus\"\n",
	} {
		dir := t.TempDir()
		legacyWrite(t, dir, ".brain.toml", body)
		_, err := ReadAreaManifestUntilStage4(dir)
		want := filepath.Join(dir, ".brain.toml") + ": [area] scope is required and must be a non-empty string"
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", name, err, want)
		}
	}
}

func TestAPolicyOnlyLoomuxManifestDeclaresNothingForBrain(t *testing.T) {
	// A .loomux/config.toml without an [area] table is policy only and
	// declares nothing, as the write barrier reads it (stage 1a, R7a): the
	// next name is asked, and the privacy of the policy file is not the
	// area's.
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), "[privacy]\nmode = \"local_only\"\n")
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
	m, err := ReadAreaManifestUntilStage4(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Scope != "brain" || m.PrivacyMode != "manual_cloud" {
		t.Errorf("Scope, PrivacyMode = %q, %q; want brain, manual_cloud", m.Scope, m.PrivacyMode)
	}
}

func TestAPolicyOnlyLoomuxManifestAloneIsErrNoManifest(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), "[layout]\nwiki = \"docs/wiki\"\n")
	_, err := ReadAreaManifestUntilStage4(dir)
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("err = %v, want ErrNoManifest", err)
	}
	want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + ", " +
		filepath.Join(".ultra-brain", "config.toml") + ", .brain.toml)"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestAnAreaTableWithoutAScopeInTheLoomuxManifestIsRefused(t *testing.T) {
	// An [area] table is a declaration, so an empty one is a scope error for
	// the loomux file and does not hand the decision to the .brain.toml
	// beside it.
	for name, body := range map[string]string{
		"empty table": "[area]\n",
		"empty scope": "[area]\nscope = \"\"\n",
	} {
		dir := t.TempDir()
		legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), body)
		legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
		_, err := ReadAreaManifestUntilStage4(dir)
		want := filepath.Join(dir, ".loomux", "config.toml") + ": [area] scope is required and must be a non-empty string"
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", name, err, want)
		}
	}
}

func TestABlankScopeIsAScope(t *testing.T) {
	// `not scope` is false for " ", measured: read_manifest accepts it.
	dir := t.TempDir()
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \" \"\n")
	m, err := ReadAreaManifestUntilStage4(dir)
	if err != nil || m.Scope != " " {
		t.Fatalf("ReadAreaManifestUntilStage4 = %+v, %v; want scope \" \"", m, err)
	}
}

func TestBrokenLegacyTomlNamesTheFile(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".ultra-brain", "config.toml"), "[area\nscope = \"k\"\n")
	_, err := ReadAreaManifestUntilStage4(dir)
	prefix := filepath.Join(dir, ".ultra-brain", "config.toml") + ": not valid TOML: "
	if err == nil || !strings.HasPrefix(err.Error(), prefix) {
		t.Fatalf("err = %v, want the prefix %q", err, prefix)
	}
}

func TestNoNameAtAllIsErrNoManifestNamingAllThree(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadAreaManifestUntilStage4(dir)
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("err = %v, want ErrNoManifest", err)
	}
	want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + ", " +
		filepath.Join(".ultra-brain", "config.toml") + ", .brain.toml)"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestALockedFirstNameIsAnErrorEvenBesideAnOpenLegacyName(t *testing.T) {
	// The first name that is a regular file decides. A locked
	// .loomux/config.toml must not hand the decision to a .brain.toml next to
	// it, which might open an area the locked file closes.
	dir := t.TempDir()
	legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"local_only\"\n")
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"k\"\n")
	testlock.Lock(t, filepath.Join(dir, ".loomux", "config.toml"))
	_, err := ReadAreaManifestUntilStage4(dir)
	if err == nil || errors.Is(err, ErrNoManifest) {
		t.Fatalf("err = %v, want a read error", err)
	}
	if !strings.HasPrefix(err.Error(), filepath.Join(dir, ".loomux", "config.toml")+": cannot be read: ") {
		t.Errorf("err = %q does not name the locked file", err)
	}
}

func TestReadManifestStillKnowsOnlyTheLoomuxName(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
	legacyWrite(t, dir, filepath.Join(".ultra-brain", "config.toml"), "[area]\nscope = \"ultra-brain\"\n")
	_, err := ReadManifest(dir)
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("ReadManifest err = %v, want ErrNoManifest", err)
	}
	if want := dir + ": no manifest found (" + filepath.Join(".loomux", "config.toml") + ")"; err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestReadManifestStillAcceptsAManifestWithoutAScope(t *testing.T) {
	// The scope rule belongs to the brain reader alone; the check chain read
	// such a manifest before this stage and still does.
	dir := t.TempDir()
	write(t, manifestIn(t, dir), "[privacy]\nmode = \"local_only\"\n")
	m, err := ReadManifest(dir)
	if err != nil || m.Scope != "" || m.PrivacyMode != "local_only" {
		t.Fatalf("ReadManifest = %+v, %v", m, err)
	}
}
```

- [ ] **Step 4: Scheitern sehen**

```bash
go test ./internal/config/ -count=1
```
Expected: FAIL beim Bauen, gemessen im Wegwerfmodul (vollständig; der Compiler bricht nach zehn Fehlern ab):
```
# github.com/xidus90/loomux/internal/config [github.com/xidus90/loomux/internal/config.test]
internal\config\legacy_test.go:15:9: undefined: legacyBrainDirUntilStage3For
internal\config\legacy_test.go:22:9: undefined: legacyBrainDirUntilStage3For
internal\config\legacy_test.go:29:9: undefined: legacyBrainDirUntilStage3For
internal\config\legacy_test.go:36:9: undefined: legacyBrainDirUntilStage3For
internal\config\legacy_test.go:45:12: undefined: LegacyBrainDirUntilStage3
internal\config\legacy_test.go:59:13: undefined: legacyBrainDirUntilStage3For
internal\config\legacy_test.go:59:74: undefined: LegacyBrainDirUntilStage3
internal\config\legacy_test.go:60:57: undefined: LegacyBrainDirUntilStage3
internal\config\legacy_test.go:69:9: undefined: LegacyBrainDirUntilStage3
internal\config\legacy_test.go:70:13: undefined: legacyBrainDirUntilStage3For
internal\config\legacy_test.go:70:13: too many errors
FAIL	github.com/xidus90/loomux/internal/config [build failed]
FAIL
```

- [ ] **Step 5: Implementieren**

(a) `internal/config/manifest.go`, Funktion `readManifestAmong` aus Step 1 — drei Stellen:

vorher:
```go
// ReadManifest reads the manifest of the area rooted at repoRoot.
func ReadManifest(repoRoot string) (*Manifest, error) {
	return readManifestAmong(repoRoot, manifestNames)
}

// readManifestAmong reads the first of names below dir that is a regular
// file.
func readManifestAmong(dir string, names []string) (*Manifest, error) {
```
nachher:
```go
// ReadManifest reads the manifest of the area rooted at repoRoot.
func ReadManifest(repoRoot string) (*Manifest, error) {
	return readManifestAmong(repoRoot, manifestNames, false)
}

// readManifestAmong reads the first of names below dir that is a regular
// file. ReadManifest asks for the one loomux name and no scope;
// ReadAreaManifestUntilStage4 asks for three names and a scope, and the
// scope is asked before the mode because `read_manifest`
// (src/brain/manifest.py:19-32) asks in that order -- a file wrong in both
// is reported for its missing scope there.
//
// requireScope expires with stage 4, together with
// ReadAreaManifestUntilStage4, its only caller that sets it. A
// `.loomux/config.toml` without an `[area]` table is policy only and
// declares nothing (stage 1a, R7a), so the scope reader asks the next name
// instead of refusing it; the two old names have no such form, and
// `read_manifest` refuses them without a scope.
func readManifestAmong(dir string, names []string, requireScope bool) (*Manifest, error) {
```

vorher (hinter dem Dekodieren):
```go
		meta, err := toml.Decode(string(data), &file)
		if err != nil {
			return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
		}

		if file.Privacy.Mode != "local_only" && file.Privacy.Mode != "manual_cloud" && file.Privacy.Mode != "automatic_cloud" {
```
nachher:
```go
		meta, err := toml.Decode(string(data), &file)
		if err != nil {
			return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
		}
		if requireScope {
			if name == manifestNames[0] && !meta.IsDefined("area") {
				continue
			}
			if file.Area.Scope == "" {
				return nil, fmt.Errorf("%s: [area] scope is required and must be a non-empty string", path)
			}
		}

		if file.Privacy.Mode != "local_only" && file.Privacy.Mode != "manual_cloud" && file.Privacy.Mode != "automatic_cloud" {
```

(b) `internal/config/legacy.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// legacyBrainDirEnvUntilStage3 points brain/* at another copy of
// ultra-brain's state directory. The recorded cases set it to the staged
// world, the way they set LOOMUX_STATE_DIR; the Python reference reads
// BRAIN_STATE_DIR instead. Expires with LegacyBrainDirUntilStage3.
const legacyBrainDirEnvUntilStage3 = "LOOMUX_LEGACY_BRAIN_DIR"

// manifestNamesUntilStage4 are the names ReadAreaManifestUntilStage4 tries,
// in this order: loomux's own first, then `manifest_path`'s two
// (src/brain/registry.py:132-138), which asks `.ultra-brain/config.toml`
// before `.brain.toml`. Expires with ReadAreaManifestUntilStage4.
var manifestNamesUntilStage4 = []string{
	filepath.Join(".loomux", "config.toml"),
	filepath.Join(".ultra-brain", "config.toml"),
	".brain.toml",
}

// LegacyBrainDirUntilStage3 is ultra-brain's state directory. brain/* reads the artefacts of
// read-only areas and the reconcile stamp from it until stage 3 moves reconcile to Go.
//
// Expires with stage 3: until then the Python side writes `graph.json`,
// `_identities.tsv`, `index.md` of read-only areas and
// `maintenance/last-run.txt` there, and loomux has no writer of its own.
// The registry is not read from here; it stays StateDir's.
func LegacyBrainDirUntilStage3() string {
	if fromEnv := os.Getenv(legacyBrainDirEnvUntilStage3); fromEnv != "" {
		return fromEnv
	}
	home, _ := os.UserHomeDir()
	return legacyBrainDirUntilStage3For(runtime.GOOS, os.Getenv, home)
}

// legacyBrainDirUntilStage3For is `_platform_default`
// (src/brain/paths.py:25-31) with its inputs made arguments, and expires
// with LegacyBrainDirUntilStage3. It asks defaultStateDir and swaps the last
// element, because the two directories differ in nothing but that name:
// both sit in LOCALAPPDATA, home\AppData\Local, XDG_STATE_HOME or
// home/.local/state.
func legacyBrainDirUntilStage3For(goos string, getenv func(string) string, home string) string {
	return filepath.Join(filepath.Dir(defaultStateDir(goos, getenv, home)), "brain")
}

// ReadAreaManifestUntilStage4 reads an area's manifest under the name loomux writes, or under
// one of ultra-brain's names until stage 4 moves the hosts: .loomux/config.toml, else
// .ultra-brain/config.toml, else .brain.toml. Only brain/* calls it.
//
// Expires with stage 4, when the area repositories carry
// `.loomux/config.toml`. Until then a `local_only` area that still declares
// itself in `.brain.toml` would be read as having no manifest, and its
// privacy would go unseen. The write barrier and ReadManifest stay with the
// one loomux name.
//
// Like `read_manifest` (src/brain/manifest.py:19-24) it refuses a manifest
// without a non-empty `[area] scope`; the other checks Python makes there
// are not repeated. A `.loomux/config.toml` without an `[area]` table
// declares nothing, and the next name is asked.
func ReadAreaManifestUntilStage4(dir string) (*Manifest, error) {
	return readManifestAmong(dir, manifestNamesUntilStage4, true)
}
```

- [ ] **Step 6: Bestehen sehen**

```bash
go test ./internal/config/ -count=1
```
Expected: `ok  	github.com/xidus90/loomux/internal/config` (im Wegwerfmodul 0,35–0,53 s).

- [ ] **Step 7: Coverage**

```bash
go test ./internal/config/ -count=1 -covermode=set -coverpkg=./internal/config/ -coverprofile="$TEMP/pkg.out"
```
Expected: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/config` beginnt und mit `coverage: 100.0% of statements in ./internal/config/` endet (gemessen im Wegwerfmodul).
```bash
go tool cover -func="$TEMP/pkg.out"
```
Erwartet: jede Funktion des Pakets 100.0%. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund. Gemessen im Wegwerfmodul: keine Zeile unter `100.0%`, darunter `LegacyBrainDirUntilStage3`, `legacyBrainDirUntilStage3For`, `ReadAreaManifestUntilStage4`, `ReadManifest` und `readManifestAmong`; den Arm `continue` für die Policy-Datei deckt `TestAPolicyOnlyLoomuxManifestDeclaresNothingForBrain`.

- [ ] **Step 8: Format und vet**

```bash
gofmt -l internal/config
```
Expected: keine Ausgabe.
```bash
go vet ./internal/config/
```
Expected: keine Ausgabe.

- [ ] **Step 9: Stagen und Tor**

```bash
git add internal/config/legacy.go internal/config/legacy_test.go internal/config/manifest.go
```
Expected: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Expected: Exit 0, keine Zeile `not covered:` von `covergate`, keine Zeile `pre-commit: inputs differ from the index`.

- [ ] **Step 10: Vor dem Commit prüfen**

```bash
git branch --show-current
```
Expected: `sdd-1b-1` (oder `master`, wenn Task 0 den Hauptcheckout gewählt hat).
```bash
git rev-parse --short HEAD
```
Expected: der Hash, den Task 0 in Step 9 berichtet — Task 0 committet nichts, HEAD steht also noch auf dem Stand, den Task 0 Step 3 mit `master` verglichen hat.
```bash
git diff --cached --stat
```
Expected: genau die drei Dateien `internal/config/legacy.go`, `internal/config/legacy_test.go`, `internal/config/manifest.go`, dann die Summenzeile `3 files changed`.

- [ ] **Step 11: Commit**

```bash
printf '%s\n' 'Read the old manifest names and brain state directory until stages 3 and 4' '' 'brain/* reads ultra-brain artefacts of read-only areas and the reconcile' 'stamp from the old state directory until stage 3, and the manifest names' '.ultra-brain/config.toml and .brain.toml until stage 4. ReadManifest and' 'the write barrier keep the one loomux name.' > "$TEMP/loomux-1b1-task1-msg.txt"
```
Expected: keine Ausgabe. (Die Nachricht hat `bin/loomux.exe check commit-msg` am 2026-09-15 mit Exit 0 passiert.)
```bash
git commit -F "$TEMP/loomux-1b1-task1-msg.txt"
```
Expected: Exit 0. Der Commit löst das Tor noch einmal aus; das ist gewollt. Danach prüft `.githooks/commit-msg` die Nachricht mit `bin/loomux.exe check commit-msg`, und Git meldet den Commit mit der Betreffzeile `Read the old manifest names and brain state directory until stages 3 and 4`.
```bash
git log -1 --format='%an <%ae>'
```
Expected: `Christoph Wübbels <christoph.wuebbels@gmail.com>` (Autor der Commits `e4bf7b5`, `311d5b2`, `5f3cb4c`).
```bash
git log -1 --format=%B
```
Expected: genau die sechs Zeilen aus `$TEMP/loomux-1b1-task1-msg.txt` und eine Leerzeile, keine Zeile `Co-Authored-By`.

- [ ] **Step 12: Bericht**

Der Bericht nennt:
- Umzugsbefunde: keine (kein Umzug). Geändert außerhalb von `legacy*.go` nur `manifest.go`, `ReadManifest` → `readManifestAmong`, belegt durch unveränderte Tests in `manifest_test.go` und `registry_test.go`.
- Ablauf im Namen: Jeder Name der befristeten Ausnahme trägt seine Stufe — `LegacyBrainDirUntilStage3`, `legacyBrainDirEnvUntilStage3`, `legacyBrainDirUntilStage3For`, `ReadAreaManifestUntilStage4`, `manifestNamesUntilStage4`; der Parameter `requireScope` von `readManifestAmong` entfällt laut seinem Kommentar mit Stufe 4. Stufe 3 entfernt außerdem `search.ReadLastRun(legacyDir)` und jedes `legacyDir`-Argument in `brain/*`.
- Policy-Datei ohne `[area]`: Eine `.loomux/config.toml` ohne `[area]`-Tabelle deklariert für `brain/*` nichts (wie Guard-Regel R7a); `ReadAreaManifestUntilStage4` geht zum nächsten Altnamen, und gibt es keinen, folgt `ErrNoManifest` mit allen drei Namen (`TestAPolicyOnlyLoomuxManifestDeclaresNothingForBrain`, `TestAPolicyOnlyLoomuxManifestAloneIsErrNoManifest`).
- `[area]` mit leerem Scope in `.loomux/config.toml`: Scope-Fehler dieser Datei, auch neben einer gültigen `.brain.toml` (`TestAnAreaTableWithoutAScopeInTheLoomuxManifestIsRefused`, Fälle leere Tabelle und `scope = ""`). Keine Paritätszeile für beide Punkte: Python kennt den Namen `.loomux/config.toml` nicht.
- Paritätszeilen, die dieser Task schafft (Task 12 trägt sie ein):
  1. Ein Bereich mit `.loomux/config.toml` neben einem Altmanifest: loomux liest `.loomux/config.toml`, wenn sie eine `[area]`-Tabelle trägt (ohne sie deklariert die Datei nichts, und loomux liest das Altmanifest wie Python); Python kennt diesen Namen nicht und liest `.ultra-brain/config.toml` bzw. `.brain.toml`.
  2. Das Legacy-Zustandsverzeichnis folgt `LOOMUX_LEGACY_BRAIN_DIR`, nicht `BRAIN_STATE_DIR` und nicht `--state-dir` der Referenz; ein leeres `XDG_STATE_HOME` ergibt unter POSIX `home/.local/state/brain`, Python den relativen Pfad `brain`.
  3. Manifest mit BOM: `tomllib` verweigert es (`not valid TOML: Invalid statement (at line 1, column 1)`), loomux liest es.
  4. `[area] scope` mit Nicht-Zeichenkette (`scope = 3`): Python meldet `[area] scope is required and must be a non-empty string`, loomux einen TOML-Typfehler (`not valid TOML: toml: line 2 (last key "area.scope"): incompatible types: TOML value has type int64; destination has type string`).
  5. Wortlaut eines kaputten Manifests nach `not valid TOML: `: Python den von `tomllib`, loomux den von BurntSushi toml v1.6.0.
  6. Ungültiges UTF-8 im Manifest: Python bricht mit `UnicodeDecodeError`-Traceback ab, loomux meldet `{path}: not valid TOML: toml: line 2 (last key "area.scope"): invalid UTF-8 byte: 0xff` (gemessen mit dem Byte im Wert von `scope`).
  7. Die übrigen Prüfungen von `read_manifest`, die `ReadAreaManifestUntilStage4` nicht macht: `[wiki] types` als Liste von Zeichenketten, `[wiki] untouched_days` ≥ 1, `[maintenance] on_merge`/`branch`, `[model]`; Glob-Listen mit falschem Typ scheitern in loomux als TOML-Fehler statt mit `must be an array of strings` bzw. `contains a non-string`.
  8. Wortlaut der Modusmeldung: loomux zitiert mit Go-`%q` (`found "bogus"`), Python mit `repr` (`found 'bogus'`) — `config` darf `brain/pytext` nicht importieren.
  9. `.brain.toml` als Verzeichnis und kein anderer Name: Python scheitert beim Lesen (`[Errno 13] Permission denied`), loomux antwortet `ErrNoManifest`.

---

### Task 2: `internal/brain/pytext` — Pythons Schreibweisen

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Task 1 ist committet (dieser Task berührt dessen Dateien nicht).

**Files:**
- Create: `internal/brain/pytext/unicode.go`, `internal/brain/pytext/repr.go`, `internal/brain/pytext/lines.go`, `internal/brain/pytext/text.go`, `internal/brain/pytext/isotime.go`, `internal/brain/pytext/path.go`
- Test: `internal/brain/pytext/digest_test.go` (neu, Ergänzung: Prüfsummen über den ganzen Unicode-Bereich), `internal/brain/pytext/unicode_test.go`, `internal/brain/pytext/repr_test.go`, `internal/brain/pytext/lines_test.go`, `internal/brain/pytext/text_test.go`, `internal/brain/pytext/isotime_test.go`, `internal/brain/pytext/path_test.go`
- Modify: `go.mod`, `go.sum` (`go get golang.org/x/text@v0.38.0`, danach `go mod tidy`, siehe Step 3 und Step 6)

**Interfaces:**
- Consumes: `golang.org/x/text/cases` (`cases.Fold() Caser`, `cases.UnicodeVersion`), `golang.org/x/text/unicode/norm` (`norm.NFC`, `norm.Version`), Standardbibliothek. Nichts aus loomux.
- Produces (Vertrag, wörtlich):
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
- Produces (Ergänzungen, alle unexportiert):
```go
const hexDigits = "0123456789abcdef"                                       // repr.go
func writeHex(b *strings.Builder, prefix string, v uint32, width int)     // repr.go
func isLineBoundary(r rune) bool                                          // lines.go
func shape(s, pattern string) bool                                        // isotime.go
func number(digits string) int                                            // isotime.go
func posixPath(p string) string                                           // path.go
func windowsPath(p string) string                                         // path.go
func windowsSplitRoot(p string) (drive, root, rel string)                 // path.go
func pathParts(rel, sep string) []string                                  // path.go
func orDot(s string) string                                               // path.go
func cherokeeCapital(r rune) rune                                         // unicode.go
// nur in Tests (digest_test.go):
var newInUnicode17 [][2]rune                                              // the 47 ranges written out in Step 1: the 4803 code points Unicode 17 assigns and Python 3.14 does not know
func isNewInUnicode17(r rune) bool
func rangeDigest(f func(string) string, skipNew bool) string
```

Die Aufrufer in den Entwürfen der Tasks 3–10 (gegrept nach `pytext.`) benutzen genau die Vertragssignaturen: `Repr`, `SplitLines`, `Strip`, `RStrip`, `ReadText`, `IsoFormat`, `ParseAwareIsoFormat`, `PathString(runtime.GOOS, p)`, `NFC`, `CaseFold(NFC(s))`. `IsSpace` ruft keiner direkt; `Strip`/`RStrip` bauen darauf.

**Arbeitsort:** `SRC=/c/Users/micro/Documents/#GIT/loomux-src/ub`. Torvoraussetzung ab diesem Task: Go 1.27 (Task 0 Step 2 prüft sie), weil `TestTheTablesAreTheOnesMeasured` die Unicode-17.0.0-Tabellen verlangt.

**Reihenfolge der Dateien:** `unicode` zuerst, weil `digest_test.go` das Modul `x/text` braucht und seine Prüfsummenschleife `rangeDigest` von `repr_test.go` benutzt wird. Danach `repr`, `lines`, `text`, `isotime`, `path`.

**Warum die erwarteten Zeichenketten in `repr_test.go` `bs + "u200b"` schreiben:** Pythons Vier-Stellen-Escape als Go-Literal `` `\u200b` `` wäre korrekt, aber ein Werkzeug zwischen Plan und Datei, das `\u200b` als Escape liest, machte daraus das Zeichen selbst — und der Test prüfte dann das Gegenteil. Die Verkettung übersteht jede solche Stelle.

**Gemessen an der Referenz (Python 3.14.7, `unicodedata.unidata_version` 16.0.0; Go 1.27.0 mit `unicode.Version`, `norm.Version`, `cases.UnicodeVersion` je 17.0.0).** Step 29 wiederholt die Messbefehle; ihre Ausgaben stehen dort wörtlich.

| Funktion | Befund |
|---|---|
| `Repr` | über alle Codepunkte außer Surrogaten gleich bis auf die 4803 Codepunkte, die Unicode 17 neu vergibt (in Python Kategorie `Cn`, also escaped; in Go druckbar) — 47 Bereiche, alle in `digest_test.go` |
| `SplitLines`, `IsSpace`, `Strip`, `RStrip` | exakt: 10 Trenner, 29 Leerzeichen, über den ganzen Bereich verglichen |
| `NFC` | exakt über den ganzen Bereich, auch NFD (gemessen im Wegwerfmodul: `nfc diffs 0`, `nfd diffs 0`) |
| `CaseFold` | `cases.Fold()` weicht an 114 Codepunkten ab: 86 Cherokee-Zeichen (x/text faltet U+13A0 bis U+13EF zu U+AB70 bis U+ABBF und U+13F0 bis U+13F5 zu U+13F8 bis U+13FD, Python zu den Großbuchstaben) und 28 Unicode-17-Buchstaben (`U+A7CE`, `U+A7D2`, `U+A7D4`, `U+16EA0` bis `U+16EB8`). `cherokeeCapital` repariert die 86; die 28 sind Paritätszeile |
| `ReadText` | `\r\n` und einzelnes `\r` → `\n`; `\x85` bleibt; BOM bleibt; ungültiges UTF-8 und kodierte Surrogate → `UnicodeDecodeError` |
| `IsoFormat` | Mikrosekunden nur wenn ≠ 0; Offset-Sekunden nur wenn ≠ 0 |
| `ParseAwareIsoFormat` | 22 Formen aware angenommen, 26 abgelehnt oder naiv — gleich; 14 Formen nimmt nur Python an (Paritätszeile) |
| `PathString` | 36 Windows- und 14 POSIX-Fälle gleich |

**Startzeit (gemessen, `GODEBUG=inittrace=1` auf einem Probe-Binary mit `cases.Fold` und `norm.NFC`):** `init golang.org/x/text/unicode/norm @0.47 ms, 0 ms clock, 0 bytes, 0 allocs`, `init golang.org/x/text/language @0.99 ms, 0 ms clock, 256 bytes, 3 allocs`, `init golang.org/x/text/cases @0.99 ms, 0 ms clock, 2600 bytes, 38 allocs`; dazu `golang.org/x/text/internal/language` mit 4112 Bytes und 8 Allokationen und `golang.org/x/text/internal/language/compact` mit 560 Bytes und 8 Allokationen (deren Zeitfelder sind nicht mitgeschrieben worden; sie gehen in die Messung von Task 15 am echten Binary ein). `pytext` selbst hat kein `init()` und keine Paketvariable, die Daten parst; die Tabellen von `newInUnicode17` stehen nur im Test.

- [ ] **Step 1: Failing tests schreiben — Unicode-Tabellen, NFC, casefold**

`internal/brain/pytext/digest_test.go`:
```go
package pytext

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// newInUnicode17 are the code points Go's `unicode.IsPrint` calls printable
// and Python 3.14.7 does not: all 4803 are category Cn in Python's Unicode
// 16 tables and assigned in Unicode 17. Measured by listing
// `not chr(c).isprintable()` from U+007F up in Python, comparing in Go and
// compressing the differences to ranges.
var newInUnicode17 = [][2]rune{
	{0x88F, 0x88F}, {0xC5C, 0xC5C}, {0xCDC, 0xCDC}, {0x1ACF, 0x1ADD}, {0x1AE0, 0x1AEB},
	{0x20C1, 0x20C1}, {0x2B96, 0x2B96}, {0xA7CE, 0xA7CF}, {0xA7D2, 0xA7D2}, {0xA7D4, 0xA7D4},
	{0xA7F1, 0xA7F1}, {0xFBC3, 0xFBD2}, {0xFD90, 0xFD91}, {0xFDC8, 0xFDCE}, {0x10940, 0x10959},
	{0x10EC5, 0x10EC7}, {0x10ED0, 0x10ED8}, {0x10EFA, 0x10EFB}, {0x11B60, 0x11B67}, {0x11DB0, 0x11DDB},
	{0x11DE0, 0x11DE9}, {0x16EA0, 0x16EB8}, {0x16EBB, 0x16ED3}, {0x16FF2, 0x16FF6}, {0x187F8, 0x187FF},
	{0x18D09, 0x18D1E}, {0x18D80, 0x18DF2}, {0x1CCFA, 0x1CCFC}, {0x1CEBA, 0x1CED0}, {0x1CEE0, 0x1CEF0},
	{0x1E6C0, 0x1E6DE}, {0x1E6E0, 0x1E6F5}, {0x1E6FE, 0x1E6FF}, {0x1F6D8, 0x1F6D8}, {0x1F777, 0x1F77A},
	{0x1F8D0, 0x1F8D8}, {0x1FA54, 0x1FA57}, {0x1FA8A, 0x1FA8A}, {0x1FA8E, 0x1FA8E}, {0x1FAC8, 0x1FAC8},
	{0x1FACD, 0x1FACD}, {0x1FAEA, 0x1FAEA}, {0x1FAEF, 0x1FAEF}, {0x1FBFA, 0x1FBFA}, {0x2B73A, 0x2B73F},
	{0x2CEA2, 0x2CEAD}, {0x323B0, 0x33479},
}

func isNewInUnicode17(r rune) bool {
	for _, span := range newInUnicode17 {
		if r >= span[0] && r <= span[1] {
			return true
		}
	}
	return false
}

// rangeDigest is sha256 over f(string(r)) followed by a zero byte for every
// code point but the surrogates, and without the Unicode 17 additions when
// skipNew is set -- the loop the Python measurement runs.
func rangeDigest(f func(string) string, skipNew bool) string {
	h := sha256.New()
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff || skipNew && isNewInUnicode17(r) {
			continue
		}
		io.WriteString(h, f(string(r)))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestTheTablesAreTheOnesMeasured(t *testing.T) {
	// The digests in this package were compared against Unicode 17 tables,
	// which Go 1.27 brings; go.mod's `go 1.25.0` lets an older toolchain
	// build the gate, and this stops it with the reason instead of a wrong
	// digest. A Go or x/text with other tables must be measured against the
	// reference again, not have its digests copied in.
	if unicode.Version != "17.0.0" || norm.Version != "17.0.0" || cases.UnicodeVersion != "17.0.0" {
		t.Fatalf("pytext tables are measured against Unicode 17.0.0 (Go 1.27); re-measure for unicode %s, norm %s, cases %s against Python 3.14 of the reference",
			unicode.Version, norm.Version, cases.UnicodeVersion)
	}
	count := 0
	for _, span := range newInUnicode17 {
		count += int(span[1]-span[0]) + 1
	}
	if count != 4803 {
		t.Errorf("newInUnicode17 holds %d code points, measured 4803", count)
	}
}
```

`internal/brain/pytext/unicode_test.go`:
```go
package pytext

import "testing"

func TestNFCLikePython(t *testing.T) {
	// unicodedata.normalize("NFC", s) on Python 3.14.7.
	cases := []struct{ in, want string }{
		{"e\U00000301", "\U000000e9"},
		{"\U0000212b", "\U000000c5"},
		{"a\U00000301\U00000328", "\U00000105\U00000301"},
		{"\U00001e9b\U00000323", "\U00001e9b\U00000323"},
		{"\U0000ac00\U000011a8", "\U0000ac01"},
		{"\U00000958", "\U00000915\U0000093c"},
	}
	for _, c := range cases {
		if got := NFC(c.in); got != c.want {
			t.Errorf("NFC(%+q) = %+q, want %+q", c.in, got, c.want)
		}
	}
}

func TestNFCMatchesPythonOverTheWholeRange(t *testing.T) {
	// sha256 over unicodedata.normalize("NFC", chr(c)) + "\0" for every code
	// point but the surrogates; x/text differs from Python on none of them.
	const python = "6e0aaa17a82be5b6f410942f1f99ae77832fb630e98780c6eaf6779ab278235d"
	if got := rangeDigest(NFC, false); got != python {
		t.Errorf("digest %s, want %s", got, python)
	}
}

func TestCaseFoldLikePython(t *testing.T) {
	// s.casefold() on Python 3.14.7.
	cases := []struct{ in, want string }{
		{"\U000000df", "ss"},
		{"\U00001e9e", "ss"},
		{"\U0000fb01", "fi"},
		{"\U000003a3", "\U000003c3"},
		{"\U000003c2", "\U000003c3"},
		{"\U00000130", "i\U00000307"},
		{"\U00000149", "\U000002bcn"},
		{"\U000001f1", "\U000001f3"},
		{"ABC", "abc"},
		{"\U000000df.md", "ss.md"},
		{"\U000003a3\U00000391\U000003a3", "\U000003c3\U000003b1\U000003c3"},
		{"\U00000130x", "i\U00000307x"},
		{"\U0000fb01le.MD", "file.md"},
		{"Stra\U000000dfe", "strasse"},
		{"\U000001c5", "\U000001c6"},
		{"\U00001f88", "\U00001f00\U000003b9"},
		{"A\U0000030a", "a\U0000030a"},
		{"\U000013a0\U0000ab70\U000013f0\U000013f8", "\U000013a0\U000013a0\U000013f0\U000013f0"},
	}
	for _, c := range cases {
		if got := CaseFold(c.in); got != c.want {
			t.Errorf("CaseFold(%+q) = %+q, want %+q", c.in, got, c.want)
		}
	}
}

func TestCaseFoldMatchesPythonOverTheWholeRange(t *testing.T) {
	// sha256 over chr(c).casefold() + "\0" for every code point but the
	// surrogates and the Unicode 17 additions. Without the Cherokee repair in
	// CaseFold this digest differs.
	const python = "10e93868ae4b30b2d8d735f20f016cae6203cd34bad840c9a7037911cef978d8"
	if got := rangeDigest(CaseFold, true); got != python {
		t.Errorf("digest %s, want %s", got, python)
	}
}
```

- [ ] **Step 2: Scheitern sehen — die Abhängigkeit fehlt**

```bash
go test ./internal/brain/pytext/
```
Expected (gemessen in einer Kopie des committeten Stands; seit `311d5b2` nur Doku geändert):
```
# github.com/xidus90/loomux/internal/brain/pytext
internal\brain\pytext\digest_test.go:10:2: no required module provides package golang.org/x/text/cases; to add it:
	go get golang.org/x/text/cases
FAIL	github.com/xidus90/loomux/internal/brain/pytext [setup failed]
FAIL
```

- [ ] **Step 3: `golang.org/x/text` holen — offline aus dem Modulcache**

```bash
GOPROXY=off go get golang.org/x/text@v0.38.0
```
Expected: `go: added golang.org/x/text v0.38.0`.
```bash
git diff go.mod go.sum
```
Expected (gemessen): `go.mod` bekommt am Ende einen eigenen Block `require golang.org/x/text v0.38.0 // indirect` — so schreibt `go get` die Zeile, gemessen sowohl mit nur Testimporten als auch mit einem Import in Produktionscode; `go 1.25.0` bleibt, kein `toolchain`-Eintrag. `go.sum` bekommt genau zwei Zeilen:
```
golang.org/x/text v0.38.0 h1:sXmwo9DwP3OK9EZ7PqAdaooSGozfl/3a6/xJcbzPRhE=
golang.org/x/text v0.38.0/go.mod h1:YXZt3QhHUKYT53r2lLKFIVi6Ao1jdzrTR/KQ09qyxF4=
```
(`x/text` v0.38.0 verlangt selbst `go 1.25.0`; seine `require`-Zeilen auf `x/tools`, `x/mod`, `x/sync` ziehen dank Modulgraph-Beschneidung nichts nach.)

- [ ] **Step 4: Scheitern sehen — die Funktionen fehlen**

```bash
go test ./internal/brain/pytext/
```
Expected (gemessen):
```
# github.com/xidus90/loomux/internal/brain/pytext [github.com/xidus90/loomux/internal/brain/pytext.test]
internal\brain\pytext\unicode_test.go:16:13: undefined: NFC
internal\brain\pytext\unicode_test.go:26:24: undefined: NFC
internal\brain\pytext\unicode_test.go:54:13: undefined: CaseFold
internal\brain\pytext\unicode_test.go:65:24: undefined: CaseFold
FAIL	github.com/xidus90/loomux/internal/brain/pytext [build failed]
FAIL
```

- [ ] **Step 5: Implementieren** — `internal/brain/pytext/unicode.go`

```go
package pytext

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// NFC is `unicodedata.normalize("NFC", s)`. Measured over every code point
// against Python 3.14: x/text answers the same for all of them, both in NFC
// and NFD, though its tables are Unicode 17 and Python's 16.
func NFC(s string) string {
	return norm.NFC.String(s)
}

// CaseFold is `s.casefold()`: full case folding, so `ß` becomes `ss`.
//
// x/text's Fold folds Cherokee towards its small letters (U+13A0 to
// U+AB70, U+13F0 to U+13F8), where CaseFolding.txt and Python fold towards
// the capitals -- 86 code points, measured. The capitals are put back
// after folding. The 28 other differences are letters Unicode 17 added,
// which Python 3.14 does not know.
func CaseFold(s string) string {
	return strings.Map(cherokeeCapital, cases.Fold().String(s))
}

// cherokeeCapital maps a small Cherokee letter to its capital and leaves
// every other rune alone.
func cherokeeCapital(r rune) rune {
	switch {
	case r >= 0xab70 && r <= 0xabbf:
		return r - 0xab70 + 0x13a0
	case r >= 0x13f8 && r <= 0x13fd:
		return r - 0x13f8 + 0x13f0
	}
	return r
}
```

- [ ] **Step 6: `go.mod` aufräumen**

`go mod tidy` nimmt das `// indirect` weg (gemessen: auch schon, wenn nur die Tests `x/text` importieren; der Schritt steht hier, damit `go.mod` einmal und endgültig angefasst wird).
```bash
GOPROXY=off go mod tidy
```
Expected: keine Ausgabe, Exit 0.
```bash
git diff go.mod
```
Expected (gemessen in einer Kopie des committeten Stands mit diesem Paket): gegenüber dem Stand vor Step 3 nur die angehängte Zeile
```
require golang.org/x/text v0.38.0
```
`BurntSushi/toml v1.6.0`, `x/sys v0.18.0`, `yaml.v3 v3.0.1` und `go 1.25.0` unverändert, kein `toolchain`-Eintrag; `go.sum` wie nach Step 3. Senkt `tidy` eine Version oder fügt es etwas anderes hinzu: `git checkout go.mod go.sum`, Step 3 wiederholen, ohne `tidy` weiter (das `// indirect` bleibt dann stehen) und Befund an den Controller.

- [ ] **Step 7: Bestehen sehen**

```bash
go test ./internal/brain/pytext/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/pytext`. Ersetzt man `strings.Map(cherokeeCapital, cases.Fold().String(s))` durch `cases.Fold().String(s)`, scheitert `TestCaseFoldLikePython` an der Cherokee-Zeile und `TestCaseFoldMatchesPythonOverTheWholeRange` an der Prüfsumme (gemessen).

- [ ] **Step 8: Failing tests schreiben — `repr`** — `internal/brain/pytext/repr_test.go`

```go
package pytext

import "testing"

// bs is one backslash. The expected texts spell Python's four-digit escape
// as bs + "u...", so that nothing between the plan and the file can turn
// the escape into the character it names.
const bs = "\\"

func TestReprQuotesAndEscapesLikePython(t *testing.T) {
	// Every right-hand side was printed by Python 3.14.7 of the reference:
	// print(ascii(s), "=>", ascii(repr(s))).
	cases := []struct{ in, want string }{
		{"x", `'x'`},
		{"it's", `"it's"`},
		{`say "hi"`, `'say "hi"'`},
		{`both ' and "`, `'both \' and "'`},
		{"a\tb", `'a\tb'`},
		{"\x00", `'\x00'`},
		{"\x7f", `'\x7f'`},
		{"\U000000e9", "'\U000000e9'"},
		{"\U0000200b", "'" + bs + "u200b'"},
		{"\U0001f600", "'\U0001f600'"},
		{`a\b`, `'a\\b'`},
		{`'\`, `"'\\"`},
		{`"\'`, `'"\\\''`},
		{"a\rb\nc", `'a\rb\nc'`},
		{"", `''`},
		{"\U00000080", `'\x80'`},
		{"\U000000a0", `'\xa0'`},
		{"\U000000ad", `'\xad'`},
		{"\U000000ff", "'\U000000ff'"},
		{"\U00000100", "'\U00000100'"},
		{"\U00002028", "'" + bs + "u2028'"},
		{"\U0000feff", "'" + bs + "ufeff'"},
		{"\U0000ffff", "'" + bs + "uffff'"},
		{"\U000e0001", `'\U000e0001'`},
		{"\U0010ffff", `'\U0010ffff'`},
		{" ", `' '`},
		{"\x1b", `'\x1b'`},
		{"\U00000378", "'" + bs + "u0378'"},
		{"\U0001fae9", "'\U0001fae9'"},
	}
	for _, c := range cases {
		if got := Repr(c.in); got != c.want {
			t.Errorf("Repr(%+q) = %+q, want %+q", c.in, got, c.want)
		}
	}
}

func TestReprWritesAByteThatIsNotUTF8AsHex(t *testing.T) {
	if got, want := Repr("a\xffb"), `'a\xffb'`; got != want {
		t.Errorf("Repr = %s, want %s", got, want)
	}
}

func TestReprMatchesPythonOverTheWholeRange(t *testing.T) {
	// sha256 over repr(chr(c)) + "\0" for every code point but the
	// surrogates and the 4803 that Unicode 17 assigns.
	const python = "61bb4be799689f2c003eb8932471a2f8d2b63c3f74c9c313a461037097a31cc9"
	if got := rangeDigest(Repr, true); got != python {
		t.Errorf("digest %s, want %s", got, python)
	}
}
```

- [ ] **Step 9: Scheitern sehen**

```bash
go test ./internal/brain/pytext/
```
Expected (gemessen):
```
# github.com/xidus90/loomux/internal/brain/pytext [github.com/xidus90/loomux/internal/brain/pytext.test]
internal\brain\pytext\repr_test.go:45:13: undefined: Repr
internal\brain\pytext\repr_test.go:52:18: undefined: Repr
internal\brain\pytext\repr_test.go:61:24: undefined: Repr
FAIL	github.com/xidus90/loomux/internal/brain/pytext [build failed]
FAIL
```

- [ ] **Step 10: Implementieren** — `internal/brain/pytext/repr.go`

```go
// Package pytext writes and reads text the way Python 3.14 of the
// ultra-brain reference does: repr, splitlines, strip, read_text,
// isoformat and fromisoformat, str(Path), NFC and casefold. Five brain
// packages print or compare such text, and each must agree with the
// reference byte for byte.
package pytext

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const hexDigits = "0123456789abcdef"

// Repr is `repr(s)` of a str. The quote is `'` unless the text holds a `'`
// and no `"`; the chosen quote and the backslash are escaped, `\t` `\n` `\r`
// keep their short forms, and every other character Python does not call
// printable becomes `\xhh`, `\uhhhh` or `\Uhhhhhhhh` by size.
//
// Printable is `unicode.IsPrint` above ASCII, which is Python's rule
// (letters, marks, numbers, punctuation, symbols, and the space) over
// Go's Unicode 17 tables; Python 3.14 has Unicode 16, so the 4803 code
// points 17 assigns print here and are escaped there.
//
// A byte that is not UTF-8 has no Python counterpart -- a str never holds
// one -- and is written as `\xhh`.
func Repr(s string) string {
	quote := byte('\'')
	if strings.IndexByte(s, '\'') >= 0 && strings.IndexByte(s, '"') < 0 {
		quote = '"'
	}
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte(quote)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			writeHex(&b, `\x`, uint32(s[i]), 2)
			i++
			continue
		}
		i += size
		switch {
		case r == rune(quote) || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r < ' ' || r == 0x7f:
			writeHex(&b, `\x`, uint32(r), 2)
		case r < 0x7f || unicode.IsPrint(r):
			b.WriteRune(r)
		case r <= 0xff:
			writeHex(&b, `\x`, uint32(r), 2)
		case r <= 0xffff:
			writeHex(&b, `\u`, uint32(r), 4)
		default:
			writeHex(&b, `\U`, uint32(r), 8)
		}
	}
	b.WriteByte(quote)
	return b.String()
}

// writeHex writes prefix and width lower-case hex digits of v.
func writeHex(b *strings.Builder, prefix string, v uint32, width int) {
	b.WriteString(prefix)
	for shift := (width - 1) * 4; shift >= 0; shift -= 4 {
		b.WriteByte(hexDigits[v>>uint(shift)&0xf])
	}
}
```

- [ ] **Step 11: Bestehen sehen**

```bash
go test ./internal/brain/pytext/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/pytext`.

- [ ] **Step 12: Failing tests schreiben — `splitlines`** — `internal/brain/pytext/lines_test.go`

```go
package pytext

import (
	"slices"
	"testing"
	"unicode"
)

func TestSplitLinesLikePython(t *testing.T) {
	// print(ascii(s), ascii(s.splitlines())) on Python 3.14.7.
	cases := []struct {
		in   string
		want []string
	}{
		{"a\n", []string{"a"}},
		{"\n", []string{""}},
		{"", nil},
		{"a\r\nb", []string{"a", "b"}},
		{"\r\r\n", []string{"", ""}},
		{"a\n\rb", []string{"a", "", "b"}},
		{"\n\n", []string{"", ""}},
		{"a", []string{"a"}},
		{"a\U00000085b\U00002028c\U00002029d\x0be\x0cf\x1cg\x1dh\x1ei\rj\r\nk\nl",
			[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}},
	}
	for _, c := range cases {
		if got := SplitLines(c.in); !slices.Equal(got, c.want) {
			t.Errorf("SplitLines(%+q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplitLinesSplitsOnTheTenBoundariesAndNothingElse(t *testing.T) {
	// [hex(c) for c in range(0x110000) if not 0xd800<=c<=0xdfff
	//  and len(('a'+chr(c)+'b').splitlines())==2] on Python 3.14.7.
	boundaries := map[rune]bool{0x0a: true, 0x0b: true, 0x0c: true, 0x0d: true, 0x1c: true,
		0x1d: true, 0x1e: true, 0x85: true, 0x2028: true, 0x2029: true}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		if split := len(SplitLines("a"+string(r)+"b")) == 2; split != boundaries[r] {
			t.Errorf("U+%04X splits = %v, Python says %v", r, split, boundaries[r])
		}
	}
}
```

- [ ] **Step 13: Scheitern sehen**

```bash
go test ./internal/brain/pytext/
```
Expected (gemessen):
```
# github.com/xidus90/loomux/internal/brain/pytext [github.com/xidus90/loomux/internal/brain/pytext.test]
internal\brain\pytext\lines_test.go:27:13: undefined: SplitLines
internal\brain\pytext\lines_test.go:42:19: undefined: SplitLines
FAIL	github.com/xidus90/loomux/internal/brain/pytext [build failed]
FAIL
```

- [ ] **Step 14: Implementieren** — `internal/brain/pytext/lines.go`

```go
package pytext

import "unicode/utf8"

// SplitLines is `s.splitlines()`: the ten line boundaries Python knows,
// `\r\n` as one, no line ends kept. An empty text has no lines, and a
// boundary at the very end opens no empty last line.
func SplitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !isLineBoundary(r) {
			i += size
			continue
		}
		lines = append(lines, s[start:i])
		i += size
		if r == '\r' && i < len(s) && s[i] == '\n' {
			i++
		}
		start = i
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// isLineBoundary is the set `splitlines` splits on, measured over the whole
// range against Python 3.14: \n \v \f \r, the three separators \x1c-\x1e,
// NEL, LINE SEPARATOR and PARAGRAPH SEPARATOR. \x1f is space to Python but
// no boundary.
func isLineBoundary(r rune) bool {
	switch r {
	case '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}
```

- [ ] **Step 15: Bestehen sehen**

```bash
go test ./internal/brain/pytext/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/pytext`.

- [ ] **Step 16: Failing tests schreiben — `isspace`, `strip`, `rstrip`, `read_text`** — `internal/brain/pytext/text_test.go`

```go
package pytext

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"unicode"
)

func TestIsSpaceIsPythonsSetOverTheWholeRange(t *testing.T) {
	// [hex(c) for c in range(0x110000) if chr(c).isspace()] on Python 3.14.7.
	python := map[rune]bool{}
	for _, r := range []rune{0x9, 0xa, 0xb, 0xc, 0xd, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x85, 0xa0,
		0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007, 0x2008, 0x2009,
		0x200a, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000} {
		python[r] = true
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if IsSpace(r) != python[r] {
			t.Errorf("IsSpace(U+%04X) = %v, Python says %v", r, IsSpace(r), python[r])
		}
	}
}

func TestStripAndRStripLikePython(t *testing.T) {
	// s.strip() and s.rstrip() on Python 3.14.7.
	cases := []struct{ in, strip, rstrip string }{
		{"  a  ", "a", "  a"},
		{"\x1c a \x1f", "a", "\x1c a"},
		{"\U000000a0a\U00003000", "a", "\U000000a0a"},
		{"\U0000200ba\U0000200b", "\U0000200ba\U0000200b", "\U0000200ba\U0000200b"},
		{"\t\n", "", ""},
		{"a b", "a b", "a b"},
	}
	for _, c := range cases {
		if got := Strip(c.in); got != c.strip {
			t.Errorf("Strip(%+q) = %+q, want %+q", c.in, got, c.strip)
		}
		if got := RStrip(c.in); got != c.rstrip {
			t.Errorf("RStrip(%+q) = %+q, want %+q", c.in, got, c.rstrip)
		}
	}
}

func TestReadTextFoldsCROnlyAndKeepsTheBOM(t *testing.T) {
	// Path.read_text(encoding="utf-8") of the same bytes on Python 3.14.7.
	cases := []struct{ name, bytes, want string }{
		{"bom-crlf", "\xef\xbb\xbfa\r\nb\rc\n\xc2\x85d\r", "\U0000feffa\nb\nc\n\U00000085d\n"},
		{"lone-cr", "x\r", "x\n"},
		{"cr-crlf", "\r\r\n", "\n\n"},
	}
	for _, c := range cases {
		path := filepath.Join(t.TempDir(), c.name)
		if err := os.WriteFile(path, []byte(c.bytes), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := ReadText(path)
		if err != nil || got != c.want {
			t.Errorf("ReadText(%s) = %+q, %v; want %+q", c.name, got, err, c.want)
		}
	}
}

func TestReadTextRefusesBytesPythonCannotDecode(t *testing.T) {
	// Python: UnicodeDecodeError for both, "invalid start byte" and
	// "invalid continuation byte" -- an encoded surrogate is not UTF-8.
	for name, bytes := range map[string]string{"invalid": "a\xffb", "surrogate": "a\xed\xa0\x80b"} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(bytes), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := ReadText(path)
		if want := path + ": not valid UTF-8"; err == nil || err.Error() != want {
			t.Errorf("ReadText(%s) err = %v, want %q", name, err, want)
		}
	}
}

func TestReadTextPassesOtherErrorsOn(t *testing.T) {
	_, err := ReadText(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want fs.ErrNotExist", err)
	}
}
```

- [ ] **Step 17: Scheitern sehen**

```bash
go test ./internal/brain/pytext/
```
Expected (gemessen):
```
# github.com/xidus90/loomux/internal/brain/pytext [github.com/xidus90/loomux/internal/brain/pytext.test]
internal\brain\pytext\text_test.go:21:6: undefined: IsSpace
internal\brain\pytext\text_test.go:22:56: undefined: IsSpace
internal\brain\pytext\text_test.go:38:13: undefined: Strip
internal\brain\pytext\text_test.go:41:13: undefined: RStrip
internal\brain\pytext\text_test.go:59:15: undefined: ReadText
internal\brain\pytext\text_test.go:74:13: undefined: ReadText
internal\brain\pytext\text_test.go:82:12: undefined: ReadText
FAIL	github.com/xidus90/loomux/internal/brain/pytext [build failed]
FAIL
```

- [ ] **Step 18: Implementieren** — `internal/brain/pytext/text.go`

```go
package pytext

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// IsSpace is `str.isspace()` of one character: the 29 code points Python
// 3.14 calls space, listed over the whole range. It is not
// `unicode.IsSpace`, which leaves out \x1c-\x1f.
func IsSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x1f, ' ',
		0x85, 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

// Strip is `s.strip()` without arguments.
func Strip(s string) string {
	return strings.TrimFunc(s, IsSpace)
}

// RStrip is `s.rstrip()` without arguments.
func RStrip(s string) string {
	return strings.TrimRightFunc(s, IsSpace)
}

// ReadText is `Path(path).read_text(encoding="utf-8")`: strict UTF-8, and
// universal newlines, which turn `\r\n` and a lone `\r` into `\n` and leave
// every other boundary alone. A byte order mark stays, because "utf-8" is
// not "utf-8-sig".
//
// Python ends with a UnicodeDecodeError traceback on bytes that are not
// UTF-8; this answers an error naming the file. Every other failure is
// os.ReadFile's, unwrapped.
func ReadText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s: not valid UTF-8", path)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n"), nil
}
```

- [ ] **Step 19: Bestehen sehen**

```bash
go test ./internal/brain/pytext/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/pytext`.

- [ ] **Step 20: Failing tests schreiben — `isoformat`, `fromisoformat`** — `internal/brain/pytext/isotime_test.go`

```go
package pytext

import (
	"testing"
	"time"
)

func TestIsoFormatLikePython(t *testing.T) {
	// d.isoformat() of the same values on Python 3.14.7.
	zone := func(seconds int) *time.Location { return time.FixedZone("", seconds) }
	cases := []struct {
		in   time.Time
		want string
	}{
		{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), "2000-01-01T00:00:00+00:00"},
		{time.Date(2000, 1, 1, 12, 30, 45, 5000, time.UTC), "2000-01-01T12:30:45.000005+00:00"},
		{time.Date(2000, 1, 1, 12, 30, 45, 123000000, time.UTC), "2000-01-01T12:30:45.123000+00:00"},
		{time.Date(2026, 9, 15, 8, 0, 0, 0, zone(7200)), "2026-09-15T08:00:00+02:00"},
		{time.Date(2026, 9, 15, 8, 0, 0, 0, zone(-19800)), "2026-09-15T08:00:00-05:30"},
		{time.Date(2026, 9, 15, 8, 0, 0, 0, zone(30)), "2026-09-15T08:00:00+00:00:30"},
		{time.Date(2026, 9, 15, 8, 0, 0, 0, zone(-3630)), "2026-09-15T08:00:00-01:00:30"},
		{time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), "0001-01-01T00:00:00+00:00"},
		{time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC), "9999-12-31T23:59:59.999999+00:00"},
		// Go only: Python holds no nanoseconds, a datetime would have cut them.
		{time.Date(2000, 1, 1, 0, 0, 0, 123456789, time.UTC), "2000-01-01T00:00:00.123456+00:00"},
		{time.Date(2000, 1, 1, 0, 0, 0, 999, time.UTC), "2000-01-01T00:00:00+00:00"},
	}
	for _, c := range cases {
		if got := IsoFormat(c.in); got != c.want {
			t.Errorf("IsoFormat(%v) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestParseAwareIsoFormatAcceptsWhatPythonReadsAsAware(t *testing.T) {
	// Left: the input; right: datetime.fromisoformat(input).isoformat() on
	// Python 3.14.7, which is also IsoFormat of the parsed value here.
	cases := []struct{ in, want string }{
		{"2000-01-01T00:00:00+00:00", "2000-01-01T00:00:00+00:00"},
		{"2000-01-01T00:00:00Z", "2000-01-01T00:00:00+00:00"},
		{"2000-01-01T00:00:00.1+00:00", "2000-01-01T00:00:00.100000+00:00"},
		{"2000-01-01T00:00:00.12+00:00", "2000-01-01T00:00:00.120000+00:00"},
		{"2000-01-01T00:00:00.123+00:00", "2000-01-01T00:00:00.123000+00:00"},
		{"2000-01-01T00:00:00.1234+00:00", "2000-01-01T00:00:00.123400+00:00"},
		{"2000-01-01T00:00:00.12345+00:00", "2000-01-01T00:00:00.123450+00:00"},
		{"2000-01-01T00:00:00.123456+00:00", "2000-01-01T00:00:00.123456+00:00"},
		{"2000-01-01T00:00:00.1234567+00:00", "2000-01-01T00:00:00.123456+00:00"},
		{"2000-01-01T00:00:00.123456789+00:00", "2000-01-01T00:00:00.123456+00:00"},
		{"2000-01-01T00:00:00,5+00:00", "2000-01-01T00:00:00.500000+00:00"},
		{"2000-01-01T00:00:00.5Z", "2000-01-01T00:00:00.500000+00:00"},
		{"2026-09-15T06:12:03.123456+02:00", "2026-09-15T06:12:03.123456+02:00"},
		{"2026-09-15T06:12:03-05:30", "2026-09-15T06:12:03-05:30"},
		{"2000-01-01 00:00:00+00:00", "2000-01-01T00:00:00+00:00"},
		{"2000-01-01T00:00:00+02:00:30", "2000-01-01T00:00:00+02:00:30"},
		{"2000-01-01T00:00:00-01:00:30", "2000-01-01T00:00:00-01:00:30"},
		{"2000-01-01T00:00:00-00:00", "2000-01-01T00:00:00+00:00"},
		{"2000-01-01T00:00:00+23:59", "2000-01-01T00:00:00+23:59"},
		{"2000-01-01T23:59:59-23:59", "2000-01-01T23:59:59-23:59"},
		{"2000-02-29T00:00:00+00:00", "2000-02-29T00:00:00+00:00"},
		{"2999-01-01T00:00:00+00:00", "2999-01-01T00:00:00+00:00"},
	}
	for _, c := range cases {
		got, ok := ParseAwareIsoFormat(c.in)
		if !ok || IsoFormat(got) != c.want {
			t.Errorf("ParseAwareIsoFormat(%+q) = %s, %v; want %s", c.in, IsoFormat(got), ok, c.want)
		}
	}
}

func TestParseAwareIsoFormatAnswersTheInstant(t *testing.T) {
	got, ok := ParseAwareIsoFormat("2026-09-15T06:12:03.5+02:00")
	if want := time.Date(2026, 9, 15, 4, 12, 3, 500000000, time.UTC); !ok || !got.Equal(want) {
		t.Errorf("ParseAwareIsoFormat = %v, %v; want %v", got, ok, want)
	}
}

func TestParseAwareIsoFormatRefusesWhatPythonRefusesOrReadsAsNaive(t *testing.T) {
	// Python 3.14.7: "naive" for the first three, ValueError for the rest.
	for _, in := range []string{
		"2000-01-01T00:00:00",
		"2000-01-01",
		"2000-01-01T00:00:00.5",
		"garbage",
		"",
		"2000-01-01T00:00:00z",
		"2000-01-01T00:00:00.+00:00",
		"2000-01-01T00:00:00+24:00",
		" 2000-01-01T00:00:00+00:00",
		"2000-01-01T00:00:00+00:00\n",
		"0000-01-01T00:00:00+00:00",
		"2000-1-01T00:00:00+00:00",
		"\U0000ff12\U0000ff10\U0000ff10\U0000ff10-01-01T00:00:00+00:00",
		"2000-01-01T00:00:00+00:00Z",
		"2000-01-01T00:00:00UTC",
		"2000-01-01T00:00:00+5:00",
		"2000-01-01T1:00:00+00:00",
		"2000-02-30T00:00:00+00:00",
		"1900-02-29T00:00:00+00:00",
		"2000-13-01T00:00:00+00:00",
		"2000-00-01T00:00:00+00:00",
		"2000-01-00T00:00:00+00:00",
		"2000-01-01T00:60:00+00:00",
		"2000-01-01T00:00:60+00:00",
		"2000-01-01T00:00:00+00:00.5",
		"2000-01-01T00:00:00+02:00:30Z",
	} {
		if got, ok := ParseAwareIsoFormat(in); ok {
			t.Errorf("ParseAwareIsoFormat(%+q) = %v, want false", in, got)
		}
	}
}

func TestParseAwareIsoFormatRefusesFormsOnlyPythonReads(t *testing.T) {
	// Python 3.14.7 reads each of these as aware; each is a parity line.
	for _, in := range []string{
		"2000-01-01t00:00:00+00:00",
		"2000-01-01X00:00:00+00:00",
		"2000-01-01\U000000e900:00:00+00:00",
		"2000-01-01T00:00+00:00",
		"2000-01-01T00+00:00",
		"20000101T000000+0000",
		"2000-01-01T00:00:00+0200",
		"2000-01-01T00:00:00+02",
		"2000-01-01T00:00:00+02:00:30.5",
		"2000-W01-1T00:00:00+00:00",
		"2000-01-01T24:00:00+00:00",
		"2000-01-01T00:00:00+00:60",
		"2000-01-01T00:00:00+00:00:60",
		"2000-01-01T00:00:00 +00:00",
	} {
		if got, ok := ParseAwareIsoFormat(in); ok {
			t.Errorf("ParseAwareIsoFormat(%+q) = %v, want false", in, got)
		}
	}
}
```

- [ ] **Step 21: Scheitern sehen**

```bash
go test ./internal/brain/pytext/
```
Expected (gemessen, vollständig — die Testdatei hat genau diese sieben Aufrufe, weniger als die zehn, nach denen der Compiler abbricht):
```
# github.com/xidus90/loomux/internal/brain/pytext [github.com/xidus90/loomux/internal/brain/pytext.test]
internal\brain\pytext\isotime_test.go:29:13: undefined: IsoFormat
internal\brain\pytext\isotime_test.go:63:14: undefined: ParseAwareIsoFormat
internal\brain\pytext\isotime_test.go:64:13: undefined: IsoFormat
internal\brain\pytext\isotime_test.go:65:65: undefined: IsoFormat
internal\brain\pytext\isotime_test.go:71:13: undefined: ParseAwareIsoFormat
internal\brain\pytext\isotime_test.go:107:17: undefined: ParseAwareIsoFormat
internal\brain\pytext\isotime_test.go:131:17: undefined: ParseAwareIsoFormat
FAIL	github.com/xidus90/loomux/internal/brain/pytext [build failed]
FAIL
```

- [ ] **Step 22: Implementieren** — `internal/brain/pytext/isotime.go`

```go
package pytext

import (
	"fmt"
	"strings"
	"time"
)

// IsoFormat is `datetime.isoformat()` of an aware value:
// `YYYY-MM-DDTHH:MM:SS`, `.ffffff` only when the microseconds are not zero,
// and the offset as `±HH:MM`, with `:SS` only when it has seconds. Go keeps
// nanoseconds and Python does not; the digits below the microsecond are
// cut, as a datetime built from this time would have cut them.
func IsoFormat(t time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%04d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second())
	if micro := t.Nanosecond() / 1000; micro != 0 {
		fmt.Fprintf(&b, ".%06d", micro)
	}
	_, offset := t.Zone()
	sign := '+'
	if offset < 0 {
		sign, offset = '-', -offset
	}
	fmt.Fprintf(&b, "%c%02d:%02d", sign, offset/3600, offset/60%60)
	if seconds := offset % 60; seconds != 0 {
		fmt.Fprintf(&b, ":%02d", seconds)
	}
	return b.String()
}

// ParseAwareIsoFormat is `datetime.fromisoformat(s)` for the one form this
// stage reads, the reconcile stamp, and answers false where Python raises
// or answers a naive value. The form accepted is
//
//	YYYY-MM-DD (T or space) HH:MM:SS [(. or ,) digits] (Z | ±HH:MM[:SS])
//
// with a valid calendar date, hour <= 23, minute and second <= 59, and
// digits past the sixth cut as Python cuts them. `_write_last_run` writes
// `now.isoformat() + "\n"` and `read_last_run` strips before parsing; the
// caller hands in `Strip(text)`, and no other form fits the writer.
//
// Python accepts more, each measured and each a line of the parity list:
// any other separator character, shortened and basic times and offsets,
// week dates, 24:00, a fraction in the offset, an offset minute or second
// of 60, and a blank before the offset. A stamp in such a form reads as
// never reconciled here.
func ParseAwareIsoFormat(s string) (time.Time, bool) {
	if len(s) < 20 || !shape(s[:19], "dddd-dd-dd?dd:dd:dd") || (s[10] != 'T' && s[10] != ' ') {
		return time.Time{}, false
	}
	year, month, day := number(s[0:4]), number(s[5:7]), number(s[8:10])
	hour, minute, second := number(s[11:13]), number(s[14:16]), number(s[17:19])
	rest := s[19:]
	micro := 0
	if rest[0] == '.' || rest[0] == ',' {
		end := 1
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if end == 1 {
			return time.Time{}, false
		}
		fraction := (rest[1:end] + "00000")[:6]
		micro, rest = number(fraction), rest[end:]
	}
	offset := 0
	switch {
	case rest == "Z":
	case (shape(rest, "?dd:dd") || shape(rest, "?dd:dd:dd")) && (rest[0] == '+' || rest[0] == '-'):
		hours, minutes, seconds := number(rest[1:3]), number(rest[4:6]), 0
		if len(rest) == 9 {
			seconds = number(rest[7:9])
		}
		if hours > 23 || minutes > 59 || seconds > 59 {
			return time.Time{}, false
		}
		offset = hours*3600 + minutes*60 + seconds
		if rest[0] == '-' {
			offset = -offset
		}
	default:
		return time.Time{}, false
	}
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if year < 1 || month < 1 || month > 12 || day < 1 || day > lastDay || hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), day, hour, minute, second, micro*1000, time.FixedZone("", offset)), true
}

// shape says whether s has the length of pattern and, position by position,
// an ASCII digit under `d`, anything under `?` and the same byte elsewhere.
func shape(s, pattern string) bool {
	if len(s) != len(pattern) {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch pattern[i] {
		case 'd':
			if s[i] < '0' || s[i] > '9' {
				return false
			}
		case '?':
		default:
			if s[i] != pattern[i] {
				return false
			}
		}
	}
	return true
}

// number is the value of a run of ASCII digits the caller has checked.
func number(digits string) int {
	n := 0
	for i := 0; i < len(digits); i++ {
		n = n*10 + int(digits[i]-'0')
	}
	return n
}
```

- [ ] **Step 23: Bestehen sehen**

```bash
go test ./internal/brain/pytext/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/pytext`. Die Stempelformen, die Task 8 (`TestReadLastRunReadsWhatPythonReads`: Writer-Form, Mikrosekunden mit `+02:00`, `Z`, `.500`, naiv, Müll, BOM) und Task 9 (`2999-01-01T00:00:00.255+02:00`) erwarten, liegen alle in den Tabellen oben.

- [ ] **Step 24: Failing tests schreiben — `str(Path)`** — `internal/brain/pytext/path_test.go`

```go
package pytext

import "testing"

func TestPathStringOnWindowsLikePureWindowsPath(t *testing.T) {
	// print(ascii(str(PureWindowsPath(s)))) on Python 3.14.7.
	cases := []struct{ in, want string }{
		{"C:/a/b", `C:\a\b`},
		{"C:/a//b/./c/", `C:\a\b\c`},
		{"C:/a/../b", `C:\a\..\b`},
		{"//server/share/x", `\\server\share\x`},
		{"//server/share", `\\server\share\`},
		{"//server/share/", `\\server\share\`},
		{"//server", `\\server`},
		{"//server/", `\\server\`},
		{"", `.`},
		{".", `.`},
		{"./", `.`},
		{"a/b", `a\b`},
		{"/a", `\a`},
		{`\a`, `\a`},
		{"C:", `C:`},
		{"C:a/b", `C:a\b`},
		{"C:/", `C:\`},
		{"c:/x", `c:\x`},
		{"1:/x", `1:\x`},
		{"ab:/c", `ab:\c`},
		{"//?/C:/x", `\\?\C:\x`},
		{"//./dev/x", `\\.\dev\x`},
		{"a/./b/", `a\b`},
		{"..", `..`},
		{"C:/Users/micro/Documents/#GIT/loomux", `C:\Users\micro\Documents\#GIT\loomux`},
		{"./a:b", `.\a:b`},
		{"///a", `\\\a`},
		{`\\?\UNC\server\share\x`, `\\?\UNC\server\share\x`},
		{"\u00e4:/x", "\u00e4:\\x"},
		{`a\\b`, `a\b`},
		{"C:.", `C:`},
		{"C:./a", `C:a`},
		{"/", `\`},
		{"//", `\\`},
		{"///", `\\\`},
		{`C:\\x`, `C:\x`},
	}
	for _, c := range cases {
		if got := PathString("windows", c.in); got != c.want {
			t.Errorf("PathString(windows, %q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPathStringElsewhereLikePurePosixPath(t *testing.T) {
	// print(ascii(str(PurePosixPath(s)))) on Python 3.14.7.
	cases := []struct{ in, want string }{
		{"C:/a/b", "C:/a/b"},
		{"C:/a//b/./c/", "C:/a/b/c"},
		{"a/../b", "a/../b"},
		{"//server/share/x", "//server/share/x"},
		{"///a", "/a"},
		{"/", "/"},
		{"", "."},
		{".", "."},
		{"a/", "a"},
		{"./a", "a"},
		{`a\b`, `a\b`},
		{"//", "//"},
		{"./a:b", "a:b"},
		{"a/./b/.", "a/b"},
	}
	for _, c := range cases {
		if got := PathString("linux", c.in); got != c.want {
			t.Errorf("PathString(linux, %q) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 25: Scheitern sehen**

```bash
go test ./internal/brain/pytext/
```
Expected (gemessen):
```
# github.com/xidus90/loomux/internal/brain/pytext [github.com/xidus90/loomux/internal/brain/pytext.test]
internal\brain\pytext\path_test.go:46:13: undefined: PathString
internal\brain\pytext\path_test.go:71:13: undefined: PathString
FAIL	github.com/xidus90/loomux/internal/brain/pytext [build failed]
FAIL
```

- [ ] **Step 26: Implementieren** — `internal/brain/pytext/path.go`

```go
package pytext

import (
	"strings"
	"unicode/utf8"
)

// PathString is `str(Path(p))` on the platform goos names: PureWindowsPath
// for "windows", PurePosixPath for every other. Taking goos as an argument
// keeps both flavours testable on either system.
//
// Both follow `PurePath._parse_path` and `_format_parsed_parts` of Python
// 3.14 (pathlib/_local.py): split off drive and root, drop empty and `.`
// parts, keep `..`, join with the separator, and answer `.` for nothing.
func PathString(goos, p string) string {
	if goos == "windows" {
		return windowsPath(p)
	}
	return posixPath(p)
}

// posixPath keeps exactly two leading slashes, as `posixpath.splitroot`
// does, and collapses one or three and more to one.
func posixPath(p string) string {
	root, rel := "", p
	switch {
	case !strings.HasPrefix(p, "/"):
	case strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "///"):
		root, rel = "//", p[2:]
	default:
		root, rel = "/", p[1:]
	}
	return orDot(root + strings.Join(pathParts(rel, "/"), "/"))
}

// windowsPath turns slashes into backslashes first, then reads drive and
// root the way `_parse_path` does: a UNC drive of exactly four parts whose
// third is neither empty, `?` nor `.` gains a root (`//server/share` is
// `\\server\share\`), and so does one of six parts. A relative first part
// that reads as a drive gets `.` in front.
func windowsPath(p string) string {
	p = strings.ReplaceAll(p, "/", `\`)
	drive, root, rel := windowsSplitRoot(p)
	if root == "" && strings.HasPrefix(drive, `\`) && !strings.HasSuffix(drive, `\`) {
		driveParts := strings.Split(drive, `\`)
		// `drv_parts[2] not in '?.'` is a substring test, so the empty part
		// counts as in it: `///a` keeps no root.
		if len(driveParts) == 4 && !strings.Contains("?.", driveParts[2]) || len(driveParts) == 6 {
			root = `\`
		}
	}
	tail := pathParts(rel, `\`)
	if drive == "" && root == "" && len(tail) > 0 {
		if first, _, _ := windowsSplitRoot(tail[0]); first != "" {
			tail = append([]string{"."}, tail...)
		}
	}
	return orDot(drive + root + strings.Join(tail, `\`))
}

// windowsSplitRoot is the pure-Python `ntpath.splitroot` of 3.14 on a path
// that holds backslashes only. A drive letter is any one character before
// `:`, counted in characters, not bytes.
func windowsSplitRoot(p string) (drive, root, rel string) {
	if strings.HasPrefix(p, `\`) {
		if !strings.HasPrefix(p, `\\`) {
			return "", `\`, p[1:]
		}
		start := 2
		if len(p) >= 8 && strings.EqualFold(p[:8], `\\?\UNC\`) {
			start = 8
		}
		index := strings.IndexByte(p[start:], '\\')
		if index < 0 {
			return p, "", ""
		}
		index += start
		index2 := strings.IndexByte(p[index+1:], '\\')
		if index2 < 0 {
			return p, "", ""
		}
		index2 += index + 1
		return p[:index2], `\`, p[index2+1:]
	}
	if _, size := utf8.DecodeRuneInString(p); len(p) > size && p[size] == ':' {
		if len(p) > size+1 && p[size+1] == '\\' {
			return p[:size+1], `\`, p[size+2:]
		}
		return p[:size+1], "", p[size+1:]
	}
	return "", "", p
}

// pathParts are the parts of rel between separators, without the empty ones
// and without `.`.
func pathParts(rel, sep string) []string {
	var parts []string
	for _, part := range strings.Split(rel, sep) {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return parts
}

// orDot is `... or '.'` of `PurePath.__str__`.
func orDot(s string) string {
	if s == "" {
		return "."
	}
	return s
}
```

- [ ] **Step 27: Bestehen sehen, formatieren, prüfen**

```bash
go test ./internal/brain/pytext/ -count=1
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/pytext` (gemessen 0,59 s allein; die Bereichsschleifen kosten `Repr` 0,14 s, `CaseFold` 0,12 s, `NFC` 0,05 s, `SplitLines` 0,05 s, `IsSpace` 0,01 s).
```bash
gofmt -l internal/brain/pytext
```
Expected: keine Ausgabe.
```bash
go vet ./internal/brain/pytext/
```
Expected: keine Ausgabe.

- [ ] **Step 28: Coverage**

```bash
go test ./internal/brain/pytext/ -count=1 -covermode=set -coverpkg=./internal/brain/pytext/ -coverprofile="$TEMP/pkg.out"
```
Expected: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/brain/pytext` beginnt und mit `coverage: 100.0% of statements in ./internal/brain/pytext/` endet.
```bash
go tool cover -func="$TEMP/pkg.out"
```
Erwartet: jede Funktion des Pakets 100.0%. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund. Die 21 Funktionszeilen, gemessen (danach folgt die Zeile `total:` mit `(statements)` und `100.0%`); die Zeilennummern in `isotime.go` ab `ParseAwareIsoFormat` sind um die eine Kommentarzeile aus Step 22 nachgerechnet:
```
github.com/xidus90/loomux/internal/brain/pytext/isotime.go:14:	IsoFormat		100.0%
github.com/xidus90/loomux/internal/brain/pytext/isotime.go:49:	ParseAwareIsoFormat	100.0%
github.com/xidus90/loomux/internal/brain/pytext/isotime.go:95:	shape			100.0%
github.com/xidus90/loomux/internal/brain/pytext/isotime.go:116:	number			100.0%
github.com/xidus90/loomux/internal/brain/pytext/lines.go:8:	SplitLines		100.0%
github.com/xidus90/loomux/internal/brain/pytext/lines.go:34:	isLineBoundary		100.0%
github.com/xidus90/loomux/internal/brain/pytext/path.go:15:	PathString		100.0%
github.com/xidus90/loomux/internal/brain/pytext/path.go:24:	posixPath		100.0%
github.com/xidus90/loomux/internal/brain/pytext/path.go:41:	windowsPath		100.0%
github.com/xidus90/loomux/internal/brain/pytext/path.go:64:	windowsSplitRoot	100.0%
github.com/xidus90/loomux/internal/brain/pytext/path.go:96:	pathParts		100.0%
github.com/xidus90/loomux/internal/brain/pytext/path.go:107:	orDot			100.0%
github.com/xidus90/loomux/internal/brain/pytext/repr.go:28:	Repr			100.0%
github.com/xidus90/loomux/internal/brain/pytext/repr.go:71:	writeHex		100.0%
github.com/xidus90/loomux/internal/brain/pytext/text.go:13:	IsSpace			100.0%
github.com/xidus90/loomux/internal/brain/pytext/text.go:23:	Strip			100.0%
github.com/xidus90/loomux/internal/brain/pytext/text.go:28:	RStrip			100.0%
github.com/xidus90/loomux/internal/brain/pytext/text.go:40:	ReadText		100.0%
github.com/xidus90/loomux/internal/brain/pytext/unicode.go:13:	NFC			100.0%
github.com/xidus90/loomux/internal/brain/pytext/unicode.go:24:	CaseFold		100.0%
github.com/xidus90/loomux/internal/brain/pytext/unicode.go:30:	cherokeeCapital		100.0%
```
Kein `//coverage:exempt`. Die Arme, die man leicht übersieht: `Repr` Nicht-UTF-8-Byte (`TestReprWritesAByteThatIsNotUTF8AsHex`), `ReadText` Lesefehler (`TestReadTextPassesOtherErrorsOn`), `ParseAwareIsoFormat` leerer Bruch (`.+00:00`) und Offset außerhalb (`+24:00`), `windowsSplitRoot` `\\?\UNC\` und UNC ohne zweiten Trenner (`//server`). Ein Test, den Umzugsverfahren Schritt 6 hier verlangt, nimmt seinen erwarteten Wert aus einer Messung an der Referenz wie in Step 29.

- [ ] **Step 29: Messbefehle wiederholen (für den Bericht)**

Nur lesend. Unter Windows halbiert `uv run` doppelte Backslashes im `-c`-Argument (gemessen: `"a\\b"` kam als `a\x08` an); die Befehle bilden Backslashes darum mit `chr(92)`. Die Ausgaben unten sind die vom 2026-09-15.

(a) Version, Unicode-Stand und die drei Prüfsummen (dauert rund eine Minute):
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c 'import hashlib, sys, unicodedata
R = [(0x88F,0x88F),(0xC5C,0xC5C),(0xCDC,0xCDC),(0x1ACF,0x1ADD),(0x1AE0,0x1AEB),(0x20C1,0x20C1),(0x2B96,0x2B96),(0xA7CE,0xA7CF),(0xA7D2,0xA7D2),(0xA7D4,0xA7D4),(0xA7F1,0xA7F1),(0xFBC3,0xFBD2),(0xFD90,0xFD91),(0xFDC8,0xFDCE),(0x10940,0x10959),(0x10EC5,0x10EC7),(0x10ED0,0x10ED8),(0x10EFA,0x10EFB),(0x11B60,0x11B67),(0x11DB0,0x11DDB),(0x11DE0,0x11DE9),(0x16EA0,0x16EB8),(0x16EBB,0x16ED3),(0x16FF2,0x16FF6),(0x187F8,0x187FF),(0x18D09,0x18D1E),(0x18D80,0x18DF2),(0x1CCFA,0x1CCFC),(0x1CEBA,0x1CED0),(0x1CEE0,0x1CEF0),(0x1E6C0,0x1E6DE),(0x1E6E0,0x1E6F5),(0x1E6FE,0x1E6FF),(0x1F6D8,0x1F6D8),(0x1F777,0x1F77A),(0x1F8D0,0x1F8D8),(0x1FA54,0x1FA57),(0x1FA8A,0x1FA8A),(0x1FA8E,0x1FA8E),(0x1FAC8,0x1FAC8),(0x1FACD,0x1FACD),(0x1FAEA,0x1FAEA),(0x1FAEF,0x1FAEF),(0x1FBFA,0x1FBFA),(0x2B73A,0x2B73F),(0x2CEA2,0x2CEAD),(0x323B0,0x33479)]
NEW = {c for a, b in R for c in range(a, b + 1)}
def digest(f, skip):
    h = hashlib.sha256()
    for c in range(0x110000):
        if 0xD800 <= c <= 0xDFFF or c in skip:
            continue
        h.update(f(chr(c)).encode("utf-8")); h.update(b"\0")
    return h.hexdigest()
print(sys.version.split()[0], unicodedata.unidata_version, len(NEW), {unicodedata.category(chr(c)) for c in NEW})
print("repr", digest(repr, NEW))
print("nfc", digest(lambda s: unicodedata.normalize("NFC", s), set()))
print("casefold", digest(str.casefold, NEW))'
```
```
3.14.7 16.0.0 4803 {'Cn'}
repr 61bb4be799689f2c003eb8932471a2f8d2b63c3f74c9c313a461037097a31cc9
nfc 6e0aaa17a82be5b6f410942f1f99ae77832fb630e98780c6eaf6779ab278235d
casefold 10e93868ae4b30b2d8d735f20f016cae6203cd34bad840c9a7037911cef978d8
```
(Die Bereiche `R` stammen aus dem Vergleich `not chr(c).isprintable()` in Python gegen `unicode.IsPrint` in Go 1.27: 4803 Unterschiede, alle Go-druckbar und Python-`Cn`.)

(b) `repr`:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c 'B = chr(92); [print(ascii(s), "=>", ascii(repr(s))) for s in ["x", "it\x27s", "say \"hi\"", "both \x27 and \"", "a\tb", "\x00", "\x7f", "\xe9", "\u200b", "\U0001f600", "a" + B + "b", "\x27" + B, "\"" + B + "\x27", "a\rb\nc", "", "\x80", "\xa0", "\xad", "\xff", "\u0100", "\u2028", "\ufeff", "\uffff", "\U000e0001", "\U0010ffff", " ", "\x1b", "\u0378", "\U0001fae9"]]'
```
```
'x' => "'x'"
"it's" => '"it\'s"'
'say "hi"' => '\'say "hi"\''
'both \' and "' => '\'both \\\' and "\''
'a\tb' => "'a\\tb'"
'\x00' => "'\\x00'"
'\x7f' => "'\\x7f'"
'\xe9' => "'\xe9'"
'\u200b' => "'\\u200b'"
'\U0001f600' => "'\U0001f600'"
'a\\b' => "'a\\\\b'"
"'\\" => '"\'\\\\"'
'"\\\'' => '\'"\\\\\\\'\''
'a\rb\nc' => "'a\\rb\\nc'"
'' => "''"
'\x80' => "'\\x80'"
'\xa0' => "'\\xa0'"
'\xad' => "'\\xad'"
'\xff' => "'\xff'"
'\u0100' => "'\u0100'"
'\u2028' => "'\\u2028'"
'\ufeff' => "'\\ufeff'"
'\uffff' => "'\\uffff'"
'\U000e0001' => "'\\U000e0001'"
'\U0010ffff' => "'\\U0010ffff'"
' ' => "' '"
'\x1b' => "'\\x1b'"
'\u0378' => "'\\u0378'"
'\U0001fae9' => "'\U0001fae9'"
```

(c) `isspace`, `splitlines`, `strip`/`rstrip`:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c 'print([hex(c) for c in range(0x110000) if chr(c).isspace()]); print([hex(c) for c in range(0x110000) if not 0xd800 <= c <= 0xdfff and len(("a" + chr(c) + "b").splitlines()) == 2]); [print(ascii(s), ascii(s.splitlines())) for s in ["a\n", "\n", "", "a\r\nb", "\r\r\n", "a\n\rb", "\n\n", "a", "a\x85b\u2028c\u2029d\x0be\x0cf\x1cg\x1dh\x1ei\rj\r\nk\nl"]]; [print(ascii(s), "strip", ascii(s.strip()), "rstrip", ascii(s.rstrip())) for s in ["  a  ", "\x1c a \x1f", "\xa0a\u3000", "\u200ba\u200b", "\t\n", "a b"]]'
```
```
['0x9', '0xa', '0xb', '0xc', '0xd', '0x1c', '0x1d', '0x1e', '0x1f', '0x20', '0x85', '0xa0', '0x1680', '0x2000', '0x2001', '0x2002', '0x2003', '0x2004', '0x2005', '0x2006', '0x2007', '0x2008', '0x2009', '0x200a', '0x2028', '0x2029', '0x202f', '0x205f', '0x3000']
['0xa', '0xb', '0xc', '0xd', '0x1c', '0x1d', '0x1e', '0x85', '0x2028', '0x2029']
'a\n' ['a']
'\n' ['']
'' []
'a\r\nb' ['a', 'b']
'\r\r\n' ['', '']
'a\n\rb' ['a', '', 'b']
'\n\n' ['', '']
'a' ['a']
'a\x85b\u2028c\u2029d\x0be\x0cf\x1cg\x1dh\x1ei\rj\r\nk\nl' ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l']
'  a  ' strip 'a' rstrip '  a'
'\x1c a \x1f' strip 'a' rstrip '\x1c a'
'\xa0a\u3000' strip 'a' rstrip '\xa0a'
'\u200ba\u200b' strip '\u200ba\u200b' rstrip '\u200ba\u200b'
'\t\n' strip '' rstrip ''
'a b' strip 'a b' rstrip 'a b'
```

(d) `read_text` — die Bytes schreibt die Shell nach `$TEMP`:
```bash
printf '\xef\xbb\xbfa\r\nb\rc\n\xc2\x85d\r' > "$TEMP/pytext-bom-crlf.bin"
```
Expected: keine Ausgabe, Exit 0.
```bash
printf 'x\r' > "$TEMP/pytext-lone-cr.bin"
```
Expected: keine Ausgabe, Exit 0.
```bash
printf '\r\r\n' > "$TEMP/pytext-cr-crlf.bin"
```
Expected: keine Ausgabe, Exit 0.
```bash
printf 'a\xffb' > "$TEMP/pytext-invalid.bin"
```
Expected: keine Ausgabe, Exit 0.
```bash
printf 'a\xed\xa0\x80b' > "$TEMP/pytext-surrogate.bin"
```
Expected: keine Ausgabe, Exit 0.
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c 'import os
from pathlib import Path
for n in ["bom-crlf", "lone-cr", "cr-crlf", "invalid", "surrogate"]:
    try:
        print(n, ascii(Path(os.environ["TEMP"], "pytext-" + n + ".bin").read_text(encoding="utf-8")))
    except ValueError as e:
        print(n, type(e).__name__, e)'
```
```
bom-crlf '\ufeffa\nb\nc\n\x85d\n'
lone-cr 'x\n'
cr-crlf '\n\n'
invalid UnicodeDecodeError 'utf-8' codec can't decode byte 0xff in position 1: invalid start byte
surrogate UnicodeDecodeError 'utf-8' codec can't decode byte 0xed in position 1: invalid continuation byte
```

(e) `isoformat`:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c 'from datetime import datetime, timedelta, timezone as tz
u = tz.utc
for d in [datetime(2000, 1, 1, tzinfo=u), datetime(2000, 1, 1, 12, 30, 45, 5, tzinfo=u), datetime(2000, 1, 1, 12, 30, 45, 123000, tzinfo=u), datetime(2026, 9, 15, 8, tzinfo=tz(timedelta(hours=2))), datetime(2026, 9, 15, 8, tzinfo=tz(timedelta(hours=-5, minutes=-30))), datetime(2026, 9, 15, 8, tzinfo=tz(timedelta(seconds=30))), datetime(2026, 9, 15, 8, tzinfo=tz(timedelta(seconds=-3630))), datetime(1, 1, 1, tzinfo=u), datetime(9999, 12, 31, 23, 59, 59, 999999, tzinfo=u)]:
    print(d.isoformat())'
```
```
2000-01-01T00:00:00+00:00
2000-01-01T12:30:45.000005+00:00
2000-01-01T12:30:45.123000+00:00
2026-09-15T08:00:00+02:00
2026-09-15T08:00:00-05:30
2026-09-15T08:00:00+00:00:30
2026-09-15T08:00:00-01:00:30
0001-01-01T00:00:00+00:00
9999-12-31T23:59:59.999999+00:00
```

(f) `fromisoformat` — die drei Gruppen in der Reihenfolge der drei Tabellentests:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c 'from datetime import datetime
for s in ["2000-01-01T00:00:00+00:00", "2000-01-01T00:00:00Z", "2000-01-01T00:00:00.1+00:00", "2000-01-01T00:00:00.12+00:00", "2000-01-01T00:00:00.123+00:00", "2000-01-01T00:00:00.1234+00:00", "2000-01-01T00:00:00.12345+00:00", "2000-01-01T00:00:00.123456+00:00", "2000-01-01T00:00:00.1234567+00:00", "2000-01-01T00:00:00.123456789+00:00", "2000-01-01T00:00:00,5+00:00", "2000-01-01T00:00:00.5Z", "2026-09-15T06:12:03.123456+02:00", "2026-09-15T06:12:03-05:30", "2000-01-01 00:00:00+00:00", "2000-01-01T00:00:00+02:00:30", "2000-01-01T00:00:00-01:00:30", "2000-01-01T00:00:00-00:00", "2000-01-01T00:00:00+23:59", "2000-01-01T23:59:59-23:59", "2000-02-29T00:00:00+00:00", "2999-01-01T00:00:00+00:00",
          "2000-01-01T00:00:00", "2000-01-01", "2000-01-01T00:00:00.5", "garbage", "", "2000-01-01T00:00:00z", "2000-01-01T00:00:00.+00:00", "2000-01-01T00:00:00+24:00", " 2000-01-01T00:00:00+00:00", "2000-01-01T00:00:00+00:00\n", "0000-01-01T00:00:00+00:00", "2000-1-01T00:00:00+00:00", "\uff12\uff10\uff10\uff10-01-01T00:00:00+00:00", "2000-01-01T00:00:00+00:00Z", "2000-01-01T00:00:00UTC", "2000-01-01T00:00:00+5:00", "2000-01-01T1:00:00+00:00", "2000-02-30T00:00:00+00:00", "1900-02-29T00:00:00+00:00", "2000-13-01T00:00:00+00:00", "2000-00-01T00:00:00+00:00", "2000-01-00T00:00:00+00:00", "2000-01-01T00:60:00+00:00", "2000-01-01T00:00:60+00:00", "2000-01-01T00:00:00+00:00.5", "2000-01-01T00:00:00+02:00:30Z",
          "2000-01-01t00:00:00+00:00", "2000-01-01X00:00:00+00:00", "2000-01-01\xe900:00:00+00:00", "2000-01-01T00:00+00:00", "2000-01-01T00+00:00", "20000101T000000+0000", "2000-01-01T00:00:00+0200", "2000-01-01T00:00:00+02", "2000-01-01T00:00:00+02:00:30.5", "2000-W01-1T00:00:00+00:00", "2000-01-01T24:00:00+00:00", "2000-01-01T00:00:00+00:60", "2000-01-01T00:00:00+00:00:60", "2000-01-01T00:00:00 +00:00"]:
    try:
        d = datetime.fromisoformat(s)
        print(ascii(s), "aware" if d.tzinfo else "naive", d.isoformat())
    except ValueError:
        print(ascii(s), "ValueError")'
```
```
'2000-01-01T00:00:00+00:00' aware 2000-01-01T00:00:00+00:00
'2000-01-01T00:00:00Z' aware 2000-01-01T00:00:00+00:00
'2000-01-01T00:00:00.1+00:00' aware 2000-01-01T00:00:00.100000+00:00
'2000-01-01T00:00:00.12+00:00' aware 2000-01-01T00:00:00.120000+00:00
'2000-01-01T00:00:00.123+00:00' aware 2000-01-01T00:00:00.123000+00:00
'2000-01-01T00:00:00.1234+00:00' aware 2000-01-01T00:00:00.123400+00:00
'2000-01-01T00:00:00.12345+00:00' aware 2000-01-01T00:00:00.123450+00:00
'2000-01-01T00:00:00.123456+00:00' aware 2000-01-01T00:00:00.123456+00:00
'2000-01-01T00:00:00.1234567+00:00' aware 2000-01-01T00:00:00.123456+00:00
'2000-01-01T00:00:00.123456789+00:00' aware 2000-01-01T00:00:00.123456+00:00
'2000-01-01T00:00:00,5+00:00' aware 2000-01-01T00:00:00.500000+00:00
'2000-01-01T00:00:00.5Z' aware 2000-01-01T00:00:00.500000+00:00
'2026-09-15T06:12:03.123456+02:00' aware 2026-09-15T06:12:03.123456+02:00
'2026-09-15T06:12:03-05:30' aware 2026-09-15T06:12:03-05:30
'2000-01-01 00:00:00+00:00' aware 2000-01-01T00:00:00+00:00
'2000-01-01T00:00:00+02:00:30' aware 2000-01-01T00:00:00+02:00:30
'2000-01-01T00:00:00-01:00:30' aware 2000-01-01T00:00:00-01:00:30
'2000-01-01T00:00:00-00:00' aware 2000-01-01T00:00:00+00:00
'2000-01-01T00:00:00+23:59' aware 2000-01-01T00:00:00+23:59
'2000-01-01T23:59:59-23:59' aware 2000-01-01T23:59:59-23:59
'2000-02-29T00:00:00+00:00' aware 2000-02-29T00:00:00+00:00
'2999-01-01T00:00:00+00:00' aware 2999-01-01T00:00:00+00:00
'2000-01-01T00:00:00' naive 2000-01-01T00:00:00
'2000-01-01' naive 2000-01-01T00:00:00
'2000-01-01T00:00:00.5' naive 2000-01-01T00:00:00.500000
'garbage' ValueError
'' ValueError
'2000-01-01T00:00:00z' ValueError
'2000-01-01T00:00:00.+00:00' ValueError
'2000-01-01T00:00:00+24:00' ValueError
' 2000-01-01T00:00:00+00:00' ValueError
'2000-01-01T00:00:00+00:00\n' ValueError
'0000-01-01T00:00:00+00:00' ValueError
'2000-1-01T00:00:00+00:00' ValueError
'\uff12\uff10\uff10\uff10-01-01T00:00:00+00:00' ValueError
'2000-01-01T00:00:00+00:00Z' ValueError
'2000-01-01T00:00:00UTC' ValueError
'2000-01-01T00:00:00+5:00' ValueError
'2000-01-01T1:00:00+00:00' ValueError
'2000-02-30T00:00:00+00:00' ValueError
'1900-02-29T00:00:00+00:00' ValueError
'2000-13-01T00:00:00+00:00' ValueError
'2000-00-01T00:00:00+00:00' ValueError
'2000-01-00T00:00:00+00:00' ValueError
'2000-01-01T00:60:00+00:00' ValueError
'2000-01-01T00:00:60+00:00' ValueError
'2000-01-01T00:00:00+00:00.5' ValueError
'2000-01-01T00:00:00+02:00:30Z' ValueError
'2000-01-01t00:00:00+00:00' aware 2000-01-01T00:00:00+00:00
'2000-01-01X00:00:00+00:00' aware 2000-01-01T00:00:00+00:00
'2000-01-01\xe900:00:00+00:00' aware 2000-01-01T00:00:00+00:00
'2000-01-01T00:00+00:00' aware 2000-01-01T00:00:00+00:00
'2000-01-01T00+00:00' aware 2000-01-01T00:00:00+00:00
'20000101T000000+0000' aware 2000-01-01T00:00:00+00:00
'2000-01-01T00:00:00+0200' aware 2000-01-01T00:00:00+02:00
'2000-01-01T00:00:00+02' aware 2000-01-01T00:00:00+02:00
'2000-01-01T00:00:00+02:00:30.5' aware 2000-01-01T00:00:00+02:00:30.500000
'2000-W01-1T00:00:00+00:00' aware 2000-01-03T00:00:00+00:00
'2000-01-01T24:00:00+00:00' aware 2000-01-02T00:00:00+00:00
'2000-01-01T00:00:00+00:60' aware 2000-01-01T00:00:00+01:00
'2000-01-01T00:00:00+00:00:60' aware 2000-01-01T00:00:00+00:01
'2000-01-01T00:00:00 +00:00' aware 2000-01-01T00:00:00+00:00
```

(g) `str(PureWindowsPath)`, `str(PurePosixPath)`:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c 'from pathlib import PureWindowsPath as W, PurePosixPath as P
B = chr(92)
for s in ["C:/a/b", "C:/a//b/./c/", "C:/a/../b", "//server/share/x", "//server/share", "//server/share/", "//server", "//server/", "", ".", "./", "a/b", "/a", B + "a", "C:", "C:a/b", "C:/", "c:/x", "1:/x", "ab:/c", "//?/C:/x", "//./dev/x", "a/./b/", "..", "C:/Users/micro/Documents/#GIT/loomux", "./a:b", "///a", B + B + "?" + B + "UNC" + B + "server" + B + "share" + B + "x", "\xe4:/x", "a" + B + B + "b", "C:.", "C:./a", "/", "//", "///", "C:" + B + B + "x"]:
    print("windows", ascii(s), ascii(str(W(s))))
for s in ["C:/a/b", "C:/a//b/./c/", "a/../b", "//server/share/x", "///a", "/", "", ".", "a/", "./a", "a" + B + "b", "//", "./a:b", "a/./b/."]:
    print("posix", ascii(s), ascii(str(P(s))))'
```
```
windows 'C:/a/b' 'C:\\a\\b'
windows 'C:/a//b/./c/' 'C:\\a\\b\\c'
windows 'C:/a/../b' 'C:\\a\\..\\b'
windows '//server/share/x' '\\\\server\\share\\x'
windows '//server/share' '\\\\server\\share\\'
windows '//server/share/' '\\\\server\\share\\'
windows '//server' '\\\\server'
windows '//server/' '\\\\server\\'
windows '' '.'
windows '.' '.'
windows './' '.'
windows 'a/b' 'a\\b'
windows '/a' '\\a'
windows '\\a' '\\a'
windows 'C:' 'C:'
windows 'C:a/b' 'C:a\\b'
windows 'C:/' 'C:\\'
windows 'c:/x' 'c:\\x'
windows '1:/x' '1:\\x'
windows 'ab:/c' 'ab:\\c'
windows '//?/C:/x' '\\\\?\\C:\\x'
windows '//./dev/x' '\\\\.\\dev\\x'
windows 'a/./b/' 'a\\b'
windows '..' '..'
windows 'C:/Users/micro/Documents/#GIT/loomux' 'C:\\Users\\micro\\Documents\\#GIT\\loomux'
windows './a:b' '.\\a:b'
windows '///a' '\\\\\\a'
windows '\\\\?\\UNC\\server\\share\\x' '\\\\?\\UNC\\server\\share\\x'
windows '\xe4:/x' '\xe4:\\x'
windows 'a\\\\b' 'a\\b'
windows 'C:.' 'C:'
windows 'C:./a' 'C:a'
windows '/' '\\'
windows '//' '\\\\'
windows '///' '\\\\\\'
windows 'C:\\\\x' 'C:\\x'
posix 'C:/a/b' 'C:/a/b'
posix 'C:/a//b/./c/' 'C:/a/b/c'
posix 'a/../b' 'a/../b'
posix '//server/share/x' '//server/share/x'
posix '///a' '/a'
posix '/' '/'
posix '' '.'
posix '.' '.'
posix 'a/' 'a'
posix './a' 'a'
posix 'a\\b' 'a\\b'
posix '//' '//'
posix './a:b' 'a:b'
posix 'a/./b/.' 'a/b'
```

(h) NFC und casefold:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c 'import unicodedata as u
for s in ["e\u0301", "\u212b", "a\u0301\u0328", "\u1e9b\u0323", "\uac00\u11a8", "\u0958"]: print("nfc", ascii(s), ascii(u.normalize("NFC", s)))
for s in ["\xdf", "\u1e9e", "\ufb01", "\u03a3", "\u03c2", "\u0130", "\u0149", "\u01f1", "ABC", "\xdf.md", "\u03a3\u0391\u03a3", "\u0130x", "\ufb01le.MD", "Stra\xdfe", "\u01c5", "\u1f88", "A\u030a", "\u13a0\uab70\u13f0\u13f8"]: print("casefold", ascii(s), ascii(s.casefold()))'
```
```
nfc 'e\u0301' '\xe9'
nfc '\u212b' '\xc5'
nfc 'a\u0301\u0328' '\u0105\u0301'
nfc '\u1e9b\u0323' '\u1e9b\u0323'
nfc '\uac00\u11a8' '\uac01'
nfc '\u0958' '\u0915\u093c'
casefold '\xdf' 'ss'
casefold '\u1e9e' 'ss'
casefold '\ufb01' 'fi'
casefold '\u03a3' '\u03c3'
casefold '\u03c2' '\u03c3'
casefold '\u0130' 'i\u0307'
casefold '\u0149' '\u02bcn'
casefold '\u01f1' '\u01f3'
casefold 'ABC' 'abc'
casefold '\xdf.md' 'ss.md'
casefold '\u03a3\u0391\u03a3' '\u03c3\u03b1\u03c3'
casefold '\u0130x' 'i\u0307x'
casefold '\ufb01le.MD' 'file.md'
casefold 'Stra\xdfe' 'strasse'
casefold '\u01c5' '\u01c6'
casefold '\u1f88' '\u1f00\u03b9'
casefold 'A\u030a' 'a\u030a'
casefold '\u13a0\uab70\u13f0\u13f8' '\u13a0\u13a0\u13f0\u13f0'
```

Weicht eine Ausgabe ab (andere Python-Version, anderer Unicode-Stand), wird nichts kopiert: der Befund geht an den Controller, und die betroffene Tabelle wird neu gemessen.

- [ ] **Step 30: Stagen und Tor**

```bash
git add go.mod go.sum internal/brain/pytext/digest_test.go internal/brain/pytext/unicode.go internal/brain/pytext/unicode_test.go internal/brain/pytext/repr.go internal/brain/pytext/repr_test.go internal/brain/pytext/lines.go internal/brain/pytext/lines_test.go internal/brain/pytext/text.go internal/brain/pytext/text_test.go internal/brain/pytext/isotime.go internal/brain/pytext/isotime_test.go internal/brain/pytext/path.go internal/brain/pytext/path_test.go
```
Expected: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Expected: Exit 0, keine Zeile `not covered:`, keine Zeile `pre-commit: inputs differ from the index`. Gemessen in einer Kopie des committeten Stands (seit `311d5b2` nur Doku) mit diesem Paket: `go test ./... -count=1 -covermode=set -coverpkg=./... -coverprofile=coverage.out` Exit 0 (`internal/brain/pytext` 1,6 s), `go run ./cmd/loomux dev covergate --profile coverage.out` Exit 0.

- [ ] **Step 31: Vor dem Commit prüfen**

```bash
git branch --show-current
```
Expected: `sdd-1b-1` (oder `master`, wenn Task 0 den Hauptcheckout gewählt hat).
```bash
git log -1 --format=%s
```
Expected: `Read the old manifest names and brain state directory until stages 3 and 4` — HEAD ist der Commit von Task 1.
```bash
git rev-parse --short HEAD
```
Expected: der Hash, den Task 1 in seinem Bericht nennt.
```bash
git diff --cached --stat
```
Expected: genau diese 15 Dateien, dann die Summenzeile `15 files changed` mit Einfügungen.

- [ ] **Step 32: Commit**

```bash
printf '%s\n' 'Write text, paths and times the way the Python reference does' '' 'pytext holds repr, splitlines, isspace, strip, read_text, isoformat and' 'fromisoformat, str(Path), NFC and casefold, each measured against Python' '3.14 of the reference. golang.org/x/text v0.38.0 supplies NFC and case' 'folding; Cherokee is folded to its capitals as Python folds it.' > "$TEMP/loomux-1b1-task2-msg.txt"
```
Expected: keine Ausgabe. (Die Nachricht hat `bin/loomux.exe check commit-msg` am 2026-09-15 mit Exit 0 passiert.)
```bash
git commit -F "$TEMP/loomux-1b1-task2-msg.txt"
```
Expected: Exit 0. Der Commit löst das Tor noch einmal aus; das ist gewollt. Danach prüft `.githooks/commit-msg` die Nachricht mit `bin/loomux.exe check commit-msg`, und Git meldet den Commit mit der Betreffzeile `Write text, paths and times the way the Python reference does`.
```bash
git log -1 --format='%an <%ae>'
```
Expected: `Christoph Wübbels <christoph.wuebbels@gmail.com>` (Autor der Commits `e4bf7b5`, `311d5b2`, `5f3cb4c`).
```bash
git log -1 --format=%B
```
Expected: genau die sechs Zeilen aus `$TEMP/loomux-1b1-task2-msg.txt` und eine Leerzeile, keine Zeile `Co-Authored-By`.

- [ ] **Step 33: Bericht**

Der Bericht nennt:
- die Messbefehle und Ausgaben aus Step 29 (wörtlich, oder „unverändert gegenüber dem Plan");
- Umzugsbefunde: keine (kein Umzug). Außerhalb von `internal/brain/pytext` nur `go.mod` (eine Zeile `require golang.org/x/text v0.38.0`) und `go.sum` (zwei Zeilen);
- Ergänzungen zum Vertrag: die Datei `digest_test.go` und die unexportierten Helfer unter „Produces";
- Abweichung vom Vertragswortlaut: `go get` allein schreibt `// indirect`; Step 6 fährt darum `go mod tidy`, das gemessen nur diese Markierung entfernt;
- Toolchain-Bindung: `TestTheTablesAreTheOnesMeasured` verlangt `unicode.Version`, `norm.Version` und `cases.UnicodeVersion` gleich `17.0.0` und bricht sonst mit `pytext tables are measured against Unicode 17.0.0 (Go 1.27); re-measure for unicode <Version>, norm <Version>, cases <Version> against Python 3.14 of the reference` ab. Ein Go älter als 1.27 (das `go 1.25.0` in `go.mod` erlaubt) oder ein anderes `x/text` macht das Tor rot. Gewollt — die Prüfsummen gelten nur für diese Tabellen und werden dann neu gemessen, nicht kopiert —, aber das Tor hängt damit an der Toolchain, nicht nur an `go.mod`. Go 1.27 ist darum Torvoraussetzung (Task 0 Step 2 prüft sie; der Plankopf nennt sie unter Tech Stack);
- `IsSpace` ist exportiert, weil der Vertrag es verlangt; kein Task 3–16 ruft es direkt, `Strip` und `RStrip` bauen darauf. Keine Änderung;
- Startzeit: die `inittrace`-Zeilen von `x/text` (oben) — `cmd/loomux` linkt `pytext` ab Task 3 über `brain/*`, also zahlen auch die Hooks diese Initialisierung (0 ms clock, zusammen rund 7,5 KB und 57 Allokationen); Task 15 misst es am echten Binary.

Paritätszeilen, die dieser Task schafft (Task 12 trägt sie ein):
1. `repr` von Zeichen, die Unicode 17 neu vergibt (4803 Codepunkte in den 47 Bereichen von `newInUnicode17`, der erste U+088F, der letzte U+323B0 bis U+33479): Python 3.14 (Unicode 16) escaped sie als `\uXXXX`/`\UXXXXXXXX`, loomux gibt sie unverändert aus.
2. `casefold` von 28 Buchstaben, die Unicode 17 neu vergibt (U+A7CE, U+A7D2, U+A7D4, U+16EA0 bis U+16EB8): loomux faltet sie, Python nicht. Betrifft `[privacy] never`-Globs mit diesen Zeichen.
3. Ungültiges UTF-8 in einer gelesenen Datei (`_identities.tsv`, `graph.json`, `index.md`, Dokument, Stempel): Python bricht mit `UnicodeDecodeError`-Traceback ab, loomux meldet `{path}: not valid UTF-8`.
4. `Repr` eines Bytes, das kein UTF-8 ist: `\xhh` — Python kennt keinen solchen `str`; erreichbar nur über Argumente oder Dateinamen, die nicht aus `ReadText` stammen.
5. `datetime.fromisoformat`-Formen, die Python zonenbehaftet liest und loomux als „kein Stempel" verwirft (14, gemessen): kleines `t` oder ein anderes Trennzeichen als `T`/Leerzeichen (`X`, `é`), verkürzte Zeiten `HH:MM` und `HH`, Grundform `20000101T000000+0000`, Offset `+0200` und `+02`, Bruch im Offset, Wochendatum `2000-W01-1`, `24:00:00`, Offset-Minute oder -Sekunde 60, Leerzeichen vor dem Offset.
6. `IsoFormat` eines Go-Zeitwerts mit Nanosekunden schneidet unter der Mikrosekunde ab (Python hält keine Nanosekunden; nicht über einen Stempel erreichbar, weil `ParseAwareIsoFormat` höchstens Mikrosekunden liefert).

Exakt über den gemessenen Bereich, keine Paritätszeile: `IsSpace`, `Strip`, `RStrip`, `SplitLines`, `NFC`, `PathString` (die 50 Fälle), `ReadText` für gültiges UTF-8.

---

### Task 3: `internal/brain/privacy` — Umzug, Sichtbarkeit, Glob-Matcher nach Python

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 2 sind committet.

`$SRC` steht für `/c/Users/micro/Documents/#GIT/loomux-src/ub`, `$LOOMUX` für den Arbeitsort aus Task 0 (wie im Umzugsverfahren); Go-Befehle laufen im Arbeitsort. `$TEMP` ist das Temp-Verzeichnis der Git-Bash. Task 1 liefert `config.ReadAreaManifestUntilStage4`, Task 2 `pytext` und `golang.org/x/text` in `go.mod`.

**Files:**
- Move: `$SRC/pkg/privacy/channel.go`, `containment.go`, `readable.go`, `privacy_test.go` → `internal/brain/privacy/` (Paketname bleibt `privacy`, Tests `privacy_test`)
- Modify (nach dem Umzug): `internal/brain/privacy/channel.go`, `containment.go`, `readable.go`, `privacy_test.go`
- Create: `internal/brain/privacy/areas.go`, `internal/brain/privacy/glob.go`
- Test: `internal/brain/privacy/areas_test.go`, `internal/brain/privacy/glob_test.go`, `internal/brain/privacy/privacy_test.go`

**Interfaces:**
- Consumes: Task 1 `config.ReadAreaManifestUntilStage4`, `config.ReadRegistry`, `config.ManifestDir`; Task 2 `pytext.Repr`, `pytext.NFC`, `pytext.CaseFold`. Außerdem vorhanden: `config.ErrNoManifest` (`internal/config/manifest.go:38`), `testlock.Lock(t testing.TB, path string)` (`internal/testlock/lock_windows.go:15`).
- Produces (wörtlich aus dem Vertrag):
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
- Produces — Ergänzungen (unexportiert, Datei `glob.go`, außer wo anders genannt):
```go
func posixParts(p string) (root string, parts []string) // PurePosixPath: Wurzel "", "/" oder "//"; Teile ohne "" und "."
func fullMatchForm(p string) string                     // was full_match dem Übersetzer gibt: str(path), "" ohne Teile
func translateGlob(pattern string) string               // glob.translate(pattern, recursive=True, include_hidden=True, seps="/") als Go-Regex
func translateSegment(segment string) string            // fnmatch._translate(part, "[^/]*", "[^/]")
func bracketBody(pat []rune, i, j int) string           // stuff einer Klammer samt Bereichs-Chunks
func indexHyphen(pat []rune, k, j int) int              // pat.find('-', k, j)
func classBody(stuff string) string                     // Mengenoperatoren, führendes ! ^ [, jedes weitere [ maskiert
func hasDrive(p string) bool                            // containment.go: PureWindowsPath(p).drive != "" ohne Wurzel
func folded(text string) string                         // readable.go: privacy._folded, NFC dann casefold
```
- Entfällt: ``var winDrive = regexp.MustCompile(`^[a-zA-Z]:`)`` (`containment.go:10`), `func globToRegex(pattern string) *regexp.Regexp` (`readable.go:12-46`).

---

- [ ] **Step 1: Umzug, Verfahren Schritt 1 — nur die vier Dateien**

```bash
mkdir -p "$LOOMUX/internal/brain/privacy"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
cp "$SRC/pkg/privacy/channel.go" "$SRC/pkg/privacy/containment.go" "$SRC/pkg/privacy/readable.go" "$SRC/pkg/privacy/privacy_test.go" "$LOOMUX/internal/brain/privacy/"
```
Erwartet: keine Ausgabe, Exit 0.

- [ ] **Step 2: Verfahren Schritte 2–3 — Paketname bleibt, Importpfade umschreiben**

Schritt 2 entfällt (`package privacy`/`package privacy_test` bleiben). Schritt 3, drei Aufrufe; ein Bezeichnerumbau entfällt, weil `config`, `testlock` und `privacy` ihre Namen behalten:
```bash
grep -rl 'github.com/xidus90/ultra-brain/pkg/config' "$LOOMUX" --include=*.go | xargs -r sed -i 's#github.com/xidus90/ultra-brain/pkg/config#github.com/xidus90/loomux/internal/config#g'
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rl 'github.com/xidus90/ultra-brain/internal/testlock' "$LOOMUX" --include=*.go | xargs -r sed -i 's#github.com/xidus90/ultra-brain/internal/testlock#github.com/xidus90/loomux/internal/testlock#g'
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rl 'github.com/xidus90/ultra-brain/pkg/privacy' "$LOOMUX" --include=*.go | xargs -r sed -i 's#github.com/xidus90/ultra-brain/pkg/privacy#github.com/xidus90/loomux/internal/brain/privacy#g'
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rl 'github.com/xidus90/ultra-brain/' "$LOOMUX/internal/brain/privacy" --include=*.go
```
Erwartet: keine Ausgabe, Exit 1.

- [ ] **Step 3: Verfahren Schritt 4 — formatieren, prüfen, erwartetes Rot lesen**

Kein `go mod tidy`: das Paket importiert nur `config` und `testlock`; `golang.org/x/text` hat Task 2 geholt.
```bash
gofmt -w internal/brain/privacy
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/privacy/...
```
Erwartet: keine Ausgabe.
```bash
go test -count=1 ./internal/brain/privacy/ > "$TEMP/task03-step3.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
grep -E -o -- '--- FAIL: [^ ]+' "$TEMP/task03-step3.txt"
```
Erwartet (gemessen 2026-09-15 gegen loomux' `config` mit Task 1): **nicht grün**, genau diese sechs Zeilen — der Test und seine fünf scheiternden Untertests; alle anderen Tests bestehen:
```
--- FAIL: TestVisibleManifest
--- FAIL: TestVisibleManifest/open
--- FAIL: TestVisibleManifest/local_only
--- FAIL: TestVisibleManifest/not_TOML
--- FAIL: TestVisibleManifest/misspelt_mode
--- FAIL: TestVisibleManifest/closed_config.toml_that_cannot_be_read_beside_an_open_.brain.toml
```
Grund: der umgezogene `VisibleManifest` ruft noch `config.ReadManifest`, das in loomux nur `.loomux/config.toml` kennt; der Test schreibt `.brain.toml` und `.ultra-brain/config.toml`. Das ist die erste Verhaltenskorrektur dieses Tasks (Schritte 4–6), kein Anlass, etwas anderes anzufassen. Das Umzugsverfahren im Plankopf lässt dieses Rot nach Schritt 4 für Task 3 ausdrücklich zu; jedes andere Rot an dieser Stelle heißt anhalten.

- [ ] **Step 4: `VisibleManifest` test-first — Test umstellen**

In `internal/brain/privacy/privacy_test.go` den Importblock ersetzen durch:
```go
import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/testlock"
)
```
Den Kommentar über `TestVisibleManifest` und die ganze Funktion (ub `privacy_test.go:58-111`) ersetzen durch die folgenden zwei Tests (der Fall `closed config.toml that cannot be read beside an open .brain.toml` bleibt wortgleich, neu sind `wantErr`, die Fälle mit `.loomux/config.toml` und der `ErrNoManifest`-Test):
```go
// VisibleManifest is the one gate the visibility callers share. Every failure
// is an error, the absent declaration included: `_visible_areas` lets
// `read_manifest` raise for each registered area, so an area without a usable
// declaration stops the whole call instead of being served or hidden. The
// names are read in the order config.ReadAreaManifestUntilStage4 gives them:
// the first one that exists as a file is the declaration, readable or not,
// except a .loomux/config.toml without an [area] table, which declares nothing.
func TestVisibleManifest(t *testing.T) {
	const open = "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"manual_cloud\"\n"
	const closed = "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"local_only\"\n"
	tests := []struct {
		name         string
		build        func(t *testing.T, dir string)
		local, cloud bool
		wantManifest bool
		wantErr      bool
	}{
		{"no manifest", func(t *testing.T, dir string) {}, false, false, false, true},
		{"open", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), open)
		}, true, true, true, false},
		{"local_only", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), closed)
		}, true, false, true, false},
		{"not TOML", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), "[area\n")
		}, false, false, false, true},
		{"misspelt mode", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"lokal_only\"\n")
		}, false, false, false, true},
		{"closed config.toml that cannot be read beside an open .brain.toml", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), open)
			path := filepath.Join(dir, ".ultra-brain", "config.toml")
			writeFile(t, path, closed)
			testlock.Lock(t, path)
		}, false, false, false, true},
		{"closed .loomux/config.toml that cannot be read beside an open .brain.toml", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), open)
			path := filepath.Join(dir, ".loomux", "config.toml")
			writeFile(t, path, closed)
			testlock.Lock(t, path)
		}, false, false, false, true},
		{"the loomux name before an open .brain.toml", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), open)
			writeFile(t, filepath.Join(dir, ".loomux", "config.toml"), closed)
		}, true, false, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.build(t, dir)
			for _, c := range []struct {
				ch   privacy.Channel
				want bool
			}{{privacy.ChannelLocal, tc.local}, {privacy.ChannelCloud, tc.cloud}} {
				m, visible, err := privacy.VisibleManifest(dir, c.ch)
				if (err != nil) != tc.wantErr {
					t.Errorf("%s: err = %v, want an error: %v", c.ch, err, tc.wantErr)
				}
				if visible != c.want {
					t.Errorf("%s: visible = %v, want %v", c.ch, visible, c.want)
				}
				if (m != nil) != tc.wantManifest {
					t.Errorf("%s: manifest = %+v, want one exactly for a readable declaration", c.ch, m)
				}
			}
		})
	}
}

func TestVisibleManifestWithoutDeclarationIsErrNoManifest(t *testing.T) {
	_, _, err := privacy.VisibleManifest(t.TempDir(), privacy.ChannelLocal)
	if !errors.Is(err, config.ErrNoManifest) {
		t.Fatalf("err = %v, want config.ErrNoManifest", err)
	}
}
```

- [ ] **Step 5: Scheitern sehen**

```bash
go test -count=1 ./internal/brain/privacy/ > "$TEMP/task03-step5.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
sed -n -E 's/^.*privacy_test\.go:[0-9]+:[0-9]+: //p' "$TEMP/task03-step5.txt"
```
Erwartet, genau diese zwei Zeilen (gemessen; Zeilennummern variieren mit dem Kommentar, der `sed` schneidet Pfad, Zeile und Spalte ab):
```
assignment mismatch: 3 variables but privacy.VisibleManifest returns 2 values
assignment mismatch: 3 variables but privacy.VisibleManifest returns 2 values
```
```bash
grep -E '^FAIL	github\.com/xidus90/loomux/internal/brain/privacy \[build failed\]$' "$TEMP/task03-step5.txt"
```
Erwartet: genau die Zeile `FAIL	github.com/xidus90/loomux/internal/brain/privacy [build failed]`, Exit 0.

- [ ] **Step 6: `VisibleManifest` implementieren**

`internal/brain/privacy/channel.go`, Importblock vorher:
```go
import (
	"errors"
	"fmt"

	"github.com/xidus90/loomux/internal/config"
)
```
nachher:
```go
import (
	"fmt"

	"github.com/xidus90/loomux/internal/config"
)
```
Kommentar und Funktion `VisibleManifest` (ub `channel.go:30-60`) ersetzen durch:
```go
// VisibleManifest reads the declaration lying in dir and answers it together
// with whether this channel may see the area at all. It is the one gate of
// every surface that serves an area -- search, catalog, read, neighbors,
// status -- and it exists because each of them had the same hole: the
// manifest error was discarded, and IsVisible counts a nil manifest as
// visible, so an area whose `local_only` declaration could not be read was
// served on the cloud channel (finding N3 of the scheibe-6 merge re-review).
//
// Every failure is the caller's error, the absent declaration included. That
// is `_visible_areas` (src/brain/core.py:248-263): `read_manifest` of
// `manifest_path` raises for a file that does not exist, does not read, is not
// TOML or names no known mode, nothing on the way to the command line catches
// it, and so one area without a usable declaration stops the whole call.
// ultra-brain's Go gate answered an absent declaration as visible and hid the
// other failures silently; loomux follows the reference.
//
// The file is found the way config.ReadAreaManifestUntilStage4 finds it:
// `.loomux/config.toml`, else ultra-brain's two names until stage 4.
//
// dir is where the manifest lies, which is not always the area: a read-only
// area keeps it in the state directory, so callers pass config.ManifestDir --
// the directory `registry.manifest_path` reads on the Python side.
func VisibleManifest(dir string, ch Channel) (*config.Manifest, bool, error) {
	manifest, err := config.ReadAreaManifestUntilStage4(dir)
	if err != nil {
		return nil, false, err
	}
	return manifest, IsVisible(manifest, ch), nil
}
```
Im Kommentar über `IsVisible` (ub `channel.go:62-67`) den zweiten Absatz vorher:
```go
// A nil manifest is visible, and only the absent declaration reaches it that
// way: VisibleManifest is where a manifest that exists and cannot be used is
// told apart from one that was never written.
```
nachher:
```go
// A nil manifest is visible. VisibleManifest never hands one on; the arm
// answers a caller that holds no declaration at all.
```
Der Code von `ParseChannel` und `IsVisible` bleibt wortgleich.
```bash
go test -count=1 ./internal/brain/privacy/
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/privacy`.

- [ ] **Step 7: `Contained` test-first — Test nach `_contained`**

In `privacy_test.go` die Zeile `"strings"` aus dem Importblock streichen (der neue Test prüft den Wortlaut exakt) und `TestContained` (ub `privacy_test.go:123-168`) ersetzen durch:
```go
// Contained is `_contained` (src/brain/core.py:649-668). Every answer below was
// measured against it on 2026-09-15 under Python 3.14.7.
func TestContained(t *testing.T) {
	good := []struct {
		relative string
		expected string
	}{
		{"doc.md", "doc.md"},
		{"sub/doc.md", "sub/doc.md"},
		{"sub\\doc.md", "sub/doc.md"},
		{"a/b/c.md", "a/b/c.md"},
		{"./a.md", "a.md"},
		{"a//b.md", "a/b.md"},
		{"a\\\\b.md", "a/b.md"},
		{"a/./b.md", "a/b.md"},
		{"a/", "a"},
		{"a/b/", "a/b"},
		{"x\\", "x"},
		{".", "."},
		{"", "."},
		{"./", "."},
		{":x.md", ":x.md"},
		{"ab:c.md", "ab:c.md"},
		{"a/1:b.md", "a/1:b.md"},
		{"a//C:x", "a/C:x"},
		{"a..b.md", "a..b.md"},
		{"...", "..."},
	}

	for _, tc := range good {
		res, err := privacy.Contained("area", tc.relative)
		if err != nil {
			t.Errorf("expected %q to be contained, got error: %v", tc.relative, err)
		}
		if res != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, res)
		}
	}

	bad := []string{
		"../escape.md",
		"..\\escape.md",
		"sub/../../escape.md",
		"a/../b.md",
		"..",
		"/abs/doc.md",
		"\\abs\\doc.md",
		"C:/abs/doc.md",
		"C:\\abs\\doc.md",
		"C:",
		"d:relative.md",
		"1:foo.md",
		"é:x.md",
		"./1:foo.md",
		".\\1:x.md",
		".//d:x",
		"//server/share/doc.md",
		"\\\\server\\share\\doc.md",
		"\\",
		"\\\\",
	}

	for _, relative := range bad {
		_, err := privacy.Contained("test-scope", relative)
		if err == nil {
			t.Errorf("expected %q to be rejected, got nil", relative)
			continue
		}
		if want := "test-scope/" + relative + " leaves the area"; err.Error() != want {
			t.Errorf("expected %q, got %q", want, err.Error())
		}
	}
}
```
Messung (Python 3.14.7, `brain.core._contained("s", relative)` je Pfad): angenommen `./a.md`→`a.md`, `a//b.md`→`a/b.md`, `a\\b.md`→`a/b.md`, `a/./b.md`→`a/b.md`, `a/`→`a`, `x\`→`x`, `.`/``/`./`→`.`, `:x.md`, `ab:c.md`, `a/1:b.md`, `a//C:x`→`a/C:x`, `a..b.md`, `...`; verweigert mit `s/{relative} leaves the area` (roh): `1:foo.md`, `é:x.md`, `./1:foo.md`, `.\1:x.md`, `.//d:x`, `C:`, `\abs\doc.md`, `\\server\share\doc.md`, `\`, `\\`, `a/../b.md`, `..`. Python 3.14 baut `PureWindowsPath(candidate)` aus `candidate.as_posix()` (`PurePath.__init__`), darum zählt das Laufwerk der bereinigten Form.

- [ ] **Step 8: Scheitern sehen**

```bash
go test -count=1 -run TestContained ./internal/brain/privacy/ > "$TEMP/task03-step8.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
sed -n -E 's/^[[:space:]]*privacy_test\.go:[0-9]+: //p' "$TEMP/task03-step8.txt"
```
Erwartet, genau diese elf Zeilen (gemessen; Zeilennummern variieren mit dem Kommentar, der `sed` schneidet das Präfix `privacy_test.go:<zeile>: ` ab):
```
expected "." to be contained, got error: area/. leaves the area
expected ".", got ""
expected "" to be contained, got error: area/ leaves the area
expected ".", got ""
expected "./" to be contained, got error: area/./ leaves the area
expected ".", got ""
expected "1:foo.md" to be rejected, got nil
expected "é:x.md" to be rejected, got nil
expected "./1:foo.md" to be rejected, got nil
expected ".\\1:x.md" to be rejected, got nil
expected ".//d:x" to be rejected, got nil
```

- [ ] **Step 9: `Contained` implementieren**

`internal/brain/privacy/containment.go` vollständig ersetzen (vorher: `regexp`/`path`-Importe, `var winDrive = regexp.MustCompile(`^[a-zA-Z]:`)`, `path.Clean` und eine Verweigerung für `.`; der Fehlerwortlaut `"%s/%s leaves the area"` bleibt):
```go
package privacy

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Contained verifies that relative stays inside the area of scope and returns
// it the way `PurePosixPath(...).as_posix()` spells it. It is `_contained`
// (src/brain/core.py:649-668), checked lexically and before anything touches
// the filesystem, "so a refusal cannot be told apart from a path that simply
// is not there".
//
// Backslashes become slashes first. The path is refused when it has a root,
// when PureWindowsPath finds a drive in it, or when a `..` component remains.
// Python 3.14 builds the PureWindowsPath from the cleaned posix form
// (`PurePath.__init__` takes `as_posix()` of a path of the other flavour), so
// the drive is asked of the cleaned path: `./1:foo.md` is refused like
// `1:foo.md`. A drive is any one character before `:` -- measured, `é:x.md`
// is refused, `:x.md` and `ab:c.md` pass. The empty path and `.` are not
// refused; they clean to `.`, as in Python.
func Contained(scope, relative string) (string, error) {
	root, parts := posixParts(strings.ReplaceAll(relative, `\`, "/"))
	cleaned := strings.Join(parts, "/")
	if root != "" || hasDrive(cleaned) {
		return "", fmt.Errorf("%s/%s leaves the area", scope, relative)
	}
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("%s/%s leaves the area", scope, relative)
		}
	}
	if cleaned == "" {
		return ".", nil
	}
	return cleaned, nil
}

// hasDrive is `PureWindowsPath(p).drive != ""` for a path without a root:
// one character, counted in runes as pathlib counts code points, then `:`.
func hasDrive(p string) bool {
	_, size := utf8.DecodeRuneInString(p)
	return size > 0 && len(p) > size && p[size] == ':'
}
```
`internal/brain/privacy/glob.go` anlegen, vorerst nur mit der Zerlegung (Step 12 ersetzt die Datei um den Übersetzer erweitert):
```go
package privacy

import "strings"

// posixParts parses p the way PurePosixPath does (`PurePath._parse_path` with
// `posixpath.splitroot`): the root is "" for a relative path, "//" for exactly
// two leading slashes -- POSIX leaves that form to the implementation and
// pathlib keeps it -- and "/" for one or three and more. The parts are the
// components between slashes that are neither empty nor `.`; `..` stays a
// part, because pathlib is lexical.
func posixParts(p string) (root string, parts []string) {
	rest := p
	switch {
	case !strings.HasPrefix(p, "/"):
	case strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "///"):
		root, rest = "//", p[2:]
	default:
		root, rest = "/", p[1:]
	}
	for _, part := range strings.Split(rest, "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return root, parts
}
```
Messung der Zerlegung (`PurePosixPath(s)`): `//a`→`('//','a')`, `///a`→`('/','a')`, `/`→`('/',)`, `//`→`('//',)`, `a/`→`('a',)`, `.` und ``→`()`, `a//b/./c/`→`('a','b','c')`.
```bash
go test -count=1 ./internal/brain/privacy/
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/privacy`.

- [ ] **Step 10: `MatchesGlobs` test-first — Tabellentest nach `matches_globs`**

`internal/brain/privacy/glob_test.go` anlegen. Die NFD-Schreibweise steht als `string(rune(0x0301))`, damit kein Editor sie still nach NFC faltet:
```go
package privacy_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
)

// Every answer below was measured against `brain.privacy.matches_globs((pattern,), relative)`
// on 2026-09-15 under Python 3.14.7: PurePosixPath.full_match over NFC and casefold, with
// the pattern translated by glob.translate and fnmatch._translate.
func TestMatchesGlobsAnswersLikePython(t *testing.T) {
	nfd := "cafe" + string(rune(0x0301))
	nfc := "caf" + string(rune(0x00e9))
	for _, tc := range []struct {
		pattern, relative string
		want              bool
	}{
		// ** as zero or more segments, * within one segment
		{"a/**/b", "a/b", true},
		{"**/x.md", "x.md", true},
		{"a/**", "a", false},
		{"review/**", "review", false},
		{"**", "a/b/c", true},
		{"**", "", true},
		{"**", ".", true},
		{".", "", true},
		{"*", "a/b", false},
		{"*", ".hidden", true},
		{"*.md", "dir/x.md", false},
		{"*.md", ".md", true},
		{"a/*", "a/b", true},
		{"a/*/c", "a/b/c", true},
		{"a/**/**/b", "a/x/y/b", true},
		{"**/secrets/**", "secrets", false},
		{"**/secrets/**", "x/secrets/y", true},
		{"a*b*c", "axxbyyc", true},
		{"***", "abc", true},
		{"?*", "", false},
		{"a*b", "a\nb", true},
		{"test?.txt", "test1.txt", true},
		{"?", "/", false},
		{"x[!/]y", "x/y", false},
		// both sides normalised like PurePosixPath; no backslash turns into a slash
		{"a//b/./c", "a/b/c", true},
		{"a/b/c", "a//b/./c", true},
		{"a/", "a", true},
		{"//a", "//a", true},
		{"/a", "a", false},
		{"secrets/**", "secrets\\key.txt", false},
		// literals are escaped
		{"a.md", "aXmd", false},
		{"a(b)", "a(b)", true},
		{"a+", "aa", false},
		// NFC and casefold on both sides
		{"ß.md", "SS.md", true},
		{"ẞ.md", "ss.md", true},
		{"ﬁle.md", "FILE.md", true},
		{"Σ.md", "ς.md", true},
		{"**/*.PEM", "k.pem", true},
		{nfd + "/*", nfc + "/x.md", true},
		{nfc + "/*", nfd + "/x.md", true},
		// bracket expressions as fnmatch._translate builds them
		{"[a-z]*.md", "x.md", true},
		{"[!a]*.md", "a.md", false},
		{"[]a]", "]", true},
		{"[!]a]", "b", true},
		{"[!]a]", "]", false},
		{"[a-]", "-", true},
		{"[z-a]x", "x", false},
		{"[z-a]x", "zx", false},
		{"[!z-a]", "q", true},
		{"[a&&b]", "&", true},
		{"[~]", "~", true},
		{"[|]", "|", true},
		{"[^a]", "^", true},
		{"[^a]", "b", false},
		{"[[:alpha:]]", "a", false},
		{"[[:alpha:]]", "[", false},
		{"[[:alpha:]]", ":]", true},
		{"[a[:alpha:]]", "a]", true},
		{"[a[:alpha:]]", "l]", true},
		{"[\\]", "\\", true},
		{"[\\-z]", "a", true},
		{"[abc", "[abc", true},
		{"[!", "[!", true},
		{"[!]", "[!]", true},
		{"[a-c-e]", "d", false},
		{"[a-c-e]", "-", true},
		{"[b-a-z]", "c", false},
		{"[b-a-z]", "-", true},
		{"[a--z]", "-", false},
		{"[a--z]", "z", true},
		{"[a--z]", "a", false},
		{"[!b-a]", "x", true},
		{"[ä-ö]", "é", true},
	} {
		if got := privacy.MatchesGlobs([]string{tc.pattern}, tc.relative); got != tc.want {
			t.Errorf("MatchesGlobs(%q, %q) = %v, want %v", tc.pattern, tc.relative, got, tc.want)
		}
	}
}

func TestMatchesGlobsAsksEveryPattern(t *testing.T) {
	if !privacy.MatchesGlobs([]string{"*.pem", "secrets/**"}, "secrets/a.md") {
		t.Error("the second pattern must be asked when the first does not match")
	}
	if privacy.MatchesGlobs(nil, "a.md") {
		t.Error("no pattern matches nothing")
	}
}
```
`TestIsReadable` bleibt wortgleich: alle elf Pfade gegen `secrets/**`, `*.pem`, `**/private/**`, `test?.txt` wurden an Python gemessen und antworten dort wie im ub-Test.

- [ ] **Step 11: Scheitern sehen**

```bash
go test -count=1 -run 'TestMatchesGlobs|TestIsReadable' ./internal/brain/privacy/ > "$TEMP/task03-step11.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
grep -E -o -- '--- FAIL: [^ ]+' "$TEMP/task03-step11.txt"
```
Erwartet: genau die Zeile `--- FAIL: TestMatchesGlobsAnswersLikePython` (`TestMatchesGlobsAsksEveryPattern` und `TestIsReadable` bestehen).
```bash
sed -n -E 's/^[[:space:]]*glob_test\.go:[0-9]+: //p' "$TEMP/task03-step11.txt"
```
Erwartet, genau diese 31 Zeilen (gemessen; Zeilennummern variieren mit dem Kommentar, der `sed` schneidet das Präfix `glob_test.go:<zeile>: ` ab):
```
MatchesGlobs("a/**/b", "a/b") = false, want true
MatchesGlobs(".", "") = false, want true
MatchesGlobs("a//b/./c", "a/b/c") = false, want true
MatchesGlobs("a/b/c", "a//b/./c") = false, want true
MatchesGlobs("a/", "a") = false, want true
MatchesGlobs("secrets/**", "secrets\\key.txt") = true, want false
MatchesGlobs("ß.md", "SS.md") = false, want true
MatchesGlobs("ẞ.md", "ss.md") = false, want true
MatchesGlobs("ﬁle.md", "FILE.md") = false, want true
MatchesGlobs("Σ.md", "ς.md") = false, want true
MatchesGlobs("café/*", "café/x.md") = false, want true
MatchesGlobs("café/*", "café/x.md") = false, want true
MatchesGlobs("[a-z]*.md", "x.md") = false, want true
MatchesGlobs("[]a]", "]") = false, want true
MatchesGlobs("[!]a]", "b") = false, want true
MatchesGlobs("[a-]", "-") = false, want true
MatchesGlobs("[!z-a]", "q") = false, want true
MatchesGlobs("[a&&b]", "&") = false, want true
MatchesGlobs("[~]", "~") = false, want true
MatchesGlobs("[|]", "|") = false, want true
MatchesGlobs("[^a]", "^") = false, want true
MatchesGlobs("[[:alpha:]]", ":]") = false, want true
MatchesGlobs("[a[:alpha:]]", "a]") = false, want true
MatchesGlobs("[a[:alpha:]]", "l]") = false, want true
MatchesGlobs("[\\]", "\\") = false, want true
MatchesGlobs("[\\-z]", "a") = false, want true
MatchesGlobs("[a-c-e]", "-") = false, want true
MatchesGlobs("[b-a-z]", "-") = false, want true
MatchesGlobs("[a--z]", "z") = false, want true
MatchesGlobs("[!b-a]", "x") = false, want true
MatchesGlobs("[ä-ö]", "é") = false, want true
```
(31 Zeilen; die zwei `café`-Zeilen sind die NFD/NFC-Paare.)

- [ ] **Step 12: `MatchesGlobs` implementieren**

`internal/brain/privacy/glob.go` vollständig ersetzen. Der Übersetzer folgt der Quelle der Referenz-Python (`python -c "import glob,fnmatch,inspect; print(inspect.getsource(glob.translate)); print(inspect.getsource(fnmatch._translate))"`, gelesen 2026-09-15, Python 3.14.7) Zweig für Zweig; `PurePath.full_match` ruft `_GlobberBase.compile` → `_compile_pattern(pat, "/", case_sensitive=True, recursive=True)` → `translate(pat, recursive=True, include_hidden=True, seps="/")`. `glob.translate` nutzt `_join_translated_parts` nicht, also keine atomaren Gruppen:
```go
package privacy

import (
	"regexp"
	"slices"
	"strings"
)

// posixParts parses p the way PurePosixPath does (`PurePath._parse_path` with
// `posixpath.splitroot`): the root is "" for a relative path, "//" for exactly
// two leading slashes -- POSIX leaves that form to the implementation and
// pathlib keeps it -- and "/" for one or three and more. The parts are the
// components between slashes that are neither empty nor `.`; `..` stays a
// part, because pathlib is lexical.
func posixParts(p string) (root string, parts []string) {
	rest := p
	switch {
	case !strings.HasPrefix(p, "/"):
	case strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "///"):
		root, rest = "//", p[2:]
	default:
		root, rest = "/", p[1:]
	}
	for _, part := range strings.Split(rest, "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return root, parts
}

// fullMatchForm is the string `PurePath.full_match` hands to the translator,
// for the path and for the pattern alike: `str(path)` when the path has parts,
// and "" when it has none, because "the string representation of an empty path
// is a single dot ('.')".
func fullMatchForm(p string) string {
	root, parts := posixParts(p)
	return root + strings.Join(parts, "/")
}

// translateGlob is `glob.translate(pattern, recursive=True,
// include_hidden=True, seps="/")` of Python 3.14 -- the call
// `_GlobberBase.compile` makes for `full_match` on a PurePosixPath -- written
// as a Go regular expression. The four constants are the `include_hidden` arm
// of that function spelt out for the one separator. Python matches with
// `re.match`, which anchors at the start; the leading `^` says the same.
func translateGlob(pattern string) string {
	const (
		oneLastSegment  = `[^/]+`
		oneSegment      = oneLastSegment + `/`
		anySegments     = `(?:.+/)?`
		anyLastSegments = `.*`
	)
	var results strings.Builder
	parts := strings.Split(pattern, "/")
	last := len(parts) - 1
	for idx, part := range parts {
		switch {
		case part == "*":
			if idx < last {
				results.WriteString(oneSegment)
			} else {
				results.WriteString(oneLastSegment)
			}
		case part == "**":
			if idx == last {
				results.WriteString(anyLastSegments)
			} else if parts[idx+1] != "**" {
				results.WriteString(anySegments)
			}
		default:
			if part != "" {
				results.WriteString(translateSegment(part))
			}
			if idx < last {
				results.WriteString("/")
			}
		}
	}
	return `^(?s:` + results.String() + `)\z`
}

// translateSegment is `fnmatch._translate(part, "[^/]*", "[^/]")` of Python
// 3.14, joined, for one segment. It walks code points, as Python indexes a
// str, and keeps the branches of the original in their order: a run of `*`,
// `?`, a bracket expression, an unclosed `[`, a literal.
//
// Three spellings differ because RE2 is not Python's `re`; what they match
// does not. An empty range, `(?!)` in Python, is a class of no character,
// since RE2 has no lookahead. A literal goes through regexp.QuoteMeta instead
// of `re.escape`. And classBody escapes `[` inside a class.
func translateSegment(segment string) string {
	pat := []rune(segment)
	n := len(pat)
	var res strings.Builder
	for i := 0; i < n; {
		c := pat[i]
		i++
		switch c {
		case '*':
			res.WriteString(`[^/]*`)
			for i < n && pat[i] == '*' {
				i++
			}
		case '?':
			res.WriteString(`[^/]`)
		case '[':
			j := i
			if j < n && pat[j] == '!' {
				j++
			}
			if j < n && pat[j] == ']' {
				j++
			}
			for j < n && pat[j] != ']' {
				j++
			}
			if j >= n {
				res.WriteString(`\[`)
				continue
			}
			stuff := bracketBody(pat, i, j)
			i = j + 1
			switch stuff {
			case "":
				res.WriteString(`[^\x00-\x{10FFFF}]`)
			case "!":
				res.WriteString(`.`)
			default:
				res.WriteString("[" + classBody(stuff) + "]")
			}
		default:
			res.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return res.String()
}

// bracketBody is the part of `fnmatch._translate` that turns the characters
// between `[` (at i) and `]` (at j) into `stuff`: without a `-`, backslashes
// doubled; with one, the chunks between range hyphens, a trailing hyphen kept,
// empty ranges removed, and backslashes and the hyphens that make no range
// escaped.
func bracketBody(pat []rune, i, j int) string {
	if !slices.Contains(pat[i:j], '-') {
		return strings.ReplaceAll(string(pat[i:j]), `\`, `\\`)
	}
	var chunks [][]rune
	k := i + 1
	if pat[i] == '!' {
		k = i + 2
	}
	for {
		k = indexHyphen(pat, k, j)
		if k < 0 {
			break
		}
		chunks = append(chunks, pat[i:k])
		i = k + 1
		k += 3
	}
	if chunk := pat[i:j]; len(chunk) > 0 {
		chunks = append(chunks, chunk)
	} else {
		last := len(chunks) - 1
		chunks[last] = append(slices.Clone(chunks[last]), '-')
	}
	for k := len(chunks) - 1; k > 0; k-- {
		if prev := chunks[k-1]; prev[len(prev)-1] > chunks[k][0] {
			chunks[k-1] = append(slices.Clone(prev[:len(prev)-1]), chunks[k][1:]...)
			chunks = slices.Delete(chunks, k, k+1)
		}
	}
	escaped := make([]string, len(chunks))
	for m, chunk := range chunks {
		escaped[m] = strings.ReplaceAll(strings.ReplaceAll(string(chunk), `\`, `\\`), "-", `\-`)
	}
	return strings.Join(escaped, "-")
}

// indexHyphen is `pat.find('-', k, j)`: the first `-` at k or after and
// before j, or -1.
func indexHyphen(pat []rune, k, j int) int {
	for ; k < j; k++ {
		if pat[k] == '-' {
			return k
		}
	}
	return -1
}

// classBody finishes `stuff` the way `fnmatch._translate` does before it wraps
// it in brackets -- `&`, `~` and `|` escaped, a leading `!` made the
// negation, a leading `^` or `[` escaped -- and then escapes every other `[`:
// Python reads it as a literal, RE2 would read `[:alpha:]` as a POSIX class.
// Every backslash in stuff opens a pair with an ASCII character, so the pair
// is copied whole.
func classBody(stuff string) string {
	stuff = strings.NewReplacer("&", `\&`, "~", `\~`, "|", `\|`).Replace(stuff)
	switch stuff[0] {
	case '!':
		stuff = "^" + stuff[1:]
	case '^', '[':
		stuff = `\` + stuff
	}
	var body strings.Builder
	for i := 0; i < len(stuff); i++ {
		switch stuff[i] {
		case '\\':
			body.WriteString(stuff[i : i+2])
			i++
		case '[':
			body.WriteString(`\[`)
		default:
			body.WriteByte(stuff[i])
		}
	}
	return body.String()
}
```
In `internal/brain/privacy/readable.go`: `globToRegex` samt Kommentar (ub `readable.go:12-46`) streichen; `MatchesGlobs` samt Kommentar (ub `readable.go:48-59`) ersetzen durch:
```go
// MatchesGlobs reports whether relative matches any of the patterns. It is
// `matches_globs` (src/brain/privacy.py:31-74): `PurePosixPath(folded(relative))
// .full_match(folded(pattern))`, folded being NFC and then casefold, on both
// sides and on every platform. The docstring there gives the reason for both
// folds: a spelling variant must not walk around a `never` pattern wherever
// the filesystem opens the file anyway.
//
// No backslash becomes a slash here, because Python turns none: to
// PurePosixPath `secrets\key.txt` is one name. The callers hand in register
// paths and the output of Contained, both spelt with slashes.
//
// Each pattern is compiled on every call. Python caches the compiled form;
// the lists are a handful of globs, and a cache would be state this package
// keeps nowhere else.
func MatchesGlobs(patterns []string, relative string) bool {
	candidate := fullMatchForm(folded(relative))
	for _, pattern := range patterns {
		if regexp.MustCompile(translateGlob(fullMatchForm(folded(pattern)))).MatchString(candidate) {
			return true
		}
	}
	return false
}

// folded is `_folded` (src/brain/privacy.py:73-74).
func folded(text string) string {
	return pytext.CaseFold(pytext.NFC(text))
}
```
Importblock von `readable.go` für diesen Zwischenstand (das alte `ReviewExcludes` braucht noch `path` und `strings`):
```go
import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)
```
```bash
go test -count=1 ./internal/brain/privacy/
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/privacy`.

- [ ] **Step 13: `ReviewExcludes` test-first — Test nach `walk.review_excludes`**

In `privacy_test.go` im Importblock `"slices"` nach `"path/filepath"` ergänzen. `TestReviewExcludes` beginnt in ub (`privacy_test.go:209-213`, bleibt wortgleich) mit
```go
func TestReviewExcludes(t *testing.T) {
	excludes, err := privacy.ReviewExcludes(nil)
	if err != nil || len(excludes) != 0 {
		t.Errorf("expected empty excludes for nil manifest, got %v, err: %v", excludes, err)
	}
```
`err` ist damit im Funktionsscope deklariert, und die neue Zuweisung `_, err = privacy.ReviewExcludes(&config.Manifest{LayoutReview: "./"})` unten übersetzt; die Schleifen deklarieren ihr eigenes `err`. In `TestReviewExcludes` die letzte Schleife vorher (ub `privacy_test.go:230-236`):
```go
	for _, invalid := range []string{".", "/", "\\", "///"} {
		mInvalid := &config.Manifest{LayoutReview: invalid}
		_, err := privacy.ReviewExcludes(mInvalid)
		if err == nil {
			t.Errorf("expected error for invalid LayoutReview %q, got nil", invalid)
		}
	}
```
nachher:
```go
	// `review_excludes` (src/brain/walk.py:96-113) refuses a value only when
	// `PurePosixPath(review).parts` is empty, and uses every other value as
	// written. Measured on 2026-09-15 under Python 3.14.7.
	for _, invalid := range []string{".", "./.", ".//."} {
		mInvalid := &config.Manifest{LayoutReview: invalid}
		_, err := privacy.ReviewExcludes(mInvalid)
		if err == nil {
			t.Errorf("expected error for invalid LayoutReview %q, got nil", invalid)
		}
	}
	_, err = privacy.ReviewExcludes(&config.Manifest{LayoutReview: "./"})
	if want := "[layout] review must not resolve to the area root, found './'"; err == nil || err.Error() != want {
		t.Errorf("expected %q, got %v", want, err)
	}
	for _, tc := range []struct {
		review string
		want   []string
	}{
		{"/", []string{"//**", "/"}},
		{"\\", []string{"\\/**", "\\"}},
		{"///", []string{"////**", "///"}},
		{" review ", []string{" review /**", " review "}},
		{"a/..", []string{"a/../**", "a/.."}},
		{"it's/.", []string{"it's/./**", "it's/."}},
	} {
		excludes, err := privacy.ReviewExcludes(&config.Manifest{LayoutReview: tc.review})
		if err != nil || !slices.Equal(excludes, tc.want) {
			t.Errorf("LayoutReview %q: got %q, err %v; want %q", tc.review, excludes, err, tc.want)
		}
	}
```
`./` steht nicht in der Schleife, weil die Zeile danach es mit dem exakten Wortlaut prüft. Messung (`review_excludes(Manifest(scope="k", layout={"review": v}))`, Python 3.14.7, 2026-09-15): `ValueError` mit `[layout] review must not resolve to the area root, found '.'` für `.`, `[layout] review must not resolve to the area root, found './'` für `./`, `[layout] review must not resolve to the area root, found './.'` für `./.` und `[layout] review must not resolve to the area root, found './/.'` für `.//.` (der Wert jeweils in `repr`); `/`→`('//**', '/')`, `\`→`('\\/**', '\\')`, `///`→`('////**', '///')`, ` review `→`(' review /**', ' review ')`, `a/..`→`('a/../**', 'a/..')`, `it's/.`→`("it's/./**", "it's/.")`, `""`→`()`.

- [ ] **Step 14: Scheitern sehen**

```bash
go test -count=1 -run TestReviewExcludes ./internal/brain/privacy/ > "$TEMP/task03-step14.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
sed -n -E 's/^[[:space:]]*privacy_test\.go:[0-9]+: //p' "$TEMP/task03-step14.txt"
```
Erwartet, genau diese sechs Zeilen (gemessen; Zeilennummern variieren mit dem Kommentar, der `sed` schneidet das Präfix `privacy_test.go:<zeile>: ` ab; die Schleife über `.`, `./.`, `.//.` bleibt still, weil ubs Fassung diese drei schon verweigert):
```
expected "[layout] review must not resolve to the area root, found './'", got [layout] review must not resolve to the area root, found "./"
LayoutReview "/": got [], err [layout] review must not resolve to the area root, found "/"; want ["//**" "/"]
LayoutReview "\\": got [], err [layout] review must not resolve to the area root, found "\\"; want ["\\/**" "\\"]
LayoutReview "///": got [], err [layout] review must not resolve to the area root, found "///"; want ["////**" "///"]
LayoutReview " review ": got ["review/**" "review"], err <nil>; want [" review /**" " review "]
LayoutReview "a/..": got [], err [layout] review must not resolve to the area root, found "a/.."; want ["a/../**" "a/.."]
```

- [ ] **Step 15: `ReviewExcludes` implementieren**

`internal/brain/privacy/readable.go` hat danach genau diesen Inhalt (`IsReadable` wortgleich aus ub; `ReviewExcludes` vorher mit `strings.TrimSpace`, `path.Clean` und `%q`):
```go
package privacy

import (
	"fmt"
	"regexp"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// MatchesGlobs reports whether relative matches any of the patterns. It is
// `matches_globs` (src/brain/privacy.py:31-74): `PurePosixPath(folded(relative))
// .full_match(folded(pattern))`, folded being NFC and then casefold, on both
// sides and on every platform. The docstring there gives the reason for both
// folds: a spelling variant must not walk around a `never` pattern wherever
// the filesystem opens the file anyway.
//
// No backslash becomes a slash here, because Python turns none: to
// PurePosixPath `secrets\key.txt` is one name. The callers hand in register
// paths and the output of Contained, both spelt with slashes.
//
// Each pattern is compiled on every call. Python caches the compiled form;
// the lists are a handful of globs, and a cache would be state this package
// keeps nowhere else.
func MatchesGlobs(patterns []string, relative string) bool {
	candidate := fullMatchForm(folded(relative))
	for _, pattern := range patterns {
		if regexp.MustCompile(translateGlob(fullMatchForm(folded(pattern)))).MatchString(candidate) {
			return true
		}
	}
	return false
}

// folded is `_folded` (src/brain/privacy.py:73-74).
func folded(text string) string {
	return pytext.CaseFold(pytext.NFC(text))
}

// IsReadable checks whether relative path is readable according to manifest never globs.
func IsReadable(manifest *config.Manifest, relative string) bool {
	if manifest == nil || len(manifest.NeverGlobs) == 0 {
		return true
	}
	return !MatchesGlobs(manifest.NeverGlobs, relative)
}

// ReviewExcludes returns the two patterns that keep the review centre from the
// cloud channel. It is `review_excludes` (src/brain/walk.py:96-113): the value
// is used as written, untrimmed, and refused only when
// `PurePosixPath(review).parts` is empty -- `.`, `./`, `./.` -- because such a
// value would turn the first pattern into a bare `**` and take the whole area
// out. `/` and `\` have parts and pass, as they do in Python.
//
// Python raises ValueError there, which the command line does not catch, so
// the reference ends in a traceback; loomux returns the same words as an error.
func ReviewExcludes(manifest *config.Manifest) ([]string, error) {
	if manifest == nil || manifest.LayoutReview == "" {
		return nil, nil
	}
	review := manifest.LayoutReview
	if root, parts := posixParts(review); root == "" && len(parts) == 0 {
		return nil, fmt.Errorf("[layout] review must not resolve to the area root, found %s", pytext.Repr(review))
	}
	return []string{review + "/**", review}, nil
}
```
```bash
go test -count=1 ./internal/brain/privacy/
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/privacy`.

- [ ] **Step 16: `VisibleAreas`, `Single`, `UnknownScope` test-first**

`internal/brain/privacy/areas_test.go` anlegen. Es nutzt `writeFile` aus `privacy_test.go`, das mit dem Umzug kam und unverändert bleibt (ub `privacy_test.go:113-121`; beide Dateien sind `package privacy_test`):
```go
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
```
Die neue Datei:
```go
package privacy_test

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)

// registered is one [[area]] of a test registry. manifestName "" writes no
// manifest; otherwise the declaration goes where config.ManifestDir looks for
// it -- into the area for a writable area, below <legacy>/areas/<flat> for a
// read-only one.
type registered struct {
	scope, mode, manifestName string
	readOnly                  bool
}

// buildWorld writes a registry into <root>/state and the manifests of its
// areas, and answers the registry and legacy directories.
func buildWorld(t *testing.T, areas ...registered) (registryDir, legacyDir string) {
	t.Helper()
	root := t.TempDir()
	registryDir = filepath.Join(root, "state")
	legacyDir = filepath.Join(root, "legacy")
	var registry strings.Builder
	for i, a := range areas {
		path := filepath.ToSlash(filepath.Join(root, fmt.Sprintf("repo-%d", i)))
		fmt.Fprintf(&registry, "[[area]]\nscope = %q\npath = %q\nreadonly = %v\n\n", a.scope, path, a.readOnly)
		if a.manifestName == "" {
			continue
		}
		dir := config.ManifestDir(config.Area{Scope: a.scope, Path: path, ReadOnly: a.readOnly}, legacyDir)
		writeFile(t, filepath.Join(dir, a.manifestName), fmt.Sprintf("[area]\nscope = %q\n\n[privacy]\nmode = %q\n", a.scope, a.mode))
	}
	writeFile(t, filepath.Join(registryDir, "registry.toml"), registry.String())
	return registryDir, legacyDir
}

func scopesOf(areas []privacy.VisibleArea) string {
	scopes := make([]string, 0, len(areas))
	for _, entry := range areas {
		scopes = append(scopes, entry.Area.Scope)
	}
	return strings.Join(scopes, ",")
}

// twoAreas lists zeta before project/alpha, the reverse of sorted order, so a
// test can tell registry order from sorting. project/alpha is read-only and
// local_only, so its manifest can only be found through the legacy directory.
func twoAreas(t *testing.T) (registryDir, legacyDir string) {
	return buildWorld(t,
		registered{scope: "zeta", mode: "manual_cloud", manifestName: ".ultra-brain/config.toml"},
		registered{scope: "project/alpha", mode: "local_only", manifestName: ".brain.toml", readOnly: true},
	)
}

func TestVisibleAreasKeepsRegistryOrderAndHidesLocalOnlyOnCloud(t *testing.T) {
	registryDir, legacyDir := twoAreas(t)
	local, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	if err != nil {
		t.Fatal(err)
	}
	if got := scopesOf(local); got != "zeta,project/alpha" {
		t.Fatalf("local: %s", got)
	}
	if m := local[1].Manifest; m == nil || m.PrivacyMode != "local_only" || !local[1].Area.ReadOnly {
		t.Fatalf("project/alpha: %+v", local[1])
	}
	cloud, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelCloud)
	if err != nil {
		t.Fatal(err)
	}
	if got := scopesOf(cloud); got != "zeta" {
		t.Fatalf("cloud: %s", got)
	}
}

func TestVisibleAreasNamesOneScope(t *testing.T) {
	registryDir, legacyDir := twoAreas(t)
	got, err := privacy.VisibleAreas(registryDir, legacyDir, "project/alpha", privacy.ChannelLocal)
	if err != nil {
		t.Fatal(err)
	}
	if scopesOf(got) != "project/alpha" {
		t.Fatalf("got %s", scopesOf(got))
	}
}

// One message for an unknown and a hidden scope, naming only what the channel
// sees: telling a cloud caller that the area exists would disclose what
// `local_only` hides (`_unknown_scope`, src/brain/core.py:266-275).
func TestVisibleAreasRefusesAnUnknownScopeNamingTheVisibleOnes(t *testing.T) {
	registryDir, legacyDir := twoAreas(t)
	for _, tc := range []struct {
		scope string
		ch    privacy.Channel
		want  string
	}{
		{"zz", privacy.ChannelLocal, "unknown scope 'zz'; known scopes are: project/alpha, zeta"},
		{"project/alpha", privacy.ChannelCloud, "unknown scope 'project/alpha'; known scopes are: zeta"},
	} {
		got, err := privacy.VisibleAreas(registryDir, legacyDir, tc.scope, tc.ch)
		if err == nil || err.Error() != tc.want || got != nil {
			t.Errorf("%s on %s: got %v, %v; want %q", tc.scope, tc.ch, got, err, tc.want)
		}
	}
}

// Python reads the manifest of every registered area before it filters by
// scope, so an area without a declaration fails a call about another area.
func TestVisibleAreasStopsAtTheFirstUnusableManifest(t *testing.T) {
	registryDir, legacyDir := buildWorld(t,
		registered{scope: "a", mode: "manual_cloud", manifestName: ".brain.toml"},
		registered{scope: "b"},
	)
	got, err := privacy.VisibleAreas(registryDir, legacyDir, "a", privacy.ChannelLocal)
	if !errors.Is(err, config.ErrNoManifest) || got != nil {
		t.Fatalf("got %v, %v; want config.ErrNoManifest", got, err)
	}
}

func TestVisibleAreasPassesTheRegistryErrorOn(t *testing.T) {
	got, err := privacy.VisibleAreas(t.TempDir(), t.TempDir(), "all", privacy.ChannelLocal)
	if !errors.Is(err, fs.ErrNotExist) || got != nil {
		t.Fatalf("got %v, %v; want a missing registry", got, err)
	}
}

func TestVisibleAreasOfAnEmptyRegistry(t *testing.T) {
	registryDir, legacyDir := buildWorld(t)
	all, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	if err != nil || len(all) != 0 {
		t.Fatalf("got %v, %v", all, err)
	}
	_, err = privacy.VisibleAreas(registryDir, legacyDir, "x", privacy.ChannelLocal)
	if want := "unknown scope 'x'; known scopes are: "; err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestSingle(t *testing.T) {
	areas := []privacy.VisibleArea{
		{Area: config.Area{Scope: "b", Path: "B"}},
		{Area: config.Area{Scope: "a", Path: "A"}},
	}
	got, err := privacy.Single(areas, "a")
	if err != nil || got.Area.Path != "A" {
		t.Fatalf("got %+v, %v", got, err)
	}
	got, err = privacy.Single(areas, "c")
	if want := "unknown scope 'c'; known scopes are: a, b"; err == nil || err.Error() != want || got.Area.Scope != "" {
		t.Fatalf("got %+v, %v; want %q", got, err, want)
	}
}

// Measured against `_unknown_scope` on 2026-09-15 under Python 3.14.7.
func TestUnknownScopeQuotesLikePython(t *testing.T) {
	for _, tc := range []struct {
		scope string
		areas []privacy.VisibleArea
		want  string
	}{
		{"zz", []privacy.VisibleArea{{Area: config.Area{Scope: "zeta"}}, {Area: config.Area{Scope: "alpha"}}}, "unknown scope 'zz'; known scopes are: alpha, zeta"},
		{"it's", nil, `unknown scope "it's"; known scopes are: `},
		{`say "hi"`, []privacy.VisibleArea{{Area: config.Area{Scope: "b"}}}, `unknown scope 'say "hi"'; known scopes are: b`},
	} {
		if got := privacy.UnknownScope(tc.scope, tc.areas).Error(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}
```
Messung (`brain.core._unknown_scope`): `("zz", [zeta, alpha])` → `unknown scope 'zz'; known scopes are: alpha, zeta`; `("it's", [])` → `unknown scope "it's"; known scopes are: ` (Leerzeichen am Ende); `('say "hi"', [b])` → `unknown scope 'say "hi"'; known scopes are: b`.

- [ ] **Step 17: Scheitern sehen**

```bash
go test -count=1 ./internal/brain/privacy/ > "$TEMP/task03-step17.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
sed -n -E 's/^.*areas_test\.go:[0-9]+:[0-9]+: //p' "$TEMP/task03-step17.txt"
```
Erwartet, genau diese elf Zeilen (gemessen; Zeilennummern variieren mit dem Kommentar, der `sed` schneidet Pfad, Zeile und Spalte ab). Der Compiler bricht nach zehn Fehlern ab, darum erscheinen `privacy.Single` und `privacy.UnknownScope` nicht:
```
undefined: privacy.VisibleArea
undefined: privacy.VisibleAreas
undefined: privacy.VisibleAreas
undefined: privacy.VisibleAreas
undefined: privacy.VisibleAreas
undefined: privacy.VisibleAreas
undefined: privacy.VisibleAreas
undefined: privacy.VisibleAreas
undefined: privacy.VisibleAreas
undefined: privacy.VisibleArea
too many errors
```
```bash
grep -E '^FAIL	github\.com/xidus90/loomux/internal/brain/privacy \[build failed\]$' "$TEMP/task03-step17.txt"
```
Erwartet: genau die Zeile `FAIL	github.com/xidus90/loomux/internal/brain/privacy [build failed]`, Exit 0.

- [ ] **Step 18: `VisibleAreas`, `Single`, `UnknownScope` implementieren**

`internal/brain/privacy/areas.go`:
```go
package privacy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// VisibleArea is one registered area a channel may see, together with the
// declaration that decided it.
type VisibleArea struct {
	Area     config.Area
	Manifest *config.Manifest
}

// VisibleAreas answers the areas the caller may see, resolved before anything
// is asked. It is `_visible_areas` (src/brain/core.py:248-263), whose reason
// holds here too: "filtering afterwards would mean the invisible area was
// queried -- and a query is already a disclosure of the question".
//
// The registry comes from registryDir. The manifest of each area comes from
// config.ManifestDir(area, legacyDir): a read-only area keeps it in
// ultra-brain's state directory until stage 3. Python reads the manifest of
// every registered area before it looks at scope, so the first registry or
// manifest error ends the call, whichever area it belongs to.
//
// scope "all" answers every visible area in registry order. Any other scope
// answers the visible areas of that name, or the UnknownScope error when there
// are none.
func VisibleAreas(registryDir, legacyDir, scope string, ch Channel) ([]VisibleArea, error) {
	areas, err := config.ReadRegistry(registryDir)
	if err != nil {
		return nil, err
	}
	var visible []VisibleArea
	for _, area := range areas {
		manifest, seen, err := VisibleManifest(config.ManifestDir(area, legacyDir), ch)
		if err != nil {
			return nil, err
		}
		if seen {
			visible = append(visible, VisibleArea{Area: area, Manifest: manifest})
		}
	}
	if scope == "all" {
		return visible, nil
	}
	var named []VisibleArea
	for _, entry := range visible {
		if entry.Area.Scope == scope {
			named = append(named, entry)
		}
	}
	if len(named) == 0 {
		return nil, UnknownScope(scope, visible)
	}
	return named, nil
}

// Single is `_single` (src/brain/core.py:535-539): the first visible area of
// that scope, or the UnknownScope error.
func Single(areas []VisibleArea, scope string) (VisibleArea, error) {
	for _, entry := range areas {
		if entry.Area.Scope == scope {
			return entry, nil
		}
	}
	return VisibleArea{}, UnknownScope(scope, areas)
}

// UnknownScope is `_unknown_scope` (src/brain/core.py:266-275), "the one
// message every tool gives for a scope it cannot serve". It names the visible
// scopes sorted, and only those: one message for an unknown and a hidden scope
// is the point, since telling a cloud caller that an area exists would
// disclose what `local_only` hides. Go sorts strings by bytes, Python by code
// points; for UTF-8 the two orders agree.
func UnknownScope(scope string, areas []VisibleArea) error {
	known := make([]string, 0, len(areas))
	for _, entry := range areas {
		known = append(known, entry.Area.Scope)
	}
	sort.Strings(known)
	return fmt.Errorf("unknown scope %s; known scopes are: %s", pytext.Repr(scope), strings.Join(known, ", "))
}
```
```bash
go test -count=1 ./internal/brain/privacy/
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/privacy`.

- [ ] **Step 19: Coverage (Verfahren Schritt 6)**

```bash
go test ./internal/brain/privacy/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Erwartet: Exit 0.
```bash
go tool cover -func="$TEMP/pkg.out" | grep -E '^github\.com/xidus90/loomux/internal/brain/privacy/'
```
Erwartet: `go tool cover -func="$TEMP/pkg.out"` — jede Funktion des Pakets 100.0%. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund. Gemessen in der Modulkopie sind es genau diese 19 Funktionen, jede mit `100.0%`, in dieser Reihenfolge: `VisibleAreas`, `Single`, `UnknownScope`, `ParseChannel`, `VisibleManifest`, `IsVisible`, `Contained`, `hasDrive`, `posixParts`, `fullMatchForm`, `translateGlob`, `translateSegment`, `bracketBody`, `indexHyphen`, `classBody`, `MatchesGlobs`, `folded`, `IsReadable`, `ReviewExcludes`. Zeilennummern variieren mit dem Kommentar.
```bash
go tool cover -func="$TEMP/pkg.out" | grep -E '^github\.com/xidus90/loomux/internal/brain/privacy/' | grep -v -E '100\.0%$'
```
Erwartet: keine Ausgabe, Exit 1.

- [ ] **Step 20: Stagen und Tor**

```bash
gofmt -l internal/brain/privacy
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/privacy/...
```
Erwartet: keine Ausgabe.
```bash
git add internal/brain/privacy/areas.go internal/brain/privacy/areas_test.go internal/brain/privacy/channel.go internal/brain/privacy/containment.go internal/brain/privacy/glob.go internal/brain/privacy/glob_test.go internal/brain/privacy/privacy_test.go internal/brain/privacy/readable.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0 (gofmt, vet, Tests mit Coverage, `covergate` grün, Pilot-Binary neu gebaut).

- [ ] **Step 21: Commit**

```bash
git branch --show-current
```
Erwartet: `sdd-1b-1` (Hauptcheckout: `master`).
```bash
git rev-parse --short HEAD
```
Erwartet: der Kurz-Hash des Commits von Task 2.
```bash
git diff --cached --stat
```
Erwartet: genau die acht Dateien `internal/brain/privacy/areas.go`, `areas_test.go`, `channel.go`, `containment.go`, `glob.go`, `glob_test.go`, `privacy_test.go`, `readable.go` und eine Summenzeile, die mit ` 8 files changed` beginnt.
```bash
printf '%s\n' 'Move privacy and answer visibility, containment and globs like Python' '' 'The package moves from ultra-brain pkg/privacy. VisibleManifest reads' 'the legacy manifest names until stage 4 and returns every failure, the' 'absent declaration included. VisibleAreas, Single and UnknownScope follow' 'core._visible_areas, _single and _unknown_scope. Contained, MatchesGlobs' 'and ReviewExcludes follow core._contained, privacy.matches_globs' '(glob.translate and fnmatch._translate of Python 3.14 over NFC and' 'casefold) and walk.review_excludes.' > "$TEMP/loomux-task03-commit.txt"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
git commit -F "$TEMP/loomux-task03-commit.txt"
```
Erwartet: Exit 0. Der Commit löst `.githooks/pre-commit` noch einmal aus (gewollt) und `.githooks/commit-msg`, das `bin/loomux.exe check commit-msg` ruft (ohne Binary `go run ./cmd/loomux check commit-msg`); beide lassen die englische Nachricht durch. Nach der Ausgabe der Hooks bestätigt Git mit genau einer Zeile, die mit `[sdd-1b-1 ` und dem Kurz-Hash beginnt und mit `] Move privacy and answer visibility, containment and globs like Python` endet.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität, gleich `git config user.name` und `git config user.email`.
```bash
git log -1 --format=%B
```
Erwartet: genau die neun Zeilen der Nachrichtendatei (Betreff, Leerzeile, sieben Zeilen Rumpf), keine `Co-Authored-By`-Zeile.

- [ ] **Step 22: Bericht**

Paritätszeilen, die dieser Task schafft (Task 12 trägt sie ein, ohne Freigabe):
1. Fehlendes Manifest eines registrierten Bereichs: Python `error: [Errno 2] No such file or directory: '<basis>\\.brain.toml'` (Pfad in `repr`, Backslashes verdoppelt); loomux `error: <verzeichnis>: no manifest found (.loomux\config.toml, .ultra-brain\config.toml, .brain.toml)` mit dem Wortlaut aus Task 1 (Namen über `filepath.Join`, auf Windows mit Backslash). Beide Exit 1, beide brechen den ganzen Aufruf ab.
2. `[layout] review` ohne Pfadteile (`.`, `./`, `./.`): Python wirft `ValueError`, den `cli.main` nicht fängt — Traceback, Exit 1; loomux `error: [layout] review must not resolve to the area root, found '.'`, Exit 1.
3. Fehlende Registry: Python `error: [Errno 2] No such file or directory: '<zustandsverzeichnis>\\registry.toml'` (Pfad in `repr`, Backslashes verdoppelt); loomux `error: <zustandsverzeichnis>\registry.toml: open <zustandsverzeichnis>\registry.toml: <Systemtext>` (gemessen auf dieser Maschine: `Das System kann die angegebene Datei nicht finden.`). Beide Exit 1.
4. Registry-Prüfungen, sichtbar über `VisibleAreas`: Python verweigert die ganze Registry bei doppeltem Scope, fehlendem `scope`/`path`, einem Scope ohne brauchbare Zeichen, zwei Scopes mit gleichem Zustandsverzeichnis und zwei `signpost` (`RegistryError`, Exit 1); loomux' `config.ReadRegistry` überspringt Einträge ohne `scope`/`path` und prüft den Rest nicht — bei doppeltem Scope liefert `VisibleAreas` beide, `Single` den ersten.

Befunde über die Umzugsregel hinaus:
- Umzugsverfahren Schritt 4 war nicht grün (Step 3, fünf Untertests von `TestVisibleManifest`); der Plankopf lässt genau dieses Rot für Task 3 zu. Der Bericht nennt es trotzdem, damit ein anderes Rot an dieser Stelle auffällt.
- Entfallen: `winDrive` (eine `regexp.MustCompile`-Paketvariable weniger) und `globToRegex`.
- Kommentare an `VisibleManifest` und `IsVisible` neu, weil das Verhalten sich geändert hat.
- `TestVisibleManifest`: Feld `wantErr`, Kopfkommentar neu, zwei Fälle neu; „no manifest" ist jetzt ein Fehler.
- `TestContained`: `.`, `""` wandern zu den angenommenen Pfaden; der Wortlaut wird exakt statt mit `strings.Contains` geprüft, und nach einem ausgebliebenen Fehler folgt `continue` (die ub-Fassung rief `err.Error()` auf `nil`).
- `TestReviewExcludes`: `/`, `\`, `///` sind angenommen statt verweigert.
- Der Glob-Übersetzer weicht in drei Schreibweisen von Pythons Regex ab (leere Klasse statt `(?!)`, `[` in Klassen maskiert, `regexp.QuoteMeta` statt `re.escape`); gegen 50 000 zufällige Muster/Pfad-Paare der Referenz stimmt jede Antwort. Die Messung lief beim Planen am 2026-09-15 in einer Modulkopie außerhalb des Repos und ist kein Schritt dieses Tasks; so lässt sie sich wiederholen. Der Erzeuger `gen_fuzz.py` (fester Seed, Antworten von `brain.privacy.matches_globs`):
  ```python
  import json
  import random
  import sys

  from brain.privacy import matches_globs

  TOKENS = list("abcAZ*?[]!-^/\\.&~|:") + ["**", "é", "é", "ß", "SS", "ss", "Σ", "ς", "ä", "ö"]
  rng = random.Random(20260915)
  cases, errors = [], 0
  while len(cases) < 50000:
      pattern = "".join(rng.choice(TOKENS) for _ in range(rng.randint(0, 8)))
      relative = "".join(rng.choice(TOKENS) for _ in range(rng.randint(0, 8)))
      try:
          cases.append([pattern, relative, matches_globs((pattern,), relative)])
      except Exception:
          errors += 1
  json.dump(cases, open(sys.argv[1], "w", encoding="utf-8"))
  print(len(cases), sum(1 for c in cases if c[2]), errors)
  ```
  Aufruf: `PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python gen_fuzz.py fuzz50k.json` — Ausgabe `50000 1176 0` (50 000 Paare, 1 176 davon Treffer, keine Ausnahme in Python). Der Prüfer ist ein Test in einem Wegwerfpaket derselben Modulkopie; er liest die Datei aus `FUZZ_JSON`:
  ```go
  func TestAgainstPython50k(t *testing.T) {
  	data, err := os.ReadFile(os.Getenv("FUZZ_JSON"))
  	if err != nil {
  		t.Fatal(err)
  	}
  	var cases [][3]any
  	if err := json.Unmarshal(data, &cases); err != nil {
  		t.Fatal(err)
  	}
  	bad := 0
  	for _, c := range cases {
  		p, r, want := c[0].(string), c[1].(string), c[2].(bool)
  		if got := privacy.MatchesGlobs([]string{p}, r); got != want {
  			bad++
  			if bad < 40 {
  				t.Errorf("%q ~ %q: got %v want %v", p, r, got, want)
  			}
  		}
  	}
  	t.Logf("%d cases, %d differ", len(cases), bad)
  }
  ```
  Ergebnis mit `go test -count=1 -v -run TestAgainstPython50k`: `50000 cases, 0 differ`, `PASS`.

---

### Task 4: `internal/brain/identity` — Umzug nach Python

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 3 sind committet.

`$SRC` steht für `/c/Users/micro/Documents/#GIT/loomux-src/ub` (wie im Umzugsverfahren); Go-Befehle laufen im Arbeitsort `$LOOMUX`. `$TEMP` ist das Temp-Verzeichnis der Git-Bash.

**Files:**
- Move: `$SRC/pkg/index/identity.go` → `internal/brain/identity/identity.go`
- Move: `$SRC/pkg/index/identity_test.go` → `internal/brain/identity/identity_test.go` (in-package-Test, `package index` → `package identity`)
- Create: `internal/brain/identity/read_python_test.go`
- Modify: `internal/brain/identity/identity.go` (Importblock, `ReadIdentities`, neu `parseRevision`)

Nur diese zwei Dateien ziehen um; `catalog.go`, `document.go`, `qmd_config.go`, `reindex.go`, `walk.go` und ihre Tests bleiben in ub.

**Interfaces:**
- Consumes: Task 2 `pytext.ReadText(path string) (string, error)`, `pytext.SplitLines(s string) []string`, `pytext.Strip(s string) string`, `pytext.Repr(s string) string`.
- Produces (Vertrag, wörtlich):
  ```go
  package identity
  const IdentitiesHeader = "doc_id\tpfad\tcontent_hash\trevision"
  type Identity struct { DocID, Relative, ContentHash string; Revision int }
  func ReadIdentities(path string) (map[string]Identity, error) // fehlt die Datei → leere Map, kein Fehler
  ```
- Produces (Ergänzung — ziehen wortgleich mit, weil sie in `identity.go` stehen; kein 1b-1-Aufrufer):
  ```go
  const CrockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
  func NewDocID() string
  func ContentHash(path string) (string, error)
  func MatchRenames(previous map[string]Identity, current map[string]string) map[string]Identity
  func RenderIdentities(identities map[string]Identity) string
  // unexportiert, wortgleich: encodeBase32, encodeBigIntBase32
  // unexportiert, neu: parseRevision(s string) (int, bool)
  ```

`ContentHash` bleibt unangetastet: Python hasht `read_bytes()` mit `\r\n` → `\n` (`identity.py:32`), genau wie ub.

**Vorab gemessen** (Python 3.14.7 der Referenz, `PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c` mit dem Ausdruck der linken Spalte):

| Ausdruck | Ergebnis |
|---|---|
| `os.path.exists('C:/x<y')`, `os.path.exists('a\x00b')`, `os.path.exists('C:/Windows/notepad.exe/sub')` | `False`, `False`, `False` — `Path.exists()` ist `os.path.exists`, jede Stat-Störung heißt „fehlt" |
| `'h\nA\ta.md\tsha256:1\t1\x1cbroken'.splitlines()` | `['h', 'A\ta.md\tsha256:1\t1', 'broken']` → Zeile 3, 1 Feld |
| `'\x1f\u3000'.strip()` | `''` (Zeile wird übersprungen) |
| `''.splitlines()[1:]`, `'h\n'.splitlines()[1:]` | `[]`, `[]` |
| `len('A\ta.md\tsha256:1'.split('\t'))`, `len('A\ta.md\tsha256:1\t1\tx'.split('\t'))` | `3`, `5` |
| `repr(r)` für `notanumber`, `+5`, `-1`, `''`, `it's`, `\u0663`, `\u00b2`, `99999999999999999999` | `'notanumber'`, `'+5'`, `'-1'`, `''`, `"it's"`, `'٣'`, `'²'`, `'99999999999999999999'` |
| `r.isdigit()` für dieselben | `False` ×5, dann `True`, `True`, `True` |
| `int('\u0663')`, `int('\u00b2')`, `int('99999999999999999999')` | `3`, `ValueError` (Traceback), `99999999999999999999` |
| `b'h\nA\ta.md\tsha256:\xff\t1\n'.decode('utf-8')` | `UnicodeDecodeError: 'utf-8' codec can't decode byte 0xff in position 16: invalid start byte` (eine `ValueError`, nicht in `main`s Fangliste → Traceback) |

**Abweichung vom Vertrag:** Der Vertrag verlangt „anderer Stat-Fehler → Fehler". Python fragt `path.exists()`, und das antwortet für **jede** Stat-Störung `False` (gemessen oben) → `{}`. Dieser Task folgt der Quelle: jeder Fehler aus `os.Stat` ergibt die leere Map. Ein vorhandener, aber unlesbarer Pfad (Verzeichnis) scheitert weiter beim Lesen.

- [ ] **Step 1: Umzug, Verfahren Schritt 1 — Dateien kopieren**

```bash
mkdir -p "$LOOMUX/internal/brain/identity"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
cp "$SRC/pkg/index/identity.go" "$SRC/pkg/index/identity_test.go" "$LOOMUX/internal/brain/identity/"
```
Erwartet: keine Ausgabe, Exit 0.

- [ ] **Step 2: Umzug, Verfahren Schritt 2 — Paketname**

```bash
sed -i 's/^package index$/package identity/; s/^package index_test$/package identity_test/' "$LOOMUX/internal/brain/identity"/*.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -h '^package ' "$LOOMUX/internal/brain/identity"/*.go
```
Erwartet: genau zwei Zeilen `package identity` (`identity_test.go` ist ein Test im Paket).

- [ ] **Step 3: Umzug, Verfahren Schritt 3 — Importpfade (nichts zu tun, prüfen)**

`identity.go` importiert nur die Standardbibliothek, und nichts in loomux importiert `pkg/index`:
```bash
grep -rl 'github.com/xidus90/ultra-brain/pkg/index' "$LOOMUX" --include=*.go
```
Erwartet: keine Ausgabe, Exit 1.

- [ ] **Step 4: Umzug, Verfahren Schritt 4 — formatieren, prüfen, grün** (ohne `go mod tidy`)

```bash
gofmt -w internal/brain/identity
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/identity/...
```
Erwartet: keine Ausgabe.
```bash
go test -count=1 -cover ./internal/brain/identity/...
```
Erwartet: Exit 0 und genau eine Zeile: `ok  	github.com/xidus90/loomux/internal/brain/identity`, die Laufzeit, dann `coverage: 100.0% of statements` (in einer Modulkopie gemessen; jeder Block des ub-Stands läuft).

- [ ] **Step 5: Failing tests nach Python** — `internal/brain/identity/read_python_test.go`

```go
package identity

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeRegister writes one register file into a fresh directory and returns its path.
func writeRegister(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "_identities.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadIdentitiesSplitsLinesLikePython(t *testing.T) {
	// Python: read_text folds \r to \n, and splitlines also breaks at \x1c.
	path := writeRegister(t, "h\nA\ta.md\tsha256:1\t1\rB\tb.md\tsha256:2\t2\x1cC\tc.md\tsha256:3\t3")
	got, err := ReadIdentities(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Identity{
		"a.md": {DocID: "A", Relative: "a.md", ContentHash: "sha256:1", Revision: 1},
		"b.md": {DocID: "B", Relative: "b.md", ContentHash: "sha256:2", Revision: 2},
		"c.md": {DocID: "C", Relative: "c.md", ContentHash: "sha256:3", Revision: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestReadIdentitiesCountsLinesLikePython(t *testing.T) {
	path := writeRegister(t, "h\nA\ta.md\tsha256:1\t1\x1cbroken")
	_, err := ReadIdentities(path)
	want := path + ": line 3: expected 4 tab-separated fields, found 1"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestReadIdentitiesSkipsLinesPythonStripsEmpty(t *testing.T) {
	// "\x1f\u3000".strip() is "" in Python; strings.TrimSpace keeps the \x1f.
	path := writeRegister(t, "h\n\x1f\u3000\nA\ta.md\tsha256:1\t1\n")
	got, err := ReadIdentities(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["a.md"].DocID != "A" {
		t.Fatalf("got %+v", got)
	}
}

func TestReadIdentitiesOfAnEmptyOrHeaderOnlyFileIsEmpty(t *testing.T) {
	for _, content := range []string{"", "doc_id\tpfad\tcontent_hash\trevision\n"} {
		got, err := ReadIdentities(writeRegister(t, content))
		if err != nil || len(got) != 0 {
			t.Fatalf("%q: got %+v, %v", content, got, err)
		}
	}
}

func TestReadIdentitiesNamesTheFieldCount(t *testing.T) {
	for _, tc := range []struct {
		row   string
		found string
	}{
		{"A\ta.md\tsha256:1", "3"},
		{"A\ta.md\tsha256:1\t1\tx", "5"},
	} {
		path := writeRegister(t, "h\n"+tc.row+"\n")
		_, err := ReadIdentities(path)
		want := path + ": line 2: expected 4 tab-separated fields, found " + tc.found
		if err == nil || err.Error() != want {
			t.Errorf("got %v, want %q", err, want)
		}
	}
}

func TestReadIdentitiesQuotesTheRevisionLikeRepr(t *testing.T) {
	for _, tc := range []struct {
		revision string
		quoted   string
	}{
		{"notanumber", "'notanumber'"},
		{"+5", "'+5'"},
		{"-1", "'-1'"},
		{"", "''"},
		{"it's", `"it's"`},
		{"\u0663", "'\u0663'"},
		{"\u00b2", "'\u00b2'"},
		{"99999999999999999999", "'99999999999999999999'"},
	} {
		path := writeRegister(t, "h\nA\ta.md\tsha256:1\t"+tc.revision+"\n")
		_, err := ReadIdentities(path)
		want := path + ": line 2: revision " + tc.quoted + " is not a number"
		if err == nil || err.Error() != want {
			t.Errorf("%q: got %v, want %q", tc.revision, err, want)
		}
	}
}

func TestReadIdentitiesKeepsTheLastRowOfAPath(t *testing.T) {
	path := writeRegister(t, "h\nA\ta.md\tsha256:1\t1\nB\ta.md\tsha256:2\t2\n")
	got, err := ReadIdentities(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["a.md"].DocID != "B" || got["a.md"].Revision != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestReadIdentitiesTreatsAnyStatFailureAsAbsent(t *testing.T) {
	// Path.exists() answers False for every OSError, a NUL byte included.
	got, err := ReadIdentities(filepath.Join(t.TempDir(), "a\x00b"))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestReadIdentitiesRefusesInvalidUTF8(t *testing.T) {
	path := writeRegister(t, "h\nA\ta.md\tsha256:\xff\t1\n")
	_, err := ReadIdentities(path)
	want := path + ": not valid UTF-8"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}
```

- [ ] **Step 6: Scheitern sehen**

```bash
go test -count=1 ./internal/brain/identity/ > "$TEMP/task04-step6.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
grep -E -o -- '--- FAIL: [^ ]+' "$TEMP/task04-step6.txt"
```
Erwartet, genau diese sechs Zeilen (gemessen gegen den unveränderten ub-Code):
```
--- FAIL: TestReadIdentitiesSplitsLinesLikePython
--- FAIL: TestReadIdentitiesCountsLinesLikePython
--- FAIL: TestReadIdentitiesSkipsLinesPythonStripsEmpty
--- FAIL: TestReadIdentitiesQuotesTheRevisionLikeRepr
--- FAIL: TestReadIdentitiesTreatsAnyStatFailureAsAbsent
--- FAIL: TestReadIdentitiesRefusesInvalidUTF8
```
Die Meldungen ohne Dateipräfix; die Temp-Pfade der Register ersetzt der zweite `sed` durch `REGISTER`, den Pfad mit dem NUL-Byte durch `STATPATH`:
```bash
sed -n -E 's/^[[:space:]]*read_python_test\.go:[0-9]+: //p' "$TEMP/task04-step6.txt" | sed -E 's/[A-Z]:[^ ]*_identities\.tsv/REGISTER/g; s/open [^ ]*: invalid argument$/open STATPATH: invalid argument/'
```
Erwartet, genau diese zwölf Zeilen (gemessen; Zeilennummern variieren mit dem Kommentar und stehen darum nicht darin):
```
REGISTER: line 2: expected 4 tab-separated fields, found 10
got REGISTER: line 2: revision "1\x1cbroken" is not a number, want "REGISTER: line 3: expected 4 tab-separated fields, found 1"
REGISTER: line 2: expected 4 tab-separated fields, found 1
"notanumber": got REGISTER: line 2: revision "notanumber" is not a number, want "REGISTER: line 2: revision 'notanumber' is not a number"
"+5": got <nil>, want "REGISTER: line 2: revision '+5' is not a number"
"-1": got REGISTER: line 2: revision "-1" is not a number, want "REGISTER: line 2: revision '-1' is not a number"
"": got REGISTER: line 2: revision "" is not a number, want "REGISTER: line 2: revision '' is not a number"
"٣": got REGISTER: line 2: revision "٣" is not a number, want "REGISTER: line 2: revision '٣' is not a number"
"²": got REGISTER: line 2: revision "²" is not a number, want "REGISTER: line 2: revision '²' is not a number"
"99999999999999999999": got REGISTER: line 2: revision "99999999999999999999" is not a number, want "REGISTER: line 2: revision '99999999999999999999' is not a number"
got map[], open STATPATH: invalid argument
got <nil>, want "REGISTER: not valid UTF-8"
```

`TestReadIdentitiesOfAnEmptyOrHeaderOnlyFileIsEmpty`, `TestReadIdentitiesNamesTheFieldCount` und `TestReadIdentitiesKeepsTheLastRowOfAPath` bestehen schon; sie halten das fest, was der Umbau nicht verlieren darf.

- [ ] **Step 7: Implementieren** — drei Stellen in `internal/brain/identity/identity.go`, der Rest bleibt wortgleich.

(a) Importblock. Vorher:
```go
import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)
```
Nachher (`errors` entfällt, es diente nur `ReadIdentities`):
```go
import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
)
```

(b) `ReadIdentities` samt Kommentar. Vorher (ub `identity.go:80-130`, wörtlich):
```go
// ReadIdentities loads the identity TSV file into memory.
// Returns an empty map if the file does not exist.
// Returns an error if any row is malformed or has an invalid revision.
func ReadIdentities(path string) (map[string]Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]Identity{}, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	identities := make(map[string]Identity)

	// Line numbers start at 2 to count the header line and match editor line displays
	for i, line := range lines {
		if i == 0 {
			continue
		}
		trimmed := strings.TrimRight(line, "\r")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}

		fields := strings.Split(trimmed, "\t")
		lineNum := i + 1
		if len(fields) != 4 {
			return nil, fmt.Errorf("%s: line %d: expected 4 tab-separated fields, found %d", path, lineNum, len(fields))
		}

		docID := fields[0]
		relative := fields[1]
		digest := fields[2]
		revisionStr := fields[3]

		rev, err := strconv.Atoi(revisionStr)
		if err != nil || rev < 0 {
			return nil, fmt.Errorf("%s: line %d: revision %q is not a number", path, lineNum, revisionStr)
		}

		identities[relative] = Identity{
			DocID:       docID,
			Relative:    relative,
			ContentHash: digest,
			Revision:    rev,
		}
	}

	return identities, nil
}
```
Nachher, ganze Funktion:
```go
// ReadIdentities loads the identity TSV file into memory, the way
// identity.read_identities does (src/brain/identity.py:55-77).
// Returns an empty map if the file does not exist; like Path.exists(), any
// failure to stat the path counts as "does not exist".
// Returns an error if the file is not UTF-8 or a row is malformed or has an
// invalid revision.
func ReadIdentities(path string) (map[string]Identity, error) {
	if _, err := os.Stat(path); err != nil {
		return map[string]Identity{}, nil
	}
	text, err := pytext.ReadText(path)
	if err != nil {
		return nil, err
	}

	lines := pytext.SplitLines(text)
	identities := make(map[string]Identity)

	// Line numbers start at 2 to count the header line and match editor line displays
	for i, line := range lines {
		if i == 0 {
			continue
		}
		if pytext.Strip(line) == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		lineNum := i + 1
		if len(fields) != 4 {
			return nil, fmt.Errorf("%s: line %d: expected 4 tab-separated fields, found %d", path, lineNum, len(fields))
		}

		docID := fields[0]
		relative := fields[1]
		digest := fields[2]
		revisionStr := fields[3]

		rev, ok := parseRevision(revisionStr)
		if !ok {
			return nil, fmt.Errorf("%s: line %d: revision %s is not a number", path, lineNum, pytext.Repr(revisionStr))
		}

		identities[relative] = Identity{
			DocID:       docID,
			Relative:    relative,
			ContentHash: digest,
			Revision:    rev,
		}
	}

	return identities, nil
}
```
Die Schleife läuft weiter über alle Zeilen und überspringt die Kopfzeile mit `if i == 0 { continue }`: `SplitLines("")` liefert eine leere Liste, ein `lines[1:]` würde dort in Panik geraten.

(c) Neu, direkt unter `ReadIdentities` (vor `MatchRenames`):
```go
// parseRevision accepts what Python's revision.isdigit() and int() accept,
// narrowed to ASCII digits and to the range of int: a sign, a space or an
// empty field is refused as Python refuses it; a non-ASCII digit and a value
// beyond int are refused where Python would read or crash (parity list).
func parseRevision(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	rev, err := strconv.Atoi(s)
	return rev, err == nil
}
```

- [ ] **Step 8: Bestehen sehen**

```bash
gofmt -l internal/brain/identity
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/identity/...
```
Erwartet: keine Ausgabe.
```bash
go test -count=1 ./internal/brain/identity/
```
Erwartet: Exit 0 und genau eine Zeile: `ok  	github.com/xidus90/loomux/internal/brain/identity` mit angehängter Laufzeit. Der umgezogene `TestReadIdentities` (Verzeichnis als Register → Fehler) bleibt grün: `os.Stat` gelingt, `pytext.ReadText` scheitert.

- [ ] **Step 9: Coverage (Verfahren Schritt 6)**

```bash
go test ./internal/brain/identity/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Erwartet: Exit 0.
```bash
go tool cover -func="$TEMP/pkg.out" | grep -E '^github\.com/xidus90/loomux/internal/brain/identity/'
```
Erwartet: `go tool cover -func="$TEMP/pkg.out"` — jede Funktion des Pakets 100.0%. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund. Gemessen in der Modulkopie sind es genau diese acht Funktionen, jede mit `100.0%`, in dieser Reihenfolge: `encodeBase32`, `encodeBigIntBase32`, `NewDocID`, `ContentHash`, `ReadIdentities`, `parseRevision`, `MatchRenames`, `RenderIdentities`. Zeilennummern variieren mit dem Kommentar. Der `strconv.Atoi`-Fehlerarm von `parseRevision` läuft über `99999999999999999999`, der leere Arm über `''`, der Nicht-Ziffern-Arm über `+5`.
```bash
go tool cover -func="$TEMP/pkg.out" | grep -E '^github\.com/xidus90/loomux/internal/brain/identity/' | grep -v -E '100\.0%$'
```
Erwartet: keine Ausgabe, Exit 1.

- [ ] **Step 10: Stagen und Tor**

```bash
git add internal/brain/identity/identity.go internal/brain/identity/identity_test.go internal/brain/identity/read_python_test.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0 (gofmt, vet, Tests mit Coverage, `covergate`, Pilot-Binary).

- [ ] **Step 11: Commit**

```bash
git branch --show-current
```
Erwartet: `sdd-1b-1` (Hauptcheckout: `master`).
```bash
git rev-parse --short HEAD
```
Erwartet: der Kurz-Hash des Commits von Task 3.
```bash
git diff --cached --stat
```
Erwartet: genau die drei Dateien `internal/brain/identity/identity.go`, `internal/brain/identity/identity_test.go`, `internal/brain/identity/read_python_test.go` und eine Summenzeile, die mit ` 3 files changed` beginnt.
```bash
printf '%s\n' 'Move the identity register reader and read it the way Python does' '' 'Lines split like str.splitlines, blank lines stripped like str.strip,' 'revisions quoted like repr and limited to ASCII digits, and any stat' 'failure counts as a missing register, as Path.exists() has it.' > "$TEMP/loomux-msg.txt"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
git commit -F "$TEMP/loomux-msg.txt"
```
Erwartet: Exit 0. Der Commit löst `.githooks/pre-commit` noch einmal aus (gewollt) und `.githooks/commit-msg`, das `bin/loomux.exe check commit-msg` ruft (ohne Binary `go run ./cmd/loomux check commit-msg`); beide lassen die englische Nachricht durch. Nach der Ausgabe der Hooks bestätigt Git mit genau einer Zeile, die mit `[sdd-1b-1 ` und dem Kurz-Hash beginnt und mit `] Move the identity register reader and read it the way Python does` endet.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität, gleich `git config user.name` und `git config user.email`.
```bash
git log -1 --format=%B
```
Erwartet: genau die fünf Zeilen der Nachrichtendatei (Betreff, Leerzeile, drei Zeilen Rumpf), keine `Co-Authored-By`-Zeile.

- [ ] **Step 12: Bericht**

Befunde gegen die Umzugsregel:
- Mitgezogen ohne 1b-1-Aufrufer: `CrockfordAlphabet`, `NewDocID`, `ContentHash`, `MatchRenames`, `RenderIdentities`, `encodeBase32`, `encodeBigIntBase32` — wortgleich, 100 % durch die umgezogenen Tests.
- `identity.go`: Import `errors` entfernt, Import `pytext` ergänzt; Kommentar über `ReadIdentities` neu (nennt Python-Herkunft und die Stat-Regel); neue Funktion `parseRevision`.
- Vertragsabweichung: jeder `os.Stat`-Fehler → leere Map (Vertrag: nur „fehlt"), weil `Path.exists()` jede Störung als „fehlt" liest (gemessen).
- Kein Test entfällt.

Paritätszeilen, die dieser Task schafft (Task 12 trägt sie ein):
1. Revision aus Nicht-ASCII-Ziffern: Python liest `٣` als 3 und bricht bei `²` mit Traceback (`ValueError`) ab; loomux verweigert beide mit `{pfad}: line {n}: revision '٣' is not a number` bzw. `{pfad}: line {n}: revision '²' is not a number`, Exit 1.
2. Revision jenseits des `int`-Bereichs (`99999999999999999999`): Python liest sie, loomux verweigert mit demselben Wortlaut.
3. Ungültiges UTF-8 in `_identities.tsv`: Python Traceback (`UnicodeDecodeError`), loomux `error: {pfad}: not valid UTF-8`, Exit 1.
4. Vorhandenes, aber unlesbares Register (etwa ein Verzeichnis): Python `error: [Errno 13] Permission denied: '{pfad}'`, loomux mit dem Go-Wortlaut `read {pfad}: ` und dem Systemtext, auf dieser Maschine `read {pfad}: Unzulässige Funktion.` (derselbe `os.ReadFile`-Fehler, den Task 5 für `graph.json` gemessen hat).

---

### Task 5: `internal/brain/graph` — Umzug, Wortlaute und Formprüfung nach Python

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 4 sind committet.

`$SRC` steht für `/c/Users/micro/Documents/#GIT/loomux-src/ub` (wie im Umzugsverfahren); Go-Befehle laufen im Arbeitsort `$LOOMUX`. `$TEMP` ist das Temp-Verzeichnis der Git-Bash.

**Files:**
- Move: `$SRC/pkg/graph/{model,neighbors,read}.go` → `internal/brain/graph/`
- Move: `$SRC/pkg/graph/{neighbors_test,read_test}.go` → `internal/brain/graph/`
- Create: `internal/brain/graph/read_python_test.go`
- Modify: `internal/brain/graph/read.go` (Literale, `ReadGraph`, `ParseGraph`, `isNumber`, neu `decodeKeepingNumbers`), `internal/brain/graph/read_test.go` (Literale), `internal/brain/graph/model.go` (Paketkommentar)

Nicht umgezogen: `render.go`, `synth.go`, `render_test.go`, `synth_test.go`. Weder `read_test.go` noch `neighbors_test.go` braucht etwas aus ihnen — kein Test entfällt.

**Interfaces:**
- Consumes: Task 2 `pytext.ReadText(path string) (string, error)` (Fehler bei ungültigem UTF-8: `{path}: not valid UTF-8`); `config.Area`, `config.ManifestDir(area Area, stateDir string) string` (`internal/config/manifest.go:439`).
- Produces (Vertrag, wörtlich):
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
- Produces (Ergänzung, unexportiert): `decodeKeepingNumbers(data []byte) (any, error)`; `checkShape` und `isNumber` bleiben unexportiert.

`Document` zieht mit, obwohl 1b-1 es nicht nutzt (nur `render.go` las es); der Vertrag nennt es.

`neighbors.go` bleibt wortgleich: Go `RenderNeighbors` liefert `fmt.Sprintf("incoming: %s\noutgoing: %s\n", inStr, outStr)` (ub `neighbors.go:42`), also **mit** Endzeilenumbruch; Python `render_neighbors` (`daemon/tools.py:27-35`) liefert denselben Text ohne Endzeilenumbruch (die zweite f-Zeile endet ohne `\n`), und `print(render_neighbors(incoming, outgoing))` hängt ihn an. Auf stdout stehen damit dieselben Bytes; `sorted` über `str` und `sort.Strings` ordnen gültiges UTF-8 gleich, Duplikate bleiben auf beiden Seiten (Fakten `moved-packages.md` §5.3).

**Vorab gemessen**

Python 3.14.7 der Referenz (`PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "$SRC" python -c` mit dem Ausdruck der linken Spalte), die Prüfungen aus `core._graph`/`_check_shape` (`core.py:584-632`) auf `json.loads`-Werten nachgestellt:

| Eingabe | Python |
|---|---|
| `os.path.exists('a\x00b')`, `os.path.exists('C:/x<y')` | `False`, `False` — `_graph` meldet dann `never indexed` |
| Wurzel `[]` | `missing ['edges', 'links']` |
| Wurzel `"x"` | `missing ['edges', 'links']` |
| Wurzel `"edges"` | `missing ['links']` (Teilzeichenkette) |
| Wurzel `["edges","links"]` | `missing []`, danach `TypeError` in `_check_shape` → Traceback |
| Wurzel `3`, `null` | `TypeError: argument of type 'int'/'NoneType' is not a container or iterable` → Traceback |
| Zählwert `3`, `-1`, `0`, `99999999999999999999` | `int`, `isinstance(v, int)` → `True` |
| Zählwert `3.0`, `1.5`, `1e2` | `float` → `False` → `the link counts are not numbers` |
| Zählwert `true` | `bool` → `True` (nimmt Python) |
| `json.loads('not json')` | `Expecting value: line 1 column 1 (char 0)` |
| `json.loads('{} x')` | `Extra data: line 1 column 4 (char 3)` |
| `json.loads('\ufeff{}')` | `Unexpected UTF-8 BOM (decode using utf-8-sig): line 1 column 1 (char 0)` |
| `json.loads('{"a": NaN}')` | `{'a': nan}` (angenommen) |
| `str(None)`, `str(1)` | `None`, `1` (`_edges` wandelt `from`/`to` so) |

Go 1.27 (`encoding/json`, `os`; Wegwerfmodul):

| Ausdruck | Go |
|---|---|
| `json.Unmarshal("not json", new(any))` | `invalid character 'o' in literal null (expecting 'u')` |
| `json.Unmarshal("", new(any))` | `unexpected end of JSON input` |
| `json.Unmarshal(valid+" x", new(any))` | `invalid character 'x' after top-level value` |
| `json.Unmarshal(valid+" {}", new(any))` | `invalid character '{' after top-level value` |
| `json.Unmarshal("\xef\xbb\xbf"+valid, new(any))` | `invalid character '\ufeff' looking for beginning of value` |
| ``json.Unmarshal([]byte(`{"edges": [], "links": {"total": NaN}}`), new(any))`` | `invalid character 'N' looking for beginning of value` |
| `json.Unmarshal("[]", &map[string]any{})` | `json: cannot unmarshal array into Go value of type map[string]interface {}` |
| `json.Unmarshal` von `{"from": 1}` in `Graph` | `json: cannot unmarshal number into Go struct field Graph.edges.0.from of type string` |
| `json.Unmarshal` von `{"from": null}` in `Graph` | kein Fehler, `From == ""` |
| `os.Stat(<tmp>\a\x00b)` | `Stat <tmp>\a\x00b: invalid argument` (mit dem rohen NUL-Byte), `errors.Is(err, fs.ErrNotExist)` → `false` |
| `os.ReadFile(<verzeichnis>)` | `read <verzeichnis>: Unzulässige Funktion.` |

- [ ] **Step 1: Umzug, Verfahren Schritt 1 — Dateien kopieren**

```bash
mkdir -p "$LOOMUX/internal/brain/graph"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
cp "$SRC/pkg/graph/model.go" "$SRC/pkg/graph/neighbors.go" "$SRC/pkg/graph/read.go" "$SRC/pkg/graph/neighbors_test.go" "$SRC/pkg/graph/read_test.go" "$LOOMUX/internal/brain/graph/"
```
Erwartet: keine Ausgabe, Exit 0.

- [ ] **Step 2: Umzug, Verfahren Schritt 2 — Paketname:** bleibt `graph` bzw. `graph_test`, nichts zu tun.

- [ ] **Step 3: Umzug, Verfahren Schritt 3 — Importpfade im ganzen Baum**

```bash
grep -rl 'github.com/xidus90/ultra-brain/pkg/config' "$LOOMUX" --include=*.go | xargs -r sed -i 's#github.com/xidus90/ultra-brain/pkg/config#github.com/xidus90/loomux/internal/config#g'
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rl 'github.com/xidus90/ultra-brain/pkg/graph' "$LOOMUX" --include=*.go | xargs -r sed -i 's#github.com/xidus90/ultra-brain/pkg/graph#github.com/xidus90/loomux/internal/brain/graph#g'
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rl 'github.com/xidus90/ultra-brain/' "$LOOMUX/internal/brain/graph" --include=*.go
```
Erwartet: keine Ausgabe, Exit 1.

Paketbezeichner bleiben (`graph.` und `config.`), der zweite Teil von Schritt 3 entfällt.

- [ ] **Step 4: Umzug, Verfahren Schritt 4 — formatieren, prüfen, grün** (ohne `go mod tidy`)

```bash
gofmt -w internal/brain/graph
```
Erwartet: keine Ausgabe. `gofmt` sortiert dabei den Importblock von `read_test.go` um (`github.com/xidus90/loomux/internal/brain/graph` vor `github.com/xidus90/loomux/internal/config`).
```bash
go vet ./internal/brain/graph/...
```
Erwartet: keine Ausgabe.
```bash
go test -count=1 -cover ./internal/brain/graph/...
```
Erwartet: Exit 0 und genau eine Zeile: `ok  	github.com/xidus90/loomux/internal/brain/graph`, die Laufzeit, dann `coverage: 100.0% of statements` (in der Modulkopie nach dem Umzug gemessen; jeder Block des ub-Stands läuft).

- [ ] **Step 5: Literale test-first — Tests auf Backticks**

Python schreibt jede Graph-Meldung mit `` `brain reindex` `` (`core.py:586`, `:591`, `:596`, `:636`). In `read_test.go` stehen die Apostrophe in 15 Zeilen (97, 109, 121–123, 147, 151, 155, 159, 165, 170, 175, 180, 184, 188), alle in doppelt gequoteten Go-Literalen:
```bash
sed -i "s/'brain reindex'/\`brain reindex\`/g" internal/brain/graph/read_test.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
go test -count=1 ./internal/brain/graph/ > "$TEMP/task05-step5.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
grep -E -o -- '--- FAIL: [^ ]+' "$TEMP/task05-step5.txt"
```
Erwartet, genau diese vier Zeilen (gemessen):
```
--- FAIL: TestReadGraph_NeverIndexed
--- FAIL: TestParseGraph_InvalidJSON
--- FAIL: TestParseGraph_MissingKeys
--- FAIL: TestParseGraph_InvalidShape
```
```bash
sed -n -E 's/^[[:space:]]*read_test\.go:[0-9]+: //p' "$TEMP/task05-step5.txt"
```
Erwartet, genau diese 15 Zeilen (gemessen; Zeilennummern variieren mit dem Kommentar und stehen darum nicht darin):
```
got "project/missing: never indexed; run 'brain reindex'", want "project/missing: never indexed; run `brain reindex`"
unexpected error message: "/path/to/graph.json: graph is not valid JSON (invalid character 'o' in literal null (expecting 'u')); delete it and run 'brain reindex'"
got "/path/to/graph.json: graph is missing edges, links; delete it and run 'brain reindex'", want "/path/to/graph.json: graph is missing edges, links; delete it and run `brain reindex`"
got "/path/to/graph.json: graph is missing edges; delete it and run 'brain reindex'", want "/path/to/graph.json: graph is missing edges; delete it and run `brain reindex`"
got "/path/to/graph.json: graph is missing links; delete it and run 'brain reindex'", want "/path/to/graph.json: graph is missing links; delete it and run `brain reindex`"
got "/path/to/graph.json: edges is not a list of from/to entries; delete it and run 'brain reindex'", want "/path/to/graph.json: edges is not a list of from/to entries; delete it and run `brain reindex`"
got "/path/to/graph.json: edges is not a list of from/to entries; delete it and run 'brain reindex'", want "/path/to/graph.json: edges is not a list of from/to entries; delete it and run `brain reindex`"
got "/path/to/graph.json: edges is not a list of from/to entries; delete it and run 'brain reindex'", want "/path/to/graph.json: edges is not a list of from/to entries; delete it and run `brain reindex`"
got "/path/to/graph.json: edges is not a list of from/to entries; delete it and run 'brain reindex'", want "/path/to/graph.json: edges is not a list of from/to entries; delete it and run `brain reindex`"
got "/path/to/graph.json: links is not a total/resolved/dropped record; delete it and run 'brain reindex'", want "/path/to/graph.json: links is not a total/resolved/dropped record; delete it and run `brain reindex`"
got "/path/to/graph.json: links is not a total/resolved/dropped record; delete it and run 'brain reindex'", want "/path/to/graph.json: links is not a total/resolved/dropped record; delete it and run `brain reindex`"
got "/path/to/graph.json: links.dropped is not a table of reasons; delete it and run 'brain reindex'", want "/path/to/graph.json: links.dropped is not a table of reasons; delete it and run `brain reindex`"
got "/path/to/graph.json: the link counts are not numbers; delete it and run 'brain reindex'", want "/path/to/graph.json: the link counts are not numbers; delete it and run `brain reindex`"
got "/path/to/graph.json: the link counts are not numbers; delete it and run 'brain reindex'", want "/path/to/graph.json: the link counts are not numbers; delete it and run `brain reindex`"
got "/path/to/graph.json: the link counts are not numbers; delete it and run 'brain reindex'", want "/path/to/graph.json: the link counts are not numbers; delete it and run `brain reindex`"
```

- [ ] **Step 6: Literale ändern** — `read.go` Zeile 20 (`ErrNotIndexed`) und die elf Meldungen in `ParseGraph` (42, 53) und `checkShape` (70, 75, 78, 81, 87, 95, 100, 104, 109):

```bash
sed -i "s/'brain reindex'/\`brain reindex\`/g" internal/brain/graph/read.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rn "'brain reindex'" internal/brain/graph
```
Erwartet: keine Ausgabe, Exit 1.
```bash
go test -count=1 ./internal/brain/graph/
```
Erwartet: Exit 0 und genau eine Zeile: `ok  	github.com/xidus90/loomux/internal/brain/graph` mit angehängter Laufzeit.

- [ ] **Step 7: Failing tests nach Python** — `internal/brain/graph/read_python_test.go`

```go
package graph_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/config"
)

func TestParseGraph_RootThatIsNotAnObjectMissesBothKeys(t *testing.T) {
	path := "/path/to/graph.json"
	want := path + ": graph is missing edges, links; delete it and run `brain reindex`"
	for _, data := range []string{`[]`, `"x"`, `3`, `null`, `"edges"`} {
		_, err := graph.ParseGraph([]byte(data), path)
		if err == nil || err.Error() != want {
			t.Errorf("%s: got %v, want %q", data, err, want)
		}
	}
}

func TestParseGraph_CountsAreIntegersAsJSONLoadsGivesThem(t *testing.T) {
	path := "/path/to/graph.json"
	want := path + ": the link counts are not numbers; delete it and run `brain reindex`"
	for _, links := range []string{
		`{"total": 3.0, "resolved": 0, "dropped": {}}`,
		`{"total": 0, "resolved": 1.5, "dropped": {}}`,
		`{"total": 1e2, "resolved": 0, "dropped": {}}`,
		`{"total": 2E1, "resolved": 0, "dropped": {}}`,
		`{"total": 0, "resolved": 0, "dropped": {"external": 2.0}}`,
		`{"total": true, "resolved": 0, "dropped": {}}`,
	} {
		_, err := graph.ParseGraph([]byte(`{"edges": [], "links": `+links+`}`), path)
		if err == nil || err.Error() != want {
			t.Errorf("%s: got %v, want %q", links, err, want)
		}
	}
	g, err := graph.ParseGraph([]byte(`{"edges": [], "links": {"total": 3, "resolved": -1, "dropped": {"external": 0}}}`), path)
	if err != nil {
		t.Fatal(err)
	}
	if g.Links.Total != 3 || g.Links.Resolved != -1 || g.Links.Dropped["external"] != 0 {
		t.Fatalf("links not parsed: %+v", g.Links)
	}
}

func TestParseGraph_InvalidJSONNamesTheDecoderError(t *testing.T) {
	path := "/path/to/graph.json"
	valid := `{"edges": [], "links": {"total": 0, "resolved": 0, "dropped": {}}}`
	for _, tc := range []struct {
		data   string
		reason string
	}{
		{"not json", "invalid character 'o' in literal null (expecting 'u')"},
		{"", "unexpected end of JSON input"},
		{valid + " x", "invalid character 'x' after top-level value"},
		{valid + " {}", "invalid character '{' after top-level value"},
		{"\xef\xbb\xbf" + valid, "invalid character '\\ufeff' looking for beginning of value"},
		{`{"edges": [], "links": {"total": NaN}}`, "invalid character 'N' looking for beginning of value"},
	} {
		_, err := graph.ParseGraph([]byte(tc.data), path)
		want := path + ": graph is not valid JSON (" + tc.reason + "); delete it and run `brain reindex`"
		if err == nil || err.Error() != want {
			t.Errorf("%q: got %v, want %q", tc.data, err, want)
		}
	}
}

func TestParseGraph_NonStringEndsFailTheTypedDecode(t *testing.T) {
	path := "/path/to/graph.json"
	_, err := graph.ParseGraph([]byte(`{"edges": [{"from": 1, "to": "b"}], "links": {"total": 0, "resolved": 0, "dropped": {}}}`), path)
	want := path + ": json: cannot unmarshal number into Go struct field Graph.edges.0.from of type string"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestReadGraph_AnUnstatablePathIsNeverIndexed(t *testing.T) {
	// Path.exists() answers False for every OSError, a NUL byte included.
	area := config.Area{Scope: "project/odd", Path: filepath.Join(t.TempDir(), "a\x00b")}
	_, err := graph.ReadGraph(area, "")
	if !errors.Is(err, graph.ErrNotIndexed) || err.Error() != "project/odd: never indexed; run `brain reindex`" {
		t.Fatalf("got %v", err)
	}
}

func TestReadGraph_InvalidUTF8NamesTheFileOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")
	if err := os.WriteFile(path, []byte("{\"edges\": [], \"links\": \xff}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := graph.ReadGraph(config.Area{Scope: "project/test", Path: dir}, dir)
	if err == nil || err.Error() != path+": not valid UTF-8" {
		t.Fatalf("got %v", err)
	}
}

func TestReadGraph_AnUnreadableGraphNamesTheFileOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := graph.ReadGraph(config.Area{Scope: "project/test", Path: dir}, dir)
	if err == nil || strings.Count(err.Error(), path) != 1 {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 8: Scheitern sehen**

```bash
go test -count=1 ./internal/brain/graph/ > "$TEMP/task05-step8.txt" 2>&1
```
Erwartet: keine Ausgabe, Exit 1.
```bash
grep -E -o -- '--- FAIL: [^ ]+' "$TEMP/task05-step8.txt"
```
Erwartet, genau diese fünf Zeilen (gemessen gegen den Stand nach Step 6):
```
--- FAIL: TestParseGraph_RootThatIsNotAnObjectMissesBothKeys
--- FAIL: TestParseGraph_CountsAreIntegersAsJSONLoadsGivesThem
--- FAIL: TestReadGraph_AnUnstatablePathIsNeverIndexed
--- FAIL: TestReadGraph_InvalidUTF8NamesTheFileOnce
--- FAIL: TestReadGraph_AnUnreadableGraphNamesTheFileOnce
```
Die Meldungen ohne Dateipräfix; die Temp-Pfade von `graph.json` ersetzt der zweite `sed` durch `GRAPH`:
```bash
sed -n -E 's/^[[:space:]]*read_python_test\.go:[0-9]+: //p' "$TEMP/task05-step8.txt" | sed -E 's/[A-Z]:[^ ]*\\graph\.json/GRAPH/g'
```
Erwartet, genau diese zwölf Zeilen (gemessen; Zeilennummern variieren mit dem Kommentar und stehen darum nicht darin; die letzte Zeile trägt den Systemtext eines deutschen Windows):
```
[]: got /path/to/graph.json: graph is not valid JSON (json: cannot unmarshal array into Go value of type map[string]interface {}); delete it and run `brain reindex`, want "/path/to/graph.json: graph is missing edges, links; delete it and run `brain reindex`"
"x": got /path/to/graph.json: graph is not valid JSON (json: cannot unmarshal string into Go value of type map[string]interface {}); delete it and run `brain reindex`, want "/path/to/graph.json: graph is missing edges, links; delete it and run `brain reindex`"
3: got /path/to/graph.json: graph is not valid JSON (json: cannot unmarshal number into Go value of type map[string]interface {}); delete it and run `brain reindex`, want "/path/to/graph.json: graph is missing edges, links; delete it and run `brain reindex`"
"edges": got /path/to/graph.json: graph is not valid JSON (json: cannot unmarshal string into Go value of type map[string]interface {}); delete it and run `brain reindex`, want "/path/to/graph.json: graph is missing edges, links; delete it and run `brain reindex`"
{"total": 3.0, "resolved": 0, "dropped": {}}: got /path/to/graph.json: json: cannot unmarshal number 3.0 into Go struct field Graph.links.total of type int, want "/path/to/graph.json: the link counts are not numbers; delete it and run `brain reindex`"
{"total": 0, "resolved": 1.5, "dropped": {}}: got /path/to/graph.json: json: cannot unmarshal number 1.5 into Go struct field Graph.links.resolved of type int, want "/path/to/graph.json: the link counts are not numbers; delete it and run `brain reindex`"
{"total": 1e2, "resolved": 0, "dropped": {}}: got /path/to/graph.json: json: cannot unmarshal number 1e2 into Go struct field Graph.links.total of type int, want "/path/to/graph.json: the link counts are not numbers; delete it and run `brain reindex`"
{"total": 2E1, "resolved": 0, "dropped": {}}: got /path/to/graph.json: json: cannot unmarshal number 2E1 into Go struct field Graph.links.total of type int, want "/path/to/graph.json: the link counts are not numbers; delete it and run `brain reindex`"
{"total": 0, "resolved": 0, "dropped": {"external": 2.0}}: got /path/to/graph.json: json: cannot unmarshal number 2.0 into Go struct field Graph.links.dropped.external of type int, want "/path/to/graph.json: the link counts are not numbers; delete it and run `brain reindex`"
got GRAPH: open GRAPH: invalid argument
got GRAPH: graph is not valid JSON (invalid character '\xff' looking for beginning of value); delete it and run `brain reindex`
got GRAPH: read GRAPH: Unzulässige Funktion.
```
In `TestParseGraph_RootThatIsNotAnObjectMissesBothKeys` besteht `null` schon, in `TestParseGraph_CountsAreIntegersAsJSONLoadsGivesThem` besteht `true` schon. Die letzte Zeile nennt den Pfad zweimal; das behebt Step 9.

`TestParseGraph_InvalidJSONNamesTheDecoderError` und `TestParseGraph_NonStringEndsFailTheTypedDecode` bestehen schon; der erste hält fest, dass der Umbau auf `json.Decoder` nachfolgende Daten weiter verweigert und Gos Wortlaut behält, der zweite, dass eine Zahl in `from` weiter scheitert.

- [ ] **Step 9: Implementieren** — `internal/brain/graph/read.go` ganz:

```go
package graph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// ErrNotIndexed is what a missing graph.json means, and it is named because
// the overview draws that state rather than reporting a failure: an area
// registered but never indexed is one of the four states in spec 6.3. The
// message is the whole suffix of the error ReadGraph builds, so naming the
// state costs nothing at the reader's end -- the sentence on screen is the
// one that stood here before the sentinel existed.
var ErrNotIndexed = errors.New("never indexed; run `brain reindex`")

// ReadGraph loads and validates graph.json for the specified area. For a
// read-only area stateDir is ultra-brain's state directory until stage 3.
//
// Like core._graph (src/brain/core.py:584-588) it asks Path.exists() first,
// which answers False for any failure to stat, and reads the file as strict
// UTF-8 with universal newlines. A read error already names the file.
func ReadGraph(area config.Area, stateDir string) (*Graph, error) {
	manifestDir := config.ManifestDir(area, stateDir)
	graphPath := filepath.Join(manifestDir, "graph.json")

	if _, err := os.Stat(graphPath); err != nil {
		return nil, fmt.Errorf("%s: %w", area.Scope, ErrNotIndexed)
	}
	text, err := pytext.ReadText(graphPath)
	if err != nil {
		return nil, err
	}

	return ParseGraph([]byte(text), graphPath)
}

// ParseGraph unmarshals and validates the contents of a graph.json file.
func ParseGraph(data []byte, path string) (*Graph, error) {
	loaded, err := decodeKeepingNumbers(data)
	if err != nil {
		return nil, fmt.Errorf("%s: graph is not valid JSON (%v); delete it and run `brain reindex`", path, err)
	}

	// A root that is not an object has neither key, as a nil map has none.
	raw, _ := loaded.(map[string]any)
	var missing []string
	if _, ok := raw["edges"]; !ok {
		missing = append(missing, "edges")
	}
	if _, ok := raw["links"]; !ok {
		missing = append(missing, "links")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%s: graph is missing %s; delete it and run `brain reindex`", path, strings.Join(missing, ", "))
	}

	if err := checkShape(raw, path); err != nil {
		return nil, err
	}

	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &g, nil
}

// decodeKeepingNumbers decodes one JSON value as json.Unmarshal would, but
// keeps numbers as json.Number, so that 3 and 3.0 stay apart the way
// Python's json.loads keeps int and float apart. Whatever the decoder refuses,
// or leaves behind after the value, json.Unmarshal is asked to name.
func decodeKeepingNumbers(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var loaded any
	decodeErr := decoder.Decode(&loaded)
	_, trailingErr := decoder.Token()
	if decodeErr != nil || trailingErr != io.EOF {
		return nil, json.Unmarshal(data, new(any))
	}
	return loaded, nil
}

func checkShape(raw map[string]any, path string) error {
	edgesRaw, ok := raw["edges"].([]any)
	if !ok {
		return fmt.Errorf("%s: edges is not a list of from/to entries; delete it and run `brain reindex`", path)
	}
	for _, edgeItem := range edgesRaw {
		edgeMap, ok := edgeItem.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: edges is not a list of from/to entries; delete it and run `brain reindex`", path)
		}
		if _, hasFrom := edgeMap["from"]; !hasFrom {
			return fmt.Errorf("%s: edges is not a list of from/to entries; delete it and run `brain reindex`", path)
		}
		if _, hasTo := edgeMap["to"]; !hasTo {
			return fmt.Errorf("%s: edges is not a list of from/to entries; delete it and run `brain reindex`", path)
		}
	}

	linksRaw, ok := raw["links"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s: links is not a total/resolved/dropped record; delete it and run `brain reindex`", path)
	}

	totalVal, hasTotal := linksRaw["total"]
	resolvedVal, hasResolved := linksRaw["resolved"]
	droppedVal, hasDropped := linksRaw["dropped"]

	if !hasTotal || !hasResolved || !hasDropped {
		return fmt.Errorf("%s: links is not a total/resolved/dropped record; delete it and run `brain reindex`", path)
	}

	droppedMap, ok := droppedVal.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: links.dropped is not a table of reasons; delete it and run `brain reindex`", path)
	}

	if !isNumber(totalVal) || !isNumber(resolvedVal) {
		return fmt.Errorf("%s: the link counts are not numbers; delete it and run `brain reindex`", path)
	}

	for _, countVal := range droppedMap {
		if !isNumber(countVal) {
			return fmt.Errorf("%s: the link counts are not numbers; delete it and run `brain reindex`", path)
		}
	}

	return nil
}

// isNumber is Python's isinstance(value, int) on what json.loads returns: an
// integer literal is an int, 3.0 and 1e2 are floats. JSON's grammar leaves a
// literal integral exactly when it has no fraction and no exponent. A boolean,
// which Python lets through, is refused (parity list).
func isNumber(v any) bool {
	number, ok := v.(json.Number)
	return ok && !strings.ContainsAny(string(number), ".eE")
}
```

Die Änderungen gegenüber dem Stand nach Step 6, Stelle für Stelle:
- **Importe:** neu `bytes`, `io`, `github.com/xidus90/loomux/internal/brain/pytext`.
- **Kommentar über `ReadGraph`:** vorher `// ReadGraph loads and validates graph.json for the specified area.`; nachher der Kommentar oben (Legacy-Verzeichnis, `Path.exists()`, strenges UTF-8).
- **`ReadGraph`:** vorher `data, err := os.ReadFile(graphPath)`, bei `errors.Is(err, os.ErrNotExist)` → `ErrNotIndexed`, sonst `fmt.Errorf("%s: %w", graphPath, err)`, dann `ParseGraph(data, graphPath)`. Nachher `os.Stat` — **jeder** Fehler → `ErrNotIndexed` —, dann `pytext.ReadText`, dessen Fehler **unverpackt** zurückgeht (ein `*PathError` wie `read <pfad>: Unzulässige Funktion.` und `{pfad}: not valid UTF-8` nennen die Datei schon; die Hülle `%s: %w` hätte sie doppelt genannt), dann `ParseGraph([]byte(text), graphPath)`.
- **`ParseGraph`, Anfang:** vorher (ub `read.go:40-43`, nach Step 6):
  ```go
  	var raw map[string]any
  	if err := json.Unmarshal(data, &raw); err != nil {
  		return nil, fmt.Errorf("%s: graph is not valid JSON (%v); delete it and run `brain reindex`", path, err)
  	}
  ``` Nachher `decodeKeepingNumbers(data)` in dieselbe Meldung, dann `raw, _ := loaded.(map[string]any)` — eine Wurzel, die kein Objekt ist, ergibt eine nil-Map ohne Schlüssel und damit `graph is missing edges, links`. Der Rest von `ParseGraph` bleibt.
- **`decodeKeepingNumbers`:** neu. `json.Decoder.Decode` liest nur den ersten Wert; das anschließende `Token()` muss `io.EOF` liefern, sonst lässt `json.Unmarshal` die Meldung formulieren (`after top-level value`). So bleibt jeder Fehlertext der, den `json.Unmarshal` heute schreibt.
- **`checkShape`:** unverändert bis auf die Literale aus Step 6.
- **`isNumber`:** vorher `switch v.(type) { case float64, int, int64: return true; default: return false }`. Nachher nur `json.Number` ohne `.`, `e`, `E`. Die alten Fälle `float64`, `int`, `int64` kommen mit `UseNumber` nicht mehr vor und entfallen.

- [ ] **Step 10: Paketkommentar** — `internal/brain/graph/model.go:1-8`, weil `render.go` und `synth.go` nicht mitziehen und es kein Front gibt, das `GET /api/graph` beantwortet. Vorher:
```go
// Package graph reads what the indexer writes about one area, renders it,
// answers neighbour questions over it, and builds synthetic graphs of the
// same shape for measurement.
//
// A read model of its own rather than decoding into a map: the front answers
// GET /api/graph from these values, and a map would let a field the indexer
// renames reach the browser unnoticed.
package graph
```
Nachher:
```go
// Package graph reads what the indexer writes about one area and answers
// neighbour questions over it.
//
// A read model of its own rather than decoding into a map: a map would let a
// field the indexer renames reach a caller unnoticed.
package graph
```

- [ ] **Step 11: Bestehen sehen**

```bash
gofmt -l internal/brain/graph
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/graph/...
```
Erwartet: keine Ausgabe.
```bash
go test -count=1 ./internal/brain/graph/
```
Erwartet: Exit 0 und genau eine Zeile: `ok  	github.com/xidus90/loomux/internal/brain/graph` mit angehängter Laufzeit.

Die umgezogenen Tests bleiben grün, und ihre Prüfungen sind schwach genug dafür (ub `graph/read_test.go`, wörtlich):
- `TestReadGraph_ReadError` (Zeilen 212-228) legt ein Verzeichnis als `graph.json` an; `os.Stat` gelingt, `ReadText` scheitert. Geprüft wird nur, dass ein Fehler kommt:
  ```go
  	_, err := graph.ReadGraph(area, tmpDir)
  	if err == nil {
  		t.Fatal("expected error reading directory as file, got nil")
  	}
  ```
- `TestParseGraph_InvalidNodesType` (Zeilen 203-210): `nodes` als Zeichenkette scheitert weiter im typisierten Decode; auch hier nur `err == nil`:
  ```go
  	json := `{"edges": [], "links": {"total": 0, "resolved": 0, "dropped": {}}, "nodes": "not-a-list"}`
  	_, err := graph.ParseGraph([]byte(json), path)
  	if err == nil {
  		t.Fatal("expected error for invalid nodes, got nil")
  	}
  ```
- `TestParseGraph_InvalidShape` (Zeilen 137-201) prüft den Wortlaut exakt (`if err.Error() != tc.want`, nach Step 5 mit Backticks). Die drei Zählwert-Fälle sind Zeichenketten, also kein `json.Number`, und treffen weiter `the link counts are not numbers`:
  ```go
  		{
  			`{"edges": [], "links": {"total": "not-int", "resolved": 0, "dropped": {}}}`,
  			path + ": the link counts are not numbers; delete it and run `brain reindex`",
  		},
  		{
  			`{"edges": [], "links": {"total": 0, "resolved": "not-int", "dropped": {}}}`,
  			path + ": the link counts are not numbers; delete it and run `brain reindex`",
  		},
  		{
  			`{"edges": [], "links": {"total": 0, "resolved": 0, "dropped": {"external": "one"}}}`,
  			path + ": the link counts are not numbers; delete it and run `brain reindex`",
  		},
  ```

- [ ] **Step 12: Coverage (Verfahren Schritt 6)**

```bash
go test ./internal/brain/graph/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Erwartet: Exit 0.
```bash
go tool cover -func="$TEMP/pkg.out" | grep -E '^github\.com/xidus90/loomux/internal/brain/graph/'
```
Erwartet: `go tool cover -func="$TEMP/pkg.out"` — jede Funktion des Pakets 100.0%. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund. Gemessen in der Modulkopie sind es genau diese sieben Funktionen, jede mit `100.0%`, in dieser Reihenfolge: `Neighbors`, `RenderNeighbors`, `ReadGraph`, `ParseGraph`, `decodeKeepingNumbers`, `checkShape`, `isNumber`. Zeilennummern variieren mit dem Kommentar. `decodeKeepingNumbers` läuft mit gültigem JSON, mit `not json` (Decode-Fehler) und mit `valid + " {}"` (Decode gelingt, `Token()` liefert `{` ohne Fehler).
```bash
go tool cover -func="$TEMP/pkg.out" | grep -E '^github\.com/xidus90/loomux/internal/brain/graph/' | grep -v -E '100\.0%$'
```
Erwartet: keine Ausgabe, Exit 1.

- [ ] **Step 13: Stagen und Tor**

```bash
git add internal/brain/graph/model.go internal/brain/graph/neighbors.go internal/brain/graph/read.go internal/brain/graph/neighbors_test.go internal/brain/graph/read_test.go internal/brain/graph/read_python_test.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0 (gofmt, vet, Tests mit Coverage, `covergate`, Pilot-Binary).

- [ ] **Step 14: Commit**

```bash
git branch --show-current
```
Erwartet: `sdd-1b-1` (Hauptcheckout: `master`).
```bash
git rev-parse --short HEAD
```
Erwartet: der Kurz-Hash des Commits von Task 4.
```bash
git diff --cached --stat
```
Erwartet: genau die sechs Dateien `internal/brain/graph/model.go`, `neighbors.go`, `neighbors_test.go`, `read.go`, `read_python_test.go`, `read_test.go` und eine Summenzeile, die mit ` 6 files changed` beginnt.
```bash
printf '%s\n' 'Move the graph reader and check graph.json the way Python does' '' 'Messages name `brain reindex` in backticks, a missing or unstatable file is' 'never indexed, the file is read as strict UTF-8, a root that is not an object' 'misses both keys, and only integer literals count as link counts.' > "$TEMP/loomux-msg.txt"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
git commit -F "$TEMP/loomux-msg.txt"
```
Erwartet: Exit 0. Der Commit löst `.githooks/pre-commit` noch einmal aus (gewollt) und `.githooks/commit-msg`, das `bin/loomux.exe check commit-msg` ruft (ohne Binary `go run ./cmd/loomux check commit-msg`); beide lassen die englische Nachricht durch. Nach der Ausgabe der Hooks bestätigt Git mit genau einer Zeile, die mit `[sdd-1b-1 ` und dem Kurz-Hash beginnt und mit `] Move the graph reader and check graph.json the way Python does` endet.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität, gleich `git config user.name` und `git config user.email`.
```bash
git log -1 --format=%B
```
Erwartet: genau die fünf Zeilen der Nachrichtendatei (Betreff, Leerzeile, drei Zeilen Rumpf), keine `Co-Authored-By`-Zeile.

- [ ] **Step 15: Bericht**

Befunde gegen die Umzugsregel:
- `read.go`: Importe `bytes`, `io`, `pytext` neu; `ReadGraph` prüft über `os.Stat` statt über den Lesefehler und gibt Lesefehler **ohne** die ub-Hülle `"%s: %w", graphPath` zurück (die Datei stand sonst zweimal in der Meldung); neue Funktion `decodeKeepingNumbers`; `isNumber` neu geschrieben; Kommentar über `ReadGraph` erweitert. Der Kommentar über `ErrNotIndexed` (Übersicht, spec 6.3) blieb wortgleich, obwohl loomux keine Übersicht hat.
- `model.go`: Paketkommentar gekürzt (nannte `render.go`, `synth.go` und das Front).
- `Document` zieht ohne Nutzer mit.
- Kein Test entfällt; `render_test.go` und `synth_test.go` bleiben mit ihren Quellen in ub.

Paritätszeilen, die dieser Task schafft (Task 12 trägt sie ein):
1. Wahrheitswert als Zählwert in `links` (`"total": true`): Python nimmt ihn (`bool` ist `int`), loomux meldet `the link counts are not numbers`.
2. Nicht-Zeichenketten in `from`/`to`: Python wandelt per `str()` (`1` → `1`, `null` → `None`); loomux bricht ab mit `{pfad}: json: cannot unmarshal number into Go struct field Graph.edges.0.from of type string`, bei einer Liste mit `cannot unmarshal array`, bei einem Objekt mit `cannot unmarshal object` an derselben Stelle (gemessen), und liest `null` als leere Zeichenkette.
3. Wortlaut des JSON-Fehlers in Klammern (gemessen, siehe „Vorab gemessen"): Python `json.JSONDecodeError` mit `Expecting value: line 1 column 1 (char 0)` für `not json`, `Extra data: line 1 column 4 (char 3)` für `{} x`, `Unexpected UTF-8 BOM (decode using utf-8-sig): line 1 column 1 (char 0)` für ein BOM; loomux `encoding/json` mit `invalid character 'o' in literal null (expecting 'u')` für `not json`, `unexpected end of JSON input` für eine leere Datei, `invalid character 'x' after top-level value` für nachfolgende Daten, `invalid character '\ufeff' looking for beginning of value` für ein BOM und `invalid character 'N' looking for beginning of value` für `NaN`.
4. `NaN`, `Infinity`, `-Infinity` in `graph.json`: Python liest sie (als Zählwert dann `the link counts are not numbers`), loomux meldet für `NaN` `graph is not valid JSON (invalid character 'N' looking for beginning of value)` (gemessen); `Infinity` und `-Infinity` sind ebenso kein JSON und enden in derselben Meldung mit dem Zeichen, an dem `encoding/json` scheitert.
5. Wurzelwert, der kein Objekt ist: bei `[]` und `"x"` gleich; `"edges"` → Python `graph is missing links` (Teilzeichenkette), `["edges","links"]`, eine Zahl oder `null` → Python Traceback (`TypeError`); loomux meldet in allen Fällen `graph is missing edges, links`.
6. Ganzzahl jenseits von `int` als Zählwert (`99999999999999999999`): Python nimmt sie, loomux `{pfad}: json: cannot unmarshal number 99999999999999999999 into Go struct field Graph.links.total of type int` (für `total` gemessen; `resolved` und `dropped.<grund>` nennen ihr Feld).
7. `nodes` oder `scope` von falschem Typ: Python liest beide nie, loomux bricht ab mit `{pfad}: json: cannot unmarshal string into Go struct field Graph.nodes of type []graph.Node` bzw. `{pfad}: json: cannot unmarshal number into Go struct field Graph.scope of type string` (gemessen; ub-Verhalten, `TestParseGraph_InvalidNodesType`).
8. Vorhandenes, aber unlesbares `graph.json` (etwa ein Verzeichnis): Python `error: [Errno 13] Permission denied: '{pfad}'`, loomux `read {pfad}: ` mit dem Systemtext, auf dieser Maschine `read {pfad}: Unzulässige Funktion.` (gemessen).
9. Ungültiges UTF-8 in `graph.json`: Python Traceback (`UnicodeDecodeError` ist kein `JSONDecodeError`), loomux `error: {pfad}: not valid UTF-8`, Exit 1.

---

### Task 6: `internal/brain/reader` — Umzug, Lesen und Abschnitte nach Python

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 5 sind committet. Dazu `SRC=/c/Users/micro/Documents/#GIT/loomux-src/ub`.

**Files:**
- Move: `$SRC/pkg/reader/read.go`, `$SRC/pkg/reader/section.go`, `$SRC/pkg/reader/reader_test.go` → `internal/brain/reader/` (Paketname `reader` bleibt)
- Modify (nach dem Umzug): `internal/brain/reader/read.go`, `internal/brain/reader/section.go`, `internal/brain/reader/reader_test.go` (ein Literal)
- Create: Test `internal/brain/reader/python_test.go`

**Interfaces:**
- Consumes: Task 2 `pytext.ReadText(path string) (string, error)`, `pytext.SplitLines(s string) []string`, `pytext.Strip(s string) string`, `pytext.RStrip(s string) string`, `pytext.Repr(s string) string`; Task 3 `privacy.Contained(scope, relative string) (string, error)`, `privacy.IsReadable(manifest *config.Manifest, relative string) bool`, `privacy.MatchesGlobs(patterns []string, relative string) bool`, `privacy.ReviewExcludes(manifest *config.Manifest) ([]string, error)`, `privacy.Channel`, `privacy.ChannelLocal`, `privacy.ChannelCloud`; `config.Area`, `config.Manifest`.
- Produces:
```go
func ReadDocument(area config.Area, manifest *config.Manifest, relative, section string, ch privacy.Channel, stateDir string) (string, error) // section "" = ganze Datei
func ExtractSection(content, heading string) (string, error)
```
  Ergänzungen: keine. Unexportiert bleibt `headingLevel(line string) int` wortgleich. `stateDir` ist im Rumpf ungenutzt (so in ub, `read.go:19`); die Signatur bleibt wegen des Vertrags.

- [ ] **Step 1: Zielverzeichnis anlegen** (Umzugsverfahren Schritt 1)

```bash
mkdir -p "$LOOMUX/internal/brain/reader"
```
Expected: keine Ausgabe, Exit 0.

- [ ] **Step 2: Nur die drei genannten Dateien kopieren**

```bash
cp "$SRC/pkg/reader/read.go" "$SRC/pkg/reader/section.go" "$SRC/pkg/reader/reader_test.go" "$LOOMUX/internal/brain/reader/"
```
Expected: keine Ausgabe, Exit 0.

Der Paketname bleibt `reader` (Quelle: `package reader` / `package reader_test`), Verfahrensschritt 2 entfällt.

- [ ] **Step 3: Importpfade umschreiben** (Verfahren Schritt 3). Zuerst prüfen, dass im ganzen Baum nur die drei neuen Dateien einen ub-Pfad tragen:

```bash
grep -rl 'github.com/xidus90/ultra-brain/' "$LOOMUX" --include=*.go
```
Expected: genau zwei Zeilen in beliebiger Reihenfolge, `$LOOMUX/internal/brain/reader/read.go` und `$LOOMUX/internal/brain/reader/reader_test.go` (mit dem ausgeschriebenen Wert von `$LOOMUX`; `section.go` importiert nichts aus ub). Weitere Treffer gibt es nicht, weil die Umzüge der Tasks 3 bis 5 ihre ub-Pfade schon umgeschrieben und committet haben. Dann:

```bash
sed -i 's#github.com/xidus90/ultra-brain/pkg/config#github.com/xidus90/loomux/internal/config#g; s#github.com/xidus90/ultra-brain/pkg/privacy#github.com/xidus90/loomux/internal/brain/privacy#g; s#github.com/xidus90/ultra-brain/pkg/reader#github.com/xidus90/loomux/internal/brain/reader#g' "$LOOMUX/internal/brain/reader/read.go" "$LOOMUX/internal/brain/reader/reader_test.go"
```
Expected: keine Ausgabe, Exit 0.

```bash
grep -rl 'github.com/xidus90/ultra-brain/' "$LOOMUX" --include=*.go
```
Expected: keine Ausgabe, Exit 1.

Die Paketbezeichner `config.`, `privacy.`, `reader.` ändern sich nicht; der zweite `sed` des Verfahrens entfällt.

- [ ] **Step 4: Formatieren, prüfen, grün vor jeder Änderung** (Verfahren Schritt 4, ohne `go mod tidy` — das Paket holt keine neue Fremdabhängigkeit)

```bash
gofmt -w "$LOOMUX/internal/brain/reader"
```
Expected: keine Ausgabe, Exit 0. `gofmt` sortiert dabei den Importblock von `read.go` und `reader_test.go` um (`internal/brain/privacy`, in `reader_test.go` auch `internal/brain/reader`, steht jetzt vor `internal/config`); das ist die einzige Formatänderung (gemessen).

```bash
go vet ./internal/brain/reader/
```
Expected: keine Ausgabe.

```bash
go test ./internal/brain/reader/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/reader`. Die Zeilen von `TestReadDocument` (`../escape.md`, `secrets/**`, `review/**`, `LayoutReview "."` auf `cloud`) verhalten sich unter Task 3s `Contained`/`MatchesGlobs`/`ReviewExcludes` wie unter ubs Fassung.

- [ ] **Step 5: Failing Tests nach Python schreiben** — `internal/brain/reader/python_test.go`

```go
package reader_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/reader"
	"github.com/xidus90/loomux/internal/config"
)

// Every expected value in this file was measured at the Python reference
// (brain.core._section and Path.read_text, Python 3.14.7, ultra-brain tag
// loomux-1a-source) on 2026-09-15.

func TestExtractSectionFollowsPython(t *testing.T) {
	for _, c := range []struct {
		name, text, heading, want string
	}{
		{"carriage return", "# A\rbody\r# B\rrest", "A", "# A\nbody\n"},
		{"crlf", "# A\r\nbody\r\n# B\r\nrest\r\n", "A", "# A\nbody\n"},
		{"form feed", "# A\fbody\f# B", "A", "# A\nbody\n"},
		{"vertical tab", "# A\vbody\v# B", "A", "# A\nbody\n"},
		{"file separator", "# A\x1cbody\x1c# B", "A", "# A\nbody\n"},
		{"group separator", "# A\x1dbody\x1d# B", "A", "# A\nbody\n"},
		{"record separator", "# A\x1ebody\x1e# B", "A", "# A\nbody\n"},
		{"next line", "# A\xc2\x85body\xc2\x85# B", "A", "# A\nbody\n"},
		{"line separator", "# A\xe2\x80\xa8body\xe2\x80\xa8# B", "A", "# A\nbody\n"},
		{"paragraph separator", "# A\xe2\x80\xa9body\xe2\x80\xa9# B", "A", "# A\nbody\n"},
		{"unit separator around the title", "#\x1fA\nbody\x1f\n", "A", "#\x1fA\nbody\n"},
		{"ideographic space after the title", "# A\xe3\x80\x80\nbody\n", "A", "# A\xe3\x80\x80\nbody\n"},
		{"no-break space before the title", "#\xc2\xa0A\nbody\n", "A", "#\xc2\xa0A\nbody\n"},
		{"unicode whitespace at the end", "# A\nbody\xe3\x80\x80\xc2\xa0\n\n", "A", "# A\nbody\n"},
		{"a hashtag line ends the section", "## A\nbody\n#tag\nafter\n", "A", "## A\nbody\n"},
		{"a deeper heading stays inside", "## A\nbody\n### B\nmore\n", "A", "## A\nbody\n### B\nmore\n"},
		{"closing hashes belong to the title", "## A ##\nbody\n", "A ##", "## A ##\nbody\n"},
		{"empty heading", "# \nx", "", "# \nx\n"},
		{"hashes only", "###\nx\n## y\n", "", "###\nx\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := reader.ExtractSection(c.text, c.heading)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("expected %q, got %q", c.want, got)
			}
		})
	}
}

func TestExtractSectionNamesTheMissingHeadingLikeRepr(t *testing.T) {
	doc := "# Title\n\nIntroduction paragraph.\n\n## First\n\nContent of first.\n"
	for _, c := range []struct {
		name, text, heading, want string
	}{
		{"plain", doc, "NonExistent", "no section titled 'NonExistent'"},
		{"apostrophe", doc, "it's", `no section titled "it's"`},
		{"both quotes", doc, `both ' and "`, `no section titled 'both \' and "'`},
		{"tab", doc, "a\tb", `no section titled 'a\tb'`},
		{"empty text", "", "A", "no section titled 'A'"},
		{"byte order mark before the hash", "\xef\xbb\xbf# A\nbody\n", "A", "no section titled 'A'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := reader.ExtractSection(c.text, c.heading)
			if err == nil || err.Error() != c.want {
				t.Fatalf("expected %q, got %v", c.want, err)
			}
		})
	}
}

// area writes files below a fresh directory and returns a writable area on it.
func area(t *testing.T, files map[string]string) config.Area {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return config.Area{Scope: "project/test", Path: dir}
}

func TestReadDocumentFoldsNewlinesLikeReadText(t *testing.T) {
	a := area(t, map[string]string{"crlf.md": "# A\r\nb\rc\n"})
	got, err := reader.ReadDocument(a, &config.Manifest{}, "crlf.md", "", privacy.ChannelLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "# A\nb\nc\n" {
		t.Errorf("expected %q, got %q", "# A\nb\nc\n", got)
	}
}

func TestReadDocumentFindsASectionBehindLoneCarriageReturns(t *testing.T) {
	a := area(t, map[string]string{"cr.md": "# A\rbody\r# B\rrest"})
	got, err := reader.ReadDocument(a, &config.Manifest{}, "cr.md", "A", privacy.ChannelLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "# A\nbody\n" {
		t.Errorf("expected %q, got %q", "# A\nbody\n", got)
	}
}

func TestReadDocumentRefusesInvalidUTF8(t *testing.T) {
	a := area(t, map[string]string{"bad.md": "# A\n\xff\n"})
	_, err := reader.ReadDocument(a, &config.Manifest{}, "bad.md", "", privacy.ChannelLocal, "")
	want := filepath.Join(a.Path, "bad.md") + ": not valid UTF-8"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadDocumentKeepsTheByteOrderMark(t *testing.T) {
	a := area(t, map[string]string{"bom.md": "\xef\xbb\xbf# A\n"})
	got, err := reader.ReadDocument(a, &config.Manifest{}, "bom.md", "", privacy.ChannelLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "\xef\xbb\xbf# A\n" {
		t.Errorf("expected %q, got %q", "\xef\xbb\xbf# A\n", got)
	}
	_, err = reader.ReadDocument(a, &config.Manifest{}, "bom.md", "A", privacy.ChannelLocal, "")
	if err == nil || err.Error() != "no section titled 'A'" {
		t.Fatalf("expected %q, got %v", "no section titled 'A'", err)
	}
}

func TestReadDocumentHandsOnAMissingFile(t *testing.T) {
	a := area(t, nil)
	_, err := reader.ReadDocument(a, &config.Manifest{}, "missing.md", "", privacy.ChannelLocal, "")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected a not-exist error, got %v", err)
	}
}

func TestReadDocumentChecksNeverBeforeTheReviewCentre(t *testing.T) {
	a := area(t, map[string]string{"review/secret.md": "x"})
	manifest := &config.Manifest{LayoutReview: "review", NeverGlobs: []string{"review/secret.md"}}
	_, err := reader.ReadDocument(a, manifest, "review/secret.md", "", privacy.ChannelCloud, "")
	want := "project/test/review/secret.md is excluded by [privacy] never"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadDocumentChecksContainmentBeforeNever(t *testing.T) {
	a := area(t, nil)
	manifest := &config.Manifest{NeverGlobs: []string{"**"}}
	_, err := reader.ReadDocument(a, manifest, "../secrets/key.txt", "", privacy.ChannelLocal, "")
	want := "project/test/../secrets/key.txt leaves the area"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadDocumentNamesTheContainedPathInRefusals(t *testing.T) {
	a := area(t, map[string]string{"secrets/key.txt": "k", "review/cases/c1.md": "c"})
	manifest := &config.Manifest{LayoutReview: "review", NeverGlobs: []string{"secrets/**"}}
	_, err := reader.ReadDocument(a, manifest, "./secrets//key.txt", "", privacy.ChannelLocal, "")
	want := "project/test/secrets/key.txt is excluded by [privacy] never"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
	_, err = reader.ReadDocument(a, manifest, "review/./cases/c1.md", "", privacy.ChannelCloud, "")
	want = "project/test/review/cases/c1.md is the review centre; refused on the cloud channel"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadDocumentAsksForTheReviewCentreOnlyOnTheCloud(t *testing.T) {
	a := area(t, map[string]string{"doc.md": "text\n"})
	got, err := reader.ReadDocument(a, &config.Manifest{LayoutReview: "."}, "doc.md", "", privacy.ChannelLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "text\n" {
		t.Errorf("expected %q, got %q", "text\n", got)
	}
}

func TestReadDocumentWithoutAPathReadsTheAreaItself(t *testing.T) {
	a := area(t, nil)
	for _, relative := range []string{"", "."} {
		_, err := reader.ReadDocument(a, &config.Manifest{}, relative, "", privacy.ChannelLocal, "")
		if err == nil || strings.Contains(err.Error(), "leaves the area") {
			t.Fatalf("%q: expected a read error on the area directory, got %v", relative, err)
		}
	}
}
```

Herkunft der erwarteten Werte (gemessen am 2026-09-15, Python 3.14.7 der Referenz, `brain.core._section` und `Path.read_text(encoding="utf-8")`):

| Eingabe (Python-Schreibweise) | Überschrift | Referenz |
|---|---|---|
| `'# A\rbody\r# B\rrest'`, ebenso mit `\r\n`, `\x0c`, `\x0b`, `\x1c`, `\x1d`, `\x1e`, U+0085, U+2028, U+2029 als Trenner | `A` | `'# A\nbody\n'` |
| `'#\x1fA\nbody\x1f\n'` | `A` | `'#\x1fA\nbody\n'` (`'\x1f'.isspace()` ist `True`) |
| `'# A'` + U+3000 + `'\nbody\n'` | `A` | unverändert zurück |
| `'#'` + U+00A0 + `'A\nbody\n'` | `A` | unverändert zurück |
| `'# A\nbody'` + U+3000 + U+00A0 + `'\n\n'` | `A` | `'# A\nbody\n'` |
| `'## A\nbody\n#tag\nafter\n'` | `A` | `'## A\nbody\n'` |
| `'## A\nbody\n### B\nmore\n'` | `A` | `'## A\nbody\n### B\nmore\n'` |
| `'## A ##\nbody\n'` | `A ##` | `'## A ##\nbody\n'` |
| `'# \nx'` | `''` | `'# \nx\n'` |
| `'###\nx\n## y\n'` | `''` | `'###\nx\n'` |
| Dokument ohne Treffer | `NonExistent` / `it's` / `both ' and "` / `a\tb` | `no section titled 'NonExistent'` / `no section titled "it's"` / `no section titled 'both \' and "'` / `no section titled 'a\tb'` |
| `''` | `A` | `no section titled 'A'` |
| U+FEFF + `'# A\nbody\n'` | `A` | `no section titled 'A'` |
| Datei `b'# A\r\nb\rc\n'` | — | `read_text` → `'# A\nb\nc\n'` |
| Datei `b'\xef\xbb\xbf# A\n'` | — | `read_text` → `'\ufeff# A\n'` (BOM bleibt) |

- [ ] **Step 6: Scheitern sehen**

```bash
go test ./internal/brain/reader/ -count=1
```
Expected: Exit 1; die letzte Zeile beginnt mit `FAIL	github.com/xidus90/loomux/internal/brain/reader`.

```bash
go test ./internal/brain/reader/ -count=1 2>&1 | grep -Eo -- '--- FAIL: [^ ]+'
```
Expected: genau diese Zeilen in dieser Reihenfolge (Namen und Meldungen von `TestExtractSectionFollowsPython` und `TestExtractSectionNamesTheMissingHeadingLikeRepr` gemessen am 2026-09-15 im Wegwerfmodul gegen ubs unverändertes `section.go`):
```
--- FAIL: TestExtractSectionFollowsPython
--- FAIL: TestExtractSectionFollowsPython/carriage_return
--- FAIL: TestExtractSectionFollowsPython/form_feed
--- FAIL: TestExtractSectionFollowsPython/vertical_tab
--- FAIL: TestExtractSectionFollowsPython/file_separator
--- FAIL: TestExtractSectionFollowsPython/group_separator
--- FAIL: TestExtractSectionFollowsPython/record_separator
--- FAIL: TestExtractSectionFollowsPython/next_line
--- FAIL: TestExtractSectionFollowsPython/line_separator
--- FAIL: TestExtractSectionFollowsPython/paragraph_separator
--- FAIL: TestExtractSectionFollowsPython/unit_separator_around_the_title
--- FAIL: TestExtractSectionFollowsPython/unicode_whitespace_at_the_end
--- FAIL: TestExtractSectionNamesTheMissingHeadingLikeRepr
--- FAIL: TestExtractSectionNamesTheMissingHeadingLikeRepr/plain
--- FAIL: TestExtractSectionNamesTheMissingHeadingLikeRepr/both_quotes
--- FAIL: TestExtractSectionNamesTheMissingHeadingLikeRepr/tab
--- FAIL: TestExtractSectionNamesTheMissingHeadingLikeRepr/empty_text
--- FAIL: TestExtractSectionNamesTheMissingHeadingLikeRepr/byte_order_mark_before_the_hash
--- FAIL: TestReadDocumentFoldsNewlinesLikeReadText
--- FAIL: TestReadDocumentFindsASectionBehindLoneCarriageReturns
--- FAIL: TestReadDocumentRefusesInvalidUTF8
--- FAIL: TestReadDocumentKeepsTheByteOrderMark
```

```bash
go test ./internal/brain/reader/ -count=1 2>&1 | grep -Eo '(unexpected error|expected) .*' | grep -v 'not valid UTF-8'
```
Expected: genau diese 19 Zeilen in dieser Reihenfolge, eine je gescheitertem Test oder Untertest außer `TestReadDocumentRefusesInvalidUTF8`; das Muster schneidet das Präfix `python_test.go:<zeile>: ` ab, Zeilennummern variieren mit dem Kommentar:
```
unexpected error: no section titled "A"
unexpected error: no section titled "A"
unexpected error: no section titled "A"
unexpected error: no section titled "A"
unexpected error: no section titled "A"
unexpected error: no section titled "A"
unexpected error: no section titled "A"
unexpected error: no section titled "A"
unexpected error: no section titled "A"
unexpected error: no section titled "A"
expected "# A\nbody\n", got "# A\nbody\u3000\u00a0\n"
expected "no section titled 'NonExistent'", got no section titled "NonExistent"
expected "no section titled 'both \\' and \"'", got no section titled "both ' and \""
expected "no section titled 'a\\tb'", got no section titled "a\tb"
expected "no section titled 'A'", got no section titled "A"
expected "no section titled 'A'", got no section titled "A"
expected "# A\nb\nc\n", got "# A\r\nb\rc\n"
unexpected error: no section titled "A"
expected "no section titled 'A'", got no section titled "A"
```

```bash
go test ./internal/brain/reader/ -count=1 2>&1 | grep -E 'expected ".*bad\.md: not valid UTF-8", got <nil>'
```
Expected: genau ein Treffer (der Pfad unter `t.TempDir()` steht in Gos `%q`-Schreibweise davor).

Grün bleiben `TestExtractSectionFollowsPython/crlf` (ubs Code faltet `\r\n` schon) und `TestExtractSectionNamesTheMissingHeadingLikeRepr/apostrophe` (für `it's` wählen Gos `%q` und `repr` dieselben doppelten Anführungszeichen). Die übrigen neuen Tests sind Stifte und schon grün: `TestReadDocumentHandsOnAMissingFile`, `TestReadDocumentChecksNeverBeforeTheReviewCentre`, `TestReadDocumentChecksContainmentBeforeNever`, `TestReadDocumentNamesTheContainedPathInRefusals`, `TestReadDocumentAsksForTheReviewCentreOnlyOnTheCloud`, `TestReadDocumentWithoutAPathReadsTheAreaItself` (grün erst durch Task 3s `Contained`, das `""` und `.` zu `.` macht; mit ubs `Contained` meldete es `project/test/ leaves the area`).

- [ ] **Step 7: `section.go` nach `core._section` umbauen und das alte Quote-Literal umstellen** — `internal/brain/reader/section.go`, Funktion `ExtractSection`, und `internal/brain/reader/reader_test.go`, `TestExtractSection`.

Importblock vorher:
```go
import (
	"fmt"
	"strings"
)
```
nachher:
```go
import (
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
)
```

Doc-Kommentar: unter der Zeile `// Matches headings where line starts with "#" and the stripped text matches heading.` zwei Zeilen ergänzen:
```go
// The lines, the strip and the quoting are Python's (core._section): splitlines,
// str.strip, str.rstrip and repr.
```

Rumpfanfang vorher:
```go
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
```
nachher:
```go
	lines := pytext.SplitLines(content)
```

Titelvergleich vorher:
```go
			title := strings.TrimSpace(strings.TrimLeft(line, "#"))
```
nachher:
```go
			title := pytext.Strip(strings.TrimLeft(line, "#"))
```

Fehlermeldung vorher:
```go
		return "", fmt.Errorf("no section titled %q", heading)
```
nachher:
```go
		return "", fmt.Errorf("no section titled %s", pytext.Repr(heading))
```

Ende vorher:
```go
	return strings.TrimRight(joined, " \t\r\n") + "\n", nil
```
nachher:
```go
	return pytext.RStrip(joined) + "\n", nil
```

Alles andere in `section.go` bleibt wortgleich (Suche nach `start`, `headingLevel`, Suche nach `end`, `strings.Join`). Die Datei lautet danach vollständig:

```go
package reader

import (
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// ExtractSection extracts a heading and its content up to the next heading
// of equal or higher level.
//
// Matches headings where line starts with "#" and the stripped text matches heading.
// The lines, the strip and the quoting are Python's (core._section): splitlines,
// str.strip, str.rstrip and repr.
func ExtractSection(content, heading string) (string, error) {
	lines := pytext.SplitLines(content)

	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			title := pytext.Strip(strings.TrimLeft(line, "#"))
			if title == heading {
				start = i
				break
			}
		}
	}

	if start == -1 {
		return "", fmt.Errorf("no section titled %s", pytext.Repr(heading))
	}

	level := headingLevel(lines[start])
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "#") {
			l := headingLevel(lines[i])
			if l <= level {
				end = i
				break
			}
		}
	}

	sectionLines := lines[start:end]
	joined := strings.Join(sectionLines, "\n")
	return pytext.RStrip(joined) + "\n", nil
}

func headingLevel(line string) int {
	return len(line) - len(strings.TrimLeft(line, "#"))
}
```
`headingLevel` zählt Bytes, Python zählt Codepunkte; `#` ist ein Byte, die Zahl ist dieselbe.

Im selben Schritt das alte Quote-Literal umstellen (Umzugsverfahren Schritt 5: Literal und der Test, der es prüft, ändern sich gemeinsam) — `internal/brain/reader/reader_test.go`, `TestExtractSection`, Fall 4.

vorher:
```go
	if !strings.Contains(err.Error(), "no section titled \"NonExistent\"") {
```
nachher:
```go
	if !strings.Contains(err.Error(), "no section titled 'NonExistent'") {
```

- [ ] **Step 8: `read.go` über `pytext.ReadText` lesen** — `internal/brain/reader/read.go`, Funktion `ReadDocument`.

Importblock vorher (nach Step 4):
```go
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)
```
nachher:
```go
import (
	"fmt"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)
```

Lesen vorher:
```go
	fullPath := filepath.Join(area.Path, inside)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	text := string(data)
	if section != "" {
```
nachher:
```go
	fullPath := filepath.Join(area.Path, inside)
	// Path.read_text(encoding="utf-8") in core.read: strict UTF-8 and
	// universal newlines, so a CRLF file answers with LF like the reference.
	text, err := pytext.ReadText(fullPath)
	if err != nil {
		return "", err
	}

	if section != "" {
```
Die Reihenfolge der Tore (Enthaltensein → `never` → Prüfzentrum nur auf `cloud` → Lesen → Abschnitt) und beide Meldungen bleiben wortgleich; sie entsprechen `core.read` (core.py:301-313).

Die Datei lautet danach vollständig. `func ReadDocument(` steht weiter in Zeile 14, weil der Importblock `os` gegen `pytext` bei gleicher Zeilenzahl tauscht; `text := string(data)` und die Leerzeile davor fallen weg, dafür kommen die zwei Kommentarzeilen dazu. Im Wegwerfmodul mit Stümpfen für `config`, `privacy` und `pytext` geprüft: `gofmt -l` und `go vet` ohne Ausgabe.

```go
package reader

import (
	"fmt"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// ReadDocument reads a document or section within an area with full privacy,
// containment, and channel enforcement.
func ReadDocument(
	area config.Area,
	manifest *config.Manifest,
	relative, section string,
	ch privacy.Channel,
	stateDir string,
) (string, error) {
	inside, err := privacy.Contained(area.Scope, relative)
	if err != nil {
		return "", err
	}

	if !privacy.IsReadable(manifest, inside) {
		return "", fmt.Errorf("%s/%s is excluded by [privacy] never", area.Scope, inside)
	}

	if ch == privacy.ChannelCloud {
		excludes, err := privacy.ReviewExcludes(manifest)
		if err != nil {
			return "", err
		}
		if privacy.MatchesGlobs(excludes, inside) {
			return "", fmt.Errorf("%s/%s is the review centre; refused on the cloud channel", area.Scope, inside)
		}
	}

	fullPath := filepath.Join(area.Path, inside)
	// Path.read_text(encoding="utf-8") in core.read: strict UTF-8 and
	// universal newlines, so a CRLF file answers with LF like the reference.
	text, err := pytext.ReadText(fullPath)
	if err != nil {
		return "", err
	}

	if section != "" {
		return ExtractSection(text, section)
	}
	return text, nil
}
```

- [ ] **Step 9: Bestehen sehen**

```bash
gofmt -l internal/brain/reader
```
Expected: keine Ausgabe.

```bash
go vet ./internal/brain/reader/
```
Expected: keine Ausgabe.

```bash
go test ./internal/brain/reader/ -count=1
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/reader`.

- [ ] **Step 10: Coverage** (Verfahren Schritt 6)

```bash
go test ./internal/brain/reader/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Expected: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/brain/reader` beginnt; die Prozentzahl dahinter bezieht sich auf `./...` und ist hier nicht die Prüfung.

```bash
go tool cover -func="$TEMP/pkg.out" | grep internal/brain/reader/
```
Expected: jede Funktion des Pakets 100.0%, gemessen im Wegwerfmodul (Zeilennummern nach Step 7 und Step 8):
```
github.com/xidus90/loomux/internal/brain/reader/read.go:14:	ReadDocument	100.0%
github.com/xidus90/loomux/internal/brain/reader/section.go:16:	ExtractSection	100.0%
github.com/xidus90/loomux/internal/brain/reader/section.go:51:	headingLevel	100.0%
```
Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.

- [ ] **Step 11: Stagen, Tor, Commit**

Zuerst stagen: Das Tor bricht mit `pre-commit: inputs differ from the index` und Exit 1 ab, solange ein `*.go` ungestagt oder ungetrackt ist.

```bash
git add internal/brain/reader/read.go internal/brain/reader/section.go internal/brain/reader/reader_test.go internal/brain/reader/python_test.go
```
Expected: keine Ausgabe, Exit 0.

```bash
sh .githooks/pre-commit
```
Expected: Exit 0 (gofmt, vet, Tests mit Coverage, `covergate`, Pilot-Binary).

```bash
git branch --show-current
```
Expected: `sdd-1b-1` (oder `master`, wenn Task 0 den Hauptcheckout gewählt hat).

```bash
git rev-parse --short HEAD
```
Expected: der Kurzhash, den Task 5 nach seinem Commit gemeldet hat.

```bash
git log -1 --format=%s
```
Expected: `Move the graph reader and check graph.json the way Python does` (der Commit von Task 5).

```bash
git diff --cached --stat
```
Expected: genau die vier Dateien `internal/brain/reader/python_test.go`, `read.go`, `reader_test.go`, `section.go`; die letzte Zeile beginnt mit ` 4 files changed`.

```bash
printf '%s\n' 'Move the document reader and read sections like Python' '' 'ReadDocument reads through pytext.ReadText, so a file answers in strict' 'UTF-8 with universal newlines. ExtractSection splits, strips and quotes' 'like core._section.' > "$TEMP/loomux-msg.txt"
```
Expected: keine Ausgabe, Exit 0.

```bash
git commit -F "$TEMP/loomux-msg.txt"
```
Expected: Exit 0. Der Commit löst `.githooks/pre-commit` noch einmal aus (gewollt) und danach `.githooks/commit-msg`, das über `bin/loomux.exe check commit-msg` prüft, dass die Nachricht englisch ist. Nach den Ausgaben der Hooks passt genau eine Zeile auf `^\[(sdd-1b-1|master) [0-9a-f]+\] Move the document reader and read sections like Python$`.

```bash
git log -1 --format='%an <%ae>'
```
Expected: `Christoph Wübbels <christoph.wuebbels@gmail.com>` (die Nutzeridentität, kein Modell).

```bash
git log -1 --format=%B
```
Expected: die Ausgabe beginnt mit genau den fünf Zeilen der Nachrichtendatei (danach nur eine Leerzeile); keine `Co-Authored-By`-Zeile.

- [ ] **Step 12: Bericht**

Paritätszeilen, die dieser Task schafft (Task 12 trägt sie ein):
1. `read --section ""`: Python sucht eine leere Überschrift (gemessen `_section('# \nx', '')` → `'# \nx\n'`), loomux gibt die ganze Datei aus (`ReadDocument`: `section ""` = ganze Datei).
2. `read` mit Pfad `.` oder leerem Pfad: Python liest das Bereichsverzeichnis und scheitert mit `[Errno 13] Permission denied: '<pfad>'` (gemessen), loomux scheitert mit dem Go-Lesefehler auf dem Verzeichnis; beide Exit 1, stderr verschieden.
3. `read` einer fehlenden Datei: Python `[Errno 2] No such file or directory: '<pfad>'` (gemessen), loomux der Go-Wortlaut von `os.ReadFile` (`open <pfad>: The system cannot find the file specified.` unter Windows).

Mitgeführt, nicht neu: ungültiges UTF-8 in einem Dokument (Paritätszeile aus Task 2; `TestReadDocumentRefusesInvalidUTF8` nagelt den loomux-Wortlaut fest) und die `ReviewExcludes`-Ablehnung, bei der Python mit Traceback endet (Task 3).

Befunde zur Umzugsregel: (a) zwei Kommentarzeilen in `section.go` und zwei in `read.go`, die die Python-Herkunft der geänderten Zeilen nennen; (b) `gofmt` hat den Importblock von `read.go` und `reader_test.go` nach dem Umschreiben der Pfade umsortiert; (c) `stateDir` bleibt ungenutzt wie in ub.

---

### Task 7: `internal/brain/catalog` — Umzug

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 6 sind committet. Dazu `SRC=/c/Users/micro/Documents/#GIT/loomux-src/ub`.

**Files:**
- Move: `$SRC/pkg/catalog/area.go`, `$SRC/pkg/catalog/root.go`, `$SRC/pkg/catalog/area_test.go`, `$SRC/pkg/catalog/root_test.go` → `internal/brain/catalog/` (Paketname `catalog` bleibt)
- Nicht: `$SRC/pkg/catalog/catalog.go`, `$SRC/pkg/catalog/catalog_test.go` (Import von `pkg/wiki`, nur vom Go-status-Stumpf gerufen)
- Modify (nach dem Umzug): `internal/brain/catalog/area.go`
- Create: Test `internal/brain/catalog/python_test.go`

**Interfaces:**
- Consumes: Task 2 `pytext.ReadText(path string) (string, error)`; `config.Area`, in Tests `config.ManifestDir(area Area, stateDir string) string`.
- Produces:
```go
func AreaArtifactDir(area config.Area, stateDir string) string
func ReadAreaCatalog(area config.Area, stateDir string) (string, error) // GEÄNDERT: pytext.ReadText
func RenderRootCatalog(areas []config.Area) string                      // "# brain\n\n" + je Bereich nach Scope sortiert "* [s](brain://s/)\n"
```
  Ergänzungen: keine. Unexportiert bleiben `var unsafeScope = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)` und `flatScope(scope string) string` wortgleich (Global Constraints: umgezogene `regexp.MustCompile`-Paketvariablen bleiben; Task 15 misst sie).

- [ ] **Step 1: Zielverzeichnis anlegen** (Umzugsverfahren Schritt 1)

```bash
mkdir -p "$LOOMUX/internal/brain/catalog"
```
Expected: keine Ausgabe, Exit 0.

- [ ] **Step 2: Nur die vier genannten Dateien kopieren**

```bash
cp "$SRC/pkg/catalog/area.go" "$SRC/pkg/catalog/root.go" "$SRC/pkg/catalog/area_test.go" "$SRC/pkg/catalog/root_test.go" "$LOOMUX/internal/brain/catalog/"
```
Expected: keine Ausgabe, Exit 0.

Der Paketname bleibt `catalog`; Verfahrensschritt 2 entfällt.

- [ ] **Step 3: Importpfade umschreiben** (Verfahren Schritt 3). Zuerst prüfen, dass im ganzen Baum nur die neuen Dateien einen ub-Pfad tragen:

```bash
grep -rl 'github.com/xidus90/ultra-brain/' "$LOOMUX" --include=*.go
```
Expected: genau vier Zeilen in beliebiger Reihenfolge, `$LOOMUX/internal/brain/catalog/area.go`, `$LOOMUX/internal/brain/catalog/root.go`, `$LOOMUX/internal/brain/catalog/area_test.go` und `$LOOMUX/internal/brain/catalog/root_test.go` (mit dem ausgeschriebenen Wert von `$LOOMUX`). Weitere Treffer gibt es nicht, weil die Umzüge der Tasks 3 bis 6 ihre ub-Pfade schon umgeschrieben und committet haben. Dann:

```bash
sed -i 's#github.com/xidus90/ultra-brain/pkg/config#github.com/xidus90/loomux/internal/config#g; s#github.com/xidus90/ultra-brain/pkg/catalog#github.com/xidus90/loomux/internal/brain/catalog#g' "$LOOMUX/internal/brain/catalog/area.go" "$LOOMUX/internal/brain/catalog/root.go" "$LOOMUX/internal/brain/catalog/area_test.go" "$LOOMUX/internal/brain/catalog/root_test.go"
```
Expected: keine Ausgabe, Exit 0.

```bash
grep -rl 'github.com/xidus90/ultra-brain/' "$LOOMUX" --include=*.go
```
Expected: keine Ausgabe, Exit 1.

Die Paketbezeichner `config.` und `catalog.` ändern sich nicht; der zweite `sed` des Verfahrens entfällt.

- [ ] **Step 4: Formatieren, prüfen, grün vor jeder Änderung** (Verfahren Schritt 4, ohne `go mod tidy`)

```bash
gofmt -w "$LOOMUX/internal/brain/catalog"
```
Expected: keine Ausgabe, Exit 0, und keine Datei ändert sich (in beiden Testdateien steht `internal/brain/catalog` schon vor `internal/config`; gemessen).

```bash
go vet ./internal/brain/catalog/
```
Expected: keine Ausgabe.

```bash
go test ./internal/brain/catalog/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/catalog`.

- [ ] **Step 5: Failing Tests nach Python schreiben** — `internal/brain/catalog/python_test.go`

```go
package catalog_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/catalog"
	"github.com/xidus90/loomux/internal/config"
)

// Every expected value in this file was measured at the Python reference
// (Path.read_text and the root lines of brain.core.catalog, Python 3.14.7,
// ultra-brain tag loomux-1a-source) on 2026-09-15.

// writableArea writes index.md with the given bytes into a fresh area.
func writableArea(t *testing.T, index string) config.Area {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	return config.Area{Scope: "project/x", Path: dir}
}

func TestReadAreaCatalogFoldsNewlinesLikeReadText(t *testing.T) {
	a := writableArea(t, "# A\r\nb\rc\n")
	got, err := catalog.ReadAreaCatalog(a, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "# A\nb\nc\n" {
		t.Errorf("expected %q, got %q", "# A\nb\nc\n", got)
	}
}

func TestReadAreaCatalogRefusesInvalidUTF8(t *testing.T) {
	a := writableArea(t, "# A\n\xff\n")
	_, err := catalog.ReadAreaCatalog(a, "")
	want := filepath.Join(a.Path, "index.md") + ": not valid UTF-8"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadAreaCatalogKeepsTheByteOrderMark(t *testing.T) {
	a := writableArea(t, "\xef\xbb\xbf# A\n")
	got, err := catalog.ReadAreaCatalog(a, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "\xef\xbb\xbf# A\n" {
		t.Errorf("expected %q, got %q", "\xef\xbb\xbf# A\n", got)
	}
}

func TestReadAreaCatalogHandsOnAMissingIndex(t *testing.T) {
	_, err := catalog.ReadAreaCatalog(config.Area{Scope: "project/x", Path: t.TempDir()}, "")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected a not-exist error, got %v", err)
	}
}

func TestAreaArtifactDirIsTheManifestDir(t *testing.T) {
	for _, scope := range []string{"project/alpha", "\xc3\x84/b", "--x--"} {
		for _, readOnly := range []bool{false, true} {
			a := config.Area{Scope: scope, Path: filepath.Join("some", "path"), ReadOnly: readOnly}
			if got, want := catalog.AreaArtifactDir(a, "state"), config.ManifestDir(a, "state"); got != want {
				t.Errorf("%q read-only %v: expected %q, got %q", scope, readOnly, want, got)
			}
		}
	}
}

func TestRenderRootCatalogSortsByCodePoint(t *testing.T) {
	areas := []config.Area{
		{Scope: "project/beta"},
		{Scope: "project/alpha"},
		{Scope: "core/hub"},
		{Scope: "Zeta"},
		{Scope: "\xc3\xa9t\xc3\xa9"},
	}
	want := "# brain\n\n* [Zeta](brain://Zeta/)\n* [core/hub](brain://core/hub/)\n* [project/alpha](brain://project/alpha/)\n* [project/beta](brain://project/beta/)\n* [\xc3\xa9t\xc3\xa9](brain://\xc3\xa9t\xc3\xa9/)\n"
	if got := catalog.RenderRootCatalog(areas); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
	if areas[0].Scope != "project/beta" || areas[4].Scope != "\xc3\xa9t\xc3\xa9" {
		t.Errorf("the caller's slice was reordered: %v", areas)
	}
}
```

Herkunft der erwarteten Werte (gemessen am 2026-09-15, Python 3.14.7 der Referenz):

| Eingabe | Referenz |
|---|---|
| `index.md` = `b'# A\r\nb\rc\n'`, `read_text(encoding="utf-8")` | `'# A\nb\nc\n'` |
| `index.md` = `b'\xef\xbb\xbf# A\n'` | `'\ufeff# A\n'` (BOM bleibt) |
| `index.md` = `b'# A\n\xff\n'` | `UnicodeDecodeError` (Python endet mit Traceback; loomux-Wortlaut aus Task 2) |
| Wurzelzeilen aus `core.catalog` (core.py:284-289) über `project/beta`, `project/alpha`, `core/hub`, `Zeta`, `été` | `'# brain\n\n* [Zeta](brain://Zeta/)\n* [core/hub](brain://core/hub/)\n* [project/alpha](brain://project/alpha/)\n* [project/beta](brain://project/beta/)\n* [\xe9t\xe9](brain://\xe9t\xe9/)\n'` |
| keine Bereiche | `'# brain\n\n'` (ub-Test `TestRenderRootCatalog`, unverändert) |

`TestAreaArtifactDirIsTheManifestDir` nagelt fest, dass die ub-Kopie `flatScope` dasselbe rechnet wie `config.flat` hinter `config.ManifestDir` (beide `strings.Trim(<[^A-Za-z0-9_.-]+ → "-">, "-")`), damit Task 10 für `catalog` und Task 5 für `graph.json` dasselbe Artefaktverzeichnis finden.

- [ ] **Step 6: Scheitern sehen**

```bash
go test ./internal/brain/catalog/ -count=1
```
Expected: Exit 1; die letzte Zeile beginnt mit `FAIL	github.com/xidus90/loomux/internal/brain/catalog`. Genau diese beiden Tests scheitern (gemessen gegen den unveränderten ub-Code):

```bash
go test ./internal/brain/catalog/ -count=1 2>&1 | grep -Eo -- '--- FAIL: [^ ]+'
```
Expected: genau diese zwei Zeilen in dieser Reihenfolge:
```
--- FAIL: TestReadAreaCatalogFoldsNewlinesLikeReadText
--- FAIL: TestReadAreaCatalogRefusesInvalidUTF8
```

```bash
go test ./internal/brain/catalog/ -count=1 2>&1 | grep -F 'expected "# A\nb\nc\n", got "# A\r\nb\rc\n"'
```
Expected: genau ein Treffer (`grep -F` vergleicht die Backslashes wörtlich; das Präfix `python_test.go:<zeile>: ` davor variiert mit dem Kommentar).

```bash
go test ./internal/brain/catalog/ -count=1 2>&1 | grep -E 'expected ".*index\.md: not valid UTF-8", got <nil>'
```
Expected: genau ein Treffer (der Pfad unter `t.TempDir()` steht in Gos `%q`-Schreibweise davor).

Grün schon vor der Änderung (Stifte): `TestReadAreaCatalogKeepsTheByteOrderMark`, `TestReadAreaCatalogHandsOnAMissingIndex`, `TestAreaArtifactDirIsTheManifestDir`, `TestRenderRootCatalogSortsByCodePoint`.

- [ ] **Step 7: `ReadAreaCatalog` über `pytext.ReadText` lesen** — `internal/brain/catalog/area.go`.

Importblock vorher (nach Step 3):
```go
import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xidus90/loomux/internal/config"
)
```
nachher:
```go
import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)
```

Funktion `ReadAreaCatalog` vorher:
```go
func ReadAreaCatalog(area config.Area, stateDir string) (string, error) {
	catalogPath := filepath.Join(AreaArtifactDir(area, stateDir), "index.md")
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```
nachher:
```go
func ReadAreaCatalog(area config.Area, stateDir string) (string, error) {
	catalogPath := filepath.Join(AreaArtifactDir(area, stateDir), "index.md")
	// Path.read_text(encoding="utf-8") in core.catalog: strict UTF-8 and
	// universal newlines, so a CRLF index answers with LF like the reference.
	text, err := pytext.ReadText(catalogPath)
	if err != nil {
		return "", err
	}
	return text, nil
}
```
`unsafeScope`, `flatScope`, `AreaArtifactDir`, beide Doc-Kommentare und `root.go` bleiben wortgleich.

- [ ] **Step 8: Bestehen sehen**

```bash
gofmt -l internal/brain/catalog
```
Expected: keine Ausgabe.

```bash
go vet ./internal/brain/catalog/
```
Expected: keine Ausgabe.

```bash
go test ./internal/brain/catalog/ -count=1
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/catalog`.

- [ ] **Step 9: Coverage** (Verfahren Schritt 6)

```bash
go test ./internal/brain/catalog/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Expected: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/brain/catalog` beginnt; die Prozentzahl dahinter bezieht sich auf `./...` und ist hier nicht die Prüfung.

```bash
go tool cover -func="$TEMP/pkg.out" | grep internal/brain/catalog/
```
Expected: jede Funktion des Pakets 100.0%, gemessen im Wegwerfmodul (die Importblöcke tauschen `os` gegen `pytext` bei gleicher Zeilenzahl, die Zeilennummern bleiben):
```
github.com/xidus90/loomux/internal/brain/catalog/area.go:14:	flatScope		100.0%
github.com/xidus90/loomux/internal/brain/catalog/area.go:20:	AreaArtifactDir		100.0%
github.com/xidus90/loomux/internal/brain/catalog/area.go:28:	ReadAreaCatalog		100.0%
github.com/xidus90/loomux/internal/brain/catalog/root.go:19:	RenderRootCatalog	100.0%
```
Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.

- [ ] **Step 10: Stagen, Tor, Commit**

Zuerst stagen: Das Tor bricht mit `pre-commit: inputs differ from the index` und Exit 1 ab, solange ein `*.go` ungestagt oder ungetrackt ist.

```bash
git add internal/brain/catalog/area.go internal/brain/catalog/root.go internal/brain/catalog/area_test.go internal/brain/catalog/root_test.go internal/brain/catalog/python_test.go
```
Expected: keine Ausgabe, Exit 0.

```bash
sh .githooks/pre-commit
```
Expected: Exit 0 (gofmt, vet, Tests mit Coverage, `covergate`, Pilot-Binary).

```bash
git branch --show-current
```
Expected: `sdd-1b-1` (oder `master`, wenn Task 0 den Hauptcheckout gewählt hat).

```bash
git rev-parse --short HEAD
```
Expected: der Kurzhash, den Task 6 nach seinem Commit gemeldet hat.

```bash
git log -1 --format=%s
```
Expected: `Move the document reader and read sections like Python` (der Commit von Task 6).

```bash
git diff --cached --stat
```
Expected: genau die fünf Dateien `internal/brain/catalog/area.go`, `area_test.go`, `python_test.go`, `root.go`, `root_test.go`; die letzte Zeile beginnt mit ` 5 files changed`.

```bash
printf '%s\n' 'Move the area catalog and read index.md like Python' '' 'ReadAreaCatalog reads through pytext.ReadText, so an index answers in' 'strict UTF-8 with universal newlines. catalog.go stays behind.' > "$TEMP/loomux-msg.txt"
```
Expected: keine Ausgabe, Exit 0.

```bash
git commit -F "$TEMP/loomux-msg.txt"
```
Expected: Exit 0. Der Commit löst `.githooks/pre-commit` noch einmal aus (gewollt) und danach `.githooks/commit-msg`, das über `bin/loomux.exe check commit-msg` prüft, dass die Nachricht englisch ist. Nach den Ausgaben der Hooks passt genau eine Zeile auf `^\[(sdd-1b-1|master) [0-9a-f]+\] Move the area catalog and read index\.md like Python$`.

```bash
git log -1 --format='%an <%ae>'
```
Expected: `Christoph Wübbels <christoph.wuebbels@gmail.com>` (die Nutzeridentität, kein Modell).

```bash
git log -1 --format=%B
```
Expected: die Ausgabe beginnt mit genau den vier Zeilen der Nachrichtendatei (danach nur eine Leerzeile); keine `Co-Authored-By`-Zeile.

- [ ] **Step 11: Bericht**

Paritätszeile, die dieser Task schafft (Task 12 trägt sie ein):
1. `catalog --scope <s>` ohne `index.md`: Python `[Errno 2] No such file or directory: '<pfad>\index.md'` (gemessen an `read_text` einer fehlenden Datei), loomux der Go-Wortlaut von `os.ReadFile` (`open <pfad>\index.md: The system cannot find the file specified.` unter Windows); beide Exit 1.

Mitgeführt, nicht neu: ungültiges UTF-8 in `index.md` (Paritätszeile aus Task 2; Python endet mit Traceback, `TestReadAreaCatalogRefusesInvalidUTF8` nagelt den loomux-Wortlaut fest). CRLF in `index.md` ist keine Zeile mehr: `ReadText` faltet wie `read_text`.

Befunde zur Umzugsregel: (a) zwei Kommentarzeilen in `ReadAreaCatalog`, die die Python-Herkunft nennen; (b) `unsafeScope`/`flatScope` duplizieren `config.unsafeInScope`/`config.flat` und `AreaArtifactDir` duplizieren `config.ManifestDir`; nach der Umzugsregel wortgleich behalten, die Gleichheit hält `TestAreaArtifactDirIsTheManifestDir` fest. Folgeaufgabe für Stufe 3, im Bericht an den Menschen genannt: `flatScope`, `unsafeScope` und `AreaArtifactDir` durch `config.ManifestDir` ersetzen und die Kopien löschen. Stufe 3 ist der Ort, weil sie `reconcile` nach Go holt und damit das befristete Lesen der Artefakte schreibgeschützter Bereiche aus `%LOCALAPPDATA%\brain` endet (Spec Zeilen 30, 147 und 343); sie entfernt dabei jedes `legacyDir`-Argument in `brain/*`, also auch den `stateDir`, den `ReadAreaCatalog` heute bekommt, und zieht `catalog.go` um, das dieselbe Artefaktablage liest.

---

### Task 8: `internal/brain/search` — Ports, Suche, Befunde, Stempel

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 7 sind committet. Dazu `SRC=/c/Users/micro/Documents/#GIT/loomux-src/ub`.

**Files:**
- Move (Umzugsverfahren, nur diese Dateien), Phase A: ub `pkg/search/{port,http,mcp,daemon,daemon_windows,daemon_other,launcher,qmd,fake}.go` und `pkg/search/{port,http,mcp,daemon,launcher,qmd}_test.go` → `internal/brain/search/`
- Move, Phase D: ub `pkg/search/{search.go,search_test.go,visibility_test.go}` → `internal/brain/search/`; alle drei werden danach nach Python neu gefasst (siehe Schritte 22–25)
- Modify: `internal/brain/search/launcher.go`, `daemon.go`, `qmd.go`, `mcp.go`, `launcher_test.go`, `daemon_test.go`, `mcp_test.go`
- Create: `internal/brain/search/stamp.go`, `stamp_test.go`, `qmd_python_test.go`, `notice_internal_test.go`

**Interfaces:**
- Consumes (alle mit Paketqualifier, nur was der Code dieses Tasks ruft):
  - Task 2: `pytext.Repr(s string) string`, `pytext.SplitLines(s string) []string`, `pytext.Strip(s string) string`, `pytext.IsoFormat(t time.Time) string`, `pytext.ParseAwareIsoFormat(s string) (time.Time, bool)`.
  - Task 3: `type privacy.Channel string`, `privacy.ChannelLocal`, `privacy.ChannelCloud`, `type privacy.VisibleArea struct{ Area config.Area; Manifest *config.Manifest }`, `func privacy.VisibleAreas(registryDir, legacyDir, scope string, ch privacy.Channel) ([]privacy.VisibleArea, error)`, `func privacy.IsReadable(manifest *config.Manifest, relative string) bool`.
  - Task 4: `func identity.ReadIdentities(path string) (map[string]identity.Identity, error)` (fehlendes Register → leere Map), `identity.Identity`.
  - heutiger Baum: `func config.ManifestDir(area config.Area, stateDir string) string`, `config.Area` (Felder `Scope`, `ReadOnly`), `config.Manifest`; `func testlock.Lock(t testing.TB, path string)` (nur `visibility_test.go`).
- Produces, umgezogen und unverändert (`package search`, ub `pkg/search`):
```go
// port.go
type Profile string
const (
	ProfileKeyword Profile = "keyword"
	ProfileFast    Profile = "fast"
	ProfileFull    Profile = "full"
)
type SearchHit struct {
	Collection string
	Scope      string
	Relative   string
	Line       int
	Title      string
	Snippet    string
	Score      float64
	ContentKey string
}
type SearchPort interface {
	Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error)
	Indexed(collection string) ([]string, error)
	Refresh(collections []string) error
	NotYetSearchable() (int, error)
	Embed(collections []string) error
}
// http.go
const (
	DefaultHost      = "localhost"
	DefaultPort      = 8765
	HandshakeTimeout = 5 * time.Second
	ProbeTimeout     = 500 * time.Millisecond
	QueryTimeout     = 300 * time.Second
)
type Session interface {
	Call(name string, arguments map[string]any) (map[string]any, error)
	Close() error
}
type HTTPSession struct {
	Host         string
	Port         int
	URL          string // wins over Host and Port when set; Tasks 11 and 12 point it at an httptest server
	HTTPClient   *http.Client
	PollInterval time.Duration
	// unexported: mu sync.Mutex, id int64
}
func NewHTTPSession(port int) *HTTPSession // http://localhost:<port>/mcp
func (s *HTTPSession) Reachable() bool
func (s *HTTPSession) WaitUntilReachable(timeout time.Duration) error
func (s *HTTPSession) Handshake() error
func (s *HTTPSession) Call(name string, arguments map[string]any) (map[string]any, error)
func (s *HTTPSession) Close() error
// fake.go
type ScriptedSearch struct {
	Hits []SearchHit
	Err  error
}
type ScriptedIndexed struct {
	Paths []string
	Err   error
}
type ScriptedPending struct {
	Count int
	Err   error
}
type FakePort struct {
	Results   []ScriptedSearch
	Refreshes []error
	Listings  map[string]ScriptedIndexed
	Pending   []ScriptedPending

	Calls            []SearchCall
	Refreshed        [][]string
	Listed           []string
	Embedded         [][]string
	SearchableCounts int
}
type SearchCall struct {
	Query       string
	Collections []string
	Profile     Profile
	N           int
}
func NewFakePort() *FakePort // Listings initialised, everything else zero
func (f *FakePort) Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error)
func (f *FakePort) Indexed(collection string) ([]string, error)
func (f *FakePort) Refresh(collections []string) error
func (f *FakePort) NotYetSearchable() (int, error)
func (f *FakePort) Embed(collections []string) error
// mcp.go
const ColdAttempts = 3
type ConnectFunc func(env map[string]string) (Session, error)
type QmdMcpOption func(*QmdMcpPort)
func WithConnect(fn ConnectFunc) QmdMcpOption
func WithColdAttempts(attempts int) QmdMcpOption
func WithCLI(cli SearchPort) QmdMcpOption
func WithBackbone(backbone Backbone) QmdMcpOption
func WithPort(port int) QmdMcpOption
type QmdMcpPort struct{ /* unexported fields only */ }
func NewQmdMcpPort(opts ...QmdMcpOption) *QmdMcpPort // connect only when none was given, cli &QmdPort{Executable: "qmd"} when none was given
func (p *QmdMcpPort) Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error)
func (p *QmdMcpPort) Close() error
func (p *QmdMcpPort) Indexed(collection string) ([]string, error)
func (p *QmdMcpPort) Refresh(collections []string) error
func (p *QmdMcpPort) NotYetSearchable() (int, error)
func (p *QmdMcpPort) Embed(collections []string) error
// daemon.go
type Backbone string
const (
	BackboneCUDA   Backbone = "cuda"
	BackboneVulkan Backbone = "vulkan"
	BackboneCPU    Backbone = "cpu"
)
const DefaultBackbone = BackboneCUDA
func ParseBackbone(s string) (Backbone, error)
func BackboneEnv(backbone Backbone) map[string]string
type DaemonSpawner func(argv []string, env []string) error
func DefaultSpawner(argv []string, env []string) error
func StartDaemon(env map[string]string, port int) error
// launcher.go
func Launcher(name string) ([]string, error)
func ResolveLauncher(executable string) ([]string, error)
// qmd.go
type RunnerFunc func(argv []string) ([]byte, []byte, int, error)
type QmdPort struct {
	Executable string
	Runner     RunnerFunc
}
func (q *QmdPort) Refresh(collections []string) error
func (q *QmdPort) NotYetSearchable() (int, error)
func (q *QmdPort) Embed(collections []string) error
```
- Produces, geändert oder neu (Vertrag, dazu die Signaturen, die der Vertrag nur beschreibt):
```go
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
- Produces, Ergänzungen dieses Tasks (alle unexportiert, kein neuer Export):
```go
// qmd.go
type launcherFailure struct{ err error }      // trägt einen Launcher-Fehler aus DefaultRunner zu invoke
func (f *launcherFailure) Error() string
func (q *QmdPort) Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error) // GEÄNDERT (vom Vertrag nicht genannt): geht über invoke wie Pythons search über _invoke
func (q *QmdPort) Indexed(collection string) ([]string, error)                                            // GEÄNDERT (vom Vertrag nicht genannt): pytext.SplitLines, Pfad ungetrimmt wie qmd.py
// daemon.go
func backboneChosenByUser() bool
// mcp.go
type QmdMcpPort struct {
	backbone Backbone
	env      map[string]string
	port     int
	connect  ConnectFunc
	cli      SearchPort
	attempts int
	notice   func(message string) // neues Feld

	session Session
	mu      sync.Mutex
}
var connectDefault = DefaultConnect                                          // Naht für notice_internal_test.go
func translateReply(reply map[string]any, collections []string) []SearchHit // GEÄNDERT (Schritt 17c, vom Vertrag nicht genannt): nimmt die Zeilennummern des Daemons vom Snippet
func withoutLineNumbers(snippet string, line int) string                     // NEU (Schritt 17c)
// search.go
func askTwice(port SearchPort, query string, collections []string, profile Profile, n int) ([]SearchHit, []string, error) // core._ask
func assemble(hits []SearchHit, areas []privacy.VisibleArea, legacyDir string, n int) (*SearchAnswer, error)           // core._assemble
// Testhelfer
func clearBackbone(t *testing.T)                                                        // daemon_test.go
func stampWorld(t *testing.T, content string) string                                    // stamp_test.go
var searchNow = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)                           // search_test.go
func areaEntry(scope, path string) string                                               // search_test.go
func writeArea(t *testing.T, root, scope, manifest string, registered ...string) string // search_test.go
func writeStamp(t *testing.T, stateDir, content string)                                 // search_test.go
func manifestOf(scope string) string                                                    // search_test.go
func closedWorld(t *testing.T, sealed bool) string                                      // visibility_test.go
```

**Messgrundlage.** Alle Wortlaute und erwarteten Werte in den Tests unten sind am 2026-09-15 an der Referenz gemessen (`uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "<Ausdruck>"` mit `PYTHONDONTWRITEBYTECODE=1`, nur lesende Ausdrücke: `core._ask` mit einem Fake-Port, `core._assemble` mit Bereichen im Speicher und nicht existierendem Zustandsverzeichnis, `cli._print_search` über `contextlib.redirect_stdout`, `datetime.fromisoformat(<text>).isoformat()`, `core.stale_reconcile` mit im Speicher ersetztem `read_last_run`, `QmdPort(runner=<stub>)`, `launcher("nonexistent-qmd-xyz")`, `brain.daemon.server.WARMING_NOTICE`). Die Zeilennummern des Daemons stammen aus dem Quelltext von qmd 2.8.3 (`%APPDATA%/npm/node_modules/@tobilu/qmd/dist`): `mcp/server.js:301` schreibt `snippet: addLineNumbers(snippet, line)`, `cli/formatter.js:17-20` teilt an `'\n'` und setzt `${startLine + i}: ` davor; `qmd_mcp.py:_translate` (die Python-Referenz) reicht den Snippet unverändert weiter. Der gesamte Go-Code dieses Tasks ist in einem Wegwerf-Modul mit einer Kopie von `internal/config` und Stubs für `pytext`, `privacy`, `identity` (Vertragssignaturen) kompiliert und grün gelaufen: Paket-Coverage 100,0 % in jeder Commit-Phase (A, B, C, D), `gofmt -l` leer, `go vet` sauber. Alle „Scheitern sehen"-Meldungen sind gemessen: die der Phase B mit den neuen Tests gegen den unveränderten ub-Code, die der Phase C ohne `stamp.go`, die der Phase D mit den ub-Dateien und einem `privacy`-Stub mit der Signatur `VisibleManifest(dir string, ch Channel) (*config.Manifest, bool, error)` aus Task 3.

---

#### Phase A — die Ports ziehen um

- [ ] **Step 1: Zielverzeichnis anlegen**

```bash
mkdir -p "$LOOMUX/internal/brain/search"
```
Erwartet: keine Ausgabe, Exit 0.

- [ ] **Step 2: Nur die Port-Dateien kopieren** (`search.go` und seine beiden Testdateien bleiben bis Phase D draußen: `search.go` ruft `privacy.VisibleManifest` mit zwei Rückgaben, Task 3 hat drei daraus gemacht — das Paket würde nicht kompilieren und Schritt 4 des Umzugsverfahrens nie grün)

```bash
cp "$SRC"/pkg/search/{port,http,mcp,daemon,daemon_windows,daemon_other,launcher,qmd,fake}.go "$SRC"/pkg/search/{port,http,mcp,daemon,launcher,qmd}_test.go "$LOOMUX/internal/brain/search/"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
ls -1 internal/brain/search
```
Erwartet, genau diese 15 Namen, einer je Zeile:
```
daemon.go
daemon_other.go
daemon_test.go
daemon_windows.go
fake.go
http.go
http_test.go
launcher.go
launcher_test.go
mcp.go
mcp_test.go
port.go
port_test.go
qmd.go
qmd_test.go
```

- [ ] **Step 3: Importpfad umschreiben** (der Paketname `search` bleibt; die Port-Dateien importieren außer der Standardbibliothek nichts, nur die Tests den ub-Pfad `github.com/xidus90/ultra-brain/pkg/search`)

```bash
grep -rl 'github.com/xidus90/ultra-brain/pkg/search' "$LOOMUX" --include=*.go | xargs -r sed -i 's#github.com/xidus90/ultra-brain/pkg/search#github.com/xidus90/loomux/internal/brain/search#g'
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rn 'github.com/xidus90/ultra-brain' internal/brain/search
```
Erwartet: keine Ausgabe, Exit 1.

- [ ] **Step 4: Grün vor jeder Änderung** (kein `go mod tidy`: keine Fremdabhängigkeit)

```bash
gofmt -l internal/brain/search
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/search/...
```
Erwartet: keine Ausgabe, Exit 0.
```bash
go test ./internal/brain/search/... -count=1 -cover 2>&1 | grep -E '^ok\s+github\.com/xidus90/loomux/internal/brain/search\s+[0-9.]+s\s+coverage: 100\.0% of statements$'
```
Erwartet: ein Treffer (die Dauer variiert; im Wegwerf-Modul 0,88 s).

- [ ] **Step 5: Stagen, Tor, Commit**

```bash
git add internal/brain/search
```
Erwartet: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0.
```bash
git branch --show-current
```
Erwartet: der Zweig aus Task 0 (`sdd-1b-1` oder `master`).
```bash
git rev-parse --short HEAD
```
Erwartet: der Commit von Task 7.
```bash
git diff --cached --stat
```
Erwartet: genau die 15 Dateien aus Step 2 unter `internal/brain/search/`, alle neu, sonst nichts.
```bash
printf '%s\n' 'Move the qmd ports of ultra-brain into internal/brain/search' > "$TEMP/loomux-task08-msg.txt"
```
Erwartet: keine Ausgabe.
```bash
git commit -F "$TEMP/loomux-task08-msg.txt"
```
Erwartet: Exit 0; der Commit löst das Tor noch einmal aus, das ist gewollt.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität.
```bash
git log -1 --format=%B
```
Erwartet: genau die Zeile `Move the qmd ports of ultra-brain into internal/brain/search`, keine `Co-Authored-By`-Zeile.

---

#### Phase B — die Ports nach Python und Spec

- [ ] **Step 6: Launcher-Tests zuerst** — in `internal/brain/search/launcher_test.go`:

`TestLauncher_NodeIsBatchShim`: nach dem Schreiben des Shims eine Zeile ergänzen (der `node.cmd` liegt jetzt nur noch über `PATH` im Blick, weil neben dem Shim allein `node.exe` zählt).

Vorher:
```go
	shimContent := `"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*`
	if err := os.WriteFile(shimPath, []byte(shimContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := search.ResolveLauncher(shimPath)
	if err == nil {
		t.Fatal("expected error when node is a batch shim, got nil")
	}
```
Nachher:
```go
	shimContent := `"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*`
	if err := os.WriteFile(shimPath, []byte(shimContent), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmpDir)

	_, err := search.ResolveLauncher(shimPath)
	if err == nil {
		t.Fatal("expected error when node is a batch shim, got nil")
	}
```

`TestLauncher_BesideNodeBat`: das Ende ab `_, err := search.ResolveLauncher(shimPath)` ersetzen und einen neuen Test anhängen.

Vorher:
```go
	_, err := search.ResolveLauncher(shimPath)
	if err == nil {
		t.Fatal("expected error when node.bat is beside, got nil")
	}
	if !strings.Contains(err.Error(), "is a batch shim") {
		t.Errorf("expected batch shim refusal error, got: %v", err)
	}
}
```
Nachher:
```go
	t.Setenv("PATH", t.TempDir())
	_, err := search.ResolveLauncher(shimPath)
	if err == nil {
		t.Fatal("expected error when only node.bat is beside, got nil")
	}
	if err.Error() != "cannot find 'node' on PATH to run "+shimPath {
		t.Errorf("a node.bat beside the shim must not count as node, got: %v", err)
	}
}

func TestLauncher_NotFoundNamesTheToolLikePython(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := search.Launcher("nonexistent-tool-xyz")
	if err == nil || err.Error() != "cannot find 'nonexistent-tool-xyz' on PATH" {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 7: Scheitern sehen**

```bash
go test ./internal/brain/search/ -count=1 -run Launcher 2>&1 | grep -E '^--- FAIL|^    launcher_test\.go:[0-9]+: '
```
Erwartet (gemessen): genau vier Treffer, der Reihe nach passend auf diese Muster. Zeilennummern variieren mit dem Kommentar, der Pfad mit dem Temp-Verzeichnis des Tests, die Dauer mit dem Rechner.
```
^--- FAIL: TestLauncher_BesideNodeBat \([0-9.]+s\)$
^    launcher_test\.go:[0-9]+: a node\.bat beside the shim must not count as node, got: .*[\]node\.bat is a batch shim; running it would hand the query to cmd\.exe$
^--- FAIL: TestLauncher_NotFoundNamesTheToolLikePython \([0-9.]+s\)$
^    launcher_test\.go:[0-9]+: got cannot find "nonexistent-tool-xyz" on PATH$
```
Im Wegwerf-Modul lautete die zweite Zeile `    launcher_test.go:160: a node.bat beside the shim must not count as node, got: C:\Users\micro\AppData\Local\Temp\TestLauncher_BesideNodeBat700079152\001\node.bat is a batch shim; running it would hand the query to cmd.exe`.
`TestLauncher_NodeIsBatchShim` besteht vorher und nachher (vorher über `node.cmd` neben dem Shim, nachher über `PATH`).

- [ ] **Step 8: Launcher implementieren** — `internal/brain/search/launcher.go`:

Import ergänzen:
```go
	"github.com/xidus90/loomux/internal/brain/pytext"
```
In `Launcher`, vorher:
```go
		return nil, fmt.Errorf("cannot find %q on PATH", name)
```
Nachher:
```go
		return nil, fmt.Errorf("cannot find %s on PATH", pytext.Repr(name))
```
In `ResolveLauncher`, vorher:
```go
	dir := filepath.Dir(executable)
	besideExe := filepath.Join(dir, "node.exe")
	besideCmd := filepath.Join(dir, "node.cmd")
	besideBat := filepath.Join(dir, "node.bat")
	var node string
	if info, err := os.Stat(besideExe); err == nil && !info.IsDir() {
		node = besideExe
	} else if info, err := os.Stat(besideCmd); err == nil && !info.IsDir() {
		node = besideCmd
	} else if info, err := os.Stat(besideBat); err == nil && !info.IsDir() {
		node = besideBat
	} else {
		nodeLooked, err := exec.LookPath("node")
		if err != nil {
			return nil, fmt.Errorf("cannot find 'node' on PATH to run %s", executable)
		}
		node = nodeLooked
	}
```
Nachher:
```go
	dir := filepath.Dir(executable)
	// qmd.py's launcher looks beside the shim for node.exe only, as the shim itself does, and
	// otherwise takes the node on PATH. A node.cmd or node.bat beside the shim is not a node;
	// one found on PATH is refused below.
	node := filepath.Join(dir, "node.exe")
	if _, err := os.Stat(node); err != nil {
		nodeLooked, err := exec.LookPath("node")
		if err != nil {
			return nil, fmt.Errorf("cannot find 'node' on PATH to run %s", executable)
		}
		node = nodeLooked
	}
```
(`os.Stat` ohne `IsDir`-Prüfung ist `Path.exists()` der Referenz.) Der Rest der Datei bleibt wortgleich, `shimSuffixes` und `npmShimPattern` eingeschlossen.

```bash
git diff --numstat -- internal/brain/search/launcher.go
```
Erwartet (gemessen gegen die ub-Datei): `8	12	internal/brain/search/launcher.go`. Andere Zahlen heißen, dass außer den drei gezeigten Stellen etwas geändert wurde.
```bash
go test ./internal/brain/search/ -count=1 -run Launcher
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/search` mit einer Dauer dahinter.

- [ ] **Step 9: Backbone-Tests zuerst** — in `internal/brain/search/daemon_test.go` den Import `"os"` ergänzen (zwischen `"errors"` und `"reflect"`), vor `func TestStartDaemonWith` einfügen:

```go
// clearBackbone removes the two backbone variables for one test; t.Setenv restores them.
func clearBackbone(t *testing.T) {
	t.Helper()
	for _, key := range []string{"QMD_LLAMA_GPU", "QMD_FORCE_CPU"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStartDaemonWith_TheUsersBackboneWins(t *testing.T) {
	for _, tc := range []struct {
		key, value string
		env        map[string]string
		appended   string
	}{
		{"QMD_LLAMA_GPU", "cuda", map[string]string{"QMD_LLAMA_GPU": "vulkan"}, "QMD_LLAMA_GPU=vulkan"},
		{"QMD_FORCE_CPU", "", map[string]string{"QMD_FORCE_CPU": "1"}, "QMD_FORCE_CPU=1"},
		{"QMD_LLAMA_GPU", "vulkan", map[string]string{"QMD_FORCE_CPU": "1"}, "QMD_FORCE_CPU=1"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			clearBackbone(t)
			t.Setenv(tc.key, tc.value)
			var captured []string
			spawner := func(_ []string, env []string) error {
				captured = env
				return nil
			}
			launcher := func(string) ([]string, error) { return []string{"qmd"}, nil }
			if err := search.StartDaemonWith(tc.env, 9000, launcher, spawner); err != nil {
				t.Fatal(err)
			}
			for _, entry := range captured {
				if entry == tc.appended {
					t.Fatalf("%s was appended over the user's %s=%q", tc.appended, tc.key, tc.value)
				}
			}
		})
	}
}
```
und in `TestStartDaemonWith` als erste Zeile (der Test prüft `QMD_LLAMA_GPU=vulkan` im Ergebnis und darf von der Umgebung des Rechners nicht abhängen):

Vorher:
```go
func TestStartDaemonWith(t *testing.T) {
	var capturedArgv []string
```
Nachher:
```go
func TestStartDaemonWith(t *testing.T) {
	clearBackbone(t)
	var capturedArgv []string
```

- [ ] **Step 10: Scheitern sehen**

```bash
go test ./internal/brain/search/ -count=1 -run StartDaemon 2>&1 | grep -E -e '--- FAIL' -e 'daemon_test\.go:[0-9]+: '
```
Erwartet (gemessen; der leere Wert zählt als gesetzt, auch unter Windows): genau sieben Treffer, der Reihe nach passend auf diese Muster. Zeilennummern variieren mit dem Kommentar, die Dauer mit dem Rechner.
```
^--- FAIL: TestStartDaemonWith_TheUsersBackboneWins \([0-9.]+s\)$
^    --- FAIL: TestStartDaemonWith_TheUsersBackboneWins/QMD_LLAMA_GPU=cuda \([0-9.]+s\)$
^        daemon_test\.go:[0-9]+: QMD_LLAMA_GPU=vulkan was appended over the user's QMD_LLAMA_GPU="cuda"$
^    --- FAIL: TestStartDaemonWith_TheUsersBackboneWins/QMD_FORCE_CPU= \([0-9.]+s\)$
^        daemon_test\.go:[0-9]+: QMD_FORCE_CPU=1 was appended over the user's QMD_FORCE_CPU=""$
^    --- FAIL: TestStartDaemonWith_TheUsersBackboneWins/QMD_LLAMA_GPU=vulkan \([0-9.]+s\)$
^        daemon_test\.go:[0-9]+: QMD_FORCE_CPU=1 was appended over the user's QMD_LLAMA_GPU="vulkan"$
```
Im Wegwerf-Modul stand in allen drei Meldungszeilen `daemon_test.go:106`.

- [ ] **Step 11: Backbone implementieren** — `internal/brain/search/daemon.go`, in `StartDaemonWith`:

Vorher:
```go
// StartDaemonWith starts a detached qmd daemon using the provided launcher and spawner.
func StartDaemonWith(env map[string]string, port int, launcher func(string) ([]string, error), spawner DaemonSpawner) error {
```
```go
	mergedEnv := os.Environ()
	for k, v := range env {
		mergedEnv = append(mergedEnv, fmt.Sprintf("%s=%s", k, v))
	}
	return spawner(argv, mergedEnv)
}
```
Nachher:
```go
// StartDaemonWith starts a detached qmd daemon using the provided launcher and spawner.
// A backbone the user already chose in the environment wins: the port's own backbone
// variables are then not appended (stage 1b-1 spec; qmd_mcp.py overrides the user).
func StartDaemonWith(env map[string]string, port int, launcher func(string) ([]string, error), spawner DaemonSpawner) error {
```
```go
	mergedEnv := os.Environ()
	if !backboneChosenByUser() {
		for k, v := range env {
			mergedEnv = append(mergedEnv, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return spawner(argv, mergedEnv)
}

// backboneChosenByUser reports whether QMD_LLAMA_GPU or QMD_FORCE_CPU is set at all, an
// empty value included: both are the user's say over the backbone.
func backboneChosenByUser() bool {
	_, gpu := os.LookupEnv("QMD_LLAMA_GPU")
	_, cpu := os.LookupEnv("QMD_FORCE_CPU")
	return gpu || cpu
}
```
`DefaultBackbone` bleibt `BackboneCUDA` (Spec und Vertrag; die Zeile „Vorgabe-Backbone Vulkan" in `search.md` ist überholt). Der Rest der Datei bleibt wortgleich.

```bash
git diff --numstat -- internal/brain/search/daemon.go
```
Erwartet (gemessen gegen die ub-Datei): `14	2	internal/brain/search/daemon.go`. Andere Zahlen heißen, dass außer den gezeigten Stellen etwas geändert wurde.
```bash
go test ./internal/brain/search/ -count=1 -run StartDaemon
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/search` mit einer Dauer dahinter.

- [ ] **Step 12: CLI-Port-Tests zuerst** — neue Datei `internal/brain/search/qmd_python_test.go`:

```go
package search_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

func TestQmdPort_StderrIsStrippedLikePython(t *testing.T) {
	port := &search.QmdPort{Runner: func([]string) ([]byte, []byte, int, error) {
		return nil, []byte("\x1c fatal \x1f\n"), 1, nil
	}}
	_, err := port.Indexed("c")
	if err == nil || err.Error() != "qmd exited with 1: fatal" {
		t.Fatalf("got %v", err)
	}
}

func TestQmdPort_IndexedSplitsLinesAndKeepsPathsLikePython(t *testing.T) {
	// "\xc2\x85" is U+0085 NEXT LINE, one of the separators str.splitlines knows.
	output := "1  d  qmd://c/a b.md  \n2  d  qmd://c/x.md\xc2\x853  d  qmd://other/y.md\r\nno uri here\n"
	port := &search.QmdPort{Runner: func([]string) ([]byte, []byte, int, error) {
		return []byte(output), nil, 0, nil
	}}
	got, err := port.Indexed("c")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a b.md  ", "x.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestQmdPort_SearchGoesThroughInvoke(t *testing.T) {
	port := &search.QmdPort{Runner: func([]string) ([]byte, []byte, int, error) {
		return nil, []byte(" broken \n"), 3, nil
	}}
	_, err := port.Search("q", []string{"c"}, search.ProfileFull, 5)
	if err == nil || err.Error() != "qmd exited with 3: broken" {
		t.Fatalf("got %v", err)
	}
}

func TestQmdPort_ALauncherRefusalPassesUnchanged(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	port := &search.QmdPort{Executable: "nonexistent-qmd-xyz"}
	_, err := port.Indexed("c")
	if err == nil || err.Error() != "cannot find 'nonexistent-qmd-xyz' on PATH" {
		t.Fatalf("got %v", err)
	}
}

func TestQmdPort_AStartFailureIsCannotRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.exe"), []byte("echo binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	port := &search.QmdPort{Executable: "broken.exe"}
	_, err := port.NotYetSearchable()
	if err == nil || !strings.HasPrefix(err.Error(), "cannot run broken.exe: ") {
		t.Fatalf("got %v", err)
	}
}

func TestDefaultRunner_ALauncherRefusalKeepsItsWords(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, _, code, err := search.DefaultRunner([]string{"nonexistent-qmd-xyz"})
	if code != 1 || err == nil || err.Error() != "cannot find 'nonexistent-qmd-xyz' on PATH" {
		t.Fatalf("code %d, err %v", code, err)
	}
}

func TestDefaultRunner_AShimIsBypassedNotHandedToCmd(t *testing.T) {
	dir := t.TempDir()
	shim := `"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*`
	if err := os.WriteFile(filepath.Join(dir, "qmd.cmd"), []byte(shim), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node.exe"), []byte("not a program"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	_, _, _, err := search.DefaultRunner([]string{"qmd", "ls", "c"})
	if err == nil || !strings.Contains(err.Error(), filepath.Join(dir, "node.exe")) {
		t.Fatalf("expected the start of the node beside the shim to fail, got %v", err)
	}
}
```
Gemessene Referenzwerte: `"\x1c fatal \x1f\n".strip()` → `'fatal'`; `QmdPort(runner=<stub>).indexed("c")` auf dieselbe Ausgabe → `('a b.md  ', 'x.md')`; `_invoke` mit Exit 1 → `qmd exited with 1: fatal`; `launcher("nonexistent-qmd-xyz")` → `cannot find 'nonexistent-qmd-xyz' on PATH`; ein `OSError` im Runner → `cannot run qmd: [Errno 2] nope`.

- [ ] **Step 13: Scheitern sehen**

```bash
go test ./internal/brain/search/ -count=1 -run 'QmdPort|DefaultRunner' 2>&1 | grep -E '^--- FAIL|^    qmd_python_test\.go:[0-9]+: '
```
Erwartet (gemessen): genau zehn Treffer, der Reihe nach passend auf diese Muster. `qmd_python_test.go` ist neu, die Zeilennummern stehen deshalb fest; die Dauer variiert mit dem Rechner. In der zweiten Zeile stehen die beiden Steuerzeichen U+001C und U+001F roh, das Muster nimmt sie als `.`.
```
^--- FAIL: TestQmdPort_StderrIsStrippedLikePython \([0-9.]+s\)$
^    qmd_python_test\.go:19: got qmd exited with 1: . fatal .$
^--- FAIL: TestQmdPort_IndexedSplitsLinesAndKeepsPathsLikePython \([0-9.]+s\)$
^    qmd_python_test\.go:34: got \["a b\.md" "x\.md[\]u00853  d  qmd://other/y\.md"\], want \["a b\.md  " "x\.md"\]$
^--- FAIL: TestQmdPort_ALauncherRefusalPassesUnchanged \([0-9.]+s\)$
^    qmd_python_test\.go:53: got cannot run nonexistent-qmd-xyz: exec: "nonexistent-qmd-xyz": executable file not found in %PATH%$
^--- FAIL: TestDefaultRunner_ALauncherRefusalKeepsItsWords \([0-9.]+s\)$
^    qmd_python_test\.go:74: code 0, err exec: "nonexistent-qmd-xyz": executable file not found in %PATH%$
^--- FAIL: TestDefaultRunner_AShimIsBypassedNotHandedToCmd \([0-9.]+s\)$
^    qmd_python_test\.go:90: expected the start of the node beside the shim to fail, got <nil>$
```
Im Wegwerf-Modul lautete die vierte Zeile `    qmd_python_test.go:34: got ["a b.md" "x.md\u00853  d  qmd://other/y.md"], want ["a b.md  " "x.md"]`.
`TestQmdPort_SearchGoesThroughInvoke` und `TestQmdPort_AStartFailureIsCannotRun` bestehen schon; sie halten das Verhalten fest, das die Zusammenlegung in Schritt 14 nicht verlieren darf.

- [ ] **Step 14: CLI-Port implementieren** — `internal/brain/search/qmd.go`:

Import ergänzen:
```go
	"github.com/xidus90/loomux/internal/brain/pytext"
```
`DefaultRunner`, vorher:
```go
// DefaultRunner executes argv using os/exec.
func DefaultRunner(argv []string) ([]byte, []byte, int, error) {
	if len(argv) == 0 {
		return nil, nil, 1, errors.New("empty command arguments")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
```
Nachher:
```go
// launcherFailure carries an error of Launcher out of DefaultRunner. qmd.py's _invoke turns
// only an OSError into "cannot run ...", so the refusal its launcher raises reaches the caller
// in its own words; invoke tells the two apart through this type to do the same.
type launcherFailure struct{ err error }

func (f *launcherFailure) Error() string { return f.err.Error() }

// DefaultRunner starts argv through Launcher, so an npm shim never hands the arguments to
// cmd.exe, as qmd.py's _default_runner does. The environment is the caller's own: the default
// backbone CUDA appends nothing, where _default_runner pins QMD_LLAMA_GPU=vulkan.
func DefaultRunner(argv []string) ([]byte, []byte, int, error) {
	if len(argv) == 0 {
		return nil, nil, 1, errors.New("empty command arguments")
	}
	launched, err := Launcher(argv[0])
	if err != nil {
		return nil, nil, 1, &launcherFailure{err: err}
	}
	cmd := exec.Command(launched[0], append(launched[1:], argv[1:]...)...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
```
(der Rest von `DefaultRunner` wortgleich).

`(*QmdPort).Search`, vorher:
```go
	runner := q.getRunner()
	stdout, stderr, exitCode, err := runner(argv)
	if err != nil {
		return nil, fmt.Errorf("cannot run %s: %w", argv[0], err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("%s exited with %d: %s", argv[0], exitCode, strings.TrimSpace(string(stderr)))
	}

	return parseQmdJSON(stdout)
}
```
Nachher:
```go
	stdout, err := q.invoke(argv)
	if err != nil {
		return nil, err
	}
	return parseQmdJSON(stdout)
}
```

`(*QmdPort).invoke`, vorher:
```go
func (q *QmdPort) invoke(argv []string) ([]byte, error) {
	runner := q.getRunner()
	stdout, stderr, exitCode, err := runner(argv)
	if err != nil {
		return nil, fmt.Errorf("cannot run %s: %w", argv[0], err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("%s exited with %d: %s", argv[0], exitCode, strings.TrimSpace(string(stderr)))
	}
	return stdout, nil
}
```
Nachher:
```go
func (q *QmdPort) invoke(argv []string) ([]byte, error) {
	runner := q.getRunner()
	stdout, stderr, exitCode, err := runner(argv)
	var launch *launcherFailure
	if errors.As(err, &launch) {
		return nil, launch.err
	}
	if err != nil {
		return nil, fmt.Errorf("cannot run %s: %w", argv[0], err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("%s exited with %d: %s", argv[0], exitCode, pytext.Strip(string(stderr)))
	}
	return stdout, nil
}
```

`(*QmdPort).Indexed`, vorher:
```go
// Indexed returns every relative path the engine holds for collection.
```
```go
	var found []string
	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		start := strings.Index(line, uriPrefix)
		if start < 0 {
			continue
		}
		rest := line[start+len(uriPrefix):]
		col, rel, _ := strings.Cut(rest, "/")
		if col == collection {
			found = append(found, strings.TrimSpace(rel))
		}
	}
	return found, nil
```
Nachher:
```go
// Indexed returns every relative path the engine holds for collection. Lines and paths are
// taken as qmd.py takes them: split like str.splitlines, the path kept as written.
```
```go
	var found []string
	for _, line := range pytext.SplitLines(string(stdout)) {
		start := strings.Index(line, uriPrefix)
		if start < 0 {
			continue
		}
		rest := line[start+len(uriPrefix):]
		col, rel, _ := strings.Cut(rest, "/")
		if col == collection {
			found = append(found, rel)
		}
	}
	return found, nil
```
`pendingLinePattern`, `parseQmdJSON`, `Refresh`, `NotYetSearchable`, `Embed` bleiben wortgleich.

```bash
git diff --numstat -- internal/brain/search/qmd.go
```
Erwartet (gemessen gegen die ub-Datei): `29	15	internal/brain/search/qmd.go`. Andere Zahlen heißen, dass außer den gezeigten Stellen etwas geändert wurde.
```bash
go test ./internal/brain/search/ -count=1 -run 'QmdPort|DefaultRunner'
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/search` mit einer Dauer dahinter (die bestehenden `TestDefaultRunner`-Fälle mit `go version` laufen jetzt über `Launcher("go")`).

- [ ] **Step 15: Aufwärm-Hinweis, Tests zuerst** — in `internal/brain/search/mcp_test.go` den Teil ab `func TestDefaultConnectWith_AlreadyReachable` bis zum Dateiende ersetzen.

Vorher (ub `mcp_test.go:512-568`, vollständig bis zum Dateiende):
```go
func TestDefaultConnectWith_AlreadyReachable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]any{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	_, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	connFn := search.DefaultConnectWith(port, nil, nil, time.Second)
	sess, err := connFn(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
}

func TestDefaultConnectWith_DaemonStartFails(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return nil, errors.New("cannot launch qmd")
	}
	connFn := search.DefaultConnectWith(64999, mockLauncher, nil, time.Millisecond)
	_, err := connFn(nil)
	if err == nil || !strings.Contains(err.Error(), "cannot launch qmd") {
		t.Errorf("expected launch error, got: %v", err)
	}
}

func TestDefaultConnectWith_WaitTimeout(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return []string{"qmd"}, nil
	}
	mockSpawner := func(argv []string, env []string) error {
		return nil
	}
	connFn := search.DefaultConnectWith(64998, mockLauncher, mockSpawner, 5*time.Millisecond)
	_, err := connFn(nil)
	if err == nil || !strings.Contains(err.Error(), "no qmd daemon answered") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestNewQmdMcpPort_Defaults(t *testing.T) {
	port := search.NewQmdMcpPort()
	if port == nil {
		t.Fatal("expected non-nil port")
	}
	_ = search.DefaultConnect(8765)
}
```
Nachher (vollständig ab `func TestDefaultConnectWith_AlreadyReachable`):
```go
func TestDefaultConnectWith_AlreadyReachable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]any{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	_, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	var heard []string
	connFn := search.DefaultConnectWith(port, nil, nil, time.Second, func(m string) { heard = append(heard, m) })
	sess, err := connFn(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
	if len(heard) != 0 {
		t.Errorf("a daemon that answers must not be announced as starting: %v", heard)
	}
}

func TestDefaultConnectWith_DaemonStartFails(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return nil, errors.New("cannot launch qmd")
	}
	var heard []string
	connFn := search.DefaultConnectWith(64999, mockLauncher, nil, time.Millisecond, func(m string) { heard = append(heard, m) })
	_, err := connFn(nil)
	if err == nil || !strings.Contains(err.Error(), "cannot launch qmd") {
		t.Errorf("expected launch error, got: %v", err)
	}
	if len(heard) != 0 {
		t.Errorf("a start that failed must not be announced: %v", heard)
	}
}

func TestDefaultConnectWith_WaitTimeout(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return []string{"qmd"}, nil
	}
	mockSpawner := func(argv []string, env []string) error {
		return nil
	}
	connFn := search.DefaultConnectWith(64998, mockLauncher, mockSpawner, 5*time.Millisecond, nil)
	_, err := connFn(nil)
	if err == nil || !strings.Contains(err.Error(), "no qmd daemon answered") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestDefaultConnectWith_AnnouncesAStartOncePerPort(t *testing.T) {
	var heard []string
	spawned := 0
	connect := search.DefaultConnectWith(64997,
		func(string) ([]string, error) { return []string{"qmd"}, nil },
		func([]string, []string) error { spawned++; return nil },
		5*time.Millisecond,
		func(message string) { heard = append(heard, message) })
	port := search.NewQmdMcpPort(search.WithConnect(connect), search.WithColdAttempts(3))
	_, err := port.Search("q", []string{"c"}, search.ProfileFast, 1)
	if err == nil || !strings.HasPrefix(err.Error(), "the search engine did not answer in 3 attempts: no qmd daemon answered on http://localhost:64997/mcp within 5ms") {
		t.Fatalf("got %v", err)
	}
	if spawned != 3 {
		t.Errorf("expected three starts, got %d", spawned)
	}
	if len(heard) != 1 || heard[0] != search.WarmingNotice {
		t.Errorf("expected the warming notice exactly once, got %q", heard)
	}
}

func TestWarmingNoticeIsTheDaemonsWording(t *testing.T) {
	const want = "starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm."
	if search.WarmingNotice != want {
		t.Fatalf("got %q", search.WarmingNotice)
	}
}

func TestNewQmdMcpPort_DefaultConnectRefusesAMissingQmdWithoutANotice(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var heard []string
	port := search.NewQmdMcpPort(search.WithPort(64996), search.WithColdAttempts(1),
		search.WithNotice(func(m string) { heard = append(heard, m) }))
	_, err := port.Search("q", []string{"c"}, search.ProfileFast, 1)
	if err == nil || err.Error() != "the search engine did not answer in 1 attempts: cannot find 'qmd' on PATH" {
		t.Fatalf("got %v", err)
	}
	if len(heard) != 0 {
		t.Errorf("no daemon was started, yet the notice came: %q", heard)
	}
}

func TestNewQmdMcpPort_Defaults(t *testing.T) {
	port := search.NewQmdMcpPort()
	if port == nil {
		t.Fatal("expected non-nil port")
	}
	_ = search.DefaultConnect(8765, nil)
}
```
Neue Datei `internal/brain/search/notice_internal_test.go` (die Verdrahtung `WithNotice` → `DefaultConnect` ist von außen nur mit einem echten Daemon-Start zu sehen; kein Test startet qmd):
```go
package search

import "testing"

func TestNewQmdMcpPortHandsItsNoticeToTheDefaultConnect(t *testing.T) {
	gotPort := 0
	var gotNotice func(string)
	connectDefault = func(port int, notice func(string)) ConnectFunc {
		gotPort, gotNotice = port, notice
		return nil
	}
	defer func() { connectDefault = DefaultConnect }()

	var heard []string
	NewQmdMcpPort(WithPort(9001), WithNotice(func(message string) { heard = append(heard, message) }))
	if gotPort != 9001 || gotNotice == nil {
		t.Fatalf("default connect got port %d and notice %v", gotPort, gotNotice != nil)
	}
	gotNotice("heard")
	if len(heard) != 1 || heard[0] != "heard" {
		t.Fatalf("the notice handed on is not the one given: %q", heard)
	}
}
```
Gemessen: `brain.daemon.server.WARMING_NOTICE` → `'starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.'`.

- [ ] **Step 16: Scheitern sehen**

```bash
go vet ./internal/brain/search/
```
Erwartet (gemessen), Exit 1 und genau:
```
# github.com/xidus90/loomux/internal/brain/search [github.com/xidus90/loomux/internal/brain/search.test]
internal\brain\search\notice_internal_test.go:8:2: undefined: connectDefault
internal\brain\search\notice_internal_test.go:12:17: undefined: connectDefault
internal\brain\search\notice_internal_test.go:15:32: undefined: WithNotice
```

- [ ] **Step 17: Aufwärm-Hinweis implementieren** — `internal/brain/search/mcp.go`:

Nach dem `const`-Block mit `ColdAttempts` einfügen:
```go
// WarmingNotice is the brain daemon's word for a search that had to start the engine
// (daemon/server.py). The Python command line never says it; loomux says it once per port.
const WarmingNotice = "starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm."
```
Nach `WithPort` einfügen:
```go
// WithNotice hands the default connect a function that hears WarmingNotice when the port had
// to start the daemon. It has no effect beside WithConnect.
func WithNotice(fn func(message string)) QmdMcpOption {
	return func(p *QmdMcpPort) {
		p.notice = fn
	}
}
```
Im `QmdMcpPort`-Struct, vorher:
```go
	attempts int

	session Session
```
Nachher:
```go
	attempts int
	notice   func(message string)

	session Session
```
`DefaultConnectWith` und `DefaultConnect`, vorher:
```go
// DefaultConnectWith returns a ConnectFunc using the given launcher, spawner, and timeout.
func DefaultConnectWith(port int, launcher func(string) ([]string, error), spawner DaemonSpawner, waitTimeout time.Duration) ConnectFunc {
	return func(env map[string]string) (Session, error) {
		session := NewHTTPSession(port)
		if !session.Reachable() {
			if err := StartDaemonWith(env, port, launcher, spawner); err != nil {
				return nil, err
			}
			if err := session.WaitUntilReachable(waitTimeout); err != nil {
				return nil, err
			}
		}
		return session, nil
	}
}

// DefaultConnect returns a ConnectFunc that connects to a local daemon on port, starting one if needed.
func DefaultConnect(port int) ConnectFunc {
	return DefaultConnectWith(port, Launcher, DefaultSpawner, 60*time.Second)
}
```
Nachher:
```go
// DefaultConnectWith returns a ConnectFunc using the given launcher, spawner, and timeout.
// notice, when not nil, hears WarmingNotice at most once for the returned ConnectFunc: right
// after a daemon start succeeded, before the wait for it.
func DefaultConnectWith(port int, launcher func(string) ([]string, error), spawner DaemonSpawner, waitTimeout time.Duration, notice func(string)) ConnectFunc {
	var once sync.Once
	return func(env map[string]string) (Session, error) {
		session := NewHTTPSession(port)
		if !session.Reachable() {
			if err := StartDaemonWith(env, port, launcher, spawner); err != nil {
				return nil, err
			}
			if notice != nil {
				once.Do(func() { notice(WarmingNotice) })
			}
			if err := session.WaitUntilReachable(waitTimeout); err != nil {
				return nil, err
			}
		}
		return session, nil
	}
}

// DefaultConnect returns a ConnectFunc that connects to a local daemon on port, starting one if needed.
func DefaultConnect(port int, notice func(string)) ConnectFunc {
	return DefaultConnectWith(port, Launcher, DefaultSpawner, 60*time.Second, notice)
}

// connectDefault is the connect NewQmdMcpPort falls back to; a test replaces it to see what
// the port hands on without starting a daemon.
var connectDefault = DefaultConnect
```
In `NewQmdMcpPort`, vorher:
```go
		p.connect = DefaultConnect(p.port)
```
Nachher:
```go
		p.connect = connectDefault(p.port, p.notice)
```
Alles Übrige in `mcp.go` (`ask`, `letGo`, `formatArguments`, `splitPath`) bleibt wortgleich; `translateReply` ändert Schritt 17c.

```bash
git diff --numstat -- internal/brain/search/mcp.go
```
Erwartet (gemessen gegen die ub-Datei): `27	4	internal/brain/search/mcp.go`. Andere Zahlen heißen, dass außer den gezeigten Stellen etwas geändert wurde.
```bash
go test ./internal/brain/search/ -count=1 -run 'DefaultConnectWith|WarmingNotice|NewQmdMcpPort'
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/search` mit einer Dauer dahinter (gemessen: acht Tests, `TestNewQmdMcpPortHandsItsNoticeToTheDefaultConnect` eingeschlossen).

- [ ] **Step 17a: Zeilennummern des Daemons, Tests zuerst** — ans Ende von `internal/brain/search/mcp_test.go`, hinter `TestNewQmdMcpPort_Defaults`, anhängen.

Warum: qmd 2.8.3 nummeriert auf dem MCP-Weg jede Snippet-Zeile (`dist/mcp/server.js:301` `snippet: addLineNumbers(snippet, line)`; `dist/cli/formatter.js:17-20` teilt an `'\n'` und setzt `${startLine + i}: ` davor). `qmd query --json`, das die Python-Referenz liest (`qmd.py:152`, ohne `--line-numbers`), nummeriert nicht, und `qmd_mcp.py:_translate` reicht den Snippet unverändert weiter. Ohne diesen Schritt druckte `loomux brain search` unter einem Treffer mit `line` 12 `    12: @@ -11,4 @@ (10 before, 5 after)`, wo die Referenz `    @@ -11,4 @@ (10 before, 5 after)` druckt, und das in jedem Profil. Der Test geht über einen `httptest`-Server, damit `line` wie vom echten Daemon als JSON-Zahl (`float64`) ankommt; jeder Fall läuft in allen drei Profilen.

```go
// qmd's daemon numbers every snippet line of a query answer (dist/mcp/server.js:301,
// addLineNumbers(snippet, line)); `qmd query --json`, which the Python reference reads, does
// not. The port takes the numbers off again, and only numbers that are exactly the daemon's.
func TestQmdMcpPort_TakesTheDaemonsLineNumbersOffTheSnippet(t *testing.T) {
	for _, tc := range []struct {
		name, fields, want string
	}{
		{"numbered from the hit's line", `"line":12,"snippet":"12: @@ -11,4 @@ (10 before, 5 after)\n13: a\n14: b\r\n15: c"`, "@@ -11,4 @@ (10 before, 5 after)\na\nb\r\nc"},
		{"an empty snippet", `"line":7,"snippet":"7: "`, ""},
		{"numbers that are not the hit's line", `"line":12,"snippet":"5: a\n6: b"`, "5: a\n6: b"},
		{"no line in the reply", `"snippet":"1: a"`, "1: a"},
		{"one part without its number", `"line":12,"snippet":"12: a\nb"`, "12: a\nb"},
		{"a line below one counts from itself", `"line":0,"snippet":"0: a\n1: b"`, "a\nb"},
	} {
		for _, profile := range []search.Profile{search.ProfileKeyword, search.ProfileFast, search.ProfileFull} {
			t.Run(tc.name+"/"+string(profile), func(t *testing.T) {
				reply := `{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"results":[{"docid":"#abc123","file":"c/a.md","title":"A","score":0.5,` + tc.fields + `}]}}}`
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(reply))
				}))
				defer ts.Close()
				port := search.NewQmdMcpPort(search.WithConnect(func(map[string]string) (search.Session, error) {
					return &search.HTTPSession{URL: ts.URL}, nil
				}))
				hits, err := port.Search("q", []string{"c"}, profile, 1)
				if err != nil {
					t.Fatal(err)
				}
				if len(hits) != 1 {
					t.Fatalf("got %d hits", len(hits))
				}
				if hits[0].Snippet != tc.want {
					t.Fatalf("snippet %q, want %q", hits[0].Snippet, tc.want)
				}
			})
		}
	}
}
```
Die vier Fälle aus dem Befund sind die ersten vier. „one part without its number" hält fest, dass nur ein vollständig nummerierter Snippet entkleidet wird; „a line below one counts from itself" hält fest, dass die Nummer aus dem rohen `line` vor der Klammerung auf 1 kommt. Die Importe von `mcp_test.go` (`net/http`, `net/http/httptest`) sind schon da.

- [ ] **Step 17b: Scheitern sehen**

```bash
go test ./internal/brain/search/ -count=1 -run TakesTheDaemonsLineNumbers 2>&1 | grep -E -e '--- FAIL' -e 'mcp_test\.go:[0-9]+: '
```
Erwartet (gemessen): genau 19 Treffer, jeder passend auf eines dieser Muster. Die erste Zeile ist die des Tests, danach je fehlschlagendem Untertest eine `--- FAIL`-Zeile und ihre Meldung, in der Reihenfolge `numbered_from_the_hit's_line`, `an_empty_snippet`, `a_line_below_one_counts_from_itself`, jeweils `keyword`, `fast`, `full`. Zeilennummern variieren mit dem Kommentar, die Dauer mit dem Rechner.
```
^--- FAIL: TestQmdMcpPort_TakesTheDaemonsLineNumbersOffTheSnippet \([0-9.]+s\)$
^    --- FAIL: TestQmdMcpPort_TakesTheDaemonsLineNumbersOffTheSnippet/(numbered_from_the_hit's_line|an_empty_snippet|a_line_below_one_counts_from_itself)/(keyword|fast|full) \([0-9.]+s\)$
^        mcp_test\.go:[0-9]+: snippet "12: @@ -11,4 @@ \(10 before, 5 after\)[\]n13: a[\]n14: b[\]r[\]n15: c", want "@@ -11,4 @@ \(10 before, 5 after\)[\]na[\]nb[\]r[\]nc"$
^        mcp_test\.go:[0-9]+: snippet "7: ", want ""$
^        mcp_test\.go:[0-9]+: snippet "0: a[\]n1: b", want "a[\]nb"$
```
Im Wegwerf-Modul stand in allen neun Meldungen `mcp_test.go:652`. Die drei übrigen Fälle bestehen vorher und nachher.

- [ ] **Step 17c: Zeilennummern des Daemons implementieren** — `internal/brain/search/mcp.go`:

Importe, vorher:
```go
import (
	"fmt"
	"strings"
	"sync"
	"time"
)
```
Nachher:
```go
import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)
```
In `translateReply`, vorher:
```go
		line := 1
		if lineVal, ok := resMap["line"]; ok && lineVal != nil {
			switch v := lineVal.(type) {
			case int:
				line = v
			case float64:
				line = int(v)
			}
			if line < 1 {
				line = 1
			}
		}

		title, _ := resMap["title"].(string)
		snippet, _ := resMap["snippet"].(string)
```
Nachher:
```go
		line := 1
		rawLine, numbered := 0, false
		if lineVal, ok := resMap["line"]; ok && lineVal != nil {
			switch v := lineVal.(type) {
			case int:
				line = v
				rawLine, numbered = v, true
			case float64:
				line = int(v)
				rawLine, numbered = int(v), true
			}
			if line < 1 {
				line = 1
			}
		}

		title, _ := resMap["title"].(string)
		snippet, _ := resMap["snippet"].(string)
		if numbered {
			snippet = withoutLineNumbers(snippet, rawLine)
		}
```
Vor `func splitPath` einfügen:
```go
// withoutLineNumbers takes off the numbers qmd's daemon puts in front of every snippet line of a
// query answer (addLineNumbers in mcp/server.js: each "\n"-separated part i begins with
// "<line+i>: "). qmd query --json, the output the Python reference reads, has none. A snippet
// in which a single part lacks its own number is handed on as it came.
func withoutLineNumbers(snippet string, line int) string {
	parts := strings.Split(snippet, "\n")
	for i, part := range parts {
		prefix := strconv.Itoa(line+i) + ": "
		if !strings.HasPrefix(part, prefix) {
			return snippet
		}
		parts[i] = part[len(prefix):]
	}
	return strings.Join(parts, "\n")
}
```
`strings.Split` statt `pytext.SplitLines`: `addLineNumbers` teilt nur an `'\n'`, ein `\r` bleibt am Teil (Fall 1). Die übrigen Zeilen von `translateReply` bleiben wortgleich.

```bash
git diff --numstat -- internal/brain/search/mcp.go
```
Erwartet (gemessen gegen die ub-Datei): `50	4	internal/brain/search/mcp.go`.
```bash
go test ./internal/brain/search/ -count=1 -run 'QmdMcpPort'
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/search` mit einer Dauer dahinter.

- [ ] **Step 18: Bestehen sehen, Coverage, Stagen, Tor, Commit**

```bash
gofmt -l internal/brain/search
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/search/...
```
Erwartet: keine Ausgabe, Exit 0.
```bash
go test ./internal/brain/search/... -count=1 -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Erwartet: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/brain/search` beginnt (im Wegwerf-Modul 1,7 s; der Test mit drei Startversuchen gegen `localhost:64997` fällt unter Windows nicht ins Gewicht).
```bash
go tool cover -func="$TEMP/pkg.out" | grep 'internal/brain/search/' | grep -v '100.0%'
```
Erwartet: keine Ausgabe, also jede Funktion des Pakets 100.0% (im Wegwerf-Modul gemessen, `translateReply` und `withoutLineNumbers` eingeschlossen). Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.
```bash
git add internal/brain/search/launcher.go internal/brain/search/launcher_test.go internal/brain/search/daemon.go internal/brain/search/daemon_test.go internal/brain/search/qmd.go internal/brain/search/qmd_python_test.go internal/brain/search/mcp.go internal/brain/search/mcp_test.go internal/brain/search/notice_internal_test.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0.
```bash
git branch --show-current
```
Erwartet: derselbe Zweig wie in Step 5.
```bash
git rev-parse --short HEAD
```
Erwartet: der Commit aus Step 5.
```bash
git diff --cached --stat
```
Erwartet: genau diese neun Dateien, zwei davon neu (`qmd_python_test.go`, `notice_internal_test.go`), sonst nichts:
```
internal/brain/search/daemon.go
internal/brain/search/daemon_test.go
internal/brain/search/launcher.go
internal/brain/search/launcher_test.go
internal/brain/search/mcp.go
internal/brain/search/mcp_test.go
internal/brain/search/notice_internal_test.go
internal/brain/search/qmd.go
internal/brain/search/qmd_python_test.go
```
```bash
printf '%s\n' 'Start qmd like the Python reference, announce a daemon start, strip line numbers' > "$TEMP/loomux-task08-msg.txt"
```
Erwartet: keine Ausgabe.
```bash
git commit -F "$TEMP/loomux-task08-msg.txt"
```
Erwartet: Exit 0; der Commit löst das Tor noch einmal aus, das ist gewollt.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität.
```bash
git log -1 --format=%B
```
Erwartet: genau die Zeile `Start qmd like the Python reference, announce a daemon start, strip line numbers`, keine `Co-Authored-By`-Zeile.

---

#### Phase C — der Reconcile-Stempel

- [ ] **Step 19: Tests zuerst** — neue Datei `internal/brain/search/stamp_test.go`:

```go
package search_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
)

// stampWorld writes content as the reconcile stamp of a fresh state directory.
func stampWorld(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "maintenance"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "maintenance", "last-run.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestReconcileConstantsAreThePythonOnes(t *testing.T) {
	if search.ReconcileInterval != 24*time.Hour {
		t.Errorf("interval %v", search.ReconcileInterval)
	}
	if search.ReconcileAdvice != "run `brain reconcile`" {
		t.Errorf("advice %q", search.ReconcileAdvice)
	}
}

// Expected values measured against the reference on 2026-09-15:
// datetime.fromisoformat(text.strip()) and .isoformat(), None for naive stamps.
func TestReadLastRunReadsWhatPythonReads(t *testing.T) {
	for _, tc := range []struct {
		name, content, iso string
		ok                 bool
	}{
		{"writer form", "2026-09-13T08:00:00+00:00\n", "2026-09-13T08:00:00+00:00", true},
		{"padded with microseconds and an offset", "  2026-09-13T08:00:00.123456+02:00\r\n", "2026-09-13T08:00:00.123456+02:00", true},
		{"zulu", "2026-09-13T08:00:00Z", "2026-09-13T08:00:00+00:00", true},
		{"three fraction digits", "2026-09-13T08:00:00.500+00:00", "2026-09-13T08:00:00.500000+00:00", true},
		{"naive", "2026-09-13T08:00:00\n", "", false},
		{"garbage", "garbage", "", false},
		{"byte order mark", "\xef\xbb\xbf2026-09-13T08:00:00+00:00", "", false},
		{"not utf-8", "\xff2026-09-13T08:00:00+00:00", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stamp, ok, err := search.ReadLastRun(stampWorld(t, tc.content))
			if err != nil {
				t.Fatal(err)
			}
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if ok && pytext.IsoFormat(stamp) != tc.iso {
				t.Fatalf("isoformat %q, want %q", pytext.IsoFormat(stamp), tc.iso)
			}
			if !ok && !stamp.IsZero() {
				t.Fatalf("no stamp must come back zero, got %v", stamp)
			}
		})
	}
}

func TestReadLastRunWithoutAStampIsNoStamp(t *testing.T) {
	stamp, ok, err := search.ReadLastRun(t.TempDir())
	if err != nil || ok || !stamp.IsZero() {
		t.Fatalf("got %v %v %v", stamp, ok, err)
	}
}

func TestReadLastRunOfAStampThatCannotBeReadIsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "maintenance", "last-run.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := search.ReadLastRun(dir); err == nil || ok {
		t.Fatalf("a directory in the stamp's place must be an error, got ok=%v err=%v", ok, err)
	}
}

// Measured: a difference of exactly 24 h is stale, 24 h less one microsecond is not.
func TestStaleCountsTheFullDay(t *testing.T) {
	stamp := time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC)
	if !search.Stale(stamp, stamp.Add(24*time.Hour)) {
		t.Error("exactly one day must be stale")
	}
	if search.Stale(stamp, stamp.Add(24*time.Hour-time.Microsecond)) {
		t.Error("one microsecond short of a day must not be stale")
	}
	if search.Stale(stamp, stamp.Add(-time.Hour)) {
		t.Error("a stamp from the future must not be stale")
	}
}

func TestStaleReconcileNamesAnAgedStampOnly(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		dir  string
		want []string
	}{
		{"aged", stampWorld(t, "2000-01-01T00:00:00+00:00\n"), []string{
			"the last full reconciliation was 2000-01-01T00:00:00+00:00, more than 24 hours ago: a source may have changed without this answer knowing (run `brain reconcile`)",
		}},
		{"fresh", stampWorld(t, "2999-01-01T00:00:00+00:00\n"), nil},
		{"naive and aged", stampWorld(t, "2000-01-01T00:00:00\n"), nil},
		{"missing", t.TempDir(), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := search.StaleReconcile(tc.dir, now)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStaleReconcileHandsAReadErrorOn(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "maintenance", "last-run.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := search.StaleReconcile(dir, time.Now()); err == nil || got != nil {
		t.Fatalf("got %q, %v", got, err)
	}
}
```
Gemessene Referenzwerte:

| Eingabe (nach `strip()`) | `fromisoformat` → `isoformat()` | zonenbehaftet |
|---|---|---|
| `2026-09-13T08:00:00+00:00` | `2026-09-13T08:00:00+00:00` | ja |
| `2026-09-13T08:00:00.123456+02:00` | `2026-09-13T08:00:00.123456+02:00` | ja |
| `2026-09-13T08:00:00Z` | `2026-09-13T08:00:00+00:00` | ja |
| `2026-09-13T08:00:00.500+00:00` | `2026-09-13T08:00:00.500000+00:00` | ja |
| `2026-09-13T08:00:00` | `2026-09-13T08:00:00` | nein → `None` |
| `garbage` | `ValueError` → `None` | — |
| BOM + `2026-09-13T08:00:00+00:00` | `ValueError` → `None` | — |

`issubclass(UnicodeDecodeError, ValueError)` → `True` (kein UTF-8 ist also `None`). `a - b >= RECONCILE_INTERVAL` für genau 24 h → `True`, für 24 h minus 1 µs → `False`. `stale_reconcile` mit Stempel `2000-01-01T00:00:00+00:00` → der Befund aus dem Test; mit `2999-01-01T00:00:00+00:00` und mit `None` → `()`.

- [ ] **Step 20: Scheitern sehen**

```bash
go vet ./internal/brain/search/
```
Erwartet (gemessen; `go vet` meldet nur den ersten Typfehler der Testdatei), Exit 1 und genau:
```
# github.com/xidus90/loomux/internal/brain/search_test
# [github.com/xidus90/loomux/internal/brain/search_test]
vet.exe: internal\brain\search\stamp_test.go:28:12: undefined: search.ReconcileInterval
```
Die Zeilennummer steht fest, weil `stamp_test.go` neu ist.

- [ ] **Step 21: Implementieren** — neue Datei `internal/brain/search/stamp.go` (neuer Code, keine Regexe, kein `init()`):

```go
package search

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// ReconcileInterval is core.RECONCILE_INTERVAL: a full reconciliation older than one day is
// reported by search and by status alike.
const ReconcileInterval = 24 * time.Hour

// ReconcileAdvice is core._RECONCILE_ADVICE; the backticks belong to the text. Until stage 3
// it names the Python command the user has.
const ReconcileAdvice = "run `brain reconcile`"

// ReadLastRun reads <stateDir>/maintenance/last-run.txt as reconcile.read_last_run does. A
// stamp that is missing, not UTF-8, unreadable as an ISO time or without a zone is no stamp
// (false, nil): Python answers None for all four, UnicodeDecodeError being a ValueError there.
// Only a stamp that exists and cannot be read is an error. Existence is Path.exists, which
// swallows every stat error.
func ReadLastRun(stateDir string) (time.Time, bool, error) {
	path := filepath.Join(stateDir, "maintenance", "last-run.txt")
	if _, err := os.Stat(path); err != nil {
		return time.Time{}, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, false, err
	}
	if !utf8.Valid(data) {
		return time.Time{}, false, nil
	}
	stamp, ok := pytext.ParseAwareIsoFormat(pytext.Strip(string(data)))
	if !ok {
		return time.Time{}, false, nil
	}
	return stamp, true, nil
}

// Stale is core._stale with the clock handed in: a full day or more, the day itself included.
func Stale(stamp, now time.Time) bool {
	return now.Sub(stamp) >= ReconcileInterval
}

// StaleReconcile is core.stale_reconcile: the one finding a search carries for a stamp that
// exists and has aged, never for a missing one.
func StaleReconcile(stateDir string, now time.Time) ([]string, error) {
	stamp, ok, err := ReadLastRun(stateDir)
	if err != nil {
		return nil, err
	}
	if !ok || !Stale(stamp, now) {
		return nil, nil
	}
	return []string{fmt.Sprintf(
		"the last full reconciliation was %s, more than 24 hours ago: "+
			"a source may have changed without this answer knowing (%s)",
		pytext.IsoFormat(stamp), ReconcileAdvice,
	)}, nil
}
```
Warum `os.ReadFile` + `utf8.Valid` statt `pytext.ReadText`: `ReadText` meldet ungültiges UTF-8 als `fmt.Errorf("%s: not valid UTF-8", path)` ohne Sentinel; „kein UTF-8 → false, nil" und „anderer Lesefehler → error" wären nur am Wortlaut zu trennen. Der Stempel ist eine Zeile; universelle Zeilenenden ändern nach `Strip` nichts (ein inneres `\r` ist auf beiden Seiten unlesbar).

```bash
go test ./internal/brain/search/ -count=1 -run 'ReadLastRun|Stale|Reconcile'
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/brain/search` mit einer Dauer dahinter.

- [ ] **Step 21a: Stagen, Tor, Commit** (der Stempel ist ein eigener Commit: ab Step 22 kompiliert das Paket bis Step 25 nicht)

```bash
gofmt -l internal/brain/search
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/search/...
```
Erwartet: keine Ausgabe, Exit 0.
```bash
go test ./internal/brain/search/... -count=1 -covermode=set -coverprofile="$TEMP/pkg.out"
```
Erwartet (im Wegwerf-Modul gemessen): Exit 0 und eine Zeile, die auf `grep -E '^ok\s+github\.com/xidus90/loomux/internal/brain/search\s+[0-9.]+s\s+coverage: 100\.0% of statements$'` passt.
```bash
go tool cover -func="$TEMP/pkg.out" | grep 'internal/brain/search/' | grep -v '100.0%'
```
Erwartet: keine Ausgabe, also jede Funktion des Pakets 100.0%, `ReadLastRun`, `Stale` und `StaleReconcile` eingeschlossen. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.
```bash
git add internal/brain/search/stamp.go internal/brain/search/stamp_test.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0.
```bash
git branch --show-current
```
Erwartet: derselbe Zweig wie in Step 5.
```bash
git rev-parse --short HEAD
```
Erwartet: der Commit aus Step 18.
```bash
git diff --cached --stat
```
Erwartet: genau `internal/brain/search/stamp.go` und `internal/brain/search/stamp_test.go`, beide neu, sonst nichts.
```bash
printf '%s\n' 'Read the reconcile stamp like core.stale_reconcile' > "$TEMP/loomux-task08-msg.txt"
```
Erwartet: keine Ausgabe.
```bash
git commit -F "$TEMP/loomux-task08-msg.txt"
```
Erwartet: Exit 0; der Commit löst das Tor noch einmal aus, das ist gewollt.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität.
```bash
git log -1 --format=%B
```
Erwartet: genau die Zeile `Read the reconcile stamp like core.stale_reconcile`, keine `Co-Authored-By`-Zeile.

---

#### Phase D — die Suche nach `core.search`

- [ ] **Step 22: `search.go` und seine Tests kopieren und Importe umschreiben**

```bash
cp "$SRC"/pkg/search/search.go "$SRC"/pkg/search/search_test.go "$SRC"/pkg/search/visibility_test.go "$LOOMUX/internal/brain/search/"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rl 'github.com/xidus90/ultra-brain/' "$LOOMUX/internal/brain/search" --include=*.go | xargs -r sed -i 's#github.com/xidus90/ultra-brain/pkg/search#github.com/xidus90/loomux/internal/brain/search#g; s#github.com/xidus90/ultra-brain/pkg/privacy#github.com/xidus90/loomux/internal/brain/privacy#g; s#github.com/xidus90/ultra-brain/pkg/config#github.com/xidus90/loomux/internal/config#g; s#github.com/xidus90/ultra-brain/internal/testlock#github.com/xidus90/loomux/internal/testlock#g'
```
Erwartet: keine Ausgabe, Exit 0.
```bash
grep -rn 'github.com/xidus90/ultra-brain' internal/brain/search
```
Erwartet: keine Ausgabe, Exit 1.
```bash
go vet ./internal/brain/search/
```
Erwartet (gemessen mit einem `privacy`-Stub in der Signatur von Task 3), Exit 1 und genau:
```
# github.com/xidus90/loomux/internal/brain/search [github.com/xidus90/loomux/internal/brain/search.test]
internal\brain\search\search.go:80:24: assignment mismatch: 2 variables but privacy.VisibleManifest returns 3 values
```
Das ist der Grund, warum Schritt 4 des Umzugsverfahrens für diese drei Dateien nicht grün werden kann: ub-`ExecuteSearch` stützt sich auf „fehlendes Manifest = sichtbar" und „kaputtes Manifest = still unsichtbar"; beides verbietet Task 3 nach `core._visible_areas`.

- [ ] **Step 23: Tests nach Python neu fassen** — `internal/brain/search/search_test.go` vollständig ersetzen durch:

```go
package search_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
)

type mockSearchPort struct {
	searchFunc func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error)
	calls      int
}

func (m *mockSearchPort) Search(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
	m.calls++
	if m.searchFunc != nil {
		return m.searchFunc(query, collections, profile, n)
	}
	return nil, nil
}

func (m *mockSearchPort) Indexed(collection string) ([]string, error) { return nil, nil }
func (m *mockSearchPort) Refresh(collections []string) error          { return nil }
func (m *mockSearchPort) NotYetSearchable() (int, error)              { return 0, nil }
func (m *mockSearchPort) Embed(collections []string) error            { return nil }

func writeRegistry(t *testing.T, dir string, content string) {
	t.Helper()
	path := filepath.Join(dir, "registry.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}
}

// searchNow is the clock of every search test. A world carries a stamp only where a test
// writes one, and then it is judged against this time.
var searchNow = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

// Measured against core._ask and core._assemble on 2026-09-15.
const (
	twiceEmptyFast = "the search engine answered empty twice in a row on profile fast; an empty answer to a meaning search is practically unreachable, so this is more likely a silent failure of the engine than an absence of matches (spec 16.14)"
	withheldOne    = "1 of the engine's hits were withheld -- excluded by [privacy] never, or from a collection this channel has no area for; the list is that many places shorter than it could have been"
	withheldTwo    = "2 of the engine's hits were withheld -- excluded by [privacy] never, or from a collection this channel has no area for; the list is that many places shorter than it could have been"
	agedStamp      = "the last full reconciliation was 2000-01-01T00:00:00+00:00, more than 24 hours ago: a source may have changed without this answer knowing (run `brain reconcile`)"
)

// areaEntry is one registry table for an area at path.
func areaEntry(scope, path string) string {
	return "[[area]]\nscope = \"" + scope + "\"\npath = \"" + filepath.ToSlash(path) + "\"\n\n"
}

// writeArea lays down an area's directory, its manifest and, when registered paths are
// given, its identity register: a registry entry alone says nothing about the area's
// privacy mode, and a hit outside the register is a finding.
func writeArea(t *testing.T, root, scope, manifest string, registered ...string) string {
	t.Helper()
	dir := filepath.Join(root, search.CollectionName(scope))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("failed to create area dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if len(registered) > 0 {
		register := "doc_id\tpfad\tcontent_hash\trevision\n"
		for i, relative := range registered {
			register += "id-" + string(rune('a'+i)) + "\t" + relative + "\tsha256:0\t1\n"
		}
		if err := os.WriteFile(filepath.Join(dir, "_identities.tsv"), []byte(register), 0o600); err != nil {
			t.Fatalf("failed to write register: %v", err)
		}
	}
	return dir
}

// writeStamp puts a reconcile stamp into a state directory.
func writeStamp(t *testing.T, stateDir, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(stateDir, "maintenance"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "maintenance", "last-run.txt"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func manifestOf(scope string) string {
	return "[area]\nscope = \"" + scope + "\"\n"
}

func TestCollectionName(t *testing.T) {
	cases := []struct {
		scope    string
		expected string
	}{
		{"knowledge", "knowledge"},
		{"project/ultra-brain", "project-ultra-brain"},
		{"project/deep/nested", "project-deep-nested"},
		{"-unsafe@scope-", "unsafe-scope"},
		{"foo--bar", "foo--bar"},
		{"a/b/c", "a-b-c"},
	}

	for _, tc := range cases {
		t.Run(tc.scope, func(t *testing.T) {
			got := search.CollectionName(tc.scope)
			if got != tc.expected {
				t.Errorf("CollectionName(%q) = %q; want %q", tc.scope, got, tc.expected)
			}
		})
	}
}

func TestExecuteSearch_RegistryNotFound(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent")
	port := &mockSearchPort{}
	_, err := search.ExecuteSearch("query", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, nonexistent, nonexistent, searchNow)
	if err == nil {
		t.Fatal("expected error for nonexistent registry, got nil")
	}
	if port.calls != 0 {
		t.Errorf("the engine was asked without a registry: %d calls", port.calls)
	}
}

func TestExecuteSearch_EmptyRegistry(t *testing.T) {
	dir := t.TempDir()
	writeRegistry(t, dir, "# empty\n")
	writeStamp(t, dir, "2000-01-01T00:00:00+00:00\n")

	port := &mockSearchPort{}
	answer, err := search.ExecuteSearch("query", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answer.Hits) != 0 || len(answer.Findings) != 0 {
		t.Errorf("expected empty answer for empty registry, got hits=%d findings=%v", len(answer.Hits), answer.Findings)
	}
	if port.calls != 0 {
		t.Errorf("expected port not to be called for empty registry, got %d calls", port.calls)
	}
}

// core._visible_areas reads the manifest of every registered area and lets the error out:
// an area without a declaration fails the search instead of being asked about blind.
func TestExecuteSearch_AnAreaWithoutADeclarationFailsTheSearch(t *testing.T) {
	dir := t.TempDir()
	bare := filepath.Join(dir, "bare")
	if err := os.MkdirAll(bare, 0o750); err != nil {
		t.Fatal(err)
	}
	writeRegistry(t, dir, areaEntry("bare", bare))

	port := &mockSearchPort{}
	if _, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow); err == nil {
		t.Fatal("expected an error for an area without a manifest")
	}
	if port.calls != 0 {
		t.Errorf("the engine was asked: %d calls", port.calls)
	}
}

func TestExecuteSearch_UnknownScope(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge)+areaEntry("project/ultra-brain", ub))

	port := &mockSearchPort{}
	_, err := search.ExecuteSearch("query", "project/missing", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	want := "unknown scope 'project/missing'; known scopes are: knowledge, project/ultra-brain"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
	if port.calls != 0 {
		t.Errorf("expected port not to be called on unknown scope error, got %d calls", port.calls)
	}
}

func TestExecuteSearch_AllScope(t *testing.T) {
	dir := t.TempDir()
	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"), "docs/intro.md")
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "note.md")
	writeRegistry(t, dir, areaEntry("project/ultra-brain", ub)+areaEntry("knowledge", knowledge))

	var requestedCols []string
	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			requestedCols = collections
			return []search.SearchHit{
				{Collection: "project-ultra-brain", Relative: "docs/intro.md", Line: 1, Title: "Introduction", Snippet: "First line\nSecond line", Score: 0.85},
				{Collection: "knowledge", Relative: "note.md", Line: 5, Title: "A Note", Snippet: "Note content", Score: 0.72},
			}, nil
		},
	}

	answer, err := search.ExecuteSearch("arch", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(requestedCols, []string{"knowledge", "project-ultra-brain"}) {
		t.Errorf("expected sorted collections [knowledge project-ultra-brain], got %v", requestedCols)
	}
	if len(answer.Hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(answer.Hits))
	}
	if answer.Hits[0].Scope != "project/ultra-brain" || answer.Hits[1].Scope != "knowledge" {
		t.Errorf("scopes %q and %q", answer.Hits[0].Scope, answer.Hits[1].Scope)
	}
	if len(answer.Findings) != 0 {
		t.Errorf("expected no findings, got %v", answer.Findings)
	}
}

// A named scope asks, and reads the register of, that area alone: the broken register of
// the other one does not fail the search.
func TestExecuteSearch_SpecificScope(t *testing.T) {
	dir := t.TempDir()
	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"), "docs/intro.md")
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	if err := os.WriteFile(filepath.Join(knowledge, "_identities.tsv"), []byte("header\nbroken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRegistry(t, dir, areaEntry("project/ultra-brain", ub)+areaEntry("knowledge", knowledge))

	var requestedCols []string
	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			requestedCols = collections
			return []search.SearchHit{{Collection: "project-ultra-brain", Relative: "docs/intro.md", Line: 1, Title: "Introduction", Snippet: "Hello", Score: 0.9}}, nil
		},
	}

	answer, err := search.ExecuteSearch("hello", "project/ultra-brain", search.ProfileFull, 3, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(requestedCols, []string{"project-ultra-brain"}) {
		t.Errorf("expected [project-ultra-brain], got %v", requestedCols)
	}
	if len(answer.Hits) != 1 || answer.Hits[0].Scope != "project/ultra-brain" || len(answer.Findings) != 0 {
		t.Errorf("expected 1 hit with scope project/ultra-brain and no findings, got %+v %v", answer.Hits, answer.Findings)
	}
}

func TestExecuteSearch_PortError(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	expectedErr := errors.New("backend timeout")
	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			return nil, expectedErr
		},
	}

	_, err := search.ExecuteSearch("query", "all", search.ProfileKeyword, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if port.calls != 1 {
		t.Errorf("a failed search is not retried, got %d calls", port.calls)
	}
}

func TestExecuteSearch_EmptyRetrySuccess(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "retry.md")
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	var port *mockSearchPort
	port = &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			if port.calls == 1 {
				return nil, nil // first call empty
			}
			return []search.SearchHit{{Collection: "knowledge", Relative: "retry.md", Line: 1, Title: "Found on retry", Score: 0.5}}, nil
		},
	}

	answer, err := search.ExecuteSearch("retry query", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.calls != 2 {
		t.Errorf("expected 2 port calls, got %d", port.calls)
	}
	if len(answer.Hits) != 1 {
		t.Fatalf("expected 1 hit from second attempt, got %d", len(answer.Hits))
	}
	if len(answer.Findings) != 0 {
		t.Errorf("expected 0 findings when retry succeeds, got %v", answer.Findings)
	}
}

func TestExecuteSearch_EmptyRetryError(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	retryErr := errors.New("backend failed on retry")
	var port *mockSearchPort
	port = &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			if port.calls == 1 {
				return nil, nil // first call empty
			}
			return nil, retryErr
		},
	}

	_, err := search.ExecuteSearch("retry err", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if !errors.Is(err, retryErr) {
		t.Fatalf("expected error %v, got %v", retryErr, err)
	}
}

func TestExecuteSearch_EmptyRetryTwiceGivesFinding(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	port := &mockSearchPort{}
	answer, err := search.ExecuteSearch("nothing", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.calls != 2 {
		t.Errorf("expected 2 port calls on double empty, got %d", port.calls)
	}
	if len(answer.Hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(answer.Hits))
	}
	if !reflect.DeepEqual(answer.Findings, []string{twiceEmptyFast}) {
		t.Fatalf("got %q", answer.Findings)
	}
}

func TestExecuteSearch_WithheldHits(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "public.md")
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			return []search.SearchHit{
				{Collection: "unregistered-collection", Relative: "secret.md", Line: 1, Title: "Secret", Score: 0.9},
				{Collection: "knowledge", Relative: "public.md", Line: 1, Title: "Public", Score: 0.8},
			}, nil
		},
	}

	answer, err := search.ExecuteSearch("mix", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answer.Hits) != 1 || answer.Hits[0].Relative != "public.md" {
		t.Fatalf("expected only public hit, got %v", answer.Hits)
	}
	if !reflect.DeepEqual(answer.Findings, []string{withheldOne}) {
		t.Fatalf("got %q", answer.Findings)
	}
}

// Measured against core._assemble with n = 2: every kept hit is checked against its register
// -- the third one too, which the cut to n removes afterwards -- and both drops are counted.
func TestExecuteSearch_RegisterFindingsCoverEveryKeptHitBeforeTheCut(t *testing.T) {
	dir := t.TempDir()
	x := writeArea(t, dir, "project/x", "[area]\nscope = \"project/x\"\n\n[privacy]\nnever = [\"secret/**\"]\n")
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("project/x", x)+areaEntry("knowledge", knowledge))

	port := &mockSearchPort{searchFunc: func(string, []string, search.Profile, int) ([]search.SearchHit, error) {
		return []search.SearchHit{
			{Collection: "project-x", Relative: "secret/a.md"},
			{Collection: "stranger", Relative: "b.md"},
			{Collection: "project-x", Relative: "notes/open.md"},
			{Collection: "knowledge", Relative: "n1.md"},
			{Collection: "knowledge", Relative: "n2.md"},
		}, nil
	}}

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 2, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, hit := range answer.Hits {
		kept = append(kept, hit.Scope+"/"+hit.Relative)
	}
	if !reflect.DeepEqual(kept, []string{"project/x/notes/open.md", "knowledge/n1.md"}) {
		t.Errorf("kept %q", kept)
	}
	want := []string{
		"project/x/notes/open.md: hit is not in the register; reindex to catch up",
		"knowledge/n1.md: hit is not in the register; reindex to catch up",
		"knowledge/n2.md: hit is not in the register; reindex to catch up",
		withheldTwo,
	}
	if !reflect.DeepEqual(answer.Findings, want) {
		t.Errorf("findings %q", answer.Findings)
	}
}

func TestExecuteSearch_LimitsHitsToN(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "file.md")
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			var hits []search.SearchHit
			for i := 1; i <= 5; i++ {
				hits = append(hits, search.SearchHit{Collection: "knowledge", Relative: "file.md", Line: i, Title: "Title", Score: 0.5})
			}
			return hits, nil
		},
	}

	answer, err := search.ExecuteSearch("limit", "all", search.ProfileFast, 2, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answer.Hits) != 2 || answer.Hits[1].Line != 2 {
		t.Errorf("expected the first 2 hits, got %+v", answer.Hits)
	}
}

// Python reads every register before it looks at a hit: a broken one fails the search even
// when no hit comes from its area.
func TestExecuteSearch_ABrokenRegisterFailsEvenWithoutItsHits(t *testing.T) {
	dir := t.TempDir()
	a := writeArea(t, dir, "a", manifestOf("a"), "x.md")
	b := writeArea(t, dir, "b", manifestOf("b"))
	if err := os.WriteFile(filepath.Join(b, "_identities.tsv"), []byte("doc_id\tpfad\tcontent_hash\trevision\nid\tb.md\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRegistry(t, dir, areaEntry("a", a)+areaEntry("b", b))

	port := &mockSearchPort{searchFunc: func(string, []string, search.Profile, int) ([]search.SearchHit, error) {
		return []search.SearchHit{{Collection: "a", Relative: "x.md"}}, nil
	}}
	_, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	want := filepath.Join(b, "_identities.tsv") + ": line 2: expected 4 tab-separated fields, found 2"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

// A read-only area keeps manifest and register in the legacy state directory, the registry
// lives elsewhere: the register that counts is the legacy one, not the checkout's.
func TestExecuteSearch_AReadOnlyAreaReadsItsRegisterFromTheLegacyDirectory(t *testing.T) {
	registryDir := t.TempDir()
	legacyDir := t.TempDir()
	checkout := writeArea(t, t.TempDir(), "lent", manifestOf("lent"))
	writeArea(t, filepath.Join(legacyDir, "areas"), "lent", manifestOf("lent"), "doc.md")
	writeRegistry(t, registryDir, areaEntry("lent", checkout)+"readonly = true\n")

	port := &mockSearchPort{searchFunc: func(string, []string, search.Profile, int) ([]search.SearchHit, error) {
		return []search.SearchHit{{Collection: "lent", Relative: "doc.md"}}, nil
	}}
	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, registryDir, legacyDir, searchNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(answer.Hits) != 1 || len(answer.Findings) != 0 {
		t.Fatalf("hits %+v, findings %q", answer.Hits, answer.Findings)
	}
}

func TestExecuteSearch_FindingsComeInPythonsOrder(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))
	writeStamp(t, dir, "2000-01-01T00:00:00+00:00\n")

	t.Run("twice empty, then the stamp", func(t *testing.T) {
		answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, &mockSearchPort{}, dir, dir, searchNow)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(answer.Findings, []string{twiceEmptyFast, agedStamp}) {
			t.Fatalf("got %q", answer.Findings)
		}
	})
	t.Run("register, withheld, then the stamp", func(t *testing.T) {
		port := &mockSearchPort{searchFunc: func(string, []string, search.Profile, int) ([]search.SearchHit, error) {
			return []search.SearchHit{{Collection: "stranger", Relative: "b.md"}, {Collection: "knowledge", Relative: "n1.md"}}, nil
		}}
		answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"knowledge/n1.md: hit is not in the register; reindex to catch up", withheldOne, agedStamp}
		if !reflect.DeepEqual(answer.Findings, want) {
			t.Fatalf("got %q", answer.Findings)
		}
	})
}

// core.search returns before the stamp is read when no area is visible.
func TestExecuteSearch_NoVisibleAreaCarriesNoStaleFinding(t *testing.T) {
	dir := t.TempDir()
	closed := writeArea(t, dir, "project/secret", "[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("project/secret", closed))
	writeStamp(t, dir, "2000-01-01T00:00:00+00:00\n")

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelCloud, &mockSearchPort{}, dir, dir, searchNow)
	if err != nil {
		t.Fatal(err)
	}
	if answer.Findings != nil || answer.Hits != nil {
		t.Fatalf("got %+v", answer)
	}
}

func TestExecuteSearch_AStampThatCannotBeReadFailsTheSearch(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))
	if err := os.MkdirAll(filepath.Join(dir, "maintenance", "last-run.txt"), 0o750); err != nil {
		t.Fatal(err)
	}
	port := &mockSearchPort{}
	if _, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow); err == nil {
		t.Fatal("expected the unreadable stamp to fail the search")
	}
	if port.calls != 2 {
		t.Errorf("the stamp is read after the engine answered, got %d calls", port.calls)
	}
}

func TestFormatSearch(t *testing.T) {
	t.Run("empty hits", func(t *testing.T) {
		ans := &search.SearchAnswer{Hits: nil}
		out := search.FormatSearch(ans)
		if out != search.NoMatches+"\n" {
			t.Errorf("expected %q, got %q", search.NoMatches+"\n", out)
		}
	})

	t.Run("formatted results with snippet indentation", func(t *testing.T) {
		ans := &search.SearchAnswer{
			Hits: []search.SearchHit{
				{Scope: "project/ultra-brain", Relative: "docs/intro.md", Line: 10, Title: "Introduction", Score: 0.854, Snippet: "@@ -1,3 @@ (header)\n# Introduction\n\nSome detail"},
				{Collection: "knowledge", Scope: "knowledge", Relative: "note.md", Line: 1, Title: "A Note", Score: 0.5, Snippet: "Single line snippet"},
			},
		}

		out := search.FormatSearch(ans)
		expected := "brain://project/ultra-brain/docs/intro.md:10  85%  Introduction\n" +
			"    @@ -1,3 @@ (header)\n" +
			"    # Introduction\n" +
			"    \n" +
			"    Some detail\n\n" +
			"brain://knowledge/note.md:1  50%  A Note\n" +
			"    Single line snippet\n\n"

		if out != expected {
			t.Errorf("FormatSearch mismatch.\nGot:\n%s\nWant:\n%s", out, expected)
		}
	})

	// Measured against cli._print_search on 2026-09-15.
	head := "brain://project/ultra-brain/docs/intro.md:10  85%  Introduction\n"
	for _, tc := range []struct {
		name, snippet, want string
	}{
		{"an empty snippet prints no line", "", head + "\n"},
		{"a trailing newline prints no extra line", "a\n", head + "    a\n\n"},
		{"every separator of str.splitlines", "a\r\nb\rc\vd\fe\x1cf\x1dg\x1eh\xc2\x85i\xe2\x80\xa8j\xe2\x80\xa9k",
			head + "    a\n    b\n    c\n    d\n    e\n    f\n    g\n    h\n    i\n    j\n    k\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hit := search.SearchHit{Scope: "project/ultra-brain", Relative: "docs/intro.md", Line: 10, Title: "Introduction", Score: 0.854, Snippet: tc.snippet}
			if got := search.FormatSearch(&search.SearchAnswer{Hits: []search.SearchHit{hit}}); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("scores round like format(score, '.0%')", func(t *testing.T) {
		for score, want := range map[float64]string{0.005: "0%", 0.015: "2%", 0.125: "12%", 0.145: "14%", 0.995: "100%", 1.0: "100%", 0.0: "0%"} {
			hit := search.SearchHit{Scope: "s", Relative: "r.md", Line: 1, Title: "T", Score: score}
			if got := search.FormatSearch(&search.SearchAnswer{Hits: []search.SearchHit{hit}}); got != "brain://s/r.md:1  "+want+"  T\n\n" {
				t.Errorf("score %v: got %q", score, got)
			}
		}
	})

	t.Run("an empty scope stays empty", func(t *testing.T) {
		hit := search.SearchHit{Collection: "knowledge", Relative: "note.md", Line: 1, Title: "A Note", Score: 0.5}
		if got := search.FormatSearch(&search.SearchAnswer{Hits: []search.SearchHit{hit}}); got != "brain:///note.md:1  50%  A Note\n\n" {
			t.Fatalf("got %q", got)
		}
	})
}

// The promise slice 2a exists for: on the cloud channel a `local_only` area
// is not filtered out of the answer, it is never asked about. Filtering
// afterwards would mean the engine was handed the question, and the question
// is already the disclosure (arch spec 7.2.1).
func TestExecuteSearch_LocalOnlyAreaIsNotAskedOnCloud(t *testing.T) {
	dir := t.TempDir()
	open := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	closed := writeArea(t, dir, "project/secret",
		"[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("knowledge", open)+areaEntry("project/secret", closed))

	var requested []string
	port := &mockSearchPort{
		searchFunc: func(_ string, collections []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
			requested = collections
			return nil, nil
		},
	}

	_, err := search.ExecuteSearch(
		"q", "all", search.ProfileFast, 5, privacy.ChannelCloud, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, col := range requested {
		if col == "project-secret" {
			t.Fatalf("the engine was asked about a local_only collection on the cloud channel: %v", requested)
		}
	}
	if len(requested) != 1 || requested[0] != "knowledge" {
		t.Errorf("expected only [knowledge], got %v", requested)
	}
}

// The counterpart: the same area is ordinary on the local channel. Without
// this, a filter that simply dropped everything would pass the test above.
func TestExecuteSearch_LocalOnlyAreaIsAskedOnLocal(t *testing.T) {
	dir := t.TempDir()
	open := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	closed := writeArea(t, dir, "project/secret",
		"[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("knowledge", open)+areaEntry("project/secret", closed))

	var requested []string
	port := &mockSearchPort{
		searchFunc: func(_ string, collections []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
			requested = collections
			return nil, nil
		},
	}

	if _, err := search.ExecuteSearch(
		"q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(requested) != 2 {
		t.Errorf("expected both collections on the local channel, got %v", requested)
	}
}

// Naming the invisible area by scope must not be a way around the channel,
// and the refusal must be the *same* refusal an area that does not exist
// gets. Not because the name is a secret -- the caller just typed it -- but
// because two different messages would let a caller tell "no such area" from
// "there, but not for you", which is exactly what local_only hides. Python
// makes them one error for this reason (core.py ScopeError).
func TestExecuteSearch_InvisibleScopeIsRefusedLikeAnAbsentOne(t *testing.T) {
	dir := t.TempDir()
	open := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	closed := writeArea(t, dir, "project/secret",
		"[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("knowledge", open)+areaEntry("project/secret", closed))

	invisible := &mockSearchPort{}
	_, errInvisible := search.ExecuteSearch(
		"q", "project/secret", search.ProfileFast, 5, privacy.ChannelCloud, invisible, dir, dir, searchNow)
	absent := &mockSearchPort{}
	_, errAbsent := search.ExecuteSearch(
		"q", "project/does-not-exist", search.ProfileFast, 5, privacy.ChannelCloud, absent, dir, dir, searchNow)

	if errInvisible == nil || errAbsent == nil {
		t.Fatalf("expected both to be refused; invisible=%v absent=%v", errInvisible, errAbsent)
	}
	// Same shape, differing only in the scope the caller themselves named.
	shape := func(err error, scope string) string {
		return strings.Replace(err.Error(), "'"+scope+"'", "<scope>", 1)
	}
	if shape(errInvisible, "project/secret") != shape(errAbsent, "project/does-not-exist") {
		t.Errorf("the two refusals differ, which tells them apart: invisible=%v absent=%v",
			errInvisible, errAbsent)
	}
	// And neither may list it among the scopes that do exist here.
	if strings.Contains(strings.SplitN(errInvisible.Error(), "known scopes are:", 2)[1], "secret") {
		t.Errorf("the invisible area is listed as known: %v", errInvisible)
	}
	if invisible.calls != 0 || absent.calls != 0 {
		t.Errorf("the engine was asked despite the refusal: %d and %d calls", invisible.calls, absent.calls)
	}
}

// With every area invisible the engine must not be asked at all: an empty
// collection list drops qmd's collection filter, so it would search
// everything -- local_only included.
func TestExecuteSearch_NoVisibleAreaAsksNothing(t *testing.T) {
	dir := t.TempDir()
	closed := writeArea(t, dir, "project/secret",
		"[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("project/secret", closed))

	port := &mockSearchPort{}
	answer, err := search.ExecuteSearch(
		"q", "all", search.ProfileFast, 5, privacy.ChannelCloud, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.calls != 0 {
		t.Errorf("the engine was asked with no visible area: %d calls", port.calls)
	}
	if len(answer.Hits) != 0 {
		t.Errorf("expected no hits, got %d", len(answer.Hits))
	}
}

// `[privacy] never` names a path no channel may reach. Python's _assemble
// drops such a hit before it ever looks the path up; the withholding line
// claims "excluded by [privacy] never", so the filter has to be there.
func TestExecuteSearch_NeverPathIsDroppedOnItsOwnChannel(t *testing.T) {
	dir := t.TempDir()
	area := writeArea(t, dir, "knowledge",
		"[area]\nscope = \"knowledge\"\n\n[privacy]\nnever = [\"secret/**\"]\n", "notes/open.md")
	writeRegistry(t, dir, areaEntry("knowledge", area))

	port := &mockSearchPort{searchFunc: func(_ string, _ []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
		return []search.SearchHit{
			{Collection: "knowledge", Relative: "secret/passwords.md", Title: "Vault", Snippet: "hunter2"},
			{Collection: "knowledge", Relative: "notes/open.md", Title: "Open", Snippet: "public"},
		}, nil
	}}

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("ExecuteSearch returned an error: %v", err)
	}
	if len(answer.Hits) != 1 || answer.Hits[0].Relative != "notes/open.md" {
		t.Fatalf("expected only the readable hit, got %+v", answer.Hits)
	}

	// The withheld path must not surface anywhere -- not in a hit, not in a
	// finding. A hit dropped by `never` is precisely the one whose name may
	// not leave the machine.
	rendered := search.FormatSearch(answer) + strings.Join(answer.Findings, "\n")
	for _, forbidden := range []string{"secret/passwords.md", "hunter2", "Vault"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("the withheld hit leaked through the output: %q in %q", forbidden, rendered)
		}
	}
	if !reflect.DeepEqual(answer.Findings, []string{withheldOne}) {
		t.Errorf("expected exactly one withholding finding counting 1, got %v", answer.Findings)
	}
}

// The count is one line for both reasons together, so two wordings cannot be
// read back to tell which filter caught a hit.
func TestExecuteSearch_NeverAndUnknownCollectionShareOneCount(t *testing.T) {
	dir := t.TempDir()
	area := writeArea(t, dir, "knowledge",
		"[area]\nscope = \"knowledge\"\n\n[privacy]\nnever = [\"secret/**\"]\n", "open.md")
	writeRegistry(t, dir, areaEntry("knowledge", area))

	port := &mockSearchPort{searchFunc: func(_ string, _ []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
		return []search.SearchHit{
			{Collection: "knowledge", Relative: "secret/a.md"},
			{Collection: "stranger", Relative: "b.md"},
			{Collection: "knowledge", Relative: "open.md"},
		}, nil
	}}

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("ExecuteSearch returned an error: %v", err)
	}
	if len(answer.Hits) != 1 {
		t.Fatalf("expected one surviving hit, got %+v", answer.Hits)
	}
	if !reflect.DeepEqual(answer.Findings, []string{withheldTwo}) {
		t.Errorf("expected a single finding counting 2, got %v", answer.Findings)
	}
}

// An area without a `never` list keeps every hit: the filter must not turn a
// missing declaration into a blanket refusal.
func TestExecuteSearch_WithoutNeverEveryHitSurvives(t *testing.T) {
	dir := t.TempDir()
	area := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "secret/a.md", "open.md")
	writeRegistry(t, dir, areaEntry("knowledge", area))

	port := &mockSearchPort{searchFunc: func(_ string, _ []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
		return []search.SearchHit{
			{Collection: "knowledge", Relative: "secret/a.md"},
			{Collection: "knowledge", Relative: "open.md"},
		}, nil
	}}

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, dir, searchNow)
	if err != nil {
		t.Fatalf("ExecuteSearch returned an error: %v", err)
	}
	if len(answer.Hits) != 2 {
		t.Fatalf("expected both hits, got %+v", answer.Hits)
	}
	if len(answer.Findings) != 0 {
		t.Errorf("expected no withholding finding, got %v", answer.Findings)
	}
}
```

Gemessene Referenzwerte hinter den Tests: `core._ask` mit zweimal `()` → Befund `twiceEmptyFast`, 2 Aufrufe; `core._assemble` mit `project/x` (`never = ("secret/**",)`) und `knowledge`, fünf Treffern wie im Test, nicht existierendem Zustandsverzeichnis und `n = 2` → Ergebnisse `[("project/x","notes/open.md"), ("knowledge","n1.md")]`, Befunde die drei Registerzeilen und `withheldTwo`; `cli._print_search`:

| Snippet | stdout |
|---|---|
| `@@ -1,3 @@ (header)\n# Introduction\n\nSome detail` + zweiter Treffer | wie im Test „formatted results" |
| `""` | `brain://project/ultra-brain/docs/intro.md:10  85%  Introduction\n\n` |
| `"a\n"` | `brain://project/ultra-brain/docs/intro.md:10  85%  Introduction\n    a\n\n` |
| `a\r\nb\rc\vd\fe\x1cf\x1dg\x1eh` U+0085 `i` U+2028 `j` U+2029 `k` | je Buchstabe eine Zeile `    x\n`, dann `\n` |
| Score 0.005 / 0.015 / 0.125 / 0.145 / 0.995 / 1.0 / 0.0 | `0%` / `2%` / `12%` / `14%` / `100%` / `100%` / `0%` |

`internal/brain/search/visibility_test.go` vollständig ersetzen durch:

```go
package search_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/testlock"
)

// closedWorld registers, beside the open area "alpha", the read-only "zz-lent" whose
// manifest is the `local_only` one in the legacy state directory while its checkout
// carries an open `.brain.toml`. With sealed it adds "zz-sealed", whose `local_only`
// `.ultra-brain/config.toml` cannot be read beside a stale `manual_cloud` `.brain.toml`.
// It returns the directory that holds the registry and the legacy state both.
func closedWorld(t *testing.T, sealed bool) string {
	t.Helper()
	root := t.TempDir()
	const open = "[privacy]\nmode = \"manual_cloud\"\n"
	const closed = "[privacy]\nmode = \"local_only\"\n"
	put := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	alpha := filepath.Join(root, "alpha")
	lent := filepath.Join(root, "lent")
	put(filepath.Join(alpha, ".brain.toml"), "[area]\nscope = \"alpha\"\n")
	put(filepath.Join(lent, ".brain.toml"), "[area]\nscope = \"zz-lent\"\n"+open)
	put(filepath.Join(root, "areas", "zz-lent", ".brain.toml"), "[area]\nscope = \"zz-lent\"\n"+closed)
	registry := "[[area]]\nscope = \"alpha\"\npath = \"" + filepath.ToSlash(alpha) + "\"\n\n" +
		"[[area]]\nscope = \"zz-lent\"\npath = \"" + filepath.ToSlash(lent) + "\"\nreadonly = true\n\n"
	if sealed {
		dir := filepath.Join(root, "sealed")
		put(filepath.Join(dir, ".brain.toml"), "[area]\nscope = \"zz-sealed\"\n"+open)
		config := filepath.Join(dir, ".ultra-brain", "config.toml")
		put(config, "[area]\nscope = \"zz-sealed\"\n"+closed)
		testlock.Lock(t, config)
		registry += "[[area]]\nscope = \"zz-sealed\"\npath = \"" + filepath.ToSlash(dir) + "\"\n"
	}
	writeRegistry(t, root, registry)
	return root
}

// N3 of the scheibe-6 merge re-review, on the surface where it matters most:
// the engine must not be asked about a closed area at all, because the
// question is already the disclosure. A read-only area's own declaration is the
// one in the legacy state directory, not the one in its checkout.
func TestExecuteSearch_AClosedDeclarationIsNotAsked(t *testing.T) {
	stateDir := closedWorld(t, false)
	for _, tc := range []struct {
		channel privacy.Channel
		want    []string
	}{
		{privacy.ChannelCloud, []string{"alpha"}},
		{privacy.ChannelLocal, []string{"alpha", "zz-lent"}},
	} {
		var requested []string
		port := &mockSearchPort{
			searchFunc: func(_ string, collections []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
				requested = collections
				return nil, nil
			},
		}
		if _, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, tc.channel, port, stateDir, stateDir, searchNow); err != nil {
			t.Fatalf("%s: %v", tc.channel, err)
		}
		if !reflect.DeepEqual(requested, tc.want) {
			t.Errorf("%s: engine asked about %v, want %v", tc.channel, requested, tc.want)
		}
	}

	port := &mockSearchPort{}
	_, err := search.ExecuteSearch("q", "zz-lent", search.ProfileFast, 5, privacy.ChannelCloud, port, stateDir, stateDir, searchNow)
	if err == nil || !strings.Contains(err.Error(), "unknown scope") {
		t.Errorf("cloud search of zz-lent: err = %v, want the unknown-scope refusal", err)
	}
	if port.calls != 0 {
		t.Errorf("cloud search of zz-lent asked the engine %d times", port.calls)
	}
}

// A declaration that exists and cannot be read is not skipped: core._visible_areas reads the
// manifest of every registered area and lets the OSError out, so the whole search fails on
// either channel and the engine is never asked.
func TestExecuteSearch_AnUnreadableDeclarationFailsTheWholeSearch(t *testing.T) {
	stateDir := closedWorld(t, true)
	for _, channel := range []privacy.Channel{privacy.ChannelCloud, privacy.ChannelLocal} {
		port := &mockSearchPort{}
		if _, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, channel, port, stateDir, stateDir, searchNow); err == nil {
			t.Errorf("%s: expected the unreadable declaration to fail the search", channel)
		}
		if port.calls != 0 {
			t.Errorf("%s: the engine was asked %d times", channel, port.calls)
		}
	}
}
```

- [ ] **Step 24: Scheitern sehen**

```bash
go vet ./internal/brain/search/
```
Erwartet (gemessen mit den neuen Testdateien und dem ub-`search.go`), Exit 1 und genau dieselben zwei Zeilen wie in Step 22:
```
# github.com/xidus90/loomux/internal/brain/search [github.com/xidus90/loomux/internal/brain/search.test]
internal\brain\search\search.go:80:24: assignment mismatch: 2 variables but privacy.VisibleManifest returns 3 values
```
Die Tests werden nicht mehr typgeprüft, solange das Paket selbst nicht kompiliert; `too many arguments in call to search.ExecuteSearch` erscheint deshalb nicht.

- [ ] **Step 25: `search.go` nach `core.search` implementieren** — `internal/brain/search/search.go` vollständig ersetzen. Wortgleich aus ub bleiben `NoMatches`, `unsafeScope`, `CollectionName`, `SearchAnswer`, der Kommentar zur Kanalregel, der Kommentar zur leeren Collection-Liste und die Wortlaute beider Befunde; neu sind die Signatur, `askTwice`, `assemble`, das Register, der Stempel, der Schnitt `ordered[:n]` am Ende und `FormatSearch` ohne Rückfall auf `Collection` und mit `pytext.SplitLines`.

```go
package search

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// NoMatches is the agreed string when a search returns no hits on either surface.
const NoMatches = "no matches"

var unsafeScope = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// CollectionName maps an area's scope to its qmd collection name.
//
// Slashes and unsafe characters become dashes, and leading/trailing dashes are stripped,
// matching the Python implementation in paths.collection_name.
func CollectionName(scope string) string {
	return strings.Trim(unsafeScope.ReplaceAllString(scope, "-"), "-")
}

// SearchAnswer contains the assembled search hits and any diagnostic findings.
type SearchAnswer struct {
	Hits     []SearchHit
	Findings []string
}

// ExecuteSearch runs a search against the areas this channel may see, as core.search does.
//
// The registry comes from registryDir. legacyDir is where read-only areas keep their
// manifest and register, and where the reconcile stamp lies, until stage 3.
// Scope can be "all" to query all visible areas, or a specific area scope; an unknown one
// -- or one invisible on this channel, which for the caller is the same thing -- is
// privacy.VisibleAreas' refusal.
//
// The channel decides which areas are asked about, and it does so *before*
// the engine is handed anything. Filtering the answer afterwards would mean
// the invisible area was queried, and the question is already the disclosure
// of the question (arch spec 7.2.1). That is the whole difference between
// this and a search that hides its results.
//
// Findings come in core.search's order: the twice-empty answer, the register findings in hit
// order, the withheld count, the aged stamp.
func ExecuteSearch(query, scope string, profile Profile, n int, channel privacy.Channel, port SearchPort, registryDir, legacyDir string, now time.Time) (*SearchAnswer, error) {
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, scope, channel)
	if err != nil {
		return nil, err
	}
	if len(areas) == 0 {
		// Not an optimisation. An empty collection list drops qmd's
		// collection filter altogether, so the engine would search
		// everything it holds -- the local_only areas included.
		return &SearchAnswer{Hits: nil, Findings: nil}, nil
	}

	collections := make([]string, 0, len(areas))
	for _, visible := range areas {
		collections = append(collections, CollectionName(visible.Area.Scope))
	}
	sort.Strings(collections)

	hits, findings, err := askTwice(port, query, collections, profile, n)
	if err != nil {
		return nil, err
	}
	answer, err := assemble(hits, areas, legacyDir, n)
	if err != nil {
		return nil, err
	}
	stale, err := StaleReconcile(legacyDir, now)
	if err != nil {
		return nil, err
	}
	findings = append(findings, answer.Findings...)
	findings = append(findings, stale...)
	return &SearchAnswer{Hits: answer.Hits, Findings: findings}, nil
}

// askTwice is core._ask: one retry on an empty answer, then a finding instead of a silence.
func askTwice(port SearchPort, query string, collections []string, profile Profile, n int) ([]SearchHit, []string, error) {
	for range 2 {
		hits, err := port.Search(query, collections, profile, n)
		if err != nil {
			return nil, nil, err
		}
		if len(hits) > 0 {
			return hits, nil, nil
		}
	}
	return nil, []string{fmt.Sprintf(
		"the search engine answered empty twice in a row on profile %s; "+
			"an empty answer to a meaning search is practically unreachable, so this is "+
			"more likely a silent failure of the engine than an absence of matches (spec 16.14)",
		profile,
	)}, nil
}

// assemble is core._assemble. The registers of all asked areas are read before the first hit
// is looked at, so a broken one fails the search whichever area the hits come from.
//
// A hit from a collection this channel has no area for, or under `[privacy] never`, is only
// counted: a path under `never` is the one whose name may not leave the machine, so it must
// not reach a finding either -- which is why both reasons share the single count and why the
// check comes before the register is so much as looked at. Every kept hit is checked against
// its register, and the list is cut to n only at the end.
func assemble(hits []SearchHit, areas []privacy.VisibleArea, legacyDir string, n int) (*SearchAnswer, error) {
	type declared struct {
		scope    string
		manifest *config.Manifest
	}
	byCollection := make(map[string]declared, len(areas))
	for _, visible := range areas {
		byCollection[CollectionName(visible.Area.Scope)] = declared{scope: visible.Area.Scope, manifest: visible.Manifest}
	}
	registers := make(map[string]map[string]identity.Identity, len(areas))
	for _, visible := range areas {
		register, err := identity.ReadIdentities(filepath.Join(config.ManifestDir(visible.Area, legacyDir), "_identities.tsv"))
		if err != nil {
			return nil, err
		}
		registers[CollectionName(visible.Area.Scope)] = register
	}

	var findings []string
	var ordered []SearchHit
	dropped := 0
	for _, hit := range hits {
		area, known := byCollection[hit.Collection]
		if !known || !privacy.IsReadable(area.manifest, hit.Relative) {
			dropped++
			continue
		}
		if _, listed := registers[hit.Collection][hit.Relative]; !listed {
			findings = append(findings, fmt.Sprintf("%s/%s: hit is not in the register; reindex to catch up", area.scope, hit.Relative))
		}
		hit.Scope = area.scope
		ordered = append(ordered, hit)
	}

	if dropped > 0 {
		findings = append(findings, fmt.Sprintf(
			"%d of the engine's hits were withheld -- excluded by [privacy] never, "+
				"or from a collection this channel has no area for; the list is that many "+
				"places shorter than it could have been",
			dropped,
		))
	}
	if len(ordered) > n {
		ordered = ordered[:max(n, 0)]
	}
	return &SearchAnswer{Hits: ordered, Findings: findings}, nil
}

// FormatSearch renders a SearchAnswer as cli._print_search prints it: the snippet split like
// str.splitlines, so an empty snippet prints no line and a trailing newline no extra one.
func FormatSearch(answer *SearchAnswer) string {
	if len(answer.Hits) == 0 {
		return NoMatches + "\n"
	}

	var sb strings.Builder
	for _, hit := range answer.Hits {
		fmt.Fprintf(&sb, "brain://%s/%s:%d  %.0f%%  %s\n", hit.Scope, hit.Relative, hit.Line, hit.Score*100, hit.Title)
		for _, line := range pytext.SplitLines(hit.Snippet) {
			fmt.Fprintf(&sb, "    %s\n", line)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
```
Der Registerpfad ist `config.ManifestDir(area, legacyDir)` + `_identities.tsv`, also Pythons `area_artifact_dir(area, state_dir) / "_identities.tsv"` mit dem Legacy-Verzeichnis als `state_dir` für schreibgeschützte Bereiche. `unsafeScope` bleibt als umgezogene Paketvariable wortgleich (Task 15 misst sie mit `inittrace`); neuer Code bringt keinen Regex.

- [ ] **Step 26: Bestehen sehen**

```bash
gofmt -l internal/brain/search
```
Erwartet: keine Ausgabe.
```bash
go vet ./internal/brain/search/...
```
Erwartet: keine Ausgabe, Exit 0.
```bash
go test ./internal/brain/search/... -count=1
```
Erwartet: Exit 0 und eine Zeile, die auf `grep -E '^ok\s+github\.com/xidus90/loomux/internal/brain/search\s+[0-9.]+s$'` passt (die Dauer variiert; im Wegwerf-Modul 2,1 s).

- [ ] **Step 27: Coverage**

```bash
go test ./internal/brain/search/... -count=1 -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Erwartet: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/brain/search` beginnt.
```bash
go tool cover -func="$TEMP/pkg.out" | grep 'internal/brain/search/' | grep -v '100.0%'
```
Erwartet: keine Ausgabe, also jede Funktion des Pakets 100.0% (im Wegwerf-Modul gemessen: `coverage: 100.0% of statements` für das ganze Paket). Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.

- [ ] **Step 28: Stagen, Tor, Commit**

```bash
git add internal/brain/search/search.go internal/brain/search/search_test.go internal/brain/search/visibility_test.go
```
Erwartet: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0.
```bash
git branch --show-current
```
Erwartet: derselbe Zweig wie in Step 5.
```bash
git rev-parse --short HEAD
```
Erwartet: der Commit aus Step 21a.
```bash
git diff --cached --stat
```
Erwartet: genau `internal/brain/search/search.go`, `internal/brain/search/search_test.go` und `internal/brain/search/visibility_test.go`, alle drei neu, sonst nichts.
```bash
printf '%s\n' 'Search like core.search: visible areas, registers and findings' > "$TEMP/loomux-task08-msg.txt"
```
Erwartet: keine Ausgabe.
```bash
git commit -F "$TEMP/loomux-task08-msg.txt"
```
Erwartet: Exit 0; der Commit löst das Tor noch einmal aus, das ist gewollt.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität.
```bash
git log -1 --format=%B
```
Erwartet: genau die Zeile `Search like core.search: visible areas, registers and findings`, keine `Co-Authored-By`-Zeile.

- [ ] **Step 29: Bericht**

Befunde zur Umzugsregel (über Paketnamen, Importpfade und die im Vertrag genannten Änderungen hinaus):
1. `(*QmdPort).Search` geht über `invoke` (Python: `search` → `_invoke`); vorher doppelte ub-Go denselben Fehlerweg mit `strings.TrimSpace`.
2. `(*QmdPort).Indexed` spaltet über `pytext.SplitLines` und lässt den Pfad ungetrimmt (gemessen: Python liefert `'a b.md  '`).
3. `ResolveLauncher` sieht neben dem Shim nur noch `node.exe` an, Existenz wie `Path.exists()` (auch ein Verzeichnis zählt); `TestLauncher_NodeIsBatchShim` bekommt `t.Setenv("PATH", tmpDir)`, `TestLauncher_BesideNodeBat` prüft jetzt die Verweigerung `cannot find 'node' on PATH to run <shim>`.
4. `TestStartDaemonWith` räumt die beiden Backbone-Variablen vorher ab (`clearBackbone`).
5. `mcp_test.go`: drei `DefaultConnectWith`-Aufrufe und `DefaultConnect` mit `notice`; neue Tests für Einmaligkeit, Schweigen bei erreichbarem Daemon und bei gescheitertem Start, Wortlaut, Standardweg ohne qmd auf `PATH`. Neue Naht `connectDefault` mit `notice_internal_test.go`.
6. `search_test.go` neu gefasst: jede Welt hat ein Manifest je Bereich (Task 3 macht ein fehlendes zum Fehler), Register, wo Treffer ohne Befund erwartet werden, alle Befunde exakt statt `strings.Contains`; `strconv.Quote` → Pythons `repr`-Form in einfachen Anführungszeichen (`'project/secret'`) in `TestExecuteSearch_InvisibleScopeIsRefusedLikeAnAbsentOne`; `TestFormatSearch` gibt dem zweiten Treffer `Scope` (kein Rückfall mehr). Entfallen: `TestExecuteSearch_DefaultStateDirAndN` (leeres `stateDir` und `n <= 0 → 5` gibt es nicht mehr). Neu: `AnAreaWithoutADeclarationFailsTheSearch`, `RegisterFindingsCoverEveryKeptHitBeforeTheCut`, `ABrokenRegisterFailsEvenWithoutItsHits`, `AReadOnlyAreaReadsItsRegisterFromTheLegacyDirectory`, `FindingsComeInPythonsOrder`, `NoVisibleAreaCarriesNoStaleFinding`, `AStampThatCannotBeReadFailsTheSearch` und drei `FormatSearch`-Untertests.
7. `visibility_test.go`: `closedWorld(t, sealed bool)`; die Behauptung „eine unlesbare Deklaration wird auch lokal nicht gefragt" ist jetzt `TestExecuteSearch_AnUnreadableDeclarationFailsTheWholeSearch` (Python bricht die ganze Suche ab).
8. `ReadLastRun` nutzt `os.ReadFile` + `utf8.Valid` statt `pytext.ReadText` (Begründung bei Schritt 21). Befund zum Vertrag: Er nennt `pytext.ReadText` unter den Consumes von Task 8; dieser Task ruft es nicht, die Consumes-Liste oben nennt es deshalb nicht.
9. `translateReply` nimmt die Zeilennummern des Daemons vom Snippet (Schritte 17a–17c). qmd 2.8.3 setzt auf dem MCP-Weg `<line+i>: ` vor jede Snippet-Zeile (`dist/mcp/server.js:301`, `dist/cli/formatter.js:17-20`); `qmd query --json`, das die Python-Referenz liest, tut es nicht. Entfernt wird das Präfix `strconv.Itoa(rawLine+i)+": "` nur, wenn jeder an `"\n"` geteilte Teil sein eigenes trägt, gezählt ab dem rohen `line` vor der Klammerung auf 1; fehlt `line` oder trägt ein Teil sein Präfix nicht, bleibt der Snippet unverändert. Das gilt in jedem Profil. Damit gleicht die stdout von `search` der Referenz, und eine Paritätszeile entfällt. Der Fake von Task 11 muss deshalb bei jedem MCP-Treffer `line` mitschicken und seine Nummerierung ab genau diesem Wert zählen (`addLineNumbers(hit.Snippet, hit.Line)` mit `"line": hit.Line`); sonst erreichen `1: ` oder `3: one` die stdout, und `TestLoomuxsPortsReadTheFake` (`"one\ntwo"`) sowie der Korpus von Task 12 werden rot.
10. Die Schritte 17a–17c und 21a sind eingeschoben statt die Folgeschritte umzunummerieren: So bleiben die Verweise auf die Schritte 22–25 und alle übrigen Schrittnummern gültig.

Paritätszeilen, die dieser Task schafft (Task 12 trägt sie ein, ohne Freigabe):
1. Rangfolge der Suche: `fast` fragt den qmd-MCP-Daemon mit `searches:[vec]` ohne LLM-Erweiterung; die Python-CLI nimmt `qmd vsearch` mit Erweiterung (Spec-Zeile 1).
2. `search --profile keyword`: Score 1/Rang und Zeile des besten Abschnitts statt BM25 und erster Fundstelle (Spec-Zeile 2).
3. Aufwärm-Hinweis `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` auf stderr, einmal je Port, wenn loomux den Daemon selbst startet; die CLI-Referenz kennt ihn nicht (Spec-Zeile 5).
4. Daemon-Start: eine vom Nutzer gesetzte `QMD_LLAMA_GPU` oder `QMD_FORCE_CPU` (auch leer) gewinnt; `qmd_mcp.py` schreibt `{**os.environ, **env}` und überschreibt sie (Spec-Zeile 6).
5. CLI-Port (`qmd ls`, `qmd status`): loomux startet qmd mit der Umgebung des Nutzers (CUDA-Vorgabe, nichts angehängt); `qmd.py:_default_runner` pinnt `QMD_LLAMA_GPU=vulkan`.
6. Wartefrist-Wortlaut: `no qmd daemon answered on http://localhost:8765/mcp within 1m0s` gegen Pythons `no qmd daemon answered on http://localhost:8765/mcp within 60s` (gemessen: `_HttpSession(8765)`, `wait_until_reachable` mit `timeout: float = 60.0`).
7. Parser der CLI-Suche (`QmdPort.Search`, in 1b-1 ungenutzt): ub-Go-Wortlaute `expected a list of hits`, `the hit is missing 'file'` und fehlendes `line`/`score` als 0; Python sagt `expected a list of hits, found dict` (`found {type(rows).__name__}`), `the hit is missing its 'file': <repr des Treffers>` (`{row!r}`) und verlangt ganzzahliges `line` und eine Zahl als `score`.
8. MCP-Antwortübersetzung gegen `qmd_mcp.py`: `col/` ohne Rest wird in Go (col, ""), Python lässt den ganzen Pfad; eine Zeile < 1 wird in Go 1, Python behält negative Zeilen; im Korpus unsichtbar, weil die Python-CLI den MCP-Port nie benutzt.
9. Unlesbarer npm-Shim: Go meldet `cannot read <shim>: <Fehler von os.ReadFile>`, Python fängt den `OSError` von `read_text` in `_invoke` und meldet `cannot run qmd: [Errno <Nummer>] <Text>: '<shim>'`.
10. Stempel, der existiert und nicht lesbar ist (etwa ein Verzeichnis): Go-Wortlaut `read <stateDir>\maintenance\last-run.txt: Incorrect function.` statt Pythons `[Errno 13] Permission denied: '<stateDir>\maintenance\last-run.txt'`.
11. `qmd status`-Zeile: `pendingLinePattern` bleibt ub-Go mit ASCII-`\s`/`\d`; Pythons `re` liest beide als Unicode-Klassen.
12. Registerpfad in Fehlermeldungen: `filepath.Join` löst `..` in einem Registry-Pfad auf, Pythons `Path` behält es.
13. `n < 1` an `ExecuteSearch` (die CLI verweigert es vorher): Go schneidet auf eine leere Liste, Pythons `ordered[:n]` schnitte von hinten.

---

### Task 9: `internal/brain/status` — neu nach `core.status`

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 8 sind committet.

**Files:**
- Create: `internal/brain/status/status.go`
- Test: `internal/brain/status/status_test.go`

**Interfaces:**
- Consumes:
  - Task 1: `config.ReadAreaManifestUntilStage4(dir string) (*config.Manifest, error)`, mittelbar über `privacy.VisibleAreas`. Zusicherung: der Altname `.brain.toml` steht in Task 1s Namensliste `manifestNamesUntilStage4`; jede Testwelt dieses Tasks schreibt ihr Manifest unter diesem Namen, mit `[area]`-Tabelle und Scope.
  - Task 2: `pytext.IsoFormat(t time.Time) string`, `pytext.PathString(goos, p string) string`, `pytext.NFC(s string) string`
  - Task 3: `type privacy.Channel string`, `privacy.ChannelLocal`, `privacy.ChannelCloud`, `type privacy.VisibleArea struct { Area config.Area; Manifest *config.Manifest }`, `privacy.VisibleAreas(registryDir, legacyDir, scope string, ch privacy.Channel) ([]privacy.VisibleArea, error)`, `privacy.IsReadable(manifest *config.Manifest, relative string) bool`, `privacy.MatchesGlobs(patterns []string, relative string) bool`
  - Task 4: `identity.ReadIdentities(path string) (map[string]identity.Identity, error)`, `type identity.Identity struct { DocID, Relative, ContentHash string; Revision int }` (genutzt: `Relative`, `ContentHash`). Zusicherung: jedes gescheiterte `os.Stat` des Registers zählt wie `Path.exists()` als fehlendes Register und ergibt eine leere Map ohne Fehler; der Bereich `empty` in `TestLinesNameDocumentsTheEngineCannotReturn` baut darauf.
  - Task 5: `graph.ReadGraph(area config.Area, stateDir string) (*graph.Graph, error)`, `graph.Graph.Links` vom Typ `graph.LinksInfo{Total int; Resolved int; Dropped map[string]int}`
  - Task 8: `search.ReadLastRun(stateDir string) (time.Time, bool, error)`, `search.Stale(stamp, now time.Time) bool`, ``const search.ReconcileAdvice = "run `brain reconcile`"``, `search.CollectionName(scope string) string`, `type search.SearchPort interface`, davon gerufen `Indexed(collection string) ([]string, error)` und `NotYetSearchable() (int, error)`.
  - Task 8, nur in Tests (ub `pkg/search/fake.go`, wortgleich umgezogen): `search.NewFakePort() *search.FakePort`, genutzte Felder `FakePort{Listings map[string]search.ScriptedIndexed; Pending []search.ScriptedPending; Listed []string; SearchableCounts int}`, `type search.ScriptedIndexed struct { Paths []string; Err error }`, `type search.ScriptedPending struct { Count int; Err error }`. Zusicherungen: `NewFakePort` legt `Listings` mit `make` an (ub `fake.go:49-51`), darum schreiben die Tests `port.Listings["kept"] = listing()` ohne eigenes `make`; `Indexed` antwortet auf eine nicht geskriptete Collection mit einem Fehler, `NotYetSearchable` bei leerem `Pending` mit `0, nil`.
  - Heutiger Code: `config.ManifestDir(area config.Area, stateDir string) string` (`internal/config/manifest.go:439`), `config.Area` (genutzt: `Scope`, `Path`), `config.Manifest` (genutzt: `IndexInclude`, `IndexUnsearched`, `internal/config/manifest.go:102` und `:104`); in Tests `testlock.Lock(t testing.TB, path string)` (`internal/testlock/lock_windows.go:15`).
- Produces:
```go
package status
func Lines(ch privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) ([]string, error)
```
  Ergänzungen (unexportiert): `lastReconcile(legacyDir string, now time.Time) (string, error)`, `dropped(counts map[string]int) string`, `registerPath(area config.Area, legacyDir string) string`, `unfindable(visible privacy.VisibleArea, port search.SearchPort, legacyDir string) ([]string, error)`, `sharedHashes(areas []privacy.VisibleArea, legacyDir string) ([]string, error)`, `notYetSearchable(port search.SearchPort) []string`.

  Die Zeilenwortlaute, die `Lines` liefert (Bezeichnungen wie im Faktenblatt `status.md` §2; Task 12 zeichnet sie auf, Task 16 liest L1 im Rauchtest). In geschweiften Klammern steht, was eingesetzt wird:
```text
L1a  last reconcile: never; run `brain reconcile`
L1b  last reconcile: {pytext.IsoFormat(stamp)}; older than 24 h, run `brain reconcile`
L1c  last reconcile: {pytext.IsoFormat(stamp)}
L2   {scope}: the search engine sees only {include[0]}; also declared: {include[1:] joined by ", "}
L3   {scope}: {pytext.PathString(runtime.GOOS, area.Path)} does not exist; skipped
L4   {scope}: never indexed; run `brain reindex`
L5   {scope}: only {resolved} of {total} links resolved ({key=value sorted by key, joined by ", "})
L6a  {scope}: {missing} of {ours} indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. {first three joined by ", "}{", " and U+2026 when more than three}
L6b  {scope}: the search engine did not answer ({error}); its index was not compared
L7a  {scope}/{relative}: same content hash as a path excluded by [privacy] never
L7b  same content hash under {n} paths: {scope/relative sorted, joined by ", "}
L8a  {pending} documents are indexed but not yet searchable; run `brain embed`
L8b  the search engine did not answer ({error}); the backlog was not counted
```

Kein Umzug: das Paket entsteht neu, test-first in drei Runden (L1 und L8, dann L2–L5, dann L6 und L7).

**Was die Referenz entscheidet** (core.py:328-532, Faktenblatt `status.md` §2–§6, nachgerechnet und gemessen):

- **Reihenfolge ist Auswertungsreihenfolge.** Python liest den Stempel vor der Registry, je Bereich das `graph.json` vor dem Register, danach alle Register ein zweites Mal für `_shared_hashes` und fragt den Rückstand zuletzt. Welcher von zwei Defekten abbricht, folgt daraus — gemessen: kaputtes `graph.json` in `two` schlägt ein kaputtes Register im übersprungenen `one` davor (keine Listung, kein Rückstand); ein kaputtes Register im übersprungenen `one` bricht erst nach der Listung von `two` ab; ein kaputtes Register in einem nicht übersprungenen Bereich bricht vor dessen Listung ab. `Lines` liest die Register deshalb wie Python zweimal.
- **L3 ist nur für schreibgeschützte Bereiche erreichbar.** Ein schreibbarer Bereich trägt sein Manifest unter dem eigenen Pfad; fehlt der Pfad, fehlt das Manifest, und `VisibleAreas` bricht ab (Python ebenso: `FileNotFoundError`). `Path.exists()` ist `os.path.exists` (gemessen: `inspect.getsource(pathlib.Path.exists)`), also zählt jedes gescheiterte `os.Stat` als „fehlt".
- **Pfadschreibweise:** `{area.path}` in L3 und der Registerpfad in den Fehlern des Registers sind `str(Path)`; `pytext.PathString(runtime.GOOS, path)` bildet beides. Gemessen: `<welt>/world1/gone/./beta/` → `<welt>\world1\gone\beta`, wobei `<welt>` das laufwerksabsolute Messverzeichnis ist, in der Registry mit `/` geschrieben.
- **Sortierung:** `sort.Strings` ordnet gültiges UTF-8 nach Codepunkt, genau wie Pythons `sorted` über `str`; `ReadText` (Task 4) garantiert gültiges UTF-8.
- **Port:** jeder Fehler von `Indexed` wird L6b, jeder von `NotYetSearchable` L8b (Vertrag; Python fängt nur `SearchUnavailable`, und jeder Fehler der Task-8-Ports entspricht einem solchen). `Indexed` wird auch bei leerem Register gerufen, `NotYetSearchable` genau einmal.
- **Testwelten:** Registry, Legacy-Verzeichnis und schreibbare Bereiche liegen in drei getrennten `t.TempDir()`, damit ein Vertauschen von `registryDir` und `legacyDir` auffällt. `FakePort` antwortet auf eine nicht geskriptete Listung mit einem Fehler — jeder Bereich, der L6 erreicht, bekommt deshalb eine Listung, sonst stünde eine L6b-Zeile mehr da.

**Messwelten** (an der Referenz mit `core.status(channel=<kanal>, port=brain.search.fake.FakePort(<listungen und rückstand der welt>), state_dir=<welt>)` gemessen; die Tests bauen dieselben Welten in `t.TempDir()` nach, ihre erwarteten Zeilen sind die gemessenen):

| Test | Welt | Ergebnis der Referenz |
|---|---|---|
| `TestLinesNameTheStampAndWhetherItIsADayOld` | Stempel naiv, `+00:00`, `Z`, genau 24 h, 24 h − 1 µs, Zukunft `.255+02:00` | `never` / alt / alt / alt / frisch `2026-09-14T12:00:00.000001+00:00` / frisch `2999-01-01T00:00:00.255000+02:00` |
| `TestLinesMatchTheMeasuredWorlds/world1 local` und `/world1 cloud` | alpha, project/beta (readonly, Pfad fort), gamma (kein Graph), delta (qmd-Fehler), epsilon (`local_only`), Stempel 2000, Rückstand 7 | 13 Zeilen, lokal gelistet `alpha, delta, epsilon`, cloud `alpha, delta` |
| `TestLinesMatchTheMeasuredWorlds/world2 cloud` | alpha, epsilon, Stempel 2999, Rückstand scheitert | 6 Zeilen, gelistet `alpha` |
| `TestLinesAbortOnABrokenGraphBeforeAnyRegister` | one (readonly, Register kaputt), two (`{"edges": []}`) | ``GraphError <repos>\repo-two\graph.json: graph is missing links; delete it and run `brain reindex` ``, gelistet `[]`, Rückstand 0 |
| `TestLinesAbortOnABrokenRegisterOfASkippedArea` | one (readonly, Register kaputt), two (heil) | `IdentityError <legacy>\areas\one\_identities.tsv: line 2: expected 4 tab-separated fields, found 1`, gelistet `['two']`, Rückstand 0 |
| `TestLinesAbortOnABrokenRegisterBeforeAskingTheEngine` | one (Register kaputt), two | `IdentityError <repos>\repo-one\_identities.tsv: line 2: expected 4 tab-separated fields, found 1`, gelistet `[]`, Rückstand 0 |
| `TestLinesNameDocumentsTheEngineCannotReturn` | alpha, project/three, whole, empty | 3 Zeilen, gelistet `alpha, project-three, whole, empty` |

`<repos>` und `<legacy>` stehen für die temporären Verzeichnisse der schreibbaren Bereiche und des Legacy-Verzeichnisses der jeweiligen Welt.

- [ ] **Step 1: Failing Tests, Runde 1 (L1, L8, Abbrüche vor dem ersten Bereich)** — `internal/brain/status/status_test.go` anlegen:

```go
package status

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/testlock"
)

// neverReconciled is the first line of every world without a usable stamp.
const neverReconciled = "last reconcile: never; run `brain reconcile`"

// asked is the moment every test asks at: the stale stamps lie before it and
// the fresh ones after, as in the recorded worlds of Task 12.
func asked() time.Time { return time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC) }

// world keeps the registry, the legacy state directory and the writable
// areas in three temporary directories of their own, so a test notices when
// Lines reads one of them in place of another.
type world struct {
	t        *testing.T
	registry string
	legacy   string
	repos    string
	entries  strings.Builder
}

func newWorld(t *testing.T) *world {
	t.Helper()
	return &world{t: t, registry: t.TempDir(), legacy: t.TempDir(), repos: t.TempDir()}
}

// write puts content at path and creates the directories above it.
func (w *world) write(path, content string) {
	w.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		w.t.Fatal(err)
	}
}

// register appends one [[area]] table. The path goes in with forward
// slashes, the way the real registry spells Windows paths, inside a TOML
// literal string so that no character of it is read as an escape.
func (w *world) register(scope, path string, readonly bool) {
	fmt.Fprintf(&w.entries, "[[area]]\nscope = '%s'\npath = '%s'\nreadonly = %t\n\n",
		scope, filepath.ToSlash(path), readonly)
}

// writable registers scope as a writable area in repos/<name>, writes its
// manifest there and returns the directory, which holds its artefacts too.
func (w *world) writable(scope, name, manifest string) string {
	dir := filepath.Join(w.repos, name)
	w.register(scope, dir, false)
	w.write(filepath.Join(dir, ".brain.toml"), fmt.Sprintf("[area]\nscope = '%s'\n%s", scope, manifest))
	return dir
}

// readOnly registers scope as a read-only area at path, writes its manifest
// into legacy/areas/<flat> and returns that directory, which holds its
// artefacts too.
func (w *world) readOnly(scope, flat, path, manifest string) string {
	dir := filepath.Join(w.legacy, "areas", flat)
	w.register(scope, path, true)
	w.write(filepath.Join(dir, ".brain.toml"), fmt.Sprintf("[area]\nscope = '%s'\n%s", scope, manifest))
	return dir
}

func (w *world) stamp(content string) {
	w.write(filepath.Join(w.legacy, "maintenance", "last-run.txt"), content)
}

// graph writes a graph.json without edges; dropped is the inside of the
// reasons object.
func (w *world) graph(dir string, total, resolved int, dropped string) {
	w.write(filepath.Join(dir, "graph.json"), fmt.Sprintf(
		`{"edges": [], "links": {"total": %d, "resolved": %d, "dropped": {%s}}}`, total, resolved, dropped))
}

// identities writes a register: the header, then the rows.
func (w *world) identities(dir string, rows ...string) {
	w.write(filepath.Join(dir, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\n"+strings.Join(rows, ""))
}

func row(docID, relative, hash string) string {
	return docID + "\t" + relative + "\t" + hash + "\t1\n"
}

// lines writes the registry as registered so far and asks.
func (w *world) lines(ch privacy.Channel, port search.SearchPort) ([]string, error) {
	w.t.Helper()
	w.write(filepath.Join(w.registry, "registry.toml"), w.entries.String())
	return Lines(ch, port, w.registry, w.legacy, asked())
}

func listing(paths ...string) search.ScriptedIndexed {
	return search.ScriptedIndexed{Paths: paths}
}

func expect(t *testing.T, got []string, err error, want ...string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines\n got %q\nwant %q", got, want)
	}
}

// refused asserts an abort: an error, no lines, no backlog asked, and the
// engine asked for exactly the listings named. It hands the error back for a
// check of its wording.
func refused(t *testing.T, got []string, err error, port *search.FakePort, listed ...string) error {
	t.Helper()
	if err == nil || got != nil {
		t.Fatalf("got %q, %v; want an error and no lines", got, err)
	}
	if !reflect.DeepEqual(port.Listed, listed) || port.SearchableCounts != 0 {
		t.Fatalf("engine asked for %q and the backlog %d times; want %q and never",
			port.Listed, port.SearchableCounts, listed)
	}
	return err
}

func TestLinesNameAMissingStampNever(t *testing.T) {
	w := newWorld(t)
	port := search.NewFakePort()
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled)
	if port.SearchableCounts != 1 {
		t.Fatalf("backlog asked %d times, want once", port.SearchableCounts)
	}
}

func TestLinesNameTheStampAndWhetherItIsADayOld(t *testing.T) {
	// Measured at the reference: datetime.fromisoformat(s).isoformat() and
	// now - stamp >= timedelta(hours=24) with now = 2026-09-15T12:00:00+00:00.
	for _, c := range []struct{ stamp, line string }{
		{"2000-01-01T00:00:00\n", neverReconciled},
		{"2000-01-01T00:00:00+00:00\n", "last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`"},
		{"2000-01-01T00:00:00Z\n", "last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`"},
		{"2026-09-14T12:00:00+00:00\n", "last reconcile: 2026-09-14T12:00:00+00:00; older than 24 h, run `brain reconcile`"},
		{"2026-09-14T12:00:00.000001+00:00\n", "last reconcile: 2026-09-14T12:00:00.000001+00:00"},
		{"2999-01-01T00:00:00.255+02:00\n", "last reconcile: 2999-01-01T00:00:00.255000+02:00"},
	} {
		w := newWorld(t)
		w.stamp(c.stamp)
		got, err := w.lines(privacy.ChannelLocal, search.NewFakePort())
		expect(t, got, err, c.line)
	}
}

func TestLinesAbortWhenTheStampCannotBeRead(t *testing.T) {
	w := newWorld(t)
	w.stamp("2000-01-01T00:00:00+00:00\n")
	testlock.Lock(t, filepath.Join(w.legacy, "maintenance", "last-run.txt"))
	port := search.NewFakePort()
	got, err := w.lines(privacy.ChannelLocal, port)
	refused(t, got, err, port)
}

func TestLinesAbortWithoutARegistry(t *testing.T) {
	port := search.NewFakePort()
	got, err := Lines(privacy.ChannelLocal, port, t.TempDir(), t.TempDir(), asked())
	refused(t, got, err, port)
}

func TestLinesAbortWhenAWritableAreaIsGone(t *testing.T) {
	// A writable area keeps its manifest under its own path, so a path that
	// is gone is a manifest that is missing, and the reference aborts on it.
	w := newWorld(t)
	w.register("alpha", filepath.Join(w.repos, "gone"), false)
	port := search.NewFakePort()
	got, err := w.lines(privacy.ChannelLocal, port)
	refused(t, got, err, port)
}

func TestLinesCountTheBacklogOnce(t *testing.T) {
	for _, c := range []struct {
		pending search.ScriptedPending
		want    []string
	}{
		{search.ScriptedPending{Count: 7}, []string{"7 documents are indexed but not yet searchable; run `brain embed`"}},
		{search.ScriptedPending{Err: errors.New("qmd exited with 1: boom")},
			[]string{"the search engine did not answer (qmd exited with 1: boom); the backlog was not counted"}},
		{search.ScriptedPending{Count: 0}, nil},
	} {
		w := newWorld(t)
		port := search.NewFakePort()
		port.Pending = []search.ScriptedPending{c.pending}
		got, err := w.lines(privacy.ChannelLocal, port)
		expect(t, got, err, append([]string{neverReconciled}, c.want...)...)
		if port.SearchableCounts != 1 {
			t.Fatalf("backlog asked %d times, want once", port.SearchableCounts)
		}
	}
}
```

- [ ] **Step 2: Scheitern sehen**

```bash
go test ./internal/brain/status/
```
Expected (gemessen 2026-09-15 im Wegwerfmodul: der committete Baum, `status_test.go` aus Step 1 und Stümpfe für die Pakete aus Task 3 und Task 8; die Zeilen hängen nur an `status_test.go`, Zeile 1 ist `package status`):
```
# github.com/xidus90/loomux/internal/brain/status [github.com/xidus90/loomux/internal/brain/status.test]
internal\brain\status\status_test.go:103:9: undefined: Lines
internal\brain\status\status_test.go:174:14: undefined: Lines
FAIL	github.com/xidus90/loomux/internal/brain/status [build failed]
FAIL
```

- [ ] **Step 3: Implementieren, Runde 1** — `internal/brain/status/status.go` anlegen:

```go
// Package status answers `brain status`: one line per thing a reader should
// know before trusting an answer, rule for rule after `core.status` of the
// Python reference (src/brain/core.py:328-532). Every line but the first
// reports a defect, so a vault in order answers with its reconcile stamp
// alone.
package status

import (
	"fmt"
	"time"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
)

// Lines answers `brain status` on one channel. The registry comes from
// registryDir; the reconcile stamp and the artefacts and manifests of
// read-only areas come from legacyDir, ultra-brain's state directory, until
// stage 3 moves reconcile to Go.
//
// The steps run in the order Python evaluates them, and that order decides
// more than the order of the lines: of two defects, the one the reference
// meets first is the one that aborts. So the stamp is read before the
// registry, and the backlog is asked last.
func Lines(ch privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) ([]string, error) {
	first, err := lastReconcile(legacyDir, now)
	if err != nil {
		return nil, err
	}
	lines := []string{first}
	if _, err := privacy.VisibleAreas(registryDir, legacyDir, "all", ch); err != nil {
		return nil, err
	}
	return append(lines, notYetSearchable(port)...), nil
}

// lastReconcile is `_last_reconcile`: the stamp is named whether or not it
// is old, because "when was this last checked" has no in-order value to be
// silent about. A missing, unparsable or naive stamp is one answer.
func lastReconcile(legacyDir string, now time.Time) (string, error) {
	stamp, ok, err := search.ReadLastRun(legacyDir)
	if err != nil {
		return "", err
	}
	if !ok {
		return "last reconcile: never; " + search.ReconcileAdvice, nil
	}
	if search.Stale(stamp, now) {
		return fmt.Sprintf("last reconcile: %s; older than 24 h, %s", pytext.IsoFormat(stamp), search.ReconcileAdvice), nil
	}
	return "last reconcile: " + pytext.IsoFormat(stamp), nil
}

// notYetSearchable is `_not_yet_searchable`: the engine's backlog, asked
// once for everything it holds, and a line instead of an abort when it does
// not answer.
func notYetSearchable(port search.SearchPort) []string {
	pending, err := port.NotYetSearchable()
	if err != nil {
		return []string{fmt.Sprintf("the search engine did not answer (%v); the backlog was not counted", err)}
	}
	if pending == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%d documents are indexed but not yet searchable; run `brain embed`", pending)}
}
```

- [ ] **Step 4: Bestehen sehen**

```bash
go test ./internal/brain/status/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/status`. Scheitert `TestLinesNameTheStampAndWhetherItIsADayOld` an der `Z`-Zeile oder am `.255`-Bruch, liest `search.ReadLastRun` (Task 8 über `pytext.ParseAwareIsoFormat`) eine Form nicht, die Python annimmt — das ist ein Befund gegen Task 2/8, nicht gegen diesen Test.

- [ ] **Step 5: Failing Tests, Runde 2 (L2–L5, Graphfehler)** — an `internal/brain/status/status_test.go` anhängen (die Importe aus Step 1 decken sie):

```go
func TestLinesNameIncludeGlobsTheEngineDoesNotSee(t *testing.T) {
	w := newWorld(t)
	for _, a := range []struct{ scope, include string }{
		{"one", `["docs/**/*.md"]`},
		{"two", `["*.md", "extra/*.md"]`},
		{"three", `["docs/**/*.md", "README*.md", "notes/*.md"]`},
	} {
		w.graph(w.writable(a.scope, a.scope, "[index]\ninclude = "+a.include+"\n"), 0, 0, "")
	}
	port := search.NewFakePort()
	port.Listings = map[string]search.ScriptedIndexed{"one": listing(), "two": listing(), "three": listing()}
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"two: the search engine sees only *.md; also declared: extra/*.md",
		"three: the search engine sees only docs/**/*.md; also declared: README*.md, notes/*.md")
}

func TestLinesSkipAReadOnlyAreaWhosePathIsGone(t *testing.T) {
	// The registry spells the path with a `.` segment and a trailing slash;
	// str(Path) drops both and turns the separators.
	w := newWorld(t)
	w.readOnly("project/beta", "project-beta", filepath.ToSlash(w.repos)+"/gone/./beta/",
		"[index]\ninclude = [\"*.md\", \"extra/*.md\"]\n")
	port := search.NewFakePort()
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"project/beta: the search engine sees only *.md; also declared: extra/*.md",
		"project/beta: "+filepath.Join(w.repos, "gone", "beta")+" does not exist; skipped")
	if len(port.Listed) != 0 {
		t.Fatalf("engine asked for %q; a skipped area is not compared", port.Listed)
	}
}

func TestLinesSkipAnAreaThatWasNeverIndexed(t *testing.T) {
	// A read-only area's graph lies in the legacy directory: `kept` has one
	// there and none in the area, `bare` one in the area and none there. Only
	// `kept` has a listing scripted; an unscripted listing would add a line.
	w := newWorld(t)
	w.writable("gamma", "repo-gamma", "")
	w.graph(w.readOnly("kept", "kept", w.repos, ""), 0, 0, "")
	w.readOnly("bare", "bare", w.repos, "")
	w.graph(w.repos, 0, 0, "")
	port := search.NewFakePort()
	port.Listings["kept"] = listing()
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"gamma: never indexed; run `brain reindex`",
		"bare: never indexed; run `brain reindex`")
}

func TestLinesReportLinksResolvedForLessThanHalf(t *testing.T) {
	for _, c := range []struct {
		total, resolved int
		dropped         string
		want            []string
	}{
		{5, 2, `"unknown_target": 2, "external": 1`, []string{"alpha: only 2 of 5 links resolved (external=1, unknown_target=2)"}},
		{4, 1, ``, []string{"alpha: only 1 of 4 links resolved ()"}},
		{4, 2, `"external": 2`, nil},
		{0, 0, ``, nil},
	} {
		w := newWorld(t)
		w.graph(w.writable("alpha", "repo-alpha", ""), c.total, c.resolved, c.dropped)
		port := search.NewFakePort()
		port.Listings["alpha"] = listing()
		got, err := w.lines(privacy.ChannelLocal, port)
		expect(t, got, err, append([]string{neverReconciled}, c.want...)...)
	}
}

func TestLinesAbortOnABrokenGraphBeforeAnyRegister(t *testing.T) {
	// The skipped area `one` carries a broken register as well; the graph of
	// `two` is met first, as in the reference.
	w := newWorld(t)
	one := w.readOnly("one", "one", filepath.ToSlash(w.repos)+"/gone", "")
	w.write(filepath.Join(one, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\nbroken\n")
	two := w.writable("two", "repo-two", "")
	w.write(filepath.Join(two, "graph.json"), "{\"edges\": []}\n")
	port := search.NewFakePort()
	port.Listings["two"] = listing()
	got, err := w.lines(privacy.ChannelLocal, port)
	err = refused(t, got, err, port)
	if want := filepath.Join(two, "graph.json") + ": graph is missing links; delete it and run `brain reindex`"; err.Error() != want {
		t.Fatalf("error\n got %q\nwant %q", err, want)
	}
}
```

- [ ] **Step 6: Scheitern sehen**

```bash
go test ./internal/brain/status/ 2>&1 | grep -E '^ *--- FAIL: ' | sed 's/ (.*//'
```
Expected (Go führt die Tests in Quelltextreihenfolge aus; die Dauer in Klammern schneidet `sed` ab):
```
--- FAIL: TestLinesNameIncludeGlobsTheEngineDoesNotSee
--- FAIL: TestLinesSkipAReadOnlyAreaWhosePathIsGone
--- FAIL: TestLinesSkipAnAreaThatWasNeverIndexed
--- FAIL: TestLinesReportLinksResolvedForLessThanHalf
--- FAIL: TestLinesAbortOnABrokenGraphBeforeAnyRegister
```
Die ersten vier scheitern an `expect`, weil Runde 1 nur die Zeile `neverReconciled` liefert; der fünfte an `refused`, weil Runde 1 kein `graph.json` liest und darum keinen Fehler meldet. Steht zusätzlich ein Test aus Step 1 in der Liste, ist Runde 1 beschädigt: erst reparieren.

- [ ] **Step 7: Implementieren, Runde 2** — `internal/brain/status/status.go` vollständig ersetzen:

```go
// Package status answers `brain status`: one line per thing a reader should
// know before trusting an answer, rule for rule after `core.status` of the
// Python reference (src/brain/core.py:328-532). Every line but the first
// reports a defect, so a vault in order answers with its reconcile stamp
// alone.
package status

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// Lines answers `brain status` on one channel. The registry comes from
// registryDir; the reconcile stamp and the artefacts and manifests of
// read-only areas come from legacyDir, ultra-brain's state directory, until
// stage 3 moves reconcile to Go.
//
// The steps run in the order Python evaluates them, and that order decides
// more than the order of the lines: of two defects, the one the reference
// meets first is the one that aborts. So the stamp is read before the
// registry, each area's graph in registry order, and the backlog is asked
// last.
func Lines(ch privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) ([]string, error) {
	first, err := lastReconcile(legacyDir, now)
	if err != nil {
		return nil, err
	}
	lines := []string{first}
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, "all", ch)
	if err != nil {
		return nil, err
	}
	for _, visible := range areas {
		area, manifest := visible.Area, visible.Manifest
		if len(manifest.IndexInclude) > 1 {
			lines = append(lines, fmt.Sprintf("%s: the search engine sees only %s; also declared: %s",
				area.Scope, manifest.IndexInclude[0], strings.Join(manifest.IndexInclude[1:], ", ")))
		}
		// Path.exists() is os.path.exists, which calls every failed stat
		// absent. Only a read-only area gets here with its path gone: a
		// writable one keeps its manifest under that path, and VisibleAreas
		// has refused it already.
		if _, err := os.Stat(area.Path); err != nil {
			lines = append(lines, fmt.Sprintf("%s: %s does not exist; skipped",
				area.Scope, pytext.PathString(runtime.GOOS, area.Path)))
			continue
		}
		if _, err := os.Stat(filepath.Join(config.ManifestDir(area, legacyDir), "graph.json")); err != nil {
			lines = append(lines, fmt.Sprintf("%s: never indexed; run `brain reindex`", area.Scope))
			continue
		}
		g, err := graph.ReadGraph(area, legacyDir)
		if err != nil {
			return nil, err
		}
		if links := g.Links; links.Total != 0 && links.Resolved*2 < links.Total {
			lines = append(lines, fmt.Sprintf("%s: only %d of %d links resolved (%s)",
				area.Scope, links.Resolved, links.Total, dropped(links.Dropped)))
		}
	}
	return append(lines, notYetSearchable(port)...), nil
}

// lastReconcile is `_last_reconcile`: the stamp is named whether or not it
// is old, because "when was this last checked" has no in-order value to be
// silent about. A missing, unparsable or naive stamp is one answer.
func lastReconcile(legacyDir string, now time.Time) (string, error) {
	stamp, ok, err := search.ReadLastRun(legacyDir)
	if err != nil {
		return "", err
	}
	if !ok {
		return "last reconcile: never; " + search.ReconcileAdvice, nil
	}
	if search.Stale(stamp, now) {
		return fmt.Sprintf("last reconcile: %s; older than 24 h, %s", pytext.IsoFormat(stamp), search.ReconcileAdvice), nil
	}
	return "last reconcile: " + pytext.IsoFormat(stamp), nil
}

// dropped renders the reasons table of L5 as Python joins
// `sorted(dropped_counts.items())`: by key, `key=value`, and nothing at all
// between the parentheses when the table is empty.
func dropped(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = fmt.Sprintf("%s=%d", key, counts[key])
	}
	return strings.Join(parts, ", ")
}

// notYetSearchable is `_not_yet_searchable`: the engine's backlog, asked
// once for everything it holds, and a line instead of an abort when it does
// not answer.
func notYetSearchable(port search.SearchPort) []string {
	pending, err := port.NotYetSearchable()
	if err != nil {
		return []string{fmt.Sprintf("the search engine did not answer (%v); the backlog was not counted", err)}
	}
	if pending == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%d documents are indexed but not yet searchable; run `brain embed`", pending)}
}
```

- [ ] **Step 8: Bestehen sehen**

```bash
go test ./internal/brain/status/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/status`. Scheitert `TestLinesAbortOnABrokenGraphBeforeAnyRegister` am Wortlaut, spricht `graph.ReadGraph` (Task 5) nicht wie Pythons `_graph` — Befund gegen Task 5.

- [ ] **Step 9: Failing Tests, Runde 3 (L6, L7, Registerfehler, gemessene Welten)** — an `internal/brain/status/status_test.go` anhängen:

```go
// collectionNotFound is what the CLI port hands on for `qmd ls` of a
// collection qmd 2.8.3 does not know (measured 2026-09-15): stderr stripped
// at both ends, one newline left inside.
const collectionNotFound = "qmd exited with 1: Collection not found: delta\nRun 'qmd ls' to see available collections."

// One file name in both normal forms, and the ellipsis of L6a, spelt by code
// point so that no editor or copy can normalise them into each other.
const (
	nfdCafe  = "Cafe" + string(rune(0x0301)) + ".md"
	nfcCafe  = "Caf" + string(rune(0x00e9)) + ".md"
	ellipsis = string(rune(0x2026))
)

func TestLinesNameDocumentsTheEngineCannotReturn(t *testing.T) {
	// alpha: `never` and `unsearched` stay out of both counts, the register's
	// NFD name is the engine's NFC one, four missing end in ", " + ellipsis.
	// project/three: exactly three missing, no tail; its collection is the
	// flat scope. whole: nothing missing. empty: no register, asked anyway.
	w := newWorld(t)
	alpha := w.writable("alpha", "repo-alpha", "[index]\nunsearched = [\"drafts/**\"]\n\n[privacy]\nnever = [\"secret/**\"]\n")
	w.graph(alpha, 0, 0, "")
	w.identities(alpha, row("01", "a.md", "sha256:1"), row("02", "b.md", "sha256:2"), row("03", "c.md", "sha256:3"),
		row("04", "d.md", "sha256:4"), row("05", "e.md", "sha256:5"), row("06", "secret/x.md", "sha256:6"),
		row("07", "drafts/y.md", "sha256:7"), row("08", nfdCafe, "sha256:8"))
	three := w.writable("project/three", "repo-three", "")
	w.graph(three, 0, 0, "")
	w.identities(three, row("11", "c.md", "sha256:a"), row("12", "a.md", "sha256:b"), row("13", "b.md", "sha256:c"))
	whole := w.writable("whole", "repo-whole", "")
	w.graph(whole, 0, 0, "")
	w.identities(whole, row("21", "x.md", "sha256:d"))
	w.graph(w.writable("empty", "repo-empty", ""), 0, 0, "")
	port := search.NewFakePort()
	port.Listings = map[string]search.ScriptedIndexed{
		"alpha":         listing("b.md", nfcCafe),
		"project-three": listing(),
		"whole":         listing("x.md"),
		"empty":         listing(),
	}
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"alpha: 4 of 6 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. a.md, c.md, d.md, "+ellipsis,
		"project/three: 3 of 3 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. a.md, b.md, c.md")
	if want := []string{"alpha", "project-three", "whole", "empty"}; !reflect.DeepEqual(port.Listed, want) {
		t.Fatalf("engine asked for %q, want %q", port.Listed, want)
	}
}

func TestLinesSayWhenTheEngineDoesNotList(t *testing.T) {
	w := newWorld(t)
	w.graph(w.writable("delta", "repo-delta", ""), 0, 0, "")
	port := search.NewFakePort()
	port.Listings["delta"] = search.ScriptedIndexed{Err: errors.New(collectionNotFound)}
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"delta: the search engine did not answer (qmd exited with 1: Collection not found: delta\nRun 'qmd ls' to see available collections.); its index was not compared")
}

func TestLinesAbortOnABrokenRegisterBeforeAskingTheEngine(t *testing.T) {
	w := newWorld(t)
	one := w.writable("one", "repo-one", "")
	w.graph(one, 0, 0, "")
	w.write(filepath.Join(one, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\nbroken\n")
	w.graph(w.writable("two", "repo-two", ""), 0, 0, "")
	port := search.NewFakePort()
	port.Listings = map[string]search.ScriptedIndexed{"one": listing(), "two": listing()}
	got, err := w.lines(privacy.ChannelLocal, port)
	err = refused(t, got, err, port)
	if want := filepath.Join(one, "_identities.tsv") + ": line 2: expected 4 tab-separated fields, found 1"; err.Error() != want {
		t.Fatalf("error\n got %q\nwant %q", err, want)
	}
}

func TestLinesNameContentKeptUnderSeveralPaths(t *testing.T) {
	// Lines follow the hash, not the path. project/beta (path gone) and gamma
	// (never indexed) are skipped by the loop and still counted here. `222`
	// lost its pair to `never`; `999` is held by withheld paths only.
	w := newWorld(t)
	alpha := w.writable("alpha", "repo-alpha", "[privacy]\nnever = [\"secret/**\"]\n")
	w.graph(alpha, 0, 0, "")
	w.identities(alpha, row("01", "a.md", "sha256:222"), row("02", "b.md", "sha256:111"), row("03", "c.md", "sha256:333"),
		row("06", "secret/x.md", "sha256:222"), row("09", "secret/w1.md", "sha256:999"), row("10", "secret/w2.md", "sha256:999"))
	beta := w.readOnly("project/beta", "project-beta", filepath.ToSlash(w.repos)+"/gone", "")
	w.identities(beta, row("11", "z.md", "sha256:111"))
	w.identities(w.writable("gamma", "repo-gamma", ""), row("21", "g.md", "sha256:333"))
	delta := w.writable("delta", "repo-delta", "")
	w.graph(delta, 0, 0, "")
	w.identities(delta, row("31", "a2.md", "sha256:333"))
	port := search.NewFakePort()
	port.Listings = map[string]search.ScriptedIndexed{"alpha": listing("a.md", "b.md", "c.md"), "delta": listing("a2.md")}
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"project/beta: "+filepath.Join(w.repos, "gone")+" does not exist; skipped",
		"gamma: never indexed; run `brain reindex`",
		"same content hash under 2 paths: alpha/b.md, project/beta/z.md",
		"alpha/a.md: same content hash as a path excluded by [privacy] never",
		"same content hash under 3 paths: alpha/c.md, delta/a2.md, gamma/g.md")
}

func TestLinesAbortOnABrokenRegisterOfASkippedArea(t *testing.T) {
	// The loop skips `one` and compares `two`; the register of `one` is read
	// for the shared hashes afterwards, before the backlog is asked.
	w := newWorld(t)
	one := w.readOnly("one", "one", filepath.ToSlash(w.repos)+"/gone", "")
	w.write(filepath.Join(one, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\nbroken\n")
	w.graph(w.writable("two", "repo-two", ""), 0, 0, "")
	port := search.NewFakePort()
	port.Listings["two"] = listing()
	got, err := w.lines(privacy.ChannelLocal, port)
	err = refused(t, got, err, port, "two")
	if want := filepath.Join(one, "_identities.tsv") + ": line 2: expected 4 tab-separated fields, found 1"; err.Error() != want {
		t.Fatalf("error\n got %q\nwant %q", err, want)
	}
}

// alphaArea is area alpha of the measured worlds: three include globs, a
// `never` and an `unsearched` glob, two of five links resolved, and a
// register the engine lists two entries of.
func alphaArea(w *world) {
	dir := w.writable("alpha", "repo-alpha", "[index]\n"+
		"include = [\"docs/**/*.md\", \"README*.md\", \"notes/*.md\"]\n"+
		"unsearched = [\"drafts/**\"]\n\n[privacy]\nnever = [\"secret/**\"]\n")
	w.graph(dir, 5, 2, `"unknown_target": 2, "external": 1`)
	w.identities(dir, row("01", "a.md", "sha256:222"), row("02", "b.md", "sha256:111"), row("03", "c.md", "sha256:333"),
		row("04", "d.md", "sha256:444"), row("05", "e.md", "sha256:555"), row("06", "secret/x.md", "sha256:222"),
		row("07", "drafts/y.md", "sha256:666"), row("08", nfdCafe, "sha256:777"),
		row("09", "secret/w1.md", "sha256:999"), row("10", "secret/w2.md", "sha256:999"))
}

// epsilonArea is a `local_only` area with an empty graph and no register.
func epsilonArea(w *world) {
	w.graph(w.writable("epsilon", "repo-epsilon", "\n[privacy]\nmode = \"local_only\"\n"), 0, 0, "")
}

func TestLinesMatchTheMeasuredWorlds(t *testing.T) {
	first := func(t *testing.T, ch privacy.Channel, listed ...string) {
		w := newWorld(t)
		w.stamp("2000-01-01T00:00:00+00:00\n")
		alphaArea(w)
		beta := w.readOnly("project/beta", "project-beta", filepath.ToSlash(w.repos)+"/gone/./beta/",
			"[index]\ninclude = [\"*.md\", \"extra/*.md\"]\n")
		w.identities(beta, row("11", "z.md", "sha256:111"))
		w.identities(w.writable("gamma", "repo-gamma", ""), row("21", "g.md", "sha256:333"))
		delta := w.writable("delta", "repo-delta", "")
		w.graph(delta, 4, 1, "")
		w.identities(delta, row("31", "a2.md", "sha256:333"))
		epsilonArea(w)
		port := search.NewFakePort()
		port.Listings = map[string]search.ScriptedIndexed{
			"alpha":   listing("b.md", nfcCafe),
			"delta":   {Err: errors.New(collectionNotFound)},
			"epsilon": listing(),
		}
		port.Pending = []search.ScriptedPending{{Count: 7}}
		got, err := w.lines(ch, port)
		expect(t, got, err,
			"last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`",
			"alpha: the search engine sees only docs/**/*.md; also declared: README*.md, notes/*.md",
			"alpha: only 2 of 5 links resolved (external=1, unknown_target=2)",
			"alpha: 4 of 6 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. a.md, c.md, d.md, "+ellipsis,
			"project/beta: the search engine sees only *.md; also declared: extra/*.md",
			"project/beta: "+filepath.Join(w.repos, "gone", "beta")+" does not exist; skipped",
			"gamma: never indexed; run `brain reindex`",
			"delta: only 1 of 4 links resolved ()",
			"delta: the search engine did not answer ("+collectionNotFound+"); its index was not compared",
			"same content hash under 2 paths: alpha/b.md, project/beta/z.md",
			"alpha/a.md: same content hash as a path excluded by [privacy] never",
			"same content hash under 3 paths: alpha/c.md, delta/a2.md, gamma/g.md",
			"7 documents are indexed but not yet searchable; run `brain embed`")
		if !reflect.DeepEqual(port.Listed, listed) {
			t.Fatalf("engine asked for %q, want %q", port.Listed, listed)
		}
	}
	t.Run("world1 local", func(t *testing.T) { first(t, privacy.ChannelLocal, "alpha", "delta", "epsilon") })
	t.Run("world1 cloud", func(t *testing.T) { first(t, privacy.ChannelCloud, "alpha", "delta") })
	t.Run("world2 cloud", func(t *testing.T) {
		w := newWorld(t)
		w.stamp("2999-01-01T00:00:00.255+02:00\n")
		alphaArea(w)
		epsilonArea(w)
		port := search.NewFakePort()
		port.Listings = map[string]search.ScriptedIndexed{"alpha": listing("b.md", nfcCafe), "epsilon": listing()}
		port.Pending = []search.ScriptedPending{{Err: errors.New("qmd exited with 1: boom")}}
		got, err := w.lines(privacy.ChannelCloud, port)
		expect(t, got, err,
			"last reconcile: 2999-01-01T00:00:00.255000+02:00",
			"alpha: the search engine sees only docs/**/*.md; also declared: README*.md, notes/*.md",
			"alpha: only 2 of 5 links resolved (external=1, unknown_target=2)",
			"alpha: 4 of 6 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. a.md, c.md, d.md, "+ellipsis,
			"alpha/a.md: same content hash as a path excluded by [privacy] never",
			"the search engine did not answer (qmd exited with 1: boom); the backlog was not counted")
		if want := []string{"alpha"}; !reflect.DeepEqual(port.Listed, want) {
			t.Fatalf("engine asked for %q, want %q", port.Listed, want)
		}
	})
}
```

- [ ] **Step 10: Scheitern sehen**

```bash
go test ./internal/brain/status/ 2>&1 | grep -E '^ *--- FAIL: ' | sed 's/ (.*//'
```
Expected (Quelltextreihenfolge; Go schreibt die Leerzeichen der Untertestnamen als `_`):
```
--- FAIL: TestLinesNameDocumentsTheEngineCannotReturn
--- FAIL: TestLinesSayWhenTheEngineDoesNotList
--- FAIL: TestLinesAbortOnABrokenRegisterBeforeAskingTheEngine
--- FAIL: TestLinesNameContentKeptUnderSeveralPaths
--- FAIL: TestLinesAbortOnABrokenRegisterOfASkippedArea
--- FAIL: TestLinesMatchTheMeasuredWorlds
    --- FAIL: TestLinesMatchTheMeasuredWorlds/world1_local
    --- FAIL: TestLinesMatchTheMeasuredWorlds/world1_cloud
    --- FAIL: TestLinesMatchTheMeasuredWorlds/world2_cloud
```
`TestLinesNameDocumentsTheEngineCannotReturn`, `TestLinesSayWhenTheEngineDoesNotList`, `TestLinesNameContentKeptUnderSeveralPaths` und die drei Untertests scheitern an `expect`, weil Runde 2 weder L6 noch L7 schreibt; die beiden Register-Abbrüche an `refused`, weil Runde 2 kein Register liest und darum keinen Fehler meldet. Steht zusätzlich ein Test aus Runde 1 oder 2 in der Liste, ist eine frühere Runde beschädigt: erst reparieren.

- [ ] **Step 11: Implementieren, Runde 3** — `internal/brain/status/status.go` vollständig ersetzen (Endstand):

```go
// Package status answers `brain status`: one line per thing a reader should
// know before trusting an answer, rule for rule after `core.status` of the
// Python reference (src/brain/core.py:328-532). Every line but the first
// reports a defect, so a vault in order answers with its reconcile stamp
// alone.
package status

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// Lines answers `brain status` on one channel. The registry comes from
// registryDir; the reconcile stamp and the artefacts and manifests of
// read-only areas come from legacyDir, ultra-brain's state directory, until
// stage 3 moves reconcile to Go.
//
// The steps run in the order Python evaluates them, and that order decides
// more than the order of the lines: of two defects, the one the reference
// meets first is the one that aborts. So the stamp is read before the
// registry, an area's graph before its register, every register a second
// time for the shared hashes after the loop, and the backlog is asked last.
func Lines(ch privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) ([]string, error) {
	first, err := lastReconcile(legacyDir, now)
	if err != nil {
		return nil, err
	}
	lines := []string{first}
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, "all", ch)
	if err != nil {
		return nil, err
	}
	for _, visible := range areas {
		area, manifest := visible.Area, visible.Manifest
		if len(manifest.IndexInclude) > 1 {
			lines = append(lines, fmt.Sprintf("%s: the search engine sees only %s; also declared: %s",
				area.Scope, manifest.IndexInclude[0], strings.Join(manifest.IndexInclude[1:], ", ")))
		}
		// Path.exists() is os.path.exists, which calls every failed stat
		// absent. Only a read-only area gets here with its path gone: a
		// writable one keeps its manifest under that path, and VisibleAreas
		// has refused it already.
		if _, err := os.Stat(area.Path); err != nil {
			lines = append(lines, fmt.Sprintf("%s: %s does not exist; skipped",
				area.Scope, pytext.PathString(runtime.GOOS, area.Path)))
			continue
		}
		if _, err := os.Stat(filepath.Join(config.ManifestDir(area, legacyDir), "graph.json")); err != nil {
			lines = append(lines, fmt.Sprintf("%s: never indexed; run `brain reindex`", area.Scope))
			continue
		}
		g, err := graph.ReadGraph(area, legacyDir)
		if err != nil {
			return nil, err
		}
		if links := g.Links; links.Total != 0 && links.Resolved*2 < links.Total {
			lines = append(lines, fmt.Sprintf("%s: only %d of %d links resolved (%s)",
				area.Scope, links.Resolved, links.Total, dropped(links.Dropped)))
		}
		missing, err := unfindable(visible, port, legacyDir)
		if err != nil {
			return nil, err
		}
		lines = append(lines, missing...)
	}
	shared, err := sharedHashes(areas, legacyDir)
	if err != nil {
		return nil, err
	}
	lines = append(lines, shared...)
	return append(lines, notYetSearchable(port)...), nil
}

// lastReconcile is `_last_reconcile`: the stamp is named whether or not it
// is old, because "when was this last checked" has no in-order value to be
// silent about. A missing, unparsable or naive stamp is one answer.
func lastReconcile(legacyDir string, now time.Time) (string, error) {
	stamp, ok, err := search.ReadLastRun(legacyDir)
	if err != nil {
		return "", err
	}
	if !ok {
		return "last reconcile: never; " + search.ReconcileAdvice, nil
	}
	if search.Stale(stamp, now) {
		return fmt.Sprintf("last reconcile: %s; older than 24 h, %s", pytext.IsoFormat(stamp), search.ReconcileAdvice), nil
	}
	return "last reconcile: " + pytext.IsoFormat(stamp), nil
}

// dropped renders the reasons table of L5 as Python joins
// `sorted(dropped_counts.items())`: by key, `key=value`, and nothing at all
// between the parentheses when the table is empty.
func dropped(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = fmt.Sprintf("%s=%d", key, counts[key])
	}
	return strings.Join(parts, ", ")
}

// registerPath is `area_artifact_dir(area, state_dir) / "_identities.tsv"`
// spelt as str(Path) spells it: the register names this path in its error
// messages, and those end up on the reader's screen.
func registerPath(area config.Area, legacyDir string) string {
	return pytext.PathString(runtime.GOOS, config.ManifestDir(area, legacyDir)+"/_identities.tsv")
}

// unfindable is `_unfindable`: documents the register holds and the engine
// does not list. Paths under `never` or `unsearched` are meant to be
// unfindable and stay out of both counts. The engine is asked even for an
// empty register, and an engine that does not answer is a line, not an
// abort. sort.Strings orders valid UTF-8 by code point, as Python's sorted
// orders str.
func unfindable(visible privacy.VisibleArea, port search.SearchPort, legacyDir string) ([]string, error) {
	area, manifest := visible.Area, visible.Manifest
	register, err := identity.ReadIdentities(registerPath(area, legacyDir))
	if err != nil {
		return nil, err
	}
	ours := make([]string, 0, len(register))
	for relative := range register {
		if privacy.IsReadable(manifest, relative) && !privacy.MatchesGlobs(manifest.IndexUnsearched, relative) {
			ours = append(ours, relative)
		}
	}
	sort.Strings(ours)
	listed, err := port.Indexed(search.CollectionName(area.Scope))
	if err != nil {
		return []string{fmt.Sprintf("%s: the search engine did not answer (%v); its index was not compared", area.Scope, err)}, nil
	}
	// Both sides in NFC: a name in NFD and in NFC is one file to the file
	// system and two strings to a comparison. The line keeps the register's
	// own spelling.
	theirs := make(map[string]bool, len(listed))
	for _, relative := range listed {
		theirs[pytext.NFC(relative)] = true
	}
	var missing []string
	for _, relative := range ours {
		if !theirs[pytext.NFC(relative)] {
			missing = append(missing, relative)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	tail := ""
	if len(missing) > 3 {
		tail = ", …" // U+2026, as Python writes it
	}
	return []string{fmt.Sprintf("%s: %d of %d indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. %s%s",
		area.Scope, len(missing), len(ours), strings.Join(missing[:min(3, len(missing))], ", "), tail)}, nil
}

// sharedHashes is `_shared_hashes`: identical bytes under several paths,
// looked for across every visible area, the skipped ones included. A path
// under `never` is counted but never named, so a pair that lost one half to
// it names the survivor alone. Lines follow the hash string, the paths inside
// a line their own order.
func sharedHashes(areas []privacy.VisibleArea, legacyDir string) ([]string, error) {
	byHash := map[string][]string{}
	withheld := map[string]int{}
	for _, visible := range areas {
		register, err := identity.ReadIdentities(registerPath(visible.Area, legacyDir))
		if err != nil {
			return nil, err
		}
		for _, entry := range register {
			if !privacy.IsReadable(visible.Manifest, entry.Relative) {
				withheld[entry.ContentHash]++
				continue
			}
			byHash[entry.ContentHash] = append(byHash[entry.ContentHash], visible.Area.Scope+"/"+entry.Relative)
		}
	}
	digests := make([]string, 0, len(byHash))
	for digest := range byHash {
		digests = append(digests, digest)
	}
	sort.Strings(digests)
	var lines []string
	for _, digest := range digests {
		paths := byHash[digest]
		if len(paths)+withheld[digest] < 2 {
			continue
		}
		if len(paths) == 1 {
			lines = append(lines, paths[0]+": same content hash as a path excluded by [privacy] never")
			continue
		}
		sort.Strings(paths)
		lines = append(lines, fmt.Sprintf("same content hash under %d paths: %s", len(paths), strings.Join(paths, ", ")))
	}
	return lines, nil
}

// notYetSearchable is `_not_yet_searchable`: the engine's backlog, asked
// once for everything it holds, and a line instead of an abort when it does
// not answer.
func notYetSearchable(port search.SearchPort) []string {
	pending, err := port.NotYetSearchable()
	if err != nil {
		return []string{fmt.Sprintf("the search engine did not answer (%v); the backlog was not counted", err)}
	}
	if pending == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%d documents are indexed but not yet searchable; run `brain embed`", pending)}
}
```

Das `…` in `tail` ist das eine Zeichen U+2026 (Bytes `E2 80 A6`), nicht drei Punkte; `TestLinesNameDocumentsTheEngineCannotReturn` prüft es über `string(rune(0x2026))`.

- [ ] **Step 12: Bestehen sehen**

```bash
go test ./internal/brain/status/
```
Expected: `ok  	github.com/xidus90/loomux/internal/brain/status`. Scheitert eine der beiden Register-Abbruchprüfungen am Wortlaut, spricht `identity.ReadIdentities` (Task 4) nicht wie `identity.py` — Befund gegen Task 4.

- [ ] **Step 13: Coverage** (Befehle aus dem Umzugsverfahren, Schritt 6)

```bash
go test ./internal/brain/status/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Expected: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/brain/status` beginnt (Dauer und Prozentsatz über `./...` variieren).

```bash
go tool cover -func="$TEMP/pkg.out" | grep internal/brain/status/
```
Expected: jede Funktion des Pakets 100.0%. Die Zeilennummern sind aus dem Block in Step 11 gezählt (Zeile 1 ist der Paketkommentar) und variieren mit dem Kommentar:
```
github.com/xidus90/loomux/internal/brain/status/status.go:35:	Lines			100.0%
github.com/xidus90/loomux/internal/brain/status/status.go:89:	lastReconcile		100.0%
github.com/xidus90/loomux/internal/brain/status/status.go:106:	dropped			100.0%
github.com/xidus90/loomux/internal/brain/status/status.go:122:	registerPath		100.0%
github.com/xidus90/loomux/internal/brain/status/status.go:132:	unfindable		100.0%
github.com/xidus90/loomux/internal/brain/status/status.go:178:	sharedHashes		100.0%
github.com/xidus90/loomux/internal/brain/status/status.go:218:	notYetSearchable	100.0%
```
Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund. Für dieses Paket ist keine Ausnahme vorgesehen.

- [ ] **Step 14: Format und vet**

```bash
gofmt -l internal/brain/status
```
Expected: keine Ausgabe.

```bash
go vet ./internal/brain/status/
```
Expected: keine Ausgabe.

- [ ] **Step 15: Stagen und Tor**

Erst stagen, dann das Tor: `.githooks/pre-commit` bricht bei ungetrackten oder ungestagten `*.go` ab.

```bash
git add internal/brain/status/status.go internal/brain/status/status_test.go
```
Expected: Exit 0, keine Ausgabe außer einer möglichen Zeilenende-Warnung von Git.

```bash
sh .githooks/pre-commit
```
Expected: Exit 0 (`covergate` ohne Befund, `bin/loomux.exe` neu gebaut).

- [ ] **Step 16: Commit**

```bash
git branch --show-current
```
Expected: `sdd-1b-1` (Hauptcheckout: `master`).

```bash
git rev-parse --short HEAD
```
Expected: der Kurzhash des Commits aus Task 8.

```bash
git diff --cached --stat
```
Expected: genau `internal/brain/status/status.go` und `internal/brain/status/status_test.go`, Summenzeile `2 files changed` mit den Einfügungen.

```bash
printf 'Add the brain status lines after core.status\n\nThe reconcile stamp, include globs the engine does not see, missing\npaths, unindexed areas, unresolved links, documents the engine cannot\nreturn, shared content hashes and the embedding backlog, in the order\nthe Python reference evaluates them.\n' > "$TEMP/loomux-commit-09.txt"
```
Expected: keine Ausgabe.

```bash
git commit -F "$TEMP/loomux-commit-09.txt"
```
Expected: Exit 0. Der Commit löst das Tor noch einmal aus; das ist gewollt. Danach steht die Zeile `[sdd-1b-1 <kurzhash>] Add the brain status lines after core.status` (Hauptcheckout: `[master <kurzhash>]`) und `2 files changed`.

```bash
git log -1 --format='%an <%ae>'
```
Expected: die Nutzeridentität aus `git config user.name` und `git config user.email`, keine Modellnennung.

```bash
git log -1 --format=%B
```
Expected: genau die Nachricht aus `$TEMP/loomux-commit-09.txt`, keine `Co-Authored-By`-Zeile.

- [ ] **Step 17: Bericht**

Der Bericht nennt:
- Paritätszeile (für Task 12): `status` bricht bei unlesbarem Stempel, fehlender Registry, unlesbarem `graph.json` oder unlesbarem Register wie die Referenz mit Exit 1 ab, nennt den Grund aber im Wortlaut von Gos Betriebssystemfehlern statt Pythons `[Errno 13] Permission denied: '<pfad>'` bzw. `[Errno 2] No such file or directory: '<pfad>'`.
- Keine weitere Abweichung: alle Zeilenarten L1a–L8b, ihre Reihenfolge und die Abbruchreihenfolge sind an der Referenz gemessen und in den Tests festgehalten.
- Abhängige Wortlaute, die diese Tests mitprüfen: `graph.ReadGraph` (Task 5, ``<pfad>: graph is missing links; delete it and run `brain reindex` ``), `identity.ReadIdentities` (Task 4, `<pfad>: line 2: expected 4 tab-separated fields, found 1`), `search.ReadLastRun` (Task 8, nimmt `Z` und Brüche mit 3 und 6 Stellen an).
- Die Zahl der Tests: 17 Testfunktionen, davon eine mit drei Untertests.

---

### Task 10: `loomux brain` in `internal/cli`

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 9 sind committet.

**Files:**
- Create: `internal/cli/brainargs.go`, Test `internal/cli/brainargs_test.go`
- Create: `internal/cli/brain.go`, Test `internal/cli/brain_test.go`
- Modify: `internal/cli/commands.go` (`"brain": brainCommand,`)

Die READMEs pflegt Task 16 (beide Sprachen, die fünf `loomux brain`-Befehle als aktive Befehle); dieser Task fasst sie nicht an.

**Interfaces:**
- Consumes:
  - Task 1: `config.LegacyBrainDirUntilStage3() string`; `config.ReadAreaManifestUntilStage4(dir string) (*config.Manifest, error)`, mittelbar über `privacy.VisibleAreas`. Zusicherung: `.loomux/config.toml` ist der erste Name in Task 1s Namensliste `manifestNamesUntilStage4`, und eine solche Datei mit `[area]`-Tabelle und Scope deklariert den Bereich. `brainWorld` legt das Manifest unter `<area>/.loomux/config.toml` ab, `brainReadOnlyWorld` unter `<legacy>/areas/project-r/.loomux/config.toml`.
  - Heutiger Code: `config.StateDir() string` (`internal/config/registry.go:141`), `config.Area`, `config.Manifest`; der Typ `command` (`internal/cli/cli.go:19`) und die Map `commands` (`internal/cli/commands.go:4`); in Tests `run(args ...string) (int, string, string)` (`internal/cli/cli_test.go:10`) und `writeFile(t *testing.T, path, body string) string` (`internal/cli/wiki_test.go:13`).
  - Task 2 (Ergänzung): `pytext.Repr(s string) string`, `pytext.Strip(s string) string`.
  - Task 3: `type privacy.Channel string`, `privacy.ChannelLocal`, `privacy.ChannelCloud`, `type privacy.VisibleArea struct { Area config.Area; Manifest *config.Manifest }`, `privacy.VisibleAreas(registryDir, legacyDir, scope string, ch privacy.Channel) ([]privacy.VisibleArea, error)`, `privacy.Single(areas []privacy.VisibleArea, scope string) (privacy.VisibleArea, error)`, `privacy.Contained(scope, relative string) (string, error)`.
  - Task 5: `graph.ReadGraph(area config.Area, stateDir string) (*graph.Graph, error)`, `graph.Neighbors(g *graph.Graph, relative string) ([]string, []string)`, `graph.RenderNeighbors(incoming, outgoing []string) string`. Zusicherung: `RenderNeighbors` formatiert `"incoming: %s\noutgoing: %s\n"`, also mit abschließendem Zeilenumbruch, und schreibt `-` für eine leere Richtung (ub `pkg/graph/neighbors.go:42`); `brainNeighbors` hängt nichts an, und die Tests erwarten `incoming: notes/c.md\noutgoing: notes/b.md\n`.
  - Task 6: `reader.ReadDocument(area config.Area, manifest *config.Manifest, relative, section string, ch privacy.Channel, stateDir string) (string, error)`.
  - Task 7: `catalog.ReadAreaCatalog(area config.Area, stateDir string) (string, error)`, `catalog.RenderRootCatalog(areas []config.Area) string`.
  - Task 8: `type search.Profile string`, `search.ProfileKeyword`, `search.ProfileFast`, `search.ProfileFull`; `type search.SearchHit struct { Collection, Scope, Relative string; Line int; Title, Snippet string; Score float64; ContentKey string }`; `type search.SearchPort interface` mit `Search`, `Indexed`, `Refresh`, `NotYetSearchable` und `Embed` (ub `pkg/search/port.go`); `type search.QmdMcpPort`, `type search.QmdMcpOption func(*search.QmdMcpPort)`, `search.NewQmdMcpPort(opts ...search.QmdMcpOption) *search.QmdMcpPort`, `search.WithNotice(fn func(message string)) search.QmdMcpOption`, `(*search.QmdMcpPort).Close() error` (ub `pkg/search/mcp.go:157`; der `io.Closer`-Zweig in `brainSearch` schließt darüber die MCP-Sitzung); `type search.RunnerFunc func(argv []string) ([]byte, []byte, int, error)`, `type search.QmdPort struct { Executable string; Runner search.RunnerFunc }`, `search.DefaultRunner(argv []string) ([]byte, []byte, int, error)`; `type search.SearchAnswer struct { Hits []search.SearchHit; Findings []string }`, `search.ExecuteSearch(query, scope string, profile search.Profile, n int, channel privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) (*search.SearchAnswer, error)`, `search.FormatSearch(answer *search.SearchAnswer) string`.
  - Task 8, nur in Tests (ub `pkg/search/fake.go`, wortgleich umgezogen): `search.NewFakePort() *search.FakePort`, genutzte Felder `FakePort{Results []search.ScriptedSearch; Calls []search.SearchCall; SearchableCounts int}`, `type search.ScriptedSearch struct { Hits []search.SearchHit; Err error }`, `type search.SearchCall struct { Query string; Collections []string; Profile search.Profile; N int }`.
  - Task 9: `status.Lines(ch privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) ([]string, error)`.
- Produces:
```go
func brainCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int
var brainSearchPort = func(notice func(string)) search.SearchPort { return search.NewQmdMcpPort(search.WithNotice(notice)) }
var brainStatusPort = func() search.SearchPort { return &search.QmdPort{Executable: "qmd", Runner: search.DefaultRunner} }
var brainNow = time.Now
```
  Testhelfer in `internal/cli/brain_test.go` (Task 12 ruft die beiden Port-Helfer; beide stellen den Seam über `t.Cleanup` zurück):
```go
func stubBrainSearchPort(t *testing.T, port search.SearchPort, notice string)
func stubBrainStatusPort(t *testing.T, port search.SearchPort)
func stubBrainNow(t *testing.T, now time.Time)
```
  Ausgabeformen (Task 16 prüft sie im Rauchtest; in geschweiften Klammern steht, was eingesetzt wird, außer in der wörtlichen obersten Usage-Zeile):
  - Usage-Fehler des obersten Parsers — kein Unterbefehl, unbekannter Unterbefehl, überzählige Argumente: stderr `usage: loomux brain {search,catalog,read,neighbors,status} ...`, dann `loomux brain: error: {meldung}`; Exit 2, stdout leer.
  - Jeder andere Usage-Fehler: stderr die Usage-Zeile des Unterbefehls (für jeden der fünf wörtlich in `TestBrainUsageLines`), dann `loomux brain {sub}: error: {meldung}`; Exit 2, stdout leer.
  - Laufzeitfehler: stderr `error: {grund}`, Exit 1, stdout leer. `search` schreibt nach den Treffern je Befund `note: {befund}` auf stderr.

  Ergänzungen (alle unexportiert, `internal/cli`):
```go
// brainargs.go
var brainSubcommands = []string{"search", "catalog", "read", "neighbors", "status"}
const brainResultCount = 5
type brainOption struct { flag, metavar, fallback string; required bool; choices []string; atLeastOne bool }
type brainParser struct { name, positional string; options []brainOption }
type brainArgs struct { positional string; values map[string]string; n int }
type brainUsageError struct { top bool; message string }
func brainTopUsage() string
func brainChoices(choices []string) string
func brainParserFor(name string) (brainParser, bool)
func (p brainParser) usage() string
func (p brainParser) parse(args []string) (brainArgs, *brainUsageError)
func brainAtLeastOne(text string) (int, error)
// brain.go
func brainRefuse(stderr io.Writer, usage, prog, message string)
func brainRun(name string, a brainArgs, stderr io.Writer) (string, []string, error)
func brainSearch(a brainArgs, channel privacy.Channel, registryDir, legacyDir string, stderr io.Writer) (string, []string, error)
func brainCatalog(scope string, channel privacy.Channel, registryDir, legacyDir string) (string, error)
func brainArea(scope string, channel privacy.Channel, registryDir, legacyDir string) (privacy.VisibleArea, error)
func brainRead(relative, scope, section string, channel privacy.Channel, registryDir, legacyDir string) (string, error)
func brainNeighbors(relative, scope string, channel privacy.Channel, registryDir, legacyDir string) (string, error)
func brainStatus(channel privacy.Channel, registryDir, legacyDir string) (string, error)
```
  Dazu unexportierte Helfer in `brainargs.go`: `brainToken`, `brainScan`, `(brainParser).option`, `(brainParser).classify`, `brainLooksNegative`, `(*brainScan).positionalAt`, `(*brainScan).optionalAt`, `(*brainScan).take`, `brainPythonInt`.

**Wie der Leser gebaut ist, und warum so.** Der Leser bildet `argparse._parse_known_args` der Referenz-Python 3.14.7 für genau diese fünf Formen nach (eine Position, jede Option mit einem Wert): Wörter werden klassifiziert wie `_parse_optional` (leer, ohne `-`, `-` allein, Wörter, deren Anfang auf `-\.?\d` passt, und Wörter mit Leerzeichen sind Werte; `--flag=wert` und `-nWERT` tragen den Wert angeklebt; das erste `--` beendet Optionen), dann abwechselnd Position (`-*A-*`, das `--` darin fällt weg) und Option verbraucht. Fehler in der Reihenfolge der Referenz: ein falscher Optionswert beim Lesen, dann fehlende Argumente, dann überzählige — letztere meldet `brain-mcp` vom **obersten** Parser (`parse_args`), nicht vom Unterbefehl. Die Quelle des Algorithmus ist gemessen, nicht erinnert. Jeder Aufruf der Referenz in diesem Task läuft mit `--no-sync` und `PYTHONDONTWRITEBYTECODE=1`, damit der Quell-Worktree weder eine `.venv`-Synchronisation noch `.pyc`-Dateien bekommt:

```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "import argparse, inspect, sys; print(sys.version.split()[0]); print(argparse.ArgumentParser('x')._negative_number_matcher.pattern); src = inspect.getsource(argparse); [print(l.strip()) for l in src.splitlines() if any(k in l for k in ('invalid choice:', 'invalid %(type)s value', 'unrecognized arguments:', 'expected one argument', 'the following arguments are required'))]"
```
Erwartet (gemessen 2026-09-15):
```
3.14.7
-\.?\d
msg = _('unrecognized arguments: %s') % ' '.join(argv)
raise ArgumentError(None, _('the following arguments are required: %s') %
None: _('expected one argument'),
msg = _('unrecognized arguments: %s') % ' '.join(argv)
msg = _('invalid %(type)s value: %(value)r')
msg = _('invalid choice: %(value)r (choose from %(choices)s)')
msg = _('invalid choice: %(value)r, maybe you meant %(closest)r? '
```
Die letzte Zeile gilt nur mit `suggest_on_error`; der Parser der Referenz setzt es nicht:

```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "import argparse; from brain.cli import _build_parser; p = _build_parser(); sub = [a for a in p._actions if isinstance(a, argparse._SubParsersAction)][0]; print(p.suggest_on_error, sub.choices['search'].suggest_on_error)"
```
Erwartet: `False False`.

Die Muster, nach denen `_parse_known_args` Positionen und Optionen mit einem Wert verbraucht, und die Schreibweise der Wahlliste:

```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "import argparse, inspect; p = argparse.ArgumentParser('x'); a = p.add_argument('query'); o = p.add_argument('--scope'); print(p._get_nargs_pattern(a)); print(p._get_nargs_pattern(o)); [print(l.strip()) for l in inspect.getsource(p._check_value).splitlines() if 'join' in l]"
```
Erwartet (gemessen 2026-09-15):
```
(-*A-*)
([A])
'choices': ', '.join(repr(str(choice)) for choice in action.choices)}
```

Jede erwartete Meldung und jedes angenommene Ergebnis in `brainargs_test.go` stammt aus einer Messung an der Referenz (`_build_parser().parse_args([...])` aus `brain.cli`, stdout/stderr umgeleitet, `SystemExit` gefangen), die Wortlaute der Konsole zusätzlich an `brain-mcp` selbst mit einem Exit-2-Aufruf bestätigt:

```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" brain-mcp search q extra
```
Erwartet (Exit 2, gemessen 2026-09-15):
```
usage: brain-mcp [-h]
                 {reindex,embed,search,catalog,read,neighbors,bench,status,mcp,convert,fetch,lint,types,retype,reconcile,cases,case,approve,hook,init,wiki,wiki-gate,daemon} ...
brain-mcp: error: unrecognized arguments: extra
```

**Abweichungen vom Vertragstext, der Quelle folgend** (auch in „contract_conflicts" gemeldet; Vertrag Task 10, Regel 2):
1. Die Wahlliste steht in Anführungszeichen: `argument --channel: invalid choice: 'x' (choose from 'local', 'cloud')` (gemessen), nicht `(choose from local, cloud)`. Die Profile in Enum-Reihenfolge: `(choose from 'fast', 'full', 'keyword')`.
2. Überzählige Argumente meldet der oberste Parser: `usage: loomux brain {search,catalog,read,neighbors,status} ...` und `loomux brain: error: unrecognized arguments: {wörter}`, nicht `loomux brain {sub}: error: {meldung}`.
3. Ohne Unterbefehl und bei unbekanntem Unterbefehl gilt ebenfalls argparse: `loomux brain: error: the following arguments are required: command` bzw. `loomux brain: error: argument command: invalid choice: 'frob' (choose from 'search', 'catalog', 'read', 'neighbors', 'status')`, beide mit der obersten Usage-Zeile und Exit 2 — statt `loomux brain: subcommand required` / `unknown subcommand "x"`.

**Stützen aus anderen Tasks**, auf die `brain_test.go` exakte Zeichenketten baut: `ExecuteSearch` reicht einen Port-Fehler unverändert weiter und setzt `SearchHit.Scope` (ub `pkg/search/search.go:131-134`, `:168`); `pytext.ReadText` reicht Lesefehler von `os.ReadFile` unverändert weiter (Task 2), `ReadAreaCatalog` und `ReadDocument` ebenso (ub `area.go:30-33`, `read.go:41-44`); `UnknownScope`, `Contained`, `ErrNotIndexed`, die Verweigerungen von `ReadDocument`, `no section titled`, L1b und L4 von `status.Lines` mit den Wortlauten des Vertrags. Nur der Wortlaut eines Registry-Fehlers gehört Task 3/`config`; der Test prüft ihn darum über `Contains`.

- [ ] **Step 1: Failing Test für den Leser** — `internal/cli/brainargs_test.go`

```go
package cli

import (
	"math"
	"reflect"
	"strconv"
	"testing"
)

func brainParserNamed(t *testing.T, name string) brainParser {
	t.Helper()
	p, ok := brainParserFor(name)
	if !ok {
		t.Fatalf("no parser for %q", name)
	}
	return p
}

func TestBrainParserForKnowsTheFiveSubcommands(t *testing.T) {
	for _, name := range brainSubcommands {
		if p, ok := brainParserFor(name); !ok || p.name != name {
			t.Fatalf("%s: %v %+v", name, ok, p)
		}
	}
	if _, ok := brainParserFor("reindex"); ok {
		t.Fatal("reindex is not a loomux brain subcommand")
	}
}

func TestBrainUsageLines(t *testing.T) {
	want := map[string]string{
		"search":    "usage: loomux brain search [--scope SCOPE] [--profile {fast,full,keyword}] [-n N] [--channel {local,cloud}] query",
		"catalog":   "usage: loomux brain catalog [--scope SCOPE] [--channel {local,cloud}]",
		"read":      "usage: loomux brain read --scope SCOPE [--section SECTION] [--channel {local,cloud}] relative",
		"neighbors": "usage: loomux brain neighbors --scope SCOPE [--channel {local,cloud}] relative",
		"status":    "usage: loomux brain status [--channel {local,cloud}]",
	}
	for name, line := range want {
		if got := brainParserNamed(t, name).usage(); got != line {
			t.Fatalf("%s:\n got %q\nwant %q", name, got, line)
		}
	}
	if got := brainTopUsage(); got != "usage: loomux brain {search,catalog,read,neighbors,status} ..." {
		t.Fatalf("top usage %q", got)
	}
}

// Every row was measured against the reference: _build_parser().parse_args()
// of ub cli.py under Python 3.14.7.
func TestBrainParseAcceptsWhatArgparseAccepts(t *testing.T) {
	search := func(query, scope, profile string, n int, channel string) brainArgs {
		return brainArgs{positional: query, n: n, values: map[string]string{
			"--scope": scope, "--profile": profile, "-n": strconv.Itoa(n), "--channel": channel}}
	}
	for _, c := range []struct {
		sub  string
		args []string
		want brainArgs
	}{
		{"search", []string{"q"}, search("q", "all", "fast", 5, "local")},
		{"search", []string{"--scope", "a", "q", "--profile=full", "-n3", "--channel=cloud"}, search("q", "a", "full", 3, "cloud")},
		{"search", []string{"q", "-n=3"}, search("q", "all", "fast", 3, "local")},
		{"search", []string{"q", "-n", " 3 "}, search("q", "all", "fast", 3, "local")},
		{"search", []string{"q", "-n", "+3"}, search("q", "all", "fast", 3, "local")},
		{"search", []string{"q", "-n", "3_0"}, search("q", "all", "fast", 30, "local")},
		{"search", []string{"q", "-n", "003"}, search("q", "all", "fast", 3, "local")},
		{"search", []string{"q", "-n", "1_000_000"}, search("q", "all", "fast", 1000000, "local")},
		{"search", []string{"q", "-n", "\u00a07\u2003"}, search("q", "all", "fast", 7, "local")},
		{"search", []string{"--", "-x"}, search("-x", "all", "fast", 5, "local")},
		{"search", []string{"-5"}, search("-5", "all", "fast", 5, "local")},
		{"search", []string{"-5x"}, search("-5x", "all", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "-5"}, search("q", "-5", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "-.5"}, search("q", "-.5", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "-"}, search("q", "-", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "-x y"}, search("q", "-x y", "fast", 5, "local")},
		{"search", []string{"-a b"}, search("-a b", "all", "fast", 5, "local")},
		{"search", []string{"--", "--"}, search("--", "all", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "a", "--scope", "b"}, search("q", "b", "fast", 5, "local")},
		{"search", []string{"q", "--scope="}, search("q", "", "fast", 5, "local")},
		{"search", []string{""}, search("", "all", "fast", 5, "local")},
		{"search", []string{"q", "--"}, search("q", "all", "fast", 5, "local")},
		{"search", []string{"--scope", "a", "--", "q"}, search("q", "a", "fast", 5, "local")},
		{"read", []string{"--scope", "s", "rel", "--section", "T"}, brainArgs{positional: "rel", n: 5,
			values: map[string]string{"--scope": "s", "--section": "T", "--channel": "local"}}},
		{"read", []string{"rel", "--scope", "s", "--section", ""}, brainArgs{positional: "rel", n: 5,
			values: map[string]string{"--scope": "s", "--section": "", "--channel": "local"}}},
		{"neighbors", []string{"--scope=s", "rel"}, brainArgs{positional: "rel", n: 5,
			values: map[string]string{"--scope": "s", "--channel": "local"}}},
		{"catalog", nil, brainArgs{n: 5, values: map[string]string{"--scope": "all", "--channel": "local"}}},
		{"catalog", []string{"--scope="}, brainArgs{n: 5, values: map[string]string{"--scope": "", "--channel": "local"}}},
		{"status", []string{"--channel", "cloud"}, brainArgs{n: 5, values: map[string]string{"--channel": "cloud"}}},
	} {
		got, err := brainParserNamed(t, c.sub).parse(c.args)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s %q:\n got %+v, %v\nwant %+v", c.sub, c.args, got, err, c.want)
		}
	}
}

// Every message was measured against the reference, as above; top marks the
// ones brain-mcp reports from its top-level parser.
func TestBrainParseRefusesWhatArgparseRefuses(t *testing.T) {
	const channelChoice = "(choose from 'local', 'cloud')"
	for _, c := range []struct {
		sub     string
		args    []string
		top     bool
		message string
	}{
		{"search", nil, false, "the following arguments are required: query"},
		{"search", []string{"--bogus"}, false, "the following arguments are required: query"},
		{"search", []string{"--"}, false, "the following arguments are required: query"},
		{"search", []string{"q", "--channel", "x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"search", []string{"q", "--channel=x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"search", []string{"q", "--channel=cloud=x"}, false, "argument --channel: invalid choice: 'cloud=x' " + channelChoice},
		{"search", []string{"x'y", "--channel", "x'y"}, false, `argument --channel: invalid choice: "x'y" ` + channelChoice},
		{"search", []string{"q", "--profile", "x"}, false, "argument --profile: invalid choice: 'x' (choose from 'fast', 'full', 'keyword')"},
		{"search", []string{"q", "-n", "0"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"q", "-n", "-3"}, false, "argument -n: must be at least 1, got -3"},
		{"search", []string{"q", "-n0"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"q", "-n=0"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"q", "-n", "-0"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"q", "-n", "-99999999999999999999999"}, false, "argument -n: must be at least 1, got -99999999999999999999999"},
		{"search", []string{"q", "-n", "x"}, false, "argument -n: invalid _at_least_one value: 'x'"},
		{"search", []string{"q", "-n", ""}, false, "argument -n: invalid _at_least_one value: ''"},
		{"search", []string{"q", "-n="}, false, "argument -n: invalid _at_least_one value: ''"},
		{"search", []string{"q", "-nx"}, false, "argument -n: invalid _at_least_one value: 'x'"},
		{"search", []string{"q", "-n", "3=4"}, false, "argument -n: invalid _at_least_one value: '3=4'"},
		{"search", []string{"q", "-n3=4"}, false, "argument -n: invalid _at_least_one value: '3=4'"},
		{"search", []string{"q", "-n", "_3"}, false, "argument -n: invalid _at_least_one value: '_3'"},
		{"search", []string{"q", "-n", "3_"}, false, "argument -n: invalid _at_least_one value: '3_'"},
		{"search", []string{"q", "-n", "3__0"}, false, "argument -n: invalid _at_least_one value: '3__0'"},
		{"search", []string{"q", "-n", "3.0"}, false, "argument -n: invalid _at_least_one value: '3.0'"},
		{"search", []string{"q", "-n", "1 2"}, false, "argument -n: invalid _at_least_one value: '1 2'"},
		{"search", []string{"q", "-n", "-"}, false, "argument -n: invalid _at_least_one value: '-'"},
		{"search", []string{"q", "-n"}, false, "argument -n: expected one argument"},
		{"search", []string{"q", "-n", "--"}, false, "argument -n: expected one argument"},
		{"search", []string{"q", "-n", "-x"}, false, "argument -n: expected one argument"},
		{"search", []string{"q", "--scope"}, false, "argument --scope: expected one argument"},
		{"search", []string{"q", "--scope", "--channel", "local"}, false, "argument --scope: expected one argument"},
		{"search", []string{"--scope", "--", "q"}, false, "argument --scope: expected one argument"},
		{"search", []string{"q", "--scope=a", "--scope"}, false, "argument --scope: expected one argument"},
		{"search", []string{"-n", "0", "--channel", "x"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"--channel", "x", "-n", "0"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"search", []string{"q", "-n", "3", "-n", "0"}, false, "argument -n: must be at least 1, got 0"},
		{"catalog", []string{"--channel", "x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"read", nil, false, "the following arguments are required: relative, --scope"},
		{"read", []string{"rel"}, false, "the following arguments are required: --scope"},
		{"read", []string{"--scope", "s"}, false, "the following arguments are required: relative"},
		{"read", []string{"rel", "--scope", "s", "--section"}, false, "argument --section: expected one argument"},
		{"read", []string{"rel", "extra"}, false, "the following arguments are required: --scope"},
		{"read", []string{"--bogus"}, false, "the following arguments are required: relative, --scope"},
		{"read", []string{"--", "rel", "--scope", "s"}, false, "the following arguments are required: --scope"},
		{"neighbors", nil, false, "the following arguments are required: relative, --scope"},
		{"neighbors", []string{"rel"}, false, "the following arguments are required: --scope"},
		{"status", []string{"--channel", "x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"status", []string{"--bogus", "--channel", "x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"search", []string{"q", "extra"}, true, "unrecognized arguments: extra"},
		{"search", []string{"q", "extra", "more"}, true, "unrecognized arguments: extra more"},
		{"search", []string{"q", "--bogus"}, true, "unrecognized arguments: --bogus"},
		{"search", []string{"--bogus", "x"}, true, "unrecognized arguments: --bogus"},
		{"search", []string{"--bogus=1", "q"}, true, "unrecognized arguments: --bogus=1"},
		{"search", []string{"q", "--", "extra"}, true, "unrecognized arguments: extra"},
		{"search", []string{"q", "--", "--", "x"}, true, "unrecognized arguments: -- x"},
		{"search", []string{"q", "--scope", "s", "--"}, true, "unrecognized arguments: --"},
		{"search", []string{"--", "q", "-n", "3"}, true, "unrecognized arguments: -n 3"},
		{"search", []string{"q", "-x"}, true, "unrecognized arguments: -x"},
		{"search", []string{"q", "-"}, true, "unrecognized arguments: -"},
		{"search", []string{"q", "-.5"}, true, "unrecognized arguments: -.5"},
		{"search", []string{"q", "-5x"}, true, "unrecognized arguments: -5x"},
		{"search", []string{"q", "-n", "3", "extra"}, true, "unrecognized arguments: extra"},
		{"search", []string{"q", "--scope", "s", "extra", "--channel", "cloud"}, true, "unrecognized arguments: extra"},
		{"catalog", []string{"extra"}, true, "unrecognized arguments: extra"},
		{"read", []string{"a", "b", "--scope", "s"}, true, "unrecognized arguments: b"},
		{"neighbors", []string{"rel", "--scope", "s", "--section", "x"}, true, "unrecognized arguments: --section x"},
		{"status", []string{"extra"}, true, "unrecognized arguments: extra"},
		{"status", []string{"--"}, true, "unrecognized arguments: --"},
		{"status", []string{"--", "x"}, true, "unrecognized arguments: -- x"},
		{"status", []string{"--channel=cloud", "--channel", "local", "z"}, true, "unrecognized arguments: z"},
	} {
		_, err := brainParserNamed(t, c.sub).parse(c.args)
		if err == nil || err.top != c.top || err.message != c.message {
			t.Fatalf("%s %q:\n got %+v\nwant top=%v %q", c.sub, c.args, err, c.top, c.message)
		}
	}
}

// Where loomux answers differently from the reference on purpose; each row is
// a line of the parity list.
func TestBrainParseDeviatesFromArgparseWhereTheParityListSays(t *testing.T) {
	// argparse widens --prof to --profile and refuses --s as ambiguous; loomux
	// knows only whole option names.
	for _, c := range []struct {
		args    []string
		message string
	}{
		{[]string{"q", "--prof", "x"}, "unrecognized arguments: --prof x"},
		{[]string{"q", "--prof=full"}, "unrecognized arguments: --prof=full"},
		{[]string{"q", "--s", "x"}, "unrecognized arguments: --s x"},
		{[]string{"q", "-h"}, "unrecognized arguments: -h"},
		{[]string{"q", "--state-dir", "x"}, "unrecognized arguments: --state-dir x"},
	} {
		_, err := brainParserNamed(t, "search").parse(c.args)
		if err == nil || !err.top || err.message != c.message {
			t.Fatalf("%q: got %+v, want %q", c.args, err, c.message)
		}
	}
	// int() accepts every Unicode decimal digit; loomux only ASCII ones.
	for _, digit := range []string{"\u0663", "\uff13"} {
		_, err := brainParserNamed(t, "search").parse([]string{"q", "-n", digit})
		if err == nil || err.message != "argument -n: invalid _at_least_one value: '"+digit+"'" {
			t.Fatalf("%q: got %+v", digit, err)
		}
	}
	// Python keeps a count of any size; loomux holds it at the largest int.
	got, err := brainParserNamed(t, "search").parse([]string{"q", "-n", "99999999999999999999999"})
	if err != nil || got.n != math.MaxInt {
		t.Fatalf("got %+v, %v", got, err)
	}
}
```

Gemessene Werte hinter den Tabellen (Python 3.14.7, `_build_parser().parse_args`): jede Zeile der ersten Tabelle nimmt die Referenz mit genau dem dort erwarteten Ergebnis an, auch die Leerraumform `-n "\u00a07\u2003"` → 7; jede Zeile der zweiten Tabelle lehnt sie mit genau der dort erwarteten Meldung ab, Exit 2 in jedem Fall; die Zeilen der dritten Tabelle ergeben an der Referenz: `--prof x` → `argument --profile: invalid choice: 'x' (choose from 'fast', 'full', 'keyword')`, `--prof=full` → angenommen, `--s x` → `ambiguous option: --s could match --scope, --state-dir`, `-h` → Hilfe auf stdout, Exit 0, `-n ٣` und `-n ３` → 3, `-n 99999999999999999999999` → angenommen.

- [ ] **Step 2: Scheitern sehen**

```bash
go test ./internal/cli/ -run 'TestBrainParserFor|TestBrainUsageLines|TestBrainParse' -count=1
```
Expected (gemessen 2026-09-15 im Wegwerfmodul: der committete Baum plus `brainargs_test.go` aus Step 1; Zeile 1 ist `package cli`, Go bricht nach zehn Fehlern ab):
```
# github.com/xidus90/loomux/internal/cli [github.com/xidus90/loomux/internal/cli.test]
internal\cli\brainargs_test.go:10:50: undefined: brainParser
internal\cli\brainargs_test.go:12:11: undefined: brainParserFor
internal\cli\brainargs_test.go:20:23: undefined: brainSubcommands
internal\cli\brainargs_test.go:21:15: undefined: brainParserFor
internal\cli\brainargs_test.go:25:14: undefined: brainParserFor
internal\cli\brainargs_test.go:43:12: undefined: brainTopUsage
internal\cli\brainargs_test.go:51:70: undefined: brainArgs
internal\cli\brainargs_test.go:52:10: undefined: brainArgs
internal\cli\brainargs_test.go:58:8: undefined: brainArgs
internal\cli\brainargs_test.go:83:63: undefined: brainArgs
internal\cli\brainargs_test.go:83:63: too many errors
FAIL	github.com/xidus90/loomux/internal/cli [build failed]
FAIL
```

- [ ] **Step 3: Den Leser schreiben** — `internal/cli/brainargs.go`

```go
package cli

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
)

// brainSubcommands are the verbs of `loomux brain`, in the order cli.py's
// _build_parser adds them: argparse names its choices in that order.
var brainSubcommands = []string{"search", "catalog", "read", "neighbors", "status"}

// brainResultCount is the default of `search -n` (cli.py:483).
const brainResultCount = 5

// brainOption is one option of a brain subcommand. Every option of the five
// takes exactly one value (cli.py:471-501, 451), so the reader knows no other
// arity.
type brainOption struct {
	flag       string
	metavar    string
	fallback   string
	required   bool
	choices    []string
	atLeastOne bool
}

// brainParser is the argument shape of one subcommand: at most one positional
// and the options in the order the reference declares them, which is the order
// argparse reports missing ones in.
type brainParser struct {
	name       string
	positional string
	options    []brainOption
}

// brainArgs is a successful parse: the positional, every option's value by flag
// (fallbacks included) and the count behind -n.
type brainArgs struct {
	positional string
	values     map[string]string
	n          int
}

// brainUsageError is argparse's ArgumentError. top marks the one error the
// reference reports from the top-level parser instead of the subcommand's:
// unrecognized arguments, which parse_args checks after the subparser returned.
type brainUsageError struct {
	top     bool
	message string
}

// brainToken is one classified argument, as _parse_known_args sees it: 'A' for
// a value, 'O' for an option and '-' for the first "--".
type brainToken struct {
	kind        byte
	option      *brainOption
	explicit    string
	hasExplicit bool
}

// brainScan is the state of one parse.
type brainScan struct {
	parser     brainParser
	args       []string
	tokens     []brainToken
	out        brainArgs
	positional bool
	extras     []string
}

// brainTopUsage is the usage line of `loomux brain` itself.
func brainTopUsage() string {
	return "usage: loomux brain {" + strings.Join(brainSubcommands, ",") + "} ..."
}

// brainChoices renders a choice list the way argparse's _check_value does:
// every choice through repr, joined by a comma and a space.
func brainChoices(choices []string) string {
	quoted := make([]string, len(choices))
	for i, choice := range choices {
		quoted[i] = pytext.Repr(choice)
	}
	return strings.Join(quoted, ", ")
}

// brainParserFor answers the argument shape of one subcommand as cli.py:471-501
// and :542 declare it, without --state-dir: loomux reads its state from
// config.StateDir and the legacy directory, never from a flag.
func brainParserFor(name string) (brainParser, bool) {
	scope := brainOption{flag: "--scope", metavar: "SCOPE", fallback: "all"}
	requiredScope := brainOption{flag: "--scope", metavar: "SCOPE", required: true}
	channel := brainOption{
		flag:     "--channel",
		fallback: string(privacy.ChannelLocal),
		choices:  []string{string(privacy.ChannelLocal), string(privacy.ChannelCloud)},
	}
	switch name {
	case "search":
		profile := brainOption{
			flag:     "--profile",
			fallback: string(search.ProfileFast),
			choices:  []string{string(search.ProfileFast), string(search.ProfileFull), string(search.ProfileKeyword)},
		}
		count := brainOption{flag: "-n", metavar: "N", fallback: strconv.Itoa(brainResultCount), atLeastOne: true}
		return brainParser{name: name, positional: "query", options: []brainOption{scope, profile, count, channel}}, true
	case "catalog":
		return brainParser{name: name, options: []brainOption{scope, channel}}, true
	case "read":
		section := brainOption{flag: "--section", metavar: "SECTION"}
		return brainParser{name: name, positional: "relative", options: []brainOption{requiredScope, section, channel}}, true
	case "neighbors":
		return brainParser{name: name, positional: "relative", options: []brainOption{requiredScope, channel}}, true
	case "status":
		return brainParser{name: name, options: []brainOption{channel}}, true
	}
	return brainParser{}, false
}

// usage is the subcommand's usage line: options first, the positional last,
// as argparse formats it, on one line and without -h and --state-dir.
func (p brainParser) usage() string {
	var b strings.Builder
	b.WriteString("usage: loomux brain " + p.name)
	for _, opt := range p.options {
		metavar := opt.metavar
		if opt.choices != nil {
			metavar = "{" + strings.Join(opt.choices, ",") + "}"
		}
		if opt.required {
			fmt.Fprintf(&b, " %s %s", opt.flag, metavar)
		} else {
			fmt.Fprintf(&b, " [%s %s]", opt.flag, metavar)
		}
	}
	if p.positional != "" {
		b.WriteString(" " + p.positional)
	}
	return b.String()
}

func (p brainParser) option(flag string) *brainOption {
	for i := range p.options {
		if p.options[i].flag == flag {
			return &p.options[i]
		}
	}
	return nil
}

// classify is argparse's _parse_optional without abbreviations: a word that is
// not an option of this subcommand but looks like one becomes an unknown
// option, which ends among the unrecognized arguments.
func (p brainParser) classify(arg string) brainToken {
	if arg == "" || arg[0] != '-' {
		return brainToken{kind: 'A'}
	}
	if opt := p.option(arg); opt != nil {
		return brainToken{kind: 'O', option: opt}
	}
	if len(arg) == 1 {
		return brainToken{kind: 'A'}
	}
	if flag, value, found := strings.Cut(arg, "="); found {
		if opt := p.option(flag); opt != nil {
			return brainToken{kind: 'O', option: opt, explicit: value, hasExplicit: true}
		}
	}
	// A single-dash option carries its value glued on: -n3.
	if arg[1] != '-' {
		if opt := p.option(arg[:2]); opt != nil {
			return brainToken{kind: 'O', option: opt, explicit: arg[2:], hasExplicit: true}
		}
	}
	if brainLooksNegative(arg) || strings.Contains(arg, " ") {
		return brainToken{kind: 'A'}
	}
	return brainToken{kind: 'O'}
}

// brainLooksNegative is argparse's _negative_number_matcher `-\.?\d` matched at
// the start of the word. No option of the five looks like a number, so such a
// word is always a value.
func brainLooksNegative(arg string) bool {
	rest := strings.TrimPrefix(arg[1:], ".")
	r, _ := utf8.DecodeRuneInString(rest)
	return unicode.IsDigit(r)
}

// parse reads the arguments behind the subcommand the way argparse's
// _parse_known_args does for this shape: positional and options interleaved,
// errors in the order argparse raises them -- a bad option value while
// scanning, then missing arguments, then unrecognized ones.
func (p brainParser) parse(args []string) (brainArgs, *brainUsageError) {
	s := &brainScan{
		parser: p,
		args:   args,
		tokens: make([]brainToken, len(args)),
		out:    brainArgs{values: map[string]string{}, n: brainResultCount},
	}
	last := -1
	ended := false
	for i, arg := range args {
		switch {
		case ended:
			s.tokens[i] = brainToken{kind: 'A'}
		case arg == "--":
			s.tokens[i] = brainToken{kind: '-'}
			ended = true
		default:
			s.tokens[i] = p.classify(arg)
		}
		if s.tokens[i].kind == 'O' {
			last = i
		}
	}

	start := 0
	for start <= last {
		next := start
		for s.tokens[next].kind != 'O' {
			next++
		}
		if start != next {
			if end := s.positionalAt(start); end > start {
				start = end
				continue
			}
			s.extras = append(s.extras, args[start:next]...)
			start = next
		}
		var err *brainUsageError
		if start, err = s.optionalAt(start); err != nil {
			return brainArgs{}, err
		}
	}
	end := s.positionalAt(start)
	s.extras = append(s.extras, args[end:]...)

	var missing []string
	if p.positional != "" && !s.positional {
		missing = append(missing, p.positional)
	}
	for _, opt := range p.options {
		if _, given := s.out.values[opt.flag]; given {
			continue
		}
		if opt.required {
			missing = append(missing, opt.flag)
			continue
		}
		s.out.values[opt.flag] = opt.fallback
	}
	if len(missing) > 0 {
		return brainArgs{}, &brainUsageError{message: "the following arguments are required: " + strings.Join(missing, ", ")}
	}
	if len(s.extras) > 0 {
		return brainArgs{}, &brainUsageError{top: true, message: "unrecognized arguments: " + strings.Join(s.extras, " ")}
	}
	return s.out, nil
}

// positionalAt is consume_positionals for at most one positional: the pattern
// `-*A-*` from start, the "--" inside it dropped. It answers where the scan
// goes on, which is start when nothing was taken.
func (s *brainScan) positionalAt(start int) int {
	if s.parser.positional == "" || s.positional {
		return start
	}
	i := start
	for i < len(s.tokens) && s.tokens[i].kind == '-' {
		i++
	}
	if i == len(s.tokens) || s.tokens[i].kind != 'A' {
		return start
	}
	s.out.positional = s.args[i]
	s.positional = true
	i++
	for i < len(s.tokens) && s.tokens[i].kind == '-' {
		i++
	}
	return i
}

// optionalAt is consume_optional for an option of one value: glued on, or the
// next word when that word is a value.
func (s *brainScan) optionalAt(start int) (int, *brainUsageError) {
	tok := s.tokens[start]
	if tok.option == nil {
		s.extras = append(s.extras, s.args[start])
		return start + 1, nil
	}
	value, stop := tok.explicit, start+1
	if !tok.hasExplicit {
		if stop == len(s.tokens) || s.tokens[stop].kind != 'A' {
			return 0, &brainUsageError{message: "argument " + tok.option.flag + ": expected one argument"}
		}
		value, stop = s.args[stop], stop+1
	}
	return stop, s.take(tok.option, value)
}

// take is _get_values for one option: the type first, then the choices.
func (s *brainScan) take(opt *brainOption, value string) *brainUsageError {
	if opt.atLeastOne {
		n, err := brainAtLeastOne(value)
		if err != nil {
			return &brainUsageError{message: "argument " + opt.flag + ": " + err.Error()}
		}
		s.out.n = n
		value = strconv.Itoa(n)
	}
	if opt.choices != nil && !slices.Contains(opt.choices, value) {
		return &brainUsageError{message: fmt.Sprintf("argument %s: invalid choice: %s (choose from %s)",
			opt.flag, pytext.Repr(value), brainChoices(opt.choices))}
	}
	s.out.values[opt.flag] = value
	return nil
}

// brainAtLeastOne is cli.py's _at_least_one: int() of the text, refused below
// one. int() is read over ASCII digits only; a count past the machine's int is
// held at the largest one, which no engine returns fewer hits for.
func brainAtLeastOne(text string) (int, error) {
	digits, ok := brainPythonInt(pytext.Strip(text))
	if !ok {
		return 0, fmt.Errorf("invalid _at_least_one value: %s", pytext.Repr(text))
	}
	value, _ := new(big.Int).SetString(digits, 10)
	if value.Sign() < 1 {
		return 0, errors.New("must be at least 1, got " + value.String())
	}
	if !value.IsInt64() || value.Int64() > math.MaxInt {
		return math.MaxInt, nil
	}
	return int(value.Int64()), nil
}

// brainPythonInt checks the grammar int() accepts in base 10 -- a sign, then
// digits with single underscores between them -- and answers the number
// without the underscores.
func brainPythonInt(s string) (string, bool) {
	sign := ""
	if s != "" && (s[0] == '+' || s[0] == '-') {
		sign, s = s[:1], s[1:]
	}
	var digits strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			digits.WriteByte(c)
		case c == '_' && i > 0 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9':
		default:
			return "", false
		}
	}
	if digits.Len() == 0 {
		return "", false
	}
	return sign + digits.String(), true
}
```

Anmerkung zu `pytext.Strip`: `int()` streift dieselben Zeichen ab wie `str.strip()` (`_Py_UNICODE_ISSPACE`), gemessen an `"\u00a07\u2003"` → 7.

- [ ] **Step 4: Bestehen sehen**

```bash
go test ./internal/cli/ -run 'TestBrainParserFor|TestBrainUsageLines|TestBrainParse' -count=1
```
Expected: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/cli` beginnt, gefolgt von der Dauer (gemessen 2026-09-15 im Wegwerfmodul: der committete Baum plus Stümpfe für `pytext.Repr`, `pytext.Strip`, `privacy.Channel*` und `search.Profile*`; `brainargs.go` braucht aus Task 2 nur `Repr` und `Strip`).

- [ ] **Step 5: Failing Test für den Befehl** — `internal/cli/brain_test.go` (nutzt `run` aus `cli_test.go:10` und `writeFile` aus `wiki_test.go:13`)

```go
package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/search"
)

// brainWorldDirs are the three directories of a brain test world.
type brainWorldDirs struct {
	state, legacy, area string
}

// brainWorld registers one writable area, project/a, whose manifest carries
// the given body after its [area] table; it points both state variables at the
// world and writes the files into the area.
func brainWorld(t *testing.T, manifest string, files map[string]string) brainWorldDirs {
	t.Helper()
	w := brainWorldDirs{state: t.TempDir(), legacy: t.TempDir(), area: t.TempDir()}
	t.Setenv("LOOMUX_STATE_DIR", w.state)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", w.legacy)
	writeFile(t, filepath.Join(w.state, "registry.toml"),
		"[[area]]\nscope = \"project/a\"\npath = "+strconv.Quote(filepath.ToSlash(w.area))+"\n")
	writeFile(t, filepath.Join(w.area, ".loomux", "config.toml"), "[area]\nscope = \"project/a\"\n"+manifest)
	for name, body := range files {
		writeFile(t, filepath.Join(w.area, filepath.FromSlash(name)), body)
	}
	return w
}

// brainReadOnlyWorld registers one read-only area, project/r, whose manifest
// and artefacts lie in the legacy directory under areas/project-r.
func brainReadOnlyWorld(t *testing.T, files map[string]string) brainWorldDirs {
	t.Helper()
	w := brainWorldDirs{state: t.TempDir(), legacy: t.TempDir(), area: t.TempDir()}
	t.Setenv("LOOMUX_STATE_DIR", w.state)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", w.legacy)
	writeFile(t, filepath.Join(w.state, "registry.toml"),
		"[[area]]\nscope = \"project/r\"\npath = "+strconv.Quote(filepath.ToSlash(w.area))+"\nreadonly = true\n")
	artefacts := filepath.Join(w.legacy, "areas", "project-r")
	writeFile(t, filepath.Join(artefacts, ".loomux", "config.toml"), "[area]\nscope = \"project/r\"\n")
	for name, body := range files {
		writeFile(t, filepath.Join(artefacts, filepath.FromSlash(name)), body)
	}
	return w
}

func stubBrainSearchPort(t *testing.T, port search.SearchPort, notice string) {
	t.Helper()
	saved := brainSearchPort
	brainSearchPort = func(announce func(string)) search.SearchPort {
		if notice != "" {
			announce(notice)
		}
		return port
	}
	t.Cleanup(func() { brainSearchPort = saved })
}

func stubBrainStatusPort(t *testing.T, port search.SearchPort) {
	t.Helper()
	saved := brainStatusPort
	brainStatusPort = func() search.SearchPort { return port }
	t.Cleanup(func() { brainStatusPort = saved })
}

func stubBrainNow(t *testing.T, now time.Time) {
	t.Helper()
	saved := brainNow
	brainNow = func() time.Time { return now }
	t.Cleanup(func() { brainNow = saved })
}

// closingPort is a search port that can be closed, like the MCP port.
type closingPort struct {
	*search.FakePort
	closed int
}

func (p *closingPort) Close() error {
	p.closed++
	return nil
}

const brainGraph = `{"scope":"project/a","nodes":[],"edges":[{"from":"notes/a.md","to":"notes/b.md"},{"from":"notes/c.md","to":"notes/a.md"}],"links":{"total":2,"resolved":2,"dropped":{}}}`

func TestBrainUsageErrorsNameTheParserThatRefused(t *testing.T) {
	const top = "usage: loomux brain {search,catalog,read,neighbors,status} ...\n"
	for _, c := range []struct {
		args   []string
		stderr string
	}{
		{[]string{"brain"}, top + "loomux brain: error: the following arguments are required: command\n"},
		{[]string{"brain", "frob"}, top + "loomux brain: error: argument command: invalid choice: 'frob' (choose from 'search', 'catalog', 'read', 'neighbors', 'status')\n"},
		{[]string{"brain", "search"}, "usage: loomux brain search [--scope SCOPE] [--profile {fast,full,keyword}] [-n N] [--channel {local,cloud}] query\n" +
			"loomux brain search: error: the following arguments are required: query\n"},
		{[]string{"brain", "read", "rel"}, "usage: loomux brain read --scope SCOPE [--section SECTION] [--channel {local,cloud}] relative\n" +
			"loomux brain read: error: the following arguments are required: --scope\n"},
		{[]string{"brain", "status", "--channel", "x"}, "usage: loomux brain status [--channel {local,cloud}]\n" +
			"loomux brain status: error: argument --channel: invalid choice: 'x' (choose from 'local', 'cloud')\n"},
		{[]string{"brain", "catalog", "extra"}, top + "loomux brain: error: unrecognized arguments: extra\n"},
	} {
		code, out, errOut := run(c.args...)
		if code != 2 || out != "" || errOut != c.stderr {
			t.Fatalf("%q: code %d, out %q\n got %q\nwant %q", c.args, code, out, errOut, c.stderr)
		}
	}
}

func TestBrainDefaultPortsAreTheQmdPorts(t *testing.T) {
	if _, ok := brainSearchPort(func(string) {}).(*search.QmdMcpPort); !ok {
		t.Fatal("brain search must ask the qmd daemon")
	}
	port, ok := brainStatusPort().(*search.QmdPort)
	if !ok || port.Executable != "qmd" || port.Runner == nil {
		t.Fatalf("brain status must ask the qmd command line, got %#v", port)
	}
}

func TestBrainSearchPrintsHitsAndThenItsNotes(t *testing.T) {
	brainWorld(t, "", nil)
	fake := search.NewFakePort()
	fake.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{
		{Collection: "project-a", Relative: "notes/a.md", Line: 3, Title: "A", Snippet: "one\ntwo", Score: 0.5},
	}}}
	stubBrainSearchPort(t, fake, "starting the search engine")

	code, out, errOut := run("brain", "search", "what", "--profile", "full", "-n", "2")
	wantOut := "brain://project/a/notes/a.md:3  50%  A\n    one\n    two\n\n"
	wantErr := "note: starting the search engine\n" +
		"note: project/a/notes/a.md: hit is not in the register; reindex to catch up\n"
	if code != 0 || out != wantOut || errOut != wantErr {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	want := []search.SearchCall{{Query: "what", Collections: []string{"project-a"}, Profile: search.ProfileFull, N: 2}}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Fatalf("calls %+v", fake.Calls)
	}
}

func TestBrainSearchWithoutMatchesSaysSoAndWhy(t *testing.T) {
	brainWorld(t, "", nil)
	fake := search.NewFakePort()
	fake.Results = []search.ScriptedSearch{{}, {}}
	stubBrainSearchPort(t, fake, "")

	code, out, errOut := run("brain", "search", "nothing")
	wantErr := "note: the search engine answered empty twice in a row on profile fast; an empty answer to a meaning " +
		"search is practically unreachable, so this is more likely a silent failure of the engine than an absence " +
		"of matches (spec 16.14)\n"
	if code != 0 || out != "no matches\n" || errOut != wantErr {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	if len(fake.Calls) != 2 || fake.Calls[1].N != 5 || fake.Calls[1].Profile != search.ProfileFast {
		t.Fatalf("calls %+v", fake.Calls)
	}
}

func TestBrainSearchJudgesTheLegacyStampAtBrainNow(t *testing.T) {
	w := brainWorld(t, "", map[string]string{
		"_identities.tsv": "doc_id\tpfad\tcontent_hash\trevision\nd1\tnotes/a.md\tsha256:00\t1\n",
	})
	writeFile(t, filepath.Join(w.legacy, "maintenance", "last-run.txt"), "2999-01-01T00:00:00+00:00\n")
	stubBrainNow(t, time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC))
	fake := search.NewFakePort()
	fake.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{
		{Collection: "project-a", Relative: "notes/a.md", Line: 1, Title: "A", Score: 1},
	}}}
	stubBrainSearchPort(t, fake, "")

	code, out, errOut := run("brain", "search", "a")
	wantErr := "note: the last full reconciliation was 2999-01-01T00:00:00+00:00, more than 24 hours ago: " +
		"a source may have changed without this answer knowing (run `brain reconcile`)\n"
	if code != 0 || out != "brain://project/a/notes/a.md:1  100%  A\n\n" || errOut != wantErr {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainSearchClosesThePortAndReportsTheEngine(t *testing.T) {
	brainWorld(t, "", nil)
	port := &closingPort{FakePort: search.NewFakePort()}
	port.Results = []search.ScriptedSearch{{Err: errors.New("the search engine did not answer in 3 attempts: refused")}}
	stubBrainSearchPort(t, port, "")

	code, out, errOut := run("brain", "search", "q")
	if code != 1 || out != "" || errOut != "error: the search engine did not answer in 3 attempts: refused\n" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	if port.closed != 1 {
		t.Fatalf("closed %d times", port.closed)
	}
}

func TestBrainSearchRefusesAnUnknownScopeBeforeAsking(t *testing.T) {
	brainWorld(t, "", nil)
	fake := search.NewFakePort()
	stubBrainSearchPort(t, fake, "")

	code, out, errOut := run("brain", "search", "q", "--scope", "nope")
	if code != 1 || out != "" || errOut != "error: unknown scope 'nope'; known scopes are: project/a\n" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("the engine was asked: %+v", fake.Calls)
	}
}

func TestBrainCatalogAnswersTheRootOrOneArea(t *testing.T) {
	index := "# project/a\n\n* [a](brain://project/a/notes/a.md)\n"
	brainWorld(t, "", map[string]string{"index.md": index})

	if code, out, errOut := run("brain", "catalog"); code != 0 || out != "# brain\n\n* [project/a](brain://project/a/)\n" {
		t.Fatalf("root: code %d\nout %q\nerr %q", code, out, errOut)
	}
	if code, out, errOut := run("brain", "catalog", "--scope", "project/a"); code != 0 || out != index {
		t.Fatalf("area: code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainCatalogHidesALocalOnlyAreaFromTheCloud(t *testing.T) {
	brainWorld(t, "[privacy]\nmode = \"local_only\"\n", map[string]string{"index.md": "# project/a\n"})

	if code, out, errOut := run("brain", "catalog", "--channel", "cloud"); code != 0 || out != "# brain\n\n" {
		t.Fatalf("root: code %d\nout %q\nerr %q", code, out, errOut)
	}
	code, out, errOut := run("brain", "catalog", "--channel", "cloud", "--scope", "project/a")
	if code != 1 || out != "" || errOut != "error: unknown scope 'project/a'; known scopes are: \n" {
		t.Fatalf("area: code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainCatalogReportsAMissingIndex(t *testing.T) {
	w := brainWorld(t, "", nil)
	_, readErr := os.ReadFile(filepath.Join(w.area, "index.md"))

	code, out, errOut := run("brain", "catalog", "--scope", "project/a")
	if code != 1 || out != "" || errOut != "error: "+readErr.Error()+"\n" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainReadsAReadOnlyAreaFromTheLegacyDirectory(t *testing.T) {
	brainReadOnlyWorld(t, map[string]string{"index.md": "# project/r\n", "graph.json": brainGraph})

	if code, out, errOut := run("brain", "catalog", "--scope", "project/r"); code != 0 || out != "# project/r\n" {
		t.Fatalf("catalog: code %d\nout %q\nerr %q", code, out, errOut)
	}
	code, out, errOut := run("brain", "neighbors", "notes/a.md", "--scope", "project/r")
	if code != 0 || out != "incoming: notes/c.md\noutgoing: notes/b.md\n" {
		t.Fatalf("neighbors: code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainReadAnswersTheFileOrOneSection(t *testing.T) {
	doc := "# Top\nintro\n## Part\nbody\n# Next\nrest\n"
	brainWorld(t, "", map[string]string{"notes/a.md": doc})

	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"brain", "read", "notes/a.md", "--scope", "project/a"}, doc},
		{[]string{"brain", "read", "--scope", "project/a", "notes/a.md", "--channel", "cloud"}, doc},
		{[]string{"brain", "read", "notes/a.md", "--scope", "project/a", "--section", "Part"}, "## Part\nbody\n"},
	} {
		if code, out, errOut := run(c.args...); code != 0 || out != c.want || errOut != "" {
			t.Fatalf("%q: code %d\nout %q\nerr %q", c.args, code, out, errOut)
		}
	}
}

func TestBrainReadRefusesWithTheReasonAndNoOutput(t *testing.T) {
	w := brainWorld(t, "[privacy]\nnever = [\"secret/**\"]\n[layout]\nreview = \"review\"\n", map[string]string{
		"notes/a.md":     "# A\n",
		"secret/k.md":    "key\n",
		"review/case.md": "case\n",
	})
	_, missing := os.ReadFile(filepath.Join(w.area, "notes", "gone.md"))

	for _, c := range []struct {
		args   []string
		stderr string
	}{
		{[]string{"../x.md", "--scope", "project/a"}, "error: project/a/../x.md leaves the area\n"},
		{[]string{"secret/k.md", "--scope", "project/a"}, "error: project/a/secret/k.md is excluded by [privacy] never\n"},
		{[]string{"review/case.md", "--scope", "project/a", "--channel", "cloud"}, "error: project/a/review/case.md is the review centre; refused on the cloud channel\n"},
		{[]string{"notes/a.md", "--scope", "project/a", "--section", "Nope"}, "error: no section titled 'Nope'\n"},
		{[]string{"notes/a.md", "--scope", "nope"}, "error: unknown scope 'nope'; known scopes are: project/a\n"},
		{[]string{"notes/gone.md", "--scope", "project/a"}, "error: " + missing.Error() + "\n"},
	} {
		code, out, errOut := run(append([]string{"brain", "read"}, c.args...)...)
		if code != 1 || out != "" || errOut != c.stderr {
			t.Fatalf("%q: code %d\nout %q\n got %q\nwant %q", c.args, code, out, errOut, c.stderr)
		}
	}
	if code, out, _ := run("brain", "read", "review/case.md", "--scope", "project/a"); code != 0 || out != "case\n" {
		t.Fatalf("the review centre stays open locally: code %d, out %q", code, out)
	}
}

func TestBrainNeighborsListsBothDirections(t *testing.T) {
	brainWorld(t, "", map[string]string{"graph.json": brainGraph})

	for _, c := range []struct {
		relative string
		want     string
	}{
		{"notes/a.md", "incoming: notes/c.md\noutgoing: notes/b.md\n"},
		{`notes\a.md`, "incoming: notes/c.md\noutgoing: notes/b.md\n"},
		{"notes/z.md", "incoming: -\noutgoing: -\n"},
	} {
		code, out, errOut := run("brain", "neighbors", c.relative, "--scope", "project/a")
		if code != 0 || out != c.want || errOut != "" {
			t.Fatalf("%s: code %d\nout %q\nerr %q", c.relative, code, out, errOut)
		}
	}
}

func TestBrainNeighborsRefusesWithTheReasonAndNoOutput(t *testing.T) {
	brainWorld(t, "", nil)

	for _, c := range []struct {
		args   []string
		stderr string
	}{
		{[]string{"notes/a.md", "--scope", "project/a"}, "error: project/a: never indexed; run `brain reindex`\n"},
		{[]string{"../x.md", "--scope", "project/a"}, "error: project/a/../x.md leaves the area\n"},
		{[]string{"notes/a.md", "--scope", "nope"}, "error: unknown scope 'nope'; known scopes are: project/a\n"},
	} {
		code, out, errOut := run(append([]string{"brain", "neighbors"}, c.args...)...)
		if code != 1 || out != "" || errOut != c.stderr {
			t.Fatalf("%q: code %d\nout %q\n got %q\nwant %q", c.args, code, out, errOut, c.stderr)
		}
	}
}

func TestBrainStatusPrintsEveryLineAtBrainNow(t *testing.T) {
	w := brainWorld(t, "", nil)
	writeFile(t, filepath.Join(w.legacy, "maintenance", "last-run.txt"), "2999-01-01T00:00:00+00:00\n")
	stubBrainNow(t, time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC))
	fake := search.NewFakePort()
	stubBrainStatusPort(t, fake)

	code, out, errOut := run("brain", "status")
	want := "last reconcile: 2999-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`\n" +
		"project/a: never indexed; run `brain reindex`\n"
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	if fake.SearchableCounts != 1 {
		t.Fatalf("the backlog was asked %d times", fake.SearchableCounts)
	}
}

func TestBrainStatusOnTheCloudLeavesALocalOnlyAreaOut(t *testing.T) {
	brainWorld(t, "[privacy]\nmode = \"local_only\"\n", nil)
	stubBrainStatusPort(t, search.NewFakePort())

	code, out, errOut := run("brain", "status", "--channel", "cloud")
	if code != 0 || out != "last reconcile: never; run `brain reconcile`\n" || errOut != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainRuntimeErrorsWriteOnlyTheErrorLine(t *testing.T) {
	w := brainWorld(t, "", nil)
	writeFile(t, filepath.Join(w.state, "registry.toml"), "[[area]\n")
	stubBrainSearchPort(t, search.NewFakePort(), "")
	stubBrainStatusPort(t, search.NewFakePort())

	for _, args := range [][]string{
		{"search", "q"},
		{"catalog"},
		{"read", "notes/a.md", "--scope", "project/a"},
		{"neighbors", "notes/a.md", "--scope", "project/a"},
		{"status"},
	} {
		code, out, errOut := run(append([]string{"brain"}, args...)...)
		// The wording belongs to config.ReadRegistry; this test pins only
		// that it arrives as one error line and nothing else.
		if code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") ||
			!strings.Contains(errOut, "registry.toml: not valid TOML") || !strings.HasSuffix(errOut, "\n") {
			t.Fatalf("%q: code %d\nout %q\nerr %q", args, code, out, errOut)
		}
	}
}
```

Woher die Zeichenketten kommen: `note: {befund}`/`error: {grund}` und die Reihenfolge stdout vor Befunden aus `cli.py:388-406` und `:2435`; `no matches`, der Befund „zweimal leer" (core.py:125-128), der Registerbefund (core.py:185) und der Stempelbefund (core.py:387-389) wörtlich; `render_neighbors` (daemon/tools.py:32-35) plus `print`; `catalog` und `read` mit `print(end="")` (cli.py:785, :788-797) — die Ausgabe ist der Text unverändert; `_section` (core.py:542-564) auf `"# Top\nintro\n## Part\nbody\n# Next\nrest\n"` mit `Part` ergibt `## Part\nbody\n` (Ebene 2 endet an `# Next`). `{score:.0%}` von 0.5 und 1 ist `50%` und `100%` (Faktenblatt search, Teil 1, Punkt 7).

- [ ] **Step 6: Scheitern sehen**

```bash
go test ./internal/cli/ -count=1
```
Expected (gemessen 2026-09-15 im Wegwerfmodul: der committete Baum, `brainargs.go` und beide Testdateien, Stümpfe für die Pakete aus Task 2, 3 und 8; Zeile 1 von `brain_test.go` ist `package cli`, Go bricht nach zehn Fehlern ab):
```
# github.com/xidus90/loomux/internal/cli [github.com/xidus90/loomux/internal/cli.test]
internal\cli\brain_test.go:57:11: undefined: brainSearchPort
internal\cli\brain_test.go:58:2: undefined: brainSearchPort
internal\cli\brain_test.go:64:21: undefined: brainSearchPort
internal\cli\brain_test.go:69:11: undefined: brainStatusPort
internal\cli\brain_test.go:70:2: undefined: brainStatusPort
internal\cli\brain_test.go:71:21: undefined: brainStatusPort
internal\cli\brain_test.go:76:11: undefined: brainNow
internal\cli\brain_test.go:77:2: undefined: brainNow
internal\cli\brain_test.go:78:21: undefined: brainNow
internal\cli\brain_test.go:118:14: undefined: brainSearchPort
internal\cli\brain_test.go:118:14: too many errors
FAIL	github.com/xidus90/loomux/internal/cli [build failed]
FAIL
```

- [ ] **Step 7: Den Befehl schreiben** — `internal/cli/brain.go`

```go
package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/catalog"
	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/reader"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/brain/status"
	"github.com/xidus90/loomux/internal/config"
)

// brainSearchPort is the engine `brain search` asks: the warm qmd daemon over
// MCP. notice hears it when this call had to start the daemon first.
var brainSearchPort = func(notice func(string)) search.SearchPort {
	return search.NewQmdMcpPort(search.WithNotice(notice))
}

// brainStatusPort is the engine `brain status` asks: the qmd command line,
// because the daemon has no `ls` and counts its backlog with another model.
var brainStatusPort = func() search.SearchPort {
	return &search.QmdPort{Executable: "qmd", Runner: search.DefaultRunner}
}

// brainNow is the clock a reconcile stamp is judged against.
var brainNow = time.Now

// brainCommand is `loomux brain`: the five read commands of brain-mcp
// (cli.py:712-810) with their argument forms, exit codes and error line.
func brainCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		brainRefuse(stderr, brainTopUsage(), "loomux brain", "the following arguments are required: command")
		return 2
	}
	parser, known := brainParserFor(args[0])
	if !known {
		brainRefuse(stderr, brainTopUsage(), "loomux brain",
			"argument command: invalid choice: "+pytext.Repr(args[0])+" (choose from "+brainChoices(brainSubcommands)+")")
		return 2
	}
	parsed, refused := parser.parse(args[1:])
	if refused != nil {
		if refused.top {
			brainRefuse(stderr, brainTopUsage(), "loomux brain", refused.message)
		} else {
			brainRefuse(stderr, parser.usage(), "loomux brain "+parser.name, refused.message)
		}
		return 2
	}
	// Nothing reaches stdout before the answer stands: a failure after half an
	// answer would leave the reader holding lines that look complete.
	out, notes, err := brainRun(parser.name, parsed, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	io.WriteString(stdout, out)
	for _, note := range notes {
		fmt.Fprintf(stderr, "note: %s\n", note)
	}
	return 0
}

// brainRefuse writes argparse's usage error: the usage line, then which parser
// refused and why.
func brainRefuse(stderr io.Writer, usage, prog, message string) {
	fmt.Fprintf(stderr, "%s\n%s: error: %s\n", usage, prog, message)
}

// brainRun answers one parsed subcommand: stdout, the notes for stderr, or the
// error. The registry is loomux's; artefacts of read-only areas and the stamp
// stay in ultra-brain's state directory until stage 3.
func brainRun(name string, a brainArgs, stderr io.Writer) (string, []string, error) {
	registryDir, legacyDir := config.StateDir(), config.LegacyBrainDirUntilStage3()
	// The parser let through only the two channel names, so the conversion
	// cannot produce a third.
	channel := privacy.Channel(a.values["--channel"])
	switch name {
	case "search":
		return brainSearch(a, channel, registryDir, legacyDir, stderr)
	case "catalog":
		text, err := brainCatalog(a.values["--scope"], channel, registryDir, legacyDir)
		return text, nil, err
	case "read":
		text, err := brainRead(a.positional, a.values["--scope"], a.values["--section"], channel, registryDir, legacyDir)
		return text, nil, err
	case "neighbors":
		text, err := brainNeighbors(a.positional, a.values["--scope"], channel, registryDir, legacyDir)
		return text, nil, err
	}
	text, err := brainStatus(channel, registryDir, legacyDir)
	return text, nil, err
}

// brainSearch is core.search plus _print_search: the hits for stdout, the
// findings as notes.
func brainSearch(a brainArgs, channel privacy.Channel, registryDir, legacyDir string, stderr io.Writer) (string, []string, error) {
	port := brainSearchPort(func(message string) {
		fmt.Fprintf(stderr, "note: %s\n", message)
	})
	if closer, ok := port.(io.Closer); ok {
		// Let go of the session whatever the answer was; the answer does not
		// depend on how letting go went.
		defer closer.Close()
	}
	answer, err := search.ExecuteSearch(a.positional, a.values["--scope"], search.Profile(a.values["--profile"]),
		a.n, channel, port, registryDir, legacyDir, brainNow())
	if err != nil {
		return "", nil, err
	}
	return search.FormatSearch(answer), answer.Findings, nil
}

// brainCatalog is core.catalog: the root catalog of the visible areas, or the
// index.md of one of them.
func brainCatalog(scope string, channel privacy.Channel, registryDir, legacyDir string) (string, error) {
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, "all", channel)
	if err != nil {
		return "", err
	}
	if scope == "all" {
		plain := make([]config.Area, len(areas))
		for i, visible := range areas {
			plain[i] = visible.Area
		}
		return catalog.RenderRootCatalog(plain), nil
	}
	area, err := privacy.Single(areas, scope)
	if err != nil {
		return "", err
	}
	return catalog.ReadAreaCatalog(area.Area, legacyDir)
}

// brainArea is the one visible area a read or a neighbour query names; every
// area of the registry is checked first, as core._visible_areas("all") does.
func brainArea(scope string, channel privacy.Channel, registryDir, legacyDir string) (privacy.VisibleArea, error) {
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, "all", channel)
	if err != nil {
		return privacy.VisibleArea{}, err
	}
	return privacy.Single(areas, scope)
}

// brainRead is core.read.
func brainRead(relative, scope, section string, channel privacy.Channel, registryDir, legacyDir string) (string, error) {
	area, err := brainArea(scope, channel, registryDir, legacyDir)
	if err != nil {
		return "", err
	}
	return reader.ReadDocument(area.Area, area.Manifest, relative, section, channel, legacyDir)
}

// brainNeighbors is core.neighbors plus render_neighbors: containment before
// the graph is read, so a path leaving the area is refused even where no graph
// exists.
func brainNeighbors(relative, scope string, channel privacy.Channel, registryDir, legacyDir string) (string, error) {
	area, err := brainArea(scope, channel, registryDir, legacyDir)
	if err != nil {
		return "", err
	}
	inside, err := privacy.Contained(scope, relative)
	if err != nil {
		return "", err
	}
	g, err := graph.ReadGraph(area.Area, legacyDir)
	if err != nil {
		return "", err
	}
	incoming, outgoing := graph.Neighbors(g, inside)
	return graph.RenderNeighbors(incoming, outgoing), nil
}

// brainStatus is core.status plus _print_status: one line each.
func brainStatus(channel privacy.Channel, registryDir, legacyDir string) (string, error) {
	lines, err := status.Lines(channel, brainStatusPort(), registryDir, legacyDir, brainNow())
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	return b.String(), nil
}
```

Dann `internal/cli/commands.go`, Map `commands`. Vorher:
```go
var commands = map[string]command{
	"check":     checkCommand,
```
Nachher:
```go
var commands = map[string]command{
	"brain":     brainCommand,
	"check":     checkCommand,
```
Der Rest der Datei bleibt wortgleich.

- [ ] **Step 8: Bestehen sehen**

```bash
gofmt -l internal/cli
```
Expected: keine Ausgabe.

```bash
go vet ./internal/cli/
```
Expected: keine Ausgabe, Exit 0 (gemessen 2026-09-15 im Wegwerfmodul: der committete Baum mit `brain.go`, `brainargs.go`, beiden Testdateien, dem geänderten `commands.go` und Stümpfen mit den Signaturen der Tasks 1 bis 9; `gofmt -l internal/cli` war dort ebenfalls leer).

```bash
go test ./internal/cli/ -count=1
```
Expected: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/cli` beginnt, gefolgt von der Dauer. Scheitert eine Zeile aus `brain_test.go` an einem Wortlaut, der einem anderen Task gehört (Liste „Stützen" oben), gilt dessen Vertrag: den Befund melden, nicht die Erwartung hier dem Ist anpassen.

- [ ] **Step 9: Coverage** (Verfahren Schritt 6)

```bash
go test ./internal/cli/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Expected: Exit 0 und eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/cli` beginnt (Dauer und Prozentsatz über `./...` variieren).

```bash
go tool cover -func="$TEMP/pkg.out" | grep -E 'internal/cli/brain(args)?\.go'
```
Expected: jede Funktion aus `brain.go` und `brainargs.go` 100.0%, genau diese 22 Zeilen:
```
github.com/xidus90/loomux/internal/cli/brain.go:36:		brainCommand		100.0%
github.com/xidus90/loomux/internal/cli/brain.go:72:		brainRefuse		100.0%
github.com/xidus90/loomux/internal/cli/brain.go:79:		brainRun		100.0%
github.com/xidus90/loomux/internal/cli/brain.go:103:		brainSearch		100.0%
github.com/xidus90/loomux/internal/cli/brain.go:122:		brainCatalog		100.0%
github.com/xidus90/loomux/internal/cli/brain.go:143:		brainArea		100.0%
github.com/xidus90/loomux/internal/cli/brain.go:152:		brainRead		100.0%
github.com/xidus90/loomux/internal/cli/brain.go:163:		brainNeighbors		100.0%
github.com/xidus90/loomux/internal/cli/brain.go:181:		brainStatus		100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:83:		brainTopUsage		100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:89:		brainChoices		100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:100:	brainParserFor		100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:132:	usage			100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:152:	option			100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:164:	classify		100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:194:	brainLooksNegative	100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:204:	parse			100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:276:	positionalAt		100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:298:	optionalAt		100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:315:	take			100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:335:	brainAtLeastOne		100.0%
github.com/xidus90/loomux/internal/cli/brainargs.go:353:	brainPythonInt		100.0%
```
Die Zeilennummern sind aus den Blöcken in Step 3 und Step 7 gezählt (Zeile 1 ist jeweils `package cli`) und variieren mit dem Kommentar. Die `brainargs.go`-Zeilen sind am 2026-09-15 im Wegwerfmodul gemessen (nur die Leser-Tests, jede Funktion 100.0%). Die `brain.go`-Zeilen sind nicht gemessen, weil die echten Pakete der Tasks 1 bis 9 dort fehlen; sie folgen aus der Zuordnung jedes Zweigs zu einem Test:
- `len(args) == 0`, unbekannter Unterbefehl, der `top`-Zweig und die Usage des Unterbefehls in `brainCommand` → `TestBrainUsageErrorsNameTheParserThatRefused`.
- Der Fehlerzweig in `brainCommand` und alle fünf Fälle von `brainRun`, dazu die Registry-Fehler in `brainCatalog`, `brainArea` und `brainStatus` → `TestBrainRuntimeErrorsWriteOnlyTheErrorLine`.
- Die Befund-Schleife in `brainCommand` und der Rumpf der Notiz-Funktion in `brainSearch` → `TestBrainSearchPrintsHitsAndThenItsNotes`.
- Der `io.Closer`-Zweig und der Fehler von `ExecuteSearch` → `TestBrainSearchClosesThePortAndReportsTheEngine`.
- `brainCatalog` ohne Scope (mit Schleifenrumpf) und mit Scope → `TestBrainCatalogAnswersTheRootOrOneArea`; der `Single`-Fehler in `brainCatalog` → `TestBrainCatalogHidesALocalOnlyAreaFromTheCloud`.
- Der `Single`-Fehler in `brainArea` und der Fehler von `ReadDocument` → `TestBrainReadRefusesWithTheReasonAndNoOutput`.
- Die Fehler von `Contained` und `ReadGraph` in `brainNeighbors` → `TestBrainNeighborsRefusesWithTheReasonAndNoOutput`; der Erfolgsweg → `TestBrainNeighborsListsBothDirections`.
- Die Schleife in `brainStatus` → `TestBrainStatusPrintsEveryLineAtBrainNow`.

Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund. Für dieses Paket ist keine Ausnahme vorgesehen.

- [ ] **Step 10: Stagen, Tor, Commit**

Erst stagen, dann das Tor: `.githooks/pre-commit` bricht bei ungetrackten oder ungestagten `*.go` ab.

```bash
git add internal/cli/brain.go internal/cli/brainargs.go internal/cli/brain_test.go internal/cli/brainargs_test.go internal/cli/commands.go
```
Expected: Exit 0, keine Ausgabe außer einer möglichen Zeilenende-Warnung von Git.

```bash
sh .githooks/pre-commit
```
Expected: Exit 0 (`covergate` ohne Befund, `bin/loomux.exe` neu gebaut).

```bash
git branch --show-current
```
Expected: `sdd-1b-1` (Hauptcheckout: `master`).

```bash
git rev-parse --short HEAD
```
Expected: der Kurzhash des Commits aus Task 9.

```bash
git diff --cached --stat
```
Expected: genau `internal/cli/brain.go`, `internal/cli/brain_test.go`, `internal/cli/brainargs.go`, `internal/cli/brainargs_test.go` und `internal/cli/commands.go`, Summenzeile `5 files changed` mit den Einfügungen.

```bash
printf '%s\n' 'Answer search, catalog, read, neighbors and status under loomux brain' '' 'The arguments are read the way argparse reads the brain-mcp forms, with' 'its usage errors and exit 2. A failure is one error line with exit 1 and' 'nothing on stdout; search writes its findings as notes after the hits.' > "$TEMP/loomux-msg.txt"
```
Expected: keine Ausgabe.

```bash
git commit -F "$TEMP/loomux-msg.txt"
```
Expected: Exit 0. Der Commit löst das Tor noch einmal aus; das ist gewollt. Danach steht die Zeile `[sdd-1b-1 <kurzhash>] Answer search, catalog, read, neighbors and status under loomux brain` (Hauptcheckout: `[master <kurzhash>]`) und `5 files changed`.

```bash
git log -1 --format='%an <%ae>'
```
Expected: die Nutzeridentität aus `git config user.name` und `git config user.email`, kein Modell.

```bash
git log -1 --format=%B
```
Expected: genau die Nachricht aus `$TEMP/loomux-msg.txt`, keine `Co-Authored-By`-Zeile.

- [ ] **Step 11: Bericht**

Paritätszeilen, die dieser Task schafft (Task 12 trägt sie ein):
1. Abkürzungen langer Optionen: Python erweitert `--prof` zu `--profile` und meldet `--s` als `ambiguous option: --s could match --scope, --state-dir` (gemessen); loomux kennt nur ganze Optionsnamen und meldet `unrecognized arguments: --prof x` bzw. `--s x`, Exit 2.
2. `--state-dir`: Python nimmt ihn an jedem der fünf Befehle; loomux meldet `unrecognized arguments: --state-dir {wert}` (im Test `--state-dir x`), Exit 2 — der Zustand kommt aus `LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR`. Die Zeile bleibt, bis der Mensch den Befund zum Vertrag unten bei der Freigabe der Paritätsliste entscheidet.
3. `-h`/`--help`: Python druckt die Hilfe auf stdout mit Exit 0, auch angeklebt (`search q -hx`, gemessen); loomux kennt keine Hilfe, das Wort endet als `unrecognized arguments` bzw. hinter dem Fehler über fehlende Argumente, Exit 2.
4. `-n` mit Nicht-ASCII-Ziffern (`٣`, `３`): Python nimmt 3 (gemessen); loomux meldet `invalid _at_least_one value`, Exit 2.
5. `-n` über `math.MaxInt` (`99999999999999999999999`): Python behält die Zahl (gemessen, Exit 0); loomux kappt auf `math.MaxInt`.
6. Usage-Zeilen: Python bricht nach Terminalbreite um, nennt `[-h]` und `[--state-dir STATE_DIR]`, und die oberste Usage sowie `argument command: invalid choice` zählen 23 Unterbefehle; loomux schreibt eine Zeile ohne beide Optionen und nennt nur die fünf. Der Korpus vergleicht stderr nicht.
7. Wörter vor dem Unterbefehl: Python nimmt `brain-mcp -- search q` an und meldet `brain-mcp -x search q` als `unrecognized arguments: -x` (gemessen); loomux liest das erste Wort als Unterbefehl und meldet `argument command: invalid choice: '--'` bzw. `'-x'`, Exit 2.
8. Brain-Daemon: Python leitet `search|catalog|read|neighbors|status` über einen laufenden brain-Daemon und druckt dann dessen Format (cli.py:769-771, `_through_daemon`); loomux fragt nie einen brain-Daemon.

Mitgeführt, nicht neu: der Aufwärm-Hinweis (Spec-Zeile 5; hier als `note: {hinweis}` auf stderr verdrahtet), `read --section ""` (Task 6), fehlende `index.md` im Go-Wortlaut (Task 7), fehlendes Manifest im loomux-Wortlaut (Task 3).

Befunde zum Vertrag:
1. Die drei Wortlaute unter „Abweichungen vom Vertragstext" folgen der gemessenen Quelle. Vertrag Task 10, Regel 2, und die Consumes-Zeile von Task 16 sind danach nachzuziehen: Fehler des obersten Parsers heißen `loomux brain: error: {meldung}`, alle anderen Usage-Fehler `loomux brain {sub}: error: {meldung}`.
2. `privacy.ParseChannel` (Vertrag, Consumes von Task 10) wird nicht gerufen: der Leser lässt nur die beiden aus `privacy.ChannelLocal` und `privacy.ChannelCloud` gebauten Wahlen durch, wie cli.py mit `choices=[c.value for c in Channel]`; `brainRun` wandelt den Wert darum ohne Prüfung in `privacy.Channel`.
3. `--state-dir`: Die Spec verlangt ihn (Abschnitt „Ausgangslage, vermessen am 2026-09-15": „Alle fünf nehmen `--channel local|cloud` (Vorgabe `local`) und `--state-dir`"; Abschnitt „Befehle, Ausgabe, Datenfluss": „Argumentformen und Vorgaben wie die Referenz"), der Vertrag schließt ihn aus („Nicht unterstützt (Paritätszeile): Abkürzungen wie `--prof`, `--state-dir`"). Nach dem Vertragskopf gilt bei Widerspruch die Spec, darum steht es hier als Befund. Der Grund, ihn trotzdem wegzulassen: loomux liest aus zwei Verzeichnissen — die Registry aus `LOOMUX_STATE_DIR` (`config.StateDir()`), Artefakte schreibgeschützter Bereiche und den Reconcile-Stempel aus `LOOMUX_LEGACY_BRAIN_DIR` (`config.LegacyBrainDirUntilStage3()`) —, und ein einzelnes `--state-dir` hat darüber keine eindeutige Bedeutung. Der Mensch entscheidet bei der Freigabe der Paritätsliste; bis dahin bleibt Paritätszeile 2, und der Leser kennt die Option nicht.
4. Die READMEs pflegt Task 16; dieser Task ändert sie nicht.

---

### Task 11: Werkzeuge für den 1b-1-Korpus — Fake-qmd, Programmform, Faltung

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 10 sind committet.

**Files:**
- Create: `internal/dev/fakeqmd/fakeqmd.go`, `internal/dev/fakeqmd/fakeqmd_test.go`, `internal/dev/fakeqmd/qmd/main.go`
- Modify: `internal/dev/recordcase/recordcase.go`, `internal/dev/recordcase/recordcase_test.go`
- Modify: `internal/dev/importcases/importcases.go`, `internal/dev/importcases/importcases_test.go`
- Modify: `internal/cli/dev.go`, `internal/cli/dev_test.go`
- Unberührt und am Ende geprüft: `testdata/cases/1a-source/**`, `testdata/cases/1a/**`

**Interfaces:**
- Consumes:
  - `internal/cases` (heutiger Code): `cases.SplitCommand(s string) ([]string, error)` (`runner.go:145`; eine Zeile aus lauter Leerzeichen ergibt `nil, nil`, ein offenes Anführungszeichen `unclosed quote or escape in command: <s>`), `cases.WorldToken` (`"{{WORLD}}"`), `cases.Normalize(data []byte, dir string) []byte`. `Record` ruft `cases.StageWorld` und `cases.CompareTrees` weiter wie bisher; dieser Task ruft sie nicht neu.
  - `internal/config` (heutiger Code): `config.ReadRegistry(stateDir string) ([]config.Area, error)` (`registry.go:91`) liest genau `<stateDir>/registry.toml`; jeder Lesefehler (fehlende Datei, kein TOML) kommt als Fehler mit dem Pfad, ein Eintrag ohne `scope` oder `path` fällt still weg. Genutztes Feld: `config.Area.Path string`.
  - `internal/dev/recordcase` (heutiger Code, geändert): `type Spec struct { Exe, Cmd, World, Stdin, Out, Notes, Compare string }`, `func Record(s Spec) error`; im Test `const helperEnv = "LOOMUX_RECORDCASE_HELPER"`, `func TestHelperProcess(t *testing.T)` (Modi `echo`, `write`, sonst `refused` mit Exit 2), `func helperSpec(t *testing.T, mode, cmd string) Spec`, `func read(t *testing.T, parts ...string) string`.
  - `internal/dev/importcases` (heutiger Code, geändert): `func TranslateWorld(dir string) error`, unverändert genutzt `func translateDir(dir string) error`; im Test `func decodeConfig(t *testing.T, dir string) map[string]any`, `func entries(t *testing.T, dir string) []string`.
  - `internal/cli` (heutiger Code, geändert): `func devRecordCase(args []string, _ io.Reader, _, stderr io.Writer) int`; im Test `func run(args ...string) (int, string, string)` (`cli_test.go:10`).
  - Task 8, nur im Test von `fakeqmd`, Paket `internal/brain/search`:
    - `func search.NewQmdMcpPort(opts ...search.QmdMcpOption) *search.QmdMcpPort`; `func search.WithConnect(fn search.ConnectFunc) search.QmdMcpOption`; `type search.ConnectFunc func(env map[string]string) (search.Session, error)`
    - `type search.Session interface { Call(name string, arguments map[string]any) (map[string]any, error); Close() error }`; `type search.HTTPSession struct` mit dem genutzten Feld `URL string` (ohne `URL` baut `url()` `http://localhost:<port>/mcp`, deshalb das Literal `&search.HTTPSession{URL: server.URL}`)
    - `type search.QmdPort struct { Executable string; Runner search.RunnerFunc }`; `type search.RunnerFunc func(argv []string) ([]byte, []byte, int, error)`
    - `type search.Profile string` mit `search.ProfileFull`, `search.ProfileKeyword`
    - `type search.SearchHit struct { Collection string; Scope string; Relative string; Line int; Title string; Snippet string; Score float64; ContentKey string }`
    - Methoden: `(*search.QmdMcpPort).Search(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error)`, `(*search.QmdPort).Search(` mit derselben Signatur `)`, `(*search.QmdPort).Indexed(collection string) ([]string, error)`, `(*search.QmdPort).NotYetSearchable() (int, error)`
    - Zusicherungen: ein Exit ≠ 0 des CLI-Ports wird `"{argv0} exited with {code}: {pytext.Strip(stderr)}"`; eine JSON-RPC-Antwort mit `error` wird im MCP-Port ein Fehler (ub `http.go:194-196` `qmd refused %s: %v`, nach allen Versuchen `the search engine did not answer in N attempts: ` gefolgt von den Fehlern der Versuche, mit `; ` getrennt, ub `mcp.go:146`); **`translateReply` entfernt das Präfix `N: ` jeder Snippet-Zeile, wenn jede Zeile ihr Präfix ab `line` trägt (Task 8, Ruling R4)**. `TestLoomuxsPortsReadTheFake` setzt diese Entfernung voraus.
  - Antwortformen, gegen die der Fake spricht: `translateReply` (ub `pkg/search/mcp.go:211-262`: `result.structuredContent.results[]` mit `file` = `c/rel`, `line`, `score`, `docid`, `title`, `snippet`) und `_parse`/`_hit` (ub `src/brain/search/qmd.py:238-266`: JSON-Array, `file` = `qmd://c/rel`, `line` ganzzahlig und Pflicht, `score` Zahl, `docid` Zeichenkette, `title`/`snippet` mit Vorgabe `""`)
- Produces (Vertrag, mit den Änderungen der Rulings R5 und R6; die MCP-Handler-Regel des Vertrags ist durch R5 überholt):
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
	SearchError string              `json:"search_error"` // NEU (R5): nicht leer: search|vsearch|query enden mit Exit 1 und diesem stderr, MCP query antwortet mit JSON-RPC-Fehler -32603
	Hits        []Hit               `json:"hits"`         // search|vsearch|query --json und MCP query
}
func Load(path string) (*Fixture, error)                                // fehlt die Datei → leere Fixture
func (f *Fixture) RunCLI(args []string, stdout, stderr io.Writer) int
func (f *Fixture) MCPHandler() http.Handler                             // GEÄNDERT (R5): snippet nummeriert wie qmds addLineNumbers ab hit.Line
func Main(args []string, getenv func(string) string, stdout, stderr io.Writer) int // Fixture aus FixtureEnv, dann RunCLI; ohne Variable Exit 2
// internal/dev/fakeqmd/qmd/main.go: package main; func main() { os.Exit(fakeqmd.Main(os.Args[1:], os.Getenv, os.Stdout, os.Stderr)) } mit //coverage:exempt

// recordcase.Spec — neue Felder
//   Argv        []string // Programm und führende Argumente; ersetzt das erste Token von Cmd; schließt Exe aus
//   Env         []string // KEY=VALUE, {{WORLD}} wird durch die gestagte Welt ersetzt
//   PathPrepend string   // Verzeichnis vor PATH des aufgezeichneten Prozesses
// immer: BRAIN_STATE_DIR=<welt>, PYTHONUTF8=1; stdout "\r\n" → "\n" vor Normalize
// importcases.TranslateWorld(dir string) error — GEÄNDERT: faltet zusätzlich in jedem Verzeichnis, das ein `path` in dir/registry.toml als "{{WORLD}}/<lokaler pfad>" nennt; eine fehlende oder unlesbare Registry faltet nichts zusätzlich und ist kein Fehler (R6)
```
- Produces, Kommandozeile `loomux dev record-case` (Exit und stderr gemessen im Wegwerfmodul):
  - neue Flags `--argv "<programm und führende argumente>"` (über `cases.SplitCommand`), `--env KEY=VALUE` (wiederholbar, `{{WORLD}}` erlaubt), `--path-prepend DIR`
  - `--exe` und `--argv` zusammen → Exit 2, `loomux dev record-case: --exe and --argv exclude each other`
  - `--argv` nicht zerlegbar → Exit 2, `loomux dev record-case: --argv: unclosed quote or escape in command: <wert>`
  - `--env` ohne `=` → Exit 2, das `flag`-Paket meldet `invalid value "<wert>" for flag -env: "<wert>" is not KEY=VALUE` und die Hilfe
  - Pflichtflags fehlen → Exit 2, `loomux dev record-case: --exe or --argv, --cmd, --world and --out are required`
  - Fehler aus `Record` → Exit 1, `loomux dev record-case: <fehler>`; neu sind `a recording names its program by exactly one of Exe and Argv`, `environment entry "<wert>" is not KEY=VALUE`, `empty command` (ein `Cmd` ohne Token, R28) und `running <programm>: <fehler>`
- Produces, Ergänzungen (alle unexportiert):
  - `fakeqmd`: `const mcpDefaultLimit = 10`; `func (f *Fixture) list(args []string, stdout, stderr io.Writer) int`; `func (f *Fixture) status(args []string, stdout, stderr io.Writer) int`; `func (f *Fixture) search(args []string, stdout, stderr io.Writer) int`; `func searchFlags(args []string, stderr io.Writer) (int, []string, bool)`; `func (f *Fixture) hitsIn(collections []string, limit int) []Hit`; `func addLineNumbers(text string, start int) string`; `type cliHit`, `type mcpHit`, `type rpcRequest`; `func rpcResult(id json.RawMessage, result any) map[string]any`; `func rpcError(id json.RawMessage, code int, message string) map[string]any`; `func writeJSON(w io.Writer, v any)`
  - `recordcase`: `func recordEnv(goos string, base []string, s Spec, tmp string) []string`; `func mergeEnv(goos string, env []string, set ...string) []string`; `func prependPath(goos string, env []string, dir string) []string`; `func sameKey(goos, a, b string) bool`
  - `importcases`: `func registeredDirs(dir string) []string` — ohne Fehlerrückgabe: R6 verlangt, jeden Lesefehler der Registry zu übergehen; ein `error`, der nie gesetzt wird, ließe in `TranslateWorld` einen toten Zweig unter 100 % stehen
  - `cli`: `type envFlags []string` mit `String() string` und `Set(value string) error`

**Woher die Zeichenketten des Fakes stammen** (gemessen am 2026-09-15 oder wörtlich aus qmd 2.8.3, `%APPDATA%/npm/node_modules/@tobilu/qmd/dist`):

| Ausgabe | Wortlaut | Beleg |
|---|---|---|
| `qmd ls <unbekannt>` | Exit 1, stdout leer, stderr `Collection not found: <c>\nRun 'qmd ls' to see available collections.\n` | gemessen (`1b-1-facts/status.md` §9); `cli/qmd.js:1455-1458` |
| `qmd ls <leer>` | Exit 0, stdout `No files found in collection: <c>\n` | `cli/qmd.js:1491` |
| `qmd ls` Zeile | ` 1 B  Jan  1 00:00  qmd://<c>/<rel>` — beide Leser nehmen nur den Teil ab `qmd://` | `cli/qmd.js:1504`; Python `qmd.py:171-178`, Go `qmd.go:178-189` |
| `qmd status` | `Documents\n`, bei N > 0 dazu `  Pending:  N need embedding (run 'qmd embed')\n` | `cli/qmd.js:389-398` |
| `--json` | Array mit 2 Leerzeichen Einzug, Schlüssel `docid, score, file, line, title, snippet`; `snippet` fehlt, wenn leer | `cli/formatter.js:68-79` |
| MCP `query` ohne `limit` | 10 Treffer | Spike `raw/mcp.json` `phrase|vec_limit_omitted`, `rare|vec_limit_omitted` |
| MCP `query` mit `collections: []` | Treffer aus allen Collections | Spike `raw/mcp.json` `rare|vec_collections_empty` |
| MCP `query`, `snippet` | jede Zeile mit `N: ` davor, N = `line` + Zeilenindex, getrennt nur an `\n`; ein leerer Snippet wird `N: ` (bei `line` 1 also `1: `); `--json` der CLI nummeriert nicht | `mcp/server.js:293` und `:301` `snippet: addLineNumbers(snippet, line)`; `store.js:4292-4295` `text.split('\n')` → `${startLine + i}: ${line}` |
| `search_error` der Fixture (Ruling R5, kein qmd-Wortlaut) | CLI `search`/`vsearch`/`query`: Exit 1, stdout leer, stderr = Wert; MCP `query`: JSON-RPC-Fehler, `code` `-32603`, `message` = Wert; `initialize` antwortet weiter | die Ports machen daraus `qmd exited with 1: <Strip(Wert)>` (CLI) und einen Fehler, der `qmd refused tools/call: ` samt dem Wert enthält (MCP) |
| Python-Kette gegen den gebauten Fake | `('notes/a b.md', 'x.md')`, `()`, `2`, Fehler `qmd exited with 1: Collection not found: repo-z\nRun 'qmd ls' to see available collections.`; mit `search_error` `'qmd exited with 1: the index is locked'` | gemessen am 2026-09-15 über `uv run --no-sync --project <ub> python` mit `PYTHONDONTWRITEBYTECODE=1`, Step 5 |

- [ ] **Step 1: Failing test für den Fake** — `internal/dev/fakeqmd/fakeqmd_test.go` anlegen:

```go
package fakeqmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

// world is a fixture with a listed, an empty and an unlisted collection, a
// backlog, and hits in two collections.
func world() *Fixture {
	return &Fixture{
		Collections: map[string][]string{"repo-a": {"notes/a b.md", "x.md"}, "empty": {}},
		Pending:     3,
		Hits: []Hit{
			{Collection: "repo-a", Relative: "notes/a b.md", Line: 3, Score: 0.75, DocID: "#abc123", Title: "A", Snippet: "one\ntwo"},
			{Collection: "repo-b", Relative: "b.md", Line: 1, Score: 0.5, DocID: "#def456", Title: "B"},
			{Collection: "repo-a", Relative: "x.md", Line: 7, Score: 0.25, DocID: "#789abc", Title: "X", Snippet: "x"},
		},
	}
}

func runCLI(f *Fixture, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := f.RunCLI(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestLoadReadsAFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), FixtureName)
	body := `{"collections":{"repo-a":["x.md"]},"pending":2,"status_error":"locked","search_error":"down",` +
		`"hits":[{"collection":"repo-a","relative":"x.md","line":4,"score":0.5,"docid":"#a1","title":"T","snippet":"s"}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := &Fixture{
		Collections: map[string][]string{"repo-a": {"x.md"}},
		Pending:     2,
		StatusError: "locked",
		SearchError: "down",
		Hits:        []Hit{{Collection: "repo-a", Relative: "x.md", Line: 4, Score: 0.5, DocID: "#a1", Title: "T", Snippet: "s"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadWithoutAFileIsAnEngineThatKnowsNothing(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), FixtureName))
	if err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runCLI(f, "ls", "repo-a"); code != 1 || out != "" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestLoadRefusesWhatIsNoFixture(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err == nil {
		t.Error("a directory: want error")
	}
	for name, body := range map[string]string{
		"broken.json":   `{"pending":`,
		"misspelt.json": `{"collection":{}}`,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), path) {
			t.Errorf("%s: want an error naming the file, got %v", name, err)
		}
	}
}

func TestListPrintsTheCollectionAsQmdDoes(t *testing.T) {
	code, out, errOut := runCLI(world(), "ls", "repo-a")
	want := " 1 B  Jan  1 00:00  qmd://repo-a/notes/a b.md\n 1 B  Jan  1 00:00  qmd://repo-a/x.md\n"
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestListOfAnEmptyCollectionSaysSo(t *testing.T) {
	code, out, _ := runCLI(world(), "ls", "empty")
	if code != 0 || out != "No files found in collection: empty\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestListOfAnUnknownCollectionFailsAsQmdDoes(t *testing.T) {
	code, out, errOut := runCLI(world(), "ls", "repo-z")
	want := "Collection not found: repo-z\nRun 'qmd ls' to see available collections.\n"
	if code != 1 || out != "" || errOut != want {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestStatusReportsTheBacklog(t *testing.T) {
	f := world()
	if code, out, _ := runCLI(f, "status"); code != 0 || out != "Documents\n  Pending:  3 need embedding (run 'qmd embed')\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	f.Pending = 0
	if code, out, _ := runCLI(f, "status"); code != 0 || out != "Documents\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestStatusFailsWithTheScriptedError(t *testing.T) {
	f := world()
	f.StatusError = "index is locked\n"
	code, out, errOut := runCLI(f, "status")
	if code != 1 || out != "" || errOut != "index is locked\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestSearchPrintsTheHitsAsQmdJSON(t *testing.T) {
	code, out, errOut := runCLI(world(), "query", "q", "--json", "-n", "1", "-c", "repo-a")
	want := `[
  {
    "docid": "#abc123",
    "score": 0.75,
    "file": "qmd://repo-a/notes/a b.md",
    "line": 3,
    "title": "A",
    "snippet": "one\ntwo"
  }
]
`
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestSearchKeepsTheFixtureOrderAcrossTheNamedCollections(t *testing.T) {
	var rows []cliHit
	code, out, _ := runCLI(world(), "vsearch", "q", "-c", "repo-a", "--json", "-c", "repo-b", "-n", "5")
	if err := json.Unmarshal([]byte(out), &rows); code != 0 || err != nil {
		t.Fatalf("code %d, err %v, out %q", code, err, out)
	}
	var files []string
	for _, row := range rows {
		files = append(files, row.File)
	}
	want := []string{"qmd://repo-a/notes/a b.md", "qmd://repo-b/b.md", "qmd://repo-a/x.md"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("files %v", files)
	}
	if strings.Count(out, `"snippet"`) != 2 {
		t.Errorf("an empty snippet is left out, as qmd does: %s", out)
	}
}

func TestSearchWithoutACollectionSearchesAllAndStopsAtN(t *testing.T) {
	var rows []cliHit
	_, out, _ := runCLI(world(), "search", "q", "--json", "-n", "2")
	if err := json.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 2 || rows[1].File != "qmd://repo-b/b.md" {
		t.Fatalf("err %v, rows %+v", err, rows)
	}
}

func TestSearchWithoutHitsPrintsAnEmptyArray(t *testing.T) {
	if code, out, _ := runCLI(world(), "search", "q", "--json", "-n", "5", "-c", "nowhere"); code != 0 || out != "[]\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// An engine that fails the search fails every search verb, and still refuses
// a call neither port makes before it fails.
func TestSearchFailsWithTheScriptedError(t *testing.T) {
	f := world()
	f.SearchError = "the index is locked\n"
	for _, verb := range []string{"search", "vsearch", "query"} {
		code, out, errOut := runCLI(f, verb, "q", "--json", "-n", "5", "-c", "repo-a")
		if code != 1 || out != "" || errOut != "the index is locked\n" {
			t.Errorf("%s: code %d, out %q, err %q", verb, code, out, errOut)
		}
	}
	if code, _, errOut := runCLI(f, "search", "q", "-n", "5"); code != 2 || !strings.HasPrefix(errOut, "fakeqmd: ") {
		t.Errorf("unexpected call: code %d, err %q", code, errOut)
	}
}

func TestRunCLIRefusesACallNeitherPortMakes(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"embed"},
		{"ls"},
		{"ls", "a", "b"},
		{"status", "--json"},
		{"search"},
		{"search", "q", "-n", "5"},
		{"search", "q", "--json"},
		{"search", "q", "--json", "-n"},
		{"search", "q", "--json", "-n", "0"},
		{"search", "q", "--json", "-n", "x"},
		{"search", "q", "--json", "-n", "5", "--full"},
	} {
		code, out, errOut := runCLI(world(), args...)
		if code != 2 || out != "" || !strings.HasPrefix(errOut, "fakeqmd: ") {
			t.Errorf("%q: code %d, out %q, err %q", args, code, out, errOut)
		}
	}
}

// post sends one JSON-RPC body and decodes the reply.
func post(t *testing.T, url, body string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var reply map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, reply
}

// queryResults digs structuredContent.results out of a query reply.
func queryResults(reply map[string]any) []any {
	result, _ := reply["result"].(map[string]any)
	content, _ := result["structuredContent"].(map[string]any)
	results, _ := content["results"].([]any)
	return results
}

func TestMCPHandlerAnswersTheHandshake(t *testing.T) {
	server := httptest.NewServer(world().MCPHandler())
	defer server.Close()
	_, reply := post(t, server.URL+"/mcp", `{"jsonrpc":"2.0","id":7,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	result, _ := reply["result"].(map[string]any)
	if reply["id"] != float64(7) || result["protocolVersion"] != "2025-06-18" {
		t.Fatalf("reply %v", reply)
	}
}

// qmd 2.8.3 numbers every snippet line on the MCP path, starting at the hit's
// line (dist/mcp/server.js:301, addLineNumbers in dist/store.js); the CLI's
// --json leaves the snippet alone.
func TestMCPHandlerAnswersTheQueryTool(t *testing.T) {
	server := httptest.NewServer(world().MCPHandler())
	defer server.Close()
	_, reply := post(t, server.URL, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"query","arguments":{"query":"q","collections":["repo-a"],"limit":1}}}`)
	results := queryResults(reply)
	if len(results) != 1 {
		t.Fatalf("reply %v", reply)
	}
	want := map[string]any{"docid": "#abc123", "file": "repo-a/notes/a b.md", "title": "A", "score": 0.75, "line": float64(3), "snippet": "3: one\n4: two"}
	if !reflect.DeepEqual(results[0], want) {
		t.Fatalf("hit %v", results[0])
	}
	_, reply = post(t, server.URL, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"query","arguments":{"query":"q","collections":["repo-b"]}}}`)
	results = queryResults(reply)
	if len(results) != 1 {
		t.Fatalf("reply %v", reply)
	}
	if hit, _ := results[0].(map[string]any); hit["snippet"] != "1: " {
		t.Fatalf("an empty snippet is numbered too: %v", results[0])
	}
}

func TestMCPHandlerGivesTenHitsWithoutALimit(t *testing.T) {
	f := &Fixture{}
	for i := range 12 {
		f.Hits = append(f.Hits, Hit{Collection: "c", Relative: fmt.Sprintf("%d.md", i), DocID: "#0"})
	}
	server := httptest.NewServer(f.MCPHandler())
	defer server.Close()
	_, reply := post(t, server.URL, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"query","arguments":{"query":"q"}}}`)
	if results := queryResults(reply); len(results) != 10 {
		t.Fatalf("reply %v", reply)
	}
}

// The engine answers the handshake and fails the search, as a qmd whose index
// breaks under the query does.
func TestMCPHandlerFailsTheQueryWithTheScriptedError(t *testing.T) {
	f := world()
	f.SearchError = "the index is locked\n"
	server := httptest.NewServer(f.MCPHandler())
	defer server.Close()
	_, reply := post(t, server.URL, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"query","arguments":{"query":"q"}}}`)
	rpcErr, _ := reply["error"].(map[string]any)
	if reply["id"] != float64(4) || reply["result"] != nil || rpcErr["code"] != float64(-32603) || rpcErr["message"] != "the index is locked\n" {
		t.Fatalf("reply %v", reply)
	}
	_, reply = post(t, server.URL, `{"jsonrpc":"2.0","id":5,"method":"initialize","params":{}}`)
	if _, ok := reply["result"].(map[string]any); !ok {
		t.Fatalf("handshake %v", reply)
	}
}

func TestMCPHandlerRefusesEveryOtherCall(t *testing.T) {
	server := httptest.NewServer(world().MCPHandler())
	defer server.Close()
	for body, code := range map[string]float64{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get","arguments":{}}}`: -32601,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`:                                        -32601,
		`{`: -32700,
	} {
		_, reply := post(t, server.URL, body)
		rpcErr, _ := reply["error"].(map[string]any)
		if rpcErr["code"] != code || !strings.HasPrefix(rpcErr["message"].(string), "fakeqmd: ") {
			t.Errorf("%s: reply %v", body, reply)
		}
	}
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET: status %d", resp.StatusCode)
	}
}

// mcpPort is loomux's MCP port, connected to a fake served at url.
func mcpPort(url string) *search.QmdMcpPort {
	return search.NewQmdMcpPort(search.WithConnect(func(map[string]string) (search.Session, error) {
		return &search.HTTPSession{URL: url}, nil
	}))
}

// cliPort is loomux's CLI port, answered by f through the Runner seam as the
// fake binary answers it.
func cliPort(f *Fixture) *search.QmdPort {
	return &search.QmdPort{Executable: "qmd", Runner: func(argv []string) ([]byte, []byte, int, error) {
		var out, errb bytes.Buffer
		code := f.RunCLI(argv[1:], &out, &errb)
		return out.Bytes(), errb.Bytes(), code, nil
	}}
}

// The fake is only worth its fixture if loomux's own ports read it: the MCP
// port over HTTP, the CLI port through its Runner seam. The MCP snippets
// arrive numbered; the port hands them on as the CLI port does, so this test
// also holds Task 8's removal of qmd's line prefix.
func TestLoomuxsPortsReadTheFake(t *testing.T) {
	f := world()
	server := httptest.NewServer(f.MCPHandler())
	defer server.Close()
	want := []search.SearchHit{
		{Collection: "repo-a", Relative: "notes/a b.md", Line: 3, Title: "A", Snippet: "one\ntwo", Score: 0.75, ContentKey: "#abc123"},
		{Collection: "repo-a", Relative: "x.md", Line: 7, Title: "X", Snippet: "x", Score: 0.25, ContentKey: "#789abc"},
	}
	hits, err := mcpPort(server.URL).Search("q", []string{"repo-a"}, search.ProfileFull, 5)
	if err != nil || !reflect.DeepEqual(hits, want) {
		t.Fatalf("MCP: err %v, hits %+v", err, hits)
	}

	cli := cliPort(f)
	if hits, err := cli.Search("q", []string{"repo-a"}, search.ProfileKeyword, 5); err != nil || !reflect.DeepEqual(hits, want) {
		t.Fatalf("CLI search: err %v, hits %+v", err, hits)
	}
	if indexed, err := cli.Indexed("repo-a"); err != nil || !reflect.DeepEqual(indexed, []string{"notes/a b.md", "x.md"}) {
		t.Fatalf("indexed %v, err %v", indexed, err)
	}
	if indexed, err := cli.Indexed("empty"); err != nil || len(indexed) != 0 {
		t.Fatalf("empty: indexed %v, err %v", indexed, err)
	}
	if pending, err := cli.NotYetSearchable(); err != nil || pending != 3 {
		t.Fatalf("pending %d, err %v", pending, err)
	}
	// The same bytes the Python reference builds from the same stderr.
	_, err = cli.Indexed("repo-z")
	if err == nil || err.Error() != "qmd exited with 1: Collection not found: repo-z\nRun 'qmd ls' to see available collections." {
		t.Fatalf("err %v", err)
	}
}

// A failing engine fails both ports with its own words, never with an empty
// answer.
func TestLoomuxsPortsFailWhenTheFakeFailsTheSearch(t *testing.T) {
	f := world()
	f.SearchError = "the index is locked\n"
	server := httptest.NewServer(f.MCPHandler())
	defer server.Close()
	hits, err := mcpPort(server.URL).Search("q", []string{"repo-a"}, search.ProfileFull, 5)
	if err == nil || hits != nil || !strings.Contains(err.Error(), "the index is locked") {
		t.Fatalf("MCP: hits %+v, err %v", hits, err)
	}
	// The same bytes the Python reference builds from the same stderr.
	hits, err = cliPort(f).Search("q", []string{"repo-a"}, search.ProfileFull, 5)
	if err == nil || hits != nil || err.Error() != "qmd exited with 1: the index is locked" {
		t.Fatalf("CLI: hits %+v, err %v", hits, err)
	}
}

func TestMainNeedsTheFixtureVariable(t *testing.T) {
	var out, errb bytes.Buffer
	code := Main([]string{"status"}, func(string) string { return "" }, &out, &errb)
	if code != 2 || errb.String() != "fakeqmd: LOOMUX_FAKE_QMD_FIXTURE is not set\n" {
		t.Fatalf("code %d, err %q", code, errb.String())
	}
}

func TestMainAnswersFromTheNamedFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), FixtureName)
	if err := os.WriteFile(path, []byte(`{"pending":4}`), 0o644); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == FixtureEnv {
			return path
		}
		return ""
	}
	var out, errb bytes.Buffer
	if code := Main([]string{"status"}, getenv, &out, &errb); code != 0 || out.String() != "Documents\n  Pending:  4 need embedding (run 'qmd embed')\n" {
		t.Fatalf("code %d, out %q, err %q", code, out.String(), errb.String())
	}
}

func TestMainReportsAFixtureItCannotRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), FixtureName)
	if err := os.WriteFile(path, []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Main([]string{"status"}, func(string) string { return path }, &out, &errb)
	if code != 2 || !strings.HasPrefix(errb.String(), "fakeqmd: "+path) {
		t.Fatalf("code %d, err %q", code, errb.String())
	}
}
```

`TestLoomuxsPortsReadTheFake` und `TestLoomuxsPortsFailWhenTheFakeFailsTheSearch` sprechen die API von Task 8 an. Ändert Task 8 den Wortlaut eines Exit ≠ 0 (Vertrag: `"{argv0} exited with {code}: {pytext.Strip(stderr)}"`), bleiben die erwarteten Zeichenketten dieselben — `Strip` und `TrimSpace` sind für diese stderr gleich, und Python liefert genau diese Bytes (gemessen, siehe Tabelle). Der MCP-Weg liefert `"3: one\n4: two"` und `"7: x"`; dass der Port `"one\ntwo"` und `"x"` zurückgibt, ist die Präfixentfernung aus Task 8 (R4). Fehlt sie, scheitert `TestLoomuxsPortsReadTheFake` mit `MCP: err <nil>, hits` und den nummerierten Snippets: dann ist Task 8 unvollständig, nicht der Fake (gemessen im Wegwerfmodul mit ub `pkg/search` plus der Entfernung nach F67: grün).

- [ ] **Step 2: Scheitern sehen**

```bash
go test ./internal/dev/fakeqmd/
```
Erwartet: Exit 1, FAIL beim Übersetzen; die letzten Zeilen sind `FAIL	github.com/xidus90/loomux/internal/dev/fakeqmd [build failed]` und `FAIL`.
```bash
go test ./internal/dev/fakeqmd/ 2>&1 | grep -E 'fakeqmd_test\.go:[0-9]+:[0-9]+: undefined: (Fixture|Hit|FixtureName|Load)$'
```
Erwartet: Treffer (gemessen am 2026-09-15, die ersten vier: `internal\dev\fakeqmd\fakeqmd_test.go:20:15: undefined: Fixture`, `:21:10: undefined: Fixture`, `:24:11: undefined: Hit`, `:32:16: undefined: Fixture`). Zeilennummern variieren mit dem Kommentar.

- [ ] **Step 3: Fake implementieren** — `internal/dev/fakeqmd/fakeqmd.go`:

```go
// Package fakeqmd stands in for qmd while the corpus of stage 1b-1 is recorded
// and replayed. One fixture per world says which documents each collection
// holds, how many wait for embedding and what a search finds; the package
// answers qmd's command line and its MCP query tool from it, so the Python
// reference and loomux ask the same engine and no model or GPU is involved.
package fakeqmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
)

// FixtureName is the fixture's file name in the root of a world.
const FixtureName = "qmd-fixture.json"

// FixtureEnv names the fixture the fake qmd binary answers from.
const FixtureEnv = "LOOMUX_FAKE_QMD_FIXTURE"

// mcpDefaultLimit is what qmd 2.8.3's MCP query returned without a limit in
// the spike of 2026-09-15: ten hits.
const mcpDefaultLimit = 10

// Hit is one search result of a fixture, in the engine's ranking order.
type Hit struct {
	Collection string  `json:"collection"`
	Relative   string  `json:"relative"`
	Line       int     `json:"line"`
	Score      float64 `json:"score"`
	DocID      string  `json:"docid"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet"`
}

// Fixture is the engine's whole state as a world declares it.
type Fixture struct {
	Collections map[string][]string `json:"collections"`  // qmd ls <c>
	Pending     int                 `json:"pending"`      // qmd status
	StatusError string              `json:"status_error"` // not empty: qmd status exits 1 with this stderr
	SearchError string              `json:"search_error"` // not empty: every search exits 1 with this stderr, MCP query is a JSON-RPC error
	Hits        []Hit               `json:"hits"`         // search|vsearch|query --json and MCP query
}

// Load reads a fixture. A world without one has an engine that knows nothing.
func Load(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Fixture{}, nil
	}
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	// A misspelt key would leave the engine silently empty, and a recording
	// would pin that emptiness as the reference's answer.
	decoder.DisallowUnknownFields()
	fixture := &Fixture{}
	if err := decoder.Decode(fixture); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return fixture, nil
}

// RunCLI answers the qmd command lines both CLI ports run and refuses every
// other one with exit code 2.
func (f *Fixture) RunCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fakeqmd: subcommand required")
		return 2
	}
	switch args[0] {
	case "ls":
		return f.list(args[1:], stdout, stderr)
	case "status":
		return f.status(args[1:], stdout, stderr)
	case "search", "vsearch", "query":
		return f.search(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "fakeqmd: unknown subcommand %q\n", args[0])
	return 2
}

// list prints a collection as qmd ls does. Both readers take only the qmd://
// location from a row, so size and date are fixed.
func (f *Fixture) list(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "fakeqmd: ls takes exactly one collection")
		return 2
	}
	relatives, ok := f.Collections[args[0]]
	if !ok {
		// qmd 2.8.3, measured 2026-09-15: exit 1, nothing on stdout, two lines on stderr.
		fmt.Fprintf(stderr, "Collection not found: %s\nRun 'qmd ls' to see available collections.\n", args[0])
		return 1
	}
	if len(relatives) == 0 {
		// qmd 2.8.3 for a collection without documents (dist/cli/qmd.js:1491).
		fmt.Fprintf(stdout, "No files found in collection: %s\n", args[0])
		return 0
	}
	for _, relative := range relatives {
		fmt.Fprintf(stdout, " 1 B  Jan  1 00:00  qmd://%s/%s\n", args[0], relative)
	}
	return 0
}

// status prints the Documents block of qmd status, reduced to the line both
// readers look for; qmd leaves that line out when nothing is pending.
func (f *Fixture) status(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "fakeqmd: status takes no arguments")
		return 2
	}
	if f.StatusError != "" {
		fmt.Fprint(stderr, f.StatusError)
		return 1
	}
	fmt.Fprintln(stdout, "Documents")
	if f.Pending > 0 {
		fmt.Fprintf(stdout, "  Pending:  %d need embedding (run 'qmd embed')\n", f.Pending)
	}
	return 0
}

// cliHit is one row of qmd's --json output, keys in qmd's order.
type cliHit struct {
	DocID   string  `json:"docid"`
	Score   float64 `json:"score"`
	File    string  `json:"file"`
	Line    int     `json:"line"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet,omitempty"`
}

// search prints the hits of the named collections as qmd --json does: an
// array, indented by two, empty as [] and never as null.
func (f *Fixture) search(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fakeqmd: a query is required")
		return 2
	}
	limit, collections, ok := searchFlags(args[1:], stderr)
	if !ok {
		return 2
	}
	if f.SearchError != "" {
		fmt.Fprint(stderr, f.SearchError)
		return 1
	}
	rows := []cliHit{}
	for _, hit := range f.hitsIn(collections, limit) {
		rows = append(rows, cliHit{
			DocID:   hit.DocID,
			Score:   hit.Score,
			File:    "qmd://" + hit.Collection + "/" + hit.Relative,
			Line:    hit.Line,
			Title:   hit.Title,
			Snippet: hit.Snippet,
		})
	}
	data, _ := json.MarshalIndent(rows, "", "  ")
	fmt.Fprintf(stdout, "%s\n", data)
	return 0
}

// searchFlags reads the switches both CLI ports pass: --json, -n N and any
// number of -c COLLECTION. The fake answers in JSON only, so --json and -n are
// required.
func searchFlags(args []string, stderr io.Writer) (int, []string, bool) {
	limit, asJSON := 0, false
	var collections []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" {
			asJSON = true
			continue
		}
		if (arg != "-n" && arg != "-c") || i+1 == len(args) {
			fmt.Fprintf(stderr, "fakeqmd: unexpected argument %q\n", arg)
			return 0, nil, false
		}
		i++
		if arg == "-c" {
			collections = append(collections, args[i])
			continue
		}
		n, err := strconv.Atoi(args[i])
		if err != nil || n < 1 {
			fmt.Fprintf(stderr, "fakeqmd: -n needs a positive number, got %q\n", args[i])
			return 0, nil, false
		}
		limit = n
	}
	if !asJSON || limit == 0 {
		fmt.Fprintln(stderr, "fakeqmd: search needs --json and -n")
		return 0, nil, false
	}
	return limit, collections, true
}

// hitsIn keeps the fixture's order as the engine's ranking. No collection
// named means every collection, as qmd's MCP query answered in the spike.
func (f *Fixture) hitsIn(collections []string, limit int) []Hit {
	var found []Hit
	for _, hit := range f.Hits {
		if len(found) == limit {
			break
		}
		if len(collections) == 0 || slices.Contains(collections, hit.Collection) {
			found = append(found, hit)
		}
	}
	return found
}

// addLineNumbers numbers every line of a snippet from start on, as qmd 2.8.3
// does on the MCP path only (dist/store.js addLineNumbers, called at
// dist/mcp/server.js:301): split at \n, an empty snippet is one empty line.
func addLineNumbers(text string, start int) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strconv.Itoa(start+i) + ": " + line
	}
	return strings.Join(lines, "\n")
}

// rpcRequest is the part of a JSON-RPC call the handler reads.
type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params struct {
		Name      string `json:"name"`
		Arguments struct {
			Collections []string `json:"collections"`
			Limit       *int     `json:"limit"`
		} `json:"arguments"`
	} `json:"params"`
}

// mcpHit is one entry of the query tool's structuredContent.results.
type mcpHit struct {
	DocID   string  `json:"docid"`
	File    string  `json:"file"`
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	Line    int     `json:"line"`
	Snippet string  `json:"snippet"`
}

// MCPHandler answers the two calls loomux's MCP port makes, the initialize
// handshake and the query tool. Every other call gets a JSON-RPC error, so a
// port that asks for more fails loudly in the corpus instead of reading
// nothing.
func (f *Fixture) MCPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			writeJSON(w, rpcError(nil, -32600, "fakeqmd: only POST is served"))
			return
		}
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, rpcError(nil, -32700, "fakeqmd: "+err.Error()))
			return
		}
		switch {
		case req.Method == "initialize":
			writeJSON(w, rpcResult(req.ID, map[string]any{
				"protocolVersion": "2025-06-18",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "fakeqmd", "version": "0"},
			}))
		case req.Method == "tools/call" && req.Params.Name == "query" && f.SearchError != "":
			// The engine is reachable and fails the search itself.
			writeJSON(w, rpcError(req.ID, -32603, f.SearchError))
		case req.Method == "tools/call" && req.Params.Name == "query":
			limit := mcpDefaultLimit
			if req.Params.Arguments.Limit != nil {
				limit = *req.Params.Arguments.Limit
			}
			results := []mcpHit{}
			for _, hit := range f.hitsIn(req.Params.Arguments.Collections, limit) {
				results = append(results, mcpHit{
					DocID:   hit.DocID,
					File:    hit.Collection + "/" + hit.Relative,
					Title:   hit.Title,
					Score:   hit.Score,
					Line:    hit.Line,
					Snippet: addLineNumbers(hit.Snippet, hit.Line),
				})
			}
			writeJSON(w, rpcResult(req.ID, map[string]any{
				"structuredContent": map[string]any{"results": results},
			}))
		default:
			writeJSON(w, rpcError(req.ID, -32601, fmt.Sprintf("fakeqmd: no answer for %s %s", req.Method, req.Params.Name)))
		}
	})
}

func rpcResult(id json.RawMessage, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func rpcError(id json.RawMessage, code int, message string) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}}
}

// writeJSON ends a reply; a client that hung up has nobody left to tell.
func writeJSON(w io.Writer, v any) {
	_ = json.NewEncoder(w).Encode(v)
}

// Main is the fake qmd binary: it answers one command line from the fixture
// FixtureEnv names.
func Main(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	path := getenv(FixtureEnv)
	if path == "" {
		fmt.Fprintf(stderr, "fakeqmd: %s is not set\n", FixtureEnv)
		return 2
	}
	fixture, err := Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "fakeqmd: %v\n", err)
		return 2
	}
	return fixture.RunCLI(args, stdout, stderr)
}
```

`internal/dev/fakeqmd/qmd/main.go`:

```go
// Command qmd is the fake qmd the stage 1b-1 recordings put on PATH in front
// of the real one. It is built for recording only and never shipped.
package main

import (
	"os"

	"github.com/xidus90/loomux/internal/dev/fakeqmd"
)

//coverage:exempt process entry; every decision lives in fakeqmd.Main, which is tested
func main() {
	os.Exit(fakeqmd.Main(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}
```

Bewusst so entschieden: `Load` verweigert unbekannte Schlüssel (ein Tippfehler in einer Welt würde sonst eine leere Engine als Referenzantwort festschreiben); `search` verlangt `--json` und `-n`, weil beide Ports sie immer übergeben und der Fake keine Tabellenausgabe kennt; ohne `-c` bzw. mit leerem `collections` gelten alle Collections wie im Spike gemessen; ein kaputtes Fixture endet in `Main` mit Exit 2. Nach Ruling R5 baut der Fake qmd an zwei Stellen echter nach: Der MCP-Weg nummeriert jede Snippet-Zeile ab `hit.Line` wie `addLineNumbers`, der CLI-Weg `--json` nicht — so vergleicht der Korpus Python-CLI und loomux-MCP auf den Bytes, die echtes qmd liefert. `SearchError` lässt eine Suche scheitern, auf der CLI nach der Prüfung der Argumente mit Exit 1, auf MCP als JSON-RPC-Fehler `-32603` (Internal error), während `initialize` weiter antwortet; ohne diese Störung gäbe es für „Nie leer statt kaputt" keinen Fall.

- [ ] **Step 4: Bestehen sehen, Coverage**

```bash
go test ./internal/dev/fakeqmd/...
```
Erwartet: `ok  	github.com/xidus90/loomux/internal/dev/fakeqmd` und `github.com/xidus90/loomux/internal/dev/fakeqmd/qmd		[no test files]`.
```bash
go test ./internal/dev/fakeqmd/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Erwartet: Exit 0, dieselben zwei Paketzeilen, je mit einer `coverage:`-Angabe über alle Pakete des Moduls (die Zahl wird hier nicht bewertet).
```bash
go tool cover -func="$TEMP/pkg.out" | grep internal/dev/fakeqmd
```
Erwartet: jede Funktion des Pakets 100.0% — `Load`, `RunCLI`, `list`, `status`, `search`, `searchFlags`, `hitsIn`, `addLineNumbers`, `MCPHandler`, `rpcResult`, `rpcError`, `writeJSON`, `Main` (gemessen am 2026-09-15 im Wegwerfmodul); `qmd/main.go` `main` mit `0.0%` trägt die Ausnahme. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.

- [ ] **Step 5: Kette gegen die Python-Referenz prüfen** (von Hand, nicht im Tor; startet den Fake, nicht qmd)

```bash
go build -o "$TEMP/loomux-fakeqmd/qmd.exe" ./internal/dev/fakeqmd/_qmd
```
Erwartet: keine Ausgabe, Exit 0; das Verzeichnis `$TEMP/loomux-fakeqmd` entsteht dabei.
```bash
printf '%s\n' '{"collections":{"repo-a":["notes/a b.md","x.md"],"empty":[]},"pending":2}' > "$TEMP/loomux-fakeqmd/fixture.json"
```
Erwartet: keine Ausgabe.
```bash
PATH="$TEMP/loomux-fakeqmd:$PATH" LOOMUX_FAKE_QMD_FIXTURE="$(cygpath -m "$TEMP/loomux-fakeqmd/fixture.json")" PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "from brain.search.qmd import QmdPort; p = QmdPort(); print(p.indexed('repo-a')); print(p.indexed('empty')); print(p.not_yet_searchable())"
```
Erwartet, gemessen am 2026-09-15 (Pythons `launcher` findet `qmd.EXE` im vorangestellten Verzeichnis vor dem npm-Shim, auch hinter `uv run`, das `.venv\Scripts` voranstellt; `--no-sync` gleicht die `.venv` im Quell-Worktree nicht ab, `PYTHONDONTWRITEBYTECODE=1` schreibt dort kein `.pyc`):
```
('notes/a b.md', 'x.md')
()
2
```
Dann die scheiternde Suche, die Task 12 als Welt `engine-fails` aufzeichnet:
```bash
printf '%s\n' '{"collections":{"repo-a":["x.md"]},"search_error":"the index is locked\n"}' > "$TEMP/loomux-fakeqmd/fixture-fails.json"
```
Erwartet: keine Ausgabe.
```bash
printf '%s\n' 'from brain.search.port import Profile, SearchUnavailable' 'from brain.search.qmd import QmdPort' '' 'try:' '    QmdPort().search("q", ("repo-a",), Profile.FULL, 5)' 'except SearchUnavailable as error:' '    print(repr(str(error)))' > "$TEMP/loomux-fakeqmd/search-fails.py"
```
Erwartet: keine Ausgabe.
```bash
PATH="$TEMP/loomux-fakeqmd:$PATH" LOOMUX_FAKE_QMD_FIXTURE="$(cygpath -m "$TEMP/loomux-fakeqmd/fixture-fails.json")" PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python "$(cygpath -m "$TEMP/loomux-fakeqmd/search-fails.py")"
```
Erwartet, gemessen am 2026-09-15:
```
'qmd exited with 1: the index is locked'
```
Weicht eine der beiden Ausgaben ab, ist Task 12 blockiert; nicht weiter, sondern den Befund melden.

- [ ] **Step 6: Failing tests für die Programmform** — `internal/dev/recordcase/recordcase_test.go` an fünf Stellen ändern, der Rest bleibt wortgleich:

(a) Zuerst und allein ans Dateiende anhängen (Ruling R28), damit das Rot ein Laufzeitfehler ist und nicht im Übersetzungsfehler der übrigen Tests untergeht:
```go
// A command of blanks splits into no token at all, so there is no old name
// for the program to replace.
func TestRecordRefusesACommandWithoutTokens(t *testing.T) {
	s := helperSpec(t, "echo", "")
	s.Cmd = "   "
	if err := Record(s); err == nil || err.Error() != "empty command" {
		t.Fatalf("err %v", err)
	}
}
```
```bash
go test ./internal/dev/recordcase/ -count=1 -run TestRecordRefusesACommandWithoutTokens
```
Erwartet: Exit 1, `--- FAIL: TestRecordRefusesACommandWithoutTokens` und ein Stacktrace, der durch `recordcase.Record` läuft.
```bash
go test ./internal/dev/recordcase/ -count=1 -run TestRecordRefusesACommandWithoutTokens 2>&1 | grep -cE '^panic: runtime error: slice bounds out of range \[1:0\]'
```
Erwartet: `1` (gemessen am 2026-09-15: `Record` schneidet `tokens[1:]` aus null Tokens).

(b) Importblock: `"reflect"` zwischen `"path/filepath"` und `"strings"` einfügen.

(c) In `TestHelperProcess` vor `default:` einfügen:
```go
	case "env":
		for _, name := range args {
			value := os.Getenv(name)
			if name == "PATH" {
				value, _, _ = strings.Cut(value, string(os.PathListSeparator))
			}
			os.Stdout.WriteString(name + "=" + value + "\n")
		}
		os.Exit(0)
	case "crlf":
		os.Stdout.WriteString("one\r\ntwo\r\n")
		os.Exit(0)
```

(d) Direkt hinter `helperSpec` einfügen:
```go
// helperArgvSpec runs the test binary as a program with leading arguments:
// the program, -test.run and "--" stand before the command's own arguments,
// as "uv run --project <ub> brain-mcp" stands before "status".
func helperArgvSpec(t *testing.T, mode, cmd string) Spec {
	t.Helper()
	s := helperSpec(t, mode, cmd)
	s.Exe = ""
	s.Argv = []string{os.Args[0], "-test.run=TestHelperProcess", "--"}
	s.Cmd = "brain-mcp " + cmd
	return s
}
```

(e) Ans Dateiende, hinter den Test aus (a), anhängen:
```go
// The Python reference runs as "uv run --project <ub> brain-mcp", reads its
// state directory and a fake qmd from the environment, and finds that qmd
// through PATH. All three have to reach the recorded process.
func TestRecordRunsAProgramWithLeadingArgumentsInItsEnvironment(t *testing.T) {
	prepend := t.TempDir()
	s := helperArgvSpec(t, "env", "BRAIN_STATE_DIR PYTHONUTF8 LOOMUX_FAKE_QMD_FIXTURE PATH")
	s.Env = []string{"LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json"}
	s.PathPrepend = prepend
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	want := "BRAIN_STATE_DIR={{WORLD}}\nPYTHONUTF8=1\nLOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json\nPATH=" + prepend + "\n"
	if got := read(t, s.Out, "stdout"); got != want {
		t.Errorf("stdout %q, want %q", got, want)
	}
	if got := read(t, s.Out, "cmd"); got != "brain-mcp BRAIN_STATE_DIR PYTHONUTF8 LOOMUX_FAKE_QMD_FIXTURE PATH\n" {
		t.Errorf("cmd %q", got)
	}
}

// Python on Windows ends every printed line with \r\n in a pipe.
func TestRecordFoldsCRLFInStdout(t *testing.T) {
	s := helperSpec(t, "crlf", "")
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s.Out, "stdout"); got != "one\ntwo\n" {
		t.Errorf("stdout %q", got)
	}
}

func TestRecordRefusesAProgramNamedTwiceOrNotAtAll(t *testing.T) {
	s := helperArgvSpec(t, "echo", "x")
	s.Exe = os.Args[0]
	if err := Record(s); err == nil {
		t.Error("Exe and Argv: want error")
	}
	s.Exe, s.Argv = "", nil
	if err := Record(s); err == nil {
		t.Error("neither Exe nor Argv: want error")
	}
}

func TestRecordRefusesAnEnvironmentEntryWithoutAValue(t *testing.T) {
	s := helperSpec(t, "echo", "x")
	s.Env = []string{"NOVALUE"}
	if err := Record(s); err == nil || !strings.Contains(err.Error(), `"NOVALUE"`) {
		t.Fatalf("err %v", err)
	}
}

func TestRecordNamesTheArgvProgramItCannotRun(t *testing.T) {
	s := helperArgvSpec(t, "echo", "x")
	s.Argv = []string{filepath.Join(t.TempDir(), "no-such-program.exe"), "run"}
	if err := Record(s); err == nil || !strings.Contains(err.Error(), "no-such-program.exe") {
		t.Fatalf("err %v", err)
	}
}

func TestMergeEnvReplacesAKeyAsThePlatformSpellsIt(t *testing.T) {
	base := []string{`Path=C:\a`, "X=1"}
	for _, c := range []struct {
		goos string
		env  []string
		set  []string
		want []string
	}{
		{"windows", base, []string{`PATH=C:\b`}, []string{"X=1", `PATH=C:\b`}},
		{"linux", []string{"Path=/a", "X=1"}, []string{"PATH=/b"}, []string{"Path=/a", "X=1", "PATH=/b"}},
		{"linux", []string{"X=1", "Y=2"}, []string{"X=3", "Z=4"}, []string{"Y=2", "X=3", "Z=4"}},
	} {
		if got := mergeEnv(c.goos, c.env, c.set...); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %v + %v: got %v, want %v", c.goos, c.env, c.set, got, c.want)
		}
	}
	if !reflect.DeepEqual(base, []string{`Path=C:\a`, "X=1"}) {
		t.Errorf("the environment it was given changed: %v", base)
	}
}

func TestPrependPathPutsTheDirectoryFirst(t *testing.T) {
	for _, c := range []struct {
		goos string
		env  []string
		dir  string
		want []string
	}{
		{"windows", []string{`Path=C:\a;C:\b`, "X=1"}, `D:\fake`, []string{"X=1", `PATH=D:\fake;C:\a;C:\b`}},
		{"linux", []string{"PATH=/a:/b"}, "/fake", []string{"PATH=/fake:/a:/b"}},
		{"linux", []string{"Path=/a"}, "/fake", []string{"Path=/a", "PATH=/fake"}},
		{"linux", []string{"PATH="}, "/fake", []string{"PATH=/fake"}},
		{"windows", []string{"X=1"}, `D:\fake`, []string{"X=1", `PATH=D:\fake`}},
	} {
		if got := prependPath(c.goos, c.env, c.dir); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %v: got %v, want %v", c.goos, c.env, got, c.want)
		}
	}
}
```

- [ ] **Step 7: Scheitern sehen**

```bash
go test ./internal/dev/recordcase/
```
Erwartet: Exit 1, FAIL beim Übersetzen; die letzten Zeilen sind `FAIL	github.com/xidus90/loomux/internal/dev/recordcase [build failed]` und `FAIL`.
```bash
go test ./internal/dev/recordcase/ 2>&1 | grep -cE 'recordcase_test\.go:[0-9]+:[0-9]+: (s\.(Argv|Env|PathPrepend) undefined \(type Spec has no field or method (Argv|Env|PathPrepend)\)|undefined: (mergeEnv|prependPath))$'
```
Erwartet: `8` (gemessen am 2026-09-15: dreimal `s.Argv`, zweimal `s.Env`, einmal `s.PathPrepend`, je einmal `mergeEnv` und `prependPath`). Zeilennummern variieren mit dem Kommentar.

- [ ] **Step 8: Programmform implementieren** — `internal/dev/recordcase/recordcase.go`, acht Stellen; `writeWorldAfter`, `copyTree`, `mkdirTemp` und alle Kommentare außerhalb dieser Stellen bleiben wortgleich.

(a) Importblock: `"runtime"` zwischen `"path/filepath"` und `"strings"` einfügen.

(b) `Spec` — nachher (gofmt richtet die alten Felder neu aus, ihre Kommentare bleiben):
```go
// Spec is one recording: which old binary, which command, which world.
type Spec struct {
	Exe         string   // absolute path of the old binary; replaces the command's first token
	Cmd         string   // command line with {{WORLD}}, first token the old name ("ulguard", "brain", "ulinit")
	World       string   // directory to stage
	Stdin       string   // file with the payload, may contain {{WORLD}}; "" for none
	Out         string   // case directory to create
	Notes       string   // text for notes.md: tag, binary, what the case shows
	Compare     string   // "" (data) or "message"
	Argv        []string // program and leading arguments; replaces the command's first token; excludes Exe
	Env         []string // KEY=VALUE for the recorded process; {{WORLD}} stands for the staged world
	PathPrepend string   // directory put in front of the recorded process's PATH
}
```

(c) `Record`, erste Zeilen — vorher:
```go
func Record(s Spec) error {
	tmp, err := mkdirTemp("", "record-*")
```
nachher:
```go
func Record(s Spec) error {
	if (s.Exe == "") == (len(s.Argv) == 0) {
		return errors.New("a recording names its program by exactly one of Exe and Argv")
	}
	for _, entry := range s.Env {
		if !strings.Contains(entry, "=") {
			return fmt.Errorf("environment entry %q is not KEY=VALUE", entry)
		}
	}
	tmp, err := mkdirTemp("", "record-*")
```

(d) `Record`, Tokens — vorher:
```go
	tokens, err := cases.SplitCommand(s.Cmd)
	if err != nil {
		return err
	}
```
nachher (`errors` ist schon importiert):
```go
	tokens, err := cases.SplitCommand(s.Cmd)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return errors.New("empty command")
	}
```

(e) `Record`, Prozessstart — vorher:
```go
	cmd := exec.Command(s.Exe, tokens[1:]...)
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "BRAIN_STATE_DIR="+tmp)
```
nachher:
```go
	program, args := s.Exe, tokens[1:]
	if len(s.Argv) > 0 {
		program, args = s.Argv[0], append(append([]string{}, s.Argv[1:]...), tokens[1:]...)
	}
	cmd := exec.Command(program, args...)
	cmd.Dir = tmp
	cmd.Env = recordEnv(runtime.GOOS, os.Environ(), s, tmp)
```

(f) `Record`, Startfehler — vorher `return fmt.Errorf("running %s: %w", s.Exe, err)`, nachher `return fmt.Errorf("running %s: %w", program, err)`.

(g) `Record`, stdout — vorher:
```go
	files := map[string][]byte{
		"cmd":      []byte(s.Cmd + "\n"),
		"exit":     []byte(fmt.Sprintf("%d\n", exit)),
		"stdout":   cases.Normalize(stdout.Bytes(), tmp),
```
nachher:
```go
	// Python writes \r\n into a pipe on Windows and loomux writes \n: the
	// recording keeps the lines, not the platform's line ends.
	recorded := bytes.ReplaceAll(stdout.Bytes(), []byte("\r\n"), []byte("\n"))
	files := map[string][]byte{
		"cmd":      []byte(s.Cmd + "\n"),
		"exit":     []byte(fmt.Sprintf("%d\n", exit)),
		"stdout":   cases.Normalize(recorded, tmp),
```

(h) Zwischen `Record` und `writeWorldAfter` einfügen:
```go
// recordEnv is the environment of a recorded process: the recorder's own, the
// state directory the old tools read, UTF-8 for Python's pipes, the entries
// the spec names, and the spec's directory in front of PATH.
func recordEnv(goos string, base []string, s Spec, tmp string) []string {
	world := filepath.ToSlash(tmp)
	env := mergeEnv(goos, base, "BRAIN_STATE_DIR="+tmp, "PYTHONUTF8=1")
	for _, entry := range s.Env {
		env = mergeEnv(goos, env, strings.ReplaceAll(entry, cases.WorldToken, world))
	}
	if s.PathPrepend != "" {
		env = prependPath(goos, env, s.PathPrepend)
	}
	return env
}

// mergeEnv sets each KEY=VALUE of set in env and drops every earlier entry of
// that key. On Windows, Path and PATH name one variable; exec's own
// deduplication is not relied on to know that.
func mergeEnv(goos string, env []string, set ...string) []string {
	merged := append([]string{}, env...)
	for _, entry := range set {
		key, _, _ := strings.Cut(entry, "=")
		kept := merged[:0]
		for _, old := range merged {
			if oldKey, _, _ := strings.Cut(old, "="); !sameKey(goos, oldKey, key) {
				kept = append(kept, old)
			}
		}
		merged = append(kept, entry)
	}
	return merged
}

// prependPath puts dir in front of env's PATH, or makes it the whole PATH.
func prependPath(goos string, env []string, dir string) []string {
	separator := ":"
	if goos == "windows" {
		separator = ";"
	}
	value := dir
	for _, entry := range env {
		if key, old, _ := strings.Cut(entry, "="); sameKey(goos, key, "PATH") && old != "" {
			value = dir + separator + old
		}
	}
	return mergeEnv(goos, env, "PATH="+value)
}

// sameKey compares two environment keys as goos does.
func sameKey(goos, a, b string) bool {
	if goos == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
```

Die Faltung `\r\n` → `\n` und `PYTHONUTF8=1` gelten für jede Aufzeichnung, auch die Form mit `--exe`; die vorhandenen 1a-Aufzeichnungen werden nicht neu aufgenommen und bleiben Byte für Byte, wie sie sind.

- [ ] **Step 9: Bestehen sehen, Coverage**

```bash
go test ./internal/dev/recordcase/
```
Erwartet: `ok`, auch `TestRecordRefusesACommandWithoutTokens` aus Step 6 (a).
```bash
go test ./internal/dev/recordcase/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Erwartet: Exit 0, eine `ok`-Zeile mit einer `coverage:`-Angabe über alle Pakete des Moduls (die Zahl wird hier nicht bewertet).
```bash
go tool cover -func="$TEMP/pkg.out" | grep internal/dev/recordcase
```
Erwartet: jede Funktion des Pakets 100.0% — `Record`, `recordEnv`, `mergeEnv`, `prependPath`, `sameKey` — außer `writeWorldAfter` (85.7%) und `copyTree` (84.2%), beide mit ihrer unveränderten Ausnahme (gemessen am 2026-09-15 im Wegwerfmodul). Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.

- [ ] **Step 10: Failing tests für die Faltung** — ans Ende von `internal/dev/importcases/importcases_test.go` anhängen (keine neuen Importe):

```go
// writeFile writes content to path, making its directory first.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const worldRegistry = `[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"

[[area]]
scope = "notes"
path = "{{WORLD}}/areas/notes"
readonly = true

[[area]]
scope = "project/away"
path = "{{WORLD}}/../escape"

[[area]]
scope = "project/elsewhere"
path = "C:/elsewhere/repo-b"

[[area]]
scope = "project/gone"
path = "{{WORLD}}/repo-gone"
`

// A writable area of a 1b-1 world lives where the registry's path puts it,
// not under areas/. Its old manifest has to become loomux's configuration
// where it lives, or loomux reads a manifest the reference never saw.
func TestTranslateWorldFoldsEveryAreaTheRegistryPlacesInTheWorld(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "world")
	writeFile(t, filepath.Join(dir, "registry.toml"), worldRegistry)
	writeFile(t, filepath.Join(dir, "repo-a", ".ultra-brain", "config.toml"),
		"[area]\nscope = \"project/a\"\n\n[privacy]\nmode = \"local_only\"\nnever = [\"secret/**\"]\n")
	writeFile(t, filepath.Join(dir, "areas", "notes", ".brain.toml"), "[area]\nscope = \"notes\"\n")
	writeFile(t, filepath.Join(parent, "escape", ".brain.toml"), "[area]\nscope = \"project/away\"\n")

	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}

	privacy, _ := decodeConfig(t, filepath.Join(dir, "repo-a"))["privacy"].(map[string]any)
	never, _ := privacy["never"].([]any)
	if privacy["mode"] != "local_only" || len(never) != 1 || never[0] != "secret/**" {
		t.Errorf("privacy of repo-a %v", privacy)
	}
	for _, area := range []string{"repo-a", filepath.Join("areas", "notes")} {
		if got := entries(t, filepath.Join(dir, area)); strings.Join(got, " ") != ".loomux" {
			t.Errorf("%s: leftovers %v", area, got)
		}
	}
	if _, err := os.Stat(filepath.Join(parent, "escape", ".brain.toml")); err != nil {
		t.Errorf("a path outside the world was translated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "repo-gone")); !os.IsNotExist(err) {
		t.Errorf("a registered directory that is not there was made: %v", err)
	}
}

// A registry that does not read is a case of its own: the replay has to meet
// it as the reference did. The import folds the rest of the world and leaves
// the registry as it was recorded.
func TestTranslateWorldLeavesAnUnreadableRegistryToTheReplay(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "registry.toml"), "[[area\n")
	writeFile(t, filepath.Join(dir, "areas", "notes", ".brain.toml"), "[area]\nscope = \"notes\"\n")
	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "registry.toml")); err != nil || string(got) != "[[area\n" {
		t.Errorf("registry %q, err %v", got, err)
	}
	if got := entries(t, filepath.Join(dir, "areas", "notes")); strings.Join(got, " ") != ".loomux" {
		t.Errorf("the area beside the registry was not folded: %v", got)
	}
}

func TestTranslateWorldReportsARegisteredAreaItCannotTranslate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "registry.toml"), "[[area]]\nscope = \"project/a\"\npath = \"{{WORLD}}/repo-a\"\n")
	writeFile(t, filepath.Join(dir, "repo-a", ".brain.toml"), "[area\n")
	if err := TranslateWorld(dir); err == nil || !strings.Contains(err.Error(), "repo-a") {
		t.Fatalf("want an error naming the area's manifest, got %v", err)
	}
}
```

- [ ] **Step 11: Scheitern sehen**

```bash
go test ./internal/dev/importcases/ -count=1 -run 'TestTranslateWorldFoldsEvery|TestTranslateWorldLeavesAnUnreadable|TestTranslateWorldReportsARegistered'
```
Erwartet: Exit 1, letzte Zeile `FAIL`.
```bash
go test ./internal/dev/importcases/ -count=1 -run 'TestTranslateWorldFoldsEvery|TestTranslateWorldLeavesAnUnreadable|TestTranslateWorldReportsARegistered' 2>&1 | grep -cE '^--- FAIL: TestTranslateWorld(FoldsEveryAreaTheRegistryPlacesInTheWorld|ReportsARegisteredAreaItCannotTranslate) '
```
Erwartet: `2`.
```bash
go test ./internal/dev/importcases/ -count=1 -run 'TestTranslateWorldFoldsEvery|TestTranslateWorldLeavesAnUnreadable|TestTranslateWorldReportsARegistered' 2>&1 | grep -cE '^--- FAIL: '
```
Erwartet: `2` — `TestTranslateWorldLeavesAnUnreadableRegistryToTheReplay` ist schon gegen den heutigen Code grün, weil `TranslateWorld` heute keine Registry liest; der Test hält fest, dass Step 12 daran nichts ändert.
```bash
go test ./internal/dev/importcases/ -count=1 -run 'TestTranslateWorldFoldsEvery|TestTranslateWorldReportsARegistered' 2>&1 | grep -cE 'importcases_test\.go:[0-9]+: (decoding the translated config in .*[\\/]world[\\/]repo-a: open |want an error naming the area.s manifest, got <nil>$)'
```
Erwartet: `2` (gemessen am 2026-09-15; hinter `open ` folgen der Pfad der fehlenden `config.toml` und der Betriebssystemtext in der Sprache des Systems). Zeilennummern variieren mit dem Kommentar.

- [ ] **Step 12: Faltung implementieren** — `internal/dev/importcases/importcases.go`, drei Stellen; `Import`, `prune`, `rewriteCommand`, `rewritePaths`, `translateDir` und alles Weitere bleiben wortgleich.

(a) Importblock — nachher (neu nur `config`; `io/fs` braucht `copyTree` weiter):
```go
import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/config"
)
```
In `translateDir` verschattet die lokale Variable `config` (`config, _, err := decode(filepath.Join(dir, ".ultraloom", "config.toml"))`) ab dort das Paket; sie bleibt wortgleich, `go vet` meldet nichts (gemessen).

(b) `TranslateWorld` ganz ersetzen — vorher:
```go
// TranslateWorld rewrites the configuration files of the old tools in dir
// (and in every dir/areas/<name>) into one .loomux/config.toml.
func TranslateWorld(dir string) error {
	if err := translateDir(dir); err != nil {
		return err
	}
	areas, err := os.ReadDir(filepath.Join(dir, "areas"))
	if err != nil {
		return nil
	}
	for _, area := range areas {
		if !area.IsDir() {
			continue
		}
		if err := translateDir(filepath.Join(dir, "areas", area.Name())); err != nil {
			return err
		}
	}
	return nil
}
```
nachher (das frühe `return nil` ohne `areas/` fiele sonst auch über die Registry-Verzeichnisse; `os.ReadDir` gibt bei einem Fehler keine Einträge zurück, die Schleife läuft dann leer wie vorher):
```go
// TranslateWorld rewrites the configuration files of the old tools in dir, in
// every dir/areas/<name> and in every directory dir/registry.toml names as
// {{WORLD}}/<path>, into one .loomux/config.toml each.
func TranslateWorld(dir string) error {
	if err := translateDir(dir); err != nil {
		return err
	}
	// A world without areas/ has no read-only area to fold.
	areas, _ := os.ReadDir(filepath.Join(dir, "areas"))
	for _, area := range areas {
		if !area.IsDir() {
			continue
		}
		if err := translateDir(filepath.Join(dir, "areas", area.Name())); err != nil {
			return err
		}
	}
	// A registered directory that is also the root or an areas/ entry was
	// folded above; translateDir finds no old file there and writes nothing.
	for _, area := range registeredDirs(dir) {
		if err := translateDir(area); err != nil {
			return err
		}
	}
	return nil
}
```

(c) Direkt hinter `TranslateWorld` einfügen:
```go
// registeredDirs names the directories the world's registry places inside the
// world. A writable area lives where its path says, not only under areas/, and
// its manifest is read where it lives. A path outside the world is no part of
// the recording.
func registeredDirs(dir string) []string {
	areas, err := config.ReadRegistry(dir)
	if err != nil {
		// A missing or unreadable registry names nothing to fold. The replay
		// reads the same file and reports it, as the recording did.
		return nil
	}
	var dirs []string
	for _, area := range areas {
		rest, ok := strings.CutPrefix(area.Path, cases.WorldToken+"/")
		if !ok || !filepath.IsLocal(filepath.FromSlash(rest)) {
			continue
		}
		dirs = append(dirs, filepath.Join(dir, filepath.FromSlash(rest)))
	}
	return dirs
}
```
`config.ReadRegistry` liest dieselbe Datei, die die Suite von Task 12 als `LOOMUX_STATE_DIR` liest; ein Eintrag ohne `scope` fällt dort wie hier weg. Nach Ruling R6 ist jeder Lesefehler der Registry, auch eine fehlende Datei, hier kein Fehler: So lässt sich eine Welt mit kaputter Registry importieren, und der Fall `brain-catalog/broken-registry` (Task 12) zeigt die Meldung der Wiedergabe. Weil `registeredDirs` nie einen Fehler meldet, hat es keine Fehlerrückgabe; ein nie gesetzter `error` ließe den Zweig `if err != nil { return err }` in `TranslateWorld` ungedeckt.

- [ ] **Step 13: Bestehen sehen, Coverage, 1a unverändert**

```bash
go test ./internal/dev/importcases/
```
Erwartet: `ok`.
```bash
go test ./internal/dev/importcases/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
```
Erwartet: Exit 0, eine `ok`-Zeile mit einer `coverage:`-Angabe über alle Pakete des Moduls (die Zahl wird hier nicht bewertet).
```bash
go tool cover -func="$TEMP/pkg.out" | grep internal/dev/importcases
```
Erwartet: jede Funktion des Pakets 100.0%, `TranslateWorld` und `registeredDirs` eingeschlossen, außer `copyTree` (82.4%) mit seiner unveränderten Ausnahme (gemessen am 2026-09-15 im Wegwerfmodul). Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.

Die 1a-Welten nennen in `registry.toml` nur `{{WORLD}}` ohne Schrägstrich oder relative Pfade (`vault/hub`, `sources`); die neue Faltung findet dort nichts. Beleg durch Neuimport:
```bash
go run ./cmd/loomux dev import-cases --map testdata/cases/1a-map.toml --from testdata/cases/1a-source --to "$TEMP/reimport-1a"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
diff -r "$TEMP/reimport-1a" testdata/cases/1a
```
Erwartet: Exit 1 und genau ein Unterschied, die in `testdata/cases/README.md` dokumentierte Handabweichung in `hook-pre-tool-use/unreadable-payload/exit` (links `1`, rechts `2`); gemessen am 2026-09-15 mit dem neuen Übersetzer im Wegwerfmodul, mit dem alten gleich. Die zwei folgenden Befehle prüfen das ohne den Pfad von `$TEMP`:
```bash
diff -r "$TEMP/reimport-1a" testdata/cases/1a | wc -l
```
Erwartet: `5`.
```bash
diff -r "$TEMP/reimport-1a" testdata/cases/1a | grep -cE '^(diff -r .*reimport-1a/hook-pre-tool-use/unreadable-payload/exit testdata/cases/1a/hook-pre-tool-use/unreadable-payload/exit|1c1|< 1|---|> 2)$'
```
Erwartet: `5`.

- [ ] **Step 14: Failing tests für `dev record-case`** — `internal/cli/dev_test.go`:

(a) In `TestDevRecordCaseNeedsItsFlags` die Zeile
```go
	if code != 2 || !strings.Contains(errOut, "loomux dev record-case: --exe, --cmd, --world and --out are required") {
```
ersetzen durch
```go
	if code != 2 || !strings.Contains(errOut, "loomux dev record-case: --exe or --argv, --cmd, --world and --out are required") {
```

(b) Hinter `TestDevRecordCaseReportsAFailedRecording` einfügen (keine neuen Importe):
```go
func TestDevRecordCaseRefusesExeAndArgvTogether(t *testing.T) {
	code, _, errOut := run("dev", "record-case", "--exe", "brain.exe", "--argv", "uv run brain-mcp",
		"--cmd", "brain-mcp status", "--world", t.TempDir(), "--out", t.TempDir())
	if code != 2 || !strings.Contains(errOut, "loomux dev record-case: --exe and --argv exclude each other") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevRecordCaseRefusesAnArgvItCannotSplit(t *testing.T) {
	code, _, errOut := run("dev", "record-case", "--argv", "'unclosed",
		"--cmd", "brain-mcp status", "--world", t.TempDir(), "--out", t.TempDir())
	if code != 2 || !strings.Contains(errOut, "loomux dev record-case: --argv: unclosed quote") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevRecordCaseRefusesAnEnvWithoutAValue(t *testing.T) {
	code, _, errOut := run("dev", "record-case", "--env", "NOVALUE")
	if code != 2 || !strings.Contains(errOut, `"NOVALUE" is not KEY=VALUE`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// The argv form with a real program: go stands in for uv, "env" for the
// leading arguments, GOWORK for what the command asks after.
func TestDevRecordCaseRecordsAProgramWithLeadingArguments(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go binary on PATH")
	}
	out := filepath.Join(t.TempDir(), "demo", "gowork")
	code, _, errOut := run("dev", "record-case",
		"--argv", "go env", "--env", "GOWORK=off", "--env", "LOOMUX_UNUSED={{WORLD}}",
		"--path-prepend", t.TempDir(), "--cmd", "old GOWORK", "--world", t.TempDir(),
		"--out", out, "--notes", "the go workspace setting")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if got, err := os.ReadFile(filepath.Join(out, "stdout")); err != nil || string(got) != "off\n" {
		t.Fatalf("%v %q", err, got)
	}
}
```

- [ ] **Step 15: Scheitern sehen**

```bash
go test ./internal/cli/ -count=1 -run TestDevRecordCase
```
Erwartet: Exit 1, letzte Zeile `FAIL`.
```bash
go test ./internal/cli/ -count=1 -run TestDevRecordCase 2>&1 | grep -cE '^--- FAIL: TestDevRecordCase(NeedsItsFlags|RefusesExeAndArgvTogether|RefusesAnArgvItCannotSplit|RefusesAnEnvWithoutAValue|RecordsAProgramWithLeadingArguments) '
```
Erwartet: `5`.
```bash
go test ./internal/cli/ -count=1 -run TestDevRecordCase 2>&1 | grep -cE 'dev_test\.go:[0-9]+: code 2(, err "|: )(loomux dev record-case: --exe, --cmd, --world and --out are required|flag provided but not defined: -(argv|env))'
```
Erwartet: `5` (gemessen am 2026-09-15 gegen den heutigen Code: die alte Pflichtmeldung einmal, `-argv` dreimal, `-env` einmal). Zeilennummern variieren mit dem Kommentar.

- [ ] **Step 16: Flags implementieren** — `internal/cli/dev.go`, drei Stellen; alle anderen Funktionen bleiben wortgleich.

(a) Importblock: `"strings"` zwischen `"os/exec"` und `"time"` einfügen, `"github.com/xidus90/loomux/internal/cases"` als erste Zeile der loomux-Gruppe vor `"github.com/xidus90/loomux/internal/dev/benchhooks"`.

(b) Direkt vor `devRecordCase` einfügen:
```go
// envFlags collects a KEY=VALUE flag that may be given more than once.
type envFlags []string

func (e *envFlags) String() string { return strings.Join(*e, " ") }

func (e *envFlags) Set(value string) error {
	if !strings.Contains(value, "=") {
		return fmt.Errorf("%q is not KEY=VALUE", value)
	}
	*e = append(*e, value)
	return nil
}
```

(c) `devRecordCase` ganz ersetzen — nachher:
```go
func devRecordCase(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev record-case", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var s recordcase.Spec
	var argv string
	var env envFlags
	fs.StringVar(&s.Exe, "exe", "", "path of the old binary")
	fs.StringVar(&argv, "argv", "", "program and leading arguments in place of the command's first token")
	fs.Var(&env, "env", "KEY=VALUE for the recorded process, {{WORLD}} allowed; repeatable")
	fs.StringVar(&s.PathPrepend, "path-prepend", "", "directory put in front of the recorded process's PATH")
	fs.StringVar(&s.Cmd, "cmd", "", "command line with {{WORLD}}")
	fs.StringVar(&s.World, "world", "", "directory to stage")
	fs.StringVar(&s.Stdin, "stdin", "", "file with the payload")
	fs.StringVar(&s.Out, "out", "", "case directory to write")
	fs.StringVar(&s.Notes, "notes", "", "text for notes.md")
	fs.StringVar(&s.Compare, "compare", "", `"" (data) or "message"`)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if s.Exe != "" && argv != "" {
		fmt.Fprintln(stderr, "loomux dev record-case: --exe and --argv exclude each other")
		return 2
	}
	tokens, err := cases.SplitCommand(argv)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev record-case: --argv: %v\n", err)
		return 2
	}
	s.Argv, s.Env = tokens, env
	if (s.Exe == "" && len(s.Argv) == 0) || s.Cmd == "" || s.World == "" || s.Out == "" {
		fmt.Fprintln(stderr, "loomux dev record-case: --exe or --argv, --cmd, --world and --out are required")
		return 2
	}
	if err := recordcase.Record(s); err != nil {
		fmt.Fprintf(stderr, "loomux dev record-case: %v\n", err)
		return 1
	}
	return 0
}
```
Ein `--argv` aus lauter Leerzeichen zerfällt in null Tokens und landet bei der Pflichtmeldung. Ein `--cmd` aus lauter Leerzeichen besteht die Pflichtprüfung und endet in `Record` mit `loomux dev record-case: empty command`, Exit 1 (Step 8 (d)). `envFlags.String` ruft das `flag`-Paket beim Drucken der Hilfe auf (`--bogus` in `TestDevRecordCaseNeedsItsFlags`); gemessen deckt das die Funktion ab.

- [ ] **Step 17: Bestehen sehen**

```bash
go test ./internal/cli/ -run 'TestDev|TestRecordedCasesOfStage1a'
```
Erwartet: Exit 0, eine Zeile `ok  	github.com/xidus90/loomux/internal/cli` mit der Laufzeit dahinter — `TestRecordedCasesOfStage1a` bleibt grün.

- [ ] **Step 18: Gesamtlauf und Coverage-Tor über die Pakete des Tasks**

```bash
go test ./internal/cases/ ./internal/dev/... ./internal/cli/ -count=1 -covermode=set -coverprofile="$TEMP/task11.out"
```
Erwartet: Exit 0, jede Paketzeile `ok` (gemessen am 2026-09-15 im Wegwerfmodul: `cases` 98,6 %, `benchhooks` 84,9 %, `covergate` 100,0 %, `fakeqmd` 100,0 %, `importcases` 98,1 %, `recordcase` 96,3 %, `swap` 100,0 %, `cli` 100,0 % der Anweisungen; alles unter 100 % liegt in bestehenden Ausnahmen), `fakeqmd/qmd` mit `coverage: 0.0% of statements`.
```bash
go run ./cmd/loomux dev covergate --profile "$TEMP/task11.out"
```
Erwartet: keine Ausgabe, Exit 0 (gemessen).
```bash
gofmt -l internal/dev internal/cli
```
Erwartet: keine Ausgabe.
```bash
git status --short testdata/cases/1a-source testdata/cases/1a
```
Erwartet: keine Ausgabe.

- [ ] **Step 19: Stagen und Tor**

Das Tor bricht mit `pre-commit: inputs differ from the index` ab, solange eine `.go`-Datei ungestagt oder ungetrackt ist; darum zuerst stagen.
```bash
git add internal/dev/fakeqmd/fakeqmd.go internal/dev/fakeqmd/fakeqmd_test.go internal/dev/fakeqmd/qmd/main.go internal/dev/recordcase/recordcase.go internal/dev/recordcase/recordcase_test.go internal/dev/importcases/importcases.go internal/dev/importcases/importcases_test.go internal/cli/dev.go internal/cli/dev_test.go
```
Erwartet: keine Ausgabe.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0 (`covergate` ohne Befund, `bin/loomux.exe` neu gebaut).

- [ ] **Step 20: Commit**

```bash
git branch --show-current
```
Erwartet: `sdd-1b-1` (Task 0; im Hauptcheckout `master`).
```bash
git rev-parse --short HEAD
```
Erwartet: der kurze Hash des Commits von Task 10.
```bash
git diff --cached --stat
```
Erwartet: genau diese neun Dateien, keine weitere.
```bash
printf '%s\n' 'Record cases from a program against a fake qmd' '' 'record-case takes a program with leading arguments, extra environment' 'and a PATH prefix, sets PYTHONUTF8, folds CRLF in the recorded stdout' 'and refuses a command without tokens. fakeqmd answers qmd ls, status,' 'search and the MCP query tool from a per-world fixture, numbers the MCP' 'snippet lines as qmd does and can fail a search. TranslateWorld also' 'folds the manifests of every area the world registry places inside the' 'world and leaves an unreadable registry to the replay.' > "$TEMP/task11-commit.txt"
```
Erwartet: keine Ausgabe.
```bash
git commit -F "$TEMP/task11-commit.txt"
```
Erwartet: Exit 0 und die Zeile `[sdd-1b-1 <hash>] Record cases from a program against a fake qmd` mit `9 files changed`. Der Commit löst das Tor noch einmal aus; das ist gewollt.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität.
```bash
git log -1 --format=%B
```
Erwartet: genau der Text aus `$TEMP/task11-commit.txt`, keine `Co-Authored-By`-Zeile.

- [ ] **Step 21: Bericht** — nennt:
  - Paritätszeilen, die dieser Task schafft oder berichtigt (Task 12 trägt sie ein):
    1. Aufzeichnungen: `\r\n` im aufgezeichneten stdout wird zu `\n`, `PYTHONUTF8=1` ist in jedem aufgezeichneten Prozess gesetzt (Spec-Zeile 4).
    2. qmd in den Aufzeichnungen ist der Fake: `ls` gibt feste Größe und Zeit aus, `status` nur `Documents` und die `Pending`-Zeile, Treffer kommen synthetisch in Fixture-Reihenfolge. Eine scheiternde Suche ist die Störung `search_error` (CLI Exit 1 mit dem Wert auf stderr, MCP JSON-RPC-Fehler `-32603`), kein gemessener qmd-Wortlaut.
    3. Snippets auf beiden Wegen (berichtigt nach F69): Echtes qmd 2.8.3 unterscheidet sich zwischen MCP-`query` und CLI `--json` nur im Präfix `N: `, das `addLineNumbers` auf dem MCP-Weg jeder Snippet-Zeile voranstellt (`dist/mcp/server.js:301`, `dist/store.js:4292-4295`). Das Ausschnittfenster ist auf beiden Wegen dasselbe: MCP `server.js:293` und CLI `cli/qmd.js:2140` rufen `extractSnippet` mit 300 Zeichen, der Position und der Länge des besten Chunks; `cli/formatter.js:58` liegt nicht auf dem Weg von `--json`. Der Fake stellt das Präfix jetzt wie qmd auf dem MCP-Weg voran, Task 8 entfernt es in `translateReply` (R4), und `TestLoomuxsPortsReadTheFake` sowie der Korpus von Task 12 belegen die Entfernung auf den Bytes, die echtes qmd liefert. Der Live-Vergleich von Task 15 legt die Snippets einer `full`-Suche zur Bestätigung nebeneinander.
  - Abweichungen vom Vertrag: `Fixture.SearchError` und die nummerierten MCP-Snippets (Ruling R5; die MCP-Handler-Regel des Vertrags ist damit überholt). `TranslateWorld` übergeht eine fehlende oder unlesbare Registry, `registeredDirs` hat deshalb keine Fehlerrückgabe (Ruling R6); Task 12 kann damit die Welt `broken-registry` importieren.
  - Befunde außerhalb des Umzugs: `Record` geriet bei einem `Cmd` ohne Token in einen Laufzeitfehler (`tokens[1:]`, gemessen `panic: runtime error: slice bounds out of range [1:0]`); behoben test-first mit der Meldung `empty command` (Step 6 (a), Step 8 (d), Ruling R28).
  - Die Messbefehle aus Step 5 und Step 13 mit ihrer Ausgabe.

---

### Task 12: Der Fallkorpus der Stufe 1b-1

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 11 sind committet.

**Files:**
- Create: `internal/cli/cases_1b1_test.go`
- Create: `testdata/cases/1b-1-worlds/**` (23 Welten, 130 Dateien, Step 4)
- Create: `testdata/cases/1b-1-source/**` (71 Aufzeichnungen, Step 6)
- Create: `testdata/cases/1b-1-map.toml`, `testdata/cases/1b-1/**` (Step 8, Step 9)
- Create: `docs/.superpowers/parity/stufe-1b-1.md` (Step 13)
- Modify: `testdata/cases/README.md` (Step 14)
- Unberührt und am Ende geprüft: `testdata/cases/1a-source/**`, `testdata/cases/1a/**`, der Quell-Worktree `C:/Users/micro/Documents/#GIT/loomux-src/ub`

**Interfaces:**
- Consumes:
  - `internal/cases` (heute, `case.go`, `runner.go`): `func cases.DiscoverCases(root string, verbFilter string) ([]*cases.Case, error)`, `func cases.RunCase(c *cases.Case, run cases.RunFunc) (*cases.RunOutcome, error)`, `type cases.RunFunc func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int`; genutzte Felder `cases.Case.Verb`, `cases.Case.Name`, `cases.Case.Notes`, `cases.RunOutcome.Passed`, `cases.RunOutcome.ActualStdout`, `cases.RunOutcome.Mismatches`
  - `internal/cli` (heute): `func cli.Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int`, in der Suite unqualifiziert, weil sie im Paket `cli` liegt
  - Über die Umgebung, nicht gerufen: `LOOMUX_STATE_DIR` liest `config.StateDir()` (heute, `internal/config/registry.go`) für die Registry; `LOOMUX_LEGACY_BRAIN_DIR` liest `config.LegacyBrainDirUntilStage3()` (Task 1) für die Artefakte schreibgeschützter Bereiche und den Stempel; `config.ReadAreaManifestUntilStage4(dir string) (*config.Manifest, error)` (Task 1) liest `.loomux/config.toml` vor den Altnamen. Task 1 faltet nichts; die Altmanifeste faltet `importcases.TranslateWorld` (Task 11) beim Import.
  - Task 10 (Entwurf, `internal/cli/brain_test.go`): `func stubBrainSearchPort(t *testing.T, port search.SearchPort, notice string)`, `func stubBrainStatusPort(t *testing.T, port search.SearchPort)`. Beide setzen die Seams `brainSearchPort` bzw. `brainStatusPort` (Vertrag) und stellen sie über `t.Cleanup` zurück; die Suite ruft die Seams nicht selbst.
  - Task 11: `const fakeqmd.FixtureName = "qmd-fixture.json"`; `func fakeqmd.Load(path string) (*fakeqmd.Fixture, error)`, ohne Datei eine leere Fixture; `func (f *fakeqmd.Fixture) RunCLI(args []string, stdout, stderr io.Writer) int`; `func (f *fakeqmd.Fixture) MCPHandler() http.Handler`, das jede Snippet-Zeile wie qmds `addLineNumbers` mit `N: ` nummeriert (Ruling R5); die Fixture-Felder `collections`, `pending`, `status_error`, `search_error` (nicht leer: `search`, `vsearch` und `query` enden mit Exit 1 und diesem stderr, das MCP-`query` antwortet mit einem JSON-RPC-Fehler; Ruling R5) und `hits` mit `collection`, `relative`, `line`, `score`, `docid`, `title`, `snippet`. `loomux dev record-case` mit `--argv PROGRAMM`, `--env KEY=VALUE`, `--path-prepend DIR`. `loomux dev import-cases` mit `func importcases.TranslateWorld(dir string) error`: faltet die Altmanifeste in der Wurzel, in jedem `areas/<name>` und in jedem Verzeichnis, das `registry.toml` als `{{WORLD}}/<pfad>` nennt, in je eine `.loomux/config.toml`; eine Registry, die sich nicht lesen lässt, überlässt `registeredDirs` der Wiedergabe (Ruling R6).
  - Task 8 (ub-API, wortgleich umgezogen): `func search.NewQmdMcpPort(opts ...search.QmdMcpOption) *search.QmdMcpPort`; `func search.WithConnect(fn search.ConnectFunc) search.QmdMcpOption` mit `type search.ConnectFunc func(env map[string]string) (search.Session, error)`; `func search.WithCLI(cli search.SearchPort) search.QmdMcpOption`; `type search.Session interface{ Call(name string, arguments map[string]any) (map[string]any, error); Close() error }`; `search.HTTPSession` mit dem Feld `URL string`; `search.QmdPort` mit den Feldern `Executable string` und `Runner search.RunnerFunc`, `type search.RunnerFunc func(argv []string) ([]byte, []byte, int, error)`; das Interface `search.SearchPort`. Nicht gerufen, aber vorausgesetzt: `translateReply` entfernt das `N: `-Präfix der MCP-Snippets (Ruling R4), und `search.ColdAttempts` (3) begrenzt die Versuche von `ask`.
- Produces (Vertrag): `func TestRecordedCasesOfStage1b1(t *testing.T)`, Fallzahl festgenagelt auf **71** (Vertrag: 59; die Rulings R6, R7 und R8 fügen zwölf Fälle hinzu)
- Produces, Ergänzung (unexportiert, nur Test): `func useRecordedQmd(t *testing.T, dir string)`

**Abweichungen vom Vertrag:**
1. Der Vertrag nennt `NewHTTPSession` unter „Consumes". `NewHTTPSession(port int)` baut die Adresse `http://localhost:<port>/mcp` (ub `pkg/search/http.go:44-53`) und kann einen `httptest`-Server nicht erreichen; die Suite nimmt wie der Entwurf von Task 11 das Literal `&search.HTTPSession{URL: server.URL}`.
2. Die Aufzeichnungsvorlage des Vertrags bekommt in jedem Aufruf `uv run --no-sync` und `--env PYTHONDONTWRITEBYTECODE=1` (Step 6).
3. 71 statt 59 Fälle über 23 statt 19 Welten (Rulings R6, R7, R8).

Die Task-12-Zeilen des Vertrags (Consumes mit `HTTPSession.URL` statt `NewHTTPSession`, die Aufzeichnungsvorlage, die Fallzahl) zieht der Controller nach; dieser Task ändert den Vertrag nicht.

**Messgrundlage.** Jede Welt, jede Fallzeile und jede erwartete Ausgabe unten ist am 2026-09-15 an der Referenz gemessen: die 23 Welten im Sitzungs-Scratchpad gestagt (`{{WORLD}}` durch den Pfad der Kopie ersetzt, wie `cases.StageWorld` es tut), dann je Fall `brain.cli.main(argv)` im Prozess unter `uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python` mit `BRAIN_STATE_DIR` = gestagte Welt, `cli._port` → `QmdPort(runner=fake_runner)` mit einem Runner, der `qmd ls`, `qmd status` und `qmd query|vsearch|search --json -n N -c <collection>` so beantwortet wie `fakeqmd.RunCLI` aus dem Entwurf von Task 11 nach Ruling R5 (bei gesetztem `search_error` enden die drei Suchbefehle mit Exit 1 und diesem stderr) — eine Nachbildung in Python, nicht das gebaute `qmd.exe`; Step 1 und Step 7 sind die ersten Messungen gegen das Binary —, und `cli._through_daemon` → `None`. Nach den Rulings R5 bis R8 ist der ganze Korpus noch einmal so gemessen: die 118 Prüfsummen der ersten 59 Fälle sind unverändert, die zwölf neuen Fälle enden alle ohne stdout. Nur lesende Aufrufe; `brain-mcp` selbst lief nicht. Exit-Code und stdout jedes Falls stehen in Step 7 als Prüfsummen und als Text; die stdout-Pfade sind wie von `cases.Normalize` durch `{{WORLD}}` ersetzt. Die Suite ist in einem Wegwerf-Modul gegen das echte `internal/cases` und Stubs der Signaturen aus Task 8, 10 und 11 übersetzt: `go vet` sauber, `gofmt -l` leer.

**Warum diese Fälle.** Je Befehl der Erfolg, jede Verweigerung und jeder Fehlerweg aus dem Spec-Abschnitt „Fehlerverhalten", Registry- und Manifestfehler bei allen fünf Befehlen (fehlende Registry bei `catalog` und `search`, kaputte Registry und Manifest mit falschem Typ bei `catalog`, fehlendes Manifest bei `catalog`, `read` und `status`, Manifest ohne `[area] scope` bei `catalog` und `neighbors`), mindestens ein Usage-Fehler je Befehl, ein unbekannter Scope bei `search`, `catalog`, `read` und `neighbors`, ein kaputtes Register bei `search` und `status`, die scheiternde Suche, `status` mit einer Welt je Zeilenart L1a–L8b (Spec, `status`), `search --profile full` als Daten, `fast` und `keyword` als Meldung (Spec, Vergleichsklassen), der `cloud`-Kanal mit einem `local_only`-Bereich. Usage- und Laufzeitfehler vergleichen **Daten**: stdout ist auf beiden Seiten leer, und „kein stdout" ist Teil des Vertrags (`Exit-Codes loomux brain`); das gilt auch für `search q -n 0` mit der Vorgabe `fast`, weil der Aufruf den Suchweg nie erreicht. Die kaputte Registry (`registry.toml` = `[[area`) ist ein Fall, weil `registeredDirs` (Task 11) jeden Lesefehler der Registry der Wiedergabe überlässt (Ruling R6). Das unlesbare Manifest ist gültiges TOML vom falschen Typ (`[area]` mit `scope = 3`): `translateDir` dekodiert jedes Altmanifest beim Import und bräche an einem Manifest ab, das kein TOML ist. Die Referenz meldet den falschen Typ als `[area] scope is required and must be a non-empty string`, loomux als TOML-Typfehler, beide mit Exit 1. Die scheiternde Suche (Spec: nie leer statt kaputt) zeichnet die Welt `engine-fails` mit dem Fixture-Feld `search_error` auf (Ruling R7); ein Daemon, der sich nicht starten lässt, bleibt bei den Unit-Tests von Task 8, weil keine Aufzeichnung das echte qmd starten darf.

Die Befehle sind für Git Bash geschrieben.

- [ ] **Step 1: Werkzeuge bauen und die Kette gegen die Referenz prüfen**

```bash
go build -o bin/loomux.exe ./cmd/loomux
```
Expected: Exit 0, keine Ausgabe.
```bash
go build -o "$TEMP/loomux-fakeqmd/qmd.exe" ./internal/dev/fakeqmd/_qmd
```
Expected: Exit 0, keine Ausgabe.
```bash
bin/loomux.exe dev record-case --env NOVALUE
```
Expected: Exit 2 und auf stderr eine Zeile mit `"NOVALUE" is not KEY=VALUE` — das Binary kennt die Flags aus Task 11. Meldet es `flag provided but not defined: -env`, ist `bin/loomux.exe` älter als Task 11: Step 1 wiederholen, nicht weiter.
```bash
grep -c -E '^func (stubBrainSearchPort|stubBrainStatusPort)\(' internal/cli/brain_test.go
```
Expected: `2` (Task 10). Eine kleinere Zahl heißt: die Suite aus Step 2 ist nicht übersetzbar; den Befund melden.

Vorbedingung aus Task 11 Step 5, hier gegen den eben gebauten Fake wiederholt (Ruling R29): Pythons `QmdPort` findet `qmd.exe` im vorangestellten Verzeichnis und liest dessen Antworten. Die Kette startet den Fake, nie qmd, und schreibt weder `.venv` noch `.pyc` unter `ub`.
```bash
printf '%s\n' '{"collections":{"repo-a":["notes/a b.md","x.md"],"empty":[]},"pending":2}' > "$TEMP/loomux-fakeqmd/fixture.json"
```
Expected: Exit 0, keine Ausgabe.
```bash
PATH="$TEMP/loomux-fakeqmd:$PATH" LOOMUX_FAKE_QMD_FIXTURE="$(cygpath -m "$TEMP/loomux-fakeqmd/fixture.json")" PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "from brain.search.qmd import QmdPort; p = QmdPort(); print(p.indexed('repo-a')); print(p.indexed('empty')); print(p.not_yet_searchable())"
```
Expected, genau (Task 11 Step 5, gemessen am 2026-09-15):
```
('notes/a b.md', 'x.md')
()
2
```
Dazu die scheiternde Suche, auf der `brain-search/engine-fails` beruht. Ein Fake ohne das Feld `search_error` verweigert die Fixture wegen `DisallowUnknownFields` mit Exit 2; Python macht auch daraus Exit 1 ohne stdout, und die Prüfsumme in Step 7 bliebe dieselbe. Nur diese Prüfung trennt beide:
```bash
printf '%s\n' '{"collections":{"project-a":["a.md"]},"search_error":"index is locked\n"}' > "$TEMP/loomux-fakeqmd/failing.json"
```
Expected: Exit 0, keine Ausgabe.
```bash
PATH="$TEMP/loomux-fakeqmd:$PATH" LOOMUX_FAKE_QMD_FIXTURE="$(cygpath -m "$TEMP/loomux-fakeqmd/failing.json")" PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "from brain.search.port import Profile, SearchUnavailable
from brain.search.qmd import QmdPort
try:
    print(QmdPort().search('q', ('project-a',), Profile.FULL, 1))
except SearchUnavailable as error:
    print(error)"
```
Expected, genau (gemessen am 2026-09-15 mit einem Runner, der wie Ruling R5 antwortet; der Aufruf ist `qmd query q --json -n 1 -c project-a`):
```
qmd exited with 1: index is locked
```
Weicht eine der beiden Ausgaben ab, ist Task 11 nicht so gebaut, wie die Aufzeichnungen es voraussetzen (beginnt die zweite mit `qmd exited with 2:`, kennt der Fake `search_error` nicht): nicht weiter, den Befund an Task 11 melden.

- [ ] **Step 2: Failing test** — `internal/cli/cases_1b1_test.go`

```go
package cli

import (
	"bytes"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/fakeqmd"
)

// TestRecordedCasesOfStage1b1 replays the recordings of brain-mcp against
// loomux brain. Every world carries the fixture the recording's fake qmd
// answered from; the same fixture answers here, over HTTP for the MCP port of
// search and through the Runner seam for the command line port of status.
func TestRecordedCasesOfStage1b1(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "1b-1"), "")
	if err != nil {
		t.Fatal(err)
	}
	// The number is pinned, not merely non-zero: a partial import must not
	// pass as parity. Raise it with the corpus when a case is added.
	if len(all) != 71 {
		t.Fatalf("expected 71 recorded cases, found %d", len(all))
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Setenv("LOOMUX_STATE_DIR", dir)
				t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
				useRecordedQmd(t, dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			if !outcome.Passed {
				// The runner names only the byte counts of a stdout mismatch;
				// the text itself is what a red case is read by.
				t.Fatalf("%s\nstdout: %q\n%s", strings.Join(outcome.Mismatches, "\n"), outcome.ActualStdout, c.Notes)
			}
		})
	}
}

// useRecordedQmd points both brain seams at the fixture of the staged world
// until the case ends: search asks the fixture's MCP handler, status and the
// MCP port's own listings run the fixture's command line.
func useRecordedQmd(t *testing.T, dir string) {
	t.Helper()
	fixture, err := fakeqmd.Load(filepath.Join(dir, fakeqmd.FixtureName))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(fixture.MCPHandler())
	t.Cleanup(server.Close)
	runner := func(argv []string) ([]byte, []byte, int, error) {
		var out, errOut bytes.Buffer
		code := fixture.RunCLI(argv[1:], &out, &errOut)
		return out.Bytes(), errOut.Bytes(), code, nil
	}
	mcp := search.NewQmdMcpPort(
		search.WithConnect(func(map[string]string) (search.Session, error) {
			return &search.HTTPSession{URL: server.URL}, nil
		}),
		search.WithCLI(&search.QmdPort{Executable: "qmd", Runner: runner}),
	)
	stubBrainSearchPort(t, mcp, "")
	stubBrainStatusPort(t, &search.QmdPort{Executable: "qmd", Runner: runner})
}
```

- [ ] **Step 3: Scheitern sehen**

```bash
go test ./internal/cli/ -run TestRecordedCasesOfStage1b1 -count=1
```
Expected (gemessen im Wegwerf-Modul; der Systemtext folgt der Sprache von Windows):
```
--- FAIL: TestRecordedCasesOfStage1b1 (0.00s)
    cases_1b1_test.go:23: GetFileAttributesEx ..\..\testdata\cases\1b-1: Das System kann den angegebenen Pfad nicht finden.
FAIL
```

- [ ] **Step 4: Welten anlegen**

Jede Datei unten wird mit genau dem gezeigten Inhalt geschrieben und endet mit genau einem Zeilenumbruch hinter der letzten gezeigten Zeile. `{{WORLD}}` bleibt wörtlich stehen. In jeder `_identities.tsv` trennen **Tabulatoren** (U+0009) die Felder — beim Schreiben keine Leerzeichen daraus machen; Step 5 prüft es. In `status-unfindable/repo-a/_identities.tsv` ist `Café.md` mit dem vorkomponierten `é` (U+00E9, NFC) geschrieben; die Fixture derselben Welt nennt die zerlegte Form als JSON-Escape `Cafe\u0301.md`. `\n`, `\r` und `\u0301` in den Fixtures sind JSON-Escapes, also Backslash und Buchstabe in der Datei.

Aufbau einer Welt: die Welt ist das Zustandsverzeichnis der Referenz (`BRAIN_STATE_DIR`) und im Lauf von loomux zugleich `LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR`. `registry.toml` in der Wurzel; ein schreibbarer Bereich trägt Manifest, `_identities.tsv`, `graph.json` und `index.md` unter seinem eigenen Pfad `repo-*`; ein schreibgeschützter Bereich trägt sie unter `areas/<flacher Scope>/`, seine Dokumente unter dem Registry-Pfad; der Stempel liegt unter `maintenance/last-run.txt` (`2000-01-01T00:00:00+00:00` veraltet, `2999-01-01T00:00:00+00:00` frisch); `qmd-fixture.json` in der Wurzel. Die `status-*`-Welten tragen weder `index.md` noch Dokumente, weil `status` keines von beiden liest; ebenso `engine-fails`, weil die Suche dort scheitert, bevor ein Dokument zählt. L3 ist nur für einen schreibgeschützten Bereich erreichbar: ein schreibbarer ohne Pfad hat kein Manifest, und dann bricht jeder Befehl ab (Task 9).

| Welt | Bereiche (Registry-Reihenfolge) | Stempel | Fixture |
|---|---|---|---|
| `vault` | `project/beta` schreibbar, `.ultra-brain/config.toml`, `local_only`, zwei Include-Globs · `notes` schreibgeschützt · `project/alpha` schreibbar, `.brain.toml`, `never = ["secret/**"]`, `review = "review"` · `project/gamma` schreibbar, nur Manifest | 2000 | Listungen dreier Collections, fünf Treffer |
| `local-only` | `project/closed`, `local_only` | — | — |
| `no-registry` | keine Registry | — | — |
| `missing-manifest` | `project/bare` ohne Manifest | — | — |
| `manifest-without-scope` | `project/vague`, Manifest mit `[area]` und leerem `scope` | — | — |
| `damaged` | `project/torn` (`graph.json` ohne `links`) · `project/garbled` (Register mit drei Feldern) | — | — |
| `broken-registry` | Registry ist kein TOML (`[[area`) | — | — |
| `manifest-wrong-type` | `project/typed`, `[area] scope = 3` | — | — |
| `engine-fails` | `project/a` heil | — | `project-a: [a.md]`, `search_error` |
| `status-stamp-missing`, `-stale`, `-fresh` | `project/a` heil | —, 2000, 2999 | `project-a: [a.md]` |
| `status-include-globs` | `project/a` heil, drei Include-Globs | 2999 | wie oben |
| `status-path-missing` | `notes` schreibgeschützt, Pfad fehlt | 2999 | — |
| `status-never-indexed` | `project/a` ohne `graph.json` | 2999 | — |
| `status-links-unresolved` | `project/a` 2 von 5, `project/b` 1 von 3, `project/c` 2 von 4 | 2999 | drei leere Listungen |
| `status-unfindable` | `project/a` mit `never`, `unsearched`, NFC-Pfad | 2999 | `Cafe\u0301.md`, `a.md`, `drafts/x.md` |
| `status-listing-fails` | `project/a` heil | 2999 | keine Collection |
| `status-hash-shared` | `project/a`, `project/b` mit zwei geteilten Hashes | 2999 | alle Pfade gelistet |
| `status-hash-withheld` | `project/a` mit `never`, `project/b` | 2999 | alle lesbaren Pfade gelistet |
| `status-backlog` | `project/a` heil | 2999 | `pending: 7` |
| `status-backlog-fails` | `project/a` heil | 2999 | `status_error` |
| `status-broken-register` | `project/a` mit Register aus drei Feldern | 2999 | `project-a: [a.md]` |

##### Welt `vault`

Fälle: `brain-search/full-hits`, `brain-search/full-cut-at-n`, `brain-search/full-one-scope`, `brain-search/full-no-matches`, `brain-search/full-cloud`, `brain-search/fast`, `brain-search/keyword`, `brain-search/unknown-scope`, `brain-search/count-below-one`, `brain-search/invalid-profile`, `brain-search/missing-query`, `brain-catalog/root`, `brain-catalog/root-cloud`, `brain-catalog/area`, `brain-catalog/read-only-area`, `brain-catalog/local-only-on-cloud`, `brain-catalog/unknown-scope`, `brain-catalog/missing-index`, `brain-catalog/invalid-channel`, `brain-read/whole-file`, `brain-read/section`, `brain-read/windows-separator`, `brain-read/review-centre-local`, `brain-read/review-centre-cloud`, `brain-read/never`, `brain-read/leaves-area`, `brain-read/missing-section`, `brain-read/missing-file`, `brain-read/read-only-area`, `brain-read/local-only-on-cloud`, `brain-read/missing-scope`, `brain-read/unknown-scope`, `brain-neighbors/both-directions`, `brain-neighbors/no-links`, `brain-neighbors/read-only-area`, `brain-neighbors/never-indexed`, `brain-neighbors/leaves-area`, `brain-neighbors/unknown-scope`, `brain-neighbors/missing-scope`, `brain-status/vault-local`, `brain-status/vault-cloud`, `brain-status/invalid-channel`.

`testdata/cases/1b-1-worlds/vault/areas/notes/.brain.toml`
```toml
[area]
scope = "notes"
```

`testdata/cases/1b-1-worlds/vault/areas/notes/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01N0001	guide.md	sha256:n1	1
```

`testdata/cases/1b-1-worlds/vault/areas/notes/graph.json`
```json
{
  "edges": [
    {"from": "guide.md", "to": "start.md"}
  ],
  "links": {"dropped": {}, "resolved": 1, "total": 1},
  "nodes": [
    {"id": "guide.md", "tags": [], "title": "guide"},
    {"id": "start.md", "tags": [], "title": "start"}
  ],
  "scope": "notes"
}
```

`testdata/cases/1b-1-worlds/vault/areas/notes/index.md`
```markdown
# notes

* [Guide](brain://notes/guide.md)
```

`testdata/cases/1b-1-worlds/vault/maintenance/last-run.txt`
```text
2000-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/vault/qmd-fixture.json`
```json
{
  "collections": {
    "notes": ["guide.md"],
    "project-alpha": ["notes/a.md", "notes/b.md", "notes/lonely.md", "review/case.md"],
    "project-beta": ["b.md"]
  },
  "hits": [
    {"collection": "project-alpha", "relative": "notes/a.md", "line": 3, "score": 0.75, "docid": "#a0001", "title": "Alpha notes", "snippet": "The first finding.\nIt spans two lines."},
    {"collection": "project-alpha", "relative": "secret/key.md", "line": 1, "score": 0.7, "docid": "#a0005", "title": "Key", "snippet": "not for anyone"},
    {"collection": "notes", "relative": "guide.md", "line": 5, "score": 0.5, "docid": "#n0001", "title": "Guide", "snippet": "How to read the guide."},
    {"collection": "project-alpha", "relative": "notes/unregistered.md", "line": 1, "score": 0.25, "docid": "#a0009", "title": "Unregistered", "snippet": ""},
    {"collection": "project-beta", "relative": "b.md", "line": 2, "score": 0.1, "docid": "#b0001", "title": "Beta", "snippet": "line one\rline two"}
  ]
}
```

`testdata/cases/1b-1-worlds/vault/registry.toml`
```toml
[[area]]
scope = "project/beta"
path = "{{WORLD}}/repo-beta"

[[area]]
scope = "notes"
path = "{{WORLD}}/sources/notes"
readonly = true

[[area]]
scope = "project/alpha"
path = "{{WORLD}}/repo-alpha"

[[area]]
scope = "project/gamma"
path = "{{WORLD}}/repo-gamma"
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/.brain.toml`
```toml
[area]
scope = "project/alpha"

[privacy]
never = ["secret/**"]

[layout]
review = "review"
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	notes/a.md	sha256:a1	1
01A0002	notes/b.md	sha256:a2	1
01A0003	notes/lonely.md	sha256:a3	1
01A0004	review/case.md	sha256:a4	1
01A0005	secret/key.md	sha256:a5	1
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/graph.json`
```json
{
  "edges": [
    {"from": "notes/a.md", "to": "notes/b.md"},
    {"from": "notes/b.md", "to": "notes/a.md"},
    {"from": "review/case.md", "to": "notes/a.md"}
  ],
  "links": {"dropped": {"external": 1}, "resolved": 3, "total": 4},
  "nodes": [
    {"id": "notes/a.md", "tags": [], "title": "a"},
    {"id": "notes/b.md", "tags": [], "title": "b"},
    {"id": "notes/lonely.md", "tags": [], "title": "lonely"},
    {"id": "review/case.md", "tags": [], "title": "case"}
  ],
  "scope": "project/alpha"
}
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/index.md`
```markdown
# project/alpha

* [Alpha notes](brain://project/alpha/notes/a.md)
* [B](brain://project/alpha/notes/b.md)
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/notes/a.md`
```markdown
# Alpha notes
The first finding.
It spans two lines.

## Second part
Details of the second part.

### Deeper
Still inside.

## Third part
Not in the section.
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/notes/b.md`
```markdown
# B

Links back to [Alpha notes](a.md).
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/notes/lonely.md`
```markdown
# Lonely
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/review/case.md`
```markdown
# Case 1

A pending review.
```

`testdata/cases/1b-1-worlds/vault/repo-alpha/secret/key.md`
```markdown
# Key

not for anyone
```

`testdata/cases/1b-1-worlds/vault/repo-beta/.ultra-brain/config.toml`
```toml
[area]
scope = "project/beta"

[privacy]
mode = "local_only"

[index]
include = ["*.md", "docs/*.md"]
```

`testdata/cases/1b-1-worlds/vault/repo-beta/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01B0001	b.md	sha256:b1	1
```

`testdata/cases/1b-1-worlds/vault/repo-beta/b.md`
```markdown
# Beta

line one
line two
```

`testdata/cases/1b-1-worlds/vault/repo-beta/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "b.md", "tags": [], "title": "b"}
  ],
  "scope": "project/beta"
}
```

`testdata/cases/1b-1-worlds/vault/repo-beta/index.md`
```markdown
# project/beta

* [Beta](brain://project/beta/b.md)
```

`testdata/cases/1b-1-worlds/vault/repo-gamma/.brain.toml`
```toml
[area]
scope = "project/gamma"
```

`testdata/cases/1b-1-worlds/vault/sources/notes/guide.md`
```markdown
# Guide

How to read the guide.

See [start](start.md).
```

##### Welt `local-only`

Fälle: `brain-search/cloud-without-visible-areas`, `brain-catalog/cloud-without-visible-areas`.

`testdata/cases/1b-1-worlds/local-only/registry.toml`
```toml
[[area]]
scope = "project/closed"
path = "{{WORLD}}/repo-closed"
```

`testdata/cases/1b-1-worlds/local-only/repo-closed/.brain.toml`
```toml
[area]
scope = "project/closed"

[privacy]
mode = "local_only"
```

`testdata/cases/1b-1-worlds/local-only/repo-closed/index.md`
```markdown
# project/closed
```

##### Welt `no-registry`

Fälle: `brain-search/missing-registry`, `brain-catalog/missing-registry`.

`testdata/cases/1b-1-worlds/no-registry/README.md`
```markdown
This world has no registry.toml: every brain command fails before it reads an area.
```

##### Welt `missing-manifest`

Fälle: `brain-catalog/missing-manifest`, `brain-read/missing-manifest`, `brain-status/missing-manifest`.

`testdata/cases/1b-1-worlds/missing-manifest/registry.toml`
```toml
[[area]]
scope = "project/bare"
path = "{{WORLD}}/repo-bare"
```

`testdata/cases/1b-1-worlds/missing-manifest/repo-bare/README.md`
```markdown
An area that never declared itself.
```

##### Welt `manifest-without-scope`

Fälle: `brain-catalog/manifest-without-scope`, `brain-neighbors/manifest-without-scope`.

`testdata/cases/1b-1-worlds/manifest-without-scope/registry.toml`
```toml
[[area]]
scope = "project/vague"
path = "{{WORLD}}/repo-vague"
```

`testdata/cases/1b-1-worlds/manifest-without-scope/repo-vague/.brain.toml`
```toml
[area]
scope = ""

[privacy]
mode = "manual_cloud"
```

Die `[area]`-Tabelle mit leerem `scope` ist Absicht: `translateDir` faltet sie in `.loomux/config.toml`, und eine `.loomux/config.toml` ganz ohne `[area]` deklarierte für `brain` nichts (Task 1) — loomux meldete dann `no manifest found` statt des Scope-Fehlers, den die Referenz meldet.

##### Welt `damaged`

Fälle: `brain-search/broken-register`, `brain-neighbors/broken-graph`, `brain-status/broken-graph`.

`testdata/cases/1b-1-worlds/damaged/registry.toml`
```toml
[[area]]
scope = "project/torn"
path = "{{WORLD}}/repo-torn"

[[area]]
scope = "project/garbled"
path = "{{WORLD}}/repo-garbled"
```

`testdata/cases/1b-1-worlds/damaged/repo-garbled/.brain.toml`
```toml
[area]
scope = "project/garbled"
```

`testdata/cases/1b-1-worlds/damaged/repo-garbled/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01G0001	three-fields.md	sha256:g1
```

`testdata/cases/1b-1-worlds/damaged/repo-torn/.brain.toml`
```toml
[area]
scope = "project/torn"
```

`testdata/cases/1b-1-worlds/damaged/repo-torn/graph.json`
```json
{"edges": []}
```

##### Welt `broken-registry`

Fälle: `brain-catalog/broken-registry`.

`testdata/cases/1b-1-worlds/broken-registry/registry.toml`
```toml
[[area
```

##### Welt `manifest-wrong-type`

Fälle: `brain-catalog/manifest-wrong-type`.

`testdata/cases/1b-1-worlds/manifest-wrong-type/registry.toml`
```toml
[[area]]
scope = "project/typed"
path = "{{WORLD}}/repo-typed"
```

`testdata/cases/1b-1-worlds/manifest-wrong-type/repo-typed/.brain.toml`
```toml
[area]
scope = 3
```

##### Welt `engine-fails`

Fälle: `brain-search/engine-fails`.

`testdata/cases/1b-1-worlds/engine-fails/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["a.md"]},
  "search_error": "index is locked\n"
}
```

`testdata/cases/1b-1-worlds/engine-fails/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/engine-fails/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/engine-fails/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1	1
```

`testdata/cases/1b-1-worlds/engine-fails/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-stamp-missing`

Fälle: `brain-status/stamp-missing`.

`testdata/cases/1b-1-worlds/status-stamp-missing/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["a.md"]}
}
```

`testdata/cases/1b-1-worlds/status-stamp-missing/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-stamp-missing/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-stamp-missing/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1	1
```

`testdata/cases/1b-1-worlds/status-stamp-missing/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-stamp-stale`

Fälle: `brain-status/stamp-stale`.

`testdata/cases/1b-1-worlds/status-stamp-stale/maintenance/last-run.txt`
```text
2000-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-stamp-stale/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["a.md"]}
}
```

`testdata/cases/1b-1-worlds/status-stamp-stale/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-stamp-stale/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-stamp-stale/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1	1
```

`testdata/cases/1b-1-worlds/status-stamp-stale/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-stamp-fresh`

Fälle: `brain-status/stamp-fresh`.

`testdata/cases/1b-1-worlds/status-stamp-fresh/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-stamp-fresh/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["a.md"]}
}
```

`testdata/cases/1b-1-worlds/status-stamp-fresh/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-stamp-fresh/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-stamp-fresh/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1	1
```

`testdata/cases/1b-1-worlds/status-stamp-fresh/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-include-globs`

Fälle: `brain-status/include-globs`.

`testdata/cases/1b-1-worlds/status-include-globs/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-include-globs/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["a.md"]}
}
```

`testdata/cases/1b-1-worlds/status-include-globs/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-include-globs/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"

[index]
include = ["docs/**/*.md", "README*.md", "notes/*.md"]
```

`testdata/cases/1b-1-worlds/status-include-globs/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1	1
```

`testdata/cases/1b-1-worlds/status-include-globs/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-path-missing`

Fälle: `brain-status/path-missing`.

`testdata/cases/1b-1-worlds/status-path-missing/areas/notes/.brain.toml`
```toml
[area]
scope = "notes"
```

`testdata/cases/1b-1-worlds/status-path-missing/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-path-missing/registry.toml`
```toml
[[area]]
scope = "notes"
path = "{{WORLD}}/sources/gone"
readonly = true
```

##### Welt `status-never-indexed`

Fälle: `brain-status/never-indexed`.

`testdata/cases/1b-1-worlds/status-never-indexed/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-never-indexed/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-never-indexed/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

##### Welt `status-links-unresolved`

Fälle: `brain-status/links-unresolved`.

`testdata/cases/1b-1-worlds/status-links-unresolved/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-links-unresolved/qmd-fixture.json`
```json
{
  "collections": {"project-a": [], "project-b": [], "project-c": []}
}
```

`testdata/cases/1b-1-worlds/status-links-unresolved/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"

[[area]]
scope = "project/b"
path = "{{WORLD}}/repo-b"

[[area]]
scope = "project/c"
path = "{{WORLD}}/repo-c"
```

`testdata/cases/1b-1-worlds/status-links-unresolved/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-links-unresolved/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {"unknown_target": 2, "external": 1}, "resolved": 2, "total": 5},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

`testdata/cases/1b-1-worlds/status-links-unresolved/repo-b/.brain.toml`
```toml
[area]
scope = "project/b"
```

`testdata/cases/1b-1-worlds/status-links-unresolved/repo-b/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 1, "total": 3},
  "nodes": [
    {"id": "b.md", "tags": [], "title": "b"}
  ],
  "scope": "project/b"
}
```

`testdata/cases/1b-1-worlds/status-links-unresolved/repo-c/.brain.toml`
```toml
[area]
scope = "project/c"
```

`testdata/cases/1b-1-worlds/status-links-unresolved/repo-c/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {"anchor": 2}, "resolved": 2, "total": 4},
  "nodes": [
    {"id": "c.md", "tags": [], "title": "c"}
  ],
  "scope": "project/c"
}
```

##### Welt `status-unfindable`

Fälle: `brain-status/unfindable`.

`testdata/cases/1b-1-worlds/status-unfindable/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-unfindable/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["Cafe\u0301.md", "a.md", "drafts/x.md"]}
}
```

`testdata/cases/1b-1-worlds/status-unfindable/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-unfindable/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"

[privacy]
never = ["private/**"]

[index]
unsearched = ["drafts/**"]
```

`testdata/cases/1b-1-worlds/status-unfindable/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	Café.md	sha256:a0	1
01A0002	a.md	sha256:a1	1
01A0003	b.md	sha256:a2	1
01A0004	c.md	sha256:a3	1
01A0005	d.md	sha256:a4	1
01A0006	e.md	sha256:a5	1
01A0007	drafts/x.md	sha256:a6	1
01A0008	private/y.md	sha256:a7	1
```

`testdata/cases/1b-1-worlds/status-unfindable/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-listing-fails`

Fälle: `brain-status/listing-fails`.

`testdata/cases/1b-1-worlds/status-listing-fails/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-listing-fails/qmd-fixture.json`
```json
{
  "collections": {}
}
```

`testdata/cases/1b-1-worlds/status-listing-fails/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-listing-fails/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-listing-fails/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1	1
```

`testdata/cases/1b-1-worlds/status-listing-fails/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-hash-shared`

Fälle: `brain-status/hash-shared`.

`testdata/cases/1b-1-worlds/status-hash-shared/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-hash-shared/qmd-fixture.json`
```json
{
  "collections": {
    "project-a": ["copy.md", "shared.md", "zeta-copy.md"],
    "project-b": ["shared.md", "zeta.md"]
  }
}
```

`testdata/cases/1b-1-worlds/status-hash-shared/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"

[[area]]
scope = "project/b"
path = "{{WORLD}}/repo-b"
```

`testdata/cases/1b-1-worlds/status-hash-shared/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-hash-shared/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	copy.md	sha256:aa	1
01A0002	shared.md	sha256:aa	1
01A0003	zeta-copy.md	sha256:00	1
```

`testdata/cases/1b-1-worlds/status-hash-shared/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "copy.md", "tags": [], "title": "copy"},
    {"id": "shared.md", "tags": [], "title": "shared"},
    {"id": "zeta-copy.md", "tags": [], "title": "zeta-copy"}
  ],
  "scope": "project/a"
}
```

`testdata/cases/1b-1-worlds/status-hash-shared/repo-b/.brain.toml`
```toml
[area]
scope = "project/b"
```

`testdata/cases/1b-1-worlds/status-hash-shared/repo-b/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01B0001	shared.md	sha256:aa	1
01B0002	zeta.md	sha256:00	1
```

`testdata/cases/1b-1-worlds/status-hash-shared/repo-b/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "shared.md", "tags": [], "title": "shared"},
    {"id": "zeta.md", "tags": [], "title": "zeta"}
  ],
  "scope": "project/b"
}
```

##### Welt `status-hash-withheld`

Fälle: `brain-status/hash-withheld`.

`testdata/cases/1b-1-worlds/status-hash-withheld/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-hash-withheld/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["open.md"], "project-b": ["key.md"]}
}
```

`testdata/cases/1b-1-worlds/status-hash-withheld/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"

[[area]]
scope = "project/b"
path = "{{WORLD}}/repo-b"
```

`testdata/cases/1b-1-worlds/status-hash-withheld/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"

[privacy]
never = ["private/**"]
```

`testdata/cases/1b-1-worlds/status-hash-withheld/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	open.md	sha256:o1	1
01A0002	private/key.md	sha256:k1	1
01A0003	private/k2.md	sha256:k2	1
01A0004	private/k3.md	sha256:k2	1
```

`testdata/cases/1b-1-worlds/status-hash-withheld/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "open.md", "tags": [], "title": "open"}
  ],
  "scope": "project/a"
}
```

`testdata/cases/1b-1-worlds/status-hash-withheld/repo-b/.brain.toml`
```toml
[area]
scope = "project/b"
```

`testdata/cases/1b-1-worlds/status-hash-withheld/repo-b/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01B0001	key.md	sha256:k1	1
```

`testdata/cases/1b-1-worlds/status-hash-withheld/repo-b/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "key.md", "tags": [], "title": "key"}
  ],
  "scope": "project/b"
}
```

##### Welt `status-backlog`

Fälle: `brain-status/backlog`.

`testdata/cases/1b-1-worlds/status-backlog/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-backlog/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["a.md"]},
  "pending": 7
}
```

`testdata/cases/1b-1-worlds/status-backlog/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-backlog/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-backlog/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1	1
```

`testdata/cases/1b-1-worlds/status-backlog/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-backlog-fails`

Fälle: `brain-status/backlog-fails`.

`testdata/cases/1b-1-worlds/status-backlog-fails/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-backlog-fails/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["a.md"]},
  "status_error": "index is locked\n"
}
```

`testdata/cases/1b-1-worlds/status-backlog-fails/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-backlog-fails/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-backlog-fails/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1	1
```

`testdata/cases/1b-1-worlds/status-backlog-fails/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

##### Welt `status-broken-register`

Fälle: `brain-status/broken-register`.

`testdata/cases/1b-1-worlds/status-broken-register/maintenance/last-run.txt`
```text
2999-01-01T00:00:00+00:00
```

`testdata/cases/1b-1-worlds/status-broken-register/qmd-fixture.json`
```json
{
  "collections": {"project-a": ["a.md"]}
}
```

`testdata/cases/1b-1-worlds/status-broken-register/registry.toml`
```toml
[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"
```

`testdata/cases/1b-1-worlds/status-broken-register/repo-a/.brain.toml`
```toml
[area]
scope = "project/a"
```

`testdata/cases/1b-1-worlds/status-broken-register/repo-a/_identities.tsv`
```text
doc_id	pfad	content_hash	revision
01A0001	a.md	sha256:a1
```

`testdata/cases/1b-1-worlds/status-broken-register/repo-a/graph.json`
```json
{
  "edges": [],
  "links": {"dropped": {}, "resolved": 0, "total": 0},
  "nodes": [
    {"id": "a.md", "tags": [], "title": "a"}
  ],
  "scope": "project/a"
}
```

- [ ] **Step 5: Welten prüfen**

```bash
find testdata/cases/1b-1-worlds -type f | wc -l
```
Expected: `130`.
```bash
awk -F'\t' 'NF != 4 {print FILENAME ": " FNR ": " NF}' $(find testdata/cases/1b-1-worlds -name _identities.tsv | sort)
```
Expected, genau zwei Zeilen (die beiden absichtlich kaputten Register; gemessen an den erzeugten Welten):
```
testdata/cases/1b-1-worlds/damaged/repo-garbled/_identities.tsv: 2: 3
testdata/cases/1b-1-worlds/status-broken-register/repo-a/_identities.tsv: 2: 3
```
```bash
grep -c "$(printf 'Caf\303\251.md')" testdata/cases/1b-1-worlds/status-unfindable/repo-a/_identities.tsv
```
Expected: `1` (das `é` ist NFC).

- [ ] **Step 6: Aufzeichnen**

Je Fall ein Aufruf, ohne laufenden brain-Daemon: ein Daemon für `%LOCALAPPDATA%\brain` antwortet einer gestagten Welt nicht, weil der Pipe-Name aus dem Zustandsverzeichnis gebildet wird (`1b-1-facts/status.md` §9). Der Rekorder setzt `BRAIN_STATE_DIR` = gestagte Welt und `PYTHONUTF8=1`, faltet `\r\n` im stdout und stellt `$TEMP/loomux-fakeqmd` vor `PATH`, sodass Pythons `launcher("qmd")` den Fake findet (Task 11 Step 5). Gegenüber der Befehlsvorlage des Vertrags stehen zwei Zusätze in jedem Aufruf, beide für „Quellen nur lesen": `uv run --no-sync` (kein Abgleich der `.venv` im Quell-Worktree) und `--env PYTHONDONTWRITEBYTECODE=1` (keine `.pyc` unter `ub/src`); beide betreffen nur, was uv und Python auf die Platte schreiben, nicht, was `brain-mcp` ausgibt; die Messung dieses Plans lief selbst mit beiden Einstellungen.

| Nr. | Verb / Fall | Welt | `--cmd` | Compare | Exit |
|---:|---|---|---|---|---:|
| 1 | brain-search/full-hits | vault | `brain-mcp search "alpha notes" --profile full` | data | 0 |
| 2 | brain-search/full-cut-at-n | vault | `brain-mcp search "alpha notes" --profile full -n 2` | data | 0 |
| 3 | brain-search/full-one-scope | vault | `brain-mcp search guide --scope notes --profile full` | data | 0 |
| 4 | brain-search/full-no-matches | vault | `brain-mcp search anything --scope project/gamma --profile full` | data | 0 |
| 5 | brain-search/full-cloud | vault | `brain-mcp search "alpha notes" --profile full --channel cloud` | data | 0 |
| 6 | brain-search/cloud-without-visible-areas | local-only | `brain-mcp search closed --profile full --channel cloud` | data | 0 |
| 7 | brain-search/fast | vault | `brain-mcp search "alpha notes"` | message | 0 |
| 8 | brain-search/keyword | vault | `brain-mcp search "alpha notes" --profile keyword` | message | 0 |
| 9 | brain-search/unknown-scope | vault | `brain-mcp search q --scope nope --profile full` | data | 1 |
| 10 | brain-search/broken-register | damaged | `brain-mcp search q --profile full` | data | 1 |
| 11 | brain-search/count-below-one | vault | `brain-mcp search q -n 0` | data | 2 |
| 12 | brain-search/invalid-profile | vault | `brain-mcp search q --profile deep` | data | 2 |
| 13 | brain-catalog/root | vault | `brain-mcp catalog` | data | 0 |
| 14 | brain-catalog/root-cloud | vault | `brain-mcp catalog --channel cloud` | data | 0 |
| 15 | brain-catalog/area | vault | `brain-mcp catalog --scope project/alpha` | data | 0 |
| 16 | brain-catalog/read-only-area | vault | `brain-mcp catalog --scope notes` | data | 0 |
| 17 | brain-catalog/local-only-on-cloud | vault | `brain-mcp catalog --scope project/beta --channel cloud` | data | 1 |
| 18 | brain-catalog/unknown-scope | vault | `brain-mcp catalog --scope nope` | data | 1 |
| 19 | brain-catalog/missing-index | vault | `brain-mcp catalog --scope project/gamma` | data | 1 |
| 20 | brain-catalog/cloud-without-visible-areas | local-only | `brain-mcp catalog --channel cloud` | data | 0 |
| 21 | brain-catalog/missing-registry | no-registry | `brain-mcp catalog` | data | 1 |
| 22 | brain-catalog/missing-manifest | missing-manifest | `brain-mcp catalog` | data | 1 |
| 23 | brain-catalog/manifest-without-scope | manifest-without-scope | `brain-mcp catalog` | data | 1 |
| 24 | brain-catalog/invalid-channel | vault | `brain-mcp catalog --channel public` | data | 2 |
| 25 | brain-read/whole-file | vault | `brain-mcp read notes/a.md --scope project/alpha` | data | 0 |
| 26 | brain-read/section | vault | `brain-mcp read notes/a.md --scope project/alpha --section "Second part"` | data | 0 |
| 27 | brain-read/windows-separator | vault | `brain-mcp read 'notes\a.md' --scope project/alpha --section Deeper` | data | 0 |
| 28 | brain-read/review-centre-local | vault | `brain-mcp read review/case.md --scope project/alpha` | data | 0 |
| 29 | brain-read/review-centre-cloud | vault | `brain-mcp read review/case.md --scope project/alpha --channel cloud` | data | 1 |
| 30 | brain-read/never | vault | `brain-mcp read secret/key.md --scope project/alpha` | data | 1 |
| 31 | brain-read/leaves-area | vault | `brain-mcp read ../repo-beta/b.md --scope project/alpha` | data | 1 |
| 32 | brain-read/missing-section | vault | `brain-mcp read notes/a.md --scope project/alpha --section Nowhere` | data | 1 |
| 33 | brain-read/missing-file | vault | `brain-mcp read notes/gone.md --scope project/alpha` | data | 1 |
| 34 | brain-read/read-only-area | vault | `brain-mcp read guide.md --scope notes` | data | 0 |
| 35 | brain-read/local-only-on-cloud | vault | `brain-mcp read b.md --scope project/beta --channel cloud` | data | 1 |
| 36 | brain-read/missing-scope | vault | `brain-mcp read notes/a.md` | data | 2 |
| 37 | brain-neighbors/both-directions | vault | `brain-mcp neighbors notes/a.md --scope project/alpha` | data | 0 |
| 38 | brain-neighbors/no-links | vault | `brain-mcp neighbors notes/lonely.md --scope project/alpha` | data | 0 |
| 39 | brain-neighbors/read-only-area | vault | `brain-mcp neighbors guide.md --scope notes` | data | 0 |
| 40 | brain-neighbors/never-indexed | vault | `brain-mcp neighbors a.md --scope project/gamma` | data | 1 |
| 41 | brain-neighbors/leaves-area | vault | `brain-mcp neighbors ../repo-beta/b.md --scope project/alpha` | data | 1 |
| 42 | brain-neighbors/unknown-scope | vault | `brain-mcp neighbors a.md --scope nope` | data | 1 |
| 43 | brain-neighbors/broken-graph | damaged | `brain-mcp neighbors a.md --scope project/torn` | data | 1 |
| 44 | brain-status/stamp-missing | status-stamp-missing | `brain-mcp status` | data | 0 |
| 45 | brain-status/stamp-stale | status-stamp-stale | `brain-mcp status` | data | 0 |
| 46 | brain-status/stamp-fresh | status-stamp-fresh | `brain-mcp status` | data | 0 |
| 47 | brain-status/include-globs | status-include-globs | `brain-mcp status` | data | 0 |
| 48 | brain-status/path-missing | status-path-missing | `brain-mcp status` | data | 0 |
| 49 | brain-status/never-indexed | status-never-indexed | `brain-mcp status` | data | 0 |
| 50 | brain-status/links-unresolved | status-links-unresolved | `brain-mcp status` | data | 0 |
| 51 | brain-status/unfindable | status-unfindable | `brain-mcp status` | data | 0 |
| 52 | brain-status/listing-fails | status-listing-fails | `brain-mcp status` | data | 0 |
| 53 | brain-status/hash-shared | status-hash-shared | `brain-mcp status` | data | 0 |
| 54 | brain-status/hash-withheld | status-hash-withheld | `brain-mcp status` | data | 0 |
| 55 | brain-status/backlog | status-backlog | `brain-mcp status` | data | 0 |
| 56 | brain-status/backlog-fails | status-backlog-fails | `brain-mcp status` | data | 0 |
| 57 | brain-status/vault-local | vault | `brain-mcp status` | data | 0 |
| 58 | brain-status/vault-cloud | vault | `brain-mcp status --channel cloud` | data | 0 |
| 59 | brain-status/broken-graph | damaged | `brain-mcp status` | data | 1 |
| 60 | brain-search/missing-query | vault | `brain-mcp search` | data | 2 |
| 61 | brain-search/missing-registry | no-registry | `brain-mcp search q --profile full` | data | 1 |
| 62 | brain-search/engine-fails | engine-fails | `brain-mcp search q --profile full` | data | 1 |
| 63 | brain-catalog/broken-registry | broken-registry | `brain-mcp catalog` | data | 1 |
| 64 | brain-catalog/manifest-wrong-type | manifest-wrong-type | `brain-mcp catalog` | data | 1 |
| 65 | brain-read/unknown-scope | vault | `brain-mcp read a.md --scope nope` | data | 1 |
| 66 | brain-read/missing-manifest | missing-manifest | `brain-mcp read a.md --scope project/bare` | data | 1 |
| 67 | brain-neighbors/missing-scope | vault | `brain-mcp neighbors notes/a.md` | data | 2 |
| 68 | brain-neighbors/manifest-without-scope | manifest-without-scope | `brain-mcp neighbors a.md --scope project/vague` | data | 1 |
| 69 | brain-status/invalid-channel | vault | `brain-mcp status --channel public` | data | 2 |
| 70 | brain-status/missing-manifest | missing-manifest | `brain-mcp status` | data | 1 |
| 71 | brain-status/broken-register | status-broken-register | `brain-mcp status` | data | 1 |

Die Reihenfolge der Tabelle ist gleichgültig: jeder Fall wird für sich aufgezeichnet, und die Suite liest die Fälle aus dem Verzeichnisbaum.

1. `brain-search/full-hits`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search "alpha notes" --profile full' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/full-hits --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: full profile: a multi-line snippet, a never hit withheld, a hit outside the register, a stale stamp'
```

2. `brain-search/full-cut-at-n`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search "alpha notes" --profile full -n 2' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/full-cut-at-n --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: the engine is asked for n hits; the withheld one costs a place'
```

3. `brain-search/full-one-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search guide --scope notes --profile full' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/full-one-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: one read-only area; its register lies in the state directory'
```

4. `brain-search/full-no-matches`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search anything --scope project/gamma --profile full' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/full-no-matches --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: the engine answers empty twice: no matches'
```

5. `brain-search/full-cloud`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search "alpha notes" --profile full --channel cloud' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/full-cloud --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: the cloud channel does not ask the local_only area'
```

6. `brain-search/cloud-without-visible-areas`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search closed --profile full --channel cloud' --world testdata/cases/1b-1-worlds/local-only --out testdata/cases/1b-1-source/brain-search/cloud-without-visible-areas --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: no visible area: no matches without asking'
```

7. `brain-search/fast`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search "alpha notes"' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/fast --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: default profile fast: the reference runs qmd vsearch, loomux the MCP vec search' --compare message
```

8. `brain-search/keyword`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search "alpha notes" --profile keyword' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/keyword --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: profile keyword: BM25 in the reference, lex search over MCP in loomux' --compare message
```

9. `brain-search/unknown-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search q --scope nope --profile full' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/unknown-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an unknown scope is a runtime error with no output'
```

10. `brain-search/broken-register`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search q --profile full' --world testdata/cases/1b-1-worlds/damaged --out testdata/cases/1b-1-source/brain-search/broken-register --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an unreadable register fails the search even without hits'
```

11. `brain-search/count-below-one`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search q -n 0' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/count-below-one --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: usage error: -n must be at least 1'
```

12. `brain-search/invalid-profile`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search q --profile deep' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/invalid-profile --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: usage error: invalid profile choice'
```

13. `brain-catalog/root`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-catalog/root --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: root catalog of every visible area, sorted by scope'
```

14. `brain-catalog/root-cloud`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog --channel cloud' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-catalog/root-cloud --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: the cloud root catalog leaves the local_only area out'
```

15. `brain-catalog/area`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-catalog/area --notes "loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a writable area's index.md, byte for byte"
```

16. `brain-catalog/read-only-area`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog --scope notes' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-catalog/read-only-area --notes "loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a read-only area's index.md from the state directory"
```

17. `brain-catalog/local-only-on-cloud`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog --scope project/beta --channel cloud' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-catalog/local-only-on-cloud --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a local_only area is an unknown scope on the cloud channel'
```

18. `brain-catalog/unknown-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog --scope nope' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-catalog/unknown-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an unknown scope is a runtime error with no output'
```

19. `brain-catalog/missing-index`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog --scope project/gamma' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-catalog/missing-index --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an area without index.md fails'
```

20. `brain-catalog/cloud-without-visible-areas`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog --channel cloud' --world testdata/cases/1b-1-worlds/local-only --out testdata/cases/1b-1-source/brain-catalog/cloud-without-visible-areas --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: no visible area: the bare root catalog'
```

21. `brain-catalog/missing-registry`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog' --world testdata/cases/1b-1-worlds/no-registry --out testdata/cases/1b-1-source/brain-catalog/missing-registry --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: no registry is a runtime error'
```

22. `brain-catalog/missing-manifest`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog' --world testdata/cases/1b-1-worlds/missing-manifest --out testdata/cases/1b-1-source/brain-catalog/missing-manifest --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a registered area without a manifest fails every command'
```

23. `brain-catalog/manifest-without-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog' --world testdata/cases/1b-1-worlds/manifest-without-scope --out testdata/cases/1b-1-source/brain-catalog/manifest-without-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a manifest without [area] scope fails every command'
```

24. `brain-catalog/invalid-channel`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog --channel public' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-catalog/invalid-channel --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: usage error: invalid channel choice'
```

25. `brain-read/whole-file`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read notes/a.md --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/whole-file --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: one file, as read_text gives it'
```

26. `brain-read/section`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read notes/a.md --scope project/alpha --section "Second part"' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/section --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a section runs to the next heading of the same or a higher level'
```

27. `brain-read/windows-separator`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd "brain-mcp read 'notes\a.md' --scope project/alpha --section Deeper" --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/windows-separator --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a backslash in the path is a separator'
```

28. `brain-read/review-centre-local`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read review/case.md --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/review-centre-local --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: the review centre stays open on the local channel'
```

29. `brain-read/review-centre-cloud`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read review/case.md --scope project/alpha --channel cloud' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/review-centre-cloud --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: the review centre is refused on the cloud channel'
```

30. `brain-read/never`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read secret/key.md --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/never --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a path under [privacy] never is refused'
```

31. `brain-read/leaves-area`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read ../repo-beta/b.md --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/leaves-area --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a path climbing out of the area is refused'
```

32. `brain-read/missing-section`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read notes/a.md --scope project/alpha --section Nowhere' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/missing-section --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a missing section is a runtime error'
```

33. `brain-read/missing-file`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read notes/gone.md --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/missing-file --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a missing file is a runtime error'
```

34. `brain-read/read-only-area`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read guide.md --scope notes' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/read-only-area --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a read-only area is read at its source path'
```

35. `brain-read/local-only-on-cloud`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read b.md --scope project/beta --channel cloud' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/local-only-on-cloud --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a local_only area is an unknown scope on the cloud channel'
```

36. `brain-read/missing-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read notes/a.md' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/missing-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: usage error: --scope is required'
```

37. `brain-neighbors/both-directions`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors notes/a.md --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-neighbors/both-directions --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: incoming and outgoing links, sorted'
```

38. `brain-neighbors/no-links`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors notes/lonely.md --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-neighbors/no-links --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an empty direction prints a dash'
```

39. `brain-neighbors/read-only-area`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors guide.md --scope notes' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-neighbors/read-only-area --notes "loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a read-only area's graph lies in the state directory"
```

40. `brain-neighbors/never-indexed`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors a.md --scope project/gamma' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-neighbors/never-indexed --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an area without graph.json is a runtime error'
```

41. `brain-neighbors/leaves-area`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors ../repo-beta/b.md --scope project/alpha' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-neighbors/leaves-area --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a path climbing out of the area is refused before the graph is read'
```

42. `brain-neighbors/unknown-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors a.md --scope nope' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-neighbors/unknown-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an unknown scope is a runtime error with no output'
```

43. `brain-neighbors/broken-graph`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors a.md --scope project/torn' --world testdata/cases/1b-1-worlds/damaged --out testdata/cases/1b-1-source/brain-neighbors/broken-graph --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a graph.json without links is a runtime error'
```

44. `brain-status/stamp-missing`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-stamp-missing --out testdata/cases/1b-1-source/brain-status/stamp-missing --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L1a: no reconcile stamp'
```

45. `brain-status/stamp-stale`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-stamp-stale --out testdata/cases/1b-1-source/brain-status/stamp-stale --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L1b: a stamp older than 24 h'
```

46. `brain-status/stamp-fresh`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-stamp-fresh --out testdata/cases/1b-1-source/brain-status/stamp-fresh --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L1c: a fresh stamp, nothing else to report'
```

47. `brain-status/include-globs`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-include-globs --out testdata/cases/1b-1-source/brain-status/include-globs --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L2: more than one include glob'
```

48. `brain-status/path-missing`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-path-missing --out testdata/cases/1b-1-source/brain-status/path-missing --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L3: a read-only area whose path does not exist'
```

49. `brain-status/never-indexed`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-never-indexed --out testdata/cases/1b-1-source/brain-status/never-indexed --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L4: an area without graph.json'
```

50. `brain-status/links-unresolved`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-links-unresolved --out testdata/cases/1b-1-source/brain-status/links-unresolved --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L5: fewer than half of the links resolved, with and without dropped reasons'
```

51. `brain-status/unfindable`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-unfindable --out testdata/cases/1b-1-source/brain-status/unfindable --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L6a: register paths qmd does not list; never, unsearched and NFC left out'
```

52. `brain-status/listing-fails`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-listing-fails --out testdata/cases/1b-1-source/brain-status/listing-fails --notes "loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L6b: qmd ls fails for the area's collection"
```

53. `brain-status/hash-shared`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-hash-shared --out testdata/cases/1b-1-source/brain-status/hash-shared --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L7b: one content hash under several paths, lines sorted by hash'
```

54. `brain-status/hash-withheld`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-hash-withheld --out testdata/cases/1b-1-source/brain-status/hash-withheld --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L7a: a shared hash whose other path is under [privacy] never'
```

55. `brain-status/backlog`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-backlog --out testdata/cases/1b-1-source/brain-status/backlog --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L8a: documents waiting for embedding'
```

56. `brain-status/backlog-fails`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-backlog-fails --out testdata/cases/1b-1-source/brain-status/backlog-fails --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: L8b: qmd status fails'
```

57. `brain-status/vault-local`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-status/vault-local --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a mixed vault on the local channel'
```

58. `brain-status/vault-cloud`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status --channel cloud' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-status/vault-cloud --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: the same vault on the cloud channel leaves the local_only area out'
```

59. `brain-status/broken-graph`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/damaged --out testdata/cases/1b-1-source/brain-status/broken-graph --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a graph.json without links aborts status'
```

60. `brain-search/missing-query`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-search/missing-query --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: usage error: the query is required'
```

61. `brain-search/missing-registry`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search q --profile full' --world testdata/cases/1b-1-worlds/no-registry --out testdata/cases/1b-1-source/brain-search/missing-registry --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: no registry is a runtime error before the engine is asked'
```

62. `brain-search/engine-fails`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp search q --profile full' --world testdata/cases/1b-1-worlds/engine-fails --out testdata/cases/1b-1-source/brain-search/engine-fails --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: the engine fails: an error with no output, never an empty answer'
```

63. `brain-catalog/broken-registry`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog' --world testdata/cases/1b-1-worlds/broken-registry --out testdata/cases/1b-1-source/brain-catalog/broken-registry --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a registry that is not TOML is a runtime error'
```

64. `brain-catalog/manifest-wrong-type`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp catalog' --world testdata/cases/1b-1-worlds/manifest-wrong-type --out testdata/cases/1b-1-source/brain-catalog/manifest-wrong-type --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an [area] scope that is not a string fails every command'
```

65. `brain-read/unknown-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read a.md --scope nope' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-read/unknown-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: an unknown scope is a runtime error with no output'
```

66. `brain-read/missing-manifest`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp read a.md --scope project/bare' --world testdata/cases/1b-1-worlds/missing-manifest --out testdata/cases/1b-1-source/brain-read/missing-manifest --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a registered area without a manifest fails read too'
```

67. `brain-neighbors/missing-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors notes/a.md' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-neighbors/missing-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: usage error: --scope is required'
```

68. `brain-neighbors/manifest-without-scope`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp neighbors a.md --scope project/vague' --world testdata/cases/1b-1-worlds/manifest-without-scope --out testdata/cases/1b-1-source/brain-neighbors/manifest-without-scope --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a manifest without [area] scope fails neighbors too'
```

69. `brain-status/invalid-channel`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status --channel public' --world testdata/cases/1b-1-worlds/vault --out testdata/cases/1b-1-source/brain-status/invalid-channel --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: usage error: invalid channel choice'
```

70. `brain-status/missing-manifest`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/missing-manifest --out testdata/cases/1b-1-source/brain-status/missing-manifest --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a registered area without a manifest aborts status'
```

71. `brain-status/broken-register`
```bash
bin/loomux.exe dev record-case --argv "uv run --no-sync --project C:/Users/micro/Documents/#GIT/loomux-src/ub brain-mcp" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" --env PYTHONDONTWRITEBYTECODE=1 --path-prepend "$TEMP/loomux-fakeqmd" --cmd 'brain-mcp status' --world testdata/cases/1b-1-worlds/status-broken-register --out testdata/cases/1b-1-source/brain-status/broken-register --notes 'loomux-1a-source (3cc72d2), brain-mcp over fakeqmd: a register row with three fields aborts status'
```

Expected je Aufruf: Exit 0 von `dev record-case` und keine Ausgabe; Exit und stdout des aufgezeichneten Befehls prüft Step 7.

- [ ] **Step 7: Aufzeichnungen prüfen**

Prüfsummen von `exit` und `stdout` jedes Falls, gemessen an der Referenz (siehe Messgrundlage):

```bash
cat > "$TEMP/task12-recorded.sha256" <<'EOF'
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-search/full-hits/exit
4caf6d0906aef042ece637f28c7be7b3ae783b4fc3e9a5fd8aff028e327c0d61  testdata/cases/1b-1-source/brain-search/full-hits/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-search/full-cut-at-n/exit
276d11ec6d5b61b7071341d70968cc532af1b09227189673ab9a02e738b48783  testdata/cases/1b-1-source/brain-search/full-cut-at-n/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-search/full-one-scope/exit
5cfe0461e87190933403188d173a6d2a5d1f63da9306b1efccccea29333747ca  testdata/cases/1b-1-source/brain-search/full-one-scope/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-search/full-no-matches/exit
c5f6287b14b07b920edad278183e010bb5fff174c7d3f43a69ec6d332c631bae  testdata/cases/1b-1-source/brain-search/full-no-matches/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-search/full-cloud/exit
60ebf13d1d3c3bf7db5945f33f140659983cb8a9ddd5c0d1696093f1a93ba5ef  testdata/cases/1b-1-source/brain-search/full-cloud/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-search/cloud-without-visible-areas/exit
c5f6287b14b07b920edad278183e010bb5fff174c7d3f43a69ec6d332c631bae  testdata/cases/1b-1-source/brain-search/cloud-without-visible-areas/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-search/fast/exit
4caf6d0906aef042ece637f28c7be7b3ae783b4fc3e9a5fd8aff028e327c0d61  testdata/cases/1b-1-source/brain-search/fast/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-search/keyword/exit
4caf6d0906aef042ece637f28c7be7b3ae783b4fc3e9a5fd8aff028e327c0d61  testdata/cases/1b-1-source/brain-search/keyword/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-search/unknown-scope/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-search/unknown-scope/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-search/broken-register/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-search/broken-register/stdout
53c234e5e8472b6ac51c1ae1cab3fe06fad053beb8ebfd8977b010655bfdd3c3  testdata/cases/1b-1-source/brain-search/count-below-one/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-search/count-below-one/stdout
53c234e5e8472b6ac51c1ae1cab3fe06fad053beb8ebfd8977b010655bfdd3c3  testdata/cases/1b-1-source/brain-search/invalid-profile/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-search/invalid-profile/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-catalog/root/exit
6cd9c8047d5b1d28b6d79d31fcc1e8a5f05e543b41943ac650b9b84769331011  testdata/cases/1b-1-source/brain-catalog/root/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-catalog/root-cloud/exit
622997f10561ab3a4440d9b1434b7317c8268f028690b35568e15af9edf11256  testdata/cases/1b-1-source/brain-catalog/root-cloud/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-catalog/area/exit
f69bfacdcd581f3d72c447a2e464a44f32ba7080d32bcd3f12b1be31fd2f864f  testdata/cases/1b-1-source/brain-catalog/area/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-catalog/read-only-area/exit
2c2875034286d91d62f91339b491960a8b646c50bb7b1b1675966a696faa5081  testdata/cases/1b-1-source/brain-catalog/read-only-area/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-catalog/local-only-on-cloud/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/local-only-on-cloud/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-catalog/unknown-scope/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/unknown-scope/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-catalog/missing-index/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/missing-index/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-catalog/cloud-without-visible-areas/exit
f7cc501a3cb383010b5b2da94f08f8658783d106735db4d9cbc04feea4503173  testdata/cases/1b-1-source/brain-catalog/cloud-without-visible-areas/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-catalog/missing-registry/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/missing-registry/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-catalog/missing-manifest/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/missing-manifest/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-catalog/manifest-without-scope/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/manifest-without-scope/stdout
53c234e5e8472b6ac51c1ae1cab3fe06fad053beb8ebfd8977b010655bfdd3c3  testdata/cases/1b-1-source/brain-catalog/invalid-channel/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/invalid-channel/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-read/whole-file/exit
be67ee13c4144a5f62259758e3a1bcb33d7bf2e2ec822f179a39797bfbe952b3  testdata/cases/1b-1-source/brain-read/whole-file/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-read/section/exit
90ddf36a8adb28efd55b9103ce7adfe6f432eb815ad34e132c356ed43e5dc0f9  testdata/cases/1b-1-source/brain-read/section/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-read/windows-separator/exit
c6479380217bc73d49ba7c1d091baff1e809697121b92120afa68d47bfc08c1b  testdata/cases/1b-1-source/brain-read/windows-separator/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-read/review-centre-local/exit
14d2d308fe3d2883a1154b4ea69764f1bbe3579065f808956c17945f7a55ccfc  testdata/cases/1b-1-source/brain-read/review-centre-local/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-read/review-centre-cloud/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/review-centre-cloud/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-read/never/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/never/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-read/leaves-area/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/leaves-area/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-read/missing-section/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/missing-section/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-read/missing-file/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/missing-file/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-read/read-only-area/exit
882674ea43279169a995ca93bf2415d2a616b1c391512e566a0507d52eba5c22  testdata/cases/1b-1-source/brain-read/read-only-area/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-read/local-only-on-cloud/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/local-only-on-cloud/stdout
53c234e5e8472b6ac51c1ae1cab3fe06fad053beb8ebfd8977b010655bfdd3c3  testdata/cases/1b-1-source/brain-read/missing-scope/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/missing-scope/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-neighbors/both-directions/exit
a4ed4c1bb5c9150bc1b3b10f279878a58b44081441ddfaf86bf4b156c738eff1  testdata/cases/1b-1-source/brain-neighbors/both-directions/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-neighbors/no-links/exit
28c8b213c7bdb816e60d9e00e2b69599d46b820bf2a9ec992d583eee4dbdb204  testdata/cases/1b-1-source/brain-neighbors/no-links/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-neighbors/read-only-area/exit
32cfc37180117966cd31b6f86fd6c0086cf2660e3a46295c730e990fe98d25a5  testdata/cases/1b-1-source/brain-neighbors/read-only-area/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-neighbors/never-indexed/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-neighbors/never-indexed/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-neighbors/leaves-area/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-neighbors/leaves-area/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-neighbors/unknown-scope/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-neighbors/unknown-scope/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-neighbors/broken-graph/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-neighbors/broken-graph/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/stamp-missing/exit
ab0b9b3592d573b2c92e50870570cc202335167ee5d7f473c39b2aa9632bdb56  testdata/cases/1b-1-source/brain-status/stamp-missing/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/stamp-stale/exit
cb741c392a479a53ddb7153e13f7efc6450728050036925e69aa6acfeb96d3cd  testdata/cases/1b-1-source/brain-status/stamp-stale/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/stamp-fresh/exit
0a98682338c67a7fb2bf88c5e7060152562c3a96d7ec97b674a07b21f031f153  testdata/cases/1b-1-source/brain-status/stamp-fresh/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/include-globs/exit
aaa87370ee6896dd16cebf5e210c88cc94d7c0e62a92ca3453134ebee52aeea8  testdata/cases/1b-1-source/brain-status/include-globs/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/path-missing/exit
1762f01fb32bbc4bdbdd396b49750802e6d121ae5397589b04ee6439257217e4  testdata/cases/1b-1-source/brain-status/path-missing/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/never-indexed/exit
168e518397d71e984f5b214e3c7e2a6b971e81e7bad06e09cca3ce6d5f134583  testdata/cases/1b-1-source/brain-status/never-indexed/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/links-unresolved/exit
7732d895fe089e9d3ff3d4973e1bab9fd6545c329fdb72388711641f9c2d584d  testdata/cases/1b-1-source/brain-status/links-unresolved/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/unfindable/exit
4a465a17f81fdf41642854eaca3f33c4cc52726130c83accf9f30f0758c69112  testdata/cases/1b-1-source/brain-status/unfindable/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/listing-fails/exit
6df899044296009eb12da53d23752bdd743272f469d301dad7f213340cc0efc6  testdata/cases/1b-1-source/brain-status/listing-fails/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/hash-shared/exit
865d9154513545616deb0a769e47ec7275c2a0ee0ac2a7cc240eeeeb09c1443c  testdata/cases/1b-1-source/brain-status/hash-shared/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/hash-withheld/exit
ed75737374af685f0a8c34aff13743dd0d53199db5b50762db7512308c76b382  testdata/cases/1b-1-source/brain-status/hash-withheld/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/backlog/exit
419f3d2ef6f32ced72387f26802b800b291e599c26f920400e07281373f9dc8a  testdata/cases/1b-1-source/brain-status/backlog/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/backlog-fails/exit
2c235c37fda68cde00917de93fc550e02b3ea349586231e80f759261c6043e35  testdata/cases/1b-1-source/brain-status/backlog-fails/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/vault-local/exit
664fd88b52f6e8f879217ba2ced3448c7fc50a27e18605e4e0bc9b258428562c  testdata/cases/1b-1-source/brain-status/vault-local/stdout
9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa  testdata/cases/1b-1-source/brain-status/vault-cloud/exit
c735af0ac23dab5edcd54101639b19180fcd13b4087f2d7b650f69572bb119bc  testdata/cases/1b-1-source/brain-status/vault-cloud/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-status/broken-graph/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-status/broken-graph/stdout
53c234e5e8472b6ac51c1ae1cab3fe06fad053beb8ebfd8977b010655bfdd3c3  testdata/cases/1b-1-source/brain-search/missing-query/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-search/missing-query/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-search/missing-registry/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-search/missing-registry/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-search/engine-fails/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-search/engine-fails/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-catalog/broken-registry/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/broken-registry/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-catalog/manifest-wrong-type/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-catalog/manifest-wrong-type/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-read/unknown-scope/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/unknown-scope/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-read/missing-manifest/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-read/missing-manifest/stdout
53c234e5e8472b6ac51c1ae1cab3fe06fad053beb8ebfd8977b010655bfdd3c3  testdata/cases/1b-1-source/brain-neighbors/missing-scope/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-neighbors/missing-scope/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-neighbors/manifest-without-scope/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-neighbors/manifest-without-scope/stdout
53c234e5e8472b6ac51c1ae1cab3fe06fad053beb8ebfd8977b010655bfdd3c3  testdata/cases/1b-1-source/brain-status/invalid-channel/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-status/invalid-channel/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-status/missing-manifest/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-status/missing-manifest/stdout
4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865  testdata/cases/1b-1-source/brain-status/broken-register/exit
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  testdata/cases/1b-1-source/brain-status/broken-register/stdout
EOF
```
Expected: Exit 0, keine Ausgabe (142 Zeilen, je Fall `exit` und `stdout`).
```bash
sha256sum -c --quiet "$TEMP/task12-recorded.sha256"
```
Expected: keine Ausgabe, Exit 0. Jede Zeile, die auf `: FAILED` endet, ist ein Befund an der Aufzeichnung, nicht an loomux: die Datei mit dem Text unten vergleichen. Typische Ursachen: das echte qmd statt des Fakes gelaufen (Treffer oder Listungen fremder Collections, `status` mit Zeilen über echte Bereiche), `BRAIN_STATE_DIR` nicht gesetzt (Bereiche aus `%LOCALAPPDATA%\brain`), `bin/loomux.exe` älter als Task 11 (CRLF im stdout). Antwortet der gebaute Fake anders als die Nachbildung, auf der die Prüfsummen beruhen (etwa eine andere Listung oder Trefferzahl), ist das ein Befund an Task 11, keine neue Aufzeichnung: melden, nicht weiter. Sonst die Ursache beheben und den Fall **neu aufzeichnen** (`rm -r testdata/cases/1b-1-source/<verb>/<fall>`, dann der Befehl aus Step 6); eine Aufzeichnung wird nie von Hand geändert.

```bash
find testdata/cases/1b-1-source -name cmd | wc -l
```
Expected: `71`.
```bash
grep -rl "AppData" testdata/cases/1b-1-source --include=stdout
```
Expected: keine Ausgabe, Exit 1 — `brain-status/path-missing` druckt den Pfad mit Backslashes, und `cases.Normalize` hat ihn zu `{{WORLD}}\sources\gone` gemacht. Steht dort ein Maschinenpfad, ist das ein Befund an `cases.Normalize`, keine Handkorrektur.
```bash
find testdata/cases/1b-1-source -name world_after
```
Expected: keine Ausgabe (kein Befehl schreibt in die Welt).
```bash
git -C "C:/Users/micro/Documents/#GIT/loomux-src/ub" status --short
```
Expected: keine Ausgabe.

Erwartete stdout, gemessen (jede Datei endet mit dem letzten gezeigten Zeilenumbruch; `brain-catalog/cloud-without-visible-areas` ist `# brain`, Leerzeile):

`brain-search/full-hits` (Exit 0)
```text
brain://project/alpha/notes/a.md:3  75%  Alpha notes
    The first finding.
    It spans two lines.

brain://notes/guide.md:5  50%  Guide
    How to read the guide.

brain://project/alpha/notes/unregistered.md:1  25%  Unregistered

brain://project/beta/b.md:2  10%  Beta
    line one
    line two

```

`brain-search/full-cut-at-n` (Exit 0)
```text
brain://project/alpha/notes/a.md:3  75%  Alpha notes
    The first finding.
    It spans two lines.

```

`brain-search/full-one-scope` (Exit 0)
```text
brain://notes/guide.md:5  50%  Guide
    How to read the guide.

```

`brain-search/full-no-matches` (Exit 0)
```text
no matches
```

`brain-search/full-cloud` (Exit 0)
```text
brain://project/alpha/notes/a.md:3  75%  Alpha notes
    The first finding.
    It spans two lines.

brain://notes/guide.md:5  50%  Guide
    How to read the guide.

brain://project/alpha/notes/unregistered.md:1  25%  Unregistered

```

`brain-search/cloud-without-visible-areas` (Exit 0)
```text
no matches
```

`brain-search/fast` (Exit 0)
```text
brain://project/alpha/notes/a.md:3  75%  Alpha notes
    The first finding.
    It spans two lines.

brain://notes/guide.md:5  50%  Guide
    How to read the guide.

brain://project/alpha/notes/unregistered.md:1  25%  Unregistered

brain://project/beta/b.md:2  10%  Beta
    line one
    line two

```

`brain-search/keyword` (Exit 0)
```text
brain://project/alpha/notes/a.md:3  75%  Alpha notes
    The first finding.
    It spans two lines.

brain://notes/guide.md:5  50%  Guide
    How to read the guide.

brain://project/alpha/notes/unregistered.md:1  25%  Unregistered

brain://project/beta/b.md:2  10%  Beta
    line one
    line two

```

`brain-catalog/root` (Exit 0)
```text
# brain

* [notes](brain://notes/)
* [project/alpha](brain://project/alpha/)
* [project/beta](brain://project/beta/)
* [project/gamma](brain://project/gamma/)
```

`brain-catalog/root-cloud` (Exit 0)
```text
# brain

* [notes](brain://notes/)
* [project/alpha](brain://project/alpha/)
* [project/gamma](brain://project/gamma/)
```

`brain-catalog/area` (Exit 0)
```text
# project/alpha

* [Alpha notes](brain://project/alpha/notes/a.md)
* [B](brain://project/alpha/notes/b.md)
```

`brain-catalog/read-only-area` (Exit 0)
```text
# notes

* [Guide](brain://notes/guide.md)
```

`brain-catalog/cloud-without-visible-areas` (Exit 0)
```text
# brain

```

`brain-read/whole-file` (Exit 0)
```text
# Alpha notes
The first finding.
It spans two lines.

## Second part
Details of the second part.

### Deeper
Still inside.

## Third part
Not in the section.
```

`brain-read/section` (Exit 0)
```text
## Second part
Details of the second part.

### Deeper
Still inside.
```

`brain-read/windows-separator` (Exit 0)
```text
### Deeper
Still inside.
```

`brain-read/review-centre-local` (Exit 0)
```text
# Case 1

A pending review.
```

`brain-read/read-only-area` (Exit 0)
```text
# Guide

How to read the guide.

See [start](start.md).
```

`brain-neighbors/both-directions` (Exit 0)
```text
incoming: notes/b.md, review/case.md
outgoing: notes/b.md
```

`brain-neighbors/no-links` (Exit 0)
```text
incoming: -
outgoing: -
```

`brain-neighbors/read-only-area` (Exit 0)
```text
incoming: -
outgoing: start.md
```

`brain-status/stamp-missing` (Exit 0)
```text
last reconcile: never; run `brain reconcile`
```

`brain-status/stamp-stale` (Exit 0)
```text
last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`
```

`brain-status/stamp-fresh` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
```

`brain-status/include-globs` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
project/a: the search engine sees only docs/**/*.md; also declared: README*.md, notes/*.md
```

`brain-status/path-missing` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
notes: {{WORLD}}\sources\gone does not exist; skipped
```

`brain-status/never-indexed` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
project/a: never indexed; run `brain reindex`
```

`brain-status/links-unresolved` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
project/a: only 2 of 5 links resolved (external=1, unknown_target=2)
project/b: only 1 of 3 links resolved ()
```

`brain-status/unfindable` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
project/a: 4 of 6 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. b.md, c.md, d.md, …
```

`brain-status/listing-fails` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
project/a: the search engine did not answer (qmd exited with 1: Collection not found: project-a
Run 'qmd ls' to see available collections.); its index was not compared
```

`brain-status/hash-shared` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
same content hash under 2 paths: project/a/zeta-copy.md, project/b/zeta.md
same content hash under 3 paths: project/a/copy.md, project/a/shared.md, project/b/shared.md
```

`brain-status/hash-withheld` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
project/b/key.md: same content hash as a path excluded by [privacy] never
```

`brain-status/backlog` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
7 documents are indexed but not yet searchable; run `brain embed`
```

`brain-status/backlog-fails` (Exit 0)
```text
last reconcile: 2999-01-01T00:00:00+00:00
the search engine did not answer (qmd exited with 1: index is locked); the backlog was not counted
```

`brain-status/vault-local` (Exit 0)
```text
last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`
project/beta: the search engine sees only *.md; also declared: docs/*.md
project/gamma: never indexed; run `brain reindex`
```

`brain-status/vault-cloud` (Exit 0)
```text
last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`
project/gamma: never indexed; run `brain reindex`
```

Ohne stdout: `brain-search/unknown-scope`, `brain-search/broken-register`, `brain-search/count-below-one`, `brain-search/invalid-profile`, `brain-catalog/local-only-on-cloud`, `brain-catalog/unknown-scope`, `brain-catalog/missing-index`, `brain-catalog/missing-registry`, `brain-catalog/missing-manifest`, `brain-catalog/manifest-without-scope`, `brain-catalog/invalid-channel`, `brain-read/review-centre-cloud`, `brain-read/never`, `brain-read/leaves-area`, `brain-read/missing-section`, `brain-read/missing-file`, `brain-read/local-only-on-cloud`, `brain-read/missing-scope`, `brain-neighbors/never-indexed`, `brain-neighbors/leaves-area`, `brain-neighbors/unknown-scope`, `brain-neighbors/broken-graph`, `brain-status/broken-graph`, `brain-search/missing-query`, `brain-search/missing-registry`, `brain-search/engine-fails`, `brain-catalog/broken-registry`, `brain-catalog/manifest-wrong-type`, `brain-read/unknown-scope`, `brain-read/missing-manifest`, `brain-neighbors/missing-scope`, `brain-neighbors/manifest-without-scope`, `brain-status/invalid-channel`, `brain-status/missing-manifest`, `brain-status/broken-register`. Das sind 35 Fälle:
```bash
find testdata/cases/1b-1-source -name stdout -empty | wc -l
```
Expected: `35`.

- [ ] **Step 8: Übersetzungstabelle** — `testdata/cases/1b-1-map.toml`

```toml
# Stage 1b-1 translates one command prefix. brain-mcp is the console script
# of the Python reference; its five read commands live under loomux brain.

[[command]]
from = "brain-mcp "
to   = "loomux brain "
```

- [ ] **Step 9: Importieren und die Übersetzung prüfen**

```bash
go run ./cmd/loomux dev import-cases --map testdata/cases/1b-1-map.toml --from testdata/cases/1b-1-source --to testdata/cases/1b-1
```
Expected: Exit 0, keine Ausgabe. Bricht der Import an der Welt `broken-registry` ab, behandelt `registeredDirs` die unlesbare Registry noch als Fehler, statt sie der Wiedergabe zu überlassen (Ruling R6): Befund an Task 11, nicht weiter.
```bash
cat testdata/cases/1b-1/*/*/cmd | cut -d' ' -f1,2 | sort | uniq -c
```
Expected: `     71 loomux brain`.
```bash
find testdata/cases/1b-1 -path '*/.loomux/config.toml' | wc -l
```
Expected: `198` — jedes Altmanifest jeder kopierten Welt ist gefaltet (`vault` 4 × 42 Fälle, `local-only` 1 × 2, `manifest-without-scope` 1 × 2, `damaged` 2 × 3, `manifest-wrong-type` 1, `engine-fails` 1, die `status-*`-Welten zusammen 18; `no-registry`, `missing-manifest` und `broken-registry` tragen keins). Weniger heißt: `TranslateWorld` faltet die Registry-Verzeichnisse nicht (Task 11 Step 12).
```bash
find testdata/cases/1b-1 -name .brain.toml -o -path '*/.ultra-brain/*'
```
Expected: keine Ausgabe.
```bash
git status --short testdata/cases/1a-source testdata/cases/1a
```
Expected: keine Ausgabe.

- [ ] **Step 10: Bestehen sehen**

```bash
go test ./internal/cli/ -run TestRecordedCasesOfStage1b1 -count=1 -v
```
Expected: `--- PASS: TestRecordedCasesOfStage1b1` mit 71 Untertests `--- PASS`, am Ende `ok  	github.com/xidus90/loomux/internal/cli`. Keine der Abweichungen aus den Berichten der Tasks 1–11 ändert Exit oder stdout eines dieser Fälle: wo loomux anders spricht (fehlende oder kaputte Registry, fehlendes Manifest, Manifest mit falschem Typ, fehlendes `index.md`, fehlende Datei, Graph- und Registerfehler, die scheiternde Suche), ist es stderr bei gleichem Exit 1, und stderr vergleicht der Korpus nicht. Zwei Voraussetzungen aus den Rulings: Der Fake nummeriert die Snippet-Zeilen des MCP-`query` wie qmd (R5), und Task 8 entfernt genau dieses Präfix (R4). Fehlt die Entfernung, sind `brain-search/full-hits`, `brain-search/full-cut-at-n`, `brain-search/full-one-scope` und `brain-search/full-cloud` rot, mit `N: ` vor jeder Snippet-Zeile im tatsächlichen stdout. `brain-search/engine-fails` endet in loomux nach `search.ColdAttempts` Versuchen ohne Wartezeit, weil jede neue Sitzung dieselbe JSON-RPC-Fehlerantwort bekommt.

- [ ] **Step 11: Rot lesen, nicht wegdrücken**

Jeder rote Fall ist entweder ein Fehler in loomux oder eine gewollte Abweichung. Die Suite nennt Exit, Bytezahlen und das tatsächliche stdout (hinter `stdout: ` im `%q`-Format); die Erwartung steht in `testdata/cases/1b-1/<verb>/<fall>/stdout`, ihr Text zusätzlich in Step 7.
- **Fehler in loomux:** im zuständigen Paket ein Test mit dem Wortlaut der Aufzeichnung, also mit Exit und stdout aus `testdata/cases/1b-1-source/<verb>/<fall>/`; `go test ./internal/<paket>/ -run <Testname> -count=1` rot sehen; reparieren; derselbe Befehl grün. Dann `go test ./internal/<paket>/ -count=1 -coverprofile="$TEMP/task12-repair.out"` und `go tool cover -func="$TEMP/task12-repair.out"` — Erwartet: jede Funktion des Pakets 100.0%. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund. Danach Step 10 wiederholen. Die Reparatur geht in den Commit dieses Tasks (Step 15 und 16). Zuständig: `privacy` für Sichtbarkeit und Verweigerungen (Task 3), `graph` (Task 5), `reader` (Task 6), `catalog` (Task 7), `search` (Task 8), `status` (Task 9), `cli` für Formen, Exit und Ausgabe (Task 10), `fakeqmd` für das, was der Fake anders antwortet als beim Aufzeichnen (Task 11).
- **Gewollte Abweichung:** die Erwartung in `testdata/cases/1b-1/<verb>/<fall>/` anpassen und **im selben Commit** in `docs/.superpowers/parity/stufe-1b-1.md` die Tabellenzeile `| <verb>/<fall> | <Exit und stdout der Aufzeichnung> | <Exit und stdout von loomux> | Task 12, Step 11: <Ursache> | |` eintragen, den letzten Satz des Abschnitts „Was die Fälle der Stufe 1b-1 decken" durch `Angepasst wurde die Erwartung von <verb>/<fall>.` ersetzen und in `testdata/cases/README.md` die Regel „A translated case may deviate" um den Fall ergänzen, so wie sie `hook-pre-tool-use/unreadable-payload` nennt. `1b-1-source/` wird nie angepasst.
- Scheitert ein Fall mit `does not call loomux`, fehlt die Regel aus Step 8; mit einer `stat`-Meldung über `<fallverzeichnis>\world` oder mit `missing world directory`, fehlt die Welt in der Aufzeichnung — beides neu importieren, nichts von Hand.

- [ ] **Step 12: Paket und Format**

```bash
gofmt -l internal/cli
```
Expected: keine Ausgabe.
```bash
go vet ./internal/cli/
```
Expected: keine Ausgabe.
```bash
go test ./internal/cli/ -count=1
```
Expected: `ok  	github.com/xidus90/loomux/internal/cli`; `TestRecordedCasesOfStage1a` bleibt grün. Coverage: der Task fügt nur eine Testdatei hinzu, keine Funktion außerhalb von `_test.go` — `covergate` im Tor sieht nichts Neues.

- [ ] **Step 13: Paritätsliste** — `docs/.superpowers/parity/stufe-1b-1.md`

„Alt" ist `brain-mcp` der Python-Referenz, „Neu" ist `loomux brain`. Die Zeilen sind die sechs der Spec und alle, die die Berichte der Tasks 1–11 nennen; wo ein Task eine Spec-Zeile wiederholt oder mehrere Tasks denselben Unterschied melden, steht er einmal, mit allen Quellen in „Begründung". Die Freigabespalte bleibt leer: Freigaben gehören dem Menschen.

````markdown
# Paritätsliste Stufe 1b-1

**Quelle:** ultra-brain `loomux-1a-source` (`3cc72d2`): Python-Referenz `src/brain/` unter Python 3.14.7 (`brain-mcp`), qmd 2.8.3 für den Spike vom 2026-09-15.
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor Stufe 1b-1 als fertig gilt. „Alt" ist die Python-Referenz, „Neu" ist `loomux brain`.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| `search --profile fast`, Rangfolge (Fall `brain-search/fast`) | `qmd vsearch`: ein LLM erweitert die Anfrage, Suche je Variante, Zusammenführung per Maximum, Schnitt unter 0,3 | qmd-MCP-Daemon mit `searches:[{type:"vec"}]`, `rerank:false`, ohne Erweiterung; Score 1/Rang | Spec-Zeile 1, Task 8: Spike 2026-09-15, `qmd query "vec: <q>" --no-rerank` liefert die MCP-Liste; der Fall vergleicht nur den Exit | |
| `search --profile keyword` (Fall `brain-search/keyword`) | BM25-Score, erste Fundstelle | gleiche Reihenfolge, Score 1/Rang, Zeile des besten Abschnitts | Spec-Zeile 2, Task 8; der Fall vergleicht nur den Exit | |
| Zustand und Manifestnamen, befristet | Registry, Artefakte schreibgeschützter Bereiche und Stempel unter `BRAIN_STATE_DIR` bzw. `--state-dir`; Manifest `.ultra-brain/config.toml`, sonst `.brain.toml` | Registry unter `LOOMUX_STATE_DIR` (`%LOCALAPPDATA%\loomux`); Artefakte und Stempel unter `LOOMUX_LEGACY_BRAIN_DIR` (sonst `%LOCALAPPDATA%\brain`) bis Stufe 3; `.loomux/config.toml` vor beiden Altnamen bis Stufe 4 — steht sie mit einer `[area]`-Tabelle neben einem Altmanifest, gilt sie; ohne `[area]` deklariert sie für `brain` nichts, und das Altmanifest gilt | Spec-Zeile 3, befristete Ausnahme; Task 1 (1, 2) | |
| Aufzeichnungen der Stufe 1b-1 | Python schreibt in eine Pipe `\r\n`, ohne `PYTHONUTF8` in cp1252 | der Rekorder faltet `\r\n` im stdout zu `\n` und setzt `PYTHONUTF8=1` in jedem aufgezeichneten Prozess; aufgezeichnet mit `uv run --no-sync` und `PYTHONDONTWRITEBYTECODE=1` | Spec-Zeile 4, Task 11 (1): ein Unterschied der Aufzeichnung, nicht des Verhaltens. Die Faltung gilt ab Task 11 für jede Aufzeichnung; die Zeile „CRLF in Aufzeichnungen" der Stufe 1a beschreibt den Rekorder davor | |
| Aufwärm-Hinweis | kein Hinweis | `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` auf stderr, einmal je Port, wenn loomux den Daemon selbst gestartet hat | Spec-Zeile 5, Task 8 (3) | |
| Backbone beim Daemon-Start | `qmd_mcp.py` schreibt `{**os.environ, **env}` und überschreibt `QMD_LLAMA_GPU`/`QMD_FORCE_CPU` des Nutzers | eine vom Nutzer gesetzte Variable, auch leer, gewinnt; sonst CUDA-Vorgabe ohne Variable | Spec-Zeile 6, Task 8 (4) | |
| CLI-Port (`qmd ls`, `qmd status`), Umgebung | `_default_runner` pinnt `QMD_LLAMA_GPU=vulkan` | Umgebung des Nutzers, CUDA-Vorgabe, nichts angehängt | Task 8 (5) | |
| Wartefrist auf den Daemon, Wortlaut | `… within 60s` | `no qmd daemon answered on http://localhost:8765/mcp within 1m0s` | Task 8 (6) | |
| Parser der CLI-Suche (`QmdPort.Search`, in 1b-1 ungenutzt) | `expected a list of hits, found dict`, `the hit is missing its 'file': {…}`; verlangt ganzzahliges `line` und eine Zahl als `score` | ub-Go-Wortlaute `expected a list of hits`, `the hit is missing 'file'`; fehlendes `line`/`score` wird 0 | Task 8 (7) | |
| MCP-Antwortübersetzung | `qmd_mcp.py`: `col/` ohne Rest behält den ganzen Pfad; negative Zeilen bleiben | `col/` wird (`col`, `""`); eine Zeile unter 1 wird 1 | Task 8 (8); im Korpus unsichtbar, die Python-CLI nutzt den MCP-Port nie | |
| Unlesbarer npm-Shim | `cannot run qmd: [Errno …] …` | `cannot read {shim}: …` | Task 8 (9) | |
| `Pending`-Zeile von `qmd status` | Pythons `re` liest `\s` und `\d` als Unicode-Klassen | `pendingLinePattern` mit ASCII-`\s`/`\d` (ub-Go) | Task 8 (11) | |
| Registerpfad mit `..` in Fehlermeldungen | `Path` behält `..` | `filepath.Join` löst `..` auf | Task 8 (12) | |
| `n < 1` an `ExecuteSearch` | `ordered[:n]` schneidet von hinten | leere Liste | Task 8 (13); die CLI verweigert `-n` unter 1 vorher (Fall `brain-search/count-below-one`) | |
| Betriebssystemfehler im Wortlaut (Fälle `brain-catalog/missing-registry`, `brain-search/missing-registry`, `brain-catalog/missing-index`, `brain-read/missing-file`) | `[Errno 2] No such file or directory: '<pfad>'` bzw. `[Errno 13] Permission denied: '<pfad>'`, Pfad in `repr` | Go-Wortlaut von `os.ReadFile` (`open <pfad>: …`, Systemtext in der Sprache von Windows), bei der Registry `<pfad>: open <pfad>: …` | Task 3 (3), Task 4 (4), Task 5 (8), Task 6 (2, 3), Task 7 (1), Task 8 (10), Task 9. Betroffen: fehlende Registry, fehlendes `index.md`, fehlende Datei bei `read`, `read` von `.` oder leerem Pfad (Python liest das Verzeichnis), `_identities.tsv`, `graph.json` oder Stempel, die existieren und nicht lesbar sind. Beide Exit 1 | |
| Fehlendes Manifest eines registrierten Bereichs (Fälle `brain-catalog/missing-manifest`, `brain-read/missing-manifest`, `brain-status/missing-manifest`) | `[Errno 2] No such file or directory: '<bereich>\\.brain.toml'` | `<bereich>: no manifest found (.loomux\config.toml, .ultra-brain\config.toml, .brain.toml)` | Task 3 (1), Wortlaut aus Task 1; beide Exit 1 und brechen den ganzen Aufruf ab | |
| `[layout] review` ohne Pfadteile (`.`, `./`, `./.`) | `ValueError`, von `cli.main` nicht gefangen: Traceback, Exit 1 | `error: [layout] review must not resolve to the area root, found '.'`, Exit 1 | Task 3 (2) | |
| Registry-Prüfungen | `RegistryError` bei doppeltem Scope, fehlendem `scope`/`path`, Scope ohne brauchbare Zeichen, gleichem Zustandsverzeichnis zweier Scopes, zwei `signpost` | `config.ReadRegistry` überspringt Einträge ohne `scope`/`path` und prüft den Rest nicht; bei doppeltem Scope liefert `VisibleAreas` beide, `Single` den ersten | Task 3 (4) | |
| Manifest mit BOM | `not valid TOML: Invalid statement (at line 1, column 1)` | gelesen | Task 1 (3) | |
| `[area] scope` als Nicht-Zeichenkette (`scope = 3`, Fall `brain-catalog/manifest-wrong-type`) | `[area] scope is required and must be a non-empty string` | `not valid TOML: toml: line 2 (last key "area.scope"): incompatible types: TOML value has type int64; destination has type string` | Task 1 (4) | |
| Wortlaut nach `not valid TOML: ` bei einem kaputten Manifest oder einer kaputten Registry (Fall `brain-catalog/broken-registry`) | der von `tomllib`, für `[[area` gemessen `Expected ']]' at the end of an array declaration (at line 1, column 7)` | der von BurntSushi toml v1.6.0 | Task 1 (5); die Registry liest `config.ReadRegistry` mit derselben Form `<pfad>: not valid TOML: <grund>` (`internal/config/registry.go:106`); beide Exit 1 | |
| Weitere Prüfungen von `read_manifest` | `[wiki] types` als Liste von Zeichenketten, `[wiki] untouched_days` ≥ 1, `[maintenance] on_merge`/`branch`, `[model]`; Glob-Listen `must be an array of strings` bzw. `contains a non-string` | nicht geprüft; Glob-Listen mit falschem Typ scheitern als TOML-Fehler | Task 1 (7) | |
| Wortlaut der Modusmeldung | `found 'bogus'` (`repr`) | `found "bogus"` (Go-`%q`) | Task 1 (8): `config` darf `brain/pytext` nicht importieren | |
| `.brain.toml` als Verzeichnis, kein anderer Name | beim Lesen `[Errno 13] Permission denied` | `no manifest found` | Task 1 (9) | |
| Leeres `XDG_STATE_HOME` (POSIX) | relativer Pfad `brain` | `home/.local/state/brain` | Task 1 (2) | |
| Ungültiges UTF-8 in einer gelesenen Datei | Traceback (`UnicodeDecodeError`) | `error: {pfad}: not valid UTF-8` für `_identities.tsv`, `graph.json`, `index.md` und Dokumente; beim Manifest `{pfad}: not valid TOML: toml: … invalid UTF-8 byte: 0xff` | Task 2 (3), Task 4 (3), Task 5 (9), Task 1 (6); Task 6 und 7 führen die Zeile mit. Der Stempel ist ausgenommen: `read_last_run` fängt den `UnicodeDecodeError` (eine `ValueError`), und `ReadLastRun` antwortet ebenso „kein Stempel" (Task 8) | |
| `repr` von Zeichen, die Unicode 17 neu vergibt (4803 Codepunkte, 47 Bereiche, darunter U+088F) | `\uXXXX`/`\UXXXXXXXX` (Python 3.14 kennt Unicode 16) | unverändert | Task 2 (1) | |
| `casefold` von 28 Unicode-17-Buchstaben (U+A7CE, U+A7D2, U+A7D4, U+16EA0…U+16EB8) | nicht gefaltet | gefaltet; betrifft `[privacy] never`-Globs mit diesen Zeichen | Task 2 (2) | |
| `Repr` eines Bytes, das kein UTF-8 ist | — (Python kennt keinen solchen `str`) | `\xhh` | Task 2 (4); nur über Argumente oder Dateinamen erreichbar, die nicht aus `ReadText` stammen | |
| Stempel-Formen außerhalb der gemessenen Teilmenge | `datetime.fromisoformat` liest 14 Formen zonenbehaftet: kleines `t` oder ein anderes Trennzeichen als `T`/Leerzeichen, `HH:MM` und `HH`, Grundform `20000101T000000+0000`, Offset `+0200` und `+02`, Bruch im Offset, Wochendatum `2000-W01-1`, `24:00:00`, Offset-Minute oder -Sekunde 60, Leerzeichen vor dem Offset | „kein Stempel": `status` meldet `last reconcile: never; …`, `search` keinen Stempelbefund | Task 2 (5) | |
| `IsoFormat` mit Nanosekunden | — | schneidet unter der Mikrosekunde ab | Task 2 (6); über einen Stempel nicht erreichbar | |
| Revision aus Nicht-ASCII-Ziffern | `٣` wird 3; `²` endet im Traceback (`ValueError`) | `{pfad}: line {n}: revision '…' is not a number`, Exit 1 | Task 4 (1) | |
| Revision jenseits von `int` (`99999999999999999999`) | gelesen | verweigert mit demselben Wortlaut | Task 4 (2) | |
| Wahrheitswert als Zählwert in `links` (`"total": true`) | angenommen (`bool` ist `int`) | `the link counts are not numbers` | Task 5 (1) | |
| Nicht-Zeichenketten in `from`/`to` | per `str()` gewandelt (`1` → `1`, `null` → `None`) | `{pfad}: json: cannot unmarshal … of type string`; `null` wird leere Zeichenkette | Task 5 (2) | |
| Wortlaut des JSON-Fehlers in Klammern | `json.JSONDecodeError` (`Expecting value: line 1 column 1 (char 0)`, `Extra data: …`) | `encoding/json` (`invalid character 'o' in literal null (expecting 'u')`) | Task 5 (3) | |
| `NaN`, `Infinity`, `-Infinity` in `graph.json` | gelesen; als Zählwert `the link counts are not numbers` | `graph is not valid JSON (invalid character 'N' …)` | Task 5 (4) | |
| Wurzelwert von `graph.json`, der kein Objekt ist | `[]` und `"x"` wie loomux; `"edges"` → `graph is missing links`; `["edges","links"]`, eine Zahl oder `null` → Traceback (`TypeError`) | immer `graph is missing edges, links` | Task 5 (5) | |
| Ganzzahl jenseits von `int` als Zählwert | angenommen | `{pfad}: json: cannot unmarshal number 99999999999999999999 into Go struct field … of type int` | Task 5 (6) | |
| `nodes` oder `scope` von falschem Typ | nie gelesen | `{pfad}: json: cannot unmarshal …` (ub-Verhalten) | Task 5 (7) | |
| `read --section ""` | sucht eine leere Überschrift (`_section('# \nx', '')` → `'# \nx\n'`) | gibt die ganze Datei aus | Task 6 (1) | |
| Abkürzungen langer Optionen | `--prof` wird `--profile`; `--s` → `ambiguous option: --s could match --scope, --state-dir` | nur ganze Namen: `unrecognized arguments: --prof x` bzw. `--s x`, Exit 2 | Task 10 (1) | |
| `--state-dir` | an allen fünf Befehlen | `unrecognized arguments: --state-dir …`, Exit 2; der Zustand kommt aus `LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR` | Task 10 (2) | |
| `-h`/`--help` | Hilfe auf stdout, Exit 0, auch angeklebt (`search q -hx`) | keine Hilfe; das Wort endet bei `unrecognized arguments` oder hinter dem Fehler über fehlende Argumente, Exit 2 | Task 10 (3) | |
| `-n` mit Nicht-ASCII-Ziffern (`٣`, `３`) | 3 | `invalid _at_least_one value`, Exit 2 | Task 10 (4) | |
| `-n` über `math.MaxInt` (`99999999999999999999999`) | die Zahl bleibt | auf `math.MaxInt` gekappt | Task 10 (5) | |
| Usage-Zeilen | nach Terminalbreite umgebrochen, mit `[-h]` und `[--state-dir STATE_DIR]`; die oberste Usage und `argument command: invalid choice` zählen 23 Unterbefehle | eine Zeile ohne beide Optionen, fünf Unterbefehle | Task 10 (6); die Fälle `brain-search/count-below-one`, `brain-search/invalid-profile`, `brain-search/missing-query`, `brain-catalog/invalid-channel`, `brain-read/missing-scope`, `brain-neighbors/missing-scope`, `brain-status/invalid-channel` vergleichen Exit und leeres stdout | |
| Wörter vor dem Unterbefehl | `brain-mcp -- search q` angenommen; `brain-mcp -x search q` → `unrecognized arguments: -x` | das erste Wort ist der Unterbefehl: `argument command: invalid choice: '--'` bzw. `'-x'`, Exit 2 | Task 10 (7) | |
| Brain-Daemon | ein laufender brain-Daemon beantwortet die fünf Befehle in seinem eigenen Format (`cli.py` `_through_daemon`) | fragt nie einen brain-Daemon | Task 10 (8); aufgezeichnet ohne Daemon, dessen Pipe-Name vom Zustandsverzeichnis abhängt | |
| qmd in den Aufzeichnungen | echtes qmd 2.8.3 | `internal/dev/fakeqmd`: `qmd ls` mit fester Größe und Zeit, `qmd status` nur `Documents` und die `Pending`-Zeile, Treffer synthetisch in Fixture-Reihenfolge; eine scheiternde Suche ist die Störung `search_error` (CLI Exit 1 mit dem Wert auf stderr, MCP JSON-RPC-Fehler `-32603`), kein gemessener qmd-Wortlaut | Task 11 (2); kein Modell und keine GPU in Aufzeichnung und Tor; `search_error` trägt den Fall `brain-search/engine-fails` | |
| Snippets auf beiden Wegen | CLI `--json` liefert den Snippet ohne Zeilennummern | echtes qmd 2.8.3 stellt im MCP-`query` jeder Snippet-Zeile `N: ` voran (`dist/mcp/server.js:301`), bei gleichem Ausschnittfenster (`server.js:293` und `cli/qmd.js:2140`); `translateReply` entfernt genau dieses Präfix, stdout ist darum gleich | Task 11 (3), berichtigt nach F69; Task 8 (Ruling R4). Der Fake nummeriert auf dem MCP-Weg wie qmd; `TestLoomuxsPortsReadTheFake` und die `full`-Fälle des Korpus belegen die Entfernung, der Live-Vergleich von Task 15 bestätigt sie am echten qmd | |
| Befunde von `search` und alle Fehlerwortlaute | auf stderr | auf stderr, eigener Wortlaut, wo die Zeilen oben es sagen | Spec, Vergleichsklassen: der Korpus vergleicht stderr nie; die exakten Zeichenketten stehen in den Unit-Tests der Tasks 3–10 | |

## Was die Fälle der Stufe 1b-1 decken

`testdata/cases/1b-1-source/` hält die Aufzeichnungen von `brain-mcp` gegen den
Fake-qmd (Beweis, nie nachbearbeitet), `testdata/cases/1b-1/` die Übersetzung,
an der loomux gemessen wird. 71 Fälle über 23 Welten, gefahren von
`internal/cli/cases_1b1_test.go`: `search` 15 (davon 2 nur Exit), `catalog` 14,
`read` 14, `neighbors` 9, `status` 19 mit einer Welt je Zeilenart L1a–L8b.
Jeder Befehl hat seinen Erfolg, jede Verweigerung, einen Usage-Fehler und die
Registry- und Manifestfehler; dazu die kaputte Registry, das Manifest mit
falschem Typ und die scheiternde Suche.
Angepasst wurde keine Erwartung.
````

Findet Step 11 eine gewollte Abweichung, bekommt die Tabelle eine Zeile in der Form aus Step 11, und der letzte Satz des Abschnitts lautet `Angepasst wurde die Erwartung von <verb>/<fall>.`

- [ ] **Step 14: `testdata/cases/README.md` nachziehen** — die Datei ganz ersetzen:

````markdown
# The case corpora

The cases here are the parity evidence of loomux against the tools it
replaces: recordings of the old tools, translated into loomux's own command
lines and replayed in process. Each stage keeps its own worlds, recordings,
translation table and suite.

| Stage | Recorded from | Suite | Cases |
|---|---|---|---:|
| 1a | `ulguard` and `ulinit` from ultraloom, `brain` from ultra-brain, both at the tag `loomux-1a-source` | `internal/cli/cases_test.go` | 19 |
| 1b-1 | `brain-mcp`, the Python reference of ultra-brain at the tag `loomux-1a-source` (`3cc72d2`), against a fake qmd | `internal/cli/cases_1b1_test.go` | 71 |

## Layout

| Path | What it holds |
|---|---|
| `1a-worlds/` | The staged project trees a case runs in: `project-writable`, `guard-allow-writable`, `guard-deny-readonly`, `plain`. Each recording copies the whole world into its own `world/`. |
| `1a-payloads/` | The hook payloads fed to a run on stdin. `{{WORLD}}` in a payload is replaced by the staged world directory. |
| `1a-source/` | The recordings of the old binaries, written by `loomux dev record-case`. `verb/name/` holds `cmd`, `exit`, `stdout`, `notes.md`, the staged `world/`, and optionally `stdin` and `compare`. |
| `1a-map.toml` | The translation table: `[[command]]` rules rewriting the head of a recorded command line (`ulguard post-edit --root …` → `loomux hook post-tool-use --host claude --root …`). Longer prefixes stand before shorter ones. |
| `1a/` | The translated cases, written by `loomux dev import-cases` from `1a-source/` and `1a-map.toml`. This is the directory the test suite runs. The old tools' configuration files (`.brain.toml`, `.ultraloom/policy.toml`, `.ultra-brain/config.toml`) are folded into one `.loomux/config.toml` in every staged world. |
| `1b-1-worlds/` | The state directories a 1b-1 case runs in. A world is the reference's `BRAIN_STATE_DIR` and, in the replay, both `LOOMUX_STATE_DIR` and `LOOMUX_LEGACY_BRAIN_DIR`: `registry.toml` with `{{WORLD}}/<path>` paths, writable areas under `repo-*` with `.brain.toml` or `.ultra-brain/config.toml`, the artefacts of read-only areas under `areas/<flat scope>/`, the reconcile stamp under `maintenance/last-run.txt` (year 2000 is stale, year 2999 fresh), and `qmd-fixture.json` for the fake qmd. `vault` serves search, catalog, read and neighbors; each `status-*` world shows one status line kind or, in `status-broken-register`, the register error that aborts status; `no-registry`, `broken-registry`, `missing-manifest`, `manifest-without-scope`, `manifest-wrong-type`, `damaged` and `engine-fails` each carry one broken input. |
| `1b-1-source/` | The recordings of `brain-mcp`, written by `loomux dev record-case --argv "uv run --no-sync --project <ub> brain-mcp"` with the fake qmd first on `PATH`. |
| `1b-1-map.toml` | One rule: `brain-mcp ` → `loomux brain `. |
| `1b-1/` | The translated cases. Besides the root and `areas/<name>`, the old manifests are folded in every directory the world's registry names as `{{WORLD}}/<path>`. A registry the import cannot read names no directory; the replay reports it. |

## What a case compares

`compare` in a case directory selects the comparison; a missing file means
`data`.

- **`compare = data`** — the exit code **and** the recorded `stdout`, byte for
  byte.
- **`compare = message`** — the exit code **alone**. The wording of a message
  is loomux's own: it is allowed to differ from the old tool's.

Independently of `compare`, a case that carries a `world_after/` directory also
has the staged tree compared against it after the run. A recording only gets a
`world_after/` when the run actually changed the world; no stage-1a and no
stage-1b-1 case does.

In 1b-1 every case compares data except `brain-search/fast` and
`brain-search/keyword`: there the reference runs qmd's command line and loomux
the MCP daemon, and the two rank differently. A usage or runtime error compares
data too — its stdout is empty on both sides, and that emptiness is part of the
contract. stderr is never compared; the findings of `search` and every error
wording are pinned by unit tests.

## The fake qmd (1b-1)

No recording and no replay starts qmd. `internal/dev/fakeqmd` answers from the
world's `qmd-fixture.json`: built as `qmd.exe` and put first on `PATH`, it
serves the reference's `qmd ls`, `qmd status` and `qmd query|vsearch|search
--json`; in the suite, the same fixture serves loomux's MCP port over
`httptest` and its command line port through the `Runner` seam. A world
without a fixture has an engine that knows nothing. A fixture with
`search_error` fails every search: `search`, `vsearch` and `query` exit 1 with
that stderr, and the MCP `query` answers with a JSON-RPC error
(`brain-search/engine-fails`). The MCP `query` numbers snippet lines as qmd
does (`N: `); loomux removes that prefix, so the recorded stdout still holds.
A daemon that cannot be started is covered by unit tests, not by a case: no
recording may start qmd.

## A recorded `stdout` is evidence, not always an expectation

The `stdout` of a `message` case is kept as it was recorded — it still names
`.brain.toml`, absolute machine paths and the old tools' wording. That text is
the record of what the old binary printed on that run, not a claim about what
loomux prints. Nothing compares it.

A 1b-1 recording folds the `\r\n` Python writes into a pipe to `\n` and runs
Python with `PYTHONUTF8=1`; nothing else in its stdout is changed.

## Rules for working with the corpus

- **A recording is evidence.** Files under `1a-source/` and `1b-1-source/` are
  never edited by hand. If a case is wrong, it is *re-recorded*, never patched:
  for 1a with the old binaries (build them from the tag worktrees, put them
  first on `PATH`, run `loomux dev record-case`); for 1b-1 with the fake qmd
  rebuilt from `internal/dev/fakeqmd/qmd` and the recording command of the
  stage plan.
- **A translated case may deviate from its recording only where the parity
  list says so.** Every such deviation has a line in
  `docs/.superpowers/parity/stufe-1a.md` or `stufe-1b-1.md`. Today there is
  one, in 1a: the exit of `hook-pre-tool-use/unreadable-payload` is 2 (the
  guard fails closed) where the old tool gave 1. 1b-1 has none.
- **A re-import throws that deviation away, and it has to be re-applied by
  hand.** The deviation lives only in `1a/`; `1a-source/` still holds the
  recorded 1, and `loomux dev import-cases` copies the recording over the
  translated case. After every re-import, set
  `1a/hook-pre-tool-use/unreadable-payload/exit` back to `2`. Nothing is silent
  about it: until it is set, `internal/cli/cases_test.go` fails on that case
  with `exit code: expected 1, got 2`.
- **An import removes what no recording backs.** `loomux dev import-cases`
  prunes every case in the target that the recordings no longer hold (and the
  verb directory that loses its last case), so a case dropped from the
  recordings cannot survive behind an unchanged case count.
- **An import reads every world's manifests.** `TranslateWorld` decodes the
  old manifests of each staged world; a manifest that is not valid TOML stops
  the import, so such a manifest is no case. `brain-catalog/manifest-wrong-type`
  records valid TOML of the wrong type instead. A registry the import cannot
  read names no directory to fold and is left to the replay
  (`brain-catalog/broken-registry`).
- **The case count is pinned.** `internal/cli/cases_test.go` fails when the 1a
  corpus does not hold exactly 19 cases, `internal/cli/cases_1b1_test.go` when
  the 1b-1 corpus does not hold exactly 71, so a partial import cannot pass as
  parity. Adding a case means raising that number.
````

- [ ] **Step 15: Stagen und Tor**

```bash
git add internal/cli/cases_1b1_test.go testdata/cases/1b-1-worlds testdata/cases/1b-1-source testdata/cases/1b-1-map.toml testdata/cases/1b-1 testdata/cases/README.md docs/.superpowers/parity/stufe-1b-1.md
```
Expected: Exit 0. Hat Step 11 ein Paket repariert, stehen dessen Dateien mit in diesem `git add`, und die Dateizahlen unten wachsen um genau diese Dateien.
```bash
sh .githooks/pre-commit
```
Expected: Exit 0 (das Tor verweigert ungestagte oder ungetrackte `testdata`, darum zuerst `git add`).
```bash
git branch --show-current
```
Expected: `sdd-1b-1` (Hauptcheckout: `master`).
```bash
git rev-parse --short HEAD
```
Expected: der Hash des Commits von Task 11.
```bash
git diff --cached --stat -- internal docs testdata/cases/README.md testdata/cases/1b-1-map.toml
```
Expected: je eine Zeile für `docs/.superpowers/parity/stufe-1b-1.md`, `internal/cli/cases_1b1_test.go`, `testdata/cases/1b-1-map.toml` und `testdata/cases/README.md`, dann die Summenzeile mit `4 files changed`. Die Zeilenzahlen hängen am Text aus Step 13 und 14; die Pfade selbst prüft:
```bash
git diff --cached --name-only -- internal docs testdata/cases/README.md testdata/cases/1b-1-map.toml
```
Expected, genau:
```
docs/.superpowers/parity/stufe-1b-1.md
internal/cli/cases_1b1_test.go
testdata/cases/1b-1-map.toml
testdata/cases/README.md
```
```bash
git diff --cached --name-only -- testdata/cases/1b-1-worlds testdata/cases/1b-1-source testdata/cases/1b-1 | cut -d/ -f3 | sort | uniq -c
```
Expected (aus Welten und Fallzeilen gerechnet: je Fall `cmd`, `exit`, `stdout`, `notes.md`, zweimal `compare`, dazu die kopierte Welt; die Übersetzung ersetzt jedes Altmanifest durch genau eine `.loomux/config.toml`):
```
   1379 1b-1
   1379 1b-1-source
    130 1b-1-worlds
```
```bash
git diff --cached --name-only | wc -l
```
Expected: `2892` (4 + 1379 + 1379 + 130): nichts anderes ist gestagt. Nach einer Reparatur aus Step 11 kommen deren Dateien hinzu: im `git add` mit aufgeführt, in der Pfadliste oben und in dieser Zahl mitgezählt.

- [ ] **Step 16: Commit**

```bash
printf '%s\n' 'Record brain-mcp against a fake qmd and hold loomux brain to it' '' '71 cases over 23 worlds: every status line kind, full search as data,' 'fast and keyword search as exit codes, and every refusal, usage error' 'and runtime error the reference reports for each command, among them' 'a broken registry and a failing search engine.' 'The parity list collects the deviations the stage has reported so far.' > "$TEMP/loomux-task12-msg.txt"
```
Expected: Exit 0, keine Ausgabe.
```bash
git commit -F "$TEMP/loomux-task12-msg.txt"
```
Expected: Exit 0. Der Commit löst das Tor noch einmal aus; das ist gewollt.
```bash
git log -1 --format='%an <%ae>'
```
Expected: die Nutzeridentität, dieselbe wie `git config user.name` und `git config user.email`.
```bash
git log -1 --format=%B
```
Expected: der Text der Nachrichtendatei (Titel, Leerzeile, fünf Zeilen), keine `Co-Authored-By`-Zeile.

- [ ] **Step 17: Bericht**

Paritätszeilen, die dieser Task schafft (in Step 13 schon eingetragen, ohne Freigabe):
1. Befunde und Fehlerwortlaute auf stderr: vom Korpus nie verglichen, exakt in den Unit-Tests.

Nicht mehr in der Liste: die beiden Zeilen „kein Fall" für `search` ohne erreichbares qmd und für eine Registry oder ein Manifest, das kein TOML ist. Beide Fehlerwege sind Fälle (`brain-search/engine-fails`, `brain-catalog/broken-registry`, `brain-catalog/manifest-wrong-type`; Rulings R6, R7). Die Zeile „Snippets auf beiden Wegen" (Task 11 (3)) ist nach F69 berichtigt: qmd 2.8.3 unterscheidet auf dem `query`-Weg nur das `N: `-Präfix, nicht das Ausschnittfenster; Task 8 entfernt das Präfix, stdout ist gleich (Rulings R4, R5).

Befunde:
- Zwei Zusätze zur Aufzeichnungsvorlage des Vertrags, beide für „Quellen nur lesen": `uv run --no-sync` und `--env PYTHONDONTWRITEBYTECODE=1`.
- Die Suite nennt bei einem roten Fall das tatsächliche stdout (`%q`); die 1a-Suite nennt nur Bytezahlen.
- Der failing test aus Step 2 scheitert in Step 3 an fehlenden Daten, nicht an fehlender Implementierung: `loomux brain` steht seit Task 10, dieser Task liefert den Korpus. Das Rot belegt, dass die Suite ohne Korpus nicht grün wird; die festgenagelte Fallzahl belegt, dass ein Teilimport nicht grün wird.
- Zwölf Fälle und vier Welten mehr als im Vertrag (Rulings R6, R7, R8): Registry- und Manifestfehler je Befehl, ein Usage-Fehler je Befehl, `read` mit unbekanntem Scope, `status` mit kaputtem Register (`status-broken-register`), die kaputte Registry (`broken-registry`), das Manifest mit falschem Typ (`manifest-wrong-type`) und die scheiternde Suche (`engine-fails`). Sie hängen an Task 11 nach den Rulings R5 und R6 (`search_error`, `registeredDirs`); Step 1 und Step 9 prüfen beides.
- `t.Chdir(dir)` steht nicht in der Suite (Ruling R19): `loomux brain` liest das Arbeitsverzeichnis nicht, und ein Arbeitsverzeichnis in der gestagten Welt hielte unter Windows `RunCase`s `os.RemoveAll` auf, sodass je Fall ein `case-run-*`-Verzeichnis liegen bliebe.
- Der Bericht von Task 2 zählt den Stempel zu den Dateien, bei denen Python an ungültigem UTF-8 mit Traceback abbricht. `read_last_run` fängt den `UnicodeDecodeError` (`reconcile.py:1026-1043`, `1b-1-facts/status.md` §3), und `ReadLastRun` (Task 8) antwortet ebenso „kein Stempel"; die Paritätszeile nimmt den Stempel darum aus.
- Die 1a-Zeile „CRLF in Aufzeichnungen" (freigegeben) beschreibt den Rekorder vor Task 11; sie bleibt unverändert, die 1b-1-Zeile nennt den neuen Stand.
- Fälle, die in Step 11 rot waren, mit Ursache und Reparatur oder Paritätszeile.
- Die Ausgabe von Step 7 (`sha256sum -c`) und Step 10.

Mensch-Schritt, im Bericht vorgelegt (Ruling R20): `testdata/cases/1b-1-source/**` ist Beweis wie `testdata/cases/1a-source/**`, aber nur `1a-source` hat eine Pfadregel in `.loomux/config.toml`. Ein Agent schreibt diese Datei nie. Der Bericht schlägt dem Menschen vor, nach dem Commit dieses Tasks den Block mit `match  = ["testdata/cases/1a-source/**"]` durch diesen zu ersetzen; `reason` bleibt wortgleich:
```toml
[[policy.paths.rules]]
match  = ["testdata/cases/1a-source/**", "testdata/cases/1b-1-source/**"]
reason = "Recordings of the old tools are evidence; re-record them, never edit them."
```

Vorab gemessene stderr der Referenz für die 35 Fehlerfälle, je die letzte Zeile (nicht verglichen, zum Lesen roter Fälle; gemessen wie in der Messgrundlage, der Pfad der gestagten Welt als `{{WORLD}}`, in den `[Errno 2]`-Meldungen mit den verdoppelten Rückstrichen von Pythons `repr`):
```text
brain-search/unknown-scope: error: unknown scope 'nope'; known scopes are: notes, project/alpha, project/beta, project/gamma
brain-search/broken-register: error: {{WORLD}}\repo-garbled\_identities.tsv: line 2: expected 4 tab-separated fields, found 3
brain-search/count-below-one: brain-mcp search: error: argument -n: must be at least 1, got 0
brain-search/invalid-profile: brain-mcp search: error: argument --profile: invalid choice: 'deep' (choose from 'fast', 'full', 'keyword')
brain-catalog/local-only-on-cloud: error: unknown scope 'project/beta'; known scopes are: notes, project/alpha, project/gamma
brain-catalog/unknown-scope: error: unknown scope 'nope'; known scopes are: notes, project/alpha, project/beta, project/gamma
brain-catalog/missing-index: error: [Errno 2] No such file or directory: '{{WORLD}}\\repo-gamma\\index.md'
brain-catalog/missing-registry: error: [Errno 2] No such file or directory: '{{WORLD}}\\registry.toml'
brain-catalog/missing-manifest: error: [Errno 2] No such file or directory: '{{WORLD}}\\repo-bare\\.brain.toml'
brain-catalog/manifest-without-scope: error: {{WORLD}}\repo-vague\.brain.toml: [area] scope is required and must be a non-empty string
brain-catalog/invalid-channel: brain-mcp catalog: error: argument --channel: invalid choice: 'public' (choose from 'local', 'cloud')
brain-read/review-centre-cloud: error: project/alpha/review/case.md is the review centre; refused on the cloud channel
brain-read/never: error: project/alpha/secret/key.md is excluded by [privacy] never
brain-read/leaves-area: error: project/alpha/../repo-beta/b.md leaves the area
brain-read/missing-section: error: no section titled 'Nowhere'
brain-read/missing-file: error: [Errno 2] No such file or directory: '{{WORLD}}\\repo-alpha\\notes\\gone.md'
brain-read/local-only-on-cloud: error: unknown scope 'project/beta'; known scopes are: notes, project/alpha, project/gamma
brain-read/missing-scope: brain-mcp read: error: the following arguments are required: --scope
brain-neighbors/never-indexed: error: project/gamma: never indexed; run `brain reindex`
brain-neighbors/leaves-area: error: project/alpha/../repo-beta/b.md leaves the area
brain-neighbors/unknown-scope: error: unknown scope 'nope'; known scopes are: notes, project/alpha, project/beta, project/gamma
brain-neighbors/broken-graph: error: {{WORLD}}\repo-torn\graph.json: graph is missing links; delete it and run `brain reindex`
brain-status/broken-graph: error: {{WORLD}}\repo-torn\graph.json: graph is missing links; delete it and run `brain reindex`
brain-search/missing-query: brain-mcp search: error: the following arguments are required: query
brain-search/missing-registry: error: [Errno 2] No such file or directory: '{{WORLD}}\\registry.toml'
brain-search/engine-fails: error: qmd exited with 1: index is locked
brain-catalog/broken-registry: error: {{WORLD}}\registry.toml: not valid TOML: Expected ']]' at the end of an array declaration (at line 1, column 7)
brain-catalog/manifest-wrong-type: error: {{WORLD}}\repo-typed\.brain.toml: [area] scope is required and must be a non-empty string
brain-read/unknown-scope: error: unknown scope 'nope'; known scopes are: notes, project/alpha, project/beta, project/gamma
brain-read/missing-manifest: error: [Errno 2] No such file or directory: '{{WORLD}}\\repo-bare\\.brain.toml'
brain-neighbors/missing-scope: brain-mcp neighbors: error: the following arguments are required: --scope
brain-neighbors/manifest-without-scope: error: {{WORLD}}\repo-vague\.brain.toml: [area] scope is required and must be a non-empty string
brain-status/invalid-channel: brain-mcp status: error: argument --channel: invalid choice: 'public' (choose from 'local', 'cloud')
brain-status/missing-manifest: error: [Errno 2] No such file or directory: '{{WORLD}}\\repo-bare\\.brain.toml'
brain-status/broken-register: error: {{WORLD}}\repo-a\_identities.tsv: line 2: expected 4 tab-separated fields, found 3
```
Die Usage-Fehler tragen davor die Usage-Zeilen von argparse, die Laufzeitfehler nur diese eine Zeile.

---

### Task 13: `loomux dev mutants`

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 12 sind committet.

**Files:**
- Create: `internal/dev/mutants/mutants.go`, `internal/dev/mutants/generate.go`, `internal/dev/mutants/round.go`
- Test: `internal/dev/mutants/generate_test.go`, `internal/dev/mutants/mutants_test.go`, `internal/dev/mutants/round_test.go`, `internal/dev/mutants/gotest_test.go`
- Modify: `internal/cli/dev.go`, `internal/cli/dev_test.go` — im Stand **nach Task 11** (Importblock mit `"strings"` und `internal/cases`, `devRecordCase` mit `--argv`)
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (§9, AGENTS.md: Doku in beiden Sprachen gleich)

Keine `testdata`-Datei: der Fixture-Quelltext steht als Go-Konstante in `generate_test.go` (Tabs und Zeilenenden überleben so jede Kopie, und das Tor findet keine ungetrackte `testdata`).

**Interfaces:**
- Consumes: Task 2 `pytext.Strip(s string) string`, `pytext.SplitLines(s string) []string` — Signaturen wie in den Interfaces von `task-02.md`. `internal/dev/*` darf `internal/brain/pytext` benutzen (Ruling R27, Global Constraints). `run(args ...string) (int, string, string)` aus `internal/cli/cli_test.go`. `devCommands`, `command` aus `internal/cli/dev.go`/`commands.go`.
- Produces (Bezeichner wie im Vertrag; Wortlaut von `ErrBaselineRed` und `GoTest` ohne Ausnahme nach den Abweichungen 1 und 3 unten):
  ```go
  package mutants
  type Mutant struct { Family string; Path string; Line int; Was, Now string }
  func Generate(path string, source []byte) []Mutant // Familien a1–a4 nach ub/tools/go_mutants.py
  type Outcome int
  const ( Passed Outcome = iota; Failed; BuildFailed; TimedOut )
  type TestFunc func(pkg, overlay string) (Outcome, error) // overlay "" = unverändert
  func GoTest(root string) TestFunc                        // go test [-overlay <json>] -count=1 -failfast -timeout 60s ./<pkg>/; kein //coverage:exempt
  type Options struct { Packages []string; Root string; Only string; Family string; Workers int }
  type Summary struct { Killed, Survived, NotCompiled, NoMutant int; Survivors []Mutant }
  var ErrBaselineRed = errors.New("the suite is not green before the round")
  func Round(opts Options, test TestFunc, w io.Writer) (Summary, error)
  ```
- Produces (Ergänzungen):
  ```go
  func (m Mutant) String() string             // Mutant.__str__ des Skripts: "(a3) fixture.go.txt:5  if a > 0 {  ->  if a >= 0 {"
  var ErrNoSources = errors.New("no source files") // Wortlaut go_mutants.py:286
  func DefaultWorkers() int                   // max(runtime.NumCPU()/2, 1)
  // unexportiert: ifLine, comparisons, splitTop, columns, span, lineSpans, lineBreak, mutate,
  //               goTimeout, patience, classify, sources, finished, roundOf, runOne, try, overlayJSON
  // internal/cli: var mutantsTest = mutants.GoTest; var mutantsRoot = os.Getwd;
  //               var mutantsNotify = signal.NotifyContext;
  //               func devMutants(args []string, _ io.Reader, stdout, stderr io.Writer) int
  //               func untilInterrupted(ctx context.Context, test mutants.TestFunc) mutants.TestFunc
  ```

**Abweichungen vom Vertrag** (Quelle bzw. Messung geht vor; im Bericht und an Task 14 gemeldet):
1. **`ErrBaselineRed`-Wortlaut:** `errors.New("the suite is not green before the round")` — wörtlich `go_mutants.py:291`, nach der Vertragsregel „Wortlaut des Skripts, wo es einen hat“. Der Bezeichner bleibt. Task 14 übernimmt diesen Wortlaut (Ruling R3).
2. **Kein Urteil `not compiled`:** Das Skript kennt je Mutant drei Wörter; `no mutant` *bedeutet* dort „kompiliert nicht“ (`go_mutants.py:312-314`). Dieser Task druckt `no mutant` für `BuildFailed` (gezählt in `Summary.NotCompiled`, Summenzeile `N do not compile and are no mutants`) und ebenfalls `no mutant` für einen Mutanten, dessen Zeile gleich bleibt (`if true {` → `if true {`; gezählt in `Summary.NoMutant`, nie gestartet, Summenzeile `N change nothing and are no mutants` nur bei N > 0). Das Wort `not compiled` erscheint nirgends; Task 14 liest die beiden Zahlen aus den Summenzeilen (Ruling R3).
3. **Kein `//coverage:exempt` an `GoTest`:** `gotest_test.go` startet den echten `go`-Befehl gegen ein Probemodul in `t.TempDir()` und einmal mit leerem `PATH`; `GoTest` steht damit bei 100,0 % (gemessen). Der Vertrag erlaubt die Ausnahme nur, er verlangt sie nicht.
4. **`--only DATEI`** ist wie im Skript eine Teilzeichenkette des Dateinamens (`arguments.only in path.name`), kein exakter Name.
5. **Exit 2 auch für `ErrNoSources`** — das Skript endet dort mit 2 (`go_mutants.py:285-287`).
6. **Summenzeile** `{n} mutants over {paket}, oracle go` wörtlich wie `go_mutants.py:323-324`; loomux kennt nur das Orakel `go`.
7. **`internal/dev/mutants` importiert `internal/brain/pytext`** (`Strip`; `SplitLines` im Zwillingstest). Die Abhängigkeitsregel des Vertrags nennt `dev/*` nicht; Ruling R27 ergänzt sie um genau diese Kante.
8. **Abbruch:** Die Spec verlangt, dass die temporären Overlay-Dateien auch bei Abbruch aufgeräumt werden; der Vertrag nennt nur „aufgeräumt auch bei Fehler“. Ein Ctrl+C beendet einen Go-Prozess, ohne dass ein `defer` läuft. `devMutants` fängt `os.Interrupt` darum über `signal.NotifyContext(context.Background(), os.Interrupt)` (Ruling R9), und `untilInterrupted` macht aus dem Abbruch den Fehler eines Laufs: Die Runde endet über ihren Fehlerweg mit Exit 1, und jedes `defer os.RemoveAll` in `try` läuft (Steps 13 bis 15).

**Vorab gemessen** (Python 3.14.7 der Referenz; `go_mutants.py` nie ausgeführt, nur per `importlib` geladen, `main` nicht gerufen). Die gemessene `fixture.go.txt` lag im Scratchpad des Planers; `fixtureSource` ist daraus Zeile für Zeile mit `strconv.Quote` erzeugt und damit byte-gleich. Der Befehl unten protokolliert die Messung; ihre ausführbare Form im Repo ist `TestGenerateMatchesTheScriptOnTheFixture`.

```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "import sys,importlib.util as u; s=u.spec_from_file_location('gm','C:/Users/micro/Documents/#GIT/loomux-src/ub/tools/go_mutants.py'); m=u.module_from_spec(s); sys.modules['gm']=m; s.loader.exec_module(m); from pathlib import Path; from collections import Counter; ms=m._mutants_of(Path('fixture.go.txt')); print(len(ms), sorted(Counter(x.family for x in ms).items()))"
```
Erwartet: `52 [('a1', 14), ('a2', 12), ('a3', 19), ('a4', 7)]`, die erste Zeile der Tabelle.

| Ausdruck (Datei = Bytes von `fixtureSource` bzw. `breaks` unten) | Ergebnis |
|---|---|
| `_mutants_of(fixture.go.txt)` | `52 [('a1', 14), ('a2', 12), ('a3', 19), ('a4', 7)]`, Reihenfolge und `now` wie `fixtureSites` |
| `_mutants_of` derselben Datei mit CRLF | identisch, 52 |
| `_mutants_of(breaks.go.txt)` | 19 (`a1` 6, `a3` 10, `a4` 3), Liste wie in `TestGenerateNumbersLinesLikeTheScript` |
| `_split_top(c, op)` für die zwölf Fälle in `TestSplitTopFindsTheOperandsOfTheCondition` | die dort erwarteten Listen (`[]` → `nil`) |
| `_columns(line, op)` für die siebzehn Fälle in `TestColumnsFindTheOperatorAsAnOperator` | die dort erwarteten Spalten |
| `text.splitlines()` für die zwölf Texte in `TestLineSpansCutWhereSplitlinesCuts` | die dort erwarteten Listen |
| `str(Mutant('a3', Path('C:/tmp/x/fixture.go.txt'), 5, '\tif a > 0 {', '\tif a >= 0 {'))` | `'(a3) fixture.go.txt:5  if a > 0 {  ->  if a >= 0 {'` |

`go test` unter `-overlay` (go1.27.0 windows/amd64, Probemodul `example.com/probe`): ein Compilerfehler druckt als erste Zeile `# example.com/probe/p [example.com/probe/p.test]`, dann die Befundzeile und danach `FAIL	example.com/probe/p [build failed]`; ein Vet-Befund druckt als erste Zeilen `# example.com/probe/p` und `# [example.com/probe/p]`, dann die Befundzeile und ebenfalls `FAIL	example.com/probe/p [build failed]`. Die letzte Zeile ist in beiden Fällen `FAIL`, der Exit-Code 1; die Fälle `compile error` und `vet error` in `TestClassifyReadsWhatGoTestPrinted` geben beide Ausgaben Zeile für Zeile wieder. Eine volle Runde über das Probemodul (`gotest_test.go`) läuft in rund 2 s, das ganze Paket in 2,2 s bis 2,4 s (zwei Messungen).

- [ ] **Step 1: Failing tests — Generierung** — `internal/dev/mutants/generate_test.go` (ganze Datei):

```go
package mutants

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// fixtureSource is the source the expected mutants were measured on with the
// script's own _mutants_of (tools/go_mutants.py under the reference's Python
// 3.14.7): 52 mutants, a1 14, a2 12, a3 19, a4 7. It is not Go that compiles;
// it gathers the cases the script's rules tell apart.
const fixtureSource = "package fixture\n" +
	"\n" +
	"// if a == b {\n" +
	"func decide(a, b int, s, t string, ok bool, ch chan int) int {\n" +
	"\tif a > 0 {\n" +
	"\t\treturn 1\n" +
	"\t}\n" +
	"\tif a != b && s == \"x&&y\" || ok {\n" +
	"\t\treturn 2\n" +
	"\t}\n" +
	"\tif f(a < b && ok) || t != \"\" {\n" +
	"\t\treturn 3\n" +
	"\t} else if a >= b {\n" +
	"\t\treturn 4\n" +
	"\t}\n" +
	"\tif v, ok := any(s).(string); !ok {\n" +
	"\t\treturn len(v)\n" +
	"\t}\n" +
	"\tif r := s[0]; r == '\"' && t != \"\" {\n" +
	"\t\treturn 5\n" +
	"\t}\n" +
	"\tif s == \"\\\"&&\" || t == `a||b` {\n" +
	"\t\treturn 6\n" +
	"\t}\n" +
	"\tfor i := 0; i < b; i++ {\n" +
	"\t\ta <<= 1\n" +
	"\t}\n" +
	"\tx := a == b // a != b\n" +
	"\ty := \"a < b\" + s\n" +
	"\tz := <-ch\n" +
	"\tif a <= b { // trailing comment keeps the brace off the end\n" +
	"\t\treturn 7\n" +
	"\t}\n" +
	"\tif \"é\" < s && t >= \"ü\" {\n" +
	"\t\treturn 8\n" +
	"\t}\n" +
	"\tswitch {\n" +
	"\tcase a > b:\n" +
	"\t\treturn 9\n" +
	"\t}\n" +
	"\thtml := `\n" +
	"<p>\n" +
	"\t<b>`\n" +
	"\t_, _, _, _ = x, y, z, html\n" +
	"\treturn 0\n" +
	"}\n"

// site is what a mutant says, without the file it was read from.
type site struct {
	Family string
	Line   int
	Now    string
}

func sites(ms []Mutant) []site {
	out := make([]site, len(ms))
	for i, m := range ms {
		out[i] = site{m.Family, m.Line, m.Now}
	}
	return out
}

// fixtureSites is _mutants_of(fixture) in the script's order, measured.
var fixtureSites = []site{
	{"a1", 5, "\tif true {"},
	{"a1", 5, "\tif false {"},
	{"a4", 5, "\tif !(a > 0) {"},
	{"a3", 5, "\tif a >= 0 {"},
	{"a1", 8, "\tif true {"},
	{"a1", 8, "\tif false {"},
	{"a4", 8, "\tif !(a != b && s == \"x&&y\" || ok) {"},
	{"a2", 8, "\tif a != b {"},
	{"a2", 8, "\tif s == \"x&&y\" || ok {"},
	{"a2", 8, "\tif a != b && s == \"x&&y\" {"},
	{"a2", 8, "\tif ok {"},
	{"a3", 8, "\tif a != b && s != \"x&&y\" || ok {"},
	{"a3", 8, "\tif a == b && s == \"x&&y\" || ok {"},
	{"a1", 11, "\tif true {"},
	{"a1", 11, "\tif false {"},
	{"a4", 11, "\tif !(f(a < b && ok) || t != \"\") {"},
	{"a2", 11, "\tif f(a < b && ok) {"},
	{"a2", 11, "\tif t != \"\" {"},
	{"a3", 11, "\tif f(a < b && ok) || t == \"\" {"},
	{"a3", 11, "\tif f(a <= b && ok) || t != \"\" {"},
	{"a3", 13, "\t} else if a > b {"},
	{"a1", 16, "\tif true {"},
	{"a1", 16, "\tif false {"},
	{"a4", 16, "\tif !(v, ok := any(s).(string); !ok) {"},
	{"a1", 19, "\tif true {"},
	{"a1", 19, "\tif false {"},
	{"a4", 19, "\tif !(r := s[0]; r == '\"' && t != \"\") {"},
	{"a2", 19, "\tif r := s[0]; r == '\"' {"},
	{"a2", 19, "\tif t != \"\" {"},
	{"a3", 19, "\tif r := s[0]; r != '\"' && t != \"\" {"},
	{"a1", 22, "\tif true {"},
	{"a1", 22, "\tif false {"},
	{"a4", 22, "\tif !(s == \"\\\"&&\" || t == `a||b`) {"},
	{"a2", 22, "\tif s == \"\\\"&&\" {"},
	{"a2", 22, "\tif t == `a||b` {"},
	{"a3", 22, "\tif s != \"\\\"&&\" || t == `a||b` {"},
	{"a3", 25, "\tfor i := 0; i <= b; i++ {"},
	{"a3", 26, "\t\ta << 1"},
	{"a3", 26, "\t\ta <=<= 1"},
	{"a3", 28, "\tx := a != b // a != b"},
	{"a3", 30, "\tz := <=-ch"},
	{"a3", 31, "\tif a < b { // trailing comment keeps the brace off the end"},
	{"a1", 34, "\tif true {"},
	{"a1", 34, "\tif false {"},
	{"a4", 34, "\tif !(\"é\" < s && t >= \"ü\") {"},
	{"a2", 34, "\tif \"é\" < s {"},
	{"a2", 34, "\tif t >= \"ü\" {"},
	{"a3", 34, "\tif \"é\" < s && t > \"ü\" {"},
	{"a3", 34, "\tif \"é\" <= s && t >= \"ü\" {"},
	{"a3", 38, "\tcase a >= b:"},
	{"a3", 43, "\t<=b>`"},
	{"a3", 43, "\t<b>=`"},
}

func TestGenerateMatchesTheScriptOnTheFixture(t *testing.T) {
	ms := Generate("fixture.go.txt", []byte(fixtureSource))
	if got := sites(ms); !slices.Equal(got, fixtureSites) {
		t.Fatalf("got %d mutants:\n%q", len(got), got)
	}
	families := map[string]int{}
	lines := strings.Split(fixtureSource, "\n")
	for _, m := range ms {
		families[m.Family]++
		if m.Path != "fixture.go.txt" || m.Was != lines[m.Line-1] {
			t.Fatalf("%+v", m)
		}
	}
	if len(ms) != 52 || families["a1"] != 14 || families["a2"] != 12 || families["a3"] != 19 || families["a4"] != 7 {
		t.Fatalf("%d mutants, families %v", len(ms), families)
	}
}

func TestGenerateReadsCRLFAsTheScriptDoes(t *testing.T) {
	crlf := strings.ReplaceAll(fixtureSource, "\n", "\r\n")
	ms := Generate("fixture.go.txt", []byte(crlf))
	if got := sites(ms); !slices.Equal(got, fixtureSites) {
		t.Fatalf("got %d mutants:\n%q", len(got), got)
	}
	for _, m := range ms {
		if strings.ContainsRune(m.Was, '\r') {
			t.Fatalf("line %d keeps its CR: %q", m.Line, m.Was)
		}
	}
}

// breaks holds every line break Python's str.splitlines knows, and \x1f,
// which it does not know.
const breaks = "package t\n\tif a == 1 {\v\tif b == 2 {\f\tx := c < d\x1c\ty := e > f\x1d\tz := g != h\x1e\tw := i <= j\xc2\x85\tv := k >= l\xe2\x80\xa8\tu := m == n\xe2\x80\xa9\tif o {\r\ts := p < q\x1f\x1fr > s\r\n"

func TestGenerateNumbersLinesLikeTheScript(t *testing.T) {
	// _mutants_of on a file holding breaks, measured: 19 mutants.
	want := []site{
		{"a1", 2, "\tif true {"},
		{"a1", 2, "\tif false {"},
		{"a4", 2, "\tif !(a == 1) {"},
		{"a3", 2, "\tif a != 1 {"},
		{"a1", 3, "\tif true {"},
		{"a1", 3, "\tif false {"},
		{"a4", 3, "\tif !(b == 2) {"},
		{"a3", 3, "\tif b != 2 {"},
		{"a3", 4, "\tx := c <= d"},
		{"a3", 5, "\ty := e >= f"},
		{"a3", 6, "\tz := g == h"},
		{"a3", 7, "\tw := i < j"},
		{"a3", 8, "\tv := k > l"},
		{"a3", 9, "\tu := m != n"},
		{"a1", 10, "\tif true {"},
		{"a1", 10, "\tif false {"},
		{"a4", 10, "\tif !(o) {"},
		{"a3", 11, "\ts := p <= q\x1f\x1fr > s"},
		{"a3", 11, "\ts := p < q\x1f\x1fr >= s"},
	}
	if got := sites(Generate("breaks.go", []byte(breaks))); !slices.Equal(got, want) {
		t.Fatalf("got %d mutants:\n%q", len(got), got)
	}
}

func TestLineSpansCutWhereSplitlinesCuts(t *testing.T) {
	// Every want is str.splitlines() of the text, measured.
	for _, c := range []struct {
		text string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a\n", []string{"a"}},
		{"\n", []string{""}},
		{"a\r\nb", []string{"a", "b"}},
		{"a\rb", []string{"a", "b"}},
		{"a\n\nb", []string{"a", "", "b"}},
		{"a\vb\fc", []string{"a", "b", "c"}},
		{"a\x1cb\x1dc\x1ed\x1fe", []string{"a", "b", "c", "d\x1fe"}},
		{"a\xc2\x85b\xe2\x80\xa8c\xe2\x80\xa9d", []string{"a", "b", "c", "d"}},
		{"a\r\r\nb", []string{"a", "", "b"}},
		{"a\n\r", []string{"a", ""}},
	} {
		var got []string
		for _, s := range lineSpans([]byte(c.text)) {
			got = append(got, c.text[s.start:s.end])
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%q: got %q, want %q", c.text, got, c.want)
		}
	}
}

// lineSpans is a twin of pytext.SplitLines that keeps offsets; both must cut
// the same pieces.
func TestLineSpansCutWhatPytextSplitLinesCuts(t *testing.T) {
	for _, source := range []string{fixtureSource, breaks, "a\r\r\nb\n\r", ""} {
		var got []string
		for _, s := range lineSpans([]byte(source)) {
			got = append(got, source[s.start:s.end])
		}
		if want := pytext.SplitLines(source); !slices.Equal(got, want) {
			t.Errorf("%q: got %q, want %q", source, got, want)
		}
	}
}

func TestSplitTopFindsTheOperandsOfTheCondition(t *testing.T) {
	// Every want is _split_top(condition, operator), measured.
	for _, c := range []struct {
		condition, operator string
		want                []string
	}{
		{"a && b", "&&", []string{"a", "b"}},
		{"a || b", "&&", nil},
		{"f(a && b) && c", "&&", []string{"f(a && b)", "c"}},
		{`s == "x&&y" && t`, "&&", []string{`s == "x&&y"`, "t"}},
		{`s == "\"&&" && t`, "&&", []string{`s == "\"&&"`, "t"}},
		{"t == `a\\&&b` && u", "&&", []string{"t == `a\\&&b`", "u"}},
		{`r == '"' && t`, "&&", []string{`r == '"'`, "t"}},
		{"a && b || c && d", "||", []string{"a && b", "c && d"}},
		{"a &&", "&&", []string{"a", ""}},
		{") && (", "&&", nil},
		{"m[a && b] && c", "&&", []string{"m[a && b]", "c"}},
		{"x && y && z", "&&", []string{"x", "y", "z"}},
	} {
		if got := splitTop(c.condition, c.operator); !slices.Equal(got, c.want) {
			t.Errorf("%q %s: got %q, want %q", c.condition, c.operator, got, c.want)
		}
	}
}

func TestColumnsFindTheOperatorAsAnOperator(t *testing.T) {
	// Every want is _columns(line, operator), measured.
	for _, c := range []struct {
		line, operator string
		want           []int
	}{
		{"a < b", "<", []int{2}},
		{"a <= b", "<", nil},
		{"a <= b", "<=", []int{2}},
		{"<p>", "<", nil},
		{"<p>", ">", nil},
		{"\t<b>`", "<", []int{1}},
		{"\t<b>`", ">", []int{3}},
		{"x := a == b // a != b", "!=", nil},
		{`y := "a < b" + s`, "<", nil},
		{"a << b", "<", []int{2}},
		{"a >>= b", ">=", []int{3}},
		{"a >>= b", ">", []int{2}},
		{"a == b ==", "==", []int{2}},
		{"x <- ch", "<", []int{2}},
		{"a != b", "!=", []int{2}},
		{"a >= b", ">", nil},
		{`r == '"' && t != ""`, "!=", nil},
	} {
		if got := columns(c.line, c.operator); !slices.Equal(got, c.want) {
			t.Errorf("%q %s: got %v, want %v", c.line, c.operator, got, c.want)
		}
	}
}

func TestMutateReplacesOneLineAndKeepsItsBreak(t *testing.T) {
	crlf := strings.ReplaceAll(fixtureSource, "\n", "\r\n")
	m := Generate("fixture.go.txt", []byte(crlf))[3] // (a3) line 5: if a >= 0 {
	got := string(mutate([]byte(crlf), m))
	if want := strings.Replace(crlf, "\tif a > 0 {\r\n", "\tif a >= 0 {\r\n", 1); got != want {
		t.Fatalf("%q", got)
	}
	last := Mutant{Line: 2, Now: "\tif true {"}
	if got := string(mutate([]byte("x\n\tif a > 0 {"), last)); got != "x\n\tif true {" {
		t.Fatalf("%q", got)
	}
}
```

- [ ] **Step 2: Scheitern sehen**

```bash
go test ./internal/dev/mutants/
```
Erwartet (gemessen per Overlay über den Zustand „nur diese Testdatei“):
```
# github.com/xidus90/loomux/internal/dev/mutants [github.com/xidus90/loomux/internal/dev/mutants.test]
internal\dev\mutants\generate_test.go:69:17: undefined: Mutant
internal\dev\mutants\generate_test.go:134:8: undefined: Generate
internal\dev\mutants\generate_test.go:153:8: undefined: Generate
internal\dev\mutants\generate_test.go:191:18: undefined: Generate
internal\dev\mutants\generate_test.go:216:21: undefined: lineSpans
internal\dev\mutants\generate_test.go:230:21: undefined: lineSpans
internal\dev\mutants\generate_test.go:258:13: undefined: splitTop
internal\dev\mutants\generate_test.go:288:13: undefined: columns
internal\dev\mutants\generate_test.go:296:7: undefined: Generate
internal\dev\mutants\generate_test.go:297:16: undefined: mutate
internal\dev\mutants\generate_test.go:297:16: too many errors
FAIL	github.com/xidus90/loomux/internal/dev/mutants [build failed]
```

- [ ] **Step 3: Implementieren — Typ und Generierung**

`internal/dev/mutants/mutants.go` (erster Stand, Step 7 ersetzt ihn ganz):

```go
// Package mutants mutates the Go decisions of a package and reports which
// mutants its suite does not notice. It ports ultra-brain's
// tools/go_mutants.py with two changes of mechanism: a mutant reaches the
// suite through `go test -overlay` instead of being written into the tree,
// and several mutants run at once.
//
// Four families, named as the reports name them: (a1) the whole condition of
// an `if`, struck out as `true` and as `false`; (a2) each operand of a `&&`
// or `||` at the top level of the condition on its own; (a3) every
// comparison operator flipped; (a4) the condition negated, because a1 gives
// no signal at a `value, ok := x.(T)` guard.
//
// A mutant that does not compile is no mutant and is counted apart. A mutant
// the suite still passes has survived, and every survivor has to be read by
// hand: it is either a missing test or a place where the code cannot tell
// the difference.
package mutants

// Mutant is one change: which file, which line, and what it says instead.
type Mutant struct {
	Family   string
	Path     string
	Line     int
	Was, Now string
}
```

`internal/dev/mutants/generate.go` (ganze Datei):

```go
package mutants

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// ifLine is an `if` whose body opens on the same line. `for` is not mutated
// by a1, a2 or a4: its condition is a loop bound, and striking it out hangs a
// run instead of failing it. Go's \s is ASCII where Python's is Unicode;
// gofmt indents with tabs, so no formatted line tells the two apart.
var ifLine = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^(\s*)if (.+) \{$`)
})

// comparisons flips equality against its negation and each ordering against
// its neighbour, longest first, so that `<=` is never read as `<`.
var comparisons = [][2]string{
	{"==", "!="},
	{"!=", "=="},
	{"<=", "<"},
	{">=", ">"},
	{"<", "<="},
	{">", ">="},
}

// Generate lists the mutants of one source file in the script's order: per
// line a1 true, a1 false, a4 and the a2 operands of `&&` then `||`, then a3
// per operator of comparisons and per column.
func Generate(path string, source []byte) []Mutant {
	var found []Mutant
	for index, span := range lineSpans(source) {
		number := index + 1
		line := string(source[span.start:span.end])
		if match := ifLine().FindStringSubmatch(line); match != nil {
			indent, condition := match[1], match[2]
			for _, r := range [][2]string{{"a1", "true"}, {"a1", "false"}, {"a4", "!(" + condition + ")"}} {
				found = append(found, Mutant{Family: r[0], Path: path, Line: number, Was: line, Now: indent + "if " + r[1] + " {"})
			}
			for _, operator := range []string{"&&", "||"} {
				for _, half := range splitTop(condition, operator) {
					found = append(found, Mutant{Family: "a2", Path: path, Line: number, Was: line, Now: indent + "if " + half + " {"})
				}
			}
		}
		for _, c := range comparisons {
			for _, column := range columns(line, c[0]) {
				found = append(found, Mutant{Family: "a3", Path: path, Line: number, Was: line, Now: line[:column] + c[1] + line[column+len(c[0]):]})
			}
		}
	}
	return found
}

// splitTop returns the operands of one boolean operator at the top level of a
// condition, or nil where the operator does not stand there. Depth and quoting
// are tracked because a `&&` inside a call's arguments or inside a string is
// not a half of this condition. The walk is over bytes where the script walks
// code points; every character it compares is ASCII, and no byte of a
// multi-byte UTF-8 sequence is.
func splitTop(condition, operator string) []string {
	var parts []string
	depth := 0
	var quote byte
	start := 0
	for index := 0; index < len(condition); {
		char := condition[index]
		switch {
		case quote != 0:
			if char == '\\' && quote != '`' {
				index += 2
				continue
			}
			if char == quote {
				quote = 0
			}
		case char == '"' || char == '\'' || char == '`':
			quote = char
		case char == '(' || char == '[' || char == '{':
			depth++
		case char == ')' || char == ']' || char == '}':
			depth--
		case depth == 0 && strings.HasPrefix(condition[index:], operator):
			parts = append(parts, condition[start:index])
			start = index + 2
			index += 2
			continue
		}
		index++
	}
	if parts == nil {
		return nil
	}
	parts = append(parts, condition[start:])
	for i, part := range parts {
		parts[i] = pytext.Strip(part)
	}
	return parts
}

// columns lists where an operator stands in a line as an operator: not as
// part of a longer one, not behind `//`, not inside a string by the count of
// double quotes before it. The empty-string tests are the script's: in Python
// "" is in every string, so an operator at the very start or the very end of
// the line is never counted.
func columns(line, operator string) []int {
	stripped, _, _ := strings.Cut(line, "//")
	var places []int
	for index := 0; ; index += len(operator) {
		found := strings.Index(stripped[index:], operator)
		if found < 0 {
			return places
		}
		index += found
		end := index + len(operator)
		after := stripped[end:min(end+1, len(stripped))]
		before := stripped[max(index-1, 0):index]
		neighbours := strings.Contains("=", after) || (strings.Contains("<>", operator) && strings.Contains("<>=!", before))
		if !neighbours && strings.Count(stripped[:index], `"`)%2 == 0 {
			places = append(places, index)
		}
	}
}

// span is one line of a source without its line break.
type span struct{ start, end int }

// lineSpans cuts a source where Python's str.splitlines cuts what
// Path.read_text returns: at \r\n, \n, \r, \v, \f, \x1c, \x1d, \x1e, U+0085,
// U+2028 and U+2029, with no empty piece after a final break. Line numbers
// then name the line the script names.
func lineSpans(source []byte) []span {
	var spans []span
	start := 0
	for index := 0; index < len(source); {
		width := lineBreak(source[index:])
		if width == 0 {
			index++
			continue
		}
		spans = append(spans, span{start, index})
		index += width
		start = index
	}
	if start < len(source) {
		spans = append(spans, span{start, len(source)})
	}
	return spans
}

// lineBreak is the width of the line break head opens with, or 0. Every break
// is ASCII or opens with a UTF-8 lead byte, so a walk over bytes finds the
// breaks a walk over code points finds.
func lineBreak(head []byte) int {
	switch {
	case bytes.HasPrefix(head, []byte("\r\n")), bytes.HasPrefix(head, []byte("\xc2\x85")):
		return 2
	case bytes.HasPrefix(head, []byte("\xe2\x80\xa8")), bytes.HasPrefix(head, []byte("\xe2\x80\xa9")):
		return 3
	case bytes.IndexByte([]byte("\n\r\v\f\x1c\x1d\x1e"), head[0]) >= 0:
		return 1
	}
	return 0
}

// mutate returns the source with the mutant's line replaced and every other
// byte kept, its line break included. The script writes the file back
// through read_text and so turns CRLF into LF; an overlay has no reason to.
func mutate(source []byte, m Mutant) []byte {
	line := lineSpans(source)[m.Line-1]
	return slices.Concat(source[:line.start], []byte(m.Now), source[line.end:])
}
```

Zur Regel „neuer Code kompiliert Regexe über `sync.OnceValue`“: `ifLine` ist die einzige Regex; die Tabelle `comparisons` ist ein Literal, kein geparster Wert.

- [ ] **Step 4: Bestehen sehen**

```bash
go test ./internal/dev/mutants/
```
Erwartet: eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/dev/mutants` beginnt, dahinter die Dauer oder `(cached)`; Exit 0.

- [ ] **Step 5: Failing tests — Berichtsform und Lesart eines Laufs** — `internal/dev/mutants/mutants_test.go` (ganze Datei):

```go
package mutants

import (
	"path/filepath"
	"testing"
)

func TestMutantPrintsTheScriptsLine(t *testing.T) {
	m := Mutant{Family: "a3", Path: filepath.Join(t.TempDir(), "fixture.go.txt"), Line: 5, Was: "\tif a > 0 {", Now: "\tif a >= 0 {"}
	if got := m.String(); got != "(a3) fixture.go.txt:5  if a > 0 {  ->  if a >= 0 {" {
		t.Fatalf("%q", got)
	}
}

func TestClassifyReadsWhatGoTestPrinted(t *testing.T) {
	// Shortened from go test's own output (go1.27.0 windows/amd64, 2026-09-15).
	for _, c := range []struct {
		name     string
		printed  string
		passed   bool
		timedOut bool
		want     Outcome
	}{
		{"green", "ok  \texample.com/probe/p\t0.131s\n", true, false, Passed},
		{"red with a markdown heading in the message", "--- FAIL: TestSign (0.00s)\n    p_test.go:14: # brain\n        \n        Sign(1) = 0\nFAIL\nFAIL\texample.com/probe/p\t0.152s\nFAIL\n", false, false, Failed},
		{"compile error", "# example.com/probe/p [example.com/probe/p.test]\n.\\ov\\compile.go:4:2: declared and not used: x\nFAIL\texample.com/probe/p [build failed]\nFAIL\n", false, false, BuildFailed},
		{"vet error", "# example.com/probe/p\n# [example.com/probe/p]\n.\\ov\\vet.go:9:19: fmt.Sprintf format %d has arg \"x\" of wrong type string\nFAIL\texample.com/probe/p [build failed]\nFAIL\n", false, false, BuildFailed},
		{"package header behind other output", "FAIL\n# example.com/probe/p\n", false, false, BuildFailed},
		{"test binary timeout", "panic: test timed out after 2s\n\trunning tests:\n\t\tTestSign (2s)\n\ngoroutine 8 [running]:\nFAIL\texample.com/probe/p\t2.150s\nFAIL\n", false, false, Failed},
		{"patience ran out", "", false, true, TimedOut},
	} {
		if got := classify(c.printed, c.passed, c.timedOut); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
```

- [ ] **Step 6: Scheitern sehen**

```bash
go test ./internal/dev/mutants/
```
Erwartet (gemessen):
```
# github.com/xidus90/loomux/internal/dev/mutants [github.com/xidus90/loomux/internal/dev/mutants.test]
internal\dev\mutants\mutants_test.go:10:14: m.String undefined (type Mutant has no field or method String)
internal\dev\mutants\mutants_test.go:22:12: undefined: Outcome
internal\dev\mutants\mutants_test.go:24:65: undefined: Passed
internal\dev\mutants\mutants_test.go:25:201: undefined: Failed
internal\dev\mutants\mutants_test.go:26:186: undefined: BuildFailed
internal\dev\mutants\mutants_test.go:27:209: undefined: BuildFailed
internal\dev\mutants\mutants_test.go:28:89: undefined: BuildFailed
internal\dev\mutants\mutants_test.go:29:181: undefined: Failed
internal\dev\mutants\mutants_test.go:30:41: undefined: TimedOut
internal\dev\mutants\mutants_test.go:32:13: undefined: classify
internal\dev\mutants\mutants_test.go:32:13: too many errors
FAIL	github.com/xidus90/loomux/internal/dev/mutants [build failed]
```

- [ ] **Step 7: Implementieren — `internal/dev/mutants/mutants.go` ganz ersetzen** (ganze Datei):

```go
// Package mutants mutates the Go decisions of a package and reports which
// mutants its suite does not notice. It ports ultra-brain's
// tools/go_mutants.py with two changes of mechanism: a mutant reaches the
// suite through `go test -overlay` instead of being written into the tree,
// and several mutants run at once.
//
// Four families, named as the reports name them: (a1) the whole condition of
// an `if`, struck out as `true` and as `false`; (a2) each operand of a `&&`
// or `||` at the top level of the condition on its own; (a3) every
// comparison operator flipped; (a4) the condition negated, because a1 gives
// no signal at a `value, ok := x.(T)` guard.
//
// A mutant that does not compile is no mutant and is counted apart. A mutant
// the suite still passes has survived, and every survivor has to be read by
// hand: it is either a missing test or a place where the code cannot tell
// the difference.
package mutants

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// Mutant is one change: which file, which line, and what it says instead.
type Mutant struct {
	Family   string
	Path     string
	Line     int
	Was, Now string
}

// String is the script's report form, which names the file and not its path.
func (m Mutant) String() string {
	return fmt.Sprintf("(%s) %s:%d  %s  ->  %s", m.Family, filepath.Base(m.Path), m.Line, pytext.Strip(m.Was), pytext.Strip(m.Now))
}

// Outcome is what one run of a package suite came to.
type Outcome int

const (
	Passed Outcome = iota
	Failed
	BuildFailed
	TimedOut
)

// TestFunc runs the suite of pkg; an empty overlay runs the tree as it stands.
type TestFunc func(pkg, overlay string) (Outcome, error)

// goTimeout is the suite's own bound on one run. A mutant that strikes out a
// cycle guard turns a walk into an endless one, and without a bound each such
// mutant costs the toolchain's ten-minute default.
const goTimeout = "60s"

// patience is the backstop above goTimeout, the script's PATIENCE["go"]. The
// test binary's bound fires first and cleanly; this one ends a go command
// that does not come back at all.
const patience = 120 * time.Second

// GoTest runs `go test` in root with the script's flags. A go command that
// does not start is an error; a run that ends, however it ends, is an Outcome.
func GoTest(root string) TestFunc {
	return func(pkg, overlay string) (Outcome, error) {
		args := []string{"test"}
		if overlay != "" {
			args = append(args, "-overlay", overlay)
		}
		args = append(args, "-count=1", "-failfast", "-timeout", goTimeout, "./"+pkg+"/")
		ctx, cancel := context.WithTimeout(context.Background(), patience)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = root
		cmd.WaitDelay = 5 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		timedOut := ctx.Err() != nil
		var exit *exec.ExitError
		if err != nil && !timedOut && !errors.As(err, &exit) {
			return 0, err
		}
		return classify(stdout.String()+stderr.String(), err == nil, timedOut), nil
	}
}

// classify reads a finished run the way the script's _run does: a run cut off
// by patience compiled and did not pass; "[build failed]" (compiler or vet)
// or a line opening with "# " (the compiler's package header) means the
// mutant did not build; otherwise the exit status decides.
func classify(printed string, passed, timedOut bool) Outcome {
	switch {
	case timedOut:
		return TimedOut
	case strings.Contains(printed, "[build failed]") || strings.Contains(printed, "\n# "):
		return BuildFailed
	case passed:
		return Passed
	}
	return Failed
}
```

- [ ] **Step 8: Bestehen sehen**

```bash
go test ./internal/dev/mutants/
```
Erwartet: eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/dev/mutants` beginnt, dahinter die Dauer oder `(cached)`; Exit 0. `GoTest` ist hier noch ungetestet; Step 9 holt es nach.

- [ ] **Step 9: Failing tests — die Runde**

`internal/dev/mutants/round_test.go` (ganze Datei):

```go
package mutants

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

const signSource = "package p\n\nfunc Sign(a int) int {\n\tif a > 0 {\n\t\treturn 1\n\t}\n\treturn 0\n}\n"

const oneSource = "package p\n\nfunc One(x int) bool {\n\tif x == 1 {\n\t\treturn true\n\t}\n\treturn false\n}\n"

// writePackage lays out one package directory under root.
func writePackage(t *testing.T, root, pkg string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(root, pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// call is one run the fake suite was asked for.
type call struct {
	pkg, overlay string
	line         string            // the line the replacement changes; "" for a baseline
	replace      map[string]string // the overlay's Replace; nil for a baseline
}

// fakeSuite is a TestFunc that starts nothing. It reads the overlay go test
// would get, finds the line the replacement changes, and answers from it.
type fakeSuite struct {
	mu       sync.Mutex
	calls    []call
	baseline func(pkg string) (Outcome, error)
	mutant   func(line string) (Outcome, error)
}

func (f *fakeSuite) test(pkg, overlay string) (Outcome, error) {
	c := call{pkg: pkg, overlay: overlay}
	if overlay != "" {
		var o struct{ Replace map[string]string }
		data, err := os.ReadFile(overlay)
		if err == nil {
			err = json.Unmarshal(data, &o)
		}
		if err != nil {
			return 0, err
		}
		c.replace = o.Replace
		for source, replacement := range o.Replace {
			c.line = changedLine(source, replacement)
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, c)
	f.mu.Unlock()
	if overlay == "" {
		return f.baseline(pkg)
	}
	return f.mutant(c.line)
}

// mutantCalls are the calls that carried an overlay.
func (f *fakeSuite) mutantCalls() []call {
	var out []call
	for _, c := range f.calls {
		if c.overlay != "" {
			out = append(out, c)
		}
	}
	return out
}

// changedLine is the first line of replacement that differs from source.
func changedLine(source, replacement string) string {
	a, _ := os.ReadFile(source)
	b, _ := os.ReadFile(replacement)
	was, now := strings.Split(string(a), "\n"), strings.Split(string(b), "\n")
	for i := range min(len(was), len(now)) {
		if was[i] != now[i] {
			return now[i]
		}
	}
	return ""
}

func green(string) (Outcome, error) { return Passed, nil }

func TestRoundReportsEveryVerdictInGenerationOrder(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{
		"p.go":      signSource,
		"p_test.go": "package p\n\nfunc probe(a int) {\n\tif a > 1 {\n\t}\n}\n",
	})
	release := make(chan struct{})
	suite := &fakeSuite{baseline: green, mutant: func(line string) (Outcome, error) {
		switch line {
		case "\tif true {":
			// Ends only after the last mutant has started, so the report
			// has to wait for it instead of printing in finishing order.
			<-release
			return Passed, nil
		case "\tif false {":
			return Failed, nil
		case "\tif !(a > 0) {":
			return BuildFailed, nil
		case "\tif a >= 0 {":
			close(release)
			return TimedOut, nil
		}
		return 0, fmt.Errorf("unexpected mutated line %q", line)
	}}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 4}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := "[1/4] SURVIVED  (a1) p.go:4  if a > 0 {  ->  if true {\n" +
		"[2/4] killed    (a1) p.go:4  if a > 0 {  ->  if false {\n" +
		"[3/4] no mutant (a4) p.go:4  if a > 0 {  ->  if !(a > 0) {\n" +
		"[4/4] killed    (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		"\n" +
		"4 mutants over p, oracle go\n" +
		"1 do not compile and are no mutants\n" +
		"1 survived:\n" +
		"  (a1) p.go:4  if a > 0 {  ->  if true {\n"
	if out.String() != want {
		t.Fatalf("report:\n%s", out.String())
	}
	if sum.Killed != 2 || sum.Survived != 1 || sum.NotCompiled != 1 || sum.NoMutant != 0 ||
		len(sum.Survivors) != 1 || sum.Survivors[0].Now != "\tif true {" {
		t.Fatalf("summary %+v", sum)
	}
	source := filepath.Join(root, "p", "p.go")
	if len(suite.calls) != 5 || suite.calls[0].pkg != "p" || suite.calls[0].overlay != "" {
		t.Fatalf("calls %+v", suite.calls)
	}
	for _, c := range suite.mutantCalls() {
		if c.pkg != "p" || len(c.replace) != 1 || filepath.Base(c.replace[source]) != "p.go" || filepath.Base(c.overlay) != "overlay.json" {
			t.Fatalf("overlay %+v", c)
		}
		if _, err := os.Stat(filepath.Dir(c.overlay)); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("%s outlived the round: %v", filepath.Dir(c.overlay), err)
		}
	}
	if data, _ := os.ReadFile(source); string(data) != signSource {
		t.Fatalf("the tree was written: %q", data)
	}
}

func TestRoundRefusesARedBaselineBeforeAnyMutant(t *testing.T) {
	for _, red := range []Outcome{Failed, BuildFailed, TimedOut} {
		root := t.TempDir()
		writePackage(t, root, "p", map[string]string{"p.go": signSource})
		writePackage(t, root, "q", map[string]string{"q.go": oneSource})
		suite := &fakeSuite{
			baseline: func(pkg string) (Outcome, error) {
				if pkg == "q" {
					return red, nil
				}
				return Passed, nil
			},
			mutant: func(string) (Outcome, error) { return Failed, nil },
		}
		var out strings.Builder
		_, err := Round(Options{Packages: []string{"p", "q"}, Root: root}, suite.test, &out)
		if !errors.Is(err, ErrBaselineRed) || err.Error() != "q: the suite is not green before the round" {
			t.Fatalf("outcome %d: %v", red, err)
		}
		if len(suite.calls) != 2 || out.Len() != 0 {
			t.Fatalf("outcome %d: calls %+v, report %q", red, suite.calls, out.String())
		}
	}
}

func TestRoundPassesOnAnErrorOfTheBaseline(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	suite := &fakeSuite{baseline: func(string) (Outcome, error) { return 0, errors.New("go: not found") }}
	if _, err := Round(Options{Packages: []string{"p"}, Root: root}, suite.test, &strings.Builder{}); err == nil || err.Error() != "go: not found" {
		t.Fatalf("%v", err)
	}
}

func TestRoundNeedsSourceFilesBeforeItAsksTheSuite(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "tests", map[string]string{"p_test.go": "package p\n"})
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	for _, c := range []struct {
		pkg, only, want string
	}{
		{"tests", "", "tests: no source files"},
		{"p", "nothing", "p: no source files"},
	} {
		suite := &fakeSuite{baseline: green}
		_, err := Round(Options{Packages: []string{c.pkg}, Root: root, Only: c.only}, suite.test, &strings.Builder{})
		if !errors.Is(err, ErrNoSources) || err.Error() != c.want || len(suite.calls) != 0 {
			t.Fatalf("%s: %v, calls %+v", c.pkg, err, suite.calls)
		}
	}
	suite := &fakeSuite{baseline: green}
	_, err := Round(Options{Packages: []string{"gone"}, Root: root}, suite.test, &strings.Builder{})
	if err == nil || errors.Is(err, ErrNoSources) || len(suite.calls) != 0 {
		t.Fatalf("missing package: %v", err)
	}
}

func TestRoundNeedsAnAbsoluteRoot(t *testing.T) {
	_, err := Round(Options{Packages: []string{"p"}, Root: "relative"}, (&fakeSuite{}).test, &strings.Builder{})
	if err == nil || err.Error() != `root "relative" is not an absolute path` {
		t.Fatalf("%v", err)
	}
}

func TestRoundFiltersByFileNameAndFamily(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"sign.go": signSource, "one.go": oneSource})
	suite := &fakeSuite{baseline: green, mutant: func(string) (Outcome, error) { return Failed, nil }}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p"}, Root: root, Only: "on", Family: "a3", Workers: 1}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := "[1/1] killed    (a3) one.go:4  if x == 1 {  ->  if x != 1 {\n" +
		"\n" +
		"1 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"0 survived:\n"
	if out.String() != want || sum.Killed != 1 {
		t.Fatalf("summary %+v, report:\n%s", sum, out.String())
	}
}

func TestRoundCountsAnUnchangedLineAsNoMutant(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"t.go": "package p\n\nfunc T() int {\n\tif true {\n\t\treturn 1\n\t}\n\treturn 0\n}\n"})
	suite := &fakeSuite{baseline: green, mutant: func(string) (Outcome, error) { return Failed, nil }}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 1}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := "[1/3] no mutant (a1) t.go:4  if true {  ->  if true {\n" +
		"[2/3] killed    (a1) t.go:4  if true {  ->  if false {\n" +
		"[3/3] killed    (a4) t.go:4  if true {  ->  if !(true) {\n" +
		"\n" +
		"3 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"1 change nothing and are no mutants\n" +
		"0 survived:\n"
	if out.String() != want || sum.NoMutant != 1 || sum.Killed != 2 || len(suite.mutantCalls()) != 2 {
		t.Fatalf("summary %+v, calls %d, report:\n%s", sum, len(suite.mutantCalls()), out.String())
	}
}

func TestRoundStopsAtTheFirstErrorAndLeavesNoOverlay(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	suite := &fakeSuite{baseline: green}
	suite.mutant = func(string) (Outcome, error) {
		if len(suite.mutantCalls()) > 1 {
			return 0, errors.New("a mutant ran after the error")
		}
		return 0, errors.New("go vanished")
	}
	var out strings.Builder
	_, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 1}, suite.test, &out)
	if err == nil || err.Error() != "go vanished" || out.Len() != 0 {
		t.Fatalf("%v, report %q", err, out.String())
	}
	calls := suite.mutantCalls()
	if len(calls) != 1 {
		t.Fatalf("calls %+v", calls)
	}
	if _, err := os.Stat(filepath.Dir(calls[0].overlay)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("overlay directory left behind: %v", err)
	}
}

func TestRoundReportsATemporaryDirectoryItCannotMake(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	missing := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TMP", missing)
	t.Setenv("TEMP", missing)
	t.Setenv("TMPDIR", missing)
	suite := &fakeSuite{baseline: green}
	_, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 1}, suite.test, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), "missing") || len(suite.mutantCalls()) != 0 {
		t.Fatalf("%v", err)
	}
}

func TestRoundReportsAnUnreadableSource(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	if err := os.Mkdir(filepath.Join(root, "p", "dir.go"), 0o755); err != nil {
		t.Fatal(err)
	}
	suite := &fakeSuite{baseline: green}
	var out strings.Builder
	_, err := Round(Options{Packages: []string{"p"}, Root: root}, suite.test, &out)
	if err == nil || len(suite.mutantCalls()) != 0 || out.Len() != 0 {
		t.Fatalf("%v, report %q", err, out.String())
	}
}

func TestRoundAddsUpEveryPackage(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	writePackage(t, root, "q", map[string]string{"q.go": oneSource})
	suite := &fakeSuite{baseline: green, mutant: func(string) (Outcome, error) { return Passed, nil }}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p", "q"}, Root: root, Family: "a3"}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := "[1/1] SURVIVED  (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		"\n" +
		"1 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"1 survived:\n" +
		"  (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		"[1/1] SURVIVED  (a3) q.go:4  if x == 1 {  ->  if x != 1 {\n" +
		"\n" +
		"1 mutants over q, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"1 survived:\n" +
		"  (a3) q.go:4  if x == 1 {  ->  if x != 1 {\n"
	if out.String() != want || sum.Survived != 2 || len(sum.Survivors) != 2 {
		t.Fatalf("summary %+v, report:\n%s", sum, out.String())
	}
}

func TestDefaultWorkersIsHalfTheProcessors(t *testing.T) {
	if got := DefaultWorkers(); got != max(runtime.NumCPU()/2, 1) {
		t.Fatalf("%d", got)
	}
}
```

`internal/dev/mutants/gotest_test.go` (ganze Datei) — der einzige Test, der den echten `go`-Befehl startet (Vorbild `TestDevRecordCaseRecordsABinary`); er baut sein Probemodul nur aus der Standardbibliothek in `t.TempDir()`:

```go
package mutants

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// probeModule writes a module of its own into a fresh directory: one package p
// whose only test asks Sign(1). It skips where no go command is on PATH.
func probeModule(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go binary on PATH")
	}
	root := t.TempDir()
	writePackage(t, root, ".", map[string]string{"go.mod": "module example.com/probe\n\ngo 1.25.0\n"})
	writePackage(t, root, "p", map[string]string{
		"p.go":      signSource,
		"p_test.go": "package p\n\nimport \"testing\"\n\nfunc TestSign(t *testing.T) {\n\tif Sign(1) != 1 {\n\t\tt.Fatal(\"Sign(1) is not 1\")\n\t}\n}\n",
	})
	return root
}

// The one test that starts the real go command: it holds the overlay file, the
// package pattern and classify to what go test does with them.
func TestGoTestRunsARoundThroughOverlays(t *testing.T) {
	root := probeModule(t)
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 2}, GoTest(root), &out)
	if err != nil {
		t.Fatal(err)
	}
	want := "[1/4] SURVIVED  (a1) p.go:4  if a > 0 {  ->  if true {\n" +
		"[2/4] killed    (a1) p.go:4  if a > 0 {  ->  if false {\n" +
		"[3/4] killed    (a4) p.go:4  if a > 0 {  ->  if !(a > 0) {\n" +
		"[4/4] SURVIVED  (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		"\n" +
		"4 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"2 survived:\n" +
		"  (a1) p.go:4  if a > 0 {  ->  if true {\n" +
		"  (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n"
	if out.String() != want || sum.Killed != 2 || sum.Survived != 2 {
		t.Fatalf("summary %+v, report:\n%s", sum, out.String())
	}
}

func TestGoTestTellsABuildFailureFromAFailure(t *testing.T) {
	root := probeModule(t)
	dir := t.TempDir()
	broken := filepath.Join(dir, "p.go")
	if err := os.WriteFile(broken, []byte("package p\n\nfunc Sign(a int) int {\n\tx := 1\n\treturn 0\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlay, overlayJSON(filepath.Join(root, "p", "p.go"), broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if outcome, err := GoTest(root)("p", overlay); err != nil || outcome != BuildFailed {
		t.Fatalf("outcome %d, err %v", outcome, err)
	}
}

func TestGoTestReportsAGoCommandThatDoesNotStart(t *testing.T) {
	root := probeModule(t)
	t.Setenv("PATH", "")
	if _, err := GoTest(root)("p", ""); err == nil {
		t.Fatal("a go command that cannot be found must be an error")
	}
}
```

- [ ] **Step 10: Scheitern sehen**

```bash
go test ./internal/dev/mutants/
```
Erwartet (gemessen):
```
# github.com/xidus90/loomux/internal/dev/mutants [github.com/xidus90/loomux/internal/dev/mutants.test]
internal\dev\mutants\gotest_test.go:32:14: undefined: Round
internal\dev\mutants\gotest_test.go:32:20: undefined: Options
internal\dev\mutants\gotest_test.go:59:34: undefined: overlayJSON
internal\dev\mutants\round_test.go:126:14: undefined: Round
internal\dev\mutants\round_test.go:126:20: undefined: Options
internal\dev\mutants\round_test.go:178:13: undefined: Round
internal\dev\mutants\round_test.go:178:19: undefined: Options
internal\dev\mutants\round_test.go:179:22: undefined: ErrBaselineRed
internal\dev\mutants\round_test.go:192:15: undefined: Round
internal\dev\mutants\round_test.go:192:21: undefined: Options
internal\dev\mutants\round_test.go:192:21: too many errors
FAIL	github.com/xidus90/loomux/internal/dev/mutants [build failed]
```

- [ ] **Step 11: Implementieren — `internal/dev/mutants/round.go`** (ganze Datei):

```go
package mutants

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// Options says what a round mutates. Root is the absolute directory the
// packages are named relative to and go test runs in.
type Options struct {
	Packages []string
	Root     string
	Only     string
	Family   string
	Workers  int
}

// Summary adds a round up over all its packages.
type Summary struct {
	Killed, Survived, NotCompiled, NoMutant int
	Survivors                               []Mutant
}

// ErrBaselineRed refuses a round whose suite fails before the first mutant:
// every mutant would count as killed.
var ErrBaselineRed = errors.New("the suite is not green before the round")

// ErrNoSources refuses a package, or an Only filter, that leaves no file to
// mutate.
var ErrNoSources = errors.New("no source files")

// DefaultWorkers is half the processors and at least one: one go test run
// already compiles and tests on several cores.
func DefaultWorkers() int {
	return max(runtime.NumCPU()/2, 1)
}

// Round mutates every package and writes the script's report to w. Every
// package is checked for sources and for a green suite before the first
// mutant runs; then each package gets its block of verdicts and sums. A
// survivor is a finding and not an error; the first error ends the round.
func Round(opts Options, test TestFunc, w io.Writer) (Summary, error) {
	var total Summary
	// An overlay names the replaced file by its absolute path, and Round does
	// not guess one from the working directory of this process.
	if !filepath.IsAbs(opts.Root) {
		return total, fmt.Errorf("root %q is not an absolute path", opts.Root)
	}
	files := make([][]string, len(opts.Packages))
	for i, pkg := range opts.Packages {
		found, err := sources(filepath.Join(opts.Root, pkg), opts.Only)
		if err != nil {
			return total, err
		}
		if len(found) == 0 {
			return total, fmt.Errorf("%s: %w", pkg, ErrNoSources)
		}
		files[i] = found
	}
	for _, pkg := range opts.Packages {
		outcome, err := test(pkg, "")
		if err != nil {
			return total, err
		}
		if outcome != Passed {
			return total, fmt.Errorf("%s: %w", pkg, ErrBaselineRed)
		}
	}
	workers := opts.Workers
	if workers < 1 {
		workers = DefaultWorkers()
	}
	for i, pkg := range opts.Packages {
		sum, err := roundOf(pkg, files[i], opts.Family, workers, test, w)
		total.Killed += sum.Killed
		total.Survived += sum.Survived
		total.NotCompiled += sum.NotCompiled
		total.NoMutant += sum.NoMutant
		total.Survivors = append(total.Survivors, sum.Survivors...)
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// sources lists the non-test Go files of a package directory whose name
// contains only, in name order.
func sources(dir, only string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") && strings.Contains(name, only) {
			found = append(found, filepath.Join(dir, name))
		}
	}
	return found, nil
}

// finished is one mutant's turn as the report needs it.
type finished struct {
	outcome   Outcome
	unchanged bool
	skipped   bool
	err       error
}

// roundOf runs one package's mutants on a pool of workers and reports each in
// generation order as soon as its turn has come, so the report reads the same
// whatever order the runs finish in.
func roundOf(pkg string, files []string, family string, workers int, test TestFunc, w io.Writer) (Summary, error) {
	var sum Summary
	originals := map[string][]byte{}
	var all []Mutant
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil {
			return sum, err
		}
		originals[path] = source
		for _, m := range Generate(path, source) {
			if family == "" || m.Family == family {
				all = append(all, m)
			}
		}
	}

	results := make([]chan finished, len(all))
	for i := range results {
		results[i] = make(chan finished, 1)
	}
	var first atomic.Pointer[error]
	var wg sync.WaitGroup
	defer wg.Wait()
	slots := make(chan struct{}, workers)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i, m := range all {
			slots <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] <- runOne(pkg, m, originals[m.Path], test, &first)
				<-slots
			}()
		}
	}()

	for i, m := range all {
		r := <-results[i]
		if r.skipped || r.err != nil {
			return sum, *first.Load()
		}
		var verdict string
		switch {
		case r.unchanged:
			verdict = "no mutant"
			sum.NoMutant++
		case r.outcome == BuildFailed:
			verdict = "no mutant"
			sum.NotCompiled++
		case r.outcome == Passed:
			verdict = "SURVIVED "
			sum.Survived++
			sum.Survivors = append(sum.Survivors, m)
		default:
			// A mutant that never answers has failed like one that answers
			// wrongly: Failed and TimedOut are both killed.
			verdict = "killed   "
			sum.Killed++
		}
		fmt.Fprintf(w, "[%d/%d] %s %s\n", i+1, len(all), verdict, m)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%d mutants over %s, oracle go\n", len(all), pkg)
	fmt.Fprintf(w, "%d do not compile and are no mutants\n", sum.NotCompiled)
	if sum.NoMutant > 0 {
		fmt.Fprintf(w, "%d change nothing and are no mutants\n", sum.NoMutant)
	}
	fmt.Fprintf(w, "%d survived:\n", len(sum.Survivors))
	for _, m := range sum.Survivors {
		fmt.Fprintf(w, "  %s\n", m)
	}
	return sum, nil
}

// runOne is one mutant's turn. A mutant that leaves its line as it was is no
// mutant and never reaches the suite; once a run has failed with an error,
// no further mutant starts.
func runOne(pkg string, m Mutant, source []byte, test TestFunc, first *atomic.Pointer[error]) finished {
	if m.Now == m.Was {
		return finished{unchanged: true}
	}
	if first.Load() != nil {
		return finished{skipped: true}
	}
	outcome, err := try(pkg, m, source, test)
	if err != nil {
		first.CompareAndSwap(nil, &err)
	}
	return finished{outcome: outcome, err: err}
}

// try hands one mutant to the suite through an overlay in a directory of its
// own, which is removed again whatever the run came to.
func try(pkg string, m Mutant, source []byte, test TestFunc) (Outcome, error) {
	dir, err := os.MkdirTemp("", "loomux-mutant-")
	if err == nil {
		defer os.RemoveAll(dir)
		replacement := filepath.Join(dir, filepath.Base(m.Path))
		overlay := filepath.Join(dir, "overlay.json")
		// A write into the directory this call has just made fails only
		// through the system; both writes share the error check that a
		// missing temporary directory already exercises.
		err = errors.Join(
			os.WriteFile(replacement, mutate(source, m), 0o644),
			os.WriteFile(overlay, overlayJSON(m.Path, replacement), 0o644),
		)
		if err == nil {
			return test(pkg, overlay)
		}
	}
	return 0, err
}

// overlayJSON is the file go test -overlay reads:
// {"Replace": {"<source>": "<replacement>"}}.
func overlayJSON(source, replacement string) []byte {
	// A map of strings holds no value json.Marshal can refuse.
	data, _ := json.Marshal(map[string]map[string]string{"Replace": {source: replacement}})
	return data
}
```

Die Zeitgrenze des Laufs gilt als getötet (`default`-Arm), der Arbeitsbaum wird nie beschrieben (`TestRoundReportsEveryVerdictInGenerationOrder` liest die Quelle nach der Runde), jedes Overlay-Verzeichnis ist nach seinem Lauf fort, auch nach einem Fehler (`TestRoundStopsAtTheFirstErrorAndLeavesNoOverlay`). Einen Abbruch mit Ctrl+C macht Step 15 zu genau so einem Fehler (`untilInterrupted`, geprüft in `TestDevMutantsCleansUpAnInterruptedRound`).

- [ ] **Step 12: Bestehen sehen und Coverage**

```bash
go test ./internal/dev/mutants/ -count=1 -covermode=set -coverprofile="$TEMP/mutants.out" > "$TEMP/task13-mutants.txt" 2>&1
```
Erwartet: Exit 0.
```bash
grep -E '^ok[[:space:]]+github\.com/xidus90/loomux/internal/dev/mutants[[:space:]]+[0-9.]+s[[:space:]]+coverage: 100\.0% of statements$' "$TEMP/task13-mutants.txt"
```
Erwartet: genau ein Treffer. Die Dauer schwankt von Lauf zu Lauf (gemessen 2,2 s; rund 2 s davon die drei `gotest_test.go`-Läufe).
```bash
go tool cover -func="$TEMP/mutants.out" > "$TEMP/task13-mutants-func.txt"
```
Erwartet: Exit 0, keine Ausgabe.
```bash
grep -vE '100\.0%$' "$TEMP/task13-mutants-func.txt"
```
Erwartet: jede Funktion des Pakets 100.0%, also keine Ausgabe; Exit 1 von `grep` (kein Treffer) ist hier das erwartete Ergebnis. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.
```bash
grep -E 'mutants\.go:[0-9]+:[[:space:]]+GoTest[[:space:]]+100\.0%$' "$TEMP/task13-mutants-func.txt"
```
Erwartet: genau ein Treffer; Zeilennummern variieren mit dem Kommentar. Der Rückgabezweig für einen nicht startbaren `go` läuft über `TestGoTestReportsAGoCommandThatDoesNotStart`, der `timedOut`-Wert ist eine Anweisung ohne eigenen Block. Darum kein `//coverage:exempt`.
```bash
gofmt -l internal/dev/mutants
```
Erwartet: keine Ausgabe.

- [ ] **Step 13: Failing tests — der Befehl** — `internal/cli/dev_test.go`, zwei Stellen:

(a) Importblock, drei Stellen: `"context"` vor `"errors"`, `"os/signal"` zwischen `"os/exec"` und `"path/filepath"`, `"github.com/xidus90/loomux/internal/dev/mutants"` unter `"github.com/xidus90/loomux/internal/dev/benchhooks"`. Task 11 ändert den Importblock von `dev_test.go` nicht; er lautet danach:
```go
import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/mutants"
)
```

(b) Ans Ende der Datei anhängen (hinter die Tests, die Task 11 eingefügt hat):
```go
// mutantsWorld lays out a root with one package p, points the dev mutants
// seams at it and at test, and restores both when the test ends.
func mutantsWorld(t *testing.T, test mutants.TestFunc) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "package p\n\nfunc Sign(a int) int {\n\tif a > 0 {\n\t\treturn 1\n\t}\n\treturn 0\n}\n"
	if err := os.WriteFile(filepath.Join(root, "p", "p.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	mutantsRoot = func() (string, error) { return root, nil }
	mutantsTest = func(dir string) mutants.TestFunc {
		if dir != root {
			t.Errorf("go test runs in %q, want %q", dir, root)
		}
		return test
	}
	t.Cleanup(func() {
		mutantsRoot = os.Getwd
		mutantsTest = mutants.GoTest
	})
}

// killEveryMutant is a suite that is green alone and red under any overlay.
func killEveryMutant(_, overlay string) (mutants.Outcome, error) {
	if overlay == "" {
		return mutants.Passed, nil
	}
	return mutants.Failed, nil
}

func TestDevMutantsRunsARound(t *testing.T) {
	mutantsWorld(t, killEveryMutant)
	code, out, errOut := run("dev", "mutants", "p", "--workers", "2")
	if code != 0 || !strings.Contains(out, "[1/4] killed    (a1) p.go:4  if a > 0 {  ->  if true {\n") ||
		!strings.Contains(out, "\n4 mutants over p, oracle go\n0 do not compile and are no mutants\n0 survived:\n") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestDevMutantsTakesFlagsBetweenPackages(t *testing.T) {
	mutantsWorld(t, killEveryMutant)
	code, out, errOut := run("dev", "mutants", "--family", "a3", "p", "--only", "p.go", "p")
	if code != 0 || strings.Count(out, "\n1 mutants over p, oracle go\n") != 2 {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestDevMutantsRefusesBadArguments(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"dev", "mutants"}, "loomux dev mutants: at least one package is required\n"},
		{[]string{"dev", "mutants", "p", "--family", "a5"}, "loomux dev mutants: --family must be one of a1, a2, a3, a4, got \"a5\"\n"},
		{[]string{"dev", "mutants", "--workers", "0", "p"}, "loomux dev mutants: --workers must be at least 1, got 0\n"},
	} {
		if code, _, errOut := run(c.args...); code != 2 || errOut != c.want {
			t.Errorf("%v: code %d, err %q", c.args, code, errOut)
		}
	}
	for _, args := range [][]string{{"dev", "mutants", "--bogus"}, {"dev", "mutants", "p", "--bogus"}} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%v: code %d", args, code)
		}
	}
}

func TestDevMutantsReportsAMissingWorkingDirectory(t *testing.T) {
	mutantsWorld(t, killEveryMutant)
	mutantsRoot = func() (string, error) { return "", errors.New("getwd: gone") }
	code, _, errOut := run("dev", "mutants", "p")
	if code != 1 || errOut != "loomux dev mutants: getwd: gone\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevMutantsRefusesARedSuiteAndAnEmptyPackage(t *testing.T) {
	mutantsWorld(t, func(string, string) (mutants.Outcome, error) { return mutants.Failed, nil })
	code, out, errOut := run("dev", "mutants", "p")
	if code != 2 || out != "" || errOut != "loomux dev mutants: p: the suite is not green before the round\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	code, _, errOut = run("dev", "mutants", "p", "--only", "nothing")
	if code != 2 || errOut != "loomux dev mutants: p: no source files\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevMutantsReportsABrokenRun(t *testing.T) {
	mutantsWorld(t, func(string, string) (mutants.Outcome, error) { return 0, errors.New("go: not found") })
	code, _, errOut := run("dev", "mutants", "p")
	if code != 1 || errOut != "loomux dev mutants: go: not found\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// Ctrl+C cannot be sent to a test process on Windows; the seam hands the
// round a context the test cancels while the first mutant runs.
func TestDevMutantsCleansUpAnInterruptedRound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	t.Setenv("TMPDIR", tmp)
	var interrupt context.CancelFunc
	mutantsNotify = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		interrupt = cancel
		return ctx, cancel
	}
	t.Cleanup(func() { mutantsNotify = signal.NotifyContext })
	mutantsWorld(t, func(_, overlay string) (mutants.Outcome, error) {
		if overlay != "" {
			if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 1 {
				t.Errorf("while the mutant runs: %v, %v", entries, err)
			}
			interrupt()
		}
		return mutants.Passed, nil
	})
	code, out, errOut := run("dev", "mutants", "p", "--workers", "1")
	if code != 1 || out != "" || errOut != "loomux dev mutants: context canceled\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 0 {
		t.Fatalf("left behind: %v, %v", entries, err)
	}
}

func TestUntilInterruptedStartsNoRunAfterTheInterrupt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := false
	test := untilInterrupted(ctx, func(string, string) (mutants.Outcome, error) {
		started = true
		return mutants.Passed, nil
	})
	if _, err := test("p", ""); !errors.Is(err, context.Canceled) || started {
		t.Fatalf("err %v, started %t", err, started)
	}
}
```

`TestDevMutantsCleansUpAnInterruptedRound` deckt den Zweig „Abbruch während eines Laufs“ ab: Mit einem Arbeiter startet nach dem ersten Fehler kein weiterer Mutant (`runOne` prüft `first`), darum erreicht kein Lauf `untilInterrupted` nach dem Abbruch. Den Zweig „Abbruch vor dem Lauf“, den mehrere Arbeiter im echten Betrieb treffen, prüft `TestUntilInterruptedStartsNoRunAfterTheInterrupt` direkt.

- [ ] **Step 14: Scheitern sehen**

```bash
go test ./internal/cli/ -run 'TestDevMutants|TestUntilInterrupted' > "$TEMP/task13-cli-red.txt" 2>&1
```
Erwartet: Exit 1. Gemessen gegen `dev.go` ohne die Änderungen aus Step 15: acht Compilerfehler, keine Zeile `too many errors`.
```bash
grep -cE '^internal.cli.dev_test\.go:[0-9]+:[0-9]+: undefined: (mutantsRoot|mutantsTest|mutantsNotify|untilInterrupted)$' "$TEMP/task13-cli-red.txt"
```
Erwartet: `8` — in der Datei der Reihe nach `mutantsRoot`, `mutantsTest`, `mutantsRoot`, `mutantsTest`, `mutantsRoot`, `mutantsNotify`, `mutantsNotify`, `untilInterrupted`. Zeilennummern variieren mit dem Kommentar.
```bash
grep -cxF 'FAIL	github.com/xidus90/loomux/internal/cli [build failed]' "$TEMP/task13-cli-red.txt"
```
Erwartet: `1`.

- [ ] **Step 15: Implementieren — `internal/cli/dev.go`**, vier Stellen; alles andere bleibt wortgleich.

(a) Importblock, drei Stellen: `"context"` zwischen `"bytes"` und `"encoding/json"`, `"os/signal"` zwischen `"os/exec"` und `"strings"`, `"github.com/xidus90/loomux/internal/dev/mutants"` zwischen `"github.com/xidus90/loomux/internal/dev/importcases"` und `"github.com/xidus90/loomux/internal/dev/recordcase"`. Mit dem Stand nach Task 11 lautet er danach:
```go
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/covergate"
	"github.com/xidus90/loomux/internal/dev/importcases"
	"github.com/xidus90/loomux/internal/dev/mutants"
	"github.com/xidus90/loomux/internal/dev/recordcase"
	"github.com/xidus90/loomux/internal/dev/swap"
)
```

(b) Direkt unter `var benchExec = benchhooks.Exec`:
```go

var mutantsTest = mutants.GoTest

var mutantsRoot = os.Getwd

var mutantsNotify = signal.NotifyContext
```

(c) In `devCommands` zwischen `"import-cases": devImportCases,` und `"record-case":  devRecordCase,`:
```go
	"mutants":      devMutants,
```

(d) Direkt hinter `devCommand` (vor dem Kommentar von `devBenchHooks`):
```go
// devMutants runs a mutation round over packages named relative to the
// working directory. Packages and flags may be mixed: Go's flag package stops
// at the first argument that is no flag, so parsing resumes behind each
// package.
func devMutants(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev mutants", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts mutants.Options
	fs.StringVar(&opts.Only, "only", "", "restrict to files whose name contains this")
	fs.StringVar(&opts.Family, "family", "", "restrict to one of a1, a2, a3, a4")
	fs.IntVar(&opts.Workers, "workers", mutants.DefaultWorkers(), "go test runs at the same time")
	for rest := args; ; rest = rest[1:] {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		opts.Packages = append(opts.Packages, rest[0])
	}
	if len(opts.Packages) == 0 {
		fmt.Fprintln(stderr, "loomux dev mutants: at least one package is required")
		return 2
	}
	switch opts.Family {
	case "", "a1", "a2", "a3", "a4":
	default:
		fmt.Fprintf(stderr, "loomux dev mutants: --family must be one of a1, a2, a3, a4, got %q\n", opts.Family)
		return 2
	}
	if opts.Workers < 1 {
		fmt.Fprintf(stderr, "loomux dev mutants: --workers must be at least 1, got %d\n", opts.Workers)
		return 2
	}
	root, err := mutantsRoot()
	if err == nil {
		// Ctrl+C would end the process before any deferred removal of an
		// overlay directory; caught, it ends the round through its error path.
		ctx, stop := mutantsNotify(context.Background(), os.Interrupt)
		defer stop()
		opts.Root = root
		_, err = mutants.Round(opts, untilInterrupted(ctx, mutantsTest(root)), stdout)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev mutants: %v\n", err)
		// A red suite or a package without sources is a wrong question, as
		// the script answers it with 2; anything else broke on the way.
		if errors.Is(err, mutants.ErrBaselineRed) || errors.Is(err, mutants.ErrNoSources) {
			return 2
		}
		return 1
	}
	return 0
}

// untilInterrupted turns an interrupt into the error of a run. A run asked
// for afterwards does not start; a run under way when it came reports the
// interrupt instead of its verdict. Round stops at the first error and waits
// for every run it started, so each overlay directory is gone before the
// command returns.
func untilInterrupted(ctx context.Context, test mutants.TestFunc) mutants.TestFunc {
	return func(pkg, overlay string) (mutants.Outcome, error) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		outcome, err := test(pkg, overlay)
		if interrupted := ctx.Err(); interrupted != nil {
			return 0, interrupted
		}
		return outcome, err
	}
}
```
Die übrigen Importe (`errors`, `flag`, `fmt`, `io`, `os`) stehen schon da; neu sind nur die drei aus (a). Ein Ctrl+C in der Konsole erreicht auch die laufenden `go test`-Kindprozesse; sie enden, `GoTest` kehrt zurück, und `untilInterrupted` meldet statt des Urteils `context canceled`. Auf einen Kindprozess, der nicht endet, wartet die Runde höchstens `patience` (120 s).

- [ ] **Step 16: Bestehen sehen und Coverage**

```bash
go test ./internal/cli/ -run 'TestDev|TestUntilInterrupted' -count=1
```
Erwartet: eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/cli` beginnt, dahinter die Dauer; Exit 0.
```bash
go test ./internal/dev/mutants/ ./internal/cli/ -count=1 -covermode=set -coverprofile="$TEMP/task13.out"
```
Erwartet: Exit 0 und zwei Zeilen `ok`, je mit `coverage: 100.0% of statements` (gemessen im Wegwerf-Modul).
```bash
go tool cover -func="$TEMP/task13.out" > "$TEMP/task13-func.txt"
```
Erwartet: Exit 0, keine Ausgabe.
```bash
grep -vE '100\.0%$' "$TEMP/task13-func.txt"
```
Erwartet: jede Funktion beider Pakete 100.0%, also keine Ausgabe (gemessen im Wegwerf-Modul); Exit 1 von `grep` (kein Treffer) ist hier das erwartete Ergebnis. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.
```bash
grep -cE 'dev\.go:[0-9]+:[[:space:]]+(devCommand|devMutants|untilInterrupted)[[:space:]]+100\.0%$' "$TEMP/task13-func.txt"
```
Erwartet: `3`. Zeilennummern variieren mit dem Kommentar.
```bash
gofmt -l internal/dev/mutants internal/cli
```
Erwartet: keine Ausgabe.

- [ ] **Step 17: Doku** — `docs/en/cli-reference.md` und `docs/de/cli-reference.md`, §9, je direkt vor der Zeile ``### `loomux dev swap --dir <bin>` ``.

Englisch:
```markdown
### `loomux dev mutants <package>... [--only <name>] [--family a1|a2|a3|a4] [--workers <n>]`
Mutates the Go decisions of each package and reports which mutants its test suite does not notice — a port of ultra-brain's `tools/go_mutants.py`.

- **Families**: `a1` the whole `if` condition as `true` and as `false`; `a2` each operand of a top-level `&&` or `||` on its own; `a3` every comparison operator flipped (`==`/`!=`, each ordering against its neighbour), not inside comments or strings; `a4` the condition negated. `for` conditions are never mutated.
- **Mechanism**: each mutant reaches `go test -overlay <json> -count=1 -failfast -timeout 60s ./<package>/` through an overlay in a temporary directory; the working tree is never written. Each overlay directory is removed after its run, also after an error or Ctrl+C. `--workers` runs that many at once (default: half the processors, at least 1). `--only` keeps files whose name contains the text.
- **Report**: one line per mutant — `killed`, `SURVIVED` or `no mutant` (does not compile, or changes nothing) — then the sums and the survivors. A run that hits the time limit counts as killed.
- **Exit codes**: `0` after a complete round, survivors included; `2` for a usage error, a package without source files, or a suite that is not green before the first mutant; `1` when a run cannot be started or the round is interrupted with Ctrl+C.

```

Deutsch:
```markdown
### `loomux dev mutants <paket>... [--only <name>] [--family a1|a2|a3|a4] [--workers <n>]`
Mutiert die Go-Entscheidungen jedes Pakets und meldet, welche Mutanten seine Testsuite nicht bemerkt — ein Port von ultra-brains `tools/go_mutants.py`.

- **Familien**: `a1` die ganze `if`-Bedingung als `true` und als `false`; `a2` jeder Operand eines `&&` oder `||` auf oberster Ebene für sich; `a3` jeder Vergleichsoperator gekippt (`==`/`!=`, jede Ordnung gegen ihren Nachbarn), nicht in Kommentaren oder Zeichenketten; `a4` die Bedingung negiert. `for`-Bedingungen werden nie mutiert.
- **Mechanik**: Jeder Mutant erreicht `go test -overlay <json> -count=1 -failfast -timeout 60s ./<paket>/` über ein Overlay in einem temporären Verzeichnis; der Arbeitsbaum wird nie beschrieben. Jedes Overlay-Verzeichnis wird nach seinem Lauf entfernt, auch nach einem Fehler oder Strg+C. `--workers` fährt so viele Läufe gleichzeitig (Vorgabe: die Hälfte der Prozessoren, mindestens 1). `--only` behält Dateien, deren Name den Text enthält.
- **Bericht**: je Mutant eine Zeile — `killed`, `SURVIVED` oder `no mutant` (kompiliert nicht oder ändert nichts) — dann die Summen und die Überlebenden. Ein Lauf, der die Zeitgrenze reißt, gilt als getötet.
- **Exit-Codes**: `0` nach einer vollständigen Runde, auch mit Überlebenden; `2` bei einem Usage-Fehler, einem Paket ohne Quelldateien oder einer Suite, die vor dem ersten Mutanten nicht grün ist; `1`, wenn ein Lauf nicht gestartet werden kann oder die Runde mit Strg+C abgebrochen wird.

```

Die README-Dateien nennen `loomux dev mutants <pkg>` schon unter „Developer & Worktree Tools“ bzw. im deutschen Gegenstück; dieser Task fasst sie nicht an.

- [ ] **Step 18: Stagen und Tor**

```bash
git add internal/dev/mutants/mutants.go internal/dev/mutants/generate.go internal/dev/mutants/round.go internal/dev/mutants/generate_test.go internal/dev/mutants/mutants_test.go internal/dev/mutants/round_test.go internal/dev/mutants/gotest_test.go internal/cli/dev.go internal/cli/dev_test.go docs/en/cli-reference.md docs/de/cli-reference.md
```
Erwartet: keine Ausgabe, Exit 0.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0 (gofmt, vet, Tests mit Coverage, `covergate` ohne Ausgabe, Pilot-Binary mit `dev mutants` neu gebaut). Die drei `gotest_test.go`-Tests starten dabei den echten `go`-Befehl gegen ein Probemodul in `t.TempDir()` (rund 2 s).

- [ ] **Step 19: Commit**

```bash
git branch --show-current
```
Erwartet: der Branch des Arbeitsorts aus Task 0 (`sdd-1b-1` im Worktree, `master` im Hauptcheckout).
```bash
git rev-parse --short HEAD
```
Erwartet: der kurze Hash des Commits von Task 12, derselbe wie `git log -1 --format=%h` vor diesem Task.
```bash
git diff --cached --stat
```
Erwartet: genau die elf Dateien aus Step 18, keine weitere.
```bash
printf '%s\n' 'Port go_mutants.py as loomux dev mutants' '' 'The four mutation families follow the script'"'"'s rules and are checked' 'against its own output on a fixture. Each mutant reaches go test' 'through an overlay in a temporary directory, several at once, so the' 'working tree is never written. A suite that is not green before the' 'first mutant ends the round with exit 2; Ctrl+C ends it with exit 1' 'after every overlay directory is removed.' > "$TEMP/task13-commit.txt"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
git commit -F "$TEMP/task13-commit.txt"
```
Erwartet: Exit 0. Der Commit löst das Tor noch einmal aus; das ist gewollt. Danach eine Zeile `[<branch> <hash>] Port go_mutants.py as loomux dev mutants` und `11 files changed`.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität (`git config user.name` und `git config user.email`).
```bash
git log -1 --format=%B
```
Erwartet: genau die Nachricht aus der Datei, keine `Co-Authored-By`-Zeile.

- [ ] **Step 20: Bericht**

Abweichungen vom Vertrag (oben 1–8), dazu an den Controller:
- Task 14 übernimmt nach Ruling R3 den Wortlaut von `ErrBaselineRed` (`the suite is not green before the round`), `ErrNoSources` mit Exit 2 und statt eines Worts `not compiled` die Summenzeilen `N do not compile and are no mutants` / `N change nothing and are no mutants`. Der Task-13-Block des Vertrags nennt noch den alten Wortlaut; ihn nachzuziehen ist Sache des Controllers.
- `dev/mutants` importiert `internal/brain/pytext` (für `Strip`); Ruling R27 nimmt `dev/*` → `brain/pytext` in die Global Constraints auf.
- Doku-Schritt (Step 17) steht nicht in den „Files“ des Vertrags; er folgt AGENTS.md.
- Das Pre-Commit-Tor startet ab diesem Task in `gotest_test.go` echtes `go` (drei Läufe gegen ein Probemodul in `t.TempDir()`, rund 2 s; Vorbild `TestDevRecordCaseRecordsABinary`). Ohne `go` auf dem `PATH` überspringen sich die drei Tests, und `GoTest` fiele unter 100 %; das Tor braucht `go` ohnehin. Kein Umbau.
- Abbruch (Abweichung 8): Ctrl+C beendet die Runde mit Exit 1 und `loomux dev mutants: context canceled`, nachdem jedes Overlay-Verzeichnis entfernt ist (`TestDevMutantsCleansUpAnInterruptedRound`).

Befunde gegen die Quelle `go_mutants.py` (Mechanik, kein Verhalten eines `brain`-Befehls):
1. Der Mutant geht per `-overlay` in den Lauf, mehrere gleichzeitig; das Skript schreibt die Datei in den Baum und stellt sie danach wieder her.
2. Die Zeilenenden der übrigen Zeilen bleiben im Overlay erhalten; das Skript schreibt über `read_text` zurück und macht aus einer CRLF-Datei auch beim Wiederherstellen dauerhaft LF.
3. Ein Mutant, dessen Zeile gleich bleibt (`if true {` → `if true {`), wird nicht gestartet und als `no mutant` gezählt; das Skript liefe ihn und meldete `SURVIVED`.
4. `ifLine` nutzt Gos ASCII-`\s`, Python ein Unicode-`\s`; nur eine Einrückung aus Nicht-ASCII-Leerraum unterschiede sie, gofmt rückt mit Tabs ein.
5. Mehrere Pakete je Aufruf; Quellen und Grünheit aller Pakete werden vor dem ersten Mutanten geprüft.
6. `--family` wird geprüft (Exit 2); das Skript nimmt jede Zeichenkette und findet dann keinen Mutanten. `--oracle parity` entfällt.
7. Quellen in Byte-Reihenfolge der Namen (`os.ReadDir`); das Skript sortiert `WindowsPath`-Objekte ohne Groß-/Kleinschreibung.
8. Eine unlesbare Quelldatei endet mit Exit 1 und Go-Wortlaut; das Skript bricht mit Traceback ab.

**Paritätszeilen, die dieser Task schafft:** keine. `dev mutants` ist ein Entwicklerwerkzeug und kein Verhalten der Python-Referenz von `brain`; die Befunde oben gehören in den Bericht, nicht in `stufe-1b-1.md`.

---

### Task 14: Die Mutationsrunde der Stufe

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 13 sind committet.

**Files:**
- Modify: `docs/.superpowers/parity/stufe-1b-1.md` (neuer Abschnitt `## Mutationsrunde`)
- Create/Modify: `mutation_test.go` in jedem Paket, dessen Überlebende einen nachgereichten Test bekommen — `internal/brain/guard/mutation_test.go` besteht seit 1a und wird ergänzt, in den anderen Paketen entsteht die Datei neu (Beispiel unten: `internal/brain/search/mutation_test.go`)
- Kein Produktionscode. Zeigt ein Überlebender einen echten Fehler (die Python-Referenz antwortet anders als der Code), gilt Step 4c; die geänderte Produktionsdatei geht dann in denselben Commit.

**Interfaces:**
- Consumes (Task 13, Wortlaut nach Ruling R3):
  - Aufruf `loomux dev mutants <paket>... [--only <name>] [--family a1|a2|a3|a4] [--workers <n>]`; `--only` ist eine Teilzeichenkette des Dateinamens, `--workers` hat die Vorgabe `max(runtime.NumCPU()/2, 1)`.
  - Exit 0 nach einer vollständigen Runde, auch mit Überlebenden.
  - Exit 2, bevor ein Mutant läuft, mit einer stderr-Zeile `loomux dev mutants: <paket>: the suite is not green before the round` (`ErrBaselineRed`) oder `loomux dev mutants: <paket>: no source files` (`ErrNoSources`); Exit 2 auch bei einem Usage-Fehler.
  - Exit 1 mit `loomux dev mutants: <grund>`, wenn ein Lauf nicht startet, und mit `loomux dev mutants: context canceled` nach Ctrl+C; die Overlay-Verzeichnisse sind in beiden Fällen entfernt.
  - Bericht je Mutant eine Zeile in Erzeugungsreihenfolge, gleich bei jeder Arbeiterzahl, mit genau drei Urteilen: `[<i>/<n>] killed    (`, `[<i>/<n>] SURVIVED  (` und `[<i>/<n>] no mutant (`, dahinter `<familie>) <datei>:<zeile>  <war>  ->  <jetzt>`. Ein Urteil `not compiled` gibt es nicht; `no mutant` steht für einen Mutanten, der nicht kompiliert, und für einen, dessen Zeile gleich bleibt.
  - Je Paket danach eine Leerzeile und die Summenzeilen `<N> mutants over <paket>, oracle go`, `<c> do not compile and are no mutants`, nur bei u > 0 `<u> change nothing and are no mutants`, dann `<s> survived:` und je Überlebendem eine um zwei Leerzeichen eingerückte Zeile.
  - `TimedOut` zählt als getötet.
- Consumes (Task 8) für das Beispiel in Step 5: `search.Stale(stamp, now time.Time) bool`, `search.ReconcileInterval`.
- Produces: nur Tests, keine exportierten Bezeichner.

**Umfang der Runde (Spec „dev mutants“, Vertrag Task 14), dreizehn Pakete:** `internal/config`, `internal/cases`, `internal/brain/wiki`, `internal/brain/guard`, `internal/hooks` (Entscheidungspakete aus 1a) und `internal/brain/pytext`, `internal/brain/identity`, `internal/brain/graph`, `internal/brain/reader`, `internal/brain/catalog`, `internal/brain/privacy`, `internal/brain/status`, `internal/brain/search` (neu in 1b-1). Nicht: `internal/brain/check`, `internal/cli`, `internal/dev/**`.

**Laufzeit, vorab gerechnet** (Spec: Mutantenzahl × Testlauf, `NUMBER_OF_PROCESSORS` auf dieser Maschine 16 → Vorgabe 8 Arbeiter):

| Paket | Mutanten (Spec) | Testlauf (Spec) | CPU-Sekunden | Wand bei 8 Arbeitern |
|---|---:|---:|---:|---:|
| `internal/brain/guard` | 712 | 5,2 s | 3.702 | ≈ 463 s |
| `internal/hooks` | 706 | 10,0 s | 7.060 | ≈ 883 s |
| `internal/config` | 156 | 0,8 s | 125 | ≈ 16 s |
| `internal/cases` | 231 | 0,7 s | 162 | ≈ 20 s |
| `internal/brain/wiki` | 200 | 1,1 s | 220 | ≈ 28 s |
| Summe 1a-Pakete | 2.005 | | 11.269 | ≈ 1.409 s ≈ 23,5 min |

Die Zahl für `internal/config` stammt von vor Task 1 (`legacy.go` kommt dazu). Step 1 zählt alle dreizehn Pakete mit `_mutants_of` aus `go_mutants.py` neu, misst ihre Testläufe und rechnet die Wand der acht neuen Pakete nach derselben Formel (CPU-Sekunden = Mutanten × Testlauf, Wand = CPU-Sekunden ÷ 8); die Werte gehen in den Bericht. Das Shell-Werkzeug bricht nach 10 Minuten ab, `internal/hooks` allein braucht rund 15: **jeder Rundenlauf startet mit `run_in_background: true`**, und der nächste Schritt wartet auf die Fertigmeldung. Die Pakete laufen nacheinander, nie zwei Runden gleichzeitig — 16 parallele `go test` würden Läufe über die 60-s-Grenze drücken, und ein Zeitüberlauf zählt als getötet: die Runde meldete dann Tötungen, die kein Test bewirkt hat. Während einer Runde wird im Arbeitsbaum nichts geändert und kein Pre-Commit-Tor gestartet (der Overlay-Lauf liest die übrigen Dateien des Pakets von der Platte). Einzige zusätzliche Last ist die Baseline-Messung von `internal/hooks` in Step 2 (Ruling R15); sie schreibt nichts in den Baum.

Protokolle: die dreizehn Rundenlogs direkt unter `$TEMP/mutants-1b1/` (nur sie tragen dort die Endung `.log`, damit die Auswertung in Step 3 genau sie liest), Vergleichsläufe unter `$TEMP/mutants-1b1/compare/`, Nachprüfungen aus Step 6 unter `$TEMP/mutants-1b1/recheck/`.

- [ ] **Step 1: Ausgangslage prüfen, Mutanten zählen, Testläufe messen**

```bash
git branch --show-current
```
Erwartet: der Branch des Arbeitsorts aus Task 0 (`sdd-1b-1` im Worktree, `master` im Hauptcheckout).
```bash
git status --short
```
Erwartet: keine Ausgabe.
```bash
git rev-parse --short HEAD
```
Erwartet: der kurze Hash des Commits von Task 13; er kommt in den Kopf des Abschnitts aus Step 7.

Dann das Tor, das zugleich `bin/loomux.exe` mit `dev mutants` aus Task 13 baut und belegt, dass alle Suiten grün sind:
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0.
```bash
mkdir -p "$TEMP/mutants-1b1/compare" "$TEMP/mutants-1b1/recheck"
```
Erwartet: keine Ausgabe, Exit 0.

**Mutanten zählen (Ruling R15).** Das Skript zählt selbst: `_mutants_of` aus `go_mutants.py` über die Nicht-Test-`*.go` jedes Pakets, per `importlib` geladen, `main` nie gerufen. Je Paket eine Zeile `<mutanten> <unverändert> <paket>`; die zweite Zahl zählt die Mutanten, deren Zeile gleich bleibt:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "import sys,importlib.util as u; from pathlib import Path; s=u.spec_from_file_location('gm','C:/Users/micro/Documents/#GIT/loomux-src/ub/tools/go_mutants.py'); m=u.module_from_spec(s); sys.modules['gm']=m; s.loader.exec_module(m); [print(len(ms), sum(x.was == x.now for x in ms), p) for p in sys.argv[1:] for ms in [[x for f in sorted(Path(p).glob('*.go')) if not f.name.endswith('_test.go') for x in m._mutants_of(f)]]]" internal/config internal/cases internal/brain/wiki internal/brain/pytext internal/brain/identity internal/brain/graph internal/brain/reader internal/brain/catalog internal/brain/privacy internal/brain/status internal/brain/search internal/brain/guard internal/hooks > "$TEMP/mutants-1b1/expected.txt"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
cat "$TEMP/mutants-1b1/expected.txt"
```
Erwartet: dreizehn Zeilen in der Reihenfolge von Step 2. Fest stehen `231 0 internal/cases`, `200 0 internal/brain/wiki`, `712 0 internal/brain/guard` und `706 0 internal/hooks` — gemessen am 2026-09-15 auf `e4bf7b5` mit genau diesem Befehl, gleich den Zahlen der Spec. `internal/config` zählte dort `156 0`; die Zeile nach Task 1 ist der Sollwert für Step 3. Die acht neuen Pakete zählt dieser Befehl zum ersten Mal. Weicht eine der vier festen Zeilen ab, hat ein Task der Stufe ein 1a-Paket geändert: Befund an den Controller, bevor die Runde startet.

**Testläufe messen.** Ein Paket nach dem anderen, damit die Dauern einander nicht verfälschen:
```bash
go test -count=1 -p 1 ./internal/config/ ./internal/cases/ ./internal/brain/wiki/ ./internal/brain/pytext/ ./internal/brain/identity/ ./internal/brain/graph/ ./internal/brain/reader/ ./internal/brain/catalog/ ./internal/brain/privacy/ ./internal/brain/status/ ./internal/brain/search/ ./internal/brain/guard/ ./internal/hooks/ > "$TEMP/mutants-1b1/durations.txt" 2>&1
```
Erwartet: Exit 0.
```bash
grep -cE '^ok[[:space:]]+github\.com/xidus90/loomux/internal/[a-z/]+[[:space:]]+[0-9.]+s$' "$TEMP/mutants-1b1/durations.txt"
```
Erwartet: `13`. Die Dauer d jeder `ok`-Zeile ist der Testlauf des Pakets; mit N aus `expected.txt` ergibt N × d die CPU-Sekunden und N × d ÷ 8 die Wand bei 8 Arbeitern. Die Dauer enthält keine Übersetzung, die Runde übersetzt das Paket je Mutant neu: Die Schätzung ist eine untere Schranke. Alle dreizehn Werte gehen in den Bericht (Step 9).

- [ ] **Step 2: Die Runde, ein Paket je Aufruf, jeder im Hintergrund** (Reihenfolge klein vor groß, damit ein Fehler im Werkzeug früh auffällt). Nach jedem Aufruf die Fertigmeldung abwarten und den Exit-Code lesen; einzige Ausnahme ist `internal/hooks`: nach dessen Start im Hintergrund zuerst die Baseline-Messung unten fahren, dann die Fertigmeldung abwarten. Erwartet je Aufruf: Exit 0, und das Log endet mit den Summenzeilen aus Step 3. Jeder andere Exit hält die Runde an; der Grund steht in der letzten Zeile des Logs (`tail -n 1` auf das Log des Aufrufs), der Befund geht an den Controller, und es wird nicht weitergezählt:
  - Exit 2 mit `loomux dev mutants: <paket>: the suite is not green before the round` — die Suite ist schon ohne Mutant rot.
  - Exit 2 mit `loomux dev mutants: <paket>: no source files` — der Paketpfad trifft kein Verzeichnis mit Quelldateien.
  - Exit 1 mit `loomux dev mutants: <grund>` — ein Lauf ließ sich nicht starten; `context canceled` heißt, die Runde wurde abgebrochen.

```bash
bin/loomux.exe dev mutants internal/config > "$TEMP/mutants-1b1/config.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/cases > "$TEMP/mutants-1b1/cases.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/wiki > "$TEMP/mutants-1b1/brain-wiki.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/pytext > "$TEMP/mutants-1b1/brain-pytext.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/identity > "$TEMP/mutants-1b1/brain-identity.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/graph > "$TEMP/mutants-1b1/brain-graph.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/reader > "$TEMP/mutants-1b1/brain-reader.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/catalog > "$TEMP/mutants-1b1/brain-catalog.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/privacy > "$TEMP/mutants-1b1/brain-privacy.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/status > "$TEMP/mutants-1b1/brain-status.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/search > "$TEMP/mutants-1b1/brain-search.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/brain/guard > "$TEMP/mutants-1b1/brain-guard.log" 2>&1
```
```bash
bin/loomux.exe dev mutants internal/hooks > "$TEMP/mutants-1b1/hooks.log" 2>&1
```

  **Baseline von `internal/hooks` während seiner Runde (Ruling R15).** Ein Testlauf von `internal/hooks` dauert allein 10 s; unter acht parallelen Läufen darf er die 60-s-Grenze nicht erreichen, sonst täuscht ein Zeitüberlauf eine Tötung vor. Gemessen wird, sobald das Log die erste Urteilszeile trägt, denn dann laufen alle acht Arbeiter:
```bash
grep -c '^\[' "$TEMP/mutants-1b1/hooks.log"
```
  Erwartet: eine Zahl ≥ 1 und Exit 0. Bei `0` (dann mit Exit 1) läuft noch der Baseline-Lauf der Runde; denselben Befehl wiederholen, bis die Zahl ≥ 1 ist (kein `sleep` im Vordergrund).
```bash
time go test -count=1 ./internal/hooks/
```
  Erwartet: eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/hooks` beginnt, und eine Zeile `real` mit `0m` vor den Sekunden, also unter 60 s. Der Wert kommt in den Kopf des Abschnitts aus Step 7. Die Messung läuft als neunter `go test` neben den acht Arbeitern und damit unter etwas mehr Last als ein Mutantenlauf. Zeigt `real` 60 s oder mehr, läuft `internal/hooks` nach der Fertigmeldung seiner Runde noch einmal mit vier Arbeitern (im Hintergrund):
```bash
bin/loomux.exe dev mutants internal/hooks --workers 4 > "$TEMP/mutants-1b1/compare/hooks-workers4.log" 2>&1
```
  Erwartet: Exit 0.
```bash
grep -E '^\[[0-9]+/[0-9]+\] ' "$TEMP/mutants-1b1/hooks.log" > "$TEMP/mutants-1b1/compare/hooks-verdicts.txt"
```
  Erwartet: keine Ausgabe, Exit 0.
```bash
grep -E '^\[[0-9]+/[0-9]+\] ' "$TEMP/mutants-1b1/compare/hooks-workers4.log" > "$TEMP/mutants-1b1/compare/hooks-workers4-verdicts.txt"
```
  Erwartet: keine Ausgabe, Exit 0.
```bash
diff "$TEMP/mutants-1b1/compare/hooks-verdicts.txt" "$TEMP/mutants-1b1/compare/hooks-workers4-verdicts.txt"
```
  Erwartet: keine Ausgabe, Exit 0. Jede Zeile, die `diff` zeigt, ist ein Mutant, den der Achter-Lauf anders beurteilt hat als der Vierer-Lauf: Dann zählen für `internal/hooks` in Step 3 und Step 7 die Werte aus `compare/hooks-workers4.log`, und der Unterschied ist ein Befund an den Controller.

- [ ] **Step 3: Auswerten.** Erst die Summen aller dreizehn Logs:

```bash
grep -E '^[0-9]+ (mutants over |do not compile and are no mutants$|change nothing and are no mutants$|survived:$)' "$TEMP/mutants-1b1/"*.log
```
Erwartet: je Log drei oder vier Zeilen, jeweils mit dem Logpfad davor, in dieser Reihenfolge: `<N> mutants over <paket>, oracle go`, `<c> do not compile and are no mutants`, nur bei u > 0 `<u> change nothing and are no mutants`, `<s> survived:`.

  **Gegen die Zählung aus Step 1 (Ruling R15):** N jedes Logs ist die erste Zahl von `expected.txt` für dasselbe Paket — für `internal/brain/guard`, `internal/hooks`, `internal/cases` und `internal/brain/wiki` also `712`, `706`, `231` und `200`, für `internal/config` und die acht neuen Pakete die in Step 1 gezählte Zahl. Eine Zeile `change nothing` steht genau bei den Paketen, deren zweite Zahl in `expected.txt` nicht `0` ist, und nennt dieselbe Zahl. Jede Abweichung ist ein Befund gegen Task 13, bevor ein Überlebender bewertet wird; die Runde gilt dann nicht, und der Task hält an.

Die Getöteten je Log:
```bash
grep -cE '^\[[0-9]+/[0-9]+\] killed    \(' "$TEMP/mutants-1b1/"*.log
```
Erwartet: je Log eine Zeile `<logpfad>:<k>`, und für jedes Paket gilt k + s + c + u = N.

Die Überlebenden:
```bash
grep -E '^\[[0-9]+/[0-9]+\] SURVIVED  \(' "$TEMP/mutants-1b1/"*.log
```
Erwartet: je Log genau s Zeilen (bei s = 0 keine), jede mit dem Logpfad davor; hat kein Paket einen Überlebenden, bleibt die Ausgabe leer, und Exit 1 von `grep` ist dann das erwartete Ergebnis. Jede Zeile wird eine Arbeitskarte mit Paket (aus dem Lognamen), Datei, Zeile, Familie, `war` und `jetzt`; der Text hinter `SURVIVED  ` heißt auf der Karte `<mutant>` und ist die Form von `Mutant.String()`.

  **Gegenprobe gegen Scheintötungen:** Alle Arbeiter lesen denselben Baum; ein Test, der unter Parallelität an einer festen Datei, einem `testlock` oder der 60-s-Grenze scheitert, zählt als `killed` und verdeckt einen Überlebenden. Darum `internal/config` einmal mit einem Arbeiter (rund 2 Minuten, im Hintergrund):
```bash
bin/loomux.exe dev mutants internal/config --workers 1 > "$TEMP/mutants-1b1/compare/config-workers1.log" 2>&1
```
  Erwartet: Exit 0.
```bash
grep -E '^\[[0-9]+/[0-9]+\] ' "$TEMP/mutants-1b1/config.log" > "$TEMP/mutants-1b1/compare/config-verdicts.txt"
```
  Erwartet: keine Ausgabe, Exit 0.
```bash
grep -E '^\[[0-9]+/[0-9]+\] ' "$TEMP/mutants-1b1/compare/config-workers1.log" > "$TEMP/mutants-1b1/compare/config-workers1-verdicts.txt"
```
  Erwartet: keine Ausgabe, Exit 0.
```bash
diff "$TEMP/mutants-1b1/compare/config-verdicts.txt" "$TEMP/mutants-1b1/compare/config-workers1-verdicts.txt"
```
  Erwartet: keine Ausgabe, Exit 0 — beide Läufe beurteilen jeden Mutanten gleich (der Bericht steht bei jeder Arbeiterzahl in Erzeugungsreihenfolge). Zeigt `diff` Zeilen, zählen die Achter-Läufe nicht: Jedes Paket läuft dann mit `--workers 1` erneut (Laufzeit ×8, weiter im Hintergrund, Logs `compare/<logname>-workers1.log`), Step 3 wertet diese Logs aus, und der Unterschied ist ein Befund an den Controller.

- [ ] **Step 4: Je Überlebendem entscheiden**, in dieser Reihenfolge:
  - **a) Ein Test fehlt:** Die Referenz (Python-Quelle, gemessener Wert oder die Bibliotheksfunktion, die der Code nachbildet) legt eine Eingabe fest, unter der Original und Mutant verschieden antworten. → Step 5.
  - **b) Der Code kann den Unterschied nicht sehen:** Es gibt keine Eingabe, unter der beide verschieden antworten — etwa weil eine frühere Prüfung den Fall schon ausschließt oder beide Zweige denselben Wert liefern. → Eine Zeile in Step 7 mit genau dieser Begründung: welche Zeile den Fall vorher ausschließt bzw. warum beide Zweige gleich sind. „Unwichtig“ ist keine Begründung.
  - **c) Ein echter Fehler:** Die Referenz antwortet anders als der unmutierte Code. → Fest in dieser Folge:
    1. Test nach Step 5 schreiben, mit dem an der Referenz gemessenen Wert.
    2. `go test ./<paket>/ -run '^<Testname>$' -count=1` — Erwartet: eine Zeile `--- FAIL: <Testname>` und Exit 1, ohne Mutant.
    3. Die kleinste Änderung am Produktionscode, die den Wert der Referenz liefert.
    4. Derselbe Befehl — Erwartet: eine Zeile, die mit `ok  	github.com/xidus90/loomux/<paket>` beginnt; Exit 0.
    5. Die Coverage-Prüfung aus Step 6.
    6. Step 6 für den Mutanten.
    7. Die geänderte Produktionsdatei geht in Step 8 in denselben Commit; der Bericht nennt den Fehler als Befund, eine Paritätszeile entsteht nicht.

- [ ] **Step 5: Den fehlenden Test schreiben** — in `<paket>/mutation_test.go`, Form wie `internal/brain/guard/mutation_test.go` aus 1a, fest in dieser Gestalt:
  1. Paket `<name>_test`; nur wenn der Fall ein unexportiertes Symbol braucht, das interne Paket `<name>`.
  2. Eine neue Datei beginnt mit dem Kopfkommentar des Beispiels unten, wortgleich. In `internal/brain/guard/mutation_test.go` kommt stattdessen der Absatz unten unter die bestehenden Tests.
  3. Je Überlebendem ein Test, benannt nach dem Fall, den er fragt, nicht nach dem Mutanten.
  4. Der erste Kommentar im Test belegt die Erwartung: entweder Quelldatei der Referenz mit Zeilennummer und der zitierten Zeile, oder der Messbefehl mit seinem Ergebnis.
  5. Die Zusicherung prüft genau die Eingabe, unter der Original und Mutant verschieden antworten.

  Erwartete Werte werden an der Referenz gemessen oder aus der Quelle mit Zeilennummer gelesen, nie aus dem Go-Code abgeleitet. Der Messbefehl hat immer diese Form, mit einem nur lesenden Ausdruck von der Arbeitskarte:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" python -c "<lesender Ausdruck>"
```
  Erwartet: Exit 0 und der Wert, der in den Kommentar des Tests kommt.

  In `internal/brain/guard/mutation_test.go` kommt unter die bestehenden Tests ein Absatz:
```go
// The tests the mutation round of stage 1b-1 added to the ones above: the
// same kind of finding, one round later.
```

  **Beispiel, vollständig** — trifft zu, wenn die Runde in `internal/brain/search/stamp.go` den a3-Mutanten `now.Sub(stamp) >= ReconcileInterval` → `now.Sub(stamp) > ReconcileInterval` überleben lässt. Neue Datei `internal/brain/search/mutation_test.go`:

```go
package search_test

import (
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/search"
)

// The tests the mutation round of stage 1b-1 asked for. They are gathered
// here rather than scattered because what they have in common is how they
// were found: each one is the case a surviving mutant proved nobody had
// asked for. Every expectation below was read from the Python reference or
// measured against it, never derived from the Go code that has to satisfy
// it.

func TestExactlyOneIntervalAfterTheStampIsAlreadyStale(t *testing.T) {
	// `_stale` (src/brain/core.py:384-385) is
	// `datetime.now(UTC) - stamp >= RECONCILE_INTERVAL`: the 24th hour
	// itself counts. One microsecond earlier is the last fresh instant --
	// Python's datetime resolution, so no finer instant exists on that side.
	stamp := time.Date(2026, 9, 13, 19, 51, 12, 255414000, time.UTC)
	if !search.Stale(stamp, stamp.Add(search.ReconcileInterval)) {
		t.Fatal("a stamp exactly 24 h old must be stale")
	}
	if search.Stale(stamp, stamp.Add(search.ReconcileInterval-time.Microsecond)) {
		t.Fatal("a stamp one microsecond short of 24 h must not be stale")
	}
}
```
  (Gegen Stubs mit den Vertragssignaturen kompiliert; mit `>` statt `>=` per `go test -overlay` eingesetzt endet er mit `mutation_test.go:24: a stamp exactly 24 h old must be stale` und `FAIL`, mit `>=` mit `ok`.)

- [ ] **Step 6: Grün ohne, rot mit dem Mutanten zeigen** — je nachgereichtem Test, mit den Feldern der Arbeitskarte. Für das Beispiel aus Step 5 lauten die Befehle so:

  Grün ohne Mutant:
```bash
go test ./internal/brain/search/ -run '^TestExactlyOneIntervalAfterTheStampIsAlreadyStale$' -count=1
```
  Erwartet: eine Zeile, die mit `ok  	github.com/xidus90/loomux/internal/brain/search` beginnt, dahinter die Dauer; Exit 0.

  Rot mit dem Mutanten — dieselbe Datei und Familie noch einmal durch die Runde (im Hintergrund, falls das Paket groß ist):
```bash
bin/loomux.exe dev mutants internal/brain/search --only stamp.go --family a3 > "$TEMP/mutants-1b1/recheck/brain-search-stamp-a3.log" 2>&1
```
  Erwartet: Exit 0.
```bash
grep -F 'killed    (a3) stamp.go:<zeile>  return now.Sub(stamp) >= ReconcileInterval  ->  return now.Sub(stamp) > ReconcileInterval' "$TEMP/mutants-1b1/recheck/brain-search-stamp-a3.log"
```
  Erwartet: genau eine Zeile `[<i>/<n>] killed    (a3) stamp.go:<zeile>  return now.Sub(stamp) >= ReconcileInterval  ->  return now.Sub(stamp) > ReconcileInterval`. `<zeile>` ist die Zeilennummer aus der `SURVIVED`-Zeile der Arbeitskarte; `<i>/<n>` weicht von der Runde ab, weil `--only` und `--family` weniger Mutanten erzeugen. Diese Zeile wird im Bericht zitiert.
```bash
grep -cF 'SURVIVED  (a3) stamp.go:<zeile>  return now.Sub(stamp) >= ReconcileInterval  ->  return now.Sub(stamp) > ReconcileInterval' "$TEMP/mutants-1b1/recheck/brain-search-stamp-a3.log"
```
  Erwartet: `0`, mit Exit 1 von `grep` (kein Treffer) — das ist hier das erwartete Ergebnis.

  Für jeden anderen Überlebenden gilt dieselbe Folge mit den Feldern der Arbeitskarte: `go test ./<paket>/ -run '^<Testname>$' -count=1` (Erwartet: `ok`, Exit 0), `bin/loomux.exe dev mutants <paket> --only <datei> --family <familie>` mit Ausgabe nach `$TEMP/mutants-1b1/recheck/<logname>-<datei>-<familie>.log` (Erwartet: Exit 0), `grep -F 'killed    <mutant>'` auf dieses Log (Erwartet: genau eine Zeile) und `grep -cF 'SURVIVED  <mutant>'` (Erwartet: `0` mit Exit 1).

  Coverage bleibt unberührt, solange nur `_test.go` dazukommt. Hat Step 4c Produktionscode geändert, prüft dieselbe Messung wie das Tor:
```bash
go test ./... -count=1 -covermode=set -coverpkg=./... -coverprofile="$TEMP/mutants-1b1/recheck/coverage.out"
```
  Erwartet: Exit 0, jedes Paket `ok` oder ohne Testdateien.
```bash
go tool cover -func="$TEMP/mutants-1b1/recheck/coverage.out" > "$TEMP/mutants-1b1/recheck/func.txt"
```
  Erwartet: keine Ausgabe, Exit 0.
```bash
grep -E '^github\.com/xidus90/loomux/<paket>/[^/]+\.go:' "$TEMP/mutants-1b1/recheck/func.txt"
```
  Erwartet: jede Funktion des Pakets 100.0%, also endet jede gezeigte Zeile auf `100.0%`; eine Funktion darunter trägt `//coverage:exempt` mit Grund. Zeigt eine Zeile weniger ohne diese Ausnahme, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.
```bash
go run ./cmd/loomux dev covergate --profile "$TEMP/mutants-1b1/recheck/coverage.out"
```
  Erwartet: keine Ausgabe, Exit 0.

- [ ] **Step 7: Abschnitt in der Paritätsliste** — an `docs/.superpowers/parity/stufe-1b-1.md` anhängen, Werte aus den Logs:

```markdown
## Mutationsrunde

Gelaufen am <Datum aus `date +%Y-%m-%d`> mit `bin/loomux.exe dev mutants <paket>` (Task 13),
Commit `<Hash aus Step 1>`, 8 Arbeiter, Protokolle unter `$TEMP/mutants-1b1/`. Ein Zeitüberlauf
zählt als getötet. Baseline von `internal/hooks` während seiner Runde: `real` <Wert aus Step 2>.
Mutantenzahlen gleich `_mutants_of` aus `go_mutants.py` (Step 1).

| Paket | Mutanten | getötet | überlebt | nicht kompiliert | kein Mutant |
|---|---:|---:|---:|---:|---:|
| internal/config | N | k | s | c | u |
| internal/cases | N | k | s | c | u |
| internal/brain/wiki | N | k | s | c | u |
| internal/brain/pytext | N | k | s | c | u |
| internal/brain/identity | N | k | s | c | u |
| internal/brain/graph | N | k | s | c | u |
| internal/brain/reader | N | k | s | c | u |
| internal/brain/catalog | N | k | s | c | u |
| internal/brain/privacy | N | k | s | c | u |
| internal/brain/status | N | k | s | c | u |
| internal/brain/search | N | k | s | c | u |
| internal/brain/guard | N | k | s | c | u |
| internal/hooks | N | k | s | c | u |

| Datei:Zeile | Familie | war → jetzt | Erledigung |
|---|---|---|---|
| internal/brain/search/stamp.go:<zeile> | a3 | `>=` → `>` | Test `TestExactlyOneIntervalAfterTheStampIsAlreadyStale`, danach `killed` |
```
  Die erste Tabelle hat genau dreizehn Zeilen, in der Reihenfolge von Step 2. Ihre Buchstaben werden durch die Zahlen aus Step 3 ersetzt: N aus `<N> mutants over`, k aus dem `grep -c` der Getöteten, s aus `<s> survived:`, c aus `<c> do not compile and are no mutants`, u aus `<u> change nothing and are no mutants` (fehlt die Zeile: `0`). Hat Step 2 den Vierer-Lauf von `internal/hooks` gezählt, stammen dessen Zahlen aus `compare/hooks-workers4.log`, und der Kopf nennt „`internal/hooks` mit 4 Arbeitern“. Die zweite Tabelle hat je Überlebendem eine Zeile. „Erledigung“ ist entweder der Testname mit dem Nachweis `killed` aus Step 6 oder die Begründung aus Step 4b. Keine Zeile bleibt ohne Erledigung.

- [ ] **Step 8: Stagen, Tor und Commit**

Grundlage des `git add` sind die eigenen Dateien, wie `git status` sie zeigt:
```bash
git status --porcelain -- '*mutation_test.go' docs/.superpowers/parity/stufe-1b-1.md
```
Erwartet: ` M docs/.superpowers/parity/stufe-1b-1.md`, dazu je neu angelegter Datei eine Zeile `?? <paket>/mutation_test.go` und, falls ergänzt, ` M internal/brain/guard/mutation_test.go`. Der Pfadausdruck `'*mutation_test.go'` trifft auch Dateien in Unterverzeichnissen (geprüft: `git ls-files -- '*mutation_test.go'` nennt heute `internal/brain/guard/mutation_test.go`).
```bash
git status --porcelain
```
Erwartet: dieselben Zeilen, dazu höchstens die in Step 4c geänderten Produktionsdateien als ` M <paket>/<datei>.go`; sonst nichts.
```bash
git add docs/.superpowers/parity/stufe-1b-1.md <jeder weitere Pfad aus den beiden Listen>
```
Erwartet: keine Ausgabe, Exit 0. Mit nur dem Beispiel aus Step 5 lautet der Befehl `git add docs/.superpowers/parity/stufe-1b-1.md internal/brain/search/mutation_test.go`.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0.
```bash
git branch --show-current
```
Erwartet: derselbe Branch wie in Step 1.
```bash
git rev-parse --short HEAD
```
Erwartet: derselbe Hash wie in Step 1 (der Commit von Task 13).
```bash
git diff --cached --stat
```
Erwartet: genau die Pfade aus dem `git add`, keine weitere Datei.
```bash
printf '%s\n' 'Hold the decisions of stage 1b-1 to a mutation round' '' 'Every survivor of loomux dev mutants over the 1a decision packages and' 'the new brain packages has a test that kills it or a recorded reason' 'why the code cannot tell the difference.' > "$TEMP/commit-task14.txt"
```
Erwartet: keine Ausgabe, Exit 0.
```bash
git commit -F "$TEMP/commit-task14.txt"
```
Erwartet: Exit 0. Der Commit löst das Tor noch einmal aus; das ist gewollt. Danach eine Zeile `[<branch> <hash>] Hold the decisions of stage 1b-1 to a mutation round`.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität (`git config user.name` und `git config user.email`).
```bash
git log -1 --format=%B
```
Erwartet: genau die Nachricht aus der Datei, keine `Co-Authored-By`-Zeile.

- [ ] **Step 9: Bericht** — nennt: je Paket die Zählung aus Step 1 (`expected.txt`), den Testlauf und die geschätzte Wand, die Summen der Runde und das Ergebnis des Abgleichs aus Step 3; die Baseline-Zeit von `internal/hooks` aus Step 2 und, falls gelaufen, den `diff` des Vierer-Laufs; den `diff` der Gegenprobe über `internal/config`; jeden Überlebenden mit Erledigung, die `killed`-Zeilen aus Step 6, jeden Befund nach Step 4c. **Paritätszeilen, die dieser Task schafft:** keine — die Runde ändert kein Verhalten gegenüber der Referenz. Ein nach Step 4c behobener Fehler schafft keine Zeile (loomux folgt danach der Referenz), er ist ein Befund.

---

### Task 15: Messen

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 14 sind committet.

**Anhaltepunkte.** Step 1 schreibt der Mensch. Step 2 hält an, wenn ein brain-Daemon läuft; den stoppt der Mensch. Vor Step 7 und vor dem `qmd mcp stop` in Step 11 sagt der Controller dem Nutzer Bescheid, weil `qmd mcp stop` auch einen Daemon beendet, den der Mensch gerade nutzt. Ein Subagent, der diesen Task fährt, hält an jedem dieser Punkte an und meldet sich beim Controller.

**Files:**
- Modify (**Mensch**): `%LOCALAPPDATA%\loomux\registry.toml` (Step 1, anhängen)
- Create: `testdata/bench/1b-1-brain.json`
- Create: `internal/brain/search/bench_test.go` (Ergänzung zum Vertrag: der eigens ausgewiesene Anteil fürs Lesen der Register braucht eine Messung ohne qmd)
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `docs/.superpowers/parity/stufe-1b-1.md` (neuer Abschnitt `## Live-Vergleich`)
- Modify, nur wenn Step 10 greift: die Quelldatei der umgestellten Paketvariable und ein `_internal_test.go` im selben Paket

**Interfaces:**
- Consumes (1a, `internal/dev/benchhooks`): JSON-Form `[{"name", "dir", "stdin", "mode", "steps": [{"argv": ["<programm>", "<argument>"]}]}]` (Tags aus `benchhooks.go`), `mode` `single`|`seq`|`par`; `loomux dev bench-hooks <cases.json> [-n 20]` schreibt die Tabelle `| case | cold (1st run) | warm median | warm min | warm max | exit codes |` auf stdout; ein Exit ≠ 0 eines Schritts ist ein Messwert, kein Abbruch.
- Consumes (Task 1): `config.ManifestDir(area config.Area, stateDir string) string`; `config.Area` wird nur weitergereicht.
- Consumes (Task 3): `privacy.VisibleAreas(registryDir, legacyDir, scope string, ch privacy.Channel) ([]privacy.VisibleArea, error)`; `privacy.VisibleArea{Area config.Area; Manifest *config.Manifest}`, gelesen wird `Area`; `privacy.ChannelLocal`.
- Consumes (Task 4): `identity.ReadIdentities(path string) (map[string]identity.Identity, error)`; ein fehlendes Register ergibt eine leere Map.
- Consumes (Task 8): `search.ExecuteSearch(query, scope string, profile search.Profile, n int, channel privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) (*search.SearchAnswer, error)`; `search.NewFakePort() *search.FakePort`, gesetzt wird das Feld `Results []search.ScriptedSearch`; `search.ScriptedSearch{Hits []search.SearchHit; Err error}`; `search.SearchHit{Collection string; Scope string; Relative string; Line int; Title string; Snippet string; Score float64; ContentKey string}`, gesetzt werden `Collection`, `Relative`, `Line`, `Title`, `Score`; `search.CollectionName(scope string) string`; `search.ProfileFast`; der Wortlaut von `search.WarmingNotice`, den Step 7 mit `note: ` davor auf stderr erwartet; die Entfernung des qmd-Präfixes `N: ` in `translateReply`, auf der die Vorhersage für `full` in Step 11 ruht.
- Consumes (Task 10): `loomux brain search|status`, Registry aus `config.StateDir()`, Legacy aus `config.LegacyBrainDirUntilStage3()`.
- Produces (nur Tests): `BenchmarkVisibleAreasOfTheRealRegistry`, `BenchmarkRegistersOfTheRealRegistry`, `BenchmarkExecuteSearchWithoutTheEngine` in `internal/brain/search/bench_test.go`; Umgebungsvariablen `LOOMUX_BENCH_REGISTRY`, `LOOMUX_BENCH_LEGACY`.

**Vorab geprüft (2026-09-15, nur gelesen):** `%LOCALAPPDATA%\loomux\registry.toml` enthält genau einen `[[area]]` (`project/loomux`); `%LOCALAPPDATA%\brain\registry.toml` zehn, keiner davon in der loomux-Registry. Jeder schreibbare der zehn trägt ein Altmanifest mit `[area] scope` (`brain-knowledge/.brain.toml`, `92 Engineering/python/.brain.toml`, `92 Engineering/craft/.brain.toml`, `ultra-brain/.brain.toml`, `ecoflow/.brain.toml`, `91 Projekte/.brain.toml`, `ultraloom/.ultra-brain/config.toml`), jeder schreibgeschützte (`project/space`, `project/iam-wiki`, `project/obsidian-ai`) ein `.brain.toml` samt `graph.json`, `index.md`, `_identities.tsv` unter `%LOCALAPPDATA%\brain\areas\<flat>`; der Stempel `%LOCALAPPDATA%\brain\maintenance\last-run.txt` lautet `2026-09-13T19:51:12.255414+00:00`. `loomux brain` löst also alle Manifeste über `ReadAreaManifestUntilStage4` auf. Die zwölf `flat`-Namen nach dem Anhängen sind verschieden, `signpost` trägt nur `knowledge` — die Registry-Prüfung der Schreibschranke (`internal/brain/guard/registry.go`) nimmt die Datei an.

- [ ] **Step 1 (Mensch): die Bereiche von ultra-brain in die loomux-Registry übernehmen.** Der folgende Block wird **unverändert** an das Ende von `%LOCALAPPDATA%\loomux\registry.toml` angehängt, hinter den bestehenden Block `project/loomux` und hinter den Worktree-Block aus Task 0. Er ist `%LOCALAPPDATA%\brain\registry.toml` Zeile für Zeile, Kommentar über `hub` eingeschlossen:

```toml
[[area]]
scope    = "knowledge"
path     = "C:/Users/micro/Documents/#GIT/brain-knowledge"
wiki     = "C:/Users/micro/Documents/#GIT/brain-knowledge/90 Wiki"
signpost = true
shared   = true

[[area]]
scope  = "engineering/python"
path   = "C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/python"
wiki   = "C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/python"
shared = true

[[area]]
scope  = "engineering/craft"
path   = "C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/craft"
wiki   = "C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/craft"
shared = true

[[area]]
scope = "project/ultra-brain"
path  = "C:/Users/micro/Documents/#GIT/ultra-brain"
wiki  = "C:/Users/micro/Documents/#GIT/ultra-brain/docs/wiki"
workspace = true

[[area]]
scope    = "project/space"
path     = "C:/Users/micro/Documents/#GIT/space"
wiki     = "C:/Users/micro/Documents/#GIT/brain-knowledge/91 Projekte/space"
readonly = true
workspace = true

[[area]]
scope    = "project/iam-wiki"
path     = "C:/Users/micro/Documents/#GIT/iam_wiki"
wiki     = "C:/Users/micro/Documents/#GIT/brain-knowledge/91 Projekte/iam-wiki"
readonly = true

[[area]]
scope    = "project/obsidian-ai"
path     = "C:/Users/micro/Documents/#GIT/#Obsidian/AI"
wiki     = "C:/Users/micro/Documents/#GIT/brain-knowledge/91 Projekte/obsidian-ai"
readonly = true

[[area]]
scope = "project/ecoflow"
path  = "C:/Users/micro/Documents/#GIT/ecoflow"
wiki  = "C:/Users/micro/Documents/#GIT/brain-knowledge/91 Projekte/ecoflow"
workspace = true

# Der Hub-Ordner des Vaults ist ein eigener Bereich, damit die KI die
# Hub-Zeiger an ihrem vertrauten Ort anlegen darf: `91 Projekte/<name>.md`
# liegt sonst außerhalb jedes schreibbaren Baums und die scharfe Schranke
# verweigert das Anlegen.
#
# Bewusst in Kauf genommen: dieser Wiki-Pfad umschließt die Wiki-Pfade von
# project/space, project/iam-wiki, project/obsidian-ai und project/ecoflow.
# Der Wiki-Verbund (Spec 3.2) nennt eine solche Überschneidung einen Fehler.
# Die Belastung ist für diese vier null, weil sie über ihren eigenen Bereich
# ohnehin schreibbar sind. Neu schreibbar wird zweierlei: die Wurzelebene von
# `91 Projekte/` selbst — Hub-Zeiger, Bundle-Katalog, graph.json,
# _identities.tsv —, und das ist der Zweck dieses Bereichs; und außerdem, aber
# auch nur, das verwaiste `91 Projekte/ultraloom/`.
[[area]]
scope = "hub"
path  = "C:/Users/micro/Documents/#GIT/brain-knowledge/91 Projekte"
wiki  = "C:/Users/micro/Documents/#GIT/brain-knowledge/91 Projekte"

[[area]]
scope = "project/ultraloom"
path = "C:/Users/micro/Documents/#GIT/ultraloom"
wiki = "C:/Users/micro/Documents/#GIT/ultraloom/docs/wiki"
workspace = true
```

  **Folgen, die der Mensch vor dem Anhängen kennt:** Die Schreibschranke jeder loomux-Sitzung öffnet sich damit für diese Bäume, wie früher `brain guard` (Spec, Mensch-Schritte). Das Ergebnis des 1a-Rauchtests „Write an `C:/Users/micro/Documents/#GIT/ultraloom/x.md` → verweigert" kehrt sich um: `project/ultraloom` ist `workspace = true`, der Write ist danach erlaubt. Und jeder registrierte Bereich braucht ab jetzt ein lesbares Manifest — fehlt eines, weil ein Worktree gelöscht, sein Block aber stehen gelassen wurde, endet **jeder** `loomux brain`-Befehl mit `error:` und Exit 1, wie `brain-mcp` es tut.

  Prüfen, dass die Datei gelesen wird und die Schranke weiter arbeitet (kein Write, nur Lesen):
```bash
grep -c '^\[\[area\]\]' "$LOCALAPPDATA/loomux/registry.toml"
```
  Erwartet: `12` mit Worktree-Block aus Task 0, sonst `11`.
```bash
env | grep '^LOOMUX_'
```
  Erwartet: keine Ausgabe (weder `LOOMUX_STATE_DIR` noch `LOOMUX_LEGACY_BRAIN_DIR` — die Messung läuft gegen die echten Verzeichnisse, anders als die 1a-Messung mit `$TEMP/loomux-bench-state`).

- [ ] **Step 2: Kein brain-Daemon aktiv** — sonst leitet `brain-mcp` jeden Befehl über ihn und druckt ein anderes Format (Spec, Ausgangslage):
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" brain-mcp daemon status
```
  Erwartet: zwei Zeilen der Form ``no daemon on \\.\pipe\brain-<16 hex>-local (local); run `brain daemon start` `` und dasselbe mit `cloud` (`src/brain/client.py:196`). `--no-sync` gleicht die `.venv` des Quell-Worktrees nicht ab, `PYTHONDONTWRITEBYTECODE=1` schreibt kein `.pyc` hinein (wie Task 12 Step 6). Beginnt eine Zeile mit `daemon ` (`client.py:199`), hält der Controller an: den Daemon stoppt der Mensch (`brain-mcp daemon stop` schreibt in `%LOCALAPPDATA%\brain`, das dieser Task nie beschreibt).

- [ ] **Step 3: Benchmark für den Anteil vor und nach der Engine** — `internal/brain/search/bench_test.go`:

```go
package search_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// The three benchmarks below split what `loomux brain search` spends before
// and after the engine answers: the registry and every area's manifest, the
// identity registers of every visible area, and the whole of ExecuteSearch
// with the engine replaced. The spec asks for the register share on its own,
// because it grows with the areas a machine registers and not with the
// query. They read the real state directories and never write them; point
// LOOMUX_BENCH_REGISTRY at loomux's and LOOMUX_BENCH_LEGACY at ultra-brain's,
// and they skip themselves when nobody did.
func benchDirs(b *testing.B) (string, string) {
	registryDir := os.Getenv("LOOMUX_BENCH_REGISTRY")
	legacyDir := os.Getenv("LOOMUX_BENCH_LEGACY")
	if registryDir == "" || legacyDir == "" {
		b.Skip("set LOOMUX_BENCH_REGISTRY to loomux's state directory and LOOMUX_BENCH_LEGACY to ultra-brain's")
	}
	return registryDir, legacyDir
}

func BenchmarkVisibleAreasOfTheRealRegistry(b *testing.B) {
	registryDir, legacyDir := benchDirs(b)
	// A registry that fails would measure the error path, which returns
	// after the first broken area and says nothing about the others.
	if _, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		_, _ = privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	}
}

func BenchmarkRegistersOfTheRealRegistry(b *testing.B) {
	registryDir, legacyDir := benchDirs(b)
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	if err != nil {
		b.Fatal(err)
	}
	paths := make([]string, len(areas))
	for i, visible := range areas {
		paths[i] = filepath.Join(config.ManifestDir(visible.Area, legacyDir), "_identities.tsv")
	}
	for _, path := range paths {
		if _, err := identity.ReadIdentities(path); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(paths)), "registers")
	for b.Loop() {
		for _, path := range paths {
			_, _ = identity.ReadIdentities(path)
		}
	}
}

func BenchmarkExecuteSearchWithoutTheEngine(b *testing.B) {
	registryDir, legacyDir := benchDirs(b)
	now := time.Now()
	// One hit from an area the real registry holds, so the register lookup
	// runs once as it does on a real answer; the engine itself is scripted.
	hit := search.SearchHit{
		Collection: search.CollectionName("engineering/python"),
		Relative:   "index.md",
		Line:       1,
		Title:      "engineering/python",
		Score:      0.5,
	}
	answer := func() error {
		port := search.NewFakePort()
		port.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{hit}}}
		_, err := search.ExecuteSearch("latenz", "all", search.ProfileFast, 5,
			privacy.ChannelLocal, port, registryDir, legacyDir, now)
		return err
	}
	if err := answer(); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		_ = answer()
	}
}
```
  (Gegen Stubs mit den Vertragssignaturen kompiliert und `gofmt`-sauber.) Im Tor überspringen sich die drei, weil die Variablen fehlen:
```bash
go vet ./internal/brain/search/
```
  Erwartet: keine Ausgabe, Exit 0.
```bash
go test ./internal/brain/search/ -run '^$' -bench 'OfTheRealRegistry|WithoutTheEngine' -benchtime 1x
```
  Erwartet: `PASS` und `ok  	github.com/xidus90/loomux/internal/brain/search` ohne Messzeilen. Die Datei ist Test-Code; `covergate` misst sie nicht.
```bash
git add internal/brain/search/bench_test.go
```
  Erwartet: keine Ausgabe.

- [ ] **Step 4: Messfälle** — `testdata/bench/1b-1-brain.json`. Die Pfade nennen den Worktree aus Task 0; arbeitet dieser Task im Hauptcheckout (Task 0), folgt unter dem JSON ein `sed`, der sie umschreibt. Die Anfrage `latenz` ist die, die ultra-brains eigener Latenz-Bench in diesem Bestand treffen lässt (`src/brain/cli.py:523-526`, Vorgabe von `--latency-query`).

```json
[
  {
    "name": "loomux version (start floor)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/bin/loomux.exe",
          "version"
        ]
      }
    ]
  },
  {
    "name": "loomux brain search latenz --profile keyword (warm daemon)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/bin/loomux.exe",
          "brain",
          "search",
          "latenz",
          "--profile",
          "keyword"
        ]
      }
    ]
  },
  {
    "name": "loomux brain search latenz --profile fast (warm daemon)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/bin/loomux.exe",
          "brain",
          "search",
          "latenz",
          "--profile",
          "fast"
        ]
      }
    ]
  },
  {
    "name": "loomux brain search latenz --profile full (warm daemon)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/bin/loomux.exe",
          "brain",
          "search",
          "latenz",
          "--profile",
          "full"
        ]
      }
    ]
  },
  {
    "name": "loomux brain status",
    "dir": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/bin/loomux.exe",
          "brain",
          "status"
        ]
      }
    ]
  }
]
```
```bash
grep -c 'loomux-sdd-1b1' testdata/bench/1b-1-brain.json
```
  Erwartet: `10` (je Fall `dir` und das Programm in `argv`). Nur im Hauptcheckout folgt:
```bash
sed -i 's#loomux-sdd-1b1#loomux#g' testdata/bench/1b-1-brain.json
```
  Erwartet: keine Ausgabe; danach zeigt der `grep -c` oben `0`.
```bash
git add testdata/bench/1b-1-brain.json
```
  Erwartet: keine Ausgabe.

- [ ] **Step 5: Frisches Binary** — das Tor baut `bin/loomux.exe` aus dem gestagten Stand und prüft, dass `bench_test.go` nichts bricht:
```bash
sh .githooks/pre-commit
```
  Erwartet: Exit 0.
```bash
mkdir -p "$TEMP/loomux-bench-1b1"
```
  Erwartet: keine Ausgabe.

- [ ] **Step 6: Warm messen.** Zwei Arten „kalt" werden getrennt: die Spalte `cold (1st run)` von `bench-hooks` ist ein kalter **Prozess** gegen einen warmen Daemon; der kalte **Daemon** kommt in Step 7. Vorher den Daemon je Profil einmal wärmen (der erste Aufruf darf den Daemon starten; sein stderr trägt dann den Aufwärm-Hinweis):
```bash
bin/loomux.exe brain search latenz --profile keyword > /dev/null 2> "$TEMP/loomux-bench-1b1/warmup-keyword.err"; echo "exit $?"
```
```bash
bin/loomux.exe brain search latenz --profile fast > /dev/null 2> "$TEMP/loomux-bench-1b1/warmup-fast.err"; echo "exit $?"
```
```bash
bin/loomux.exe brain search latenz --profile full > /dev/null 2> "$TEMP/loomux-bench-1b1/warmup-full.err"; echo "exit $?"
```
  Erwartet je `exit 0`. Dann die Messung (im Hintergrund, `full` und `status` brauchen je 21 Läufe mehrerer Sekunden):
```bash
bin/loomux.exe dev bench-hooks testdata/bench/1b-1-brain.json -n 20 > "$TEMP/loomux-bench-1b1/warm.md" 2>&1; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
grep -c '^| loomux ' "$TEMP/loomux-bench-1b1/warm.md"
```
  Erwartet: `5` (je Fall eine Tabellenzeile, die mit `| ` und dem Fallnamen beginnt, `benchhooks.go:66`; jeder Fallname beginnt mit `loomux `). Jede der fünf Zeilen endet mit Exit-Codes `[0]`. Ein anderer Exit-Code in der Spalte ist ein Befund, der vor dem Eintrag geklärt wird; ein Bereich, dessen Manifest nach Step 1 nicht lesbar ist, zeigt sich hier als Exit `1` in jeder `brain`-Zeile.

- [ ] **Step 7: Daemon-kalt je Profil, dreimal.** `qmd mcp stop` beendet den Daemon, den `loomux brain search` mit `qmd mcp --http --daemon --port 8765` gestartet hat; qmd antwortet `Stopped QMD MCP server (PID <n>).` oder, ohne laufenden Daemon, `Not running (no PID file).`, beides mit Exit 0 (`@tobilu/qmd/dist/cli/qmd.js:4204-4230`). Kein anderer GPU-Prozess darf dabei laufen (Spec: ein CUDA-Daemon stürzte neben einem Vulkan-CLI-Prozess ab). `qmd mcp stop` beendet auch einen Daemon, den der Mensch gerade nutzt — vorher Bescheid geben. Zuerst prüfen, ob Git Bash `qmd` findet:
```bash
command -v qmd
```
  Erwartet: ein Pfad unter `%APPDATA%/npm`. Ohne Ausgabe (nur `qmd.CMD` vorhanden, das Bash nicht über PATHEXT auflöst) lautet jeder `qmd mcp stop` unten stattdessen `cmd //c "qmd mcp stop"`.

  Die neun Läufe, je ein Stopp vor jedem Lauf. Die Klammer schreibt die Ausgabe von `time` in eine eigene `.time`-Datei; `echo` zeigt den Exit-Code von `loomux` (in Git Bash gemessen: `{ time sh -c 'exit 3' > /dev/null ; } 2> t; echo "exit $?"` zeigt `exit 3`). Erwartet für jeden `qmd mcp stop`: `Stopped QMD MCP server (PID <n>).` oder `Not running (no PID file).`. Erwartet für jeden Lauf: `exit 0`.

  **Schwelle.** `real` in der `.time`-Datei liegt höchstens bei 0,5 s TCP-Probe plus den 5,7 s Modell-Laden, die `search.WarmingNotice` selbst nennt, plus dem Dreifachen des Spec-Werts für die erste Anfrage an einen frisch gestarteten Daemon: `keyword` ≤ 6,3 s (0,5 + 5,7 + 3 × 0,023), `fast` ≤ 22,4 s (0,5 + 5,7 + 3 × 5,4), `full` ≤ 41,9 s (0,5 + 5,7 + 3 × 11,9). Die Schwelle liegt bewusst unter der 60-s-Wartegrenze von `DefaultConnect`: Ein Lauf, der diese reißt, endet ohnehin mit Exit 1. Sie ist eine Planschranke aus benannten Konstanten, an keinem echten Kaltstart gemessen. Ein Wert darüber ist ein Befund an den Controller; ein Lauf mit Exit 0 wird trotzdem eingetragen.
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile keyword > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-keyword-1.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-keyword-1.time"; echo "exit $?"
```
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile keyword > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-keyword-2.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-keyword-2.time"; echo "exit $?"
```
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile keyword > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-keyword-3.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-keyword-3.time"; echo "exit $?"
```
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile fast > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-fast-1.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-fast-1.time"; echo "exit $?"
```
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile fast > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-fast-2.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-fast-2.time"; echo "exit $?"
```
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile fast > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-fast-3.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-fast-3.time"; echo "exit $?"
```
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile full > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-full-1.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-full-1.time"; echo "exit $?"
```
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile full > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-full-2.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-full-2.time"; echo "exit $?"
```
```bash
qmd mcp stop
```
```bash
{ time bin/loomux.exe brain search latenz --profile full > /dev/null 2> "$TEMP/loomux-bench-1b1/cold-full-3.err" ; } 2> "$TEMP/loomux-bench-1b1/cold-full-3.time"; echo "exit $?"
```
  Jeder Lauf muss den Daemon selbst gestartet haben; sonst hat der Stopp nicht gewirkt, und die Messung zählt nicht:
```bash
grep -c "^note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.$" "$TEMP/loomux-bench-1b1/"cold-*.err
```
  Erwartet: neun Zeilen, jede endet auf `.err:1`.
```bash
grep '^real' "$TEMP/loomux-bench-1b1/"cold-*.time
```
  Erwartet: neun Zeilen, die `grep -E 'cold-(keyword|fast|full)-[123]\.time:real[[:space:]]+[0-9]+m[0-9]+\.[0-9]+s$'` trifft; jede liegt unter der Schwelle ihres Profils.

- [ ] **Step 8: Den Anteil der Register ausweisen** — gegen die echten Verzeichnisse, nur lesend:
```bash
LOOMUX_BENCH_REGISTRY="$LOCALAPPDATA/loomux" LOOMUX_BENCH_LEGACY="$LOCALAPPDATA/brain" go test ./internal/brain/search/ -run '^$' -bench 'OfTheRealRegistry|WithoutTheEngine' -benchtime 50x -benchmem > "$TEMP/loomux-bench-1b1/registers.txt" 2>&1
```
  Erwartet: Exit 0.
```bash
grep -cE '^Benchmark(VisibleAreasOfTheRealRegistry|RegistersOfTheRealRegistry|ExecuteSearchWithoutTheEngine)-[0-9]+[[:space:]].* ns/op' "$TEMP/loomux-bench-1b1/registers.txt"
```
  Erwartet: `3` (die Endung `-<GOMAXPROCS>` hängt an der Maschine).
```bash
grep -E '^BenchmarkRegistersOfTheRealRegistry-[0-9]+[[:space:]]' "$TEMP/loomux-bench-1b1/registers.txt"
```
  Erwartet: eine Zeile mit `12.00 registers` bzw. `11.00 registers`, derselben Zahl wie `grep -c` in Step 1 (`b.ReportMetric` druckt Gleitkomma). Lesart: `ExecuteSearchWithoutTheEngine` ist alles, was `brain search` außer Prozessstart und qmd tut; `Registers` ist der Anteil, den die Spec eigens ausgewiesen haben will; `VisibleAreas` ist die Registry plus je Bereich ein Manifest.

- [ ] **Step 9: Startzeit vor und nach dem Einzug.** „Vor" ist der Commit vor Task 3 — der Elternteil des Commits, der `internal/brain/privacy/channel.go` anlegt. Das Archiv geht nach `$TEMP`, der Arbeitsbaum bleibt unberührt:
```bash
git log --diff-filter=A --format=%h -- internal/brain/privacy/channel.go
```
  Erwartet: genau ein Kurz-Hash (der Task-3-Commit).
```bash
mkdir -p "$TEMP/loomux-pre3"
```
  Erwartet: keine Ausgabe.
```bash
git archive "$(git log --diff-filter=A --format=%h -- internal/brain/privacy/channel.go)^" | tar -x -C "$TEMP/loomux-pre3"
```
  Erwartet: keine Ausgabe.
```bash
go -C "$TEMP/loomux-pre3" build -o "$TEMP/loomux-pre3/loomux.exe" ./cmd/loomux
```
  Erwartet: keine Ausgabe, Exit 0.

  Je dreimal (Werte als Spanne wie im 1a-Eintrag):
```bash
GODEBUG=inittrace=1 "$TEMP/loomux-pre3/loomux.exe" --version 2>> "$TEMP/loomux-bench-1b1/inittrace-pre3.txt"
```
```bash
GODEBUG=inittrace=1 bin/loomux.exe --version 2>> "$TEMP/loomux-bench-1b1/inittrace-post.txt"
```
  Erwartet je Aufruf: eine Zeile `loomux <version>` auf stdout (`internal/cli/cli.go`), Exit 0.

  Auswerten — jede Zeile mit mindestens 1 ms clock (Format `init <paket> @<t> ms, <clock> ms clock, <bytes> bytes, <allocs> allocs`, das fünfte Feld ist die clock), dann alle loomux-, x/text- und toml-Zeilen:
```bash
awk '$5+0 >= 1.0' "$TEMP/loomux-bench-1b1/inittrace-post.txt"
```
  Erwartet: keine Zeile, deren zweites Feld mit `github.com/xidus90/loomux/` beginnt. Zeilen der Laufzeit, der Standardbibliothek und von Fremdpaketen kommen mit Paket und clock in den Eintrag; bekannt ist `github.com/BurntSushi/toml/internal` mit 22 ms clock (Eintrag vom 2026-09-15 12:00).
```bash
grep -E 'xidus90/loomux|golang.org/x/text|BurntSushi' "$TEMP/loomux-bench-1b1/inittrace-post.txt"
```
  Erwartet: in jedem der drei Läufe dieselben Pakete, darunter `github.com/xidus90/loomux/internal/brain/search` und `github.com/xidus90/loomux/internal/brain/catalog` (umgezogene `regexp.MustCompile`-Paketvariablen) und die `golang.org/x/text`-Pakete, die Task 2 gemessen hat (`unicode/norm`, `internal/language`, `internal/language/compact`, `language`, `cases`), jede Zeile unter 1 ms clock.
```bash
grep -E 'xidus90/loomux|golang.org/x/text|BurntSushi' "$TEMP/loomux-bench-1b1/inittrace-pre3.txt"
```
  Erwartet: keine Zeile aus `internal/brain/search`, `internal/brain/catalog` oder `golang.org/x/text` — vor Task 3 linkt `cmd/loomux` keines der neuen `internal/brain/*`-Pakete.

  Regel: Keine `init`-Zeile aus `github.com/xidus90/loomux/...` erreicht 1 ms. `cmd/loomux` linkt `internal/brain/*` über die Befehlstabelle, also zahlt auch `--version` deren Paketvariablen; zu nennen sind die umgezogenen `regexp.MustCompile`-Variablen in `internal/brain/search` (`launcher.go`, `qmd.go`, `search.go`) und `internal/brain/catalog` (`area.go`) mit ihrer clock und jede `golang.org/x/text`-Zeile, die „vor" noch nicht erscheint. Eine loomux-Zeile ≥ 1 ms führt zu Step 10.

- [ ] **Step 10: Neuer Code mit Startkosten, test-first auf `sync.OnceValue`.** Die Global Constraints lassen Paketvariablen mit `regexp.MustCompile` nur im umgezogenen Code stehen; neuer Code kompiliert beim ersten Gebrauch. Nach den Entwürfen der Tasks 1 bis 14 hat neuer Code keine solche Variable: Task 3 kompiliert seine Globs je Aufruf, `stamp.go` (Task 8) und `internal/brain/status` (Task 9) bringen keinen Regex, und die einzige neue Regex, `ifLine` in `internal/dev/mutants` (Task 13), ist schon `sync.OnceValue`. Die Liste belegt das:
```bash
grep -rlE '^(var )?[[:space:]]*[A-Za-z0-9_]+[[:space:]]*= regexp\.MustCompile' --include='*.go' --exclude='*_test.go' internal | LC_ALL=C sort
```
  Erwartet genau diese sieben Dateien (die letzten drei stehen heute schon da, gemessen am 2026-09-15; die ersten vier ziehen mit Task 7 und Task 8 um; Testdateien linken nicht ins Binary und zählen nicht):
```
internal/brain/catalog/area.go
internal/brain/search/launcher.go
internal/brain/search/qmd.go
internal/brain/search/search.go
internal/brain/wiki/parse.go
internal/config/manifest.go
internal/verify/commit/language.go
```
  Zeigt Step 9 keine loomux-Zeile ≥ 1 ms, endet dieser Schritt hier. Liegt eine Zeile ≥ 1 ms in einem Paket dieser sieben Dateien, ist es umgezogener Code oder Code aus 1a: Er bleibt wortgleich, und die Zeile ist ein Befund an den Controller. Neuer Code ist nur eine achte Datei. Für ihre Paketvariable gilt das folgende Verfahren; `<name>` und `<muster>` stehen in der Zeile, die `grep -nE` mit demselben Ausdruck in dieser Datei zeigt, `<paket>` ist ihr Paketname und `<verzeichnis>` ihr Paketverzeichnis.

  (a) Test zuerst, neue Datei `<verzeichnis>/oncevalue_internal_test.go` im Paket der Quelle (nicht `_test`):
```go
package <paket>

import "testing"

func TestPatternIsCompiledOnFirstUse(t *testing.T) {
	if <name>() != <name>() {
		t.Fatal("<name> must hand back the one pattern it compiled")
	}
}
```
```bash
go test ./<verzeichnis>/ -run '^TestPatternIsCompiledOnFirstUse$' -count=1
```
  Erwartet: `FAIL` mit dem Build-Fehler `invalid operation: cannot call <name> (variable of type *regexp.Regexp): *regexp.Regexp is not a function` (mit Go 1.27 in einem Wegwerf-Modul gemessen).

  (b) Umstellen: `"sync"` in den Importblock, die Deklaration wird
```go
var <name> = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(<muster>)
})
```
  und jede Nutzung `<name>.` wird `<name>().`.
```bash
go test ./<verzeichnis>/ -count=1 -coverprofile="$TEMP/oncevalue.out"
```
  Erwartet: `ok` mit dem Paketpfad (im Wegwerf-Modul gemessen: `coverage: 100.0% of statements`).
```bash
go tool cover -func="$TEMP/oncevalue.out"
```
  Erwartet: jede Funktion des Pakets 100.0%. Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün); `//coverage:exempt` nur mit dem dort verlangten Grund.
```bash
gofmt -l <verzeichnis>
```
  Erwartet: keine Ausgabe.

  (c) Stagen, dann das Tor, dann die Messung wiederholen:
```bash
git add <verzeichnis>/oncevalue_internal_test.go <verzeichnis>/<quelldatei>
```
  Erwartet: keine Ausgabe.
```bash
sh .githooks/pre-commit
```
  Erwartet: Exit 0 (baut `bin/loomux.exe` neu).
```bash
GODEBUG=inittrace=1 bin/loomux.exe --version 2>> "$TEMP/loomux-bench-1b1/inittrace-post-oncevalue.txt"
```
  Dreimal. Erwartet je Aufruf: `loomux <version>`, Exit 0; danach zeigt `awk '$5+0 >= 1.0'` auf diese Datei keine Zeile dieses Pakets mehr. Der Eintrag in Step 13 nennt beide Messungen.

- [ ] **Step 11: Live-Vergleich `status` und `full`, Python zuerst.** Pythons CLI-Port setzt bei jedem `qmd`-Aufruf `QMD_LLAMA_GPU=vulkan`; darum läuft er ohne CUDA-Daemon, und loomux startet den Daemon erst danach. Jeder `uv run` hier trägt `--no-sync` und `PYTHONDONTWRITEBYTECODE=1` wie Step 2. Alle Befehle unmittelbar hintereinander, Uhrzeit festhalten:
```bash
mkdir -p "$TEMP/loomux-live-1b1"
```
  Erwartet: keine Ausgabe.

  Ein Vorlauf prüft, dass `brain-mcp` ohne Abgleich der `.venv` startet (gemessen am 2026-09-15: nichts auf stderr, Exit 0). Scheitert er, gleicht der Mensch die `.venv` ab; dieser Task tut es nie:
```bash
PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" brain-mcp --help > /dev/null; echo "exit $?"
```
  Erwartet: `exit 0` und nichts auf stderr.
```bash
qmd mcp stop
```
  Erwartet: `Stopped QMD MCP server (PID <n>).` oder `Not running (no PID file).`. Hat `command -v qmd` in Step 7 nichts gezeigt, lautet der Befehl `cmd //c "qmd mcp stop"`.
```bash
date -u +%Y-%m-%dT%H:%M:%SZ
```
  Erwartet: eine Zeit der Form `JJJJ-MM-TTThh:mm:ssZ`; das ist die Anfangszeit.
```bash
PYTHONUTF8=1 PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" brain-mcp status > "$TEMP/loomux-live-1b1/py-status.out" 2> "$TEMP/loomux-live-1b1/py-status.err"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
PYTHONUTF8=1 PYTHONDONTWRITEBYTECODE=1 uv run --no-sync --project "C:/Users/micro/Documents/#GIT/loomux-src/ub" brain-mcp search latenz --profile full > "$TEMP/loomux-live-1b1/py-search-full.out" 2> "$TEMP/loomux-live-1b1/py-search-full.err"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
bin/loomux.exe brain search latenz --profile full > "$TEMP/loomux-live-1b1/lx-search-full.out" 2> "$TEMP/loomux-live-1b1/lx-search-full.err"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
bin/loomux.exe brain status > "$TEMP/loomux-live-1b1/lx-status.out" 2> "$TEMP/loomux-live-1b1/lx-status.err"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
date -u +%Y-%m-%dT%H:%M:%SZ
```
  Erwartet: eine Zeit derselben Form; das ist die Endzeit.

  Vergleich (Python schreibt über die Pipe `\r\n`; Paritätszeile 4 der Spec):
```bash
diff <(tr -d '\r' < "$TEMP/loomux-live-1b1/py-status.out") "$TEMP/loomux-live-1b1/lx-status.out"
```
  Erwartet: genau die eingefügten Zeilen aus der Vorhersage unten.
```bash
diff <(tr -d '\r' < "$TEMP/loomux-live-1b1/py-search-full.out" | grep '^brain://') <(grep '^brain://' "$TEMP/loomux-live-1b1/lx-search-full.out")
```
  Erwartet: keine Ausgabe.
```bash
diff <(tr -d '\r' < "$TEMP/loomux-live-1b1/py-search-full.out") "$TEMP/loomux-live-1b1/lx-search-full.out"
```
  Erwartet: keine Ausgabe.
```bash
diff <(tr -d '\r' < "$TEMP/loomux-live-1b1/py-search-full.err") "$TEMP/loomux-live-1b1/lx-search-full.err"
```
  Erwartet: genau die eingefügte Zeile aus der Vorhersage unten.

  **Vorhergesagte Unterschiede — alles darüber hinaus ist ein Befund:**
  - `status`: loomux' Registry beginnt mit `project/loomux` und (nach Task 0) `project/loomux-sdd-1b1`; beide Bäume tragen kein `graph.json`, also steht direkt nach der ersten Zeile je ``<scope>: never indexed; run `brain reindex` `` — `diff` zeigt genau diese eine bzw. zwei eingefügten Zeilen. Die übrigen Zeilen (L1 aus dem Stempel `2026-09-13T19:51:12.255414+00:00`, L2, L5–L8 der zehn Bereiche) sind byte-gleich, weil die Blöcke in derselben Reihenfolge angehängt wurden.
  - `search --profile full` stdout: auf beiden Wegen gleich. qmd 2.8.3 stellt im MCP-`query` jeder Snippet-Zeile `N: ` voran (`dist/mcp/server.js:301`, `addLineNumbers(snippet, line)`), und Task 8 entfernt genau dieses Präfix in `translateReply`; das Ausschnittfenster ist auf beiden Wegen dasselbe (Spike: `full` auf beiden Wegen 4 von 4). Darum bleiben der `diff` der `brain://`-Zeilen (Reihenfolge, URI, Zeile, Score, Titel) und der volle `diff` leer. Jede Abweichung, in einer Trefferzeile wie in einer Snippet-Zeile, ist ein Befund.
  - `search` stderr: loomux hat den Daemon selbst gestartet und schreibt als erste Zeile den Aufwärm-Hinweis (Paritätszeile 5 der Spec); die `note:`-Befunde danach sind gleich.
  - Exit-Codes je `exit 0`.

- [ ] **Step 12: Abschnitt `## Live-Vergleich`** an `docs/.superpowers/parity/stufe-1b-1.md` anhängen, und in die Abweichungstabelle derselben Datei diese Zeile (Freigabe leer; der Code-Span mit innerem Backtick steht in doppelten Backticks):

```markdown
| Live-Vergleich `status` | `brain-mcp` liest `%LOCALAPPDATA%\brain\registry.toml` (10 Bereiche) | `loomux brain` liest `%LOCALAPPDATA%\loomux\registry.toml` (dieselben 10 plus `project/loomux`, `project/loomux-sdd-1b1`) und meldet für beide `` never indexed; run `brain reindex` `` | eine Registry für Schranke und Datenbefehle (Spec, Entscheidungen „Zustand") | |
```

  Die Zeilenzahlen für die Tabelle des Abschnitts:
```bash
wc -l "$TEMP/loomux-live-1b1/py-status.out" "$TEMP/loomux-live-1b1/lx-status.out" "$TEMP/loomux-live-1b1/py-search-full.out" "$TEMP/loomux-live-1b1/lx-search-full.out"
```
  Erwartet: vier Zeilen mit Zahl und Datei, dann eine `total`-Zeile; `lx-status.out` hat eine bzw. zwei Zeilen mehr als `py-status.out` (Vorhersage in Step 11), die beiden `search`-Dateien gleich viele (`wc -l` zählt `\n`, das `\r` der Python-Dateien ändert die Zahl nicht).

  Der Abschnitt; jede spitze Klammer nennt den Schritt, aus dem ihr Wert kommt, und jede `diff`-Ausgabe steht wörtlich darin (eine leere als `keine Ausgabe`):

````markdown
## Live-Vergleich

Python (`brain-mcp` im Tag-Worktree `loomux-1a-source`, ohne brain-Daemon) und loomux
(`bin/loomux.exe` aus dem Tor von `<Kurz-Hash aus git rev-parse --short HEAD>`) gegen den
echten Bestand, von <Anfangszeit aus Step 11> bis <Endzeit aus Step 11> (UTC), ohne
`LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR`, der qmd-Daemon vor dem ersten Befehl gestoppt.

| Befehl | Exit | Zeilen stdout |
|---|---:|---:|
| `brain-mcp status` | <Exit aus Step 11> | <Zahl für py-status.out aus diesem Step> |
| `brain-mcp search latenz --profile full` | <Exit aus Step 11> | <Zahl für py-search-full.out aus diesem Step> |
| `loomux brain search latenz --profile full` | <Exit aus Step 11> | <Zahl für lx-search-full.out aus diesem Step> |
| `loomux brain status` | <Exit aus Step 11> | <Zahl für lx-status.out aus diesem Step> |

### `status`

```text
<Ausgabe des ersten diff aus Step 11>
```

### `search --profile full`, Trefferzeilen

```text
<Ausgabe des zweiten diff aus Step 11>
```

### `search --profile full`, stdout

```text
<Ausgabe des dritten diff aus Step 11>
```

### `search --profile full`, stderr

```text
<Ausgabe des vierten diff aus Step 11>
```

Vorhergesagt waren genau die eingefügten `never indexed`-Zeilen in `status` und der
Aufwärm-Hinweis als erste stderr-Zeile von loomux. <Entweder „Nichts darüber hinaus." oder
jede weitere Abweichung mit dem Befund, den der Bericht in Step 15 nennt.>
````

- [ ] **Step 13: Eintrag in beiden Sprachen**, hinter dem letzten Eintrag der jeweiligen Datei (heute `## 2026-09-15 12:00 — The Exec Hook Against a Function Hook (Claude Mods)` in `docs/en/benchmarks.md` und `## 2026-09-15 12:00 — Der Exec-Hook gegen einen Function Hook (Claude Mods)` in `docs/de/benchmarks.md`), in der Form dieses letzten Eintrags: Überschrift mit Datum und Uhrzeit, Absatz zu Repo, Worktree, Zweig und Commit, **Goal/Ziel**, **Method/Methode**, Tabellen, `### Reading`/`### Lesart` mit nummerierten Punkten. Arbeitet der Task im Hauptcheckout (Task 0), lautet der Repo-Absatz „Repository `loomux`, main worktree, branch `master`" bzw. „Repo `loomux`, Haupt-Worktree, Zweig `master`". Jede spitze Klammer nennt die Quelle ihres Werts; eine Zahl ohne Datei unter `$TEMP/loomux-bench-1b1/` gehört nicht hinein.
```bash
date -r "$TEMP/loomux-bench-1b1/warm.md" +'%Y-%m-%d %H:%M'
```
  Erwartet: Datum und Uhrzeit, zu der Step 6 die Tabelle geschrieben hat; das ist die Überschrift.
```bash
git rev-parse --short HEAD
```
  Erwartet: der Kurz-Hash des Task-14-Commits.

  Englisch:

```markdown
## <date and time from date -r> — The Brain Data Commands on the Real Registry

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1`, branch
`sdd-1b-1`, commit `<short hash>` plus the uncommitted tree of Task 15 — this task adds
the measurement cases, three benchmarks and this entry; it does not change a command.

**Goal.** Show what `loomux brain search` costs warm and cold per profile against the
target of ≤ 150 ms for `--profile fast` warm end to end (spec: 63–83 ms qmd, 5–17 ms
probe and handshake, ~35 ms Go start floor), with the share for reading every identity
register named on its own; what `loomux brain status` costs; and whether moving the
brain packages in added start time.

**Method.** `loomux dev bench-hooks testdata/bench/1b-1-brain.json -n 20` against the
real state directories (`%LOCALAPPDATA%\loomux` with <areas from Step 1> areas, `%LOCALAPPDATA%\brain`
for read-only artefacts and the reconcile stamp; neither `LOOMUX_STATE_DIR` nor
`LOOMUX_LEGACY_BRAIN_DIR` set), query `latenz`, after one warming call per profile:
the cold column is a cold process against a warm qmd daemon. The cold daemon is
measured separately: `qmd mcp stop`, then one timed `brain search` per profile, three
times. The register share: `go test ./internal/brain/search/ -run '^$' -bench 'OfTheRealRegistry|WithoutTheEngine' -benchtime 50x -benchmem`
with `LOOMUX_BENCH_REGISTRY`/`LOOMUX_BENCH_LEGACY` on the same directories. Start
time: `GODEBUG=inittrace=1 loomux --version`, three runs each, on the binary of the
commit before Task 3 and on `bin/loomux.exe`.

<the table from warm.md, unchanged>

| cold daemon (`qmd mcp stop` before each run) | run 1 | run 2 | run 3 |
|---|---:|---:|---:|
| brain search latenz --profile keyword | <real from cold-keyword-1.time> | <real from cold-keyword-2.time> | <real from cold-keyword-3.time> |
| brain search latenz --profile fast | <real from cold-fast-1.time> | <real from cold-fast-2.time> | <real from cold-fast-3.time> |
| brain search latenz --profile full | <real from cold-full-1.time> | <real from cold-full-2.time> | <real from cold-full-3.time> |

| benchmark (50 runs) | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| VisibleAreasOfTheRealRegistry | <ns/op from registers.txt> | <B/op from registers.txt> | <allocs/op from registers.txt> |
| RegistersOfTheRealRegistry (<registers from registers.txt> registers) | <ns/op from registers.txt> | <B/op from registers.txt> | <allocs/op from registers.txt> |
| ExecuteSearchWithoutTheEngine | <ns/op from registers.txt> | <B/op from registers.txt> | <allocs/op from registers.txt> |

### Reading

1. **The target value.** `brain search --profile fast` warm median against 150 ms, and
   against the start floor (`loomux version`) of the same table.
2. **The registers.** `RegistersOfTheRealRegistry` in ms and as a share of the fast warm
   median; `ExecuteSearchWithoutTheEngine` as everything loomux does besides process
   start and qmd.
3. **Cold daemon.** The three runs per profile against the spike's 23 ms / 5.4 s / 11.9 s
   for a freshly started daemon. The warm `full` median above is not a new query: the
   bench repeats `latenz` 21 times, which the spike measured at ~250 ms, where a new
   query on a warm daemon took 4.2–7.9 s.
4. **`brain status`.** Warm median; it asks the qmd CLI once per indexed area and once
   for the backlog, so it is bound by qmd process starts, not by Go.
5. **Start time.** Every `init` line of 1 ms or more before and after, the loomux lines
   of the moved `regexp` package variables with their clock, the `golang.org/x/text`
   lines that appear only after the move, and, where a line of new code reached 1 ms,
   its move to `sync.OnceValue` with the line measured again.
```

  Deutsch, in `docs/de/benchmarks.md`, mit demselben Inhalt. Wie im Eintrag vom 2026-09-15 12:00 bleibt der Befehl in der Falltabelle englisch, der Klammerzusatz wird deutsch (`(warmer Daemon)`, `(Startboden)`); Zahlen mit Dezimalkomma (`24,5 ms`) und Tausenderpunkt (`916.728 ns/op`):

```markdown
## <Datum und Uhrzeit aus date -r> — Die Brain-Datenbefehle auf der echten Registry

Repo `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1`, Zweig
`sdd-1b-1`, Commit `<Kurz-Hash>` plus der uncommittete Baum von Task 15 — dieser Task
fügt die Messfälle, drei Benchmarks und diesen Eintrag hinzu; er ändert keinen Befehl.

**Ziel.** Zeigen, was `loomux brain search` warm und kalt je Profil kostet, gemessen am
Zielwert ≤ 150 ms für `--profile fast` warm Ende zu Ende (Spec: 63–83 ms qmd, 5–17 ms
Probe und Handshake, rund 35 ms Go-Startboden), mit dem Anteil fürs Lesen aller
Identitätsregister eigens ausgewiesen; was `loomux brain status` kostet; und ob der
Einzug der Brain-Pakete Startzeit hinzugefügt hat.

**Methode.** `loomux dev bench-hooks testdata/bench/1b-1-brain.json -n 20` gegen die
echten Zustandsverzeichnisse (`%LOCALAPPDATA%\loomux` mit <Bereiche aus Step 1> Bereichen,
`%LOCALAPPDATA%\brain` für die Artefakte schreibgeschützter Bereiche und den
Reconcile-Stempel; weder `LOOMUX_STATE_DIR` noch `LOOMUX_LEGACY_BRAIN_DIR` gesetzt),
Anfrage `latenz`, nach einem Aufwärmaufruf je Profil: Die Kalt-Spalte ist ein kalter
Prozess gegen einen warmen qmd-Daemon. Der kalte Daemon ist getrennt gemessen:
`qmd mcp stop`, dann ein zeitgemessenes `brain search` je Profil, dreimal. Der
Registeranteil: `go test ./internal/brain/search/ -run '^$' -bench 'OfTheRealRegistry|WithoutTheEngine' -benchtime 50x -benchmem`
mit `LOOMUX_BENCH_REGISTRY`/`LOOMUX_BENCH_LEGACY` auf denselben Verzeichnissen.
Startzeit: `GODEBUG=inittrace=1 loomux --version`, je drei Läufe, am Binary des Commits
vor Task 3 und an `bin/loomux.exe`.

<die Tabelle aus warm.md mit dem Kopf | Fall | kalt (1. Lauf) | warmer Median | warmes Min | warmes Max | Exit-Codes |, Klammerzusätzen auf Deutsch und Dezimalkomma>

| kalter Daemon (`qmd mcp stop` vor jedem Lauf) | Lauf 1 | Lauf 2 | Lauf 3 |
|---|---:|---:|---:|
| brain search latenz --profile keyword | <real aus cold-keyword-1.time> | <real aus cold-keyword-2.time> | <real aus cold-keyword-3.time> |
| brain search latenz --profile fast | <real aus cold-fast-1.time> | <real aus cold-fast-2.time> | <real aus cold-fast-3.time> |
| brain search latenz --profile full | <real aus cold-full-1.time> | <real aus cold-full-2.time> | <real aus cold-full-3.time> |

| Benchmark (50 Läufe) | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| VisibleAreasOfTheRealRegistry | <ns/op aus registers.txt> | <B/op aus registers.txt> | <allocs/op aus registers.txt> |
| RegistersOfTheRealRegistry (<registers aus registers.txt> Register) | <ns/op aus registers.txt> | <B/op aus registers.txt> | <allocs/op aus registers.txt> |
| ExecuteSearchWithoutTheEngine | <ns/op aus registers.txt> | <B/op aus registers.txt> | <allocs/op aus registers.txt> |

### Lesart

1. **Der Zielwert.** Der warme Median von `brain search --profile fast` gegen 150 ms und
   gegen den Startboden (`loomux version`) derselben Tabelle.
2. **Die Register.** `RegistersOfTheRealRegistry` in ms und als Anteil am warmen
   `fast`-Median; `ExecuteSearchWithoutTheEngine` als alles, was loomux außer Prozessstart
   und qmd tut.
3. **Kalter Daemon.** Die drei Läufe je Profil gegen die 23 ms / 5,4 s / 11,9 s des Spikes
   für einen frisch gestarteten Daemon. Der warme `full`-Median oben ist keine neue
   Anfrage: Die Messung wiederholt `latenz` 21-mal, was der Spike mit ~250 ms gemessen hat,
   während eine neue Anfrage an einen warmen Daemon 4,2–7,9 s brauchte.
4. **`brain status`.** Der warme Median; der Befehl fragt die qmd-CLI einmal je indiziertem
   Bereich und einmal nach dem Rückstand, ist also an qmd-Prozessstarts gebunden, nicht an Go.
5. **Startzeit.** Jede `init`-Zeile mit 1 ms oder mehr vor und nach dem Einzug, die
   loomux-Zeilen der umgezogenen `regexp`-Paketvariablen mit ihrer clock, die
   `golang.org/x/text`-Zeilen, die erst nach dem Einzug erscheinen, und, wo eine Zeile
   neuen Codes 1 ms erreichte, ihre Umstellung auf `sync.OnceValue` samt neu gemessener Zeile.
```

  Hält der Zielwert nicht, sagt Punkt 1 in beiden Sprachen das mit dem Posten, der ihn reißt — ein Umbau folgt nicht in diesem Task.
```bash
git diff --stat docs/en/benchmarks.md docs/de/benchmarks.md
```
  Erwartet: beide Dateien nur mit Einfügungen (`+`), keine Löschung — der neue Eintrag steht hinter dem letzten, nichts davor ändert sich.

- [ ] **Step 14: Stagen, Tor, Commit**

```bash
git add testdata/bench/1b-1-brain.json internal/brain/search/bench_test.go docs/en/benchmarks.md docs/de/benchmarks.md docs/.superpowers/parity/stufe-1b-1.md
```
Erwartet: keine Ausgabe. Hat Step 10 gegriffen, sind dessen zwei Dateien schon gestagt.
```bash
sh .githooks/pre-commit
```
Erwartet: Exit 0.
```bash
git branch --show-current
```
Erwartet: `sdd-1b-1` (im Hauptcheckout nach Task 0: `master`).
```bash
git rev-parse --short HEAD
```
Erwartet: der Kurz-Hash des Task-14-Commits (derselbe wie in Step 13).
```bash
git diff --cached --stat
```
Erwartet: genau die fünf Dateien, dazu die zwei aus Step 10, falls er gegriffen hat.
```bash
printf '%s\n' 'Measure the brain data commands on the real registry' '' 'Warm and cold search per profile, status, the share of the identity' 'registers, start time before and after the move, and a live comparison' 'with the Python reference.' > "$TEMP/commit-task15.txt"
```
Erwartet: keine Ausgabe.
```bash
git commit -F "$TEMP/commit-task15.txt"
```
Erwartet: Exit 0; der Commit löst das Tor noch einmal aus, das ist gewollt.
```bash
git log -1 --format='%an <%ae>'
```
Erwartet: die Nutzeridentität, kein Modell und kein Agent.
```bash
git log -1 --format=%B
```
Erwartet: genau die Nachricht aus `$TEMP/commit-task15.txt`, keine `Co-Authored-By`-Zeile.

- [ ] **Step 15: Bericht** — nennt die Tabellen, das Urteil zum Zielwert, die neun kalten Läufe gegen ihre Schwellen aus Step 7, jede `init`-Zeile ≥ 1 ms mit Paket, ob Step 10 gegriffen hat, die vier `diff`-Ausgaben und jeden Befund. **Paritätszeilen, die dieser Task schafft:** die Zeile „Live-Vergleich `status`" aus Step 12. Tritt im Live-Vergleich ein Unterschied außerhalb der Vorhersage auf, auch in einer Snippet-Zeile von `search --profile full`, ist er ein Befund, keine stille Zeile.

---

### Task 16: Rauchtest und Abschluss

Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis 15 sind committet.

**Files:**
- Modify: `docs/.superpowers/parity/stufe-1b-1.md` (Abschnitt `## Pilot`)
- Modify: `README.md`, `README.de.md` (Ergänzung zum Vertrag: `AGENTS.md` verlangt, die READMEs mit jeder Implementierungsänderung auf den tatsächlichen Stand zu bringen; die README-Pflege der Stufe liegt ganz in diesem Task)
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (Abschnitt 7 `loomux brain`; Ergänzung zum Vertrag nach `AGENTS.md`, im Stil des Abschnitts `loomux dev mutants`, den Task 13 in Abschnitt 9 schreibt)

**Interfaces:**
- Consumes (Task 10): `loomux brain search|catalog|read|neighbors|status` mit Exit 0/1/2 und den Usage-Zeilen aus `TestBrainUsageLines`. Zwei Formen des Usage-Fehlers (`TestBrainUsageErrorsNameTheParserThatRefused`): Verweigert der Unterbefehl (fehlendes Argument, ungültige Wahl, `-n` kleiner 1), folgen seine Usage-Zeile und `loomux brain <sub>: error: {argparse-Wortlaut}`; verweigert der oberste Parser (kein oder unbekannter Unterbefehl, überzählige Argumente), folgen `usage: loomux brain {search,catalog,read,neighbors,status} ...` und `loomux brain: error: {argparse-Wortlaut}`. Laufzeitfehler `error: {grund}` auf stderr ohne stdout, Exit 1.
- Consumes (Task 7): `catalog.RenderRootCatalog(areas []config.Area) string` — `"# brain\n\n"` und je Bereich nach Scope sortiert `* [s](brain://s/)\n`.
- Consumes (Task 5): `graph.RenderNeighbors(incoming, outgoing []string) string` — `incoming: <a>, <b>\noutgoing: <c>\n`, `-` für eine leere Richtung, beide Listen sortiert.
- Consumes (Task 3): der Verweigerungswortlaut `{scope}/{relative} leaves the area` aus `privacy.Contained`.
- Consumes (Task 8): der Stempelbefund von `search` auf stderr, ``note: the last full reconciliation was {iso}, more than 24 hours ago: a source may have changed without this answer knowing (run `brain reconcile`)``, und der Wortlaut von `search.WarmingNotice` (für die cli-reference).
- Consumes (Task 9): `status.Lines(ch privacy.Channel, port search.SearchPort, registryDir, legacyDir string, now time.Time) ([]string, error)` mit der Zeile L1 ``last reconcile: {iso}; older than 24 h, run `brain reconcile` `` und der Zeile ``{scope}: never indexed; run `brain reindex` ``.
- Consumes (Task 11): `internal/dev/fakeqmd/qmd/main.go`, die einzige neue Stelle mit `//coverage:exempt` in der Stufe.
- Consumes (Task 12): `TestRecordedCasesOfStage1b1` in `internal/cli`; (Task 13) den Abschnitt `loomux dev mutants` in beiden `cli-reference.md` als Stilvorbild; (Task 14) Abschnitt `## Mutationsrunde` mit dreizehn Paketzeilen; (Task 15) Einträge in beiden `benchmarks.md`, Abschnitt `## Live-Vergleich`, die angehängte Registry.
- Produces: nichts im Code.

**Gewählter Bereich für `read`, `neighbors` und den Einzelkatalog: `engineering/python`** (schreibbar, Artefakte im Baum `C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/python`, gelesen 2026-09-15): `.brain.toml` mit `scope = "engineering/python"`, `index.md` ohne `\r` (`grep -c $'\r'` → `0`), `graph.json` mit genau drei Kanten, alle aus `index.md` (nach `_schema.md`, `audit.md`, `log.md`), `_identities.tsv` mit vier Zeilen. `project/loomux` eignet sich nicht: dort liegt kein `index.md`.

- [ ] **Step 1: Voraussetzungen**
```bash
grep -c '^\[\[area\]\]' "$LOCALAPPDATA/loomux/registry.toml"
```
  Erwartet: `12` (mit dem Worktree-Block aus Task 0) bzw. `11` — sonst fehlt Task 15 Step 1, und der Rauchtest wartet auf den Menschen. Gegen genau diese Zahl zählt Step 3 die Bereiche.
```bash
env | grep '^LOOMUX_'
```
  Erwartet: keine Ausgabe.
```bash
git status --porcelain
```
  Erwartet: keine Ausgabe (der Stand nach Task 15 ist committet).
```bash
sh .githooks/pre-commit
```
  Erwartet: Exit 0 (baut `bin/loomux.exe` aus dem Stand nach Task 15).
```bash
git rev-parse --short HEAD
```
  Erwartet: der Kurz-Hash des Task-15-Commits; er steht in Step 10 im Abschnitt `## Pilot`.
```bash
date +%Y-%m-%d
```
  Erwartet: das heutige Datum; es steht in Step 10 im Abschnitt `## Pilot`.
```bash
mkdir -p "$TEMP/loomux-pilot-1b1"
```
  Erwartet: keine Ausgabe.

- [ ] **Step 2 (Controller fragt den Nutzer): Weisung für den Rauchtest.** Die Spec zählt den Rauchtest zu den Mensch-Schritten; in Stufe 1a hat ihn der Controller nur auf Weisung des Nutzers gefahren (`stufe-1a.md`, Abschnitt `## Pilot`). Der Controller fragt darum den Nutzer, ob er die Steps 3 bis 9 auf dessen Weisung fährt oder ob der Mensch sie selbst fährt. Ohne ausdrückliches Ja fährt kein Agent die Steps 3 bis 9; der Task wartet dann, bis der Mensch sie mit denselben Befehlen gefahren hat und die Dateien unter `$TEMP/loomux-pilot-1b1/` vorliegen. Ein Subagent, der diesen Task fährt, hält hier an und meldet sich beim Controller.

- [ ] **Step 3: `catalog` über alle Bereiche**
```bash
bin/loomux.exe brain catalog > "$TEMP/loomux-pilot-1b1/catalog.out" 2> "$TEMP/loomux-pilot-1b1/catalog.err"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
cat "$TEMP/loomux-pilot-1b1/catalog.err"
```
  Erwartet: keine Ausgabe.
```bash
grep -c '^\* \[' "$TEMP/loomux-pilot-1b1/catalog.out"
```
  Erwartet: dieselbe Zahl wie `grep -c` in Step 1.
```bash
cat "$TEMP/loomux-pilot-1b1/catalog.out"
```
  Erwartet: stdout genau (Byte-Sortierung; ohne Worktree-Block entfällt die Zeile `project/loomux-sdd-1b1`):
```
# brain

* [engineering/craft](brain://engineering/craft/)
* [engineering/python](brain://engineering/python/)
* [hub](brain://hub/)
* [knowledge](brain://knowledge/)
* [project/ecoflow](brain://project/ecoflow/)
* [project/iam-wiki](brain://project/iam-wiki/)
* [project/loomux](brain://project/loomux/)
* [project/loomux-sdd-1b1](brain://project/loomux-sdd-1b1/)
* [project/obsidian-ai](brain://project/obsidian-ai/)
* [project/space](brain://project/space/)
* [project/ultra-brain](brain://project/ultra-brain/)
* [project/ultraloom](brain://project/ultraloom/)
```

- [ ] **Step 4: `catalog` eines Bereichs** — byte-gleich zu dessen `index.md`:
```bash
bin/loomux.exe brain catalog --scope engineering/python > "$TEMP/loomux-pilot-1b1/catalog-python.out"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
cmp "$TEMP/loomux-pilot-1b1/catalog-python.out" "/c/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/python/index.md"; echo "cmp $?"
```
  Erwartet: `cmp 0`.

- [ ] **Step 5: `read` mit Abschnitt**
```bash
bin/loomux.exe brain read index.md --scope engineering/python --section Dateien > "$TEMP/loomux-pilot-1b1/read.out"; echo "exit $?"
```
```bash
printf '## Dateien\n\n* [_schema](_schema.md)\n* [audit](audit.md)\n* [log](log.md)\n\n> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog.\n' | cmp - "$TEMP/loomux-pilot-1b1/read.out"; echo "cmp $?"
```
  Erwartet: `exit 0`, `cmp 0`. Der erwartete Text ist an der Referenz gemessen: `_section(Path('C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/python/index.md').read_text(encoding='utf-8'), 'Dateien')` ergab `'## Dateien\n\n* [_schema](_schema.md)\n* [audit](audit.md)\n* [log](log.md)\n\n> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog.\n'`.

- [ ] **Step 6: `neighbors`**
```bash
bin/loomux.exe brain neighbors index.md --scope engineering/python > "$TEMP/loomux-pilot-1b1/neighbors.out"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
printf 'incoming: -\noutgoing: _schema.md, audit.md, log.md\n' | cmp - "$TEMP/loomux-pilot-1b1/neighbors.out"; echo "cmp $?"
```
  Erwartet: `cmp 0` (keine Kante führt nach `index.md`; die drei Ziele byte-sortiert, `_` vor `a`).

- [ ] **Step 7: Eine Verweigerung und beide Formen des Usage-Fehlers** — die Fehlerwege laufen auf der echten Registry wie im Korpus:
```bash
bin/loomux.exe brain read ../x.md --scope engineering/python > "$TEMP/loomux-pilot-1b1/leave.out" 2> "$TEMP/loomux-pilot-1b1/leave.err"; echo "exit $?"
```
  Erwartet: `exit 1`.
```bash
cat "$TEMP/loomux-pilot-1b1/leave.out" "$TEMP/loomux-pilot-1b1/leave.err"
```
  Erwartet: genau `error: engineering/python/../x.md leaves the area` (stdout leer; an der Referenz gemessen: `_contained('engineering/python', '../x.md')` wirft `AccessDenied 'engineering/python/../x.md leaves the area'`).

  Der Unterbefehl verweigert:
```bash
bin/loomux.exe brain read index.md > "$TEMP/loomux-pilot-1b1/usage.out" 2> "$TEMP/loomux-pilot-1b1/usage.err"; echo "exit $?"
```
  Erwartet: `exit 2`.
```bash
printf 'usage: loomux brain read --scope SCOPE [--section SECTION] [--channel {local,cloud}] relative\nloomux brain read: error: the following arguments are required: --scope\n' | cmp - "$TEMP/loomux-pilot-1b1/usage.err"; echo "cmp $?"
```
  Erwartet: `cmp 0`. Referenz, gemessen am 2026-09-15: `brain-mcp read: error: the following arguments are required: --scope`, Exit 2; ihre Usage bricht nach Terminalbreite um und nennt `[-h]` und `[--state-dir STATE_DIR]` (Task 10, Bericht Punkt 6).

  Der oberste Parser verweigert:
```bash
bin/loomux.exe brain catalog extra > "$TEMP/loomux-pilot-1b1/top-usage.out" 2> "$TEMP/loomux-pilot-1b1/top-usage.err"; echo "exit $?"
```
  Erwartet: `exit 2`.
```bash
printf 'usage: loomux brain {search,catalog,read,neighbors,status} ...\nloomux brain: error: unrecognized arguments: extra\n' | cmp - "$TEMP/loomux-pilot-1b1/top-usage.err"; echo "cmp $?"
```
  Erwartet: `cmp 0`. Referenz, gemessen am 2026-09-15: `brain-mcp: error: unrecognized arguments: extra`, Exit 2 — der oberste Parser meldet sich, nicht `brain-mcp catalog`.
```bash
cat "$TEMP/loomux-pilot-1b1/usage.out" "$TEMP/loomux-pilot-1b1/top-usage.out"
```
  Erwartet: keine Ausgabe.

- [ ] **Step 8: `search`**
```bash
bin/loomux.exe brain search latenz --profile fast > "$TEMP/loomux-pilot-1b1/search.out" 2> "$TEMP/loomux-pilot-1b1/search.err"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
grep -vcE '^(brain://.+:[0-9]+  [0-9]+%  .*|    .*|)$' "$TEMP/loomux-pilot-1b1/search.out"
```
  Erwartet: `0` — jede Zeile ist eine Trefferzeile `brain://<scope>/<pfad>:<zeile>  <n>%  <titel>`, eine um vier Leerzeichen eingerückte Snippet-Zeile oder die Leerzeile nach einem Treffer (`.+` statt `[^ ]+`, weil Pfade wie `92 Engineering/python/index.md` Leerzeichen tragen; an einer Beispielausgabe gemessen). Zeigt der Befehl `1`, gibt es keinen Treffer, und `cat "$TEMP/loomux-pilot-1b1/search.out"` zeigt genau `no matches`.
```bash
grep -vc '^note: ' "$TEMP/loomux-pilot-1b1/search.err"
```
  Erwartet: `0` (stderr trägt nur `note:`-Zeilen).
```bash
tail -n 1 "$TEMP/loomux-pilot-1b1/search.err"
```
  Erwartet, solange der Stempel `2026-09-13T19:51:12.255414+00:00` steht, genau (der Stempelbefund kommt nach den übrigen Befunden, Task 8):
  ``note: the last full reconciliation was 2026-09-13T19:51:12.255414+00:00, more than 24 hours ago: a source may have changed without this answer knowing (run `brain reconcile`)``

- [ ] **Step 9: `status`**
```bash
bin/loomux.exe brain status > "$TEMP/loomux-pilot-1b1/status.out" 2> "$TEMP/loomux-pilot-1b1/status.err"; echo "exit $?"
```
  Erwartet: `exit 0`.
```bash
cat "$TEMP/loomux-pilot-1b1/status.err"
```
  Erwartet: keine Ausgabe.
```bash
head -n 2 "$TEMP/loomux-pilot-1b1/status.out"
```
  Erwartet, solange der Stempel steht, genau diese zwei Zeilen (L1 aus Task 9, dann der erste Bereich der Registry ohne `graph.json`; dieselbe Form wie der Live-Vergleich aus Task 15):
```
last reconcile: 2026-09-13T19:51:12.255414+00:00; older than 24 h, run `brain reconcile`
project/loomux: never indexed; run `brain reindex`
```

- [ ] **Step 10: Abschnitt `## Pilot`** in `docs/.superpowers/parity/stufe-1b-1.md`, Form des Pilot-Abschnitts von `stufe-1a.md` (Absatz zur Durchführung, abgehakte Liste, gemessener Absatz). Zuerst die drei Zahlen für den letzten Absatz:
```bash
grep -c '^brain://' "$TEMP/loomux-pilot-1b1/search.out"
```
  Erwartet: eine Zahl von `0` bis `5` (Vorgabe `-n 5`).
```bash
grep -c '^note: ' "$TEMP/loomux-pilot-1b1/search.err"
```
  Erwartet: eine Zahl ab `1` (mindestens der Stempelbefund aus Step 8).
```bash
wc -l < "$TEMP/loomux-pilot-1b1/status.out"
```
  Erwartet: eine Zahl ab `2` (Step 9).

  Der Abschnitt; jede spitze Klammer nennt den Schritt, aus dem ihr Wert kommt:
```markdown
## Pilot

Der Rauchtest (Task 16) ist am <Datum aus Step 1> auf Weisung des Nutzers vom Controller
gefahren und hier abgehakt worden. Gemessen wurde gegen die echte Registry
(`%LOCALAPPDATA%\loomux\registry.toml`, nach Task 15 Schritt 1 mit den zehn Bereichen
von ultra-brain) und ultra-brains Zustandsverzeichnis (`%LOCALAPPDATA%\brain`), mit
`bin/loomux.exe` aus dem Tor von `<Kurz-Hash aus Step 1>`, ohne `LOOMUX_STATE_DIR` und
`LOOMUX_LEGACY_BRAIN_DIR`. Was damit **nicht** belegt ist: der Kanal `cloud` auf echten
Daten und ein Bereich mit `local_only` — beides deckt der Korpus.

- [x] `brain catalog` → Exit 0, so viele Bereiche wie `[[area]]`-Blöcke in der Registry, byte-sortiert.
- [x] `brain catalog --scope engineering/python` → Exit 0, byte-gleich zu `index.md`.
- [x] `brain read index.md --scope engineering/python --section Dateien` → Exit 0, der an der Referenz gemessene Abschnitt.
- [x] `brain neighbors index.md --scope engineering/python` → Exit 0, `incoming: -`, `outgoing: _schema.md, audit.md, log.md`.
- [x] `brain read ../x.md --scope engineering/python` → Exit 1, `error: engineering/python/../x.md leaves the area`.
- [x] `brain read index.md` → Exit 2, `loomux brain read: error: the following arguments are required: --scope` (Unterbefehl).
- [x] `brain catalog extra` → Exit 2, `loomux brain: error: unrecognized arguments: extra` (oberster Parser).
- [x] `brain search latenz --profile fast` → Exit 0, Stempelbefund als letzte `note:`-Zeile.
- [x] `brain status` → Exit 0, erste Zeile der Stempel, zweite `project/loomux: never indexed`.

Gemessen am <Datum aus Step 1>, Binary `bin/loomux.exe` aus dem Tor von `<Kurz-Hash aus Step 1>`:
`catalog` <Exit aus Step 3> mit <Zahl aus `grep -c '^\* \['` in Step 3> Bereichen;
`catalog --scope engineering/python` <Exit aus Step 4>, `cmp` <Ergebnis aus Step 4>;
`read index.md --section Dateien` <Exit aus Step 5>, `cmp` <Ergebnis aus Step 5>;
`neighbors index.md` <Exit aus Step 6>, `cmp` <Ergebnis aus Step 6>. Verweigert mit
<Exit aus Step 7>: `read ../x.md` („leaves the area"). Usage-Fehler mit <Exit aus Step 7>:
`read index.md` beim Unterbefehl, `catalog extra` beim obersten Parser, `cmp` <beide
Ergebnisse aus Step 7>. `search latenz --profile fast` <Exit aus Step 8> mit <Zahl der
`brain://`-Zeilen aus diesem Step> Treffern und <Zahl der `note:`-Zeilen aus diesem Step>
`note:`-Zeilen; `status` <Exit aus Step 9> mit <Zeilenzahl aus diesem Step> Zeilen.
```
  Hat der Mensch die Steps 3 bis 9 selbst gefahren (Step 2), lautet der erste Satz: „Der Rauchtest (Task 16) ist am <Datum aus Step 1> vom Nutzer gefahren und vom Controller hier abgehakt worden." Ein Punkt, der nicht wie erwartet ausfällt, wird nicht abgehakt; er steht mit dem tatsächlichen Ergebnis darunter und ist ein Befund.

- [ ] **Step 11: READMEs auf den Stand bringen** — genau diese Änderungen, der Rest bleibt. Jede Stelle ist am Text verankert, nicht an Zeilennummern. Vorher belegen, dass der heutige Text dasteht (Stand `e4bf7b5`, gemessen am 2026-09-15; die Nutzeränderungen an beiden READMEs sind mit diesem Commit eingecheckt):
```bash
grep -cxF -e 'Loomux is currently executing its staged fusion plan (Stage 1a pilot complete; subsequent stages in active development):' -e '| Semantic QMD Index | Embedding and neural search integration with local caching in `~/.cache/qmd`. | 🚧 **In Migration** (Stage 3) |' -e 'Commands currently active in the Stage 1a pilot vs. specified for subsequent fusion and graph stages:' -e '### Active Commands (Stage 1a Pilot)' -e 'loomux wiki gate                    # gate wiki freshness and structural constraints' -e 'loomux brain search "<query>"       # hybrid semantic & keyword search across wiki and ADRs' -e 'loomux brain catalog               # list all tracked areas, topics, and identity documents' -e 'loomux brain read <path>           # read a wiki page, concept, or identity definition' README.md
```
  Erwartet: `8` (jede der acht Ankerzeilen genau einmal).
```bash
grep -cxF -e 'Loomux setzt derzeit seinen mehrstufigen Fusionsplan um (Stufe 1a Pilot abgeschlossen; Folgestufen in aktiver Entwicklung):' -e '| Semantischer QMD-Index | Einbettung lokaler Vektoren und neuronaler Suche mit Caching in `~/.cache/qmd`. | 🚧 **In Migration** (Stufe 3) |' -e 'Aktive Befehle des Stufe-1a-Piloten im Vergleich zu spezifizierten Befehlen der Folge- und Graph-Stufen:' -e '### Aktive Befehle (Stufe 1a Pilot)' -e 'loomux wiki gate                    # Erzwingt Frische und strukturelle Schranken des Wikis' -e 'loomux brain search "<anfrage>"     # Hybride semantische & Keyword-Suche über Wiki und ADRs' -e 'loomux brain catalog               # Listet alle verwalteten Bereiche, Themen und Identitäten' -e 'loomux brain read <pfad>           # Liest eine Wiki-Seite, ein Konzept oder eine Identität' -e 'loomux dev mutants <paket>          # Führt Mutationstests über kritische Entscheidungspakete aus' README.de.md
```
  Erwartet: `9`. Zeigt einer der beiden Befehle eine andere Zahl, hat sich der Text seit `e4bf7b5` geändert; der Task hält an und meldet es, statt die Stellen zu raten.

  `README.md`, die Statuszeile unter `## Feature & Status Matrix`, vorher:
```
Loomux is currently executing its staged fusion plan (Stage 1a pilot complete; subsequent stages in active development):
```
  nachher:
```
Loomux is currently executing its staged fusion plan (Stage 1a pilot and Stage 1b-1 data commands complete; subsequent stages in active development):
```
  `README.md`, hinter der Zeile
```
| Semantic QMD Index | Embedding and neural search integration with local caching in `~/.cache/qmd`. | 🚧 **In Migration** (Stage 3) |
```
  eine neue Zeile:
```
| Brain Data Commands | `loomux brain search`, `catalog`, `read`, `neighbors` and `status` over the one registry, held to the Python reference by a recorded case corpus. | ✅ **Implemented** (Stage 1b-1) |
```
  `README.md`, die Zeile unter `## CLI Reference` samt der folgenden Überschrift, vorher:
```
Commands currently active in the Stage 1a pilot vs. specified for subsequent fusion and graph stages:

### Active Commands (Stage 1a Pilot)
```
  nachher:
```
Commands active after Stages 1a and 1b-1 vs. specified for subsequent fusion and graph stages:

### Active Commands (Stages 1a and 1b-1)
```
  `README.md`, hinter `loomux wiki gate                    # gate wiki freshness and structural constraints` (im aktiven Block) fünf Zeilen:
```
loomux brain search "<query>"       # search the visible areas through the qmd daemon (--profile keyword|fast|full)
loomux brain catalog [--scope S]    # the root catalog of the visible areas, or one area's index.md
loomux brain read <path> --scope S  # one file of an area, or one section of it (--section)
loomux brain neighbors <path> --scope S  # incoming and outgoing links of one page
loomux brain status                 # what to know before trusting an answer
```
  `README.md`, im Block unter `### Specified Commands (Second Brain & Services — Stages 2–3 & W1–W5)` diese drei Zeilen streichen (sie sind jetzt aktiv); `loomux brain reconcile` bleibt:
```
loomux brain search "<query>"       # hybrid semantic & keyword search across wiki and ADRs
loomux brain catalog               # list all tracked areas, topics, and identity documents
loomux brain read <path>           # read a wiki page, concept, or identity definition
```

  `README.de.md`, die Statuszeile unter `## Funktions- & Status-Matrix`, vorher:
```
Loomux setzt derzeit seinen mehrstufigen Fusionsplan um (Stufe 1a Pilot abgeschlossen; Folgestufen in aktiver Entwicklung):
```
  nachher:
```
Loomux setzt derzeit seinen mehrstufigen Fusionsplan um (Stufe-1a-Pilot und Datenbefehle der Stufe 1b-1 abgeschlossen; Folgestufen in aktiver Entwicklung):
```
  `README.de.md`, hinter der Zeile
```
| Semantischer QMD-Index | Einbettung lokaler Vektoren und neuronaler Suche mit Caching in `~/.cache/qmd`. | 🚧 **In Migration** (Stufe 3) |
```
  eine neue Zeile:
```
| Brain-Datenbefehle | `loomux brain search`, `catalog`, `read`, `neighbors` und `status` über die eine Registry, an der Python-Referenz durch einen aufgezeichneten Fallkorpus gemessen. | ✅ **Implementiert** (Stufe 1b-1) |
```
  `README.de.md`, die Zeile unter `## CLI-Referenz` samt der folgenden Überschrift, vorher:
```
Aktive Befehle des Stufe-1a-Piloten im Vergleich zu spezifizierten Befehlen der Folge- und Graph-Stufen:

### Aktive Befehle (Stufe 1a Pilot)
```
  nachher:
```
Aktive Befehle nach den Stufen 1a und 1b-1 im Vergleich zu spezifizierten Befehlen der Folge- und Graph-Stufen:

### Aktive Befehle (Stufen 1a und 1b-1)
```
  `README.de.md`, hinter `loomux wiki gate                    # Erzwingt Frische und strukturelle Schranken des Wikis`:
```
loomux brain search "<anfrage>"     # Durchsucht die sichtbaren Bereiche über den qmd-Daemon (--profile keyword|fast|full)
loomux brain catalog [--scope S]    # Wurzelkatalog der sichtbaren Bereiche oder das index.md eines Bereichs
loomux brain read <pfad> --scope S  # Eine Datei eines Bereichs oder einen Abschnitt daraus (--section)
loomux brain neighbors <pfad> --scope S  # Eingehende und ausgehende Links einer Seite
loomux brain status                 # Was man wissen muss, bevor man einer Antwort traut
```
  `README.de.md`, im Block unter `### Spezifizierte Befehle (Second Brain & Dienste — Stufen 2–3 & W1–W5)` diese drei Zeilen streichen; `loomux brain reconcile` bleibt:
```
loomux brain search "<anfrage>"     # Hybride semantische & Keyword-Suche über Wiki und ADRs
loomux brain catalog               # Listet alle verwalteten Bereiche, Themen und Identitäten
loomux brain read <pfad>           # Liest eine Wiki-Seite, ein Konzept oder eine Identität
```
  `README.de.md`, am Ende desselben Blocks, direkt vor `---` und `## Architekturentscheidungen: Was wir bauen, ablösen und weglassen`, steht heute ein kaputter Codeblock: Der Zaun hinter `loomux init` schließt den Block, `loomux dev mutants` steht außerhalb, und der Zaun danach öffnet einen neuen Block, sodass jeder folgende Zaun der Datei um eins verschoben paart; der Abschnitt „Developer & Worktree Tools" der englischen Fassung fehlt. Belegt: `grep -c '^```' README.de.md` zeigt `15`, eine ungerade Zahl, gegen `16` in `README.md`. Vorher:
````
loomux init                         # Richtet Hooks, Einstellungen und Skills in erkannten Agenten ein
```
loomux dev mutants <paket>          # Führt Mutationstests über kritische Entscheidungspakete aus
```
````
  nachher:
````
loomux init                         # Richtet Hooks, Einstellungen und Skills in erkannten Agenten ein
```

### Entwickler- & Worktree-Werkzeuge
```bash
loomux worktree mirror              # Synchronisiert NTFS-Junctions und Spiegel für Agenten-Worktrees
loomux dev covergate --profile <p>  # Prüft das strikte 100-%-Coverage-Tor pro Funktion
loomux dev bench-hooks              # Misst die Latenz der Hook-Ausführung gegen die Grundlinie von < 35 ms
loomux dev mutants <paket>          # Führt Mutationstests über kritische Entscheidungspakete aus
```
````

  Nachher prüfen:
```bash
git diff --numstat README.md README.de.md
```
  Erwartet genau (an einer Kopie des Stands `e4bf7b5` mit genau diesen Änderungen gemessen):
```
15	6	README.de.md
9	6	README.md
```
```bash
grep -c '^loomux brain ' README.md README.de.md
```
  Erwartet: `README.md:6` und `README.de.md:6` (vorher je `4`: drei spezifizierte Zeilen und `reconcile`; nachher fünf aktive Zeilen und `reconcile`).
```bash
grep -c '^```' README.md README.de.md
```
  Erwartet: `README.md:16` und `README.de.md:16`.

- [ ] **Step 12: CLI-Referenz auf den Stand bringen** — `docs/en/cli-reference.md` und `docs/de/cli-reference.md`, Abschnitt 7. Die Einträge `search`, `catalog` und `read` beschreiben heute geplante Befehle; die fünf Datenbefehle ersetzen sie, im Stil des Abschnitts `loomux dev mutants`, den Task 13 in Abschnitt 9 schreibt (Überschrift mit der vollen Aufrufform, ein Satz, fette Stichpunkte, Exit-Codes zuletzt). `loomux brain lint` und `loomux brain reconcile` bleiben unverändert stehen. Die Aufrufformen folgen `TestBrainUsageLines` (Task 10), die Ausgaben der Tabelle „Befehle, Ausgabe, Datenfluss" der Spec, die Fehlerformen `TestBrainUsageErrorsNameTheParserThatRefused` (Task 10). Vorher belegen:
```bash
grep -cxF -e '## 7. Second Brain & Wiki (`loomux brain`)' -e '### `loomux brain search "<query>"`' -e '### `loomux brain catalog`' -e '### `loomux brain read <path>`' docs/en/cli-reference.md
```
  Erwartet: `4`.
```bash
grep -cxF -e '## 7. Second Brain & Wiki (`loomux brain`)' -e '### `loomux brain search "<anfrage>"`' -e '### `loomux brain catalog`' -e '### `loomux brain read <pfad>`' docs/de/cli-reference.md
```
  Erwartet: `4`.

  `docs/en/cli-reference.md`, vorher (von der Abschnittsüberschrift bis vor ``### `loomux brain lint` ``):
```markdown
## 7. Second Brain & Wiki (`loomux brain`)

### `loomux brain search "<query>"`
Performs hybrid semantic and keyword search across wiki pages, ADRs, and concepts.

### `loomux brain catalog`
Lists all managed wiki areas, topic hierarchies, and identity documents.

### `loomux brain read <path>`
Reads a typed wiki page, concept definition, or ADR.
```
  nachher:
```markdown
## 7. Second Brain & Wiki (`loomux brain`)

The five data commands read the areas of the one registry (`registry.toml` in `LOOMUX_STATE_DIR`) and answer as ultra-brain's `brain-mcp` does; a recorded case corpus (`testdata/cases/1b-1`) holds them to it. Until stage 3, a read-only area keeps its artefacts (`index.md`, `graph.json`, `_identities.tsv`) and the reconcile stamp in ultra-brain's state directory: `LOOMUX_LEGACY_BRAIN_DIR`, defaulting to `%LOCALAPPDATA%\brain` on Windows and to `$XDG_STATE_HOME/brain` or `~/.local/state/brain` on POSIX. Until stage 4, an area directory whose `.loomux/config.toml` is missing or has no `[area]` table is read through `.ultra-brain/config.toml` or `.brain.toml`.

- **Channel**: every command takes `--channel local|cloud` (default `local`). An area with `[privacy] mode = "local_only"` does not exist on `cloud`; `[privacy] never` globs apply on every channel.
- **Usage errors** (exit `2`): the usage line, then `loomux brain <command>: error: <reason>` for a missing argument, an invalid choice or `-n` below 1, and `loomux brain: error: <reason>` when the command is missing or unknown or arguments are left over.
- **Runtime errors** (exit `1`): `error: <reason>` on `stderr` and nothing on `stdout` — an unknown scope, a refusal, a missing section, a broken `graph.json` or identity register, a missing or unreadable manifest of any registered area, a missing or broken registry, a search engine that cannot be reached.
- **Advice**: the messages name `brain reindex`, `brain reconcile` and `brain embed`, the commands of ultra-brain, until stage 3 rewrites them.

### `loomux brain search <query> [--scope <scope>] [--profile keyword|fast|full] [-n <n>] [--channel local|cloud]`
Searches the visible areas (`--scope all` by default) through the qmd MCP daemon at `http://localhost:8765/mcp`.

- **Profiles**: `fast` (default) vector search without reranking or query expansion; `keyword` BM25 keyword search; `full` the hybrid chain with expansion and reranking. `-n` (default `5`) must be at least 1.
- **Daemon**: when nothing answers there, loomux starts `qmd mcp --http --daemon --port 8765` detached, writes `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` on `stderr` and waits up to 60 s for it. The backbone defaults to CUDA; a `QMD_LLAMA_GPU` or `QMD_FORCE_CPU` the user set stays untouched.
- **Output**: per hit `brain://<scope>/<path>:<line>  <score>%  <title>`, each snippet line indented by four spaces, then an empty line; exactly `no matches` without hits.
- **Notes**: after the hits, `note: <finding>` lines on `stderr`, in this order — the engine answered empty twice, a hit is missing from its area's identity register, how many hits were withheld, the reconcile stamp is 24 hours old or older.
- **Exit codes**: `0` with hits or `no matches`; `1` for a runtime error, including an engine that does not answer — never an empty answer instead; `2` for a usage error.

### `loomux brain catalog [--scope <scope>] [--channel local|cloud]`
Prints the catalog of the visible areas, or of one area.

- **Output**: with `--scope all` (default) `# brain`, an empty line and `* [<scope>](brain://<scope>/)` per visible area, sorted by scope; with a named scope, that area's `index.md` byte for byte.
- **Exit codes**: `0`; `1` for an unknown scope, a missing `index.md` or another runtime error; `2` for a usage error.

### `loomux brain read <path> --scope <scope> [--section <title>] [--channel local|cloud]`
Prints one file of an area, or one section of it.

- **Reading**: strict UTF-8 with line ends folded to `\n`. `--section` prints from the heading with that title to the next heading of the same or a higher level.
- **Refusals** (exit `1`): `<scope>/<path> leaves the area`; `<scope>/<path> is excluded by [privacy] never`; `<scope>/<path> is the review centre; refused on the cloud channel`; `no section titled '<title>'`.
- **Exit codes**: `0`; `1` for a refusal or another runtime error; `2` for a usage error.

### `loomux brain neighbors <path> --scope <scope> [--channel local|cloud]`
Prints the links into and out of one page, read from the area's `graph.json`.

- **Output**: `incoming: <a>, <b>` and `outgoing: <c>`, each list sorted, `-` for a direction without links.
- **Exit codes**: `0`; `1` for a refusal, an area without `graph.json` (``<scope>: never indexed; run `brain reindex` ``) or another runtime error; `2` for a usage error.

### `loomux brain status [--channel local|cloud]`
Prints what to know before trusting an answer, one line per finding.

- **Lines, in this order**: always the last reconciliation (``last reconcile: never; run `brain reconcile` ``, ``last reconcile: <iso>; older than 24 h, run `brain reconcile` `` or `last reconcile: <iso>`); per visible area in registry order, include globs the search engine does not see, a path that does not exist, an area never indexed, fewer than half of its links resolved, and indexed documents the search engine does not know; across all visible areas, the same content under several paths; once, documents indexed but not yet searchable.
- **Search engine**: two lines ask the qmd CLI (`qmd ls <collection>`, `qmd status`); when it does not answer, the line says so and the command goes on.
- **Exit codes**: `0`; `1` for a runtime error (registry, manifest, stamp, `graph.json`, identity register); `2` for a usage error.
```
  `docs/de/cli-reference.md`, vorher (von der Abschnittsüberschrift bis vor ``### `loomux brain lint` ``):
```markdown
## 7. Second Brain & Wiki (`loomux brain`)

### `loomux brain search "<anfrage>"`
Führt hybride semantische und Keyword-Suche über Wiki-Seiten, ADRs und Konzepte aus.

### `loomux brain catalog`
Listet alle verwalteten Wiki-Bereiche, Themen-Hierarchien und Identitäten auf.

### `loomux brain read <pfad>`
Liest eine typisierte Wiki-Seite, eine Konzept-Definition oder ein ADR.
```
  nachher:
```markdown
## 7. Second Brain & Wiki (`loomux brain`)

Die fünf Datenbefehle lesen die Bereiche der einen Registry (`registry.toml` in `LOOMUX_STATE_DIR`) und antworten wie `brain-mcp` von ultra-brain; ein aufgezeichneter Fallkorpus (`testdata/cases/1b-1`) hält sie daran. Bis Stufe 3 liegen die Artefakte eines schreibgeschützten Bereichs (`index.md`, `graph.json`, `_identities.tsv`) und der Reconcile-Stempel im Zustandsverzeichnis von ultra-brain: `LOOMUX_LEGACY_BRAIN_DIR`, Standard `%LOCALAPPDATA%\brain` unter Windows und `$XDG_STATE_HOME/brain` oder `~/.local/state/brain` unter POSIX. Bis Stufe 4 wird ein Bereichsverzeichnis, dessen `.loomux/config.toml` fehlt oder keine `[area]`-Tabelle trägt, über `.ultra-brain/config.toml` oder `.brain.toml` gelesen.

- **Kanal**: Jeder Befehl nimmt `--channel local|cloud` (Standard `local`). Ein Bereich mit `[privacy] mode = "local_only"` existiert im Kanal `cloud` nicht; `[privacy] never`-Globs gelten in jedem Kanal.
- **Usage-Fehler** (Exit `2`): die Usage-Zeile, dann `loomux brain <befehl>: error: <grund>` bei fehlendem Argument, ungültiger Wahl oder `-n` kleiner 1, und `loomux brain: error: <grund>`, wenn der Befehl fehlt oder unbekannt ist oder Argumente übrig bleiben.
- **Laufzeitfehler** (Exit `1`): `error: <grund>` auf `stderr` und nichts auf `stdout` — ein unbekannter Scope, eine Verweigerung, ein fehlender Abschnitt, ein kaputtes `graph.json` oder Identitätsregister, ein fehlendes oder unlesbares Manifest irgendeines registrierten Bereichs, eine fehlende oder kaputte Registry, eine nicht erreichbare Suchmaschine.
- **Ratschläge**: Die Meldungen nennen `brain reindex`, `brain reconcile` und `brain embed`, die Befehle von ultra-brain, bis Stufe 3 sie umschreibt.

### `loomux brain search <anfrage> [--scope <scope>] [--profile keyword|fast|full] [-n <n>] [--channel local|cloud]`
Durchsucht die sichtbaren Bereiche (Standard `--scope all`) über den qmd-MCP-Daemon unter `http://localhost:8765/mcp`.

- **Profile**: `fast` (Standard) Vektorsuche ohne Reranking und ohne Anfrageerweiterung; `keyword` BM25-Keyword-Suche; `full` die hybride Kette mit Erweiterung und Reranking. `-n` (Standard `5`) muss mindestens 1 sein.
- **Daemon**: Antwortet dort niemand, startet loomux `qmd mcp --http --daemon --port 8765` entkoppelt, schreibt `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` auf `stderr` und wartet bis zu 60 s auf ihn. Das Backbone ist standardmäßig CUDA; ein vom Nutzer gesetztes `QMD_LLAMA_GPU` oder `QMD_FORCE_CPU` bleibt unangetastet.
- **Ausgabe**: je Treffer `brain://<scope>/<pfad>:<zeile>  <score>%  <titel>`, jede Snippet-Zeile um vier Leerzeichen eingerückt, dann eine Leerzeile; ohne Treffer genau `no matches`.
- **Befunde**: nach den Treffern `note: <befund>`-Zeilen auf `stderr`, in dieser Reihenfolge — die Suchmaschine antwortete zweimal leer, ein Treffer fehlt im Identitätsregister seines Bereichs, wie viele Treffer zurückgehalten wurden, der Reconcile-Stempel ist 24 Stunden alt oder älter.
- **Exit-Codes**: `0` mit Treffern oder `no matches`; `1` bei einem Laufzeitfehler, auch wenn die Suchmaschine nicht antwortet — nie eine leere Antwort stattdessen; `2` bei einem Usage-Fehler.

### `loomux brain catalog [--scope <scope>] [--channel local|cloud]`
Gibt den Katalog der sichtbaren Bereiche oder eines Bereichs aus.

- **Ausgabe**: mit `--scope all` (Standard) `# brain`, eine Leerzeile und je sichtbarem Bereich `* [<scope>](brain://<scope>/)`, nach Scope sortiert; mit einem benannten Scope das `index.md` dieses Bereichs byte-gleich.
- **Exit-Codes**: `0`; `1` bei unbekanntem Scope, fehlendem `index.md` oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain read <pfad> --scope <scope> [--section <titel>] [--channel local|cloud]`
Gibt eine Datei eines Bereichs oder einen Abschnitt daraus aus.

- **Lesen**: streng UTF-8, Zeilenenden zu `\n` gefaltet. `--section` gibt ab der Überschrift mit diesem Titel bis zur nächsten Überschrift gleicher oder höherer Ebene aus.
- **Verweigerungen** (Exit `1`): `<scope>/<pfad> leaves the area`; `<scope>/<pfad> is excluded by [privacy] never`; `<scope>/<pfad> is the review centre; refused on the cloud channel`; `no section titled '<titel>'`.
- **Exit-Codes**: `0`; `1` bei einer Verweigerung oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain neighbors <pfad> --scope <scope> [--channel local|cloud]`
Gibt die Links in eine Seite hinein und aus ihr heraus aus, gelesen aus dem `graph.json` des Bereichs.

- **Ausgabe**: `incoming: <a>, <b>` und `outgoing: <c>`, jede Liste sortiert, `-` für eine Richtung ohne Links.
- **Exit-Codes**: `0`; `1` bei einer Verweigerung, einem Bereich ohne `graph.json` (``<scope>: never indexed; run `brain reindex` ``) oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain status [--channel local|cloud]`
Gibt aus, was man wissen muss, bevor man einer Antwort traut, eine Zeile je Befund.

- **Zeilen, in dieser Reihenfolge**: immer der letzte Abgleich (``last reconcile: never; run `brain reconcile` ``, ``last reconcile: <iso>; older than 24 h, run `brain reconcile` `` oder `last reconcile: <iso>`); je sichtbarem Bereich in Registry-Reihenfolge Include-Globs, die die Suchmaschine nicht sieht, ein Pfad, der nicht existiert, ein nie indizierter Bereich, weniger als die Hälfte aufgelöster Links und indizierte Dokumente, die die Suchmaschine nicht kennt; über alle sichtbaren Bereiche derselbe Inhalt unter mehreren Pfaden; einmal Dokumente, die indiziert, aber noch nicht durchsuchbar sind.
- **Suchmaschine**: Zwei Zeilen fragen die qmd-CLI (`qmd ls <collection>`, `qmd status`); antwortet sie nicht, sagt die Zeile das, und der Befehl läuft weiter.
- **Exit-Codes**: `0`; `1` bei einem Laufzeitfehler (Registry, Manifest, Stempel, `graph.json`, Identitätsregister); `2` bei einem Usage-Fehler.
```
  Nachher prüfen:
```bash
git diff --numstat docs/en/cli-reference.md docs/de/cli-reference.md
```
  Erwartet genau (an einer Kopie des heutigen Stands mit genau diesen Änderungen gemessen; Task 13 ändert nur Abschnitt 9):
```
39	6	docs/de/cli-reference.md
39	6	docs/en/cli-reference.md
```
```bash
grep -c '^### `loomux brain ' docs/en/cli-reference.md docs/de/cli-reference.md
```
  Erwartet: `docs/en/cli-reference.md:7` und `docs/de/cli-reference.md:7` (vorher je `5`: `search`, `catalog`, `read`, `lint`, `reconcile`).

- [ ] **Step 13: Abschlussprüfung** gegen „Stufe 1b-1 ist fertig, wenn" der Spec:
  1. Fälle grün oder freigegeben in der Liste:
```bash
go test ./internal/cli/ -run '^TestRecordedCasesOfStage1b1$' -count=1
```
     Erwartet: `ok  	github.com/xidus90/loomux/internal/cli`. Jede Zeile der Abweichungstabelle in `stufe-1b-1.md` steht mit leerer Freigabespalte — **die Freigaben füllt der Mensch**; bis dahin ist Punkt 1 offen.
  2. Coverage 100 % je Funktion, jede Ausnahme begründet. Das Tor läuft in Step 14 nach dem Stagen; hier die Liste der Ausnahmen:
```bash
grep -rc '^//coverage:exempt' --include='*.go' cmd internal | grep -v ':0$' | LC_ALL=C sort
```
     Erwartet genau diese 14 Dateien mit 24 Stellen — heute 13 Dateien mit 23 Stellen (gemessen am 2026-09-15), dazu der Prozesseinstieg des Fakes aus Task 11:
```
cmd/loomux/main.go:1
internal/brain/wiki/lint.go:1
internal/cases/runner.go:2
internal/dev/benchhooks/benchhooks.go:1
internal/dev/fakeqmd/qmd/main.go:1
internal/dev/importcases/importcases.go:1
internal/dev/recordcase/recordcase.go:2
internal/hooks/post_edit.go:4
internal/hooks/status.go:1
internal/hooks/worktree.go:2
internal/sessions/state.go:1
internal/testlock/lock_other.go:2
internal/testlock/lock_windows.go:2
internal/worktree/junction/junction_windows.go:3
```
     Die umgezogenen ub-Pakete tragen kein `//coverage:exempt`, und die Tasks 2, 8, 9 und 13 setzen keins, auch nicht an `mutants.GoTest` (Task 13 misst es mit 100,0 %). Jede weitere Datei oder abweichende Zahl ist ein Befund; jede Stelle nennt ihren Grund in derselben Zeile.
  3. Mutationsrunde gelaufen, jeder Überlebende dokumentiert:
```bash
grep -n '^## Mutationsrunde$' docs/.superpowers/parity/stufe-1b-1.md
```
     Erwartet: genau eine Zeile, die `grep -E '^[0-9]+:## Mutationsrunde$'` trifft (die Zeilennummer hängt am Stand der Datei); die Pakettabelle darunter hat dreizehn Zeilen einschließlich `internal/brain/guard`, `internal/hooks`, `internal/config`, `internal/cases`, `internal/brain/wiki`, und jede Überlebenden-Zeile hat eine Erledigung.
  4. Werte eingetragen:
```bash
grep -c '^## .* — The Brain Data Commands on the Real Registry$' docs/en/benchmarks.md
```
     Erwartet: `1`.
```bash
grep -c '^## .* — Die Brain-Datenbefehle auf der echten Registry$' docs/de/benchmarks.md
```
     Erwartet: `1`. Beide Einträge tragen warm und kalt je Profil, `brain status`, `inittrace` vor und nach, den ausgewiesenen Registeranteil und das Urteil zu ≤ 150 ms (Task 15 Step 13).
  5. Rauchtest:
```bash
sed -n '/^## Pilot$/,/^## /p' docs/.superpowers/parity/stufe-1b-1.md | grep -c '^- \[x\] `brain '
```
     Erwartet: `9` — die Steps 3 bis 9 dieses Tasks, Abschnitt `## Pilot` vollständig abgehakt (gezählt von der Überschrift `## Pilot` bis zur nächsten `## `-Überschrift oder zum Dateiende, damit Kästchen anderer Abschnitte nicht mitzählen). Eine kleinere Zahl heißt: Ein Punkt ist nicht wie erwartet ausgefallen und steht als Befund darunter.

- [ ] **Step 14: Stagen, Tor, Commit**
```bash
git add docs/.superpowers/parity/stufe-1b-1.md README.md README.de.md docs/en/cli-reference.md docs/de/cli-reference.md
```
  Erwartet: keine Ausgabe.
```bash
sh .githooks/pre-commit
```
  Erwartet: Exit 0 (gofmt, vet, Tests mit Coverage, `covergate` ohne Ausgabe, Pilot-Binary neu gebaut).
```bash
git branch --show-current
```
  Erwartet: `sdd-1b-1` (im Hauptcheckout nach Task 0: `master`).
```bash
git rev-parse --short HEAD
```
  Erwartet: der Kurz-Hash aus Step 1.
```bash
git diff --cached --stat
```
  Erwartet: genau die fünf Dateien.
```bash
printf '%s\n' 'Smoke-test the brain commands on the real registry' '' 'The five data commands run once each against the registry the human' 'extended in the measurement task. The READMEs name them as active, and' 'the CLI reference describes their forms, output and exit codes.' > "$TEMP/commit-task16.txt"
```
  Erwartet: keine Ausgabe.
```bash
git commit -F "$TEMP/commit-task16.txt"
```
  Erwartet: Exit 0; der Commit löst das Tor noch einmal aus, das ist gewollt.
```bash
git log -1 --format='%an <%ae>'
```
  Erwartet: die Nutzeridentität, kein Modell und kein Agent.
```bash
git log -1 --format=%B
```
  Erwartet: genau die Nachricht aus `$TEMP/commit-task16.txt`, keine `Co-Authored-By`-Zeile.

- [ ] **Step 15 (Mensch):** die Freigaben der Paritätsliste; der Rauchtest, falls der Nutzer ihn sich in Step 2 vorbehalten hat; die Entscheidung, `sdd-1b-1` nach `master` zu übernehmen; danach den Worktree entfernen **und im selben Zug** dessen Block `project/loomux-sdd-1b1` aus `%LOCALAPPDATA%\loomux\registry.toml` streichen — ein registrierter Bereich ohne Manifest lässt jeden `loomux brain`-Befehl mit `error:` enden.

- [ ] **Step 16: Bericht** — nennt, wer den Rauchtest gefahren hat (Step 2), je Befehl Exit-Code und Abweichung von der Erwartung, den Stand der fünf Abschlusspunkte (Punkt 1 offen bis zur Freigabe), die README-Änderungen und den neuen Abschnitt 7 beider CLI-Referenzen. **Paritätszeilen, die dieser Task schafft:** keine; ein unerwarteter Ausfall eines Rauchtestpunkts ist ein Befund.
