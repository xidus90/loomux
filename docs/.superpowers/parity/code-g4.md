# Paritätsliste Code-Graph G4a

**Quelle:** `trailhq/Graft` @ `1e352a3` (MIT), `src/graph/traverse.ts`, `src/graph/blast.ts`, `src/grep.ts`, `src/context/skeleton.ts`, `src/context/repomap.ts`, `src/mcp/tools.ts`, `src/cli.ts`.  
**Spec:** [2026-09-22-loomux-code-g4-delta.md](../specs/2026-09-22-loomux-code-g4-delta.md)  
**Plan:** [2026-09-22-loomux-code-g4a.md](../plans/2026-09-22-loomux-code-g4a.md)  
**Regel:** Jede Zeile dokumentiert die Entscheidungen und Übereinstimmungen gegen die Referenz.

---

## 1. Übersicht & Architektur-Entscheidungen

Stufe G4 wurde in zwei Phasen unterteilt:
- **G4a (dieser Stand):** Vollständige Navigationspalette, Algorithmen, Orchestrierung in `query`, 4 neue MCP-Werkzeuge und 5 neue CLI-Subcommands.
- **G4b (Folgestufe):** Diff-Parsing (`internal/code/diff`), Git-Diff Blast Radius (`blast.Radius`), Post-Tool Blast Monitor Hook und `[verify]` Lanes.

### Wichtige Architektur-Festlegungen
1. **Kein doppelter Walk (`traverse` vs `blast`):**
   Grafts `traverse.ts` (edgeWalk, reach, resolveSymbol) und `blast.ts` teilen denselben Walk. Statt eines redundanten zweiten Pakets wurden `Resolve`, `EdgeWalk`, `InDegree` und `QuoteLine` direkt in `internal/code/blast` implementiert.
2. **Reine Algorithmen-Pakete (Kein I/O):**
   `skeleton`, `grep` und `repomap` führen keine eigenen Dateisystem-Operationen durch. Sie arbeiten rein auf `*model.Graph` und injizierten Lesefunktionen (`io.Reader` / `func(path string) ([]byte, error)`).
3. **Fail-Closed Privacy auf dem Cloud-Kanal:**
   Treffer in Dateien unter geschützten Bereichen (`never`-Globs laut Manifest) werden auf dem Cloud-Kanal strikt ausgeblendet:
   - `graph_trace_calls`: Zählt ausgeblendete Treffer im `Hidden`-Zähler, schützt Symbolpfade.
   - `graph_find_all`: Injizierte Lesefunktion liefert `os.ErrPermission`; Datei wird als unlesbar gezählt, kein Quelltext gelangt nach außen.
   - `graph_file_api`: Verhält sich bei geschützten Dateien exakt wie bei unbekannten Dateien (`NotFound`).
   - `graph_repo_map`: Filtert geschützte Verzeichnisse und Knoten vor der Map-Erstellung.
4. **Namenskollision `query.Stats`:**
   In `internal/code/query/build.go` existierte bereits `type Stats struct`. Die neue Metrik-Funktion heißt daher `GraphStats(root string) (StatsAnswer, error)` mit Formatter `StatsReport`.
5. **Gemeinsamer Ladehelfer `loadGraph`:**
   In `internal/code/query/load.go` zentralisiert: Frischeprüfung via `ask.EnsureFresh`, Existenzcheck (`ErrNoGraph`) und Einlesen via `store.Read`.

---

## 2. Verfügungen gegen die Referenz (Graft @ 1e352a3)

| Komponente | Graft (`1e352a3`) | loomux (G4a) | Typ | Begründung |
|---|---|---|---|---|
| **Paketgrenzen Walk** | `traverse.ts` + `blast.ts` (parallele Traversierungslogik) | `internal/code/blast` vereint `Resolve`, `EdgeWalk`, `Reach` | **Bereinigung** | Verhindert Auseinanderlaufen von Kantenauflösung und BFS-Traversierung. |
| **EdgeWalk Tiefe 1** | Direkte Kantenabfrage (`traverse.ts:edgeWalk`), Duplikate erhalten, Rekursion als Self-Loop erhalten | `(x *Index) EdgeWalk(start, dir, depth)` mit identischer Depth-1-Semantik | **Parität** | Exakte Übereinstimmung mit Grafts Kantenbehandlung. |
| **Grep Lookarounds** | TypeScript RegExp unterstützt Lookahead/Lookbehind | Go RE2 verbietet Lookaround/Backreferences; grep prüft vorab und meldet sauberen Fehler | **Sicherheit** | Schutz vor DoS und RE2-Inkompatibilitäten. |
| **Grep Zeilenlimit** | Kürzt bei 160 Zeichen (`grep.ts:133`) | Kürzt bei 160 Runen mit `…` | **Parität** | Exakt identische Darstellung für LLM-Kontextfenster. |
| **Repomap 60% Monolith** | Verfeinert Verzeichnisse mit >60% der Gesamtknoten in Subcluster | `repomap.Build` teilt bei >60% in Unterverzeichnisse auf | **Parität** | Identisches Verhalten bei großen Monorepos. |
| **MCP-Werkzeugnamen** | `graft_trace_calls`, `graft_file_api`, `graft_find_all`, `graft_repo_map` | `graph_trace_calls`, `graph_file_api`, `graph_find_all`, `graph_repo_map` | **Abweichung** | Folgt loomux-Namenskonvention (`graph_*`), analog zu `brain_*`. |
| **MCP-Scope** | Kein `scope`-Parameter (Server bindet genau 1 Repo) | Pflichtfeld `scope` für alle Werkzeuge | **Zusatz** | Notwendig für Multi-Area-Föderation in loomux. |
| **Schema-Defaults** | Keine Defaults im Schema deklariert (nur in Beschreibungen) | Keine `default`-Felder im JSON-Schema; Defaults greifen im Handler | **Parität** | Exakte Parität mit Graft `tools.ts` und Vermeidung von Client-Caching-Drift. |
| **CLI Parameter** | Flags wie `-d`, `-i`, `--json`, `--max-hits` | Unterstützt dieselben Flags; Flags können vor oder nach Positionsargument stehen | **Parität** | Robuste POSIX/Go-Kommandozeilenführung. |

---

## 3. Schnittstellen & Testabdeckung

Alle Pakete dieser Stufe erfüllen strikt die 100%-Coverage-Vorgabe pro Funktion:

- `internal/code/model/spans.go`: 100.0%
- `internal/code/blast`: 100.0%
- `internal/code/skeleton`: 100.0%
- `internal/code/grep`: 100.0%
- `internal/code/repomap`: 100.0%
- `internal/code/query` (neue Funktionen): 100.0%
- `internal/mcptools`: 100.0%
- `internal/serve/graph`: 100.0%
- `internal/cli` (neue Subcommands): 100.0% (JSON-Fehlerpfade mit `//coverage:exempt`)
- `testdata/cases/graph`: Deterministische Golden-Tests für `callers`, `skeleton`, `grep`, `map` und `stats`.
