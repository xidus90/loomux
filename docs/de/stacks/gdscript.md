# GDScript: Frameworks, die loomux fährt

Für GDScript-Projekte fährt loomux Frameworks von Drittanbietern für Tests, Abdeckung und Lint. Diese Seite nennt, welche das sind, woran loomux sie erkennt und welche Godot-Linie jedes trägt.

Die Tabelle stammt aus `internal/verify/frameworks.toml`. Ein Test vergleicht jede Tabellenzeile Zelle für Zelle mit den Daten, auf Englisch und auf Deutsch, damit keine von beiden abweichen kann.

Ein Framework, das in der Tabelle fehlt, ist nicht verboten. loomux fährt, was es findet, und sagt, was es über eine Fassung, die es nicht kennt, nicht weiß.

„Erkannt an“ gibt das Erkennungsmuster als `Datei:Text` wieder: Die Datei muss den Text enthalten, als Teilkette. Wo die Zeile erprobt ist, ist es das Muster, das das `when_found` der Lane trägt; bei einer unerprobten Zeile ist es, was eine Erkennung verwenden würde, denn noch erkennt keine Lane dieses Framework — loomux fährt nur, was es gefahren hat. Ein Test hält die Daten und die Presets in beide Richtungen gegeneinander, sodass weder ein erprobtes Muster außerhalb seiner Lane noch ein unerprobtes innerhalb einer Lane durchgeht.

Die Zelle ist leer, wo die Lane keine Erkennung braucht: Die Lint-Lane läuft auf jedem GDScript-Projekt, also nennt keine der beiden gdtoolkit-Zeilen ein Muster.

Unter Godot 3 liest gdformat (gdtoolkit 3) keine Konfiguration und kann `addons/` nicht auslassen, sodass ein Projekt mit Addons im Baum an `gdformat --check` scheitert; das gdtoolkit von Godot 4 beachtet `excluded_directories` in `.gdformatrc`.

| Framework | Art | Erkannt an | Trägt | Quelle |
|---|---|---|---|---|
| gdUnit4 | test | `plugin.cfg:name="gdUnit4"` | Godot 4.3 bis 4.8 | https://github.com/godot-gdunit-labs/gdUnit4 |
| NanoCoverage | coverage | `plugin.cfg:name="NanoCoverage"` | Godot 4 | https://github.com/IgorBayerl/nano-coverage-godot |
| gdtoolkit4 | lint |  | Godot 4 | https://github.com/scony/godot-gdscript-toolkit |
| GUT | test | `plugin.cfg:name="Gut"` | GUT 7.x trägt Godot 3, 9.x trägt Godot 4; das Muster unterscheidet sie nicht | https://github.com/bitwes/Gut |
| gdtoolkit3 | lint |  | Godot 3 | https://github.com/scony/godot-gdscript-toolkit |
| godot-code-coverage (unerprobt) | coverage | `plugin.cfg:name="Coverage"` | auf Godot 3.5 dokumentiert, 3.6 unbestätigt | https://github.com/jamie-pate/godot-code-coverage |

## Unerprobte Zeilen

Eine als unerprobt markierte Zeile hat loomux noch nicht gegen eine echte Installation der genannten Linie gefahren. Offen ist je Zeile:

- godot-code-coverage (unerprobt): eine Abdeckungs-Lane durch loomux auf Godot 3.5 fahren und eine auf 3.6.

## Rust-Extensionen (godot-rust)

`loomux check godot-suite` und `loomux godot rescan` bauen eine godot-rust-Extension des Projekts, bevor sie Godot starten, weil die Suiten und der Import die gebaute Bibliothek laden. Die Lanes des Stacks `rust` ordnen sich nicht gegen eine GDScript-Lane (ein `after` bleibt in einem Stack und einem Bereich); darum liegt der Bau in diesen beiden Befehlen.

- **Was zählt**: jede `.gdextension` im Projekt, deren `[configuration]` `entry_symbol = "gdext_rust_init"` hat. Eine C++-Extension oder das Addon eines Dritten hat ein anderes Symbol und bleibt unberührt.
- **Was gebaut wird**: loomux wählt keinen Plattformschlüssel. Godot wählt eine Bibliothek über Feature-Tags, und echte Projekte schreiben `linux.editor.x86_64`, Schlüssel ohne Architektur und Ähnliches. Stattdessen baut jeder `[libraries]`-Pfad, der in `<verz>/target/<profil>/` zeigt (ein `res://`-Pfad wird gegen das Verzeichnis von `project.godot` aufgelöst), dieses Profil einmal für den Host: `debug` als `cargo build`, `release` als `cargo build --release`, jedes andere als `cargo build --profile <profil>`. `linux.debug`, `linux.editor` und `windows.debug` auf `target/debug` sind ein Bau. Ein Pfad über ein Target-Triple (`target/wasm32-unknown-emscripten/debug/`) ist kein Host-Bau und wird übergangen. Ein Eintrag, dessen Schlüssel den Feature-Tag `release` trägt (ein durch Punkte getrenntes Schlüsselsegment gleich `release` oder `template_release`: `linux.release.x86_64`, `windows.release`), wird ebenfalls übergangen, weil die Suite ohne Fenster und der Editor-Import die Debug- oder Editor-Bibliothek laden und ein Release-Bau nur die Kosten des kalten Baus verdoppelte. Schlüssel mit `debug`, `editor` oder ohne Bau-Tag zählen, und das Profil bleibt das aus dem Pfad. Die übliche godot-rust-Vorlage mit einem `debug`- und einem `release`-Pfad je Plattform baut also nur `debug`. Ist jeder Rust-Eintrag nur `release`, wird nichts gebaut, und der Lauf sagt das in einem Hinweis.
- **Wo**: im nächsten Verzeichnis über `<verz>/target`, das eine `Cargo.toml` hält, gesucht bis zur Repository-Wurzel und nicht weiter (dem nächsten Verzeichnis mit `.git`, auch als Datei in einem Worktree; ohne git-Checkout dem Verzeichnis um das Godot-Projekt, sodass `godot/` und `rust/` nebeneinander gefunden werden); bei einem Workspace ist das seine Wurzel, Cargo baut also jedes Mitglied. Ein `target/` außerhalb des Repositorys wird nicht gebaut (der Lauf sagt das in einem Hinweis), und eine `Cargo.toml` über dem Repository wird nie benutzt.
- **Bekannte Lücke**: Ein eigenes Zielverzeichnis schickt Cargos Ausgabe dorthin, wohin die `.gdextension` nicht zeigt; Godot lädt dann ohne Warnung die alte Bibliothek am eingetragenen Pfad. Ist `CARGO_TARGET_DIR` in der Umgebung gesetzt, sagt der Lauf das in einem Hinweis. Ein `build.target-dir` in `.cargo/config.toml` wirkt genauso und gibt keinen Hinweis: loomux liest Cargos Konfiguration nicht.
- **Fehler**: Ein fehlschlagender `cargo build` beendet den Lauf mit der Ausgabe von Cargo, bevor eine Engine startet. Fehlt `cargo` im `PATH`, meldet der Lauf `cargo is not on PATH` als Befund, so wie derselbe Befehl ein fehlendes Godot-Binary meldet.

### Zeitbudget

Der Bau läuft im Prozess `loomux check godot-suite` und zählt darum gegen das Zeitlimit der Lane, `[verify].timeout`, vorgabemäßig 600 s. Gemessen am 2026-11-15 mit einem Crate, das nur von `godot = "=0.5.5"` abhängt (Cargo 1.99.1, 16 logische Kerne, kaltes `target/`): `cargo build` 42 s (`cargo build --release` brauchte 59 s, ein Release-Eintrag wird aber nicht gebaut). Das passt auf einer solchen Maschine in die Vorgabe und lässt Raum für die Suite. Ein größeres Crate oder eine Maschine mit wenigen Kernen kann darüber liegen; dann `[verify].timeout` erhöhen oder einmal vorab `cargo build` fahren, damit die Lane nur noch baut, was sich geändert hat. Cargo baut inkrementell, ein warmes `target/` kostet Sekunden.

### Windows: die Bibliothek ist in Benutzung

Windows überschreibt keine DLL, die ein laufender Prozess geladen hat. Gemessen unter Windows 11 mit dem Headless-Editor (`--headless --editor`) und einem Headless-Spiel (`--headless --path`) auf Godot 4.7.1 (Mono-Konsolenbuild) mit godot-rust 0.5.5 und gdUnit4 6.2.2. Nicht gemessen: der Editor mit Fenster, andere Godot-Versionen, Hot Reload.

- **Der Headless-Editor hielt die DLL nicht.** `godot --headless --editor` lud eine Kopie (`~demo.dll` neben `demo.dll`); `cargo build` und `loomux check godot-suite` bauten und testeten also, während der Editor das Projekt offen hatte. Die geladene Datei hing nicht von `reloadable` ab: mit `true` wie mit `false` dasselbe. Was der Schlüssel für Hot Reload bewirkt, ist ungetestet.
- **Ein Headless-Spiel hielt sie.** `godot --headless --path <Projekt>` (ohne `--editor`) lud `demo.dll` selbst, und `cargo build` scheiterte dann mit `failed to remove file ... demo.dll` (`os error 5`), ebenfalls bei beiden Werten von `reloadable`. Ein laufendes Spiel vor der Prüfung beenden.
