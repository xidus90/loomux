# GDScript: Frameworks, die loomux fährt

Für GDScript-Projekte fährt loomux Frameworks von Drittanbietern für Tests, Abdeckung und Lint. Diese Seite nennt, welche das sind, woran loomux sie erkennt und welche Godot-Linie jedes trägt.

Die Tabelle stammt aus `internal/verify/frameworks.toml`. Ein Test vergleicht jede Tabellenzeile Zelle für Zelle mit den Daten, auf Englisch und auf Deutsch, damit keine von beiden abweichen kann.

Ein Framework, das in der Tabelle fehlt, ist nicht verboten. loomux fährt, was es findet, und sagt, was es über eine Fassung, die es nicht kennt, nicht weiß.

„Erkannt an“ gibt das Erkennungsmuster als `Datei:Text` wieder: Die Datei muss den Text enthalten, als Teilkette. Wo die Zeile erprobt ist, ist es das Muster, das das `when_found` der Lane trägt; bei einer unerprobten Zeile ist es, was eine Erkennung verwenden würde, denn noch erkennt keine Lane dieses Framework — loomux fährt nur, was es gefahren hat. Ein Test hält die Daten und die Presets in beide Richtungen gegeneinander, sodass weder ein erprobtes Muster außerhalb seiner Lane noch ein unerprobtes innerhalb einer Lane durchgeht.

Die Zelle ist leer, wo die Lane keine Erkennung braucht: Die Lint-Lane läuft auf jedem GDScript-Projekt, also nennt keine der beiden gdtoolkit-Zeilen ein Muster.

| Framework | Art | Erkannt an | Trägt | Quelle |
|---|---|---|---|---|
| gdUnit4 | test | `plugin.cfg:name="gdUnit4"` | Godot 4.3 bis 4.8 | https://github.com/godot-gdunit-labs/gdUnit4 |
| NanoCoverage | coverage | `plugin.cfg:name="NanoCoverage"` | Godot 4 | https://github.com/IgorBayerl/nano-coverage-godot |
| gdtoolkit4 | lint |  | Godot 4 | https://github.com/scony/godot-gdscript-toolkit |
| GUT (unerprobt) | test | `plugin.cfg:name="Gut"` | GUT 7.x trägt Godot 3, 9.x trägt Godot 4; das Muster unterscheidet sie nicht | https://github.com/bitwes/Gut |
| gdtoolkit3 (unerprobt) | lint |  | Godot 3 | https://github.com/scony/godot-gdscript-toolkit |
| godot-code-coverage (unerprobt) | coverage | `plugin.cfg:name="Coverage"` | auf Godot 3.5 dokumentiert, 3.6 unbestätigt | https://github.com/jamie-pate/godot-code-coverage |

## Unerprobte Zeilen

Eine als unerprobt markierte Zeile hat loomux noch nicht gegen eine echte Installation der genannten Linie gefahren. Offen ist je Zeile:

- GUT (unerprobt): seine Tests durch loomux auf einem Godot-3-Projekt fahren und auf einem Godot-4-Projekt, um zu sehen, welche Linie eine gefundene Installation trägt.
- gdtoolkit3 (unerprobt): die Lint-Lane durch loomux auf einem Godot-3-Projekt mit der 3.x-Fassung des Toolkits fahren.
- godot-code-coverage (unerprobt): eine Abdeckungs-Lane durch loomux auf Godot 3.5 fahren und eine auf 3.6.
