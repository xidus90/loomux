# C++ im Code-Graphen, in der Prüfkette und an Godot (GDExtension)

**Stand:** 2026-11-10 · Teilprojekt 2 von
`2026-11-10-graph-sprachen-programm.md` · baut auf den Brücken und auf C#
(`Result.Enclosing`, `RawEdge.Member = "new"`)

## Ziel

C++ steht in Graph, Prüfkette und Kompatibilitätsliste auf der Stufe von
Python. Eine GDExtension in C++ (godot-cpp) wird von GDScript (`extends`,
`.new()`), von Szenen (Knotentyp) und von Szenen-Verbindungen erreicht.

## Nicht-Ziele

- **Kein Präprozessor.** `#if`-Zweige werden alle gelesen, Makros nicht
  expandiert, `#include` nur als Kante.
- **Keine Typinferenz und keine Überladungsauflösung.** Bei gleichem Namen
  gilt die erste Definition.
- **Keine `.c`-Dateien.** C ist eine andere Sprache. `.h` wird als C++
  gelesen: Die meisten C++-Projekte nutzen es, auch die Beispiele von
  godot-cpp.
- **Keine Kante für Deklarationen ohne Rumpf.** Ein Knoten entsteht nur
  für eine Definition.

## Extraktor `internal/code/extract/cpp`

- **Grunddaten:**
  - Sprache `cpp`, Version `cpp/1@<Parser>`, Grammatik
    `gotreesitter/grammars/cpp`
  - Endungen `.cpp`, `.cc`, `.cxx`, `.c++`, `.hpp`, `.hh`, `.hxx`, `.h++`,
    `.h`, `.inl`, `.ipp`, `.tpp`
- **Vorverarbeitung:**
  - Jede Zeile, die nur aus einem Bezeichner in Großbuchstaben besteht,
    wahlweise mit einer Argumentliste ohne `;`, `{` oder `}`, wird vor dem
    Parsen durch gleich viele Leerzeichen ersetzt. Beispiele: `GDCLASS(X, Y)`,
    `Q_OBJECT`, `NLOHMANN_JSON_NAMESPACE_BEGIN`.
  - Ausnahme: Ist das nächste Zeichen nach der Zeile, das kein Leerraum ist,
    ein `{`, bleibt die Zeile stehen. So bleibt `TEST_F(Suite, Name)` mit der
    Klammer auf der nächsten Zeile erhalten (Allman-Stil). Tests decken beide
    Klammerstile ab.
  - Die Offsets bleiben dadurch gleich. Spans, Signaturen und Rumpftexte
    stammen aus dem Original.
  - `GDCLASS(X, Base)` wird vorher als Godot-Angebot erkannt (siehe unten).
- **Abgeschnittener Baum (gotreesitter v0.57.0, gefunden am 2026-11-11):**
  - `using X = A<B<T>>::type;` in einer Template-Klasse beendet den Baum
    an dieser Stelle; der Rest der Datei fehlt ohne Meldung.
    `ParseStoppedEarly` bleibt falsch.
  - Die C-Laufzeit (tree-sitter 0.26.1) parst dieselbe Eingabe fehlerfrei.
    Der Fehler liegt also in gotreesitter. Abdeckung im Probelauf: fmt 46 %,
    nlohmann 27 %.
  - **Umgehung:**
    - Die rechte Seite einer solchen Alias-Deklaration wird zu `int` plus
      Leerzeichen gleicher Länge maskiert; Zeilenumbrüche bleiben stehen.
      Das Ziel eines Alias liest der Graph ohnehin nicht.
    - Endet der Baum trotzdem vor dem Quelltext, wird der gelesene Anfang
      ausgeblendet und der Rest neu geparst. Nach dem Schnitt fehlen
      Namespace und Owner.
  - Ein Upstream-Issue mit dem kleinsten Beispiel wird dem Nutzer
    vorgeschlagen.
  - `if consteval` verliert ebenfalls den ganzen Rest der Datei, nicht nur
    die nächste Definition; das Neu-Parsen deckt es ab.
- **Fehlertoleranz:** Eine Definition zählt, wenn ihr eigener Teilbaum keinen
  ERROR enthält, auch unter einem ERROR-Vorfahren. Gemessen in der
  Programm-Spec: 99 % gegen 16–43 % mit der Regel von Python.
- **Knoten:**
  - Datei.
  - Typen mit Rumpf: `class`, `struct`, `union`, `enum`; Art `class`.
    Verschachtelte Typen heißen `Outer.Inner`.
  - Funktionen mit Rumpf: freie Funktionen (Art `function`) und Methoden in
    der Klasse (Art `method`, Owner = Klasse).
  - Methoden außerhalb der Klasse (`void ns::A::B::m() {}`): Art `method`,
    Name `m`, Owner = das letzte Segment des Qualifiers (`B`). Ob das eine
    Klasse oder ein Namespace ist, kann der Extraktor nicht sehen: `void
    util::z() {}` bekommt den Owner `util`, und erst der Resolver macht daraus
    eine freie Funktion im Namespace `util` (`rules.symbolOf` liest Owner
    trotzdem als Klasse). Der ganze Qualifier `ns::A::B` steht in
    `Result.Enclosing` unter der ID der Methode; ob `ns` ein Namespace oder
    eine Klasse ist, klärt der Resolver.
  - Der Owner einer Methode in der Klasse kommt nur aus echten
    `class_specifier`/`struct_specifier`-Vorfahren. Ein zerbrochener
    Klassenkopf unter einem ERROR gibt der Methode keinen erfundenen Owner;
    sie wird dann eine freie Funktion. Ein Test hält das fest. Bricht nur
    der Rumpf, bekommt die Klasse keinen Knoten; ihre heilen Methoden
    behalten sie als Owner und gehören im Resolver zu keinem Typ.
  - Template-Definitionen wie ihr Inneres.
  - Operatoren heißen `operator+` usw., Konstruktoren und Destruktoren wie
    in C++ (`X`, `~X`).
- **IDs:**
  - wie bei C#, nur `.` als Trenner: `rel#X`, `rel#Outer.Inner`, `rel#X.m`,
    auch bei Methoden außerhalb der Klasse (`rel#B.m`)
  - gleicher Name in einer Datei (Überladung): Ordinalzahl über
    `extract.MintID`
- **`Exported`:**
  - Mitglieder: `public`-Abschnitt; in `struct` ist `public` die Vorgabe.
  - Freie Funktionen: nicht `static` und nicht in einem anonymen Namespace.
- **Namespaces:** `namespace a::b {}` und verschachtelte Namespaces kommen
  nach `Result.Enclosing`, für den Resolver, nie in den Owner:
  - Typen und freie Funktionen: der Namespace (`a::b`)
  - Methoden außerhalb der Klasse: der umgebende Namespace plus der ganze
    Qualifier (siehe oben)
  - Ein anonymer Namespace heißt `""`.
- **Imports:** `#include "x.h"` und `#include <a/b.hpp>` werden zu
  `extract.Import{Path: "x.h"}` plus einer `imports`-Rohkante. Der Resolver
  sucht die Datei.
- **Rohkanten:**
  - `extends` je Eintrag von `base_class_clause`, ohne Template-Argumente.
  - `calls`:
    - `f()`
    - `a::b::f()`: Receiver `a::b`
    - `obj.f()` und `p->f()`: Receiver `obj`, wird nicht aufgelöst, wie
      Python
    - `this->f()`: Receiver `this`
    - `new T()`: `Member = "new"`. Ein bloßes `T()` sieht aus wie `f()`; der
      Resolver fällt dort nach den Funktionen auf den Typ und dessen
      Konstruktor zurück (wie `construct` bei Python).
  - **Godot-Bindungen** im Rumpf von `_bind_methods` (Kind `references`):
    - `ClassDB::bind_method(D_METHOD("name", …), &X::m)`,
      `bind_static_method`, `bind_vararg_method`: Rohkante mit
      `Member = "godot.method"`, `Specifier = "name"`, Receiver `X`,
      Name `m`.
    - `ADD_SIGNAL(MethodInfo("name", …))`: Rohkante mit
      `Member = "godot.signal"`, `Specifier = "name"` und Receiver = der
      Klasse von `_bind_methods` (Owner).
    - `extract.go` dokumentiert diese dritte Bedeutung von `Member`.
- **Godot-Angebot im Extraktor:** Eine Klasse mit Rumpf, deren Text
  `GDCLASS(<Name>, …)` enthält, bietet `godot.class <Name>` an; Ziel ist der
  Klassenknoten. Godot verlangt `GDCLASS` für jede registrierte Klasse.

## Resolver `internal/code/resolve/cpp.go`

- **Includes:**
  - `"x.h"` zuerst relativ zum Verzeichnis der Datei.
  - Dann für `"x.h"` und `<a/b.hpp>`: jede C++-Datei, deren Pfad auf
    `/a/b.hpp` endet (Endung über ganze Segmente). Genau ein Treffer ergibt
    eine Kante. Mehrere Treffer: der mit dem kürzesten Pfad, und nur wenn
    er eindeutig ist.
  - Kein Treffer: keine Kante (Systemheader).
  - Eine Kante je Paar, kleinste Zeile.
- **Typauflösung** wie bei C#, ohne `using`-Aliase:
  1. verschachtelte Typen
  2. eigener Namespace und die Namespaces darüber
  3. `using namespace X;` der Datei
  4. global
- **Definition und Klasse:** Eine Methode außerhalb der Klasse mit Owner
  `A::B` gehört zur Klasse, die die Typauflösung für `A::B` findet, auch in
  einer anderen Datei. So findet `this->f()` in `x.cpp` die Methode `f` aus
  dem Header und aus `x.cpp`.
- **Kanten:**
  - `imports`
  - `extends`; `implements` gibt es nicht, C++ kennt keine Schnittstellen
  - `calls`:
    - `f()`: erst Methoden der eigenen Klasse und ihrer Basen, dann freie
      Funktionen nach der Typ- und Namespace-Folge
    - `this->f()`
    - `A::f()`: Klasse A oder Namespace A
    - `new T()`/`T()`: der Konstruktor, sonst der Typ
  - `references` für jede Godot-Bindung auf die gebundene Methode
- **Angebote aus Bindungen** (`langOut.offers`):
  - `godot.method <Specifier>` auf die aufgelöste Methode `X::m`
  - `godot.signal <Specifier>` auf die Klasse X
  - Löst sich `X::m` nicht auf, entsteht kein Angebot.
- **`Unresolved`:** nur für `extends`.

## Godot-Verdrahtung

- **GDScript `extends GDExample` und `GDExample.new()`:** gebaut, die
  Übergabe `godot.class` trifft jetzt die C++-Klasse. Das gilt auch, wenn die
  Quellen außerhalb des Projekts liegen (`src/` neben `project/`): Angebote
  ohne Realm sieht jedes Projekt.
- **Neu: Szenenknoten mit C++-Typ.** `[node name="X" type="GDExample"]` in
  einer `.tscn` übergibt `godot.class GDExample` mit der Relation
  `references` (Szene → Klasse).
  - Der GDScript-Resolver bekommt dafür eine Übergabe.
  - Eingebaute Typen (`Sprite2D`) finden kein Angebot und bleiben ohne
    Kante; eine Liste eingebauter Typen braucht es nicht.
  - Version `gdscript/7`.
- **Neu: Verbindungen auf Knoten mit C++-Typ.** Eine Szene verbindet ein
  Signal eines GDExtension-Knotens oder ruft eine gebundene Methode auf
  einem solchen Knoten auf. Der Knoten hat kein Skript, also passt der Scope
  über Dateien nicht. Die Brücke bekommt dafür einen Scope über die Klasse:
  - **Kern:** `extract.Export` bekommt `Class string` (`omitempty`): die
    Godot-Klasse, deren Mitglied angeboten wird. `extract.Bridge` bekommt
    `Class string`.
  - **Regel in `bridgeEdges`:** Trägt eine Übergabe eine `Class`, gelten nur
    Angebote mit derselben `Class`. Ein Treffer gilt dann als „im Scope“
    (extracted), sonst greift `scopedMiss` (`fallNone`). Ohne `Class` bleibt
    alles wie bisher.
  - **GDScript-Resolver:** Ein Verbindungsende auf einem Knoten ohne
    Skript, aber mit `type="X"`, übergibt `godot.method`/`godot.signal` mit
    `Class = X`. Das gehört zusammen mit der Knotentyp-Übergabe oben zu
    `gdscript/7`.
  - **C++-Resolver:** setzt `Class` auf die Klasse der Bindung.
  - **Basisklassen:** Ein Signal, das eine Basisklasse von X bindet, wird
    nicht gefunden; nur die Klasse selbst zählt. Bekannte Lücke.

## Restliche Graph-Stellen

- **`blast.IsTestPath`:**
  - Dateiname `test_*`, `*_test.*`, `*_tests.*`, `*Test.<ext>` oder
    `*Tests.<ext>`
  - oder ein Segment heißt `test`, `tests` oder `unittest` (Groß- und
    Kleinschreibung egal)
  - Negativfälle im Test: `Contest.cpp`, `latest_version.cpp`.
- **`repomap.langFromPath`, `blast.isFileQuery`:** die Endungen.
- **`corpus.pruneDirs`:** `cmake-build-*` geht nicht über die Liste, weil
  sie Segmentnamen vergleicht. Das bleibt so; `build` steht schon drin.

## Prüfkette: Stack `cpp` (besteht)

- **Neu:** `graph` und `rules` mit denselben Befehlen wie Python.
- **`carrierRank`:** `go, python, gdscript, csharp, cpp`.
- **`IsSource` und `contentExts`:** die Endungen oben.
- **`.c` bleibt beim Stack `cpp`** für lint, types und test, wie heute. Der
  Graph liest es nicht.
- **`support.md`:** Zeile `cpp` mit graph und rules.

## Kompatibilität

- `runtime = "cpp"`, `runtimeOf["cpp"] = "cpp"`, Versionen `17`, `20`, `23`.
- **`26` ist eine Vorab-Probe des Scouts, kein Listeneintrag.** Ein Eintrag
  würde blockieren; AGENTS.md verlangt für Vorabversionen Proben, die nichts
  blockieren. Die Quelle des Scouts für `cpp` ist eine feste Angabe im Code:
  Vorab `26`, bis ISO ihn veröffentlicht und er von Hand in die Liste wandert.
- **Grammatik-Probe vor jedem Syntax-Fixture:** strukturierte Bindungen,
  `concept`/`requires`, deducing `this`, `if consteval`, `^^T`, `[: :]`,
  `pre`/`post`/`contract_assert`, `template for`, `Ts...[0]`.
  - **Ergebnis vom 2026-11-10:** C++17- und C++20-Formen, deducing `this`,
    Pack-Indexing, Contracts, `template for` und Reflexion parsen ohne
    ERROR.
  - **Lücke `if consteval` (C++23):** Die Funktion mit diesem Konstrukt wird
    zu einem ERROR, und dieser verschluckt die nächste Definition der Datei.
    Das Syntax-Fixture 23 verlangt darum nur, was trotzdem ankommt. Ein
    Extraktor-Test hält fest, wie weit der Schaden reicht. `support.md` nennt
    die Lücke.
- **Einträge:**

  | Version | Fixture | Arten |
  |---|---|---|
  | 17, 20, 23 | `cpp-cmake-gtest` | lint, types, test, coverage, graph |
  | 17, 20, 23 | `cpp-syntax-<v>` | graph |
  | 17 | `godot4-cpp` (godot-cpp-Layout: `project/` mit Szene und GDScript, `src/` mit GDExtension) | graph |

- **Fixture `cpp-cmake-gtest`:**
  - GoogleTest 1.18.0 über `FetchContent`, exakt gepinnt (Tag und Hash).
  - `prepare` setzt `CMAKE_CXX_STANDARD` auf die Version des Eintrags und
    konfiguriert `build/` (die Lanes verlangen `build/CMakeCache.txt`).
- **Scout:**
  - C++ hat keine maschinenlesbare Release-Quelle der Standards. Der Scout
    überspringt `cpp` ausdrücklich, mit Begründung im Code und in
    `support.md`; ein Test hält das fest.
  - Ein Supportende gibt es nicht. Ein Standard verlässt die Liste nur von
    Hand.
- **CI:**
  - Einrichtung in `compat-setup.sh` im Zweig `cpp)`: auf Linux kommen
    `clang-tidy` und `clang-format` per apt und `gcovr` per `uv tool install`
    mit Pin 8.8; CMake und GCC bringt der Runner mit.
  - Windows läuft für `cpp` nicht: Die Einträge tragen dasselbe Merkmal, das
    Godot-3-Einträge auf ein Betriebssystem begrenzt (nachsehen in `compat.go`
    / `compat.yml`). Gibt es dieses Merkmal nicht, ist das ein Befund für den
    Plan.

## Tests (Auszug)

- **Extraktor:**
  - Knoten je Art
  - Definition außerhalb der Klasse
  - verschachtelte Namespaces
  - Includes
  - Rohkanten mit Zeilen
  - Makro-Maske: `GDCLASS` in der Klasse, die Methoden bekommen Knoten;
    Spans aus dem Original
  - Definition unter einem ERROR-Vorfahren mit heilem Teilbaum: Knoten
  - Bindungen und das `GDCLASS`-Angebot
  - nil-Grammatik
- **Resolver:**
  - Include relativ, über die Endung, mehrdeutig
  - Methode außerhalb der Klasse wird ihrer Header-Klasse zugeordnet
  - Basen, Aufrufe, Konstruktor
  - Bindung zu Angebot
  - `extends` als `Unresolved`
- **Godot, mit echten Extraktoren:**
  - GDScript `extends` und `.new()` auf eine C++-Klasse
  - Szenenknoten-Typ
  - Quellen in `src/` neben `project/`
  - fremdes Projekt
- **Parse-Abnahme:**
  - `loomux graph build` auf godot-cpp, nlohmann/json und fmt läuft ohne
    Absturz.
  - Die Knotenzahl wird gegen eine unabhängige Zählung verglichen: per
    Regex je Datei die Funktionsköpfe vor `{`, also `) {`, `) const {` und
    `)` gefolgt von `{` auf der nächsten Zeile. Ziel sind mindestens 80 %.
  - Die 99 % der Programm-Spec zählen nur, was die Grammatik als Definition
    erkennt. Bruchstücke, die sie gar nicht erkennt, fehlen in beiden
    Spalten; dieser blinde Fleck steht in `support.md`.

## Abnahme

- `go test ./...` grün, 100 % je Funktion.
- Der Graph dieses Repos ist gleich bis auf den Stempel (kein C++ außerhalb
  von `testdata/`).
