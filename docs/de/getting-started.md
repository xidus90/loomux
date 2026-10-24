# Ersteinrichtung

← [loomux installieren](install.md) · [Übersicht](README.md) · [Konfiguration](configuration.md) →

Diese Seite führt ein Projekt von einem installierten loomux zu einer laufenden Einrichtung: Sie initialisiert das Repository, prüft die Konfiguration, prüft das Ergebnis, bindet deinen Agenten an und zeigt, wie man einen Bereich von Hand einrichtet. Sie setzt voraus, dass loomux so installiert ist, wie [loomux installieren](install.md) es beschreibt; die Beispiele rufen es als `loomux` auf. Liegt der Ordner nicht auf dem `PATH`, ruf das Binary mit seinem vollen Pfad auf.

---

## 1. Repository initialisieren

Wechsle in das Wurzelverzeichnis deines Projekts und führe aus:
```bash
loomux init
```
Dieser Befehl:
1. Erkennt vorhandene Coding-Agenten in deinem Arbeitsbereich (`.claude/` → Claude Code, `.agents/hooks.json`, `.agents/skills/` oder `GEMINI.md` → Antigravity).
2. Fragt je Modul (`hooks`, `brain`, `graph`) `all`, `each` oder `none` und dann die Commit-Sprache.
3. Zeigt jede Änderung als Diff — `.loomux/config.toml` (Module, Commit-Sprache, Policy-Regeln des Stacks; die Prüfketten kommen aus den Presets), `.gitignore`, Hook-Einträge der Agenten, Git-Hooks, Skills — und schreibt nur, was du bestätigst.
4. Legt das neueste Release nach `%LOCALAPPDATA%\loomux\bin\loomux.exe`, das die Hook-Einträge rufen, wenn dort noch keines liegt (nur unter Windows; siehe [loomux installieren](install.md)).

Alle Teile, Flags und Exit-Codes stehen in der [CLI-Referenz](cli-reference.md#11-projekt-einrichten-loomux-init).

> [!NOTE]
> Mit dem Flag `--dry-run` siehst du vorab, welche Dateien berührt würden, ohne Änderungen vorzunehmen:
> `loomux init --dry-run`

---

## 2. Konfiguration prüfen

Öffne `.loomux/config.toml`. `loomux init` hat nur geschrieben, was von den Vorgaben abweicht (siehe [Der erste Tag](configuration.md#der-erste-tag)). Eine kleine Konfiguration mit eigenen Regeln sieht so aus:

```toml
# .loomux/config.toml

[policy.paths]
# Sensible Dateien vor unbefugten Agenten-Schreibzugriffen schützen
rules = [
  { match = [".env*", "*.pem", "*.key"], reason = "Secrets dürfen nicht von Agenten bearbeitet werden" },
  { match = ["package-lock.json", "go.sum"], reason = "Lockfiles werden nur durch Paketmanager verwaltet" }
]

[policy.commands]
# Zerstörerische oder unautorisierte Befehle blockieren
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Pushen zum Remote erfordert eine explizite menschliche Entscheidung" }
]

[verify.go.test]
# Lanes kommen aus eingebauten Presets je erkanntem Stack; eine Tabelle ändert
# nur die Schlüssel, die sie nennt. `loomux check precommit --show` zeigt, was läuft.
measuring = "go test ./... -count=1 -covermode=set -coverpkg=example.com/my-project/... -coverprofile={coverprofile}"
```

---

## 3. Einrichtung prüfen

Starte die Diagnose:
```bash
loomux status
```
Die Ausgabe zeigt dir den Status aller Schutz- und Prüfmechanismen (Auszug):
```text
================================================================================
 loomux Hook Inspection
================================================================================
Project Root:    C:\Projects\my-project
Detected Stacks: [go]
...
[PostToolUse] (Matcher: Write|Edit|NotebookEdit)
  -> loomux hook post-tool-use (profile `edit`: lint, types):
     * go (*.go) lint: go vet ./... ; {loomux} check gofmt {file} [preset, parallel]
...
--------------------------------------------------------------------------------
 Lane Tools On This Machine
--------------------------------------------------------------------------------
 [OK] Every configured lane's tool is on PATH.
```

> [!NOTE]
> Die Schreibschranke öffnet die Bäume, die in der globalen Registry stehen
> (`%LOCALAPPDATA%\loomux\registry.toml`), dazu einige weitere Orte wie das
> Gedächtnis der Agenten und das Scratchpad der Sitzung. Ein verknüpfter Git-Worktree
> eines Repos, das mit `workspace = true` registriert ist, zählt zu diesem Repo und
> braucht keinen eigenen Eintrag; die Schranke erkennt den Worktree an Gits eigenen
> Worktree-Dateien, ohne dafür einen `git`-Prozess zu starten. Ein ohne
> `git worktree repair` verschobener Worktree bleibt gesperrt.

---

## 4. Den Agenten anbinden

`loomux init` schreibt die Hook-Einträge der Hosts, die es erkennt (Schritt 1). Dieser Abschnitt zeigt sie und wie du einen MCP-Client anbindest.

### Claude Code
`loomux init` schreibt diese Einträge in `.claude/settings.json` und ruft das Binary über seinen Pfad auf (`"${LOCALAPPDATA}/loomux/bin/loomux.exe"`); hier stehen sie mit `loomux`. `loomux status` nennt, was fehlt. Die sechs Einträge, mit den Fristen, die `init` schreibt:
```json
{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "timeout": 20,
      "command": "loomux hook session-start --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "PreToolUse": [{"matcher": "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell",
      "hooks": [{"type": "command", "timeout": 15,
      "command": "loomux hook pre-tool-use --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "PostToolUse": [{"matcher": "Write|Edit|MultiEdit|NotebookEdit",
      "hooks": [{"type": "command", "timeout": 60,
      "command": "loomux hook post-tool-use --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "Stop": [{"hooks": [{"type": "command", "timeout": 300,
      "command": "loomux hook stop --host claude --root \"${CLAUDE_PROJECT_DIR}\" --budget 270s"}]}],
    "SubagentStart": [{"hooks": [{"type": "command", "timeout": 30,
      "command": "loomux hook subagent-start --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "SubagentStop": [{"hooks": [{"type": "command", "timeout": 30,
      "command": "loomux hook subagent-stop --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}]
  }
}
```
Versucht Claude Code ein `Write`, `Edit` oder einen Shell-Befehl auszuführen, prüft der Wächter ihn gegen die Regeln, innerhalb eines Budgets von **35 ms**. Verweigert eine Regel die Aktion, wird sie mit Exit-Code 2 blockiert und die Begründung erscheint direkt im Chat des Agenten.

### Google Antigravity
Für Antigravity werden die Hooks in `.agents/hooks.json` eingetragen:
```json
{
  "loomux": {
    "PreInvocation": [{"type": "command", "command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook session-start --host antigravity --root ..", "timeout": 20}],
    "PreToolUse": [{"matcher": "write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file|run_command|send_command_input|manage_task", "hooks": [{"type": "command", "command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook pre-tool-use --host antigravity --root ..", "timeout": 15}]}],
    "PostToolUse": [{"matcher": "write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file", "hooks": [{"type": "command", "command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook post-tool-use --host antigravity --root ..", "timeout": 60}]}],
    "Stop": [{"type": "command", "command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook stop --host antigravity --root .. --budget 270s", "timeout": 300}]
  }
}
```
`loomux init` schreibt diese Gruppe; agy führt sie über `cmd.exe` aus `.agents/` aus, daher `%LOCALAPPDATA%` und `--root ..`. Nach einem Upgrade `loomux init` erneut ausführen: Ein eigener Eintrag unter einem älteren Matcher bleibt, wie er ist, und `init` hängt einen Block für die fehlenden Werkzeuge an, etwa `manage_task`. Wie die Hooks Antigravity antworten, steht in der CLI-Referenz.

### Cursor & MCP-Clients
Starte den lokalen MCP-Dienst:
```bash
loomux serve
```
Oder binde die stdio-Brücke direkt in deine Cursor MCP-Konfiguration ein:
```json
{
  "mcpServers": {
    "loomux": {
      "command": "C:/Users/<du>/AppData/Local/loomux/bin/loomux.exe",
      "args": ["mcp", "--channel", "local"]
    }
  }
}
```
Für Claude Code schreibt `loomux init` diesen Eintrag in `.mcp.json`, und zwar
als `${LOCALAPPDATA}/loomux/bin/loomux.exe`, das Claude Code auflöst; für einen
anderen Client den Pfad in der Form angeben, die er liest.

---

## 5. Einen Bereich von Hand einrichten

Ein Bereich besteht aus zwei Erklärungen: einem Eintrag in der Registry der
Maschine, der sagt, wo der Baum liegt und was darin geschrieben werden darf,
und einem Manifest im Baum, das sagt, was er ist. `loomux area add` schreibt
beide, und `loomux init` ruft es als seinen Teil `area`; dieser Abschnitt ist
für einen Menschen, der sie von Hand schreibt. Die
Schreibschranke verweigert einem Agenten `.loomux/config.toml`. Die
Registry schützt sie vor den Dateiwerkzeugen des Hosts nur, solange kein
registrierter Baum das Zustandsverzeichnis umschließt, und Shell-Zeilen oder die
Dateiwerkzeuge von MCP-Servern prüft sie nicht dagegen. Das alles gilt innerhalb
der Grenzen, die unter
[Hooks](hooks.md#7-der-entscheidungsweg-von-pre-tool-use) aufgelistet sind. Wo beide Dateien
liegen, steht unter
[Wo was liegt](configuration.md#4-wo-was-liegt).

### Der Registry-Eintrag
`%LOCALAPPDATA%\loomux\registry.toml`, eine `[[area]]`-Tabelle je Bereich:

```toml
[[area]]
scope     = "project/mein-projekt"
path      = "C:/Users/ich/Documents/GIT/mein-projekt"
wiki      = "C:/Users/ich/Documents/GIT/mein-projekt/docs/wiki"
workspace = true
```

- `scope` und `path` sind Pflicht und müssen nicht leere Zeichenketten sein.
- `workspace = true` öffnet den ganzen `path` für schreibende Werkzeuge.
- `wiki` öffnet das Bündel, das es nennt. Ein beschreibbarer Bereich, dessen
  Manifest `[layout] wiki` erklärt, bekommt dieses Bündel auch ohne den
  Schlüssel geöffnet — sofern das Manifest im registrierten `path` (oder einem
  verknüpften Worktree davon) liegt und sein Scope passt.
- `readonly = true` macht das `wiki` des Bereichs zur Verbotszone: dort
  verweigert die Schranke jeden Schreibaufruf, auch wenn ein beschreibbarer
  Bereich das Verzeichnis umschließt. Einen `workspace`-Baum schließt das nicht;
  das ist eine eigene Frage. Das Manifest eines lesenden Bereichs wird aus dem
  Zustandsverzeichnis gelesen, nicht aus seinem Baum.
- Höchstens ein Bereich darf `signpost = true` setzen.

### Das Manifest
`.loomux/config.toml` im Wurzelverzeichnis des Bereichs, neben den
Policy-Regeln:

```toml
[area]
scope = "project/mein-projekt"

[layout]
wiki = "docs/wiki"

[index]
include    = ["**/*.md"]
exclude    = [".venv/**", "node_modules/**"]
unsearched = ["docs/.superpowers/**"]

[privacy]
mode  = "manual_cloud"
never = ["privat/**"]
```

- **`[area] scope`** nennt den Registry-Eintrag. Eine `.loomux/config.toml` ohne
  `[area]`-Tabelle ist nur Policy und erklärt keinen Bereich.
- **`[layout] wiki`** ist relativ zum Repowurzelverzeichnis, mit
  Schrägstrichen; es darf das Repo weder verlassen noch seine Wurzel nennen. Es
  ist einer von mehreren Wegen, die Wiki-Lane des post-edit-Hooks zu starten:
  die Lane startet auch, wenn das Manifest eine `[wiki]`-Tabelle oder
  `wiki = true` trägt oder wenn `index.md` oder `bundle.toml` in `wiki/` (oder,
  wo das fehlt oder leer ist, in `docs/wiki/`) `okf_version` enthält. Das
  Bündel nimmt der Hook aus diesem Schlüssel, wenn er ein vorhandenes
  Verzeichnis nennt, sonst `docs/wiki`, sonst `wiki`, sonst ein Wiki neben dem
  Repo.
- **`[index] include`** — die Suchmaschine kennt ein Muster je Sammlung und
  sieht nur den ersten Glob; `loomux brain status` nennt die übrigen.
- **`[index] exclude`** nennt Pfade, die `loomux reindex` nicht indexiert.
- **`[index] unsearched`** erklärt, was lesbar ist, aber nie gesucht wird. qmd
  betritt keine Punktverzeichnisse; `docs/.superpowers/**` ist also über
  `brain read` und `brain neighbors` erreichbar, über `brain search` nicht.
  Deklariert, zählt `brain status` diese Dateien nicht mehr als fehlend.
- **`[privacy] mode`** ist `local_only`, `manual_cloud` (Vorgabe) oder
  `automatic_cloud`; jeder andere Wert wird abgelehnt. Auf dem Cloud-Kanal
  existiert ein `local_only`-Bereich nicht: keine Treffer, keine Inhalte, und
  sein Scope gilt als unbekannt. Verschachtelung hebt das nicht auf: Sein Wiki
  im Baum eines anderen Bereichs bleibt auch über dessen Scope verborgen.
- **`[privacy] never`** nennt Pfade, die kein Kanal erreicht.

### Was schiefgeht
- **Jeder `loomux brain`-Befehl scheitert mit `no manifest found`.** Die Befehle
  lesen das Manifest jedes registrierten Bereichs, bevor sie `--scope` ansehen;
  ein Bereich ohne Erklärung lässt also auch Aufrufe über alle anderen
  scheitern; nur ein Eintrag mit `workspace = true` ohne Erklärung wird still
  übersprungen. Eine `.loomux/config.toml` ohne `[area]` zählt als keine, und ebenso
  ein Bereich, dessen Verzeichnis keine `.loomux/config.toml` hat, egal, was sonst
  dort liegt. Eine `.loomux/config.toml`, die sich nicht prüfen lässt
  (etwa fehlende Rechte), zählt dagegen nicht als fehlend: Der Aufruf scheitert,
  auch bei einem Workspace.
  Die Schreibschranke stört das nicht: dort ist es normal, einen Bereich vor
  seinem Manifest zu registrieren.
- **Jeder Schreibaufruf wird mit `loomux cannot read the registry, so it
  refuses` abgelehnt.** Die Schranke liest die Registry streng: ein fehlender
  `scope` oder `path`, ein doppelter Scope, zwei Scopes, die auf denselben
  Namen im Zustandsverzeichnis fallen, ein zweites `signpost`, `[area]` statt
  `[[area]]` oder das Manifest eines registrierten Bereichs, das seine eigenen
  Prüfungen nicht besteht, schließt jeden Baum. Offen bleiben nur das Memory
  der Agenten, das Scratchpad der Sitzung und die Dateien, die `open.toml`
  nennt. Die `brain`-Befehle sind
  nachsichtiger und überspringen einen Eintrag ohne `scope` oder `path`
  wortlos.
- **Seiten unter einem Punktverzeichnis werden nie gefunden.** Das ist eine
  Grenze der Suchmaschine, kein Fehler; sie gehören in `unsearched`.

---

## 6. Nächste Schritte

- **[loomux installieren](install.md)**: Plattformen, Aktualisierung und Kanäle, Bau aus dem Quellcode.
- **[Konfigurations-Referenz](configuration.md)**: Vollständige Übersicht aller `.loomux/config.toml`-Sektionen (`[modules]`, `[policy]`, `[verify]`, `[worktree]`).
- **[CLI-Befehlsreferenz](cli-reference.md)**: Das komplette Handbuch aller Befehle, Flags und Exit-Codes.
- **[Architektur & Konzepte](architecture.md)**: Erfahre mehr über Karpathys LLM OS, Googles Knowledge Items und AST-GraphRank.
- **[Hook-Lebenszyklus](hooks.md)**: Details zur Sub-35ms Schreibschranke, zum Blast-Radius-Monitor und zum Event-Stream.

← [loomux installieren](install.md) · [Übersicht](README.md) · [Konfiguration](configuration.md) →
