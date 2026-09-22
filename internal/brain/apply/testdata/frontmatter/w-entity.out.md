---
type: Entity
title: Der brain-Daemon
description: Der langlebige Prozess, der Index und Modelle hält — und warum er unter
  Windows über WMI startet.
open_conflicts: 0
realization: implemented
implemented_in: cba7c8b
sources:
- id: architektur-spec
  resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
  doc_id: 01M0QGS594F2KCWTWK9XV07M05
  content_hash: sha256:0000000000000000000000000000000000000000000000000000000000000000
  revision: 8
generated:
  at: '2026-09-22T08:16:27.936837+00:00'
verified:
- by: human:tester
  at: '2026-09-22T08:16:27.936837+00:00'
---

Ein langlebiger Python-Prozess hält Index und Modelle im Speicher. Darauf setzen
dünne Klienten auf: ein CLI-Client und ein zustandsloser MCP-Adapter, beide über
eine Named Pipe beziehungsweise einen Unix-Socket.

**Kein Dienst, kein Autostart.** Der erste Klient, der keine Pipe vorfindet,
startet ihn und wartet auf Bereitschaft.

Das gilt bis zum Umzug am 2026-09-16: loomux fragt keinen brain-Daemon mehr,
weder über Pipe noch über Socket, sondern beantwortet `search`, `catalog`,
`read`, `neighbors` und `status` im eigenen Prozess. Der einzige Daemon, den es
noch kennt, ist der von [qmd](qmd.md) — angesprochen über HTTP, nicht über eine
Pipe.

## Der Windows-Umweg

„Startet ihn" trägt unter Windows nicht ohne Weiteres: Ein MCP-Wirt legt für den
Front-Prozess ein Job-Objekt an, das am Sitzungsende den **ganzen Baum** beendet
— also auch den Daemon und den qmd-Daemon darunter. Jede neue Sitzung zahlte
dann den kalten Modellstart erneut: **16 s statt 0,8 s**. Ein Ausbruch aus dem
Job hilft nicht, weil das Job-Objekt das dafür nötige Kennzeichen nicht trägt.
Der Klient startet den Daemon deshalb über WMI — der Prozess wird vom
WMI-Anbieter erzeugt und gehört keinem Job. Der Grundsatz bleibt unberührt: Der
Daemon entsteht weiterhin auf Zuruf des ersten Klienten und gehört dem
angemeldeten Benutzer.

## Warum er nötig ist

Der MCP-Adapter existiert, weil Claude Code je Sitzung und Projekt einen eigenen
MCP-Prozess startet; ohne Trennung hielte jede Sitzung eigenen Index und eigenes
Modell im Speicher. Und die Latenzmessung hat den Daemon von einer
Bequemlichkeit zur Voraussetzung befördert: Eine kalte Bedeutungssuche von einer
halben Minute würde niemand zweimal abwarten
([Suche, Profile und Messwerte](../topics/suche-und-profile.md)).

## Zwei Bereitschaftsstufen

Stufe 1 — Rohr offen, Graph und Registrierung geladen — kostet 1132 ms. Stufe 2
— erste Suche beantwortet — kostet 9982 ms mit kaltem Modell, aber nur 1219 ms,
wenn qmds eigener Daemon noch läuft. Dass nur der *erste* Start den Modellstart
zahlt, ist der gemessene Gewinn: **[qmd](qmd.md)s Daemon überlebt den
brain-Daemon, der ihn gestartet hat.**

## Verhalten bei Störungen

Läuft der Daemon nicht, startet der Klient ihn und meldet die Wartezeit. Ist der
Index veraltet, werden Treffer geliefert und sichtbar als veraltet markiert.
Fehlt oder bricht qmd ab, funktionieren Katalog und Lesen weiter und die Suche
meldet den Ausfall. Eine beschädigte Zustandsdatenbank wird verworfen und neu
gebaut — sie ist ohnehin abgeleitet.

Siehe [Grundsätze und Vertrauenskette](../topics/architektur-grundsaetze.md):
Ausfälle degradieren, sie blockieren nicht.
