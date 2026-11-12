# JavaScript, JSX, TypeScript und TSX im Code-Graphen, in der Prüfkette und in der Kompatibilitätsliste — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** JS, JSX, TS und TSX stehen in Graph, Prüfkette und Kompatibilitätsliste auf der Stufe von Python, für Node 22, 24 und 26; 27 läuft als Vorab-Probe.

**Architecture:**
- Ein Extraktor-Paket `ecmascript` mit zwei Sprachen (`javascript`, `typescript`) und ein Resolver `resolve/ecmascript.go`.
- Der Resolver gruppiert nach Familie: JS und TS werden gemeinsam aufgelöst.
- Die Aliase aus `tsconfig.json`/`jsconfig.json` kommen als `resolve.Module` herein, wie `go.mod`.
- Der Stack `typescript` gibt es schon. Er bekommt:
  - `graph` und `rules`
  - die Erkennung reiner JS-Projekte
  - Wächter auf `node_modules/.bin/<tool>`
  - eine Coverage über linecov
- Die Laufzeit `node` kommt neu in den Compat, mit `nodejs.org/dist/index.json` als Release-Quelle.
- Die Web-Brücken liefert dieser Plan nur als Übergaben und Angebote der JS-Seite. Die HTML/CSS-Gegenseite folgt im Web-Plan (`2026-11-11-web-design.md`).

**Tech Stack:** Go, gotreesitter v0.57.0 (`grammars/javascript`, `grammars/typescript`, `grammars/tsx`); in den Fixtures Node 22/24/26, ESLint 10.14.0, typescript-eslint 8.72.2, TypeScript 6.0.3 und 7.0.2, Vitest 5.1.0, @vitest/coverage-v8 5.1.0, Vite 8.3.8.

**Spec:** `docs/.superpowers/specs/2026-11-11-ecmascript-design.md` (Programm: `2026-11-10-graph-sprachen-programm.md`, Pins: `2026-11-10-graph-sprachen-recherche.md`)

## Global Constraints

- **Wie im Java- und PHP-Plan:**
  - TDD, 100 % Coverage je Funktion (sonst `//coverage:exempt <Grund>`), kein `init()`
  - Code englisch, Doku deutsch und unter `docs/en/` englisch
  - keine Historie im Code
  - Conventional Commits ohne `Co-Authored-By`; nie pushen. Der Hook ist das Tor.
- **Grunddaten:**
  - Sprachen `javascript` (`.js`, `.mjs`, `.cjs`, `.jsx`) und `typescript` (`.ts`, `.mts`, `.cts`, `.tsx`)
  - Versionen `javascript/1` und `typescript/1`
  - Familie `ecmascript`
- **Pins exakt** (am 2026-11-12 nachgeschlagen mit `npm view`, `gh api` und endoflife.date):
  - ESLint 10.14.0
  - typescript-eslint 8.72.2 (Peer: `typescript >=4.8.4 <6.1.0`)
  - TypeScript 6.0.3 (neueste 6.0.x) und 7.0.2 (`latest`)
  - Vitest 5.1.0 und @vitest/coverage-v8 5.1.0 (Peer `vitest: 5.1.0` exakt)
  - Vite 8.3.8: ein Peer von Vitest, also im Fixture selbst gepinnt
  - `actions/setup-node` v7.1.0 = `949feb2413d6458794dcd2491c4babbbce0c15c1`, derselbe Pin wie in `dsh-loomux.yml`
  - Node-Linien: 22 (EOL 2027-04-30), 24 (2028-04-30), 26 (2029-04-30); 27 ist Alpha seit 2026-10-28
- **Abweichungen von der Spec, im Ledger festgehalten.** Seit 2026-11-12 stehen sie als Entscheidung in der Spec selbst:
  - TypeScript 6.0.x ist 6.0.3.
  - Vite und @vitest/coverage-v8 fehlen in der Spec.
  - Node-Release-Plan: Ab Node 27 wird jede Linie LTS und durchläuft vorher eine Alpha-Phase (nodejs/Release README). Die Regel „ungerade Linien zählen nicht“ entfällt; jede Linie unter 26 ist ohnehin am Ende. Die Vorab-Probe ist das Alpha der nächsten Linie aus `dist/index.json`, nicht RC oder nightly.
  - Das Repo hat `.js` auch außerhalb von `testdata/`: Unter `dsh-loomux/` liegen 12 `.js`-Dateien, die neu in den Graphen kommen. Das kommt in den Changelog.

## Review Focus

1. **`import "./a.js"` in einer TS-Datei, neben der nur `a.ts` liegt.** Die
   Kante führt auf `a.ts`. Liegen `a.js` und `a.ts` nebeneinander, gewinnt
   `a.js` (wie geschrieben). Test in Task 3.
2. **Zyklus aus `export * from`** (a → b → a) und ein Re-Export, der den
   Namen nicht hat. Es gibt keine Endlosschleife und keine Kante. Test in Task 3.
3. **Verschachteltes Paket mit eigener `tsconfig.json`**, darin Kommentare,
   nachgestellte Kommas und `paths` mit `*`. Die nächstgelegene Konfiguration
   gilt, die der Wurzel nicht. Test in Task 2.
4. **`.js` mit JSX-Attribut `class=`** (bekannte Lücke). Der Rest der Datei
   kommt an, und Knoten nach der Fehlerstelle bleiben. Test in Task 1.
5. **Eine Minor-RC einer gelisteten Node-Linie** (`26.4.0-rc.0`). Sie gibt
   keinen Scout-Fund, weil die Einträge nur die Major nennen. Test in Task 5.

## Grenze zum Web-Plan

- **Vertrag `<script src>`:** Der Web-Plan übernimmt ihn wörtlich; Task 3 baut die Leseseite.

  ```go
  // One per <script src> of an HTML page, in a Result with Language "html".
  extract.Import{Kind: "script", Path: src} // src exactly as written in the attribute
  ```

  - `scriptPages` liest `src` als URL (`url.Parse`), so wie der Browser lädt:
    - Query und Fragment fallen weg (`app.js?v=3#x` ist `app.js`).
    - Prozent-Escapes werden dekodiert (`a%20b.js` ist `a b.js`).
    - Ein ungültiges Escape (`%zz`) gibt keine Seite.
  - Ein `src` mit Schema (`https:`, `data:`) oder mit `//` am Anfang nennt keine Datei und fällt weg.
  - `/x.js` gilt relativ zur Repo-Wurzel, ohne den führenden `/`.
  - Alles andere gilt relativ zum Verzeichnis der Seite (`path.Join`).
  - Eine Rohkante `imports` je Skript schreibt der HTML-Extraktor selbst. Aufgelöst wird sie im Web-Resolver, nicht hier.
- **Lieferumfang dieses Plans:**
  - die Namespaces und ihre Regeln
  - die JS-Übergaben und -Angebote
  - `scriptPages`
- **Lieferumfang des Web-Plans:**
  - der HTML- und der CSS-Extraktor
  - der Web-Resolver
  - die Prüfung der Regel-Spalten durch blast-audit
- **Scope für den Web-Plan:** `classList.*`, `getElementsByClassName` und `querySelector*(".a")` in einem klassischen Skript haben hier keinen Scope.
  - Das Skript importiert keine Stylesheets, und ohne Scope gibt `css.class` keine Kante.
  - Der Web-Plan setzt als Scope die Stylesheets der Seiten, die das Skript einbinden. `scriptPages` liefert dafür die Seiten.
- **Zurückgestellt, wie bei `go.mod`:** Eine geänderte `tsconfig.json` macht den Graphen für `graph-fresh` nicht veraltet, weil die Frische-Probe nur Quelldateien sieht. Das kommt ins Ledger und wird hier nicht gebaut.

---

### Task 1: Extraktor `ecmascript` und Registrierung

**Files:**
- Create: `internal/code/extract/ecmascript/` mit `ecmascript.go`, `edges.go`, `web.go` und den Tests (`ecmascript_test.go`, `internal_test.go`, `lines_test.go`), gebaut wie `extract/php`
- Modify:
  - `extract/extract.go`:
    - Namespaces `NamespaceHTMLID` (`html.id`), `NamespaceCSSClass` (`css.class`), `NamespaceCSSID` (`css.id`), `NamespaceJSFunction` (`js.function`) und `Namespaces`
    - Doku von `Import.Kind`, `Import.Name`/`Alias` und `RawEdge.Member` für JS
  - `resolve/bridge.go`: die vier Regeln in `builtinRules`, übernommen aus der Tabelle der Web-Spec. `bridge_internal_test` verlangt eine Regel je Namespace.
  - `extract/all` mit `all_test.go` und `sourceset` (Endungen und Kommentar)
  - `blast.IsTestPath`
  - `blast.isFileQuery`: `.mjs`, `.cjs`, `.mts`, `.cts`
  - `repomap.langFromPath`: alle acht Endungen
  - `internal/notices/NOTICE.md` über `dev notices`
  - `corpus.pruneDirs` bleibt: `node_modules` und `dist` stehen drin, `.next` fällt als Punktordner heraus. Damit ist die offene Frage der Spec erledigt.

**Interfaces:**
- **Produces:**
  - `ecmascript.JavaScript` und `ecmascript.TypeScript`, beide `extract.Language`, mit den Grammatiken nach der Tabelle der Spec. `.jsx` liest die tsx-Grammatik, `.js` die JavaScript-Grammatik.
  - Für eine Datei auf `.d.ts` gibt `File` nur den Dateiknoten zurück, ohne Symbole und ohne Rohkanten. Sie bleibt also im Graphen, und der Resolver erkennt sie in der Probe-Folge.
- **Knoten und IDs:**
  - `rel#f`, `rel#C`, `rel#C.m` mit `Owner` `C`, den Konstruktor als `rel#C.constructor`
  - Namespace-Funktionen als `rel#X.f`
  - Für `interface` die Art `interface`, für `type` die Art `type`, für `enum` die Art `class`
  - Ein anonymer `export default function`/`class` und `module.exports = function () {}` heißen `default`.
  - `Exported` gilt nach der Spec, auch für einen in `module.exports`/`exports.f` genannten Namen.
  - `exports.f = function …`/`=> …` und `module.exports.f = …` ergeben einen Knoten `f` der Art `function`.
- **Imports:** ein `extract.Import` je Bindung, statt eines neuen Feldes.
  - `Name` ist der Name in der Zieldatei: `default`, `*` für einen Namensraum oder der Name selbst.
  - `Alias` ist der lokale Name.
  - `Kind` ist eines von:
    - `""`: statisches `import`
    - `require`: `require("x")`; `const m = require` ergibt `Name "*"`, `const {f: g} = require` ergibt `Name "f", Alias "g"`
    - `dynamic`: `import("x")`, ohne Bindung
    - `reexport`: `export {a as b} from "x"` ergibt `Name "a", Alias "b"`; `export * from "x"` ergibt `Name "*"`; `export * as ns from` ergibt `Name "*", Alias "ns"`
    - `export`: eine Export-Liste ohne Pfad, also `export {a as b}`, `export default ident` (`Alias "default"`) oder `module.exports = {f, g}`
  - Ein `import "./x.css"` ohne Bindung ist `Import{Path}` allein.
  - Je Import gibt es eine `imports`-Rohkante in derselben Reihenfolge, außer für `Kind "export"`, so wie PHP es hält.
- **Rohkanten:**
  - `extends`/`implements` mit `Name` und, bei `ns.Base`, mit `Receiver`
  - `calls`:
    - `f()` setzt `Name`.
    - `ns.f()` setzt `Name` und `Receiver "ns"`.
    - `this.m()` und `super.m()` setzen `Receiver "this"` bzw. `"super"`.
    - `new X()` setzt `Member "new"`.
  - Ein Aufruf gehört zur innersten Funktion oder Methode mit Knoten, sonst zur Datei.
- **Web:**
  - **Übergaben, die erst der Resolver scoped:** `getElementById("x")` und `querySelector*("#x")` werden zur Rohkante `references` mit `Member "html.id"` und `Name "x"`, so wie C++ `godot.method` kodiert. `#x` gibt zusätzlich eine zweite Rohkante mit `Member "css.id"`.
  - **Übergaben mit festem Scope:** `.a` in `querySelector*`, `getElementsByClassName("a b")`, `classList.add/remove/toggle/contains("a")` und JSX `className="a b"` werden zu `Result.Bridges` mit `css.class`, Relation `references`, eine je Klasse.
    - Der Scope sind die per `import` genannten `.css`/`.scss`-Pfade, verbunden mit dem Verzeichnis der Datei.
    - Ohne solche Pfade bleibt der Scope leer, und die Regel gibt dann keine Kante.
  - **Angebote:** JSX `id="x"` wird zu `Result.Exports{html.id, x, Target: <Komponente>}`.

- [ ] **Step 1: Failing Tests:**
  - `TestLanguagesDescribeThemselves`: Namen, Versionen und Endungen beider Sprachen
  - Knoten jeder Art aus der Spec, auch hinter `export` und `export default`, `const f = () => …`, `namespace X {}` und den anonymen `default`
  - `Exported` in allen Formen der Spec
  - Imports in jeder Form oben mit `Name`, `Alias` und `Kind`, dazu die Rohkanten mit Zeile
  - Rohkanten aller Aufrufformen und der Basen
  - Web: jede Übergabe und jedes Angebot, ein `querySelector("div .a")` gibt nichts, eine Liste `"#x, .a"` gibt beides
  - `.d.ts` hat nur den Dateiknoten
  - `.jsx` mit `class=` parst fehlerfrei, `.js` mit `class=` behält seine Knoten außerhalb der Fehlerstelle (Review Focus 4)
  - Grammatik-Probe der Spec (Import-Attribute, Flag `v`, `using`, Enum, abstrakte Klasse, `#private`) mit `ParseErrors == 0`
  - nil-Grammatik wie bei PHP
  - Registrierung:
    - `all_test`-Pins
    - `IsTestPath` mit `a.test.ts`, `a.spec.jsx`, `__tests__/a.js`, `test/a.mjs`, Negativfälle `latest.js`, `contest.ts` und `src/test-utils.ts`
    - `repomap` für `.mts` und `.cjs`
    - `TestNamespacesListsEveryConstant`
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implementieren.
  - Die Knotentypen jeder Grammatik zuerst messen, mit einem Dump-Test, der vor dem Commit gelöscht wird. Die gemessenen Typen kommen in den Paketkommentar, wie bei PHP.
  - JS und TS teilen den Code. Die Sprache entscheidet nur über `Language`, `Version` und die Grammatik je Endung.
- [ ] **Step 4:** PASS mit 100 %. Probelauf auf vite und express im Scratchpad-Korpus: kein Absturz, ParseErrors wie in der Parse-Prüfung des Programms.
- [ ] **Step 5: Commit** `feat(graph): read JavaScript and TypeScript files in the code graph`.

### Task 2: Familien im Resolver und `tsconfig` als `Module`

**Files:**
- Modify:
  - `resolve/resolve.go`: `familyOf`, Gruppierung, `edgesOf`
  - `resolve/bridge.go`: die Regel gegen Selbstbeantwortung
  - `resolve/gomod.go`: `Module`, `Modules`, `ModulesFrom` und die Go-Seite von `importDir`
  - `query/build.go`: `goModPaths` wird `moduleFiles`
  - `verify/rules/edges.go` und `verify/rules/files.go` (`wantsContent`)
- Test: `resolve/resolve_test.go`, `resolve/gomod_test.go`, `resolve/bridge_internal_test.go`, `query/build_test.go`, `verify/rules/files_test.go`

**Interfaces:**
- **Produces:**
  - `func familyOf(lang string) string`: `"javascript"` und `"typescript"` ergeben `"ecmascript"`, jede andere Sprache sich selbst.
  - `GraphWith` gruppiert nach `familyOf` und hält dabei die Ankunftsreihenfolge der Dateien (Kommentar zu `byLanguage`). Es löst die Familien in sortierter Folge auf.
  - `Meta.Languages` bleibt die sortierte Liste der einzelnen Sprachen.
  - `edgesOf` verzweigt auf die Familie. `case "ecmascript"` folgt in Task 3; bis dahin behält die Familie nur ihre `contains`-Kanten.
  - `bridgeEdges` vergleicht `familyOf(o.lang) == familyOf(h.lang)` statt der Sprachen.
  - Die Felder von `Module`:

    ```go
    type Module struct {
        Dir  string
        Path string    // go.mod: the module path; "" for a tsconfig
        TS   *TSConfig // nil for a go.mod
    }
    type TSConfig struct {
        BaseURL string              // relative to Dir, as written; "" without one
        Paths   map[string][]string // compilerOptions.paths as written
    }
    ```

  - `ModulesFrom` nimmt zusätzlich `tsconfig.json` und `jsconfig.json`. Liegen beide in einem Verzeichnis, gewinnt `tsconfig.json`.
  - Das JSON wird mit `encoding/json` gelesen, nachdem `stripJSONC(string) string` Kommentare und nachgestellte Kommas außerhalb von Strings entfernt hat. Eine neue Abhängigkeit gibt es nicht.
  - Eine Konfiguration ohne `compilerOptions.paths` und ohne `baseUrl` ergibt kein Modul. Eine unlesbare wird übersprungen, statt den Build abzubrechen; Go kennt diesen Fall nicht.
  - `importDir` überspringt Module mit `TS != nil`.
  - `func nearestTSConfig(mods []Module, file string) *Module` gibt das Modul mit dem längsten `Dir` zurück, das `file` enthält.

- [ ] **Step 1: Failing Tests:**
  - Eine `.ts`- und eine `.js`-Datei landen in einer Gruppe, `Meta.Languages` ist `["javascript", "typescript"]`.
  - Ein Handover von `javascript` wird nicht von einem `typescript`-Angebot beantwortet, ein `cpp`-Handover weiter von `csharp`.
  - Die Byte-Gleichheit zweier Builds bleibt (bestehender Test).
  - `ModulesFrom` liest Review Focus 3 und `jsconfig.json`; eine Konfiguration ohne Aliase gibt nichts; Go-Module stehen weiter vorn, längster Pfad zuerst.
  - `build` und `rules.Edges` reichen eine `tsconfig.json` durch, `wantsContent("a/tsconfig.json")` ist wahr.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** PASS mit 100 %. Alle bestehenden Resolver-Tests bleiben unverändert grün.
- [ ] **Step 5: Commit** `refactor(graph): resolve languages by family and read tsconfig path aliases`.

### Task 3: Resolver `internal/code/resolve/ecmascript.go`

**Interfaces:**
- **Consumes:**
  - die Imports und Rohkanten aus Task 1
  - `familyOf`, `Module.TS` und `nearestTSConfig` aus Task 2
- **Produces:**
  - `func resolveECMAScript(files []extract.Result, mods []Module, pages map[string][]string) langOut`
    - `pages` ordnet einem Skriptpfad die HTML-Seiten zu, die es per `<script src>` einbinden.
    - Ein `case "ecmascript"` in `edgesOf` ruft die Funktion auf.
  - `func scriptPages(files []extract.Result) map[string][]string` in `ecmascript.go` ordnet jedem Skriptpfad die Seiten zu, sortiert und ohne Duplikate.
    - Gelesen werden die Ergebnisse mit `Language "html"` nach dem Vertrag `<script src>` im Abschnitt „Grenze zum Web-Plan“.
    - Ein `extract.Import` mit anderem `Kind` wird übergangen.
- **Regeln:**
  - **Import-Ziele:**
    - Endungsfolge nach der Spec, `.js` → `.ts` nach `node16`
    - Aliase über `nearestTSConfig`: `paths` mit einem `*`, sonst `baseUrl`
    - Ein Treffer auf `.d.ts` gibt keine Kante.
    - Ein Paketname gibt keine Kante.
    - Eine Kante je Paar, mit der kleinsten Zeile.
  - **Bindungen und Re-Exporte:**
    - `Kind "reexport"` und `"export"` werden mit einer Besucht-Menge je (Datei, Name) verfolgt.
    - Ein `require`-Namensraum, der aufgerufen wird (`m()`), zielt auf `default`.
  - **Kanten:**
    - Aufrufe nach der Reihenfolge der Spec. `this`/`super` laufen über die Basen der Klasse, die selbst über Bindungen aufgelöst werden.
    - `Unresolved` gibt es nur für Basen mit lokalem oder relativ importiertem Namen.
  - **Web:**
    - **`html.id`/`css.id`:** Aus jeder Rohkante mit `Member "html.id"` oder `"css.id"` wird ein `handover` mit `lang` gleich der Sprache der Datei. Der Scope ist `pages[f.Path]`.
    - **Eingebettete Teile:** Ein eingebettetes Ergebnis erkennt man an einem Pfad ohne ECMAScript-Endung. Es bekommt `[f.Path]` als Scope.
    - **`js.function`-Angebote:** Eine Datei bietet ihre Funktionen auf Modulebene als `js.function <name>` an, wenn sie klassisch ist (kein `Import`, kein Symbol mit `Exported`) und entweder in `pages` steht oder ein eingebetteter Teil ist.
  - `GraphWith` übergibt `scriptPages(files)` an `edgesOf`, so wie es `paths` für Godot tut.

- [ ] **Step 1: Failing Tests** über `GraphWith` mit echten `ecmascript`-Ergebnissen:
  - **Imports:**
    - relativ mit jeder Endung und mit `/index.*`
    - Review Focus 1
    - `.d.ts` und Paket ohne Kante
    - Alias über `paths` und über `baseUrl`
  - **Bindungen:**
    - Standard, benannt mit Alias, `* as`
    - CommonJS in allen Formen der Spec
    - Re-Exporte und Review Focus 2
  - **Aufrufe:**
    - alle Formen der Spec, `new X()` auf `constructor`, sonst auf die Klasse
    - `this.m()` geerbt
  - **Basen:** `extends`/`implements`, `Unresolved` nur bei lokalem oder relativem Namen
  - **Familie:** JS ruft TS und umgekehrt.
  - **Web, mit handgebauten Ergebnissen:**
    - eine `Result{Language: "html"}` mit `Import{Kind: "script"}` und einem Element-Angebot `html.id`
    - eine `Result{Language: "css"}` mit einem `css.class`-Angebot
    - Erwartet werden:
      - die Kante von `getElementById` aufs Element, extracted im Scope
      - die `className`-Kante auf die importierte CSS-Regel
      - das `js.function`-Angebot nur für das klassische Skript, nicht für ein CommonJS-Modul
      - der Inline-Teil mit der eigenen Seite als Scope
  - **`scriptPages`:** `src` relativ, `/`-relativ, `https://…` und `//cdn…` ohne Seite, ein anderes `Kind` übergangen, zwei Seiten für ein Skript sortiert
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implementieren nach dem Aufbau von `resolve/php.go`.
- [ ] **Step 4:** PASS mit 100 %.
  - Probelauf auf vite und express.
  - Den Graphen dieses Repos vor und nach dem Task bauen (`loomux graph build`). Neu sind nur die Knoten und Kanten unter `dsh-loomux/`.
  - `loomux check rules --cached` und `loomux check blast-audit --cached --threshold 5` bleiben grün.
- [ ] **Step 5: Commit** `feat(graph): resolve JavaScript and TypeScript imports, bindings and calls`.

### Task 4: Prüfkette, Stack `typescript`

**Files:**
- `detect/signals.go`, `detect/detect.go`:
  - `signal.below` wird `[]string` (eine der Endungen genügt), `holds(root, area string, exts []string) bool`. Die Gradle-Kotlin-Zeilen werden zu `[]string{".java"}`.
  - Neue Zeile `{path: "package.json", below: ecmaExts, stacks: []string{"typescript"}}` mit den acht Endungen. Das ist das bestehende Muster „Signal plus Quellen darunter“; eine `javascript`-Variante wie `maven`/`gradle` wird es nicht, denn JS und TS teilen alle Werkzeuge und unterscheiden sich nur in `types`, das ein Wächter regelt.
  - `ambiguity` bekommt `unlessBelow []string`: Die Frage „package.json without tsconfig.json“ kommt nur noch für einen Bereich ohne JS- oder TS-Quelle, etwa Werkzeug-`package.json` in einem Python-Repo.
- `presets.toml`:
  - `[stack.typescript.graph]` und `[stack.typescript.rules]` wie bei PHP
  - unter `[extensions]` `.mjs`, `.cjs`, `.mts`, `.cts` auf `typescript`
  - **Wächter** (`when_found`, PHP-Muster) und Coverage über linecov:

    | Lane | `when_found` | Befehle |
    |---|---|---|
    | lint | `node_modules/.bin/eslint` | wie bisher |
    | variant `biome` lint | `node_modules/.bin/biome` (eine Variante erbt den Wächter nicht, `effective.go`) | wie bisher |
    | types | `tsconfig.json`, `node_modules/.bin/tsc` | wie bisher |
    | test | `node_modules/.bin/vitest` | `npx vitest run`; `measuring = "npx vitest run --coverage --coverage.reporter=lcov"` |
    | coverage | `node_modules/.bin/vitest`, `node_modules/@vitest/coverage-v8/package.json` | `{loomux} check linecov --report coverage/lcov.info`; `measure` wie `measuring`; `after = "test"` |

  - Der Kommentar über `[stack.typescript]` sagt in einem Satz, warum es Wächter gibt: Ohne TTY nimmt `npx` `--yes` an und lädt die neueste Version ungepinnt nach.
- `plan.go`: `carrierRank` endet auf `…, "php", "typescript"`; `IsSource` bekommt die acht Endungen
- `rules/files.go`: `contentExts` bekommt die acht Endungen
- `support.md` (en, de):
  - Erkennung über `package.json` mit JS- oder TS-Quellen
  - graph und rules für `typescript`
  - Wächter und linecov
  - Die Notiz „no version pinned“ wird durch die Node-Linien ersetzt.
  - Die Lücke `class=` in `.js` kommt dazu.
- `configuration.md` (en, de), wenn sie die Lanes je Stack aufzählt

**Gate dieses Repos** (am Code nachgeprüft: `detect` liest die Wurzel und eine Ebene darunter, `check precommit` erkennt zur Laufzeit):
- `dsh-loomux/` mit `package.json` und `index.js` wird damit ein Bereich mit dem Stack `typescript`.
- Kein einziger Lauf wird daraus:
  - lint: `dsh-loomux` hat keine Abhängigkeiten, also kein `node_modules/.bin/eslint`.
  - types: Es gibt keine `tsconfig.json`.
  - test und coverage: `package.json` nennt kein `vitest` (`tests_when`).
  - graph und rules gehören zur Wurzel, und Go trägt sie.
- „unavailable“ ist nicht rot (`CheckVerdict`), also bleibt das Ergebnis des Gates gleich; nur die Notizen kommen dazu.
- `ci/dsh-loomux.sh` mit `node --test` bleibt unberührt.

- [ ] **Step 1: Failing Tests:**
  - `detect`:
    - `package.json` mit `src/a.js` ergibt `typescript`, ohne Frage.
    - `package.json` allein ergibt keinen Stack und die Frage.
    - Gradle-Kotlin mit `.java` bleibt `java`.
  - Plan-Jobs `graph` und `rules` für `typescript`; `typescript` trägt den Graphen nur ohne einen höher gereihten Stack.
  - Wächter je Lane:
    - ohne `node_modules` alle vier Lanes `unavailable` mit „none found: node_modules/.bin/…“
    - `types` ohne `tsconfig.json` ebenso
    - `biome` mit eigenem Wächter
  - Coverage:
    - `check` mit `test` und `coverage` plant `measuring` und danach linecov auf `coverage/lcov.info`.
    - Der Eintrag `vitest` in `linecov/tools.toml` liest den Bericht. Ändert sich daran nichts, kommt das als Notiz in den Test.
  - **Gate-Test:** ein Baum wie `dsh-loomux/` (Wurzel mit `go.mod`, darunter `dsh-loomux/package.json` ohne Abhängigkeiten, `index.js`, `test/a.test.js`). `Plan` gibt keinen `typescript`-Job, der läuft; `CheckVerdict` bleibt 0.
  - `IsSource("a.cts")`
  - Spiegel-Tests (`presets_test`, `schema_test`, `rules_lane_test`, `detect`-Namensliste)
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** PASS.
  - `sh ci/gate.sh` auf diesem Repo: grün, wie vorher.
  - Die Ausgabe nennt die `dsh-loomux`-Lanes als „none found“ und keinen Lauf von `npx`.
- [ ] **Step 5: Commit** `feat(verify): check JavaScript projects and guard the TypeScript lanes on installed tools`.

### Task 5: Kompatibilität für Node

**Files:**
- `compat.go`: `runtimeOf["typescript"] = "node"`
- `compat.toml`:
  - Einträge der Spec-Tabelle: 22, 24 und 26 mit `ts-vitest-eslint`; 26 mit `ts7-vitest`; 22, 24 und 26 mit `js-syntax-<v>`
  - Ein Kommentarblock wie bei PHP: Die Pins stehen im `package-lock.json` des Fixtures, die Einträge nennen keine Werkzeuge.
- **Fixtures** unter `internal/compat/testdata/`:
  - **`ts-vitest-eslint`:**
    - `package.json` mit den exakten Pins aus den Constraints und TypeScript 6.0.3, dazu `package-lock.json`, erzeugt mit Node 22
    - `tsconfig.json`, `eslint.config.mjs` mit `tseslint.configs.recommended`, `vitest.config.ts`
    - `src/` mit Basis- und Kindklasse und einem `.js`-Modul, das ein `.ts`-Modul importiert
    - `test/` und `expect.toml`; die Tests decken `src/` zu 100 % ab, weil linecov jetzt prüft
  - **`ts7-vitest`:** wie oben mit TypeScript 7.0.2, ohne ESLint
  - **`js-syntax-22`, `-24`, `-26`:** je eine `syntax.mjs`; 26 zusätzlich mit `syntax.ts`
    - Die Syntax je Linie steht in der Spec. Vor dem Schreiben gegen die V8-Version der Linie prüfen.
    - Belegt 26 keine neue JS-Syntax, gilt die Ledger-Abweichung aus der Spec. `TestSyntaxFixtureNeedsItsVersion` prüft nur Go und bleibt unberührt.
- `run.go`:
  - `installNPM(dir string, e Entry) error` analog `installComposer`, mit einer Test-Naht `npm`
  - Aufruf: `npm ci --no-audit --no-fund`, wenn `e.Runtime == "node"` und `package-lock.json` im Fixture liegt
  - Ohne das wären die Lanes nach den Wächtern aus Task 4 „unavailable“, und der Eintrag würde nichts belegen.
- `scout.go`:
  - `nodeSource = "https://nodejs.org/dist/index.json"` in `runtimeSources["node"]`
  - Parser: `v26.13.1` ergibt `Release{"26.13.1", false}`, `v27.0.0-alpha.0.0.0` ergibt `Release{"27.0.0-alpha.0.0.0", true}`
  - `preRe` liest die Node-Alpha `N.0.0-alpha.…` als Major allein, so wie `-ea`
  - `scoutPre` bekommt den `g.parts == 2`-Wächter für `newMinor`, den `scoutGroup` schon hat
  - `eolProducts["node"] = "nodejs"`
  - Testkopie `testdata/node-releases.json`, gekürzt aus der echten Antwort
- **CI:**
  - `compat-setup.sh`: `go | dotnet | php | node) ;;`
  - `compat.yml`: setup-node auf dem SHA der Constraints, mit `node-version: ${{ steps.setup.outputs.install }}`
  - **Alpha in der Probe:** In der setup-node-Doku steht nur nightly, rc und v8-canary.
    - Erst prüfen, ob eine exakte Version `27.0.0-alpha.0.0.0` aus `dist` installiert wird.
    - Falls nicht, lädt `compat-setup.sh` im Zweig `node)` für eine Probe das Archiv aus `nodejs.org/dist/<v>/` und prüft es gegen `SHASUMS256.txt`.
    - Das Ergebnis kommt ins Ledger.
- `support.md` (en, de): die Kompatibilitätszeilen für Node

- [ ] **Step 1: Failing Tests:**
  - layer1, `TestEveryFixtureIsUsed`, `docs_test`
  - `run_test`: `npm ci` läuft für einen Node-Eintrag mit Lanes, nicht für einen reinen Graph-Eintrag
  - Scout:
    - Parser für beide Versionsformen
    - Einträge 22/24/26 mit der Testkopie ergeben einen Fund `pre` mit Version `27.0.0-alpha.0.0.0` und `Probe.Version "27"`
    - Review Focus 5
    - EOL über `nodejs`
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** PASS.
  - Die Graph-Einträge über `dev compat run` lokal laufen lassen.
  - Die Lanes von `ts-vitest-eslint` und `ts7-vitest` mit dem lokalen Node laufen lassen. Fehlen die Linien 22 und 24 lokal, ist CI der Beweis; das kommt ins Ledger.
- [ ] **Step 5: Commit** `feat(compat): verify TypeScript and JavaScript on Node 22, 24 and 26`.

### Task 6: Doku

**Files:**
- Wiki:
  - `topics/code-graph.md`: Sprachen, Familie, Bindungen, Aliase, Web-Übergaben
  - `log.md`
  - die Seiten, deren Quell-Pins sich ändern: `lint/wiki-sources` meldet sie, danach `loomux wiki repin`
- `cli-reference.md`, `configuration.md`, `architecture.md` (en, de)
- **READMEs:**
  - die Code-Graph-Zeile beider READMEs
  - Die Roadmap-Zeile „TypeScript/TSX in the code graph“ entfällt (AGENTS.md).
  - In „Compatibility matrix for the other stacks“ fällt TypeScript aus der Liste der offenen Stacks.

- [ ] **Commit** `docs: describe JavaScript and TypeScript in the code graph and the check chain`.
