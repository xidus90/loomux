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
| cpp | `CMakeLists.txt`, `CMakePresets.json`, `meson.build`, `*.cpp`, `*.hpp` | clang-tidy; clang-format auf der bearbeiteten Datei | der CMake-Build | ctest, wo `CMakeLists.txt` Tests einschaltet | gcovr, 100 % der Zeilen | — | — | braucht ein konfiguriertes `build/` (`CMakeCache.txt`); keine Compiler-Version festgelegt |
| shell | `*.sh`, `.shellcheckrc` | shellcheck, nur auf der bearbeiteten Datei | — | — | — | — | — | keine Version festgelegt |
| sql | `.sqlfluff` | sqlfluff | — | — | — | — | — | keine Version festgelegt |
| rust | `Cargo.toml` | clippy, `cargo fmt` | — | — | — | — | — | keine Version festgelegt |

„Keine Version festgelegt“ heißt: loomux ruft die Version des Werkzeugs auf,
die das Projekt oder die Maschine mitbringt, und hat keine gegen eine andere
geprüft.

## Geprüfte Kombinationen

Jede Zeile ist eine Kombination, die loomux vor einem Major-Release und bei jeder Änderung der Liste wirklich fährt: die Laufzeit, die Addons und die Werkzeuge in den genannten Versionen, über ein Fixture-Projekt, auf Linux und Windows. Die Tabelle ist `internal/compat/compat.toml`; ein Test hält sie Zelle für Zelle gegen diese Seite und gegen das Feld `proven` jeder Framework-Zeile ([GDScript](stacks/gdscript.md)). Wie die Läufe arbeiten, steht in der [CLI-Referenz](cli-reference.md#loomux-dev-compat-matrixrunscout-flags), welche Sperre sie vor einem Release setzen in [Releases](releases.md#wie-ein-pull-request-zum-release-wird).

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
| python | python | 3.15 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.15 | python-pyright | — | coverage 7.16.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| go | go | 1.26 | go-module | — | — | lint, test, coverage, graph |
| go | go | 1.27 | go-module | — | — | lint, test, coverage, graph |

## Was der Code-Graph liest

Der Graph (`loomux graph`, die Lane `graph`, `graph_*` über MCP) deckt Go,
Python und GDScript ab. TypeScript und C++ stehen auf der
[Roadmap](../../README.de.md#roadmap).

| Sprache | Liest | Löst auf | Lässt aus |
|---|---|---|---|
| Go | Pakete, Typen, Funktionen, Methoden | Aufrufe über Pakete des Moduls, Importe | typgeprüfte Aufrufe (`go/types`) |
| Python | Module, Klassen, Funktionen, Methoden | Importe, Aufrufe, Basisklassen | `.pyi`-Stubs |
| GDScript | Skripte, innere Klassen, Funktionen, Signale; Szenen, Ressourcen, `project.godot`, `.uid`-Dateien | `res://`- und `uid://`-Pfade, `class_name`, Autoloads (Szenen-Autoloads über ihr Wurzelskript), geerbte Aufrufe, Signale per Name und als Wert, Szenenverbindungen | C#, VisualScript, Skripte inline in einer Szene, Typen von lokalen Variablen und Feldern |

Die vollständigen Regeln stehen in der
[CLI-Referenz](cli-reference.md#loomux-graph-build---root-pfad---no-reuse).

## Stacks ohne Sprache

`wiki` und `wiki-sources` prüfen das Wiki eines Projekts, `project` führt
Befehle aus, die ein Projekt selbst nennt; keiner davon liest Code einer
Sprache. Siehe [Konfiguration](configuration.md#stacks).
