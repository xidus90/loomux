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

Die fünf Datenbefehle lesen die Bereiche der einen Registry (`registry.toml` in `LOOMUX_STATE_DIR`) und antworten wie `brain-mcp` von ultra-brain; ein aufgezeichneter Fallkorpus (`testdata/cases/1b-1`) hält sie daran. Bis Stufe 3 liegen die Artefakte eines schreibgeschützten Bereichs (`index.md`, `graph.json`, `_identities.tsv`) und der Reconcile-Stempel im Zustandsverzeichnis von ultra-brain: `LOOMUX_LEGACY_BRAIN_DIR`, Standard `%LOCALAPPDATA%\brain` unter Windows und `$XDG_STATE_HOME/brain` oder `~/.local/state/brain` unter POSIX. Bis Stufe 4 wird ein Bereichsverzeichnis, dessen `.loomux/config.toml` fehlt oder keine `[area]`-Tabelle trägt, über `.ultra-brain/config.toml` oder `.brain.toml` gelesen.

- **Kanal**: Jeder Befehl nimmt `--channel local|cloud` (Standard `local`). Ein Bereich mit `[privacy] mode = "local_only"` existiert im Kanal `cloud` nicht; `[privacy] never`-Globs gelten in jedem Kanal.
- **Usage-Fehler** (Exit `2`): die Usage-Zeile, dann `loomux brain <befehl>: error: <grund>` bei fehlendem Argument, ungültiger Wahl oder `-n` kleiner 1, und `loomux brain: error: <grund>`, wenn der Befehl fehlt oder unbekannt ist oder Argumente übrig bleiben.
- **Laufzeitfehler** (Exit `1`): `error: <grund>` auf `stderr` und nichts auf `stdout` — ein unbekannter Scope, eine Verweigerung, ein fehlender Abschnitt, ein kaputtes `graph.json` oder Identitätsregister, ein fehlendes oder unlesbares Manifest irgendeines registrierten Bereichs, eine fehlende oder kaputte Registry, eine nicht erreichbare Suchmaschine.
- **Ratschläge**: Die Meldungen nennen `brain reindex`, `brain reconcile` und `brain embed`, die Befehle von ultra-brain, bis Stufe 3 sie umschreibt.

### `loomux brain search <anfrage> [--scope <scope>] [--profile fast|full|keyword] [-n <n>] [--channel local|cloud]`
Durchsucht die sichtbaren Bereiche (Standard `--scope all`) über den qmd-MCP-Daemon unter `http://localhost:8765/mcp`.

- **Profile**: `fast` (Standard) Vektorsuche ohne Reranking und ohne Anfrageerweiterung; `keyword` BM25-Keyword-Suche; `full` die hybride Kette mit Erweiterung und Reranking. `-n` (Standard `5`) muss mindestens 1 sein.
- **Daemon**: Antwortet dort niemand, startet loomux `qmd mcp --http --daemon --port 8765` entkoppelt, schreibt `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` auf `stderr` und wartet bis zu 60 s auf ihn. Das Backbone ist standardmäßig CUDA; ein vom Nutzer gesetztes `QMD_LLAMA_GPU` oder `QMD_FORCE_CPU` bleibt unangetastet.
- **Ausgabe**: je Treffer `brain://<scope>/<pfad>:<zeile>  <score>%  <titel>`, jede Snippet-Zeile um vier Leerzeichen eingerückt, dann eine Leerzeile; ohne Treffer genau `no matches`.
- **Befunde**: nach den Treffern `note: <befund>`-Zeilen auf `stderr`, in dieser Reihenfolge — die Suchmaschine antwortete zweimal leer, ein Treffer fehlt im Identitätsregister seines Bereichs, wie viele Treffer zurückgehalten wurden, der Reconcile-Stempel ist 24 Stunden alt oder älter.
- **Exit-Codes**: `0` mit Treffern oder `no matches`; `1` bei einem Laufzeitfehler, auch wenn die Suchmaschine nicht antwortet — nie eine leere Antwort stattdessen; `2` bei einem Usage-Fehler.

### `loomux brain catalog [--scope <scope>] [--channel local|cloud]`
Gibt den Katalog der sichtbaren Bereiche oder eines Bereichs aus.

- **Ausgabe**: mit `--scope all` (Standard) `# brain`, eine Leerzeile und je sichtbarem Bereich `* [<scope>](brain://<scope>/)`, nach Scope sortiert; mit einem benannten Scope das `index.md` dieses Bereichs byte-gleich.
- **Exit-Codes**: `0`; `1` bei unbekanntem Scope, fehlendem `index.md` oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain read <pfad> --scope <scope> [--section <titel>] [--channel local|cloud]`
Gibt eine Datei eines Bereichs oder einen Abschnitt daraus aus.

- **Lesen**: streng UTF-8, Zeilenenden zu `\n` gefaltet. `--section` gibt ab der Überschrift mit diesem Titel bis zur nächsten Überschrift gleicher oder höherer Ebene aus.
- **Verweigerungen** (Exit `1`): `<scope>/<pfad> leaves the area`; `<scope>/<pfad> is excluded by [privacy] never`; `<scope>/<pfad> is the review centre; refused on the cloud channel`; `no section titled '<titel>'`.
- **Exit-Codes**: `0`; `1` bei einer Verweigerung oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain neighbors <pfad> --scope <scope> [--channel local|cloud]`
Gibt die Links in eine Seite hinein und aus ihr heraus aus, gelesen aus dem `graph.json` des Bereichs.

- **Ausgabe**: `incoming: <a>, <b>` und `outgoing: <c>`, jede Liste sortiert, `-` für eine Richtung ohne Links.
- **Exit-Codes**: `0`; `1` bei einer Verweigerung, einem Bereich ohne `graph.json` (``<scope>: never indexed; run `brain reindex` ``) oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain status [--channel local|cloud]`
Gibt aus, was man wissen muss, bevor man einer Antwort traut, eine Zeile je Befund.

- **Zeilen, in dieser Reihenfolge**: immer der letzte Abgleich (``last reconcile: never; run `brain reconcile` ``, ``last reconcile: <iso>; older than 24 h, run `brain reconcile` `` oder `last reconcile: <iso>`); je sichtbarem Bereich in Registry-Reihenfolge Include-Globs, die die Suchmaschine nicht sieht, ein Pfad, der nicht existiert, ein nie indizierter Bereich, weniger als die Hälfte aufgelöster Links und indizierte Dokumente, die die Suchmaschine nicht kennt; über alle sichtbaren Bereiche derselbe Inhalt unter mehreren Pfaden; einmal Dokumente, die indiziert, aber noch nicht durchsuchbar sind.
- **Suchmaschine**: Zwei Zeilen fragen die qmd-CLI (`qmd ls <collection>`, `qmd status`); antwortet sie nicht, sagt die Zeile das, und der Befehl läuft weiter.
- **Exit-Codes**: `0`; `1` bei einem Laufzeitfehler (Registry, Manifest, Stempel, `graph.json`, Identitätsregister); `2` bei einem Usage-Fehler.

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

### `loomux dev mutants <paket>... [--only <name>] [--family a1|a2|a3|a4] [--workers <n>]`
Mutiert die Go-Entscheidungen jedes Pakets und meldet, welche Mutanten seine Testsuite nicht bemerkt — ein Port von ultra-brains `tools/go_mutants.py`.

- **Familien**: `a1` die ganze `if`-Bedingung als `true` und als `false`; `a2` jeder Operand eines `&&` oder `||` auf oberster Ebene für sich; `a3` jeder Vergleichsoperator gekippt (`==`/`!=`, jede Ordnung gegen ihren Nachbarn), nicht in Kommentaren oder Zeichenketten; `a4` die Bedingung negiert. `for`-Bedingungen werden nie mutiert.
- **Mechanik**: Jeder Mutant erreicht `go test -overlay <json> -count=1 -failfast -timeout 60s ./<paket>/` über ein Overlay in einem temporären Verzeichnis; der Arbeitsbaum wird nie beschrieben. Jedes Overlay-Verzeichnis wird nach seinem Lauf entfernt, auch nach einem Fehler oder Strg+C. `--workers` fährt so viele Läufe gleichzeitig (Vorgabe: die Hälfte der Prozessoren, mindestens 1). `--only` behält Dateien, deren Name den Text enthält.
- **Bericht**: je Mutant eine Zeile — `killed`, `SURVIVED` oder `no mutant` (kompiliert nicht oder ändert nichts) — dann die Summen und die Überlebenden. Ein Lauf, der die Zeitgrenze reißt, gilt als getötet.
- **Exit-Codes**: `0` nach einer vollständigen Runde, auch mit Überlebenden; `2` bei einem Usage-Fehler, einem Paket ohne Quelldateien oder einer Suite, die vor dem ersten Mutanten nicht grün ist; `1`, wenn ein Lauf nicht gestartet werden kann oder die Runde mit Strg+C abgebrochen wird.

### `loomux dev swap --dir <bin>`
Tauscht das laufende `loomux.exe`-Binary atomar gegen `loomux.new.exe` aus (löst Windows Dateisperren-Konflikte).
