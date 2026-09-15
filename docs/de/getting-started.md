# Erste Schritte mit Loomux

Dieser Leitfaden führt dich in unter 3 Minuten durch die Installation von Loomux, die Initialisierung eines Repositories und die Anbindung von Loomux an deine KI-Coding-Agenten (**Claude Code**, **Google Antigravity** und **Cursor**).

---

## 1. Voraussetzungen

- **Go**: Version 1.25 oder neuer (zum Bauen aus dem Quellcode oder via `go install`).
- **Git**: Version 2.30 oder neuer.
- **Betriebssystem**:
  - **Windows (x64)**: Erstklassige native Unterstützung (NTFS-Junctions, Windows Job Objects).
  - **Linux / macOS**: Vollständig unterstützt (Symlinks, POSIX-Prozessgruppen).

---

## 2. Installation

### Option A: Installation via Go (Empfohlen)
```bash
go install github.com/xidus90/loomux/cmd/loomux@latest
```
Stelle sicher, dass `$GOPATH/bin` (oder `%USERPROFILE%\go\bin`) in deiner System-Umgebungsvariable `PATH` enthalten ist.

### Option B: Aus dem Quellcode bauen
```bash
git clone https://github.com/xidus90/loomux.git
cd loomux
go build -o bin/loomux.exe ./cmd/loomux
```

Installation überprüfen:
```bash
loomux --version
# Ausgabe: loomux 0.1.0-fusion
```

---

## 3. Schnellstart in 3 Schritten

### Schritt 1: Repository initialisieren
Wechsle in das Wurzelverzeichnis deines Projekts und führe aus:
```bash
loomux init
```
Dieser Befehl:
1. Erkennt vorhandene Coding-Agenten in deinem Arbeitsbereich (`.claude/`, `.agents/`, `.cursor/`).
2. Erstellt das Konfigurationsverzeichnis `.loomux/`.
3. Erzeugt eine Starter-Konfiguration `.loomux/config.toml` (mit Schreibschranken und Prüfketten).
4. Richtet die Agenten-Hook-Dateien so ein, dass sie `loomux hook` aufrufen.

> [!NOTE]
> Mit dem Flag `--dry-run` siehst du vorab, welche Dateien berührt würden, ohne Änderungen vorzunehmen:
> `loomux init --dry-run`

### Schritt 2: Konfiguration anpassen
Öffne `.loomux/config.toml`. Eine minimale Starter-Konfiguration sieht so aus:

```toml
# .loomux/config.toml

[project]
name = "mein-projekt"

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

[verify]
# Prüfkette, die am Rundenende automatisch ausgeführt wird
lanes = [
  { name = "lint", command = "golangci-lint run" },
  { name = "test", command = "go test -v ./..." }
]
```

### Schritt 3: Einrichtung verifizieren
Starte die Diagnose:
```bash
loomux status
```
Die Ausgabe zeigt dir den Status aller Schutz- und Prüfmechanismen:
```text
=== loomux Hook Inspection ===
Projektwurzel:    C:\Projekte\mein-projekt
Erkannte Hosts:   Claude Code, Antigravity
Hook-Status:      PreToolUse (Aktiv), PostToolUse (Aktiv), Stop (Aktiv)
Prüfketten-Lanes: lint (golangci-lint), test (go test)
Schreibschranke:  Aktiv (Globale Registry + Projekt-Policy)
Gesamtstatus:     BEREIT (Grün)
```

> [!NOTE]
> Die Schreibschranke öffnet nur die Bäume, die in der globalen Registry stehen
> (`%LOCALAPPDATA%\loomux\registry.toml`). Ein verknüpfter Git-Worktree eines Repos,
> das mit `workspace = true` registriert ist, zählt zu diesem Repo und braucht keinen
> eigenen Eintrag; die Schranke liest dafür Gits Worktree-Dateien und startet keinen
> `git`-Prozess. Ein ohne `git worktree repair` verschobener Worktree bleibt gesperrt.

---

## 4. Anbindung an Agenten-Harnesses

Loomux integriert sich nahtlos in alle gängigen Agenten-Umgebungen:

### Claude Code
`loomux init` registriert die Hooks automatisch in `.claude/settings.json`:
```json
{
  "hooks": {
    "PreToolUse": "loomux hook pre-tool-use --host claude",
    "PostToolUse": "loomux hook post-tool-use --host claude",
    "SessionStart": "loomux hook session-start --host claude",
    "Stop": "loomux hook stop --host claude"
  }
}
```
Sobald Claude Code versucht, ein `Write`, `Edit` oder einen Shell-Befehl auszuführen, prüft Loomux die Regeln in **unter 35 ms**. Bei einem Verstoß wird die Aktion mit Exit-Code 2 blockiert und die Begründung erscheint direkt im Chat des Agenten.

### Google Antigravity
Für Antigravity werden die Hooks in `.agents/hooks.json` eingetragen:
```json
{
  "hooks": [
    {
      "event": "PreToolUse",
      "command": "loomux hook pre-tool-use --host antigravity"
    },
    {
      "event": "PostToolUse",
      "command": "loomux hook post-tool-use --host antigravity"
    }
  ]
}
```

### Cursor & MCP-Clients
Starte den lokalen MCP-Dienst:
```bash
loomux serve --port 8080
```
Oder binde die stdio-Brücke direkt in deine Cursor MCP-Konfiguration ein:
```json
{
  "mcpServers": {
    "loomux": {
      "command": "loomux",
      "args": ["mcp"]
    }
  }
}
```

---

## 5. Nächste Schritte

- **[Konfigurations-Referenz](configuration.md)**: Vollständige Übersicht aller `.loomux/config.toml`-Sektionen (`[policy]`, `[verify]`, `[worktree]`, `[graph]`).
- **[CLI-Befehlsreferenz](cli-reference.md)**: Das komplette Handbuch aller Befehle, Flags und Exit-Codes.
- **[Architektur & Konzepte](architecture.md)**: Erfahre mehr über Karpathys LLM OS, Googles Knowledge Items und Grafts GraphRank.
- **[Hook-Lebenszyklus](hooks.md)**: Details zur Sub-35ms Schreibschranke, zum Blast-Radius-Monitor und zum Event-Stream.
