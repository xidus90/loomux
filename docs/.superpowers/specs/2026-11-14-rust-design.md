# Rust im Code-Graphen, in der Prüfkette und an Godot (gdext)

**Stand:** 2026-11-14 · **Zweig:** `feat/graph-rust` (ab `origin/master`,
v1.27.0) · folgt dem Muster von
`2026-11-10-graph-sprachen-programm.md` (C#, C++), baut auf den Brücken
(`2026-11-10-graph-bruecken-design.md`) und der Godot-Verdrahtung von C++
(`2026-11-10-cpp-design.md`, Abschnitt „Godot-Verdrahtung“)

## Anlass und Ziel

Der Mensch startet ein Projekt in Rust mit Godot. loomux soll dort von Anfang
an voll mitarbeiten, für reines Rust und für Rust mit Godot über godot-rust
(gdext). Heute erkennt loomux `Cargo.toml` als Stack `rust` und hat einen
Lint-Eintrag; Graph, Tests, Coverage und Kompatibilitätsliste fehlen.

Erfolg heißt:

- **Graph:** `loomux graph build` liest `.rs`-Dateien. Module, `use`,
  `impl` und Traits sind aufgelöst, Aufrufe über Dateien und Crates eines
  Workspaces hinweg verbunden. Blast, Repo-Map, Ask und `check rules` arbeiten
  auf Rust wie auf Python.
- **Godot:** Eine Rust-Klasse (`#[derive(GodotClass)]`) wird von GDScript
  (`extends`, `.new()`), von Szenen (Knotentyp) und von Szenen-Verbindungen
  auf `#[func]` und `#[signal]` erreicht, wie heute eine C++-Klasse.
- **Prüfkette:** lint (clippy, rustfmt), types (`cargo check`), test
  (`cargo test`), coverage (`cargo-llvm-cov`, lcov) mit 100 %-Regel, graph
  und rules laufen über die bestehenden Profile.
- **Kompatibilität:** Rust steht in `compat.toml`, mit Scout-Quelle und CI.

Der Mensch hat entschieden: Graph zuerst, Linting gehört dazu, alles in
dieser einen Spec, gegliedert in Phasen und Aufgaben.

## Nicht-Ziele

- **Keine Makro-Expansion.** Was ein Makro erzeugt (`#[derive]`,
  `macro_rules!`, prozedurale Makros), sieht der Graph nicht. Ausnahmen sind
  die gdext-Attribute, die der Extraktor als Text liest, und Aufrufe in
  Makro-Argumenten (siehe Extraktor).
- **Keine Typinferenz.** `x.f()` mit unbekanntem Typ von `x` wird nur über
  einen eindeutigen Methodennamen verbunden, als `inferred`.
- **Kein Trait-Dispatch.** Ein Aufruf über `dyn Trait` oder einen generischen
  Parameter trifft nicht die Implementierungen.
- **Kein `cfg`-Auswerten.** Alle `#[cfg(...)]`-Zweige werden gelesen.
- **Keine Abhängigkeiten aus crates.io.** Kanten enden an der Grenze des
  Repos; `std`, `godot` und andere Crates ergeben keine Kante.
- **Kein Bauen von C++- oder C#-Extensions in der Godot-Prüfung.** Nur eine
  Rust-Extension wird vor der GDScript-Suite gebaut (Entscheidung 10).
- **Keine `.gdextension`-Datei im Graphen.** Die Brücke verbindet über
  Klassennamen. Die Datei liest nur `check godot-suite`, um die Extension zu
  bauen.
- **Keine Paarungsregel für Tests** in `rules/tests.toml`. C# und C++ haben
  auch keine.

## Wege (Approaches)

| Weg | Was | Abwägung |
|---|---|---|
| **A — Extraktor + eigener Resolver** *(gewählt)* | Tree-sitter liest jede Datei, ein Resolver baut den Modulbaum aus `Cargo.toml`, `mod` und `use` und löst Aufrufe auf | Im Prozess, schnell, gleiche Bauart wie Python/C#/C++. Blast und Regeln tragen voll. Größter Brocken ist der Resolver. |
| B — nur Extraktor, Aufrufe über eindeutige Namen | Knoten, `contains`, Imports; Aufrufe nur bei global eindeutigem Namen | Ein Drittel des Aufwands, aber blind bei `new`, `ready`, `process`, `update` — gerade in gdext-Code. |
| C — rust-analyzer | Kanten aus dem Sprachserver (`lsp_resolved`) | Exakt, aber externes Werkzeug und `cargo check` pro Build, zu langsam für den Hook-Pfad, bricht mit dem Tree-sitter-Kern. Verworfen. |

Der Mensch hat **A** gewählt.

## Messung (gemacht am 2026-11-14)

gotreesitter v0.57.0 (in `go.mod` schon gepinnt, neueste Version; die
Grammatik ist tree-sitter-rust v0.24.2, die neueste), flach geklonte Repos,
Definitionen = `function_item`, `function_signature_item`, `struct_item`,
`enum_item`, `trait_item`, `impl_item`, `mod_item`:

| Repo | Dateien | mit ERROR | Definitionen | ohne ERROR-Vorfahr | eigener Teilbaum ohne ERROR |
|---|---|---|---|---|---|
| godot-rust/gdext | 384 | 1 (0,3 %) | 8068 | 100 % | 100 % |
| BurntSushi/ripgrep | 102 | 0 | 3857 | 100 % | 100 % |
| tokio-rs/tokio | 816 | 0 | 11426 | 100 % | 100 % |
| godot-rust/demo-projects | 16 | 0 | 121 | 100 % | 100 % |

**Befund:** Die Grammatik trägt. Es gilt die **Kopfregel** wie bei C#, Java
und PHP: Eine Definition, deren Kopf einen ERROR enthält, bekommt keinen
Knoten (`treesitter.HoldsError` auf dem Kopf). Bei 0,3 % fehlerhaften Dateien
ist die Wahl zwischen Kopf- und Teilbaumregel ohne messbare Folge.

Geprüft an einem gdext-Beispiel: Attribute (`#[derive(GodotClass)]`,
`#[class(...)]`, `#[func]`, `#[signal]`) sind `attribute_item`-Geschwister
**vor** dem Item, ihre Argumente ein ungeparster `token_tree`. Ein
`#[signal] fn hit(damage: i32);` ist ein `function_signature_item`, kein
ERROR. In `assert_eq!(helper(1), 2)` steht `helper` als `identifier` vor
einem `token_tree`, der mit `(` beginnt.

## Entscheidungen

1. **Eine Sprache `rust`, Endung `.rs`.** Version `rust/1@<Parser>`,
   Grammatik `gotreesitter/grammars/rust`.
2. **`Cargo.toml` wird gelesen wie `go.mod` und `tsconfig.json`**: über
   `resolve.ModuleFile` und `resolve.Modules`, nicht als Quelldatei.
   `.toml` kommt nicht in `sourceset.extensions`. `resolve.Module` bekommt
   ein Feld `Cargo *CargoManifest` (nil sonst), neben `TS`. Gelesen mit dem
   TOML-Parser, den loomux schon hat. *Kosten, falls falsch:* ein weiterer
   Leser; die Resolver-Schnittstelle bleibt.
3. **Methoden über Dateigrenzen.** `impl Player` kann in jedem Modul des
   Crates stehen. Der Extraktor gibt eine Methode als Knoten mit
   `Owner = "Player"` (der Typ, wie geschrieben) und einer `contains`-Kante
   von der Datei aus, oder vom Typ, wenn der Typ in derselben Datei steht.
   Eine `contains`-Kante über Dateigrenzen gibt es nicht. So hält es C++ mit
   Methoden, die außerhalb der Klasse definiert sind (`resolve/cpp.go:40-44`,
   `:170`). Der Resolver findet Methoden über `Owner` in einem Index je
   Crate. *Kosten, falls falsch:* keine; Blast findet Methoden über ihre
   Aufrufe.
4. **Tests am Knoten.** Rust-Tests stehen meist in der Produktivdatei
   (`#[cfg(test)] mod tests`). Heute entscheidet jede Stelle über den Pfad
   (`blast/radius.go:173`, `:192`, `blast/resolve.go:142`,
   `query/audit.go:60`, `ask/ask.go:241`, `resolve/csharp.go:150`,
   `resolve/java.go:131`).
   - `model.Node` bekommt `Test bool` (`json:"test,omitempty"`). `decode.go`
     weist unbekannte Felder nicht ab; `omitempty` lässt alle bestehenden
     Graphen byte-gleich. `schemaVersion` bleibt 3.
   - Der Rust-Extraktor setzt `Test` für eine Funktion mit `#[test]` (auch
     `#[tokio::test]` und jedes Attribut, dessen letzter Pfadteil `test`
     heißt) und für alles innerhalb eines Moduls mit `#[cfg(test)]`.
   - Neu `blast.IsTestNode(n *model.Node) bool` = `n.Test ||
     IsTestPath(n.Path)`. Jede genannte Stelle, die einen Knoten in der Hand
     hat, fragt künftig `IsTestNode`; Stellen, die nur einen Pfad haben,
     bleiben bei `IsTestPath`.
   - **Falle im Blast-Signal:** „Ein Test hat sich mitgeändert“ vergleicht
     heute Pfade mit dem Diff. Ein Test in derselben Datei zählt dann immer
     als geändert. Für Testknoten aus einer Produktivdatei zählt stattdessen,
     ob der Diff die Zeilen ihres Spans berührt. Die Hunks liegen in
     `blast.Radius` schon vor (`radius.go:112`, `f.Hunks`); `signal` bekommt
     sie neben `inDiff`. Ein zurückgehaltener Pfad (`withheld`) hat keine
     Hunks und zählt weiter als ganz geändert.
   - `signal` liefert heute für eine Testdatei `na` (`radius.go:173`). Für
     eine Produktivdatei mit Inline-Tests bleibt das Signal bestehen; nur
     Testknoten als Ausgangspunkt werden übersprungen.
   - *Kosten, falls falsch:* ein Feld im Modell; ohne es meldet Blast für
     jede Rust-Änderung „kein Test“ oder „Test geändert“ falsch.
5. **Aufrufe in Makro-Argumenten** (`assert_eq!(f(x), 3)`,
   `println!("{}", g())`) liest der Extraktor heuristisch aus dem
   `token_tree`: ein `identifier` oder ein Pfad aus `identifier`/`::`,
   gefolgt von einem `token_tree`, der mit `(` beginnt, ist ein Aufruf;
   `. identifier (` ein Methodenaufruf. Konfidenz `inferred`. Ohne das sähe
   kein Test, der über `assert!` aufruft, seinen Code. *Kosten, falls
   falsch:* einzelne Scheinkanten, als `inferred` markiert und damit
   für den Agenten sichtbar (`2026-11-10-kanten-konfidenz-design.md`).
6. **`inferred` nur nach der bestehenden Regel:** ein Aufruf, den der
   Resolver allein über einen eindeutigen Namen zuordnet, und die
   Makro-Heuristik. Alles, was über Pfad, `use` oder `impl`-Typ aufgelöst
   ist, ist `extracted`.
7. **Godot-Angebote aufgeteilt wie bei C++:** `godot.class` ist ein
   `Export` des Extraktors (wie `extract/cpp/edges.go:215`), denn der
   Godot-Name (`#[class(rename=X)]`) steht am `struct` in derselben Datei.
   `godot.method` und `godot.signal` bildet der Resolver (wie
   `resolve/cpp.go:82-100`), denn `#[func]` und `#[signal]` stehen in einem
   `#[godot_api] impl`, das in einer anderen Datei liegen kann; `Class` =
   Godot-Name des `impl`-Typs.
8. **Werkzeugwahl Coverage: `cargo-llvm-cov`** (0.9.1 vom 2026-09-06,
   gepflegt, lcov, läuft unter Windows MSVC). `cargo-tarpaulin` (0.37.6) ist
   ebenfalls gepflegt, misst aber mit ptrace nur unter Linux x86_64 und
   anderswo über denselben LLVM-Weg. *Kosten, falls falsch:* ein anderer
   `measuring`-Befehl; der lcov-Leser bleibt.
9. **Compat-Achse:** Upstream pflegt nur die neueste stable-Version
   (endoflife.date `rust`). Die Achse ist darum **eine Laufzeitversion**, die
   aktuelle stable (heute `1.100`). Dazu kommen Syntax-Fixtures je
   **Edition** (`2021`, `2024`), die nur den Graphen prüfen, wie
   `cpp-syntax-NN`. 2015 und 2018 bleiben weg: Neue Projekte legt Cargo in
   der neuesten Edition an, und der Graph liest beide gleich.
   *Kosten, falls falsch:* zwei weitere Syntax-Einträge.
10. **GDScript-Tests gegen die Rust-Extension (Entscheidung des Menschen:
    jetzt).** Die Lanes kennen `after` nur innerhalb eines Stacks und einer
    Area (`plan.go:937-969`); das Crate (`rust/`) und das Godot-Projekt
    (`godot/`) sind zwei Areas zweier Stacks. Statt einer neuen Ordnung über
    Stacks hinweg baut **`check godot-suite` die Extension selbst**, bevor
    es Godot startet, und ebenso `loomux godot rescan` vor dem Import:
    - Es liest jede `.gdextension` im Projekt mit `entry_symbol` =
      `gdext_rust_init`. Keine Wahl eines Plattformschlüssels: Godot wählt
      über Feature-Tags, und echte Projekte schreiben `linux.editor.x86_64`,
      Schlüssel ohne Architektur und Ähnliches. Stattdessen gilt: Jeder
      `[libraries]`-Pfad, der in `target/<profil>/` zeigt, baut dieses
      Profil einmal für den Host (`debug` → ohne Flag, `release` →
      `--release`, ein anderes → `--profile <profil>`). Das Crate ist das
      nächste `Cargo.toml` über diesem `target/`.
    - **Bekannte Lücke:** Ein eigenes Zielverzeichnis (`CARGO_TARGET_DIR`,
      `build.target-dir` in `.cargo/config.toml`) bricht die Suche nach dem
      Crate; dann wird nichts gebaut, und die Lane sagt das in einer Notiz.
    - **Zeitbudget:** Ein kalter Bau des `godot`-Crates dauert Minuten. Der
      Bauende prüft, ob Lanes in den Profilen `precommit` und `stop` ein
      Zeitlimit haben, und legt fest, wie die Suite damit umgeht.
    - **Windows:** Hat der Godot-Editor die DLL geladen, kann `cargo build`
      sie womöglich nicht überschreiben. Der Bauende prüft das
      (`reloadable = true` lässt Godot ab 4.2 eine Kopie laden; *überholt,
      siehe die Notiz vom 2026-11-15 am Ende*); die Abnahme
      enthält diesen Fall.
    - Scheitert `cargo build`, scheitert die Lane mit dessen Ausgabe; fehlt
      `cargo`, meldet sie `missing-tool`.
    - Andere `.gdextension` (C++, fremde Addons) werden nicht angefasst.
    - *Kosten, falls falsch:* ein Baulauf mehr pro Suite (Cargo baut
      inkrementell); später eine allgemeine Ordnung über Stacks, falls C++
      und C# dasselbe brauchen.
11. **Ausschluss mit Begründung über Blockkommentare (Entscheidung des
    Menschen).** Das Profil `rust` in `linecov/tools.toml` bekommt die
    Block-Marker `LCOV_EXCL_START` (mit Grund) und `LCOV_EXCL_STOP`, wie das
    Profil `lcov` sie schon hat (`tools.toml:69-70`). Der bestehende
    Einzeilen-Marker `coverage(off)` bleibt. Auf stable ist die Form:
    `// LCOV_EXCL_START: <Grund>` … `// LCOV_EXCL_STOP`.

## Phasen und Aufgaben

Jede Aufgabe ist TDD und endet mit 100 % Coverage je Funktion. Doku und Wiki
werden **in derselben Aufgabe** nachgezogen, die den Code ändert (Regel aus
CLAUDE.md), nicht in einer Schlussphase. Die Spiegel-Tests der Programm-Spec
sind das Tor: `all_test`, `sourceset`-Gleichheit, `support_test`, `layer1`,
`TestEveryFixtureIsUsed`, `notices_test`, `docs_test`.

### Phase 1 — Extraktor `internal/code/extract/rust`

**Aufgabe 1.1 — Knoten und `contains`**
- `Language{}`: `Name()` = `rust`, `Version()` = `rust/1@` +
  `treesitter.Parser`, `Extensions()` = `.rs`.
- Knoten:
  - die Datei;
  - `function_item` auf Modulebene und in Inline-Modulen (`mod x { … }`);
  - `struct_item`, `enum_item`, `union_item`, `trait_item`, `type_item`
    (Alias);
  - `macro_definition` (`macro_rules! name`), Art `macro`;
  - Methoden: `function_item` und `function_signature_item` im
    `declaration_list` eines `impl_item` oder `trait_item`.
- Namen: Inline-Module qualifizieren mit `::` (`tests::adds`); Methoden
  heißen `Typ.methode` wie in den anderen Sprachen, `Owner` = Typ wie
  geschrieben, ohne Generika (`impl<T> Foo<T>` → `Foo`).
- Ein `impl` hat keinen eigenen Knoten.
- Die Kopfregel (siehe Messung) mit einem Test, der eine Definition mit
  kaputtem Kopf zeigt, und einem, der zeigt, was daneben ankommt.
- `Exported` = `pub` in jeder Form (`pub`, `pub(crate)`, …).

**Aufgabe 1.2 — `use`, `mod`, Imports**
- `use_declaration` → `extract.Import` je gebundenem Namen: `scoped_identifier`,
  `use_as_clause` (Alias), `scoped_use_list` und `use_list` (verschachtelt
  ausgefaltet), `use_wildcard` (Name `*`). `pub use` wird markiert
  (Re-Export, für den Resolver).
- `mod x;` (ohne Rumpf) → Rohkante `imports` auf das Modul `x`, mit
  `#[path = "..."]`, falls vorhanden, als Specifier.
- `extern crate x as y;` → Import mit Alias.

**Aufgabe 1.3 — Aufrufe, `implements`, `extends`**
- `call_expression`:
  - `identifier` → bare call;
  - `scoped_identifier` (`a::b::f`, `Typ::new`, `Self::f`) → Name mit
    Receiver = Pfad;
  - `field_expression` (`x.f()`) → Member-Aufruf; bei `self.f()` Owner = der
    `impl`-Typ.
- `macro_invocation` → `calls` auf ein `macro_rules!` gleichen Namens
  (bare), dazu die Heuristik aus Entscheidung 5 im `token_tree`.
- `impl Trait for Typ` → `implements` Typ → Trait (beide als Pfad).
- `trait A: B + C` → `extends` A → B, A → C.
- Ein Aufruf gehört zur innersten Definition mit Knoten, sonst zur Datei.

**Aufgabe 1.4 — Tests am Knoten**
- `model.Node.Test` (Entscheidung 4), gesetzt vom Rust-Extraktor.
- `blast.IsTestNode`; alle Stellen aus Entscheidung 4 umgestellt, mit Test
  je Stelle; die Span-Regel für das Blast-Signal.
- `blast.IsTestPath` für `.rs`: jede Datei unter einem Verzeichnis `tests/`
  oder `benches/` (Cargos Integrationstests und Benchmarks), und jede Datei
  namens `tests.rs` oder `test.rs`. Letzteres fängt das ausgelagerte
  Testmodul (`#[cfg(test)] mod tests;` → `tests.rs`), dessen Attribut am
  `mod` der Elterndatei steht und das der Extraktor darum nicht sieht.
  *(Entscheidung des Planers: Der Resolver ändert heute keine Knoten; eine
  Namensregel deckt die übliche Form ohne neuen Mechanismus. Ein anders
  benanntes ausgelagertes Testmodul ist eine bekannte Lücke.)*
  Negativfall im Test: `src/tests_util.rs` zählt nicht.

**Aufgabe 1.5 — gdext-Rohdaten**
- Attribute vor einem Item werden als Text gelesen (Pfad + `token_tree`).
- `#[derive(..., GodotClass, ...)]` am `struct` → Godot-Klasse; Name =
  `rename=X` aus `#[class(...)]`, sonst der Struct-Name. Mit
  `godot::prelude::GodotClass` qualifiziert zählt ebenfalls.
- In einem `impl` mit `#[godot_api]`: `#[func]` (Name aus `rename=`, sonst
  Methodenname) und `#[signal]` (Methodenname) als Rohkanten mit `Member` =
  `godot.method` / `godot.signal`, Receiver = der `impl`-Typ, wie die
  C++-Bindungen (`extract/cpp/edges.go:129-174`).
- `#[rpc(...)]` gilt wie `#[func]`.

**Wiki/Doku in Phase 1:** keine; die Sprache ist noch nicht in
`extract/all` (Aufgabe 2.5).

### Phase 2 — Resolver `internal/code/resolve/rust.go`

**Aufgabe 2.1 — `Cargo.toml` lesen**
- `ModuleFile` erkennt `Cargo.toml`; `Modules` füllt `CargoManifest`:
  - `[package] name`; Bibliotheksname = `[lib] name` oder `name` mit `-` →
    `_`;
  - Crate-Wurzeln: `[lib] path` oder `src/lib.rs`; `[[bin]] path` oder
    `src/main.rs` und `src/bin/*.rs`, `src/bin/*/main.rs`; `tests/*.rs`,
    `benches/*.rs`, `examples/*.rs` als eigene Wurzeln;
  - `[workspace] members` (mit `*`-Globs) und `exclude`;
  - `[dependencies]`, `[dev-dependencies]` mit `path = …`: der Name im Code
    (der Schlüssel, oder `package = …` umbenannt) zeigt auf das Crate im
    Repo. `workspace = true` folgt `[workspace.dependencies]`.
- Ein `Cargo.toml` mit Syntaxfehler: Das Crate fällt aus der Auflösung, der
  Build bricht nicht ab (wie ein kaputtes `tsconfig.json`).
- Ein geändertes `Cargo.toml` macht den Graphen **nicht** veraltet, genau
  wie `go.mod` und `tsconfig.json` heute
  (`freshness/fingerprint.go:122-133`, Wiki `topics/code-graph.md:689-691`).
  Die Wiki-Zeile nennt `Cargo.toml` mit. *(Entscheidung des Planers: die
  Freshness-Probe läuft im Hook-Pfad unter dem Abhängigkeits-Tor; das
  zu ändern ist ein eigenes Thema für alle drei Dateien.)*

**Aufgabe 2.2 — Modulbaum**
- Je Crate-Wurzel: `mod x;` in einer Wurzel-, `mod.rs`- oder
  gleichnamigen Datei → `x.rs` oder `x/mod.rs` im passenden Verzeichnis
  (`a.rs` → `a/x.rs`). `#[path]` geht vor.
- Inline-Module sind Teil des Baums.
- Jede `.rs`-Datei, die kein Baum erreicht, bleibt im Graphen mit ihren
  Knoten; ihre Pfade lösen relativ zur eigenen Datei auf (`self::`,
  `super::` gehen dann ins Leere).

**Aufgabe 2.3 — Pfade und `use`**
- `crate::`, `self::`, `super::`, `::name`, ein Crate-Name aus Aufgabe 2.1.
- Ein Name in einer Datei bindet, in dieser Reihenfolge: Definition im
  selben Modul → `use` (mit Alias) → Glob-`use` → Crate-Name. Zwei Treffer
  derselben Stufe ergeben keine Kante.
- `pub use` wird verfolgt (Re-Export-Ketten, mit Zyklusschutz wie
  `python.go bound`).
- `Unresolved` nur für `implements`/`extends`, wie bei C++.

**Aufgabe 2.4 — Kanten**
- `contains` wie vom Extraktor (Entscheidung 3); Methodenindex je Crate
  über `Owner`.
- Aufrufe: bare → Modul/`use`; Pfad → Funktion oder assoziierte Funktion
  (`Typ::new` → Methode `new` eines `impl Typ` im Crate); `Self::`/`self.`
  → Methode des `impl`-Typs, sonst seiner Traits mit Default-Rumpf;
  `x.f()` → eindeutige Methode `f` im Crate, `inferred`, sonst keine Kante.
- Tupel-Struct- und Enum-Variant-Konstruktoren (`Foo(1)`,
  `Option::Some`) ergeben eine Kante auf den Typ nur, wenn der Typ im Repo
  definiert ist.
- `implements`, `extends` über Aufgabe 2.3.
- `imports`: Datei → Datei des Moduls (`mod x;`), Datei → Datei der
  Definition (`use`).

**Aufgabe 2.5 — Einhängen und Graph-Stellen**
- `resolve.edgesOf`: `case "rust"`.
- `extract/all.Languages` und die Pins in `all_test.go`.
- `sourceset.extensions` um `.rs`; `legacy_test.go` bleibt eingefroren.
- `repomap.langFromPath`: `.rs` → `rust`.
- `blast.isFileQuery`: `.rs` steht schon drin.
- `corpus.pruneDirs`: `target` steht schon drin.
- `ask.testNames`: nichts zu tun, die Pfadregel liegt in `IsTestPath`.
- `NOTICE.md` neu erzeugen (Grammatik `rust`, MIT).
- **Wiki/Doku:** `docs/wiki/topics/code-graph.md` (Absatz „Rust“ wie bei
  C#/C++, Test-Pfad-Liste, „Was offen“ mit den Lücken unten),
  `docs/wiki/log.md`, `docs/{en,de}/support.md` (Tabelle „was der Graph
  liest“), `docs/{en,de}/architecture.md` (Paketliste), beide READMEs
  (Sprachliste).

### Phase 3 — Godot-Brücke

**Aufgabe 3.1 — Angebote**
- Der Rust-Extraktor bietet `godot.class <Name>` auf den Struct-Knoten an
  (`Result.Exports`, Entscheidung 7).
- Der Rust-Resolver bietet an:
  - `godot.method <Name>` und `godot.signal <Name>` auf den Methodenknoten,
    `Class` = Godot-Name des `impl`-Typs (über den Struct, auch aus einer
    anderen Datei des Crates).
- Löst sich der `impl`-Typ nicht auf, entsteht kein Angebot.
- Ein Crate außerhalb des Godot-Projekts (`rust/` neben `godot/`) bietet
  ohne Realm an; jedes Projekt sieht es (`bridge.go:178`).

**Aufgabe 3.2 — Tests der Verdrahtung**
- `resolve/godot_rust_test.go` nach dem Vorbild von `godot_cpp_test.go`:
  `extends PlayerX`, `PlayerX.new()`, `.tscn`-Knoten `type="PlayerX"`,
  Verbindung auf `#[signal] hit` und auf `#[func(rename=do_move)]`, alles
  mit `rename` und ohne.
- Zwei Rust-Klassen gleichen Godot-Namens in einem Realm: keine Kante
  (`cardOne`), wie bei C++.

**Aufgabe 3.3 — Compat-Fixture `godot4-rust`**
- Layout wie die gdext-Vorlage: `godot/project.godot`, `godot/*.gd`,
  `godot/*.tscn`, `rust/Cargo.toml`, `rust/src/lib.rs`, `rust/src/player.rs`.
- `godot/rust.gdextension` mit `entry_symbol = "gdext_rust_init"` und
  `[libraries]` auf `res://../rust/target/debug/…` für Linux, Windows und
  macOS; `godot/addons/gdUnit4` und eine Suite `godot/test/player_test.gd`,
  die die Rust-Klasse erzeugt und eine `#[func]` aufruft.
- `rust/Cargo.toml`: `crate-type = ["cdylib"]`, `godot` exakt gepinnt
  (heute `=0.5.5`), `api-4-x`-Feature passend zur Godot-Version des
  Eintrags.
- `expect.toml` mit den Kanten aus Aufgabe 3.2.
- **Wiki:** `topics/code-graph.md` Abschnitt Godot-Angebote um Rust.

**Aufgabe 3.4 — `check godot-suite` und `godot rescan` bauen die
Rust-Extension** (Entscheidung 10)
- Leser für `.gdextension` (INI-Form, `[configuration]` und `[libraries]`),
  Wahl des Plattformschlüssels (*überholt, siehe die Notiz vom 2026-11-15 am
  Ende*), Suche des Crates, Aufruf von `cargo build` über den bestehenden
  Prozess-Weg der Suite.
- Tests: Extension gebaut vor dem Godot-Start (Reihenfolge über eine
  Naht), `release`-Pfad, C++-`.gdextension` unberührt, `cargo` fehlt →
  `missing-tool`, Baufehler → Lane rot mit Ausgabe, kein `.gdextension` →
  nichts passiert.
- **Wiki/Doku:** `docs/{en,de}/stacks/gdscript.md` (die Suite baut eine
  Rust-Extension), `cli-reference.md` (`check godot-suite`,
  `godot rescan`), `topics/code-graph.md` nicht betroffen, `log.md`.

### Phase 4 — Prüfkette (Stack `rust`)

**Aufgabe 4.1 — Presets** (`internal/verify/presets.toml`)
- `tests_when`: `*.rs:#[test]` und `tests/` (der Bauende prüft, dass die
  Inhaltsform mit Glob in `tests_when` greift, sonst `Cargo.toml` allein).
- **lint:** `cargo clippy --all-targets --config build.warnings="deny"`
  und `cargo fmt --check`; `on_file`: `cargo fmt --check -- {file}`.
  `build.warnings` ist in Cargo stabil und ersetzt `-- -D warnings`, das den
  Build-Cache verwirft. Der Bauende prüft die Schreibweise gegen die
  Cargo-Doku der aktuellen stable.
- **types:** `cargo check --all-targets`.
- **test:** `cargo test`; `measuring`: `cargo llvm-cov --lcov
  --output-path {coverprofile}` (führt die Tests selbst aus; `cargo-llvm-cov`
  legt kein Verzeichnis an, das `.data` von `{coverdata}` gibt es nicht).
- **coverage:** `{loomux} check linecov --report {coverprofile}`,
  `after = "test"`; das Profil `rust` in `linecov/tools.toml` erhält den
  Berichtspfad.
- **graph**, **rules:** dieselben Befehle wie die anderen Stacks.
- **Ausschluss mit Begründung:** die Block-Marker aus Entscheidung 11, mit
  Test im Profil-Fixture `linecov/testdata/tools/rust` (ein mehrzeiliger
  Rumpf zwischen START und STOP, ein START ohne Grund, ein START ohne
  STOP).
- **Testcode in der Coverage:** Inline-Testmodule laufen bei `cargo test`
  selbst und sind damit gedeckt. Es gibt keinen eigenen Ausschluss und
  keine Kopplung von `linecov` an den Graphen. *(Entscheidung des Planers;
  eine ungedeckte Hilfsverzweigung im Testcode fällt wie jede andere auf.)*
- **Sperren:** clippy, check und test teilen `target/`; Cargo sperrt selbst.
  Der Bauende prüft, ob die Lanes eine Sperre wie bei C#
  (`plan.go:592-596`) brauchen, oder ob `after =` reicht.

**Aufgabe 4.2 — Plan-Stellen**
- `carrierRank` (`plan.go:270`): `rust` hinten anhängen.
- `IsSource` (`plan.go:308-310`), `rules/files.go contentExts`: `.rs`.
- Die Tests, die `rust` als Beispiel für einen nicht gelisteten Stack
  nutzen (`plan_test.go:1700-1740`, `rules_lane_test.go:92`,
  `effective_test.go:278`), bekommen einen anderen nicht gelisteten Stack
  (`shell` oder `sql`). Sie werden umgezogen, nicht gelöscht.
- `detect/signals.go:95`: `Cargo.toml` bleibt; `owns` wie bei C#/C++, falls
  die Regel dort gleich greift (Bauender).

**Aufgabe 4.3 — Werkzeuge**
- Keine `{tool:…}`-Pins für Rust (die gibt es nur für uv-Werkzeuge).
  Fehlt `cargo-llvm-cov`, meldet die Lane `missing-tool` wie jede andere.
  `cargo`, `clippy`, `rustfmt` und `llvm-tools-preview` kommen aus rustup.
- **Wiki/Doku:** `docs/{en,de}/support.md` Zeile `rust`,
  `configuration.md` (Preset-Tabelle, `tests_when`-Beispiel),
  `cli-reference.md` (linecov-Werkzeugliste, Ausschluss auf stable),
  `hooks.md` (Zeile `rust`), `log.md`.

### Phase 5 — Kompatibilität

**Aufgabe 5.1 — Einträge und Fixtures**
- `runtimeOf`: `rust` → `rust`.
- `compat.toml`:
  - `rust-cargo` auf `1.100`, `kinds = ["lint", "types", "test",
    "coverage", "graph"]`; Fixture: ein Workspace mit zwei Crates (Lib + Bin),
    Inline-Tests, ein Integrationstest unter `tests/`, ein Ausschluss mit
    Begründung;
  - `rust-syntax-2021`, `rust-syntax-2024`, `kinds = ["graph"]`, mit den
    Formen ihrer Edition (2024: `let`-Ketten, `gen`-Blöcke soweit die
    Grammatik sie kennt, `unsafe extern`);
  - `godot4-rust` aus Aufgabe 3.3 zweimal: als Rust-Eintrag mit
    `kinds = ["graph"]`, und als GDScript-Eintrag auf der neuesten
    Godot-Version mit `kinds = ["test"]` (die Suite baut die Extension und
    läuft headless). Der GDScript-Job braucht dafür zusätzlich die
    Rust-Toolchain; der Bauende legt fest, in welchem CI-Job das läuft.
- Scout: `godot-rust/gdext` als Bibliotheksquelle neben `gdUnit4`
  (`scout.go:104`), damit eine neue `godot`-Version des Crates ein Befund
  wird.
- Für bekannte Grammatik-Lücken (tree-sitter-rust #250: `safe` in
  `unsafe extern`) ein Test in `layer1_test.go`, der zeigt, was trotzdem
  ankommt, wie für C# 15 und C++23.

**Aufgabe 5.2 — Vorbereitung und CI**
- `run.go`: die Version eines Eintrags wählt die Toolchain
  (`RUSTUP_TOOLCHAIN` = Version).
- `ci/compat-setup.sh`: Fall `rust` — rustup installiert die Toolchain mit
  `clippy`, `rustfmt`, `llvm-tools-preview`; `cargo-llvm-cov` exakt gepinnt
  (heute 0.9.1).
- `.github/workflows/compat.yml`: Job `rust` über `compat-run.yml`, in die
  Liste des Pflichtjobs. Eine Toolchain-Action, falls nötig, per Commit-SHA
  gepinnt (`dtolnay/rust-toolchain` hat keine Tags;
  `actions-rust-lang/setup-rust-toolchain` v2.0.1 hat welche). Der Bauende
  wählt und begründet.

**Aufgabe 5.3 — Scout**
- Laufzeitquelle: die stable-Version aus einer maschinenlesbaren Quelle des
  Rust-Projekts (Kanal-Manifest `channel-rust-stable.toml` oder die
  GitHub-Releases von `rust-lang/rust`); Vorabversion = beta.
- EOL: endoflife.date `rust`. Es hinkt nach (am 2026-11-14 fehlte 1.100);
  der Scout darf eine neuere stable aus der Laufzeitquelle nicht als
  „unbekannt“ werten.
- Jede neue stable ist ein Scout-Befund (alle sechs Wochen). **Alle
  Rust-Einträge**, auch die nur-Graph-Einträge `rust-syntax-*` und
  `godot4-rust`, tragen dieselbe Version und ziehen gemeinsam um. Sonst
  meldet `scoutEOL` (`scout.go:312-319`) sie nach sechs Wochen als beendet,
  und `Scout` schlägt sie nie zum Umzug vor (`scout.go:248-250`).
- Eine Version, die neuer ist als der neueste Zyklus, den endoflife.date
  kennt, überspringt `scoutEOL` für Rust, statt den Lauf scheitern zu
  lassen (`scout.go:337-339`).
  *(Beides Entscheidung des Planers.)*
- **Wiki/Doku:** `support.md` (geprüfte Kombinationen), README-Roadmap:
  Rust fällt aus der Zeile „remaining five stacks“, `log.md`.

### Phase 6 — Abnahme

- `go test ./...` grün, 100 % je Funktion.
- Der Graph dieses Repos ist gleich bis auf den Stempel (kein Rust außerhalb
  von `testdata/`).
- An ripgrep, tokio und gdext (die Repos der Messung): `graph build` ohne
  Abbruch, `graph stats` nennt Rust-Knoten und aufgelöste Kanten, die Zeit
  liegt im Rahmen der anderen Sprachen. Kennzahlen (Anteil aufgelöster
  Aufrufe, Anteil `inferred`) in die Wiki-Seite.
- An godot-rust/demo-projects: GDScript → Rust-Klasse sichtbar.
- Unter Windows: `check godot-suite` baut und testet, während der
  Godot-Editor das Projekt offen hat.
- `cargo-llvm-cov` schreibt womöglich absolute `SF:`-Pfade; ein Test im
  lcov-Leser zeigt, dass sie auf Repo-Pfade abgebildet werden.
- Am Projekt des Menschen, sobald es existiert: `loomux check` mit allen
  Lanes grün.
- `CHANGELOG` über den PR (Skill `release-pr`).

## Bekannte Lücken (kommen in „Was offen“ der Wiki-Seite)

- Makro-erzeugter Code fehlt; Aufrufe in Makro-Argumenten sind geraten.
- Kein Trait-Dispatch, keine Typinferenz.
- Ein Signal oder eine Methode aus einer Basisklasse wird nicht gefunden
  (wie bei C++).
- `#[var]`/`#[export]`-Eigenschaften ergeben keine Kante.
- Methoden in `#[godot_api] impl INode for X` (`ready`, `process`) ruft die
  Engine; sie haben im Graphen keinen Aufrufer.

## Dem Bauenden überlassen

- Paketaufteilung im Extraktor (`rust.go`, `edges.go`, `attrs.go`).
- Die Reihenfolge der Aufgaben innerhalb einer Phase.
- Fixture-Inhalte im Einzelnen.

## Fragen an den Menschen (beantwortet am 2026-11-14)

- **Ausschluss auf stable:** Blockkommentare `LCOV_EXCL_START`/`STOP`
  (Entscheidung 11). Verworfen: Nightly nur für den Coverage-Lauf
  (Rust-üblich, aber eine Nightly-Toolchain, die brechen kann), und ein
  Rust-Parser in `linecov` (der meiste Code).
- **GDScript-Tests gegen Rust-Klassen:** jetzt mit rein (Entscheidung 10,
  Aufgabe 3.4).

## Entscheidungen des Planers zum Gegenlesen

- `Cargo.toml` macht den Graphen nicht veraltet (wie `go.mod`).
- Ausgelagerte Testmodule über den Dateinamen `tests.rs`/`test.rs`.
- Kein Coverage-Ausschluss für Inline-Testcode.
- Alle Rust-Compat-Einträge ziehen gemeinsam um; `scoutEOL` überspringt
  Versionen, die endoflife.date noch nicht kennt.
- Die Extension baut `check godot-suite` selbst, statt einer Ordnung über
  Stacks.
- Kopfregel für ERROR-Knoten; Makro-Aufrufe als `inferred`.

## Notiz vom 2026-11-15: Abnahme der Windows-DLL-Sperre

Die Annahme im Abschnitt „Windows“ (`reloadable = true` lasse Godot eine
Kopie laden, die das Original für Cargo freigibt) ist durch die Abnahme
überholt; der alte Satz bleibt zur Nachvollziehbarkeit stehen. Gemessen
unter Windows 11 mit dem Headless-Editor (`--headless --editor`) und einem
Headless-Spiel (`--headless --path`) auf Godot 4.7.1 (Mono-Konsolenbuild),
godot-rust 0.5.5, gdUnit4 6.2.2: Der Headless-Editor lud eine Kopie
(`~demo.dll`), `cargo build` und `check godot-suite` überschrieben
`demo.dll` mit `reloadable = true` wie mit `false`; ein Headless-Spiel hielt
`demo.dll` offen, `cargo build` scheiterte mit `os error 5`, ebenfalls bei
beiden Werten. Nicht gemessen: Editor mit Fenster, andere Godot-Versionen,
Hot Reload. Siehe `docs/{de,en}/stacks/gdscript.md`.

## Notiz vom 2026-11-15: Aufgabe 3.4 nach dem Gegenlesen

Der Satz in Aufgabe 3.4, der Leser treffe eine „Wahl des Plattformschlüssels“,
ist durch Entscheidung 10 überholt und bleibt zur Nachvollziehbarkeit stehen:
Es wird kein Schlüssel gewählt. Jeder `[libraries]`-Pfad, der in
`target/<profil>/` zeigt, baut sein Profil einmal für den Host; ein Eintrag,
dessen Schlüssel den Feature-Tag `release` oder `template_release` trägt, wird
übergangen, weil Suite und Import die Debug- oder Editor-Bibliothek laden.
Ebenfalls berichtigt: Die Suche nach dem Crate endet an der Repository-Wurzel
(ein `target/` oder Crate darüber wird nicht gebaut und gemeldet), und ein
gesetztes `CARGO_TARGET_DIR` bricht die Suche nicht ab, sondern bekommt eine
Notiz, weil Cargo dann dorthin baut und Godot die alte Bibliothek lädt; ein
`build.target-dir` in `.cargo/config.toml` bleibt eine Lücke ohne Notiz. Das
`on_file` der Lane `lint` (`cargo fmt --check -- {file}`) prüft das Crate oder
den Workspace, nicht nur die Datei.
