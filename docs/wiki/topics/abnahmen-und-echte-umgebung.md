---
type: Topic
title: Was erst die echte Umgebung zeigt
description: Das wiederkehrende Muster des Projekts — die schwersten Befunde entstanden nie im Testlauf, sondern beim ersten Gebrauch.
open_conflicts: 0
realization: in_progress
sources:
  - id: benchmarks
    resource: brain://project/loomux/docs/de/benchmarks.md
    doc_id: 01M47W91R5MPENKN7N34H6QTNT
    content_hash: "sha256:73cf99840e93d14ff9d4cc7a385fb33fe7f8733867d1ee8523a43faeb943c1d8"
    revision: 1
---

Über die Scheiben hinweg wiederholt sich ein Muster, das eine eigene Seite
verdient: **Die schwersten Befunde entstanden nicht im Testlauf, sondern beim
ersten Gebrauch in der echten Umgebung.** Der Testlauf war jedes Mal grün.

## Die Sorten, die immer wiederkehren

**Ein Nullergebnis sieht aus wie ein Erfolg.** Ein Graph mit 0 von 979
aufgelösten Links ist von einem Bestand ohne Links nicht zu unterscheiden; das
Programm verhielt sich richtig und sagte nur nirgends, dass es gerade alles
verworfen hatte (Plan Scheibe 1).
Gegenmittel ist jedes Mal dasselbe: **eine Quote ausgeben, nicht nur ein
Ergebnis.**

**Ein grüner Test kann eine Welt prüfen, die es nicht gibt.** Die Fixture der
Entdopplung schrieb ein Register, das der Indexer gar nicht erzeugen kann. Der
Test war nicht falsch — er war **unverankert**
(Abnahme der Scheibe 2a).

**Die Umgebung bringt eigene Regeln mit, die keine Testsuite kennt.** Die
Zeichenkodierung der Standardausgabe, die Zeilenenden einer Sicherungskopie, ein
Job-Objekt, das den Prozessbaum beendet — alles Eigenschaften des Wirts, nicht
des Codes ([Der brain-Daemon](../entities/brain-daemon.md)).

**Der Bestand steht nicht still.** Eine parallel arbeitende Sitzung im selben
Baum bindet jede Aussage der Form „zwei Läufe sind identisch" an einen
Zeitpunkt. Und ein Dateizähler taugt deshalb nicht als Unverändert-Nachweis.

**Ein Befund ist selten nur der Befund.** Der Ausschluss, der 22 fremde
Navigationsseiten unsichtbar machte, war zugleich — unbemerkt — die zweitgrößte
Ursache des schwersten offenen Befunds desselben Protokolls.

**Eine Aussage galt enger, als sie formuliert war.** „Es gibt den Fall nicht"
stimmte je Bereich und war über Bereichsgrenzen hinweg falsch. Der Beleg lag
vor, ungesehen.

**Eine Sorge, die niemand gemessen hat, ist keine Zahl.** Die befürchtete
Langsamkeit lag bei 1,3 s. Sie bleibt als Prüfpunkt stehen — aber als Zahl,
nicht als Vermutung.

## Was daraus als Arbeitsweise folgt

- Die Abnahme läuft **außerhalb** des Arbeitsbereichs, gegen echte Bestände.
- Während der Abnahme wird **kein Produktivcode angefasst** — auch dort nicht,
  wo der Lauf einen Fehler findet. Sonst misst das Protokoll einen Zustand, den
  es selbst erzeugt hat.
- Befunde werden **nummeriert und einzeln protokolliert**, auch die, die
  niemand erwartet hat.
- Was vertagt wird, wird **eingecheckt** — damit es nicht mit dem
  Arbeitsverzeichnis stirbt.

Siehe [Die Stufen und ihre Abnahme](scheiben-und-abnahme.md).
