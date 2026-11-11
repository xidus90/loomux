# PHP im Code-Graphen, in der Prüfkette und in der Kompatibilitätsliste

**Stand:** 2026-11-11 · Teilprojekt 4 von
`2026-11-10-graph-sprachen-programm.md`

## Ziel

PHP steht in Graph, Prüfkette und Kompatibilitätsliste auf der Stufe von
Python, für PHP 8.2 bis 8.5. 8.6 läuft als Vorab-Probe des Scouts, bis es
am 2026-11-19 erscheint.

## Nicht-Ziele

- Keine Auflösung über Composer-Autoload-Regeln (`psr-4`). Die Namespaces
  stehen in den Dateien.
- Keine Typinferenz (`$obj->m()` bleibt ohne Kante, außer `$this->m()`).
- Keine Templates (Blade, Twig); HTML in einer `.php`-Datei wird nicht
  gelesen.

## Extraktor `internal/code/extract/php`

- **Grunddaten:** Sprache `php`, Endung `.php`, Version `php/1@<Parser>`,
  Grammatik `gotreesitter/grammars/php` (die Variante mit HTML-Rahmen; was
  außerhalb von `<?php` steht, wird übersprungen).
- **Knoten:**
  - `class`, `interface`, `trait`, `enum`; Art `interface` für
    Schnittstellen, sonst `class`
  - Methoden (Art `method`, Owner der Typ)
  - freie Funktionen (Art `function`)
- **Weitere Felder:**
  - **`Exported`:** Methoden nach `public` (Vorgabe in PHP: public); Typen
    und Funktionen immer.
  - **Namespaces:** `namespace A\B;` und `namespace A\B { }`, auch mehrere
    je Datei. Der Namespace jedes Typs und jeder Funktion steht in
    `Result.Enclosing`.
  - **Imports:** `use A\B\C;`, `use A\B\{C, D as E};`, `use function A\f;`,
    `use const A\X;`. Jeder wird ein `extract.Import` mit Pfad und Alias
    (Art `function`/`const` in einem dokumentierten Feld) plus eine
    `imports`-Rohkante. `require`/`include` mit Stringliteral wird zur
    `imports`-Rohkante auf den Pfad.
- **Rohkanten:**
  - `extends` aus `base_clause`, `class_interface_clause` und `use T;` in
    der Klasse (Traits)
  - `calls`:
    - `f()`
    - `A\f()`
    - `X::m()`, `self::m()`, `static::m()`, `parent::m()`
    - `$this->m()`
    - `new X()` mit `Member "new"`
- **Grammatik-Probe vom 2026-11-11:**
  - 8.2 bis 8.4 und 8.6 (partielle Anwendung) parsen ohne Fehler.
  - 8.5 hat eine Lücke: `clone($o, [...])` gibt einen ERROR im Rumpf; der
    Rest kommt an.

## Resolver `internal/code/resolve/php.go`

Namensauflösung wie PHP selbst:
- vollständig qualifiziert `\A\B`
- Alias aus `use`
- sonst der eigene Namespace plus Name
- bei Funktionen der Rückfall auf den globalen Namespace

**Kanten:**
- **`imports`:**
  - `use A\B\C;` geht auf die Datei, die `A\B\C` deklariert. Ein fremder
    Name gibt keine Kante.
  - `require`/`include` geht auf die Datei relativ zur Datei.
  - Eine Kante je Paar, kleinste Zeile.
- **`extends`/`implements`:** nach der Art des Ziels. Traits geben
  `extends`. `Unresolved` gibt es nur hier.
- **`calls`:**
  - `$this->m()` und `self::m()`: Typ, Basen, Traits
  - `parent::m()`: die Basen
  - `static::m()`: wie `self`
  - `X::m()` über die Namensauflösung
  - `f()` über Namespace und Rückfall
  - `new X()`: `__construct`, sonst der Typ

## Restliche Graph-Stellen

- **`blast.IsTestPath`:** Segment `tests` oder `test` (Groß- und
  Kleinschreibung egal) oder Dateiname `*Test.php`. Negativfall
  `Contest.php`.
- **`repomap`, `isFileQuery`:** `.php`.
- **`corpus.pruneDirs`:** `vendor` steht schon drin.

## Prüfkette: Stack `php`

- **Erkennung:** `composer.json` oder `phpunit.xml`/`phpunit.xml.dist`. Ein
  `composer.json` allein ohne `.php`-Datei genügt; Composer heißt PHP.
- **Lanes:** Werkzeuge kommen über `composer` aus dem Projekt
  (`vendor/bin/…`). Fehlt eines, läuft es über `composer exec` nicht; die
  Lane meldet „nicht verfügbar“, wie andere Stacks ohne Werkzeug. Die
  Mechanik dafür steht im Plan nach dem Muster der vorhandenen
  `needs`/`when_found`.
  - **lint:**
    - `vendor/bin/phpcs` mit einer Projekt-Konfiguration (`phpcs.xml`,
      `.phpcs.xml`, `phpcs.xml.dist`), sonst `--standard=PSR12`
    - auf Datei: `vendor/bin/phpcs {file}`
    - zusätzlich immer `php -l` auf Datei
  - **types:** `vendor/bin/phpstan analyse --no-progress`; ohne
    `phpstan.neon(.dist)` mit `--level=max` auf `src`
  - **test:** `vendor/bin/phpunit`
  - **coverage:**
    - `XDEBUG_MODE=coverage vendor/bin/phpunit --coverage-clover {coverdata}`
    - dann `{loomux} check linecov --report {coverdata}`
    - linecov kennt `phpunit` schon. Ein Bericht im Clover-Format wird
      geprüft; braucht es ein anderes Format, entscheidet der Plan.
  - **graph** und **rules** wie Python
- **Weitere Stellen:**
  - `carrierRank`: `… java, php`
  - `IsSource` und `contentExts`: `.php`
  - `support.md` (en, de): Zeile `php`

## Kompatibilität

- `runtime = "php"`, `runtimeOf["php"] = "php"`.
- **Einträge:**

  | Version | Fixture | Arten |
  |---|---|---|
  | 8.2 | `php-phpunit11` | lint, types, test, coverage, graph |
  | 8.3 | `php-phpunit12` | lint, types, test, coverage, graph |
  | 8.4, 8.5 | `php-phpunit13` | lint, types, test, coverage, graph |
  | 8.2, 8.3, 8.4, 8.5 | `php-syntax-<v>` | graph |

  PHPUnit 13 verlangt PHP ≥ 8.4, 12 verlangt ≥ 8.3, 11 verlangt ≥ 8.2.
  PHPStan, PHP_CodeSniffer und Xdebug werden im Plan recherchiert und exakt
  gepinnt, ebenso `composer.lock`.
- **Scout:**
  - Releases aus `https://www.php.net/releases/index.php?json&version=8`
    (nur stabile)
  - Vorabversionen aus
    `https://www.php.net/release-candidates.php?format=json` (Einträge mit
    `release.enabled != false`)
  - Die Probe installiert über setup-php den Nightly-Build des Zweigs (für
    8.6 bietet setup-php kein RC an), mit Xdebug 3.6.0alpha1.
  - Supportende über endoflife.date `php`
  - Ein 8.6-Syntax-Fixture (partielle Anwendung) entsteht mit der Probe;
    in die Liste kommt 8.6 erst nach dem Release.
- **CI:** `shivammathur/setup-php` (SHA-gepinnt) mit Version,
  `coverage: xdebug` und `tools: composer`; `compat-setup.sh` mit
  `php) ;;`.

## Tests und Abnahme

Wie bei C#:
- Extraktor-, Resolver- und Spiegel-Tests
- Probelauf auf laravel/framework und composer/composer: kein Absturz,
  ParseErrors wie in der Parse-Prüfung
- Der Graph dieses Repos bleibt bis auf den Stempel gleich.
