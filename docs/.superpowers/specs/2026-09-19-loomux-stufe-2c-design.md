# Stufe 2c: `stop`, `subagent-start`, `subagent-stop`, Antigravity-Adapter

**Stand:** 2026-09-22, umgesetzt für Claude Code, auch der Eintrag in der
eingecheckten `.claude/settings.json` (Task 16). Offen ist der
Antigravity-Adapter (Tasks 14 und 15 des Plans). Was bei Planung und Umsetzung
vom Entwurf abwich, steht unter „Nachträge“ am Ende; wo Text und Nachtrag sich
widersprechen, gilt der Nachtrag. Rahmen:
`2026-09-14-loomux-fusion-design.md`, Stufe 2, Teilstufe 2c; Vorgänger
`2026-09-19-loomux-stufe-2a-design.md`. Quelle ist ultraloom am Tag
`loomux-1a-source` (`9d01a60`): `src/ultraloom/hooks/stop.py`,
`subagent_start.py`, `subagent_stop.py`, `state.py`.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Antigravity | Die Messung ist Teil von 2c, als **Halt** unmittelbar vor dem Adapter. Was sie nicht belegt, bleibt Naht mit `ErrNoAdapter` und steht in der Abweichungsliste |
| Was `stop` prüft | Eingebautes Profil **`stop`**, Vorgabe gleich `precommit`, überschreibbar in `[verify.profiles]`. Die Basis rückt nach jedem grünen Lauf des Profils vor |
| Befund der Subagenten | Geparkt in einer Datei je Agent, zugestellt vom **`stop` des Hauptagenten** mit einem Halt, der nicht zählt |
| `wiki-gate` am Rundenende | Als **Lane** `lint/wiki` im Check-Scope, nicht als eigener Stop-Eintrag |
| Blockzähler | **3 in Folge**; ein grüner Lauf setzt ihn zurück |
| Parität | Kleiner Aufzeichnungsdurchgang für das gleichbleibende Verhalten, dazu jede Absicht der Python-Tests als Go-Test oder Eintrag der Abweichungsliste |
| Lokale Branches | Gehören in den Snapshot der Subagenten |
| Fingerabdruck | Der Inhaltsbaum über einen temporären Index (`git write-tree`), nicht HEAD + Diff + untracked |
| Marker | `.loomux/no-verify` statt `.claude/.no-verify`, hostneutral; die eingebaute Policy-Regel zieht mit |
| Remote | Nur `origin`, wie in Python |
| Ausgabe | Ein Halt ist Exit 2 mit Text auf stderr, für beide Hosts; kein JSON `decision: "block"` |

## Befunde, die den Entwurf formen

Gelesen und gemessen am 2026-09-19.

- **Es gibt keine aufgezeichneten Fälle** für `stop` und `subagent-*`, weder
  in ultraloom noch in loomux. Es gibt die Python-Tests `tests/hooks/test_stop.py`
  (26), `test_subagent_start.py` (4), `test_subagent_stop.py` (16).
- **Der Python-Bericht von `subagent-stop` erreicht niemanden.** Stdout mit
  Exit 0 landet bei `SubagentStop` nur im Debug-Log von Claude Code (Hooks-Referenz,
  Abschnitt „Exit code 0“; Ausnahmen sind nur `UserPromptSubmit`,
  `UserPromptExpansion`, `SessionStart`, `PostModelSwitch`). `additionalContext`
  und Exit 2 bei `SubagentStop` gehen an den **Subagenten** und lassen ihn
  weiterlaufen (Changelog v2.1.163, Issue anthropics/claude-code#65495). Der
  richtige Empfänger ist der Hauptagent.
- **Claude Code gibt einen Stop-Hook selbst auf**, nach 8 Blockaden in Folge
  ohne Fortschritt (`CLAUDE_CODE_STOP_HOOK_BLOCK_CAP`). Die Vorgabefrist für
  Command-Hooks ist 600 s; die 300 s aus ultraloom waren eine Wahl in den
  Settings, kein Host-Limit.
- **Hooks eines Ereignisses laufen parallel.** Zwei Subagenten, die in einer
  Nachricht starten, schreiben gleichzeitig. Python hielt alle Snapshots in der
  einen Sitzungsdatei und verlor dabei einen.
- **Antigravity ist ungemessen.** Nie wurde eine agy-Nutzlast für `Stop` oder
  `PreInvocation` mitgeschnitten; im Print-Modus feuern beide nicht
  (`specs-ul/2026-09-10-antigravity-hook-messung.md`). Einen `SubagentStop` hat
  agy nicht. Die Aussage „sofort baubar“ der Fusions-Spec gilt nur für den
  Claude-Teil.
- **Kosten, gemessen auf diesem Repo** (7.322 Dateien, warm, Windows 11):

  | Was | Zeit |
  |---|---|
  | `loomux check precommit` (alles grün) | 21,4–22,0 s |
  | `loomux check lint` | 0,6 s |
  | `loomux wiki-gate --root .` | 0,13 s |
  | Fingerabdruck: Index kopieren, `git add -A`, `git rm --cached -r .loomux/state`, `git write-tree` | 224–245 ms |
  | zum Vergleich `git diff --binary HEAD` + `git ls-files -o --exclude-standard`, ohne Inhalte zu hashen | 353 ms |
  | `git ls-remote origin` (24 Refs) | 1,30–1,45 s |

  Der Baum aus dem temporären Index war auf sauberem Stand gleich `HEAD^{tree}`.
  Eine Pfadangabe `':!.loomux/state'` bei `git add` scheitert, sobald der Pfad
  ignoriert ist („paths are ignored“); deshalb `git rm --cached` danach.
- **Code heute:** `hook.go` kennt drei Ereignisse, `hook_test.go:72` prüft, dass
  `stop` unbekannt ist. `hosts.Payload` trägt nur `Event` und `SessionID`.
  `writeClaudeContext` schreibt fest `SessionStart`, `verify.WriteEdit` fest
  `PostToolUse`. `SessionState` hat `Blocks`, `Snapshots`, `Base`; nur `Base`
  wird gelesen. Die Antigravity-Stubs liegen in `internal/hosts/codex.go`.
  `internal/hooks/status.go:151-174` empfiehlt `loomux wiki-gate` als
  Stop-Eintrag und prüft nur Pre- und PostToolUse.

## Pakete

- **`internal/cli/hook.go`**: kennt zusätzlich `stop`, `subagent-start`,
  `subagent-stop`. Ein fehlerhafter Aufruf ist bei allen dreien Exit 1 (die
  Runde endet). `--budget` gilt auch für `stop`, Vorgabe **270 s**
  (`hooks.DefaultStopBudget`).
- **`internal/hosts`**:
  - `Payload` bekommt `AgentID` und `AgentType`. `stop_hook_active` wird nicht
    gelesen; der Zähler ist die eine Quelle (wie `stop.py:111-117`).
  - `WriteContext(host, event, w, lines)` nimmt das Ereignis; `hookEventName`
    ist nicht mehr fest.
  - Antigravity zieht in `antigravity.go`. Codex bleibt Naht.
- **`internal/hooks`**:
  - `stop.go`: der Ablauf unten.
  - `subagent.go`: Snapshot, Vergleich, Befund.
  - `wikigate.go`: die Check-Lane `lint/wiki`, die `wiki.GateReport` im
    Prozess ruft und die Ausgabe über `Job.Fn` einfängt, wie die Edit-Lane in
    `post_edit.go:182`. `stop` und `internal/cli/check.go` rufen dieselbe
    Funktion; so prüft auch `loomux check lint` das Wiki.
  - `status.go`: empfiehlt `hook stop` statt `wiki-gate` und prüft alle
    fünf Einträge.
- **`internal/sessions`**:
  - Sitzungsdatei `.loomux/state/hooks/<session>.json` mit `base`, `blocks`,
    `green`. `snapshots` entfällt. Nur `stop` schreibt `blocks` und `green`,
    und `Stop` hat genau einen Eintrag, also keinen Wettlauf.
  - Agent-Dateien `.loomux/state/hooks/<session>/agents/<agent_id>.json`,
    eine je Subagent, **atomar** geschrieben (Temp-Datei im selben Verzeichnis,
    dann Rename). `agent_id` durchläuft `safeName`.
  - `ReadState` verträgt fehlendes `green` wie heute fehlendes `base`.
  - `Forget` entfernt auch `<session>/` mit `RemoveAll`. `Others` überspringt
    Verzeichnisse schon (`sessions.go:48`).
  - Die Paketdoku („die Python-Seite schreibt“) wird berichtigt.
- **`internal/verify`**: eingebautes Profil `stop` = `precommit`; ein Profil
  darf `stop` heißen, es ist kein reservierter Name. Der Stack `wiki` hat im
  Check-Scope die Lane `lint`, die `hooks` als `Fn`-Job einsetzt.
- **`internal/gitwork`**: `ContentTree(root)` (Fingerabdruck), `LsRemote(root,
  remote)` über `child` mit Frist 10 s, `LocalHeads(root)`
  (`git for-each-ref refs/heads`), `LogOneline(root, from, to)`.
- **`internal/hooks/guard.go`** (Policy): Die eingebaute Regel für
  `.claude/.no-verify` (`guard.go:41`) wird zu `.loomux/no-verify`; die Regel
  für `.loomux/state/hooks/**` (`guard.go:44`) deckt die Agent-Dateien schon ab.

**Abhängigkeiten:** `hooks → verify, sessions, gitwork, hosts, brain/wiki`.
`verify` importiert weiter nichts aus `hooks`.

## Ablauf von `stop`

1. **Nutzlast lesen.** Kein JSON oder keine `session_id` → Exit 1. Keine
   gemeinsame Ersatzdatei (`stop.py:102-109`).
2. **Befunde einsammeln.** Jede Agent-Datei der Sitzung mit `finding` wird
   gelesen; ihr Text ist für stderr vorgemerkt. Dateien mit `snapshot` bleiben
   liegen: Der Subagent läuft noch. Gelesene Befunddateien werden gelöscht,
   nachdem der Text geschrieben ist. Befunde kommen **vor** dem Marker: Was ein
   Subagent an Remote und Branches getan hat, soll der Hauptagent auch ohne
   Prüfung sehen.
3. **Marker.** Existiert `.loomux/no-verify`, entfällt die Kette. Exit 0, mit
   Befunden Exit 2. Ein Fehler beim Nachsehen zählt als „kein Marker“.
4. **Zähler.** Bei `blocks ≥ 3`: auf stderr
   `gave up after 3 consecutive blocks; base stays at <sha>. Fix the lanes or set .loomux/no-verify.`
   `blocks` wird 0, die Kette läuft nicht. Exit 0, mit Befunden Exit 2.
5. **Inhaltsstand.**
   - `tree` = `gitwork.ContentTree`: das echte Index in eine Temp-Datei im
     Zustandsverzeichnis kopiert, `GIT_INDEX_FILE` darauf, `git add -A`,
     `git rm -r -q --cached --ignore-unmatch .loomux/state`, `git write-tree`.
     Ignorierte Dateien fehlen von selbst, auch `bin/loomux.exe`.
   - **Kein Git-Repo:** kein `tree`, keine Basis, keine Warnung; die Kette
     läuft jedes Mal. Dieselbe Regel wie bei `session-start`.
   - **Unborn HEAD:** Die Basis ist der leere Baum.
   - **Kein `base` im Zustand:** Basis ist HEAD, mit der Warnung auf stderr,
     dass Committetes dieser Sitzung unsichtbar bleibt.
   - **Ein Git-Befehl scheitert in einem Repo:** Exit 2, die Blockade zählt.
6. **Nichts Neues.** `tree == green` oder `tree == base^{tree}` → Exit 0, mit
   Befunden Exit 2. Die Kette läuft nicht.
7. **Konfiguration.** `ReadConfig`, `LoadPresets`, `Resolve`,
   `ExpandProfile(cfg, "stop")`. Ein Ladefehler ist Exit 1.
8. **Kette** mit `verify.Plan`/`Run`, Check-Scope, Budget aus `--budget`.
   `missing-tool`, `unready` und `timed-out` sind rot, wie bei `check`.
   - **Alles grün:** `base` = HEAD, `green` = `tree`, `blocks` = 0. Exit 0,
     mit Befunden Exit 2.
   - **Eine Lane rot:** `blocks` + 1, Exit 2.
   - **Keine rot, eine `budget`:**
     `not everything was verified; raise --budget or shrink the stop profile`,
     Exit 1. `base`, `green`, `blocks` bleiben.
   - **Nichts lief** (nur `unavailable`, `not-applicable`):
     `nothing was verified`, Exit 1. `base`, `green`, `blocks` bleiben.
   - Die Cover-Dateien behandelt `stop` wie `check`: `PrepareCover` vor,
     `CleanCover` nach dem Lauf.
9. **Ausgabe auf stderr:** erst die Befunde, dann nur die **roten** Lanes mit
   ihrer Ausgabe im Format von `check`. Grüne Kopfzeilen gehören nicht in den
   Kontext des Agenten.

**Zählt als Blockade:** rote Kette, Git-Fehler. **Zählt nicht:** ein Halt nur
wegen Befunden.

`green` ist inhaltsadressiert. Ein Commit ohne Inhaltsänderung löst keinen
Lauf aus, und ein Resume, bei dem `session-start` `base` neu schreibt, macht
`green` nicht falsch.

## `subagent-start` und `subagent-stop`

**Snapshot:**

```json
{"snapshot": {"head": "<sha|\"\">", "heads": {"refs/heads/x": "<sha>"},
              "refs": {"refs/heads/master": "<sha>", "refs/tags/v1^{}": "<sha>"},
              "remote": "ok|unavailable"}}
```

- `refs` aus `git ls-remote origin` über `child`, Frist 10 s, mit
  `GIT_TERMINAL_PROMPT=0` und `GIT_SSH_COMMAND="ssh -o BatchMode=yes"`; ohne
  TTY hinge ein Credential-Prompt sonst bis zur Frist. Der Plan prüft, dass
  `gitenv` diese Variablen nicht entfernt. `^{}`-Zeilen annotierter Tags sind
  eigene Refs.
- `heads` aus `git for-each-ref refs/heads`, `head` aus `git rev-parse HEAD`
  (unborn: leer).
- Kein `origin`, Frist abgelaufen oder kein Git: `remote: "unavailable"`.

**`subagent-start`:** Fehlt `session_id` oder `agent_id` → Exit 1. Sonst den
Snapshot in die Agent-Datei, Exit 0.

**`subagent-stop`:** Gleiche Prüfung. Keine Agent-Datei → Exit 0, still. Sonst
neuer Snapshot, Vergleich, je Befund eine Zeile, eingeleitet mit
`subagent <agent_id>: ` (Wortlaut von `subagent_stop.py`):

- `origin <ref> is new at <h>` · `origin <ref> is gone; it was <h>` ·
  `origin <ref> moved <a> -> <b>`
- `branch <name> is new at <h>` · `branch <name> is gone; it was <h>` ·
  `branch <name> moved <a> -> <b>`
- `new commit <oneline>` je Commit aus `git log --oneline <alt>..<neu>`, für
  `HEAD` und für jeden bewegten lokalen Branch, ohne Doppelte.
- Ist eine Seite `unavailable`: keine `origin`-Zeilen, nur
  `remote could not be read at start` bzw. `… at stop`.

Mit Befund wird die Agent-Datei zu `{"finding": ["…"]}`, ohne Befund wird sie
gelöscht. Exit immer 0: Der Empfänger ist der Hauptagent.

**Bekannte Grenze:** Ein Befund, der nach dem letzten `Stop` der Sitzung
entsteht, wird nie zugestellt; `Forget` räumt ihn weg.

## Antigravity

Geplant nach `specs-ul/2026-09-10-go-hooks-drei-hosts-design.md`, jede Zeile
unter dem Vorbehalt des Halts vor dem Adapter:

| loomux | agy-Ereignis | zu messen |
|---|---|---|
| `stop` | `Stop`, flache Liste ohne `{matcher, hooks}` | Hält Exit 2 die Runde an? Liest agy stdout-JSON? Eigene Frist? |
| `subagent-start` | `PreToolUse`, Matcher `invoke_subagent` | Name des Matchers; was als `agent_id` dient |
| `subagent-stop` | `PostToolUse`, Matcher `invoke_subagent` | Dieselbe ID wie beim Start? |
| `session-start` | `PreInvocation` + Marker | Je Runde oder je Sitzung? Kann es Kontext einspeisen? |

Die Messung läuft in einer interaktiven agy-Sitzung (nicht `agy -p`) in einem
vertrauten Ordner, mit einem Rekorder-Hook in `.agents/hooks.json`, der jede
Nutzlast in eine Datei schreibt. Der Adapter übersetzt in dasselbe
`hosts.Payload`; die Hooks bleiben hostblind. **Rückfall, schon jetzt
festgelegt:** Arbeitsverzeichnis `.agents/` (`hosts.FindRoot`), Exit 2 als
einziges Signal; was nicht belegt ist, bleibt `ErrNoAdapter`. Die Ergebnisse
gehen als Nachträge in diese Spec.

## Messung auf der Claude-Seite

Vor `subagent-*`: Ein Rekorder-Hook in `.claude/settings.local.json` für
`Stop`, `SubagentStart`, `SubagentStop` schreibt die Nutzlasten eines
Subagentenlaufs mit. Zu belegen: `SubagentStart` und `SubagentStop` tragen die
`session_id` des Hauptagenten, beide tragen `agent_id` (gleich) und
`agent_type`. Stimmt eins nicht, fände `subagent-stop` nie seinen Snapshot und
die Funktion wäre still tot; dann ändert ein Nachtrag den Schlüssel der
Agent-Datei. Die drei Nutzlasten werden Testdaten unter
`testdata/cases/2c-payloads/`.

## Parität

### Aufzeichnen

- Aus ultraloom am Tag `loomux-1a-source` mit `loomux dev record-case`; die
  Prüfkette ersetzt `faketool` wie in 2a, Git ist echt.
- Welten unter `testdata/cases/2c-worlds/`, je eine `git.toml` (siehe
  „Infrastruktur“ unten); das Remote liegt in der Welt, weil sie beim
  Abspielen in ein Temp-Verzeichnis kopiert wird.
- Fälle `stop`: grün, rot, ohne Basis, Marker, kaputte Nutzlast, nichts
  geändert. Fälle `subagent-start`/`-stop`: ohne `agent_id`, ohne Snapshot,
  Ref bewegt, Ref neu, Ref weg, neuer Commit.

### Übersetzen (`2c-map.toml`)

| Alt | Neu |
|---|---|
| `ultraloom hook stop\|subagent-start\|subagent-stop` | `loomux hook … --host claude --root {{WORLD}}` |
| `.ultraloom/hooks/<id>.json` `base`, `blocks` | `.loomux/state/hooks/<id>.json` |
| `snapshots[<agent>]` (roher `ls-remote`-Text + `HEAD`-Zeile) | `<id>/agents/<agent>.json`, geparst in `refs` und `head`; `heads` leer |
| `.claude/.no-verify` | `.loomux/no-verify` |
| `[verify]` der Welt | Faltung aus 2a: `[verify].<art>` → `[verify.project].<art>`, erkannte Stacks `<art> = false` |

### Infrastruktur, die die Aufzeichnung erst möglich macht

Berichtigt am 2026-09-19 vor dem Plan: Die Maschinerie aus 2a trägt diese
Fälle nicht. Gegen den Code gelesen:

- `recordcase.Record` setzt nur `cmd.Stdout`; stderr geht verloren, und
  `stop.py` schreibt alles auf stderr.
- `cases.StageWorld` kopiert Dateien. Es gibt keine Git-Welten, und ein
  `world/.git/` lässt sich im loomux-Repo nicht committen.
- `world_after` vergleicht Byte für Byte. Python und Go schreiben die
  Zustandsdatei in verschiedenen Bytes (`internal/sessions/state.go:34-39`),
  und ein Git-Index, den `git diff` auffrischt, macht jede Welt „geändert“.
- `RunCase` vergleicht nur stdout und kennt weder Sitzung noch Agent.

Deshalb vier Bausteine vor der ersten Aufzeichnung:

1. **Git-Welten.** Eine Welt mit `git.toml` wird nach `StageWorld` zu einem
   Repo, in `Record` wie in `RunCase` (`cases.BuildGitWorld`). Die Datei
   nennt Commits mit ihren Dateien, optional ein Bare-Remote unter
   `.origin.git` mit `url = ./.origin.git` und was dorthin gepusht wird, und
   eine Liste lokaler Branches. Deterministisch: feste Identität, feste
   Datumswerte je Commit, `core.autocrlf=false`, `commit.gpgsign=false`; die
   SHAs sind unter Windows und Linux gleich. Nach dem Aufbau ersetzt es in
   allen übrigen Dateien der Welt `{{COMMIT:<n>}}` durch den SHA des n-ten
   Commits. Git läuft dafür nicht über `child`, weil `gitenv` die
   Identitätsvariablen streicht, die hier gesetzt werden müssen. `.git/**`
   und `.origin.git/**` fallen aus `world_after` und aus `CompareTrees`.
   Die Dateien des Prüfstands selbst (`git.toml`, `faketool.json`,
   `.origin.git/`, `.ultraloom/`, `.loomux/state/`) schreibt der Aufbau nach
   `.git/info/exclude`; sonst wären sie für `stop` eine Änderung.
2. **stderr aufnehmen.** Der Rekorder schreibt stderr in eine Datei
   `stderr` des Falls, als Beleg für die Abweichungsliste; verglichen wird
   sie nicht.
3. **Zustand falten.** `dev import-cases` übersetzt
   `.ultraloom/hooks/<id>.json` in `world` und `world_after` in das Layout
   von loomux: `base` und `blocks` in `.loomux/state/hooks/<id>.json`, jeder
   Eintrag aus `snapshots` in `<id>/agents/<agent>.json` mit geparsten
   `refs` und `head`, geschrieben über `internal/sessions`, also in Go-Bytes.
4. **Vergleichsklassen** in `LoadCase`/`RunCase` (siehe unten).

### Vergleichsklassen

- **`state`** (neu), für `stop`: Exit-Code, dazu `base` und `blocks` der
  Sitzungsdatei in `world_after` gegen die nach dem Lauf, beide über
  `sessions.ReadState` gelesen. `green` und alle übrigen Dateien zählen nicht.
  So ist das Urteil gepinnt: grün heißt, `base` ist gewandert; rot heißt,
  `blocks` ist gestiegen.
- **`finding`** (neu), für `subagent-stop`: Exit-Code, dazu die stdout-Zeilen
  von Python gegen die `finding`-Zeilen aller Agent-Dateien unter
  `.loomux/state/hooks/*/agents/` nach dem Lauf, per Glob gefunden, sortiert.
  Ohne Agent-Datei gilt die leere Liste.
- **`message`** (aus 1a), für `subagent-start` und die Fehlerfälle: nur der
  Exit-Code.
- **Meldungstexte** (`gave up …`, `no base …`, `nothing was verified`) werden
  nicht verglichen. Die aufgenommene stderr-Datei belegt den alten Wortlaut
  in der Abweichungsliste.

Jede der 46 Absichten der Python-Tests wird ein Go-Test oder ein Eintrag der
Abweichungsliste.

## Abweichungsliste `parity/stufe-2c.md`

1. Ein Git-Fehler in einem Repo ist Exit 2 und zählt; Python endete mit 1,
   obwohl der Kommentar in `stop.py:141-146` das Gegenteil wollte.
2. Der Zähler zählt Blockaden in Folge; grün setzt zurück. Python zählte je
   Sitzung, das Tor war nach drei roten Rundenenden für den Rest der Sitzung aus.
3. Profil `stop` statt `--checks`; die Basis rückt nach jedem grünen Lauf des
   Profils vor. Unter `--checks` rückte sie nie vor.
4. Unveränderter Inhalt (`green`, `base^{tree}`) löst keinen Lauf aus.
5. Kein Git-Repo: Die Kette läuft; Python endete an `WorktreeError`.
6. Unborn HEAD: leerer Baum als Basis; Python behielt still die alte Basis.
7. Aufgebrauchtes Budget ist Exit 1 ohne Urteil.
8. Befunde der Subagenten über `stop` statt über stdout.
9. `subagent-stop` ohne Snapshot ist still; Python schrieb eine Zeile, die
   ohnehin niemand sah.
10. Ein nicht lesbarer Remote wird auf beiden Seiten gleich behandelt; Python
    meldete danach jede Ref als neu oder weg.
11. Lokale Branches im Snapshot.
12. Eine Datei je Agent statt `snapshots`; aufgeräumt nach dem Vergleich und
    von `Forget`.
13. Marker `.loomux/no-verify` statt `.claude/.no-verify`.
14. `wiki-gate` als Lane statt als eigener Stop-Eintrag.
15. Ausgabe von `stop`: Lane-Zeilen, nur rote, statt `kind: output`;
    Meldungstexte neu formuliert.
16. Eine neue, nicht ignorierte Datei ist eine Änderung. Python fragte
    `git diff --name-only <base>` und sah untracked Dateien nicht; eine Runde,
    die nur neue Dateien anlegte, lief ungeprüft durch.
17. Was die Antigravity-Messung ergibt.

## Selbstnutzung

**Halt:** Der Mensch trägt in `.claude/settings.json` ein:

| Ereignis | Befehl | Frist |
|---|---|---|
| `Stop` | `hook stop --host claude --root … --budget 270s` | 300 |
| `SubagentStart` | `hook subagent-start --host claude --root …` | 30 |
| `SubagentStop` | `hook subagent-stop --host claude --root …` | 30 |

Die 300 s sind eine Wahl: Bei 22 s für `precommit` reicht das weit, und ein
hängendes Tor soll eine Runde nicht zehn Minuten halten. Danach prüft loomux
jedes eigene Rundenende mit dem Profil `stop` = `precommit`.

## Messen

In `docs/en/benchmarks.md` und `docs/de/benchmarks.md`, kalt und warm:

- `stop` ohne neuen Inhalt (Ziel etwa 300 ms, bestimmt vom Fingerabdruck),
- `stop` mit Fake-Runner als Eigenzeit,
- `subagent-start` und `subagent-stop` (bestimmt von `ls-remote`),
- `ContentTree` allein.

Vor der Mutationsrunde wird ihre Laufzeit geschätzt: Jeder Mutant in `stop`
und `subagent` startet echtes Git.

## Fertig, wenn

Die fünf Kriterien der Fusions-Spec, dazu:

- `docs/en/hooks.md` und `docs/de/hooks.md`: Phase 3 des Diagramms, Tabelle
  in §8, der Satz über unbekannte Ereignisse; `cli-reference.md:179`;
  `migration.md` (en, de) nach `AGENTS.md`; READMEs.
- Fusions-Spec: Zeile 2c auf ✅, der Vorbehalt „sofort baubar“ korrigiert.
- Release `release:minor`, `feat(hooks)`. Vor dem ersten Push `release-pr`.

## Reihenfolge für den Plan

0. **Halt:** Messung auf der Claude-Seite (Rekorder, ein Subagent).
1. Infrastruktur: Git-Welten, stderr im Rekorder, Zustand falten,
   Vergleichsklassen `state` und `finding`; dann Aufzeichnung, Welten,
   `2c-map.toml`.
2. `sessions`: `green`, Agent-Dateien, atomares Schreiben, `Forget`.
3. `gitwork`: `ContentTree`, `LsRemote`, `LocalHeads`, `LogOneline`.
4. `hosts`: `AgentID`, `AgentType`, Ereignis in `WriteContext`.
5. `verify` und `hooks/wikigate.go`: Profil `stop`, Lane `lint/wiki` im
   Check-Scope, `cli/check`.
6. `stop`.
7. `subagent-start`, `subagent-stop`.
8. Verteiler in `cli/hook`, Policy-Regel des Markers umziehen, `status.go`.
9. **Halt:** Antigravity-Messung.
10. Antigravity-Adapter nach der Messung.
11. **Halt:** Der Mensch trägt die Settings ein.
12. Selbstnutzung, Vergleichsklassen, Abweichungsliste.
13. Mutanten.
14. Messen.
15. Doku.

## Nachträge

Eingetragen am 2026-09-22. 1–10 hat der Plan beim Rechnen gegen den Code
gefunden (Abschnitt „Nachträge an die Spec“ in
`plans/2026-09-19-loomux-stufe-2c.md`), 11–20 hat die Umsetzung entschieden
oder gemessen, 21 nennt, was offen ist; ab 22 steht, was die Durchsicht des
ganzen Zweigs vor dem Pull-Request berichtigt hat. Maßgeblich ist der Code in
`internal/hooks/stop.go`, `subagent.go`, `wikigate.go`, `internal/sessions`
und `internal/gitwork`; jede Abweichung gegen Python steht mit Begründung in
`parity/stufe-2c.md`.

1. **Kein `GIT_SSH_COMMAND`.** `ls-remote` bekommt nur
   `GIT_TERMINAL_PROMPT=0` (`internal/gitwork/remote.go`). Die Variable
   schlüge `core.sshCommand` und die SSH-Einstellung des Nutzers; ein SSH ohne
   TTY fragt nicht, sondern scheitert, und die Frist von 10 s fängt den Rest.
   `gitenv` streicht `GIT_TERMINAL_PROMPT` nicht. Der Abschnitt „Snapshot“
   oben gilt insoweit nicht.
2. **Remote-`HEAD` bleibt in `refs`.** Wie in Python: ein bewegter
   Standard-Branch meldet zwei Zeilen, `origin HEAD moved …` und
   `origin refs/heads/master moved …`.
3. **`heads` fehlt in übersetzten Snapshots.** Ein Snapshot mit
   `"heads": null` vergleicht keine Branches, einer mit `{}` schon. Go schreibt
   nie `null`: „hatte keine Branches“ und „wusste nichts von Branches“ bleiben
   unterscheidbar.
4. **Ein ignoriertes Wurzelverzeichnis** (`gitwork.ErrIgnoredRoot`) zählt für
   `stop` wie „kein Repo“: kein Fingerabdruck, die Kette läuft jedes Mal. Das
   Tor sagt dazu nichts; der Text von `ErrIgnoredRoot` ist dort unerreichbar.
5. **`lint/wiki` steht am Ende.** `cli/check.go` und `stop` hängen die
   Wiki-Lane hinter die geplanten Jobs, weil `After` Indizes sind. Die
   Reihenfolgeregel der 2a-Spec („Arten in Anfrage-Reihenfolge“) gilt für
   `lint/wiki` nicht.
6. **„Nichts geprüft“ heißt `CheckVerdict` mit Notizen.** Eine angefragte Art
   ohne Lane, die lief, ist kein Grün: Exit 1 mit den Notizen und
   `nothing was verified for these kinds; the base stays`. Das ersetzt den
   Arm „nur `unavailable`, `not-applicable`“ aus Schritt 8.
7. **Befunde ohne Präfix in der Datei.** Die Agent-Datei trägt die Zeilen ohne
   `subagent <id>: `; `stop` setzt das Präfix beim Zustellen, die
   Vergleichsklasse `finding` beim Vergleich.
8. **Eine Basis, die es nicht mehr gibt, zählt wie keine.** Nach `--amend`
   oder Rebase und einem `gc` löst `TreeOf(base)` nicht mehr auf: Warnung
   `base <sha> is gone; measuring from HEAD`, HEADs Baum als Basis, ein
   grüner Lauf setzt die Basis neu. Kein Git-Fehler, der hält und zählt.
9. **Mit Befunden wird auch Exit 1 zu Exit 2.** Ein Ladefehler, ein
   Planfehler oder ein aufgebrauchtes Budget halten die Runde an, wenn im
   selben Aufruf Befunde zugestellt wurden: die Dateien sind danach weg, und
   nur ein Halt sorgt dafür, dass der Hauptagent sie liest. Das zählt nicht als
   Blockade. Ausnahme ist die Runde, in der der Zähler aufgibt: sie endet mit 0
   und räumt die Befunde darum nicht weg (Nachtrag 23).
10. **Die Welten sind überall dieselben Bytes.** `.gitattributes` setzt
    `* text=auto eol=lf`; die SHAs der Git-Welten sind auf jeder Maschine
    gleich. Der Linux-Lauf in `ci.yml` fährt die Fallsuiten heute nicht.
11. **Abweichung 16 war falsch und ist gestrichen.** Pythons
    `worktree.changed_since` vereinigt `git diff --name-only <base>` mit
    `git status --porcelain -uall` (`src/ultraloom/worktree.py:129`, `:131`,
    `:134`); eine neue, nicht ignorierte Datei war auf beiden Seiten eine
    Änderung. Die Aufzeichnung hat es gemessen, die Wiedergabe bestätigt es
    (`hook-stop/untracked-only` besteht). Die Abweichungsliste zählt deshalb
    ab dort um eins herunter; die Zeile 17 oben ist dort Eintrag 16.
12. **Die Wiki-Lane prüft nur das Bündel.** `lint/wiki` ruft
    `wiki.LintBundle` und zählt die Befunde mit `Severity == check.Error`,
    dieselben, die `GateReport` als `wiki-lint:*` meldet — nicht
    `wiki.GateReport` selbst, wie „Pakete“ oben sagt. Dessen Drift-Regel
    (Code geändert, Doku nicht) hätte jeden reinen Code-Commit und damit jedes
    Rundenende von loomux gehalten; ultraloom hatte sie nur unter
    `[wiki] mode = "brain"`. `loomux wiki-gate` behält sie. Kosten, wenn
    falsch: Drift wird nirgends mehr automatisch erzwungen.
13. **Ein festsitzender Befund zählt als Blockade.** Lässt sich die Datei eines
    Befunds nach dem Zustellen nicht entfernen, käme er an jedem Rundenende
    wieder und hielte jedes an; der Marker hilft nicht, weil er Befunde nicht
    überspringt. Also zählt er, die Aufgeben-Regel begrenzt die Reihe — seit
    Nachtrag 23 wie bei einer roten Kette: drei Runden angehalten, die vierte
    durchgelassen —, und solange er feststeckt, setzt auch eine grüne Kette den
    Zähler nicht zurück. „Zählt nicht: ein Halt nur wegen Befunden“ gilt nur für
    Befunde, die weggeräumt sind.
14. **Ein Befund wird nur ergänzt, nie fallengelassen.** `subagent-start`
    schreibt den neuen Snapshot neben einen geparkten Befund, statt die Datei
    zu ersetzen; `subagent-stop` hängt seine Zeilen hinten an, ältester Lauf
    zuerst, ohne Entdoppelung; `stop` räumt genau die zugestellten Zeilen weg,
    entschieden gegen ein erneutes Lesen unmittelbar vor dem Schreiben, und
    lässt eine Datei mit Snapshot stehen. Grund: ein Controller, der einen
    Subagenten unter derselben ID fortsetzt, darf nicht verlieren, was dessen
    letzter Lauf an `origin` getan hat. Ein Schreiben genau zwischen erneutem
    Lesen und Umbenennen verliert eine Zeile; eine Sperre baut diese Stufe
    nicht. Das ersetzt „Mit Befund wird die Agent-Datei zu `{"finding": …}`“
    und „Dateien mit `snapshot` bleiben liegen“ aus den Abschnitten oben.
15. **Die Claude-Nutzlasten sind in einer laufenden Sitzung gemessen**, nicht
    in einer frischen, wie „Messung auf der Claude-Seite“ vorsah: der Rekorder
    stand in `.claude/settings.local.json`, und Claude Code 2.1.276 liest die
    Datei mitten in der Sitzung. Alle drei Annahmen halten — dieselbe
    `agent_id` bei Start und Stop, die `session_id` des Hauptagenten,
    `agent_type` — und die Nutzlasten liegen unter
    `testdata/cases/2c-payloads/`, die `Stop`-Nutzlast vom Rundenende
    derselben Sitzung.
16. **Die Infrastruktur ist gebaut wie oben berichtigt:** ein Aufbau der
    Git-Welten aus `git.toml` (`cases.BuildGitWorld`) und die zwei
    Vergleichsklassen `state` und `finding`. Dazu kam eine Falle, die erst die
    Fallsuite zeigte: `Case.Path` war relativ und wurde nach dem Wechsel ins
    Welt-Verzeichnis aufgelöst: ein `state`-Fall las auf der erwarteten Seite
    keine Sitzungsdatei und bestand, ohne ein Feld anzusehen, ein
    `message`-Fall mit `world_after` scheiterte laut in `CompareTrees`.
    `finding` war nie betroffen; seine Erwartung ist das beim Laden gelesene
    stdout. Behoben in `internal/cases` (`f02e2ae`), mit einem Test, der
    gegen den alten Code scheitert. Ergebnis: 15 Fälle über 15 Welten, 13
    bestehen, 2 sind genehmigte Abweichungen (`hook-stop/gave-up`,
    `hook-subagent-stop/no-snapshot`).
17. **Der kopierte Index liegt im Temp-Verzeichnis des Systems**
    (`os.TempDir()`), nicht im Zustandsverzeichnis, wie Schritt 5 sagt: ein
    Zustandsverzeichnis, das sich nicht schreiben lässt, darf kein Git-Fehler
    werden, der jede Runde hält.
18. **`status` prüft sechs Ereignisse**, nicht fünf: `PreToolUse`,
    `PostToolUse`, `SessionStart`, `Stop`, `SubagentStart`, `SubagentStop`,
    je `[OK]` oder `[INFO]`.
19. **Aufgeräumt wird seltener, als „Bekannte Grenze“ annimmt.**
    `sessions.Forget` ruft nur `loomux worktree unlink`, und nur in einem
    verknüpften Worktree mit `[worktree] mirror`. Anderswo bleiben Agent-Dateien
    liegen, die niemand zustellt. In diesem Repository läuft `Forget` nie:
    `[worktree] mirror` ist in `.loomux/config.toml` auskommentiert, und
    `WorktreeUnlink` kehrt vor `Forget` zurück, wenn nichts gespiegelt ist. Das
    betrifft die Sitzungsdateien und die `agents/`-Verzeichnisse gleichermaßen. Dazu, gesehen bei der Selbstnutzung am
    2026-09-22: ein Subagent, den der Wirt ohne `SubagentStop` beendet (ein
    Reviewer starb an einem Rate-Limit), lässt seinen Snapshot liegen.
20. **Gemessen und selbst benutzt.** Ein Rundenende ohne neuen Inhalt kostet
    auf diesem Repository 169,5 ms warm (Ziel etwa 300 ms), davon 107–115 ms
    der Fingerabdruck; `subagent-start` 101,4 ms gegen ein lokales Remote und
    1.038,7 ms gegen GitHub, fast ganz `ls-remote` (`docs/de/benchmarks.md`,
    Eintrag vom 2026-09-20). Am 2026-09-22 liefen die drei Hooks in der
    Sitzung, die diese Stufe baute, über `.claude/settings.local.json`: das
    Tor hielt eine Runde mit einem absichtlichen `vet`-Befund und dem
    zugestellten Befund eines Subagenten, und die nächste Runde lief grün und
    setzte `base` und `green` (`parity/stufe-2c.md`, „Selbstnutzung“). Offen
    aus diesem Lauf: eine rote Test-Lane schreibt ihre ganze Ausgabe in den
    Kontext des Agenten.
21. **Offen.** Antigravity (Tasks 14 und 15): nicht gemessen, kein Adapter;
    `--host antigravity` endet bei allen drei Hooks mit Exit 1 über
    `ErrNoAdapter`, und Eintrag 16 der Abweichungsliste ist leer. Bis das
    erledigt ist, steht 2c in der Fusions-Spec nicht auf ✅. Der Eintrag der
    drei Hooks in der eingecheckten `.claude/settings.json` (Task 16) ist am
    2026-09-22 gemacht, auf Anweisung des Menschen vom Controller
    geschrieben.
22. **`session-start` setzt die Basis nur, wenn die Sitzung noch keine hat.**
    Der Eintrag in `.claude/settings.json` hat keinen Matcher, also feuert
    `SessionStart` auch bei `resume`, `clear` und `compact`, mit derselben
    `session_id`. Rückte die Basis dort auf HEAD, läge alles, was die Sitzung
    seit dem letzten grünen Lauf committet hat, in der Basis: ein roter Commit,
    danach eine Kompaktierung, und der nächste `stop` fände den Baum gleich dem
    der Basis und ließe die Runde ungeprüft enden. `recordBase` schreibt
    deshalb nur in eine Sitzung ohne `base`; weiter bewegt sie allein ein
    grüner Lauf. Die Quelle tat es bei jedem Aufruf (Abweichung 25).
23. **Befunde gehen weder am Zähler noch am Marker verloren.** Die Reihenfolge
    ist Nutzlast, Befunde, Zähler, Marker, Baum, Konfiguration, Kette. Die
    Befunde werden zuerst geschrieben; gibt der Zähler auf, endet die Runde mit
    0, ohne sie wegzuräumen, und das nächste Rundenende — der Zähler wieder
    auf 0 — stellt sie erneut zu und hält an. Sonst werden sie nach dem Zähler
    weggeräumt und halten die Runde an, gleich was der Zähler sagt und auch
    mit gesetztem Marker: der Marker überspringt die Kette, nicht die Befunde.
    Vorher stand der Marker vor dem Aufgeben-Arm, und ein Halt für Befunde galt
    nur unter `blocks < 3`; mit `blocks` = 3 und Marker wurden Befunde bei
    jedem Rundenende geschrieben, gelöscht und mit Exit 0 verworfen. Weil der
    Aufgeben-Arm den Zähler liest, bevor ein Befund weggeräumt wird, hält eine
    Reihe festsitzender Befunde jetzt drei Runden an und lässt die vierte
    durch, wie eine rote Kette; vorher waren es zwei und die dritte.
24. **Branches anderer Worktrees gehören nicht zum Befund.** Lokale Branches
    teilen sich alle Worktrees eines Repos; eine parallele Sitzung, die ihren
    Branch bewegte, während ein Subagent lief, stand als dessen Arbeit im
    Befund (gesehen am 2026-09-22 mit `claude/planung-von-3-c56c81`, Commits
    `1b5b072` und `0978bb2` aus einem anderen Worktree).
    `gitwork.LocalBranches` liest deshalb zu jedem Branch `%(worktreepath)`
    und vergleicht ihn mit `rev-parse --show-toplevel`, nach
    `filepath.Clean` und unter Windows ohne Groß- und Kleinschreibung. Der
    Schnappschuss trägt die anderswo ausgecheckten Branches als `elsewhere`;
    ein Branch, der am Start oder am Stop anderswo ausgecheckt ist, fällt
    ganz aus dem Vergleich — keine `branch`-Zeile, kein Commit-Bereich.
    Nirgends ausgecheckte Branches und der Branch dieses Worktrees bleiben.
    Folge: ein Subagent mit `isolation: "worktree"` committet ohne Zeile; das
    Ergebnis des Agent-Werkzeugs nennt seinen Worktree. Für `origin` gibt es
    keinen solchen Filter, der Push einer anderen Sitzung liest sich wie der
    des Subagenten; Code und Doku sprechen darum von dem, was sich bewegt hat,
    während ein Subagent lief, nicht von dem, was er getan hat. Das
    Zeilenformat bleibt byteweise gleich.
