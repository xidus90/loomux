# Befunde des Antigravity-Reviews beheben — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die gehaltenen Befunde des Code-Reviews auf master schließen: Der Guard fällt für jede Antigravity-Eingabe geschlossen aus, `init` bringt fehlende Werkzeuge in bestehende Hookdateien, post-edit antwortet über die hosts-Naht, session-start wiederholt nichts, und Doku und Arbeitspapiere stimmen mit dem Code.

**Architecture:** Vier Bereiche in fester Reihenfolge auf einem Zweig: Guard (`internal/hooks/guard.go`, `pretool.go`), Hookdateien (`internal/setup/hostfile`), post-edit (`internal/hooks/post_edit.go`, `internal/verify/report.go`, `internal/hosts`), session-start (`internal/hooks/hook_session_start.go`, `internal/hosts/antigravity.go`); zuletzt die Arbeitspapiere. Jeder Task ist ein Commit, die Doku steht im Commit ihrer Änderung.

**Tech Stack:** Go (Version aus `go.mod`), Standardbibliothek, `go test -overlay` für Mutationsproben, Git Bash und cmd.exe unter Windows 11, agy (Antigravity CLI) für die Messung in Task 0.

**Spec:** `docs/.superpowers/specs/2026-09-27-antigravity-review-findings-design.md` (mit den Nachträgen beim Planen: Tab verweigert, Zeilenfortsetzung verweigert, unlesbare und leere Zeilenwerte, Codex ab einer genannten Datei, Tests 1.6 und 4.1).

Worktree: `C:/Users/micro/Documents/#GIT/loomux/.worktrees/review-findings`, Zweig `fix/antigravity-review-findings`, geplant auf a7805df8 (v4.0.0) und auf `origin/master` 9428f0af (v4.1.0) rebased; `internal/` ist zwischen beiden gleich, die Doku-Anker sind auf 9428f0af nachgezogen. `<scratch>` ist das Scratchpad der ausführenden Sitzung.

## Global Constraints

- Coverage 100 % je Funktion; eine Ausnahme nur mit `//coverage:exempt <Grund>` direkt über `func`. Keine ist geplant.
- Code, Kommentare, Fehlermeldungen und Commit-Nachrichten englisch; Nutzerdoku in `docs/en` und `docs/de` gleich; Arbeitspapiere unter `docs/.superpowers/` deutsch.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Commits nach Conventional Commits; eine Nachricht nennt keinen Plan, keine Spec, keine Stufe, keinen Task und keine Befund-Kennung; Scope ist ein Codebereich.
- Autor und Committer ist der Mensch (`git log -1 --format='%an <%ae>'` nach jedem Commit); kein `Co-Authored-By`, keine Modellnennung.
- Niemand außer dem Menschen pusht. Ein Subagent pusht nie.
- Mehrzeilige Commit-Nachrichten per Write in eine Datei und `git commit -F`; die Ausgabe des pre-commit-Tors (`sh ci/gate.sh`) erst ganz in eine Datei, dann filtern.
- Backslashes in Go-Quellen, Tests und Commit-Texten nur über Write/Edit, nie per Heredoc oder `sed`; danach mit `grep` die Bytes nachlesen. Nicht-ASCII-Testzeichen als `\x..`/`\u....`-Escape im Go-Quelltext, nicht als rohes Zeichen.
- Vor jedem git-Schreibbefehl `git rev-parse --show-toplevel` gegen den Worktree-Pfad lesen.
- `go test -overlay` zusammen mit `-coverprofile` nimmt hier die echte Datei statt der Überlagerung: Mutationsproben ohne `-cover`, Coverage nur im echten Baum.
- Label `release:patch`; `CHANGELOG.md` wird nicht von Hand geändert, die Einträge stehen am Ende dieses Plans für den PR-Rumpf.
- `.githooks/pre-commit` verweigert einen Commit, dessen Gate-Eingaben vom Index abweichen: vor `git commit` alles Zugehörige stagen und keine ungestagten Reste an Go-Quellen, `go.mod`/`go.sum` oder der Konfiguration liegen lassen.
- `docs/*/migration.md` hält `internal/plancheck` gegen die Fusions-Spec: in einer Tabellenzelle kein rohes `|` (als `\|` schreiben, ein Backtick-Span schützt nicht), kein Fettdruck in der ersten Zelle außer der Stufenkennung, Status- und Prioritätszelle unverändert. Nach jeder Änderung an `migration.md` oder der Fusions-Spec `go test ./internal/plancheck/ -count=1`.
- Doku-Anker sind auf `origin/master` 9428f0af (v4.1.0) gesetzt; ein Ausführender sucht nach den zitierten Worten, nicht nach Zeilennummern, und liest vor dem ersten Task `git log -1 origin/master` nach `git fetch`. Ist master weiter, erst rebasen und die Anker des Tasks prüfen.

## Review Focus

Eingaben, die die Spec meint, aber nicht ausdrücklich prüft; zu jeder steht der Test im genannten Task:

1. **Ein Tab in einer Tipp-Eingabe.** Eine interaktive Shell vervollständigt am Tab: `"git pus\t origin main\n"` läuft dort als `git push origin main`. Erwartet: verweigert. Test in Task 1 (`TestATypedInputIsJudgedOnlyAsWholeLines`).
   Ebenso eine **Zeilenfortsetzung**: bash setzt eine Zeile nach `\`, PowerShell nach `` ` `` in der nächsten fort, `"loomux \\\ninit\n"` (Go-Schreibweise) läuft dort als `loomux init`. Geteilt wären beide Zeilen harmlos; heute fängt `writesConfiguration` den ungeteilten Wert. Erwartet: verweigert. Test in Task 1 (`TestATypedInputIsJudgedOnlyAsWholeLines`).
2. **Ein Zeilenschlüssel mit einem Wert, der kein String ist, oder einem leeren String.** `{"Action":"kill","Input":42}` darf nicht als still durchgehen, `{"Action":"kill","Input":""}` soll still bleiben, und ein unlesbarer `command` neben einem lesbaren `COMMAND` darf den lesbaren nicht verdecken. Tests in Task 1 (`TestAQuietActionExcusesOnlyAMissingLine`, `TestAnUnreadableLineValueHidesNoOther`).
3. **Zwei Werte in einem Aufruf, ein Fragment und eine ganze verbotene Zeile.** Beide Gründe müssen kommen, in der Reihenfolge der sortierten Schlüssel. Test in Task 1 (`TestATypedInputIsJudgedOnlyAsWholeLines`, Fall „two values“).
4. **`--host codex` mit einer Datei, deren Endung die Presets ignorieren.** Die Datei ist genannt, also Exit 1 mit dem Adapterfehler; ohne Datei 0. Test in Task 10 (`TestPostEditAnswersCodexWithItsMissingAdapter`).
5. **JSON `null` als pre-tool-use-Nutzlast.** Dekodiert zu einer nil-Map und bleibt bei der Schranke, die es als „kein Objekt“ verweigert, nicht bei der Policy. Test in Task 2 (`TestAPayloadThatIsNoObjectKeepsTheBarriersWording`).

## Hinweise für den Controller

- **Task 0 führt der Mensch aus.** Ein Agent darf agy nicht starten. Der Controller nennt dem Nutzer den Befehl aus Task 0 Schritt 4 (mit `!`-Präfix in Git Bash) und dispatcht Task 4 erst nach dem Urteil „weiter“. Task 1–3 und 5–12 hängen nicht daran und können vorher laufen. Landet Task 4 deshalb nach Task 5–12, wird sein Commit vor dem PR an seinen Platz hinter Task 3 gesetzt, ohne interaktives Rebase: `git rebase -i` mit einem per Skript geschriebenen Todo (`GIT_SEQUENCE_EDITOR`) oder Cherry-Picks auf einen neuen Zweig ab dem Commit von Task 3. Der Skill `release-pr` gruppiert die Commits vor dem PR ohnehin neu.
- **Subagenten** nach der Regel des Nutzers mit `model: "opus"`, `effort: "low"`, beides ausdrücklich. Implementierer arbeiten im Worktree, nie im Haupt-Checkout; nach jedem Task `git log -1 --format='%an <%ae>'` lesen.
- **Geteilte Doku-Stellen:** `docs/*/hooks.md` Abschnitt 7 ändern Task 1 („Paths and command lines“), Task 2 („Never exit 1“, Mermaid-Kante) und Task 4 (neuer Absatz nach „Paths are compared relative to the root“); den Antigravity-Punkt in `docs/*/cli-reference.md` Abschnitt 11 ändern Task 1 (Satz über `run_command`) und Task 4 (Satz über den älteren Matcher); den post-edit-Teil ändern Task 9 und Task 10 nacheinander; den session-start-Satz Task 11 und Task 12. Jeder Task setzt am Text an, nicht an Zeilennummern, und auf dem Stand des vorigen auf.
- **Task 13 läuft zuletzt** und braucht das Ergebnis aus Task 0 (Version und Ausgang A oder B).
- **Stop-Hook während eines Hintergrund-Tasks:** Meldet er Rot, während ein Implementierer mitten in seiner RED-Phase ist, zählt nur das Tor des Commits.
- **Vor dem PR:** `agy --version` erneut lesen; weicht es von der Version aus Task 0 ab, Task 0 wiederholen. Dann der Skill `release-pr`.

---

### Task 0: agy mit zwei PreToolUse-Blöcken messen (Vorbedingung für Task 4)

Kein Code-Task. Der Controller führt ihn aus, bevor Task 4 dispatcht wird. Ein Agent darf keinen Agenten starten (der Auto-Modus verweigert das; so lief auch die Probe vom 2026-09-25). Also nennt der Controller dem Nutzer den Befehl aus Schritt 4 und wartet auf sein Ergebnis.

**Vorlagen (nur lesen):** `docs/.superpowers/parity/stufe-4a-2.md` ab „**Inzwischen gemessen**“ (~628) und „Probe mit agy 1.2.11, 2026-09-25“ (~646), `docs/.superpowers/specs-ul/2026-09-10-antigravity-hook-messung.md` (Befund 2: ein Hook läuft aus `.agents/`, ein nackter relativer Befehl wird nicht gefunden; Befund 3: projektlokale Hooks laden nur in einem vertrauten Ordner; Befund 4: im Print-Modus braucht es `--dangerously-skip-permissions`) und `docs/.superpowers/specs/2026-09-25-antigravity-hooks-integration-design.md`.

Bekannte Fakten, gelesen am 2026-09-27:
- `agy --version` gibt `1.2.11`.
- `~/.gemini/antigravity-cli/settings.json` führt `trustedWorkspaces: ["C:\\Users\\micro", …]`. Der Scratchpad liegt darunter und ist damit vertraut.
- agy loggt nach `~/.gemini/antigravity-cli/log/cli-<JJJJMMTT_hhmmss>.log`; `--log-file <pfad>` legt das Log woanders hin. Das Laden steht dort als `hooks_manager.go:53] loaded <n> named hooks from <m> hooks.json file(s)`. Ein Parse-Fehler steht als `hooks.go:118] failed to parse hooks for … : invalid hook …`, ein leer gewordener Hook als `declarative_config_loader.go:… skipping component during resolution: empty component: pre-tool hook "<name>" is empty`.

- [ ] **Schritt 1: Version festhalten.** `agy --version` ausführen und die Ausgabe mit Datum und Uhrzeit notieren. Gibt es eine andere Version als 1.2.11, gilt die Messung für diese Version, und die Version steht in der Notiz.
- [ ] **Schritt 2: Probe im Scratchpad anlegen.** Der Ordner ist `P=<scratchpad>/agy-two-blocks`, nie direkt unter `C:\Users\micro`. Beide Dateien mit Write schreiben, nicht per Heredoc, weil sie Backslashes tragen.
  `P/.agents/log.cmd`:
  ```bat
  @echo off
  echo ---- block %1 >> "%~dp0log.txt"
  more >> "%~dp0log.txt"
  echo. >> "%~dp0log.txt"
  exit /b 0
  ```
  `P/.agents/hooks.json` (`<P>` ist der absolute Pfad mit Schrägstrichen; beide Blöcke rufen dasselbe Skript, nur mit anderem Kennbuchstaben, so wie Task 4 später denselben Befehl zweimal schreibt):
  ```json
  {
    "loomux": {
      "PreToolUse": [
        {
          "matcher": "write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input",
          "hooks": [{"type": "command", "command": "<P>/.agents/log.cmd A", "timeout": 15}]
        },
        {
          "matcher": "manage_task",
          "hooks": [{"type": "command", "command": "<P>/.agents/log.cmd B", "timeout": 15}]
        }
      ],
      "PreInvocation": [
        {"type": "command", "command": "<P>/.agents/log.cmd I", "timeout": 15}
      ]
    }
  }
  ```
  `PreInvocation` steht flach, als Liste von Handlern ohne `hooks`-Hülle, wie in der versionierten `.agents/hooks.json`: Eine gruppierte Form lässt agy 1.2.11 die ganze Datei verwerfen (`parity/stufe-4a-2.md`, Probe vom 2026-09-25) und nähme die Hauptmessung mit. Der Eintrag zeichnet im selben Lauf die `PreInvocation`-Nutzlast auf, denn ob sie `invocationNum` überhaupt trägt, ist ungemessen: die einzige aufgezeichnete (`testdata/cases/2c-payloads/agy-inv.json`, agy 1.2.2) trägt nur `conversationId` und `stepIdx`.
  Gegenprobe von Hand: `echo {"probe":1} | "<P>/.agents/log.cmd" X` in cmd.exe, danach muss `log.txt` einen Eintrag `---- block X` mit der Nutzlast tragen. Danach `log.txt` löschen.
- [ ] **Schritt 3: agy einen Aufruf aus Block 2 machen lassen.** `manage_task` kam am 2026-09-25 für das Beenden eines Hintergrundbefehls (`{"Action":"kill","TaskId":…}`).
- [ ] **Schritt 4: Den Befehl dem Nutzer nennen** (Git Bash):
  ```sh
  cd "<P>" && agy -p "Create a.txt containing A with write_to_file. Then start the shell command 'ping -n 60 127.0.0.1' in the background with run_command, and stop it again with manage_task before it finishes. Then stop." --add-dir "<P als Windows-Pfad>" --dangerously-skip-permissions --print-timeout 280s --log-file "<P>/agy.log"; cat .agents/log.txt
  ```
- [ ] **Schritt 5: Auswerten.** Dazu `.agents/log.txt` und `<P>/agy.log` lesen, und zwar ganz, nicht gefiltert.
  - **Weiter mit Task 4**, wenn alles davon gilt: `agy.log` hat zwei `loaded … from N`-Zeilen, und die zweite (nach `workspaceDirs=[<P>`) zählt eine Datei mehr als die erste (am 2026-09-25: 2, dann 3; die +1 ist `<P>/.agents/hooks.json`, denn agy loggt keinen Pfad der Dateien, die es lädt); keine `failed to parse hooks`- und keine `empty component`-Zeile nennt `<P>` oder `loomux`. Die Warnung `failed to parse hooks for plugin at …superpowers` und `empty component: pre-tool hook "command_assessor"` stehen in jedem Lauf und betreffen `<P>` nicht. `log.txt` hat mindestens einen Eintrag `block A` mit `write_to_file` oder `run_command` im `toolCall` und mindestens einen Eintrag `block B` mit `"name":"manage_task"`, und kein Eintrag `block A` trägt `"name":"manage_task"`, kein Eintrag `block B` trägt `write_to_file` oder `run_command`.
  - **Nicht eindeutig**, wenn `log.txt` nur Einträge von `block A` hat und agy laut seiner Ausgabe `manage_task` nie gerufen hat. Dann einmal mit einer Kontrolle nachmessen, die den Mechanismus auf ein Werkzeug stellt, das sicher kommt: Block 1 hat den Matcher `write_to_file|replace_file_content|multi_replace_file_content`, Block 2 hat `run_command`, der Prompt bleibt derselbe. Kommt `block B` mit `run_command`, gilt das als „weiter“, mit dieser Einschränkung in der Notiz.
  - **Stopp, den Nutzer fragen, Task 4 nicht bauen** (Spec 2.1: „Scheitert die Messung, wird 2.1 nicht gebaut“): ein Parse-Fehler oder ein `empty component` für diese Datei; gar keine Nutzlast; nur `block A`, obwohl agy `manage_task` gerufen hat (Block 2 wurde also nicht geladen oder nicht gewählt); nur `block B`, obwohl `a.txt` entstand (Block 1 wurde nicht geladen); oder ein Werkzeugaufruf, der unter beiden Blöcken protokolliert ist, obwohl die Matcher disjunkt sind (agy beachtet die Matcher nicht; dann liefe ein angehängter Block jeden Aufruf doppelt). Dem Nutzer die Log-Zeilen vorlegen. Task 5 und Task 6 hängen nicht an der Messung.
  - **Erst den Aufbau prüfen, nicht agy:** Fehlt die zweite `loaded`-Zeile oder bleibt der Zähler gleich, ist der Aufbau schuld (Vertrauen, Pfad), nicht agy: Aufbau prüfen und wiederholen, bevor Ausgang B notiert wird. Nennt ein Parse-Fehler `<P>` und `PreInvocation`, den `PreInvocation`-Eintrag entfernen und wiederholen, bevor über Task 4 geurteilt wird.
  - Nachtrag 2026-09-28: Eine einzelne `loaded N named hooks from N`-Zeile nach `workspaceDirs=[<P>…]` zählt `<P>` schon mit und ist den zwei Zeilen mit der +1 gleichwertig. Beobachtet am 2026-09-27 mit agy 1.2.11: `<P>` stand schon beim Anlegen des Servers in `workspaceDirs`, agy lud also einmal, `loaded 3 named hooks from 3 hooks.json file(s)`; die Nutzlasten in `log.txt` bestätigten beide Blöcke.
  - **`block I`** (unabhängig vom Urteil über Task 4): die `PreInvocation`-Nutzlasten der Reihe nach lesen und festhalten, ob sie `invocationNum` tragen, in welcher Form (Zahl oder String) und mit welchem Wert beim ersten und beim zweiten Modellaufruf. Fehlt `block I` ganz, ist das Feld weiter ungemessen.
- [ ] **Schritt 6: Ergebnis für Task 13 festhalten**, noch nicht in das Arbeitspapier: Datum, Uhrzeit, `agy --version`, Aufbau (die zwei Matcher), der Prompt, die beiden `loaded …`-Zeilen, die Kopfzeilen der Einträge aus `log.txt` (`block A` + Werkzeugname, `block B` + Werkzeugname), was die `block I`-Nutzlasten zu `invocationNum` zeigen (Feld vorhanden oder nicht, Form, erste Werte) und das Urteil. Task 13 überträgt das als datierten Nachtrag nach `docs/.superpowers/parity/stufe-4a-2.md`.

---

### Task 1: Eine Werkzeugtabelle, Schlüssel ohne Schreibung, Zeilen zuerst, nur ganze Zeilen

**Files:**
- Modify: `internal/hooks/guard.go:7-19` (Imports), `internal/hooks/guard.go:129-170` (`commandTools`, `judgedOrRefused`, `quietActions`, `commandLines`), `internal/hooks/guard.go:256-271` (Befehlsblock in `checkTool`)
- Test: `internal/hooks/guard_test.go:180-241` (ersetzt `TestGitPushIsRefusedOnRunCommand`, erweitert `TestManageTaskSendInputIsJudgedByTheCommandRules`, ersetzt `TestARunCommandWithoutALineIsRefused`)
- Doku: `docs/en/hooks.md:357-371`, `docs/de/hooks.md:373-387`, `docs/en/cli-reference.md:1318-1321`, `docs/de/cli-reference.md:1374-1378`, `docs/en/migration.md:38`, `docs/de/migration.md:39`

**Interfaces:**
- Consumes: nichts (erste Aufgabe)
- Produces:
  - `type commandTool struct { keys []string; lineOptional bool; whole bool; quiet []string }`
  - `var commandTools map[string]commandTool` (Schlüssel: `Bash`, `PowerShell`, `run_command`, `send_command_input`, `manage_task`) — Task 3 iteriert über die Schlüssel
  - `func commandLines(tool commandTool, input map[string]any) (values []string, ok bool)`
  - `func quietAction(tool commandTool, input map[string]any) bool`
  - `func typedLines(name, value string) (lines []string, refusal string)`
  - `judgedOrRefused` und `quietActions` entfallen.

- [ ] **Step 1: Failing test schreiben**

In `internal/hooks/guard_test.go` den Block von `// Antigravity's run_command is judged by the same command rules, under every` (Zeile 180) bis zur schließenden Klammer von `TestARunCommandWithoutALineIsRefused` (Zeile 241) vollständig durch folgenden Block ersetzen. Die Imports (`regexp`, `strings`, `config`) sind schon da. Backslashes nur per Edit/Write schreiben, nie per Heredoc, danach mit `grep -n 'x1b\|u009b' internal/hooks/guard_test.go` nachlesen, und `grep -n 'loomux \\\\' internal/hooks/guard_test.go` muss die Zeile mit `"loomux \\\ninit\n"` zeigen (drei Backslashes vor dem `n`).

```go
// Every command tool is judged by the same command rules, under each of its
// keys in any case. The loop runs over the table itself, so a tool that joins
// it is held to the rules without a new test.
func TestGitPushIsRefusedOnEveryCommandTool(t *testing.T) {
	root := t.TempDir()
	checked := 0
	for name, tool := range commandTools {
		line := "git push origin main"
		if !tool.whole {
			line += "\n"
		}
		for _, key := range tool.keys {
			for _, spelling := range []string{key, strings.ToLower(key), strings.ToUpper(key)} {
				reasons := checkTool(root, name, map[string]any{spelling: line}, config.Policy{})
				if len(reasons) != 1 || reasons[0] != "Whether commits reach the remote is a human's decision." {
					t.Fatalf("[%s %s] reasons %v", name, spelling, reasons)
				}
				checked++
			}
		}
	}
	if checked == 0 {
		t.Fatal("commandTools names no key, so nothing was judged")
	}
}

// A second spelling of a key is judged beside the first, not in its place:
// a harmless line under one cannot carry a forbidden one under the other.
func TestEveryCasingOfALineKeyIsJudged(t *testing.T) {
	root := t.TempDir()
	const push = "Whether commits reach the remote is a human's decision."
	for tool, input := range map[string]map[string]any{
		"manage_task": {"Action": "send_input", "Input": "echo hi\n", "input": "git push origin main\n"},
		"run_command": {"CommandLine": "go test ./...", "COMMANDLINE": "git push origin main"},
	} {
		if reasons := checkTool(root, tool, input, config.Policy{}); len(reasons) != 1 || reasons[0] != push {
			t.Fatalf("[%s] reasons %v", tool, reasons)
		}
	}
}

// agy 1.2.11 types into a task run_command left open with manage_task,
// Action send_input and the line under Input (measured 2026-09-25): that line
// is judged. list, status and kill carry none and pass; any other Action,
// or none, is judged, so one without Input is refused.
func TestManageTaskSendInputIsJudgedByTheCommandRules(t *testing.T) {
	root := t.TempDir()
	const push = "Whether commits reach the remote is a human's decision."
	send := map[string]any{"Action": "send_input", "Input": "git push origin main\n", "TaskId": "c/task-6"}
	if reasons := checkTool(root, "manage_task", send, config.Policy{}); len(reasons) != 1 || reasons[0] != push {
		t.Fatalf("send_input: reasons %v", reasons)
	}
	for _, action := range []string{"kill", "list", "status"} {
		quiet := map[string]any{"Action": action, "TaskId": "c/task-2"}
		if reasons := checkTool(root, "manage_task", quiet, config.Policy{}); len(reasons) != 0 {
			t.Fatalf("%s: reasons %v, want none", action, reasons)
		}
	}
	// An action agy's schema does not name may carry a line all the same.
	other := map[string]any{"Action": "sendInput", "Input": "git push origin main\n"}
	if reasons := checkTool(root, "manage_task", other, config.Policy{}); len(reasons) != 1 || reasons[0] != push {
		t.Fatalf("sendInput: reasons %v", reasons)
	}
	for _, input := range []map[string]any{{"Action": "send_input"}, {"TaskId": "c/task-6", "Text": "git push"}} {
		reasons := checkTool(root, "manage_task", input, config.Policy{})
		if len(reasons) != 1 || !strings.Contains(reasons[0], "no command line in this manage_task call") {
			t.Fatalf("%v: reasons %v", input, reasons)
		}
	}
	// Whole lines that break no rule pass, an empty one included.
	for _, line := range []string{"y\n", "echo hi\r\n", "\n"} {
		typed := map[string]any{"Action": "send_input", "Input": line, "TaskId": "c/task-6"}
		if reasons := checkTool(root, "manage_task", typed, config.Policy{}); len(reasons) != 0 {
			t.Fatalf("%q: reasons %v, want none", line, reasons)
		}
	}
}

// The lines a call carries are judged whatever its Action says; a quiet
// Action only excuses a call that carries none, and only when every key
// spelled action names a quiet one.
func TestAQuietActionExcusesOnlyAMissingLine(t *testing.T) {
	root := t.TempDir()
	const push = "Whether commits reach the remote is a human's decision."
	for _, input := range []map[string]any{
		{"Action": "kill", "Input": "git push origin main\n"},
		{"Action": "status", "Input": "git push origin main\n"},
		{"Action": "kill", "action": "send_input", "Input": "git push origin main\n"},
	} {
		if reasons := checkTool(root, "manage_task", input, config.Policy{}); len(reasons) != 1 || reasons[0] != push {
			t.Fatalf("%v: reasons %v", input, reasons)
		}
	}
	for _, input := range []map[string]any{
		{"Action": "kill", "action": "send_input"},
		{"Action": "kill", "action": 42},
		{"Action": 42},
		// A line the guard cannot read might be the one the tool types, and
		// an empty one types nothing: neither is a line to judge.
		{"Action": "kill", "Input": 42},
		{"Action": "kill", "Input": nil},
		{"Action": "send_input", "Input": ""},
	} {
		reasons := checkTool(root, "manage_task", input, config.Policy{})
		if len(reasons) != 1 || !strings.Contains(reasons[0], "no command line in this manage_task call") {
			t.Fatalf("%v: reasons %v", input, reasons)
		}
	}
	for _, input := range []map[string]any{
		{"Action": "kill", "TaskId": "t"},
		{"Action": "list"},
		{"action": "status"},
		{"Action": "kill", "ACTION": "list"},
		{"Action": "kill", "Input": ""},
	} {
		if reasons := checkTool(root, "manage_task", input, config.Policy{}); len(reasons) != 0 {
			t.Fatalf("%v: reasons %v, want none", input, reasons)
		}
	}
}

// A value the guard cannot read does not stop the others from being judged:
// a Bash call, whose line is optional, still has its push found beside it.
func TestAnUnreadableLineValueHidesNoOther(t *testing.T) {
	root := t.TempDir()
	input := map[string]any{"command": 42, "COMMAND": "git push origin main"}
	if reasons := checkTool(root, "Bash", input, config.Policy{}); len(reasons) != 1 || reasons[0] != "Whether commits reach the remote is a human's decision." {
		t.Fatalf("reasons %v", reasons)
	}
}

// What an agent types into an open task is judged only as whole lines
// without control characters: a fragment may be finished by the next call,
// a backspace or an escape sequence edits the line, and a continuation lets
// the next line finish it, after the guard has read it. All are refused, a
// quiet Action no excuse for any.
func TestATypedInputIsJudgedOnlyAsWholeLines(t *testing.T) {
	root := t.TempDir()
	const rule = "loomux judges what an agent types into a task only as whole lines without control characters; this "
	cases := []struct{ input, want string }{
		{"git pu", "does not end its line, so it refuses"},
		{"x", "does not end its line, so it refuses"},
		{"\x03", "does not end its line, so it refuses"},
		{"\x1b[A\n", "carries a control character, so it refuses"},
		{"git pusx\bh origin main\n", "carries a control character, so it refuses"},
		{"ls\x00\n", "carries a control character, so it refuses"},
		{"ls\x7f\n", "carries a control character, so it refuses"},
		{"ls\u009b2J\n", "carries a control character, so it refuses"},
		// An interactive shell completes a word at a tab: "git pus<Tab>"
		// becomes "git push" after the guard has read "git pus".
		{"git pus\t origin main\n", "carries a control character, so it refuses"},
		// A backslash or a backtick at a line end continues the line: bash
		// and PowerShell run "loomux init" from the two lines below.
		{"loomux \\\ninit\n", "does not end its line, so it refuses"},
		{"loomux `\ninit\n", "does not end its line, so it refuses"},
		{"git \\\npush origin main\n", "does not end its line, so it refuses"},
	}
	for _, tool := range []string{"manage_task", "send_command_input"} {
		for _, c := range cases {
			input := map[string]any{"Action": "send_input", "Input": c.input}
			reasons := checkTool(root, tool, input, config.Policy{})
			if len(reasons) != 1 || reasons[0] != rule+tool+" input "+c.want {
				t.Fatalf("[%s] %q: reasons %v", tool, c.input, reasons)
			}
		}
	}
	quiet := map[string]any{"Action": "kill", "Input": "x"}
	if reasons := checkTool(root, "manage_task", quiet, config.Policy{}); len(reasons) != 1 || reasons[0] != rule+"manage_task input does not end its line, so it refuses" {
		t.Fatalf("kill with a fragment: reasons %v", reasons)
	}
	// Each value is judged on its own: a fragment under one spelling does not
	// stop the whole line under the other from being judged.
	both := map[string]any{"Action": "send_input", "Input": "git pu", "input": "git push origin main\n"}
	want := []string{rule + "manage_task input does not end its line, so it refuses", "Whether commits reach the remote is a human's decision."}
	if reasons := checkTool(root, "manage_task", both, config.Policy{}); !slices.Equal(reasons, want) {
		t.Fatalf("two values: reasons %v", reasons)
	}
	// A whole command line is no typing: run_command ends no line and may
	// carry what a shell reads as a control character.
	if reasons := checkTool(root, "run_command", map[string]any{"CommandLine": "printf '\x1b[0m'"}, config.Policy{}); len(reasons) != 0 {
		t.Fatalf("run_command: reasons %v, want none", reasons)
	}
}

// Several lines typed in one call are split at every line end and each is
// judged on its own, so a rule anchored at the start of a line reaches the
// second one too. The rule is ^rm\s, not the built-in push rule: its (^|\s)
// matches a line end, so a push in line 2 would be caught without any
// splitting and the test could not fail.
func TestATypedInputIsJudgedLineByLine(t *testing.T) {
	root := t.TempDir()
	policy := config.Policy{Commands: []config.CommandRule{{
		Regex: regexp.MustCompile(`^rm\s`), Source: `^rm\s`, Reason: "no rm at the start of a line",
	}}}
	for _, typed := range []string{"echo hi\nrm -rf x\n", "echo hi\r\nrm -rf x\r\n", "echo hi\rrm -rf x\r"} {
		for _, tool := range []string{"manage_task", "send_command_input"} {
			reasons := checkTool(root, tool, map[string]any{"Action": "send_input", "Input": typed}, policy)
			if len(reasons) != 1 || reasons[0] != "no rm at the start of a line" {
				t.Fatalf("[%s] %q: reasons %v", tool, typed, reasons)
			}
		}
	}
}

// A tool whose line is not optional is refused when the guard finds no line
// under any of its keys, not waved through; a Bash or PowerShell call without
// a command stays unjudged, as it was. The loop runs over the table, so the
// closed default reaches a tool that joins it.
func TestEveryCommandToolWithoutALineIsRefused(t *testing.T) {
	root := t.TempDir()
	checked := 0
	for name, tool := range commandTools {
		if tool.lineOptional {
			if reasons := checkTool(root, name, map[string]any{}, config.Policy{}); len(reasons) != 0 {
				t.Fatalf("[%s] reasons %v, want none", name, reasons)
			}
			continue
		}
		reasons := checkTool(root, name, map[string]any{"Cmd": "git push"}, config.Policy{})
		if len(reasons) != 1 || reasons[0] != "loomux found no command line in this "+name+" call, so it cannot judge it and refuses" {
			t.Fatalf("[%s] reasons %v", name, reasons)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no command tool in the table refuses a call without a line")
	}
}
```

- [ ] **Step 2: Rot prüfen**

Run: `go test ./internal/hooks/ -run 'CommandTool|Casing|QuietAction|TypedInput|ManageTask|Unreadable' -count=1 > <scratch>/t1-red.txt 2>&1; tail -6 <scratch>/t1-red.txt`

Expected (die zwei Tabellenschleifen greifen auf Felder zu, die es noch nicht gibt, also baut das Paket nicht):
```
guard_test.go:188:12: tool.whole undefined (type []string has no field or method whole)
guard_test.go:191:28: tool.keys undefined (type []string has no field or method keys)
guard_test.go:392:11: tool.lineOptional undefined (type []string has no field or method lineOptional)
FAIL	github.com/xidus90/loomux/internal/hooks [build failed]
```
(Nach den Nachträgen zu Tab, unlesbaren und leeren Werten neu gemessen, mit genau den Blöcken aus Step 1 und Step 3 per `go test -overlay`: Rot wie oben, Grün für `internal/hooks`, `internal/cli`, `internal/cases`, `internal/hosts`, `internal/brain/guard`, `go vet` still, `commandLines`, `quietAction`, `typedLines` und `checkTool` je 100.0 % in einer Baumkopie. Nach dem Nachtrag zur Zeilenfortsetzung erneut per Overlay gemessen: Rot mit den drei Zeilen oben (188, 191, 392), Grün für den gefilterten Lauf und für `internal/hooks`, `internal/cli`, `internal/cases`, `internal/brain/guard`, `go vet` still, die vier Funktionen je 100.0 % in einer Baumkopie, Mutationen 7 und 13 wie in Step 5.)
Die Verhaltensrotphase ist beim Planen zusätzlich gemessen (die zwei Schleifentests vorübergehend entfernt): `--- FAIL: TestEveryCasingOfALineKeyIsJudged` (`[manage_task] reasons []`), `--- FAIL: TestAQuietActionExcusesOnlyAMissingLine` (`map[Action:kill Input:git push origin main …]`), `--- FAIL: TestATypedInputIsJudgedOnlyAsWholeLines` (`[manage_task] "git pu": reasons []`), `--- FAIL: TestATypedInputIsJudgedLineByLine` (`[manage_task] "echo hi\nrm -rf x\n": reasons []`); `TestManageTaskSendInputIsJudgedByTheCommandRules` lief grün.

- [ ] **Step 3: Implementieren**

In `internal/hooks/guard.go` den Importblock (Zeilen 7-19) ersetzen durch:

```go
import (
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/shellwords"
)
```

Den Block von `// commandTools are the tools that run a shell line, each with the argument` (Zeile 129) bis zur schließenden Klammer von `commandLines` (Zeile 170) vollständig ersetzen durch:

```go
// commandTool says how the guard finds the shell lines in a call to a tool
// that runs them. Every zero value is the closed case: a tool added with
// nothing set refuses a call without a line and has what it carries judged
// as typing, whole lines only.
//
// Antigravity's run_command sends CommandLine (measured with agy 1.2.11 on
// 2026-09-25); command_line is the other spelling agy.exe carries. agy
// 1.2.11 types a line into a task run_command left open with manage_task,
// Action send_input and the line under Input (measured on 2026-09-25); the
// model wrote the line end itself, so Input reaches the terminal as it
// stands. send_command_input is the older tool for the same, which agy.exe
// still carries; its argument name is not measured, Input is the guess.
// manage_task's schema in agy 1.2.11 names list, status, kill and
// send_input; the first three carry no line.
type commandTool struct {
	keys         []string // argument names the line may stand under, matched without case
	lineOptional bool     // a call without a line passes: Claude's shells, whose tool always carries one
	whole        bool     // the value is a whole command line, not keystrokes typed into an open terminal
	quiet        []string // Actions whose calls carry no line
}

// commandTools are the tools that run a shell line; every line found is
// judged, for WriteTargets' reason.
var commandTools = map[string]commandTool{
	"Bash":               {keys: []string{"command"}, lineOptional: true, whole: true},
	"PowerShell":         {keys: []string{"command"}, lineOptional: true, whole: true},
	"run_command":        {keys: []string{"CommandLine", "command_line"}, whole: true},
	"send_command_input": {keys: []string{"Input"}},
	"manage_task":        {keys: []string{"Input"}, quiet: []string{"list", "status", "kill"}},
}

// commandLines is every non-empty string a call to tool carries under one of
// its keys, in any case, in the sorted order of the call's keys so that the
// reasons come out the same on every run. The values are found before the
// Action is read: a quiet Action does not stop a line it carries from being
// judged. A call is answered with ok false when a key holds a value that is
// no string, which might be the line the tool runs, and when it carries none
// and is not quiet: a line the guard cannot find would switch off every
// command rule without a word. The strings found are judged either way.
func commandLines(tool commandTool, input map[string]any) (values []string, ok bool) {
	readable := true
	for _, key := range slices.Sorted(maps.Keys(input)) {
		if !slices.ContainsFunc(tool.keys, func(name string) bool { return strings.EqualFold(name, key) }) {
			continue
		}
		switch value := input[key].(type) {
		case string:
			// An empty value types and runs nothing.
			if value != "" {
				values = append(values, value)
			}
		default:
			readable = false
		}
	}
	if !readable {
		return values, false
	}
	if len(values) > 0 {
		return values, true
	}
	return nil, quietAction(tool, input)
}

// quietAction says whether a call names an Action that carries no line.
// Every key spelled action counts, and each must name a quiet Action as a
// string: a second spelling with another Action, or a value that is no
// string, might be the one the tool obeys.
func quietAction(tool commandTool, input map[string]any) bool {
	named := false
	for key, value := range input {
		if !strings.EqualFold(key, "action") {
			continue
		}
		// A value that is no string reads as "", which no quiet list holds.
		if action, _ := value.(string); !slices.Contains(tool.quiet, action) {
			return false
		}
		named = true
	}
	return named
}

// typedLines is the lines a terminal would run from what an agent types into
// an open task, or the reason the guard cannot tell. Only whole lines without
// control characters can be judged: a fragment may be finished by the next
// call, and a backspace, an escape sequence, a tab an interactive shell
// completes at, or another key the terminal binds edits the line after the
// guard has read it. A line ending in a backslash or a backtick is not whole
// either: bash and PowerShell continue it on the next line.
func typedLines(name, value string) (lines []string, refusal string) {
	const rule = "loomux judges what an agent types into a task only as whole lines without control characters; this "
	if !strings.HasSuffix(value, "\n") && !strings.HasSuffix(value, "\r") {
		return nil, rule + name + " input does not end its line, so it refuses"
	}
	if strings.ContainsFunc(value, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\r' }) {
		return nil, rule + name + " input carries a control character, so it refuses"
	}
	lines = strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == '\r' })
	// A backslash or a backtick at a line end continues the line in bash or
	// PowerShell: the next line finishes it, after the guard has read it.
	for _, line := range lines {
		if strings.HasSuffix(line, `\`) || strings.HasSuffix(line, "`") {
			return nil, rule + name + " input does not end its line, so it refuses"
		}
	}
	return lines, ""
}
```

(`unicode.IsControl` ist genau die Kategorie Cc: U+0000–U+001F, U+007F, U+0080–U+009F; ein ungültiges UTF-8-Byte kommt aus `encoding/json` schon als U+FFFD an.)

Die ganze Funktion `checkTool` wird zu:

```go
// checkTool judges one tool call against the built-in rules and the project's
// own, and answers every reason it found: a caller that wants to say why it
// refuses needs all of them, not the first.
func checkTool(root, tool string, input map[string]any, policy config.Policy) []string {
	var reasons []string
	if guard.IsWritingTool(tool) {
		for _, target := range guard.WriteTargets(input) {
			rel := relativePath(target, root)
			for _, rule := range append(builtinPathRules, policy.Paths...) {
				for _, glob := range rule.Match {
					matched, err := matchGlob(glob, rel)
					if err != nil {
						// A rule nobody can evaluate is a rule nobody can trust,
						// and the call it would have judged goes no further.
						reasons = append(reasons, fmt.Sprintf(
							"loomux cannot read the glob %q of the rule %q, so it refuses: %v",
							glob, rule.Reason, err))
						break
					}
					if matched {
						reasons = append(reasons, rule.Reason)
						break
					}
				}
			}
		}
	}
	if shell, found := commandTools[tool]; found {
		values, ok := commandLines(shell, input)
		if !ok && !shell.lineOptional {
			reasons = append(reasons, "loomux found no command line in this "+tool+" call, so it cannot judge it and refuses")
		}
		for _, value := range values {
			lines := []string{value}
			if !shell.whole {
				var refusal string
				if lines, refusal = typedLines(tool, value); refusal != "" {
					reasons = append(reasons, refusal)
					continue
				}
			}
			for _, line := range lines {
				for _, rule := range append(builtinCommands(), policy.Commands...) {
					if rule.Regex.MatchString(line) {
						reasons = append(reasons, rule.Reason)
					}
				}
				if writesConfiguration(line) {
					reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads, and merge-hook install and remove write executable hooks into repositories; a human runs them. An agent proposes a change with `loomux config set|unset … --propose`, which a human applies")
				}
			}
		}
	}
	return reasons
}
```

Danach `gofmt -l internal/hooks` (leer) und `grep -rn "judgedOrRefused\|quietActions" internal/` (leer).

- [ ] **Step 4: Grün prüfen**

Run: `go test ./internal/hooks/ -run 'CommandTool|Casing|QuietAction|TypedInput|ManageTask|Unreadable' -count=1`, dann `go vet ./internal/hooks/` und `go test ./internal/hooks/ -count=1 -coverprofile=<scratch>/t1.out > <scratch>/t1-full.txt 2>&1` und `go tool cover -func=<scratch>/t1.out | grep -E 'guard.go:.*(commandLines|quietAction|typedLines|checkTool)'`

Expected: `ok  	github.com/xidus90/loomux/internal/hooks`; die vier Funktionen je `100.0%`. Zusätzlich `go test ./internal/cli/ ./internal/hosts/ ./internal/cases/ ./internal/brain/guard/ -count=1` → alle `ok` (beim Planen gemessen). Nach Step 6 kommt `go test ./internal/plancheck/ -count=1` dazu (dort ausgeschrieben), weil Step 6 `migration.md` ändert.

- [ ] **Step 5: Mutationsprobe**

Je eine Änderung per `go test -overlay` (Kopie von `guard.go` im Scratchpad, Ersetzung per Python-Skript statt `sed`, wegen der Backslashes), jeweils mit `-run 'CommandTool|Casing|QuietAction|TypedInput|ManageTask|Unreadable'`:
1. `strings.EqualFold(name, key)` → `name == key`: rot `TestGitPushIsRefusedOnEveryCommandTool`, `TestEveryCasingOfALineKeyIsJudged`.
2. `if !shell.whole {` → `if false {`: rot `TestATypedInputIsJudgedOnlyAsWholeLines`, `TestATypedInputIsJudgedLineByLine`.
3. `if len(values) > 0 {` → `if len(values) > 0 && !quietAction(tool, input) {` (stille Action zuerst): rot `TestAQuietActionExcusesOnlyAMissingLine`, `TestATypedInputIsJudgedOnlyAsWholeLines`.
4. `named = true` → `return true` (nur der erste action-Schlüssel zählt): rot `TestAQuietActionExcusesOnlyAMissingLine`.
5. `unicode.IsControl(r) &&` → `unicode.IsControl(r) && r < 0x20 &&` (DEL und C1 fallen weg): rot `TestATypedInputIsJudgedOnlyAsWholeLines` (`"ls\x7f\n": reasons []`); mit `false && unicode.IsControl(r) &&` rot mit `"\x1b[A\n": reasons []`.
6. Die Zeilenende-Bedingung in `typedLines` → `false`: rot `TestATypedInputIsJudgedOnlyAsWholeLines`.
7. `lines = strings.FieldsFunc(…)` → `lines = []string{value}`: rot `TestATypedInputIsJudgedLineByLine` (`"echo hi\nrm -rf x\n": reasons []`) und `TestATypedInputIsJudgedOnlyAsWholeLines` (der ungeteilte Wert `"loomux \\\ninit\n"` endet nicht auf `\` und bekommt den Grund von `writesConfiguration` statt des Fragment-Grunds).
8. `if !ok && !shell.lineOptional {` → `if !ok {`: rot `TestEveryCommandToolWithoutALineIsRefused`.
9. `readable = false` → nichts (Zeile entfernen): rot `TestAQuietActionExcusesOnlyAMissingLine` (`{"Action":"kill","Input":42}` geht durch).
10. `if !readable { return values, false }` → `if !readable { return nil, false }`: rot `TestAnUnreadableLineValueHidesNoOther`.
11. `if value != "" {` → `{`: rot `TestAQuietActionExcusesOnlyAMissingLine` (`{"Action":"kill","Input":""}` wird als Fragment verweigert).
12. In `typedLines` `&& r != '\n' && r != '\r'` → `&& r != '\t' && r != '\n' && r != '\r'` (Tab wieder erlaubt): rot `TestATypedInputIsJudgedOnlyAsWholeLines` (`"git pus\t origin main\n": reasons []`).
13. In `typedLines` die Schleife `for _, line := range lines { … }` über die Zeilenenden entfernen (Fortsetzung wieder erlaubt): rot `TestATypedInputIsJudgedOnlyAsWholeLines` (`guard_test.go:342: [manage_task] "loomux \\\ninit\n": reasons []`).

Die Punkte 9–12 kamen nach der Planungsprüfung dazu (nicht leere Strings, unlesbare Werte, Tab), Punkt 13 nach dem Review (Zeilenfortsetzung); sie sind wie 1–8 per Overlay gemessen und fallen je mit dem genannten Test.

Hinweis: `go test -overlay` zusammen mit `-coverprofile` nahm unter go1.27.0 die echte Datei statt der Überlagerung (Build-Fehler gegen den alten Typ); Mutationsproben also ohne `-cover` fahren.

- [ ] **Step 6: Doku**

`docs/en/hooks.md`, der Absatz, der mit `**Paths and command lines, not content.**` beginnt (Zeilen 358-372, endet mit `What a tool writes into a file is not judged.`), wird ganz ersetzt durch:

```markdown
**Paths and command lines, not content.** A writing tool — `Write`, `Edit`,
`MultiEdit`, `NotebookEdit`, and Antigravity's `write_to_file`,
`replace_file_content` and `multi_replace_file_content` — yields every target
it names under `file_path`, `notebook_path`, `TargetFile` or `target_file`; all
of them are judged, not the first one found. `Bash` and `PowerShell` yield
their `command`, and Antigravity's `run_command` its command line under
`CommandLine` (measured with agy 1.2.11; `commandLine` and `command_line`,
the other spellings agy.exe carries, are judged too). agy types into a task
`run_command` left open with `manage_task`, whose `send_input` action yields
what it types under `Input` (measured with agy 1.2.11);
`send_command_input`, the older tool for the same that agy.exe still carries,
yields `Input`, a name that is not measured. Argument names are matched
without regard to case, and every value found is judged, so a harmless
`Input` cannot hide a forbidden `input`; a value under one of them that is no
string is refused (for `Bash` and `PowerShell` it is ignored, since their
tool always sends a string), and an empty one carries no line. What an agent
types into a task is judged only as whole lines: each value must end with a
line end, carry no control character other than the line ends, and end no
line in a backslash or a backtick, which bash and PowerShell continue on the
next line; it is split at every line end and each line is judged on its own.
A fragment, a lone key, Ctrl-C, an arrow key, a backspace, a tab or a line
continuation is refused, because the terminal could finish, edit or complete
the line after the guard has read it; `kill` still
ends a task. `manage_task`'s `list`, `status` and `kill` carry no line and
pass, but only a call without a line: a line is judged whatever the action
says, and a call counts as quiet only when every key spelled `action` names
one of the three; any other action, or none, is judged. A `run_command`,
`send_command_input` or `manage_task` carrying none of its names is refused.
What a tool writes into a file is not judged.
```

`docs/de/hooks.md`, der Absatz, der mit `**Pfade und Befehlszeilen, kein Inhalt.**` beginnt (Zeilen 374-388, endet mit `Was ein Werkzeug in eine Datei schreibt, wird nicht geprüft.`), wird ganz ersetzt durch:

```markdown
**Pfade und Befehlszeilen, kein Inhalt.** Ein schreibendes Werkzeug — `Write`,
`Edit`, `MultiEdit`, `NotebookEdit` und Antigravitys `write_to_file`,
`replace_file_content` und `multi_replace_file_content` — liefert jedes Ziel,
das es unter `file_path`, `notebook_path`, `TargetFile` oder `target_file`
nennt; geprüft werden alle, nicht das erste gefundene. `Bash` und `PowerShell`
liefern ihr `command`, Antigravitys `run_command` seine Befehlszeile unter
`CommandLine` (gemessen mit agy 1.2.11; `commandLine` und `command_line`, die
anderen Schreibweisen in agy.exe, werden mitgeprüft). In eine von
`run_command` offen gelassene Aufgabe tippt agy mit `manage_task`, dessen
Aktion `send_input` das Getippte unter `Input` liefert (gemessen mit agy
1.2.11); `send_command_input`, das ältere Werkzeug dafür, das agy.exe noch
trägt, liefert `Input`, ein ungemessener Name. Argumentnamen werden ohne
Rücksicht auf die Schreibung verglichen, und jeder gefundene Wert wird
geprüft, ein harmloses `Input` verdeckt also kein verbotenes `input`; ein
Wert darunter, der kein String ist, wird verweigert (bei `Bash` und
`PowerShell` übergangen, weil ihr Werkzeug immer einen String schickt), ein
leerer trägt keine Zeile. Was ein Agent in eine Aufgabe tippt, wird nur als
ganze Zeilen geprüft: Jeder Wert muss auf ein Zeilenende enden, darf außer
den Zeilenenden kein Steuerzeichen tragen und keine Zeile auf einen
Backslash oder einen Backtick enden lassen, mit denen bash und PowerShell in
der nächsten Zeile fortsetzen; er wird an jedem Zeilenende geteilt, und jede
Zeile wird für sich geprüft. Ein Bruchstück, eine einzelne Taste, Ctrl-C,
eine Pfeiltaste, ein Rückschritt, ein Tab oder eine Zeilenfortsetzung wird
verweigert, weil
das Terminal die Zeile vollenden, ändern oder ergänzen könnte, nachdem der
Wächter sie gelesen hat; `kill` beendet eine Aufgabe weiter. `list`, `status`
und `kill` von `manage_task` tragen keine Zeile und laufen durch, aber nur ein
Aufruf ohne Zeile: Eine Zeile wird geprüft, gleich welche Aktion der Aufruf
nennt, und still ist ein Aufruf nur, wenn jeder Schlüssel mit der Schreibung
`action` eine der drei nennt; jede andere Aktion und ein Aufruf ohne Aktion
wird geprüft. Ein `run_command`, `send_command_input` oder `manage_task` ohne
einen seiner Namen wird verweigert. Was ein Werkzeug in eine Datei schreibt,
wird nicht geprüft.
```

`docs/en/cli-reference.md`, im Antigravity-Punkt (Zeilen 1318-1320) die umbrochenen Zeilen

```
  refuses the whole file otherwise. A `run_command`, a `send_command_input`
  and a `manage_task` that sends input to a task are judged by the same
  command rules as `Bash`; one whose command line the guard cannot find is
  refused. An entry from before `manage_task` joined the matcher is kept and
```

ersetzen durch:

```
  refuses the whole file otherwise. A `run_command` is judged by the same
  command rules as `Bash`; what a `send_command_input` or a `manage_task`
  types into a task is judged the same way, line by line, and only as whole
  lines without control characters or a line continuation. A call whose
  command line the guard cannot find is refused; `manage_task`'s `list`,
  `status` and `kill` pass only while they carry no line. An entry from before `manage_task` joined the matcher is kept and
```

(Die erste und die letzte Zeile bleiben wörtlich stehen; nur der Satz dazwischen ändert sich. Den umbrochenen Rest darf das Umbrechen nicht berühren, damit Task 4 seinen Anker `An entry from before `manage_task` joined the matcher is kept and` findet.)

`docs/de/cli-reference.md`, im Antigravity-Punkt (Zeilen 1374-1378) die umbrochenen Zeilen

```
  `matcher` und `hooks`: agy 1.2.11 verwirft sonst die ganze Datei. Ein
  `run_command`, ein `send_command_input` und ein `manage_task`, das einer
  Aufgabe Eingabe schickt, werden nach denselben Befehlsregeln beurteilt wie
  `Bash`; eines, in dem der Wächter keine Befehlszeile findet, wird
  verweigert. Ein Eintrag von vor `manage_task` im Matcher bleibt stehen und
```

ersetzen durch:

```
  `matcher` und `hooks`: agy 1.2.11 verwirft sonst die ganze Datei. Ein
  `run_command` wird nach denselben Befehlsregeln beurteilt wie `Bash`; was
  ein `send_command_input` oder ein `manage_task` in eine Aufgabe tippt,
  ebenso, Zeile für Zeile und nur als ganze Zeilen ohne Steuerzeichen und
  ohne Zeilenfortsetzung. Ein Aufruf, in dem der Wächter keine Befehlszeile
  findet, wird verweigert; `list`, `status` und `kill` von `manage_task`
  laufen nur durch, solange sie keine Zeile tragen. Ein Eintrag von vor `manage_task` im Matcher bleibt stehen und
```

(Der folgende Satz über den älteren Matcher gehört zu `init` (Task 4) und bleibt hier unberührt.)

`docs/en/migration.md:38`, Zeile 4a-2: das Teilstück
`` `run_command` (`CommandLine`, measured) and `send_command_input` (`Input`, not measured) and `manage_task` with `send_input` (`Input`, measured) go through the command rules; ``
ersetzen durch:
`` `run_command` (`CommandLine`, measured) and `send_command_input` (`Input`, not measured) and `manage_task` with `send_input` (`Input`, measured) go through the command rules, argument names matched without case, typed input only as whole lines without control characters or a line continuation and judged line by line, and a quiet `manage_task` action passing only without a line; ``

`docs/de/migration.md:39`, Zeile 4a-2: das Teilstück
`` `run_command` (`CommandLine`, gemessen) und `send_command_input` (`Input`, ungemessen) und `manage_task` mit `send_input` (`Input`, gemessen) laufen durch die Befehlsregeln; ``
ersetzen durch:
`` `run_command` (`CommandLine`, gemessen) und `send_command_input` (`Input`, ungemessen) und `manage_task` mit `send_input` (`Input`, gemessen) laufen durch die Befehlsregeln, Argumentnamen ohne Rücksicht auf die Schreibung, getippte Eingabe nur als ganze Zeilen ohne Steuerzeichen und ohne Zeilenfortsetzung und Zeile für Zeile, eine stille Aktion von `manage_task` nur ohne Zeile; ``

(Kein `|` und kein `**` in den Ersatztexten der Tabellenzeile: `internal/plancheck` teilt die Zeile an jedem ungeschützten `|` und liest die Stufe aus dem Fettdruck der ersten Zelle.)

Run: `go test ./internal/plancheck/ -count=1`
Expected: `ok  	github.com/xidus90/loomux/internal/plancheck` (mit genau diesen Ersatztexten gemessen).

- [ ] **Step 7: Commit**

`git rev-parse --show-toplevel` und `git branch --show-current` lesen (Worktree, Zweig `fix/antigravity-review-findings`). Dann:

```
git add internal/hooks/guard.go internal/hooks/guard_test.go docs/en/hooks.md docs/de/hooks.md docs/en/cli-reference.md docs/de/cli-reference.md docs/en/migration.md docs/de/migration.md
```

Nachricht per Write nach `<scratch>/msg-guard-lines.txt`:

```
fix(guard): judge every line an Antigravity task call carries

The guard described each command tool in three maps, and a tool missing
from the refusal map failed open. One table now describes each tool, and
its zero value is the closed case: no line refuses, and what the tool
carries is judged as typing.

Argument names are matched without case and every match is judged, so a
harmless Input no longer hides a forbidden input. Lines are collected
before the Action is read: a quiet manage_task action (list, status, kill)
only excuses a call that carries no line, and only when every action key
names one. A value under a line key that is no string refuses the call;
for Bash and PowerShell it is ignored, since their tool always sends a
string. What send_command_input and
manage_task type into a task is judged only as whole lines without control
characters, split at each line end, and a line that ends in a backslash or
a backtick is refused as unfinished, because a fragment, an escape sequence
or a continuation can change the line after the guard has read it.
```

`git commit -F <scratch>/msg-guard-lines.txt > <scratch>/commit-guard-lines.log 2>&1`; bei Rot die Datei nach `--- FAIL` durchsuchen. Danach `git log -1 --format='%an <%ae>'` (der Mensch, kein Modell).

### Task 2: Einen dekodierten Aufruf ohne Werkzeugnamen verweigern

**Files:**
- Modify: `internal/hooks/pretool.go:43-50` (`policyReasons` und sein Kommentar)
- Test: `internal/hooks/pretool_test.go:104-112` (`TestAPayloadWithoutAToolNameReachesTheBarrier` wird umgekehrt und umbenannt, ein Test für Nutzlasten, die kein Objekt sind, kommt dazu)
- Doku: `docs/en/hooks.md:338-339` (Mermaid) und `:350-356` („Never exit 1“), `docs/de/hooks.md:354-355` und `:366-372`

**Interfaces:**
- Consumes: `checkTool(root, tool string, input map[string]any, policy config.Policy) []string` aus Task 1 (unverändert in der Signatur)
- Produces: `func policyReasons(object map[string]any, root string) ([]string, error)` — Signatur unverändert; neu: ein nicht-nil `object`, für das `guard.Call` keinen Namen liefert, ergibt `[]string{"loomux found no tool name in this call, so it cannot judge it and refuses"}, nil`, ohne `readPolicy` zu rufen.

- [ ] **Step 1: Failing test schreiben**

In `internal/hooks/pretool_test.go` den Kommentar und die Funktion `TestAPayloadWithoutAToolNameReachesTheBarrier` (Zeilen 104-112) ganz ersetzen durch:

```go
// A payload that decodes but names no tool is nothing the policy can judge,
// and a call nobody judged does not pass: a host that spells its tool name
// under another key would otherwise run every call past every rule. The
// refusal comes before the policy file is read, so a broken one does not
// change its wording.
func TestACallWithoutAToolNameIsRefused(t *testing.T) {
	root, state := world(t, "[[policy.paths.rules]\n")
	const want = "loomux found no tool name in this call, so it cannot judge it and refuses"
	for _, input := range []string{
		`{"tool_input":{"file_path":"main.go"}}`,
		`{"tool_input":{"command":"git push"}}`,
		`{"toolCall":{"args":{"CommandLine":"git push"}}}`,
		`{"toolCall":"x"}`,
		`{"tool_name":42,"tool_input":{"command":"git push"}}`,
		`{}`,
	} {
		if code, _, errOut := call(t, root, state, input); code != 2 || !strings.Contains(errOut, want) {
			t.Fatalf("%s: code %d, err %q", input, code, errOut)
		}
	}
}

// A payload that is no object is the barrier's to refuse, in the barrier's
// words: the policy never sees it.
func TestAPayloadThatIsNoObjectKeepsTheBarriersWording(t *testing.T) {
	root, state := world(t, "")
	// null decodes into a nil map, and the barrier refuses it as no object.
	for _, input := range []string{"not json", "[]", `"x"`, "null"} {
		code, _, errOut := call(t, root, state, input)
		if code != 2 || strings.Contains(errOut, "no tool name") || strings.Contains(errOut, "loomux policy refused") {
			t.Fatalf("%s: code %d, err %q", input, code, errOut)
		}
	}
}
```

- [ ] **Step 2: Rot prüfen**

Run: `go test ./internal/hooks/ -run 'ToolName|NoObject' -count=1 -v > <scratch>/t2-red.txt 2>&1; grep -E '^--- |pretool_test|^FAIL' <scratch>/t2-red.txt`

Expected:
```
    pretool_test.go:121: {"tool_input":{"file_path":"main.go"}}: code 0, err ""
--- FAIL: TestACallWithoutAToolNameIsRefused (0.01s)
--- PASS: TestAPayloadThatIsNoObjectKeepsTheBarriersWording (0.00s)
FAIL	github.com/xidus90/loomux/internal/hooks
```
(`-run 'ToolName'` trifft auch `TestCheckToolNamesTheConfigurationReason`, der grün bleibt.) Der zweite Test hält den heutigen Stand fest und ist schon grün; er wird in Step 5 durch die Mutation rot.

- [ ] **Step 3: Implementieren**

In `internal/hooks/pretool.go` Kommentar und Funktion `policyReasons` ganz ersetzen durch:

```go
// policyReasons judges the call against the policy. A payload that did not
// decode arrives as nil and yields no reasons: refusing it is the barrier's
// job, with the barrier's wording, one step later. An object that names no
// tool is refused here, before the policy file is read: no rule can be
// matched against it, and a call nobody judged does not pass.
func policyReasons(object map[string]any, root string) ([]string, error) {
	if object == nil {
		return nil, nil
	}
	tool, input := guard.Call(object)
	if tool == "" {
		return []string{"loomux found no tool name in this call, so it cannot judge it and refuses"}, nil
	}
	policy, err := readPolicy(root)
	if err != nil {
		return nil, err
	}
	return checkTool(root, tool, input, policy), nil
}
```

(JSON `null` dekodiert zu einer nil-Map und geht wie bisher an die Schranke.) Aufgezeichnete Fälle: alle `stdin` unter `testdata/cases/1a/hook-pre-tool-use/*` tragen `tool_name` oder `toolCall.name` außer `unreadable-payload` (`not json`, bleibt bei der Schranke); `testdata/cases/1a-payloads/session.json` (`{"hook_event_name":"SessionStart"}`) gehört zu `session-start`. Kein Fall kippt.

- [ ] **Step 4: Grün prüfen**

Run: `go test ./internal/hooks/ -run 'ToolName|NoObject' -count=1`, dann `go vet ./internal/hooks/`, `go test ./internal/hooks/ -count=1 -coverprofile=<scratch>/t2.out > <scratch>/t2-full.txt 2>&1`, `go tool cover -func=<scratch>/t2.out | grep pretool.go` und die Pakete, die die Fälle abspielen: `go test ./internal/cli/ ./internal/cases/ ./internal/hosts/ -count=1 > <scratch>/t2-cases.txt 2>&1`

Expected: `ok  	github.com/xidus90/loomux/internal/hooks`; `PreToolUse 100.0%`, `policyReasons 100.0%`; `ok` für `internal/cli`, `internal/cases`, `internal/hosts` (beim Planen gemessen).

- [ ] **Step 5: Mutationsprobe**

Per `go test -overlay` mit `-run 'ToolName|NoObject'`:
1. Den Block `if object == nil { return nil, nil }` entfernen: rot `TestAPayloadThatIsNoObjectKeepsTheBarriersWording` (`not json` bekommt jetzt den Policy-Grund).
2. `return []string{"loomux found no tool name …"}, nil` → `return nil, nil`: rot `TestACallWithoutAToolNameIsRefused`.

- [ ] **Step 6: Doku**

`docs/en/hooks.md`, im Mermaid-Diagramm die Zeile `    named -->|no| barrier` ersetzen durch die zwei Zeilen:

```
    named -->|"no object"| barrier
    named -->|"an object without a tool name"| deny
```

`docs/en/hooks.md`, der Absatz, der mit `**Never exit 1.**` beginnt, wird ganz ersetzt durch:

```markdown
**Never exit 1.** A host reads 1 as a non-blocking error and runs the tool
anyway. So every way this hook can fail — a missing or unknown `--host`, stdin
that cannot be read, a payload that is no JSON object, an object that names no
tool, a broken `.loomux/config.toml`, a panic — refuses with 2
(`internal/cli/hook.go`, `internal/hooks/pretool.go`). A policy that waved
calls through as soon as its own configuration is unreadable would be exactly
the barrier one believes to be there and which is not. A call without a tool
name is refused by the policy before the configuration is read, since no rule
can be matched against it; a payload that is no object is left to the write
barrier, which refuses it in its own words.
```

`docs/de/hooks.md`, im Mermaid-Diagramm die Zeile `    named -->|nein| barrier` ersetzen durch:

```
    named -->|"kein Objekt"| barrier
    named -->|"Objekt ohne Werkzeugnamen"| deny
```

`docs/de/hooks.md`, der Absatz, der mit `**Nie Exit 1.**` beginnt, wird ganz ersetzt durch:

```markdown
**Nie Exit 1.** Ein Host liest 1 als nicht blockierenden Fehler und führt das
Werkzeug trotzdem aus. Darum lehnt jeder Weg, auf dem dieser Hook scheitern
kann, mit 2 ab — ein fehlendes oder unbekanntes `--host`, ein unlesbares stdin,
eine Nutzlast, die kein JSON-Objekt ist, ein Objekt ohne Werkzeugnamen, eine
kaputte `.loomux/config.toml`, eine Panik (`internal/cli/hook.go`,
`internal/hooks/pretool.go`). Eine Policy, die Aufrufe durchwinkt, sobald ihre
eigene Konfiguration unlesbar ist, wäre genau die Schranke, die man für
vorhanden hält und die es nicht ist. Einen Aufruf ohne Werkzeugnamen verweigert
die Policy, bevor sie die Konfiguration liest, weil keine Regel auf ihn passen
kann; eine Nutzlast, die kein Objekt ist, bleibt der Schreibschranke, die sie
mit ihrem eigenen Wortlaut ablehnt.
```

Der Absatz „**The configuration is read for every call with a tool name**“ / „**Die Konfiguration wird bei jedem Aufruf mit Werkzeugnamen gelesen**“ bleibt richtig und unverändert.

- [ ] **Step 7: Commit**

`git rev-parse --show-toplevel` (Worktree `.worktrees/review-findings`) und `git branch --show-current` (`fix/antigravity-review-findings`) lesen. Dann:

```
git add internal/hooks/pretool.go internal/hooks/pretool_test.go docs/en/hooks.md docs/de/hooks.md
```

Nachricht per Write nach `<scratch>/msg-guard-name.txt`:

```
fix(guard): refuse a call that names no tool

A decoded payload without a tool name reached no rule, and the write
barrier let it pass, so a host that spells its tool name under another
key ran every call past the policy. The policy now refuses such an object
before it reads the configuration. A payload that is no object still goes
to the barrier and is refused in its words.
```

`git commit -F <scratch>/msg-guard-name.txt > <scratch>/commit-guard-name.log 2>&1`; danach `git log -1 --format='%an <%ae>'`.

### Task 3: Die Matcher an die Werkzeuglisten binden

**Files:**
- Create: `internal/hooks/matcher_test.go`
- Create: `internal/brain/guard/matcher_test.go`
- Modify: `internal/setup/hostfile/table.go:54-57` (Kommentar über den Antigravity-Einträgen)

**Interfaces:**
- Consumes: `commandTools map[string]commandTool` (Task 1); `writingTools map[string]bool` (`internal/brain/guard/guard.go:62`); `hostfile.Entries(host hosts.Host, binary string) []hostfile.Entry`, `hostfile.Checkout`, `hosts.HostClaude`, `hosts.HostAntigravity`
- Produces: nichts für andere Tasks

- [ ] **Step 1: Failing test schreiben**

Zyklusprüfung vorab: `go list -deps ./internal/setup/hostfile | grep loomux` zeigt nur `internal/hosts`, `internal/shellwords`, `internal/setup/hostfile`; kein Import von `internal/hooks` oder `internal/brain/guard`.

`internal/hooks/matcher_test.go`:

```go
package hooks

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/setup/hostfile"
)

// The hook entries init writes are kept by hand, apart from the table the
// guard judges by. A command tool missing from every PreToolUse matcher is
// never shown to the guard, and its lines run unjudged.
func TestEveryCommandToolIsInAPreToolUseMatcher(t *testing.T) {
	var matched []string
	for _, host := range []hosts.Host{hosts.HostClaude, hosts.HostAntigravity} {
		for _, entry := range hostfile.Entries(host, hostfile.Checkout) {
			if entry.Event == "PreToolUse" {
				matched = append(matched, strings.Split(entry.Matcher, "|")...)
			}
		}
	}
	for name := range commandTools {
		if !slices.Contains(matched, name) {
			t.Errorf("%s is in commandTools but in no PreToolUse matcher (%v)", name, matched)
		}
	}
}
```

`internal/brain/guard/matcher_test.go`:

```go
package guard

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/setup/hostfile"
)

// The hook entries init writes are kept by hand, apart from writingTools. A
// writing tool missing from the PreToolUse matchers is never shown to the
// barrier, and one missing from the PostToolUse matchers is never checked
// after it wrote.
func TestEveryWritingToolIsInThePreAndPostToolUseMatchers(t *testing.T) {
	matched := map[string][]string{}
	for _, host := range []hosts.Host{hosts.HostClaude, hosts.HostAntigravity} {
		for _, entry := range hostfile.Entries(host, hostfile.Checkout) {
			matched[entry.Event] = append(matched[entry.Event], strings.Split(entry.Matcher, "|")...)
		}
	}
	for name := range writingTools {
		for _, event := range []string{"PreToolUse", "PostToolUse"} {
			if !slices.Contains(matched[event], name) {
				t.Errorf("%s is in writingTools but in no %s matcher (%v)", name, event, matched[event])
			}
		}
	}
}
```

- [ ] **Step 2: Rot prüfen**

Die Tests halten einen Stand fest, der heute stimmt; rot werden sie gegen einen gekappten Matcher. Per `go test -overlay` mit einer Kopie von `table.go`, in der `|manage_task"` zu `"` wird:

Run: `go test -overlay <scratch>/drop-manage-task.json ./internal/hooks/ ./internal/brain/guard/ -run 'Matcher' -count=1`

Expected:
```
--- FAIL: TestEveryCommandToolIsInAPreToolUseMatcher (0.00s)
    matcher_test.go:26: manage_task is in commandTools but in no PreToolUse matcher ([Write Edit MultiEdit NotebookEdit Bash PowerShell write_to_file replace_file_content multi_replace_file_content run_command send_command_input])
FAIL	github.com/xidus90/loomux/internal/hooks
ok  	github.com/xidus90/loomux/internal/brain/guard
```

Mit einer Kopie, in der `writers := "write_to_file|` zu `writers := "` wird:
```
ok  	github.com/xidus90/loomux/internal/hooks
--- FAIL: TestEveryWritingToolIsInThePreAndPostToolUseMatchers (0.00s)
    matcher_test.go:26: write_to_file is in writingTools but in no PreToolUse matcher (…)
    matcher_test.go:26: write_to_file is in writingTools but in no PostToolUse matcher (…)
FAIL	github.com/xidus90/loomux/internal/brain/guard
```

- [ ] **Step 3: Implementieren**

Keine Produktionsänderung außer dem Kommentar. In `internal/setup/hostfile/table.go` im Zweig `case hosts.HostAntigravity:` den Kommentar über `writers := …` ganz ersetzen durch:

```go
		// agy runs a hook from .agents/, one below the project root. Its
		// PreInvocation and Stop take a flat list of handlers: agy 1.2.11
		// refuses the whole file when either holds a {"hooks": [...]} block
		// (measured 2026-09-25), which would leave the guard unloaded.
		// The matchers are kept by hand; tests in internal/hooks and
		// internal/brain/guard hold them to the guard's command tools and
		// the barrier's writing tools, so a tool dropped here fails them.
```

- [ ] **Step 4: Grün prüfen**

Run: `go test ./internal/hooks/ ./internal/brain/guard/ -run 'Matcher' -count=1`, dann `go vet ./internal/hooks/ ./internal/brain/guard/ ./internal/setup/hostfile/` und `go test ./internal/hooks/ ./internal/brain/guard/ ./internal/setup/hostfile/ -count=1 > <scratch>/t3-full.txt 2>&1`

Expected: `ok` für alle drei Pakete. Keine neue Produktionsfunktion, die Coverage bleibt wie nach Task 2.

- [ ] **Step 5: Mutationsprobe**

Wie Step 2: `|manage_task` aus dem Antigravity-PreToolUse-Matcher in `table.go` entfernen → rot `TestEveryCommandToolIsInAPreToolUseMatcher`; `write_to_file|` aus `writers` entfernen → rot `TestEveryWritingToolIsInThePreAndPostToolUseMatchers`; `|run_command` entfernen → rot `TestEveryCommandToolIsInAPreToolUseMatcher`. Der Test in `internal/brain/guard` wird durch das Wegfallen von `manage_task` nicht rot und kann es nicht: `manage_task` ist kein schreibendes Werkzeug.

- [ ] **Step 6: Doku**

Keine Nutzerdoku; das Verhalten ändert sich nicht. Nur der Kommentar aus Step 3.

- [ ] **Step 7: Commit**

Toplevel und Zweig lesen. Dann:

```
git add internal/hooks/matcher_test.go internal/brain/guard/matcher_test.go internal/setup/hostfile/table.go
```

Nachricht per Write nach `<scratch>/msg-matchers.txt`:

```
test(hooks): hold the hook matchers to the guard's tool lists

The matchers init writes are kept by hand, apart from the tools the guard
judges and the barrier checks. A tool dropped from a matcher is never
shown to either, and its calls run unjudged without a failing test. Two
tests now read the entries init writes and require every command tool in
a PreToolUse matcher and every writing tool in a PreToolUse and a
PostToolUse matcher.
```

`git commit -F <scratch>/msg-matchers.txt > <scratch>/commit-matchers.log 2>&1`; danach `git log -1 --format='%an <%ae>'`.

---

### Task 4: Einen Block für die Werkzeuge anhängen, die einem eigenen Matcher fehlen

Voraussetzung: Task 0 ergab „weiter“.

**Files:**
- Modify: `internal/setup/hostfile/merge.go:1-9` (Paket-Doku), `:42-53` (`Result.Notes`), `:55-125` (`Merge`), `:201-242` (`find`), neu nach `find`: `missingTools`, `toolNames`
- Modify: `internal/setup/hostfile/merge_test.go:547-603` (`TestMergeRecognisesLoomuxEntriesWhereverTheyStand`: der Fall „old matcher“ kippt auf „angehängt“)
- Modify: `internal/setup/plan_test.go:680-690` (`TestAnOwnEntryUnderAnOldMatcherIsNamed` wird umbenannt und kippt)
- Test: `internal/setup/hostfile/merge_test.go` (neue Tests am Ende der Datei)
- Doku: `docs/en/hooks.md`, `docs/de/hooks.md`, `docs/en/getting-started.md`, `docs/de/getting-started.md`, `docs/en/cli-reference.md`, `docs/de/cli-reference.md`

**Interfaces:**
- Consumes: `Entries(host hosts.Host, binary string) []Entry`, `blockFor(entry Entry) map[string]any`, `strconvQuote(s string) string` und `firstCommand(item map[string]any) string` (Testhelfer in `merge_test.go`), `commands(t, merged, container, event)`; alle schon vorhanden.
- Produces: `func find(list []any, entry Entry) (own, foreign bool, elsewhere []string, stale string)` (vorher `elsewhere string`), `func missingTools(want string, have []string) (missing []string, counted bool)`, `func toolNames(matcher string) (names []string, ok bool)`. Die Note für einen Eintrag unter einem älteren Matcher lautet `<Event>: kept an own entry under matcher <A>[, <B>…]`, mit angehängtem Block `…; added one for <n1>|<n2>`; ein eigener Block ohne `matcher`-Schlüssel heißt dort `(none)`. Der Slot ist `<Event>/<fehlende Namen mit |>`, zum Beispiel `PreToolUse/manage_task` oder `PreToolUse/MultiEdit`. `Merge` behält seine Signatur.

- [ ] **Step 1: Failing test schreiben**

In `internal/setup/hostfile/merge_test.go` wird der Schleifenkörper von `TestMergeRecognisesLoomuxEntriesWhereverTheyStand` ersetzt, also alles von `pre := …` bis vor `// Another hook of ours under another matcher is no stand-in.`. So sieht die ganze Funktion danach aus:

```go
func TestMergeRecognisesLoomuxEntriesWhereverTheyStand(t *testing.T) {
	b := Canonical
	all := func(skip string) string {
		var blocks []string
		for _, e := range Entries(claude, b) {
			if e.Event == "PreToolUse" {
				continue
			}
			blocks = append(blocks, `"`+e.Event+`": [{"matcher": "`+e.Matcher+`", "hooks": [{"type": "command", "command": `+strconvQuote(e.Command)+`}]}]`)
		}
		return strings.Join(blocks, ", ") + ", " + skip
	}
	pre := `"${LOCALAPPDATA}/loomux/bin/loomux.exe" hook pre-tool-use --host claude --root x`
	// Ours second in a block of the project on our matcher.
	second := `"PreToolUse": [{"matcher": "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell", "hooks": [{"type": "command", "command": "echo x"}, {"type": "command", "command": ` + strconvQuote(pre) + `}]}]`
	existing := []byte(`{"hooks": {` + all(second) + `}}`)
	got, err := Merge(claude, existing, Entries(claude, b))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(got.Added) != 0 || !bytes.Equal(got.Merged, existing) {
		t.Errorf("second: added %v\n%s", got.Added, got.Merged)
	}
	// A command of ours under ulinit's matcher, from before MultiEdit joined
	// it, stays and gets a block for MultiEdit beside it.
	ulinit := `"PreToolUse": [{"matcher": "Write|Edit|NotebookEdit|Bash|PowerShell", "hooks": [{"type": "command", "command": ` + strconvQuote(pre) + `}]}]`
	got, err = Merge(claude, []byte(`{"hooks": {`+all(ulinit)+`}}`), Entries(claude, b))
	if err != nil {
		t.Fatalf("old matcher: %v", err)
	}
	note := "PreToolUse: kept an own entry under matcher Write|Edit|NotebookEdit|Bash|PowerShell; added one for MultiEdit"
	if !reflect.DeepEqual(got.Added, []string{"PreToolUse/MultiEdit"}) || !reflect.DeepEqual(got.Notes, []string{note}) {
		t.Errorf("old matcher: added %v, notes %v", got.Added, got.Notes)
	}
	wantPre := []string{pre, Entries(claude, b)[1].Command}
	if prePre := commands(t, got.Merged, "hooks", "PreToolUse"); !reflect.DeepEqual(prePre, wantPre) {
		t.Errorf("old matcher: PreToolUse = %q, want %q", prePre, wantPre)
	}
	// Another hook of ours under another matcher is no stand-in.
	other := `{"hooks": {"PreToolUse": [{"matcher": "Read", "hooks": [{"type": "command", "command": "loomux hook post-tool-use"}, "x"]}]}}`
	if got, _ := Merge(claude, []byte(other), Entries(claude, b)); !slices.Contains(got.Added, "PreToolUse/Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell") {
		t.Errorf("added %v", got.Added)
	}
	if hookEventOf("loomux hook") != "" || hookEventOf(`"unclosed`) != "" {
		t.Error("hookEventOf reads a hook that is not there")
	}
}
```

Am Ende von `internal/setup/hostfile/merge_test.go` kommt hinzu (keine neuen Imports; `bytes`, `encoding/json`, `reflect`, `testing` und `hosts` sind schon importiert):

```go
// antigravityWithMatcher is the file init writes for Antigravity, its
// PreToolUse block under matcher instead of the one Entries wants.
func antigravityWithMatcher(t *testing.T, matcher string) []byte {
	t.Helper()
	fresh, err := Merge(hosts.HostAntigravity, nil, Entries(hosts.HostAntigravity, Canonical))
	if err != nil {
		t.Fatal(err)
	}
	want := `"matcher": ` + strconvQuote(Entries(hosts.HostAntigravity, Canonical)[1].Matcher)
	if !bytes.Contains(fresh.Merged, []byte(want)) {
		t.Fatalf("no PreToolUse matcher to replace in\n%s", fresh.Merged)
	}
	return bytes.Replace(fresh.Merged, []byte(want), []byte(`"matcher": `+strconvQuote(matcher)), 1)
}

// An own block from before a tool joined the matcher gets a block for that
// tool beside it, with the same command; the old block stays as it is.
func TestMergeAddsABlockForTheToolsAnOldMatcherLacks(t *testing.T) {
	old := "write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input"
	existing := antigravityWithMatcher(t, old)
	wanted := Entries(hosts.HostAntigravity, Canonical)
	got, err := Merge(hosts.HostAntigravity, existing, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !reflect.DeepEqual(got.Added, []string{"PreToolUse/manage_task"}) {
		t.Fatalf("added %v, want the block for manage_task", got.Added)
	}
	note := "PreToolUse: kept an own entry under matcher " + old + "; added one for manage_task"
	if !reflect.DeepEqual(got.Notes, []string{note}) {
		t.Fatalf("notes %v, want %q", got.Notes, note)
	}
	var root map[string]map[string][]map[string]any
	if err := json.Unmarshal(got.Merged, &root); err != nil {
		t.Fatalf("result: %v\n%s", err, got.Merged)
	}
	pre := root["loomux"]["PreToolUse"]
	if len(pre) != 2 || pre[0]["matcher"] != old || pre[1]["matcher"] != "manage_task" ||
		firstCommand(pre[0]) != firstCommand(pre[1]) || firstCommand(pre[1]) != wanted[1].Command {
		t.Fatalf("PreToolUse = %v, want the old block and one for manage_task with the same command", pre)
	}
}

// The appended block and the old one together cover the matcher, so the
// next run finds both, adds nothing and names both.
func TestMergeAddsNothingTwiceForASplitMatcher(t *testing.T) {
	old := "write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input"
	wanted := Entries(hosts.HostAntigravity, Canonical)
	first, err := Merge(hosts.HostAntigravity, antigravityWithMatcher(t, old), wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	again, err := Merge(hosts.HostAntigravity, first.Merged, wanted)
	if err != nil {
		t.Fatalf("second Merge: %v", err)
	}
	if len(again.Added) != 0 || !bytes.Equal(again.Merged, first.Merged) {
		t.Fatalf("second merge added %v\n%s", again.Added, again.Merged)
	}
	note := "PreToolUse: kept an own entry under matcher " + old + ", manage_task"
	if !reflect.DeepEqual(again.Notes, []string{note}) {
		t.Fatalf("notes %v, want %q", again.Notes, note)
	}
}

// Two old blocks of ours whose matchers together cover the wanted one leave
// nothing to add.
func TestMergeAddsNothingWhereTwoOldBlocksCoverTheMatcher(t *testing.T) {
	wanted := []Entry{{Event: "PreToolUse", Matcher: "Write|Edit|Bash", Command: "loomux hook pre-tool-use"}}
	existing := []byte(`{"hooks":{"PreToolUse":[` +
		`{"matcher":"Write|Edit","hooks":[{"type":"command","command":"loomux hook pre-tool-use"}]},` +
		`{"matcher":"Bash","hooks":[{"type":"command","command":"loomux hook pre-tool-use --old"}]}]}}`)
	got, err := Merge(claude, existing, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Added) != 0 || !bytes.Equal(got.Merged, existing) ||
		!reflect.DeepEqual(got.Notes, []string{"PreToolUse: kept an own entry under matcher Write|Edit, Bash"}) {
		t.Fatalf("added %v, notes %v", got.Added, got.Notes)
	}
}

// Only a flat list of names can be counted. An own matcher that is a regular
// expression, or a wanted one that is, gets the note alone, and so does one
// that already names every wanted tool and more.
func TestMergeOnlyNotesAnOldMatcherItCannotCountOrThatLacksNothing(t *testing.T) {
	for _, tc := range []struct{ have, want string }{
		{".*", "Write|Bash"},
		{"Write|Ba.*", "Write|Bash"},
		{"Write||Bash", "Write|Bash|Read"},
		{"Write", "Write|Ba.*"},
		{"Write|Bash|Read", "Write|Bash"},
	} {
		wanted := []Entry{{Event: "PreToolUse", Matcher: tc.want, Command: "loomux hook pre-tool-use"}}
		existing := []byte(`{"hooks":{"PreToolUse":[{"matcher":` + strconvQuote(tc.have) +
			`,"hooks":[{"type":"command","command":"loomux hook pre-tool-use"}]}]}}`)
		got, err := Merge(claude, existing, wanted)
		if err != nil {
			t.Fatalf("%s over %s: %v", tc.want, tc.have, err)
		}
		note := "PreToolUse: kept an own entry under matcher " + tc.have
		if len(got.Added) != 0 || !bytes.Equal(got.Merged, existing) || !reflect.DeepEqual(got.Notes, []string{note}) {
			t.Errorf("%s over %s: added %v, notes %v", tc.want, tc.have, got.Added, got.Notes)
		}
	}
}

// An entry without a matcher is no flat list either: a Stop of ours under a
// matcher it does not want is kept with the note. So is a PreToolUse block
// of ours without a matcher key, which Claude Code runs for every tool; the
// note names its matcher (none).
func TestMergeOnlyNotesAMatcherlessEntryKeptUnderAMatcher(t *testing.T) {
	for _, tc := range []struct {
		existing string
		wanted   Entry
		note     string
	}{
		{
			`{"hooks":{"Stop":[{"matcher":"x","hooks":[{"type":"command","command":"loomux hook stop"}]}]}}`,
			Entry{Event: "Stop", Command: "loomux hook stop"},
			"Stop: kept an own entry under matcher x",
		},
		{
			`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"loomux hook pre-tool-use"}]}]}}`,
			Entry{Event: "PreToolUse", Matcher: "Write|Edit|Bash", Command: "loomux hook pre-tool-use"},
			"PreToolUse: kept an own entry under matcher (none)",
		},
	} {
		got, err := Merge(claude, []byte(tc.existing), []Entry{tc.wanted})
		if err != nil {
			t.Fatalf("%s: %v", tc.wanted.Event, err)
		}
		if len(got.Added) != 0 || !bytes.Equal(got.Merged, []byte(tc.existing)) || !reflect.DeepEqual(got.Notes, []string{tc.note}) {
			t.Fatalf("%s: added %v, notes %v", tc.wanted.Event, got.Added, got.Notes)
		}
	}
}
```

In `internal/setup/plan_test.go` wird `TestAnOwnEntryUnderAnOldMatcherIsNamed` ganz ersetzt durch:

```go
func TestAnOwnEntryUnderAnOldMatcherGetsABlockForTheToolItLacks(t *testing.T) {
	settings := `{"hooks": {"PreToolUse": [{"matcher": "Write|Edit|NotebookEdit|Bash|PowerShell", "hooks": ` +
		`[{"type": "command", "command": "loomux hook pre-tool-use --host claude"}]}]}}`
	root := world(t, map[string]string{".claude/settings.json": settings})
	p := plan(t, gather(t, root, ""))
	c, _ := changeOf(p, ".claude/settings.json")
	if strings.Count(c.After, "hook pre-tool-use") != 2 || !strings.Contains(c.After, `"matcher": "MultiEdit"`) ||
		!hasNote(p, ".claude/settings.json: PreToolUse: kept an own entry under matcher Write|Edit|NotebookEdit|Bash|PowerShell; added one for MultiEdit") {
		t.Errorf("settings =\n%s\nnotes = %v", c.After, p.Notes)
	}
}
```

Der Plan schreibt nach dieser Änderung Folgendes: `hostEntries` (`internal/setup/plan.go:314-335`) übernimmt `Result.Merged` als `After` des Changes für `.claude/settings.json` bzw. `.agents/hooks.json`. Der Diff zeigt den alten Block unverändert und dahinter einen zweiten Block `{"matcher": "MultiEdit"|"manage_task", "hooks": [{"type": "command", "command": <der Befehl aus Entries>, "timeout": 15}]}`. Die Notiz steht als `<pfad>: PreToolUse: kept an own entry under matcher …; added one for …` in `p.Notes`. `Added` selbst gibt der Plan nicht aus, aber die Datei wird geschrieben, sobald der Mensch zustimmt.

- [ ] **Step 2: Rot prüfen**

Run: `go test ./internal/setup/hostfile/ ./internal/setup/ -run 'TestMergeAddsABlock|TestMergeAddsNothing|TestMergeOnlyNotes|TestMergeRecognises|TestAnOwnEntryUnderAnOldMatcher' -count=1`

Expected (so gesehen, Zeilennummern je nach Stand):
```
--- FAIL: TestMergeRecognisesLoomuxEntriesWhereverTheyStand (0.00s)
    merge_test.go:579: old matcher: added [], notes [PreToolUse: kept an own entry under matcher Write|Edit|NotebookEdit|Bash|PowerShell]
--- FAIL: TestMergeAddsABlockForTheToolsAnOldMatcherLacks (0.00s)
    merge_test.go:716: added [], want the block for manage_task
--- FAIL: TestMergeAddsNothingTwiceForASplitMatcher (0.00s)
--- FAIL: TestMergeAddsNothingWhereTwoOldBlocksCoverTheMatcher (0.00s)
    merge_test.go:768: added [], notes [PreToolUse: kept an own entry under matcher Write|Edit]
--- FAIL: TestMergeOnlyNotesAMatcherlessEntryKeptUnderAMatcher (0.00s)
    merge_test.go:…: PreToolUse: added [PreToolUse/Write|Edit|Bash], notes []
FAIL	github.com/xidus90/loomux/internal/setup/hostfile
--- FAIL: TestAnOwnEntryUnderAnOldMatcherGetsABlockForTheToolItLacks (0.01s)
    plan_test.go:688: settings =
FAIL	github.com/xidus90/loomux/internal/setup
```
`TestMergeOnlyNotesAnOldMatcherItCannotCountOrThatLacksNothing` und der `Stop`-Fall von `TestMergeOnlyNotesAMatcherlessEntryKeptUnderAMatcher` sind schon heute grün. Sie halten fest, was sich nicht ändern darf, und die Mutationsprobe in Step 5 macht sie rot. Der `PreToolUse`-Fall ohne `matcher`-Schlüssel ist heute rot: `Merge` hängt dort den gewünschten Eintrag daneben (beim Review gemessen), nach Task 4 behält es den Block und nennt ihn `(none)`.

- [ ] **Step 3: Implementieren**

In `internal/setup/hostfile/merge.go` wird der erste Absatz der Paket-Doku (Zeilen 4–9) ersetzt durch:

```go
// An entry is ours when its command calls a loomux binary (Owned); nothing
// else marks it. An entry of ours already on its event and matcher is kept
// as it is, whatever its command says, and a hook of the project on the
// same slot is reported and left alone. An entry of ours under an older
// matcher is kept and named too; where it and the wanted matcher are flat
// lists of tool names, a block for the tools it lacks is appended beside
// it, so an upgrade guards a tool that joined the matcher since. Nothing is
// rewritten and nothing removed: where the merge has nothing to add, the
// file comes back byte for byte.
```

`Result` wird ganz ersetzt durch:

```go
// Result is what Merge made of a hook file. Every slot is named
// "<Event>/<Matcher>".
type Result struct {
	Merged  []byte   // equal to the input when Added is empty
	Added   []string // an entry of ours appended
	Kept    []string // an entry of ours already there
	Foreign []string // someone else's hook on the same event and matcher, kept
	// Notes names an entry of ours kept under another matcher, with the
	// tools a block was appended for, and, in
	// Antigravity's file, another group that already runs one of our
	// commands.
	Notes []string
}
```

`Merge` wird ganz ersetzt durch:

```go
// Merge adds the wanted entries of host to existing, the content of its hook
// file (nil when there is none yet).
func Merge(host hosts.Host, existing []byte, wanted []Entry) (Result, error) {
	file := Path(host)
	if file == "" {
		return Result{}, fmt.Errorf("host %s has no hook file", host)
	}
	key := container(host)
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return Result{}, fmt.Errorf("%s is not a JSON object: %w", file, err)
		}
		// null is the one document that parses into a nil map without an
		// error; writing over it would repair a file that is no object.
		if root == nil {
			return Result{}, fmt.Errorf("%s is not a JSON object: its root is null", file)
		}
	}
	raw, present := root[key]
	hooks, ok := raw.(map[string]any)
	if present && !ok {
		return Result{}, fmt.Errorf("%s: [%s] is not an object", file, key)
	}
	if hooks == nil {
		hooks = map[string]any{}
	}

	result := Result{Merged: existing}
	if host == hosts.HostAntigravity {
		result.Notes = twiceRun(root, key, wanted)
	}
	for _, entry := range wanted {
		rawList, listed := hooks[entry.Event]
		list, ok := rawList.([]any)
		if listed && !ok {
			return Result{}, fmt.Errorf("%s: [%s].%s is not a list", file, key, entry.Event)
		}
		slot := entry.Event + "/" + entry.Matcher
		own, foreign, elsewhere, stale := find(list, entry)
		if foreign {
			result.Foreign = append(result.Foreign, slot)
		}
		if stale != "" {
			result.Notes = append(result.Notes, slot+": kept an own entry that runs "+stale+
				" instead of "+entry.Command+"; update it by hand")
		}
		if !own && len(elsewhere) > 0 {
			// A block without a matcher key runs for every tool; the note
			// names it so rather than with an empty name.
			matchers := slices.Clone(elsewhere)
			for i, m := range matchers {
				if m == "" {
					matchers[i] = "(none)"
				}
			}
			kept := entry.Event + ": kept an own entry under matcher " + strings.Join(matchers, ", ")
			if missing, counted := missingTools(entry.Matcher, elsewhere); counted && len(missing) > 0 {
				rest := entry
				rest.Matcher = strings.Join(missing, "|")
				hooks[entry.Event] = append(list, blockFor(rest))
				result.Added = append(result.Added, entry.Event+"/"+rest.Matcher)
				result.Notes = append(result.Notes, kept+"; added one for "+rest.Matcher)
				continue
			}
			result.Notes = append(result.Notes, kept)
			own = true
		}
		if own {
			result.Kept = append(result.Kept, slot)
			continue
		}
		hooks[entry.Event] = append(list, blockFor(entry))
		result.Added = append(result.Added, slot)
	}
	if len(result.Added) == 0 {
		return result, nil
	}
	root[key] = orderHooks(hooks)
	result.Merged = formatRoot(extractTopEntries(existing), root, key)
	return result, nil
}
```

`find` wird ganz ersetzt, und direkt danach (vor `commandsOf`) kommen die beiden neuen Funktionen:

```go
// find looks at the blocks of list for entry: own when a block on its
// matcher calls loomux in any of its commands, foreign when a block there
// does not. elsewhere is the matcher of every own block under another
// matcher that runs the same hook, in file order -- an entry from before the
// matcher changed, such as one of ours under ulinit's, and a block Merge
// appended beside it for the tools it lacked -- and empty when there is
// none; Merge counts what they cover, so no tool runs the hook twice. A
// block without a matcher key counts as "". stale is the first
// loomux command of an own block on the matcher when none of them is
// entry's command -- an old binary or an old subcommand, kept but not
// current -- and "" otherwise.
func find(list []any, entry Entry) (own, foreign bool, elsewhere []string, stale string) {
	event := hookEventOf(entry.Command)
	current := false
	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		var mine []string
		sameHook := false
		for _, command := range commandsOf(item) {
			if Owned(command) {
				mine = append(mine, command)
				sameHook = sameHook || (event != "" && hookEventOf(command) == event)
			}
		}
		switch {
		case matcherOf(item) == entry.Matcher && len(mine) > 0:
			own = true
			current = current || slices.Contains(mine, entry.Command)
			if stale == "" {
				stale = mine[0]
			}
		case matcherOf(item) == entry.Matcher:
			foreign = true
		case sameHook:
			elsewhere = append(elsewhere, matcherOf(item))
		}
	}
	if current {
		stale = ""
	}
	return own, foreign, elsewhere, stale
}

// missingTools is the names of want that none of have names, in the order
// of want. counted is false when a matcher is no flat list of names -- a
// regular expression, an empty one: what it covers cannot be counted, so
// nothing is added beside it.
func missingTools(want string, have []string) (missing []string, counted bool) {
	names, counted := toolNames(want)
	if !counted {
		return nil, false
	}
	covered := map[string]bool{}
	for _, matcher := range have {
		got, counted := toolNames(matcher)
		if !counted {
			return nil, false
		}
		for _, name := range got {
			covered[name] = true
		}
	}
	for _, name := range names {
		if !covered[name] {
			missing = append(missing, name)
		}
	}
	return missing, true
}

// toolNames splits a flat matcher at | into its names, each of ASCII
// letters, digits and underscores; ok is false for anything else.
func toolNames(matcher string) (names []string, ok bool) {
	names = strings.Split(matcher, "|")
	for _, name := range names {
		if name == "" || strings.ContainsFunc(name, func(r rune) bool {
			return r != '_' && (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
		}) {
			return nil, false
		}
	}
	return names, true
}
```

Imports bleiben unverändert (`strings` und `slices` sind schon da). Außer `Merge` ruft niemand `find` (grep `find(` in `internal/setup/hostfile`).

- [ ] **Step 4: Grün prüfen**

Run: `go test ./internal/setup/hostfile/ ./internal/setup/ -run 'TestMergeAddsABlock|TestMergeAddsNothing|TestMergeOnlyNotes|TestMergeRecognises|TestAnOwnEntryUnderAnOldMatcher' -count=1`, dann `go test ./internal/setup/... -count=1`.
Expected: `ok  github.com/xidus90/loomux/internal/setup/hostfile` und `ok  github.com/xidus90/loomux/internal/setup`, alle anderen `internal/setup/...` ebenfalls `ok`. Zur Coverage: `go test ./internal/setup/hostfile/ -coverprofile=<scratch>/c.out -count=1 && go tool cover -func=<scratch>/c.out | grep merge.go`. `Merge`, `find`, `missingTools`, `toolNames` und `commandsOf` müssen je 100.0% zeigen. Achtung: `-overlay` zusammen mit `-coverprofile` instrumentiert die Originaldatei, nicht die Overlay-Kopie. Coverage also nur im echten Baum messen.

- [ ] **Step 5: Mutationsprobe** (je eine per `go test -overlay` auf einer Kopie von `merge.go`, der Baum bleibt unverändert; Aufruf wie in Step 4)
  - `counted && len(missing) > 0` → `counted`: rot werden `TestMergeOnlyNotesAnOldMatcherItCannotCountOrThatLacksNothing` (Obermenge), `TestMergeAddsNothingTwiceForASplitMatcher` und `TestMergeAddsNothingWhereTwoOldBlocksCoverTheMatcher`.
  - `for _, matcher := range have {` → `for _, matcher := range have[:1] {` (nur der erste Block zählt): rot werden `TestMergeAddsNothingWhereTwoOldBlocksCoverTheMatcher` und `TestMergeAddsNothingTwiceForASplitMatcher`.
  - in `find` `case sameHook:` → `case sameHook && len(elsewhere) == 0:` (das alte „nur der erste“): dieselben zwei werden rot.
  - in `toolNames` `name == "" || strings.ContainsFunc(` → `name == "" && strings.ContainsFunc(` (Zeichenprüfung weg): rot werden `TestMergeOnlyNotesAnOldMatcherItCannotCountOrThatLacksNothing` und `TestMergeOnlyNotesAMatcherlessEntryKeptUnderAMatcher`.
  - `name == "" ||` → `false ||` (leerer Name gilt als Name): dieselben zwei werden rot.
  - in `Merge` die Schleife `for i, m := range matchers { … }` entfernen (leerer Matcher ohne Namen): rot wird `TestMergeOnlyNotesAMatcherlessEntryKeptUnderAMatcher` (`notes [PreToolUse: kept an own entry under matcher ]`).
  Die ersten fünf Mutanten wurden beim Planen geprüft und sind gefallen, der sechste beim Einarbeiten des Reviews (per Overlay; dort auch Rot und Grün des neuen Falls und `go vet` still). Den `PreToolUse`-Fall ohne `matcher`-Schlüssel erreicht der dritte Mutant (`case sameHook && len(elsewhere) == 0`) nicht: dort gibt es nur einen eigenen Block.

- [ ] **Step 6: Doku**

`docs/en/hooks.md`, Abschnitt 7: Direkt nach dem Absatz „**Paths are compared relative to the root.** …“ (endet mit „`.loomux/config.toml`; where it finds none, only the built-in rules apply.“) und vor „**Every reason, not the first.**“ kommt ein neuer Absatz. Der Absatz „**Paths and command lines, not content.**“ davor gehört dem Guard-Bereich und bleibt hier unberührt.

```markdown
**An older matcher gets a block beside it.** `loomux init` never rewrites a
hook entry of ours. When one of ours stands under a matcher an earlier release
or a hand edit gave it — a loomux command that replaced ulinit's under
`Write|Edit|NotebookEdit|Bash|PowerShell` in `.claude/settings.json`, or an
Antigravity group from before `manage_task` joined `PreToolUse` — init keeps
it, appends a second block with the same command for the tools it lacks
(`MultiEdit`, `manage_task`), and says so in a note. That works only where both
matchers are plain lists of tool names joined by `|`; an entry under a regular
expression such as `.*`, or without a matcher, is kept and named, and what it
misses is added by hand. A second run counts both blocks and adds nothing. An
entry that still runs `ulguard` is not ours: init adds the whole loomux block
beside it and leaves ulguard to you.
```

`docs/de/hooks.md`, Abschnitt 7: Direkt nach dem Absatz „**Pfade werden relativ zur Wurzel verglichen.** …“ (endet mit „eingebauten Regeln.“) und vor „**Jeder Grund, nicht der erste.**“ kommt:

```markdown
**Ein älterer Matcher bekommt einen Block daneben.** `loomux init` schreibt
einen eigenen Hook-Eintrag nie um. Steht einer unserer Einträge unter einem
Matcher, den ihm ein früheres Release oder eine Handänderung gab — ein
loomux-Befehl, der ulinits unter `Write|Edit|NotebookEdit|Bash|PowerShell` in
`.claude/settings.json` ersetzt hat, oder eine Antigravity-Gruppe von vor
`manage_task` in `PreToolUse` —, lässt `init` ihn stehen, hängt für die
fehlenden Werkzeuge (`MultiEdit`, `manage_task`) einen zweiten Block mit
demselben Befehl an und sagt es in einer Notiz. Das geht nur, wo beide
Matcher schlichte Listen von Werkzeugnamen mit `|` dazwischen sind; ein
Eintrag unter einem regulären Ausdruck wie `.*` oder ohne Matcher bleibt
stehen und wird genannt, und was ihm fehlt, ergänzt man von Hand. Ein zweiter
Lauf zählt beide Blöcke und fügt nichts hinzu. Ein Eintrag, der noch
`ulguard` ruft, ist nicht unserer: `init` fügt den ganzen loomux-Block
daneben ein und überlässt ulguard dir.
```

`docs/en/getting-started.md`, Abschnitt „### Google Antigravity“: Der Absatz nach dem JSON-Block (`` `loomux init` writes this group; agy runs it through `cmd.exe` from `.agents/`, hence `%LOCALAPPDATA%` and `--root ..`. The CLI reference explains how the hooks answer Antigravity. ``) wird ganz ersetzt durch:

```markdown
`loomux init` writes this group; agy runs it through `cmd.exe` from `.agents/`, hence `%LOCALAPPDATA%` and `--root ..`. Run `loomux init` again after an upgrade: an entry of ours under an older matcher stays as it is, and init appends a block for the tools it lacks, such as `manage_task`. The CLI reference explains how the hooks answer Antigravity.
```

`docs/de/getting-started.md`, Abschnitt „### Google Antigravity“: Der Absatz nach dem JSON-Block (`` `loomux init` schreibt diese Gruppe; agy führt sie über `cmd.exe` aus `.agents/` aus, daher `%LOCALAPPDATA%` und `--root ..`. Wie die Hooks Antigravity antworten, steht in der CLI-Referenz. ``) wird ganz ersetzt durch:

```markdown
`loomux init` schreibt diese Gruppe; agy führt sie über `cmd.exe` aus `.agents/` aus, daher `%LOCALAPPDATA%` und `--root ..`. Nach einem Upgrade `loomux init` erneut ausführen: Ein eigener Eintrag unter einem älteren Matcher bleibt, wie er ist, und `init` hängt einen Block für die fehlenden Werkzeuge an, etwa `manage_task`. Wie die Hooks Antigravity antworten, steht in der CLI-Referenz.
```

`docs/en/cli-reference.md`, Abschnitt 11, Punkt „**Antigravity** gets four entries …“ (Zeilen 1321-1322): der umbrochene Satz

```
An entry from before `manage_task` joined the matcher is kept and
  named in a note; add `|manage_task` to it by hand.
```

(der Text `An entry from before` … `and` steht am Zeilenende, vor dem Guard-Bereich-Commit hinter `refused. `, danach hinter `carry no line. `) wird ersetzt durch:

```
An entry from before `manage_task` joined the matcher is kept, and init
  appends a block for `manage_task` beside it with the same command and a
  note; a Claude Code entry of ours under ulinit's matcher gets one for
  `MultiEdit` the same way. An entry under a matcher that is no plain list of tool
  names, such as `.*`, is only named, and what it lacks is added by hand.
```

Das folgende `It also gets` bleibt am Anfang der nächsten Zeile stehen.

`docs/de/cli-reference.md`, derselbe Punkt (Zeilen 1378-1379): der umbrochene Satz

```
Ein Eintrag von vor `manage_task` im Matcher bleibt stehen und
  wird in einer Notiz genannt; `|manage_task` ergänzt man von Hand.
```

wird ersetzt durch:

```
Ein Eintrag von vor `manage_task` im Matcher bleibt stehen, und `init`
  hängt mit demselben Befehl und einer Notiz einen Block für `manage_task`
  daneben; ein eigener Eintrag von Claude Code unter ulinits Matcher bekommt
  ebenso einen für `MultiEdit`. Ein Eintrag unter einem Matcher, der keine
  schlichte Liste von Werkzeugnamen ist, etwa `.*`, wird nur genannt, und
  was ihm fehlt, ergänzt man von Hand.
```

Das folgende `Dazu kommen die Skills unter` bleibt in der nächsten Zeile stehen.

(Der Guard-Bereich (Task 1) ändert in denselben Punkten den Satz davor. Hier wird nur der eine Satz ersetzt; wer später landet, setzt auf dem Stand des anderen auf.)

Danach per grep prüfen, ob noch eine Stelle das alte Verhalten beschreibt: `grep -rnw "by hand\|von Hand" docs/en docs/de README.md README.de.md | grep -i "matcher\|manage_task"`. Erwartet: nur die neuen Sätze in `cli-reference.md`; `docs/de/migration.md:39` erscheint ohne `-w` über „von Handlern“ und bleibt unverändert. Achtung: der Satz ist umbrochen, grep sieht nur die Zeile mit „by hand“/„von Hand“; zusätzlich `grep -rn --exclude=2026-09-27-antigravity-review-findings.md "add .|manage_task. to it\|.|manage_task. ergänzt" docs`. Erwartet: kein Treffer.

- [ ] **Step 7: Commit**

Vorher `git rev-parse --show-toplevel` gegen den Worktree-Pfad und `git branch --show-current` lesen. Dann:

```sh
git add internal/setup/hostfile/merge.go internal/setup/hostfile/merge_test.go internal/setup/plan_test.go docs/en/hooks.md docs/de/hooks.md docs/en/getting-started.md docs/de/getting-started.md docs/en/cli-reference.md docs/de/cli-reference.md
```

Die Nachricht per Write in `<scratch>/commit-append-block.txt`:

```
fix(setup): append a block for the tools an own matcher lacks

An own hook entry under an older matcher -- a loomux command under
ulinit's Write|Edit|NotebookEdit|Bash|PowerShell in .claude/settings.json,
or an Antigravity group from before manage_task joined PreToolUse -- was
kept with a note, so init left every tool that joined the matcher since
unguarded on an existing installation.

Where the wanted matcher and every own one under another matcher are flat
lists of tool names, Merge now appends a block with the same command for
the names none of them covers and says so in the note. find returns all
such matchers, so a second run counts the old block and the appended one
together and adds nothing. A regular-expression or empty matcher still
gets the note alone. Nothing is rewritten or removed.
```

`git commit -F <scratch>/commit-append-block.txt > <scratch>/commit-append-block.log 2>&1`. Der pre-commit-Hook fährt `sh ci/gate.sh`. Wird er rot, zuerst `<scratch>/commit-append-block.log` nach `--- FAIL` durchsuchen. Danach `git log -1 --format='%an <%ae>'` lesen: Autor ist der Mensch, kein Co-Authored-By.

---

### Task 5: Die versionierte `.agents/hooks.json` an `Entries` halten

**Files:**
- Modify: `.agents/hooks.json:5` (PreToolUse-Matcher)
- Test: `internal/setup/hostfile/merge_test.go` (neuer Test direkt nach `TestTheRepositorysOwnSettingsNeedNoChange`, ~673)

**Interfaces:**
- Consumes: `Merge`, `Entries`, `BinaryOf` (unverändert), `hosts.HostAntigravity`. Hängt nicht an der Messung aus Task 0 und nicht an Task 4: rot ist der Test mit und ohne Task 4 (ohne Task 4 mit der Notiz allein, mit Task 4 mit `Added [PreToolUse/manage_task]`).
- Produces: `TestTheRepositorysOwnAntigravityHooksNeedNoChange`.

- [ ] **Step 1: Failing test schreiben** — direkt nach `TestTheRepositorysOwnSettingsNeedNoChange` in `internal/setup/hostfile/merge_test.go`. Er ist genauso gebaut: dieselbe relative Datei, `BinaryOf` für das Binary und dieselben vier Bedingungen.

```go
// The tracked .agents/hooks.json drifts the same way: its matcher stayed
// behind Entries when manage_task joined it. This holds it to what init
// would write, as the test above holds .claude/settings.json.
func TestTheRepositorysOwnAntigravityHooksNeedNoChange(t *testing.T) {
	own, err := os.ReadFile("../../../.agents/hooks.json")
	if err != nil {
		t.Fatal(err)
	}
	host := hosts.HostAntigravity
	got, err := Merge(host, own, Entries(host, BinaryOf(host, own)))
	if err != nil || len(got.Added) != 0 || len(got.Notes) != 0 || !bytes.Equal(got.Merged, own) {
		t.Fatalf("added %v, notes %v, err %v; init would change .agents/hooks.json", got.Added, got.Notes, err)
	}
}
```

- [ ] **Step 2: Rot prüfen**

Run: `go test ./internal/setup/hostfile/ -run 'TestTheRepositorysOwn' -count=1`
Expected (nach Task 4 gesehen):
```
--- FAIL: TestTheRepositorysOwnAntigravityHooksNeedNoChange (0.00s)
    merge_test.go:821: added [PreToolUse/manage_task], notes [PreToolUse: kept an own entry under matcher write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input; added one for manage_task], err <nil>; init would change .agents/hooks.json
FAIL	github.com/xidus90/loomux/internal/setup/hostfile
```
Wird Task 4 wegen Task 0 nicht gebaut, lautet die Zeile `added [], notes [PreToolUse: kept an own entry under matcher write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input], …`.

- [ ] **Step 3: Implementieren** — in `.agents/hooks.json` wird nur Zeile 5 von Hand geändert (ein Block, keine Anhängung, kein `loomux init`):

```json
        "matcher": "write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input|manage_task",
```

Einrückung, Zeilenenden und der Rest der Datei bleiben Byte für Byte; `Merge` gibt bei nichts Hinzuzufügendem die Eingabe zurück. Der Matcher ist wörtlich der aus `internal/setup/hostfile/table.go:63`. Die Beispiele in `docs/en/getting-started.md:256` und `docs/de/getting-started.md:259` tragen ihn schon mit `manage_task` und bleiben unverändert.

- [ ] **Step 4: Grün prüfen**

Run: `go test ./internal/setup/hostfile/ -run 'TestTheRepositorysOwn' -count=1 -v`, dann `go test ./internal/setup/hostfile/ -count=1`.
Expected: `--- PASS: TestTheRepositorysOwnSettingsNeedNoChange`, `--- PASS: TestTheRepositorysOwnAntigravityHooksNeedNoChange`, `ok  github.com/xidus90/loomux/internal/setup/hostfile`. Keine Produktionsfunktion ändert sich, also keine neue Coverage-Frage.

- [ ] **Step 5: Mutationsprobe** — `|manage_task` aus Zeile 63 in `internal/setup/hostfile/table.go` (Antigravity-PreToolUse-Matcher) per Overlay entfernen. Dann muss `TestTheRepositorysOwnAntigravityHooksNeedNoChange` rot werden: Der Matcher der Datei (mit `manage_task`) ist nun eine flache Obermenge des gewünschten, also hängt `Merge` nach Task 4 nichts an (`Added` bleibt leer), notiert aber `PreToolUse: kept an own entry under matcher …|manage_task`, und der Test verlangt keine Notiz. Zweite Probe: `.agents/hooks.json` kann man nicht per Overlay tauschen, weil der Test sie zur Laufzeit liest. Darum reicht als Gegenprobe Step 2 auf der unveränderten Datei.

- [ ] **Step 6: Doku** — keine Verhaltensänderung für Nutzer. Den datierten Nachtrag in `docs/.superpowers/parity/stufe-4a-2.md` („Jetzt im Matcher“ ~684 galt nur für `table.go`) schreibt Task 13.

- [ ] **Step 7: Commit**

Vorher `git rev-parse --show-toplevel` und `git branch --show-current` lesen. Dann:

```sh
git add .agents/hooks.json internal/setup/hostfile/merge_test.go
```

Die Nachricht per Write in `<scratch>/commit-tracked-hooks.txt`:

```
fix(setup): guard manage_task in the tracked Antigravity hook file

The repository's own .agents/hooks.json still carried the PreToolUse
matcher from before manage_task joined it, so an agy session in this
checkout typed into a task past the command rules. Its matcher is now the
one Entries writes, and a test holds the file to what init would write,
as one already does for .claude/settings.json.
```

`git commit -F <scratch>/commit-tracked-hooks.txt > <scratch>/commit-tracked-hooks.log 2>&1`, danach `git log -1 --format='%an <%ae>'`.

---

### Task 6: Kommentare zu Blöcken mit flachem Befehl und hooks-Liste

**Files:**
- Modify: `internal/setup/hostfile/merge.go` (Kommentar über `commandsOf`, auf a7805df8 Zeilen 244–246)
- Modify: `internal/setup/hostfile/merge_test.go` (Kommentar über `TestMergeFindsAnEntryInTheListOfABlockWithACommand`, auf a7805df8 ~643)

**Interfaces:**
- Consumes: nichts Neues. Produces: nichts; nur Kommentare, keine Codeänderung an `find` oder `commandsOf`.

- [ ] **Step 1: Failing test schreiben** — entfällt, nur Kommentare. Der Beweis ist, dass `gofmt -l` leer bleibt und die Tests unverändert grün sind.
- [ ] **Step 2: Rot prüfen** — entfällt.
- [ ] **Step 3: Implementieren**

Der Kommentar über `commandsOf` in `internal/setup/hostfile/merge.go` wird ersetzt. Die Funktion bleibt unverändert und steht hier ganz:

```go
// commandsOf is every command of a block, in order: a flat handler's own
// command first, then those of its hooks list. loomux never writes a block
// with both, and whether a host runs one of them or both is not measured, so
// both are read, and a command of ours in either makes the block ours.
func commandsOf(item map[string]any) []string {
	var out []string
	if command, ok := item["command"].(string); ok {
		out = append(out, command)
	}
	hooks, _ := item["hooks"].([]any)
	for _, raw := range hooks {
		hook, _ := raw.(map[string]any)
		if command, ok := hook["command"].(string); ok {
			out = append(out, command)
		}
	}
	return out
}
```

Der Kommentar über dem Test in `internal/setup/hostfile/merge_test.go` wird ersetzt. Der Test bleibt unverändert:

```go
// A block that holds a command of its own beside a hooks list: which of the
// two a host runs is not measured, so both are read. Our entry in the list
// is found, and none is added beside it.
func TestMergeFindsAnEntryInTheListOfABlockWithACommand(t *testing.T) {
	entry := Entry{Event: "Stop", Command: Canonical + " hook stop --host claude"}
	existing := []byte(`{"hooks":{"Stop":[{"command":"echo x","hooks":[{"type":"command","command":` +
		string(EncodeJSON(entry.Command, "", "")) + `}]}]}}`)
	got, err := Merge(claude, existing, []Entry{entry})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Added) != 0 || !slices.Equal(got.Kept, []string{"Stop/"}) {
		t.Fatalf("added %v, kept %v; want the entry found", got.Added, got.Kept)
	}
}
```

- [ ] **Step 4: Grün prüfen** — Run: `gofmt -l internal/setup/hostfile/`, Expected: keine Ausgabe. Dann `go test ./internal/setup/hostfile/ -count=1`, Expected: `ok  github.com/xidus90/loomux/internal/setup/hostfile`. Und `go vet ./internal/setup/hostfile/`, Expected: keine Ausgabe.
- [ ] **Step 5: Mutationsprobe** — entfällt, weil sich kein Verhalten ändert. Zur Kontrolle, dass der Kommentar stimmt (beim Review gemessen): `if command, ok := item["command"].(string); ok {` per Overlay auf `if command, ok := item["command"].(string); ok && false {` setzen (ein bloßes `if false {` baut nicht, weil der Rumpf `command` braucht). Erwartet: rot `TestMergeWritesAntigravitysFlatEventsFlat` und `TestMergeNamesAnotherGroupThatRunsOurCommand` (beide mergen eine Datei erneut, deren flache `PreInvocation`/`Stop`-Handler dann nicht mehr erkannt werden: `added [PreInvocation/ Stop/]`), grün `TestBinaryOfReadsEveryCommandOfABlock` (dort ist die Liste gemeint). Kommt es anders, das Ergebnis im Bericht nennen; die Kommentaränderung hängt nicht daran.
- [ ] **Step 6: Doku** — keine Nutzerdoku; die Stellen sind die zwei Kommentare oben.
- [ ] **Step 7: Commit**

Vorher `git rev-parse --show-toplevel` und `git branch --show-current` lesen. Dann:

```sh
git add internal/setup/hostfile/merge.go internal/setup/hostfile/merge_test.go
```

Die Nachricht per Write in `<scratch>/commit-mixed-block.txt`:

```
docs(hostfile): say why both commands of a mixed block are read

The comments claimed a host runs both the flat command and the hooks list
of a block that carries both. That was never measured, and loomux never
writes such a block; they now say that both are read because which one a
host runs is unknown.
```

`git commit -F <scratch>/commit-mixed-block.txt > <scratch>/commit-mixed-block.log 2>&1`, danach `git log -1 --format='%an <%ae>'`.

---

### Task 7: Ein Satz für das Budget

**Files:**
- Modify: `internal/verify/report.go:71-101` (`SkipPrefix`, `EditReport`)
- Modify: `internal/hooks/post_edit.go:84-107` (`RunPostEdit`), `:109-153` (`checkEdit`)
- Test: `internal/verify/report_test.go`, `internal/hooks/post_edit_test.go`

**Interfaces:**
- Consumes: nichts aus früheren Tasks dieses Bereichs.
- Produces:
  - `func verify.BudgetSkipped(name string) string` — die Zeile ohne Zeilenumbruch: `"loomux hook post-tool-use: lane skipped, the edit budget ran out: " + name`
  - `func verify.EditReport(stderr io.Writer, outs []Outcome, aside string) (red bool, notices []string)` — eine Zeile je Hinweis, ohne `\n`
  - `const skipPrefix` (unexportiert; `SkipPrefix` entfällt). Entscheidung: Nach Task 10 braucht außerhalb von `verify` niemand das Präfix — der einzige Nutzer war `post_edit_test.go:814`, der zählt künftig das Literal `"lane skipped, "`. Innerhalb von `verify` nutzen `report.go` und `report_test.go` es.
  - `checkEdit(...) (int, []string)` (unexportiert, hooks)

- [ ] **Step 1: Failing test schreiben**

In `internal/verify/report_test.go` den Helfer `writeEdit` (Zeile ~90-96) ersetzen und direkt dahinter den neuen Test einfügen:

```go
// writeEdit reports one file's lanes the way post-edit does for a call that
// names only that file.
func writeEdit(stdout, stderr io.Writer, outs []Outcome, aside string) int {
	red, notices := EditReport(stderr, outs, aside)
	WriteNotices(stdout, strings.Join(notices, "\n"))
	if red {
		return 2
	}
	return 0
}

// The budget's notice is the one cli-reference.md shows, for a lane and for
// a file alike.
func TestBudgetSkippedIsTheDocumentedSentence(t *testing.T) {
	want := "loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go"
	if got := BudgetSkipped("lint/go"); got != want {
		t.Fatalf("%q", got)
	}
}
```

In `internal/hooks/post_edit_test.go`:
- `TestPostEditSharesOneBudgetAcrossTheFiles` (~Zeile 792): `skipped := "the edit budget ran out: " + filepath.ToSlash(filepath.Join(root, "b.go"))` wird zu
  ```go
  	skipped := verify.BudgetSkipped(filepath.ToSlash(filepath.Join(root, "b.go")))
  ```
- `TestPostEditSaysTheSkipsOfEveryFileInOneDocument` (~Zeile 814): `if strings.Count(said, verify.SkipPrefix) != 2 {` wird zu
  ```go
  	if strings.Count(said, "lane skipped, ") != 2 {
  ```

- [ ] **Step 2: Rot prüfen**

Run: `go test ./internal/verify/ -run 'TestBudgetSkipped' -count=1`
Expected (gesehen, gegen den unveränderten Code):
```
report_test.go:94:36: cannot use notices (variable of type string) as []string value in argument to strings.Join
report_test.go:95:5: non-boolean condition in if statement
report_test.go:105:12: undefined: BudgetSkipped
FAIL	github.com/xidus90/loomux/internal/verify [build failed]
```
(Ein Refactor der Signatur: rot ist hier der Bau; der Wortlaut-Test ist das Ziel der Mutationsprobe.)

- [ ] **Step 3: Implementieren**

`internal/verify/report.go`: `SkipPrefix` und `EditReport` (Zeile 71-101) ersetzen durch:

```go
// skipPrefix begins every notice of a lane or a file post-edit did not run.
const skipPrefix = "loomux hook post-tool-use: lane skipped, "

// BudgetSkipped is the notice for a lane or a file the edit budget did not
// reach, named by name: one sentence for both, so the model reads one form.
func BudgetSkipped(name string) string {
	return skipPrefix + "the edit budget ran out: " + name
}

// EditReport reports the lanes of one edited file: red lanes on stderr,
// which blocks the edit, and as notices for the model the lanes it had to
// skip and whatever else the hook has to say, one line each. The aside is
// dropped when a lane is red: the finding matters more, and stderr stays the
// finding's. The notices of every file of a call go to WriteNotices together,
// because a host reads stdout as one document.
func EditReport(stderr io.Writer, outs []Outcome, aside string) (red bool, notices []string) {
	for _, o := range outs {
		switch {
		case Red(o.State, ScopeEdit):
			red = true
			fmt.Fprintf(stderr, "%s: %s\n", o.Job.Name, o.State)
			if o.Output != "" {
				fmt.Fprintf(stderr, "%s\n", strings.TrimSuffix(o.Output, "\n"))
			}
		case o.State == StateBudget:
			notices = append(notices, BudgetSkipped(o.Job.Name))
		case o.State == StateMissingTool, o.State == StateUnready:
			notices = append(notices, skipPrefix+o.Output)
		}
	}
	if aside != "" && !red {
		notices = append(notices, aside)
	}
	return red, notices
}
```

`WriteNotices` bleibt in diesem Task unverändert (fällt in Task 10).

`internal/hooks/post_edit.go`, ganzes `RunPostEdit`:

```go
// RunPostEdit runs the lanes of the `edit` profile for the edited file's
// stack, as [verify] and the presets lay them out, or the wiki lint for a
// wiki page. A red lane blocks the edit with 2; a config it cannot read ends
// with 1, which shows the error and blocks nothing. A call that names several
// files checks each, within one budget for them all, and ends with the worst
// of their codes.
func RunPostEdit(stdin io.Reader, stdout, stderr io.Writer, root string, env EditEnv) int {
	files := editedFiles(stdin)
	if len(files) == 0 {
		return ExitOK
	}
	// Tools run in their area, so every path handed to them must be absolute.
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	facts := detect.Detect(os.DirFS(root))
	eff, err := editLoad(root, facts)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: %v\n", err)
		return ExitInternal
	}
	// One budget for the call: a host's timeout is per hook, not per file.
	// The first file runs as a single edit always has; each later one gets
	// what is left, and none once that is spent.
	start := env.Now()
	runID := verify.NewRunID(start, os.Getpid())
	code := ExitOK
	var notices []string
	for i, raw := range files {
		fileEnv, id := env, runID
		// A budget of 0 is none, and stays none for every file.
		if i > 0 && env.Budget > 0 {
			fileEnv.Budget = start.Add(env.Budget).Sub(env.Now())
			if fileEnv.Budget <= 0 {
				// stdout for the model at exit 0; stderr as well, which a
				// host reads at exit 2 and agy keeps in its log.
				skipped := verify.BudgetSkipped(raw)
				notices = append(notices, skipped)
				fmt.Fprintln(stderr, skipped)
				continue
			}
			// Coverage files of their own; CleanCover matches `<runID>-`.
			id += "." + strconv.Itoa(i)
		}
		fileCode, said := checkEdit(stderr, root, raw, id, eff, facts, fileEnv)
		code = max(code, fileCode)
		notices = append(notices, said...)
	}
	verify.WriteNotices(stdout, strings.Join(notices, "\n"))
	return code
}
```

In `checkEdit` ändern sich Signatur, `fail`, der Ignored-Zweig und die Prüfung von `red`; ganze Funktion:

```go
// checkEdit runs the lanes for one edited file, and answers its code with
// what it has to tell the model.
func checkEdit(stderr io.Writer, root, raw, runID string, eff verify.Effective, facts detect.Facts, env EditEnv) (int, []string) {
	fail := func(err error) (int, []string) {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: %v\n", err)
		return ExitInternal, nil
	}
	ext := strings.ToLower(filepath.Ext(raw))
	if slices.Contains(eff.Ignored, ext) {
		return ExitOK, nil
	}
	var jobs []verify.Job
	var err error
	if eff.Extensions[ext] == "wiki" {
		jobs = wikiJobs(eff, facts, root, raw)
	} else if jobs, err = editJobs(eff, root, raw, runID, env); err != nil {
		return fail(err)
	}
	if err := verify.PrepareCover(root); err != nil {
		return fail(err)
	}
	outs := verify.Run(jobs, verify.RunOptions{
		Scope: verify.ScopeEdit, MaxParallel: eff.Config.MaxParallel, Timeout: eff.Config.Timeout,
		Budget: env.Budget, Start: env.Start, Look: env.Look, Now: env.Now,
	})
	aside := ""
	if eff.Extensions[ext] == "go" && !slices.ContainsFunc(outs, func(o verify.Outcome) bool { return verify.Red(o.State, verify.ScopeEdit) }) {
		if rel, ok := relInRoot(root, raw); ok {
			aside = blastAside(root, rel, func(p string) ([]byte, error) {
				return os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
			})
		}
	}
	code := ExitOK
	red, notices := verify.EditReport(stderr, outs, aside)
	if red {
		code = ExitDenied
	}
	// The same rule as a check: a red edit keeps its files for whoever looks
	// into it, and a file left behind costs disk, not the verdict.
	if err := verify.CleanCover(root, runID, code == ExitOK); err != nil {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: cleaning coverage files: %v\n", err)
	}
	return code, notices
}
```

- [ ] **Step 4: Grün prüfen**

Run: `go test ./internal/verify/ -run 'TestBudgetSkipped' -count=1`, dann `go test ./internal/verify/ ./internal/hooks/ ./internal/cli/ -count=1` und `go vet ./internal/verify/ ./internal/hooks/`
Expected: `ok  	github.com/xidus90/loomux/internal/verify`, `ok  	.../internal/hooks`, `ok  	.../internal/cli`; vet still. `BudgetSkipped`, `EditReport`, `RunPostEdit`, `checkEdit` je 100.0 % in `go tool cover -func`.

- [ ] **Step 5: Mutationsprobe**

In `BudgetSkipped` `"the edit budget ran out: "` zu `"the budget ran out: "` ändern → `TestBudgetSkippedIsTheDocumentedSentence` und `TestEditReportNamesEverySkipAndEveryRed` rot. In `EditReport` `if aside != "" && !red` zu `if aside != ""` → `TestEditReportDropsTheAsideOnRed` rot.

- [ ] **Step 6: Doku**

Kein Verhalten ändert sich; der Wortlaut in `docs/en|de/cli-reference.md` bleibt. Einziger Kommentar: der zu `skipPrefix` (oben im Code, „a lane or a file“).

- [ ] **Step 7: Commit**

```
git add internal/verify/report.go internal/verify/report_test.go internal/hooks/post_edit.go internal/hooks/post_edit_test.go
```
Nachricht per Write nach `<scratch>/commit-7.txt`:
```
refactor(verify): build the edit budget's skip notice in one place

The sentence for a lane and for a file the edit budget did not reach was
spelled twice, in the edit report and in post-edit's file loop.
BudgetSkipped builds it for both. EditReport answers whether a lane was
red and one notice per line instead of an exit code and a joined string,
and the skip prefix is no longer exported.
```
`git commit -F <scratch>/commit-7.txt > <scratch>/gate-7.out 2>&1`, danach `tail` der Datei; bei Rot nach `--- FAIL` suchen.

### Task 8: Der Testhelfer liest den Kontext nach Schlüssel

**Files:**
- Modify: `internal/hooks/post_edit_test.go:819-832` (`editContextOf`), `:653-659` (`TestPostEditNamesTheCallersOfAChangedSymbol`), Importe
- Test: `internal/hooks/post_edit_test.go`

**Interfaces:**
- Consumes: Task 7 (Tests stehen auf `verify.BudgetSkipped`).
- Produces: `func editContextOf(t testing.TB, stdout string) string` (Testhelfer, hooks); `type failRecorder`, `func refuses(stdout string) bool` (Testhelfer). Tasks 9 und 10 nutzen `editContextOf`.

- [ ] **Step 1: Failing test schreiben**

Importblock in `post_edit_test.go` um `"io"` (hinter `"errors"`) und `"runtime"` (hinter `"path/filepath"`) ergänzen. **Zuerst nur** den Parameter von `editContextOf` auf `testing.TB` weiten (Rumpf bleibt der alte Struct-Decode, `"io"` noch nicht importieren) und hinter `editContextOf` einfügen:

```go
// failRecorder is a testing.TB whose Fatalf records the failure and ends its
// goroutine, as the real one does, so that a helper's refusal can be tested.
type failRecorder struct {
	testing.TB
	failed bool
}

func (r *failRecorder) Helper() {}

func (r *failRecorder) Fatalf(string, ...any) {
	r.failed = true
	runtime.Goexit()
}

// refuses says whether editContextOf fails the test on stdout.
func refuses(stdout string) bool {
	r := &failRecorder{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		editContextOf(r, stdout)
	}()
	<-done
	return r.failed
}

// The helper reads the context as a host does, by its exact keys and from
// one document; a key in another case, a missing key and a second document
// all fail the test that uses it.
func TestEditContextOfReadsOnlyTheKeysAHostReads(t *testing.T) {
	for _, stdout := range []string{
		`{"HookSpecificOutput":{"AdditionalContext":"x"}}`,
		`{"hookSpecificOutput":{}}`,
		`{"hookSpecificOutput":{"additionalContext":"x"}}` + "\n" + `{"hookSpecificOutput":{"additionalContext":"y"}}`,
		``,
	} {
		if !refuses(stdout) {
			t.Errorf("the helper passed %q", stdout)
		}
	}
	if got := editContextOf(t, `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"x"}}`+"\n"); got != "x" {
		t.Fatalf("%q", got)
	}
}
```

- [ ] **Step 2: Rot prüfen**

Run: `go test ./internal/hooks/ -run 'TestEditContextOf' -count=1`
Expected (gesehen, mit dem alten Rumpf):
```
--- FAIL: TestEditContextOfReadsOnlyTheKeysAHostReads (0.00s)
    post_edit_test.go:872: the helper passed "{\"HookSpecificOutput\":{\"AdditionalContext\":\"x\"}}"
    post_edit_test.go:872: the helper passed "{\"hookSpecificOutput\":{}}"
FAIL
```

- [ ] **Step 3: Implementieren**

`editContextOf` ganz ersetzen (jetzt `"io"` importieren):

```go
// editContextOf is the additionalContext of the one JSON document stdout
// holds, walked by the exact keys a host reads: a struct decoding would match
// them regardless of case and pass an envelope no host reads.
func editContextOf(t testing.TB, stdout string) string {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var said map[string]any
	if err := decoder.Decode(&said); err != nil {
		t.Fatalf("stdout has to be one JSON document, got %q: %v", stdout, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("stdout has to be one JSON document, got more: %q", stdout)
	}
	specific, ok := said["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("stdout has no hookSpecificOutput object: %q", stdout)
	}
	context, ok := specific["additionalContext"].(string)
	if !ok {
		t.Fatalf("stdout has no hookSpecificOutput.additionalContext string: %q", stdout)
	}
	return context
}
```

In `TestPostEditNamesTheCallersOfAChangedSymbol` die Handwanderung (`var said map[string]any` … `context, _ := specific["additionalContext"].(string)`) ersetzen durch:

```go
	context := editContextOf(t, so)
```

(`TestPostEditSkipsAMissingToolOutLoud` behält seine Wanderung, weil es zusätzlich `hookEventName` liest.)

- [ ] **Step 4: Grün prüfen**

Run: `go test ./internal/hooks/ -run 'TestEditContextOf|TestPostEditNamesTheCallers|TestPostEditShares|TestPostEditSaysTheSkips' -count=1`, dann `go test ./internal/hooks/ -count=1`, `go vet ./internal/hooks/`, `gofmt -l internal/hooks`
Expected: `ok  	github.com/xidus90/loomux/internal/hooks`. Nur Testcode; keine Produktionsfunktion ändert sich, die Coverage bleibt.

- [ ] **Step 5: Mutationsprobe**

In `editContextOf` den Block `if err := decoder.Decode(new(any)); err != io.EOF { … }` entfernen → `TestEditContextOfReadsOnlyTheKeysAHostReads` rot (das zweite Dokument geht durch). `specific, ok := …; if !ok` durch `specific, _ := …` ohne Prüfung ersetzen → rot für `{"hookSpecificOutput":{}}`? Nein, der dort fehlende Schlüssel trifft die zweite Prüfung; stattdessen `context, ok := …; if !ok` entfernen (`context, _ :=`) → rot für `{"hookSpecificOutput":{}}`.

- [ ] **Step 6: Doku**

Keine; Testcode ohne Verhaltensänderung.

- [ ] **Step 7: Commit**

```
git add internal/hooks/post_edit_test.go
```
`<scratch>/commit-8.txt`:
```
test(hooks): read post-edit's context by its exact keys

editContextOf decoded stdout into a struct, and encoding/json matches
struct keys regardless of case, so an envelope with HookSpecificOutput or
AdditionalContext passed although no host reads it. The helper now walks
the decoded map by the exact keys, fails on a missing one, and requires
exactly one JSON document; a counter-test holds it to that.
```
`git commit -F <scratch>/commit-8.txt > <scratch>/gate-8.out 2>&1`.

### Task 9: Jeder Skip-Hinweis auch auf stderr

**Files:**
- Modify: `internal/verify/report.go` (neu `Skipped`; `EditReport` samt Kommentar)
- Modify: `internal/hooks/post_edit.go` (Datei-Skip in `RunPostEdit`, Kommentar dort)
- Test: `internal/verify/report_test.go`, `internal/hooks/post_edit_test.go`
- Doku: `docs/en/hooks.md:227-231`, `docs/de/hooks.md:233-237`, `docs/en/configuration.md:409-412`, `docs/de/configuration.md:417-421`, `docs/en/cli-reference.md:327`, `docs/de/cli-reference.md:347`, `README.md:83`, `README.de.md:83`

**Interfaces:**
- Consumes: `verify.BudgetSkipped(name string) string`, `EditReport(...) (red bool, notices []string)`, `skipPrefix` (Task 7); `editContextOf` (Task 8).
- Produces: `func verify.Skipped(stderr io.Writer, notices []string, notice string) []string` — schreibt `notice + "\n"` nach stderr, hängt `notice` an, schreibt nie stdout.

- [ ] **Step 1: Failing test schreiben**

`internal/verify/report_test.go`: `TestEditReportSkipsQuietlyAndBlocksOnRed` umbenennen und den ersten Block ändern (der Name wäre sonst falsch):

```go
// A skipped lane passes the edit and is said on both streams: stdout for the
// model at exit 0, stderr for a host that reads it at exit 2.
func TestEditReportSkipsOutLoudAndBlocksOnRed(t *testing.T) {
	var so, se strings.Builder
	skip := Outcome{Job: Job{Name: "lint/python"}, State: StateMissingTool, Output: `"ruff" is not on PATH: ruff check .`}
	if code := writeEdit(&so, &se, []Outcome{skip}, ""); code != 0 || se.String() != skipPrefix+`"ruff" is not on PATH: ruff check .`+"\n" {
		t.Fatalf("%d %q", code, se.String())
	}
	if !strings.Contains(so.String(), `lane skipped, \"ruff\" is not on PATH`) || !strings.Contains(so.String(), `"hookEventName":"PostToolUse"`) {
		t.Fatalf("%q", so.String())
	}
	so.Reset()
	red := Outcome{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x\n"}
	if code := writeEdit(&so, &se, []Outcome{red}, ""); code != 2 || !strings.Contains(se.String(), "vet: x") {
		t.Fatalf("%d %q", code, se.String())
	}
}
```

In `TestEditReportNamesEverySkipAndEveryRed` die stderr-Erwartung `if se.String() != "lint/go: failed\nvet: x\nlint/sql: blocked\n" {` ersetzen durch:

```go
	wantErr := "loomux hook post-tool-use: lane skipped, the edit budget ran out: test/go\n" +
		"loomux hook post-tool-use: lane skipped, run the Godot editor once to import the project\n" +
		"lint/go: failed\nvet: x\nlint/sql: blocked\n"
	if se.String() != wantErr {
```

Vor `// Stdout that is not valid JSON turns a passed hook into a hook-error notice,` einfügen:

```go
// A red lane blocks the edit and a host reads stderr alone: the lane the
// edit could not check is named there beside the finding.
func TestEditReportNamesASkippedLaneOnStderr(t *testing.T) {
	var se strings.Builder
	outs := []Outcome{
		{Job: Job{Name: "lint/python"}, State: StateFailed, Output: "E1 bad"},
		{Job: Job{Name: "types/python"}, State: StateMissingTool, Output: `"uv" is not on PATH: uv run mypy`},
	}
	red, _ := EditReport(&se, outs, "")
	if !red || !strings.Contains(se.String(), skipPrefix+`"uv" is not on PATH: uv run mypy`+"\n") || !strings.Contains(se.String(), "E1 bad") {
		t.Fatalf("%v %q", red, se.String())
	}
}

// The helper both streams go through: stderr gets the line, the notices get
// it appended, and nothing else is written.
func TestSkippedSaysTheNoticeOnStderrAndKeepsIt(t *testing.T) {
	var se strings.Builder
	notices := Skipped(&se, []string{"first"}, "second")
	if se.String() != "second\n" || len(notices) != 2 || notices[1] != "second" {
		t.Fatalf("%q %q", se.String(), notices)
	}
}
```

`internal/hooks/post_edit_test.go`, vor `// Every file of one call may have lanes to skip; they are said in one JSON` einfügen:

```go
// A red lane blocks the edit, and a host then reads only stderr: a lane the
// edit skipped beside it is named there too, not only on stdout.
func TestPostEditNamesSkippedLanesWhenRed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\nname = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seen := []string{}
	env := editEnv(t, func(s child.Spec) child.Result {
		if strings.Contains(strings.Join(s.Argv, " "), "ruff check") {
			return child.Result{Code: 1, Stdout: "a.py:1:1: F401 unused\n"}
		}
		return child.Result{}
	}, &seen)
	// ruff comes through uvx, mypy through uv: only uv is missing.
	env.Look = func(name string) (string, error) {
		if name == "uv" {
			return "", errors.New("not found")
		}
		return name, nil
	}
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(`{"tool_input":{"file_path":"a.py"}}`), &so, &se, root, env)
	if code != ExitDenied || !strings.Contains(se.String(), "F401 unused") || !strings.Contains(se.String(), `lane skipped, "uv" is not on PATH`) {
		t.Fatalf("%d %q", code, se.String())
	}
}
```

- [ ] **Step 2: Rot prüfen**

Run: `go test ./internal/hooks/ -run 'TestPostEditNamesSkippedLanesWhenRed' -count=1`
Expected (gesehen):
```
--- FAIL: TestPostEditNamesSkippedLanesWhenRed (0.01s)
    post_edit_test.go:822: 2 "lint/python: failed\na.py:1:1: F401 unused\n"
FAIL
```
Run: `go test ./internal/verify/ -run 'TestEditReport|TestSkipped' -count=1`
Expected: `report_test.go:171:13: undefined: Skipped` / `FAIL	github.com/xidus90/loomux/internal/verify [build failed]`

- [ ] **Step 3: Implementieren**

`internal/verify/report.go`: vor `EditReport` einfügen und `EditReport` ganz ersetzen:

```go
// Skipped says notice, a lane or a file post-edit did not run, on stderr and
// adds it to notices. A host reads stderr at exit 2 and the notices at exit
// 0, so a skip is heard whatever the call ends with. It never writes stdout:
// the notices go there once, when the call's code is known.
func Skipped(stderr io.Writer, notices []string, notice string) []string {
	fmt.Fprintln(stderr, notice)
	return append(notices, notice)
}

// EditReport reports the lanes of one edited file: red lanes on stderr,
// which blocks the edit, and every lane it had to skip through Skipped, in
// lane order, then whatever else the hook has to say, one notice a line. The
// aside is dropped when a lane is red: the finding matters more, and stderr
// stays the finding's. The notices of every file of a call go to WriteNotices
// together, because a host reads stdout as one document.
func EditReport(stderr io.Writer, outs []Outcome, aside string) (red bool, notices []string) {
	for _, o := range outs {
		switch {
		case Red(o.State, ScopeEdit):
			red = true
			fmt.Fprintf(stderr, "%s: %s\n", o.Job.Name, o.State)
			if o.Output != "" {
				fmt.Fprintf(stderr, "%s\n", strings.TrimSuffix(o.Output, "\n"))
			}
		case o.State == StateBudget:
			notices = Skipped(stderr, notices, BudgetSkipped(o.Job.Name))
		case o.State == StateMissingTool, o.State == StateUnready:
			notices = Skipped(stderr, notices, skipPrefix+o.Output)
		}
	}
	if aside != "" && !red {
		notices = append(notices, aside)
	}
	return red, notices
}
```

`internal/hooks/post_edit.go`, in `RunPostEdit` der Budget-Zweig der Schleife:

```go
			if fileEnv.Budget <= 0 {
				// Named like a lane the budget did not reach, on both
				// streams: which one a host reads depends on the call's code.
				notices = verify.Skipped(stderr, notices, verify.BudgetSkipped(raw))
				continue
			}
```

- [ ] **Step 4: Grün prüfen**

Run: dieselben zwei Befehle, dann `go test ./internal/verify/ ./internal/hooks/ ./internal/cli/ -count=1`, `go vet ./internal/verify/ ./internal/hooks/`
Expected: `ok` für alle drei Pakete. `Skipped`, `EditReport`, `RunPostEdit` je 100.0 %.

- [ ] **Step 5: Mutationsprobe**

In `Skipped` die Zeile `fmt.Fprintln(stderr, notice)` entfernen → `TestSkippedSaysTheNoticeOnStderrAndKeepsIt`, `TestEditReportNamesASkippedLaneOnStderr`, `TestPostEditNamesSkippedLanesWhenRed` und `TestPostEditSharesOneBudgetAcrossTheFiles` rot. Im `StateMissingTool`-Zweig von `EditReport` `Skipped(stderr, …)` durch `append(notices, skipPrefix+o.Output)` ersetzen → `TestPostEditNamesSkippedLanesWhenRed` rot.

- [ ] **Step 6: Doku**

`docs/en/hooks.md`, den Punkt, der mit „- **Skipped, not failed**: a lane whose tool is not on the `PATH`, a Godot“ beginnt (Zeile 227-231, endet mit „below writes into the same field.“), ersetzen durch:
```markdown
- **Skipped, not failed**: a lane whose tool is not on the `PATH`, a Godot
  project not yet imported, and a lane the budget (`--budget`, default 50 s)
  did not reach. A skip blocks nothing and is named on `stderr`, which is
  what a host reads when another lane is red and the hook exits 2, and, when
  the hook exits 0, in `hookSpecificOutput.additionalContext`; for a `.go`
  file the blast monitor below writes into the same field.
```
`docs/de/hooks.md`, den Punkt, der mit „- **Übersprungen, nicht rot**: eine Lane, deren Werkzeug nicht auf dem `PATH`“ beginnt (Zeile 232-237, endet mit „Blast-Monitor unten in dasselbe Feld.“), ersetzen durch:
```markdown
- **Übersprungen, nicht rot**: eine Lane, deren Werkzeug nicht auf dem `PATH`
  liegt, ein noch nicht importiertes Godot-Projekt und eine Lane, die das
  Budget (`--budget`, Vorgabe 50 s) nicht mehr erreicht. Ein Skip blockiert
  nichts und steht auf `stderr`, das ein Host liest, wenn eine andere Lane
  rot ist und der Hook mit 2 endet, und, wenn der Hook mit 0 endet, in
  `hookSpecificOutput.additionalContext`; bei einer `.go`-Datei schreibt der
  Blast-Monitor unten in dasselbe Feld.
```
`docs/en/configuration.md`, den Punkt, der mit „- **Verdict of the post-edit hook:** a red lane exits 2 with its output on“ beginnt (Zeile 411-414, endet mit „lanes and exits 0.“), ersetzen durch:
```markdown
- **Verdict of the post-edit hook:** a red lane exits 2 with its output on
  `stderr`. A lane it skipped blocks nothing and is named on `stderr` as well,
  and at exit 0 in `hookSpecificOutput.additionalContext` on `stdout`. A file
  whose ending no active stack claims gets no lanes and exits 0.
```
`docs/de/configuration.md`, den Punkt, der mit „- **Urteil des post-edit-Hooks:** Eine rote Lane endet mit Exit 2 und ihrer“ beginnt (Zeile 419-423, endet mit „mit 0.“), ersetzen durch:
```markdown
- **Urteil des post-edit-Hooks:** Eine rote Lane endet mit Exit 2 und ihrer
  Ausgabe auf `stderr`. Eine übersprungene Lane blockiert nichts und steht
  ebenfalls auf `stderr`, bei Exit 0 außerdem in
  `hookSpecificOutput.additionalContext` auf `stdout`. Eine Datei, deren
  Endung kein aktiver Stack beansprucht, bekommt keine Lanes und endet mit 0.
```
`docs/en/cli-reference.md`, die Zeile, die mit „- **Skipped lanes**: a lane whose tool is not on the `PATH`“ beginnt (Zeile 328), ersetzen durch:
```markdown
- **Skipped lanes**: a lane whose tool is not on the `PATH`, a Godot project not yet imported, and every lane the budget did not reach are skipped, not failed. Each is named on `stderr`, and at exit 0 on `stdout` as `{"hookSpecificOutput":{"additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go","hookEventName":"PostToolUse"}}`. A file of a call the budget did not reach is named the same way.
```
`docs/de/cli-reference.md`, die Zeile, die mit „- **Übersprungene Lanes**: Eine Lane, deren Werkzeug nicht auf dem `PATH` liegt“ beginnt (Zeile 348), ersetzen durch:
```markdown
- **Übersprungene Lanes**: Eine Lane, deren Werkzeug nicht auf dem `PATH` liegt, ein noch nicht importiertes Godot-Projekt und jede Lane, die das Budget nicht mehr erreicht, werden übersprungen, nicht rot. Jede steht auf `stderr`, und bei Exit 0 auf `stdout` als `{"hookSpecificOutput":{"additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go","hookEventName":"PostToolUse"}}`. Eine Datei eines Aufrufs, die das Budget nicht mehr erreicht, steht dort ebenso.
```
`README.md:83`: `        Verify-->>Agent: Exit 2 with the finding` → `        Verify-->>Agent: Exit 2 with the finding and the skipped lanes`
`README.de.md:83`: `        Verify-->>Agent: Exit 2 mit dem Befund` → `        Verify-->>Agent: Exit 2 mit dem Befund und den übersprungenen Lanes`
Die Kommentare in `report.go` und `post_edit.go` stehen oben im Code.

- [ ] **Step 7: Commit**

```
git add internal/verify/report.go internal/verify/report_test.go internal/hooks/post_edit.go internal/hooks/post_edit_test.go docs/en/hooks.md docs/de/hooks.md docs/en/configuration.md docs/de/configuration.md docs/en/cli-reference.md docs/de/cli-reference.md README.md README.de.md
```
`<scratch>/commit-9.txt`:
```
fix(hooks): name every skipped post-edit lane on stderr as well

A lane skipped for a missing tool, an unready project or a spent budget
was named only in the context on stdout, which a host reads at exit 0.
When another lane was red the hook exited 2, the host read stderr alone,
and the skip went unheard. verify.Skipped writes each skip to stderr and
keeps it for the context; the edit report uses it for lanes and post-edit
for a file the shared budget did not reach, so both are said the same way.
```
`git commit -F <scratch>/commit-9.txt > <scratch>/gate-9.out 2>&1`.

### Task 10: post-edit antwortet über die hosts-Naht, kein stdout bei Exit ≠ 0

**Files:**
- Modify: `internal/hooks/post_edit.go:27-56` (`EditEnv`, `PostToolUse`), `RunPostEdit`
- Modify: `internal/verify/report.go` (`WriteNotices` entfällt, Import `encoding/json`, Kommentar `EditReport`)
- Modify: `internal/cli/hook.go:147`
- Modify: `internal/hosts/answer.go:38-41`, `internal/hosts/claude.go:67-69`, `internal/hosts/hostio.go:108`
- Test: `internal/hooks/post_edit_test.go`, `internal/verify/report_test.go`, `internal/cli/hook_test.go`, `internal/hosts/answer_test.go`
- Doku: `docs/en|de/cli-reference.md` (Skipped lanes, Blast monitor, Exit Codes), `docs/en|de/hooks.md` (Skipped-Punkt, Blast-Monitor-Einleitung, „A red lane comes first“)

**Interfaces:**
- Consumes: `verify.BudgetSkipped`, `verify.Skipped`, `EditReport(...) (bool, []string)` (Tasks 7, 9); `editContextOf(t testing.TB, stdout string) string` (Task 8); `hosts.ParseHost`, `hosts.WriteContext(host Host, event string, w io.Writer, lines []string) error`, `hosts.HostClaude|HostCodex` (bestehend).
- Produces:
  - `func hooks.PostToolUse(stdin io.Reader, stdout, stderr io.Writer, root, hostName string, budget time.Duration) int`
  - `EditEnv.Host hosts.Host` (erstes Feld)
  - `cli.postToolUse` hat die Signatur von `hooks.PostToolUse`; `runHook` reicht `host` durch.
  - `verify.WriteNotices` gibt es nicht mehr.

Entscheidung zu Codex: `WriteContext` wird bei Code 0 gerufen, sobald der Payload eine Datei nennt; eine Datei mit ignorierter Endung (`data.json`) zählt dabei als genannt, also endet `--host codex` auch dort mit 1. Nur ein Payload ohne Datei (oder ein fehlgeschlagener Aufruf) endet vor jedem Schreiben mit 0.

- [ ] **Step 1: Failing test schreiben**

**1a — Verhaltenstests, die gegen den Stand nach Task 9 kompilieren.** In `post_edit_test.go` vor `// Every file of one call may have lanes to skip; they are said in one JSON` einfügen:

```go
// Notices reach stdout through the host's adapter, Claude's encoder, which
// names the event first and leaves markup as it is. The second file is out of
// budget and never read, so its name may hold what no file name on Windows may.
func TestPostEditWritesTheContextThroughTheHostAdapter(t *testing.T) {
	root := goProject(t)
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	odd := filepath.ToSlash(filepath.Join(root, "<b>.go"))
	args, _ := json.Marshal(map[string]string{"TargetFile": filepath.ToSlash(filepath.Join(root, "a.go")), "target_file": odd})
	payload := `{"conversationId":"c1","toolCall":{"name":"write_to_file","args":` + string(args) + `}}`
	now := time.Now()
	env := editEnv(t, func(child.Spec) child.Result {
		now = now.Add(DefaultBudget)
		return child.Result{}
	}, &[]string{})
	env.Now = func() time.Time { return now }
	var so, se bytes.Buffer
	if code := RunPostEdit(strings.NewReader(payload), &so, &se, root, env); code != ExitOK {
		t.Fatalf("%d %q", code, se.String())
	}
	if !strings.HasPrefix(so.String(), `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":`) || !strings.Contains(so.String(), "<b>.go") {
		t.Fatalf("%q", so.String())
	}
	if got := editContextOf(t, so.String()); !strings.HasSuffix(got, verify.BudgetSkipped(odd)) {
		t.Fatalf("%q", got)
	}
}

// A host reads stdout only at exit 0. A red call writes nothing there; the
// file the budget did not reach is named on stderr beside the finding.
func TestPostEditWritesNoStdoutWhenRed(t *testing.T) {
	root := goProject(t)
	now := time.Now()
	env := editEnv(t, func(child.Spec) child.Result {
		now = now.Add(DefaultBudget)
		return child.Result{Code: 1, Stdout: "vet: bad\n"}
	}, &[]string{})
	env.Now = func() time.Time { return now }
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(agyCall(t, root, "a.go", "b.go")), &so, &se, root, env)
	if code != ExitDenied || so.Len() != 0 {
		t.Fatalf("%d %q", code, so.String())
	}
	if !strings.Contains(se.String(), "vet: bad") || !strings.Contains(se.String(), verify.BudgetSkipped(filepath.ToSlash(filepath.Join(root, "b.go")))) {
		t.Fatalf("%q", se.String())
	}
}

// The aside of a green file goes when another file of the same call is red:
// the call's finding matters more than who calls the green one.
func TestPostEditDropsEveryAsideWhenAnyFileIsRed(t *testing.T) {
	root := graphRepo(t)
	writeRepoFile(t, root, "calc/calc.go", calcHead+strings.Replace(addSource, "a + b", "b + a", 1)+"\n"+subSource)
	args, _ := json.Marshal(map[string]string{
		"TargetFile":  filepath.ToSlash(filepath.Join(root, "calc", "calc.go")),
		"target_file": filepath.ToSlash(filepath.Join(root, "main.go")),
	})
	payload := `{"conversationId":"c1","toolCall":{"name":"write_to_file","args":` + string(args) + `}}`
	code, so, se, _ := postEdit(t, root, payload, func(s child.Spec) child.Result {
		if strings.Contains(strings.Join(s.Argv, " "), "gofmt main.go") {
			return child.Result{Code: 1, Stdout: "main.go\n"}
		}
		return child.Result{}
	})
	if code != ExitDenied || so != "" {
		t.Fatalf("%d %q %q", code, so, se)
	}
}
```

**1b — Host-Tests.** Import `"github.com/xidus90/loomux/internal/hosts"` in `post_edit_test.go` (zwischen `detect` und `verify`, die Leerzeile davor entfällt). In `editEnv` und im `EditEnv{…}`-Literal von `TestPostEditSharesOneBudgetAcrossTheFiles` als erstes Feld `Host: hosts.HostClaude,` setzen — ohne das endet jeder Test an `WriteContext` mit `unknown host ""` und 1. Die vier Aufrufe `PostToolUse(…, root, DefaultBudget)` / `PostToolUse(…, t.TempDir(), DefaultBudget)` (Zeile ~347, ~363, ~500, ~528) bekommen `"claude"` vor `DefaultBudget`. Hinter den 1a-Tests einfügen:

```go
// Codex has no adapter yet: a call that checked a file ends with the seam's
// refusal instead of an answer in another host's shape, and a call naming no
// file ends before anything is written, as it does for every host.
func TestPostEditAnswersCodexWithItsMissingAdapter(t *testing.T) {
	root := goProject(t)
	env := editEnv(t, passing, &[]string{})
	env.Host = hosts.HostCodex
	var so, se bytes.Buffer
	if code := RunPostEdit(strings.NewReader(`{"tool_input":{"file_path":"a.go"}}`), &so, &se, root, env); code != ExitInternal || so.Len() != 0 || !strings.Contains(se.String(), "no adapter for this host") {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
	so.Reset()
	se.Reset()
	// A file whose ending the presets ignore is named all the same.
	if code := RunPostEdit(strings.NewReader(`{"tool_input":{"file_path":"data.json"}}`), &so, &se, root, env); code != ExitInternal || so.Len() != 0 {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
	so.Reset()
	se.Reset()
	if code := RunPostEdit(strings.NewReader(`{"conversationId":"c1"}`), &so, &se, root, env); code != ExitOK || so.Len() != 0 || se.Len() != 0 {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
}

// A host nobody knows is refused before any lane runs: stderr carries that
// refusal alone, and answering in the wrong shape is worse than not answering.
func TestPostToolUseRefusesAnUnknownHost(t *testing.T) {
	var so, se bytes.Buffer
	if code := PostToolUse(strings.NewReader(`{"tool_input":{"file_path":"data.json"}}`), &so, &se, t.TempDir(), "vim", DefaultBudget); code != ExitInternal || so.Len() != 0 || se.String() != "loomux hook post-tool-use: unknown host \"vim\": expected claude, antigravity or codex\n" {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
}
```

`internal/cli/hook_test.go`: die zwei Stubs bekommen den Host-Parameter —
`postToolUse = func(_ io.Reader, _, _ io.Writer, _, _ string, budget time.Duration) int {` (in `TestHookPostToolUsePassesTheBudgetOn`) und
`postToolUse = func(io.Reader, io.Writer, io.Writer, string, string, time.Duration) int { called = true; return 2 }` (in `TestHookEndsAtOnceWhenTheHooksModuleIsOff`). Vor `// A budget that is no duration is a malformed call, and a malformed` einfügen:

```go
// --host reaches post-edit as given, so its answer takes that host's shape.
func TestHookPostToolUsePassesTheHostOn(t *testing.T) {
	saved := postToolUse
	t.Cleanup(func() { postToolUse = saved })
	var got string
	postToolUse = func(_ io.Reader, _, _ io.Writer, _, host string, _ time.Duration) int {
		got = host
		return 0
	}
	if code, _, errOut := runWith(`{}`, "hook", "post-tool-use", "--host", "antigravity", "--root", t.TempDir()); code != 0 || got != "antigravity" {
		t.Fatalf("code %d, host %q, err %q", code, got, errOut)
	}
}
```

`internal/verify/report_test.go`: ab `// writeEdit reports one file's lanes` bis Dateiende ersetzen (der Umschlag ist nicht mehr Sache von `verify`; die Tests prüfen die Hinweise als Liste; `editContext`, `writeEdit` und `TestWriteNotices` entfallen). Importblock wird `"errors"`, `"slices"`, `"strings"`, `"testing"`, `"time"` (ohne `encoding/json`, `io`):

```go
// The budget's notice is the one cli-reference.md shows, for a lane and for
// a file alike.
func TestBudgetSkippedIsTheDocumentedSentence(t *testing.T) {
	want := "loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go"
	if got := BudgetSkipped("lint/go"); got != want {
		t.Fatalf("%q", got)
	}
}

// A skipped lane passes the edit and is said twice: as a notice for the
// model at exit 0, and on stderr for a host that reads it at exit 2.
func TestEditReportSkipsOutLoudAndBlocksOnRed(t *testing.T) {
	var se strings.Builder
	skip := Outcome{Job: Job{Name: "lint/python"}, State: StateMissingTool, Output: `"ruff" is not on PATH: ruff check .`}
	said := skipPrefix + `"ruff" is not on PATH: ruff check .`
	red, notices := EditReport(&se, []Outcome{skip}, "")
	if red || se.String() != said+"\n" || !slices.Equal(notices, []string{said}) {
		t.Fatalf("%v %q %q", red, se.String(), notices)
	}
	se.Reset()
	failed := Outcome{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x\n"}
	if red, _ := EditReport(&se, []Outcome{failed}, ""); !red || !strings.Contains(se.String(), "vet: x") {
		t.Fatalf("%v %q", red, se.String())
	}
}

func TestEditReportNamesEverySkipAndEveryRed(t *testing.T) {
	var se strings.Builder
	outs := []Outcome{
		{Job: Job{Name: "types/go"}, State: StateOK, Output: "fine\n"},
		{Job: Job{Name: "test/go"}, State: StateBudget, Output: "part"},
		{Job: Job{Name: "test/gdscript"}, State: StateUnready, Output: "run the Godot editor once to import the project"},
		{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x"},
		{Job: Job{Name: "lint/sql"}, State: StateBlocked, BlockedBy: "x"},
	}
	red, notices := EditReport(&se, outs, "")
	if !red {
		t.Fatal("red lanes block the edit")
	}
	wantErr := "loomux hook post-tool-use: lane skipped, the edit budget ran out: test/go\n" +
		"loomux hook post-tool-use: lane skipped, run the Godot editor once to import the project\n" +
		"lint/go: failed\nvet: x\nlint/sql: blocked\n"
	if se.String() != wantErr {
		t.Fatalf("%q", se.String())
	}
	want := []string{
		"loomux hook post-tool-use: lane skipped, the edit budget ran out: test/go",
		"loomux hook post-tool-use: lane skipped, run the Godot editor once to import the project",
	}
	if !slices.Equal(notices, want) {
		t.Fatalf("%q", notices)
	}
}

// A red lane blocks the edit and a host reads stderr alone: the lane the
// edit could not check is named there beside the finding.
func TestEditReportNamesASkippedLaneOnStderr(t *testing.T) {
	var se strings.Builder
	outs := []Outcome{
		{Job: Job{Name: "lint/python"}, State: StateFailed, Output: "E1 bad"},
		{Job: Job{Name: "types/python"}, State: StateMissingTool, Output: `"uv" is not on PATH: uv run mypy`},
	}
	red, _ := EditReport(&se, outs, "")
	if !red || !strings.Contains(se.String(), skipPrefix+`"uv" is not on PATH: uv run mypy`+"\n") || !strings.Contains(se.String(), "E1 bad") {
		t.Fatalf("%v %q", red, se.String())
	}
}

// The helper both streams go through: stderr gets the line, the notices get
// it appended, and nothing else is written.
func TestSkippedSaysTheNoticeOnStderrAndKeepsIt(t *testing.T) {
	var se strings.Builder
	notices := Skipped(&se, []string{"first"}, "second")
	if se.String() != "second\n" || len(notices) != 2 || notices[1] != "second" {
		t.Fatalf("%q %q", se.String(), notices)
	}
}

// A run with nothing skipped and nothing to add has nothing to say.
func TestEditReportGreenIsSilent(t *testing.T) {
	var se strings.Builder
	if red, notices := EditReport(&se, []Outcome{{Job: Job{Name: "lint/go"}, State: StateOK}}, ""); red || notices != nil || se.Len() != 0 {
		t.Fatalf("%v %q %q", red, notices, se.String())
	}
}

func TestEditReportCarriesTheAsideOnAGreenRun(t *testing.T) {
	var se strings.Builder
	aside := "[graph] a.go: changed F; callers in other files:\n  G (b.go)"
	red, notices := EditReport(&se, []Outcome{{Job: Job{Name: "lint/go"}, State: StateOK}}, aside)
	if red || se.Len() != 0 || !slices.Equal(notices, []string{aside}) {
		t.Fatalf("%v %q %q", red, se.String(), notices)
	}
}

func TestEditReportPutsTheAsideAfterTheSkips(t *testing.T) {
	var se strings.Builder
	skip := Outcome{Job: Job{Name: "test/go"}, State: StateBudget}
	red, notices := EditReport(&se, []Outcome{skip}, "[graph] x")
	want := []string{"loomux hook post-tool-use: lane skipped, the edit budget ran out: test/go", "[graph] x"}
	if red || !slices.Equal(notices, want) {
		t.Fatalf("%v %q", red, notices)
	}
}

// A red lane matters more than who calls the edited code; the aside goes,
// the skips of another lane stay as they were.
func TestEditReportDropsTheAsideOnRed(t *testing.T) {
	var se strings.Builder
	failed := Outcome{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x"}
	if red, notices := EditReport(&se, []Outcome{failed}, "[graph] x"); !red || notices != nil {
		t.Fatalf("%v %q", red, notices)
	}
	skip := Outcome{Job: Job{Name: "test/go"}, State: StateBudget}
	red, notices := EditReport(&se, []Outcome{failed, skip}, "[graph] x")
	if !red || !slices.Equal(notices, []string{BudgetSkipped("test/go")}) {
		t.Fatalf("%v %q", red, notices)
	}
}
```

`internal/hosts/answer_test.go` (nur Testeingaben, weil post-edit für agy jetzt `injectSteps` schreibt): in `TestAnswerKeepsAntigravitysRedEdit` Kommentarzeile 2 `// warning, and whatever stdout came beside it is dropped.` und Eingabe `` `{"injectSteps":[]}` `` statt `` `{"hookSpecificOutput":{}}` ``; in `TestAnswerDropsAntigravitysPostEditNotices` Kommentar
```go
// A passed or unjudged edit on Antigravity ends with 0 and drops the
// notices post-edit wrote as injectSteps; the empty stdout left is an answer
// agy 1.2.11 takes without a hook error.
```
und Eingabe `` `{"injectSteps":[{"ephemeralMessage":"x"}]}` ``.

- [ ] **Step 2: Rot prüfen**

Run (nur 1a, gegen den Stand nach Task 9): `go test ./internal/hooks/ -run 'TestPostEditWritesTheContextThroughTheHostAdapter|TestPostEditWritesNoStdoutWhenRed|TestPostEditDropsEveryAsideWhenAnyFileIsRed' -count=1`
Expected (gesehen):
```
--- FAIL: TestPostEditWritesTheContextThroughTheHostAdapter (0.01s)
    post_edit_test.go:848: "{\"hookSpecificOutput\":{\"additionalContext\":\"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go\\nloomux hook post-tool-use: lane skipped, the edit budget ran out: C:/…/001/\\u003cb\\u003e.go\",\"hookEventName\":\"PostToolUse\"}}\n"
--- FAIL: TestPostEditWritesNoStdoutWhenRed (0.01s)
    post_edit_test.go:868: 2 "{\"hookSpecificOutput\":{\"additionalContext\":\"loomux hook post-tool-use: lane skipped, the edit budget ran out: C:/…/001/b.go\",\"hookEventName\":\"PostToolUse\"}}\n"
--- FAIL: TestPostEditDropsEveryAsideWhenAnyFileIsRed (0.02s)
    post_edit_test.go:892: 2 "{\"hookSpecificOutput\":{\"additionalContext\":\"[graph] calc/calc.go: changed Add; callers in other files:\\n  TestAdd (calc/calc_test.go)\\n  main (main.go)\",\"hookEventName\":\"PostToolUse\"}}\n" "lint/go: failed\n…"
FAIL
```
Run (mit 1b): `go test ./internal/hooks/ ./internal/cli/ -run 'TestPostEditAnswersCodex|TestPostToolUseRefusesAnUnknownHost|TestHookPostToolUsePassesTheHostOn' -count=1`
Expected: `post_edit_test.go:29:3: unknown field Host in struct literal of type EditEnv`, `too many arguments in call to PostToolUse`, `FAIL	github.com/xidus90/loomux/internal/hooks [build failed]`.

- [ ] **Step 3: Implementieren**

`internal/hooks/post_edit.go`: Import `"github.com/xidus90/loomux/internal/hosts"` zwischen `detect` und `verify`. `EditEnv` und `PostToolUse` ganz:

```go
// EditEnv is what a post-edit run needs from outside: the host it answers,
// what starts a tool and finds it on the PATH, which binary {loomux} names,
// the budget, the clock, and whether Godot has imported a project. A nil
// ImportReady asks the disk.
type EditEnv struct {
	Host        hosts.Host
	Start       func(child.Spec) child.Result
	Look        func(string) (string, error)
	Loomux      string
	Budget      time.Duration
	Now         func() time.Time
	ImportReady func(dir string) bool
}
```

```go
// PostToolUse checks the files an edit touched, with the real tools, and
// answers the host hostName names in that host's shape.
func PostToolUse(stdin io.Reader, stdout, stderr io.Writer, root, hostName string, budget time.Duration) int {
	host, err := hosts.ParseHost(hostName)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: %v\n", err)
		return ExitInternal
	}
	loomux, err := editExecutable()
	if err != nil {
		loomux = "loomux"
	}
	return RunPostEdit(stdin, stdout, stderr, root, EditEnv{
		Host: host, Start: child.Run, Look: exec.LookPath, Loomux: loomux, Budget: budget, Now: time.Now, ImportReady: verify.ImportReady,
	})
}
```

`RunPostEdit` ganz:

```go
// RunPostEdit runs the lanes of the `edit` profile for the edited file's
// stack, as [verify] and the presets lay them out, or the wiki lint for a
// wiki page. A red lane blocks the edit with 2; a config it cannot read ends
// with 1, which shows the error and blocks nothing. A call that names several
// files checks each, within one budget for them all, and ends with the worst
// of their codes. What it has to tell the model goes out once, through the
// host's adapter, and only when that code is 0.
func RunPostEdit(stdin io.Reader, stdout, stderr io.Writer, root string, env EditEnv) int {
	files := editedFiles(stdin)
	if len(files) == 0 {
		return ExitOK
	}
	// Tools run in their area, so every path handed to them must be absolute.
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	facts := detect.Detect(os.DirFS(root))
	eff, err := editLoad(root, facts)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: %v\n", err)
		return ExitInternal
	}
	// One budget for the call: a host's timeout is per hook, not per file.
	// The first file runs as a single edit always has; each later one gets
	// what is left, and none once that is spent.
	start := env.Now()
	runID := verify.NewRunID(start, os.Getpid())
	code := ExitOK
	var notices []string
	for i, raw := range files {
		fileEnv, id := env, runID
		// A budget of 0 is none, and stays none for every file.
		if i > 0 && env.Budget > 0 {
			fileEnv.Budget = start.Add(env.Budget).Sub(env.Now())
			if fileEnv.Budget <= 0 {
				// Named like a lane the budget did not reach, on both
				// streams: which one a host reads depends on the call's code.
				notices = verify.Skipped(stderr, notices, verify.BudgetSkipped(raw))
				continue
			}
			// Coverage files of their own; CleanCover matches `<runID>-`.
			id += "." + strconv.Itoa(i)
		}
		fileCode, said := checkEdit(stderr, root, raw, id, eff, facts, fileEnv)
		code = max(code, fileCode)
		notices = append(notices, said...)
	}
	// A host reads stdout only at exit 0. At any other code it reads stderr,
	// which already holds every finding and every skip, so an aside of a
	// green file in a red call goes with the rest.
	if code != ExitOK {
		return code
	}
	if err := hosts.WriteContext(env.Host, "PostToolUse", stdout, notices); err != nil {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}
```

`internal/verify/report.go`: `WriteNotices` samt Kommentar löschen, `"encoding/json"` aus dem Import nehmen; der Kommentar über `EditReport` wird:

```go
// EditReport reports the lanes of one edited file: red lanes on stderr,
// which blocks the edit, and every lane it had to skip through Skipped, in
// lane order, then whatever else the hook has to say, one notice a line. The
// aside is dropped when a lane is red: the finding matters more, and stderr
// stays the finding's. The caller hands the notices of every file of a call
// to the host's adapter together, because a host reads stdout as one
// document, and only when the call ends with 0.
```

`internal/cli/hook.go:147`: `return postToolUse(stdin, stdout, stderr, resolved, host, budget)`.

`internal/hosts/answer.go`, Kommentar im Zweig `event == "post-tool-use"`:

```go
		// What post-edit writes on stdout is the host's context, for agy
		// injectSteps, and whether agy reads them after a PostToolUse is
		// unmeasured. Its guide names {} as a PostToolUse's answer, and an
		// empty stdout passes as well (measured with agy 1.2.11 on
		// 2026-09-25, no hook error in its log).
```

`internal/hosts/claude.go`, erste drei Zeilen des Kommentars über `writeClaudeContext`:

```go
// writeClaudeContext puts lines where the model will read them, as the
// answer to event: it serves any event the caller names, today SessionStart
// and PostToolUse.
```

`internal/hosts/hostio.go`, erste Zeile des Kommentars über `WriteContext` wird:

```go
// WriteContext hands lines back for the model to read. session-start and
// post-tool-use answer through it, post-tool-use only at exit 0.
```

`internal/hosts/hostio.go`, der zweite Absatz des Paketkommentars (Zeilen 3–5) wird:

```go
// One normalised payload in, one normalised answer out. The core packages see
// these two types and never a JSON envelope, so a second host costs an adapter
// and not a second copy of the hook. The tool events take their call another
// way in: pre-tool-use and post-tool-use read it through guard.Call, which
// knows each host's shape. Their answer goes out here like every other,
// post-tool-use's context through WriteContext.
```

- [ ] **Step 4: Grün prüfen**

Run: beide Befehle aus Step 2, dann `go test ./internal/verify/ ./internal/hooks/ ./internal/cli/ ./internal/hosts/ -count=1`, `go vet ./internal/verify/ ./internal/hooks/ ./internal/cli/ ./internal/hosts/`, `gofmt -l internal/`
Expected: `ok` für alle vier Pakete, vet und gofmt still. `PostToolUse`, `RunPostEdit`, `checkEdit`, `EditReport`, `Skipped`, `BudgetSkipped`, `runHook`, `WriteContext`, `Answer` je 100.0 %. Dann `grep -rn "WriteNotices\|SkipPrefix" internal docs/en docs/de` → keine Treffer (`docs/.superpowers/` nennt beide Namen in diesem Plan und seiner Spec und bleibt außen vor).

- [ ] **Step 5: Mutationsprobe**

In `RunPostEdit` den Block `if code != ExitOK { return code }` entfernen → `TestPostEditWritesNoStdoutWhenRed` und `TestPostEditDropsEveryAsideWhenAnyFileIsRed` rot. `return ExitInternal` im `WriteContext`-Fehlerzweig durch `return ExitOK` ersetzen → `TestPostEditAnswersCodexWithItsMissingAdapter` rot. In `internal/cli/hook.go` `host` durch `"claude"` ersetzen → `TestHookPostToolUsePassesTheHostOn` rot. In `PostToolUse` `return ExitInternal` nach `ParseHost` entfernen → `TestPostToolUseRefusesAnUnknownHost` rot.

- [ ] **Step 6: Doku**

`docs/en/cli-reference.md`, die in Task 9 geschriebene Zeile, die mit „- **Skipped lanes**: a lane whose tool is not on the `PATH`“ beginnt (Zeile 328), ersetzen durch:
```markdown
- **Skipped lanes**: a lane whose tool is not on the `PATH`, a Godot project not yet imported, and every lane the budget did not reach are skipped, not failed. Each is named on `stderr`, and at exit 0 on `stdout` in the host's shape, for Claude Code as `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go"}}`, with `<`, `>` and `&` left as they are. A file of a call the budget did not reach is named the same way. At any other exit code nothing is written to `stdout`.
```
Die Zeile, die mit „- **Blast monitor**: after a `.go` edit with no red lane,“ beginnt (Zeile 329), ersetzen durch:
```markdown
- **Blast monitor**: after a `.go` edit, when no lane of the call is red, the direct callers in other files of every symbol the edit changed or removed, measured against the graph on disk, follow the skipped lanes in the same `additionalContext`. Silent without a graph and never a finding; see [Hooks](hooks.md#the-blast-monitor).
```
Die Zeile, die mit „- **Exit Codes**: `0` (all lanes passed, skipped, or nothing to run), `1` (malformed call, such as a missing `--host`, or a `[verify]` that cannot be loaded)“ beginnt (Zeile 330), ersetzen durch:
```markdown
- **Exit Codes**: `0` (all lanes passed, skipped, or nothing to run), `1` (malformed call, such as a missing or unknown `--host`, a `[verify]` that cannot be loaded, or `--host codex` once the call names a file: Codex has no adapter yet, so the hook refuses rather than answer in another host's shape), `2` (a lane failed, timed out or is blocked; its output on `stderr`).
```
`docs/de/cli-reference.md`, die in Task 9 geschriebene Zeile, die mit „- **Übersprungene Lanes**: Eine Lane, deren Werkzeug nicht auf dem `PATH` liegt“ beginnt (Zeile 348), ersetzen durch:
```markdown
- **Übersprungene Lanes**: Eine Lane, deren Werkzeug nicht auf dem `PATH` liegt, ein noch nicht importiertes Godot-Projekt und jede Lane, die das Budget nicht mehr erreicht, werden übersprungen, nicht rot. Jede steht auf `stderr`, und bei Exit 0 auf `stdout` in der Form des Hosts, für Claude Code als `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go"}}`, `<`, `>` und `&` unverändert. Eine Datei eines Aufrufs, die das Budget nicht mehr erreicht, steht dort ebenso. Bei jedem anderen Exit-Code schreibt der Hook nichts auf `stdout`.
```
Die Zeile, die mit „- **Blast-Monitor**: Nach einem Edit an einer `.go`-Datei ohne rote Lane“ beginnt (Zeile 349), ersetzen durch:
```markdown
- **Blast-Monitor**: Nach einem Edit an einer `.go`-Datei, wenn keine Lane des Aufrufs rot ist, folgen den übersprungenen Lanes im selben `additionalContext` die direkten Aufrufer in anderen Dateien jedes Symbols, das der Edit gegenüber dem Graphen auf der Platte geändert oder entfernt hat. Ohne Graph schweigt er, und ein Befund ist er nie; siehe [Hooks](hooks.md#der-blast-monitor).
```
Die Zeile, die mit „- **Exit-Codes**: `0` (alle Lanes grün, übersprungen oder nichts zu fahren), `1` (fehlerhafter Aufruf, etwa ein fehlendes `--host`,“ beginnt (Zeile 350), ersetzen durch:
```markdown
- **Exit-Codes**: `0` (alle Lanes grün, übersprungen oder nichts zu fahren), `1` (fehlerhafter Aufruf, etwa ein fehlendes oder unbekanntes `--host`, ein `[verify]`, das sich nicht laden lässt, oder `--host codex`, sobald der Aufruf eine Datei nennt: Codex hat noch keinen Adapter, der Hook verweigert also, statt in der Form eines anderen Hosts zu antworten), `2` (eine Lane ist gescheitert, abgelaufen oder blockiert; ihre Ausgabe auf `stderr`).
```
`docs/en/hooks.md`, den in Task 9 geschriebenen Punkt, der mit „- **Skipped, not failed**: a lane whose tool is not on the `PATH`, a Godot“ beginnt (nach Task 9 Zeile 227-232, endet mit „file the blast monitor below writes into the same field.“), ersetzen durch:
```markdown
- **Skipped, not failed**: a lane whose tool is not on the `PATH`, a Godot
  project not yet imported, and a lane the budget (`--budget`, default 50 s)
  did not reach. A skip blocks nothing and is named on `stderr`, which is
  what a host reads when another lane is red and the hook exits 2, and, when
  the hook exits 0, in the host's context: `hookSpecificOutput.additionalContext`
  for Claude Code; for a `.go` file the blast monitor below writes into the
  same field. At any other exit code the hook writes nothing to `stdout`.
```
Den Einleitungssatz unter `### The blast monitor`, der mit „After the lanes of a `.go` edit, and only when none of them is red, the hook“ beginnt und mit „compares each symbol's body hash with the graph's nodes of the same path:“ endet (nach Task 9 Zeile 240-243), ersetzen durch:
```markdown
After the lanes of a `.go` edit, and only when no lane of the call is red,
the hook tells the model who calls what the edit just changed. It reads the
graph on disk (`.loomux/state/graph/wiring.json`), extracts the edited file
again and compares each symbol's body hash with the graph's nodes of the
same path:
```
Den Punkt, der mit „- **A red lane comes first.** When a lane is red the hook exits 2 with the“ beginnt und mit „and `stdout` stays valid JSON or empty.“ endet (nach Task 9 Zeile 272-274), ersetzen durch:
```markdown
- **A red lane comes first.** When a lane of the call is red, in any of the
  files it names, the hook exits 2 with the finding on `stderr` and writes
  nothing to `stdout`, the blast context of a green file included: the
  finding matters more, and a host reads only `stderr` at exit 2.
```
`docs/de/hooks.md`, den in Task 9 geschriebenen Punkt, der mit „- **Übersprungen, nicht rot**: eine Lane, deren Werkzeug nicht auf dem `PATH`“ beginnt (nach Task 9 Zeile 232-238, endet mit „Blast-Monitor unten in dasselbe Feld.“), ersetzen durch:
```markdown
- **Übersprungen, nicht rot**: eine Lane, deren Werkzeug nicht auf dem `PATH`
  liegt, ein noch nicht importiertes Godot-Projekt und eine Lane, die das
  Budget (`--budget`, Vorgabe 50 s) nicht mehr erreicht. Ein Skip blockiert
  nichts und steht auf `stderr`, das ein Host liest, wenn eine andere Lane
  rot ist und der Hook mit 2 endet, und, wenn der Hook mit 0 endet, im
  Kontext des Hosts: `hookSpecificOutput.additionalContext` für Claude Code;
  bei einer `.go`-Datei schreibt der Blast-Monitor unten in dasselbe Feld. Bei
  jedem anderen Exit-Code schreibt der Hook nichts auf `stdout`.
```
Die Einleitung unter `### Der Blast-Monitor`, die mit „Nach den Lanes eines Edits an einer `.go`-Datei, und nur wenn keine davon rot“ beginnt und mit „Symbols mit den Knoten desselben Pfads im Graphen:“ endet (nach Task 9 Zeile 247-251), ersetzen durch:
```markdown
Nach den Lanes eines Edits an einer `.go`-Datei, und nur wenn keine Lane des
Aufrufs rot ist, sagt der Hook dem Modell, wer aufruft, was der Edit gerade
geändert hat. Er liest den Graphen auf der Platte
(`.loomux/state/graph/wiring.json`), extrahiert die bearbeitete Datei neu und
vergleicht den Rumpf-Hash jedes Symbols mit den Knoten desselben Pfads im
Graphen:
```
Den Punkt, der mit „- **Eine rote Lane geht vor.** Ist eine Lane rot, endet der Hook mit Exit 2“ beginnt und mit „ist wichtiger, und `stdout` bleibt gültiges JSON oder leer.“ endet (nach Task 9 Zeile 283-285), ersetzen durch:
```markdown
- **Eine rote Lane geht vor.** Ist eine Lane des Aufrufs rot, in welcher
  seiner Dateien auch immer, endet der Hook mit Exit 2 und dem Befund auf
  `stderr` und schreibt nichts auf `stdout`, auch nicht den Blast-Kontext
  einer grünen Datei: der Befund ist wichtiger, und ein Host liest bei Exit 2
  nur `stderr`.
```
`docs/en|de/configuration.md` (Urteil aus Task 9) bleibt: es nennt die Claude-Form, die sich nicht ändert.

**Changelog-Zeile für den PR-Rumpf** (CHANGELOG.md selbst wird nicht bearbeitet):
```markdown
### Fixed
- `loomux hook post-tool-use` names every lane it skipped, and every file of a call the shared budget did not reach, on stderr as well, so the model hears them when the edit is blocked. This corrects the v2.14.2 entry: the notices of an edit naming several files reached no host until now, because Antigravity does not read post-tool-use's stdout and Claude Code's edits name one file; they now arrive on stderr.
- `loomux hook post-tool-use` writes nothing to stdout unless it exits 0, so the callers of a green file no longer follow a red file of the same call, and with `--host claude` its context keeps `<`, `>` and `&` as they are. With `--host codex` it ends with exit 1 once the call names a file, as the Codex seam does elsewhere, instead of answering in Claude Code's shape.
```

- [ ] **Step 7: Commit**

```
git add internal/hooks/post_edit.go internal/hooks/post_edit_test.go internal/verify/report.go internal/verify/report_test.go internal/cli/hook.go internal/cli/hook_test.go internal/hosts/answer.go internal/hosts/answer_test.go internal/hosts/claude.go internal/hosts/hostio.go docs/en/cli-reference.md docs/de/cli-reference.md docs/en/hooks.md docs/de/hooks.md
```
`<scratch>/commit-10.txt`:
```
fix(hooks): answer post-edit through the host's adapter, and only at exit 0

post-edit wrote Claude Code's envelope itself, whatever --host said, and
wrote it at every exit code. A host reads stdout only at exit 0, so a red
call's context was never read, and the callers of a green file followed a
red file of the same call there. The envelope was also encoded apart from
the Claude adapter, with markup escaped and the keys in another order.

PostToolUse now takes the host, and RunPostEdit hands its notices to
hosts.WriteContext once, when the call ends with 0; an adapter that fails
ends the hook with 1. Claude Code gets the adapter's envelope, Antigravity's
injectSteps are dropped by the answer as before, and Codex, which has no
adapter, gets exit 1 once a file was checked instead of Claude's shape.
verify.WriteNotices goes.
```
`git commit -F <scratch>/commit-10.txt > <scratch>/gate-10.out 2>&1`.

---

### Task 11: Revive nur beim ersten Aufruf

**Files:**
- Modify: `internal/hooks/hook_session_start.go:52-73` (Rumpf von `SessionStart` zwischen `hosts.Read` und `hosts.WriteContext`)
- Modify: `internal/hosts/hostio.go:87-92` (Doku von `Payload.Repeat`)
- Modify: `docs/en/hooks.md` (~445-449), `docs/de/hooks.md` (~466-469)
- Test: `internal/hooks/hook_session_start_test.go`

**Interfaces:**
- Consumes: `hosts.Payload.Repeat bool`, `sessions.Revive(root, sessionID string) error`, `recordBase(sessionID, root string) error` (alle unverändert)
- Produces: keine neue Signatur; Verhalten: `sessions.Revive` läuft nur bei `!payload.Repeat`, weiterhin vor `recordBase`.

- [ ] **Step 1: Failing test schreiben** — In `internal/hooks/hook_session_start_test.go` den Test `TestHookSessionStartSaysAMarkerItCannotRemoveOnALaterInvocation` samt Kommentar (heute ~481-496) vollständig durch diese drei Tests ersetzen (Importe sind alle schon vorhanden: `bytes`, `os`, `path/filepath`, `strings`, `testing`, `time`, `config`, `sessions`):

```go
// A later PreInvocation revives nothing, so a marker it cannot take away is
// not said again: nothing retires an agy conversation between two model
// calls, and the first invocation already said it.
func TestHookSessionStartSaysNothingOfAMarkerOnALaterInvocation(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:1}}","green":"`+goneSHA+`","blocks":0}`)
	busy := filepath.Join(root, filepath.FromSlash(sessions.StateDir), "s1.ended", "inside")
	if err := os.MkdirAll(busy, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"conversationId":"s1","invocationNum":2}`), &stdout, &stderr, root, "antigravity")
	if code != ExitOK || stdout.Len() != 0 {
		t.Fatalf("%d %q %q", code, stdout.String(), stderr.String())
	}
}

// The first PreInvocation revives, so a marker it cannot take away is said.
func TestHookSessionStartSaysAMarkerItCannotRemoveOnTheFirstInvocation(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:1}}","green":"`+goneSHA+`","blocks":0}`)
	busy := filepath.Join(root, filepath.FromSlash(sessions.StateDir), "s1.ended", "inside")
	if err := os.MkdirAll(busy, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"conversationId":"s1","invocationNum":1}`), &stdout, &stderr, root, "antigravity")
	if code != ExitOK || !strings.Contains(stdout.String(), "injectSteps") || !strings.Contains(stdout.String(), "may not count for worktree unlink") {
		t.Fatalf("%d %q %q", code, stdout.String(), stderr.String())
	}
}

// The session counts again before its base is filed, so a base that cannot be
// written leaves it counted all the same. A directory where the session's file
// belongs makes the write fail; Revive's own touch of that path still lands,
// and it is how the order shows.
func TestHookSessionStartRevivesBeforeABaseItCannotWrite(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	root := project(t)
	gitInit(t, root)
	file := filepath.Join(root, filepath.FromSlash(sessions.StateDir), "s1.json")
	if err := os.MkdirAll(file, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(file, old, old); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d; stderr: %s", code, stderr.String())
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.ModTime().Before(time.Now().Add(-time.Hour)) {
		t.Fatalf("the session was not made young before the base failed: %s", info.ModTime())
	}
}
```

  Hinweis: `TestHookSessionStartRevivesBeforeABaseItCannotWrite` ist schon am heutigen Code grün (er hält die Reihenfolge fest, die die Änderung nicht brechen darf); rot wird nur der erste Test.

- [ ] **Step 2: Rot prüfen** — Run: `go test ./internal/hooks/ -run 'TestHookSessionStart(SaysNothingOfAMarker|SaysAMarkerItCannotRemoveOnTheFirst|RevivesBefore)' -count=1`
  Expected:
  ```
  --- FAIL: TestHookSessionStartSaysNothingOfAMarkerOnALaterInvocation (0.23s)
      hook_session_start_test.go:494: 0 "{\"injectSteps\":[{\"ephemeralMessage\":\"loomux: this session may not count for worktree unlink, ... s1.ended: Das Verzeichnis ist nicht leer.\"}]}\n" ""
  FAIL
  ```

- [ ] **Step 3: Implementieren** — `SessionStart` in `internal/hooks/hook_session_start.go` vollständig so (nur der Block ab dem Kommentar „A session worktree unlink retired“ bis vor `hosts.WriteContext` ändert sich; Revive bleibt VOR `recordBase`):

```go
func SessionStart(stdin io.Reader, stdout, stderr io.Writer, root, hostName string) int {
	host, err := hosts.ParseHost(hostName)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}
	payload, err := hosts.Read(host, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}

	// A session worktree unlink retired and that now resumes under the same
	// id counts again, and before the base is looked at, so that a base that
	// cannot be written does not leave it uncounted. A session that stays
	// uncounted is told in its context, the one channel a host reads at exit
	// 0: exit 1 would drop the context lines with it.
	//
	// Only at the first start: nothing retires an agy conversation between
	// two of its model calls (Retire is reached only through worktree unlink,
	// which is wired for Claude alone), so a later PreInvocation has nothing
	// to revive and a marker it cannot remove is not news. Should unlink ever
	// be wired for agy, this is to be decided again.
	var lines []string
	if payload.SessionID != "" && !payload.Repeat {
		if err := sessions.Revive(root, payload.SessionID); err != nil {
			lines = append(lines, "loomux: this session may not count for worktree unlink, so another session ending here may remove its junctions: "+err.Error())
		}
	}
	if err := recordBase(payload.SessionID, root); err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}

	// The base is filed and the warnings were said at the first start.
	if !payload.Repeat {
		lines = append(lines, staleBinary(root)...)
		lines = append(lines, updateWarnings(config.StateDir(), runtime.GOOS)...)
	}

	if err := hosts.WriteContext(host, "SessionStart", stdout, lines); err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}
```

  In `internal/hosts/hostio.go` den Kommentar über `Repeat bool` ersetzen:

```go
	// Repeat says the host fires this start again within one run: agy's
	// PreInvocation, where session-start is wired, comes before every model
	// call and counts them in invocationNum. What a start does is done once,
	// at the first, reviving a retired session included: nothing retires an
	// agy conversation between two model calls, since worktree unlink is
	// wired for Claude alone. Whether the count begins at 0 or 1 is not
	// measured; from 1 on, the worst case is one announcement too many.
	Repeat bool
```

  Achtung: `\n` in den `Fprintf`-Strings nur über Edit/Write schreiben, kein Heredoc (halbierte Backslashes); Dateien mit LF-Zeilenenden, danach `gofmt -l internal/hooks internal/hosts` leer.

- [ ] **Step 4: Grün prüfen** — Run: `go test ./internal/hooks/ -run 'TestHookSessionStart' -count=1`, dann `go test ./internal/hooks/ ./internal/hosts/ -count=1`. Expected: `ok  	github.com/xidus90/loomux/internal/hooks` und `ok  	github.com/xidus90/loomux/internal/hosts`. Coverage: `go test ./internal/hooks/ -count=1 -coverprofile=<scratch>/c.out` und `go tool cover -func=<scratch>/c.out | grep hook_session_start.go` → `SessionStart 100.0%`, `recordBase 100.0%`.

- [ ] **Step 5: Mutationsprobe** — (a) `&& !payload.Repeat` aus der Revive-Bedingung streichen → `TestHookSessionStartSaysNothingOfAMarkerOnALaterInvocation` rot. (b) Den `if payload.SessionID != "" && !payload.Repeat { … Revive … }`-Block hinter den `recordBase`-Block verschieben → `TestHookSessionStartRevivesBeforeABaseItCannotWrite` rot („the session was not made young before the base failed“). (c) `!payload.Repeat` durch `payload.Repeat` ersetzen → `TestHookSessionStartSaysAMarkerItCannotRemoveOnTheFirstInvocation` und `TestHookSessionStartSaysAMarkerItCannotRemove` (Claude) rot. Probe per `go test -overlay` mit einer mutierten Kopie im Scratchpad; Coverage-Läufe NICHT mit `-overlay` (siehe Validierung: `-coverprofile` ignoriert das Overlay hier).

- [ ] **Step 6: Doku** —
  `docs/en/hooks.md`, im Absatz, der mit „**Claude Code and Antigravity have adapters**“ beginnt (heute Zeilen 445-448), diese vier Zeilen

  ```markdown
  `session-start` runs on `PreInvocation`, which fires before every model call
  and counts them in `invocationNum`; only the first one announces, and a
  later one speaks only to say that the session could not be counted again
  for worktree unlink.
  ```

  ersetzen durch:

  ```markdown
  `session-start` runs on `PreInvocation`, which fires before every model call
  and counts them in `invocationNum`; only the first one announces and revives
  a session that worktree unlink retired. Nothing retires an agy conversation
  between two model calls, since unlink is wired for Claude alone, so a later
  one stays silent.
  ```

  `docs/de/hooks.md`, im Absatz, der mit „**Claude Code und Antigravity haben Adapter**“ beginnt (heute Zeilen 465-468), diese vier Zeilen

  ```markdown
  Ein unbekanntes Ereignis bleibt auf jedem Wirt Exit 2. `session-start` läuft
  auf `PreInvocation`, das vor jedem Modellaufruf feuert und sie in
  `invocationNum` zählt; nur der erste meldet sich, ein späterer nur, um zu
  sagen, dass die Sitzung für worktree unlink nicht wieder mitzählt.
  ```

  ersetzen durch:

  ```markdown
  Ein unbekanntes Ereignis bleibt auf jedem Wirt Exit 2. `session-start` läuft
  auf `PreInvocation`, das vor jedem Modellaufruf feuert und sie in
  `invocationNum` zählt; nur der erste meldet sich und zählt eine Sitzung
  wieder mit, die worktree unlink abgemeldet hat. Nichts meldet eine
  agy-Unterhaltung zwischen zwei Modellaufrufen ab, denn unlink ist nur für
  Claude verdrahtet; ein späterer Aufruf bleibt deshalb still.
  ```

  Die nächste Zeile (`subagent-start` and … / `subagent-start` und) bleibt stehen.

  Danach `grep -rn "later one speaks\|nicht wieder mitzählt\|on a later invocation\|OnALaterInvocation" docs internal --exclude-dir=.superpowers`. Erwartet: kein Treffer, der das alte Verhalten beschreibt. Der neue Test `TestHookSessionStartSaysNothingOfAMarkerOnALaterInvocation` trifft und stimmt.

- [ ] **Step 7: Commit** — `git add internal/hooks/hook_session_start.go internal/hooks/hook_session_start_test.go internal/hosts/hostio.go docs/en/hooks.md docs/de/hooks.md`; Nachricht per Write nach `<scratch>/msg11.txt`:

```
fix(hooks): revive a session only at the first session start

agy fires session-start before every model call. A later call now leaves
the end marker alone: nothing retires an agy conversation between two
model calls, since worktree unlink is wired for Claude alone, so a marker
that cannot be removed was said again on every call. The revive still runs
before the base is filed, so a base that cannot be written leaves the
session counted.
```

  `git commit -F <scratch>/msg11.txt > <scratch>/commit11.log 2>&1`; bei Rot `grep -e "--- FAIL" <scratch>/commit11.log`. Danach `git log -1 --format='%an <%ae>'` lesen.

### Task 12: invocationNum als Zahl oder Dezimal-String

**Files:**
- Modify: `internal/hosts/antigravity.go:3-9` (Import `strconv`), `:13-15` (protojson-Kommentar), `:44-45` (Repeat), neue Funktion `invocationOf` nach `readAntigravity`
- Modify: `docs/en/hooks.md`, `docs/de/hooks.md` (der in Task 11 ersetzte Satz)
- Test: `internal/hosts/antigravity_test.go`

**Interfaces:**
- Consumes: `hosts.Payload.Repeat` aus Task 11 (Semantik unverändert)
- Produces: `func invocationOf(v any) int64` (package `hosts`, unexportiert)

- [ ] **Step 1: Failing test schreiben** — in `internal/hosts/antigravity_test.go` direkt vor `// TestWriteAntigravityContext verifies …` einfügen (Importe `strings`, `testing`, `hosts` vorhanden):

```go
// protojson writes a 64-bit counter as a decimal string, and the width of
// invocationNum is not measured, so both spellings count; anything else, or a
// string that is no whole number, reads as the first invocation.
func TestReadAntigravityReadsTheInvocationAsANumberOrADecimalString(t *testing.T) {
	for body, repeat := range map[string]bool{
		`{"conversationId":"s1","invocationNum":2}`:     true,
		`{"conversationId":"s1","invocationNum":"2"}`:   true,
		`{"conversationId":"s1","invocationNum":"12"}`:  true,
		`{"conversationId":"s1","invocationNum":1}`:     false,
		`{"conversationId":"s1","invocationNum":"1"}`:   false,
		`{"conversationId":"s1","invocationNum":"0"}`:   false,
		`{"conversationId":"s1","invocationNum":"two"}`: false,
		`{"conversationId":"s1","invocationNum":"2.0"}`: false,
		`{"conversationId":"s1","invocationNum":""}`:    false,
		`{"conversationId":"s1","invocationNum":true}`:  false,
		`{"conversationId":"s1","invocationNum":null}`:  false,
		`{"conversationId":"s1","invocationNum":[2]}`:   false,
		`{"conversationId":"s1"}`:                       false,
	} {
		got, err := hosts.Read(hosts.HostAntigravity, strings.NewReader(body))
		if err != nil {
			t.Fatalf("%s: %v", body, err)
		}
		if got.Repeat != repeat {
			t.Errorf("%s: Repeat = %v, want %v", body, got.Repeat, repeat)
		}
	}
}
```

- [ ] **Step 2: Rot prüfen** — Run: `go test ./internal/hosts/ -run TestReadAntigravityReadsTheInvocation -count=1`
  Expected:
  ```
  --- FAIL: TestReadAntigravityReadsTheInvocationAsANumberOrADecimalString (0.00s)
      antigravity_test.go:150: {"conversationId":"s1","invocationNum":"2"}: Repeat = false, want true
      antigravity_test.go:150: {"conversationId":"s1","invocationNum":"12"}: Repeat = false, want true
  FAIL
  ```

- [ ] **Step 3: Implementieren** — Importblock von `internal/hosts/antigravity.go`:

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)
```

  `readAntigravity` vollständig (Doku-Kommentar mit dem neuen protojson-Satz) und die neue Funktion dahinter:

```go
// readAntigravity decodes an Antigravity hook payload.
//
// Antigravity (agy 1.2.2) emits protojson payloads for command hooks.
// protojson writes a 64-bit integer as a decimal string and a 32-bit one as a
// number; whether agy sends invocationNum at all, and in which width, is not
// measured (the recorded PreInvocation payload,
// testdata/cases/2c-payloads/agy-inv.json, carries only conversationId and
// stepIdx), so invocationOf takes both. The session identifier is delivered in `conversationId` (standard protojson camelCase)
// or `conversation_id`.
//
// Like readClaude, it refuses what is not an object (non-JSON, array, string, number, null).
// Fields are read by type assertion so mistyped values read as absent ("").
//
// For SubagentStart/SubagentStop: Antigravity passes toolCall in PreToolUse,
// but only stepIdx, error, and conversationId in PostToolUse -- no agent_id.
// Thus AgentID and AgentType are returned as empty (""), causing subagent-start/stop
// to reject with exit 1 ("payload carries no agent_id").
func readAntigravity(r io.Reader) (Payload, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Payload{}, fmt.Errorf("reading stdin: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		var wrongType *json.UnmarshalTypeError
		if errors.As(err, &wrongType) {
			return Payload{}, errNotAnObject
		}
		return Payload{}, fmt.Errorf("stdin is not JSON: %w", err)
	}
	if payload == nil {
		return Payload{}, errNotAnObject
	}
	sessionID, _ := payload["conversationId"].(string)
	if sessionID == "" {
		sessionID, _ = payload["conversation_id"].(string)
	}
	return Payload{SessionID: sessionID, Repeat: invocationOf(payload["invocationNum"]) > 1}, nil
}

// invocationOf reads invocationNum in either of protojson's spellings: a JSON
// number, or a decimal string, which is how it writes a 64-bit integer. Any
// other value, and a string that is no whole number, is 0; a Repeat read false
// by mistake only says the start's announcements once more.
func invocationOf(v any) int64 {
	switch v := v.(type) {
	case float64:
		return int64(v)
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}
```

- [ ] **Step 4: Grün prüfen** — Run: `go test ./internal/hosts/ -run TestReadAntigravityReadsTheInvocation -count=1`, dann `go test ./internal/hosts/ ./internal/hooks/ -count=1`; Expected: `ok` für beide. Coverage: `go test ./internal/hosts/ -count=1 -coverprofile=<scratch>/c.out`, `go tool cover -func=<scratch>/c.out | grep antigravity.go` → `readAntigravity 100.0%`, `invocationOf 100.0%`, `writeAntigravityContext 100.0%`; Paket `coverage: 100.0% of statements`.

- [ ] **Step 5: Mutationsprobe** — (a) den `case string:`-Arm streichen → `TestReadAntigravityReadsTheInvocationAsANumberOrADecimalString` rot (`"2"`, `"12"`). (b) den `case float64:`-Arm streichen → derselbe Test rot (`2` als Zahl) und `TestHookSessionStartOnAntigravity` rot. (c) im Fehlerarm `return n` statt `return 0` → bleibt grün, weil `ParseInt` im Fehlerfall 0 liefert (außer bei Überlauf; kein eigener Test nötig, der Arm dokumentiert die Absicht).

- [ ] **Step 6: Doku** — Den in Task 11 geschriebenen Satz ergänzen.
  `docs/en/hooks.md`: nach „… so a later one stays silent.“ anhängen: „`invocationNum` is read as a number or as a decimal string, protojson's spelling of a 64-bit integer.“
  `docs/de/hooks.md`: nach „… ein späterer Aufruf bleibt deshalb still.“ anhängen: „`invocationNum` wird als Zahl oder als Dezimal-String gelesen, protojsons Schreibweise einer 64-Bit-Ganzzahl.“

- [ ] **Step 7: Commit** — `git add internal/hosts/antigravity.go internal/hosts/antigravity_test.go docs/en/hooks.md docs/de/hooks.md`; Nachricht per Write nach `<scratch>/msg12.txt`:

```
fix(hosts): read agy's invocationNum as a number or a decimal string

protojson writes a 64-bit integer as a decimal string, and whether agy
sends invocationNum at all, and in which width, is not measured. Read as a
number only, a string count left
every model call looking like the first, and session-start announced its
warnings before each one.
```

  `git commit -F <scratch>/msg12.txt > <scratch>/commit12.log 2>&1`; danach `git log -1 --format='%an <%ae>'`.

---

### Task 13: Arbeitspapiere und Testkommentar nachführen

**Files:**
- Modify: `docs/.superpowers/parity/stufe-4a-2.md:689` (neuer Abschnitt nach der Zeile „sperrt, Post-Edit prüft aus `toolCall`).“, vor `## Offen`)
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md:739` (Zeile #23 an Ort und Stelle; danach `go test ./internal/plancheck/ -count=1`)
- Modify: `docs/.superpowers/specs/2026-09-25-antigravity-hooks-integration-design.md:7-8, 63-65, 75-79, 81-94, 115, 137-138, 146-152`
- Modify: `docs/.superpowers/plans/2026-09-25-antigravity-hooks-integration.md:144` (Nachtrag am Dateiende)
- Modify: `internal/setup/helpers_test.go:53-57` (nur Kommentar)
- Test: keiner (nur Prosa und ein Kommentar); Prüfung per grep und `go vet`

**Interfaces:**
- Consumes: die Regeln aus Task 1 (Werkzeugtabelle `commandTools` mit `keys`, `lineOptional`, `whole`, `quiet`; Schlüssel per `EqualFold`; Zeilen vor der `Action`; Tipp-Eingaben nur als ganze Zeilen), Task 2 (Aufruf ohne Werkzeugnamen verweigert), Task 4 (Anhängen fehlender Werkzeuge, nur nach der Messung aus Task 0), Task 10 (post-edit über `hosts.WriteContext`, nichts auf stdout bei Exit ≠ 0), Task 11 (`Revive` nur beim ersten Aufruf), Task 12 (`invocationNum` als Zahl oder Dezimal-String), und das Ergebnis der Messung aus Task 0 (agy-Version und Ausgang).
- Produces: nichts, was Code nutzt.

Diese Task läuft als letzte, nach Task 0–12, damit die Papiere die endgültigen Regeln nennen. Alle Änderungen per Edit (nie per sed/Heredoc: die Texte tragen Backticks, `\n` und `\r`). Historische Zeilen (datiert 2026-09-25) bleiben stehen; ein datierter Nachtrag folgt ihnen. Nur Fusions-Nachtrag #23 wird an Ort und Stelle umformuliert, weil er noch „Freigabe offen“ ist.

- [ ] **Step 1: Stand vor der Änderung festhalten (Rot)**

Run (Git Bash, aus dem Worktree; der Plan selbst zitiert jedes Muster und wird ausgenommen):

```sh
X=--exclude=2026-09-27-antigravity-review-findings.md
grep -rnF $X 'nur beim ersten' docs/.superpowers
grep -rnF $X '`run_command` und `send_command_input`' docs/.superpowers
grep -rnF $X '|run_command|send_command_input"' docs/.superpowers
grep -nF 'danach nur, wenn eine Sitzung seither ungezählt blieb' docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md
grep -nF 'a space, & or a parenthesis' internal/setup/helpers_test.go
```

Expected (auf 9428f0af plus den Planzweig gesehen):
- `nur beim ersten`: `parity/code-g2a.md:165` (Graph-Mutant, anderer Zusammenhang), `parity/stufe-4a-2.md:684`, `plans/2026-09-25-antigravity-hooks-integration.md:142`, `specs/2026-09-27-antigravity-review-findings-design.md:41`, `:317`, `:410`. Die Fusions-Spec trifft nicht mehr, master hat #23 umformuliert.
- `` `run_command` und `send_command_input` ``: `specs/2026-09-25-antigravity-hooks-integration-design.md:7`, `:81`
- `|run_command|send_command_input"`: `specs/2026-09-25-antigravity-hooks-integration-design.md:115`
- `danach nur, wenn …`: `:739` (Nachtrag #23)
- `helpers_test.go:55`

- [ ] **Step 2: Paritätsakte — Nachtrag nach „Umbau nach der Probe“**

In `docs/.superpowers/parity/stufe-4a-2.md` nach der Zeile `  sperrt, Post-Edit prüft aus `toolCall`).` und vor der Leerzeile vor `## Offen` einfügen (eine Leerzeile davor):

```markdown

### Nachtrag nach dem Code-Review, 2026-09-27

Die Zeilen oben geben den Stand vom 2026-09-25; wo sie davon abweichen, gilt:

- `manage_task` steht im Matcher von `PreToolUse` und in den Befehlsregeln,
  Argument `Input` (gemessen mit agy 1.2.11, siehe oben). Die Werkzeuge
  stehen in einer Tabelle `commandTools`; deren Nullwerte sind der
  geschlossene Fall.
- Die Argumentnamen gelten ohne Rücksicht auf die Schreibung, und jeder
  Treffer wird geprüft. Eine gefundene Zeile wird immer geprüft, gleich
  welche `Action` der Aufruf nennt: `{"Action":"kill","Input":"git push
  origin main\n"}` wird als Push verweigert. `list`, `status` und `kill`
  entschuldigen nur das Fehlen einer Zeile, und nur, wenn jeder Schlüssel
  `action` in jeder Schreibung einen dieser drei Strings trägt. „Die ersten
  drei laufen durch“ oben gilt also nur für einen Aufruf ohne Zeile.
- Was `send_command_input` und `manage_task` tippen, geht nur als ganze
  Zeilen durch: Der Wert endet auf `\n` oder `\r` und trägt kein
  Steuerzeichen außer `\n` und `\r` (kein C0, also auch kein Tab, an dem eine
  interaktive Shell ein Wort vervollständigt, kein DEL, kein C1, also auch
  kein U+009B), und keine seiner Zeilen endet auf `\` oder `` ` ``, mit denen
  bash und PowerShell die Zeile in der nächsten fortsetzen. Jede nicht leere
  Zeile läuft durch die Befehlsregeln. Ein Wert unter einem Zeilenschlüssel,
  der kein String ist, wird verweigert, ein leerer trägt keine Zeile.
  Ein Fragment, Ctrl-C, eine Pfeiltaste, ein Tab, ein Backspace oder eine
  Zeilenfortsetzung wird verweigert; eine Aufgabe beendet `kill`.
- Ein dekodierter Aufruf ohne Werkzeugnamen wird verweigert („loomux found
  no tool name in this call, so it cannot judge it and refuses“); bis hier
  ließ ihn die Policy durch.
- Ein wiederholtes `PreInvocation` sagt nichts: `sessions.Revive` läuft nur
  beim ersten Aufruf, eine seither beendete Sitzung wird dort also nicht
  wieder gezählt und nicht gemeldet. Nichts beendet eine agy-Unterhaltung
  zwischen zwei Modellaufrufen (`Retire` ruft nur `WorktreeUnlink`, das nur
  für Claude verdrahtet ist); wird unlink je für agy verdrahtet, ist das neu
  zu entscheiden.
- `invocationNum` wird als Zahl oder als Dezimal-String gelesen; ob und in
  welcher Form agy es schickt, ist ungemessen: die aufgezeichnete
  PreInvocation-Nutzlast (`testdata/cases/2c-payloads/agy-inv.json`, agy
  1.2.2) trägt nur `conversationId` und `stepIdx`. Ein unlesbarer oder
  fehlender Wert gilt als erster Aufruf und wiederholt nur Ankündigungen.
- post-edit schreibt seinen Kontext bei Exit 0 über `hosts.WriteContext`,
  für agy als `injectSteps`; `hosts.Answer` verwirft post-tool-use-stdout
  weiter. agy liest `injectSteps` auf PostToolUse: ungemessen. Bei Exit ≠ 0
  schreibt post-edit nichts auf stdout, und jeder Hinweis auf eine
  übersprungene Lane oder Datei steht auf stderr.
- Messung zu `init` mit zwei eigenen Blöcken unter `PreToolUse`: siehe die
  nächste Zeile.
```

Hat Task 0 `block I`-Nutzlasten aufgezeichnet, ersetzt der Ausführende im `invocationNum`-Punkt oben den Teil ab „ob und in welcher Form“ bis „`stepIdx`.“ durch den gemessenen Befund, datiert und mit der agy-Version: Feld vorhanden oder nicht, Zahl oder String, Wert beim ersten und zweiten Aufruf. Fehlt das Feld, sagt der Punkt zusätzlich, dass `Repeat` damit nie gesetzt wird und `session-start` sich vor jedem Modellaufruf meldet. Fehlt `block I` ganz, bleibt der Punkt unverändert.

Danach genau eine der beiden folgenden Zeilen anfügen, je nach dem Ausgang der Messung aus Task 0; `2026-09-27` und `1.2.11` ersetzt der Ausführende durch Datum und Version aus der Notiz von Task 0 Schritt 6, wörtlich; sonst bleibt der Satz unverändert:

Ausgang A (agy lud die Datei und rief den Hook für ein Werkzeug des zweiten Blocks):

```markdown
- Gemessen am 2026-09-27 mit agy 1.2.11: Eine `hooks.json`, deren Gruppe
  `loomux` unter `PreToolUse` zwei Blöcke mit verschiedenen Matchern trägt,
  lädt ohne Parse-Fehler im Log, und agy ruft den Hook für ein Werkzeug des
  zweiten Blocks. `init` hängt deshalb für einen eigenen Block mit älterem,
  flachem Matcher einen Block mit den fehlenden Werkzeugen an.
```

Ausgang B (Messung gescheitert; Task 4 wurde nicht gebaut):

```markdown
- Gemessen am 2026-09-27 mit agy 1.2.11: Eine `hooks.json`, deren Gruppe
  `loomux` unter `PreToolUse` zwei Blöcke mit verschiedenen Matchern trägt,
  lädt agy nicht, oder agy ruft den Hook für ein Werkzeug des zweiten Blocks
  nicht. `init` hängt deshalb nichts an und meldet einen älteren Matcher nur;
  wie `manage_task` in bestehende Installationen kommt, entscheidet der
  Nutzer neu.
```

- [ ] **Step 3: Fusions-Spec, Nachtrag #23 an Ort und Stelle**

In `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` die ganze Zeile, die mit `| 23 | — | Das Antwortprotokoll der Hooks gegenüber Antigravity` beginnt (heute Zeile 739), durch diese eine Zeile ersetzen:

```markdown
| 23 | — | Das Antwortprotokoll der Hooks gegenüber Antigravity und die Form von `.agents/hooks.json` | Stufe 4a-2: ein Adapter `hosts.Answer`, den `loomux hook` einmal für jeden Austrittspfad ruft, eine Panik eingeschlossen. Für Antigravity: `pre-tool-use` verweigert mit Exit 2, `post-tool-use` warnt mit Exit 2 und schreibt bei Exit 0 seinen Kontext über `hosts.WriteContext` als `injectSteps`, den `hosts.Answer` verwirft, ein gehaltener Stop wird `{"decision":"continue","reason":…}` auf stdout mit Exit 0, jeder andere Code ungleich 0 endet mit 0; ein unbekanntes Ereignis bleibt Exit 2. `session-start` auf `PreInvocation` meldet sich beim ersten `invocationNum` (Zahl oder Dezimal-String) und belebt die Sitzung nur dann; ein späterer sagt nichts. `run_command`, `send_command_input` und `manage_task` (Aktion `send_input`, Argument `Input`) stehen in einer Werkzeugtabelle und laufen durch die Befehlsregeln: jede Zeile unter einem ihrer Argumentnamen, ohne Rücksicht auf die Schreibung, wird geprüft, gleich welche `Action` der Aufruf nennt; die stillen Aktionen `list`, `status` und `kill` von `manage_task` entschuldigen nur das Fehlen einer Zeile, sonst wird ohne Zeile verweigert; was `send_command_input` und `manage_task` tippen, geht nur als ganze Zeilen ohne Steuerzeichen und ohne Zeilenfortsetzung (`\` oder `` ` `` am Zeilenende) durch. Ein Wert unter einem Zeilenschlüssel, der kein String ist, macht den Aufruf unprüfbar und wird verweigert; ein leerer String trägt keine Zeile. Ein Aufruf ohne Werkzeugnamen wird verweigert. Post-Edit liest die Ziele aus dem `toolCall` des PostToolUse; keine Ablage. `PreInvocation` und `Stop` stehen flach in `.agents/hooks.json` | Gemessen mit agy 1.2.8 und 1.2.11 am 2026-09-25 (`parity/stufe-4a-2.md`): gruppierte `Stop`/`PreInvocation` lassen agy die ganze Datei verwerfen, der Wächter lädt dann nicht; PostToolUse trägt `toolCall` entgegen dem Leitfaden; Exit 2 in PostToolUse bricht nicht ab; `continue` hält den Stop. Nachgemessen am 2026-09-25 mit agy 1.2.11: in einen offenen Befehl tippt agy über `manage_task` mit `send_input` und `Input`, beenden kommt als `manage_task` mit `kill`, nicht über `send_command_input` (`parity/stufe-4a-2.md`). Ungemessen: der Argumentname von `send_command_input`, ob `PreInvocation` `invocationNum` überhaupt trägt, seine Form und Breite und ob die Zählung bei 0 oder 1 beginnt, und ob agy `injectSteps` auf PostToolUse liest | vorgeschlagen 2026-09-25, nachgeführt 2026-09-27 nach dem Code-Review, Freigabe offen |
```

- [ ] **Step 4: Antigravity-Spec vom 2026-09-25**

`docs/.superpowers/specs/2026-09-25-antigravity-hooks-integration-design.md`:

(a) Zeile 7 (`**Überarbeitet:** …`) bleibt; sie endet auf zwei Leerzeichen. Direkt dahinter eine neue Zeile 8 einfügen, ebenfalls mit zwei Leerzeichen am Ende:

```markdown
**Nachgetragen:** 2026-09-27 nach dem Code-Review: `manage_task` im Matcher und in den Befehlsregeln, Zeilen vor der `Action`, Tipp-Eingaben nur als ganze Zeilen, ein Aufruf ohne Werkzeugnamen verweigert, `session-start` still bei einem späteren `invocationNum`, post-edit über `hosts.WriteContext`  
```

(b) In 2.2 den Punkt `* **Antigravity, `post-tool-use`:** Exit 2 bleibt: …` (heute Zeilen 63-65, drei Zeilen bis `Exit 1 wird 0.`) ersetzen durch:

```markdown
* **Antigravity, `post-tool-use`:** Exit 2 bleibt: agy gibt stderr dem Modell
  als Warnung und bricht nicht ab. stdout fällt weg (agy erwartet `{}`); Exit 1
  wird 0. Nachtrag 2026-09-27: post-edit schreibt seinen Kontext bei Exit 0
  über `hosts.WriteContext` als `injectSteps`, `hosts.Answer` verwirft ihn;
  ob agy `injectSteps` auf PostToolUse liest, ist ungemessen. Bei Exit ≠ 0
  schreibt post-edit nichts auf stdout, und jeder Hinweis auf eine
  übersprungene Lane oder Datei steht auf stderr. Codex: `hosts.WriteContext`
  antwortet `ErrNoAdapter`, `post-tool-use --host codex` endet bei Code 0
  mit 1, sobald der Aufruf eine Datei nennt.
```

(c) Den Absatz `` `session-start` läuft auf `PreInvocation`, … `` (heute Zeilen 75-79, fünf Zeilen bis `Einen `SessionStart`-Hook kennt `hooks.json` nicht.`) ersetzen durch:

```markdown
`session-start` läuft auf `PreInvocation`, das vor jedem Modellaufruf feuert
und die Aufrufe in `invocationNum` zählt; `hosts.Payload.Repeat` ist gesetzt ab
`invocationNum > 1`, als Zahl oder als Dezimal-String gelesen (protojson kann
64-Bit-Zähler als String schicken; Form und Breite des Felds sind ungemessen,
und ob agy es überhaupt schickt, auch: die aufgezeichnete PreInvocation-Nutzlast,
`testdata/cases/2c-payloads/agy-inv.json`, trägt nur `conversationId` und `stepIdx`).
Bei `Repeat` schreibt `session-start` keinen Kontext und belebt die Sitzung
nicht wieder: `sessions.Revive` läuft nur beim ersten Aufruf (Nachtrag
2026-09-27). Ob die Zählung bei 0 oder 1 beginnt, ist ungemessen;
schlimmstenfalls meldet es sich einmal zu oft. Einen `SessionStart`-Hook kennt
`hooks.json` nicht.
```

(d) Abschnitt 2.3 ersetzen: die Überschrift `### 2.3 `run_command` und `send_command_input` (`internal/hooks/guard.go`)`, den Absatz darunter bis `verweigert. Beide stehen im Matcher von `PreToolUse`.` und den folgenden Absatz `Nachgemessen am 2026-09-25 mit agy 1.2.11: agy tippt eine Eingabe nicht über` bis `(`parity/stufe-4a-2.md`, `internal/hooks/guard.go`).`, heute Zeilen 81-94. Neu:

```markdown
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
```

(e) Im Codeblock von 2.5 die Zeile mit `{Event: "PreToolUse", Matcher: writers + "|run_command|send_command_input", …` (heute Zeile 115) ersetzen durch diese Zeile, gleiche Einrückung (vier Leerzeichen plus vier):

```go
        {Event: "PreToolUse", Matcher: writers + "|run_command|send_command_input|manage_task", Command: hook("pre-tool-use"), Timeout: 15},
```

(f) In der Tabelle von Abschnitt 3 die beiden Zeilen `| `run_command`/`send_command_input` ohne bekannte Befehlszeile | Verweigert mit Exit 2. |` und `| Wiederholtes `PreInvocation` | `session-start` schreibt keinen Kontext. |` (heute 137-138) ersetzen durch diese drei; die dritte Zeile ist neu:

```markdown
| `run_command`/`send_command_input`/`manage_task` ohne erkennbare Zeile (bei `manage_task` außer `list`, `status`, `kill`), eine Tipp-Eingabe ohne Zeilenende, mit Steuerzeichen oder mit einer Zeile, die auf `\` oder `` ` `` endet, ein Aufruf ohne Werkzeugnamen | Verweigert mit Exit 2. |
| Wiederholtes `PreInvocation` | `session-start` schreibt keinen Kontext und belebt die Sitzung nicht wieder. |
| `post-tool-use --host codex`, Aufruf nennt eine Datei (auch eine mit ignorierter Endung) | Exit 1 mit dem Adapterfehler (`ErrNoAdapter`); ohne Datei Exit 0 (Nachtrag 2026-09-27). |
```

(g) In Abschnitt 4 die Punkte 1 und 2 ersetzen, von `1. **`internal/hosts/`**:` bis `einem späteren `invocationNum`.` (heute 146-152). Neu:

```markdown
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
```

- [ ] **Step 5: Plan vom 2026-09-25 — Nachtrag am Ende**

In `docs/.superpowers/plans/2026-09-25-antigravity-hooks-integration.md` die abgehakten Schritte stehen lassen und am Dateiende (nach `- [x] **Step 7: Doku, Gate**`) anfügen:

```markdown

**Nachtrag 2026-09-27 (Code-Review):** Step 4 und Step 5 beschreiben den Stand
vom 2026-09-25. Seither steht auch `manage_task` im Matcher von `PreToolUse`
(`writers + "|run_command|send_command_input|manage_task"`) und in den
Befehlsregeln; jede gefundene Zeile wird geprüft, gleich welche `Action` der
Aufruf nennt, die Argumentnamen ohne Rücksicht auf die Schreibung, und was
`send_command_input` und `manage_task` tippen, geht nur als ganze Zeilen ohne
Steuerzeichen und ohne Zeilenfortsetzung durch; ein Wert unter einem
Zeilenschlüssel, der kein String ist, macht den Aufruf unprüfbar und wird
verweigert; ein leerer String trägt keine Zeile; ein Aufruf ohne
Werkzeugnamen wird verweigert.
`session-start` sagt bei einem späteren `invocationNum` (Zahl oder
Dezimal-String) nichts und belebt die Sitzung nicht wieder. post-edit schreibt
seinen Kontext bei Exit 0 über `hosts.WriteContext`, für agy als
`injectSteps`, das `hosts.Answer` verwirft (ob agy es auf PostToolUse liest,
ist ungemessen); bei Exit ≠ 0 schreibt es nichts auf stdout. Die Aussage unter
„Architecture“, Antigravity bekomme Kontext als `injectSteps` auf stdout, gilt
damit nur für `session-start`. Stand der Regeln: `parity/stufe-4a-2.md`,
„Nachtrag nach dem Code-Review, 2026-09-27“.
```

- [ ] **Step 6: Testkommentar in `internal/setup/helpers_test.go`**

Den Kommentar über `func localAppData` ersetzen durch:

```go
// localAppData is an empty directory for LOCALAPPDATA whose path cmdSplits
// passes. t.TempDir() is one unless the temp directory or the test's name
// carries a character cmdSplits rejects (C:\Users\Jane Doe\…); then the
// directory is made at the root of the same volume instead and removed
// afterwards.
```

Run: `go vet ./internal/setup/` und `go test ./internal/setup/ -count=1`
Expected: kein Befund; `ok  	github.com/xidus90/loomux/internal/setup`

- [ ] **Step 7: Grün prüfen (grep)**

Run (Git Bash, aus dem Worktree):

```sh
X=--exclude=2026-09-27-antigravity-review-findings.md
grep -rnF $X 'nur beim ersten' docs/.superpowers
grep -rnF $X '`run_command` und `send_command_input`' docs/.superpowers
grep -rnF $X '|run_command|send_command_input"' docs/.superpowers
grep -nF 'danach nur, wenn eine Sitzung seither ungezählt blieb' docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md
grep -nF 'a space, & or a parenthesis' internal/setup/helpers_test.go
grep -nF 'Nachtrag nach dem Code-Review, 2026-09-27' docs/.superpowers/parity/stufe-4a-2.md
grep -nF 'Gemessen am 2026-09-27 mit agy' docs/.superpowers/parity/stufe-4a-2.md
grep -nF 'nachgeführt 2026-09-27' docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md
```

Expected:
- `nur beim ersten`: nur noch diese Treffer:
  - `parity/code-g2a.md:165` (Graph-Mutant, anderer Zusammenhang)
  - `parity/stufe-4a-2.md:684`: historische Zeile vom 2026-09-25, der Nachtrag darunter berichtigt sie
  - `plans/2026-09-25-antigravity-hooks-integration.md:142`: abgehakter Schritt, der Nachtrag am Ende berichtigt ihn
  - `specs/2026-09-25-antigravity-hooks-integration-design.md:89`: der neue Satz aus Step 4 (c), „`sessions.Revive` läuft nur beim ersten Aufruf“; stimmt
  - `specs/2026-09-27-antigravity-review-findings-design.md:41`, `:317`, `:410`: betreffen `Revive`, stimmen

  Die Fusions-Spec trifft nicht.
- `` `run_command` und `send_command_input` ``: nur `specs/2026-09-25-antigravity-hooks-integration-design.md:7`. Das ist die historische „Überarbeitet“-Zeile; Zeile 8 trägt den Nachtrag.
- `|run_command|send_command_input"`: kein Treffer (Exit 1).
- `danach nur, wenn …`: kein Treffer (Exit 1).
- `a space, & or a parenthesis`: kein Treffer (Exit 1).
- Die drei letzten greps: je genau ein Treffer.

Eine Mutationsprobe entfällt, es gibt keinen Code. Stattdessen: Wer in Step 4 (e) `|manage_task` weglässt, sieht den dritten grep wieder `:115` melden.

- [ ] **Step 8: Commit**

Vor dem Commit `git rev-parse --show-toplevel` gegen den Worktree-Pfad und `git branch --show-current` (`fix/antigravity-review-findings`) lesen.

```sh
git add docs/.superpowers/parity/stufe-4a-2.md docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md docs/.superpowers/specs/2026-09-25-antigravity-hooks-integration-design.md docs/.superpowers/plans/2026-09-25-antigravity-hooks-integration.md internal/setup/helpers_test.go
```

Commit-Text per Write nach `<scratch>/commit-13.txt`:

```text
docs: describe the current Antigravity guard and session-start rules

The Antigravity design documents under docs/.superpowers still said that
only run_command and send_command_input reach the command rules, that
list, status and kill pass whatever they carry, and that a repeated
PreInvocation revives a session. They now describe manage_task in the
matcher, lines judged before the Action, typed input accepted only as
whole lines without a continuation, a call without a tool name refused, a
repeated PreInvocation that says nothing, and post-edit context written
through the hosts seam.

The localAppData test comment names cmdSplits instead of listing a
subset of the characters it rejects.
```

```sh
git commit -F <scratch>/commit-13.txt > <scratch>/gate-13.log 2>&1; echo $?
```

Expected: `0`; bei Rot erst `<scratch>/gate-13.log` nach `--- FAIL` durchsuchen. Danach `git log -1 --format='%an <%ae>'` lesen: der Nutzer, kein Co-Author.

---

## Nicht Teil dieses Plans

- Folgearbeit auf master, schon vor diesem Zweig vorhanden (beim Review per Probe gefunden): Die eingebaute Push-Regel `(^|\s)git\s+push(\s|$)` lässt auf jedem Befehlswerkzeug `;git push`, `&&git push`, `(git push)` und ein geschütztes Leerzeichen oder VT (U+000B) als Trenner durch, und `writesConfiguration` teilt eine ganze Befehlszeile nur an `\n`, nicht an einem einzelnen `\r`; zu weiten ist die Regel auf Trenner ohne Leerraum samt Tests, und `splitSegments` soll auch an `\r` schneiden.

## PR-Rumpf: Release und Changelog

`Release: patch — closes guard, init and post-edit gaps the review of the Antigravity fixes found; no command, flag, config format or wired host's exit code changes.`

Gilt Ausgang B aus Task 0 (Task 4 nicht gebaut), entfällt der dritte Security-Punkt.

```markdown
## Changelog

### Security
- Antigravity: `loomux hook pre-tool-use` judges every argument name of a command tool regardless of case, judges a line a `manage_task` call carries whatever its `Action` says, and refuses a value under such a name that is no string.
- Antigravity: what an agent types into a task with `manage_task` or `send_command_input` passes only as whole lines without control characters, and a line ending in a backslash or a backtick, which bash and PowerShell continue on the next line, is refused, so a command split across calls or lines, edited with a backspace or completed at a tab no longer slips past the command rules. A single keystroke without Enter, Ctrl-C and arrow keys can no longer be sent to a task; `kill` still ends one.
- `loomux init` appends a block for the tools an own hook entry under an older matcher lacks, `manage_task` for Antigravity and `MultiEdit` for a Claude Code entry of ours under ulinit's matcher, so an upgrade guards them without a hand edit.
- `loomux hook pre-tool-use` refuses a call that names no tool instead of letting it pass.

### Fixed
- `loomux hook post-tool-use` names every lane it skipped, and every file of a call the shared budget did not reach, on stderr as well, so the model hears them when the edit is blocked. This corrects the v2.14.2 entry: the notices of an edit naming several files reached no host until now, because Antigravity does not read post-tool-use's stdout and Claude Code's edits name one file; they now arrive on stderr.
- `loomux hook post-tool-use` writes nothing to stdout unless it exits 0, so the callers of a green file no longer follow a red file of the same call, and with `--host claude` its context keeps `<`, `>` and `&` as they are. With `--host codex` it ends with exit 1 once the call names a file, as the Codex seam does elsewhere, instead of answering in Claude Code's shape.
- Antigravity: `loomux hook session-start` revives a session only at the first model call, so a marker it cannot remove is no longer repeated before every later one.
- Antigravity: `invocationNum` is also read as a decimal string, protojson's spelling of a 64-bit integer.
```
