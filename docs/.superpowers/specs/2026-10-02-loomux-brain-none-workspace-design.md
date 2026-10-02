# `init --brain=none` registriert das Projekt als Workspace

Stand 2026-10-02. Entschieden mit dem Nutzer im Brainstorming desselben Tages.

## Problem

`loomux init --yes --brain=none` lässt ein Projekt ohne Registry-Eintrag. Die
Registrierung ist der Teil `area` des Brain-Moduls (`internal/setup/parts.go`,
Zeile 41), und mit `--brain=none` fällt sie weg. Der Schreibwächter kennt aber
nur Bäume, die die Registry nennt: `writableRoots` öffnet das Wiki eines nicht
schreibgeschützten Bereichs und `path`, wenn der Bereich `workspace = true`
hat (`internal/brain/guard/guard.go`, Zeilen 150–169); die Wurzel, die der Wirt
mit `--root` übergibt, öffnet er nicht. So beschreibt es auch die Doku
(`docs/en/configuration.md`, Zeile 991: „lets tools write where the registry
declares an area“).

Folge, gemessen am 2026-10-02 mit loomux 6.1.0: In `iam_backend`,
`iam_frontend` und `iam_workers`, alle mit `--brain=none` eingerichtet,
verweigert der Wächter jeden Write mit „lies outside every writable tree“; in
`ecoflow` mit Bereich geht derselbe Write durch. Die 4e-Umstellungsspec
(`2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md`, Zeilen 266–268)
wollte für diese Projekte „keinen eigenen Bereich und kein Wiki“ und zugleich
„Hooks, Guards und Graph wie bei allen anderen“; beides zugleich geht mit dem
heutigen `init` nicht.

Überbrückt ist es seit dem 2026-10-02 von Hand: der Nutzer hat die drei
Projekte als Bereiche ohne `wiki` mit `workspace = true` in die Registry
eingetragen; danach lässt der Wächter dort Writes durch (Exit 0, gemessen).

## Entscheidung

1. **Ein neuer `init`-Teil `workspace` im Modul Hooks.** Er gehört zum
   Wächter, nicht zum Brain: ohne ihn schreibt kein Agent im Projekt.
2. **Er läuft nur, wenn das Brain-Modul aus ist.** Ist Brain an, registriert
   wie bisher der Teil `area` den Bereich mit Wiki; `workspace` bleibt aus,
   damit nichts doppelt registriert wird.
3. **Er schreibt nur die Registry**, einen Bereich mit `scope`, `path` und
   `workspace = true`, **ohne `wiki`**, über `config.AddArea` unter der
   Registry-Sperre, wie `area add` es tut. Er schreibt nichts in
   `.loomux/config.toml`, legt kein Wiki-Gerüst an, keine Routing-Regel und
   keinen Merge-Hook.
4. **Dieselben Bedingungen wie `area`:** nur in einem frischen Projekt (kein
   loomux-Checkout), nur ohne `[area]` in `.loomux/config.toml` und nur, wenn
   die Registry an dieser Wurzel noch keinen Bereich hat. Ein vorhandener
   Eintrag bleibt unberührt, ein zweiter Lauf von `init` ändert nichts. Der
   Bereichsname ist derselbe wie bei `area` (`Choice.Scope`, sonst
   `project/<Verzeichnisname>`).
5. **Keine neue Form von `area add`** (Entscheidung des Nutzers: „nur init, so
   wenig wie möglich“). Ein Mensch, der so einen Bereich ohne `init` braucht,
   trägt ihn von Hand ein.
6. **Der Wächter ändert sich nicht.** Er öffnet `path` eines
   Workspace-Bereichs schon heute.

## Was ein Bereich ohne Wiki sonst bewirkt

Gelesen im Code von `origin/master` am 2026-10-02: Ein Bereich ohne `wiki`
wird von Index, Wartung und Prüfung übersprungen (`internal/brain/index/walk.go`
`OwnWikiPrefix` gibt nil, `check/run/run.go` und `check/run/targets.go`,
`check/house/federation.go`, `apply/resolve.go`). Der Plan hält das mit einem
Test fest (Reindex legt für den Bereich keine Sammlung an), statt sich auf das
Lesen zu verlassen.

## Tests

- `init --yes --brain=none` in einer frischen Testwelt plant den Teil
  `workspace` und schreibt einen Registry-Eintrag ohne `wiki` mit
  `workspace = true`; eine Write-Nutzlast im Projekt geht danach durch den
  Wächter (Exit 0), vorher nicht (Exit 2).
- Ein zweiter Lauf ändert die Registry nicht; ein schon vorhandener Eintrag an
  dieser Wurzel bleibt byte-gleich, und der Plan nennt den Grund als Notiz.
- Mit Brain an plant `init` `area` wie bisher und nicht `workspace`.
- `--dry-run` nennt die Aktion und schreibt nichts.
- Der Reindex legt für den Workspace-Bereich keine Sammlung an.
- Die Teile-Liste (`init --help`, Auswahl) nennt `workspace` im Modul Hooks.
- Aufgezeichnete Fälle unter `testdata/cases` bleiben byte-gleich, wo sie
  nicht `--brain=none` fahren; wo doch, wird der Unterschied im Plan benannt.

## Doku

- `docs/en/cli-reference.md` und `docs/de/cli-reference.md`, Abschnitt `init`:
  `--brain=none` schaltet das Brain-Modul ab, registriert das Projekt aber als
  Workspace ohne Wiki, damit der Schreibwächter den Projektbaum öffnet; der
  neue Teil in der Teile-Liste.
- `docs/en|de/configuration.md` beim Satz zur Registry als Quelle des
  Wächters: ein Bereich ohne `wiki` öffnet nur `path`.
- Die 4e-Umstellungsspec bekommt einen Nachtrag: „kein eigener Bereich“ heißt
  „kein Brain-Bereich“; die drei Projekte haben seit dem 2026-10-02 einen
  Workspace-Eintrag (von Hand, künftig durch `init`).

## Nicht Teil davon

- Die Projektwurzel ohne Registry-Eintrag schreibbar zu machen (verworfen: es
  öffnete jeden Ordner, in dem eine Sitzung startet).
- `area add --no-wiki`.
- Die übrigen Befunde der Welle (Profil-Arten ohne Prüfbares, `verify.profiles`
  per `config set`, Repo-Wurzel als Wiki, `dev bench cases` ohne Hooks).
