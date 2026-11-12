# Unterstützte Sprachen und Versionen

Welche Sprachen loomux prüft, was es für jede ausführt und gegen welche
Versionen das geprüft ist. Eine Zeile je Stack von `[verify]`; die
Einzelheiten stehen auf den unten verlinkten Seiten.

Wächter und Schreibschranke lesen Pfade und Befehle, keinen Code. Sie gelten
für jede Sprache gleich und haben hier keine Spalte.

## Übersicht

Eine Zelle nennt, was das eingebaute Preset ausführt; `—` heißt, loomux
liefert für diese Art nichts mit, und ein Projekt, das sie will, legt sie
unter `[verify.<stack>]` an ([Konfiguration](configuration.md#stacks)). Ein
Test hält jede Arten-Spalte gegen die eingebetteten Presets
(`internal/verify/presets.toml`), auf Englisch und auf Deutsch; eine Zelle
kann so keine Lane versprechen, die das Binary nicht hat.

| Stack | Erkannt an | lint | types | test | coverage | graph | rules | Versionen |
|---|---|---|---|---|---|---|---|---|
| go | `go.mod` | `go vet`, `gofmt` | — | `go test` | `gocover`, 100 % je Funktion | Code-Graph | Ebenenkarte | Go-Syntax bis 1.27, der `go/parser` der Toolchain, mit der loomux gebaut ist; eine Go-Datei, die nicht parst, lässt den Graph-Build scheitern; Go 1.26 und 1.27 geprüft ([Geprüfte Kombinationen](#geprüfte-kombinationen)) |
| python | `pyproject.toml`, `requirements.txt`, `uv.lock`, `manage.py` | ruff | mypy; pyright oder das mypy des Projekts, wo konfiguriert | pytest | coverage.py | Code-Graph | Ebenenkarte | Python 3, über `uv`; das mypy des Presets braucht 1.16 oder neuer; Werkzeugversionen festlegbar mit [`tools`](configuration.md#stacks); Python 3.11 bis 3.15 geprüft ([Geprüfte Kombinationen](#geprüfte-kombinationen)) |
| typescript | `tsconfig.json` neben `package.json`; `package.json` mit einer `.js`-, `.mjs`-, `.cjs`-, `.jsx`-, `.ts`-, `.mts`-, `.cts`- oder `.tsx`-Datei darunter, ohne Werkzeugkonfiguration daneben (`*.config.*`, `.*rc.*`) und ohne verschachtelte Pakete; `biome.json` | eslint; biome, wo konfiguriert | `tsc`, wo der Bereich selbst eine `tsconfig.json` hat | vitest, wo `package.json` es nennt | der lcov-Bericht von vitest über `check linecov`, wo `@vitest/coverage-v8` installiert ist | Code-Graph | Ebenenkarte | JavaScript zählt als TypeScript; die Werkzeuge aus `node_modules/.bin` des Bereichs: vor `npm install` sind ihre Lanes nicht verfügbar, weil `npx` die neueste Version ungepinnt nachladen würde, und Yarn Plug'n'Play ohne `node_modules` lässt sie dauerhaft nicht verfügbar; Node 22, 24 und 26 geprüft mit ESLint 10.14.0, typescript-eslint 8.72.2, TypeScript 6.0.3 und Vitest 5.1.0, Node 26 auch mit TypeScript 7.0.2 ([Geprüfte Kombinationen](#geprüfte-kombinationen)) |
| vue | `*.vue`, `vue.config.js` | — | `vue-tsc` | — | — | — | — | keine Version festgelegt |
| svelte | `*.svelte`, `svelte.config.js` | — | `svelte-check` | — | — | — | — | keine Version festgelegt |
| css | `*.css`, `*.scss`, `.stylelintrc*` | stylelint | — | — | — | — | — | keine Version festgelegt |
| html | `*.html`, `.htmlhintrc` | htmlhint | — | — | — | — | — | keine Version festgelegt |
| gdscript | `project.godot`, `.gdlintrc` | gdlint, gdformat (gdtoolkit) | — | gdUnit4 über `check godot-suite` oder GUT über `check gut-suite`, wo installiert | NanoCoverage über `check linecov`, wo installiert | Code-Graph | Ebenenkarte | Graph: Godot 3.x und 4.x, geprüft an 3.x, 4.2 und 4.7; test: gdUnit4 auf Godot 4.3 bis 4.8, GUT 7 auf Godot 3 und GUT 9 auf Godot 4 (geprüft an 3.6 und 4.7); coverage: Godot 4.3 bis 4.8; lint: Godot 3 und 4 (unter Godot 3 scheitert ein Projekt mit Addons im Baum an `gdformat --check`, das `addons/` nicht auslassen kann), gdtoolkit festlegbar mit [`tools`](configuration.md#stacks); geprüft an Godot 3.6, 4.5 und 4.7 ([Geprüfte Kombinationen](#geprüfte-kombinationen), [Einzelheiten](stacks/gdscript.md)) |
| csharp | `*.sln`, `*.slnx`, `*.csproj`, `global.json`, `Directory.Build.props` | `dotnet format` | `dotnet build -warnaserror` | `dotnet test`, wo ein `.csproj` `Microsoft.NET.Test.Sdk` oder `xunit.v3` nennt | coverlet (`XPlat Code Coverage`) über `check linecov`; auf Microsoft.Testing.Platform die Code-Coverage-Erweiterung von Microsoft (Cobertura), wo ein `.csproj` `Microsoft.Testing.Extensions.CodeCoverage` nennt | Code-Graph | Ebenenkarte | das .NET SDK der Maschine; eine `global.json`, die `Microsoft.Testing.Platform` als Test-Runner wählt, stellt test und coverage auf MTP um; .NET 10 und 11 geprüft, mit VSTest und mit MTP ([Geprüfte Kombinationen](#geprüfte-kombinationen)) |
| cpp | `CMakeLists.txt`, `CMakePresets.json`, `meson.build`, `*.cpp`, `*.hpp` | clang-tidy über die Compile-Kommandos (`run-clang-tidy`); clang-format auf der bearbeiteten Datei | der CMake-Build | ctest, nach dem Build, wo `CMakeLists.txt` Tests einschaltet | gcovr, 100 % der Zeilen | Code-Graph | Ebenenkarte | braucht ein konfiguriertes `build/` (`CMakeCache.txt`); ein `CMakeLists.txt`, `CMakePresets.json` oder `meson.build` in der Wurzel macht die Wurzel zum einzigen Bereich; C++17, 20 und 23 geprüft mit GCC auf Linux ([Geprüfte Kombinationen](#geprüfte-kombinationen)); keine Compiler-Version festgelegt |
| java | `pom.xml` (Maven); `build.gradle`, `settings.gradle` und ihre `.kts`-Formen, wo darunter eine `.java`-Datei liegt (Gradle) | Maven: Checkstyle, wo `pom.xml` das `maven-checkstyle-plugin` konfiguriert; Gradle: `checkstyleMain checkstyleTest`, wo ein `build.gradle*` `checkstyle` nennt; ein Kompilierfehler zeigt sich auch hier | Maven: `compile test-compile`; Gradle: `compileJava compileTestJava` | Maven: `mvn test`; Gradle: `test`; nach dem Kompilieren | JaCoCo über `check linecov`: Maven über das Plugin 0.8.16 ohne Konfiguration, Gradle wo der Build `jacoco` anwendet | Code-Graph | Ebenenkarte | JDK, Maven und Gradle der Maschine, Gradle über den Wrapper des Projekts, wo es einen hat; ein `pom.xml` oder Gradle-Build in der Wurzel macht die Wurzel zum einzigen Bereich; mit beiden in einem Bereich prüft Maven, und jeder Bereich wählt seinen eigenen Build; ein Android-Plugin schaltet die Build-Lanes ab; Java 8, 11, 17, 21, 25 und 27 geprüft auf Temurin mit Maven 3.10.0, Java 21 auch mit Gradle 9.9.1 ([Geprüfte Kombinationen](#geprüfte-kombinationen)) |
| php | `composer.json`, `phpunit.xml`, `phpunit.xml.dist`, `phpunit.dist.xml` | PHP_CodeSniffer, PSR-12, wo das Projekt nichts konfiguriert; dazu `php -l` auf der bearbeiteten Datei, auch vor `composer install` | PHPStan; wo das Projekt nichts konfiguriert, Stufe `max` auf den Autoload-Pfaden von `composer.json`, sonst auf `src`, sonst nicht verfügbar | PHPUnit | der Cobertura-Bericht von PHPUnit über `check linecov`, mit Xdebug | Code-Graph | Ebenenkarte | das `php` der Maschine, die Werkzeuge aus `vendor/bin` des Projekts: Vor `composer install` sind ihre Lanes nicht verfügbar; ein `composer.json` in der Wurzel macht die Wurzel zum einzigen Bereich; ohne `composer.json` (oder `phpunit.xml`) wird kein php-Stack erkannt und keine dieser Lanes läuft, der Code-Graph liest die `.php`-Dateien trotzdem; PHP 8.2, 8.3, 8.4 und 8.5 geprüft, mit PHPUnit 11 bis 13, PHPStan 2.3.5 und PHP_CodeSniffer 4.0.4 ([Geprüfte Kombinationen](#geprüfte-kombinationen)) |
| shell | `*.sh`, `.shellcheckrc` | shellcheck, nur auf der bearbeiteten Datei | — | — | — | — | — | keine Version festgelegt |
| sql | `.sqlfluff` | sqlfluff | — | — | — | — | — | keine Version festgelegt |
| rust | `Cargo.toml` | clippy, `cargo fmt` | — | — | — | — | — | keine Version festgelegt |

„Keine Version festgelegt“ heißt: loomux ruft die Version des Werkzeugs auf,
die das Projekt oder die Maschine mitbringt, und hat keine gegen eine andere
geprüft.

## Geprüfte Kombinationen

Jede Zeile ist eine Kombination, die loomux vor einem Major-Release und bei jeder Änderung der Liste wirklich fährt: die Laufzeit, die Addons und die Werkzeuge in den genannten Versionen, über ein Fixture-Projekt, auf Linux und Windows (C++ nur auf Linux). Die Tabelle ist `internal/compat/compat.toml`; ein Test hält sie Zelle für Zelle gegen diese Seite und gegen das Feld `proven` jeder Framework-Zeile ([GDScript](stacks/gdscript.md)). Wie die Läufe arbeiten, steht in der [CLI-Referenz](cli-reference.md#loomux-dev-compat-matrixrunscout-flags), welche Sperre sie vor einem Release setzen in [Releases](releases.md#wie-ein-pull-request-zum-release-wird).

| Stack | Laufzeit | Version | Fixture | Addons | Werkzeuge | Arten |
|---|---|---|---|---|---|---|
| gdscript | godot | 3.6 | godot3-gut | GUT 7.4.3 | — | test, graph |
| gdscript | godot | 3.6 | godot3-plain | — | gdtoolkit 3.* | lint, graph |
| gdscript | godot | 4.5 | godot4-plain | — | gdtoolkit 4.* | lint, graph |
| gdscript | godot | 4.7 | godot4-plain | — | gdtoolkit 4.* | lint, graph |
| gdscript | godot | 4.5 | godot4-gdunit4 | gdUnit4 6.2.2, NanoCoverage 0.1.1 | gdtoolkit 4.* | lint, test, coverage, graph |
| gdscript | godot | 4.7 | godot4-gdunit4 | gdUnit4 6.2.2, NanoCoverage 0.1.1 | gdtoolkit 4.* | lint, test, coverage, graph |
| gdscript | godot | 4.7 | godot4-gut | GUT 9.7.1 | gdtoolkit 4.* | lint, test, graph |
| python | python | 3.11 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.12 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.13 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.14 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.15 | python-mypy | — | coverage 7.17.*, mypy 2.4.*, pytest 9.2.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.15 | python-pyright | — | coverage 7.17.*, pytest 9.2.*, ruff 0.17.* | lint, types, test, coverage, graph |
| go | go | 1.26 | go-module | — | — | lint, test, coverage, graph |
| go | go | 1.27 | go-module | — | — | lint, test, coverage, graph |
| go | go | 1.26 | go-syntax-1.26 | — | — | graph |
| go | go | 1.27 | go-syntax-1.27 | — | — | graph |
| csharp | dotnet | 10.0 | csharp-xunit | — | — | lint, types, test, coverage, graph |
| csharp | dotnet | 11.0 | csharp-xunit | — | — | lint, types, test, coverage, graph |
| csharp | dotnet | 10.0 | csharp-xunit-mtp | — | — | lint, types, test, coverage, graph |
| csharp | dotnet | 11.0 | csharp-xunit-mtp | — | — | lint, types, test, coverage, graph |
| csharp | dotnet | 10.0 | csharp-syntax-10.0 | — | — | graph |
| csharp | dotnet | 11.0 | csharp-syntax-11.0 | — | — | graph |
| csharp | dotnet | 10.0 | godot4-csharp | — | — | graph |
| cpp | cpp | 17 | cpp-cmake-gtest | — | — | lint, types, test, coverage, graph |
| cpp | cpp | 20 | cpp-cmake-gtest | — | — | lint, types, test, coverage, graph |
| cpp | cpp | 23 | cpp-cmake-gtest | — | — | lint, types, test, coverage, graph |
| cpp | cpp | 17 | cpp-syntax-17 | — | — | graph |
| cpp | cpp | 20 | cpp-syntax-20 | — | — | graph |
| cpp | cpp | 23 | cpp-syntax-23 | — | — | graph |
| cpp | cpp | 17 | godot4-cpp | — | — | graph |
| java | java | 8 | java-maven-junit5 | — | — | lint, types, test, coverage, graph |
| java | java | 11 | java-maven-junit5 | — | — | lint, types, test, coverage, graph |
| java | java | 17 | java-maven-junit6-17 | — | — | lint, types, test, coverage, graph |
| java | java | 21 | java-maven-junit6 | — | — | lint, types, test, coverage, graph |
| java | java | 25 | java-maven-junit6 | — | — | lint, types, test, coverage, graph |
| java | java | 27 | java-maven-junit6 | — | — | lint, types, test, coverage, graph |
| java | java | 21 | java-gradle-junit6 | — | — | lint, types, test, coverage, graph |
| java | java | 8 | java-syntax-8 | — | — | graph |
| java | java | 11 | java-syntax-11 | — | — | graph |
| java | java | 17 | java-syntax-17 | — | — | graph |
| java | java | 21 | java-syntax-21 | — | — | graph |
| java | java | 25 | java-syntax-25 | — | — | graph |
| java | java | 27 | java-syntax-27 | — | — | graph |
| php | php | 8.2 | php-phpunit11 | — | — | lint, types, test, coverage, graph |
| php | php | 8.3 | php-phpunit12 | — | — | lint, types, test, coverage, graph |
| php | php | 8.4 | php-phpunit13 | — | — | lint, types, test, coverage, graph |
| php | php | 8.5 | php-phpunit13 | — | — | lint, types, test, coverage, graph |
| php | php | 8.2 | php-syntax-8.2 | — | — | graph |
| php | php | 8.3 | php-syntax-8.3 | — | — | graph |
| php | php | 8.4 | php-syntax-8.4 | — | — | graph |
| php | php | 8.5 | php-syntax-8.5 | — | — | graph |
| typescript | node | 22 | ts-vitest-eslint | — | — | lint, types, test, coverage, graph |
| typescript | node | 24 | ts-vitest-eslint | — | — | lint, types, test, coverage, graph |
| typescript | node | 26 | ts-vitest-eslint | — | — | lint, types, test, coverage, graph |
| typescript | node | 26 | ts7-vitest | — | — | types, test, coverage, graph |
| typescript | node | 22 | js-syntax-22 | — | — | graph |
| typescript | node | 24 | js-syntax-24 | — | — | graph |
| typescript | node | 26 | js-syntax-26 | — | — | graph |

Die C#-Einträge nennen kein Werkzeug: Format, Build und Test bringt das .NET SDK mit, und `csharp-xunit` legt seine Testpakete selbst fest (xunit.v3.mtp-off 4.0.3, xunit.runner.visualstudio 4.0.1, Microsoft.NET.Test.Sdk 18.11.1, coverlet.collector 10.1.0). Es ist xunit.v3 ohne Microsoft.Testing.Platform: Das Paket `xunit.v3` bringt ab 4.0 MTP v2 mit, und `dotnet test` verweigert ab dem .NET-10-SDK den VSTest-Weg, den die Vorgabe (`--collect "XPlat Code Coverage"`) geht. `csharp-xunit-mtp` ist dasselbe Projekt, wie xunit.v3 es von Haus aus anlegt (xunit.v3 4.0.3, Microsoft.Testing.Extensions.CodeCoverage 18.12.0), mit einer `global.json`, die `Microsoft.Testing.Platform` als Test-Runner wählt; der Lauf setzt seine SDK-Festlegung daneben. Ein Lauf lässt die Projekte auf die .NET-Version des Eintrags zielen (`net10.0`, `net11.0`).

Eine C++-Version ist ein ISO-Standard. Auch die C++-Einträge nennen kein Werkzeug: CMake und GCC bringt der Linux-Runner mit, clang-tidy 23.1.4, clang-format 23.1.5 und gcovr 8.8 kommen über `uv tool install` (das Wheel von clang-tidy nennt sein `run-clang-tidy` `run-clang-tidy.py`; das Setup legt einen Link unter dem Namen an, den die Vorgabe aufruft), und `cpp-cmake-gtest` holt GoogleTest 1.18.0 selbst, festgelegt über die Release-URL und SHA-256. Ein Lauf konfiguriert `build/` mit dem Standard des Eintrags (`-DCMAKE_CXX_STANDARD=23`). C++ läuft nur auf Linux: Der Windows-Runner bringt MSVC mit, das die GCC-Coverage-Flags des Fixtures nicht nimmt. `godot4-cpp` hält eine GDExtension im Layout von godot-cpp (`project/` und `src/`).

Eine Java-Version ist ein Feature-Release von Eclipse Temurin, das `actions/setup-java` installiert. Auch die Java-Einträge nennen kein Werkzeug: CI holt Maven 3.10.0 von Maven Central und prüft es auf beiden Systemen gegen seine SHA-512, denn die Runner bringen verschiedene mit, und `java-gradle-junit6` fährt Gradle 9.9.1 über seinen Wrapper, der die Distribution über SHA-256 festlegt. Die Fixtures legen ihre Plugins selbst fest (maven-compiler-plugin 3.16.0, maven-surefire-plugin 3.6.1, maven-checkstyle-plugin 3.6.0; gemessen wird mit JaCoCo 0.8.16), JUnit und Checkstyle nach dem Java, auf dem sie laufen: JUnit 6 und Checkstyle 12 brauchen Java 17, Checkstyle 14 Java 21, also fahren 8 und 11 JUnit 5.14.4 und Checkstyle 9.3, 17 JUnit 6.1.3 und Checkstyle 12.3.1, ab 21 JUnit 6.1.3 und Checkstyle 14.4.0. Ein Lauf kompiliert das Fixture auf der Version des Eintrags: `maven.compiler.release`, `source` und `target`, wo es auch auf Java 8 läuft, dessen javac kein `--release` kennt, und die Toolchain von Gradle. Java 27 brachte keine eigene finale Syntax, darum belegt `java-syntax-27` die Formen von 25.

Eine PHP-Version ist ein Release-Zweig von php.net, den `shivammathur/setup-php` 2.40.0 mit Xdebug 3.5.3 und Composer 2.11.2 installiert. Auch die PHP-Einträge nennen kein Werkzeug: Jedes Fixture legt PHPUnit, PHPStan 2.3.5 und PHP_CodeSniffer 4.0.4 in seiner `composer.lock` fest, die ein Lauf vor den Lanes in jede Arbeitskopie installiert (`composer install --no-interaction --prefer-dist`). PHPUnit setzt die Untergrenze: 11.5.57 braucht PHP 8.2, 12.5.43 PHP 8.3, 13.4.6 PHP 8.4.1; darum läuft `php-phpunit11` auf 8.2, `php-phpunit12` auf 8.3 und `php-phpunit13` auf 8.4 und 8.5, jede Lock-Datei aufgelöst für das niedrigste PHP ihres Fixtures. Die Fixtures konfigurieren weder PHP_CodeSniffer noch PHPStan, die Lanes fahren also PSR-12 und Stufe `max`.

Eine Node-Version ist eine Release-Linie von nodejs.org, die `actions/setup-node` 7.1.0 installiert. Auch die Node-Einträge nennen kein Werkzeug: Jedes Fixture legt seine Werkzeuge in seiner `package-lock.json` fest, die ein Lauf vor den Lanes in jede Arbeitskopie installiert (`npm ci --no-audit --no-fund`). `ts-vitest-eslint` fährt ESLint 10.14.0 mit typescript-eslint 8.72.2 auf TypeScript 6.0.3; typescript-eslint nimmt kein TypeScript 7, darum fährt `ts7-vitest` TypeScript 7.0.2 ohne ESLint, auf der neuesten Linie. Beide fahren Vitest 5.1.0 mit @vitest/coverage-v8 5.1.0 auf Vite 8.3.8, und in beiden importiert ein JavaScript-Modul ein TypeScript-Modul. Node 26 brachte keine eigene JavaScript-Syntax, darum belegt `js-syntax-26` die Formen von 24, daneben TypeScript, das Node ausführt, indem es die Typen entfernt.

Die Liste führt jede Go-, Python-, Godot-, .NET-, Java-, PHP- und Node-Version, die ihr Upstream noch pflegt, nicht nur die neueste. Die Releases von Node kommen aus dem Dist-Index von nodejs.org (`dist/index.json`), der das Alpha der nächsten Linie (`27.0.0-alpha.0.0.0`) als Vorabversion führt; ein Eintrag nennt eine Node-Linie nur mit ihrer Major, darum zählt ein Release Candidate einer neuen Minor einer gelisteten Linie nicht. Das Supportende von Node kommt von endoflife.date unter `nodejs`. Die Feature-Releases von Java kommen aus der Adoptium-API (`available_releases`); ihr Tip ist, solange er nur Early-Access-Builds hat, die Vorabversion (`28-ea`), und das Supportende kommt von endoflife.date unter `eclipse-temurin`. Das neueste Release von PHP kommt aus der Release-Liste von php.net, seine Release Candidates aus der eigenen Liste von php.net (ein Zweig, dessen Release abgeschaltet ist, hat keinen); `php-phpunit11` und `php-phpunit12` sind zurückgehalten, darum fährt nur `php-phpunit13` das nächste als Probe. setup-php bietet keinen Release Candidate an, darum fährt eine Probe von `8.6.0RC5` den Nightly-Build des Zweigs 8.6 mit dem Xdebug, das setup-php dafür einrichtet: Xdebug 3.6.0alpha1 liegt nicht auf PECL, woher setup-php eine festgelegte Version holt. C++ hat keine maschinenlesbare Quelle für seine Standards: Der Scout führt den nächsten selbst, als Vorabversion `26-draft`, bis ISO ihn veröffentlicht und er von Hand in die Liste wandert; ein Ende der Pflege meldet er für C++ nicht, denn auch ein Standard verlässt die Liste von Hand. Jeden Montag fragt der Kompatibilitäts-Scout (`loomux dev compat scout`) upstream nach dreierlei:

- **Eine neue stabile Version:** Er fährt sie als Probe und öffnet ein Issue mit dem Eintrag, der hinzukommt. Ein Eintrag, dessen Fixture mit Absicht unter neueren Versionen bleibt, weil sein Framework dort endet, trägt in der Liste `held` mit dem Grund; nach Neuerem fragt der Scout für ihn nicht. Jede andere Linie unter der neuesten Version ihrer Laufzeit meldet weiter, bis sie nachzieht.
- **Eine Alpha, Beta oder ein Release Candidate der nächsten Version:** Er fährt sie vorab als Probe. Ein Fehlschlag öffnet nur ein Issue und blockiert keinen Pull Request; ein Pre-Release kommt nie in die Liste, das Release zieht nach, sobald die Version stabil ist.
- **Eine gelistete Version nach Ende der Pflege:** Die Antwort kommt von [endoflife.date](https://endoflife.date) (`isEol`). Die Version fällt aus der Liste. Der Pull Request, der sie austrägt, trägt mindestens `release:patch` und nennt im `## Changelog` unter `Removed` die letzte loomux-Version, die sie noch unterstützt, etwa: „Python 3.10 is no longer supported; v1.19.2 is the last release that supports it.“

## Was der Code-Graph liest

Der Graph (`loomux graph`, die Lane `graph`, `graph_*` über MCP) deckt Go,
Python, GDScript, C#, C++, Java, PHP, JavaScript und TypeScript ab.

| Sprache | Liest | Löst auf | Lässt aus |
|---|---|---|---|
| Go | Pakete, Typen, Funktionen, Methoden | Aufrufe über Pakete des Moduls, Importe | typgeprüfte Aufrufe (`go/types`) |
| Python | Module, Klassen, Funktionen, Methoden | Importe, Aufrufe, Basisklassen | `.pyi`-Stubs |
| GDScript | Skripte, innere Klassen, Funktionen, Signale; Szenen, Ressourcen, `project.godot`, `.uid`-Dateien | `res://`- und `uid://`-Pfade, `class_name`, Autoloads (Szenen-Autoloads über ihr Wurzelskript), geerbte Aufrufe, Signale per Name und als Wert, Szenenverbindungen, auch zu C#-Methoden, -Signalen und `[GlobalClass]`-Klassen | VisualScript, Skripte inline in einer Szene, Typen von lokalen Variablen und Feldern |
| C# | Dateien, Typen, Methoden, Konstruktoren, Eigenschaften, Indexer, `[Signal]`-Delegates; Syntax bis C# 14 vollständig | `using`, Basen (`extends`/`implements`), Aufrufe, Konstruktoren | Typen lokaler Variablen, Methoden in einer zweiten Datei einer `partial`-Klasse für Szenenverbindungen; aus C# 15: `union` (kein Typknoten), `closed record class` (kein Knoten), ein Typ mit Erweiterungs-Indexer (kein Knoten), `break`/`continue` mit Marke (Fehler im Rumpf, die übrigen Mitglieder kommen an) |
| C++ | Dateien (`.cpp`, `.h`, `.hpp` und Verwandte), Klassen, Structs, Unions, Enums, Funktionen und Methoden mit Rumpf; GDExtension-Klassen (`GDCLASS`) und ihre Bindungen; Syntax von C++17, 20 und 23 | `#include`, Basisklassen, Aufrufe per Name oder qualifiziertem Namen; GDScript-`extends` und `.new()`, Knotentypen und Verbindungen in Szenen zu GDExtension-Klassen | den Präprozessor (Makros bleiben unexpandiert, jeder `#if`-Zweig wird gelesen), Typen und Überladungen (Member-Aufrufe `x.f()` bleiben offen), `.c`-Dateien, ein Signal, das eine Basisklasse einer GDExtension-Klasse bindet (eine Szenenverbindung findet nur die Klasse selbst), ein Mitglied, das nur die C++-Klasse hat, auf einem Knoten mit Skript und Typ; aus C++23: `if consteval` (die Funktion damit bekommt keinen Knoten, und die Definition vor der schließenden Klammer ihres Namespace geht verloren; was folgt, kommt ohne Namespace und Owner an; eine so abgeschnittene Datei wird erst ab einer Zeile in Spalte 0 neu gelesen, und ein Namespace oder eine Klasse, die am Schnitt offen war, kostet eine Definition) |
| Java | Dateien, Klassen, Schnittstellen, Enums, Records, Annotationstypen, verschachtelte Typen, Methoden, Konstruktoren; Syntax bis Java 21 vollständig | Imports (einzeln, `*`, `static`), Basen (`extends`/`implements`), Aufrufe per Name oder qualifiziertem Namen, `new` | Typen und Überladungen (Member-Aufrufe `obj.m()` bleiben offen), Typen lokaler Variablen; aus Java 25: `import module` (gelesen, nennt aber keine Datei), eine Anweisung vor `super(…)` (Fehler im Konstruktorrumpf, der Knoten und die Mitglieder danach kommen an) |
| PHP | Dateien, Klassen, Schnittstellen, Traits, Enums, Methoden, Funktionen, auch in mehreren `namespace`-Blöcken einer Datei und zwischen HTML; Syntax bis PHP 8.4 vollständig | `use` von Klassen und Funktionen (ein `use const` wird gelesen, ergibt aber keine Kante), `require`/`include` eines Stringliterals, Basen (`extends`/`implements`, Traits), Aufrufe per Name oder qualifiziertem Namen, `self::`, `static::`, `parent::`, `$this->`, `new` | Typen von Variablen (Member-Aufrufe `$obj->m()` bleiben offen); aus PHP 8.5: `clone($o, [...])` und der Cast `(void)` (Fehler im Rumpf, der Rest kommt an); eine Datei voller Heredocs mit PHP-Code darin kann rund 20 s zum Parsen brauchen und Deklarationen an Parse-Fehler verlieren; der Graph behält das Ergebnis jeder Datei, bis sie sich ändert, wartet also einmal, aber `check rules` parst jede Datei bei jedem Lauf neu: ist `[rules]` gesetzt, kehrt die Wartezeit bei jedem Edit und jedem Commit wieder |
| JavaScript, TypeScript | Dateien (`.js`, `.mjs`, `.cjs`, `.jsx`, `.ts`, `.mts`, `.cts`, `.tsx`), Funktionen, Klassen, Methoden, an ein `const`, `let` oder `var` gebundene Funktionen auf Modulebene, CommonJS `exports.f = …` und `module.exports.f = …`; TS-Schnittstellen, -Typen, -Enums, jede Deklaration eines `namespace` (als `N.f`) | `import`, `export … from`, `require` und `import()` eines Stringliterals, Pfad-Aliase aus `tsconfig.json`/`jsconfig.json`, Basen, Aufrufe per Name, `this.`, `super.`, über einen importierten Namensraum, `new` | Importe von Paketen, `extends` in `tsconfig.json`, Typen (Member-Aufrufe `obj.m()` bleiben offen, ebenso statische `C.s()` und `N.f()` in einen TS-Namespace), Deklarationsdateien (`.d.ts`, `.d.mts`, `.d.cts`: nur der Dateiknoten), verschachtelte Funktionen, Objektmethoden, Ambient-Deklarationen (`declare …`); ein JSX-Attribut `class=` in einer `.js`-Datei (die Definition, die es enthält, bekommt keinen Knoten, und der Rest der Datei wird erst ab Deklarationen in Spalte 0 neu gelesen; `.jsx` liest es) |

Die vollständigen Regeln stehen in der
[CLI-Referenz](cli-reference.md#loomux-graph-build---root-pfad---no-reuse).

## Stacks ohne Sprache

`wiki` und `wiki-sources` prüfen das Wiki eines Projekts, `project` führt
Befehle aus, die ein Projekt selbst nennt; keiner davon liest Code einer
Sprache. Siehe [Konfiguration](configuration.md#stacks).
