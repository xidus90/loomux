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
    content_hash: "sha256:d75c995eb2af5df759fc9a06ba0e5695a94a99dbcf736283133b0c3e62ac4210"
    revision: 9
  - id: cli-referenz
    resource: brain://project/loomux/docs/de/cli-reference.md
    doc_id: 01M47W91R5FRSPN6BE643656ZE
    content_hash: "sha256:ef612ecf8271f6d70ee53a169f4734b13920d7dd2bd74eb41d41863e2e471003"
    revision: 53
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

- **C#** liest er auf demselben Kern (`internal/code/extract/csharp`, Endung
  `.cs`): Dateien, Typen (auch `record`, `struct`, `interface`, `enum`,
  verschachtelt als `Outer.Inner`), Methoden, Konstruktoren, Operatoren,
  Eigenschaften, Indexer und `[Signal]`-Delegates, Syntax bis C# 14
  vollständig. Namespaces sind keine Knoten; `Result.Enclosing` trägt den
  Namespace jedes Typknotens für den Resolver, und `Node.Owner` bleibt der
  Typ eines Mitglieds. Ein `using` (auch `using static`, `using A = X.Y.T`
  und `global using`, das in jeder Datei gilt) ist ein Import. Es gibt kein
  Roslyn und kein MSBuild, also keine Typen lokaler Variablen und keine
  Projektdateien. Der Resolver (`internal/code/resolve/csharp.go`) sucht
  einen Typnamen in dieser Folge: verschachtelte Typen des umgebenden Typs,
  der eigene Namespace und die umschließenden von innen nach außen,
  `using`-Aliase, geöffnete Namespaces, der globale Namespace; mehrere
  Treffer auf einer Stufe sind mehrdeutig und ergeben keine Kante. Die Teile
  einer `partial class` sind ein logischer Typ (gleicher Namespace und
  Name): ihre Mitglieder und Basen werden vereinigt, eine Kante auf den Typ
  zeigt auf den Teil mit dem kleinsten Pfad. Ein `using` zeigt wie ein
  Go-Import auf eine Vertreterdatei des Namespace (kleinster Pfad unter den
  Nicht-Testdateien), nicht auf jede Datei, sonst würde jeder Nutzer zum
  Abhängigen jeder Datei und der Blast-Radius bliebe aufgebläht. Ein
  `new T()` (`RawEdge.Member = "new"`) trifft den Konstruktor, sonst den Typ;
  bei Überladungen zeigt die Kante auf die erste Deklaration. Ein
  unqualifiziertes `M()` sucht im eigenen Typ und seinen Basen, dann in
  jedem umgebenden Typ samt Basen von innen nach außen, zuletzt im einen
  Typ, den ein `using static` anbietet (dasselbe `using static` doppelt in
  Kraft zählt einmal). Die Basis eines Primärkonstruktors
  (`record D(int A) : Base(A)`) ist eine `extends`-Kante auf `Base`.
  Attribute erkennt er auch qualifiziert und mit Endung (`[Godot.Signal]`,
  `[GlobalClassAttribute]`). Wo die
  Grammatik scheitert, bleibt der Rest der Datei erhalten; C# 15 hat vier
  bekannte Lücken (`union` und `closed record class` ohne Typknoten, ein Typ
  mit Erweiterungs-Indexer ohne Knoten, `break`/`continue` mit Marke als
  Fehler im Rumpf). `corpus.pruneDirs` kennt `obj`, die Zwischenausgabe von
  MSBuild, nicht `bin`
  (`docs/.superpowers/specs/2026-11-10-csharp-design.md`).

- **C++** liest er auf demselben Kern (`internal/code/extract/cpp`, Endungen
  `.cpp`, `.cc`, `.cxx`, `.c++`, `.hpp`, `.hh`, `.hxx`, `.h++`, `.h`, `.inl`,
  `.ipp` und `.tpp`, nicht `.c`): Dateien, Klassen, Structs, Unions und Enums
  mit Rumpf (verschachtelt als `Outer.Inner`), Funktionen und Methoden mit
  Rumpf, auch die außerhalb der Klasse (`void ns::A::m() {}`). Deren
  `Node.Owner` ist das letzte Segment des Qualifiers; ob das eine Klasse oder
  ein Namespace ist, entscheidet erst der Resolver (`void util::z() {}` hat den
  Owner `util` und wird dort eine freie Funktion im Namespace `util`).
  Namespaces sind keine Knoten, und `Result.Enclosing` trägt den Namespace
  jedes Typs und jeder freien Funktion, bei einer Methode außerhalb der Klasse Namespace und ganzen
  Qualifier, deren Aufteilung der Resolver klärt. Es gibt keinen Präprozessor:
  jeder `#if`-Zweig wird gelesen, Makros bleiben unexpandiert, ein `#include`
  ist nur ein Import. Drei Eingriffe halten den Parser bei Laune. Eine Zeile,
  die nur ein Makro in Großbuchstaben mit Argumentliste trägt (`GDCLASS(X, Y)`,
  `Q_OBJECT`), wird vor dem Parsen durch gleich viele Leerzeichen ersetzt,
  sofern nicht `{` folgt; die Offsets bleiben, Spans und Rumpftexte stammen aus
  dem Original. Die rechte Seite eines Alias mit Template-Argumenten und
  Qualifier (`using X = A<B<T>>::type;`) wird außerhalb von Kommentaren und
  Literalen zu `int` maskiert, weil `gotreesitter` den Baum an dieser Stelle
  ohne Meldung beendet, die C-Laufzeit aber nicht. Endet der Baum trotzdem vor
  dem Quelltext, liest der Extraktor den Rest blockweise neu. Eine Definition
  zählt, wenn ihr eigener Teilbaum keinen ERROR enthält, auch unter einem
  ERROR-Vorfahren; ein zerbrochener Klassenkopf gibt der Methode keinen
  erfundenen Owner. Eine Klasse mit einem ERROR im Rumpf bekommt keinen
  Knoten; ihre heilen Methoden behalten sie als Owner, gehören im Resolver aber
  zu keinem Typ, und zwei gleichnamige Klassen in zwei Namespaces einer Datei
  bleiben zwei Typen. Der Resolver (`internal/code/resolve/cpp.go`) sucht ein
  `#include "x.h"` zuerst neben der Datei, dann über die Pfadendung in ganzen
  Segmenten (bei mehreren Treffern gilt der kürzeste Pfad, wenn er eindeutig
  ist; ein Systemheader ergibt keine Kante). Einen Typnamen sucht er in dieser
  Folge: verschachtelte Typen, der eigene Namespace und die umschließenden,
  `using namespace` der Datei, global. Eine Methode außerhalb der Klasse gehört
  zu der Klasse, die diese Folge für ihren Qualifier findet, auch in einer
  anderen Datei; so trifft `this->f()` die Methode aus dem Header und aus der
  `.cpp`. Eine `static`-Funktion und ein Typ in einem anonymen Namespace
  gelten nur in ihrer Datei und gehen auf derselben Stufe der Folge vor, nicht
  vor einer näheren (`a::helper` schlägt im Namespace `a` ein dateilokales
  globales `helper`). Ein Aufruf `x.f()` oder `p->f()` auf einem Objekt
  bleibt ohne Kante, weil der Graph keine Typen kennt; `f()`, `this->f()`,
  `A::f()` und `new T()` oder `T()` (der Konstruktor, sonst der Typ) lösen
  sich auf. Nur eine Basis ohne Ziel steht unter `Unresolved`. Für Godot bietet
  eine Klasse mit `GDCLASS(<Name>, …)` `godot.class` an, und der Rumpf von
  `_bind_methods` ergibt für `ClassDB::bind_method` und `ADD_SIGNAL`
  `references`-Kanten und `godot.method`- und `godot.signal`-Angebote mit der
  Klasse als `Class`
  (`docs/.superpowers/specs/2026-11-10-cpp-design.md`).

- **Java** liest er auf demselben Kern (`internal/code/extract/java`, Endung
  `.java`): Dateien, Klassen, Schnittstellen, Enums, Records und
  Annotationstypen (verschachtelt als `Outer.Inner`), Methoden und
  Konstruktoren, auch in Enum-Rümpfen; der kompakte Konstruktor eines Records
  heißt wie der Record, und die Methoden einer kompakten Quelldatei (Java 25)
  sind Funktionen. Ein anonymer Typ hat keinen Knoten, seine Aufrufe gehören
  zum Mitglied darum herum. Das Paket ist kein Knoten: `Result.Package` trägt
  den Namen aus `package a.b;` für den Resolver, denn Java hat ein Paket je
  Datei und keinen Namespace, der Typen umschließt. Ein `import` (einzeln,
  `*`, `static` und `import module`) ist ein Import. Es gibt keinen Compiler
  und keine `pom.xml`-Auswertung, also keine Typen lokaler Variablen und
  keine Quellwurzeln: das Paket steht in der Datei. Der Resolver
  (`internal/code/resolve/java.go`) sucht einen Typnamen in dieser Folge:
  verschachtelte Typen des umgebenden Typs und der äußeren, Typen der Datei,
  Einzel-Typ-Importe, Typen desselben Pakets (in jeder Datei), Typen der
  `*`-Importe, voll qualifizierte Namen; mehrere Treffer auf einer Stufe sind
  mehrdeutig und ergeben keine Kante. Ein Einzel-Typ-Import und ein
  `static`-Import zeigen auf die Datei, die den Typ deklariert, ein
  `*`-Import wie ein Go-Import auf eine Vertreterdatei des Pakets (kleinster
  Pfad unter den Nicht-Testdateien); ein Paket ohne Datei im Graphen
  (`java.util`) und `import module` ergeben keine Kante. Eine Basis ist
  `extends` oder `implements`, je nach Art des Ziels; nur eine Basis ohne Ziel
  steht unter `Unresolved`. Ein unqualifiziertes `m()` sucht im Typ und
  seinen Basen (zyklenfest), dann in den umgebenden Typen, dann unter den
  Methoden auf oberster Ebene einer kompakten Quelldatei (Java 25; auch aus
  einer Klasse, die sie deklariert), zuletzt in den `static`-Importen; `this.m()`, `super.m()` (dieses nur in den Basen),
  `X.m()` über die Typauflösung und `new X()` (der Konstruktor, sonst der
  Typ) lösen sich auf, bei Überladungen gilt die erste Deklaration. Ein
  `obj.m()` auf einer Variablen bleibt ohne Kante. Wo die Grammatik
  scheitert, bleibt der Rest der Datei erhalten; Java 8 bis 21 parsen ohne
  Fehler, Java 25 hat zwei Lücken (`import module` kommt als Fehler im
  Import, den der Extraktor trotzdem liest, und eine Anweisung vor
  `super(…)` ist ein Fehler im Rumpf des Konstruktors, dessen Aufrufe fehlen;
  die Mitglieder danach kommen an). Java 27 bringt keine eigene Syntax, das
  Syntax-Fixture belegt die Formen von 25. `corpus.pruneDirs` kennt `target`
  und `build`
  (`docs/.superpowers/specs/2026-11-11-java-design.md`).

- **PHP** liest er auf demselben Kern (`internal/code/extract/php`, Endung
  `.php`): Dateien, Klassen, Schnittstellen, Traits und Enums, Methoden und
  Funktionen, wo immer die Deklaration läuft (PHP legt eine Funktion in einem
  `if` oder Rumpf im Namespace der Datei an, also bekommt sie dort ihren
  Knoten). Ein Namespace ist kein Knoten und steht in keiner Kennung:
  `Result.Enclosing` trägt ihn je Typ und Funktion, `Result.Package` den der
  ersten Deklaration. Das ist eine Vereinfachung des Resolvers, nicht die Regel
  von PHP: Der Resolver nimmt an, dass Code außerhalb jeder Deklaration in ihm
  läuft, PHP führt ihn im Namespace seines eigenen `namespace`-Abschnitts aus;
  auch
  mehrere `namespace`-Blöcke einer Datei und HTML dazwischen (der Text außerhalb
  von `<?php … ?>` nennt nichts) kommen an. Ein anonymer Typ hat keinen
  Knoten, seine Aufrufe gehören zum Code darum herum. Ein Name behält seine
  Qualifizierung; die rohe Kante trennt ihn am letzten Backslash, der Receiver
  ist, was davor steht. Ein `use` (Klasse, `use function`, `use const`, auch
  gruppiert) ist ein Import mit der Art `function` oder `const`, ein
  `require` oder `include` (auch `_once`) mit einem Literal oder einem an
  `__DIR__` oder `dirname(__FILE__)` gehängten Literal einer mit der Art
  `require`; was erst zur Laufzeit entsteht, ist keiner. Ein statischer Aufruf
  trägt in `RawEdge.Member` die Marke `::`, ein Aufruf über eine Variable
  (`$obj->m()`, `$obj::m()`) die Marke `.`, `new X()` die Marke `new`.
  Der Resolver (`internal/code/resolve/php.go`) vergleicht Namen ohne
  Rücksicht auf die Schreibung, wie PHP, und löst sie so auf: `\A\B` voll
  qualifiziert, `namespace\B` im aktuellen Namespace, ein unqualifizierter
  Name über das `use` seiner Art, `A\B` über das `use` oder den Alias von `A`,
  sonst im aktuellen Namespace; eine Klasse verlässt ihn nie, nur eine
  unqualifizierte **Funktion** fällt auf den globalen zurück. Deklarieren
  zwei Dateien denselben Namen, gilt der kleinste Pfad. Ein `use` einer Klasse
  oder `use function` zeigt auf die Datei, die den Namen deklariert, ein
  `require` auf die Datei, die sein Pfad relativ zur einbindenden nennt;
  `use const`, das `use` eines Namespace und ein Name ohne Datei im Graphen
  ergeben keine Kante. Eine Basis ist `extends` oder `implements` nach der Art
  des Ziels; ein Trait, den ein `use` im Rumpf nennt, ist eine Basis wie
  jede andere und geht beim Suchen den übrigen Basen vor (eine Methode des
  Traits überschreibt die geerbte); nur eine Basis ohne Ziel steht unter
  `Unresolved`. `f()`, `A\f()`, `$this->m()`, `self::m()`, `static::m()`,
  `parent::m()` (nur in den Basen, nie in einem Trait), `X::m()` und `new X()`
  (der Konstruktor, sonst der Typ) lösen sich auf; `$obj->m()` bleibt ohne
  Kante. Wo die Grammatik scheitert, bleibt der Rest der Datei erhalten; PHP
  8.2 bis 8.4 parsen ohne Fehler, die partielle Anwendung von 8.6 auch, und 8.5
  hat zwei Lücken. `corpus.pruneDirs` kannte `vendor` schon, ein Projekt mit
  `composer.json` hat darum nie die Pakete im Graphen
  (`docs/.superpowers/specs/2026-11-11-php-design.md`).

- **JavaScript und TypeScript** liest er auf demselben Kern, mit einem
  Extraktor für zwei Sprachen (`internal/code/extract/ecmascript`):
  `javascript` für `.js`, `.mjs`, `.cjs` und `.jsx`, `typescript` für `.ts`,
  `.mts`, `.cts` und `.tsx`. `.jsx` und `.tsx` liest die tsx-Grammatik, weil
  die JavaScript-Grammatik am JSX-Attribut `class=` (Solid, Preact) bricht;
  `.js` bleibt bei der JavaScript-Grammatik. Knoten sind Funktionen und
  Klassen auf oberster Ebene des Moduls, auch hinter `export` und `export
  default` (ein anonymer Standard-Export heißt `default`), Methoden (der
  Konstruktor als `constructor`), ein `const`, `let` oder `var`, das an eine
  Funktion oder einen Pfeil gebunden ist, CommonJS `exports.f = …` und
  `module.exports.f = …`, in TypeScript dazu Schnittstellen, Typaliase, Enums
  und jede Deklaration eines `namespace` als `N.f`; `export = function …`
  und `export = class …` heißen `default` wie `module.exports = function
  …`. Verschachtelte
  Funktionen, Objektmethoden und Ambient-Deklarationen bekommen keinen
  Knoten; eine Deklarationsdatei (`.d.ts`, `.d.mts`, `.d.cts`) behält nur
  ihren Dateiknoten, an dem der Resolver sie erkennt. `Exported` trägt,
  was `export` trägt oder eine Export-Liste nennt. Jede Bindung eines Imports
  ist ein `extract.Import`: `Name` ist der Name in der Zieldatei, `Alias` der
  lokale, `Kind` unterscheidet `import`, `require`, `import()`, Re-Export und
  Export-Liste. Der Resolver (`internal/code/resolve/ecmascript.go`) löst
  beide Sprachen gemeinsam auf, als eine **Familie**, weil echte Projekte sie
  mischen. Einen Importpfad liest er wie eine URL, mit derselben Funktion
  wie das `src` eines `<script>` (`localPath`): Query und Fragment (Vites
  `?worker`) fallen weg, Escapes werden dekodiert, ein führendes `#` bleibt
  Name. Einen relativen Import probiert er wie geschrieben, dann mit
  `.ts`, `.tsx`, `.d.ts`, `.js`, `.jsx`, `.mjs`, `.cjs`, dann als `/index`;
  in einer TypeScript-Datei findet `./a.js` auch `a.ts` (node16-Regel von
  tsc), und eine getroffene Deklarationsdatei gibt dort keine Kante. Jeder
  andere Name geht über `paths` und `baseUrl` der nächstgelegenen
  `tsconfig.json` oder `jsconfig.json`; der Build reicht sie wie `go.mod` als
  `resolve.Module` herein, gelesen ohne `extends`-Kette, mit Kommentaren und
  nachgestellten Kommas. Eine verschachtelte Konfiguration ohne `paths` und
  `baseUrl` gibt kein Modul, ihre Dateien nehmen die Aliase der äußeren; eine
  unlesbare überspringt der Build, nur ein unlesbares `go.mod` lässt ihn
  scheitern. Ein Paketname ergibt keine Kante. Ein importierter Name zeigt
  auf das gleichnamige Export-Symbol der Zieldatei, Re-Exporte und
  `export *` werden zyklenfest verfolgt, CommonJS-`require` bindet einen
  Namensraum oder, destrukturiert, einen Namen. `f()`, `ns.f()` über einen
  importierten Namensraum, `this.m()`, `super.m()`, `new X()` (der
  Konstruktor, sonst die Klasse) und ein JSX-Tag mit großem
  Anfangsbuchstaben, `<X/>` oder `<ns.X/>`, als Aufruf der Komponente lösen
  sich auf (React, Preact und Solid rufen sie dort auf; ein kleingeschriebenes
  Tag ist ein DOM-Element); nur eine Basis, deren Namen ein Import an eine
  Datei des Repositorys bindet, steht unter `Unresolved`.
  Node 22, 24 und 26 sind über die Kompatibilitätsliste geprüft; Node 26
  bringt keine eigene Syntax, das Syntax-Fixture belegt die Formen von 24 und
  TypeScript daneben. `corpus.pruneDirs` kannte `node_modules` und `dist`
  schon (`docs/.superpowers/specs/2026-11-11-ecmascript-design.md`). Ein
  minifiziertes Bündel (`*.min.js`, `*.min.mjs`, `*.min.cjs`) lässt
  `internal/code/sourceset` aus: Es ist generiert, liegt oft unter der
  Größengrenze und kostet Sekunden an Parsefehlern (d3.min.js: 2 s, 16
  Fehler).

- **HTML, CSS und SCSS** liest er auf demselben Kern, mit zwei Extraktoren
  für drei Sprachen: `html` (`internal/code/extract/html`, `.html` und
  `.htm`, Version `html/1`) und `css` und `scss`
  (`internal/code/extract/stylesheet`, `.css` und `.scss`, Versionen
  `css/1` und `scss/1`). Eine Seite hat ihren Dateiknoten und je Element
  mit `id` einen Knoten der Art `element` namens `#id`; ein zweites
  Element derselben `id` bekommt eine Ordinalzahl. `<link
  rel="stylesheet" href>` und `<script src>` sind Importe der Arten
  `stylesheet` und `script`, `<a href>` ist Navigation und keiner. Die
  Rümpfe der Inline-Skripte einer Sprache und aller `<style>` sind je ein
  **eingebetteter Teil** (`Result.Embedded`), ein Teil je Sprache und Seite,
  nicht je Block: Der Extraktor der Sprache liest eine Kopie der Seite, in
  der alles außer diesen Rümpfen ausgeblendet ist (`mask`), sodass Offsets,
  Zeilen und Spans in die Seite zeigen, und hinter jeden Skript-Rumpf
  schreibt die Maske ein `;`, damit zwei Skripte nicht zu einer Anweisung
  verschmelzen. Ein Skript ist JavaScript ohne `type`, mit `module` oder
  einem JavaScript-MIME-Typ, TypeScript mit `lang="ts"` oder
  `type="text/typescript"`; jeder andere Typ ist Daten oder Vorlage. Ein
  Teil hat keinen eigenen Dateiknoten, seine Definitionen hängen per
  `contains` an der Seite, und die IDs vergibt die Seite gemeinsam: erst
  die Elemente, dann jeder Teil, also sind `<div id="app">` und
  `<style>#app{}` `p.html##app` und `p.html##app~2`. Eine Regel
  (`rule_set`, auch in `@media`, `@supports`, `@layer` oder einem Mixin)
  ist ein Knoten der Art `rule`; ihr **Name** ist die Selektorliste mit
  Leerraum zu einem Leerzeichen gefaltet, `~` bleibt erhalten (es steht in
  einem Selektor nie vor einer Ziffer, und nur `~` mit Ziffern ist eine
  Ordinalzahl), die Signatur hält den Text wie geschrieben, und eine
  verschachtelte SCSS-Regel heißt `Eltern › Kind`. SCSS fügt `@mixin`
  (Art `mixin`), `@function` (Art `function`) und Platzhalter `%p` (Art
  `rule`) hinzu. Jede Klasse eines Selektors bietet `css.class` an, jede
  ID `css.id`, und `&__el` hängt sein Suffix an die Klassen, auf die der
  Elternselektor endet. `@include` und der Aufruf einer Funktion in einem
  SCSS-Wert sind `calls`, `@extend` ist `extends`; plain CSS ruft nichts.
  Wo die SCSS-Grammatik eine ganze Datei in einen ERROR zöge, maskiert der
  Extraktor drei Formen vor dem Parsen in einer Kopie gleicher Länge: das
  `%` von `@extend %p`, ein Map-Literal `($k: v)` als Argument und `#{$v}`
  in einem Namen in einem Wert. HTML ist eine eigene Familie, CSS und SCSS
  sind eine (`stylesheet`). **Alle Verweise einer Seite liest eine
  Funktion, `pageRef`** (`internal/code/resolve/web.go`): `<script src>`,
  `<link href>`, `@import`, `@use` und der relative Stylesheet-Import eines
  Skripts werden über `localPath` als URL gelesen, `/x` ist `x` an der
  Wurzel des Repositorys, jeder andere Pfad relativ zum Ordner der Datei.
  Ein SCSS-Import probiert mit Endung den Pfad und sein Partial
  (`_x.scss`), ohne Endung `_x.scss`, `x.scss` und `x/_index.scss`. `@include m` und `f()` lösen
  sich in der Breite über die Importe auf, die Datei selbst zuerst, `ns.m`
  nur über das `@use`, das `ns` bindet, und dessen `@forward`-Kette; die
  erste Tiefe mit dem Namen entscheidet, und nur mit genau einem Treffer.
  `@extend` sieht jede Tiefe. Was die Seiten laden und verlinken, liest
  `webOf` einmal je Build aus allen Seiten; es gibt den Geltungsbereich der
  Web-Brücken
  (`docs/.superpowers/specs/2026-11-11-web-design.md`).

Jede Familie löst in einem eigenen Namensindex auf (`familyOf` in
`internal/code/resolve/resolve.go`): meist ist das eine Sprache, nur
JavaScript und TypeScript, die einander importieren, teilen sich einen,
ebenso CSS und SCSS. Eine
Familie löst an der Stelle ihrer ersten Sprache in der sortierten Liste auf,
damit der Graph einer Familie aus einer Sprache Byte für Byte gleich bleibt;
das Meta des Graphen listet weiter die einzelnen Sprachen. Über eine
Sprachgrenze geht ein Godot-Pfad, der eine Datei nennt, die ein anderer
Extraktor liest (etwa eine `.py`-, `.go`- oder `.cs`-Datei): er landet auf
deren Dateiknoten. Ebenso landen `<link>` und `<script src>` einer Seite und
der Stylesheet-Import eines Skripts auf dem Dateiknoten des Stylesheets oder
Skripts. Eine Datei, die kein Extraktor liest (eine Textur), ergibt
keine Kante.

Symbol zu Symbol über eine Sprachgrenze geht nur über eine **Brücke**
(`internal/code/resolve/bridge.go`): Eine Seite bietet einen Namen in einem
Namensraum an (Export), die andere nennt ihn (Brücke), und nachdem jede
Familie aufgelöst ist, verbindet ein sprachneutraler Schritt beide nach den
Regeln des Namensraums: genau ein Treffer oder alle, ein Geltungsbereich
(für `godot.*` das Godot-Projekt; ein Export außerhalb jedes Projekts, etwa
GDExtension-Quellen neben `project/`, gilt in allen), und ob ohne Treffer im
genannten Bereich gesucht oder geraten wird. Den Namen eines Exports legt die
anbietende Seite fest. GDScript bietet jede `class_name` eines Projekts als
`godot.class` an und übergibt, was es selbst nicht binden kann: ein
`extends X` und `X.new()` mit flachem Namen und das Ende einer Szenen-
`[connection]`, deren Skript keine GDScript-Datei ist. Eigene Exports
beantworten eigene Übergaben nie, geprüft je Familie: Eine Übergabe, die der
Resolver von GDScript weiterreicht, beantwortet kein GDScript-Export, eine
des Resolvers von JavaScript und TypeScript kein Export dieser Familie, denn
er hat ihren Index schon durchsucht. C# bietet an, was Godot kennt: jeden Typ mit
`[GlobalClass]` als
`godot.class` und, aus der Klasse, die wie die Datei heißt (`Player.cs` →
`Player`, so verlangt es Godot), jede Methode als `godot.method` und jedes
`[Signal]`-Delegate auf `EventHandler` als `godot.signal` ohne diese Endung,
je Name einmal. Damit trifft eine Szenen-`[connection]` mit `method="OnHit"`
die C#-Methode und eine mit `signal="Hit"` das Delegate (`references`), und
ein GDScript-`X.new()` auf eine `[GlobalClass]`-Klasse ergibt eine
`calls`-Kante; ob Godot den C#-Namen in GDScript kennt, ist nicht belegt, der
Code nennt ihn aber, und ohne die Brücke bliebe er ohne Ziel. C++ bietet
`godot.class` für jede Klasse mit `GDCLASS` an und für jede Bindung in
`_bind_methods` `godot.method` und `godot.signal`; ein GDScript-`extends` und
`.new()` und der Knotentyp einer Szene (`[node … type="GDExample"]`, eine
`references`-Kante von der Szene auf die Klasse) treffen so die C++-Klasse,
auch wenn die Quellen außerhalb des Projekts liegen (`src/` neben `project/`).
Ein eingebauter Typ (`Sprite2D`) findet kein Angebot und bleibt ohne Kante. Eine
Verbindung auf einen Knoten ohne Skript, aber mit Typ, hat keine Datei, in der
sie sucht: Dafür trägt ein Export wie eine Brücke eine **Klasse**
(`Export.Class`, `Bridge.Class`), und gilt sie, zählen nur Angebote derselben
Klasse; ohne Treffer in ihr entsteht keine Kante, geraten wird nicht
(`docs/.superpowers/specs/2026-11-10-graph-bruecken-design.md`,
`docs/.superpowers/specs/2026-11-10-cpp-design.md`). Ein
Extraktor kann Teile einer Datei in einer anderen Sprache als eingebettete
Ergebnisse melden (`Result.Embedded`, etwa Inline-Skripte); sie landen im
Index ihrer eigenen Familie.

Das Web hat vier Brücken-Namensräume. Jede Übergabe ist eine Rohkante mit
dem Namensraum in `RawEdge.Member`, und den Geltungsbereich setzt der
Resolver, nie der Extraktor; so liest jeder Verweis dieselbe `pageRef`.

| Namensraum | Karte | ohne Geltungsbereich | ohne Treffer im Geltungsbereich |
|---|---|---|---|
| `css.class` | alle Treffer | keine Kante | keine Kante |
| `css.id` | alle Treffer | keine Kante | keine Kante |
| `html.id` | genau einer | im Repo, `inferred` | keine Kante |
| `js.function` | genau einer | im Repo, `inferred` | keine Kante |

- **Seite → Regel:** `class="a b"` übergibt je Klasse `css.class` mit der
  Relation `references` von der Datei; Geltungsbereich sind die Stylesheets,
  die die Seite verlinkt, und die Seite selbst (ihr `<style>`).
- **Seite → Funktion:** ein `on*`-Attribut, dessen Wert mit einem einfachen
  Aufruf beginnt (`go(…)`, `return go(…)`), übergibt `js.function` mit der
  Relation `calls`; Geltungsbereich sind die Skripte, die die Seite lädt,
  und die Seite selbst. Angeboten werden die Funktionen auf oberster Ebene
  eines klassischen Skripts (ohne `import`, `export`, `require` und
  `module.exports`): inline vom HTML-Extraktor, als Datei, die eine Seite
  lädt, vom Resolver.
- **Skript → Element:** `document.getElementById("x")` übergibt `html.id`,
  ein `querySelector*` allein aus `#x` und `.a` zusätzlich `css.id` und
  `css.class`; ein JSX-`id` bietet `html.id` an (Ziel ist die Komponente).
  Geltungsbereich sind die Seiten, die das Skript laden (`pagesOf`, aus
  `webOf`); für einen eingebetteten Teil die eigene Seite.
- **Skript → Regel:** `getElementsByClassName`, `classList.*`, ein
  JSX-`className` und die Klassen und IDs eines `querySelector*` übergeben
  `css.class` und `css.id`. Geltungsbereich (`styleScope`) sind die
  Stylesheets, die das Modul importiert (`./`/`../` über `pageRef`, alles
  andere über `paths` und `baseUrl` der `tsconfig.json`, Query wie `?inline`
  abgeschnitten), dann je ladender Seite deren verlinkte Stylesheets und die
  Seite selbst. Ein solcher Import ist zugleich eine `imports`-Kante auf das
  Stylesheet, damit der Blast-Radius eines Stylesheets seine Importeure
  zeigt.

`css.class` und `css.id` raten nie: Eine Regel wie `.btn`, die zwanzig
Komponenten nennen, bekäme sonst bei jedem CSS-Edit zwanzig geratene
Aufrufer. Das bleibt so, weil `blast-audit` geratene Kanten mitzählt
(`blast.New` legt jede Walk-Kante in den In-Grad).

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
  GDScript, C#, C++, Java, PHP, TypeScript (der auch reine JavaScript-Projekte
  prüft), CSS und HTML; die Lane läuft je Lauf einmal, in der Wurzel, getragen vom
  ranghöchsten aktiven Stack mit Graph-Befehl (go, python, gdscript, csharp, cpp, java, php,
  typescript, css, html; ein Stack ohne Rang folgt in Byte-Reihenfolge), und `graph = false` unter einem Stack schaltet sie
  für das ganze Projekt ab. Testdateien sind `_test.go` und für Python
  `test_*.py`, `*_test.py`, `tests.py`, `conftest.py` und alles unter
  `tests/` oder `test/`, für GDScript `*_test.gd` und alles unter `test/`
  oder `tests/`, für C++ ein Dateiname `test_*`, `*_test`, `*_tests`, `*Test`
  oder `*Tests` und alles unter einem Verzeichnis `test`, `tests` oder
  `unittest` (in jeder Schreibung; `Contest.cpp` zählt nicht), für Java alles unter
  `src/test/` und ein Dateiname `*Test`, `*Tests`, `*IT` oder `Test*` (`Contest.java` zählt
  nicht), für PHP ein Dateiname `*Test.php` (`Contest.php` zählt nicht) und alles unter einem
  Verzeichnis `test` oder `tests` in jeder Schreibung, für JavaScript und
  TypeScript `*.test.*`, `*.spec.*` und alles unter einem Verzeichnis
  `__tests__`, `test` oder `tests` (so geschrieben); Szenen und Ressourcen (`.tscn`, `.tres`) zählen nur dort,
  `project.godot` nie. Für die Web-Brücken entscheidet eine bestehende
  Regel: `blast.signal` prüft nur Saaten der Art `function` oder `method`,
  denn nur sie sind Verhalten. Regeln, Elemente und Mixins geben darum das
  Signal `na` und nie einen Befund, auch wenn sechs Seiten mit
  `class="btn"` der Regel `.btn` sechs sichere Kanten geben; eine
  SCSS-`@function` mit genug Aufrufern und ohne Test gibt einen, wie eine
  Go-Funktion ohne Test.
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
  Ressourcen), G5d (C++, samt GDExtension) und G5b (JavaScript, JSX,
  TypeScript und TSX) sind gebaut, ebenso HTML, CSS und SCSS mit den
  Web-Brücken. Der erste
  Plan, Tree-sitter als WebAssembly über `wazero`, ist verworfen.
- **Eingebaute Skripte einer Szene** (`[sub_resource type="GDScript"]`) bleiben
  ohne Kante.
- **Handler in der zweiten `partial`-Datei:** Ein Angebot von C# gilt nur für
  die Klasse, die wie die Datei heißt, und eine Szenen-Verbindung sucht nur in
  dem Skript, das sie nennt; die Methode in der zweiten Datei einer
  `partial class` bleibt darum ohne Kante.
- **Lücken von C++:**
  - Ein Signal, das eine Basisklasse der GDExtension-Klasse bindet, wird von
    der Szenen-Verbindung nicht gefunden; nur die Klasse selbst zählt.
  - Ein Knoten mit Skript und Typ löst eine Verbindung im Skript auf; ein
    Mitglied, das nur die C++-Klasse hat, bleibt ohne Kante.
  - Makros sind nicht expandiert; was eines erst erzeugt, fehlt im Graphen.
  - `if consteval` (C++23) verliert den Rest der Datei in `gotreesitter`; der
    Extraktor liest ihn neu, aber die Funktion mit dem Konstrukt bekommt
    keinen Knoten.
  - Das Neulesen beginnt nur in einer Zeile ab Spalte 0, und ein
    Namespace oder eine Klasse, die am Schnitt offen war, geht verloren: Was
    folgt, nennt keinen, und die schließende Klammer zerbricht die Definition
    davor, die dann keinen Knoten bekommt, je Schnitt eine.
- **Lücken von Java:**
  - Typen und Überladungen kennt der Graph nicht: `obj.m()` ergibt keine
    Kante, bei Überladungen gilt die erste Deklaration.
  - Ob ein Receiver eine Variable ist, entscheidet eine Namensmenge der
    ganzen Datei (Felder, Parameter, lokale Variablen, Enum-Konstanten):
    Heißt irgendeine Variable der Datei `T`, verliert auch `T.m()` seine
    Kante. Die Java-Konvention (Typen groß, Variablen klein) hält das selten.
  - `import module` und eine Anweisung vor `super(…)` (Java 25) liegen jenseits
    der Grammatik von `gotreesitter`; Aufrufe in einem solchen Konstruktor
    fehlen.
- **Lücken von PHP:**
  - Typen von Variablen kennt der Graph nicht: `$obj->m()` und `$obj::m()`
    ergeben keine Kante.
  - `clone($o, [...])` und der Cast `(void)` (PHP 8.5) liegen jenseits der
    Grammatik von `gotreesitter`; je ein Fehler im Rumpf darum herum, der nur
    einen Aufruf im Fehler kostet, der Rest kommt an.
  - Eine Datei mit vielen Heredocs, die PHP-Code enthalten, braucht etwa 20 s
    zum Parsen und kann durch Parsefehler Deklarationen verlieren; der
    Extraktions-Cache des Graphen hält ihr Ergebnis, bis sich die Datei
    ändert, der Graph wartet also einmal. `check rules` hat keinen Cache und
    parst (`internal/verify/rules/edges.go`) jede Datei bei jedem Lauf neu;
    ist `[rules]` gesetzt, kehrt die Wartezeit bei jedem Edit und jedem Commit
    wieder. Ein Parse-Budget fehlt noch.
  - Ohne `composer.json` (und ohne `phpunit.xml`) erkennt die Prüfkette keinen
    Stack `php`, der Graph liest die `.php`-Dateien trotzdem. Die Lanes des
    Stacks brauchen die Werkzeuge in `vendor/bin`; fehlen sie, sind nur
    diese Lanes `unavailable`, nicht rot. Die Lanes von Graph und Regeln laufen
    weiter, und ein Edit führt trotzdem `php -l` aus.
- **Lücken von JavaScript und TypeScript:**
  - Typen kennt der Graph nicht: `obj.m()` auf einer Variablen ergibt keine
    Kante, ebenso ein statisches `C.s()` und `N.f()` in einen
    TypeScript-Namespace, obwohl Klasse und Namespace Knoten haben.
  - Ein JSX-Attribut `class=` in einer `.js`-Datei bricht die
    JavaScript-Grammatik: Die Definition, die es enthält, bekommt keinen
    Knoten, und der Rest der Datei wird nur ab Deklarationen in Spalte 0 neu
    gelesen. `.jsx` liest es mit der tsx-Grammatik.
  - Eine geänderte `tsconfig.json` oder `jsconfig.json` macht den Graphen für
    `graph-fresh` nicht veraltet, wie bei `go.mod`: die Frischeprobe sieht nur
    Quelldateien. Ein Alias greift erst mit dem nächsten Neubau.
  - Paketnamen, Workspace-Pakete und die `extends`-Kette einer
    `tsconfig.json` liest der Resolver nicht, ebenso wenig Subpath-Importe
    (`#types/…`) aus dem Feld `imports` einer `package.json`; vites
    `packages/vite/src/node/index.ts` hat 21 davon. Die Spec nennt nur die
    Aliase aus `tsconfig.json`.
  - Die Lanes des Stacks `typescript` brauchen die Werkzeuge in einem
    `node_modules/.bin` des Bereichs oder darüber bis zur Wurzel, wie `npx`
    sie findet; vor `npm install` und unter Yarn Plug'n'Play sind sie
    `unavailable`, nicht rot. Die Mitglieder, die ein Workspace in der Wurzel
    deklariert, sind eigene Bereiche, auch zwei Ebenen tief; ein Muster mit
    `**` findet nur eine Ebene.
- **Lücken von HTML, CSS und SCSS:**
  - Ein geteiltes Skript auf zwei Seiten, die beide `id="app"` haben,
    erreicht keines der Elemente: `html.id` will genau einen Treffer
    (`cardOne`). Ebenso bleibt `onclick="init()"` ohne Kante, wenn zwei
    Skripte der Seite `init` definieren.
  - Die `id` eines Elements erreicht keine Regel `#id`: `class` übergibt
    `css.class`, aber `css.id` übergibt nur JavaScript.
  - `.sass` und `.less` werden nur gelintet (beim Edit über
    `stylelint {file}`), nicht in den Graphen gelesen; die indentierte
    Sass-Syntax kennt die Grammatik nicht.
  - Ein wurzelrelatives `import "/x.css"` aus einem Skript geht nicht über
    `pageRef`, sondern über die Aliase der `tsconfig.json`; ohne
    `paths`/`baseUrl` ergibt es keine Kante. Die Wurzel `/` eines Bundlers
    ist Projektkonfiguration, die Repo-Wurzel zu raten gäbe falsche Kanten.
  - Eine Seite, die nur gekürzte Ordner verlinkt (`dist/`, `build/`),
    bekommt keine Kante auf ihre Stylesheets und keine für ihre Klassen.
  - Der Klassen-Geltungsbereich einer Seite folgt keinem `@import`/`@use`
    ihrer verlinkten Stylesheets.
  - SCSS: keine Lade-Pfade (`node_modules`, `--load-path`); `-` und `_` in
    Namen gelten als verschieden; `@forward … as`, `show` und `hide` werden
    nicht berücksichtigt; `@use "x"` findet kein `x.css`; zwei Definitionen
    in derselben Tiefe ergeben keine Kante; ein unqualifizierter Aufruf sieht
    auch Mitglieder eines `@use` ohne `as *`, was Sass ablehnen würde;
    `@extend %p, %q` liest nur `%p`, und ein verschachtelter Platzhalter
    (`.a › %p`) ist für `@extend` unerreichbar.
  - Grenzen der Grammatiken: Vorlagen-Tags (`<% %>`, `<?php ?>`) zerlegen
    die Elemente um sie herum (Knoten ja, Span nur der Start-Tag); ein
    einzelnes, nie geschlossenes `<p class>` am Dateiende verliert seine
    Klassen; `import source` in einem Inline-Skript und `url(./x)` ohne
    Anführungszeichen in `<style>` sind Parsefehler im Teil; ein
    großgeschriebenes `@IMPORT` verliert den Import und die Regel danach;
    `@use "x" with (…)` in einem ERROR gibt keinen Import, ein
    Keyword-Argument `f($a: 1)` in einem Wert verliert den Aufruf, und eine
    Map mit nackten Schlüsseln als Argument oder eine Interpolation mit
    Aufruf oder Zeilenumbruch in einem Namen kann eine ganze Datei kosten.
  - Weitere lokale Fehler der SCSS-Grammatik kosten nur, was in ihnen
    steht: ein abschließendes Komma und verschachtelte Maps
    (`defaults(( … ))`) in großen Maps, eine Map nach einem Schlüsselwort
    (`@return ($k: $x)`, `@each … in ($k: 1)`), `@include m using ($a)`,
    `@media ($a: 1)` und eine Map-Zuweisung über mehrere Zeilen. Eine Map
    mit nackten Schlüsseln als Vorgabe eines Mixins kostet den Knoten des
    Mixins, ein Aufruf in einem Map-Wert (`("a": 1, "b": f(2))`) seine
    Kante; eine Interpolation über mehrere Zeilen in einem Wertnamen kann
    einen Müllknoten hinterlassen. Eine verschachtelte Regel, die mit einem
    Kombinator beginnt, heißt je nach Kontext `.a › .b` oder `.a › > .b`,
    und ein verschachtelter privater Platzhalter (`.a { %-p {} }`) gilt als
    exportiert.
  - Inline-Skripte und Klassen: Ein `//`-Kommentar am Ende eines Skripts
    verschluckt ein Skript, das in derselben Zeile folgt; teilen sich ein
    Modul und ein klassisches Skript eine Zeile, werden die Funktionen des
    Moduls als `js.function` angeboten (die Angebote hängen an der
    Startzeile); ein nicht geschlossenes `{{` in einer Klasse übergibt die Tokens danach;
    eine `id`, die zweimal kollidiert, bekommt `~2~2` statt `~3`.
  - Prüfkette: Seiten und Stylesheets eines verschachtelten
    Bereichs lintet der Bereich darüber mit; eine rote html-Lane mit
    mehreren Befehlen druckt jeden Befehl mit seiner ganzen Seitenliste.
  - Was der Extraktor bewusst so liest: `:not(.d)` bietet `d` an; Entities
    in Attributen werden nicht dekodiert; ein `on*`-Wert mit
    Mitgliedsaufruf (`a.b()`) übergibt nichts; Klassen mit
    Vorlagen-Platzhalter (`{{ x }}`) oder `%` (Tailwinds `w-[50%]`)
    entfallen.
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
