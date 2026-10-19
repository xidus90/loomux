---
status: active
---

# Code-Graph für Godot 3.x und 4.x

## Ziel

Der GDScript-Graph liest bisher die Syntax von Godot 4 gut und die von
Godot 3 nur so weit, wie sie zufällig mit 4 übereinstimmt. Ziel: Jede noch
gepflegte Version beider Linien (3.x und 4.0 bis 4.7) ergibt einen Graphen
mit denselben Kanten für dieselbe Absicht, gleich wie die Version sie
schreibt.

## Befund (Probe am 2026-10-19, gotreesitter v0.55.1)

Die Grammatik parst alle geprüften Formen beider Linien ohne ERROR-Knoten
(`tool`, `export(...)`, `onready`, `setget`, `remote`/`master`/`puppet`,
`yield`, `._ready()`, `class_name X, "icon"`, `@export_*`, `@abstract`,
`static var`, variadische Parameter, typisierte Arrays und Dictionaries,
Lambdas, `await`, `uid://`, Szenen in `format=2` und `format=3`). Was fehlt,
liegt im Extraktor und im Resolver:

| Form | Linie | heute |
|---|---|---|
| `._ready()` (Aufruf der Basis) | 3 | keine Kante |
| `connect("s", self, "m")`, `disconnect`, `is_connected` | 3, String-Form auch 4 | keine Kante |
| `X.connect("s", …)`, `X.emit_signal("s")` | 3 | nur Kante auf X |
| `yield(obj, "s")` | 3 | keine Kante |
| `await s`, `await X.s` | 4 | keine Kante auf das Signal |
| `setget set_a, get_a`; `get = _g, set = _s` | 3; 4 | keine Kante |
| `funcref(self, "m")`, `Callable(self, "m")` | 3; 4 | keine Kante |
| `call("m")`, `call_deferred`, `callv`, `rpc`, `rpc_id`, `rpc_unreliable(_id)` | beide | keine Kante |
| `m.call()`, `m.call_deferred()`, `m.bind()`, `var f = m`, `s.connect(m)` | 4 | keine Kante auf m |
| `extends "res://a.gd".Inner` | beide | keine Kante |
| `@abstract func f()` ohne Rumpf | 4.5+ | kein Knoten |
| `[connection signal from to method]` in Szenen | beide | keine Kante |
| `ExtResource( 1 )` (Ganzzahl-ID) / `ExtResource("1_ab")` | 3 / 4 | Skript eines Szenenknotens nicht gelesen |
| `uid://…` in Skripten, `run/main_scene`, Autoloads | 4.0+ (Skripte ab 4.4) | keine Kante |
| Autoload, der eine Szene ist | beide | bindet die Szene, nicht ihr Skript |

## Entscheidungen

1. **Keine Versionserkennung.** Jede Form hat in beiden Linien höchstens eine
   Bedeutung (`._ready()` ist in 4 ungültig, `connect` mit drei Argumenten
   nur in 3). Der Graph entscheidet nach der Form, nicht nach
   `config_version`. Ein Projekt, das während der Migration beide Formen
   mischt, bekommt beide Kanten.
2. **Verdrahtung ist `references`, nicht `calls`.** Wer ein Signal verbindet
   oder eine Methode als Wert weitergibt, ruft sie nicht auf. `callers` und
   `blast` gehen über `references` ohnehin. `calls` bekommen nur die Formen,
   die sofort aufrufen: `call`, `call_deferred`, `callv`, `rpc*` und
   `m.call*()`.
3. **Ein Name, der eine Methode oder ein Signal der Klasse ist, ist eine
   Referenz darauf**, sobald kein Klassenname ihn bindet. Lokale Variablen
   und Parameter einer Funktion verdecken ihn (die Grenze: Schleifen- und
   `match`-Bindungen verdecken nicht).
4. **Strings zählen nur als Literal.** `connect(name, …)` mit einer Variablen
   ergibt keine Kante.
5. **Szenen:** Der Extraktor liest die Knotenpfade einer Szene, das Skript
   jedes Knotens (`script = ExtResource(id)`) und die Instanzen
   (`instance=ExtResource(id)`). Eine `[connection]` ergibt eine Referenz von
   der Szene auf die Methode im Skript des Zielknotens und auf das Signal im
   Skript des Quellknotens. Ein instanzierter Knoten ohne eigenes Skript
   nimmt das Wurzelskript der instanzierten Szene.
6. **UIDs:** Jede `.uid`-Datei (4.4+) und der Kopf jeder Szene und Ressource
   (`uid="uid://…"`) melden ihre UID; der Resolver ersetzt `uid://…` durch
   den Pfad. `.uid` wird gelesen, ergibt aber keinen Knoten.
7. **Autoload als Szene** bindet das Wurzelskript der Szene. Ein `preload`
   einer Szene bindet weiter die Szene: dort ist der Name eine `PackedScene`.
8. **Extraktorversion** `gdscript/4`: Jeder Graph eines Godot-Projekts wird
   einmal neu gebaut.

## Nicht im Umfang

Inline-Skripte (`[sub_resource type="GDScript"]`), C#, Typen über lokale
Variablen hinweg (`var p: Player = …; p.f()`), Signale über Ausdrücke
(`get_node("x").connect(…)`), `_global_script_classes` in der
`project.godot` von 3.x (die `class_name` der Skripte reicht).
