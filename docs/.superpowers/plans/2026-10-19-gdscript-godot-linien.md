# Plan: Code-Graph für Godot 3.x und 4.x

Spec: `docs/.superpowers/specs/2026-10-19-gdscript-godot-linien-design.md`.
Jede Aufgabe testgetrieben: erst ein Resolver-Test in
`internal/code/resolve/gdscript_test.go` (echter Extraktor, echte Auflösung),
der rot ist, dann der Code, dann `go test ./internal/code/...` und die
Coverage je Funktion.

1. **Basis-Aufruf `.f()`** (3.x): `base_call` wird zum Aufruf mit Empfänger
   `super`.
2. **Signale über Strings:** `connect`/`disconnect`/`is_connected` mit
   Literal, mit und ohne Empfänger; `X.emit_signal("s")`; `yield(obj, "s")`;
   `await s` und `await X.s`. Der Resolver kennt `is_connected` und `await`
   als Signalverben.
3. **Methoden als Wert:** Referenzen auf Methoden und Signale der Klasse,
   verdeckt von Parametern und lokalen Variablen; `self.m` ohne Aufruf;
   `funcref(self, "m")`, `Callable(self, "m")`, `connect("s", self, "m")`.
4. **Sofortaufrufe über Namen:** `call`/`call_deferred`/`callv`/`rpc`/
   `rpc_unreliable` (Name als erstes Argument), `rpc_id`/`rpc_unreliable_id`
   (als zweites), mit und ohne Empfänger; `m.call()`, `m.call_deferred()`,
   `m.callv()`, `m.rpc()`, `m.rpc_id()`.
5. **Eigenschaften:** `setget` und `get = …, set = …` referenzieren ihre
   Methoden.
6. **Kopf:** `extends "res://a.gd".Inner`; `@abstract func f()` ohne Rumpf
   bekommt einen Knoten.
7. **Szenen:** Knotenpfade, Skripte, Instanzen, `[connection]`; ID als Zahl
   und als String; Wurzelskript als Alias `.` der Szene.
8. **Autoload als Szene** bindet das Wurzelskript.
9. **UIDs:** `.uid`-Dateien und `uid=` im Kopf; Auflösung von `uid://` in
   `target`; `.uid` in `sourceset` und den Extensions des Extraktors.
10. **Version** `gdscript/4`, Paketkommentare, Doku (`docs/en`,
    `docs/de` als Patch für den Menschen, Wiki `topics/code-graph.md`),
    README-Roadmap, Abnahme am 3.x- und 4.x-Probeprojekt im Scratchpad.
