# JavaScript, JSX, TypeScript und TSX im Code-Graphen

**Stand:** 2026-11-11 · Teilprojekt 5 von
`2026-11-10-graph-sprachen-programm.md`

## Ziel

JS, JSX, TS und TSX stehen in Graph, Prüfkette und Kompatibilitätsliste auf
der Stufe von Python, für Node 22, 24 und 26. JS und TS lösen sich
gegenseitig auf, wie in echten Projekten, die beides mischen.

## Nicht-Ziele

- **Keine Workspace-Pakete und kein `extends` in `tsconfig.json`.** Ein
  Paketname bleibt ohne Kante. Pfad-Aliase sind dagegen drin (siehe
  Resolver): `@/…` ist der Standard in Vite- und Next-Vorlagen.
- **Keine Typinferenz.** `obj.m()` bleibt ohne Kante, außer bei `this`
  oder einem importierten Namensraum.
- **Keine Vue- und Svelte-Dateien.** Ihre Stacks haben nur types-Lanes.
- **Keine `.d.ts`-Dateien.** Sie wiederholen Deklarationen, wie `.pyi` bei
  Python.

## Sprachen und Grammatiken

| Endung | Sprache (`Result.Language`) | Grammatik |
|---|---|---|
| `.js`, `.mjs`, `.cjs` | `javascript` | `grammars/javascript` |
| `.jsx` | `javascript` | `grammars/tsx` |
| `.ts`, `.mts`, `.cts` | `typescript` | `grammars/typescript` |
| `.tsx` | `typescript` | `grammars/tsx` |

- **JSX mit der tsx-Grammatik.** Die JavaScript-Grammatik bricht an
  JSX-Vorlagen ab (33 % der `.jsx` in vite, mit beiden Laufzeiten gleich).
  Die Ursache ist ein JSX-Attribut namens `class` (Solid, Preact);
  `className` (React) parst. Die tsx-Grammatik parst beides fehlerfrei.
  `.js` bleibt bei der JavaScript-Grammatik, die JSX mit `className` kann.
  `class=` in einer `.js`-Datei ist eine Lücke in `support.md`.
- **Ein Paket `internal/code/extract/ecmascript`** mit zwei
  `extract.Language`-Werten: `ecmascript.JavaScript` (Version
  `javascript/1@…`) und `ecmascript.TypeScript` (`typescript/1@…`).
- **Grammatik-Probe vom 2026-11-11:** Import-Attribute, das RegExp-Flag
  `v`, `using`, Namespaces, Enums, abstrakte Klassen und `#private` parsen
  ohne Fehler.

## Extraktor

- **Knoten:**
  - Funktionen und Klassen auf Modulebene, auch hinter `export` und
    `export default`
  - `const f = (…) => …` und `const f = function …` auf Modulebene (Art
    `function`)
  - Methoden in Klassen (Art `method`), Konstruktor `constructor`
  - TS: `interface` und `type` (Art `interface` bzw. `type`), `enum` (Art
    `class`), Funktionen in `namespace X {}` als `X.f`
- **`Exported`:** mit `export` oder in einer `export { … }`-Liste genannt.
- **Imports:**
  - `import … from "x"`, `export … from "x"`
  - `import("x")` mit Stringliteral, `require("x")` mit Stringliteral
  - Jede Bindung wird zu einem `extract.Import`, ein neues Feld gibt es
    nicht (Entscheidung des Plans):
    - `Name` ist der Name in der Zieldatei (`default`, `*`, der Name).
    - `Alias` ist der lokale Name.
    - `Kind` ist `""`, `require`, `dynamic`, `reexport` oder `export`.
      `export` ohne Pfad trägt Export-Listen und `module.exports`.
    - Dazu eine `imports`-Rohkante je Import mit Pfad.
  - `.d.ts` behält nur den Dateiknoten, damit der Resolver ihn erkennt.
  - Ein anonymer Standard-Export heißt `default`.
- **Rohkanten:**
  - `extends` und `implements`
  - `calls`:
    - `f()`
    - `ns.f()` (Receiver = Namensraum-Bindung)
    - `this.m()`, `super.m()`
    - `new X()` mit `Member "new"`
- **Web-Brücken** (Teilprojekt 6 baut die Gegenseite, dieses Teilprojekt
  liefert die Übergaben):
  - `document.getElementById("x")` übergibt `html.id x`. Der Extraktor
    schreibt eine Rohkante `references` mit `Member "html.id"` (wie C++
    `godot.method`); den Handover mit Scope baut der Resolver. `#x` in
    `querySelector*` gibt dazu eine zweite mit `Member "css.id"`.
  - Für `css.class`:
    - `querySelector`/`querySelectorAll` mit einem Literal, das nur aus
      `#x` oder `.a` besteht (oder aus einer Liste solcher) → `html.id`
      bzw. `css.class`
    - `getElementsByClassName("a b")`
    - `classList.add/remove/toggle/contains("a")`
  - In JSX: `className="a b"` übergibt `css.class` je Klasse,
    `id="x"` bietet `html.id x` an (Ziel ist die Komponente, die das JSX
    enthält).
  - **Hinweis vorab (Web-Plan, Task 5):** Auch jede `css.class`-Übergabe
    ist eine Rohkante `references` mit `Member "css.class"`, keine
    `Result.Bridges`. Den Scope von `css.class` und `css.id` bildet der
    Resolver, wie den von `html.id`.
  - **`js.function`-Angebote:** Funktionen auf oberster Ebene eines
    klassischen Skripts bieten `js.function <name>` an. Klassisch heißt:
    kein `import`/`export`, kein `require`, kein `module.exports`.
    - Die Angebote macht der Resolver, nur für Dateien, die eine
      HTML-Seite per `<script src>` einbindet.
    - Ein CommonJS-Modul bietet damit nichts an.
  - **Scope einer `css.class`- oder `css.id`-Übergabe** (JSX-`className`,
    `classList.*`, `getElementsByClassName`, `querySelector*`), ohne
    Duplikate in dieser Reihenfolge:
    - die CSS- und SCSS-Dateien, die das Modul importiert: relativ
      (`import "./x.css"`) oder über die `paths` der nächsten tsconfig
      (`import "@/x.css"`). Query und Fragment fallen weg
      (`./x.css?inline`).
    - für jede Seite, die das Skript einbindet, deren verlinkte
      Stylesheets und die Seite selbst (ihre `<style>`-Teile). Ein
      Inline-Skript hat die eigene Seite.
    - Ein leerer Scope gibt keine Kante.
    - Ein solcher Import gibt dazu eine `imports`-Kante auf das Stylesheet.
  - **Scope einer `getElementById`-Übergabe:** die HTML-Seiten, deren
    `<script src>` auf diese Datei zeigt. Den Scope setzt der Resolver, weil
    nur er die Seiten kennt. Ein Inline-Skript bekommt die eigene Seite.
  - **Vertrag mit Teilprojekt 6:** Der HTML-Extraktor schreibt jedes
    `<script src>` als `extract.Import{Kind: "script", Path: <src wie
    geschrieben>}` in ein Ergebnis mit `Language "html"`.
    - Ein `src` mit Schema (`https:`) oder mit `//` am Anfang gibt keine
      Seite.
    - `/x.js` gilt relativ zur Repo-Wurzel, alles andere relativ zum
      Verzeichnis der Seite.
    - Daraus baut `scriptPages` in diesem Teilprojekt die Zuordnung
      Skript → Seiten.
  - Die Namespaces `html.id`, `css.class`, `css.id` und `js.function`
    kommen in `extract.Namespaces`, ihre Regeln gleich mit, aus der Tabelle
    der Web-Spec. Der Test `bridge_internal_test` verlangt eine Regel je
    Namespace. Teilprojekt 6 darf die Spalten noch ändern.
  - **Geschlossen im Web-Plan (Task 5):** `classList.*`,
    `getElementsByClassName` und `querySelector(".a")` in einem
    klassischen Skript bekommen als Scope die Stylesheets der Seiten, die
    das Skript einbinden (siehe oben).

## Resolver `internal/code/resolve/ecmascript.go`

- **Familie:** `javascript` und `typescript` werden zusammen aufgelöst.
  - `GraphWith` gruppiert dafür nach Familie (`familyOf`). Gibt es keine
    Familie, ist die Sprache selbst die Familie. Das Meta listet weiter
    die einzelnen Sprachen.
  - Eigene Übergaben dürfen von der eigenen Familie nicht beantwortet
    werden; die Regel gegen Selbstbeantwortung prüft die Familie.
- **Import-Ziele:**
  - relativ (`./`, `../`) zum Verzeichnis der Datei
  - Die Endung wird probiert in der Reihenfolge wie geschrieben, `.ts`,
    `.tsx`, `.d.ts` (nur zur Erkennung, ohne Kante), `.js`, `.jsx`,
    `.mjs`, `.cjs`, dann `/index.*` in derselben Folge.
  - `.js` im Import einer TS-Datei findet auch die `.ts`-Datei (Regel
    `moduleResolution: node16`).
  - **Aliase** aus `compilerOptions.paths` und `baseUrl` in der
    `tsconfig.json` (oder `jsconfig.json`), die der Datei am nächsten
    liegt. Gelesen wird ohne die `extends`-Kette.
    - Den Inhalt reicht der Build als `Module` herein, wie `go.mod` für Go
      (`resolve/gomod.go`).
    - JSON mit Kommentaren und nachgestellten Kommas wird toleriert.
    - Zurückgestellt, wie bei `go.mod`: Eine geänderte `tsconfig.json`
      macht den Graphen für `graph-fresh` nicht veraltet. Die
      Frische-Probe sieht nur Quelldateien.
  - Paketnamen: keine Kante.
  - Eine Kante je Paar, kleinste Zeile.
- **CommonJS:**
  - `const m = require("./m")` bindet einen Namensraum, über den `m.f()`
    aufgelöst wird; `const { f } = require("./m")` bindet einen Namen.
  - Exportiert wird über `module.exports = { f, g }`,
    `module.exports = function …`, `exports.f = …` und
    `module.exports.f = …`.
- **Bindungen:**
  - Ein importierter Name zeigt auf das gleichnamige exportierte Symbol der
    Zieldatei; `default` auf das Standard-Export-Symbol.
  - Re-Exporte (`export { x } from`, `export * from`) werden zyklenfest
    verfolgt.
- **Kanten:**
  - `calls`:
    - `f()`: lokale Definition, dann die importierte Bindung
    - `ns.f()`: Namensraum-Import, dann das Export-Symbol
    - `this.m()`: Methode der Klasse und ihrer Basen
    - `super.m()`
    - `new X()`: `constructor`, sonst die Klasse
  - `extends` und `implements` über dieselben Bindungen. `Unresolved` gibt
    es nur für Basen, deren Name lokal oder relativ importiert ist; Pakete
    zählen nicht.

## Restliche Graph-Stellen

- **`blast.IsTestPath`:** `*.test.*`, `*.spec.*`, Segmente `__tests__`,
  `test`, `tests`.
- **`repomap`, `isFileQuery`:** alle Endungen oben.
- **`corpus.pruneDirs`:** bleibt. `node_modules` und `dist` stehen drin,
  `.next` überspringt die Walk als Punktordner.

## Prüfkette

- **Stack `typescript`** (gibt es) bekommt `graph` und `rules`. Pin-Check:
  - typescript-eslint 8.72 verträgt kein TypeScript 7. Ein Projekt mit
    TS 7 und ESLint ist Sache des Projekts.
  - Die Fixtures pinnen TypeScript 6.0.3.
- **Reine JS-Projekte:** `package.json` mit einer JS- oder TS-Quelle
  darunter macht den Stack `typescript` an.
  - Das nutzt das `below`-Muster der Gradle-Kotlin-Zeile. `below` wird dafür
    eine Liste von Endungen.
  - Die Frage „package.json ohne tsconfig.json“ bleibt nur für einen Bereich
    ohne solche Quellen.
  - Eine eigene `javascript`-Variante gibt es nicht, weil JS und TS dieselben
    Werkzeuge haben.
- **Wächter wie bei PHP** (`when_found`):
  - Jede Lane verlangt ihr Werkzeug unter `node_modules/.bin/`, die Lane
    `types` zusätzlich eine `tsconfig.json`.
  - Ohne `npm install` sind die Lanes „nicht verfügbar“. `npx` lüde sonst
    ohne TTY die neueste Version ungepinnt nach (npm-exec-Doku).
  - Im Gate dieses Repos wird `dsh-loomux/` damit zu einem Bereich mit dem
    Stack `typescript`. Alle seine Lanes bleiben nicht verfügbar: kein
    ESLint, keine `tsconfig.json`, kein Vitest. Das Gate läuft wie vorher.
- **Coverage über linecov**, wie C#, Java und PHP:
  - `measuring`/`measure` ist `npx vitest run --coverage --coverage.reporter=lcov`.
  - Danach folgt `{loomux} check linecov --report coverage/lcov.info`.
- **`carrierRank`:** `… php, typescript`.
- **`IsSource` und `contentExts`:** die Endungen oben.
- **`support.md`:** graph und rules für `typescript`.

## Kompatibilität

- `runtime = "node"`, `runtimeOf["typescript"] = "node"`.
- **Einträge:**

  | Version | Fixture | Arten |
  |---|---|---|
  | jede gepflegte Linie | `ts-vitest-eslint` (ESLint 10.14.0, typescript-eslint 8.72.2, TypeScript 6.0.3, Vitest 5.1.0, @vitest/coverage-v8 5.1.0, Vite 8.3.8) | lint, types, test, coverage, graph |
  | jüngste Linie | `ts7-vitest` (TypeScript 7.0.2, Vitest 5.1.0, @vitest/coverage-v8 5.1.0, Vite 8.3.8; kein ESLint, weil typescript-eslint TS 7 ausschließt) | types, test, coverage, graph |
  | jede gepflegte Linie | `js-syntax-<v>` | graph |

  - **„Gepflegt“** heißt, was endoflife.date `nodejs` als gepflegt führt.
    Am 2026-11-12 sind das 22, 24 und 26.
  - **Neuer Release-Plan von Node** (nodejs/Release, README):
    - Ab Node 27 durchläuft jede Linie erst eine Alpha-Phase und wird
      danach LTS.
    - Die Regel nach geraden und ungeraden Linien entfällt. Jede ungerade
      Linie unter 26 ist ohnehin am Ende.
    - 27 ist seit 2026-10-28 Alpha.
  - Jeder Lane-Lauf führt vorher `npm ci` auf dem `package-lock.json` des
    Fixtures aus.
  Bringt eine Linie keine neue Syntax, belegt ihr Fixture die Formen der
  Vorgängerin. Das ist eine bewusste Abweichung von „Syntax, die diese
  Version einführte“ und steht im Ledger.

  Vitest 5.1 verlangt Node ^22.12, ^24 oder ≥ 26; alle drei passen.
- **Syntax je Node-Linie:**
  - 22: Flag `v`, Import-Attribute
  - 24: zusätzlich RegExp-Modifikatoren, doppelte benannte Gruppen,
    `using`
  - 26: wie 24, plus die TS-Formen aus dem `.ts`-Teil des Fixtures
- **Scout:**
  - Releases aus `https://nodejs.org/dist/index.json`
  - **Vorab** ist das Alpha der nächsten Linie, wie es in `dist/index.json`
    steht (heute `v27.0.0-alpha.0.0.0`).
    - Eine Minor-RC einer gelisteten Linie zählt nicht, denn die Einträge
      nennen nur die Major.
    - `preRe` und `scoutPre` brauchen dafür eine Anpassung.
  - Supportende über endoflife.date `nodejs`
- **CI:**
  - `actions/setup-node` v7.1.0, gepinnt auf
    `949feb2413d6458794dcd2491c4babbbce0c15c1` wie in `dsh-loomux.yml`,
    mit der Version des Eintrags
  - `compat-setup.sh` mit `node) ;;`
  - Ob setup-node ein Alpha installieren kann, ist offen. Falls nicht, lädt
    die Probe das Alpha aus `nodejs.org/dist` und prüft es gegen
    `SHASUMS256.txt`.

## Tests und Abnahme

- Wie bei C#: Extraktor-, Resolver- und Spiegel-Tests.
- Probelauf auf vite und express: kein Absturz, ParseErrors wie in der
  Parse-Prüfung.
- **Dieses Repo hat `.js` auch außerhalb von `testdata/`.** Die
  ursprüngliche Annahme war falsch:
  - Unter `dsh-loomux/` liegen 12 `.js`-Dateien. Sie kommen neu in den
    Graphen; das ist gewollt und kommt in den Changelog.
  - `check rules` und `blast-audit` bleiben auf dem Repo grün.
