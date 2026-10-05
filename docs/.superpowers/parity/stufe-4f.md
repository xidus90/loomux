# Stufe 4f — Gleichheitsprüfung gegen die Ursprungsrepos

**Stand:** 2026-10-05, Zweig `docs/predecessor-equality`.
- **Aufgenommen:** gegen den Baum `f956589a`, also `origin/master` 80db6bb0
  (4e ✅) mit der Brücke.
- **Umgesetzt:** auf `origin/master` 253f46f1, wo die Brücke als #80 gemergt
  ist. Der Inhalt ist gleich, `git diff f956589a 253f46f1` ist leer.
- **Nachgeprüft:** Beide Ursprungsrepos haben auch nach dem Umsetzen keinen
  neuen Commit (`git fetch --all`, jüngster Commit über alle Zweige vom
  2026-09-14).
**Bezug:** Spec `docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md`
(„PR A: Gleichheitsprüfung“), Fusions-Spec „#24 im Einzelnen“, Schritt 1.
**Ursprungsrepos:** `C:/Users/micro/Documents/#GIT/ultraloom` (Remote
`xidus90/ultra-loom`) und `C:/Users/micro/Documents/#GIT/ultra-brain` (Remote
`xidus90/ultra-brain`), beide vorher mit `git fetch --all` geholt. Beide
werden nur gelesen.

## 1. Gleichheitsbeleg

Befehle: `git -C <repo> rev-parse --short <tag>^{}`, `git -C <repo> rev-parse
--short master`, `git -C <repo> merge-base --is-ancestor origin/master master`
und `git -C <repo> rev-list --count origin/master..master`.

| Repo | Tag | Commit | `master` lokal | `origin/master` | gleich |
|---|---|---|---|---|---|
| ultraloom | `loomux-1a-source` | `9d01a60` | `9d01a60` | `d025d80` | ja, Tag = `master` |
| ultra-brain | `loomux-1a-source` | `3cc72d2` | `3cc72d2` | `08d985a` | ja, Tag = `master` |
| ultra-brain | `loomux-3-source` | `3cc72d2` | `3cc72d2` | `08d985a` | ja, Tag = `master` |

`origin/master` liegt in beiden Repos **hinter** dem lokalen `master`. Er ist
Vorfahre davon, mit 3 lokalen Commits voraus in ultraloom und 19 in
ultra-brain. Der lokale `master` ist also der jüngste Stand der Hauptlinie,
und jeder Aufnahme-Tag zeigt auf ihn. Die Commits, die nur lokal liegen, sind
nie gepusht worden. Für die Gleichheit zählt das nicht, denn aufgenommen wurde
vom lokalen Stand.

**Zuordnung der Stufen zu den Tags** (Quellenzeilen der Akten):

| Stufe | Quelle | Beleg |
|---|---|---|
| 1a | ultraloom `9d01a60`, ultra-brain `3cc72d2`, beide `loomux-1a-source` | `stufe-1a.md:3` |
| 1b-1 | ultra-brain `loomux-1a-source` (`3cc72d2`) | `stufe-1b-1.md:3` |
| 1b-2 | ultra-brain, Referenz-Worktree `loomux-1a-source` | `stufe-1b-2.md:3-6` |
| 2a | ultraloom `loomux-1a-source` (`9d01a60`) | `stufe-2a.md:3` |
| 2b | ultraloom `commit-msg` ohne Tag | `stufe-2b.md:3`; `git log loomux-1a-source..master -- src/ultraloom/commit tests/commit` ist leer, also gilt derselbe Stand |
| 2c | ultraloom `loomux-1a-source` (`9d01a60`) | `stufe-2c.md:3` |
| 3a, 3b, 3c | ultra-brain `loomux-3-source` (`3cc72d2`) | `stufe-3a.md:3`, `stufe-3b.md:3`, `stufe-3c.md:10` |
| 4a-2 | ultraloom `9d01a60` (`cmd/init`), ultra-brain `3cc72d2` (Skills, post-merge) | `stufe-4a-2.md:13-18` |
| 4c-1, 4d | ultra-brain `loomux-3-source` (`3cc72d2`) | `stufe-4c-1.md:3`, `stufe-4d.md:3` |

**Ergebnis:** Jede Aufnahme stammt vom Kopf der Hauptlinie ihres Repos. Seit
der Aufnahme hat sich die Hauptlinie nicht bewegt: Der letzte Commit auf
`master` ist vom 2026-09-13 (ultraloom) bzw. 2026-09-14 (ultra-brain). Eine
Neuaufnahme wegen einer bewegten Referenz ist nicht nötig. Was sich nur auf
Seitenzweigen bewegt hat, ist Sache der Inventur (Abschnitt 3).

**Seitenzweige**, gezählt mit `git rev-list --count master..<zweig>` und
umgekehrt:

| Repo | Zweig | vor `master` | hinter `master` |
|---|---|---|---|
| ultraloom | `claude/ultra-loom-brain-fusion-a5bb17` (auch remote) | 4 | 0 |
| ultraloom | `claude/wiki-stufe-2` | 2 | 0 |
| ultraloom | `feature/agent-harness` | 61 | 5 |
| ultraloom | `fix-audit-scheibe5` | 1 | 16 |
| ultraloom | `update-offene-aufgaben-archiv` | 0 | 5 |
| ultra-brain | `claude/eager-engelbart-46d10e` | 2 | 14 |
| ultra-brain | `claude/exciting-sanderson-efc25b` | 2 | 208 |
| ultra-brain | `claude/scheibe-9b` | 1 | 338 |
| ultra-brain | `docs/artefakte-nach-lebensdauer` | 3 | 189 |
| ultra-brain | `feature/artefakte-nach-lebensdauer` | 44 | 39 |
| ultra-brain | `feature/scheibe-6-lokales-modell` | 0 | 44 |
| ultra-brain | `otter` | 0 | 362 |

**Arbeitsbäume:** Ungetrackt oder geändert ist in ultraloom `.gitignore`,
`docs/wiki/log.md`, `.githooks/post-merge`, `_identities.tsv`, mehrere
`index.md`, `docs/.superpowers/specs/2026-08-24-multi-provider-llm-design.md`,
`graph.json`, `guard.exe` und `layout.json`. In ultra-brain sind es
`OFFENE_AUFGABEN.md`, ein Plan, eine Spec, zwei Übergaben sowie
`scripts/install.ps1` und `scripts/install.sh`. Die Inventur ordnet sie ein.

**Sauberer Arbeitsbaum.** Die Spec verlangt für den Beleg, dass der
Arbeitsbaum zur Zeit der Aufnahme sauber war. Die älteren Akten sagen
dazu nichts. Heute stehen dort Änderungen, und die Stufen 1a bis 2c tragen
nur diesen Beleg, keine Stichprobe. Darum folgt jede Änderung einzeln, mit
Änderungszeit (`ls -la --time-style=+%F`) und dem Grund, warum sie keine
Aufnahme beeinflusst:

| Repo | Pfad | Zeit | Warum ohne Einfluss |
|---|---|---|---|
| ultraloom | `.gitignore` | — | eine Zeile `+/docs/wiki/.obsidian` (`git diff -- .gitignore`); kein aufgenommener Befehl liest sie |
| ultraloom | `docs/wiki/log.md` | — | nur Zeilenenden (`git diff --stat` zeigt die Datei nicht als geändert) |
| ultraloom | `.githooks/post-merge` | 2026-10-02 | nach jeder Aufnahme entstanden (`loomux merge-hook`) |
| ultraloom | `_identities.tsv`, `graph.json`, die `index.md`-Kataloge | 2026-09-22 | Ausgaben des Indexlaufs von ultra-brain; die aufgenommenen ultraloom-Befehle (`check`, Hooks, `commit-msg`, `ulinit`) lesen sie nicht |
| ultraloom | `guard.exe` | 2026-09-13 | verirrter Build von `cmd/guard`; die Aufnahmen rufen `ulguard`, nicht diese Datei |
| ultraloom | `layout.json` | 2026-09-10 | Ausgabe von `brain layout`, von keinem aufgenommenen Befehl gelesen |
| ultraloom | `specs/2026-08-24-multi-provider-llm-design.md` | — | Doku, #22 |
| ultra-brain | `OFFENE_AUFGABEN.md`, Plan, Spec, zwei Übergaben | — | Doku, von keinem Befehl gelesen |
| ultra-brain | `scripts/install.{ps1,sh}` | — | Installationsskripte, gehören nicht zum aufgenommenen Pfad (`brain-mcp.exe` aus `.venv`) |

Die Aufnahmen hängen damit nur am getrackten Stand des Tags.

## 2. Stichprobe der Reproduzierbarkeit

Jede Stufe, für die ein Aufnahmeskript liegt, wurde am 2026-10-05 neu
aufgenommen. Danach wurde mit `git status`/`git diff` gegen die versionierten
Aufnahmen verglichen und der Baum zurückgesetzt. Nach jeder Stufe war
`git status --porcelain testdata/` leer.

**Werkzeuge:**
- `brain-mcp.exe` aus `ultra-brain/.venv` mit Python 3.14.7.
- `go1.27.0 windows/amd64`.
- `loomux dev record-case` aus diesem Baum (`0.0.0-dev`, Stand `f956589a`).
- `fakeqmd` aus `internal/dev/fakeqmd/_qmd`.
- Für 3c die Go-Referenz `brain` vom Tag `3cc72d2`. Sie ist in den Scratchpad
  gebaut, weil das in `record.sh` genannte
  `C:/Users/micro/Documents/#GIT/brain-3c.exe` nicht mehr auf der Platte
  liegt.
- `pdftotext` auf dem PATH ist Xpdf 4.06. Die 4d-Aufnahme nennt Poppler
  25.07.0 für `dev record-poppler`; die 4d-Fälle unter `4d-source` hängen
  nicht daran.

**Wie die Unterschiede eingeordnet sind:** Für jede geänderte, neue oder
gelöschte Datei werden die entfernten und die hinzugefügten Zeilen
verglichen. Vorher werden flüchtige Werte maskiert: Zeitstempel, Datum
(auch im Pfad), ULID, SHA-256, Commit-Hash, Epoche und Dauer. Neue Dateien
gehen vorher per intent-to-add in den Index, damit ein Fallordner, dessen
Name das Datum trägt, mit seinem alten Gegenstück verglichen wird. Sind die
Zeilen nach dem Maskieren paarweise gleich, ist die Datei Klasse (b), sonst
(c). Das Skript liegt im Scratchpad der Sitzung (`classify.py`). Es brauchte
drei Fassungen:
- Die erste scheiterte an Pfaden mit Umlauten.
- Die zweite hielt die Löschungen versehentlich vom Vergleich fern.
- Die dritte las Inhaltszeilen mit `---` als Diff-Kopf.

Alle Zahlen unten stammen aus der dritten Fassung.

| Stufe | Fälle | Pfade geändert | (a) gleich | (b) nur flüchtig | Binär | (c) Verhalten | Exit des Laufs |
|---|---|---|---|---|---|---|---|
| 3a | 28 | 68 | Rest | 54 | 0 | 0, dazu 14 neue `stderr` (siehe unten) | 0 |
| 3b | 24 | 33 | Rest | 30 | 3 | 0 | 0 |
| 3c | 35 | 0 | alle | 0 | 0 | 0 | 0 |
| 4a-2 | 14 | 0 | alle | 0 | 0 | 0 | 0 |
| 4c-1 | 7 | 21 | Rest | 21 | 0 | 0 | 0 |
| 4d | 29 | 23 | Rest | 23 | 0 | 0 | 0 |

**Erklärungen:**
- **3b, Binär:** Die drei Binärdateien sind
  `approve/{amend,reject,success}/world_after/maintenance/index`, der
  Git-Index eines Weltrepos. Er trägt Stat-Daten (mtime, inode), ist also
  flüchtig.
- **3a, `stderr`:** Die 14 neuen Dateien sind die Standardfehlerausgabe von
  `area-add/no-registry`, `embed/*`, `reconcile/{no-review-centre,
  two-review-centres, unreadable-case}` und `reindex/*`. Keine versionierte
  3a-Aufnahme hat eine `stderr`-Datei: 0 der 100 `stderr`-Dateien unter
  `testdata/cases` gehören zu 3a (`git ls-files 'testdata/cases/**/stderr'`),
  und der Aufnahme-Commit `530de1e4` enthält keine. Am Stand von `530de1e4`
  schreibt der Recorder eine nicht leere Standardfehlerausgabe allerdings
  schon (`git show 530de1e4:internal/dev/recordcase/recordcase.go`, Zeilen
  84–107). Das kam mit `36d64029` (`git log -S'cmd.Stderr' --
  internal/dev/recordcase`, verfasst 2026-09-22 11:04). `530de1e4` ist um
  15:43 verfasst und um 19:02 committet.
  - **Schluss, nicht belegt:** Die 3a-Fälle wurden vermutlich mit einem
    Recorder-Binary aufgenommen, das älter war als `36d64029`, und der Zweig
    wurde danach auf 2c gesetzt. Das ist eine Folgerung aus diesen
    Zeitpunkten.
  - **Was gilt:** Die 14 Dateien tragen echte Meldungen der Referenz
    (`class-3a.txt`). `stdout`, `exit` und `world_after` derselben Fälle
    sind (a) oder (b). Weil keine versionierte 3a-Aufnahme eine `stderr`
    trägt, prüft die Wiedergabe von 3a heute ohnehin keine
    Standardfehlerausgabe.
  - Damit ist es ein erklärter Unterschied, der mit dieser Akte vorgelegt
    wird. Ein Befund am Verhalten ist es nicht.
- **3a, Skript:** `stufe-3a-orakel/record.sh` schreibt fest nach
  `…/worktrees/planung-von-3-c56c81`, einem Worktree, den es nicht mehr gibt.
  Aufgenommen wurde aus einer Kopie im Scratchpad, in der `WT` auf diesen
  Worktree zeigt. Die Datei im Baum bleibt, wie sie ist; sie geht mit PR C.
- **4c-1, 3a, 3b:** Die Prüffälle heißen `a-<datum>-5bd8`. Ihr Ordner zieht
  bei der Neuaufnahme auf das heutige Datum um. Nach dem Maskieren ist er
  gleich.
- **1a bis 2c:** Es gibt kein Aufnahmeskript, also trägt der Beleg aus
  Abschnitt 1 allein.

**Ergebnis:** Die Aufnahmen sind reproduzierbar. Es gibt keinen Befund der
Klasse (c). Die Unterschiede sind erklärt (flüchtige Werte, die 14
`stderr`-Dateien von 3a) und werden mit dieser Akte zur Kenntnis vorgelegt.

## 3. Inventur beider Repos

Je Repo hat ein Subagent nur gelesen: die Oberfläche auf `master` am Code
statt an der Doku, jeden Commit der Seitenzweige und jede ungetrackte oder
geänderte Datei. Jede Stelle bekam genau eine Zuordnung:
- **gebaut:** mit Fundstelle in loomux,
- **Stufe/Folgeprojekt:** mit Zeile der Fusions-Spec,
- **weggefallen:** mit Nachtrag oder Aktenzeile,
- **Doku**,
- **erzeugtes Artefakt**,
- **ohne Zuordnung.**

Die Funde ohne Zuordnung, die etwas tragen, hat der Controller per grep
nachgeprüft.

| Repo | Oberfläche `master` | Commits auf Seitenzweigen | Arbeitsbaum | ohne Zuordnung |
|---|---|---|---|---|
| ultraloom | 49 Zeilen (Befehle, Flags, Schlüssel, Hooks, Skills, Vorlagen) | 68 (4 + 2 + 61 + 1) | 14 Pfade | 5 |
| ultra-brain | rund 34 Python- und 19 Go-Befehle, rund 70 Flags, 5 MCP-Werkzeuge, 5 Skills, 11 Hook- und Vorlagendateien, rund 35 Schlüssel und Variablen | 55 (2 + 2 + 1 + 3 + 44, dazu 3 unter `refs/original`) | 7 Dateien | 7, dazu 2 Sicherungsbefunde |

**Ergebnis:** Keine Fähigkeit ist ohne Zuordnung verloren.
- **ultraloom:** Die Prüfkette, die Hooks, `commit-msg`, `ulguard`, `ulinit`
  und die Flow-Laufzeit von `feature/agent-harness` sind gebaut. Die
  Laufzeit liegt unter `internal/flow/*`, die 61 Commits ordnen sich ihr
  Thema für Thema zu.
- **ultra-brain:** Alle Befehle bis auf einen Alias sind gebaut, ebenso die
  MCP-Werkzeuge, Manifest, Registry und Skills. `layout` und `web` gehen
  mit dem Web-Folgeprojekt, und `feature/artefakte-nach-lebensdauer` ist
  durch #20 und #29 entschieden.

Die offenen Punkte unten sind klein oder betreffen die Sicherung vor dem
Löschen.

**Nachgeprüft** (Befehl → Ergebnis):

| Fund | Befehl | Ergebnis |
|---|---|---|
| `NeighbourWiki` ohne Aufrufer | `git grep -n NeighbourWiki -- '*.go'` | nur `internal/detect/edges.go:44` und Tests |
| Drei ulflow-Pläne fehlen | `ls docs/.superpowers/plans-ul/` gegen `git ls-tree feature/agent-harness` | in loomux nur die drei `-befunde.md`; `…-welle-0-und-1.md`, `…-welle-2.md`, `…-welle-3.md` liegen nur auf dem Zweig |
| Alias `brain-mcp wiki gate` | `grep -n '"gate"' src/brain/cli.py` | `cli.py:665`, Unterbefehl von `wiki` neben `wiki-gate` |
| `docs_language` | `git grep -n docs_language -- internal cmd` | kein Treffer; nur `ulinit` las es |
| Korpus von `claude/scheibe-9b` | `git show --stat 835e118` | `bench/cases/reconcile/stat-cache-hit/…` u. a., nur auf dem Zweig |
| Nicht gepushte Stände | `git rev-list --count origin/master..master`; `git for-each-ref refs/remotes` | ultraloom 3, ultra-brain 19; auf den Remotes liegen nur `origin/master` und in ultraloom `origin/claude/ultra-loom-brain-fusion-a5bb17`, alle anderen Zweige sind nur lokal |

**Ohne Zuordnung, mit Vorschlag:**

| # | Repo | Stelle | Vorschlag |
|---|---|---|---|
| 1 | ultraloom | `NeighbourWiki` (`internal/detect/edges.go:44`), früher für `ulinit --wiki-mode neighbour_repo`; #26 legt das Wiki ins Projekt, ein Wegfall des Nachbar-Repos steht nirgends | Wegfall, Funktion samt Tests in 4f PR B löschen |
| 2 | ultraloom | `docs_language` aus `ulinit` (`parity/stufe-4e.md:547-549`, Befund 3 ohne Freigabe in #28) | Wegfall |
| 3 | ultraloom | `ultraloom run --no-model` (Diagnoselauf) | Eingang in die Spec von Flow B/M3 |
| 4 | ultraloom | drei ulflow-Pläne nur auf `feature/agent-harness`, obwohl die Spec von Flow A (Zeile 65) die Arbeitspapiere unverändert mitnimmt | in `plans-ul/` kopieren |
| 5 | ultra-brain | `brain-mcp wiki gate`, Alias von `wiki-gate` | Wegfall |
| 6 | ultra-brain | Wurzelquellen `Bauanleitung_Second-Brain.pdf`, `NoteGPT_Transcript…txt` (Quellen der alten Spec §19), in brain-knowledge nicht unter diesem Namen gefunden | vor dem Löschen am Inhalt prüfen, ob brain-knowledge sie trägt; sonst in den Eingang übernehmen |
| 7 | ultra-brain | `claude/eager-engelbart-46d10e` (`5ffd955`, `c6325b4`): Verdrahtung eines Entwickler-MCP, keine Fähigkeit | Wegfall |
| 8 | ultra-brain | Korpus von `claude/scheibe-9b` (`stat-cache-hit`, `unexpected-args`, `clean-no-changes`, `standing-case-manual`): Verhalten gebaut und per Unit-Test geprüft, Fälle nicht gleichnamig | Wegfall; Exit 2 für `reconcile` mit unerwartetem Argument einmal proben |
| 9 | ultra-brain | `cdf5dfd` auf `docs/artefakte-nach-lebensdauer`: „reconcile schiebt den Hash einer Zeile vor, die keine Seite zitiert, ohne die Revision“ | als Satz in die Roadmap-Zeile „Register revisions only from review“ |
| 10 | ultra-brain | `scripts/install.{sh,ps1}` von ultra-brain selbst; #15 nennt nur die von ultraloom | #15 um „und die von ultra-brain“ ergänzen |
| 11 | ultra-brain | Web 7a-2 (`edges-cross.json`) und 7b (Seitenleser, Volltextsuche, Prüfzentrum im Web) aus `OFFENE_AUFGABEN.md`; die Roadmap-Zeile W2 nennt sie nicht | in die Eingangsliste des Web-Folgeprojekts |

**Sicherung vor dem Löschen der Repos.** Nur auf dieser Platte liegen:
- **ultraloom:**
  - 3 Commits auf `master`, darunter der Aufnahme-Tag `loomux-1a-source`,
  - `feature/agent-harness` (61 Commits), `claude/wiki-stufe-2`,
    `fix-audit-scheibe5`,
  - `refs/original/refs/heads/feat/policy-baukasten`: 17 Commits auf keinem
    Zweig (`git rev-list --count … --not --branches`). Es sind Kopien vor
    einem Umschreiben; ihre Betreffzeilen stehen auch auf Zweigen,
  - die ungetrackten Dateien.
- **ultra-brain:**
  - 19 Commits auf `master`, darunter beide Aufnahme-Tags,
  - fünf Seitenzweige, `refs/original/…` und sieben ungetrackte Dateien.

Vorschlag: Je Repo ein `git bundle --all` samt einem Archiv der
ungetrackten Dateien, abgelegt dort, wo #24 (e) die Aufzeichnungen sichert
(Tag oder Release-Anhang). Das gehört vor jedes Löschen eines Ursprungsrepos.

## 4. Vor PR B

Proben vom 2026-10-05, Zweig `refactor/drop-predecessor-references`, vor dem
Löschen der Verweise. Sie lesen nur; die Registry ist die echte dieser
Maschine, geschrieben wird dort nichts.

### Registry

Es gilt `$LOCALAPPDATA/loomux/registry.toml`, also
`C:\Users\micro\AppData\Local/loomux/registry.toml`: `LOOMUX_STATE_DIR` ist
nicht gesetzt, und `defaultStateDir` (`internal/config/registry.go:229`) bildet
unter Windows `%LOCALAPPDATA%\loomux`.

Befehl: `ls -la "$REG"; cat "$REG"` mit
`REG="${LOOMUX_STATE_DIR:-$LOCALAPPDATA/loomux}/registry.toml"`. Die Datei hat
1816 Byte und zwölf Einträge `[[area]]`; die Pfade stehen in der Tabelle
darunter, wie die Registry sie schreibt.

### Bereiche

Befehl: ein Skript, das jede Zeile `path = "…"` der Registry liest und je Pfad
prüft, ob `.loomux/config.toml` mit einer Zeile `[area]` da ist, ob eine
`.brain.toml` liegt und ob es einen Ordner `.ultra-brain/` gibt (mit Zahl der
`*.md` darin; `find "$p/.ultra-brain" -name '*.md' | wc -l`).

| Pfad | `[area]` in `.loomux/config.toml` | `.brain.toml` | `.ultra-brain/` |
|---|---|---|---|
| `brain-knowledge/92 Engineering/craft` | ja | nein | nein |
| `brain-knowledge/92 Engineering/python` | ja | nein | nein |
| `brain-knowledge/91 Projekte` | ja | nein | nein |
| `brain-knowledge` | ja | nein | nein |
| `D:/GitHub/classic-game-bench` | ja | nein | nein |
| `ecoflow` | ja | nein | nein |
| `iam_wiki` | ja | nein | nein |
| `loomux` | ja | nein | nein |
| `space` | ja | nein | nein |
| `iam_backend` | nein, Arbeitsbereich (`workspace = true`) | nein | nein |
| `iam_frontend` | nein, Arbeitsbereich (`workspace = true`) | nein | nein |
| `iam_workers` | nein, Arbeitsbereich (`workspace = true`) | nein | nein |

(Pfade unter `C:/Users/micro/Documents/#GIT/`, wo nichts anderes steht.)

Die drei Bereiche ohne `[area]` sind genau die drei Einträge der Registry, die
`workspace = true` tragen und kein `wiki`; ihre `.loomux/config.toml` hat
`[modules]`, `[guard]`, `[commit]`, `[maintenance]`, `[verify]` (Befehl:
`grep -n '^\[' <pfad>/.loomux/config.toml`). Die übrigen Arbeitsbereiche
(`classic-game-bench`, `ecoflow`, `loomux`, `space`) tragen neben
`workspace = true` ein `wiki` und `[area]`. Kein Bereich hat ein Altmanifest
ohne `[area]`, keiner eine `.brain.toml`, keiner einen Ordner `.ultra-brain/`.
Die Vorbedingung der Spec vor dem Merge ist erfüllt: Nach dem Löschen der
Altmanifest-Leser wird kein registrierter Bereich still manifestlos, und kein
Bereich bringt `*.md` unter `.ultra-brain/` zum Indexieren mit.

### `reconcile` mit unerwartetem Argument

`reconcileCommand` (`internal/cli/maintenance.go`) prüft mit `refusesArguments`
vor jedem Zugriff auf den Zustand. Die Probe lief trotzdem gegen einen leeren
Zustandsordner (`LOOMUX_STATE_DIR` auf einen frischen Scratch-Ordner).

Befehl: `LOOMUX_STATE_DIR="$SCRATCH/t1-state" go run ./cmd/loomux reconcile unexpected-arg`
gab auf stderr `loomux reconcile: unrecognized arguments: unexpected-arg` und
`exit status 2`; `go run` selbst meldet dabei Exit 1 und gibt den Code des
Programms nur als Text weiter. Darum noch einmal mit dem gebauten Programm:
`go build -o "$SCRATCH/loomux-t1.exe" ./cmd/loomux`, dann
`LOOMUX_STATE_DIR="$SCRATCH/t1-state" "$SCRATCH/loomux-t1.exe" reconcile unexpected-arg`:
dieselbe Meldung, Exit 2. `ls -A "$SCRATCH/t1-state" | wc -l` gab 0: der
Zustandsordner blieb leer. Das gilt für den Fall des Korpus von
`claude/scheibe-9b` (#32): ein unerwartetes Argument wird verweigert, bevor
der Zustand berührt wird.

### Ein Altmanifest allein antwortet wie kein Manifest (Unit-Probe)

Elf Tests hielten bis dahin fest, dass ein Verzeichnis mit nur `.brain.toml`
(oder `.ultra-brain/config.toml`) abgelehnt wird und die Meldung `an old
manifest lies there` trägt. Sie wurden auf die neue Erwartung umgestellt
(„dieselbe Antwort wie für ein leeres Verzeichnis“), liefen zuerst gegen den
alten Leser rot und danach, nach dem Wegfall von `OldManifestNames`,
`ErrOldManifest` und `oldManifestError`, grün. Danach wurden sie gelöscht: Sie
wiederholen nur den Fall „kein Manifest“, den die Nachbartests halten
(`…TakesAPolicyOnlyConfigAsNoDeclaration`, `TestReadAreaDeclarationWithoutAFile…`,
`TestIsUndeclaredTakesExactlyTheTwoAbsences`).

Befehl (RED, vor der Änderung): `go test ./internal/config/ ./internal/brain/...
./internal/cli/ -run 'Undeclared|OldManifest|Old'`. Jeder der elf war
`--- FAIL`. Befehl (grün, nach der Änderung): `go test -v -run '<die elf
Testnamen>'` über die neun Pakete, alle `--- PASS`.

| Aufrufer | Test (gelöscht) | Antwort mit Altmanifest | Ergebnis |
|---|---|---|---|
| `config.ReadAreaDeclaration` | `TestReadAreaDeclarationTakesAnOldManifestAsNone` | `<dir>: no manifest found (.loomux\config.toml)`, `IsUndeclared` | wie ein leeres Verzeichnis, grün |
| `apply.resolve` (Tresor) | `TestResolveTakesAVaultUnderAnOldNameAsNone` (beide Namen) | gleicher Fehlertext wie ohne Datei | wie ein leeres Verzeichnis, grün |
| `house.hubFolder` | `TestHubFolderTakesAnOldManifestAsNone` | gleiche Antwort wie ohne Datei | wie ein leeres Verzeichnis, grün |
| `run.areaManifest` | `TestAreaManifestTakesAnOldManifestAsNone` | kein Manifest, kein Befund | wie ein leeres Verzeichnis, grün |
| `convert.Areas` | `TestAreasTakesAnAreaWithOnlyAnOldManifestAsNone` | Modus `manual_cloud`, kein Manifest | wie ein leeres Verzeichnis, grün |
| `maintenance.Manifests` | `TestManifestsLeavesOutAnAreaWithOnlyAnOldManifest` | ausgelassen | wie ein leeres Verzeichnis, grün |
| `privacy.VisibleAreas` | `TestVisibleAreasLeavesOutAWorkspaceWithOnlyAnOldManifest` | Arbeitsbereich ausgelassen | wie ein leeres Verzeichnis, grün |
| `wiki.DeclaredTypesIn` | `TestDeclaredTypesInTakesAnOldManifestAsNone` | `nil, nil` | wie ein leeres Verzeichnis, grün |
| `loomux brain catalog` | `TestBrainTakesAnOldManifestAsNone` | Exit 1, `no manifest found (…)` ohne Hinweis | wie ein leeres Verzeichnis, grün |
| `loomux lint --scope all` | `TestLintPassesOverAWorkspaceWithOnlyAnOldManifest` | Arbeitsbereich still übergangen | wie ein leeres Verzeichnis, grün |
| `cli.sweepContext` | `TestSweepContextTakesAnOldManifestAsNone` | Vorgaben des Laufs | wie ein leeres Verzeichnis, grün |

### Ein Altmanifest allein, Ende zu Ende über die CLI

Kein Shell-Lauf (der Wächter verweigert `convert` und `area add` aus Bash, und
eine echte Registry darf nicht entstehen), sondern eine Wegwerf-Testdatei in
`internal/cli`, die über `run(...)` zwei Welten fährt, je eine Registry mit
einem Bereich `project/a` (Wiki, `LOOMUX_STATE_DIR` und `XDG_CONFIG_HOME` in
einem Scratch-Ordner): in der einen liegt nur eine `.brain.toml`, in der
anderen nichts. Je Befehl wurden Exit, stdout und stderr verglichen, der
Scratch-Pfad durch `<base>` ersetzt. Die Testdatei wurde danach gelöscht und
nicht committet.

Befehl: `go test ./internal/cli/ -run OldManifestProbe -v`.

| Befehl | Exit (alt / leer) | stdout | stderr | gleich |
|---|---|---|---|---|
| `brain status` | 1 / 1 | leer | `error: <base>/project-a: no manifest found (.loomux\config.toml)` | ja |
| `brain catalog --scope all` | 1 / 1 | leer | dieselbe Zeile | ja |
| `brain catalog --scope project/a` | 1 / 1 | leer | dieselbe Zeile | ja |
| `lint --scope project/a` | 0 / 0 | `project/a`, `no findings`, `no findings` | leer | ja |
| `lint --scope all` | 0 / 0 | wie oben | leer | ja |
| `reconcile` | 1 / 1 | leer | `error: no area declares [layout] review; there is nowhere to put a case` | ja |

`convert` ist über den Unit-Test der ersten Tabelle geprobt. Ein Bereich ohne
Manifest bleibt für die Brain-Leser ein Fehler (wie vor der Änderung für ein
Verzeichnis ohne jede Datei); neu ist nur, dass die Altdatei daran nichts mehr
ändert und nicht mehr genannt wird.

### Mutanten des Lesers

Per `go test -overlay` über `./internal/config/ ./internal/brain/...
./internal/cli/`, nach dem Löschen der elf Tests. Jeder Mutant baut; ein
unveränderter Kontrolllauf blieb grün (24 Pakete `ok`).

| Mutant | Tötende Zeile |
|---|---|
| `IsUndeclared` ohne `\|\| errors.Is(err, ErrNoArea)` | `areadeclaration_test.go:128` (`IsUndeclared(the configuration declares no [area]) = false`), dazu `TestResolveWalksPastAPolicyOnlyConfig` |
| `IsUndeclared` ohne `errors.Is(err, ErrNoManifest) \|\|` | `areadeclaration_test.go:128` (`IsUndeclared(no manifest found) = false`), dazu `TestApprove…` |
| `err != nil &&` statt `err != nil \|\|` | Nullzeiger in `TestReadAreaDeclarationWithoutAFileIsErrNoManifestWithoutAHint` (`areadeclaration_test.go:50`) |
| Zusatz: `info.Mode().IsRegular()` entfernt | `TestReadAreaDeclarationTakesOnlyRegularFiles` (`areadeclaration_test.go:65`) |

## Anhang: Commits der Seitenzweige und ihre Zuordnung

Gelesen mit `git log --format="%h %s" master..<zweig>`. Thema für Thema
zugeordnet, die Belegstelle in loomux nach den Berichten der Inventur.

**ultraloom**

| Zweig | Commits | Thema | Zuordnung |
|---|---|---|---|
| `claude/ultra-loom-brain-fusion-a5bb17` | `eed20b5` `5c1d132` `66a9ee7` `9d190fe` | Fusions-Spec, Plan 1a, Messungen | Doku; die Fusions-Spec liegt in loomux |
| `claude/wiki-stufe-2` | `560709c` `be0011d` | eine benannte Gruppe in `.agents/hooks.json`, `null` an der Wurzel ablehnen | gebaut: `internal/setup/hostfile/merge.go:61,72-75` (#21) |
| `fix-audit-scheibe5` | `02f10c3` | Platzhaltertext in `docs/wiki/audit.md` | Doku; nicht mitgenommen (`parity/umzug-wiki-doku.md`) |
| `feature/agent-harness` | `9139738` `8efa515` `8258f51` `410ff8b` `c290ffa` `9499005` `22adeb1` `0bb9eeb` `a90b19c` `c822c98` `462ea2b` | Merges der Bahnen `ulflow/*` | ohne eigenen Inhalt |
| `feature/agent-harness` | `96df882` `a48a880` `51aa20b` `9de6eea` `3c2ff0c` `8383fe8` `68c7107` `17c082a` `dae5c31` `6167abe` `5b3ebb8` `d7041e5` | Entwurf, Pläne, Befundakten von ulflow | Doku; Specs in `specs-ul/`, Befunde in `plans-ul/`, drei Pläne fehlen (Fund 4) |
| `feature/agent-harness` | `ae6e806` `2e88a17` `6f26a77` | Verträge, Katalog, `coerce` | gebaut: `internal/flow/{block,catalog,coerce}.go` |
| `feature/agent-harness` | `7a414ac` `382a17b` `0df4eaa` | Bedingungen, Kappen, Vorlagen | gebaut: `internal/flow/expr`, `internal/flow/tmpl` |
| `feature/agent-harness` | `de2e371` `7dfd806` `4aa18f2` `4c8144f` `883428f` | Lader in sieben Stufen, Parameter | gebaut: `internal/flow/load` |
| `feature/agent-harness` | `e255caf` `52e5d19` `0c18458` `7b47bd7` `d2dd57f` `e057172` `08b5e4a` | Runner, Nachverfolgung, Pause am Tor, Fehlerkante | gebaut: `internal/flow/runner` |
| `feature/agent-harness` | `6e03b86` `8feedd9` `9c0516d` | Blöcke Tor, Ausgang, Agent | gebaut: `internal/flow/blocks` |
| `feature/agent-harness` | `60b6d76` `60db71d` | Journal, Fingerabdrücke | gebaut: `internal/flow/journal` |
| `feature/agent-harness` | `1c6ed46` `a36e847` `879b52f` | Laufnummern, Marker, Anspruch | gebaut: `internal/flow/runs` |
| `feature/agent-harness` | `9663aeb` `aeecf59` `2580803` | Modell-Attrappe, `[agent]`, Modellkette, Werkzeugprofile | gebaut: `internal/flow/model`, `internal/config/agent.go` |
| `feature/agent-harness` | `216e0f4` `1ef037c` `55b3186` | geänderte Dateien beim Laufstart, Umbenennungen | gebaut: `internal/gitwork` |
| `feature/agent-harness` | `2ef95ec` `ad81011` `ca633e0` `e8f4236` | `ulflow run/resume/replay/show/list`, goldenes Journal | gebaut: `loomux flow …`, `go test ./flows` |
| `feature/agent-harness` | `1707bdc` | Sitzungsstart nennt das Programm, das einen Lauf fortsetzt | gebaut: `internal/hooks/flowruns.go` |
| `feature/agent-harness` | `3c3c53e` | `install.ps1/sh` installieren ulflow | weggefallen (#15) |

Grundlage für `feature/agent-harness` ist die Spec von Flow A
(`2026-09-26-loomux-flow-a-design.md`, Zeilen 10 und 55–80): Das
Python-Nebeneinander fällt weg, die Geschichte zieht nicht mit, die Commits
wurden nach Thema neu geschnitten.

**ultra-brain**

| Zweig | Commits | Thema | Zuordnung |
|---|---|---|---|
| `claude/eager-engelbart-46d10e` | `5ffd955` `c6325b4` | Verdrahtung eines Entwickler-MCP | ohne Zuordnung, Wegfall vorgeschlagen (Fund 7) |
| `claude/exciting-sanderson-efc25b` | `7fa0181` `cbdd537` | fehlendes Laufwerk aus der Laufwerksmaske | gebaut: `internal/brain/guard/path_windows_test.go:51` |
| `claude/scheibe-9b` | `835e118` | Plan 9b, Korpus von neun reconcile-Fällen | Verhalten gebaut (`internal/brain/maintenance/scan.go`), Korpus ohne Zuordnung (Fund 8) |
| `docs/artefakte-nach-lebensdauer` | `0288eda` `8417075` | Entwurf und Plan | Doku; #20, #29 |
| `docs/artefakte-nach-lebensdauer` | `cdf5dfd` | Regel zum Vorschieben unzitierter Zeilen | ohne Zuordnung, Eingang in die Roadmap (Fund 9) |
| `feature/artefakte-nach-lebensdauer` | `5dc9275` `47c5d0e` `5e0e4eb` `68ce190` `489645a` `009f5a1` `9b1e175` `079b8c6` | Thema 1: Artefakte ins Zustandsverzeichnis | Roadmap „Vielleicht“ (#29) |
| `feature/artefakte-nach-lebensdauer` | `5dd3d59` `73579ab` | Thema 2: nur Geburten und Umbenennungen ins Register | Roadmap „Kommt“ (#29) |
| `feature/artefakte-nach-lebensdauer` | `1584af8` `23d657e` `e7b0d9f` `7d4548d` | Thema 3: Aliase, Grabsteine, Union-Merge | Roadmap „Kommt“ (#29) |
| `feature/artefakte-nach-lebensdauer` | `441f7ce` `4591b19` `3bb6bab` `8db9685` | Thema 4: mehrdeutige Umbenennung | in loomux schon gelöst (`parity/artefakte-nach-lebensdauer.md:30`) |
| `feature/artefakte-nach-lebensdauer` | `afdf198` `5f6e61a` `5dffda8` `4d35035` `3e33b35` `9ca743b` `4044249` `51d402f` `88fc4a7` `f0fe34e` | Thema 6: Bereich aus einem verknüpften Worktree | Roadmap „Kommt“ (#29) |
| `feature/artefakte-nach-lebensdauer` | `785e3b8` `f19aaad` `adb42d1` `cdb2531` | Thema 7: Fehler der Worktree-Auflösung des Zweigs | entfallen mit Thema 6 (`:33`) |
| `feature/artefakte-nach-lebensdauer` | `909295a` `de95053` `b68813f` `80e2248` `720d1f8` `3a2885d` `e33174c` `33d34ed` | Thema 8: frühere Indexausgabe räumen | Roadmap „Vielleicht“ (#29) |
| `feature/artefakte-nach-lebensdauer` | `c5b978e` `ad4b992` | Thema 9: Schranke und Registercommit | Roadmap (#29) |
| `feature/artefakte-nach-lebensdauer` | `45ef87a` `cb2ea0e` | Spec und Plan tragen, Übergabe | Doku |
| `refs/original/refs/heads/claude/scheibe-7a` | `134d7a7` `d2b1500` `5d8194d` | Kopien vor einem Umschreiben | auf `master` als `ea2f5fd` `fea5bc3` `cf56bff`; Web-Folgeprojekt |

Zusammen sind das 44 Commits auf `feature/artefakte-nach-lebensdauer`, 61 auf
`feature/agent-harness` (11 Merges, 12 Doku, 38 Code), dazu die kleinen
Zweige. Die Berichte der beiden Inventur-Agenten kamen als Rückmeldung in
die Sitzung und wurden nicht als Datei abgelegt; diese Tabellen geben ihren
Inhalt je Commit wieder, die Zählungen sind gegen `git log` nachgerechnet.
