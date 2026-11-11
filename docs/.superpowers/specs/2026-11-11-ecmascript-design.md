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
  - Jeder wird zu einem `extract.Import`: der Pfad, die gebundenen Namen
    (Standard, benannt mit Alias, Namensraum `* as`) in einem
    dokumentierten Feld, dazu eine `imports`-Rohkante.
- **Rohkanten:**
  - `extends` und `implements`
  - `calls`:
    - `f()`
    - `ns.f()` (Receiver = Namensraum-Bindung)
    - `this.m()`, `super.m()`
    - `new X()` mit `Member "new"`
- **Web-Brücken** (Teilprojekt 6 baut die Gegenseite, dieses Teilprojekt
  liefert die Übergaben):
  - `document.getElementById("x")` übergibt `html.id x`.
  - Für `css.class`:
    - `querySelector`/`querySelectorAll` mit einem Literal, das nur aus
      `#x` oder `.a` besteht (oder aus einer Liste solcher) → `html.id`
      bzw. `css.class`
    - `getElementsByClassName("a b")`
    - `classList.add/remove/toggle/contains("a")`
  - In JSX: `className="a b"` übergibt `css.class` je Klasse,
    `id="x"` bietet `html.id x` an (Ziel ist die Komponente, die das JSX
    enthält).
  - **`js.function`-Angebote:** Funktionen auf oberster Ebene eines
    klassischen Skripts bieten `js.function <name>` an. Klassisch heißt:
    kein `import`/`export`, kein `require`, kein `module.exports`.
    - Die Angebote macht der Resolver, nur für Dateien, die eine
      HTML-Seite per `<script src>` einbindet.
    - Ein CommonJS-Modul bietet damit nichts an.
  - **Scope einer JSX-`className`-Übergabe:** die CSS- und SCSS-Dateien, die
    das Modul importiert (`import "./x.css"`).
  - **Scope einer `getElementById`-Übergabe:** die HTML-Seiten, deren
    `<script src>` auf diese Datei zeigt. Den Scope setzt der Resolver, weil
    nur er die Seiten kennt. Ein Inline-Skript bekommt die eigene Seite.
  - Die Namespaces `html.id`, `css.class`, `css.id` und `js.function`
    kommen in `extract.Namespaces`; die Regeln setzt Teilprojekt 6.

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
- **`corpus.pruneDirs`:** `node_modules` und `dist` stehen drin; dazu
  `.next`? Ein Punktordner wird ohnehin übersprungen.

## Prüfkette

- **Stack `typescript`** (gibt es) bekommt `graph` und `rules`. Die übrigen
  Lanes bleiben, mit einem Pin-Check:
  - typescript-eslint 8.72 verträgt kein TypeScript 7. Ein Projekt mit
    TS 7 und ESLint ist Sache des Projekts.
  - Die Fixtures pinnen TypeScript 6.0.x.
- **`carrierRank`:** `… php, typescript`.
- **`IsSource` und `contentExts`:** die Endungen oben.
- **`support.md`:** graph und rules für `typescript`.

## Kompatibilität

- `runtime = "node"`, `runtimeOf["typescript"] = "node"`.
- **Einträge:**

  | Version | Fixture | Arten |
  |---|---|---|
  | jede gepflegte Linie | `ts-vitest-eslint` (ESLint 10.14.0, typescript-eslint 8.72.2, TypeScript 6.0.x, Vitest 5.1.0) | lint, types, test, coverage, graph |
  | jüngste Linie | `ts7-vitest` (TypeScript 7.0.2, Vitest 5.1.0; kein ESLint, weil typescript-eslint TS 7 ausschließt) | types, test, coverage, graph |
  | jede gepflegte Linie | `js-syntax-<v>` | graph |

  „Gepflegt“ heißt, was endoflife.date `nodejs` als unterstützt führt, mit
  ungeraden Linien. Heute sind das 22, 24 und 26; der Plan prüft es nach.
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
  - Vorab ist die nächste gerade Linie, solange es nur RCs oder nightlies
    gibt; ungerade Linien zählen nicht als gepflegt.
  - Supportende über endoflife.date `nodejs`
- **CI:** `actions/setup-node` (SHA-gepinnt) mit der Version des Eintrags;
  `compat-setup.sh` mit `node) ;;`.

## Tests und Abnahme

- Wie bei C#: Extraktor-, Resolver- und Spiegel-Tests.
- Probelauf auf vite und express: kein Absturz, ParseErrors wie in der
  Parse-Prüfung.
- Dieses Repo hat `.js`-Dateien nur unter `testdata/`, sein Graph bleibt
  gleich. Prüfen, ob eine Datei außerhalb liegt, die dann neu im Graphen
  steht. Das ist dann gewollt und kommt in den Changelog.
