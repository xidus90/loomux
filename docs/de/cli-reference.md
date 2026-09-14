# Loomux CLI-Referenzhandbuch

Dieses Handbuch dokumentiert Syntax, Flags, Standard-I/O-Verträge und Exit-Codes aller Loomux-Kommandozeilenbefehle.

---

## 1. Globale Konventionen

### Exit-Codes
Loomux nutzt eine strikte Exit-Code-Semantik, die exakt auf die Schnittstellen moderner Coding-Agenten abgestimmt ist:

| Exit-Code | Bedeutung | Verhalten im Harness (Claude / Antigravity) |
|---|---|---|
| **`0`** | **Erfolg / Erlaubt** | Die Werkzeugausführung wird fortgesetzt; die Runde ist erfolgreich. |
| **`1`** | **Mitteilung / Warnung** | Vom Harness als rein informativ bzw. „weitermachen“ interpretiert. |
| **`2`** | **Strikte Ablehnung / Blockiert** | Die Schreibschranke oder Policy hat die Aktion blockiert; Abbruch des Aufrufs. |

### Globale Flags & Umgebung
- `--root <pfad>`: Explizite Angabe der Projektwurzel. Wird dieses Flag weggelassen, wandert Loomux im Verzeichnisbaum aufwärts, bis es die erste `.loomux/config.toml` findet.
- `LOOMUX_STATE_DIR`: Überschreibt das globale Zustandsverzeichnis (Standard: `%LOCALAPPDATA%\loomux` unter Windows, `~/.local/state/loomux` unter POSIX).

---

## 2. Policy & Prüfketten (`loomux check`)

### `loomux check commit-msg <datei>`
Prüft eine Git-Commit-Nachricht auf englische Sprache und Einhaltung der Formatregeln.

- **Argumente**: `<datei>` — Pfad zur Commit-Nachrichtendatei (`COMMIT_EDITMSG`).
- **Verhalten**:
  - Erzwingt englische Sprache für Titel und Textkörper.
  - Verwirft Konversations-Präambeln (z. B. *„Sure, I'll commit that...“*).
  - Validiert die Länge der ersten Betreffzeile.
- **Exit-Codes**: `0` (Gültig), `1` (Ungültige Commit-Nachricht mit Begründung auf `stderr`).

### `loomux check gofmt [pfade...]`
Überprüft Go-Quelldateien auf Formatierungskonformität, ohne sie zu verändern.

- **Argumente**: Optionale Verzeichnisse oder Dateipfade (Standard: Arbeitsverzeichnis).
- **Exit-Codes**: `0` (Korrekt formatiert), `1` (Unformatierte Dateien auf `stdout` gelistet).

---

## 3. Agenten-Harness-Hooks (`loomux hook`)

Hook-Einstiegspunkte werden von Coding-Agenten synchron bei Werkzeugaufrufen gestartet.

```bash
loomux hook <event> --host <claude|antigravity|codex> [--root <pfad>]
```

### `loomux hook pre-tool-use`
Prüft Projekt-Policy und globale Schreibschranke, bevor der Agent ein Werkzeug ausführt.

- **Standard-Input (stdin)**: JSON-Nutzlast des aufrufenden Agenten:
  ```json
  {
    "tool_name": "Write",
    "tool_input": {
      "file_path": "C:/Projekte/repo/.env"
    }
  }
  ```
- **Laufzeit-Budget**: `<35ms` Kaltstart-Boden.
- **Standard-Output / Fehler**:
  - Bei Ablehnung: JSON-Ablehnungs-Umschlag auf `stdout`, Begründung auf `stderr`.
- **Exit-Codes**:
  - `0`: Gestattet.
  - `2`: Verweigert (Policy-Verletzung oder Schreibzugriff außerhalb registrierter Bereiche).

### `loomux hook post-tool-use`
Wird unmittelbar nach Abschluss einer Dateiänderung oder eines Shell-Befehls ausgeführt.

- **Standard-Input (stdin)**: Name des Werkzeugs und Eingabe-Payload.
- **Verhalten**:
  - Berechnet Hash geänderter Dateien.
  - Ermittelt sofort den Blast-Radius betroffener Symbole.
  - Gibt Warnungen inline aus, falls kritische Aufrufer berührt wurden.
- **Exit-Codes**: Immer `0` (blockiert niemals das Rundenende).

### `loomux hook session-start`
Kündigt den Beginn einer Sitzung an und synchronisiert die Host-Umgebung.

- **Flags**: `--host <h>` (Pflichtfeld), `--root <r>`.
- **Verhalten**:
  - Prüft Arbeitsbaum-Sauberkeit und Frische.
  - Richtet bei Subagenten-Sitzungen isolierte Worktree-Junction-Spiegel ein.
- **Exit-Codes**: `0` (Erfolg), `1` (Fehlendes Flag oder ungültiger Host).

---

## 4. Diagnose-Doktor (`loomux status`)

### `loomux status [--root <pfad>]`
Inspiziert den Zustand des aktuellen Repositories, der Prüfketten, erkannten Agenten und Hooks.

```bash
loomux status
```
- **Aliase**: `loomux explain`, `loomux doctor`.
- **Ausgabe-Details**:
  - Pfad der Projektwurzel und deklarierte Bereiche.
  - Aktive Agenten-Harnesses (`.claude/`, `.agents/`, `.cursor/`).
  - Status der Schreibschranke und Anzahl der aktiven Policy-Regeln.
  - Konfigurierte `[verify]`-Lanes.
- **Exit-Codes**: `0` (Bereit), `1` (Konfigurationsfehler).

---

## 5. Worktree-Spiegelung (`loomux worktree`)

Verwaltet isolierte Git-Worktrees für Subagenten mit gespiegelten Abhängigkeiten.

### `loomux worktree link [--root <pfad>]`
Spiegelt konfigurierte Verzeichnisse (`node_modules`, `.cache`) per NTFS-Junction (Windows) oder Symlink (POSIX) in den aktiven Worktree.

### `loomux worktree unlink [--root <pfad>]`
Entfernt Junction-Spiegel nach Beendigung der Subagenten-Sitzung sicher, ohne Dateien im Haupt-Repository zu berühren.

### `loomux worktree remove <worktree-pfad>`
Löscht einen isolierten Worktree-Pfad vollständig und bereinigt alle Verknüpfungen.

- **Exit-Codes**: `0` (Sauber entfernt), `1` (Pfad kann nicht geprüft werden oder Git verweigert Löschung).

---

## 6. Code-Graph-Engine (`loomux graph`)

### `loomux graph build [dir]`
Parst Quellcodedateien in den deterministischen AST-Code-Graphen und schreibt `.loomux/state/graph/wiring.json`.

- **Flags**:
  - `--deep`: Reichert Symbole mit LLM-Crux-Zusammenfassungen an (gecached).
  - `--extensions <exts>`: Beschränkt die zu verarbeitenden Dateiendungen (z. B. `.go .ts`).

### `loomux graph ask "<anfrage>" [dir]`
Sucht Code-Symbole gerankt nach **Personalized PageRank** über den AST-Aufrufgraph.

- **Ausgabe**: Gerankte Symbole mit Datei, Zeilen und inline eingeblendeten Crux-Spans ($0 Token-Lesekosten).
- **Flags**: `--json` (maschinenlesbare Ausgabe).

### `loomux graph callers <symbol> [dir]`
Zeigt, wer ein Symbol aufruft, importiert, implementiert oder erweitert.

- **Flags**:
  - `--direction out`: Umgekehrte Richtung — was das Symbol selbst aufruft.
  - `-d <tiefe>`: Transitive Tiefe (`-d all` für die vollständige transitive Hülle).

### `loomux graph blast [dir]`
Berechnet den Blast-Radius eines Git-Diffs gegen den Working Tree oder Merge-Base.

- **Flags**:
  - `--base <ref>`: Diff gegen Git-Referenz (z. B. `origin/main`).
  - `--format markdown`: Formatiert die Ausgabe als fertigen GitHub-PR-Kommentar.
  - `--export-viz <dir>`: Exportiert eine interaktive HTML-Visualisierung des Blast-Radius.

### `loomux graph skeleton <datei>`
Gibt alle Funktions-, Typ-, Interface- und Methodensignaturen ohne Rümpfe aus (~10x Token-Ersparnis).

### `loomux graph map [dir]`
Zeigt Verzeichnis-Cluster, lokale Hubs und globale Codebasis-Hotspots gerankt nach Kanten-Kopplung.

### `loomux graph check [dir]`
Prüft, ob der Code-Graph gegenüber dem Live-Arbeitsbaum veraltet ist.
- **Exit-Codes**: `0` (Frisch), `1` (Veraltet / Drift erkannt).

### `loomux graph viz [dir]`
Startet die lokale interaktive D3-Force / WebGL Graph-Visualisierung im Browser.
- **Flags**: `--port <p>`, `--no-open`.

---

## 7. Second Brain & Wiki (`loomux brain`)

### `loomux brain search "<anfrage>"`
Führt hybride semantische und Keyword-Suche über Wiki-Seiten, ADRs und Konzepte aus.

### `loomux brain catalog`
Listet alle verwalteten Wiki-Bereiche, Themen-Hierarchien und Identitäten auf.

### `loomux brain read <pfad>`
Liest eine typisierte Wiki-Seite, eine Konzept-Definition oder ein ADR.

### `loomux brain lint`
Validiert Wiki-Links (`[[Seite]]`), verwaiste Dokumente, tote Referenzen und Frontmatter-Taxonomien.

### `loomux brain reconcile`
Synchronisiert Zustandsänderungen, Identitätsregister und Vektorindex-Sammlungen.

---

## 8. Dienste & MCP-Gateway (`loomux serve` / `loomux mcp`)

### `loomux serve [--port <port>]`
Startet den langlebigen localhost HTTP-Dienst mit dem Web OS und dem Root-MCP-Gateway.

- **Token-Bootstrap**: Gibt eine Einmal-URL aus (`http://127.0.0.1:<port>/?token=<hex>`). Der erste Aufruf setzt ein sicheres Session-Cookie.
- **SSE-Stream**: Streamt Echtzeit-Ereignisse aus `.loomux/state/journal/events.jsonl` über `/api/events`.

### `loomux mcp`
Stdio-Transportbrücke, die Claude Code, Cursor und Antigravity direkt mit Loomux-Tools verbindet:
- `graph_find_code`
- `graph_file_api`
- `graph_trace_calls`
- `graph_find_all`
- `graph_repo_map`
- `graph_check_freshness`
- `brain_search`
- `brain_catalog`

---

## 9. Entwickler-Prüftore (`loomux dev`)

### `loomux dev covergate --profile <coverage.out>`
Erzwingt ein striktes 100 % Test-Coverage-Tor pro Funktion.

- **Regel**: Jede nicht ausgenommene Funktion unter 100,0 % bricht das Tor ab.
- **Ausnahmen**: Nur zulässig mit `//coverage:exempt <begründung>` unmittelbar vor der `func`-Deklaration.

### `loomux dev swap --dir <bin>`
Tauscht das laufende `loomux.exe`-Binary atomar gegen `loomux.new.exe` aus (löst Windows Dateisperren-Konflikte).
