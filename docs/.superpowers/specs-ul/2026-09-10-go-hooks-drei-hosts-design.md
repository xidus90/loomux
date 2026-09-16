# Hooks in Go, für drei Hosts

**Datum:** 2026-09-10
**Stand:** entworfen, nicht umgesetzt

## Ziel

Kein Hook dieses Projekts läuft mehr über Python. Die vier Ereignisse, die
heute `uv run --project … ultraloom hook …` aufrufen, werden Unterbefehle von
`ulguard`, und sie funktionieren auf Claude Code, auf Antigravity (Gemini) und
— über eine benannte, noch leere Naht — auf Codex (ChatGPT).

Python verschwindet damit aus dem Hook-Pfad, nicht aus dem Projekt: der
Agent-Flow `flows/verify_until_green.py` und `model/*` hängen am
`claude-agent-sdk==0.2.143` und bleiben, ebenso der Einsprung
`[project.scripts] ultraloom` in `pyproject.toml`, den sie brauchen. Auch die
Coverage-Lane rechnet weiter mit `uv run coverage` als Kindprozess. „Go" heißt
hier: die Entscheidung liegt in Go. Nicht: auf der Maschine ist kein Python.

## Ausgangslage, vermessen

Vier von sechs Hook-Ereignissen laufen über Python (`.claude/settings.json`),
zwei über Go (`ulguard`). Der Abhängigkeitsschluss der vier:

| Python-Modul | LOC | Testzeilen |
|---|---|---|
| `hooks/` (payload, state, session_start, subagent_start, subagent_stop, stop, cli) | 604 | 576+ |
| `checks.py` | 844 | 1455 |
| `process.py` | 848 | 846 |
| `config.py` | 448 | 837 |
| `worktree.py` | 249 | 404 |
| `journal.py`, `gate.py`, `state.py` | 213 | 593 |
| `hooks/coverage-check.py` | 129 | 222 |

Rund 3.300 Zeilen Produktivcode und knapp 5.000 Zeilen Tests. Die Tests sind
der Kostentreiber, nicht der Code.

Drei Vermutungen wurden vor dem Entwurf widerlegt und sind hier festgehalten,
damit sie nicht wiederkehren:

- `internal/verify` ist **kein** Port des Runners, sondern 191 Zeilen Helfer
  (Coverage-Parsing, gofmt).
- `internal/coverage/check.go` portiert `hooks/coverage-check.py` **nicht**. Es
  beantwortet für `ulinit`, *ob* ein Projekt eine Schwelle erzwingt; das Skript
  *führt* die Lane aus. Ähnlicher Name, zwei Aufgaben.
- `cmd/guard/post_edit.go:45` startet Kinder als schlichtes
  `exec.Command("cmd","/c",…)` ohne eigene Zeitgrenze. Für ein 300-Sekunden-Gate
  mit hängenden Werkzeugketten ist das keine Grundlage; `internal/child` ist neu
  zu schreiben.

Ebenfalls vermessen: `internal/mirrorcfg` liest nur `[worktree]` und sagt im
Paketkommentar, jede andere Tabelle „belongs to the Python side". Dieser Satz
wird durch `internal/ulconfig` falsch und muss mit.

## Hostmatrix

Belegt aus `antigravity/0.24.0/docs/MIGRATION.md:133-142`: Antigravity feuert
`PreToolUse`, `PostToolUse`, `PreInvocation` (Claudes `UserPromptSubmit`) und
`Stop`. `SessionStart`, `SessionEnd`, `PreCompact`, `Notification` und
`SubagentStop` **fallen weg** — der Host hat sie nicht.

| ultraloom-Hook | Claude Code | Antigravity | Codex |
|---|---|---|---|
| Schreibschranke | `PreToolUse` | `PreToolUse` | Naht |
| Nach-Bearbeitung | `PostToolUse` | `PostToolUse` | Naht |
| `session-start` | `SessionStart` | `PreInvocation` + Sitzungsmarke | Naht |
| `subagent-start` | `SubagentStart` | `PreToolUse`, Matcher `invoke_subagent` | Naht |
| `subagent-stop` | `SubagentStop` | `PostToolUse`, Matcher `invoke_subagent` | Naht |
| `stop` (Gate) | `Stop` | `Stop`, **flache** Liste | Naht |

Die Subagent-Ereignisse sind auf Antigravity verlustfrei nachbaubar, weil
`invoke_subagent` dort ein Werkzeug ist: ein Matcher darauf in Pre- und
PostToolUse liefert Start und Ende. `session-start` hat keinen Auslöser; es
wird über `PreInvocation` plus einer Sitzungsmarke im Zustandsverzeichnis
nachgebaut — einmal je Sitzung melden, danach nur bei Änderung des
Gate-Zustands.

Die flache Liste bei `Stop` ist kein Detail: Antigravity nimmt dort kein
`{matcher, hooks}`, Claude schon. `ulinit` braucht deshalb einen zweiten
Schreiber für `.agents/hooks.json` neben dem für `.claude/settings.json`.

### Codex: was belegt ist und was nicht

Codex ist auf der Entwicklungsmaschine nicht installiert (`which codex` leer,
kein `~/.codex`). Belegt ist allein
`superpowers/6.3.0/docs/porting-to-a-new-harness.md:240-243, 788`: Codex hat
einen `hooks/hooks.json`-Mechanismus, „runs no session-start hook", und das
Superpowers-Manifest erklärt `hooks` absichtlich leer, um die Erkennung zu
unterdrücken. Welche Ereignisse Codex feuert und in welcher Form, ist von hier
nicht nachrechenbar.

Dieselbe Doku warnt vor genau der naheliegenden Verwechslung: „A hook *system*
is not a session-start *event*." Ein Host führte nur Pre/Post-Tool und Stop,
während die Zeichenkette `SessionStart` in seinem Binary Telemetrie war.

Der Codex-Adapter wird deshalb als Naht gebaut: Vertrag dokumentiert, ein
einziger Test, der belegt, dass ein unbekannter Host **geschlossen** fällt
(Exit 2). Kein Code, der aussieht, als würde er laufen.

### Zwei ungemessene Punkte, Stufe 0

Bevor Code darauf baut, wird gemessen:

1. **Liest Antigravity die JSON-Hülle auf stdout, oder zählt nur der
   Exit-Code?** ultra-brain hat das am 2026-09-06 gemessen und in
   `hooks/guard.sh` festgehalten: die deny-Hülle allein wurde nicht gelesen,
   eine Probedatei landete trotz Verweigerung auf der Platte, erst Exit 2
   wirkte.
2. **Kann `PreInvocation` Kontext in das Modell schreiben?** Wenn nicht, ist
   `session-start` auf Gemini nicht nachbaubar und der Inhalt gehört als Prosa
   nach `GEMINI.md`.
3. **Hat Antigravity eine harte Zeitgrenze für Stop-Handler?** Das Gate braucht
   300 Sekunden. Liegt der Deckel darunter, ist es dort nicht portierbar —
   unabhängig davon, wie gut der Port ist. Dann bleibt auf Gemini nur eine
   verkürzte Kette oder ein Gate, das den Lauf anstößt und nicht auf ihn
   wartet.

## Bausteine

Drei Schichten.

**Kern**, hostunabhängig, kennt keine JSON-Hülle:

| Paket | Aus | Zweck |
|---|---|---|
| `internal/child` | `process.py` | ein Kind mit Zeitgrenze, Prozessbaum-Tötung, Pipe-Absaugung |
| `internal/lanes` | `checks.py`, `runner.py` | Lanes ausführen, Ergebnisse sammeln |
| `internal/ulconfig` | `config.py` | `.ultraloom/config.toml` vollständig lesen |
| `internal/gitwork` | `worktree.py` | HEAD, geänderte Dateien, Vergleich seit Basis |
| `internal/journal` | `journal.py`, `gate.py`, `state.py` | Lauf-Journal, Gate- und Sitzungszustand |

**Naht**, `internal/hostio`: eine normalisierte Nutzlast (`Event`, `Tool`,
`Paths`, `SessionID`) und eine normalisierte Antwort (`Verdict`: durchlassen,
verweigern mit Grund, Kontext nachschieben). Der Kern sieht nur diese zwei
Typen.

**Adapter**, `internal/hostio/{claude,antigravity,codex}.go`: Nutzlast
erkennen, Nutzlast lesen, Antwort schreiben.

Der Host kommt als `--host claude|antigravity|codex` im Kommando, geschrieben
von `ulinit`. Eine Erkennung an der Nutzlast, wie `brain guard` sie tut, war
der erste Entwurf und ist verworfen: sie unterscheidet `tool_input.file_path`
von `TargetFile` und damit *Werkzeug*-Nutzlasten. Die vier Ereignisse hier sind
auf Claude keine Werkzeugereignisse — SessionStart, Stop und das
Subagent-Paar tragen kein `tool_input`, es gibt also nichts zu erkennen.
`brain guard` hilft nicht weiter, weil sein `extractCall` ebenfalls
Werkzeugnutzlasten liest.

Das Flag ist auch kein zweiter Ort für dieselbe Wahrheit: `ulinit` schreibt
`.claude/settings.json` und `.agents/hooks.json` aus einer Tabelle und kennt
den Host in dem Moment, in dem es das Kommando bildet. Und es ist der einzige
Weg, der schon vor der Messung von Antigravitys Stop-Nutzlast trägt.

Die Wurzel: das Arbeitsverzeichnis eines Hooks ist auf Antigravity nicht die
Projektwurzel, sondern das Verzeichnis der `hooks.json`, also `.agents/`
(`MIGRATION.md:157`, am 2026-09-10 nachgemessen). Ohne `--root` wird von dort
aufwärts bis zur ersten `.ultraloom/config.toml` gesucht; ein gegebenes
`--root` schlägt das.

Eine frühere Fassung dieses Absatzes behauptete, `${CLAUDE_PROJECT_DIR}` sei
dort nie gesetzt. Das war falsch abgeschrieben: die Doku sagt
`${CLAUDE_PLUGIN_ROOT}`, und über `CLAUDE_PROJECT_DIR` sagt sie nichts. Ob
Antigravity die Variable setzt, ist ungemessen — die Wurzelsuche ruht allein
auf dem Arbeitsverzeichnis.

Unterbefehle in `cmd/guard/`, nach dem Muster der sechs vorhandenen in
`cmd/guard/main.go`: `hook session-start`, `hook subagent-start`,
`hook subagent-stop`, `hook stop`, `check all`, `check coverage`.

## Fehlerverhalten: eine Exit-1-Disziplin, die bleibt

`payload.py` kennt 0 durchlassen, 1 interner Fehler, 2 blockieren. `stop.py`
gibt an sieben Stellen 1 zurück: `:100`, `:109`, `:146`, `:207`, `:218`,
`:227`, `:248`.

Der erste Entwurf dieses Abschnitts nannte das einen Defekt — eine kaputte
`.ultraloom/config.toml` schalte das Gate „still" ab. Das war aus den
Exit-Codes gelesen und nicht gegen den Code gerechnet, und es hält nicht: jede
dieser Stellen trägt ihr Argument daneben, und keine schweigt.

- `:109`, kein `session_id`: ein Zähler, den alle Sitzungen ohne Kennung
  teilen, würde eine Sitzung aufgeben, weil eine andere dreimal blockiert
  wurde. „Exit 1 never holds the turn, so the cost of refusing here is a line
  on stderr."
- `:207`, kaputte `[verify]`-Tabelle: „not a finding about the work, and
  holding a turn over one would leave nobody able to fix it from inside the
  session." Das Argument trägt: eine gehaltene Runde kann die Datei nicht
  bearbeiten, die sie hält.
- `:248`, Kette komplett unbenutzbar: „no verdict about the work, and so
  neither a block nor something to count."

Ebenso ist `stop_hook_active` **absichtlich** nicht gelesen (`:111-117`): es
sagt „schon einmal blockiert" und nie wie oft, kann die Obergrenze also nicht
tragen, und eine zweite Quelle, die dem Zähler widersprechen kann, macht das
Verhalten des Gates genau dann unerklärlich, wenn jemand versucht,
herauszukommen. Die Schranke ist der `MAX_BLOCKS`-Zähler im Sitzungszustand.
Der Port übernimmt beides.

Eine echte Unstimmigkeit bleibt, und nur sie wird repariert: `:141-146`. Der
Kommentar verlangt, eine Frage, die git nicht beantworten konnte, dürfe „not
end a turn as if the answer had been 'clean'" — der Code darunter verlässt mit
1, und 1 beendet die Runde. Kommentar und Code widersprechen sich. Der Go-Port
verlässt dort mit 2 und zählt den Block, womit der Kommentar gilt und die
Schleife über `MAX_BLOCKS` weiter begrenzt ist.

Kein `fail_open`-Schalter, und keine Umstellung der übrigen sechs Stellen.

## Nachweis

TDD, Test vor Code, 100 % Coverage. Vorlage sind die knapp 5.000 Python-Testzeilen
— jeder Fall wird gegen den Code nachgerechnet, nicht abgeschrieben. Der
Abschnitt über das Fehlerverhalten zeigt, wozu diese Regel da ist: dort hatte
nicht der Test das falsche Verhalten festgehalten, sondern der erste Entwurf
dieses Spec die richtige Begründung überlesen.

Golden-Dateien für die zwei bekannten Antwortformen — Claude und Antigravity —
damit ein Formatwechsel als Testfehler auffällt und nicht im Betrieb. Codex hat
keine bekannte Form; dort steht die Fail-Closed-Prüfung an ihrer Stelle.

Zwei plattformgebundene Coverage-Ausschlüsse, beide begründet:

- Job-Objekt (Windows) und `killpg` (POSIX) laufen je nur auf einem System. Die
  Wahl liegt hinter einer Funktion nach dem Muster von `terminator` in
  `process.py`, damit jeder Arm auf dem anderen System als Einheit testbar ist;
  Dateien mit Build-Tag, wie `internal/junction/junction_windows.go` es schon
  macht.
- Der Codex-Adapter hat einen Test: unbekannter Host ⇒ Exit 2. Mehr wäre
  geraten.

## Stufen

Jede Stufe endet grün und wird einzeln übergeben. Gearbeitet wird im Worktree
`.worktrees/go-hooks`: die Sitzung, die den Stop-Hook umschreibt, läuft unter
ihm.

**Stufe 0 — Messen und Vorarbeit, kein Verhalten geändert.**
`internal/settings/merge.go:407` (`toolKey`) lernt `ulguard hook <x>` wie
`ultraloom hook <x>` zu lesen. Ohne das tragen alle vier Ereignisse den
Schlüssel `ulguard`, und die Eintragsidentität stimmt nur noch zufällig über
`firstOwnedIndex`. Dazu die zwei Messungen aus der Hostmatrix.

**Stufe 1 — Naht und leichtester Hook.** `internal/hostio` mit Claude- und
Antigravity-Adapter und der Codex-Naht, `internal/journal`, `gitwork` nur mit
HEAD. Dann `ulguard hook session-start` (90 Zeilen Python).

**Stufe 2 — Subagent-Paar.** `internal/child`: `cmd.WaitDelay`, Job-Objekt auf
Windows, `killpg` auf POSIX. Dazu `gitwork` vollständig. Dann
`hook subagent-start` und `hook subagent-stop`.

**Stufe 3 — Stop-Gate.** `internal/ulconfig`, `internal/lanes`, dann
`hook stop` samt `MAX_BLOCKS`-Zähler und der einen Reparatur an `:146`.

**Stufe 4 — Lanes als Befehl.** `ulguard check all` und `check coverage`
ersetzen `uv run ultraloom check all` im erzeugten Pre-Commit-Hook
(`cmd/init/run.go:337`) und `hooks/coverage-check.py`. Der Python-Arm der
Coverage-Lane bleibt ein Kindprozess.

**Stufe 5 — Schnitt.** `cmd/init/run.go` schaltet alle Vorlagen um und bekommt
den zweiten Schreiber für `.agents/hooks.json`. Dann fallen
`src/ultraloom/hooks/`, `checks.py`, `runner.py`, `process.py`, `config.py`,
`worktree.py`, `journal.py`, `gate.py`, `state.py` und ihre Tests. `cli.py`
verliert `hook` und `check`; `tests/test_cli.py` (1410 Zeilen) und
`tests/test_module_boundary.py` (260) sind eigene Aufgaben, keine
Nebenwirkung.

Reihenfolge innerhalb jeder Stufe: der Go-Hook steht und ist grün, *dann*
schaltet `ulinit` um, *dann* fällt der Python-Hook. Nie umgekehrt.

## Installation

Zuletzt und nur hinter grünem Gate. `~/go/bin/ulguard` ist ein 41-Byte-Shim auf
`ulguard.exe`; ein `go install` schaltet die Schreibschranke jeder laufenden
ultraloom-Sitzung um, auch der migrierenden. Das bisherige `ulguard.exe` wird
vorher als `.old` beiseitegelegt — der Rückfallweg, den die Maschine für
`brain.exe` schon von Hand führt.

## Außerhalb des Schnitts

- `flows/verify_until_green.py`, `model/*` und
  `pyproject.toml [project.scripts] ultraloom`: bleiben Python am
  `claude-agent-sdk`.
- `commit/language.py` (1037 Zeilen): kein Hook, nicht betroffen.
- `ultra-brain/hooks/git/ultraloom.sh:21` ruft `python -c` für ein
  `os.path.relpath`. Ein Pfadausdruck, kein Urteil, und in einem anderen Repo.
  Eigene Runde.
- `.ultraloom/config.toml` hat keine Wiki-Gate-Mode gesetzt, deshalb schreibt
  `ulinit` die `brain`-Einträge nicht (`cmd/init/run.go:704-720`). Der
  `brain guard`-Eintrag in `.claude/settings.json` wurde am 2026-09-10 von Hand
  in init's Form nachgetragen (Timeout 15, `ultraLoomOwned`). Ob die Wiki-Mode
  gesetzt wird — und damit auch `brain wiki-gate` auf Stop dazukommt —
  entscheidet eine eigene Runde.
