# loomux Stufe 3c — Implementierungsplan: Brain pflegen

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Wiki-Schicht bleibt in Form, ohne dass ultra-brain noch
gebraucht wird. `loomux brain check file|bundle|all` prüft Seiten, Bündel und
die Föderation nach OKF, Hausregeln und Föderationsregeln. `loomux lint
--scope all|<scope>` lintet jedes registrierte Bündel. `loomux wiki
init|types|retype` legt ein Bündel an, zählt die Seitentypen über alle
Bereiche und benennt einen Typ in einem Bündel um. `loomux serve` holt einen
fälligen `reconcile` selbst nach und hängt den Hinweis an seine Antworten.

**Architecture:** Vier Stränge, die nur am Ende in der CLI und der Fallsuite
zusammenlaufen.
1. **Prüfen** (Umzug): `ultra-brain/pkg/check/{okf,house,run}` zieht nach
   `internal/brain/check/{okf,house,run}`. `finding.go` liegt seit 1a dort,
   `internal/brain/wiki` und `internal/config` sind feldgleich. Es fehlt
   **keine** Abhängigkeit.
2. **Wiki-Werkzeuge** (Neuschrift aus Python): die Ränge der Seitentypen
   (`types.py`), die Volkszählung (`census.py`), das Umbenennen (`retype.py`)
   und die CLI `loomux wiki`. Das Anlegen (`InitBundle`) liegt seit 3a in
   `internal/brain/wiki/scaffold.go`.
3. **Lint über Bereiche** (Neuschrift aus `cli.py:_lint_targets`/`_lint`):
   `loomux lint --scope`. Die Regeln kommen aus Strang 1.
4. **Upkeep** (Neuschrift aus `daemon/server.py:Upkeep`): ein Durchgang beim
   Start, danach alle 24 h. Tool-Aufrufe warten auf den ersten Durchgang, und
   jede Antwort bekommt ihre Hinweiszeilen.

**Tech Stack:** Go ≥ 1.25, `github.com/BurntSushi/toml`, `gopkg.in/yaml.v3`,
`github.com/modelcontextprotocol/go-sdk` (alle schon in `go.mod`). Als Tor
dient `loomux check precommit`. Neue Abhängigkeiten gibt es keine.

**Spec:** `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md`,
Abschnitt „3c im Einzelnen“. Mit diesem Plan kommen dort „Befunde 3c“ und
„Entscheidungen 3c“ hinzu (Task 0).

## Befunde, gegen den Code gelesen am 2026-09-23

Gelesen wurde gegen loomux `e3cab29` und ultra-brain `08d985a` (Remote-Stand
von `master`). Der Tag `loomux-3-source` (`3cc72d2`) liegt nur auf dem Rechner
des Menschen; Task 0 gleicht die Befunde gegen ihn ab.

**B1. Für `check` gibt es keine Python-Referenz.** Das einzige Konsolenskript
der Python-Form ist `brain-mcp` (`pyproject.toml:19-24`), und dessen
Kommentar sagt, es kenne „kein `guard` und kein `check`“. `check` gibt es nur
im Go-Binary (`cmd/brain/main.go:93-94`, `checkCommand` `:921-980`, Hilfe
`:193-196`). Für diese Befehle ist also die **Go-Form die Referenz**,
aufgezeichnet über `loomux dev record-case --exe <brain.exe>`. Stufe 1a hat
`brain guard|lint|wiki-gate` genauso aufgezeichnet (`testdata/cases/1a-map.toml:16-24`).
Die Regel der Spec, „Python für alles“, gilt für `check` nicht und steht als
Abweichung in der Akte.

**B2. Der Name `loomux check` ist belegt.** `check` führt zu `checkCommand`
(`internal/cli/check.go:53`). Dort sind `check all` („alle vier Arten“,
`internal/verify/plan.go:82`) und der reservierte Profilname `all`
(`internal/verify/schema.go:36`) schon vergeben. Die READMEs führen den Befehl
bereits als `loomux brain check` (`README.md:191`, `README.de.md:193`). Unter
`loomux brain` kollidiert er mit nichts (`internal/cli/brainargs.go:21`).
→ Entscheidung E1.

**B3. `check code` ist durch die Prüfkette überholt.** Die Lanes
`gofmt → ruff → mypy → pytest → coverage` aus `[check].lanes`
(`pkg/check/code/runner.go:11-17`) hat loomux alle in
`internal/verify/presets.toml:40-78`, dazu Reihenfolge, Blockade und
Coverage-Tor. `Manifest.Lanes` wird geparst (`internal/config/manifest.go:191-215`),
aber nirgends gelesen. → Entscheidung E2.

**B4. Die Ausgabe von `check`** ist Text auf stdout in der Form `[severity]
axis/rule scope/relative: message` (`check.Render`, `internal/brain/check/finding.go:106`).
Notizen erscheinen nur mit `--notes`. Der Exit ist 0, 1 bei mindestens einem
Fehler-Befund und 2, wenn der Lauf nicht stattfand (fehlende oder unbekannte
Breite, Pfad fehlt oder ist keine reguläre Datei, Registry unlesbar, Scope
unbekannt; `main.go:1047-1100,1132`). Flags dürfen überall stehen; loomux hat
dafür `parseInterspersed` (`internal/cli/interspersed.go:18`).

**B5. Umfang des Umzugs** (ohne / mit Tests): `okf` 784 / 1.805,
`house` 1.301 / 3.385, `run` 665 / 1.882, zusammen **2.750 / 7.072** Zeilen.
Fixtures gibt es nicht; die Tests bauen ihre Welten inline (`writeBundle`,
`vault`, `signpost`) und importieren nur `check`, `config`, `wiki` und
Geschwisterpakete. Ausnahme ist `run/inner_parallel_test.go`: Er misst
ultra-brains eigenes `docs/wiki` und sagt selbst, dass die Variante nicht
ausgeliefert wird (`:7-11`). Er zieht nicht mit.

**B6. `CheckAll` ist parallel**, eine Goroutine je Bereich (`run.go:182-222`),
mit einem gemeinsamen `now`. Die Föderationsregeln laufen erst, wenn alle
Bündel gelesen sind. `check bundle` braucht die **ganze** Registrierung, nicht
nur den Gegenstand (`run.go:140-151`).

**B7. Die Namen der Wiki-Werkzeuge sind in der Spec falsch gezählt.**
Python kennt drei Befehle, nicht vier:
- `brain types` druckt `census()` (`cli.py:570,754,1669-1677`). „types“ und
  „census“ sind **ein** Befehl.
- `brain retype --scope --from --to` (`cli.py:575-582,876-907`).
- `brain wiki init --scope` → `init_bundle` (`cli.py:659-663,859-873`). Einen
  Befehl `scaffold` gibt es nicht.

`types` und `retype` stehen in Python auf oberster Ebene. Unter `wiki` gibt es
nur `init` und `gate` (`cli.py:661,664`). → Entscheidung E3.

**B8. Das Anlegen ist schon umgezogen.** `wiki.InitBundle`
(`internal/brain/wiki/scaffold.go:11-14,96-112`) schreibt über
`lock.ReplaceText` und wird von `area add` benutzt (`internal/cli/area.go:330,335`).
Für `wiki init` fehlt nur die Verdrahtung: Registry lesen, die zwei
Weigerungen, die geschriebenen Pfade drucken.

**B9. Den Seitentypen fehlen die Ränge.** loomux hat nur die Vereinigung
`knownTypes`/`KnowsType` (`internal/config/manifest.go:47-84,232-244`).
`census` und `retype` brauchen die Stufen `core`, `catalogue`, `origin`,
`declared` und `unknown` aus `types.py` sowie `ALIASES` (`Design Decision`
und `Architecture Decision` werden zu `Decision`, `types.py:30-33`).

**B10. `retype` schreibt ohne Sperre.** Es läuft über `write_if_changed`
(`writer.py:6-23`), faltet eine geschriebene Seite ganz auf LF
(`retype.py:74`) und ersetzt nur einen ungequoteten Skalar in `type:`
(`retype.py:79-109`, Regex `:91`). Gerüstdateien, kaputtes Frontmatter,
nicht dekodierbares UTF-8 und gequotete oder gefaltete Werte überspringt es.
loomux schreibt über `lock.ReplaceText`. Das ist eine Abweichung mit Grund,
und gemeldet wird nur, was sich in Bytes geändert hat.

**B11. `lint` heute und in der Referenz.**
- loomux verlangt genau eine Datei (`internal/cli/wiki.go:22-24`, Exit 2).
- Python lintet ohne Pfad oder mit `--scope all` jedes Bündel mit
  `wiki_path`, mit `--scope S` genau eines (`cli.py:1562-1593`). Weigerungen:
  der Scope ist unbekannt, es gibt keinen Wiki-Pfad, oder das Wiki fehlt
  (`… run \`brain wiki init --scope …\` first`, `cli.py:1573-1580`).
- Ausgabe von Python: je Bereich eine Kopfzeile, darunter `  {relative}:{rule}: {message}`
  oder `  no findings`. Am Ende steht `no findings` (Exit 0) oder `{n}
  findings ({e} errors, {w} warnings)` (Exit 1 nur bei Fehlern,
  `cli.py:1655-1666`).
- Ein Argument, das eine Datei ist oder auf `.md` endet, geht an
  `lint_single_file` (`lint.py:590-611`). Das ist der 1a-Pfad von
  `loomux lint <file>` und bleibt, wie er ist.

**B12. Regeln, die loomux fehlen.** `lint.py` `RULES` (`:539-552`) hat zwölf
Regeln, loomux `lint.go` fünf (`missing-type`, `conflict-count`,
`outside-area`, `dead-link`, `orphan`). Es fehlen `broken-frontmatter`,
`no-sources`, `unlisted-area`, `wrong-direction`, `untouched`, `stale`,
`implemented-without-commit` und `long-planned`. Die Go-Form hat sie alle als
`house/*` (`page.go:105,161,228,261,298`, `bundle.go:234,284,325,376,415,454`,
`federation.go:242,327`). Der Lint über Bereiche baut deshalb auf Strang 1 auf
und bekommt keine zweite Regelschicht.

**B13. `BundleContext` liegt brach.** `DeclaredTypes` und `IsProject` gibt es
(`internal/brain/wiki/model.go:100-106`), aber `LintBundle(wikiRoot)`
(`lint.go:101`) nimmt keinen Kontext entgegen.

**B14. Upkeep in der Referenz** (`daemon/server.py:106-310`):
- Pro Prozess gibt es einen Upkeep, gestartet mit `serve()` (`:508,534`).
  `keep_up` fährt den ersten Durchgang und schläft danach 24 h (`:163-167`).
  Fällig ist er, wenn der Stempel fehlt oder `now - stamp >= 24h` ist
  (`_due`, `:200-203`; `RECONCILE_INTERVAL`, `core.py:27`). Ist nichts
  fällig, hat ein Mensch von Hand abgeglichen, und der Bericht wird `None`
  (`:176-183`).
- Der erste Durchgang ist ein Tor. `call_tool` wartet vor **jedem** Werkzeug
  auf `caught_up()` und meldet vorher `CATCH_UP_NOTICE` als Fortschritt
  (`:43-47,387-400`). Das Tor öffnet sich im `finally`, also auch bei einem
  Fehler (`:195-198`).
- Fehler (`ReconcileError`, `RegistryError`, `OSError`) landen in `failure`.
  Sie werden nie geworfen, und der nächste gute Durchgang löscht sie
  (`:184-194`). Ohne Registry oder bei leerer Registry gibt es keinen Bericht
  (`:320-334`). Ohne Prüfzentrum entsteht eine Fehlerzeile.
- Die Zeilen (`_trailer`, `:445-469`): `status` bekommt `notes` ganz,
  `search`, `read`, `catalog` und `neighbors` bekommen `headline`, alle außer
  `search` zusätzlich `core.stale_reconcile`. Jede Zeile beginnt mit `! ` und
  kommt an den ersten Textblock. Fehlerantworten bleiben unberührt
  (`_with_notes`, `:472-487`). Auf dem Cloud-Kanal wird die Ursache durch
  „(the cause is named on the local channel)“ ersetzt, und Bereiche sind nach
  Sichtbarkeit gefiltert (`:251-262,286-317`).
- Die Texte nennen `brain reconcile` und `brain cases`.

**B15. Upkeep in loomux.**
- `serve` importiert `maintenance` noch nicht (`internal/serve/serve.go:14-22`).
- Den Abgleich gibt es: `maintenance.Reconcile(areas, lookup, now) (Report, error)`
  (`internal/brain/maintenance/reconcile.go:113`) und `ErrNoReviewCentre` (`:89`).
- Ebenso die Stempel-Leser: `search.ReadLastRun`, `Stale`, `StaleReconcile`
  und `ReconcileInterval` (`internal/brain/search/stamp.go:17,33,54,60`).
- Die Verdrahtung der CLI steht in `catchUp` (`internal/cli/maintenance.go:60-81`),
  mit `time.Now().UTC()` wegen der Fall-IDs.
- Die Hinweiszeilen gehören in `handler`/`withFindings`
  (`internal/serve/brain/tools.go:39-99`).
- Die 1b-2-Welten stempeln `2999-01-01`, damit nichts läuft
  (`parity/stufe-1b-2.md:113-123`).
- Ein Fall mit fälligem Stempel lässt sich nicht deterministisch aufzeichnen:
  Die Referenz gleicht ab, und der Text trägt die Zeit. Upkeep wird deshalb in
  Go mit falscher Uhr geprüft, dazu höchstens Fälle mit `compare = outcome`.

**B16. Die Regel für den Importgraphen gilt weiter.**
`internal/cli/imports_test.go:65,81-100` verbietet `hooks` den Import von
`apply`, `evidence`, `maintenance`, `vcs`, `serve` und `bridge`. Mit 3c
kommen `brain/check/{okf,house,run}` auf die Liste. `serve` darf
`maintenance` importieren (Spec `:147`).

**B17. `--state-dir` nimmt loomux nicht an** (`docs/en/cli-reference.md:411`).
Jeder Python-Befehl dieser Stufe hat das Flag. Die Übersetzung lässt es weg,
denn der Rekorder setzt den Zustandsort über die Umgebung. Das steht als
Abweichung in der Akte.

**Berichtigt in Task 0 (2026-09-23).** B5, B12, B14 und B17 sind teilweise
falsch, dazu kommt B18; die Berichtigungen und die verschobenen Anker stehen
in `parity/stufe-3c.md`, „Prüfung gegen den Tag“. Was davon die Tasks
ändert, ist dort eingearbeitet: Task 4 (Importe der Tests), Task 10 (E5,
Exit 1), Task 11 und 12 (Upkeep).

## Entscheidungen 3c — freigegeben am 2026-09-23

Alle acht sind freigegeben, E5 in geänderter Form.

| # | Frage | Vorschlag | Begründung |
|---|---|---|---|
| E1 | Name von `check` | `loomux brain check file\|bundle\|all` | `loomux check all` ist vergeben (B2). Die READMEs führen den Namen schon so. Die Brain-Skills aus Stufe 4 rufen `brain check`, und `brain check` wird zu `loomux brain check` mit einer einzigen Übersetzungsregel |
| E2 | `check code` | Entfällt | Die Prüfkette aus 2a deckt es ab (B3). Eine zweite Lane-Konfiguration stünde neben `[verify]`. `Manifest.Lanes` bleibt geparst und ungelesen, bis Stufe 4 (`migrate`) es nach `[verify]` überträgt. In der Akte als Wegfall |
| E3 | Namen der Wiki-Werkzeuge | `loomux wiki init\|types\|retype`, alle unter `wiki` | Das sind die drei Befehle der Referenz (B7). `init` heißt wie in der Referenz. `types` und `retype` wandern unter `wiki`, weil loomux oben keine Wiki-Verben ohne Präfix mehr aufnimmt. `census` und `scaffold` sind keine eigenen Befehle; die Spec wird korrigiert |
| E4 | Referenz für `check` | Go-Binary `brain.exe` vom Tag `loomux-3-source` | Eine Python-Form gibt es nicht (B1). Abweichungen der loomux-Form von der Go-Form heilt der Plan, er übernimmt sie nicht |
| E5 | Regeln für `lint --scope` | **Freigegeben geändert (2026-09-23):** die zwölf Regeln von `lint.py` mit deren Texten, Namen und Auslösern als eigener Regelsatz für `lint --scope`; `lint.go` behält die Go-Form von 1a (Reichweite nachentschieden am 2026-09-23); `house` wird nicht abgebildet | Task 0 fand 4 von 13 Texten ungleich, 2 Ids ungleich, `broken-frontmatter` nicht in `house` und abweichende Auslöser (Akte, „Texte house gegen lint.py“). Eine Abbildung wäre eine versteckte zweite Regelschicht |
| E6 | Einzeldatei `loomux lint <file>` | Bleibt, wie 1a sie gebaut hat | Die 1a-Fälle sind grün. `--scope` ist ein neuer Pfad daneben, kein Umbau |
| E7 | Upkeep-Texte | `brain reconcile`/`brain cases` werden zu `loomux reconcile`/`loomux cases` | Anders als 1b-1 hat 3c keinen aufgezeichneten Fall, der den alten Text festhält. `search.ReconcileAdvice` bleibt, solange die 1b-1-Fälle ihn tragen, und wird mit Stufe 4 umgestellt |
| E8 | Upkeep und `reindex` | Upkeep ruft **nur** `reconcile` | So hält es die Referenz. Die Auflage aus `parity/stufe-3a.md:257-275` (Pfad der qmd-Konfiguration als Argument) greift erst, wenn der Upkeep einmal `reindex` rufen soll |

## Global Constraints

- **Sprache — `AGENTS.md` gilt.** Code, Kommentare, Fehlermeldungen und
  Commit-Nachrichten sind englisch. Deutsch sind nur die Dokumente unter
  `docs/.superpowers/` und die deutschen Handbuchseiten. Übernommene
  Nutzermeldungen der Referenz sind in 3c englisch und bleiben wörtlich.
- **Commit-Nachrichten nennen kein Arbeitspapier**: kein `(3c)`, kein
  „Task 7“. Der Scope nennt ein Codegebiet: `check`, `wiki`, `lint`, `serve`,
  `cli`, `cases`. `parity` ist ein Arbeitspapier und kein Scope; Commits,
  die nur Akte, Plan oder Spec ändern, heißen schlicht `docs:`.
- **TDD, 100 % Coverage je Funktion.** Jeder Ausschluss bekommt
  `//coverage:exempt <Grund>` direkt über `func`. Umziehende Pakete bringen
  ihre Tests mit und werden beim Umzug auf 100 % gehoben.
- **Die Referenz** ist ultra-brain am Tag `loomux-3-source` (`3cc72d2`). Für
  `lint`, `wiki init`, `types`, `retype` und den Upkeep ist es Python, für
  `check` das Go-Binary (E4).
- **Zustandsort:** Gelesen wird über `config.NewArtifactLookup` (neu zuerst,
  alt als Rückfall), geschrieben nur nach `LOOMUX_STATE_DIR`.
- **Startzeit-Regel:** kein `init()`, und keine Paketvariable parst
  eingebettete Daten. Die Regeltabellen von `okf` und `house` sind Literale.
- **`hooks` importiert nichts aus `brain/check/{okf,house,run}`**, und
  `imports_test.go` wird entsprechend erweitert (B16).
- **Geschrieben wird nur atomar**, über `lock.ReplaceText` (`retype`, `wiki init`).
- **Commits:** Mehrzeilige Nachrichten gehen über eine Datei im Scratchpad und
  `git commit -F <pfad>`. Kein `Co-Authored-By:` auf ein Modell.
- **Ein Shell-Befehl je Aufruf.** Vor jedem Commit werden Zweig und HEAD
  gelesen (`git status -sb`).
- **Subagenten:** `model: "opus"`, `effort: "low"`, beides explizit gesetzt.
- **Vorgeklärt, damit kein Subagent es zweimal erhebt:**
  - `internal/brain/check/finding.go` ist codegleich mit
    `pkg/check/finding.go`, nur der Paketkommentar weicht ab. Er zieht nicht
    noch einmal um.
  - `internal/brain/wiki/{parse,model}.go` sind byte-gleich mit `pkg/wiki`,
    `lint.go` weicht in einer Zeile ab. Die Imports von `pkg/wiki` werden zu
    `internal/brain/wiki`.
  - `config.Area` ist feldgleich, und `config.Manifest` ist eine Obermenge
    (`Path`, `LayoutInbox` kommen hinzu).
  - `cases.SplitCommand` wird nur von `pkg/check/code` gebraucht, und das
    entfällt (E2).

## File Structure

| Datei | Verantwortung | Task |
|---|---|---|
| `docs/.superpowers/parity/stufe-3c.md` | Akte | 0, laufend |
| `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md` | „Befunde 3c“, „Entscheidungen 3c“, Namen korrigiert | 0 |
| `internal/brain/check/okf/{errors,soft}.go` | Umzug der OKF-Achse | 1 |
| `internal/brain/check/house/page.go` | Umzug der Hausregeln je Seite | 2 |
| `internal/brain/check/house/bundle.go` | Umzug der Hausregeln je Bündel | 2 |
| `internal/brain/check/house/federation.go` | Umzug der Föderationsregeln | 3 |
| `internal/brain/check/run/{run,targets}.go` | Umzug: `CheckFile`, `CheckBundle`, `CheckAll`, `Targets`, ohne `CheckCode` | 4 |
| `internal/cli/brainargs.go`, `internal/cli/braincheck.go` | `loomux brain check` | 5 |
| `internal/brain/wiki/types.go` | Ränge, `RankOf`, `Aliases` | 6 |
| `internal/brain/wiki/census.go` | Volkszählung und Darstellung | 7 |
| `internal/brain/wiki/retype.go` | Umbenennen im Frontmatter | 8 |
| `internal/cli/wikicmd.go` | `loomux wiki init\|types\|retype` | 9 |
| `internal/brain/pytext/url.go` | `SplitURL`, `Unquote`, `DecodeReplace` | 10 |
| `internal/brain/wiki/sweeppage.go`, `sweep.go` | Leser und zwölf Regeln von `lint.py` (E5) | 10 |
| `internal/brain/wiki/sweep.go` | Ziele und Bericht des Lints über Bereiche | 10 |
| `internal/cli/wiki.go` | `loomux lint --scope` | 10 |
| `internal/serve/upkeep.go` | `Upkeep`: Durchgang, Tor, Fehler, Zeilen | 11 |
| `internal/serve/brain/tools.go`, `internal/serve/serve.go` | Tor und Hinweiszeilen an den Werkzeugen | 12 |
| `internal/cli/imports_test.go` | Importregeln für `check/*` und `serve → maintenance` | 5, 12 |
| `testdata/cases/3c{,-source,-worlds}/`, `testdata/cases/3c-map.toml` | Aufnahmen, Übersetzung, Welten | 13 |
| `internal/cli/cases_3c_test.go` | Fallsuite 3c | 13 |

---

### Task 0: Vorbedingungen und Freigaben — kein Code

**Files:**
- Create: `docs/.superpowers/parity/stufe-3c.md`
- Modify: `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md`

- [ ] **Step 1: Arbeitsort lesen**

Run: `git status -sb`
Expected: ein Zweig, nicht `master`, sauber.

- [ ] **Step 2: Referenz-Tag prüfen**

Run: `git -C ../ultra-brain rev-parse --short loomux-3-source`
Expected: `3cc72d2`

Run: `git -C ../ultra-brain log --oneline loomux-3-source..master -- pkg/check src/brain/wiki src/brain/cli.py src/brain/daemon`
Expected: leer. Ist die Ausgabe nicht leer, anhalten und den Menschen
fragen, ob der Tag verschoben wird.

- [ ] **Step 3: Befunde gegen den Tag abgleichen**

Die Zeilenangaben B1–B14 stammen von `08d985a`. Die Anker werden am Tag
nachgeschlagen (`git -C ../ultra-brain show loomux-3-source:<pfad>`), und
jede Verschiebung wird in der Akte berichtigt.

- [ ] **Step 4: Meldungstexte `house` gegen `lint.py` vergleichen (für E5)**

Je Regel aus B12 die Meldung in `pkg/check/house/*.go` neben die in
`src/brain/wiki/lint.py` legen. Ergebnis als Tabelle in der Akte: Regel ·
Text Go · Text Python · gleich/ungleich.

- [ ] **Step 5: Halt — Freigabe E1–E8**

Der Mensch bestätigt oder ändert jede Zeile der Tabelle „Entscheidungen 3c“.
Das Ergebnis steht mit Datum in der Akte. Ohne Freigabe beginnt kein Code-Task.

- [ ] **Step 6: Akte anlegen und Spec nachtragen**

`parity/stufe-3c.md`, deutsch, mit diesen Abschnitten:
- „Vorbedingungen, festgestellt am <Datum>“
- „Befunde“ (B1–B17, berichtigt)
- „Entscheidungen“ (E1–E8 mit Freigabedatum)
- „Texte house gegen lint.py“ (Step 4)
- eine leere Abweichungsliste (Fall · Alt · Neu · Begründung), vorbelegt mit:
  `check` gegen Go statt Python (E4), `check code` entfällt (E2), `--state-dir`
  (B17), `retype` unter Sperre und atomar (B10), die Upkeep-Texte nennen
  `loomux` (E7).

In der Spec entsteht unter „3c im Einzelnen“ ein Unterabschnitt „Befunde 3c“
mit einem Verweis auf die Akte. Die Namensliste wird auf `wiki
init|types|retype` berichtigt, und `check` wird zu `brain check`.

Die freigegebenen Entscheidungen, die eine Stufe umschneiden (E2 Wegfall
von `check code`, E3 die Namen der Wiki-Werkzeuge), stehen nach `AGENTS.md`
zuerst in der Fusions-Spec: Die Zeile 3c der Stufentabelle, Nachtrag #1 und
die Wegfall-Liste werden hier nachgeführt, nicht erst in Task 16. Die Zeile
3c in `docs/{de,en}/migration.md` folgt im selben Commit.

- [ ] **Step 7: Commit**

```bash
git add docs/.superpowers/parity/stufe-3c.md docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md docs/de/migration.md docs/en/migration.md
```

```bash
git commit -m "docs: open the record for keeping the brain in shape"
```

---

### Task 1: `internal/brain/check/okf` — Umzug der OKF-Achse

**Files:**
- Create: `internal/brain/check/okf/errors.go`, `soft.go` und die Tests dazu
- Quelle: `ultra-brain/pkg/check/okf/` (784 / 1.805 Zeilen)

- [ ] **Step 1: Tests umziehen und rot sehen**

Die `_test.go`-Dateien werden kopiert und die Imports umgeschrieben
(`pkg/wiki` → `internal/brain/wiki`, `pkg/config` → `internal/config`,
`pkg/check` → `internal/brain/check`).

Run: `go test ./internal/brain/check/okf/`
Expected: FAIL, weil das Paket fehlt.

- [ ] **Step 2: Code umziehen**

`errors.go` und `soft.go` werden mit denselben Umschreibungen kopiert.
`type Context = wiki.BundleContext` bleibt.

Run: `go test ./internal/brain/check/okf/`
Expected: PASS

- [ ] **Step 3: Auf 100 % heben**

Run: `go test -coverprofile=cover.out ./internal/brain/check/okf/`
Run: `go run ./cmd/loomux check gocover --profile cover.out`
Expected: 100 % je Funktion. Wo nicht, fehlt ein Test. Ein Ausschluss nur
mit Grund.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(check): check pages against the Open Knowledge Format"
```

---

### Task 2: `internal/brain/check/house` — Seite und Bündel

**Files:**
- Create: `internal/brain/check/house/page.go`, `bundle.go` und die Tests dazu
- Quelle: `pkg/check/house/{page,bundle}.go` (313 + 459 Zeilen)

- [ ] **Step 1: Tests umziehen, rot sehen.** Das Vorgehen ist wie in Task 1.
      `federation_test.go` wartet auf Task 3.
- [ ] **Step 2: Code umziehen, grün sehen.**
- [ ] **Step 3: Uhr prüfen.** `untouched`, `stale` und `long-planned` hängen am
      Datum. Die Tests müssen mit festem `now` laufen, und es darf kein
      `time.Now()` im Paket stehen. Findet sich eines, geht es als Parameter
      an den Aufrufer. Das ist ein Umbau gegen die Go-Form und kommt in die
      Akte.
- [ ] **Step 4: 100 %** (wie Task 1, Step 3).
- [ ] **Step 5: Commit** — `feat(check): check pages and bundles against the house rules`

---

### Task 3: `internal/brain/check/house` — Föderation

**Files:**
- Create: `internal/brain/check/house/federation.go`, `federation_test.go`
- Quelle: `pkg/check/house/federation.go` (529 / 874 Zeilen)

- [ ] **Step 1: Tests umziehen, rot sehen.**
- [ ] **Step 2: Code umziehen, grün sehen.** `Federation`, `FederationFor`,
      `wrong-direction`, `unlisted-area`, dazu Signpost und Hub über
      `config.HubLayout` (`internal/config/manifest.go:280`).
- [ ] **Step 3: 100 %.**
- [ ] **Step 4: Commit** — `feat(check): check the federation of areas`

---

### Task 4: `internal/brain/check/run` — die drei Breiten

**Files:**
- Create: `internal/brain/check/run/run.go`, `targets.go` und die Tests dazu
- Quelle: `pkg/check/run/{run,targets}.go` (577 + 88 Zeilen), **ohne**
  `CheckCode` und ohne `inner_parallel_test.go` (B5, E2)

- [ ] **Step 1: Tests umziehen, rot sehen.** Die Testfälle für `CheckCode`
      fallen weg und stehen in der Akte als Wegfall mit Verweis auf E2.
      `run_test.go` importiert `internal/testlock` (in loomux vorhanden) und
      setzt `BRAIN_STATE_DIR`, das zu `LOOMUX_STATE_DIR` wird;
      `bundle_test.go` importiert `time/tzdata` (Akte, B5 berichtigt).
- [ ] **Step 2: Code umziehen, grün sehen.** Die Registry wird über
      `config.NewArtifactLookup` gelesen statt über `config.StateDir()`
      (`run.go:538`). So gilt die Leseregel des Zustandsorts.
- [ ] **Step 3: Parallelität prüfen.** `go test -race ./internal/brain/check/run/`
      Expected: PASS. Ein gemeinsames `now` je Lauf (B6).
- [ ] **Step 4: 100 %.**
- [ ] **Step 5: Commit** — `feat(check): check one file, one bundle or every area`

---

### Task 5: `loomux brain check`

**Files:**
- Modify: `internal/cli/brainargs.go`: `check` kommt als Verb dazu, und die
  Reihenfolge der Hilfe bleibt die der Referenz. `check` hängt hinten an,
  weil es in `brain-mcp` nicht vorkommt.
- Create: `internal/cli/braincheck.go`, `braincheck_test.go`
- Modify: `internal/cli/imports_test.go` (B16)

- [ ] **Step 1: Tests schreiben.** Je Exit-Pfad aus B4 ein Test:
  - `file` sauber (0) und `file` mit Fehler-Befund (1)
  - `file` ohne Pfad, `file` auf ein Verzeichnis (2)
  - `bundle` ohne `--scope`, `bundle` mit unbekanntem Scope (2)
  - `all` über zwei Bereiche mit einem `wrong-direction`
  - unbekannte Breite (2), `code` ergibt „unknown width“ (2, E2)
  - `--notes` an beliebiger Stelle

  Dazu ein Test im Importgraphen: `hooks` erreicht `brain/check/okf`, `house`
  und `run` nicht, `cli` erreicht sie.
- [ ] **Step 2: Rot sehen.**
- [ ] **Step 3: Umsetzen.** Die Argumente liest `parseInterspersed`, gedruckt
      wird über `check.Render`/`check.ExitCode`. Die Diagnose geht als
      `error: …` auf stderr, wie in `main.go:921-980`.
- [ ] **Step 4: Grün sehen, 100 %.**
- [ ] **Step 5: Commit** — `feat(cli): check a page, a bundle or every area with brain check`

---

### Task 6: `internal/brain/wiki/types.go` — Ränge der Seitentypen

**Files:**
- Create: `internal/brain/wiki/types.go`, `types_test.go`
- Quelle: `src/brain/wiki/types.py` (68), Tests `tests/wiki/test_types.py` (10 Tests)

- [ ] **Step 1: Die zehn Tests als Go-Tabellentest schreiben, rot sehen.**
- [ ] **Step 2: Umsetzen.** `Rank` (`Core`, `Catalogue`, `Origin`,
      `Declared`, `Unknown`), `RankOf(name string, declared []string) Rank` und
      `Aliases`. Die Menge `core`/`catalogue`/`origin` ist ein Literal.
- [ ] **Step 3: Doppelung prüfen.** `config.knownTypes`
      (`internal/config/manifest.go:47-84`) muss dieselbe Vereinigung sein. Ein
      Test hält beide gleich; `knownTypes` wird **nicht** umgebaut, denn
      `config` darf `wiki` nicht importieren.
- [ ] **Step 4: 100 %.**
- [ ] **Step 5: Commit** — `feat(wiki): rank page types as core, catalogue, origin or declared`

---

### Task 7: `internal/brain/wiki/census.go` — Volkszählung

**Files:**
- Create: `internal/brain/wiki/census.go`, `census_test.go`
- Quelle: `src/brain/wiki/census.py` (135), Tests `tests/wiki/test_census.py` (13)

- [ ] **Step 1: Tests schreiben, rot sehen.** Dazu gehören ein Bereich ohne
      `wiki_path`, ein fehlendes Verzeichnis (beides still übersprungen),
      Gerüstdateien, `(no type)`, der schlechteste Rang gewinnt über Bereiche,
      ein Alias, und die Sortierung nach Summe absteigend, dann nach Name.
- [ ] **Step 2: Umsetzen.** `Census(areas []config.Area, manifests func(config.Area) (config.Manifest, error)) []TypeCount`
      und `RenderCensus`. Die Zeile lautet
      `{marker}{name} [{rank}]{alias_note}: {total} ({breakdown})`
      (`census.py:131-134`). Die Reihenfolge der Seiten muss der von `rglob`
      entsprechen; die Summen hängen nicht daran, die Aufschlüsselung
      schon.
- [ ] **Step 3: 100 %.**
- [ ] **Step 4: Commit** — `feat(wiki): count page types across every area`

---

### Task 8: `internal/brain/wiki/retype.go` — einen Typ umbenennen

**Files:**
- Create: `internal/brain/wiki/retype.go`, `retype_test.go`
- Quelle: `src/brain/wiki/retype.py` (109), Tests `tests/wiki/test_retype.py` (19)

- [ ] **Step 1: Tests schreiben, rot sehen.**
  - Nur ein ungequoteter Skalar wird ersetzt.
  - Übersprungen werden gequotete oder gefaltete Werte, kaputtes
    Frontmatter, nicht dekodierbares UTF-8 und Gerüstdateien (`_schema.md`,
    `index.md` bleiben unberührt).
  - Eine geschriebene Seite wird ganz auf LF gefaltet.
  - Ein zweiter Lauf ändert nichts.
  - Gemeldet wird nur ein Pfad, dessen Bytes sich geändert haben.
- [ ] **Step 2: Umsetzen.** `Retype(wikiRoot, from, to string) ([]string, error)`.
      Vorher wird mit `os.ReadFile` verglichen, geschrieben über
      `lock.ReplaceText` (B10).
- [ ] **Step 3: 100 %.**
- [ ] **Step 4: Commit** — `feat(wiki): rename one page type in one bundle`

---

### Task 9: `loomux wiki init|types|retype`

**Files:**
- Create: `internal/cli/wikicmd.go`, `wikicmd_test.go`
- Modify: `internal/cli/commands.go`: `"wiki": wikiCommand` kommt dazu, und
  `wiki-gate` bleibt, wie es ist.

- [ ] **Step 1: Tests schreiben, rot sehen.** Die Weigerungen sind wörtlich
      aus `cli.py:865-870,884-904` übernommen, als `error: …` auf stderr
      mit Exit 1:
  - `no area named {scope!r} in the registry`
  - `area {scope!r} declares no wiki path; …`
  - `area {scope!r} is read-only; its bundle cannot be renamed`

  Die Warnung bei unbekanntem Zieltyp geht auf stderr, der Lauf geht weiter
  (Exit 0). `{scope!r}` ist Pythons `repr`, also `'scope'` in einfachen
  Anführungszeichen, über `pytext.Repr` (`internal/brain/pytext/repr.go:28`). `wiki types` endet immer mit Exit 0.
  `wiki init` druckt die geschriebenen Pfade.
- [ ] **Step 2: Umsetzen.** Gelesen wird über `config.NewArtifactLookup` und
      `config.ReadRegistry(lookup.Primary)`. Die Manifeste kommen über
      `config.ResolvedAreaDir(area, lookup.Primary, lookup.Fallback)` und
      `config.ReadAreaManifestUntilStage4`, wie in `run` (Task 4): Die
      Bereiche dieses Rechners tragen noch `.brain.toml`, und
      `config.ReadManifest` kennt nur `.loomux/config.toml`. Dasselbe gilt
      für `Census` (Task 7) und für den `BundleContext` des Sweeps (Task 10).
- [ ] **Step 3: Grün sehen, 100 %.**
- [ ] **Step 4: Commit** — `feat(cli): scaffold, count and retype wiki pages with loomux wiki`

---

### Task 10: `loomux lint --scope all|<scope>`

Nach E5 in zwei Teilen: erst die Regeln, dann der Sweep. Die Reichweite von
E5 wurde am 2026-09-23 nachentschieden: `lint.go`, `wiki-gate`, die Lane und
der post-edit-Hook bleiben die Go-Form von 1a (der 1a-Fall `lint/valid-page`
trägt `type: concept`, das `lint.py` ablehnt). Die Regeln von `lint.py`
bekommen einen eigenen Regelsatz mit eigenem Leser.

**Files:**
- Create: `internal/brain/pytext/url.go`: `SplitURL`, `Unquote`,
  `DecodeReplace` nach Python 3.14, gegen `stufe-3c-orakel/url.py` und
  `unquote.py` gemessen
- Create: `internal/brain/wiki/sweeppage.go`: der Leser von `page.py`
- Create: `internal/brain/wiki/sweep.go`: `SweepBundle`, die zwölf Regeln
- Create: `internal/brain/wiki/testdata/sweep/python.golden`: die Ausgabe von
  `stufe-3c-orakel/sweep.py`
- Modify: `internal/cli/wiki.go`, Create: `internal/cli/lintsweep.go`

- [x] **Step 1–3: Regeln gegen das Orakel.** Die Welt von `sweep.py` in Go
      nachgebaut; die Ausgabe gleicht der von `lint_bundle` bis auf den Text
      eines YAML-Fehlers.
- [x] **Step 4: Halt entfällt** (`lint.go` und die Lane unverändert).
- [ ] **Step 5: Commit** — `feat(wiki): check bundles by every rule of the reference lint`
- [ ] **Step 6: Sweep-Tests nach `tests/wiki/test_lint_cli.py`, rot sehen**
      (12 Tests):
  - der Sweep über alle Bereiche und ein Bereich per `--scope S`
  - die drei Weigerungen, jede mit `error: …` auf stderr und **Exit 1**
    (`cli.py:2408-2436`). „Wiki fehlt“ bricht auch den Sweep über alle ab
    (`:1583-1586`)
  - Signpost und geteilte Bereiche
  - Kopfzeile je Bereich, eingerückte Befunde oder `  no findings`
  - die Schlusszeile und die Exits (0; 1 nur bei Fehlern)

  Ohne Pfad und ohne `--scope` gilt `all` (B11). Ein Pfad auf eine Datei oder
  auf `*.md` geht an den 1a-Pfad (E6). Jeder andere Pfad wird ignoriert wie in
  Python; das ist ein eigener Test.
- [ ] **Step 7: Sweep umsetzen, grün sehen, 100 %.** Der 1a-Einzelpfad und
      seine Fälle (`go test ./internal/cli/ -run TestCases1a`) bleiben grün.
- [ ] **Step 8: Commit** — `feat(lint): lint every registered bundle with --scope`

---

### Task 11: `internal/serve/upkeep.go` — der Durchgang

**Files:**
- Create: `internal/serve/upkeep.go`, `upkeep_test.go`

- [ ] **Step 1: Tests mit falscher Uhr schreiben, rot sehen.** Die Nähte
      entsprechen der Referenz: `now func() time.Time`, `interval
      time.Duration` und `run func(now time.Time) (*maintenance.Report, error)`.
      Getestet wird:
  - Fällig ohne Stempel und bei `now - stamp >= 24h`; nicht fällig, dann
    bleibt der Bericht leer (B14).
  - Das Tor: `CaughtUp()` blockiert bis zum Ende des ersten Durchgangs und
    öffnet auch bei einem Fehler.
  - Ein Fehler landet in `Failure()`. Der nächste Takt löscht ihn, auch einer,
    in dem nichts fällig war (`server.py:188` steht hinter beiden Zweigen).
  - `Settled()` wird nach dem ersten Durchgang wahr; nur bis dahin meldet das
    Tor `CATCH_UP_NOTICE` (`server.py:388`).
  - Ohne Registry oder bei leerer Registry gibt es keinen Bericht.
  - `ErrNoReviewCentre` ergibt eine Fehlerzeile.
  - Die Zeilen `Notes`, `Headline` und `Failed` sind wörtlich aus
    `server.py:205-283` übernommen, mit den Namen aus E7, dazu die Variante
    für den Cloud-Kanal und der Sichtbarkeitsfilter.
  - Die Schleife: Zwei Ticks mit einer Uhr, die 24 h springt, ergeben zwei
    Durchgänge. Die Schleife läuft über einen injizierten Ticker, nicht über
    `time.Sleep`.
- [ ] **Step 2: Umsetzen.** `run` ruft `maintenance.Reconcile(areas, lookup, now().UTC())`,
      so wie `catchUp` in `internal/cli/maintenance.go:60-81`. Der
      Durchgang läuft in einer Goroutine, und das Tor ist ein Kanal, der im
      `defer` geschlossen wird. Pro Prozess gibt es einen Upkeep (B14).
- [ ] **Step 3: `go test -race ./internal/serve/`, 100 %.**
- [ ] **Step 4: Commit** — `feat(serve): catch up on the daily reconciliation`

---

### Task 12: Tor und Hinweiszeilen an den Werkzeugen

**Files:**
- Modify: `internal/serve/serve.go` (Upkeep in `Options` und `Run`
  starten), `internal/serve/brain/tools.go` (`handler`, `withFindings`)
- Modify: `internal/cli/imports_test.go`: `serve` erreicht `maintenance`,
  `hooks` weiterhin nicht.

- [ ] **Step 1: Tests schreiben, rot sehen.**
  - Vor jedem Werkzeug wartet das Tor. `CATCH_UP_NOTICE` geht als
    Fortschritt nur, solange der erste Durchgang nicht fertig ist
    (`server.py:43-47,387-400`).
  - `status` bekommt `notes` **ohne** Präfix. `search`, `read`, `catalog` und
    `neighbors` bekommen `headline` mit `! ` davor, alle außer `search`
    zusätzlich die Zeile aus `search.StaleReconcile` (`server.py:445-468`).
  - Die Antwort wird **ein** Textblock: der Text des ersten Blocks, darunter
    die Zeilen, mit `\n` verbunden. Weitere Blöcke fallen weg; ist der erste
    Block kein Text, beginnt die Antwort mit einer Leerzeile
    (`_with_notes`, `server.py:471-485`).
  - Fehlerantworten und Antworten ohne Zeilen bleiben unberührt.
  - Die Graph-Werkzeuge (`internal/serve/graph`) bekommen nichts; das gibt es
    in der Referenz nicht.
- [ ] **Step 2: Umsetzen.** Die `Deps` bekommen ein schmales Interface
      `Upkeep { CaughtUp(ctx) error; Trailer(cmd string, channel privacy.Channel) []string }`,
      damit `serve/brain` `maintenance` nicht selbst importiert.
- [ ] **Step 3: Die 1b-2-Fallsuite bleibt grün.** Ihre Welten stempeln
      `2999-01-01`, also läuft kein Durchgang, und es entsteht keine Zeile.
      Run: `go test ./internal/cli/ -run TestCases1b2 -v`
      Expected: PASS
- [ ] **Step 4: 100 %.**
- [ ] **Step 5: Commit** — `feat(serve): hold the first answer for the catch-up and append its notes`

---

### Task 13: Fallsuite 3c — aufzeichnen, übersetzen, abspielen

**Files:**
- Create: `testdata/cases/3c-worlds/`, `3c-source/`, `3c/` und `3c-map.toml`
- Create: `docs/.superpowers/parity/stufe-3c-orakel/record.sh`, `record_all.sh`
  (nach dem Muster von `stufe-3b-orakel/`)
- Create: `internal/cli/cases_3c_test.go`

- [ ] **Step 1: Halt — der Mensch baut `brain.exe` vom Tag** (`go build -o brain.exe ./cmd/brain`
      in `ultra-brain` am Tag `loomux-3-source`). Die Python-Fälle laufen
      über `brain-mcp` wie in 3b.
- [ ] **Step 2: Welten bauen**, mindestens diese Fälle:

| Befehl | Fälle |
|---|---|
| `brain check file` | sauber · Frontmatter kaputt (Exit 1, `bench/cases/check/file-frontmatter-broken` übernehmen) · Pfad fehlt (2) · Pfad ist ein Verzeichnis (2) |
| `brain check bundle` | sauber · `no-sources` · `long-planned` · Scope unbekannt (2) · ohne `--scope` (2) |
| `brain check all` | zwei Bereiche sauber · `wrong-direction` · `unlisted-area` · Registry fehlt (2) · `--notes` |
| `lint` | Sweep sauber · Sweep mit Fehlern und Warnungen · `--scope S` · Scope unbekannt · kein Wiki-Pfad · Wiki fehlt |
| `wiki init` | neues Bündel · teilweise vorhanden · Scope unbekannt · kein Wiki-Pfad |
| `wiki types` | leer · drei Bereiche mit Alias und unbekanntem Typ |
| `wiki retype` | umbenennen · zweiter Lauf ohne Änderung · nur Lesen (1) · unbekannter Zieltyp (Warnung) · gequoteter Wert bleibt |

- [ ] **Step 3: Aufzeichnen.** `check` läuft über `--exe <brain.exe>`, der
      Rest über `brain-mcp`. Vergleichsklassen:
  - **Daten** mit exaktem stdout: `check`, `lint`, `wiki types`
  - **Meldungen** (Exit und Dateiwelt exakt): `wiki init`, `wiki retype`
- [ ] **Step 4: Übersetzen.** `3c-map.toml`, längere Präfixe zuerst:
  - `brain check` → `loomux brain check`
  - `brain-mcp lint` → `loomux lint`
  - `brain-mcp wiki init` → `loomux wiki init`
  - `brain-mcp types` → `loomux wiki types`
  - `brain-mcp retype` → `loomux wiki retype`
  - `--state-dir …` entfällt (B17)

  Run: `go run ./cmd/loomux dev import-cases --from testdata/cases/3c-source --to testdata/cases/3c --map testdata/cases/3c-map.toml`
- [ ] **Step 5: Abspielen.**
      Run: `go test ./internal/cli/ -run TestCases3c -v`
      Expected: PASS. Ein Unterschied in stdout oder in der Dateiwelt wird
      geheilt. Freigegeben wird er nur, wenn der Mensch ihn mit Grund in die
      Akte einträgt.
- [ ] **Step 6: Commit**, getrennt nach Welten und Aufnahmen
      (`test(cases): record …`) und Suite (`test(cases): hold check, lint and wiki to the recorded reference`).

---

### Task 14: **Halt** — Selbstnutzung

Gegen die echte Registry auf dem Rechner des Menschen:

- [ ] `loomux brain check all`: Befunde neben `brain check all` der Go-Form
      legen und gleich sehen.
- [ ] `loomux lint --scope all`: neben `uv run brain lint --scope all`.
- [ ] `loomux wiki types`: neben `uv run brain types`.
- [ ] `loomux serve` mit einem Stempel älter als 24 h: Der erste Aufruf von
      `status` wartet, und die Zeile erscheint. `last-run.txt` ist danach
      neu.
- [ ] `wiki retype` und `wiki init` nur auf Entscheidung des Menschen. Sie
      schreiben in `brain-knowledge`, dessen Sicherung offen ist (Spec,
      „Selbstnutzung“).

Das Ergebnis kommt mit Datum in die Akte, Abschnitt „Selbstnutzung“.

---

### Task 15: Mutationsrunde, Messungen, Startzeit

- [ ] **Mutanten** über die Entscheidungspakete:

```bash
./bin/loomux.exe dev mutants ./internal/brain/check/house
```

Dasselbe für `./internal/brain/check/okf`, `./internal/brain/check/run`,
`./internal/brain/wiki` (nur `types`, `census`, `retype`, `sweep`) und
`./internal/serve` (nur `upkeep`). Jeder Überlebende bekommt einen Test oder
einen begründeten Eintrag in der Akte unter „Überlebende Mutanten“.

- [ ] **Messungen**, der Median aus 10 warmen Läufen, eingetragen in
      `docs/de/benchmarks.md` und `docs/en/benchmarks.md`:

| Fall | Vergleich |
|---|---|
| `loomux brain check all` über die echte Registry | gegen `brain.exe check all` |
| `loomux lint --scope all` | gegen `uv run brain lint --scope all` |
| `loomux wiki types` | gegen `uv run brain types` |

- [ ] **Startzeit:** `$env:GODEBUG="inittrace=1"; ./bin/loomux.exe --version`
      Expected: kein neues `init` über 500 Allokationen aus `check/*`,
      `wiki` oder `serve`.
- [ ] **Tor:** `go run ./cmd/loomux check precommit`
      Expected: grün, Coverage 100 %.

---

### Task 16: Doku und Migrationsplan

- [ ] `docs/de/migration.md` und `docs/en/migration.md`: Stufe 3c wird ✅ mit
      dem, was sie gebracht hat. Die Abhängigkeitsspalte von **4** wird neu
      gelesen („3 ✅“), und die Funktionszeilen für `check`, `lint --scope`,
      Wiki-Typen und Upkeep werden umgestellt.
- [ ] Fusions-Spec: Stufentabelle und Reihenfolge, Nachtrag #1 und #2
      „gebaut mit 3c“, `check code` als Wegfall (E2).
- [ ] `docs/{de,en}/cli-reference.md`: `brain check`, `lint --scope` und
      `wiki init|types|retype` kommen dazu. Der alte Eintrag `loomux brain
      lint` (`cli-reference.md:403`) wird berichtigt.
- [ ] `README.md` und `README.de.md`: `brain check` und die Wiki-Befehle
      wandern von „Spezifiziert“ zu „Aktiv“, und die Stufenliste bekommt 3c.
- [ ] `go run ./cmd/loomux check precommit` ist grün, samt `lint/wiki`.
- [ ] Commit: `docs: document keeping the brain in shape`
- [ ] Vor dem Push gilt der Skill `release-pr`: Label `release:minor` und ein
      Changelog-Block, geprüft mit `loomux dev release parse-body`.

## Fertig, wenn

1. E1–E8 freigegeben sind, mit Datum in `parity/stufe-3c.md`,
2. alle übersetzten Fälle von 3c grün oder mit Grund freigegeben sind,
   und die Suiten 1a, 1b-1 und 1b-2 grün bleiben,
3. die Coverage 100 % ist und jeder Ausschluss begründet,
4. die Mutationsrunde gelaufen ist und die Überlebenden dokumentiert sind,
5. die drei Messungen eingetragen sind,
6. die Selbstnutzung gelaufen ist oder ihr Fehlen begründet ist,
7. `hooks` nichts aus `check/*` und `maintenance` erreicht (`imports_test.go`).

Danach ist Stufe 3 ganz fertig. Stufe 4 (`init`, `migrate`, Brain-Skills mit
`brain check`) hat dann keine offene Abhängigkeit mehr innerhalb von loomux.
