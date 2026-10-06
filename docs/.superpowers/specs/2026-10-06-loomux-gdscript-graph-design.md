# GDScript im Code-Graphen (G5c) — Design

Stand 2026-10-06. Roadmap-Zeile „GDScript in the code graph“ (G5c), vorgezogen
vor G5b (TypeScript): Die Abhängigkeit war eine Reihenfolge, keine technische.

## Ziel

`loomux graph build` liefert für ein Godot-4-Projekt einen brauchbaren Graphen.
Heute ergibt es in `space` 0 Dateien, 0 Knoten, 0 Kanten, weil es nur Go und
Python gibt. Erfolg heißt: In `space` zeigt `graph stats` die `.gd`-Dateien unter
`godot/` mit Klassen, Funktionen, Signalen und aufgelösten Kanten (extends,
preload/load, Verweise auf `class_name` und Autoloads, Aufrufe), die Szenen
hängen an ihren Skripten, und `project.godot` hängt an Autoloads und Hauptszene.

## Leitregel: Projektdateien sind die Quelle

Hat eine Sprache Projekt- oder Konfigurationsdateien, liest der Extraktor sie
als Pflichtquelle. Ihre Regeln werden nicht nachgebaut und nicht geraten. Für
Godot ist das `project.godot`: Es legt die `res://`-Wurzel fest und nennt die
Autoloads. Fehlt es über einer Datei, gibt es keine `res://`-Kante.

Die `class_name`-Tabelle baut der Resolver aus den `class_name`-Anweisungen
selbst. Godots eigene Tabelle (`.godot/global_script_class_cache.cfg`) ist ein
erzeugter, meist nicht versionierter Cache, abgeleitet aus genau diesen
Anweisungen; gelesen wird also die Quelle.

## Entscheidungen

1. **Eine Sprache `gdscript` mit vier Endungen** (`.gd`, `.godot`, `.tres`,
   `.tscn`). `resolve.Graph` löst je Sprache getrennt auf (`edgesOf`). Eine
   Szene→Skript-Kante entsteht nur, wenn beide in derselben Gruppe liegen. Zwei
   Sprachen mit einem Familienbegriff im Resolver wurden verworfen: Umbau für
   einen Fall, und bei einer späteren zweiten Godot-Sprache (C#, C++) würden
   sonst die Auflösungsregeln mehrerer Sprachen über einen Topf laufen.
2. **`project.godot` ist eine Datei der Gruppe.** Ein anderes `*.godot` bekommt
   nur seinen Dateiknoten.
3. **Ein `res://`-Ziel darf eine Datei jeder Sprache sein.** `resolveGDScript`
   bekommt die Pfadmenge des ganzen Builds. Damit trifft eine Szene→`.cs`- oder
   →`.gdextension`-Kante den Dateiknoten, sobald es einen Extraktor dafür gibt.
   Ziele, die keine Datei des Graphen sind (Texturen, Audio), ergeben keine
   Kante.
4. **Klassennamen über Sprachgrenzen** (C++ `GDCLASS`, C# `[GlobalClass]`)
   kommen erst mit der zweiten Godot-Sprache: als zweiter Durchgang in
   `resolve.Graph` mit gemeinsamer Namenstabelle.
5. **Knotenmodell:** Ein Skript mit `class_name` bekommt einen `class`-Knoten;
   ein Skript ohne `class_name` keinen.
6. **`godot/addons/` wird mitgenommen**, ohne Sonderfall und ohne Konfiguration.
   Godot behandelt addons als Teil des Projekts, ihre `class_name` sind global.
7. **gotreesitter v0.55.0 → v0.55.1** als eigener erster Commit
   `build(deps)`. Die Grammatik-Blobs von `gdscript`, `godot_resource` und
   `python` sind in beiden Versionen gleich (`BlobSHA256` verglichen); geändert
   hat sich die Laufzeit (Python-Stack-Cap, tiefe Konfliktgabeln).
   `treesitter.Parser` wird `gotreesitter/v0.55.1`.

## Extraktor `internal/code/extract/gdscript`

`Language{}` mit `Name()` = `gdscript`, `Version()` =
`gdscript/1@` + `treesitter.Parser`, `Extensions()` = `.gd`, `.godot`, `.tres`,
`.tscn`. `File` verteilt nach `path.Ext` auf die Grammatik `gdscript` (`.gd`)
oder `godot_resource` (der Rest).

### `.gd`: Knoten

- Dateiknoten wie bei jeder Sprache.
- `class_name X` → `class`-Knoten `X` über die ganze Datei. Signatur: die
  `class_name_statement`-Zeile; sie kann `extends` tragen.
- Innere Klassen in jeder Tiefe → `class`, qualifiziert `X.Inner.Deeper`; ohne
  `class_name` `Inner.Deeper`. Owner ist die umgebende Klasse.
- `func` und `static func` → `method`, Owner = umgebende Klasse. Auf Skriptebene
  eines Skripts ohne `class_name` → `function` an der Datei.
- `signal` → Knotenart `signal`, Owner wie bei Funktionen.
- `Exported`: Der Name beginnt nicht mit `_`.
- Keine Knoten: `var`, `const`, `enum`. Ihr Text bleibt im Suchtext von Klasse
  oder Datei.
- Eine Definition, die der Parser um verworfenen Text herum gebaut hat (ERROR im
  Kopf), bekommt keinen Knoten, und ihre Aufrufe zählen nicht. Die Regel ist die
  von `python.header`.
- `contains`: Datei → Klasse → Methoden, Signale, innere Klassen. Ohne
  `class_name` hängen sie direkt an der Datei.

### `.gd`: Rohkanten

- **extends**, Quelle = Klasse, sonst Datei:
  - `extends Foo` / `extends Foo.Inner`: Name und Receiver;
  - `extends "res://x.gd"` oder ein relativer Pfad: Specifier.
  - Quellen: `extends_statement` auf Skriptebene, Feld `extends` in
    `class_name_statement` und `class_definition`.
- **imports**:
  - Jedes `preload("…")` und `load("…")` mit Zeichenkettenliteral wird zur
    Kante Datei → Specifier.
  - `const X = preload/load(…)` und `var X = preload/load(…)` auf Skriptebene
    ergeben zusätzlich `Import{Alias: X, Path: …}`.
  - Nicht: `ResourceLoader.load(…)` und Aufrufe ohne Literal.
- **calls**, Quelle = innerste Definition mit Knoten, sonst Klasse oder Datei:
  - `foo()` und `self.foo()`: Name allein (bei `self` mit Owner = eigene Klasse,
    wo es eine gibt);
  - `X.foo()` und `X.Y.foo()`: Name mit Receiver-Kette aus Bezeichnern
    (`attribute` mit `identifier`-Kindern, letztes Kind `attribute_call`);
  - `super.foo()` und `super()`: Receiver `super`;
  - sonst keine Kante (`get_node("x").q()`, `a[0].f()`).
- **references**:
  - Jeder Bezeichner in einer Definition, auch in Typannotationen und bei
    `is`/`as`, je Quelle und Name einmal.
  - Gilt nicht für Definitionsnamen (Knotentyp `name`).
  - Der Resolver behält nur Treffer. Das ist sicher, weil Godot einen lokalen
    Namen, der eine globale Klasse überdeckt, als `SHADOWED_GLOBAL_IDENTIFIER`
    meldet.
- **Signale**:
  - `sig.emit()`, `sig.connect(…)` und `sig.disconnect(…)` laufen als calls mit
    Receiver `sig` zum Resolver;
  - `emit_signal("sig")` wird zur Rohkante references mit dem Namen aus dem
    Literal.

### `.tscn`, `.tres`

- Dateiknoten. Der Suchtext ist wie überall auf `MaxBodyChars` gekappt.
- Jede Sektion `ext_resource` mit Attribut `path` wird zur Kante imports Datei
  → Specifier.
- Ein `ext_resource` nur mit `uid` ergibt keine Kante.

### `project.godot`

- Dateiknoten.
- Sektion `[autoload]`: Jede Property `Name="*res://…"` wird zu
  `Import{Alias: Name, Path: res://…}` (führendes `*` entfernt) plus einer Kante
  imports.
- `[application]` `run/main_scene` wird zur Kante imports.

### Parsefehler

`ParseErrors` wie bei Python: ERROR- und MISSING-Knoten. Eine Datei mit Fehlern
bleibt im Graphen, `graph build` scheitert nicht daran.

## Resolver `resolveGDScript`

Ein Arm in `edgesOf` neben `resolvePython`. Jede Kante braucht genau ein Ziel,
sonst entfällt sie. Konfidenz immer `extracted`, kein Rückfall auf einen
geratenen eindeutigen Namen.

### Projekte und Pfade

- Jedes `project.godot` der Gruppe bildet ein Projekt; die Wurzel ist sein
  Ordner. Eine Datei gehört zum nächsten Vorfahren-`project.godot`.
- `res://a/b` → `<wurzel>/a/b`. Ein Pfad ohne Schema ist relativ zum Ordner der
  Datei.
- Das Ziel muss in der Pfadmenge des ganzen Builds stehen (Entscheidung 3).
- Ohne Projekt keine `res://`-Kante und keine Autoloads. Relative Pfade gehen
  weiter.

### Klassen

- Jedes Skript ist eine Klasse. Ihr Knoten ist der `class`-Knoten bei
  `class_name`, sonst der Dateiknoten.
- Ihre Basis ist das Ziel ihrer `extends`-Kante. Die Basiskette wird hinauf
  gelaufen; jeder Knoten wird einmal besucht, ein Zyklus endet.

### Bindung eines Namens `X`, in dieser Reihenfolge

1. innere Klasse der eigenen Klasse oder einer umgebenden;
2. Skript-Alias (`Import` mit Alias) der Datei;
3. `class_name` des Projekts;
4. Autoload des Projekts.

Zwei gleiche `class_name` in einem Projekt ergeben keine Bindung.

### Kanten

- **contains**: wie vom Extraktor.
- **imports**: Specifier nach „Projekte und Pfade“; je Datei und Ziel einmal,
  nie auf sich selbst.
- **extends**: Name und Receiver nach der Bindung, Specifier als Pfad. Ist das
  Ziel ein Skript, ist es dessen Klassenknoten. Eine Engine-Basis (`Node`)
  ergibt keine Kante.
- **calls**:
  - Name allein, `self.`: eigene Klasse, dann ihre Basiskette.
  - `super`: ab der Basis. `super()` sucht den Namen der rufenden Funktion.
  - Receiver ist ein Signal der eigenen Klasse oder einer Basis und der Name
    `emit`, `connect` oder `disconnect`: Es entsteht eine Kante references auf
    das Signal.
  - Sonst Receiver-Kette: Das erste Glied wird gebunden, jedes weitere ist eine
    innere Klasse, und `foo` wird dort und in der Basiskette gesucht.
  - `new` zielt auf das eigene `_init` der Klasse, wenn sie genau eines hat,
    sonst auf den Klassenknoten (wie `construct` bei Python).
  - Trifft nichts, gibt es keine Kante. Das gilt für eingebaute Funktionen,
    Engine-Methoden, Instanzen und Parameter.
- **references**: Bezeichner nach der Bindung. Verweise auf eine Klasse, die die
  Quelle selbst enthält, entfallen. `emit_signal("x")` sucht das Signal in der
  eigenen Klasse und ihrer Basiskette.

## Umgebung

- `sourceset.extensions` bekommt die vier Endungen. Der Test in `extract/all`
  hält beide Listen gleich. Der Hook-Pfad bleibt reines Go.
- `all.Languages()` hängt `gdscript.Language{}` an. Damit ändert sich
  `all.Version()`, und jeder bestehende loomux-Graph wird einmal neu gebaut,
  auch in reinen Go- oder Python-Repos.
- Speichert der Godot-Editor eine `.tscn`, wird der Graph stale wie bei jeder
  Quelldatei.
- `blast.IsTestPath` bekommt einen `.gd`-Arm: Suffix `_test.gd` oder ein Pfad
  unter `test/` oder `tests/`.
  - `test` ist der Standard-Suchordner von gdUnit4
    (`DEFAULT_TEST_LOOKUP_FOLDER` in dessen `GdUnitSettings.gd`).
  - **Grenze:** Ein in `project.godot` umgestellter `test_lookup_folder` wird
    nicht gelesen, weil `IsTestPath(p string)` keinen Projektkontext hat. Das
    ist in `space` nicht gesetzt.
- Preset `[stack.gdscript.graph]` mit denselben Befehlen wie
  `[stack.python.graph]` (`check graph-fresh`, `check blast-audit --cached
  --threshold 5`). Neue Lanes starten in der Schonfrist.

## Tests

- TDD, 100 % je Funktion. Eine Mutationsrunde je Regel, in der jede
  Teilbedingung einzeln gestrichen und jede Bindungsstufe einzeln abgeschaltet
  wird.
- **Extraktor:** Quelltext direkt an `File()`, je Form aus „Extraktor“ ein Fall,
  dazu ERROR-Fälle (Kopf kaputt, Körper kaputt).
- **Resolver**, mit `extract.Result`-Fixtures:
  - zwei Projekte in einem Repo; eine Datei ohne `project.godot`;
  - zwei gleiche `class_name`; ein `extends`-Zyklus;
  - jede Bindungsstufe einzeln und je mit einem gleichnamigen Gegenkandidaten
    der nächsten Stufe;
  - ein `res://`-Ziel in einer Datei fremder Sprache und eines auf eine Textur;
  - eine Methode, die nur in der Basis der Basis steht (Kette mit drei
    Gliedern).
- **Ende-zu-Ende:** `graph build` auf einer Godot-Welt im Temp-Ordner, nicht
  unter `testdata/`, das sourceset überspringt.
- **Stempel:** Den alten Stempel tragen `extract/treesitter/doc.go`,
  `model/graph_test.go` und `hooks/blast_monitor_test.go` (gegreppt
  2026-10-06). Die Replay-Fälle unter `testdata/cases` halten ihn nicht.

## Doku und Roadmap

- `docs/{en,de}/architecture.md`, `cli-reference.md` (`graph build`: Endungen,
  Sprachen, Parsefehler) und `configuration.md` (Extraktorliste, Preset).
- Wikiseite `docs/wiki/topics/code-graph.md`.
- `README.md` und `README.de.md`:
  - Die Zeile G5c verlässt die Roadmap, G5d hängt an „G5c ✅“.
  - Neue Zeile „Python-Quellwurzeln aus einer Projektdatei, falls eine sie
    hergibt“: `pyRoots` errät die Wurzeln heute. Ob `pyproject.toml` sie
    festlegt, ist offen; es tut das nicht standardisiert.
- `docs/{en,de}/benchmarks.md`: `graph build` in `space` kalt und warm, mit
  Dateien, Knoten und Kanten je Relation.

## Nicht im Umfang

- `uid://`-Auflösung: 0 Vorkommen in den `.gd` von `space`, und Szenen schreiben
  `path` immer mit.
- `var`, `const` und `enum` als Knoten.
- `connect(callable)` als Kante auf die Funktion.
- Klassennamen über Sprachgrenzen (Entscheidung 4).
- Extraktion im Hook-Pfad.

## Geprobt

Probeskript `scratchpad/gdprobe/main.go` (Baumdump), gotreesitter v0.55.0, am
2026-10-06:

- `sample.gd` (alle Formen) und `one.gd` (`class_name Foo extends Bar`,
  Aufrufketten) parsen ohne ERROR, mit den Knotentypen:
  - `class_name_statement` (Felder `name`, `extends`);
  - `extends_statement` (Kind `type` oder `string`);
  - `function_definition` (`static_keyword`, Felder `name`, `body`);
  - `class_definition` (Felder `name`, `extends`, `body: class_body`);
  - `signal_statement`; `const_statement` und `variable_statement` mit Feld
    `value`;
  - `call` (`identifier`, Feld `arguments`);
  - `attribute` mit `identifier`- und `.`-Kindern und `attribute_call` am Ende.
- `space/godot/ui/app_root.tscn` parst ohne ERROR: `section` → `identifier`
  `ext_resource`, `attribute` mit `identifier` und `string`.
- `space/godot/project.godot` parst ohne ERROR: Sektion `autoload` mit
  `property` → `path` und `string`.
- Die Blob-Hashes v0.55.0 gegen v0.55.1 wurden per `BlobSHA256` in
  `grammars/<g>/<g>.go` verglichen.
