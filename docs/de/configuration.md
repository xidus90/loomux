# Loomux Konfigurations-Referenz

Dieses Dokument bietet eine vollständige Referenz für `.loomux/config.toml`, die zentrale Konfigurationsdatei für Loomux-Projekte.

---

## 1. Grundsätze der Konfiguration

1. **Vom Menschen gepflegt, vom Wächter geschützt**:
   > [!IMPORTANT]
   > `.loomux/config.toml` wird **niemals von einem KI-Agenten bearbeitet**. Die Schreibschranke blockiert jeden Schreibversuch eines Agenten auf `.loomux/config.toml`. Änderungen werden vorgeschlagen; ein Mensch schreibt und committet sie.
2. **Deterministisch & Strikt**:
   Alle regulären Ausdrücke und Pfad-Globs werden beim ersten Gebrauch kompiliert. Enthält eine Regel einen ungültigen Regex oder fehlt eine Begründung (`reason`), bricht Loomux sofort mit einer präzisen Fehlermeldung unter Nennung der exakten Zeile ab.
3. **Trennung von Konfiguration und Zustand**:
   - `.loomux/config.toml`: Versionierte, von Menschen definierte Richtlinien und Prüfketten.
   - `.loomux/state/`: Flüchtiger, maschinengeschriebener Zustand (Sitzungsdaten, Journal, Caches). Stets in `.gitignore`.

---

## 2. Die Konfigurations-Sektionen

### `[project]`
Metadaten zur Beschreibung des Projekts.

```toml
[project]
name = "loomux"
version = "0.1.0"
agents = ["claude", "antigravity", "cursor"]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `name` | String | Projekt-Bezeichner für Namensräume und Registrierungen. |
| `version` | String | Optionale Versionsnummer. |
| `agents` | Array von Strings | Aktive Agenten-Harnesses, die durch `loomux init` angebunden werden. |

---

### `[policy.paths]` (Pfad-Schutzregeln)
Definiert Pfadmuster, die Coding-Agenten weder erstellen noch bearbeiten dürfen.

```toml
[policy.paths]
rules = [
  { match = [".env", ".env.*"], reason = "Secrets und Umgebungsdateien dürfen nicht von Agenten bearbeitet werden" },
  { match = ["*.pem", "*.key", "id_rsa*"], reason = "Private Schlüssel und Zertifikate werden vom Menschen verwaltet" },
  { match = [".loomux/config.toml"], reason = "Die Wächter-Konfiguration ist vor Agenten-Schreibzugriffen geschützt" },
  { match = ["package-lock.json", "go.sum", "uv.lock"], reason = "Lockfiles werden durch Paketmanager verwaltet, nicht manuell" }
]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `rules` | Array von Tabellen | Liste der Pfad-Inspektionsregeln. |
| `rules[].match` | String oder Array von Strings | Glob-Muster mit `**`-Unterstützung (z. B. `.aws/**`, `*.key`). |
| `rules[].reason` | String (**Pflichtfeld**) | Begründung, die dem Agenten bei einer Ablehnung angezeigt wird. |

---

### `[policy.commands]` (Befehlsausführungs-Regeln)
Definiert Muster für Shell-Befehle (`Bash`, `PowerShell`), die blockiert werden müssen.

```toml
[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Commits zum Remote zu pushen erfordert eine menschliche Entscheidung" },
  { regex = '(^|\s)rm\s+-rf\s+/', reason = "Zerstörerische Wurzelverzeichnis-Löschbefehle sind verboten" },
  { regex = '(^|\s)pip\s+install\s+-r', reason = "Abhängigkeiten müssen über Lockfiles und Workflows verwaltet werden" }
]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `rules` | Array von Tabellen | Liste der Befehls-Inspektionsregeln. |
| `rules[].regex` | String (**Pflichtfeld**) | Go RE2-kompatibler regulärer Ausdruck. |
| `rules[].reason` | String (**Pflichtfeld**) | Begründung, die dem Agenten bei einer Ablehnung angezeigt wird. |

---

### `[verify]` (Prüfketten & Quality-Gates)
Definiert die Prüftabelle, die am Rundenende (`Stop`-Hook) oder vor dem Commit ausgeführt wird.

```toml
[verify]
timeout = 300 # Sekunden

lanes = [
  { name = "format", command = "gofmt -l cmd internal" },
  { name = "vet", command = "go vet ./..." },
  { name = "test", command = "go test -v ./..." },
  { name = "covergate", command = "loomux dev covergate --profile coverage.out" }
]

[verify.commit]
language = "en"       # Erzwingt englische Commit-Nachrichten
max_first_line = 72   # Maximale Länge der ersten Titelzeile
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `timeout` | Integer | Globales Zeitlimit in Sekunden für die Prüfkette (Standard: 300). |
| `lanes` | Array von Tabellen | Einzelne Prüfschritte, die ausgeführt werden. |
| `lanes[].name` | String | Eindeutiger Name der Prüf-Lane. |
| `lanes[].command` | String | Auszuführender Shell-Befehl in der Projektwurzel. |
| `commit.language` | String | Erzwungene Sprache für Commit-Nachrichten (`"en"`). |
| `commit.max_first_line` | Integer | Maximale Zeichenlänge der ersten Zeile. |

---

### `[worktree]` (Subagent-Isolierungsspiegel)
Konfiguriert Verzeichnisse, die automatisch per NTFS-Junction (Windows) oder Symlink (POSIX) in isolierte Subagent-Worktrees gespiegelt werden.

```toml
[worktree]
mirrors = [
  "node_modules",
  ".cache",
  "vendor",
  "testdata/large-fixtures"
]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `mirrors` | Array von Strings | Pfade aus dem Haupt-Checkout, die gespiegelt werden, um teure Neu-Downloads zu vermeiden. |

---

### `[graph]` (Code-Graph-Einstellungen)
Konfiguriert AST-Extraktion, Indizierungsgrenzen und Frischeprüfung.

```toml
[graph]
extensions = [".go", ".ts", ".tsx", ".py", ".rs"]
exclude = ["vendor/**", "dist/**", "node_modules/**", "**/*_test.go"]
freshness_check = "hash" # "mtime" oder "hash"
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `extensions` | Array von Strings | Dateiendungen, die in den AST-Code-Graphen aufgenommen werden. |
| `exclude` | Array von Strings | Glob-Muster, die bei der Graph-Erstellung ignoriert werden. |
| `freshness_check` | String | Strategie für Arbeitsbaum-Drift-Erkennung: `"mtime"` (<1ms) oder `"hash"` (<3ms, bitgenau). |

---

### `[skills]` (Kuratierte Best-Practice-Suiten)
Konfiguriert sprachspezifische Review-Regeln und Synchronisationsziele.

```toml
[skills]
suites = ["review-go", "review-security", "review-typescript"]
sync = [".claude/skills", ".agents/skills"]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `suites` | Array von Strings | Aktive Review-Suiten, die im Loomux-Binary mitgeliefert werden. |
| `sync` | Array von Strings | Zielordner, in die `SKILL.md`-Dateien abgelegt werden. |

---

### `[privacy]` (Datenschutz & Geheimnisschutz)
Garantiert, dass sensible Daten unter keinen Umständen an externe Modelle oder Logs weitergegeben werden.

```toml
[privacy]
mode = "manual_cloud"
never = [
  "**/.env*",
  "**/credentials.json",
  "**/*.pem",
  "**/*.key"
]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `mode` | String | `"local_only"`, `"manual_cloud"` (Vorgabe) oder `"automatic_cloud"`; jeder andere Wert wird abgelehnt. Ein `local_only`-Bereich existiert auf dem Cloud-Kanal nicht. |
| `never` | Array von Strings | Glob-Muster der Pfade, die kein Kanal erreicht. |

---

## 3. Vollständiges kommentiertes Muster (`config.toml`)

```toml
# ==============================================================================
# Loomux Projektkonfiguration: .loomux/config.toml
# ==============================================================================

[project]
name = "loomux"
version = "0.1.0-fusion"
agents = ["claude", "antigravity", "cursor"]

# --- Pfad-Schutzregeln (Exit-Code 2 bei Verstoß) -----------------------------
[policy.paths]
rules = [
  { match = [".env*", "*.pem", "*.key", "id_rsa*"], reason = "Secrets dürfen nicht von Agenten bearbeitet werden" },
  { match = [".loomux/config.toml"], reason = "Wächter-Regeln werden vom Menschen verwaltet und sind schreibgeschützt" },
  { match = [".claude/.no-verify"], reason = "Prüfschranken dürfen nicht durch Agenten umgangen werden" },
  { match = ["go.sum", "package-lock.json", "uv.lock"], reason = "Lockfiles werden durch Paketmanager verwaltet, nicht manuell" }
]

# --- Befehlsausführungs-Regeln (Exit-Code 2 bei Verstoß) --------------------
[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Pushen zum Remote erfordert eine menschliche Entscheidung" },
  { regex = '(^|\s)rm\s+-rf\s+/', reason = "Löschung des System-Wurzelverzeichnisses ist verboten" }
]

# --- Prüfkette für das Rundenende --------------------------------------------
[verify]
timeout = 180

lanes = [
  { name = "gofmt", command = "gofmt -l cmd internal" },
  { name = "vet", command = "go vet ./..." },
  { name = "tests", command = "go test ./..." },
  { name = "covergate", command = "go run ./cmd/loomux dev covergate --profile coverage.out" }
]

[verify.commit]
language = "en"
max_first_line = 72

# --- Worktree-Isolierungsspiegel ---------------------------------------------
[worktree]
mirrors = [
  "node_modules",
  ".cache",
  "bin"
]

# --- Code-Graph & AST-Einstellungen ------------------------------------------
[graph]
extensions = [".go", ".ts", ".py"]
exclude = ["vendor/**", "dist/**"]
freshness_check = "hash"

# --- Kuratierte Sprach-Review-Suiten -----------------------------------------
[skills]
suites = ["review-go", "review-security"]
sync = [".claude/skills", ".agents/skills"]

# --- Datenschutz-Schranken ---------------------------------------------------
[privacy]
mode = "manual_cloud"
never = [".env*", "*.key", "credentials.json"]
```

---

## 4. Wo was liegt

| | Ort |
|---|---|
| Zustandsverzeichnis | unter Windows `%LOCALAPPDATA%\loomux`, sonst `~\AppData\Local\loomux`; auf anderen Systemen `$XDG_STATE_HOME/loomux`, sonst `~/.local/state/loomux` |
| Bereichsregistry | `<Zustandsverzeichnis>\registry.toml` |
| Manifest eines beschreibbaren Bereichs | `<Bereichspfad>\.loomux\config.toml` |
| Manifest eines lesenden Bereichs, wie die Schreibschranke es liest | `<Zustandsverzeichnis>\areas\<scope>\.loomux\config.toml` |
| Artefakte eines lesenden Bereichs (`index.md`, `graph.json`, `_identities.tsv`) und sein Manifest, wie `loomux brain` sie liest | `%LOCALAPPDATA%\brain\areas\<scope>\` bis Stufe 3 |
| Artefakte eines beschreibbaren Bereichs | sein `path` |
| Stempel des letzten Reconcile | `%LOCALAPPDATA%\brain\maintenance\last-run.txt` bis Stufe 3 |

`LOOMUX_STATE_DIR` überschreibt das Zustandsverzeichnis,
`LOOMUX_LEGACY_BRAIN_DIR` das Verzeichnis von ultra-brain; einen
Kommandozeilenschalter gibt es für keines von beiden. Das Altverzeichnis gibt
es, weil ultra-brain diese Artefakte noch schreibt: vor Stufe 3 hat loomux
keinen eigenen Indexer.

`<scope>` ist der Scope, zu einem Verzeichnisnamen geplättet: jede Folge von
Zeichen außer `A-Z a-z 0-9 _ . -` wird zu einem `-`, und Striche an beiden Enden
fallen weg; aus `project/loomux` wird `project-loomux`.

**Das Wissen liegt an keinem dieser Orte.** Es liegt dort, wohin `path` und
`wiki` eines Bereichs zeigen — in einem Repo oder in einem gewöhnlichen
Obsidian-Vault unter Git. loomux hält keine Kopie davon.

---

## 5. Die Schreibschranke und das Memory der Agenten

Die Schreibschranke (`loomux hook pre-tool-use`) lässt Werkzeuge schreiben, wo
die Registry einen Bereich erklärt, und darüber hinaus immer an drei Orten —
für jeden Nutzer, ohne Eintrag in der Registry:

| Wo | offen ist, unterhalb von |
|---|---|
| Memory von Claude Code | `$CLAUDE_CONFIG_DIR/projects/<projekt>/memory/`, sonst `~/.claude/projects/<projekt>/memory/` |
| Scratchpad einer Claude-Code-Sitzung | `<temp>/claude/<projekt>/<sitzung>/scratchpad/` |
| Memory von Antigravity | `~/.gemini/{antigravity,antigravity-cli,antigravity-ide}/knowledge/` und `…/brain/<gesprächs-id>/` |

- Ein Aufruf, der nur dorthin schreibt, geht auch durch, wenn die Registry
  nicht lesbar ist.
- `.loomux/config.toml` bleibt auch dort gesperrt; über das Manifest wird vor
  allem anderen entschieden.
- Jeder Pfad wird vor dem Vergleich aufgelöst; ein Link aus dem Memory in einen
  gesperrten Baum wird also am Ziel beurteilt. In einem Aufruf, der auch
  anderswohin schreibt, bleibt ein Memory-Ziel in einer Verbotszone gesperrt.
- Den Rest von `~/.claude` und `~/.gemini` öffnet die Ausnahme nicht,
  `antigravity-backup` eingeschlossen: `settings.json`, Hooks und Plugins
  steuern den Agenten selbst, und wer sie umschreibt, schaltet seine eigenen
  Schranken ab.
- Claude Code lädt `MEMORY.md` in spätere Sitzungen desselben Projekts — was
  ein Agent dort schreibt, wirkt also wie eine Anweisung an künftige Sitzungen;
  ob Antigravity `knowledge/` ebenso lädt, ist nicht gemessen. Das ist mit der
  Entscheidung vom 2026-09-13 bewusst in Kauf genommen.

Eine Ablehnung mit `lies outside every writable tree` nennt, wo geschrieben
werden darf: die erlaubten Bäume, danach das Scratchpad der Sitzung und die
Memory-Bäume, sofern sie sich benennen lassen.
