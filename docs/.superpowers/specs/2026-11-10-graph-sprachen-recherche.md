# Recherche: Versionen, Werkzeuge und Syntax je Sprache (Stand 2026-11-10)

Quellen sind Live-Abrufe vom 2026-11-10 (endoflife.date, Herstellerseiten,
GitHub-, npm-, NuGet- und PyPI-Registries). Die Snippets, die nicht von den
genannten Seiten stammen, sind ungeprüft, bis ein Fixture sie kompiliert.

## Unterstützte Versionen

| Stack | Versionen (EOL) | Vorab |
|---|---|---|
| .NET | 11 (GA 2026-11-10, STS), 10 (LTS); 8 und 9 enden 2026-11-10 | — |
| Java (Temurin) | 8 (2030-12), 11 (2027-10), 17 (2027-10), 21 (2029-12), 25 (2031-09), 27 (2027-03) | 28 EA |
| PHP | 8.2 (2026-12-31), 8.3 (2027-12-31), 8.4 (2028-12-31), 8.5 (2029-12-31) | 8.6 RC5, GA 2026-11-19 |
| Node | 22 (2027-04-30), 24 (2028-04-30), 26 (2029-04-30) | — |
| C++ | ISO 17, 20, 23 veröffentlicht; 26 technisch fertig, ISO-Veröffentlichung unbestätigt | 26 |
| Godot | 4.7.2 aktuell; 4.6, 4.5 und 3.6 gepflegt; godot-cpp 10.0.0 (unabhängig versioniert, ab Godot 4.3) | 4.8-dev |

Quellen: <https://endoflife.date/api/dotnet.json>,
<https://dotnet.microsoft.com/en-us/platform/support/policy/dotnet-core>,
<https://endoflife.date/api/eclipse-temurin.json>,
<https://endoflife.date/api/php.json>,
<https://wiki.php.net/todo/php86>,
<https://endoflife.date/api/nodejs.json>,
<https://en.cppreference.com/w/cpp/compiler_support>,
<https://endoflife.date/api/godot.json>,
<https://api.github.com/repos/godotengine/godot-cpp/releases>

## Werkzeuge (neueste Version, gepflegt)

- **C#:**
  - `dotnet format` und `dotnet build`/`dotnet test` aus dem SDK
  - xunit.v3 4.0.3, NUnit 5.0.0, MSTest 4.5.2
  - coverlet.collector 10.1.0, Microsoft.Testing.Extensions.CodeCoverage 18.12.0
- **C++:**
  - clang-tidy und clang-format (LLVM 23.1.5), cppcheck 2.22.0
  - GoogleTest 1.18.0, Catch2 3.16.1
  - gcovr 8.8, CMake 4.4.5
- **Java:**
  - Checkstyle 14.4.0, PMD 7.29.0, SpotBugs 4.10.5, Error Prone 2.52.0
  - JUnit 6.1.3, JaCoCo 0.8.16
  - Maven 3.10.0, Gradle 9.9.1
- **PHP:**
  - PHPStan 2.3.5, PHP_CodeSniffer 4.0.4, PHP-CS-Fixer 3.95.30
  - PHPUnit 13.4.6 (PHP ≥ 8.4; 12 für 8.3, 11 für 8.2)
  - Xdebug 3.5.3 (PCOV kaum gepflegt)
- **Web:**
  - TypeScript 7.0.2 (typescript-eslint 8.72.2 erlaubt nur < 6.1, darum TS 6.0 im Fixture)
  - ESLint 10.14.0, Biome 2.6.6, oxlint 1.93.0
  - Vitest 5.1.0, Jest 30.5.2
  - html-validate 11.21.0, vnu-jar 26.11.9
  - stylelint 18.0.0 (stylelint-config-standard-scss 17.0.0 verlangt stylelint 17)
  - Dart Sass 1.108.1 (LibSass und node-sass am Ende)

## Syntax je Version (für die Syntax-Fixtures)

**C#**
- **14:**
  - `field` im Accessor
  - `extension<T>(IEnumerable<T> s) { … }`
  - `a?.B = c;`
  - Lambda-Modifikatoren ohne Typ
- **15:**
  - `union Pet(Cat, Dog);`
  - `closed record class X;`
  - `break outer;` / `continue outer;`
  - `[with(capacity: 8), .. xs]`
  - Erweiterungs-Indexer
  - Memory-Safety nur als Preview

Grammatik-Probe (gotreesitter v0.57.0):
- C# 14: ohne Fehler; `extension_declaration` mit `extension_body` enthält die Mitglieder.
- C# 15 hat vier Lücken. Die vierte fand der Compat-Lauf: Ein
  Erweiterungs-Indexer kostet den umgebenden Typ seinen Knoten. Die drei
  übrigen:
  - `union` wird als lokale Funktion gelesen, ohne Fehler, aber ohne Typknoten.
  - `closed record class` gibt einen ERROR.
  - `break outer` gibt einen ERROR im Methodenrumpf; die folgenden Mitglieder kommen an.

**Java**
- **8:** Lambda, `default`-Methode, Typannotationen
- **11:** `var` in Lambda-Parametern
- **17:** `sealed … permits`
- **21:** Record-Muster, `switch`-Muster mit `when`
- **25:**
  - `import module java.base;`
  - Anweisungen vor `super(…)`
  - kompakte Quelldatei mit `void main()`
- **27:** keine finale neue Syntax. Das Syntax-Fixture belegt dann die Formen der Vorgänger auf 27.

Quelle: <https://en.wikipedia.org/wiki/Java_version_history> (openjdk.org war gesperrt).

**PHP**
- **8.2:** `readonly class`, DNF-Typen `(A&B)|null`, Konstanten in Traits
- **8.3:** typisierte Klassenkonstanten, `C::{$n}`, `#[\Override]`
- **8.4:** Property-Hooks, `public private(set)`, `new P()->v`
- **8.5:** `|>`, `clone($o, [...])`, `(void)`, Closures in Konstantenausdrücken
- **8.6 (RC):** partielle Anwendung `f(?, ...)`, `#[\Override]` auf Konstanten

**JavaScript**
- **ES2024:** RegExp-Flag `v` (Node 20+)
- **ES2025:** Import-Attribute `with { type: "json" }` (Node 18.20/20.10+), RegExp-Modifikatoren und doppelte benannte Gruppen (Node 23+)
- **ES2026:** keine Syntax
- **Vorab (ES2027):** `using` und `await using` (Node 24+)
- **TypeScript 6.0 und 7.0:** keine neue Syntax. 7.0 macht `module Foo {}` und `asserts` zu Fehlern.

Quelle: <https://raw.githubusercontent.com/tc39/proposals/main/finished-proposals.md>, MDN-BCD.

**C++**
- **17:** strukturierte Bindungen, `if constexpr`
- **20:** `concept`/`requires`, Koroutinen, `<=>`
- **23:** deducing `this`, `if consteval`
- **26:**
  - Pack-Indexing `Ts...[0]`
  - Reflexion `^^T` / `[: m :]`
  - Contracts `pre`/`post`/`contract_assert`
  - `template for`

## Nachtrag 2026-11-11: Pins für Java und PHP

**Java** (Maven Central, maven.apache.org, docs.junit.org, checkstyle.org, docs.gradle.org)

| Werkzeug | Version | Mindest-Java |
|---|---|---|
| Maven | 3.10.0 | 8 |
| maven-compiler-plugin | 3.16.0 | 8 |
| maven-surefire-plugin | 3.6.1 | 8 |
| maven-checkstyle-plugin | 3.6.0 (bringt Checkstyle 9.3 mit) | 8 |
| Checkstyle | 9.3 / 10.26.1 / 12.x / 14.4.0 | 8 / 11 / 17 / 21 |
| jacoco-maven-plugin | 0.8.16 (Klassen bis Java 27) | 8 |
| JUnit | 5.14.4 / 6.1.3 | 8 / 17 |
| Gradle | 9.9.1 | Gradle läuft auf 17 bis 27; Java 27 ab Gradle 9.8 |

Weitere Fakten:
- **setup-java:** v6.2.0 = `77dd5b2ce225ea11469928ac317850245a74ae78`. Der
  Tag `v6` ist veraltet.
- **Maven auf den Runnern:** ubuntu-latest hat 3.10.0, windows-latest
  3.9.16.
- **Adoptium** (`/v3/info/available_releases`): `available_releases`
  endet bei 27, `tip_version` ist 28; LTS sind 8, 11, 17, 21 und 25.

**PHP** (Packagist, GitHub, xdebug.org, php.net)

| Werkzeug | Version | PHP |
|---|---|---|
| PHPStan | 2.3.5 | ≥ 7.4 |
| PHP_CodeSniffer | 4.0.4 | ≥ 7.2 |
| PHPUnit | 11.5.57 / 12.5.43 / 13.4.6 | ≥ 8.2 / ≥ 8.3 / ≥ 8.4.1 |
| Composer | 2.11.2 | ≥ 7.2.5 |
| Xdebug | 3.5.3 (8.2–8.5); 3.6.0alpha1 (8.6) | |

Weitere Fakten:
- **setup-php:** 2.40.0 = `eb7c497e18156a6bbabfef1d3a82760b9eda3962`.
  `8.6` installiert dort einen Nightly-Build, nicht RC5.
- **Release-Quellen:**
  - `https://www.php.net/releases/index.php?json&version=8` listet nur
    stabile Releases.
  - `https://www.php.net/release-candidates.php?format=json` listet RCs
    (Einträge mit `release.enabled != false`). 8.6.0 ist dort RC5.
