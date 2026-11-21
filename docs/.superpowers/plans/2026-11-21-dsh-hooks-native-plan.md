# Native DSH-Hooks (dsh-loomux-hooks) – Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Ziel:** Der loomux-Hook-Pfad (Schreibschranke, Check-Kette, Stop-Steuerung) läuft in jeder DeepSeek-Harness-Sitzung über das neue, abhängigkeitsfreie Plugin `dsh-loomux-hooks`.

**Architektur:** Das Plugin meldet sich an den Cordis-Seams `agent/created`, `tools/pre-execute`, `tools/post-execute`, `agent/turn-stopping`, `subagent/start`, `subagent/end`, liest Hook-Definitionen im Claude-Dialekt aus einer Auflösungskette und spawnnt `loomux.exe hook <event> --host dsh` per `child_process` mit dem Wirts-Token – echte Exit-Codes, schreibfähige Kinder. Auf der Go-Seite ist `dsh` ein neuer Gastwert, der Payload-Analyse und Antwort-Hülle vollständig mit `claude` teilt.

**Tech Stack:** Go 1.24+ (Bestand), Node-ESM-Paket ohne Laufzeit-Abhängigkeiten (Bruder von `dsh-loomux/`), `node --test` mit Abdeckungs-Gate.

**Spec:** `docs/.superpowers/specs/2026-11-21-dsh-hooks-native-design.md` (enthält die Spike-Beweistabelle; von ihr stammt die Seam-Tafel).

## Global constraints

- Branch `feat/dsh-native-hooks`; niemand außer einem Menschen pusht.
- Code, Kommentare, Fehlermeldungen, Commit-Texte Englisch; dieses Papier und die Spec bleiben deutsch.
- JS: ESM, `"type": "module"`, keine Laufzeit-Abhängigkeit, `engines.node >= 22.19.0`, `node --test --experimental-test-coverage --test-coverage-lines=100 --test-coverage-functions=100`; Zweigabdeckung nicht gehalten. Lizenz `PolyForm-Noncommercial-1.0.0`.
- Go: 100 % je Funktion (`//coverage:exempt <Grund>` direkt über `func` als einzige Ausnahme); kein `init()`; Gate `sh ci/gate.sh` muss grün sein.
- Commits nach Conventional Commits, ein Thema pro Commit, keine Arbeitspapier-Namen im Text.
- Seam-Namen des Plugins stehen als zentrale Konstanten in `index.js` (Cordis ist alpha).
- Die Entscheidungszuordnung pro Seam ist die Tafel der Spec („Seams und Zuordnung"); Exit 2 sperrt an allen drei Werkzeugpunkten, JSON-Kanäle zusätzlich.

## Dateistruktur

| Datei | Verantwortung |
| --- | --- |
| `internal/hosts/hostio.go` | Gastwert `HostDsh`, `ParseHost`, Payload-Zerlegung (teilt den claude-Zweig) |
| `internal/cli/hook.go` | Hilfetext nennt `dsh` (Logik unverändert) |
| `internal/setup/hostfile/table.go` | `Entries`/`Path` für `HostDsh`: `.loomux/dsh/hooks.json`, Matcher `.*` |
| `internal/setup/` (bestehender Init-Pfad) | schreibt die dsh-Hook-Datei aus der Tafel |
| `dsh-loomux-hooks/package.json` | Paket wie `dsh-loomux/package.json` gespiegelt, `dsh.bundle`-Deklaration |
| `dsh-loomux-hooks/cordis.patch.yml` | Plugin-Anmeldung beim Profil (Muster `dsh-loomux/cordis.patch.yml`) |
| `dsh-loomux-hooks/index.js` | Seam-Anmeldung, Payload-Bau, Steuerungs-Zuordnung, Audit-Zeilen |
| `dsh-loomux-hooks/src/config.js` | Auflösungskette + Claude-Dialekt-Parser + Matcher-Prüfung |
| `dsh-loomux-hooks/src/spawn.js` | ein Hook-Kind-Prozess: stdin-Payload, Stdio-Fang, Timeout |
| `dsh-loomux-hooks/src/decision.js` | Exit-Code/Stdout → normalisierte Entscheidung der Spec-Tafel |
| `dsh-loomux-hooks/test/*.test.js` | Je Datei eine Testdatei, echte `node -e`-Kinder im Spawn-Test |
| `ci/dsh-loomux-hooks.sh`, `.github/workflows/dsh-loomux-hooks.yml` | Spiegel der `dsh-loomux`-Pendants |
| `docs/en/dsh-hooks.md`, `docs/de/dsh-hooks.md` | Doku seitenweise in beiden Sprachen |
| `README.md`, `README.de.md` | DSH-Abschnitt und Roadmap-Zeile |

## Task 1: Gast `dsh` auf der Gastschiene (Go)

**Erzeugt:** `hosts.HostDsh Host = "dsh"`; `ParseHost("dsh")` und alle
Payload-Leser akzeptieren den claude-förmigen DSH-Payload;
`Answer(HostDsh, …)` Durchlauf wie claude (ohne Codeänderung belegt).

- [ ] In `internal/hosts/hostio_test.go` failing tests: `ParseHost("dsh")`
      liefert `HostDsh`; die Fehlermeldung von `ParseHost("q")` zählt `dsh`
      auf; der claude-Payload-Fixture zerlegt mit `HostDsh` feldgleich wie
      mit `HostClaude` (Session-ID, Agent-Felder, Ereignis).
- [ ] `go test ./internal/hosts` – muss scheitern.
- [ ] `hostio.go`: Konstante `HostDsh Host = "dsh"`, `ParseHost`-Fall,
      die beiden `case HostClaude:`-Zweige (Zeilen ~106 und ~131) zu
      `case HostClaude, HostDsh:`; Kommentar, dass DSH den claude-Payload
      spricht (Spike 2026-11-21, gemessen an `dsh-llm 0.2.2-rc.1`).
- [ ] `internal/cli/hook.go`: usage- und `--host`-Texte auf
      `claude, dsh, antigravity or codex`; `hostio.go`-Fehlermeldung ebenso.
- [ ] `go test ./internal/hosts ./internal/cli` – grün.
- [ ] Commit `feat(hosts): speak the claude payload for the dsh host`.

## Task 2: Tafel-Eintrag und Init für die dsh-Hook-Datei (Go)

**Erzeugt:** `hostfile.Entries(HostDsh, binary)` und
`hostfile.Path(HostDsh) == ".loomux/dsh/hooks.json"`; ein Init-Pfad
(Flag oder Unterbefehl nach dem Muster des bestehenden claude-Schritts)
schreibt die Datei in claude-förmiger Gestalt (Container `hooks`).

- [ ] In `internal/setup/hostfile/` failing test `TestEntriesDsh`:
      sechs Einträge (`SessionStart`, `PreToolUse`, `PostToolUse`, `Stop`,
      `SubagentStart`, `SubagentStop`) mit Matcher `.*` statt
      `Write|Edit|…` (DSH-Werkzeugnamen sind `write`, `edit`, `pwsh`),
      Timeout-Felder wie claude, `Stop` mit `--budget 270s`,
      `Command` endet auf `hook <event> --host dsh --root "${CLAUDE_PROJECT_DIR}"`.
- [ ] Failing test `TestPathDsh` → `.loomux/dsh/hooks.json`.
- [ ] `go test ./internal/setup/hostfile` – scheitert.
- [ ] `table.go`: `HostDsh`-Zweige in `Entries`/`Path`; Container bleibt
      `hooks` (claude-Form, wie das Spike-`hooks.json` bewiesen hat).
- [ ] Init-Pfad: dort, wo claude sein `settings.json` schreibt, den
      dsh-Gast mit derselben Tafel und `Path(HostDsh)` bedienen (Ordner
      `.loomux/dsh/` anlegen); `Owned`/`hookEventOf` bleiben gültig.
- [ ] `go test ./internal/setup/...` grün; `sh ci/gate.sh` grün.
- [ ] Commit `feat(setup): install the dsh hook file from the host table`.

## Task 3: Paketgerüst und Hook-Kind-Läufer (JS)

**Erzeugt:** `dsh-loomux-hooks/{package.json,cordis.patch.yml,index.js,src/spawn.js}`;
`runHook(command, payload, {cwd, timeoutMs}) -> {exitCode, stdout, stderr, timedOut}`.

- [ ] `package.json` als Spiegel von `dsh-loomux/package.json`
      (Name `dsh-loomux-hooks`, Beschreibung für Hooks, gleiche
      `exports`/`files`/`scripts`/`engines`/`publishConfig`, `dsh.bundle`
      mit `patch: "./cordis.patch.yml"`); `cordis.patch.yml` meldet das
      Plugin nach dem Muster von `dsh-loomux/cordis.patch.yml` an.
- [ ] `test/spawn.test.js` failing: `node -e` mit Ausgabeproduktion schreibt
      stdout und Exit 0; `process.exit(2)` liefert `exitCode 2`; ein
      stdin lesendes Kind empfängt das JSON-Payload; `timeoutMs` tötet ein
      schlafendes Kind (`timedOut true`, `exitCode` null); ein
      `#`-Ordnerpfad im Kommando überlebt den Spawn (Windows-Realtest).
- [ ] `node --test test/spawn.test.js` – scheitert.
- [ ] `src/spawn.js`: `child_process.spawn` mit `shell: true`, cwd,
      `env: {...process.env, CLAUDE_PROJECT_DIR: cwd}`; stdin =
      `JSON.stringify(payload) + "\n"`; Timeout über `AbortSignal`;
      close/`error` zu einem Ergebnis; keine Ausnahmen nach außen.
- [ ] Test grün. Commit `feat(hooks): add dsh hook plugin skeleton with a faithful hook process runner`.

## Task 4: Auflösungskette und Claude-Dialekt-Parser (JS)

**Erzeugt:** `resolveConfig(explicitPath, workdir) -> {entries, sourcePath} | null`;
Parser für `{hooks:{<Ereignis>:[{matcher,hooks:[{type:"command",command,timeout}]}]}}`.

- [ ] `test/config.test.js` failing: Kette `configPath` →
      `<workdir>/.loomux/dsh/hooks.json` → `<workdir>/.claude/settings.json`
      → Codex-Position (`hooks/hooks.json` neben
      `.codex-plugin/plugin.json` wie in `internal/hosts/codex.go`
      dokumentiert); erster Treffer gewinnt; keine Datei → `null`;
      `type != "command"` wird verworfen; Matcher `Write[` (kaputte Regex)
      verwirft die ganze Datei mit Grund; `SessionStart`/`Stop` ohne
      Matcher behalten Matcher `.*`; `timeout`-Sekunden → `timeoutMs`.
- [ ] `node --test test/config.test.js` – scheitert.
- [ ] `src/config.js` schreiben (nur `node:fs`/`node:path`).
- [ ] Test grün. Commit `feat(hooks): resolve hook definitions along the dsh chain`.

## Task 5: Entscheidungs-Maschine (JS)

**Erzeugt:** `decide(seam, {exitCode, stdout, stderr, timedOut}) -> {kind, reason?, context?}`
mit `kind ∈ none|deny|ask|block|steer|context`.

- [ ] `test/decision.test.js` failing, eine Gruppe je Spec-Tafel-Zeile:
      Pre: exit 2 → deny mit stderr-Grund; stdout
      `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny"},"…}`
      → deny; `ask` → ask; `permissionDecision:"allow"` → none;
      Post: exit 2 → block; deny → block;
      `additionalContext` → context; Stop: exit 2 → steer; stdout
      `{"decision":"block","reason":…}` → steer mit reason;
      `{"decision":"approve"}` → none; SessionStart/SubagentStart:
      `hookSpecificOutput.additionalContext` (korrektes
      `hookEventName`) → context; falsches `hookEventName` lässt
      ereignisbezogene Felder verfallen; Exit 1, Timeout,Spawn-Fehler →
      none (interner Fehler sperrt nie).
- [ ] `node --test test/decision.test.js` – scheitert.
- [ ] `src/decision.js` schreiben.
- [ ] Test grün. Commit `feat(hooks): map hook exits and json onto cordis decisions`.

## Task 6: Seam-Verdrahtung im Plugin (JS)

**Erzeugt:** `index.js`: `export const name`, `inject`, `default function
apply(ctx, config)`; Payload-Bauer; Audit-Zeilen; Loopschutz.

- [ ] `test/index.test.js` failing mit `ctx`-Attrappen (Seam-Registratur
      abfangen, `agent.inject`, `agent.steer`, Session-Log-Anhängen):
      ohne gefundene Datei werden keine Seams gemeldet; `agent/created`
      erwartet das Hook-Ergebnis **vor** Rückkehr und injiziert context;
      `tools/pre-execute` gibt bei deny `{kind:"deny", reason}` und sonst
      `next()` zurück; `tools/post-execute` liefert
      `{kind:"block", feedback:[{type:"text",…}]}` und stellt context
      voran; `agent/turn-stopping` lenkt höchstens achtmag pro
      Agentenlauf und zählt; `subagent/start` wartet und injiziert ins
      Kind, `subagent/end` meldet nur; Payload-Felder wie im Spike
      beobachtet (`session_id` aus `agent.session.header.id`,
      `transcript_path ""`, `cwd`, `tool_name = exec.name`,
      `tool_input = exec.arguments`, `tool_response` als Text,
      `stop_hook_active` aus dem Zähler, Subagenten `agent_type
      "general-purpose"`); Audit-Zeile `hook/invoked`/`hook/result`
      mit `point`, `handlerId`, `decision`, `exitCode`, `stderrSummary`
      (500 Zeichen Kappe); ist kein Log-Dienst da, schreibt `ctx.logger.warn`.
- [ ] `node --test test/index.test.js` – scheitert.
- [ ] `index.js` schreiben: Seam-Namen als Konstanten obenan;
      `resolveCwd` nach dem Muster von `dsh-loomux/index.js`;
      Hook-Befehle laufen durch `runHook` mit `cwd` des Agenten und
      `CLAUDE_PROJECT_DIR`; Plugin-Konfiguration `{ configPath?, binary?,
      defaultTimeoutMs? }`.
- [ ] `npm test` im Paket grün bei 100 % Linien/Funktionen.
- [ ] Commit `feat(hooks): drive the cordis seams from the hook file`.

## Task 7: CI und Abdeckungs-Gate (JS)

- [ ] `ci/dsh-loomux-hooks.sh` nach `ci/dsh-loomux.sh` (same Flags,
      anderes Paketverzeichnis, ausführbar); `.github/workflows/
      dsh-loomux-hooks.yml` nach dem `dsh-loomux`-Workflow
      (Touch-Pfad `dsh-loomux-hooks/**`, Tag `dsh-loomux-hooks-v*`).
- [ ] `sh ci/dsh-loomux-hooks.sh` lokal grün.
- [ ] Commit `ci(hooks): run the dsh hook package tests at full coverage`.

## Task 8: Doku und Roadmap

- [ ] `docs/en/dsh-hooks.md` + `docs/de/dsh-hooks.md` (gleicher Dateiname):
      Installation über plugin_manager oder Profil-Patch,
      Auflösungskette, Seam-Tafel, Fehlerbilder (Exit-Verhalten,
      Sandbox-Hinweis, `stop_hook_active`), Verweis auf Spike-Ergebnisse.
- [ ] `README.md`/`README.de.md`: DSH-Abschnitt beim Grep-Paket um den
      Hook-Pfad erweitern; Roadmap-Zeile „native DSH-Hooks
      (`dsh-loomux-hooks`)“ aus `Coming` entfernen (Erledigt-Zeile verlässt
      die Roadmap) – in beiden Dateien, Sprache der Datei folgend.
- [ ] `sh ci/gate.sh` grün. Commit `docs: document the native dsh hook path`.

## Task 9: Echtheit in einer DSH-Sitzung (kein Commit)

- [ ] `go run ./cmd/loomux init --yes` (or the dsh hook-file step of it)
      writes `.loomux/dsh/hooks.json` in this checkout; verify the six
      entries and `.*` matchers by reading the file.
- [ ] `npm pack` im Plugin-Ordner, im Profil installieren
      (`pnpm add <tgz>` unter `%USERPROFILE%\.dsh\profiles\desktop`,
      `bundles`-Eintrag nach `dsh-loomux`-Muster), Nutzer startet neu.
- [ ] Checkliste in der Sitzung: Sperrtext der Schreibschranke bei
      `spike-block.txt` erscheint als Tool-Fehler; Stop-Steuerung kommt
      genau einmal zurück; `SessionStart`-Kontext steht als erste
      User-Nachricht eines neuen Agenten; `hook/invoked`/`hook/result`
      stehen im Session-Log (`session.v4.jsonl.zstd`).
- [ ] Danach: `release-pr`-Skill; Push durch den Menschen.
