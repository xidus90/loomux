# C# im Code-Graphen, in der Prüfkette und an Godot — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** C# steht in Graph, Prüfkette und Kompatibilitätsliste auf der Stufe von Python, und Godot-Szenen und GDScript erreichen C# über die Brücken.

**Architecture:**
- Ein Tree-sitter-Extraktor `csharp` nach dem Muster von `extract/python`.
- Ein Resolver `resolve/csharp.go` nach dem Muster von `resolve/python.go`.
- Godot-Angebote als `Result.Exports`; der Brückenschritt aus PR #149 verbindet sie.
- Prüfkette und Kompatibilität sind Daten (`presets.toml`, `compat.toml`) plus die Spiegel-Tests, die sie schon halten.

**Tech Stack:** Go, gotreesitter v0.57.0 (`grammars/c_sharp`), .NET SDK 10/11 in CI.

**Spec:** `docs/.superpowers/specs/2026-11-10-csharp-design.md` (Programm: `docs/.superpowers/specs/2026-11-10-graph-sprachen-programm.md`)

## Global Constraints

- **TDD.** Jede Funktion hat 100 % Coverage, sonst `//coverage:exempt <Grund>` direkt über `func`.
- **Kein `init()`.** Keine Paketvariable, die eingebettete Daten parst.
- **Sprache.** Code, Kommentare und Commits sind englisch; Doku ist deutsch, außer unter `docs/en/`.
- **Kommentare.** Andere Abstraktionsebene als der Code; keine Historie, kein Task, kein Plan im Code.
- **Commits.** Conventional Commits, Autor ist der Mensch. Kein `Co-Authored-By`, keine Werbezeile. Agenten pushen nie.
- **Tor.** Der Commit-Hook führt Tests, Coverage und Wiki-Lint aus; das ist das Tor. Kein `--no-verify`.
- **Versionen.** Neue Fremdwerkzeuge in Fixtures werden exakt gepinnt, auf die am 2026-11-10 recherchierte neueste Version:
  - xunit.v3 4.0.3
  - Microsoft.NET.Test.Sdk: die neueste laut NuGet, im Task nachgeschlagen
  - coverlet.collector 10.1.0
- **Sprachname** `csharp`, Endung `.cs`, Version `csharp/1`.

## Review Focus

1. **Datei mit `#if`/`#region`-Präprozessor um Mitglieder.** Die Mitglieder müssen trotzdem Knoten bekommen. Test in Task 1.
2. **Gleicher Typname in zwei Namespaces.** Der Resolver darf nur bei Treffern auf derselben Stufe `ReasonAmbiguous` melden. Test in Task 3.
3. **Generische Typen** `class A<T> : B<T>` und Aufrufe `M<int>()`. Die Namen sind ohne Typargumente. Test in Tasks 1 und 3.
4. **Szene mit C#-Skript in einem Unterordner eines zweiten Godot-Projekts.** Keine Kante über die Projektgrenze. Test in Task 4.
5. **Partielle Klasse über zwei Dateien** (Pflicht bei Godot-Skripten). Keine Mehrdeutigkeit, vereinigte Mitglieder. Test in Task 3.
6. **Generierter Code unter `obj/`** landet nicht im Graphen. Test in Task 2.

---

### Task 1: Extraktor `internal/code/extract/csharp`

**Files:**
- Create: `internal/code/extract/csharp/csharp.go` (Language, File, Knoten, Namespaces)
- Create: `internal/code/extract/csharp/edges.go` (Imports, Basen, Aufrufe, Godot-Angebote)
- Tests: `csharp_test.go`, `lines_test.go`, `internal_test.go` im selben Paket

**Interfaces:**
- Produces: `csharp.Language{}` mit `Name() == "csharp"`, `Version() == "csharp/1@" + treesitter.Parser`, `Extensions() == []string{".cs"}`, `File(rel, src) (extract.Result, error)`.
- Produces: die Form der Knoten-IDs:
  - Typen `rel#Typ` bzw. `rel#Outer.Inner`, Mitglieder `rel#Typ.M`, wie Python, ohne Namespace in der ID.
  - Jeder Typ- und Mitgliedsknoten bekommt eine Rohkante `contains` von seinem Elternknoten.
  - `Node.Owner` ist bei Mitgliedern der (qualifizierte) Typname, bei verschachtelten Typen der äußere Typ, bei Typen auf Namespace-Ebene leer (wie Python).
  - Neues Kernfeld in `internal/code/extract/extract.go`: `Result.Enclosing map[model.NodeID]string` mit `json:"enclosing,omitempty"`. Doc-Kommentar: „der Namespace oder das Paket, in dem jeder Typknoten deklariert ist, für den Resolver; Sprachen mit einem Paket je Datei nutzen `Package`“. Der Extraktor füllt ihn für jeden Typknoten (`""` global).
  - Prüfen, ob der Extraktions-Cache (`query/cache.go`) `Result` vollständig per JSON speichert. Sonst den Rundweg für `Enclosing` ergänzen, mit Test.
- Produces: Rohkanten wie Python. `RawEdge.Receiver` ist die Kette vor dem letzten Namen (`"Util.Helper"`), `"this"` oder `"base"`. `RawEdge.Name` ist der Name. `new T()` trägt `Relation: calls`, Receiver und Name wie `T.M()`, und `Member: "new"`. Der Doc-Kommentar von `RawEdge.Member` in `extract.go` nennt diese zweite Bedeutung.
- Produces: `extract.Import{Path: "System.Collections.Generic", Alias: "A"?}`. Welche Felder `extract.Import` hat, liest der Implementierer in `extract.go` nach.
  - `using static` setzt einen Marker: `Alias: "static"`, wenn `Import` kein besseres Feld hat.
  - `global using` wird wie `using` behandelt.

- [ ] **Step 1: Failing Tests schreiben** (Fixtures inline als C#-Quelltext):
  - `TestLanguageDescribesCSharp`: Name, Version, Endungen.
  - `TestTypesAndMembersGetNodes`:
    - class, struct, record, record struct, interface (Art `interface`), enum
    - verschachtelte Klasse `Outer.Inner`
    - Methode, Konstruktor (Name = Typname), Eigenschaft mit Accessor-Rumpf
    - `[Signal]`-Delegate mit Art `signal`
    - IDs, Spans, `Exported` nach den Vorgaben aus der Spec
  - `TestNamespacesSetTheOwnerOfTypes`: Block-, dateiweiter und verschachtelter Namespace, global `""`.
  - `TestUsingFormsBecomeImports`: `using`, `using static`, `global using`, Alias; je eine `imports`-Rohkante mit Zeile.
  - `TestBaseListBecomesExtendsEdges`: `class A<T> : B<T>, IFoo`; Namen ohne Typargumente, qualifizierte Basis `X.Y.Z` mit Receiver `X.Y`.
  - `TestCallsBelongToInnermostMember`: bare `M()`, `this.M()`, `base.M()`, `T.M()`, `A.B.T.M()`, `new T()`, `M<int>()`. Ein Aufruf in einem Feldinitialisierer gehört zum Typ.
  - `TestPreprocessorAroundMembers`: `#if DEBUG` / `#region` um Methoden; alle Methoden haben Knoten.
  - `TestGodotOffers` (Datei `Player.cs`):
    - `[GlobalClass]` ergibt `godot.class`, auch bei einer zweiten Klasse der Datei
    - die Methoden von `Player` ergeben `godot.method`, je Name einmal; eine Überladung `OnHit(int)` / `OnHit()` ergibt ein Angebot auf die erste
    - Methoden einer zweiten Klasse `Other` und einer verschachtelten Klasse ergeben kein Angebot
    - `[Signal] delegate void HitEventHandler()` in `Player` ergibt `godot.signal Hit`
    - `[Signal] delegate void Hit()` (ohne Endung) ergibt kein Angebot
    - `Target` ist jeweils die Knoten-ID
  - `TestEnclosingNamesTheNamespaceOfEveryType`
  - `TestDefinitionInErrorGetsNoNode`: kaputter Kopf, Rest der Datei liest weiter.
  - `lines_test.go`: Zeile jeder Rohkante.
  - `internal_test.go`: nil-Grammatik ergibt einen Fehler.
- [ ] **Step 2:** `go test ./internal/code/extract/csharp/` → FAIL (Paket fehlt).
- [ ] **Step 3: Implementieren.**
  - Knotentypen, wie gotreesitter sie liefert (gemessen am 2026-11-10):
    - `compilation_unit`
    - `using_directive` (Felder `name` beim Alias; Kind `qualified_name`/`identifier`; Token `static`/`global`)
    - `file_scoped_namespace_declaration`, `namespace_declaration` (Feld `name`, `body: declaration_list`)
    - `class_declaration`, `struct_declaration`, `record_declaration`, `interface_declaration`, `enum_declaration` (Felder `name`, `body`; Kind `base_list`; `attribute_list` → `attribute` → Feld `name`)
    - `method_declaration`, `constructor_declaration` (Felder `name`, `parameters`, `body`), `property_declaration` (`accessor_list`), `delegate_declaration`
    - `invocation_expression` (Felder `function`, `arguments`), `member_access_expression` (Felder `expression`, `name`), `object_creation_expression` (Feld `type`), `generic_name`
  - Der Kopf endet vor `body`. Ein Ausdrucksrumpf `=>` ist `body: arrow_expression_clause`.
  - Der Paket-Doc-Kommentar listet die gelesenen Knotentypen wie `python.go`.
- [ ] **Step 4:** `go test ./internal/code/extract/csharp/ -cover` → PASS, 100 %.
- [ ] **Step 5: Commit** `feat(graph): extract C# files`.

### Task 2: C# in der Liste der Sprachen und den Graph-Nebenstellen

**Files:**
- Modify: `internal/code/extract/all/all.go`, `all/all_test.go`
- Modify: `internal/code/sourceset/sourceset.go`, ggf. `legacy_test.go`
- Modify: `internal/code/blast/radius.go` (`IsTestPath`), `internal/code/ask/ask.go` (`testNames`), `internal/code/repomap/map.go` (`langFromPath`), `internal/code/blast/resolve.go` (`isFileQuery`), `internal/code/corpus/walk.go` (`pruneDirs` + `obj`, nicht `bin`)
- Modify: `internal/notices/NOTICE.md` über `go run ./cmd/loomux dev notices`
- Modify: `internal/code/query/build_test.go`, wenn er Sprachen zählt

- [ ] **Step 1: Failing Tests:**
  - in `all_test.go` die erwartete Namensfolge, den kombinierten Versionsstring (sortiert), `For(".cs")` und die Endungsliste nachziehen
  - `IsTestPath`-Fälle (`FooTests.cs`, `Game.Tests/X.cs`, `tests/X.cs` wahr; `Test/Testing.cs`? Ein Segment `Test` ist wahr; `Contest.cs` falsch)
  - `pruneDirs` mit `obj`; `bin` wird weiter gelesen
  - `langFromPath(".cs")`
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implementieren.**
  - Wo eine Liste sortiert gepinnt ist, gilt die bestehende Ordnung.
  - `TestTheNoticeIsCurrent` wird über `dev notices` grün.
- [ ] **Step 4:** `go test ./internal/code/... ./internal/notices/...` → PASS.
- [ ] **Step 5: Commit** `feat(graph): read C# files in the code graph`.

### Task 3: Resolver `internal/code/resolve/csharp.go`

**Interfaces:**
- Consumes: Task 1 (Owner = Namespace bei Typen, Typname bei Mitgliedern; Rohkanten).
- Produces: `resolveCSharp(files []extract.Result) langOut` und `case "csharp"` in `edgesOf`. Die Angebote aus `Result.Exports` sammelt `GraphWith` schon generisch, `langOut.offers` bleibt leer.

- [ ] **Step 1: Failing Tests** in `resolve/csharp_test.go`, alle über `GraphWith` mit echten `csharp.File`-Ergebnissen:
  - Typauflösung je Stufe (verschachtelt, eigener Namespace, äußerer Namespace, Alias, `using`, `global using` aus anderer Datei, global)
  - zwei Treffer auf einer Stufe ergeben `ReasonAmbiguous`; keiner ergibt `ReasonNone`
  - `implements` für Schnittstellen, `extends` sonst
  - `imports` auf die Vertreterdatei des Namespace (kleinster Pfad unter den Nicht-Testdateien nach `blast.IsTestPath`, sonst unter allen), dedupliziert, kleinste Zeile, nie auf sich selbst; ein fremder Namespace (`System`) gibt keine Kante
  - partielle Typen in zwei Dateien: `new Player()` und `extends Player` treffen den Teil mit dem kleinsten Pfad ohne `ReasonAmbiguous`; `this.M()` findet M im anderen Teil; Basen aus beiden Teilen gelten
  - Aufrufe: bare über die Basen (Zyklus `A : B`, `B : A` terminiert), `this.`, `base.`, `T.M()`, `using static`, `new T()` auf den Konstruktor, sonst auf den Typ
  - Überladung: die erste Deklaration
  - `Unresolved` nur für `extends`/`implements`
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implementieren nach dem Aufbau von `resolve/python.go` (Index, Schleife, Aufrufe nach den Basen).
- [ ] **Step 4:** `go test ./internal/code/resolve/ -cover` → PASS, 100 % in `csharp.go`.
- [ ] **Step 5: Commit** `feat(graph): resolve C# usings, bases and calls`.

### Task 4: Godot-Brücken mit dem echten C#-Extraktor

**Files:**
- Test: `internal/code/resolve/godot_csharp_test.go`. Fixtures sind ein `project.godot`, eine `.tscn` mit `[ext_resource type="Script" path="res://Player.cs"]` und `[connection signal="Hit" … method="OnHit"]`, `.gd`-Dateien und `.cs`-Dateien; Extraktion über `all.For`.

- [ ] **Step 1: Tests:**
  - Szene → `Player.cs#Player.OnHit` (`calls` bzw. die Relation, die die GDScript-Übergabe trägt), auch wenn `OnHit` überladen ist
  - Szene → Signal-Delegate
  - GDScript `Mob.new()` → `[GlobalClass] Mob`
  - Szene → `Player.cs` als Dateiknoten
  - Handler in zweiter `partial`-Datei: keine Kante
  - zweites Godot-Projekt: keine Kante
- [ ] **Step 2:** Laufen lassen. Erwartet: PASS ohne Produktionsänderung, weil die Brücken stehen. Fällt einer, ist das ein Befund. Dann gilt der systematische Fehlerweg; nie den Test anpassen.
- [ ] **Step 3: Commit** `test(graph): pin Godot scenes reaching C# scripts`.

### Task 5: Prüfkette, Stack `csharp`

**Files:**
- Modify: `internal/verify/presets.toml` (Endung `.cs`, `[stack.csharp]` mit lint/types/test/coverage/graph/rules nach der Spec)
- Modify: `internal/verify/schema.go` (`StackNames` + `csharp`)
- Modify: `internal/verify/plan.go` (`carrierRank`, `IsSource`)
- Modify: `internal/verify/rules/files.go` (`contentExts`)
- Modify: `internal/detect/signals.go` (Zeilen aus der Spec)
- Modify: `docs/en/support.md`, `docs/de/support.md` (Zeile `csharp`)
- Modify: `presets_test.go` (`TestTreeSitterGraphLanesAreTheGoOne` + `csharp`)

- [ ] **Step 1: Failing Tests:**
  - die Schleife in `presets_test` um `csharp` erweitern
  - Erkennung eines Ordners mit `x.csproj` meldet `csharp`
  - Plan für einen C#-Ordner: Lanes mit den erwarteten Befehlen
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implementieren.**
  - Coverage: Prüfen, ob `check linecov` ohne `--report` den `coverage.cobertura.xml` von coverlet selbst findet (`linecov/check.go reportFiles`, `tools.toml` `reports`).
    - Findet er ihn: `test.measuring` mit `--collect "XPlat Code Coverage" --results-directory <Pfad je Lauf>` (alte `TestResults/<guid>/` dürfen nicht gelesen werden), `coverage.commands = ["{loomux} check linecov …"]`.
    - Sonst ist der kleinste Weg, ihn finden zu lassen, Teil dieses Tasks, mit Test.
  - `tests_when = ["*.csproj:Microsoft.NET.Test.Sdk"]`, wenn das Muster-Format Globs vor `:` erlaubt (nachsehen); sonst die nächstliegende Form.
- [ ] **Step 4:** `go test ./internal/verify/... ./internal/detect/...` → PASS.
- [ ] **Step 5: Commit** `feat(verify): check C# projects with dotnet`.

### Task 6: Kompatibilität für .NET

**Files:**
- Modify: `internal/compat/compat.go`:
  - `runtimeOf["csharp"] = "dotnet"`
  - `versionRe` wird `^\d+(\.\d+)?$`
  - die Meldung `stacks are …` aus `runtimeOf` bilden statt fest
- Modify: `internal/compat/compat.toml` (Einträge aus der Spec)
- Create: `internal/compat/testdata/csharp-xunit/`:
  - `Lib/Lib.csproj` (net10.0; `TargetFrameworks` je Eintrag über `prepare`, falls nötig), `Lib/Calc.cs`
  - `Lib.Tests/Lib.Tests.csproj` (xunit.v3, Microsoft.NET.Test.Sdk, coverlet.collector, exakt gepinnt), `Lib.Tests/CalcTests.cs`
  - `csharp-xunit.sln`, `expect.toml` mit `[[break]]` je Art
- Create: `internal/compat/testdata/csharp-syntax-10.0/`, `csharp-syntax-11.0/`:
  - Formen von C# 14 bzw. 15 aus der Recherche (dem Ledger entnehmen)
  - `expect.toml` mit Kanten, die nur bei richtigem Parse entstehen
- Create: `internal/compat/testdata/godot4-csharp/`:
  - `project.godot`, `Main.tscn`, `Player.cs`, `Mob.cs` (`[GlobalClass]`), `spawn.gd`
  - `expect.toml` mit den Kanten Szene → Methode und Szene → Signal
- Modify: `internal/compat/run.go`: `prepare` für `dotnet` setzt `TargetFramework` auf `net<version>`, wenn die Fixture das braucht
- Modify: `internal/compat/scout.go`:
  - `dotnetSource`, Parser für `releases-index.json`
  - Stufe aus `preview`/`rc`
  - EOL über endoflife.date `dotnet`
- Modify: `ci/compat-setup.sh` (`dotnet) ;;`), `.github/workflows/compat.yml` (`actions/setup-dotnet` mit der neuesten Hauptversion der Action, SHA-gepinnt wie die übrigen Actions; `dotnet-version` aus der Beschreibung)
- Modify: `docs/{en,de}/support.md` (Tabelle „Verified combinations“)

- [ ] **Step 1: Failing Tests:**
  - `layer1`: jede Entry besteht Erkennung, Presets und Plan
  - `TestEveryFixtureIsUsed`
  - `docs_test`
  - Scout-Parser-Test mit einer gespeicherten Kopie von `releases-index.json` unter `testdata`
  - Versionsformat `21` und `10.0` gültig, `1.2.3` ungültig
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implementieren.**
- [ ] **Step 4:** `go test ./internal/compat/...` → PASS.
  - Lokal mit installiertem `dotnet` zusätzlich `go run ./cmd/loomux dev compat run <key>` für `csharp-xunit@10.0`, wenn `dotnet` da ist.
  - Fehlt `dotnet`, im Ledger festhalten; CI ist dann der Beweis.
- [ ] **Step 5: Commit** `feat(compat): verify C# on .NET 10 and 11`.

### Task 7: Doku

**Files:**
- `docs/wiki/topics/code-graph.md`:
  - C# unter den Sprachen
  - Brücken jetzt mit C#
  - offener Punkt C# entfernen, die `partial`-Lücke nennen
- `docs/{en,de}/cli-reference.md` (graph build: Endungen, Sprachen)
- `docs/{en,de}/configuration.md` (Extraktorliste)
- `docs/{en,de}/architecture.md`
- `README.md` und `README.de.md`: Code-Graph-Zeile; Roadmap-Zeile „C# in the code graph (Godot)“ entfällt
- `docs/{en,de}/stacks/csharp.md`, wenn `support.md` auf Stack-Seiten verlinkt (nachsehen; Vorbild `gdscript.md`)

- [ ] **Step 1:** Doku schreiben. Der Wiki-Lint läuft im Commit-Hook.
- [ ] **Step 2: Commit** `docs: describe C# in the code graph and the check chain`.
