# loomux

🌐 **[English](README.md)** | **[Deutsch](README.de.md)** · 📖 **[Docs (EN)](docs/en/)** | **[Handbuch (DE)](docs/de/)**

**Ein Go-Binary, das KI-Coding-Agenten bewacht: Hooks und eine Schreibschranke, eine Prüfkette, ein Code-Graph und ein Projekt-Wiki.**

Claude Code und Antigravity arbeiten über Hooks mit loomux: Jedes Schreiben über die Dateiwerkzeuge des Hosts wird vor der Ausführung gegen die Regeln des Projekts und eine Schreibschranke geprüft, jeder Shell-Befehl gegen die Regeln des Projekts, und die Prüfkette läuft nach jeder Bearbeitung und am Ende jeder Runde. Cursor arbeitet nur als MCP-Client mit loomux, und Codex ist noch nicht angebunden. Über den MCP-Dienst fragt ein Agent den Code-Graphen und das Projekt-Wiki, statt das Repository erneut zu lesen.

- **Ein Binary.** loomux selbst braucht keine Python- oder Node-Laufzeit; die Werkzeuge, die es aufruft, etwa qmd für die Wiki-Suche und die Linter der Prüf-Lanes, können eine eigene brauchen.
- **Schneller Wächter.** Der Wächter entscheidet innerhalb eines Budgets von 35 ms.
- **Getestet je Funktion.** 100 % Testabdeckung je Funktion; eine Ausnahme braucht eine Begründung im Code.
- **Keine Telemetrie.** loomux selbst geht nur für Updates über `gh`, für das `git ls-remote origin` der Subagent-Hooks und auf ausdrückliche Befehle ins Netz. Die Prüfwerkzeuge, die seine Lanes starten (`uvx`, `npx`, `uv run --with`), können von sich aus Pakete herunterladen.
- **Windows zuerst.** Binaries für Linux und macOS gibt es auch, getestet wird aber unter Windows; Hook-Einrichtung und Selbstaktualisierung gibt es nur unter Windows.

> **Woher kommt der Name?**  
> **Loomux** verbindet den **Loom** (den Webstuhl, der die Fäden aus Agenten-Loops, Code-Graphen und Team-Wissen zu einem dichten, nahtlosen Gewebe verwebt) mit der Endung **-ux** (inspiriert von der Unix-Philosophie robuster, modularer Betriebssystem-Werkzeuge sowie einer kompromisslosen Developer & Agent Experience).

---

## Schnellstart (Windows)

Du brauchst die [GitHub CLI](https://cli.github.com/), angemeldet per `gh auth login`; sie übernimmt den Download und später die Selbstaktualisierung.

1. Lade das neueste stabile Release, prüfe es gegen `SHA256SUMS` und installiere es als `loomux.exe`. Führe das in PowerShell aus:

   ```powershell
   $bin = "$env:LOCALAPPDATA\loomux\bin"
   New-Item -ItemType Directory -Force $bin | Out-Null
   gh release download --repo xidus90/loomux --pattern 'loomux_*_windows_amd64.exe' --pattern SHA256SUMS --dir $bin --clobber
   $exe = Get-ChildItem $bin -Filter 'loomux_*_windows_amd64.exe' | Select-Object -First 1
   if (-not $exe) { throw 'download failed: see the gh error above' }
   $want = ((Select-String -Path "$bin\SHA256SUMS" -Pattern $exe.Name -SimpleMatch).Line -split '\s+')[0]
   if ((Get-FileHash $exe.FullName -Algorithm SHA256).Hash -eq $want) { Move-Item -Force $exe.FullName "$bin\loomux.exe"; Remove-Item "$bin\SHA256SUMS" } else { throw "checksum mismatch: $($exe.Name)" }
   ```

2. Richte in einer beliebigen Sitzung dein eigenes Projekt ein (`cd <projekt>`). `init` zeigt jede Änderung als Diff und schreibt nur, was du mit `y` bestätigst:

   ```powershell
   & "$env:LOCALAPPDATA\loomux\bin\loomux.exe" init
   ```

3. Prüfe die Einrichtung:

   ```powershell
   & "$env:LOCALAPPDATA\loomux\bin\loomux.exe" status
   ```

Die vollen Pfade sind nötig, weil nichts den Ordner auf deinen `PATH` setzt. Unter Linux oder macOS siehe [Linux und macOS](docs/de/install.md#linux-und-macos); alles Weitere zur Installation steht in [loomux installieren](docs/de/install.md), der nächste Schritt ist die [Ersteinrichtung](docs/de/getting-started.md).

---

## Wie es zusammenhängt

```mermaid
flowchart TD
    subgraph Core["loomux (ein einziges Go-Binary)"]
        P1["1. Hooks & Wächter<br/><b>Policy, Schreibschranke, Prüfkette</b>"]
        P2["2. Skills & Review<br/><b>Best-Practice-Suiten, graphgestützter Review</b>"]
        P3["3. Code-Graph & Loop<br/><b>AST, PageRank, Blast Radius</b>"]
        P4["4. Second Brain<br/><b>Wiki, Prüfzentrum, qmd-Suche</b>"]
        P5["5. LLM OS & UI<br/><b>Lokale Weboberfläche</b>"]
        Serve["loomux serve<br/><b>MCP-Gateway</b>"]
    end

    Agents["Coding-Agenten<br/>(Claude Code / Antigravity)"]
    Human["Entwickler / Team"]

    Agents <--> |"Hooks: stdin, Exitcode"| P1
    Agents <--> |"MCP über loomux mcp"| Serve
    Serve --> P3
    Serve --> P4
    Agents <-.-> |"MCP-Prompts, Skill-Dateien"| P2
    Human --> |".githooks: check precommit"| P1
    Human --> |"CLI: graph ask, callers, blast"| P3
    Human --> |"CLI: brain, reindex, cases, approve"| P4
    Human <--> |"Browser, localhost: loomux ui"| P5
```

*Eine gestrichelte Linie ist spezifiziert und nicht gebaut; was als Nächstes kommt, steht in der [Roadmap](#roadmap).*

---

## Funktionen

| Funktion | Was sie leistet | Doku |
|---|---|---|
| **Wächter und Schreibschranke** | Hooks prüfen jedes Schreiben über die Dateiwerkzeuge des Hosts und jeden Shell-Befehl gegen die Pfade, Befehle und Schreibregeln des Projekts, bevor sie laufen, und verweigern sie mit Nennung der Regel. `.loomux/config.toml` ist für Agenten schreibgeschützt, außer ein Mensch schaltet für das Projekt mit `loomux config llm on` den Schalter `llm_config` ein; das alles gilt innerhalb der Grenzen, die in den [Hooks](docs/de/hooks.md#7-der-entscheidungsweg-von-pre-tool-use) aufgelistet sind. | [Hooks](docs/de/hooks.md) |
| **Prüfkette und Tor** | Standardmäßig laufen nach jeder Bearbeitung die Lanes für Lint und Typen über die geänderte Datei; am Rundenende und im Git-Hook laufen die Lanes für Lint, Typen, Test, Coverage, Graph und Regeln über den neuen Inhalt der Runde und über den Commit. In einem Projekt mit `.loomux/armed.toml` ist eine Lane, die die Datei nicht nennt, in Probe: Sie läuft und berichtet, lässt aber nichts scheitern, bis sie scharf gestellt ist. `loomux check linecov` hält jede ausführbare Zeile an einer Regel, in jeder Sprache, deren Coverage-Werkzeug LCOV, Cobertura, JaCoCo-XML oder ein Go-Profil schreibt: gelaufen, oder mit Begründung ausgenommen; Bereich und Marker stammen aus den eigenen Einstellungen des Werkzeugs. `loomux check godot-suite` führt die gdUnit4-Suite eines Godot-Projekts headless in Scherben aus, auf Wunsch mit Nano Coverage, und `loomux godot rescan` erneuert dessen Klassen-Cache; die `test`- und `coverage`-Lanes von `gdscript` nutzen sie, wo das Projekt die Addons mitbringt. `loomux check rules` hält ein Projekt an seiner eigenen Ebenenkarte (`[rules.layers]`): keine Kante von einer Ebene in eine, die sie nicht benutzen darf, ein Testmodul für jede Quelle einer Ebene, die eines verlangt, keine Quelle außerhalb jeder Ebene und keine Coverage-Instrumentierung in einem Commit; ein Projekt ohne `[rules]` bekommt keine Regel-Lane; GDScript-Projekte führen im Lint nun außerdem `gdformat --check` aus. | [Konfiguration](docs/de/configuration.md#verify-prüfketten--quality-gates), [Ebenenkarte](docs/de/configuration.md#rules-die-ebenenkarte), [CLI-Referenz](docs/de/cli-reference.md#loomux-check-rules---root-path---cached) |
| **Code-Graph** | Ein lokaler, deterministischer Graph aus Go-, Python-, GDScript-, C#-, C++-, Java-, PHP-, JavaScript-, TypeScript-, HTML-, CSS- und SCSS-Code, JSX und TSX eingeschlossen, wobei JavaScript und TypeScript einander und über die Pfad-Aliase der `tsconfig.json` auflösen; HTML-Seiten mit ihren Inline-Skripten und -Stilen, die ihre Stylesheets und Skripte erreichen, ihre Klassen die Stilregeln und ihre `onclick`-Handler die Skriptfunktionen, während ein Skript die Elemente der Seiten erreicht, die es laden; SCSS-Mixins, -Funktionen und `@extend` über `@use` hinweg; GDScript von Godot 3 wie Godot 4, mit Szenen, ihren Signalverbindungen und `uid://`-Pfaden, und C#- und C++-Code, den Szenen und GDScript über die Brücken erreichen (C#-Methoden, `[Signal]`-Delegates und `[GlobalClass]`-Klassen; GDExtension-Klassen, ihre gebundenen Methoden und Signale sowie Knotentypen und Verbindungen einer Szene). `graph ask` rankt Symbole nach lexikalischer Relevanz und Personalized PageRank, `callers`, `skeleton`, `grep` und `map` navigieren darin, und `blast` zeigt, was ein Git-Diff erreicht. `graph grep` durchsucht jede Textdatei, die git nicht ignoriert, nicht nur die Dateien des Graphen, über einen Trigramm-Index neben dem Graphen, und scannt denselben Korpus, wenn der Index das Muster nicht einengen kann. | [CLI-Referenz §6](docs/de/cli-reference.md#6-code-graph-engine-loomux-graph) |
| **Wiki und Second Brain** | Markdown-Wikis, als Bereiche eines Repositorys registriert, Seite für Seite gelesen und durchsucht über qmd (`brain search`), das du getrennt installierst. `reindex` baut die Kataloge, den Link-Graphen und das Identitätsregister neu, `reconcile` öffnet für jede Wiki-Seite, deren Quelle sich geändert hat, einen Prüffall, und ein Mensch entscheidet, was ein Fall ändert (`cases`, `approve`). | [CLI-Referenz §7](docs/de/cli-reference.md#7-second-brain--wiki-loomux-brain) |
| **MCP-Dienst** | `loomux serve` bietet die Werkzeuge `brain_*` und `graph_*` auf dem Loopback an, ein Token je Kanal (lokal oder Cloud). `loomux mcp` ist die stdio-Brücke, die ein Host startet; sie bietet nur die Werkzeuge der Module an, die das Projekt eingeschaltet lässt. | [CLI-Referenz §8](docs/de/cli-reference.md#8-mcp-dienst--stdio-brücke-loomux-serve--loomux-mcp) |
| **Weboberfläche** | `loomux ui` bietet hinter einer Einmal-Anmeldung eine lokale Weboberfläche auf `127.0.0.1` an; ein Mensch startet sie, und der Wächter verweigert sie einem Agenten. Ein Dashboard aus Kacheln (Projekte, Config-Vorschläge, wartende Flow-Läufe, Prüffälle, Indexzustand, Version), die der Nutzer anordnet; je eine Einstellungsseite für jedes Projekt und für die Maschine, die jeden Schlüssel mit Wert und Herkunft zeigen und eine Änderung erst schreiben, nachdem ihr Diff gezeigt wurde, auch Regeln, Agentenvorschläge und der Schalter `llm_config`; ein Assistent, der ein Projekt in den Stufen von `loomux init` einrichtet. Die maschinenweite `config.toml` nimmt außerdem `[policy]`-Regeln, die in jedem Projekt gelten und die kein Projekt aufheben kann. | [CLI-Referenz](docs/de/cli-reference.md#loomux-ui---no-browser), [Erste Schritte](docs/de/getting-started.md#der-zweite-weg-der-browser), [Konfiguration](docs/de/configuration.md#regeln-für-jedes-projekt-policy) |
| **Flows** | Flows als Daten: ein Graph aus Knoten in TOML, der läuft, an einem Tor auf die Antwort eines Menschen wartet und sich aus einem Journal fortsetzen und wiedergeben lässt (`flow run\|resume\|replay\|show\|list`). Tor- und Ausgangsknoten laufen heute; Agentenknoten stehen in der [Roadmap](#roadmap). | [Flows](docs/de/flows.md) |
| **Kompatibilitätsmatrix** | `loomux dev compat matrix\|run\|scout` fährt jede unterstützte Kombination aus Godot, Python, Go, .NET, C++, Java, PHP und Node mit ihren Frameworks und Werkzeugen wirklich auf Linux und Windows; ein `release:major`-Pull-Request und jede Änderung der Liste warten darauf, und ein wöchentlicher Job eröffnet für jede neue Upstream-Version, die der Liste fehlt, ein Issue, prüft Alpha-, Beta- und RC-Versionen vorab, ohne etwas zu blockieren, und meldet gelistete Versionen nach Ende der Pflege. | [Unterstützte Versionen](docs/de/support.md#geprüfte-kombinationen) |
| **Worktree-Spiegel** (nur Windows) | `worktree link\|unlink\|remove` legen die Verzeichnisse, die ein Projekt nennt, als Junctions in seine verknüpften Git-Worktrees und nehmen sie wieder heraus. | [CLI-Referenz §5](docs/de/cli-reference.md#5-worktree-spiegelung-loomux-worktree) |
| **Releases und Selbstaktualisierung** (nur Windows) | `serve` holt täglich über `gh` das neueste Release des Kanals, dem das Binary folgt (stabil oder Beta), prüft es gegen `SHA256SUMS` und tauscht das Binary; `loomux upgrade` tut dasselbe von Hand und wechselt den Kanal. | [loomux installieren](docs/de/install.md#updates-und-kanäle) |

---

## Dokumentation

Leitfäden und technische Handbücher sind unter [`docs/de/`](docs/de/) gegliedert:

| Handbuch | Beschreibung |
|---|---|
| 📦 **[loomux installieren](docs/de/install.md)** | Voraussetzungen, Schnellstart unter Windows, einen Download prüfen, das maschinenweite Binary, Updates und Kanäle, Linux und macOS sowie der Bau aus dem Quellcode. |
| 🚀 **[Erste Schritte](docs/de/getting-started.md)** | Die Ersteinrichtung nach der Installation: `loomux init`, die Konfiguration und die Einrichtung prüfen und deinen Agenten-Harness anbinden (Hooks für Claude Code und Antigravity, MCP für Cursor). |
| 🧭 **[Unterstützte Sprachen und Versionen](docs/de/support.md)** | Welche Sprachen loomux prüft, was es für jede ausführt (lint, types, test, coverage, Code-Graph, Ebenenkarte) und gegen welche Versionen das geprüft ist, bis zu jeder geprüften Kombination aus Laufzeit, Framework und Werkzeug. |
| ⚙️ **[Konfigurations-Referenz](docs/de/configuration.md)** | Vollständige Referenz für `.loomux/config.toml` (`[verify]`, `[policy]`, `[modules]`, `[commit]`, `[worktree]`, `[privacy]`, `[model]`, `[agent]`, `[flow]`). |
| 📖 **[CLI-Referenzhandbuch](docs/de/cli-reference.md)** | Detailliertes Handbuch aller Befehle, Flags, stdin-JSON-Nutzlasten und Exit-Codes. |
| 🪝 **[Hook-Lebenszyklus & Integration](docs/de/hooks.md)** | Technische Spezifikation des 4-Phasen-Hook-Zyklus, der Host-Formate und des entkoppelten SSE-Ereignisstroms. |
| 🏛️ **[Architektur & Konzepte](docs/de/architecture.md)** | Das theoretische Fundament: Andrej Karpathys LLM OS, Googles Knowledge Items (KI), AST-GraphRank und der Schreibschranken-Kernel. |
| 🔀 **[Flows](docs/de/flows.md)** | Flows als Daten: das Ordnerformat, Rollen und Modelle, der Katalog und das Überschreiben, einen Flow beitragen und warum ein Tor einem Menschen gehört. |
| 🏷️ **[Releases und Pull Requests](docs/de/releases.md)** | Wie ein gemergter Pull Request zum Release wird, Beta-Releases, einen Pull Request öffnen und die Release-Einrichtung der Maintainer. |
| ⏱️ **[Leistungs-Benchmarks](docs/de/benchmarks.md)** | Chronologische Leistungsmessungen mit Datum, Basis und Änderung, kalt und warm. |
| 📊 **[Benchmark-Matrix](docs/de/benchmarks/matrix.md)** | Open-Source-Matrix über Top-Sprachen hinweg mit Detailberichten pro Sprache und Repository. |

---

## Roadmap

Was loomux als Nächstes bekommt. *Priorität* ordnet die Zeilen (1 zuerst); die Roadmap legt sie fest, und sie zu ändern ist Entscheidung des Nutzers.

### Kommt

| Funktion | Was sie bringt | Hängt ab von | Priorität |
|---|---|---|---|
| **Fusion von tgrep und Code-Graph** | `graph grep` durchsucht den ganzen Suchkorpus über einen Trigramm-Index (Index-Zustände, Zähler, Privacy-Kanal). Das DSH-Plugin `dsh-loomux` liefert ihn als `grep`-Werkzeug aus: kein Rückfall, und ein Fehler wird nie eine leere Liste. Der erste Benchmark gegen tgrep fand loomux bei Literal, Alternation und `-i` 3–13-mal langsamer; die Ursache war nicht das Laden von Wiring-Datei und Index, sondern zwei serielle Walks je Aufruf und ein vollständiger Neubau nach jeder Änderung. Jetzt bedient ein paralleler Walk beide Frischeprüfungen, eine Änderung an bis zu 200 Dateien wird aus dem Index plus dem Lesen dieser Dateien beantwortet (kein Neubau), `loomux serve` baut abgewichene Graphen im Hintergrund neu, und `--trust-index` überspringt den Walk. Gemessen auf einem Baum mit etwa 8500 Dateien (Windows, Median): Literal 74 ms statt 490, Alternation 103–113 ms, `-i` 71–83 ms, eine Änderung an 20 Dateien etwa 100 ms statt 4 s, und mit `--trust-index` Literal und Alternation so schnell wie tgrep (24 und 64–70 ms gegen 27 und 66–72 ms) und `-i` dreimal schneller. Der erste Aufruf nach einem Commit braucht 65–91 ms, unter dem Ziel von 108 ms. Offen: tgrep und `dsh-tgrep` ablösen (das entscheidet der Mensch nach der Benchmark-Abnahme) | Suchkorpus mit Trigramm-Index ✅ | — |
| **Syntax je Version für Python und Godot** | Der Code-Graph bekommt je gepflegter Python- und Godot-Version einen Syntax-Test, wie Go ihn schon hat, und liest, was jede Version der Sprache hinzufügt. Python kommt zuerst | Versionsbreite für Go ✅ | — |
| **Lesesperre** | Der Wächter verweigert auch das Lesen gesperrter Pfade, über `Read` und über lesende Shell-Befehle; heute verweigern seine Regeln nur das Schreiben, auch die maschinenweiten | — | 3 |
| **Flow-Laufzeit** | Flows als Daten: ein Graph aus Knoten in TOML, der läuft, an einem Tor auf die Antwort eines Menschen wartet und sich aus einem Journal fortsetzen und wiedergeben lässt. Gebaut: das Ordnerformat mit seiner Ladeprüfung, der Katalog, Rollen, die `[agent]` an Modelle bindet, Overlays, `[flow] default` und `overrides`, die Wächterregeln für Torantworten, Laufdateien und mitgelieferte Flows, `loomux flow run\|resume\|replay\|show\|list` und der Hinweis des Session-Starts auf wartende Läufe ([Flows](docs/de/flows.md)). Tor-, Ausgangs- und Agentenknoten laufen; Agentenknoten gehen über die CLIs von Claude Code, Antigravity (`agy`) und qoder, Strata über `claude` ([Entscheidung](docs/wiki/syntheses/modelladapter-ueber-die-cli.md)). Noch offen: `verify-until-green` als Daten-Flow für die Bausteine, die sie noch braucht; sie wird auch einen Diagnoselauf ohne Modell (`run --no-model`) nehmen | Flow-Kern ohne Modelle ✅ | 4 |
| **API-Modelladapter** | Agentenknoten erreichen Claude und Gemini auch über deren APIs: Läufe ohne installierte CLI, genaue Kostenerfassung, strukturierte Antworten ohne Parser. loomux führt die Werkzeugschleife dann selbst und muss dort die Schreibschranke und den Wächter halten und die Schlüssel irgendwo ablegen | Flow-Laufzeit | — |
| **Parallele Agentenknoten** | Fan-out und Join als Knotenart: mehrere Agentenknoten eines Flows laufen gleichzeitig. Die Zahl der gleichzeitigen Agenten ist begrenzt, mit Vorgabe, konfigurierbar und abschaltbar | Flow-Laufzeit | — |
| **Flows über MCP** | `flow_list`, `flow_show`, `flow_run` (ein Lauf in einem Kindprozess, den `serve` ablöst, mit sofortiger Antwort der Laufnummer) und `flow_status` auf dem lokalen Kanal, mit einer Statusdatei je Lauf, damit ein abgebrochener Lauf erkannt und fortgesetzt wird; kein Werkzeug beantwortet ein Tor | Flow-Kern ohne Modelle ✅ | 4 |
| **Entwicklungszyklus als Default-Flow** | Von der Planung bis zum Pull Request: Klärung, Spec und Plan, jeweils von einem Fächer aus Prüflinsen geprüft, dann je Aufgabe Recherche, Test zuerst, Bau, Prüfkette, Codereview und Nacharbeit, zum Schluss Doku, Abschlussreview und Commit. Ein Mensch antwortet an festen Toren und immer dann, wenn ein Modell nicht weiterkommt, und pusht | Flow-Laufzeit | 4 |
| **Community-Flows** | Der Entwicklungszyklus ist nur die Vorgabe. Gebaut: der Katalog unter `flows/catalog/` mit seinem Beitragstest (`go test ./flows` gegen ein Golden-Journal), `[flow] default` zur Wahl des Flows, Overlays einzelner Anweisungen und Fragen und Rollen, die ein Projekt an eigene Modelle bindet. Echte Läufe beigetragener Flows mit Agentenknoten nutzen die CLI-Modelladapter | Flow-Kern ohne Modelle ✅ | 4 |
| **Brain-Web-App** | Das Second Brain im Browser: Markdown-Editor, ADR-Katalog, Wissensgraph, die Kanten über Bereichsgrenzen (`edges-cross.json`) und im Browser ein Seitenleser, die Volltextsuche und das Prüfzentrum | Weboberfläche ✅ | 5 |
| **Graph-Visualizer** | Ein interaktiver Code-Graph mit Kanten-Chips, Typfiltern und Blast-Overlays und eine 3D-Ansicht; Code-Symbole verknüpft mit ADRs und Entwurfsdoku | Weboberfläche ✅, Graph-Navigation ✅ | 5 |
| **Skill-Suiten und Review** | Eingebettete Best-Practice-Regeln je Sprache (Go, Python, TypeScript, Rust); ein graphgestütztes Review, das `graph_blast` liest und die ADR-Treue prüft; verteilt über `.loomux/config.toml`, Host-Ordner, MCP-Prompts und die Web-Oberfläche | Blast-Radius eines Diffs ✅ | 5 |
| **Flow-Editor und Kanban** | Flows als Graph in der Weboberfläche zeichnen, wiedergeben und debuggen, im selben Format wie die Flow-Dateien; ein Kanban-Board, das Agentenschleifen, Prüf-Lanes und Subagenten live verfolgt | Weboberfläche ✅, Flow-Laufzeit | 5 |
| **Dashboard mit Drag-and-drop** | Die Kacheln des Dashboards lassen sich per Drag-and-drop verschieben statt mit den Knöpfen Up und Down der Anordnungsseite; das braucht JavaScript, das die Oberfläche heute nicht hat | Weboberfläche ✅ | 5 |
| **Sammelspeichern und Ordnerbrowser in der Weboberfläche** | Eine Einstellungsseite speichert mehrere geänderte Schlüssel mit einem Diff und einem Schreibvorgang statt Schlüssel für Schlüssel, und der Einrichtungsassistent lässt einen Ordner auswählen, statt seinen absoluten Pfad zu tippen | Weboberfläche ✅ | — |
| **Kompatibilitätsmatrix für die übrigen Stacks** | Die Kompatibilitätsliste deckt Godot, Python, Go, .NET, C++, Java, PHP sowie TypeScript, HTML und CSS auf Node ab; die übrigen fünf Stacks (Vue, Svelte, Shell, SQL, Rust) bekommen ihre Einträge, Fixtures und Laufzeit-Einrichtung in einem zweiten Pull Request | Kompatibilitätsmatrix ✅ | — |
| **Python-Quellwurzeln aus einer Projektdatei** | Der Python-Resolver errät, wo absolute Importe beginnen (Repository-Wurzel, der Ordner über jedem Top-Level-Paket, jeder Ordner mit `manage.py`); wo eine Projektdatei die Wurzeln festlegt, soll sie gelesen werden. Ob `pyproject.toml` das tut, ist offen: nur in werkzeugeigenen Formen, wenn überhaupt | — | — |
| **Typisierte Felder im GDScript-Resolver** | Aufrufe und Signale auf einem Feld mit Typannotation (`var _screen: ColonyScreen`, dann `_screen.refresh()` oder `_screen.build_requested.connect(…)`) ergeben heute keine Kante. Der Extraktor hielte den annotierten Typ jedes Felds fest, der Resolver bindet das Feld darüber; lokale Namen und Parameter, die ein Feld verdecken, brauchen Regeln | — | — |
| **`verify.profiles` über `config set`** | `loomux config set verify.profiles… --propose` lehnt den Schlüssel als unbekannt ab; der Vorschlagsweg dieses Repositorys erreicht ihn nicht | — | — |
| **Eine Repo-Wurzel als Wiki** | `[layout] wiki` lehnt die Wurzel eines Repositorys ab; ein Wiki-Repository wie ein reines Wiki-Projekt hat so keine Wiki-Lane | — | — |
| **`dev bench cases` ohne Hook** | `dev bench cases` lehnt eine `settings.json` ohne Hook ab, statt nur die Zusatzfälle zu schreiben | — | — |
| **Registerrevisionen nur aus der Prüfung** | Der Indexlauf schreibt nur Neuanlagen und Umbenennungen in `_identities.tsv` und zählt die Revision nicht mehr bei jeder Hashänderung hoch; zwei Rechner mit derselben Geschichte schreiben so dieselben Zeilen in ein versioniertes Register; `reconcile` schiebt den Hash einer Registerzeile, die keine Seite zitiert, vor, ohne ihre Revision zu erhöhen | — | — |
| **Registerkennungen, die nie entfallen** | `_identities.tsv` behält jede `doc_id`: Aliase für einen Pfad, ein Grabstein für eine verschwundene Datei, `merge=union` für zwei Zweige; heute überschreibt ein doppelter Pfad still, und eine gelöschte Datei verliert ihre Zeile | — | — |
| **`reindex` aus einem verknüpften Worktree** | `reindex` findet den Bereich eines verknüpften Worktrees über das gemeinsame Git-Verzeichnis und schreibt nur das Register dieses Zweigs; heute indexiert es immer den registrierten Pfad | Registerkennungen, die nie entfallen | — |
| **Die Schreibschranke schützt das Register** | Die Schranke verweigert jedes Schreiben auf `_identities.tsv` in einem registrierten Bereich, über Write/Edit und über die Shell; heute darf ein Agent es ändern | — | — |
| **Was ein Projekt beim Sitzungsstart braucht** | `hook session-start` warnt, wenn `core.hooksPath` nicht auf die versionierten Hooks zeigt oder das Repository kein Arbeitsbaum ist, und ein Projekt kann eigene Schritte einhängen; heute behält ein Projekt wie `space` dafür ein eigenes Sitzungsstart-Skript neben loomux, für diese Warnungen, einen Auszug aus dem Log und ein eigenes Godot-`user://` je Worktree | — | — |

### Vielleicht

| Funktion | Was sie bringt | Warum nur vielleicht |
|---|---|---|
| **Claude-Mods-Adapter** | Setzt die Schreibschranke in einen `tool.check`-Function-Hook ([claude-code#91870](https://github.com/anthropics/claude-code/issues/91870)), der über `$.mcp.call` mit einem langlebigen loomux spricht: kein Spawn je Aufruf, ein `ask`-Urteil und eine gerenderte Begründung | Nur Claude Code; der Exec-Hook bleibt der portable Weg |
| **Artefakte im Zustandsverzeichnis** | Ein schreibbarer Bereich behält im Baum nur seinen Wurzelkatalog; `graph.json` und die Kataloge je Verzeichnis wandern ins Zustandsverzeichnis, und ein Schritt des Umzugs räumt die alten aus jedem Bereich | Noch verlangt nichts danach, und die Kataloge je Verzeichnis sind Navigation im Baum |
| **`reindex` committet das Register** | `reindex` committet eine Änderung am Register über `internal/brain/vcs`, wie es `approve` tut | Noch verlangt kein Fall danach; `reindex` schreibt, ohne zu committen |

Ein Pull Request, der Roadmap-Arbeit beginnt, abschließt, hinzufügt oder
streicht, ändert diesen Abschnitt und sein englisches Gegenstück in
[`README.md`](README.md#roadmap).

---

## Credits

Code-Graph, Ranking und Blast-Radius sind angeregt von [trailhq/Graft](https://github.com/trailhq/Graft); die portierten Teile und ihre MIT-Lizenz stehen in [NOTICE.md](internal/notices/NOTICE.md).

---

## Lizenz

PolyForm Noncommercial License 1.0 (`LICENSE.md`).
