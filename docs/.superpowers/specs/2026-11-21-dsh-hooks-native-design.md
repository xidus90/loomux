# Native DSH-Hook-Brücke (dsh-loomux-hooks)

Stand 2026-11-21. Geplanter Branch `feat/dsh-native-hooks`.

## Anlass

`dsh-loomux` deckt in DeepSeek Harness (DSH) heute nur das `grep`-Werkzeug
ab; der Hook-Pfad von loomux (Schreibschranke, Check-Kette, Stop-Steuerung)
läuft in DSH-Sitzungen nicht. Am 2026-11-20/21 wurde der Hook-Pfad mit dem
offiziellen Bridge-Paket `@deepseek-ai/dsh-hooks-claude-code` 0.2.2-rc.1 auf
den nativen Cordis-Seams live gespidet (montiert per Profil-Patch
`insert:`, protokollierende Hooks, sperrande Marker-Tests). Ergebnis – alle
Zeilen live belegt:

| Beobachtung | Konsequenz |
| --- | --- |
| Bridge module-loadet nicht, bis ihre Peer-Pakete (`@deepseek-ai/dsh-llm`, `@deepseek-ai/dsh-hook-protocol` in Kernversion 0.2.2-rc.1) von Hand installiert sind | eigenständiges Paket muss abhängigkeitsfrei sein |
| Payload im Claude-Dialekt; `tool_name` = nativer DSH-Name (`write`, `edit`, `pwsh`); `transcript_path` immer `""`; `stop_hook_active` immer `false` | loomux-Matcher müssen die DSH-Namen kennen; Loopschutz braucht eigene Zählung |
| Entscheidungskanäle wirken nur als JSON bei Exit 0: `permissionDecision` deny/ask (Pre), deny→block+feedback (Post), `decision:"block"`→`agent.steer` (Stop) | exit-2-Kanal von loomux' Claude-Gast ist unbrauchbar |
| Hook-Kind-Prozess läuft schreibgesperrt: EPERM auf Workspace, `%TEMP%`, `%APPDATA%`, `%LOCALAPPDATA%` (ACL-Diagnose: kein Rechtefehler, Sandbox so) | State-Schreiben und Check-Kette (`go build`/`go test`) sind im Hook-Prozess der Bridge unmöglich |
| `SessionStart`-`additionalContext` erscheint im neuen Agenten als erste User-Nachricht; `SubagentStart`-Injektion läuft ungewartet und kommt erst nach Step 1 an | Injektion muss gewartet werden (Rennrennen der Bridge) |
| `hook/invoked`/`hook/result`-Paare schreiben in das Session-Log (wendengebundene Punkte) | Audit-Kanal existiert und wird nachgebaut |

Die Bridge kann den loomux-Hook-Pfad nicht tragen: sie verliert Exit-Codes,
sperrt dem Kind jedes Schreiben und gewinnt das Subagent-Rennen nicht.
Entscheidung des Nutzers vom 2026-11-21: natives Cordis-Plugin als zweites
Paket.

## Ziel

1. Der loomux-Hook-Pfad läuft in jeder DSH-Sitzung: `SessionStart`,
   `PreToolUse`, `PostToolUse`, `Stop`, `SubagentStart`, `SubagentStop` –
   die sechs Ereignisse, die `loomux hook` heute kennt. `UserPromptSubmit`
   registriert das Plugin nicht, solange loomux kein solches Ereignis hat
   (Nachtrag 2026-11-21 nach Blick in `internal/cli/hook.go`).
2. Neues npm-Paket `dsh-loomux-hooks/` im Repo: ESM, ohne Laufzeit-Abhängigkeit,
   mit `dsh.bundle`-Deklaration, damit plugin_manager es verwalten kann.
3. Spawn von `loomux.exe` durch `child_process` mit dem Wirts-Token: echte
   Exit-Codes 0/1/2 nach loomux' Guard-Protokoll; Kindprozesse sind
   schreibfähig (loomux-State, Check-Kette).
4. loomux-Gastschiene `dsh` (`internal/hosts/dsh.go`): gleiches stdin-Payload
   und gleiche Antwort-Hülle wie der claude-Gast, Event-Tabelle und
   Loopschutz auf DSH angepasst.
5. Hook-Dateien im Claude-Dialekt; Auflösung in dieser Reihenfolge:
   `configPath` aus der Plugin-Konfiguration, sonst
   `.loomux/dsh/hooks.json`, sonst `.claude/settings.json`, sonst die
   Codex-Position (`hooks.json` wie in `internal/hosts/codex.go` dokumentiert);
   findet sich nichts, registriert das Plugin keine Hooks.
6. Installation: loomux kann die dsh-Hook-Datei schreiben, analog zur
   claude-Gast-Registrierung (Einträge mit Matcher `.*` und absolutem Pfad
   auf das loomux-Binary).
7. Coverage wie gehabt: JS 100 % Linien/Funktionen, Go 100 % je Funktion;
   Doku in beiden Sprachen, READMEs und Roadmap-Zeile aktualisiert.

## Nicht-Ziel

- wiki/check als DSH-Werkzeuge über MCP (bleibt Roadmap-Maybe).
- Änderungen am Grep-Paket `dsh-loomux`.
- Kein Editor oder Oberfläche für die Hook-Dateien.
- Weitere DSH-Ereignisse als die sechs registrierten (insbesondere
  `UserPromptSubmit`, bis loomux das Ereignis hat).
- Das Bridge-Paket selbst: es bleibt, wie es ist; loomux nutzt es nicht.

## Entwurf

### Plugin `dsh-loomux-hooks/`

- `package.json`: `dsh.bundle`-Deklaration nach dem Muster von `dsh-loomux`,
  ESM-Einstieg, keine Abhängigkeiten, Versionsschema `dsh-loomux-hooks-v*`.
- Plugin-Konfiguration: `{ configPath?: string, binary?: string,
  defaultTimeoutMs?: number }`; ohne `configPath` greift die
  Auflösungskette (Ziel 5) relativ zum Arbeitsverzeichnis des Agenten,
  ermittelt über den Working-Directory-Dienst.
- Fehler beim Lesen oder Parsen der Hook-Datei: warnten und keine Hooks
  registrieren (Brückentrauen), die Sitzung läuft unbehelligt weiter.
- Der Parser akzeptiert den Claude-Dialekt: Events, Matcher-Gruppen,
  ausschließlich `type: "command"`; ein Matcher, der keine gültige Regex ist,
  verwirft die ganze Datei (Brückentraues Verhalten, gematcht wird mit
  `RegExp#test` auf das Ereignis-Subjekt: Tool-Name, Session-Quelle,
  Subagent-Typ, sonst leerer String).
- Seams und Zuordnung:

  | Cordis-Seam | Ereignis | Entscheidung |
  | --- | --- | --- |
  | `agent/created` | SessionStart | gewartet; `additionalContext` → `agent.inject` |
  | `tools/pre-execute` | PreToolUse | exit 2 oder deny → `{kind:"deny"}`; ask → `{kind:"ask"}` |
  | `tools/post-execute` | PostToolUse | exit 2 oder deny → `{kind:"block"}` + Rückmeldung; Kontext voranstellen |
  | `agent/turn-stopping` | Stop | exit 2 oder `decision:"block"` → `agent.steer(Grundsatznachricht)` |
  | `subagent/start` | SubagentStart | gewartet; Kontext → `child.inject` |
  | `subagent/end` | SubagentStop | laufen lassen, Entscheidung ignorieren |

- Spawn: `child_process.spawn(command, { shell: true, cwd, env })` mit
  `CLAUDE_PROJECT_DIR`; stdin ist das JSON-Payload mit Zeilenumbruch;
  Timeout abbricht das Kind und gilt als interner Fehler; Exit ungleich
  0/1/2 zählt wie 1.
- Loopschutz: Plugin-seitig härteste Obergrenze von acht Stop-Steuerungen
  pro Agentenlauf; zusätzlich meldet `stop_hook_active` den Zustand aus
  loomux' State, den der dsh-Gast schreibt (jetzt schreibfähig).
- Audit: `hook/invoked`/`hook/result` im bekannten Zeilenformat in das
  Session-Log; ist der Dienst nicht erreichbar, schreibt das Plugin über
  `ctx.logger` und verliert sonst nichts.

### loomux `internal/hosts/dsh.go`

- Gastschiene wie `claude`: stdin-Feldnamen identisch (das Spike-Payload ist
  ohnehin Claude-kompatibel), Antwort-Hülle identisch, Exit-Codes 0/1/2 wie
  im Guard-Protokoll.
- Eigenschaft der Gasttafel: Ereignisliste um die sieben DSH-Ereignisse;
  `transcript_path` darf leer sein und wird nicht gebraucht; Schreibschranke,
  Check-Kette und State bleiben die bestehenden.
- CLI: gerufen wird wie bei jedem Gast: `loomux hook <event> --host dsh`;
  die Installation schreibt die dsh-Hook-Datei
  nach `.loomux/dsh/hooks.json` (Vorlage: dieselben Befehle wie der
  claude-Gast, Matcher `.*`, Timeout-Felder übernommen).

### Tests

- JS: `node:test` mit `--experimental-test-coverage`; Mock-Spawn, Mock-Agent
  und Mock-Sitzung; `sh ci/dsh-loomux-hooks.sh` schlägt unter 100 % Linien-
  und Funktionsabdeckung fehl (Bruderregel zu `dsh-loomux`),
  Linienabzweigungen sind nicht gehalten; eigener Workflow für Pull Requests,
  die das Paket berühren, und Publishing am Tag `dsh-loomux-hooks-v*`.
- Go: 100 % je Funktion; Replay-Fälle für den dsh-Gast unter
  `testdata/cases/` nach dem bestehenden Muster.
- Windows-Echtheitstest des Spawns mit einem Pfad, der `#` enthält
  (der loomux-Checkout selbst).

### Doku

- `docs/en/` und `docs/de/`: gleichnamige Seite zur DSH-Integration
  (Setup über plugin_manager, Hook-Auflösungskette, Fehlerbilder).
- `README.md` und `README.de.md`: DSH-Abschnitt beim Grep-Paket ergänzen;
  Roadmap-Zeile `Coming` → nach Merge in die Doku überführt.

## Risiken

- Die exakte Session-Log-API für die Audit-Zeilen ist erst bei der
  Implementierung belegt; der Rückfall auf `ctx.logger` ist Teil des Entwurfs,
  damit ein fehlender Audit-Kanal kein Funktionsverlust ist.
- Die Cordis-Seams sind alpha; die Seam-Namen stehen als zentrale
  Zeichenkettenkonstanten, damit ein Umbau eine einzige Stelle ist.
- `shell: true` und Anführungszeichen: der Spawn-Test fährt den loomux-eigenen
  Checkout-Pfad (`#GIT`) durch.
