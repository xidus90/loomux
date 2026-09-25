# Entwurf: Vollständige Integration der Antigravity-Hooks in Loomux

**Datum:** 2026-09-25  
**Verfasser:** Antigravity / Gemini  
**Status:** Genehmigt  
**Bezug:** Fusions-Spec Nachtrag #21 und #23, `specs-ul/2026-09-10-antigravity-hook-messung.md`, Stufe 4a-2 / 4e  
**Überarbeitet:** 2026-09-25 nach einem Review: Antwort an den Wirt als ein Adapter, `run_command`, Pufferung je Aufruf  

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
4. `PostToolUse` bei Antigravity enthält in der Nutzlast weder `toolCall` noch Dateipfade, weshalb `PreToolUse` den geänderten Pfad im Sitzungs-State puffern muss.
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

Die Hooks selbst bleiben wirtsneutral: `RunStop`, `RunPostEdit` und
`SessionStart` antworten mit 0, 1 oder 2 und schreiben den Grund nach stderr,
wie für Claude Code. `loomux hook` hält stdout zurück, bis der Code feststeht,
kopiert stderr mit und ruft einmal `hosts.Answer(host, event, stdout, code,
out, reason)`. Damit gilt die Abbildung für jeden Austrittspfad, auch für
die frühen (`ReadModules`, fehlende `conversationId`, `nothing was
verified`), ohne einen Zweig je Pfad im Hook-Kern.

* **Claude Code, Codex:** stdout und Code unverändert.
* **Antigravity, `pre-tool-use`:** unverändert, Exit 2 mit `denyEnvelope`
  (am 2026-09-25 mit agy 1.2.8 gemessen: die Schreibaktion unterbleibt).
* **Antigravity, `stop` mit Code 2:** Exit 0 und
  `{"decision":"continue","reason":"<stderr des Tors>"}`. Der Grund trägt
  die roten Spuren und die Befunde der Subagenten, weil beide nach stderr
  gehen. Aus den Zeichenketten des Binarys gelesen, nicht in einem Lauf
  gemessen.
* **Antigravity, `post-tool-use` mit Code 2:** Exit 0 und
  `{"injectSteps":[{"ephemeralMessage":"<stderr>"}]}`. Ungemessen: der Kanal
  ist nur für `PreInvocation` belegt.
* **Antigravity, sonst:** Exit 0. Bei `post-tool-use` fällt stdout weg
  (Claudes `additionalContext`, das agy nicht liest), sonst wird es
  durchgereicht (`session-start` trägt dort seine `injectSteps`).
* Scheitert das Schreiben der Entscheidung, endet der Hook mit 2: ein
  gescheiterter Befehl kommt dem Halten am nächsten.
* `MaxBlocks` bleibt beim Tor: die Aufgabe-Runde endet mit Code 0 und damit
  ohne `continue`.

### 2.3 `run_command` (`internal/hooks/guard.go`)

`commandTools` ordnet jedem Shell-Werkzeug die Argumentnamen seiner
Befehlszeile zu: `Bash` und `PowerShell` `command`, `run_command`
`CommandLine`, `commandLine` und `command_line` — die drei Schreibweisen, die
`agy.exe` enthält. Welche agy wirklich sendet, ist ungemessen; deshalb wird
jede vorhandene geprüft, und ein `run_command` ohne eine davon wird
verweigert, statt die Befehlsregeln still auszuschalten.

### 2.4 PostToolUse Pfad-Pufferung (`internal/sessions/pending.go`)

Antigravity sendet bei `PostToolUse` nur `stepIdx`, `conversationId` und
`error`:
1. `PreToolUse` legt nach einer Erlaubnis jedes Ziel eines schreibenden
   Aufrufs ab: `sessions.RecordPendingEdit(root, conversationId, stepIdx,
   ziel)`, je Aufruf eine eigene Datei unter
   `.loomux/state/hooks/<safeName(id)>/pending/<stepIdx>/edit-*`. Zwei
   parallele Schreibaufrufe eines Schritts überschreiben sich so nicht.
   Scheitert das Ablegen, sagt der Hook es auf stderr und lässt den
   Aufruf zu: verloren ist die Nachprüfung, nicht die Sperre.
2. `PostToolUse` nimmt mit `sessions.TakePendingEdits` alle Ziele des
   Schritts und löscht nur die gelesenen Dateien, sodass ein Geschwister, das
   währenddessen ablegt, seine behält. Jedes Ziel, das auf der Platte steht,
   läuft durch die Post-Edit-Spuren, alle in einem Budget, jede mit eigener
   runID; der schlechteste Code gewinnt.
3. Ein gescheiterter Aufruf (`error` gesetzt) nimmt nichts: ein Geschwister
   seines Schritts kann noch nach seiner Datei kommen.
4. Jedes Post-Edit verwirft die Ablagen früherer Schritte
   (`DropPendingEditsBefore`); das setzt aufsteigende `stepIdx` voraus. So
   bleibt je Konversation höchstens ein Schritt liegen, denn für Antigravity
   ruft kein Ereignis `Forget`. Bei `[modules] hooks = false` wird nichts
   abgelegt.

**Ungemessen:** ob Pre und Post eines Aufrufs denselben `stepIdx` tragen und
ob `stepIdx` aufsteigt. Tragen Pre und Post verschiedene, findet Post nie
etwas und prüft still nichts. Bei zwei parallelen Aufrufen eines Schritts
prüft der erste erfolgreiche Post beide Dateien, die zweite womöglich vor
ihrem Schreiben.

### 2.5 Setup-Tabelle & Aktivierung (`internal/setup/hostfile/table.go`)

Die Hook-Einträge für Antigravity in `internal/setup/hostfile/table.go` werden um `PreInvocation` und `Stop` vervollständigt:

```go
case hosts.HostAntigravity:
    writers := "write_to_file|replace_file_content|multi_replace_file_content"
    hook := func(name string) string {
        return AntigravityBinary + " hook " + name + " --host antigravity --root .."
    }
    return []Entry{
        {Event: "PreInvocation", Command: hook("session-start"), Timeout: 20},
        {Event: "PreToolUse", Matcher: writers + "|run_command", Command: hook("pre-tool-use"), Timeout: 15},
        {Event: "PostToolUse", Matcher: writers, Command: hook("post-tool-use"), Timeout: 60},
        {Event: "Stop", Command: hook("stop") + " --budget 270s", Timeout: 300},
    }
```

Die erzeugte `.agents/hooks.json` bindet alle 4 Hooks im Namensraum `"loomux"`.

---

## 3. Fehlerbehandlung & Kantenfälle

| Kantenfall | Verhalten |
|---|---|
| Fehlende `conversationId` | `hosts.Read` liefert `SessionID = ""`; das Tor meldet es auf stderr, `hosts.Answer` beendet für Antigravity mit Exit 0. |
| `hooks = false`, eine unlesbare Konfiguration, `nothing was verified` | Code 0 oder 1 aus dem Hook, für Antigravity Exit 0; der Grund steht auf stderr. |
| Pfad mit Leerzeichen in `%LOCALAPPDATA%` | `plan.go` fängt dies ab und warnt, dass `cmd.exe` ungequotete Pfade teilen würde. |
| Abweichender `stepIdx` bei `PostToolUse` | Kein Eintrag für den Schritt; der Hook prüft nichts und endet mit Exit 0. Ungemessen, ob das vorkommt. |
| Zwei Schreibaufrufe in einem Schritt | Beide Ziele liegen ab; der erste Post prüft beide. |
| Gescheiterter Aufruf (`error` gesetzt) | Die Ziele des Schritts werden genommen und verworfen. |
| `run_command` ohne bekannte Befehlszeile | Verweigert mit Exit 2. |
| 3 aufeinanderfolgende Stop-Blockaden | `RunStop` gibt auf, leert den Zähler und beendet mit Exit 0, ohne `continue`. |

---

## 4. Teststrategie & 100 % Abdeckung

Gemäß der Regel aus `AGENTS.md` („Coverage is 100% per function“):
1. **`internal/hosts/`**:
   * `writeAntigravityContext` mit Zeilen, ohne Zeilen und bei fehlschlagendem Writer.
   * `Answer` für jeden Wirt, jedes Ereignis und jeden Code, dazu ein stdout, das nicht schreibt.
2. **`internal/sessions/`**:
   * `RecordPendingEdit` und `TakePendingEdits`: zwei Ziele eines Schritts, ein anderer Schritt, eine Traversal-ID, `Forget`, jeder Fehlerpfad.
3. **`internal/hooks/`**:
   * `checkTool` mit `run_command` unter jeder Schreibweise und ohne Befehlszeile.
   * `bufferPendingEdits` und `editedFiles`: jede Schreibweise, jeder Ausschluss, ein gescheiterter Aufruf.
4. **`internal/cli/`**:
   * `loomux hook stop --host antigravity` mit gehaltenem Tor und mit unlesbaren Modulen.
5. **`internal/setup/hostfile/`**:
   * Test der 4 Antigravity-Einträge in `table_test.go` und `merge_test.go`.
6. **End-to-End**:
   * Ausführen von `ci/gate.sh` bzw. `go run ./cmd/loomux check precommit`.
   * Offen, vom Menschen auszuführen: die Probe in `parity/stufe-4a-2.md` mit einem laufenden agy.
