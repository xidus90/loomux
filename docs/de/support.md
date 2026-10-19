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
| go | `go.mod` | `go vet`, `gofmt` | — | `go test` | `gocover`, 100 % je Funktion | Code-Graph | Ebenenkarte | Go-Syntax bis 1.27, der `go/parser` der Toolchain, mit der loomux gebaut ist; eine Go-Datei, die nicht parst, lässt den Graph-Build scheitern |
| python | `pyproject.toml`, `requirements.txt`, `uv.lock`, `manage.py` | ruff | mypy; pyright oder das mypy des Projekts, wo konfiguriert | pytest | coverage.py | Code-Graph | Ebenenkarte | Python 3, über `uv`; das mypy des Presets braucht 1.16 oder neuer; Werkzeugversionen festlegbar mit [`tools`](configuration.md#stacks); keine Minor-Version einzeln geprüft |
| typescript | `tsconfig.json` neben `package.json`, `biome.json` | eslint; biome, wo konfiguriert | `tsc` | vitest, wo `package.json` es nennt | vitest mit Coverage | — | — | `.js` und `.jsx` zählen als TypeScript; keine Version festgelegt |
| vue | `*.vue`, `vue.config.js` | — | `vue-tsc` | — | — | — | — | keine Version festgelegt |
| svelte | `*.svelte`, `svelte.config.js` | — | `svelte-check` | — | — | — | — | keine Version festgelegt |
| css | `*.css`, `*.scss`, `.stylelintrc*` | stylelint | — | — | — | — | — | keine Version festgelegt |
| html | `*.html`, `.htmlhintrc` | htmlhint | — | — | — | — | — | keine Version festgelegt |
| gdscript | `project.godot`, `.gdlintrc` | gdlint, gdformat (gdtoolkit) | — | gdUnit4 über `check godot-suite`, wo installiert | NanoCoverage über `check linecov`, wo installiert | Code-Graph | Ebenenkarte | Graph: Godot 3.x und 4.x, geprüft an 3.x, 4.2 und 4.7; test und coverage: Godot 4.3 bis 4.8; lint: Godot 4, Godot 3 unbelegt, gdtoolkit festlegbar mit [`tools`](configuration.md#stacks) ([Einzelheiten](stacks/gdscript.md)) |
| cpp | `CMakeLists.txt`, `CMakePresets.json`, `meson.build`, `*.cpp`, `*.hpp` | clang-tidy; clang-format auf der bearbeiteten Datei | der CMake-Build | ctest, wo `CMakeLists.txt` Tests einschaltet | gcovr, 100 % der Zeilen | — | — | braucht ein konfiguriertes `build/` (`CMakeCache.txt`); keine Compiler-Version festgelegt |
| shell | `*.sh`, `.shellcheckrc` | shellcheck, nur auf der bearbeiteten Datei | — | — | — | — | — | keine Version festgelegt |
| sql | `.sqlfluff` | sqlfluff | — | — | — | — | — | keine Version festgelegt |
| rust | `Cargo.toml` | clippy, `cargo fmt` | — | — | — | — | — | keine Version festgelegt |

„Keine Version festgelegt“ heißt: loomux ruft die Version des Werkzeugs auf,
die das Projekt oder die Maschine mitbringt, und hat keine gegen eine andere
geprüft.

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
