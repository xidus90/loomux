# C# im Code-Graphen, in der Prüfkette und an Godot

**Stand:** 2026-11-10 · Teilprojekt 1 von
`2026-11-10-graph-sprachen-programm.md` · baut auf den Brücken
(`2026-11-10-graph-bruecken-design.md`)

## Ziel

C# steht in Graph, Prüfkette und Kompatibilitätsliste auf derselben Stufe wie
Python. Ein Godot-Projekt mit C#-Skripten bekommt Kanten von Szenen und
GDScript auf C#-Methoden, -Signale und `[GlobalClass]`-Klassen.

## Nicht-Ziele

- **Keine Typinferenz für lokale Variablen.** `var h = new Helper(); h.Run()`
  bleibt ohne Kante, wie in Python.
- **Kein Roslyn, kein MSBuild-Auswerten.** Projekte und Assemblies werden
  nicht gelesen; der Graph sieht Dateien und Namespaces.
- **Keine `partial`-Verbindung über Dateien hinweg** beim Godot-Methoden-
  Angebot. Eine Szene-Verbindung sucht nur im Skript, das sie nennt. Ein
  Handler in der zweiten Datei einer `partial class` bleibt ohne Kante; das
  ist eine bekannte Lücke.
- **Kein Erben über die Sprachgrenze.** C# erbt nie von GDScript und
  umgekehrt (Godot-Doku „Cross-language scripting“).

## Extraktor `internal/code/extract/csharp`

- Sprache `csharp`, Endung `.cs`, Version `csharp/1@<Parser>`, Grammatik
  `gotreesitter/grammars/c_sharp`.
- **Knoten:**
  - Datei
  - Typen: `class`, `struct`, `record`, `record struct`, `interface`, `enum`;
    Art `class` (`interface` für Schnittstellen). Verschachtelte Typen haben
    den qualifizierten Namen `Outer.Inner` und hängen per `contains` am
    äußeren Typ.
  - Mitglieder: Methoden, Konstruktoren (Name = Typname), Operatoren mit
    Rumpf, Eigenschaften und Indexer mit Accessor-Rümpfen; Art `method`,
    `Owner` = Typname
  - Delegates mit `[Signal]`: Art `signal`
- **`Exported`** ist `public` oder `internal` oder ein Mitglied einer
  Schnittstelle. Fehlt ein Modifikator, gilt die Vorgabe von C#: Typen
  `internal`, Mitglieder `private`.
- **Namespaces:**
  - Block (`namespace A.B { }`) und dateiweit (`namespace A.B;`), auch
    verschachtelt
  - `Result.Package` ist der Namespace des ersten Typs der Datei.
  - Neues Kernfeld `Result.Enclosing map[model.NodeID]string` (`omitempty`):
    der Namespace jedes Typknotens, für den Resolver. Gilt auch für C++ und
    PHP (mehrere Namespaces je Datei); Java nutzt `Package`.
  - `Node.Owner` bleibt der Typ eines Mitglieds. `rules.symbolOf` liest
    Owner als „Klasse des Symbols“; ein Namespace dort wäre falsch.
- **Imports:** `using X.Y;`, `using static X.Y.T;`, `global using`,
  `using A = X.Y.T;`. Jeder wird zu einem `extract.Import` mit Pfad und
  Alias, plus einer `imports`-Rohkante mit `Specifier`.
- **Rohkanten:**
  - `extends` je Eintrag der `base_list`; der Resolver entscheidet zwischen
    `extends` und `implements`.
  - `calls`:
    - `M()` (bare)
    - `this.M()`, `base.M()`
    - `T.M()` und `A.B.T.M()` (Receiver als Kette)
    - `new T(...)` (Konstruktor): `RawEdge.Member = "new"`. `extract.go`
      dokumentiert diese zweite Bedeutung von `Member` auf `calls`-Kanten.
  - Ein Aufruf gehört zum innersten Mitglied mit Knoten, sonst zum Typ, sonst
    zur Datei.
- **Fehlertoleranz:** Eine Definition, deren Kopf in einem ERROR liegt,
  bekommt keinen Knoten, wie in Python. Der Rest der Datei wird gelesen.
- **Godot-Angebote** (`Result.Exports`; Ziel ist der Knoten):
  - Jeder Typ mit Attribut `GlobalClass`: `godot.class <Name>`
  - **Skriptklasse:** Godot verlangt, dass die Klasse eines C#-Skripts wie
    die Datei heißt (`Player.cs` → `Player`). Nur deren Methoden und Signale
    werden angeboten, je Name einmal (die erste Deklaration; Überladungen
    geben kein zweites Angebot). Verschachtelte Typen und andere Klassen der
    Datei bieten nichts an. So bleibt `cardOne` im genannten Skript erfüllt.
  - Methode: `godot.method <Name>`
  - Delegate mit `[Signal]`, der auf `EventHandler` endet:
    `godot.signal <Name ohne EventHandler>`. Ein `[Signal]`-Delegate ohne
    diese Endung wird nicht angeboten; Godot verlangt sie.

## Resolver `internal/code/resolve/csharp.go`

Ein Index über die C#-Dateien allein:
- Namespace → Typen
- Typ → Mitglieder und Basen
- Datei → `using`-Liste

**Typauflösung** eines Namens N in Datei F mit Namespace S, in dieser Folge:
1. verschachtelte Typen des umgebenden Typs
2. Typen in S und dessen umschließenden Namespaces, innen nach außen
3. `using`-Aliase der Datei
4. Typen der per `using` (auch `global using` irgendeiner Datei) geöffneten
   Namespaces
5. der globale Namespace

Ein Treffer ergibt eine Kante. Mehrere Treffer auf derselben Stufe ergeben
`ReasonAmbiguous`, keiner `ReasonNone`. Gilt nur für Namen der eigenen Sprache.

**Partielle Typen:** Gleicher Namespace und gleicher qualifizierter Name sind
ein logischer Typ. Zwei Duplikate ohne `partial` kompilieren nicht, das
Schlüsselwort wird darum nicht gelesen.
- Die Mitglieder und Basen aller Teile werden vereinigt.
- Ziel einer Kante auf den Typ ist der Teil in der Datei mit dem kleinsten
  Pfad.
- `this.M()` findet M auch im anderen Teil.
- Godot-Angebote bleiben je Datei (siehe oben).

**Kanten:**
- **`imports`:** wie Go auf ein Paket. Ziel ist eine Vertreterdatei des
  Namespace: die mit dem kleinsten Pfad unter den Nicht-Testdateien, die ihn
  deklarieren (sonst unter allen). Eine Kante je Paar, kleinste Zeile, nie
  auf sich selbst. Kein Treffer im Repo: keine Kante (wie Python). Eine Kante
  auf jede Datei des Namespace würde jeden Nutzer zum Abhängigen jeder Datei
  machen und den Blast aufblähen.
- **`extends` / `implements`:** Ziel ist eine Schnittstelle → `implements`,
  sonst `extends`. Nicht aufgelöst → `Unresolved`.
- **`calls`:**
  - `M()`: Mitglied des eigenen Typs, dann seiner Basen (rekursiv,
    zyklenfest), dann `using static`-Typen
  - `this.M()`: wie `M()` ohne `using static`
  - `base.M()`: erst die Basen
  - `T.M()`: Typauflösung von T, dann dessen Mitglied M
  - `new T()`: Konstruktor von T, sonst der Typ selbst
  - Überladungen: Bei gleichem Namen zeigt die Kante auf die erste
    Deklaration im Quelltext. Signaturen werden nicht verglichen.
- Nicht aufgelöste Aufrufe ergeben keine `Unresolved`-Einträge; die gibt es
  nur für `extends`, wie in Python.

## Godot-Verdrahtung

Gebaut ist schon alles in den Brücken. Neu ist nur der Anbieter.
- **Szene → C#-Methode:** `[connection]` mit `method="OnHit"` auf einen
  Knoten mit `Player.cs` ergibt eine Kante auf `Player.cs#Player.OnHit`.
- **Szene → C#-Signal:** `signal="Hit"` auf einen Knoten mit C#-Skript
  ergibt eine Kante auf den Delegate.
- **GDScript `X.new()`** auf eine `[GlobalClass]`-Klasse ergibt eine
  `calls`-Kante. Ob Godot einen C#-Global-Namen in GDScript kennt, ist nicht
  belegt; der Code nennt den Namen aber, und ohne die Brücke bliebe er ohne
  Ziel. Das Compat-Fixture verlangt diese Kante darum nicht, nur der
  Unit-Test.
- **Pfade:** Eine Szene nennt ihr C#-Skript per `res://`-Pfad. Diese Kante
  baut der GDScript-Resolver schon heute über die Pfadmenge aller Dateien.
  Ein Test beweist, dass sie jetzt auf einen echten C#-Dateiknoten zeigt.

## Restliche Graph-Stellen

- `blast.IsTestPath` für `.cs`: Datei endet auf `Tests.cs` oder `Test.cs`,
  oder ein Pfadsegment endet auf `.Tests` oder `.Test`, oder ein Segment heißt
  `tests` oder `test`
- `ask.testNames`: dieselben Muster
- `repomap.langFromPath`: `.cs` → `csharp`
- `blast.isFileQuery`: `.cs`
- `corpus.pruneDirs`: `obj` (Zwischenausgabe von MSBuild). `bin` nicht: Die
  Liste gilt für alle Sprachen, und `bin/` hält in vielen Repos echte
  Skripte.

## Prüfkette: Stack `csharp`

- **Endungen:** `.cs` → `csharp`, `.csproj` und `.sln` werden nicht
  zugeordnet.
- **Erkennung** (`detect/signals.go`): `*.sln`, `*.slnx`, `*.csproj`,
  `global.json`, `Directory.Build.props`. Variante `xunit` usw. gibt es
  nicht; `dotnet test` ist unabhängig vom Framework.
- **Lanes:**
  - lint: `dotnet format --verify-no-changes`; auf Datei
    `dotnet format --verify-no-changes --include {file}`
  - types: `dotnet build --nologo -warnaserror`
  - test: `dotnet test --nologo`; messend mit
    `--collect "XPlat Code Coverage" --results-directory {coverdir}` (genauer
    Platzhalter im Plan nach `verify/placeholders`)
  - coverage: `{loomux} check linecov --report <cobertura>`, nach `test`
  - graph und rules: dieselben Befehle wie Python
  - `tests_when`: ein `.csproj`, das `Microsoft.NET.Test.Sdk` nennt
- **Rang:** `carrierRank` wird `go, python, gdscript, csharp`; `IsSource`
  und `contentExts` bekommen `.cs`.
- **Config-Schema:** keine Werkzeug-Zeile; das SDK bringt alles mit.
- **Doku:** `support.md` (en, de) mit der Zeile `csharp`, alle sechs Arten.

## Kompatibilität

- **Neu:** `runtime = "dotnet"`, `runtimeOf["csharp"] = "dotnet"`. Das
  Versionsformat wird `\d+(\.\d+)?` (Programm-Spec).
- **Einträge:**

  | Version | Fixture | Arten |
  |---|---|---|
  | 10.0 | `csharp-xunit` | lint, types, test, coverage, graph |
  | 11.0 | `csharp-xunit` | lint, types, test, coverage, graph |
  | 10.0 | `csharp-syntax-10.0` (C# 14) | graph |
  | 11.0 | `csharp-syntax-11.0` (C# 15) | graph |
  | 10.0 | `godot4-csharp` (Szene, GDScript und C#) | graph |

- **`godot4-csharp`:** `expect.toml` verlangt die Kanten Szene → C#-Methode
  und Szene → C#-Signal.
- **Scout:**
  - Releases aus
    `https://raw.githubusercontent.com/dotnet/core/main/release-notes/releases-index.json`;
    ein neuer Parser, Vorabversionen an `preview`/`rc` erkannt
  - Supportende über endoflife.date `dotnet`
- **CI:** `actions/setup-dotnet` in `compat.yml` mit der Version des
  Eintrags; `compat-setup.sh` bekommt den Zweig `dotnet)`, der nichts tut,
  wie `go)`.
- **Syntax-Fixtures:** Formen, die C# 14 bzw. C# 15 neu brachten (aus der
  Recherche). Kann die Grammatik eine Form nicht, wird das als Lücke mit Test
  festgehalten: Der Test zeigt, welche Knoten trotzdem ankommen.
- **Geprobt am 2026-11-10:**
  - **C# 14** parst ohne Fehler. Die Mitglieder eines `extension(...)`-Blocks
    (`extension_declaration` → `extension_body`) bekommen Knoten mit dem
    umgebenden statischen Typ als Owner.
  - **C# 15** hat vier Grammatik-Lücken. Die vierte fand der Compat-Lauf: Ein
    Erweiterungs-Indexer kostet den umgebenden Typ seinen Knoten. Die drei
    übrigen:
    - `union Pet(Cat, Dog);` wird als lokale Funktion gelesen und bekommt
      keinen Typknoten.
    - `closed record class X;` wird zu einem ERROR, also kein Knoten.
    - `break outer;` erzeugt einen ERROR im Rumpf; die Mitglieder danach
      kommen an.
  - `support.md` nennt die Lücken. Das Syntax-Fixture 11.0 verlangt nur, was
    trotzdem ankommt.

## Tests (Auszug; der Plan nennt jeden)

- **Extraktor:**
  - Name, Version, Endungen
  - Knoten je Deklarationsart, verschachtelt, Namespaces (Block,
    dateiweit, verschachtelt)
  - Imports in allen vier Formen
  - Rohkanten mit Zeilen
  - Godot-Angebote (`GlobalClass`, `[Signal]` mit und ohne `EventHandler`,
    Methoden)
  - Definition im ERROR ohne Knoten
  - Parse-Fehler über die nil-Grammatik
- **Resolver:**
  - jede Stufe der Typauflösung
  - Mehrdeutigkeit
  - `implements` gegen `extends`
  - `base.M()` und Basen-Zyklus
  - `using static`
  - Konstruktor
  - Import-Dedup
- **Brücken mit echtem Extraktor** (`GraphWith` aus gdscript + csharp):
  - Szene → Methode
  - Szene → Signal
  - `.new()` → `GlobalClass`
  - Methode in zweiter `partial`-Datei ohne Kante
  - anderes Godot-Projekt ohne Kante
- **Prüfkette und Compat** über die bestehenden Spiegel-Tests.

## Abnahme

- `go test ./...` grün, Coverage 100 % je Funktion.
- `loomux graph build` auf `jellyfin` und auf den C#-Demos aus
  `godot-demo-projects`: kein Absturz, ParseErrors wie in der Parse-Prüfung,
  Kanten Szene → C# in den Demos vorhanden.
- Der Graph dieses Repos ist vor und nach der Änderung gleich, bis auf den
  Extraktor-Stempel. Das Repo hat keine `.cs`-Datei außerhalb von
  `testdata/`, und `testdata/` wird nicht gelesen (geprüft am 2026-11-10).
