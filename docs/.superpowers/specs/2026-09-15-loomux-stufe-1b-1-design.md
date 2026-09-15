# loomux Stufe 1b-1 — die fünf Datenbefehle und `dev mutants`

**Datum:** 2026-09-15
**Stand:** entworfen, nicht umgesetzt
**Bezug:** [Fusions-Spec](2026-09-14-loomux-fusion-design.md), Zeile „1b" der Stufentabelle. Stufe 1b
wird dreigeteilt; diese Spec deckt 1b-1. 1b-2 (`serve`, MCP, Brücke, Upkeep) und 1b-3 (Wiki- und
Doku-Umzug) bekommen je eigene Specs.
**Messgrundlage:** Kontextaufnahme und qmd-Port-Spike vom 2026-09-15 (Rohdaten
`count-mutants.js`, `spike.py`, `raw/*.json` im Scratchpad der Sitzung; die tragenden Zahlen stehen
unten).

## Ziel

`loomux brain search|catalog|read|neighbors|status` beantworten dieselben Fragen wie
`brain search|catalog|read|neighbors|status` aus ultra-brain, mit belegter Parität zur
**Python**-Referenz, und `loomux dev mutants` misst, welche Entscheidungen der Code-Basis kein Test
bemerkt. Danach kann 1b-2 einen MCP-Dienst auf diese Befehle stellen, ohne sie neu zu erfinden.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Zuschnitt von 1b | Dreiteilung: 1b-1 Datenbefehle + `dev mutants`, 1b-2 `serve`/MCP/Brücke/Upkeep, 1b-3 Wiki- und Doku-Umzug. Je eigene Spec, eigener Plan, einzeln abnehmbar |
| Namensraum | `loomux brain search\|catalog\|read\|neighbors\|status`. `loomux status` bleibt die Hook-Inspektion aus 1a. Die MCP-Werkzeuge in 1b-2 heißen entsprechend `brain_search` usw. (passt zur [Code-Graph-Spec](2026-09-14-loomux-code-graph-design.md)) |
| qmd-Weg | Warmer qmd-MCP-Daemon über HTTP. `fast` = reine Vektorsuche `{searches:[{type:"vec"}], rerank:false}` ohne LLM-Erweiterung |
| Entstehung des Go-Codes | Umziehen, was nachweislich trägt, Rest neu nach der Python-Referenz (siehe Architektur) |
| Zustand | Eine Registry: `%LOCALAPPDATA%\loomux\registry.toml` für Schranke und Datenbefehle. Artefakte schreibgeschützter Bereiche und der Reconcile-Stempel werden befristet aus `%LOCALAPPDATA%\brain` gelesen |
| Parität von `status` | Synthetische Welten je Zeilenart im Tor, dazu einmal ein Live-Vergleich in der Paritätsliste |
| `dev mutants` | Paralleler Go-Port der vier Familien, mutiert nur Modulkopien; volle Runde inklusive der Entscheidungspakete aus 1a |

### Änderungen gegenüber der Fusions-Spec

- **„1b: neuer Go-Code, kein Umzug"** gilt nicht mehr uneingeschränkt. ultra-brains Go-`catalog`,
  `reader`, `graph` und `privacy` sind nahe Parität (gleiche Ausgabe, gleiche Privacy-Tore,
  byte-gleiche Meldungen), und der qmd-MCP-Port in `pkg/search` mappt die Profile korrekt. Diese Teile
  ziehen mit Tests um; neu entstehen nur, was fehlt oder falsch ist.
- **Harter Schnitt, befristet durchbrochen:** Bis Stufe 3 `reconcile` nach Go holt, pflegt die
  Python-Seite die Artefakte unter `%LOCALAPPDATA%\brain`. loomux liest sie dort lesend, statt eine
  Kopie zu führen, die veraltet. Die Ausnahme sitzt an einer Stelle und läuft mit Stufe 3 ab.
- **MCP-Werkzeugnamen:** Richtung `brain_*` statt der unpräfixierten Namen der Fusions-Spec; formal
  festgelegt in 1b-2.
- **Die offene Frage „woher die abweichende Rangfolge der Go-Suche kommt"** ist beantwortet (siehe
  Ausgangslage) und wird die erste Zeile der Paritätsliste.

## Ausgangslage, vermessen am 2026-09-15

### Die Python-Referenz

- Alle fünf Befehle sind Funktionen in `src/brain/core.py` (668 Zeilen): `search` :76, `catalog`
  :278, `read` :292, `neighbors` :316, `status` :328. `cli.py` verdrahtet nur (Parser :471-502,
  Drucker :388-429, Dispatch :712-808). `--channel` hat die Vorgabe `local` (`cli.py:451`).
- Die Referenz liegt im Tag-Worktree `C:\Users\micro\Documents\#GIT\loomux-src\ub` (Tag
  `loomux-1a-source`, `3cc72d2`) und läuft dort über das Python-Konsolenskript
  `uv run brain-mcp <befehl> …` (`pyproject.toml`: `brain-mcp = "brain.cli:main"`; der Name `brain`
  gehört dem Go-Binary auf dem PATH).
- `src/brain/search/qmd_mcp.py:289` hat am Tag einen Syntaxfehler
  (`except OSError, json.JSONDecodeError, SearchUnavailable:`). Die Python-CLI nutzt diesen Port
  nicht (`_port()` gibt den CLI-Port zurück, `cli.py:385`), die Aufzeichnung ist davon nicht
  betroffen.
- Kein vorhandener Paritätstest deckt die fünf Befehle
  (`tests/test_go_python_parity.py` prüft nur `check` und `lint`).

### Der Go-Stand in ultra-brain

| Paket | Code | Test | Befund |
|---|---:|---:|---|
| `pkg/catalog` | 141 | 178 | nahe Parität |
| `pkg/reader` | 101 | 168 | nahe Parität, byte-gleiche Verweigerungen |
| `pkg/graph` | 439 | 649 | nahe Parität (`neighbors`) |
| `pkg/privacy` | 193 | 237 | Kanäle, `never`-Globs |
| `pkg/search` | 1.278 | 2.599 | qmd-MCP-Port korrekt; kein Identitätsregister, zwei Befunde fehlen |
| `pkg/index` | 1.496 | 2.699 | reindex — Stufe 3 |
| `pkg/mcp` | 562 | 1.033 | handgerollt — 1b-2 neu |

Es gibt zwei unverträgliche Go-`status` (CLI-Stumpf: Arbeitsverzeichnis und Seitenzahl;
MCP-Dispatcher: Kanal, Zustandsverzeichnis, Bereichsliste); keiner entspricht der Referenz. Das
MCP-Schema bewirbt die Profile `fast, balanced, deep`, die es nicht gibt; ein unbekanntes Profil
wird still zu `full`.

### Warum Go und Python verschieden ranken (Spike)

Gegen die Collection `project-space`, vier Anfragen, je `keyword`, `fast`, `full`, `-n 5`, über
beide Wege:

- **Go mappt die Profile korrekt** (`pkg/search/mcp.go:184`, `formatArguments`, wie
  `qmd_mcp.py:169`). Die Differenz liegt in qmd 2.8.3: Unter „fast" laufen zwei Pipelines.
  `qmd vsearch` (Weg der Python-CLI) lässt ein LLM die Anfrage erweitern, sucht je Variante, führt per
  Maximum zusammen und schneidet unter 0,3 ab. MCP `searches:[vec]` sucht einmal und rankt nach
  Position (Score 1/Rang).
- **Beleg:** `qmd query "vec: <q>" --no-rerank` auf der CLI liefert Reihenfolge, Zeilen und Scores
  exakt wie MCP.
- `full`: auf beiden Wegen identisch (4 von 4). `keyword`: gleiche Reihenfolge (4 von 4), aber
  BM25-Score und erste Fundstelle (CLI) gegen 1/Rang und Zeile des besten Abschnitts (MCP). `fast`:
  gleiche Reihenfolge nur in 2 von 4.
- **Kosten:**

| Weg | keyword | fast | full |
|---|---:|---:|---:|
| CLI, Prozess je Anfrage (neue Anfrage) | 179–191 ms | 10,5–13,9 s | 20,0–24,1 s |
| MCP-Daemon warm | 5–17 ms | 63–83 ms | 4,2–7,9 s neu, ~250 ms wiederholt |
| MCP-Daemon frisch gestartet, erste Anfrage | 23 ms | 5,4 s | 11,9 s |

- Der Daemon lauscht nur auf `[::1]`; ein IPv4-Test meldet ihn fälschlich als geschlossen.
- Während der Messung stürzte ein vorhandener CUDA-Daemon mit `ggml-cuda.cu:106: CUDA error` ab.
  Ein Ereignis, kein Urteil; der Start in 1b-1 wählt Vulkan als Vorgabe.

### Laufzeitdaten

- Registry heute unter `%LOCALAPPDATA%\brain\registry.toml` (10 Bereiche); loomux' Registry kennt nur
  `project/loomux`.
- Je schreibgeschütztem Bereich unter `%LOCALAPPDATA%\brain\areas\<scope>`: `_identities.tsv`
  (`doc_id`, Pfad, `content_hash`, Revision), `graph.json`, `index.md`, `layout.json`. Schreibbare
  Bereiche tragen ihre Artefakte im Repo. Reconcile-Stempel:
  `%LOCALAPPDATA%\brain\maintenance\last-run.txt`.
- qmd-Index: `~/.cache/qmd/index.sqlite`; Collections heißen nach dem Scope, jedes Zeichen außerhalb
  `[A-Za-z0-9_.-]` wird `-` (`project/space` → `project-space`).
- Keine neue Modulabhängigkeit: die umziehenden Pakete brauchen nur stdlib, `BurntSushi/toml`,
  `gopkg.in/yaml.v3`, `golang.org/x/sys`, die loomux schon führt.

## Architektur

### Pakete

```
internal/brain/catalog    umgezogen aus ub pkg/catalog  (+ Tests)
internal/brain/reader     umgezogen aus ub pkg/reader   (+ Tests)
internal/brain/graph      umgezogen aus ub pkg/graph    (+ Tests)   → neighbors
internal/brain/privacy    umgezogen aus ub pkg/privacy  (+ Tests)   → Kanäle, never-Globs
internal/brain/search     qmd-MCP-Port umgezogen; neu: Identitätsregister, zwei Befunde
internal/brain/identity   neu: liest _identities.tsv
internal/brain/status     neu, Regel für Regel nach core.status
internal/cli/brain.go     loomux brain search|catalog|read|neighbors|status
internal/dev/mutants      neu: vier Familien, N Arbeiter auf Modulkopien
```

Umzugsregel wie in 1a: Code und Tests wortgleich bis auf Paketnamen, Importpfade, genannte Literale
und neue Tests für 100 %; jede weitere Änderung ist ein Befund im Task-Bericht.

### Abhängigkeitsregeln

- `brain/*` darf `config` benutzen und einander; sonst nichts aus loomux.
- `cli` ruft `brain/*`.
- `hooks` importiert kein `brain/*` außer dem, was es seit 1a nutzt (`brain/guard`, `brain/wiki`).
  Der Pfad an jedem Edit zieht keinen der neuen Befehle nach.

### Die befristete Ausnahme

`internal/config` bekommt neben `ManifestDir` genau eine Funktion, die für schreibgeschützte Bereiche
das Artefaktverzeichnis unter `%LOCALAPPDATA%\brain\areas\<flat-scope>` und den Reconcile-Stempel
unter `%LOCALAPPDATA%\brain\maintenance\` findet. Ihr Name trägt den Ablauf („bis Stufe 3"), ihr
Test prüft, dass sie nur für schreibgeschützte Bereiche antwortet. Stufe 3 löscht sie mit `reconcile`
in Go. Für Tests und Fallwelten überschreibt `LOOMUX_LEGACY_BRAIN_DIR` den Ort, so wie
`LOOMUX_STATE_DIR` das Zustandsverzeichnis; die Variable entfällt mit der Funktion.

### qmd

- `brain search` und `brain status` finden den Daemon auf `localhost:8765`, geprüft auf IPv4 **und**
  `[::1]`, per TCP-Probe und MCP-`initialize`-Handshake.
- Antwortet keiner, startet loomux ihn wie die Python-Seite (`qmd mcp --http --daemon`), entkoppelt
  vom eigenen Prozess, Vorgabe-Backbone Vulkan (per Umgebung änderbar wie `QMD_LLAMA_GPU`).
- In 1b-2 übernimmt `serve` diese Rolle; 1b-1 baut den Port so, dass `serve` ihn ohne Umbau hält.

### Startzeit

Kein umgezogenes und kein neues Paket parst in `init()` oder in einer Paketvariable Daten. Nachweis
mit `GODEBUG=inittrace=1 loomux --version` vor und nach dem Einzug; der Befund steht im
Benchmark-Eintrag.

## Befehle, Ausgabe, Datenfluss

Alle fünf nehmen `--channel local|cloud` (Vorgabe `local`). Ein Bereich mit
`[privacy] mode = "local_only"` existiert im Kanal `cloud` nicht; `never`-Globs gelten in jedem Kanal.

| Befehl | stdout |
|---|---|
| `loomux brain search <anfrage> [--scope S] [--profile keyword\|fast\|full] [-n N]` | je Treffer `brain://{scope}/{pfad}:{zeile}  {score}%  {titel}`, darunter die Snippet-Zeilen mit vier Leerzeichen, dann eine Leerzeile; ohne Treffer genau `no matches`. Befunde auf stderr als `note: …` |
| `loomux brain catalog [--scope all\|S]` | `# brain`, Leerzeile, je sichtbarem Bereich `* [scope](brain://scope/)`, nach Scope sortiert; ein benannter Bereich gibt sein `index.md` byte-gleich aus |
| `loomux brain read <scope> <pfad> [--section S]` | die Datei oder ein Abschnitt, unverändert |
| `loomux brain neighbors <scope> <pfad>` | genau zwei Zeilen: `incoming: a, b` und `outgoing: c`, `-` wenn eine Richtung leer ist |
| `loomux brain status` | nach `core.status`, in dieser Reihenfolge: Reconcile-Stempel (immer); je sichtbarem Bereich Include-Glob-Hinweis, fehlender Pfad, „never indexed", „nur X von Y Links aufgelöst" (unter der Hälfte), unauffindbare Dokumente; danach gleiche Bytes an zwei Orten und noch nicht durchsuchbare Dokumente |

Befunde von `search`: „Treffer steht nicht im Identitätsregister" und „letzter vollständiger
Reconcile älter als 24 Stunden" (nur bei vorhandenem, gealtertem Stempel; ein fehlender Stempel steht
nur in `status`).

Datenfluss: loomux-Registry → sichtbare Bereiche je Kanal → Artefaktverzeichnis → `_identities.tsv`,
`graph.json`, `index.md` → für `search` und zwei `status`-Zeilen der qmd-Daemon.

Die Ratschläge in den Meldungen nennen `brain reindex` und `brain reconcile` — bis Stufe 3 sind das
die Python-Befehle, die der Nutzer tatsächlich hat. Stufe 3 schreibt sie auf `loomux brain …` um.

## Fehlerverhalten

- **Usage-Fehler** (fehlendes Argument, unbekanntes `--profile`, unbekannter Kanal) → Exit 2, die
  Meldung nennt den Befehl.
- **Verweigerungen** (Bereich im Kanal unsichtbar, Pfad verlässt den Bereich, `never`-Glob,
  Prüfstelle im Kanal `cloud`) → Exit 1, Grund auf stderr. Wo die Referenz einen Wortlaut hat, gilt
  er; jede abweichende Exit-Code-Zuordnung ist eine Paritätszeile.
- **Nie leer statt kaputt:** Ist qmd nicht erreichbar und nicht startbar, antwortet `search` nicht
  mit `no matches`, sondern mit Exit 1 und einer Meldung, die den gescheiterten Schritt nennt (nicht
  installiert, Port geschlossen, Handshake gescheitert) samt Start- oder Installationsbefehl.
  `status` schreibt dafür eine eigene Zeile.
- **Aufwärmen:** Solange der Daemon das Modell lädt, schreibt `search` vorher einen Hinweis auf
  stderr; bei warmem Daemon nicht.
- **Kaputte Registry** → Exit 1 mit Dateinamen. Dieselbe Datei liest die Schreibschranke, die daran
  geschlossen scheitert.
- **Fehlende Artefakte:** Ohne `graph.json` meldet `status` „never indexed; run `brain reindex`",
  `neighbors` scheitert mit demselben Rat. Fehlt für einen schreibgeschützten Bereich das
  Übergangsverzeichnis, nennt die Meldung beide Orte, an denen gesucht wurde.
- **Unlesbares Identitätsregister:** `search` liefert die Treffer und schreibt einen Befund, dass die
  Registerprüfung ausfiel.
- **`dev mutants`:** Ist die Suite vor der Runde nicht grün → Exit 2, keine Runde. Ein Mutant, der
  nicht kompiliert, zählt gesondert; einer, der die Zeitgrenze reißt, gilt als getötet. Modulkopien
  werden auch bei Abbruch aufgeräumt; der Arbeitsbaum wird nie berührt.

## Paritätsnachweis und Tests

### Fallkorpus

- Unter `testdata/cases/1b-1/`, Format und Werkzeuge wie in 1a (`record-case`, `import-cases`,
  Runner im Prozess, Fallzahl in der Suite festgenagelt).
- Aufgezeichnet gegen die Python-Referenz am Tag (`uv run brain-mcp <befehl> …` im ub-Worktree).
  `record-case` bekommt dafür eine Form, die ein Programm mit Argumenten statt eines einzelnen
  Binaries aufruft.
- Übersetzt: Präfix `brain-mcp ` → `loomux brain `, Zustandsverzeichnis umgebogen
  (`BRAIN_STATE_DIR` → `LOOMUX_STATE_DIR` plus `LOOMUX_LEGACY_BRAIN_DIR` für die befristete
  Ausnahme).
- Fälle je Befehl: Erfolg, jede Verweigerung und jeder Fehlerweg aus dem Abschnitt Fehlerverhalten.

### qmd an der Prozessgrenze

Echte MCP-Antworten werden einmal gegen den laufenden Daemon aufgezeichnet und in Fallsuite und
Go-Tests von einem lokalen HTTP-Stub (`httptest`) zurückgespielt — deterministisch, ohne GPU, im Tor
lauffähig. Die Rohdaten des Spikes sind der erste Bestand.

### Vergleichsklassen

- `catalog`, `read`, `neighbors`, `status` → **Daten** (stdout exakt).
- `search --profile full` → **Daten**; auf beiden qmd-Wegen gemessen identisch.
- `search --profile fast` und `--profile keyword` → **Meldung** (Exit-Code und Welt), weil die
  Referenz dort die CLI-Pipeline fährt. Zwei Paritätszeilen, Beleg ist der Spike.

### `status`

- Je Zeilenart eine synthetische Welt (Stempel fehlt/alt/frisch, mehrere Include-Globs, fehlender
  Pfad, kein `graph.json`, unaufgelöste Links unter der Hälfte, unauffindbares Dokument, gleiche Bytes
  an zwei Orten, noch nicht durchsuchbar), gegen die Referenz aufgezeichnet.
- Einmal ein Live-Vergleich auf dieser Maschine: Python und loomux in derselben Minute gegen den
  echten Bestand, Ausgabe nebeneinander in der Paritätsliste. Nicht im Tor.

### Go-Tests und Tore

- Umgezogene Pakete bringen ihre Tests mit und werden auf 100 % je Funktion gehoben; neuer Code
  entsteht test-first; `covergate` im Pre-Commit-Tor wie bisher.
- Die Python-Referenz läuft nur in den alten Repos, nie in loomux.

### Paritätsliste

`docs/.superpowers/parity/stufe-1b-1.md`, Form wie Stufe 1a. Bereits bekannt:
1. Rangfolge der Go-Suche: beantwortet durch den Spike, `fast` ohne Erweiterung.
2. `search --profile keyword`: Score 1/Rang und Zeile des besten Abschnitts statt BM25 und erster
   Fundstelle.
3. Befristetes Lesen aus `%LOCALAPPDATA%\brain` bis Stufe 3.

Keine Abweichung, nur vermerkt: Ein unbekanntes `--profile` endet mit Exit 2 — so wie `argparse` in
der Referenz; ultra-brains Go-Fassung machte still `full` daraus.

## `dev mutants`

- `loomux dev mutants <paket>… [--only DATEI] [--family a1|a2|a3|a4] [--workers N]`.
- Familien wie `ub/tools/go_mutants.py`: (a1) ganze `if`-Bedingung auf `true` und `false`; (a2) jede
  Hälfte eines `&&`/`||` auf oberster Ebene; (a3) jeder Vergleichsoperator gekippt (`==`/`!=`, jede
  Ordnung gegen ihren Nachbarn); (a4) Bedingung negiert. `for`-Bedingungen werden nicht mutiert.
- Jeder Arbeiter hält eine eigene Kopie des Moduls (temporäres Verzeichnis), setzt einen Mutanten ein,
  fährt `go test -count=1 -failfast -timeout 60s ./<paket>/` und stellt die Datei zurück.
- Bericht: je Mutant `killed`, `SURVIVED` oder `no mutant`; am Ende Summen und die Liste der
  Überlebenden. Exit 0 nach einer vollständigen Runde, auch mit Überlebenden — beurteilt werden sie
  von Hand.
- Gezählt am 2026-09-15 mit denselben Regeln: `brain/guard` 712, `hooks` 706, `config` 156, `cases`
  231, `brain/wiki` 200 Mutanten; ein Testlauf 5,2 s / 10,0 s / 0,8 s / 0,7 s / 1,1 s — seriell rund
  drei Stunden Obergrenze, mit acht Arbeitern rund 25–40 Minuten.
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
   63–83 ms qmd, 5–17 ms Probe und Handshake und rund 35 ms Go-Startboden;
5. die fünf Befehle auf der echten Registry laufen (Rauchtest wie in 1a).

## Mensch-Schritte

- Die Bereiche aus `%LOCALAPPDATA%\brain\registry.toml` in `%LOCALAPPDATA%\loomux\registry.toml`
  übernehmen; der Controller liefert eine Vorlage. Das erweitert die Schreibschranke auf diese
  Bereiche, wie früher `brain guard`.
- Freigaben der Paritätsliste und der Rauchtest.

## Nicht in 1b-1

- `serve`, MCP über Streamable HTTP, stdio-Brücke, Kanalbindung, Upkeep, Job-Object-Frage unter einem
  MCP-Wirt — Stufe 1b-2. Dort auch: SDK-Version (die Spec nennt `go-sdk v1.8.0`, auf der Maschine
  liegt nur v1.6.0, dessen Abhängigkeiten nicht vollständig im Cache sind) und die Frage, ob der Kanal
  Adresse bleibt (Python-Entscheidung) oder Feld der Anfrage wird (Fusions-Spec).
- Wiki- und Doku-Umzug — Stufe 1b-3.
- `reindex`, `reconcile`, `pkg/index` — Stufe 3. `loomux migrate` — Stufe 4.
- Die zurückgestellten Minor-Befunde aus Stufe 1a, außer 1b-1 fasst dieselbe Stelle ohnehin an.

## Offen und vor dem Bau zu klären

- Der genaue Installations- und Startbefehl von qmd auf dieser Maschine (Launcher-Logik in
  `ub/src/brain/search/qmd.py` und `pkg/search`: Node, Shim-Umgehung, Umgebungsvariablen) — der Plan
  liest ihn aus dem Code, statt ihn zu raten.
- Die Exit-Codes der Python-Referenz je Verweigerung (`AccessDenied`, `SearchUnavailable`,
  `GraphError`, `KeyError`): die Aufzeichnung legt sie fest; jede Abweichung von der Konvention „Exit 1"
  wird eine Paritätszeile.
- Die genaue Semantik der `status`-Zeilen „unauffindbar" und „noch nicht durchsuchbar" gegen den
  MCP-Port (Python fragt sie über den CLI-Port); der Plan misst das, bevor er die Welten baut.
