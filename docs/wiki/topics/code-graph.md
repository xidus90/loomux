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
    content_hash: "sha256:70612a3be33cedafe25245fa8d4d5ac4e50bbfd995e2808bf21b58321b285865"
    revision: 10
  - id: cli-referenz
    resource: brain://project/loomux/docs/de/cli-reference.md
    doc_id: 01M47W91R5FRSPN6BE643656ZE
    content_hash: "sha256:14491f4bc5a190adce3aca779b1c25bdb98f5e18e108775eb8cb46c2c76999e4"
    revision: 58
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
Gelesen wird nur, was git nicht ignoriert, nach denselben Regeln wie der
Suchkorpus unten (`internal/code/sourceset`); eine getrackte Datei bleibt,
außerhalb eines git-Repositorys gilt keine Regel. Die Regeln kommen aus dem
Record des Suchkorpus, solange dessen Ignore-Quellen und `.git/config` stehen,
sonst von git: jeder git-Prozess kostet unter Windows etwa 70 ms, und die
Sonde läuft vor jeder Frage an den Graphen (`docs/de/benchmarks.md`,
2026-11-17).
Bis 2026-11-17 las der Graph auch Ignoriertes, etwa die HTML-Berichte unter
einem ignorierten `game/reports/` eines Godot-Projekts.

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
  Ein Aufruf, ein Signal oder ein `await` auf einem Empfänger, dessen Klasse
  die Datei nennt, landet bei dieser Klasse und ihren Basen: ein Feld, ein
  Parameter oder ein lokaler Name mit Typannotation, `:=` aus `T.new()`,
  `x as T` oder einer Funktion derselben Klasse mit deklariertem
  Rückgabetyp, und `(x as T).f()`. Der Extraktor setzt dafür die Klasse an
  die Stelle des Namens; ein lokaler Name verdeckt ein Feld, `self.x` meint
  immer das Feld. Ein Empfänger ohne bekannte Klasse bleibt ohne Kante, wie
  in Python: kein Raten über den Methodennamen.
  Die Rückrufe, die die Engine auf jeder Instanz aufruft (`_init`,
  `_notification`, `_enter_tree`, `_ready`, `_exit_tree`, `_process`,
  `_physics_process`, `_input`, `_shortcut_input`, `_unhandled_input`,
  `_unhandled_key_input`, `_gui_input`, `_draw`), nennt keine Zeile des
  Projekts. `T.new()` ruft darum die auf, die `T` definiert oder erbt, und
  eine Szene referenziert die des Skripts, das einer ihrer Knoten ausführt.
  Ein Test, der die Klasse erzeugt oder die Szene lädt, erreicht so eine
  Änderung an `_ready`; ohne diese Kanten blieb `_ready` ohne Test, und
  `blast-audit` meldete die Klasse mit ihrem Eingangsgrad, sobald dieselbe
  Änderung eine Konstante der Klasse traf.
  Was die Grammatik an gültigem Godot ablehnt, schreibt der Extraktor vor dem
  Parsen in einer Kopie gleicher Länge um, sodass Spans und Hashes auf die
  Datei zeigen: In GDScript wird eine reine Kommentarzeile geleert (ihre
  Einrückung hielt die Grammatik für die des Blocks) und `else: if` zu
  `elif`; in Szenen, Ressourcen und `project.godot` wird ein Schlüssel in
  Anführungszeichen oder mit `.` oder `-` (`run/main_scene.web`) zu einem mit
  `_`. Offen bleibt `remote` als Name.

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

- **Rust** liest er auf demselben Kern (`internal/code/extract/rust`, Endung
  `.rs`): Dateien, Funktionen, Structs, Enums, Unions, Traits, Typaliase und
  `macro_rules!`-Definitionen auf Modulebene sowie die Methoden eines `impl`
  oder Traits. Ein `impl` hat keinen Knoten; seine Methoden gehören über
  `Node.Owner` dem Typ, den es nennt, und hängen nur dann an dessen Knoten,
  wenn der Typ in derselben Datei steht, denn er kann in jeder Datei der Crate
  deklariert sein. Ein Inline-Modul ist kein Knoten; sein Pfad qualifiziert die
  Kennungen darin (`rel#tests::adds`), ein Modul in einer eigenen Datei ist
  diese Datei. Tests stehen in Rust in der Datei des getesteten Codes und
  tragen darum die Marke am Knoten (siehe „Tests am Knoten“): `#[test]`, jedes
  Attribut, dessen Pfad auf `test` endet, und alles unter `#[cfg(test)]`. Die
  Registrierungen von godot-rust (gdext) liest er als Text aus den Attributen
  vor einem Item: ein Struct mit `derive(GodotClass)` wird als Godot-Klasse
  angeboten, `#[func]`, `#[rpc]` und `#[signal]` in einem `#[godot_api]`-`impl`
  hinterlassen eine rohe `references`-Kante, auch mit umbenanntem Namen. `use`,
  `mod x;`, Aufrufe, Makroaufrufe, `impl Trait for` und Supertraits liest der
  Extraktor als Rohkanten. Der Resolver (`internal/code/resolve/rust.go`)
  baut je Crate-Wurzel den Modulbaum: `mod x;` führt auf `x.rs` oder
  `x/mod.rs` im Verzeichnis des deklarierenden Moduls (aus `a.rs` nach `a/`,
  aus einem Inline-Modul in dessen Verzeichnis), und `#[path]` geht vor.
  Innerhalb eines Pakets zieht die Bibliothek (`[lib] path`, sonst
  `src/lib.rs`) zuerst, die Pakete in der Reihenfolge ihrer `Cargo.toml`;
  eine Datei, die zwei Bäume erreichen, gehört dem ersten. Die Wurzeln, die
  Cargo nach ihrem Ort findet (`src/bin/`, `tests/`, `benches/`,
  `examples/`), nimmt er aus den Dateien selbst, denn `ModulesFrom` sieht
  kein Verzeichnis. Eine Datei, die kein Baum erreicht, ist eine eigene
  Wurzel: `self::` meint sie selbst, `super::` und `crate::` führen ins
  Leere, und ihre `mod x;` sucht sie, wo ihr Name sie hinlegt (aus `a.rs` in
  `a/`, aus `mod.rs`, `lib.rs` und `main.rs` daneben), denn meist hat sie ein
  Makro deklariert. Ein Pfad beginnt mit `crate::`, `self::`, `super::`,
  `::name` oder einem Namen. Gelesen wird er gegen Modultabellen, die der
  Resolver einmal vor allen Aufrufen festlegt: je Modul und Namensraum (Typen
  und Module, Werte, Makros), von innen und von außen gesehen, Name auf
  Kandidaten, in dieser Rangfolge: Definition im selben Modul, `use` (mit
  Alias), Glob-`use`. Welche Stufe einen Namen beantwortet, steht im
  Quelltext: Ein `use` bindet für die ganze Datei, wo immer es steht, und
  verdeckt die Globs für seinen Namen auch dann, wenn es nichts im Repository
  nennt (`use std::fmt::Display`). Sein eigener Pfad liest an ihm vorbei, und
  nennt er den eigenen Namen der Datei zurück, auch an allen anderen, die
  das tun: Mehrere `mod tests { use super::X; }` in einer Datei nennen so das
  X der Globs. Der
  Crate-Name (eine Pfadabhängigkeit oder die eigene Bibliothek) zählt als
  erstes Segment, wenn weder eine Definition noch ein `use` den Namen
  bindet; bietet ihn auch ein Glob an, ist er mehrdeutig. Die Tabellen sind
  der kleinste Fixpunkt über Kandidatenmengen: Ein Pfadsegment steht für
  alle seine Kandidaten, ein Glob öffnet jedes Modul, das sein Pfad nennt,
  und ab leeren Tabellen fügt jeder Durchlauf nur hinzu, bis einer nichts
  mehr ändert. Das endet in Polynomialzeit, auch in einem dichten Kreis von
  Globs, und hängt weder von der Reihenfolge der Dateien noch von den Namen
  der Module ab. Über Mehrdeutigkeit entscheidet erst der fertige Fixpunkt:
  Eine Kante gibt es nur für genau einen Kandidaten, zu dem kein Segment mit
  mehr als einem Kandidaten führt, und ein Glob durch ein solches Segment
  macht jeden Namen mehrdeutig, den er anbietet. Ein privates Item
  zählt nur im eigenen Modul und darunter. `Typ::f()` trifft die Methode `f`
  eines `impl Typ` irgendwo in der Crate, `self.f()` und `Self::f()` die des
  eigenen `impl`-Typs, sonst den Default einer Trait, die er implementiert,
  oder ihrer Supertraits, Stufe für Stufe; der Methodenindex je Crate geht
  über `Owner`, und zwei Typen gleichen Namens in einer Crate oder zwei
  Methoden gleichen Namens eines Typs (`fmt` aus `Display` und `Debug`)
  geben keine Kante statt einer falschen. Ein Tupel-Struct (`Pt(1)`) und
  eine Enum-Variante (`Kind::A(1)`) treffen den Typ. Ein `impl Trait for
  Typ` startet am Typ, auch wenn er in einer anderen Datei steht, und
  `implements` und `extends` stehen ohne Ziel (eine Trait der
  Standardbibliothek) unter `Unresolved`. `mod x;` und ein `use` ergeben
  `imports` auf die Datei des Moduls oder der Definition, das `use` ohne
  Zeile, weil `extract.Import` keine trägt. Was über Pfad, `use` oder
  `impl`-Typ aufgelöst ist, ist `extracted`; `inferred` sind `x.f()` (die
  einzige Methode `f` der Crate), ein Makro, das das Modul nicht sieht (das
  einzige `macro_rules!` dieses Namens), und jeder Aufruf aus den Argumenten
  eines Makros. Probelauf am 2026-11-15 über alle `.rs`-Dateien des
  Checkouts (`graph build` lässt in gdext einige aus): ripgrep löst 29,6 %
  der rohen Aufrufe auf (6661 von 22 473), davon 49,3 % `inferred`; tokio
  12,8 % (7610 von 59 230), davon 37,2 %; gdext 24,4 % (7530 von 30 800),
  davon 32,2 %. Ein großer Teil der rohen Aufrufe zielt auf die
  Standardbibliothek und fremde Crates, und tokio deklariert die meisten
  Module in `cfg_*! { mod x; }`, die der Extraktor nicht sieht. `graph build`
  selbst deckt sich mit dem Probelauf an den aufgelösten Aufrufkanten, außer in
  gdext: ripgrep 6661 (3287 `inferred`, 49,3 %), tokio 7610 (2833, 37,2 %),
  gdext 7112 (2212, 31,1 %) statt 7530 (32,2 %) des Probelaufs; es liest 363
  der 384 `.rs`-Dateien, sieben Parse-Fehler in `virtual_definitions.rs`. Zeit
  eines Neubaus ohne Cache: ripgrep 1,2 s (55 500 Zeilen), tokio 3,4 s
  (191 000), gdext 1,9 s (117 000): 48 000 bis 63 000 Zeilen je Sekunde für
  Rust. Zum Vergleich braucht dieses Repo mit allen seinen Sprachen 3,4 s für
  298 000 Zeilen, etwa 86 500 je Sekunde; Rust liegt im Rahmen. In den `demo-projects` von godot-rust (Szenen ohne
  GDScript) führt `graph callers` von `Hud` auf `Hud.tscn`, in gdext von
  `ArrayTest` auf `InheritTests.gd` (`extends ArrayTest`). Eine
  Definition, die der Parser um verworfenen Text herum neu aufgebaut hat,
  bekommt keinen Knoten. `corpus.pruneDirs` kennt `target` schon
  (`docs/.superpowers/specs/2026-11-14-rust-design.md`). Der Build reicht
  jede `Cargo.toml` wie `go.mod` als `resolve.Module` herein
  (`Module.Cargo`): Paketname, Name der Bibliothek (`[lib] name`, sonst der
  Paketname mit `_`) und ihre Wurzel (`[lib] path`, sonst `src/lib.rs`),
  Crate-Wurzeln (`src/lib.rs`, `src/main.rs`, `src/bin/`, `tests/`,
  `benches/`, `examples/` und jeder `path`) und Pfadabhängigkeiten, auch über
  `workspace = true`. Die Workspace-Mitglieder liest er nicht: Jedes
  Mitglied ist eine eigene `Cargo.toml` und damit ein eigenes Modul. Was nur
  das Dateisystem entscheidet (Globs der Crate-Wurzeln), liest
  `resolve.Modules`; `ModulesFrom` kennt nur den Text. Eine `Cargo.toml` mit
  Syntaxfehler gibt kein Modul und keinen Fehler.

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
Rust (godot-rust) teilt es ebenso auf: Der Extraktor bietet jedes Struct mit
`derive(GodotClass)` als `godot.class` an, unter `#[class(rename = X)]` oder
seinem eigenen Namen, und der Resolver jede `#[func]`/`#[rpc]` als
`godot.method` (unter `rename =`, sonst dem Methodennamen) und jedes
`#[signal]` als `godot.signal`, beide auf den Methodenknoten. Die Klasse ist
der Godot-Name des `impl`-Typs: Der Resolver sucht den einen Typ dieses
Namens in der Crate (Struct, Enum, Union, Trait oder Typalias; zwei davon
ergeben kein Angebot), auch wenn er in einer anderen Datei steht als das
`#[godot_api] impl`, und nimmt dessen `godot.class`; ohne Godot-Klasse
entsteht kein Angebot. Eine Crate neben dem Projekt (`rust/` neben `godot/`)
gilt in jedem (`docs/.superpowers/specs/2026-11-14-rust-design.md`).
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

**Tests am Knoten:** Ein Test muss nicht in einer Testdatei liegen: Rust
schreibt sie meist als Modul in die Produktivdatei. Ein Knoten trägt darum
`test` (`model.Node.Test`; im JSON nur, wenn gesetzt, sodass bestehende Graphen
byte-gleich bleiben), und `blast.IsTestNode` fragt Knotenmarke und Pfadregel
zusammen. Blast-Signal, `blast-audit` (Aufrufer aus Tests zählen nicht), die
Auflösung von Namen und das Ranking von `graph ask` urteilen über den Knoten;
wo nur ein Pfad vorliegt (die Namespace-Zuordnung von C# und Java), bleibt es
bei `blast.IsTestPath`. Beim Signal zählt ein Test aus einer Testdatei als
geändert, wenn die Datei im Diff liegt; ein markierter Test in einer
Produktivdatei nur, wenn ein Hunk seinen Span schneidet, sonst gälte er bei
jeder Änderung der Datei als mitgeändert. Ein Pfad ohne Hunks (zurückgehalten
oder ohne Hunk gelistet) zählt ganz, ein markierter Knoten als Saat gibt `na`.

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

**Die Graphseite.** Die Weboberfläche zeichnet den Code-Graphen unter
`/graph/<Projekt>` in 2D, mit force-graph im Binary. Der Server fasst ihn zu
Verzeichnissen zusammen (zu Beginn bis 150 Knoten, nie mehr als 1000, größer
nach der Summe des PageRank ihrer Mitglieder); ein Doppelklick öffnet ein
Verzeichnis oder eine Datei. Neben dem Code stehen die Dokumente des
Wiki-Bereichs, und eine Kante geht von einem Dokument zu dem Code, den es
nennt: `reindex` liest dafür Backtick-Pfade und Markdown-Links der Dokumente
und schreibt die, deren Pfad es gibt, als `code_refs` in `graph.json`. Auf ein
Symbol löst erst die Seite auf, gegen den heutigen Graphen; eine veraltete
Zeilenangabe zeigt darum ohne Warnung auf ein anderes Symbol. Regel und Grenzen
stehen in `docs/de/cli-reference.md` unter `loomux ui` und `loomux reindex`.

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
  prüft), CSS, HTML und Rust; die Lane läuft je Lauf einmal, in der Wurzel, getragen vom
  ranghöchsten aktiven Stack mit Graph-Befehl (go, python, gdscript, csharp, cpp, java, php,
  typescript, css, html, rust; ein Stack ohne Rang folgt in Byte-Reihenfolge), und `graph = false` unter einem Stack schaltet sie
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
  `__tests__`, `test` oder `tests` (so geschrieben), für Rust alles unter
  einem Verzeichnis `tests` oder `benches` und eine Datei `tests.rs` oder
  `test.rs` (`src/tests_util.rs` zählt nicht); Szenen und Ressourcen
  (`.tscn`, `.tres`) zählen nur dort,
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
- **Lücken von Rust:**
  - Makros sind nicht expandiert: Was eines erzeugt, fehlt, auch ein
    `mod x;` in einem Makro (tokios `cfg_*!`), dessen Datei dann eine eigene
    Wurzel ist. Aufrufe in den Argumenten eines Makros sind geraten
    (`inferred`).
  - Kein Trait-Dispatch, keine Typinferenz: `x.f()` trifft nur die einzige
    Methode `f` der Crate (`inferred`), auch wenn `x` ein Typ der
    Standardbibliothek ist, und mehrere ergeben keine Kante.
  - Ein `use` in einem Inline-Modul bindet für die ganze Datei, und ein
    `pub use` darin bietet seinen Namen unter dem Dateimodul an, nicht unter dem
    Inline-Modul. `use`-Importe tragen keine Zeile. Das gilt auch für ein `use`
    im Rumpf einer Funktion: Es bindet für die ganze Datei. Mit
    `use crate::prelude::*;` auf Dateiebene ergibt `fn a() { use
    crate::other::spawn; spawn(); } fn b() { spawn(); }` für `b` die Kante
    `other::spawn` (`extracted`), wo Rust `prelude::spawn` auflöst.
  - Nur ein `use`, dessen Pfad aus Schlüsselwörtern und Kindmodulen auf die
    eigene Datei führt, liest an den anderen solchen vorbei. Erreicht ein
    zweites `use` die Datei über einen Alias (`use crate as me; use
    me::f;`), lesen beide einander und bleiben leer, und der Glob-Name
    darunter geht verloren: keine Kante statt einer falschen.
  - Eine Datei, die zwei Bäume erreichen (Bibliothek und Binary deklarieren
    beide `mod x;`), löst ihre Pfade nur im ersten auf.
  - Ein Signal oder eine Methode aus einer Basisklasse wird von einer
    Szenen-Verbindung nicht gefunden (wie bei C++).
  - `#[var]`- und `#[export]`-Eigenschaften ergeben keine Kante.
  - Methoden in `#[godot_api] impl INode for X` (`ready`, `process`) ruft die
    Engine; sie haben im Graphen keinen Aufrufer.
  - `[target.*.dependencies]` liest der Cargo-Leser nicht.
- **Lücken von JavaScript und TypeScript:**
  - Typen kennt der Graph nicht: `obj.m()` auf einer Variablen ergibt keine
    Kante, ebenso ein statisches `C.s()` und `N.f()` in einen
    TypeScript-Namespace, obwohl Klasse und Namespace Knoten haben.
  - Ein JSX-Attribut `class=` in einer `.js`-Datei bricht die
    JavaScript-Grammatik: Die Definition, die es enthält, bekommt keinen
    Knoten, und der Rest der Datei wird nur ab Deklarationen in Spalte 0 neu
    gelesen. `.jsx` liest es mit der tsx-Grammatik.
  - Eine geänderte `tsconfig.json`, `jsconfig.json` oder `Cargo.toml` macht den
    Graphen für `graph-fresh` nicht veraltet, wie bei `go.mod`: die
    Frischeprobe sieht nur Quelldateien. Ein Alias greift erst mit dem
    nächsten Neubau.
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
  `graph viz` ordnete die Spec G4 und G5 zu. Die Brücke steht als `code_refs`
  und wird auf der Graphseite der Weboberfläche gezeichnet; der Befehl
  `graph viz` ist nicht gebaut. Offen bleiben die 3D-Ansicht und die
  Blast-Overlays (die Roadmap im README führt beide unter dem
  Graph-Visualizer) und die Abfrage über MCP, welche Wikiseiten einen Code
  beschreiben.

Quelle: die Architektur und die CLI-Referenz der Nutzerdoku; offene Stufen
führt die Roadmap im README.
