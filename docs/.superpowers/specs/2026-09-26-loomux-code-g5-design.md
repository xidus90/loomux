# loomux G5 — Mehrsprachige Extraktion

**Datum:** 2026-09-26  
**Stand:** entworfen am 2026-09-26; die Abschnitte hat der Nutzer im Brainstorming einzeln
freigegeben (Entscheidungen in §10), die Umsetzung von G5a am selben Tag. Nach zwei Reviews am
selben Tag ergänzt: Leser des Graphen (§6), Parser-Version im Cache-Schlüssel, Body-Text im Cache,
die Graph-Lane für Python, ein Namensindex je Sprache, Konstruktor- und Dunder-Regel, Abnahme nur
an Kopien. Während der Umsetzung am selben Tag nachgeschärft: Cache-Hinweise und Bericht (§5), die
Python-Regeln, wo die Umsetzung sie genauer fassen musste, und zwei offene Punkte (§7). **G5a
umgesetzt und abgenommen am 2026-09-26** (Akte `docs/.superpowers/parity/code-g5.md`, Messung in
`docs/en/benchmarks.md`). Vor dem Merge ergänzt: Modulpfade lösen über die Quellwurzeln der
importierenden Datei auf (§7, E7).  
**Ergänzt:** [`2026-09-14-loomux-code-graph-design.md`](2026-09-14-loomux-code-graph-design.md)
§3.1, §3.2, §6.2, §9 und §10 (Zeile G5) sowie §12.1 von
[`2026-09-17-loomux-code-g2-design.md`](2026-09-17-loomux-code-g2-design.md). Wo sich die Dateien
widersprechen, gilt diese.  
**Arbeitsort:** Branch `feat/graph-python`, abgezweigt von `master` `b5c99cf1`, Worktree
`.claude/worktrees/g5-python`.

## 1. Warum es dieses Delta gibt

Die Säule-3-Spec plant G5 als Tree-sitter über WebAssembly: `wazero` als Laufzeit, Grammatiken mit
`wasi-sdk` nach `wasm32-wasi` gebaut, ein AOT-Cache unter `%LOCALAPPDATA%`. Den offenen Messpunkt
dazu (§11: „läuft `tree-sitter-typescript` unter `wasi-sdk` ohne JS-Host-Importe?“) hat nie jemand
eingelöst. Eine Probe am 2026-09-25/26 hat stattdessen einen Weg gefunden, der ganz ohne
WebAssembly auskommt (§2). Außerdem hat der Nutzer die Sprachen neu gesetzt: statt TypeScript,
Python und Rust jetzt **Python, TypeScript/TSX, GDScript und C++**, von einfach nach schwer. Go
behält seinen nativen Extraktor.

Dazu kommen Befunde am Code von `master`, die jede zweite Sprache betreffen:

- `query.Extract` (`internal/code/query/build.go`) parst bei **jedem** Build alle Dateien neu.
  Für Go reichen dafür ~45 ms über dieses Repo. Tree-sitter braucht auf den Abnahme-Repos 2–4 s
  (§2), und die Frage-Pfade (`ask.EnsureFresh`) bauen bei Drift neu.
- Ein Fehler aus `golang.File` bricht den ganzen Build ab. Tree-sitter liefert immer einen Baum,
  im Zweifel mit Fehlerknoten; ein Abbruch bei jedem davon machte Python-Repos unbaubar.
- `golang.Version` ist fest verdrahtet in `query/build.go`, `check.go`, `refresh.go`, `ask.go`,
  `load.go`, in `resolve.Graph` und im Blast-Monitor (`internal/hooks/blast_monitor.go`).
- `internal/hooks` importiert `extract/golang` (Blast-Monitor). Ein Tree-sitter-Extraktor, der am
  selben Paket hinge, holte die Laufzeit in den Hook-Pfad.
- `blast.IsTestPath` kennt nur `_test.go`; `parity/code-g4.md` §3 vertagt die übrigen Sprachen
  ausdrücklich auf G5.

## 2. Die Probe

Wegwerfcode im Scratchpad der Sitzung, gemessen am 2026-09-25/26 auf AMD Ryzen 7 9800X3D,
Windows 11, Go 1.27.0, `CGO_ENABLED=0`.

**Recherche.** `github.com/malivvan/tree-sitter` (Tree-sitter über `wazero`) ist eine
Vorabversion mit fünf Sternen und ohne Liste der Grammatiken. `github.com/odvcencio/gotreesitter`
(MIT, v0.55.0) ist ein Nachbau der Tree-sitter-Laufzeit in reinem Go: er liest die Parse-Tabellen
der Upstream-Grammatiken, 206 Grammatiken, externe Scanner in Go, Queries und ein `Tagger`.
Abhängigkeiten: `golang.org/x/sync` und `gopkg.in/yaml.v3`, beide schon in `go.mod`. Die vier
Sprachen gibt es als eigene Pakete `grammars/python`, `grammars/typescript`, `grammars/tsx`,
`grammars/gdscript`, `grammars/cpp`.

**Parsen** (Dateien über 1 MB ausgelassen wie in `sourceset`; Dot-Verzeichnisse, `node_modules`,
`venv` übersprungen):

| Repo | Sprache | Dateien | mit Fehlerknoten | Summe | Grammatik laden |
|---|---|---:|---:|---:|---:|
| `iam_backend` | Python | 399 | 0 | 2,3 s | 11 ms |
| `ultra-brain` | Python | 182 | 0 | 2,0 s | 12 ms |
| `iam_frontend` | TypeScript | 160 | 0 | 1,3 s | 62 ms |
| `iam_frontend` | TSX | 322 | 0 | 2,0 s | 29 ms |
| `space` | GDScript (mit Addons) | 653 | 3 | 4,2 s | 2,5 ms |
| PrusaSlicer `libslic3r` | C++ | 397 | 193 | 3,6 s | 37–57 ms |

Einzelne Dateien brauchen bis 1,4 s (ein 1-MB-`codes.py`), typisch sind 0,4–1,1 ms je KB, bei
TypeScript 1,9 ms. Gegen die
C-Laufzeit (`web-tree-sitter` 0.25.10 in node, also selbst WASM) ist `gotreesitter` bei Python 2×,
bei C++ 4,7× langsamer.

**C++ gegen die Referenz.** Mit `tree-sitter-cpp` 0.23.4 meldet die C-Laufzeit 55 der 397 Dateien
mit Fehlerknoten, `gotreesitter` 193; 140 davon nur bei `gotreesitter`. Gegen eine ältere
Grammatik (`tree-sitter-wasms` 0.1.13) war das Bild ähnlich einseitig (103 nur bei
`gotreesitter`). Die Lücke liegt darum wahrscheinlich an der Laufzeit, nicht an der
Grammatikversion — belegt ist das nicht, denn den Stand, den `gotreesitter` selbst einbettet
(`tree-sitter-cpp` `8b5b49eb`, kein Tag), hat die Probe nicht gegen die C-Laufzeit gefahren. Ob die
Lücke Kanten kostet, sagt die Zahl der Dateien nicht; das prüft G5d (§9).

**Kosten im Binary.** loomux mit einem Import aller drei Grammatiken Python, GDScript, C++
(Scratch-Worktree, Parser erreichbar, damit der Linker ihn behält):

| | vorher | nachher |
|---|---:|---:|
| Größe `loomux.exe` | 24,3 MB | 36,6 MB |
| Summe `init` (`GODEBUG=inittrace=1 --version`) | 2,49 ms, 141 Pakete | 4,47 ms, 148 Pakete |
| `--version` warm-min (`dev bench-hooks`, 3 × 50) | 13,5–17,0 ms | 16,3–19,5 ms |

`hook pre-tool-use` lag im Rauschen einer belasteten Maschine (Mediane 20–60 ms statt der üblichen
~10 ms); die Nachmessung auf ruhiger Maschine ist Teil von G5a (§8). Die Größe steckt in der
Laufzeit selbst: eine auf drei Sprachen gestutzte Kopie des Moduls sparte 0,5 MB, Build-Tags
(`grammar_subset_*`) ebenso wenig. Die Init-Kosten fallen bei **jedem** Aufruf an, auch bei Hooks —
eine Importtrennung schützt davor nicht, es ist ein Binary.

**Folgerung.** Kein `wazero`, kein WASM-Build, kein AOT-Cache, keine C-Werkzeugkette.
`gotreesitter` als direkte Abhängigkeit (§10, E1).

## 3. Umfang und Schnitt

G5 ist **nur Extraktion**. `graph viz` führt die Migrationstabelle unter W3 und bleibt dort.

| Stufe | Inhalt | Abnahme |
|---|---|---|
| **G5a** | Extraktor-Schnittstelle, Kern für Tree-sitter, Mehrsprachigkeit in `sourceset`, kombinierte Version, Cache je Datei, CGo-Freiheitstor; dazu **Python** | `iam_backend`, `ultra-brain` |
| **G5b** | **TypeScript/TSX** | `iam_frontend` |
| **G5c** | **GDScript** | `space` |
| **G5d** | **C++**, erst nach bestandener Recall-Prüfung (§9) | `nano-coverage-godot` |

Jede Stufe ist ein eigener Plan und ein eigener Pull Request; G5b bis G5d bekommen je ein kurzes
Delta, wenn sie dran sind. Diese Spec legt G5a fest und für die übrigen nur die Leitplanken.

## 4. Pakete und Schnittstelle (G5a)

```
internal/code/extract/
├── extract.go        # Language, Result, RawEdge, Import — keine Sprache, keine Fremdabhängigkeit
├── all/              # die Liste der Sprachen und ihre kombinierte Version
├── golang/           # Logik unverändert; erfüllt Language
├── treesitter/       # gemeinsamer Kern, einziger Ort neben den Sprachpaketen, der gotreesitter kennt
└── python/           # Tabelle der Knotentypen, Owner- und Import-Regeln
```

**`extract.Language`:**

```go
type Language interface {
	Name() string         // "go", "python"
	Version() string      // "go/1", "python/1"
	Extensions() []string // ".go"; ".py"
	File(rel, source string) (Result, error)
}
```

`Result` bekommt gegenüber `golang.Result` die Felder `Language string` und `ParseErrors int`;
`Package` bleibt (Go füllt es, Python lässt es leer). `RawEdge` und `Import` ziehen unverändert von
`golang` nach `extract`. Ihre Felder tragen Python schon: `Owner` für `self.foo()`, `Receiver` für
`modul.foo()`, `Specifier` für den Importpfad.

**`extract/all`** hält eine feste Liste `[]extract.Language{golang.Language{}, python.Language{}}`,
ohne `init()` und ohne Registrierung zur Laufzeit (Projektregel). `all.Version()` ist die sortierte,
mit `+` verbundene Kette der Sprachversionen (`go/1+python/1`); `all.For(rel)` wählt nach Endung.

**`extract/treesitter`** ist der Kern aus Ansatz 3 (§10, E3): Baumlauf über eine Tabelle je Sprache
(welche Knotentypen Definitionen, Aufrufe, Importe sind), Span und Signatur aus Byte-Offsets,
Body-Hash wie beim Go-Extraktor, ID-Minting `rel#Qualifiziert` mit Ordinal bei Doppeln, Zählen und
Überspringen von Fehlerknoten. Die Grammatik lädt beim ersten `File` der Sprache, nie auf
Paketebene: `grammars/python.Language()` cached sie selbst, darum kein eigenes `sync.Once`. Ein
Parser je Datei, weil `gotreesitter`s Parser nicht nebenläufig nutzbar ist und eine Paketvariable
Zustand wäre.

**Parser-Version.** `python/1` ändert sich nur, wenn loomux' Extraktor sich ändert. Ein Update von
`gotreesitter` kann aber andere Bäume liefern, ohne diese Kennung zu berühren; der Cache (§5)
lieferte dann alte Ergebnisse, und `Meta.Extractor` sähe frisch aus. Darum trägt
`extract/treesitter` eine Konstante `Parser = "gotreesitter/v0.55.0"`, und jede Tree-sitter-Sprache
hängt sie an ihre Version an (`python/1@gotreesitter/v0.55.0`). Ein Test liest `go.mod` und schlägt
fehl, wenn die Konstante nicht zur dort gepinnten Version passt. So kann niemand die Abhängigkeit
heben, ohne dass Cache und Graph neu gebaut werden. `runtime/debug.ReadBuildInfo` wäre die
Alternative ohne Konstante, liefert im Testbinary aber nicht verlässlich dieselbe Antwort wie im
Pilot-Binary.

**`resolve`** trennt die Sprachregeln: `resolve/golang.go` (die heutige Logik), `resolve/python.go`.
`resolve.Graph` nimmt `[]extract.Result`, teilt nach `Language` und baut **je Sprache einen eigenen
Index** (`repoIndex` mit `global`, `byOwner`, `perFile` …). Ein gemeinsamer Index bräche die
Go-Auflösung in gemischten Repos still: der Rückfall „ein im Repo eindeutiger Name“
(`resolveCall`, Zweig `default`) fände neben der Go-Funktion eine gleichnamige Python-Funktion und
verwürfe die Kante; `byOwner` mischte eine Go-Methode `T.Run` mit einer Python-Methode `T.Run`.
Jede Kante wird nur innerhalb ihrer Sprache aufgelöst, Kanten über Sprachgrenzen gibt es nicht.
Ein gemischtes Golden (§8) mit absichtlich gleichen Namen in Go und Python hält das fest.

**Meta.** `Meta.Extractor` wird `all.Version()`. Das Feld bleibt ein String, das Schema bleibt 2:
Knoten und Kanten behalten ihre Form, und die neue Kennung erzwingt den Neubau jedes alten Graphen
ohnehin. `Meta.Languages` zählt die Sprachen auf, die tatsächlich Dateien haben.

**Abhängigkeitsgrenze.** Nur `extract/treesitter` und die Tree-sitter-Sprachpakete importieren
`gotreesitter`. Ein Test (nach dem Muster von `TestHooksNeverImportTheInstaller`) hält
`internal/hooks` davon frei; `hooks` importiert weiter nur `extract/golang`.

## 5. Datenfluss, Cache, Frische (G5a)

**Dateimenge.** `sourceset` kennt eine feste Liste der Endungen statt nur `.go`. Es importiert
`extract` nicht, damit der Hook-Pfad nicht am Parser hängt (derselbe Grund wie im Paketkommentar
von `sourceset`). Ein Test stellt sicher, dass die Liste gleich der Vereinigung von `Extensions()`
über `all` ist. Die übersprungenen Verzeichnisse bleiben; Dot-Verzeichnisse (`.venv`, `.godot`)
überspringt `SkipDir` schon heute.

**Cache je Datei.** `.loomux/state/graph/cache/extract.json` hält `rel → {version, sha256,
result, bodies}`. `bodies` ist ein eigenes Feld (Knoten-ID → Body-Text): `model.Node.BodyText`
trägt `json:"-"`, ein JSON-Cache von `Result` verlöre ihn still, und `lexicon.Build` bekäme bei
einem warmen Build Knoten ohne Rumpf-Tokens — `ask` würde nur warm schlechter, ohne Fehler. Den
Beweis führt die Abnahme 3 in §8. `query.Extract` liest und hasht weiter jede
Datei — der Grundsatz im Kommentar dort („a stat may decide whether a query rebuilds; it may not
decide what the rebuild looks at“) bleibt — und parst nur, wo Hash oder Sprachversion abweichen.
`resolve.Graph` läuft immer über alle Ergebnisse, weil die Auflösung dateiübergreifend ist.

- Fehlt der Cache: ein kalter Start ohne Hinweis. Ein erster Build hat keinen, das ist keine
  Nachricht.
- Ist er nicht lesbar oder trägt er eine andere Formatversion: alles parsen, ein Hinweis über
  `notice` („extract cache ignored, parsing every file: …“), kein Fehler. Ein Eintrag mit anderem
  Hash oder anderer Sprachversion wird still neu geparst; der Eintrag einer Datei, die es nicht mehr
  gibt, fällt mit dem nächsten Build heraus.
- Ein Cache, der nicht geschrieben werden kann: Hinweis, der Build steht (wie der
  Frische-Datensatz heute).
- `graph check` liest den Cache und schreibt ihn nie; ein unlesbarer Cache kostet es nur Parsezeit
  und bleibt still.
- `graph build --no-reuse` erzwingt den Vollbau (das Flag nennt die Säule-3-Spec §7 schon).
- Ob JSON als Format reicht, misst G5a an der Cache-Größe von `iam_backend`; bei Bedarf wird es
  ein anderes Format, ohne dass sich der Vertrag ändert.

**Frische.** `freshness.Write`, `check`, `refresh`, `ask.EnsureFresh` und `load` bekommen
`all.Version()` statt `golang.Version`. Ein Graph mit anderer Kennung ist „foreign“, wie heute:
`graph check` meldet ihn als FOREIGN, `RefreshGraph` (`query/refresh.go`) baut ihn neu. Die Lane
`check graph-fresh` ruft `RefreshGraph` und bleibt darum beim ersten Lauf nach dem Update grün
(„graph rebuilt“); rot wird sie davon nicht.

**Blast-Monitor.** Er vergleicht heute `g.Meta.Extractor != golang.Version`. Mit der kombinierten
Kennung schwiege er für immer. Er prüft künftig, ob `golang.Version` ein Glied der Kette in
`Meta.Extractor` ist, und importiert dafür `extract/all` **nicht**. Er bleibt Go-only: das Laden
einer Grammatik (bis 62 ms) und Einzeldateien bis 1,4 s passen in kein Hook-Budget. Eine
Python-Datei, die ein Edit ändert, macht den Graphen veraltet; das repariert der nächste Build.

**Parsefehler.** Definitionen innerhalb eines Fehlerknotens und wiederhergestellte Definitionen
(§7) werden übersprungen, die Datei bleibt
mit ihrem Dateiknoten im Graphen, `Result.ParseErrors` zählt die Fehlerknoten (ERROR und MISSING)
und einen mehr, wenn der Parser vorzeitig anhielt (`Tree.ParseStoppedEarly`: ein Limit, eine
Zeitgrenze, ein Abbruch) — der Baum hält dann nur, was er bis dahin erreicht hat, und der Rest
muss nicht als ERROR erscheinen. Die Zahl bleibt eine Untergrenze: ein wiederhergestellter Baum
kann Text still verlieren. `graph build`
meldet die Summe je Sprache und die ersten fünf Dateien; der Exit-Code bleibt 0. `golang.File`
bricht bei einem Syntaxfehler weiter ab — das zu ändern wäre eine Verhaltensänderung außerhalb
von G5.

**Bericht.** `graph build` zeigt je Sprache eine Zeile mit Dateien, neu geparst, wiederverwendet
und Parsefehlern, bei Parsefehlern darunter die ersten fünf Dateien. Die erste Zeile zählt die
Kanten je Relation und nennt `extends` nur, wenn es welche gibt: ein Go-Graph hat keine, und sein
Bericht bleibt byte-gleich mit dem vor G5a. `graph stats` bleibt
unverändert: es zählt die Sprachen schon nach Endung aus dem Graphen, und Wiederverwendung und
Parsefehler sind Eigenschaften eines Builds, nicht des Graphen. Beides zeigt nur `graph build`.

**Nicht in G5a.** Das Flag `--languages` für `graph build` (Säule-3-Spec §7) und ein Abschnitt
`[graph]` mit `languages`, `exclude`, `max_depth` in `.loomux/config.toml`. Beide braucht keine
Abnahme; die Sprache folgt aus der Endung, der Ausschluss aus `sourceset`.

## 6. Die Leser des Graphen (G5a)

Extraktion und Auflösung sind nur die Hälfte; die Abnahme 2 (§8) hängt an dem, was den Graphen
liest. Durchgesehen am 2026-09-26 auf `.go`, `_test.go`, `golang.`, Export-Prüfungen und
Bezeichner-Zerlegung:

| Leser | Stand | Änderung in G5a |
|---|---|---|
| `blast.IsTestPath` (`blast/radius.go`), genutzt von `blast.Radius` und `query/audit.go` | nur `_test.go` | Tabelle nach Endung (§7) |
| `resolve.isTestFile` (`resolve/resolve.go`) | nur `_test.go` | gehört zur Go-Auflösung und zieht mit nach `resolve/golang.go`; Python braucht es nicht |
| `blast/resolve.go`: Pfad eines Diff-Hunks | kennt `.py`, `.ts`, `.tsx`, `.cpp`, `.h` … schon | keine |
| `lexicon.Tokenize` | trennt an camelCase **und** an jedem Nicht-Alnum-Zeichen, also auch an `_` | keine; `snake_case` zerfällt schon in Tokens |
| `repomap.langFromPath` | kennt `.py`, `.ts`, `.tsx` | keine |
| `query/{build,check,refresh,ask,load}.go` | `golang.Version`, `golang.Result` | `all.Version()`, `extract.Result` (§4, §5) |
| `hooks/blast_monitor.go` | `golang.Version`-Gleichheit | Glied der Kette (§5) |
| `grep`, `skeleton`, `internal/serve/graph` | nichts Go-Eigenes gefunden | keine |
| `[verify]`, `presets.toml` | eine Lane `graph` gibt es nur unter `[stack.go.graph]`; `Plan` legt je Stapel einen Graph-Job an (Bereich `.`) | siehe unten |

**Die Graph-Lane für Python.** Ohne Eintrag im Preset hätte ein reines Python-Repo keine
Graph-Lane. Ein zweiter Eintrag `[stack.python.graph]` mit denselben Befehlen gäbe in einem
gemischten Repo (`ultra-brain` hat Go und Python) zwei Jobs, die denselben Graphen neu bauen. Der
Graph gehört der Wurzel, nicht dem Stapel: G5a ergänzt das Preset um `[stack.python.graph]` mit
denselben Befehlen, und `Plan` legt für die Art `graph` **einen** laufenden Job je Lauf an. Ihn
trägt der erste Stapel in Byte-Reihenfolge, der einen Graph-Befehl hat. Jeder weitere Stapel mit
Graph-Befehl bekommt `not-applicable` mit dem Grund `graph covered by graph/<stapel>`. Ein reines
Go-Repo sieht damit denselben Bericht wie heute (`graph/go` läuft, `graph/shell` ohne Befehl);
neu ist die Zeile nur in Repos mit mehreren Graph-Stapeln, und die gab es vor G5a nicht.

**`graph = false` gilt für das Projekt.** Weil der Graph der Wurzel gehört, schaltet ein
ausdrückliches `graph = false` unter `[verify.<stapel>]` eines aktiven Stapels die Graph-Lane des
ganzen Projekts ab: jeder Graph-Job, auch der, der sonst trüge, bekommt `not-applicable` mit dem
Grund `graph switched off under [verify.<stapel>]`, benannt nach dem ersten solchen Stapel in
Byte-Reihenfolge, und die Graph-Prüfung (git-Aufrufe) läuft nicht. So behält ein bestehendes
Opt-out seine Bedeutung, wenn ein Repo zu seinem Go einen Python-Stapel bekommt; ohne diese Regel
trüge dann `graph/python` den Graphen weiter. Bleibt kein aktiver Stapel mit Graph-Befehl, gilt
die Trägerregel unverändert, und die Lanes melden `no command` wie vor G5a: ein reines Go-Repo mit
`graph = false` unter `[verify.go]` sieht denselben Bericht wie heute, ebenso ein gemischtes Repo,
das beide Stapel abschaltet. Ein Schalter unter einem Stapel, den das Projekt nicht hat, zählt
nicht. Unterschieden wird am Override der Konfiguration (`Lane.Off`), das nur `graph = false`
setzt; ein Stapel ohne Graph-Preset hat es nie. Preis: wer in einem gemischten Repo nur die
Python-Lane abschalten wollte, verliert auch die von Go — eine Graph-Lane je Stapel gibt es seit
G5a nicht mehr.

## 7. Python (G5a)

**Knoten.** Wie bei Go keine Knoten für Variablen und Konstanten.

- `file` je Modul.
- `class`; `function` auf Modulebene; `method` für ein `def` direkt im Rumpf einer Klasse,
  `Owner` = Klassenname. Geschachtelte `def` und `class` in Funktionen bekommen keinen Knoten,
  eine Klasse in einer Klasse ebenso wenig; sie gehören zum Rumpf um sie herum, samt ihren Aufrufen.
- **Modulebene** ist das Modul selbst und jeder Block eines `if`/`elif`/`else`, `try`/`except`/
  `else`/`finally` oder `with` darin, beliebig geschachtelt, nie innerhalb eines `def` oder einer
  `class`. Eine Definition dort bindet ihren Namen im Modul, und `try: from x import f` /
  `except ImportError: def f(): …` oder `if TYPE_CHECKING:` sind Alltag. Doppelte Namen bekommen
  Ordinale (`f~2`). In `for`, `while` und `match` gibt es keinen Knoten.
- **Wiederhergestellte Definitionen** bekommen keinen Knoten, und aus ihrem Teilbaum zählt kein
  Aufruf — wie bei einer Definition in einem Fehlerknoten. Wiederhergestellt ist eine Definition,
  deren Kopf (die Kinder vor dem ersten direkten `:`) in beliebiger Tiefe einen ERROR-Knoten
  enthält, die kein direktes `:` hat, oder deren `decorated_definition` vor der Definition einen
  ERROR trägt, auch tief in einem Decorator. Grund: ihr Span ist geraten und schluckt oft die
  nächste Definition (`def n(self)` ohne `:` wird mit dem folgenden `def o` ein Knoten `o` über die
  Zeilen von `n`). Preis: `def f(a, $):` fehlt samt Aufrufen, bis die Datei wieder parst. Ein
  MISSING-Token (`def broken(:`) und ein ERROR im Rumpf lassen den Knoten stehen.
- **Span** schließt Decorators ein. Aufrufe in Decorators, Default-Werten und Annotationen gehören
  darum der dekorierten Definition: eine Änderung dort trifft ihren Knoten, und `blast` zeigt dessen
  Aufrufer. Preis: ein Decorator-Aufruf, der beim Import läuft, zählt zur Funktion statt zum Modul.
- **Signatur** ist der Kopf vom Anfang der Definition bis zum `:`, das ihn schließt
  (`def f(a, b) -> T`, `class C(Base)`, `async def f(a)` — `async` bleibt). Das Ende ist dieses
  Token, nicht der Anfang des Rumpfs: ein Kommentar dahinter (`def f(a):  # noqa`) bleibt draußen,
  und tiefere Doppelpunkte in Annotation, Default oder Lambda gewinnen nie. Das Sprachpaket wählt
  das Ende, der Kern kennt keine Knotentypen.
- **Exported**: der Name beginnt nicht mit `_` — oder er ist ein Dunder
  (`__init__`, `__eq__`, beginnt und endet mit `__`): das ist Protokoll, nicht privat, und gehört in
  `graph_file_api`. `__all__` wird nicht gelesen.
- `.pyi`-Stubs gehören nicht zur Dateimenge; sie doppelten jede Definition. Endungen gelten genau
  und mit Groß- und Kleinschreibung, wie `.go` schon vorher: `x.GO` ist für die Go-Werkzeugkette
  keine Go-Datei, und der Umbau der Schnittstelle durfte keine Datei neu aufnehmen. Preis: `Foo.PY`
  fehlt auch auf einer Platte, die Groß- und Kleinschreibung nicht unterscheidet.

**Kanten.**

- `contains`: Datei → Klasse und Funktion, Klasse → Methode.
- `imports`: Datei → Datei im Repo, für `import a.b`, `from a.b import c` und relative Importe
  (`from . import x`, `from ..m import y`). Ziel ist `a/b.py` oder `a/b/__init__.py` unter einer
  Quellwurzel. Quellwurzeln sind die Repo-Wurzel; das Verzeichnis über jedem Paket oberster Ebene
  (ein Verzeichnis mit `__init__.py`, dessen Elternverzeichnis keines hat) — das ist `src/` für ein
  Paket, das aus `src/<paket>` installiert wird, und das Projektverzeichnis eines Pakets eine Ebene
  tiefer im Monorepo; und jedes Verzeichnis mit einer `manage.py`, denn Django startet sie mit
  ihrem eigenen Verzeichnis im Suchpfad, und seine Apps importieren einander von dort, mit oder ohne
  `__init__.py`. Die frühere Regel „jedes `src/`, das ein Paket enthält“ ist darin enthalten. Ein
  loses Modul oder ein Verzeichnis ohne `__init__.py` macht keine Wurzel. Innerhalb einer Wurzel
  schlägt das Paket das Modul.

  **Welche Wurzel gilt, hängt an der importierenden Datei.** Jede Wurzel hat ihre eigene Tabelle
  (Modulpfad → Datei). Ein absoluter Modulpfad wird zuerst über die Wurzeln aufgelöst, die die
  importierende Datei enthalten — eine Wurzel `r` enthält `p`, wenn `p` mit `r/` beginnt, und die
  Repo-Wurzel enthält jede Datei —, die tiefste zuerst; die erste, deren Tabelle den Pfad kennt,
  entscheidet. So findet `from tests.helpers import make` in `tools/cli/tests/test_cli.py` die
  Helfer unter `tools/cli/tests/` (Wurzel `tools/cli`, ohne `tools/cli/__init__.py`), dieselbe Zeile
  in `tests/test_core.py` die unter `tests/`; und eine Kopie `examples/demo/config/` nimmt dem
  `config/` an der Repo-Wurzel keine Kante mehr. Kennt keine dieser Wurzeln den Pfad, gilt die
  Tabelle über alle Wurzeln, mit der Mehrdeutigkeitsregel: ein Pfad, den zwei Wurzeln auf
  verschiedene Dateien abbilden, gibt dort keine Kante. Ein solcher Pfad löst also für jede Datei
  auf, die in einer der beiden Wurzeln liegt, nach der tieferen, die sie enthält; ist eine der
  beiden die Repo-Wurzel, ist das jede Datei (`main.py` neben `lib/` und `src/lib/` bekommt das
  `lib/` der Wurzel, so wie `python main.py` sein eigenes Verzeichnis vorn in den Suchpfad legt).
  Nur eine Datei außerhalb beider fällt auf die Mehrdeutigkeitsregel zurück: `import lib` in
  `tools/x.py`, neben `a/lib/` und `b/lib/` unter den Wurzeln `a` und `b` und ohne `lib` an der
  Repo-Wurzel — keine Kante statt einer geratenen. Die Tabelle über alle Wurzeln lässt eine Datei
  auch in das Paket einer Wurzel greifen, die sie nicht enthält: `main.py` an der Repo-Wurzel
  erreicht mit `from apps.x import helper` die Datei `backend/apps/x.py` neben
  `backend/manage.py`. Streng nach Python fände es sie nur mit `backend/` im Suchpfad; als Rückfall
  bleibt die Kante, weil sie die wahrscheinliche ist (so war es vor dieser Regel, und so bleibt
  es). Die Regel gilt für jede Suche nach einem Modul: Ziele von `imports` samt der Kante auf das
  Untermodul `X.name`, die Bindungen der From-Importe, Modul-Receiver (`import a.b`, `a.b.foo()`),
  Konstruktoren über ein Modul, Basisklassen (`class C(Base)` über einen From-Import, `class
  C(mod.Base)` über ein Modul) und die Re-Export-Kette, in der bei jedem Schritt die Datei die
  importierende ist, die den verfolgten From-Import enthält. Relative Importe lösen über den Pfad
  auf und brauchen keine Wurzel. Stdlib und installierte Pakete ergeben keine Kante. Gegen die
  Abnahme-Repos geprüft: `ultra-brain` legt sein Paket unter `src/brain` (`packages =
  ["src/brain"]` in `pyproject.toml`), `iam_backend` ist ein Django-Projekt mit Paketen direkt an
  der Wurzel (`apps/`, `core/`, `config/`). Beide deckt die Regel; ein Django-Projekt unter
  `backend/manage.py` deckt erst die `manage.py`-Wurzel.
- `extends`: Klasse → Basisklasse, wenn deren Name im selben Modul oder über einen Import samt
  Re-Exporten auf genau eine Klasse auflösbar ist; Confidence `extracted`. `class S(S)` erweitert nie sich
  selbst, denn Python liest die Basen, bevor es den Namen bindet: ist die definierende Klasse das
  einzige `S` ihres Moduls, bietet das Modul keines an, und das `S` aus `from base import S` ist
  die Basis. Zwei gleichnamige Klassen im Modul bleiben mehrdeutig und fallen auf den Import
  durch; welche wo gebunden ist, hinge an Reihenfolge und Zweigen, die der Index nicht kennt.
  Preis: `class T(T)` nach einem früheren `T` desselben Moduls bekommt keine Kante.
- `calls`, der Reihe nach:
  1. `foo()` — Definition im selben Modul, dann über `from m import foo` samt Re-Exporten (siehe
     unten; `extracted`); sonst ein im Repo eindeutiger Name derselben Sprache (`inferred`). Drei
     Arten von Namen werden nie geraten, weil die Datei sagt, dass sie nicht aus dem Repo kommen
     oder dass die Antwort keine einzelne Definition ist: ein Builtin (jeder öffentliche Name des
     Moduls `builtins`, `print`, `open`, `len`, `super` …), sonst sammelte eine Repo-Funktion
     `open` jedes `open()` des Repos; ein Name, den nur From-Importe aus Modulen **außerhalb** des
     Repos binden (`from json import load`), oder dessen Re-Export-Kette außerhalb des Repos
     endet, im Kreis läuft oder zu tief wird; und ein Name, dessen From-Import auf zwei
     Definitionen führt (`from m import foo as bar`, `m` definiert `foo` in `if` und `else`):
     schon welche der beiden gebunden ist, wäre geraten, und ein eindeutiges `bar` anderswo ist es
     sicher nicht. Das weicht vom wörtlichen „sonst“ ab. Preis: eine Repo-Funktion, die einen
     Builtin überschattet, findet nur der Weg über dasselbe Modul oder einen Import, und ein Name,
     dessen From-Import von außerhalb kommt, bekommt keine Kante, auch wenn eine gleichnamige
     Funktion im Repo liegt.

     **Außerhalb** ist ein Modul, das keine Datei hier trifft und dessen erstes Segment weder ein
     Verzeichnis noch eine `.py`-Datei irgendwo unter den Python-Dateien des Repos benennt
     (`json`, `django`). Trifft `from apps.users.models import helper` keine Datei, gibt es aber ein
     Verzeichnis `apps`, ist das Modul eines des Repos, das nur keine Quellwurzel erreicht — ein
     tiefer verschachteltes Projekt, ein Modulpfad, den zwei Wurzeln mehrdeutig gemacht haben und
     den keine Wurzel der importierenden Datei kennt, ein
     Geschwistermodul, das ein Skript über seinen bloßen Namen importiert —, und das Raten über den
     eindeutigen Namen bleibt. Ein relativer Import benennt einen Ort im Repo schon durch seine
     Form und ist nie außerhalb. Bindet eine Datei den Namen zweimal, einmal von außerhalb und
     einmal aus einem solchen Modul des Repos, bleibt das Raten ebenfalls.
  2. `self.foo()`, `cls.foo()` — Methode der umgebenden Klasse, dann ihrer Basisklassen im Repo
     (`extracted`), in die Breite, jede Klasse einmal, sodass ein Zyklus der Basen endet. Die
     erste Klasse, die den Namen definiert, entscheidet; definiert sie ihn zweimal, gibt es keine
     Kante. So hält es Python: die Klasse überschattet jede Basis, auch mit zwei Definitionen.
  3. `mod.foo()` mit importiertem `mod`, `Klasse.foo()` mit bekannter Klasse — die Funktion oder
     Methode dort, oder die, die das Modul weiterreicht (`extracted`).
  4. `obj.foo()` mit unbekanntem Empfänger — keine Kante. So hält es der Go-Resolver mit einem
     unbekannten Receiver; eine Namensraterei füllte `callers` und `blast` mit Rauschen.
  5. Ein Aufruf, dessen Ziel nach 1. oder 3. eine **Klasse** ist (`Klasse()`, `mod.Klasse()`), ist
     ein Konstruktor: die Kante geht an `Klasse.__init__`, wenn die Klasse selbst eines definiert,
     sonst an den Klassenknoten. Nur so zeigt `blast` bei einer Änderung an `__init__` dessen
     Aufrufer; `contains` ist keine Laufkante (`Relation.IsWalk`), eine Kante nur an die Klasse
     käme an `__init__` nie an.

**Re-Exporte.** Nennt ein From-Import (`from lib import Base`) oder ein Modul-Empfänger (`lib.X`,
`lib.work()`) eine Moduldatei hier, die den Namen nicht selbst definiert, folgt der Resolver deren
eigenem From-Import dieses Namens (`from .impl import Base, work` in `lib/__init__.py`), von Datei
zu Datei. Der relative Pfad gilt von der Datei aus, die den Import hält, und gesucht wird der
importierte Name, nicht sein Alias (`from .impl import work as job` reicht `job` als `work` aus
`lib/impl.py` weiter). Die Definition am Ende zählt wie ein direkter Import (`extracted`): die
weiterreichende Datei sagt, wo sie liegt. Das gilt für Basisklassen, für `foo()` über einen
From-Import, für `mod.foo()`, `mod.Klasse.foo()` und Konstruktoren, und damit für `self.foo()`
über eine so gefundene Basis. Auf jeder Stufe gilt die Regel des ersten Schritts: eine Bindung an
ein Modul, das keine Datei hier ist, wird übergangen, und die erste Bindung an eine Datei hier
entscheidet. Eine Kette endet ohne Kante,

- wenn sie außerhalb des Repos endet (`compat.py` reicht `load` aus `json` weiter): kein Raten;
- wenn sie zu einer Datei und einem Namen zurückkommt, die sie schon passiert hat, oder mehr als
  acht Schritte bräuchte (`maxReExports`): kein Raten; echte Paketketten sind ein, zwei Schritte
  lang;
- mehrdeutig, wenn die Datei am Ende den Namen zweimal definiert: keine Kante und kein Raten.

Endet sie in einer Datei hier, die den Namen weder definiert noch per From-Import bindet — ein
Wildcard-Import, eine Zuweisung, die der Index nicht sieht —, bleibt für `foo()` das Raten über den
eindeutigen Namen. Preis: eine lange Kette kann auf eine Definition zeigen, die Python zur Laufzeit
überschattet.

**Testpfade.** `blast.IsTestPath` bekommt eine Tabelle nach Endung und bleibt ohne Abhängigkeiten.
Für Python: `test_*.py` und `*_test.py` (was pytest von sich aus sammelt), `tests.py` (die
App-Vorlage von Django, und ein Name, den unittests `test*.py` findet), `conftest.py` (die Fixtures
von pytest) und jede Datei unter einem Verzeichnis `tests/` oder `test/` (die Helfer neben den
Tests). Ohne `tests.py` wäre ein gewöhnlicher Django-Commit, der eine Funktion und ihren Test in
`shop/tests.py` ändert, im Tor rot (`[none]`). `testing.py`, `contest.py`, `attests/` und
`latest/` bleiben draußen. `graph ask` stuft Testdateien nach derselben Regel herab, zusätzlich zu
den Namen, die die Referenz ohnehin kennt (`__tests__/`, `spec/`, `.test.ts`).

**Offen nach G5a.**

- Eine Python-Klasse als Saat behält im Blast das Testsignal `na`: als Verhalten zählen nur
  Funktionen und Methoden (`blast.signal`). Eine Änderung an einer Klasse ohne `__init__` oder an
  einem Klassenattribut läuft darum still durch `blast-audit`. Diese Spec hatte es nicht
  entschieden, und eine breitere Regel in `signal` berührt auch das Go-Tor.
- Die Suche in den Basen (Regel 2) geht in die Breite und ist nicht die C3-MRO. Beide weichen
  nur voneinander ab, wenn eine spätere Basis nicht von der Basis einer früheren abstammt: bei
  `D(B, C)`, `B(A)` mit `A.f` und `C.f`, `C` nicht von `A` abgeleitet, wählt die Breitensuche
  `C.f`, Python `A.f` (MRO `D B A C`). In einer echten Raute (`C(A)`) wählen beide `C.f`.
- Eine subskribierte Basis (`class C(Base[T])`, `Generic[T]`) gibt keine `extends`-Kante: der
  Extraktor nimmt als Basis nur Namen und Attributketten.
- Kein Überschatten durch Geltungsbereiche: ein Parameter oder eine lokale Variable, die wie ein
  importiertes Modul heißt, löst als dieses Modul auf (`import models` und
  `def f(models): models.save()` zeigen auf `save` im Modul `models`).
- Eine Unterklasse ohne eigenes `__init__` schickt Konstruktoraufrufe an den Klassenknoten, nicht
  an das geerbte `__init__` (Regel 5, so entschieden): `blast` auf eine Änderung an diesem
  `__init__` zeigt die Aufrufer der Unterklasse nicht.

## 8. Tore, Tests, Messung, Doku (G5a)

**Tore.**

- CGo-Freiheitstor in `ci/gate.sh`: `CGO_ENABLED=0 go build ./cmd/loomux` (angekündigt in §12.1
  der G2-Spec).
- Abhängigkeitstest: `internal/hooks` erreicht `gotreesitter` nicht (`go list -deps`).
- 100 % Coverage je Funktion für alle neuen Pakete; `gotreesitter` liegt außerhalb von `coverpkg`.
  Ein `//coverage:exempt` nur mit Grund, etwa für den Fall, dass eine eingebettete Grammatik nicht
  lädt.
- `gotreesitter` in der neuesten Version zum Zeitpunkt des Baus (heute v0.55.0), die übrigen Pins
  werden mitgeprüft.

**Regression.** Die Goldens unter `testdata/cases/graph/` tragen keine Extraktorkennung und bleiben
byte-gleich (geprüft am 2026-09-26: kein `"extractor"`, `stats.golden` zeigt `Languages: go`). Die
`.py`-Dateien, die loomux führt, liegen unter `docs/.superpowers/` (ein Dot-Verzeichnis) und unter
`testdata/`; beide überspringt `sourceset.SkipDir`. Der Graph dieses Repos bekommt also keinen
Python-Knoten. Darum gilt die starke Form, und zwar auf **demselben Quellbaum**: G5a ändert den
Quelltext von loomux selbst, der Graph des Arbeitsbaums muss sich also ändern. Die Grundlinie ist
der Graph, den das alte Binary über den Baum am Commit `b5c99cf1` baut (gesichert am 2026-09-26 im
Scratchpad der Sitzung; der Build ist deterministisch, zwei Läufe byte-gleich). Die Probe baut mit
dem neuen Binary über einen Worktree, der auf genau diesem Commit steht, und vergleicht:
`wiring.json` und `ask-index.json` byte-gleich bis auf `Meta.Extractor` — das einzige Feld, das sich
bewegen darf. Das belegt, dass die Schnittstelle den Go-Extraktor nicht verändert.

**Golden-Files.** Ein kleines, handgeschriebenes Korpus unter `testdata/cases/graph/python/` belegt
jede Regel aus §7: relativer Import, `from … import`, Methode über `self`, Aufruf in die
Basisklasse, `mod.foo()`, Konstruktor an `__init__` und an eine Klasse ohne `__init__`,
unbekannter Empfänger ohne Kante, geschachtelte Funktion ohne Knoten, Decorator im Span, Dunder als
exportiert, eine Datei mit Fehlerknoten, ein Re-Export über `pkg/__init__.py` (Basis, `self`-Aufruf
in sie, `pkg.start()`), ein Django-Projekt unter `backend/manage.py` ohne `__init__.py`, ein
Geschwistermodul, das ein Skript über seinen bloßen Namen importiert (geraten, nicht außerhalb),
und Testpfade (`tests/`, `test/`, Djangos `tests.py`; `TestPythonGolden` prüft sie gegen
`blast.IsTestPath`). Ein zweites Korpus
`testdata/cases/graph/mixed/` hat Go und Python mit absichtlich gleichen Namen (eine Funktion und
eine Methode `T.Run` in beiden Sprachen); die Go-Kanten müssen dieselben sein wie ohne die
Python-Dateien. Die Abnahme-Repos werden keine Goldens.

**Messung** in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`, auf ruhiger Maschine, vorher
gegen nachher, kalt und warm:

- `hook pre-tool-use`, `hook post-tool-use` auf einer `.go`-Datei, `--version`, Summe `inittrace`,
  Binary-Größe.
- `graph build` auf `ultra-brain` und `iam_backend`: kalt, warm mit Cache, nach einer geänderten
  Datei; Größe von `extract.json`.
- Die Probe aus §2 mit den Werkzeugversionen.

**Doku.** Zusammen mit dieser Spec, weil eine Entscheidung zuerst in die Fusions-Spec gehört
(`AGENTS.md`):

- Fusions-Spec: die Entscheidung `wazero` → `gotreesitter` (§10, E1) und die Teilung von G5 in
  G5a–d, in Stufen- und Prio-Tabelle.
- Säule-3-Spec: Stand-Zeile und §10 verweisen auf dieses Delta.

Im Pull Request von G5a:

- `docs/en/migration.md`, `docs/de/migration.md`: Zeilen G5a–d, Fähigkeitszeile „Multi-Language
  AST“ ohne `wazero` und AOT-Cache.
- `README.md`, `README.de.md`, die Wiki-Seite `docs/wiki/topics/code-graph.md`.

**Abnahme G5a.** Sie läuft nur an **Kopien** der Abnahme-Repos im Scratchpad
(`git clone --local <repo> <scratchpad>/<name>`), nie in den Repos selbst: `graph build` schreibt
`.loomux/state/graph/` (Graph, Cache, Frische-Datensatz) in die Wurzel, und die fremden Repos
liegen außerhalb der beschreibbaren Bäume. Dasselbe gilt für die Messung.

1. `graph build` läuft auf `ultra-brain` und `iam_backend` ohne Abbruch; der Bericht zeigt
   Python-Dateien und Parsefehler.
2. `graph ask`, `graph callers`, `graph blast` liefern an je drei von Hand gewählten Symbolen
   plausible Treffer; das Protokoll steht in einer Akte `docs/.superpowers/parity/code-g5.md`.
3. Ein zweiter Build ohne Änderung parst null Dateien neu, und ein warmer Build und ein Build mit
   `--no-reuse` schreiben byte-gleiche `wiring.json` und `ask-index.json`. Der Vergleich fängt jede
   Drift des Caches, auch einen verlorenen Body-Text.
4. Das neue Binary baut über den Baum am Commit `b5c99cf1` eine `wiring.json` und eine
   `ask-index.json`, die gegen die Grundlinie byte-gleich sind bis auf `Meta.Extractor` (siehe
   Regression).

## 9. Leitplanken für G5b bis G5d

- **G5b TypeScript/TSX.** `.ts`, `.tsx`; `.d.ts` nicht (wie `.pyi`). ES-Importe relativ und über
  `paths`/`baseUrl` aus `tsconfig.json`; Klassen, Methoden, Funktionen, Arrow-Funktionen in
  `const`; `this.foo()` wie `self.foo()`. Testpfade `*.test.ts(x)`, `*.spec.ts(x)`, `__tests__/`.
- **G5c GDScript.** Eine Datei ist eine Klasse; `class_name` macht sie global bekannt; `extends`
  mit Name oder Pfad; `preload("res://…")` und `load("res://…")` als Import, `res://` ist die
  Wurzel mit `project.godot`. Innere Klassen (`class X:`) als `class`. Signale zunächst ohne Kante.
  Die drei Dateien mit Fehlerknoten aus §2 werden vorher angesehen.
- **G5d C++.** Eintrittsbedingung vor jedem Extraktorcode: auf den 140 Dateien, die nur
  `gotreesitter` als fehlerhaft meldet, werden Definitionen und Aufrufe gegen die C-Referenz
  gezählt. Erst danach fällt die Entscheidung, ob `gotreesitter` für C++ reicht. `#include`,
  Namensauflösung über Namespaces nur heuristisch, alles `inferred`.

## 10. Getroffene Entscheidungen

| # | Entscheidung | Grund |
|---|---|---|
| E1 | `gotreesitter` als direkte Abhängigkeit statt `wazero` + WASM, und keine gestutzte Kopie unter `third_party/` | Die Probe (§2) zeigt, dass es ohne WASM-Build und C-Werkzeugkette geht. Eine Kopie mit Init erst bei Bedarf spart ~2 ms Init, kostet aber bei jedem Update einer schnell laufenden Bibliothek den Patch neu — und die C++-Lücke braucht gerade die Upstream-Fixe. Die ~2 ms Init sind deterministisch und damit angenommen; die Nachmessung in G5a dokumentiert sie. Steigt der warme Median von `hook pre-tool-use` auf ruhiger Maschine um 3 ms oder mehr, ist die Kopie der Ausweg und wird dem Nutzer vorgelegt |
| E2 | Sprachen Python → TypeScript/TSX → GDScript → C++; Go bleibt nativ | Wunsch des Nutzers, von einfach nach schwer; `go/parser` ist schneller als jede Tree-sitter-Laufzeit und steckt schon im Hook-Pfad |
| E3 | Ansatz 3: Schnittstelle je Sprache mit gemeinsamem Tree-sitter-Kern | Präzision wie bei Go (Owner, Receiver) bei wenig Wiederholung. Verworfen: ein generischer Tags-Extraktor, der Aufrufe nur über den Namen auflöst und damit `callers` und `blast` schwächt |
| E4 | G5 ist nur Extraktion | `graph viz` steht unter W3 |
| E5 | Abnahme an `iam_backend`, `ultra-brain` (Python), `iam_frontend` (TS/TSX), `space` (GDScript), `nano-coverage-godot` (C++) | Wahl des Nutzers; loomux selbst hat keine dieser Sprachen |
| E6 | Der Post-Edit-Monitor bleibt Go-only | Ladezeit der Grammatik und Parsezeit einzelner Dateien (§2) passen in kein Hook-Budget |
| E7 | Ein absoluter Modulpfad löst zuerst über die Quellwurzeln auf, die die importierende Datei enthalten, die tiefste zuerst; erst danach über die Tabelle aller Wurzeln mit der Mehrdeutigkeitsregel (§7, `imports`) | Wahl des Nutzers (Option A, vor dem Merge von G5a). Mit einer einzigen Tabelle für das ganze Repo löschte ein gleichnamiges Paket unter einer zweiten Wurzel — eine Kopie unter `examples/`, ein zweites `tests/` — die richtigen Kanten beider für jede Datei; in zwei nachgebauten Fällen fiel so jede Kante weg |
