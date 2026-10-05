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
  `testdata/cases` gehören zu 3a, und der Aufnahme-Commit `530de1e4` vom
  2026-09-22 enthält keine. Der Recorder schrieb sie damals also nicht. Das
  ist eine Änderung am Recorder, nicht an der Referenz. `stdout`, `exit` und
  `world_after` derselben Fälle sind (a) oder (b).
- **3a, Skript:** `stufe-3a-orakel/record.sh` schreibt fest nach
  `…/worktrees/planung-von-3-c56c81`, einem Worktree, den es nicht mehr gibt.
  Aufgenommen wurde aus einer Kopie im Scratchpad, in der `WT` auf diesen
  Worktree zeigt. Die Datei im Baum bleibt, wie sie ist; sie geht mit PR C.
- **4c-1, 3a, 3b:** Die Prüffälle heißen `a-<datum>-5bd8`. Ihr Ordner zieht
  bei der Neuaufnahme auf das heutige Datum um. Nach dem Maskieren ist er
  gleich.
- **1a bis 2c:** Es gibt kein Aufnahmeskript, also trägt der Beleg aus
  Abschnitt 1 allein.

**Ergebnis:** Die Aufnahmen sind reproduzierbar. Kein Befund der Klasse (c),
nichts dem Nutzer zur Entscheidung vorzulegen.
