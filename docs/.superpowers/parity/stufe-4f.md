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
