# Brücken zwischen Sprachen im Code-Graphen: Design

Stand 2026-11-10. Teilprojekt A von vier (A Brückenkern, B Graph JS/TS/TSX,
C Graph HTML/CSS/SCSS, D Prüfkette und Kompatibilitätsmatrix der Web-Stacks).
B und C bauen auf A auf; D ist unabhängig.

## Anlass und Ziel

Jede Sprache löst heute in einem eigenen Namensindex auf
(`internal/code/resolve/resolve.go`, `GraphWith`). Über eine Sprachgrenze
geht nur ein Godot-Pfad, der eine Datei nennt; er landet auf deren
Dateiknoten. Eine Kante von Symbol zu Symbol über eine Sprachgrenze gibt es
nicht.

Die Web-Stacks brauchen genau das (HTML-Klasse → CSS-Regel, `onclick` →
JS-Funktion, `querySelector` → HTML-Element), Godot später ebenso (Szene →
C#-Methode, GDScript `extends` → C++-Klasse aus einer GDExtension). Ziel von A:
ein sprachneutraler Mechanismus, an den B, C und spätere Extraktoren für C#
und C++ nur noch andocken, ohne Kern, Resolver oder GDScript umzubauen.

## Ziele

1. `extract.Result` trägt drei neue Felder: `Exports`, `Bridges`, `Embedded`.
2. `resolve.GraphWith` löst nach den Kanten der einzelnen Sprachen alle
   Brücken in einem sprachneutralen Schritt auf.
3. Jeder Namensraum hat feste Regeln für Kardinalität, Zielbereich und
   Geltungsbereich (Realm).
4. GDScript nutzt den Mechanismus schon: es meldet seine Klassen und reicht
   ab, was es selbst nicht auflösen kann. Ein Test belegt die Wege Szene →
   C#-Methode, Szene → C#-Signal und GDScript `extends` → C++-Klasse mit
   Ergebnissen, wie ein künftiger Extraktor sie liefern würde.
5. `gotreesitter` auf v0.57.0, die neueste Version (globale Regel: neueste
   Version, exakt gepinnt). Die Grammatiken für B und C hat schon v0.56.0.

## Nicht-Ziele

- Keine Extraktoren für JS/TS, HTML, CSS, SCSS (B, C), C# oder C++ (Roadmap
  „C++ in the code graph“; C# kommt als neue Zeile dazu). In A feuert der
  Weg zu C# und C++ darum nur im Test; im echten Build gibt es noch keine
  Datei, die `godot.*` exportiert.
- Kein Abgleich über Namen quer durch alle Sprachen. Eine Brücke ist immer
  eine feste Paarung über einen Namensraum.
- Keine dynamischen Aufrufe: `node.Call("X")` aus GDScript in C# oder
  `call("x")` auf einer Instanz unbekannten Typs bleiben ohne Kante.
- Kein Erben über eine Sprachgrenze außer GDScript → GDExtension (C++).
  GDScript darf nicht von einem C#-Skript erben und umgekehrt (Godot-Doku
  „Cross-language scripting“, Abschnitt „Inheritance“,
  https://docs.godotengine.org/en/stable/tutorials/scripting/cross_language_scripting.html).
- Aufrufe auf eine Basisklasse, die erst die Brücke findet (`f()` in einem
  Skript, das von einer C++-Klasse erbt), bleiben ohne Kante: GDScript löst
  Aufrufe vor dem Brückenschritt auf.
- Eingebaute Skripte einer Szene (`[sub_resource type="GDScript"]`) liest A
  nicht, und `Embedded` ist dafür **nicht** gedacht: Szene und Skript sind
  beide `gdscript`, ein eingebettetes Teil in der Sprache der Datei ist
  verboten (siehe „Eingebettete Teile“). Sie gehören später ins `Result` der
  Szene selbst. Offen ist dafür, wie der GDScript-Resolver ein zweites
  Skript in einer `.tscn` schlüsselt (seine Maps sind je Pfad, und `Package`
  ist bei Nicht-`.gd`-Dateien die uid). Das ist eine eigene Folgearbeit.
- `graph check` bleibt wie es ist: Es vergleicht Knoten (ID und BodyHash,
  `internal/code/query/check.go`), keine Kanten, also auch keine
  Brückenkanten.
- Keine Option `[graph] web_root` (siehe C).

## Anti-Ziele

- Kein Bruch bestehender Graphen über das hinaus, was der Versionswechsel
  ohnehin auslöst (einmaliger Neubau, siehe „Cache und Frische“).
- Keine Phantomkanten: Ein Name, der in der Zielsprache mehrdeutig ist, gibt
  in einem Namensraum mit Kardinalität „genau eins“ keine Kante. Ein Name
  findet nie eine Klasse eines anderen Godot-Projekts.
- GDScript zu GDScript ändert sich nicht: dieselben Kanten, dieselben
  `Unresolved`-Einträge mit denselben Gründen.
- Keine Sonderfälle je Sprache im Brückenschritt. Er kennt Namensräume, keine
  Sprachen.

## Vom Menschen entschieden

- Weg „allgemein“ statt „nur Web“: ein Mechanismus für alle Paarungen.
- Vorlage graphify (Graphify-Labs/graphify, Stand 2026-11-09), übernommen
  wird das Prinzip, nicht der Code:
  - Ein Verweis löst nur in einem Bereich auf, den er selbst benennt (bei
    graphify: XAML `Click=` nur in der Code-behind-Datei aus `x:Class`).
  - Sprachen mit echtem Zusammenspiel teilen einen Index (graphify:
    Familie `jsts`). Bei uns wird das ein Extraktor für vier Endungen (B),
    keine Familientabelle.
- Kardinalität je Namensraum (Hinweis des Advisors): `css.*` verbindet mit
  allen Treffern, `godot.*` braucht genau einen.
- Eingebettete Sprachen über `Result.Embedded`, nicht über eine Sprache je
  Knoten.
- Godot auf C# und C++ wird jetzt vorbereitet, gebaut wird nur der Kern und
  die GDScript-Seite.

## Bau

### Vertrag in `internal/code/extract`

```go
// Export is a definition a file offers across a language boundary.
type Export struct {
	Namespace Namespace    `json:"namespace"`
	Name      string       `json:"name"`
	Target    model.NodeID `json:"target"`
}

// Bridge is a use that names a definition in another language.
type Bridge struct {
	Source    model.NodeID   `json:"source"`
	Relation  model.Relation `json:"relation"`
	Namespace Namespace      `json:"namespace"`
	Name      string         `json:"name"`
	Scope     []string       `json:"scope,omitempty"` // repo-relative files; empty: the whole realm
	File      string         `json:"file"`
	Line      int            `json:"line,omitempty"`
}
```

`Namespace` ist ein String-Typ; jeder Namensraum ist eine exportierte
Konstante in `extract` (`NamespaceGodotClass = "godot.class"` usw.). Eine
Liste `Namespaces` nennt alle; daran prüft ein Test die Tabelle im
Resolver.

`Result` bekommt `Exports []Export`, `Bridges []Bridge` und
`Embedded []Result`, alle `omitempty`.

- Den Namen eines Exports bestimmt der Extraktor der **Zielseite**, nicht der
  Kern. Ein künftiger C#-Extraktor meldet den Delegate `HitEventHandler` als
  `godot.signal Hit`, ein C++-Extraktor `ClassDB::bind_method(D_METHOD("get_speed"), …)`
  als `godot.method get_speed`. So braucht der Kern keine Namensabbildung je
  Sprache.
- Die Datei eines Exports ist der `Path` des `Result`, das ihn trägt; ein
  eigenes Feld braucht es nicht.

### Eingebettete Teile

`Embedded` ist ein Teil der Datei in einer anderen Sprache (Inline-Skript in
HTML, später ein eingebautes Skript einer Szene). Regeln:

- Ein eingebettetes `Result` hat denselben `Path` wie die Datei, eine eigene
  `Language`, Zeilen bezogen auf die ganze Datei und Knoten-IDs, die der
  einbettende Extraktor eindeutig macht.
- Es trägt **keinen** Dateiknoten (sonst steht der Knoten doppelt im Graphen,
  `model.Validate`). Seine Knoten hängen per `contains` unter einem Knoten der
  Datei.
- Je Datei höchstens **ein** eingebettetes `Result` je Sprache, und nie in
  der Sprache der Datei selbst. Mehrere Inline-Skripte einer HTML-Datei fasst
  der einbettende Extraktor zu einem `Result` zusammen. So bleibt jede
  Sprachgruppe bei einem `Result` je Pfad, und die Indizes der Resolver,
  die nach `f.Path` schlüsseln (`resolve.go`, `python.go`, `gdscript.go`),
  bleiben richtig.
- Keine Verschachtelung über eine Ebene hinaus.
- Der Extraktions-Cache trägt eingebettete Teile im Eintrag ihrer Datei.
  `BodyText` (`json:"-"`) wird beim Schreiben und Lesen auch für die Knoten
  eingebetteter Teile umgelagert (`query/cache.go`), sonst verliert die
  Wortsuche von `ask` deren Text.
- Der Cache-Eintrag wird mit `Version()` der Datei-Sprache gestempelt
  (`query/build.go`). Ein Extraktor, der Teile einbettet, nimmt darum die
  Versionen der eingebetteten Extraktoren in seine eigene `Version()` auf;
  sonst käme nach einer Änderung am eingebetteten Extraktor ein alter Teil
  aus dem Cache. In A bettet niemand ein; die Regel gilt ab C.
- Statistik (`query/build.go`):
  - `LangStats.Files` zählt je Sprache die Dateien mit einem Teil in dieser
    Sprache; eine HTML-Datei mit Inline-Skript zählt also unter `html` und
    unter der Skriptsprache. Die Gesamtzahl der Dateien zählt jede Datei
    einmal.
  - `Parsed` und `Reused` folgen der Datei: wird sie neu geparst, gelten
    alle ihre Teile als geparst.
  - Parse-Fehler eines eingebetteten Teils zählen unter seiner Sprache; in
    `ErrorFiles` steht der Pfad der Datei.
  - `NoSymbol` betrachtet nur die Datei selbst.
- `Meta.Languages` nennt auch Sprachen, die nur eingebettet vorkommen.
  `query/stats.go` und `repomap/map.go` leiten Sprachen aus Dateiendungen ab
  und zeigen eingebettete Sprachen nicht; das bleibt so.

### Brückenschritt in `internal/code/resolve`

`GraphWith` macht aus `files` zuerst eine flache Liste (jedes `Embedded`
neben seine Datei), gruppiert wie bisher nach Sprache und löst die Kanten
jeder Sprache auf. Danach sammelt es alle Exports in einen Index
Namensraum → Realm → Name → Treffer (Datei, Knoten, Sprache) und löst jede
Brücke auf. Die Sprache eines Treffers ist die `Language` des `Result`, das
den Export trägt; bei einem eingebetteten Teil also dessen Sprache, nicht
die der Datei.

Exports und Brücken kommen je aus zwei Quellen:

1. vom Extraktor (`Result.Exports`, `Result.Bridges`), wenn er schon beim
   Lesen weiß, was er anbietet oder dass ein Ziel in einer anderen Sprache
   liegt (künftig C# und C++; C: HTML → CSS);
2. vom Resolver einer Sprache, wenn erst er es entscheiden kann (GDScript,
   siehe unten: nur er kennt die Projekte und weiß, welcher Name in seinem
   Index fehlt). `edgesOf` gibt dafür neben Kanten und `Unresolved` Exports
   und Übergaben zurück. Eine Übergabe ist eine Brücke und, falls der
   Resolver einen `Unresolved`-Eintrag dafür angelegt hat, ein Verweis auf
   diesen Eintrag, der das Zusammenfügen der Listen aller Sprachen übersteht
   (etwa ein Zeiger oder ein Index samt Versatz). Einträge werden erst
   entfernt, wenn alle Übergaben ausgewertet sind.

**Namensraum-Tabelle.** Jeder Namensraum hat einen Eintrag mit vier
Eigenschaften:

| Namensraum | Kardinalität | Realm | ohne Zielbereich | Zielbereich ohne Treffer |
|---|---|---|---|---|
| `godot.class` | genau eins | Godot-Projekt | ganzer Realm, `extracted` (Godots ClassDB gilt je Projekt) | keine Kante |
| `godot.method` | genau eins | Godot-Projekt | keine Kante | keine Kante |
| `godot.signal` | genau eins | Godot-Projekt | keine Kante | keine Kante |

- **Realm** ist der Geltungsbereich eines Namens. Für `godot.*` ist es das
  Verzeichnis der nächsten `project.godot` über der Datei, so wie der
  GDScript-Resolver Klassennamen heute je Projekt hält
  (`resolve/gdscript.go`, `classes` je Wurzel). Die Regel ist asymmetrisch:
  - Eine **Brücke** braucht einen Realm. Eine Datei ohne Projekt darüber
    bekommt in `godot.*` keine Kante.
  - Ein **Export** aus einer Datei ohne Projekt darüber gilt in jedem
    Godot-Realm des Repos. Grund: GDExtension-Quellen liegen meist neben dem
    Projekt, nicht darin (`godot-cpp-template`: `src/*.cpp` neben
    `project/project.godot`). Liegt so ein Name in mehreren Projekten vor,
    macht „genau eins“ daraus Mehrdeutigkeit, keine Phantomkante.

  Die Wurzeln ermittelt
  der Brückenschritt aus den Pfaden aller Dateien des Builds; er kennt dafür
  den Dateinamen `project.godot`, nicht die Sprache GDScript. Ein Realm ist
  ein Paar (Wurzel, vorhanden), kein bloßer String: Ein Projekt im
  Wurzelverzeichnis des Repos hat die Wurzel `""` (`dirOf`,
  `resolve/gomod.go`) und ist trotzdem ein Realm, verschieden von „kein
  Realm“.
- C trägt `css.class`, `css.id`, `html.id` und `js.function` nach (dort:
  Kardinalität „alle“, Realm das ganze Repo, Ausweichen auf den ganzen Realm
  als `inferred`).
- Die Tabelle ist ein Parameter einer internen Funktion; `GraphWith` reicht
  die eingebaute Tabelle hinein, Tests eine eigene. So decken Tests jede
  Kombination aus Kardinalität und Ausweichen mit Testnamensräumen ab, ohne
  dass die eingebaute Tabelle sie kennt.
- Ein Test hält fest: Jede Konstante aus `extract.Namespaces` steht in der
  eingebauten Tabelle.

**Regeln des Schritts:**

- Gesucht wird im Realm der Brücke und unter den Exports ohne Realm. Hat die
  Brücke keinen Realm (keine `project.godot` darüber), ergibt sie keine
  Kante.
- **GDScript beantwortet keine eigenen Übergaben.** Exports aus Dateien der
  Sprache, deren Resolver die Brücke übergeben hat, zählen für diese Brücke
  nicht: Der GDScript-Resolver hat den Namen in seinem Projekt schon gesucht
  und nicht gefunden. Allgemein: Eine Übergabe eines Resolvers sucht nur in
  Exports anderer Sprachen. Brücken aus `Result.Bridges` unterliegen dieser
  Regel nicht.
- Mit Zielbereich zählen nur Exports aus diesen Dateien. Treffer dort ergeben
  `extracted`.
- Ohne Treffer im Zielbereich gilt die letzte Spalte, ohne Zielbereich die
  vorletzte.
- „Genau eins“ mit mehreren Treffern ergibt keine Kante. „Alle“ ergibt eine
  Kante je Treffer.
- Ein Namensraum, der nicht in der Tabelle steht, ergibt keine Kante.
- Eine Brücke auf ihre eigene Quelle ergibt keine Kante.
- **Zusammenfassen:** Brückenkanten werden untereinander nach (Quelle, Ziel,
  Relation) zusammengefasst; die kleinste Zeile bleibt, `extracted` schlägt
  `inferred`. Gibt es für denselben Schlüssel schon eine Kante aus dem
  Resolver einer Sprache, entfällt die Brückenkante.
- **`Unresolved`:** Trägt eine Übergabe einen `Unresolved`-Eintrag, so
  - entfällt der Eintrag, wenn die Brücke eine Kante ergibt;
  - wird sein Grund `ReasonAmbiguous`, wenn die Brücke mehrere Treffer hat;
  - bleibt er unverändert (`ReasonNone`), wenn sie keinen hat.
- Die Reihenfolge ist deterministisch (Sortierung wie bisher), damit ein
  Graph byte-gleich bleibt.

### GDScript-Seite

- **Export:** Der GDScript-**Resolver** meldet `godot.class X` für jede
  Klasse, die er unter einem Projekt registriert (`classes` je Wurzel),
  auf den Klassenknoten. Ein `class_name` in einem Skript ohne
  `project.godot` darüber registriert Godot nicht, also wird er auch nicht
  exportiert; sonst gälte er als Export ohne Realm in jedem Projekt.
  Auflösen innerhalb von GDScript bleibt wie bisher; der Export ist für
  andere Sprachen da (künftig etwa ein C#-Skript, das eine GDScript-Klasse
  per Name nennt).
- **`extends X`:** Übergeben wird genau dann, wenn die Rohkante keinen Pfad
  und keinen Empfänger trägt (`Specifier == ""`, `Receiver == ""`) und
  `base` `ReasonNone` liefert. Dann übergibt der Resolver eine Brücke
  `godot.class X` (Relation `extends`) ohne Zielbereich, mit dem
  `Unresolved`-Eintrag, den er heute schon anlegt. Nicht übergeben werden
  ein gepunkteter Name (`extends A.B`, auch wenn `A` bindet und `B` fehlt:
  Godots ClassDB kennt nur flache Namen) und eine fehlende innere Klasse
  hinter einem Pfad (`extends "res://a.gd".Missing`).
- **`X.new()`:** Die `calls`-Rohkante mit `Receiver == "X"`, `Name == "new"`
  übergibt eine Brücke `godot.class X` (Relation `calls`), wenn `X` nicht
  bindet. Einen `Unresolved`-Eintrag gibt es dafür heute nicht und künftig
  auch nicht. Die zusätzliche `references`-Rohkante auf `X`, die der
  Extraktor bei `.new()` ebenfalls meldet, wird nicht übergeben: Sie läuft
  über die Suche nach Mitgliedern, und eine Übergabe dort träfe jeden Namen,
  der als Wert genommen wird. Eine C++-Klasse bekommt bei `.new()` darum nur
  die `calls`-Kante, eine GDScript-Klasse wie heute beide. Auch eine lokale
  Variable (`scene.new()`) wird so übergeben; sie findet nur dann eine
  Klasse, wenn eine andere Sprache genau diesen Namen exportiert, was bei
  Godots PascalCase-Klassennamen praktisch nicht vorkommt.
- **Engine-Klassen** wie `Node2D` exportiert niemand. `extends Node2D` bleibt
  ohne Kante und mit `ReasonNone` in `Unresolved`; `Node2D.new()` bleibt
  ohne Kante und ohne Eintrag.
- Löst eine `extends`-Brücke auf (GDExtension-Klasse in C++), sieht die
  Ebenenregel `rules.layers.*.bases` (`internal/verify/rules/edges.go`) die
  Basisklasse als aufgelöste Kante, die über `may_use` geprüft wird, nicht
  mehr als unaufgelösten Namen.
- **`[connection]`:** Beide Enden einer Connection sind heute
  `references`-Kanten gleicher Form (Skript oder instanziierte Szene als
  Specifier, `resource.go`); der Resolver sucht das Mitglied als Methode,
  Funktion oder Signal im Wurzelskript. Damit eine Übergabe den richtigen
  Namensraum wählt, markiert der Extraktor jedes Ende: `RawEdge` bekommt ein
  Feld `Member model.Kind` (`"signal"` für das Signal am Quellknoten,
  `"method"` für den Handler am Zielknoten, sonst leer). Der Weg von GDScript
  zu GDScript wertet das Feld nicht aus und bleibt unverändert.
- **Übergabe einer Connection:** Ist das Wurzelskript eine Datei des Builds,
  die nicht zu den Dateien des GDScript-Resolvers gehört (also eine Datei
  einer anderen Sprache; eine Szene oder Ressource ist es nie), übergibt der
  Resolver statt der
  Suche eine Brücke `godot.method` bzw. `godot.signal` mit dem Wurzelskript
  als Zielbereich. Ist das Wurzelskript keine Datei des Builds (heute jede
  `.cs`-Datei, weil kein Extraktor sie liest), bleibt es wie heute ohne
  Kante.
- `version` von GDScript steigt auf `gdscript/6` (`Member`).

### Test für Godot auf C# und C++

Ein Test in `internal/code/resolve` baut `extract.Result`s von Hand, so wie
ein künftiger C#- und ein C++-Extraktor sie liefern würden (Sprache `csharp`
bzw. `cpp`, Exports `godot.class`, `godot.method`, `godot.signal`). Jeder
Fall enthält die nötige `project.godot`, sofern nicht anders gesagt. Belegt
wird:

- Szene mit `Player.cs` und `[connection signal="Hit" … method="on_hit"]`
  vom C#-Knoten auf einen GDScript-Knoten → `references` auf das C#-Signal
  `Hit` und auf die GDScript-Methode `on_hit`, beide `extracted`.
- Szene mit Handler in `Player.cs` → Kante auf die C#-Methode.
- `extends MyNode` in GDScript, `MyNode` von C++ im selben Projekt
  exportiert → `extends`-Kante, kein Eintrag in `Unresolved`.
- C++-Export in `src/`, Projekt in `game/` (Layout wie
  `godot-cpp-template`) → `extends`-Kante, kein Eintrag in `Unresolved`.
- C++-Export in einem anderen Godot-Projekt → keine Kante, `ReasonNone`.
- Derselbe Name als C++-Export ohne Realm und als C++-Export im Projekt →
  keine Kante, `ReasonAmbiguous`.
- Zwei C++-Exports `godot.class MyNode` im Projekt → keine Kante,
  `ReasonAmbiguous`.
- `project.godot` im Wurzelverzeichnis des Repos, C++-Export im selben Repo
  → `extends`-Kante (Realm mit Wurzel `""`).
- `extends Only` in Projekt `b`, `class_name Only` in Projekt `a` → wie
  heute keine Kante, `ReasonNone` (Realm).
- `extends Only` in Projekt `b`, `class_name Only` in `loose/only.gd` ohne
  `project.godot` darüber → wie heute keine Kante, `ReasonNone` (kein Export
  ohne Projekt, und eine Übergabe sucht nur in fremden Sprachen).
- GDScript-Klasse `Only` mit Export im selben Projekt wie ein C#-Skript, das
  sie per Brücke nennt → Kante (der GDScript-Export dient fremden Sprachen).
- `extends "res://a.gd".Missing` mit C++-Export `Missing` → keine Kante,
  `ReasonNone` (keine Übergabe hinter einem Pfad).
- `MyNode.new()` mit C++-Export `MyNode` → `calls`-Kante, kein
  `Unresolved`-Eintrag.
- `extends Node2D` → keine Kante, `ReasonNone`.
- Skript ohne `project.godot` darüber mit `extends MyNode` und C++-Export →
  keine Kante, Eintrag bleibt `ReasonNone`.

Einen Extraktor nur für Tests in `extract/all` gibt es nicht; die
Sprachliste dort bleibt fest.

### Cache und Frische

- `gotreesitter` v0.57.0 ändert `treesitter.Parser` und damit die Version
  von GDScript und Python; `gdscript/6` kommt hinzu. Jeder bestehende Graph
  gilt danach einmal als fremd und wird neu gebaut. Das steht in den
  Release-Notes.
- Mit der Version ziehen nach: `internal/notices/NOTICE.md` (über
  `loomux dev notices`), die Tests, die `gdscript/5` festhalten
  (`extract/all/all_test.go`, `extract/gdscript/gdscript_test.go`).
- `cacheVersion` (`query/cache.go`) bleibt: die neuen Felder sind
  `omitempty` und lesen sich in alten Einträgen leer.

### Ebenenregeln

Brückenkanten sind gewöhnliche aufgelöste Kanten und zählen für
`[rules.layers]` wie jede andere. In A entstehen sie im echten Build noch
nicht (siehe Nicht-Ziele); die Release-Notes von B und C nennen die
möglichen neuen Verstöße.

## Abnahme

- `go test ./...` grün, Coverage der geänderten Pakete 100 %.
- Die Tests aus „Test für Godot auf C# und C++“ und je ein Test für jede
  Regel des Brückenschritts, jede Kombination aus Kardinalität und
  Ausweichen und jede Regel zu eingebetteten Teilen (Cache mit `BodyText`,
  Statistik, kein Dateiknoten).
- Graph eines Godot-Repos (Testprojekt der GDScript-Abnahme) vor und nach A:
  gleiche Knoten, Kanten und `Unresolved`, nur die Versionsstempel
  unterscheiden sich.
- Graph dieses Repos vor und nach A (gotreesitter v0.57.0): gleiche Knoten
  und Kanten. Weicht etwas ab, wird es vor dem Merge geklärt.

## Wiki und README

Im selben PR nachziehen:

- `docs/wiki/topics/code-graph.md`: Absatz „Jede Sprache löst in einem eigenen
  Namensindex auf …“ um die Brücken erweitern; Abschnitt „Was offen ist“ um
  C# ergänzen.
- Roadmap in `README.md` und `README.de.md`: neue Zeile „C# in the code
  graph (Godot)“ bzw. „C# im Code-Graphen (Godot)“ unter „Coming“, abhängig von A; die Zeile „C++ in the code graph“
  nennt die Brücken als Grundlage. Die Priorität der neuen Zeile entscheidet
  der Mensch im PR. Die Zeile „TypeScript/TSX in the code graph“ ändert erst B.
