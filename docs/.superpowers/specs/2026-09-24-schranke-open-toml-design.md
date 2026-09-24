# Die Schreibschranke hält einzelne Dateien aus `open.toml` offen

**Datum:** 2026-09-24
**Stand:** umgesetzt
**Betrifft:** `internal/brain/guard` (`open.go`, `Decide`),
`docs/{en,de}/configuration.md` §4 und §5, `docs/{en,de}/getting-started.md`,
`docs/{en,de}/benchmarks.md`

## Anlass

Die globale `~/.claude/CLAUDE.md` des Nutzers bindet
`~/.claude/AGENT_LEARNINGS.md` ein und verlangt von jedem Agenten, die Datei zu
pflegen. Am 2026-09-24 verweigerte die Schranke das Anhängen eines Eintrags:
Die Memory-Ausnahme (`specs-ub/2026-09-13-schranke-memory-offen-design.md`)
öffnet nur `projects/<projekt>/memory/` und hält den Rest von `~/.claude` mit
Absicht zu.

## Entscheidungen

Vom Nutzer am 2026-09-24:

1. **Konfiguration statt Code.** Ein erster Entwurf öffnete
   `<config>/AGENT_LEARNINGS.md` fest im Code (PR #21, erste Fassung). Die Datei
   ist aber die Konvention dieses Nutzers, nicht die von Claude Code: fest im
   Code bekäme jeder loomux-Nutzer eine Stelle unter `~/.claude`, die jeder Agent
   schreiben darf und die künftige Sitzungen als Anweisung laden. Das kehrt die
   Abwägung der Memory-Spec um, die eine zweite Freigabeliste (Weg C) verwarf,
   weil niemand sie brauchte — jetzt braucht sie jemand.
2. **Eine eigene Datei `<Zustandsverzeichnis>/open.toml`**, keine Tabelle in
   `registry.toml`. `loomux area add` schreibt die Registry über `AddArea` aus
   ihren `[[area]]`-Einträgen neu und verlöre eine Tabelle daneben still. Eine
   eigene Datei fasst kein Schreiber an; ist sie kaputt, bleibt nur diese
   Freigabe zu, die Registry und `brain` merken nichts.

## Entwurf

- `files = [...]`, sonst kein Schlüssel. Nur einzelne Dateien: ein Eintrag muss
  absolut sein, darf kein vorhandenes Verzeichnis nennen und nicht im
  Zustandsverzeichnis liegen (dort halten `registry.toml` und `open.toml` die
  Grenzen der Schranke; ein Eintrag dort ließe einen Agenten sich selbst
  Freigaben erteilen). Keine Globs.
- Ganz oder gar nicht: ein unbrauchbarer Eintrag, ein unbekannter Schlüssel,
  ungültiges TOML oder eine unlesbare Datei, und nichts darin öffnet. Die
  Ablehnung endet dann mit `<pfad> is ignored: <grund>`, sonst sähe der Nutzer
  nur die Verweigerung und nie, dass seine Datei nicht gilt.
- Offen zu denselben Bedingungen wie das Memory: gelesen vor der Registry, also
  auch bei unlesbarer Registry offen; das Manifest geht vor; in einem gemischten
  Aufruf wird nur das Ziel außerhalb verweigert. Pfade werden vor dem Vergleich
  aufgelöst, verglichen wird in Komponenten.
- Kein Agent schreibt `open.toml`: das Zustandsverzeichnis liegt außerhalb
  jedes Baums, den die Schranke öffnet.

## Kosten

`openFiles` läuft bei jedem schreibenden Aufruf: 0,08 ms ohne Datei, 0,22 ms mit
einem Eintrag (`benchmarks.md`, 2026-09-24 10:45).

## Tests

`internal/brain/guard/open_test.go`: Durchlass neben einem Wiki, bei Registry
ohne Bereich und bei unlesbarer Registry; nichts neben der Datei; gemischter
Aufruf; jede Art kaputter Datei mit ihrem Grund in der Ablehnung; unlesbare
Datei; Junction-Schleife als Eintrag und als Zustandsverzeichnis; die Ablehnung
nennt die Dateien, auch ohne beschreibbaren Baum.
