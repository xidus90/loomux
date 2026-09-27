# Entwurf: Vollständige Integration der Antigravity-Hooks in Loomux

**Datum:** 2026-09-25  
**Verfasser:** Antigravity / Gemini  
**Status:** Umgesetzt (gemergt, Stufe 4a-2; Akte `parity/stufe-4a-2.md`)  
**Bezug:** Fusions-Spec Nachtrag #21 und #23, `specs-ul/2026-09-10-antigravity-hook-messung.md`, Stufe 4a-2 / 4e  
**Überarbeitet:** 2026-09-25 nach drei Reviews und der Probe mit agy 1.2.11: Antwort an den Wirt als ein Adapter, `run_command` und `send_command_input`, Post-Edit aus `toolCall`, flache `Stop`/`PreInvocation`  
**Nachgetragen:** 2026-09-27 nach dem Code-Review: `manage_task` im Matcher und in den Befehlsregeln, Zeilen vor der `Action`, Tipp-Eingaben nur als ganze Zeilen, ein Aufruf ohne Werkzeugnamen verweigert, `session-start` still bei einem späteren `invocationNum`, post-edit über `hosts.WriteContext`  

---

## 1. Motivation und Ziel

Loomux vereint Wächter, Prüfkette und Wissenssystem in einem einzigen Go-Binary (*One Go binary*). Für Claude Code sind Lebenszyklus-Hooks (`SessionStart`, `PreToolUse`, `PostToolUse`, `Stop`, `SubagentStart`, `SubagentStop`) bereits vollständig integriert.

Auf dem Branch `feat/init-antigravity` wurde die Setup-Mechanik (`loomux init`) für Antigravity gebaut:
- Schreibt `.agents/hooks.json` und `.agents/skills/`.
- `PreToolUse` und `PostToolUse` wurden verdrahtet. Der Live-Durchstich vom 2026-09-25 mit `agy 1.2.8` belegte, dass der Wächter mit Exit 2 unter Antigravity zuverlässig sperrt.

Was für die vollständige Hook-Integration von Antigravity noch fehlt:
1. `writeAntigravityContext` in `internal/hosts/antigravity.go` ist noch ein Platzhalter (`ErrNoAdapter`), weshalb `PreInvocation` (Sitzungsstart) noch nicht in `table.go` eingetragen ist.
2. Jeder Hook außer dem Wächter beendet Blockaden und interne Fehler mit Exit-Code 2 oder 1, was Antigravity als Command-Fehler wertet und abbricht, statt den Agenten geordnet im Kontext weiterarbeiten zu lassen.
3. `run_command` steht im Matcher von `PreToolUse`, aber die Befehlsregeln (etwa gegen `git push`) prüfen nur `Bash` und `PowerShell`.
4. Post-Edit liest bei Antigravity kein Ziel. Die frühere Annahme, `PostToolUse` trage weder `toolCall` noch Dateipfade, hat die Probe mit agy 1.2.11 widerlegt: `toolCall` ist dabei.
5. `table.go` muss um `PreInvocation` (Sitzungsstart) und `Stop` (Stop-Tor) ergänzt werden, damit `loomux init` alle 4 Hooks in `.agents/hooks.json` generiert.

---

## 2. Architektur & Komponenten

### 2.1 Kontext-Emission (`internal/hosts/antigravity.go`)

Antigravity verarbeitet auf `stdout` keine Claude-spezifischen Felder wie `hookSpecificOutput.additionalContext`, sondern erwartet JSON im Format:
```json
{
  "injectSteps": [
    {
      "ephemeralMessage": "..."
    }
  ]
}
```

* **Signatur:** `func writeAntigravityContext(w io.Writer, event string, lines []string) error`
* **Verhalten:**
  * Wenn `len(lines) == 0`: Schreibt nichts auf `w` und liefert `nil`.
  * Wenn `lines` nicht leer: Verbindet die Zeilen mit `\n` und schreibt obiges JSON auf `w`.
  * I/O-Fehler werden gewrappt.

### 2.2 Antwort an den Wirt (`internal/hosts/answer.go`, `internal/cli/hook.go`)

Die Hooks bleiben wirtsneutral: `RunStop`, `RunPostEdit` und `SessionStart`
antworten mit 0, 1 oder 2 und schreiben den Grund nach stderr, wie für Claude
Code. `loomux hook` hält stdout zurück, bis der Code feststeht, kopiert
stderr mit (der Puffer zuerst, damit ein toter stderr den Grund nicht
kostet), fängt eine Panik ab und ruft einmal `hosts.Answer(host, event,
stdout, code, out, reason)`.

Gemessen mit agy 1.2.8 und 1.2.11 am 2026-09-25 (`parity/stufe-4a-2.md`):

* **Claude Code, Codex:** stdout und Code unverändert.
* **Antigravity, `pre-tool-use`:** unverändert, Exit 2 mit `denyEnvelope`
  verweigert den Aufruf.
* **Antigravity, `post-tool-use`:** Exit 2 bleibt: agy gibt stderr dem Modell
  als Warnung und bricht nicht ab. stdout fällt weg (agy erwartet `{}`); Exit 1
  wird 0. Nachtrag 2026-09-27: post-edit schreibt seinen Kontext bei Exit 0
  über `hosts.WriteContext` als `injectSteps`, `hosts.Answer` verwirft ihn;
  ob agy `injectSteps` auf PostToolUse liest, ist ungemessen. Bei Exit ≠ 0
  schreibt post-edit nichts auf stdout, und jeder Hinweis auf eine
  übersprungene Lane oder Datei steht auf stderr. Codex: `hosts.WriteContext`
  antwortet `ErrNoAdapter`, `post-tool-use --host codex` endet bei Code 0
  mit 1, sobald der Aufruf eine Datei nennt.
* **Antigravity, `stop` mit Code 2:** Exit 0 und
  `{"decision":"continue","reason":"<stderr des Tors>"}`; agy tritt erneut in
  die Schleife ein. `MaxBlocks` beendet mit Code 0 und damit ohne `continue`.
* **Antigravity, sonst:** Exit 0, stdout durchgereicht (`session-start` trägt
  dort seine `injectSteps`).
* **Unbekanntes Ereignis:** Exit 2 auf jedem Wirt; es kann der Eintrag einer
  Sperre sein.
* Scheitert das Schreiben der Stop-Entscheidung, endet der Hook mit 2.

`session-start` läuft auf `PreInvocation`, das vor jedem Modellaufruf feuert
und die Aufrufe in `invocationNum` zählt; `hosts.Payload.Repeat` ist gesetzt ab
`invocationNum > 0`, als Zahl oder als Dezimal-String gelesen (protojson kann
64-Bit-Zähler als String schicken). Gemessen am 2026-09-27 mit agy 1.2.11
(`parity/stufe-4a-2.md`): `PreInvocation` trägt `invocationNum` als JSON-Zahl,
0, 1, 2, 3 je Modellaufruf; die Zählung beginnt bei 0. Ungemessen bleiben die
String-Form eines 64-Bit-Zählers und die Breite des Felds.
Bei `Repeat` schreibt `session-start` keinen Kontext und belebt die Sitzung
nicht wieder: `sessions.Revive` läuft nur beim ersten Aufruf (Nachtrag
2026-09-27). Ein unlesbarer oder fehlender Wert gilt als erster Aufruf und
wiederholt nur Ankündigungen. Einen `SessionStart`-Hook kennt `hooks.json`
nicht.

### 2.3 `run_command`, `send_command_input` und `manage_task` (`internal/hooks/guard.go`)

Nachgemessen am 2026-09-25 mit agy 1.2.11: agy tippt eine Eingabe nicht über
`send_command_input`, sondern über `manage_task` mit `Action` `send_input` und
`Input` (`parity/stufe-4a-2.md`).

Stand 2026-09-27 (Nachtrag nach dem Code-Review): `commandTools` ist eine
Tabelle je Werkzeug mit den Argumentnamen seiner Zeile, ob ein Aufruf ohne
Zeile durchgeht, ob der Wert eine ganze Befehlszeile ist und welche `Action`
keine Zeile trägt: `Bash` und `PowerShell` `command`, `run_command`
`CommandLine` (gemessen) und `command_line`, `send_command_input` `Input`
(ungemessen), `manage_task` `Input` (gemessen mit agy 1.2.11) mit den stillen
Aktionen `list`, `status` und `kill`. Die Argumentnamen gelten ohne Rücksicht
auf die Schreibung, und jeder Treffer wird geprüft, gleich welche `Action` der
Aufruf nennt; eine stille `Action` entschuldigt nur das Fehlen einer Zeile, und
nur, wenn jeder Schlüssel `action` in jeder Schreibung einen stillen String
trägt. Ohne Zeile verweigert der Guard jeden Aufruf außer bei `Bash` und
`PowerShell`. Was `send_command_input` und `manage_task` tippen, geht nur als
ganze Zeilen durch: Der Wert endet auf `\n` oder `\r`, trägt kein
Steuerzeichen außer `\n` und `\r`, auch keinen Tab, und keine seiner Zeilen
endet auf `\` oder `` ` ``, mit denen bash und PowerShell fortsetzen; jede
nicht leere Zeile läuft durch die Befehlsregeln. Ein Wert unter einem
Zeilenschlüssel, der kein String ist, macht den Aufruf unprüfbar und wird
verweigert, außer bei `Bash` und `PowerShell`; ein leerer String trägt keine
Zeile. Ein Aufruf ohne Werkzeugnamen wird verweigert. Die drei agy-Werkzeuge
stehen im Matcher von `PreToolUse` (`internal/setup/hostfile/table.go`); die
Befehlsregeln stehen in `internal/hooks/guard.go`,
`TestEveryCommandToolIsInAPreToolUseMatcher` in
`internal/hooks/matcher_test.go` bindet beides aneinander.

### 2.4 Post-Edit liest den Aufruf (`internal/hooks/post_edit.go`)

Das `PostToolUse` von agy 1.2.11 trägt `toolCall` mit `args` neben `stepIdx`
und `error`, anders als sein Hook-Leitfaden sagt. `editedFiles` liest die Ziele
daraus mit `guard.Call` und `guard.WriteTargets`, wie `pre-tool-use`, und bei
Claude aus `tool_input`. Ein Aufruf mit `error` prüft nichts. Mehrere Ziele
eines Aufrufs teilen sich ein Budget (ein Budget 0 bleibt unbegrenzt), jede
Datei hat ihre eigene runID. Die frühere Ablage in `internal/sessions` entfällt.

### 2.5 Setup-Tabelle & Aktivierung (`internal/setup/hostfile`)

```go
case hosts.HostAntigravity:
    writers := "write_to_file|replace_file_content|multi_replace_file_content"
    hook := func(name string) string {
        return AntigravityBinary + " hook " + name + " --host antigravity --root .."
    }
    return []Entry{
        {Event: "PreInvocation", Command: hook("session-start"), Timeout: 20, Flat: true},
        {Event: "PreToolUse", Matcher: writers + "|run_command|send_command_input|manage_task", Command: hook("pre-tool-use"), Timeout: 15},
        {Event: "PostToolUse", Matcher: writers, Command: hook("post-tool-use"), Timeout: 60},
        {Event: "Stop", Command: hook("stop") + " --budget 270s", Timeout: 300, Flat: true},
    }
```

`Flat` schreibt den Handler direkt in die Liste des Ereignisses statt in einen
Block mit `hooks`: agy 1.2.11 verwirft die ganze Datei, wenn `Stop` oder
`PreInvocation` einen Block tragen (`command hook must specify 'command'`),
und der Wächter lädt dann nicht. `commandsOf` erkennt einen flachen Handler als
eigenen, damit ein zweiter Lauf nichts doppelt einträgt.

---

## 3. Fehlerbehandlung & Kantenfälle

| Kantenfall | Verhalten |
|---|---|
| Fehlende `conversationId` | `hosts.Read` liefert `SessionID = ""`; das Tor meldet es auf stderr, für Antigravity Exit 0. |
| `hooks = false`, unlesbare Konfiguration, `nothing was verified`, Panik | Code 0 oder 1 aus dem Hook, für Antigravity Exit 0; der Grund steht auf stderr. |
| Pfad mit Leerzeichen oder `cmd.exe`-Syntax in `%LOCALAPPDATA%` | `plan.go` schreibt keine Einträge und sagt es. |
| Gescheiterter Aufruf (`error` gesetzt) | Post-Edit prüft nichts. |
| `run_command`/`send_command_input`/`manage_task` ohne erkennbare Zeile (bei `manage_task` außer `list`, `status`, `kill`), eine Tipp-Eingabe ohne Zeilenende, mit Steuerzeichen oder mit einer Zeile, die auf `\` oder `` ` `` endet, ein Aufruf ohne Werkzeugnamen | Verweigert mit Exit 2. |
| Wiederholtes `PreInvocation` | `session-start` schreibt keinen Kontext und belebt die Sitzung nicht wieder. |
| `post-tool-use --host codex`, Aufruf nennt eine Datei (auch eine mit ignorierter Endung) | Exit 1 mit dem Adapterfehler (`ErrNoAdapter`); ohne Datei Exit 0 (Nachtrag 2026-09-27). |
| 3 aufeinanderfolgende Stop-Blockaden | `RunStop` gibt auf, leert den Zähler und beendet mit Exit 0, ohne `continue`. |

---

## 4. Teststrategie & 100 % Abdeckung

Gemäß der Regel aus `AGENTS.md` („Coverage is 100% per function“):
1. **`internal/hosts/`**: `writeAntigravityContext`; `Answer` für jeden Wirt,
   jedes Ereignis und jeden Code, dazu ein stdout, das nicht schreibt;
   `Repeat` aus `invocationNum` als Zahl und als Dezimal-String.
2. **`internal/hooks/`**: `checkTool` tabellengetrieben über `commandTools`
   (jedes Werkzeug verweigert einen Push, jedes außer `Bash` und `PowerShell`
   einen Aufruf ohne Zeile), Argumentnamen in jeder Schreibung, eine stille
   `Action` mit Zeile, Tipp-Eingaben ohne Zeilenende, mit Steuerzeichen oder
   mit einer Zeilenfortsetzung,
   ein Aufruf ohne Werkzeugnamen; `editedFiles` mit `toolCall`, `error` und
   ohne Ziel; Budget und runID über mehrere Dateien; `session-start` bei einem
   späteren `invocationNum` schreibt nichts und belebt die Sitzung nicht
   wieder.
3. **`internal/cli/`**: relatives `--root`, fehlerhafte Aufrufe, unbekanntes
   Ereignis, Panik, gehaltener Stop.
4. **`internal/setup/hostfile/`**: die 4 Einträge, die flache Form beim
   Schreiben und beim erneuten Zusammenführen.
5. **End-to-End**: `ci/gate.sh`; die agy-Probe in `parity/stufe-4a-2.md`.
