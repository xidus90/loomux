# HTML, CSS und SCSS im Code-Graphen, mit Brücken zu JS

**Stand:** 2026-11-11 · Teilprojekt 6 von
`2026-11-10-graph-sprachen-programm.md` · hängt an Teilprojekt 5
(ECMAScript)

## Ziel

HTML, CSS und SCSS kommen in den Graphen. Eine Seite erreicht über
Brücken ihre Stylesheets und Skripte, ihre Klassen die CSS-Regeln, und
ein Skript die Elemente der Seite. Die Prüfkette prüft HTML gegen den
Standard (html-validate) und CSS/SCSS mit stylelint.

## Nicht-Ziele

- **Kein Rendern, kein Kaskadenmodell.** Welche Regel am Ende greift,
  entscheidet der Graph nicht. Eine Klasse zeigt auf jede Regel, die sie
  nennt.
- **Keine indentierte Sass-Syntax** (`.sass`). Die Grammatik kennt sie
  nicht.
- **Keine Vorlagen-Sprachen** (EJS, Handlebars). Ihre Platzhalter
  erscheinen als Text oder ERROR, gelesen wird trotzdem, was steht.

## Extraktoren

**`html`** (Endungen `.html`, `.htm`; Version `html/1`)
- **Knoten:**
  - die Datei
  - jedes Element mit `id`, Art `element`, Name `#id`, Owner leer
  - Ein doppeltes `id` bekommt eine Ordinalzahl.
- **Imports** (Rohkante `imports`):
  - `<link rel="stylesheet" href>`
  - `<script src>`
  - `<a href>` nicht: ein Link ist Navigation, keine Abhängigkeit
- **Eingebettet** (`Result.Embedded`):
  - `<script>` ohne `src` ist ein JavaScript-Teil, mit `type="module"` ein
    Modul. Ein Teil mit `lang="ts"` oder `type="text/typescript"` ist
    TypeScript.
  - `<style>` ist ein CSS-Teil.
  - **Ein eingebettetes Ergebnis je Sprache und Datei, nicht je Block.**
    Alle `<script>`-Rümpfe einer Sprache werden zusammen geparst, der Rest
    der Datei ausgeblendet; die Offsets bleiben so gleich.
    - Die Vergabe der IDs ist dadurch gemeinsam: Zwei Inline-Skripte, die
      beide `function init()` definieren, ergeben `page.html#init` und
      `page.html#init~2`.
    - Ein Teil hat keinen eigenen Dateiknoten. Seine Definitionen hängen
      per `contains` am Dateiknoten der HTML-Datei.
    - Ein Test hält beides fest: eindeutige IDs und genau einen
      Dateiknoten.
  - Spans und Zeilen zeigen in die HTML-Datei.
- **Angebote und Übergaben:**
  - `id="x"` bietet `html.id x` an; Ziel ist der Elementknoten.
  - `class="a b"` übergibt je Klasse `css.class` mit Relation `references`
    von der Datei. Der Scope sind die verlinkten Stylesheets plus die
    eigenen `<style>`-Teile.
  - `onclick="go()"` und die anderen `on*`-Attribute übergeben
    `js.function go`, wenn der Wert mit einem einfachen Aufruf beginnt.
    Scope sind die eingebundenen Skripte und die eigenen Inline-Skripte.

**`css`** (`.css`, Version `css/1`) und **`scss`** (`.scss`, Version
`scss/1`) in einem Paket `internal/code/extract/stylesheet`:
- **Knoten:**
  - die Datei
  - jede Regel (`rule_set`), Art `rule`. Den Namen legt der Plan fest,
    nachdem er geprüft hat, wo Code IDs an `#` teilt oder `~N` abschneidet
    (CLI- und MCP-Argumente). Der Selektortext enthält Leerzeichen, `#` und
    `~`. Vorschlag: Name = Selektortext mit Leerraum zu einem Leerzeichen
    gefaltet, `~` ersetzt; die Signatur hält den Originaltext.
    Verschachtelte SCSS-Regeln heißen `Eltern › Kind`.
  - SCSS: `@mixin` (Art `mixin`), `@function` (Art `function`),
    Platzhalter `%x` (Art `rule`)
- **Imports:**
  - CSS: `@import "x.css"` und `@import url("x.css")`
  - SCSS: `@use`, `@forward`, `@import`
  - Partials werden über den Resolver aufgelöst: `_x.scss`, `x.scss` und
    `x/_index.scss`.
  - `sass:`-Module geben keine Kante.
- **Angebote** (Ziel = Regelknoten):
  - jede Klasse eines Selektors bietet `css.class <name>` an, jede ID
    `css.id <name>`
  - mehrere Regeln pro Name sind die Regel, nicht die Ausnahme
- **SCSS-Rohkanten:**
  - `@include m` gibt `calls` auf das Mixin
  - der Aufruf einer eigenen `@function` gibt `calls`
  - `@extend %p` oder `.k` gibt `extends`
- **Grammatik-Probe vom 2026-11-11:** Drei Formen geben einen lokalen
  ERROR, die Regeln danach kommen an:
  - `@use "x" as y`
  - `@extend` (im Block)
  - Range-Media-Queries `(width >= 576px)` in CSS und SCSS

  `support.md` nennt die Lücken. Im Probelauf auf bootstrap (84 % der
  `.scss` mit Fehlern) zeigt ein Test, dass die Regeln trotzdem Knoten
  bekommen.

## Brückenregeln (`builtinRules`)

| Namespace | Karte | Realm | ohne Scope | Scope ohne Treffer |
|---|---|---|---|---|
| `css.class` | alle Treffer | Repo | keine Kante | keine Kante |
| `css.id` | alle Treffer | Repo | keine Kante | keine Kante |
| `html.id` | genau einer | Repo | Repo, inferred | keine Kante |
| `js.function` | genau einer | Repo | Repo, inferred | keine Kante |

- **Keine geratenen Kanten für `css.class` und `css.id`.** Eine Regel wie
  `.btn`, die zwanzig Komponenten nennen, würde sonst bei jedem CSS-Edit zum
  Hotspot mit zwanzig Aufrufern und `blast-audit` rot machen. Zählt
  `blast-audit` geratene Kanten nicht mit, kann der Plan die Spalten
  wieder auf „Repo, inferred“ setzen. Das prüft der Plan als Erstes.
- **Erster Web-Task: ein Index der Angebote** nach (Namespace, Name). Der
  Brückenschritt durchsucht heute für jede Übergabe alle Angebote; bei
  Klassen wäre das Übergaben mal Angebote je Build.

- **Ein Treffer im Scope ist sicher (`extracted`).** Der Scope ist das, was
  die Seite verlinkt und einbettet.
- **Außerhalb des Scopes wird geraten (`inferred`).** Das ist das, was die
  `inferred`-Markierung aus v1.20 sichtbar macht.
- **`css.id`** beantwortet dieselben Übergaben wie `html.id`, wenn eine
  Seite ein Element per ID stylt. JS `getElementById` sucht `html.id`. Der
  Vollständigkeit halber bekommt auch ein `#x` in `querySelector`
  zusätzlich eine `css.id`-Übergabe; das baut Teilprojekt 5.

## Resolver

- **`resolve/web.go`:**
  - Imports von HTML und CSS auf Dateien: relativ, mit `/` als
    Wurzel-relativ zur Repo-Wurzel, SCSS-Partials
  - SCSS: `calls` auf Mixins und Funktionen über die per `@use`/`@import`
    erreichbaren Dateien (zyklenfest), `extends` auf Platzhalter und
    Klassen
- **Familien:** `css` und `scss` bilden eine Familie, `html` eine eigene.
- **Brücken:** HTML zu CSS und JS zu HTML laufen ausschließlich über die
  Brücken.

## Prüfkette

- **`html`:** lint mit `npx --yes html-validate@11.21.0` (ersetzt
  `htmlhint`), auf Datei `… {file}`. Eine Konfiguration des Projekts
  (`.htmlvalidate.json`) gewinnt; ohne sie `--preset standard` (prüft den
  HTML-Standard).
- **`css`:**
  - lint mit stylelint 17.x (neueste; im Plan nachschlagen) und
    `stylelint-config-standard-scss` 17.0.0, beide exakt über `npx --yes -p`
  - eine Projekt-Konfiguration gewinnt
- **graph und rules** für `html` und `css` wie Python.
- **`carrierRank`:** `… typescript, css, html`.
- **`IsSource` und `contentExts`:** die Endungen.
- **`support.md`:** die Zeilen `html` und `css` mit graph und rules.

## Kompatibilität

- `runtimeOf["css"] = "node"`, `runtimeOf["html"] = "node"`. HTML und CSS
  haben keine Versionen; ihre Werkzeuge laufen auf Node.
- **Einträge:**

  | Version | Fixture | Arten |
  |---|---|---|
  | 22, 24, 26 | `web-site` (HTML, CSS, SCSS, JS mit allen Brücken) | lint, graph |

- **`expect.toml` von `web-site`** verlangt die Kanten:
  - HTML → CSS-Datei (`imports`)
  - HTML-Klasse → CSS-Regel (`references`)
  - JS → HTML-Element (`references`)
  - `onclick` → JS-Funktion
  - SCSS `@include` → Mixin
- Ein Syntax-Fixture je Node-Linie gibt es hier nicht: HTML und CSS sind
  lebende Standards ohne Versionen.

## Tests und Abnahme

- Extraktor-, Resolver-, Brücken- und Spiegel-Tests.
- Probelauf auf bootstrap und vite: kein Absturz, Regeln und Elemente als
  Knoten, Brückenkanten in bootstrap/site.
