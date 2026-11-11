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
| typescript | `tsconfig.json` neben `package.json`, `biome.json` | eslint; biome, wo konfiguriert | `tsc` | vitest, wo `package.json` es nennt | vitest mit Coverage | — | — | `.js` und `.jsx` zählen als TypeScript; keine Version festgelegt |
| vue | `*.vue`, `vue.config.js` | — | `vue-tsc` | — | — | — | — | keine Version festgelegt |
| svelte | `*.svelte`, `svelte.config.js` | — | `svelte-check` | — | — | — | — | keine Version festgelegt |
| css | `*.css`, `*.scss`, `.stylelintrc*` | stylelint | — | — | — | — | — | keine Version festgelegt |
| html | `*.html`, `.htmlhintrc` | htmlhint | — | — | — | — | — | keine Version festgelegt |
| gdscript | `project.godot`, `.gdlintrc` | gdlint, gdformat (gdtoolkit) | — | gdUnit4 über `check godot-suite` oder GUT über `check gut-suite`, wo installiert | NanoCoverage über `check linecov`, wo installiert | Code-Graph | Ebenenkarte | Graph: Godot 3.x und 4.x, geprüft an 3.x, 4.2 und 4.7; test: gdUnit4 auf Godot 4.3 bis 4.8, GUT 7 auf Godot 3 und GUT 9 auf Godot 4 (geprüft an 3.6 und 4.7); coverage: Godot 4.3 bis 4.8; lint: Godot 3 und 4 (unter Godot 3 scheitert ein Projekt mit Addons im Baum an `gdformat --check`, das `addons/` nicht auslassen kann), gdtoolkit festlegbar mit [`tools`](configuration.md#stacks); geprüft an Godot 3.6, 4.5 und 4.7 ([Geprüfte Kombinationen](#geprüfte-kombinationen), [Einzelheiten](stacks/gdscript.md)) |
| csharp | `*.sln`, `*.slnx`, `*.csproj`, `global.json`, `Directory.Build.props` | `dotnet format` | `dotnet build -warnaserror` | `dotnet test`, wo ein `.csproj` `Microsoft.NET.Test.Sdk` nennt | coverlet (`XPlat Code Coverage`) über `check linecov` | Code-Graph | Ebenenkarte | das .NET SDK der Maschine; .NET 10 und 11 geprüft ([Geprüfte Kombinationen](#geprüfte-kombinationen)) |
| cpp | `CMakeLists.txt`, `CMakePresets.json`, `meson.build`, `*.cpp`, `*.hpp` | clang-tidy über die Compile-Kommandos (`run-clang-tidy`); clang-format auf der bearbeiteten Datei | der CMake-Build | ctest, nach dem Build, wo `CMakeLists.txt` Tests einschaltet | gcovr, 100 % der Zeilen | Code-Graph | Ebenenkarte | braucht ein konfiguriertes `build/` (`CMakeCache.txt`); ein `CMakeLists.txt`, `CMakePresets.json` oder `meson.build` in der Wurzel macht die Wurzel zum einzigen Bereich; C++17, 20 und 23 geprüft mit GCC auf Linux ([Geprüfte Kombinationen](#geprüfte-kombinationen)); keine Compiler-Version festgelegt |
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

Die C#-Einträge nennen kein Werkzeug: Format, Build und Test bringt das .NET SDK mit, und `csharp-xunit` legt seine Testpakete selbst fest (xunit.v3.mtp-off 4.0.3, xunit.runner.visualstudio 4.0.1, Microsoft.NET.Test.Sdk 18.11.1, coverlet.collector 10.1.0). Es ist xunit.v3 ohne Microsoft.Testing.Platform: Das Paket `xunit.v3` bringt ab 4.0 MTP v2 mit, und `dotnet test` verweigert ab dem .NET-10-SDK den VSTest-Weg, den die Vorgabe (`--collect "XPlat Code Coverage"`) geht. Ein Lauf lässt die Projekte auf die .NET-Version des Eintrags zielen (`net10.0`, `net11.0`).

Eine C++-Version ist ein ISO-Standard. Auch die C++-Einträge nennen kein Werkzeug: CMake und GCC bringt der Linux-Runner mit, clang-tidy 23.1.4, clang-format 23.1.5 und gcovr 8.8 kommen über `uv tool install` (das Wheel von clang-tidy nennt sein `run-clang-tidy` `run-clang-tidy.py`; das Setup legt einen Link unter dem Namen an, den die Vorgabe aufruft), und `cpp-cmake-gtest` holt GoogleTest 1.18.0 selbst, festgelegt über die Release-URL und SHA-256. Ein Lauf konfiguriert `build/` mit dem Standard des Eintrags (`-DCMAKE_CXX_STANDARD=23`). C++ läuft nur auf Linux: Der Windows-Runner bringt MSVC mit, das die GCC-Coverage-Flags des Fixtures nicht nimmt. `godot4-cpp` hält eine GDExtension im Layout von godot-cpp (`project/` und `src/`).

Die Liste führt jede Go-, Python-, Godot- und .NET-Version, die ihr Upstream noch pflegt, nicht nur die neueste. C++ hat keine maschinenlesbare Quelle für seine Standards: Der Scout führt den nächsten selbst, als Vorabversion `26-draft`, bis ISO ihn veröffentlicht und er von Hand in die Liste wandert; ein Ende der Pflege meldet er für C++ nicht, denn auch ein Standard verlässt die Liste von Hand. Jeden Montag fragt der Kompatibilitäts-Scout (`loomux dev compat scout`) upstream nach dreierlei:

- **Eine neue stabile Version:** Er fährt sie als Probe und öffnet ein Issue mit dem Eintrag, der hinzukommt.
- **Eine Alpha, Beta oder ein Release Candidate der nächsten Version:** Er fährt sie vorab als Probe. Ein Fehlschlag öffnet nur ein Issue und blockiert keinen Pull Request; ein Pre-Release kommt nie in die Liste, das Release zieht nach, sobald die Version stabil ist.
- **Eine gelistete Version nach Ende der Pflege:** Die Antwort kommt von [endoflife.date](https://endoflife.date) (`isEol`). Die Version fällt aus der Liste. Der Pull Request, der sie austrägt, trägt mindestens `release:patch` und nennt im `## Changelog` unter `Removed` die letzte loomux-Version, die sie noch unterstützt, etwa: „Python 3.10 is no longer supported; v1.19.2 is the last release that supports it.“

## Was der Code-Graph liest

Der Graph (`loomux graph`, die Lane `graph`, `graph_*` über MCP) deckt Go,
Python, GDScript, C# und C++ ab. TypeScript steht auf der
[Roadmap](../../README.de.md#roadmap).

| Sprache | Liest | Löst auf | Lässt aus |
|---|---|---|---|
| Go | Pakete, Typen, Funktionen, Methoden | Aufrufe über Pakete des Moduls, Importe | typgeprüfte Aufrufe (`go/types`) |
| Python | Module, Klassen, Funktionen, Methoden | Importe, Aufrufe, Basisklassen | `.pyi`-Stubs |
| GDScript | Skripte, innere Klassen, Funktionen, Signale; Szenen, Ressourcen, `project.godot`, `.uid`-Dateien | `res://`- und `uid://`-Pfade, `class_name`, Autoloads (Szenen-Autoloads über ihr Wurzelskript), geerbte Aufrufe, Signale per Name und als Wert, Szenenverbindungen, auch zu C#-Methoden, -Signalen und `[GlobalClass]`-Klassen | VisualScript, Skripte inline in einer Szene, Typen von lokalen Variablen und Feldern |
| C# | Dateien, Typen, Methoden, Konstruktoren, Eigenschaften, Indexer, `[Signal]`-Delegates; Syntax bis C# 14 vollständig | `using`, Basen (`extends`/`implements`), Aufrufe, Konstruktoren | Typen lokaler Variablen, Methoden in einer zweiten Datei einer `partial`-Klasse für Szenenverbindungen; aus C# 15: `union` (kein Typknoten), `closed record class` (kein Knoten), ein Typ mit Erweiterungs-Indexer (kein Knoten), `break`/`continue` mit Marke (Fehler im Rumpf, die übrigen Mitglieder kommen an) |
| C++ | Dateien (`.cpp`, `.h`, `.hpp` und Verwandte), Klassen, Structs, Unions, Enums, Funktionen und Methoden mit Rumpf; GDExtension-Klassen (`GDCLASS`) und ihre Bindungen; Syntax von C++17, 20 und 23 | `#include`, Basisklassen, Aufrufe per Name oder qualifiziertem Namen; GDScript-`extends` und `.new()`, Knotentypen und Verbindungen in Szenen zu GDExtension-Klassen | den Präprozessor (Makros bleiben unexpandiert, jeder `#if`-Zweig wird gelesen), Typen und Überladungen (Member-Aufrufe `x.f()` bleiben offen), `.c`-Dateien, ein Signal, das eine Basisklasse einer GDExtension-Klasse bindet (eine Szenenverbindung findet nur die Klasse selbst), ein Mitglied, das nur die C++-Klasse hat, auf einem Knoten mit Skript und Typ; aus C++23: `if consteval` (die Funktion damit bekommt keinen Knoten, und die Definition vor der schließenden Klammer ihres Namespace geht verloren; was folgt, kommt ohne Namespace und Owner an; eine so abgeschnittene Datei wird erst ab einer Zeile in Spalte 0 neu gelesen, und ein Namespace oder eine Klasse, die am Schnitt offen war, kostet eine Definition) |

Die vollständigen Regeln stehen in der
[CLI-Referenz](cli-reference.md#loomux-graph-build---root-pfad---no-reuse).

## Stacks ohne Sprache

`wiki` und `wiki-sources` prüfen das Wiki eines Projekts, `project` führt
Befehle aus, die ein Projekt selbst nennt; keiner davon liest Code einer
Sprache. Siehe [Konfiguration](configuration.md#stacks).
