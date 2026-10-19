# GDScript: frameworks loomux drives

For GDScript projects loomux drives third-party frameworks for tests, coverage and lint. This page lists which ones, how loomux recognises each, and which Godot line each carries.

The table comes from `internal/verify/frameworks.toml`. A test compares every table row with the data, cell by cell, in English and in German, so neither can drift from the other.

A framework missing from the table is not forbidden. loomux drives what it finds and says what it does not know about a version it has not seen.

"Found by" gives the detection pattern as `file:text`: the file must contain the text, as a substring. Where the row is proven it is the pattern the lane's `when_found` carries; on an unproven row it is what a detection would use, since no lane detects that framework yet — loomux drives only what it has run. A test holds the data and the presets against each other both ways, so neither a proven pattern outside its lane nor an unproven one inside a lane passes.

The cell is empty where the lane needs no detection: the lint lane runs on any GDScript project, so neither gdtoolkit row names a pattern.

| Framework | Kind | Found by | Carries | Source |
|---|---|---|---|---|
| gdUnit4 | test | `plugin.cfg:name="gdUnit4"` | Godot 4.3 to 4.8 | https://github.com/godot-gdunit-labs/gdUnit4 |
| NanoCoverage | coverage | `plugin.cfg:name="NanoCoverage"` | Godot 4 | https://github.com/IgorBayerl/nano-coverage-godot |
| gdtoolkit4 | lint |  | Godot 4 | https://github.com/scony/godot-gdscript-toolkit |
| GUT | test | `plugin.cfg:name="Gut"` | GUT 7.x carries Godot 3, 9.x carries Godot 4; the pattern does not tell them apart | https://github.com/bitwes/Gut |
| gdtoolkit3 (unproven) | lint |  | Godot 3 | https://github.com/scony/godot-gdscript-toolkit |
| godot-code-coverage (unproven) | coverage | `plugin.cfg:name="Coverage"` | documented for Godot 3.5, unconfirmed on 3.6 | https://github.com/jamie-pate/godot-code-coverage |

## Unproven rows

A row marked unproven has not been run by loomux against a real installation of the line it names. Still to do for each:

- gdtoolkit3 (unproven): run the lint lane through loomux on a Godot 3 project with the 3.x release of the toolkit.
- godot-code-coverage (unproven): run a coverage lane through loomux on Godot 3.5, and one on 3.6.
