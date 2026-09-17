# loomux Stufe 1b-1 — die fünf Datenbefehle und `dev mutants`

**Datum:** 2026-09-15 (überarbeitet nach den Faktenblättern desselben Tages)
**Stand:** umgesetzt, abgeschlossen 2026-09-17 (Plan `2026-09-15-loomux-stufe-1b-1.md`)
**Bezug:** [Fusions-Spec](2026-09-14-loomux-fusion-design.md), Zeile „1b" der Stufentabelle. Stufe 1b
wird dreigeteilt; diese Spec deckt 1b-1. 1b-2 (`serve`, MCP, Brücke, Upkeep) und 1b-3 (Wiki- und
Doku-Umzug) bekommen je eigene Specs.
**Messgrundlage:** Kontextaufnahme, qmd-Port-Spike und vier Faktenblätter vom 2026-09-15
(`.superpowers/sdd/1b-1-facts/{moved-packages,search,status,loomux-now}.md`, git-ignoriert; die
tragenden Befunde stehen unten).

## Ziel

`loomux brain search|catalog|read|neighbors|status` beantworten dieselben Fragen wie
`brain-mcp search|catalog|read|neighbors|status` aus ultra-brain, mit belegter Parität zur
**Python**-Referenz, und `loomux dev mutants` misst, welche Entscheidungen der Code-Basis kein Test
bemerkt. Danach kann 1b-2 einen MCP-Dienst auf diese Befehle stellen, ohne sie neu zu erfinden.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Zuschnitt von 1b | Dreiteilung: 1b-1 Datenbefehle + `dev mutants`, 1b-2 `serve`/MCP/Brücke/Upkeep, 1b-3 Wiki- und Doku-Umzug. Je eigene Spec, eigener Plan, einzeln abnehmbar |
| Namensraum | `loomux brain search\|catalog\|read\|neighbors\|status`. `loomux status` bleibt die Hook-Inspektion aus 1a. Die MCP-Werkzeuge in 1b-2 heißen entsprechend `brain_search` usw. |
| qmd für `search` | Warmer qmd-MCP-Daemon über HTTP. `fast` = `{searches:[{type:"vec"}], rerank:false}` ohne LLM-Erweiterung |
| qmd für `status` | qmd-CLI (`qmd ls`, `qmd status`) — qmd-MCP hat kein `ls`, und `qmd status` zählt mit dem konfigurierten Embedding-Modell |
| GPU-Backbone | CUDA als Vorgabe wie beide Referenzen; eine vom Nutzer gesetzte `QMD_LLAMA_GPU` bzw. `QMD_FORCE_CPU` gewinnt |
| Entstehung des Go-Codes | Umziehen, was nachweislich trägt, Rest neu nach der Python-Referenz |
| Zustand | Eine Registry: `%LOCALAPPDATA%\loomux\registry.toml` für Schranke und Datenbefehle |
| Befristete Ausnahme | `brain/*` liest Artefakte schreibgeschützter Bereiche und den Reconcile-Stempel aus `%LOCALAPPDATA%\brain` (bis Stufe 3) und die Manifest-Altnamen `.ultra-brain/config.toml`, `.brain.toml`, wo keine `.loomux/config.toml` steht (bis Stufe 4). Die Schreibschranke bleibt streng |
| Referenz im Zweifel | Wo die Python-Referenz Verhalten oder Wortlaut hat, gilt es — auch wo ultra-brains Go-Fassung abweicht |
| Parität von `status` | Synthetische Welten je Zeilenart im Tor, dazu einmal ein Live-Vergleich in der Paritätsliste |
| `dev mutants` | Paralleler Go-Port der vier Familien; der Mutant geht per `go test -overlay` in den Lauf, keine Datei im Baum wird beschrieben; volle Runde inklusive der Entscheidungspakete aus 1a |

### Änderungen gegenüber der Fusions-Spec

- **„1b: neuer Go-Code, kein Umzug"** gilt nicht mehr uneingeschränkt. ultra-brains Go-`catalog`,
  `reader`, `graph`, `privacy` und die beiden qmd-Ports sind nahe Parität; sie ziehen mit Tests um und
  werden dort korrigiert, wo die Referenz anders spricht.
- **Harter Schnitt, befristet durchbrochen:** Bis Stufe 3 `reconcile` nach Go holt, pflegt die
  Python-Seite die Artefakte unter `%LOCALAPPDATA%\brain`; bis Stufe 4 die Wirte umstellt, tragen die
  Bereichs-Repos ihre alten Manifestnamen. `brain/*` liest beides lesend, statt Kopien zu führen oder
  Privacy-Einstellungen zu übersehen.
- **MCP-Werkzeugnamen:** Richtung `brain_*`; formal festgelegt in 1b-2.
- **Die offene Frage „woher die abweichende Rangfolge der Go-Suche kommt"** ist beantwortet (siehe
  Ausgangslage) und wird die erste Zeile der Paritätsliste.

## Ausgangslage, vermessen am 2026-09-15

### Die Python-Referenz

- Alle fünf Befehle sind Funktionen in `src/brain/core.py`: `search` :76, `catalog` :278, `read` :292,
  `neighbors` :316, `status` :328. `cli.py` verdrahtet nur; Dispatch :712-808, `status` :810.
- Aufruf über das Konsolenskript `brain-mcp` (`pyproject.toml`: `brain-mcp = "brain.cli:main"`; der
  Name `brain` gehört dem Go-Binary auf dem PATH), im Tag-Worktree
  `C:\Users\micro\Documents\#GIT\loomux-src\ub` (Tag `loomux-1a-source`, `3cc72d2`):
  `uv run --project <ub> brain-mcp <befehl> …` — gemessen lauffähig aus einem fremden Verzeichnis.
- Argumentformen: `search <anfrage> [--scope all] [--profile keyword|fast|full]` (Vorgabe `fast`)
  `[-n N]` (Vorgabe 5, `must be at least 1, got N`); `catalog [--scope all]`;
  `read <pfad> --scope S [--section T]`; `neighbors <pfad> --scope S`; `status`. Alle fünf nehmen
  `--channel local|cloud` (Vorgabe `local`) und `--state-dir`. Ungültige Wahl → `argparse` → Exit 2.
- Läuft ein brain-Daemon, leitet die CLI über ihn und druckt ein ganz anderes Format; ohne Daemon
  nutzt sie für **jedes** Profil die qmd-CLI (`qmd vsearch|search|query --json`). Fehler aus
  `GraphError`, `IdentityError`, `ManifestError`, `RegistryError`, `OSError`, `SearchUnavailable`
  enden mit `error: {e}` auf stderr und Exit 1, ohne stdout.
- Über eine Pipe schreibt Python unter Windows `\r\n` und cp1252 (Python 3.14, `utf8_mode=0`); `…`
  (U+2026) wird zu Byte 0x85, ein Pfad außerhalb cp1252 bricht mit `UnicodeEncodeError`.
- `src/brain/search/qmd_mcp.py:289` hat am Tag einen Syntaxfehler; die CLI nutzt diesen Port nicht.
- Kein vorhandener Paritätstest deckt die fünf Befehle.

### Der Go-Stand in ultra-brain

| Paket / Dateien | Befund |
|---|---|
| `pkg/catalog` `area.go`, `root.go` | nahe Parität; `catalog.go` (`BuildCatalog`, `RenderCatalogJSON`) dient nur dem Go-status-Stumpf und zieht nicht um |
| `pkg/reader` `read.go`, `section.go` | nahe Parität; `ExtractSection` spaltet nur `\n` (Python `splitlines` spaltet mehr), liefert Rohbytes (Python faltet CRLF, liest streng UTF-8) |
| `pkg/graph` `model.go`, `neighbors.go`, `read.go` | nahe Parität; `render.go`, `synth.go` dienen reindex/layout/bench und ziehen nicht um |
| `pkg/privacy` | **schwächerer Glob-Matcher** als Python (kein `a/**/b` gegen `a/b`, keine Klassen `[a-z]`, kein NFC, nur `ToLower` statt casefold); `VisibleManifest` versteckt einen kaputten Bereich still und behandelt einen fehlenden als sichtbar ohne `never` |
| `pkg/search` `http.go`, `mcp.go`, `daemon*.go`, `launcher.go`, `port.go`, `fake.go` | qmd-MCP-Port, Profile korrekt gemappt (`mcp.go:184`); Start mit CUDA-Vorgabe, Backbone-Umgebung wird **nach** `os.Environ()` angehängt und überschreibt den Nutzer |
| `pkg/search` `qmd.go` | qmd-CLI-Port; `DefaultRunner` umgeht den Launcher (auf Windows `qmd.CMD` über `cmd.exe`) und setzt kein Backbone |
| `pkg/search` `search.go` | `ExecuteSearch`, `FormatSearch`, `CollectionName`, `NoMatches`; ohne Identitätsregister und Befunde; Snippets über `\n` statt `splitlines` |
| `pkg/index` `identity.go` | Identitätsregister mit den Python-Texten, aber `%q`-Quotes, `strconv.Atoi` (nimmt `+5`), spaltet nur `\n` |

Messages in `graph`, `reader`, `privacy` nutzen Apostrophe und `%q`, wo Python Backticks und `repr`
schreibt. Alle genutzten Bezeichner aus `pkg/config` existieren in loomux' `internal/config` mit
gleicher Signatur; `pkg/catalog` importiert `pkg/wiki`, dessen `ReadPage` in `internal/brain/wiki`
gleich existiert. Coverage heute 99,4 % (`catalog.go` unter 100 %, zieht nicht um).

### Warum Go und Python verschieden ranken (Spike)

- **Go mappt die Profile korrekt.** Die Differenz liegt in qmd 2.8.3: `qmd vsearch` (Weg der
  Python-CLI) lässt ein LLM die Anfrage erweitern, sucht je Variante, führt per Maximum zusammen und
  schneidet unter 0,3 ab; MCP `searches:[vec]` sucht einmal und rankt nach Position (Score 1/Rang).
  Beleg: `qmd query "vec: <q>" --no-rerank` liefert exakt die MCP-Liste.
- `full`: auf beiden Wegen identisch (4 von 4). `keyword`: gleiche Reihenfolge, aber BM25-Score und
  erste Fundstelle gegen 1/Rang und Zeile des besten Abschnitts.

| Weg | keyword | fast | full |
|---|---:|---:|---:|
| CLI, Prozess je Anfrage (neue Anfrage) | 179–191 ms | 10,5–13,9 s | 20,0–24,1 s |
| MCP-Daemon warm | 5–17 ms | 63–83 ms | 4,2–7,9 s neu, ~250 ms wiederholt |
| MCP-Daemon frisch gestartet, erste Anfrage | 23 ms | 5,4 s | 11,9 s |

- qmd bindet den Namen `localhost`; beide Referenzen wählen ihn über den Namen an, `127.0.0.1` wird
  abgewiesen. Adresse `http://localhost:8765/mcp`, TCP-Probe 500 ms, `initialize`-Handshake
  (`protocolVersion 2025-06-18`) mit 5 s Frist.
- Während des Spikes stürzte ein CUDA-Daemon einmal ab, als gleichzeitig ein Vulkan-CLI-Prozess die
  GPU belegte. Pythons eigene Messung über den HTTP-Daemon: 15 von 15 Kaltstarts überlebten.

### `go test -overlay`

Gemessen: derselbe Mutant in `internal/config`, per `-overlay` im unveränderten Baum eingesetzt, wird
in 0,60 s getötet — so schnell wie in einer Modulkopie (0,59 s), ohne die 606 Dateien und ~200 ms
einer Kopie. Der Build-Cache wird wiederverwendet.

## Architektur

### Pakete

```
internal/brain/catalog    umgezogen aus ub pkg/catalog (area.go, root.go + Tests)
internal/brain/reader     umgezogen aus ub pkg/reader (+ Tests), Abschnitte und Lesen nach Python
internal/brain/graph      umgezogen aus ub pkg/graph (model.go, neighbors.go, read.go + Tests)
internal/brain/privacy    umgezogen aus ub pkg/privacy (+ Tests), Glob-Matcher und Sichtbarkeit nach Python
internal/brain/identity   umgezogen aus ub pkg/index/identity.go (+ Test), Texte und Parsen nach Python
internal/brain/search     beide qmd-Ports und search.go umgezogen; neu: Befunde, Launcher im CLI-Port
internal/brain/status     neu, Regel für Regel nach core.status
internal/cli/brain.go     loomux brain search|catalog|read|neighbors|status
internal/dev/mutants      neu: vier Familien, N Arbeiter, go test -overlay
```

Umzugsregel wie in 1a: Code und Tests wortgleich bis auf Paketnamen, Importpfade, genannte Literale
und neue Tests für 100 %; jede weitere Änderung ist ein Befund im Task-Bericht. „Nach Python" heißt:
die Abweichung aus der Ausgangslage wird test-first auf das Referenzverhalten gebracht.

### Abhängigkeitsregeln

- `brain/*` darf `config` benutzen und einander; sonst nichts aus loomux.
- `cli` ruft `brain/*`.
- `hooks` importiert kein neues `brain/*`-Paket. Der Pfad an jedem Edit zieht keinen der neuen Befehle
  nach.

### Die befristete Ausnahme

Genau zwei Funktionen in `internal/config`, beide mit dem Ablauf im Namen und im Kommentar:

1. **Artefakte und Stempel (bis Stufe 3):** für schreibgeschützte Bereiche das Verzeichnis
   `<legacy>\areas\<flat-scope>` statt `<loomux>\areas\<flat-scope>`, und der Reconcile-Stempel
   `<legacy>\maintenance\last-run.txt`. `<legacy>` ist `LOOMUX_LEGACY_BRAIN_DIR`, sonst
   `%LOCALAPPDATA%\brain` (unter POSIX `$XDG_STATE_HOME/brain`, sonst `~/.local/state/brain`).
2. **Manifest-Altnamen (bis Stufe 4):** liest in einem Bereichsverzeichnis `.loomux/config.toml`; fehlt
   sie, `.ultra-brain/config.toml`, dann `.brain.toml`. Nur `brain/*` ruft diese Funktion; die
   Schreibschranke (`internal/brain/guard`) und `config.ReadManifest` bleiben bei `.loomux/config.toml`.

Die Registry kommt immer aus `config.StateDir()` (`%LOCALAPPDATA%\loomux`). Wo ein Befehl heute ein
einzelnes `stateDir` an umgezogenen Code reicht, bekommt dieser Code das Legacy-Verzeichnis für
Artefakte und Manifeste schreibgeschützter Bereiche, und die Registry wird getrennt gelesen.

### qmd

- **Daemon (search):** Probe und Handshake gegen `localhost:8765` wie in der Referenz. Antwortet keiner,
  startet loomux `<launcher qmd> mcp --http --daemon --port 8765` entkoppelt (Windows
  `DETACHED_PROCESS`, sonst `Setsid`), wartet bis 60 s im 250-ms-Takt. Backbone-Vorgabe CUDA (keine
  Variable), Vulkan → `QMD_LLAMA_GPU=vulkan`, CPU → `QMD_FORCE_CPU=1`; ist eine dieser Variablen beim
  Nutzer gesetzt, bleibt sie unangetastet.
- **CLI (status):** `qmd ls <collection>` und `qmd status` über den Launcher (Shim-Umgehung wie in
  Pythons `_default_runner`), mit derselben Backbone-Regel.
- Collection-Name: jedes Zeichen außerhalb `[A-Za-z0-9_.-]` wird `-`, führende und abschließende `-`
  entfallen (`project/space` → `project-space`).
- In 1b-2 übernimmt `serve` den Daemon; der Port bleibt so, dass `serve` ihn ohne Umbau hält.

### Startzeit

Kein umgezogenes und kein neues Paket parst in `init()` oder in einer Paketvariable Daten. Nachweis
mit `GODEBUG=inittrace=1 loomux --version` vor und nach dem Einzug; der Befund steht im
Benchmark-Eintrag.

## Befehle, Ausgabe, Datenfluss

Argumentformen und Vorgaben wie die Referenz (siehe Ausgangslage), mit `loomux brain` statt
`brain-mcp`. Ein Bereich mit `[privacy] mode = "local_only"` existiert im Kanal `cloud` nicht;
`never`-Globs gelten in jedem Kanal.

| Befehl | stdout |
|---|---|
| `search` | je Treffer `brain://{scope}/{pfad}:{zeile}  {score:.0%}  {titel}`, darunter jede Zeile von `snippet.splitlines()` mit vier Leerzeichen, dann eine Leerzeile; ohne Treffer genau `no matches`. Exit 0 |
| `catalog` | `# brain`, Leerzeile, je sichtbarem Bereich `* [{scope}](brain://{scope}/)`, nach Scope sortiert; ein benannter Bereich gibt sein `index.md` aus, streng UTF-8 gelesen und mit Zeilenenden zu `\n` gefaltet |
| `read` | die Datei oder ein Abschnitt, wie `core.read` sie liest |
| `neighbors` | `incoming: a, b` und `outgoing: c`, `-` wenn eine Richtung leer ist |
| `status` | siehe unten |

**Befunde von `search`** auf stderr als `note: {befund}`, in dieser Reihenfolge: der Befund „zweimal
leer", je Treffer der Registerbefund, die Zahl zurückgehaltener Treffer, der Stempelbefund. Ohne
sichtbare Bereiche kehrt `search` ohne Befunde zurück. Die Register aller sichtbaren Bereiche werden
gelesen, bevor ein Treffer geprüft wird. Wortlaute:
- `{scope}/{relative}: hit is not in the register; reindex to catch up`
- ``the last full reconciliation was {iso}, more than 24 hours ago: a source may have changed without this answer knowing (run `brain reconcile`)`` — nur bei zonenbehaftetem Stempel und
  `jetzt - stempel >= 24 h`. `{iso}` wie Pythons `isoformat()`: `+00:00`, Mikrosekunden nur wenn
  ungleich null.

**`status`**, eine Zeile je `print`, in dieser Reihenfolge:
1. ``last reconcile: never; run `brain reconcile` `` / ``last reconcile: {iso}; older than 24 h, run `brain reconcile` `` / `last reconcile: {iso}` (immer; unparsebarer oder naiver Stempel zählt als `never`).
2. Je sichtbarem Bereich in Registry-Reihenfolge:
   - `{scope}: the search engine sees only {include[0]}; also declared: {include[1:] mit ", "}` bei zwei oder mehr Include-Globs;
   - `{scope}: {pfad} does not exist; skipped` (Pfad in Windows-Schreibweise), dann weiter zum nächsten Bereich;
   - ``{scope}: never indexed; run `brain reindex` `` ohne `graph.json`, dann weiter;
   - `{scope}: only {resolved} of {total} links resolved ({k=v nach Schlüssel sortiert})` bei `total` und `resolved*2 < total`;
   - ``{scope}: {fehlend} of {unsere} indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. {erste drei}{", …" wenn mehr}`` — unsere = sortierte Pfade aus `_identities.tsv` ohne `never`- und `unsearched`-Globs, ihre = NFC von `qmd ls <collection>`; scheitert qmd: `{scope}: the search engine did not answer ({fehler}); its index was not compared`.
3. Über alle sichtbaren Bereiche, nach Hash-Zeichenkette sortiert: `same content hash under {n} paths: {scope/pfad sortiert, ", "}` bzw. `{pfad}: same content hash as a path excluded by [privacy] never`.
4. Einmal: ``{n} documents are indexed but not yet searchable; run `brain embed` `` aus `qmd status` (`^\s*Pending:\s+(\d+)\s+need embedding`, kein Treffer = 0); scheitert qmd: `the search engine did not answer ({fehler}); the backlog was not counted`.

Die Ratschläge nennen `brain reindex`, `brain reconcile`, `brain embed` — bis Stufe 3 sind das die
Python-Befehle, die der Nutzer hat. Stufe 3 schreibt sie um.

Datenfluss: loomux-Registry → Manifest je Bereich (befristete Altnamen) → sichtbare Bereiche je Kanal
→ Artefaktverzeichnis (schreibgeschützt: befristet Legacy) → `_identities.tsv`, `graph.json`,
`index.md`, Stempel → qmd-Daemon für `search`, qmd-CLI für zwei `status`-Zeilen.

## Fehlerverhalten

- **Usage-Fehler** (fehlendes Argument, ungültige Wahl, `-n` kleiner 1) → Exit 2, wie `argparse` der
  Referenz.
- **Laufzeitfehler** → `error: {grund}` auf stderr, Exit 1, kein stdout — wie die Referenz. Darunter:
  unbekannter Scope (`unknown scope 'x'; known scopes are: …`, Python-`repr`-Quotes), Verweigerungen
  (`{scope}/{pfad} leaves the area`, `… is excluded by [privacy] never`,
  `… is the review centre; refused on the cloud channel`), fehlender Abschnitt
  (`no section titled 'x'`), kaputtes `graph.json` (Wortlaut mit Backticks wie Python), unlesbares
  Identitätsregister (`{pfad}: line {n}: expected 4 tab-separated fields, found {k}` bzw.
  `… revision 'x' is not a number`), fehlendes oder unlesbares Manifest **irgendeines** registrierten
  Bereichs, kaputte Registry.
- **Nie leer statt kaputt:** Ist qmd nicht erreichbar und nicht startbar, antwortet `search` nicht mit
  `no matches`, sondern mit Exit 1 und einer Meldung, die den gescheiterten Schritt nennt.
- **Aufwärmen:** Hat `search` den Daemon selbst gestartet, schreibt es vor der ersten Anfrage einen
  Hinweis auf stderr, dass das Modell geladen wird. Einen schon laufenden, noch kalten Daemon kann
  loomux nicht erkennen. Die CLI-Referenz kennt den Hinweis nicht — Paritätszeile.
- **`dev mutants`:** Ist die Suite vor der Runde nicht grün → Exit 2, keine Runde. Ein Mutant, der
  nicht kompiliert, zählt gesondert; einer, der die Zeitgrenze reißt, gilt als getötet. Temporäre
  Overlay-Dateien werden auch bei Abbruch aufgeräumt; der Arbeitsbaum wird nie beschrieben.

## Paritätsnachweis und Tests

### Fallkorpus

- Unter `testdata/cases/1b-1/`, Format und Werkzeuge wie in 1a, Fallzahl in einer eigenen Suite
  festgenagelt.
- Aufgezeichnet gegen die Python-Referenz **ohne laufenden brain-Daemon**, über
  `uv run --project <ub> brain-mcp …`. `record-case` bekommt eine Programmform (Programm plus
  führende Argumente), setzt `PYTHONUTF8=1` und normalisiert `\r\n` im aufgezeichneten stdout zu `\n`.
- Übersetzt: Präfix `brain-mcp ` → `loomux brain `; `BRAIN_STATE_DIR` → `LOOMUX_STATE_DIR` und
  `LOOMUX_LEGACY_BRAIN_DIR`. `TranslateWorld` faltet Altmanifeste in **jedem** Bereichsverzeichnis einer
  Welt, nicht nur in der Wurzel und unter `areas/`.
- Fälle je Befehl: Erfolg, jede Verweigerung und jeder Fehlerweg aus dem Abschnitt Fehlerverhalten.

### qmd in Tests

- MCP-Port: ein lokaler HTTP-Stub (`httptest`) spielt einmal gegen den echten Daemon aufgezeichnete
  Antworten zurück. Die Rohdaten des Spikes sind der erste Bestand.
- CLI-Port: der vorhandene `Runner`-Seam liefert aufgezeichnete Ausgaben von `qmd ls` und `qmd status`.
- Kein Test braucht eine GPU oder einen laufenden Daemon.

### Vergleichsklassen

- `catalog`, `read`, `neighbors`, `status` → **Daten** (stdout exakt).
- `search --profile full` → **Daten**.
- `search --profile fast` und `--profile keyword` → **Meldung** (Exit-Code), weil die Referenz dort
  die CLI-Pipeline fährt.
- stderr vergleicht der Korpus nie. Die Befunde von `search` und alle Fehlerwortlaute belegen darum
  Go-Unit-Tests mit exakten Zeichenketten.

### `status`

- Je Zeilenart eine synthetische Welt (Stempel fehlt/alt/frisch, mehrere Include-Globs, fehlender
  Pfad, kein `graph.json`, unaufgelöste Links unter der Hälfte, unauffindbares Dokument, qmd antwortet
  nicht, gleiche Bytes an zwei Orten, noch nicht durchsuchbar), gegen die Referenz aufgezeichnet.
- Einmal ein Live-Vergleich auf dieser Maschine: Python und loomux in derselben Minute gegen den echten
  Bestand, nebeneinander in der Paritätsliste. Nicht im Tor.

### Go-Tests und Tore

- Umgezogene Pakete bringen ihre Tests mit und werden auf 100 % je Funktion gehoben; jede Korrektur
  „nach Python" entsteht test-first; `covergate` im Pre-Commit-Tor wie bisher.
- Die Python-Referenz läuft nur beim Aufzeichnen, nie im Tor.

### Paritätsliste

`docs/.superpowers/parity/stufe-1b-1.md`, Form wie Stufe 1a. Bereits bekannt:
1. Rangfolge der Go-Suche: beantwortet durch den Spike, `fast` ohne LLM-Erweiterung.
2. `search --profile keyword`: Score 1/Rang und Zeile des besten Abschnitts statt BM25 und erster
   Fundstelle.
3. Befristetes Lesen aus `%LOCALAPPDATA%\brain` (bis Stufe 3) und der Manifest-Altnamen (bis Stufe 4).
4. Aufzeichnungen: `\r\n` im Python-stdout wird zu `\n` normalisiert, `PYTHONUTF8=1` gesetzt.
5. Aufwärm-Hinweis, wenn loomux den Daemon selbst startet — in der CLI-Referenz nicht vorhanden.
6. Backbone-Variablen: eine vom Nutzer gesetzte Variable gewinnt; die Referenz überschreibt sie.

## `dev mutants`

- `loomux dev mutants <paket>… [--only DATEI] [--family a1|a2|a3|a4] [--workers N]`.
- Familien wie `ub/tools/go_mutants.py`: (a1) ganze `if`-Bedingung auf `true` und `false`; (a2) jede
  Hälfte eines `&&`/`||` auf oberster Ebene; (a3) jeder Vergleichsoperator gekippt (`==`/`!=`, jede
  Ordnung gegen ihren Nachbarn), nicht in Kommentaren oder Zeichenketten; (a4) Bedingung negiert.
  `for`-Bedingungen werden nicht mutiert.
- Je Mutant: die mutierte Datei in ein temporäres Verzeichnis schreiben, eine Overlay-JSON
  `{"Replace": {"<absoluter Pfad der Quelle>": "<temporäre Datei>"}}` daneben, dann
  `go test -overlay <json> -count=1 -failfast -timeout 60s ./<paket>/`. N Arbeiter laufen parallel mit
  getrennten Overlays; der Arbeitsbaum wird nie beschrieben.
- Bericht: je Mutant `killed`, `SURVIVED` oder `no mutant`; am Ende Summen und die Liste der
  Überlebenden. Exit 0 nach einer vollständigen Runde, auch mit Überlebenden.
- Gezählt mit denselben Regeln: `brain/guard` 712, `hooks` 706, `config` 156, `cases` 231, `brain/wiki`
  200 Mutanten; ein Testlauf 5,2 s / 10,0 s / 0,8 s / 0,7 s / 1,1 s.
- Die Runde der Stufe läuft über `internal/brain/guard`, `internal/hooks`, `internal/config`,
  `internal/cases`, `internal/brain/wiki` und die neuen `internal/brain/*`-Pakete. Jeder Überlebende
  bekommt eine Zeile: nachgereichter Test oder begründet, warum der Code den Unterschied nicht sehen
  kann.

## Stufe 1b-1 ist fertig, wenn

1. alle übersetzten Fälle grün sind oder freigegeben in der Paritätsliste stehen;
2. die Coverage 100 % je Funktion ist, jede Ausnahme begründet;
3. die Mutationsrunde gelaufen ist, einschließlich der Entscheidungspakete aus 1a, und jeder
   Überlebende dokumentiert ist;
4. die Werte gemessen und in `docs/en/benchmarks.md` und `docs/de/benchmarks.md` eingetragen sind:
   `brain search` warm und kalt je Profil, `brain status`, `inittrace` vor und nach dem Einzug.
   Zielwert `loomux brain search --profile fast` warm **≤ 150 ms** Ende zu Ende, hergeleitet aus
   63–83 ms qmd, 5–17 ms Probe und Handshake und rund 35 ms Go-Startboden; das Lesen aller
   Identitätsregister gehört dazu und wird eigens ausgewiesen;
5. die fünf Befehle auf der echten Registry laufen (Rauchtest wie in 1a).

## Mensch-Schritte

- Die Bereiche aus `%LOCALAPPDATA%\brain\registry.toml` in `%LOCALAPPDATA%\loomux\registry.toml`
  übernehmen; der Controller liefert eine Vorlage. Das erweitert die Schreibschranke auf diese
  Bereiche, wie früher `brain guard`.
- Freigaben der Paritätsliste und der Rauchtest.

## Nicht in 1b-1

- `serve`, MCP über Streamable HTTP, stdio-Brücke, Kanalbindung, Upkeep, Job-Object-Frage unter einem
  MCP-Wirt — Stufe 1b-2. Dort auch die SDK-Version (die Fusions-Spec nennt `go-sdk v1.8.0`, auf der
  Maschine liegt nur v1.6.0 ohne vollständige Abhängigkeiten) und die Frage, ob der Kanal Adresse
  bleibt oder Feld der Anfrage wird.
- Wiki- und Doku-Umzug — Stufe 1b-3.
- `reindex`, `reconcile`, `pkg/index` außer `identity.go`, `graph/render.go`, `graph/synth.go`,
  `catalog/catalog.go` — Stufe 3. `loomux migrate` und die Umstellung der Wirte — Stufe 4.
- Die zurückgestellten Minor-Befunde aus Stufe 1a, außer 1b-1 fasst dieselbe Stelle ohnehin an.
