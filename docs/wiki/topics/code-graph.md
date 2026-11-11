---
type: Topic
title: Der Code-Graph
description: Wie loomux Symbole und Kanten ohne Modell aus dem Quelltext zieht, Treffer mit Personalized PageRank ordnet und den Blast-Radius einer Änderung berechnet.
open_conflicts: 0
realization: in_progress
sources:
  - id: architektur
    resource: brain://project/loomux/docs/de/architecture.md
    doc_id: 01M47W91R5WKHDNYTXW8ZXTCSW
    content_hash: "sha256:73987ba3b351a544cd39c56a30ec02d5f214b0bb161a5bca39f5e552228eea88"
    revision: 3
  - id: cli-referenz
    resource: brain://project/loomux/docs/de/cli-reference.md
    doc_id: 01M47W91R5FRSPN6BE643656ZE
    content_hash: "sha256:0d41d2c963a77fa739c81c1d7b4231a1c5e65865ed94a0b738cbd4a3047138a8"
    revision: 34
---

## Wofür

Ein Coding-Agent erkundet ein Repository sonst in jeder Sitzung blind von
vorn. Der Code-Graph gibt ihm stattdessen einen **deterministischen Symbol-
und Aufrufgraphen, ohne Modell und ohne Kosten**: Symbole, Aufrufe,
Typ-Hierarchien und Importe. Leitsatz: **„Lexik schlägt vor, der Graph
entscheidet."** Eine Vektordatenbank braucht es dafür nicht.

Das Vorbild steht in `NOTICE.md` unter „Ported sources“. Übernommen sind Extraktion, Ranking und die
Frischeprüfung; statt des Kernlogik-Exzerpts (Crux) des Vorbilds hängt `graph ask
--source` den eigenen Span des Symbols an, höchstens 80 Zeilen. Verworfen sind Laufzeit, Cloud und
Telemetrie:

| Vorbild | loomux |
|---|---|
| Node.js mit nativen C++-Bindings | ein Go-Binary, CGo-frei |
| Symbol-Hashes an eine Cloud-API | Verknüpfung nur lokal, mit dem eigenen Wiki |
| Nutzungsstatistik an fremde Server | keine Telemetrie |

## Wie der Graph entsteht

`loomux graph build` zieht Symbole und Kanten aus dem Quelltext und schreibt
sie nach `.loomux/state/graph/wiring.json` — Maschinenzustand, git-ignoriert.

- **Go** liest er allein mit `go/parser`. Die Spec sah für den Offline-Bau
  noch `go/types` vor; gebaut ist der Extraktor ohne `go/types` und ohne
  Build (`internal/code/extract/golang`). Gelesen werden die gepflegten
  Go-Versionen, heute 1.26 und 1.27 mit `new(expr)`, selbstbezüglichen
  Typparametern, generischen Methoden und beförderten Feldern als Schlüssel
  in Struct-Literalen.
  Geprüft wird das je Version über die Kompatibilitätsliste, ein
  Pre-Release vorab als Probe des Compat-Scouts ohne Gate.
- **Python** liest er seit G5a auf `gotreesitter`, einer Tree-sitter-Laufzeit
  in reinem Go (`internal/code/extract/python` auf dem gemeinsamen Kern
  `extract/treesitter`). Das Binary bleibt CGo-frei; ein Build mit
  `CGO_ENABLED=0` im Tor hält das fest. Eine Datei mit Syntaxfehlern bricht
  den Build nicht ab: sie behält ihren Dateiknoten, und der Bericht zählt die
  Fehler.
- **Godot-Projekte** liest er seit G5c auf demselben Kern
  (`internal/code/extract/gdscript`): GDScript, Szenen (`.tscn`), Ressourcen
  (`.tres`) und `project.godot`, als eine Sprache, damit eine Szene ihr Skript
  erreicht. `project.godot` legt die `res://`-Wurzel und die Autoloads fest;
  ohne es gibt es keine `res://`-Kante. Ein Pfad ohne Schema gilt bei
  `preload` ab dem Ordner des Skripts, bei `load` ab `res://` (so liest
  Godot es), bei einem Autoload ab dem Ordner der `project.godot`. Ein Skript mit `class_name` ist ein
  Klassenknoten über die ganze Datei. Godot 3 und Godot 4 liest er nach der
  Form, nicht nach der Version: `.f()`, `setget`, `yield` und
  `connect("s", self, "m")` ergeben dieselben Kanten wie `super.f()`,
  `get =`/`set =`, `await` und `s.connect(m)`. Methoden und Signale als Wert
  oder per String-Literal sind Referenzen (`call*` und `rpc*` Aufrufe), eine
  `[connection]` einer Szene referenziert Handler und Signal, ein
  Szenen-Autoload bindet das Wurzelskript, und `uid://` löst sich über die
  `.uid`-Dateien und die UIDs im Kopf von Szenen und Ressourcen auf
  (`docs/.superpowers/specs/2026-10-19-gdscript-godot-linien-design.md`).

Jede Sprache löst in einem eigenen Namensindex auf. Über eine Sprachgrenze geht
ein Godot-Pfad, der eine Datei nennt, die ein anderer Extraktor liest (etwa
eine `.py`- oder `.go`-Datei): er landet auf deren Dateiknoten. Eine Datei,
die kein Extraktor liest (eine Textur, heute auch ein `.cs`-Skript), ergibt
keine Kante.

Symbol zu Symbol über eine Sprachgrenze geht nur über eine **Brücke**
(`internal/code/resolve/bridge.go`): Eine Seite bietet einen Namen in einem
Namensraum an (Export), die andere nennt ihn (Brücke), und nachdem jede
Sprache aufgelöst ist, verbindet ein sprachneutraler Schritt beide nach den
Regeln des Namensraums: genau ein Treffer oder alle, ein Geltungsbereich
(für `godot.*` das Godot-Projekt; ein Export außerhalb jedes Projekts, etwa
GDExtension-Quellen neben `project/`, gilt in allen), und ob ohne Treffer im
genannten Bereich gesucht oder geraten wird. Den Namen eines Exports legt die
anbietende Seite fest. GDScript bietet jede `class_name` eines Projekts als
`godot.class` an und übergibt, was es selbst nicht binden kann: ein
`extends X` und `X.new()` mit flachem Namen und das Ende einer Szenen-
`[connection]`, deren Skript keine GDScript-Datei ist. Eigene Exports
beantworten eigene Übergaben nie, also ändert sich GDScript zu GDScript
nicht. Heute liefert noch kein Extraktor C# oder C++; die Brücken nach Godot
sind gebaut und getestet, greifen aber erst mit diesen Extraktoren
(`docs/.superpowers/specs/2026-11-10-graph-bruecken-design.md`). Ein
Extraktor kann Teile einer Datei in einer anderen Sprache als eingebettete
Ergebnisse melden (`Result.Embedded`, etwa Inline-Skripte); sie landen im
Index ihrer eigenen Sprache.

Unveränderte Dateien nimmt der Build aus dem
Extraktions-Cache (`.loomux/state/graph/cache/extract.json`), statt sie neu zu
parsen; gelesen und gehasht wird trotzdem jede, und `--no-reuse` parst alle.
`loomux graph check` extrahiert erneut, vergleicht mit dem Graphen auf der
Platte und endet mit 0, wenn er frisch ist, mit 1 bei Drift.

**Die Pakete liegen unter `internal/code/`, nicht unter dem `internal/graph/`
der Spec.** Die Trennung, die die Spec verlangt, gilt trotzdem: der
Hook-Pfad bindet kein MCP-SDK, und ein Tor-Test liest dafür den Importgraphen.

## Wie er ordnet

**Personalized PageRank („GraphRank"):** Lexikalische Treffer (BM25 auf
Symbolnamen und Doku) sind die Saat, dann eine Power-Iteration über den
ungerichteten Graphen mit Neustart-Wahrscheinlichkeit 0,25 und 25 Schritten.
Die Masse isolierter Knoten fließt an die Saat zurück, die Summe bleibt 1; bei
Gleichstand entscheidet die Symbol-ID alphabetisch. `loomux graph ask` fährt
diese Kette.

**Blast-Radius:** Gerichtete Kanten (`calls`, `imports`, `extends`,
`implements`, `references`), in Richtung `in` (wer hängt von mir ab?) oder
`out` (wovon hänge ich ab?), transitiv per Breitensuche mit Tiefengrenze.
`loomux graph blast` führt einen git-Diff auf die Symbole, die seine Hunks
berühren, und auf das, was sie erreicht.

**Geratene Kanten:** Jede Kante trägt eine Konfidenz. `inferred` heißt: Der
Resolver hat einen Aufruf nur über einen Namen zugeordnet, der im ganzen Repo
genau einmal vorkommt. In Go trifft das auch jeden Aufruf in eine andere Datei
desselben Pakets; in diesem Repo ist gut jede vierte `calls`-Kante so geraten.
`graph callers`, `graph blast`, ihre MCP-Werkzeuge und der Post-Edit-Hook
hängen solchen Treffern `inferred` an. Ein Treffer gilt als geraten, sobald
eine Kante auf seinem Weg geraten ist; erreicht ihn ein sicherer Weg gleicher
Länge, gilt er als sicher und zeigt dessen Relation. Längere Wege zählen
nicht. Das JSON trägt dafür `"inferred": true`. Zählungen und Signale
(In-Degree, Test-Signal von `blast`, Ranking) werten geratene Kanten weiter
voll.

**Funktionswerte und Ketten (Go):** Steht eine Funktion als Wert statt als
Aufruf da (`Deps{Blast: query.Blast}`, `var postToolUse = hooks.PostToolUse`,
`register(s.handle)`), entsteht eine Kante `references` vom umschließenden
Symbol, auf Paketebene von der Datei. Wertpositionen sind Argumente,
Elemente und Werte von Composite-Literalen, rechte Seiten und `return`. Ein
bloßer Name wird nur in seiner Datei und seinem Paket gesucht, nie
repo-weit. Ein Aufruf auf einem Aufrufergebnis (`blast.New(g).Reach()`, auch
`x := blast.New(g); x.Reach()`) löst über den ersten Ergebnistyp der inneren
Funktion auf, gelesen aus ihrer Signatur, und nur im Paket dieser Funktion;
er ist so sicher wie der innere Aufruf. Vorher endeten Ketten an diesen
Stellen, und grep fand mehr als der Graph. Messung: `docs/de/benchmarks.md`,
Eintrag vom 2026-11-11 „Funktionswerte und Ketten im Go-Graph“. Ein
Aufruf über einen lokalen Namen (`mk()`, `m["k"]()`) nennt keine Funktion
des Pakets und ergibt keine Kante. Ergebnistypen aus anderen Paketen und
Interface-Dispatch bleiben offen. `graph callers` zitiert die Zeile, auf der
die Kante steht, nicht die erste Erwähnung des Namens.

## Die Oberflächen

| Befehl | MCP-Werkzeug | Zweck |
|---|---|---|
| `graph ask` | `graph_find_code` | Suche über Symbole, nach GraphRank geordnet |
| `graph check` | `graph_check_freshness` | Drift zwischen Arbeitsbaum und Graph |
| `graph skeleton` | `graph_file_api` | Signaturen einer Datei ohne Rümpfe |
| `graph callers` | `graph_trace_calls` | Aufrufer oder Aufgerufene, bis zur Hülle |
| `graph grep` | `graph_find_all` | Regex über den Suchkorpus, gruppiert nach Symbol, nach Kopplung geordnet |
| `graph map` | `graph_repo_map` | Repo-Karte mit Hubs und Hotspots |
| `graph blast` | `graph_blast` | Blast-Radius eines git-Diffs |

`graph grep` sucht nicht nur die Dateien des Graphen, sondern jede Textdatei,
die git nicht ignoriert: `graph build` legt dafür neben dem Graphen einen
Suchkorpus-Record und einen Trigramm-Index an, und `graph grep` antwortet aus
dem Index, wenn das Muster ein Literal enthält, sonst aus einem Scan derselben
Liste. `--channel cloud` liest dabei die `never`-Deklaration des Projekts.

**Warum ein Aufruf früher langsam war, und was ihn schnell macht.** Die erste
Messung gegen tgrep schob die Langsamkeit auf das Laden von Wiring-Datei und
Index; das Profil zeigte etwas anderes: zwei serielle Walks je Aufruf auf
einem sauberen Baum und, nach einer Änderung, einen Neubau von rund 4 s.
Daraus folgt die heutige Form, deren Einzelheiten in
`docs/de/cli-reference.md` stehen:

- **Ein paralleler Walk** bedient Korpus und Graph, und der Index wird nur
  abgebildet und lazy gelesen; die Symbole kommen aus einer eigenen Beiakte
  (`grepspans.idx`), nicht aus `wiring.json`.
- **Das Overlay:** Betrifft die Abweichung nur Dateiinhalt und höchstens 200
  Dateien je Korpus und Graph, baut `grep` nichts neu. Der alte Index liefert
  die Kandidaten der unveränderten Dateien, geänderte und neue werden gelesen
  und tragen kein Symbol (lieber keins als ein falsches). `index` meldet dann
  `overlay`. Ein bewegter `.git/index` zählt nicht als Abweichung, wenn git
  dieselben Dateien verfolgt; der erste Aufruf nach einem Commit liest dafür
  die Pfadliste selbst aus `.git/index` (Version 2–4, sonst fragt er
  `git ls-files`) und schreibt den neuen Stempel in den Record, auch unter
  `--no-refresh`.
- **Record und Index als Paar:** Der Build schreibt den Index vor dem Record;
  scheitert der Index, bleibt der alte Record stehen. Trägt der Index trotzdem
  einen anderen Digest als der Record (oder fehlt er), baut der nächste Aufruf
  und der Wächter neu, statt bis zur nächsten Dateiänderung `stale` zu scannen.
- **`--trust-index`** lässt den Walk ganz weg: Dateiliste und Zähler stammen
  aus dem Index, die Symbole aus der Beiakte, `index` meldet `trusted`, und
  `index_built` nennt das Alter des Index. Auf einem geänderten Baum ist die
  Antwort bewusst die des Index. Der Schalter ist keine Voreinstellung,
  weil er stumm falsch antworten kann; das Plugin setzt ihn nur über die
  Option `trustIndex`. Unter `--in` und `--glob` zählt er alle Binärdateien in
  `files_in_scope` mit, weil die Dateitabelle sie keinem Präfix zuordnen kann.
- **Der Wächter in `loomux serve`:** Der Dienst beobachtet die Wurzeln der
  registrierten Bereiche mit gebautem Graphen, prüft 2 s nach der letzten
  Änderung, spätestens 10 s nach der ersten einer Folge (dieselbe
  Driftprüfung wie der Einmalaufruf), und baut nur bei Abweichung neu. Was die
  Ignore-Regeln des Records ausschließen, löst nichts aus. `graph build` nimmt dieselbe Neubau-Sperre und wartet bis
  zu 30 s auf einen laufenden Neubau, sodass sich zwei Builds nie
  verschränken. Ein Graph eines anderen Extraktors fasst er nicht an. Das
  Overlay bleibt die Grundlage: Ohne Dienst baut der Aufruf ab 200 Dateien
  selbst neu. `graph_find_all` ruft dieselbe Suche und profitiert von beidem.

Die Zahlen (Windows, Baum mit etwa 8500 Dateien) stehen in
`docs/de/benchmarks.md` unter „Abnahme des Starts von graph grep“. Offen aus dieser Abnahme: der erste Aufruf nach einem Commit
liegt über dem Ziel von 108 ms, und ob tgrep abgelöst wird, entscheidet der
Mensch.

Dazu `graph build` und `graph stats`. Die Spec nannte sechs MCP-Werkzeuge;
`graph_blast` kam als siebtes hinzu. `loomux serve` beantwortet sie neben den
fünf `brain_*`-Werkzeugen. **Auf dem Cloud-Kanal schließen sie im Zweifel:**
was ein `local_only`-Bereich verbirgt, verraten auch die Graph-Werkzeuge
nicht ([Datenschutz und Kanäle](datenschutz-und-kanaele.md)).

## In Hook und Prüfkette

- **Post-Edit-Hook, rein informativ:** Nach einem Edit an einer `.go`-Datei
  ohne rote Lane nennt er die direkten Aufrufer in anderen Dateien dessen, was
  sich geändert hat (höchstens zehn). Ohne Graph oder bei einem Parsefehler
  schweigt er; **einen Edit blockiert er nie.** Eine `.py`-Datei bekommt
  keinen: jeder solche Edit zahlte das Laden einer Grammatik und das Parsen
  der Datei, gegen ein Ziel von unter 100 ms Eigenzeit des Monitors.
  `internal/hooks` importiert die Tree-sitter-Laufzeit darum nicht.
- **Prüfart `graph`:** `check graph-fresh` baut bei Drift neu und ist nur rot,
  wenn der Neubau scheitert; `check blast-audit` ist rot, wenn ein geänderter
  Bereich mit genug Aufrufern keinen geänderten Test hat. Die Lane-Namen der
  Spec (`graph-freshness`) wurden dabei ersetzt. Presets haben Go, Python,
  GDScript und C#; die Lane läuft je Lauf einmal, in der Wurzel, getragen vom
  ranghöchsten aktiven Stack mit Graph-Befehl (go, python, gdscript, csharp; ein Stack
  ohne Rang folgt in Byte-Reihenfolge), und `graph = false` unter einem Stack schaltet sie
  für das ganze Projekt ab. Testdateien sind `_test.go` und für Python
  `test_*.py`, `*_test.py`, `tests.py`, `conftest.py` und alles unter
  `tests/` oder `test/`, für GDScript `*_test.gd` und alles unter `test/`
  oder `tests/`; Szenen und Ressourcen (`.tscn`, `.tres`) zählen nur dort,
  `project.godot` nie.
- **Prüfart `rules`:** Die sechste Art liest denselben Extraktor-Kern, aber
  nie den gespeicherten Graphen: `check rules` baut ihn im Prozess aus den
  Dateien, die git kennt (mit `--cached` aus dem Index), und prüft jede
  aufgelöste Kante außer `contains` gegen die Ebenenkarte `[rules.layers]` des
  Projekts. Sie braucht darum keinen frischen Index und läuft auch in CI.
  Dazu kommen fehlende Testmodule, Quellen ohne Ebene und, nur beim Commit,
  liegen gebliebene Coverage-Instrumentierung. Wie `graph` gehört sie zur
  Wurzel (ein Job je Lauf, getragen vom ranghöchsten Stack), wird aber ohne
  Tabelle `[rules]` gar nicht geplant. Eine Ausnahme im Code gibt es nicht;
  eine Freigabe geht über `config set rules.layers.<ebene>.may_use … --propose`
  und einen Menschen. Die Zuordnung einer Datei zu einer Ebene entscheidet der
  spezifischste Glob (an der Wurzel verankert vor jeder Tiefe, ohne Platzhalter
  vor mit, mehr feste Segmente vor weniger).
- **Stop-Hook mit Blast-Logik:** `graph` steht auch im Profil `stop`. Am
  Rundenende prüft `blast-audit --cached` über eine Kopie des Index, die den
  ganzen Arbeitsbaum samt unversionierter Dateien trägt, alles gegen `HEAD`;
  ein Befund hält die Runde mit Exit 2 an wie ein roter Test.

## Was bewusst fehlt

Kein Gegenstück zum Cloud-Brain des Vorbilds, keine Telemetrie, kein `upgrade`-Befehl: das
Wissen ist lokal ([Architektur-Grundsätze](architektur-grundsaetze.md)), und
das Binary kommt über die üblichen Wege.

## Was offen ist

- **Mehrsprachige Extraktion** (G5): G5a — die Schnittstelle, der Kern auf
  `gotreesitter`, Python und der Cache — ist fertig und an zwei Python-Repos
  abgenommen (`docs/.superpowers/parity/code-g5.md` in den Arbeitspapieren des
  Archiv-Release `archive/parity-recordings`); G5c (GDScript, Szenen,
  Ressourcen) ist gebaut. Offen sind G5b (TypeScript/TSX) und
  G5d (C++, erst nach einer Recall-Prüfung gegen die C-Laufzeit). Der erste
  Plan, Tree-sitter als WebAssembly über `wazero`, ist verworfen.
- **C# und C++ für Godot:** Die Brücken stehen; es fehlen die Extraktoren,
  die `godot.class`, `godot.method` und `godot.signal` anbieten (C#:
  `[GlobalClass]`, Signal-Delegates, Methoden; C++: `GDREGISTER_CLASS`,
  `ClassDB::bind_method`). Aufrufe auf eine Basisklasse, die erst die Brücke
  findet, und eingebaute Skripte einer Szene (`[sub_resource type="GDScript"]`)
  bleiben ohne Kante.
- **Eine Python-Klasse als Saat** gibt im Blast das Testsignal `na`: als
  Verhalten zählen nur Funktionen und Methoden, eine Änderung an einer Klasse
  ohne `__init__` läuft darum still durch `blast-audit`.
- **Die Konfiguration `[graph]`** mit `languages`, `exclude` und `max_depth`
  aus der Spec: das Schema kennt heute nur den Schalter `graph` unter
  `[modules]`.
- Die Brücke von Code-Symbolen zu Wiki-Seiten und ADRs und der Viewer
  `graph viz` ordnete die Spec G4 und G5 zu; die Roadmap im README führt
  beide heute unter dem Graph-Visualizer.

Quelle: die Architektur und die CLI-Referenz der Nutzerdoku; offene Stufen
führt die Roadmap im README.
