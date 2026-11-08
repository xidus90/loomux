# Versionsbreite: Go im Code-Graph und Pre-Releases im Scout

Stand 2026-11-08. Erste von zwei Sessions zur Versionsnorm; die zweite zieht
Python nach.

## Absicht

loomux soll nicht nur die neueste Version einer Sprache lesen, sondern jede,
die ihr Upstream noch pflegt, und die laufenden Pre-Releases (Alpha, Beta,
RC) der nächsten mit prüfen. Das ist als Norm in `AGENTS.md` (Abschnitt
Rules) festgehalten. Der Wunsch dahinter ist größere vertikale
Unterstützung: mehr Versionen je Sprache, nicht mehr Sprachen.

„Unterstützt“ heißt: Eine Datei mit der Syntax dieser Version wird gelesen
**und** die Sprachmittel, die sie einführt, ergeben die richtigen Kanten. Nur
fehlerfrei parsen reicht nicht.

## Ausgangslage (recherchiert 2026-11-08)

- Go pflegt die zwei jüngsten Hauptversionen: heute 1.26 und 1.27
  (go.dev/dl, `go1.27.3`, `go1.26.10`). Eine 1.28-Beta oder ein RC gibt es
  noch nicht; nach Gos Takt erscheint der erste RC um den Dezember.
- `internal/compat/compat.toml` führt Go 1.26 und 1.27 mit der Art `graph`,
  beide auf demselben Fixture `go-module` (`calc/`, drei `calls`-Kanten).
  `compat.yml` läuft jeden Eintrag auf Linux und Windows.
- **Der Graph-Teil dieser Einträge prüft keine Version.** `compat.yml` baut
  loomux mit der Toolchain aus `go.mod` (`go1.27.3`) und setzt die Go-Version
  des Eintrags erst danach auf („Go entries run with their own Go; loomux is
  built already“). `go/parser` ist in loomux eingebaut, also liest der Graph
  immer mit `go1.27.3`. Das ist für stabile Versionen richtig, denn genau das
  Binary wird ausgeliefert; es fehlt nur ein Fixture, das die Syntax der
  jeweiligen Version enthält.
- `compat-scout.yml` (montags) fragt Upstream nach Versionen, die die Liste
  nicht hat, lässt jede als Probe laufen und hält je Fund ein Issue.
  Pre-Releases verwirft er ausdrücklich (`scout.go`: „Patch releases and
  pre-releases never count“; `parseReleases` nimmt bei Go nur `stable`, bei
  Python nur `!pre_release`, bei GitHub nur `!prerelease` und bei Godot nur
  `-stable`).
- Ein Ende der Pflege erkennt der Scout nicht; eine Version bleibt in der
  Liste, bis jemand sie austrägt.
- Sprachänderungen laut Release Notes:
  - Go 1.26: `new(expr)`; ein generischer Typ darf sich in seiner
    Typparameterliste selbst nennen (`type Adder[A Adder[A]] interface …`).
  - Go 1.27: Methoden mit eigenen Typparametern; Schlüssel in
    Struct-Literalen dürfen beliebige Feldselektoren sein; die
    Funktions-Typinferenz gilt in allen Zuweisungskontexten.
- Ob der Extraktor (`internal/code/extract/golang`) diese Formen richtig
  verbindet, ist ungetestet. `extract.go:337` läuft über `ft.TypeParams` einer
  Funktion; ob eine generische Methode dort ankommt, ist offen.

## Ziele

1. **Go-Fixtures je gepflegter Version.** Jede gepflegte Go-Hauptversion hat
   in der Kompatibilitätsliste ein Fixture mit der Syntax, die sie eingeführt
   hat, und die erwarteten Kanten dazu (`expect.toml`). Was bisher nicht oder
   falsch verbunden wird, wird im Extraktor nachgezogen.
2. **Pre-Releases im Scout, für alle Laufzeiten.** Der Scout meldet Alpha,
   Beta und RC von Go, Python und Godot. Jeder Fund läuft als Probe und
   bekommt ein Issue, so wie heute eine neue stabile Version.
3. **Go-Pre-Release mit eigener Toolchain.** Eine Go-Probe einer
   Pre-Release-Version baut loomux mit genau dieser Toolchain, bevor sie den
   Graph prüft. Nur so prüft sie, ob der nächste Parser das Fixture liest.
4. **Ende der Pflege erkennen.** Der Scout meldet einen Listeneintrag, dessen
   Version das Upstream nicht mehr pflegt, damit er ausgetragen wird.
5. **Doku.** `docs/wiki/topics/code-graph.md` nennt die gelesenen
   Go-Versionen und den Weg der Pre-Releases; die Roadmap in `README.md` und
   `README.de.md` wird nachgezogen; die CLI-/Compat-Doku unter `docs/en` und
   `docs/de`, soweit sie Scout oder Liste beschreibt.

## Nicht-Ziele

- Python-Fixtures je Version und Lücken der Python-Grammatik: das ist die
  zweite Session. Der Scout-Umbau hier gilt aber schon für Python.
- Godot-Fixtures je Version: eigene spätere Arbeit.
- Releases mit einer RC-Toolchain bauen. Das veröffentlichte loomux zieht
  erst mit der fertigen Version nach (Entscheidung des Nutzers).

## Anti-Ziele

- **Ein Pre-Release blockiert keinen Pull Request.** Ein roter
  Pre-Release-Lauf erzeugt nur ein Issue (Entscheidung des Nutzers). Ein
  Pre-Release darf deshalb nie zum Pflichtcheck `compat` eines PRs werden.
- Kein Gate, das grün ist, ohne die Version zu prüfen (der heutige Zustand
  des Graph-Teils).
- Versionen nicht aus dem Gedächtnis: Liste und Scout stützen sich auf
  Upstream-Quellen.

## Festgelegt

- Stabile Go-Einträge bauen loomux weiter mit der Toolchain aus `go.mod`,
  denn das ist das ausgelieferte Binary.
- Der Scout wird in dieser Session für alle drei Laufzeiten umgebaut.
- Pre-Releases: Pipeline prüft vorab, Fehlschlag nur als Issue.

- **Ende der Pflege heißt Austragen, ohne Ausnahme.** Pflegt das Upstream
  eine Version nicht mehr, fliegt sie auch bei uns aus der Liste; das gilt
  auch für Godot 3. Der Pull Request, der sie austrägt, nennt im
  `## Changelog` unter `Removed` die letzte loomux-Version, die sie noch
  unterstützt (z. B. „Python 3.10 is no longer supported; v1.19.2 is the
  last release that supports it.“). So steht es auch in `AGENTS.md`.

## Meine Entscheidungen (vom Nutzer nicht ausdrücklich bestätigt)

- **Pre-Releases bleiben Proben, nie Listeneinträge.** Sie stehen nicht in
  `compat.toml`, sondern laufen nur aus dem Scout heraus. Damit kann ein
  Pre-Release den PR-Check `compat` gar nicht erreichen. Erscheint die
  stabile Version, meldet der Scout sie wie heute, und sie kommt regulär in
  die Liste.
- Ein Fixture je Go-Hauptversion; das Fixture einer Version darf die Syntax
  älterer Versionen mitbenutzen, muss aber jedes neue Sprachmittel seiner
  eigenen enthalten.
- **Syntax-Fixtures sind zusätzliche Einträge nur mit der Art `graph`.**
  `go-module` bleibt die gemeinsame Linie für `lint`, `test`, `coverage`
  und für den Scout. Ein Fixture mit 1.27-Syntax lässt sich unter Go 1.26
  weder vetten noch testen, deshalb keine Ersatzlösung. Die Syntax-Fixtures
  (z. B. `go-syntax-1.26`, `go-syntax-1.27`) zählen für den Scout nicht als
  eigene Linie; sonst meldete jede Linie dieselbe neue Version noch einmal.
  Eine Prüfung der Liste verlangt stattdessen: Jede Go-Version, die in der
  Liste steht, hat ihr Syntax-Fixture.
- **Ein Lauf mit einer Go-Version über der `go.mod`-Toolchain baut loomux mit
  der eigenen Toolchain.** Das gilt für stabile Proben ebenso wie für
  Pre-Releases. Eine solche Probe prüft das neueste vorhandene Fixture, ist
  also ein Regressionslauf; das Syntax-Fixture der neuen Version entsteht
  erst, wenn sie in die Liste kommt. Eine Prüfung der Liste verlangt dann:
  Die höchste Go-Version der Liste liegt nicht über der Toolchain in
  `go.mod`. Wer eine neue Version einträgt, zieht also `go.mod` mit.
- **Das Ende der Pflege wird nur für Laufzeiten gemeldet** (Go, Python,
  Godot), nicht für Addons und Tools. Die Norm spricht von Sprachen.
- **Die Meldung zum Ende der Pflege ist ein Issue ohne Probe** mit eigener
  Titelform (z. B. `compat: python 3.10 is past end of support`). Es gilt als
  erledigt, sobald kein Eintrag dieser Version mehr in der Liste steht, und
  läuft nicht in der Probenmatrix. Der Issue-Text nennt die aktuelle
  loomux-Version als die letzte, die die Version unterstützt, damit der
  Changelog-Satz des Austrage-PRs fertig dasteht.
- **Für Python und Godot ist der Graph-Teil einer Probe vorerst nur ein
  Regressionslauf.** Auch sie lesen im Binary (gotreesitter, der
  GDScript-Extraktor), also prüft der `graph`-Teil einer Probe von Python
  3.16a1 oder einer Godot-Beta keine neue Syntax. Diese Session deckt bei
  ihnen die laufzeitabhängigen Arten ab (`lint`, `types`, `test`,
  `coverage`); der Graph je Version folgt mit ihren Fixtures (Python-Session,
  später Godot).
- **Ende der Pflege kommt für alle drei Laufzeiten von endoflife.date**
  (`/api/v1/products/<go|python|godot>`, Feld `isEol`). Eine Quelle für alle
  Laufzeiten; python.org und Godot veröffentlichen kein maschinenlesbares
  Pflegeende. Gepflegt geprüft am 2026-11-08 (Repo nicht archiviert, Push am
  selben Tag). Stand dort: Go 1.26/1.27, Python 3.11–3.15, Godot 3.6 und
  4.5–4.7 gepflegt, also die heutige Liste. Der go.dev-Standardfeed (ohne
  `include=all`, am 2026-11-08 `go1.27.3`, `go1.26.10`) dient nur zum
  Gegenprüfen; für Pre-Releases fragt der Scout go.dev mit `include=all`.
- `release.yml` baut mit `go-version-file: go.mod`; das ausgelieferte Binary
  liest also mit der Toolchain aus `go.mod`, worauf die Toolchain-Regel oben
  beruht.
- Godot-Pre-Releases stehen nur in `godotengine/godot-builds`, nicht im
  Release-Feed von `godotengine/godot`; der Scout fragt dort.
- Das Issue eines Pre-Release-Funds trägt die Pre-Release-Kennung im Titel
  (z. B. `compat: go-module/go 1.28rc1 is out`), damit ein späterer RC ein
  neues Issue ergibt und das Issue der stabilen Version davon getrennt
  bleibt. Ein älteres Pre-Release-Issue derselben Linie schließt der Scout,
  sobald ein neueres oder die stabile Version da ist.

## Dem Bauenden überlassen

- Wie Pre-Release-Versionen geparst und verglichen werden (Go `1.28rc1`,
  Python `3.16.0a1`, Godot `4.8-beta2`) und wie `Finding`/`Resolved` damit
  umgehen.
- Wie die Pre-Release-Laufzeiten installiert werden: `actions/setup-go` für
  Go-RCs, uv für Python-Pre-Releases, für Godot die Assets der
  Pre-Release-Builds (das heutige `install_godot` nimmt nur `-stable`).
- Wie der Go-Probe-Lauf loomux mit der Probe-Toolchain baut (z. B.
  `GOTOOLCHAIN=local` nach dem zweiten `setup-go`), ohne die stabilen Läufe
  zu ändern.
- Welche Kanten jedes neue Go-Sprachmittel ergeben soll, festgelegt am
  bestehenden Kantenmodell des Extraktors.
