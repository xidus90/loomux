---
type: Topic
title: Die Scheiben und ihre Abnahmen
description: Acht Scheiben, jede mit einem Fertig-Kriterium, das ohne Codelektüre prüfbar ist.
open_conflicts: 0
realization: in_progress
sources:
  - id: architektur-spec
    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M39G4J14CK311B66GRSAK7HQ
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
---

Der Bau ist in Scheiben zerlegt, und **jede Scheibe trägt ein Fertig-Kriterium,
das ohne Codelektüre prüfbar ist**. Nach jeder Scheibe läuft das zutreffende
Messprogramm; eine Scheibe ohne Zahlen gilt nicht als fertig.

| # | Scheibe | Fertig, wenn |
|---|---|---|
| 0 | Fundament ohne Code | Suchleiter läuft im Alltag; Vergleich mit und ohne einmal gefahren; Spike: trägt qmd sprachübergreifend? |
| 1 | Indexer | zwei Läufe auf unverändertem Bestand erzeugen **byteweise identische** Dateien |
| 2a | Suchkette ohne Daemon | die fünf schreibfreien Werkzeuge laufen als direkte CLI über drei externe Bestände; Datenschutznachweis; leeres Ergebnis von Fehler unterscheidbar |
| 2b | Messwerk | `brain bench` in allen Profilen und mit Latenzmessung gelaufen, Protokolle im Vault |
| 2c | Daemon, IPC und MCP | Daemon hält Index und Modelle, Klient startet ihn; der MCP-Adapter setzt auf denselben Kern; Datenschutznachweis über den echten Kanal wiederholt |
| 3 | Wiki-Schicht plus die zwei Skills | präparierte Notiz: drei Widersprüche markiert, harmlose Ergänzung eingearbeitet, nichts überschrieben, die Konfliktzahl stimmt; eine Recherche landet auf Wunsch als Entwurf im Wiki |
| 4 | Import und Konverter | zwei Transkripte ergeben zweimal hintereinander dieselbe Datei, mit Absätzen statt Zeitstempelfragmenten |
| 5 | Brain Maintenance plus Merge-Hook | nachgebauter Fall vollständig durchlaufen; Selbstheilung beim Umbenennen; erfundenes Zitat fällt durch, wörtliches geht durch; im Analysepaket **kein Quelltext** |
| 6 | Lokales Modell (opt-in) | Vorschlag ohne ausgehenden Verkehr; Gegenprobe abgeschaltet ebenfalls ohne Verkehr |
| 7 | Web-App | Freigabe in der App erzeugt dieselben Wirkungen wie über die CLI |

## Warum Scheibe 2 dreigeteilt ist

Ihr Fertig-Kriterium bündelte vier Teilsysteme in einem Satz — und die Messung,
die über das Prozessmodell entscheiden soll, hätte sonst mitten in der Scheibe
gelegen, deren Bau sie bestimmt. **2a baut die Kette, 2b misst sie, 2c setzt den
Daemon darunter.** Der Datenschutznachweis steht in 2a, weil der Kanal dort zum
Begriff wird; 2c wiederholt ihn nur am echten Adapter.

## Der Plan je Scheibe

Jede gebaute Scheibe hat eine eigene Planseite; sie führt die Aufgaben und das,
was die Ausführung am Plan widerlegt hat.

- 0 — [Fundament ohne Code](../sources/plan-scheibe-0-fundament.md)
- 1 — [Indexer](../sources/plan-scheibe-1-indexer.md)
- 2a — [Suchkette ohne Daemon](../sources/plan-scheibe-2a-suchkette.md), dazu
  die [Nacharbeit nach der Abnahme](../sources/plan-scheibe-2a-nacharbeit.md)
- 2b — [Messwerk](../sources/plan-scheibe-2b-messwerk.md), dazu der
  [Prüfkorpus v1](../sources/plan-pruefkorpus-v1.md)
- 2c — [Daemon, IPC und gehaltener Unterprozess](../sources/plan-scheibe-2c1-daemon.md)
  und [MCP-Fronten](../sources/plan-scheibe-2c2-mcp-fronten.md)

## Reihenfolge

Scheibe 3 und 4 hängen nur am Kern, nicht aneinander; Scheibe 5 braucht beide;
die Web-App braucht alle. Sie entsteht nach dem Muster „Zielbilder zuerst":
Mockups erzeugen, dann Schritt für Schritt nachbauen mit Abnahme im Browser —
keine Beschreibungen als Auftrag.

## Bau- und Qualitätsregeln

TDD mit dem fehlschlagenden Test zuerst; **100 % Coverage, gemessen**,
Ausschlüsse nur mit begründendem Kommentar; keine ungeprüften Typen;
Python ≥ 3.14; `uv` für alles. **Kein Sprachmodell in Tests** — der
Prüfvorschlag wird gegen aufgezeichnete Antworten geprüft, und die Suche steht
in Tests hinter einer Attrappe. Reproduzierbarkeit ist Pflicht: gleiche Eingabe,
byteweise gleiche Ausgabe.

Bis zum Umzug am 2026-09-16 hieß diese Zerlegung „Scheiben" und war Python;
loomux baut in **Stufen** (1a, 1b, 2, 3, 4) und ist Go, sonst gelten die Regeln
unverändert. Die Abnahme ist dabei enger gefasst: Eine Stufe ist fertig, wenn
alle übersetzten Fälle grün sind oder freigegeben in der Abweichungsliste
stehen, die Coverage 100 % ist und jeder Ausschluss begründet, die
Mutationsrunde der Stufe gelaufen ist und ihre Überlebenden dokumentiert sind
(ab Stufe 1b, weil `loomux dev mutants` dort entsteht; die Runde von 1b
schließt die Entscheidungspakete aus 1a ein), die Zielwerte der Stufe gemessen
und in `docs/en/benchmarks.md`
und `docs/de/benchmarks.md` eingetragen sind, und das loomux-Repo die
Funktionen der Stufe selbst benutzt.

Siehe auch [Suche, Profile und Messwerte](suche-und-profile.md) und
[Die Wiki-Schicht](wiki-schicht.md); Quelle ist
[Architektur-Design ultra-brain](../sources/architektur-spec.md).
