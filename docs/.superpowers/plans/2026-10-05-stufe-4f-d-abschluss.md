# Stufe 4f, PR D: Abschluss der Migration — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Nach diesem PR nennt der Baum von loomux die Vorgänger und Graft nur noch in Benchmarks, Messchronik, fünf Changelog-Zeilen, dem Lizenzhinweis und je einem Ideensatz; der Migrationsplan und alle Arbeitspapiere der Migration liegen im Archiv-Release.

**Architecture:** Erst kommt der neue Lizenzabschnitt (Code mit Test). Danach werden Namen und Verweise in Code, Doku und Wiki neu gefasst, und die Wiki-Quellen zeigen auf die deutsche Nutzerdoku statt auf die Papiere. Der Mensch lädt den Papier-Tarball hoch, danach verlassen die Papiere den Baum. Als letzter Commit schrumpft die Ausnahmetabelle des Tor-Tests, und `graft` kommt in die Suchliste. Jeder Commit läuft durch das Tor.

**Tech Stack:** Go 1.x (Tests mit `go test`), `git archive`, `gh`, loomux selbst (`reindex`, `lint`, `brain check`, `config … --propose`).

**Spec:** `docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md`, Abschnitt „PR D“ samt Punkt 7. Dazu die Fusions-Spec `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, Nachträge #36, #37 und #38.

## Global Constraints

- `.loomux/config.toml` schreibt kein Agent. Eine Änderung läuft als `loomux config … set … --propose`, angewandt wird sie vom Menschen. Kommentare in dieser Datei ändert der Mensch von Hand.
- Kein Agent pusht. Commits tragen den Nutzer als Autor, kein `Co-Authored-By:` auf ein Modell, keine Werbezeile.
- Commit-Köpfe folgen Conventional Commits und nennen kein Arbeitspapier (kein Plan, keine Stufe, kein Task). Ein Scope nennt einen Bereich des Codes.
- Die Suchliste des Tor-Tests bleibt wörtlich: `(?i)ultraloom|ultra-brain|ulguard|ulinit|ulflow|brain-mcp|brain guard|ultraloomowned|\.brain\.toml|\.ultra-brain|specs-u[lb]/|plans-u[lb]/|bench-ub/`. Task 11 hängt `|(?:^|[^_])graft` an.
- Diese Formel nennt ein archiviertes Papier (#38): „`docs/.superpowers/<pfad>` in the working papers of the archive release `archive/parity-recordings`“ (englisch in Code und englischer Doku) bzw. „`docs/.superpowers/<pfad>` in den Arbeitspapieren des Archiv-Release `archive/parity-recordings`“ (deutsch).
- Die drei Neustart-Papiere bleiben im Baum: `docs/.superpowers/specs/2026-10-05-loomux-release-neustart-design.md` und `docs/.superpowers/plans/2026-10-05-release-bruecke.md` (auf `master`). Das dritte, `plans/2026-10-05-release-neustart.md`, liegt nur auf `feat/release-restart`.
- `docs/{en,de}/benchmarks.md` und `testdata/bench/**` werden nicht angefasst, auch nicht ihre Pfade nach `docs/.superpowers/`.
- 100 % Coverage je Funktion. Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Code, Kommentare und Meldungen sind englisch. Doku unter `docs/de/`, `README.de.md` und `docs/wiki` ist deutsch, Arbeitspapiere sind deutsch.
- Text mit Backslashes nur per Write/Edit, nie per Heredoc, `python -c` oder `sed`. `git show <rev>:<pfad>` in Git Bash nur mit `MSYS_NO_PATHCONV=1`.
- Die Ausgabe von Torläufen und Commits wird nie verworfen. Sie geht in eine Datei im Scratchpad.
- Diese Plandatei ändert nach Task 1 niemand im Baum. Den Fortschritt hakt der Controller in `$S/plan.md` ab; Task 9 Schritt 1 kopiert sie zurück und committet sie allein. Vor jedem Commit zeigt `git status --short` nur die Dateien des Tasks.
- Kein Text, den dieser PR in den Baum bringt oder den der Release-Prozess später hineinschreibt, trägt einen Namen der Suchliste oder `graft`. Das gilt für Commit-Texte, die neuen `log.md`-Einträge und den `## Changelog`-Block des PR-Texts, den das Release nach `CHANGELOG.md` schreibt. Vor `release-pr` läuft das Muster aus Task 11 über den PR-Text.

### Probe-Zustand für `reindex`, `lint` und `brain check`

Die Registry des Nutzers (`%LOCALAPPDATA%\loomux\registry.toml`) zeigt mit `project/loomux` auf den Haupt-Checkout, nicht auf diesen Worktree. Jeder Lauf in diesem Plan nutzt deshalb ein eigenes Zustandsverzeichnis. Einmal anlegen, mit `$S` = Scratchpad und `$W` = Worktree-Pfad mit Vorwärtsschrägstrichen:

~~~bash
go build -o "$S/loomux.exe" ./cmd/loomux
mkdir -p "$S/probe/state" "$S/probe/xdg"
~~~

`$S/probe/state/registry.toml` per Write anlegen (`$W` ausgeschrieben):

~~~toml
[[area]]
scope = "project/loomux"
path = "<W>"
wiki = "<W>/docs/wiki"
workspace = true
~~~

Die drei Läufe, jeder als eigener Aufruf:

~~~bash
LOOMUX_STATE_DIR="$S/probe/state" XDG_CONFIG_HOME="$S/probe/xdg" "$S/loomux.exe" reindex > "$S/probe/reindex.txt" 2>&1
LOOMUX_STATE_DIR="$S/probe/state" XDG_CONFIG_HOME="$S/probe/xdg" "$S/loomux.exe" lint --scope project/loomux > "$S/probe/lint.txt" 2>&1
LOOMUX_STATE_DIR="$S/probe/state" XDG_CONFIG_HOME="$S/probe/xdg" "$S/loomux.exe" brain check bundle --scope project/loomux > "$S/probe/check.txt" 2>&1
~~~

Gemessen am 2026-10-05: `reindex` warnt, dass kein Bereich ein Prüfzentrum erklärt (erwartet), und schreibt nur `_identities.tsv` sowie die ignorierten Kataloge. `lint` und `brain check` melden auf `1364a830` „no findings“. Sie prüfen weder, ob eine Quelle existiert, noch ob ihr Hash stimmt. Die Prüfung in Task 7 Schritt 5 übernimmt das.

## Review Focus

1. **Tarball und Löschmenge weichen ab.** Ein Papier fehlt im Archiv, oder ein Neustart-Papier landet darin. Task 10 Schritt 3 vergleicht die Dateiliste des Tarballs mit den gelöschten Pfaden.
2. **Eine Wiki-Quelle passt nicht zum Register.** `doc_id`, `content_hash` oder `revision` stimmen nicht mit `_identities.tsv` überein, und `lint` sieht das nicht. Task 7 Schritt 5 prüft jede Quelle per Skript.
3. **`graft` trifft das Falsche oder verfehlt etwas.** Das Muster trifft `GIT_GRAFT_FILE` oder verfehlt `Graft's`, `graftEdges` oder `WithGraftsDefaults`. Das testet Task 11 Schritt 1.
4. **Ein umformulierter Kommentar verliert den Quellpfad des Originals.** Ohne `src/…ts` lässt sich der Port nicht mehr gegen das Original halten. Task 3 Schritt 4 zählt die `src/`-Pfade vorher und nachher.
5. **Ein Markdown-Link zeigt auf eine gelöschte Datei.** Gemeint sind `migration.md` und `docs/.superpowers/…` in Nutzerdoku und README. Task 10 Schritt 7 greppt nach solchen Links.

---

### Task 1: Spec-Nachtrag und Plan committen

**Files:**
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` (Zeile #38, schon geschrieben)
- Modify: `docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md` (PR D, Punkt 7, schon geschrieben)
- Create: `docs/.superpowers/plans/2026-10-05-stufe-4f-d-abschluss.md` (dieser Plan)

- [ ] **Step 1: Commit**

~~~bash
git add docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md docs/.superpowers/plans/2026-10-05-stufe-4f-d-abschluss.md
git commit -m "docs: record the rulings for closing the migration" > "$S/commit-t1.txt" 2>&1
~~~

---

### Task 2: `dev notices` schreibt einen Abschnitt für portierte Quellen

**Files:**
- Create: `internal/dev/notices/ported.md`
- Modify: `internal/dev/notices/notices.go` (Paketdoku, eingebettete Datei, `Render`)
- Test: `internal/dev/notices/notices_test.go`
- Regenerate: `internal/notices/NOTICE.md`

**Interfaces:**
- Produces: In `NOTICE.md` steht der Abschnitt `## Ported sources` zwischen den Grammatiken und dem Zipf-Hinweis. Task 11 nimmt `internal/dev/notices/ported.md` und `internal/notices/NOTICE.md` als Ausnahmen.

- [ ] **Step 1: Write the failing test**

In `TestRenderNamesEveryLinkedPieceOnce` kommen in die `want`-Liste:

~~~go
		"## Ported sources\n", "1e352a3fbee9ed9a5964ac35332ef0e5c63de49c",
		"Copyright (c) 2026 Context Graph Engine contributors",
~~~

Die Bedingung darunter bekommt zusätzlich `|| strings.Count(text, "## Ported sources") != 1`.

In `TestRenderOrdersTheSections` kommt zwischen `"## tree-sitter grammar python\n",` und `"# Third-party notice: German word frequencies",` die Zeile:

~~~go
		"## Ported sources\n",
~~~

Der Kommentar über `TestRenderOrdersTheSections` lautet dann: `// The notice reads the same on every run: Go first, the modules and then the grammars each by name, the ported sources, the word frequencies last.`

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/dev/notices -run 'TestRender(NamesEveryLinkedPieceOnce|OrdersTheSections)' -count=1 > "$S/t2-red.txt" 2>&1`
Expected: FAIL, `missing "## Ported sources\n"` bzw. `"## Ported sources\n" at -1`.

- [ ] **Step 3: Write minimal implementation**

`internal/dev/notices/ported.md` per Write anlegen. Der Lizenztext ist wörtlich der von `trailhq/Graft` am Commit `1e352a3`. Er wurde am 2026-10-05 per `gh api repos/trailhq/Graft/contents/LICENSE?ref=1e352a3` gelesen. Vor dem Schreiben erneut lesen und vergleichen:

~~~~markdown
## Ported sources

Parts of `internal/code` (extraction, resolution, PageRank, freshness, grep, repo map and blast radius) are ported from trailhq/Graft <https://github.com/trailhq/Graft> at commit 1e352a3fbee9ed9a5964ac35332ef0e5c63de49c, under the MIT license.

### LICENSE

```
MIT License

Copyright (c) 2026 Context Graph Engine contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
~~~~

In `notices.go` kommt `_ "embed"` in den Importblock. Unter `licenseFile` kommt:

~~~go
// portedNotice is the notice of code loomux ported from another project
// rather than linking it as a module; no go list can find it.
//
//go:embed ported.md
var portedNotice string
~~~

In `Render` steht vor `b.WriteString("\n" + model.ZipfNotice())`:

~~~go
	b.WriteString("\n" + clean([]byte(portedNotice)) + "\n")
~~~

Die Paketdoku (Zeilen 1–5) lautet dann: `// Package notices writes NOTICE.md: the license of every third-party piece the loomux binary links -- Go's standard library, each module, each tree-sitter grammar whose package is imported --, of the code ported from other projects, and the notice of the embedded word frequencies. A test holds the committed file to what this renders, so a new dependency cannot ship without its notice.`

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/dev/notices -count=1 > "$S/t2-green.txt" 2>&1`
Expected: PASS.

- [ ] **Step 5: Regenerate NOTICE.md and run the notice test**

~~~bash
go run ./cmd/loomux dev notices > "$S/t2-notices.txt" 2>&1
go test ./internal/notices ./internal/release ./internal/cli -run 'Notice|Build|DevNotices' -count=1 > "$S/t2-notice-test.txt" 2>&1
~~~

Expected: `internal/notices/NOTICE.md` enthält genau einmal `## Ported sources`, die Tests sind grün.

- [ ] **Step 6: Mutation**

Zwei Overlays prüfen, dass ein Test die Änderung trägt. Mutant 1 entfernt die neue `WriteString`-Zeile. Mutant 2 schiebt sie hinter den Zipf-Hinweis. Beide müssen rot werden: Mutant 1 in beiden Tests, Mutant 2 in `TestRenderOrdersTheSections`. Die Overlay-JSON wird per Write geschrieben, mit Pfaden in der Form `C:/…`. Die Ausgabe geht nach `$S/t2-mut-<n>.txt`.

- [ ] **Step 7: Commit**

~~~bash
git add internal/dev/notices/ported.md internal/dev/notices/notices.go internal/dev/notices/notices_test.go internal/notices/NOTICE.md
git commit -m "feat(dev): quote the license of ported code in NOTICE.md" > "$S/commit-t2.txt" 2>&1
~~~

---

### Task 3: Der Name des portierten Originals verlässt den Code

**Files:** jede Datei aus `git grep -l -i -E "(^|[^_])graft" -- internal ':(exclude)internal/dev/notices' ':(exclude)internal/notices' ':(exclude)internal/plancheck'`. Gezählt am 2026-10-05 sind es 32 Dateien unter `internal/code/**`, dazu `internal/cli/graph_test.go` und `internal/serve/graph/tools_test.go`. `internal/gitenv/gitenv.go` (`GIT_GRAFT_FILE`) bleibt unverändert.

**Interfaces:**
- Produces: Unter `internal/` außerhalb der Notices nennt kein Code mehr `graft`, außer `GIT_GRAFT_FILE`. Task 11 verlässt sich darauf.

- [ ] **Step 1: Count the source paths before**

Run: `git grep -h -o -E "src/[a-z/._-]+\.ts" -- internal/code internal/cli internal/serve | sort | uniq -c > "$S/t3-src-before.txt"`

- [ ] **Step 2: Rewrite by these rules**

Die Regeln gelten der Reihe nach, Treffer für Treffer:

| Vorher | Nachher |
|---|---|
| `// Ported from trailhq/Graft @ 1e352a3 (MIT), <src>.` (auch mit Zusatz nach `<src>`) | `// Ported from <src> (MIT; origin under "Ported sources" in NOTICE.md).`, Zusatz bleibt |
| `… are ported from trailhq/Graft @ 1e352a3` (Testvektoren) | `… are ported (see "Ported sources" in NOTICE.md)` |
| `from trailhq/Graft @ 1e352a3` in anderen Sätzen | `from the original (see "Ported sources" in NOTICE.md)` |
| `Graft's <x>` | `the original's <x>` |
| `Graft` als Subjekt oder Objekt | `the original` |
| `Reference: trailhq/Graft measures …` | `Reference: the original measures …` |
| `graftEdges` (`internal/code/pagerank/dangling_bench_test.go`) | `tripleEdges` |
| `TestFileBindsALocalVariableOnlyInTheFourFormsGraftKnows` | `TestFileBindsALocalVariableOnlyInTheFourFormsTheOriginalKnows` |
| `TestFindCodeReachesTheAreaRootWithGraftsDefaults` | `TestFindCodeReachesTheAreaRootWithTheOriginalsDefaults` |
| `want Graft's 5000` (`extract_test.go`) | `want the original's 5000` |

Die Aussage jedes Satzes bleibt gleich. Satzbau und Umbruch werden so angepasst, dass der Kommentar weiter in seine Spalte passt (`gofmt` ändert keine Kommentare). Dateinamen und Zeilenangaben des Originals (`resolve.ts:229-237`) bleiben stehen.

- [ ] **Step 3: Verify nothing is left**

Run: `git grep -n -i -E "(^|[^_])graft" -- internal ':(exclude)internal/dev/notices' ':(exclude)internal/notices' ':(exclude)internal/plancheck'`
Expected: keine Ausgabe.

- [ ] **Step 4: Count the source paths after**

Run: `git grep -h -o -E "src/[a-z/._-]+\.ts" -- internal/code internal/cli internal/serve | sort | uniq -c > "$S/t3-src-after.txt"` und danach `diff "$S/t3-src-before.txt" "$S/t3-src-after.txt"`.
Expected: keine Ausgabe.

- [ ] **Step 5: Build, format and test**

Run: `gofmt -l internal && go vet ./internal/... && go test ./internal/code/... ./internal/cli ./internal/serve/... -count=1 > "$S/t3-test.txt" 2>&1`
Expected: `gofmt -l` gibt nichts aus, alle Tests sind grün.

- [ ] **Step 6: Commit**

~~~bash
git add -A internal
git commit -m "refactor(code): name the ported original only in NOTICE.md" > "$S/commit-t3.txt" 2>&1
~~~

---

### Task 4: Der Name verlässt die Nutzerdoku und das Wiki, außer im Ideensatz

**Files:**
- Modify: `README.md` (Tabelle „Architectural Decisions“ Z. 345–352, Doku-Tabelle Z. 372, Z. 112)
- Modify: `README.de.md` (Z. 352–359, Z. 379)
- Modify: `docs/en/architecture.md` (Z. 122, 124, 203), `docs/de/architecture.md` (Z. 122, 124, 207)
- Modify: `docs/en/README.md:12`, `docs/de/README.md:12`
- Modify: `docs/en/cli-reference.md:967`, `docs/de/cli-reference.md:999`
- Modify: `docs/en/getting-started.md:290`, `docs/de/getting-started.md:293`
- Modify: `docs/wiki/topics/code-graph.md` (Z. 23–31, 118)

**Interfaces:**
- Produces: genau vier Ideenzeilen, die `(?i)(inspired by|angeregt von) \[?trailhq/Graft` treffen, je eine in `README.md`, `README.de.md`, `docs/en/architecture.md` und `docs/de/architecture.md`. Task 11 nimmt sie als Zeilenausnahme.

- [ ] **Step 1: The four idea sentences**

Jeder Ideensatz steht auf **einer** physischen Zeile, ohne Umbruch, mit Linktext und URL auf derselben Zeile. Der Tor-Test prüft zeilenweise. Eine Folgezeile mit `Graft` ohne `inspired by`/`angeregt von` wäre in Task 11 rot.

- `README.md`, direkt unter der Tabelle „Architectural Decisions“: `The code graph, its ranking and the blast radius are inspired by [trailhq/Graft](https://github.com/trailhq/Graft); the ported parts and their MIT license are listed in [NOTICE.md](internal/notices/NOTICE.md).`
- `README.de.md`, an derselben Stelle: `Code-Graph, Ranking und Blast-Radius sind angeregt von [trailhq/Graft](https://github.com/trailhq/Graft); die portierten Teile und ihre MIT-Lizenz stehen in [NOTICE.md](internal/notices/NOTICE.md).`
- `docs/en/architecture.md:124`: `Text embeddings alone cannot determine whether modifying `function A` breaks `function B`. Loomux applies graph engineering principles inspired by [trailhq/Graft](https://github.com/trailhq/Graft):`
- `docs/de/architecture.md:124`: Den deutschen Satz bis zum Doppelpunkt so fassen, dass er `angeregt von [trailhq/Graft](https://github.com/trailhq/Graft):` enthält.

- [ ] **Step 2: Everything else loses the name**

- Die Spalte „Origin / Inspiration“ der README-Tabellen heißt in den Graft-Zeilen `Code-graph model` (de: `Code-Graph-Vorbild`).
- Die Zellen formulieren ohne Namen: „Its crux is an excerpt an LLM chose …“, „Requires Node.js >=20 …“, „Syncs symbol hashes …“, „Sends usage statistics …“ (de entsprechend).
- Die Überschrift `## 4. … (Graft)` verliert die Klammer.
- `Graft AST GraphRank` / `Grafts AST-GraphRank` wird `AST GraphRank` (Doku-Tabellen und `getting-started.md`).
- `README.md:112` verlinkt den Anker der umbenannten Überschrift (`#4-conceptual-pillar-iii-structural-graph-intelligence`). `README.de.md` hat keinen solchen Link (gesucht am 2026-10-05).
- `cli-reference.md`: Der Halbsatz „both Graft's values“ / „beides Grafts Werte“ entfällt.
- `architecture.md:203/207`: Ein Graft-Satz wird ohne Namen gefasst.
- Wiki `code-graph.md`: „Vorbild ist `trailhq/Graft`.“ wird „Das Vorbild steht in `NOTICE.md` unter „Ported sources“.“. Die Tabellenspalte „Graft“ heißt „Vorbild“, `Grafts Kernlogik-Exzerpt` wird `das Kernlogik-Exzerpt des Vorbilds`, „Kein `graft brain`-Pendant“ wird „Kein Gegenstück zum Cloud-Brain des Vorbilds“.

- [ ] **Step 3: Verify**

Run: `git grep -n -i -E "(^|[^_])graft" -- . ':(exclude)docs/.superpowers' ':(exclude)docs/en/benchmarks.md' ':(exclude)docs/de/benchmarks.md' ':(exclude)internal/notices' ':(exclude)internal/dev/notices' ':(exclude)internal/plancheck'`
Expected: genau die vier Ideenzeilen aus Step 1.

Run: `go test ./internal/plancheck -count=1 > "$S/t4-test.txt" 2>&1`
Expected: PASS.

Kein Test prüft Anker in der Doku (gesucht am 2026-10-05). Deshalb von Hand: `git grep -n "conceptual-pillar-iii\|saeule-iii\|säule-iii" -- '*.md' ':(exclude)docs/.superpowers'`. Jeder Treffer muss genau den Anker nennen, den GitHub aus der neuen Überschrift bildet: Kleinbuchstaben, Leerzeichen zu `-`, Satzzeichen außer `-` entfernt.

- [ ] **Step 4: Commit**

~~~bash
git add README.md README.de.md docs/en docs/de docs/wiki/topics/code-graph.md
git commit -m "docs: keep the code graph's model as an idea and a notice" > "$S/commit-t4.txt" 2>&1
~~~

---

### Task 5: Der Migrationsplan und seine Prüfung gehen; der Tor-Test zieht nach `internal/namecheck`

**Files:**
- Delete: `docs/en/migration.md`, `docs/de/migration.md`, `internal/plancheck/plancheck.go`, `internal/plancheck/plancheck_test.go`
- Move: `internal/plancheck/references.go` → `internal/namecheck/references.go`, `internal/plancheck/references_test.go` → `internal/namecheck/references_test.go`
- Modify: `AGENTS.md` (Regel zum Migrationsplan Z. 103–113, Roadmap-Regel Z. 114–121)
- Modify: `README.md`, `README.de.md` (Bildunterschrift Z. 47–49, Abschnitt „Migration Plan“ Z. 140–148, Roadmap-Einleitung Z. 152–156, Web-Zeile Z. 169, CLI-Satz Z. 201, Doku-Zeile Z. 377; de entsprechend)
- Modify: `docs/en/README.md:17`, `docs/de/README.md:17` (Zeile löschen)
- Modify: `docs/en/architecture.md:152-153,186`, `docs/de/architecture.md:152,190`
- Modify: `docs/en/cli-reference.md:1362`, `docs/de/cli-reference.md:1419`

**Interfaces:**
- Produces: Das Paket `namecheck` mit unveränderter API: `References`, `Exception`, `exceptions()`, `predecessorNames()`. Task 11 ändert es.

- [ ] **Step 1: Move and delete**

~~~bash
git rm -q docs/en/migration.md docs/de/migration.md internal/plancheck/plancheck.go internal/plancheck/plancheck_test.go
mkdir internal/namecheck
git mv internal/plancheck/references.go internal/namecheck/references.go
git mv internal/plancheck/references_test.go internal/namecheck/references_test.go
~~~

- [ ] **Step 2: Package name and self exceptions**

In beiden Dateien wird `package plancheck` zu `package namecheck`. Über `package namecheck` in `references.go` kommt:

~~~go
// Package namecheck holds the repository to one promise: the names of the
// tools loomux replaced stand only where an exception lets them.
~~~

In `exceptions()` fallen die Zeilen mit `Owner: "plan-end"` (die beiden `docs/*/migration.md`). Die beiden `Owner: "self"`-Zeilen zeigen jetzt auf `internal/namecheck/references.go` und `internal/namecheck/references_test.go`.

- [ ] **Step 3: AGENTS.md**

Der Absatz „- The migration plan is …“ entfällt ganz. Die Roadmap-Regel lautet:

~~~markdown
- The roadmap is the section `Roadmap` in `README.md` and `README.de.md`: open
  work (follow-up projects, the open code-graph stages, optional features),
  split into "Coming" and "Maybe", each row with its stage, dependencies and
  priority where it has them. Every pull request that starts, finishes, adds
  or drops such work updates both in the same pull request; a finished row
  leaves the roadmap and its capability goes into the documentation it now
  belongs to. The roadmap sets the priorities itself; changing one is the
  user's decision, made in the pull request that enters it.
~~~

- [ ] **Step 4: READMEs and docs**

- Die Bildunterschrift (Z. 47–49) lautet: `*A dashed line is specified and not built; what comes next is on the [roadmap](#roadmap).*` (de entsprechend).
- Der Abschnitt `## Migration Plan` / `## Migrationsplan` samt Trennlinie entfällt.
- Die Roadmap-Einleitung lautet: `What loomux will gain next. *Priority* orders the rows (1 first); the roadmap sets it, and changing it is the user's decision.` (de entsprechend). Der Link auf die Fusions-Spec und der Satz zu 4f fallen.
- In der Web-Zeile wird `(components from `ultra-brain/web`)` zu `(components of the predecessor's web app, kept in the archive release `archive/parity-recordings`)`, de: `(Komponenten der Web-App des Vorgängers, im Archiv-Release `archive/parity-recordings`)`.
- Der CLI-Satz (Z. 201) endet: `… and what is not built yet is on the [roadmap](#roadmap).`
- Die Doku-Tabellenzeile „Migration Plan“ fällt in beiden READMEs und in `docs/{en,de}/README.md`.
- `architecture.md` en Z. 152–153: `> … What is still open (stages G4c and G5b to G5d) is on the [roadmap](../../README.md#roadmap); every command is in the [CLI reference](cli-reference.md).` Z. 186: `All of it is built since Stage G4b:`. Deutsch entsprechend.
- `cli-reference.md`: Den Klammerverweis auf den Migrationsplan streichen und den Satz grammatisch schließen. Vorher den ganzen Absatz lesen.

- [ ] **Step 5: Verify and test**

Run: `git grep -n -e "migration\.md" -e plancheck -- . ':(exclude)docs/.superpowers' ':(exclude)docs/wiki'`
Expected: keine Ausgabe.

Run: `go vet ./internal/namecheck && go test ./internal/namecheck -count=1 > "$S/t5-test.txt" 2>&1`
Expected: PASS. `TestTheRepositoryNamesNoPredecessor` ist grün, weil die Web-Zeile den Namen nicht mehr trägt.

- [ ] **Step 6: Commit**

~~~bash
git add -A AGENTS.md README.md README.de.md docs/en docs/de internal/namecheck
git commit -m "chore: retire the migration plan and its check" > "$S/commit-t5.txt" 2>&1
~~~

---

### Task 6: Das Register kennt die deutsche Nutzerdoku (mit Menschenschritt)

**Files:**
- Modify (Mensch): `.loomux/config.toml` (`[index] include` und der Kommentar darüber)
- Modify: `.gitignore` (eine Zeile)
- Regenerate: `_identities.tsv`

**Interfaces:**
- Produces: `_identities.tsv` führt jede Datei aus `git ls-files 'docs/de/*.md'` mit `doc_id`, `content_hash` und `revision`. Task 7 liest diese Zeilen.

- [ ] **Step 1: Propose the include (agent)**

Als eigener Aufruf, ohne Kette:

~~~bash
go run ./cmd/loomux config --root . set index.include 'docs/wiki/**/*.md, docs/.superpowers/**/*.md, docs/de/*.md' --propose
~~~

Gemessen am 2026-10-05: Der Wächter (`hook pre-tool-use --host claude`) lässt genau diese Zeile durch. Dieselbe Zeile mit `--yes` verweigert er. Dann `go run ./cmd/loomux config --root . proposals`. Expected: ein Vorschlag, dessen Diff nur die `include`-Zeile ändert.

- [ ] **Step 2: Human applies and edits the comment**

Dem Menschen nennen:

~~~bash
go run ./cmd/loomux config --root "C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/admiring-zhukovsky-faf0b9" apply <id>
~~~

Den Kommentar über `[index]` ersetzt er von Hand durch:

~~~toml
# The wiki, the working papers and the top level of the German docs, which
# the wiki pages cite; testdata/cases holds loomux's own expectations, and a
# catalog written into them would change them.
~~~

Weiter geht es erst, wenn `git diff .loomux/config.toml` genau diese beiden Änderungen zeigt.

- [ ] **Step 3: Ignore the new catalog**

`reindex` schreibt einen Katalog `docs/de/index.md`, wie es `docs/index.md` schon tut. In `.gitignore` kommt unter `/docs/index.md` die Zeile `/docs/de/index.md`.

- [ ] **Step 4: Reindex**

Den Probe-Zustand anlegen (siehe Global Constraints) und dann `reindex` laufen lassen.
Expected: Exit 0. `cut -f2 _identities.tsv | grep -c '^docs/de/'` ist gleich `git ls-files 'docs/de/*.md' | grep -c -v '^docs/de/.*/'`. `git status --short` zeigt nur `_identities.tsv`, `.gitignore` und `.loomux/config.toml`.

- [ ] **Step 5: Commit**

~~~bash
git add .loomux/config.toml .gitignore _identities.tsv
git commit -m "chore(config): register the German docs for the wiki to cite" > "$S/commit-t6.txt" 2>&1
~~~

Verweigert der Wächter das `git add` von `.loomux/config.toml`, nennt der Agent dem Menschen genau diese beiden Befehle und wartet.

---

### Task 7: Das Wiki zitiert die Nutzerdoku und nennt die Vorgänger nicht mehr

**Files:**
- Modify: die 15 Inhaltsseiten mit `sources:` unter `docs/wiki/{entities,syntheses,topics}/`
- Modify: `docs/wiki/_schema.md` (Z. 16–20), `docs/wiki/log.md`

**Interfaces:**
- Consumes: Zeilen `docs/de/*.md` und `docs/wiki/**` aus `_identities.tsv` (Task 6)
- Produces: Keine Wiki-Datei nennt mehr einen Namen der Suchliste oder einen Pfad in die Archive. Task 11 nimmt `docs/wiki/log.md` und die Archivregel aus der Tabelle.

- [ ] **Step 1: New sources per page**

Vorschlag für die neuen Quellen, je Seite:

| Seite | neue Quellen |
|---|---|
| `entities/brain-daemon.md` | `docs/de/architecture.md` |
| `entities/okf.md` | `docs/de/architecture.md` |
| `entities/qmd.md` | `docs/de/cli-reference.md` |
| `syntheses/gegenpruefung-vor-jeder-designempfehlung.md` | `docs/wiki/topics/architektur-grundsaetze.md` |
| `syntheses/warum-fast-die-vorgabe-bleibt.md` | `docs/de/benchmarks.md`, `docs/wiki/topics/suche-und-profile.md` |
| `topics/abnahmen-und-echte-umgebung.md` | `docs/de/benchmarks.md` |
| `topics/architektur-grundsaetze.md` | `docs/de/architecture.md` |
| `topics/brain-maintenance.md` | `docs/de/cli-reference.md` |
| `topics/code-graph.md` | `docs/de/architecture.md`, `docs/de/cli-reference.md` |
| `topics/datenmodell-und-bereiche.md` | `docs/de/configuration.md` |
| `topics/datenschutz-und-kanaele.md` | `docs/de/configuration.md` |
| `topics/scheiben-und-abnahme.md` | `docs/de/architecture.md` |
| `topics/schreibschranke.md` | `docs/de/configuration.md`, `docs/de/hooks.md` |
| `topics/suche-und-profile.md` | `docs/de/cli-reference.md`, `docs/de/benchmarks.md` |
| `topics/wiki-schicht.md` | `docs/de/architecture.md`, `docs/de/cli-reference.md` |

Jede Seite wird gegen ihre neue Quelle gelesen. Behandelt die vorgeschlagene Datei das Thema der Seite nicht, wählt der Implementierer eine andere aus `docs/de/*.md` und nennt den Tausch im Bericht. Widerspricht eine Aussage der Seite ihrer neuen Quelle, gilt die Wiki-Regel: beides festhalten, einen Konfliktkasten setzen, `open_conflicts` hochzählen und dem Controller melden. Nichts wird still umgeschrieben.

- [ ] **Step 2: Write the source entries**

Jeder alte Eintrag fällt. Jeder neue hat diese Form, mit `doc_id`, `content_hash` und `revision` wörtlich aus der Zeile der Datei in `_identities.tsv`:

~~~yaml
  - id: <kurzer-name>
    resource: brain://project/loomux/docs/de/architecture.md
    doc_id: <doc_id aus _identities.tsv>
    content_hash: "sha256:<aus _identities.tsv>"
    revision: <aus _identities.tsv>
~~~

- [ ] **Step 3: Prose**

- Prosa, die einen Pfad nach `docs/.superpowers/` zitiert, nennt ihn mit der deutschen Formel aus den Global Constraints oder verliert den Pfad, wo er nur Herkunft war. Das betrifft `brain-maintenance.md:130`, `wiki-schicht.md:157`, `code-graph.md:126,139`, `scheiben-und-abnahme.md:48,101` und `warum-fast-die-vorgabe-bleibt.md:41`.
- „den Stand führt `docs/de/migration.md`“ wird „offene Stufen führt die Roadmap im README“.
- `_schema.md` Z. 16–20: Die Rohquellen sind die Nutzerdoku unter `docs/de/` (oberste Ebene) und, bei Synthesen, Wiki-Seiten. Beide stehen im Identitätsregister, also zitiert jede Seite sie direkt.

- [ ] **Step 4: log.md**

- Die Zeilen 47, 67, 93, 97 und 107 werden ohne Altnamen gefasst: „das Altprojekt“, „das Register des Vorgängers“, „das Bündel des Vorgängers“. Pfade in die Archive bekommen die Formel.
- Zwei neue Einträge ganz oben, datiert 2026-10-05:
  - (1) `topics/code-graph.md` nennt das Vorbild nur noch über `NOTICE.md` (die Änderung aus Task 4). Quelle: `internal/notices/NOTICE.md`.
  - (2) Die Quellen aller Inhaltsseiten zeigen auf `docs/de/` oder Wiki-Seiten, weil die Arbeitspapiere ins Archiv-Release gehen; die Vorgänger werden nicht mehr genannt. Quelle: Code von loomux (`.loomux/config.toml`, `[index]`).

- [ ] **Step 5: Check every source against the register**

Ein Prüfskript per Write nach `$S/t7-sources.py` (PEP 723, ohne Abhängigkeiten), aufgerufen mit `uv run --script "$S/t7-sources.py" > "$S/t7-sources.txt" 2>&1`. Es liest `_identities.tsv` (Spalten `doc_id`, `pfad`, `content_hash`, `revision`) und jede Datei unter `docs/wiki/**/*.md`. Je `sources:`-Eintrag prüft es:

1. Die `resource` ist `brain://project/loomux/<pfad>`, und `<pfad>` steht im Register.
2. `doc_id`, `content_hash` und `revision` sind gleich dem Registereintrag.

Es druckt jede Abweichung und am Ende `<n> sources, <m> mismatches`.
Expected: `m = 0` und `n ≥ 15`.

Gegenprobe vor dem echten Lauf: In einer Kopie einer Seite im Scratchpad wird eine `revision` verfälscht. Das Skript muss dafür genau eine Abweichung melden.

- [ ] **Step 6: Lint and check**

Reihenfolge, weil eine Synthese eine Wiki-Seite zitiert, deren Hash sich durch diesen Task ändert:
1. Alle Seiten außer den beiden Synthesen fertig schreiben.
2. `reindex`.
3. Die Synthese-Quellen auf Wiki-Seiten mit den neuen Registerwerten füllen. Die Quellen der Synthesen auf `docs/de/` stehen schon.
4. Erneut `reindex`.
5. `lint` und `brain check` im Probe-Zustand.
6. Schritt 5 erneut.

Eine Synthese zitiert keine andere Synthese, darum endet die Kette hier.
Expected: `lint` meldet `no findings`, `brain check` gibt Exit 0, das Skript meldet `0 mismatches`.

Run: `git grep -n -i -E "ultraloom|ultra-brain|ulguard|ulinit|ulflow|brain-mcp|brain guard|\.brain\.toml|\.ultra-brain|specs-u[lb]/|plans-u[lb]/|bench-ub/|(^|[^_])graft" -- docs/wiki`
Expected: keine Ausgabe.

- [ ] **Step 7: Commit**

~~~bash
git add docs/wiki _identities.tsv
git commit -m "docs(wiki): cite the German docs instead of the working papers" > "$S/commit-t7.txt" 2>&1
~~~

---

### Task 8: Verweise auf archivierte Papiere zeigen ins Archiv

**Files:**
- Modify: `internal/brain/search/mutation_test.go:420`, `internal/cli/cases_{2a,2b,2c,3a,3b,3c,4a2}_test.go` (je ein Kommentar), `internal/config/declaration.go:25`, `internal/hosts/codex.go:32`, `internal/hosts/hostio.go:153`, `internal/hosts/hostio_test.go:32`
- Modify: `AGENTS.md:4`
- Modify: `README.md` (Abschnitt „Specifications & Internal Working Papers“, Z. 383–389), `README.de.md` (Gegenstück)
- Modify: `internal/brain/apply/testdata/frontmatter/w-*.{in,out}.md` (12 Dateien)

- [ ] **Step 1: Code comments**

Jede Nennung `docs/.superpowers/<pfad>` in den genannten Go-Dateien wird zu „`docs/.superpowers/<pfad>` in the working papers of the archive release `archive/parity-recordings`“. Zeilenangaben bleiben stehen, der Umbruch wird angepasst. Beispiel `internal/hosts/codex.go:32`:

~~~go
// (docs/.superpowers/specs/2026-09-10-go-hooks-drei-hosts-design.md:104-108
// in the working papers of the archive release archive/parity-recordings).
~~~

- [ ] **Step 2: AGENTS.md and READMEs**

`AGENTS.md:4`: `Design: the fusion design among the working papers of the archive release `archive/parity-recordings`.`

Den Abschnitt „Specifications & Internal Working Papers“ ersetzen durch:

~~~markdown
## Specifications & Internal Working Papers

The design papers, plans and parity records of the fusion are in
`working-papers.tar.gz` of the archive release
[`archive/parity-recordings`](https://github.com/xidus90/loomux/releases/tag/archive/parity-recordings).
New working papers live under `docs/.superpowers/`.
~~~

Deutsch entsprechend in `README.de.md`.

- [ ] **Step 3: Frontmatter fixtures**

In den zwölf Dateien wird jeder Pfad unter `docs/.superpowers/` durch einen neutralen Pfad ersetzt, in `.in` und `.out` gleich. Es gibt nur zwei Ersetzungen:
- `docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md` → `docs/de/architecture.md`
- jeder andere Pfad dort → `docs/de/configuration.md`

Erst `git grep -n "docs/.superpowers" -- internal/brain/apply/testdata` lesen, damit kein Pfad übersehen wird.

Run: `go test ./internal/brain/apply -count=1 > "$S/t8-apply.txt" 2>&1`
Expected: PASS. Fällt ein Fall wegen Umbruchs oder Länge (`w-long-description`), wird die erwartete `.out`-Datei nicht von Hand angepasst. Der Fall geht an den Controller.

- [ ] **Step 4: Verify**

Run: `git grep -n "docs/.superpowers/" -- . ':(exclude)docs/.superpowers' ':(exclude)_identities.tsv'`
Expected: Jede verbleibende Zeile ist eine der folgenden:
- Die Formel steht im selben Satz.
- Die Zeile liegt in `docs/{en,de}/benchmarks.md`.
- Es ist ein Muster wie `docs/.superpowers/**` in `getting-started.md`, `.gitignore`, `.loomux/config.toml`, `internal/setup/**` oder `docs/wiki/log.md` (Chronik mit Formel).
- Es ist eine Regel in `internal/plancheck`, jetzt `internal/namecheck`, die Task 11 entfernt.
- Es ist `AGENTS.md:9`/`:37` (Ablage künftiger Papiere).

Run: `go vet ./internal/... && go test ./internal/cli ./internal/hosts ./internal/config ./internal/brain/... -count=1 > "$S/t8-test.txt" 2>&1`
Expected: PASS.

- [ ] **Step 5: Commit**

~~~bash
git add -A AGENTS.md README.md README.de.md internal
git commit -m "docs: point at the working papers in the archive release" > "$S/commit-t8.txt" 2>&1
~~~

---

### Task 9: Papier-Tarball bauen, hochladen lassen, prüfen (mit Menschenschritt)

**Files:** keine im Baum. Ergebnis liegt in `$S/archive/`.

- [ ] **Step 1: Last state of the papers**

Dieser Plan wird mit allen abgehakten Schritten bis hier committet, falls er sich geändert hat (`docs: …`). Danach `git status --short` → leer.

- [ ] **Step 2: Build**

~~~bash
mkdir -p "$S/archive"
git archive --format=tar HEAD -- docs/.superpowers ':(exclude)docs/.superpowers/specs/2026-10-05-loomux-release-neustart-design.md' ':(exclude)docs/.superpowers/plans/2026-10-05-release-bruecke.md' | gzip -n -9 > "$S/archive/working-papers.tar.gz"
~~~

Den Build zweimal fahren, die beiden `sha256sum` müssen gleich sein.

`tar` aus dem Scratchpad heraus aufrufen, mit relativem Pfad, weil Git-Bash-`tar` `C:` als Host liest. Erwartet:
- `tar -tzf working-papers.tar.gz | grep -v '/$' | wc -l` ist gleich `git ls-files docs/.superpowers | wc -l` minus 2.
- Kein Neustart-Papier ist darin.

- [ ] **Step 3: SHA256SUMS**

~~~bash
gh release download archive/parity-recordings -p SHA256SUMS -D "$S/archive"
~~~

Eine Zeile anhängen, in derselben Form wie die übrigen: `<hash> *working-papers.tar.gz`, mit `<hash>` aus `sha256sum "$S/archive/working-papers.tar.gz"`.

- [ ] **Step 4: Human uploads**

Dem Menschen nennen, in Bash-Syntax mit `C:/…`-Pfaden:

~~~bash
gh release upload archive/parity-recordings "<S>/archive/working-papers.tar.gz" "<S>/archive/SHA256SUMS" --clobber
~~~

- [ ] **Step 5: Verify the upload**

~~~bash
mkdir -p "$S/archive/check"
gh release download archive/parity-recordings -D "$S/archive/check"
~~~

Dann im Ordner `check` den Befehl `sha256sum -c SHA256SUMS` ausführen. Erwartet:
- jede Zeile `OK`, auch die alten Anhänge,
- `cmp` des heruntergeladenen Tarballs mit dem lokalen ist gleich.

---

### Task 10: Die Papiere verlassen den Baum

**Files:**
- Delete: jede Datei aus `git ls-files docs/.superpowers` außer den beiden Neustart-Papieren
- Regenerate: `_identities.tsv`

- [ ] **Step 1: Keep a copy of this plan**

`cp docs/.superpowers/plans/2026-10-05-stufe-4f-d-abschluss.md "$S/plan.md"`. Task 11 liest von dort.

- [ ] **Step 2: Delete**

~~~bash
git rm -r -q docs/.superpowers
git checkout HEAD -- docs/.superpowers/specs/2026-10-05-loomux-release-neustart-design.md docs/.superpowers/plans/2026-10-05-release-bruecke.md
~~~

- [ ] **Step 3: Tarball equals the deletion**

~~~bash
git diff --cached --name-only --diff-filter=D | sort > "$S/t10-deleted.txt"
~~~

Dann aus `$S/archive` heraus `tar -tzf working-papers.tar.gz | grep -v '/$' | sort > ../t10-archived.txt` und `diff "$S/t10-deleted.txt" "$S/t10-archived.txt"`.
Expected: keine Ausgabe.

- [ ] **Step 4: Reindex**

Mit dem Probe-Zustand. Expected: `grep "docs/.superpowers/" _identities.tsv | cut -f2` nennt genau die beiden Neustart-Papiere (gemessen am 2026-10-05).

- [ ] **Step 5: Lint, check and the source script**

`lint` und `brain check` sind clean. `uv run --script "$S/t7-sources.py"` meldet 0 Abweichungen.

- [ ] **Step 6: Gate**

Run: `sh ci/gate.sh > "$S/t10-gate.txt" 2>&1`
Expected: Exit 0.

- [ ] **Step 7: No link to a deleted file**

Run: `git grep -n -E "\]\([^)]*(migration\.md|\.superpowers/)" -- '*.md' ':(exclude)docs/.superpowers' ':(exclude)docs/en/benchmarks.md' ':(exclude)docs/de/benchmarks.md'`
Expected: keine Ausgabe.

- [ ] **Step 8: Commit**

~~~bash
git add -A docs/.superpowers _identities.tsv
git commit -m "docs: move the migration's working papers to the archive release" > "$S/commit-t10.txt" 2>&1
~~~

---

### Task 11: Der Tor-Test lässt nur noch Chronik, Changelog und den Hinweis auf das Vorbild zu

**Files:**
- Modify: `internal/namecheck/references.go`
- Test: `internal/namecheck/references_test.go`

- [ ] **Step 1: Write the failing tests**

`TestTheTableLetsOnlyWorkingPapersAndArchivesStandUnderDocsSuperpowers`, `TestTheReadmeRuleTakesRoadmapRowsOnly` und `TestTheArchiveRuleTakesArchiveLinesInTheWikiOnly` fallen. Neu:

~~~go
func TestGraftIsANameExceptAfterAnUnderscore(t *testing.T) {
	files := map[string]string{
		"a.go": "// Ported from trailhq/Graft\n",
		"b.go": "\t\"GIT_GRAFT_FILE\",\n",
		"c.go": "func graftEdges() {}\n",
		"d.go": "func TestWithGraftsDefaults(t *testing.T) {}\n",
		"e.md": "Graft's crux\n",
	}
	got, _ := References(keys(files), reader(files), nil)
	want := []string{
		"a.go:1: // Ported from trailhq/Graft",
		"c.go:1: func graftEdges() {}",
		"d.go:1: func TestWithGraftsDefaults(t *testing.T) {}",
		"e.md:1: Graft's crux",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTheIdeaRuleTakesTheIdeaSentenceOnly(t *testing.T) {
	files := map[string]string{
		"README.md":               "Ranking is inspired by [trailhq/Graft](u).\n| model | Graft |\n",
		"README.de.md":            "Ranking ist angeregt von [trailhq/Graft](u).\nGrafts Crux\n",
		"docs/en/architecture.md": "principles inspired by [trailhq/Graft](u):\n",
		"docs/de/architecture.md": "angeregt von [trailhq/Graft](u):\n",
		"docs/en/guide.md":        "inspired by [trailhq/Graft](u)\n",
	}
	got, _ := References(keys(files), reader(files), exceptions())
	want := []string{
		"README.de.md:2: Grafts Crux",
		"README.md:2: | model | Graft |",
		"docs/en/guide.md:1: inspired by [trailhq/Graft](u)",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNoExceptionIsLeftForPapersWikiPlanOrRoadmap(t *testing.T) {
	files := map[string]string{
		"docs/.superpowers/specs/a.md":                   "ultraloom\n",
		"docs/.superpowers/specs-ub/a.md":                "ultraloom\n",
		"docs/wiki/log.md":                               "ultraloom\n",
		"docs/wiki/a.md":                                 "docs/.superpowers/specs-ub/x.md\n",
		"_identities.tsv":                                "x\tdocs/.superpowers/bench-ub/y.md\n",
		"internal/brain/apply/testdata/frontmatter/w.md": "specs-ub/x.md\n",
		"docs/en/migration.md":                           "ultraloom\n",
		"README.md":                                      "| **Web** | ultra-brain/web | W1 |\n",
	}
	got, _ := References(keys(files), reader(files), exceptions())
	if len(got) != len(files) {
		t.Fatalf("got %d findings, want one per file: %q", len(got), got)
	}
}

func TestTheNoticeStandsWholeAndTheGeneratorFileToo(t *testing.T) {
	files := map[string]string{
		"internal/notices/NOTICE.md":     "from trailhq/Graft\nGraft's license\n",
		"internal/dev/notices/ported.md": "trailhq/Graft <u>\n",
		"internal/dev/notices/other.md":  "trailhq/Graft\n",
	}
	got, _ := References(keys(files), reader(files), exceptions())
	want := []string{"internal/dev/notices/other.md:1: trailhq/Graft"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
~~~

Die Meldung in `TestTheRepositoryNamesNoPredecessor` lautet dann: `"names of the predecessor tools or of the ported original outside the exceptions in references.go:\n%s"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/namecheck -count=1 > "$S/t11-red.txt" 2>&1`
Expected: FAIL.
- `TestGraftIsANameExceptAfterAnUnderscore`: `got []`.
- `TestTheIdeaRuleTakesTheIdeaSentenceOnly`: `got []`.
- `TestNoExceptionIsLeftForPapersWikiPlanOrRoadmap`: weniger Befunde als Dateien.
- `TestTheNoticeStandsWholeAndTheGeneratorFileToo`: `got []`.

- [ ] **Step 3: Write minimal implementation**

In `references.go` hängt `predecessorNames` `|(?:^|[^_])graft` an die Suchliste an. Der Kommentar darüber lautet:

~~~go
// The names of the two tools loomux replaced, and of the project its code
// graph is ported from: each may stand only in an exception. `graft` after
// an underscore is git's GIT_GRAFT_FILE, not the project.
~~~

`inTheArchives` und `followUps` fallen samt Kommentar. Neu ist:

~~~go
// ideaLine is the one sentence per README and architecture page that names
// where the code graph's idea came from.
var ideaLine = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)(inspired by|angeregt von) \[?trailhq/Graft`)
})
~~~

`exceptions` mit seinem Kommentar:

~~~go
// exceptions are the places a name may stand: the measurement chronicle,
// the five changelog lines that are history, the license notice of the
// ported code and the sentence that names its idea.
var exceptions = sync.OnceValue(func() []Exception {
	return []Exception{
		{Path: "docs/en/benchmarks.md", Owner: "history"},
		{Path: "docs/de/benchmarks.md", Owner: "history"},
		{Path: "testdata/bench/1a-hooks.json", Owner: "history"},
		{Path: "testdata/bench/search/v1/baseline/", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "7.0.1", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "7.0.0", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "4.2.2", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "2.5.0", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "2.3.0", Owner: "history"},
		{Path: "internal/notices/NOTICE.md", Owner: "license"},
		{Path: "internal/dev/notices/ported.md", Owner: "license"},
		{Path: "README.md", Line: ideaLine(), Owner: "idea"},
		{Path: "README.de.md", Line: ideaLine(), Owner: "idea"},
		{Path: "docs/en/architecture.md", Line: ideaLine(), Owner: "idea"},
		{Path: "docs/de/architecture.md", Line: ideaLine(), Owner: "idea"},
		{Path: "internal/namecheck/references.go", Owner: "self"},
		{Path: "internal/namecheck/references_test.go", Owner: "self"},
	}
})
~~~

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/namecheck -count=1 -cover > "$S/t11-green.txt" 2>&1`
Expected: PASS, auch `TestTheRepositoryNamesNoPredecessor`, Coverage 100 %. Fällt der Repository-Test, nennt er die Zeile. Sie wird in ihrem Task-Thema korrigiert (Fixup auf den Commit dieses Themas), nicht durch eine neue Ausnahme.

- [ ] **Step 5: Mutation**

Drei Overlays mit Pfaden in der Form `C:/…`, die Ausgabe je nach `$S/t11-mut-<n>.txt`:
1. `[^_]` → `.?` muss `TestGraftIsANameExceptAfterAnUnderscore` rot machen (`b.go`).
2. `ideaLine` ohne `\[?` muss `TestTheIdeaRuleTakesTheIdeaSentenceOnly` rot machen (die README-Zeile mit `[`).
3. Die Zeile `internal/dev/notices/ported.md` aus der Tabelle streichen muss `TestTheNoticeStandsWholeAndTheGeneratorFileToo` rot machen.

- [ ] **Step 6: Gate and commit**

Run: `sh ci/gate.sh > "$S/t11-gate.txt" 2>&1`, Expected: Exit 0.

~~~bash
git add internal/namecheck
git commit -m "test(namecheck): allow only the chronicle, the changelog and the code graph's notice" > "$S/commit-t11.txt" 2>&1
~~~

---

## Nach dem Plan

- **Pull Request** mit dem Skill `release-pr`:
  - Label `release:minor`.
  - Zeile `Release: minor — loomux dev notices quotes the license of ported code`.
  - Changelog-Block mit `### Added` (Abschnitt „Ported sources“ in `NOTICE.md`) und `### Removed` (der Migrationsplan `docs/{en,de}/migration.md`, offene Arbeit steht auf der Roadmap).
- **Neustart-Zweig:** `feat/release-restart` nach dem Merge per `git rebase --onto` umsetzen. Sein Plan nennt `internal/plancheck` (Z. 40 und 135). Beim Rebase wird daraus `internal/namecheck`, sonst läuft ein `go test ./internal/plancheck` dort ins Leere.
- **Nicht in diesem Plan** (kein Name der Suchliste, kein toter Verweis):
  - die Meldungen „recorded cases“ in `internal/cli/cases_*_test.go`,
  - die `.loomux`-Zeilen in `testdata/cases/2a/check/*/notes.md`,
  - holprige `stderr`-Dateien,
  - die Wendungen „the reference“ in `cli-reference.md`.
