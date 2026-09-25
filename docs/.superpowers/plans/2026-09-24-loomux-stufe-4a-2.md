# Stufe 4a-2: `loomux init` — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux init` richtet ein Projekt ein — Binary am kanonischen Ort,
`.loomux/config.toml` über Schema und Zeileneditor, Host-Einträge, Git-Hooks
samt post-merge, Skills, `AGENTS.md`, `.gitignore`, `.mcp.json` — gegliedert
in die Module Basis, Hooks, Wiki und Graph, und ersetzt ulinit und die zwei
Handbefehle eines frischen Klons.

**Architecture:** `init` ist ein Planer und ein Schreiber. `internal/setup`
erkennt (Stacks, Hosts, vorhandene Dateien), bildet aus Modulen und ihren
Teilen eine Liste von Änderungen — jede Datei mit altem und neuem Text, dazu
wenige Handlungen ohne Datei (Binary, `git config`, `area add`, `merge-hook
install`, `graph build`) — und schreibt sie erst nach Bestätigung, den
Zustand zuletzt. `.loomux/config.toml` ändert nur der Zeileneditor aus 4a-1
und nur, was von der Vorgabe abweicht. Host-Einträge und Git-Hooks rufen das
Binary über `${LOCALAPPDATA}` und werden nur geschrieben, wenn dieses Binary
da ist.

**Tech Stack:** Go des Moduls, `third_party/toml`, `internal/config/schema`,
`internal/config/edit`, `internal/tui`, `internal/selfupdate`,
`internal/swap`, `internal/lock`, `internal/detect`, `internal/cases`
(`gitworld`), `embed` für die Vorlagen.

**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`,
Abschnitte „Entscheidungen“, „4a-2 im Einzelnen“, „Parität“, „Fehlerverhalten“,
„Selbstnutzung“, „Messen“.

**Referenzen, festgelegt am 2026-09-24:**
- ulinit: `ultraloom` Tag `loomux-1a-source` = `9d01a60` = HEAD; `cmd/init`
  und seine 14 internen Pakete sind dort ohne Änderung im Arbeitsbaum.
- post-merge: `ultra-brain` Tag `loomux-3-source` = `3cc72d2` = HEAD. Die
  Python-Form heißt `brain-mcp hook install|status|remove`
  (`pyproject.toml:19-24`); das Go-Binary `brain` kennt `hook` nicht
  (`cmd/brain/main.go:80-124`).

## Befunde, gegen den Code gelesen am 2026-09-24

**B1. Es zieht wenig um.** `go list -deps ./cmd/init` nennt 14 Pakete mit
~2.700 Zeilen, dazu `cmd/init` mit 1.481 (ohne Tests). Nach der Spec fallen
`vendoring`, `brainpath` und die Vorlagen für `brain.toml`, `policy.toml` und
`.ultraloom/config.toml` weg. Gegen loomux gelesen fällt mehr weg:

| Paket (Zeilen ohne/mit Tests) | Befund |
|---|---|
| `detect` (412/525) | gibt es in loomux schon (`internal/detect`, `Detect(fs.FS) Facts`) |
| `gitenv` (89/55) | gibt es in loomux schon (`internal/gitenv`) |
| `commit` (62/40) | Englisch-Wortliste; loomux hat `check commit-msg` mit `[commit]` |
| `coverage`, `verify` (98+213) | Stützen von `ulinit check …`; loomux hat `check coverage`, `check lint`; `check types` (dmypy) hat keinen Nutzer in loomux |
| `tomlstr` (61) | nur für `vendoring`/`render` |
| `answers` (184/191) | Felder `commit_language`, `coverage_threshold`, `wiki_mode`, `protect_migrations`, `forbid_pip_install` sind heute Schlüssel des Schemas oder Regeln in `[policy]` |
| `interview` (313/493) | fragt über `bufio` auf stdin; die Spec stellt auf `tui` um |
| `tooling` (112/128) | prüft Stack-Werkzeuge (uv, ruff, …) und **installiert** sie über winget/curl; die Spec prüft `git`, `qmd`, `pdftotext`, `yt-dlp`, Ollama und installiert nie |
| `render` (260/596) | übrig bleiben nur `AGENTS.md.tmpl` und `verify-until-green`; `session-handover` fällt weg (#9) |
| `write` (203/322) | **zieht um**: Anlegen mit `O_EXCL`, Weigerung bei Junctions und `\`/`:` im Namen |
| `settings` (467/504) | **zieht in Teilen um**: das Zusammenführen in `.claude/settings.json` |
| `journal` | ist keine Abhängigkeit von `cmd/init` |

Die Zeile „die umgezogenen Tests“ der Paritätstabelle trägt also wenig; der
größte Teil von `init` ist neu und wird über Golden-Tests gehalten.

**B2. Mängel der Referenz, die nicht mitziehen** (Digest ulinit, `run.go`):
`--dry-run` kann Werkzeuge installieren (`:207-249`); Installationen laufen
über die `git`-Hülle, die Exit 1 ohne Ausgabe als Erfolg nimmt
(`main.go:185`); `--tool-path` wird verworfen (`:190-205`); `core.hooksPath`,
`.githooks/commit-msg` und `.gitignore` entstehen nach dem Schreiben und fehlen
in `--dry-run` und `installed.toml`; `installed.toml` wird nie aktualisiert;
`answers.toml` schlägt Flags. `init` baut jeden dieser Punkte anders; die Akte
führt sie als Abweichungen.

**B3. Wer einen Eintrag besitzt.** ulinit erkennt eigene Blöcke an
`"ultraLoomOwned": true`, entfernt nie, schreibt kein `.bak` und formatiert
die Datei neu (`settings/merge.go:360-457`). Die Spec will Erkennung an „der
Eintrag ruft ein loomux-Binary“ und ein `.bak`. Die `.claude/settings.json`
von loomux trägt keine Marke; die Logik von ulinit hielte sie für fremd.

**B4. Antigravity hat keine Referenz.** ulinit schreibt kein
`.agents/hooks.json` (`run.go:790`); es gibt nur eine handgeschriebene Datei in
`ultraloom`. Antigravity-Nutzlasten tragen keine `agent_id`, darum darf `init`
dort kein `subagent-*` eintragen (`internal/hosts/antigravity.go:10-22`).

**B5. `selfupdate.Run` kann nicht zum ersten Mal installieren.** Es überspringt,
wenn das laufende Binary nicht schon am kanonischen Ort liegt, bei
`0.0.0-dev` und außerhalb von Windows (`internal/selfupdate/update.go:74-83`);
`fetch` und `latest` sind nicht exportiert. `init` braucht einen exportierten
Erstinstall-Einstieg, der dieselbe Kette nimmt: `gh release download`,
`SHA256SUMS`, `--version` des geholten Binarys, `swap.Swap`.

**B6. `tui` hat keine Mehrfachauswahl** (Abweichung beim Bau von 4a-1). Die
Wahl „einzeln“ je Modul braucht sie.

**B7. post-merge ohne eingebackene Werte.** Der Hook der Referenz backt
Common-Dir, Zweig und Ereignispfad ein (`merge_events.py:78-115`). Die
Zustände `stale path`, `stale branch` und `shared hook path` hängen daran. Der
loomux-Hook ruft `merge-hook record`, das alles zur Laufzeit liest. Außerdem
schreibt `brain init` `merge_branch`, liest aber `branch`
(`init.py:150-152` gegen `manifest.py:41-43`); das Schema von loomux kennt
`[maintenance] branch`.

**B8. Die Flags sind durch den Wächter festgelegt.** `internal/hooks/guard.go:412`
erlaubt einem Agenten `init` nur mit `--dry-run` oder `--detect-only` als
eigenem Wort; `guard_test.go` nennt `--yes` und `--hooks=all` als verweigert.

**B9. Die Einträge von loomux selbst** (`.claude/settings.json`): Matcher
`Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell` mit Timeout 15 für
`pre-tool-use`, `Write|Edit|MultiEdit|NotebookEdit` mit 60 für
`post-tool-use`, `stop --budget 270s` mit 300, `session-start` 20,
`subagent-*` 30. ulinit schreibt Timeout 10 und kein `MultiEdit`. Die Tabelle
von `init` folgt loomux, damit `init --dry-run` auf loomux nichts ändert.

## Entscheidungen — freigegeben am 2026-09-24

| # | Frage | Entscheidung (alle wie vorgeschlagen) |
|---|---|---|
| E1 | Umfang des Umzugs | Nur `write` und das Zusammenführen aus `settings` ziehen mit ihren Tests um. `interview`/`answers` werden auf `tui` und Schema neu geschrieben; `render` bleibt als zwei Vorlagen. `commit`, `coverage`, `verify`, `tomlstr`, `tooling` und `ulinit check …` fallen weg, eingetragen in der Akte |
| E2 | Was in `.loomux/state/answers.toml` steht | Nur, was kein Schlüssel des Schemas ist: gewählte Hosts und je Modul die gewählten Teile. Alles andere steht in `.loomux/config.toml`, damit ein zweiter Lauf nicht zwei Quellen gegeneinander hält |
| E3 | Besitz eines Host-Eintrags | Ein Eintrag gehört `init`, wenn sein Befehl ein loomux-Binary ruft (`loomux`, `loomux.exe`, `bin/loomux.exe`, der kanonische Pfad); keine Marke. Fremde Einträge bleiben stehen, auch `ulguard` und `brain guard`, und werden gemeldet. Vor dem ersten Umschreiben ein `.bak`; eine Datei, die kein JSON ist oder deren `hooks` kein Objekt ist, wird nicht angefasst |
| E4 | Zustände von `merge-hook status` | `installed`, `missing`, `not installed`, `unrecorded`, `orphaned` und `refused` wie in der Referenz; `stale path`, `stale branch` und `shared hook path` entfallen, weil nichts eingebacken ist. `record` prüft Common-Dir und Zweig gegen die einwilligenden Bereiche der Registry |
| E5 | `.mcp.json` | Befehl `${LOCALAPPDATA}/loomux/bin/loomux.exe` mit `mcp --channel local`, dieselbe Form wie die Host-Einträge (#15), statt `loomux` über den `PATH`. Claude Code löst `${VAR}` in `.mcp.json` auf; ein Messschritt bestätigt das vor dem Bau. Weiter nur, wenn der Nutzerbereich keinen Server `loomux` kennt |

Ohne Frage gesetzt, weil Spec oder Code es schon festlegen:

- Erstinstall als neues `selfupdate.Install(ctx, Options) Result` neben `Run`
  (B5).
- Mehrfachauswahl als `tui.Pick` (B6).
- Schlägt der Binary-Schritt fehl, schreibt `init` auch **keine** Host- und
  Git-Hook-Einträge, die das kanonische Binary rufen: Ein Wächter-Eintrag auf
  eine fehlende Datei endet mit Exit 127 und ließe jeden Aufruf durch
  (Spec, „Host-Einträge“). Ein Checkout wie loomux, dessen Einträge
  `bin/loomux.exe` rufen, baut sein Binary stattdessen dort.
- Flags: `--root`, `--dry-run`, `--detect-only`, `--yes`,
  `--hooks=all|each|none`, `--brain=…`, `--graph=…`, `--hosts=claude,antigravity`.

Beim Planen gesetzt, weil die Spec nur den Namen nennt; der Mensch sieht sie
mit dem Plan und kann sie ändern:

- **pre-push** eines Wirts verweigert einen Push nach `main` oder `master`,
  wie `.githooks/pre-push` dieses Repos (Task 9).
- **Sicherungen** liegen unter `.loomux/state/backup/<pfad>.bak`, nicht neben
  der Datei; `.gitignore` braucht dann nur `/.loomux/state/` (Tasks 9, 12).
- **Der Hook der Referenz gilt als eigener:** `merge-hook install` ersetzt eine
  Datei mit `# brain post-merge hook`, statt sie zu verweigern (Task 6).
- **Der Wächter** verweigert einem Agenten `merge-hook install` und `remove`,
  wie `area add` (Task 14).
- **`[verify]`** schreibt `init` nicht; es fragt nach keiner Abweichung vom
  Preset.

## Offen und vor dem Bau zu messen

Task 1 misst, was `${LOCALAPPDATA}` in `.mcp.json`, in einem Git-Hook und bei
Antigravity auflöst, und wo Antigravity Skills eines Projekts sucht. Das
Arbeitsverzeichnis eines stdio-MCP-Servers aus dem Nutzerbereich misst noch
der Mensch aus 4a-1; `init` hängt nicht daran.

## Global Constraints

- Code, Bezeichner, Kommentare, Fehlermeldungen, Vorlagen und Commits
  englisch; dieser Plan und die Akte deutsch.
- Coverage 100 % je Funktion; eine Ausnahme nur mit `//coverage:exempt <Grund>`
  direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst; eine
  `embed.FS`-Variable ist erlaubt, gelesen wird beim ersten Gebrauch. Kein
  `regexp.MustCompile` auf Paketebene (`cmd/loomux/start_test.go`).
- `internal/hooks` importiert nichts aus `internal/setup/...` (Tor-Test,
  Task 14).
- `init` installiert keine fremde Software, auch nicht mit `--yes`; es prüft
  und nennt den Installationsbefehl.
- Vorgaben werden nie in `.loomux/config.toml` geschrieben; geschrieben wird nur
  über `lock.ReplaceText` und erst, wenn `schema.Validate` den Text annimmt.
- Eine vorhandene Datei, die `init` nicht zusammenführen kann, wird
  übersprungen und gemeldet, nie überschrieben.
- Host-Einträge und Git-Hooks, die ein Binary rufen, entstehen nur, wenn dieses
  Binary nach dem Binary-Schritt an seinem Ort liegt.
- Zustand zuletzt: `.loomux/state/installed.toml` wird als letzte Datei
  geschrieben.
- Exit-Codes von `init`: 0 fertig oder Bestätigung verweigert, 1 eigener
  Fehler (Schreiben, Binary-Schritt, `area add`), 2 falscher Aufruf, keine
  Konsole bei offenen Fragen, oder eine Datei, die `init` zusammenführen soll
  und nicht lesen kann.
- Commit-Nachrichten nach Conventional Commits, ohne Plan-, Stufen- oder
  Aufgabennamen; Autor ist der Nutzer, kein Modell als Mitautor. Vor jedem
  Commit Zweig und HEAD lesen; mehrzeilige Nachrichten über eine Datei und
  `git commit -F`. Backslashes in Code und Tests nur über Write/Edit, danach
  mit `grep` nachlesen.

## Review Focus

- **Der Wächter fällt nicht durch einen fehlenden Pfad.** Scheitert der
  Binary-Schritt (kein `gh`, kein Netz), schreibt `init` keinen Host-Eintrag
  und keinen Git-Hook auf das kanonische Binary und endet mit 1 — ein
  `pre-tool-use`, dessen Programm fehlt, endet mit 127 und lässt jeden Aufruf
  durch. Test in Task 13.
- **Selbstnutzung ändert nichts.** `init --dry-run` im Checkout von loomux
  meldet keine Änderung an `.claude/settings.json`, `.githooks/` und
  `.loomux/config.toml`. Test in Task 11 über eine Kopie dieser drei Dateien.
- **Alte Einträge bleiben allein.** Ein Wirt mit `ulguard …` und `brain guard`
  bekommt die loomux-Einträge **dazu**, die alten bleiben unverändert und
  werden gemeldet; ein zweiter Lauf fügt keinen zweiten loomux-Block hinzu.
  Test in Task 8.
- **Kaputte Dateien werden nicht repariert.** Eine `settings.json`, die kein
  JSON ist oder deren `hooks` kein Objekt ist, eine `.mcp.json` ebenso, und eine
  `config.toml`, die ein Lader ablehnt: keine Änderung, Meldung mit Datei,
  Exit 2. Tests in Task 8 und Task 11.
- **Ein Abbruch in der Mitte ist beim nächsten Lauf offen.** Fehlt
  `installed.toml` oder nennt es eine Datei nicht, zeigt der nächste Lauf die
  fehlenden Teile wieder an. Test in Task 12.

## Ausführungsreihenfolge und Menschenschritte

Zwei Schritte führt **der Mensch** aus; ein Subagent versucht sie nie, der
Orchestrator hält dort an und fragt:

- **Task 1** (Messungen an Claude Code, Git for Windows und Antigravity),
- **Task 15** (Selbstnutzung: `init --yes` schreibt, das darf nur ein Mensch).

Task 8 (Antigravity-Einträge), Task 10 (Skill-Ort von Antigravity) und Task 11 (`.mcp.json`) hängen an Task 1.
Reihenfolge: **1 (Mensch, parallel) · 2 · 3 · 4 · 5 · 6 · 7 · 9 · 10 · 8 · 11
· 12 · 13 · 14 · 15 (Mensch) · 16 · 17**; vor Task 10 muss Task 1 fertig sein.

**Zustandsort und Einträge.** Host-Einträge und Git-Hooks rufen wörtlich
`${LOCALAPPDATA}/loomux/bin/loomux.exe`. `init` prüft darum nach dem
Binary-Schritt genau diesen Pfad, mit `LOCALAPPDATA` aus der Umgebung
aufgelöst, und nicht `selfupdate.Canonical(config.StateDir())`. Zeigt
`config.StateDir()` woanders hin (`LOOMUX_STATE_DIR` gesetzt), schlägt der Teil
`binary` mit der Meldung `the state directory is moved by LOOMUX_STATE_DIR;
host entries would call %LOCALAPPDATA%\loomux\bin\loomux.exe` fehl, und die
Einträge entfallen. Tests setzen deshalb `LOCALAPPDATA` auf ein
Wegwerf-Verzeichnis und `LOOMUX_STATE_DIR` auf `<LOCALAPPDATA>/loomux`.

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/selfupdate/install.go` (neu) | `Install`: das erste Binary an den kanonischen Ort |
| `internal/tui/pick.go` (neu) | `Pick`: Mehrfachauswahl |
| `internal/setup/write/atomic.go` (Umzug) | Anlegen mit `O_EXCL`, Weigerung bei Junctions |
| `internal/config/manifest.go`, `declaration.go` | `Manifest.OnMerge`, `Manifest.MergeBranch` |
| `internal/brain/maintenance/mergehook.go` (neu) | Hooktext, Ort, `InstallHooks`, `HookStatus`, `RemoveHooks`, `RecordMerge` |
| `internal/cli/mergehook.go` (neu) | `loomux merge-hook install\|status\|remove\|record` |
| `internal/setup/hostfile/table.go` (neu) | die Einträge je Host |
| `internal/setup/hostfile/merge.go` (Umzug aus `settings`) | Zusammenführen ohne Marke, `.bak` |
| `internal/setup/gitfiles/gitfiles.go` (neu) | pre-commit, pre-push, commit-msg; `core.hooksPath`; `.gitignore` |
| `internal/setup/templates/…` (neu) | `AGENTS.md`, `verify-until-green`, fünf Brain-Skills, eingebettet |
| `internal/setup/facts.go`, `parts.go`, `plan.go`, `configtext.go`, `mcpjson.go`, `tools.go` (neu) | Erkennen, Teile, Änderungen |
| `internal/setup/apply.go`, `state.go` (neu) | Schreiben in Reihenfolge, `answers.toml`, `installed.toml` |
| `internal/cli/init.go`, `init_ui.go` (neu) | `loomux init` |
| `internal/cli/commands.go` | Einträge `"init"` und `"merge-hook"` |
| `internal/cli/imports_test.go` | Tor-Test gegen `internal/setup/...` im Hook-Pfad |
| `testdata/cases/4a2-*` (neu) | Welten, Aufnahmen und Übersetzung für `merge-hook` |
| `internal/cli/cases_4a2_test.go` (neu) | die Suite |
| `docs/.superpowers/parity/stufe-4a-2.md` (neu) | Akte |
| `docs/{en,de}/cli-reference.md`, `README*.md`, `AGENTS.md` | Befehl dokumentiert, Handbefehle ersetzt |
| `docs/{en,de}/benchmarks.md`, `docs/{en,de}/migration.md` | Messungen, 4a-2 ✅ |

---

### Task 1 (Mensch): Messen, was `${LOCALAPPDATA}` auflöst

Kein Code. Der Orchestrator nennt die Schritte, der Mensch führt sie aus und
trägt die Ergebnisse unter „Messungen vor dem Bau“ in diesen Plan ein.

- [ ] **Step 1: `.mcp.json` in Claude Code.** In einem Wegwerf-Verzeichnis
  mit `git init` eine `.mcp.json` anlegen:

```json
{"mcpServers":{"loomux-probe":{"command":"${LOCALAPPDATA}/loomux/bin/loomux.exe","args":["mcp","--channel","local"]}}}
```

  Claude Code dort starten, den Server freigeben, `/mcp` öffnen.
  Erwartet: `loomux-probe` verbunden, `brain_status` gelistet. Danach die
  Freigabe zurücknehmen und das Verzeichnis löschen.

- [ ] **Step 2: Git-Hook unter Git for Windows.** Im selben Wegwerf-Repo:

```sh
mkdir -p .git/hooks
printf '#!/bin/sh\n"${LOCALAPPDATA}/loomux/bin/loomux.exe" --version > probe.txt\nexit 0\n' > .git/hooks/post-commit
git commit --allow-empty -m probe
cat probe.txt
```

  Erwartet: `loomux <version> …` in `probe.txt`. Dasselbe einmal aus
  PowerShell und einmal aus der Git Bash.

- [ ] **Step 3: Antigravity.** Im Wegwerf-Repo `.agents/hooks.json` mit einem
  `PreToolUse`-Eintrag anlegen, dessen Befehl
  `"${LOCALAPPDATA}/loomux/bin/loomux.exe" --version > "${LOCALAPPDATA}/loomux-probe.txt"`
  ist, Antigravity dort eine Datei schreiben lassen und nachsehen, ob die Datei
  entstand. Außerdem eine Skill-Datei unter `.agents/skills/probe/SKILL.md`
  anlegen und prüfen, ob Antigravity den Skill listet.

- [ ] **Step 4: Eintragen.** Ergebnisse mit Datum unter „Messungen vor dem
  Bau“. **Wenn Step 3 scheitert:** Task 8 schreibt für Antigravity keine
  Einträge und meldet stattdessen „antigravity: hook entries not written; the
  host does not expand ${LOCALAPPDATA}“; Task 10 schreibt keine Skills nach
  `.agents/skills/`. **Wenn Step 1 scheitert:** Task 11 schreibt `.mcp.json`
  nicht und meldet den Befehl `claude mcp add --scope user loomux -- <pfad>
  mcp --channel local`.

## Messungen vor dem Bau

Gemessen am 2026-09-24 auf Wunsch des Nutzers vom Agenten, in Wegwerf-Ordnern
im Scratchpad, mit `loomux 2.11.1` am kanonischen Ort.

| Frage | Ergebnis |
|---|---|
| Step 2: Git-Hook mit `"${LOCALAPPDATA}/loomux/bin/loomux.exe"` unter Git for Windows | läuft, ausgelöst aus Git Bash und aus PowerShell: `loomux 2.11.1 (beta)`, Exit 0 |
| Step 1: `.mcp.json` mit `${LOCALAPPDATA}` in Claude Code | über `claude -p --mcp-config .mcp.json --strict-mcp-config`: Server `connected`, `brain_*`-Werkzeuge gelistet. Der Freigabeweg einer Projekt-`.mcp.json` (`Pending approval`) ist nicht gegangen; er braucht eine interaktive Sitzung |
| Step 3a: Skill-Ort von Antigravity | `{workspace}/.agents/skills/{skill_name}/SKILL.md`, wörtlich aus den Hilfetexten von agy 1.2.8 |
| Step 3b: `${LOCALAPPDATA}` in `.agents/hooks.json` | **nein.** agy 1.2.8 startet Hook-Befehle über `cmd.exe`: `${LOCALAPPDATA}` bleibt wörtlich stehen, `%LOCALAPPDATA%` wird aufgelöst |
| Step 3c: Anführungszeichen um das Programm | **brechen.** agy reicht `"…"` als `\"…\"` an cmd weiter; cmd sucht dann ein Programm namens `\"C:\…\loomux.exe\"`. Ohne Anführungszeichen läuft `%LOCALAPPDATA%/loomux/bin/loomux.exe --version` (Schrägstriche gehen) |
| Beobachtet: Hook scheitert | agy blockiert den Werkzeugaufruf („a pre-tool hook blocked the file creation“) — anders als Claude Code, das nur bei Exit 2 blockt |
| Beobachtet: Workspace | `agy -p` arbeitet ohne `--add-dir` in `~/.gemini/antigravity-cli/scratch` und lädt die Projekt-Hooks dann nicht |

Folgen für den Bau: Die Antigravity-Einträge (Task 8) rufen
`%LOCALAPPDATA%/loomux/bin/loomux.exe` ohne Anführungszeichen; ein
`LOCALAPPDATA` mit Leerzeichen bricht diese Form, `init` muss das prüfen und
dann keine Einträge schreiben. `--root .` stimmt nicht, weil ein Hook in
`.agents/` läuft (Messung vom 2026-09-10, Befund 2); richtig ist `..` oder die
Suche aufwärts. Skills für Antigravity gehen nach `.agents/skills/` (Task 10).
`.mcp.json` (Task 11) bleibt, wie gebaut.

---

### Task 2: `selfupdate.Install` — das erste Binary

**Files:**
- Create: `internal/selfupdate/install.go`
- Test: `internal/selfupdate/install_test.go`

**Interfaces:**
- Consumes: `latest`, `fetch`, `installedVersion`, `Newer`, `Canonical`,
  `swap.Swap`, `lock.TryAcquire` aus demselben Paket.
- Produces: `func Install(ctx context.Context, o Options) Result` —
  `Updated` mit Version, wenn es ein Release gelegt hat; `Current`, wenn am
  kanonischen Ort schon mindestens das neueste Release liegt; `Skipped`
  außerhalb von Windows; `Failed` mit `Err`; `Busy` bei gehaltener Sperre.
  `o.Executable` und `o.Version` werden **nicht** gelesen: `init` läuft aus
  einem Checkout oder `go run` und darf trotzdem installieren. Schreibt
  `update.json` nicht (das bleibt `serve` und `self-update`).

- [ ] **Step 1: Failing tests**

```go
package selfupdate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fresh is a state directory with nothing installed, and the options of an
// init that runs from anywhere.
func fresh(t *testing.T, f *fakeGH) Options {
	t.Helper()
	return Options{
		StateDir: filepath.Join(t.TempDir(), "state"), Executable: `C:\src\loomux\bin\loomux.exe`,
		Version: DevVersion, Channel: "beta", GOOS: "windows", GOARCH: "amd64",
		Run: f.run, Now: func() time.Time { return stamp },
	}
}

func TestInstallPutsTheNewestReleaseWhereNothingWas(t *testing.T) {
	o := fresh(t, release("2.12.1"))
	res := Install(context.Background(), o)
	if res.Outcome != Updated || res.Version != "2.12.1" || res.Err != nil {
		t.Fatalf("Install = %+v", res)
	}
	if got := canonicalBody(t, o); got != "binary 2.12.1" {
		t.Fatalf("canonical binary = %q", got)
	}
	if _, err := os.Stat(StatusPath(o.StateDir)); !os.IsNotExist(err) {
		t.Fatalf("Install must not write update.json: %v", err)
	}
}

func TestInstallLeavesANewerOrEqualBinaryAlone(t *testing.T) {
	f := release("2.12.1")
	f.installed = "loomux 2.12.1 (beta)\n"
	o := fresh(t, f)
	exe := Canonical(o.StateDir)
	if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("kept"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := Install(context.Background(), o)
	if res.Outcome != Current || res.Version != "2.12.1" {
		t.Fatalf("Install = %+v", res)
	}
	if got := canonicalBody(t, o); got != "kept" {
		t.Fatalf("binary replaced: %q", got)
	}
}

func TestInstallReplacesAnOlderBinary(t *testing.T) {
	f := release("2.12.1")
	f.installed = "loomux 2.11.0 (beta)\n"
	o := fresh(t, f)
	exe := Canonical(o.StateDir)
	_ = os.MkdirAll(filepath.Dir(exe), 0o700)
	_ = os.WriteFile(exe, []byte("old"), 0o755)
	if res := Install(context.Background(), o); res.Outcome != Updated {
		t.Fatalf("Install = %+v", res)
	}
}

func TestInstallSkipsOffWindows(t *testing.T) {
	o := fresh(t, release("2.12.1"))
	o.GOOS = "linux"
	if res := Install(context.Background(), o); res.Outcome != Skipped || res.Err == nil {
		t.Fatalf("Install = %+v", res)
	}
}

func TestInstallFailsWithoutGh(t *testing.T) {
	f := release("2.12.1")
	f.fail["list"] = errors.New("gh not found; install GitHub CLI and run gh auth login")
	o := fresh(t, f)
	res := Install(context.Background(), o)
	if res.Outcome != Failed || res.Err == nil {
		t.Fatalf("Install = %+v", res)
	}
	if _, err := os.Stat(Canonical(o.StateDir)); !os.IsNotExist(err) {
		t.Fatalf("a failed install left a binary: %v", err)
	}
}

func TestInstallFailsWhenTheDownloadFails(t *testing.T) {
	f := release("2.12.1")
	f.fail["download"] = errors.New("HTTP 404")
	if res := Install(context.Background(), fresh(t, f)); res.Outcome != Failed {
		t.Fatalf("Install = %+v", res)
	}
}

func TestInstallFailsWhenTheStateDirectoryCannotBeMade(t *testing.T) {
	o := fresh(t, release("2.12.1"))
	if err := os.WriteFile(o.StateDir, nil, 0o600); err != nil { // a file where the directory goes
		t.Fatal(err)
	}
	if res := Install(context.Background(), o); res.Outcome != Failed {
		t.Fatalf("Install = %+v", res)
	}
}

func TestInstallStepsAsideForAPassInProgress(t *testing.T) {
	o := fresh(t, release("2.12.1"))
	_ = os.MkdirAll(o.StateDir, 0o700)
	held, ok, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil || !ok {
		t.Fatal(err)
	}
	defer held.Release()
	if res := Install(context.Background(), o); res.Outcome != Busy {
		t.Fatalf("Install = %+v", res)
	}
}
```

Der Test importiert dafür `github.com/xidus90/loomux/internal/lock`, wie
`update_test.go` es tut. Ebenso einen Test für einen Lock, der sich nicht öffnen lässt,
nach dem Muster von `TestRunFailsWhenTheLockCannotBeOpened`, und einen für
einen Swap, der scheitert, nach `TestRunFailsWhenEverySlotIsHeld`.

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/selfupdate -run TestInstall`
Expected: FAIL, `undefined: Install`.

- [ ] **Step 3: Implementieren**

```go
package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/swap"
)

// Install puts the newest release at the canonical location, whatever runs
// it. Run replaces only the binary it runs from, which a first install has
// not got: init runs from a checkout or go run, and the place it fills is
// empty. The chain is Run's -- gh, SHA256SUMS, the fetched binary's own
// --version, swap -- so nothing reaches the canonical location that an update
// would not have put there. It never copies the running binary: a checkout
// build at that place would be replaced by the next update anyway, and until
// then every host would run someone's work in progress.
//
// update.json stays serve's and self-update's record; an install says nothing
// about the last update pass.
func Install(ctx context.Context, o Options) Result {
	if o.GOOS != "windows" {
		return Result{Outcome: Skipped, Err: errors.New("the machine-wide binary is installed on Windows only")}
	}
	canonical := Canonical(o.StateDir)
	dir := filepath.Dir(canonical)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{Outcome: Failed, Err: fmt.Errorf("create %s: %w", dir, err)}
	}
	handle, held, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	if !held {
		return Result{Outcome: Busy, Err: errors.New("update in progress")}
	}
	defer handle.Release()

	rel, err := latest(ctx, o.Run, o.Channel)
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	if have, ok := installedVersion(ctx, o.Run, canonical); ok && !Newer(rel.Tag, have) {
		return Result{Outcome: Current, Version: have}
	}
	ver := strings.TrimPrefix(rel.Tag, "v")
	if err := fetch(ctx, o, rel.Tag, dir); err != nil {
		return Result{Outcome: Failed, Version: ver, Err: err}
	}
	if err := swap.Swap(dir); err != nil {
		return Result{Outcome: Failed, Version: ver, Err: err}
	}
	return Result{Outcome: Updated, Version: ver}
}
```

`installedVersion` ruft die Datei auch, wenn sie fehlt; der Aufruf scheitert
dann, und `ok` ist falsch — genau der Fall „nichts da“. Prüfen, dass
`fakeGH` für einen fehlenden Pfad `installed` antwortet: Er tut es, weil er am
Namen und nicht an der Datei entscheidet. Darum setzt
`TestInstallPutsTheNewestReleaseWhereNothingWas` `f.installed` nicht, und die
leere Antwort ist keine Version.

- [ ] **Step 4: Grün sehen**

Run: `go test ./internal/selfupdate -cover`
Expected: PASS, `coverage: 100.0% of statements` für `install.go`
(`go test -coverprofile=c.out ./internal/selfupdate && go tool cover -func=c.out | grep install.go`).

- [ ] **Step 5: Commit**

```bash
git add internal/selfupdate/install.go internal/selfupdate/install_test.go
git commit -m "feat(selfupdate): install the newest release where nothing is installed yet"
```

---

### Task 3: `tui.Pick` — Mehrfachauswahl

**Files:**
- Create: `internal/tui/pick.go`
- Test: `internal/tui/pick_test.go`

**Interfaces:**
- Consumes: `Terminal`, `Row`, `fit`, `clearScreen`, `reverse`, `reset`, `eol`
  aus `list.go`.
- Produces: `func Pick(t Terminal, title string, rows []Row, chosen []bool) ([]bool, bool, error)`
  — Leertaste schaltet die Zeile unter dem Cursor, `a` schaltet alle an oder,
  wenn alle an sind, alle aus, Enter bestätigt (`ok=true`), Esc/`q`/Ctrl-C
  bricht ab (`ok=false`, die Eingabe unverändert zurück). `chosen` wird nicht
  verändert; die Antwort ist eine Kopie.

- [ ] **Step 1: Failing tests**

```go
package tui

import (
	"io"
	"slices"
	"strings"
	"testing"
)

func parts() []Row {
	return []Row{
		{Group: "hooks", Label: "host entries", Note: "claude"},
		{Group: "hooks", Label: "git hooks"},
		{Group: "hooks", Label: "verify-until-green skill"},
	}
}

func TestPickTogglesAndConfirms(t *testing.T) {
	in := []bool{true, true, true}
	term := Script(80, 20, Keys("down", " ", "enter")...)
	got, ok, err := Pick(term, "hooks", parts(), in)
	if err != nil || !ok || !slices.Equal(got, []bool{true, false, true}) {
		t.Fatalf("got %v %v %v", got, ok, err)
	}
	if !slices.Equal(in, []bool{true, true, true}) {
		t.Fatal("Pick changed its input")
	}
	out := term.Output()
	for _, want := range []string{"[x] host entries", "[ ] git hooks", "space toggle"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q", want)
		}
	}
}

func TestPickAllTogglesEveryRow(t *testing.T) {
	got, _, _ := Pick(Script(80, 20, Keys("a", "enter")...), "t", parts(), []bool{true, false, true})
	if !slices.Equal(got, []bool{true, true, true}) {
		t.Fatalf("a with one off turns all on: %v", got)
	}
	got, _, _ = Pick(Script(80, 20, Keys("a", "enter")...), "t", parts(), []bool{true, true, true})
	if !slices.Equal(got, []bool{false, false, false}) {
		t.Fatalf("a with all on turns all off: %v", got)
	}
}

func TestPickCancels(t *testing.T) {
	for _, k := range [][]Key{Keys("esc"), Keys("q"), Keys("ctrl-c")} {
		got, ok, err := Pick(Script(80, 20, append(Keys(" "), k...)...), "t", parts(), []bool{true, true, true})
		if ok || err != nil || !slices.Equal(got, []bool{true, true, true}) {
			t.Fatalf("%v: %v %v %v", k, got, ok, err)
		}
	}
}

func TestPickStopsAtTheEnds(t *testing.T) {
	got, _, _ := Pick(Script(80, 20, Keys("up", " ", "down", "down", "down", " ", "enter")...), "t", parts(), []bool{true, true, true})
	if !slices.Equal(got, []bool{false, true, false}) {
		t.Fatal(got)
	}
}

func TestPickReportsTheEndOfInput(t *testing.T) {
	if _, _, err := Pick(Script(80, 20), "t", parts(), make([]bool, 3)); err != io.EOF {
		t.Fatal(err)
	}
}

func TestPickScrollsInASmallWindow(t *testing.T) {
	many := make([]Row, 30)
	for i := range many {
		many[i] = Row{Group: "g", Label: string(rune('a' + i%26))}
	}
	keys := make([]Key, 0, 30)
	for range 25 {
		keys = append(keys, Key{Name: "down"})
	}
	term := Script(40, 8, append(keys, Key{Rune: ' '}, Key{Name: "enter"})...)
	got, ok, err := Pick(term, "t", many, make([]bool, 30))
	if err != nil || !ok || !got[25] {
		t.Fatalf("%v %v", ok, err)
	}
	for _, frame := range strings.Split(term.Output(), "\x1b[H\x1b[2J") {
		if n := strings.Count(frame, "\n"); n > 7 {
			t.Fatalf("a frame of %d lines in a window of 8", n+1)
		}
	}
}
```

Vor dem Schreiben nachlesen, wie `TestListScrollsInASmallWindow` die Frames
zählt (Ausgabe von `Scripted.Output()` mit `\n` statt `\r\n`), und dieselbe
Zählung nehmen.

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/tui -run TestPick`
Expected: FAIL, `undefined: Pick`.

- [ ] **Step 3: Implementieren**

```go
package tui

import (
	"slices"
	"strings"
)

// Pick shows rows with a check box each and returns which are chosen: space
// toggles the row under the cursor, a toggles every row (all off when all are
// on, else all on), enter confirms, esc, q or ctrl-c cancel and hand back the
// input unchanged. The input is never written to.
func Pick(t Terminal, title string, rows []Row, chosen []bool) ([]bool, bool, error) {
	state := slices.Clone(chosen)
	cursor := 0
	for {
		drawPick(t, title, rows, state, cursor)
		k, err := t.ReadKey()
		if err != nil {
			return slices.Clone(chosen), false, err
		}
		switch {
		case k.Name == "up" && cursor > 0:
			cursor--
		case k.Name == "down" && cursor < len(rows)-1:
			cursor++
		case k.Rune == ' ' && len(rows) > 0:
			state[cursor] = !state[cursor]
		case k.Rune == 'a':
			all := !slices.Contains(state, false)
			for i := range state {
				state[i] = !all
			}
		case k.Name == "enter":
			return state, true, nil
		case k.Name == "esc" || k.Name == "ctrl-c" || k.Rune == 'q':
			return slices.Clone(chosen), false, nil
		}
	}
}

// drawPick is draw with a box in front of every label; it shares List's
// window arithmetic, so a frame never scrolls its title away.
func drawPick(t Terminal, title string, rows []Row, state []bool, cursor int) {
	width, height := t.Size()
	lines := []string{fit(title, width), fit("↑↓ move · space toggle · a all · enter accept · q cancel", width)}
	visible := make([]int, len(rows))
	for i := range rows {
		visible[i] = i
	}
	room := max(height-len(lines), 2)
	top := windowTop(rows, visible, cursor, room)
	used, group := 0, ""
	for i := top; i < len(rows); i++ {
		need := 1
		if rows[i].Group != group {
			need++
		}
		if used+need > room {
			break
		}
		if rows[i].Group != group {
			lines = append(lines, fit("["+rows[i].Group+"]", width))
			group = rows[i].Group
		}
		box := "[ ] "
		if state[i] {
			box = "[x] "
		}
		line := fit("  "+box+rows[i].Label+"  "+rows[i].Note, width)
		if i == cursor {
			line = reverse + line + reset
		}
		lines = append(lines, line)
		used += need
	}
	_, _ = t.Write([]byte(clearScreen + strings.Join(lines, eol)))
}
```

- [ ] **Step 4: Grün sehen**

Run: `go test ./internal/tui -cover`
Expected: PASS, 100 % für `pick.go`.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/pick.go internal/tui/pick_test.go
git commit -m "feat(tui): choose several rows at once with Pick"
```

---

### Task 4: `internal/setup/write` — Umzug aus ulinit

**Files:**
- Create: `internal/setup/write/atomic.go`, `internal/setup/write/atomic_test.go`
  (beide aus `ultraloom` am Tag `loomux-1a-source`)

**Interfaces:**
- Produces (unverändert aus der Referenz): `type Plan struct { Create map[string]string; Skip []string }`,
  `func (Plan) Names() []string`, `func Prepare(root string, files map[string]string) (Plan, error)`,
  `func Commit(root string, p Plan) ([]string, error)`,
  `func CheckParents(root, name string) error`.

- [ ] **Step 1: Kopieren**

```bash
mkdir -p internal/setup/write
git -C "C:/Users/micro/Documents/#GIT/ultraloom" show loomux-1a-source:internal/write/atomic.go > internal/setup/write/atomic.go
git -C "C:/Users/micro/Documents/#GIT/ultraloom" show loomux-1a-source:internal/write/atomic_test.go > internal/setup/write/atomic_test.go
```

- [ ] **Step 2: Anpassen**
  - Importpfade `github.com/xidus90/ultra-loom/...` → `github.com/xidus90/loomux/...`.
  - Paketkommentar: „ulinit“ → „loomux init“; wo er `.ultraloom/` nennt,
    `.loomux/`.
  - Jede Paketvariable, die mehr als einen Funktionswert hält, und jedes
    `regexp.MustCompile` auf Paketebene in einen Aufruf verlegen
    (Global Constraints).
  - Der Test mit `mklink /J` bleibt; er läuft nur unter Windows.

- [ ] **Step 3: Grün und Coverage**

Run: `go test ./internal/setup/write -cover`
Expected: PASS, `coverage: 100.0%`. Eine Funktion unter 100 % bekommt einen
Test; `//coverage:exempt` nur für einen Zweig, den das Betriebssystem nicht
auslöst, mit Grund.

- [ ] **Step 4: Commit**

```bash
git add internal/setup/write
git commit -m "feat(setup): create project files without overwriting any"
```

---

### Task 5: `[maintenance]` im Manifest lesen

**Files:**
- Modify: `internal/config/manifest.go` (`Manifest`, `manifestFile`)
- Modify: `internal/config/declaration.go:93-98` (Werte behalten statt verwerfen)
- Test: `internal/config/manifest_test.go`, `internal/config/declaration_test.go`

**Interfaces:**
- Produces: `Manifest.OnMerge bool`, `Manifest.MergeBranch string` — `branch`
  aus `[maintenance]`, Vorgabe `"main"` wie `manifest.py:41-43` der Referenz.
  Der Altname `merge_branch` wird **nicht** gelesen (B7).

- [ ] **Step 1: Failing tests** — in `declaration_test.go`:

```go
func TestTheDeclarationCarriesTheMergeConsent(t *testing.T) {
	m := declared(t, "[area]\nscope = \"project/x\"\n[maintenance]\non_merge = true\nbranch = \"master\"\n")
	if !m.OnMerge || m.MergeBranch != "master" {
		t.Fatalf("OnMerge=%v MergeBranch=%q", m.OnMerge, m.MergeBranch)
	}
}

func TestTheMergeBranchDefaultsToMain(t *testing.T) {
	m := declared(t, "[area]\nscope = \"project/x\"\n")
	if m.OnMerge || m.MergeBranch != "main" {
		t.Fatalf("OnMerge=%v MergeBranch=%q", m.OnMerge, m.MergeBranch)
	}
}

func TestTheOldMergeBranchNameIsNotRead(t *testing.T) {
	m := declared(t, "[area]\nscope = \"project/x\"\n[maintenance]\non_merge = true\nmerge_branch = \"dev\"\n")
	if m.MergeBranch != "main" {
		t.Fatalf("merge_branch must stay unread, got %q", m.MergeBranch)
	}
}
```

`declared` ist der Helfer, den `declaration_test.go` schon hat oder den dieser
Schritt anlegt: Text in `t.TempDir()/.loomux/config.toml`, dann
`ReadDeclaration`. Vorher nachlesen, ob `merge_branch` als unbekannter
Schlüssel verweigert wird; tut der Leser das, erwartet der dritte Test den
Fehler statt `"main"` und heißt `TestTheOldMergeBranchNameIsRefused`. Dieselben
drei Fälle für `ReadManifest` (die alten Namen `.ultra-brain/config.toml` und
`.brain.toml`) in `manifest_test.go`.

- [ ] **Step 2: Rot sehen** — `go test ./internal/config -run Merge`, FAIL.

- [ ] **Step 3: Implementieren** — in `Manifest`:

```go
	// OnMerge is the area's consent to have merges recorded (`[maintenance]
	// on_merge`); MergeBranch the branch whose merges count, "main" when the
	// declaration names none, as the reference's reader has it.
	OnMerge     bool
	MergeBranch string
```

In `declaration` die beiden Aufrufe von `optionalBool`/`optionalString`
behalten und die Werte übernehmen; ein leerer `branch` wird `"main"`. In
`manifestFile` einen Block `Maintenance struct { OnMerge bool
\`toml:"on_merge"\`; Branch string \`toml:"branch"\` } \`toml:"maintenance"\``
und nach dem Dekodieren dieselbe Vorgabe.

- [ ] **Step 4: Grün** — `go test ./internal/config -cover`, 100 %.

- [ ] **Step 5: Commit**

```bash
git add internal/config
git commit -m "feat(config): read the merge consent and branch of an area"
```

---

### Task 6: post-merge — Hook, Ort, Einwilligung, Ereignis

**Files:**
- Create: `internal/brain/maintenance/mergehook.go`
- Modify: `internal/brain/maintenance/events.go` (`AppendEvent` exportieren)
- Modify: `internal/brain/maintenance/reconcile.go:230` (`manifestsOf` →
  exportiertes `Manifests`, Aufrufer anpassen)
- Test: `internal/brain/maintenance/mergehook_test.go`

**Interfaces:**
- Consumes: `config.Area`, `config.Manifest.OnMerge`/`MergeBranch` (Task 5),
  `config.ArtifactLookup`, `EventsPath`, `internal/gitenv`.
- Produces:

```go
// Git runs git in dir and returns its trimmed stdout.
type Git func(dir string, args ...string) (string, error)

// HookState is one line of install, status or remove.
type HookState struct{ State, Scope, Repo, Detail string }

const HookMarker = "# loomux post-merge hook"
func HookText() string
func Manifests(areas []config.Area, lookup config.ArtifactLookup) (map[string]*config.Manifest, error)
func InstallHooks(areas []config.Area, lookup config.ArtifactLookup, git Git) ([]HookState, error)
func HookStatus(areas []config.Area, lookup config.ArtifactLookup, git Git) ([]HookState, error)
func RemoveHooks(areas []config.Area, lookup config.ArtifactLookup, git Git) ([]HookState, error)
func RecordMerge(dir string, areas []config.Area, lookup config.ArtifactLookup, git Git, now time.Time) (bool, error)
func AppendEvent(stateDir string, e MergeEvent) error
func HookFailed(states []HookState) bool // refused or no repository
```

Zustände (E4): `installed`, `missing`, `not installed`, `unrecorded`,
`orphaned`, `refused`, `removed`, `no repository`. Die Einrichtungen stehen in
`<zustand>/maintenance/hooks.tsv`, eine Zeile `scope\trepo\thook`; eine Zeile
der Referenz mit vier oder fünf Feldern wird über ihre ersten drei gelesen.

Der Hooktext:

```sh
#!/bin/sh
# loomux post-merge hook -- records that something landed. Nothing more.
#
# It runs inside git merge: it may not block, fail or print. Which repository
# and branch count is decided by `loomux merge-hook record` from the registry,
# so nothing of this machine is baked in and the file may be checked in.
"${LOCALAPPDATA}/loomux/bin/loomux.exe" merge-hook record >/dev/null 2>&1
exit 0
```

Eine Datei mit der Marke der Referenz (`# brain post-merge hook`) gilt als
eigene und wird ersetzt: Sie ist der Vorgänger desselben Hooks, und 4e stellt
die Wirte um (Abweichung in der Akte).

- [ ] **Step 1: Failing tests** — über echte Repos in `t.TempDir()`,
  `git init -b main`, ein Commit; `git` über `exec.Command` mit
  `gitenv.Clean(os.Environ())`. Helfer:

```go
func realGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitenv.Clean(os.Environ())
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// consenting is a state directory with one registered area in a fresh
// repository whose declaration consents to merges on branch.
func consenting(t *testing.T, branch string) (config.ArtifactLookup, []config.Area, string) {
	t.Helper()
	state, repo := t.TempDir(), t.TempDir()
	mustGit(t, repo, "init", "-b", "main")
	mustGit(t, repo, "commit", "--allow-empty", "-m", "base")
	write(t, filepath.Join(repo, ".loomux", "config.toml"),
		"[area]\nscope = \"project/a\"\n[maintenance]\non_merge = true\nbranch = \""+branch+"\"\n")
	areas := []config.Area{{Scope: "project/a", Path: repo, WikiPath: filepath.Join(repo, "docs", "wiki"), Workspace: true}}
	return config.ArtifactLookup{Primary: state}, areas, repo
}
```

  Tests (je einer, Namen sagen die Behauptung):
  - `TestInstallWritesTheHookWhereGitLooksForIt` — `.git/hooks/post-merge`
    trägt `HookText()`, Zustand `installed`, `hooks.tsv` nennt Bereich, Repo,
    Datei.
  - `TestInstallHonoursACoreHooksPath` — `core.hooksPath = .githooks` → Datei
    unter `.githooks/post-merge`.
  - `TestInstallSkipsAnAreaThatDoesNotConsent` — `on_merge` fehlt → keine
    Zeile, keine Datei.
  - `TestInstallRefusesAForeignHook` — vorhandene Datei ohne Marke →
    `refused`, Datei unverändert, `HookFailed` wahr.
  - `TestInstallReplacesItsOwnAndTheReferencesHook` — Datei mit
    `# brain post-merge hook` und Datei mit `HookMarker` → beide `installed`,
    Inhalt `HookText()`.
  - `TestInstallReportsAnAreaThatIsNoRepository` → `no repository`,
    `HookFailed` wahr.
  - `TestStatusNamesEveryState` — Tabelle: installiert → `installed`; Datei
    von Hand gelöscht → `missing`; einwilligend ohne Datei → `not installed`;
    eigene Datei ohne Zeile in `hooks.tsv` → `unrecorded`; Einwilligung
    entzogen oder Bereich aus der Registry → `orphaned`.
  - `TestRemoveTakesOurHookAndForgetsIt`, `TestRemoveLeavesAForeignHook`
    (`refused`, Zeile bleibt), `TestRemoveWithoutAnyInstallationIsNotAnError`.
  - `TestAForeignHookThatIsNotUTF8IsStillRefused`.
  - `TestADamagedRecordLineIsSkipped` und
    `TestAReferenceRecordLineIsReadByItsFirstThreeFields`.
  - `TestRecordAppendsOneEventForAMergeOnTheConsentingBranch` — in `repo`
    Zweig `side` mit einem Commit, `git merge --no-ff side` auf `main`, dann
    `RecordMerge(repo, …)` → `true`; `ReadEvents` liefert ein Ereignis mit
    `Repo` = `--show-toplevel`, `First` = `ORIG_HEAD`, `Last` = `HEAD`,
    `Branch = "main"`, `At` = `now` auf die Sekunde, in UTC.
  - `TestRecordIsSilentOnAnotherBranch` → `false`, keine Datei.
  - `TestRecordIgnoresAMergeInAForeignRepository` — ein zweites Repo, das in
    keiner Registry steht → `false`.
  - `TestRecordCountsAMergeInAWorktreeOfTheArea` — `git worktree add`, dort
    mergen → `true`, `Repo` ist der Worktree.
  - `TestRecordWritesOneLineWhenTwoAreasShareTheRepository`.
  - `TestRecordCreatesTheMaintenanceDirectory`.
  - `TestRecordWithoutORIGHEADWritesNothing` → `false`, kein Fehler.

- [ ] **Step 2: Rot sehen** — `go test ./internal/brain/maintenance -run 'Hook|Record'`.

- [ ] **Step 3: Implementieren.** Ablauf je Funktion:
  - `Manifests` ist `manifestsOf` unter neuem Namen; `reconcile.go` ruft es.
  - Ein Bereich willigt ein, wenn sein Manifest `OnMerge` trägt; ein fehlendes
    Manifest ist ein stilles Nein (Referenz `_manifest_of`).
  - Repo: `git(area.Path, "rev-parse", "--path-format=absolute", "--show-toplevel")`;
    ein Fehler ist `no repository`. Hookordner:
    `git(area.Path, "rev-parse", "--path-format=absolute", "--git-path", "hooks")`,
    Datei `<ordner>/post-merge`.
  - Eigen ist eine Datei, deren Text `HookMarker` oder `# brain post-merge
    hook` enthält; gelesen mit `os.ReadFile` und `bytes.Contains`, damit
    nicht dekodierbare Bytes nichts ausmachen.
  - Geschrieben wird mit `lock.ReplaceText`, LF-Zeilenenden, danach
    `os.Chmod(path, 0o755)`. Zwei Bereiche mit demselben Hookordner teilen eine
    Datei; die zweite Einrichtung schreibt denselben Text und bekommt ihre
    eigene Zeile in `hooks.tsv`.
  - `hooks.tsv` wird über `lock.ReplaceText` neu geschrieben; ohne Zeile wird
    die Datei gelöscht (Referenz).
  - `RecordMerge`: `common`, `here`, `branch`
    (`symbolic-ref --quiet --short HEAD`), `first`
    (`rev-parse --quiet --verify ORIG_HEAD`), `last` (`rev-parse HEAD`) aus
    `dir`; scheitert einer, `false, nil`. Für jeden einwilligenden Bereich
    `git(area.Path, "rev-parse", "--path-format=absolute", "--git-common-dir")`;
    stimmt er mit `common` überein (`strings.EqualFold` nach
    `filepath.Clean`, weil Windows) und `branch == manifest.MergeBranch`, wird
    **einmal** `AppendEvent(lookup.Primary, MergeEvent{Repo: here, First:
    first, Last: last, Branch: branch, At: now.UTC()})` geschrieben.
  - `AppendEvent` schreibt über das vorhandene `appendLine` die Zeile
    `repo\tfirst\tlast\tbranch\t<now als 2006-01-02T15:04:05Z>` — dieselbe
    Form, die der Hook der Referenz mit `date -u` schreibt und die
    `parseEvent` liest.

- [ ] **Step 4: Grün** — `go test ./internal/brain/maintenance -cover`, 100 %
  für `mergehook.go` und `events.go`; die Suiten 3a und 3b bleiben grün
  (`go test ./internal/cli -run 'Cases3a|Cases3b'`).

- [ ] **Step 5: Commit**

```bash
git add internal/brain/maintenance
git commit -m "feat(maintenance): install a post-merge hook and record merges without baked-in paths"
```

---

### Task 7: `loomux merge-hook` und seine Fälle

**Files:**
- Create: `internal/cli/mergehook.go`, `internal/cli/mergehook_test.go`
- Modify: `internal/cli/commands.go` (`"merge-hook": mergeHookCommand`)
- Create: `testdata/cases/4a2-worlds/…`, `testdata/cases/4a2-source/…`,
  `testdata/cases/4a2-map.toml`, `testdata/cases/4a2/…`
- Create: `internal/cli/cases_4a2_test.go`
- Create: `docs/.superpowers/parity/stufe-4a-2.md` (Akte, erster Abschnitt)

**Interfaces:**
- Consumes: Task 6.
- Produces: `func mergeHookCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int`.
  `install`, `status`, `remove`: je Zustand eine Zeile
  `<state>: <scope> — <repo>` und ` [<detail>]`, wenn es einen gibt; ohne
  Zeile: `no area consents with [maintenance] on_merge = true, and no hook is
  installed`; Exit 1, wenn `HookFailed`, sonst 0. `record`: kein Argument,
  arbeitet im Arbeitsverzeichnis, schreibt nie etwas auf stdout oder stderr und
  endet **immer** mit 0. Unbekannter Unterbefehl oder Argument: Nutzung auf
  stderr, 2.

- [ ] **Step 1: Failing tests** in `mergehook_test.go` — über
  `LOOMUX_STATE_DIR` in `t.TempDir()` mit einer Registry, die ein Repo aus
  `t.TempDir()` nennt:
  - `TestMergeHookInstallPrintsOneLinePerArea` (Exit 0, Zeile
    `installed: project/a — <repo>`).
  - `TestMergeHookInstallFailsOnARefusal` (Exit 1, `refused`).
  - `TestMergeHookStatusSaysSoWhenThereIsNothing` (Exit 0, der Satz oben).
  - `TestMergeHookRecordAlwaysExitsZero` — ohne Registry, außerhalb eines
    Repos, mit kaputter Registry: je Exit 0 und leere Ausgaben.
  - `TestMergeHookRefusesAWrongCall` — `merge-hook`, `merge-hook x`,
    `merge-hook status extra` → 2.

- [ ] **Step 2: Rot sehen, Step 3: implementieren** nach dem Muster von
  `areaCommand` (`internal/cli/area.go:54`): Unterbefehl aus `args[0]`, dann
  `config.ReadRegistry(config.StateDir())`, `config.ArtifactLookup{Primary:
  config.StateDir(), Fallback: config.LegacyBrainDir()}` (Namen der
  vorhandenen Funktionen in `internal/config/artifacts.go` nachlesen),
  `realGit` wie in Task 6 als `maintenance.Git`.

- [ ] **Step 4: Fälle aufzeichnen.** Welten unter `testdata/cases/4a2-worlds/`
  nach dem Muster von `3a-worlds` (Registry mit `{{WORLD}}/repo-a`,
  `git.toml` mit `dir = "repo-a"`, Manifest `.brain.toml` mit `[maintenance]
  on_merge = true`, `branch = "main"`):
  `hook-consenting`, `hook-no-consent`, `hook-foreign` (eine
  `post-merge` ohne Marke über `[worktree]` in `.git/hooks`, sonst über
  `core.hooksPath = hooks` in `git.toml` und eine Datei `repo-a/hooks/post-merge`),
  `hook-own-earlier` (die Hookdatei der Referenz), `hook-no-repository`
  (Bereich ohne `git.toml`), `hook-orphaned` (`hooks.tsv` nennt einen
  Bereich, den die Registry nicht mehr hat), `hook-empty` (Registry ohne
  einwilligenden Bereich). Aufzeichnen mit dem Aufruf von 3a
  (`docs/.superpowers/parity/stufe-3a-orakel/record.sh` als Vorlage, als
  `stufe-4a-2-orakel/record_all.sh` kopieren):

```sh
loomux dev record-case --argv "<ultra-brain>/.venv/Scripts/brain-mcp.exe" \
  --world testdata/cases/4a2-worlds/hook-consenting \
  --out testdata/cases/4a2-source/hook/install-consenting -- hook install
```

  Je Welt die Verben, die etwas zeigen: `install` für alle, `status` für
  `hook-consenting` (nach `install` in `[worktree]` vorbereitet),
  `hook-orphaned` und `hook-empty`, `remove` für `hook-consenting`,
  `hook-foreign` und `hook-empty`.

- [ ] **Step 5: Übersetzen.** `4a2-map.toml`:

```toml
# Stage 4a-2 translates the merge hook of brain-mcp, the Python reference's
# console script: `brain-mcp hook` becomes `loomux merge-hook`, because
# `hook` is the namespace of the host hooks in loomux.
manifests = "verbatim"

[[command]]
from = "brain-mcp hook"
to   = "loomux merge-hook"
```

  Dann `go run ./cmd/loomux dev import-cases testdata/cases/4a2-source testdata/cases/4a2 --map testdata/cases/4a2-map.toml`
  (genaue Flags aus `testdata/cases/README.md` und dem Aufruf für 3c
  nachlesen). Jeder Fall bekommt `compare = message`, dessen stdout die
  Referenz auf Deutsch schreibt (`kein Bereich …`); die übrigen vergleichen
  stdout nach dem Falten von `{{WORLD}}`. Die Hookdatei selbst liegt unter
  `.git/hooks` und wird von der Aufnahme nicht kopiert; wo sie über
  `core.hooksPath` im Arbeitsbaum liegt, wird sie aus `world_after`
  ausgenommen (Weg dafür in `internal/cases` nachlesen; gibt es keinen,
  bekommt der Fall `compare = message` und die Akte einen Satz).

- [ ] **Step 6: Suite** `cases_4a2_test.go` nach dem Muster von
  `cases_3c_test.go`. Run: `go test ./internal/cli -run Cases4a2`. Expected:
  PASS. Jeder Unterschied, der bleibt, kommt als Abweichung mit Grund in die
  Akte; E4 steht dort schon als Entscheidung mit Datum.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/mergehook.go internal/cli/mergehook_test.go internal/cli/commands.go internal/cli/cases_4a2_test.go testdata/cases/4a2* docs/.superpowers/parity/stufe-4a-2.md docs/.superpowers/parity/stufe-4a-2-orakel
git commit -m "feat(cli): install, show and remove the merge hook with loomux merge-hook"
```

---

### Task 8: Host-Einträge — Tabelle und Zusammenführen

**Files:**
- Create: `internal/setup/hostfile/table.go`, `table_test.go`
- Create: `internal/setup/hostfile/merge.go`, `merge_test.go` (Umzug und Umbau
  von `internal/settings/merge.go` der Referenz)

**Interfaces:**
- Consumes: `hosts.Host`, `internal/shellwords`.
- Produces:

```go
type Entry struct {
	Event, Matcher, Command string
	Timeout                 int
}

// Canonical and Checkout are the two binaries an entry may call.
const Canonical = `"${LOCALAPPDATA}/loomux/bin/loomux.exe"`
const Checkout = `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe"`

func Entries(host hosts.Host, binary string) []Entry
func Path(host hosts.Host) string // ".claude/settings.json", ".agents/hooks.json"
func Owned(command string) bool

type Result struct {
	Merged  []byte   // equal to the input when Added is empty
	Added   []string // "<Event>/<Matcher>"
	Kept    []string // an entry of ours already there
	Foreign []string // someone else's hook on the same event and matcher, kept
}
func Merge(host hosts.Host, existing []byte, wanted []Entry) (Result, error)
func BinaryOf(host hosts.Host, existing []byte) string // Checkout when an owned entry calls bin/loomux.exe, else Canonical
```

Die Tabelle für Claude (B9), `<b>` ist `binary`, `<r>` ist
`--root "${CLAUDE_PROJECT_DIR}"`:

| Event | Matcher | Befehl | Timeout |
|---|---|---|---|
| SessionStart | — | `<b> hook session-start --host claude <r>` | 20 |
| PreToolUse | `Write\|Edit\|MultiEdit\|NotebookEdit\|Bash\|PowerShell` | `<b> hook pre-tool-use --host claude <r>` | 15 |
| PostToolUse | `Write\|Edit\|MultiEdit\|NotebookEdit` | `<b> hook post-tool-use --host claude <r>` | 60 |
| Stop | — | `<b> hook stop --host claude <r> --budget 270s` | 300 |
| SubagentStart | — | `<b> hook subagent-start --host claude <r>` | 30 |
| SubagentStop | — | `<b> hook subagent-stop --host claude <r>` | 30 |

Für Antigravity (B4) nur, wenn Task 1 Step 3 gelang: `PreToolUse` mit
Matcher `write_to_file|replace_file_content|multi_replace_file_content|run_command`
auf `<b> hook pre-tool-use --host antigravity --root .` und `PostToolUse` mit
den drei Schreibwerkzeugen auf `<b> hook post-tool-use --host antigravity
--root .`, Timeouts wie bei Claude; die Form der Datei ist
`{"loomux": {"PreToolUse": [...], "PostToolUse": [...]}}` wie die
handgeschriebene Datei in `ultraloom/.agents/hooks.json`. Den Wert von
`--root` und die Matcher bestätigt Task 1 Step 3; weicht die Messung ab, gilt
sie. Scheiterte die Messung, gibt `Entries(HostAntigravity, …)` `nil` zurück.

- [ ] **Step 1: Umzug.**

```bash
mkdir -p internal/setup/hostfile
git -C "C:/Users/micro/Documents/#GIT/ultraloom" show loomux-1a-source:internal/settings/merge.go > internal/setup/hostfile/merge.go
git -C "C:/Users/micro/Documents/#GIT/ultraloom" show loomux-1a-source:internal/settings/merge_test.go > internal/setup/hostfile/merge_test.go
```

  Paketname `hostfile`. Behalten: das Lesen als `map[string]any`, die
  Rundreise der obersten Schlüssel (`extractTopEntries`, `formatRoot`), die
  Reihenfolge der Ereignisse (`orderHooks`), die Weigerung bei kaputtem JSON,
  bei `hooks`, das kein Objekt ist, und bei einer Ereignisliste, die keine
  Liste ist. Umbauen (E3):
  - `OwnerKey` und jedes Schreiben von `"ultraLoomOwned"` entfallen.
  - `find` wird: der erste Block mit gleichem Matcher, dessen erster Befehl
    `Owned` ist → **Kept**, nichts ändert sich; ein Block mit gleichem
    Matcher, der nicht `Owned` ist → **Foreign** melden und trotzdem
    anhängen, außer ein eigener Block ist schon da; sonst anhängen →
    **Added**.
  - Kein Umschreiben eines vorhandenen eigenen Blocks, kein Entfernen.
  - Ist `Added` leer, ist `Merged` **byte-gleich** mit `existing`.
  - Tests der Referenz, die die Marke oder das Ersetzen eines eigenen Blocks
    prüfen, werden auf das neue Verhalten umgeschrieben; die Akte nennt jeden
    so geänderten Test.

- [ ] **Step 2: Failing tests (neu)** in `merge_test.go`:
  - `TestMergeLeavesLoomuxsOwnSettingsByteForByte` — die heutige
    `.claude/settings.json` dieses Repos (als Testdatei unter
    `testdata/loomux-settings.json` kopiert) gegen
    `Entries(HostClaude, Canonical)` → `Added` leer, `Kept` sechs,
    `Merged == existing`.
  - `TestMergeAddsBesideOldEntriesAndNamesThem` — Datei mit
    `ulguard --root "${CLAUDE_PROJECT_DIR}"` unter `PreToolUse` mit dem
    Matcher von ulinit und `brain guard` unter `Write|Edit|MultiEdit|NotebookEdit`
    → sechs `Added`, die alten Befehle stehen unverändert im Ergebnis, und ein
    zweiter `Merge` über das Ergebnis hat `Added` leer.
  - `TestMergeRefusesWhatItCannotRead` — Tabelle: `not json`, `[]`,
    `{"hooks": []}`, `{"hooks": {"Stop": {}}}` → Fehler, der die Datei nennt.
  - `TestMergeKeepsForeignTopLevelKeysInTheirOrder` — `permissions`, `env`,
    `model` vor und nach `hooks` bleiben in ihrer Reihenfolge.
  - `TestOwnedKnowsEveryLoomuxBinary` — wahr für `loomux hook stop`,
    `loomux.exe …`, `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe" hook …`,
    `Canonical + " hook …"`, `C:\x\loomux.exe …`; falsch für `ulguard …`,
    `brain guard`, `uv run loomux-ish`, `"loomuxer.exe"`, leer.
  - `TestBinaryOfFollowsTheCheckout` — loomux' Datei → `Checkout`; leere
    Datei und eine mit `Canonical` → `Canonical`.
  - `TestEntriesMatchTheTable` für Claude; für Antigravity nach dem Ergebnis
    von Task 1.

- [ ] **Step 3: Implementieren** `table.go`, `Owned` (erstes Wort über
  `shellwords`, `filepath.Base` nach `strings.ReplaceAll(w, "\\", "/")`,
  gleich `loomux` oder `loomux.exe`, Groß- und Kleinschreibung egal) und
  `BinaryOf` (irgendein eigener Befehl, dessen erstes Wort auf
  `/bin/loomux.exe` endet und `${CLAUDE_PROJECT_DIR}` enthält).

- [ ] **Step 4: Grün** — `go test ./internal/setup/hostfile -cover`, 100 %.

- [ ] **Step 5: Commit**

```bash
git add internal/setup/hostfile
git commit -m "feat(setup): add the host hook entries beside whatever a project already has"
```

---

### Task 9: Git-Hooks, `core.hooksPath`, `.gitignore`

**Files:**
- Create: `internal/setup/gitfiles/gitfiles.go`, `gitfiles_test.go`

**Interfaces:**
- Produces:

```go
// Hooks are the three git hooks of a host project, keyed by file name.
func Hooks(binary string) map[string]string // "pre-commit", "pre-push", "commit-msg"
// RunsAGate says whether an existing hook already runs a check chain: a
// loomux, ultraloom or ulguard call, or ci/gate.sh.
func RunsAGate(text string) bool
func GitignoreLines() []string   // "/.loomux/state/" -- backups live there too (Task 12)
func WithGitignore(text string) (string, bool) // appends the missing lines under one comment;
                                               // ".loomux/state/" and "/.loomux/state" count as present
```

  Die drei Hooks (sh, LF, `<b>` = `binary` ohne die umgebenden
  Anführungszeichen, im Text wieder in `"…"`):

```sh
#!/bin/sh
# loomux pre-commit hook: the check chain of .loomux/config.toml.
exec "<b>" check precommit
```

```sh
#!/bin/sh
# loomux commit-msg hook: the commit rules of [commit].
exec "<b>" check commit-msg "$1"
```

```sh
#!/bin/sh
# loomux pre-push hook: nobody but a human pushes to the default branch, and
# a human reads this before overriding it.
while read -r local_ref local_sha remote_ref remote_sha; do
	case "$remote_ref" in
	refs/heads/main | refs/heads/master)
		echo "pre-push: pushing to ${remote_ref#refs/heads/} is refused; open a pull request" >&2
		exit 1
		;;
	esac
done
exit 0
```

  Der Inhalt von pre-push folgt `.githooks/pre-push` dieses Repos, erweitert
  um `main`; die Spec nennt nur den Namen. Das steht als gesetzte Abweichung in
  der Akte.

- [ ] **Step 1: Failing tests** — `TestHooksCallTheGivenBinary` (für
  `Canonical` und `Checkout` steht der Pfad im Text, jede Datei beginnt mit
  `#!/bin/sh\n`, kein `\r`); `TestRunsAGateKnowsTheOldAndNewChains` (wahr für
  `.githooks/pre-commit` dieses Repos, eine Zeile `ulguard …`, `uv run
  ultraloom check precommit`, `loomux check precommit`; falsch für ein leeres
  Skript und `echo hi`); `TestPrePushRefusesTheDefaultBranches` (das Skript
  mit `sh` gegen stdin `refs/heads/a 1 refs/heads/main 2` → Exit 1, gegen
  `refs/heads/feat` → 0; übersprungen, wenn `sh` fehlt);
  `TestWithGitignoreAppendsOnlyWhatIsMissing` (leer, teilweise, vollständig,
  ohne letzten Zeilenumbruch, mit CRLF — die Zeilenenden der Datei bleiben).

- [ ] **Step 2: Rot, Step 3: implementieren, Step 4: grün** —
  `go test ./internal/setup/gitfiles -cover`, 100 %.

- [ ] **Step 5: Commit**

```bash
git add internal/setup/gitfiles
git commit -m "feat(setup): write the git hooks and ignore lines of a host project"
```

---

### Task 10: Vorlagen — `AGENTS.md` und die Skills

**Files:**
- Create: `internal/setup/templates/templates.go`, `templates_test.go`
- Create: `internal/setup/templates/files/AGENTS.md.tmpl`
- Create: `internal/setup/templates/files/skills/verify-until-green/SKILL.md`
- Create: `internal/setup/templates/files/skills/brain-{ingest,land,research,review,wiki-plan}/SKILL.md`

**Interfaces:**
- Produces:

```go
type File struct{ Path, Text string } // Path relative to the project root, slash-separated
type Vars struct{ Project, CommitLanguage string; Hosts []hosts.Host }
func AgentsMD(v Vars) (string, error)
func Skills(names []string, host hosts.Host) ([]File, error) // .claude/skills/<n>/SKILL.md or .agents/skills/<n>/SKILL.md
func SkillNames(module string) []string // "hooks": verify-until-green; "brain": the five
```

`//go:embed files` auf einer `embed.FS`-Variable; `text/template` wird erst
in `AgentsMD` geparst (Global Constraints).

- [ ] **Step 1: Inhalte schreiben.**
  - `AGENTS.md.tmpl`: aus `ultraloom/internal/render/templates/AGENTS.md.tmpl`
    am Tag, englisch, ohne `uv`, `ulguard`, `.ultraloom/` und Hook-Shims; mit
    dem Satz, dass `.loomux/config.toml` nur ein Mensch schreibt und ein
    Agent `loomux config set … --propose` nimmt, und mit der Commit-Sprache aus
    `Vars.CommitLanguage`.
  - `verify-until-green`: aus der Vorlage der Referenz, die Schleife ruft
    `loomux check all` statt `uv run ultraloom check all`.
  - Die fünf Brain-Skills: aus `ultra-brain/.claude/skills/*/SKILL.md` am Tag
    `loomux-3-source`, **ins Englische übersetzt** (#7), sonst inhaltlich
    gleich. Umgeschrieben: `brain catalog|read|search|case|cases` →
    `loomux brain catalog|read|search`, `loomux case|cases`;
    `brain check bundle|all` → `loomux brain check bundle|all`;
    `uv run brain-mcp wiki init` → `loomux wiki init` (die Warnung, das
    Go-Binary kenne `wiki`, entfällt); `uv run brain-mcp approve …` →
    `loomux approve …`. Zuerst die Befehle von `loomux brain --help` und
    `loomux --help` lesen und jeden Namen dagegen prüfen. Die Übersetzung
    darf ein Subagent machen; ihr Ergebnis wird Absatz für Absatz gegen das
    Original gelesen.

- [ ] **Step 2: Failing tests** — `TestAgentsMDNamesTheLanguageAndTheProposalRule`;
  `TestSkillsGoWhereTheHostLooks` (Claude `.claude/skills/…`, Antigravity den
  Ort aus Task 1); `TestNoSkillCallsAnOldCommand` (kein Skill enthält
  `uv run`, `brain-mcp`, `ultraloom`, `ulguard` oder ein `brain ` am
  Zeilen- oder Backtick-Anfang, das nicht `loomux brain ` ist);
  `TestEveryLoomuxCommandInASkillExists` (jedes `loomux <wort>` ist ein
  Schlüssel der Befehlstabelle — über eine Liste, die der Test von
  `internal/cli` bekommt, weil `templates` `cli` nicht importieren darf:
  der Test steht darum in `internal/cli/templates_test.go`);
  `TestSkillsAreEnglish` (keine Umlaute, kein `ß`).

- [ ] **Step 3–4: Implementieren, grün** — `go test ./internal/setup/templates ./internal/cli -run 'Template|Skill|AgentsMD' -cover`.

- [ ] **Step 5: Commit**

```bash
git add internal/setup/templates internal/cli/templates_test.go
git commit -m "feat(setup): ship AGENTS.md and the skills a project gets, in English and on loomux commands"
```

---

### Task 11: Der Planer — Erkennen, Teile, Änderungen

**Files:**
- Create: `internal/setup/facts.go`, `parts.go`, `plan.go`, `configtext.go`,
  `mcpjson.go`, `tools.go` und je ein `_test.go`
- Create: `internal/setup/testdata/loomux/` (Kopie von `.claude/settings.json`,
  `.githooks/*` und `.gitignore` dieses Repos; `.loomux/config.toml` liegt
  dort als `loomux-config.toml`, weil der Wächter jeden Pfad
  `.loomux/config.toml` für Agenten sperrt. Der Test kopiert sie zur Laufzeit
  an ihren Ort unter `t.TempDir()`. Kopiert wird mit `cp` in der Shell, nicht
  mit Write.)

**Interfaces:**
- Consumes: `detect.Detect`, `detect.HooksPath`, `hostfile` (Task 8),
  `gitfiles` (Task 9), `templates` (Task 10), `schema`, `edit`,
  `schema.Validate`.
- Produces:

```go
type Facts struct {
	Root       string
	Detect     detect.Facts
	Hosts      []hosts.Host // .claude/ -> claude, .agents/ -> antigravity; neither -> claude
	HooksPath  string       // core.hooksPath, "" when unset
	Config     string       // .loomux/config.toml, "" when missing
	UserMCP    bool         // ~/.claude.json names an mcpServers entry "loomux"
	Registered bool         // a registry area's Path is Root (config.ReadRegistry(config.StateDir()))
	Checkout   bool         // go.mod declares github.com/xidus90/loomux
	Binary     string       // hostfile.Canonical or hostfile.Checkout
}
func Gather(root, home string, git detect.Runner) (Facts, error)

type Part struct {
	Module  schema.Module
	ID      string // "binary", "config", "gitignore", "agents-md", "mcp-json", "tools",
	               // "host-entries", "git-hooks", "verify-skill",
	               // "area", "merge-hook", "brain-skills", "graph-build"
	Label   string
	Default bool
}
func Parts(f Facts) []Part

type Choice struct {
	Hosts          []hosts.Host
	Parts          map[string]bool
	CommitLanguage string // "en" unless asked otherwise
	Scope          string // "project/<directory name>" unless asked otherwise
}
func DefaultChoice(f Facts, answers Answers) Choice

type Change struct {
	Part          string
	Path          string // slash-separated, relative to Root
	Before, After string
	Exists        bool
}
func (c Change) Empty() bool { return c.Before == c.After }

type Action struct{ Part, ID, Describe string } // "binary-install", "binary-build", "hooks-path", "area-add", "merge-hook", "graph-build"
type Plan struct {
	Changes []Change // Empty() ones are dropped
	Actions []Action
	Notes   []string // skipped files, foreign entries, missing tools, the user-scope MCP server
}
func Build(f Facts, c Choice, read func(rel string) ([]byte, bool, error)) (Plan, error)
```

Die Teile je Modul (Spec, Tabelle „Ablauf“, mit E1–E5):

| Modul | Teil | Vorgabe | Wirkung in `Build` |
|---|---|---|---|
| Basis | `binary` | an | `binary-build`, wenn `Checkout` oder `Binary == Checkout`; sonst `binary-install` |
| Basis | `config` | an | `[modules]` (nur `false` wird geschrieben), `[commit] language` (nur, wenn nicht `en`), die Regeln aus dem Katalog unten |
| Basis | `gitignore` | an | `gitfiles.WithGitignore` |
| Basis | `agents-md` | an | `AGENTS.md`, nur wenn keine da ist (#10) |
| Basis | `mcp-json` | an | `.mcp.json`, außer `UserMCP` (dann nur eine Notiz) |
| Basis | `tools` | an | nur Notizen: `git`, `qmd`, `pdftotext`, `yt-dlp`, `ollama` fehlen → Notiz mit Installationsbefehl |
| Hooks | `host-entries` | an | `hostfile.Merge` je gewähltem Host |
| Hooks | `git-hooks` | an, wenn `.git` da ist | die drei Hooks in den Hookordner, `hooks-path`, wenn nötig |
| Hooks | `verify-skill` | an | `verify-until-green` je Host |
| Wiki | `area` | an, wenn `[area]` fehlt **und** `Registered` falsch ist | `area-add` mit `Scope` und `docs/wiki` |
| Wiki | `merge-hook` | an, wenn `.git` da ist | `merge-hook` |
| Wiki | `brain-skills` | an | die fünf Skills je Host |
| Graph | `graph-build` | an, wenn ein Stack erkannt ist | `graph-build` |

**Vorgaben aus dem Bestand.** `DefaultChoice` beginnt bei dem, was schon gilt:
Module und Commit-Sprache aus `schema.Current(f.Config)` (ein `[modules]
graph = false` bleibt aus, ein `language = "de"` bleibt `de`), Hosts und
Teile aus `answers.toml`, erst danach die Tabelle oben. `init --yes` ändert
einen vorhandenen Wert darum nie. Sonst gälte die Regel der Spec nicht, dass
ein vorhandener Wert sich nur gezielt ändert.

**Ein Checkout von loomux** (`Checkout`) bekommt nur, was seine eingecheckten
Dateien schon haben: `binary` (als `binary-build`), `config`, `gitignore`,
`host-entries` und `git-hooks` sind an. `agents-md`, `mcp-json`,
`verify-skill`, `brain-skills`, `area` und `graph-build` sind aus; ein Mensch
kann sie im Dialog anwählen. Ohne diese Vorgabe hinterließe `init --yes`
auf einem frischen Klon neue Dateien, und `git status` wäre nicht leer.

Regelkatalog für `[policy]` (aus ulinits Fragen, jetzt ohne Frage und nur,
wenn der Stack erkannt ist und die Regel noch nicht im Text steht):

| Stack | Regel |
|---|---|
| `django` | `[[policy.paths.rules]]` `match = ["**/migrations/[0-9][0-9][0-9][0-9]_*.py"]`, `reason = "Applied migrations are history; add a new one instead."` |
| `uv` | `[[policy.commands.rules]]` wie im Block unten |

```toml
[[policy.commands.rules]]
regex  = '(^|[\n;&|(`])\s*pip\s+install([^\w-]|$)'
reason = "uv, never pip."
```

Das ist die Beispielregel aus `.loomux/config.toml` dieses Repos, Zeichen für
Zeichen.

Die `regex` wird über **Write** in die Go-Quelle geschrieben und danach mit
`grep -n 'pip' internal/setup/configtext.go` nachgelesen (Backslashes,
Global Constraints).

Hookordner der Git-Hooks: `HooksPath`, wenn gesetzt (relativ zu `Root`
aufgelöst); sonst `.githooks/` mit der Handlung `hooks-path`
(`git config core.hooksPath .githooks`). Eine vorhandene Hookdatei bleibt,
Notiz `"<datei>: kept; it runs a gate already"`, wenn `gitfiles.RunsAGate`,
sonst `"<datei>: kept; a hook of the project is already there"`.

`.mcp.json` wird wie `settings.json` als `map[string]any` gelesen; kaputtes
JSON oder ein `mcpServers`, das kein Objekt ist → Fehler; ein vorhandener
Schlüssel `loomux` bleibt. Der neue Eintrag:

```json
{"mcpServers": {"loomux": {"command": "${LOCALAPPDATA}/loomux/bin/loomux.exe", "args": ["mcp", "--channel", "local"]}}}
```

Installationsbefehle der Werkzeug-Notiz: vor dem Schreiben je Werkzeug mit
`winget search <name>` die Id prüfen und die geprüfte Zeile in eine Tabelle in
`tools.go` schreiben; wo `winget` keine hat (`qmd`), den Befehl aus dem README
des Projekts nehmen und die Quelle im Kommentar nennen.

- [ ] **Step 1: Failing tests**
  - `TestSelfUseChangesNothing` — `Facts` aus `testdata/loomux/` (Root eine
    Kopie in `t.TempDir()`, `.git` als leeres Verzeichnis, `HooksPath =
    ".githooks"`, `Checkout = true`), `DefaultChoice`, `Build` → keine
    `Change` für `.claude/settings.json`, `.githooks/*` und
    `.loomux/config.toml`; `binary-build` steht in `Actions`.
  - `TestAFreshRepositoryGetsEveryPart` — leeres Repo mit `go.mod` eines
    anderen Moduls → Änderungen für `.loomux/config.toml` (nur, wenn eine
    Regel greift; bei reinem Go keine), `.gitignore`, `AGENTS.md`,
    `.mcp.json`, `.claude/settings.json` (sechs Einträge mit
    `hostfile.Canonical`), `.githooks/{pre-commit,pre-push,commit-msg}`,
    `.claude/skills/verify-until-green/SKILL.md`, fünf Brain-Skills;
    Handlungen `binary-install`, `hooks-path`, `area-add`, `merge-hook`,
    `graph-build`.
  - `TestAModuleSwitchedOffIsWrittenAsFalse` — `hooks` abgewählt →
    `[modules] hooks = false` im neuen Text, kein Host-Eintrag, keine
    Git-Hooks, kein `verify-skill`; alle drei an → kein `[modules]`.
  - `TestTheCommitLanguageIsWrittenOnlyWhenItIsNotTheDefault`.
  - `TestTheDefaultChoiceKeepsWhatTheConfigSays` — `.loomux/config.toml` mit
    `[commit] language = "de"` und `[modules] graph = false`,
    `DefaultChoice`, `Build` → keine Änderung an der Datei.
  - `TestARegisteredRootGetsNoAreaAdd` — `Registered` wahr, `[area]` fehlt
    → kein `area-add`.
  - `TestACheckoutGetsOnlyWhatItHasCheckedIn` — `Checkout` →
    `Parts` mit den Vorgaben aus „Ein Checkout von loomux“, `Build` über
    `testdata/loomux` → keine neue Datei.
  - `TestTheRuleCatalogFollowsTheStacks` — Django-Welt bekommt die
    Pfadregel, uv-Welt die Befehlsregel, ein zweiter `Build` über das
    Ergebnis keine zweite.
  - `TestAConfigTheLoadersRejectIsNotTouched` — `.loomux/config.toml` mit
    `[commit] threshold = "x"` → `Build` gibt einen Fehler mit dem Dateinamen.
  - `TestABrokenSettingsFileStopsThePlan` und
    `TestABrokenMCPFileStopsThePlan`.
  - `TestAUserScopeServerLeavesMCPJsonAlone` — `UserMCP` → keine Änderung,
    eine Notiz.
  - `TestAnExistingHookIsKeptAndNamed` — `.githooks/pre-commit` mit
    `ulguard …` → keine Änderung daran, Notiz „runs a gate already“.
  - `TestAnExistingAgentsMDIsKept`.
  - `TestGatherReadsHostsHooksPathAndTheUserScope` — mit einem falschen
    `detect.Runner` und einem `home` aus `t.TempDir()` mit `.claude.json`.
  - `TestNoRepositoryMeansNoGitParts`.

- [ ] **Step 2: Rot, Step 3: implementieren.** `configtext.go` rechnet den
  neuen Text nur mit `edit.Set`, `edit.Remove` und `edit.AppendBlock` und
  prüft ihn am Ende mit `schema.Validate`; ein vorhandener Text wird vorher
  ebenso geprüft, und ein Fehler bricht `Build` ab (Spec, „Fehlerverhalten“).

- [ ] **Step 4: Grün** — `go test ./internal/setup -cover`, 100 %.

- [ ] **Step 5: Commit**

```bash
git add internal/setup
git commit -m "feat(setup): plan what init changes, file by file, from modules and their parts"
```

---

### Task 12: Der Schreiber und der Zustand

**Files:**
- Create: `internal/setup/apply.go`, `state.go`, `apply_test.go`, `state_test.go`

**Interfaces:**
- Consumes: Task 11, `write.Prepare`/`Commit` (Task 4), `lock.ReplaceText`.
- Produces:

```go
type Answers struct {
	Hosts []string        `toml:"hosts"`
	Parts map[string]bool `toml:"parts"`
}
func ReadAnswers(root string) (Answers, error) // .loomux/state/answers.toml; missing -> zero value
type Runner func(a Action) error
type Report struct {
	Written, Skipped, Failed []string // paths and action ids
	Refused                  []string // changes the human declined
}
// Apply writes the approved changes and runs the actions in this order:
// binary, files, hooks-path, area-add, merge-hook, graph-build, answers,
// installed. A binary step that fails drops every change and action that
// calls the binary -- host entries, git hooks, merge-hook -- and reports
// them as failed.
func Apply(root string, p Plan, c Choice, approve func(Change) bool, run Runner, binaryThere func() bool, now time.Time) (Report, error)
```

  Einzelheiten:
  - Neue Dateien über `write.Prepare`/`Commit` (nie überschreiben); geänderte
    über `lock.ReplaceText`. Vor dem ersten Umschreiben einer vorhandenen
    Datei, die nicht `.loomux/config.toml` ist, wird ihr alter Text nach
    `.loomux/state/backup/<pfad>.bak` gelegt, wenn dort noch keiner liegt.
  - Nach dem Binary-Schritt entscheidet `binaryThere()`; ist es falsch,
    entfallen die Änderungen der Teile `host-entries` und `git-hooks` und die
    Handlung `merge-hook`.
  - `.loomux/state/answers.toml` (Hosts, Teile) und zuletzt
    `.loomux/state/installed.toml`:

```toml
version = "2.12.1"
at = 2026-09-24T18:00:00Z
files = [".claude/settings.json", ".githooks/pre-commit"]
actions = ["binary-install", "hooks-path"]
```

    `installed.toml` wird bei jedem Lauf neu geschrieben, wenn etwas geschah.

- [ ] **Step 1: Failing tests**
  - `TestApplyWritesInOrderAndStateLast` — ein `Runner`, der die Reihenfolge
    der Handlungen und über `os.Stat` die Existenz von `installed.toml`
    mitschreibt: `installed.toml` fehlt bei jeder Handlung und steht am Ende.
  - `TestAMissingBinaryDropsWhatCallsIt` — `binaryThere` falsch → keine
    `.claude/settings.json`, keine `.githooks/*`, kein `merge-hook`;
    `AGENTS.md` und `.gitignore` geschrieben; `Failed` nennt die entfallenen
    Teile.
  - `TestADeclinedChangeIsNotWrittenAndNotAnError`.
  - `TestAnExistingFileIsBackedUpOnceBeforeItChanges`.
  - `TestAnInterruptedRunIsOpenOnTheNextOne` — ein `Runner`, der bei
    `area-add` scheitert → `installed.toml` fehlt; ein neuer `Build` über
    denselben Baum zeigt `area-add` wieder, die geschriebenen Dateien nicht
    mehr.
  - `TestAnswersRoundTrip` und `TestMissingAnswersAreTheZeroValue`.

- [ ] **Step 2–4:** rot, implementieren, grün mit 100 %.

- [ ] **Step 5: Commit**

```bash
git add internal/setup
git commit -m "feat(setup): apply an init plan with backups, the binary first and the state last"
```

---

### Task 13: `loomux init`

**Files:**
- Create: `internal/cli/init.go`, `internal/cli/init_ui.go`, `internal/cli/init_test.go`
- Modify: `internal/cli/commands.go` (`"init": initCommand`)

**Interfaces:**
- Consumes: Tasks 2, 3, 11, 12; `openTerminal` (`config_ui.go`),
  `areaCommand`, `graphCommand`, `mergeHookCommand`, `selfupdate.Install`.
- Produces: `func initCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int`
  und die Nähte

```go
var installBinary = func(ctx context.Context) selfupdate.Result { return selfupdate.Install(ctx, selfUpdateOptions(selfupdate.SourceCLI)) }
var buildCheckout = func(root string) error // go build -o bin/loomux.new.exe ./cmd/loomux, then swap.Swap(root/bin), as .githooks/pre-commit does
var runAction = func(name string, args []string, stdout, stderr io.Writer) int // "area", "merge-hook", "graph" -> the command of that name
```

`runAction` ist die Naht, über die Tests `area add` (das `reindex` und damit
`qmd` startet), `merge-hook install` und `graph build` ersetzen. Im Betrieb
ist es ein `switch` auf `areaCommand`, `mergeHookCommand` und `graphCommand`,
**nicht** ein Nachschlagen in `commands`: `commands` nennt `initCommand`, und
eine Paketvariable, die `commands` liest, ergäbe einen Initialisierungszyklus.

  Flags (B8): `--root DIR` (Vorgabe: das Arbeitsverzeichnis), `--dry-run`,
  `--detect-only`, `--yes`, `--hooks=all|each|none`, `--brain=all|each|none`,
  `--graph=all|each|none`, `--hosts=claude,antigravity`. `--detect-only` mit
  einem anderen Flag außer `--root` ist ein falscher Aufruf.

  Ablauf:
  1. `setup.Gather`; `--detect-only` schreibt `Facts` als JSON (zwei
     Leerzeichen Einzug) auf stdout, Exit 0.
  2. `DefaultChoice` aus `answers.toml`. Mit `--yes` oder `--dry-run` ohne
     Konsole gelten die Vorgaben. Sonst braucht `init` eine Konsole; ohne:
     `loomux init: init asks questions; run it in a terminal, or pass --yes
     or --dry-run`, Exit 2.
  3. Fragen auf der Konsole: je Modul ein `tui.Input` mit den Wahlen
     `all`, `each`, `none` (Vorgabe aus dem Flag oder `all`); `each` öffnet
     `tui.Pick` über die Teile des Moduls. Dann `tui.Input` für die
     Commit-Sprache (Wahlen `en`, `de`) und, wenn `area` gewählt ist, für den
     Bereich (Vorgabe `project/<ordnername>`, Prüfung: nicht leer, kein
     Leerzeichen).
  4. `setup.Build`; ein Fehler → Meldung mit Datei auf stderr, Exit 2.
  5. Zusammenfassung auf stdout: je Änderung `edit.Diff(Before, After)` unter
     der Zeile `--- <pfad>`, dann die Handlungen, dann die Notizen. Mit
     `--dry-run` Ende hier, Exit 0.
  6. Bestätigen: mit `--yes` alles, sonst je Änderung und Handlung
     `tui.Confirm`.
  7. `setup.Apply` mit einem `Runner`, der `binary-install` über
     `installBinary` (ein `Failed` → Fehler mit dem Rat `gh auth login` und
     `loomux self-update`), `binary-build` über `buildCheckout`,
     `hooks-path` über `git config core.hooksPath .githooks`, `area-add` über
     `areaCommand([]string{"add", "--path", root, "--scope", scope, "--yes"}, …)`,
     `merge-hook` über `mergeHookCommand([]string{"install"}, …)` und
     `graph-build` über `graphCommand([]string{"build", "--root", root}, …)`
     führt; `binaryThere` prüft den Pfad, den die Einträge wörtlich rufen:
     `os.Getenv("LOCALAPPDATA") + "/loomux/bin/loomux.exe"` bzw.
     `<root>/bin/loomux.exe` für einen Checkout (siehe „Zustandsort und
     Einträge“ oben). Vor `installBinary` prüft `init`, dass
     `config.StateDir()` gleich `%LOCALAPPDATA%\loomux` ist; sonst scheitert
     `binary-install` mit der Meldung von dort.
  8. Bericht; Exit 1, wenn `Failed` nicht leer ist, sonst 0.

- [ ] **Step 1: Failing tests** (Zustand über `LOOMUX_STATE_DIR` in
  `t.TempDir()`, `installBinary` und `buildCheckout` ersetzt, Repo aus
  `git init`):
  - `TestInitDetectOnlyPrintsTheFacts` — JSON mit `Hosts`, `Detect`,
    `HooksPath`; nichts geschrieben.
  - `TestInitDryRunShowsEveryChangeAndWritesNothing` — stdout nennt
    `.claude/settings.json`, `.githooks/pre-commit`, `binary-install`; der
    Baum ist danach byte-gleich (Hash über alle Dateien vorher und nachher).
  - `TestInitYesSetsUpAFreshRepository` — `installBinary` legt eine Datei
    an `Canonical` → Exit 0, alle Dateien aus Task 11 da,
    `installed.toml` zuletzt, `git config core.hooksPath` ist `.githooks`.
  - `TestInitWritesNoHookEntryWhenTheBinaryIsMissing` — `installBinary`
    gibt `Failed` ohne Datei → Exit 1, keine `.claude/settings.json`, keine
    `.githooks/*`, stderr nennt `gh auth login`; `AGENTS.md` ist da.
  - `TestInitChecksThePathTheEntriesCall` — `installBinary` legt die Datei
    nach `LOOMUX_STATE_DIR/bin`, das **nicht** unter `LOCALAPPDATA` liegt →
    Exit 1, keine Einträge, stderr nennt `LOOMUX_STATE_DIR`.
  - `TestInitAsksPerModule` — `openTerminal` gibt ein `tui.Script`:
    Basis `all`, Hooks `each` mit abgewähltem `git-hooks`, Wiki `none`,
    Graph `all`, Sprache `de`, dann `y` für jede Frage → keine
    `.githooks/*`, kein Brain-Skill, `[commit] language = "de"`,
    `[modules] brain = false`.
  - `TestInitRefusesWithoutATerminal` → Exit 2, nichts geschrieben.
  - `TestInitRefusesABrokenSettingsFile` → Exit 2, die Datei unverändert.
  - `TestInitSecondRunChangesNothing` — zweimal `--yes` → der zweite Lauf
    meldet keine Änderung und keine Handlung außer denen, die immer laufen
    (keine).
  - `TestInitRefusesAWrongCall` — `--detect-only --yes`, `--hooks=some`,
    ein Positionsargument → 2.
  - `TestInitOnTheCheckoutBuildsInsteadOfInstalling` — Welt aus
    `internal/setup/testdata/loomux` mit `go.mod` von loomux → `buildCheckout`
    gerufen, `installBinary` nicht.

- [ ] **Step 2–4:** rot, implementieren, grün;
  `go test ./internal/cli -run TestInit -cover`, 100 % für `init.go` und
  `init_ui.go`.

- [ ] **Step 5: Wächter gegenlesen.** `go test ./internal/hooks -run Guard`
  bleibt grün; die erlaubten Formen `loomux init --dry-run` und
  `--detect-only` sind genau die, die `initCommand` ohne Schreiben ausführt.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/init.go internal/cli/init_ui.go internal/cli/init_test.go internal/cli/commands.go
git commit -m "feat(cli): set up a project with loomux init"
```

---

### Task 14: Tor-Test und Wächter für `merge-hook`

**Files:**
- Modify: `internal/cli/imports_test.go`
- Modify: `internal/hooks/guard.go` (`wordsWriteConfiguration`), `guard_test.go`

- [ ] **Step 1: Failing tests.**
  - Im Tor-Test die verbotenen Pakete um `internal/setup` und alles darunter
    erweitern: `internal/hooks` darf sie nicht erreichen.
  - In `guard_test.go`: verweigert für einen Agenten `loomux merge-hook
    install` und `loomux merge-hook remove` (sie legen ausführbare Hooks in
    fremde Repos, wie `area add` in fremde Konfiguration schreibt); erlaubt
    `loomux merge-hook status` und `loomux merge-hook record`.

- [ ] **Step 2–4:** rot, in `wordsWriteConfiguration` einen Zweig
  `case "merge-hook":` mit `args[0] == "install" || args[0] == "remove"`
  ergänzen und die Ablehnungsbegründung um „merge-hook install and remove“
  erweitern; grün.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/imports_test.go internal/hooks
git commit -m "feat(guard): refuse agents installing or removing merge hooks"
```

---

### Task 15: **Halt** — Selbstnutzung (Mensch)

Der Orchestrator hält an; ein Agent ruft `init` nur mit `--dry-run` oder
`--detect-only`.

- [ ] **Agent:** `go run ./cmd/loomux init --dry-run` in diesem Checkout.
  Expected: keine Änderung an `.claude/settings.json`, `.githooks/` und
  `.loomux/config.toml`; `binary-build` als Handlung. Die Ausgabe kommt in die
  Akte.
- [ ] **Mensch:** frischer Klon von loomux in ein Wegwerf-Verzeichnis, dort

```sh
go run ./cmd/loomux init --yes
git config core.hooksPath
ls bin/loomux.exe
go run ./cmd/loomux init --dry-run
git status --porcelain
```

  Expected: `.githooks`, das Binary liegt da, der zweite Aufruf zeigt keine
  Änderung, und `git status --porcelain` ist leer (`bin/` und
  `.loomux/state/` sind ignoriert). Danach ist der Klon ohne die zwei
  Handbefehle einsatzbereit.
- [ ] **Mensch:** ein Wirt nach Wahl (Vorschlag: eine Kopie von `ecoflow`)
  mit `loomux init` interaktiv; `pre-tool-use` greift in einer neuen
  Claude-Code-Sitzung dort (ein `git push` wird verweigert).
- [ ] Ergebnisse mit Datum in die Akte, Abschnitt „Selbstnutzung“.

---

### Task 16: Mutationsrunde, Messung, Akte

- [ ] **Mutanten:**

```bash
./bin/loomux.exe dev mutants ./internal/setup
```

  Dasselbe für `./internal/setup/hostfile`, `./internal/setup/gitfiles`,
  `./internal/brain/maintenance` (nur `mergehook.go`) und
  `./internal/selfupdate` (nur `install.go`). Jeder Überlebende bekommt einen
  Test oder einen begründeten Eintrag in der Akte.
- [ ] **Messung**, Median aus 10 warmen Läufen, kalt einmal:
  `loomux init --dry-run` auf loomux; `loomux hook pre-tool-use` vor und nach
  dieser Stufe (die warme Zeit darf nicht messbar steigen). Eingetragen in
  `docs/en/benchmarks.md` und `docs/de/benchmarks.md`.
- [ ] **Startzeit:** `$env:GODEBUG="inittrace=1"; ./bin/loomux.exe --version`
  — kein neues `init` über 500 Allokationen aus `internal/setup/...`.
- [ ] **Akte** `parity/stufe-4a-2.md` vollständig: Referenzen, E1–E5 mit
  Datum, B2 als Abweichungen, die umgeschriebenen Tests aus Task 8, die
  Fallsuite, die gesetzten Punkte (pre-push, Ort der Sicherungen, Marke der
  Referenz als eigene, Wächter für `merge-hook`), Selbstnutzung, Überlebende.
- [ ] **Tor:** `go run ./cmd/loomux check precommit`, grün.
- [ ] Commit: `docs: record the parity of init and the merge hook`

---

### Task 17: Doku und Migrationsplan

- [ ] `docs/{en,de}/cli-reference.md`: `loomux init` (Flags, Module, Teile,
  Exit-Codes, Wächter) und `loomux merge-hook`.
- [ ] `README.md`, `README.de.md`: `init` und `merge-hook` von „Spezifiziert“
  zu „Aktiv“; „Opening a fresh clone“ mit `go run ./cmd/loomux init --yes`
  statt der zwei Handbefehle.
- [ ] `AGENTS.md`, Abschnitt „Commands“: die zwei Handbefehle werden
  `go run ./cmd/loomux init --yes`, gerufen von einem Menschen.
- [ ] `docs/{de,en}/migration.md`: 4a-2 ✅ mit dem, was die Stufe gebracht hat;
  die Abhängigkeiten von 4c und 4e neu lesen; die Funktionszeilen für `init`,
  post-merge und die Brain-Skills umstellen.
- [ ] Fusions-Spec: Nachträge #5, #7, #8, #10, #11, #15, #16 „gebaut mit
  4a-2“; die Stufenzeile.
- [ ] `go run ./cmd/loomux check precommit` grün.
- [ ] Commit: `docs: document loomux init and the merge hook`
- [ ] Vor dem Push der Skill `release-pr`: Label `release:minor`, ein
  Changelog-Block (`Added`: `loomux init`, `loomux merge-hook`,
  `selfupdate`-Erstinstall als Teil von `init`), geprüft mit
  `loomux dev release parse-body`.

## Fertig, wenn

1. E1–E5 freigegeben sind (erledigt, 2026-09-24) und die Messungen von Task 1
   eingetragen,
2. die Fälle von `merge-hook` grün oder mit Grund freigegeben sind und die
   Suiten aller früheren Stufen grün bleiben,
3. die Coverage 100 % ist und jeder Ausschluss begründet,
4. die Mutationsrunde gelaufen ist und die Überlebenden dokumentiert sind,
5. die Messungen eingetragen sind,
6. die Selbstnutzung von Task 15 gelaufen ist,
7. `internal/hooks` nichts aus `internal/setup` erreicht.

## Selbstprüfung des Plans

- **Spec-Abdeckung:** Umzug (E1, Tasks 4, 8), Ablauf mit Modulen und Flags
  (Tasks 11, 13), Host-Einträge über `${LOCALAPPDATA}` (Task 8, Messung
  Task 1), Binary an den kanonischen Ort (Tasks 2, 12, 13), post-merge
  (Tasks 5–7), Skills (Task 10), `AGENTS.md` (#10) und `.gitignore` (#11)
  (Tasks 9–11), `.mcp.json` (E5, Task 11), `--detect-only` (#16, Task 13),
  Fehlerverhalten (Tasks 11–13), Selbstnutzung (Task 15), Messen (Task 16).
  `[verify]` schreibt `init` nicht: Die Spec will „nur Abweichungen vom
  Preset“, und `init` fragt nach keiner.
- **Typen:** `Change`, `Action`, `Plan`, `Choice`, `Facts` in Task 11
  definiert und in 12 und 13 mit denselben Namen gebraucht;
  `hostfile.Canonical`/`Checkout` in 8 definiert, in 11 gebraucht;
  `maintenance.Git` in 6, in 7 gebraucht.
- **Review Focus:** jede der fünf Zeilen hat ihren Test (Tasks 8, 11, 12, 13).
